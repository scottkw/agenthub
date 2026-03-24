package main

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agenthub/agenthub/internal/relay"
	"github.com/coder/websocket"
)

// setupAttachTest creates a relay HubManager + Server backed by io.Pipe pairs
// (same pattern as relay.setupTestServer). It starts an httptest.Server and
// registers cleanup for all resources.
//
// Returns: serverURL, manager, ptyWrite (test writes PTY output here),
// inputRead (test reads PTY input here), and the session ID.
func setupAttachTest(t *testing.T) (serverURL string, mgr *relay.HubManager, ptyWrite *io.PipeWriter, inputRead *io.PipeReader, sessionID string) {
	t.Helper()

	// ptyOutput pipe: hub reads from ptyOutputR, test writes to ptyOutputW.
	ptyOutputR, ptyOutputW := io.Pipe()

	// inputCapture pipe: hub writes to inputCaptureW, test reads from inputCaptureR.
	inputCaptureR, inputCaptureW := io.Pipe()

	const sid = "attach-test-session"
	mgr = relay.NewHubManager()
	mgr.Create(sid, ptyOutputR, inputCaptureW, nil)

	srv := httptest.NewServer(relay.NewServer(mgr, nil))
	t.Cleanup(func() {
		srv.Close()
		mgr.Shutdown()
		ptyOutputW.Close()
		inputCaptureW.Close()
	})

	return srv.URL, mgr, ptyOutputW, inputCaptureR, sid
}

// dialTestWS dials a WebSocket connection to the test server for the given session ID.
func dialTestWS(t *testing.T, serverURL, sessionID string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/sessions/" + sessionID + "/ws"
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("dialTestWS: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// TestCmdAttach_MissingArgs verifies that cmdAttach returns an error containing
// "usage" when called with no arguments (CLI-05: argument validation).
func TestCmdAttach_MissingArgs(t *testing.T) {
	client := testSetup(t)
	err := cmdAttach(client, []string{})
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected error to contain %q, got %q", "usage", err.Error())
	}
}

// TestMakeClientResizeFrame verifies that makeClientResizeFrame encodes the
// resize frame using MsgResize2 (0x11) with big-endian cols and rows (CLI-06).
func TestMakeClientResizeFrame(t *testing.T) {
	// Basic case: cols=120, rows=40 -> [0x11, 0, 120, 0, 40]
	got := makeClientResizeFrame(120, 40)
	want := []byte{relay.MsgResize2, 0, 120, 0, 40}
	if !bytes.Equal(got, want) {
		t.Errorf("makeClientResizeFrame(120, 40) = %v, want %v", got, want)
	}

	// Verify first byte is MsgResize2 (0x11), NOT MsgResize (0x02).
	if got[0] != relay.MsgResize2 {
		t.Errorf("first byte = 0x%02x, want MsgResize2 (0x%02x)", got[0], relay.MsgResize2)
	}
	if got[0] == relay.MsgResize {
		t.Errorf("first byte must NOT be MsgResize (0x%02x)", relay.MsgResize)
	}

	// Big-endian encoding: cols=256 -> [1, 0]; rows=512 -> [2, 0]
	got2 := makeClientResizeFrame(256, 512)
	want2 := []byte{relay.MsgResize2, 1, 0, 2, 0}
	if !bytes.Equal(got2, want2) {
		t.Errorf("makeClientResizeFrame(256, 512) = %v, want %v", got2, want2)
	}
}

// TestAttachSession_DetachKey verifies that sending the detach key byte causes
// attachSession to return nil (clean detach without error) (CLI-07).
func TestAttachSession_DetachKey(t *testing.T) {
	serverURL, _, _, _, sessionID := setupAttachTest(t)
	conn := dialTestWS(t, serverURL, sessionID)

	var stdout bytes.Buffer
	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { stdinR.Close(); stdinW.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- attachSession(ctx, conn, stdinR, &stdout, 0x1C)
	}()

	// Write the detach key byte — this should cause a clean return.
	// Small delay to ensure attachSession goroutines are running.
	time.Sleep(10 * time.Millisecond)
	if _, err := stdinW.Write([]byte{0x1C}); err != nil {
		t.Fatalf("stdinW.Write detach key: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("attachSession returned error %v, want nil (clean detach)", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("attachSession did not return within 5s after detach key")
	}
}

// TestAttachSession_OutputReceived verifies that scrollback output written
// before connecting is replayed to stdout (CLI-05, CLI-08: scrollback replay).
func TestAttachSession_OutputReceived(t *testing.T) {
	serverURL, _, ptyWrite, _, sessionID := setupAttachTest(t)

	// Write PTY output BEFORE dialing — this goes into scrollback.
	const payload = "hello world"
	if _, err := ptyWrite.Write([]byte(payload)); err != nil {
		t.Fatalf("ptyWrite: %v", err)
	}

	// Give hub time to process and store in scrollback.
	time.Sleep(30 * time.Millisecond)

	conn := dialTestWS(t, serverURL, sessionID)

	var stdout bytes.Buffer
	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { stdinR.Close(); stdinW.Close() })

	// Context cancels after 500ms — we just need scrollback to arrive.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run attachSession — it will receive the scrollback snapshot then context times out.
	_ = attachSession(ctx, conn, stdinR, &stdout, 0x1C)

	if !strings.Contains(stdout.String(), payload) {
		t.Errorf("stdout does not contain scrollback %q; got: %q", payload, stdout.String())
	}
}

// TestAttachSession_LiveOutput verifies that output written to the PTY after
// connecting is received by attachSession and written to stdout (CLI-05).
func TestAttachSession_LiveOutput(t *testing.T) {
	serverURL, _, ptyWrite, _, sessionID := setupAttachTest(t)
	conn := dialTestWS(t, serverURL, sessionID)

	var stdout bytes.Buffer
	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { stdinR.Close(); stdinW.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = attachSession(ctx, conn, stdinR, &stdout, 0x1C)
	}()

	// Brief delay to ensure attachSession goroutines are running.
	time.Sleep(20 * time.Millisecond)

	// Write live PTY output.
	const livePayload = "live data"
	if _, err := ptyWrite.Write([]byte(livePayload)); err != nil {
		t.Fatalf("ptyWrite: %v", err)
	}

	// Poll stdout until live data appears or timeout.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(stdout.String(), livePayload) {
			cancel() // clean stop
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	t.Errorf("stdout does not contain live output %q after 5s; got: %q", livePayload, stdout.String())
}

// TestAttachSession_CtrlCPassthrough verifies that Ctrl-C (0x03) is forwarded
// as a raw input byte to the PTY and NOT swallowed as a signal (CLI-06).
func TestAttachSession_CtrlCPassthrough(t *testing.T) {
	serverURL, _, _, inputRead, sessionID := setupAttachTest(t)
	conn := dialTestWS(t, serverURL, sessionID)

	var stdout bytes.Buffer
	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { stdinR.Close(); stdinW.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = attachSession(ctx, conn, stdinR, &stdout, 0x1C)
	}()

	// Brief delay to ensure stdinPump is running.
	time.Sleep(20 * time.Millisecond)

	// Write Ctrl-C byte to stdin pipe.
	if _, err := stdinW.Write([]byte{0x03}); err != nil {
		t.Fatalf("stdinW.Write Ctrl-C: %v", err)
	}

	// Read from the PTY input pipe — should receive the Ctrl-C byte.
	readDone := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 16)
		n, _ := inputRead.Read(buf)
		readDone <- buf[:n]
	}()

	select {
	case received := <-readDone:
		found := false
		for _, b := range received {
			if b == 0x03 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PTY input did not contain Ctrl-C (0x03); got bytes: %v", received)
		}
	case <-time.After(5 * time.Second):
		t.Error("PTY did not receive Ctrl-C within 5s")
	}
	cancel()
}

// TestAttachSession_InputForwarded verifies that keyboard input written to stdin
// is forwarded as raw bytes to the PTY stdin (CLI-06).
func TestAttachSession_InputForwarded(t *testing.T) {
	serverURL, _, _, inputRead, sessionID := setupAttachTest(t)
	conn := dialTestWS(t, serverURL, sessionID)

	var stdout bytes.Buffer
	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { stdinR.Close(); stdinW.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = attachSession(ctx, conn, stdinR, &stdout, 0x1C)
	}()

	// Brief delay to ensure stdinPump is running.
	time.Sleep(20 * time.Millisecond)

	// Write keyboard input.
	const input = "hello"
	if _, err := stdinW.Write([]byte(input)); err != nil {
		t.Fatalf("stdinW.Write: %v", err)
	}

	// Read from PTY input capture pipe.
	readDone := make(chan string, 1)
	go func() {
		buf := make([]byte, len(input)+16)
		n, _ := inputRead.Read(buf)
		readDone <- string(buf[:n])
	}()

	select {
	case received := <-readDone:
		if received != input {
			t.Errorf("PTY received %q, want %q", received, input)
		}
	case <-time.After(5 * time.Second):
		t.Error("PTY did not receive input within 5s")
	}
	cancel()
}
