package webserver_test

import (
	"net"
	"testing"

	"github.com/agenthub/agenthub/internal/webserver"
)

func TestIsTailscaleIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"100.64.0.1", true},
		{"100.127.255.254", true},
		{"100.64.0.0", true},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"8.8.8.8", false},
		{"100.128.0.1", false}, // just outside the /10 range
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("failed to parse IP: %s", tt.ip)
		}
		got := webserver.IsTailscaleIP(ip)
		if got != tt.expected {
			t.Errorf("IsTailscaleIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestListInterfaces(t *testing.T) {
	ifaces, err := webserver.ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces failed: %v", err)
	}

	// Must return non-empty list on any dev machine
	if len(ifaces) == 0 {
		t.Fatal("expected at least one non-loopback interface")
	}

	for _, iface := range ifaces {
		// Each entry must have non-empty Name and IP
		if iface.Name == "" {
			t.Error("interface has empty Name")
		}
		if iface.IP == "" {
			t.Error("interface has empty IP")
		}

		// IP must be parseable
		ip := net.ParseIP(iface.IP)
		if ip == nil {
			t.Errorf("interface IP %q is not parseable", iface.IP)
			continue
		}

		// No loopback addresses should appear
		if ip.IsLoopback() {
			t.Errorf("loopback IP %s should not appear in results", iface.IP)
		}

		// No link-local addresses should appear
		if ip.IsLinkLocalUnicast() {
			t.Errorf("link-local IP %s should not appear in results", iface.IP)
		}

		// Must be IPv4 (To4 != nil)
		if ip.To4() == nil {
			t.Errorf("expected IPv4 only, got %s", iface.IP)
		}
	}
}

func TestNetworkInterfaceStruct(t *testing.T) {
	// Verify the struct has the required fields by constructing one
	iface := webserver.NetworkInterface{
		Name:        "tailscale0",
		IP:          "100.64.0.1",
		IsTailscale: true,
	}

	if iface.Name != "tailscale0" {
		t.Errorf("Name field mismatch: got %q", iface.Name)
	}
	if iface.IP != "100.64.0.1" {
		t.Errorf("IP field mismatch: got %q", iface.IP)
	}
	if !iface.IsTailscale {
		t.Error("IsTailscale field mismatch: expected true")
	}
}

func TestTailscaleIPDetectedInList(t *testing.T) {
	// This test verifies the IsTailscale field is set correctly in ListInterfaces results.
	// We can't guarantee a Tailscale interface exists on every machine,
	// but we verify that any 100.64.0.0/10 IPs in the result are flagged.
	ifaces, err := webserver.ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces failed: %v", err)
	}

	for _, iface := range ifaces {
		ip := net.ParseIP(iface.IP)
		if ip == nil {
			continue
		}
		expectedTailscale := webserver.IsTailscaleIP(ip)
		if iface.IsTailscale != expectedTailscale {
			t.Errorf("interface %s IP %s: IsTailscale=%v, want %v",
				iface.Name, iface.IP, iface.IsTailscale, expectedTailscale)
		}
	}
}
