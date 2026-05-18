//go:build !linux && !windows

// Phase 110 / PTY-02 (macOS + BSDs): no-op exit-detector stub.
//
// macOS and the BSDs cleanly surface io.EOF on the PTY master after the
// child process exits — the existing Hub.Run read loop terminates naturally
// and no detector is needed.
//
// Linux uses exit_linux.go (syscall.Wait4 polling — Phase 110).
// Windows uses exit_windows.go (WaitForSingleObject polling — v3.3.1
// Phase 109 UAT finding).
package pty

// startExitDetector is a compile-time no-op on macOS/BSD targets. The
// signature mirrors exit_linux.go and exit_windows.go so the wire-up in
// native.go is platform-agnostic.
func startExitDetector(_ *Session) {}
