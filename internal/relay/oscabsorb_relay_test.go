package relay

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// Phase 115 / Issue #60: integration tests that exercise InputAbsorber on the
// daemon-direct relay path (handleSession). Mirrors the Phase 111 webserver-
// layer integration suite (internal/webserver/oscabsorb_relay_test.go) but
// targets the relay.Server directly — the path used by the Wails desktop
// attach and by CLI `agenthub attach`. These two surfaces bypass the
// webserver wrapper and were leaking OSC 10/11 + DA1 responses into PTY
// stdin until this phase landed.

// readPipeMustTimeout asserts that no data arrives at r within the given
// timeout — i.e. the input was absorbed before reaching the PTY.
func readPipeMustTimeout(t *testing.T, r *io.PipeReader, timeout time.Duration, assertion string) {
	t.Helper()
	readDone := make(chan []byte, 1)
	readErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			readErr <- err
			return
		}
		readDone <- buf[:n]
	}()
	select {
	case data := <-readDone:
		t.Fatalf("%s — PTY pipe received %d bytes %q (expected absorption)", assertion, len(data), data)
	case err := <-readErr:
		t.Fatalf("%s — pipe read error: %v", assertion, err)
	case <-time.After(timeout):
		// success
	}
}

// readPipeWithTimeout reads bytes from r with the given timeout; fails the
// test if no bytes arrive (used to assert that keystrokes DO pass through).
func readPipeWithTimeout(t *testing.T, r *io.PipeReader, timeout time.Duration) []byte {
	t.Helper()
	readDone := make(chan []byte, 1)
	readErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			readErr <- err
			return
		}
		readDone <- buf[:n]
	}()
	select {
	case data := <-readDone:
		return data
	case err := <-readErr:
		t.Fatalf("pipe read: %v", err)
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for pipe data")
	}
	return nil
}

func TestRelay_handleSession_OSC10ReplyAbsorbed(t *testing.T) {
	srv, _, _, inputRead, sessionID := setupTestServer(t)
	conn := dialWS(t, srv.URL, sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := []byte("\x1b]10;rgb:cccc/cccc/cccc\x1b\\")
	if err := conn.Write(ctx, websocket.MessageBinary, MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	readPipeMustTimeout(t, inputRead, 300*time.Millisecond,
		"OSC 10 reply must be absorbed before reaching PTY (daemon-direct relay path)")
}

func TestRelay_handleSession_OSC11ReplyAbsorbed(t *testing.T) {
	srv, _, _, inputRead, sessionID := setupTestServer(t)
	conn := dialWS(t, srv.URL, sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := []byte("\x1b]11;rgb:1d1d/1f1f/2121\x1b\\")
	if err := conn.Write(ctx, websocket.MessageBinary, MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	readPipeMustTimeout(t, inputRead, 300*time.Millisecond,
		"OSC 11 reply must be absorbed before reaching PTY (daemon-direct relay path)")
}

func TestRelay_handleSession_DA1ReplyAbsorbed(t *testing.T) {
	srv, _, _, inputRead, sessionID := setupTestServer(t)
	conn := dialWS(t, srv.URL, sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := []byte("\x1b[?62;4;9;22c")
	if err := conn.Write(ctx, websocket.MessageBinary, MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	readPipeMustTimeout(t, inputRead, 300*time.Millisecond,
		"DA1 reply must be absorbed before reaching PTY (daemon-direct relay path)")
}

// TestRelay_handleSession_KeystrokesStillForwarded — regression guard. The
// absorber wiring must not block ordinary user input.
func TestRelay_handleSession_KeystrokesStillForwarded(t *testing.T) {
	srv, _, _, inputRead, sessionID := setupTestServer(t)
	conn := dialWS(t, srv.URL, sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := []byte("ls -la\r")
	if err := conn.Write(ctx, websocket.MessageBinary, MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	got := readPipeWithTimeout(t, inputRead, 500*time.Millisecond)
	if !bytes.Equal(got, payload) {
		t.Fatalf("keystrokes did not arrive at PTY intact: got %q, want %q", got, payload)
	}
}

// TestRelay_handleSession_OSC11SplitAcrossTwoFrames proves per-subscriber
// absorber state survives MsgInput frame boundaries on the daemon-direct
// relay path (mirrors the webserver-layer test).
func TestRelay_handleSession_OSC11SplitAcrossTwoFrames(t *testing.T) {
	srv, _, _, inputRead, sessionID := setupTestServer(t)
	conn := dialWS(t, srv.URL, sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Split mid-envelope — first frame opens OSC 11; second frame closes it.
	frame1 := []byte("\x1b]11;rgb:1d1d/")
	frame2 := []byte("1f1f/2121\x1b\\")

	if err := conn.Write(ctx, websocket.MessageBinary, MakeInputFrame(frame1)); err != nil {
		t.Fatalf("write frame1: %v", err)
	}
	if err := conn.Write(ctx, websocket.MessageBinary, MakeInputFrame(frame2)); err != nil {
		t.Fatalf("write frame2: %v", err)
	}
	readPipeMustTimeout(t, inputRead, 300*time.Millisecond,
		"OSC 11 split across two frames must be absorbed in full (state must persist between frames)")
}

// TestRelay_handleSession_MixedReplyAndKeystrokes — single MsgInput frame
// carrying both an absorbable reply and real user keystrokes; only the
// keystrokes should reach the PTY.
func TestRelay_handleSession_MixedReplyAndKeystrokes(t *testing.T) {
	srv, _, _, inputRead, sessionID := setupTestServer(t)
	conn := dialWS(t, srv.URL, sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// "ls\r" + OSC 11 reply + "pwd\r"
	payload := []byte("ls\r\x1b]11;rgb:1d1d/1f1f/2121\x1b\\pwd\r")
	if err := conn.Write(ctx, websocket.MessageBinary, MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}

	// Drain reads until we either accumulate the expected keystrokes or time
	// out — the absorber may emit "ls\r" and "pwd\r" as one or two writes
	// depending on internal buffering.
	want := []byte("ls\rpwd\r")
	var got []byte
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) && len(got) < len(want) {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		got = append(got, readPipeWithTimeout(t, inputRead, remaining)...)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("mixed payload at PTY: got %q, want %q", got, want)
	}
}
