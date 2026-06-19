//go:build !linux && !windows

package pty

// ReapNaturalExit reaps a naturally-exited child (PTY master EOF already
// observed) and caches its real exit code from ProcessState.
//
// On macOS/BSD go-pty installs NO waitOnContext goroutine — that exists only on
// Windows (see go-pty cmd_windows.go) — and these platforms run no exit detector
// (reap is driven by master EOF; exit_other.go is a no-op). So nothing calls
// cmd.Wait() on the natural-exit path and ProcessState is never populated. The
// engine would then read a cached -1 and normalize it to 0 (D-10), making the
// Hub's CARD-08 stopped-err state unreachable for any non-zero natural exit.
//
// Safety: the child is already dead (EOF), so cmd.Wait() returns immediately
// without blocking. go-pty's wait() guards a double Wait ("Wait was already
// called"), and the kill path (killSession) is gated out by the engine's
// IsKilled() check before this runs — so this call races nothing. ProcessState
// is read in the same goroutine after Wait returns (happens-after; no race).
func (s *Session) ReapNaturalExit() {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()
	if cmd == nil {
		return
	}
	_ = cmd.Wait()
	if cmd.ProcessState != nil {
		s.SetExitCode(cmd.ProcessState.ExitCode())
	}
}
