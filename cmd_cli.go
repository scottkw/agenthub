package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/agenthub/agenthub/internal/daemon"
	"github.com/agenthub/agenthub/internal/webserver"
	qrcode "github.com/skip2/go-qrcode"
)

// usage prints the command reference to stderr.
func usage() {
	fmt.Fprint(os.Stderr, `Usage: agenthub [command] [flags]

Run with no arguments to launch the desktop GUI.

Commands:
  new <agent> <path> [-- <extra-args>...]     Create a new terminal session
  list [--json]                               List all active sessions
  kill <id>                                   Kill a session
  rename <id> <name>                          Rename a session
  attach <id>                                 Attach to a session (interactive PTY)
  serve <id>                                  Enable web sharing for a session
  unserve <id>                                Disable web sharing for a session
  web                                         Open the web dashboard
  health                                      Check Tailscale health status
  qr <id>                                     Print QR code for a session URL
  settings                                    Open the settings panel
  daemon <subcommand>                         Manage the background daemon
    daemon run                                Run daemon in foreground
    daemon install                            Install as system service
    daemon uninstall                          Uninstall system service
    daemon start                              Start the system service
    daemon stop                               Stop the system service
    daemon status [--json]                    Show daemon status

Run 'agenthub <command> --help' for command-specific flags.
`)
}

// cmdNew creates a new session. On success it writes the session UUID to out.
// extraArgs are forwarded to the agent process (tokens after "--" on the command line).
func cmdNew(client *daemon.DaemonClient, args []string, extraArgs []string, out io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: agenthub new <agent> <path>")
	}
	agent, workDir := args[0], args[1]
	name := filepath.Base(workDir)
	id, err := client.CreateSession(agent, name, workDir, extraArgs, 0, 0)
	if err != nil {
		return fmt.Errorf("agenthub new: %w", err)
	}
	fmt.Fprintln(out, id)
	return nil
}

// cmdList lists all sessions in a tabwriter table, or as JSON with --json.
func cmdList(client *daemon.DaemonClient, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	sessions, err := client.ListSessions()
	if err != nil {
		return fmt.Errorf("agenthub list: %w", err)
	}
	if *jsonOut {
		if sessions == nil {
			sessions = []daemon.SessionInfo{}
		}
		return json.NewEncoder(out).Encode(sessions)
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tAGENT\tSTATUS")
	for _, s := range sessions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Name, s.CLI, s.State)
	}
	w.Flush()
	return nil
}

// cmdKill terminates a session silently.
func cmdKill(client *daemon.DaemonClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: agenthub kill <session-id>")
	}
	if err := client.KillSession(args[0]); err != nil {
		return fmt.Errorf("agenthub kill: %w", err)
	}
	return nil
}

// cmdRename renames a session silently.
func cmdRename(client *daemon.DaemonClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: agenthub rename <session-id> <name>")
	}
	if err := client.RenameSession(args[0], args[1]); err != nil {
		return fmt.Errorf("agenthub rename: %w", err)
	}
	return nil
}

// cmdWeb dispatches web sub-commands: start, stop, status.
func cmdWeb(client *daemon.DaemonClient, args []string, out io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: agenthub web <start|stop|status>")
	}
	switch args[0] {
	case "start":
		return cmdWebStart(client, args[1:], out)
	case "stop":
		return cmdWebStop(client)
	case "status":
		return cmdWebStatus(client, args[1:], out)
	default:
		return fmt.Errorf("usage: agenthub web <start|stop|status>")
	}
}

// cmdWebStart validates Tailscale health then starts the web server, printing the URL.
func cmdWebStart(client *daemon.DaemonClient, _ []string, out io.Writer) error {
	port := 443
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h := webserver.CheckHealth(ctx)
	if !h.Connected {
		return fmt.Errorf("Tailscale is not connected")
	}
	if h.IP == "" {
		return fmt.Errorf("Tailscale IP not available")
	}
	if !h.HasCerts {
		return fmt.Errorf("Tailscale HTTPS certificates not enabled")
	}
	url, err := client.StartWebServer(h.IP, port, h.Domain)
	if err != nil {
		return fmt.Errorf("agenthub web start: %w", err)
	}
	fmt.Fprintln(out, url)
	return nil
}

// cmdWebStop stops the web server silently.
func cmdWebStop(client *daemon.DaemonClient) error {
	if err := client.StopWebServer(); err != nil {
		return fmt.Errorf("agenthub web stop: %w", err)
	}
	return nil
}

// cmdWebStatus prints the web server running/url/addr key-value block, or JSON with --json.
func cmdWebStatus(client *daemon.DaemonClient, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("web-status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	resp, err := client.GetWebServerStatus()
	if err != nil {
		return fmt.Errorf("agenthub web status: %w", err)
	}
	if *jsonOut {
		return json.NewEncoder(out).Encode(resp)
	}
	fmt.Fprintf(out, "%-12s%v\n", "running:", resp.Running)
	if resp.Running {
		fmt.Fprintf(out, "%-12s%v\n", "url:", resp.URL)
		fmt.Fprintf(out, "%-12s%v\n", "addr:", resp.Addr)
	}
	return nil
}

// cmdServe enables web serving for the given session.
func cmdServe(client *daemon.DaemonClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: agenthub serve <session-id>")
	}
	if err := client.ToggleWebServing(args[0], true); err != nil {
		return fmt.Errorf("agenthub serve: %w", err)
	}
	return nil
}

// cmdUnserve disables web serving for the given session.
func cmdUnserve(client *daemon.DaemonClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: agenthub unserve <session-id>")
	}
	if err := client.ToggleWebServing(args[0], false); err != nil {
		return fmt.Errorf("agenthub unserve: %w", err)
	}
	return nil
}

// cmdHealth prints a 5-line Tailscale health key-value block, or JSON with --json.
func cmdHealth(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h := webserver.CheckHealth(ctx)
	if *jsonOut {
		return json.NewEncoder(out).Encode(h)
	}
	fmt.Fprintf(out, "%-12s%v\n", "installed:", h.Installed)
	fmt.Fprintf(out, "%-12s%v\n", "connected:", h.Connected)
	fmt.Fprintf(out, "%-12s%v\n", "has-certs:", h.HasCerts)
	fmt.Fprintf(out, "%-12s%v\n", "ip:", h.IP)
	fmt.Fprintf(out, "%-12s%v\n", "domain:", h.Domain)
	return nil
}

// cmdSettings prints current configuration values in a human-readable format (read-only).
func cmdSettings(client *daemon.DaemonClient, out io.Writer) error {
	socketPath := daemon.DefaultSocketPath()
	fmt.Fprintf(out, "%-14s%s\n", "socket-path:", socketPath)

	port, err := client.GetRelayPort()
	if err != nil {
		fmt.Fprintf(out, "%-14s%s\n", "relay-port:", "(unavailable)")
	} else {
		fmt.Fprintf(out, "%-14s%d\n", "relay-port:", port)
	}

	paths, err := client.GetCLIPaths()
	if err != nil || len(paths) == 0 {
		fmt.Fprintf(out, "%-14s%s\n", "cli-paths:", "(none)")
	} else {
		first := true
		for name, p := range paths {
			if first {
				fmt.Fprintf(out, "%-14s%s=%s\n", "cli-paths:", name, p)
				first = false
			} else {
				fmt.Fprintf(out, "%-14s%s=%s\n", "", name, p)
			}
		}
	}
	return nil
}

// cmdQR renders a QR code for the given session URL in the terminal.
func cmdQR(client *daemon.DaemonClient, args []string, out io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: agenthub qr <session-id>")
	}
	resp, err := client.GetWebServerStatus()
	if err != nil || !resp.Running {
		return fmt.Errorf("web server not running — use 'agenthub web start' first")
	}
	url := fmt.Sprintf("%s/sessions/%s", resp.URL, args[0])
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("agenthub qr: %w", err)
	}
	fmt.Fprint(out, q.ToSmallString(false))
	fmt.Fprintln(out, url)
	return nil
}
