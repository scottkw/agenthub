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
