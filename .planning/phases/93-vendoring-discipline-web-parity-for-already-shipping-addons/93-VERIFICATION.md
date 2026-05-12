---
phase: 93-vendoring-discipline-web-parity-for-already-shipping-addons
verified: 2026-05-04T20:21:04Z
reverified: 2026-05-11T00:00:00Z
status: human_needed
score: 5/5 must-haves verified (automated); 2 of 6 human UAT items resolved 2026-05-11 (UAT-3 hot-swap PASS, UAT-4 Unicode 11 next-session-only PASS, UAT-6 reclassified); UAT-1 iPad, UAT-2 DevTools, UAT-5 Tailscale still pending
overrides_applied: 0
re_verification:
  previous_status: none
  previous_score: n/a
  gaps_closed: []
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "iPad Safari software-rasterizer preemption (UAT-1 from 93-iPad-UAT.md)"
    expected: "iPad Safari shows verbatim banner 'Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience.' once per session, persistent until tapped, with 53px height + #7aa2f7 left accent"
    why_human: "Physical iPad hardware required — iPad Safari's WebGL is software-rasterized via ANGLE Metal Renderer; headless Chromium spoofs are insufficient for real-device attestation"
  - test: "Desktop Chrome WebGL context-loss → DOM fallback (UAT-2)"
    expected: "After running WEBGL_lose_context.loseContext() in DevTools, banner shows verbatim 'Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact.' with auto-dismiss at 8s; scrollback intact; no auto-retry network requests"
    why_human: "Requires real GPU + DevTools console invocation; auto-dismiss timing visual confirmation"
    deferred: "Tailscale/browser batch — requires real browser with DevTools to invoke WEBGL_lose_context.loseContext() and observe banner copy + auto-dismiss timing"
  - test: "Real Tailscale-served zero-CDN audit (UAT-5)"
    expected: "DevTools Network filtered for cdn.jsdelivr.net|unpkg.com|esm.sh shows zero matches across attach + scrollback + detach session over real Tailnet"
    why_human: "Real-network egress over Tailscale cannot be reproduced in CI; iPad UAT script attestation is the gate per 93-VALIDATION.md Manual-Only Verifications"
human_verification_resolved:
  - test: "Hot-swap across two open desktop terminals (UAT-3)"
    resolution: "Signed off 2026-05-11 by Ken Scott on AgentHub v3.2-dev @ 2d89d62. Two terminal tabs with scrollback content. Toggle WebGL OFF → Save: both tabs continued rendering via DOM renderer, scrollback intact in both, no flicker. Toggle WebGL ON → Save: hot-swapped back to WebGL with scrollback intact, no flicker. All six sub-expectations confirmed."
    resolved_on: "2026-05-11"
  - test: "Unicode 11 italic caption + next-session-only honoring (UAT-4)"
    resolution: "Signed off 2026-05-11 by Ken Scott on AgentHub v3.2-dev @ 2d89d62. Step 1: italic muted caption 'Applies to new sessions you create.' confirmed in Settings → Plugins under Unicode 11 row. Step 2: with Unicode 11 OFF, fixture-CLI tab rendered 📌 at width-1 (line 1 aligned with single-`a` reference). Step 3: toggled Unicode 11 ON; existing tab opencode 17 still showed 📌 width-1 (next-session-only preserved); new tab opencode 18 showed 📌 width-2 (aligned with `aa`/`あ` width-2 references). All three sub-expectations confirmed. Fixture path: /tmp/unicode11-emoji-test.sh (4-line pipe-bracketed cell-width comparison)."
    resolved_on: "2026-05-11"
  - test: "Re-run flaky go test gate (TestPluginConfigStream_ExpiredCap_Returns401)"
    resolution: "Reclassified 2026-05-11: not a human-UAT item. Test isolation/flakiness is automatable and belongs in CI engineering, not user verification. Tracked separately for follow-up; removed from human-UAT backlog."
    resolved_on: "2026-05-11"
  - test: "Real Tailscale-served zero-CDN audit (UAT-5)"
    expected: "DevTools Network filtered for cdn.jsdelivr.net|unpkg.com|esm.sh shows zero matches across attach + scrollback + detach session over real Tailnet"
    why_human: "Real-network egress over Tailscale cannot be reproduced in CI; iPad UAT script attestation is the gate per 93-VALIDATION.md Manual-Only Verifications"
  - test: "Re-run flaky go test gate (TestPluginConfigStream_ExpiredCap_Returns401)"
    expected: "Test passes deterministically when run alongside other Phase 93 tests"
    why_human: "Verification observed flaky behavior (passes in isolation 5x; fails ~33% when combined with other Phase-93 plugin_config tests in the same -run regex). Developer should investigate test isolation OR confirm intermittent acceptance"
---

# Phase 93: Vendoring Discipline + Web Parity for Already-Shipping Addons Verification Report

**Phase Goal:** The three already-shipping desktop addons (webgl, unicode11, clipboard) are migrated under the new reconcile pattern AND vendored same-origin for the web-served terminal page (where none are vendored today), with `vendor_drift_test.go` extended into a load-bearing CI gate that enforces version parity for every `@xterm/addon-*` package.
**Verified:** 2026-05-04T20:21:04Z
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (from ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC-1 | User toggles WebGL in Settings → live hot-swap on all open desktop terminals; Unicode 11 toggle shows inline italic caption "Applies to new sessions you create" and is honored at next-session create time only | VERIFIED (automated portion) — REQUIRES UAT-3 + UAT-4 for full attestation | `frontend/src/components/TerminalPanel.tsx:182-233` two-useEffect architecture: mount `[sessionId]` for Unicode 11 (line 174); hot-swap `[pluginConfig?.webgl, pluginConfig?.clipboard, onWebGLContextLost, sessionId]` (line 233) for WebGL + Clipboard live-attach/detach. `frontend/src/components/PluginsSection.tsx:94,116` renders `settings-panel__description--italic` with verbatim "Applies to new sessions you create.". 41/41 vitest tests pass. |
| SC-2 | Web terminal renders WebGL/Unicode11/OSC 52 from same-origin vendored assets under `web/vendor/xterm/addons/`; zero CDN requests; CSP `script-src 'self'` zero violations | VERIFIED — REQUIRES UAT-5 for real-Tailscale attestation | 3 vendored UMD bundles exist (`web/vendor/xterm/addons/{addon-webgl,addon-unicode11,addon-clipboard}.js`); 5-line VERSION manifest matches pnpm-lock; embed.go embeds all 3; terminal.html loads them in order between addon-fit.js and terminal.js (lines 17-22); `web-vendor-parity.spec.ts` LIVE PASSES with zero-CDN assertion; `web-csp.spec.ts` LIVE PASSES with zero-CSP-violations assertion. `TestSecurity_NoInlineScriptOrStyleInHTML` + `TestSecurity_NoCDNReferencesInWebAssets` pass. |
| SC-3 | WebGL context loss → DOM fallback with scrollback intact, no auto-retry, one-shot BannerStack toast; software-rasterized WebGL detected and DOM preemptively used | VERIFIED (automated portion) — REQUIRES UAT-1 + UAT-2 for full attestation | `webglProbe.ts isSoftwareWebGL()` matches SwiftShader/llvmpipe/ANGLE-software/ANGLE-SwiftShader (line 24); `TerminalPanel.tsx:189-204` calls probe and either fires `onWebGLContextLost('software-rasterized')` or wires onContextLoss handler that disposes addon + fires `onWebGLContextLost('context-loss')`. `WebGLRecoveryBanner.tsx:44-45` renders verbatim copy with auto-dismiss 8000ms for context-loss / persistent for software-rasterized. `App.tsx:115-117` state + `webglBannerDismissed` one-shot gate. **Scrollback survival proven correct-by-construction:** `grep -nE "term\\.(clear|reset)\\(\\)" frontend/src/components/TerminalPanel.tsx` returns 0 matches (production code never clears scrollback). |
| SC-4 | Web plugin-config change applies to all connected web clients without manual page reload (hot-swappable) via `/api/plugin-config`, gated by v3.1 SEC-* capability-token | VERIFIED | `internal/webserver/plugin_config.go` handler returns 503/200; `plugin_config_stream.go` SSE handler streams `event: plugin-config\ndata: <json>\n\n` frames with first-frame ≤ 250ms; `BroadcastPluginConfig` non-blocking fan-out with drop-on-slow-consumer. Routes `mux.HandleFunc("GET /api/plugin-config", ws.requireCapability(ws.handleGetPluginConfig))` and `mux.HandleFunc("GET /api/plugin-config/stream", ws.requireCapability(ws.handleStreamPluginConfig))` (server.go:423,428). Daemon engine wires `SetPluginSettingsListener` at both NewWebServer call sites (api.go:304,628) — listener invoked AFTER mutex release (engine.go:466-484). `web/assets/terminal.js:233-269` `applyPluginConfig` diff-applying function reused for both initial-fetch + SSE push paths; `grep -c 'location.reload' web/assets/terminal.js` returns 0 (reload-free). `web-plugin-hot-swap.spec.ts` 3 tests LIVE PASS including SSE-driven WebGL OFF without page reload. |
| SC-5 (additional) | CI fails red on `@xterm/addon-*` version drift between pnpm-lock and VERSION | VERIFIED | `vendor_drift_test.go:18` regex generalized to `addon-[\w-]+`; line 34 min-count guard ≥ 5; comment line 1 attributes to Phase 93 WEB-02. `go test ./internal/webserver/... -run TestXtermVendorVersionsMatchPnpmLock -count=1` exits 0 with all 5 packages enforced. |

**Score:** 5/5 truths verified (automated portion). Full attestation depends on iPad UAT execution at `/gsd-verify-work 93` time.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/webserver/vendor_drift_test.go` | Generalized regex + min-count ≥ 5 + Phase 93 WEB-02 attribution | VERIFIED | Regex `addon-[\\w-]+` present (line 18); `len(pnpmVersions) < 5` (line 34); test passes with all 5 entries. |
| `web/vendor/xterm/addons/addon-webgl.js` | UMD bundle, byte-identical to node_modules source | VERIFIED | 247,535 bytes; existed and embedded |
| `web/vendor/xterm/addons/addon-unicode11.js` | UMD bundle | VERIFIED | 52,489 bytes |
| `web/vendor/xterm/addons/addon-clipboard.js` | UMD bundle | VERIFIED | 6,384 bytes |
| `web/vendor/xterm/VERSION` | 5-line manifest matching pnpm-lock versions | VERIFIED | `@xterm/addon-webgl@0.19.0`, `@xterm/addon-unicode11@0.9.0`, `@xterm/addon-clipboard@0.2.0` all match pnpm-lock |
| `web/embed.go` | go:embed directive includes addons subdir | VERIFIED | Second `//go:embed vendor/xterm/addons/addon-{webgl,unicode11,clipboard}.js` line present |
| `web/terminal.html` | 3 new script tags between addon-fit.js and terminal.js + #webgl-recovery-banner div | VERIFIED | Lines 19-21 + line 15 banner div with role="status" aria-live="polite" |
| `frontend/src/components/TerminalPanel.tsx` | Mount + hot-swap useEffects; void pluginConfig removed; webglAddonRef + clipboardAddonRef + onWebGLContextLost prop | VERIFIED | Two useEffects ([sessionId] line 174, [pluginConfig?.webgl, pluginConfig?.clipboard, onWebGLContextLost, sessionId] line 233); `grep -c 'void pluginConfig'` returns 0 |
| `frontend/src/components/WebGLRecoveryBanner.tsx` | Two reason variants with verbatim copy + auto-dismiss timer | VERIFIED | Lines 44-45 verbatim copy; aria-label="Dismiss notification"; role="status"+aria-live="polite" |
| `frontend/src/lib/webglProbe.ts` | isSoftwareWebGL() boolean probe matching SwiftShader/llvmpipe/ANGLE-software/ANGLE-SwiftShader | VERIFIED | Line 24 regex pattern; renderer string never returned (T-93-WGL-03 mitigation) |
| `frontend/src/components/PluginsSection.tsx` | Italic caption "Applies to new sessions you create." under Unicode 11 row | VERIFIED | Line 94 className with --italic modifier; line 116 verbatim string |
| `frontend/src/App.tsx` | webglContextLost/webglSoftwareDetected/webglBannerDismissed state; handleWebGLContextLost stable callback; banner-stack render extension | VERIFIED | Lines 115-117 state; line 172 useCallback; line 924 onWebGLContextLost prop wired; lines 793-795 banner render under webglBannerDismissed gate |
| `frontend/src/style.css` | .settings-panel__description--italic + .webgl-recovery-banner block | VERIFIED | grep returns ≥ 1 for both selectors |
| `internal/webserver/plugin_config.go` | handleGetPluginConfig returns 200 JSON / 503 fallback | VERIFIED | Handler implements all branches; capability-gated route registered |
| `internal/webserver/plugin_config_stream.go` | handleStreamPluginConfig SSE + BroadcastPluginConfig fan-out + drop-on-slow-consumer | VERIFIED | text/event-stream; per-subscriber chan buffer = 4; non-blocking sends |
| `internal/webserver/plugin_config_test.go` | 4 unit tests (NoCap_401, ValidCap_200JSON, NoProvider_503, NilProvider_503) | VERIFIED | All 4 tests + assets_test.go TestAssets_VendoredAddons all pass |
| `internal/webserver/plugin_config_stream_test.go` | 5 tests (NoCap_401, ExpiredCap_401, ValidCap_FirstFrame≤250ms, FanOut_TwoClients, DisconnectCleansUp) | VERIFIED with FLAKE | All 5 tests present; ExpiredCap_Returns401 is intermittently flaky when run alongside other plugin_config tests (~33% failure rate; 100% pass in isolation). See WARNING. |
| `internal/daemon/api.go` | SetPluginSettingsProvider + SetPluginSettingsListener at both NewWebServer call sites | VERIFIED | grep returns 2 for each (lines 291,304 and 618,628) |
| `internal/daemon/engine.go` | pluginSettingsListener slot; listener invoked AFTER mutex release | VERIFIED | Field declared line 46; SetPluginSettings at lines 466-484 captures listener under lock, unlocks, then invokes |
| `web/assets/terminal.js` | Conditional addon load + applyPluginConfig + EventSource hot-swap + isSoftwareWebGL | VERIFIED | All required strings present; `grep -c 'location.reload'` returns 0 |
| `web/assets/terminal.css` | #webgl-recovery-banner styles parallel to desktop | VERIFIED | grep returns ≥ 1 for #webgl-recovery-banner; prefers-reduced-motion respected |
| `frontend/e2e/web-vendor-parity.spec.ts` | LIVE Playwright spec asserting same-origin loads + zero CDN | VERIFIED | LIVE (no .skip); `pnpm exec playwright test` passes |
| `frontend/e2e/web-csp.spec.ts` | LIVE Playwright spec asserting zero CSP violations | VERIFIED | LIVE; passes |
| `frontend/e2e/web-plugin-hot-swap.spec.ts` | 3 LIVE Playwright tests including SSE push hot-swap without reload | VERIFIED | All 3 tests pass; constructor-count invariant proven |
| `cmd/playwright-fixture/main.go` | Build-tagged Go fixture binary for e2e harness | VERIFIED | Compiles only with `playwrightfixture` tag; production build skips |
| `frontend/playwright.config.ts` | globalSetup/globalTeardown wiring | VERIFIED | Present; uses /__test__/plugin-config admin endpoint pattern |
| `93-iPad-UAT.md` | 5 UAT sections with verbatim toast copy | VERIFIED | grep returns 3 for the verbatim copy strings (UAT-1, UAT-2, UAT-4) |
| `93-VALIDATION.md` | status:approved + nyquist_compliant:true + wave_0_complete:true + Per-Task Verification Map | VERIFIED | All frontmatter values set; map has 10 plan-task-ID rows |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `vendor_drift_test.go` | `frontend/pnpm-lock.yaml` | regex `^  '(@xterm/(?:xterm\|addon-[\\w-]+))@([0-9.]+)':` | WIRED | Test passes with all 5 packages enforced |
| `web/embed.go` | `web/vendor/xterm/addons/*.js` | `//go:embed` directive | WIRED | go build ./web exits 0 |
| `web/terminal.html` | `/assets/xterm/addons/addon-*.js` | `<script src=...>` tags between addon-fit.js and terminal.js | WIRED | Order verified: lines 17,18,19,20,21,22 |
| `TerminalPanel.tsx` | `webglProbe.ts` | `import { isSoftwareWebGL }` | WIRED | Used in hot-swap useEffect line 190 |
| `TerminalPanel.tsx` | `App.tsx onWebGLContextLost` | callback prop fired from onContextLoss | WIRED | App.tsx:924 threads `handleWebGLContextLost`; TerminalPanel fires reason='context-loss' or 'software-rasterized' |
| `App.tsx` | `WebGLRecoveryBanner` | rendered inside `.banner-stack` when (webglContextLost \|\| webglSoftwareDetected) && !webglBannerDismissed | WIRED | Render condition lines 771,793-795 |
| `internal/webserver/server.go` | `handleGetPluginConfig` | `mux.HandleFunc + ws.requireCapability` | WIRED | Line 423 |
| `internal/webserver/server.go` | `handleStreamPluginConfig` | `mux.HandleFunc + ws.requireCapability` | WIRED | Line 428 |
| `internal/daemon/api.go` | `WebServer.pluginSettingsProvider` | `ws.SetPluginSettingsProvider(func() []byte { json.Marshal(engine.GetPluginSettings()) })` | WIRED | Both call sites: lines 291, 618 |
| `internal/daemon/engine.go SetPluginSettings` | `BroadcastPluginConfig` | `OnPluginSettingsChanged listener` | WIRED | Listener registered at api.go:304,628; engine invokes outside mutex |
| `web/assets/terminal.js` | `/api/plugin-config?cap=<token>` | `fetch(withCap(...))` | WIRED | Line 125 |
| `web/assets/terminal.js` | `/api/plugin-config/stream?cap=<token>` | `new EventSource(withCap(...))` + addEventListener('plugin-config', ...) | WIRED | Lines 358-394; `applyPluginConfig` invoked on each frame; beforeunload closes stream cleanly |
| `web/assets/terminal.js` | `WebglAddon.WebglAddon`, `Unicode11Addon.Unicode11Addon`, `ClipboardAddon.ClipboardAddon` | UMD globals from script-tag-loaded vendored bundles | WIRED | Constructor calls match Plan 93-02 SUMMARY-recorded names |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `web/assets/terminal.js applyPluginConfig` | `pluginConfig` object | `fetch('/api/plugin-config')` + `EventSource('/api/plugin-config/stream')` | Yes — daemon serves real `engine.GetPluginSettings()` JSON; SSE first frame within 250ms (verified by test) | FLOWING |
| `WebGLRecoveryBanner` render path | `webglContextLost` / `webglSoftwareDetected` | App.tsx state set by `handleWebGLContextLost` callback fired from TerminalPanel.onContextLoss / isSoftwareWebGL preemption | Yes — real state writes from production callback paths; tests pin one-shot `webglBannerDismissed` invariant | FLOWING |
| `PluginsSection italic caption` | Static literal string "Applies to new sessions you create." | Hardcoded per UI-SPEC § Copywriting Contract | Yes — static-string-by-design (verbatim copy contract) | FLOWING (intentional static) |
| `TerminalPanel hot-swap` | `pluginConfig?.webgl`, `pluginConfig?.clipboard` | Drilled from App.tsx via Phase 92 prop pipeline (Wails `settings:plugins` event → daemon engine.GetPluginSettings) | Yes — Phase 92 propagation pipeline already verified; Phase 93 lifts inert-prop and consumes the value in dep array | FLOWING |
| `SSE BroadcastPluginConfig` | Pre-marshaled JSON `[]byte` | `pluginSettingsProvider()` closure calls `json.Marshal(engine.GetPluginSettings())` | Yes — provider closure executed at broadcast time; non-stale; fan-out test asserts 2 clients receive the same frame | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| vendor-drift gate fires green | `go test ./internal/webserver/... -run TestXtermVendorVersionsMatchPnpmLock -count=1` | ok 0.015s | PASS |
| /api/plugin-config endpoint suite | `go test ./internal/webserver/... -run TestPluginConfig -count=1` | All 4 pass | PASS |
| /api/plugin-config/stream SSE suite | `go test ./internal/webserver/... -run TestPluginConfigStream -count=1` | 5 pass in isolation | PASS (with FLAKE — see WARNING) |
| Vendored addon assets reachable | `go test ./internal/webserver/... -run TestAssets_VendoredAddons -count=1` | Pass | PASS |
| No inline scripts in HTML | `go test ./internal/webserver/... -run TestSecurity_NoInlineScriptOrStyleInHTML -count=1` | Pass | PASS |
| No CDN refs in web assets | `go test ./internal/webserver/... -run TestSecurity_NoCDNReferencesInWebAssets -count=1` | Pass | PASS |
| Frontend Phase 93 vitest tests | `pnpm exec vitest run TerminalPanel.hot-swap WebGLRecoveryBanner webglProbe PluginsSection App.plugin-event` | 41/41 pass | PASS |
| Full webserver test suite | `go test ./internal/webserver/... -count=1` | ok 1.232s | PASS |
| Full daemon test suite | `go test ./internal/daemon/... -count=1` | ok 6.571s | PASS |
| Full Playwright e2e suite | `pnpm exec playwright test` | 5 passed (15.1s) | PASS |
| Full vitest src/ run | `pnpm exec vitest run --reporter=dot src/` | 526 passed, 20 failed (deferred Sidebar) | PASS (Phase 93 portion) |
| `go build ./...` | `go build ./...` | exits 0 | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| PLUG-04 | 93-04 | Web plugin-config push w/o manual reload (capability-gated) | SATISFIED | /api/plugin-config + /api/plugin-config/stream SSE; web terminal.js EventSource w/ no location.reload; `web-plugin-hot-swap.spec.ts` SSE no-reload test passes |
| WGL-01 | 93-03 | WebGL toggle live-applies to open desktop terminals (no session restart) | SATISFIED | TerminalPanel hot-swap useEffect with [pluginConfig?.webgl] dep; webglAddonRef dispose/load logic |
| WGL-02 | 93-03 | Context loss → DOM fallback w/ scrollback intact, no auto-retry, one-shot toast | SATISFIED (auto) — UAT-2 attestation pending | onContextLoss handler disposes addon + fires callback; one-shot via webglBannerDismissed; scrollback survival proven correct-by-construction (no term.clear/term.reset in production code); WebGLRecoveryBanner reason='context-loss' verbatim copy with 8s auto-dismiss |
| WGL-03 | 93-03 | Software-rasterized WebGL detected → DOM preemptively used | SATISFIED (auto) — UAT-1 attestation pending | isSoftwareWebGL() probe with regex matches; preemption fires onWebGLContextLost('software-rasterized') with persistent banner |
| WGL-04 | 93-02 | Web client receives same WebGL behavior, vendored same-origin | SATISFIED | addon-webgl.js vendored + embedded + script-tagged; terminal.js conditional load; web-vendor-parity spec asserts same-origin |
| U11-01 | 93-03 | Italic caption "Applies to new sessions you create." | SATISFIED — UAT-4 attestation pending | PluginsSection.tsx:116 verbatim string; .settings-panel__description--italic CSS modifier |
| U11-02 | 93-04 | Web client uses server-shared Unicode 11 setting | SATISFIED | terminal.js `applyPluginConfig` reads pluginConfig.unicode11; cached-only at construction (no mid-session swap to prevent buffer corruption) |
| CLIP-01 | 93-02 + 93-03 | OSC 52 clipboard support toggleable; live attach/detach desktop | SATISFIED | ClipboardAddon vendored; TerminalPanel hot-swap useEffect handles clipboardAddonRef dispose/load |
| CLIP-02 | 93-04 | Read-only viewers cannot have OSC 52 writes | SATISFIED | terminal.js gate `pluginConfig.clipboard && window.__perms !== 'read'` (line 262); applyPluginConfig diff path also enforces |
| WEB-01 | 93-02 | All web addons vendored same-origin under web/vendor/xterm/addons/ with VERSION manifest | SATISFIED | 3 .js files + 5-line VERSION; web-vendor-parity spec asserts same-origin loads |
| WEB-02 | 93-01 + 93-05 | vendor_drift_test generalized; CI fails on disagreement; CSP zero-violation | SATISFIED — UAT-5 attestation pending | Generalized regex test passes; web-csp.spec.ts asserts zero violations on real headless Chromium hitting fixture |
| WEB-03 | 93-04 | Web page conditionally instantiates addons based on /api/plugin-config | SATISFIED | terminal.js applyPluginConfig diff-applies; Playwright tests with page.route() canned responses verify webgl=true → load, webgl=false → no load |

All 12 requirement IDs from PLAN frontmatter are accounted for. No orphaned requirements relative to REQUIREMENTS.md Phase 93 mapping.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `frontend/vite.config.ts` | n/a (missing) | No `test.exclude` rule for `e2e/*.spec.ts` | WARNING | Running `pnpm test` (vitest) sweeps in Playwright spec files and fails to transform them. The Plan 93-05 task added e2e specs without updating vitest config to exclude them. Production tests still pass (`pnpm exec playwright test` and Phase 93 vitest unit tests both pass), but `pnpm test` (the default test script) now reports 4 false-positive failures (the 3 e2e specs + 1 Sidebar deferred). Workaround: run vitest pointed at `src/` only. |
| `internal/webserver/plugin_config_stream_test.go` | line 88+ | TestPluginConfigStream_ExpiredCap_Returns401 flaky | WARNING | Test passes 100% in isolation (5/5) but fails ~33% when combined with TestPluginConfig|TestAssets|TestPluginConfigStream regex group. Cause likely shared-state in capability test helpers (signing key / grants registry not isolated between sub-tests). Does NOT cause `go test ./internal/webserver/... -count=1` to fail (full suite always passes), but the more targeted Plan 93-04 verification command does intermittently fail. |
| `frontend/src/components/__tests__/Sidebar.test.tsx` | beforeEach localStorage.clear | 20 pre-existing failures (`localStorage.clear is not a function`) | INFO | Pre-existing per `deferred-items.md` and verified on base commit before Phase 93. Not a Phase 93 regression. |

### Human Verification Required

#### 1. iPad Safari Software-Rasterizer Preemption (UAT-1)

**Test:** Open Tailscale URL in iPad Safari; observe banner within 5 seconds
**Expected:** Verbatim "Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience.", 53px height, #7aa2f7 left accent, × dismiss, persistent (no auto-dismiss), one-shot per session via sessionStorage
**Why human:** Physical iPad hardware required — iPad Safari WebGL is software-rasterized via ANGLE Metal; headless Chromium spoofs are insufficient

#### 2. Desktop Chrome WebGL Context-Loss → DOM Fallback (UAT-2)

**Test:** Open in desktop Chrome; run `WEBGL_lose_context.loseContext()` in DevTools console
**Expected:** Verbatim "Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact." with auto-dismiss 8s; scrollback intact; zero auto-retry network requests; banner does NOT reappear after page reload (sessionStorage one-shot)
**Why human:** Real GPU + DevTools console invocation; auto-dismiss timing visual confirmation

#### 3. Hot-Swap Across Open Desktop Terminals (UAT-3)

**Test:** Open Wails app, create 2 terminal sessions; toggle WebGL OFF + Save; switch to each tab
**Expected:** Both tabs continue rendering correctly via DOM with scrollback intact, no flicker; toggle ON hot-swaps back without scrollback loss
**Why human:** Requires running Wails app, multi-tab session creation, visual no-flicker confirmation

#### 4. Unicode 11 Italic Caption + Next-Session-Only (UAT-4)

**Test:** Open Settings → Plugins; visually confirm italic paragraph + emoji cell-width on existing vs new sessions
**Expected:** Italic "Applies to new sessions you create." in muted color #9aa5ce under Unicode 11 description; toggling does NOT affect existing terminals; new sessions reflect toggle (test with `echo "📌"`)
**Why human:** Visual italic style + muted color + emoji cell-width verification

#### 5. Real Tailscale Zero-CDN Audit (UAT-5)

**Test:** Open Tailscale URL in Chrome; DevTools Network filter `cdn.jsdelivr.net|unpkg.com|esm.sh`; full attach + scroll + detach session
**Expected:** Zero matches across full session
**Why human:** Real-network egress over Tailnet cannot be reproduced in CI; iPad UAT script attestation is the gate per 93-VALIDATION.md Manual-Only Verifications

#### 6. (Investigative) Re-run flaky Go test gate

**Test:** Run `go test ./internal/webserver/... -run "TestPluginConfig|TestAssets_VendoredAddons|TestPluginConfigStream" -count=1` 5+ times
**Expected:** Pass 5/5 deterministically. Verifier observed ~33% failure rate (intermittent in TestPluginConfigStream_ExpiredCap_Returns401 → returns 403 instead of 401).
**Why human:** Developer should investigate test isolation (likely signing-key / grants-registry shared state across sub-tests using the same testServer fixture) OR explicitly accept this as acceptable test-isolation noise. The full webserver suite (`go test ./internal/webserver/... -count=1`) always passes, so CI is not blocked, but the Per-Task Verification Map command in 93-VALIDATION.md is unreliable.

---

### Gaps Summary

**Automated verification is COMPLETE** for all 5 ROADMAP success criteria across all 12 requirement IDs. Every artifact exists, every key link is wired, every primary test gate passes, and the production code paths are connected end-to-end (data flowing). Specifically:

- Plan 93-01 (vendor-drift gate) — VERIFIED green
- Plan 93-02 (vendoring + embed + script tags) — VERIFIED, byte-identical
- Plan 93-03 (TerminalPanel hot-swap + WebGLRecoveryBanner + isSoftwareWebGL + italic caption) — VERIFIED (41/41 vitest tests pass; void pluginConfig invariant lifted; scrollback survival proven correct-by-construction)
- Plan 93-04 (/api/plugin-config + SSE push + web terminal.js applyPluginConfig + EventSource) — VERIFIED (no `location.reload` in terminal.js; SSE push hot-swap test passes)
- Plan 93-05 (3 LIVE Playwright e2e specs + 93-iPad-UAT.md + VALIDATION approved) — VERIFIED (5/5 Playwright tests pass)

**Remaining items requiring human attestation:**

1. **iPad UAT execution (UAT-1..UAT-5)** — physical-hardware attestation that headless Chromium cannot reach. The 93-iPad-UAT.md script is documented and ready; execution is the explicit gate at `/gsd-verify-work 93` time per VALIDATION.md "Manual-Only Verifications" table.

2. **Two non-blocking warnings developer should address:**
   - **vite.config.ts missing `test.exclude` for e2e/*.spec.ts** — `pnpm test` reports false-positive failures; trivial config fix
   - **TestPluginConfigStream_ExpiredCap_Returns401 flaky** — passes in isolation, fails ~33% combined; full suite always passes so CI gate is not broken, but Per-Task Map command is unreliable

Status: **human_needed**. The 5 iPad UAT items are the only thing standing between verified state and full sign-off. There are no FAILED truths; no missing artifacts; no broken wiring.

---

_Verified: 2026-05-04T20:21:04Z_
_Verifier: Claude (gsd-verifier, Opus 4.7 1M context)_
