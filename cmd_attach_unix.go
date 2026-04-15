//go:build !windows

package main

import (
	"context"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/attach"
)

// watchResize delegates to the shared attach package which listens for SIGWINCH
// (terminal resize) signals on Unix platforms and sends resize frames to the relay.
func watchResize(ctx context.Context, conn *websocket.Conn) {
	attach.WatchResize(ctx, conn)
}
