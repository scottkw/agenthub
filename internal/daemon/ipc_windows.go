//go:build windows

package daemon

import (
	"context"
	"net"
	"os"

	winio "github.com/tailscale/go-winio"
)

func listenDaemonSocket(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}

func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, path)
}

func removeDaemonSocket(path string) error {
	if isWindowsNamedPipe(path) {
		return nil
	}
	return os.Remove(path)
}
