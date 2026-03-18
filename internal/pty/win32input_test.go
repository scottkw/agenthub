package pty

import (
	"bytes"
	"testing"
)

// TestWin32Input_BasicKeystrokes verifies that a key-down event for 'a'
// (Unicode char 0x61) produces byte 0x61.
// Sequence: ESC [ 65 ; 30 ; 97 ; 1 ; 0 ; 1 _
// Fields: Vk=65, Sc=30, Uc=97 ('a'), Kd=1 (down), Cs=0, Rep=1
func TestWin32Input_BasicKeystrokes(t *testing.T) {
	input := []byte("\x1b[65;30;97;1;0;1_")
	out, remainder := parseWin32Chunk(input)
	if !bytes.Equal(out, []byte{0x61}) {
		t.Errorf("expected [0x61], got %v", out)
	}
	if len(remainder) != 0 {
		t.Errorf("expected empty remainder, got %v", remainder)
	}
}

// TestWin32Input_KeyUpFiltered verifies that key-up events (Kd=0) are silently
// dropped and produce no output.
// Sequence: ESC [ 65 ; 30 ; 97 ; 0 ; 0 ; 1 _  (Kd=0)
func TestWin32Input_KeyUpFiltered(t *testing.T) {
	input := []byte("\x1b[65;30;97;0;0;1_")
	out, remainder := parseWin32Chunk(input)
	if len(out) != 0 {
		t.Errorf("expected no output for key-up, got %v", out)
	}
	if len(remainder) != 0 {
		t.Errorf("expected empty remainder, got %v", remainder)
	}
}

// TestWin32Input_Passthrough verifies that bytes without ESC pass through
// unchanged.
func TestWin32Input_Passthrough(t *testing.T) {
	input := []byte("hello")
	out, remainder := parseWin32Chunk(input)
	if !bytes.Equal(out, []byte("hello")) {
		t.Errorf("expected 'hello', got %q", out)
	}
	if len(remainder) != 0 {
		t.Errorf("expected empty remainder, got %v", remainder)
	}
}

// TestWin32Input_IncompleteSequence verifies that a partial ESC sequence at
// the end of the buffer is held back as a remainder (not emitted).
func TestWin32Input_IncompleteSequence(t *testing.T) {
	input := []byte("abc\x1b[65;30")
	out, remainder := parseWin32Chunk(input)
	if !bytes.Equal(out, []byte("abc")) {
		t.Errorf("expected 'abc', got %q", out)
	}
	if !bytes.Equal(remainder, []byte("\x1b[65;30")) {
		t.Errorf("expected '\\x1b[65;30' remainder, got %q", remainder)
	}
}

// TestWin32Input_MixedContent verifies that literal bytes before and after a
// win32-input-mode sequence are passed through and the key char is interpolated.
func TestWin32Input_MixedContent(t *testing.T) {
	input := []byte("pre\x1b[65;30;97;1;0;1_post")
	out, remainder := parseWin32Chunk(input)
	expected := append([]byte("pre"), append([]byte{0x61}, []byte("post")...)...)
	if !bytes.Equal(out, expected) {
		t.Errorf("expected %q, got %q", expected, out)
	}
	if len(remainder) != 0 {
		t.Errorf("expected empty remainder, got %v", remainder)
	}
}
