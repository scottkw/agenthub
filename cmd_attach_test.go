package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/attach"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/statusbar"
	"github.com/scottkw/agenthub/internal/tailnet"
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

// TestMakeClientResizeFrame verifies that MakeClientResizeFrame encodes the
// resize frame using MsgResize2 (0x11) with big-endian cols and rows (CLI-06).
func TestMakeClientResizeFrame(t *testing.T) {
	// Basic case: cols=120, rows=40 -> [0x11, 0, 120, 0, 40]
	got := attach.MakeClientResizeFrame(120, 40)
	want := []byte{relay.MsgResize2, 0, 120, 0, 40}
	if !bytes.Equal(got, want) {
		t.Errorf("MakeClientResizeFrame(120, 40) = %v, want %v", got, want)
	}

	// Verify first byte is MsgResize2 (0x11), NOT MsgResize (0x02).
	if got[0] != relay.MsgResize2 {
		t.Errorf("first byte = 0x%02x, want MsgResize2 (0x%02x)", got[0], relay.MsgResize2)
	}
	if got[0] == relay.MsgResize {
		t.Errorf("first byte must NOT be MsgResize (0x%02x)", relay.MsgResize)
	}

	// Big-endian encoding: cols=256 -> [1, 0]; rows=512 -> [2, 0]
	got2 := attach.MakeClientResizeFrame(256, 512)
	want2 := []byte{relay.MsgResize2, 1, 0, 2, 0}
	if !bytes.Equal(got2, want2) {
		t.Errorf("MakeClientResizeFrame(256, 512) = %v, want %v", got2, want2)
	}
}

// TestAttachSession_DetachKey verifies that sending the detach key byte causes
// AttachSession to return nil (clean detach without error) (CLI-07).
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
		done <- attach.AttachSession(ctx, conn, stdinR, &stdout, 0x1C, nil, nil)
	}()

	// Write the detach key byte — this should cause a clean return.
	// Small delay to ensure AttachSession goroutines are running.
	time.Sleep(10 * time.Millisecond)
	if _, err := stdinW.Write([]byte{0x1C}); err != nil {
		t.Fatalf("stdinW.Write detach key: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("AttachSession returned error %v, want nil (clean detach)", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("AttachSession did not return within 5s after detach key")
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

	// Run AttachSession — it will receive the scrollback snapshot then context times out.
	_ = attach.AttachSession(ctx, conn, stdinR, &stdout, 0x1C, nil, nil)

	if !strings.Contains(stdout.String(), payload) {
		t.Errorf("stdout does not contain scrollback %q; got: %q", payload, stdout.String())
	}
}

// safeBuf is a bytes.Buffer protected by a mutex for concurrent test access.
type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// TestAttachSession_LiveOutput verifies that output written to the PTY after
// connecting is received by AttachSession and written to stdout (CLI-05).
func TestAttachSession_LiveOutput(t *testing.T) {
	serverURL, _, ptyWrite, _, sessionID := setupAttachTest(t)
	conn := dialTestWS(t, serverURL, sessionID)

	// Use a mutex-protected buffer to avoid the data race between
	// WsOutputPump (writes) and the polling loop (reads).
	var stdout safeBuf
	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { stdinR.Close(); stdinW.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = attach.AttachSession(ctx, conn, stdinR, &stdout, 0x1C, nil, nil)
	}()

	// Brief delay to ensure AttachSession goroutines are running.
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
		_ = attach.AttachSession(ctx, conn, stdinR, &stdout, 0x1C, nil, nil)
	}()

	// Brief delay to ensure StdinPump is running.
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

// TestPrintAttachBanner verifies that printAttachBanner writes the expected
// connection banner containing session name, CLI type, hostname, and detach hint (RMTE-02).
func TestPrintAttachBanner(t *testing.T) {
	var buf bytes.Buffer
	printAttachBanner(&buf, "my-session", "claude", "macbook-pro.local")
	output := buf.String()

	// Must contain session name
	if !strings.Contains(output, "my-session") {
		t.Errorf("banner missing session name; got: %s", output)
	}
	// Must contain CLI type
	if !strings.Contains(output, "claude") {
		t.Errorf("banner missing CLI type; got: %s", output)
	}
	// Must contain hostname
	if !strings.Contains(output, "macbook-pro.local") {
		t.Errorf("banner missing hostname; got: %s", output)
	}
	// Must contain detach key hint
	if !strings.Contains(output, `Ctrl-\`) {
		t.Errorf("banner missing detach key hint; got: %s", output)
	}
	// Must contain separator lines
	if !strings.Contains(output, "─────") {
		t.Errorf("banner missing separator line; got: %s", output)
	}
}

// TestPrintAttachBanner_EmptyName verifies that an empty session name
// is replaced by "unnamed" in the banner (RMTE-02).
func TestPrintAttachBanner_EmptyName(t *testing.T) {
	var buf bytes.Buffer
	printAttachBanner(&buf, "", "claude", "host.local")
	if !strings.Contains(buf.String(), "unnamed") {
		t.Errorf("empty name should show 'unnamed'; got: %s", buf.String())
	}
}

// TestPrintAttachBanner_NoOptionalFields verifies that the banner omits
// separator characters when CLI and hostname are empty (RMTE-02).
func TestPrintAttachBanner_NoOptionalFields(t *testing.T) {
	var buf bytes.Buffer
	printAttachBanner(&buf, "session-1", "", "")
	output := buf.String()
	if strings.Contains(output, "│") {
		t.Errorf("should not contain separator when CLI and hostname are empty; got: %s", output)
	}
	if !strings.Contains(output, "session-1") {
		t.Errorf("banner missing session name; got: %s", output)
	}
}

// TestCmdAttach_RemoteBannerShowsHostname verifies that printAttachBanner with
// a remote hostname displays the hostname in the banner output (REM-05).
func TestCmdAttach_RemoteBannerShowsHostname(t *testing.T) {
	var buf bytes.Buffer
	printAttachBanner(&buf, "remote-project", "claude", "macbook")
	output := buf.String()
	if !strings.Contains(output, "macbook") {
		t.Errorf("banner missing remote hostname; got: %s", output)
	}
	if !strings.Contains(output, "remote-project") {
		t.Errorf("banner missing session name; got: %s", output)
	}
	if !strings.Contains(output, "claude") {
		t.Errorf("banner missing CLI type; got: %s", output)
	}
}

// TestCmdAttach_RemoteSessionNotFound verifies the error message when a session
// does not exist on the remote peer (REM-05).
func TestCmdAttach_RemoteSessionNotFound(t *testing.T) {
	// Create an httptest TLS server that returns zero sessions.
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]struct{}{})
	}))
	defer ts.Close()

	err := cmdAttachRemoteWithClient(
		"macbook", "nonexistent-session",
		"macbook.ts.net", ts.URL, ts.Client(), nil, byte(0x1C), false,
	)
	if err == nil {
		t.Fatal("expected error for missing remote session, got nil")
	}
	if !strings.Contains(err.Error(), "not found on remote host") {
		t.Errorf("expected error containing %q, got %q", "not found on remote host", err.Error())
	}
	if !strings.Contains(err.Error(), "nonexistent-session") {
		t.Errorf("expected error containing session ID, got %q", err.Error())
	}
}

// TestCmdAttach_UnknownRemoteHost verifies the error message when the hostname
// doesn't match any tailnet peer (REM-05).
func TestCmdAttach_UnknownRemoteHost(t *testing.T) {
	peers := []tailnet.Peer{
		{Hostname: "macbook", DNSName: "macbook.ts.net.", Online: true},
		{Hostname: "desktop", DNSName: "desktop.ts.net.", Online: true},
	}
	err := buildUnknownHostError("laptop", peers)
	if err == nil {
		t.Fatal("expected error for unknown host, got nil")
	}
	if !strings.Contains(err.Error(), "unknown remote host") {
		t.Errorf("expected error containing %q, got %q", "unknown remote host", err.Error())
	}
	if !strings.Contains(err.Error(), "laptop") {
		t.Errorf("expected error containing hostname %q, got %q", "laptop", err.Error())
	}
	if !strings.Contains(err.Error(), "macbook") {
		t.Errorf("expected error containing available peer %q, got %q", "macbook", err.Error())
	}
	if !strings.Contains(err.Error(), "desktop") {
		t.Errorf("expected error containing available peer %q, got %q", "desktop", err.Error())
	}
}

// TestCmdAttach_UnknownRemoteHost_NoPeers verifies the error when no peers exist.
func TestCmdAttach_UnknownRemoteHost_NoPeers(t *testing.T) {
	err := buildUnknownHostError("laptop", nil)
	if err == nil {
		t.Fatal("expected error for unknown host, got nil")
	}
	if !strings.Contains(err.Error(), "no tailnet peers found") {
		t.Errorf("expected error containing %q, got %q", "no tailnet peers found", err.Error())
	}
}

// TestPrintDetachMessage verifies that printDetachMessage writes
// the "Detached." confirmation to the writer (RMTE-02).
func TestPrintDetachMessage(t *testing.T) {
	var buf bytes.Buffer
	printDetachMessage(&buf)
	if !strings.Contains(buf.String(), "Detached.") {
		t.Errorf("detach message missing; got: %s", buf.String())
	}
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
		_ = attach.AttachSession(ctx, conn, stdinR, &stdout, 0x1C, nil, nil)
	}()

	// Brief delay to ensure StdinPump is running.
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

// TestWsOutputPump_MsgMeta verifies that MsgMeta frames are parsed correctly
// and that bar.SetViewerCount can be called without panicking (SB-04).
func TestWsOutputPump_MsgMeta(t *testing.T) {
	var stdout safeBuf

	// Create a minimal bar to receive viewer count updates.
	bar := statusbar.New(&stdout, statusbar.Options{
		SessionName: "test",
		AgentType:   "claude",
		Hostname:    "host",
		CreatedAt:   time.Now(),
		Position:    statusbar.Bottom,
		Fd:          os.Stdout.Fd(),
	})
	// Don't call Start() -- we just need Set methods to work.

	// Send a MsgMeta frame with viewer count.
	count := 3
	frame := relay.MakeMeta(relay.MetaPayload{ViewerCount: &count})

	// Verify the MsgMeta parsing logic directly.
	msgType, payload, ferr := relay.ParseFrame(frame)
	if ferr != nil {
		t.Fatalf("ParseFrame error: %v", ferr)
	}
	if msgType != relay.MsgMeta {
		t.Errorf("expected MsgMeta (0x%02x), got 0x%02x", relay.MsgMeta, msgType)
	}
	var meta relay.MetaPayload
	if err := json.Unmarshal(payload, &meta); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if meta.ViewerCount == nil || *meta.ViewerCount != 3 {
		t.Errorf("expected viewerCount=3, got %v", meta.ViewerCount)
	}

	// Verify bar.SetViewerCount doesn't panic on a non-started bar.
	bar.SetViewerCount(*meta.ViewerCount)
}

// TestLockedWriter_ConcurrentWrites verifies that LockedWriter serializes
// concurrent writes without interleaving (prevents PTY/bar output corruption).
func TestLockedWriter_ConcurrentWrites(t *testing.T) {
	var buf safeBuf
	lw := attach.NewLockedWriter(&buf)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			msg := fmt.Sprintf("msg-%03d\n", n)
			_, _ = lw.Write([]byte(msg))
		}(i)
	}
	wg.Wait()

	// All 100 messages should be present and none should be interleaved.
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 100 {
		t.Errorf("expected 100 lines, got %d", len(lines))
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "msg-") || len(line) != 7 {
			t.Errorf("garbled line detected: %q", line)
		}
	}
}

// TestWsOutputPump_IgnoresUnknownFrameTypes verifies that frames with unknown
// type bytes are silently ignored (existing behavior preserved after MsgMeta addition).
func TestWsOutputPump_IgnoresUnknownFrameTypes(t *testing.T) {
	// Verify that frames with unknown types are silently ignored.
	// This is existing behavior but worth asserting since we added MsgMeta handling.
	unknownFrame := []byte{0xFF, 0x01, 0x02}
	msgType, _, err := relay.ParseFrame(unknownFrame)
	if err != nil {
		t.Fatalf("ParseFrame should not error on unknown type: %v", err)
	}
	if msgType == relay.MsgOutput || msgType == relay.MsgMeta {
		t.Errorf("unknown frame type 0xFF should not match known types")
	}
}
