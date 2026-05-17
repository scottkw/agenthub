//go:build !windows

package daemon

import (
	"context"
	"net"
	"os"
	"time"
)

func listenDaemonSocket(path string) (net.Listener, error) {
	return net.Listen("unix", path)
}

func dialDaemonSocket(ctx context.Context, path string) (net.Conn, error) {
	return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", path)
}

func removeDaemonSocket(path string) error {
	return os.Remove(path)
}
