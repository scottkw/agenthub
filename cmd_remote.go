package main

import (
	"context"
	"net/http"
	"strings"

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

// fetchPeerSessions fetches sessions from a single peer. Delegates to
// tailnet.FetchPeerSessions for HTTP fetch with IP fallback.
func fetchPeerSessions(ctx context.Context, peer tailnet.Peer) []CLIRemoteSession {
	sessions := tailnet.FetchPeerSessions(ctx, peer)
	return peerSessionsToCLI(sessions)
}

// fetchPeerSessionsWithClient fetches sessions using an injected HTTP client.
// Used in tests to bypass TLS with httptest servers.
func fetchPeerSessionsWithClient(ctx context.Context, baseURL string, client *http.Client) []CLIRemoteSession {
	sessions := tailnet.FetchPeerSessionsWithClient(ctx, baseURL, client)
	return peerSessionsToCLI(sessions)
}

// peerSessionsToCLI converts shared PeerSession values to CLI-specific type.
func peerSessionsToCLI(sessions []tailnet.PeerSession) []CLIRemoteSession {
	result := make([]CLIRemoteSession, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, CLIRemoteSession{
			ID:       s.ID,
			Name:     s.Name,
			CLIType:  s.CLIType,
			Status:   s.Status,
			Hostname: s.Hostname,
			FQDN:     s.FQDN,
		})
	}
	return result
}
