//go:build linux

package pty

// ReapNaturalExit is a no-op on Linux. The Wait4-polling exit detector
// (exit_linux.go) already reaps the child and caches the real exit code
// (POSIX 128+signal convention) before it closes the PTY master and Hub.Done
// fires. Calling cmd.Wait() here would race the detector's syscall-level reap.
func (s *Session) ReapNaturalExit() {}
