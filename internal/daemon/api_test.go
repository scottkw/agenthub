package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// testDaemon creates an API server with a fresh engine on a temp socket.
// Returns the API, a DaemonClient connected to it, and the socket path.
func testDaemon(t *testing.T) (*API, *DaemonClient, string) {
	t.Helper()
	engine := NewSessionEngine()
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
