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
//
// Safe to call cmd.Wait() here because MarkKilled() has been set before this
// function is called — the exit watcher goroutine checks IsKilled() and skips
// its own cmd.Wait(), so there is no concurrent caller.
func killSession(s *Session) error {
	if s.cmd == nil || s.cmd.Process == nil {
		s.closePTY()
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
	// Gated by closePTY() so a racing exit detector (Linux) cannot double-close.
	s.closePTY()

	// Wait for process to exit, with 2-second timeout.
	//
	// Phase 117 / PAPER-03: the Wait goroutine below is bounded-lifetime, not
	// a true leak. After SIGKILL the kernel terminates the process and Wait
	// returns; the goroutine then closes(done) and exits. On the rare
	// "process stuck in D-state" path (uninterruptible kernel I/O), the
	// goroutine may outlive killSession's return, but completes when the OS
	// eventually reaps the process. We do NOT block killSession on this —
	// the caller has its own SLA. Goroutine resource cost: one stack frame
	// (~4 KiB) for at most the duration of the stuck I/O.
	done := make(chan struct{})
	go func() {
		_ = s.cmd.Wait()
		// Cache exit code under our mutex so ExitCode() can read it safely.
		if s.cmd.ProcessState != nil {
			s.SetExitCode(s.cmd.ProcessState.ExitCode())
		}
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
			// Goroutine still running — process is stuck (e.g. D-state). It
			// will complete when the OS finally reaps the process; we return
			// to the caller now per the SLA. See goroutine-lifetime note above.
		}
	}

	// Cancel the context.
	if s.cancel != nil {
		s.cancel()
	}

	return nil
}
