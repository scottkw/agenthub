---
phase: 11-new-session-modal
plan: 01
subsystem: api
tags: [go, wails, pty, typescript]

# Dependency graph
requires:
  - phase: 10-per-tab-font-size
    provides: stable session creation flow used as baseline
provides:
  - OpenDirectoryDialog Go bound method with os.UserHomeDir fallback
  - WorkDir field on pty.CreateRequest
  - cmd.Dir assignment in native.go PTY process start
  - Updated CreateSession(cli, name, workDir) 3-arg signature
  - TypeScript binding stubs for OpenDirectoryDialog and updated CreateSession
affects:
  - 11-new-session-modal plan 02 (NewSessionModal frontend component)
  - 11-new-session-modal plan 03 (integration wiring)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wails runtime.OpenDirectoryDialog with OpenDialogOptions for native OS folder picker"
    - "WorkDir flows: CreateSession(workDir) -> CreateRequest.WorkDir -> cmd.Dir"

key-files:
  created: []
  modified:
    - app.go
    - internal/pty/backend.go
    - internal/pty/native.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/App.tsx
    - frontend/src/components/NewSessionModal.tsx

key-decisions:
  - "cmd.Dir assigned before cmd.Env (before cmd.Start) — critical for Windows ConPTY which reads Dir during Start"
  - "Empty workDir leaves PTY process in parent working directory — acceptable Go os/exec default"
  - "App.tsx call site passes workDir='' placeholder — modal in plan 11-02 will supply the real value"

patterns-established:
  - "WorkDir propagation: API arg -> struct field -> cmd.Dir"

requirements-completed: [SESS-03, SESS-04]

# Metrics
duration: 3min
completed: 2026-03-19
---

# Phase 11 Plan 01: Go Backend Plumbing for Folder Dialog and WorkDir Summary

**Go PTY backend extended with native folder dialog (OpenDirectoryDialog) and WorkDir propagation through CreateRequest to cmd.Dir — TypeScript binding stubs updated to match.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-19T21:44:16Z
- **Completed:** 2026-03-19T21:47:36Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added `WorkDir string` to `pty.CreateRequest` and assigned `cmd.Dir = req.WorkDir` in native.go before `cmd.Start()`, enabling per-session working directory for all PTY processes
- Added `OpenDirectoryDialog(defaultDir string)` bound method to app.go with `os.UserHomeDir()` fallback when defaultDir is empty
- Updated `CreateSession` from 2-arg to 3-arg signature `(cli, name, workDir string)` passing WorkDir through to CreateRequest
- Updated TypeScript stubs in App.d.ts and App.js to match new Go signatures; fixed pre-existing `OpenDirectoryDialog` import error in NewSessionModal.tsx

## Task Commits

1. **Task 1: Add WorkDir to CreateRequest and assign cmd.Dir in native.go** - `2a61049` (feat)
2. **Task 2: Add OpenDirectoryDialog and update CreateSession to 3-arg signature** - `dd80c1d` (feat)

## Files Created/Modified

- `internal/pty/backend.go` - Added `WorkDir string` field to CreateRequest struct
- `internal/pty/native.go` - Added `cmd.Dir = req.WorkDir` before cmd.Env assignment
- `app.go` - Added `OpenDirectoryDialog` method; updated `CreateSession` to 3-arg signature with `WorkDir: workDir` in CreateRequest literal
- `frontend/src/wailsjs/go/main/App.d.ts` - Updated `CreateSession` signature; added `OpenDirectoryDialog` export declaration
- `frontend/src/wailsjs/go/main/App.js` - Updated `CreateSession` binding; added `OpenDirectoryDialog` binding
- `frontend/src/App.tsx` - Updated `CreateSession` call site to pass `workDir=''` placeholder
- `frontend/src/components/NewSessionModal.tsx` - Removed unused `React` import (auto-fix)

## Decisions Made

- `cmd.Dir` assigned immediately after `p.CommandContext` and before `cmd.Env` — the plan explicitly notes this is critical for Windows ConPTY which reads `Dir` during `Start()`
- Empty `workDir` string leaves the PTY process in the parent working directory — correct Go os/exec default behavior, no special handling needed
- Existing `App.tsx` call site updated with `workDir=''` placeholder — the real value will be supplied by the NewSessionModal built in plan 11-02

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated App.tsx CreateSession call site from 2 to 3 args**
- **Found during:** Task 2 (TS stub update)
- **Issue:** After updating `CreateSession` TS declaration to require 3 args, the existing `App.tsx` call site `CreateSession(cliName, defaultName)` became a type error
- **Fix:** Updated call site to `CreateSession(cliName, defaultName, '')` — empty string placeholder until modal provides real workDir
- **Files modified:** `frontend/src/App.tsx`
- **Verification:** `tsc --noEmit` passes cleanly
- **Committed in:** `dd80c1d` (Task 2 commit)

**2. [Rule 1 - Bug] Removed unused React import from NewSessionModal.tsx**
- **Found during:** Task 2 (TypeScript verification)
- **Issue:** Pre-existing `TS6133: 'React' is declared but its value is never read` in NewSessionModal.tsx (from plan 11-02 commit) was blocking clean `tsc --noEmit`
- **Fix:** Changed `import React, { useState } from 'react'` to `import { useState } from 'react'`
- **Files modified:** `frontend/src/components/NewSessionModal.tsx`
- **Verification:** `tsc --noEmit` exits 0
- **Committed in:** `dd80c1d` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 - Bug)
**Impact on plan:** Both fixes necessary for TypeScript type correctness. No scope creep.

## Issues Encountered

- Plan 11-02 had already been committed before plan 11-01 ran (NewSessionModal existed but its `OpenDirectoryDialog` import had no backing Go method). Plan 11-01 execution retroactively fixed the missing binding, resolving the pre-existing TS error.

## Next Phase Readiness

- Go backend plumbing complete — `OpenDirectoryDialog` and `WorkDir` propagation ready for use
- Plan 11-02 (NewSessionModal component) is already committed and now fully wired to working bindings
- Plan 11-03 (integration) can proceed with `go build ./...` and `tsc --noEmit` both passing cleanly

---
*Phase: 11-new-session-modal*
*Completed: 2026-03-19*
