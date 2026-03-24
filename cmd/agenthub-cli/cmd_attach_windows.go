//go:build windows

package main

import (
	"context"

	"github.com/coder/websocket"
)

// watchResize is a no-op on Windows. SIGWINCH is not available on Windows,
// so resize propagation is not supported in this release. The remote PTY
// keeps whatever dimensions were set at session creation time.
func watchResize(_ context.Context, _ *websocket.Conn) {
}
