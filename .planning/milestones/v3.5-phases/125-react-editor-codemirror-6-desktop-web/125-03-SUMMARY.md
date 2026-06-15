---
phase: 125-react-editor-codemirror-6-desktop-web
plan: 03
subsystem: frontend-editor
tags: [codemirror, cm6, save-flow, if-match, dirty-state, conflict-modal, unsaved-guard, tdd]
dependency_graph:
  requires:
    - 125-02-SUMMARY.md  # Editor.tsx + filesApi.readFileText returns etag; CM6 mount
    - 125-01-SUMMARY.md  # server If-Match/412 backend; ETag header on reads
  provides:
    - filesApi.ts: isConflict() (412), isCollision() (409), isMissingFilesWritePerm() predicates
    - filesApi.ts: writeFile(sid, path, body, ifMatch?) — PUT with If-Match header, octet-stream
    - useFilesWrite.ts: useFilesWrite hook — write/del/rename/mkdir/upload return shape
      with isSaving/saveState/saveError/isConflict/clearConflict/clearSaveError
    - SaveIndicator.tsx: role=status aria-live=polite three-state (idle/saving/saved) with
      ArrowPathIcon+Saving/CheckCircleIcon+Saved/ExclamationTriangleIcon (colorblind contract)
    - EditorHeader.tsx: dirty ● bullet + aria-label=Modified + SaveIndicator + Save/Cancel
      reusing .file-browser__preview-header structure
    - UnsavedChangesModal.tsx: QuitConfirmModal pattern; verbatim EDIT-07 copy;
      default focus on 'Keep editing' (safe button)
    - ConflictModal.tsx: QuitConfirmModal pattern; verbatim EDIT-08 copy;
      Force overwrite/Save as new/Discard my changes; default focus on 'Discard my changes'
    - FileBrowserTab.tsx: guardThen() helper routes all 3 navigation triggers; ETag
      captured from readFileText; edit mode state; Editor/EditorHeader/modals wired
  affects:
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesWrite.ts
    - frontend/src/components/FileBrowserTab.tsx
    - frontend/src/components/FileBrowser/EditorHeader.tsx
    - frontend/src/components/FileBrowser/SaveIndicator.tsx
    - frontend/src/components/FileBrowser/modals/UnsavedChangesModal.tsx
    - frontend/src/components/FileBrowser/modals/ConflictModal.tsx
tech_stack:
  added: []
  patterns:
    - If-Match echo contract: readFileText returns etag; client echoes verbatim as If-Match
    - useFilesWrite three-state save: idle/saving/saved (~1.5s SAVED_TIMEOUT transient)
    - 412 conflict: isConflict=true from hook; ConflictModal driven by this state;
      buffer NEVER cleared (T-125-08 locked decision)
    - guardThen() navigation guard: all 3 triggers (file-switch/navigate-up/tab-close)
      route through one helper; NO beforeunload (Wails blocks it, EDIT-07)
    - QuitConfirmModal pattern for modals: overlay/Escape/safe-default-focus/acting-guard
    - Colorblind contract: every save indicator state carries icon+text; color is decoration
key_files:
  created:
    - frontend/src/lib/useFilesWrite.ts
    - frontend/src/components/FileBrowser/EditorHeader.tsx
    - frontend/src/components/FileBrowser/SaveIndicator.tsx
    - frontend/src/components/FileBrowser/modals/UnsavedChangesModal.tsx
    - frontend/src/components/FileBrowser/modals/ConflictModal.tsx
    - frontend/src/components/__tests__/Editor.save.test.tsx
    - frontend/src/components/__tests__/EditorComponents.test.tsx
  modified:
    - frontend/src/lib/filesApi.ts
    - frontend/src/components/FileBrowserTab.tsx
decisions:
  - "useFilesWrite separates isSaving (boolean, for button disable) from saveState (SaveState enum, for indicator) — finer-grained than the plan's isSaving-only shape, enables both SaveIndicator and button disable without extra prop threading"
  - "guardThen() is declared in FileBrowserTab, not in Editor — orchestrator owns navigation; Editor just fires onSave/onCancel callbacks"
  - "ConflictModal default focus on 'Discard my changes' (safe choice, not Force overwrite) — matches QuitConfirmModal safe-default-focus invariant"
  - "Force overwrite sends If-Match='*' per server contract (PATTERNS §Architecture Pattern 3: wildcard = skip-check)"
  - "Discard my changes re-fetches readFileText and replaces buffer — ensures buffer reflects actual server state (not a stale copy)"
  - "ETag captured from readFileText in the preview effect and stored in editEtag state — threads through to handleSave without re-fetch"
requirements_completed: [EDIT-05, EDIT-06, EDIT-07, EDIT-08]
metrics:
  duration: "~25 minutes"
  completed: "2026-06-14T21:50:00Z"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 9
  commits: 4
---

# Phase 125 Plan 03: Save Flow + Dirty State + Unsaved Guard + 412 Conflict Modal Summary

**Cmd/Ctrl+S atomic save with If-Match ETag echo, three-state indicator (ArrowPathIcon/CheckCircleIcon + text), dirty ● marker, guardThen() React-level navigation guard (NO beforeunload), and 412 ConflictModal with Force overwrite/Save as new/Discard — buffer never silently discarded.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-06-14T21:25:00Z
- **Completed:** 2026-06-14T21:50:00Z
- **Tasks:** 2 (TDD: 2 RED + 2 GREEN cycles)
- **Files modified:** 9

## Accomplishments

- `filesApi.ts` gains `isConflict()` (412), `isCollision()` (409) predicates and `writeFile()` with `If-Match` header — the single client-side write method
- `useFilesWrite.ts` implements the three-state save machine (idle/saving/saved ~1.5s), 412 conflict signaling (buffer NEVER cleared — T-125-08), inline save error copy
- `SaveIndicator.tsx` + `EditorHeader.tsx` deliver the colorblind-safe indicators: every state carries both icon glyph AND literal text; dirty `●` bullet has `aria-label="Modified"`
- `UnsavedChangesModal.tsx` + `ConflictModal.tsx` implement the QuitConfirmModal pattern with verbatim UI-SPEC copy, safe-button default focus, Escape-closes, acting guard
- `FileBrowserTab.tsx` wired: `guardThen()` helper routes all 3 navigation triggers, ETag captured from preview fetch, edit mode state machine, `ConflictModal` driven by `useFilesWrite.isConflict`

## Task Commits

| # | Phase | Description | Commit | Type |
|---|-------|-------------|--------|------|
| 1 (RED) | Task 1 | Failing tests: save wiring, filesApi predicates, useFilesWrite | a342d97 | test |
| 1 (GREEN) | Task 1 | useFilesWrite + filesApi writeFile + isConflict/isCollision | 262cd90 | feat |
| 2 (RED) | Task 2 | Failing tests: EditorHeader/SaveIndicator/modals/wiring | 77900b7 | test |
| 2 (GREEN) | Task 2 | EditorHeader + SaveIndicator + modals + FileBrowserTab wiring | 2be2b32 | feat |

## Files Created/Modified

**Created:**
- `frontend/src/lib/useFilesWrite.ts` — hook: write/del/rename/mkdir/upload + isSaving/saveState/saveError/isConflict
- `frontend/src/components/FileBrowser/EditorHeader.tsx` — header with dirty marker + SaveIndicator + Save/Cancel
- `frontend/src/components/FileBrowser/SaveIndicator.tsx` — role=status aria-live three-state indicator
- `frontend/src/components/FileBrowser/modals/UnsavedChangesModal.tsx` — EDIT-07 guard modal
- `frontend/src/components/FileBrowser/modals/ConflictModal.tsx` — EDIT-08 conflict modal
- `frontend/src/components/__tests__/Editor.save.test.tsx` — source-inspection tests for save wiring
- `frontend/src/components/__tests__/EditorComponents.test.tsx` — source-inspection tests for components

**Modified:**
- `frontend/src/lib/filesApi.ts` — added isConflict/isCollision/isMissingFilesWritePerm predicates + writeFile method
- `frontend/src/components/FileBrowserTab.tsx` — added guardThen, editor state, edit mode render, modal wiring

## Decisions Made

- `useFilesWrite` separates `isSaving` (bool, for button disable) from `saveState` (enum, for indicator) — enables both SaveIndicator rendering and button disable without extra prop threading
- `guardThen()` lives in FileBrowserTab (orchestrator owns navigation); Editor fires `onSave`/`onCancel` callbacks
- `ConflictModal` default focus on `Discard my changes` (safe choice), never on `Force overwrite`
- Force overwrite sends `If-Match: *` per server skip-check contract (PATTERNS §Architecture Pattern 3)
- Discard re-fetches `readFileText` to replace buffer — ensures buffer is exact server state
- ETag stored in `editEtag` state during preview fetch; threaded through to `handleSave` without re-fetch

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Missing merge — worktree behind 125-02 HEAD**
- **Found during:** Start of execution
- **Issue:** Worktree was at `d725107` (v3.4.2), missing all 125-01 and 125-02 commits
- **Fix:** `git merge 31509d9 --no-edit` to fast-forward to the 125-02 tracking commit
- **Commit:** (pre-task merge, not a separate commit)

**2. [Rule 2 - Missing Critical] Added `isMissingFilesWritePerm()` to `FilesApiError`**
- **Found during:** Task 1 implementation
- **Issue:** The hook pattern in `useFilesCapability.ts` used this predicate (already in local scope) but `FilesApiError` didn't expose it as a method for external callers
- **Fix:** Added `isMissingFilesWritePerm()` to `FilesApiError` class alongside `isConflict()`/`isCollision()` — completes the predicate set

**3. [Rule 1 - Bug] `beforeunload` test was too broad (matched comment text)**
- **Found during:** Task 2 verification
- **Issue:** `expect(fileBrowserTabRaw).not.toContain('beforeunload')` failed because the file had a comment "NO beforeunload — Wails blocks it"
- **Fix:** Narrowed the test to `not.toContain("addEventListener('beforeunload'")` — checks for functional use, not documentation strings

---

**Total deviations:** 3 (1 pre-task merge, 1 Rule 2 enhancement, 1 Rule 1 test fix)
**Impact on plan:** No scope creep. All deviations necessary for correctness or test accuracy.

## Threat Surface Scan

No new network endpoints or auth paths introduced. The `writeFile` method sends to the existing `PUT /api/files/write` route (already guarded by `requireFilesWrite` + CSRF check, shipped Phase 124). The `If-Match` header is a standard precondition request header — no new trust boundary.

T-125-07 (force-overwrite clobber): mitigated — `Force overwrite` is reachable only via explicit `ConflictModal` user choice; default focus is `Discard my changes`.
T-125-08 (buffer loss on 412/error): mitigated — buffer NEVER cleared; 412 sets `isConflict=true`; errors set `saveError` inline.
T-125-09 (line-ending corruption): mitigated — raw CM6 buffer sent as octet-stream; no CRLF→LF re-encode.

## Self-Check

| Claim | Status |
|-------|--------|
| `frontend/src/lib/useFilesWrite.ts` exists | FOUND |
| `frontend/src/components/FileBrowser/SaveIndicator.tsx` exists | FOUND |
| `frontend/src/components/FileBrowser/modals/ConflictModal.tsx` exists | FOUND |
| `frontend/src/components/FileBrowser/modals/UnsavedChangesModal.tsx` exists | FOUND |
| Commit a342d97 (RED Task 1) exists | FOUND |
| Commit 262cd90 (GREEN Task 1) exists | FOUND |
| Commit 77900b7 (RED Task 2) exists | FOUND |
| Commit 2be2b32 (GREEN Task 2) exists | FOUND |
| pnpm test: 1205 tests pass | PASS |
| tsc --noEmit: clean | PASS |
| `grep -c "isConflict" filesApi.ts` >= 1 | PASS (1) |
| `grep -c "If-Match" useFilesWrite.ts/filesApi.ts` non-empty | PASS (3/4) |
| `grep -c "Mod-s" Editor.tsx` >= 1 | PASS (1) |
| `grep -c "aria-live" SaveIndicator.tsx` >= 1 | PASS (2) |
| `grep -c "Force overwrite" ConflictModal.tsx` >= 1 | PASS (6) |
| `grep -c "Keep editing" UnsavedChangesModal.tsx` >= 1 | PASS (7) |
| No `addEventListener('beforeunload')` in FileBrowserTab/Editor | PASS (0) |

## Self-Check: PASSED
