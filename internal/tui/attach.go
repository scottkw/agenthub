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

	// Status bar: only if stdin is *os.File and a terminal.
	var bar *statusbar.Bar
	if f, ok := a.stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
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
			Fd:          f.Fd(),
		})
		bar.Start()
		defer bar.Stop()
	}

	// Raw mode: only if stdin is *os.File.
	if f, ok := a.stdin.(*os.File); ok {
		oldState, err := term.MakeRaw(int(f.Fd()))
		if err != nil {
			return err
		}
		defer term.Restore(int(f.Fd()), oldState) //nolint:errcheck

		// Send initial resize frame.
		if cols, rows, err := term.GetSize(int(f.Fd())); err == nil {
			frame := attach.MakeClientResizeFrame(uint16(cols), uint16(rows))
			_ = conn.Write(ctx, websocket.MessageBinary, frame)
		}
	}

	// Start platform-specific resize watcher (SIGWINCH on Unix, no-op on Windows).
	// WatchResize manages its own goroutine internally.
	attach.WatchResize(ctx, conn)

	// 0x1C is Ctrl-\ (the detach key).
	return attach.AttachSession(ctx, conn, a.stdin, lw, 0x1C, bar, nil)
}
