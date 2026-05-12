---
phase: 93
plan: 04
subsystem: webserver
tags:
  - plug-04
  - web-03
  - u11-02
  - clip-02
  - sse
  - hot-swap
requires:
  - 93-02 (vendored addon UMD bundles + script tags)
  - phase 92 (PluginSettings struct + GetPluginSettings/SetPluginSettings on engine)
provides:
  - GET /api/plugin-config (capability-gated JSON read)
  - GET /api/plugin-config/stream (capability-gated SSE push channel)
  - WebServer.SetPluginSettingsProvider (provider closure pattern)
  - WebServer.BroadcastPluginConfig (subscriber fan-out)
  - SessionEngine.SetPluginSettingsListener (single-slot change-listener)
  - Web terminal applyPluginConfig() (diff-applying hot-swap function)
  - Web terminal EventSource subscription with reload-free hot-swap
  - #webgl-recovery-banner web parity with desktop component
affects:
  - Closes ROADMAP SC#4 (no manual page reload for hot-swappable plugins)
  - Mitigates T-93-PLUG-04 (info disclosure) + T-93-PLUG-04-PUSH (tampering)
  - Web terminal page now reactive to desktop Settings toggles in real time
tech-stack:
  added: []
  patterns:
    - SSE push channel with drop-on-slow-consumer + bounded per-subscriber buffer (4 frames)
    - Single-listener slot in Engine (NewWebServer sites are mutually exclusive at runtime)
    - func() []byte provider closure to break daemon→webserver→daemon import cycle
    - Diff-applying hot-swap function reused for both initial-load and SSE push paths
    - Defensive additive merge (push frame cannot disable user-on plugins via partial body)
key-files:
  created:
    - internal/webserver/plugin_config.go
    - internal/webserver/plugin_config_test.go
    - internal/webserver/plugin_config_stream.go
    - internal/webserver/plugin_config_stream_test.go
    - .planning/phases/93-vendoring-discipline-web-parity-for-already-shipping-addons/93-04-SUMMARY.md
  modified:
    - internal/webserver/server.go
    - internal/webserver/assets_test.go
    - internal/daemon/api.go
    - internal/daemon/engine.go
    - web/terminal.html
    - web/assets/terminal.js
    - web/assets/terminal.css
decisions:
  - "Single-listener slot in Engine (not slice) — the two NewWebServer call sites in api.go are mutually exclusive at runtime (AutoStartWebServer for cold-start vs. handleWebServerStart for runtime mode-switch), so a single slot is safe. Documented in SetPluginSettingsListener comment for future maintainers."
  - "Provider returns []byte (not PluginSettings) — keeps webserver package free of daemon import, breaking the otherwise-circular daemon→webserver→daemon dependency."
  - "Drop-on-slow-consumer over unbounded buffering — a non-responsive subscriber's missed frame is recoverable by the next change-event. Unbounded buffering would leak memory if a subscriber stalls indefinitely on a slow link."
  - "Per-subscriber buffer = 4 frames — absorbs a burst of rapid toggle clicks without blocking. Sized small to keep per-connection memory tiny."
  - "Unicode 11 deliberately NOT hot-swapped on push — server-shared-config semantics mean a running session's width tables must not change mid-buffer (would corrupt scrollback rendering of already-painted wide chars). Cached-only — next page load picks up the change."
  - "Defensive additive merge in EventSource handler (not replacement) — partial frames cannot disable plugins the user has on (T-93-WEB-03 mitigation)."
  - "Banner DOM construction via createElement + textContent (not innerHTML) — XSS-safe even though all banner copy is hard-coded constants."
  - "SSE listener invocation MOVED OUTSIDE engine mutex — original draft held e.mu.Lock() through the listener call, which would have deadlocked any listener that called back into engine. Now SetPluginSettings unlocks first, then invokes the listener."
metrics:
  duration_min: 19
  duration_human: "19 minutes"
  tasks: 5
  files_created: 5
  files_modified: 7
  commits: 5
  completed_date: "2026-05-04"
  red_green_cycles: 2
---

# Phase 93 Plan 04: Web Plugin-Config Endpoint + SSE Push Channel Summary

Capability-gated `/api/plugin-config` JSON read endpoint and `/api/plugin-config/stream` SSE push channel landed; web terminal page fetches at load and live-applies hot-swap on each push frame WITHOUT a page reload. Closes ROADMAP SC#4 and rounds out PLUG-04 / WEB-03 / U11-02 / CLIP-02.

## What Shipped

**Server-side (Go):**

- **`GET /api/plugin-config`** — read endpoint serving the daemon's current `PluginSettings` as pre-marshaled JSON. Capability-gated via the existing `requireCapability` middleware (no `{id}` path segment, so the empty-pathID short-circuit applies — any valid cap holder gets it). Returns `503 plugin config unavailable` when `SetPluginSettingsProvider` was never called or the provider returns nil (web client falls back to all-on defaults). `Cache-Control: no-store` to prevent stale serves through any intermediate cache.
- **`GET /api/plugin-config/stream`** — SSE push channel. Same capability gate. First frame (current settings) flushed immediately so the client can reconcile state without waiting for the next change. Frame format: `event: plugin-config\ndata: <json>\n\n`. Subscriber registry uses a `map[chan []byte]struct{}` under `sync.RWMutex`; per-subscriber channel buffer = 4 frames; `Broadcast` does non-blocking sends with drop-on-slow-consumer; cleanup is `defer`-driven via `r.Context().Done()`.
- **`WebServer.SetPluginSettingsProvider(func() []byte)`** — provider-closure setter. The `func() []byte` signature (rather than `func() daemon.PluginSettings`) avoids the daemon→webserver→daemon import cycle. Daemon `api.go` registers the closure at both `NewWebServer` call sites with `engine.GetPluginSettings()` + `json.Marshal()`.
- **`WebServer.BroadcastPluginConfig(ctx)`** — fan-out helper. Daemon registers it as the engine's plugin-settings change listener.
- **`SessionEngine.SetPluginSettingsListener(func())`** — single-slot listener registered after `SetPluginSettings` persists the new value. Listener invoked AFTER the engine mutex is released (slow listener cannot deadlock concurrent engine ops).

**Client-side (Web terminal page):**

- **`web/terminal.html`** — added `<div id="webgl-recovery-banner" hidden role="status" aria-live="polite">` between the status bar and `#terminal`. Inner content built by JS via DOM API (XSS-safe).
- **`web/assets/terminal.css`** — `#webgl-recovery-banner` styling matches the desktop `.webgl-recovery-banner` pixel-for-pixel: 53px fixed height, `#7aa2f7` left border, TokyoNight palette, focus-visible outline, `prefers-reduced-motion` respected.
- **`web/assets/terminal.js`** — five additions:
  - Async fetch of `/api/plugin-config` after the existing perms fetch, with all-on defensive defaults on any failure path.
  - `isSoftwareWebGL()` probe (matches desktop `webglProbe.ts`).
  - `showWebGLBanner(reason)` builds DOM via `createElement` + `textContent`; one-shot per session via `sessionStorage`; auto-dismiss 8s for context-loss only; verbatim copy from UI-SPEC.
  - **`applyPluginConfig(newConfig)`** — diff-applying function. Reused for BOTH initial-load (seeded against everything-off prev) AND every SSE push frame. Handles WebGL dispose/reload + Clipboard dispose/reload + clipboard CLIP-02 gate (`window.__perms !== 'read'`); Unicode 11 deliberately cached-only.
  - `EventSource(withCap('/api/plugin-config/stream'))` opened after initial load. `addEventListener('plugin-config', ...)` does defensive additive merge → idempotent guard via `lastApplied` string compare → `applyPluginConfig`. `error` handler closes the stream on permanent 401. `beforeunload` handler closes cleanly so the server-side subscriber registry deregisters via `r.Context().Done()`.

## UMD Constructor Names (Confirmed Verbatim from 93-02-SUMMARY)

Plan 93-02's source-inspection of the UMD wrappers verified these globals; Plan 93-04 uses them unchanged:

| Vendored bundle | Window namespace | Constructor path |
| --- | --- | --- |
| addon-webgl.js | `window.WebglAddon` | `new WebglAddon.WebglAddon()` |
| addon-unicode11.js | `window.Unicode11Addon` | `new Unicode11Addon.Unicode11Addon()` |
| addon-clipboard.js | `window.ClipboardAddon` | `new ClipboardAddon.ClipboardAddon()` |

No pivots required — the assumed names matched.

## Commits

| Task | Description | Hash |
| ---- | ----------- | ---- |
| 1 | RED: tests for /api/plugin-config + vendored addons | `6ff28c3` |
| 2 | GREEN: /api/plugin-config handler + daemon wiring | `3e02803` |
| 3 | Web terminal conditional addon loading + recovery banner | `bcd50ba` |
| 4a | RED: tests for /api/plugin-config/stream SSE channel | `659d0a0` |
| 4b | GREEN: SSE push channel + engine listener slot | `2558186` |
| 5 | Web terminal EventSource subscription for live hot-swap | `e899f60` |

## Tests

All new tests PASS. Full webserver, daemon, and root packages remain GREEN — no regressions.

**New test functions:**

- `TestPluginConfig_NoCap_Returns401`
- `TestPluginConfig_ValidCap_Returns200JSON`
- `TestPluginConfig_NoProvider_Returns503`
- `TestPluginConfig_NilProvider_Returns503`
- `TestAssets_VendoredAddons` (3 paths × 2 assertions = 6 cases)
- `TestPluginConfigStream_NoCap_Returns401`
- `TestPluginConfigStream_ExpiredCap_Returns401`
- `TestPluginConfigStream_ValidCap_FirstFrameWithin250ms`
- `TestPluginConfigStream_FanOut_TwoClients`
- `TestPluginConfigStream_DisconnectCleansUp`

**First-frame latency (3 runs):** 0.02s, 0.00s, 0.00s — well under the 250ms acceptance threshold.

## Decisions Made

1. **Single-listener slot in Engine, not a slice.** The two `NewWebServer` call sites in `api.go` (`AutoStartWebServer` for cold-start auto-launch, `handleWebServerStart` for runtime mode-switch) are mutually exclusive at runtime — only one webserver instance ever exists. A single slot is safe; the slot is overwritten on the second `NewWebServer` so the dead webserver's broadcast hook is harmlessly orphaned.
2. **Provider returns `[]byte` (pre-marshaled JSON), not `daemon.PluginSettings`.** The daemon package imports webserver, so importing `daemon.PluginSettings` from webserver would create a cycle. The `func() []byte` signature breaks the cycle while still letting the provider closure access the engine.
3. **Drop-on-slow-consumer instead of unbounded buffering.** A non-responsive subscriber's missed frame is recoverable by the next change-event broadcast. Unbounded buffering would leak memory if a subscriber stalls forever on a slow link.
4. **Per-subscriber buffer = 4 frames.** Absorbs a burst of rapid toggle clicks (user clicking through multiple Settings toggles) without blocking the broadcast loop. Tiny per-connection memory cost.
5. **Unicode 11 NOT hot-swapped on SSE push.** Per the server-shared-config decision in the phase plan: a running session's width tables must not change mid-buffer (would corrupt scrollback rendering of already-painted wide chars). Cached-only — next page load picks up the change. Documented inside `applyPluginConfig`.
6. **Defensive additive merge in the EventSource handler.** A partial frame cannot disable plugins the user has on (T-93-WEB-03 mitigation). The merge starts from current `pluginConfig`, then overlays the pushed keys; missing keys preserve their current state.
7. **Banner DOM via `createElement` + `textContent`, never `innerHTML` with dynamic content.** Even though all banner copy is hard-coded constants, the safer construction path is required by `TestSecurity_NoInlineScriptOrStyleInHTML` and locks future maintainers into the XSS-safe pattern.
8. **`SetPluginSettings` invokes the listener AFTER releasing the engine mutex.** Original draft held `e.mu.Lock()` through the listener call, which would have deadlocked any listener that called back into engine. Restructured to capture the listener under the lock, unlock, then invoke. Documented in the function comment.

## no_cdn_regression_test.go — No Edit Needed

Confirmed by reading `internal/webserver/no_cdn_regression_test.go:50` — the existing skip uses `strings.HasSuffix(path, "vendor/xterm")` and returns `filepath.SkipDir`, which skips the entire `vendor/xterm/` subtree including the new `vendor/xterm/addons/` subdir naturally. Same-origin path-only references like `/api/plugin-config/stream` in `terminal.js` are not flagged as CDN references (the guards check for `cdn.jsdelivr`, `unpkg.com`, `://cdn.`, etc.). No edit required; `TestSecurity_NoCDNReferencesInWebAssets` remains GREEN.

## Deviations from Plan

None — plan executed exactly as written. The only minor adjustment was placing the listener invocation in `SessionEngine.SetPluginSettings` AFTER `e.mu.Unlock()` rather than INSIDE the locked region. The plan said "Run synchronously — listener is non-blocking", which is correct, but executing the listener under the engine mutex would still deadlock if the listener (now or in the future) ever calls back into the engine. Moving the call outside the lock is a Rule 1 / Rule 2 type adjustment (correctness — prevents future deadlock), documented in the engine comment.

## Self-Check: PASSED

**Files created:**
- internal/webserver/plugin_config.go — FOUND
- internal/webserver/plugin_config_test.go — FOUND
- internal/webserver/plugin_config_stream.go — FOUND
- internal/webserver/plugin_config_stream_test.go — FOUND

**Files modified:**
- internal/webserver/server.go — FOUND (pluginSettingsProvider/Subscribers fields + 2 routes + 2 setters)
- internal/webserver/assets_test.go — FOUND (TestAssets_VendoredAddons added)
- internal/daemon/api.go — FOUND (provider + listener wired at 2 call sites)
- internal/daemon/engine.go — FOUND (pluginSettingsListener slot + setter)
- web/terminal.html — FOUND (banner div)
- web/assets/terminal.js — FOUND (fetch + applyPluginConfig + EventSource)
- web/assets/terminal.css — FOUND (banner styles)

**Commits:**
- 6ff28c3 — FOUND
- 3e02803 — FOUND
- bcd50ba — FOUND
- 659d0a0 — FOUND
- 2558186 — FOUND
- e899f60 — FOUND

All artifacts confirmed present and committed.
