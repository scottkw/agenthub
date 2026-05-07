package relay

import (
	"bytes"
	"io"
	"testing"
	"time"
)

// TestImage_ByteFidelity_MultiClient — Phase 96 IMG-04.
//
// Verifies that sixel/IIP escape sequences pass through the relay
// byte-buffer architecture VERBATIM to multiple subscribers, both
// for real-time fan-out and for ScrollbackSnapshot replay (the
// mid-stream-join scenario for late-joining clients).
//
// This test runs GREEN immediately. Per 96-RESEARCH.md §"Architectural
// Responsibility Map" + §"Cross-tier note for IMG-04", the relay
// architecture (internal/relay/scrollback.go raw byte buffer +
// internal/relay/hub.go 32 KiB chunk pass-through with NO line buffering
// and NO escape parsing) STRUCTURALLY guarantees byte-fidelity for any
// PTY output — sixel/IIP bytes are bytes like any other. No Wave 1
// implementation work is needed; this test exists to lock in regression
// defense.
//
// The 256 KiB scrollback truncation cap is acknowledged as a known
// limitation per STATE.md §Decisions and 96-RESEARCH.md §"User
// Constraints / Locked Decisions". This test stays well under that
// cap to assert byte-fidelity; the truncation behavior is a separate
// pre-existing v3.1 property and is out of scope for Phase 96.
func TestImage_ByteFidelity_MultiClient(t *testing.T) {
	// Synthetic small sixel byte stream. Parser-validity is NOT
	// material — the test asserts byte-for-byte equality on both
	// subscribers, NOT that the bytes decode to anything visually.
	sixelInput := []byte("\x1bPq#0;2;100;0;0#1;2;0;100;0!10A!10@-\x1b\\")

	pr, pw := io.Pipe()
	hub := NewHub("test-image-fidelity", pr, io.Discard, DefaultScrollbackBytes, nil)

	// Subscribe two clients BEFORE any data is written; they should
	// observe identical real-time fan-out.
	sub1 := &Subscriber{Msgs: make(chan []byte, 16), CloseSlow: func() {}}
	sub2 := &Subscriber{Msgs: make(chan []byte, 16), CloseSlow: func() {}}
	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	// Start the hub; it will read from pr until EOF.
	go hub.Run()
	defer hub.Shutdown()

	// Write the sixel stream; close the pipe to signal EOF.
	go func() {
		_, _ = pw.Write(sixelInput)
		_ = pw.Close()
	}()

	// Drain both subscribers; concatenate the payload bytes after
	// stripping the 1-byte MsgOutput frame prefix per chunk.
	var got1, got2 []byte
	timeout := time.After(2 * time.Second)
	for len(got1) < len(sixelInput) || len(got2) < len(sixelInput) {
		select {
		case frame := <-sub1.Msgs:
			if len(frame) > 0 && frame[0] == MsgOutput {
				got1 = append(got1, frame[1:]...)
			}
		case frame := <-sub2.Msgs:
			if len(frame) > 0 && frame[0] == MsgOutput {
				got2 = append(got2, frame[1:]...)
			}
		case <-timeout:
			t.Fatalf("timed out waiting for fan-out: got1=%d got2=%d want=%d", len(got1), len(got2), len(sixelInput))
		}
	}

	if !bytes.Equal(got1, sixelInput) {
		t.Errorf("client 1 received corrupted sixel stream:\nwant: %x\ngot:  %x", sixelInput, got1)
	}
	if !bytes.Equal(got2, sixelInput) {
		t.Errorf("client 2 received corrupted sixel stream:\nwant: %x\ngot:  %x", sixelInput, got2)
	}

	// Verify scrollback (mid-stream-join scenario) preserves bytes.
	// Wait briefly for hub.Run() to flush the scrollback record after
	// fan-out (the broadcast and scrollback append happen in the same
	// goroutine, but scheduling can race the Done() drain).
	deadline := time.Now().Add(2 * time.Second)
	var snapshot []byte
	for time.Now().Before(deadline) {
		snapshot = hub.ScrollbackSnapshot()
		if len(snapshot) >= 1+len(sixelInput) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// The scrollback is a sequence of framed chunks; for this single-write
	// synthetic test the entire payload arrives as one chunk preceded by
	// one MsgOutput byte. Verify framing then strip the prefix.
	if len(snapshot) == 0 {
		t.Fatalf("scrollback snapshot is empty; expected MsgOutput-framed sixel bytes")
	}
	if snapshot[0] != MsgOutput {
		t.Fatalf("scrollback frame at offset 0 has unexpected type byte 0x%02x", snapshot[0])
	}
	fromSnapshot := snapshot[1:]
	if !bytes.Equal(fromSnapshot, sixelInput) {
		t.Errorf("scrollback corrupted sixel stream:\nwant: %x\ngot:  %x", sixelInput, fromSnapshot)
	}
}
