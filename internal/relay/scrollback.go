package relay

import "sync"

// DefaultScrollbackBytes is the default maximum scrollback buffer size (256 KiB).
const DefaultScrollbackBytes = 256 * 1024

// Scrollback is a bounded, mutex-protected byte buffer.
// When the buffer exceeds its maximum capacity, the oldest bytes are discarded.
type Scrollback struct {
	mu  sync.Mutex
	buf []byte
	max int
}

// NewScrollback creates a new Scrollback with the given maximum byte capacity.
func NewScrollback(maxBytes int) *Scrollback {
	return &Scrollback{
		buf: make([]byte, 0, maxBytes),
		max: maxBytes,
	}
}

// Append adds data to the scrollback buffer.
// If the resulting length exceeds max, the oldest bytes are dropped from the front.
func (s *Scrollback) Append(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf = append(s.buf, data...)

	if len(s.buf) > s.max {
		// Discard oldest bytes so the buffer fits within max.
		excess := len(s.buf) - s.max
		// Shift contents left — avoids a separate allocation.
		copy(s.buf, s.buf[excess:])
		s.buf = s.buf[:s.max]
	}
}

// Snapshot returns an independent copy of the current buffer contents.
// Mutating the returned slice does not affect the internal buffer.
func (s *Scrollback) Snapshot() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]byte, len(s.buf))
	copy(out, s.buf)
	return out
}
