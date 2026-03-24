//go:build windows

package daemon

import (
	"fmt"
	"time"

	winio "github.com/tailscale/go-winio"
)

// cleanupStaleWindowsPipe probes a Windows named pipe to determine if a daemon
// is actively listening. Named pipes are kernel objects with no filesystem entry;
// there is nothing to os.Remove when no server is present.
func cleanupStaleWindowsPipe(path string) error {
	timeout := 500 * time.Millisecond
	conn, err := winio.DialPipe(path, &timeout)
	if err != nil {
		// Any dial error (pipe absent, timeout, busy) means we could not
		// establish a connection. Safe to allow a fresh daemon start.
		// Named pipes vanish when the last server handle closes — no cleanup needed.
		return nil
	}
	conn.Close()
	return fmt.Errorf("daemon already running at %s", path)
}
