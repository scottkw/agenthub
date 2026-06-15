# Phase 125: React Editor (CodeMirror 6) — Desktop + Web - Pattern Map

**Mapped:** 2026-06-14
**Files analyzed:** 16 (1 net-new backend method-extension, 9 new frontend, 4 modified frontend, 2 test/fixture)
**Analogs found:** 16 / 16 (every file has a concrete in-repo analog)

> **THE ONE BACKEND TASK.** This is a React/Playwright phase with exactly **one** load-bearing server change: `internal/files/write.go` `Handler.Write` must read `If-Match`, compare it against the on-disk file's `mtime-size` validator (via the already-shipped `Sandbox.Stat`), and return **412** on mismatch. Everything else server-side (atomic write, upload, delete, rename, mkdir, capability gate, CSRF, routes) shipped in Phases 123/124 and is verified present. Do **not** add capability logic to `write.go` (the daemon socket is loopback-trust — see the file header comment lines 11-13).

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/files/write.go` (MODIFIED — `Handler.Write`) | controller (HTTP handler) | request-response (write w/ precondition) | sibling `Handler.Upload`/`Handler.Read` in same pkg | exact (same method set) |
| `internal/files/write_test.go` (MODIFIED — add `TestWrite_IfMatch*`) | test | — | existing `TestWriteFileAtomic_Overwrite` in same file | exact |
| `internal/files/handler.go` (OPTIONAL — emit `ETag` on Read) | controller | request-response | `Handler.Read` lines 291-302 (already sets Last-Modified) | exact (same method) |
| `internal/webserver/vendor_drift_test.go` (MODIFIED) | test | batch (file parse) | `TestXtermVendorVersionsMatchPnpmLock` (same file) | role-match (different vendoring shape — see note) |
| `frontend/src/components/Editor.tsx` (NEW) | component | event-driven (CM6 transactions) | `PreviewPane.tsx` (the read-mode pane it replaces) | role-match |
| `frontend/src/components/FileBrowser/EditorHeader.tsx` (NEW) | component | request-response | `PreviewPane.tsx` header lines 94-117 | exact (reuses `.file-browser__preview-header`) |
| `frontend/src/components/FileBrowser/SaveIndicator.tsx` (NEW) | component | event-driven (status) | `StatusLine` `role=status` / `QuitConfirmModal.dotColor` | role-match |
| `frontend/src/lib/useFilesWrite.ts` (NEW) | hook | CRUD (write/del/rename/mkdir/upload) | `useFilesCapability.ts` (hook shape) + `FilesApiClient` methods | role-match |
| `frontend/src/lib/filesApi.ts` (MODIFIED — add write methods) | service (API client) | CRUD + file-I/O (upload XHR) | existing `readFileText`/`listFiles` methods (same file) | exact |
| `frontend/src/lib/useFilesCapability.ts` (MODIFIED — add `canWrite`) | hook | request-response (probe) | the existing `files.read` probe (same file) | exact |
| `frontend/src/lib/languageFor.ts` (NEW) | utility | transform (ext → LanguageSupport) | no analog — RESEARCH §Code Examples is the source | NO ANALOG |
| `frontend/src/components/FileBrowser/modals/*Modal.tsx` (NEW ×5) | component (modal) | request-response | `QuitConfirmModal.tsx` (overlay/dialog/focus/Escape) | exact |
| `frontend/src/components/FileBrowser/FileRow.tsx` (MODIFIED — row actions) | component | request-response | existing `FileRow.tsx` icon + `iconFor()` (same file) | exact |
| `frontend/src/components/FileBrowser/PreviewPane.tsx` (MODIFIED — Edit btn) | component | request-response | existing `DownloadButton` slot lines 105-115 (same file) | exact |
| `cmd/playwright-fixture/main.go` (MODIFIED — mint `files.write` cap) | test fixture | — | existing owner/viewer cap mint lines 178-202 | exact |
| `frontend/e2e/global-setup.ts` (MODIFIED — parse `WRITE_CAP=`) | test setup | — | existing `VIEWER_CAP=` parse line 148 | exact |
| `frontend/e2e/files-write.spec.ts` (NEW) | test (e2e) | — | existing `files-browser.spec.ts` | role-match |

---

## Pattern Assignments

### `internal/files/write.go` → `Handler.Write` (controller, request-response) — **THE NET-NEW BACKEND CHANGE**

**Analog:** sibling methods in the SAME file (`Handler.Upload` lines 91-145, `writeWriteError` lines 236-249) + `Handler.Read` in `handler.go` (Last-Modified semantics) + `Sandbox.Stat` (the validator source, already shipped).

**Current `Handler.Write` (write.go:47-79) — what exists today (NO If-Match anywhere):**
```go
func (h *Handler) Write(w http.ResponseWriter, r *http.Request) {
	sb, _, err := h.sandboxFor(r)
	if err != nil { http.Error(w, "session not found", http.StatusNotFound); return }
	rel := r.URL.Query().Get("path")
	if rel == "" { http.Error(w, "path is required", http.StatusBadRequest); return }
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil { /* ... 413 / 400 ... */ }
	if err := sb.WriteFileAtomic(rel, data); err != nil {   // ← INSERT If-Match check BEFORE this line
		writeWriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(FileWriteResponse{Path: rel, Size: int64(len(data))})
}
```

**The validator primitive ALREADY exists** — `Sandbox.Stat` (sandbox.go:169-180) returns `os.FileInfo`, validates the path, is `os.Root`-confined:
```go
func (s *Sandbox) Stat(relPath string) (os.FileInfo, error) { /* validateAndClean + root.Stat */ }
```

**Net-new insert (after the `rel == ""` guard, BEFORE `WriteFileAtomic`):**
```go
// EDIT-05/08 optimistic concurrency. If-Match present + not wildcard → the
// caller asserts a known on-disk validator; reject (412) if the file changed.
// Wildcard ("*") or absent header → Force-overwrite / new-file path; proceed.
if ifMatch := r.Header.Get("If-Match"); ifMatch != "" && ifMatch != "*" {
	if fi, statErr := sb.Stat(rel); statErr == nil { // target exists; missing → new file, skip
		cur := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))
		if ifMatch != cur {
			http.Error(w, "file modified by another process", http.StatusPreconditionFailed)
			return
		}
	}
}
```
- Add `"fmt"` to the import block (write.go:19-27 currently has no `fmt`).
- **Validator format is the load-bearing contract** (RESEARCH Open Q6): `"<ModTime().UnixNano()>-<Size()>"` quoted. The client MUST derive the byte-identical string. `FileEntry.mtime` on the wire is `time.RFC3339` (`handler.go:173` formats `ModTime().UTC().Format(time.RFC3339)`), which is **NOT** UnixNano — so the safest path (RESEARCH-recommended) is to also emit `ETag` on Read (below) and have the client echo it verbatim. If you derive client-side instead, the `FileEntry.mtime` RFC3339 string cannot reconstruct UnixNano — you would need a server `ETag` or a new stat field. **Planner: pick "server emits ETag, client echoes" to eliminate the format-mismatch risk.**

**Error mapping is already centralized** — `writeWriteError` (write.go:236-249) maps `fs.ErrExist → 409` (the collision path EDIT-09/10 depends on, verified line 244-245). The 412 is a precondition, not a `WriteFileAtomic` error, so it lives inline in `Write` (above), NOT in `writeWriteError`.

**Shared-by-all-surfaces:** `Handler.Write` is the single method mounted on daemon + webserver + remote proxy (webserver mounts it via `SetFilesHandler`, server.go:150-163). The 412 check is written once and all three surfaces inherit it.

---

### `internal/files/handler.go` → `Handler.Read` (OPTIONAL — emit ETag)

**Analog:** the same method, lines 291-302, which already sets `Last-Modified` and delegates to `http.ServeContent`.

**Current (handler.go:291-302):**
```go
w.Header().Set("Content-Type", contentType)
if fi.Size() == 0 {
	w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
	w.WriteHeader(http.StatusOK)
	return
}
http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)  // sets Last-Modified, weak modtime ETag
```
**Add (recommended, to make the client echo path robust):**
```go
w.Header().Set("ETag", fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size())))
```
Set this on BOTH the zero-byte branch and before `ServeContent`. `fmt` is already imported nowhere in handler.go — add it, or build the string with `strconv`. The frontend `readFileText` then reads `res.headers.get('etag')` and threads it as the `If-Match` value on the next write. This is the **A3-vs-server-emit decision (Open Q6)** — emitting ETag is the lower-risk choice.

---

### `frontend/src/lib/filesApi.ts` (service, CRUD + file-I/O) — add write methods

**Analog:** existing methods in the SAME file — `readFileText` (lines 146-157), `buildQuery` (lines 102-108), `fetchOrThrow` (lines 110-122), `FilesApiError` (lines 57-85).

**Reuse the query builder + error mapping verbatim. Add a 412 predicate** to `FilesApiError` mirroring the existing `isOverCap()` (line 82-84):
```ts
/** 412 → If-Match precondition failed; another process changed the file (EDIT-08). */
isConflict(): boolean { return this.status === 412 }
/** 409 → name collision on write/rename/mkdir/upload (EDIT-09/10). */
isCollision(): boolean { return this.status === 409 }
```
**Write method (mirror `readFileText`'s URL+fetchOrThrow shape; add the If-Match header):**
```ts
async writeFile(sid: string, path: string, body: BodyInit, ifMatch?: string): Promise<void> {
  const url = `${this.baseURL}${this.pathPrefix}/write?${this.buildQuery(sid, path).toString()}`
  const headers: Record<string, string> = { 'Content-Type': 'application/octet-stream' }
  if (ifMatch) headers['If-Match'] = ifMatch
  await this.fetchOrThrow(url, { method: 'PUT', headers, body }) // throws FilesApiError(412) on conflict
}
```
- `del`/`rename`/`mkdir` follow the same `buildQuery` + `fetchOrThrow` pattern with `method: 'DELETE'` / `'POST'`. Rename POSTs a JSON body `{oldRel, newRel}` (matches `renameRequest`, write.go:171-174).
- **Upload is the exception — NOT `fetchOrThrow`.** EDIT-10 needs per-file `N%`, which `fetch` cannot provide (RESEARCH Pitfall 6). Use `XMLHttpRequest` with `xhr.upload.onprogress`, posting a `FormData` with a single `file` part + `dir` field (matches `Handler.Upload`, write.go:105-131 takes one `file` part). The cap token rides as a query param via `buildQuery` (already includes `cap` when present, line 106).

---

### `frontend/src/lib/useFilesCapability.ts` (hook) — add `canWrite`

**Analog:** the SAME file's existing `files.read` probe (lines 31-57) and `CapabilityState` 4-state (line 17).

**Existing probe shape to mirror:**
```ts
await client.listFiles(sessionId, '.')
if (!cancelled) setState('present')
// catch: err.isMissingFilesReadPerm() → 'denied'; else 'probe-failed'
```
**Per-surface `canWrite` source (RESEARCH Pitfall 2 + Open Q5 — CRITICAL, planner must confirm):**
- **Web-share:** probe the write route. `requireFilesWrite` returns 403-without-`files.write` (verified `capability_mw.go:147`). Mirror `isMissingFilesReadPerm` with a new `isMissingFilesWritePerm()` matching `files.write` in the body.
- **Desktop (Wails):** the daemon socket is **auth-less** — a write probe always succeeds regardless of the owner toggle. Derive `canWrite` from `SessionInfo.FilesWrite` (shipped 124), NOT from a probe. The `useFilesCapability` list-probe trick does NOT work for write on desktop.

---

### `frontend/src/lib/useFilesWrite.ts` (NEW hook)

**Analog:** `useFilesCapability.ts` hook structure (`useCallback`, `useState`, `AbortController` cleanup pattern lines 27-57) + the new `filesApi.ts` write methods above.

**Shape (per UI-SPEC §Component Tree):** `useFilesWrite(client, sessionId) → { write, del, rename, mkdir, upload, isSaving, saveError }`. The three-state save (`idle/saving/saved`) is driven here: set `isSaving` true before the PUT, on 200 snapshot + `Saved` for ~1.5s (mirror the Settings transient referenced in UI-SPEC §Color), on 412 surface the conflict (do not clear buffer — locked decision), on other errors set `saveError`.

---

### `frontend/src/components/Editor.tsx` (NEW component)

**Analog:** `PreviewPane.tsx` — the read-only pane Editor.tsx replaces in edit mode (RESEARCH Anti-Pattern: reuse the preview bytes, do NOT re-fetch). The CM6 mount itself has no in-repo analog; RESEARCH §Code Examples (Compartment + Cmd-S keymap) is the source.

**Reuse from PreviewPane:** the `.file-browser__preview` section wrapper + `data-testid="file-browser-preview"` (PreviewPane.tsx:87-92) so Playwright targets the pane without branching on read/edit mode. Mount CM6 imperatively in `useEffect` (NOT a React wrapper — RESEARCH Anti-Pattern: a wrapper hides `Compartment`). Initial `doc` = the text PreviewPane already holds (no re-fetch). `Compartment.reconfigure()` flips read-only↔editable without remount (EDIT-02).

**Heroicons** are `@heroicons/react/24/outline` at 14px (UI-SPEC, verified in `FileRow.tsx:1-9` and `PreviewPane.tsx:19`). Editor uses `PencilSquareIcon`, `ArrowPathIcon` (saving), `CheckCircleIcon` (saved), `ExclamationTriangleIcon` (error/conflict).

---

### `frontend/src/components/FileBrowser/EditorHeader.tsx` + `SaveIndicator.tsx` (NEW)

**Analog:** `PreviewPane.tsx` header (lines 94-117) — `.file-browser__preview-header` / `__preview-name` / right-aligned icon button. Reuse this structure verbatim; replace the DownloadButton slot with DirtyMarker + SaveIndicator + Save/Cancel buttons.

**SaveIndicator** is a `role="status" aria-live="polite"` region (UI-SPEC colorblind contract — icon+text carry meaning, color is decoration). Status colors reuse `QuitConfirmModal.dotColor` values (`#9ece6a` saved / `#e0af68` saving / `#f7768e` error — QuitConfirmModal.tsx:11-19). The dirty marker `●` carries `aria-label="Modified"`.

---

### `frontend/src/components/FileBrowser/PreviewPane.tsx` (MODIFIED — Edit button)

**Analog:** the existing `DownloadButton` slot in the SAME file (lines 105-115).

**Pattern (insert a Pencil button after Download, gated on `canWrite && text/markdown && !isBinary`):**
```tsx
{canWrite && (state.kind === 'text' || state.kind === 'markdown') && (
  <button
    className="file-browser__btn file-browser__btn--icon"
    aria-label={`Edit ${filename ?? ''}`}
    title="Edit"
    onClick={onEdit}
  >
    <PencilSquareIcon width={14} height={14} aria-hidden="true" />
  </button>
)}
```
No auto-edit (EDIT-03). Binary files never get the button — absence is the colorblind-safe signal.

---

### `frontend/src/components/FileBrowser/FileRow.tsx` (MODIFIED — row actions)

**Analog:** the existing `iconFor()` + `.file-browser__row-icon` (same file, lines 30-39, 110-119) and the 14px heroicon convention.

**Pattern:** add a `FileRowActions` cluster (Pencil/Rename/Move/Delete icon buttons) revealed on `:hover`/`:focus-within`, gated `canWrite && !entry.isDir && !entry.isBinary` for Edit (UI-SPEC §1). Each button is `.file-browser__btn--icon` with `aria-label="Edit {name}"` etc. Pass new optional props (`canWrite`, `onEdit`, `onRename`, ...) — keep them optional so existing `FileRow` callers/tests don't break.

---

### `frontend/src/components/FileBrowser/modals/*.tsx` (NEW ×5: Unsaved / Conflict / Delete / Collision / MoveTo)

**Analog:** `QuitConfirmModal.tsx` (the entire file is the pattern — UI-SPEC names it explicitly).

**Reuse verbatim (QuitConfirmModal.tsx):**
- Overlay click-to-cancel + `e.stopPropagation()` on the dialog (lines 57-64).
- `role="dialog"` + `aria-modal="true"` + `aria-labelledby` (lines 59-62).
- Escape-closes via `window.addEventListener('keydown')` (lines 26-32).
- **Default focus on the SAFE button** via `ref` + `useEffect(focus)` (lines 23, 35-37, 95-101). This is the colorblind/safety contract: Cancel / Keep editing / Discard my changes hold default focus; the destructive button (`#f7768e` Delete / Force overwrite / Replace) is never default-focused.
- `acting` guard to disable buttons during the async op (lines 22, 97-114).

All copy strings are **VERBATIM** from UI-SPEC §Copywriting Contract (locked by EDIT-07/08/09/10/11). Delete-dir variant states `{N} files inside` — count via a client-side `listFiles` walk before opening the modal (RESEARCH Open Q3; avoids a server change).

---

### `frontend/src/lib/languageFor.ts` (NEW utility) — NO ANALOG

No in-repo analog (CM6 is brand new). Source is RESEARCH §Code Examples "Lazy language detection by extension": `@codemirror/language-data` registry `.find()` + `desc.load()` dynamic import; Bash/shell via `@codemirror/legacy-modes/mode/shell` + `StreamLanguage` (RESEARCH Open Q2 — `@codemirror/lang-shell` does not exist).

---

### `internal/files/write_test.go` (MODIFIED — add `TestWrite_IfMatch*`)

**Analog:** `TestWriteFileAtomic_Overwrite` (same file, lines 65-89) for the sandbox setup, AND `internal/files/handler_test.go` for the `httptest` Handler-level pattern (the If-Match check lives in the HTTP `Handler.Write`, not the `Sandbox` primitive, so the new test must exercise the handler with `httptest.NewRequest` + an `If-Match` header — handler_test.go is the right analog for that wiring).

**Three cases (RESEARCH Wave 0):** 200 on matching validator, 412 on mismatch, 200 on new file with no If-Match header. Build the expected validator the same way `Handler.Write` does (`"<UnixNano>-<size>"`).

---

### `internal/webserver/vendor_drift_test.go` (MODIFIED) — version-parity, NOT file-copy

**Analog:** `TestXtermVendorVersionsMatchPnpmLock` in the SAME file (lines 20-74) — the pnpm-lock parse loop (lines 27-36) and the per-package compare (lines 59-73).

**KEY DIVERGENCE (RESEARCH Open Q1):** the xterm test asserts `pnpm-lock.yaml` == `web/vendor/xterm/VERSION` (a web-served copy). **CodeMirror has NO `web/vendor/` presence** — it is Vite-bundled into `frontend/dist`, not web-served. The UI-SPEC's `web/vendor/codemirror/` wording is **superseded/inaccurate** (RESEARCH Open Q1 + Pitfall on `web/vendor/codemirror/`). The new CodeMirror test must assert **`frontend/package.json` declared versions == `frontend/pnpm-lock.yaml` resolved versions** for every `@codemirror/*` + `codemirror` package. Reuse the lock-parse loop (swap the `pnpmXtermKeyRe` regex for a `@codemirror/` one), drop the `VERSION`-file half, read `package.json` instead. Do NOT fabricate a `web/vendor/codemirror/VERSION` manifest.

---

### `cmd/playwright-fixture/main.go` (MODIFIED — mint `files.write` cap)

**Analog:** the existing owner + viewer cap mint (SAME file, lines 178-202) and the env emission block (lines 263-269).

**Pattern (mirror viewerClaims, lines 191-202):**
```go
writeClaims := capability.Claims{
	SID: sessionID, Perms: "read,files.read,files.write",
	IAT: time.Now().Unix(), GrantID: "grant-playwright-fixture-write", V: 1,
}
writeToken, err := capability.Sign(writeClaims, fixedTestKey) // log.Fatalf on err
ws.AddGrant(sessionID, writeClaims.GrantID)
// ... then in the emission block (after line 265):
fmt.Printf("WRITE_CAP=%s\n", writeToken)
```
The existing viewer `read`-only cap (no `files.write`) covers EDIT-13's 403-without-cap scenario. The new `WRITE_CAP` covers the web-share write + 412 scenarios (RESEARCH Pitfall 7).

---

### `frontend/e2e/global-setup.ts` (MODIFIED — parse `WRITE_CAP=`)

**Analog:** the `VIEWER_CAP=` parse line (line 148) and the `FixtureEnv` resolve object (lines 156-164) in the SAME file.

**Pattern (mirror line 148 + add to the resolve object + the `FixtureEnv` interface in `fixture-env.ts:14-17`):**
```ts
if (line.startsWith('WRITE_CAP=')) writeCap = line.slice('WRITE_CAP='.length).trim()
// ...add `writeCap` to resolveEnv({...}) and to the FixtureEnv interface.
```
Watch the exact-prefix comment on line 146 — `CAP=` vs `WRITE_CAP=` ordering does not collide (different prefixes), but keep the `startsWith` checks distinct.

---

### `frontend/e2e/files-write.spec.ts` (NEW e2e — 14 scenarios)

**Analog:** `files-browser.spec.ts` (the Phase 120 merge-gate suite) — imports (`@playwright/test`, `loadFixtureEnv`, `filesApiURL`/`appUrl`/`viewerAppUrl` from `fixture-env`, lines 40-50), `test.describe.configure({ mode: 'serial' })` (line 53), and the cross-browser (Chromium+Firefox+WebKit via `playwright.config.ts`) structure. Add a `writeAppUrl(env)` helper alongside `viewerAppUrl` using the new `WRITE_CAP`. The 14 scenarios are enumerated in RESEARCH §Validation Architecture (EDIT-13 row).

---

## Shared Patterns

### Heroicons (all new/modified components)
**Source:** `@heroicons/react/24/outline` at `width={14} height={14} aria-hidden="true"` — verified `FileRow.tsx:1-9`, `PreviewPane.tsx:19`.
**Apply to:** Editor, EditorHeader, SaveIndicator, all modals, FileRow actions, PreviewPane Edit button, breadcrumb toolbar.

### BEM `.file-browser__*` classes (all new frontend chrome)
**Source:** `.file-browser__btn` / `__btn--icon` / `__preview-header` / `__preview-name` (verified `PreviewPane.tsx:95-117`, UI-SPEC §Design System).
**Apply to:** every button and header. UI-SPEC mandates **zero new CSS component families** and **zero new color hexes** (TokyoNight only). Colorblind contract is release-blocking — every state carries icon+text; verify at source level (glyph + literal text token), never by eye.

### Modal overlay/dialog/focus pattern
**Source:** `QuitConfirmModal.tsx` (overlay click-cancel, `role=dialog`+`aria-modal`, Escape-closes, safe-button default focus, `acting` guard).
**Apply to:** UnsavedChangesModal, ConflictModal, DeleteConfirmModal, CollisionConfirmModal, MoveToPickerModal.

### API client query + error mapping
**Source:** `FilesApiClient.buildQuery` (filesApi.ts:102-108), `fetchOrThrow` (110-122), `FilesApiError.is*()` predicates (57-85).
**Apply to:** all new write methods (write/del/rename/mkdir). **Exception:** upload uses XHR (not `fetchOrThrow`) for progress (RESEARCH Pitfall 6).

### Server status-mapping convention
**Source:** `writeWriteError` (write.go:236-249) — `ErrPathValidation→403`, `fs.ErrNotExist→404`, `fs.ErrExist→409`, else 500. The new **412 is inline in `Handler.Write`** (a precondition, not a write error).
**Apply to:** the one backend change. Client maps 412→`isConflict()`, 409→`isCollision()`.

### `os.Root`-confined validator source
**Source:** `Sandbox.Stat` (sandbox.go:169-180) — already shipped, path-validated, root-confined. The If-Match check calls this.
**Apply to:** `Handler.Write` 412 check.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `frontend/src/lib/languageFor.ts` | utility | transform | CodeMirror is brand-new; no extension→LanguageSupport mapper exists. Use RESEARCH §Code Examples + `@codemirror/language-data`. |
| Editor CM6 mount internals (`Editor.tsx` CM6 wiring) | component | event-driven | No CodeMirror in the repo today. Pane wrapper reuses PreviewPane; the CM6 `useEffect`/`Compartment`/keymap body has no analog — RESEARCH §Code Examples is authoritative. |

---

## Metadata

**Analog search scope:** `internal/files/`, `internal/webserver/`, `frontend/src/components/`, `frontend/src/components/FileBrowser/`, `frontend/src/lib/`, `frontend/e2e/`, `cmd/playwright-fixture/`
**Files scanned (read in full or targeted):** write.go, handler.go, sandbox.go (Stat), write_test.go, vendor_drift_test.go, filesApi.ts, useFilesCapability.ts, PreviewPane.tsx, FileRow.tsx, QuitConfirmModal.tsx, global-setup.ts, files-browser.spec.ts, playwright-fixture/main.go, webserver/server.go (SetFilesHandler), capability_test.go (route list grep)
**Pattern extraction date:** 2026-06-14
