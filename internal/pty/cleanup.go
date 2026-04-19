//go:build !windows

package pty

import (
	"syscall"
	"time"
)

// killSession terminates the process group associated with a session.
// On POSIX systems it sends SIGHUP to the process group for graceful shutdown,
// waits up to 2 seconds, then sends SIGKILL if the process has not exited.
//
// IMPORTANT: Does NOT call cmd.Wait() — go-pty's internal waitOnContext goroutine
// handles process reaping when the context is cancelled. Calling cmd.Wait()
// concurrently with waitOnContext causes a data race on go-pty internal state.
func killSession(s *Session) error {
	if s.cmd == nil || s.cmd.Process == nil {
		if s.pty != nil {
			_ = s.pty.Close()
		}
		if s.cancel != nil {
			s.cancel()
		}
		return nil
	}

	// go-pty sets Setsid: true, which means the child's PGID == its PID.
	pgid := s.cmd.Process.Pid

	// Graceful shutdown: SIGHUP to the entire process group.
	_ = syscall.Kill(-pgid, syscall.SIGHUP)

	// Close the PTY master — unblocks any pending reads on the PTY slave.
	if s.pty != nil {
		_ = s.pty.Close()
	}

	// Cancel the context — triggers go-pty's internal waitOnContext goroutine
	// which calls cmd.Wait() and populates ProcessState.
	if s.cancel != nil {
		s.cancel()
	}

	// Check if process exited after SIGHUP + context cancel.
	// Use signal 0 to probe — returns error if process is gone.
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(-pgid, 0); err != nil {
		return nil // Process group is gone.
	}

	// Still alive — force kill.
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	time.Sleep(100 * time.Millisecond)

	return nil
}
