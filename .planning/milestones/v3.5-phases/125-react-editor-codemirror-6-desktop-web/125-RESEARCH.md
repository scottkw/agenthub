# Phase 125: React Editor (CodeMirror 6) — Desktop + Web - Research

**Researched:** 2026-06-14
**Domain:** In-app CodeMirror 6 text editor + full write-affordance suite (create/mkdir/delete/rename/move/upload/drag-drop) wired into the existing v3.4 `FileBrowserTab`, cross-surface (Wails desktop + web-share), with ETag/If-Match optimistic concurrency
**Confidence:** HIGH (all integration claims verified against live source; CodeMirror packages verified against npm registry + official docs; the one load-bearing gap — write route does not honor If-Match — verified directly in `internal/files/write.go`)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Editor: CodeMirror 6, syntax highlighting by extension (Go, TS, Python, JSON, YAML, Markdown, Bash, HTML, CSS, + common langs).
- Edit button absent for binary files and callers without `files.write`.
- Files > 500 KB → large-file warning before edit; files approaching 5 MB cap → disable syntax highlighting with in-editor notice.
- Save: Cmd/Ctrl+S → atomic write (temp+sync+rename) with `If-Match: <etag>` header. Three-state save indicator (idle / saving… / saved ~1.5s). Dirty-state bullet/asterisk. Unsaved-changes guard is React-level only — NO `beforeunload` (Wails blocks it).
- Conflict: If-Match mismatch (HTTP 412) → "This file was modified by another process" with [Force overwrite] / [Save as new file] / [Discard my changes]. Buffer NEVER silently discarded.
- Write affordances (create file, mkdir, delete, rename, cross-dir move via "Move to…" picker, single + multi-file upload w/ per-file progress, drag-and-drop) visible/operable only when canWrite. 409 collision → "A file named X already exists. Replace it?" with Cancel as default.
- Recursive directory delete → confirm with file count.
- Testing: Playwright cross-browser e2e (Chromium + Firefox + WebKit). Zero CSP violations. `vendor_drift_test.go` keeps CodeMirror packages version-matched.

### Claude's Discretion
Component decomposition, CodeMirror extension wiring, state management, vendoring/bundling approach — at Claude's discretion guided by success criteria, the UI-SPEC, and existing FileBrowserTab patterns.

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped. (Note: Auto-save, Monaco, in-TUI CodeMirror, real-time collab, git integration, recursive search, binary hex editor, directory-zip upload, drag-out-to-FS download are all OUT per REQUIREMENTS.md "Out of Scope".)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EDIT-01 | CodeMirror 6 packages installed via pnpm + `vendor_drift_test.go` version parity; zero new CSP amendments | §Standard Stack, §Pitfall: vendor-drift mechanism is package.json↔pnpm-lock (NOT `web/vendor/codemirror/` — see §Open Question 1); §CSP confirmed unchanged |
| EDIT-02 | `Editor.tsx` mounts CM6; `Compartment.reconfigure()` read-only↔editable, no remount | §Code Examples (Compartment pattern); §Architecture Pattern 1 |
| EDIT-03 | Pencil Edit toggle in preview header; NO auto-edit; hidden when `isBinary` OR `!canWrite` | §Architecture Pattern 1; `isBinary` already on `FileEntry` (verified `filesApi.ts:20`) |
| EDIT-04 | Syntax highlighting by extension (Go, TS/TSX, JS/JSX, Python, JSON, YAML, MD, Bash/shell, HTML, CSS, common) | §Standard Stack (lang packs); §Code Examples (language-data detection); Bash via `@codemirror/legacy-modes` (see §Open Question 2) |
| EDIT-05 | Cmd/Ctrl+S → `PUT /api/files/write` with `If-Match: <etag>` (etag = mtime+size) | §CRITICAL FINDING — server write route does NOT honor If-Match today; net-new server work required (§Architecture Pattern 3, §Pitfall 1) |
| EDIT-06 | Dirty indicator off `EditorState.doc` vs snapshot; three-state save (idle/saving/saved ~1.5s) | §Code Examples; §Architecture Pattern 2 |
| EDIT-07 | React-level unsaved-changes guard (NOT `beforeunload`) on file-switch / navigate-up / tab-close | §Architecture Pattern 4; §Pitfall 5 |
| EDIT-08 | 412 conflict UX: [Force overwrite] / [Save as new file] / [Discard my changes] | §Architecture Pattern 3; depends on EDIT-05 server change |
| EDIT-09 | create file / mkdir / delete (file+recursive-dir w/ count) / rename / "Move to…" picker; 409 → Replace? | §Architecture Pattern 5; server routes verified live (`write.go`); recursive-dir count = §Open Question 3 |
| EDIT-10 | Upload single+multi via `<input multiple>` + drag-drop; per-file progress queue; 409 overwrite warn | §Architecture Pattern 6; §Pitfall 6 (XHR for progress, not fetch) |
| EDIT-11 | Large-file guard: >500KB warn-then-proceed; near 5MB → plain-text mode + notice | §Architecture Pattern 7; §Pitfall 4 |
| EDIT-12 | `useFilesWrite` hook + `useFilesCapability.canWrite`; all affordances gated | §Architecture Pattern 8; §Pitfall 2 (canWrite probe) |
| EDIT-13 | Playwright cross-browser e2e (14 scenarios) — merge gate | §Validation Architecture; §Pitfall 7 (fixture needs files.write cap variant) |
</phase_requirements>

## Summary

Phase 125 wires a CodeMirror 6 editor and the complete write-affordance suite into the v3.4 read-side `FileBrowserTab`, on both the Wails desktop and web-share surfaces. The frontend stack is fully ratified by milestone research and verified here against the npm registry: `codemirror@6.0.2` + `@codemirror/*` language packs, Vite-bundled (no CDN, no web worker, no CSP change). The Go write engine (`internal/files/write.go`), the `files.write` capability middleware (`requireFilesWrite` with CSRF Origin check), the `DaemonClient` write methods, and the webserver write routes ALL already shipped in Phases 123–124 and are verified present in the live tree. This phase is therefore **primarily a frontend phase** — with **one load-bearing exception**.

**CRITICAL FINDING (verified in source):** The shipped write route (`PUT /api/files/write`) does **NOT** read `If-Match`, does **NOT** return 412, and the read route (`http.ServeContent`) emits only `Last-Modified` — **no `ETag` header is set anywhere** in `internal/files/`. EDIT-05 and EDIT-08 (the If-Match → 412 optimistic-concurrency contract — a locked decision) cannot be satisfied by the existing engine. Phase 125 must add a small server-side change: compute a weak validator from mtime+size and have `Write` compare the inbound `If-Match` against the on-disk file's current validator, returning 412 on mismatch. This is the single backend task in an otherwise React/Playwright phase. Phase 123 "froze the write engine" for the sandbox primitives — but If-Match was scoped to the editor phase, not 123, so this is in-scope new work, not a 123 regression.

**Primary recommendation:** Decompose into 5 waves: (0) test scaffolding + the server If-Match/412 + ETag change + fixture `files.write` cap; (1) editor core (mount, Compartment toggle, language detection, large-file guard); (2) save flow + dirty state + 412 conflict; (3) directory write affordances (create/mkdir/rename/delete/move); (4) upload + drag-drop + progress queue; (5) Playwright cross-browser e2e merge gate + vendor-drift gate. Build CodeMirror directly via `useEffect` (no React wrapper — the `Compartment` API must stay accessible). Use **XHR not fetch** for upload progress. The `vendor_drift_test.go` gate must assert package.json↔pnpm-lock parity, NOT a `web/vendor/codemirror/` directory (CodeMirror is Vite-bundled into `dist`, not web-served like xterm — see Open Question 1).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Text editing UI / syntax highlighting | Browser/Client (React + CM6) | — | Pure client concern; CM6 runs in WebView and web-share browser identically |
| Edit/affordance gating (`canWrite`) | Browser/Client (`useFilesCapability`) | API (cap middleware enforces) | UI hides affordances; server is the real authority (403 if bypassed) |
| Atomic file write | API/Backend (`Sandbox.WriteFileAtomic`) | — | Already shipped (123); temp+Sync+rename inside `os.Root` |
| If-Match / 412 conflict detection | API/Backend (`Handler.Write`) | Browser (sends header, renders modal) | **NET-NEW server logic** — server is the only place that can compare on-disk validator |
| ETag/validator emission | API/Backend (read + stat) | Browser (derives from mtime+size) | Read route emits Last-Modified only today; client can derive `mtime+size` from `stat`/list entry without a server ETag header (cheaper path — see §Architecture Pattern 3) |
| Capability enforcement | API/Backend (`requireFilesWrite` + CSRF Origin) | — | Already shipped (124); webserver tier only, daemon socket is loopback-trust |
| Multipart upload parse + size cap | API/Backend (`Handler.Upload`) | Browser (FormData/XHR) | Already shipped (123); 50 MiB `MaxBytesReader` |
| Transport (local socket vs webserver+cap vs daemon proxy) | API/Backend | — | Invisible to UI; `FilesApiClient.pathPrefix` already abstracts it (verified `filesApi.ts:36-47`) |

## Standard Stack

### Core (NEW frontend dependencies — all verified on npm registry 2026-06-14)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `codemirror` | 6.0.2 | Metapackage (`basicSetup`, `EditorView`, `EditorState`) | Ratified milestone decision; MIT; org-scoped official packages from codemirror.net |
| `@codemirror/state` | 6.6.0 | `EditorState`, `Compartment`, transactions | Read-only↔editable toggle requires `Compartment` directly |
| `@codemirror/view` | 6.43.1 | `EditorView`, `keymap` (Cmd-S intercept) | DOM layer |
| `@codemirror/commands` | 6.10.3 | default keymap, history | EDIT-01 names it explicitly |
| `@codemirror/language` | 6.12.3 | language facets, `LanguageSupport` | Required by lang packs |
| `@codemirror/language-data` | 6.5.2 | lazy 120+ lang registry (dynamic import) | Extension-based detection (EDIT-04) |
| `@codemirror/lang-go` | 6.0.1 | Go | EDIT-04 |
| `@codemirror/lang-python` | 6.2.1 | Python | EDIT-04 |
| `@codemirror/lang-javascript` | 6.2.5 | JS/JSX/TS/TSX | EDIT-04 (handles TS via config) |
| `@codemirror/lang-json` | 6.0.2 | JSON | EDIT-04 |
| `@codemirror/lang-yaml` | 6.1.3 | YAML | EDIT-04 |
| `@codemirror/lang-markdown` | 6.5.0 | Markdown | EDIT-04 |
| `@codemirror/lang-html` | 6.4.11 | HTML | EDIT-04 |
| `@codemirror/lang-css` | 6.3.1 | CSS | EDIT-04 |
| `@codemirror/legacy-modes` | 6.5.3 | Bash/shell (Lezer has no native bash grammar) | EDIT-04 names "Bash/shell"; provided via legacy StreamLanguage — see §Open Question 2 |
| `@codemirror/theme-one-dark` | 6.1.3 | dark theme base | Structurally TokyoNight-adjacent; UI-SPEC wants TokyoNight tokens (see §Open Question 4) |

### Supporting (optional, common langs to round out EDIT-04 "common languages")
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@codemirror/lang-rust` | 6.0.2 | Rust | Project ships Go binaries; Rust is "common" enough to include |
| `@codemirror/lang-cpp` | 6.0.3 | C/C++ | "common languages" coverage |

(SQL, XML, PHP, Java, etc. are reachable lazily via `@codemirror/language-data` without separate top-level deps — see §Code Examples.)

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| CodeMirror 6 | Monaco | REJECTED at milestone: requires `worker-src blob:` CSP amendment + 2.4–6 MB bundle. Hard architectural blocker. |
| Direct `useEffect` wiring | `@uiw/react-codemirror` | REJECTED: wrapper hides the `Compartment` API needed for the read-only↔editable toggle (EDIT-02). |
| `@codemirror/legacy-modes` shell | `@codemirror/lang-shell` | Does NOT exist on npm (verified NOT FOUND). legacy-modes `shell` is the canonical CM6 path. |
| XHR upload progress | `fetch` + streams | `fetch` has no upload-progress event; per-file `N%` (EDIT-10) requires `XMLHttpRequest.upload.onprogress`. |

**Installation:**
```bash
cd frontend/
pnpm add codemirror@6.0.2 \
  @codemirror/state@6.6.0 @codemirror/view@6.43.1 @codemirror/commands@6.10.3 \
  @codemirror/language@6.12.3 @codemirror/language-data@6.5.2 \
  @codemirror/lang-go@6.0.1 @codemirror/lang-python@6.2.1 @codemirror/lang-javascript@6.2.5 \
  @codemirror/lang-json@6.0.2 @codemirror/lang-yaml@6.1.3 @codemirror/lang-markdown@6.5.0 \
  @codemirror/lang-html@6.4.11 @codemirror/lang-css@6.3.1 @codemirror/lang-rust@6.0.2 \
  @codemirror/lang-cpp@6.0.3 @codemirror/legacy-modes@6.5.3 @codemirror/theme-one-dark@6.1.3
```
No new Go dependencies. No `web/vendor/` additions (CodeMirror is Vite-bundled into `frontend/dist`, embedded via the existing `-tags wailsassets` `staticAppFS`). No CSP changes.

**Version verification:** All 17 packages confirmed via `npm view <pkg> version` on 2026-06-14 — every version above is the current published version (exact match to milestone STACK.md). `@codemirror/lang-shell` confirmed NOT to exist (use `@codemirror/legacy-modes`).

## Package Legitimacy Audit

slopcheck was unavailable in this session (`pip install slopcheck` failed silently — no network/registry access for the tool). Per the graceful-degradation protocol, packages are tagged `[ASSUMED]` for slopcheck status BUT are independently corroborated: all are `@codemirror/*` org-scoped packages documented on codemirror.net (the canonical authoritative source), the milestone STACK.md verified them via `npm view`, and this session re-verified every version against the live npm registry. The org scope `@codemirror/` is owned by Marijn Haverbeke (CodeMirror author) — not a slopsquat vector.

| Package | Registry | Source Repo | Registry Verify (this session) | slopcheck | Disposition |
|---------|----------|-------------|-------------------------------|-----------|-------------|
| `codemirror` | npm | github.com/codemirror/basic-setup | ✓ 6.0.2 | [ASSUMED] | Approved (canonical, org-scoped) |
| `@codemirror/state` | npm | github.com/codemirror/state | ✓ 6.6.0 | [ASSUMED] | Approved |
| `@codemirror/view` | npm | github.com/codemirror/view | ✓ 6.43.1 | [ASSUMED] | Approved |
| `@codemirror/commands` | npm | github.com/codemirror/commands | ✓ 6.10.3 | [ASSUMED] | Approved |
| `@codemirror/language` | npm | github.com/codemirror/language | ✓ 6.12.3 | [ASSUMED] | Approved |
| `@codemirror/language-data` | npm | github.com/codemirror/language-data | ✓ 6.5.2 | [ASSUMED] | Approved |
| `@codemirror/lang-*` (go/python/javascript/json/yaml/markdown/html/css/rust/cpp) | npm | github.com/codemirror/lang-* | ✓ all resolved | [ASSUMED] | Approved |
| `@codemirror/legacy-modes` | npm | github.com/codemirror/legacy-modes | ✓ 6.5.3 | [ASSUMED] | Approved |
| `@codemirror/theme-one-dark` | npm | github.com/codemirror/theme-one-dark | ✓ 6.1.3 | [ASSUMED] | Approved |

**Packages removed due to slopcheck [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

Recommendation for planner: a single `checkpoint:human-verify` task confirming the pnpm-lock resolved versions + integrity hashes after `pnpm add` is sufficient given the canonical org provenance — no need to gate each of the 17 individually.

## Architecture Patterns

### System Architecture Diagram

```
                       ┌─────────────────────────────────────────────┐
   USER (Cmd/Ctrl+S,   │              React FileBrowserTab            │
   click Edit, drop    │  ┌──────────┐   ┌─────────────────────────┐ │
   files) ───────────▶ │  │Breadcrumb│   │ PreviewPane             │ │
                       │  │ +write   │   │  (read mode: TextPreview│ │
                       │  │ toolbar  │   │   /Markdown/Image)      │ │
                       │  └────┬─────┘   │   ── Edit toggle ──▶     │ │
                       │       │         │  Editor.tsx (edit mode) │ │
                       │       │         │   CM6 view (Compartment)│ │
                       │       │         └───────────┬─────────────┘ │
                       │       │ create/mkdir/del/   │ save(content, │
                       │       │ rename/move/upload  │   ifMatch)    │
                       │       ▼                     ▼               │
                       │   useFilesWrite hook  ──── derives etag     │
                       │       │  (mtime+size from stat/list entry)  │
                       └───────┼─────────────────────────────────────┘
                               │ fetch / XHR (upload) — pathPrefix abstracts transport
              ┌────────────────┼───────────────────────────────────┐
              ▼                ▼                                     ▼
   LOCAL: daemon socket   WEB-SHARE: webserver           REMOTE: daemon proxy
   (no auth, loopback)    requireFilesWrite              /api/files/remote/{sid}/*
              │           (HasPerm files.write           (forwards body — 124 CAP-10)
              │            + CSRF Origin check)                      │
              └────────────────┴───────────────────────────────────┘
                               ▼
                    internal/files/Handler  (write.go — SHIPPED 123)
                      Write  ── NEW: read If-Match, compare mtime+size, 412 ──▶
                      Upload / Delete / Rename / Mkdir
                               ▼
                    Sandbox.WriteFileAtomic (temp+Sync+rename inside os.Root)
```

### Recommended Project Structure
```
frontend/src/
├── components/
│   ├── Editor.tsx                  # NEW — CM6 mount, Compartment toggle, large-file guard
│   ├── FileBrowserTab.tsx          # MODIFIED — write toolbar, edit toggle, modals root
│   └── FileBrowser/
│       ├── PreviewPane.tsx         # MODIFIED — Edit button in header (canWrite && text && !binary)
│       ├── FileRow.tsx             # MODIFIED — hover/focus FileRowActions cluster
│       ├── EditorHeader.tsx        # NEW — filename + dirty marker + save indicator + Save/Cancel
│       ├── SaveIndicator.tsx       # NEW — three-state role=status aria-live
│       ├── UploadQueuePanel.tsx    # NEW — per-file progress (XHR)
│       ├── UploadDropOverlay.tsx   # NEW — drag-drop target
│       ├── InlineNameInput.tsx     # NEW — create/mkdir/rename inline row
│       └── modals/                 # NEW — Unsaved / Conflict / Delete / Collision / MoveTo
│           └── *.tsx               # all reuse QuitConfirmModal pattern
├── lib/
│   ├── filesApi.ts                 # MODIFIED — add write/upload/delete/rename/mkdir + etag
│   ├── useFilesCapability.ts       # MODIFIED — add canWrite probe state
│   ├── useFilesWrite.ts            # NEW — { write, del, rename, mkdir, upload, isSaving, saveError }
│   └── languageFor.ts             # NEW — extension → CM6 LanguageSupport (lazy)
└── e2e/
    └── files-write.spec.ts         # NEW — 14-scenario cross-browser merge gate

internal/files/write.go             # MODIFIED — Write reads If-Match, returns 412 (EDIT-05/08)
internal/files/handler.go           # MODIFIED (optional) — emit ETag on Read alongside Last-Modified
internal/webserver/vendor_drift_test.go  # MODIFIED — CodeMirror package.json↔pnpm-lock parity
```

### Pattern 1: Read-only ↔ editable via Compartment (EDIT-02) — no remount
**What:** A single persistent `EditorView`; flip editability with `Compartment.reconfigure()`. Initial content = the already-loaded preview text (NO re-fetch — reuse the bytes PreviewPane already holds).
**When:** On Edit toggle and on Cancel.
**Example:** see §Code Examples.

### Pattern 2: Dirty state off `EditorState.doc` vs saved snapshot (EDIT-06)
**What:** Keep `savedSnapshot: string`. On every transaction, `dirty = view.state.doc.toString() !== savedSnapshot`. On 200 save, reset snapshot. Three-state indicator is a `role="status" aria-live="polite"` region (icon+text carry meaning; color is decoration — colorblind contract).

### Pattern 3: ETag = mtime+size, derived client-side; server compares (EDIT-05, EDIT-08)
**What:** The locked decision is "ETag = mtime + size, not full-content hash." The client ALREADY has `mtime` and `size` for the open file (from the `stat`/list `FileEntry` — verified `filesApi.ts:13-22`). So the client computes the validator string itself (e.g. `` `"${mtime}-${size}"` ``) and sends it as `If-Match`. **The server must be extended** to: on `Write`, stat the existing target, compute the same `mtime-size` validator, and if the inbound `If-Match` does not match → return `412 Precondition Failed` (do NOT write). If the target does not exist yet (new file), `If-Match` is omitted and the write proceeds. Optionally also emit `ETag: "<mtime>-<size>"` on the Read response for symmetry, but the client does not need it (stat already provides the parts).
**Why this shape:** It avoids any server-held state and any full-content hashing, exactly per the locked decision. The 412 check lives in the **`Handler.Write` method** (shared by daemon + webserver + remote proxy) so all three surfaces get it for free.
**Conflict resolution branches (EDIT-08):**
- `Force overwrite` → re-PUT with `If-Match` omitted (or a wildcard the server treats as "skip check").
- `Save as new file` → PUT to a new path `{basename}-copy{ext}`, no If-Match.
- `Discard my changes` → re-fetch server content, replace buffer, clear dirty. (Buffer preserved until user explicitly discards.)

### Pattern 4: React-level navigation guard (EDIT-07) — NEVER `beforeunload`
**What:** Intercept three triggers inside React before they mutate the open file: (1) selecting a different file in `FileListPane`, (2) breadcrumb navigate-up, (3) tab close. If `dirty`, open `UnsavedChangesModal` (default focus on "Keep editing") and only proceed on explicit choice. Wails blocks `beforeunload`, so there is no browser-native fallback — the guard MUST be entirely in React state/handlers.

### Pattern 5: Inline name inputs (create/mkdir/rename) + modal pickers (move/delete)
**What:** create-file/mkdir/rename use an inline `InlineNameInput` row (Enter commits, Esc cancels) — not modals. Move uses `MoveToPickerModal` (directory tree). Delete uses `DeleteConfirmModal` (file vs recursive-dir variant). 409 (server returns `StatusConflict` on `fs.ErrExist` — verified `write.go:245`) → `CollisionConfirmModal` (Cancel default focus, Replace destructive).

### Pattern 6: Upload via XHR for per-file progress (EDIT-10)
**What:** One `XMLHttpRequest` per file (the server `Upload` handler takes one `file` part — verified `write.go:109`). Track `xhr.upload.onprogress` → `N%` per queue row. Multi-file = N parallel/sequential XHRs into the queue. Cross-dir "Move" is a `rename` (server `Rename` validates both paths and is move-capable — verified `write.go:180-201` + FSW-02), NOT copy+delete.

### Pattern 7: Large-file + binary guards (EDIT-11, EDIT-03)
**What:** `FileEntry.size` (already known) drives the gate: >500KB → `LargeFileNotice` warn-then-proceed banner; ≥~5MB → mount CM6 with NO language pack (plain text) + persistent "Syntax highlighting disabled" caption. Binary (`FileEntry.isBinary`, already on the wire) → no Edit affordance anywhere. Note the server read cap is 5 MiB (`maxPreviewBytes`, verified `handler.go`) so files >5 MiB cannot even be loaded — the "near 5MB" plain-text tier is the practical ceiling.

### Pattern 8: `canWrite` capability probe (EDIT-12)
**What:** Extend `useFilesCapability` to also resolve `canWrite`. The existing hook probes `GET /api/files/list` and maps 403-with-"files.read" → denied (verified `useFilesCapability.ts:46`). For write, the cleanest probe is a cheap write-route preflight OR (preferred) derive `canWrite` from the capability token itself when available. Because the daemon socket is auth-less, on the **desktop** surface `canWrite` reflects the per-session owner toggle (read from the daemon's session info — `SessionInfo.FilesWrite`, shipped 124, verified in 124-05 summary), while on **web-share** `canWrite` must be probed (a `files.write`-denied response maps the denied state, mirroring the read probe). See §Open Question 5.

### Anti-Patterns to Avoid
- **Re-fetching file content on Edit toggle.** The preview already loaded it — reuse. Re-fetch only on "Discard my changes" (412 path).
- **`fetch` for upload progress.** No progress events — use XHR.
- **Normalizing CRLF→LF on save.** CM6 normalizes line endings on input; preserve original by sending the raw buffer; do not re-encode (Pitfall: line-ending corruption).
- **`beforeunload` for the unsaved guard.** Wails blocks it (EDIT-07).
- **A `web/vendor/codemirror/` directory.** CodeMirror is Vite-bundled, not web-served. The vendor-drift gate is version-parity, not file-copy (see Open Question 1).
- **A React CodeMirror wrapper.** Hides `Compartment` (EDIT-02 blocker).
- **Silently discarding the buffer on 412 or 401.** Locked: buffer NEVER discarded without explicit user action.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Syntax highlighting | regex/Shiki-in-editor | CM6 Lezer lang packs | Incremental, cursor-aware, contextual; Shiki is static-HTML only |
| Editable toggle | unmount/remount editor | `Compartment.reconfigure()` | Preserves scroll/selection/undo; no flicker |
| Language detection | hand map of extensions | `@codemirror/language-data` registry | 120+ langs, lazy `import()`, Vite code-splits |
| Atomic write / temp-rename | client-side anything | server `Sandbox.WriteFileAtomic` (SHIPPED 123) | TOCTOU-safe, inside `os.Root` |
| Multipart parse + size cap | manual body streaming | server `Handler.Upload` (SHIPPED 123) | `MaxBytesReader` before parse |
| Capability gate / CSRF | client-only checks | `requireFilesWrite` (SHIPPED 124) | Server is the authority; Origin check already present |
| Cross-dir move | copy+delete | server `Rename` (SHIPPED 123) | Single atomic op; both paths sandbox-validated |
| Upload progress | timers/estimates | `XMLHttpRequest.upload.onprogress` | Real bytes-sent events |

**Key insight:** ~90% of the backend is already shipped and verified in the live tree. The ONLY backend gap is If-Match/412 in `Handler.Write`. Everything else this phase touches is React + Playwright.

## Runtime State Inventory

This is a feature-addition phase, not a rename/refactor — most categories are N/A, but two are load-bearing:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no datastore keys renamed | None |
| Live service config | `SessionInfo.FilesWrite` per-session toggle (shipped 124) governs owner `canWrite`; default OFF. Web-share viewer `files.write` opt-in (shipped 124). | None new — Phase 125 READS these signals; does not create them |
| OS-registered state | None | None — verified no Task Scheduler/launchd involvement |
| Secrets/env vars | Cap tokens carry `files.write` perm (shipped 124). `$EDITOR` is Phase 126, not here. | None |
| Build artifacts | After `pnpm add` of CodeMirror, `frontend/dist` bundle grows ~135KB gz core +35KB/lang; embedded via `-tags wailsassets`. Production Wails build MUST rebuild dist (project memory: prod needs `-tags wailsassets`). | Rebuild + re-embed dist; vendor_drift_test asserts version parity |

## Common Pitfalls

### Pitfall 1: Assuming the write route already does If-Match (it does NOT)
**What goes wrong:** Planning EDIT-05/08 as "frontend just sends a header." The shipped `Handler.Write` (verified `write.go:47-79`) reads the body and calls `WriteFileAtomic` — it never inspects `If-Match`, never returns 412. No `ETag` header is set on Read (ServeContent emits `Last-Modified` only).
**Why it happens:** Phase 123 "froze the write engine" — but only the sandbox primitives + atomic write. If-Match was scoped to the editor phase.
**How to avoid:** Add a server task in Wave 0: `Handler.Write` stats the existing target, computes `mtime-size`, compares to `If-Match`, returns 412 on mismatch. Add a `write_test.go` case for the 412 path. This is shared logic → all three surfaces inherit it.
**Warning signs:** A plan that has zero Go changes; an e2e 412 scenario with no server-side 412 emission.

### Pitfall 2: `canWrite` cannot be probed the same way as `canRead` on the desktop surface
**What goes wrong:** Reusing the `useFilesCapability` list-probe for write. The daemon socket is auth-less (loopback trust) — a `GET /api/files/list` against the daemon always succeeds regardless of write enablement, so a probe won't reveal whether writes are enabled.
**How to avoid:** On desktop, derive `canWrite` from the daemon session signal (`SessionInfo.FilesWrite`, shipped 124). On web-share, the webserver `requireFilesWrite` returns 403 without the cap, so a probe (or token inspection) works. See §Open Question 5 — recommend the planner confirm the exact `canWrite` source per surface.
**Warning signs:** Edit button shows on desktop even when the owner has not enabled writes.

### Pitfall 3: Wails WebView Tab / Cmd-V conflicts (cross-surface parity watch)
**What goes wrong:** CM6 captures Tab for indentation and Cmd-V for paste; the Wails `app.go` has a Phase 49 macOS Cmd-V clipboard handler that may double-paste; Tab may navigate focus instead of indenting.
**How to avoid:** During UAT verify Tab inserts indentation and Cmd-V does not double-paste inside CM6 in the WebView. If a conflict surfaces, conditionally suppress the app.go Cmd-V handler when CM6 has focus (mirror the existing `isXtermFocused.ts` focus-guard pattern, verified present in `lib/`). User is colorblind — verify any visual state at source level.
**Warning signs:** Tab leaves the editor; paste duplicates content on macOS.

### Pitfall 4: Large-file freeze (Lezer parse on multi-hundred-KB files)
**What goes wrong:** Loading a 2MB minified JS into CM6 with a language pack freezes parse for seconds.
**How to avoid:** >500KB warn-then-proceed; near-5MB → mount with NO language extension (plain text). The 5 MiB server read cap is the hard ceiling.

### Pitfall 5: Unsaved-changes guard missing a trigger
**What goes wrong:** Guarding tab-close but not file-switch or breadcrumb navigate-up → silent buffer loss.
**How to avoid:** All three triggers route through one `guardThen(action)` helper that checks `dirty`.

### Pitfall 6: Upload progress with `fetch` (impossible)
**What goes wrong:** `fetch` exposes download streams but no upload-progress event; the per-file `N%` requirement (EDIT-10) is unimplementable with fetch.
**How to avoid:** XHR per file with `upload.onprogress`.

### Pitfall 7: Playwright fixture has no `files.write` cap
**What goes wrong:** The playwright-fixture mints owner `read,write,files.read` and viewer `read` (verified `cmd/playwright-fixture/main.go:175-198`) — neither carries `files.write`. EDIT-13's "web-share write with a files.write cap" and "403 without the cap" scenarios cannot run.
**How to avoid:** Wave 0 extends the fixture to mint a third cap variant `read,files.read,files.write` (write-enabled viewer) and emit it as a new env line (e.g. `WRITE_CAP=`), parsed by `global-setup.ts` (mirrors the existing `VIEWER_CAP=` handling, verified `global-setup.ts:147-148`). The existing viewer `read`-only cap covers the 403 scenario.
**Warning signs:** e2e write scenarios silently testing against the auth-less daemon instead of the cap-gated webserver.

## Code Examples

### CM6 mount with Compartment toggle + Cmd-S keymap (EDIT-02, EDIT-05, EDIT-06)
```typescript
// frontend/src/components/Editor.tsx (sketch)
// Source: https://codemirror.net/examples/readonly/ + https://codemirror.net/docs/ref/#commands
import { EditorView, keymap } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { basicSetup } from 'codemirror'
import { oneDark } from '@codemirror/theme-one-dark'

const editable = new Compartment()
const language = new Compartment()

const view = new EditorView({
  parent: mountEl,
  state: EditorState.create({
    doc: initialContent, // reuse preview bytes — NO re-fetch
    extensions: [
      basicSetup,
      oneDark,
      language.of([]),                       // set lazily per extension
      editable.of([EditorView.editable.of(false), EditorState.readOnly.of(true)]),
      keymap.of([{ key: 'Mod-s', preventDefault: true, run: () => { void onSave(); return true } }]),
      EditorView.updateListener.of((u) => {
        if (u.docChanged) setDirty(u.state.doc.toString() !== savedSnapshot)
      }),
    ],
  }),
})

// Enter edit mode (canWrite && user clicked pencil):
view.dispatch({ effects: editable.reconfigure([
  EditorView.editable.of(true), EditorState.readOnly.of(false),
]) })
```

### Lazy language detection by extension (EDIT-04)
```typescript
// frontend/src/lib/languageFor.ts
// Source: https://github.com/codemirror/language-data
import { languages } from '@codemirror/language-data'
import { StreamLanguage } from '@codemirror/language'
import { shell } from '@codemirror/legacy-modes/mode/shell'

export async function languageFor(filename: string) {
  const ext = filename.split('.').pop()?.toLowerCase() ?? ''
  if (['sh', 'bash', 'zsh'].includes(ext)) return StreamLanguage.define(shell) // legacy-modes
  const desc = languages.find(l => l.extensions?.includes(ext) || l.filename?.test(filename))
  if (!desc) return []                // unknown → plain text
  const support = await desc.load()   // dynamic import; Vite code-splits
  return support
}
```

### Save with derived If-Match (EDIT-05)
```typescript
// frontend/src/lib/useFilesWrite.ts (sketch) — etag derived from the FileEntry the UI already holds
const etag = `"${entry.mtime}-${entry.size}"`        // mtime+size, per locked decision
const res = await fetch(`${base}/api/files/write?${q}`, {
  method: 'PUT',
  headers: { 'Content-Type': 'application/octet-stream', 'If-Match': etag },
  body: buffer,                                       // raw bytes, no re-encode
})
if (res.status === 412) openConflictModal()           // EDIT-08
```

### Server-side If-Match check (NET-NEW — EDIT-05 backend)
```go
// internal/files/write.go — inside Handler.Write, BEFORE WriteFileAtomic
// Source: pattern derived from existing handler.go ServeContent modtime semantics
if ifMatch := r.Header.Get("If-Match"); ifMatch != "" && ifMatch != "*" {
    if fi, statErr := sb.Stat(rel); statErr == nil {     // target exists
        cur := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))
        if ifMatch != cur {
            http.Error(w, "file modified by another process", http.StatusPreconditionFailed)
            return
        }
    }
    // target missing → new file; If-Match is moot, proceed
}
```
*(Client and server MUST agree on the exact validator format — recommend `"<mtime-unix-nano>-<size>"`. The client's `FileEntry.mtime` is a string from the wire; the planner must align the client derivation with whatever timestamp format `Stat` returns — see §Open Question 6.)*

### Vendor-drift gate for CodeMirror (EDIT-01)
```go
// internal/webserver/vendor_drift_test.go — NEW test, parallel to the xterm one.
// CodeMirror is Vite-bundled (NOT web-served), so there is no web/vendor/codemirror/VERSION.
// The gate instead asserts package.json declared versions == pnpm-lock.yaml resolved versions
// for every @codemirror/* and codemirror package — catching a lockfile/manifest drift.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Monaco for in-app editors | CodeMirror 6 for CSP-strict / single-binary apps | CM6 GA 2022; Sourcegraph migrated 2023 | CM6 is the correct fit for vendored/no-worker discipline |
| `@codemirror/lang-shell` | does not exist | — | Use `@codemirror/legacy-modes` `shell` |
| `ETag` from server | client-derived mtime+size validator | this phase (locked decision) | No server state, no full-content hash |

**Deprecated/outdated:**
- CodeMirror 5 + `react-codemirror2` — wrong major version; do not use.
- `@uiw/react-codemirror` — viable but hides `Compartment`; not for this phase.

## Project Constraints (from CLAUDE.md + project memory)

- **JS/TS:** `camelCase`, `PascalCase` components, ESLint + Prettier, TypeScript types. `pnpm` (NOT npm/yarn) for installs.
- **Wails production build:** requires `-tags wailsassets` (embed.FS for correct MIME types) — the CodeMirror bundle must be in `frontend/dist` before a prod build; `wails dev` and prod must both be exercised.
- **Wails DevTools disabled in production** (project memory) — UATs needing the inspector use `wails dev` or web-share to regular Chrome.
- **User is COLORBLIND** (release-blocking): verify every editor state and destructive confirmation at the SOURCE level (glyph + literal text token in code), never by eye. Save indicator, dirty marker, conflict/delete modals all carry icon+text; color is decoration only (UI-SPEC colorblind table is the contract).
- **Cross-surface parity is RELEASE-BLOCKING:** desktop (Wails) ⇔ web-share must show byte-identical UX. Every affordance, modal, copy string, keyboard shortcut identical; only transport differs.
- **Testing:** `vitest` for unit (source-inspection pattern like `TerminalPanel.test.tsx`), Playwright for cross-browser e2e. 80%+ coverage in critical components.
- **Cross-check GitHub issues during UAT** (project memory): scan scottkw/agenthub open issues before recording UAT pass.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | slopcheck status `[ASSUMED]` for all 17 CM packages (tool unavailable; corroborated by npm registry + canonical org provenance) | Package Legitimacy Audit | LOW — org-scoped official packages, version-verified twice |
| A2 | `@codemirror/theme-one-dark` is an acceptable starting theme; a full TokyoNight port is a follow-on | Standard Stack | MEDIUM — UI-SPEC says "zero new hexes" and wants TokyoNight tokens; one-dark introduces non-TokyoNight hexes (see Open Q4) |
| A3 | Client derives etag from `FileEntry.mtime`+`size` without a server ETag header | Pattern 3 | MEDIUM — depends on mtime string format alignment (Open Q6) |
| A4 | `canWrite` is derived from `SessionInfo.FilesWrite` on desktop and probed on web-share | Pattern 8 | MEDIUM — exact source not yet confirmed in frontend (Open Q5) |
| A5 | Bash/shell via `@codemirror/legacy-modes` satisfies EDIT-04 "Bash/shell" | Standard Stack | LOW — this is the canonical CM6 mechanism (no native Lezer bash) |

## Open Questions

1. **Vendor-drift mechanism for a Vite-bundled (not web-served) dependency.**
   - What we know: `vendor_drift_test.go` today asserts `web/vendor/xterm/VERSION` == `pnpm-lock.yaml`. CodeMirror has NO `web/vendor/` presence (verified `web/vendor/` contains only `xterm`); it is bundled into `frontend/dist` by Vite. The UI-SPEC's reference to `web/vendor/codemirror/` is inaccurate.
   - What's unclear: whether the planner should (a) assert package.json↔pnpm-lock parity (recommended) or (b) fabricate a `web/vendor/codemirror/VERSION` manifest purely to satisfy the EDIT-01 wording.
   - Recommendation: option (a) — assert declared vs resolved version parity for all `@codemirror/*` + `codemirror`. Note the UI-SPEC `web/vendor/codemirror/` wording as superseded.

2. **Bash highlighting package.** `@codemirror/lang-shell` does not exist (verified NOT FOUND). Recommendation: `@codemirror/legacy-modes` `shell` via `StreamLanguage`. Confirm acceptable for EDIT-04.

3. **Recursive-dir delete file count source.** EDIT-09 modal states "all N files inside." The server `Delete` (verified `write.go:150`) does recursive removal but returns no pre-count. Recommendation: count via a `GET /api/files/list` walk client-side before opening the modal, OR add a count to the delete response (server change). Planner to choose; client-walk avoids a server change.

4. **Theme: one-dark vs TokyoNight tokens.** UI-SPEC mandates "zero new hexes" and TokyoNight CM6 theme. `@codemirror/theme-one-dark` introduces one-dark hexes. Recommendation: ship a small hand-rolled CM6 theme (`EditorView.theme({...})`) using the existing TokyoNight hexes from `style.css` (the colorblind/color contract is release-blocking) rather than `theme-one-dark`. This is a few dozen lines; the lib import can be dropped.

5. **`canWrite` source per surface.** Confirm: desktop reads `SessionInfo.FilesWrite`; web-share probes a write route or inspects the cap. The frontend has no `canWrite` today.

6. **etag timestamp format alignment.** The client `FileEntry.mtime` is a wire string; the server stat returns `time.Time`. The client-derived `If-Match` and the server-computed validator MUST use the identical format. Recommendation: server emits an `ETag` header on Read/Stat (`"<mtime-unix-nano>-<size>"`) and the client echoes it verbatim — this sidesteps any format-mismatch risk entirely (slightly more server work than A3, but robust). Planner to decide A3 (client-derives) vs this (server-emits-echo).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| pnpm | install CM packages | ✓ (project standard) | — | — |
| Vite | bundle CM into dist | ✓ | 8.x | — |
| `@playwright/test` | EDIT-13 e2e | ✓ | ^1.59.1 (verified package.json) | — |
| Playwright browsers (Chromium/Firefox/WebKit) | EDIT-13 | likely (suite exists) | — | `pnpm exec playwright install` if missing |
| Go write engine (write.go) | EDIT-05/09/10 | ✓ shipped 123 | — | — |
| `requireFilesWrite` + routes | web-share writes | ✓ shipped 124 | — | — |
| npm registry (CM packages) | install | ✓ all 17 verified | see table | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** Playwright browser binaries may need `playwright install` on a fresh checkout.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4 (unit, source-inspection) + @playwright/test ^1.59.1 (cross-browser e2e) + Go `testing` (write.go) |
| Config file | `frontend/playwright.config.ts` (Chromium+Firefox+WebKit, serial, fixture via globalSetup) |
| Quick run command | `cd frontend && pnpm test` (vitest) ; `go test ./internal/files/...` |
| Full suite command | `cd frontend && pnpm exec playwright test` ; `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EDIT-05 | server returns 412 on If-Match mismatch | unit (Go) | `go test ./internal/files/ -run TestWrite_IfMatch` | ❌ Wave 0 |
| EDIT-02/06 | Compartment toggle; dirty off doc-vs-snapshot | unit (vitest, source) | `pnpm test Editor` | ❌ Wave 0 |
| EDIT-12 | canWrite gates affordances | unit (vitest) | `pnpm test useFilesCapability` | ⚠️ extend existing |
| EDIT-13 | local save / web-share write w/ cap / 403 / create / mkdir / delete file / delete dir recursive / rename / move / single upload / multi upload / 412 / binary no-edit / large-file | e2e (3 browsers) | `pnpm exec playwright test files-write.spec.ts` | ❌ Wave 0 (new spec + fixture write cap) |
| EDIT-01 | CodeMirror version parity | unit (Go) | `go test ./internal/webserver/ -run VendorDrift` | ⚠️ extend existing |
| (CSP) | zero CSP violations after editor ships | e2e | extend `web-csp.spec.ts` console-error assertion | ⚠️ extend existing |

### Sampling Rate
- **Per task commit:** `pnpm test` (vitest, <30s) + `go test ./internal/files/...` for the write task.
- **Per wave merge:** `pnpm exec playwright test` (the new spec for the relevant wave) + `go test ./...`.
- **Phase gate:** full cross-browser Playwright suite green + vendor-drift green before `/gsd:verify-work`.

### Wave 0 Gaps
- [ ] `internal/files/write_test.go` — extend with `TestWrite_IfMatch` (412 mismatch, 200 match, new-file no-header) — covers EDIT-05
- [ ] `cmd/playwright-fixture/main.go` — mint third cap `read,files.read,files.write`; emit `WRITE_CAP=` — covers EDIT-13 web-share
- [ ] `frontend/e2e/global-setup.ts` — parse `WRITE_CAP=` (mirror `VIEWER_CAP=`)
- [ ] `frontend/e2e/files-write.spec.ts` — new 14-scenario spec
- [ ] `internal/webserver/vendor_drift_test.go` — CodeMirror version-parity test (package.json↔pnpm-lock)
- [ ] `frontend/src/components/__tests__/Editor.test.tsx` — source-inspection unit test
- [ ] Framework install: `pnpm exec playwright install` if browsers absent

## Security Domain

`security_enforcement` is enabled (not set to false). Most write-path security is already shipped + audited in 123/124 and re-audited in Phase 127 (SEC). Phase 125's NEW surface is small.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no (web-share cap is the auth; shipped 124) | — |
| V3 Session Management | no | — |
| V4 Access Control | yes | `requireFilesWrite` + `HasPerm(files.write)` (shipped 124); client `canWrite` gate is UX-only, server is authority |
| V5 Input Validation | yes | server `validateAndClean` on every path (shipped 123); upload filename `filepath.Base` (shipped); client must not trust its own gate |
| V6 Cryptography | no | cap tokens signed elsewhere (capability pkg) |
| V12 Files & Resources | yes | atomic write, 50 MiB cap, denylist, symlink-safe `os.Root` (all shipped 123); **If-Match 412 is the NEW control this phase adds** |

### Known Threat Patterns for React + Go file-write
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Lost-update (concurrent edit overwrite) | Tampering | If-Match → 412 (NET-NEW this phase, EDIT-05/08) |
| CSRF on write verbs | Spoofing/Tampering | `requireFilesWrite` Origin check (shipped 124, verified `capability_mw.go:160`) |
| Path traversal on write/rename/upload | Tampering | `validateAndClean` both paths; `filepath.Base` on upload name (shipped 123) |
| Buffer loss on 401/412 (data integrity for the user) | DoS (user data) | NEVER discard buffer without explicit user action (locked decision) |
| Binary file corruption via editor round-trip | Tampering | `isBinary` → no Edit affordance (EDIT-03) |
| XSS via rendered file content | — | CM6 renders as text, not HTML; preview already omits `rehype-raw` (verified test exists) |

## Sources

### Primary (HIGH confidence)
- Live AgentHub source (verified this session): `internal/files/write.go` (no If-Match/412 — load-bearing), `internal/files/handler.go` (Read uses ServeContent, Last-Modified only, no ETag), `internal/files/types.go` (FileEntry size+mtime), `internal/webserver/server.go` (write routes mounted under `requireFilesWrite`), `internal/webserver/capability_mw.go` (Origin/CSRF check present), `internal/webserver/csp_mw.go` (style-src 'unsafe-inline', no worker-src), `frontend/src/lib/filesApi.ts`, `frontend/src/lib/useFilesCapability.ts`, `frontend/src/components/FileBrowserTab.tsx`, `frontend/playwright.config.ts`, `cmd/playwright-fixture/main.go` (cap variants), `frontend/e2e/global-setup.ts`, `internal/webserver/vendor_drift_test.go`
- npm registry (`npm view <pkg> version`, 2026-06-14): all 17 CodeMirror packages confirmed at the versions listed; `@codemirror/lang-shell` confirmed NOT FOUND
- Milestone v3.5 research (HIGH): `.planning/research/STACK.md`, `ARCHITECTURE.md`, `PITFALLS.md`
- Phase 123/124 SUMMARY files (shipped engine + capability + GUI toggle, verified)

### Secondary (MEDIUM confidence)
- codemirror.net official docs (Compartment/readonly/language-data patterns) — cited in milestone STACK.md, well-established
- 125-UI-SPEC.md (component tree, copy contract, colorblind table) — internal design contract

### Tertiary (LOW confidence)
- None requiring validation beyond the Open Questions above.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all 17 packages version-verified against npm; CM6 ratified at milestone
- Architecture: HIGH — backend shipped + verified in source; the one gap (If-Match) verified by direct read
- Pitfalls: HIGH — each tied to a verified source fact (auth-less socket, no ETag, fixture caps)
- Theme/colorblind alignment: MEDIUM — Open Q4 (one-dark vs TokyoNight tokens)

**Research date:** 2026-06-14
**Valid until:** 2026-07-14 (stable — CodeMirror 6.x is mature; the only volatility is if Phase 126/127 lands shared changes to write.go before 125 executes)
