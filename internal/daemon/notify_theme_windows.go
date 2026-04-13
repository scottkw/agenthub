//go:build windows

package daemon

import "github.com/scottkw/agenthub/internal/pty"

// signalThemeChange is a no-op on Windows. SIGUSR2 does not exist
// on Windows; OpenCode on Windows does not support signal-based
// theme refresh.
func signalThemeChange(_ *pty.Session) error {
	return nil
}
