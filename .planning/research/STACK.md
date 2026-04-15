# Stack Research

**Domain:** Multi-client WebSocket fan-out, CLI status bar, TUI mode for Go/Wails terminal manager
**Researched:** 2026-04-14
**Confidence:** HIGH (all versions verified via GitHub API and Go module proxy; architecture verified against existing codebase)

---

## Context: What Already Exists

The relay fan-out infrastructure is **already implemented** (`internal/relay/`). `Hub` supports N subscribers per session via channel fan-out with slow-client drop. `Scrollback` provides 256 KiB ring buffer. `Server` handles subscribe-before-snapshot ordering to prevent gap. The multi-client feature (GitHub #13) is **a daemon wiring and protocol gap**, not a missing library — the relay layer already works.

The status bar (GitHub #8) needs: ANSI cursor-save/restore sequences to render a persistent bottom bar without alternate screen, plus an elapsed-time ticker.

The TUI mode (GitHub #7) needs: a new binary entry point using a full TUI framework.

---

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `charm.land/bubbletea/v2` | v2.0.5 | TUI framework for TUI mode | MVU architecture composes naturally with Go; v2 Cursed Renderer handles resize and repaint correctly; 41k GitHub stars, active Charmbracelet org maintenance; far superior testing story vs tview |
| `charm.land/lipgloss/v2` | v2.0.3 | Layout and styling for both status bar and TUI | CSS-like declarative API; works standalone (no Bubble Tea required) for the status bar use case; v2 deterministic rendering eliminates v1 I/O lock-up bugs |
| `charm.land/bubbles/v2` | v2.1.0 | Pre-built TUI components | Viewport (scrollable content), list, table, spinner, text input — all compatible with Bubble Tea v2; avoids writing standard widgets from scratch |
| `github.com/charmbracelet/x/ansi` | v0.11.7 | ANSI sequences for status bar | DECSC/DECRC cursor save-restore, MoveCursor, EraseLineLeft — needed to draw/clear the persistent bottom bar in pass-through raw mode without alternate screen |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/charmbracelet/x/exp/teatest` | v0.0.0-20260413... | Bubble Tea v2 test helper | Unit testing TUI models with `NewTestModel`, fixed terminal dimensions, golden output; use for every non-trivial Bubble Tea model |
| `github.com/taigrr/bubbleterm` | v0.2.0 | PTY terminal widget inside Bubble Tea | Embeds a live PTY session as a scrollable Bubble Tea component; uses `creack/pty` (already in go.mod as indirect dep); needed for TUI mode to display agent sessions |
| `golang.org/x/term` | already in go.mod | Terminal raw mode, size query | Already used in `cmd_attach.go`; needed for status bar elapsed ticker and resize signal handling |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `tea.WithWindowSize(w,h)` | Hermetic test sizing | Pass `80, 24` in tests to get deterministic output without mocking a terminal |
| `teatest.NewTestModel` | Bubble Tea model test harness | Captures program output for assertion; keep golden files with `text eol=lf` in `.gitattributes` to prevent CRLF corruption |

---

## Installation

```bash
# TUI framework (Bubble Tea v2 ecosystem)
go get charm.land/bubbletea/v2@v2.0.5
go get charm.land/lipgloss/v2@v2.0.3
go get charm.land/bubbles/v2@v2.1.0

# ANSI sequences (status bar)
go get github.com/charmbracelet/x/ansi@v0.11.7

# PTY terminal widget (TUI mode — embeds live sessions)
go get github.com/taigrr/bubbleterm@v0.2.0

# Testing
go get github.com/charmbracelet/x/exp/teatest
```

Note: `charm.land/*` and `github.com/charmbracelet/*` are mirrors — both resolve to the same module at the same version. Use `charm.land` vanity paths in import statements (it's the canonical v2 path per upstream).

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Bubble Tea v2 (TUI) | tview v0.42.0 | Never for this project — see detailed comparison below |
| `x/ansi` (status bar) | Raw `fmt.Fprintf` with hardcoded ESC strings | Acceptable for a one-off, but `x/ansi` is already a transitive dep via charmbracelet packages and provides named constants that survive code review |
| `bubbleterm` (PTY widget) | Roll custom ANSI-parsing widget | Only if bubbleterm proves too limited (it's pre-v1, spec coverage is incomplete); bubbleterm explicitly notes it may replace the internal emulator library — keep it behind an interface |
| `charm.land/bubbles/v2` viewport | Custom scrollback viewer | Only if multi-column layout or mouse selection is needed; the bubbles viewport supports soft-wrap, gutters, and mouse wheel out of the box |

---

## TUI Framework Comparison: Bubble Tea v2 vs tview

### Layout System

**Bubble Tea v2:** Layout is computed in the `View()` function using Lipgloss `Width()`, `Height()`, and `Join*` helpers. The model owns all dimensions; resize propagates as `tea.WindowSizeMsg`. Layout is code — no widget hierarchy, no retained tree. This integrates naturally with the existing Go codebase style.

**tview:** Layout uses `Flex` and `Grid` primitives with a retained widget tree. `Grid` supports responsive breakpoints (show/hide columns at N chars wide). More familiar to developers coming from GUI frameworks, but requires managing a widget object graph with its own internal state.

**Verdict for this project: Bubble Tea v2.** The daemon already manages session state as pure data structures. Projecting that state into a string via `View()` is simpler than mapping it onto a widget tree.

### Widget Support

**Bubble Tea v2 + Bubbles v2:** Viewport, list (with filtering), table, spinner, text input, textarea, progress bar, paginator. All components are pure `tea.Model` values — composable without lifecycle hooks.

**tview:** Richer out-of-the-box widget set: TreeView, Form, DropDown, Modal, Pages. Useful for database browsers and admin UIs.

**Verdict: tview** has more widgets overall, but **Bubble Tea** has every widget this project needs. tview's extras (form, tree) are not used here.

### PTY Terminal Embedding (Critical for TUI Mode)

**Bubble Tea v2:** `bubbleterm` (v0.2.0) embeds a PTY as a Bubble Tea component. Parses ANSI sequences, maintains cursor state, renders frames as ANSI-preserved strings. Uses `creack/pty` which is already an indirect dep. The TUI mode requires displaying multiple live AI agent sessions — this is the critical path.

**tview:** No native PTY widget. The `ANSIWriter()` helper translates a subset of ANSI color sequences into tview color tags, but does not handle cursor positioning, alternate screen, or full VT100 emulation. Embedding a live PTY session requires building a custom widget from scratch — no maintained third-party option was found in the 2025-2026 Go ecosystem.

**Verdict: Bubble Tea v2 wins.** This is disqualifying for tview. The TUI mode must display live AI coding agent output which uses full xterm-256color sequences (OpenCode, Claude Code both require this). tview's ANSI support covers colorized text only.

### Theming

**Bubble Tea v2 + Lipgloss v2:** Color profiles auto-detect terminal capabilities (true color, 256-color, ANSI, no-color). v2 styles are deterministic (v1 had I/O contention bugs with concurrent renders). The existing 138 xterm theme hex values feed directly into `lipgloss.Color()`.

**tview:** Theming via `tcell.Style` on individual primitives. No equivalent of Lipgloss style inheritance. No bulk theme apply.

**Verdict: Bubble Tea v2.** Lipgloss v2 color profiles compose with the existing theme data with no conversion.

### Testing

**Bubble Tea v2:** `teatest` provides `NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))`. Send key messages, tick the clock, assert `FinalOutput()` against golden strings. `tea.WithWindowSize(w, h)` enables deterministic test sizes directly. MVU architecture means `Update()` is a pure function — unit testable with zero framework overhead.

**tview:** `Application.Run()` panics when called from `go test` (rivo/tview issue #591, open 2021-present). Tests using `tcell.SimulationScreen` hit timeouts waiting for the event loop. No first-party test helper. Workarounds require running in a subprocess or manual event injection.

**Verdict: Bubble Tea v2.** The tview testing limitation is a structural issue with no official fix. The existing AgentHub codebase has 8K lines of Go tests; adding an untestable UI layer is a regression.

### Community Health

| Metric | Bubble Tea v2 | tview |
|--------|--------------|-------|
| GitHub Stars | 41,562 | 13,768 |
| Last Updated | 2026-04-15 | 2026-04-15 |
| Org Backing | Charmbracelet (company) | Individual maintainer (rivo) |
| Versioning | Stable v2.0.5 | v0.42.0 (semver pre-v1 technically) |
| Ecosystem | lipgloss, bubbles, wish, x/ansi | tcell only |

Both are actively maintained. Bubble Tea wins on ecosystem breadth and organizational backing.

### Summary Verdict: Use Bubble Tea v2

The PTY embedding gap in tview is a hard blocker. The untestable Application.Run() is a second hard blocker. Bubble Tea v2 + bubbleterm covers the PTY case, integrates with the existing theme data via Lipgloss, and matches the project's testing standard.

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| tview for TUI mode | No PTY/full-terminal embedding; Application.Run() untestable from go test | Bubble Tea v2 + bubbleterm |
| Bubble Tea v1 (`github.com/charmbracelet/bubbletea` without `/v2`) | v2 is stable; v1 has known lipgloss I/O contention bugs; v2 Cursed Renderer is orders of magnitude more efficient | `charm.land/bubbletea/v2` |
| gorilla/websocket for new code | Already replaced by coder/websocket; panics on concurrent writes | `github.com/coder/websocket` (already in go.mod) |
| New fan-out library for multi-client | `internal/relay` Hub already implements multi-subscriber fan-out with slow-client drop and scrollback replay | Wire daemon to create Hub per session on creation, not on first attach |
| Redis/NATS/external broker for fan-out | Unnecessary; all clients attach to the same daemon process | In-process Hub with `sync.Mutex` + buffered channel per subscriber (already built) |
| `termkit/skeleton` for TUI tab management | GPL-3.0 license incompatible with project's likely MIT/BSD direction; 62 stars, pre-v1 | Implement tab routing directly in root Bubble Tea model (`currentTab int` + `[]tea.Model` slice) |
| `bubbleterm` without an interface wrapper | v0.2.0 is pre-v1; README explicitly warns the emulator library may be swapped | Wrap behind a `TerminalWidget interface { Update(tea.Msg) (TerminalWidget, tea.Cmd); View() string }` |

---

## Stack Patterns by Feature

**Multi-client fan-out (GitHub #13) — no new library needed:**
- The `internal/relay` Hub already supports N subscribers
- Current gap: Hub lifetime may be tied to the first client connection; Hub must be created at session creation time and persist across all client attach/detach cycles
- Second client subscribes to existing Hub → gets scrollback replay automatically via existing `ScrollbackSnapshot()` + subscribe-before-snapshot pattern
- Work: plumb Hub creation into the session engine's `CreateSession` path; manage Hub lifecycle in `HubManager`

**tmux-style CLI status bar (GitHub #8) — lipgloss + x/ansi, no Bubble Tea:**
- Use `github.com/charmbracelet/x/ansi` constants: `SaveCursor`, `MoveCursor(rows, 0)`, `EraseLineRight`, `RestoreCursor`
- Use `charm.land/lipgloss/v2` to render the status line string: `[session | agent | host | HH:MM:SS | Ctrl-\\ to detach]`
- Use `golang.org/x/term.GetSize()` for terminal width at render time
- Goroutine: tick every second to update elapsed time; re-render on SIGWINCH
- Write to `os.Stderr` to avoid mixing with raw PTY stdout stream
- Pattern: `DECSC → CUP(rows,0) → EL → render → DECRC` — no alternate screen, no cursor flash

**TUI mode (GitHub #7) — Bubble Tea v2 + bubbleterm:**
- Entry: `agenthub tui` subcommand (or auto-detect headless environment)
- Root model: tab bar (session tabs + New), active session panel fills `height - tabBarHeight - statusBarHeight`
- Session panel: `bubbleterm.Model` receiving frames from daemon WebSocket connection
- Session list sidebar: `bubbles/v2/list.Model`, toggleable
- Footer: lipgloss-styled row showing agent type, hostname, web URL, session count
- Daemon connectivity: reuse existing `daemon.DaemonClient` HTTP/JSON + relay WebSocket — TUI is a new front-end to the existing daemon, not a new daemon

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `charm.land/bubbletea/v2@v2.0.5` | `charm.land/lipgloss/v2@v2.0.3` | v2 of both; designed together by Charmbracelet |
| `charm.land/bubbles/v2@v2.1.0` | `charm.land/bubbletea/v2@v2.0.5` | bubbles v2 targets bubbletea v2; do not mix bubbles v1 with bubbletea v2 |
| `github.com/taigrr/bubbleterm@v0.2.0` | `charm.land/bubbletea/v2` | Verify import path targets v2 before adding; if it imports bubbletea v1, wrap it or fork the relevant code |
| `github.com/charmbracelet/x/ansi@v0.11.7` | standalone | No bubbletea dependency; safe in status bar code running outside any TUI framework |
| `github.com/charmbracelet/x/exp/teatest` | `charm.land/bubbletea/v2` | `x/exp` = unstable API; pin to a commit if API drift causes breakage between milestone phases |

---

## Sources

- Context7 `/charmbracelet/bubbletea` — View struct, WindowSizeMsg, layout patterns, test sizing via `tea.WithWindowSize`
- Context7 `/charmbracelet/lipgloss` — border styles, table rendering, color profiles
- Context7 `/charmbracelet/bubbles` — viewport with soft-wrap and gutters, textarea, list
- Context7 `/rivo/tview` — grid layout, flex, input field, focus delegation; no PTY widget found
- GitHub API (2026-04-14): bubbletea 41,562 stars; tview 13,768 stars; lipgloss 11,050; bubbles 8,198
- Go module proxy: bubbletea v2.0.5, lipgloss v2.0.3, bubbles v2.1.0, tview v0.42.0, tcell v3.1.2, x/ansi v0.11.7
- https://pkg.go.dev/github.com/taigrr/bubbleterm — PTY terminal widget v0.2.0, 0BSD license, pre-v1
- https://github.com/charmbracelet/bubbletea/discussions/1374 — Bubble Tea v2 confirmed: Cursed Renderer, Mode 2026 sync, `tea.WithWindowSize` test support
- https://github.com/rivo/tview/issues/591 — Application.Run() panics under go test (open, structural)
- https://github.com/rivo/tview/issues/326 — SimulationScreen testing hits timeouts (confirmed community pain)
- Codebase: `internal/relay/hub.go`, `server.go`, `scrollback.go` — fan-out infrastructure exists; multi-client is a wiring gap not a library gap

---

*Stack research for: AgentHub v2.0 — multi-client sessions, CLI status bar, TUI mode*
*Researched: 2026-04-14*
