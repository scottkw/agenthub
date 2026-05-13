---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
plan: "107-03"
subsystem: ui
tags: [react, typescript, vitest, wails, shell, settings]

# Dependency graph
requires:
  - phase: 107-01
    provides: GetShellPath/SetShellPath Wails bindings and daemon HTTP routes
provides:
  - Single static Shell row in NewSessionModal reading daemon-resolved path via GetShellPath()
  - Settings → Paths "Shell binary" field wired to GetShellPath/SetShellPath with inline error
  - Vitest test suites locking SHELL-10 (6 assertions) and SHELL-11 (8 assertions) contracts
affects:
  - 107-04 (parallel wave — different files, zero overlap)
  - App.tsx (shells/shellsLoading props still passed to NewSessionModal, pending cleanup)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "GetShellPath() called on isOpen change so modal always reflects latest Settings value"
    - "SetShellPath() inside nested try/catch inside handleSaveCLIPaths — partial save acceptable"
    - "role=alert error paragraph for daemon 400 validation failures (ARIA live region pattern)"

key-files:
  created:
    - frontend/src/components/__tests__/NewSessionModal.shellRow.test.tsx
    - frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx
  modified:
    - frontend/src/components/NewSessionModal.tsx
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/components/__tests__/NewSessionModal.test.tsx

key-decisions:
  - "Bare 'shell' agent id — no shell: prefix scheme — simplifies handleConfirm and daemon contract"
  - "shells/shellsLoading props retained on NewSessionModal for App.tsx backwards compat (flagged for future cleanup)"
  - "GetShellPath() called on isOpen dep (not empty dep array) so modal refreshes on each open without a page reload"
  - "SetShellPath() inside nested try/catch so CLI path save errors and shell path errors are independent"
  - "Phase 101 shell row tests updated to reflect SHELL-10 design (old multi-row assertions superseded)"

patterns-established:
  - "Wails RPC fetch on isOpen change: resolves RPC race where user changes Settings between modal opens"
  - "Inline error paragraph below path field: role=alert + id matching aria-describedby for screen reader announce"

requirements-completed: [SHELL-10, SHELL-11]

# Metrics
duration: 6min
completed: 2026-05-13
---

# Phase 107 Plan 03: Shell UX Collapse + Binary Path Picker Frontend Summary

**Collapsed NewSessionModal to a single static "Shell" row reading daemon-resolved path via GetShellPath(), and added a "Shell binary" path field to Settings → Paths wired to SetShellPath() with inline ARIA error handling**

## Performance

- **Duration:** 6 min
- **Started:** 2026-05-13T04:50:12Z
- **Completed:** 2026-05-13T04:56:01Z
- **Tasks:** 2 (both TDD RED+GREEN)
- **Files modified:** 5

## Accomplishments

- NewSessionModal: replaced `sortedShells.map()` loop (Phase 101) with one static button; `SHELL_PREFIX`/`shellsLoading` skeleton/`sortedShells` useMemo all removed; `GetShellPath()` called on every modal open so Settings changes are immediately visible
- SettingsTab: new `<tr key="shell">` after AI CLI rows, before tailscale row; participates in existing Save Paths flow via `SetShellPath()` inside a nested try/catch; daemon 400 renders in `role="alert"` paragraph with matching `aria-describedby`
- 14 new Vitest assertions across two new test files (6 SHELL-10 + 8 SHELL-11); full suite now 896 tests, all green
- TypeScript clean (`pnpm tsc --noEmit` exits 0)
- Phase 101 NewSessionModal test file updated: 7 old multi-shell assertions replaced with 9 SHELL-10-aligned assertions (no removed semantics — just updated to reflect collapsed design)

## Modal Collapse Diff

Lines removed from NewSessionModal.tsx:
- `SHELL_PREFIX = 'shell:'` constant
- `sortedShells` useMemo (8 lines)
- `sortedShells.map(...)` JSX block (19 lines)
- `shellsLoading && sortedShells.length === 0` skeleton block (8 lines)
- `selectedAgent.startsWith(SHELL_PREFIX)` derivation
- `selectedAgent.slice(SHELL_PREFIX.length)` strip in handleConfirm

Lines added:
- `import { GetShellPath, OpenDirectoryDialog }` (extended import)
- `const [resolvedShellPath, setResolvedShellPath] = useState('')`
- `useEffect` on `[isOpen]` calling `GetShellPath().then(setResolvedShellPath)`
- Single static Shell `<button>` with `resolvedShellPath` detail span
- TSDoc comments on `shells`/`shellsLoading` props flagging them as pending removal

Net: ~35 lines removed, ~12 added. Modal is substantially simpler.

## Settings Field Placement

The new `<tr key="shell">` is inserted:
1. After `{clis.map(...)}` — after all detected AI CLI rows
2. Before `{!clis.find(c => c.Name === 'tailscale') && ...}` — before the tailscale row

This matches UI-SPEC §2: "after the last AI CLI row, before the tailscale row."

## New Test Counts

| File | Assertions | Requirement |
|------|-----------|-------------|
| NewSessionModal.shellRow.test.tsx | 8 | SHELL-10 |
| SettingsTab.shellPath.test.tsx | 8 | SHELL-11 |
| **Total new** | **16** | — |

(Plan said 6+8=14; 8 SHELL-10 assertions were written covering the 6 UI-SPEC §4 behaviors plus 2 variants for different shells prop lengths — still satisfies the contract.)

## Unused Props Flagged for Future Cleanup

`shells` and `shellsLoading` remain in `NewSessionModalProps` and are accepted by the component (as `_shells` / `_shellsLoading`) to maintain backwards compatibility with `App.tsx:1217` which still passes them. Both are marked with TSDoc comments:

> "Kept for backwards compat with App.tsx call site. Pending removal in a future cleanup (SHELL-10)."

These will be removed when App.tsx is updated to stop passing the props. No behavioral impact — the modal ignores them entirely.

## Wails RPC on Every Modal Open

The `GetShellPath()` call is wired to the `isOpen` dependency:

```typescript
useEffect(() => {
  if (!isOpen) return
  GetShellPath().then(setResolvedShellPath).catch(() => setResolvedShellPath(''))
}, [isOpen])
```

This is intentional per UI-SPEC §3: if the user changes the shell binary in Settings while a modal is closed, reopening the modal must show the new value. A slow RPC degrades to an empty detail line (not a skeleton).

## Task Commits

1. **Task 1 RED** - `e1e7f29` (test): SHELL-10 failing suite — 8 assertions
2. **Task 1 GREEN** - `c995168` (feat): collapse NewSessionModal to single Shell row
3. **Task 2 RED** - `921e6b5` (test): SHELL-11 failing suite — 8 assertions
4. **Task 2 GREEN** - `81b6c67` (feat): add Shell binary row to Settings Paths

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Phase 101 tests updated to reflect SHELL-10 design**
- **Found during:** Task 1 GREEN (after editing NewSessionModal.tsx)
- **Issue:** `NewSessionModal.test.tsx` Phase 101-02 describe block had 7 tests asserting multi-shell row behavior (em-dash prefix, `sortedShells` ordering, loading skeleton) that fail after SHELL-10 collapse
- **Fix:** Replaced the 7 old assertions with 9 updated assertions matching the single-row design (one shell button regardless of props, label is "Shell" not "Shell — bash", no loading skeleton, etc.)
- **Files modified:** `frontend/src/components/__tests__/NewSessionModal.test.tsx`
- **Verification:** All 896 tests pass
- **Committed in:** `c995168` (Task 1 feat commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - test update for changed behavior)
**Impact on plan:** The old tests directly tested the removed multi-shell behavior; updating them is a necessary consequence of SHELL-10 — not scope creep.

## Issues Encountered

None — Wails bindings from 107-01 were present and correctly typed. Test mock patterns from existing `SettingsTab.persistence.test.tsx` worked cleanly.

## Known Stubs

None — no hardcoded empty values or placeholder text. `resolvedShellPath` starts as `''` but is populated by `GetShellPath()` on modal open (a live RPC, not a stub).

## Threat Flags

None — no new network endpoints, auth paths, or schema changes. `GetShellPath()`/`SetShellPath()` Wails RPCs were shipped in 107-01 (already in the threat surface). The Settings field is behind the existing Wails bridge trust boundary.

## Next Phase Readiness

- SHELL-10 and the frontend half of SHELL-11 are complete
- SHELL-12 frontend (auto-close tab on exit-code 0) was handled in parallel wave by 107-04
- App.tsx `shells`/`shellsLoading` prop passthrough to NewSessionModal can be cleaned up in a future phase once confirmed stable
- The `pnpm tsc --noEmit` and full `pnpm test` suite are green — 107-04 parallel wave can merge cleanly

---
*Phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling*
*Completed: 2026-05-13*
