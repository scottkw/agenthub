//go:build !windows

package pty

import "syscall"

// killSignal0 sends signal 0 to the given pid/pgid to test process existence.
// Returns nil if the process exists, non-nil if it does not.
func killSignal0(pid int) error {
	return syscall.Kill(pid, 0)
}
