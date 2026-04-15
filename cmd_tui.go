package main

import (
	"fmt"
	"os"

	"github.com/scottkw/agenthub/internal/daemon"
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

	return tui.Run(client)
}
