package main

import (
	"context"
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

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	// Daemon sub-command: run daemon mode without EnsureDaemon.
	// This is what EnsureDaemon spawns — MUST be handled before any client setup.
	if cmd == "daemon" {
		daemon.RunDaemon()
		return
	}

	// All other commands: auto-start daemon, create client, dispatch.
	socketPath := daemon.DefaultSocketPath()
	if err := daemon.EnsureDaemon(socketPath); err != nil {
		fmt.Fprintf(os.Stderr, "agenthub: %v\n", err)
		os.Exit(1)
	}
	client := daemon.NewDaemonClient(socketPath)

	args := os.Args[2:]
	var err error

	switch cmd {
	case "new":
		err = cmdNew(client, args, os.Stdout)
	case "list":
		err = cmdList(client, os.Stdout)
	case "kill":
		err = cmdKill(client, args)
	case "rename":
		err = cmdRename(client, args)
	case "web":
		err = cmdWeb(client, args, os.Stdout)
	case "serve":
		err = cmdServe(client, args)
	case "unserve":
		err = cmdUnserve(client, args)
	case "health":
		err = cmdHealth(os.Stdout)
	case "qr":
		err = cmdQR(client, args, os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "agenthub: unknown command %q\n", cmd)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// usage prints the command reference to stderr.
func usage() {
	fmt.Fprint(os.Stderr, `Usage: agenthub <command> [args]

Commands:
  new <agent> <path>     Create a new session
  list                   List all sessions
  kill <id>              Terminate a session
  rename <id> <name>     Rename a session
  serve <id>             Enable web serving for a session
  unserve <id>           Disable web serving for a session
  web start              Start the Tailscale web server
  web stop               Stop the Tailscale web server
  web status             Show web server status
  health                 Check Tailscale health
  qr <id>                Display session QR code in terminal
`)
}

// cmdNew creates a new session. On success it writes the session UUID to out.
func cmdNew(client *daemon.DaemonClient, args []string, out io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: agenthub new <agent> <path>")
	}
	agent, workDir := args[0], args[1]
	name := filepath.Base(workDir)
	id, err := client.CreateSession(agent, name, workDir)
	if err != nil {
		return fmt.Errorf("agenthub new: %w", err)
	}
	fmt.Fprintln(out, id)
	return nil
}

// cmdList lists all sessions in a tabwriter table.
func cmdList(client *daemon.DaemonClient, out io.Writer) error {
	sessions, err := client.ListSessions()
	if err != nil {
		return fmt.Errorf("agenthub list: %w", err)
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
		return cmdWebStatus(client, out)
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

// cmdWebStatus prints the web server running/url/addr key-value block.
func cmdWebStatus(client *daemon.DaemonClient, out io.Writer) error {
	resp, err := client.GetWebServerStatus()
	if err != nil {
		return fmt.Errorf("agenthub web status: %w", err)
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

// cmdHealth prints a 5-line Tailscale health key-value block.
func cmdHealth(out io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h := webserver.CheckHealth(ctx)
	fmt.Fprintf(out, "%-12s%v\n", "installed:", h.Installed)
	fmt.Fprintf(out, "%-12s%v\n", "connected:", h.Connected)
	fmt.Fprintf(out, "%-12s%v\n", "has-certs:", h.HasCerts)
	fmt.Fprintf(out, "%-12s%v\n", "ip:", h.IP)
	fmt.Fprintf(out, "%-12s%v\n", "domain:", h.Domain)
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
	fmt.Fprint(out, q.ToString(false))
	fmt.Fprintln(out, url)
	return nil
}
