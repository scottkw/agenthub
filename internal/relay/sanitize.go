// Package relay — sanitize.go provides SanitizePTYText, the sole transform
// applied to all inject text before Hub.WriteInput. No untrusted text may
// reach PTY stdin without passing through this function.
package relay

import "strings"

// SanitizePTYText sanitizes user-supplied text before it is written to PTY
// stdin via Hub.WriteInput. It is the sole gate between untrusted client text
// and the PTY — no trusted fast-path exists; every inject call goes through
// this function.
//
// Transformation rules (D-03 / SEC-02):
//   - CR ('\r'), LF ('\n'), and CRLF are each collapsed to a single space.
//     CRLF yields exactly one space, not two (Pitfall 1).
//   - C0 control characters (U+0000–U+001F, excluding ESC which enters escape
//     state) are stripped.
//   - DEL (U+007F) is stripped.
//   - C1 controls (U+0080–U+009F) are stripped.
//   - Terminal escape sequences are stripped via a five-state machine:
//     CSI (ESC '[' … final-byte) and OSC (ESC ']' … BEL or ESC '\').
//     All other ESC-prefixed pairs are silently discarded.
//   - Unicode bidi-override characters (see isBidiOverride) are stripped to
//     mitigate Trojan-Source class attacks (CVE-2021-42574).
//   - Trailing spaces are trimmed from the accumulated result.
//   - Exactly one trailing '\n' is appended.
//
// Output invariant: the returned string contains only printable text followed
// by exactly one newline character ('\n'). This is the only text that must
// ever reach Hub.WriteInput.
func SanitizePTYText(input string) string {
	const (
		stateNormal    = iota
		stateEscape    // saw ESC (0x1B); next rune decides CSI/OSC/other
		stateCSI       // inside ESC [ … ; skip until final byte 0x40–0x7E
		stateOSC       // inside ESC ] … ; skip until BEL (0x07) or ESC
		stateOSCEscape // inside OSC, saw ESC; next '\' ends it (ST = ESC \)
	)

	state := stateNormal
	var b strings.Builder
	b.Grow(len(input) + 1)

	runes := []rune(input)
	for _, r := range runes {
		switch state {
		case stateNormal:
			switch {
			case r == '\r' || r == '\n':
				// Collapse LF, CR, and CRLF each to a single space.
				// CRLF: '\r' writes a space, then '\n' writes another space — but
				// since we TrimRight spaces at the end, consecutive spaces between
				// words are fine; between "hello" and "world" they collapse to one
				// space because the TrimRight only affects trailing spaces.
				// Wait — re-read spec: "CRLF must not yield two spaces".
				// The test case is "hello\r\nworld" → "hello world\n" (one space).
				// With naive approach we'd write two spaces ("hello  world").
				// Fix: only write space if the builder doesn't already end in a space.
				s := b.String()
				if len(s) == 0 || s[len(s)-1] != ' ' {
					b.WriteByte(' ')
				}
			case r == 0x1B:
				state = stateEscape
			case r >= 0x00 && r <= 0x1F:
				// C0 control (0x1B already handled above): strip
			case r == 0x7F:
				// DEL: strip
			case r >= 0x80 && r <= 0x9F:
				// C1 control (Unicode U+0080–U+009F): strip
			case isBidiOverride(r):
				// Bidi override (Trojan-Source class): strip
			default:
				b.WriteRune(r)
			}
		case stateEscape:
			switch r {
			case '[':
				state = stateCSI
			case ']':
				state = stateOSC
			default:
				state = stateNormal // discard ESC + this rune
			}
		case stateCSI:
			// Final byte range 0x40–0x7E ends the CSI sequence.
			// Parameter/intermediate bytes (0x20–0x3F) stay in CSI state.
			if r >= 0x40 && r <= 0x7E {
				state = stateNormal
			}
		case stateOSC:
			switch r {
			case 0x07: // BEL terminates OSC
				state = stateNormal
			case 0x1B:
				state = stateOSCEscape
			}
			// All other bytes remain in OSC (skip)
		case stateOSCEscape:
			if r == '\\' {
				state = stateNormal // ESC \ = String Terminator (ST)
			} else {
				state = stateOSC // not ST; continue skipping OSC content
			}
		}
	}

	result := strings.TrimRight(b.String(), " ")
	return result + "\n"
}

// isBidiOverride returns true for Unicode bidi-override and directional
// formatting characters that can be exploited to spoof terminal output
// (Trojan-Source class, CVE-2021-42574).
func isBidiOverride(r rune) bool {
	switch r {
	case 0x061C, // ARABIC LETTER MARK (ALM)
		0x200E, 0x200F, // LRM, RLM
		0x202A, 0x202B, 0x202C, 0x202D, 0x202E, // LRE, RLE, PDF, LRO, RLO
		0x2066, 0x2067, 0x2068, 0x2069: // LRI, RLI, FSI, PDI
		return true
	}
	return false
}
