package main

import (
	"context"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/attach"
)

// watchResize delegates to the shared attach package which handles
// platform-specific resize detection internally (SIGWINCH on Unix, no-op on
// Windows).
func watchResize(ctx context.Context, conn *websocket.Conn) {
	attach.WatchResize(ctx, conn)
}
