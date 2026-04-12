---
phase: 68-agent-tailscale-discovery-install-instructions
plan: "02"
subsystem: frontend
tags: [welcome-tab, install-instructions, homebrew, INST-01]
dependency_graph:
  requires: []
  provides: [INST-01]
  affects: [frontend/src/components/WelcomeTab.tsx]
tech_stack:
  added: []
  patterns: [source-inspection-tests, raw-import-assertion]
key_files:
  modified:
    - frontend/src/components/WelcomeTab.tsx
    - frontend/src/components/__tests__/WelcomeTab.test.tsx
decisions:
  - "Updated macOS install command from 'brew install agenthub' to full 'brew tap scottkw/agenthub && brew install --cask agenthub' single-line command"
  - "Test assertion updated to full command string to lock exact format and prevent substring false positives"
metrics:
  duration: "~5 minutes"
  completed: "2026-04-11"
  tasks_completed: 1
  files_modified: 2
requirements:
  - INST-01
---

# Phase 68 Plan 02: Update macOS Install Command Summary

**One-liner:** macOS Welcome tab install command updated to `brew tap scottkw/agenthub && brew install --cask agenthub` — single copyable command replacing the prior broken `brew install agenthub`.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Update macOS install command and test assertion | add6dc1 | WelcomeTab.tsx, WelcomeTab.test.tsx |

## What Was Built

Pure string replacement in two files:

1. **`frontend/src/components/WelcomeTab.tsx` line 94** — The macOS `<code>` block now shows `brew tap scottkw/agenthub && brew install --cask agenthub` instead of the incorrect `brew install agenthub` (which would fail without the tap being added first).

2. **`frontend/src/components/__tests__/WelcomeTab.test.tsx` line 33** — The macOS install test assertion updated to `toContain('brew tap scottkw/agenthub && brew install --cask agenthub')`. The old assertion `toContain('brew install agenthub')` was a substring of the new string and would still have passed — updating it locks the exact command format and prevents false positives if the command is later shortened.

No structural changes, no new CSS classes, no new components. The `.welcome-tab__code` class handles all styling unchanged.

## Verification

- `grep "brew tap scottkw/agenthub" frontend/src/components/WelcomeTab.tsx` — 1 match confirmed
- `grep "brew tap scottkw/agenthub" frontend/src/components/__tests__/WelcomeTab.test.tsx` — 1 match confirmed
- `cd frontend && pnpm test` — 323/323 tests passed, 18 test files, exit 0

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — the install command is wired display text, not a placeholder.

## Threat Flags

None — display-only string change, no new network surface or trust boundary.

## Self-Check

- [x] `frontend/src/components/WelcomeTab.tsx` — FOUND, contains correct brew tap command
- [x] `frontend/src/components/__tests__/WelcomeTab.test.tsx` — FOUND, contains updated assertion
- [x] Commit `add6dc1` — FOUND in git log

## Self-Check: PASSED
