# Feature Research

**Domain:** Read-only file browser tab (desktop + web + TUI) — AgentHub v3.4
**Researched:** 2026-05-20
**Confidence:** HIGH — primary sources are MDN/ARIA spec, charmbracelet official docs, VS Code
documentation, ranger/lf docs; competitor UX observations are MEDIUM (public docs + screenshots).

> Scope: v3.4 ships Issue #62 (read-only file browser tab for GUI + web) and the v3.4 slice of #64
> (TUI browse+preview parity). Write operations, editor integration, and TUI $EDITOR shell-out are
> all v3.5. This research covers only the read-only shape.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features a user expects from any embedded file browser. Missing these makes the tab feel half-shipped.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Directory listing with file/folder distinction** | Every file browser since 1984 shows this; without it users cannot orient themselves | LOW | Icons or glyphs to distinguish directories, files, symlinks; symlinks should show `→ target` in description |
| **Keyboard navigation (arrows, Enter, Backspace)** | Power users — the primary AgentHub audience — expect a keyboard-first interface; mouse-only feels broken | MEDIUM | Up/Down navigate list; Enter opens dir or triggers preview; Backspace/Left goes up one level; Home/End jump; PageUp/PageDown for long dirs |
| **Breadcrumb / path bar** | Users need to know where they are; every file browser (VS Code Explorer, Finder, FileZilla, GitHub) shows the current path | LOW | Clickable segments to navigate up; never allows navigating above session cwd root; display relative path from session cwd |
| **Sort by name / size / mtime** | All three are expected in any list view; size and mtime are used constantly in dev workflows | LOW | Default: name ascending, directories first. Sort header clicks toggle asc/desc. Persistence per-session is nice-to-have not required. |
| **Directories-first ordering** | Every major file browser (VS Code, macOS Finder column/list, Windows Explorer, GitHub) puts directories at top when sorting by name | LOW | Sticky regardless of sort column in v3.4 (directories always precede files within the same sort); can be made optional in v3.5. |
| **Type-ahead filter / search** | Users expect to type to narrow a long directory listing; VS Code Explorer (Ctrl-E / type directly), ranger (/), lf (/), nnn (/) all do this | MEDIUM | See Cmd-F collision note below. Default activation: `/` key (not Cmd-F — Cmd-F is the existing scrollback search bar). Filter scoped to current directory only in v3.4; recursive search is v3.5. |
| **Text file preview** | GitHub web viewer, VS Code, Cyberduck all preview text files inline; this is the core read-only value proposition | MEDIUM | Preview panel for text files up to 5 MB (configurable cap). Display raw text with monospace font. |
| **Markdown rendered preview** | Developers constantly open README.md, CHANGELOG.md, docs — expecting rendered markdown is now standard (GitHub, VS Code, GitLab all render it) | MEDIUM | Render markdown in the preview pane. A lightweight renderer (marked.js or similar, already-vendored or simple) is sufficient; not a full GitHub-flavored pipeline. |
| **Image preview** | VS Code Explorer shows image preview on click; Cyberduck shows thumbnails; GitHub shows inline image preview | MEDIUM | PNG, JPEG, WebP, GIF, SVG (render as `<img>` for raster; inline `<img>` for SVG is fine given CSP is `img-src 'self' data:`). Respect aspect ratio. Size: fit-within pane. |
| **Binary file refusal message** | GitHub, GitLab, VS Code all display an explicit message when a file cannot be previewed; silence is worse than refusal | LOW | See exact copy recommendation in Anti-Features section. |
| **Empty directory state** | Every file browser shows "this folder is empty"; without it users think it's broken | LOW | Simple centered message: "Empty directory" with the current path shown. |
| **Loading / error states** | Remote sessions have latency; network errors happen; permission errors happen | LOW-MEDIUM | Spinner during list fetch; specific error messages for permission-denied, not-found, network error; broken-symlink indicator. |
| **File count / status line** | VS Code status bar shows item count; ranger shows it at bottom; users want context on large directories | LOW | Status line below the list: `n items` or `n directories, m files`. For remote sessions: add `(remote)` indicator. |

### Differentiators (Competitive Advantage)

Features that fit AgentHub's specific context — a session-management app with Tailscale remote access — better than a generic file manager.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Remote session visual indicator** | AgentHub's Tailscale model means the file browser may be talking to a remote daemon; users need to know this is remote so they understand latency is normal | LOW | Badge or label in the path bar / status line: `via tailnet` (or hostname) for remote sessions. Not a blocking indicator — just ambient context. |
| **Lazy-load / streaming for remote dirs** | Remote sessions over the relay have real latency; showing a spinner per-directory-open is better than blocking the whole tab | MEDIUM | Daemon returns listing synchronously but the React component can show a row spinner while the `GET /api/files/list` round-trips. For TUI: show `Loading…` status line while fetching. |
| **Preview size indicator** | When previewing large-ish files (500 KB — 5 MB), show the file size so users know why it loaded slowly; prevents "is it broken?" confusion | LOW | Show `42 KB` or `3.2 MB` in the preview pane header alongside the filename. |
| **"Too large to preview" download affordance** | GitHub shows "View raw" + "Download"; for AgentHub, users should be able to trigger a Range-capable download instead of nothing | LOW | When file exceeds preview cap, show: "File too large to preview (8.4 MB). [Download]" button that triggers the Range-capable `/api/files/read` endpoint. |
| **Capability-gated web-share behavior** | Users who shared their session with `files.read` OFF should see the file browser disabled with an explanation, not a broken 403 | LOW | When a web-share viewer lacks `files.read`, show: "File browsing is not enabled for this shared session." at the tab level. |
| **Broken symlink visibility** | Developer sessions frequently have broken symlinks (virtualenv artifacts, stale build outputs). VS Code Explorer shows broken symlinks dimmed. | LOW | Show broken symlinks with a `⚠ broken symlink` annotation in the description column. Do not follow them (daemon already rejects). |
| **Session cwd as root enforcement** | AgentHub sandboxes to session cwd. Making this visually clear — no ".." above root, breadcrumb stops at root — prevents user confusion when navigation stops | LOW | Breadcrumb top segment is always the session cwd displayed as `~/project` or an abbreviated path. Navigation stops here. Up arrow / Backspace is a no-op at root. |
| **TUI file browser with TokyoNight palette** | GUI/TUI cross-surface parity is a release-blocking contract. The TUI surface needs the same browse+preview capability with consistent visual language. | MEDIUM | Lipgloss-bordered directory listing, TokyoNight palette (matches existing TUI), viewport split with `JoinHorizontal`. No `bubbles/filepicker` (see below). |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Cmd-F to open filter** | Users already use Cmd-F universally for "find in this view" | Cmd-F is already wired to the xterm.js scrollback search find bar (v3.2 Phase 94). Opening file filter with Cmd-F when the tab's terminal might still intercept it would create a collision; worse, it trains different Cmd-F semantics per tab | Use `/` key (ranger/lf/nnn convention, universally known in TUI tools) or a dedicated filter input that activates only in the file browser tab. If Cmd-F is pressed while the file browser tab is active (not a terminal tab), it is safe to use — but this needs explicit focus-conditional handling mirroring the `SRC-01` pattern from v3.2. Document the collision risk explicitly in the implementation phase. |
| **Write operations (upload / delete / rename / mkdir)** | Users naturally want to manage files, not just read them | Explicitly deferred to v3.5 per milestone scope. Shipping any write UI in v3.4 grows the sandboxed FS API attack surface before the read path has baked under real use | v3.5 |
| **Recursive filter / search** | "Find all files named X in the project" is a common workflow | Requires server-side recursive walk (potentially expensive for large trees over remote relay), response streaming, and a significantly more complex UI to surface results. Read-only v3.4 has no foundation for this yet | Current directory filter only in v3.4. File by file recursion via CLI (`find`, `rg`) is already available in the terminal tabs. |
| **Editor (CodeMirror / Monaco) for read-only display** | "Why not show source with syntax highlighting?" | Editor library decision is deferred to v3.5 (write side). Pulling in a 2 MB editor bundle just for syntax highlighting in v3.4 is premature; plain monospace text suffices for a read-only preview that establishes the API shape | Plain text preview. Syntax highlighting is v3.5 when the editor library decision is made. |
| **bubbles/filepicker as TUI implementation** | Charm provides a filepicker component that seems purpose-built | The `filepicker` component is selection-dialog oriented, not browse-pane oriented. It has no built-in read-only mode, no preview integration, no viewport split, and poor layout integration with existing bordered frames. Deep customization requires effectively rewriting it. | Custom `list.Model` + `viewport.Model` split with `lipgloss.JoinHorizontal` — matches the pattern already established in AgentHub's TUI (two-pane sidebar+content). Custom rendering costs ~200 LoC but gives full control over TokyoNight palette, status line, and breadcrumb display. |
| **Drag-out for download in the TUI** | Drag-and-drop is how desktop file managers transfer files | TUI has no drag-and-drop. Not applicable. | Press Enter on a file in TUI to trigger download / copy path to clipboard if CLI tools allow. |
| **Image preview in TUI** | Users see images in GUI, expect parity | Sixel/chafa rendering in TUI is a separate rendering system; AgentHub's TUI runs in a real terminal (not xterm.js) so sixel could work, but it introduces a complex dependency chain and is not part of the v3.4 scope | TUI shows: "Use the desktop or web app to preview images." Consistent with the existing TUI principle of referring users to the GUI for rich-media features. |
| **Multiple sort column persistence** | "I want mtime sort in this directory and name sort in that one" | Per-directory sort state requires a session-scoped store that doesn't exist yet; adds complexity for marginal gain | One global sort setting per file-browser session tab, remembered in component state only (no persistence in v3.4). |
| **Context menu on right-click (web)** | Right-click should give a context menu with "Download", "Copy path" etc | Context menus add click-event intercept overhead and interact badly with browser native context menus on Tailscale-served web. The download affordance is already in the preview pane. | Explicit "Download" button in the preview pane header. "Copy path" as a toolbar button. No right-click menu in v3.4. |
| **Full path paste to navigate (path bar)** | macOS Finder address bar accepts pasted absolute paths | Sandbox enforcement requires path cleaning, symlink resolution, and prefix-check on every navigation; pasting an absolute path outside the session cwd sandbox must be rejected. The rejection UX (error message vs silent clamp) requires design work. | Breadcrumb segments are clickable for navigation; type-ahead filter handles find-within-dir. Absolute path input is v3.5 with explicit sandbox violation UI designed. |

---

## Feature Dependencies

```
Sandboxed FS API (daemon: GET /api/files/list, /stat, /read)
    └──required by──> FileBrowserTab.tsx (desktop + web)
    └──required by──> TUI Files view
    └──required by──> capability-gated access (files.read bit)

files.read capability bit (v3.1 cap-token system)
    └──required by──> web-share viewer access control
    └──required by──> 403 denial UX at tab level

FileBrowserTab.tsx
    └──depends on──> Sandboxed FS API
    └──uses──> existing Tab pattern (singleton or per-session tab)
    └──uses──> existing BannerStack (for error states)
    └──uses──> existing session context (cwd, session ID, local vs remote)
    └──has──> directory listing pane (left/main)
    └──has──> preview pane (right/inline)
    └──has──> breadcrumb path bar (top)
    └──has──> sort controls (name/size/mtime, asc/desc, dirs-first)
    └──has──> type-ahead filter (/key activation, current dir scope)

Preview pane
    └──depends on──> GET /api/files/read (Range-capable)
    └──requires──> MIME type detection (daemon stat response)
    └──renders──> text (plain monospace), markdown (rendered), images (PNG/JPEG/WebP/GIF/SVG)
    └──refuses──> binary (message + download affordance)
    └──cap-checks──> 5 MB default hard cap (show size in header, offer download above cap)

TUI Files view
    └──depends on──> Sandboxed FS API (same endpoints as GUI)
    └──uses──> bubbles/v2 list.Model (custom delegate, NOT filepicker)
    └──uses──> bubbles/v2 viewport.Model (preview pane)
    └──layout──> lipgloss.JoinHorizontal(lipgloss.Top, listing, preview)
    └──uses──> TokyoNight palette (existing TUI token set)
    └──matches──> GUI feature set: dir listing, type-ahead, breadcrumb, text/md preview
    └──differs──> no image preview, no download gesture (show referral message)

Remote session (tailnet)
    └──all endpoints route through existing relay
    └──adds──> visual indicator in path bar ("via tailnet") — no new protocol changes
    └──adds──> spinner on per-fetch latency (uses existing loading state pattern)
```

### Dependency Notes

- **Cmd-F collision:** The v3.2 find bar (SRC-01..05) uses a focus-conditional pattern: Cmd-F only intercepts when the xterm Terminal instance has focus. The file browser tab has no xterm Terminal, so Cmd-F is technically safe here. However, the file browser should use `/` as its filter activation key to match TUI conventions and avoid any ambiguity. If Cmd-F is later added as a secondary shortcut for filter, it must be gated on `fileBrowserTabActive && !xterm.hasFocus`.

- **bubbles/v2 vs v1:** AgentHub already uses `charmbracelet/bubbles/v2` (confirmed in PROJECT.md tech stack). The `list.Model` in v2 requires getter/setter methods, not exported fields. The custom delegate pattern (implementing `list.ItemDelegate` with `Render`, `Height`, `Spacing`, `Update`) is the right integration point for custom file entry rendering with icons, sizes, and symlink annotations.

- **viewport for preview:** `bubbles/v2/viewport` handles scrollable content with `SetContent(string)`, default vim-style keybindings (j/k, Ctrl-d/u, PageUp/PageDown), and composable with `lipgloss.JoinHorizontal`. For the TUI preview pane this is the correct primitive — not a custom string printer.

- **CSP:** The existing `script-src 'self'` and `img-src 'self' data:` (verify current policy) covers image preview via `<img src="blob:">` or data-URLs derived from the `/api/files/read` response. No new CSP carve-outs needed if images are fetched via `fetch()` and converted to object URLs in-browser. Confirm this in Phase 1 of v3.4.

---

## Layout Pattern Recommendation

### Why NOT tree + list (VS Code Explorer style) inside the tab

AgentHub already has a left sidebar. Adding a second tree pane inside the file browser tab creates a dual-sidebar visual: `[AgentHub sidebar] | [file tree pane] | [file list pane]`. This is visually crowded and creates a conceptual mismatch — the AgentHub sidebar is for app-level navigation (sessions, settings) while the file tree would be for within-session navigation. Users would need to track two distinct navigation hierarchies simultaneously.

VS Code can do this because its sidebar IS the file tree — there is no separate app-level sidebar above it. AgentHub does not have that luxury.

### Why NOT miller columns (macOS Finder columns view)

Miller columns work well for deep hierarchies where you explore multiple branches simultaneously. File trees inside an AI session's working directory are typically shallow (2-4 levels) and users usually know roughly what they are looking for. The columns model also requires significant horizontal space; inside a tab that may be 800–1200px wide, three usable columns would each be ~280px — not enough for typical paths. Miller columns also have notorious horizontal-scroll complexity for deeper trees.

### Recommended: Single-pane list with inline preview (GitHub web viewer model)

The best fit for AgentHub's existing chrome is a **flat single-pane directory listing with a slide-in or side-by-side preview panel**, matching GitHub's file viewer and VS Code's Explorer single-panel-click-to-preview behavior.

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ [AgentHub sidebar]  │  /project/src  >  components  >  Button.tsx  [remote] │
│                     │ ─────────────────────────────────────────────────────  │
│                     │ Name ↑        Size       Modified          Preview     │
│                     │ ─────────────────────────────────────────────────────  │
│                     │ 📁 __tests__             2026-05-18        ──────────  │
│                     │ 📁 utils                 2026-05-17        Button.tsx  │
│                     │ 📄 Button.tsx      3 KB  2026-05-20        ────────── │
│                     │ 📄 index.ts        1 KB  2026-05-19        <preview>  │
│                     │ ────────────────────────              (text/markdown   │
│                     │ 4 items          [/] filter                /image)     │
└──────────────────────────────────────────────────────────────────────────────┘
```

- When the pane is narrow (< ~600px), the preview hides and the list fills the width. Clicking/Enter on a file opens the preview in a slide-over or modal.
- When wide, the list is ~40% of the tab width and the preview fills the rest.
- The breadcrumb path bar is pinned at the top of the tab content area, below the tab chrome.
- No tree pane. Navigation is purely linear: click/Enter into a directory, Backspace/click breadcrumb to go up.

This matches what users expect from a **file browser embedded in a session-management app** (not a standalone file manager): simple, list-based, good for opening specific files you already know are there or doing quick directory exploration.

---

## Browse Interaction Details

### Keyboard Navigation (GUI + Web)

The file list must behave as an ARIA `tree` (folders with children) or `grid`/`listbox` (flat list) — for a flat single-pane model, `role="listbox"` or `role="grid"` is the correct ARIA pattern (not `role="tree"` which implies a persistent visible hierarchy). Given the flat navigation model:

| Key | Behavior |
|-----|----------|
| Up / Down arrows | Move selection |
| Enter | Open directory (navigate into) / open preview for file |
| Backspace | Navigate up one level (no-op at root) |
| Home / End | Jump to first / last item |
| PageUp / PageDown | Scroll by page |
| `/` | Activate type-ahead filter input |
| Escape | Clear filter if active; otherwise does nothing |

**Type-ahead filter:** Activated by `/`. As the user types, the list narrows to items whose name contains the typed string (case-insensitive substring, not fuzzy — predictable for this context). Results are current directory only. The filter state is shown in the status line: `Filtering: "comp"` with item count. ESC clears.

**Single-click vs double-click semantics:** Single click selects (shows preview). Double-click opens directory. This matches VS Code Explorer and GitHub. Rationale: double-click matches OS file browser conventions for "open"; single-click for preview matches VS Code's "peek" behavior.

**Drag-out:** Not in scope for v3.4 (web security model for drag-out-to-download is complex; the download button in the preview pane covers the use case).

---

## Preview Pane Details

### When to Render What

| File Type | Preview |
|-----------|---------|
| Text (plain, source code, config, .env, .txt, .log) | Raw monospace text, no syntax highlighting in v3.4 |
| Markdown (.md, .markdown, .mdx) | Rendered markdown (use existing or vendored renderer) |
| Images (PNG, JPEG, WebP, GIF, SVG) | `<img>` element with `object-fit: contain`, max-height of preview pane. Size displayed in header. |
| Binary (detected by MIME type or null-byte heuristic in daemon) | Refusal message + download button (see copy below) |
| File > 5 MB (default cap, configurable) | "File too large to preview" message + download button |
| Empty file | "Empty file (0 bytes)" |
| Symlink to file | Preview the target, with a header note: `→ actual/path` |
| Broken symlink | "Broken symlink: target does not exist" |

### Binary / Too Large Copy

Both cases should be informative and offer a next action:

**Binary file:** "This file cannot be previewed (binary content). [Download]"
- Matches GitHub's "Sorry, we can't display non-text files inline." pattern but adds the download CTA.

**Too large:** "This file is too large to preview (12.3 MB, limit is 5 MB). [Download]"
- Shows actual size and current limit so the user understands the constraint.

### Image Preview Sizing

- `object-fit: contain` inside the preview pane bounds.
- Preserve aspect ratio. No upscaling beyond natural size.
- Display dimensions below the image: `1920 × 1080 px — 245 KB`.
- For SVGs: render with `<img>` (not inline SVG injection, which would be an XSS risk even for internal files). CSP `img-src 'self' blob:` covers this if served via the files endpoint.

### Preview Pane in TUI

The TUI cannot render images (no sixel for this path — see anti-features). It renders:
- Text and markdown: plain text in a `viewport.Model` (markdown stripped of formatting, or rendered as lipgloss-styled plain text for headings/emphasis — a simple pass over the content).
- Binary / too large: "Use the desktop or web app to preview this file type."
- Status line shows: relative path + file type + size.

---

## Sort and Filter Defaults

| Column | Default | Toggle Behavior |
|--------|---------|-----------------|
| Name | Ascending (A→Z) | Click/key: toggle asc/desc |
| Size | Descending on first click | Toggle asc/desc |
| Modified (mtime) | Descending on first click (newest first) | Toggle asc/desc |
| Directories-first | Always ON in v3.4 | Not user-toggleable in v3.4 (v3.5) |

**Filter scope:** Current directory only. Not recursive.

**Filter activation:** `/` key. This avoids the Cmd-F collision with xterm.js scrollback search. The bubbles `list.Model` uses `/` as its default filter key natively — this aligns well for TUI consistency.

---

## Path Bar / Breadcrumb

- Segments are clickable (navigate up to that level).
- The root segment is the session cwd, displayed as a shortened form: `~/project` or `/home/user/project`. It is not clickable (already at root).
- When user attempts to navigate above root (via Backspace at root, or clicking root breadcrumb): no-op. Root breadcrumb is not a link, it is plain text.
- No free-form path input in v3.4. Absolute path paste is an anti-feature this milestone (see above). The breadcrumb is display-only + clickable segments only.
- For remote sessions: show `(remote: hostname)` or `via tailnet` as a non-clickable badge at the right end of the path bar.

---

## Remote Session Handling

- **Visual indicator:** A `remote` badge in the path bar or status line showing the session hostname. Not a blocking UI — just ambient context. Matches how the CLI attach bar shows `hostname` already.
- **Latency:** `GET /api/files/list` goes through the existing relay for remote sessions. The React component shows a per-directory spinner (not a full-tab loading state) while waiting. This matches how RemoteSessionsPanel handles its 30s auto-refresh: data shows while stale until fresh.
- **Large preview downloads over relay:** The daemon streams the file via Range-capable `GET /api/files/read`. The preview pane should show a "Downloading…" progress indicator while the response streams, especially for files approaching the 5 MB cap. A simple progress bar or "Loading (2.1 MB / 5 MB)…" line is sufficient.
- **Network error:** Show in the listing pane: "Could not load directory — network error. [Retry]". Use BannerStack for persistent network error messaging, not an inline replaced-list message.

---

## Empty / Loading / Error States

| State | Display |
|-------|---------|
| **Loading directory** | Spinner row replacing the list; "Loading…" in status line |
| **Empty directory** | Centered: "This directory is empty." (with the path in the breadcrumb for context) |
| **Permission denied** | "Permission denied: cannot list this directory." with the path. No retry affordance (it won't change without write access). |
| **Not found (path deleted mid-session)** | "Directory not found. It may have been moved or deleted. [Go to root]" |
| **Network error (remote session)** | BannerStack error: "Lost connection to remote session. [Retry]" |
| **Broken symlink in listing** | Row shows `⚠ broken symlink` annotation in dimmed text. Still selectable; preview pane shows "Broken symlink: target does not exist." |
| **Empty file** | Preview pane: "Empty file (0 bytes)." |
| **File read error** | "Could not read this file — permission denied." or generic "read error." |

---

## TUI File Browser Patterns

### Architecture Decision: Custom list.Model, Not bubbles/filepicker

The `bubbles/filepicker` component (v2) is a selection-dialog pattern: its core abstraction is "pick a file and return the selected path." It has no built-in read-only browse mode, no viewport split, no preview integration, and its styling is constrained to predefined `Styles` struct fields. For AgentHub's use case — a persistent tab with list+preview split — custom is the right choice.

**Recommended TUI architecture:**

```
TUI Files View (tea.Model)
├── list.Model (custom ItemDelegate)
│     └── FileItem{name, isDir, size, mtime, isSymlink, isBroken}
│     └── FilterValue() = name (fuzzy match via built-in list filter)
│     └── Delegate.Render: icon + name + size + mtime, TokyoNight colors
│     └── Default keys: up/down/pgup/pgdown/home/end + "/" filter
├── viewport.Model (preview pane)
│     └── SetContent(previewText) — plain text or stripped markdown
│     └── Keys: j/k, Ctrl-d/u, PageUp/PageDown (viewport defaults)
├── breadcrumb []string (current path components)
├── focused: "list" | "preview"
└── Tab key: toggle focus between list and preview panes

View():
    left  := lipgloss.NewStyle().Border(...).Width(leftW).Render(m.list.View())
    right := lipgloss.NewStyle().Border(...).Width(rightW).Render(m.preview.View())
    body  := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
    return lipgloss.JoinVertical(lipgloss.Left, breadcrumbBar, body, statusLine)
```

### Split Pane Width Allocation

For terminals wider than ~120 columns: 40% list / 60% preview.
For terminals 80–119 columns: 50% / 50%.
For terminals < 80 columns: full-width list, no preview pane (show preview on Enter in a modal overlay using the existing TUI overlay pattern).

### Preview in TUI

- Text files: pass through `viewport.SetContent()`. Use `lipgloss.NewStyle().Foreground(tokyonightFg)` for the content.
- Markdown: render as plain text with simple styling: `#` headings become bold, `**` becomes bright. No full markdown-to-ANSI pipeline needed. An `ansi`-aware string formatter is sufficient.
- Binary / image: "Use the desktop or web app to preview this file type." centered in the viewport.
- Large files: "File too large to preview in TUI (12 MB)." — no download affordance in TUI (no system browser integration from TUI context).

### TUI Filter

The `list.Model` has built-in filter with `/` activation. Use it directly — no need to reimplement. Filter state is shown in the list's built-in filter status bar. This matches the GUI's `/` activation choice, giving consistent UX across surfaces.

---

## Accessibility (GUI + Web)

### Required for WCAG AA

| Requirement | Implementation |
|-------------|----------------|
| **Keyboard-only operation** | Full keyboard navigation with the keybindings in the table above. No mouse-only affordance. |
| **ARIA landmark for file list** | `role="region"` with `aria-label="File browser"` wrapping the entire tab content. |
| **List semantics** | The flat listing: `role="listbox"` with `aria-label="Directory contents"`. Each item: `role="option"`. Selected item: `aria-selected="true"`. |
| **Preview region** | `role="region"` with `aria-label="File preview"`. Dynamic content: `aria-live="polite"` so screen readers announce file name / type when preview changes. |
| **Breadcrumb nav** | `<nav aria-label="Path">` with `<ol>` of `<li>` segments; current location as `aria-current="page"` on the last segment. |
| **Sort button accessibility** | Sort column buttons: `aria-sort="ascending"` / `"descending"` / `"none"`. |
| **Focus management** | When navigating into a directory, focus moves to the first item in the new listing (not back to the breadcrumb). |
| **Error messages** | Error states announced via `role="alert"` or `aria-live="assertive"`. |
| **Selection contrast** | Selected item background must meet 3:1 contrast ratio against the list background. Use TokyoNight selection tokens (already audited in v1.14 Phase 72). |
| **Icon non-text content** | Directory/file icons: `aria-hidden="true"` on the icon, visible text name carries the accessible label. |
| **Type-ahead filter input** | When `/` activates filter: move focus to the filter input element. `aria-label="Filter files"`, `role="searchbox"`. |

### Contrast for Selection States

AgentHub's existing WCAG audit (v1.14 Phase 72) established `#9aa5ce` on dark backgrounds. For file browser selection, use the TokyoNight selection highlight (`#2d4f67` background with `#c0caf5` text) — this is the same selection color used in the existing TUI. Verify 4.5:1 contrast ratio for text on selection background before shipping.

---

## User Expectation Gap: Embedded vs Standalone

Users approaching a file browser **inside a session management app** have different expectations than users of a standalone file manager (Finder, Explorer, Cyberduck):

| Dimension | Standalone file manager | AgentHub file browser tab |
|-----------|-------------------------|--------------------------|
| **Scope expectation** | Full filesystem, mounted volumes, network shares | Just "what is in this session's working directory" — users already understand the session has a cwd |
| **Action expectation** | Copy, move, delete, rename, create | Read, preview, download — users accept read-only when the context is "inspect this session's files" |
| **Navigation depth** | Arbitrary, often with bookmarks | Shallow — 2-4 levels is typical for a project directory |
| **Discoverability** | Primary use case | Secondary use case — the primary value is the terminal; file browser is supporting context |
| **Refresh expectation** | Automatic (filesystem events) | Manual or on-navigate — users in a dev session understand files change; a Refresh button is sufficient |
| **Find/search scope** | Full filesystem search | Current directory filter is sufficient; they have `find`/`rg` in the terminal |
| **Download affordance** | Copy/move to another location | Download to local machine — especially important for remote Tailscale sessions |

The key implication: **the bar is lower than a standalone file manager** but **higher than a tree sidebar** in an IDE. Users expect a clean functional read-only browser, not macOS Finder feature parity.

---

## MVP Definition (v3.4)

### Launch With

- [ ] **Directory listing:** name, size, mtime columns, directories-first, sorted by name ascending by default
- [ ] **Keyboard navigation:** Up/Down/Enter/Backspace/Home/End/PageUp/PageDown
- [ ] **Breadcrumb path bar:** clickable segments, sandboxed to session cwd, remote indicator
- [ ] **Type-ahead filter:** `/` key activation, current directory scope, ESC to clear
- [ ] **Sort by name / size / mtime:** click column header toggles asc/desc
- [ ] **Text preview:** raw monospace, 5 MB cap, size shown in header
- [ ] **Markdown rendered preview:** markdown → HTML render, same 5 MB cap
- [ ] **Image preview:** PNG / JPEG / WebP / GIF / SVG, aspect-ratio preserved
- [ ] **Binary refusal:** explicit message + Download button
- [ ] **Too large refusal:** explicit message with size + cap + Download button
- [ ] **Empty directory state:** "This directory is empty."
- [ ] **Loading / error states:** spinner, permission-denied, not-found, network-error messages
- [ ] **Broken symlink display:** annotated row, preview shows "broken symlink" message
- [ ] **Capability-gate for web-share viewers without files.read:** tab-level message
- [ ] **TUI Files view:** list.Model + viewport.Model split, TokyoNight palette, text/markdown preview, image/binary referral message, type-ahead filter, breadcrumb status line
- [ ] **Cross-surface parity:** GUI + Web + TUI all expose browse+preview (with image delta noted)
- [ ] **ARIA landmarks + keyboard accessibility:** listbox, region, nav breadcrumb, aria-live preview

### Add After Validation (v3.5)

- [ ] **Write operations:** upload, delete, rename, mkdir — v3.5 explicit
- [ ] **Syntax-highlighted preview:** editor library (CodeMirror 6 vs Monaco) decided in v3.5
- [ ] **Recursive filter / file search**
- [ ] **Absolute path input in breadcrumb**
- [ ] **Sort persistence per session**
- [ ] **Auto-refresh on filesystem change (inotify / FSEvents)**
- [ ] **TUI $EDITOR shell-out**
- [ ] **Context menu (right-click)**
- [ ] **Directories-first toggle**

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Directory listing + keyboard nav | HIGH | LOW | P1 |
| Breadcrumb path bar | HIGH | LOW | P1 |
| Type-ahead filter (/ key) | HIGH | MEDIUM (Cmd-F collision avoidance) | P1 |
| Text file preview | HIGH | MEDIUM | P1 |
| Markdown rendered preview | HIGH | MEDIUM | P1 |
| Sort name/size/mtime | MEDIUM | LOW | P1 |
| Image preview | MEDIUM | MEDIUM | P1 |
| Binary / too-large refusal + download | HIGH | LOW | P1 |
| Empty / error / loading states | HIGH | LOW | P1 |
| Broken symlink display | MEDIUM | LOW | P1 |
| Remote session indicator | MEDIUM | LOW | P1 |
| Capability gate for web-share | HIGH | LOW | P1 |
| TUI Files view (custom list+viewport) | HIGH (parity contract) | MEDIUM | P1 |
| ARIA accessibility | HIGH (release contract) | MEDIUM | P1 |
| Lazy-load / download progress for remote | LOW-MEDIUM | MEDIUM | P2 |
| Context menu (right-click) | LOW | MEDIUM | P3 |
| Absolute path paste input | LOW | MEDIUM | P3 (v3.5) |
| Recursive filter | MEDIUM | HIGH | P3 (v3.5) |
| Syntax highlighting | MEDIUM | HIGH | P3 (v3.5, editor library) |
| Write operations | HIGH | HIGH | OUT (v3.5) |
| Directories-first toggle | LOW | LOW | P3 (v3.5) |
| TUI image preview (sixel) | LOW | HIGH | OUT (v3.4); revisit v3.5 |

---

## Competitor Feature Analysis

| Feature | VS Code Explorer | GitHub web viewer | Ranger (TUI) | Cyberduck/FileZilla | AgentHub v3.4 (proposed) |
|---------|-----------------|-------------------|--------------|---------------------|--------------------------|
| Layout | Tree pane in sidebar + list (but sidebar IS the tree pane) | Single-pane list with inline preview | Three-column miller (parent / current / preview) | Split: local + remote panes | Single-pane list + slide-in/side-by-side preview within tab |
| Keyboard nav | Full arrow + Enter; j/k with Vim mode extension | Arrow keys partial; primarily mouse | vi-style hjkl; / for search | Tab between panes, arrow keys | Up/Down/Enter/Backspace/Home/End/PgUp/PgDn + / filter |
| Filter | Explorer filter (Ctrl-P for file, Ctrl-E for within Explorer) | Search in repo (separate feature) | / in ranger | Name filter in toolbar | / key, current dir scope |
| Sort | Name/size/mtime with dirs-first option | Name/type; dirs first by default | By name, type, time | Name/size/date | Name/size/mtime with dirs-first |
| Preview | Click file → opens in editor | Click file → inline preview with syntax highlight (for text); image inline | Right panel shows preview (text/image via w3m/ueberzug) | Preview via macOS Quick Look | Side-by-side pane: text/markdown/image; binary refusal + download |
| Binary handling | "Binary files cannot be displayed" | "Sorry, we can't display non-text files inline. You can view the raw file or download it." | Shows hex dump option | Download only | "This file cannot be previewed (binary content). [Download]" |
| Too-large handling | Warns + offers Raw view | "The file is too large to display." + Raw/Download links | Shows size, offers download via opener | Streams large files | "File too large to preview (N MB, limit 5 MB). [Download]" |
| Markdown | Side-by-side preview (Cmd-K V) | Rendered inline | Not rendered (plain text) | Not rendered | Rendered in preview pane |
| Image preview | Preview in editor pane | Inline in viewer | Via w3m/ueberzug (external) | Via Quick Look (macOS) | Inline in preview pane, aspect-ratio preserved |
| Remote indicator | No (always local) | No (always web) | No | Strong: local vs remote explicit split panes | `(remote: hostname)` badge in path bar |
| Accessibility | Good; full keyboard, ARIA tree | Moderate; some ARIA landmark issues | TUI-only; inherently keyboard-first | Moderate | ARIA listbox + region + breadcrumb nav; aria-live preview |
| TUI parity | N/A | N/A | IS the TUI | N/A | Custom list.Model + viewport.Model, TokyoNight palette |

---

## Sources

**ARIA / Accessibility (HIGH confidence — official spec):**
- [ARIA: tree role — MDN Web Docs](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/tree_role)
- [Tree View Pattern — W3C WAI-ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/patterns/treeview/)
- [ARIA: treeitem role — MDN](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Roles/treeitem_role)

**Charmbracelet / Bubble Tea (HIGH confidence — official docs):**
- [bubbles list.Model — pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles/list)
- [bubbles list README — GitHub](https://github.com/charmbracelet/bubbles/blob/master/list/README.md)
- [bubbles/v2 viewport — pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles/v2/viewport)
- [bubbles filepicker — pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles/filepicker)
- [charmbracelet/bubbles — GitHub](https://github.com/charmbracelet/bubbles)
- [charmbracelet/bubbletea — GitHub](https://github.com/charmbracelet/bubbletea)

**TUI file manager patterns (MEDIUM confidence — docs + community):**
- [ranger wiki — Keybindings](https://github.com/ranger/ranger/wiki/Keybindings)
- [ranger — Arch Wiki](https://wiki.archlinux.org/title/Ranger)
- [Terminal file managers — DEV Community](https://dev.to/ccoveille/terminal-file-managers-1b5l)
- [tere — terminal file explorer](https://github.com/mgunyho/tere)

**VS Code Explorer (MEDIUM confidence — public docs):**
- [VS Code User Interface](https://code.visualstudio.com/docs/getstarted/userinterface)
- [VS Code Custom Layout](https://code.visualstudio.com/docs/configure/custom-layout)

**GitHub / GitLab file viewer limits (MEDIUM confidence — official docs + community):**
- [About large files on GitHub — GitHub Docs](https://docs.github.com/en/repositories/working-with-files/managing-large-files/about-large-files-on-github)
- [GitHub community: too big / binary files not displayed](https://github.com/orgs/community/discussions/46179)
- [GitLab forum: PDF too large to display](https://forum.gitlab.com/t/this-pdf-is-too-large-to-display-please-download-to-view/105829)

**Cyberduck (MEDIUM confidence — official docs):**
- [Cyberduck Browser documentation](https://docs.cyberduck.io/cyberduck/browser/)

**Miller columns (MEDIUM confidence — Wikipedia + UX articles):**
- [Miller columns — Wikipedia](https://en.wikipedia.org/wiki/Miller_columns)
- [Interaction Design for Trees — Medium](https://medium.com/@hagan.rivers/interaction-design-for-trees-5e915b408ed2)

---
*Feature research for: AgentHub v3.4 read-only file browser (GUI + web + TUI parity)*
*Researched: 2026-05-20*
