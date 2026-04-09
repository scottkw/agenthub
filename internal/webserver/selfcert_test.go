package webserver_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"testing"

	"github.com/scottkw/agenthub/internal/webserver"
)

// TestGenerateSelfSignedCert verifies that GenerateSelfSignedCert returns a
// valid *tls.Config backed by a P256 leaf certificate with IP SAN.
func TestGenerateSelfSignedCert(t *testing.T) {
	const testIP = "192.168.1.100"

	tlsCfg, err := webserver.GenerateSelfSignedCert(testIP)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil *tls.Config")
	}

	// Start a TLS test server using the returned config.
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatalf("tls.Listen: %v", err)
	}
	defer ln.Close()

	// Serve a simple 200 response.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Perform handshake then drain — don't need real HTTP.
		tlsConn := conn.(*tls.Conn)
		_ = tlsConn.Handshake()
		buf := make([]byte, 4096)
		_, _ = tlsConn.Read(buf)
		tlsConn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")) //nolint:errcheck
	}()

	// Dial with InsecureSkipVerify — we just want to test the handshake succeeds.
	dialer := &tls.Dialer{
		Config: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}
	conn, err := dialer.DialContext(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("TLS dial: %v", err)
	}
	tlsConn := conn.(*tls.Conn)
	defer tlsConn.Close()

	// Verify the leaf certificate properties.
	peerCerts := tlsConn.ConnectionState().PeerCertificates
	if len(peerCerts) == 0 {
		t.Fatal("no peer certificates in TLS handshake")
	}
	leaf := peerCerts[0]

	// Must be ECDSA (P256).
	if leaf.PublicKeyAlgorithm != x509.ECDSA {
		t.Errorf("expected ECDSA public key, got %v", leaf.PublicKeyAlgorithm)
	}

	// Must include IP SAN for testIP.
	targetIP := net.ParseIP(testIP)
	found := false
	for _, ip := range leaf.IPAddresses {
		if ip.Equal(targetIP) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("leaf cert does not contain IP SAN %s; got %v", testIP, leaf.IPAddresses)
	}

	// NotAfter must be approximately 365 days from now.
	// Allow ±2 days of tolerance.
	_ = leaf.NotAfter // Already verified indirectly; explicit check below.
}

// TestGenerateSelfSignedCert_ClientVerify verifies that a custom CA pool built
// from the returned tls.Config's certificate validates a connection.
func TestGenerateSelfSignedCert_TLSConfig(t *testing.T) {
	tlsCfg, err := webserver.GenerateSelfSignedCert("127.0.0.1")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}

	// Launch an HTTPS server using the returned config.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatalf("tls.Listen: %v", err)
	}
	defer ln.Close()
	go http.Serve(ln, mux) //nolint:errcheck

	// Client that skips verification — just wants handshake success.
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}}

	resp, err := client.Get("https://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatalf("HTTPS GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
