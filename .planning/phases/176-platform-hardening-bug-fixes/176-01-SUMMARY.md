---
phase: 176-platform-hardening-bug-fixes
plan: 01
subsystem: infra
tags: [go, wails, gtk, webkit2gtk, linux, wayland, cross-platform]

# Dependency graph
requires: []
provides:
  - "darwin-guarded macOS role menus (AppMenu/EditMenu/WindowMenu) in appMenu()"
  - "Linux-only WEBKIT_DISABLE_DMABUF_RENDERER env guard in runGUI()"
affects: [176-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "goruntime alias for stdlib runtime to avoid collision with wails/v2/pkg/runtime in main.go"
    - "os.LookupEnv-gated os.Setenv to respect a pre-existing user env value (never overwrite)"

key-files:
  created: []
  modified:
    - main.go

key-decisions:
  - "Aliased stdlib runtime as goruntime rather than renaming the existing Wails runtime import, per plan D-08"
  - "No Linux Edit submenu added (D-09) — deferred unless copy/paste verification proves broken"

patterns-established:
  - "First runtime.GOOS platform guard in main.go — future OS-conditional logic in this file should reuse the goruntime alias"

requirements-completed: [BUG-05]

coverage:
  - id: D1
    description: "main.go compiles/vets clean with the aliased stdlib runtime import; exactly three goruntime.GOOS == \"darwin\" guards wrap AppMenu/EditMenu/WindowMenu; File/Help submenus remain unconditional"
    requirement: "BUG-05"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... && grep -c 'goruntime.GOOS == \"darwin\"' main.go == 3"
        status: pass
    human_judgment: false
  - id: D2
    description: "WEBKIT_DISABLE_DMABUF_RENDERER set to 1 only on Linux and only when not already user-set, before wails.Run()"
    requirement: "BUG-05"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... && grep -c 'WEBKIT_DISABLE_DMABUF_RENDERER' main.go == 2"
        status: pass
    human_judgment: false
  - id: D3
    description: "Live Linux/Wayland GUI launch (no segfault, menu bar works, no DMABUF freeze) — not verifiable on this macOS dev box"
    human_judgment: true
    rationale: "Requires a real Linux/Wayland compositor to observe the GTK backend menu construction and WebKit2GTK renderer behavior; macOS dev box cannot cross-run this. Per D-11, the reporter's from-source verification (Pop!_OS 24.04/COSMIC/Wayland) is accepted as sufficient to ship; a manual TESTING.md M-NN item is added in plan 176-04 for opportunistic future confirmation."

# Metrics
duration: 3min
completed: 2026-07-09
status: complete
---

# Phase 176 Plan 01: Linux GUI segfault + DMABUF freeze fix Summary

**Darwin-guarded the three Wails role menus and added a Linux-only WEBKIT_DISABLE_DMABUF_RENDERER env guard in main.go, fixing the Linux desktop GUI dead-on-arrival bug (#124/BUG-05).**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-09T13:07:10Z
- **Completed:** 2026-07-09T13:09:45Z
- **Tasks:** 2 completed
- **Files modified:** 1 (main.go)

## Accomplishments
- Wrapped `menu.AppMenu()`, `menu.EditMenu()`, `menu.WindowMenu()` in `appMenu()` each in their own `goruntime.GOOS == "darwin"` guard, preventing Wails' GTK backend from dereferencing their nil SubMenu on Linux (the segfault-on-launch root cause).
- Left the custom `File` and `Help` submenu construction unconditional so they still render on every platform.
- Added a Linux-only, user-override-respecting guard in `runGUI()` that sets `WEBKIT_DISABLE_DMABUF_RENDERER=1` before `wails.Run(...)` when the var is not already present — addressing the WebKit2GTK DMABUF GPU-renderer freeze under Wayland.
- Aliased the stdlib `runtime` package as `goruntime` to avoid collision with the already-imported `wailsapp/wails/v2/pkg/runtime`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Alias stdlib runtime and darwin-guard the three role menus in appMenu()** - `2bdf7edc` (fix)
2. **Task 2: Set WEBKIT_DISABLE_DMABUF_RENDERER on Linux in runGUI() with a user-override guard** - `9625f760` (fix)

_No TDD tasks in this plan — GUI-only fix verified from source (build/vet/grep), per plan's `<verification>` section._

## Files Created/Modified
- `main.go` - Added `goruntime "runtime"` import alias; darwin-guarded `menu.AppMenu()`/`menu.EditMenu()`/`menu.WindowMenu()` in `appMenu()`; added Linux-only `WEBKIT_DISABLE_DMABUF_RENDERER` env guard at the top of `runGUI()` before `wails.Run(...)`.

## Decisions Made
- Followed plan's D-08/D-09/D-10 exactly: `goruntime` alias (not renaming the Wails import), no proactive Linux Edit submenu, and an `os.LookupEnv`-gated `os.Setenv` for the DMABUF var.
- No deviations from the plan's prescribed approach.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `go build ./...`, `go vet ./...`, and `go test .` (full package, 31.9s) all pass after both commits.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- main.go changes are self-contained and compile/vet clean; no downstream code depends on this fix beyond the manual TESTING.md M-NN item plan 176-04 will add.
- Live Linux/Wayland verification (segfault-free launch, working menu bar, no freeze) remains unautomatable on this macOS dev box; per plan D-11 this does not block shipping — the reporter's from-source verification on Pop!_OS 24.04/COSMIC/Wayland is accepted as sufficient, with the opportunistic manual check tracked via 176-04's TESTING.md addition.
- Ready for 176-02/176-03/176-04 to proceed independently (this plan has no downstream code dependents within phase 176 per its `depends_on: []`).

---
*Phase: 176-platform-hardening-bug-fixes*
*Completed: 2026-07-09*

## Self-Check: PASSED

- FOUND: main.go
- FOUND: 2bdf7edc (Task 1 commit)
- FOUND: 9625f760 (Task 2 commit)
- FOUND: SUMMARY.md
</content>
