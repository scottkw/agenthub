package relay

import (
	"bytes"
	"testing"
)

func TestMakeOutputFrame(t *testing.T) {
	data := []byte("hello")
	frame := MakeOutputFrame(data)
	if len(frame) != len(data)+1 {
		t.Fatalf("expected length %d, got %d", len(data)+1, len(frame))
	}
	if frame[0] != MsgOutput {
		t.Errorf("expected type byte 0x%02x, got 0x%02x", MsgOutput, frame[0])
	}
	if !bytes.Equal(frame[1:], data) {
		t.Errorf("payload mismatch: got %v, want %v", frame[1:], data)
	}
}

func TestMakeInputFrame(t *testing.T) {
	data := []byte("world")
	frame := MakeInputFrame(data)
	if frame[0] != MsgInput {
		t.Errorf("expected type byte 0x%02x, got 0x%02x", MsgInput, frame[0])
	}
	if !bytes.Equal(frame[1:], data) {
		t.Errorf("payload mismatch: got %v, want %v", frame[1:], data)
	}
}

func TestMakeResizeFrame(t *testing.T) {
	frame := MakeResizeFrame(80, 24)
	// Expected: [MsgResize, 0x00, 0x50, 0x00, 0x18]
	expected := []byte{MsgResize, 0x00, 0x50, 0x00, 0x18}
	if len(frame) != 5 {
		t.Fatalf("expected 5 bytes, got %d", len(frame))
	}
	if !bytes.Equal(frame, expected) {
		t.Errorf("resize frame mismatch: got %v, want %v", frame, expected)
	}
}

func TestParseFrameOutput(t *testing.T) {
	data := []byte("hello")
	frame := MakeOutputFrame(data)
	msgType, payload, err := ParseFrame(frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgType != MsgOutput {
		t.Errorf("expected type 0x%02x, got 0x%02x", MsgOutput, msgType)
	}
	if !bytes.Equal(payload, data) {
		t.Errorf("payload mismatch: got %v, want %v", payload, data)
	}
}

func TestParseFrameInput(t *testing.T) {
	data := []byte("world")
	frame := MakeInputFrame(data)
	msgType, payload, err := ParseFrame(frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgType != MsgInput {
		t.Errorf("expected type 0x%02x, got 0x%02x", MsgInput, msgType)
	}
	if !bytes.Equal(payload, data) {
		t.Errorf("payload mismatch: got %v, want %v", payload, data)
	}
}

func TestParseFrameResize(t *testing.T) {
	frame := MakeResizeFrame(80, 24)
	msgType, payload, err := ParseFrame(frame)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgType != MsgResize {
		t.Errorf("expected type 0x%02x, got 0x%02x", MsgResize, msgType)
	}
	// payload should be the 4 bytes of cols+rows
	expected := []byte{0x00, 0x50, 0x00, 0x18}
	if !bytes.Equal(payload, expected) {
		t.Errorf("resize payload mismatch: got %v, want %v", payload, expected)
	}
}

func TestParseFrameEmpty(t *testing.T) {
	_, _, err := ParseFrame([]byte{})
	if err == nil {
		t.Error("expected error on empty frame, got nil")
	}
}

func TestParseFrameSingleByte(t *testing.T) {
	// Single byte: type only, empty payload — should succeed
	msgType, payload, err := ParseFrame([]byte{MsgPing})
	if err != nil {
		t.Fatalf("unexpected error on single-byte frame: %v", err)
	}
	if msgType != MsgPing {
		t.Errorf("expected type 0x%02x, got 0x%02x", MsgPing, msgType)
	}
	if len(payload) != 0 {
		t.Errorf("expected empty payload, got %v", payload)
	}
}

func TestFrameRoundTrip(t *testing.T) {
	types := []struct {
		name  string
		frame []byte
		typ   byte
	}{
		{"output", MakeOutputFrame([]byte("data")), MsgOutput},
		{"input", MakeInputFrame([]byte("data")), MsgInput},
	}
	for _, tc := range types {
		t.Run(tc.name, func(t *testing.T) {
			msgType, payload, err := ParseFrame(tc.frame)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if msgType != tc.typ {
				t.Errorf("expected type 0x%02x, got 0x%02x", tc.typ, msgType)
			}
			if !bytes.Equal(payload, []byte("data")) {
				t.Errorf("payload mismatch")
			}
		})
	}
}
