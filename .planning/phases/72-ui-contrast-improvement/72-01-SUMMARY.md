---
phase: 72-ui-contrast-improvement
plan: "01"
subsystem: frontend/testing
tags: [wcag, contrast, testing, vitest, css, ui-polish]
dependency_graph:
  requires: []
  provides: [72-01-contrast-tests]
  affects: [frontend/src/components/__tests__/style.contrast.test.ts]
tech_stack:
  added: []
  patterns: [fs.readFileSync CSS inspection, WCAG luminance formula, selector-scoped regex assertions]
key_files:
  created:
    - frontend/src/components/__tests__/style.contrast.test.ts
  modified: []
decisions:
  - "Used selector-scoped regex patterns (not plain toContain) to ensure assertions are specific to the correct CSS rule block — prevents false passes if #565f89 appears in a comment or unrelated selector"
  - "Embedded WCAG helper functions directly in test file rather than importing — keeps the test self-contained and avoids module dependency issues"
  - "RED state confirmed as expected — 12 selector assertions fail against unmodified style.css, 3 contrast math tests pass, 1 preservation test passes"
metrics:
  duration: "~5 minutes"
  completed: "2026-04-14"
  tasks_completed: 1
  files_created: 1
requirements_completed: [UI-01]
---

# Phase 72 Plan 01: WCAG Contrast Test File Summary

WCAG AA contrast regression test file created with 16 tests covering all UI-01 color assertions — math tests pass, selector tests fail (RED state) confirming the test harness correctly detects the #565f89 contrast failures.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create WCAG contrast test file with all UI-01 assertions | 4e45e68 | frontend/src/components/__tests__/style.contrast.test.ts |

## Decisions Made

1. **Selector-scoped regex over plain toContain** — `expect(css).not.toMatch(/\.tab\s*\{[^}]*color:\s*#565f89/)` ensures assertions target the specific CSS rule block. A simple `not.toContain('#565f89')` would fail as long as any #565f89 exists anywhere in the file — even in a comment or different rule. The regex pattern scopes to the exact selector block.

2. **Embedded WCAG helper functions** — `sRGB`, `relativeLuminance`, and `contrastRatio` functions are defined directly in the test file, not imported from a utility. This keeps the test file fully self-contained with no additional module dependencies.

3. **RED state design** — The test file is intentionally authored to fail against the unmodified `style.css`. This confirms that the test harness correctly detects contrast violations. Plan 02 will update the CSS to make all 12 selector assertions pass.

## Test Results (RED State)

Total tests: **16**

| Group | Tests | Status |
|-------|-------|--------|
| UI-01: replacement color passes WCAG AA on all backgrounds | 3 | PASS |
| UI-01: tab bar contrast — no failing #565f89 text color | 4 | FAIL (RED) |
| UI-01: settings panel contrast — no failing #565f89 text color | 5 | FAIL (RED) |
| UI-01: welcome tab contrast — no failing #565f89 text color | 2 | FAIL (RED) |
| UI-01: modal contrast — no failing #565f89 text color | 1 | FAIL (RED) |
| UI-01: intentionally-dim elements preserved | 1 | PASS |

**Passing:** 4 (3 contrast math + 1 preservation)
**Failing:** 12 (selector checks detecting current #565f89 violations — expected RED state)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. This is a test file only — no UI stubs present.

## Threat Flags

None. This plan creates a test file only — no network endpoints, auth paths, file access beyond local CSS, or schema changes.

## Self-Check: PASSED

- File exists: `frontend/src/components/__tests__/style.contrast.test.ts` — FOUND
- Commit `4e45e68` exists in git log — FOUND
- File contains `import { describe, it, expect } from 'vitest'` — YES
- File contains `readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')` — YES
- File contains `function contrastRatio` — YES
- File contains `function relativeLuminance` — YES
- File contains `function sRGB` — YES
- File contains `contrastRatio('#9aa5ce', '#16161e')` — YES
- File contains `contrastRatio('#9aa5ce', '#1a1b26')` — YES
- File contains `contrastRatio('#9aa5ce', '#1e2030')` — YES
- File contains `.toBeGreaterThanOrEqual(4.5)` — YES
- File contains all 12 selector regex patterns — YES
- File contains `.tab-status-bar__state--inactive` preservation test — YES
- Test run produces 16 tests: 4 pass, 12 fail (expected RED state) — CONFIRMED
