package relay

// Phase 175 Plan 04 Task 2 — BUG-04 (#119, Problem 2): reconnect reconstructs
// alt-screen mode after the 256 KiB scrollback ring wraps past a single
// ESC[?1049h (alt-screen enter) escape sequence.
//
// Root cause (RESEARCH.md "Bug 4"): a full-screen TUI (Claude Code, Gemini
// CLI, OpenCode) emits ESC[?1049h once, then only differential repaints. If
// the 256 KiB ring (internal/relay/scrollback.go) wraps past that single
// mode-set sequence, the raw bytes replayed to a newly-connecting client
// (hub.ScrollbackSnapshot()) are alt-screen *content* with no mode-switch
// marker — xterm.js paints them into the wrong (main) buffer, producing a
// blank/garbled window on reconnect.
//
// Fix (175-04): Hub.EnsureLiveEmulator lazily constructs a continuously-fed
// VT emulator (mirroring internal/daemon/engine.go:773-818's
// GetSessionStyledTailLines precedent) whose Hub.RenderSnapshot() reconnect
// preamble re-establishes alt-screen mode + current screen content,
// independent of the raw scrollback ring's later wraps.
//
// This test simulates the realistic ordering: EnsureLiveEmulator is called
// once, right after the TUI's alt-screen-enter preamble arrives (mirroring
// the near-immediate loopback WS connection the desktop owner's own
// TerminalPanel makes when a session tab opens — relay/server.go's
// handleSession). The emulator then survives — continuously fed by
// Run()'s drain loop — while the RAW scrollback ring wraps past the
// preamble far later, proving RenderSnapshot() never re-derives from the
// (by-then-truncated) ring (RESEARCH "Pitfall 3").

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/testutil"
)

// altScreenEnter is the DEC private-mode sequence a full-screen TUI emits
// exactly once on entering the alternate screen buffer.
const altScreenEnter = "\x1b[?1049h"

func TestScrollbackAltScreenReplay(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("altscreen-wrap-test", r, w, DefaultScrollbackBytes, nil)

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	defer func() {
		_ = w.Close()
		<-done
	}()

	// Write the alt-screen enter sequence once, followed by a small amount of
	// alt-screen content — this is exactly what a real full-screen TUI emits
	// once on startup (RESEARCH "Bug 4").
	altScreenPreamble := []byte(altScreenEnter + "alt-screen content line 1\r\n")
	if _, err := w.Write(altScreenPreamble); err != nil {
		t.Fatalf("write alt-screen preamble: %v", err)
	}

	// EnsureLiveEmulator NOW, immediately after the preamble arrives and
	// BEFORE the ring-wrapping filler below — mirrors the realistic ordering
	// where the desktop owner's own loopback WS connection (relay/server.go
	// handleSession) is already live from session start, well before any
	// remote guest's later reconnect. The bootstrap below reads
	// ScrollbackSnapshot() while it still contains the preamble; from this
	// point on Run()'s drain loop keeps the emulator continuously fed
	// (feedLiveEmulator), so its alt-screen state survives the ring wrap
	// that follows.
	hub.EnsureLiveEmulator()

	// Feed enough filler PTY output to wrap the 256 KiB ring PAST the
	// preamble above. Writing well past DefaultScrollbackBytes guarantees the
	// ring has fully cycled at least once, discarding the preamble's bytes.
	filler := bytes.Repeat([]byte("x"), 4*1024)
	totalWritten := len(altScreenPreamble)
	for totalWritten <= DefaultScrollbackBytes+len(altScreenPreamble) {
		if _, err := w.Write(filler); err != nil {
			t.Fatalf("write filler: %v", err)
		}
		totalWritten += len(filler)
	}

	// Subscribe a fresh client — mirrors a guest reconnecting after the wrap.
	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "web"}
	hub.Subscribe(sub)

	// Sanity check: prove the ring has actually wrapped past the alt-screen
	// enter sequence — the raw snapshot must no longer contain it. This is
	// the fixture-validity assertion required by the plan's acceptance
	// criteria (independent of the RED assertion below).
	rawSnapshot := hub.ScrollbackSnapshot()
	if bytes.Contains(rawSnapshot, []byte(altScreenEnter)) {
		t.Fatalf("fixture did not wrap the ring past %q — raw snapshot still contains it "+
			"(len=%d); increase the filler volume", altScreenEnter, len(rawSnapshot))
	}

	// The GREEN assertion: the emulator-derived reconnect preamble (not the
	// raw, by-now-truncated scrollback) still reconstructs alt-screen mode.
	// EnsureLiveEmulator is idempotent — this call is a no-op (the emulator
	// was already constructed above, before the ring wrapped).
	hub.EnsureLiveEmulator()
	reconnectPreamble := hub.RenderSnapshot()
	if !bytes.Contains(reconnectPreamble, []byte(altScreenEnter)) {
		t.Errorf("reconnect preamble does not reconstruct alt-screen mode: missing %q in %q",
			altScreenEnter, reconnectPreamble)
	}
	// The preamble must also carry the current screen content (not just the
	// mode marker) — proving a reconnecting client sees real content, not an
	// empty alt-screen buffer.
	if !bytes.Contains(reconnectPreamble, []byte("x")) {
		t.Errorf("reconnect preamble does not contain current screen content (filler 'x'): %q",
			reconnectPreamble)
	}
}

// TestReconnectPreamble_RawWhenNotAltScreen is the #109 regression guard: a
// normal (non-alt-screen) session must replay the BYTE-FAITHFUL raw scrollback,
// not the emulator's re-rendered grid — the rendered grid desyncs the guest's
// xterm from the host TUI and reintroduces the guest layout garble (#109).
func TestReconnectPreamble_RawWhenNotAltScreen(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("preamble-mainscreen-test", r, w, DefaultScrollbackBytes, nil)
	done := make(chan struct{})
	go func() { hub.Run(); close(done) }()
	defer func() { _ = w.Close(); <-done }()

	// Plain main-screen output with a distinctive escape sequence that would NOT
	// survive an emulator render unchanged.
	payload := []byte("\x1b[1mBold\x1b[0m normal line\r\n")
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	testutil.WaitFor(t, 2*time.Second, func() bool { return len(hub.ScrollbackSnapshot()) > 0 }, "scrollback never populated")
	hub.EnsureLiveEmulator()

	if got, raw := hub.ReconnectPreamble(), hub.ScrollbackSnapshot(); !bytes.Equal(got, raw) {
		t.Errorf("non-alt-screen must use byte-faithful raw replay (#109); got %q, want raw %q", got, raw)
	}
}

// TestReconnectPreamble_RawWhileAltMarkerInRing: an alt-screen session whose
// ESC[?1049h marker is STILL in the ring must ALSO use raw replay — raw is
// byte-faithful and self-sufficient here, so #109 stays fixed.
func TestReconnectPreamble_RawWhileAltMarkerInRing(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("preamble-alt-marker-present-test", r, w, DefaultScrollbackBytes, nil)
	done := make(chan struct{})
	go func() { hub.Run(); close(done) }()
	defer func() { _ = w.Close(); <-done }()

	if _, err := w.Write([]byte(altScreenEnter + "alt content line\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	testutil.WaitFor(t, 2*time.Second, func() bool { return bytes.Contains(hub.ScrollbackSnapshot(), []byte(altScreenEnter)) }, "alt marker never appeared in ring")
	hub.EnsureLiveEmulator()

	raw := hub.ScrollbackSnapshot()
	if !bytes.Contains(raw, []byte(altScreenEnter)) {
		t.Fatalf("fixture invalid: marker already gone from ring")
	}
	if got := hub.ReconnectPreamble(); !bytes.Equal(got, raw) {
		t.Errorf("alt-screen with marker still in ring must use raw replay (#109); got rendered grid instead")
	}
}

// TestReconnectPreamble_EmulatorFallbackAfterWrap: only once the ring has
// wrapped PAST the alt-screen-enter marker does the preamble fall back to the
// emulator snapshot (BUG-04 #119 Problem 2) — the sole case where raw replay
// would be blank/garbled.
func TestReconnectPreamble_EmulatorFallbackAfterWrap(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("preamble-alt-wrapped-test", r, w, DefaultScrollbackBytes, nil)
	done := make(chan struct{})
	go func() { hub.Run(); close(done) }()
	defer func() { _ = w.Close(); <-done }()

	pre := []byte(altScreenEnter + "alt-screen content line 1\r\n")
	if _, err := w.Write(pre); err != nil {
		t.Fatalf("write preamble: %v", err)
	}
	hub.EnsureLiveEmulator()

	filler := bytes.Repeat([]byte("x"), 4*1024)
	total := len(pre)
	for total <= DefaultScrollbackBytes+len(pre) {
		if _, err := w.Write(filler); err != nil {
			t.Fatalf("write filler: %v", err)
		}
		total += len(filler)
	}
	testutil.WaitFor(t, 2*time.Second, func() bool { return !bytes.Contains(hub.ScrollbackSnapshot(), []byte(altScreenEnter)) }, "ring never wrapped past alt marker")

	raw := hub.ScrollbackSnapshot()
	got := hub.ReconnectPreamble()
	if bytes.Equal(got, raw) {
		t.Errorf("after ring wrapped past the alt marker, expected emulator fallback, got raw replay")
	}
	if !bytes.Contains(got, []byte(altScreenEnter)) {
		t.Errorf("emulator fallback must reconstruct the alt-screen marker; got %q", got)
	}
}

// TestReconnectPreamble_EmulatorWhenMainScreenRingWrapped is the M-51 guard:
// a MAIN-SCREEN full-screen app (top/htop on macOS render in the main screen via
// absolute cursor positioning, NOT the alternate screen) whose 256 KiB ring has
// WRAPPED must reconnect from the emulator's complete snapshot — NOT the raw,
// now-incomplete tail (which replays scrambled columns). This fails under the old
// alt-screen-only discriminator, which took the raw branch for any non-alt-screen
// session regardless of ring truncation.
func TestReconnectPreamble_EmulatorWhenMainScreenRingWrapped(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("preamble-mainscreen-wrapped-test", r, w, DefaultScrollbackBytes, nil)
	done := make(chan struct{})
	go func() { hub.Run(); close(done) }()
	defer func() { _ = w.Close(); <-done }()

	// Main-screen output only — NO alt-screen enter sequence anywhere.
	if _, err := w.Write([]byte("early main-screen content\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	hub.EnsureLiveEmulator()

	filler := bytes.Repeat([]byte("y"), 4*1024)
	total := 0
	for total <= DefaultScrollbackBytes+4*1024 {
		if _, err := w.Write(filler); err != nil {
			t.Fatalf("write filler: %v", err)
		}
		total += len(filler)
	}
	testutil.WaitFor(t, 2*time.Second, func() bool { return hub.scrollback.Truncated() }, "ring never reported truncated")

	raw := hub.ScrollbackSnapshot()
	if bytes.Contains(raw, []byte(altScreenEnter)) {
		t.Fatalf("fixture invalid: unexpected alt-screen marker present")
	}
	got := hub.ReconnectPreamble()
	if len(got) == 0 {
		t.Fatal("ReconnectPreamble returned nothing")
	}
	if bytes.Equal(got, raw) {
		t.Errorf("main-screen session with a WRAPPED ring must reconnect from the emulator's "+
			"complete snapshot, not the incomplete raw tail (M-51 top garble)")
	}
}

// TestReconnectPreamble_EagerEmulatorCapturesHeaderBeforeWrap is the M-51
// eager-emulator guard. It simulates top's in-place positioned header: a
// full-screen MAIN-screen app writes a header ONCE at row 1 (absolute cursor
// home), then only ever repaints a lower body region — it NEVER rewrites the
// header structure (top overwrites only the changing numbers in place). When
// the raw 256 KiB ring wraps past that one-time header write and a guest joins
// LATE (its EnsureLiveEmulator call comes only AFTER the wrap), the emulator
// can only reconstruct the header if it has been alive and fed since BEFORE the
// wrap — i.e. eagerly, from the first PTY byte.
//
// This is RED with the old lazy build: EnsureLiveEmulator (called late, at the
// guest connect) bootstraps a FRESH emulator from the already-truncated ring,
// which no longer contains the header write, so row 1 renders blank/garbled.
// It is GREEN with the eager build: the emulator saw the header at byte 1 and
// still holds it in row 1 at reconnect time.
func TestReconnectPreamble_EagerEmulatorCapturesHeaderBeforeWrap(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("preamble-eager-header-test", r, w, DefaultScrollbackBytes, nil)
	done := make(chan struct{})
	go func() { hub.Run(); close(done) }()
	defer func() { _ = w.Close(); <-done }()

	// The header is written ONCE at absolute home (row 1) — exactly what a
	// full-screen app emits at startup and never structurally rewrites.
	if _, err := w.Write([]byte("\x1b[H" + "HEADER-XYZ")); err != nil {
		t.Fatalf("write header: %v", err)
	}

	// DELIBERATELY do NOT call EnsureLiveEmulator here. The realistic M-51
	// scenario is a LATE-joining guest: the emulator's first (lazy) construction
	// would come only at the guest connect below, AFTER the ring has wrapped.

	// Body repaints: reposition to row 5 each cycle and rewrite a lower region,
	// exactly like top's process rows. These never touch row 1, so the header
	// stays in the emulator grid — but they DO fill the raw ring and wrap it past
	// the one-time header write above.
	bodyUpdate := append([]byte("\x1b[5H"), bytes.Repeat([]byte("y"), 40)...)
	total := len("\x1b[H" + "HEADER-XYZ")
	for total <= DefaultScrollbackBytes+1024 {
		if _, err := w.Write(bodyUpdate); err != nil {
			t.Fatalf("write body update: %v", err)
		}
		total += len(bodyUpdate)
	}

	// Wait until the ring has actually wrapped past the header write — the raw
	// tail must no longer contain it (fixture-validity + reconnect discriminator).
	testutil.WaitFor(t, 2*time.Second, func() bool {
		return hub.scrollback.Truncated() && !bytes.Contains(hub.ScrollbackSnapshot(), []byte("HEADER-XYZ"))
	}, "ring never wrapped past the header write")

	raw := hub.ScrollbackSnapshot()
	if bytes.Contains(raw, []byte("HEADER-XYZ")) {
		t.Fatalf("fixture invalid: raw ring still contains the header write")
	}

	// A guest joins LATE and its WS handler calls EnsureLiveEmulator for the
	// first time — AFTER the wrap. The reconnect preamble (wrapped → emulator)
	// must still carry the header, which is only possible if the emulator was
	// fed eagerly from the first PTY byte.
	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "web"}
	hub.Subscribe(sub)
	hub.EnsureLiveEmulator()

	got := hub.ReconnectPreamble()
	if len(got) == 0 {
		t.Fatal("ReconnectPreamble returned nothing")
	}
	if !bytes.Contains(got, []byte("HEADER-XYZ")) {
		t.Errorf("late-join reconnect preamble lost the one-time header write (M-51): "+
			"the emulator was not fed before the ring wrapped; got %q", got)
	}
}

// TestLiveEmulatorFollowsResize is the CR-01 regression guard (code review
// 175-REVIEW.md): the live per-hub VT emulator is built once at the initial PTY
// size in EnsureLiveEmulator and must follow later authoritative PTY resizes,
// or RenderSnapshot's reconnect preamble is mis-dimensioned after a guest joins
// with a smaller viewport (ResizeClient's host-authority min-arbiter shrinks the
// PTY — the core BUG-04 multi-viewer path). Without resizeLiveEmulator this test
// fails: the emulator stays at the 220x50 fallback geometry after the resize.
func TestLiveEmulatorFollowsResize(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("resize-follow-test", r, w, DefaultScrollbackBytes, nil)

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	defer func() {
		_ = w.Close()
		<-done
	}()

	// Some initial PTY output, then build the emulator. No ResizeClient has run
	// yet, so it is built at the Cols()/Rows() fallback geometry (220x50).
	if _, err := w.Write([]byte("initial output\r\n")); err != nil {
		t.Fatalf("write initial output: %v", err)
	}
	hub.EnsureLiveEmulator()

	// Read geometry under emuMu — the drain goroutine mutates liveEmu under the
	// same lock, so an unguarded read would race (-race).
	readGeom := func() (int, int) {
		hub.emuMu.Lock()
		defer hub.emuMu.Unlock()
		return hub.liveEmu.Width(), hub.liveEmu.Height()
	}

	if gotW, gotH := readGeom(); gotW != 220 || gotH != 50 {
		t.Fatalf("pre-resize emulator geometry = %dx%d, want 220x50 (fallback)", gotW, gotH)
	}

	// A local-origin subscriber joins reporting a small viewport. ResizeClient's
	// min-arbiter makes 40x10 the authoritative PTY grid and (with the fix)
	// propagates it to the live emulator. resizeFn is nil here, which is fine —
	// ResizeClient still runs broadcastResize + resizeLiveEmulator.
	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	hub.Subscribe(sub)
	if err := hub.ResizeClient(sub, 40, 10); err != nil {
		t.Fatalf("ResizeClient: %v", err)
	}

	if gotW, gotH := readGeom(); gotW != 40 || gotH != 10 {
		t.Errorf("post-resize emulator geometry = %dx%d, want 40x10 — the live emulator did "+
			"not follow the PTY resize (CR-01: RenderSnapshot would be mis-dimensioned)", gotW, gotH)
	}
}

// TestLiveEmulatorResizeDiscardsStaleContent is the M-51 resize-churn guard
// (debug session m51-top-header-garble). xvt.Emulator.Resize() is a DESTRUCTIVE,
// non-reflow grid truncate/pad: after a mid-session geometry change, content laid
// out for the PRE-resize geometry persists on the wrong rows. A full-screen app
// (top/htop/vim) fully redraws on SIGWINCH, so the CORRECT post-resize screen is
// derivable entirely from post-resize bytes; any surviving pre-resize content is
// stale, wrong-geometry garbage that never self-heals (top only re-lays-out its
// header on resize, so a partial in-place body update afterwards never rewrites
// the header rows). This is the residual M-51 header garble.
//
// The fix rebuilds the live emulator EMPTY at the new geometry on every resize,
// so pre-resize content cannot survive into the reconnect preamble. This test is
// RED with the old destructive emu.Resize() (the STALE header at row 1 survives
// the shrink and a subsequent lower-row body update never touches it) and GREEN
// once resizeLiveEmulator discards + rebuilds.
func TestLiveEmulatorResizeDiscardsStaleContent(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("resize-discard-stale-test", r, w, DefaultScrollbackBytes, nil)
	done := make(chan struct{})
	go func() { hub.Run(); close(done) }()
	defer func() { _ = w.Close(); <-done }()

	// A full-screen app writes a positioned header at row 1 (absolute home) — the
	// structural header top draws ONCE and only patches in place thereafter.
	if _, err := w.Write([]byte("\x1b[H" + "STALE-HEADER")); err != nil {
		t.Fatalf("write header: %v", err)
	}
	// Synchronize: wait until the eagerly-built emulator has actually rendered the
	// header before we resize (otherwise the feed could race the resize).
	testutil.WaitFor(t, 2*time.Second, func() bool {
		return bytes.Contains(hub.RenderSnapshot(), []byte("STALE-HEADER"))
	}, "emulator never captured the pre-resize header")

	// A local viewer's geometry change churns the PTY/emulator grid (window resize,
	// sidebar toggle, font-size change, or xterm.js FitAddon's mount-then-settle
	// double-fit). ResizeClient's min-arbiter drives resizeLiveEmulator(40,10).
	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	hub.Subscribe(sub)
	if err := hub.ResizeClient(sub, 40, 10); err != nil {
		t.Fatalf("ResizeClient: %v", err)
	}

	// After the resize, top emits a partial in-place body update at a LOWER row —
	// it does NOT rewrite the row-1 header (top only re-lays-out the header on the
	// SIGWINCH redraw itself, which a real full redraw would have fully replaced).
	// This models the state where the header rows are left untouched post-resize.
	if _, err := w.Write([]byte("\x1b[3H" + "new-body-row")); err != nil {
		t.Fatalf("write body update: %v", err)
	}
	testutil.WaitFor(t, 2*time.Second, func() bool {
		return bytes.Contains(hub.RenderSnapshot(), []byte("new-body-row"))
	}, "emulator never captured the post-resize body update")

	// The reconnect preamble must NOT carry pre-resize content: a rebuilt-empty
	// emulator holds only post-resize bytes. If STALE-HEADER survives, the guest
	// sees a header laid out for the wrong geometry (the M-51 garble).
	got := hub.RenderSnapshot()
	if bytes.Contains(got, []byte("STALE-HEADER")) {
		t.Errorf("pre-resize header survived the resize into the reconnect preamble "+
			"(M-51 resize-churn garble): destructive emu.Resize() kept stale row-1 content; "+
			"got %q", got)
	}
}
