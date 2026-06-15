package webserver_test

import (
	"net"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/webserver"
)

// TestGetLANIP verifies that GetLANIP returns a non-empty non-loopback IPv4.
func TestGetLANIP(t *testing.T) {
	ip, err := webserver.GetLANIP()
	if err != nil {
		// On CI runners with no network interfaces this may fail; treat as skip.
		t.Skipf("GetLANIP returned error (acceptable in restricted env): %v", err)
	}
	if ip == "" {
		t.Fatal("expected non-empty IP")
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		t.Fatalf("returned value %q is not a valid IP address", ip)
	}
	// Must not be loopback.
	if strings.HasPrefix(ip, "127.") {
		t.Errorf("expected non-loopback IP, got %q", ip)
	}
	if parsed.IsLoopback() {
		t.Errorf("expected non-loopback IP, got %q", ip)
	}
	// Must be valid IPv4.
	if parsed.To4() == nil {
		t.Errorf("expected IPv4 address, got %q", ip)
	}
}

// TestGetLANIP_ExcludesTailscale verifies that isTailscaleIP correctly
// identifies the Tailscale CGNAT range (100.64.0.0/10).
func TestGetLANIP_ExcludesTailscale(t *testing.T) {
	tests := []struct {
		ip       string
		isTScale bool
	}{
		{"100.64.0.1", true},      // first address in range
		{"100.100.100.100", true}, // typical Tailscale IP
		{"100.127.255.255", true}, // last address in range
		{"100.63.0.1", false},     // just below the range
		{"100.128.0.1", false},    // just above the range
		{"192.168.1.1", false},    // private but not Tailscale
		{"10.0.0.1", false},       // private but not Tailscale
		{"172.16.0.1", false},     // private but not Tailscale
		{"8.8.8.8", false},        // public IP
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("invalid test IP %q", tt.ip)
		}
		got := webserver.IsTailscaleIP(ip)
		if got != tt.isTScale {
			t.Errorf("IsTailscaleIP(%q) = %v, want %v", tt.ip, got, tt.isTScale)
		}
	}
}
