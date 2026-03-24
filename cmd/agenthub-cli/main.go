package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/agenthub/agenthub/internal/daemon"
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
	// Stubs for Plan 02 commands.
	case "serve", "unserve", "web", "health", "qr":
		fmt.Fprintln(os.Stderr, "agenthub: command not yet implemented")
		os.Exit(1)
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
