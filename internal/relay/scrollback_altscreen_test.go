package relay

// Phase 175 Plan 02 Task 3 — Wave 0 RED scaffold for BUG-04 (#119, Problem 2):
// reconnect reconstructs alt-screen mode after the 256 KiB scrollback ring
// wraps past a single ESC[?1049h (alt-screen enter) escape sequence.
//
// Root cause (RESEARCH.md "Bug 4"): a full-screen TUI (Claude Code, Gemini
// CLI, OpenCode) emits ESC[?1049h once, then only differential repaints. If
// the 256 KiB ring (internal/relay/scrollback.go) wraps past that single
// mode-set sequence, the raw bytes replayed to a newly-connecting client
// (hub.ScrollbackSnapshot()) are alt-screen *content* with no mode-switch
// marker — xterm.js paints them into the wrong (main) buffer, producing a
// blank/garbled window on reconnect.
//
// 175-04 is expected to add a live, continuously-fed VT emulator per Hub
// (mirroring internal/daemon/engine.go:773-818's GetSessionStyledTailLines
// precedent) whose reconnect preamble re-establishes alt-screen mode +
// current screen content instead of, or in addition to, the raw
// ScrollbackSnapshot(). This test is skip-guarded until that method exists —
// see the inline "175-04:" marker below for the assertion swap.

import (
	"bytes"
	"io"
	"testing"
)

// altScreenEnter is the DEC private-mode sequence a full-screen TUI emits
// exactly once on entering the alternate screen buffer.
const altScreenEnter = "\x1b[?1049h"

func TestScrollbackAltScreenReplay(t *testing.T) {
	t.Skip("RED until 175-04 adds the live per-hub VT emulator + RenderSnapshot — BUG-04")

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

	// 175-04: switch this to the new emulator-derived reconnect preamble
	// (e.g. hub.RenderSnapshot()) and assert it reconstructs alt-screen mode
	// (contains altScreenEnter or an equivalent mode marker) plus the current
	// screen content — proving a reconnecting client no longer sees a blank
	// window. The method does not exist yet, so this assertion is left
	// against ScrollbackSnapshot() (known-losing today) purely so this file
	// compiles ahead of 175-04 introducing the new method.
	reconnectPreamble := hub.ScrollbackSnapshot()
	if !bytes.Contains(reconnectPreamble, []byte(altScreenEnter)) {
		t.Errorf("reconnect preamble does not reconstruct alt-screen mode: missing %q "+
			"(BUG-04 — raw ScrollbackSnapshot() alone loses the mode-set byte sequence "+
			"after ring wrap; 175-04 must replace this with an emulator-derived preamble)",
			altScreenEnter)
	}
}
