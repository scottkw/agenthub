package main

import (
	"context"
	"fmt"
	"os"

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

	// fetchRemoteFn delegates to tailnet.FetchAllPeerSessions and converts
	// to TUI types. Uses the shared concurrent fetch with IP fallback.
	fetchRemoteFn := func(ctx context.Context) []tui.ListRemoteGroup {
		peers, err := client.ListTailnetPeers()
		if err != nil || len(peers) == 0 {
			return nil
		}
		groups := tailnet.FetchAllPeerSessions(ctx, peers)
		result := make([]tui.ListRemoteGroup, 0, len(groups))
		for _, g := range groups {
			entries := make([]tui.RemoteSessionEntry, 0, len(g.Sessions))
			for _, s := range g.Sessions {
				entries = append(entries, tui.RemoteSessionEntry{
					ID:       s.ID,
					Name:     s.Name,
					CLIType:  s.CLIType,
					Status:   s.Status,
					Hostname: s.Hostname,
					FQDN:     s.FQDN,
					URL:      s.URL,
				})
			}
			result = append(result, tui.ListRemoteGroup{Hostname: g.Hostname, Sessions: entries})
		}
		return result
	}

	return tui.Run(client, fetchRemoteFn, daemon.BuildVersion)
}
