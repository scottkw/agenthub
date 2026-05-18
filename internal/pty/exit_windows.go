//go:build windows

// v3.3.1 Phase 109 UAT finding / SHELL-12-windows parity:
//
// Like Linux (Phase 110), Windows ConPTY does NOT cleanly surface io.EOF on
// the PTY master after the child process exits — the relay Hub.Run read loop
// blocks indefinitely and the daemon's session:exit event never fires. The
// GUI tab and TUI list entry therefore stay open after a clean `exit` at the
// shell prompt, violating the v3.3 Phase 107 SHELL-12 auto-close contract.
//
// This detector mirrors exit_linux.go's structure but uses Windows-native
// WaitForSingleObject + GetExitCodeProcess for polling.
package pty

import (
	"time"

	"golang.org/x/sys/windows"
)

// winExitPollInterval matches the Linux cadence (linuxExitPollInterval).
const winExitPollInterval = 100 * time.Millisecond

// startExitDetector spawns a Windows-only goroutine that polls
// WaitForSingleObject for the session's child process and closes the PTY
// master when the child exits.
//
// Locking and lifecycle mirror the Linux detector at exit_linux.go:
//   - IsKilled gate at every tick — killSession owns the kill path
//   - On natural exit: SetExitCode → CancelContext → closePTY (sync.Once)
//   - closePTY is the gated helper added during Phase 110 race fix so a
//     racing killSession (cleanup_windows.go) cannot double-close go-pty's
//     conPtyMaster handle.
//
// Build-time symmetry: exit_other.go's signature is `func(s *Session)` with
// build tag `!linux && !windows`, so native.go's `startExitDetector(sess)`
// call resolves cleanly on every platform.
func startExitDetector(s *Session) {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	// Capture PID once, outside the goroutine.
	pid := s.cmd.Process.Pid

	go func() {
		// Open a handle with SYNCHRONIZE (for WaitForSingleObject) +
		// PROCESS_QUERY_LIMITED_INFORMATION (for GetExitCodeProcess). We
		// open our own handle rather than reuse cmd.Process's so go-pty's
		// internal waitOnContext goroutine and our detector don't race on
		// the same OS handle. Closing OUR handle on detector exit does
		// not affect the parent process's ability to Wait() — kernel
		// handle refcounting keeps the process object alive until the
		// last handle is closed.
		handle, err := windows.OpenProcess(
			windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION,
			false,
			uint32(pid),
		)
		if err != nil {
			// Process already gone or insufficient permissions — silently
			// return; the relay-side path will eventually drain.
			return
		}
		defer windows.CloseHandle(handle)

		ticker := time.NewTicker(winExitPollInterval)
		defer ticker.Stop()

		for range ticker.C {
			// Bail if kill path owns this session.
			if s.IsKilled() {
				return
			}

			// Non-blocking wait — 0 ms timeout. Returns WAIT_OBJECT_0 if
			// the process has exited, WAIT_TIMEOUT if still running.
			result, err := windows.WaitForSingleObject(handle, 0)
			if err != nil {
				// Handle invalidated or other syscall failure — let the
				// relay-side path drain and clean up.
				return
			}
			if result != windows.WAIT_OBJECT_0 {
				// Still running (WAIT_TIMEOUT) or abandoned mutex etc.
				continue
			}

			// Process exited. TOCTOU re-check mirrors exit_linux.go's
			// WR-01 mitigation: if MarkKilled fired during the syscall,
			// the kill path owns the state transition.
			if s.IsKilled() {
				return
			}

			// Read the raw exit code. On Windows, normal exits use the
			// program's own exit code (e.g. 0 for clean `exit`); abnormal
			// termination surfaces STATUS_* values (large uint32). The
			// v3.3 Phase 107 SHELL-12 frontend `autoCloseRef` logic
			// treats ANY natural exit as "close the tab" (per project
			// memory project_shell_exit_toast_descoped), so we do not
			// need to normalize signaled exits the way Linux does.
			var raw uint32
			if err := windows.GetExitCodeProcess(handle, &raw); err != nil {
				// Process gone but GetExitCodeProcess failed — defensive 0.
				s.SetExitCode(0)
			} else {
				s.SetExitCode(int(raw))
			}

			// Sequence locked from exit_linux.go: cache exit code (above),
			// cancel context, close PTY to unblock the relay read loop.
			s.CancelContext()
			// closePTY is sync.Once-gated so a racing killSession (in
			// cleanup_windows.go) cannot double-close the ConPTY handle.
			s.closePTY()
			return
		}
	}()
}
