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
