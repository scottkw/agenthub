package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/tailnet"
	"github.com/scottkw/agenthub/internal/tui"
	"golang.org/x/term"
)

// cmdTUI launches the interactive Bubble Tea terminal UI.
func cmdTUI(client *daemon.DaemonClient) error {
	// Pre-check: stdout must be a TTY (UI-SPEC: non-TTY fallback).
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("agenthub tui requires a terminal. Redirect to a TTY or use 'agenthub list' instead")
	}

	// Validate daemon is reachable before starting the TUI program.
	if err := client.Health(); err != nil {
		return fmt.Errorf("cannot connect to daemon: %w", err)
	}

	// fetchRemoteFn is a callback that fetches remote sessions from tailnet peers.
	// It wraps package-main functions (fetchPeerSessions, ListTailnetPeers) to avoid
	// an import cycle between internal/tui and package main.
	fetchRemoteFn := func(ctx context.Context) []tui.ListRemoteGroup {
		peers, err := client.ListTailnetPeers()
		if err != nil {
			log.Printf("[warn] tailnet peer discovery failed: %v", err)
			return nil
		}
		if len(peers) == 0 {
			return nil
		}
		groupMap := make(map[string][]tui.RemoteSessionEntry)
		for _, p := range peers {
			fqdn := strings.TrimSuffix(p.DNSName, ".")
			peerSessions, err := fetchPeerSessions(ctx, fqdn, tailnet.DefaultProbePort)
			if err != nil {
				log.Printf("[warn] peer session fetch failed: peer=%s err=%v", fqdn, err)
				continue
			}
			for _, s := range peerSessions {
				groupMap[p.Hostname] = append(groupMap[p.Hostname], tui.RemoteSessionEntry{
					ID:       s.ID,
					Name:     s.Name,
					CLIType:  s.CLIType,
					Status:   s.Status,
					Hostname: p.Hostname,
					FQDN:     fqdn,
					URL:      fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, tailnet.DefaultProbePort, s.ID),
				})
			}
		}
		var groups []tui.ListRemoteGroup
		for hostname, sess := range groupMap {
			groups = append(groups, tui.ListRemoteGroup{Hostname: hostname, Sessions: sess})
		}
		sort.Slice(groups, func(i, j int) bool { return groups[i].Hostname < groups[j].Hostname })
		return groups
	}

	return tui.Run(client, fetchRemoteFn)
}
