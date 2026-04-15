//go:build windows

package attach

import (
	"context"

	"github.com/coder/websocket"
)

// WatchResize is a no-op on Windows. SIGWINCH is not available on Windows,
// so resize propagation is not supported in this release. The remote PTY
// keeps whatever dimensions were set at session creation time.
func WatchResize(_ context.Context, _ *websocket.Conn) {
}
