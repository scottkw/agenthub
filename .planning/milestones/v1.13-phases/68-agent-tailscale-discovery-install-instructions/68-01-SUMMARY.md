---
phase: 68-agent-tailscale-discovery-install-instructions
plan: "01"
subsystem: daemon/path-augmentation
tags: [go, path, discovery, snap, flatpak, cargo, windows, tdd]
dependency_graph:
  requires: []
  provides: [extended-path-augmentation, platformExtraBins]
  affects: [internal/daemon/path.go, internal/daemon/path_windows.go, internal/daemon/path_other.go, internal/daemon/path_test.go]
tech_stack:
  added: []
  patterns: [build-tags, platform-split-files, tdd-red-green]
key_files:
  created:
    - internal/daemon/path_windows.go
    - internal/daemon/path_other.go
  modified:
    - internal/daemon/path.go
    - internal/daemon/path_test.go
decisions:
  - "Used build-tagged path_windows.go / path_other.go rather than runtime.GOOS checks in path.go — follows existing codebase convention (process_windows.go, socket_windows.go)"
  - "snap /snap/bin and flatpak system dir included as plain hardcoded candidates — os.Stat skip handles absence cleanly without any detection logic"
  - "Windows nodejs path uses Programs/nodejs (not Programs/nodejs/bin) — matches plan spec for node installer location"
metrics:
  duration: "~10 minutes"
  completed: "2026-04-12T03:57:28Z"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 4
---

# Phase 68 Plan 01: Extend AugmentServicePath with cargo, snap, flatpak, and Windows paths

**One-liner:** Extended daemon PATH augmentation to discover agent CLIs installed via snap, flatpak, cargo, and Windows npm/pnpm/node/Tailscale with build-tagged platform helpers.

## What Was Built

Extended `AugmentServicePath()` in `internal/daemon/path.go` to add four new cross-platform candidate directories (cargo, snap, flatpak system, flatpak user), plus a `platformExtraBins()` call that returns Windows-specific paths on Windows and nil elsewhere.

Two new build-tagged files implement `platformExtraBins()`:
- `path_windows.go` (`//go:build windows`): returns `%APPDATA%\npm`, `%LOCALAPPDATA%\pnpm`, `%LOCALAPPDATA%\Programs\nodejs`, and `C:\Program Files\Tailscale`
- `path_other.go` (`//go:build !windows`): returns nil

Three new tests verify the new behavior: `TestAugmentServicePath_Cargo`, `TestAugmentServicePath_FlatpakUser`, and `TestPlatformExtraBins_NonWindows`.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create platform-specific path helpers and tests (TDD RED) | d072fee | path_windows.go, path_other.go, path_test.go |
| 2 | Extend AugmentServicePath candidates and run tests green | 1697ef6 | path.go |

## Verification

- `go build ./internal/daemon/...` — exit 0
- `go test ./internal/daemon/... -count=1` — 70 tests, all PASS
- `grep -c platformExtraBins internal/daemon/path.go` — 1
- `grep -c platformExtraBins internal/daemon/path_windows.go` — 2
- `grep -c platformExtraBins internal/daemon/path_other.go` — 2

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. All new candidate paths are fully wired into `AugmentServicePath()` with stat-checks.

## Threat Flags

None. All new paths are hardcoded well-known system/user directories. No user input determines which paths are added. The threat model's T-68-01 and T-68-02 dispositions (accept) are correctly applied — no mitigations needed beyond what `os.Stat` already provides.

## Self-Check: PASSED

- `internal/daemon/path_windows.go` — exists, `//go:build windows` on line 1, contains `func platformExtraBins() []string`, `filepath.Join(appdata, "npm")`, `filepath.Join(local, "pnpm")`, `` `C:\Program Files\Tailscale` ``
- `internal/daemon/path_other.go` — exists, `//go:build !windows` on line 1, contains `func platformExtraBins() []string`, `return nil`
- `internal/daemon/path_test.go` — contains `TestAugmentServicePath_Cargo`, `TestAugmentServicePath_FlatpakUser`, `TestPlatformExtraBins_NonWindows`
- `internal/daemon/path.go` — contains `.cargo/bin`, `/snap/bin`, `/var/lib/flatpak/exports/bin`, `flatpak/exports/bin` (user), `candidates = append(candidates, platformExtraBins()...)`, doc comment contains `snap, flatpak, cargo`
- Commits d072fee and 1697ef6 — verified in git log
