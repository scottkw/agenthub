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

	"github.com/agenthub/agenthub/internal/relay"
	"github.com/agenthub/agenthub/internal/webserver"
	"github.com/coder/websocket"
)

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
	baseURL := ws.BaseURL()

	ws.EnableSession("sess1")

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
	defer ws.Stop()

	// Create a hub with a pipe so we can send output
	pr, pw := io.Pipe()
	hub := mgr.Create("sess1", pr, io.Discard, nil)
	_ = hub
	ws.EnableSession("sess1")

	baseURL := ws.BaseURL()
	wsURL := strings.Replace(baseURL, "https://", "wss://", 1) + "/sessions/sess1/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPClient: client,
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
	ws, client := testServer(t)
	baseURL := ws.BaseURL()

	ws.EnableSession("sess1")
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

func TestSessionAccessWithoutAuth(t *testing.T) {
	ws, _ := testServer(t)
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
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for web-enabled session without auth, got %d", resp.StatusCode)
	}
}
