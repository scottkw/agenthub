// Phase 93 PLUG-04 / WEB-03 — capability-gated /api/plugin-config endpoint.
//
// Serves the daemon's current PluginSettings as pre-marshaled JSON to any
// caller bearing a valid capability token. The web terminal page fetches
// this at load to gate addon instantiation (webgl, unicode11, clipboard).
//
// Sourcing: ws.pluginSettingsProvider, set once at daemon startup via
// SetPluginSettingsProvider. The func() []byte signature is deliberate —
// avoids importing daemon.PluginSettings into the webserver package
// (would create a cycle since daemon imports webserver).
package webserver

import (
	"net/http"
)

// handleGetPluginConfig serves the daemon's current plugin settings as JSON.
// Phase 93 PLUG-04 / WEB-03.
//
// requireCapability has already verified the cap; if pluginSettingsProvider
// is nil (daemon never wired) or returns nil (e.g. json.Marshal failed),
// return 503 so the web client can fall through to its built-in defaults
// (per 93-RESEARCH §"WEB-03 — Web terminal conditionally loads addons from
// /api/plugin-config").
func (ws *WebServer) handleGetPluginConfig(w http.ResponseWriter, r *http.Request) {
	if ws.pluginSettingsProvider == nil {
		http.Error(w, "plugin config unavailable", http.StatusServiceUnavailable)
		return
	}
	body := ws.pluginSettingsProvider()
	if body == nil {
		http.Error(w, "plugin config unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store") // server-shared config; never cache
	_, _ = w.Write(body)
}
