package webserver_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/webserver"
)

// ssExtTestKey is a deterministic 32-byte HMAC key used by external-package
// tests to mint capabilities after Phase 87 gated the API routes.
var ssExtTestKey = func() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(0xA0 + i)
	}
	return key
}()

// capForSession mints a read+write capability for sessionID, registers its
// grant_id on ws (so requireCapability's grant-list check passes), and
// returns the URL-ready token. The caller must have already installed
// ssExtTestKey via ws.SetSigningKey.
func capForSession(t *testing.T, ws *webserver.WebServer, sessionID string) string {
	t.Helper()
	claims := capability.Claims{
		SID:     sessionID,
		Perms:   "read,write",
		IAT:     time.Now().Unix(),
		GrantID: "ext-grant-" + sessionID,
		V:       1,
	}
	token, err := capability.Sign(claims, ssExtTestKey)
	if err != nil {
		t.Fatalf("capability.Sign: %v", err)
	}
	ws.AddGrant(sessionID, claims.GrantID)
	return token
}

// selfSignedTLSForTest generates an in-memory self-signed CA and leaf cert for
// 127.0.0.1. Returns a server TLS config and an HTTP client that trusts the CA.
func selfSignedTLSForTest(t *testing.T) (*tls.Config, *http.Client) {
	t.Helper()
	// Generate CA
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Test CA"}},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	// Generate leaf for 127.0.0.1
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	leafCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	leafKeyDER, _ := x509.MarshalECPrivateKey(leafKey)
	leafKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})
	tlsCert, err := tls.X509KeyPair(leafCertPEM, leafKeyPEM)
	if err != nil {
		t.Fatal(err)
	}
	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}
	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}}
	return serverTLS, client
}

// testServer creates a WebServer in test mode using the TLSConfig override.
// It returns the server and a TLS-enabled HTTP client for making requests.
func testServer(t *testing.T) (*webserver.WebServer, *http.Client) {
	t.Helper()
	manager := relay.NewHubManager()
	tlsCfg, client := selfSignedTLSForTest(t)
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })
	return ws, client
}

// TestWebServerDashboardNoAuthRequired verifies that GET /dashboard is publicly
// accessible without authentication.
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

func TestWebServerSessionListAPI(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	baseURL := ws.BaseURL()

	ws.EnableSession("sess1")
	token := capForSession(t, ws, "sess1")

	resp, err := client.Get(baseURL + "/api/sessions?cap=" + token)
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
	// Phase 87 D-18: response contains ONLY the cap-bound session, never
	// more than one item, even if other sessions are enabled.
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 session (D-18), got %d: %v", len(items), items)
	}
	if items[0].ID != "sess1" {
		t.Errorf("expected id=sess1, got %q", items[0].ID)
	}
	// Name falls back to session ID when no resolver is set.
	if items[0].Name != "sess1" {
		t.Errorf("expected name fallback to 'sess1', got %q", items[0].Name)
	}
}

func TestWebServerSessionListAPIWithResolver(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	baseURL := ws.BaseURL()

	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == "sess1" {
			return "My Session", "claude", "running", "test-host.local"
		}
		return "", "", "", ""
	})
	ws.EnableSession("sess1")
	token := capForSession(t, ws, "sess1")

	resp, err := client.Get(baseURL + "/api/sessions?cap=" + token)
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

func TestWebServerWSS(t *testing.T) {
	mgr := relay.NewHubManager()
	tlsCfg, client := selfSignedTLSForTest(t)
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, mgr)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	ws.SetSigningKey(ssExtTestKey)
	defer ws.Stop()

	// Create a hub with a pipe so we can send output
	pr, pw := io.Pipe()
	hub := mgr.Create("sess1", pr, io.Discard, nil)
	_ = hub
	ws.EnableSession("sess1")
	token := capForSession(t, ws, "sess1")

	baseURL := ws.BaseURL()
	wsURL := strings.Replace(baseURL, "https://", "wss://", 1) + "/sessions/sess1/ws?cap=" + token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Phase 88: include Origin header so requireAllowedOrigin middleware passes.
	wsHeaders := http.Header{}
	wsHeaders.Set("Origin", baseURL)
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPClient: client,
		HTTPHeader: wsHeaders,
	})
	if err != nil {
		t.Fatalf("websocket.Dial: %v", err)
	}
	defer conn.CloseNow()

	// Write output to hub — should arrive as MsgOutput frame.
	// Note: NotifyViewerCount sends a MsgMeta frame immediately on subscribe,
	// so we skip any leading MsgMeta frames and wait for the first MsgOutput.
	testData := []byte("hello from hub")
	go func() {
		time.Sleep(50 * time.Millisecond)
		pw.Write(testData)
	}()

	var msg []byte
	for {
		msgType, m, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("Read from WSS: %v", err)
		}
		if msgType != websocket.MessageBinary {
			t.Errorf("expected binary message, got %v", msgType)
			break
		}
		if len(m) > 0 && (m[0] == relay.MsgMeta || m[0] == relay.MsgPresence || m[0] == relay.MsgResize) {
			continue // skip server-push housekeeping frames (meta, presence, Phase 157 join-push resize)
		}
		msg = m
		break
	}
	if len(msg) == 0 || msg[0] != relay.MsgOutput {
		t.Errorf("expected MsgOutput frame, got %v", msg)
	}
	payload := msg[1:]
	if !bytes.Equal(payload, testData) {
		t.Errorf("expected payload %q, got %q", testData, payload)
	}
}

// TestWebServerToggle verifies the enabled/disabled toggle semantics through
// the capability-gated /sessions/{id} route (Phase 87). With a valid cap and
// the session enabled, the terminal page loads (200). After DisableSession,
// the grant-list + web-enabled check in requireCapability rejects with 403
// (not 404, because the cap is structurally valid — it's just been
// invalidated by the toggle, mirroring D-15).
func TestWebServerToggle(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	baseURL := ws.BaseURL()

	ws.EnableSession("sess1")
	token := capForSession(t, ws, "sess1")
	resp, err := client.Get(baseURL + "/sessions/sess1?cap=" + token)
	if err != nil {
		t.Fatalf("GET /sessions/sess1: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("enabled session: expected 200, got %d", resp.StatusCode)
	}

	// Disable sess1 — requireCapability's IsSessionEnabled check now fails,
	// so the request is rejected as revoked (403). D-15 toggle-off clears
	// grants; this test doesn't call ClearGrants explicitly, so the 403
	// comes from the web-enabled check rather than the grant-list check,
	// which is equivalent end-to-end (both map to StatusForbidden).
	ws.DisableSession("sess1")
	resp2, err := client.Get(baseURL + "/sessions/sess1?cap=" + token)
	if err != nil {
		t.Fatalf("GET /sessions/sess1 after disable: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("disabled session: expected 403, got %d", resp2.StatusCode)
	}
}

func TestQREndpoint(t *testing.T) {
	ws, client := testServer(t)
	baseURL := ws.BaseURL()

	ws.EnableSession("sess-qr")

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
// without authentication.
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

// TestBaseURL_UsesFQDN verifies that BaseURL() uses the configured FQDN, not the bind IP.
func TestBaseURL_UsesFQDN(t *testing.T) {
	tlsCfg, _ := selfSignedTLSForTest(t)
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "myhost.example.ts.net",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, relay.NewHubManager())
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Start(); err != nil {
		t.Fatal(err)
	}
	defer ws.Stop()
	base := ws.BaseURL()
	if !strings.HasPrefix(base, "https://myhost.example.ts.net:") {
		t.Errorf("BaseURL should use FQDN, got %q", base)
	}
}

func TestLoginRouteNotRegistered(t *testing.T) {
	ws, client := testServer(t)
	body, _ := json.Marshal(map[string]string{"password": "test"})
	resp, err := client.Post(ws.BaseURL()+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /login: %v", err)
	}
	defer resp.Body.Close()
	// POST /login should not be registered — expect 405 Method Not Allowed
	// (Go 1.22+ method-based routing returns 405 for wrong method, 404 for unmatched path)
	if resp.StatusCode == http.StatusOK {
		t.Error("POST /login should not return 200 — route should be removed")
	}
}

func TestTokenRouteNotRegistered(t *testing.T) {
	ws, client := testServer(t)
	ws.EnableSession("sess1")
	resp, err := client.Post(ws.BaseURL()+"/api/sessions/sess1/token", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/sessions/sess1/token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("POST /api/sessions/sess1/token should not return 200 — route should be removed")
	}
}

// TestSessionListIncludesHostname verifies that GET /api/sessions returns items
// with a "hostname" JSON key.
func TestSessionListIncludesHostname(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	baseURL := ws.BaseURL()

	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == "sess1" {
			return "My Session", "claude", "running", "test-host.local"
		}
		return "", "", "", ""
	})
	ws.EnableSession("sess1")
	token := capForSession(t, ws, "sess1")

	resp, err := client.Get(baseURL + "/api/sessions?cap=" + token)
	if err != nil {
		t.Fatalf("GET /api/sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var items []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 session, got %d", len(items))
	}
	if items[0].Hostname != "test-host.local" {
		t.Errorf("expected hostname 'test-host.local', got %q", items[0].Hostname)
	}
}

// TestSessionInfoEndpoint verifies that GET /api/sessions/{id}/info returns
// full session metadata for an enabled session.
func TestSessionInfoEndpoint(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	baseURL := ws.BaseURL()

	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == "sess1" {
			return "My Session", "claude", "running", "test-host.local"
		}
		return id, "", "", ""
	})
	ws.EnableSession("sess1")
	token := capForSession(t, ws, "sess1")

	resp, err := client.Get(baseURL + "/api/sessions/sess1/info?cap=" + token)
	if err != nil {
		t.Fatalf("GET /api/sessions/sess1/info: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var item struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		CLIType  string `json:"cli_type"`
		Status   string `json:"status"`
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		t.Fatalf("decode session info: %v", err)
	}
	if item.ID != "sess1" {
		t.Errorf("expected id 'sess1', got %q", item.ID)
	}
	if item.Name != "My Session" {
		t.Errorf("expected name 'My Session', got %q", item.Name)
	}
	if item.CLIType != "claude" {
		t.Errorf("expected cli_type 'claude', got %q", item.CLIType)
	}
	if item.Status != "running" {
		t.Errorf("expected status 'running', got %q", item.Status)
	}
	if item.Hostname != "test-host.local" {
		t.Errorf("expected hostname 'test-host.local', got %q", item.Hostname)
	}
}

// TestSessionInfoEndpoint_NotEnabled verifies that GET /api/sessions/{id}/info
// rejects a request whose capability binds to a session that is not
// web-enabled. Phase 87: requireCapability's grant-list check (and its
// web-enabled cross-check) returns 403 when the session isn't live.
func TestSessionInfoEndpoint_NotEnabled(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	baseURL := ws.BaseURL()

	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		return "test", "claude", "running", "testhost"
	})
	// Mint a cap for sess1, but do NOT call EnableSession. The AddGrant is
	// still performed so the grant-list check passes — the rejection comes
	// from the web-enabled defense-in-depth cross-check inside
	// requireCapability.
	token := capForSession(t, ws, "sess1")

	resp, err := client.Get(baseURL + "/api/sessions/sess1/info?cap=" + token)
	if err != nil {
		t.Fatalf("GET /api/sessions/sess1/info: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for non-enabled session, got %d", resp.StatusCode)
	}
}

// TestSessionInfoEndpoint_NotFound verifies that GET /api/sessions/{id}/info
// returns 404 for a session whose resolver returns defaults (i.e. the
// session ID isn't registered with the engine), even when the session is
// web-enabled and has a valid cap. The 404 comes from handleSessionInfo's
// resolver-defaults check, not from requireCapability.
func TestSessionInfoEndpoint_NotFound(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	baseURL := ws.BaseURL()

	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		// Return default values — session not found in resolver
		return id, "", "", ""
	})
	ws.EnableSession("nonexistent")
	token := capForSession(t, ws, "nonexistent")

	resp, err := client.Get(baseURL + "/api/sessions/nonexistent/info?cap=" + token)
	if err != nil {
		t.Fatalf("GET /api/sessions/nonexistent/info: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent session, got %d", resp.StatusCode)
	}
}

// TestSessionAccessWithoutAuth is the inverted form of the pre-Phase-87
// "tailnet membership is sufficient" check. After Phase 87, a request to
// /sessions/{id} without a ?cap= parameter must be rejected (401), even
// when the session is web-enabled. This is the HTTP-layer expression of
// SEC-02/SEC-03.
func TestSessionAccessWithoutAuth(t *testing.T) {
	ws, _ := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	ws.EnableSession("sess1")
	// Use a fresh client with InsecureSkipVerify — no cookies, no tokens
	freshClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}}
	resp, err := freshClient.Get(ws.BaseURL() + "/sessions/sess1")
	if err != nil {
		t.Fatalf("GET /sessions/sess1: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for web-enabled session without capability, got %d", resp.StatusCode)
	}
}

// insecureClient returns an *http.Client that skips TLS certificate verification.
// Used for local-mode tests where the self-signed cert is not in a trusted pool.
func insecureClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}}
}

// TestLocalModeStart verifies that a WebServer in local mode starts, requires
// Basic Auth, and returns 200 when correct credentials are supplied.
func TestLocalModeStart(t *testing.T) {
	manager := relay.NewHubManager()
	tlsCfg, _ := selfSignedTLSForTest(t)
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		Mode:      "local",
		Password:  "testpass",
		TLSConfig: tlsCfg, // override so we don't need GenerateSelfSignedCert in test
	}
	ws, err := webserver.NewWebServer(cfg, manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })

	client := insecureClient()
	base := ws.BaseURL()

	// Request without auth → 401
	resp, err := client.Get(base + "/dashboard")
	if err != nil {
		t.Fatalf("GET /dashboard (no auth): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no auth: expected 401, got %d", resp.StatusCode)
	}

	// Request with correct password → 200
	req, _ := http.NewRequest(http.MethodGet, base+"/dashboard", nil)
	req.SetBasicAuth("user", "testpass")
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /dashboard (with auth): %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("with auth: expected 200, got %d", resp2.StatusCode)
	}
}

// TestBaseURL_LocalMode verifies that BaseURL() returns an IP-based URL (not FQDN)
// when Mode is "local".
func TestBaseURL_LocalMode(t *testing.T) {
	tlsCfg, _ := selfSignedTLSForTest(t)
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "myhost.example.ts.net",
		Mode:      "local",
		Password:  "testpass",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, relay.NewHubManager())
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Start(); err != nil {
		t.Fatal(err)
	}
	defer ws.Stop()

	base := ws.BaseURL()
	if !strings.HasPrefix(base, "https://127.0.0.1:") {
		t.Errorf("local mode BaseURL should use BindIP, got %q", base)
	}
	if strings.Contains(base, "myhost.example.ts.net") {
		t.Errorf("local mode BaseURL should not contain FQDN, got %q", base)
	}
}

// TestBaseURL_TailscaleMode verifies that BaseURL() returns an FQDN-based URL
// when Mode is "tailscale" (existing behavior preserved).
func TestBaseURL_TailscaleMode(t *testing.T) {
	tlsCfg, _ := selfSignedTLSForTest(t)
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "myhost.example.ts.net",
		Mode:      "tailscale",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, relay.NewHubManager())
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Start(); err != nil {
		t.Fatal(err)
	}
	defer ws.Stop()

	base := ws.BaseURL()
	if !strings.HasPrefix(base, "https://myhost.example.ts.net:") {
		t.Errorf("tailscale mode BaseURL should use FQDN, got %q", base)
	}
}

// TestMode_Accessor verifies that the Mode() accessor returns the configured mode.
func TestMode_Accessor(t *testing.T) {
	tlsCfg, _ := selfSignedTLSForTest(t)
	for _, mode := range []string{"local", "tailscale", ""} {
		cfg := webserver.Config{
			BindIP:    "127.0.0.1",
			Port:      0,
			FQDN:      "host.ts.net",
			Mode:      mode,
			Password:  "pw",
			TLSConfig: tlsCfg,
		}
		ws, err := webserver.NewWebServer(cfg, relay.NewHubManager())
		if err != nil {
			t.Fatal(err)
		}
		if got := ws.Mode(); got != mode {
			t.Errorf("Mode() = %q, want %q", got, mode)
		}
	}
}

// dialWebWS is a shared helper for the VIEW-03/VIEW-02 webserver tests that
// dials a capability-gated WebSocket session, sets the required Origin header
// so requireAllowedOrigin passes, and returns the open connection.
func dialWebWS(t *testing.T, httpClient *http.Client, ws *webserver.WebServer, sessionID, token string) *websocket.Conn {
	t.Helper()
	baseURL := ws.BaseURL()
	wsURL := strings.Replace(baseURL, "https://", "wss://", 1) + "/sessions/" + sessionID + "/ws?cap=" + token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsHeaders := http.Header{}
	wsHeaders.Set("Origin", baseURL)
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPClient: httpClient,
		HTTPHeader: wsHeaders,
	})
	if err != nil {
		t.Fatalf("dialWebWS: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// readWebFrame reads the next WebSocket binary frame from conn with a 5-second
// timeout. It skips MsgMeta and MsgPresence housekeeping frames.
func readWebFrame(t *testing.T, conn *websocket.Conn) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		_, m, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("readWebFrame: %v", err)
		}
		if len(m) > 0 && (m[0] == relay.MsgMeta || m[0] == relay.MsgPresence) {
			continue
		}
		return m
	}
}

// newWebTestHub sets up a WebServer + HubManager, creates a session hub with a
// PTY pipe, subscribes a synthetic local subscriber, and applies the given
// cols×rows host grid via ResizeClient. Returns the server, HTTP client, hub,
// and PTY write-end for test-driven output.
func newWebTestHub(t *testing.T, sessionID string, cols, rows int) (*webserver.WebServer, *http.Client, *relay.Hub, *io.PipeWriter) {
	t.Helper()
	mgr := relay.NewHubManager()
	tlsCfg, httpClient := selfSignedTLSForTest(t)
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, mgr)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	ws.SetSigningKey(ssExtTestKey)
	t.Cleanup(func() { _ = ws.Stop() })

	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })

	hub := mgr.Create(sessionID, pr, io.Discard, nil)
	ws.EnableSession(sessionID)

	// Stamp the hub's PTY grid via a synthetic local subscriber.
	localSub := &relay.Subscriber{
		Origin: "local",
		Msgs:   make(chan []byte, 64),
	}
	localSub.CloseSlow = func() {}
	hub.Subscribe(localSub)
	if err := hub.ResizeClient(localSub, cols, rows); err != nil {
		t.Fatalf("ResizeClient: %v", err)
	}

	return ws, httpClient, hub, pw
}

// TestWebJoin_PushesResizeBeforeScrollback asserts VIEW-03 on the web path:
// the first non-meta binary frame the web client receives is a 0x02 MsgResize
// carrying the hub's authoritative grid (80×24) and it arrives BEFORE the
// scrollback snapshot.
func TestWebJoin_PushesResizeBeforeScrollback(t *testing.T) {
	const sessionID = "web-join-resize-test"
	ws, httpClient, hub, pw := newWebTestHub(t, sessionID, 80, 24)

	// Write PTY data so the guest gets a non-empty scrollback to follow the resize.
	if _, err := pw.Write([]byte("scroll-data")); err != nil {
		t.Fatalf("ptyWrite: %v", err)
	}
	// Poll until the hub's scrollback is non-empty (hub drains the pipe asynchronously).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(hub.ScrollbackSnapshot()) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(hub.ScrollbackSnapshot()) == 0 {
		t.Fatal("scrollback never populated")
	}

	token := capForSession(t, ws, sessionID)
	conn := dialWebWS(t, httpClient, ws, sessionID, token)

	// First non-meta frame must be the VIEW-03 resize push: 0x02 with cols=80, rows=24.
	first := readWebFrame(t, conn)
	if len(first) != 5 {
		t.Fatalf("first non-meta frame: got %d bytes, want 5 (MsgResize)", len(first))
	}
	if first[0] != relay.MsgResize {
		t.Fatalf("first non-meta frame type: got 0x%02x, want MsgResize (0x%02x)", first[0], relay.MsgResize)
	}
	gotCols := int(uint16(first[1])<<8 | uint16(first[2]))
	gotRows := int(uint16(first[3])<<8 | uint16(first[4]))
	if gotCols != hub.Cols() || gotRows != hub.Rows() {
		t.Errorf("resize frame dims %dx%d don't match hub.Cols()/hub.Rows() %dx%d",
			gotCols, gotRows, hub.Cols(), hub.Rows())
	}
	if gotCols != 80 || gotRows != 24 {
		t.Errorf("resize frame: got %dx%d, want 80x24", gotCols, gotRows)
	}

	// The next non-meta frame must be the scrollback snapshot, proving the resize
	// arrived BEFORE the replayed bytes.
	second := readWebFrame(t, conn)
	if len(second) == 0 || second[0] != relay.MsgOutput {
		t.Errorf("second non-meta frame type: got 0x%02x, want MsgOutput (0x%02x)", second[0], relay.MsgOutput)
	}
}

// TestWebReadPump_DropsGuestResize asserts VIEW-02 / T-157-02 on the web path:
// a MsgResize2 (0x11) frame sent by a web guest must NOT change the hub's
// authoritative PTY grid.
func TestWebReadPump_DropsGuestResize(t *testing.T) {
	const sessionID = "web-drop-resize-test"
	ws, httpClient, hub, _ := newWebTestHub(t, sessionID, 100, 40)

	token := capForSession(t, ws, sessionID)
	conn := dialWebWS(t, httpClient, ws, sessionID, token)

	// Drain join-time frames (meta, presence, resize push) so the connection
	// remains unblocked and the server's read pump stays live.
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer drainCancel()
	go func() {
		for {
			_, _, err := conn.Read(drainCtx)
			if err != nil {
				return
			}
		}
	}()
	time.Sleep(150 * time.Millisecond) // let join-time frames arrive and be drained

	// Send a MsgResize2 (0x11) from the web client requesting a much larger grid.
	// Big-endian uint16: cols=200 → [0x00, 0xC8]; rows=60 → [0x00, 0x3C].
	resize2Frame := []byte{relay.MsgResize2, 0x00, 0xC8, 0x00, 0x3C}
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer sendCancel()
	if err := conn.Write(sendCtx, websocket.MessageBinary, resize2Frame); err != nil {
		t.Fatalf("send MsgResize2: %v", err)
	}

	// Give the server's read pump time to process the frame.
	time.Sleep(50 * time.Millisecond)

	// Hub grid must be unchanged — web guest resize is dropped at the call site
	// (VIEW-02 / T-157-02) in addition to the hub origin gate.
	if hub.Cols() != 100 || hub.Rows() != 40 {
		t.Errorf("hub grid changed after guest MsgResize2: got %dx%d, want 100x40",
			hub.Cols(), hub.Rows())
	}
}
