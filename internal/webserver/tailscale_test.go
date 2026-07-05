package webserver

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tailscale.com/ipn/ipnstate"
)

// stubBinary creates a fake tailscale binary in a temp dir and returns its path.
func stubBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	stubPath := filepath.Join(dir, "tailscale")
	if err := os.WriteFile(stubPath, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatalf("writing stub binary: %v", err)
	}
	return stubPath
}

func TestCheckHealth_NotRunning(t *testing.T) {
	stub := stubBinary(t)
	h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, fmt.Errorf("dial unix: connection refused")
	}, stub, nil, nil)

	// Binary found but daemon unreachable
	if !h.BinaryFound {
		t.Error("expected BinaryFound=true when stub binary exists")
	}
	if h.DaemonUp {
		t.Error("expected DaemonUp=false when daemon unreachable")
	}
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
	if h.PlatformHint != runtime.GOOS {
		t.Errorf("expected PlatformHint=%q, got %q", runtime.GOOS, h.PlatformHint)
	}
}

func TestCheckHealth_BackendState(t *testing.T) {
	stub := stubBinary(t)
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
			}, stub, nil, nil)

			if !h.BinaryFound {
				t.Errorf("state=%s: expected BinaryFound=true", tc.state)
			}
			if !h.DaemonUp {
				t.Errorf("state=%s: expected DaemonUp=true when daemon reachable", tc.state)
			}
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
	stub := stubBinary(t)
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
			}, stub, nil, nil)

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
	stub := stubBinary(t)
	h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
		return &ipnstate.Status{
			BackendState: "Running",
			TailscaleIPs: []netip.Addr{netip.MustParseAddr("100.64.0.1")},
			CertDomains:  []string{"myhost.tail46d69a.ts.net"},
		}, nil
	}, stub, nil, nil)

	if !h.BinaryFound {
		t.Error("expected BinaryFound=true")
	}
	if !h.DaemonUp {
		t.Error("expected DaemonUp=true")
	}
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
	if h.PlatformHint != runtime.GOOS {
		t.Errorf("expected PlatformHint=%q, got %q", runtime.GOOS, h.PlatformHint)
	}
}

func TestCheckHealth_BinaryNotFound(t *testing.T) {
	// Skip if tailscale is actually installed on this machine, since
	// detectTailscaleBinary will find it via well-known paths or PATH
	// even when customPath is invalid.
	if detectTailscaleBinary("") != "" {
		t.Skip("tailscale binary found on host; cannot test not-found cascade")
	}

	fnCalled := false
	h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
		fnCalled = true
		return nil, nil
	}, "/nonexistent/path/tailscale", nil, nil)

	if fnCalled {
		t.Error("statusFunc should not be called when binary is not found")
	}
	if h.BinaryFound {
		t.Error("expected BinaryFound=false")
	}
	if h.DaemonUp {
		t.Error("expected DaemonUp=false")
	}
	if h.Installed {
		t.Error("expected Installed=false")
	}
	if h.Connected {
		t.Error("expected Connected=false")
	}
	if h.PlatformHint != runtime.GOOS {
		t.Errorf("expected PlatformHint=%q, got %q", runtime.GOOS, h.PlatformHint)
	}
}

func TestCheckHealth_DaemonStopped(t *testing.T) {
	stub := stubBinary(t)
	h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, fmt.Errorf("connect: connection refused")
	}, stub, nil, nil)

	if !h.BinaryFound {
		t.Error("expected BinaryFound=true")
	}
	if h.DaemonUp {
		t.Error("expected DaemonUp=false when daemon stopped")
	}
	if h.Installed {
		t.Error("expected Installed=false when daemon stopped")
	}
	if h.Connected {
		t.Error("expected Connected=false when daemon stopped")
	}
}

func TestCheckHealth_CustomPathPrecedence(t *testing.T) {
	// Create two stub binaries at different paths
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	customPath := filepath.Join(dir1, "tailscale")
	otherPath := filepath.Join(dir2, "tailscale")
	os.WriteFile(customPath, []byte("#!/bin/sh\necho custom\n"), 0755)
	os.WriteFile(otherPath, []byte("#!/bin/sh\necho other\n"), 0755)

	// Pass customPath — health check should succeed because binary exists at custom path
	h := checkHealth(context.Background(), func(ctx context.Context) (*ipnstate.Status, error) {
		return &ipnstate.Status{BackendState: "Running"}, nil
	}, customPath, nil, nil)

	if !h.BinaryFound {
		t.Error("expected BinaryFound=true with custom path")
	}
	if !h.DaemonUp {
		t.Error("expected DaemonUp=true")
	}
	if !h.Connected {
		t.Error("expected Connected=true")
	}
}

func TestDetectTailscaleBinary_CustomPath(t *testing.T) {
	stub := stubBinary(t)
	got := detectTailscaleBinary(stub)
	if got != stub {
		t.Errorf("expected detectTailscaleBinary(%q)=%q, got %q", stub, stub, got)
	}
}

func TestDetectTailscaleBinary_InvalidCustomPath(t *testing.T) {
	// Invalid custom path should not panic, should fall through to well-known/PATH
	got := detectTailscaleBinary("/nonexistent/path/tailscale")
	// On CI or machines without tailscale, this may return "" or a real path if
	// tailscale is installed. Just verify no panic and result is a string.
	_ = got
}

func TestDetectTailscaleBinary_Empty(t *testing.T) {
	// Empty custom path should not panic. On machines without tailscale installed
	// at well-known paths, this may still find tailscale on PATH.
	got := detectTailscaleBinary("")
	// Just verify no panic and result is a string.
	_ = got
}

// TestCheckHealth_AcceptDNS covers DNS-03: the AcceptDNS field in TailscaleHealth
// must reflect the node's current accept-dns setting (CorpDNS from Tailscale prefs).
//
// DNS-03: When the node is Connected and the Tailscale prefs report accept-dns
// disabled (CorpDNS=false), TailscaleHealth.AcceptDNS must be false. When prefs
// report CorpDNS=true, AcceptDNS must be true. When the prefs probe returns an
// error, AcceptDNS must be false and must not panic.
//
// This test is written against the SHAPE that Plan 03 will deliver:
//   - checkHealth accepts a prefsFunc injectable (mirroring the statusFunc injection idiom)
//   - prefsFunc returns (CorpDNS bool, error)
//   - TailscaleHealth gains an AcceptDNS bool field
//
// The test intentionally fails to compile until Plan 03 adds:
//  1. type prefsFunc (or equivalent) to tailscale.go
//  2. AcceptDNS bool to TailscaleHealth
//  3. an updated checkHealth signature that accepts the prefs probe
//
// RED signal: this file will not compile until the above changes land.
func TestCheckHealth_AcceptDNS(t *testing.T) {
	stub := stubBinary(t)

	tests := []struct {
		name          string
		corpDNS       bool
		prefsErr      error
		wantAcceptDNS bool
	}{
		{
			name:          "connected and CorpDNS=false → AcceptDNS=false",
			corpDNS:       false,
			prefsErr:      nil,
			wantAcceptDNS: false,
		},
		{
			name:          "connected and CorpDNS=true → AcceptDNS=true",
			corpDNS:       true,
			prefsErr:      nil,
			wantAcceptDNS: true,
		},
		{
			name:          "prefs probe returns error → AcceptDNS=false, no panic",
			corpDNS:       false,
			prefsErr:      fmt.Errorf("localapi prefs unavailable"),
			wantAcceptDNS: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			statusFn := func(ctx context.Context) (*ipnstate.Status, error) {
				return &ipnstate.Status{BackendState: "Running"}, nil
			}
			// prefsFn mirrors the statusFunc injection idiom: an injectable function
			// returning a CorpDNS bool and an error. Plan 03 will add this parameter
			// to checkHealth. Until then, the compile error is the intended RED signal.
			prefsFn := func(ctx context.Context) (bool, error) {
				return tc.corpDNS, tc.prefsErr
			}
			// NOTE: checkHealth does not yet accept a prefsFunc — this will not compile
			// until Plan 03 adds the parameter. That compile failure IS the RED signal.
			h := checkHealth(context.Background(), statusFn, stub, prefsFn, nil)
			if h.AcceptDNS != tc.wantAcceptDNS {
				t.Errorf("AcceptDNS = %v; want %v", h.AcceptDNS, tc.wantAcceptDNS)
			}
		})
	}
}

// TestCheckHealth_PermissionLimited covers SC1 (FIX-05, Issue #120): when the
// SDK statusFunc fails (as it does when the macsys sameuserproof file is
// root:admin 0640 and unreadable by a non-admin account) AND the injected
// perm probe reports the macsys-permission-limited condition, checkHealth
// must report PermissionLimited=true while NEVER setting Connected=true
// (hard SC — no false Connected). HasCerts and IP must also stay at their
// zero values since the underlying status is genuinely unknown.
func TestCheckHealth_PermissionLimited(t *testing.T) {
	stub := stubBinary(t)

	statusFn := func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, fmt.Errorf("dial unix: connection refused")
	}
	permFn := func(ctx context.Context) bool {
		return true
	}

	h := checkHealth(context.Background(), statusFn, stub, nil, permFn)

	if !h.BinaryFound {
		t.Error("expected BinaryFound=true")
	}
	if !h.PermissionLimited {
		t.Error("expected PermissionLimited=true when the perm probe reports macsys permission limitation")
	}
	if h.Connected {
		t.Error("SC violation: expected Connected=false even when PermissionLimited=true (no false Connected)")
	}
	if h.HasCerts {
		t.Error("expected HasCerts=false in the permission-limited state")
	}
	if h.IP != "" {
		t.Errorf("expected IP=\"\" in the permission-limited state, got %q", h.IP)
	}
}

// TestCheckHealth_DaemonDown_NotPermissionLimited covers D-02: a genuinely-down
// daemon (no macsys signature / dead liveness dial) must NOT be flagged
// permission-limited — the injected perm probe returning false preserves
// today's not-connected behavior.
func TestCheckHealth_DaemonDown_NotPermissionLimited(t *testing.T) {
	stub := stubBinary(t)

	statusFn := func(ctx context.Context) (*ipnstate.Status, error) {
		return nil, fmt.Errorf("connect: connection refused")
	}
	permFn := func(ctx context.Context) bool {
		return false
	}

	h := checkHealth(context.Background(), statusFn, stub, nil, permFn)

	if h.PermissionLimited {
		t.Error("expected PermissionLimited=false when the perm probe reports no macsys permission limitation")
	}
	if h.DaemonUp {
		t.Error("expected DaemonUp=false when daemon is genuinely down")
	}
	if h.Connected {
		t.Error("expected Connected=false when daemon is genuinely down")
	}
}
