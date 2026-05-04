//go:build playwrightfixture

// Package main is a Playwright e2e test fixture for Phase 93 web parity specs.
//
// This binary boots an in-process AgentHub web server with:
//   - Self-signed TLS on 127.0.0.1 (random port)
//   - One pre-seeded session ("playwright-test-session") backed by an io.Pipe
//     PTY stub (no real shell — the page only needs to attach for the addon
//     load assertions; PTY data is irrelevant).
//   - A capability token signed with a fixed test key.
//   - SetPluginSettingsProvider wired to an atomic.Value holding the latest
//     plugin settings JSON, so /api/plugin-config returns mutable state.
//   - A separate plain-HTTP admin listener on 127.0.0.1 (random port) exposing
//     POST /__test__/plugin-config for the SSE hot-swap spec to flip server-
//     side settings + trigger BroadcastPluginConfig().
//
// Build-tagged 'playwrightfixture' so it never compiles into release builds
// (T-93-iPad-UAT-EVASION mitigation: the /__test__ surface is fixture-only).
//
// Usage (from frontend/playwright.config.ts webServer):
//
//	go run -tags=playwrightfixture ./cmd/playwright-fixture
//
// On startup the binary writes one line to stdout in KEY=VALUE form (then
// continues running until SIGTERM):
//
//	BASE_URL=https://127.0.0.1:PORT
//	CAP=<token>
//	ADMIN_URL=http://127.0.0.1:ADMIN_PORT
//	READY=1
//
// The Playwright config's webServer.url field is used to wait for readiness.
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/webserver"
)

const (
	sessionID    = "playwright-test-session"
	sessionName  = "Playwright Test Session"
	sessionAgent = "claude"
	sessionHost  = "fixture-host"
)

// fixedTestKey is a 32-byte deterministic HMAC signing key for the fixture.
// Tokens minted with this key are valid only against this fixture's webserver
// (because the webserver's SetSigningKey is called with the same value).
// Never used in production — gated by the playwrightfixture build tag.
var fixedTestKey = func() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}()

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("[playwright-fixture] ")

	// 1. Self-signed TLS for 127.0.0.1.
	tlsCfg, err := selfSignedTLS()
	if err != nil {
		log.Fatalf("selfSignedTLS: %v", err)
	}

	// 2. Hub manager + one pre-seeded session backed by io.Pipe stub.
	manager := relay.NewHubManager()
	ptyOutR, ptyOutW := io.Pipe()
	inputCaptureR, inputCaptureW := io.Pipe()
	_ = inputCaptureR
	manager.Create(sessionID, ptyOutR, inputCaptureW, nil)

	// 3. WebServer in tailscale-mode with TLSConfig override (mirrors
	//    testServerWithHub but without *testing.T).
	cfg := webserver.Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		Mode:      "tailscale",
		TLSConfig: tlsCfg,
	}
	ws, err := webserver.NewWebServer(cfg, manager)
	if err != nil {
		log.Fatalf("NewWebServer: %v", err)
	}
	ws.SetSigningKey(fixedTestKey)
	ws.EnableSession(sessionID)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == sessionID {
			return sessionName, sessionAgent, "running", sessionHost
		}
		return id, "", "", ""
	})

	// 4. Plugin settings: atomic.Value holding pre-marshaled JSON. Default
	//    is everything-on so the vendor-parity spec sees all addons load.
	var pluginJSON atomic.Value
	pluginJSON.Store(mustMarshal(defaultPluginSettings()))
	ws.SetPluginSettingsProvider(func() []byte {
		v := pluginJSON.Load()
		if v == nil {
			return nil
		}
		return v.([]byte)
	})

	// 5. Start the WebServer's HTTPS listener.
	if err := ws.Start(); err != nil {
		log.Fatalf("ws.Start: %v", err)
	}

	// 6. Mint a read,write capability for the test session.
	claims := capability.Claims{
		SID:     sessionID,
		Perms:   "read,write",
		IAT:     time.Now().Unix(),
		GrantID: "grant-playwright-fixture",
		V:       1,
	}
	token, err := capability.Sign(claims, fixedTestKey)
	if err != nil {
		log.Fatalf("capability.Sign: %v", err)
	}
	ws.AddGrant(sessionID, claims.GrantID)

	// 7. Admin HTTP server (plain, separate port) for /__test__/plugin-config.
	adminLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("admin listen: %v", err)
	}
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/__test__/plugin-config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var settings map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, "decode body: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Re-marshal to ensure canonical key set is present and well-formed.
		data, err := json.Marshal(settings)
		if err != nil {
			http.Error(w, "marshal: "+err.Error(), http.StatusInternalServerError)
			return
		}
		pluginJSON.Store(data)
		// Trigger SSE broadcast so any subscribed client receives the new frame.
		ws.BroadcastPluginConfig(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
	adminMux.HandleFunc("/__test__/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	adminSrv := &http.Server{
		Handler:           adminMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := adminSrv.Serve(adminLn); err != nil && err != http.ErrServerClosed {
			log.Printf("admin Serve: %v", err)
		}
	}()

	// 8. Print the URLs/cap and READY signal so playwright.config.ts can
	//    parse stdout and pass the values via env vars to specs.
	baseURL := "https://" + ws.Addr()
	adminURL := "http://" + adminLn.Addr().String()
	fmt.Printf("BASE_URL=%s\n", baseURL)
	fmt.Printf("CAP=%s\n", token)
	fmt.Printf("ADMIN_URL=%s\n", adminURL)
	fmt.Println("READY=1")

	// 9. Wait for SIGTERM/SIGINT, then clean up.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	log.Println("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = adminSrv.Shutdown(shutdownCtx)
	_ = ws.Stop()
	manager.Shutdown()
	_ = ptyOutW.Close()
	_ = inputCaptureW.Close()
}

func defaultPluginSettings() map[string]bool {
	return map[string]bool{
		"webgl":     true,
		"unicode11": true,
		"clipboard": true,
		"search":    true,
		"webLinks":  true,
		"image":     true,
		"serialize": true,
		"progress":  false,
	}
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// selfSignedTLS generates an in-memory self-signed CA + leaf for 127.0.0.1.
// Mirrors the helper used by `testServerWithHub` in the non-test package.
func selfSignedTLS() (*tls.Config, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Playwright Fixture CA"}},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(2 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, err
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(2 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	leafCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		return nil, err
	}
	leafKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})
	tlsCert, err := tls.X509KeyPair(leafCertPEM, leafKeyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
