package relay

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	xvt "github.com/charmbracelet/x/vt"
)

// ErrReadOnly is returned by HandleInject when the subscriber has the read-only
// cap and is therefore not permitted to write to PTY stdin (SEC-01, D-04).
var ErrReadOnly = errors.New("relay: inject rejected: read-only capability")

// ErrInjectNotRecorded is returned by HandleInject when the PTY write succeeded
// but persisting/broadcasting the chat record failed (e.g. the chat cap was
// reached or the line was too large). It signals a deliberate divergence: the
// inject reached the live terminal but no chat record exists. The read pump
// turns this into a NAK so the originating client is informed rather than the
// failure being silently swallowed (WR-02 — "let it crash", not silent fallback).
var ErrInjectNotRecorded = errors.New("relay: inject delivered to terminal but not recorded in chat")

// MaxInjectTextBytes is the explicit upper bound on the raw text of a single
// MsgSessionInject frame, enforced in HandleInject before any PTY write. It
// makes the inject bound intentional rather than implicitly relying on the
// coder/websocket default read limit (IN-03). 64 KiB is generous for a pasted
// command line yet well under the chat-layer maxChatLineBytes (1 MiB).
const MaxInjectTextBytes = 64 * 1024

// ErrInjectTooLarge is returned by HandleInject when the raw inject text exceeds
// MaxInjectTextBytes. The read pump turns it into a NAK, so the bound is enforced
// before the PTY write and independent of the WS library default (IN-03).
var ErrInjectTooLarge = errors.New("relay: inject rejected: text exceeds maximum size")

// Subscriber represents a single WebSocket connection subscribed to a session's output.
type Subscriber struct {
	// Msgs is the outbound channel for framed messages. Buffered to 256 frames.
	Msgs chan []byte
	// CloseSlow is called in its own goroutine when the Msgs channel is full.
	// Implementations should close the WebSocket and call Unsubscribe.
	CloseSlow func()

	// ReadOnly: if true, input frames from this client are discarded by the server read pump. (MC-03)
	ReadOnly bool

	// Name: optional client identity from ?client= query param. (MC-05)
	Name string

	// Cols, Rows: last reported terminal dimensions from this client.
	// Read/written under hub.mu. (MC-06)
	Cols int
	Rows int

	// Phase 152: identity fields — set once at subscribe time, read by read pump.
	TailnetID  string                        // "local" or Tailscale node public key string
	Origin     string                        // "local" (relay loopback) or "web" (webserver Tailscale)
	PersonKey  string                        // TailnetID + ":" + Origin — the collapse key (D-04)
	Alias      string                        // current display name (mutable via MsgAliasSet)
	AliasSetFn func(personKey, alias string) // persistence callback; avoids import cycle with daemon
}

// presenceState is the per-person collapsed entry in the Hub presence roster.
// Guarded by hub.mu.
type presenceState struct {
	TailnetID string
	Origin    string
	Alias     string
	ConnCount int // number of active Subscriber connections for this PersonKey
}

// Hub manages a single PTY session's output fan-out.
// One goroutine (Run) drains the reader; N subscribers receive every frame.
type Hub struct {
	sessionID  string
	reader     io.Reader
	writer     io.Writer
	scrollback *Scrollback
	resizeFn   func(cols, rows int) error

	mu          sync.Mutex
	subscribers map[*Subscriber]struct{}
	done        chan struct{}
	closed      bool
	closeOnce   sync.Once

	// ptyCols, ptyRows: current PTY dimensions as set by the host-authority arbiter.
	// Only local-origin subscribers drive these fields; the last host grid is frozen
	// when no local host is connected (D-01). (VIEW-01, VIEW-02)
	ptyCols int
	ptyRows int

	// Phase 152: presence/typing state — guarded by mu.
	presenceRoster  map[string]*presenceState // personKey → collapsed presence state
	typingRoster    map[string]*time.Timer    // personKey → 5s TTL timer
	lastTypingBcast map[string]time.Time      // personKey → last typing-start broadcast time (rate limit)
	typingTTL       time.Duration             // injectable for tests; default 5s

	// Phase 153: persist+broadcast callback. Wired by engine.go after Hub+ChatStore
	// are both constructed. Nil-safe: HandleInject skips persist+broadcast when nil.
	// Guarded by mu.
	chatAppendFn func(ChatMessage) (ChatMessage, error)

	// Phase 175-04 (BUG-04, #119): live per-hub VT emulator + its own guard.
	// liveEmu is nil until EnsureLiveEmulator's first call (lazy — T-175-04-01:
	// never constructed for a session that never gets a shared viewer). Once
	// constructed it lives for the rest of the hub's life, continuously fed by
	// Run()'s drain loop via feedLiveEmulator, so its state survives any later
	// wrap of the raw scrollback ring — RenderSnapshot() never re-derives from
	// the (possibly truncated) ring (RESEARCH "Pitfall 3").
	//
	// emuMu is a SEPARATE lock from mu (T-175-04-02): a slow/stuck emulator
	// write must never stall the PTY-drain/broadcast loop or contend with
	// subscriber fan-out.
	emuMu   sync.Mutex
	liveEmu *xvt.Emulator
}

// altScreenEnterSeq is the DEC private-mode sequence a full-screen TUI emits
// exactly once on entering the alternate screen buffer (ESC[?1049h).
// RenderSnapshot prefixes it onto the reconnect preamble whenever the live
// emulator reports IsAltScreen(), so a reconnecting/late-joining client's
// xterm.js re-enters the correct buffer before painting replayed content —
// reconstructing the mode marker the raw 256 KiB scrollback ring may have
// wrapped past (BUG-04, #119).
const altScreenEnterSeq = "\x1b[?1049h"

// liveEmulatorScrollbackLines bounds the live per-hub VT emulator's own
// internal scrollback buffer (T-175-04-01). The emulator exists solely to
// produce a current-screen RenderSnapshot() — it does not need real
// scrollback depth — so this is deliberately small, mirroring the
// emuRows=50 row bound GetSessionStyledTailLines (internal/daemon/engine.go)
// uses for its own headless emulator.
const liveEmulatorScrollbackLines = 50

// liveEmuQueryStripPattern mirrors internal/daemon/engine.go's
// queryStripPattern byte-for-byte. It strips terminal-query and in-band-
// resize escape sequences that would otherwise elicit a blocking response
// write into charmbracelet/x/vt's unbuffered response pipe (Emulator.pw) —
// nothing ever drains Emulator.Read() on this feed path, so an unstripped
// query would deadlock both EnsureLiveEmulator's bootstrap and Run()'s drain
// loop (#96/#100). SGR/color sequences are deliberately preserved so styled
// output survives.
//
// relay MUST NOT import internal/daemon (daemon already imports relay — an
// import back the other way would be a cycle), so this pattern is
// duplicated here rather than shared.
var liveEmuQueryStripPattern = regexp.MustCompile(
	`\x1b\[[0-9;]*c` + // DA1 (and any params before c)
		`|\x1b\[>[0-9;]*c` + // DA2
		`|\x1b\[[0-9;]*n` + // DSR (5n/6n and any CSI n)
		`|\x1b\[\?[0-9;]*n` + // DECXCPR (ESC[?...n)
		`|\x1b\[[0-9;]*\$p` + // DECRQM ANSI
		`|\x1b\[\?[0-9;]*\$p` + // DECRQM DEC
		`|\x1b\[\?2048[hl]` + // in-band resize set/reset (set triggers a pw write)
		`|\x1b\]1[012];\?(?:\x07|\x1b\\)`, // OSC 10/11/12 color query, BEL- or ST-terminated
)

// stripMsgOutputBytes drops every relay.MsgOutput (0x01) framing byte from
// data. Mirrors GetSessionStyledTailLines's inline strip: Hub.Run wraps each
// PTY-read chunk in a MsgOutput frame via MakeOutputFrame BEFORE appending it
// to scrollback, so ScrollbackSnapshot() interleaves this framing byte
// throughout its contents.
func stripMsgOutputBytes(data []byte) []byte {
	stripped := make([]byte, 0, len(data))
	for _, b := range data {
		if b != MsgOutput {
			stripped = append(stripped, b)
		}
	}
	return stripped
}

// NewHub constructs a Hub for the given session.
// scrollbackBytes controls the scrollback buffer capacity.
// resizeFn is called when a resize event is received; may be nil.
func NewHub(sessionID string, reader io.Reader, writer io.Writer, scrollbackBytes int, resizeFn func(cols, rows int) error) *Hub {
	return &Hub{
		sessionID:       sessionID,
		reader:          reader,
		writer:          writer,
		scrollback:      NewScrollback(scrollbackBytes),
		resizeFn:        resizeFn,
		subscribers:     make(map[*Subscriber]struct{}),
		done:            make(chan struct{}),
		presenceRoster:  make(map[string]*presenceState), // Pitfall 4 — must init to avoid nil map panic
		typingRoster:    make(map[string]*time.Timer),
		lastTypingBcast: make(map[string]time.Time),
		typingTTL:       5 * time.Second,
	}
}

// Resize calls the resize callback registered at construction time.
// If no callback was provided it is a no-op.
func (h *Hub) Resize(cols, rows int) error {
	if h.resizeFn != nil {
		return h.resizeFn(cols, rows)
	}
	return nil
}

// Subscribe adds a subscriber to receive future frames.
// Must be called before Hub.Run or while Run is active under the mu lock.
// If sub.PersonKey is non-empty, the presence roster is updated (ConnCount
// incremented or a new entry created). Callers should call NotifyPresence
// after Subscribe returns to push the updated roster.
func (h *Hub) Subscribe(sub *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribers[sub] = struct{}{}
	if sub.PersonKey != "" {
		if s, ok := h.presenceRoster[sub.PersonKey]; ok {
			s.ConnCount++
		} else {
			h.presenceRoster[sub.PersonKey] = &presenceState{
				TailnetID: sub.TailnetID,
				Origin:    sub.Origin,
				Alias:     sub.Alias,
				ConnCount: 1,
			}
		}
	}
}

// RegisterPresence performs ONLY the presence-roster registration that
// Subscribe's `if sub.PersonKey != ""` block does, decoupled from adding the
// subscriber to the broadcast set.
//
// Phase 155-05 (PARITY-01) two-phase web path: the WSS handler calls Subscribe
// immediately after the WebSocket Accept (so delivery starts at once, before
// the latency-bearing WhoIs identity lookup) with an empty PersonKey, then —
// once WhoIs resolves and sub.TailnetID/PersonKey/Alias are set — calls
// RegisterPresence to add the roster entry. It MUST be called only after the
// sub's identity fields are set, and only once per connection.
//
// The relay loopback path (server.go) sets identity before Subscribe and keeps
// registering presence in one shot via Subscribe; it does NOT call this.
//
// Callers should call NotifyPresence after RegisterPresence returns to push the
// updated roster. No-op when sub.PersonKey is empty.
func (h *Hub) RegisterPresence(sub *Subscriber) {
	if sub.PersonKey == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.presenceRoster[sub.PersonKey]; ok {
		s.ConnCount++
	} else {
		h.presenceRoster[sub.PersonKey] = &presenceState{
			TailnetID: sub.TailnetID,
			Origin:    sub.Origin,
			Alias:     sub.Alias,
			ConnCount: 1,
		}
	}
}

// Unsubscribe removes a subscriber and returns true when a presence broadcast
// is warranted (i.e. the last connection for sub.PersonKey has dropped).
// Callers should call NotifyPresence when presenceChanged is true.
//
// Changed signature from Phase 152: now returns (presenceChanged bool).
// Existing call sites that discard the return value continue to compile.
func (h *Hub) Unsubscribe(sub *Subscriber) (presenceChanged bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers, sub)
	if sub.PersonKey != "" {
		if s, ok := h.presenceRoster[sub.PersonKey]; ok {
			s.ConnCount--
			if s.ConnCount <= 0 {
				delete(h.presenceRoster, sub.PersonKey)
				// Cancel and remove any active typing timer for this person.
				if t, ok := h.typingRoster[sub.PersonKey]; ok {
					t.Stop()
					delete(h.typingRoster, sub.PersonKey)
				}
				delete(h.lastTypingBcast, sub.PersonKey)
				presenceChanged = true
			}
		}
	}
	return
}

// SubscriberCount returns the number of currently subscribed clients. (MC-04)
func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

// RemoteViewerCount returns the number of currently subscribed clients whose
// Origin is "web" — i.e. real remote/shared viewers, excluding the app's own
// internal Origin=="local" WebSocket subscribers (TerminalPanel, ChatPanel,
// status watcher, Hub-card preview). Unlike SubscriberCount, this is a raw
// per-connection count with no PersonKey collapse (D-01, D-02, D-03 — Phase
// 168 / FIX-04, #121). SubscriberCount itself is left unchanged; it is still
// consumed by relay/server.go's NotifyViewerCount.
func (h *Hub) RemoteViewerCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	count := 0
	for s := range h.subscribers {
		if s.Origin == "web" {
			count++
		}
	}
	return count
}

// DisconnectWebViewers force-closes every currently subscribed Origin=="web"
// subscriber for this session's hub, leaving Origin=="local" subscribers
// (the app's own TerminalPanel/ChatPanel/status-watcher/preview connections)
// completely untouched (D-05, Phase 168 / FIX-02, #117).
//
// It reuses the exact same close mechanism as broadcastResize's close-on-full
// path — go sub.CloseSlow() — rather than inventing a second termination
// mechanism. Matching subscribers are collected into a local slice under
// h.mu, the lock is released, and ONLY THEN is CloseSlow called per
// subscriber. This unlock-before-IO ordering is mandatory: CloseSlow calls
// back into Unsubscribe, which re-enters h.mu — calling CloseSlow while still
// holding h.mu self-deadlocks (T-157-04, mirrored from broadcastResize's
// documented hazard at hub.go:288-291/311-322).
//
// DisconnectWebViewers does not revoke the session's capability token — web
// viewers may reconnect with the same join code (D-06). Full access
// revocation remains the existing "disable web sharing" path.
func (h *Hub) DisconnectWebViewers() {
	h.mu.Lock()
	targets := make([]*Subscriber, 0, len(h.subscribers))
	for s := range h.subscribers {
		if s.Origin == "web" {
			targets = append(targets, s)
		}
	}
	h.mu.Unlock()

	for _, sub := range targets {
		go sub.CloseSlow()
	}
}

// ResizeClient stores the subscriber's reported dimensions and applies the
// host-authority PTY-size policy (VIEW-01, VIEW-02):
//
//   - VIEW-02 origin gate: non-local (web/remote) subscribers return immediately;
//     their reported size never enters the arbiter and cannot drive or DoS the PTY
//     grid (T-157-01 mitigation).
//   - D-02 min-among-local: the PTY grid tracks the smallest terminal among all
//     connected local-origin subscribers so every host viewer sees the full output.
//   - D-01 freeze: when no local subscriber currently reports a positive size the
//     last host grid is preserved — the PTY is never reset to zero or to a guest size.
//   - VIEW-01 broadcast: each grid change is broadcast to all subscribers via a
//     MsgResize (0x02) frame before resizeFn is called, so guests learn the new
//     authoritative grid immediately.
//
// broadcastResize and resizeFn are called AFTER releasing hub.mu (unlock-before-IO
// discipline) to avoid self-deadlock and PTY-syscall contention.
func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
	// VIEW-02: non-local origin is a read-only guest — it may not drive the PTY grid.
	// Return immediately; the guest's reported size is not recorded anywhere.
	if sub.Origin != "local" {
		return nil
	}

	h.mu.Lock()
	sub.Cols = cols
	sub.Rows = rows

	// D-02 min-among-local: iterate only local-origin subscribers with a positive size.
	minCols, minRows := 0, 0
	for s := range h.subscribers {
		if s.Origin != "local" || s.Cols <= 0 || s.Rows <= 0 {
			continue
		}
		if minCols == 0 || s.Cols < minCols {
			minCols = s.Cols
		}
		if minRows == 0 || s.Rows < minRows {
			minRows = s.Rows
		}
	}

	// D-01 freeze: if no local subscriber reports a positive size, leave ptyCols/ptyRows
	// unchanged (last host grid persists). Only update when we have a valid min.
	needResize := (minCols > 0 && minRows > 0) && (minCols != h.ptyCols || minRows != h.ptyRows)
	if needResize {
		h.ptyCols = minCols
		h.ptyRows = minRows
	}
	pc, pr := h.ptyCols, h.ptyRows
	h.mu.Unlock() // unlock-before-IO: broadcastResize and resizeFn must not run under mu

	if needResize {
		// VIEW-01: broadcast the new authoritative grid to all subscribers.
		h.broadcastResize(uint16(pc), uint16(pr))
		// CR-01: keep the live emulator's geometry in lockstep with the PTY so a
		// later reconnect preamble (RenderSnapshot) is not mis-dimensioned. Safe
		// here — hub.mu is already released (resizeLiveEmulator takes only emuMu).
		h.resizeLiveEmulator(pc, pr)
		if h.resizeFn != nil {
			return h.resizeFn(minCols, minRows)
		}
	}
	return nil
}

// broadcastResize sends a MsgResize (0x02) frame encoding cols and rows to all
// subscribers using a non-blocking send. Slow subscribers (full channel) have
// CloseSlow called in a new goroutine. Mirrors BroadcastMeta (hub.go:306).
//
// MUST be called after hub.mu is released (self-acquires mu internally; calling
// while holding mu causes a self-deadlock — T-157-04 mitigation).
func (h *Hub) broadcastResize(cols, rows uint16) {
	frame := MakeResizeFrame(cols, rows)
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subscribers {
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()
		}
	}
}

// Run is the drain goroutine. It reads from the PTY reader in 32 KiB chunks,
// wraps each chunk in a MakeOutputFrame, appends to scrollback, and broadcasts
// to all subscribers. Slow subscribers (full channel) have CloseSlow called in a
// separate goroutine so they cannot block the drain loop.
//
// Run returns when the reader returns any error (typically io.EOF on PTY close).
// On return, Run calls Shutdown() to signal completion.
func (h *Hub) Run() {
	defer h.Shutdown()

	buf := make([]byte, 32*1024)
	for {
		n, err := h.reader.Read(buf)
		if n > 0 {
			// Copy the live slice before broadcasting — each frame allocation
			// is independent so slow subscribers cannot corrupt fast ones.
			frame := MakeOutputFrame(buf[:n])
			// CR-03 (code review): append to scrollback AND feed the live
			// emulator as ONE atomic step under emuMu — see recordFrame. This
			// prevents a boundary frame being counted twice (once by
			// EnsureLiveEmulator's bootstrap ScrollbackSnapshot, once by the live
			// feed), the TestHub_TwoClientsFanOut "hello worldhello world" race.
			h.recordFrame(frame, buf[:n])
			h.broadcast(frame)
		}
		if err != nil {
			return
		}
	}
}

// recordFrame appends the framed bytes to the scrollback ring AND feeds the raw
// (pre-framing) PTY bytes to the live emulator (if one has been built) as ONE
// atomic step under emuMu. Called from Run()'s drain loop immediately after the
// PTY Read returns, OUTSIDE hub.mu (emuMu is a separate lock, T-175-04-02).
//
// Atomicity vs EnsureLiveEmulator's bootstrap is REQUIRED (CR-03, code review):
// EnsureLiveEmulator builds the emulator and bootstraps it once from
// ScrollbackSnapshot() while holding emuMu. If the scrollback append and the
// live feed were separate operations, a frame could be appended to scrollback,
// picked up by a concurrent bootstrap, and THEN fed again by the live feed —
// double-counting it (the TestHub_TwoClientsFanOut "hello worldhello world"
// race). Holding emuMu across both here guarantees each frame is captured
// exactly once: either it is already in the scrollback the bootstrap reads (and
// the nil-guarded feed below no-ops because liveEmu is not yet set) OR it
// arrives after liveEmu is set (and only the live feed applies it).
//
// The anti-hang mechanism for the emulator write is liveEmuQueryStripPattern
// (not goroutine isolation): stripping the query/response sequences lets
// emu.Write return synchronously with no drain goroutine — the proven engine.go
// #96/#100 mitigation. A query sequence NOT covered by that pattern would block
// emu.Write and therefore this drain loop.
func (h *Hub) recordFrame(frame, raw []byte) {
	h.emuMu.Lock()
	defer h.emuMu.Unlock()
	h.scrollback.Append(frame)
	if h.liveEmu == nil {
		return
	}
	clean := liveEmuQueryStripPattern.ReplaceAll(raw, nil)
	h.liveEmu.Write(clean) //nolint:errcheck // emulator Write never returns a meaningful error
}

// resizeLiveEmulator propagates an authoritative PTY resize to the live per-hub
// VT emulator (CR-01). EnsureLiveEmulator builds the emulator once at the
// initial PTY size and it would otherwise never follow later resizes, leaving
// RenderSnapshot's reconnect preamble mis-dimensioned after a guest-driven grid
// change (the core BUG-04 multi-viewer path via ResizeClient's min-arbiter).
// Takes ONLY emuMu (never hub.mu): EnsureLiveEmulator acquires emuMu→hub.mu, so
// this MUST be called after hub.mu is released, or the reverse lock order would
// deadlock. No-op until the emulator exists.
func (h *Hub) resizeLiveEmulator(cols, rows int) {
	h.emuMu.Lock()
	defer h.emuMu.Unlock()
	if h.liveEmu == nil {
		return
	}
	h.liveEmu.Resize(cols, rows)
}

// broadcast sends frame to all current subscribers using a non-blocking send.
// Subscribers with a full channel have their CloseSlow invoked in a new goroutine.
func (h *Hub) broadcast(frame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for sub := range h.subscribers {
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()
		}
	}
}

// BroadcastMeta sends a metadata frame to all subscribers using a non-blocking
// send. Slow subscribers have CloseSlow called — MsgMeta must never block the
// PTY drain loop. Used by relay.Server and webserver to push viewer count updates.
func (h *Hub) BroadcastMeta(frame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for sub := range h.subscribers {
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()
		}
	}
}

// Shutdown signals that the hub has stopped. Safe to call multiple times.
func (h *Hub) Shutdown() {
	h.closeOnce.Do(func() {
		h.mu.Lock()
		h.closed = true
		h.mu.Unlock()
		close(h.done)
	})
}

// ---------------------------------------------------------------------------
// Phase 152: Presence roster methods
// ---------------------------------------------------------------------------

// BroadcastPresence sends a MsgPresence frame to all subscribers using a
// non-blocking send. Identical to BroadcastMeta — separated for clarity.
func (h *Hub) BroadcastPresence(frame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subscribers {
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()
		}
	}
}

// BroadcastExcept sends a frame to all subscribers EXCEPT the excluded one.
// Used for typing-start broadcasts so the sender does not see their own indicator.
func (h *Hub) BroadcastExcept(frame []byte, exclude *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subscribers {
		if sub == exclude {
			continue
		}
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()
		}
	}
}

// CurrentPresence returns a snapshot of the current presence roster.
// The returned slice is safe to use after this method returns.
func (h *Hub) CurrentPresence() []PresenceEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	entries := make([]PresenceEntry, 0, len(h.presenceRoster))
	for key, s := range h.presenceRoster {
		entries = append(entries, PresenceEntry{
			PersonKey: key,
			TailnetID: s.TailnetID,
			Origin:    s.Origin,
			Alias:     s.Alias,
			ConnCount: s.ConnCount,
		})
	}
	return entries
}

// UpdateAlias updates the roster entry for personKey so the next CurrentPresence
// reflects the new alias. No-op if personKey is not in the roster.
func (h *Hub) UpdateAlias(personKey, alias string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.presenceRoster[personKey]; ok {
		s.Alias = alias
	}
}

// NotifyPresence pushes a full MsgPresence roster frame to all subscribers.
// Must be called OUTSIDE hub.mu (it acquires mu internally).
func NotifyPresence(hub *Hub) {
	roster := hub.CurrentPresence()
	frame := MakePresenceFrame(PresencePayload{Participants: roster})
	hub.BroadcastPresence(frame)
}

// ---------------------------------------------------------------------------
// Phase 152: Typing TTL methods
// ---------------------------------------------------------------------------

// UpdateTyping updates the typing state for sub.PersonKey.
//   - typing=true: broadcasts typing:true to all OTHER subscribers (sender excluded),
//     resets the per-person TTL timer, and applies a per-person rate limit of 500ms
//     to avoid broadcast storms (T-152-03).
//   - typing=false: cancels the TTL timer and broadcasts typing:false to ALL subscribers.
//
// Must be called OUTSIDE hub.mu (acquires and releases mu internally).
// The h.closed guard prevents post-shutdown panics (Pitfall 2 / T-152-07).
func (h *Hub) UpdateTyping(sub *Subscriber, typing bool) {
	personKey := sub.PersonKey
	alias := sub.Alias

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}

	if !typing {
		// Explicit stop — cancel any live timer.
		if t, ok := h.typingRoster[personKey]; ok {
			t.Stop()
			delete(h.typingRoster, personKey)
		}
		delete(h.lastTypingBcast, personKey)
		h.mu.Unlock()
		// Broadcast typing:false to ALL subscribers (including the sender).
		NotifyTyping(h, personKey, alias, false)
		return
	}

	// typing=true: apply rate limit — skip re-broadcast if last one was < 500ms ago.
	now := time.Now()
	shouldBcast := true
	if last, ok := h.lastTypingBcast[personKey]; ok && now.Sub(last) < 500*time.Millisecond {
		shouldBcast = false
	}

	// Reset (or start) the TTL timer.
	if t, ok := h.typingRoster[personKey]; ok {
		t.Stop()
	}
	h.typingRoster[personKey] = time.AfterFunc(h.typingTTL, func() {
		h.mu.Lock()
		if h.closed {
			h.mu.Unlock()
			return // Pitfall 2: do not broadcast after shutdown
		}
		delete(h.typingRoster, personKey)
		delete(h.lastTypingBcast, personKey)
		h.mu.Unlock()
		// TTL fired — broadcast typing:false to all (sender already gone or idle).
		NotifyTyping(h, personKey, alias, false)
	})

	if shouldBcast {
		h.lastTypingBcast[personKey] = now
	}
	h.mu.Unlock() // release BEFORE broadcasting (ResizeClient discipline)

	if shouldBcast {
		// Broadcast typing:true to all EXCEPT the sender (Pitfall 5 / T-152-03).
		frame := MakeTypingFrame(TypingPayload{PersonKey: personKey, Alias: alias, Typing: true})
		h.BroadcastExcept(frame, sub)
	}
}

// NotifyTyping broadcasts a MsgTyping frame to the appropriate audience.
// For typing=false it fans out to ALL subscribers (sender may be gone).
// For typing=true the caller (UpdateTyping) uses BroadcastExcept — this
// function is called only for the typing=false (stop/TTL) case from UpdateTyping.
func NotifyTyping(hub *Hub, personKey, alias string, typing bool) {
	frame := MakeTypingFrame(TypingPayload{PersonKey: personKey, Alias: alias, Typing: typing})
	hub.BroadcastPresence(frame) // reuse the all-subscribers fan-out
}

// WriteInput writes raw input bytes to the underlying PTY writer (stdin).
func (h *Hub) WriteInput(data []byte) error {
	_, err := h.writer.Write(data)
	return err
}

// ---------------------------------------------------------------------------
// Phase 153: inject machinery
// ---------------------------------------------------------------------------

// SetChatAppendFn wires the persist+broadcast callback for SessionInject messages.
// Must be called before the first WebSocket connection is accepted for this session
// (i.e. from engine.go CreateSession). Safe to call concurrently with hub.mu.
// Mirrors the Subscriber.AliasSetFn assignment pattern.
func (h *Hub) SetChatAppendFn(fn func(ChatMessage) (ChatMessage, error)) {
	h.mu.Lock()
	h.chatAppendFn = fn
	h.mu.Unlock()
}

// ChatAppendFnWired reports whether a chatAppendFn has been set.
// Used by the playwright-fixture admin server for diagnostic assertions only.
func (h *Hub) ChatAppendFnWired() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.chatAppendFn != nil
}

// BroadcastChat sends a MsgChat frame to all subscribers using a non-blocking
// send. Slow subscribers have CloseSlow called. Identical fan-out to BroadcastMeta
// — separated for clarity (MsgChat is a different frame type than MsgMeta).
func (h *Hub) BroadcastChat(frame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for sub := range h.subscribers {
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()
		}
	}
}

// HandleInject is called by the read pump when a MsgSessionInject frame arrives.
// It: (1) gates on !sub.ReadOnly (SEC-01, D-04), returning ErrReadOnly for RO clients;
// (2) sanitizes the text via SanitizePTYText (SEC-02, D-03); (3) writes the sanitized
// bytes to PTY stdin via WriteInput; (4) if chatAppendFn is non-nil, persists a
// DISPLAY-SAFE form of the text (SanitizeChatContent — bidi/C0/C1 stripped) as a
// ChatMessage{SessionInject:true} and broadcasts a MsgChat frame to all subs.
//
// WR-01: stored/broadcast content is display-safe text, NOT raw keystrokes. The
// PTY sanitizer protects stdin, but chat.jsonl / BroadcastChat / Export() are
// renderer-level surfaces subject to the same Trojan-Source (CVE-2021-42574)
// spoofing, so the dangerous bytes must be stripped before persistence too.
//
// CRITICAL: HandleInject does NOT hold hub.mu during WriteInput, chatAppendFn, or
// BroadcastChat — follows the ResizeClient unlock-before-IO discipline (Pitfall 4).
// sub.ReadOnly is read without a lock: it is set once at subscribe time and never mutated.
func (h *Hub) HandleInject(sub *Subscriber, text string) error {
	// Gate: RO clients may never write to PTY stdin (SEC-01).
	// sub.ReadOnly is set once at subscribe time; no lock required.
	if sub.ReadOnly {
		return ErrReadOnly
	}

	// IN-03: enforce an explicit, intentional size cap on the raw inject text
	// before any PTY write, so the bound does not silently depend on the
	// coder/websocket default read limit. Oversize frames are NAK'd, not written.
	if len(text) > MaxInjectTextBytes {
		return ErrInjectTooLarge
	}

	sanitized := SanitizePTYText(text)
	// IN-02: control-only input (e.g. "\x1b[2J" or "\x00") is non-empty and so
	// survives the read-pump ip.Text != "" guard, but SanitizePTYText collapses
	// it to a bare "\n". Treat a whitespace-only post-sanitize result as empty:
	// skip BOTH the PTY write (no spurious Enter keystroke) and the chat
	// persist/broadcast. Returning nil yields no NAK — there is no error, the
	// inject was simply a no-op.
	if strings.TrimSpace(sanitized) == "" {
		return nil
	}
	if err := h.WriteInput([]byte(sanitized)); err != nil {
		return err
	}

	// Persist and broadcast — read chatAppendFn under mu, then release before calling.
	h.mu.Lock()
	fn := h.chatAppendFn
	h.mu.Unlock()

	if fn != nil {
		// Persist a DISPLAY-SAFE form of the text (WR-01): the PTY already
		// received the SanitizePTYText output above, but chat.jsonl /
		// BroadcastChat / Export() render content, so strip the bidi/C0/C1
		// spoofing vectors here too (CVE-2021-42574, SEC-02).
		msg, err := fn(ChatMessage{
			AuthorID:      sub.TailnetID,
			AuthorAlias:   sub.Alias,
			Content:       SanitizeChatContent(text), // display-safe, not raw keystrokes
			SessionInject: true,
		})
		if err != nil {
			// WR-02: the PTY write above ALREADY succeeded, but the chat record
			// failed (ErrChatCapReached / ErrChatMessageTooLarge under normal
			// operation). Do NOT swallow it — surface a NAK so the originating
			// client learns the terminal and chat thread diverged. The PTY write
			// is deliberately NOT rolled back: the inject's primary job (reach the
			// terminal) succeeded; only the chat mirror failed. Returning the
			// error lets the read pump emit MakeInjectErrorFrame to the sender.
			return fmt.Errorf("%w: %v", ErrInjectNotRecorded, err)
		}
		// BroadcastChat acquires its own lock — must not hold hub.mu here.
		h.BroadcastChat(MakeChatFrame(msg))
	}

	return nil
}

// HandleChatSend is called by the read pump when a MsgChatSend frame arrives.
// It: (1) sanitizes the content via SanitizeChatContent (T-154-01); (2) if the
// sanitized result is empty, returns nil silently (no-op, matching MsgTyping
// behavior); (3) persists the message via chatAppendFn and broadcasts a MsgChat
// frame to all subscribers.
//
// D-06 reconciliation (Phase 163): RO clients are full chat participants — they
// may post chat, set typing indicators, and set their alias. ONLY MsgInput (PTY
// keystrokes) and MsgSessionInject (@session) remain RO-gated. The SEC-01
// ErrChatReadOnly gate (T-154-03) has been removed to honor D-06.
//
// CRITICAL: HandleChatSend NEVER calls WriteInput — chat send must not touch
// PTY stdin. Only MsgSessionInject (0x35) writes to PTY (T-154-02, D-02).
//
// Unlock-before-IO discipline (Pitfall 4): hub.mu is released before calling
// chatAppendFn or BroadcastChat, mirroring HandleInject (hub.go:520–522).
// sub.ReadOnly is read without a lock: it is set once at subscribe time and
// never mutated.
func (h *Hub) HandleChatSend(sub *Subscriber, content string) error {
	// T-154-01: sanitize for display-safety (bidi/C0/C1 stripping). Do NOT
	// use SanitizePTYText — that function is for PTY stdin, not chat content.
	sanitized := SanitizeChatContent(content)
	if sanitized == "" {
		// Silent no-op: matches MsgTyping behavior for control-char-only input.
		return nil
	}

	// Persist and broadcast — read chatAppendFn under mu, then release before calling.
	h.mu.Lock()
	fn := h.chatAppendFn
	h.mu.Unlock()

	if fn == nil {
		return fmt.Errorf("relay: HandleChatSend: chatAppendFn not wired")
	}

	msg, err := fn(ChatMessage{
		AuthorID:    sub.TailnetID,
		AuthorAlias: sub.Alias,
		Content:     sanitized,
		// SessionInject is deliberately false: this is chat-send, not inject.
	})
	if err != nil {
		return fmt.Errorf("relay: HandleChatSend: persist failed: %w", err)
	}
	// BroadcastChat acquires its own lock — must not hold hub.mu here.
	h.BroadcastChat(MakeChatFrame(msg))
	return nil
}

// InjectErrorReason maps a HandleInject error to a stable, client-safe reason
// string for MakeInjectErrorFrame. Internal plumbing detail — e.g. a PTY
// "io: read/write on closed pipe" surfaced by WriteInput — must never reach a
// remote client, so unrecognized (write) errors collapse to a generic
// "inject failed" string. Callers log the detailed error server-side (IN-01).
func InjectErrorReason(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrReadOnly):
		return "inject rejected: read-only access"
	case errors.Is(err, ErrInjectTooLarge):
		return "inject rejected: text too large"
	case errors.Is(err, ErrInjectNotRecorded):
		return "inject delivered to terminal but not recorded in chat"
	default:
		return "inject failed"
	}
}

// ScrollbackSnapshot returns a copy of the current scrollback buffer contents.
// Callers should Subscribe before calling ScrollbackSnapshot to avoid missing
// frames written between the snapshot and the first message in Msgs.
func (h *Hub) ScrollbackSnapshot() []byte {
	return h.scrollback.Snapshot()
}

// EnsureLiveEmulator lazily constructs the hub's live per-hub VT emulator on
// first call (T-175-04-01 — never built for sessions that never get a shared
// viewer) and bootstraps it once from the CURRENT ScrollbackSnapshot so its
// state matches the live PTY before Run()'s drain loop starts feeding it
// further frames. Idempotent: a second call is a no-op — once constructed,
// the emulator is never rebuilt or re-bootstrapped from scrollback again, so
// its state survives any later wrap of the raw 256 KiB ring (BUG-04, #119;
// RESEARCH "Pitfall 3": must feed the LIVE stream continuously, never
// reconstruct from the truncated ring).
//
// Callers (the WS handler sites) should call this once per new connection,
// before RenderSnapshot, ahead of any reconnect preamble write.
func (h *Hub) EnsureLiveEmulator() {
	h.emuMu.Lock()
	defer h.emuMu.Unlock()
	if h.liveEmu != nil {
		return
	}

	cols, rows := h.Cols(), h.Rows()
	emu := xvt.NewEmulator(cols, rows)
	// T-175-04-01: bound the emulator's own scrollback — it is only ever used
	// to render the CURRENT screen, never scrolled through.
	emu.SetScrollbackSize(liveEmulatorScrollbackLines)

	bootstrap := stripMsgOutputBytes(h.ScrollbackSnapshot())
	clean := liveEmuQueryStripPattern.ReplaceAll(bootstrap, nil)
	emu.Write(clean) //nolint:errcheck // emulator Write never returns a meaningful error

	h.liveEmu = emu
}

// RenderSnapshot returns the live emulator's current screen as a
// MsgOutput-framed reconnect preamble (BUG-04, #119): when the emulator
// reports IsAltScreen(), altScreenEnterSeq is prefixed so a reconnecting or
// late-joining client's xterm.js re-enters alternate-screen mode BEFORE
// painting the replayed content — reconstructing the mode marker the raw
// scrollback ring may have wrapped past.
//
// Callers MUST call EnsureLiveEmulator before RenderSnapshot — this method
// does not construct the emulator itself. Returns nil if no emulator has
// been constructed yet, OR if there is genuinely nothing to show (fresh
// session, no PTY output at all, main screen) — matching the pre-existing
// scrollback-replay contract's `len(snapshot) > 0` guard at both call sites,
// so connecting to a brand-new session does not emit a spurious empty
// preamble frame ahead of the first real PTY output.
func (h *Hub) RenderSnapshot() []byte {
	h.emuMu.Lock()
	defer h.emuMu.Unlock()
	if h.liveEmu == nil {
		return nil
	}

	// Render() pads every row up to the emulator's full height (default 50 —
	// Hub.Rows()'s fallback) with trailing blank lines. Trim them so a normal
	// (non-alt-screen) session's preamble is byte-identical to what raw PTY
	// output would have produced, matching the pre-existing scrollback-replay
	// contract for ordinary (non-wrapped, non-full-screen) sessions. Mirrors
	// GetSessionStyledTailLines' own "trim trailing blank rows" convention
	// (internal/daemon/engine.go).
	content := strings.TrimRight(h.liveEmu.Render(), "\n")
	altScreen := h.liveEmu.IsAltScreen()
	if content == "" && !altScreen {
		// Nothing to show yet, and no mode marker to reconstruct — sending an
		// empty frame would be pure overhead and would break the "no data
		// received before real output arrives" contract callers rely on.
		return nil
	}
	if altScreen {
		content = altScreenEnterSeq + content
	}
	return MakeOutputFrame([]byte(content))
}

// Cols returns the current PTY column width as set by the host-authority resize
// arbiter. Returns 220 when no resize has been applied (fallback for
// scrollback VT extraction — wide enough to avoid spurious line wrapping).
// Phase 139 / CARD-05.
func (h *Hub) Cols() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.ptyCols <= 0 {
		return 220
	}
	return h.ptyCols
}

// Rows returns the current PTY row height as set by the host-authority resize
// arbiter. Returns 50 when no resize has been applied (fallback mirrors
// engine.go emuRows default; needed by VIEW-03 join-push in Plan 02).
func (h *Hub) Rows() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.ptyRows <= 0 {
		return 50
	}
	return h.ptyRows
}

// Done returns a channel that is closed when the hub shuts down.
func (h *Hub) Done() <-chan struct{} {
	return h.done
}
