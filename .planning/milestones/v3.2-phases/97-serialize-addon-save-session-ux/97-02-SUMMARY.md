---
phase: 97-serialize-addon-save-session-ux
plan: "02"
subsystem: ui
tags: [phase-97, serialize, lib-helpers, pure-functions, wave-1, ansi-strip, filename-sanitize, vitest]

# Dependency graph
requires:
  - phase: 97-01
    provides: Wave 0 RED scaffolds for stripAnsi.test.ts and sanitizeFilename.test.ts; addon-serialize vendored

provides:
  - "frontend/src/lib/stripAnsi.ts — pure ANSI-stripping helper using /\\x1b\\[\\??[0-9;]*[a-zA-Z]/g regex, audit-verified against SerializeAddon emit vocabulary"
  - "frontend/src/lib/sanitizeFilename.ts — pure 4-step filename sanitizer with Windows reserved name guard, leading-dot guard, whitespace collapse; returns 'session' as neutral fallback"
  - "12 unit tests GREEN (6 per helper) covering all escape categories and all filename guard cases"

affects: [97-03, plan-97-03-app-tsx-saver, app-tsx-handleRequestSave]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure helper pattern: single-file named export, doc-comment citing phase + req IDs + RESEARCH section, NO DOM/network/logging"
    - "Regex constant extracted to module scope (ANSI_ESCAPE_RE, WINDOWS_RESERVED_RE) for clarity and testability"
    - "TDD RED→GREEN: replace expect.fail() scaffolds with real assertions; implementation satisfies all 6 test cases per helper"

key-files:
  created:
    - frontend/src/lib/stripAnsi.ts
    - frontend/src/lib/sanitizeFilename.ts
  modified:
    - frontend/src/lib/__tests__/stripAnsi.test.ts
    - frontend/src/lib/__tests__/sanitizeFilename.test.ts

key-decisions:
  - "Strip regex /\\x1b\\[\\??[0-9;]*[a-zA-Z]/g is audit-verified against SerializeAddon._serializeString() emit vocabulary; covers SGR/ECH/CUF/CUB/CUU/CUD/DEC private modes; no OSC/DCS sequences emitted per excludeModes:true flag"
  - "sanitizeFilename returns literal 'session' (not empty string) for all unsafe inputs — neutral, descriptive fallback that never collides with a real user session name"
  - "Both helpers import nothing (zero transitive deps) — matches plan requirement; strip logic is single-pass replace()"

patterns-established:
  - "Pure helper pattern: follow urlSafety.ts header style; no default exports; no class wrappers; camelCase function names"
  - "Filename fallback literal: 'session' is the canonical fallback for all guard branches in sanitizeFilename"

requirements-completed: [SER-01]

# Metrics
duration: 8min
completed: 2026-05-07
---

# Phase 97 Plan 02: stripAnsi + sanitizeFilename Pure Helpers Summary

**Two zero-dep TypeScript helpers that strip ANSI escapes (single-regex replace) and sanitize filenames (4-step pipeline with Windows reserved-name guard) — 12 unit tests flipped from RED to GREEN, ready for Plan 97-03 App.tsx import**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-05-07T19:20:00Z
- **Completed:** 2026-05-07T19:28:46Z
- **Tasks:** 1
- **Files modified:** 4

## Accomplishments
- Implemented `frontend/src/lib/stripAnsi.ts` — ~25 lines, regex `/\x1b\[\??[0-9;]*[a-zA-Z]/g` in single-pass `String.prototype.replace()`, doc-comment cites Phase 97 SER-01 + RESEARCH §"ANSI Output Audit"
- Implemented `frontend/src/lib/sanitizeFilename.ts` — ~32 lines, 4-step pipeline (trim/whitespace-collapse → char-allowlist → leading-dot guard → Windows reserved name guard), returns `'session'` as neutral fallback
- Flipped all 12 RED scaffolds (6 per helper) to GREEN assertions covering every escape category and every filename guard case; `pnpm tsc --noEmit` exits 0

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement stripAnsi() + sanitizeFilename() pure helpers and flip both RED test scaffolds to GREEN** - `b4dd59b` (feat)

## Files Created/Modified
- `frontend/src/lib/stripAnsi.ts` — Pure ANSI-stripping helper; single-regex replace over SerializeAddon emit vocabulary (SGR, ECH, CUF/CUB/CUU/CUD, DEC private modes)
- `frontend/src/lib/sanitizeFilename.ts` — Pure 4-step filename sanitizer; Windows reserved name guard (CON/PRN/AUX/NUL/COM[1-9]/LPT[1-9]); leading-dot guard; `'session'` fallback
- `frontend/src/lib/__tests__/stripAnsi.test.ts` — 6 RED expect.fail() scaffolds replaced with real assertions (SGR, ECH, cursor-moves, DEC private modes, plain-text, round-trip fixture)
- `frontend/src/lib/__tests__/sanitizeFilename.test.ts` — 6 RED expect.fail() scaffolds replaced with real assertions (path traversal, empty, leading-dot, Windows reserved, whitespace collapse, preserved chars)

## Decisions Made
- Regex constant extracted to module scope (`ANSI_ESCAPE_RE`) to avoid RegExp object recreation on every call and improve readability — regex literal with `g` flag is stateless in a `replace()` call so this is safe
- `WINDOWS_RESERVED_RE` uses `$` anchor to match full basename only (not prefix match), preventing false positives like `CONSOLE`
- Purity guard grep in plan acceptance criteria catches doc-comment text ("NO DOM" etc.) — this is expected; the actual code lines contain no DOM/network/console references; verified with comment-filtered grep

## Deviations from Plan

None - plan executed exactly as written. Both helpers match the exact code shown in the plan's `<action>` steps. Test files match the exact assertions specified.

## Issues Encountered

- **Worktree base reset required:** Worktree was at `cfd0155` (pre-97-01); the `<worktree_branch_check>` correctly identified that `06ade9ef` was not an ancestor, and `git reset --hard 06ade9ef` was performed per the worktree_branch_check protocol. After reset, all 97-01 artifacts (RED scaffold test files, vendored serialize addon) were present as expected.
- **node_modules absent in worktree frontend:** Worktree had no `node_modules/` under `frontend/`; `pnpm install --frozen-lockfile` succeeded in 4.6s.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. Both helpers are pure functions that accept strings and return strings. No threat flags to report.

## Known Stubs

None — both helpers are complete implementations. No placeholder logic, no hardcoded empty values that flow to rendering.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `stripAnsi` and `sanitizeFilename` are ready for Plan 97-03 to import in `App.tsx handleRequestSave`
- Import paths: `import { stripAnsi } from './lib/stripAnsi'` and `import { sanitizeFilename } from './lib/sanitizeFilename'`
- No blockers

---
*Phase: 97-serialize-addon-save-session-ux*
*Completed: 2026-05-07*
