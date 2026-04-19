//go:build !windows

package pty

import (
	"syscall"
	"time"
)

// killSession terminates the process group associated with a session.
// On POSIX systems it sends SIGHUP to the process group for graceful shutdown,
// polls for exit up to 2 seconds, then sends SIGKILL if the process has not exited.
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
	// which calls cmd.Wait() and reaps the process.
	if s.cancel != nil {
		s.cancel()
	}

	// Poll for process group to exit (up to 2 seconds).
	// Signal 0 probes without sending — returns error when process is gone.
	if waitForProcessGroupExit(pgid, 2*time.Second) {
		return nil
	}

	// Still alive — force kill the entire process group.
	_ = syscall.Kill(-pgid, syscall.SIGKILL)

	// Wait up to 1 more second for SIGKILL to take effect.
	waitForProcessGroupExit(pgid, 1*time.Second)

	return nil
}

// waitForProcessGroupExit polls until the process group is gone or timeout expires.
// Returns true if the process group exited, false if still alive.
func waitForProcessGroupExit(pgid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pgid, 0); err != nil {
			return true // Process group is gone.
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
