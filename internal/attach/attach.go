// Package attach provides shared attach logic for connecting to a relay
// WebSocket session. Both the CLI (cmd_attach.go) and TUI (internal/tui/attach.go)
// import this package to avoid duplicating I/O pump and resize logic.
package attach

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/statusbar"
)

// LockedWriter serialises concurrent writes to an underlying io.Writer.
// Used to prevent interleaving of PTY output and status bar draw sequences.
type LockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// NewLockedWriter returns a new LockedWriter wrapping w.
func NewLockedWriter(w io.Writer) *LockedWriter {
	return &LockedWriter{w: w}
}

func (lw *LockedWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.w.Write(p)
}

// AttachSession is the testable core of the attach flow. It runs two I/O
// pumps concurrently and returns when either completes or ctx is cancelled.
func AttachSession(ctx context.Context, conn *websocket.Conn, stdin io.Reader, stdout io.Writer, detachKey byte, bar *statusbar.Bar, onFrame func()) error {
	type result struct{ err error }

	stdinDone := make(chan result, 1)
	wsDone := make(chan result, 1)

	go func() {
		stdinDone <- result{StdinPump(ctx, conn, stdin, detachKey)}
	}()
	go func() {
		wsDone <- result{WsOutputPump(ctx, conn, stdout, bar, onFrame)}
	}()

	select {
	case <-stdinDone:
	case <-wsDone:
	case <-ctx.Done():
	}

	conn.Close(websocket.StatusNormalClosure, "detach") //nolint:errcheck
	return nil
}

// StdinPump reads from r, scans for the detach key, and forwards input to the
// relay via MakeInputFrame. It returns nil on clean detach.
func StdinPump(ctx context.Context, conn *websocket.Conn, r io.Reader, detachKey byte) error {
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

// WsOutputPump reads WebSocket messages from conn and writes MsgOutput
// payloads to w. It handles the initial scrollback snapshot (first message,
// which may be large) and subsequent live frames. MsgMeta frames update the
// bar viewer count (SB-04). onFrame is called on every received frame to
// track connection liveness (SB-05).
func WsOutputPump(ctx context.Context, conn *websocket.Conn, w io.Writer, bar *statusbar.Bar, onFrame func()) error {
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if onFrame != nil {
			onFrame()
		}

		msgType, payload, ferr := relay.ParseFrame(msg)
		if ferr != nil {
			continue
		}
		switch msgType {
		case relay.MsgOutput:
			if _, werr := w.Write(payload); werr != nil {
				return werr
			}
		case relay.MsgMeta:
			// SB-04: update bar with viewer count from server push.
			if bar != nil {
				var meta relay.MetaPayload
				if err := json.Unmarshal(payload, &meta); err == nil && meta.ViewerCount != nil {
					bar.SetViewerCount(*meta.ViewerCount)
				}
			}
		}
	}
}

// MakeClientResizeFrame builds a MsgResize2 frame for client-to-server resize.
// Uses MsgResize2 (0x11) which the server's read pump handles at relay/server.go.
// Do NOT use relay.MakeResizeFrame() — it uses MsgResize (0x02) which the
// server ignores for client-originated resize messages.
func MakeClientResizeFrame(cols, rows uint16) []byte {
	return []byte{
		relay.MsgResize2,
		byte(cols >> 8), byte(cols),
		byte(rows >> 8), byte(rows),
	}
}
