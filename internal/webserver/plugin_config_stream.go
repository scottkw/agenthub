// Phase 93 PLUG-04 push channel — Server-Sent Events stream for plugin-config
// changes. Closes ROADMAP SC#4 ("no manual page reload for hot-swappable
// plugins").
//
// Subscribers register a buffered channel in ws.pluginConfigSubscribers,
// flush an initial frame (the current PluginSettings) so the client can
// reconcile state without waiting for the next change, then stream frames
// until the request context is canceled. BroadcastPluginConfig (called by
// the daemon engine on SetPluginSettings) non-blocking-sends to every
// subscriber — slow consumers drop frames rather than blocking the
// broadcast.
package webserver

import (
	"context"
	"fmt"
	"net/http"
)

// sseChannelBuffer is the per-subscriber buffer depth. A slow consumer
// misses frames once full (drop-on-slow-consumer); fast consumers continue
// to receive. Sized at 4 to absorb burst sequences from rapid toggle clicks
// while remaining tiny per-connection.
const sseChannelBuffer = 4

// handleStreamPluginConfig serves an SSE stream of plugin-config changes.
// Phase 93 PLUG-04 push channel.
//
// Frame format: `event: plugin-config\ndata: <json>\n\n` (per W3C SSE spec).
// Browsers' EventSource API delivers the JSON via addEventListener('plugin-config').
//
// requireCapability has already verified the cap; if the provider is nil or
// returns nil, return 503 (web client falls back to fetch-on-load defaults).
func (ws *WebServer) handleStreamPluginConfig(w http.ResponseWriter, r *http.Request) {
	if ws.pluginSettingsProvider == nil {
		http.Error(w, "plugin config unavailable", http.StatusServiceUnavailable)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering

	ch := make(chan []byte, sseChannelBuffer)
	ws.pluginConfigMu.Lock()
	ws.pluginConfigSubscribers[ch] = struct{}{}
	ws.pluginConfigMu.Unlock()
	defer func() {
		ws.pluginConfigMu.Lock()
		delete(ws.pluginConfigSubscribers, ch)
		ws.pluginConfigMu.Unlock()
	}()

	// Send the initial frame (current settings) immediately so the client
	// can reconcile state without waiting for the next change-event.
	// Acceptance test pins this within 250ms of connection.
	if body := ws.pluginSettingsProvider(); body != nil {
		if _, err := fmt.Fprintf(w, "event: plugin-config\ndata: %s\n\n", body); err != nil {
			return
		}
		flusher.Flush()
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case body, ok := <-ch:
			if !ok || body == nil {
				return
			}
			if _, err := fmt.Fprintf(w, "event: plugin-config\ndata: %s\n\n", body); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// BroadcastPluginConfig delivers the current PluginSettings JSON to every
// active SSE subscriber. Non-blocking: a subscriber whose channel buffer is
// full silently misses this frame (drop-on-slow-consumer) — the next
// change-event triggers another broadcast attempt; persistent slowness is
// bounded.
//
// Called by the daemon engine after SetPluginSettings completes (wired in
// internal/daemon/api.go at the NewWebServer call sites via
// engine.SetPluginSettingsListener).
//
// The ctx parameter is currently unused; it is part of the signature so a
// future implementation can plumb cancellation into the per-subscriber send
// (e.g. a select with a global timeout) without an API break.
func (ws *WebServer) BroadcastPluginConfig(_ context.Context) {
	if ws.pluginSettingsProvider == nil {
		return
	}
	body := ws.pluginSettingsProvider()
	if body == nil {
		return
	}
	ws.pluginConfigMu.RLock()
	defer ws.pluginConfigMu.RUnlock()
	for ch := range ws.pluginConfigSubscribers {
		select {
		case ch <- body:
		default:
			// drop-on-slow-consumer
		}
	}
}
