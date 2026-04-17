---
phase: 82-minimize-to-tray
plan: "01"
subsystem: daemon-settings
tags: [go, daemon, settings, persistence, wails, tray]
dependency_graph:
  requires: []
  provides: [startMinimized-engine, startMinimized-api, startMinimized-client, startMinimized-wails-bindings, domReady-gate]
  affects: [app.go, internal/daemon/engine.go, internal/daemon/api.go, internal/daemon/client.go, frontend/src/wailsjs/go/main/App.d.ts, frontend/src/wailsjs/go/main/App.js]
tech_stack:
  added: []
  patterns: [daemon-settings-persistence, wails-binding-stub, domReady-conditional-show]
key_files:
  created: []
  modified:
    - internal/daemon/engine.go
    - internal/daemon/engine_settings_test.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - app.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
decisions:
  - "Fixed loadSettingsFromDisk to decouple CLIPaths and startMinimized loading — the original condition skipped all fields when CLIPaths was nil, breaking any settings.json with only startMinimized"
  - "domReady fallback: when daemon unreachable (client nil or GetStartMinimized errors), startMinimized stays false and window shows normally — safe default per CONTEXT.md"
  - "Wails binding stubs (App.d.ts, App.js) are manually maintained in this project; added GetStartMinimized/SetStartMinimized exports following existing patterns"
metrics:
  duration: "~15 min"
  completed: "2026-04-17T12:54:24Z"
  tasks_completed: 2
  files_modified: 7
---

# Phase 82 Plan 01: Backend persistence and API for start-minimized preference

## One-liner

Full backend vertical slice for start-minimized: SessionEngine persistence + daemon HTTP API + typed client + Wails bindings + conditional domReady window gate.

## What Was Built

### Task 1: SessionEngine startMinimized persistence (commit b0aaf19)

Extended `SessionEngine` and `daemonSettings` to store and persist a `startMinimized bool`:

- Added `startMinimized bool` field to `SessionEngine` struct
- Added `StartMinimized bool` to `daemonSettings` with `json:"startMinimized,omitempty"` tag
- Fixed `loadSettingsFromDisk` to decouple loading: old code skipped the entire settings body when `CLIPaths` was nil — now each field is loaded independently so `{"startMinimized":true}` (no cliPaths) loads correctly
- Updated `saveSettingsToDisk` to include `StartMinimized: e.startMinimized`
- Added `GetStartMinimized() bool` and `SetStartMinimized(val bool)` engine methods with proper RLock/Lock patterns
- Added `TestStartMinimizedPersistence` (set/verify/reload/verify false) and `TestStartMinimizedWithoutCLIPaths` (cliPaths-absent JSON loads correctly)

### Task 2: API routes, client methods, Wails bindings, domReady gate (commit 4b931ee)

- **api.go**: Added `GET /settings/start-minimized` and `PATCH /settings/start-minimized` routes; handlers use `a.engine.GetStartMinimized()` / `a.engine.SetStartMinimized()`
- **client.go**: Added `GetStartMinimized() (bool, error)` and `SetStartMinimized(val bool) error` typed client methods via `doJSON`
- **app.go**: Added `GetStartMinimized() bool` and `SetStartMinimized(val bool) error` Wails bindings; modified `domReady` to gate `runtime.WindowShow` and `setDockVisible` on the persisted preference — daemon-unreachable path falls back to showing window
- **App.d.ts / App.js**: Added `GetStartMinimized` and `SetStartMinimized` exports to Wails binding stubs

## Verification Results

```
=== RUN   TestSettingsPersistence       PASS
=== RUN   TestStartMinimizedPersistence PASS
=== RUN   TestStartMinimizedWithoutCLIPaths PASS
go build ./...  EXIT 0
grep count GetStartMinimized|SetStartMinimized in app.go: 7 (2 func defs + 2 calls in domReady + 3 in bindings)
GET/PATCH /settings/start-minimized routes registered in api.go
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed existing loadSettingsFromDisk logic**
- **Found during:** Task 1
- **Issue:** The original `loadSettingsFromDisk` used `json.Unmarshal(data, &s) == nil && s.CLIPaths != nil` — this means any `settings.json` that contained `startMinimized` without `cliPaths` would have the entire body silently skipped. The plan explicitly identified and required this fix.
- **Fix:** Restructured to unmarshal first, return on error, then apply each field independently with separate nil check for CLIPaths
- **Files modified:** `internal/daemon/engine.go`
- **Commit:** b0aaf19

**2. [Rule 2 - Pattern] Wails bindings are manual stubs, not auto-generated**
- **Found during:** Task 2 verification
- **Issue:** `wails generate module` in this Wails v2 project does not auto-generate `App.d.ts`/`App.js` — these are manually maintained stubs with a misleading "AUTO-GENERATED" comment. Running `wails generate module` produced no binding output.
- **Fix:** Appended `GetStartMinimized` and `SetStartMinimized` exports to both stub files following the existing hand-written pattern
- **Files modified:** `frontend/src/wailsjs/go/main/App.d.ts`, `frontend/src/wailsjs/go/main/App.js`
- **Commit:** 4b931ee

## Known Stubs

None — all new methods wire directly to engine state or daemon API. No placeholder data.

## Threat Flags

None — no new network endpoints or trust boundary changes beyond what was planned. The PATCH /settings/start-minimized endpoint is bound to the Unix socket (local only, 0600 ownership) per T-82-03 accepted disposition.

## Self-Check: PASSED

- `internal/daemon/engine.go` — FOUND, contains `GetStartMinimized` and `SetStartMinimized`
- `internal/daemon/engine_settings_test.go` — FOUND, contains `TestStartMinimizedPersistence` and `TestStartMinimizedWithoutCLIPaths`
- `internal/daemon/api.go` — FOUND, contains `handleGetStartMinimized` and `handleSetStartMinimized`
- `internal/daemon/client.go` — FOUND, contains `GetStartMinimized` and `SetStartMinimized`
- `app.go` — FOUND, contains `GetStartMinimized`, `SetStartMinimized`, and `domReady` gate
- `frontend/src/wailsjs/go/main/App.d.ts` — FOUND, contains exported functions
- `frontend/src/wailsjs/go/main/App.js` — FOUND, contains exported functions
- Commit b0aaf19 — FOUND
- Commit 4b931ee — FOUND
