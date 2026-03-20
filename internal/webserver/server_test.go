package webserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/agenthub/agenthub/internal/relay"
	"github.com/agenthub/agenthub/internal/webserver"
	"github.com/coder/websocket"
)

// testServer creates a WebServer in test mode with an in-memory CA.
// It returns the server and a TLS-enabled HTTP client for making requests.
func testServer(t *testing.T) (*webserver.WebServer, *http.Client) {
	t.Helper()
	manager := relay.NewHubManager()
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0, // random port
		ConfigDir: t.TempDir(),
	}
	ws, err := webserver.NewWebServer(cfg, manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.SetPassword("testpass"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })

	client := ws.TestClient()
	return ws, client
}

// login performs a POST /login and returns the session cookie value.
func login(t *testing.T, client *http.Client, baseURL, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"password": password})
	resp, err := client.Post(baseURL+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /login: expected 200, got %d: %s", resp.StatusCode, b)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "agenthub_session" {
			return c.Value
		}
	}
	t.Fatal("POST /login: no agenthub_session cookie in response")
	return ""
}

// TestWebServerDashboardNoAuthRequired verifies that GET /dashboard is publicly
// accessible (the HTML JS handles login state client-side). This replaces the
// old "RequiresAuth" check which reflected the pre-fix broken behaviour.
func TestWebServerDashboardNoAuthRequired(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/dashboard")
	if err != nil {
		t.Fatalf("GET /dashboard: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestWebServerLoginSetsCoookie(t *testing.T) {
	ws, client := testServer(t)
	body, _ := json.Marshal(map[string]string{"password": "testpass"})
	resp, err := client.Post(ws.BaseURL()+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	found := false
	for _, c := range resp.Cookies() {
		if c.Name == "agenthub_session" {
			found = true
			if c.Value == "" {
				t.Error("agenthub_session cookie value is empty")
			}
		}
	}
	if !found {
		t.Error("no agenthub_session cookie in response")
	}
}

func TestWebServerLoginBadPassword(t *testing.T) {
	ws, client := testServer(t)
	body, _ := json.Marshal(map[string]string{"password": "wrongpass"})
	resp, err := client.Post(ws.BaseURL()+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestWebServerDashboardAfterLogin(t *testing.T) {
	ws, client := testServer(t)
	baseURL := ws.BaseURL()
	login(t, client, baseURL, "testpass")

	resp, err := client.Get(baseURL + "/dashboard")
	if err != nil {
		t.Fatalf("GET /dashboard: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "AgentHub Dashboard") {
		t.Error("dashboard HTML should contain 'AgentHub Dashboard'")
	}
}

func TestWebServerSessionListAPI(t *testing.T) {
	ws, client := testServer(t)
	baseURL := ws.BaseURL()

	// Add a session to manager and enable it
	manager := relay.NewHubManager()
	_ = manager // not used directly; ws has its own manager
	ws.EnableSession("sess1")

	login(t, client, baseURL, "testpass")

	resp, err := client.Get(baseURL + "/api/sessions")
	if err != nil {
		t.Fatalf("GET /api/sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	// sess1 is enabled but not in the hub manager — it should still appear
	found := false
	for _, item := range items {
		if item.ID == "sess1" {
			found = true
			// Name falls back to session ID when no resolver is set
			if item.Name != "sess1" {
				t.Errorf("expected name fallback to 'sess1', got %q", item.Name)
			}
		}
	}
	if !found {
		t.Errorf("expected sess1 in sessions, got %v", items)
	}
}

func TestWebServerSessionListAPIWithResolver(t *testing.T) {
	ws, client := testServer(t)
	baseURL := ws.BaseURL()

	ws.SetSessionResolver(func(id string) (string, string, string) {
		if id == "sess1" {
			return "My Session", "claude", "running"
		}
		return "", "", ""
	})
	ws.EnableSession("sess1")

	login(t, client, baseURL, "testpass")

	resp, err := client.Get(baseURL + "/api/sessions")
	if err != nil {
		t.Fatalf("GET /api/sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var items []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		CLIType string `json:"cli_type"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 session, got %d", len(items))
	}
	if items[0].Name != "My Session" {
		t.Errorf("expected name 'My Session', got %q", items[0].Name)
	}
	if items[0].CLIType != "claude" {
		t.Errorf("expected cli_type 'claude', got %q", items[0].CLIType)
	}
	if items[0].Status != "running" {
		t.Errorf("expected status 'running', got %q", items[0].Status)
	}
}

func TestWebServerTokenAccess(t *testing.T) {
	ws, client := testServer(t)
	baseURL := ws.BaseURL()

	// Enable a session so terminal.html can be served
	ws.EnableSession("sess1")

	// Get a token via the dashboard API (login first)
	login(t, client, baseURL, "testpass")

	resp, err := client.Post(baseURL+"/api/sessions/sess1/token", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/sessions/sess1/token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST token: expected 200, got %d: %s", resp.StatusCode, b)
	}
	var tokenResp struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if tokenResp.Token == "" {
		t.Error("token is empty")
	}
	if !strings.Contains(tokenResp.URL, "token=") {
		t.Errorf("URL should contain token param, got %s", tokenResp.URL)
	}

	// Now access /sessions/sess1 with the token WITHOUT a session cookie (use fresh client)
	freshClient := ws.TestClient()
	// Don't login with freshClient
	tokenURL := fmt.Sprintf("%s/sessions/sess1?token=%s", baseURL, tokenResp.Token)
	resp2, err := freshClient.Get(tokenURL)
	if err != nil {
		t.Fatalf("GET session with token: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp2.Body)
		t.Errorf("expected 200 with valid token, got %d: %s", resp2.StatusCode, b)
	}
}

func TestWebServerTokenAccessInvalid(t *testing.T) {
	ws, client := testServer(t)
	_ = client
	ws.EnableSession("sess1")

	freshClient := ws.TestClient()
	resp, err := freshClient.Get(ws.BaseURL() + "/sessions/sess1?token=invalidtoken")
	if err != nil {
		t.Fatalf("GET session with invalid token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with invalid token, got %d", resp.StatusCode)
	}
}

func TestWebServerCACertDownload(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/ca.crt")
	if err != nil {
		t.Fatalf("GET /ca.crt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "pem") && !strings.Contains(ct, "x-pem-file") {
		t.Errorf("expected PEM content-type, got %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	block, _ := pem.Decode(body)
	if block == nil {
		t.Error("CA cert response is not valid PEM")
	}
}

func TestWebServerWSS(t *testing.T) {
	mgr := relay.NewHubManager()
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		ConfigDir: t.TempDir(),
	}
	ws, err := webserver.NewWebServer(cfg, mgr)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.SetPassword("testpass"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	defer ws.Stop()

	// Create a hub with a pipe so we can send output
	pr, pw := io.Pipe()
	hub := mgr.Create("sess1", pr, io.Discard, nil)
	_ = hub
	ws.EnableSession("sess1")

	// Use a TLS client that trusts the CA
	client := ws.TestClient()
	baseURL := ws.BaseURL()

	// Login to get session cookie
	login(t, client, baseURL, "testpass")

	// Connect via WSS using the authenticated client's cookie jar
	wsURL := strings.Replace(baseURL, "https://", "wss://", 1) + "/sessions/sess1/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build http.Header with the session cookie
	jar := client.Jar
	cookies := jar.Cookies(mustParseURL(baseURL))
	headers := http.Header{}
	for _, c := range cookies {
		headers.Add("Cookie", c.String())
	}

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPClient: client,
		HTTPHeader: headers,
	})
	if err != nil {
		t.Fatalf("websocket.Dial: %v", err)
	}
	defer conn.CloseNow()

	// Write output to hub — should arrive as MsgOutput frame
	testData := []byte("hello from hub")
	go func() {
		time.Sleep(50 * time.Millisecond)
		pw.Write(testData)
	}()

	msgType, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("Read from WSS: %v", err)
	}
	if msgType != websocket.MessageBinary {
		t.Errorf("expected binary message, got %v", msgType)
	}
	if len(msg) == 0 || msg[0] != relay.MsgOutput {
		t.Errorf("expected MsgOutput frame, got %v", msg)
	}
	payload := msg[1:]
	if !bytes.Equal(payload, testData) {
		t.Errorf("expected payload %q, got %q", testData, payload)
	}
}

func TestWebServerToggle(t *testing.T) {
	mgr := relay.NewHubManager()
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		ConfigDir: t.TempDir(),
	}
	ws, err := webserver.NewWebServer(cfg, mgr)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.SetPassword("testpass"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	defer ws.Stop()

	client := ws.TestClient()
	baseURL := ws.BaseURL()

	// Enable sess1, login, access terminal page
	ws.EnableSession("sess1")
	login(t, client, baseURL, "testpass")

	resp, err := client.Get(baseURL + "/sessions/sess1")
	if err != nil {
		t.Fatalf("GET /sessions/sess1: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("enabled session: expected 200, got %d", resp.StatusCode)
	}

	// Disable sess1 — should return 404
	ws.DisableSession("sess1")
	resp2, err := client.Get(baseURL + "/sessions/sess1")
	if err != nil {
		t.Fatalf("GET /sessions/sess1 after disable: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("disabled session: expected 404, got %d", resp2.StatusCode)
	}
}

func TestQREndpoint(t *testing.T) {
	ws, client := testServer(t)
	baseURL := ws.BaseURL()

	ws.EnableSession("sess-qr")
	login(t, client, baseURL, "testpass")

	resp, err := client.Get(baseURL + "/api/sessions/sess-qr/qr")
	if err != nil {
		t.Fatalf("GET /api/sessions/sess-qr/qr: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "image/png" {
		t.Errorf("expected Content-Type image/png, got %q", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) < 4 {
		t.Fatalf("PNG body too short: %d bytes", len(body))
	}
	if body[0] != 0x89 || body[1] != 'P' || body[2] != 'N' || body[3] != 'G' {
		t.Errorf("expected PNG magic bytes, got %v", body[:4])
	}
}

func TestQREndpointNotEnabled(t *testing.T) {
	ws, client := testServer(t)
	baseURL := ws.BaseURL()

	login(t, client, baseURL, "testpass")

	// "not-enabled-session" is NOT web-enabled — should return 404
	resp, err := client.Get(baseURL + "/api/sessions/not-enabled-session/qr")
	if err != nil {
		t.Fatalf("GET /api/sessions/not-enabled-session/qr: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-enabled session, got %d", resp.StatusCode)
	}
}

// TestDashboardNoAuthReturns200 verifies that GET /dashboard is accessible
// without authentication — required for the login form to be reachable.
func TestDashboardNoAuthReturns200(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestAPISessionsStillRequiresAuth verifies that /api/sessions is still protected.
func TestAPISessionsStillRequiresAuth(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/api/sessions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// mustParseURL is a test helper.
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
