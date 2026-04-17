package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/scottkw/agenthub/internal/tailnet"
)

// CLIRemoteSession is a simplified remote session representation for CLI use.
type CLIRemoteSession struct {
	ID       string // session ID (without hostname prefix)
	Name     string
	CLIType  string
	Status   string
	Hostname string // peer hostname (for display)
	FQDN     string // peer FQDN stripped of trailing dot (for URL construction)
}

// parseRemoteID splits a session identifier on the first colon character.
// If no colon is present, it returns ("", id, false).
// If a colon is found, it returns (hostname, sessionID, true).
func parseRemoteID(id string) (hostname, sessionID string, isRemote bool) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", id, false
}

// resolveRemotePeer finds a peer by hostname (case-insensitive) and returns
// the peer's FQDN with the trailing dot stripped. Returns ("", false) if not found.
func resolveRemotePeer(peers []tailnet.Peer, hostname string) (fqdn string, found bool) {
	for _, p := range peers {
		if strings.EqualFold(p.Hostname, hostname) {
			return strings.TrimSuffix(p.DNSName, "."), true
		}
	}
	return "", false
}

// resolveRemotePeerWithIPs is like resolveRemotePeer but also returns TailscaleIPs
// for IP-based fallback when DNS resolution fails.
func resolveRemotePeerWithIPs(peers []tailnet.Peer, hostname string) (fqdn string, tailscaleIPs []string, found bool) {
	for _, p := range peers {
		if strings.EqualFold(p.Hostname, hostname) {
			return strings.TrimSuffix(p.DNSName, "."), p.TailscaleIPs, true
		}
	}
	return "", nil, false
}

// fetchPeerSessions fetches /api/sessions from a single peer over HTTPS.
// It uses TLS 1.2 minimum and a 5-second timeout. Returns empty slice on any error.
// No InsecureSkipVerify — Tailscale Let's Encrypt certs are publicly trusted.
// If DNS-based fetch fails and tailscaleIPs are provided, retries using the first IP
// with TLS ServerName set to the FQDN for certificate validation.
func fetchPeerSessions(ctx context.Context, fqdn string, port int, tailscaleIPs ...string) ([]CLIRemoteSession, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	url := fmt.Sprintf("https://%s:%d/api/sessions", fqdn, port)
	sessions, err := doFetchPeerSessions(ctx, url, client)
	if len(sessions) > 0 || len(tailscaleIPs) == 0 {
		return sessions, err
	}
	// DNS or connection failure — try Tailscale IP fallback.
	ipClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: fqdn,
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	ipURL := fmt.Sprintf("https://%s:%d/api/sessions", tailscaleIPs[0], port)
	return doFetchPeerSessionsWithHost(ctx, ipURL, ipClient, fmt.Sprintf("%s:%d", fqdn, port))
}

// fetchPeerSessionsWithClient is an internal helper that allows tests to inject
// a custom HTTP client (e.g., httptest.NewTLSServer's client).
func fetchPeerSessionsWithClient(ctx context.Context, baseURL string, client *http.Client) ([]CLIRemoteSession, error) {
	url := baseURL + "/api/sessions"
	return doFetchPeerSessions(ctx, url, client)
}

// doFetchPeerSessionsWithHost performs an HTTP request with a custom Host header.
// Used for IP-fallback where TLS ServerName needs the FQDN but the URL uses an IP.
func doFetchPeerSessionsWithHost(ctx context.Context, url string, client *http.Client, host string) ([]CLIRemoteSession, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return []CLIRemoteSession{}, nil
	}
	req.Host = host
	return doFetchRequest(req, client)
}

// doFetchPeerSessions performs the actual HTTP request and JSON decode.
func doFetchPeerSessions(ctx context.Context, url string, client *http.Client) ([]CLIRemoteSession, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return []CLIRemoteSession{}, nil
	}
	return doFetchRequest(req, client)
}

// doFetchRequest executes a prepared HTTP request and decodes the session list response.
func doFetchRequest(req *http.Request, client *http.Client) ([]CLIRemoteSession, error) {
	resp, err := client.Do(req)
	if err != nil {
		return []CLIRemoteSession{}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []CLIRemoteSession{}, nil
	}
	var items []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		CLIType  string `json:"cli_type"`
		Status   string `json:"status"`
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return []CLIRemoteSession{}, nil
	}
	sessions := make([]CLIRemoteSession, 0, len(items))
	for _, item := range items {
		sessions = append(sessions, CLIRemoteSession{
			ID:      item.ID,
			Name:    item.Name,
			CLIType: item.CLIType,
			Status:  item.Status,
		})
	}
	return sessions, nil
}
