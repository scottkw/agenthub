# Stack Research — v3.5 Write-Side File Browser + In-App Code Editor

**Domain:** Write operations (create/upload/delete/rename/mkdir) + in-app code editor with syntax highlighting + TUI $EDITOR shell-out
**Researched:** 2026-06-14
**Confidence:** HIGH (all library versions verified against npm registry; Bubble Tea v2 API verified against Context7/pkg.go.dev; CSP impact verified against live codebase; Monaco CSP issues verified against tracked GitHub issues)

This is a **subsequent-milestone STACK.md** — it does NOT re-survey the React/Wails/Go base stack already validated through v3.4. All validated stack items (`os.OpenRoot`, `wailsapp/mimetype`, `react-markdown`, `remark-gfm`, `shiki`, `charmbracelet/glamour`, `charm.land/bubbles/v2@v2.1.0`, Go 1.26.3, React 19.2.4, Vite 8, Vitest 4) are in-place and confirmed via `go.mod` and `frontend/package.json`. This document covers only the NEW capabilities required for v3.5.

Go version confirmed: **go 1.26.3**. Bubble Tea version confirmed: **charm.land/bubbletea/v2 v2.0.6**.

---

## TL;DR

1. **Editor library**: Use **CodeMirror 6** (`codemirror@6.0.2` + `@codemirror/*` packages). **Do NOT use Monaco.** Monaco requires `worker-src blob:` in CSP — a new CSP amendment that violates project security discipline. CodeMirror 6 is already covered by the existing `style-src 'self' 'unsafe-inline'` policy (inline style injection is already allowed for xterm.js). Bundle 135 KB gzipped vs Monaco's 2.4–6+ MB.

2. **Syntax highlighting**: CodeMirror 6 provides its own Lezer-based syntax highlighting via per-language packages (`@codemirror/lang-*`). **Shiki is not needed for the editor** — it stays in place for the read-only file preview pane (shipped v3.4). Do not add a separate highlighter alongside the editor.

3. **File upload (Go side)**: Use `net/http` stdlib (`r.ParseMultipartForm` + `r.FormFile`). `mime/multipart` is sufficient for browser multipart POST. Gate with `http.MaxBytesReader` before parsing. No new Go dep needed.

4. **TUI $EDITOR shell-out**: Use `tea.Exec(cmd, callback)` + implement `tea.ExecCommand` interface. This is **already proven and in use** in v3.4's `internal/tui/attach.go` (`attachCmd` struct). The `$EDITOR` shell-out follows the identical pattern — implement `ExecCommand`, call `tea.Exec`, handle `editorDoneMsg` in `Update`. No new dep, no new API to learn.

**CSP status after v3.5**: Zero new CSP amendments required. CodeMirror 6 inline style injection is already permitted by `style-src 'self' 'unsafe-inline'` (Amendment 1, Phase 89). No `worker-src`, no `script-src` changes needed.

---

## Recommended Stack

### Core Technologies (in-place — for integration reference only)

| Technology | Version | Relevant to v3.5 |
|------------|---------|-----------------|
| Go | 1.26.3 | `os.OpenRoot` write methods; `mime/multipart` for upload |
| `charm.land/bubbletea/v2` | v2.0.6 | `tea.Exec` + `tea.ExecCommand` for $EDITOR shell-out |
| `charmbracelet/glamour` | v0.8.0 | Markdown preview (unchanged from v3.4) |
| React | 19.2.4 | `CodeEditorTab.tsx` — already in use |
| Vite | 8.x | Build tooling — no change needed |
| `shiki` | 4.2.0 (in v3.4 as 4.1.0) | Read-only file preview pane syntax highlighting — already shipped |

### New Frontend Dependencies (v3.5 adds)

| Library | Version | Purpose | Bundle size (min+gz) | CSP impact |
|---------|---------|---------|---------------------|------------|
| `codemirror` | 6.0.2 | Editor metapackage (`basicSetup` + `EditorView` + `EditorState`) | ~135 KB gzipped (full bundle with minification) | None new — `style-src 'unsafe-inline'` already set (Amendment 1, Phase 89) |
| `@codemirror/state` | 6.6.0 | State model (`EditorState`, `Compartment`, transactions) | Included in `codemirror` metapackage | None |
| `@codemirror/view` | 6.43.1 | DOM rendering (`EditorView`, `keymap`) | Included in `codemirror` metapackage | None |
| `@codemirror/lang-javascript` | 6.2.5 | JS/TS syntax highlighting + indentation | ~35 KB per lang (lazy loaded) | None |
| `@codemirror/lang-python` | 6.2.1 | Python syntax highlighting | Same | None |
| `@codemirror/lang-go` | 6.0.1 | Go syntax highlighting | Same | None |
| `@codemirror/lang-markdown` | 6.5.0 | Markdown syntax highlighting in editor (separate from preview) | Same | None |
| `@codemirror/lang-json` | 6.0.2 | JSON syntax highlighting | Same | None |
| `@codemirror/lang-yaml` | 6.1.3 | YAML syntax highlighting | Same | None |
| `@codemirror/lang-css` | 6.3.1 | CSS syntax highlighting | Same | None |
| `@codemirror/lang-html` | 6.4.11 | HTML syntax highlighting | Same | None |
| `@codemirror/lang-rust` | 6.0.2 | Rust syntax highlighting | Same | None |
| `@codemirror/lang-cpp` | 6.0.3 | C/C++ syntax highlighting | Same | None |
| `@codemirror/language-data` | 6.5.2 | Lazy language registry (120+ langs via dynamic import) | ~10 KB core; langs loaded on demand | None |
| `@codemirror/theme-one-dark` | 6.1.3 | Tokyo-Night-compatible dark theme | ~5 KB | None — CSS injected via existing `'unsafe-inline'` |

**Total new JS weight**: ~135 KB gzipped for the editor core + ~35 KB per language loaded lazily. For the 13 curated languages above: ~135 KB + ~350 KB loaded-on-demand = ~485 KB maximum, only if all 13 languages are opened in the same session. In practice, 1–2 languages active at once.

**No new Go dependencies** are required for v3.5. Write operations extend the existing `internal/files/` package using Go stdlib exclusively.

### New Go Capabilities (stdlib only — no new modules)

| Capability | Package | Purpose |
|-----------|---------|---------|
| Write file | `os.Root.Create` / `os.Root.OpenFile` | Create or overwrite files within sandbox root |
| Delete file | `os.Root.Remove` | Remove a file within sandbox root |
| Rename/move | `os.Root.Rename` | Rename or move within sandbox (cross-dir allowed within root) |
| Create directory | `os.Root.Mkdir` | Create directory within sandbox root |
| Multipart upload | `net/http` + `mime/multipart` | Parse browser `multipart/form-data` uploads |
| Upload size guard | `net/http.MaxBytesReader` | Wrap `r.Body` before `ParseMultipartForm` to enforce size cap |

All these methods exist on the `os.Root` type (Go 1.24+). Verified via pkg.go.dev — `os.Root` exposes `Create`, `Mkdir`, `OpenFile`, `Remove`, `Rename` in addition to the v3.4-used `Open`. No new module import.

---

## Decision 1: Editor Library — CodeMirror 6 vs Monaco

### Recommendation: CodeMirror 6

**Verdict**: CodeMirror 6. Rationale is architectural, not preference.

### Comparison Matrix

| Criterion | CodeMirror 6 | Monaco Editor |
|-----------|-------------|---------------|
| **Bundle size (gzipped)** | ~135 KB (minified; verified via official bundle example docs) | 2.4–6+ MB total; TypeScript worker alone is 6.68 MB uncompressed (monaco-editor/issues/5154) |
| **CSP: script-src** | No new requirement — uses no eval, no workers | Requires `worker-src blob:` for language workers (confirmed: payloadcms/payload#10229, keycloak/keycloak#32901); falls back to main-thread with warnings when workers blocked |
| **CSP: style-src** | Requires `'unsafe-inline'` (dynamic `<style>` injection) — ALREADY PERMITTED by Amendment 1 (Phase 89) | Also requires `'unsafe-inline'` + inline style violations (monaco-editor/issues/4927) |
| **Worker model** | No web workers in core; Lezer parser runs synchronously on main thread | Mandatory per-language web workers (`editor.worker`, `json.worker`, `ts.worker`, etc.); worker loading via `MonacoEnvironment.getWorker` |
| **Vite/Wails compatibility** | Clean Vite ESM — `pnpm add codemirror` is sufficient | Requires `vite-plugin-monaco-editor` or manual `getWorker` config; Vite 8 regression with Oxc minifier (vitejs/vite#22009) |
| **Vendoring feasibility** | Standard Vite-bundled npm dep — no web-served files, no CDN. Zero `vendor_drift_test.go` changes. | Same npm bundling — BUT worker `.js` files must be reachable via URL at runtime, complicating embed.FS + single-binary packaging |
| **Mobile/iPad touch (web-share)** | Native platform selection and editing on iOS; CodeMirror 6 is widely reported mobile-friendly; no virtual-keyboard regression | Less mobile-tested; worker overhead adds latency on slower tablet CPUs |
| **Read-only to editable transition** | `EditorView.editable.of(false)` + `EditorState.readOnly.of(true)` configurable via `Compartment.reconfigure()` — clean runtime toggle, no remount | `editor.updateOptions({ readOnly: true/false })` — also clean, but irrelevant given CSP blocker |
| **Language packs** | Per-package: `@codemirror/lang-*` (one npm pkg per language, lazy-loadable); `@codemirror/language-data` registry for 120+ langs | Built-in 70+ language support — but baked into the large worker bundles |
| **License** | MIT | MIT |
| **Proven migration precedent** | Sourcegraph migrated FROM Monaco TO CM6: -43% JS download (2.4 MB Monaco alone); Replit uses CM6; Firefox DevTools uses CM6 | VS Code, CodeSandbox, StackBlitz — all projects with different CSP/bundle constraints than AgentHub |

### The CSP Blocker

Monaco requires web workers for its language features. The standard integration loads workers via blob URLs (`new Worker(blob:...)`). This requires `worker-src blob:` in CSP.

AgentHub's current CSP (`csp_mw.go`) has no `worker-src` directive — the `script-src` fallback applies, which is `'self' 'wasm-unsafe-eval'`. Blob URLs are NOT same-origin, so Monaco workers would silently fail and fall back to synchronous in-main-thread mode (causing UI freezes on large files).

Alternatives to blob URLs (inline workers, AMD loader from static files) exist but require significant Monaco configuration work that defeats Monaco's purpose. The cost-benefit is negative: Monaco is a wrong-fit library for this CSP and single-binary architecture.

**CodeMirror 6 has no web worker requirement.** Its Lezer-based parser runs synchronously on the main thread. This is CSP-clean by design.

### Read-Only to Editable Transition

CodeMirror 6 uses `Compartment` to reconfigure facets at runtime without remounting:

```typescript
import { Compartment } from "@codemirror/state";
import { EditorView, basicSetup } from "codemirror";
import { EditorState } from "@codemirror/state";

const editableCompartment = new Compartment();

// Initial state: read-only (matches v3.4 plain-text rendering)
const extensions = [
  basicSetup,
  editableCompartment.of([
    EditorView.editable.of(false),
    EditorState.readOnly.of(true),
  ]),
];

// When files.write capability confirmed AND user clicks Edit button:
view.dispatch({
  effects: editableCompartment.reconfigure([
    EditorView.editable.of(true),
    EditorState.readOnly.of(false),
  ]),
});
```

The editor renders identically in both modes (same syntax highlighting, same theme, same line numbers). No remount or component teardown required.

### Language Detection and Dynamic Loading

Use `@codemirror/language-data` for file-extension-based language detection and lazy loading:

```typescript
import { languages } from "@codemirror/language-data";
import { Compartment } from "@codemirror/state";

const langCompartment = new Compartment();

async function getLanguageExtension(filename: string) {
  const ext = filename.split(".").pop() ?? "";
  const langDesc = languages.find(l =>
    l.extensions?.includes(ext) || l.filename?.test(filename)
  );
  if (!langDesc) return [];
  const lang = await langDesc.load(); // dynamic import — Vite code-splits automatically
  return langCompartment.of(lang.language.support);
}
```

This pattern loads language packs lazily — the initial bundle contains only the registry metadata (~10 KB), not the grammar bytecode. Languages are fetched via Vite code-splitting on first open.

---

## Decision 2: Syntax Highlighting in the Editor

### Recommendation: CodeMirror 6 Lezer grammars (built-in to lang packages)

Do not add Shiki alongside the editor. Shiki is a static HTML highlighter designed for pre-rendered output. CodeMirror 6 provides live incremental syntax highlighting via its Lezer parser, which supports:
- Incremental re-parse on keystrokes (O(change_size) not O(file_size))
- Contextual highlighting (scope-aware, not regex-only)
- The same Lezer grammars that Shiki 4.x uses for several languages

**Shiki v3.4 usage is unchanged**: Shiki remains in the read-only file preview pane (`FileBrowserTab.tsx`) for syntax-highlighted static preview of files that have NOT been opened for editing. When a file is opened in the CodeMirror editor, the editor's own Lezer highlighting replaces Shiki's static output.

### Bundle Size Implications

| Scenario | JS loaded (gzipped) |
|----------|-------------------|
| Editor core only (`codemirror` package) | ~135 KB |
| + 1 language (e.g., `@codemirror/lang-go`) | ~170 KB |
| + 13 curated languages (all loaded simultaneously) | ~485 KB |
| Monaco minimum (editor.worker + core) | ~2.4 MB (Sourcegraph measurement) |
| Monaco with TypeScript worker | ~8+ MB uncompressed |

### Theme

Use `@codemirror/theme-one-dark` (6.1.3) as the base for v3.5. One Dark is structurally similar to TokyoNight and serves as an immediate dark theme. A full TokyoNight port can be done as a follow-on if visual parity becomes a release requirement — it is not complex (CodeMirror themes are pure TS extension objects with hex colors).

---

## Decision 3: File Upload — Browser Side + Go Side

### Recommendation: `multipart/form-data` POST + stdlib `mime/multipart` (no new dep)

**Browser side** (React `FileBrowserTab.tsx`): Standard `<input type="file">` with `FormData` API:

```typescript
async function uploadFile(
  sessionId: string,
  dir: string,
  file: File,
  cap: string
) {
  const form = new FormData();
  form.append("file", file);
  form.append("dir", dir);
  const res = await fetch(`/api/files/upload?session=${sessionId}`, {
    method: "POST",
    headers: { Authorization: `Bearer ${cap}` },
    body: form,
  });
  if (!res.ok) throw new Error(await res.text());
}
```

No JS library needed. `FormData` + `fetch` is universal in all supported browsers (Chromium, Firefox, WebKit, Safari on iPad).

**Go side** (`internal/files/handler.go` — new endpoint):

```go
// POST /api/files/upload
func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
    // MaxBytesReader wraps body FIRST — prevents OOM before parsing
    r.Body = http.MaxBytesReader(w, r.Body, 50<<20) // 50 MB upload cap

    if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB in-memory
        http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
        return
    }

    dir := r.FormValue("dir") // relative path within session root

    file, header, err := r.FormFile("file")
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    defer file.Close()

    // os.Root.Create sandboxes the path — cannot escape WorkDir
    dst, err := h.sandbox.Create(filepath.Join(dir, header.Filename))
    if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
    defer dst.Close()

    if _, err := io.Copy(dst, file); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
}
```

**Why stdlib is sufficient**: `mime/multipart` is Go's standard implementation of RFC 2046/2388. It:
- Parses `multipart/form-data` correctly for all major browsers
- Streams to disk for files exceeding `maxMemory` parameter
- Exposes `multipart.FileHeader.Filename` for the uploaded filename
- Has zero external deps

The security pattern: `MaxBytesReader` wraps the body FIRST (prevents OOM from oversized uploads), then `ParseMultipartForm` parses into memory/disk, then `os.Root.Create` sandboxes the destination path. Same three-layer defense as v3.4's read handler.

**Security note**: Go 1.22+ fixed `GO-2024-2599` (multipart disk unbounded). Current go 1.26.3 is safe. The `MaxBytesReader` pattern is still required to prevent network-level DoS regardless of Go version.

---

## Decision 4: TUI $EDITOR Shell-Out — Bubble Tea v2

### Recommendation: `tea.Exec` + `tea.ExecCommand` interface (already in use — zero new API)

This is not a new API. `tea.Exec` is already implemented in v3.4 for TUI attach (`internal/tui/attach.go`). The `$EDITOR` shell-out follows the **identical pattern** with a different `ExecCommand` implementation body.

### API Verification (Context7 + codebase)

In Bubble Tea v2 (`charm.land/bubbletea/v2 v2.0.6`), the function is `tea.Exec` (NOT `tea.ExecProcess` — that was the v1 name):

```go
// tea.Exec suspends the TUI, runs the ExecCommand, resumes.
// Confirmed in pkg.go.dev and in use at internal/tui/update.go:
//   return m, tea.Exec(cmd, func(err error) tea.Msg { return attachDoneMsg{err: err} })
func tea.Exec(c tea.ExecCommand, fn tea.ExecCallback) tea.Cmd
```

### $EDITOR ExecCommand Implementation

```go
// editorCmd implements tea.ExecCommand to shell out to $EDITOR.
// Structure is identical to attachCmd in attach.go.
type editorCmd struct {
    path   string    // absolute path to file (resolved through os.Root first)
    stdin  io.Reader
    stdout io.Writer
    stderr io.Writer
}

func (e *editorCmd) SetStdin(r io.Reader)  { e.stdin = r }
func (e *editorCmd) SetStdout(w io.Writer) { e.stdout = w }
func (e *editorCmd) SetStderr(w io.Writer) { e.stderr = w }

func (e *editorCmd) Run() error {
    editorBin := os.Getenv("EDITOR")
    if editorBin == "" {
        editorBin = "vi" // universal fallback
    }
    cmd := exec.Command(editorBin, e.path)
    cmd.Stdin = e.stdin   // Bubble Tea provides raw stdin
    cmd.Stdout = e.stdout // Bubble Tea provides raw stdout
    cmd.Stderr = e.stderr
    return cmd.Run()
}
```

**Calling site** (`Model.Update` in `internal/tui/update.go`):

```go
case tea.KeyPressMsg:
    if msg.String() == "e" && m.files.selectedIsFile() {
        return m, tea.Exec(&editorCmd{path: m.files.selectedAbsPath()}, func(err error) tea.Msg {
            return editorDoneMsg{err: err}
        })
    }
```

### What Bubble Tea Does on `tea.Exec`

1. Suspends its renderer (clears alt screen if active)
2. Restores terminal to cooked mode
3. Calls `cmd.SetStdin/SetStdout/SetStderr` with its wrapped I/O
4. Calls `cmd.Run()` — blocks until $EDITOR exits
5. Re-enters alt screen, resumes renderer
6. Delivers `editorDoneMsg` to `Update`

No teardown code needed in `editorCmd.Run()` — Bubble Tea handles the suspend/resume lifecycle. Contrast with `attachCmd.Run()` which manually sets raw mode (needed for PTY byte-level input); `editorCmd` does NOT set raw mode — the $EDITOR handles its own terminal mode.

### Path Safety for $EDITOR

The path passed to `editorCmd` must be an absolute real path on the local filesystem. The `$EDITOR` process runs as a child of the TUI process and is outside the `os.Root` sandbox. Security responsibility stays with the file handler: the path must have been resolved through `os.Root` first (confirming it is within the session's WorkDir). For remote sessions, $EDITOR shell-out is local-only — the remote files capability goes through the daemon proxy, not local disk, so there is no $EDITOR equivalent for remote sessions (correct: same as v3.4 remote read was API-only, not fs-local).

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Monaco Editor (`monaco-editor@0.55.1`) | Requires `worker-src blob:` CSP amendment — blocked by project security discipline. Workers silently fall back to main-thread when blocked, causing UI freezes. Bundle 2.4–6+ MB. Vite 8 Oxc minifier regression (vitejs/vite#22009). | CodeMirror 6 |
| `@monaco-editor/react` (suren-atoyan) | Wraps Monaco — inherits ALL Monaco CSP issues | `codemirror` direct |
| `vite-plugin-monaco-editor` | Automates Monaco worker configuration — does NOT resolve `worker-src blob:` CSP requirement | N/A |
| Shiki in the editor | Static HTML highlighter — no incremental re-parse, no cursor, no editing support | CodeMirror 6's Lezer grammars (built into lang packages) |
| `highlight.js` or `prism` in the editor | Same reason as Shiki — static highlighters only | CodeMirror 6's Lezer grammars |
| `react-codemirror2` (uiwjs v5 wrapper) | Wraps CodeMirror 5, not 6 — wrong version | `codemirror` (v6) |
| `@uiw/react-codemirror` React wrapper | Opinionated abstraction; hides `Compartment` API needed for read-only toggle; adds dep for no gain | Wire CodeMirror 6 directly in `useEffect` |
| CDN-loaded CodeMirror | Violates `vendor_drift_test.go` discipline and `script-src 'self'` CSP | Vite-bundled npm dep (these are build-time bundled, NOT web-served terminal page assets) |
| `multer` or other Node.js upload libs | Wrong runtime — Go server | stdlib `mime/multipart` |
| `tea.ExecProcess` | v1 Bubble Tea API name — compile error with `charm.land/bubbletea/v2` | `tea.Exec` (already in codebase, confirmed in `internal/tui/update.go`) |
| `os/exec.Command` called directly in `Update()` | Blocks the TUI event loop — freezes renderer | `tea.Exec` (suspends TUI cleanly) |
| Write paths via `filepath.Join` without `os.Root` | TOCTOU-vulnerable — same risk as v3.4 read side | All write paths through `os.Root` methods (`Create`, `Mkdir`, `Remove`, `Rename`) |
| `rehype-raw` with react-markdown (v3.4 carry-forward) | Enables raw HTML passthrough in file preview — XSS risk | Omit `rehype-raw` (already absent in v3.4) |

---

## CSP Impact Assessment

| New Capability | CSP Change Required | Notes |
|---------------|-------------------|-------|
| CodeMirror 6 editor | None | `style-src 'unsafe-inline'` already set (Amendment 1, Phase 89); no workers, no eval, no WASM |
| CodeMirror 6 lang packs | None | Loaded via Vite code-splitting (bundled JS, same-origin) |
| File upload (POST multipart) | None | `form-action 'self'` already set; `connect-src 'self'` covers fetch to same-origin API |
| Save file (PUT/PATCH text) | None | `connect-src 'self'` already covers fetch to same-origin API |
| DELETE/rename/mkdir | None | Same as above |
| `$EDITOR` shell-out | N/A | TUI-only; no browser surface, no CSP involvement |
| `files.write` capability bit | None | Protocol-level; no new CSP directives |

**Zero new CSP amendments required for v3.5.** This is a strong architectural win for choosing CodeMirror 6.

**Desktop WebView note**: The CSP middleware (`csp_mw.go`) applies only to webserver routes (`/sessions/{id}`, `/dashboard`, `/join`). The Wails desktop WebView uses Chromium's default policy (no `Content-Security-Policy` response header). CodeMirror 6 works correctly in both surfaces with identical configuration.

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Editor | CodeMirror 6 | Monaco Editor 0.55.1 | `worker-src blob:` CSP requirement is a hard architectural blocker; 2.4–6 MB bundle; Vite 8 regression |
| Editor | CodeMirror 6 | Ace Editor | Older architecture; less active development; not tree-shakeable; inferior mobile support |
| Editor | CodeMirror 6 | `@uiw/react-codemirror` React wrapper | Opinionated abstraction hides `Compartment` API needed for read-only toggle |
| Editor syntax | Lezer (CodeMirror built-in) | Shiki alongside CodeMirror | Static HTML highlighter; CodeMirror already provides live incremental highlighting |
| Upload (Go) | stdlib `mime/multipart` | Any Node.js upload library | Wrong runtime |
| Upload (Go) | stdlib `mime/multipart` | Manual body streaming | stdlib already handles streaming to disk for large files |
| TUI editor shell-out | `tea.Exec` + `ExecCommand` | `tea.ExecProcess` | v1 API name; renamed to `tea.Exec` in v2 |
| TUI editor shell-out | `tea.Exec` shell-out to `$EDITOR` | Embedded `bubbles/v2/textarea` | textarea is for single-line/small inputs; $EDITOR is the Unix-idiomatic pattern for file editing in a TUI; no new dep |

---

## Integration Points with Existing Architecture

| New Capability | Where It Hooks In | Protocol / Contract Change |
|---------------|-------------------|-----------------------------|
| `POST /api/files/upload` | `internal/files/handler.go` new handler; behind `requireFilesWrite` middleware | Additive — new route in existing `/api/files/*` namespace |
| `PUT /api/files/write` (save from editor) | Same handler; new verb | Additive |
| `DELETE /api/files/delete` | Same handler | Additive |
| `POST /api/files/rename` | Same handler | Additive |
| `POST /api/files/mkdir` | Same handler | Additive |
| `files.write` capability bit | `internal/daemon` cap token — new bit alongside `files.read` (parallels v3.4 pattern exactly); `HasPerm` comma-split handles it automatically | Additive — add constant + `requireFilesWrite` middleware |
| CodeMirror editor in `FileBrowserTab.tsx` | Replaces v3.4 plain-text `<pre>` rendering; editor mounted in `useEffect`, unmounted on cleanup | No protocol change; same `/api/files/read` endpoint for initial load |
| `editorCmd` (TUI) | New `internal/tui/editor.go`; mirrors `internal/tui/attach.go` structure exactly | No protocol change |
| Remote write parity | Extends daemon proxy `/api/files/remote/{sid}/{op}` to forward write verbs | Additive to existing proxy pattern (v3.4 REMOTE-01..05) |
| TD-4 (`FileBrowserTab.tsx` hardening) | Existing component; source-inspection test cleanup | No new deps |
| TD-5 (`ExchangeJoinCode` shim cleanup) | Existing Wails binding | No new deps |

---

## Installation Summary

```bash
# Frontend (Vite-bundled build deps — NOT served to web terminal page; no vendor_drift_test.go changes)
cd frontend/

# Editor core + curated language packs + theme
pnpm add codemirror@6.0.2 \
         @codemirror/lang-javascript@6.2.5 \
         @codemirror/lang-python@6.2.1 \
         @codemirror/lang-go@6.0.1 \
         @codemirror/lang-markdown@6.5.0 \
         @codemirror/lang-json@6.0.2 \
         @codemirror/lang-yaml@6.1.3 \
         @codemirror/lang-css@6.3.1 \
         @codemirror/lang-html@6.4.11 \
         @codemirror/lang-rust@6.0.2 \
         @codemirror/lang-cpp@6.0.3 \
         @codemirror/language-data@6.5.2 \
         @codemirror/theme-one-dark@6.1.3

# No new Go deps. Write operations use os.Root (stdlib Go 1.24+), mime/multipart (stdlib), net/http.MaxBytesReader (stdlib).
# No go.mod changes needed beyond existing deps.
```

No `vendor_drift_test.go` changes. No `web/vendor/` changes. No `embed.go` changes. No CSP changes.

---

## Version Compatibility

| Package | Version | Compatible With | Notes |
|---------|---------|-----------------|-------|
| `codemirror` | 6.0.2 | React 19, Vite 8, TypeScript 5.9 | Framework-agnostic; wired via `useEffect` — no React peer dep |
| `@codemirror/lang-*` | see table above | `codemirror@6.x` | All `@codemirror/*` packages use unified v6 versioning |
| `@codemirror/language-data` | 6.5.2 | `codemirror@6.x` | Uses dynamic `import()` — Vite code-splits automatically |
| `@codemirror/theme-one-dark` | 6.1.3 | `codemirror@6.x` | Pure CSS injection via `'unsafe-inline'` — already permitted |
| `tea.Exec` / `tea.ExecCommand` | `charm.land/bubbletea/v2 v2.0.6` | Already in go.mod | API confirmed in use in `internal/tui/attach.go` and `internal/tui/update.go` |
| `os.Root` write methods (`Create`, `Mkdir`, etc.) | Go 1.26.3 | Go 1.24+ | All write methods available |
| `mime/multipart` | stdlib | All Go versions | `MaxBytesReader` pattern required; CVE GO-2024-2599 fixed in Go 1.22+ |

---

## Sources

- [CodeMirror bundle size docs](https://codemirror.net/examples/bundle/) — ~400 KB unminified → ~135 KB gzipped with minification; verified via Context7 `/codemirror/website`
- [Monaco worker-src blob CSP issue — payloadcms/payload#10229](https://github.com/payloadcms/payload/issues/10229) — confirmed `worker-src blob:` required
- [Monaco CSP issues — keycloak/keycloak#32901](https://github.com/keycloak/keycloak/issues/32901) — confirmed `worker-src blob:` required; recommends replacing Monaco with CSP-compliant alternative
- [Monaco 0.55 bundle size — monaco-editor#5154](https://github.com/microsoft/monaco-editor/issues/5154) — ts.worker.js 6.68 MB, main bundle 3.88 MB
- [Sourcegraph migration blog: Monaco to CodeMirror](https://sourcegraph.com/blog/migrating-monaco-codemirror) — 43% JS reduction; Monaco alone was 2.4 MB of search page JS; HIGH confidence
- [Vite 8 Monaco Oxc minifier regression — vitejs/vite#22009](https://github.com/vitejs/vite/issues/22009) — additional Vite 8 incompatibility
- [CodeMirror CSP style-src issue — codemirror/dev#395](https://github.com/codemirror/dev/issues/395) — inline style injection confirmed; covered by existing Amendment 1
- [CodeMirror read-only docs — codemirror.net/examples/readonly](https://codemirror.net/examples/readonly/) — `EditorView.editable.of(false)` + `EditorState.readOnly.of(true)` pattern; Context7 verified
- [Bubble Tea v2 `tea.Exec` API — pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbletea) — Context7 confirmed; `tea.Exec` is v2 name; in production use in `internal/tui/update.go`
- [Go mime/multipart pkg.go.dev](https://pkg.go.dev/mime/multipart) — stdlib sufficiency confirmed; security limits documented
- [Go os.Root blog post](https://go.dev/blog/osroot) — `Create`, `Mkdir`, `OpenFile`, `Remove`, `Rename` all documented for Go 1.24+
- npm registry (verified via `npm view`): `codemirror@6.0.2`, `@codemirror/view@6.43.1`, `@codemirror/state@6.6.0`, `@codemirror/language-data@6.5.2`, `shiki@4.2.0`, `monaco-editor@0.55.1`
- `/Users/ken/dev/agenthub/go.mod` — confirmed `charm.land/bubbletea/v2 v2.0.6`, `go 1.26.3`
- `/Users/ken/dev/agenthub/frontend/package.json` — confirmed React 19.2.4, Vite 8, existing deps
- `/Users/ken/dev/agenthub/internal/webserver/csp_mw.go` — confirmed existing CSP: `style-src 'self' 'unsafe-inline'` already set; no `worker-src` directive
- `/Users/ken/dev/agenthub/internal/tui/attach.go` + `update.go` — confirmed `tea.Exec` + `tea.ExecCommand` pattern in production use

---
*Stack research for: v3.5 write-side file browser + in-app code editor (Issues #63, #64, umbrella #24)*
*Researched: 2026-06-14*
