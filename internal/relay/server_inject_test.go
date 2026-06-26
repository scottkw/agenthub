package relay

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// writerFunc is a func([]byte)(int,error) adapter that implements io.Writer.
// Used to build a counting PTY writer for inject tests without an io.Discard sink.
type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// setupInjectTestServer creates a test relay.Server with a counting PTY writer
// and returns: the httptest.Server, HubManager, the session ID, and a pointer to
// the per-write counter. The counting writer increments on every Write call so
// tests can assert PTY writes were (or were not) performed.
//
// A minimal in-memory chatAppendFn is wired on the hub so that BroadcastChat fires
// (required for TestInject_RWCap_WritesToPTY to observe the MsgChat broadcast).
// The function fills in missing ChatMessage fields (ID, TimestampMs, SchemaVersion,
// SessionID) so the frame round-trips as a well-formed ChatMessage.
func setupInjectTestServer(t *testing.T) (*httptest.Server, *HubManager, string, *atomic.Int32) {
	t.Helper()

	var ptyWriteCount atomic.Int32
	ptyOutputR, ptyOutputW := io.Pipe()

	countingWriter := writerFunc(func(p []byte) (int, error) {
		ptyWriteCount.Add(1)
		return len(p), nil
	})

	const sessionID = "inject-session"
	manager := NewHubManager()
	hub := manager.Create(sessionID, ptyOutputR, countingWriter, nil)

	// Wire a simple in-memory chatAppendFn — stands in for daemon.ChatStore.AppendMessage
	// without importing daemon. Fills the fields ChatStore would normally fill.
	hub.SetChatAppendFn(func(msg ChatMessage) (ChatMessage, error) {
		if msg.ID == "" {
			msg.ID = "test-id"
		}
		if msg.SessionID == "" {
			msg.SessionID = sessionID
		}
		if msg.SchemaVersion == 0 {
			msg.SchemaVersion = ChatSchemaVersion
		}
		if msg.TimestampMs == 0 {
			msg.TimestampMs = 1
		}
		return msg, nil
	})

	srv := NewServer(manager, nil, nil)
	ts := httptest.NewServer(srv)
	t.Cleanup(func() {
		ts.Close()
		manager.Shutdown()
		ptyOutputW.Close()
	})
	return ts, manager, sessionID, &ptyWriteCount
}

// dialInjectWS dials a WebSocket to the inject test server with optional query string.
func dialInjectWS(t *testing.T, ts *httptest.Server, sessionID, query string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/sessions/" + sessionID + "/ws"
	if query != "" {
		wsURL += "?" + query
	}
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("dialInjectWS: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// waitForFrameType reads frames from conn (skipping frames of other types) and
// returns the first frame whose type byte matches msgType. Fails on timeout.
// Timeout: 3 seconds (sub-second in practice).
func waitForFrameType(t *testing.T, conn *websocket.Conn, msgType byte, label string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("waitForFrameType(%s): %v", label, err)
		}
		if len(rawMsg) == 0 {
			continue
		}
		if rawMsg[0] == msgType {
			return rawMsg
		}
		// Skip frames of other types (e.g. MsgMeta, MsgPresence).
	}
}

// TestInject_RWCap_WritesToPTY verifies MENTION-02: a RW client sending a
// MsgSessionInject frame causes the sanitized text to be written to PTY stdin
// (ptyWriteCount > 0) and a MsgChat broadcast with SessionInject:true is
// received by the sender (all subscribers observe the broadcast).
func TestInject_RWCap_WritesToPTY(t *testing.T) {
	ts, _, sessionID, ptyWriteCount := setupInjectTestServer(t)

	// Dial as a RW client (no readonly param).
	conn := dialInjectWS(t, ts, sessionID, "")

	// Drain the initial MsgMeta and MsgPresence frames before sending inject.
	// We drain by discarding non-MsgChat frames until the goroutine is stable.
	// A short sleep is sufficient here — the hub sends presence synchronously on Subscribe.
	time.Sleep(50 * time.Millisecond)

	const injectText = "run tests"
	injectPayload, _ := json.Marshal(InjectPayload{Text: injectText})
	frame := append([]byte{MsgSessionInject}, injectPayload...)

	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
		t.Fatalf("write inject frame: %v", err)
	}

	// Wait for the MsgChat broadcast — all subscribers receive it (including sender).
	rawChat := waitForFrameType(t, conn, MsgChat, "RW inject chat broadcast")

	// Assert PTY received a write.
	if ptyWriteCount.Load() == 0 {
		t.Error("expected PTY write count > 0 after RW inject")
	}

	// Decode the MsgChat payload and verify SessionInject:true.
	_, chatPayload, err := ParseFrame(rawChat)
	if err != nil {
		t.Fatalf("ParseFrame MsgChat: %v", err)
	}
	var msg ChatMessage
	if err := json.Unmarshal(chatPayload, &msg); err != nil {
		t.Fatalf("unmarshal ChatMessage: %v", err)
	}
	if !msg.SessionInject {
		t.Errorf("ChatMessage.SessionInject = false, want true")
	}
	// The persisted content must be the original text (Pitfall 6 / A1/A3).
	if msg.Content != injectText {
		t.Errorf("ChatMessage.Content = %q, want %q (original pre-sanitize text)", msg.Content, injectText)
	}
}

// TestInject_OnlyDedicatedFrame verifies MENTION-03: sending a MsgChatSend frame
// (0x31 — normal chat send) does NOT trigger any PTY write. Only the dedicated
// MsgSessionInject verb (0x35) is capable of writing to PTY stdin (D-02).
func TestInject_OnlyDedicatedFrame(t *testing.T) {
	ts, _, sessionID, ptyWriteCount := setupInjectTestServer(t)

	conn := dialInjectWS(t, ts, sessionID, "")
	time.Sleep(50 * time.Millisecond)

	// Send a MsgChatSend frame (chat-only, NEVER writes to PTY per D-02).
	chatSendPayload, _ := json.Marshal(map[string]string{"content": "hello PTY"})
	chatSendFrame := append([]byte{MsgChatSend}, chatSendPayload...)
	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageBinary, chatSendFrame); err != nil {
		t.Fatalf("write MsgChatSend: %v", err)
	}

	// Also send a stray/unknown frame type to confirm only the inject verb writes.
	strayFrame := []byte{0x50, 0x01, 0x02, 0x03}
	if err := conn.Write(ctx, websocket.MessageBinary, strayFrame); err != nil {
		t.Fatalf("write stray frame: %v", err)
	}

	// Give the server time to process both frames.
	time.Sleep(100 * time.Millisecond)

	// PTY write count must be exactly zero.
	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after MsgChatSend/stray frame, want 0 (only MsgSessionInject writes to PTY)", count)
	}
}

// TestInject_ROCap_RelayPath verifies SEC-01 (relay path): a read-only client
// sending a hand-crafted MsgSessionInject frame receives a MsgInjectError NAK
// and causes zero PTY writes. The gate holds against a raw frame regardless of
// any client-side suppression.
func TestInject_ROCap_RelayPath(t *testing.T) {
	ts, _, sessionID, ptyWriteCount := setupInjectTestServer(t)

	// Dial as a RO client — sub.ReadOnly will be true.
	roConn := dialInjectWS(t, ts, sessionID, "readonly=1")
	time.Sleep(50 * time.Millisecond)

	// Send a hand-crafted MsgSessionInject frame directly, bypassing any client-side check.
	injectPayload, _ := json.Marshal(InjectPayload{Text: "evil command"})
	frame := append([]byte{MsgSessionInject}, injectPayload...)
	ctx := context.Background()
	if err := roConn.Write(ctx, websocket.MessageBinary, frame); err != nil {
		t.Fatalf("roConn write inject: %v", err)
	}

	// Must receive a MsgInjectError NAK frame within timeout.
	nakFrame := waitForFrameType(t, roConn, MsgInjectError, "RO inject NAK")
	if len(nakFrame) < 2 {
		t.Fatalf("MsgInjectError frame too short: %v", nakFrame)
	}
	if nakFrame[0] != MsgInjectError {
		t.Errorf("first byte = 0x%02x, want 0x%02x (MsgInjectError)", nakFrame[0], MsgInjectError)
	}

	// PTY stdin must have received zero bytes — the gate held.
	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after RO inject attempt, want 0 (SEC-01 gate failed)", count)
	}
}
