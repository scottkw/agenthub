# Phase 93 — Deferred Items

Out-of-scope discoveries logged during execution; not in any plan's
files-modified list.

## Pre-existing test environment failures

### Sidebar.test.tsx — `localStorage.clear is not a function` (20 failing tests)

**Discovered during:** Plan 93-03 Task 3 verification (`pnpm exec vitest run`).
**Root cause:** Pre-existing — verified on base commit `e858261` (before any
93-03 changes); same 20 tests fail with the same `TypeError`. The test file
calls `localStorage.clear()` in `beforeEach`, but the jsdom environment in
this project does not expose a writeable localStorage in some contexts.
**Affected:** `frontend/src/components/__tests__/Sidebar.test.tsx` (only).
**Not a 93-03 regression:** baseline run (before Task 3) shows the same
failure; baseline run after Task 3 shows the same failure with the same
20 tests.
**Recommended fix (out of scope here):** add a Vitest setup file that polyfills
localStorage on `window` (e.g., `Object.defineProperty(window, 'localStorage', { value: ... })`)
or switch the test environment to `happy-dom` for the affected suite.
**Tracking:** for a future infra plan; should not block Phase 93.
