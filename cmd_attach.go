package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/attach"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/statusbar"
	"github.com/scottkw/agenthub/internal/tailnet"
	"golang.org/x/term"
)

// cmdAttach connects the local terminal to a running daemon session via the
// relay WebSocket server. It sets the terminal to raw mode, relays all I/O,
// and restores the terminal on every exit path.
func cmdAttach(client *daemon.DaemonClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: agenthub attach <session-id>")
	}

	// Parse optional flags. Default detach key: Ctrl-backslash (0x1C).
	detachKey := byte(0x1C)
	readOnly := false
	clientName := ""
	statusTop := false
	for _, arg := range args[1:] {
		if arg == "--readonly" {
			readOnly = true
		} else if arg == "--status-top" {
			statusTop = true
		} else if len(arg) > 9 && arg[:9] == "--client=" {
			clientName = arg[9:]
		} else if len(arg) > 13 && arg[:13] == "--detach-key=" {
			val := arg[13:]
			switch val {
			case `ctrl-\`, "ctrl-backslash":
				detachKey = 0x1C
			default:
				if len(val) == 1 {
					detachKey = val[0]
				}
			}
		}
	}

	// Detect remote session ID format (hostname:session-id).
	hostname, sessionID, isRemote := parseRemoteID(args[0])
	if isRemote {
		return cmdAttachRemote(client, hostname, sessionID, detachKey, statusTop)
	}

	// Must be run in an interactive terminal.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("attach: stdin is not a terminal")
	}

	// Get the relay port from the daemon.
	port, err := client.GetRelayPort()
	if err != nil {
		return fmt.Errorf("attach: %w", err)
	}

	// Verify the session exists and capture its metadata for the banner.
	sessions, err := client.ListSessions()
	if err != nil {
		return fmt.Errorf("attach: %w", err)
	}
	var session *daemon.SessionInfo
	for _, s := range sessions {
		if s.ID == sessionID {
			session = &s
			break
		}
	}
	if session == nil {
		return fmt.Errorf("attach: session %q not found", sessionID)
	}

	// MC-03, MC-05: build WebSocket URL with optional query params.
	u := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("127.0.0.1:%d", port),
		Path:   fmt.Sprintf("/sessions/%s/ws", sessionID),
	}
	q := url.Values{}
	if readOnly {
		q.Set("readonly", "1")
	}
	if clientName != "" {
		q.Set("client", clientName)
	}
	if len(q) > 0 {
		u.RawQuery = q.Encode()
	}
	wsURL := u.String()

	// Create signal context that cancels on SIGTERM or SIGHUP.
	// Do NOT catch SIGINT — in raw mode Ctrl-C is byte 0x03, not a signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("attach: dial relay: %w", err)
	}
	defer conn.CloseNow()

	// Wrap stdout in a LockedWriter to serialize PTY output and bar draws.
	stdout := attach.NewLockedWriter(os.Stdout)

	// Create status bar if stdout is a TTY (SB-03).
	var bar *statusbar.Bar
	if term.IsTerminal(int(os.Stdout.Fd())) {
		createdAt, _ := time.Parse(time.RFC3339, session.CreatedAt)
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		pos := statusbar.Bottom
		if statusTop {
			pos = statusbar.Top
		}
		bar = statusbar.New(stdout, statusbar.Options{
			SessionName: session.Name,
			AgentType:   session.CLI,
			Hostname:    session.Hostname,
			CreatedAt:   createdAt,
			Position:    pos,
			Fd:          os.Stdout.Fd(),
		})
		bar.Start()
		defer bar.Stop()
	} else {
		// Non-TTY path: show one-shot banner on stderr (bar is suppressed).
		printAttachBanner(os.Stderr, session.Name, session.CLI, session.Hostname)
	}

	// Put terminal in raw mode. Restore on every exit path.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("attach: raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck

	// Send initial resize so PTY dimensions match the local terminal.
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err == nil {
		frame := attach.MakeClientResizeFrame(uint16(cols), uint16(rows))
		_ = conn.Write(ctx, websocket.MessageBinary, frame)
	}

	// Start platform-specific SIGWINCH watcher (no-op on Windows).
	watchResize(ctx, conn)

	err = attach.AttachSession(ctx, conn, os.Stdin, stdout, detachKey, bar, nil)
	printDetachMessage(os.Stderr)
	return err
}

// cmdAttachRemote handles attaching to a session on a remote tailnet peer.
// It resolves the hostname to FQDN via tailnet peer discovery, verifies the
// session exists on the remote peer, and connects via WSS relay.
func cmdAttachRemote(client *daemon.DaemonClient, hostname, sessionID string, detachKey byte, statusTop bool) error {
	// Must be run in an interactive terminal.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("attach: stdin is not a terminal")
	}

	// Resolve hostname to peer via tailnet peer discovery.
	peers, err := client.ListTailnetPeers()
	if err != nil {
		return fmt.Errorf("attach: discover peers: %w", err)
	}
	var peer *tailnet.Peer
	for _, p := range peers {
		if strings.EqualFold(p.Hostname, hostname) {
			peer = &p
			break
		}
	}
	if peer == nil {
		return buildUnknownHostError(hostname, peers)
	}
	fqdn := strings.TrimSuffix(peer.DNSName, ".")

	// Construct base URL for fetching sessions and WSS relay.
	baseURL := fmt.Sprintf("https://%s:%d", fqdn, tailnet.DefaultProbePort)

	return cmdAttachRemoteWithClient(hostname, sessionID, fqdn, baseURL, nil, peer, detachKey, statusTop)
}

// cmdAttachRemoteWithClient is the testable core of the remote attach flow.
// It accepts an HTTP client and base URL for testing with httptest servers.
// If httpClient is nil, a production TLS client is used and peer is used for
// IP-fallback fetch via tailnet.FetchPeerSessions.
func cmdAttachRemoteWithClient(hostname, sessionID, fqdn, baseURL string, httpClient *http.Client, peer *tailnet.Peer, detachKey byte, statusTop bool) error {
	// Verify the session exists on the remote peer and get its metadata.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var remoteSessions []CLIRemoteSession
	if httpClient != nil {
		remoteSessions = fetchPeerSessionsWithClient(ctx, baseURL, httpClient)
	} else if peer != nil {
		remoteSessions = fetchPeerSessions(ctx, *peer)
	}

	var session *CLIRemoteSession
	for _, s := range remoteSessions {
		if s.ID == sessionID {
			session = &s
			break
		}
	}
	if session == nil {
		return fmt.Errorf("attach: session %q not found on remote host %q", sessionID, hostname)
	}

	// Construct WSS URL to remote peer's relay.
	wsURL := fmt.Sprintf("wss://%s:%d/sessions/%s/ws", fqdn, tailnet.DefaultProbePort, sessionID)

	// Create signal context (same as local attach).
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	conn, _, err := websocket.Dial(sigCtx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("attach: dial remote relay: %w", err)
	}
	defer conn.CloseNow()

	// Wrap stdout in a LockedWriter.
	stdout := attach.NewLockedWriter(os.Stdout)

	// Create status bar if stdout is a TTY (SB-03).
	var bar *statusbar.Bar
	if term.IsTerminal(int(os.Stdout.Fd())) {
		pos := statusbar.Bottom
		if statusTop {
			pos = statusbar.Top
		}
		bar = statusbar.New(stdout, statusbar.Options{
			SessionName: session.Name,
			AgentType:   session.CLIType,
			Hostname:    hostname,
			CreatedAt:   time.Now(), // CLIRemoteSession has no CreatedAt
			Position:    pos,
			Fd:          os.Stdout.Fd(),
		})
		bar.Start()
		defer bar.Stop()
	} else {
		printAttachBanner(os.Stderr, session.Name, session.CLIType, hostname)
	}

	// Put terminal in raw mode (same as local attach).
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("attach: raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck

	// Send initial resize.
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err == nil {
		frame := attach.MakeClientResizeFrame(uint16(cols), uint16(rows))
		_ = conn.Write(sigCtx, websocket.MessageBinary, frame)
	}

	// Start resize watcher (platform-specific).
	watchResize(sigCtx, conn)

	// SB-05: connection state watcher for remote sessions.
	// Sets "reconnecting" if no frame received for >5s.
	var onFrame func()
	if bar != nil {
		var mu sync.Mutex
		var lastFrame time.Time

		onFrame = func() {
			mu.Lock()
			lastFrame = time.Now()
			mu.Unlock()
			bar.SetConnectionState("")
		}
		onFrame() // initialize with current time

		// Watcher goroutine: set "reconnecting" if no frame for 5s.
		go func() {
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					mu.Lock()
					lf := lastFrame
					mu.Unlock()
					if time.Since(lf) > 5*time.Second {
						bar.SetConnectionState("reconnecting")
					}
				case <-sigCtx.Done():
					return
				}
			}
		}()
	}

	err = attach.AttachSession(sigCtx, conn, os.Stdin, stdout, detachKey, bar, onFrame)
	printDetachMessage(os.Stderr)
	return err
}

// buildUnknownHostError creates a helpful error message when a hostname
// doesn't match any tailnet peer, listing available peer hostnames.
func buildUnknownHostError(hostname string, peers []tailnet.Peer) error {
	var names []string
	for _, p := range peers {
		names = append(names, p.Hostname)
	}
	if len(names) == 0 {
		return fmt.Errorf("attach: unknown remote host %q — no tailnet peers found", hostname)
	}
	return fmt.Errorf("attach: unknown remote host %q\nAvailable peers: %s", hostname, strings.Join(names, ", "))
}

// printAttachBanner writes the connection banner to w (typically os.Stderr).
// It shows session name, CLI type, hostname, and detach key hint.
func printAttachBanner(w io.Writer, name, cli, hostname string) {
	displayName := name
	if displayName == "" {
		displayName = "unnamed"
	}
	fmt.Fprintf(w, "───────────────────────────────────\n")
	fmt.Fprintf(w, " %s", displayName)
	if cli != "" {
		fmt.Fprintf(w, " │ %s", cli)
	}
	if hostname != "" {
		fmt.Fprintf(w, " │ %s", hostname)
	}
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, " Press Ctrl-\\ to detach.\n")
	fmt.Fprintf(w, "───────────────────────────────────\n")
}

// printDetachMessage writes the detach confirmation to w (typically os.Stderr).
func printDetachMessage(w io.Writer) {
	fmt.Fprintf(w, "\nDetached.\n")
}
