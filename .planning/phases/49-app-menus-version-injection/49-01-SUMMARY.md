---
phase: 49-app-menus-version-injection
plan: "01"
subsystem: go-backend
tags: [menus, version-injection, wails, ldflags, macos]
dependency_graph:
  requires: []
  provides: [GetVersion-bound-method, appMenu-function, Version-package-var, ldflags-build-pipeline]
  affects: [frontend/wailsjs, build.sh, release.yml]
tech_stack:
  added: []
  patterns: [wails-menu-api, go-ldflags-version-injection, package-level-context-for-callbacks]
key_files:
  created: []
  modified:
    - main.go
    - app.go
    - build.sh
    - .github/workflows/release.yml
decisions:
  - Package-level appCtx for menu callback context (avoids closure complexity)
  - AppMenu ordered first per STATE.md Wails pitfall documentation
  - openGitHubCallback points to scottkw/agenthub (correct post-v1.8 repo path)
  - -tags wailsassets added to build_macos() per project memory constraint
metrics:
  duration: "155s"
  completed: "2026-04-07"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 4
---

# Phase 49 Plan 01: App Menus and Version Injection Summary

## One-liner

macOS menu bar (AppMenu, File, EditMenu, WindowMenu, Help) wired into Wails options.App.Menu with package-level Version var injected via ldflags across all 6 build paths (3 local + 3 CI).

## What Was Built

**Task 1: Go backend menus and version variable**

- `main.go`: Added `var Version = "dev"` and `var appCtx context.Context` at package scope
- `main.go`: Added `appMenu()` function building AppMenu > File > EditMenu > WindowMenu > Help in correct macOS order
- `main.go`: Added `openGitHubCallback()` using `runtime.BrowserOpenURL(appCtx, ...)` for Help menu item
- `main.go`: Wired `Menu: appMenu()` into `options.App` struct in `runGUI()`
- `app.go`: Added `appCtx = ctx` at top of `startup()` to populate the package-level context for menu callbacks
- `app.go`: Added `GetVersion() string` bound method returning the `Version` package variable

**Task 2: Build system ldflags version injection**

- `build.sh build_macos()`: Added `LOCAL VERSION="${BUILD_VERSION:-dev}"`, `-ldflags "-X main.Version=${VERSION}"`, and `-tags wailsassets`
- `build.sh build_windows()`: Added `local VERSION="${BUILD_VERSION:-dev}"` and `-ldflags "-X main.Version=${VERSION}"`
- `build.sh build_linux()`: Added `local VERSION="${BUILD_VERSION:-dev}"`, `-e VERSION="${VERSION}"` to Docker run, and `-ldflags "-w -s -X main.Version=${VERSION}"` to the inner `go build`
- `release.yml`: Added `go-build-args: -ldflags "-X main.Version=${{ github.ref_name }}"` to all three `wails-build-action` steps (macOS, Windows, Linux)

## Verification Results

- `go build -tags wailsassets ./...` exits 0
- `grep -c 'X main.Version' build.sh` returns 3
- `grep -c 'X main.Version' .github/workflows/release.yml` returns 3
- `var Version = "dev"` at package scope in main.go
- `func (a *App) GetVersion() string` in app.go
- AppMenu appears at line 88, EditMenu at line 95 (correct ordering)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. GetVersion() returns the Version variable which will be properly injected by ldflags in CI and local builds. Frontend consumption of GetVersion() is deferred to Plan 02.

## Self-Check: PASSED

- [x] main.go modified: present and compiled
- [x] app.go modified: present and compiled
- [x] build.sh modified: all 3 functions updated
- [x] .github/workflows/release.yml modified: all 3 build steps updated
- [x] Task 1 commit: 0323d5e
- [x] Task 2 commit: 876c308
