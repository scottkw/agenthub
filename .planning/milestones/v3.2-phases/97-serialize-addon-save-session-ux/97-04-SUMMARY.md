---
plan: 97-04
phase: 97-serialize-addon-save-session-ux
status: complete
tasks: 2/2
completed: 2026-05-07
---

# Plan 97-04 Summary

## What was built

**Task 1 — TerminalPanel SerializeAddon hot-swap arm:**
- `frontend/src/components/TerminalPanel.tsx`: imported `SerializeAddon` from `@xterm/addon-serialize`, added `serializeAddonRef` alongside other addon refs, added optional `onRegisterSaver` prop to `TerminalPanelProps`.
- Extended hot-swap useEffect with a SerializeAddon arm placed after the WebLinks arm — mirrors the Clipboard arm at lines 367-379 (NOT the Image arm — Serialize is a pure buffer-walker, hot-swap-friendly per 97-PATTERNS.md).
- On attach: registers the saver closure via `onRegisterSaver(tabId, () => addon.serialize({ excludeModes: true }))`. On detach: unregisters with `onRegisterSaver(tabId, null)`.
- Mount-useEffect cleanup also calls `onRegisterSaver(tabId, null)` to prevent registry leaks (Pitfall #6).
- Hot-swap dep array extended with `pluginConfig?.serialize` and `onRegisterSaver`.
- Flipped 9 RED `expect.fail()` scaffolds in `TerminalPanel.test.tsx` to GREEN source-scan assertions, including a hot-swap-vs-mount placement regression test that guards the architectural distinction.

**Task 2 — TabBar Save Terminal As… menuitem + App.tsx bridge removal:**
- `frontend/src/components/TabBar.tsx`: added a SECOND `<button role="menuitem">` labeled "Save Terminal As…" (U+2026 horizontal ellipsis) inside the existing right-click context menu (lines 152-170 region). Click handler invokes the new `onRequestSave` prop.
- `frontend/src/App.tsx`: removed the `@ts-expect-error` wave-bridge attributes — TerminalPanel and TabBar prop interfaces now formally accept the saver callbacks, so no escape hatch is required.
- `frontend/src/components/__tests__/TabBar.test.tsx`: flipped 3 RED scaffolds to GREEN source-scan assertions covering the new menuitem label, the U+2026 ellipsis literal, and the onRequestSave wiring.

## Commits

- `ed3c70e` feat(97-04): extend TerminalPanel with SerializeAddon hot-swap arm + flip 9 RED scaffolds GREEN
- `322aae5` feat(97-04): add Save Terminal As… menuitem to TabBar + remove App.tsx wave-bridges

## Key files

### Modified
- `frontend/src/components/TerminalPanel.tsx` (+45 / SerializeAddon hot-swap arm)
- `frontend/src/components/__tests__/TerminalPanel.test.tsx` (+69 / flip 9 GREEN)
- `frontend/src/components/TabBar.tsx` (+18 / Save Terminal As… menuitem)
- `frontend/src/components/__tests__/TabBar.test.tsx` (+22 / flip 3 GREEN)
- `frontend/src/App.tsx` (-2 / remove @ts-expect-error wave-bridges)

## Verification

- `pnpm tsc --noEmit` clean (App.tsx no longer has wave-bridge attributes)
- `pnpm vitest run TabBar.test.tsx TerminalPanel.test.tsx App.saver.test.tsx` — 70/70 GREEN
- All 12 RED scaffolds across TerminalPanel + TabBar test files now GREEN

## Operational note

Agent execution stalled in stream watchdog after Task 1 commit but before Task 2
commit and SUMMARY.md write. Self-checks for Task 1 had passed; Task 2 changes
were complete in the worktree but uncommitted. Orchestrator recovered:

1. Cherry-picked Task 1 commit (`784cbce` → `ed3c70e`) onto main.
2. Saved Task 2 uncommitted diff via `git -C <worktree> diff` (106 lines).
3. Applied the patch onto main with `git apply`, ran tsc + vitest to confirm
   no regression, then committed Task 2 (`322aae5`).
4. Authored this SUMMARY.md from the agent's commit messages and worktree state.

## Self-Check: PASSED
