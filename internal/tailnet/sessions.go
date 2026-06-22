package tailnet

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// PeerSession describes a single session on a remote peer.
type PeerSession struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CLIType  string `json:"cli_type"`
	Status   string `json:"status"`
	Hostname string `json:"hostname"` // peer hostname (for display)
	FQDN     string `json:"fqdn"`     // peer FQDN (for URL construction)
	URL      string `json:"url"`      // pre-built session URL
}

// PeerSessionGroup groups sessions by peer hostname.
type PeerSessionGroup struct {
	Hostname string        `json:"hostname"`
	Sessions []PeerSession `json:"sessions"`
}

// FetchPeerSessions fetches /api/sessions from a single peer over HTTPS.
// Uses TLS 1.2 minimum and a 5-second timeout. Returns empty slice on any error.
// If DNS-based fetch fails and the peer has TailscaleIPs, retries using the
// first IP with TLS ServerName set to the FQDN for certificate validation.
func FetchPeerSessions(ctx context.Context, peer Peer) []PeerSession {
	fqdn := strings.TrimSuffix(peer.DNSName, ".")
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	url := fmt.Sprintf("https://%s:%d/api/sessions", fqdn, DefaultProbePort)
	sessions := doFetchSessions(ctx, url, client, "")
	// DNS or connection failure — try Tailscale IP fallback.
	if len(sessions) == 0 && len(peer.TailscaleIPs) > 0 {
		ipClient := &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					ServerName: fqdn,
					MinVersion: tls.VersionTLS12,
				},
			},
		}
		ipURL := fmt.Sprintf("https://%s:%d/api/sessions", peer.TailscaleIPs[0], DefaultProbePort)
		host := fmt.Sprintf("%s:%d", fqdn, DefaultProbePort)
		sessions = doFetchSessions(ctx, ipURL, ipClient, host)
	}
	// Enrich each session with peer context and URL.
	for i := range sessions {
		sessions[i].Hostname = peer.Hostname
		sessions[i].FQDN = fqdn
		if sessions[i].URL == "" {
			sessions[i].URL = fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, DefaultProbePort, sessions[i].ID)
		}
	}
	return sessions
}

// FetchPeerSessionsWithClient fetches sessions using an injected HTTP client.
// Used in tests to bypass TLS with httptest servers.
func FetchPeerSessionsWithClient(ctx context.Context, baseURL string, client *http.Client) []PeerSession {
	url := baseURL + "/api/sessions"
	return doFetchSessions(ctx, url, client, "")
}

// FetchAllPeerSessions discovers and fetches sessions from all given peers
// concurrently (max 5 goroutines). Returns groups sorted by hostname.
func FetchAllPeerSessions(ctx context.Context, peers []Peer) []PeerSessionGroup {
	var mu sync.Mutex
	groupMap := make(map[string][]PeerSession)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, p := range peers {
		p := p
		g.Go(func() error {
			sessions := FetchPeerSessions(gctx, p)
			if len(sessions) == 0 {
				return nil
			}
			mu.Lock()
			groupMap[p.Hostname] = append(groupMap[p.Hostname], sessions...)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	groups := make([]PeerSessionGroup, 0, len(groupMap))
	for hostname, sess := range groupMap {
		groups = append(groups, PeerSessionGroup{Hostname: hostname, Sessions: sess})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Hostname < groups[j].Hostname })
	return groups
}

// ShareableSessionMeta contains the minimum metadata needed to list and pick a
// remote session. Returned by GET /api/sessions/meta (open endpoint, no cap
// required). Contains ONLY non-sensitive metadata — never cap tokens, grants, or
// session content (RB-03 no-enumeration contract).
// Broadcast join-code fields removed: design rejected per CONTEXT.md D-10.
// Credentials are delivered out of band by the owner; this struct carries no creds.
type ShareableSessionMeta struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	CLIType string `json:"cli_type"`
	Status  string `json:"status"`
	URL     string `json:"url"`
}

// PeerSessionMetaGroup groups shareable-session metadata by peer hostname.
// Reachable discriminates between an unreachable peer (Reachable=false, Sessions=[])
// and a reachable peer with zero shareable sessions (Reachable=true, Sessions=[]).
// This nil-vs-empty distinction is the RB-04 fix — peers are never silently dropped.
type PeerSessionMetaGroup struct {
	Hostname  string                 `json:"hostname"`
	Reachable bool                   `json:"reachable"`
	Sessions  []ShareableSessionMeta `json:"sessions"`
}

// doFetchSessionsMeta performs the HTTP GET and JSON decode for /api/sessions/meta.
// If host is non-empty, it is set as the request Host header (used for IP-fallback
// TLS validation). Returns nil on any error or non-200 status (signals unreachable).
// Returns a non-nil (possibly empty) slice on 200 — the nil-vs-empty distinction
// is load-bearing: nil=unreachable, []ShareableSessionMeta{}=reachable-zero-sessions.
func doFetchSessionsMeta(ctx context.Context, url string, client *http.Client, host string) []ShareableSessionMeta {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	if host != "" {
		req.Host = host
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var items []ShareableSessionMeta
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil
	}
	// items may be an empty slice (valid: peer has zero shareable sessions).
	// Return it as-is — non-nil signals "reachable" to the caller.
	return items
}

// FetchPeerSessionsMeta fetches /api/sessions/meta from a single peer over HTTPS.
// Uses TLS 1.2 minimum and a 5-second timeout. Returns (sessions, true) on success
// and (nil, false) on any error (peer unreachable or non-200 response).
//
// Mirrors the FetchPeerSessions DNS→IP-fallback pattern: tries the DNS name first,
// then falls back to the first TailscaleIP with ServerName set to the FQDN for
// certificate validation (Pitfall 6: fresh client per IP-fallback call).
func FetchPeerSessionsMeta(ctx context.Context, peer Peer) ([]ShareableSessionMeta, bool) {
	fqdn := strings.TrimSuffix(peer.DNSName, ".")
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	url := fmt.Sprintf("https://%s:%d/api/sessions/meta", fqdn, DefaultProbePort)
	sessions := doFetchSessionsMeta(ctx, url, client, "")
	if sessions == nil && len(peer.TailscaleIPs) > 0 {
		// IP fallback — create a fresh client with ServerName set to the FQDN
		// so that certificate validation still matches the peer's hostname.
		// The TailscaleIP may include a port (e.g. in test environments); parse it.
		ipHost, ipPort, err := net.SplitHostPort(peer.TailscaleIPs[0])
		var ipURL string
		var hostHeader string
		if err != nil {
			// No embedded port — use the IP as-is with DefaultProbePort.
			ipHost = peer.TailscaleIPs[0]
			ipURL = fmt.Sprintf("https://%s:%d/api/sessions/meta", ipHost, DefaultProbePort)
			hostHeader = fmt.Sprintf("%s:%d", fqdn, DefaultProbePort)
		} else {
			// Embedded port (e.g. test environment: "127.0.0.1:PORT").
			// Use the IP and its already-known port directly; bypass DefaultProbePort.
			ipURL = fmt.Sprintf("https://%s:%s/api/sessions/meta", ipHost, ipPort)
			hostHeader = fmt.Sprintf("%s:%s", fqdn, ipPort)
		}
		ipClient := &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{ServerName: fqdn, MinVersion: tls.VersionTLS12},
			},
		}
		sessions = doFetchSessionsMeta(ctx, ipURL, ipClient, hostHeader)
	}
	if sessions == nil {
		return nil, false // unreachable
	}
	// Enrich URL if empty (defensive — the webserver sets it, but guard for stubs).
	for i := range sessions {
		if sessions[i].URL == "" {
			sessions[i].URL = fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, DefaultProbePort, sessions[i].ID)
		}
	}
	return sessions, true
}

// FetchPeerSessionsMetaWithClient fetches /api/sessions/meta using an injected
// HTTP client. Used in tests to bypass TLS with httptest servers (the caller
// provides a redirectingClient that trusts the test server's self-signed cert).
// Returns (sessions, true) on success; (nil, false) on error.
func FetchPeerSessionsMetaWithClient(ctx context.Context, baseURL string, client *http.Client) ([]ShareableSessionMeta, bool) {
	url := baseURL + "/api/sessions/meta"
	sessions := doFetchSessionsMeta(ctx, url, client, "")
	if sessions == nil {
		return nil, false
	}
	return sessions, true
}

// FetchAllPeerSessionsMeta discovers shareable-session metadata from ALL given peers
// concurrently (max 5 goroutines). Returns ALL peers in the result — even unreachable
// ones (Reachable=false, Sessions=[]) and those with zero shareable sessions
// (Reachable=true, Sessions=[]). No peer is ever silently dropped (RB-01/RB-04 fix).
// Results are sorted by Hostname for determinism.
//
// Do NOT use FetchAllPeerSessions for metadata — that function calls the cap-gated
// /api/sessions endpoint and silently drops peers with empty session lists (sessions.go:93).
func FetchAllPeerSessionsMeta(ctx context.Context, peers []Peer) []PeerSessionMetaGroup {
	var mu sync.Mutex
	groups := make([]PeerSessionMetaGroup, 0, len(peers))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, p := range peers {
		p := p
		g.Go(func() error {
			sessions, reachable := FetchPeerSessionsMeta(gctx, p)
			if sessions == nil {
				sessions = []ShareableSessionMeta{}
			}
			mu.Lock()
			groups = append(groups, PeerSessionMetaGroup{
				Hostname:  p.Hostname,
				Reachable: reachable,
				Sessions:  sessions,
			})
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	sort.Slice(groups, func(i, j int) bool { return groups[i].Hostname < groups[j].Hostname })
	return groups
}

// doFetchSessions performs the HTTP request and JSON decode. If host is non-empty,
// it is set as the request Host header (used for IP-fallback TLS validation).
func doFetchSessions(ctx context.Context, url string, client *http.Client, host string) []PeerSession {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	if host != "" {
		req.Host = host
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var items []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		CLIType  string `json:"cli_type"`
		Status   string `json:"status"`
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil
	}
	sessions := make([]PeerSession, 0, len(items))
	for _, item := range items {
		sessions = append(sessions, PeerSession{
			ID:      item.ID,
			Name:    item.Name,
			CLIType: item.CLIType,
			Status:  item.Status,
		})
	}
	return sessions
}
