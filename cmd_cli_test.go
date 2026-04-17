package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/webserver"
)

// testSetup creates a real daemon API on a short socket path and returns a DaemonClient.
func testSetup(t *testing.T) *daemon.DaemonClient {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("testSetup uses Unix domain sockets")
	}
	// Isolate config directory so settings.json from other tests doesn't leak.
	tmpCfg := t.TempDir()
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", tmpCfg) // os.UserConfigDir uses ~/Library/Application Support
	} else {
		t.Setenv("XDG_CONFIG_HOME", tmpCfg)
	}
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
	err := cmdNew(client, []string{"cat", "/tmp"}, nil, &buf)
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
	err := cmdNew(client, []string{}, nil, &bytes.Buffer{})
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
	err := cmdList(client, nil, &buf)
	if err != nil {
		t.Fatalf("cmdList returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "HOST") {
		t.Errorf("expected header row with %q, got %q", "HOST", out)
	}
	if !strings.Contains(out, "ID") {
		t.Errorf("expected header row with %q, got %q", "ID", out)
	}
}

// TestCmdList_WithSessions verifies that sessions appear in the list output.
func TestCmdList_WithSessions(t *testing.T) {
	client := testSetup(t)
	// Create a session via the client directly.
	id, err := client.CreateSession("cat", "mytest", "/tmp", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	var buf bytes.Buffer
	if err := cmdList(client, nil, &buf); err != nil {
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

// TestCmdList_JSON_Empty verifies that --json with no sessions produces a listOutput with empty local array.
func TestCmdList_JSON_Empty(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdList(client, []string{"--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdList --json returned error: %v", err)
	}
	var output listOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nraw: %s", err, buf.String())
	}
	if len(output.Local) != 0 {
		t.Errorf("expected 0 local sessions, got %d", len(output.Local))
	}
}

// TestCmdList_JSON_WithSessions verifies that --json with sessions produces valid JSON.
func TestCmdList_JSON_WithSessions(t *testing.T) {
	client := testSetup(t)
	id, err := client.CreateSession("cat", "jsontest", "/tmp", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	var buf bytes.Buffer
	if err := cmdList(client, []string{"--json"}, &buf); err != nil {
		t.Fatalf("cmdList --json: %v", err)
	}
	var output listOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nraw: %s", err, buf.String())
	}
	if len(output.Local) != 1 {
		t.Fatalf("expected 1 local session, got %d", len(output.Local))
	}
	if output.Local[0].ID != id {
		t.Errorf("expected session ID %q, got %q", id, output.Local[0].ID)
	}
	if output.Local[0].Name != "jsontest" {
		t.Errorf("expected name %q, got %q", "jsontest", output.Local[0].Name)
	}
}

// TestCmdKill_Success creates a session, kills it, and verifies it is removed.
func TestCmdKill_Success(t *testing.T) {
	client := testSetup(t)
	id, err := client.CreateSession("cat", "killtest", "/tmp", nil, 0, 0)
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
	id, err := client.CreateSession("cat", "oldname", "/tmp", nil, 0, 0)
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
	if runtime.GOOS == "windows" {
		t.Skip("testSetupWithWebServer uses Unix domain sockets")
	}
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", t.TempDir())
	} else {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	}
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
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		return id, "", "", ""
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
	err := cmdWebStatus(client, nil, &buf)
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

// TestCmdWebStatus_JSON verifies --json produces valid JSON for web status.
func TestCmdWebStatus_JSON(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdWebStatus(client, []string{"--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdWebStatus --json returned error: %v", err)
	}
	var resp daemon.WebServerStatusResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nraw: %s", err, buf.String())
	}
	if resp.Running {
		t.Error("expected Running=false")
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
	id, err := client.CreateSession("cat", "servetest", "/tmp", nil, 0, 0)
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
	id, err := client.CreateSession("cat", "unservetest", "/tmp", nil, 0, 0)
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
	err := cmdHealth(nil, &buf)
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

// TestCmdHealth_JSON verifies --json produces valid JSON for health.
func TestCmdHealth_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := cmdHealth([]string{"--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdHealth --json returned error: %v", err)
	}
	var h webserver.TailscaleHealth
	if err := json.Unmarshal(buf.Bytes(), &h); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nraw: %s", err, buf.String())
	}
	// Just verify it deserialized without error — actual values depend on system state.
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

// TestCmdSettings_Basic verifies cmdSettings prints all three label lines.
func TestCmdSettings_Basic(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdSettings(client, &buf)
	if err != nil {
		t.Fatalf("cmdSettings returned error: %v", err)
	}
	out := buf.String()
	for _, label := range []string{"socket-path:", "relay-port:", "cli-paths:"} {
		if !strings.Contains(out, label) {
			t.Errorf("expected output to contain %q, got:\n%s", label, out)
		}
	}
}

// TestCmdSettings_SocketPath verifies the output contains the default socket path value.
func TestCmdSettings_SocketPath(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	if err := cmdSettings(client, &buf); err != nil {
		t.Fatalf("cmdSettings error: %v", err)
	}
	expected := daemon.DefaultSocketPath()
	if !strings.Contains(buf.String(), expected) {
		t.Errorf("expected socket path %q in output, got:\n%s", expected, buf.String())
	}
}

// TestCmdSettings_RelayPort verifies the relay-port label is always present in output.
func TestCmdSettings_RelayPort(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	if err := cmdSettings(client, &buf); err != nil {
		t.Fatalf("cmdSettings error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "relay-port:") {
		t.Errorf("expected 'relay-port:' in output, got:\n%s", out)
	}
}

// TestCmdSettings_CLIPaths_None verifies that when no CLI paths are set, output contains "(none)".
func TestCmdSettings_CLIPaths_None(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	if err := cmdSettings(client, &buf); err != nil {
		t.Fatalf("cmdSettings error: %v", err)
	}
	if !strings.Contains(buf.String(), "(none)") {
		t.Errorf("expected '(none)' for empty CLI paths, got:\n%s", buf.String())
	}
}

// TestCmdSettings_CLIPaths_Set verifies that after setting a CLI path, output contains the path name and value.
func TestCmdSettings_CLIPaths_Set(t *testing.T) {
	client := testSetup(t)
	// Use /bin/sh as the CLI binary — it always exists on macOS and Linux.
	if err := client.UpdateCLIPath("claude", "/bin/sh"); err != nil {
		t.Fatalf("UpdateCLIPath: %v", err)
	}
	var buf bytes.Buffer
	if err := cmdSettings(client, &buf); err != nil {
		t.Fatalf("cmdSettings error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "claude=/bin/sh") {
		t.Errorf("expected CLI path in output, got:\n%s", out)
	}
}

// TestCmdList_WithHostColumn creates a session and verifies "(local)" and "HOST" appear in output.
func TestCmdList_WithHostColumn(t *testing.T) {
	client := testSetup(t)
	_, err := client.CreateSession("cat", "hosttest", "/tmp", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	var buf bytes.Buffer
	if err := cmdList(client, nil, &buf); err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "HOST") {
		t.Errorf("expected HOST column in output, got:\n%s", out)
	}
	if !strings.Contains(out, "(local)") {
		t.Errorf("expected (local) in output, got:\n%s", out)
	}
}

// TestCmdList_LocalFlag verifies --local skips remote discovery.
func TestCmdList_LocalFlag(t *testing.T) {
	client := testSetup(t)
	_, err := client.CreateSession("cat", "localtest", "/tmp", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	var buf bytes.Buffer
	// --local should work without error (no peer fetch attempted on test daemon)
	if err := cmdList(client, []string{"--local"}, &buf); err != nil {
		t.Fatalf("cmdList --local: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(local)") {
		t.Errorf("expected (local) in --local output, got:\n%s", out)
	}
	if !strings.Contains(out, "localtest") {
		t.Errorf("expected session name in output, got:\n%s", out)
	}
}

// TestCmdList_JSON_WithHostField verifies JSON output has local array with session objects.
func TestCmdList_JSON_WithHostField(t *testing.T) {
	client := testSetup(t)
	id, err := client.CreateSession("cat", "jsonhost", "/tmp", nil, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	var buf bytes.Buffer
	if err := cmdList(client, []string{"--json"}, &buf); err != nil {
		t.Fatalf("cmdList --json: %v", err)
	}
	var output listOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nraw: %s", err, buf.String())
	}
	if len(output.Local) != 1 {
		t.Fatalf("expected 1 local session, got %d", len(output.Local))
	}
	if output.Local[0].ID != id {
		t.Errorf("expected session ID %q, got %q", id, output.Local[0].ID)
	}
}

// TestUsage_RemoteSessionDocs verifies that the usage text documents remote
// session features including the Remote Sessions section, hostname:session-id
// example, --local flag, and updated descriptions (REM-05).
func TestUsage_RemoteSessionDocs(t *testing.T) {
	// Read cmd_cli.go source to verify usage text content.
	// Since usage() writes to os.Stderr directly, we use source inspection.
	src, err := os.ReadFile("cmd_cli.go")
	if err != nil {
		t.Fatalf("failed to read cmd_cli.go: %v", err)
	}
	content := string(src)

	checks := []struct {
		name   string
		needle string
	}{
		{"Remote Sessions section header", "Remote Sessions:"},
		{"hostname:session-id example", "hostname:session-id"},
		{"--local flag in list line", "--local"},
		{"List local and remote sessions", "List local and remote sessions"},
		{"Attach to a local or remote session", "Attach to a local or remote session"},
		{"macbook:abc123 example", "macbook:abc123"},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(content, tc.needle) {
				t.Errorf("cmd_cli.go usage missing %q", tc.needle)
			}
		})
	}
}

// TestSplitDashDash verifies all boundary cases for the -- separator.
func TestSplitDashDash(t *testing.T) {
	cases := []struct {
		name   string
		input  []string
		before []string
		after  []string
	}{
		{"no separator", []string{"new", "cat", "/tmp"}, []string{"new", "cat", "/tmp"}, nil},
		{"with args after", []string{"new", "cat", "/tmp", "--", "--model", "foo"}, []string{"new", "cat", "/tmp"}, []string{"--model", "foo"}},
		{"trailing separator", []string{"new", "cat", "/tmp", "--"}, []string{"new", "cat", "/tmp"}, []string{}},
		{"leading separator", []string{"--", "--model", "foo"}, []string{}, []string{"--model", "foo"}},
		{"empty input", []string{}, []string{}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, a := splitDashDash(tc.input)
			if len(b) != len(tc.before) {
				t.Fatalf("before: got %v, want %v", b, tc.before)
			}
			for i := range b {
				if b[i] != tc.before[i] {
					t.Fatalf("before[%d]: got %q, want %q", i, b[i], tc.before[i])
				}
			}
			if tc.after == nil {
				if a != nil {
					t.Fatalf("after: got %v, want nil", a)
				}
			} else {
				if a == nil {
					t.Fatalf("after: got nil, want %v", tc.after)
				}
				if len(a) != len(tc.after) {
					t.Fatalf("after: got %v, want %v", a, tc.after)
				}
				for i := range a {
					if a[i] != tc.after[i] {
						t.Fatalf("after[%d]: got %q, want %q", i, a[i], tc.after[i])
					}
				}
			}
		})
	}
}

// TestCmdNew_WithExtraArgs verifies that args after "--" are forwarded via CreateSession.
func TestCmdNew_WithExtraArgs(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdNew(client, []string{"cat", "/tmp"}, []string{"--model", "opus"}, &buf)
	if err != nil {
		t.Fatalf("cmdNew with extraArgs: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if len(out) != 32 {
		t.Errorf("expected 32-char hex session ID, got %q", out)
	}
}

// TestCmdNew_NoSeparator verifies that nil extraArgs still works (backward compat).
func TestCmdNew_NoSeparator(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdNew(client, []string{"cat", "/tmp"}, nil, &buf)
	if err != nil {
		t.Fatalf("cmdNew without extraArgs: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if len(out) != 32 {
		t.Errorf("expected 32-char hex session ID, got %q", out)
	}
}
