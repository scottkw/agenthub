//go:build windows

package pty

import (
	"io"
)

// ParseWin32Input reads win32-input-mode sequences from r, decodes them into
// raw bytes, and writes the result to w.
//
// This function is the streaming entry point used on Windows when Windows
// Terminal has enabled win32-input-mode via ESC[?9001h.  It translates the
// verbose keyboard event sequences into the raw byte stream that the CLI
// application (e.g. claude, codex) expects.
func ParseWin32Input(r io.Reader, w io.Writer) error {
	var pending []byte
	buf := make([]byte, 4096)

	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := append(pending, buf[:n]...)
			out, remainder := parseWin32Chunk(chunk)
			pending = remainder
			if len(out) > 0 {
				if _, werr := w.Write(out); werr != nil {
					return werr
				}
			}
		}
		if err != nil {
			// Flush any remaining pending bytes on EOF/error.
			if len(pending) > 0 {
				_, _ = w.Write(pending)
			}
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
