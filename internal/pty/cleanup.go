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
// Note: The PTY is closed BEFORE calling Wait to prevent Wait from blocking
// indefinitely when the PTY master is still open.
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

	// Close the PTY master first — otherwise cmd.Wait() may block indefinitely
	// waiting for the PTY slave to be closed, even after the child has exited.
	if s.pty != nil {
		_ = s.pty.Close()
	}

	// Wait for process to exit, with 2-second timeout.
	// Use WaitForExit() which wraps cmd.Wait() in sync.Once to avoid racing
	// with the exit watcher goroutine in engine.go.
	done := make(chan struct{})
	go func() {
		s.WaitForExit()
		close(done)
	}()

	select {
	case <-done:
		// Exited cleanly.
	case <-time.After(2 * time.Second):
		// Force kill the process group.
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		// Wait again after SIGKILL.
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			// Give up — process is truly stuck.
		}
	}

	// Cancel the context.
	if s.cancel != nil {
		s.cancel()
	}

	return nil
}
