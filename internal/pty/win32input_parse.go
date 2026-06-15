// Package pty — win32input_parse.go
// Stateless parser for win32-input-mode sequences.
// NO build tag: this file compiles on all platforms so the unit tests in
// win32input_test.go (also untagged) can run on any OS.
//
// Win32-input-mode background:
//   Windows Terminal enables win32-input-mode by writing ESC[?9001h to the
//   terminal.  In this mode, every key event is encoded as a CSI sequence:
//     ESC [ Vk ; Sc ; Uc ; Kd ; Cs ; Rep _
//   where:
//     Vk  = Virtual-Key code
//     Sc  = Scan code
//     Uc  = Unicode character (code point — this is what we emit)
//     Kd  = Key-down flag (1 = down, 0 = up)
//     Cs  = Control-key state bitmask
//     Rep = Repeat count
//
//   We emit Uc (as a UTF-8 encoded rune) only for key-down events (Kd == 1).
//   Key-up events and focus events (ESC[I / ESC[O) are silently dropped.

package pty

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf8"
)

// parseWin32Chunk parses a chunk of bytes that may contain win32-input-mode
// CSI sequences mixed with literal data.
//
// Returns:
//   - out:       bytes that should be forwarded to the application
//   - remainder: any incomplete ESC sequence at the end of data that must be
//     prepended to the next chunk before parsing again
func parseWin32Chunk(data []byte) (out []byte, remainder []byte) {
	var buf bytes.Buffer

	for len(data) > 0 {
		escIdx := bytes.IndexByte(data, 0x1b) // ESC
		if escIdx < 0 {
			// No ESC — pass everything through.
			buf.Write(data)
			return buf.Bytes(), nil
		}

		// Emit bytes before the ESC.
		if escIdx > 0 {
			buf.Write(data[:escIdx])
			data = data[escIdx:]
		}

		// data now starts with ESC.  Try to consume a complete CSI sequence.
		end, consumed, keyByte := findCSIEnd(data)
		if end < 0 {
			// Incomplete sequence — hold back for next call.
			return buf.Bytes(), data
		}

		if consumed {
			// Valid win32-input-mode sequence — emit the decoded key byte if any.
			if keyByte != 0 {
				buf.WriteByte(keyByte)
			}
		} else {
			// Not a recognised sequence — emit raw bytes as-is.
			buf.Write(data[:end])
		}
		data = data[end:]
	}

	return buf.Bytes(), nil
}

// findCSIEnd inspects data (which must start with ESC) for a complete CSI
// sequence ending with '_'.
//
// Returns:
//   - end:      number of bytes consumed (including trailing terminator)
//   - consumed: true if this was a recognised win32-input-mode sequence
//   - keyByte:  the byte to emit (0 = emit nothing)
//
// Returns end == -1 if the sequence is incomplete (need more data).
func findCSIEnd(data []byte) (end int, consumed bool, keyByte byte) {
	// Minimum: ESC [ params _  → at least 4 bytes
	if len(data) < 2 {
		return -1, false, 0
	}
	if data[1] != '[' {
		// ESC followed by something other than '[' — not a CSI; consume 2 bytes raw.
		return 2, false, 0
	}

	// Look for the CSI terminator.  Win32-input-mode uses '_' (0x5F).
	// Focus events use 'I' (0x49) and 'O' (0x4F).
	// We scan for any letter/punctuation that terminates a CSI sequence.
	for i := 2; i < len(data); i++ {
		c := data[i]
		if c >= 0x40 && c <= 0x7E {
			// Found a terminator.
			seq := data[:i+1]
			switch c {
			case '_':
				// Win32-input-mode keyboard event.
				kb, _ := decodeWin32Key(seq)
				return i + 1, true, kb
			case 'I', 'O':
				// Focus event — drop silently.
				return i + 1, true, 0
			default:
				// Unknown CSI sequence — pass through raw.
				return i + 1, false, 0
			}
		}
		// Still in parameter bytes — continue.
	}

	// No terminator found yet — sequence is incomplete.
	return -1, false, 0
}

// decodeWin32Key parses a win32-input-mode sequence of the form:
//
//	ESC [ Vk ; Sc ; Uc ; Kd ; Cs ; Rep _
//
// and returns the byte to emit and whether parsing succeeded.
// Returns (0, false) for key-up events or parse errors.
func decodeWin32Key(seq []byte) (byte, bool) {
	// Strip ESC [ prefix and _ suffix.
	if len(seq) < 4 || seq[0] != 0x1b || seq[1] != '[' || seq[len(seq)-1] != '_' {
		return 0, false
	}
	params := string(seq[2 : len(seq)-1])
	parts := strings.Split(params, ";")
	if len(parts) < 4 {
		return 0, false
	}

	// Field 3 (index 3) is Kd — key-down flag.
	kd, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil || kd != 1 {
		// Key-up or parse error — drop.
		return 0, false
	}

	// Field 2 (index 2) is Uc — Unicode code point.
	uc, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil || uc <= 0 {
		return 0, false
	}

	// Encode the Unicode code point as UTF-8.
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], rune(uc))
	if n == 1 {
		// ASCII — return single byte.
		return buf[0], true
	}
	// Multi-byte rune: for now return the first byte only.
	// Phase 2 will handle full UTF-8 sequences via a []byte return path.
	return buf[0], true
}
