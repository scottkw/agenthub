package relay

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/testutil"
)

// setupTestServer creates a test HTTP server with a relay Server backed by a
// HubManager. The PTY is simulated by an io.Pipe pair:
//   - ptyWrite: callers write PTY output here (simulates PTY -> Hub direction)
//   - inputRead: callers read PTY input here (simulates client input arriving at PTY)
//
// Returns the httptest server, manager, ptyWrite, inputRead and the test session ID.
func setupTestServer(t *testing.T) (*httptest.Server, *HubManager, *io.PipeWriter, *io.PipeReader, string) {
	t.Helper()

	// ptyOutput pipe: hub reads from ptyOutputR, test writes to ptyOutputW.
	ptyOutputR, ptyOutputW := io.Pipe()

	// inputCapture pipe: hub writes to inputCaptureW, test reads from inputCaptureR.
	inputCaptureR, inputCaptureW := io.Pipe()

	const sessionID = "test-session"
	manager := NewHubManager()
	manager.Create(sessionID, ptyOutputR, inputCaptureW, nil)

	srv := httptest.NewServer(NewServer(manager, nil, nil))
	t.Cleanup(func() {
		srv.Close()
		manager.Shutdown()
		ptyOutputW.Close()
		inputCaptureW.Close()
	})

	return srv, manager, ptyOutputW, inputCaptureR, sessionID
}

// dialWS dials a WebSocket connection to the given test server for the given session.
func dialWS(t *testing.T, serverURL, sessionID string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/sessions/" + sessionID + "/ws"
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("dialWS: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// readFrame reads one binary message from the connection and returns (msgType, payload).
// Times out after 5 seconds.
func readFrame(t *testing.T, conn *websocket.Conn) (byte, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	msgType, payload, err := ParseFrame(msg)
	if err != nil {
		t.Fatalf("readFrame ParseFrame: %v", err)
	}
	return msgType, payload
}

// readDataFrame reads frames from conn, skipping any MsgMeta, MsgPresence,
// and MsgResize frames, and returns the first PTY-data frame. Used in tests
// that predate Phase 157 and only care about PTY output: meta, presence, and
// join-push resize frames are all server-push housekeeping, not PTY data.
// Tests that must assert the ordering of the join-push resize should use
// readFrame directly instead.
func readDataFrame(t *testing.T, conn *websocket.Conn) (byte, []byte) {
	t.Helper()
	for {
		msgType, payload := readFrame(t, conn)
		if msgType == MsgMeta || msgType == MsgPresence || msgType == MsgResize || msgType == MsgSelf {
			continue // skip server-push housekeeping frames (Phase 161: MsgSelf added)
		}
		return msgType, payload
	}
}

// dialWSWithQuery dials a WebSocket connection with optional query parameters appended.
func dialWSWithQuery(t *testing.T, serverURL, sessionID, query string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/sessions/" + sessionID + "/ws"
	if query != "" {
		wsURL += "?" + query
	}
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("dialWSWithQuery: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// TestServer_ReadOnlyClientInputDiscarded verifies MC-03: a read-only client's MsgInput
// frames are silently discarded — the PTY writer receives no data.
func TestServer_ReadOnlyClientInputDiscarded(t *testing.T) {
	srv, _, ptyWrite, inputRead, sessionID := setupTestServer(t)

	// Connect as read-only.
	roClient := dialWSWithQuery(t, srv.URL, sessionID, "readonly=1")

	// Also connect a normal client so we can verify output still works.
	_ = dialWS(t, srv.URL, sessionID)

	// Send an input frame from the read-only client.
	inputFrame := MakeInputFrame([]byte("should-be-discarded"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := roClient.Write(ctx, websocket.MessageBinary, inputFrame); err != nil {
		t.Fatalf("roClient send input: %v", err)
	}

	// Give the server a moment to process the frame.
	time.Sleep(100 * time.Millisecond)

	// Try to read from the PTY input pipe with a short deadline.
	// If read-only enforcement works, nothing should arrive.
	readBuf := make([]byte, 256)
	readDone := make(chan int, 1)
	readErr := make(chan error, 1)
	go func() {
		n, err := inputRead.Read(readBuf)
		if err != nil {
			readErr <- err
			return
		}
		readDone <- n
	}()

	select {
	case n := <-readDone:
		t.Fatalf("PTY received %d bytes %q from read-only client — should have been discarded", n, readBuf[:n])
	case err := <-readErr:
		t.Fatalf("inputRead error: %v", err)
	case <-time.After(300 * time.Millisecond):
		// Success — no data arrived at the PTY within the timeout.
	}

	// Verify the read-only client still receives output.
	const output = "visible to readonly"
	if _, err := ptyWrite.Write([]byte(output)); err != nil {
		t.Fatalf("ptyWrite: %v", err)
	}
	gotType, gotPayload := readDataFrame(t, roClient)
	if gotType != MsgOutput {
		t.Errorf("readonly client: got type 0x%02x, want MsgOutput", gotType)
	}
	if string(gotPayload) != output {
		t.Errorf("readonly client: got payload %q, want %q", gotPayload, output)
	}
}

// TestServer_ReadOnlyClientReceivesOutput verifies MC-03 positive path: a read-only
// client receives PTY output normally despite being unable to send input.
func TestServer_ReadOnlyClientReceivesOutput(t *testing.T) {
	srv, _, ptyWrite, _, sessionID := setupTestServer(t)

	roClient := dialWSWithQuery(t, srv.URL, sessionID, "readonly=true")

	const output = "hello readonly viewer"
	if _, err := ptyWrite.Write([]byte(output)); err != nil {
		t.Fatalf("ptyWrite: %v", err)
	}

	gotType, gotPayload := readDataFrame(t, roClient)
	if gotType != MsgOutput {
		t.Errorf("got type 0x%02x, want MsgOutput", gotType)
	}
	if string(gotPayload) != output {
		t.Errorf("got payload %q, want %q", gotPayload, output)
	}
}

// TestServer_ClientNameQueryParam verifies MC-05: the ?client= query param is parsed
// and the subscriber count reflects the connected client.
func TestServer_ClientNameQueryParam(t *testing.T) {
	srv, manager, _, _, sessionID := setupTestServer(t)

	_ = dialWSWithQuery(t, srv.URL, sessionID, "client=macbook")

	hub, ok := manager.Get(sessionID)
	if !ok {
		t.Fatal("session not found in manager")
	}
	// The connection handler subscribes asynchronously; poll until it registers
	// rather than sleeping a fixed interval that races the Subscribe (issue #80).
	testutil.WaitFor(t, 2*time.Second, func() bool {
		return hub.SubscriberCount() >= 1
	}, "SubscriberCount stayed < 1: connection handler never registered the subscriber")
}

// TestHub_TwoClientsFanOut verifies criterion 1: two WS clients connected to the
// same session both receive the same PTY output simultaneously.
func TestHub_TwoClientsFanOut(t *testing.T) {
	srv, _, ptyWrite, _, sessionID := setupTestServer(t)

	clientA := dialWS(t, srv.URL, sessionID)
	clientB := dialWS(t, srv.URL, sessionID)

	// Write PTY output.
	const payload = "hello world"
	if _, err := ptyWrite.Write([]byte(payload)); err != nil {
		t.Fatalf("ptyWrite: %v", err)
	}

	// Both clients must receive MsgOutput with "hello world".
	gotTypeA, gotPayloadA := readDataFrame(t, clientA)
	gotTypeB, gotPayloadB := readDataFrame(t, clientB)

	if gotTypeA != MsgOutput {
		t.Errorf("clientA: got type 0x%02x, want MsgOutput (0x%02x)", gotTypeA, MsgOutput)
	}
	if string(gotPayloadA) != payload {
		t.Errorf("clientA: got payload %q, want %q", gotPayloadA, payload)
	}
	if gotTypeB != MsgOutput {
		t.Errorf("clientB: got type 0x%02x, want MsgOutput (0x%02x)", gotTypeB, MsgOutput)
	}
	if string(gotPayloadB) != payload {
		t.Errorf("clientB: got payload %q, want %q", gotPayloadB, payload)
	}
}

// TestHub_ReconnectScrollback verifies criterion 2: a disconnected client reconnects
// and receives scrollback replay then resumes live output.
func TestHub_ReconnectScrollback(t *testing.T) {
	srv, _, ptyWrite, _, sessionID := setupTestServer(t)

	// Client A connects and confirms receipt of first output.
	clientA := dialWS(t, srv.URL, sessionID)

	const before = "before disconnect"
	if _, err := ptyWrite.Write([]byte(before)); err != nil {
		t.Fatalf("ptyWrite before: %v", err)
	}
	readDataFrame(t, clientA) // consume — confirms receipt; clientA has it.

	// Disconnect clientA.
	clientA.CloseNow()

	// Write output that goes to scrollback only (no live subscribers).
	const during = "during disconnect"
	if _, err := ptyWrite.Write([]byte(during)); err != nil {
		t.Fatalf("ptyWrite during: %v", err)
	}

	// Small delay to ensure hub drain goroutine processes the write before client B connects.
	time.Sleep(20 * time.Millisecond)

	// Client B connects — simulates reconnect.
	clientB := dialWS(t, srv.URL, sessionID)

	// The first message client B receives may be a MsgMeta frame (viewer count update).
	// Skip meta frames to get to the scrollback snapshot (MsgOutput).
	snapshotType, snapshotPayload := readDataFrame(t, clientB)
	if snapshotType != MsgOutput {
		t.Errorf("snapshot msg type: got 0x%02x, want MsgOutput", snapshotType)
	}

	// The snapshot is the raw scrollback bytes, which is the concatenation of
	// MakeOutputFrame("before disconnect") + MakeOutputFrame("during disconnect").
	// After ParseFrame on the first byte we get the first segment type + the rest.
	// Actually the snapshot itself is the whole scrollback buffer (already framed).
	// We expect both strings to appear somewhere in the full snapshot.
	fullSnapshot := string(snapshotPayload)
	if !strings.Contains(fullSnapshot, before) {
		t.Errorf("snapshot missing %q; got: %q", before, fullSnapshot)
	}
	if !strings.Contains(fullSnapshot, during) {
		t.Errorf("snapshot missing %q; got: %q", during, fullSnapshot)
	}

	// Write live output after reconnect.
	const after = "after reconnect"
	if _, err := ptyWrite.Write([]byte(after)); err != nil {
		t.Fatalf("ptyWrite after: %v", err)
	}

	liveType, livePayload := readDataFrame(t, clientB)
	if liveType != MsgOutput {
		t.Errorf("live msg type: got 0x%02x, want MsgOutput", liveType)
	}
	if string(livePayload) != after {
		t.Errorf("live payload: got %q, want %q", livePayload, after)
	}
}

// TestHub_InputFanOut verifies criterion 3: input from any connected client reaches
// the PTY and produces output visible to all clients.
func TestHub_InputFanOut(t *testing.T) {
	srv, _, ptyWrite, inputRead, sessionID := setupTestServer(t)

	clientA := dialWS(t, srv.URL, sessionID)
	clientB := dialWS(t, srv.URL, sessionID)

	// Client A sends an input frame.
	inputFrame := MakeInputFrame([]byte("ls\n"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := clientA.Write(ctx, websocket.MessageBinary, inputFrame); err != nil {
		t.Fatalf("clientA send input: %v", err)
	}

	// Read the input from the PTY side.
	gotInput := make([]byte, 3)
	if _, err := io.ReadFull(inputRead, gotInput); err != nil {
		t.Fatalf("inputRead: %v", err)
	}
	if string(gotInput) != "ls\n" {
		t.Errorf("PTY received %q, want %q", gotInput, "ls\n")
	}

	// Write output back to PTY (simulates PTY responding).
	const response = "file1 file2"
	if _, err := ptyWrite.Write([]byte(response)); err != nil {
		t.Fatalf("ptyWrite response: %v", err)
	}

	// Both clients receive the output.
	typeA, payloadA := readDataFrame(t, clientA)
	typeB, payloadB := readDataFrame(t, clientB)

	if typeA != MsgOutput || string(payloadA) != response {
		t.Errorf("clientA: got type=0x%02x payload=%q, want MsgOutput %q", typeA, payloadA, response)
	}
	if typeB != MsgOutput || string(payloadB) != response {
		t.Errorf("clientB: got type=0x%02x payload=%q, want MsgOutput %q", typeB, payloadB, response)
	}
}

// TestHub_SlowClientDisconnected verifies that a slow client (full send buffer) is
// disconnected while other clients continue receiving data uninterrupted.
//
// Design: slowClient never reads, so its 256-slot Msgs channel fills. normalClient
// reads actively in a concurrent goroutine so its channel stays drained.
// MsgMeta frames are sent on each subscribe event (SB-04), so slowClient receives
// 2 MsgMeta frames before the flood — we account for this by flooding with 300
// frames (> 256-2=254 remaining slots) to guarantee overflow.
func TestHub_SlowClientDisconnected(t *testing.T) {
	if testing.Short() {
		t.Skip("timing-sensitive under race detector and CI")
	}
	srv, _, ptyWrite, _, sessionID := setupTestServer(t)

	// Normal client that actively reads — drains concurrently with the flood so
	// its 256-slot Msgs channel never fills.
	normalClient := dialWS(t, srv.URL, sessionID)

	// Slow client — we dial it but never read from it, so its 256-entry channel fills up.
	slowClient := dialWS(t, srv.URL, sessionID)

	// Drain normalClient concurrently so its Msgs channel never overflows.
	normalDone := make(chan struct{})
	go func() {
		defer close(normalDone)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, _, err := normalClient.Read(ctx)
			cancel()
			if err != nil {
				return
			}
		}
	}()

	// Flood the PTY pipe with 300 distinct messages to overflow the slow client's buffer.
	// We write each message individually to maximize frame count.
	for i := range 300 {
		msg := []byte{byte(i & 0xFF), byte(i >> 8)}
		if _, err := ptyWrite.Write(msg); err != nil {
			// Pipe might be closed already; that's fine after enough writes.
			break
		}
	}

	// Slow client should have been disconnected (CloseSlow closes the WS connection).
	// Drain any buffered frames until we get the close error.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, _, err := slowClient.Read(ctx)
		cancel()
		if err != nil {
			// Expected — connection was closed by CloseSlow.
			return
		}
	}
	t.Error("slowClient: expected connection to be closed within 5s, but it remained open")
}

// TestRelayJoin_PushesResizeBeforeScrollback asserts VIEW-03 on the relay path:
// a guest's first non-meta frame is a 0x02 MsgResize with the hub's authoritative
// grid (cols=120, rows=30) and it arrives BEFORE the scrollback snapshot.
func TestRelayJoin_PushesResizeBeforeScrollback(t *testing.T) {
	srv, manager, ptyWrite, _, sessionID := setupTestServer(t)

	hub, ok := manager.Get(sessionID)
	if !ok {
		t.Fatal("hub not found in manager")
	}

	// Create a synthetic local subscriber so ResizeClient accepts the call and
	// stamps the hub's ptyCols/ptyRows to 120×30.
	localSub := &Subscriber{
		Origin: "local",
		Msgs:   make(chan []byte, 64),
	}
	localSub.CloseSlow = func() {}
	hub.Subscribe(localSub)
	if err := hub.ResizeClient(localSub, 120, 30); err != nil {
		t.Fatalf("ResizeClient: %v", err)
	}

	// Write PTY data so scrollback is non-empty — the ordering assertion only
	// makes sense if there is scrollback for the resize to precede.
	if _, err := ptyWrite.Write([]byte("terminal-output")); err != nil {
		t.Fatalf("ptyWrite: %v", err)
	}
	// Let the hub drain the PTY pipe into the scrollback buffer before the guest joins.
	testutil.WaitFor(t, 2*time.Second, func() bool {
		snap := hub.ScrollbackSnapshot()
		return len(snap) > 0
	}, "scrollback never populated")

	// Connect the guest.
	conn := dialWS(t, srv.URL, sessionID)

	// Read non-housekeeping frames in order, skipping MsgMeta/MsgPresence/MsgSelf
	// but NOT MsgResize so we can verify the ordering (resize arrives before scrollback).
	readOrdered := func() (byte, []byte) {
		for {
			typ, payload := readFrame(t, conn)
			if typ == MsgMeta || typ == MsgPresence || typ == MsgSelf {
				continue // skip server-push housekeeping frames (Phase 161: MsgSelf added)
			}
			return typ, payload
		}
	}

	// The first non-meta/non-presence frame must be the VIEW-03 resize push.
	firstType, firstPayload := readOrdered()
	if firstType != MsgResize {
		t.Fatalf("first non-meta frame: got type 0x%02x, want MsgResize (0x%02x)", firstType, MsgResize)
	}
	if len(firstPayload) != 4 {
		t.Fatalf("MsgResize payload length: got %d, want 4", len(firstPayload))
	}
	gotCols := int(uint16(firstPayload[0])<<8 | uint16(firstPayload[1]))
	gotRows := int(uint16(firstPayload[2])<<8 | uint16(firstPayload[3]))
	if gotCols != hub.Cols() || gotRows != hub.Rows() {
		t.Errorf("resize frame dims %dx%d don't match hub.Cols()/hub.Rows() %dx%d",
			gotCols, gotRows, hub.Cols(), hub.Rows())
	}
	if gotCols != 120 || gotRows != 30 {
		t.Errorf("resize frame: got %dx%d, want 120x30", gotCols, gotRows)
	}

	// The next non-meta frame must be the scrollback snapshot (MsgOutput),
	// proving the resize arrived BEFORE the replayed bytes.
	secondType, _ := readOrdered()
	if secondType != MsgOutput {
		t.Errorf("second non-meta frame: got type 0x%02x, want MsgOutput (0x%02x)", secondType, MsgOutput)
	}
}
