package relay

import (
	"io"
	"sync"
)

// Subscriber represents a single WebSocket connection subscribed to a session's output.
type Subscriber struct {
	// Msgs is the outbound channel for framed messages. Buffered to 256 frames.
	Msgs chan []byte
	// CloseSlow is called in its own goroutine when the Msgs channel is full.
	// Implementations should close the WebSocket and call Unsubscribe.
	CloseSlow func()

	// ReadOnly: if true, input frames from this client are discarded by the server read pump. (MC-03)
	ReadOnly bool

	// Name: optional client identity from ?client= query param. (MC-05)
	Name string

	// Cols, Rows: last reported terminal dimensions from this client.
	// Read/written under hub.mu. (MC-06)
	Cols int
	Rows int
}

// Hub manages a single PTY session's output fan-out.
// One goroutine (Run) drains the reader; N subscribers receive every frame.
type Hub struct {
	sessionID  string
	reader     io.Reader
	writer     io.Writer
	scrollback *Scrollback
	resizeFn   func(cols, rows int) error

	mu          sync.Mutex
	subscribers map[*Subscriber]struct{}
	done        chan struct{}
	closed      bool
	closeOnce   sync.Once

	// ptyCols, ptyRows: current PTY dimensions as set by the max-wins arbiter. (MC-06)
	ptyCols int
	ptyRows int
}

// NewHub constructs a Hub for the given session.
// scrollbackBytes controls the scrollback buffer capacity.
// resizeFn is called when a resize event is received; may be nil.
func NewHub(sessionID string, reader io.Reader, writer io.Writer, scrollbackBytes int, resizeFn func(cols, rows int) error) *Hub {
	return &Hub{
		sessionID:   sessionID,
		reader:      reader,
		writer:      writer,
		scrollback:  NewScrollback(scrollbackBytes),
		resizeFn:    resizeFn,
		subscribers: make(map[*Subscriber]struct{}),
		done:        make(chan struct{}),
	}
}

// Resize calls the resize callback registered at construction time.
// If no callback was provided it is a no-op.
func (h *Hub) Resize(cols, rows int) error {
	if h.resizeFn != nil {
		return h.resizeFn(cols, rows)
	}
	return nil
}

// Subscribe adds a subscriber to receive future frames.
// Must be called before Hub.Run or while Run is active under the mu lock.
func (h *Hub) Subscribe(sub *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribers[sub] = struct{}{}
}

// Unsubscribe removes a subscriber.
func (h *Hub) Unsubscribe(sub *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers, sub)
}

// SubscriberCount returns the number of currently subscribed clients. (MC-04)
func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

// ResizeClient stores the subscriber's reported dimensions and calls resizeFn
// only when the new maximum across all subscribers differs from the current PTY size.
// Implements max-wins policy: PTY dimensions track the max of all active subscribers. (MC-06)
//
// resizeFn is called AFTER releasing hub.mu to avoid contending the broadcast
// drain loop with a potentially blocking PTY resize syscall.
func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
	h.mu.Lock()
	sub.Cols = cols
	sub.Rows = rows

	maxCols, maxRows := 0, 0
	for s := range h.subscribers {
		if s.Cols > maxCols {
			maxCols = s.Cols
		}
		if s.Rows > maxRows {
			maxRows = s.Rows
		}
	}

	needResize := (maxCols > 0 || maxRows > 0) && (maxCols != h.ptyCols || maxRows != h.ptyRows)
	if needResize {
		h.ptyCols = maxCols
		h.ptyRows = maxRows
	}
	h.mu.Unlock() // release BEFORE calling resizeFn

	if needResize && h.resizeFn != nil {
		return h.resizeFn(maxCols, maxRows)
	}
	return nil
}

// Run is the drain goroutine. It reads from the PTY reader in 32 KiB chunks,
// wraps each chunk in a MakeOutputFrame, appends to scrollback, and broadcasts
// to all subscribers. Slow subscribers (full channel) have CloseSlow called in a
// separate goroutine so they cannot block the drain loop.
//
// Run returns when the reader returns any error (typically io.EOF on PTY close).
// On return, Run calls Shutdown() to signal completion.
func (h *Hub) Run() {
	defer h.Shutdown()

	buf := make([]byte, 32*1024)
	for {
		n, err := h.reader.Read(buf)
		if n > 0 {
			// Copy the live slice before broadcasting — each frame allocation
			// is independent so slow subscribers cannot corrupt fast ones.
			frame := MakeOutputFrame(buf[:n])
			h.scrollback.Append(frame)
			h.broadcast(frame)
		}
		if err != nil {
			return
		}
	}
}

// broadcast sends frame to all current subscribers using a non-blocking send.
// Subscribers with a full channel have their CloseSlow invoked in a new goroutine.
func (h *Hub) broadcast(frame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for sub := range h.subscribers {
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()
		}
	}
}

// BroadcastMeta sends a metadata frame to all subscribers using a non-blocking
// send. Slow subscribers have CloseSlow called — MsgMeta must never block the
// PTY drain loop. Used by relay.Server and webserver to push viewer count updates.
func (h *Hub) BroadcastMeta(frame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for sub := range h.subscribers {
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()
		}
	}
}

// Shutdown signals that the hub has stopped. Safe to call multiple times.
func (h *Hub) Shutdown() {
	h.closeOnce.Do(func() {
		h.mu.Lock()
		h.closed = true
		h.mu.Unlock()
		close(h.done)
	})
}

// WriteInput writes raw input bytes to the underlying PTY writer (stdin).
func (h *Hub) WriteInput(data []byte) error {
	_, err := h.writer.Write(data)
	return err
}

// ScrollbackSnapshot returns a copy of the current scrollback buffer contents.
// Callers should Subscribe before calling ScrollbackSnapshot to avoid missing
// frames written between the snapshot and the first message in Msgs.
func (h *Hub) ScrollbackSnapshot() []byte {
	return h.scrollback.Snapshot()
}

// Done returns a channel that is closed when the hub shuts down.
func (h *Hub) Done() <-chan struct{} {
	return h.done
}
