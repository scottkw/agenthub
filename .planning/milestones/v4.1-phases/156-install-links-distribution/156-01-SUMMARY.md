---
phase: 156-install-links-distribution
plan: "01"
subsystem: frontend
tags: [install-strings, source-gate, vitest, tdd, welcome-screen]
status: complete
completed: "2026-06-27"
duration: "2 minutes"
tasks_completed: 3
files_changed: 3

dependency_graph:
  requires: []
  provides:
    - frontend/src/components/__tests__/WelcomeTab.install.test.tsx
    - frontend/src/components/WelcomeTab.tsx (corrected strings)
    - TESTING.md (INSTALL-02 traceability row)
  affects:
    - Welcome screen install commands shown to users
    - INSTALL-01 (curl URL correct)
    - INSTALL-02 (winget id + repo link correct)

tech_stack:
  added: []
  patterns:
    - source-gate vitest test (readFileSync, no render) — matches style.hub.test.ts pattern
    - TDD RED→GREEN flow for string correction

key_files:
  created:
    - frontend/src/components/__tests__/WelcomeTab.install.test.tsx
  modified:
    - frontend/src/components/WelcomeTab.tsx
    - TESTING.md

decisions:
  - "Line 54 span left as <span> text-only (no <a href> added) — per locked docs/install-links-fix.md decision"
  - "Test uses node:fs/node:path imports (matching RESEARCH blueprint exactly)"
  - "Traceability row added for INSTALL-02 (the vitest source-gate); INSTALL-01 traceability will be added in plan 156-02 with the build-script test"
---

# Phase 156 Plan 01: Fix Welcome Screen Install Strings Summary

Corrected three wrong distribution strings on the Welcome screen and locked them with a vitest source-gate test following TDD RED→GREEN discipline.

## One-liner

Source-gate vitest test enforcing correct raw.githubusercontent.com curl URL, `scottkw.agenthub` winget id, and `github.com/scottkw/agenthub` repo link in WelcomeTab.tsx.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Add WelcomeTab source-gate test | 2fa73b84 | frontend/src/components/__tests__/WelcomeTab.install.test.tsx |
| 2 (GREEN) | Fix three wrong Welcome screen strings | 324e930b | frontend/src/components/WelcomeTab.tsx |
| 3 | Register new vitest test in TESTING.md | 9d83339b | TESTING.md |

## What Was Built

- **WelcomeTab.install.test.tsx** — 6-assertion source-gate test reading WelcomeTab.tsx as a raw string (no render). Asserts three correct strings and negatively asserts three wrong strings. Pattern matches the established `style.hub.test.ts` source-gate convention.

- **WelcomeTab.tsx (3 edits):**
  - Line 42: `agenthub.dev/install.sh` → `raw.githubusercontent.com/scottkw/agenthub/main/scripts/install.sh`
  - Line 46: `winget install agenthub` → `winget install scottkw.agenthub`
  - Line 54: `github.com/agenthub-dev/agenthub` → `github.com/scottkw/agenthub` (remains `<span>`, no href)

- **TESTING.md:** vitest count 126 → 127, Total 501 → 502; INSTALL-02 traceability row added.

## Verification Results

- `cd frontend && pnpm vitest run src/components/__tests__/WelcomeTab.install.test.tsx` — 6/6 GREEN
- `bash tests/check-traceability-paths.sh` — exits 0, all paths verified
- Line 54 confirmed `<span>` (no `<a href>` introduced)
- macOS brew command (line 38) confirmed unchanged

## Deviations from Plan

None — plan executed exactly as written. All three strings corrected exactly per RESEARCH blueprint values.

## Known Stubs

None. All three install strings are now the correct, live values:
- Linux curl URL resolves to a real raw.githubusercontent.com file (scripts/install.sh, created in plan 156-02)
- winget id `scottkw.agenthub` is the correct catalog identifier
- Repo link `github.com/scottkw/agenthub` is the correct public repo URL

## Threat Flags

No new threat surface introduced. T-156-05 and T-156-06 (from plan threat model) are now mitigated:
- T-156-05: Source-gate asserts exact `raw.githubusercontent.com/scottkw/agenthub` host, negative-asserts `agenthub.dev`
- T-156-06: Source-gate asserts `scottkw.agenthub` id, negative-asserts bare `agenthub` id

## Self-Check

## Self-Check: PASSED

All files found on disk:
- FOUND: frontend/src/components/__tests__/WelcomeTab.install.test.tsx
- FOUND: frontend/src/components/WelcomeTab.tsx
- FOUND: TESTING.md
- FOUND: .planning/phases/156-install-links-distribution/156-01-SUMMARY.md

All commits verified in git log:
- FOUND: 2fa73b84 (test: RED source-gate)
- FOUND: 324e930b (feat: GREEN string fixes)
- FOUND: 9d83339b (chore: TESTING.md registration)
