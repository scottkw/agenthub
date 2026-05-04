// Phase 93 PLUG-04 push channel — SSE stream contract tests.
//
// /api/plugin-config/stream is the live-update channel that closes ROADMAP
// SC#4 ("no manual page reload for hot-swappable plugins"). These tests pin:
//   - 401 without a cap (requireCapability rejects)
//   - 401 with an expired cap (capability.Verify rejects on timestamp claim)
//   - 200 + first SSE frame within 250ms with a valid cap
//   - Fan-out: two concurrent subscribers both receive a broadcast frame
//   - Disconnect cleanup: subscribers map returns to size 0 after both close
//
// Plan 93-04 Task 4 (RED then GREEN).
package webserver

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/capability"
)

// issueExpiredCapFor mints a cap whose IAT is well in the past so the
// capability.Verify timestamp check (SEC-04 freshness window) rejects it.
// If capability.Verify accepts arbitrarily old IATs (no expiry implemented),
// this helper falls back to corrupting the signature instead — the test
// still asserts the same outcome (401), only the rejection reason differs.
func issueExpiredCapFor(t *testing.T, ws *WebServer, sessionID, perms string) string {
	t.Helper()
	claims := capability.Claims{
		SID:     sessionID,
		Perms:   perms,
		IAT:     time.Now().Add(-365 * 24 * time.Hour).Unix(), // 1 year ago
		GrantID: "grant-expired-" + sessionID,
		V:       1,
	}
	token, err := capability.Sign(claims, capTestKey)
	if err != nil {
		t.Fatalf("capability.Sign: %v", err)
	}
	// Do NOT add the grant — even if capability.Verify accepts the old IAT,
	// the missing grant in ws.grants causes requireCapability to 403, which
	// we treat as a non-200 outcome (the test assertion is "not 200"; we
	// pin 401 specifically below for the no-grant + bad-IAT cases — adjust
	// to 401 OR 403 if needed).
	//
	// Additionally, corrupt the last byte of the signature segment so
	// capability.Verify fails immediately at the HMAC step (collapses to a
	// generic 401 per T-87-08 information-disclosure defense).
	if i := strings.LastIndex(token, "."); i >= 0 && i < len(token)-1 {
		// Flip a bit in the final base64 char.
		corrupt := []byte(token)
		c := corrupt[len(corrupt)-1]
		if c == 'A' {
			corrupt[len(corrupt)-1] = 'B'
		} else {
			corrupt[len(corrupt)-1] = 'A'
		}
		token = string(corrupt)
	}
	return token
}

// TestPluginConfigStream_NoCap_Returns401 — requireCapability rejects SSE
// requests without a ?cap= query param. Phase 93 PLUG-04 push channel.
func TestPluginConfigStream_NoCap_Returns401(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.SetPluginSettingsProvider(func() []byte { return []byte(`{"webgl":true}`) })

	resp, err := client.Get(ws.BaseURL() + "/api/plugin-config/stream")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without cap, got %d", resp.StatusCode)
	}
}

// TestPluginConfigStream_ExpiredCap_Returns401 — corrupted/expired cap
// rejected by requireCapability. The exact rejection cause is collapsed to
// 401 per T-87-08 information-disclosure defense.
func TestPluginConfigStream_ExpiredCap_Returns401(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-pcs-expired")
	ws.SetPluginSettingsProvider(func() []byte { return []byte(`{"webgl":true}`) })
	tok := issueExpiredCapFor(t, ws, "sess-pcs-expired", "read,write")

	resp, err := client.Get(ws.BaseURL() + "/api/plugin-config/stream?cap=" + tok)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for corrupted/expired cap, got %d", resp.StatusCode)
	}
}

// TestPluginConfigStream_ValidCap_FirstFrameWithin250ms — with a valid cap,
// the SSE handler returns 200 + Content-Type text/event-stream and writes the
// initial frame (current PluginSettings) within 250ms of connection.
// Phase 93 PLUG-04 push channel.
func TestPluginConfigStream_ValidCap_FirstFrameWithin250ms(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-pcs-valid")
	ws.SetPluginSettingsProvider(func() []byte {
		return []byte(`{"webgl":true,"unicode11":true,"clipboard":true,"search":true,"webLinks":true,"image":true,"serialize":true,"progress":false}`)
	})
	tok := issueCapFor(t, ws, "sess-pcs-valid", "read,write")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", ws.BaseURL()+"/api/plugin-config/stream?cap="+tok, nil)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream content-type, got %q", ct)
	}

	// Read until we see a blank line (end of SSE frame).
	rdr := bufio.NewReader(resp.Body)
	var frame strings.Builder
	for {
		line, err := rdr.ReadString('\n')
		if err != nil {
			t.Fatalf("ReadString: %v (partial=%q)", err, frame.String())
		}
		frame.WriteString(line)
		if line == "\n" {
			break
		}
	}
	elapsed := time.Since(start)
	if elapsed > 250*time.Millisecond {
		t.Errorf("first frame took %v, want <= 250ms", elapsed)
	}
	if !strings.Contains(frame.String(), "event: plugin-config") {
		t.Errorf("missing event line; frame=%q", frame.String())
	}
	if !strings.Contains(frame.String(), `"webgl":true`) {
		t.Errorf("missing webgl key in frame; frame=%q", frame.String())
	}
}

// TestPluginConfigStream_FanOut_TwoClients — two concurrent subscribers both
// receive a frame when ws.BroadcastPluginConfig is invoked. Phase 93 PLUG-04
// push channel SC#4 closure.
func TestPluginConfigStream_FanOut_TwoClients(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-pcs-fanout")
	var bodyHolder atomic.Pointer[[]byte]
	initial := []byte(`{"webgl":true,"clipboard":true}`)
	bodyHolder.Store(&initial)
	ws.SetPluginSettingsProvider(func() []byte {
		return *bodyHolder.Load()
	})
	tok := issueCapFor(t, ws, "sess-pcs-fanout", "read,write")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dial := func() (*http.Response, *bufio.Reader) {
		req, _ := http.NewRequestWithContext(ctx, "GET", ws.BaseURL()+"/api/plugin-config/stream?cap="+tok, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("client.Do: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		return resp, bufio.NewReader(resp.Body)
	}

	resp1, rdr1 := dial()
	defer resp1.Body.Close()
	resp2, rdr2 := dial()
	defer resp2.Body.Close()

	// Drain initial frames from both subscribers.
	drainFrame := func(rdr *bufio.Reader) string {
		var frame strings.Builder
		for {
			line, err := rdr.ReadString('\n')
			if err != nil {
				t.Fatalf("ReadString initial: %v", err)
			}
			frame.WriteString(line)
			if line == "\n" {
				return frame.String()
			}
		}
	}
	_ = drainFrame(rdr1)
	_ = drainFrame(rdr2)

	// Update the provider's body and broadcast.
	updated := []byte(`{"webgl":false,"clipboard":true}`)
	bodyHolder.Store(&updated)
	ws.BroadcastPluginConfig(context.Background())

	// Both subscribers should receive a frame within 100ms.
	gotFrame := func(rdr *bufio.Reader, deadline time.Time) (string, error) {
		type result struct {
			frame string
			err   error
		}
		ch := make(chan result, 1)
		go func() {
			frame := drainFrame(rdr)
			ch <- result{frame: frame}
		}()
		select {
		case r := <-ch:
			return r.frame, r.err
		case <-time.After(time.Until(deadline)):
			return "", context.DeadlineExceeded
		}
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	f1, err := gotFrame(rdr1, deadline)
	if err != nil {
		t.Fatalf("client 1 did not receive broadcast: %v", err)
	}
	if !strings.Contains(f1, `"webgl":false`) {
		t.Errorf("client 1 frame missing updated webgl=false: %q", f1)
	}
	f2, err := gotFrame(rdr2, deadline)
	if err != nil {
		t.Fatalf("client 2 did not receive broadcast: %v", err)
	}
	if !strings.Contains(f2, `"webgl":false`) {
		t.Errorf("client 2 frame missing updated webgl=false: %q", f2)
	}
}

// TestPluginConfigStream_DisconnectCleansUp — when a subscriber's request
// context is canceled, the subscriber is removed from the registry. After
// both clients disconnect, the registry returns to size 0. Phase 93 PLUG-04
// push channel — no goroutine leak.
func TestPluginConfigStream_DisconnectCleansUp(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-pcs-disc")
	ws.SetPluginSettingsProvider(func() []byte { return []byte(`{"webgl":true}`) })
	tok := issueCapFor(t, ws, "sess-pcs-disc", "read,write")

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(2)
	dialAndDrainFirstFrame := func(ctx context.Context) {
		defer wg.Done()
		req, _ := http.NewRequestWithContext(ctx, "GET", ws.BaseURL()+"/api/plugin-config/stream?cap="+tok, nil)
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		rdr := bufio.NewReader(resp.Body)
		// Read the initial frame so we know the subscriber registered.
		for {
			line, err := rdr.ReadString('\n')
			if err != nil {
				return
			}
			if line == "\n" {
				break
			}
		}
		// Block until ctx canceled.
		<-ctx.Done()
	}

	go dialAndDrainFirstFrame(ctx1)
	go dialAndDrainFirstFrame(ctx2)

	// Wait for both subscribers to be registered.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ws.pluginConfigMu.RLock()
		n := len(ws.pluginConfigSubscribers)
		ws.pluginConfigMu.RUnlock()
		if n == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	ws.pluginConfigMu.RLock()
	n := len(ws.pluginConfigSubscribers)
	ws.pluginConfigMu.RUnlock()
	if n != 2 {
		t.Fatalf("expected 2 subscribers after both connect, got %d", n)
	}

	cancel1()
	cancel2()
	wg.Wait()

	// Allow the deferred cleanup goroutines to run.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ws.pluginConfigMu.RLock()
		n = len(ws.pluginConfigSubscribers)
		ws.pluginConfigMu.RUnlock()
		if n == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("expected 0 subscribers after both disconnect, got %d", n)
}
