# Phase 121: TUI Files View - Context

**Gathered:** 2026-05-20
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

TUI users can browse and preview files for any session using keyboard navigation inside a lipgloss-bordered file browser pane — with the same sandboxed cwd constraint, type-ahead filter, and text/markdown preview available in the desktop and web surfaces.

**Requirements:** TUI-01, TUI-02, TUI-03, TUI-04, TUI-05, TUI-06, TUI-07, TUI-08, TUI-09, TUI-10

</domain>

<decisions>
## Implementation Decisions

### Locked (from ROADMAP success criteria)
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

</decisions>

<code_context>
## Existing Code Insights

- TUI lives in `internal/tui/` (Bubble Tea + lipgloss + Charmbracelet stack)
- Existing tab system (sessions tab, plugin streams, etc.)
- Existing lipgloss border style + TokyoNight palette tokens to reuse
- Existing help overlay system (`?` key)
- `internal/daemon/client.go` Phase 118 methods: ListFiles, StatFile, ReadFile, HeadFile
- glamour likely already a dependency (used elsewhere for markdown?) or needs adding

</code_context>

<specifics>
## Specific Ideas

Components likely needed:
- `tabFiles` model state in main TUI model (alongside tabSessions, etc.)
- `filesUpdate` + `filesView` functions (or files.go module)
- `filesListMsg`, `filesStatMsg`, `filesReadMsg`, `filesErrMsg` `tea.Msg` types
- `loadDirCmd`, `loadFileCmd`, `statFileCmd` `tea.Cmd` constructors
- Filter input as either a sub-bubble or simple string-state
- `pathTruncate` helper for status line left-truncation (`…/foo/bar.ts`)

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
