---
status: deferred
deferred_to: v3.3
deferred_on: 2026-05-12
deferred_reason: "Remaining UAT items (Scenario 1 chafa iterm2 inline render desktop+web, Scenario 2 two-client mid-stream image join) require chafa output piped to a raw shell on a Tailscale-served terminal session. AgentHub v3.2 ships agent sessions only; shell session type deferred to v3.3+ (see v3.2-RELEASE-BLOCKERS.md). v3.2 signs off on automated coverage."
phase: 96
phase_name: image-addon-csp-audit
score: 4/4 automated; 2 of 4 human UATs resolved 2026-05-11; 2 remaining items deferred to v3.3 with shell-session feature
created: 2026-05-07
reverified: 2026-05-11
requirements: [IMG-01, IMG-02, IMG-03, IMG-04]
plans: [96-01, 96-02, 96-03, 96-04, 96-05, 96-06]
re_verification:
  previous_status: none
  previous_score: n/a
  gaps_closed: []
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Scenario 1 — chafa --format=iterm2 image renders inline on desktop AND web"
    expected: "Both clients paint identical inline images; no CSP / WebAssembly / 'wasm-unsafe-eval' console errors"
    why_human: "Visual fidelity at the renderer/canvas layer cannot be asserted by source-scan or unit tests; chromedp e2e proves zero CSP violations but does not prove the canvas paints pixels"
    deferred: "Tailscale/browser batch — needs both desktop and web-served clients side-by-side for fidelity comparison"
  - test: "Scenario 2 — Two-client mid-stream image join (IMG-04 visual)"
    expected: "Second client joining a session with a previously-rendered image gets the image via scrollback replay; identical colors and dimensions to first client; no CSP / WASM errors"
    why_human: "Byte-fidelity unit test proves the relay tier is byte-clean; only a real second-client renderer can confirm the rendered output matches"
    deferred: "Tailscale batch — needs two simultaneously-attached clients"
human_verification_resolved:
  - test: "Scenario 3 — Settings → Image toggle next-session-only affordance"
    resolution: "PASS 2026-05-11 — Italic caption 'Applies to new sessions you create.' confirmed visible in Settings → Plugins under Image row. With Image ON, Session A rendered a sixel red strip via /tmp/image-test.sh. Toggle Image OFF: Session A's rendered sixel remained visible (not retroactively removed); new Session B emitted the same sixel sequence but rendered NO image (addon not loaded). Toggle Image ON: Session A unchanged (no new render); new Session C rendered the sixel correctly. All three sub-expectations confirmed."
    result: pass
    resolved_on: "2026-05-11"
  - test: "Scenario 4 — 50 MB sixel fixture FIFO eviction at 16 MB cap (IMG-02)"
    resolution: "PASS 2026-05-11 — Drove /tmp/sixel-flood.sh emitting 600 sixel images at ~57 KB rendered each (~34 MB total, 2x the cap). All four checks confirmed by tester: (1) tab did NOT crash / freeze / show Page Unresponsive; (2) AgentHub memory stabilized in Activity Monitor (no unbounded growth); (3) scrolling up showed oldest images as gray/empty placeholders (FIFO evicted); (4) newest images at bottom rendered fully in their cycling R/G/B/Y color. storageLimit=16 (the default override per Phase 96) is enforced live."
    result: pass
    resolved_on: "2026-05-11"
side_observations:
  - phase: 96
    observed_on: "2026-05-11"
    observation: "iTerm2 inline-image protocol (OSC 1337 ; File = ... : base64 BEL) did NOT render in a fixture tab even with Image plugin ON; the same fixture's sixel sequence in the same tab rendered correctly. Possibilities: (a) ImageAddon construction does not enable iipSupport even though @xterm/addon-image docs say it defaults to true (TerminalPanel.tsx:235 passes only storageLimit + enableSizeReports); (b) my fixture's OSC 1337 syntax is incompatible with the addon's parser; (c) iIP is intentionally disabled in v3.2 for security/scope and only sixel ships. Sixel is the documented Phase 96 path (the verification tables specifically test 'sixel storage cap') so this is not a blocker for IMG-01/IMG-02/IMG-03 but is worth confirming the intent. Engineering follow-up: confirm whether iTerm2 IIP rendering is supposed to work in v3.2; if yes, audit the addon construction options."
---

# Phase 96: Image Addon + CSP Audit — Verification Report

**Phase Goal:** Inline sixel + iTerm2 IIP rendering ships with the heaviest addon and the only one that might require CSP amendment — gated by a mandatory pre-phase audit of `addon-image.js` source for `URL.createObjectURL` / `new Worker(` / `blob:` usage, with a dedicated multi-client byte-fidelity replay regression test and a tab-OOM guard via a 16 MB storage cap.

**Verified:** 2026-05-07
**Status:** human_needed (4/4 automated dimensions green; 4 visual UAT scenarios await human sign-off)
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (Success Criteria)

| # | Truth (ROADMAP SC) | Status | Evidence |
|---|--------------------|--------|----------|
| SC-1 | Pre-phase audit committed to RESEARCH.md; CSP amended only if required (matching v3.1 D-09 rigor); chromedp CSP zero-violation suite green on Chromium | VERIFIED | `96-RESEARCH.md` §"Mandatory Pre-Phase CSP Audit" (lines 83+) documents 0 Workers / 0 eval / 6 WASM bootstraps; `csp_mw.go:107-113` adds `'wasm-unsafe-eval'`; `csp_mw.go:23,36-57` carries Amendment 2 documentation block (5 v3.1 D-09-rigor elements); chromedp e2e `internal/webserver/browser_csp_image_e2e_test.go` (//go:build e2e) PASSES locally in 4.5s with zero CSP violations |
| SC-2 | Inline image rendering desktop+web; default ON; toggle marked "applies to new sessions you create"; sixel/IIP renders inline on both | VERIFIED (code paths) + HUMAN_NEEDED (visual sign-off) | Desktop: `TerminalPanel.tsx:5,107,193-218,294-298` constructs `ImageAddon` in MOUNT useEffect with `enableSizeReports: false` and `pluginConfig?.imageConfig?.storageLimit ?? 16`. Web: `web/assets/terminal.js:240-258` mirrors construction; `web/terminal.html:50` loads vendored UMD; `web/embed.go:11` embeds the asset. Default ON: `plugin_settings.go:108` defaults `Image: true` (Phase 92). Caption: `PluginsSection.tsx:135-137` renders italic `'Applies to new sessions you create.'`. Visual sixel render on desktop AND web is HUMAN-UAT Scenarios 1+3 |
| SC-3 | Per-terminal sixel/IIP storage hard-capped at 16 MB by default (override of upstream 100 MB); user-adjustable via Advanced disclosure (Phase 99 owns UI; Phase 96 ships pass-through plumbing); FIFO eviction regression at the cap | VERIFIED (plumbing) + HUMAN_NEEDED (live FIFO eviction observation) | `plugin_settings.go:60-63,108` defines `ImageConfig{StorageLimit int}` defaulting to 16. PATCH route + handler at `api.go:78,634-650` validates `[1, 1000]`. `engine.go:537-554` sub-key writer. `client.go:177-185` DaemonClient wrapper. `app.go:572-589` `(*App).SetImageConfig`. TS bindings `App.d.ts:143` + `App.js:86`. Constructor pass-through: `TerminalPanel.tsx:205-211` and `web/assets/terminal.js:250-253`. Advanced disclosure UI defers to Phase 99/PUI-03 (per ROADMAP). 50 MB FIFO eviction live observation is HUMAN-UAT Scenario 4; storageLimit pass-through structurally guarantees the addon's internal LRU is exercised |
| SC-4 | Multi-client byte-fidelity: second client joining mid-stream receives correctly-rendered image during scrollback replay; relay audit confirms no line-based buffering or escape filtering corrupts sixel bytes | VERIFIED (relay byte stream) + HUMAN_NEEDED (visual byte fidelity) | `internal/relay/image_byte_fidelity_test.go` `TestImage_ByteFidelity_MultiClient` PASSES — fans synthetic sixel `\x1bPq...!10A!10@-\x1b\\` to two subscribers, asserts byte-for-byte equality on fan-out AND `ScrollbackSnapshot()`. RESEARCH §"Architectural Responsibility Map" + direct reads of `internal/relay/scrollback.go` (raw 256 KiB byte buffer) + `internal/relay/hub.go` (32 KiB chunked pass-through; no line buffering, no escape parsing) confirm structural guarantee. Visual second-client rendering is HUMAN-UAT Scenario 2 |

**Score:** 4/4 truths verified at code level. SC-2/SC-3/SC-4 carry human_needed for visual/behavioral sign-off per HUMAN-UAT runbook.

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/webserver/csp_mw.go` | `'wasm-unsafe-eval'` in script-src; Amendment 2 doc block | VERIFIED | Line 113: `b.WriteString("script-src 'self' 'wasm-unsafe-eval'; ")`. Lines 36-57 contain v3.1 D-09-rigor 5-element documentation (policy spec, rationale, RESEARCH citation, narrow-vs-broad contrast, browser support floors). |
| `internal/daemon/plugin_settings.go` | `ImageConfig{StorageLimit int}`; default 16 | VERIFIED | Lines 60-63 type def; line 80 nested field on PluginSettings; line 108 `defaultPluginSettings()` returns `ImageConfig: ImageConfig{StorageLimit: 16}`. |
| `internal/daemon/engine.go` | `(*SessionEngine).SetImageConfig` sub-key writer | VERIFIED | Line 554. Concurrency contract verbatim mirror of SetWebLinksConfig (lock → mutate → save → capture listener → unlock → invoke). |
| `internal/daemon/api.go` | PATCH `/settings/image-config` with `[1, 1000]` range gate | VERIFIED | Line 78 route registration; line 634 handler with MaxBytesReader(8 KiB) + DisallowUnknownFields + range gate. |
| `internal/daemon/client.go` | `(*DaemonClient).SetImageConfig` wrapper | VERIFIED | Line 185. Routes through `c.doJSON(http.MethodPatch, "/settings/image-config", cfg, nil)`. |
| `app.go` | `(*App).SetImageConfig` Wails method with daemon-fanout | VERIFIED | Line 589. nil-client guard + sub-key write + re-fetch + synthesize fallback + WR-05 nil-ctx guard + EventsEmit `settings:plugins`. |
| `frontend/src/wailsjs/go/main/App.d.ts` + `App.js` | Wails TS/JS bindings | VERIFIED | App.d.ts:143 `SetImageConfig(arg1: daemon.ImageConfig): Promise<void>`; App.js:86 `Call('main.App.SetImageConfig', [cfg])`. |
| `frontend/src/wailsjs/go/models.ts` | `daemon.ImageConfig` class + nested `imageConfig` field | VERIFIED | Lines 54-58 class def; line 75 field; lines 103-108 constructor wiring (Phase 92 hand-edit pin). |
| `frontend/src/components/TerminalPanel.tsx` | ImageAddon constructed in MOUNT useEffect with `enableSizeReports: false` | VERIFIED | Line 5 import; line 107 `imageAddonRef`; lines 193-218 construction inside MOUNT useEffect (NOT hot-swap) with `enableSizeReports: false` (Pitfall #8 guard) and `storageLimit: pluginConfig?.imageConfig?.storageLimit ?? 16`; lines 294-298 dispose-on-unmount. |
| `frontend/src/components/PluginsSection.tsx` | Italic caption "Applies to new sessions you create." under Image row | VERIFIED | Lines 135-137 renderRow with 4th caption arg, identical character-for-character to Phase 93 unicode11 caption (line 130). |
| `web/vendor/xterm/addons/addon-image.js` | UMD bundle byte-identical to node_modules copy | VERIFIED | 79,399 bytes; Plan 96-06 confirms `cmp` byte-identity with `frontend/node_modules/@xterm/addon-image/lib/addon-image.js`. |
| `web/embed.go` | `//go:embed vendor/xterm/addons/addon-image.js` | VERIFIED | Line 11 (dedicated directive line). |
| `web/terminal.html` | `<script>` tag for vendored addon-image.js | VERIFIED | Line 50, after addon-web-links.js (49) and before terminal.js (64). UMD load order correct. |
| `web/assets/terminal.js` | Web-side ImageAddon construction with same pass-through | VERIFIED | Lines 126-129 pluginConfig defaults `imageConfig: { storageLimit: 16 }`; lines 240-258 next-session-only construction with `storageLimit` pass-through and `enableSizeReports: false`; line 858 reset-seed parity. |
| `web/vendor/xterm/VERSION` | 8 entries including `@xterm/addon-image@0.9.0` | VERIFIED | 8 lines, addon-image entry present. |
| `internal/webserver/vendor_drift_test.go` | Min-count guard bumped to 8 | VERIFIED | Line 34: `if len(pnpmVersions) < 8`. |
| `internal/webserver/browser_csp_image_e2e_test.go` | chromedp e2e with `//go:build e2e` | VERIFIED | File exists (5,293 bytes); `//go:build e2e` line 1; `TestBrowserCSP_TerminalImage_NoViolations` PASSES locally in 4.5s. |
| `internal/relay/image_byte_fidelity_test.go` | Multi-client byte-fidelity test | VERIFIED | `TestImage_ByteFidelity_MultiClient` PASSES — synthetic sixel byte stream fanned to 2 subscribers + ScrollbackSnapshot byte-equality. |
| `.planning/phases/96-image-addon-csp-audit/96-HUMAN-UAT.md` | 4 manual UAT scenarios | VERIFIED | 4 scenarios (chafa visual, multi-client mid-stream, Settings toggle, 50 MB sixel storage cap) with Setup / Procedure / Pass / Fail / Sign-off. |

---

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `App.tsx` PluginSettings event listener | `daemon.ImageConfig` | EventsOn `settings:plugins` payload | WIRED | App.plugin-event.test.tsx 15/15 pass; `app.go:589-...` EventsEmit fires full PluginSettings post-write. |
| `TerminalPanel.tsx` ImageAddon construction | `pluginConfig.imageConfig.storageLimit` | Constructor pass-through (mount useEffect) | WIRED | grep `pluginConfig?.imageConfig?.storageLimit ?? 16` count == 1; test 4 in TerminalPanel.test.tsx asserts. |
| Frontend `App.SetImageConfig` (TS) | `(*App).SetImageConfig` (Go) | Wails Call binding | WIRED | App.d.ts:143 + App.js:86 + app.go:589. |
| `(*App).SetImageConfig` | `(*DaemonClient).SetImageConfig` | HTTP PATCH | WIRED | app.go:593 calls `a.client.SetImageConfig(cfg)`. |
| `(*DaemonClient).SetImageConfig` | PATCH `/settings/image-config` | `c.doJSON` | WIRED | client.go:185 + api.go:78. |
| `handleSetImageConfig` | `engine.SetImageConfig` | Direct method call | WIRED | api.go:649 calls `a.engine.SetImageConfig(req)`. |
| `engine.SetImageConfig` | Disk persistence | `e.saveSettingsToDisk()` under lock | WIRED | engine.go:554 + sub-key writer concurrency contract; TestSetImageConfigPreservesSiblings verifies reload. |
| `web/terminal.html` script load | `ImageAddon` global | UMD `e.ImageAddon=t()` | WIRED | terminal.html:50 loads vendor; terminal.js:251 consumes `new ImageAddon.ImageAddon(...)`. |
| `web/embed.go` | Served at `/assets/xterm/addons/addon-image.js` | go:embed → fs.FS | WIRED | embed.go:11 directive; chromedp e2e exercises the served path with zero CSP violations. |
| Relay PTY → ScrollbackSnapshot | Second-client subscriber | Hub fan-out (32 KiB chunks, no line buffering) | WIRED | TestImage_ByteFidelity_MultiClient PASSES; structural guarantee confirmed by RESEARCH architectural map. |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|---|---|---|---|---|
| `TerminalPanel.tsx` ImageAddon | `pluginConfig.imageConfig.storageLimit` | App.tsx state populated by `GetPluginSettings()` Wails call → daemon → disk | YES (default 16 from `defaultPluginSettings()`; user-overridable via PATCH) | FLOWING |
| `web/assets/terminal.js` ImageAddon | `pluginConfig.imageConfig.storageLimit` | SSE/WS payload from daemon GetPluginSettings; default `{ storageLimit: 16 }` seed if event hasn't fired yet | YES | FLOWING |
| `relay.ScrollbackSnapshot()` | `MsgOutput`-framed bytes from PTY | `relay/hub.go` 32 KiB chunked pass-through with NO line buffering / NO escape parsing | YES (verbatim PTY bytes) | FLOWING |
| `engine.SetImageConfig` listener | `e.pluginSettingsListener` (snapshot under lock) | Captured at Lock(); released before invoke | YES (single-fire per write per TestSetImageConfigPreservesSiblings) | FLOWING |

No HOLLOW or DISCONNECTED artifacts found. The IMG-02 default of 16 MB flows from `defaultPluginSettings()` → `GetPluginSettings()` → frontend `pluginConfig` → ImageAddon constructor with no static-empty fallback breaking the chain.

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Daemon ImageConfig defaults + sub-key writer + handler valid + handler rejected | `go test ./internal/daemon/ -run "TestSetImageConfig\|TestHandleSetImageConfig\|TestDefaultPluginSettings" -count=1` | ok in 0.04s | PASS |
| Relay byte-fidelity multi-client | `go test ./internal/relay/ -run TestImage_ByteFidelity_MultiClient -count=1 -v` | PASS in 0.008s | PASS |
| CSP middleware tests (token-aware unsafe-eval guard + wasm-unsafe-eval present) | `go test ./internal/webserver/ -run TestCSPHeaders -count=1` | ok in 0.012s | PASS |
| Vendor drift test (8-package guard, addon-image present) | `go test ./internal/webserver/ -run TestXtermVendorVersions -count=1 -v` | PASS in 0.009s | PASS |
| chromedp CSP-zero-violation e2e on Chromium with sixel injection | `go test -tags=e2e ./internal/webserver/ -run TestBrowserCSP_TerminalImage -count=1` | ok in 4.575s | PASS |
| Frontend TerminalPanel + PluginsSection source-scan tests | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx src/components/__tests__/PluginsSection.test.tsx` | 55 passed (2 files) in 755ms | PASS |
| App.plugin-event imageConfig round-trip | `pnpm exec vitest run src/__tests__/App.plugin-event.test.tsx` | 15 passed in 416ms | PASS |
| Vendored addon-image UMD asset present at expected path | `ls -la web/vendor/xterm/addons/addon-image.js` | 79,399 bytes | PASS |

All automated gates green. No spot-check failures.

---

### Requirements Coverage

| Requirement | Source Plan(s) | Description (REQUIREMENTS.md) | Status | Evidence |
|---|---|---|---|---|
| IMG-01 | 96-01, 96-04, 96-05, 96-06 | User can enable/disable inline image support in Settings (default ON); the toggle is clearly marked as "applies to new sessions you create" | SATISFIED (code) + HUMAN_NEEDED (visual) | `Image bool` defaults true (plugin_settings.go); italic caption verbatim under Image row (PluginsSection.tsx:135-137); construction in MOUNT useEffect (next-session-only) (TerminalPanel.tsx:193-218); web mirror (terminal.js:240-258); UAT Scenarios 1, 3 |
| IMG-02 | 96-01, 96-02, 96-04, 96-05, 96-06 | Per-terminal sixel/IIP storage hard-capped at 16 MB decoded RGBA by default (override of upstream 100 MB); user can adjust via Advanced disclosure | SATISFIED (plumbing; UI deferred to Phase 99/PUI-03 per ROADMAP) + HUMAN_NEEDED (FIFO eviction observation) | ImageConfig{StorageLimit:16} default; full PATCH/RPC plumbing (engine.SetImageConfig + handleSetImageConfig + DaemonClient.SetImageConfig + (*App).SetImageConfig + TS bindings); constructor pass-through on desktop + web; UAT Scenario 4 |
| IMG-03 | 96-01, 96-03, 96-06 | Web-served Tailscale clients receive same inline image rendering as desktop, with v3.1 CSP either unchanged or amended (matching v3.1 D-09 rigor) only if pre-phase audit confirms `addon-image` requires `worker-src 'self' blob:` | SATISFIED (Chromium); cross-browser deferred to Phase 99 SC-4 per VALIDATION.md | Pre-phase audit committed (RESEARCH §"Mandatory Pre-Phase CSP Audit"; finding: NOT worker-src/blob: but `'wasm-unsafe-eval'`); Amendment 2 documentation block in csp_mw.go matches v3.1 D-09 rigor (5 elements); chromedp e2e PASSES with zero CSP violations on Chromium; web vendor lockstep complete |
| IMG-04 | 96-01, 96-06 | Second client joining session mid-stream receives correctly-rendered images during scrollback replay (multi-client byte-fidelity preserved through `internal/relay/`) | SATISFIED (byte stream) + HUMAN_NEEDED (visual byte fidelity) | TestImage_ByteFidelity_MultiClient PASSES; relay tier byte-clean by architecture (verified RESEARCH map + scrollback.go + hub.go review); UAT Scenario 2 |

**Orphaned requirements:** None. All 4 IMG-* requirements are accounted for across 6 plans.

**Cross-phase deferrals (per VALIDATION.md):**
- IMG-02 user-facing Advanced `<details>` disclosure UI → Phase 99 / PUI-03
- IMG-03 cross-browser Safari + Firefox + iPad CSP zero-violation → Phase 99 SC-4

These deferrals are explicit in ROADMAP / VALIDATION.md and are NOT Phase 96 gaps.

---

### Anti-Patterns Found

Anti-pattern scan across modified files (csp_mw.go, plugin_settings.go, engine.go, api.go, client.go, app.go, models.ts, TerminalPanel.tsx, PluginsSection.tsx, terminal.js, embed.go, terminal.html, vendor_drift_test.go, browser_csp_image_e2e_test.go, image_byte_fidelity_test.go):

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| n/a | n/a | No TODO/FIXME/XXX/HACK/PLACEHOLDER markers found | INFO | Clean |
| n/a | n/a | No `return null` / `return {}` / `return []` empty-return stubs in production paths | INFO | Clean |
| n/a | n/a | No `console.log`-only function bodies | INFO | Clean |
| n/a | n/a | No `=> {}` empty handlers; the only try/catch silent-swallow is `imageAddonRef.current.dispose() catch { /* ignore */ }` which is intentional cleanup-fault-tolerance, paralleling Phase 95 web-links pattern | INFO | Acceptable; documented in plan |
| `TerminalPanel.tsx:205` | `?? 16` default | Defensive default | INFO | Required for SC-3 default-when-unset behavior; matches `defaultPluginSettings()` source of truth |

**Pre-existing items NOT introduced by Phase 96** (logged in `deferred-items.md`):
- `frontend/src/components/FindBar/__tests__/FindBar.animation.test.tsx:15` — TS6133 unused `beforeEach` import (Phase 94 origin). Not a Phase 96 gap.
- `frontend/src/components/__tests__/Sidebar.test.tsx` — 20 React 19 / jsdom `root.unmount()` cleanup failures (pre-existing). Not a Phase 96 gap.

No blocking anti-patterns introduced by Phase 96.

---

## Human Verification Items

The following 4 scenarios are documented in `96-HUMAN-UAT.md` with full Setup / Procedure / Pass / Fail criteria. Each MUST be signed off before flipping success criteria GREEN.

### 1. Scenario 1 — chafa --format=iterm2 image rendering (IMG-01 SC-2 visual)

**Test:** Build production app (`pnpm run build && wails build -tags wailsassets`); open desktop terminal AND web-served Tailscale URL; run `chafa --format=iterm2 ~/Pictures/chart.png` in BOTH; observe DevTools console.

**Expected:** Desktop renders inline image; web renders identical image (same colors, proportions); zero CSP / WebAssembly / `'wasm-unsafe-eval'` console errors; no ghost placeholder cells on first paint.

**Why human:** Visual fidelity at the canvas/pixel layer cannot be asserted by source-scan or unit tests. The chromedp e2e proves zero CSP violations occur during WASM bootstrap; this scenario proves the canvas paints the actual image.

### 2. Scenario 2 — Two-client mid-stream image join (IMG-04 SC-4 visual)

**Test:** Render an image in a desktop session; while desktop session is still open, navigate to the SAME session URL in a separate browser (mid-stream join); observe the web terminal during scrollback replay.

**Expected:** Web terminal displays the previously-rendered image as part of scrollback replay; identical colors/dimensions to desktop; no broken/garbled rendering; no CSP/WASM errors. (Use a small image — well under the 256 KiB serialized-sixel scrollback cap.)

**Why human:** The byte-fidelity unit test proves the relay tier preserves bytes; only a second client's rendering result confirms the visual byte-fidelity round-trip end-to-end.

### 3. Scenario 3 — Settings → Image toggle next-session-only affordance (IMG-01 SC-2 behavior)

**Test:** Open session A and render an image; navigate to Settings → Plugins; toggle Image OFF and Save; confirm session A still renders images; open new session B and confirm sixel sequences appear as printable garbage (NOT rendered); toggle Image ON and Save; confirm session A unchanged; open new session C and confirm images render.

**Expected:** Italic caption `Applies to new sessions you create.` visible directly under Image toggle; toggling does NOT re-attach on already-open terminals (next-session-only invariant); only NEW sessions pick up the toggle change.

**Why human:** Live next-session-only semantics require real sessions; source-scan proves caption presence and dep-array invariants but cannot prove behavioral semantics in a running app.

### 4. Scenario 4 — 50 MB sixel fixture FIFO eviction at 16 MB cap (IMG-02 SC-3 behavior)

**Test:** Generate ~50 MB synthetic sixel stream (e.g., 13 chafa renders of a solid 1024×1024 PNG). Pipe through fresh terminal session. Observe DevTools Performance/Memory tab (web) or Activity Monitor (desktop) during and after.

**Expected:** Tab does not crash / freeze / show "Aw, Snap"; memory stabilizes (not unbounded growth); older images may show as gray placeholders (`showPlaceholder: true` upstream default after FIFO eviction at the 16 MB decoded-RGBA cap); newest images render fully.

**Why human:** Tab-OOM is a browser-side resource-pressure outcome; addon's FIFO eviction is internal to its WASM decoder (upstream-trusted code). The constructor's `storageLimit: 16` pass-through is verified in code; the live behavior under pressure must be observed.

---

## Gaps Summary

**No gaps.** All 4 phase success criteria are verified at the code level:
- SC-1 (CSP audit + amendment + chromedp e2e): VERIFIED end-to-end
- SC-2 (desktop + web inline rendering, default ON, caption): code paths VERIFIED; visual sign-off in HUMAN-UAT
- SC-3 (16 MB cap with pass-through plumbing): VERIFIED end-to-end (UI deferred to Phase 99/PUI-03 per ROADMAP)
- SC-4 (multi-client byte-fidelity through relay): VERIFIED end-to-end

All 4 IMG-* requirements are accounted for. No orphaned requirements. No blocking anti-patterns. Pre-existing test failures (Sidebar React 19 + FindBar TS6133) are documented as out-of-scope and pre-date Phase 96.

The `human_needed` status reflects the explicit HUMAN-UAT runbook authored as part of this phase (per VALIDATION.md "Manual-Only Verifications" section); the orchestrator will persist these 4 items to HUMAN-UAT.md for sign-off before flipping success criteria GREEN.

---

## Cross-Phase Deferrals (NOT gaps)

Per VALIDATION.md "Out-of-Phase (Cross-Phase Gates)" and ROADMAP:

| Item | Owning Phase | Status |
|---|---|---|
| Advanced `<details>` disclosure UI exposing `storageLimit` slider/input | Phase 99 / PUI-03 | DEFERRED — Phase 96 ships only the daemon struct + RPC + hardcoded 16 MB default; UI exposure is explicit Phase 99 scope |
| Cross-browser CSP zero-violation suite (Safari + Firefox + iPad Safari) | Phase 99 SC-4 | DEFERRED — Phase 96 ships Chromium-only chromedp e2e (matching Phase 89 precedent); release-gate cross-browser run is Phase 99 |

---

_Verified: 2026-05-07_
_Verifier: Claude (gsd-verifier, Opus 4.7 1M)_
