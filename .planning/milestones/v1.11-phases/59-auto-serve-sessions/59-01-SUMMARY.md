---
phase: 59-auto-serve-sessions
plan: "01"
subsystem: daemon, webserver, frontend
tags: [auto-serve, web-server, session-management, tailscale]
dependency_graph:
  requires: []
  provides: [SERVE-01, SERVE-02]
  affects: [internal/daemon/api.go, internal/daemon/process.go, internal/daemon/types.go, internal/webserver/server.go, frontend/src/App.tsx]
tech_stack:
  added: []
  patterns: [daemon auto-start, enrichment pattern for API responses, TDD red-green]
key_files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/daemon/types.go
    - internal/daemon/api.go
    - internal/daemon/process.go
    - internal/daemon/api_test.go
    - frontend/src/App.tsx
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/components/__tests__/DaemonManagerPanel.test.tsx
decisions:
  - "Enrichment in handleListSessions (not engine.go) keeps WebEnabled out of SessionEngine which has no web server reference"
  - "AutoStartWebServer is idempotent: returns nil when webServer already set, enabling safe repeat calls"
  - "Export IsSessionEnabled (not pass function arg to engine) keeps webserver package API clean"
  - "Add webEnabled to SessionInfo TypeScript type in hand-maintained App.d.ts stub"
metrics:
  duration_seconds: 208
  completed_date: "2026-04-09"
  tasks_completed: 2
  files_changed: 8
requirements: [SERVE-01, SERVE-02]
---

# Phase 59 Plan 01: Daemon Auto-Serve Sessions Summary

**One-liner:** Daemon auto-starts web server on Tailscale connect (SERVE-01) and auto-enables web serving for every new session (SERVE-02), with frontend state seeding for both new and restored sessions.

## What Was Built

### Task 1: Daemon-side auto-start and auto-enable with tests

**internal/webserver/server.go**
- Renamed `isSessionEnabled` → `IsSessionEnabled` (exported) to allow daemon API to query per-session state

**internal/daemon/types.go**
- Added `WebEnabled bool \`json:"webEnabled"\`` to `SessionInfo`

**internal/daemon/api.go**
- Added `AutoStartWebServer(ip string, port int, fqdn string) error` method — idempotent, mirrors `handleWebServerStart` without HTTP context
- `handleListSessions` now enriches `SessionInfo.WebEnabled` from the running web server via `ws.IsSessionEnabled(id)` (enrichment in API layer, not engine)
- `handleCreateSession` auto-enables web serving for new sessions when `a.webServer != nil`

**internal/daemon/process.go**
- After `api.Start(socketPath)`, calls `webserver.CheckHealth()` with a 5s timeout; if Tailscale is connected with certs and IP, calls `api.AutoStartWebServer(h.IP, 7443, h.Domain)` (SERVE-01)
- Added import for `github.com/scottkw/agenthub/internal/webserver`

**internal/daemon/api_test.go**
- Added `TestAutoStartWebServer_AlreadyRunning` — verifies idempotent no-op
- Added `TestCreateSession_AutoWebEnable` — verifies new sessions get `WebEnabled=true` when web server is injected
- Added `TestCreateSession_NoAutoEnable` — verifies new sessions get `WebEnabled=false` when no web server

### Task 2: Frontend webEnabled state seeding

**frontend/src/wailsjs/go/main/App.d.ts**
- Added `webEnabled: boolean` to `SessionInfo` interface

**frontend/src/App.tsx**
- `init()`: after restoring sessions, seeds `webEnabled` map and `sessionURLs` from `s.webEnabled` field for web-enabled sessions (inside `if (running)` guard)
- `createTab`: after `CreateSession` succeeds, if `webServerRunning`, sets `webEnabled[sessionId] = true` and fetches/sets `sessionURLs[sessionId]`
- `createTab` dependency array updated to include `webServerRunning`

**frontend/src/components/__tests__/DaemonManagerPanel.test.tsx**
- Added `webEnabled: boolean` to local `SessionInfo` interface and all mock objects (auto-fix: required field added to type)

## Verification Results

- `go build ./...` — exits 0
- `go test ./internal/daemon/...` — 3 new tests pass, all existing tests pass
- `go test ./internal/webserver/...` — all tests pass (IsSessionEnabled export backward compatible)
- `go test ./...` — all 8 packages pass
- `pnpm tsc --noEmit` — exits 0
- `pnpm test` — 271 tests pass (14 test files)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing functionality] DaemonManagerPanel test mock missing webEnabled field**
- **Found during:** Task 2 (TypeScript type check)
- **Issue:** After adding `webEnabled: boolean` (required) to `SessionInfo` interface, the test file had a local `SessionInfo` interface and mock objects without the new field, causing 6 TypeScript errors
- **Fix:** Added `webEnabled: boolean` to the local interface and `webEnabled: false` to all 3 mock objects in DaemonManagerPanel.test.tsx
- **Files modified:** `frontend/src/components/__tests__/DaemonManagerPanel.test.tsx`
- **Commit:** 817ebbd

## Known Stubs

None — all features are fully wired. The `webEnabled` field flows from the daemon's web server state through the API, JSON serialization, TypeScript types, and frontend state.

## Self-Check: PASSED

Files exist:
- internal/webserver/server.go — FOUND
- internal/daemon/types.go — FOUND
- internal/daemon/api.go — FOUND
- internal/daemon/process.go — FOUND
- internal/daemon/api_test.go — FOUND
- frontend/src/App.tsx — FOUND
- frontend/src/wailsjs/go/main/App.d.ts — FOUND

Commits exist:
- d093677 — feat(59-01): daemon auto-start web server and auto-enable sessions
- 817ebbd — feat(59-01): frontend webEnabled state seeding for new and restored sessions
