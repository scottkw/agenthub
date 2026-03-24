package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agenthub/agenthub/internal/daemon"
	"github.com/agenthub/agenthub/internal/relay"
	"github.com/agenthub/agenthub/internal/webserver"
)

// testSetup creates a real daemon API on a short socket path and returns a DaemonClient.
func testSetup(t *testing.T) *daemon.DaemonClient {
	t.Helper()
	engine := daemon.NewSessionEngine()
	api := daemon.NewAPI(engine)
	// Use short socket path to avoid macOS 103-char sun_path limit.
	socketPath := fmt.Sprintf("/tmp/aht%d_%d.sock", os.Getpid(), time.Now().UnixNano()%10000)
	t.Cleanup(func() { os.Remove(socketPath) })
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("api.Start: %v", err)
	}
	t.Cleanup(func() { api.Stop() })
	// Brief sleep to let server goroutine start accepting.
	time.Sleep(10 * time.Millisecond)
	return daemon.NewDaemonClient(socketPath)
}

// TestCmdNew_Success creates a session and verifies a UUID is printed to stdout.
// Uses "cat" as the CLI since it is always present on macOS/Linux.
func TestCmdNew_Success(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdNew(client, []string{"cat", "/tmp"}, &buf)
	if err != nil {
		t.Fatalf("cmdNew returned error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	// Session IDs are 32-char lowercase hex strings (16 random bytes).
	if len(out) != 32 {
		t.Errorf("expected 32-char hex session ID, got %q (len=%d)", out, len(out))
	}
	for _, c := range out {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("session ID contains non-hex character %q: %q", c, out)
			break
		}
	}
}

// TestCmdNew_MissingArgs verifies that missing args returns an error with "usage".
func TestCmdNew_MissingArgs(t *testing.T) {
	client := testSetup(t)
	err := cmdNew(client, []string{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected error to contain %q, got %q", "usage", err.Error())
	}
}

// TestCmdList_Empty verifies that the header row is printed even when no sessions exist.
func TestCmdList_Empty(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdList(client, &buf)
	if err != nil {
		t.Fatalf("cmdList returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ID") {
		t.Errorf("expected header row with %q, got %q", "ID", out)
	}
}

// TestCmdList_WithSessions verifies that sessions appear in the list output.
func TestCmdList_WithSessions(t *testing.T) {
	client := testSetup(t)
	// Create a session via the client directly.
	id, err := client.CreateSession("cat","mytest", "/tmp")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	var buf bytes.Buffer
	if err := cmdList(client, &buf); err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, id) {
		t.Errorf("expected session ID %q in list output, got:\n%s", id, out)
	}
	if !strings.Contains(out, "mytest") {
		t.Errorf("expected session name %q in list output, got:\n%s", "mytest", out)
	}
}

// TestCmdKill_Success creates a session, kills it, and verifies it is removed.
func TestCmdKill_Success(t *testing.T) {
	client := testSetup(t)
	id, err := client.CreateSession("cat","killtest", "/tmp")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := cmdKill(client, []string{id}); err != nil {
		t.Fatalf("cmdKill: %v", err)
	}
	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after kill, got %d", len(sessions))
	}
}

// TestCmdRename_Success creates a session, renames it, and verifies the new name.
func TestCmdRename_Success(t *testing.T) {
	client := testSetup(t)
	id, err := client.CreateSession("cat","oldname", "/tmp")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := cmdRename(client, []string{id, "newname"}); err != nil {
		t.Fatalf("cmdRename: %v", err)
	}
	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "newname" {
		t.Errorf("expected name %q, got %q", "newname", sessions[0].Name)
	}
}

// TestCmdRename_MissingArgs verifies that missing args returns an error with "usage".
func TestCmdRename_MissingArgs(t *testing.T) {
	client := testSetup(t)
	err := cmdRename(client, []string{"onlyone"})
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected error to contain %q, got %q", "usage", err.Error())
	}
}

// testSetupWithWebServer creates a daemon with an injected running WebServer.
// Returns client and a cleanup-registered web server reference.
// Used for cmdServe/cmdUnserve tests that require a running web server.
func testSetupWithWebServer(t *testing.T) *daemon.DaemonClient {
	t.Helper()
	engine := daemon.NewSessionEngine()
	api := daemon.NewAPI(engine)
	socketPath := fmt.Sprintf("/tmp/aht%d_%d.sock", os.Getpid(), time.Now().UnixNano()%10000)
	t.Cleanup(func() { os.Remove(socketPath) })
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("api.Start: %v", err)
	}

	// Create and inject a test web server (no TLS, localhost, random port).
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: "127.0.0.1",
		Port:   0,
		FQDN:   "localhost",
	}, relay.NewHubManager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	ws.SetSessionResolver(func(id string) (string, string, string) {
		return id, "", ""
	})
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	api.SetWebServerForTest(ws)

	t.Cleanup(func() {
		_ = ws.Stop()
		api.Stop()
	})
	time.Sleep(10 * time.Millisecond)
	return daemon.NewDaemonClient(socketPath)
}

// TestCmdWebStatus_NotRunning verifies web status prints "running:" and "false" when not running.
func TestCmdWebStatus_NotRunning(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdWebStatus(client, &buf)
	if err != nil {
		t.Fatalf("cmdWebStatus returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "running:") {
		t.Errorf("expected output to contain %q, got %q", "running:", out)
	}
	if !strings.Contains(out, "false") {
		t.Errorf("expected output to contain %q, got %q", "false", out)
	}
}

// TestCmdWebStop verifies cmdWebStop returns no error (server may or may not be running).
func TestCmdWebStop(t *testing.T) {
	client := testSetup(t)
	// Stopping when not running should return no error (server returns 204).
	err := cmdWebStop(client)
	if err != nil {
		t.Errorf("cmdWebStop returned unexpected error: %v", err)
	}
}

// TestCmdServe_Success creates a session and enables web serving for it.
func TestCmdServe_Success(t *testing.T) {
	client := testSetupWithWebServer(t)
	id, err := client.CreateSession("cat", "servetest", "/tmp")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := cmdServe(client, []string{id}); err != nil {
		t.Fatalf("cmdServe returned error: %v", err)
	}
}

// TestCmdServe_MissingArgs verifies that missing args returns an error containing "usage".
func TestCmdServe_MissingArgs(t *testing.T) {
	client := testSetup(t)
	err := cmdServe(client, []string{})
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected error to contain %q, got %q", "usage", err.Error())
	}
}

// TestCmdUnserve_Success creates a session and disables web serving for it.
func TestCmdUnserve_Success(t *testing.T) {
	client := testSetupWithWebServer(t)
	id, err := client.CreateSession("cat", "unservetest", "/tmp")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := cmdUnserve(client, []string{id}); err != nil {
		t.Fatalf("cmdUnserve returned error: %v", err)
	}
}

// TestCmdHealth_OutputFormat verifies cmdHealth prints exactly 5 key-value lines.
func TestCmdHealth_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	err := cmdHealth(&buf)
	if err != nil {
		t.Fatalf("cmdHealth returned error: %v", err)
	}
	out := buf.String()
	// Verify all 5 labels are present.
	for _, label := range []string{"installed:", "connected:", "has-certs:", "ip:", "domain:"} {
		if !strings.Contains(out, label) {
			t.Errorf("expected output to contain %q, got:\n%s", label, out)
		}
	}
	// Count non-empty lines — should be exactly 5.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 output lines, got %d:\n%s", len(lines), out)
	}
}

// TestCmdQR_WebNotRunning verifies cmdQR returns an error when web server is not running.
func TestCmdQR_WebNotRunning(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdQR(client, []string{"some-id"}, &buf)
	if err == nil {
		t.Fatal("expected error when web server not running, got nil")
	}
	if !strings.Contains(err.Error(), "web server not running") {
		t.Errorf("expected error to contain %q, got %q", "web server not running", err.Error())
	}
}

// TestCmdQR_MissingArgs verifies that missing args returns an error containing "usage".
func TestCmdQR_MissingArgs(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdQR(client, []string{}, &buf)
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected error to contain %q, got %q", "usage", err.Error())
	}
}

// TestCmdWeb_MissingSubcommand verifies that missing subcommand returns an error with "usage: agenthub web".
func TestCmdWeb_MissingSubcommand(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdWeb(client, []string{}, &buf)
	if err == nil {
		t.Fatal("expected error for missing subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "usage: agenthub web") {
		t.Errorf("expected error to contain %q, got %q", "usage: agenthub web", err.Error())
	}
}
