---
phase: 149-google-antigravity-agent
plan: "03"
subsystem: docs/testing
tags: [testing, documentation, regression-suite, agent-detection]
dependency_graph:
  requires: ["149-01", "149-02"]
  provides: ["TESTING.md AGENT-01 registration", "README waitlist note", "full-suite green gate"]
  affects: ["TESTING.md", "README.md", "internal/release/no_autosave_test.go"]
tech_stack:
  added: []
  patterns: ["TESTING.md standing convention", "traceability path-check"]
key_files:
  created: []
  modified:
    - TESTING.md
    - README.md
    - internal/release/no_autosave_test.go
decisions:
  - "M-15 category I item added to TESTING.md Section 5 (D-03 waitlist fallback)"
  - "README updated in both intro paragraph and CLI auto-detection bullet"
  - "cmd/playwright-fixture/dist added to SER-03 skip-list (pre-existing untracked build artifact)"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-23"
  tasks_completed: 3
  tasks_total: 3
  files_changed: 3
---

# Phase 149 Plan 03: TESTING.md Registration + README + Full Suite Summary

TESTING.md updated with 5 AGENT-01 traceability rows and M-15 live-launch manual item; README documents Google Antigravity with waitlist note; full project test suite (Go + vitest + tsc + vite build) all green.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Register agy tests in TESTING.md Suite Manifest + Traceability | eed713df | TESTING.md |
| 2 | Add M-15 live-launch item and README waitlist note | fc48490f | TESTING.md, README.md |
| 3 | Full-suite phase gate | 29f5e77f | internal/release/no_autosave_test.go |

## What Was Built

**TESTING.md — Suite Manifest (Section 2):** Appended a Phase 149 / AGENT-01 sentence to the running "> Note:" line documenting that five existing files were extended with no file-count delta.

**TESTING.md — Traceability Map (Section 4):** Added five AGENT-01 rows:
- `internal/pty/detect_test.go` — TestKnownCLIs / TestDetectCLIs_FindsAgy / TestDetectCLI_AgyNotFound
- `internal/daemon/path_windows_test.go` — TestPlatformExtraBins_WindowsIncludesAgyBin
- `internal/status/detector_test.go` — TestDetector_AgyIdle / TestDetector_AgyWaiting
- `frontend/src/lib/agentBadge.test.ts` — agentBadgeModifier('agy') === 'agy'
- `frontend/src/components/__tests__/style.hub.test.ts` — agy three CSS color sites + WCAG comment

`bash tests/check-traceability-paths.sh` exits 0.

**TESTING.md — Manual Checklist (Section 5):** Added new "Category I — Live Agent Launch (AGENT-01)" with M-15 item covering live agy REPL launch, OAuth auth, badge color `#ff9e64`, and card/tab color lockstep — including _Why not automatable_ and _Source_ lines per format parity with M-14.

**README.md:** Added "Google Antigravity" to both the intro paragraph (with waitlist note) and the "CLI auto-detection" feature bullet (with waitlist note referencing TESTING.md M-15).

## Full Suite Results

| Suite | Result |
|-------|--------|
| `go test ./...` | PASS (all packages) |
| `pnpm exec vitest run` | PASS (117 files, 1878 tests) |
| `pnpm exec tsc --noEmit` | PASS (no type errors) |
| `pnpm exec vite build` | PASS (built in 287ms) |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] SER-03 test scanning untracked playwright fixture build artifact**
- **Found during:** Task 3 (full-suite gate)
- **Issue:** `TestSER03_NoAutoSavePatterns` in `internal/release/no_autosave_test.go` scanned `cmd/playwright-fixture/dist/assets/index-Dklc5ak1.js` (an untracked, locally-built Playwright fixture bundle). The minified React/HTML code contained `autoSave` as a standard HTML textarea attribute name — not an auto-save feature. The test's skip-list excluded `frontend/dist` and `dist` but not `cmd/playwright-fixture/dist`.
- **Fix:** Added `"cmd/playwright-fixture/dist": true` to the `skipDirs` map, matching the rationale of the existing `frontend/dist` entry (minified/mangled build artifacts may incidentally contain forbidden substrings).
- **Files modified:** `internal/release/no_autosave_test.go`
- **Commit:** 29f5e77f

## Known Stubs

None — this plan is documentation-only; no UI components or data sources.

## Threat Flags

None — this plan adds prose documentation only. No new network endpoints, auth paths, or schema changes. T-149-03 (auth-guidance text) is trivially satisfied: README directs users to run `agy auth login` externally, no credential input fields added.

## Self-Check: PASSED

- TESTING.md exists and contains M-15: confirmed
- TESTING.md AGENT-01 count >= 5: `grep -c 'AGENT-01' TESTING.md` = 6
- README.md contains "Antigravity" (case-insensitive count >= 1): 2 occurrences
- README.md contains "waitlist": confirmed
- `bash tests/check-traceability-paths.sh` exits 0: confirmed
- Commits eed713df, fc48490f, 29f5e77f all exist in git log: confirmed
