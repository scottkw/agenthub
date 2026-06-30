// Package webserver — injectable funnelClient interface seam (Phase 165, FNL-02).
//
// Mirrors the statusFunc/prefsFunc injection idiom in tailscale.go:28-34.
// Production: ws.funnelClient = &ws.lc (concrete *local.Client set by NewWebServer).
// Tests:       ws.funnelClient = &fakeFunnelClient{...} (funnel_test.go).
package webserver

import (
	"context"

	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
)

// funnelClient is the narrow injectable interface covering the three
// local.Client methods needed by EnableFunnel and DisableFunnel.
// This seam allows CI tests to exercise the full EnableFunnel body
// without a live tailscaled daemon.
type funnelClient interface {
	// GetServeConfig returns the current serve config (nil, nil when unconfigured).
	// The returned struct carries an ETag that SetServeConfig sends as If-Match
	// for optimistic concurrency — always call GetServeConfig before SetServeConfig
	// (Pitfall 3 / T-165-04).
	GetServeConfig(ctx context.Context) (*ipn.ServeConfig, error)

	// SetServeConfig atomically replaces the serve config.
	// Pass nil to clear all serve settings (ClearLingeringFunnel use-case).
	SetServeConfig(ctx context.Context, config *ipn.ServeConfig) error

	// StatusWithoutPeers returns the local node's status without peer details.
	// Used for CheckFunnelAccess prerequisite checks before acquiring ws.mu
	// (blocking Unix-socket call — Anti-Pattern 5 / T-165-03).
	StatusWithoutPeers(ctx context.Context) (*ipnstate.Status, error)
}

// FunnelClientForTest is an exported type alias for the funnelClient interface
// so the daemon test package (internal/daemon/funnel_test.go) can create fake
// implementations without importing unexported types (Pitfall 13 — cross-package seam).
// Mirrors the SetWebServerForTest pattern in daemon/api.go:366.
type FunnelClientForTest = funnelClient

// SetFunnelClientForTest injects a test double for the funnelClient seam.
// Must be called before EnableFunnel/DisableFunnel so tests use the fake
// instead of the production local.Client. The ForTest suffix matches
// SetWebServerForTest in daemon/api.go.
func (ws *WebServer) SetFunnelClientForTest(fc FunnelClientForTest) {
	ws.mu.Lock()
	ws.funnelClient = fc
	ws.mu.Unlock()
}
