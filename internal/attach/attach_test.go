package attach

import (
	"bytes"
	"testing"
)

func TestLockedWriter(t *testing.T) {
	var buf bytes.Buffer
	lw := NewLockedWriter(&buf)
	n, err := lw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if buf.String() != "hello" {
		t.Errorf("expected 'hello', got %q", buf.String())
	}
}

func TestMakeClientResizeFrame(t *testing.T) {
	frame := MakeClientResizeFrame(80, 24)
	if len(frame) != 5 {
		t.Fatalf("expected 5 bytes, got %d", len(frame))
	}
	// MsgResize2 = 0x11
	if frame[0] != 0x11 {
		t.Errorf("expected first byte 0x11 (MsgResize2), got 0x%02x", frame[0])
	}
	// 80 = 0x0050, 24 = 0x0018
	cols := uint16(frame[1])<<8 | uint16(frame[2])
	rows := uint16(frame[3])<<8 | uint16(frame[4])
	if cols != 80 {
		t.Errorf("expected cols=80, got %d", cols)
	}
	if rows != 24 {
		t.Errorf("expected rows=24, got %d", rows)
	}
}
