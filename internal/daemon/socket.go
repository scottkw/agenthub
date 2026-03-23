package daemon

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// DefaultSocketPath returns the platform-appropriate Unix socket path for the
// agenthub daemon. On Windows it returns a named pipe path; on all other
// platforms it returns a path under the user config directory.
//
// The parent directory is created if it does not already exist.
func DefaultSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\agenthub-daemon`
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "agenthub")
	_ = os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "daemon.sock")
}

// ValidateSocketPath returns an error if path is too long for a Unix socket.
// The macOS/Linux sun_path limit is 104 bytes including the null terminator,
// so the maximum usable length is 103 characters.
func ValidateSocketPath(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if len(path) > 103 {
		return fmt.Errorf("socket path too long (%d chars, max 103): %s", len(path), path)
	}
	return nil
}

// CleanupStaleSocket probes whether anything is listening on path.
//
//   - If the file does not exist: returns nil (nothing to clean up).
//   - If the file exists but nothing is listening (stale / crash leftover):
//     removes the file and returns nil.
//   - If something is actively listening: returns an error containing
//     "already running" so callers can surface a helpful message.
func CleanupStaleSocket(path string) error {
	conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err != nil {
		// Could not connect — either no file or nothing listening.
		if os.IsNotExist(err) {
			return nil
		}
		// Any other dial error (ECONNREFUSED, timeout, etc.) means stale file.
		_ = os.Remove(path)
		return nil
	}
	// Connection succeeded — daemon is running.
	conn.Close()
	return fmt.Errorf("daemon already running at %s", path)
}
