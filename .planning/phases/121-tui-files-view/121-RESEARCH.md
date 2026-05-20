# Phase 121: TUI Files View - Research

**Researched:** 2026-05-20
**Domain:** Bubble Tea v2 read-only file browser sub-model wired into existing AgentHub TUI; consumes Phase 118 `DaemonClient.{ListFiles,StatFile,ReadFile,HeadFile}` over the trusted local socket
**Confidence:** HIGH

## Summary

Phase 121 adds a per-session Files view to the AgentHub TUI as a new `tabID` value (`tabFiles`) alongside the existing `tabHome / tabSessions / tabRemote / tabSettings`. The view is opened from the Sessions list by pressing `f` on a selected local session and displays a lipgloss-bordered two-pane layout: a custom file list (left) joined via `lipgloss.JoinHorizontal` with a `bubbles/v2/viewport` preview pane (right), all rendered through the existing TokyoNight `Styles` token set.

All filesystem I/O is dispatched through `tea.Cmd` closures that call `m.client.ListFiles/StatFile/ReadFile/HeadFile` and return typed `tea.Msg` values — never `os.ReadDir` / `os.Open` inside `Update`. The daemon socket (Phase 118) is the trust boundary: the TUI passes only `(sessionID, relPath)` and the sandbox + 5 MiB cap + path validation enforcement happens server-side. Markdown previews use `charmbracelet/glamour@v0.8.0` (already in `go.sum` as transitive — promote to direct dep). Plain text reuses an existing `bubbles/v2/viewport`.

Key-dispatch priority slots **above main view, below kill-confirm/new-session/QR/help** — exactly between Priority 5 (help overlay) and Priority 6 (tab cycling) in `handleKey`. The Files view introduces its own internal sub-states (filter active / preview pane focused / list focused), but those are handled inside `handleFilesKey` rather than at the top-level priority stack.

**Primary recommendation:** Create `internal/tui/files.go` containing a `filesModel` struct embedded inside the main `Model`, with `filesUpdate(msg tea.Msg) tea.Cmd` and `renderFilesTab(cw, ch int) string` entrypoints called from the existing `Update` / `renderContentPane` dispatch tables. Reuse the existing `Styles`, `wrapInFrame`, `truncate`, and tea.Cmd patterns; do NOT use `bubbles/v2/filepicker` (it's a selection-dialog primitive and cannot enforce the "never above cwd" constraint cleanly).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Open files view: press `f` on selected session in Sessions list
- Layout: lipgloss-bordered Files view, list on left + preview on right, TokyoNight palette
- Keymap: Up/Down/PgUp/PgDn (cursor), Enter (enter dir), Backspace/Left (up), Backspace-at-root = no-op
- Preview: text (≤5MB) → raw; .md → ANSI markdown via `charmbracelet/glamour`
- Refusals: binary → "Use desktop or web to preview"; over-5MB → "Too large to preview, use desktop or web to download"
- `/` activates type-ahead filter (current directory only); Escape clears + dismisses
- Status line: session-cwd-relative path (left-truncated `…/utils/helper.ts`), file count, selection position
- All FS I/O via `tea.Cmd` returning `tea.Msg` — NO synchronous `os.ReadDir`/`os.Open` inside `Update`
- `?` help overlay in tabFiles mode shows Files keybindings
- Key-dispatch priority: above main view, below kill-confirm/new-session/QR/help overlays
- Uses Phase 118 `DaemonClient.{ListFiles, StatFile, ReadFile}` (NOT webserver routes)

### Claude's Discretion
- Implementation file location (likely `internal/tui/files.go` or sub-package)
- Tab/view state machine integration with existing Bubble Tea model
- Filter input box style (lipgloss border or inline)

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TUI-01 | `internal/tui/files.go` sub-model with custom list + `viewport.Model` joined via `lipgloss.JoinHorizontal`, bordered with TokyoNight palette | Standard Stack §Bubbles components; Architecture §System Architecture Diagram; existing `wrapInFrame` helper in `internal/tui/view.go:246` |
| TUI-02 | "Files" entry reachable per-session via Sessions list (press `f`); closing returns to prior tab | Decisions §Tab integration; existing `openTab`/`cycleTab` in `internal/tui/model.go:188-205`. Press `f` opens `tabFiles` with selected session ID bound. |
| TUI-03 | Up/Down/PgUp/PgDn/Enter/Backspace/Left navigation; Backspace-at-root no-op | Architecture §Key Dispatch; Pitfall §TUI-PITFALL-2 (Backspace ambiguity). Server-side `os.Root` enforces sandbox if anything slips through. |
| TUI-04 | Text ≤5MB → raw; .md → glamour ANSI; binary → refusal; over-cap → refusal | Standard Stack §`charmbracelet/glamour`; daemon already returns 413 on >5MiB and sets MIME via `FileEntry.IsBinary` field |
| TUI-05 | `/` activates type-ahead filter (current dir only); Escape clears + dismisses | Standard Stack §Filter input; Pitfall §TUI-PITFALL-2; mirrors GUI behavior (UI-04, Phase 120) |
| TUI-06 | Status line: left-truncated relative path, file count, selection position | Code Examples §pathTruncateLeft; existing `truncate()` in view.go:754 is right-truncate — new helper required |
| TUI-07 | ALL FS I/O via `tea.Cmd` returning `tea.Msg` | Pitfall §TUI-PITFALL-1 (synchronous freeze); existing pattern in `internal/tui/cmds.go:13-72` (fetchSessions, etc.) |
| TUI-08 | Works against local AND remote (tailnet) sessions | Open Question §OQ-1: remote sessions not in v3.4 TUI scope per PITFALLS.md Pitfall 13. Confirm with planner: local-only or tailnet routing. |
| TUI-09 | `?` help overlay updated with file-browser keybindings | Architecture §Help integration; existing `help.go:buildHelpContent` needs view-aware section emission |
| TUI-10 | Key-dispatch priority above main view but below kill-confirm/new-session/QR overlay/help | Architecture §Key Dispatch; existing 6-level priority in `update.go:114-153` — insert Files between Priority 5 (help) and Priority 6 (tab cycling) |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Path sandbox + 5 MiB cap + traversal rejection | Daemon (Phase 118) | — | `os.Root` lives in `internal/files/`; TUI must NOT re-implement validation |
| HTTP/Unix-socket transport | `DaemonClient` (Phase 118) | — | TUI just calls typed Go methods; the socket is the trust boundary |
| Async dispatch (tea.Cmd) | TUI (`internal/tui/files.go`) | — | Bubble Tea owns I/O scheduling; no project pattern bypasses it |
| List/preview rendering | TUI lipgloss + bubbles/v2/viewport | — | TUI owns terminal layout; reuse existing `Styles` tokens |
| Markdown ANSI rendering | `charmbracelet/glamour` (TUI side) | — | Glamour produces ANSI; viewport scrolls it |
| Filter state | TUI (`filesModel`) | — | Pure UI state; no daemon involvement |
| Session cwd resolution | Daemon (`engine.GetSessionWorkDir`) | — | TUI never sees absolute paths; sends `path="."` to scope to session cwd |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.0.6 | Async I/O via `tea.Cmd` returning `tea.Msg` | Already in use across the TUI (Phase 86 tabs, Phase 77 modals) |
| `charm.land/bubbles/v2` | v2.1.0 | `viewport.Model` for preview pane scroll; `textinput.Model` for filter input | Already direct dep; `textinput` already used for new-session modal + inline rename (`model.go:127`) |
| `charm.land/lipgloss/v2` | v2.0.3 | Border, palette, `JoinHorizontal` for two-pane layout | Already in use throughout `view.go` |
| `github.com/charmbracelet/glamour` | v0.8.0 | ANSI markdown rendering for `.md` previews | Already a transitive dep — `go list -m github.com/charmbracelet/glamour` confirms v0.8.0 [VERIFIED: `go list -m`] |
| `github.com/scottkw/agenthub/internal/daemon` | n/a (internal) | `DaemonClient.{ListFiles,StatFile,ReadFile,HeadFile}` typed methods | Phase 118 prerequisite, frozen API |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/scottkw/agenthub/internal/files` | n/a (internal) | `FileEntry` and `FileListResponse` re-exported via `daemon.FileEntry` alias | Decoding `ListFiles` results — already aliased per Phase 118 Plan 05 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom list rendering | `bubbles/v2/filepicker` | **Rejected.** Filepicker is a selection-dialog primitive — it walks the host filesystem directly (cannot route through `DaemonClient`), and "never above cwd" requires intercepting its internal `Back` key, which fights the abstraction. Use a hand-rolled list backed by `[]files.FileEntry` slice (parallel to how the existing Sessions tab renders `[]daemon.SessionInfo`). |
| Custom list rendering | `bubbles/v2/list` | **Rejected.** `list.Model` has built-in `/` filter but its item model wants `FilterValue() string` on each item, plus it pulls in pagination/help overlays that conflict with the existing footer. The Sessions tab does NOT use `list.Model` — it renders rows manually via `renderSessionRow`. Stay consistent. |
| `glamour` for preview | Hand-rolled ANSI conversion | **Rejected.** Glamour is already in `go.sum`, MIT, supports `WithAutoStyle()` for light/dark adaptation, and is a one-line API. No reason to reinvent. |
| `viewport.Model` for preview | Plain `strings.Split` + window slice | **Rejected.** `viewport` handles scroll position, PgUp/PgDn, key bindings consistently. Promoting it to the project costs nothing — it's already in `bubbles/v2`. |

**Installation:**
```bash
# Glamour is already in go.sum as a transitive dep; promote to direct dep:
go get github.com/charmbracelet/glamour@v0.8.0
go mod tidy
```

**Version verification:**
- `charm.land/bubbletea/v2` v2.0.6 — [VERIFIED: `go list -m`]
- `charm.land/bubbles/v2` v2.1.0 — [VERIFIED: `go list -m`]
- `charm.land/lipgloss/v2` v2.0.3 — [VERIFIED: `go list -m`]
- `github.com/charmbracelet/glamour` v0.8.0 — [VERIFIED: `go list -m`]

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| `github.com/charmbracelet/glamour` | Go modules | ~5 yrs | n/a (Go modules) | github.com/charmbracelet/glamour | n/a (Go) | Approved — promoted from transitive |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*Glamour is already pinned at v0.8.0 in `go.sum` as a transitive dep — `go get @v0.8.0` re-records it as direct without any network fetch. No new third-party code enters the supply chain.*

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  TUI (Bubble Tea event loop)                                    │
│                                                                  │
│  KeyPressMsg ──► handleKey (update.go:114) ──► priority cascade │
│                                                                  │
│  Priority 1: editing (rename)                                    │
│  Priority 2: kill confirm                                        │
│  Priority 3: new session modal                                   │
│  Priority 4: QR overlay                                          │
│  Priority 5: help overlay                                        │
│  *** Priority 5.5: tabFiles active ──► handleFilesKey ***       │
│  Priority 6: tab cycling ([ ])                                  │
│  Priority 7: pane-focus dispatch (sidebar vs content)            │
│                                                                  │
│  handleFilesKey ──► sub-priority cascade:                        │
│    1. filter active (textinput consumes printable keys)          │
│    2. preview-focused (PgUp/PgDn scroll viewport)                │
│    3. list-focused (Up/Down/Enter/Backspace navigate)            │
│                                                                  │
│  All dir/file ops emit tea.Cmd ──► goroutine ──► tea.Msg:        │
│                                                                  │
│  ┌─ loadDirCmd(client, sid, relPath) ─────────────────┐         │
│  │   client.ListFiles(ctx, sid, relPath)              │         │
│  │   returns: filesListMsg{entries, truncated, err}   │         │
│  └─────────────────────────────────────────────────────┘         │
│                                                                  │
│  ┌─ statFileCmd(client, sid, relPath) ────────────────┐         │
│  │   client.StatFile(ctx, sid, relPath)               │         │
│  │   returns: filesStatMsg{entry, err}                │         │
│  └─────────────────────────────────────────────────────┘         │
│                                                                  │
│  ┌─ readFileCmd(client, sid, relPath) ────────────────┐         │
│  │   client.ReadFile(ctx, sid, relPath)               │         │
│  │   returns: filesReadMsg{data, mime, err}           │         │
│  │   (daemon enforces 5 MiB cap → 413 → typed error)  │         │
│  └─────────────────────────────────────────────────────┘         │
│                                                                  │
│  Update merges msg into filesModel state ──► re-render           │
└─────────────────────────────────────────────────────────────────┘
                          │
                          ▼ Unix socket / named pipe (trusted)
┌─────────────────────────────────────────────────────────────────┐
│  Daemon (Phase 118)                                              │
│                                                                  │
│  GET /api/files/list?session=<id>&path=<rel>                     │
│  GET /api/files/stat?session=<id>&path=<rel>                     │
│  GET /api/files/read?session=<id>&path=<rel>                     │
│  HEAD /api/files/read?session=<id>&path=<rel>                    │
│                                                                  │
│  engine.GetSessionWorkDir(sid) ──► resolved cwd                  │
│  files.NewSandbox(cwd) ──► *os.Root                              │
│  Sandbox.{List,Stat,Read} ──► kernel-atomic openat2              │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/tui/
├── files.go              # NEW — filesModel, handleFilesKey, renderFilesTab
├── files_cmds.go         # NEW — loadDirCmd / statFileCmd / readFileCmd / filesListMsg etc.
├── files_test.go         # NEW — table-driven tests using testModel() pattern
├── model.go              # MODIFIED — add tabFiles iota, filesModel field
├── update.go             # MODIFIED — insert filesKey priority slot; handle filesListMsg/filesStatMsg/filesReadMsg
├── view.go               # MODIFIED — add tabFiles case in renderContentPane dispatch
├── keys.go               # MODIFIED — extend KeyMap with FilesOpen ('f'), FilterStart ('/'), FilterEsc, etc.
├── help.go               # MODIFIED — view-aware sections; show Files bindings when activeTabID() == tabFiles
└── tui.go                # MODIFIED — initialize filesModel zero-value (lazy)
```

### Pattern 1: Sub-Model Embedded in Main Model

**What:** Phase 121 introduces enough new state (cwd-relative path, entries slice, viewport, filter input, preview content, selected index, scroll position) that a flat `Model` becomes noisy. Wrap it in a `filesModel` struct embedded as `m.files filesModel`.

**When to use:** Whenever a Bubble Tea tab/view has >5 fields of its own state, the existing project convention (per Phase 77 modal grouping) is sub-grouping with `// File browser state (Phase 121)` comment markers — but for state coherence, prefer a separate struct.

**Example:**

```go
// internal/tui/files.go

package tui

import (
    "charm.land/bubbles/v2/textinput"
    "charm.land/bubbles/v2/viewport"
    "github.com/scottkw/agenthub/internal/daemon"
)

// filesModel holds all state for the Files view.
type filesModel struct {
    sessionID   string                  // session whose cwd we're scoped to
    cwd         string                  // relative path within sandbox; "" = cwd root
    entries     []daemon.FileEntry      // current directory listing
    truncated   bool                    // ListFiles set X-Directory-Truncated
    selected    int                     // cursor position in entries (after filtering)
    loading     bool                    // listing in flight
    err         error                   // last error (sticky until next nav)

    // Filter
    filterActive bool
    filterInput  textinput.Model

    // Preview pane
    preview       viewport.Model
    previewKind   previewKind            // text / markdown / binary / overcap / empty / err
    previewMime   string
    previewLoading bool
    previewErr     error
    previewFocused bool                   // PgUp/PgDn route to viewport vs list

    // Status line cache
    statusPath  string                  // last-rendered left-truncated path
}

type previewKind int

const (
    previewEmpty previewKind = iota
    previewText
    previewMarkdown
    previewBinary
    previewOverCap
    previewErr
)
```

### Pattern 2: tea.Cmd Constructor With Context Timeout

**What:** Every filesystem op gets its own `tea.Cmd` factory that closes over the client + session ID + relative path. Mirrors the existing `fetchSessions` pattern in `cmds.go:13`.

**Example:**

```go
// internal/tui/files_cmds.go

package tui

import (
    "context"
    "time"

    tea "charm.land/bubbletea/v2"
    "github.com/scottkw/agenthub/internal/daemon"
)

type filesListMsg struct {
    sessionID string
    relPath   string
    entries   []daemon.FileEntry
    truncated bool
    err       error
}

type filesReadMsg struct {
    sessionID string
    relPath   string
    data      []byte
    mime      string
    err       error
}

type filesHeadMsg struct {
    sessionID string
    relPath   string
    size      int64
    mime      string
    err       error
}

// loadDirCmd dispatches a ListFiles request for the given session/path.
// 5-second timeout protects against slow daemon (Tailscale-paused, OOM, etc.).
func loadDirCmd(client *daemon.DaemonClient, sid, relPath string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        entries, truncated, err := client.ListFiles(ctx, sid, relPath)
        return filesListMsg{
            sessionID: sid, relPath: relPath,
            entries: entries, truncated: truncated, err: err,
        }
    }
}

// readFileCmd dispatches a ReadFile request. Daemon enforces 5 MiB cap → 413 → typed error.
func readFileCmd(client *daemon.DaemonClient, sid, relPath string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        data, mime, err := client.ReadFile(ctx, sid, relPath)
        return filesReadMsg{
            sessionID: sid, relPath: relPath,
            data: data, mime: mime, err: err,
        }
    }
}

// headFileCmd preflights for binary / over-cap classification BEFORE attempting Read.
// Use this for the click-to-preview flow so we never download a 50 MiB file then
// discard it; HEAD returns Content-Length + Content-Type instantly.
func headFileCmd(client *daemon.DaemonClient, sid, relPath string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        size, mime, _, err := client.HeadFile(ctx, sid, relPath)
        return filesHeadMsg{
            sessionID: sid, relPath: relPath,
            size: size, mime: mime, err: err,
        }
    }
}
```

### Pattern 3: Key Dispatch Insertion

**What:** Insert one new priority slot in `handleKey` between Priority 5 (help overlay) and Priority 6 (tab cycling). When `activeTabID() == tabFiles` and no higher-priority modal is active, route to `handleFilesKey`.

**Example:**

```go
// internal/tui/update.go (modified)

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    // Priority 1: Inline rename captures all keys
    if m.editing { return m.handleRenameKey(msg) }
    // Priority 2: Kill confirmation dialog
    if m.modal == modalKillConfirm { return m.handleKillConfirmKey(msg) }
    // Priority 3: New session modal
    if m.modal == modalNewSession { return m.handleNewSessionKey(msg) }
    // Priority 4: QR overlay (Phase 78)
    if m.qrSession != nil { return m.handleQRKey(msg) }
    // Priority 5: Help overlay
    if m.showHelp {
        if key.Matches(msg, m.keys.Help) || msg.String() == "esc" {
            m.showHelp = false
            return m, nil
        }
        return m, nil
    }
    // *** Priority 5.5: Files tab active (Phase 121) ***
    if m.activeTabID() == tabFiles {
        return m.handleFilesKey(msg)
    }
    // Priority 6: Tab cycling
    if key.Matches(msg, m.keys.PrevTab) { m.cycleTab(-1); return m, nil }
    if key.Matches(msg, m.keys.NextTab) { m.cycleTab(+1); return m, nil }
    // Priority 7: Pane-focus dispatch
    if m.panesFocus == focusSidebar { return m.handleSidebarKey(msg) }
    return m.handleContentKey(msg)
}
```

**Why this position:** The tab cycling keys (`[`, `]`) must STILL work while the Files view is up — users need to switch back to Sessions. So Priority 5.5 dispatches to `handleFilesKey` which itself MUST forward `[` and `]` back to the parent (or call `m.cycleTab` directly). Alternatively: invert — only intercept Files-specific keys, fall through everything else. **Recommend: Files-handler intercepts only its own keys, lets unknown keys fall through to a `return m, nil` (swallow) or explicitly delegates `[ ] q Q ?` back. See `handleQRKey` (`update.go:157`) for the swallowing pattern.**

### Pattern 4: Help Overlay View-Awareness

**What:** `buildHelpContent` in `help.go:37` currently emits a single static set of sections. Phase 121 needs it to emit a different "Sessions" or "Files" section based on `m.activeTabID()`.

**Example:**

```go
// internal/tui/help.go (modified)

func (m Model) buildHelpContent() string {
    // ...keyStyle, descStyle, groupStyle as before...
    var sections []string

    // Group 1: Navigation (universal)
    sections = append(sections, groupStyle.Render("Navigation"))
    sections = append(sections,
        formatBinding("j/k, Up/Down", "Move up/down"),
        formatBinding("[/]", "Cycle tabs"),
        formatBinding("Tab", "Toggle sidebar/content"),
    )

    // Group 2: per-view section
    sections = append(sections, "")
    switch m.activeTabID() {
    case tabFiles:
        sections = append(sections, groupStyle.Render("Files"))
        sections = append(sections,
            formatBinding("Enter", "Enter directory / preview file"),
            formatBinding("Backspace, Left", "Up one directory"),
            formatBinding("PgUp/PgDn", "Page list / scroll preview"),
            formatBinding("/", "Filter (current dir)"),
            formatBinding("Esc", "Clear filter / back to Sessions"),
        )
    default:
        sections = append(sections, groupStyle.Render("Sessions"))
        sections = append(sections,
            formatBinding("Enter", "Attach to session"),
            formatBinding("f", "Open files view"),     // NEW
            formatBinding("q", "QR code / URL"),
            formatBinding("n", "New session"),
            formatBinding("d", "Kill session"),
            formatBinding("r", "Rename session"),
        )
    }

    // Group 3: General
    sections = append(sections, "")
    sections = append(sections, groupStyle.Render("General"))
    sections = append(sections,
        formatBinding("?", "Toggle help"),
        formatBinding("Q, Ctrl+C", "Quit"),
    )
    // ...close hint as before...
}
```

### Pattern 5: Tab Opening From Sessions List

**What:** Press `f` on a selected local session in `handleContentKey` → resolve session ID → set `m.files.sessionID` → `m.openTab(tabFiles)` → emit `loadDirCmd(client, sid, ".")` to populate.

**Example:**

```go
// internal/tui/update.go handleContentKey — add new case:

case key.Matches(msg, m.keys.FilesOpen):
    if len(m.unifiedList) == 0 {
        return m, nil
    }
    entry := m.unifiedList[m.selected]
    if entry.kind == entryRemote {
        m.toast = "File browser not available for remote sessions in v3.4"
        m.toastKind = toastInfo
        m.toastExp = time.Now().Add(2 * time.Second)
        return m, nil
    }
    if entry.kind != entryLocal || entry.session == nil {
        return m, nil
    }
    sid := entry.session.ID
    // Reset filesModel for new session
    m.files = newFilesModel(sid, m.contentWidth(), m.height-3)
    m.openTab(tabFiles)
    return m, loadDirCmd(m.client, sid, ".")
```

### Anti-Patterns to Avoid

- **Don't use `bubbles/v2/filepicker`:** It walks the host filesystem directly. Cannot route through DaemonClient. Cannot enforce "never above cwd" without fighting the abstraction. STACK.md §TUI Files View flagged this as a workaround; the cleaner answer is don't use it at all.
- **Don't use `bubbles/v2/list` with `FilterValue` items:** It conflicts with the existing footer hint bar and pulls in its own help overlay. Sessions tab renders rows manually — Files tab follows the same pattern.
- **Don't call `os.ReadDir` / `os.Open` in `Update`:** Pitfall §TUI-PITFALL-1. Even on local sessions, the I/O MUST go through the daemon socket so the sandbox check runs.
- **Don't store `data []byte` of preview content longer than needed:** A 5 MiB string in `filesModel.preview` lives for the whole TUI session if the user never navigates away. When loading a new preview, replace; when leaving `tabFiles`, clear: `m.files.preview = viewport.Model{}`.
- **Don't fetch full file body before classifying:** Use `HeadFile` first to learn size + MIME. Only if size ≤ 5 MiB AND text-class do we call `ReadFile`. This avoids transferring 4.9 MiB of binary garbage just to discard it.
- **Don't truncate the status path with the existing `truncate()` helper:** That helper is right-truncate (`"long_filenam..."`). Status line needs left-truncate (`"…/utils/helper.ts"`). Add `truncateLeft` helper.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Path sandboxing in TUI | Re-validate paths client-side before sending | Trust the daemon's `os.Root` enforcement | Phase 118 `internal/files` is fuzz-proven (FuzzSandboxPath, 0 crashes / 60s). Re-validating client-side adds attack surface (e.g., bypass via case-folding on macOS) without adding defense. |
| Markdown → ANSI conversion | Hand-rolled regex / `tcell` styling | `glamour.Render(md, "dark")` | Glamour supports `WithAutoStyle()`, GFM, code-block syntax fragments, lists/tables. Already in go.sum. |
| Preview pane scroll | Manual `strings.Split` + window slice | `bubbles/v2/viewport` | Viewport handles scroll position, half-page jumps, mouse (off in AgentHub but viewport ignores cleanly), key bindings, wrap. |
| MIME detection for refusal logic | Sniff first 512 bytes client-side | `daemon.FileEntry.IsBinary` + `daemon.FileEntry.MIME` | Daemon already detects via `wailsapp/mimetype` (200+ types) — see Phase 118 Plan 02 §MIME cascade. TUI just reads the field. |
| Filter substring match | Custom Trie / B-tree | `strings.Contains(strings.ToLower(name), q)` in a loop | Directory listings are bounded (10K max — daemon truncates). Linear scan is imperceptibly fast and matches STACK.md guidance for the React side. |
| Async dispatch infrastructure | goroutines + channels written by hand | `tea.Cmd` returning `tea.Msg` | This is the Bubble Tea project pattern — see `cmds.go:13-72`. Channels would break the Update loop's single-threading invariant. |
| Help overlay | Re-implement from scratch for Files view | Modify existing `buildHelpContent` to switch on `m.activeTabID()` | Existing overlay does centering, border, title injection. Just feed it different sections per view. |
| Path truncation (left-side) | Inline `[]rune` slicing in every render | Add `truncateLeft(s string, max int) string` helper next to existing `truncate()` | One helper, one test, used in status line render only. |

**Key insight:** Phase 118 deliberately produced typed Go methods on `DaemonClient` so downstream phases never touch the filesystem directly. Phase 121's job is **TUI rendering and event dispatch only** — every "filesystem question" gets answered by a `tea.Cmd` that calls `client.ListFiles / ReadFile / HeadFile`. Re-implementing any sandbox / MIME / cap logic in the TUI is an anti-pattern that defeats the design.

## Common Pitfalls

### Pitfall TUI-PITFALL-1: Synchronous I/O Freezes the Render Loop

**What goes wrong:** Calling `os.ReadDir`, `os.Open`, or even `client.ListFiles(ctx, ...)` directly inside `Update` blocks the entire Bubble Tea event loop until I/O completes. On slow disks, NFS, or when the daemon is responding slowly, this freezes the terminal for seconds. Keystrokes are queued or lost.

**Why it happens:** Bubble Tea v2's `Update` contract is "non-blocking; return a `tea.Cmd` for any I/O." It's tempting to write `entries, _ := client.ListFiles(...)` inline because the project's Sessions tab does NOT use `client.ListSessions` synchronously — it goes through `fetchSessions` (`cmds.go:13`) which wraps it in `func() tea.Msg { ... }`.

**How to avoid:** ALL FS operations MUST be wrapped in `tea.Cmd`. Use the `loadDirCmd / readFileCmd / headFileCmd` factories shown in Code Examples. Include a 5-second timeout (`context.WithTimeout`) so a wedged daemon doesn't leave the TUI in permanent "loading" state.

**Warning signs:** TUI freezes when entering a directory. Type-ahead keystrokes appear all at once after directory load. Filter input lags visibly behind keypresses.

**Phase to address:** This phase. PITFALLS.md §Pitfall 7 calls this out specifically as a v3.4 TUI risk and lists it as "MANDATORY" on the merge checklist.

### Pitfall TUI-PITFALL-2: Backspace Ambiguity (Filter vs Navigate-Up)

**What goes wrong:** When the user is typing in the filter, Backspace should delete a character. When the filter is empty, Backspace should navigate up one directory. Without explicit dispatch ordering, the same keypress can do both (delete the last filter char AND navigate up in the same Update).

**Why it happens:** Bubble Tea processes one message per `Update` call, but if the filter input's `Update` consumes Backspace and ALSO `filesModel.Update` runs the navigate-up branch on the same `tea.KeyPressMsg`, both fire.

**How to avoid:** Strict cascade inside `handleFilesKey`:
```go
if m.files.filterActive {
    if msg.String() == "esc" {
        m.files.filterActive = false
        m.files.filterInput.SetValue("")
        return m, nil
    }
    // Backspace, printable chars, Left/Right cursor — ALL go to textinput
    var cmd tea.Cmd
    m.files.filterInput, cmd = m.files.filterInput.Update(msg)
    return m, cmd
}
// Filter NOT active — Backspace = navigate up
if msg.String() == "backspace" || msg.String() == "left" {
    if m.files.cwd == "" || m.files.cwd == "." {
        return m, nil    // no-op at root (TUI-03 locked decision)
    }
    parent := parentDir(m.files.cwd)
    return m, loadDirCmd(m.client, m.files.sessionID, parent)
}
```

**Warning signs:** Pressing Backspace in the filter both deletes a character AND navigates up. Filter loses its value when navigating.

**Phase to address:** This phase. PITFALLS.md §Pitfall 12 cited this as a class of bug previously caused TUI regression in v3.1 (kill-confirm modal Backspace conflict).

### Pitfall TUI-PITFALL-3: Preview Memory Leak on Long Sessions

**What goes wrong:** Every file preview stores the file body in `filesModel.preview` (a `viewport.Model` which holds the rendered string). If the user navigates through hundreds of files, the GC sees old previews kept alive only because Go's escape analysis decided to heap-allocate. Long TUI sessions creep up to 100 MiB+ resident.

**Why it happens:** `viewport.Model` retains its content until `.SetContent` is called with a replacement (or the viewport is reassigned). The previous content is dropped, but a 5 MiB string that escaped to the heap can sit in a generation pool.

**How to avoid:**
1. ALWAYS call `m.files.preview.SetContent("")` before navigating away from `tabFiles`.
2. On `tabFiles` close (e.g., user presses `Esc` at the root with empty filter), set `m.files.preview = viewport.Model{}` to drop the entire allocation.
3. Reuse the existing `viewport.Model` across previews — don't create a new one per file. The `SetContent` API replaces in-place.

**Warning signs:** `ps`/`htop` shows TUI process RSS growing to >50 MiB after browsing 100+ files. Memory profiles show retained `[]byte` arrays.

**Phase to address:** This phase. Plan should include a `closeFilesTab()` helper that does the cleanup.

### Pitfall TUI-PITFALL-4: glamour Color Profile Mismatch With TokyoNight

**What goes wrong:** `glamour.Render(md, "dark")` uses glamour's built-in "dark" style, which renders headings in a default purple that clashes with AgentHub's TokyoNight palette. Worse, `glamour.WithAutoStyle()` detects the terminal background and may render "light" on a dark terminal that happens to use a light-themed scheme.

**Why it happens:** Glamour has its own theme system — it's NOT aware of lipgloss's `Styles` tokens. The `m.hasDark` field in the main `Model` is set from `tea.BackgroundColorMsg`, but glamour doesn't consume that.

**How to avoid:** Explicitly pass the style based on `m.hasDark`:
```go
import "github.com/charmbracelet/glamour"

style := "dark"
if !m.hasDark { style = "light" }
out, err := glamour.Render(string(data), style)
```

For better palette consistency, glamour also supports a custom `ansi.StyleConfig` via `glamour.WithStyles(...)` — but for v3.4, the built-in `dark`/`light` is acceptable and matches what STACK.md §TUI Markdown Preview recommended.

**Warning signs:** Markdown previews look visually inconsistent with the rest of the TUI. Headings appear in colors that don't match the TokyoNight palette.

**Phase to address:** This phase, optionally. The locked decision says "ANSI markdown via charmbracelet/glamour" — the planner has discretion on whether to ship with built-in `dark`/`light` styling or a custom config. Recommend: ship with built-in, defer custom palette to v3.5.

### Pitfall TUI-PITFALL-5: Daemon Truncation Signal Ignored

**What goes wrong:** `DaemonClient.ListFiles` returns `(entries, truncated bool, err)`. For directories with >10,000 entries, the daemon caps at 10K and sets `truncated = true`. If the TUI ignores the bool, users see no indication that more entries exist — they think the directory is exactly 10K large.

**How to avoid:** Surface truncation in the status line: `"foo/  •  10000 entries (truncated)"`. Add a one-line note inside the file list pane footer when `m.files.truncated` is true.

**Phase to address:** This phase. The Status Line spec (TUI-06) already mentions file count — when truncated, show "10000+ entries" or "(truncated)".

### Pitfall TUI-PITFALL-6: Forgetting to Reload on Refresh / Reopen

**What goes wrong:** When the user opens `tabFiles` for session A, navigates to `subdir/`, then switches back to Sessions, selects session B, presses `f` — the Files view shows session A's last `subdir/` listing because the model was reused.

**How to avoid:** On every `f` keypress in Sessions list, RESET `filesModel` to a zero value with the new session ID and dispatch `loadDirCmd(client, sid, ".")`. Don't rely on "is this the same session?" comparison — always reset to give a clean per-session state.

**Warning signs:** Files tab shows stale paths after switching between sessions. Filter state persists across sessions.

**Phase to address:** This phase. The `openTab(tabFiles)` flow should always re-initialize `filesModel`.

### Pitfall TUI-PITFALL-7: Tab Cycling Keys Trapped Inside Files View

**What goes wrong:** If `handleFilesKey` is a hard "swallow all unknown keys" handler (mirroring `handleQRKey` `update.go:157`), then `[` and `]` get consumed inside the Files view and tab cycling stops working when Files is the active tab.

**How to avoid:** EXPLICITLY forward known cross-cutting keys back to the parent. The `handleFilesKey` cascade should:
1. Process its own keys (Up/Down, Enter, Backspace, `/`, `Esc`, PgUp/PgDn, `?`).
2. For `[`, `]`, `Q`, `Ctrl+C`, `?`, `Tab`: re-dispatch via `m.cycleTab(...)` / `tea.Quit` / etc.
3. Default: swallow (return `m, nil`) — DO NOT fall through to the main key handler, that re-enters the file-browser priority slot.

**Warning signs:** Pressing `]` while in Files view does nothing. `?` doesn't open help.

**Phase to address:** This phase. Specify exactly which keys forward and which absorb in the plan.

## Code Examples

Verified patterns from the existing codebase + glamour API.

### Loading a directory asynchronously

```go
// Initial open from Sessions list
m.files = newFilesModel(sid, m.contentWidth(), m.height-3)
m.openTab(tabFiles)
return m, loadDirCmd(m.client, sid, ".")

// Handling the result in Update (above the main switch in update.go:Update):
case filesListMsg:
    if msg.sessionID != m.files.sessionID {
        // stale response — ignore (e.g., user switched session before this returned)
        return m, nil
    }
    m.files.loading = false
    if msg.err != nil {
        m.files.err = msg.err
        return m, nil
    }
    m.files.err = nil
    m.files.cwd = msg.relPath
    m.files.entries = msg.entries
    m.files.truncated = msg.truncated
    m.files.selected = 0
    return m, nil
```

### Preview decision tree using HEAD-then-READ

```go
// User presses Enter on a non-directory entry — preflight to decide preview kind:
case filesHeadMsg:
    if msg.sessionID != m.files.sessionID { return m, nil }
    if msg.err != nil {
        m.files.previewKind = previewErr
        m.files.previewErr = msg.err
        return m, nil
    }
    // 5 MiB cap matches daemon limit (Phase 118 / FS-05)
    const previewCap = 5 * 1024 * 1024
    if msg.size > previewCap {
        m.files.previewKind = previewOverCap
        m.files.preview.SetContent("Too large to preview, use desktop or web to download")
        return m, nil
    }
    // Heuristic: text/* → text/markdown preview; everything else → binary refusal.
    // The daemon's IsBinary field is the authoritative classifier — prefer the
    // entry from the latest ListFiles (m.files.entries) when available.
    if !strings.HasPrefix(msg.mime, "text/") {
        m.files.previewKind = previewBinary
        m.files.preview.SetContent("Use desktop or web to preview")
        return m, nil
    }
    // Issue the actual read.
    m.files.previewLoading = true
    return m, readFileCmd(m.client, msg.sessionID, msg.relPath)

case filesReadMsg:
    if msg.sessionID != m.files.sessionID { return m, nil }
    m.files.previewLoading = false
    if msg.err != nil {
        m.files.previewKind = previewErr
        m.files.previewErr = msg.err
        return m, nil
    }
    // .md or text/markdown → glamour
    isMarkdown := strings.HasSuffix(strings.ToLower(msg.relPath), ".md") ||
                  strings.HasSuffix(strings.ToLower(msg.relPath), ".markdown") ||
                  strings.HasPrefix(msg.mime, "text/markdown")
    if isMarkdown {
        style := "dark"
        if !m.hasDark { style = "light" }
        out, err := glamour.Render(string(msg.data), style)
        if err != nil {
            m.files.previewKind = previewText
            m.files.preview.SetContent(string(msg.data))
        } else {
            m.files.previewKind = previewMarkdown
            m.files.preview.SetContent(out)
        }
    } else {
        m.files.previewKind = previewText
        m.files.preview.SetContent(string(msg.data))
    }
    return m, nil
```

### Left-truncate helper (new utility)

```go
// internal/tui/view.go — add next to truncate()

// truncateLeft truncates from the left, prefixing "…/" when shortened.
// Use for the Files status line where the leaf-end is the high-information
// part. Example: truncateLeft("a/b/c/d/utils/helper.ts", 18) → "…/utils/helper.ts"
func truncateLeft(s string, maxWidth int) string {
    runes := []rune(s)
    if len(runes) <= maxWidth { return s }
    if maxWidth <= 2 { return string(runes[len(runes)-maxWidth:]) }
    // Reserve 2 for "…/", take the rightmost (maxWidth-2) runes
    keep := maxWidth - 2
    // Prefer a path-segment boundary if there's one within the kept window
    tail := runes[len(runes)-keep:]
    if i := strings.IndexRune(string(tail), '/'); i > 0 && i < keep {
        return "…/" + string(tail[i+1:])
    }
    return "…/" + string(tail)
}
```

### Two-pane render with lipgloss.JoinHorizontal

```go
// internal/tui/files.go

func (m Model) renderFilesTab(cw, ch int) string {
    borderColor := m.styles.BorderNormal
    if m.panesFocus == focusContent {
        borderColor = m.styles.BorderAccent
    }

    // Allocate 40% to list, 60% to preview (preview is the high-info pane)
    listW := cw * 40 / 100
    previewW := cw - listW - 1 // -1 for the vertical separator

    listPane := m.renderFilesListPane(listW, ch-2)
    previewPane := m.renderFilesPreviewPane(previewW, ch-2)

    sep := lipgloss.NewStyle().Foreground(m.styles.BorderNormal).
        Render(strings.Repeat("│\n", ch-2))

    body := lipgloss.JoinHorizontal(lipgloss.Top, listPane, sep, previewPane)
    title := fmt.Sprintf(" Files: %s ", m.files.sessionID[:min(8, len(m.files.sessionID))])
    return m.wrapInFrame(body, title, cw, borderColor)
}
```

### Help section per-view

```go
// internal/tui/help.go — buildHelpContent (modified)

switch m.activeTabID() {
case tabFiles:
    sections = append(sections, groupStyle.Render("Files"))
    sections = append(sections,
        formatBinding("Up/Down", "Move cursor"),
        formatBinding("PgUp/PgDn", "Page list / scroll preview"),
        formatBinding("Enter", "Enter directory / preview file"),
        formatBinding("Backspace, Left", "Up one directory"),
        formatBinding("/", "Filter (current dir)"),
        formatBinding("Esc", "Clear filter / dismiss view"),
    )
default:
    // existing Sessions group, plus:
    sections = append(sections, formatBinding("f", "Open files view"))
}
```

## Runtime State Inventory

Not applicable. Phase 121 is greenfield — adds new files (`internal/tui/files.go`, `files_cmds.go`, `files_test.go`) and edits existing TUI files (`model.go`, `update.go`, `view.go`, `keys.go`, `help.go`, `tui.go`). No rename / refactor / migration involved.

Verified:
- **Stored data:** None — no new database keys, collections, or persisted state. The TUI is stateless across restarts.
- **Live service config:** None — no n8n, Datadog, Tailscale ACLs touched.
- **OS-registered state:** None — no Task Scheduler / launchd / systemd / pm2 entries.
- **Secrets / env vars:** None — daemon-local socket needs no cap token (Phase 118 / FS-WEB-01).
- **Build artifacts:** None — `go build ./...` recompiles the TUI from source; no installed packages.

## Open Questions

1. **OQ-1: Remote (tailnet) session support — in-scope for v3.4 or deferred?**
   - What we know: TUI-08 says "works against local AND remote (tailnet) sessions; uses the same daemon-local HTTP API for local and Tailscale HTTPS for remote (no relay frames)." Phase 120 (GUI) made a different call per Pitfall 13: defer remote to v3.5 with explicit "not available" message.
   - What's unclear: For the TUI to call a remote peer's HTTPS webserver, it needs to construct an `http.Client` that hits the remote tailnet IP with the viewer cap token — that's a wholly new pattern not present in the existing TUI (which only talks to the local daemon socket today). This may be a non-trivial cross-cutting addition.
   - Recommendation: **Defer remote in the TUI for v3.4** with a toast "File browser not available for remote sessions in v3.4" when pressing `f` on an `entryRemote` (matches GUI behavior). Plan should confirm with user. Locked decision in CONTEXT.md says daemon-local socket only — interpreting that as "remote out of scope for v3.4 TUI."

2. **OQ-2: Filter input style — bordered or inline?**
   - What we know: CONTEXT.md grants Claude's discretion. The existing `dirInput`/`argsInput` (new-session modal) use plain `textinput.Model` with no border, label-on-left layout.
   - What's unclear: Bordered filter visually distinguishes "you're typing a filter" from "you're navigating." Inline (e.g., prefix "/ filter: foo") is more compact.
   - Recommendation: **Inline.** Prefix with `/` glyph, render in `m.styles.FgAccent`. Saves screen real estate. Mirror the lipgloss footer style.

3. **OQ-3: Preview pane scroll keys — share PgUp/PgDn with list navigation, or require focus toggle?**
   - What we know: TUI-03 locks "PgUp/PgDn (cursor)" for list navigation. The preview pane also wants PgUp/PgDn for scroll.
   - What's unclear: Without a focus indicator (which pane gets PgUp/PgDn), users will be confused. Options: (a) PgUp/PgDn always for list, viewport gets Shift+PgUp/PgDn; (b) Tab cycles focus between list/preview, focused pane gets PgUp/PgDn; (c) Always scroll preview if it's longer than the pane, otherwise paginate list.
   - Recommendation: **(b) Tab cycles focus.** Active pane gets a colored border (BorderAccent vs BorderNormal). The existing `panesFocus` enum already encodes content-vs-sidebar focus — extend the model to add `m.files.previewFocused bool` and toggle on Tab.

4. **OQ-4: What happens to `tabFiles` when the underlying session is killed?**
   - What we know: When a session is killed via `d`, `engine.delete` removes the entry from `sessionWorkDirs`. The next `ListFiles` call returns 404 ("session not found or has no working directory").
   - What's unclear: Should the Files view auto-close on session death, or show a stale "Session no longer running" message?
   - Recommendation: **Show a stale message, don't auto-close.** Auto-close is a surprise behavior. On `filesListMsg.err` matching "session not found", set `m.files.err` to a friendly message and let the user `Esc` out.

## Environment Availability

Not applicable — Phase 121 has no new external tools, services, runtimes, or CLI utilities. All dependencies are already present:

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All | ✓ | 1.26.1 | — |
| `charm.land/bubbletea/v2` | tea.Cmd, tea.Msg | ✓ | v2.0.6 | — |
| `charm.land/bubbles/v2` | viewport, textinput | ✓ | v2.1.0 | — |
| `charm.land/lipgloss/v2` | border, palette, JoinHorizontal | ✓ | v2.0.3 | — |
| `github.com/charmbracelet/glamour` | markdown → ANSI | ✓ (transitive) | v0.8.0 | — — promote to direct dep with `go get @v0.8.0` |
| Phase 118 `DaemonClient.{ListFiles,StatFile,ReadFile,HeadFile}` | All TUI file ops | ✓ | Merged 2026-05-20 | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package + table-driven sub-tests |
| Config file | None (Go convention) |
| Quick run command | `go test -run '^TestFiles' ./internal/tui/ -count=1` |
| Full suite command | `go test ./internal/tui/... -count=1 -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| TUI-01 | `filesModel` constructed with viewport + textinput, lipgloss border applied | unit | `go test -run TestFiles_RenderTwoPane ./internal/tui/` | ❌ Wave 0 |
| TUI-02 | Press `f` on local session entry opens `tabFiles`; on remote shows toast | unit | `go test -run TestFiles_OpenFromSessions ./internal/tui/` | ❌ Wave 0 |
| TUI-03 | Up/Down/PgUp/PgDn move cursor; Enter into dir; Backspace at root no-op | unit | `go test -run TestFiles_Navigation ./internal/tui/` | ❌ Wave 0 |
| TUI-04 | Text → raw; .md → glamour wraps content; binary → refusal string; over-cap → refusal string | unit | `go test -run TestFiles_PreviewKinds ./internal/tui/` | ❌ Wave 0 |
| TUI-05 | `/` activates filter; Escape clears + dismisses; filtered list re-renders | unit | `go test -run TestFiles_Filter ./internal/tui/` | ❌ Wave 0 |
| TUI-06 | Status line shows left-truncated path with `…/leaf` form, count, position | unit | `go test -run 'TestTruncateLeft\|TestFiles_StatusLine' ./internal/tui/` | ❌ Wave 0 |
| TUI-07 | `Update` never calls `os.ReadDir` / `os.Open` synchronously — source-grep guard | unit | `go test -run TestFiles_NoSyncFSCalls ./internal/tui/` (greps `files.go` for forbidden symbols) | ❌ Wave 0 |
| TUI-08 | Local session: tea.Cmd dispatched correctly. Remote: toast + no tab open (per OQ-1 recommendation) | unit | `go test -run TestFiles_RemoteRefusal ./internal/tui/` | ❌ Wave 0 |
| TUI-09 | `?` overlay shows Files section when `activeTabID() == tabFiles` | unit | `go test -run TestFiles_HelpOverlay ./internal/tui/` | ❌ Wave 0 |
| TUI-10 | Key dispatch: with `modal == modalKillConfirm` AND Files active, Backspace cancels kill (not navigate-up) | unit | `go test -run TestFiles_KeyDispatchPriority ./internal/tui/` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -run '^TestFiles' ./internal/tui/ -count=1`
- **Per wave merge:** `go test ./internal/tui/... -count=1 -race && go vet ./internal/tui/...`
- **Phase gate:** `go test ./internal/... -count=1 -race && go build ./...` green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/tui/files_test.go` — covers TUI-01..TUI-10 (table-driven sub-tests, one per requirement)
- [ ] `testModel()` extension — already exists in `update_test.go:14`; reuse and add a `withFilesTab()` helper that calls `m.openTab(tabFiles)` + injects `m.files = newFilesModel(...)`
- [ ] No new framework install required — Go `testing` is stdlib and already in use across `internal/tui/` (10 existing test files: `attach_test.go`, `help_test.go`, `modal_test.go`, `styles_test.go`, `update_test.go`, `view_test.go`)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Daemon-local Unix socket / named pipe is the trust boundary (Phase 118 / WEB-01); TUI sends no credentials |
| V3 Session Management | no | TUI passes a session ID (string); the daemon owns session lifecycle |
| V4 Access Control | partial | TUI must NOT bypass daemon-enforced sandbox; defense is server-side (`os.Root`), TUI is a client |
| V5 Input Validation | yes | TUI must not crash / panic on malformed `FileEntry` (e.g., `Name=""`, `Size<0`, embedded NUL) returned by daemon — defensive `if name == "" { continue }` in the render loop |
| V6 Cryptography | no | No keys, no certs, no signing in this phase |

### Known Threat Patterns for Bubble Tea TUI consuming Phase 118 DaemonClient

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Long-running daemon RPC blocks TUI render loop (DoS via slow disk / NFS) | Denial-of-service | Wrap every client call in `context.WithTimeout(ctx, 5s)`; on timeout return a typed `filesErrMsg{err: ctx.Err()}` |
| TUI re-implements path sanitization client-side, gets it wrong, allows traversal | Tampering | DO NOT validate paths client-side. Send raw user input to daemon. `os.Root` is the only correct enforcement layer (PITFALLS.md Pitfall 1, fuzz-proven). |
| Malformed `FileEntry.Name` containing ANSI escape sequences (terminal injection) | Spoofing / Information disclosure | Sanitize before render. Use `ansi.Strip()` from `github.com/charmbracelet/x/ansi` (already in `go.mod`) on every `entry.Name` before passing to lipgloss. The daemon controls the filesystem, but a misbehaving user could create a file named with literal ANSI escapes. |
| Stale `filesListMsg` for session A arrives after user switched to session B | Race / Confusion | Compare `msg.sessionID == m.files.sessionID` before applying; discard stale msgs silently (see Code Examples). |
| Preview pane retains 5 MiB content after navigation, memory leak | DoS via local resource exhaustion | Clear `viewport.SetContent("")` on every navigation; zero `m.files.preview` when closing `tabFiles`. |
| Cap-token leakage: TUI accidentally logs a cap token | Information disclosure | N/A — TUI uses Unix socket; never holds a cap token. |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | "Closing the Files view returns to the prior TUI tab" is satisfied by NOT removing `tabFiles` from `openTabs` but rather cycling away via `[`/`]`. The user can also leave the tab open and use `[`/`]` to flip between Files and Sessions. | TUI-02 interpretation | If user expects `Esc` to close the tab entirely (remove from `openTabs`), need to add a tab-close mechanism. CONTEXT.md is ambiguous. Recommendation: planner clarifies with user; default to "tab stays open, Esc clears filter only." |
| A2 | The TUI's existing color profile detection (`tea.BackgroundColorMsg` → `m.hasDark`) is sufficient input for glamour's style selection. | Pitfall TUI-PITFALL-4 mitigation | If terminals report background color incorrectly, glamour may render mismatched. Workaround: planner adds a Settings flag for "force light/dark markdown style" if reports surface. |
| A3 | Phase 121 is local-session-only in v3.4 (defer remote tailnet sessions to v3.5). | OQ-1 recommendation; TUI-08 partial fulfilment | If user wants remote in v3.4, scope expands meaningfully — requires new `DaemonClient`-like helper for remote tailnet peers, cap token routing, TLS validation. Planner MUST confirm before plans freeze. |
| A4 | `bubbles/v2/viewport` (in `charm.land/bubbles/v2 v2.1.0`) supports `SetContent` with ANSI-styled string and renders correctly. | Standard Stack §Core | High confidence — viewport is the standard scrolling pane and explicitly handles ANSI per glamour docs. Mitigation: integration test verifies markdown preview rendering doesn't double-escape ANSI. |
| A5 | `glamour@v0.8.0`'s "dark" style is a reasonable default that doesn't conflict catastrophically with TokyoNight. | Pitfall TUI-PITFALL-4 | Visual taste — not a correctness issue. Easy to adjust post-merge if feedback is negative. |

## Sources

### Primary (HIGH confidence)
- `go list -m github.com/charmbracelet/glamour charm.land/bubbles/v2 charm.land/bubbletea/v2 charm.land/lipgloss/v2` — verified v0.8.0 / v2.1.0 / v2.0.6 / v2.0.3 against the local module graph
- `internal/daemon/client.go:376-484` — `DaemonClient.ListFiles / StatFile / ReadFile / HeadFile` signatures, error shapes, ctx requirement
- `internal/files/types.go:26-46` — `FileEntry` and `FileListResponse` JSON shape and field semantics
- `internal/tui/update.go:114-153` — existing 6-level key dispatch priority cascade
- `internal/tui/cmds.go:13-72` — existing `tea.Cmd` patterns (`fetchSessions`, `fetchWebStatus`, `nextTick`, `createSession`, `killSession`, `renameSession`, `fetchRemoteSessions`)
- `internal/tui/view.go:42-165` — existing tab dispatch (`renderContentPane` switch on `activeTabID()`)
- `internal/tui/styles.go:52-89` — existing TokyoNight `Styles` token set and `newStyles` constructor
- `internal/tui/model.go:50-55` — existing `tabID` iota (Phase 121 adds `tabFiles`)
- `internal/tui/help.go:37-88` — existing help overlay `buildHelpContent` structure
- `.planning/research/STACK.md` — v3.4 milestone STACK (glamour promotion, viewport recommendation, filepicker rejection)
- `.planning/research/PITFALLS.md` — Pitfalls 7 (sync I/O), 12 (TUI key routing), 13 (singleton/remote sessions)
- `.planning/phases/118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi/118-05-SUMMARY.md` — DaemonClient method shapes, error mapping, 5 MiB cap enforcement location

### Secondary (MEDIUM confidence)
- [charmbracelet/glamour v0.8.0 pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/glamour@v0.8.0) — Render API and built-in styles (per STACK.md citation)
- [charm.land/bubbles/v2 viewport pkg.go.dev](https://pkg.go.dev/charm.land/bubbles/v2/viewport) — SetContent / scroll API (per STACK.md citation)

### Tertiary (LOW confidence)
- None. All claims are backed by either local source inspection or already-verified upstream milestone research.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every dep is already in `go.sum`; `go list -m` confirmed versions in this session
- Architecture: HIGH — patterns mirror existing AgentHub TUI code paths (cmds.go, update.go priority cascade, view.go dispatch, help.go sections)
- Pitfalls: HIGH — drawn from in-repo PITFALLS.md research with TUI-specific entries (Pitfalls 7, 12, 13) plus four additional discovered during this read (memory leak in preview pane, color-profile mismatch, truncation signal handling, stale-msg race)

**Research date:** 2026-05-20
**Valid until:** 2026-06-19 (30 days; stable Bubble Tea v2 ecosystem, no upstream churn expected)

## RESEARCH COMPLETE
