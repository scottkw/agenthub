package webserver

import (
	"bytes"
	"testing"
)

// TestInputAbsorber_AbsorbAndPassthrough covers single-call absorb cases (filter
// returns empty for OSC 10/11/DA1 envelopes) and passthrough cases (filter
// returns input unchanged for keystrokes, OSC 52, OSC 8, bracketed paste, etc.).
//
// Each subtest constructs a FRESH absorber to avoid state bleed.
func TestInputAbsorber_AbsorbAndPassthrough(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want []byte
	}{
		// --- ABSORB ---
		{"A1_OSC10_ST", []byte("\x1b]10;rgb:cccc/cccc/cccc\x1b\\"), []byte{}},
		{"A2_OSC10_BEL", []byte("\x1b]10;rgb:cccc/cccc/cccc\x07"), []byte{}},
		{"A3_OSC11_ST", []byte("\x1b]11;rgb:cccc/cccc/cccc\x1b\\"), []byte{}},
		{"A4_OSC11_BEL", []byte("\x1b]11;rgb:cccc/cccc/cccc\x07"), []byte{}},
		{"A5_DA1_xterm256", []byte("\x1b[?1;2c"), []byte{}},
		{"A6_DA1_multiparam", []byte("\x1b[?62;4;9;22c"), []byte{}},

		// --- PASSTHROUGH ---
		{"P1_keystrokes", []byte("ls\r"), []byte("ls\r")},
		{"P2a_arrow_up", []byte("\x1b[A"), []byte("\x1b[A")},
		{"P2b_arrow_down", []byte("\x1b[B"), []byte("\x1b[B")},
		{"P2c_arrow_right", []byte("\x1b[C"), []byte("\x1b[C")},
		{"P2d_arrow_left", []byte("\x1b[D"), []byte("\x1b[D")},
		{"P3_F5", []byte("\x1b[15~"), []byte("\x1b[15~")},
		{"P4_CtrlRight", []byte("\x1b[1;5C"), []byte("\x1b[1;5C")},
		{"P5_AltA", []byte("\x1ba"), []byte("\x1ba")},
		{"P6_OSC52_clipboard", []byte("\x1b]52;c;SGVsbG8=\x1b\\"), []byte("\x1b]52;c;SGVsbG8=\x07")},
		{"P7_OSC8_hyperlink", []byte("\x1b]8;;https://example.com\x1b\\"), []byte("\x1b]8;;https://example.com\x07")},
		{"P8a_bracketed_paste_start", []byte("\x1b[200~"), []byte("\x1b[200~")},
		{"P8b_bracketed_paste_end", []byte("\x1b[201~"), []byte("\x1b[201~")},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := &InputAbsorber{}
			got := a.Filter(tc.in)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("case %q: input=%q want=%q got=%q", tc.name, tc.in, tc.want, got)
			}
		})
	}
}

// TestInputAbsorber_BareEscDelayedPassthrough is Test P9: a bare ESC followed
// by a non-control byte must round-trip across two Filter calls.
//
// Call 1: Filter("\x1b")        → "" (state moves to stateGotEsc; ESC buffered)
// Call 2: Filter(":")           → "\x1b:" (state returns to stateOutside)
func TestInputAbsorber_P9_BareEscThenColon(t *testing.T) {
	a := &InputAbsorber{}
	got1 := a.Filter([]byte("\x1b"))
	if len(got1) != 0 {
		t.Errorf("call1: want empty (ESC buffered), got %q", got1)
	}
	got2 := a.Filter([]byte(":"))
	if !bytes.Equal(got2, []byte("\x1b:")) {
		t.Errorf("call2: want %q, got %q", "\x1b:", got2)
	}
}

// TestInputAbsorber_S1_SplitAfterIntroducer — OSC 11 reply split at "\x1b]" boundary.
func TestInputAbsorber_S1_SplitAfterIntroducer(t *testing.T) {
	a := &InputAbsorber{}
	out1 := a.Filter([]byte("\x1b]"))
	out2 := a.Filter([]byte("11;rgb:cccc/cccc/cccc\x1b\\"))
	if total := len(out1) + len(out2); total != 0 {
		t.Errorf("want 0 total bytes emitted; got %d (call1=%q call2=%q)", total, out1, out2)
	}
}

// TestInputAbsorber_S2_SplitMidBody — OSC 11 reply split in the middle of the body.
func TestInputAbsorber_S2_SplitMidBody(t *testing.T) {
	a := &InputAbsorber{}
	out1 := a.Filter([]byte("\x1b]11;rgb:cccc/"))
	out2 := a.Filter([]byte("cccc/cccc\x1b\\"))
	if total := len(out1) + len(out2); total != 0 {
		t.Errorf("want 0 total bytes emitted; got %d (call1=%q call2=%q)", total, out1, out2)
	}
}

// TestInputAbsorber_S3_SplitBeforeTerminator — OSC 11 split right before the ST terminator.
func TestInputAbsorber_S3_SplitBeforeTerminator(t *testing.T) {
	a := &InputAbsorber{}
	out1 := a.Filter([]byte("\x1b]11;rgb:cccc/cccc/cccc"))
	out2 := a.Filter([]byte("\x1b\\"))
	if total := len(out1) + len(out2); total != 0 {
		t.Errorf("want 0 total bytes emitted; got %d (call1=%q call2=%q)", total, out1, out2)
	}
}

// TestInputAbsorber_S4_SplitBetweenEscAndBackslash — OSC 11 split between the ESC and `\` of ST.
func TestInputAbsorber_S4_SplitBetweenEscAndBackslash(t *testing.T) {
	a := &InputAbsorber{}
	out1 := a.Filter([]byte("\x1b]11;rgb:cccc/cccc/cccc\x1b"))
	out2 := a.Filter([]byte("\\"))
	if total := len(out1) + len(out2); total != 0 {
		t.Errorf("want 0 total bytes emitted; got %d (call1=%q call2=%q)", total, out1, out2)
	}
}

// TestInputAbsorber_S5_DA1SplitAtQuestionBoundary — DA1 reply split between "\x1b[" and "?1;2c".
func TestInputAbsorber_S5_DA1SplitAtQuestionBoundary(t *testing.T) {
	a := &InputAbsorber{}
	out1 := a.Filter([]byte("\x1b["))
	out2 := a.Filter([]byte("?1;2c"))
	if total := len(out1) + len(out2); total != 0 {
		t.Errorf("want 0 total bytes emitted; got %d (call1=%q call2=%q)", total, out1, out2)
	}
}

// TestInputAbsorber_M1_MixedTraffic — keystrokes interleaved with an OSC 11 reply.
// Input: "ls\r\x1b]11;rgb:cccc/cccc/cccc\x1b\\pwd\r"
// Want : "ls\rpwd\r"
func TestInputAbsorber_M1_MixedTraffic(t *testing.T) {
	a := &InputAbsorber{}
	got := a.Filter([]byte("ls\r\x1b]11;rgb:cccc/cccc/cccc\x1b\\pwd\r"))
	want := []byte("ls\rpwd\r")
	if !bytes.Equal(got, want) {
		t.Errorf("M1 mixed traffic: want %q got %q", want, got)
	}
}

// TestInputAbsorber_R1_OSCOverflowFlushedAsPassthrough — an unclosed OSC envelope
// flooded past maxEnvelopeBytes must be flushed as passthrough (not silently
// dropped), state reset to stateOutside, and subsequent normal input still flows.
func TestInputAbsorber_R1_OSCOverflowFlushedAsPassthrough(t *testing.T) {
	a := &InputAbsorber{}
	// Start an OSC envelope and flood it past the cap.
	huge := make([]byte, 0, maxEnvelopeBytes+128)
	huge = append(huge, '\x1b', ']')
	for i := 0; i < maxEnvelopeBytes+64; i++ {
		huge = append(huge, 'X')
	}
	got := a.Filter(huge)
	if len(got) == 0 {
		t.Fatalf("R1: expected flushed-as-passthrough bytes, got empty")
	}
	if !bytes.HasPrefix(got, []byte("\x1b]")) {
		prefixLen := 4
		if len(got) < prefixLen {
			prefixLen = len(got)
		}
		t.Errorf("R1: expected output to begin with the original introducer, got prefix %q", got[:prefixLen])
	}
	// State should be reset — subsequent normal keystrokes pass through cleanly.
	follow := a.Filter([]byte("ok\r"))
	if !bytes.Equal(follow, []byte("ok\r")) {
		t.Errorf("R1: subsequent keystrokes want %q got %q", "ok\r", follow)
	}
}

// TestInputAbsorber_R2_MalformedOSCInterruptedByNonStEsc — `\x1b]11;rgb:cccc\x1bA`
// the inner ESC is NOT followed by `\`, so this is not a valid ST terminator.
// Conservative behavior: flush the OSC body as passthrough and handle the new
// ESC sequence normally. Net round-trip MUST be the exact input.
func TestInputAbsorber_R2_MalformedOSCInterruptedByNonStEsc(t *testing.T) {
	a := &InputAbsorber{}
	in := []byte("\x1b]11;rgb:cccc\x1bA")
	got := a.Filter(in)
	if !bytes.Equal(got, in) {
		t.Errorf("R2 malformed OSC: want round-trip %q got %q", in, got)
	}
}
