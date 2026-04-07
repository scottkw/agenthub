package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/relay"
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

	// Parse optional --detach-key flag. Default: Ctrl-backslash (0x1C).
	detachKey := byte(0x1C)
	for _, arg := range args[1:] {
		if len(arg) > 13 && arg[:13] == "--detach-key=" {
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
		return cmdAttachRemote(client, hostname, sessionID, detachKey)
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
			s := s // capture loop variable
			session = &s
			break
		}
	}
	if session == nil {
		return fmt.Errorf("attach: session %q not found", sessionID)
	}

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sessions/%s/ws", port, sessionID)

	// Create signal context that cancels on SIGTERM or SIGHUP.
	// Do NOT catch SIGINT — in raw mode Ctrl-C is byte 0x03, not a signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("attach: dial relay: %w", err)
	}
	defer conn.CloseNow()

	// Print connection banner to stderr before entering raw mode.
	printAttachBanner(os.Stderr, session.Name, session.CLI, session.Hostname)

	// Put terminal in raw mode. Restore on every exit path.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("attach: raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck

	// Send initial resize so PTY dimensions match the local terminal.
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err == nil {
		frame := makeClientResizeFrame(uint16(cols), uint16(rows))
		_ = conn.Write(ctx, websocket.MessageBinary, frame)
	}

	// Start platform-specific SIGWINCH watcher (no-op on Windows).
	watchResize(ctx, conn)

	err = attachSession(ctx, conn, os.Stdin, os.Stdout, detachKey)
	printDetachMessage(os.Stderr)
	return err
}

// cmdAttachRemote handles attaching to a session on a remote tailnet peer.
// It resolves the hostname to FQDN via tailnet peer discovery, verifies the
// session exists on the remote peer, and connects via WSS relay.
func cmdAttachRemote(client *daemon.DaemonClient, hostname, sessionID string, detachKey byte) error {
	// Must be run in an interactive terminal.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("attach: stdin is not a terminal")
	}

	// Resolve hostname to FQDN via tailnet peer discovery.
	peers, err := client.ListTailnetPeers()
	if err != nil {
		return fmt.Errorf("attach: discover peers: %w", err)
	}
	fqdn, found := resolveRemotePeer(peers, hostname)
	if !found {
		return buildUnknownHostError(hostname, peers)
	}

	// Construct base URL for fetching sessions and WSS relay.
	baseURL := fmt.Sprintf("https://%s:%d", fqdn, tailnet.DefaultProbePort)

	return cmdAttachRemoteWithClient(hostname, sessionID, fqdn, baseURL, nil, detachKey)
}

// cmdAttachRemoteWithClient is the testable core of the remote attach flow.
// It accepts an HTTP client and base URL for testing with httptest servers.
// If httpClient is nil, a production TLS client is used.
func cmdAttachRemoteWithClient(hostname, sessionID, fqdn, baseURL string, httpClient *http.Client, detachKey byte) error {
	// Verify the session exists on the remote peer and get its metadata.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var remoteSessions []CLIRemoteSession
	var fetchErr error
	if httpClient != nil {
		remoteSessions, fetchErr = fetchPeerSessionsWithClient(ctx, baseURL, httpClient)
	} else {
		remoteSessions, fetchErr = fetchPeerSessions(ctx, fqdn, tailnet.DefaultProbePort)
	}
	_ = fetchErr // fetchPeerSessions returns empty slice on error

	var session *CLIRemoteSession
	for _, s := range remoteSessions {
		s := s
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

	// Print banner with remote hostname shown clearly.
	printAttachBanner(os.Stderr, session.Name, session.CLIType, hostname)

	// Put terminal in raw mode (same as local attach).
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("attach: raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck

	// Send initial resize.
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err == nil {
		frame := makeClientResizeFrame(uint16(cols), uint16(rows))
		_ = conn.Write(sigCtx, websocket.MessageBinary, frame)
	}

	// Start resize watcher (platform-specific).
	watchResize(sigCtx, conn)

	err = attachSession(sigCtx, conn, os.Stdin, os.Stdout, detachKey)
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

// attachSession is the testable core of the attach flow. It runs two I/O
// pumps concurrently and returns when either completes or ctx is cancelled.
func attachSession(ctx context.Context, conn *websocket.Conn, stdin io.Reader, stdout io.Writer, detachKey byte) error {
	type result struct{ err error }

	stdinDone := make(chan result, 1)
	wsDone := make(chan result, 1)

	go func() {
		stdinDone <- result{stdinPump(ctx, conn, stdin, detachKey)}
	}()
	go func() {
		wsDone <- result{wsOutputPump(ctx, conn, stdout)}
	}()

	select {
	case <-stdinDone:
	case <-wsDone:
	case <-ctx.Done():
	}

	conn.Close(websocket.StatusNormalClosure, "detach") //nolint:errcheck
	return nil
}

// stdinPump reads from r, scans for the detach key, and forwards input to the
// relay via MakeInputFrame. It returns nil on clean detach.
func stdinPump(ctx context.Context, conn *websocket.Conn, r io.Reader, detachKey byte) error {
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := r.Read(buf)
		if n > 0 {
			// Scan for detach key.
			detachIdx := -1
			for i := 0; i < n; i++ {
				if buf[i] == detachKey {
					detachIdx = i
					break
				}
			}

			if detachIdx >= 0 {
				// Send bytes before the detach key (if any), then detach cleanly.
				if detachIdx > 0 {
					frame := relay.MakeInputFrame(buf[:detachIdx])
					if werr := conn.Write(ctx, websocket.MessageBinary, frame); werr != nil {
						return werr
					}
				}
				return nil // clean detach
			}

			// No detach key found — send entire buffer as one frame.
			frame := relay.MakeInputFrame(buf[:n])
			if werr := conn.Write(ctx, websocket.MessageBinary, frame); werr != nil {
				return werr
			}
		}

		if err != nil {
			return err
		}
	}
}

// wsOutputPump reads WebSocket messages from conn and writes MsgOutput
// payloads to w. It handles the initial scrollback snapshot (first message,
// which may be large) and subsequent live frames.
func wsOutputPump(ctx context.Context, conn *websocket.Conn, w io.Writer) error {
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		msgType, payload, ferr := relay.ParseFrame(msg)
		if ferr != nil {
			continue
		}
		if msgType == relay.MsgOutput {
			if _, werr := w.Write(payload); werr != nil {
				return werr
			}
		}
	}
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

// makeClientResizeFrame builds a MsgResize2 frame for client-to-server resize.
// Uses MsgResize2 (0x11) which the server's read pump handles at relay/server.go.
// Do NOT use relay.MakeResizeFrame() — it uses MsgResize (0x02) which the
// server ignores for client-originated resize messages.
func makeClientResizeFrame(cols, rows uint16) []byte {
	return []byte{
		relay.MsgResize2,
		byte(cols >> 8), byte(cols),
		byte(rows >> 8), byte(rows),
	}
}
