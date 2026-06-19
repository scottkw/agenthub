//go:build windows

package pty

// ReapNaturalExit is a no-op on Windows. go-pty's waitOnContext goroutine
// (cmd_windows.go) plus the WaitForSingleObject exit detector (exit_windows.go)
// already reap the child and cache the real exit code before Hub.Done fires.
func (s *Session) ReapNaturalExit() {}
