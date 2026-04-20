//go:build phase87_wave2

// Package webserver test helpers relocated from security-review/ for Phase 87.
//
// These helpers are gated behind the phase87_wave2 build tag so they do NOT
// compile until Plan 03 wires the capability package into WebServer (adds
// signingKey, joinCodes, grants fields and the requireCapability middleware).
// Until then `go test ./internal/webserver/...` skips this file entirely.
//
// Helpers provided:
//   - selfSignedTLSForTest: in-memory self-signed CA + leaf for 127.0.0.1
//   - testServer:           webserver.WebServer in tailnet-equivalent test mode
//   - testServerWithHub:    WebServer + live relay.Hub backed by pipes
//   - dialWebServerWS:      dial a wss:// WebSocket against the test server
//   - readPipeWithTimeout:  read from an io.PipeReader with a deadline
//
// Source: security-review/internal_webserver_server_test.go:29-198 (copied
// verbatim aside from adjustments required by the internal-package position:
// this file is in package webserver, so bare identifiers replace the
// webserver.* prefixes used by the external webserver_test package).
package webserver

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/relay"
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
func testServer(t *testing.T) (*WebServer, *http.Client) {
	t.Helper()
	manager := relay.NewHubManager()
	tlsCfg, client := selfSignedTLSForTest(t)
	cfg := Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		TLSConfig: tlsCfg,
	}
	ws, err := NewWebServer(cfg, manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	t.Cleanup(func() { _ = ws.Stop() })
	return ws, client
}

// testServerWithHub creates a tailscale-mode WebServer plus a live relay hub
// backed by pipes so tests can validate end-to-end browser->PTY behavior.
// Returns the server, the HTTP client, a writer to feed output into the hub's
// PTY-output stream, and a reader to observe input captured from the hub.
func testServerWithHub(t *testing.T, sessionID string) (*WebServer, *http.Client, *io.PipeWriter, *io.PipeReader) {
	t.Helper()

	manager := relay.NewHubManager()
	ptyOutputR, ptyOutputW := io.Pipe()
	inputCaptureR, inputCaptureW := io.Pipe()
	manager.Create(sessionID, ptyOutputR, inputCaptureW, nil)

	tlsCfg, client := selfSignedTLSForTest(t)
	cfg := Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		Mode:      "tailscale",
		TLSConfig: tlsCfg,
	}
	ws, err := NewWebServer(cfg, manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	ws.EnableSession(sessionID)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == sessionID {
			return "Security Review Session", "codex", "running", "review-host"
		}
		return id, "", "", ""
	})

	t.Cleanup(func() {
		_ = ws.Stop()
		manager.Shutdown()
		_ = ptyOutputW.Close()
		_ = inputCaptureW.Close()
	})

	return ws, client, ptyOutputW, inputCaptureR
}

// readPipeWithTimeout reads the next chunk of bytes from r with the given
// timeout. If no bytes arrive within the timeout, t.Fatalf is called. This
// is the primary mechanism by which SEC-05 tests assert that MsgInput
// frames do NOT reach the PTY pipe (timeout = "input was blocked").
func readPipeWithTimeout(t *testing.T, r *io.PipeReader, timeout time.Duration) []byte {
	t.Helper()

	readDone := make(chan []byte, 1)
	readErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			readErr <- err
			return
		}
		readDone <- buf[:n]
	}()

	select {
	case data := <-readDone:
		return data
	case err := <-readErr:
		t.Fatalf("pipe read: %v", err)
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for pipe data")
	}
	return nil
}

// dialWebServerWS dials a wss:// WebSocket against baseURL+path using the
// supplied HTTP client (for TLS trust) and optional headers (e.g. Origin
// overrides). The returned connection is closed in t.Cleanup.
func dialWebServerWS(t *testing.T, httpClient *http.Client, baseURL, path string, headers http.Header) *websocket.Conn {
	t.Helper()

	wsURL := "wss" + strings.TrimPrefix(baseURL, "https") + path
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPClient: httpClient,
		HTTPHeader: headers,
	})
	if err != nil {
		t.Fatalf("websocket.Dial(%s): %v", wsURL, err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}
