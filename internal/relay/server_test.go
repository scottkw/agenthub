package relay

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
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

	srv := httptest.NewServer(NewServer(manager, nil))
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
	gotTypeA, gotPayloadA := readFrame(t, clientA)
	gotTypeB, gotPayloadB := readFrame(t, clientB)

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
	readFrame(t, clientA) // consume — confirms receipt; clientA has it.

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

	// The first message client B receives is the scrollback snapshot.
	// The snapshot contains ALL prior framed output concatenated.
	// We need to read all snapshot bytes in one or more reads, but in our
	// server implementation the snapshot is sent as a single binary message.
	snapshotType, snapshotPayload := readFrame(t, clientB)
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

	liveType, livePayload := readFrame(t, clientB)
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
	typeA, payloadA := readFrame(t, clientA)
	typeB, payloadB := readFrame(t, clientB)

	if typeA != MsgOutput || string(payloadA) != response {
		t.Errorf("clientA: got type=0x%02x payload=%q, want MsgOutput %q", typeA, payloadA, response)
	}
	if typeB != MsgOutput || string(payloadB) != response {
		t.Errorf("clientB: got type=0x%02x payload=%q, want MsgOutput %q", typeB, payloadB, response)
	}
}

// TestHub_SlowClientDisconnected verifies that a slow client (full send buffer) is
// disconnected while other clients continue receiving data uninterrupted.
func TestHub_SlowClientDisconnected(t *testing.T) {
	if testing.Short() {
		t.Skip("timing-sensitive under race detector and CI")
	}
	srv, _, ptyWrite, _, sessionID := setupTestServer(t)

	// Normal client that actively reads.
	normalClient := dialWS(t, srv.URL, sessionID)

	// Slow client — we dial it but never read from it, so its 256-entry channel fills up.
	slowClient := dialWS(t, srv.URL, sessionID)

	// Flood the PTY pipe with 300 distinct messages to overflow the slow client's buffer.
	// We write each message individually to maximize frame count.
	for i := range 300 {
		msg := []byte{byte(i & 0xFF), byte(i >> 8)}
		if _, err := ptyWrite.Write(msg); err != nil {
			// Pipe might be closed already; that's fine after enough writes.
			break
		}
	}

	// Read at least 256 frames on the normal client to confirm it's still alive.
	// We use a liberal timeout per-read to tolerate goroutine scheduling.
	received := 0
	for received < 256 {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, _, err := normalClient.Read(ctx)
		cancel()
		if err != nil {
			t.Fatalf("normalClient stopped receiving at frame %d: %v", received, err)
		}
		received++
	}

	// Slow client should have been disconnected (CloseSlow closes the WS connection).
	// Drain any buffered frames until we get the close error — the server sent a
	// StatusPolicyViolation close frame which will eventually appear as a read error.
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
