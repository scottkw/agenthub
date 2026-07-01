package daemon

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/relay"
)

// TestAPIDisconnectViewers_ClosesOnlyWebOrigin asserts POST
// /sessions/{id}/disconnect-viewers is served on the daemon-local mux and
// drives (*Hub).DisconnectWebViewers() for the target session: web-origin
// subscribers are closed, the local-origin subscriber is untouched. Phase 168
// / FIX-02, #117.
func TestAPIDisconnectViewers_ClosesOnlyWebOrigin(t *testing.T) {
	api, _, socketPath := testDaemon(t)

	status, body := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"disconnect-test","workDir":""}`)
	if status != 201 {
		t.Fatalf("POST /sessions: want 201, got %d; body: %s", status, body)
	}
	var cr CreateResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	t.Cleanup(func() { rawDelete(t, socketPath, "/sessions/"+cr.ID) })

	hub, ok := api.engine.Manager().Get(cr.ID)
	if !ok {
		t.Fatalf("hub for session %s not found in manager", cr.ID)
	}

	var mu sync.Mutex
	webClosed, localClosed := false, false

	webSub := &relay.Subscriber{
		Msgs: make(chan []byte, 256),
		CloseSlow: func() {
			mu.Lock()
			defer mu.Unlock()
			webClosed = true
		},
		Origin: "web",
	}
	localSub := &relay.Subscriber{
		Msgs: make(chan []byte, 256),
		CloseSlow: func() {
			mu.Lock()
			defer mu.Unlock()
			localClosed = true
		},
		Origin: "local",
	}
	hub.Subscribe(webSub)
	hub.Subscribe(localSub)

	status, body = rawPost(t, socketPath, "/sessions/"+cr.ID+"/disconnect-viewers", "")
	if status != 204 {
		t.Fatalf("POST disconnect-viewers: want 204, got %d; body: %s", status, body)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		got := webClosed
		mu.Unlock()
		if got {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for web-origin subscriber to be closed")
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if localClosed {
		t.Fatal("disconnect-viewers closed the local-origin subscriber — must be untouched (D-05)")
	}
}

// TestAPIDisconnectViewers_UnknownSession asserts a 404 for a session ID with
// no active hub, rather than a panic or 204.
func TestAPIDisconnectViewers_UnknownSession(t *testing.T) {
	_, _, socketPath := testDaemon(t)

	status, _ := rawPost(t, socketPath, "/sessions/does-not-exist/disconnect-viewers", "")
	if status != 404 {
		t.Fatalf("POST disconnect-viewers for unknown session: want 404, got %d", status)
	}
}
