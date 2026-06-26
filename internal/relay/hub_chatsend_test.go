package relay

// Phase 154 Plan 01 — Hub.HandleChatSend unit tests.
//
// These tests cover the four required behaviors:
//   1. RO subscriber → ErrChatReadOnly, no persist, no broadcast.
//   2. Empty-after-sanitize content → nil return, no persist, no broadcast (silent no-op).
//   3. RW subscriber + non-empty content → chatAppendFn called once + BroadcastChat once.
//   4. HandleChatSend NEVER calls WriteInput (no PTY write).
//
// Test harness mirrors the SetChatAppendFn pattern from server_inject_test.go.

import (
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

// makeChatSendTestHub creates a Hub backed by an io.Pipe and wires a minimal
// in-memory chatAppendFn. The appendCount counter is incremented on every
// chatAppendFn call so tests can assert exactly how many persists occurred.
// Returns: hub, the pre-subscribed RW subscriber, appendCount pointer, and a
// broadcast receiver channel (buffered 64).
//
// A subscriber is pre-subscribed so that BroadcastChat has somewhere to send.
func makeChatSendTestHub(t *testing.T) (*Hub, *Subscriber, *atomic.Int32, chan []byte) {
	t.Helper()

	r, w := io.Pipe()
	t.Cleanup(func() { w.Close() })

	const sessionID = "chatsend-test-session"
	hub := NewHub(sessionID, r, io.Discard, DefaultScrollbackBytes, nil)

	// Wire minimal chatAppendFn that fills ID/SchemaVersion fields like ChatStore would.
	var appendCount atomic.Int32
	hub.SetChatAppendFn(func(msg ChatMessage) (ChatMessage, error) {
		appendCount.Add(1)
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

	// Pre-subscribe a RW receiver so BroadcastChat can deliver frames.
	rxCh := make(chan []byte, 64)
	sub := &Subscriber{
		Msgs:      rxCh,
		CloseSlow: func() {},
		TailnetID: "test-tailnet-id",
		Alias:     "Tester",
		PersonKey: "test-tailnet-id:local",
	}
	hub.Subscribe(sub)

	return hub, sub, &appendCount, rxCh
}

// makePTYCountingHub creates a Hub where the writer is a counting writer,
// so tests can assert WriteInput was (or was not) called.
// Returns: hub, the pre-subscribed RW subscriber, and the PTY write counter.
func makePTYCountingHub(t *testing.T) (*Hub, *Subscriber, *atomic.Int32) {
	t.Helper()

	r, _ := io.Pipe()

	var ptyWriteCount atomic.Int32
	countingWriter := writerFunc(func(p []byte) (int, error) {
		ptyWriteCount.Add(1)
		return len(p), nil
	})

	const sessionID = "chatsend-pty-test"
	hub := NewHub(sessionID, r, countingWriter, DefaultScrollbackBytes, nil)

	hub.SetChatAppendFn(func(msg ChatMessage) (ChatMessage, error) {
		msg.ID = "test-id"
		msg.SchemaVersion = ChatSchemaVersion
		return msg, nil
	})

	sub := &Subscriber{
		Msgs:      make(chan []byte, 64),
		CloseSlow: func() {},
		TailnetID: "local",
		Alias:     "Owner",
		PersonKey: "local:local",
	}
	hub.Subscribe(sub)

	return hub, sub, &ptyWriteCount
}

// TestHandleChatSend_ROReturnsError verifies SEC-01 / T-154-03: a read-only subscriber
// calling HandleChatSend receives ErrChatReadOnly with no persist and no broadcast.
func TestHandleChatSend_ROReturnsError(t *testing.T) {
	hub, sub, appendCount, rxCh := makeChatSendTestHub(t)

	// Mark subscriber as read-only (set once at subscribe time, never mutated).
	sub.ReadOnly = true

	err := hub.HandleChatSend(sub, "hello from read-only client")

	if !errors.Is(err, ErrChatReadOnly) {
		t.Errorf("HandleChatSend RO: got err = %v, want ErrChatReadOnly", err)
	}
	if appendCount.Load() != 0 {
		t.Errorf("HandleChatSend RO: chatAppendFn called %d times, want 0", appendCount.Load())
	}
	// No broadcast frame should arrive on the receive channel.
	select {
	case frame := <-rxCh:
		t.Errorf("HandleChatSend RO: unexpected broadcast frame received: type=0x%02x", frame[0])
	case <-time.After(50 * time.Millisecond):
		// correct: no broadcast
	}
}

// TestHandleChatSend_EmptyAfterSanitize verifies the silent no-op behavior:
// content that collapses to empty after SanitizeChatContent returns nil and
// causes neither a persist nor a broadcast. Mirrors MsgTyping behavior.
func TestHandleChatSend_EmptyAfterSanitize(t *testing.T) {
	hub, sub, appendCount, rxCh := makeChatSendTestHub(t)

	// C0 control chars only — SanitizeChatContent strips them → empty string.
	controlOnly := "\x00\x01\x02\x1b\x7f"
	err := hub.HandleChatSend(sub, controlOnly)

	if err != nil {
		t.Errorf("HandleChatSend empty-after-sanitize: got err = %v, want nil", err)
	}
	if appendCount.Load() != 0 {
		t.Errorf("HandleChatSend empty-after-sanitize: chatAppendFn called %d times, want 0", appendCount.Load())
	}
	select {
	case frame := <-rxCh:
		t.Errorf("HandleChatSend empty-after-sanitize: unexpected broadcast frame type=0x%02x", frame[0])
	case <-time.After(50 * time.Millisecond):
		// correct: no broadcast
	}
}

// TestHandleChatSend_RWPersistsAndBroadcasts verifies the happy path:
// a RW subscriber with non-empty content causes chatAppendFn to be called
// exactly once and a MsgChat frame to be broadcast to all subscribers.
// AuthorID and AuthorAlias must match the subscriber's identity fields.
// SessionInject must be false (this is chat-send, not inject).
func TestHandleChatSend_RWPersistsAndBroadcasts(t *testing.T) {
	hub, sub, appendCount, rxCh := makeChatSendTestHub(t)

	const content = "hello from chat"
	err := hub.HandleChatSend(sub, content)
	if err != nil {
		t.Fatalf("HandleChatSend RW: unexpected error: %v", err)
	}

	// chatAppendFn must have been called exactly once.
	if appendCount.Load() != 1 {
		t.Errorf("HandleChatSend RW: chatAppendFn called %d times, want 1", appendCount.Load())
	}

	// A MsgChat broadcast frame must arrive on rxCh within timeout.
	var chatFrame []byte
	select {
	case f := <-rxCh:
		chatFrame = f
	case <-time.After(2 * time.Second):
		t.Fatal("HandleChatSend RW: timeout waiting for MsgChat broadcast")
	}

	// Validate the frame type byte.
	if len(chatFrame) == 0 || chatFrame[0] != MsgChat {
		t.Fatalf("HandleChatSend RW: broadcast frame type=0x%02x, want 0x%02x (MsgChat)", chatFrame[0], MsgChat)
	}

	// Decode and validate the ChatMessage payload.
	_, payload, err := ParseFrame(chatFrame)
	if err != nil {
		t.Fatalf("HandleChatSend RW: ParseFrame: %v", err)
	}
	var msg ChatMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("HandleChatSend RW: unmarshal ChatMessage: %v", err)
	}
	if msg.AuthorID != sub.TailnetID {
		t.Errorf("ChatMessage.AuthorID = %q, want %q", msg.AuthorID, sub.TailnetID)
	}
	if msg.AuthorAlias != sub.Alias {
		t.Errorf("ChatMessage.AuthorAlias = %q, want %q", msg.AuthorAlias, sub.Alias)
	}
	if msg.SessionInject {
		t.Error("ChatMessage.SessionInject = true, want false (this is chat-send, not inject)")
	}
	if msg.Content == "" {
		t.Error("ChatMessage.Content is empty, want non-empty sanitized content")
	}
}

// TestHandleChatSend_NoPTYWrite verifies T-154-02: HandleChatSend must never
// call WriteInput. The hub is constructed with a counting writer; after a
// successful HandleChatSend the PTY write counter must be exactly zero.
func TestHandleChatSend_NoPTYWrite(t *testing.T) {
	hub, sub, ptyWriteCount := makePTYCountingHub(t)

	if err := hub.HandleChatSend(sub, "should not reach PTY"); err != nil {
		t.Fatalf("HandleChatSend: unexpected error: %v", err)
	}

	// Give any asynchronous path time to fire — none should.
	time.Sleep(50 * time.Millisecond)

	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after HandleChatSend, want 0 (T-154-02: chat send must never write PTY)", count)
	}
}
