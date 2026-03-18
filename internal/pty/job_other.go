//go:build !windows

package pty

import "os"

// assignJobObject is a no-op on non-Windows platforms.
func assignJobObject(sess *Session, proc *os.Process) {}
