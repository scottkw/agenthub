---
phase: 147
plan: "04"
subsystem: regression-test-convention
tags: [help-page, testing-md, traceability, regression-convention]
dependency_graph:
  requires: [147-01, 147-02, 147-03]
  provides: [testing-md-help-registration]
  affects: [TESTING.md]
tech_stack:
  added: []
  patterns: [traceability-path-check, manual-checklist-convention]
key_files:
  created: []
  modified:
    - TESTING.md
decisions:
  - "All four HELP-01 traceability rows and M-14 were already added by 147-01 executor — no duplicate rows created"
  - "Total count in §2 corrected from 468 to 472 (147-01 bumped vitest 112→116 but forgot to update Total)"
metrics:
  duration: ~5 minutes
  completed: "2026-06-23"
  tasks_completed: 2
  files_count: 1
---

# Phase 147 Plan 04: Regression Test Convention Registration Summary

**One-liner:** TESTING.md reconciliation confirming 147-01 already satisfied all traceability and manual-checklist requirements; fixed Total count 468→472 (arithmetic error from 147-01).

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Register Help test files in Suite Manifest + Traceability Map | already satisfied by 147-01 (00fe2c8e) | TESTING.md §2, §4 |
| 2 | Add M-14 manual UAT item for the live Wails Help page | already satisfied by 147-01 (00fe2c8e) | TESTING.md §5 |
| Fix | Correct Total count 468→472 (Rule 1 — arithmetic error) | ca5c5072 | TESTING.md §2 |

## What Was Built

### Reconciliation Finding

The 147-01 executor fully satisfied both plan tasks before this wave-4 plan ran:

**Task 1 — Already done by 147-01:**
- §2 vitest count updated to **116** (was 112; +4 Help test files)
- §4 Traceability Map has four HELP-01 rows:
  - `frontend/src/components/__tests__/HelpTab.test.tsx` — HELP_TAB constant + CSS token source gates
  - `frontend/src/components/__tests__/HelpSearch.test.tsx` — search label, clear button, empty-state, mark highlight
  - `frontend/src/components/__tests__/HelpSectionNav.test.tsx` — section nav buttons, aria-current, onSectionChange
  - `frontend/src/components/__tests__/HelpContent.test.tsx` — react-markdown import gate, BrowserOpenURL, no raw `<a href>`
- Each path column contains a repo-relative `.tsx` path only (no test/describe names)

**Task 2 — Already done by 147-01:**
- §5 Category H "In-App Help Page (HELP-01)" added with M-14 item covering:
  - Sidebar Help button opens Help tab in live native WebView
  - Full Markdown render (headings, paragraphs, inline code spans)
  - Left section-nav active tracking via IntersectionObserver scroll-spy
  - Debounced search with highlighted snippets and "Go to section" jump
  - External links (GitHub, Website) open system default browser
  - Rationale: Wails native webview inaccessible to Playwright/headless automation
  - Source: Phase 147 / HELP-01

### Deviation Fixed (Rule 1 — Bug)

**Total count arithmetic error in §2 Suite Manifest:**
- 147-01 bumped vitest from 112 to 116 (+4) but did not update the Total row
- Pre-147-01: Go 348 + vitest 112 + Playwright 7 + build-script 1 = 468 (correct)
- Post-147-01: Go 348 + vitest 116 + Playwright 7 + build-script 1 = 472 (but Total still showed 468)
- Fixed: Total updated from **468** to **472** (commit ca5c5072)

## Test Results

| Check | Result |
|-------|--------|
| `bash tests/check-traceability-paths.sh` | OK: all traceability paths exist (exit 0) |
| §2 vitest count | 116 (matches actual 116 files on disk) |
| §2 Total count | 472 (corrected from stale 468) |
| §4 HELP-01 rows | 4 rows, all paths resolve on disk |
| §5 M-14 item | Present under Category H |

## Deviations from Plan

### Already Satisfied by 147-01 (Informational)

**Task 1 — Register Help test files in Suite Manifest + Traceability Map**
- Pre-satisfied by commit 00fe2c8e (147-01 Task 4)
- No duplicate rows added

**Task 2 — Add M-14 manual UAT item for the live Wails Help page**
- Pre-satisfied by commit 00fe2c8e (147-01 Task 4)
- No duplicate M-14 item added

### Auto-fixed: Total count arithmetic (Rule 1 — Bug)

**Found during:** Task 1 reconciliation

**Issue:** 147-01 bumped vitest count 112→116 in the individual vitest row but left the Total row at 468. Actual sum is 348+116+7+1=472.

**Fix:** Changed `**468**` to `**472**` in the Total row of §2.

**Files modified:** `TESTING.md`

**Commit:** ca5c5072

## Known Stubs

None — documentation-only plan, no component stubs.

## Threat Flags

No new network endpoints, auth paths, file access patterns, or schema changes. Documentation update only.

## Self-Check: PASSED

- TESTING.md — FOUND
- §2 vitest count = 116 — VERIFIED
- §2 Total count = 472 — VERIFIED
- §4 HELP-01 rows (4) — VERIFIED
- §5 M-14 item — VERIFIED
- `bash tests/check-traceability-paths.sh` — exits 0
- Commit ca5c5072 (fix Total count) — FOUND
