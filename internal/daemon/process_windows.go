//go:build windows

package daemon

import (
	"os/exec"
	"syscall"
)

// startDetachedDaemon spawns the daemon as a detached child process on Windows.
// CREATE_NEW_PROCESS_GROUP prevents the daemon from being killed when the parent exits.
func startDetachedDaemon(exe string) error {
	cmd := exec.Command(exe, "daemon")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
