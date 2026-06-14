---
phase: 125-react-editor-codemirror-6-desktop-web
plan: 04
subsystem: frontend-editor
tags: [write-affordances, file-browser, modals, inline-input, canWrite, tdd, edit-09, edit-12]
dependency_graph:
  requires:
    - 125-03-SUMMARY.md  # useFilesWrite hook stubs + isCollision() predicate
    - 125-02-SUMMARY.md  # canWrite from useFilesCapability; Editor.tsx CM6 mount
  provides:
    - filesApi.ts: del() DELETE, rename() POST {oldRel,newRel}, mkdir() POST
    - useFilesWrite.ts: real del/rename/mkdir implementations (Plan 03 stubs replaced)
    - InlineNameInput.tsx: inline create-file/mkdir/rename input (Enter commits, Escape cancels)
    - FileRow.tsx: FileRowActions cluster (Edit/Rename/Move/Delete) gated on canWrite
    - BreadcrumbBar.tsx: New file (DocumentPlusIcon) + New folder (FolderPlusIcon) toolbar buttons
    - DeleteConfirmModal.tsx: file + recursive-dir variants; {N} files inside count; Cancel default focus
    - CollisionConfirmModal.tsx: 409 replace modal; Cancel DEFAULT focus (locked EDIT-09/10)
    - MoveToPickerModal.tsx: dirs-only tree picker; Move here primary; rename cross-dir move
    - FileBrowserTab.tsx: all write affordances wired; F2 rename; countFilesRecursive; modal wiring
    - FileListPane.tsx: optional canWrite + onRowEdit/Rename/Move/Delete props; F2 key handler
  affects:
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesWrite.ts
    - frontend/src/components/FileBrowserTab.tsx
    - frontend/src/components/FileBrowser/FileRow.tsx
    - frontend/src/components/FileBrowser/FileListPane.tsx
    - frontend/src/components/FileBrowser/BreadcrumbBar.tsx
    - frontend/src/components/FileBrowser/InlineNameInput.tsx
    - frontend/src/components/FileBrowser/modals/DeleteConfirmModal.tsx
    - frontend/src/components/FileBrowser/modals/CollisionConfirmModal.tsx
    - frontend/src/components/FileBrowser/modals/MoveToPickerModal.tsx
tech_stack:
  added: []
  patterns:
    - QuitConfirmModal pattern: overlay/Escape/safe-default-focus/acting-guard applied to all 3 new modals
    - cancelBtnRef focus on Cancel (safe action) — never on Delete/Replace (destructive)
    - countFilesRecursive: client-side listFiles walk for dir delete count (avoids server change)
    - cross-dir move = rename: rename(oldRel, newRel) with different parent paths (EDIT-09)
    - 409 collision: CollisionConfirmModal with Cancel DEFAULT focus (EDIT-09/10 locked)
    - canWrite gate: all write affordances (buttons, inline input, row actions) rendered only when canWrite=true
key_files:
  created:
    - frontend/src/components/FileBrowser/InlineNameInput.tsx
    - frontend/src/components/FileBrowser/modals/DeleteConfirmModal.tsx
    - frontend/src/components/FileBrowser/modals/CollisionConfirmModal.tsx
    - frontend/src/components/FileBrowser/modals/MoveToPickerModal.tsx
    - frontend/src/components/FileBrowser/__tests__/FileRowActions.test.tsx
  modified:
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesWrite.ts
    - frontend/src/components/FileBrowserTab.tsx
    - frontend/src/components/FileBrowser/FileRow.tsx
    - frontend/src/components/FileBrowser/FileListPane.tsx
    - frontend/src/components/FileBrowser/BreadcrumbBar.tsx
decisions:
  - "FileRowActions threaded through FileListPane (optional props) rather than rendered directly in FileBrowserTab — preserves the list component's ownership of row rendering; new props are optional so existing callers/tests compile without change"
  - "Inline rename input rendered below FileListPane (not replacing the row in-place) — simplest approach; avoids re-architecting the list virtualization; UX is close enough to in-place for plan scope"
  - "countFilesRecursive is async walk on client side (RESEARCH Open Q3 resolved) — no server change needed; walk errors silently return 0 (conservative display)"
  - "Cross-dir move = rename(oldRel, newRel) — server Rename validates both paths (T-125-10); no copy+delete semantics anywhere in the UI"
  - "CollisionConfirmModal Cancel DEFAULT focus is enforced by cancelBtnRef.current?.focus() on open — verified at source level (not by eye)"
  - "Destructive button color #f7768e is inline style on Delete/Replace buttons; these buttons never hold cancelBtnRef; verified at source"
requirements_completed: [EDIT-09, EDIT-12]
metrics:
  duration: "~40 minutes"
  completed: "2026-06-14T23:10:00Z"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 11
  commits: 3
---

# Phase 125 Plan 04: Directory Write Affordances Summary

**del/rename/mkdir API methods + InlineNameInput + FileRowActions + BreadcrumbBar toolbar + DeleteConfirmModal (file + recursive-dir with count) + CollisionConfirmModal (Cancel-default 409) + MoveToPickerModal (rename cross-dir) wired into FileBrowserTab with F2 rename — all gated on canWrite.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-06-14T22:30:00Z
- **Completed:** 2026-06-14T23:10:00Z
- **Tasks:** 2 (TDD: 1 RED + 2 GREEN cycles)
- **Files modified:** 11

## Accomplishments

- `filesApi.ts` gains `del()` (DELETE), `rename()` (POST `{oldRel, newRel}`), `mkdir()` (POST) — all using `buildQuery` + `fetchOrThrow`; throw `FilesApiError(409)` on collision
- `useFilesWrite.ts` Plan 03 stubs replaced with real `client.del/rename/mkdir` calls
- `InlineNameInput.tsx` — Enter commits (trimmed, non-empty), Escape cancels, focus+select on mount; `Filename…`/`Folder name…` placeholders verbatim from UI-SPEC
- `FileRow.tsx` — `FileRowActions` cluster (Edit/Rename/Move/Delete icon buttons) revealed on `:hover`/`:focus-within`, gated `canWrite`; Edit additionally gated on `!isDir && !isBinary`; all props optional — existing callers unaffected
- `BreadcrumbBar.tsx` — `DocumentPlusIcon` New file + `FolderPlusIcon` New folder toolbar buttons before refresh, `canWrite` only
- `DeleteConfirmModal.tsx` — file + recursive-dir variants; dir body: `Delete "{name}" and all {N} files inside? This cannot be undone.`; `TrashIcon`/`ExclamationTriangleIcon` glyphs; `Cancel` default-focused (`cancelBtnRef`); `Delete` destructive (#f7768e) never default-focused
- `CollisionConfirmModal.tsx` — `A file named "{name}" already exists. Replace it?`; `Cancel` DEFAULT-focused (EDIT-09/10 locked); `Replace` destructive never default-focused
- `MoveToPickerModal.tsx` — lazy-loading dirs-only tree; `Move here` primary disabled until target selected; `rename(oldRel, newRel)` cross-dir call; `Cancel` + Escape
- `FileBrowserTab.tsx` — `del/rename/mkdir` destructured; `countFilesRecursive` async walk; all handlers wired (`handleDeleteRequest/Confirm`, `handleRenameRequest`, `handleMoveRequest/Confirm`, `handleNewFile/Folder`, `handleInlineCommit`); `BreadcrumbBar` + `FileListPane` wired with `canWrite` + action callbacks; all 3 new modals rendered at tab root; 409 → `CollisionConfirmModal` from inline input and move ops
- `FileListPane.tsx` — optional `canWrite` + `onRowEdit/Rename/Move/Delete` props threaded to `FileRow`; `F2` key triggers `onRowRename` on selected row

## Task Commits

| # | Phase | Description | Commit | Type |
|---|-------|-------------|--------|------|
| RED | Tasks 1+2 | Failing tests: all write affordances | b6894a5 | test |
| GREEN | Task 1 | filesApi del/rename/mkdir + useFilesWrite + InlineNameInput + FileRowActions + toolbar + modals | f02f03a | feat |
| GREEN | Task 2 | Wire FileBrowserTab + FileListPane with write affordances + F2 rename | 9fadec8 | feat |

## Decisions Made

- `FileRowActions` threaded through `FileListPane` via optional props — preserves list component's row-rendering ownership; existing callers/tests compile without change
- Inline rename input rendered below `FileListPane` (not replacing the row in-place) — simplest approach; avoids re-architecting list virtualization
- `countFilesRecursive` is an async client-side `listFiles` walk — no server change needed (RESEARCH Open Q3 resolved); errors silently return 0
- Cross-dir move = `rename(oldRel, newRel)` — server validates both paths (T-125-10); no copy+delete anywhere
- `CollisionConfirmModal` `Cancel` DEFAULT focus enforced by `cancelBtnRef.current?.focus()` on open — verified at source
- Destructive button `#f7768e` inline style; these buttons never hold `cancelBtnRef` — verified at source

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 1 RED tests needed modal files to load**
- **Found during:** Task 1 RED test run
- **Issue:** `FileRowActions.test.tsx` uses dynamic `import()` for all 3 modal files; vite fails to resolve non-existent files during test suite initialization (all-or-nothing module analysis), making even Task 1 tests fail before running
- **Fix:** Created all 3 modal implementations alongside Task 1 GREEN (combined into one feat commit). The modal tests (Task 2 targets) already passed since modals were complete.
- **Impact:** No scope deviation — modals were the Task 2 deliverable anyway; they were just created earlier than planned

None - plan executed with one deviation needed for test infrastructure.

## Threat Surface Scan

No new network endpoints. New client-side write methods (`del`, `rename`, `mkdir`) hit existing server routes (`DELETE /api/files/delete`, `POST /api/files/rename`, `POST /api/files/mkdir`) that were already guarded by `requireFilesWrite` + CSRF check (Phase 123/124).

T-125-10 (rename/move destination-path traversal): mitigated — server `Rename` validates BOTH paths via `validateAndClean` (shipped Phase 123, FSW-02); UI is advisory only.
T-125-11 (recursive delete escaping sandbox): mitigated — server recursive delete walk stays within `os.Root` (shipped Phase 123, FSW-04); client count-walk is read-only `listFiles`.
T-125-12 (accidental destructive delete/replace): mitigated — every destructive op behind a modal with Cancel default-focused; recursive-dir delete states the file count before confirming; verified at source level.

## Self-Check

| Claim | Status |
|-------|--------|
| `frontend/src/components/FileBrowser/InlineNameInput.tsx` exists | FOUND |
| `frontend/src/components/FileBrowser/modals/DeleteConfirmModal.tsx` exists | FOUND |
| `frontend/src/components/FileBrowser/modals/CollisionConfirmModal.tsx` exists | FOUND |
| `frontend/src/components/FileBrowser/modals/MoveToPickerModal.tsx` exists | FOUND |
| Commit b6894a5 (RED) exists | FOUND |
| Commit f02f03a (GREEN Task 1) exists | FOUND |
| Commit 9fadec8 (GREEN Task 2) exists | FOUND |
| `pnpm test -- --run FileRowActions`: 36 pass | PASS |
| `pnpm test -- --run` all: 1241 pass | PASS |
| `pnpm exec tsc --noEmit`: clean | PASS |
| `grep -c "files inside" DeleteConfirmModal.tsx` >= 1 | PASS (4) |
| `grep -c "already exists" CollisionConfirmModal.tsx` >= 1 | PASS (5) |
| `grep -c "Move here" MoveToPickerModal.tsx` >= 1 | PASS (6) |
| `grep -c "canWrite" FileRow.tsx` >= 1 | PASS (7) |
| `grep -c "oldRel" filesApi.ts` >= 1 | PASS (3) |
| `grep -c "Filename…" InlineNameInput.tsx` >= 1 | PASS (2) |
| Cancel default-focused (cancelBtnRef) in Delete + Collision modals | PASS |
| Destructive buttons (#f7768e) NOT default-focused | PASS |
| Cross-dir move uses rename (not copy+delete) | PASS |
| No modifications to STATE.md or ROADMAP.md | PASS |

## Self-Check: PASSED
