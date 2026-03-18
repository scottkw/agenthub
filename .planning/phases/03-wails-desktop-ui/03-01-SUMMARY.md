---
phase: 03-wails-desktop-ui
plan: 01
subsystem: ui
tags: [wails, react, xterm, go, pty, relay, websocket, vite, typescript]

# Dependency graph
requires:
  - phase: 01-pty-foundation
    provides: SessionBackend interface, NativePTYBackend, DetectCLIs, SessionRegistry
  - phase: 02-session-registry-websocket-relay
    provides: HubManager, relay.Server, relay.Hub, MsgResize2 protocol

provides:
  - Wails v2 App struct with 7 bound Go methods (CreateSession, ListSessions, RenameSession, KillSession, DetectCLIs, GetRelayPort, UpdateCLIPath)
  - Wails project config (wails.json) at repo root
  - React frontend scaffold with xterm.js 6 and Vite 8
  - Relay resize path wired end-to-end (MsgResize2 WS frame → Hub.Resize → backend.Resize)
  - Hub.Resize method with resizeFn callback
  - HubManager.Create updated with resizeFn parameter

affects:
  - 03-02 (terminal UI — calls all 7 bound methods via Wails JS bindings)
  - 03-03 (system tray — uses App struct's beforeClose, startup, shutdown hooks)

# Tech tracking
tech-stack:
  added:
    - github.com/wailsapp/wails/v2 v2.11.0
    - react 19.2.4
    - react-dom 19.2.4
    - "@xterm/xterm" 6.0.0
    - "@xterm/addon-fit" 0.11.0
    - "@xterm/addon-unicode11" 0.9.0
    - "@xterm/addon-webgl" 0.19.0
    - "@xterm/addon-clipboard" 0.2.0
    - vite 8.0.0
    - vitest 4.1.0
    - typescript 5.9.3
  patterns:
    - Wails App struct with startup/shutdown/beforeClose lifecycle hooks
    - os.DirFS stub for non-Wails go test builds (avoids //go:embed path conflicts)
    - resizeFn callback injected into Hub at construction time (avoids circular dependency)
    - tabNames/cliPaths maps protected by sync.RWMutex in App struct

key-files:
  created:
    - cmd/agenthub/app.go
    - cmd/agenthub/app_test.go
    - cmd/agenthub/assets_stub.go
    - wails.json
    - frontend/package.json
    - frontend/vite.config.ts
    - frontend/tsconfig.json
    - frontend/tsconfig.node.json
    - frontend/index.html
    - frontend/src/main.tsx
    - frontend/src/App.tsx
    - frontend/src/style.css
    - .gitignore
  modified:
    - cmd/agenthub/main.go
    - internal/relay/hub.go
    - internal/relay/hub_test.go
    - internal/relay/manager.go
    - internal/relay/server.go
    - internal/relay/server_test.go
    - go.mod
    - go.sum

key-decisions:
  - "os.DirFS stub in assets_stub.go (build tag !wailsassets) — Go embed all:frontend/dist can't reach repo-root frontend/ from cmd/agenthub/ (no .. in embed paths, symlinks not followed)"
  - "resizeFn callback injected into Hub at NewHub construction time rather than passing backend directly — keeps relay package free of pty import cycle"
  - "HubManager.Create accepts resizeFn parameter — matches how App.CreateSession wires per-session resize to backend.Resize"
  - "App.ctx set to context.Background() in testApp helper — startup() not called in tests but backend.Create requires non-nil context"

patterns-established:
  - "Wails lifecycle: startup() allocates listener synchronously before goroutine Serve — prevents GetRelayPort race"
  - "beforeClose returns true + WindowHide — window hides to tray, actual quit via Plan 03 tray menu"
  - "Frontend scaffold at repo root frontend/ — standard Wails layout; wails.json assetdir ./frontend/dist"

requirements-completed: [TERM-01, TERM-02, CLI-03]

# Metrics
duration: 11min
completed: 2026-03-18
---

# Phase 3 Plan 01: Wails Desktop App Scaffold Summary

**Wails v2 App struct with 7 bound methods (session CRUD, relay port, CLI detection), relay MsgResize2 wired to backend.Resize via Hub callback, and React 19 + xterm.js 6 frontend scaffold building to dist/**

## Performance

- **Duration:** 11 min
- **Started:** 2026-03-18T14:33:49Z
- **Completed:** 2026-03-18T14:45:22Z
- **Tasks:** 2
- **Files modified:** 18

## Accomplishments

- App struct with all 7 Wails-bound methods passing 7 unit tests
- Relay resize path fully wired: MsgResize2 WebSocket frame → Hub.Resize → resizeFn → backend.Resize
- React frontend scaffold with xterm.js dependencies builds successfully via `pnpm run build`
- wails.json project config created at repo root
- .gitignore added to exclude node_modules, dist, build artifacts

## Task Commits

1. **Task 1 (RED)** - `2c8855c` (test: add failing tests for App bound methods)
2. **Task 1 (GREEN)** - `e900f3a` (feat: Wails scaffold + App struct with 7 bound methods)
3. **Task 2** - `8675df1` (feat: relay resize wiring + React frontend scaffold)
4. **Task 2 (fixup)** - `d9c3283` (chore: add .gitignore and remove tracked node_modules/dist)
5. **Task 2 (fixup)** - `bbc92f6` (chore: remove symlink cmd/agenthub/frontend)

## Files Created/Modified

- `cmd/agenthub/app.go` — App struct with 7 Wails-bound methods
- `cmd/agenthub/main.go` — Wails entrypoint replacing Phase 1 smoke-test
- `cmd/agenthub/app_test.go` — 7 unit tests for bound methods
- `cmd/agenthub/assets_stub.go` — os.DirFS stub for go test (non-Wails builds)
- `wails.json` — Wails project configuration
- `internal/relay/hub.go` — Added resizeFn field, Hub.Resize method
- `internal/relay/manager.go` — HubManager.Create accepts resizeFn
- `internal/relay/server.go` — NewServer accepts backend, MsgResize2 wired
- `internal/relay/hub_test.go` — Updated NewHub calls to pass nil resizeFn
- `internal/relay/server_test.go` — Updated Create/NewServer calls
- `frontend/package.json` — React 19, xterm.js 6, Vite 8, TypeScript
- `frontend/vite.config.ts` — Vite + React plugin + vitest config
- `frontend/tsconfig.json` — Strict TypeScript, ES2020, bundler resolution
- `frontend/tsconfig.node.json` — TypeScript config for Vite config file
- `frontend/index.html` — Vite entry HTML
- `frontend/src/main.tsx` — React root render
- `frontend/src/App.tsx` — Placeholder component
- `frontend/src/style.css` — Dark theme reset
- `.gitignore` — Excludes node_modules, dist, Go artifacts
- `go.mod` + `go.sum` — Added wails/v2 v2.11.0

## Decisions Made

- **os.DirFS stub instead of //go:embed**: Go's embed directive prohibits `..` in paths and does not follow symlinks. Since `cmd/agenthub/main.go` can't reach repo-root `frontend/dist/` via embed, we use a build-tag stub (`!wailsassets`) that provides `os.DirFS(".")`. For production, `wails build` with `-tags wailsassets` activates the real embed.
- **resizeFn callback in Hub**: Rather than passing `pty.SessionBackend` directly into `internal/relay`, we inject a `func(cols, rows int) error` closure at construction time. This keeps the relay package independent of the pty package (no import cycle).
- **testApp sets context.Background()**: The Wails `startup()` lifecycle is bypassed in tests. `backend.Create` requires a non-nil context, so `testApp()` sets `app.ctx = context.Background()` before calling bound methods.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] App.ctx nil in test helper**
- **Found during:** Task 1 (GREEN phase — first test run)
- **Issue:** `testApp()` simulated startup by opening a TCP listener but did not set `app.ctx`. `backend.Create` called `context.WithCancel(ctx)` which panicked on nil parent context.
- **Fix:** Added `app.ctx = context.Background()` to `testApp()` helper; added `context` import.
- **Files modified:** `cmd/agenthub/app_test.go`
- **Verification:** All 7 tests pass
- **Committed in:** `e900f3a` (part of Task 1 commit)

**2. [Rule 3 - Blocking] options.App field name mismatch**
- **Found during:** Task 1 (build failure)
- **Issue:** Plan specified `BeforeClose:` field but Wails uses `OnBeforeClose:`
- **Fix:** Updated `main.go` to use `OnBeforeClose: app.beforeClose`
- **Files modified:** `cmd/agenthub/main.go`
- **Verification:** `go build ./cmd/agenthub/...` succeeds
- **Committed in:** `e900f3a` (part of Task 1 commit)

**3. [Rule 3 - Blocking] //go:embed path conflict**
- **Found during:** Task 1 (build failure)
- **Issue:** `//go:embed all:frontend/dist` in `cmd/agenthub/main.go` failed — Go embed doesn't follow symlinks and prohibits `..` paths; frontend/ is at repo root, not under cmd/agenthub/
- **Fix:** Created `assets_stub.go` with build tag `!wailsassets` using `os.DirFS(".")` for test/dev builds. Main.go references package-level `assets` var satisfied by the stub.
- **Files modified:** `cmd/agenthub/assets_stub.go` (created), `cmd/agenthub/main.go` (removed embed directive)
- **Verification:** `go build ./cmd/agenthub/...` succeeds, `go test ./cmd/agenthub/...` passes
- **Committed in:** `e900f3a` (part of Task 1 commit)

**4. [Rule 3 - Blocking] node_modules accidentally committed**
- **Found during:** Task 2 commit
- **Issue:** No .gitignore existed; `git add frontend/` included all of node_modules (2000+ files)
- **Fix:** Created `.gitignore` excluding `frontend/node_modules/`, `frontend/dist/`, Go artifacts; ran `git rm -r --cached` to remove from tracking
- **Files modified:** `.gitignore` (created)
- **Committed in:** `d9c3283`

---

**Total deviations:** 4 auto-fixed (1 bug, 3 blocking)
**Impact on plan:** All fixes necessary for the code to compile and tests to pass. No scope creep.

## Issues Encountered

- **Pre-existing flaky test `TestHub_SlowClientDisconnected`**: Both normalClient and slowClient have 256-entry channel buffers. When 300 messages are written quickly without reading from normalClient, normalClient's buffer can also fill and receive CloseSlow before slowClient. This fails ~50% of runs. Verified pre-existing by running on original code before changes. Documented in `deferred-items.md`.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 03-02 (terminal UI) can now call all 7 bound methods via Wails JS bindings
- `wails dev` will serve the React app in dev mode against the Go backend
- Plan 03-03 (system tray) can use the `beforeClose`/`shutdown`/`startup` hooks already in place
- Frontend scaffold needs replacing with full tab/terminal UI (App.tsx is a placeholder)

---
*Phase: 03-wails-desktop-ui*
*Completed: 2026-03-18*
