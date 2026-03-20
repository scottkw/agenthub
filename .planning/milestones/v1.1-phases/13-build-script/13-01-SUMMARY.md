---
phase: 13-build-script
plan: 01
subsystem: build
tags: [build, wails, cross-platform, shell]
dependency_graph:
  requires: []
  provides: [build.sh]
  affects: []
tech_stack:
  added: []
  patterns: [bash-argument-parsing, shell-function-dispatch, docker-cross-compile]
key_files:
  created:
    - build.sh
  modified: []
decisions:
  - "build/bin/ was already gitignored — no .gitignore changes needed"
  - "sign_and_notarize() left as stub exiting 1 — Plan 02 implements real signing pipeline"
  - "Dispatch order in --all: macos first (native/fastest), then windows, then linux (Docker/slowest)"
metrics:
  duration: "2m 36s"
  completed_date: "2026-03-20"
  tasks_completed: 2
  files_changed: 1
---

# Phase 13 Plan 01: Build Script Summary

## One-liner

Cross-platform build dispatch script using wails build with mingw32 for Windows and Docker cross-wails image for Linux.

## What Was Built

`build.sh` — a single-entry-point shell script at the project root that compiles AgentHub for macOS, Windows, and Linux using Wails v2.

**Capabilities:**
- `--platform macos` — invokes `wails build -platform darwin/universal -clean`
- `--platform windows` — sets `CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1`, invokes `wails build -platform windows/amd64 -clean`
- `--platform linux` — runs Docker container `ghcr.io/abjrcode/cross-wails:v2.6.0`
- `--all` — runs all three platforms sequentially (macOS → Windows → Linux)
- `--sign` — hook for Plan 02 code signing (currently stubs with exit 1 and clear message)

**Prerequisite checks:**
- Wails binary at `$(go env GOPATH)/bin/wails` must be executable
- `x86_64-w64-mingw32-gcc` must be on PATH for Windows builds
- `docker info` must succeed for Linux builds (clear error if Docker Desktop not running)

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 | Create build.sh with argument parsing and all platform build functions | ea3362c | build.sh |
| 2 | Validate build.sh with macOS native build integration test | — | (validation only) |

## Verification Results

- `bash -n build.sh` — syntax OK
- `test -x build.sh` — executable OK
- `bash build.sh --platform macos` — exit 0, produced `build/bin/agenthub.app`
- `build/bin/agenthub.app/Contents/MacOS/agenthub` — binary exists
- `bash build.sh` (no args) — exits 1, prints "Usage:"
- `bash build.sh --platform bogus` — exits 1, prints "Usage:"
- `cd frontend && pnpm test` — 73/73 tests pass (no regressions)

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- build.sh exists: FOUND
- ea3362c commit exists: FOUND
- build/bin/agenthub.app produced by integration test: FOUND
- 73 frontend tests passing: CONFIRMED
