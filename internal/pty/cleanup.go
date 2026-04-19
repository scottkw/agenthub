//go:build !windows

package pty

import (
	"syscall"
	"time"
)

// killSession terminates the process group associated with a session.
// On POSIX systems it sends SIGHUP to the process group for graceful shutdown,
// cancels the context (triggers go-pty reaping via waitpid), polls for exit
// up to 2 seconds, then sends SIGKILL if the process has not exited.
//
// Context cancel is safe here because MarkKilled() has been called before this
// function — the exit watcher goroutine will skip cmd.Wait() and not race with
// go-pty's waitOnContext goroutine.
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

	// Cancel context — triggers go-pty's waitOnContext goroutine which calls
	// waitpid to reap the leader process. This is safe because MarkKilled()
	// prevents the exit watcher from racing with waitOnContext.
	if s.cancel != nil {
		s.cancel()
	}

	// Poll for process group to exit (up to 2 seconds).
	// go-pty's waitOnContext reaps the leader via waitpid; SIGHUP kills
	// the remaining group members. Signal-0 probe returns ESRCH once all
	// processes in the group are reaped.
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
