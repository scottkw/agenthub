---
phase: 118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi
plan: 01
subsystem: internal/files (sandboxed read-only filesystem)
tags: [security, sandbox, fuzz, toctou, files, foundation]
one_liner: "TOCTOU-safe Sandbox built on Go 1.24+ os.Root with 49-payload FuzzSandboxPath gate (60s, zero crashes)"
dependency_graph:
  requires: []
  provides:
    - "files.NewSandbox(workDir) (*Sandbox, error)"
    - "(*Sandbox).Open(relPath) (*os.File, error)"
    - "(*Sandbox).Stat(relPath) (os.FileInfo, error)"
    - "(*Sandbox).RootPath() string"
    - "TOCTOU-safe atomic-open semantics consumable by Plans 02 (Handler), 05 (daemon routes)"
  affects:
    - "internal/files (new package; zero internal coupling)"
tech_stack:
  added:
    - "Go stdlib os.Root / os.OpenRoot (Go 1.24+)"
    - "Go stdlib filepath.EvalSymlinks (cwd resolution at sandbox creation)"
    - "Go stdlib testing.F (FuzzSandboxPath, 49 seed payloads)"
  patterns:
    - "Cached EvalSymlinks-resolved rootPath at NewSandbox time; never re-resolved per request"
    - "Layered-defense validateRelativePath BEFORE os.Root: null bytes / absolute / drive letter / UNC / ADS colon / Windows device names / traversal"
    - "Cross-platform Windows device name reject list (CON/NUL/PRN/AUX/COM1-9/LPT1-9) applied on Linux/macOS too"
    - "filepath.Clean BEFORE final traversal check; os.Root is the terminal security boundary"
key_files:
  created:
    - "internal/files/sandbox.go"
    - "internal/files/sandbox_test.go"
  modified: []
decisions:
  - "Use os.Root.Open per-request (atomic kernel-level) rather than two-step EvalSymlinks+Open — eliminates the symlink TOCTOU race window (CVE-2026-27976 Zed class)"
  - "Seed FuzzSandboxPath via f.Add() calls in source (not testdata/fuzz/ files) — 49 payloads visible in code review per PITFALLS.md Pitfall 8 recommendation"
  - "Windows device names rejected on ALL platforms (not //go:build windows) — Windows-written directories copied to Linux can contain literal CON.txt"
  - "Total ban on ':' in relative paths — rare in legitimate filenames; security trade-off favours blanket rejection of NTFS alternate data streams"
  - "Empty workDir is a hard rejection — silently coercing '' to process cwd would be the exact 'silent fallback' anti-pattern CLAUDE.md warns against"
metrics:
  duration: "3m55s"
  completed: "2026-05-20"
  tasks_completed: 1
  files_created: 2
  files_modified: 0
  fuzz_executions: 4489979
  fuzz_interesting: 200
  fuzz_crashes: 0
  fuzz_duration: "61.032s"
---

# Phase 118 Plan 01: FS Sandbox Core (internal/files) Summary

## One-Liner

TOCTOU-safe `internal/files.Sandbox` built on Go 1.24+ `*os.Root` with a 49-payload `FuzzSandboxPath` merge gate that runs `-fuzztime=60s` with zero crashes — the foundation for Phase 118's daemon file routes and Phase 119's webserver mount.

## What Was Built

Plan 01 lands the dependency-free `internal/files/` package, which every subsequent file-API surface (Plans 02-05, then Phases 119-121) consumes:

- **`internal/files/sandbox.go`** — the `Sandbox` type that wraps an EvalSymlinks-resolved absolute working directory. `Open(relPath)` and `Stat(relPath)` open via `os.OpenRoot(rootPath).Open(cleaned)` — atomic at the kernel level (openat2 on Linux), eliminating the TOCTOU race window that exploited the Zed code editor (CVE-2026-27976, 8.8 CVSS). `RootPath()` exposes the cached resolved path for diagnostics.
- **`validateRelativePath`** — layered-defense check applied BEFORE `os.Root` sees the path. Rejects:
  - empty paths, null bytes anywhere
  - `filepath.IsAbs` paths (`/etc/passwd`, `C:\Windows`)
  - drive-letter prefix `X:` (catches `C:/...` on non-Windows where `IsAbs` misses it)
  - UNC paths (`\\server\share`, `//server/share`)
  - any `:` (alternate data streams — total ban; rare in legitimate filenames)
  - Windows reserved device names (CON, NUL, PRN, AUX, COM1-9, LPT1-9), case-insensitive, with or without extension, **on all platforms** (a Windows-written directory copied to Linux can contain a literal `CON.txt`)
  - `..` traversal that escapes after `filepath.Clean` (defense-in-depth; `os.Root` also rejects)
- **`internal/files/sandbox_test.go`** — 6 unit tests + 1 sub-tests-laden validator test (24 reject cases, 6 accept cases) + 1 symlink-escape test (skipped on Windows) + **`FuzzSandboxPath`** seeded with the full 49-payload corpus from `.planning/research/PITFALLS.md` §Fuzz Corpus Skeleton.

## Tests / Verification

### Unit tests (`go test ./internal/files/`)

```
PASS: TestSandbox_OpensViaOSRoot
PASS: TestNewSandbox_EmptyWorkDirRejected
PASS: TestNewSandbox_NonExistentRejected
PASS: TestNewSandbox_FileNotDirRejected
PASS: TestSandbox_OpenSubpath
PASS: TestSandbox_OpenDot
PASS: TestValidatePath_Rejects (24 sub-cases)
PASS: TestValidatePath_AcceptsLegitimate (6 sub-cases)
PASS: TestSandbox_SymlinkEscapeBlocked
PASS: FuzzSandboxPath (49 seed cases — short-mode harness)
ok  	github.com/scottkw/agenthub/internal/files	0.019s
```

### Fuzz merge gate (`go test -fuzz=FuzzSandboxPath -fuzztime=60s ./internal/files/`)

```
fuzz: elapsed: 1m0s, execs: 4489979 (92079/sec), new interesting: 151 (total: 200)
PASS
ok  	github.com/scottkw/agenthub/internal/files	61.032s
```

**4,489,979 executions in 60 seconds, 200 interesting inputs discovered, zero crashes.** Merge gate satisfied — every payload either is rejected by `validateRelativePath` or is opened to a handle that successfully `Stat()`s inside the root (proxy for "no escape"). The 200 interesting inputs reflect the wide rejection surface — most accepted inputs are legitimate-shaped relative paths inside the temp root, which the fuzzer kept exploring without finding a panic or escape.

## Requirements Satisfied

| Req ID | Description | Evidence |
|--------|-------------|----------|
| FS-01 | Sandbox uses `os.OpenInRoot`, not EvalSymlinks+Open | `sandbox.go::Open` uses `root.Open(cleaned)`; `TestSandbox_OpensViaOSRoot` + `TestSandbox_SymlinkEscapeBlocked` |
| FS-08 | Sandbox rejects abs / `..` / encoded / Unicode / null / device-name / ADS / short-name / trailing-dot / escaped-symlink | `TestValidatePath_Rejects` (24 cases) + `TestSandbox_SymlinkEscapeBlocked` |
| FS-09 | `FuzzSandboxPath` runs `-fuzztime=60s` with zero crashes | 4.49M execs, 0 crashes, 200 interesting in 61s — captured above |

FS-02 (sessionWorkDirs), FS-03/04/05/06 (daemon routes), FS-07 (0-byte read), FS-10 (HasPerm), FS-12 (capability bit), FS-14 (settings migration) are downstream of this plan (Plans 02-05). FS-11 and FS-13 are Phase 119 (webserver gating).

## Deviations from Plan

**None.** Plan 01's scope is the foundation package — sandbox + tests + fuzz corpus — and was executed exactly as specified by RESEARCH.md Pattern 1 + Pattern 2 + Pitfall 8 (corpus via `f.Add` not testdata files). The implementation choices map 1:1 to RESEARCH.md's Architectural Responsibility Map row "Path sandboxing (TOCTOU-safe resolve)".

The orchestrator's prompt referenced plan file `118-01-PLAN.md` which does not exist on disk — only `118-05-PLAN.md` exists (Plan 05 depends on summaries from Plans 01-04). The success criteria in the orchestrator prompt explicitly call out `go test ./internal/files/...` and `FuzzSandboxPath` — the plan-01 work in isolation — so this plan was executed against the ROADMAP / REQUIREMENTS / RESEARCH triad (which all unambiguously specify the same internal/files package + fuzz gate). No deviation from intent; the missing PLAN.md is a planning-phase artifact gap that does not affect the deliverables.

## Commits

| Hash | Type | Description |
|------|------|-------------|
| 7cedcfa | test | add failing tests for internal/files Sandbox (RED) |
| 351abd9 | feat | implement TOCTOU-safe internal/files Sandbox (GREEN) |

## Known Stubs

None. The package exports `NewSandbox`, `Open`, `Stat`, `RootPath` — all fully wired, no placeholder returns, no TODO markers in the code.

## TDD Gate Compliance

- **RED (7cedcfa, `test(118-01):`):** committed before implementation — `go test ./internal/files/` reported "no non-test Go files" (build failure proves the absence of the implementation).
- **GREEN (351abd9, `feat(118-01):`):** committed after implementation — all unit tests pass + fuzz gate passes 60s/zero crashes.
- **REFACTOR:** not needed; first implementation passes the gate cleanly.

## Self-Check

- `[ -f internal/files/sandbox.go ]` → FOUND
- `[ -f internal/files/sandbox_test.go ]` → FOUND
- `git log --all --oneline | grep -q 7cedcfa` → FOUND
- `git log --all --oneline | grep -q 351abd9` → FOUND
- `go test ./internal/files/ -count=1` → PASS
- `go test -fuzz=FuzzSandboxPath -fuzztime=60s ./internal/files/` → PASS (4.5M execs, 0 crashes)

## Self-Check: PASSED
