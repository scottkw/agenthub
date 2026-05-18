//go:build linux

// Phase 110 / PTY-02: Linux-only Wait4-polling exit detector.
//
// On Linux, the PTY master Read() does NOT surface io.EOF after the child
// process exits (unlike macOS/BSD). Without an explicit close, the relay
// Hub.Run loop blocks indefinitely and the engine.go natural-exit goroutine
// never fires — Issue #57.
//
// This detector polls syscall.Wait4(pid, &status, WNOHANG) at 100 ms cadence.
// When the child has exited, it:
//   1. Caches the real exit code via sess.SetExitCode (POSIX 128+signal convention).
//   2. Calls sess.CancelContext() BEFORE closing the PTY — this fires
//      cmd.Cancel (Process.Kill, a no-op on an already-exited child) while
//      we still hold the PID, eliminating the PID-recycle race window
//      described in RESEARCH §10 Pitfall 2.
//   3. Closes the PTY master, which makes the blocked Hub.Run Read return,
//      which closes hub.Done(), which unblocks the engine.go:328
//      natural-exit goroutine that calls onExit.
//
// The detector returns silently if IsKilled() is true at any tick — killSession
// (cleanup.go) owns Wait() and PTY close on the kill path.
package pty

import (
	"syscall"
	"time"
)

// linuxExitPollInterval is the cadence at which the detector polls Wait4.
// 100 ms matches the existing engine.go defensive sleep and is well below
// the upstream Wails 500 ms poll latency, so users perceive sub-second
// response on natural exit. Wait4 with WNOHANG is one of the cheapest
// syscalls — kernel checks the ZOMBIE flag and returns immediately.
const linuxExitPollInterval = 100 * time.Millisecond

// startExitDetector spawns a Linux-only goroutine that polls Wait4 for the
// session's child process and closes the PTY master when the child exits.
// No-op on other platforms (see exit_other.go).
//
// Safe to call exactly once per session, immediately after the process has
// been started and the session registered. Must NOT call the cmd Wait method
// — that is killSession's exclusive turf (cleanup.go) and concurrent reap
// races with go-pty's exec.Cmd state machine.
func startExitDetector(s *Session) {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	// Capture PID once, outside the goroutine — avoids repeated field access
	// and ensures we always Wait4 on the original PID even if s.cmd is
	// mutated elsewhere.
	pid := s.cmd.Process.Pid

	go func() {
		ticker := time.NewTicker(linuxExitPollInterval)
		defer ticker.Stop()

		for range ticker.C {
			// If kill path owns this session, bail silently — killSession
			// will reap the child and close the PTY itself.
			if s.IsKilled() {
				return
			}

			var status syscall.WaitStatus
			wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
			if err != nil {
				// ECHILD or unexpected — let whichever path reaped the child
				// own state transition. Nothing more to do here.
				return
			}
			if wpid == 0 {
				// Child still running; poll again next tick.
				continue
			}

			// wpid > 0 — child has exited. Derive POSIX exit code.
			var exitCode int
			switch {
			case status.Exited():
				exitCode = status.ExitStatus()
			case status.Signaled():
				// POSIX 128+signal convention (e.g. SIGKILL=9 -> 137).
				exitCode = 128 + int(status.Signal())
			default:
				// Stopped/continued/other rare states — defensive 0.
				exitCode = 0
			}

			// Sequence (Q1 locked): cache exit code, cancel context (kills
			// cmd context while we still own the PID), close PTY to unblock
			// the relay read loop.
			s.SetExitCode(exitCode)
			s.CancelContext()

			s.mu.Lock()
			p := s.pty
			s.mu.Unlock()
			if p != nil {
				_ = p.Close()
			}
			return
		}
	}()
}
