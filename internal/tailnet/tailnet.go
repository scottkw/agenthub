// Package tailnet provides Tailscale peer discovery and HTTP probe functions
// for finding other AgentHub instances on the tailnet.
//
// Usage:
//
//	peers, err := tailnet.DiscoverAndProbe(ctx)
//
// This calls DiscoverPeers (via local tailscaled) and probes each online peer
// for a running AgentHub instance at DefaultProbePort.
package tailnet

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
)

// DefaultProbePort is the TCP port AgentHub listens on for HTTPS connections.
// This matches the default in SettingsPanel.tsx.
const DefaultProbePort = 7443

// Peer represents an online Tailscale peer. DNSName retains its trailing dot
// as returned by the Tailscale daemon; callers must strip it before use in URLs.
type Peer struct {
	Hostname     string   `json:"hostname"`
	DNSName      string   `json:"dnsName"`      // FQDN with trailing dot (e.g. "host.ts.net.")
	TailscaleIPs []string `json:"tailscaleIPs"` // IP addresses as strings
	OS           string   `json:"os"`
	Online       bool     `json:"online"`
}

// statusFunc is the injectable status function type for testability.
// It mirrors the pattern in internal/webserver/tailscale.go.
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

// probeFunc is the injectable probe function type for testability.
type probeFunc func(ctx context.Context, peer Peer) bool

// discoverPeers is the internal testable discovery function.
// It accepts an injected statusFn so tests can pass a fake without a live daemon.
// It returns only online peers. The returned slice is never nil (may be empty).
func discoverPeers(ctx context.Context, fn statusFunc) ([]Peer, error) {
	status, err := fn(ctx)
	if err != nil {
		return nil, err
	}
	peers := make([]Peer, 0, len(status.Peer))
	for _, p := range status.Peer {
		if !p.Online {
			continue
		}
		pr := Peer{
			Hostname: p.HostName,
			DNSName:  p.DNSName,
			OS:       p.OS,
			Online:   true,
		}
		for _, ip := range p.TailscaleIPs {
			pr.TailscaleIPs = append(pr.TailscaleIPs, ip.String())
		}
		peers = append(peers, pr)
	}
	return peers, nil
}

// DiscoverPeers queries the local tailscaled daemon for online peers.
// It returns only online peers. ctx should carry a short timeout.
func DiscoverPeers(ctx context.Context) ([]Peer, error) {
	var lc local.Client
	return discoverPeers(ctx, lc.Status)
}

// probePeer is the internal testable probe function.
// It sends a GET request to https://{peer.DNSName_stripped}:{DefaultProbePort}/api/sessions
// using the provided HTTP client, returning true if the response is 200 OK.
func probePeer(ctx context.Context, peer Peer, client *http.Client) bool {
	// DNSName ends with a dot per Tailscale spec — strip it for URL construction.
	host := strings.TrimSuffix(peer.DNSName, ".")
	url := fmt.Sprintf("https://%s:%d/api/sessions", host, DefaultProbePort)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ProbePeer probes whether the given peer is running AgentHub.
// It uses a 2-second timeout and system-CA TLS (Tailscale Let's Encrypt certs
// are publicly trusted — no skip-verify flag is used).
func ProbePeer(ctx context.Context, peer Peer) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	return probePeer(probeCtx, peer, client)
}

// probeAll probes all peers concurrently, capped at 5 goroutines via errgroup.SetLimit.
// It returns only the peers for which fn returns true.
func probeAll(ctx context.Context, peers []Peer, fn probeFunc) []Peer {
	var mu sync.Mutex
	var found []Peer

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	for _, p := range peers {
		g.Go(func() error {
			if fn(gctx, p) {
				mu.Lock()
				found = append(found, p)
				mu.Unlock()
			}
			return nil
		})
	}
	_ = g.Wait()
	return found
}

// DiscoverAndProbe discovers all online Tailscale peers and probes each one
// to determine which are running AgentHub. It returns only peers that responded
// successfully to the probe.
func DiscoverAndProbe(ctx context.Context) ([]Peer, error) {
	peers, err := DiscoverPeers(ctx)
	if err != nil {
		return nil, err
	}
	return probeAll(ctx, peers, ProbePeer), nil
}
