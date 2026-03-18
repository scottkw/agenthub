package webserver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// GenerateCA generates a new ECDSA P-256 CA key and self-signed CA certificate.
// Returns the private key, parsed certificate, DER-encoded cert bytes, and any error.
func GenerateCA() (*ecdsa.PrivateKey, *x509.Certificate, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"AgentHub Local CA"}},
		NotBefore:             time.Now().Add(-time.Minute), // clock skew buffer
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, nil, err
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, nil, err
	}

	return key, cert, der, nil
}

// LoadOrCreateCA loads the CA cert and key from dir if they exist, or generates
// new ones and persists them. Returns the private key, parsed certificate, DER bytes.
func LoadOrCreateCA(dir string) (*ecdsa.PrivateKey, *x509.Certificate, []byte, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, nil, nil, err
	}

	crtPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	crtPEM, crtErr := os.ReadFile(crtPath)
	keyPEM, keyErr := os.ReadFile(keyPath)

	if crtErr == nil && keyErr == nil {
		// Load existing CA from disk
		key, err := loadECPrivateKey(keyPEM)
		if err != nil {
			return nil, nil, nil, err
		}
		cert, der, err := loadCertificate(crtPEM)
		if err != nil {
			return nil, nil, nil, err
		}
		return key, cert, der, nil
	}

	// Generate new CA
	key, cert, der, err := GenerateCA()
	if err != nil {
		return nil, nil, nil, err
	}

	// Persist CA cert as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(crtPath, certPEM, 0600); err != nil {
		return nil, nil, nil, err
	}

	// Persist CA key as PEM
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, nil, err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, privPEM, 0600); err != nil {
		return nil, nil, nil, err
	}

	return key, cert, der, nil
}

// GenerateLeafCert generates an in-memory ECDSA P-256 leaf certificate signed by the CA.
// The leaf cert has SAN IPAddresses set to bindIP. The leaf key is never written to disk.
func GenerateLeafCert(caKey *ecdsa.PrivateKey, caCert *x509.Certificate, bindIP net.IP) (tls.Certificate, error) {
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: bindIP.String()},
		IPAddresses:  []net.IP{bindIP},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// BuildTLSConfig returns a tls.Config with the provided leaf certificate and MinVersion TLS 1.2.
func BuildTLSConfig(leafCert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{leafCert},
		MinVersion:   tls.VersionTLS12,
	}
}

// ExportCACertPath returns the path to the ca.crt file in dir.
// Used to provide users with the path for OS trust store installation.
func ExportCACertPath(dir string) string {
	return filepath.Join(dir, "ca.crt")
}

// loadECPrivateKey decodes a PEM-encoded EC private key.
func loadECPrivateKey(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	return x509.ParseECPrivateKey(block.Bytes)
}

// loadCertificate decodes a PEM-encoded certificate and returns the parsed cert and DER bytes.
func loadCertificate(pemBytes []byte) (*x509.Certificate, []byte, error) {
	block, _ := pem.Decode(pemBytes)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, block.Bytes, nil
}
