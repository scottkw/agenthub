package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/attach"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/statusbar"
	"golang.org/x/term"
)

// attachCmd implements tea.ExecCommand to run the full PTY attach flow
// while Bubble Tea suspends its renderer and input handling.
type attachCmd struct {
	client    *daemon.DaemonClient
	sessionID string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

func (a *attachCmd) SetStdin(r io.Reader)  { a.stdin = r }
func (a *attachCmd) SetStdout(w io.Writer) { a.stdout = w }
func (a *attachCmd) SetStderr(w io.Writer) { a.stderr = w }

// Run executes the attach flow: dials the relay WebSocket, sets up raw mode
// and status bar, then runs I/O pumps until detach (Ctrl-\) or disconnect.
func (a *attachCmd) Run() error {
	port, err := a.client.GetRelayPort()
	if err != nil {
		return err
	}

	sessions, err := a.client.ListSessions()
	if err != nil {
		return err
	}
	var session *daemon.SessionInfo
	for _, s := range sessions {
		if s.ID == a.sessionID {
			session = &s
			break
		}
	}
	if session == nil {
		return fmt.Errorf("session %q not found", a.sessionID)
	}

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sessions/%s/ws", port, a.sessionID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()

	lw := attach.NewLockedWriter(a.stdout)

	// Use os.Stdin directly for terminal operations. Bubble Tea v2 wraps
	// stdin in a cancelreader, so a.stdin is NOT *os.File — using it for
	// raw mode and status bar would silently skip both, leaving the terminal
	// in cooked mode where Ctrl-\ sends SIGQUIT instead of detach byte 0x1C.
	fd := os.Stdin.Fd()
	isTTY := term.IsTerminal(int(fd))

	// Status bar: only if stdin is a terminal.
	var bar *statusbar.Bar
	if isTTY {
		createdAt, _ := time.Parse(time.RFC3339, session.CreatedAt)
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		bar = statusbar.New(lw, statusbar.Options{
			SessionName: session.Name,
			AgentType:   session.CLI,
			Hostname:    session.Hostname,
			CreatedAt:   createdAt,
			Position:    statusbar.Bottom,
			Fd:          fd,
		})
		bar.Start()
		defer bar.Stop()
	}

	// Raw mode: required for byte-level input (detach key detection).
	if isTTY {
		oldState, err := term.MakeRaw(int(fd))
		if err != nil {
			return err
		}
		defer term.Restore(int(fd), oldState) //nolint:errcheck

		// Send initial resize frame.
		if cols, rows, err := term.GetSize(int(fd)); err == nil {
			frame := attach.MakeClientResizeFrame(uint16(cols), uint16(rows))
			_ = conn.Write(ctx, websocket.MessageBinary, frame)
		}
	}

	// Start platform-specific resize watcher (SIGWINCH on Unix, no-op on Windows).
	// WatchResize manages its own goroutine internally.
	attach.WatchResize(ctx, conn)

	// Use os.Stdin for the input pump so we read raw bytes directly,
	// bypassing Bubble Tea's cancelreader wrapper.
	// 0x1C is Ctrl-\ (the detach key).
	return attach.AttachSession(ctx, conn, os.Stdin, lw, 0x1C, bar, nil)
}
