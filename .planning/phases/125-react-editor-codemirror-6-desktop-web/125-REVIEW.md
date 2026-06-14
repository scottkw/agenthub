---
phase: 125-react-editor-codemirror-6-desktop-web
reviewed: 2026-06-14T00:00:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - internal/files/write.go
  - internal/files/handler.go
  - frontend/src/lib/filesApi.ts
  - frontend/src/lib/useFilesWrite.ts
  - frontend/src/lib/useFilesCapability.ts
  - frontend/src/lib/languageFor.ts
  - frontend/src/components/Editor.tsx
  - frontend/src/components/FileBrowserTab.tsx
  - frontend/src/components/FileBrowser/modals/ConflictModal.tsx
  - frontend/src/components/FileBrowser/UploadQueuePanel.tsx
findings:
  critical: 3
  warning: 7
  info: 5
  total: 15
status: resolved
resolution: fixed
resolution_note: >
  All 3 critical + 7 warning + IN-01/IN-04/IN-05 fixed. CR-01 TOCTOU narrowed via
  re-stat-before-rename in WriteFileAtomic (ErrPreconditionFailed→412); CR-02 HEAD
  /api/files/write registered through requireFilesWrite so the canWrite probe gates
  correctly; CR-03 save-as-new re-points editor + refreshes etag + disambiguates -copy;
  WR-07 upload now 409s on collision without overwrite=1 (Replace sends overwrite=1).
  Commits 890d811, 02201be, 7228900. go test -race green (files+webserver), frontend
  1273 tests + tsc + build clean. IN-02/IN-03 skipped (cosmetic, out of fix scope).
---

# Phase 125: Code Review Report

**Reviewed:** 2026-06-14
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

Reviewed the milestone-centerpiece editor + write surface (CodeMirror 6 editor, optimistic-concurrency write path, upload XHR flow, conflict resolution UI, and per-surface canWrite gating). Cross-referenced the Go sandbox (`sandbox.go`), the webserver capability middleware (`capability_mw.go`), route registration (`server.go`), and the write tests.

Positives confirmed during review: the If-Match validator format is consistent across emission (`handler.go:297`), comparison (`write.go:65`), and tests (`write_test.go:637`) — no RFC3339-vs-UnixNano drift. The editor renders content as TEXT via CodeMirror's DOM layer with **no** `dangerouslySetInnerHTML`/`innerHTML` anywhere — the stored-XSS gate holds. The 412 conflict buffer is never silently discarded. The server enforces the real write gate via `requireFilesWrite` (cap + HasPerm + CSRF Origin), so the advisory UI gate cannot be exploited to bypass authorization at the server.

However, three Critical defects exist: a genuine TOCTOU window in the optimistic-concurrency check, a broken web-share `canWrite` probe that fails-open to `true` because the route does not accept HEAD, and a "Save as new file" conflict path that re-introduces the same conflict it was meant to escape. Several Warnings concern the upload-refresh race, the conflict-flag race in the save state machine, and stale-closure issues in the upload queue.

## Critical Issues

### CR-01: TOCTOU race between Stat and WriteFileAtomic in optimistic-concurrency check

**File:** `internal/files/write.go:63-89`
**Issue:** The If-Match precondition reads the file's validator via `sb.Stat(rel)` (line 64), compares it (line 66), and only later calls `sb.WriteFileAtomic(rel, data)` (line 86). Between the `Stat` and the rename inside `WriteFileAtomic`, another process can modify the file. The whole point of EDIT-08 optimistic concurrency is to reject a write when the on-disk state changed since the client last read it — but a concurrent writer that lands *after* the `Stat` check passes is silently clobbered. This is the exact data-loss class the feature exists to prevent. `WriteFileAtomic` uses a rename, which is atomic for crash-durability, but it does **not** re-validate the destination's mtime/size under the rename, so the check-then-act is not atomic with respect to the validator.

This is a real (if narrow) window on the most-exposed write surface. Two web-share clients editing the same file can both pass the `Stat` check and the second write wins with no 412.

**Fix:** The validator check cannot be made fully atomic without OS-level support, but the window must be acknowledged and minimized, and the check should be pushed as close to the rename as possible. Preferred: thread the expected validator into `WriteFileAtomic` and re-stat the destination immediately before `root.Rename`, returning a sentinel (e.g. `ErrPreconditionFailed`) that `writeWriteError` maps to 412:
```go
// in WriteFileAtomic, just before writeAtomicRename:
if expectedValidator != "" {
    if fi, err := root.Stat(cleaned); err == nil {
        cur := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))
        if cur != expectedValidator {
            _ = root.Remove(tmp)
            return ErrPreconditionFailed
        }
    }
}
```
At minimum, document the residual window explicitly; the current comment claims the check is sound when it is not.

### CR-02: Web-share canWrite probe fails-open to `true` because the write route rejects HEAD

**File:** `frontend/src/lib/filesApi.ts:279-282`, `frontend/src/lib/useFilesCapability.ts:117-133`; route at `internal/webserver/server.go:512`
**Issue:** `probeWrite` issues `HEAD /api/files/write` (filesApi.ts:281). The route is registered only as `PUT /api/files/write` (server.go:512). Go's `http.ServeMux` returns **405 Method Not Allowed** for a HEAD against a PUT-only pattern — and critically, the 405 is produced by the mux *before* `requireFilesWrite` runs, so the response body never contains "files.write". In `useFilesCapability.ts:123-132`, the catch branch treats any error that is *not* `isMissingFilesWritePerm()` as `setCanWrite(true)` (line 131). A 405 is not a files.write-403, so the probe resolves `canWrite = true` for **every** web-share viewer, including viewers whose cap lacks `files.write`.

The comment at filesApi.ts:279-282 even anticipates "Throws FilesApiError(405) if HEAD is not supported — callers treat any non-403-files.write error as canWrite=true" — but that is precisely the bug: the probe can *never* return the 403-with-files.write signal because it never reaches the middleware. The probe is dead as a gate; it always returns 200-equivalent intent. A read-only web-share viewer will be shown write affordances (New file, Upload, Edit, Delete). The server's `requireFilesWrite` still blocks the actual PUT (so this is not a server-side authz bypass), but the UI gate is completely defeated, contradicting EDIT-12 and producing a confusing UX where every action 403s.

**Fix:** Either register a HEAD (or OPTIONS) handler for the write route that runs through `requireFilesWrite`, or change the probe to a method the route accepts. A correct, side-effect-free probe is a PUT to a sentinel path that fails path validation *after* the perm check — but the cleanest fix is a dedicated capability endpoint. Concretely, register HEAD on the write path so the middleware fires:
```go
mux.HandleFunc("HEAD /api/files/write", ws.requireFilesWrite(filesDispatch(func(h *files.Handler) http.HandlerFunc { return h.Write })))
```
and have `Handler.Write` short-circuit HEAD with 200 + no body before reading the request body. Then the 403-with-"files.write" body is reachable and the probe gates correctly.

### CR-03: "Save as new file" conflict path can re-trigger the same 412 / silently fail

**File:** `frontend/src/components/FileBrowserTab.tsx:1358-1372`
**Issue:** On a 412, `onSaveAsNew` derives `{base}-copy{ext}` and calls `writeFile(newPath, editContent)` with **no** If-Match (FileBrowserTab.tsx:1370). But `editEtag` state still holds the *original* file's ETag, and more importantly the derived name is computed only from `editingEntry.name` — `editingEntry` is unchanged, so after the save-as-new the editor is still pointed at the *original* file path. The next manual save (handleSave, line 962-977) writes to `joinPath(path, editingEntry.name)` with the stale `editEtag`, which will 412 again against the original file the user was trying to escape. The "save as new" does not re-point `editingEntry`, does not update `editEtag`, and does not select the new file — so the user's subsequent edits silently target the original conflicted file with a stale validator. Additionally, if `{base}-copy{ext}` itself already exists, the write has no If-Match and no collision handling here, so it force-overwrites an unrelated existing `-copy` file with no prompt (data loss).

**Fix:** After a successful save-as-new, re-point the editor at the new file and refresh its ETag, and guard the derived name against collision:
```ts
onSaveAsNew={() => {
  clearConflict()
  if (!editingEntry) return
  const ext = editingEntry.name.includes('.') ? '.' + editingEntry.name.split('.').pop()! : ''
  const base = ext ? editingEntry.name.slice(0, editingEntry.name.lastIndexOf('.')) : editingEntry.name
  const newName = `${base}-copy${ext}`
  const newPath = joinPath(path, newName)
  // collision check: if newName already exists, prompt or disambiguate (-copy-2, ...)
  void client.writeFile(sessionId, newPath, editContent).then(() => {
    refresh()
    setEditingEntry({ ...editingEntry, name: newName })
    // re-read to capture the new file's ETag
    void client.readFileText(sessionId, newPath).then((b) => setEditEtag(b.etag))
    setEditDirty(false)
  })
}}
```

## Warnings

### WR-01: Upload refresh logic is inverted — `didRefresh` never prevents and never reliably triggers refresh

**File:** `frontend/src/components/FileBrowserTab.tsx:785-847`
**Issue:** `didRefresh` is initialized `false` (line 785), set to `false` again on every successful upload (line 814, with a comment "mark that we need a refresh"), and the final guard is `if (!didRefresh) refresh()` (line 847). Because `didRefresh` is only ever `false`, `refresh()` always runs — even when *every* file was skipped (over-cap) or failed, in which case the comment's intent ("refresh after any successful upload") is violated (it refreshes even with zero successes). The variable is dead logic: it neither tracks successes nor prevents redundant refreshes. The name and comment describe behavior the code does not implement.

**Fix:** Track actual successes and refresh only when at least one occurred:
```ts
let didSucceed = false
// ... on the success branch:
didSucceed = true
// ... after the loop:
if (didSucceed) refresh()
```

### WR-02: Conflict-flag race — handleSave reads `isConflict` from a stale closure

**File:** `frontend/src/components/FileBrowserTab.tsx:962-977`, and consumers at 1324-1334
**Issue:** `handleSave` awaits `writeFile(...)` then checks `if (!isConflict)` (line 971) to decide whether to commit the new snapshot. But `isConflict` is captured from the render closure (it's in the dependency array, line 976), so the value read is the value *at the time handleSave was created*, not the value `writeFile` just set via `setIsConflict(true)`. React state updates are async; the `isConflict` in scope is stale. After a 412, `writeFile` sets `isConflict=true`, but the `handleSave` closure still sees the old `false` and proceeds to `setEditContent(body)` / `setEditDirty(false)` (lines 972-974) — clearing the dirty flag even though the save failed with a conflict. The same stale-read defeats the guard in `UnsavedChangesModal.onSave` (line 1325) which gates "proceed with deferred navigation" on `!isConflict`. This can navigate away from / clear the dirty state of a buffer whose save just conflicted.

**Fix:** Have `write()` return a discriminated result (e.g. `'saved' | 'conflict' | 'error'`) instead of relying on the async `isConflict` state being readable synchronously after `await`:
```ts
// useFilesWrite: write returns Promise<'saved' | 'conflict' | 'error'>
const outcome = await writeFile(filePath, body, editEtag)
if (outcome === 'saved') { setEditContent(body); setEditDirty(false) }
```

### WR-03: Sequential upload uses stale `uploadQueue.length` for start index

**File:** `frontend/src/components/FileBrowserTab.tsx:782-783, 849`
**Issue:** `startIdx = uploadQueue.length` (line 783) reads the queue length from the closure captured when `handleUploadFiles` was created (dependency `uploadQueue.length`, line 849). If a second batch of files is dropped/selected while a first batch is still in flight, both invocations capture the same `uploadQueue.length` and write to overlapping `queueIdx` ranges via `next[queueIdx] = ...`, corrupting/overwriting each other's progress rows. The `setUploadQueue((prev) => [...prev, ...newItems])` append is correct, but the index math that follows assumes `startIdx` equals the post-append base, which is only true for a single in-flight batch.

**Fix:** Derive the start index from inside the functional updater, or track items by a stable id rather than positional index. Simplest: capture the appended items' identities (e.g. a per-item `id`) and update by id lookup instead of `queueIdx`.

### WR-04: Save-as-new and force-overwrite use `editEtag`/`editContent` that may be stale after async state churn

**File:** `frontend/src/components/FileBrowserTab.tsx:1353-1357`
**Issue:** `onForceOverwrite` calls `writeFile(joinPath(path, editingEntry?.name ?? ''), editContent, '*')`. If `editingEntry` is null (race: modal open but entry cleared by a concurrent navigation/guard), `editingEntry?.name ?? ''` produces `joinPath(path, '')` which `joinPath` returns as `path` itself (a directory) when base is non-root, or `''` at root. Writing to a directory path will 500 server-side (`WriteFileAtomic` rename onto a dir), and writing to `''` produces a 400 "path is required". Neither is handled; the conflict modal closes (clearConflict ran) and the user gets a silent failure or an unhandled rejection.

**Fix:** Guard `editingEntry` before issuing the write, identical to the `onSaveAsNew` `if (editingEntry)` guard:
```ts
onForceOverwrite={() => {
  clearConflict()
  if (!editingEntry) return
  void writeFile(joinPath(path, editingEntry.name), editContent, '*')
}}
```

### WR-05: ConflictModal `acting` guard never resets when caller does not reopen, blocking retries

**File:** `frontend/src/components/FileBrowser/modals/ConflictModal.tsx:76-78, 117-153`
**Issue:** All three action buttons set `acting = true` (lines 119, 132, 145) which disables every button. `acting` is reset to `false` only in the effect keyed on `isOpen` becoming true (lines 76-78). The force-overwrite and save-as-new handlers in the parent re-issue a `writeFile` that can itself 412 again (see CR-03). On a re-412, the parent sets `isConflict` true again — but the modal was already mounted with `isOpen=true`; if `isConflict` was cleared then re-set within the same tick, the `isOpen` transition may not re-fire the reset effect reliably, leaving the buttons permanently disabled (`acting=true`) with no way to act. The modal can become a dead-end.

**Fix:** Reset `acting` whenever the modal is shown using a more robust trigger, and re-enable on the conflict re-firing. Reset `acting=false` in the same effect that opens the modal AND clear it if `onCancel`/Escape fires:
```ts
useEffect(() => { setActing(false) }, [isOpen])
// and in handleKeyDown / overlay click, call setActing(false) before onCancel()
```

### WR-06: Upload pre-check magic number duplicates server cap and can drift

**File:** `frontend/src/components/FileBrowserTab.tsx:791`
**Issue:** `const MAX_BYTES = 50 * 1024 * 1024` is hardcoded inline in the upload loop, duplicating the server's `maxUploadBytes = 50 << 20` (write.go:35). If the server cap changes, this client pre-check silently diverges, either rejecting files the server would accept or sending files the server will 413 (wasting a full upload). This is a correctness coupling between two surfaces (cross-surface parity is release-blocking per project memory).

**Fix:** Export the cap from a single shared constant (e.g. in `filesApi.ts` as `MAX_UPLOAD_BYTES`) and import it both here and wherever the limit is referenced. Add a server-side test asserting the value matches the documented milestone-locked 50 MiB.

### WR-07: `handleUploadReplace` re-uploads without overwrite semantics — guaranteed re-collision

**File:** `frontend/src/components/FileBrowserTab.tsx:852-892`
**Issue:** The "Replace" action on a 409 upload row simply calls `upload(path, item.file, ...)` again (line 870). The server's Upload handler routes through `WriteFileAtomic`, which *does* overwrite (rename-over). But the original 409 came from the server — meaning the upload path returns 409 on collision *somewhere* (the code's own error mapping and comment at lines 865-869 acknowledge "the server may 409 again if it requires explicit overwrite"). If the server does 409 on existing files for upload (as the collision handling implies), re-calling the identical upload will 409 again and the row flips to `failed`, with the inline TODO admitting no real fix exists. The Replace affordance is presented to the user but cannot actually replace. Either the affordance is misleading or the server contract is misunderstood — both are defects on the locked EDIT-10 flow.

**Fix:** Determine the actual server upload-collision contract. If Upload always overwrites (rename-over), the 409 path is unreachable and the collision state is dead code — remove it. If Upload 409s on existing, implement delete-then-upload or add a server `overwrite=1` form field. Do not ship a Replace button that re-issues an identical request.

## Info

### IN-01: `Editor` ignores `largeDismissed`/`syntaxDisabled` recompute on prop change

**File:** `frontend/src/components/Editor.tsx:163-233`
**Issue:** The CM6 mount effect has an empty dependency array (line 233) and the comment states filename/fileSize "do not remount." But `FileBrowserTab` reuses one `<Editor>` instance across file switches (the editor is conditionally rendered on `editingEntry !== null`, and React may reconcile rather than remount when switching files within edit mode). If a user opens a small file, then (via row-edit) switches to a large file without leaving edit mode, the size guards (`isLargeFile`, `isSyntaxDisabledSize`) recompute from new props but the editor doc and language compartment were set from the first file's content. In practice `editingEntry` going null between switches forces a remount, so this is latent rather than active — flagging for robustness.

**Fix:** Add a `key={editingEntry.name}` to the `<Editor>` in FileBrowserTab.tsx:1279 to force a clean remount on file switch, making the empty-dep assumption sound.

### IN-02: `languageFor` treats files with no extension and `Makefile`/`Dockerfile` inconsistently

**File:** `frontend/src/lib/languageFor.ts:26, 35-38`
**Issue:** `ext = filename.split('.').pop()?.toLowerCase() ?? ''` — for a file named `Makefile` (no dot), `split('.').pop()` returns `'makefile'` (the whole name), not `''`. The subsequent `languages.find` checks `l.extensions?.includes(ext)` with `ext='makefile'` (won't match) then `l.filename?.test(filename)` (may match Makefile). It happens to work via the filename regex, but the `ext` derivation is misleading and would mis-handle a file literally named `Makefile.bak` differently. Low impact (plain-text fallback is safe), flagged for clarity.

**Fix:** Detect the no-dot case explicitly: `const ext = filename.includes('.') ? filename.split('.').pop()!.toLowerCase() : ''`.

### IN-03: `onRowEdit` double-sets `selected` and has a no-op branch

**File:** `frontend/src/components/FileBrowserTab.tsx:1197-1211`
**Issue:** `setSelected(entry.name)` is called twice (lines 1199 and 1203) inside the same guarded callback, and the inner `if (selected === entry.name && ...)` reads `selected` from a stale closure that cannot equal `entry.name` on the same tick it was just set. The `handleEdit()` call (line 1207) only fires when the preview was *already* loaded for a previously-selected row; the common path (select-then-edit) relies on a separate user action. Dead/confusing branch.

**Fix:** Remove the duplicate `setSelected` and the unreachable same-tick equality branch; rely on the preview effect + explicit Edit affordance.

### IN-04: `headFile` / `readFileText` parse `content-length` without NaN guard

**File:** `frontend/src/lib/filesApi.ts:169-170, 181-182`
**Issue:** `Number.parseInt(sizeHeader, 10)` on a malformed or absent header yields `NaN`; the ternary guards absence (`sizeHeader ? ... : 0`) but a present-but-non-numeric header (e.g. proxy injecting garbage) yields `NaN`, which then flows into preview size logic (`size === 0` checks, `humanSize`). Low risk on a trusted loopback/proxy but the most-exposed surface should not propagate `NaN`.

**Fix:** `const n = Number.parseInt(sizeHeader ?? '', 10); const size = Number.isFinite(n) ? n : 0`.

### IN-05: `Upload` swallows multipart parse errors as 413 unconditionally

**File:** `internal/files/write.go:114-117`
**Issue:** `ParseMultipartForm` failures are always mapped to 413 "request too large" (line 115). A malformed multipart body (not over-cap) — e.g. truncated boundary — is a 400 client error, not a 413. Mislabeling makes the upload-queue error mapping (which branches on `isOverCap()`) show the "too large (max 50 MB)" skip copy for a corrupt-but-small upload, confusing the user.

**Fix:** Distinguish the cap error from a shape error:
```go
if err := r.ParseMultipartForm(8 << 20); err != nil {
    var maxErr *http.MaxBytesError
    if errors.As(err, &maxErr) { http.Error(w, "request too large", http.StatusRequestEntityTooLarge); return }
    http.Error(w, "malformed multipart form: "+err.Error(), http.StatusBadRequest); return
}
```

---

_Reviewed: 2026-06-14_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
