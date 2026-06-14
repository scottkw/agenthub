---
phase: 125-react-editor-codemirror-6-desktop-web
plan: 05
subsystem: frontend-upload
tags: [upload, xhr, progress, drag-drop, file-browser, canWrite, tdd, edit-10, edit-12]
dependency_graph:
  requires:
    - 125-04-SUMMARY.md  # CollisionConfirmModal, BreadcrumbBar toolbar, canWrite gates
    - 125-03-SUMMARY.md  # useFilesWrite hook shape, filesApi client
  provides:
    - filesApi.ts: uploadFile(sid, dir, file, onProgress) — XHR, FormData(file+dir), 409/413 errors
    - useFilesWrite.ts: upload real implementation (Plan-03 stub replaced)
    - UploadQueuePanel.tsx: per-file N% progress, done/failed/over-cap/collision states
    - UploadDropOverlay.tsx: drag-and-drop target overlay over the list container
    - BreadcrumbBar.tsx: ArrowUpTrayIcon Upload button (canWrite only, triggers hidden input)
    - FileBrowserTab.tsx: upload queue state, drag/drop handlers, hidden file input wiring
  affects:
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesWrite.ts
    - frontend/src/components/FileBrowser/UploadQueuePanel.tsx
    - frontend/src/components/FileBrowser/UploadDropOverlay.tsx
    - frontend/src/components/FileBrowser/BreadcrumbBar.tsx
    - frontend/src/components/FileBrowserTab.tsx
    - frontend/src/components/FileBrowser/__tests__/Upload.test.tsx
tech_stack:
  added: []
  patterns:
    - XMLHttpRequest for upload (not fetch) — upload.onprogress provides per-file N% events
    - FormData with one 'file' part + 'dir' field matching Handler.Upload server contract
    - cap token via URL query param (buildQuery pattern, same as other filesApi methods)
    - UploadQueuePanel uses .new-session-modal* chrome (UI-SPEC §Design System)
    - Colorblind contract: CheckCircleIcon+Done, ExclamationTriangleIcon+Failed — icon+text, never color alone
    - role=status aria-live=polite for upload progress announcements
    - Drag-over list container shows UploadDropOverlay; drop enqueues files
    - Hidden <input type=file multiple> triggered by Upload toolbar button
    - Per-file 409: inline Replace? prompt does not block other queue rows
    - Pre-flight 50 MiB check skips over-cap files before XHR send
key_files:
  created:
    - frontend/src/components/FileBrowser/UploadQueuePanel.tsx
    - frontend/src/components/FileBrowser/UploadDropOverlay.tsx
    - frontend/src/components/FileBrowser/__tests__/Upload.test.tsx
  modified:
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesWrite.ts
    - frontend/src/components/FileBrowser/BreadcrumbBar.tsx
    - frontend/src/components/FileBrowserTab.tsx
decisions:
  - "uploadFile() uses XMLHttpRequest (not fetchOrThrow) — fetch has no upload-progress API; XHR.upload.onprogress is the only per-file N% mechanism (PATTERNS §Upload exception, RESEARCH Pitfall 6)"
  - "Cap token in URL query param (not Authorization header) — consistent with all other filesApi methods; Cap rides in the query string via the same buildQuery pattern"
  - "Pre-flight 50 MiB size check on the client side before XHR — avoids sending bytes only to get a 413; server's MaxBytesReader is still the authoritative enforcement (T-125-14)"
  - "Per-file 409 collision shown as inline Replace? prompt in the queue row — does not block other files; each file proceeds independently (EDIT-10 requirement)"
  - "UploadQueuePanel uses .new-session-modal* chrome — reuses existing CSS family per UI-SPEC; no new component family"
  - "Drag-over overlay is position: absolute inside position: relative list container — scoped to listing, not full tab"
  - "Hidden file input onChange clears value after enqueue — allows re-uploading the same filename"
  - "handleUploadFiles respects canWrite gate before enqueueing — affordances are fully gated at both UI and handler level"
metrics:
  duration: "~35 minutes"
  completed: "2026-06-14T20:32:00Z"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 7
  commits: 3
---

# Phase 125 Plan 05: Upload — XHR Progress + Drag-Drop + 409/413 Handling Summary

**Single + multi-file upload via hidden file input and drag-and-drop, with per-file XHR progress queue (N%), Done/Failed states carrying icon+text (colorblind contract), 409 per-row Replace? prompt (does not block other files), and 413 over-cap skip message — all gated on canWrite.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-06-14T19:57:00Z
- **Completed:** 2026-06-14T20:32:00Z
- **Tasks:** 2 (TDD: RED + GREEN)
- **Files modified:** 7

## Accomplishments

- `filesApi.ts` gains `uploadFile(sid, dir, file, onProgress)`: XMLHttpRequest (not fetch), `FormData` with one `'file'` part + `'dir'` field matching server `Handler.Upload`; `xhr.upload.onprogress` reports integer 0-100% to callback; `load` event maps 409 → `FilesApiError.isCollision()`, 413 → `isOverCap()`; cap token rides in URL query param
- `useFilesWrite.ts` Plan-03 upload stub replaced with real `client.uploadFile()` call threading `sessionId`, `dir`, `file`, `onProgress`
- `UploadQueuePanel.tsx` — `.new-session-modal*` chrome; per-file row with determinate N% progress bar + N% text; `CheckCircleIcon + "Done"` on success; `ExclamationTriangleIcon + "Failed — try again"` on failure; `ExclamationTriangleIcon + '"{name}" is too large (max 50 MB) and was skipped.'` on over-cap; inline `Replace` button on 409 collision (does not block other rows); `role="status" aria-live="polite"` for progress announcements
- `UploadDropOverlay.tsx` — absolutely positioned overlay over the list container; dashed `#7aa2f7` border; `ArrowUpTrayIcon` + verbatim `"Drop files to upload here"` copy; `data-testid="upload-drop-overlay"` for tests; `onDrop` extracts `FileList` → `File[]`
- `BreadcrumbBar.tsx` — `ArrowUpTrayIcon` Upload icon button added before refresh; gated on `canWrite && onUpload`; `aria-label="Upload"`; new `onUpload?` prop added to `BreadcrumbBarProps`
- `FileBrowserTab.tsx` — `upload` destructured from `useFilesWrite`; `uploadQueue` state (array of `UploadQueueItem`); `isDragOver` state; `uploadInputRef` (hidden `<input type=file multiple>`); `handleUploadFiles` queue loop with pre-flight 50 MiB check, sequential per-file XHR, live progress updates, collision/over-cap/failed state routing; `handleUploadReplace` re-uploads collision items; `handleDragOver/Leave/Drop` handlers on list container; `UploadDropOverlay` rendered inside list container when `isDragOver && canWrite`; `UploadQueuePanel` rendered at tab root when queue non-empty; `BreadcrumbBar` receives `onUpload` callback

## Task Commits

| # | Phase | Description | Commit | Type |
|---|-------|-------------|--------|------|
| RED | Tasks 1+2 | Failing tests: Upload XHR source + functional + UploadQueuePanel/DropOverlay/BreadcrumbBar | 2738eb7 | test |
| GREEN | Tasks 1+2 | filesApi.uploadFile XHR + useFilesWrite real upload + all UI components + FileBrowserTab wiring | 1d7381d | feat |

## Decisions Made

- `uploadFile()` uses XMLHttpRequest (not `fetchOrThrow`) — fetch has no upload-progress API; XHR is the only mechanism for per-file N% (PATTERNS §Upload exception, RESEARCH Pitfall 6)
- Cap token in URL query param — consistent with all filesApi methods; no Authorization header approach needed
- Pre-flight 50 MiB client-side check before XHR — avoids sending bytes that will be rejected; server's `MaxBytesReader` is still the real enforcement (T-125-14)
- Per-file 409 shown as inline Replace? prompt in the queue row — does not block other rows (EDIT-10 requirement verbatim)
- `UploadQueuePanel` uses `.new-session-modal*` chrome (UI-SPEC §Design System) — zero new CSS families
- Drag-over overlay is `position: absolute` inside `position: relative` list container — scoped to listing, not full tab

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] XHR mock needed instance reference to mutate status before `load` event**
- **Found during:** Task 1 functional tests (RED phase)
- **Issue:** The XHR listener closure captures `xhr` (the mock instance). Calling `loadFn.call({ status: 409, ... })` doesn't affect `xhr.status` in the closure — the mock's `status` was always 200. Tests for 409/413 error paths returned `null` instead of `FilesApiError`.
- **Fix:** Redesigned the XHR mock to store `this` reference in `xhrInstanceRef` during construction, so tests can mutate `xhrInstanceRef['status']` before calling `fireLoad()`, which then calls the listener with the mutated instance as context.
- **Files modified:** `Upload.test.tsx`
- **Impact:** Tests now correctly validate error-path behavior; no change to production code

**2. [Rule 1 - Bug] TypeScript: `FilesApiError` used as type from dynamic import**
- **Found during:** Task 2 tsc check
- **Issue:** `err as FilesApiError` inside a dynamically-imported block caused TS2749 — class from dynamic import can't be used directly as a type annotation.
- **Fix:** Changed to `err as InstanceType<typeof mod.FilesApiError>` after importing the module as `mod`.
- **Files modified:** `Upload.test.tsx`
- **Commit:** included in GREEN commit `1d7381d`

## Threat Surface Scan

No new network endpoints introduced. `uploadFile()` POSTs to the existing `/api/files/upload` route (Handler.Upload, shipped Phase 123 FSW-05/12). Security properties are inherited:
- T-125-13 (filename injection `../`): mitigated by server `filepath.Base + validateAndClean` (Phase 123) — client cannot bypass
- T-125-14 (over-cap >50 MiB): client adds pre-flight size check; server's `MaxBytesReader` is the authoritative enforcement; client surfaces the skip copy

## Self-Check

| Claim | Status |
|-------|--------|
| `frontend/src/components/FileBrowser/UploadQueuePanel.tsx` exists | FOUND |
| `frontend/src/components/FileBrowser/UploadDropOverlay.tsx` exists | FOUND |
| `frontend/src/components/FileBrowser/__tests__/Upload.test.tsx` exists | FOUND |
| Commit 2738eb7 (RED) exists | FOUND |
| Commit 1d7381d (GREEN) exists | FOUND |
| `pnpm test -- --run Upload`: 32 pass | PASS |
| `pnpm test -- --run` all: 1273 pass | PASS |
| `pnpm exec tsc --noEmit`: clean | PASS |
| `grep -c "XMLHttpRequest" filesApi.ts` >= 1 | PASS (2) |
| `grep -c "onprogress" filesApi.ts` >= 1 | PASS (1) |
| `grep -c "FormData" filesApi.ts` >= 1 | PASS (1) |
| `grep -c "Drop files to upload here" UploadDropOverlay.tsx` >= 1 | PASS (2) |
| `grep -c "Failed — try again" UploadQueuePanel.tsx` >= 1 | PASS (3) |
| `grep -c "is too large (max 50 MB)" UploadQueuePanel.tsx` >= 1 | PASS (1) |
| Upload gated on canWrite (BreadcrumbBar + FileBrowserTab) | PASS |
| Drag-drop gated on canWrite (handleDragOver guard) | PASS |
| Per-file 409 inline Replace? does not block other rows | PASS |
| Colorblind contract: CheckCircleIcon+Done, ExclamationTriangleIcon+Failed | PASS |
| role=status aria-live=polite in UploadQueuePanel | PASS |
| No modifications to STATE.md or ROADMAP.md | PASS |

## Self-Check: PASSED
