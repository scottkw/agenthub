---
phase: 95-web-links-addon-security-hardening
plan: 05
subsystem: daemon-rpc
tags: [phase-95, web-links, sub-key-rpc, settings-persistence, live-toggle, wave-3]

# Dependency graph
requires:
  - phase: 92-plugin-settings-foundation
    provides: PluginSettings struct + Wails RPC + settings:plugins event + hand-edit bindings pattern
  - phase: 93-vendoring-discipline-web-parity-for-already-shipping-addons
    provides: pluginSettingsListener (PLUG-04 SSE push hook reused for live toggle)
  - phase: 94-search-addon-find-bar-desktop-web
    provides: Plan 94-07 SetSearchConfig sub-key writer precedent (engine + api + client + Wails RPC + EventsEmit pattern verbatim)
  - phase-95-plan: 01
    provides: WebLinksConfig struct + defaults + Wave 0 RED test scaffolds awaiting flip
provides:
  - "(*SessionEngine).SetWebLinksConfig sub-key writer (mirror of Plan 94-07 SetSearchConfig)"
  - "PATCH /settings/web-links-config route + handleSetWebLinksConfig (8 KiB body cap + DisallowUnknownFields)"
  - "DaemonClient.SetWebLinksConfig (HTTP PATCH wrapper, used by app.go)"
  - "(*App).SetWebLinksConfig Wails RPC: client write + readback + EventsEmit('settings:plugins', full)"
  - "Hand-edited App.d.ts/App.js Wails bindings exposing SetWebLinksConfig"
  - "2 daemon Go tests (sub-key preserves siblings + defaults-merge migration) flipped GREEN"
  - "1 frontend test (App.plugin-event.test.tsx webLinksConfig nested constructor + JSON round-trip) flipped GREEN"
affects:
  - 95-06-PLAN (web parity Playwright e2e + terminal.js applyPluginConfig — sub-key write path now exists; 95-06 wires the SSE-driven web hot-swap arm)
  - 99-PUI-03 (Settings advanced disclosure for WebLinksConfig sub-fields — sub-key write path is callable from any Wails-bound TS code)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sub-key Wails RPC: client.SetXxx + engine.SetXxx + handler with 8 KiB body cap + DisallowUnknownFields + EventsEmit('settings:plugins', full) — mirrors Phase 94 SetSearchConfig verbatim"
    - "Hand-edited App.d.ts/App.js bindings (Phase 92 STATE.md decision): wails generate module is not part of this build pipeline; bindings are appended in the same shape as the existing SetSearchConfig entry"

key-files:
  created: []
  modified:
    - "internal/daemon/engine.go"
    - "internal/daemon/api.go"
    - "internal/daemon/client.go"
    - "internal/daemon/web_links_config_test.go"
    - "app.go"
    - "frontend/src/wailsjs/go/main/App.d.ts"
    - "frontend/src/wailsjs/go/main/App.js"
    - "frontend/src/__tests__/App.plugin-event.test.tsx"

key-decisions:
  - "Routed app.go SetWebLinksConfig through a.client.SetWebLinksConfig (HTTP PATCH to daemon) rather than the plan's literal a.engine.SetWebLinksConfig direct call. The Phase 92 architecture has app.go as a thin Wails-runtime layer over the in-process daemon HTTP API; the existing SetSearchConfig precedent at app.go:505-523 calls a.client.SetSearchConfig, not a.engine. Following the actual precedent preserves the Phase 92 boundary; the plan's literal text would have bypassed the daemon HTTP layer and broken parity with how Wails-bound RPCs everywhere else in app.go talk to the engine."
  - "Hand-edited App.js binding uses Call('main.App.SetWebLinksConfig', [cfg]) shape (matches existing SetSearchConfig stub) rather than the plan's literal window['go']['main']['App']['SetWebLinksConfig'](arg1) form. The existing Wails bindings file uses Call(...) consistently across all 60+ exports; introducing a single window['go']['main']['App'][...] indirection would break the runtime/runtime.js Call helper indirection that abstracts over Wails v2 / v3 differences."
  - "Added DaemonClient.SetWebLinksConfig (internal/daemon/client.go) — not in the plan's files_modified list but architecturally required: app.go calls a.client.SetXxx (not a.engine.SetXxx directly), and the existing SetSearchConfig client method confirms this pattern. Without the client method, app.go would not compile."

requirements-completed: [LNK-05, LNK-06]

# Metrics
duration: 18min
completed: 2026-05-06
---

# Phase 95 Plan 05: Sub-Key RPC + Persistence + Live-Toggle Plumbing Summary

**Implemented `engine.SetWebLinksConfig` + `PATCH /settings/web-links-config` + `(*App).SetWebLinksConfig` mirroring Phase 94 Plan 07's `SetSearchConfig` sub-key path verbatim. Hand-edited Wails bindings expose the new RPC. Three Wave 0 RED scaffolds flip GREEN: 2 daemon Go tests (sibling preservation + defaults-merge migration) and 1 frontend test (`webLinksConfig` nested constructor + JSON round-trip).**

## Performance

- **Duration:** ~18 min
- **Completed:** 2026-05-06
- **Tasks:** 2 / 2
- **Files modified:** 8 (4 Go, 4 frontend/bindings)
- **Lines added:** ~360 (engine 26, api 24, client 10, daemon test 154, app 36, App.d.ts 6, App.js 3, App.plugin-event.test.tsx ~60)

## Accomplishments

- **`(*SessionEngine).SetWebLinksConfig`** added to `internal/daemon/engine.go`: mutates ONLY `pluginSettings.WebLinksConfig` under `e.mu.Lock()`, calls `saveSettingsToDisk` while held, captures the listener and invokes it post-`Unlock`. Verbatim Phase 94-07 SetSearchConfig shape.
- **`PATCH /settings/web-links-config`** + **`handleSetWebLinksConfig`** registered in `internal/daemon/api.go`: 8 KiB body cap (`http.MaxBytesReader`) + `DisallowUnknownFields` defense-in-depth identical to the search-config route.
- **`DaemonClient.SetWebLinksConfig`** added to `internal/daemon/client.go`: `c.doJSON(http.MethodPatch, "/settings/web-links-config", cfg, nil)` — mirror of `SetSearchConfig`. Required by `app.go` since the Phase 92 architecture routes Wails RPCs through the daemon HTTP API.
- **`(*App).SetWebLinksConfig`** wired in `app.go`: calls `a.client.SetWebLinksConfig(cfg)`, re-fetches full PluginSettings via `a.client.GetPluginSettings()`, falls back to a synthesized payload on readback failure, then `runtime.EventsEmit(a.ctx, "settings:plugins", full)`. App.tsx's existing `EventsOn('settings:plugins')` subscription replaces `pluginConfig` wholesale; the WebLinksConfig sub-key shows up via the nested struct.
- **Hand-edited Wails bindings** (Phase 92 decision — `wails generate module` is not part of this build pipeline): `App.d.ts` declares `SetWebLinksConfig(arg1: daemon.WebLinksConfig): Promise<void>`; `App.js` exports the runtime stub via the project's standard `Call('main.App.SetWebLinksConfig', [cfg])` indirection.
- **3 Wave 0 RED scaffolds → GREEN:**
  - `TestSetWebLinksConfigPreservesSiblings`: writes a baseline PluginSettings with every non-web-links field at non-default values; calls `SetWebLinksConfig` with a new config; asserts the listener fired exactly once, the sub-key updated, all 9 sibling fields survived, and the change persisted to disk through a fresh-engine reload.
  - `TestPluginSettingsMigration_WebLinksConfig`: writes a Phase 94-shape `settings.json` (no `webLinksConfig` key) to a temp config dir; calls `loadSettingsFromDisk`; asserts `WebLinksConfig` populated to platform/all-confirm-on defaults; spot-checks the marshal round-trip preserves the merged state.
  - `App.plugin-event.test.tsx` "PluginSettings shape includes webLinksConfig nested object": constructs a `daemon.PluginSettings` with the nested `webLinksConfig` shape, asserts it's a `daemon.WebLinksConfig` instance, and asserts JSON round-trip preserves all 4 sub-fields (modifier + 3 confirm flags).

## Task Commits

Each task was committed atomically on `worktree-agent-a52f74e6e978d37d5`:

1. **Task 1: daemon `SetWebLinksConfig` + PATCH route + 2 Go tests** — `9e2dc33` (feat)
2. **Task 2: `(*App).SetWebLinksConfig` + Wails bindings + flip frontend test GREEN** — `2b70c45` (feat)

## Files Modified

### Daemon (Go, 4 files)

- `internal/daemon/engine.go` — `+26` lines: `SetWebLinksConfig` method appended directly below `SetSearchConfig`; preserves the mutex / listener pattern.
- `internal/daemon/api.go` — `+24` lines: route registration + `handleSetWebLinksConfig` handler (mirrors `handleSetSearchConfig`).
- `internal/daemon/client.go` — `+10` lines: `DaemonClient.SetWebLinksConfig` HTTP wrapper.
- `internal/daemon/web_links_config_test.go` — `2 t.Skip stubs replaced` with ~154 lines of real assertions across 2 tests.

### Frontend / Bindings (4 files)

- `app.go` — `+36` lines: `(*App).SetWebLinksConfig` Wails RPC.
- `frontend/src/wailsjs/go/main/App.d.ts` — `+6` lines: `SetWebLinksConfig` declaration.
- `frontend/src/wailsjs/go/main/App.js` — `+3` lines: `SetWebLinksConfig` runtime stub.
- `frontend/src/__tests__/App.plugin-event.test.tsx` — `expect.fail RED scaffold replaced` with ~60 lines of real assertions (constructor instance check + JSON round-trip).

## LNK-05 / LNK-06 Status

- **LNK-05 (live toggle desktop):** ✅ The existing Phase 92 PluginsSection `webLinks` boolean toggle drives App.tsx state via the existing `settings:plugins` event; TerminalPanel hot-swap (Plan 95-04) reattaches the addon. The new sub-key write path is dormant in v3.2 — the Settings advanced disclosure for sub-fields (modifier, confirm flags) ships in Phase 99 / PUI-03. The sub-key path is **callable today** by any Wails-bound TS code (e.g., a future hotkey or context-menu action).
- **LNK-06 (toggle persistence):** ✅ The boolean `webLinks` field has persisted via the full SetPluginSettings path since Phase 92. The new sub-key path adds disk persistence for `webLinksConfig.{modifier,confirmOSC8,confirmIDN,confirmTyposquat}` without requiring a follow-up plan when Phase 99 surfaces the UI.

The web-served live-toggle parity (Playwright e2e + `terminal.js applyPluginConfig` arm) is owned by Plan **95-06**, which can now exercise the new PATCH route end-to-end.

## Wails `wails generate module` Outcome

**Not run — hand-edit used per Phase 92 STATE.md decision.** The Phase 92 / 94 precedent established that `wails generate module` is not part of the standard build pipeline (some Wails versions only regenerate during full `wails build`); the project's `App.d.ts` / `App.js` are deliberately maintained as hand-edited files with stable formatting (the `Call('main.App.X', [arg])` indirection pattern in App.js is project convention). The new `SetWebLinksConfig` entries match the formatting + indirection style of the existing `SetSearchConfig` entries directly above them.

## Decisions Made

1. **`a.client.SetWebLinksConfig` (not `a.engine.SetWebLinksConfig`)** in `app.go`. The plan's literal `<action>` text wrote `a.engine.SetWebLinksConfig(cfg)` directly, but the actual Phase 92 architecture has `app.go` as a thin Wails-runtime layer over the in-process daemon **HTTP** API. The existing `SetSearchConfig` precedent at `app.go:505-523` calls `a.client.SetSearchConfig(cfg)`, not `a.engine`. Following the actual precedent verbatim preserves the Phase 92 boundary and avoids bypassing the daemon HTTP layer. This is a Rule 1 deviation (bug — plan text contradicts itself by claiming "mirror SetSearchConfig verbatim" while writing a literal that doesn't match SetSearchConfig).

2. **`Call('main.App.X', [arg])` shape (not `window['go']['main']['App']['X'](arg)`)** in App.js. The plan's literal `<action>` text used the `window['go']...` indirection, but the actual `App.js` runtime stub file uses a `Call` helper imported from `runtime.js` for all 60+ existing exports. Introducing a single `window['go']` form would have created an inconsistency and bypassed the helper that abstracts Wails v2 / v3 differences. The new `SetWebLinksConfig` stub matches the existing `SetSearchConfig` stub line shape verbatim.

3. **Added `DaemonClient.SetWebLinksConfig`** (`internal/daemon/client.go`) — not in the plan's `files_modified` list. Architecturally required: app.go calls `a.client.SetXxx`, not `a.engine.SetXxx` directly. Without this method, `go build ./...` would have failed. The existing `DaemonClient.SetSearchConfig` confirms the pattern.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] Route through `a.client.SetWebLinksConfig`, not `a.engine.SetWebLinksConfig`**
- **Found during:** Task 2 (writing `(*App).SetWebLinksConfig`).
- **Issue:** Plan literal `a.engine.SetWebLinksConfig(cfg)` would bypass the daemon HTTP API and break parity with the actual Phase 94-07 SetSearchConfig precedent (`app.go:505-523` calls `a.client.SetSearchConfig`, not `a.engine`). The plan's text claims "mirror Phase 94 Plan 07 SetSearchConfig verbatim" but its literal example contradicts that.
- **Fix:** Followed the actual `SetSearchConfig` precedent: `a.client.SetWebLinksConfig(cfg)` + `a.client.GetPluginSettings()` readback + `EventsEmit`. Added the corresponding `DaemonClient.SetWebLinksConfig` HTTP wrapper to make app.go compile.
- **Files modified:** `app.go`, `internal/daemon/client.go`
- **Verification:** `go build ./...` exits 0; manual cross-reference against `app.go:505-523` SetSearchConfig confirms identical shape.
- **Committed in:** `2b70c45` (Task 2).

**2. [Rule 1 — Bug] App.js stub uses `Call(...)` indirection**
- **Found during:** Task 2 (hand-editing `App.js`).
- **Issue:** Plan literal `window['go']['main']['App']['SetWebLinksConfig'](arg1)` does not match the existing `App.js` convention — every other binding (60+ entries) uses the imported `Call('main.App.X', [arg])` helper from `runtime.js`. The existing `SetSearchConfig` stub uses `Call`.
- **Fix:** Used `export const SetWebLinksConfig = (cfg) => Call('main.App.SetWebLinksConfig', [cfg])` to match `SetSearchConfig`. The plan's literal acceptance criterion `grep -E "window\['go'\]\['main'\]\['App'\]\['SetWebLinksConfig'\]" frontend/src/wailsjs/go/main/App.js` therefore does NOT pass, but the architectural intent (a runtime stub that delegates to the Wails RPC channel for `main.App.SetWebLinksConfig`) is preserved correctly via the project's standard helper.
- **Files modified:** `frontend/src/wailsjs/go/main/App.js`
- **Verification:** Test `App.plugin-event.test.tsx` 13/13 PASS; the binding is reachable at runtime (the same `Call` helper is exercised by the SetSearchConfig stub already in production).
- **Committed in:** `2b70c45` (Task 2).

**3. [Rule 3 — Blocking] Added `DaemonClient.SetWebLinksConfig` (not in plan's files_modified)**
- **Found during:** Task 2 (compiling app.go).
- **Issue:** `app.go SetWebLinksConfig` calls `a.client.SetWebLinksConfig(cfg)` per the actual SetSearchConfig precedent, but `internal/daemon/client.go` had no such method.
- **Fix:** Appended a 10-line `DaemonClient.SetWebLinksConfig` mirroring `DaemonClient.SetSearchConfig` (HTTP PATCH to `/settings/web-links-config`).
- **Files modified:** `internal/daemon/client.go`
- **Verification:** `go build ./...` exits 0 (was failing without this fix).
- **Committed in:** `9e2dc33` (Task 1, batched with the daemon-side changes since the client is a daemon-package file).

---

**Total deviations:** 3 auto-fixed (2 Rule 1 plan-literal-vs-precedent, 1 Rule 3 blocking).

**Impact on plan:** All three deviations are necessary corrections of the plan literal to match the actual Phase 94-07 SetSearchConfig precedent the plan explicitly says to "mirror verbatim." The architectural intent (sub-key write path that round-trips through the daemon HTTP layer + the standard EventsEmit broadcast) is preserved exactly. The plan's `must_haves.truths` list ("preserving every other PluginSettings field," "PATCH /settings/web-links-config registered," "(*App).SetWebLinksConfig Wails RPC exists") all hold true.

## Issues Encountered

- **Pre-existing failures out of scope** (logged to phase deferred list, not fixed):
  - `frontend/src/components/__tests__/Sidebar.test.tsx` — multiple Sidebar test failures pre-date this plan (last touched in commit `dd25dfb`, Phase 70-01); not in `files_modified`. Out of scope per plan boundary.
  - `frontend/src/components/FindBar/__tests__/FindBar.animation.test.tsx(15,47): error TS6133: 'beforeEach' is declared but its value is never read` — already documented in `.planning/phases/95-web-links-addon-security-hardening/deferred-items.md` from Plan 95-01.
  - Plan 95-04 RED scaffold `TerminalPanel.web-links.test.tsx` still failing as expected (waiting for 95-04 implementation).
  - Plan 95-02 RED scaffolds `urlSafety.test.ts` + `openLink.test.ts` still failing as expected.
  - Plan 95-03 RED scaffold `LinkConfirmPopover.test.tsx` still failing as expected.

- **`pnpm test --run` does not work in this project** — package.json `scripts.test` is `vitest run`, so `pnpm test --run` produces "Unknown option: 'run'" since the runner already has `run`. Used `pnpm exec vitest run` directly (matches CI invocation per Phase 94 SUMMARY).

## User Setup Required

None — no external service configuration; no schema-version bump (the `WebLinksConfig` key already lived under v3.2 plugins block since Plan 95-01).

## Next Phase Readiness

- **Plan 95-06 (web parity):** unblocked. The PATCH route is live; the SSE broadcast on `engine.SetWebLinksConfig` (via `pluginSettingsListener`) means `/api/plugin-config/stream` web subscribers will receive a frame on every web-links-config change. 95-06 wires the Playwright e2e + `terminal.js applyPluginConfig` arm.
- **Phase 99 / PUI-03:** the sub-key write path is callable today. `SetWebLinksConfig` from any TS code (e.g., a `<details>` advanced-disclosure panel in Settings) will work; no further daemon/app/binding changes required for the UI to land.
- **Plan 95-04 (TerminalPanel hot-swap):** can run in parallel with this plan (file sets are disjoint); the App.plugin-event.test.tsx GREEN flip in this plan does not depend on 95-04's TerminalPanel changes.

## Self-Check: PASSED

Verified post-Write that all claims hold:

| Claim | Check | Result |
|-------|-------|--------|
| `engine.SetWebLinksConfig` exists | `grep -c "func (e \*SessionEngine) SetWebLinksConfig" internal/daemon/engine.go` | == 1 |
| Sub-key mutation only | `grep -q "e.pluginSettings.WebLinksConfig = cfg" internal/daemon/engine.go` | FOUND |
| `saveSettingsToDisk` calls >= 3 | `grep -c "saveSettingsToDisk()" internal/daemon/engine.go` | 6 (multiple existing + new) |
| PATCH route registered | `grep -q "PATCH /settings/web-links-config" internal/daemon/api.go` | FOUND |
| Handler exists | `grep -q "func (a \*API) handleSetWebLinksConfig" internal/daemon/api.go` | FOUND |
| `DaemonClient.SetWebLinksConfig` exists | `grep -q "func (c \*DaemonClient) SetWebLinksConfig" internal/daemon/client.go` | FOUND |
| 2 Go tests present | `grep -c "^func Test" internal/daemon/web_links_config_test.go` | == 2 |
| No `t.Skip` remaining | `grep -c "t.Skip" internal/daemon/web_links_config_test.go` | == 0 |
| Sub-key Go test passes | `go test ./internal/daemon/ -run TestSetWebLinksConfigPreservesSiblings` | PASS |
| Migration Go test passes | `go test ./internal/daemon/ -run TestPluginSettingsMigration_WebLinksConfig` | PASS |
| Full daemon sweep passes | `go test ./internal/daemon/... -count=1` | ok |
| `go vet` clean | `go vet ./internal/daemon/...` | clean |
| `gofmt` clean | `gofmt -l internal/daemon/...` | empty |
| `(*App).SetWebLinksConfig` exists | `grep -q "func (a \*App) SetWebLinksConfig" app.go` | FOUND |
| Routes via client (not engine) | `grep -q "a.client.SetWebLinksConfig(cfg)" app.go` | FOUND |
| EventsEmit count >= 2 | `grep -c "runtime.EventsEmit(a.ctx, \"settings:plugins\"" app.go` | == 3 |
| App.d.ts declaration | `grep -q "export function SetWebLinksConfig(arg1: daemon.WebLinksConfig)" frontend/src/wailsjs/go/main/App.d.ts` | FOUND |
| App.js stub | `grep -q "SetWebLinksConfig" frontend/src/wailsjs/go/main/App.js` | FOUND |
| `expect.fail` removed | `grep -c "expect.fail" frontend/src/__tests__/App.plugin-event.test.tsx` | == 0 |
| Frontend prop-drill test passes | `pnpm exec vitest run src/__tests__/App.plugin-event.test.tsx` | 13/13 PASS |
| Go full build clean | `go build ./...` | exits 0 |
| Commit hashes recorded | `git log --oneline` | `9e2dc33`, `2b70c45` |
| No accidental deletions | `git diff --diff-filter=D --name-only HEAD~2 HEAD` | empty |

---
*Phase: 95-web-links-addon-security-hardening*
*Completed: 2026-05-06*
