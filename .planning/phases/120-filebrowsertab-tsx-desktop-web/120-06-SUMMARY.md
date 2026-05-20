---
phase: 120-filebrowsertab-tsx-desktop-web
plan: 06
subsystem: ui
tags: [filebrowser, web-share, wails, mode-detection, parity, cross-surface, vite, playwright, embed-fs]

# Dependency graph
requires:
  - phase: 120-04
    provides: FileBrowserTab + FileBrowser/* leaf components (already accept baseURL + capToken)
  - phase: 120-05
    provides: Playwright fixture with seeded test tree + viewer cap + appUrl()/viewerAppUrl()
provides:
  - "frontend/src/lib/webMode.ts — pure-function single source of truth for web-vs-desktop mode"
  - "App.tsx web-mode awareness: skips Wails RPC suite + sources fbBaseURL/capToken from URL"
  - "Playwright fixture embedded React bundle (build-tag gated): /app/ serves the SPA under tests"
  - "Two DOM-level e2e scenarios (13 + 14) closing the v3.5 web-share parity deferral"
  - "Vite relative-base build so the same bundle serves under both Wails / and webserver /app/"
affects:
  - "v3.5 follow-on phases (remote-on-desktop browse from GUI — still deferred but now isolated)"
  - "Any future SPA features that need to honor web vs desktop (consult lib/webMode)"

# Tech tracking
tech-stack:
  added: []  # no new runtime dependencies
  patterns:
    - "Mode-detection module: pure function + Location override for unit tests"
    - "Build-tag-gated embed.FS in test fixtures (dist-copy strategy to avoid //go:embed escape)"
    - "Source-inspection vitest via App.tsx?raw (mirrors existing App.test.tsx pattern)"

key-files:
  created:
    - frontend/src/lib/webMode.ts
    - frontend/src/lib/__tests__/webMode.test.ts
    - frontend/src/components/__tests__/App.fileBrowserMode.test.tsx
    - cmd/playwright-fixture/assets_prod.go
    - cmd/playwright-fixture/assets_stub.go
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/__tests__/App.test.tsx
    - frontend/vite.config.ts
    - frontend/e2e/global-setup.ts
    - frontend/e2e/files-browser.spec.ts
    - frontend/e2e/fixture-env.ts
    - cmd/playwright-fixture/main.go
    - .gitignore

key-decisions:
  - "Pathname is the canonical web-vs-desktop signal — not navigator.userAgent or wails globals"
  - "webMode.ts has zero React/Wails imports — safe to load from any code path"
  - "Vite base: './' chosen so the same built bundle serves correctly under / and /app/"
  - "Fixture embeds cmd/playwright-fixture/dist/ (copied at setup time) because //go:embed cannot escape its package"
  - "SettingsTab/WelcomeTab skipped in web mode rather than gating each Wails call inside leaf components"
  - "Remote-on-desktop browse remains a v3.5 follow-on — out of scope for v3.4 parity closure"

patterns-established:
  - "lib/webMode: detectMode() + readWebModeParams() — pure-function single-source-of-truth for mode"
  - "Mode-gated useEffect early-returns: `if (mode === 'web') return` at the top of every Wails-bound effect"
  - "Test-fixture embed strategy: copy dist into package-local subdir, gate with -tags=wailsassets"

requirements-completed: [UI-01, UI-11, UI-13, UI-14]

# Metrics
duration: 14min
completed: 2026-05-20
---

# Phase 120 Plan 06: Web-share Parity Closure Summary

**App-level web-mode detection (lib/webMode) + URL-param-driven session/cap binding in App.tsx + DOM-level Playwright cells embedding the React bundle in the fixture — closes the v3.5 deferral flagged in 120-VERIFICATION.md Human Verification #2.**

## Performance

- **Duration:** 14 min
- **Started:** 2026-05-20T23:00:04Z
- **Completed:** 2026-05-20T23:14:20Z
- **Tasks:** 3 (RED/GREEN/REFACTOR pairs collapsed where appropriate)
- **Files modified:** 13 (5 created, 8 modified)

## Accomplishments

- **Closed Human Verification #2 parity gap.** Loading `/app/?session=…&cap=…` in a regular browser now mounts a fully-functional file-browser tab without Wails dependencies; loading with a viewer cap (no `files.read`) renders the PermissionDeniedTakeover with the verbatim copy. Both paths verified DOM-level on chromium + firefox + webkit.
- **Single source of truth for mode detection.** `frontend/src/lib/webMode.ts` exports pure-function `detectMode()` + `readWebModeParams()` — no Wails imports, no React imports, no side effects. Unit-tested with 14 vitest cases.
- **App.tsx mode-aware throughout.** Init useEffect, retryInit, daemon-manager poll, remote-sessions poll, update poller, SettingsTab mount, WelcomeTab mount, and the file-browser tab fbBaseURL + capToken selection all consult `mode = detectMode()`. Desktop behavior is bit-for-bit unchanged.
- **Playwright fixture embeds the React bundle.** `cmd/playwright-fixture/assets_prod.go` (build-tagged) + `global-setup.ts` (builds React, copies dist, builds Go with `-tags=playwrightfixture,wailsassets`) wire /app/ to serve the SPA under tests. Scenarios 13 + 14 added; total cells now 15 × 3 = 45 (up from 39).
- **Vite relative-base unblocks the parity path.** `base: './'` so index.html's asset references work under both `/` (Wails) and `/app/` (webserver) without colliding with the legacy `/assets/` mount.

## Task Commits

1. **Task 1 RED — webMode tests** — `33c86f7` (test)
2. **Task 1 GREEN — webMode module** — `d6da022` (feat)
3. **Task 2 RED — App.fileBrowserMode tests** — `3539094` (test)
4. **Task 2 GREEN — App.tsx web-mode refactor** — `f22c4c5` (feat)
5. **Task 3a — playwright fixture embed.FS wiring** — `787f8ab` (feat)
6. **Task 3b — vite relative base + global-setup dist-copy** — `a999beb` (feat)
7. **Task 3c — SettingsTab/WelcomeTab web-mode gating (Rule 1/2 fix)** — `56bfdee` (fix)
8. **Task 3d — DOM-level e2e scenarios 13 + 14** — `4f9671a` (test)

## Files Created/Modified

### Created
- `frontend/src/lib/webMode.ts` — pure-function `detectMode()` + `readWebModeParams()`
- `frontend/src/lib/__tests__/webMode.test.ts` — 14 vitest cases (pathname matrix + URL-param matrix)
- `frontend/src/components/__tests__/App.fileBrowserMode.test.tsx` — 14 source-inspection cases
- `cmd/playwright-fixture/assets_prod.go` — build-tag-gated embed of `dist/`
- `cmd/playwright-fixture/assets_stub.go` — nil-FS stub for the non-wailsassets dev build

### Modified
- `frontend/src/App.tsx` — `mode = detectMode()`, `webParams = useMemo(...)`, 5 `mode === 'web'` early-returns, mode-aware tab gate, auto-open effect, mode-gated SettingsTab + initial state
- `frontend/src/components/__tests__/App.test.tsx` — initial-state assertions updated to match new mode-aware literals
- `frontend/vite.config.ts` — `base: './'` for dual-mount compatibility
- `frontend/e2e/global-setup.ts` — vite-build + dist-copy + wailsassets build tag
- `frontend/e2e/files-browser.spec.ts` — scenarios 13 + 14, architecture-note rewrite
- `frontend/e2e/fixture-env.ts` — appUrl JSDoc rewrite (parity-gap note removed)
- `cmd/playwright-fixture/main.go` — `ws.SetStaticAppFS(staticAppFixture())` wiring
- `.gitignore` — ignore `cmd/playwright-fixture/dist/`

## Decisions Made

- **Pathname is the canonical signal for web vs desktop.** No navigator.userAgent sniffing, no Wails-global probing. `pathname.startsWith('/app/')` mirrors the server-side route layout (`internal/webserver/server.go:566` mounts `/app/` exactly).
- **`webMode.ts` has zero React/Wails imports.** Pure utility module, safe to load from any code path (including future server-render paths that may not have full window context yet).
- **Vite `base: './'`.** The webserver mounts `/assets/` for legacy xterm/webfs assets; the SPA's default absolute `/assets/...` paths collided with that mount under `/app/`. Relative base resolves to `/app/assets/...` (served by the SPA route) AND `/assets/...` under Wails desktop. No webserver route changes needed.
- **Embed `cmd/playwright-fixture/dist/`, not `frontend/dist/`.** Go's `//go:embed` cannot escape its package directory. `global-setup.ts` copies `frontend/dist/` into the fixture package at setup time, gated by mtime caching so repeated runs are cheap.
- **Skip SettingsTab + WelcomeTab in web mode at the App.tsx layer.** Both leaf components call Wails RPCs on mount; gating each inside the component would be invasive. Skipping the mount entirely is simpler and respects the principle that the web-share viewer is a single-purpose surface (file browser only).
- **Remote-on-desktop browse stays deferred.** This plan does not add a path for the desktop GUI to point FileBrowserTab at a remote session's web-share URL + cap. The inline comment at the file-browser tab gate documents this is a v3.5 follow-on. The desktop file-browser tab keeps wiring against the local relay port (the proven path).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Vite default absolute `base: '/'` collided with webserver's `/assets/` mount**
- **Found during:** Task 3 e2e verification — scenario 12 failed because `/app/` served index.html but its `/assets/index-*.css` reference 404'd against the legacy `/assets/` route.
- **Issue:** The webserver mounts `GET /assets/` to serve xterm/webfs vendor assets (long-standing). When the SPA was loaded under `/app/`, its absolute-path asset references hit that mount instead of the SPA mount, breaking the entire React render.
- **Fix:** Changed `frontend/vite.config.ts` to `base: './'` so the build emits relative asset paths (e.g. `./assets/index-XYZ.css`). Verified the same build works under both Wails (`/`) and webserver (`/app/`).
- **Files modified:** frontend/vite.config.ts
- **Verification:** scenario 12 passes; scenarios 13 + 14 mount the React tree without CSS/JS 404s.
- **Committed in:** `a999beb`

**2. [Rule 1 - Bug] SettingsTab + WelcomeTab made unconditional Wails RPC calls on mount**
- **Found during:** Task 3 e2e verification — scenario 12 surfaced `[SettingsTab] loadWebState: Error: Wails bridge not available for main.App.IsWebServerRunning` as console.error (allowlist violation) and a 404 from WelcomeTab's `/agenthub-title-logo.png`.
- **Issue:** SettingsTab is mounted unconditionally via `<div style={{ display: ... }}>` and calls IsWebServerRunning + HasCTDisclosure + GetCLIPaths on mount. WelcomeTab mounted briefly because activeId defaulted to WELCOME_TAB.id. Both Wails calls fail under a browser, polluting the page-error allowlist.
- **Fix:** Wrapped SettingsTab in `{mode !== 'web' && (...)}`; changed `initialTabs` + initial `activeId` to be mode-aware (`mode === 'web' ? [] : [WELCOME_TAB]` / `mode === 'web' ? null : WELCOME_TAB.id`). Web-mode shell now boots with empty tabs and lets the auto-open effect mount the file browser cleanly.
- **Files modified:** frontend/src/App.tsx, frontend/src/components/__tests__/App.test.tsx (assertion updates)
- **Verification:** All 45 e2e cells PASS; all 1047 vitest cases PASS.
- **Committed in:** `56bfdee`

---

**Total deviations:** 2 auto-fixed (1 blocking config issue, 1 bug surfaced by the parity wiring).
**Impact on plan:** Both fixes were necessary to deliver the truth that "Loading /app/?session=…&cap=… in a regular browser mounts the React shell without crashing on Wails RPC failures." Neither expanded scope — both addressed pre-existing latent bugs that were only reachable once the parity wiring landed.

## Issues Encountered

- The 1.15 MB minified JS bundle warning from Vite is pre-existing — not addressed in this plan (out of scope per PLAN constraint).
- `module.register()` deprecation warnings from Playwright/Node are pre-existing (Playwright 1.59.1 on Node 25.5.0).

## E2E Cell Count

| Project | Before | After | Net new |
| --- | --- | --- | --- |
| chromium | 13 | 15 | +2 |
| firefox | 13 | 15 | +2 |
| webkit | 13 | 15 | +2 |
| **Total cells** | **39** | **45** | **+6** |

All 45 cells PASS locally.

## Verification Gate Results

Per PLAN.md `<verification>` source-inspection gates:

| Gate | Required | Actual |
| --- | --- | --- |
| `grep -c "detectMode()" frontend/src/App.tsx` | ≥ 1 | 1 |
| `grep -c "window.location.origin" frontend/src/App.tsx` | ≥ 1 | 1 |
| `grep -c "wailsjs" frontend/src/lib/webMode.ts` | 0 | 0 |
| `grep -cE "file-browser-(tab|permission-denied)" frontend/e2e/files-browser.spec.ts` | ≥ 2 | 3 |
| `grep -cE "if \(mode === 'web'\) (return\|\\{)" frontend/src/App.tsx` | ≥ 2 | 5 |

All gates green.

## Architectural Notes for v3.5

- **Embed dist via package-local copy.** Future test fixtures that need the React bundle should mirror the `cmd/playwright-fixture/assets_prod.go` + `global-setup.ts` dist-copy strategy. `//go:embed` cannot escape its package, so copy the canonical `frontend/dist/` into a package-local subdir at setup time and gate with a build tag.
- **Remote-on-desktop browse is now isolated, not entangled.** The deferred v3.5 path (desktop GUI pointing FileBrowserTab at a remote session's web-share URL + cap) is a single inline comment at App.tsx's file-browser tab gate. Implementing it requires only: (1) a way for the desktop UI to pass `(baseURL, capToken)` overrides into `handleOpenFileBrowser`, and (2) a UX trigger on the RemoteSessionsPanel.
- **Mode-detection contract.** Any future SPA code path that needs to differ between desktop and web MUST consult `lib/webMode.detectMode()`. Do not inline `window.location.pathname.startsWith('/app/')` — the lookalike-boundary edge cases (e.g. `/apps`) and the unit-test override pattern matter.
- **Vite base: relative is the contract.** Do not switch to absolute base under any condition; it breaks the dual-mount property.

## Next Phase Readiness

- v3.4 web-share parity gap is closed. The /gsd:verify-work pass that flagged Human Verification #2 should now find scenarios 13 + 14 covering the React DOM path under all three browsers.
- Manual UAT items remain for the user:
  - (Human Verification #1) Wails desktop click-path verification (unchanged from prior plans — desktop behavior is bit-for-bit unchanged).
  - (Human Verification #2) Web-share viewer parity — now automatable; user should re-run `/gsd:verify-work 120`.
  - (Human Verification #3) WARNINGS triage (unchanged).

## Self-Check: PASSED

- `frontend/src/lib/webMode.ts` — exists (verified)
- `frontend/src/lib/__tests__/webMode.test.ts` — exists, 14 cases pass
- `frontend/src/components/__tests__/App.fileBrowserMode.test.tsx` — exists, 14 cases pass
- `cmd/playwright-fixture/assets_prod.go` — exists, build verified with both tag combos
- `cmd/playwright-fixture/assets_stub.go` — exists, dev-build verified
- All 8 task commits present in git log (`33c86f7`, `d6da022`, `3539094`, `f22c4c5`, `787f8ab`, `a999beb`, `56bfdee`, `4f9671a` — all reachable from HEAD)
- All 1047 vitest cases pass; tsc --noEmit exits 0; 45 playwright cells pass; Go relay/webserver/daemon/files tests pass.
- FileBrowserTab.tsx, FilesApiClient, and FileBrowser/* leaf components show zero diff vs base (verified with `git diff 58b302b -- ...`).
- STATE.md and ROADMAP.md show zero diff vs base (per worktree-mode contract — orchestrator updates those).

---

_Phase: 120-filebrowsertab-tsx-desktop-web_
_Completed: 2026-05-20_
