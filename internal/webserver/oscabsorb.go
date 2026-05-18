package webserver

import (
	"bytes"
	"log/slog"
)

// Phase 111 / Issue #54: streaming state machine that absorbs xterm.js's
// SYNTHESIZED replies to terminal queries (OSC 10/11 color queries and DA1
// device-attributes queries) emitted by the browser, BEFORE they reach the
// PTY's stdin.
//
// Three envelope shapes are absorbed:
//
//	OSC 10 FG color reply : "\x1b]10;rgb:RRRR/GGGG/BBBB" + (ST | BEL)
//	OSC 11 BG color reply : "\x1b]11;rgb:RRRR/GGGG/BBBB" + (ST | BEL)
//	DA1 reply             : "\x1b[?<digits-and-semicolons>c"
//
//	where ST = "\x1b\\" and BEL = "\x07".
//
// All other input (user keystrokes, arrow keys, function keys, OSC 52 clipboard,
// OSC 8 hyperlinks, bracketed-paste markers, …) is forwarded byte-for-byte.
//
// The absorber is per-WebSocket-subscriber. It MUST persist across MsgInput
// frames because envelopes may split across frames. The zero-value
// (&InputAbsorber{}) is the legitimate "outside, empty buffers" starting state
// — do not add a constructor.

const maxEnvelopeBytes = 4096

type absorbState int

const (
	stateOutside absorbState = iota
	stateGotEsc
	stateInOSC
	stateInOSCSeenEsc
	stateInCSI
)

// InputAbsorber is a streaming state machine that filters OSC 10/11 and DA1
// replies out of the browser→PTY input stream.
//
// The Filter method is the only public surface. Single-goroutine use only
// (the WS read pump goroutine).
type InputAbsorber struct {
	state  absorbState
	oscBuf []byte // body of an in-progress OSC envelope (after "\x1b]", excludes terminator)
	csiBuf []byte // body of an in-progress CSI envelope (after "\x1b[", includes the final byte once seen)
}

// Filter consumes in and returns the bytes that should be forwarded to the PTY.
// An empty (but non-nil) slice means "all input was absorbed; write nothing."
func (a *InputAbsorber) Filter(in []byte) []byte {
	out := make([]byte, 0, len(in))
	for _, b := range in {
		switch a.state {
		case stateOutside:
			if b == 0x1b { // ESC
				a.state = stateGotEsc
			} else {
				out = append(out, b)
			}

		case stateGotEsc:
			switch b {
			case ']':
				a.state = stateInOSC
				a.oscBuf = a.oscBuf[:0]
			case '[':
				a.state = stateInCSI
				a.csiBuf = a.csiBuf[:0]
			default:
				// Bare ESC followed by ordinary byte (Alt-key combo, vim cmd-mode,
				// or anything that isn't an OSC/CSI introducer). Flush both bytes
				// as passthrough.
				out = append(out, 0x1b, b)
				a.state = stateOutside
			}

		case stateInOSC:
			switch b {
			case 0x07: // BEL terminator
				out = a.completeOSC(out)
				a.state = stateOutside
			case 0x1b: // possible ST terminator (next byte must be '\')
				a.state = stateInOSCSeenEsc
			default:
				if len(a.oscBuf) >= maxEnvelopeBytes {
					// Overflow guard: flush conservatively as passthrough; do not
					// silently drop bytes. Reset state and re-process this byte
					// from stateOutside (CLAUDE.md §"Silent fallbacks").
					out = append(out, 0x1b, ']')
					out = append(out, a.oscBuf...)
					out = append(out, b)
					a.oscBuf = a.oscBuf[:0]
					a.state = stateOutside
				} else {
					a.oscBuf = append(a.oscBuf, b)
				}
			}

		case stateInOSCSeenEsc:
			if b == '\\' {
				// Valid ST terminator — envelope complete.
				out = a.completeOSC(out)
				a.state = stateOutside
			} else {
				// Inner ESC was NOT part of a valid ST terminator. Flush the
				// in-progress OSC as passthrough, then re-process the ESC plus
				// the current byte as a new sequence.
				out = append(out, 0x1b, ']')
				out = append(out, a.oscBuf...)
				out = append(out, 0x1b, b)
				a.oscBuf = a.oscBuf[:0]
				a.state = stateOutside
			}

		case stateInCSI:
			if len(a.csiBuf) >= maxEnvelopeBytes {
				// Overflow guard — same conservative behavior as OSC overflow.
				out = append(out, 0x1b, '[')
				out = append(out, a.csiBuf...)
				out = append(out, b)
				a.csiBuf = a.csiBuf[:0]
				a.state = stateOutside
				break
			}
			a.csiBuf = append(a.csiBuf, b)
			// CSI final byte is the first byte in the range 0x40–0x7E that is
			// NOT a parameter-byte ('0'–'9', ';', ':') and NOT an intermediate
			// byte (0x20–0x2F). The simplest robust check: any byte ≥0x40 that
			// is not the leading '?' / '>' / '<' / '=' private-marker (which
			// only appears as the first byte) terminates the CSI sequence.
			if b >= 0x40 && b <= 0x7E {
				out = a.completeCSI(out)
				a.csiBuf = a.csiBuf[:0]
				a.state = stateOutside
			}
		}
	}
	return out
}

// completeOSC decides whether the just-finished OSC envelope should be absorbed
// (OSC 10 or OSC 11 color reply) or flushed as passthrough (OSC 52, OSC 8,
// anything else). Reconstructs with the BEL terminator on passthrough — both
// terminators are accepted by OSC consumers, so the canonicalization is safe.
func (a *InputAbsorber) completeOSC(out []byte) []byte {
	code, _, _ := bytes.Cut(a.oscBuf, []byte{';'})
	codeStr := string(code)
	if codeStr == "10" || codeStr == "11" {
		slog.Debug("oscabsorb: absorbed OSC envelope",
			slog.String("code", codeStr),
			slog.Int("len_envelope", len(a.oscBuf)+3)) // +"\x1b]" + terminator
		a.oscBuf = a.oscBuf[:0]
		return out
	}
	out = append(out, 0x1b, ']')
	out = append(out, a.oscBuf...)
	out = append(out, 0x07)
	a.oscBuf = a.oscBuf[:0]
	return out
}

// completeCSI decides whether the just-finished CSI envelope is a DA1 reply
// ("\x1b[?...c") and should be absorbed, or anything else (arrow keys, function
// keys, bracketed-paste markers, …) and should be flushed as passthrough.
// csiBuf includes the final byte (the byte that terminated the sequence).
func (a *InputAbsorber) completeCSI(out []byte) []byte {
	n := len(a.csiBuf)
	if n >= 2 && a.csiBuf[0] == '?' && a.csiBuf[n-1] == 'c' {
		slog.Debug("oscabsorb: absorbed DA1 envelope",
			slog.String("code", "DA1"),
			slog.Int("len_envelope", n+2)) // +"\x1b["
		return out
	}
	out = append(out, 0x1b, '[')
	out = append(out, a.csiBuf...)
	return out
}
