package daemon

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/tailnet"
	"github.com/scottkw/agenthub/internal/webserver"
)

// testDaemon creates an API server with a fresh engine on a temp socket.
// Returns the API, a DaemonClient connected to it, and the socket path.
func testDaemon(t *testing.T) (*API, *DaemonClient, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	engine := NewSessionEngine()
	// Isolate from real settings.json that NewSessionEngine may have loaded.
	engine.configDir = t.TempDir()
	engine.cliPaths = make(map[string]string)
	engine.startMinimized = false
	api := NewAPI(engine)
	// Use short socket path — macOS t.TempDir() paths exceed the 103-char limit.
	socketPath := shortSocketPath(t, "api.sock")
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("api.Start: %v", err)
	}
	t.Cleanup(func() { api.Stop() })
	client := NewDaemonClient(socketPath)
	// Brief sleep to let server goroutine start accepting.
	time.Sleep(10 * time.Millisecond)
	return api, client, socketPath
}

// dialUnix returns an http.Transport dialer that dials the given Unix socket.
func dialUnix(socketPath string) func(context.Context, string, string) (net.Conn, error) {
	return func(_ context.Context, _, _ string) (net.Conn, error) {
		return net.DialTimeout("unix", socketPath, 2*time.Second)
	}
}

// rawGet performs GET http://daemon{path} via Unix socket and returns status code + body bytes.
func rawGet(t *testing.T, socketPath, path string) (int, []byte) {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{DialContext: dialUnix(socketPath)}}
	resp, err := client.Get("http://daemon" + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

// rawPost performs POST http://daemon{path} with the given JSON body.
func rawPost(t *testing.T, socketPath, path, body string) (int, []byte) {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{DialContext: dialUnix(socketPath)}}
	resp, err := client.Post("http://daemon"+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// rawDelete performs DELETE http://daemon{path} via Unix socket.
func rawDelete(t *testing.T, socketPath, path string) (int, []byte) {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{DialContext: dialUnix(socketPath)}}
	req, err := http.NewRequest(http.MethodDelete, "http://daemon"+path, nil)
	if err != nil {
		t.Fatalf("NewRequest DELETE %s: %v", path, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// rawPatch performs PATCH http://daemon{path} with the given JSON body.
func rawPatch(t *testing.T, socketPath, path, body string) (int, []byte) {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{DialContext: dialUnix(socketPath)}}
	req, err := http.NewRequest(http.MethodPatch, "http://daemon"+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest PATCH %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PATCH %s: %v", path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

func TestAPIHealth(t *testing.T) {
	old := BuildVersion
	BuildVersion = "v2.1.2-test"
	t.Cleanup(func() { BuildVersion = old })

	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/health")
	if status != 200 {
		t.Errorf("GET /health: want 200, got %d", status)
	}
	var h HealthResponse
	if err := json.Unmarshal(body, &h); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if h.Status != "ok" {
		t.Errorf("health status: want %q, got %q", "ok", h.Status)
	}
	if h.Version != "v2.1.2-test" {
		t.Errorf("health version: want %q, got %q", "v2.1.2-test", h.Version)
	}
}

func TestAPIListSessionsEmpty(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/sessions")
	if status != 200 {
		t.Errorf("GET /sessions: want 200, got %d", status)
	}
	var sessions []SessionInfo
	if err := json.Unmarshal(body, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("want 0 sessions, got %d", len(sessions))
	}
}

func TestAPICreateSession(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"test","workDir":""}`)
	if status != 201 {
		t.Errorf("POST /sessions: want 201, got %d; body: %s", status, body)
	}
	var resp CreateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if resp.ID == "" {
		t.Error("create response: empty ID")
	}
}

func TestAPIGetSession(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	_, createBody := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"my-tab","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(createBody, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	status, body := rawGet(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID))
	if status != 200 {
		t.Errorf("GET /sessions/%s: want 200, got %d", cr.ID, status)
	}
	var info SessionInfo
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatalf("decode session info: %v", err)
	}
	if info.ID != cr.ID {
		t.Errorf("session ID mismatch: got %q, want %q", info.ID, cr.ID)
	}
	if info.Name != "my-tab" {
		t.Errorf("session name: got %q, want %q", info.Name, "my-tab")
	}
}

func TestAPIGetSessionNotFound(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, _ := rawGet(t, socketPath, "/sessions/nonexistent-id")
	if status != 404 {
		t.Errorf("GET /sessions/nonexistent-id: want 404, got %d", status)
	}
}

func TestAPIDeleteSession(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	_, createBody := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"del-me","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(createBody, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	status, _ := rawDelete(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID))
	if status != 204 {
		t.Errorf("DELETE /sessions/%s: want 204, got %d", cr.ID, status)
	}

	// Subsequent GET should return 404.
	status, _ = rawGet(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID))
	if status != 404 {
		t.Errorf("GET after DELETE: want 404, got %d", status)
	}
}

func TestAPIRenameSession(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	_, createBody := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"old-name","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(createBody, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	t.Cleanup(func() { rawDelete(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID)) })

	status, _ := rawPatch(t, socketPath, fmt.Sprintf("/sessions/%s/name", cr.ID), `{"name":"new-name"}`)
	if status != 204 {
		t.Errorf("PATCH /sessions/%s/name: want 204, got %d", cr.ID, status)
	}

	// Subsequent GET shows new name.
	_, body := rawGet(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID))
	var info SessionInfo
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatalf("decode session info after rename: %v", err)
	}
	if info.Name != "new-name" {
		t.Errorf("name after rename: got %q, want %q", info.Name, "new-name")
	}
}

func TestAPIGetSessionStatus(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	_, createBody := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"st-tab","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(createBody, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	t.Cleanup(func() { rawDelete(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID)) })

	status, body := rawGet(t, socketPath, fmt.Sprintf("/sessions/%s/status", cr.ID))
	if status != 200 {
		t.Errorf("GET /sessions/%s/status: want 200, got %d", cr.ID, status)
	}
	var sr StatusResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	valid := map[string]bool{"running": true, "waiting": true, "idle": true, "errored": true}
	if !valid[sr.Status] {
		t.Errorf("invalid status %q", sr.Status)
	}
}

func TestAPIGetCLIPaths(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/settings/cli-paths")
	if status != 200 {
		t.Errorf("GET /settings/cli-paths: want 200, got %d", status)
	}
	var paths map[string]string
	if err := json.Unmarshal(body, &paths); err != nil {
		t.Fatalf("decode cli paths: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty map, got %v", paths)
	}
}

func TestAPIUpdateCLIPath(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, _ := rawPatch(t, socketPath, "/settings/cli-paths/claude", `{"path":"/bin/cat"}`)
	if status != 204 {
		t.Errorf("PATCH /settings/cli-paths/claude: want 204, got %d", status)
	}

	// Subsequent GET shows the entry.
	_, body := rawGet(t, socketPath, "/settings/cli-paths")
	var paths map[string]string
	if err := json.Unmarshal(body, &paths); err != nil {
		t.Fatalf("decode cli paths: %v", err)
	}
	if paths["claude"] != "/bin/cat" {
		t.Errorf("cli path for 'claude': got %q, want %q", paths["claude"], "/bin/cat")
	}
}

func TestAPIRelayPort(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	port, err := api.StartRelay()
	if err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	if port <= 0 {
		t.Fatalf("StartRelay returned invalid port: %d", port)
	}

	status, body := rawGet(t, socketPath, "/relay-port")
	if status != 200 {
		t.Errorf("GET /relay-port: want 200, got %d", status)
	}
	var resp RelayPortResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode relay port response: %v", err)
	}
	if resp.Port <= 0 {
		t.Errorf("relay port: want > 0, got %d", resp.Port)
	}
	if resp.Port != port {
		t.Errorf("relay port mismatch: got %d, want %d", resp.Port, port)
	}
}

func TestAPIWebServerStatus_NotRunning(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/webserver/status")
	if status != 200 {
		t.Errorf("GET /webserver/status: want 200, got %d", status)
	}
	var resp WebServerStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode webserver status response: %v", err)
	}
	if resp.Running {
		t.Errorf("webserver status: want running=false, got true")
	}
}

func TestAPIWebServe_NoServer(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, _ := rawPost(t, socketPath, "/sessions/xxx/web-serve", `{"enabled":true}`)
	if status != 400 {
		t.Errorf("POST /sessions/xxx/web-serve with no server: want 400, got %d", status)
	}
}

func TestAPICreateSessionWithArgs(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawPost(t, socketPath, "/sessions",
		`{"cli":"cat","name":"args-test","workDir":"","args":["--flag","value"]}`)
	if status != 201 {
		t.Errorf("POST /sessions with args: want 201, got %d; body: %s", status, body)
	}
	var resp CreateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if resp.ID == "" {
		t.Error("create response with args: empty ID")
	}
}

func TestAPIListSessionsHostname(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"h-test","workDir":""}`)

	_, body := rawGet(t, socketPath, "/sessions")
	var sessions []SessionInfo
	if err := json.Unmarshal(body, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected at least 1 session")
	}
	if sessions[0].Hostname == "" {
		t.Error("SessionInfo.Hostname is empty — want non-empty hostname")
	}
}

func TestClientCreateSessionWithArgs(t *testing.T) {
	_, client, _ := testDaemon(t)
	id, err := client.CreateSession("cat", "client-args-test", "", []string{"--extra", "arg"}, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession with args: %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession with args returned empty ID")
	}
}

func TestAutoStartWebServer_AlreadyRunning(t *testing.T) {
	api, _, _ := testDaemon(t)
	// Inject a fake web server so a.webServer != nil.
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: "127.0.0.1",
		Port:   0,
		FQDN:   "test.local",
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer for test: %v", err)
	}
	api.SetWebServerForTest(ws)
	// AutoStartWebServer should no-op when already running.
	err = api.AutoStartWebServer("100.64.0.1", 7443, "test.ts.net", "tailscale", "")
	if err != nil {
		t.Errorf("AutoStartWebServer with existing server: want nil, got %v", err)
	}
}

// TestHandleCreateSession_NoAutoEnable (SEC-01 behavioral test):
// Creating a session while the web server is running MUST NOT auto-enable web
// serving for it. The user must explicitly toggle web-serving ON to issue
// capabilities and expose the session (D-06 grant gesture).
func TestHandleCreateSession_NoAutoEnable(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	// Create a WebServer (without Start — no TLS needed for EnableSession check).
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: "127.0.0.1",
		Port:   0,
		FQDN:   "test.local",
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer for test: %v", err)
	}
	api.SetWebServerForTest(ws)

	// Create a session while the web server is running.
	status, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"no-auto","workDir":""}`)
	if status != 201 {
		t.Fatalf("POST /sessions: want 201, got %d; body: %s", status, body)
	}
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	// SEC-01: the session MUST NOT be web-enabled automatically.
	if ws.IsSessionEnabled(cr.ID) {
		t.Errorf("SEC-01 violation: session %s is web-enabled immediately after creation — auto-enable block was not removed", cr.ID)
	}

	// Also verify via the /sessions API that WebEnabled is false.
	_, listBody := rawGet(t, socketPath, "/sessions")
	var sessions []SessionInfo
	if err := json.Unmarshal(listBody, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	for _, s := range sessions {
		if s.ID == cr.ID && s.WebEnabled {
			t.Errorf("SEC-01: session %s reports WebEnabled=true, want false (auto-enable must be removed)", cr.ID)
		}
	}
}

func TestCreateSession_NoAutoEnable(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	// No web server set — create a session.
	status, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"no-web","workDir":""}`)
	if status != 201 {
		t.Fatalf("POST /sessions: want 201, got %d; body: %s", status, body)
	}
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	// GET /sessions should show WebEnabled=false.
	_, listBody := rawGet(t, socketPath, "/sessions")
	var sessions []SessionInfo
	if err := json.Unmarshal(listBody, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	for _, s := range sessions {
		if s.ID == cr.ID && s.WebEnabled {
			t.Errorf("session %s: want WebEnabled=false, got true", cr.ID)
		}
	}
}

func TestGetLocalPassword(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	api.SetLocalPassword("abc123")

	status, body := rawGet(t, socketPath, "/webserver/local-password")
	if status != 200 {
		t.Errorf("GET /webserver/local-password: want 200, got %d", status)
	}
	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode local-password response: %v", err)
	}
	if resp["password"] != "abc123" {
		t.Errorf("local password: want %q, got %q", "abc123", resp["password"])
	}
}

func TestGetLocalPassword_TailscaleMode(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	// No SetLocalPassword called — Tailscale mode, password should be empty.

	status, body := rawGet(t, socketPath, "/webserver/local-password")
	if status != 200 {
		t.Errorf("GET /webserver/local-password (tailscale mode): want 200, got %d", status)
	}
	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode local-password response: %v", err)
	}
	if resp["password"] != "" {
		t.Errorf("local password in tailscale mode: want empty, got %q", resp["password"])
	}
}

func TestHandleTailnetPeers(t *testing.T) {
	t.Run("returns cached peers as JSON array", func(t *testing.T) {
		api, _, socketPath := testDaemon(t)

		testPeers := []tailnet.Peer{
			{Hostname: "mac1", DNSName: "mac1.tail.ts.net.", TailscaleIPs: []string{"100.64.0.1"}, OS: "macOS", Online: true},
			{Hostname: "linux2", DNSName: "linux2.tail.ts.net.", TailscaleIPs: []string{"100.64.0.2"}, OS: "linux", Online: true},
		}
		// Pre-populate cache to avoid needing a live Tailscale daemon.
		api.tailnetCache.set(testPeers)

		status, body := rawGet(t, socketPath, "/tailnet/peers")
		if status != 200 {
			t.Errorf("GET /tailnet/peers: want 200, got %d; body: %s", status, body)
		}

		var peers []tailnet.Peer
		if err := json.Unmarshal(body, &peers); err != nil {
			t.Fatalf("decode peers response: %v", err)
		}
		if len(peers) != 2 {
			t.Errorf("want 2 peers, got %d", len(peers))
		}
		if peers[0].Hostname != "mac1" {
			t.Errorf("first peer hostname: want %q, got %q", "mac1", peers[0].Hostname)
		}
	})

	t.Run("returns empty array when no Tailscale daemon available", func(t *testing.T) {
		api, _, socketPath := testDaemon(t)

		// Pre-set cache with empty slice to simulate unavailable Tailscale.
		api.tailnetCache.set([]tailnet.Peer{})

		status, body := rawGet(t, socketPath, "/tailnet/peers")
		if status != 200 {
			t.Errorf("GET /tailnet/peers empty: want 200, got %d; body: %s", status, body)
		}

		var peers []tailnet.Peer
		if err := json.Unmarshal(body, &peers); err != nil {
			t.Fatalf("decode empty peers response: %v", err)
		}
		if len(peers) != 0 {
			t.Errorf("want 0 peers, got %d", len(peers))
		}
	})

	t.Run("client ListTailnetPeers returns cached peers", func(t *testing.T) {
		api, client, _ := testDaemon(t)

		testPeers := []tailnet.Peer{
			{Hostname: "node1", DNSName: "node1.tail.ts.net.", TailscaleIPs: []string{"100.64.0.3"}, OS: "linux", Online: true},
		}
		api.tailnetCache.set(testPeers)

		peers, err := client.ListTailnetPeers()
		if err != nil {
			t.Fatalf("ListTailnetPeers: %v", err)
		}
		if len(peers) != 1 {
			t.Errorf("want 1 peer, got %d", len(peers))
		}
		if peers[0].Hostname != "node1" {
			t.Errorf("peer hostname: want %q, got %q", "node1", peers[0].Hostname)
		}
	})
}

// TestAutoStartWebServer_CreatesNewServer verifies the positive path of
// AutoStartWebServer: when a.webServer is nil, a successful call sets it to
// non-nil and GET /webserver/status subsequently returns running=true (SERVE-01).
func TestAutoStartWebServer_CreatesNewServer(t *testing.T) {
	api, _, socketPath := testDaemon(t)

	// Precondition: no web server running yet.
	statusBefore, bodyBefore := rawGet(t, socketPath, "/webserver/status")
	if statusBefore != 200 {
		t.Fatalf("GET /webserver/status: want 200, got %d", statusBefore)
	}
	var respBefore WebServerStatusResponse
	if err := json.Unmarshal(bodyBefore, &respBefore); err != nil {
		t.Fatalf("decode status before: %v", err)
	}
	if respBefore.Running {
		t.Fatal("precondition failed: web server already running before AutoStartWebServer call")
	}

	// Call AutoStartWebServer with local mode on loopback — no Tailscale certs needed.
	err := api.AutoStartWebServer("127.0.0.1", 0, "", "local", "testpassword")
	if err != nil {
		t.Fatalf("AutoStartWebServer: unexpected error: %v", err)
	}

	// Postcondition: GET /webserver/status must report running=true.
	statusAfter, bodyAfter := rawGet(t, socketPath, "/webserver/status")
	if statusAfter != 200 {
		t.Fatalf("GET /webserver/status after start: want 200, got %d", statusAfter)
	}
	var respAfter WebServerStatusResponse
	if err := json.Unmarshal(bodyAfter, &respAfter); err != nil {
		t.Fatalf("decode status after: %v", err)
	}
	if !respAfter.Running {
		t.Errorf("AutoStartWebServer: want web server running=true, got false")
	}
}

// TestAutoStartWebServer_LocalModeRequiresPassword verifies that calling
// AutoStartWebServer with mode="local" and an empty password returns an error
// instead of starting a server without authentication (SERVE-01 guard).
func TestAutoStartWebServer_LocalModeRequiresPassword(t *testing.T) {
	api, _, _ := testDaemon(t)

	err := api.AutoStartWebServer("127.0.0.1", 0, "", "local", "")
	if err == nil {
		t.Error("AutoStartWebServer(local, empty password): want non-nil error, got nil")
	}
}

func TestHandleNotifyThemeChange(t *testing.T) {
	_, _, socketPath := testDaemon(t)

	// POST /theme/notify with no active sessions should return 204.
	status, _ := rawPost(t, socketPath, "/theme/notify", "")
	if status != 204 {
		t.Errorf("POST /theme/notify (empty engine): want 204, got %d", status)
	}
}

func TestHandleNotifyThemeChange_WithSessions(t *testing.T) {
	_, _, socketPath := testDaemon(t)

	// Create a session (uses real PTY via cat).
	createStatus, createBody := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"notify-test","workDir":""}`)
	if createStatus != 201 {
		t.Fatalf("POST /sessions: want 201, got %d; body: %s", createStatus, createBody)
	}

	// POST /theme/notify should return 204 even with active sessions.
	// cat is not opencode, so no signal is sent — but the route should succeed.
	status, _ := rawPost(t, socketPath, "/theme/notify", "")
	if status != 204 {
		t.Errorf("POST /theme/notify (with cat session): want 204, got %d", status)
	}
}

func TestClientNotifyThemeChange(t *testing.T) {
	_, client, _ := testDaemon(t)
	err := client.NotifyThemeChange()
	if err != nil {
		t.Errorf("client.NotifyThemeChange: want nil, got %v", err)
	}
}

// TestAPI_ListSessionsViewerCount verifies that ViewerCount in the session list
// API response reflects the actual hub subscriber count (MC-04).
func TestAPI_ListSessionsViewerCount(t *testing.T) {
	api, _, socketPath := testDaemon(t)

	// Create a session via the API.
	status, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"vc-test","workDir":""}`)
	if status != 201 {
		t.Fatalf("POST /sessions: want 201, got %d; body: %s", status, body)
	}
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	t.Cleanup(func() { rawDelete(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID)) })

	// Capture the baseline ViewerCount. CreateSession starts a status.Watch
	// goroutine that subscribes to the hub, so the baseline is typically 1
	// (the status detector). We measure the delta rather than absolute values.
	_, listBody := rawGet(t, socketPath, "/sessions")
	var sessions []SessionInfo
	if err := json.Unmarshal(listBody, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	var found *SessionInfo
	for i := range sessions {
		if sessions[i].ID == cr.ID {
			found = &sessions[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("session %s not found in list", cr.ID)
	}
	baseline := found.ViewerCount

	// Subscribe a client to the session's hub.
	hub, ok := api.engine.Manager().Get(cr.ID)
	if !ok {
		t.Fatalf("hub for session %s not found in manager", cr.ID)
	}
	sub := &relay.Subscriber{
		Msgs:      make(chan []byte, 256),
		CloseSlow: func() {},
	}
	hub.Subscribe(sub)
	defer hub.Unsubscribe(sub)

	// Verify ViewerCount increased by 1 with our subscriber.
	_, listBody2 := rawGet(t, socketPath, "/sessions")
	var sessions2 []SessionInfo
	if err := json.Unmarshal(listBody2, &sessions2); err != nil {
		t.Fatalf("decode sessions after subscribe: %v", err)
	}
	var found2 *SessionInfo
	for i := range sessions2 {
		if sessions2[i].ID == cr.ID {
			found2 = &sessions2[i]
			break
		}
	}
	if found2 == nil {
		t.Fatalf("session %s not found in list after subscribe", cr.ID)
	}
	if found2.ViewerCount != baseline+1 {
		t.Errorf("ViewerCount with 1 extra subscriber: want %d, got %d", baseline+1, found2.ViewerCount)
	}

	// Unsubscribe and verify ViewerCount returns to baseline.
	hub.Unsubscribe(sub)
	_, listBody3 := rawGet(t, socketPath, "/sessions")
	var sessions3 []SessionInfo
	if err := json.Unmarshal(listBody3, &sessions3); err != nil {
		t.Fatalf("decode sessions after unsubscribe: %v", err)
	}
	var found3 *SessionInfo
	for i := range sessions3 {
		if sessions3[i].ID == cr.ID {
			found3 = &sessions3[i]
			break
		}
	}
	if found3 == nil {
		t.Fatalf("session %s not found in list after unsubscribe", cr.ID)
	}
	if found3.ViewerCount != baseline {
		t.Errorf("ViewerCount after unsubscribe: want %d, got %d", baseline, found3.ViewerCount)
	}
}

// TestAPIGetStartMinimized verifies GET /settings/start-minimized returns 200
// with {"startMinimized": false} when no value has been set (TRAY-02).
func TestAPIGetStartMinimized(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/settings/start-minimized")
	if status != 200 {
		t.Errorf("GET /settings/start-minimized: want 200, got %d; body: %s", status, body)
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode start-minimized response: %v", err)
	}
	if val, ok := resp["startMinimized"]; !ok {
		t.Error("response missing 'startMinimized' key")
	} else if val {
		t.Errorf("initial startMinimized: want false, got true")
	}
}

// TestAPISetStartMinimized verifies PATCH /settings/start-minimized returns 204
// and the subsequent GET reflects the updated value (TRAY-02 / TRAY-03).
func TestAPISetStartMinimized(t *testing.T) {
	_, _, socketPath := testDaemon(t)

	// PATCH to true — expect 204.
	patchStatus, patchBody := rawPatch(t, socketPath, "/settings/start-minimized", `{"startMinimized":true}`)
	if patchStatus != 204 {
		t.Errorf("PATCH /settings/start-minimized: want 204, got %d; body: %s", patchStatus, patchBody)
	}

	// GET — expect true.
	getStatus, getBody := rawGet(t, socketPath, "/settings/start-minimized")
	if getStatus != 200 {
		t.Errorf("GET /settings/start-minimized after PATCH: want 200, got %d", getStatus)
	}
	var resp map[string]bool
	if err := json.Unmarshal(getBody, &resp); err != nil {
		t.Fatalf("decode start-minimized response after PATCH: %v", err)
	}
	if !resp["startMinimized"] {
		t.Errorf("startMinimized after PATCH true: want true, got false")
	}
}

// TestAPISetStartMinimizedInvalidBody verifies PATCH /settings/start-minimized
// with invalid JSON returns 400 (TRAY-02 input validation).
func TestAPISetStartMinimizedInvalidBody(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, _ := rawPatch(t, socketPath, "/settings/start-minimized", `not-json`)
	if status != 400 {
		t.Errorf("PATCH /settings/start-minimized (invalid body): want 400, got %d", status)
	}
}

// --- Phase 87 Plan 04 tests ---------------------------------------------

// configureCapabilityStateForTest wires a synthetic signing key and join-code
// manager onto both the API and its WebServer so capability flows work in
// tests without needing a real daemon startup sequence.
func configureCapabilityStateForTest(t *testing.T, api *API, ws *webserver.WebServer) []byte {
	t.Helper()
	key, err := capability.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	api.signingKeyMu.Lock()
	api.signingKey = key
	api.signingKeyMu.Unlock()
	jc := capability.NewJoinCodeManager(5 * time.Minute)
	api.joinCodes = jc
	ws.SetSigningKey(key)
	ws.SetJoinCodes(jc)
	return key
}

// probeGrant mints a fresh capability token for (sessionID, grantID) using
// the WS's current signing key, then performs a live HTTPS GET against
// /api/sessions/{id}/info. Returns true iff the request succeeds with 200
// (i.e. the grant is still active AND the session is web-enabled). Any
// status other than 200 — including 401 (nil key / missing cap), 403
// (revoked / mismatched), or 404 (not enabled) — returns false.
//
// Uses an in-process httptest-like flow by sending the request through
// ws.BaseURL(), which is only valid after ws.Start() has assigned a
// listener. Tests that call probeGrant therefore construct a live TLS
// listener via newLoopbackTLSListener.
func probeGrant(t *testing.T, ws *webserver.WebServer, key []byte, sessionID, grantID string) bool {
	t.Helper()
	claims := capability.Claims{
		SID:     sessionID,
		Perms:   "read,write",
		IAT:     time.Now().Unix(),
		GrantID: grantID,
		V:       1,
	}
	tok, err := capability.Sign(claims, key)
	if err != nil {
		t.Fatalf("probeGrant: Sign: %v", err)
	}
	base := ws.BaseURL()
	if base == "" {
		t.Fatalf("probeGrant: ws.BaseURL is empty; start the server first")
	}
	url := base + "/api/sessions/" + sessionID + "/info?cap=" + tok
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// TestHandleWebServe_ToggleOnEnablesSession: toggle web-serving ON for a
// session, assert status 204 and that the session becomes web-enabled. The
// authoritative capability issuance path is the separate POST
// /sessions/{id}/capabilities endpoint (task 87-04-02).
func TestHandleWebServe_ToggleOnEnablesSession(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: "127.0.0.1",
		Port:   0,
		FQDN:   "test.local",
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer for test: %v", err)
	}
	api.SetWebServerForTest(ws)
	configureCapabilityStateForTest(t, api, ws)

	// Create a session (no auto-enable per SEC-01).
	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"toggle-on","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	if ws.IsSessionEnabled(cr.ID) {
		t.Fatalf("precondition failed: session %s already web-enabled before toggle", cr.ID)
	}

	// Toggle web-serving ON.
	status, respBody := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID),
		`{"enabled":true}`)
	if status != http.StatusNoContent {
		t.Errorf("POST /sessions/%s/web-serve {enabled:true}: want 204, got %d; body: %s", cr.ID, status, respBody)
	}

	if !ws.IsSessionEnabled(cr.ID) {
		t.Errorf("after toggle-on: session %s IsSessionEnabled=false, want true", cr.ID)
	}
}

// TestHandleWebServe_ToggleOffClearsGrants: toggle ON, register a probe
// grant, assert the grant is active via an HTTPS round-trip, toggle OFF,
// assert the grant is no longer active. This enforces D-15 (toggle-off
// permanently clears grants).
func TestHandleWebServe_ToggleOffClearsGrants(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	lnCfg := newLoopbackTLSListener(t)
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: lnCfg,
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer for test: %v", err)
	}
	ws.SetSessionResolver(func(sid string) (string, string, string, string) {
		return "probe", "cat", "running", "localhost"
	})
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })
	api.SetWebServerForTest(ws)
	key := configureCapabilityStateForTest(t, api, ws)

	// Create + toggle ON.
	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"toggle-off","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	status, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`)
	if status != http.StatusNoContent {
		t.Fatalf("toggle-on: want 204, got %d", status)
	}

	// Register a probe grant directly so we can observe its clearance.
	probe := "probe-grant-id-12345"
	ws.AddGrant(cr.ID, probe)

	// Sanity: probe grant is active before toggle-off.
	if !probeGrant(t, ws, key, cr.ID, probe) {
		t.Fatalf("probe grant not active before toggle-off")
	}

	// Toggle OFF.
	status, _ = rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":false}`)
	if status != http.StatusNoContent {
		t.Fatalf("toggle-off: want 204, got %d", status)
	}

	// D-15: after toggle-off, ALL grants for this session must be cleared.
	if probeGrant(t, ws, key, cr.ID, probe) {
		t.Errorf("D-15 violation: probe grant still active after toggle-off")
	}
	// The session must also be disabled.
	if ws.IsSessionEnabled(cr.ID) {
		t.Errorf("after toggle-off: session %s still enabled, want disabled", cr.ID)
	}
}

// TestOnExit_ClearsGrants (RESEARCH Pitfall 1): when a session's onExit
// callback fires, ws.ClearGrants MUST be called alongside ws.DisableSession.
// Without this, grants leak beyond session lifetime.
//
// Uses the test-only onExit builder on *API which skips the 10-second grace
// period timer and invokes the cleanup synchronously. The real code path
// still uses time.AfterFunc(10*time.Second, ...), which is exercised
// indirectly by TestHandleWebServe_ToggleOffClearsGrants (same underlying
// ClearGrants call, no timer).
func TestOnExit_ClearsGrants(t *testing.T) {
	api, _, _ := testDaemon(t)
	lnCfg := newLoopbackTLSListener(t)
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: lnCfg,
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer for test: %v", err)
	}
	ws.SetSessionResolver(func(sid string) (string, string, string, string) {
		return "exit-test", "cat", "running", "localhost"
	})
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })
	api.SetWebServerForTest(ws)
	key := configureCapabilityStateForTest(t, api, ws)

	// Register a session + grant, then invoke the onExit cleanup directly.
	sessionID := "exit-test-session"
	ws.EnableSession(sessionID)
	ws.AddGrant(sessionID, "grant-to-clear")

	if !probeGrant(t, ws, key, sessionID, "grant-to-clear") {
		t.Fatalf("precondition: grant not active before onExit")
	}

	// Invoke the synchronous test-only onExit cleanup (bypasses the 10-sec
	// time.AfterFunc grace). This exercises the exact same ClearGrants +
	// DisableSession pair used by the production onExit closure.
	api.runSessionExitCleanupForTest(sessionID)

	if ws.IsSessionEnabled(sessionID) {
		t.Errorf("onExit did not DisableSession: session still enabled")
	}
	// D-15 / Pitfall 1: grants for this session must be cleared.
	if probeGrant(t, ws, key, sessionID, "grant-to-clear") {
		t.Errorf("Pitfall 1: onExit did not call ClearGrants — probe grant still active")
	}
}

// TestStartup_LoadsOrGeneratesSigningKey: starting the API with an empty
// config directory generates capability.key on disk, and sets api.signingKey
// (and ws.signingKey) to non-nil.
func TestStartup_LoadsOrGeneratesSigningKey(t *testing.T) {
	// Use an isolated temp config directory so this test doesn't clobber
	// the real ~/.config/agenthub/capability.key.
	tmpDir := t.TempDir()
	engine := NewSessionEngine()
	engine.configDir = tmpDir
	engine.cliPaths = make(map[string]string)
	engine.startMinimized = false
	api := NewAPI(engine)

	// Bootstrap the capability state (this is the production startup call
	// that task 87-04-01 introduces on *API).
	if err := api.BootstrapCapabilityState(); err != nil {
		t.Fatalf("BootstrapCapabilityState: %v", err)
	}

	// capability.key must now exist in the config dir.
	keyPath := filepath.Join(tmpDir, "capability.key")
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("capability.key not found at %s: %v", keyPath, err)
	}
	if info.Size() != 32 {
		t.Errorf("capability.key size: got %d, want 32", info.Size())
	}

	// api.signingKey must be non-nil.
	api.signingKeyMu.RLock()
	key := api.signingKey
	api.signingKeyMu.RUnlock()
	if key == nil {
		t.Fatalf("api.signingKey is nil after BootstrapCapabilityState")
	}
	if len(key) != 32 {
		t.Errorf("signing key length: got %d, want 32", len(key))
	}

	// api.joinCodes must be non-nil.
	if api.joinCodes == nil {
		t.Errorf("api.joinCodes is nil after BootstrapCapabilityState")
	}

	// A subsequent call must reload the same key (not regenerate).
	api2 := NewAPI(engine)
	if err := api2.BootstrapCapabilityState(); err != nil {
		t.Fatalf("second BootstrapCapabilityState: %v", err)
	}
	api2.signingKeyMu.RLock()
	key2 := api2.signingKey
	api2.signingKeyMu.RUnlock()
	if key2 == nil {
		t.Fatalf("api2.signingKey is nil after reload")
	}
	for i := range key {
		if key[i] != key2[i] {
			t.Errorf("signing key mismatch after reload: keys differ at byte %d", i)
			break
		}
	}
}

// --- Task 2 (IPC handler) tests -----------------------------------------

// TestIPCHandlers_CapabilityRoundTrip: POST /sessions/{sid}/capabilities
// returns four fields (readUrl, writeUrl, readCode, writeCode). POSTing the
// readCode to /join/exchange returns a URL containing the same cap token as
// readUrl. This verifies the end-to-end join-code exchange.
func TestIPCHandlers_CapabilityRoundTrip(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	lnCfg := newLoopbackTLSListener(t)
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: lnCfg,
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer with TLS: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })
	api.SetWebServerForTest(ws)
	configureCapabilityStateForTest(t, api, ws)

	// Create + toggle ON.
	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"round-trip","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if st, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`); st != http.StatusNoContent {
		t.Fatalf("toggle-on: want 204, got %d", st)
	}

	// POST /sessions/{id}/capabilities
	st, issueBody := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/capabilities", cr.ID), ``)
	if st != http.StatusOK {
		t.Fatalf("POST /sessions/%s/capabilities: want 200, got %d; body: %s", cr.ID, st, issueBody)
	}
	var issue IssueCapabilitiesResponse
	if err := json.Unmarshal(issueBody, &issue); err != nil {
		t.Fatalf("decode IssueCapabilitiesResponse: %v", err)
	}
	if issue.ReadURL == "" || issue.WriteURL == "" || issue.ReadCode == "" || issue.WriteCode == "" {
		t.Fatalf("IssueCapabilitiesResponse has empty field: %+v", issue)
	}
	if !strings.Contains(issue.ReadURL, "?cap=") {
		t.Errorf("ReadURL missing ?cap=: %q", issue.ReadURL)
	}

	// POST /join/exchange with readCode — must return URL with same ?cap= as readUrl.
	exchangeBody := fmt.Sprintf(`{"code":%q}`, issue.ReadCode)
	st, xBody := rawPost(t, socketPath, "/join/exchange", exchangeBody)
	if st != http.StatusOK {
		t.Fatalf("POST /join/exchange: want 200, got %d; body: %s", st, xBody)
	}
	var xResp ExchangeJoinCodeResponse
	if err := json.Unmarshal(xBody, &xResp); err != nil {
		t.Fatalf("decode ExchangeJoinCodeResponse: %v", err)
	}
	// Extract ?cap= token from ReadURL and assert xResp.URL contains it.
	readTok := extractCapToken(issue.ReadURL)
	exchangedTok := extractCapToken(xResp.URL)
	if readTok == "" || exchangedTok == "" {
		t.Fatalf("could not extract cap tokens: readURL=%q exchanged=%q", issue.ReadURL, xResp.URL)
	}
	if readTok != exchangedTok {
		t.Errorf("join-code exchange produced different token: got %q, want %q", exchangedTok, readTok)
	}
}

// TestIPCHandlers_ExpiredCodeReturns410: issue a join code with a very short
// TTL, sleep past it, POST /join/exchange → 410 Gone.
func TestIPCHandlers_ExpiredCodeReturns410(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	lnCfg := newLoopbackTLSListener(t)
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: lnCfg,
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })
	api.SetWebServerForTest(ws)

	// Custom setup: use a very short join-code TTL so the test can expire
	// codes without simulated clocks.
	key, err := capability.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	api.signingKeyMu.Lock()
	api.signingKey = key
	api.signingKeyMu.Unlock()
	shortJC := capability.NewJoinCodeManager(50 * time.Millisecond)
	api.joinCodes = shortJC
	ws.SetSigningKey(key)
	ws.SetJoinCodes(shortJC)

	// Create + toggle ON.
	_, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"expired-code","workDir":""}`)
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if st, _ := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/web-serve", cr.ID), `{"enabled":true}`); st != http.StatusNoContent {
		t.Fatalf("toggle-on: want 204, got %d", st)
	}

	// Issue caps (this also registers join codes).
	st, issueBody := rawPost(t, socketPath, fmt.Sprintf("/sessions/%s/capabilities", cr.ID), ``)
	if st != http.StatusOK {
		t.Fatalf("issue: want 200, got %d; body: %s", st, issueBody)
	}
	var issue IssueCapabilitiesResponse
	if err := json.Unmarshal(issueBody, &issue); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Sleep past the 50ms TTL.
	time.Sleep(200 * time.Millisecond)

	st, _ = rawPost(t, socketPath, "/join/exchange", fmt.Sprintf(`{"code":%q}`, issue.ReadCode))
	if st != http.StatusGone {
		t.Errorf("expired code: want 410 Gone, got %d", st)
	}
}

// TestIPCHandlers_RegenerateSigningKey_SwapsKey: POST
// /capability/regenerate-key replaces the on-disk key AND the in-memory key,
// and updates ws.signingKey via SetSigningKey.
func TestIPCHandlers_RegenerateSigningKey_SwapsKey(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix domain sockets via api.Start(/tmp/...); production uses named pipes")
	}
	tmpDir := t.TempDir()
	engine := NewSessionEngine()
	engine.configDir = tmpDir
	engine.cliPaths = make(map[string]string)
	api := NewAPI(engine)
	if err := api.BootstrapCapabilityState(); err != nil {
		t.Fatalf("BootstrapCapabilityState: %v", err)
	}
	// Wire a WebServer so regenerate-key has a ws to call SetSigningKey on.
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: "127.0.0.1",
		Port:   0,
		FQDN:   "test.local",
	}, engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	api.SetWebServerForTest(ws)
	// Install the freshly-bootstrapped key onto the WS so regenerate can
	// swap it later.
	api.signingKeyMu.RLock()
	initialKey := append([]byte(nil), api.signingKey...)
	api.signingKeyMu.RUnlock()
	ws.SetSigningKey(initialKey)

	// Read current file bytes.
	initialFile, err := os.ReadFile(filepath.Join(tmpDir, "capability.key"))
	if err != nil {
		t.Fatalf("read capability.key: %v", err)
	}

	// Start the API on a socket, then POST /capability/regenerate-key.
	socketPath := shortSocketPath(t, "regen.sock")
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("api.Start: %v", err)
	}
	t.Cleanup(func() { api.Stop() })
	time.Sleep(10 * time.Millisecond)

	st, regenBody := rawPost(t, socketPath, "/capability/regenerate-key", ``)
	if st != http.StatusOK {
		t.Fatalf("POST /capability/regenerate-key: want 200, got %d; body: %s", st, regenBody)
	}

	// On-disk key bytes must have changed.
	newFile, err := os.ReadFile(filepath.Join(tmpDir, "capability.key"))
	if err != nil {
		t.Fatalf("read capability.key after regen: %v", err)
	}
	if sameBytes(initialFile, newFile) {
		t.Errorf("capability.key unchanged after regenerate")
	}

	// In-memory signing key must have changed.
	api.signingKeyMu.RLock()
	newKey := api.signingKey
	api.signingKeyMu.RUnlock()
	if sameBytes(initialKey, newKey) {
		t.Errorf("api.signingKey unchanged after regenerate")
	}
}

// --- shared helpers for Task 2 tests ------------------------------------

// sameBytes compares two byte slices for equality.
func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// extractCapToken pulls the value of the ?cap= query parameter from a URL.
func extractCapToken(urlStr string) string {
	i := strings.Index(urlStr, "?cap=")
	if i < 0 {
		return ""
	}
	tok := urlStr[i+len("?cap="):]
	if amp := strings.Index(tok, "&"); amp >= 0 {
		tok = tok[:amp]
	}
	return tok
}

// newLoopbackTLSListener constructs a self-signed TLS config bound to 127.0.0.1
// so WebServer.Start() can produce a real listener for tests that need
// BaseURL to be non-empty.
func newLoopbackTLSListener(t *testing.T) *tls.Config {
	t.Helper()
	cfg, err := webserver.GenerateSelfSignedCert("127.0.0.1")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}
	return cfg
}

// --- Phase 100 Plan 04: GET /shells + shell-session lifecycle ---------------

// TestHandleListShells_EmptyPATH (SHELL-04 wire format).
// With PATH pointing at an empty temp dir AND $SHELL unset, DiscoverShells
// must produce a non-nil empty slice, and the JSON wire body must be
// `{"shells":[]}` — never `{"shells":null}`.
func TestHandleListShells_EmptyPATH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	// Empty PATH so no known shell is discovered via exec.LookPath.
	t.Setenv("PATH", t.TempDir())
	// POSIX synthetic-default entry is keyed on $SHELL; empty $SHELL suppresses
	// the synthetic "shell" entry per Plan 01 H4 contract
	// (TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry).
	t.Setenv("SHELL", "")

	_, client, socketPath := testDaemon(t)

	// Wire-format assertion: status + raw body shape.
	status, body := rawGet(t, socketPath, "/shells")
	if status != 200 {
		t.Fatalf("GET /shells: want 200, got %d; body: %s", status, body)
	}

	// SHELL-04 must-have: empty discovery returns `"shells":[]`, not null.
	if bytes.Contains(body, []byte(`"shells":null`)) {
		t.Errorf("GET /shells body must not contain \"shells\":null when empty; got: %s", body)
	}
	if !bytes.Contains(body, []byte(`"shells":[]`)) {
		t.Errorf("GET /shells body must contain \"shells\":[] when empty; got: %s", body)
	}

	var resp ShellsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode ShellsResponse: %v; body: %s", err, body)
	}
	if resp.Shells == nil {
		t.Error("ShellsResponse.Shells must be non-nil (initialised empty slice), got nil")
	}
	if len(resp.Shells) != 0 {
		t.Errorf("expected 0 shells with empty PATH+SHELL, got %d: %+v", len(resp.Shells), resp.Shells)
	}

	// Sanity: client.ListShells round-trips the same wire body.
	shells, err := client.ListShells()
	if err != nil {
		t.Fatalf("client.ListShells: %v", err)
	}
	if shells == nil {
		t.Error("client.ListShells() must return a non-nil slice, got nil")
	}
	if len(shells) != 0 {
		t.Errorf("client.ListShells empty-PATH: want 0, got %d", len(shells))
	}
}

// TestHandleListShells_PopulatedPATH (SHELL-04 wire format, dev-host smoke).
// Uses the real host PATH; asserts at least one shell is discovered and each
// entry has the required wire fields populated.
func TestHandleListShells_PopulatedPATH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, client, socketPath := testDaemon(t)

	status, body := rawGet(t, socketPath, "/shells")
	if status != 200 {
		t.Fatalf("GET /shells: want 200, got %d; body: %s", status, body)
	}

	var resp ShellsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode ShellsResponse: %v; body: %s", err, body)
	}
	if len(resp.Shells) < 1 {
		t.Fatalf("expected at least one shell on dev host PATH, got 0; body: %s", body)
	}

	knownNames := map[string]bool{
		"bash":       true,
		"zsh":        true,
		"pwsh":       true,
		"powershell": true,
		"shell":      true,
	}
	sawKnown := false
	for i, s := range resp.Shells {
		if s.Name == "" {
			t.Errorf("Shells[%d].Name is empty", i)
		}
		if s.Path == "" {
			t.Errorf("Shells[%d].Path is empty (name=%q)", i, s.Name)
		}
		if s.DisplayName == "" {
			t.Errorf("Shells[%d].DisplayName is empty (name=%q)", i, s.Name)
		}
		if len(s.Argv) < 1 {
			t.Errorf("Shells[%d].Argv must have len >= 1; got %v (name=%q)", i, s.Argv, s.Name)
		}
		if knownNames[s.Name] {
			sawKnown = true
		}
	}
	if !sawKnown {
		t.Errorf("expected at least one shell in {bash,zsh,pwsh,powershell,shell}, got %+v", resp.Shells)
	}

	// Client-side round-trip parity: same count, same first entry.
	clientShells, err := client.ListShells()
	if err != nil {
		t.Fatalf("client.ListShells: %v", err)
	}
	if len(clientShells) != len(resp.Shells) {
		t.Errorf("client.ListShells len mismatch: client=%d, raw=%d", len(clientShells), len(resp.Shells))
	}
}

// TestShellSessionLifecycle_StatusOnlyRunningOrStopped (SHELL-09 end-to-end).
// Drives a full create→list-poll→kill cycle for a bash session and asserts
// Status only takes values in {"running", "stopped"} — never
// "waiting"/"error"/"errored"/"idle". POSIX-only because cli="bash" requires
// a POSIX shell binary on PATH.
func TestShellSessionLifecycle_StatusOnlyRunningOrStopped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell (bash)")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not installed on PATH")
	}

	_, client, _ := testDaemon(t)

	id, err := client.CreateSession("bash", "lifecycle-test", "", nil, 80, 24)
	if err != nil {
		t.Fatalf("client.CreateSession(bash): %v", err)
	}
	if id == "" {
		t.Fatal("CreateSession returned empty ID")
	}

	// Forbidden statuses per SHELL-09: shell sessions must never surface
	// AI-CLI heuristic states.
	forbidden := map[string]bool{
		"waiting": true,
		"error":   true,
		"errored": true,
		"idle":    true,
	}

	// findStatus returns the SessionInfo for the test session, or (zero, false)
	// if it has been removed from the registry (e.g. after KillSession).
	findStatus := func(t *testing.T) (SessionInfo, bool) {
		t.Helper()
		sessions, err := client.ListSessions()
		if err != nil {
			t.Fatalf("client.ListSessions: %v", err)
		}
		for _, s := range sessions {
			if s.ID == id {
				return s, true
			}
		}
		return SessionInfo{}, false
	}

	// Brief wait for the session to register.
	time.Sleep(100 * time.Millisecond)

	// Sample 5 times over ~1s while the bash PTY is alive: Status must always
	// be "running" (never flicker to waiting/idle/error/errored).
	for i := 0; i < 5; i++ {
		info, ok := findStatus(t)
		if !ok {
			t.Fatalf("session %s missing during running-phase sample %d", id, i)
		}
		if forbidden[info.Status] {
			t.Errorf("sample %d: Status=%q is forbidden for shell sessions (SHELL-09); want \"running\" or \"stopped\"", i, info.Status)
		}
		if info.Status != "running" && info.Status != "stopped" {
			t.Errorf("sample %d: Status=%q not in {running,stopped}", i, info.Status)
		}
		if info.Status != "running" {
			// The bash PTY is alive — Status should be "running".
			t.Errorf("sample %d: expected Status=\"running\" while bash PTY alive, got %q", i, info.Status)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Kill the session and poll until either (a) the session is removed from
	// the registry, or (b) its State transitions to "stopped". Throughout the
	// poll, Status must remain in {running,stopped} and never appear in
	// `forbidden`.
	if err := client.KillSession(id); err != nil {
		t.Fatalf("client.KillSession: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	terminated := false
	for time.Now().Before(deadline) {
		info, ok := findStatus(t)
		if !ok {
			// KillSession removes the session from the registry — this is the
			// expected terminal state for an explicitly-killed shell session.
			terminated = true
			break
		}
		if forbidden[info.Status] {
			t.Errorf("post-kill sample: Status=%q is forbidden for shell sessions (SHELL-09)", info.Status)
		}
		if info.Status != "running" && info.Status != "stopped" {
			t.Errorf("post-kill sample: Status=%q not in {running,stopped}", info.Status)
		}
		if info.State == "stopped" {
			terminated = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !terminated {
		t.Errorf("session %s did not terminate (state=stopped or removed) within 2s of KillSession", id)
	}

	// Final SHELL-04 anchor: client.ListShells must still answer cleanly even
	// after a session lifecycle (no engine state corruption).
	if _, err := client.ListShells(); err != nil {
		t.Errorf("client.ListShells after session lifecycle: %v", err)
	}
}

// --- Phase 101 Plan 01: shell-web-share-warned route ----------------------

// TestAPIGetShellWebShareWarned_Default verifies the default value is false
// on a fresh engine.
func TestAPIGetShellWebShareWarned_Default(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/settings/shell-web-share-warned")
	if status != 200 {
		t.Errorf("GET /settings/shell-web-share-warned: want 200, got %d (body=%s)", status, string(body))
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, string(body))
	}
	if resp["value"] != false {
		t.Errorf("default value: got %v, want false", resp["value"])
	}
}

// TestAPIPatchShellWebShareWarned_FlipsTrue verifies that PATCH flips the
// engine state and a subsequent GET observes the new value.
func TestAPIPatchShellWebShareWarned_FlipsTrue(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawPatch(t, socketPath, "/settings/shell-web-share-warned", `{"value":true}`)
	if status != 204 {
		t.Errorf("PATCH /settings/shell-web-share-warned: want 204, got %d (body=%s)", status, string(body))
	}

	status, body = rawGet(t, socketPath, "/settings/shell-web-share-warned")
	if status != 200 {
		t.Errorf("GET after PATCH: want 200, got %d", status)
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp["value"] != true {
		t.Errorf("value after PATCH: got %v, want true", resp["value"])
	}
}

// TestAPIPatchShellWebShareWarned_BadBody verifies that a malformed body
// produces a 400 response.
func TestAPIPatchShellWebShareWarned_BadBody(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, _ := rawPatch(t, socketPath, "/settings/shell-web-share-warned", `not-json`)
	if status != 400 {
		t.Errorf("PATCH /settings/shell-web-share-warned (bad body): want 400, got %d", status)
	}
}

// TestDaemonClient_GetSetShellWebShareWarned_RoundTrip verifies that the
// DaemonClient wrapper round-trips through the HTTP API.
func TestDaemonClient_GetSetShellWebShareWarned_RoundTrip(t *testing.T) {
	_, client, _ := testDaemon(t)

	v, err := client.GetShellWebShareWarned()
	if err != nil {
		t.Fatalf("initial GetShellWebShareWarned: %v", err)
	}
	if v != false {
		t.Errorf("initial value: got %v, want false", v)
	}

	if err := client.SetShellWebShareWarned(true); err != nil {
		t.Fatalf("SetShellWebShareWarned(true): %v", err)
	}

	v, err = client.GetShellWebShareWarned()
	if err != nil {
		t.Fatalf("post-set GetShellWebShareWarned: %v", err)
	}
	if v != true {
		t.Errorf("post-set value: got %v, want true", v)
	}
}
