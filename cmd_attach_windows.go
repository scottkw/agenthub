//go:build windows

package main

import (
	"context"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/attach"
)

// watchResize delegates to the shared attach package. On Windows this is a
// no-op since SIGWINCH is not available.
func watchResize(ctx context.Context, conn *websocket.Conn) {
	attach.WatchResize(ctx, conn)
}
