//go:build !windows

package daemon

import (
	"os/exec"
	"syscall"
)

// startDetachedDaemon spawns the daemon as a detached child process on Unix.
// Setsid creates a new session so the daemon is not killed when the parent exits.
func startDetachedDaemon(exe string) error {
	cmd := exec.Command(exe, "daemon")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
