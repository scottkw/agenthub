package relay

// Phase 154 Plan 01 — relay read-pump MsgChatSend dispatch tests.
//
// Tests drive a real relay.Server WebSocket subscriber through the read pump
// and assert:
//   1. A MsgChatSend frame from a RW subscriber causes a MsgChat broadcast to
//      all subscribers.
//   2. A MsgChatSend frame from a RO subscriber is silently dropped (no broadcast,
//      no NAK frame) per RESEARCH Open Question 1 recommendation.
//   3. A MsgChatSend frame with malformed / empty-content JSON is silently ignored.
//
// Harness reuses setupInjectTestServer (server_inject_test.go) which wires a
// Hub with a counting PTY writer and an in-memory chatAppendFn.
//
// TESTING.md registration is deferred to plan 154-06.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestChatSend_RWBroadcasts_RelayPath verifies that a RW client sending a
// MsgChatSend frame (0x31) over the relay WebSocket receives a MsgChat (0x30)
// broadcast — the end-to-end dispatch through the relay read pump.
func TestChatSend_RWBroadcasts_RelayPath(t *testing.T) {
	ts, _, sessionID, ptyWriteCount := setupInjectTestServer(t)

	// Dial as a RW client (no readonly param).
	conn := dialInjectWS(t, ts, sessionID, "")

	// Allow the hub to send initial MsgMeta + MsgPresence frames.
	time.Sleep(50 * time.Millisecond)

	// Build and send a MsgChatSend frame with valid JSON payload.
	payload, _ := json.Marshal(ChatSendPayload{Content: "relay chat send test"})
	frame := append([]byte{MsgChatSend}, payload...)

	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
		t.Fatalf("write MsgChatSend: %v", err)
	}

	// A MsgChat broadcast must be received within timeout.
	chatFrame := waitForFrameType(t, conn, MsgChat, "RW chat-send broadcast")

	if chatFrame[0] != MsgChat {
		t.Errorf("broadcast frame type = 0x%02x, want 0x%02x (MsgChat)", chatFrame[0], MsgChat)
	}

	// Decode and verify: SessionInject must be false (chat-send, not inject).
	_, chatPayload, err := ParseFrame(chatFrame)
	if err != nil {
		t.Fatalf("ParseFrame MsgChat: %v", err)
	}
	var msg ChatMessage
	if err := json.Unmarshal(chatPayload, &msg); err != nil {
		t.Fatalf("unmarshal ChatMessage: %v", err)
	}
	if msg.SessionInject {
		t.Error("ChatMessage.SessionInject = true, want false (chat-send must not mark as inject)")
	}
	if msg.Content != "relay chat send test" {
		t.Errorf("ChatMessage.Content = %q, want %q", msg.Content, "relay chat send test")
	}

	// PTY write count must be exactly zero — chat send NEVER writes to PTY.
	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after MsgChatSend, want 0 (T-154-02)", count)
	}
}

// TestChatSend_RODropped_RelayPath verifies SEC-01 relay path for chat:
// a RO client sending a hand-crafted MsgChatSend frame receives no broadcast
// and no NAK frame (silent drop per RESEARCH Open Question 1).
func TestChatSend_RODropped_RelayPath(t *testing.T) {
	ts, manager, sessionID, ptyWriteCount := setupInjectTestServer(t)

	// Dial a RO client.
	roConn := dialInjectWS(t, ts, sessionID, "readonly=1")

	// Dial a second RW client as a broadcast witness.
	rwConn := dialInjectWS(t, ts, sessionID, "")
	_ = manager // referenced to satisfy the return value

	time.Sleep(50 * time.Millisecond)

	// Send a hand-crafted MsgChatSend from the RO client.
	payload, _ := json.Marshal(ChatSendPayload{Content: "evil chat from RO"})
	frame := append([]byte{MsgChatSend}, payload...)
	ctx := context.Background()
	if err := roConn.Write(ctx, websocket.MessageBinary, frame); err != nil {
		t.Fatalf("roConn write MsgChatSend: %v", err)
	}

	// The RW witness must NOT receive a MsgChat broadcast within timeout.
	// We also check that the RO client doesn't receive one.
	// We wait 200ms to be confident no frame arrives.
	type result struct {
		frame []byte
		who   string
	}
	done := make(chan result, 2)
	checkConn := func(conn *websocket.Conn, who string) {
		go func() {
			deadlineCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			for {
				_, rawMsg, err := conn.Read(deadlineCtx)
				if err != nil {
					// deadline or closed — no MsgChat seen
					done <- result{nil, who}
					return
				}
				if len(rawMsg) > 0 && rawMsg[0] == MsgChat {
					done <- result{rawMsg, who}
					return
				}
			}
		}()
	}
	checkConn(roConn, "ro")
	checkConn(rwConn, "rw")

	for i := 0; i < 2; i++ {
		r := <-done
		if r.frame != nil {
			t.Errorf("%s client received unexpected MsgChat broadcast after RO chat send (SEC-01 relay path)", r.who)
		}
	}

	// PTY write count must also be zero.
	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after RO MsgChatSend, want 0", count)
	}
}

// TestChatSend_MalformedIgnored_RelayPath verifies that a MsgChatSend frame
// with a malformed or empty-content JSON body is silently ignored by the
// relay read pump, matching MsgTyping/MsgAliasSet behavior.
func TestChatSend_MalformedIgnored_RelayPath(t *testing.T) {
	ts, _, sessionID, ptyWriteCount := setupInjectTestServer(t)

	conn := dialInjectWS(t, ts, sessionID, "")
	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Case 1: malformed JSON (not JSON at all).
	malformedFrame := append([]byte{MsgChatSend}, []byte("NOT JSON")...)
	if err := conn.Write(ctx, websocket.MessageBinary, malformedFrame); err != nil {
		t.Fatalf("write malformed frame: %v", err)
	}

	// Case 2: valid JSON but empty content.
	emptyPayload, _ := json.Marshal(ChatSendPayload{Content: ""})
	emptyFrame := append([]byte{MsgChatSend}, emptyPayload...)
	if err := conn.Write(ctx, websocket.MessageBinary, emptyFrame); err != nil {
		t.Fatalf("write empty-content frame: %v", err)
	}

	// Wait briefly — no MsgChat or MsgInjectError should arrive.
	time.Sleep(150 * time.Millisecond)

	// Drain any remaining frames looking for unexpected MsgChat.
	drainCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(drainCtx)
		if err != nil {
			break // deadline
		}
		if len(rawMsg) > 0 && rawMsg[0] == MsgChat {
			t.Error("received unexpected MsgChat broadcast after malformed/empty MsgChatSend")
		}
		if len(rawMsg) > 0 && rawMsg[0] == MsgInjectError {
			t.Error("received unexpected MsgInjectError NAK after MsgChatSend (chat send must not NAK per RESEARCH Open Question 1)")
		}
	}

	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after malformed chat send, want 0", count)
	}
}
