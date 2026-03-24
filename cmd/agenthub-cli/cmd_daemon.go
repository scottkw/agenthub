package main

import (
	"fmt"
	"io"

	"github.com/agenthub/agenthub/internal/daemon"
)

// serviceControlFunc is the function used to control the service. Package-level var
// allows test injection of a mock.
var serviceControlFunc = daemon.ServiceControl

// cmdDaemon dispatches daemon subcommands: install, uninstall, start, stop, run.
// When called with no arguments, it runs the daemon directly (backward compat
// with EnsureDaemon's startDetachedDaemon which spawns `exe daemon`).
func cmdDaemon(args []string, out io.Writer) error {
	if len(args) == 0 {
		// No subcommand: run daemon directly (backward compat with EnsureDaemon).
		daemon.RunDaemon()
		return nil
	}
	switch args[0] {
	case "run":
		daemon.RunDaemon()
		return nil
	case "install":
		if err := serviceControlFunc("install"); err != nil {
			return fmt.Errorf("agenthub daemon install: %w", err)
		}
		fmt.Fprintln(out, "daemon service installed")
		return nil
	case "uninstall":
		if err := serviceControlFunc("uninstall"); err != nil {
			return fmt.Errorf("agenthub daemon uninstall: %w", err)
		}
		fmt.Fprintln(out, "daemon service uninstalled")
		return nil
	case "start":
		if err := serviceControlFunc("start"); err != nil {
			return fmt.Errorf("agenthub daemon start: %w", err)
		}
		fmt.Fprintln(out, "daemon service started")
		return nil
	case "stop":
		if err := serviceControlFunc("stop"); err != nil {
			return fmt.Errorf("agenthub daemon stop: %w", err)
		}
		fmt.Fprintln(out, "daemon service stopped")
		return nil
	default:
		return fmt.Errorf("unknown daemon subcommand %q; usage: agenthub daemon <install|uninstall|start|stop|run>", args[0])
	}
}
