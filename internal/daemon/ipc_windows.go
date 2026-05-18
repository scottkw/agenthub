//go:build windows

package daemon

import (
	"context"
	"net"
	"os"
	"time"

	winio "github.com/tailscale/go-winio"
)

func listenDaemonSocket(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}

// dialDaemonSocket dials the daemon's named pipe.
//
// If the caller-supplied ctx already has a deadline, it is honored verbatim.
// Otherwise we wrap a 2-second timeout to match the Unix variant's fast-fail
// behavior — without it, winio.tryDialPipe spins on ERROR_PIPE_BUSY indefinitely
// when the dial runs under context.Background() (which is what DaemonClient.doJSON
// produces today via http.NewRequest). Missing pipes (ERROR_FILE_NOT_FOUND) fail
// immediately regardless and aren't affected by this timeout.
func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
	}
	return winio.DialPipeContext(ctx, path)
}

func removeDaemonSocket(path string) error {
	if isWindowsNamedPipe(path) {
		return nil
	}
	return os.Remove(path)
}
