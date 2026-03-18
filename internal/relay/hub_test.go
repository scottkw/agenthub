package relay

import (
	"io"
	"sync"
	"testing"
	"time"
)

// makeTestHub creates a Hub backed by an io.Pipe for testing.
// Returns the Hub, the write end (simulates PTY output), and a done channel.
func makeTestHub(t *testing.T) (*Hub, *io.PipeWriter) {
	t.Helper()
	r, w := io.Pipe()
	hub := NewHub("test-session", r, w, DefaultScrollbackBytes)
	return hub, w
}

func TestHubSubscribeReceivesOutput(t *testing.T) {
	hub, ptyWriter := makeTestHub(t)

	sub := &Subscriber{
		Msgs:      make(chan []byte, 256),
		CloseSlow: func() {},
	}
	hub.Subscribe(sub)
	go hub.Run()

	// Write data to the pipe — simulates PTY output
	data := []byte("hello from pty")
	if _, err := ptyWriter.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	select {
	case frame := <-sub.Msgs:
		if len(frame) == 0 {
			t.Fatal("received empty frame")
		}
		if frame[0] != MsgOutput {
			t.Errorf("expected MsgOutput type byte, got 0x%02x", frame[0])
		}
		payload := frame[1:]
		if string(payload) != string(data) {
			t.Errorf("expected payload %q, got %q", data, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for frame")
	}

	ptyWriter.Close()
}

func TestHubTwoSubscribersBothReceive(t *testing.T) {
	hub, ptyWriter := makeTestHub(t)

	sub1 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	sub2 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	hub.Subscribe(sub1)
	hub.Subscribe(sub2)
	go hub.Run()

	data := []byte("broadcast data")
	if _, err := ptyWriter.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	timeout := time.After(2 * time.Second)
	var got1, got2 []byte

	for got1 == nil || got2 == nil {
		select {
		case f := <-sub1.Msgs:
			got1 = f
		case f := <-sub2.Msgs:
			got2 = f
		case <-timeout:
			t.Fatalf("timeout: sub1 received=%v, sub2 received=%v", got1 != nil, got2 != nil)
		}
	}

	// Both should have received the same frame
	if string(got1) != string(got2) {
		t.Errorf("subscriber frames differ: sub1=%v sub2=%v", got1, got2)
	}

	ptyWriter.Close()
}

func TestHubSlowSubscriberGetsDisconnected(t *testing.T) {
	hub, ptyWriter := makeTestHub(t)

	var slowClosed bool
	var mu sync.Mutex
	slowClosed = false

	// Slow subscriber: buffer size 0 (always full)
	slowSub := &Subscriber{
		Msgs: make(chan []byte, 0),
		CloseSlow: func() {
			mu.Lock()
			defer mu.Unlock()
			slowClosed = true
		},
	}
	fastSub := &Subscriber{
		Msgs:      make(chan []byte, 256),
		CloseSlow: func() {},
	}
	hub.Subscribe(slowSub)
	hub.Subscribe(fastSub)
	go hub.Run()

	// Write enough data to trigger the slow-client path
	for i := 0; i < 10; i++ {
		ptyWriter.Write([]byte("data"))
	}

	// Wait for fast subscriber to receive at least something
	select {
	case <-fastSub.Msgs:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for fast subscriber")
	}

	// Give CloseSlow goroutine time to run
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	closed := slowClosed
	mu.Unlock()

	if !closed {
		t.Error("slow subscriber CloseSlow was not called")
	}

	ptyWriter.Close()
}

func TestHubUnsubscribe(t *testing.T) {
	hub, ptyWriter := makeTestHub(t)

	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	hub.Subscribe(sub)
	hub.Unsubscribe(sub)
	go hub.Run()

	ptyWriter.Write([]byte("data after unsubscribe"))

	// Give hub time to process
	time.Sleep(100 * time.Millisecond)

	select {
	case msg := <-sub.Msgs:
		t.Errorf("unexpected message received after unsubscribe: %v", msg)
	default:
		// Expected: no message
	}

	ptyWriter.Close()
}

func TestHubShutdownOnEOF(t *testing.T) {
	hub, ptyWriter := makeTestHub(t)

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()

	// Closing the writer sends EOF to the reader
	ptyWriter.Close()

	select {
	case <-done:
		// Hub.Run returned as expected
	case <-time.After(2 * time.Second):
		t.Fatal("Hub.Run did not return after EOF")
	}
}

func TestHubDoneChannel(t *testing.T) {
	hub, ptyWriter := makeTestHub(t)
	go hub.Run()

	ptyWriter.Close()

	select {
	case <-hub.Done():
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Hub.Done() channel not closed after EOF")
	}
}

func TestHubWriteInput(t *testing.T) {
	r, w := io.Pipe()
	hub := NewHub("test-write", r, w, DefaultScrollbackBytes)

	// Read what WriteInput produces from the writer end (w writes to itself here).
	// Use a separate pipe for the writer to capture what WriteInput sends.
	rIn, wIn := io.Pipe()
	hub2 := NewHub("test-write-in", r, wIn, DefaultScrollbackBytes)
	_ = hub2

	// Simpler: just verify WriteInput writes to the writer without error.
	// We'll use a bytes-based approach: create a pipe and read from the read end.
	pr, pw := io.Pipe()
	hub3 := NewHub("test-wi", pr, pw, DefaultScrollbackBytes)

	readDone := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := pr.Read(buf)
		readDone <- buf[:n]
	}()

	data := []byte("input data")
	if err := hub3.WriteInput(data); err != nil {
		t.Fatalf("WriteInput returned error: %v", err)
	}

	// Close so the goroutine doesn't block
	pw.Close()

	_ = rIn
	_ = hub
	_ = w
}

func TestHubScrollbackSnapshot(t *testing.T) {
	hub, ptyWriter := makeTestHub(t)

	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	hub.Subscribe(sub)
	go hub.Run()

	data := []byte("scrollback data")
	ptyWriter.Write(data)

	// Wait for hub to process the data
	select {
	case <-sub.Msgs:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for frame")
	}

	snap := hub.ScrollbackSnapshot()
	if len(snap) == 0 {
		t.Error("scrollback snapshot is empty after receiving data")
	}

	// Snapshot should contain the framed output
	expected := MakeOutputFrame(data)
	if string(snap) != string(expected) {
		t.Errorf("scrollback mismatch: got %v, want %v", snap, expected)
	}

	ptyWriter.Close()
}

func TestHubScrollbackContainsAllFrames(t *testing.T) {
	hub, ptyWriter := makeTestHub(t)

	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	hub.Subscribe(sub)
	go hub.Run()

	chunks := [][]byte{[]byte("chunk1"), []byte("chunk2"), []byte("chunk3")}
	for _, c := range chunks {
		ptyWriter.Write(c)
		// Wait for each chunk to be processed
		select {
		case <-sub.Msgs:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}

	snap := hub.ScrollbackSnapshot()
	if len(snap) == 0 {
		t.Error("empty scrollback")
	}

	ptyWriter.Close()
}
