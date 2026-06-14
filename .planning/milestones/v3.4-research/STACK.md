# Stack Research — v3.4 Read-Only File Browser

**Domain:** Sandboxed filesystem API + desktop/web file browser tab + TUI file listing/preview
**Researched:** 2026-05-20
**Confidence:** HIGH (Go stdlib and major libraries verified against official docs and release pages; React library versions verified against npm/GitHub; confidence notes per item below)

This is a **subsequent-milestone STACK.md** — it does NOT re-survey the React/Wails/Go base stack already validated through v3.3.1. All validated stack items (xterm.js 6, React 19.2.4, Vite 8, Vitest 4, Go 1.26.1, Bubble Tea v2, lipgloss/v2, bubbles/v2, charmbracelet/glamour) are in-place and confirmed via `go list -m all` and `frontend/package.json`. This document covers only the NEW capabilities required for v3.4.

Go version confirmed: **go 1.26.1** — `os.Root` (added in Go 1.24) is fully available.

---

## TL;DR

1. **Go sandboxing**: Use `os.OpenRoot` / `os.Root` (stdlib, Go 1.24+) for traversal-resistant FS access. No new Go module needed for safe path resolution.
2. **MIME detection**: Use `github.com/wailsapp/mimetype` v1.4.1 — already in `go.sum` as a Wails transitive dep; not yet in `go.mod` as a direct dep. Promote to direct. Do NOT add `gabriel-vasile/mimetype` as a separate dep (it is the upstream origin of wailsapp/mimetype).
3. **HTTP Range streaming**: Use `http.ServeContent` (stdlib). `io.ReadSeeker` via `os.Root.Open()` → `*os.File` is sufficient. No new dep.
4. **Markdown rendering (React)**: Add `react-markdown@10.1.0` + `remark-gfm` (GFM tables/strikethrough/task lists). CSP-clean: no `eval`, no inline scripts, virtual DOM only.
5. **Syntax highlighting (read-only preview)**: Add `shiki@4.1.0` with `createJavaScriptRegexEngine()` — avoids Oniguruma WASM, no `'wasm-unsafe-eval'` CSP amendment needed. Use `shiki/core` fine-grained bundle with ~15 curated languages.
6. **Type-ahead filter**: No library. `useMemo` + inline substring match is sufficient for a directory listing (typically <10K entries). Optionally `fuzzysort@3.1.0` (~3 KB gzipped) if fuzzy matching is requested.
7. **TUI markdown/file preview**: `charmbracelet/glamour@v0.8.0` — already a transitive dep (`go list -m all` confirmed). Promote to direct dep. Renders markdown with ANSI styling.
8. **TUI file listing**: `charm.land/bubbles/v2@v2.1.0` `filepicker` component — already in `go.mod` as a direct dep (it's the Charm v2 ecosystem module used by the existing TUI).
9. **Fuzz testing**: `testing.F` (stdlib). No external fuzz library needed. Seed corpus of known path-traversal payloads in `testdata/fuzz/FuzzSandboxPath/`.

**CSP status**: Zero new CSP amendments required. All choices are `script-src 'self'`-clean.

---

## Recommended Stack

### Core Technologies (in-place — for integration reference only)

| Technology | Version | Relevant to v3.4 |
|------------|---------|-----------------|
| Go | 1.26.1 | `os.Root` available (added Go 1.24); `http.ServeContent` for Range |
| `charm.land/bubbles/v2` | v2.1.0 | `filepicker` component for TUI Files view |
| `charmbracelet/glamour` | v0.8.0 | Markdown rendering in TUI (transitive dep, promote to direct) |
| `github.com/wailsapp/mimetype` | v1.4.1 | MIME detection (transitive dep via Wails, promote to direct) |
| React | 19.2.4 | `FileBrowserTab.tsx` — already in use |
| Vite + pnpm | 8.x | Build tooling — no change |

### New Go Dependencies (v3.4 adds)

| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| `github.com/wailsapp/mimetype` | v1.4.1 | MIME type detection from magic bytes | Already an indirect dep via Wails; promoting to direct. Detects 200+ MIME types vs stdlib's ~15. MIT license. Forked from `gabriel-vasile/mimetype` — do not add both. |

No other new Go deps. Everything else uses stdlib (`os.Root`, `http.ServeContent`, `testing.F`) or already-direct deps (`charm.land/bubbles/v2`, `charmbracelet/glamour`).

### New Frontend Dependencies (v3.4 adds)

| Library | Version | Purpose | Bundle size (min+gz) | CSP impact |
|---------|---------|---------|---------------------|------------|
| `react-markdown` | 10.1.0 | Markdown rendering in preview pane | ~43 KB | None — virtual DOM, no eval, no inline scripts |
| `remark-gfm` | latest (^4) | GFM extensions: tables, strikethrough, task lists, autolink | ~8 KB | None |
| `shiki` | 4.1.0 | Read-only syntax highlighting in preview pane | ~12 KB core + ~20–40 KB per language (lazy) | None with JS engine — see note below |

**shiki CSP note**: Shiki's default Oniguruma WASM engine requires `'wasm-unsafe-eval'` in CSP (confirmed via GitHub issue vercel/streamdown#384). Using `createJavaScriptRegexEngine()` (available since Shiki v1.x, confirmed stable in v4.x) switches to native JS `RegExp` — no WASM, no eval, no new CSP amendment. This is the mandatory configuration for AgentHub.

**remark-gfm version note**: `remark-gfm@^4` is the ESM-only v4 release compatible with `react-markdown@10`. Verify with `pnpm add remark-gfm@^4`.

### Optional Frontend Dependency (if fuzzy type-ahead is requested)

| Library | Version | Purpose | Bundle size | CSP impact |
|---------|---------|---------|------------|------------|
| `fuzzysort` | 3.1.0 | Fuzzy match with score + highlight | ~3 KB min+gz | None |

Default recommendation: skip `fuzzysort` for v3.4. `useMemo` + `filename.toLowerCase().includes(query)` is sufficient for a directory listing and avoids a dep. If users request fuzzy matching (SublimeText-style), add `fuzzysort@3.1.0` (MIT, zero deps, 3 KB). Do not use the `react-fuzzysort` wrapper — wire directly.

### Development Tools (unchanged)

| Tool | Purpose | Notes for v3.4 |
|------|---------|----------------|
| `testing.F` (stdlib) | Path-traversal fuzz testing | Seed corpus in `testdata/fuzz/FuzzSandboxPath/` |
| Playwright (Chromium + Firefox + WebKit) | e2e for `FileBrowserTab` | Already in use via `@playwright/test^1.59.1` |
| Vitest 4 + jsdom 29 | Frontend unit tests | `?raw` source-inspection for React components without DOM |

---

## Go Sandboxing: `os.Root` vs Alternatives

### Recommendation: `os.Root` (stdlib)

**Go 1.24** added `os.OpenRoot(dir string) (*os.Root, error)` and `os.OpenInRoot(dir, path string) (*os.File, error)`. Since the project uses Go 1.26.1, these are fully available.

`os.Root` provides:
- Blocks `../` components that resolve outside the root
- Blocks absolute paths (e.g., `/etc/passwd`)
- Prevents symlinks escaping the root (RESOLVE_BENEATH semantics)
- Eliminates TOCTOU races via kernel-level atomic operations (openat2 on Linux)
- Windows: blocks reserved device names (`NUL`, `COM1`, `COM2`, etc.)

**Usage pattern for the FS handler:**

```go
// Open sandbox root once per request, scoped to session WorkDir
root, err := os.OpenRoot(session.WorkDir)
if err != nil { ... }
defer root.Close()

// Safe: cannot escape WorkDir regardless of what untrustedPath contains
f, err := root.Open(untrustedPath)
```

**Windows gotcha — EvalSymlinks link-type bug (Go issue #71165):** `filepath.EvalSymlinks` on Windows does not respect the Windows distinction between file-symlinks and directory-symlinks (open as of 2026-05-20, labeled NeedsDecision/Backlog). This is NOT a concern for `os.Root` — `os.Root.Open()` uses the kernel-level `openat2`/`NtCreateFile` path which does not use `EvalSymlinks` internally. Do NOT use `filepath.EvalSymlinks` as a pre-check on Windows; use `os.Root` exclusively.

**Additional validation layer** (defense in depth — not a replacement for `os.Root`):
```go
func validateRelativePath(p string) error {
    if p == "" { return errors.New("empty path") }
    if strings.ContainsRune(p, 0) { return errors.New("null byte in path") }
    // filepath.Clean is a pre-screen only; os.Root is the real guard
    cleaned := filepath.Clean(p)
    if filepath.IsAbs(cleaned) { return errors.New("absolute path rejected") }
    // Windows drive letters and UNC
    if len(cleaned) >= 2 && cleaned[1] == ':' { return errors.New("drive letter rejected") }
    if strings.HasPrefix(cleaned, `\\`) { return errors.New("UNC path rejected") }
    return nil
}
```

### Why not `github.com/cyphar/filepath-securejoin` (v0.6.1)?

- Legacy `SecureJoin` API is explicitly documented as "fundamentally unsafe against TOCTOU attacks" by its own maintainer.
- The modern `pathrs-lite` sub-API only works on Linux — Windows is unsupported.
- `os.Root` is the stdlib solution that supersedes it for this use case.
- `filepath-securejoin` is dual-licensed BSD-3/MPL-2.0. MPL-2.0 has a weak copyleft clause — avoid introducing it unless strictly necessary.

### Why not `go-billy/v5` (already in go.sum as a transitive dep)?

- `go-billy/v5 ChrootOS` had a critical path traversal CVE (CVE-2023-49569) demonstrating that the abstraction is easy to misuse.
- `go-billy` is designed for virtual filesystems (git operations), not syscall-level sandboxing.
- No advantage over `os.Root` for this use case.
- Do NOT use for sandboxing.

---

## HTTP Range Streaming: `http.ServeContent`

### Recommendation: stdlib `http.ServeContent`

`http.ServeContent(w, r, name, modtime, content)` handles:
- `Range` header parsing and partial-content (206) responses
- Multi-range (multipart/byteranges) automatically
- `If-Modified-Since`, `If-None-Match`, `ETag` conditional requests
- Content-Type sniffing from the `name` parameter
- Seek-to-end for Content-Length determination

`os.Root.Open()` returns `*os.File` which implements `io.ReadSeeker` — pass directly to `http.ServeContent`. No hand-rolled partial content handling needed.

**Pattern:**
```go
// GET /api/files/read?path=<relative>
root, _ := os.OpenRoot(session.WorkDir)
defer root.Close()
f, err := root.Open(relPath)
if err != nil { http.Error(w, "not found", 404); return }
defer f.Close()
stat, _ := f.Stat()
http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
```

**5 MB read cap for preview pane**: The `/api/files/read` endpoint should enforce a content-length check from `stat.Size()` before serving. For the preview pane, return `400` if `stat.Size() > 5*1024*1024` with a `"file too large for preview"` JSON error. The download path (no preview limit) uses the same endpoint with the client sending `Accept: application/octet-stream` or an explicit `?download=1` flag — serve via `ServeContent` without the size check in that case.

No new Go dep. `net/http.ServeContent` is sufficient.

---

## MIME Detection

### Recommendation: `github.com/wailsapp/mimetype` v1.4.1 (promote to direct dep)

`github.com/wailsapp/mimetype` is already in `go.sum` as an indirect dep from Wails. It is a fork of `gabriel-vasile/mimetype` (MIT) maintained by the Wails team. The upstream `gabriel-vasile/mimetype@v1.4.13` (latest as of 2026-02-01) has the same API — promoting `wailsapp/mimetype` to a direct dep avoids duplicating the same library under two module paths.

**Why not `http.DetectContentType`?**
- Only examines first 512 bytes
- Returns only ~15 MIME types
- Cannot distinguish `.json`, `.csv`, `.ts`, `.go`, `.md` from `text/plain`
- Cannot detect Office formats (`.docx`, `.xlsx`, `.pptx`)

**wailsapp/mimetype** detects 200+ MIME types via magic bytes, handles text/binary differentiation, and has no external deps.

**Usage pattern:**
```go
// Stat first (cheap), then detect on first N bytes
f, _ := root.Open(relPath)
mtype, err := mimetype.DetectReader(f)
// Reset read position before ServeContent
f.Seek(0, io.SeekStart)
```

**JS side (browser)**: Do not use `file-type` npm package — it uses WASM and is over-engineered for this use case. The Go daemon returns `Content-Type` in the HTTP response; the React frontend reads `response.headers.get('content-type')` to decide preview mode (text/markdown/image/binary). No new JS MIME library needed.

---

## Markdown Rendering (React Preview Pane)

### Recommendation: `react-markdown@10.1.0` + `remark-gfm@^4`

**`react-markdown@10.1.0`** (latest stable, released March 7, 2025):
- Renders markdown as React virtual DOM — no `dangerouslySetInnerHTML`
- No `eval`, no inline scripts — fully compatible with `script-src 'self'`
- Safe by default: HTML is stripped unless `rehype-raw` is explicitly added (do NOT add `rehype-raw` for v3.4 — no HTML passthrough needed)
- ~43 KB minified+gzipped (base, per community measurement; includes remark + unified)
- ESM-only; Vite handles tree-shaking correctly

**`remark-gfm@^4`**: Adds GitHub Flavored Markdown extensions (tables, strikethrough, task lists, autolinks). ~8 KB. Required for displaying Claude Code's markdown output correctly.

**CSP impact: none.** Confirmed: react-markdown renders via React's virtual DOM reconciler, not via DOM innerHTML or script injection.

**Vendoring note**: react-markdown is a Vite-bundled frontend dep only — it is NOT served to the web-served terminal page. The file browser tab is a desktop+web feature that uses the React build path, not the `web/vendor/xterm/` vendoring pipeline. No `vendor_drift_test.go` changes needed for this dep.

**Installation:**
```bash
# In frontend/
pnpm add react-markdown@10.1.0 remark-gfm@^4
```

**Usage pattern:**
```tsx
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

<ReactMarkdown remarkPlugins={[remarkGfm]}>
  {markdownContent}
</ReactMarkdown>
```

### Why not `marked` or `micromark` directly?

- `marked` requires `DOMPurify` or explicit sanitization to be safe — extra dep, extra risk.
- `micromark` is the underlying parser for `react-markdown`; using it directly means writing your own React renderer.
- `react-markdown` gives the right abstraction at the right layer.

---

## Read-Only Syntax Highlighting (Code Preview)

### Recommendation: `shiki@4.1.0` with JavaScript RegExp engine + fine-grained bundle

**Version**: 4.1.0 (latest stable, confirmed May 2026)

**Critical configuration — JavaScript engine, not WASM:**

Shiki's default Oniguruma engine uses WebAssembly and requires `'wasm-unsafe-eval'` in CSP (confirmed via Vercel/streamdown#384). This would be a **new CSP amendment** — a red flag per the quality gate.

Shiki v1+ provides `createJavaScriptRegexEngine()` from `shiki/engine/javascript` which uses native `RegExp` — no WASM, no eval, no CSP amendment needed.

```typescript
import { createHighlighterCore } from 'shiki/core';
import { createJavaScriptRegexEngine } from 'shiki/engine/javascript';

const highlighter = await createHighlighterCore({
  themes: [import('shiki/themes/tokyo-night')],  // matches TUI palette
  langs: [
    import('shiki/langs/typescript'),
    import('shiki/langs/javascript'),
    import('shiki/langs/go'),
    import('shiki/langs/python'),
    import('shiki/langs/markdown'),
    import('shiki/langs/json'),
    import('shiki/langs/yaml'),
    import('shiki/langs/bash'),
    import('shiki/langs/css'),
    import('shiki/langs/html'),
  ],
  engine: createJavaScriptRegexEngine(),  // NO WASM, NO unsafe-eval
});
```

**Bundle approach**: Use `shiki/core` (fine-grained bundle) — import only the languages needed for the v3.4 file browser preview. Shiki's full bundle is 6.4 MB minified; core + 10 languages is approximately 200–300 KB minified before Vite tree-shaking. Load the highlighter lazily (dynamic `import()`) — only needed when the preview pane renders a code file.

**Theme**: Use `tokyo-night` to match the existing TUI and GUI TokyoNight palette.

**CSP impact: none.** JavaScript engine confirmed CSP-clean under `script-src 'self'`.

**Vendoring note**: Shiki is a Vite-bundled frontend dep, NOT served to the web terminal page. No `vendor_drift_test.go` entry needed.

**Installation:**
```bash
pnpm add shiki@4.1.0
```

### Why not Prism or highlight.js?

| Library | Problem |
|---------|---------|
| `highlight.js` (full) | ~1 MB unminified for all languages; custom language builds reduce it but add build complexity |
| `react-syntax-highlighter` | Wraps both prism and hljs; heavy; complex bundle (esbuild issue #1836: tons of unnecessary files) |
| Prism | Older ecosystem; Prism v2 is in development but not stable; smaller community momentum than Shiki |

Shiki v4 with fine-grained bundle + JS engine is the current best-practice for CSP-strict apps (confirmed via Nuxt blog post "Evolution of Shiki v1.0" and streamdown fix).

---

## Type-Ahead Filter

### Recommendation: `useMemo` + substring match (no library for v3.4)

A directory listing is bounded by filesystem conventions: even a large `node_modules` directory is well under 50K entries. A simple `useMemo` substring filter is O(n) and renders imperceptibly fast for ≤50K entries.

```typescript
const filtered = useMemo(() =>
  query
    ? entries.filter(e => e.name.toLowerCase().includes(query.toLowerCase()))
    : entries,
  [entries, query]
);
```

**If fuzzy matching is requested post-v3.4**: Add `fuzzysort@3.1.0` (MIT, zero deps, ~3 KB min+gz). Do NOT add the `react-fuzzysort` wrapper — wire to `fuzzysort.go()` directly in a `useMemo`.

---

## TUI Files View: Bubble Tea Components

### TUI Markdown Preview: `charmbracelet/glamour@v0.8.0` (promote to direct dep)

`charmbracelet/glamour` is already a **transitive dep** (`go list -m all` confirms `github.com/charmbracelet/glamour v0.8.0`). Promoting to a direct dep in `go.mod` requires a single `go get` call — no new binary is downloaded.

Glamour renders markdown to ANSI-styled terminal output:
```go
import "github.com/charmbracelet/glamour"

out, err := glamour.Render(markdownContent, "dark")
// out is an ANSI-escaped string ready to embed in a lipgloss viewport
```

Use `glamour.WithAutoStyle()` so it adapts to the terminal's color profile — consistent with the existing TokyoNight TUI palette.

**v3.4 scope**: Glamour is used only for markdown preview in the TUI read-only preview pane. Plain text files use a raw `viewport` (already in `charm.land/bubbles/v2`). Binary files show a static "Use desktop or web to preview" message.

### TUI Directory Listing: `charm.land/bubbles/v2@v2.1.0` `filepicker`

`charm.land/bubbles/v2` is already a **direct dep** in `go.mod` (confirmed). The `filepicker` sub-package provides:

- Directory navigation with `Up`/`Down`/`PageUp`/`PageDown`/`Back`/`Open` keybindings
- `ShowPermissions`, `ShowSize`, `ShowHidden` display options
- `AllowedTypes` filter (not relevant for v3.4 browse-all mode)
- `HighlightedPath()` returns the currently selected entry path
- `DirAllowed: true`, `FileAllowed: true` for full navigation

**v3.4 customization required**: The stock `filepicker` component allows navigating above its start directory. AgentHub's requirement is "never above session cwd." This requires wrapping the `filepicker` model and intercepting the `Back`/`Open` messages when at the cwd root:

```go
type FilesModel struct {
    cwd    string
    picker filepicker.Model
}

func (m FilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if key, ok := msg.(tea.KeyMsg); ok {
        // Block Back key when picker is already at cwd
        if key.Type == tea.KeyLeft || key.String() == "backspace" {
            if m.picker.CurrentDirectory == m.cwd {
                return m, nil // swallow the Back key
            }
        }
    }
    // ... delegate to picker
}
```

**Type-ahead filter in TUI**: The `bubbles/v2/list` component has built-in filtering via the `/` key — consider using `list.Model` with custom `list.Item` entries instead of `filepicker` if type-ahead is required at the list level. The `filepicker` component does not have built-in type-ahead; a separate `textinput.Model` for filtering is straightforward to add.

### TUI Text Preview: `bubbles/v2/viewport`

The `charm.land/bubbles/v2` `viewport` sub-package provides a scrollable pane — already in use in the existing TUI layout. Use it as the read-only preview pane for text and markdown. No new component needed.

---

## Fuzz Testing for Path Traversal

### Recommendation: `testing.F` (stdlib)

Go's built-in fuzzer (`go test -fuzz=FuzzSandboxPath -fuzztime=60s`) is sufficient. No external fuzz library needed.

**Required seed corpus** (place in `internal/daemon/testdata/fuzz/FuzzSandboxPath/`):
```
../etc/passwd
../../etc/shadow
/etc/passwd
....//....//etc/passwd
%2e%2e%2fetc%2fpasswd
..%2Fetc%2Fpasswd
\..\..\Windows\System32
C:\Windows\System32\config\SAM
\\server\share
null\x00byte
COM1
NUL
..\path
../path
```

**Fuzz test structure:**
```go
func FuzzSandboxPath(f *testing.F) {
    // Seed corpus — known path traversal payloads
    f.Add("../etc/passwd")
    f.Add("../../etc/shadow")
    f.Add("/etc/passwd")
    f.Add("C:\\Windows\\System32")
    f.Add("\\\\server\\share")
    f.Add("file\x00with\x00nulls")
    f.Add("COM1")
    
    f.Fuzz(func(t *testing.T, p string) {
        // validateRelativePath should reject all traversal attempts
        // os.Root should never produce a path outside WorkDir
        root, err := os.OpenRoot(t.TempDir())
        require.NoError(t, err)
        defer root.Close()
        
        // Must not panic; may return error
        f, err := root.Open(p)
        if err == nil {
            f.Close()
            // If Open succeeded, verify the resolved path is under root
            // (os.Root enforces this at the kernel level — this is a sanity check)
        }
    })
}
```

**CI gate**: Add `go test -fuzz=FuzzSandboxPath -fuzztime=30s` as a required step in the PR checks for the `internal/daemon` package. The fuzz test is a merge-blocker, not optional.

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Go sandbox | `os.Root` (stdlib) | `cyphar/filepath-securejoin@v0.6.1` | Legacy API is TOCTOU-unsafe; modern API Linux-only; MPL-2.0 license concern |
| Go sandbox | `os.Root` (stdlib) | `go-billy/v5 ChrootOS` | CVE-2023-49569 path traversal; designed for virtual FS, not syscall sandboxing |
| Go sandbox | `os.Root` (stdlib) | `filepath.EvalSymlinks` + prefix check | Two-step is TOCTOU-prone; Windows link-type bug (issue #71165) unresolved |
| MIME detection | `wailsapp/mimetype` (already indirect) | `gabriel-vasile/mimetype@v1.4.13` | Same codebase; adding both creates a duplicate dep under two module paths |
| MIME detection | `wailsapp/mimetype` | `http.DetectContentType` (stdlib) | Only 512 bytes, ~15 types; cannot distinguish JSON/YAML/Go/Markdown from text/plain |
| HTTP Range | `http.ServeContent` (stdlib) | Hand-rolled partial content | ServeContent is battle-tested for multipart range + ETags; no upside to replacing it |
| Markdown (React) | `react-markdown@10` | `marked` | Requires DOMPurify for safe rendering; extra dep; lacks React virtual DOM integration |
| Markdown (React) | `react-markdown@10` | `micromark` directly | Underlying parser without React renderer — requires custom component tree |
| Syntax highlight | `shiki@4.1.0` + JS engine | `shiki@4.1.0` + WASM (default) | WASM engine requires `'wasm-unsafe-eval'` CSP amendment — red flag |
| Syntax highlight | `shiki@4.1.0` | `highlight.js` | Full bundle ~1 MB; custom language builds add build complexity; less active ecosystem |
| Syntax highlight | `shiki@4.1.0` | `react-syntax-highlighter` | Heavy wrapper over prism/hljs; esbuild issue #1836 (unnecessary files); no tree-shaking |
| Type-ahead | `useMemo` + substring | `fuzzysort@3.1.0` | Not needed for v3.4; add only if user requests fuzzy matching |
| TUI markdown | `glamour@v0.8.0` (already dep) | Hand-rolled ANSI | Glamour is already in go.sum and supports dark/light auto-detection |
| TUI file list | `bubbles/v2/filepicker` | Custom lipgloss list | filepicker already handles keybindings, permissions, size display; custom wrapper is minimal |
| JS MIME (browser) | `response.headers.get('content-type')` | `file-type` npm package | file-type uses WASM; over-engineered; daemon already returns correct Content-Type |
| Go fuzz | `testing.F` (stdlib) | External fuzz library (dvyukov/go-fuzz) | stdlib fuzzer is sufficient; dvyukov/go-fuzz predates Go 1.18 native fuzzing and is unmaintained |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `shiki` with default (Oniguruma) engine | Triggers `'wasm-unsafe-eval'` CSP violation — confirmed via vercel/streamdown#384 | `shiki` with `createJavaScriptRegexEngine()` |
| `rehype-raw` plugin with `react-markdown` | Enables raw HTML passthrough — creates XSS risk in file preview pane | Omit `rehype-raw` entirely for v3.4 |
| `filepath.EvalSymlinks` as primary sandbox guard | TOCTOU-prone on all platforms; Windows link-type bug (Go issue #71165) unresolved | `os.Root.Open()` exclusively |
| `gabriel-vasile/mimetype` as a direct dep | Duplicates `wailsapp/mimetype` which is already in go.sum | `wailsapp/mimetype` promoted to direct dep |
| CDN-served Shiki or react-markdown | Violates vendor-only discipline (`vendor_drift_test.go` CI gate); CSP `script-src 'self'` forbids it | Vite-bundled npm deps (these are frontend build deps, not web-served terminal page assets) |
| `@xterm/addon-image` or any terminal addon | These are for the terminal panel, not the file browser | React component tree for `FileBrowserTab.tsx` |
| CodeMirror 6 / Monaco editor | v3.5 decision — editor lib only relevant for write side | Deferred to v3.5 |
| Cloud Commander integration | Locked to native UI — not in scope for any milestone | Native file browser UI |
| Upload / write / rename / mkdir deps | v3.5 scope | Deferred |

---

## Integration Points with Existing Architecture

| New Capability | Where It Hooks In | Protocol Change |
|---------------|-------------------|-----------------|
| `GET /api/files/list` | `internal/webserver` new route, behind cap-token middleware (`files.read` cap bit) | None — same HTTP/JSON over Tailscale TLS |
| `GET /api/files/stat` | Same as above | None |
| `GET /api/files/read` | Same as above; returns file bytes via `http.ServeContent` + `os.Root` | None — adds `Range` / `206` support |
| `files.read` cap bit | `internal/daemon` capability token — default ON for session owner, default OFF for web-share viewers | Additive to existing capability bit system |
| `FileBrowserTab.tsx` | New React tab component, wired to session context menu + Sessions panel | None |
| TUI Files view | New `internal/tui/files.go` model, added to existing two-pane tab bar | None |
| `os.Root` sandbox | `internal/daemon` new `files.go` handler | None |
| `glamour` markdown preview | `internal/tui/files.go` — call `glamour.Render()` on `.md` files in preview pane | None |

**CSP regression risk**: None. All new choices are verified `script-src 'self'`-clean. The existing CSP policy (`script-src 'self'`, `style-src 'self' 'unsafe-inline'`, `'wasm-unsafe-eval'` for addon-image) is unchanged by v3.4.

---

## Installation Summary

```bash
# Go: promote indirect deps to direct (no new binary download)
go get github.com/wailsapp/mimetype@v1.4.1
go get github.com/charmbracelet/glamour@v0.8.0  # already indirect; now direct

# Frontend (Vite-bundled build deps — NOT served to web terminal page)
cd frontend/
pnpm add react-markdown@10.1.0 remark-gfm@^4 shiki@4.1.0

# Optional (only if fuzzy type-ahead requested)
pnpm add fuzzysort@3.1.0
```

No `vendor_drift_test.go` changes. No web/vendor/ changes. No `embed.go` changes. No CSP changes.

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `os.Root` | Go 1.24+ | Go 1.26.1 confirmed — fully available |
| `wailsapp/mimetype@v1.4.1` | All platforms | Already in go.sum as indirect dep |
| `charm.land/bubbles/v2@v2.1.0` | `charm.land/bubbletea/v2@v2.0.6` | Both already direct deps |
| `charmbracelet/glamour@v0.8.0` | Go 1.18+ | Already in go.sum as transitive dep |
| `react-markdown@10.1.0` | React 19.x | ESM-only; compatible with Vite 8 |
| `remark-gfm@^4` | `react-markdown@10.x` | v4 is the ESM version; v3 is CJS-only |
| `shiki@4.1.0` (JS engine) | React 19 / Vite 8 | `shiki/engine/javascript` available since v1.x; no WASM peer dep |
| `shiki/core` fine-grained | Vite 8 tree-shaking | Dynamic `import()` for lazy language loading |

---

## Sources

- [Go 1.24 os.Root blog post](https://go.dev/blog/osroot) — confirms traversal-resistant semantics, TOCTOU elimination, Windows device name blocking
- [Go issue #71165](https://github.com/golang/go/issues/71165) — `filepath.EvalSymlinks` Windows link-type bug (open, unresolved)
- [Gabriel-vasile/mimetype v1.4.13](https://pkg.go.dev/github.com/gabriel-vasile/mimetype) — latest upstream; 200+ MIME types, MIT license
- [cyphar/filepath-securejoin v0.6.1](https://pkg.go.dev/github.com/cyphar/filepath-securejoin) — modern API Linux-only; legacy API TOCTOU-unsafe per maintainer docs
- [go-billy CVE-2023-49569](https://dailycve.com/go-git-go-billy-path-traversal-symlink-following-cve-2023-49569-critical/) — confirms ChrootOS path traversal risk
- [react-markdown GitHub](https://github.com/remarkjs/react-markdown) — v10.1.0 stable, no dangerouslySetInnerHTML, virtual DOM rendering
- [Shiki JavaScript RegExp engine](https://shiki.style/guide/regex-engines) — native RegExp, no WASM
- [Shiki CSP/eval issue — vercel/streamdown#384](https://github.com/vercel/streamdown/issues/384) — confirms WASM engine requires `wasm-unsafe-eval`; JS engine fixes it
- [Shiki bundle sizes](https://shiki.style/guide/bundles) — full 6.4 MB; core ~12 KB + per-lang imports
- [charm.land/bubbles/v2@v2.1.0 filepicker](https://pkg.go.dev/charm.land/bubbles/v2@v2.1.0/filepicker) — confirmed filepicker component, keybindings, v2.1.0 released Mar 2026
- [charmbracelet/glamour v0.8.0](https://pkg.go.dev/github.com/charmbracelet/glamour@v0.8.0) — ANSI markdown renderer, MIT, already transitive dep
- `go list -m all` — confirmed all charmbracelet/v2 ecosystem, glamour, wailsapp/mimetype as existing deps
- `frontend/package.json` — confirmed React 19.2.4, Vite 8, Vitest 4, Playwright confirmed

---
*Stack research for: v3.4 read-only file browser (Issue #62, v3.4 slice of #64)*
*Researched: 2026-05-20*
