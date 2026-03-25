//go:build !windows

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/coder/websocket"
	"golang.org/x/term"
)

// watchResize listens for SIGWINCH (terminal resize) signals on Unix platforms
// and sends a MsgResize2 frame to the relay whenever the terminal dimensions
// change. It runs in a background goroutine and exits when ctx is cancelled.
func watchResize(ctx context.Context, conn *websocket.Conn) {
	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)
	go func() {
		defer signal.Stop(winchCh)
		for {
			select {
			case <-winchCh:
				cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
				if err != nil {
					continue
				}
				frame := makeClientResizeFrame(uint16(cols), uint16(rows))
				_ = conn.Write(ctx, websocket.MessageBinary, frame)
			case <-ctx.Done():
				return
			}
		}
	}()
}
