package webserver

import (
	"context"
	"runtime"

	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
)

// TailscaleHealth is the result of a health check against the local tailscaled daemon.
// Serialised to JSON and returned to the Wails frontend via GetTailscaleStatus().
type TailscaleHealth struct {
	Installed    bool   `json:"installed"`    // daemon socket reachable (legacy compat: true when DaemonUp)
	Connected    bool   `json:"connected"`    // BackendState == "Running"
	HasCerts     bool   `json:"hasCerts"`     // len(CertDomains) > 0
	IP           string `json:"ip"`           // first TailscaleIP as string, empty if not connected
	Domain       string `json:"domain"`       // first CertDomain e.g. hostname.ts.net, empty if none
	BinaryFound  bool   `json:"binaryFound"`  // tailscale binary exists on disk
	DaemonUp     bool   `json:"daemonUp"`     // tailscaled daemon socket reachable
	PlatformHint string `json:"platformHint"` // runtime.GOOS value for frontend platform-specific instructions
}

// statusFunc is the injectable status function type for testability.
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

// checkHealth is the internal testable health check function.
// It accepts an injected statusFn so tests can pass a fake without a live daemon.
// The 4-state cascade: binary detection -> daemon probe -> connection state -> certs.
func checkHealth(ctx context.Context, fn statusFunc, customPath string) TailscaleHealth {
	h := TailscaleHealth{PlatformHint: runtime.GOOS}

	// Step 1: Binary detection — custom path -> well-known -> PATH
	binary := detectTailscaleBinary(customPath)
	if binary == "" {
		return h // BinaryFound=false, DaemonUp=false, Installed=false, Connected=false
	}
	h.BinaryFound = true

	// Step 2: Daemon socket probe
	status, err := fn(ctx)
	if err != nil {
		return h // BinaryFound=true, DaemonUp=false, Installed=false, Connected=false
	}
	h.DaemonUp = true
	h.Installed = true // legacy compat: Installed now means daemon reachable

	// Step 3: Connection state
	h.Connected = status.BackendState == "Running"

	// Step 4: Certs (only if connected)
	if h.Connected {
		h.HasCerts = len(status.CertDomains) > 0
		if len(status.TailscaleIPs) > 0 {
			h.IP = status.TailscaleIPs[0].String()
		}
		if len(status.CertDomains) > 0 {
			h.Domain = status.CertDomains[0]
		}
	}
	return h
}

// CheckHealth queries tailscaled via local.Client and returns TailscaleHealth.
// ctx should carry a short timeout (3-5 seconds) to prevent UI hangs.
func CheckHealth(ctx context.Context) TailscaleHealth {
	var lc local.Client
	return checkHealth(ctx, lc.StatusWithoutPeers, "")
}

// CheckHealthWithCustomPath queries tailscaled and returns TailscaleHealth,
// checking customPath first for the tailscale binary before well-known paths.
func CheckHealthWithCustomPath(ctx context.Context, customPath string) TailscaleHealth {
	var lc local.Client
	return checkHealth(ctx, lc.StatusWithoutPeers, customPath)
}
