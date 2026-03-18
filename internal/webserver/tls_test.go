package webserver_test

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/agenthub/agenthub/internal/webserver"
)

func TestGenerateCA(t *testing.T) {
	key, cert, der, err := webserver.GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if cert == nil {
		t.Fatal("expected non-nil cert")
	}
	if len(der) == 0 {
		t.Fatal("expected non-empty DER bytes")
	}

	// Verify it is a self-signed CA
	if !cert.IsCA {
		t.Error("expected IsCA=true")
	}
	if !cert.BasicConstraintsValid {
		t.Error("expected BasicConstraintsValid=true")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("expected KeyUsageCertSign in KeyUsage")
	}

	// Verify self-signed (issuer == subject)
	if cert.Issuer.String() != cert.Subject.String() {
		t.Errorf("expected self-signed: issuer %q != subject %q", cert.Issuer.String(), cert.Subject.String())
	}
}

func TestGenerateLeafCert(t *testing.T) {
	caKey, caCert, _, err := webserver.GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	bindIP := net.ParseIP("127.0.0.1")
	leafCert, err := webserver.GenerateLeafCert(caKey, caCert, bindIP)
	if err != nil {
		t.Fatalf("GenerateLeafCert failed: %v", err)
	}

	// Parse the leaf cert from tls.Certificate
	if len(leafCert.Certificate) == 0 {
		t.Fatal("expected at least one certificate in tls.Certificate")
	}
	parsed, err := x509.ParseCertificate(leafCert.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse leaf cert: %v", err)
	}

	// Verify SAN IPAddresses is populated
	if len(parsed.IPAddresses) == 0 {
		t.Error("expected IPAddresses SAN to be populated")
	}
	found := false
	for _, ip := range parsed.IPAddresses {
		if ip.Equal(bindIP) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected bindIP %s in IPAddresses SAN", bindIP)
	}

	// Verify ExtKeyUsage includes ServerAuth
	hasServerAuth := false
	for _, eku := range parsed.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
			break
		}
	}
	if !hasServerAuth {
		t.Error("expected ExtKeyUsageServerAuth in ExtKeyUsage")
	}

	// Verify chain: leaf cert should verify against CA
	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	opts := x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if _, err := parsed.Verify(opts); err != nil {
		t.Errorf("x509.Verify failed: %v", err)
	}
}

func TestLoadOrCreateCA(t *testing.T) {
	dir := t.TempDir()

	// First call: should create files
	key1, cert1, der1, err := webserver.LoadOrCreateCA(dir)
	if err != nil {
		t.Fatalf("first LoadOrCreateCA failed: %v", err)
	}

	// Check files were created
	crtPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")
	if _, err := os.Stat(crtPath); os.IsNotExist(err) {
		t.Errorf("expected ca.crt to be created at %s", crtPath)
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("expected ca.key to be created at %s", keyPath)
	}

	// Second call: should load from disk
	key2, cert2, der2, err := webserver.LoadOrCreateCA(dir)
	if err != nil {
		t.Fatalf("second LoadOrCreateCA failed: %v", err)
	}

	// Both calls should return identical CA cert bytes
	if len(der1) != len(der2) {
		t.Errorf("CA cert DER bytes differ: len1=%d len2=%d", len(der1), len(der2))
	}
	for i := range der1 {
		if der1[i] != der2[i] {
			t.Errorf("CA cert DER bytes differ at index %d", i)
			break
		}
	}

	// Certs should have matching serial numbers
	if cert1.SerialNumber.Cmp(cert2.SerialNumber) != 0 {
		t.Error("loaded cert serial number does not match created cert")
	}

	// Keys should match (compare public key)
	if key1.PublicKey.X.Cmp(key2.PublicKey.X) != 0 || key1.PublicKey.Y.Cmp(key2.PublicKey.Y) != 0 {
		t.Error("loaded key does not match created key")
	}

	_ = cert1
	_ = cert2
}

func TestBuildTLSConfig(t *testing.T) {
	caKey, caCert, _, err := webserver.GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	bindIP := net.ParseIP("127.0.0.1")
	leafCert, err := webserver.GenerateLeafCert(caKey, caCert, bindIP)
	if err != nil {
		t.Fatalf("GenerateLeafCert failed: %v", err)
	}

	cfg := webserver.BuildTLSConfig(leafCert)
	if cfg == nil {
		t.Fatal("expected non-nil tls.Config")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS 1.2, got %d", cfg.MinVersion)
	}
}

func TestLeafCertSANRequired(t *testing.T) {
	caKey, caCert, _, err := webserver.GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	bindIP := net.ParseIP("192.168.1.100")
	leafCert, err := webserver.GenerateLeafCert(caKey, caCert, bindIP)
	if err != nil {
		t.Fatalf("GenerateLeafCert failed: %v", err)
	}

	parsed, err := x509.ParseCertificate(leafCert.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse leaf cert: %v", err)
	}

	// SAN IPAddresses MUST be populated — without it, browsers reject the cert
	if len(parsed.IPAddresses) == 0 {
		t.Fatal("leaf cert is missing SAN IPAddresses — browser would reject with ERR_CERT_COMMON_NAME_INVALID")
	}
	found := false
	for _, ip := range parsed.IPAddresses {
		if ip.Equal(bindIP) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %s in SAN IPAddresses", bindIP)
	}
}
