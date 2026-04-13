//go:build !windows

package daemon

import (
	"syscall"

	"github.com/scottkw/agenthub/internal/pty"
)

// signalThemeChange sends SIGUSR2 to the session's child process,
// causing OpenCode to re-query the terminal palette and repaint.
func signalThemeChange(sess *pty.Session) error {
	return sess.Signal(syscall.SIGUSR2)
}
