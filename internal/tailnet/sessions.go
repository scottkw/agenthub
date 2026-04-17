package tailnet

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
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
		log.Printf("[debug] peer session fetch failed: url=%s err=%v", url, err)
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
