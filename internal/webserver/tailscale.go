package webserver

import (
	"context"

	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
)

// TailscaleHealth is the result of a health check against the local tailscaled daemon.
// Serialised to JSON and returned to the Wails frontend via GetTailscaleStatus().
type TailscaleHealth struct {
	Installed bool   `json:"installed"` // tailscaled socket reachable
	Connected bool   `json:"connected"` // BackendState == "Running"
	HasCerts  bool   `json:"hasCerts"`  // len(CertDomains) > 0
	IP        string `json:"ip"`        // first TailscaleIP as string, empty if not connected
	Domain    string `json:"domain"`    // first CertDomain e.g. hostname.ts.net, empty if none
}

// statusFunc is the injectable status function type for testability.
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

// checkHealth is the internal testable health check function.
// It accepts an injected statusFn so tests can pass a fake without a live daemon.
func checkHealth(ctx context.Context, fn statusFunc) TailscaleHealth {
	status, err := fn(ctx)
	if err != nil {
		return TailscaleHealth{Installed: false}
	}
	h := TailscaleHealth{Installed: true}
	h.Connected = status.BackendState == "Running"
	h.HasCerts = len(status.CertDomains) > 0
	if len(status.TailscaleIPs) > 0 {
		h.IP = status.TailscaleIPs[0].String()
	}
	if len(status.CertDomains) > 0 {
		h.Domain = status.CertDomains[0]
	}
	return h
}

// CheckHealth queries tailscaled via local.Client and returns TailscaleHealth.
// ctx should carry a short timeout (3-5 seconds) to prevent UI hangs.
func CheckHealth(ctx context.Context) TailscaleHealth {
	var lc local.Client
	return checkHealth(ctx, lc.StatusWithoutPeers)
}
