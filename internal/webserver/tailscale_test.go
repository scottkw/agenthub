package webserver

import (
	"context"
	"fmt"
	"net/netip"
	"testing"

	"tailscale.com/ipn/ipnstate"
)

func TestCheckHealth_NotRunning(t *testing.T) {
	h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, fmt.Errorf("dial unix: connection refused")
	})

	if h.Installed {
		t.Error("expected Installed=false when daemon unreachable")
	}
	if h.Connected {
		t.Error("expected Connected=false when daemon unreachable")
	}
	if h.HasCerts {
		t.Error("expected HasCerts=false when daemon unreachable")
	}
	if h.IP != "" {
		t.Errorf("expected IP=\"\" when daemon unreachable, got %q", h.IP)
	}
	if h.Domain != "" {
		t.Errorf("expected Domain=\"\" when daemon unreachable, got %q", h.Domain)
	}
}

func TestCheckHealth_BackendState(t *testing.T) {
	tests := []struct {
		state         string
		wantConnected bool
	}{
		{state: "Stopped", wantConnected: false},
		{state: "NeedsLogin", wantConnected: false},
		{state: "Starting", wantConnected: false},
		{state: "Running", wantConnected: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.state, func(t *testing.T) {
			h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
				return &ipnstate.Status{BackendState: tc.state}, nil
			})

			if !h.Installed {
				t.Errorf("state=%s: expected Installed=true when daemon reachable", tc.state)
			}
			if h.Connected != tc.wantConnected {
				t.Errorf("state=%s: expected Connected=%v, got %v", tc.state, tc.wantConnected, h.Connected)
			}
		})
	}
}

func TestCheckHealth_CertDomains(t *testing.T) {
	tests := []struct {
		name         string
		domains      []string
		wantHasCerts bool
		wantDomain   string
	}{
		{
			name:         "no cert domains",
			domains:      nil,
			wantHasCerts: false,
			wantDomain:   "",
		},
		{
			name:         "has cert domain",
			domains:      []string{"host.ts.net"},
			wantHasCerts: true,
			wantDomain:   "host.ts.net",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
				return &ipnstate.Status{
					BackendState: "Running",
					CertDomains:  tc.domains,
				}, nil
			})

			if h.HasCerts != tc.wantHasCerts {
				t.Errorf("expected HasCerts=%v, got %v", tc.wantHasCerts, h.HasCerts)
			}
			if h.Domain != tc.wantDomain {
				t.Errorf("expected Domain=%q, got %q", tc.wantDomain, h.Domain)
			}
		})
	}
}

func TestCheckHealth_FullyHealthy(t *testing.T) {
	h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
		return &ipnstate.Status{
			BackendState: "Running",
			TailscaleIPs: []netip.Addr{netip.MustParseAddr("100.64.0.1")},
			CertDomains:  []string{"myhost.tail46d69a.ts.net"},
		}, nil
	})

	if !h.Installed {
		t.Error("expected Installed=true")
	}
	if !h.Connected {
		t.Error("expected Connected=true")
	}
	if !h.HasCerts {
		t.Error("expected HasCerts=true")
	}
	if h.IP != "100.64.0.1" {
		t.Errorf("expected IP=\"100.64.0.1\", got %q", h.IP)
	}
	if h.Domain != "myhost.tail46d69a.ts.net" {
		t.Errorf("expected Domain=\"myhost.tail46d69a.ts.net\", got %q", h.Domain)
	}
}
