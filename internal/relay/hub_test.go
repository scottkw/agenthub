package relay

import (
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/testutil"
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

	// CloseSlow runs in the hub goroutine; poll for it instead of a fixed sleep
	// that races the eviction under load (issue #80).
	testutil.WaitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return slowClosed
	}, "slow subscriber CloseSlow was not called")

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

// ---------------------------------------------------------------------------
// Host-authority PTY-size arbiter tests (VIEW-01, VIEW-02, D-01, D-02)
// Replaces the three MC-06 max-wins tests (TestHub_ResizeMaxWinsPolicy,
// TestHub_ResizeClientNoOpWhenDimensionsUnchanged,
// TestHub_ResizeClientUnsubscribeDoesNotShrink).
// ---------------------------------------------------------------------------

// TestHub_ResizeHostAuthority_MinAmongLocal verifies D-02: with two local-origin
// subscribers reporting different sizes the PTY grid tracks the smaller (min),
// not the larger (max). Each time the min changes resizeFn is called once.
func TestHub_ResizeHostAuthority_MinAmongLocal(t *testing.T) {
	r, w := io.Pipe()
	defer w.Close()

	var mu sync.Mutex
	var resizeCalls [][]int

	hub := NewHub("test-resize-min", r, w, DefaultScrollbackBytes, func(cols, rows int) error {
		mu.Lock()
		resizeCalls = append(resizeCalls, []int{cols, rows})
		mu.Unlock()
		return nil
	})

	sub1 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	sub2 := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	hub.Subscribe(sub1)
	hub.Subscribe(sub2)

	// sub1 claims 220x50 — only local subscriber so far; grid becomes 220x50.
	if err := hub.ResizeClient(sub1, 220, 50); err != nil {
		t.Fatalf("ResizeClient(sub1, 220, 50) error: %v", err)
	}

	// sub2 claims 80x24 — min across local subscribers is now 80x24; grid shrinks.
	if err := hub.ResizeClient(sub2, 80, 24); err != nil {
		t.Fatalf("ResizeClient(sub2, 80, 24) error: %v", err)
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

	// PTY grid must reflect min-among-local (80x24), not max (220x50).
	hub.mu.Lock()
	gotCols, gotRows := hub.ptyCols, hub.ptyRows
	hub.mu.Unlock()
	if gotCols != 80 || gotRows != 24 {
		t.Errorf("ptyCols/ptyRows = %d/%d, want 80/24 (min-among-local)", gotCols, gotRows)
	}
}

// TestHub_ResizeClientNoOpWhenDimensionsUnchanged verifies that resizeFn is not
// called when a local-origin subscriber reports the same dimensions twice.
func TestHub_ResizeClientNoOpWhenDimensionsUnchanged(t *testing.T) {
	r, w := io.Pipe()
	defer w.Close()

	var mu sync.Mutex
	var resizeCalls int

	hub := NewHub("test-resize-noop", r, w, DefaultScrollbackBytes, func(cols, rows int) error {
		mu.Lock()
		resizeCalls++
		mu.Unlock()
		return nil
	})

	// Stamp Origin:"local" so the subscriber passes the host-authority origin gate.
	sub := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	hub.Subscribe(sub)

	// First call — triggers resize.
	if err := hub.ResizeClient(sub, 80, 24); err != nil {
		t.Fatalf("first ResizeClient error: %v", err)
	}

	// Second call with identical dimensions — no-op.
	if err := hub.ResizeClient(sub, 80, 24); err != nil {
		t.Fatalf("second ResizeClient error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if resizeCalls != 1 {
		t.Errorf("expected 1 resizeFn call, got %d", resizeCalls)
	}
}

// TestHub_ResizeFreezeLastHostSize verifies D-01: when the last local-origin host
// disconnects the PTY grid is frozen at the last host size. A subsequent web-origin
// guest resize MUST NOT change the PTY grid or trigger resizeFn (VIEW-02).
func TestHub_ResizeFreezeLastHostSize(t *testing.T) {
	r, w := io.Pipe()
	defer w.Close()

	var mu sync.Mutex
	var resizeCalls [][]int

	hub := NewHub("test-resize-freeze", r, w, DefaultScrollbackBytes, func(cols, rows int) error {
		mu.Lock()
		resizeCalls = append(resizeCalls, []int{cols, rows})
		mu.Unlock()
		return nil
	})

	host := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	guest := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "web"}
	hub.Subscribe(host)
	hub.Subscribe(guest)

	// Host sets 200x50 — triggers resize.
	if err := hub.ResizeClient(host, 200, 50); err != nil {
		t.Fatalf("ResizeClient(host, 200, 50) error: %v", err)
	}

	// Host disconnects — no local subscribers remain.
	hub.Unsubscribe(host)

	// Guest reports 80x24 — origin gate must reject; resizeFn must NOT be called again.
	if err := hub.ResizeClient(guest, 80, 24); err != nil {
		t.Fatalf("ResizeClient(guest, 80, 24) error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Only the initial host resize should have triggered resizeFn.
	if len(resizeCalls) != 1 {
		t.Fatalf("expected 1 resizeFn call, got %d: %v", len(resizeCalls), resizeCalls)
	}
	if resizeCalls[0][0] != 200 || resizeCalls[0][1] != 50 {
		t.Errorf("resizeCalls[0] = %v, want [200, 50]", resizeCalls[0])
	}

	// PTY grid must be frozen at host's last size (200x50), not the guest's 80x24.
	hub.mu.Lock()
	gotCols, gotRows := hub.ptyCols, hub.ptyRows
	hub.mu.Unlock()
	if gotCols != 200 || gotRows != 50 {
		t.Errorf("ptyCols/ptyRows = %d/%d after host disconnect, want 200/50 (frozen)", gotCols, gotRows)
	}
}

// TestHub_ResizeIgnoresWebOrigin verifies VIEW-02 (T-157-01): a web-origin
// subscriber can never drive the PTY grid regardless of the dimensions it reports.
// resizeFn must not be called and ptyCols/ptyRows must remain at zero (no resize
// has occurred from an authoritative local-origin host).
func TestHub_ResizeIgnoresWebOrigin(t *testing.T) {
	r, w := io.Pipe()
	defer w.Close()

	var resizeCalled bool
	hub := NewHub("test-resize-web", r, w, DefaultScrollbackBytes, func(cols, rows int) error {
		resizeCalled = true
		return nil
	})

	guest := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "web"}
	hub.Subscribe(guest)

	if err := hub.ResizeClient(guest, 1920, 1080); err != nil {
		t.Fatalf("ResizeClient(guest) unexpected error: %v", err)
	}

	if resizeCalled {
		t.Error("resizeFn was called for a web-origin subscriber; want no-op (VIEW-02)")
	}

	hub.mu.Lock()
	gotCols, gotRows := hub.ptyCols, hub.ptyRows
	hub.mu.Unlock()
	if gotCols != 0 || gotRows != 0 {
		t.Errorf("ptyCols/ptyRows = %d/%d, want 0/0 — web guest must not mutate PTY grid", gotCols, gotRows)
	}
}

// TestHub_ResizeBroadcastsToSubscribers verifies VIEW-01: a local-host resize
// causes a MsgResize (0x02) frame with the correct cols/rows to be delivered to
// all subscribers (including a web guest) via broadcastResize.
func TestHub_ResizeBroadcastsToSubscribers(t *testing.T) {
	r, w := io.Pipe()
	defer w.Close()

	hub := NewHub("test-resize-bcast", r, w, DefaultScrollbackBytes, nil)

	host := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	guest := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "web"}
	hub.Subscribe(host)
	hub.Subscribe(guest)

	// Host triggers resize to 132x40.
	if err := hub.ResizeClient(host, 132, 40); err != nil {
		t.Fatalf("ResizeClient(host, 132, 40) error: %v", err)
	}

	// Drain the guest's channel — it should receive a 5-byte MsgResize frame.
	select {
	case frame := <-guest.Msgs:
		if len(frame) != 5 {
			t.Fatalf("expected 5-byte resize frame, got %d bytes: %v", len(frame), frame)
		}
		if frame[0] != MsgResize {
			t.Errorf("frame[0] = 0x%02x, want MsgResize (0x02)", frame[0])
		}
		cols := uint16(frame[1])<<8 | uint16(frame[2])
		rows := uint16(frame[3])<<8 | uint16(frame[4])
		if cols != 132 || rows != 40 {
			t.Errorf("decoded (cols, rows) = (%d, %d), want (132, 40)", cols, rows)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for MsgResize broadcast to guest")
	}
}

// TestHub_RowsFallback verifies that Rows() returns the 50 fallback before any
// resize, and returns the authoritative ptyRows after a local-host resize.
func TestHub_RowsFallback(t *testing.T) {
	hub, w := makeTestHub(t)
	defer w.Close()

	// Before any resize: fallback must be 50 (mirrors engine.go emuRows default).
	if got := hub.Rows(); got != 50 {
		t.Errorf("Rows() before resize = %d, want 50 (fallback)", got)
	}

	// Wire a local-origin host and trigger a resize.
	host := &Subscriber{Msgs: make(chan []byte, 256), CloseSlow: func() {}, Origin: "local"}
	hub.Subscribe(host)
	if err := hub.ResizeClient(host, 132, 40); err != nil {
		t.Fatalf("ResizeClient error: %v", err)
	}

	if got := hub.Rows(); got != 40 {
		t.Errorf("Rows() after resize = %d, want 40", got)
	}
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
