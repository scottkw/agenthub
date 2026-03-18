package relay

import (
	"bytes"
	"sync"
	"testing"
)

func TestScrollbackAppendSnapshot(t *testing.T) {
	sb := NewScrollback(1024)
	sb.Append([]byte("hello"))
	sb.Append([]byte(" world"))
	snap := sb.Snapshot()
	expected := []byte("hello world")
	if !bytes.Equal(snap, expected) {
		t.Errorf("expected %q, got %q", expected, snap)
	}
}

func TestScrollbackTruncatesOldestBytes(t *testing.T) {
	// max capacity = 10 bytes
	sb := NewScrollback(10)
	sb.Append([]byte("12345"))  // 5 bytes
	sb.Append([]byte("67890"))  // 5 bytes — now at max
	sb.Append([]byte("ABCDE"))  // 5 more — should discard oldest 5

	snap := sb.Snapshot()
	if len(snap) > 10 {
		t.Errorf("snapshot exceeds max capacity: len=%d", len(snap))
	}
	// The newest data should be present
	if !bytes.Contains(snap, []byte("ABCDE")) {
		t.Errorf("expected newest data 'ABCDE' in snapshot %q", snap)
	}
}

func TestScrollbackTruncateAtExactBoundary(t *testing.T) {
	sb := NewScrollback(5)
	sb.Append([]byte("12345"))
	snap := sb.Snapshot()
	if !bytes.Equal(snap, []byte("12345")) {
		t.Errorf("expected %q, got %q", "12345", snap)
	}
	// Appending one more byte should drop the oldest
	sb.Append([]byte("X"))
	snap = sb.Snapshot()
	if len(snap) > 5 {
		t.Errorf("snapshot exceeds max: len=%d", len(snap))
	}
	if snap[len(snap)-1] != 'X' {
		t.Errorf("expected last byte 'X', got %q", snap[len(snap)-1])
	}
}

func TestScrollbackSnapshotIsACopy(t *testing.T) {
	sb := NewScrollback(1024)
	sb.Append([]byte("original"))
	snap := sb.Snapshot()
	// Mutate snapshot
	snap[0] = 'X'
	// Snapshot again — should still be "original"
	snap2 := sb.Snapshot()
	if snap2[0] == 'X' {
		t.Error("snapshot mutation affected internal buffer")
	}
	if !bytes.Equal(snap2, []byte("original")) {
		t.Errorf("expected %q, got %q", "original", snap2)
	}
}

func TestScrollbackConcurrentSafety(t *testing.T) {
	sb := NewScrollback(64 * 1024)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sb.Append([]byte("concurrent data"))
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = sb.Snapshot()
		}()
	}
	wg.Wait()
}

func TestScrollbackDefaultBytes(t *testing.T) {
	if DefaultScrollbackBytes != 256*1024 {
		t.Errorf("DefaultScrollbackBytes should be 256*1024, got %d", DefaultScrollbackBytes)
	}
}
