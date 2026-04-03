package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/scottkw/agenthub/internal/daemon"
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
		return fmt.Errorf("unknown daemon subcommand %q; usage: agenthub daemon <install|uninstall|start|stop|status|run>", args[0])
	}
}

// cmdDaemonStatus checks daemon reachability and prints status.
func cmdDaemonStatus(client *daemon.DaemonClient, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("daemon-status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	running := client.Health() == nil

	if *jsonOut {
		type statusResp struct {
			Running bool `json:"running"`
		}
		return json.NewEncoder(out).Encode(statusResp{Running: running})
	}
	fmt.Fprintf(out, "%-12s%v\n", "running:", running)
	return nil
}
