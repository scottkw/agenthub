package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agenthub/agenthub/internal/daemon"
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
