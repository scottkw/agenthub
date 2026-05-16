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

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/tailnet"
	"github.com/scottkw/agenthub/internal/webserver"
	qrcode "github.com/skip2/go-qrcode"
)

// usage prints the command reference to stderr.
func usage() {
	fmt.Fprint(os.Stderr, `Usage: agenthub [command] [flags]

Run with no arguments to launch the desktop GUI.

Commands:
  new <agent> <path> [-- <extra-args>...]     Create a new terminal session
  new shell [<path>]                          Create a new raw shell session
    --shell=bash|zsh|pwsh|powershell           Pick a specific shell (default: system default)
  list [--json] [--local]                     List local and remote sessions
  kill <id>                                   Kill a session
  rename <id> <name>                          Rename a session
  attach <id>                                 Attach to a local or remote session
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
  tui                                         Launch interactive terminal UI

Remote Sessions:
  list                                        Shows local + remote sessions grouped by host
  list --local                                Shows only local sessions
  attach <hostname:session-id>                Attach to a remote session (e.g., macbook:abc123)

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

// cmdNewShell creates a new raw shell session (Phase 101 SHELL-02).
//
// Argv shape:
//
//	agenthub new shell [<path>]
//
//   - `<path>` is optional; if omitted, workDir is "" and the daemon resolves to $HOME.
//   - The daemon resolves the spawned shell binary from the Settings-stored
//     shellPath (engine.go:500-530, GetShellPath/SetShellPath at 670-705).
//   - `extraArgs` (the `--` tail) is intentionally NOT forwarded to shells
//     (per Phase 100 Anti-Pattern A6); a non-fatal stderr warning is emitted instead.
//
// Phase 108 PARITY-CLI-01: The --shell=bash|zsh|pwsh|powershell flag was
// removed in Phase 108 (hard removal, no deprecation period). The daemon
// resolves the shell binary via Settings shellPath (engine.go:500-530), so the
// per-shell CLI override is redundant. Passing --shell=anything now produces
// the Go flag package's default `flag provided but not defined: -shell`
// stderr and a non-zero exit.
//
// Locked stderr error strings (remaining after Phase 108):
//   - extra args after --:   `agenthub new shell: extra arguments are not forwarded to shell sessions; ignoring [...]`
//   - daemon unreachable:    `agenthub new shell: daemon unreachable: <err>`
func cmdNewShell(client *daemon.DaemonClient, args []string, extraArgs []string, out io.Writer) error {
	fs := flag.NewFlagSet("new shell", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // we emit our own error copy
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "agenthub new shell: %v\n", err)
		return err
	}
	const cli = "shell"
	workDir := ""
	positionals := fs.Args()
	if len(positionals) > 0 {
		workDir = positionals[0]
	}
	if len(extraArgs) > 0 {
		fmt.Fprintf(os.Stderr, "agenthub new shell: extra arguments are not forwarded to shell sessions; ignoring %v\n", extraArgs)
	}
	// Session name mirrors cmdNew: basename of workDir, or the cli name when workDir is empty
	// (daemon resolves "" → $HOME, but the CLI doesn't know $HOME at this layer).
	name := cli
	if workDir != "" {
		name = filepath.Base(workDir)
	}
	id, err := client.CreateSession(cli, name, workDir, nil, 0, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agenthub new shell: daemon unreachable: %v\n", err)
		return err
	}
	fmt.Fprintln(out, id)
	return nil
}

// listOutput is the JSON structure for cmdList --json output.
type listOutput struct {
	Local  []daemon.SessionInfo `json:"local"`
	Remote []listRemoteGroup    `json:"remote,omitempty"`
}

// listRemoteGroup groups remote sessions by peer hostname.
type listRemoteGroup struct {
	Hostname string             `json:"hostname"`
	Sessions []CLIRemoteSession `json:"sessions"`
}

// cmdList lists all sessions in a tabwriter table, or as JSON with --json.
// Includes remote sessions from tailnet peers unless --local is specified.
func cmdList(client *daemon.DaemonClient, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output as JSON")
	localOnly := fs.Bool("local", false, "show only local sessions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	sessions, err := client.ListSessions()
	if err != nil {
		return fmt.Errorf("agenthub list: %w", err)
	}
	if sessions == nil {
		sessions = []daemon.SessionInfo{}
	}

	// Fetch remote sessions (unless --local).
	var remoteGroups []listRemoteGroup
	if !*localOnly {
		peers, _ := client.ListTailnetPeers()
		if len(peers) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			groups := tailnet.FetchAllPeerSessions(ctx, peers)
			for _, g := range groups {
				cliSessions := make([]CLIRemoteSession, 0, len(g.Sessions))
				for _, s := range g.Sessions {
					cliSessions = append(cliSessions, CLIRemoteSession{
						ID:       s.ID,
						Name:     s.Name,
						CLIType:  s.CLIType,
						Status:   s.Status,
						Hostname: s.Hostname,
						FQDN:     s.FQDN,
					})
				}
				remoteGroups = append(remoteGroups, listRemoteGroup{
					Hostname: g.Hostname,
					Sessions: cliSessions,
				})
			}
		}
	}

	if *jsonOut {
		output := listOutput{
			Local:  sessions,
			Remote: remoteGroups,
		}
		return json.NewEncoder(out).Encode(output)
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "HOST\tID\tNAME\tAGENT\tSTATUS")
	for _, s := range sessions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "(local)", s.ID, s.Name, s.CLI, s.State)
	}
	for _, group := range remoteGroups {
		for _, s := range group.Sessions {
			fmt.Fprintf(w, "%s\t%s:%s\t%s\t%s\t%s\n", s.Hostname, s.Hostname, s.ID, s.Name, s.CLIType, s.Status)
		}
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
	url, err := client.StartWebServer(h.IP, port, h.Domain, "tailscale", "")
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
