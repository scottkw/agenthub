package webserver

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/relay"
)

// Phase 111 / Issue #54: integration tests that exercise the full WS read pump
// from MakeInputFrame(...) on the wire down to the PTY-input pipe assertion.
// Each test shape mirrors capability_test.go:TestSecurity_ReadOnlyCapabilityBlocksMsgInput
// — the only difference is the cap perms (read,write) and the payload bytes.

// originHeader is a tiny convenience used by every relay-test to satisfy
// the requireAllowedOrigin middleware.
func originHeader(baseURL string) http.Header {
	h := http.Header{}
	h.Set("Origin", baseURL)
	return h
}

func TestRelay_OSC10ReplyAbsorbedBeforePTY(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-osc10")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-osc10", "read,write")

	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-osc10/ws?cap="+token,
		originHeader(ws.BaseURL()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := []byte("\x1b]10;rgb:cccc/cccc/cccc\x1b\\")
	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	readPipeMustTimeout(t, inputReader, 300*time.Millisecond,
		"OSC 10 reply must be absorbed before reaching PTY")
}

func TestRelay_OSC11ReplyAbsorbedBeforePTY(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-osc11")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-osc11", "read,write")

	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-osc11/ws?cap="+token,
		originHeader(ws.BaseURL()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := []byte("\x1b]11;rgb:cccc/cccc/cccc\x1b\\")
	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	readPipeMustTimeout(t, inputReader, 300*time.Millisecond,
		"OSC 11 reply must be absorbed before reaching PTY")
}

func TestRelay_DA1ReplyAbsorbedBeforePTY(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-da1")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-da1", "read,write")

	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-da1/ws?cap="+token,
		originHeader(ws.BaseURL()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := []byte("\x1b[?1;2c")
	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	readPipeMustTimeout(t, inputReader, 300*time.Millisecond,
		"DA1 reply must be absorbed before reaching PTY")
}

// TestRelay_KeystrokesStillForwarded — regression guard. Wiring change must
// NOT block normal user input.
func TestRelay_KeystrokesStillForwarded(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-keys")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-keys", "read,write")

	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-keys/ws?cap="+token,
		originHeader(ws.BaseURL()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	want := []byte("hello\r")
	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame(want)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	got := readPipeWithTimeout(t, inputReader, 1*time.Second)
	if !bytes.Equal(got, want) {
		t.Errorf("keystrokes regression: want %q got %q", want, got)
	}
}

// TestRelay_OSC11SplitAcrossTwoFrames proves per-subscriber state survives
// across MsgInput frame boundaries: send two halves of an OSC 11 reply in two
// separate WebSocket binary frames; assert ZERO bytes reach the PTY pipe.
func TestRelay_OSC11SplitAcrossTwoFrames(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-split")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-split", "read,write")

	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-split/ws?cap="+token,
		originHeader(ws.BaseURL()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	frame1 := relay.MakeInputFrame([]byte("\x1b]11;rgb:cccc/"))
	frame2 := relay.MakeInputFrame([]byte("cccc/cccc\x1b\\"))
	if err := conn.Write(ctx, websocket.MessageBinary, frame1); err != nil {
		t.Fatalf("write frame1: %v", err)
	}
	// Give the read pump time to process frame1 independently.
	time.Sleep(20 * time.Millisecond)
	if err := conn.Write(ctx, websocket.MessageBinary, frame2); err != nil {
		t.Fatalf("write frame2: %v", err)
	}
	readPipeMustTimeout(t, inputReader, 300*time.Millisecond,
		"OSC 11 reply split across two WS frames must be absorbed in full")
}

// TestRelay_MixedReplyAndKeystrokes — one MsgInput frame carrying real keystrokes
// before AND after an OSC 11 reply. Only the keystrokes should reach the pipe.
func TestRelay_MixedReplyAndKeystrokes(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-mixed")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-mixed", "read,write")

	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-mixed/ws?cap="+token,
		originHeader(ws.BaseURL()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload := []byte("ls\r\x1b]11;rgb:cccc/cccc/cccc\x1b\\pwd\r")
	want := []byte("ls\rpwd\r")
	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame(payload)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}

	// Drain the pipe until we either have len(want) bytes OR we exceed a
	// wall-clock budget. readPipeWithTimeout reads up to 1024 bytes per call,
	// which is enough for "ls\rpwd\r" in a single read, but we loop defensively.
	var got bytes.Buffer
	deadline := time.Now().Add(1 * time.Second)
	for got.Len() < len(want) && time.Now().Before(deadline) {
		chunk := readPipeWithTimeout(t, inputReader, 500*time.Millisecond)
		got.Write(chunk)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Errorf("mixed reply+keystrokes: want %q got %q", want, got.Bytes())
	}
}
