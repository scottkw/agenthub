//go:build !windows

package pty

import "io"

// ParseWin32Input is a no-op passthrough on non-Windows platforms.
// On POSIX systems, Windows Terminal is not used and win32-input-mode sequences
// never appear in the input stream, so we simply copy bytes unchanged.
func ParseWin32Input(r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}
