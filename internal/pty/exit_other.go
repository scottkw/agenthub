//go:build !linux

// Phase 110 / PTY-02: non-Linux no-op stub for the exit detector.
//
// macOS and the BSDs surface io.EOF on the PTY master after the child
// process exits — the existing Hub.Run read loop terminates naturally and
// no detector is needed.
//
// Windows ConPTY exit semantics are out of scope for Phase 110; a future
// phase will close that gap separately.
package pty

// startExitDetector is a compile-time no-op on non-Linux targets. The
// signature mirrors exit_linux.go so the wire-up in native.go is platform-
// agnostic.
func startExitDetector(_ *Session) {}
