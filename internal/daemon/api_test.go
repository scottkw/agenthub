package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

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

func TestCreateSession_AutoWebEnable(t *testing.T) {
	api, _, socketPath := testDaemon(t)
	// Create a WebServer (without Start — no TLS needed for EnableSession).
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: "127.0.0.1",
		Port:   0,
		FQDN:   "test.local",
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer for test: %v", err)
	}
	api.SetWebServerForTest(ws)

	// Create a session — it should be auto-enabled.
	status, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"auto-web","workDir":""}`)
	if status != 201 {
		t.Fatalf("POST /sessions: want 201, got %d; body: %s", status, body)
	}
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	// GET /sessions should show WebEnabled=true for this session.
	_, listBody := rawGet(t, socketPath, "/sessions")
	var sessions []SessionInfo
	if err := json.Unmarshal(listBody, &sessions); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.ID == cr.ID {
			found = true
			if !s.WebEnabled {
				t.Errorf("session %s: want WebEnabled=true, got false", cr.ID)
			}
		}
	}
	if !found {
		t.Errorf("session %s not found in list", cr.ID)
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
