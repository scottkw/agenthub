package webserver

// Phase 154 Plan 01 / Phase 163 Plan 01 — webserver read-pump MsgChatSend dispatch tests.
//
// Tests drive a real webserver WebSocket subscriber through the read pump
// and assert:
//   1. A MsgChatSend frame from a RW JWT client causes a MsgChat broadcast.
//   2. A MsgChatSend frame from a RO JWT client ALSO causes a MsgChat broadcast
//      (D-06 reconciliation, Phase 163: RO clients are full chat participants).
//   3. A MsgChatSend frame with malformed/empty-content JSON is silently ignored.
//
// The RO status derives from claims.Perms in the signed JWT (same as
// TestInjectRO_WebPath). The SEC-01 inject gate remains in hub.HandleInject;
// only the chat-send gate was loosened per D-06.
//
// TESTING.md registration is deferred to plan 154-06.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/relay"
)

// writerFuncChat is a func([]byte)(int,error) adapter that implements io.Writer.
// Used to back the hub with a counting PTY writer; distinct from writerFuncInj
// (inject_test.go) to avoid duplicate identifier in the same test binary.
type writerFuncChat func(p []byte) (int, error)

func (f writerFuncChat) Write(p []byte) (int, error) { return f(p) }

// setupChatSendTestServer creates a tailscale-mode WebServer backed by a hub
// with a counting PTY writer and an in-memory chatAppendFn. Returns: server,
// HTTP client, session ID, and the PTY write counter.
func setupChatSendTestServer(t *testing.T) (*WebServer, *http.Client, string, *atomic.Int32) {
	t.Helper()

	const sessionID = "chatsend-web-session"

	var ptyWriteCount atomic.Int32
	countingWriter := writerFuncChat(func(p []byte) (int, error) {
		ptyWriteCount.Add(1)
		return len(p), nil
	})

	manager := relay.NewHubManager()
	ptyOutputR, ptyOutputW := io.Pipe()
	hub := manager.Create(sessionID, ptyOutputR, countingWriter, nil)

	// Wire a minimal chatAppendFn that fills ID/SchemaVersion like ChatStore would.
	hub.SetChatAppendFn(func(msg relay.ChatMessage) (relay.ChatMessage, error) {
		if msg.ID == "" {
			msg.ID = "test-id"
		}
		if msg.SessionID == "" {
			msg.SessionID = sessionID
		}
		if msg.SchemaVersion == 0 {
			msg.SchemaVersion = relay.ChatSchemaVersion
		}
		if msg.TimestampMs == 0 {
			msg.TimestampMs = 1
		}
		return msg, nil
	})

	tlsCfg, client := selfSignedTLSForTest(t)
	cfg := Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		Mode:      "tailscale",
		TLSConfig: tlsCfg,
	}
	ws, err := NewWebServer(cfg, manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	ws.SetSigningKey(capTestKey)
	ws.EnableSession(sessionID)

	t.Cleanup(func() {
		_ = ws.Stop()
		manager.Shutdown()
		_ = ptyOutputW.Close()
	})

	return ws, client, sessionID, &ptyWriteCount
}

// waitForMsgChatWebServer reads frames from conn, skipping non-MsgChat frames,
// and returns the first relay.MsgChat frame received. Returns nil if the
// context deadline passes without a MsgChat frame.
func waitForMsgChatWebServer(t *testing.T, conn *websocket.Conn, timeout time.Duration) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(ctx)
		if err != nil {
			return nil // deadline or connection closed
		}
		if len(rawMsg) > 0 && rawMsg[0] == relay.MsgChat {
			return rawMsg
		}
	}
}

// TestChatSend_RWBroadcasts_WebPath verifies the happy path on the webserver
// read-pump: a RW JWT client sending MsgChatSend receives a MsgChat broadcast.
func TestChatSend_RWBroadcasts_WebPath(t *testing.T) {
	ws, client, sessionID, ptyWriteCount := setupChatSendTestServer(t)

	// Mint a RW capability token.
	token := issueCapFor(t, ws, sessionID, "read,write")

	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())
	conn := dialWebServerWS(t, client, ws.BaseURL(),
		"/sessions/"+sessionID+"/ws?cap="+token, headers)

	time.Sleep(50 * time.Millisecond)

	// Send a MsgChatSend frame.
	payload, _ := json.Marshal(relay.ChatSendPayload{Content: "webserver chat send test"})
	frame := append([]byte{relay.MsgChatSend}, payload...)

	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
		t.Fatalf("write MsgChatSend: %v", err)
	}

	// A MsgChat broadcast must arrive within 3 seconds.
	chatFrame := waitForMsgChatWebServer(t, conn, 3*time.Second)
	if chatFrame == nil {
		t.Fatal("TestChatSend_RWBroadcasts_WebPath: timeout waiting for MsgChat broadcast")
	}
	if chatFrame[0] != relay.MsgChat {
		t.Errorf("frame type = 0x%02x, want 0x%02x (MsgChat)", chatFrame[0], relay.MsgChat)
	}

	// Decode and verify: SessionInject must be false.
	_, chatPayload, err := relay.ParseFrame(chatFrame)
	if err != nil {
		t.Fatalf("ParseFrame MsgChat: %v", err)
	}
	var msg relay.ChatMessage
	if err := json.Unmarshal(chatPayload, &msg); err != nil {
		t.Fatalf("unmarshal ChatMessage: %v", err)
	}
	if msg.SessionInject {
		t.Error("ChatMessage.SessionInject = true, want false (chat-send is not inject)")
	}
	if msg.Content != "webserver chat send test" {
		t.Errorf("ChatMessage.Content = %q, want %q", msg.Content, "webserver chat send test")
	}

	// PTY write count must be exactly zero.
	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after MsgChatSend, want 0 (T-154-02)", count)
	}
}

// TestChatSend_ROCanPost_WebPath verifies D-06 (Phase 163) on the webserver path:
// a RO JWT client sending MsgChatSend DOES receive a MsgChat broadcast and must
// NOT receive a MsgInjectError NAK. PTY write count must remain zero.
// Both "read" and "read,files.read" perm shapes are exercised (mirrors TestInjectRO_WebPath).
func TestChatSend_ROCanPost_WebPath(t *testing.T) {
	roPerms := []struct {
		name  string
		perms string
	}{
		{"browse_off", "read"},
		{"browse_on", "read,files.read"},
	}

	for _, tc := range roPerms {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ws, client, sessionID, ptyWriteCount := setupChatSendTestServer(t)

			// Mint a RO capability token.
			token := issueCapFor(t, ws, sessionID, tc.perms)

			headers := http.Header{}
			headers.Set("Origin", ws.BaseURL())
			conn := dialWebServerWS(t, client, ws.BaseURL(),
				"/sessions/"+sessionID+"/ws?cap="+token, headers)

			time.Sleep(50 * time.Millisecond)

			// Send a MsgChatSend from the RO client.
			payload, _ := json.Marshal(relay.ChatSendPayload{Content: "chat from RO web client"})
			frame := append([]byte{relay.MsgChatSend}, payload...)
			ctx := context.Background()
			if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
				t.Fatalf("perms=%q: write MsgChatSend: %v", tc.perms, err)
			}

			// A MsgChat broadcast MUST arrive within timeout; no MsgInjectError NAK.
			chatFrame := waitForMsgChatWebServer(t, conn, 3*time.Second)
			if chatFrame == nil {
				t.Errorf("perms=%q: expected MsgChat broadcast after RO chat send (D-06 web path), got none", tc.perms)
			} else if chatFrame[0] != relay.MsgChat {
				t.Errorf("perms=%q: frame type = 0x%02x, want 0x%02x (MsgChat)", tc.perms, chatFrame[0], relay.MsgChat)
			}

			// Drain for NAK check — no MsgInjectError should arrive.
			drainCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			for {
				_, rawMsg, err := conn.Read(drainCtx)
				if err != nil {
					break // deadline
				}
				if len(rawMsg) > 0 && rawMsg[0] == relay.MsgInjectError {
					t.Errorf("perms=%q: received unexpected MsgInjectError NAK after MsgChatSend (chat send must not NAK)", tc.perms)
					break
				}
			}

			if count := ptyWriteCount.Load(); count != 0 {
				t.Errorf("perms=%q: PTY write count = %d after RO chat send, want 0 (T-154-02)", tc.perms, count)
			}
		})
	}
}

// TestChatSend_MalformedIgnored_WebPath verifies that a MsgChatSend frame
// with malformed or empty-content JSON is silently ignored by the webserver
// read pump (no MsgChat broadcast, no NAK).
func TestChatSend_MalformedIgnored_WebPath(t *testing.T) {
	ws, client, sessionID, ptyWriteCount := setupChatSendTestServer(t)

	token := issueCapFor(t, ws, sessionID, "read,write")
	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())
	conn := dialWebServerWS(t, client, ws.BaseURL(),
		"/sessions/"+sessionID+"/ws?cap="+token, headers)

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()

	// Case 1: malformed JSON payload.
	malformedFrame := append([]byte{relay.MsgChatSend}, []byte("NOT JSON")...)
	if err := conn.Write(ctx, websocket.MessageBinary, malformedFrame); err != nil {
		t.Fatalf("write malformed frame: %v", err)
	}

	// Case 2: valid JSON but empty content field.
	emptyPayload, _ := json.Marshal(relay.ChatSendPayload{Content: ""})
	emptyFrame := append([]byte{relay.MsgChatSend}, emptyPayload...)
	if err := conn.Write(ctx, websocket.MessageBinary, emptyFrame); err != nil {
		t.Fatalf("write empty-content frame: %v", err)
	}

	// Drain frames for 300ms — no MsgChat or MsgInjectError must appear.
	drainCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(drainCtx)
		if err != nil {
			break
		}
		if len(rawMsg) > 0 && rawMsg[0] == relay.MsgChat {
			t.Error("received unexpected MsgChat broadcast after malformed/empty MsgChatSend")
		}
		if len(rawMsg) > 0 && rawMsg[0] == relay.MsgInjectError {
			t.Error("received unexpected MsgInjectError after MsgChatSend (chat send must not NAK)")
		}
	}

	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after malformed chat send, want 0", count)
	}
}
