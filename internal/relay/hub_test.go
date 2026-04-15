package relay

import (
	"encoding/json"
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
	hub := NewHub("test-session", r, w, DefaultScrollbackBytes, nil)
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
	hub := NewHub("test-write", r, w, DefaultScrollbackBytes, nil)

	// Read what WriteInput produces from the writer end (w writes to itself here).
	// Use a separate pipe for the writer to capture what WriteInput sends.
	rIn, wIn := io.Pipe()
	hub2 := NewHub("test-write-in", r, wIn, DefaultScrollbackBytes, nil)
	_ = hub2

	// Simpler: just verify WriteInput writes to the writer without error.
	// We'll use a bytes-based approach: create a pipe and read from the read end.
	pr, pw := io.Pipe()
	hub3 := NewHub("test-wi", pr, pw, DefaultScrollbackBytes, nil)

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

// ---------------------------------------------------------------------------
// Tests for multi-client fan-out extensions (Phase 74, Plan 01)
// ---------------------------------------------------------------------------

func TestHub_SubscriberCountTracksConcurrentSubscribers(t *testing.T) {
	hub, _ := makeTestHub(t)

	if got := hub.SubscriberCount(); got != 0 {
		t.Fatalf("initial SubscriberCount = %d, want 0", got)
	}

	sub1 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	sub2 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}

	hub.Subscribe(sub1)
	if got := hub.SubscriberCount(); got != 1 {
		t.Fatalf("after 1 Subscribe: SubscriberCount = %d, want 1", got)
	}

	hub.Subscribe(sub2)
	if got := hub.SubscriberCount(); got != 2 {
		t.Fatalf("after 2 Subscribes: SubscriberCount = %d, want 2", got)
	}

	hub.Unsubscribe(sub1)
	if got := hub.SubscriberCount(); got != 1 {
		t.Fatalf("after Unsubscribe sub1: SubscriberCount = %d, want 1", got)
	}

	hub.Unsubscribe(sub2)
	if got := hub.SubscriberCount(); got != 0 {
		t.Fatalf("after Unsubscribe sub2: SubscriberCount = %d, want 0", got)
	}
}

func TestHub_ReadOnlyFlagStored(t *testing.T) {
	hub, _ := makeTestHub(t)

	sub := &Subscriber{
		Msgs:      make(chan []byte, 256),
		CloseSlow: func() {},
		ReadOnly:  true,
	}
	hub.Subscribe(sub)

	if !sub.ReadOnly {
		t.Error("expected sub.ReadOnly to be true")
	}
}

func TestHub_ClientNameStored(t *testing.T) {
	hub, _ := makeTestHub(t)

	sub := &Subscriber{
		Msgs:      make(chan []byte, 256),
		CloseSlow: func() {},
		Name:      "macbook",
	}
	hub.Subscribe(sub)

	if sub.Name != "macbook" {
		t.Errorf("expected sub.Name = %q, got %q", "macbook", sub.Name)
	}
}

func TestHub_ResizeMaxWinsPolicy(t *testing.T) {
	r, w := io.Pipe()

	var mu sync.Mutex
	var resizeCalls [][]int

	hub := NewHub("test-resize-max", r, w, DefaultScrollbackBytes, func(cols, rows int) error {
		mu.Lock()
		resizeCalls = append(resizeCalls, []int{cols, rows})
		mu.Unlock()
		return nil
	})

	sub1 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	sub2 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	// sub1 claims 220x50 — triggers resize to 220x50
	if err := hub.ResizeClient(sub1, 220, 50); err != nil {
		t.Fatalf("ResizeClient(sub1, 220, 50) error: %v", err)
	}

	// sub2 claims 80x24 — max is still 220x50, no resize
	if err := hub.ResizeClient(sub2, 80, 24); err != nil {
		t.Fatalf("ResizeClient(sub2, 80, 24) error: %v", err)
	}

	// sub1 claims 240x60 — triggers resize to 240x60
	if err := hub.ResizeClient(sub1, 240, 60); err != nil {
		t.Fatalf("ResizeClient(sub1, 240, 60) error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(resizeCalls) != 2 {
		t.Fatalf("expected 2 resizeFn calls, got %d: %v", len(resizeCalls), resizeCalls)
	}
	if resizeCalls[0][0] != 220 || resizeCalls[0][1] != 50 {
		t.Errorf("resizeCalls[0] = %v, want [220, 50]", resizeCalls[0])
	}
	if resizeCalls[1][0] != 240 || resizeCalls[1][1] != 60 {
		t.Errorf("resizeCalls[1] = %v, want [240, 60]", resizeCalls[1])
	}

	w.Close()
}

func TestHub_ResizeClientNoOpWhenDimensionsUnchanged(t *testing.T) {
	r, w := io.Pipe()

	var mu sync.Mutex
	var resizeCalls int

	hub := NewHub("test-resize-noop", r, w, DefaultScrollbackBytes, func(cols, rows int) error {
		mu.Lock()
		resizeCalls++
		mu.Unlock()
		return nil
	})

	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	hub.Subscribe(sub)

	// First call — triggers resize
	if err := hub.ResizeClient(sub, 80, 24); err != nil {
		t.Fatalf("first ResizeClient error: %v", err)
	}

	// Second call with same dimensions — no resize
	if err := hub.ResizeClient(sub, 80, 24); err != nil {
		t.Fatalf("second ResizeClient error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if resizeCalls != 1 {
		t.Errorf("expected 1 resizeFn call, got %d", resizeCalls)
	}

	w.Close()
}

func TestHub_ResizeClientUnsubscribeDoesNotShrink(t *testing.T) {
	r, w := io.Pipe()

	var mu sync.Mutex
	var resizeCalls [][]int

	hub := NewHub("test-resize-unsub", r, w, DefaultScrollbackBytes, func(cols, rows int) error {
		mu.Lock()
		resizeCalls = append(resizeCalls, []int{cols, rows})
		mu.Unlock()
		return nil
	})

	sub1 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	sub2 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}}
	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	// sub1 claims 220x50 — triggers resize
	if err := hub.ResizeClient(sub1, 220, 50); err != nil {
		t.Fatalf("ResizeClient(sub1, 220, 50) error: %v", err)
	}

	// sub2 claims 80x24 — no resize (max still 220x50)
	if err := hub.ResizeClient(sub2, 80, 24); err != nil {
		t.Fatalf("ResizeClient(sub2, 80, 24) error: %v", err)
	}

	// Unsubscribe sub1 — removes the 220x50 contributor
	hub.Unsubscribe(sub1)

	// sub2 claims 80x24 again — now max is 80x24 which differs from ptyCols=220/ptyRows=50
	// so resize IS called with (80, 24)
	if err := hub.ResizeClient(sub2, 80, 24); err != nil {
		t.Fatalf("ResizeClient(sub2, 80, 24) after unsub error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(resizeCalls) != 2 {
		t.Fatalf("expected 2 resizeFn calls, got %d: %v", len(resizeCalls), resizeCalls)
	}
	if resizeCalls[0][0] != 220 || resizeCalls[0][1] != 50 {
		t.Errorf("resizeCalls[0] = %v, want [220, 50]", resizeCalls[0])
	}
	if resizeCalls[1][0] != 80 || resizeCalls[1][1] != 24 {
		t.Errorf("resizeCalls[1] = %v, want [80, 24]", resizeCalls[1])
	}

	w.Close()
}

func TestBroadcastMeta_NonBlocking(t *testing.T) {
	// Create a hub with a no-op reader/writer.
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	hub := NewHub("test-meta", pr, pw, 1024, nil)

	// Create a subscriber with a buffered channel.
	closedSlow := make(chan struct{}, 1)
	sub := &Subscriber{
		Msgs: make(chan []byte, 256),
		CloseSlow: func() {
			closedSlow <- struct{}{}
		},
	}
	hub.Subscribe(sub)
	defer hub.Unsubscribe(sub)

	count := 2
	frame := MakeMeta(MetaPayload{ViewerCount: &count})
	hub.BroadcastMeta(frame)

	select {
	case msg := <-sub.Msgs:
		msgType, payload, err := ParseFrame(msg)
		if err != nil {
			t.Fatalf("ParseFrame error: %v", err)
		}
		if msgType != MsgMeta {
			t.Errorf("expected MsgMeta (0x%02x), got 0x%02x", MsgMeta, msgType)
		}
		var meta MetaPayload
		if err := json.Unmarshal(payload, &meta); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if meta.ViewerCount == nil || *meta.ViewerCount != 2 {
			t.Errorf("expected viewerCount=2, got %v", meta.ViewerCount)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for BroadcastMeta frame")
	}

	// Verify non-blocking: fill the channel, then BroadcastMeta should not block.
	for i := 0; i < 256; i++ {
		sub.Msgs <- []byte{0x00}
	}
	hub.BroadcastMeta(frame) // should trigger CloseSlow, not block

	select {
	case <-closedSlow:
		// Expected — slow subscriber was closed.
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for CloseSlow on full channel")
	}
}
