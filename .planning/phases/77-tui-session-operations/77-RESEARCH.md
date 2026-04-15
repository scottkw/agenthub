# Phase 77: TUI Session Operations - Research

**Researched:** 2026-04-15
**Domain:** Bubble Tea v2 exec/suspend pattern, Bubbles v2 textinput, modal state machines, daemon RPC integration
**Confidence:** HIGH

## Summary

Phase 77 extends the Phase 76 TUI foundation with four session lifecycle operations: attach (via `tea.Exec`), create (modal form), kill (confirmation dialog), and rename (inline edit). The UI design contract is locked in 77-UI-SPEC.md. This research focuses entirely on implementation mechanics.

The primary technical challenge is the attach flow (TUI-03). Bubble Tea v2's `tea.Exec` mechanism releases the terminal (exits alt-screen, stops the renderer, cancels input reader), runs a blocking `ExecCommand.Run()`, then restores the terminal (re-enters alt-screen, restarts renderer, reinitializes input). The existing `cmd_attach.go` attach logic (WebSocket dial, raw mode, status bar, I/O pumps, Ctrl-\ detach) must be wrapped in an `ExecCommand` implementation. The critical insight: `tea.Exec` provides stdin/stdout/stderr via `SetStdin`/`SetStdout`/`SetStderr` -- the attach wrapper must use THESE handles instead of `os.Stdin`/`os.Stdout` directly. This was verified by reading the actual Bubble Tea v2 source code (`exec.go` lines 101-129).

The secondary challenges are well-understood modal/inline-edit patterns: a `textinput.Model` from `bubbles/v2/textinput` (already available via the bubbles/v2 module in go.mod) handles cursor movement, character editing, and clipboard. The new-session modal manages a 3-field form (agent picker + 2 text inputs) with focus cycling. The kill confirmation is a minimal Yes/No toggle. Inline rename replaces the Name column with a textinput. All daemon RPC calls (`CreateSession`, `KillSession`, `RenameSession`) already exist in `internal/daemon/client.go` and must be wrapped in `tea.Cmd` closures to avoid blocking the MVU loop.

**Primary recommendation:** Implement a new `attachCmd` struct satisfying `tea.ExecCommand` that wraps the existing `attachSession()` function. Extract common attach setup logic (WebSocket dial, session lookup, status bar creation, raw mode) into a shared helper callable from both `cmd_attach.go` and the new `attachCmd.Run()`. Add modal/editing state fields to the existing Model and route keys through a priority-based dispatch: editing > modal > help > main view.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TUI-03 | User can attach to a session from the list (TUI suspends, raw PTY attach, TUI resumes on detach) | `tea.Exec` + `ExecCommand` interface verified in bubbletea v2.0.5 source; attach logic reuse via `attachSession()` documented; full attach sequence mapped |
| TUI-04 | User can create a new session via modal (agent picker, working directory, extra args) | `textinput.New()` API verified in bubbles v2.1.0; `pty.DetectCLIs()` returns `[]DetectedCLI{Name, DisplayName, Path}`; `client.CreateSession()` signature documented |
| TUI-05 | User can kill a session with confirmation dialog | `client.KillSession(id)` is DELETE /sessions/{id}; compact modal with Yes/No toggle; default No for safety |
| TUI-06 | User can rename a session via inline edit or modal | `client.RenameSession(id, name)` is PATCH /sessions/{id}/name; inline textinput replaces Name column; no duplicate-name validation (daemon allows duplicates) |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Attach flow orchestration | `internal/tui` (ExecCommand) | `cmd_attach.go` (shared logic) | TUI dispatches `tea.Exec`; the ExecCommand wrapper calls shared attach logic |
| Terminal release/restore | `bubbletea/v2` Program | -- | Bubble Tea owns alt-screen exit/enter, raw mode, renderer stop/start |
| WebSocket I/O during attach | `cmd_attach.go` (attachSession) | `internal/relay` | Existing stdinPump/wsOutputPump run inside ExecCommand.Run() |
| New session modal UI | `internal/tui` (modal sub-state) | `bubbles/v2/textinput` | TUI renders modal; textinput handles field editing |
| Agent detection | `internal/pty` (DetectCLIs) | `internal/tui` (cached) | pty.DetectCLIs() called once, cached for TUI lifetime |
| Session CRUD operations | `internal/daemon/client.go` | `internal/tui/cmds.go` (tea.Cmd) | DaemonClient provides typed HTTP methods; TUI wraps in async Cmd |
| Kill confirmation UI | `internal/tui` (modal sub-state) | -- | Simple Yes/No toggle, no external components |
| Inline rename UI | `internal/tui` (editing state) | `bubbles/v2/textinput` | textinput replaces Name column during edit |
| Toast notifications | `internal/tui` (model state) | -- | Struct with text + expiry; rendered in footer; tea.Tick for auto-dismiss |
| Key dispatch routing | `internal/tui/update.go` | -- | Priority: editing > modal > help > main view |

## Standard Stack

### Core (already in go.mod from Phase 76)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.0.5 | MVU framework, `tea.Exec` for attach | [VERIFIED: `go list -m` 2026-04-15] |
| `charm.land/lipgloss/v2` | v2.0.3 | Modal borders, adaptive colors, layout | [VERIFIED: `go list -m` 2026-04-15] |
| `charm.land/bubbles/v2` | v2.1.0 | `textinput.Model` for rename/modal fields, `key` for bindings | [VERIFIED: `go list -m` 2026-04-15] |

### Supporting (already in go.mod)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/term` | v0.41.0 | `term.MakeRaw`, `term.GetSize` inside attach | [VERIFIED: go.mod] Raw mode during attach |
| `github.com/coder/websocket` | (in go.mod) | WebSocket dial/read/write during attach | [VERIFIED: go.mod] Used by attachSession() |
| `internal/daemon` | -- | DaemonClient: CreateSession, KillSession, RenameSession | [VERIFIED: client.go] All methods exist |
| `internal/pty` | -- | DetectCLIs() for agent picker | [VERIFIED: detect.go] Returns []DetectedCLI |
| `internal/relay` | -- | MakeInputFrame, ParseFrame for attach I/O | [VERIFIED: cmd_attach.go uses it] |
| `internal/statusbar` | -- | Status bar during attach | [VERIFIED: cmd_attach.go creates Bar] |

### No New Dependencies Required

All code in Phase 77 uses libraries already in go.mod. The `textinput` sub-package is part of `charm.land/bubbles/v2` which was added in Phase 76. No `go get` needed.

## Architecture Patterns

### System Architecture Diagram

```
                        internal/tui/ (MVU)
                        ==================

User Key Press
  |
  v
Update(tea.KeyPressMsg)
  |
  +--[editing?]----> handleRenameKey()
  |                    Enter -> renameSession cmd -> renameSessionMsg
  |                    Esc   -> discard, exit editing
  |                    other -> delegate to textinput.Update
  |
  +--[modal==kill?]--> handleKillKey()
  |                    y/Enter(on Yes) -> killSession cmd -> killSessionMsg
  |                    n/Esc/Enter(on No) -> close dialog
  |                    Left/Right/Tab -> toggle Yes/No
  |
  +--[modal==new?]---> handleNewSessionKey()
  |                    Tab/Shift+Tab -> cycle focus (agent/dir/args)
  |                    Left/Right (agent focused) -> cycle agent picker
  |                    Enter -> validate -> createSession cmd -> createSessionMsg
  |                    Esc   -> close modal
  |                    other -> delegate to textinput.Update (if text field focused)
  |
  +--[showHelp?]-----> ? or Esc -> close help
  |
  +--[main view]-----> dispatch:
       Enter -> tea.Exec(attachCmd, callback) -> attachDoneMsg
       n     -> open new-session modal
       d     -> open kill confirmation
       r     -> enter inline rename
       R     -> refresh
       j/k   -> navigate
       ...

                      tea.Exec flow (attach)
                      =====================

 tea.Exec(attachCmd, callback)
   |
   +--> bubbletea: releaseTerminal (exit alt-screen, stop renderer, cancel reader)
   |
   +--> attachCmd.SetStdin(programInput)
   +--> attachCmd.SetStdout(programOutput)
   +--> attachCmd.SetStderr(os.Stderr)
   |
   +--> attachCmd.Run()
   |      |
   |      +--> client.GetRelayPort()
   |      +--> client.ListSessions() -> find session
   |      +--> websocket.Dial(wsURL)
   |      +--> term.MakeRaw(stdin.Fd())
   |      +--> statusbar.New(lockedWriter{stdout}, opts).Start()
   |      +--> attachSession(ctx, conn, stdin, stdout, detachKey, bar, nil)
   |      |      |
   |      |      +--> stdinPump (reads stdin, sends to WS, detects Ctrl-\)
   |      |      +--> wsOutputPump (reads WS, writes to stdout)
   |      |      +--> <blocks until detach or disconnect>
   |      |
   |      +--> bar.Stop()
   |      +--> term.Restore(stdin.Fd(), oldState)
   |      +--> return nil (clean detach) or error
   |
   +--> bubbletea: RestoreTerminal (re-enter alt-screen, restart renderer)
   |
   +--> callback(err) -> sends attachDoneMsg to Update
   |
   +--> Update(attachDoneMsg) -> refresh sessions, show error toast if any
```

### Recommended Project Structure (Phase 77 additions)

```
internal/tui/
  model.go          # EXTEND: add modal/editing state fields, new message types
  update.go         # EXTEND: priority-based key dispatch, new message handlers
  view.go           # EXTEND: modal/inline-edit rendering, updated hint bar
  help.go           # MODIFY: update keybinding groups (rename r, refresh R, add d)
  keys.go           # MODIFY: reassign r=rename, R=refresh, add d=kill, r=rename
  styles.go         # EXTEND: add modal color tokens (bg.modal, fg.danger, etc.)
  cmds.go           # EXTEND: add createSession, killSession, renameSession cmds
  tui.go            # MODIFY: cache DetectCLIs in newModel or on first modal open
  attach.go         # NEW: attachCmd struct implementing tea.ExecCommand
  modal.go          # NEW: new-session modal rendering + agent picker
  *_test.go         # EXTEND: tests for all new state transitions and views
```

### Pattern 1: tea.Exec for Attach (TUI-03)

**What:** Implement `tea.ExecCommand` interface to run the full attach flow while Bubble Tea pauses.
**When to use:** Whenever the TUI needs to yield full terminal control to another process.

```go
// Source: [VERIFIED: bubbletea v2.0.5 exec.go lines 58-65]
// ExecCommand interface:
type ExecCommand interface {
    Run() error
    SetStdin(io.Reader)
    SetStdout(io.Writer)
    SetStderr(io.Writer)
}
```

Implementation pattern:

```go
// attach.go
type attachCmd struct {
    client    *daemon.DaemonClient
    sessionID string
    stdin     io.Reader
    stdout    io.Writer
    stderr    io.Writer
}

func (a *attachCmd) SetStdin(r io.Reader)  { a.stdin = r }
func (a *attachCmd) SetStdout(w io.Writer) { a.stdout = w }
func (a *attachCmd) SetStderr(w io.Writer) { a.stderr = w }

func (a *attachCmd) Run() error {
    // 1. Get relay port
    port, err := a.client.GetRelayPort()
    if err != nil {
        return err
    }

    // 2. Find session metadata
    sessions, err := a.client.ListSessions()
    if err != nil {
        return err
    }
    var session *daemon.SessionInfo
    for _, s := range sessions {
        if s.ID == a.sessionID {
            s := s
            session = &s
            break
        }
    }
    if session == nil {
        return fmt.Errorf("session %q not found", a.sessionID)
    }

    // 3. Build WebSocket URL
    wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sessions/%s/ws", port, a.sessionID)

    // 4. Dial WebSocket
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    conn, _, err := websocket.Dial(ctx, wsURL, nil)
    if err != nil {
        return err
    }
    defer conn.CloseNow()

    // 5. Wrap stdout for status bar serialization
    lw := &lockedWriter{w: a.stdout}

    // 6. Status bar (if stdin is a file with Fd)
    var bar *statusbar.Bar
    if f, ok := a.stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
        // ... create bar with session metadata ...
        bar = statusbar.New(lw, opts)
        bar.Start()
        defer bar.Stop()
    }

    // 7. Raw mode
    if f, ok := a.stdin.(*os.File); ok {
        oldState, err := term.MakeRaw(int(f.Fd()))
        if err != nil {
            return err
        }
        defer term.Restore(int(f.Fd()), oldState)

        // Initial resize
        if cols, rows, err := term.GetSize(int(f.Fd())); err == nil {
            frame := makeClientResizeFrame(uint16(cols), uint16(rows))
            _ = conn.Write(ctx, websocket.MessageBinary, frame)
        }
    }

    // 8. Run I/O pumps (blocks until detach)
    return attachSession(ctx, conn, a.stdin, lw, 0x1C, bar, nil)
}
```

**Critical detail:** `tea.Exec` calls `releaseTerminal(false)` which exits alt-screen but does NOT reset the terminal to cooked mode. The `attachCmd.Run()` method must call `term.MakeRaw()` itself. After `Run()` returns, Bubble Tea calls `RestoreTerminal()` which re-enters alt-screen, restarts the renderer, and reinitializes input -- the TUI picks up right where it left off. [VERIFIED: bubbletea v2.0.5 exec.go lines 101-129 and tea.go lines 1323-1360]

**Dispatch in Update:**

```go
case key.Matches(msg, m.keys.Attach):
    if len(m.sessions) == 0 || m.sessions[m.selected].Status == "errored" {
        m.toast = "Session not available"
        m.toastExp = time.Now().Add(2 * time.Second)
        return m, nil
    }
    cmd := &attachCmd{
        client:    m.client,
        sessionID: m.sessions[m.selected].ID,
    }
    return m, tea.Exec(cmd, func(err error) tea.Msg {
        return attachDoneMsg{err: err}
    })
```

### Pattern 2: Modal Sub-State for New Session (TUI-04)

**What:** Add modal state to the existing Model, route keys based on modal state.
**When to use:** Any overlay that captures keyboard input.

The model gains these fields (per UI-SPEC):

```go
// model.go additions
type modalState int
const (
    modalNone modalState = iota
    modalNewSession
    modalKillConfirm
)

// Added to Model struct:
modal         modalState
agentIdx      int               // current agent picker index
dirInput      textinput.Model   // directory field
argsInput     textinput.Model   // arguments field
focusedField  int               // 0=agent, 1=directory, 2=arguments
detectedCLIs  []pty.DetectedCLI // cached
killTarget    *daemon.SessionInfo
killFocusYes  bool
editing       bool
editInput     textinput.Model
editOriginal  string
editSessionID string
```

**textinput.Model creation pattern:**

```go
// Source: [VERIFIED: bubbles v2.1.0 textinput/textinput.go New() line 157]
ti := textinput.New()
ti.Placeholder = "/Users/ken/dev/project"
ti.SetWidth(50) // modal inner width minus label width
ti.Focus()      // returns tea.Cmd for cursor blink
```

**Focus cycling (Tab/Shift+Tab):**

```go
func (m Model) cycleFocus(forward bool) (Model, tea.Cmd) {
    // Blur current field
    switch m.focusedField {
    case 1:
        m.dirInput.Blur()
    case 2:
        m.argsInput.Blur()
    }

    if forward {
        m.focusedField = (m.focusedField + 1) % 3
    } else {
        m.focusedField = (m.focusedField + 2) % 3 // -1 mod 3
    }

    // Focus new field
    var cmd tea.Cmd
    switch m.focusedField {
    case 1:
        cmd = m.dirInput.Focus()
    case 2:
        cmd = m.argsInput.Focus()
    }
    return m, cmd
}
```

### Pattern 3: Priority-Based Key Dispatch

**What:** Route key events through a priority chain based on UI state.
**When to use:** Any TUI with overlapping modal/editing states.

```go
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    // Priority 1: Inline rename captures all keys
    if m.editing {
        return m.handleRenameKey(msg)
    }
    // Priority 2: Kill confirmation dialog
    if m.modal == modalKillConfirm {
        return m.handleKillConfirmKey(msg)
    }
    // Priority 3: New session modal
    if m.modal == modalNewSession {
        return m.handleNewSessionKey(msg)
    }
    // Priority 4: Help overlay
    if m.showHelp {
        if key.Matches(msg, m.keys.Help) || msg.String() == "esc" {
            m.showHelp = false
            return m, nil
        }
        return m, nil
    }
    // Priority 5: Main view
    return m.handleMainKey(msg)
}
```

### Pattern 4: tea.Cmd for Async RPC (All Operations)

**What:** Wrap daemon client calls in `tea.Cmd` closures to avoid blocking Update.
**When to use:** Every daemon API call from the TUI.

```go
// cmds.go additions
func createSession(client *daemon.DaemonClient, cli, name, workDir string, args []string) tea.Cmd {
    return func() tea.Msg {
        id, err := client.CreateSession(cli, name, workDir, args, 0, 0)
        return createSessionMsg{id: id, err: err}
    }
}

func killSession(client *daemon.DaemonClient, id string) tea.Cmd {
    return func() tea.Msg {
        err := client.KillSession(id)
        return killSessionMsg{err: err}
    }
}

func renameSession(client *daemon.DaemonClient, id, name string) tea.Cmd {
    return func() tea.Msg {
        err := client.RenameSession(id, name)
        return renameSessionMsg{err: err}
    }
}
```

### Pattern 5: Toast with Auto-Dismiss

**What:** Lightweight notification system using model state + tea.Tick.
**When to use:** Success/error feedback for async operations.

The existing Phase 76 model already has `toast` and `toastExp` fields. Phase 77 enhances this with a toast kind for styling:

```go
type toastKind int
const (
    toastInfo toastKind = iota    // fg.muted (in-flight)
    toastSuccess                   // status.running color
    toastError                     // status.errored color
)

// In Model (replace simple string):
toast     string
toastKind toastKind
toastExp  time.Time
```

The toast expiry is already handled by checking `time.Now().Before(m.toastExp)` in `renderWebStatus()`. The existing 2-second tick also naturally clears expired toasts on each render. No separate dismiss timer needed -- the view check is sufficient. [VERIFIED: view.go lines 237-239]

### Anti-Patterns to Avoid

- **`tea.Suspend` for attach:** Sends SIGTSTP which backgrounds the process. Use `tea.Exec` which releases the terminal for a blocking operation. [VERIFIED: 77-UI-SPEC.md anti-pattern #1]
- **Using `os.Stdin`/`os.Stdout` in attachCmd.Run():** Must use the io.Reader/io.Writer provided by SetStdin/SetStdout. Bubble Tea provides these from its own program input/output. [VERIFIED: exec.go lines 111-113]
- **Blocking Update() with RPC calls:** All daemon calls must be wrapped in `tea.Cmd`. Blocking Update freezes the UI. [VERIFIED: Bubble Tea MVU design]
- **Calling `pty.DetectCLIs()` on every modal open:** PATH scan is O(n) exec.LookPath calls. Cache once on first modal open. [VERIFIED: 77-UI-SPEC.md anti-pattern #9]
- **Client-side directory validation:** Daemon validates on CreateSession. Client-side `os.Stat` adds latency. [VERIFIED: 77-UI-SPEC.md anti-pattern #4]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Text input with cursor movement | Custom rune buffer + cursor positioning | `bubbles/v2/textinput.Model` | Handles word-jump, clipboard paste, Unicode, overflow scrolling |
| Terminal release/restore for attach | Manual alt-screen exit + raw mode management | `tea.Exec(ExecCommand, callback)` | Bubble Tea manages renderer pause, input cancellation, and state restoration |
| WebSocket attach I/O | New attach implementation | Existing `attachSession()` from cmd_attach.go | Already handles stdinPump, wsOutputPump, detach key, status bar, MsgMeta |
| Adaptive light/dark colors | Manual TERM detection | `lipgloss.LightDark(hasDark)` + `tea.BackgroundColorMsg` | Already implemented in Phase 76 styles.go |
| Session CRUD API calls | Direct HTTP requests | `daemon.DaemonClient` methods | Already exists: CreateSession, KillSession, RenameSession |
| Agent detection | Custom PATH scanning | `pty.DetectCLIs()` | Already handles all known CLIs with LookPath |

**Key insight:** Phase 77 should NOT duplicate any attach logic from cmd_attach.go. The `attachSession()` function (lines 348-369) is already factored as a testable core that takes io.Reader/io.Writer params. The ExecCommand wrapper just provides the setup (WebSocket dial, raw mode, status bar) and calls this function.

## Common Pitfalls

### Pitfall 1: stdin is Not Always *os.File Inside tea.Exec

**What goes wrong:** `attachCmd.Run()` calls `term.MakeRaw(int(a.stdin.(*os.File).Fd()))` but Bubble Tea may provide a non-file reader.
**Why it happens:** `tea.Exec` calls `SetStdin(p.input)` where `p.input` is the program's input. In normal operation this is `os.Stdin`, but in tests it might be a `*bytes.Buffer`.
**How to avoid:** Type-assert `a.stdin.(*os.File)` with the `ok` pattern. If not a file, skip raw mode and status bar (test path). In production the Program is created without `WithInput()` so `p.input` is `os.Stdin`.
**Warning signs:** Panic on type assertion during tests.

### Pitfall 2: Keybinding Conflict on `r` Reassignment

**What goes wrong:** Phase 76 uses `r` for refresh. Phase 77 reassigns `r` to rename and `R` to refresh. If only keys.go is updated but help.go is not, the help overlay shows stale bindings.
**Why it happens:** Help content is hardcoded in `buildHelpContent()`, not derived from the KeyMap.
**How to avoid:** Update both `keys.go` AND `help.go` AND `renderHintBar()` in the same task. Update tests that assert on help content.
**Warning signs:** Help overlay shows `r: Refresh list` instead of `r: Rename session`.

### Pitfall 3: textinput Consuming Keys That Should Be Intercepted

**What goes wrong:** When a textinput field is focused in the new-session modal, pressing `Enter` gets consumed by textinput rather than triggering form submission.
**Why it happens:** textinput's `Update()` has a `default:` case that calls `insertRunesFromUserInput([]rune(msg.Text))` for unrecognized keys. However, `Enter` produces a KeyPressMsg with `Code: tea.KeyEnter` and empty `Text`, so it passes through without insertion but IS consumed (textinput returns the same model).
**How to avoid:** Intercept `Tab`, `Shift+Tab`, `Enter`, and `Esc` BEFORE passing the message to `textinput.Update()`. Only delegate non-intercepted keys. [VERIFIED: textinput.go line 647 -- default case calls insertRunesFromUserInput(msg.Text)]
**Warning signs:** Enter key does nothing in modal.

### Pitfall 4: Modal Open During Tick Refresh Causes State Corruption

**What goes wrong:** While the new-session modal is open, a 2-second tick fires and updates `m.sessions`. If the selected index changes, the kill target or rename target might point to a different session.
**Why it happens:** Tick fires unconditionally and modifies `m.sessions`.
**How to avoid:** Store the target session ID (not index) in modal/editing state. The `killTarget` field stores a `*daemon.SessionInfo` (copied, not a pointer into the slice). The `editSessionID` field stores the session ID string. After the tick updates sessions, these remain stable because they reference the ID, not the index.
**Warning signs:** Killing session A actually kills session B because the list shifted.

### Pitfall 5: lockedWriter and io.Reader Accessibility from main Package

**What goes wrong:** `attachCmd` lives in `internal/tui` but `lockedWriter`, `attachSession`, `stdinPump`, `wsOutputPump`, and `makeClientResizeFrame` live in `package main` (cmd_attach.go).
**Why it happens:** cmd_attach.go is in the main package, which cannot be imported by internal packages.
**How to avoid:** Either (a) extract the shared attach logic into an `internal/attach` package, or (b) duplicate the minimal needed functions in `internal/tui/attach.go`. Option (a) is cleaner but requires refactoring cmd_attach.go. Option (b) is faster but creates duplication. **Recommended: Option (a)** -- extract `attachSession`, `stdinPump`, `wsOutputPump`, `lockedWriter`, and `makeClientResizeFrame` into `internal/attach/attach.go`, then import from both `cmd_attach.go` and `internal/tui/attach.go`.
**Warning signs:** Import cycle or "cannot import main" compilation errors.

### Pitfall 6: Raw Mode vs Bubble Tea's Terminal State

**What goes wrong:** After `tea.Exec` calls `releaseTerminal`, the terminal is in cooked mode. `attachCmd.Run()` sets raw mode. After `Run()` returns, `RestoreTerminal` re-initializes the terminal for Bubble Tea. If `Run()` fails to restore cooked mode before returning, Bubble Tea's reinit may conflict.
**Why it happens:** `term.MakeRaw` + `defer term.Restore` must execute before `Run()` returns.
**How to avoid:** Use `defer term.Restore(fd, oldState)` immediately after `term.MakeRaw()`. The defer ensures restoration even on panic/error paths.
**Warning signs:** Terminal garbled after failed attach attempt.

## Code Examples

### Example 1: ExecCommand Callback Message

```go
// model.go -- new message type
type attachDoneMsg struct {
    err error
}

// update.go -- handler
case attachDoneMsg:
    // Refresh sessions (state may have changed during attach)
    cmds := []tea.Cmd{fetchSessions(m.client), fetchWebStatus(m.client)}
    if msg.err != nil {
        m.toast = fmt.Sprintf("Attach error: %s", msg.err)
        m.toastKind = toastError
        m.toastExp = time.Now().Add(3 * time.Second)
    }
    return m, tea.Batch(cmds...)
```

### Example 2: textinput Configuration for Modal Fields

```go
// Source: [VERIFIED: bubbles v2.1.0 textinput/textinput.go lines 156-174]
func newDirInput(cwd string, width int, hasDark bool) textinput.Model {
    ti := textinput.New()
    ti.Placeholder = cwd
    ti.SetValue(cwd)
    ti.SetWidth(width)
    ti.Prompt = ""  // no prompt -- label rendered separately
    ti.CharLimit = 256

    // Custom styles per UI-SPEC color tokens
    styles := textinput.DefaultStyles(hasDark)
    // Customize focused/blurred text and placeholder styles here
    ti.SetStyles(styles)

    return ti
}
```

### Example 3: Agent Picker Cycle (Left/Right)

```go
// No textinput -- just index cycling over DetectedCLIs
func (m Model) cycleAgent(forward bool) Model {
    if len(m.detectedCLIs) == 0 {
        return m
    }
    if forward {
        m.agentIdx = (m.agentIdx + 1) % len(m.detectedCLIs)
    } else {
        m.agentIdx = (m.agentIdx + len(m.detectedCLIs) - 1) % len(m.detectedCLIs)
    }
    return m
}
```

### Example 4: Kill Confirmation Toggle

```go
func (m Model) handleKillConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    s := msg.String()
    switch {
    case s == "y":
        // Quick-confirm Yes
        return m.executeKill()
    case s == "n", s == "esc":
        // Quick-confirm No or cancel
        m.modal = modalNone
        m.killTarget = nil
        return m, nil
    case s == "enter":
        if m.killFocusYes {
            return m.executeKill()
        }
        // No is focused -- cancel
        m.modal = modalNone
        m.killTarget = nil
        return m, nil
    case s == "left", s == "right", s == "h", s == "l", s == "tab":
        m.killFocusYes = !m.killFocusYes
        return m, nil
    }
    return m, nil
}

func (m Model) executeKill() (tea.Model, tea.Cmd) {
    id := m.killTarget.ID
    m.modal = modalNone
    m.killTarget = nil
    m.toast = "Killing session..."
    m.toastKind = toastInfo
    m.toastExp = time.Now().Add(10 * time.Second) // replaced by success/error toast
    return m, killSession(m.client, id)
}
```

### Example 5: Inline Rename Entry/Exit

```go
// Enter rename mode
case key.Matches(msg, m.keys.Rename):
    if len(m.sessions) == 0 {
        return m, nil
    }
    s := m.sessions[m.selected]
    m.editing = true
    m.editSessionID = s.ID
    m.editOriginal = s.Name
    m.editInput = textinput.New()
    m.editInput.Prompt = ""
    m.editInput.SetValue(s.Name)
    m.editInput.SetWidth(m.nameColWidth())
    m.editInput.CursorEnd()
    cmd := m.editInput.Focus()
    return m, cmd

// Handle rename keys
func (m Model) handleRenameKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    s := msg.String()
    switch {
    case s == "enter":
        name := strings.TrimSpace(m.editInput.Value())
        if name == "" {
            m.toast = "Name cannot be empty"
            m.toastKind = toastError
            m.toastExp = time.Now().Add(2 * time.Second)
            return m, nil // keep editing
        }
        m.editing = false
        if name == m.editOriginal {
            return m, nil // no change, no API call
        }
        m.toast = "Renaming..."
        m.toastKind = toastInfo
        m.toastExp = time.Now().Add(10 * time.Second)
        return m, renameSession(m.client, m.editSessionID, name)
    case s == "esc":
        m.editing = false
        return m, nil
    default:
        var cmd tea.Cmd
        m.editInput, cmd = m.editInput.Update(msg)
        return m, cmd
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `tea.Suspend` (sends SIGTSTP) | `tea.Exec` (releases terminal for blocking cmd) | bubbletea v0.24+ | ExecCommand runs in-process, no SIGTSTP, cleaner resume |
| v1 `tea.KeyMsg` with `tea.Key` struct | v2 `tea.KeyPressMsg` with `.Code` and `.String()` | bubbletea v2.0.0 | Code is rune for printable keys, named constant for special keys |
| v1 `View() string` | v2 `View() tea.View` with declarative `.AltScreen` field | bubbletea v2.0.0 | No more `tea.WithAltScreen` program option |
| Bubbles v1 `textinput.Model` | Bubbles v2 `textinput.Model` with `SetStyles()` | bubbles v2.0.0 | Import path `charm.land/bubbles/v2/textinput` |

**Deprecated/outdated:**
- `tea.Suspend`: Still exists but sends SIGTSTP. NOT suitable for running a process in the terminal. Use `tea.Exec` instead.
- `tea.WithAltScreen`: v1 program option. In v2, set `v.AltScreen = true` in `View()`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `p.input` is `os.Stdin` when Program created without `WithInput()` | Pattern 1 (tea.Exec) | If p.input is not *os.File, raw mode and status bar skip -- functional but degraded |
| A2 | `SIGWINCH` watcher goroutine from `watchResize()` will work during tea.Exec because the program is still alive | Pattern 1 | If SIGWINCH is blocked during Exec, terminal resize during attach won't propagate -- minor UX issue |
| A3 | Extracting attach logic to `internal/attach` package will not break `cmd_attach.go` callers | Pitfall 5 | If extraction introduces import cycles, must use option (b) duplication instead |

## Open Questions

1. **SIGWINCH during tea.Exec**
   - What we know: `tea.Exec` calls `releaseTerminal` which sets `ignoreSignals=1`. The `watchResize` goroutine from `cmd_attach.go` listens for SIGWINCH separately.
   - What's unclear: When the attach ExecCommand starts, does it need its own SIGWINCH watcher? The existing `watchResize()` in `cmd_attach.go` sets up its own signal channel.
   - Recommendation: The `attachCmd.Run()` should call `watchResize(ctx, conn)` to handle resize during attach, just like `cmd_attach.go` does. Bubble Tea's signal suppression is irrelevant because `watchResize` registers its own signal handler.

2. **Shared attach logic extraction**
   - What we know: `attachSession()`, `stdinPump()`, `wsOutputPump()`, `lockedWriter`, and `makeClientResizeFrame()` are in `package main`.
   - What's unclear: Whether extracting to `internal/attach` will cause import cycles with other main-package functions.
   - Recommendation: Analyze imports. These functions depend on `relay`, `statusbar`, `websocket`, and `term` -- all internal packages. No cycle risk. `parseRemoteID` and remote-specific functions stay in main.

3. **textinput virtual cursor vs real cursor**
   - What we know: `textinput.New()` defaults to `useVirtualCursor: true` with `CursorBlock` shape.
   - What's unclear: Whether virtual cursor rendering conflicts with Bubble Tea's own cursor management during modal display.
   - Recommendation: Use default virtual cursor. If rendering issues appear, switch to real cursor via `ti.SetVirtualCursor(false)`.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib, go1.26.2) |
| Config file | none -- Go's built-in test runner |
| Quick run command | `go test ./internal/tui/... -count=1 -timeout 30s` |
| Full suite command | `go test ./... -count=1 -timeout 120s` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TUI-03a | Enter key dispatches tea.Exec (returns non-nil Cmd) | unit | `go test ./internal/tui/... -run TestUpdate_AttachDispatch -count=1` | Wave 1 |
| TUI-03b | attachDoneMsg refreshes sessions, shows error toast on failure | unit | `go test ./internal/tui/... -run TestUpdate_AttachDone -count=1` | Wave 1 |
| TUI-03c | Attach skipped when session errored (toast shown) | unit | `go test ./internal/tui/... -run TestUpdate_AttachErroredSession -count=1` | Wave 1 |
| TUI-03d | ExecCommand.Run() calls attachSession (integration) | manual | Manual: launch TUI, press Enter on session, verify raw PTY, Ctrl-\ detach, TUI resumes | N/A |
| TUI-04a | `n` key opens new-session modal | unit | `go test ./internal/tui/... -run TestUpdate_NewSessionModalOpen -count=1` | Wave 2 |
| TUI-04b | Tab cycles focus through 3 fields | unit | `go test ./internal/tui/... -run TestModal_FocusCycle -count=1` | Wave 2 |
| TUI-04c | Left/Right cycles agent picker | unit | `go test ./internal/tui/... -run TestModal_AgentCycle -count=1` | Wave 2 |
| TUI-04d | Enter submits with validation (empty dir rejected) | unit | `go test ./internal/tui/... -run TestModal_SubmitValidation -count=1` | Wave 2 |
| TUI-04e | Esc cancels modal, returns to main view | unit | `go test ./internal/tui/... -run TestModal_Cancel -count=1` | Wave 2 |
| TUI-04f | createSessionMsg success updates list and shows toast | unit | `go test ./internal/tui/... -run TestUpdate_CreateSessionMsg -count=1` | Wave 2 |
| TUI-04g | Modal rendered with correct dimensions and fields | unit (view) | `go test ./internal/tui/... -run TestView_NewSessionModal -count=1` | Wave 2 |
| TUI-05a | `d` key opens kill confirmation with default No | unit | `go test ./internal/tui/... -run TestUpdate_KillConfirmOpen -count=1` | Wave 2 |
| TUI-05b | `y` quick-confirms kill | unit | `go test ./internal/tui/... -run TestKill_QuickYes -count=1` | Wave 2 |
| TUI-05c | `n` or Esc cancels kill | unit | `go test ./internal/tui/... -run TestKill_Cancel -count=1` | Wave 2 |
| TUI-05d | Left/Right toggles Yes/No focus | unit | `go test ./internal/tui/... -run TestKill_ToggleFocus -count=1` | Wave 2 |
| TUI-05e | killSessionMsg success shows toast, refreshes list | unit | `go test ./internal/tui/... -run TestUpdate_KillSessionMsg -count=1` | Wave 2 |
| TUI-05f | Kill dialog renders session name and danger styling | unit (view) | `go test ./internal/tui/... -run TestView_KillConfirmDialog -count=1` | Wave 2 |
| TUI-06a | `r` key enters inline rename with pre-filled name | unit | `go test ./internal/tui/... -run TestUpdate_RenameStart -count=1` | Wave 3 |
| TUI-06b | Navigation keys suppressed during rename | unit | `go test ./internal/tui/... -run TestRename_NavigationSuppressed -count=1` | Wave 3 |
| TUI-06c | Enter submits rename, Esc cancels | unit | `go test ./internal/tui/... -run TestRename_SubmitAndCancel -count=1` | Wave 3 |
| TUI-06d | Empty name rejected with toast | unit | `go test ./internal/tui/... -run TestRename_EmptyRejected -count=1` | Wave 3 |
| TUI-06e | Same name = no API call | unit | `go test ./internal/tui/... -run TestRename_SameNameNoOp -count=1` | Wave 3 |
| TUI-06f | renameSessionMsg refreshes list | unit | `go test ./internal/tui/... -run TestUpdate_RenameSessionMsg -count=1` | Wave 3 |
| TUI-06g | Inline edit replaces Name column in view | unit (view) | `go test ./internal/tui/... -run TestView_InlineRename -count=1` | Wave 3 |
| KEY-01 | `r` = rename (not refresh), `R` = refresh | unit | `go test ./internal/tui/... -run TestUpdate_KeyReassignment -count=1` | Wave 1 |
| KEY-02 | Updated help overlay shows correct bindings | unit | `go test ./internal/tui/... -run TestHelp_UpdatedBindings -count=1` | Wave 1 |
| KEY-03 | Updated hint bar shows all Phase 77 actions | unit (view) | `go test ./internal/tui/... -run TestView_HintBar -count=1` | Wave 1 |
| TOAST-01 | Toast kind determines color (info/success/error) | unit (view) | `go test ./internal/tui/... -run TestView_ToastKind -count=1` | Wave 2 |

### Sampling Rate

- **Per task commit:** `go test ./internal/tui/... -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1 -timeout 120s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] No new framework install needed (bubbles/v2/textinput already available via existing go.mod)
- [ ] `internal/tui/attach.go` -- NEW file for ExecCommand implementation
- [ ] `internal/tui/modal.go` -- NEW file for modal rendering
- [ ] Test files for new functionality can be added alongside implementation (same wave pattern as Phase 76)

### Manual-Only Verifications (UAT)

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Attach -> detach -> TUI resume (full flow) | TUI-03 | Requires real terminal, WebSocket, PTY process | Launch TUI, select session, press Enter, verify raw PTY, press Ctrl-\, verify TUI resumes with updated list |
| Status bar visible during attach | TUI-03 | Status bar uses DECSTBM which needs real terminal | During attach, verify bottom bar shows session name/agent/hostname |
| New session modal layout and field focus | TUI-04 | Visual rendering verification | Press `n`, verify modal centered, Tab through fields, verify cursor visibility, Left/Right on agent |
| Kill confirmation default-No safety | TUI-05 | UX safety verification | Press `d`, verify No is highlighted by default, press Enter to cancel (not kill) |
| Inline rename cursor and styling | TUI-06 | Visual rendering of textinput in row | Press `r`, verify Name column becomes editable with cursor, type new name, Enter to submit |
| Toast appearance and auto-dismiss | All | Timing-dependent rendering | Create session, verify "Creating session..." then "Session created" toast, verify auto-dismiss after 2s |

## Project Constraints (from CLAUDE.md)

- **Go:** `go fmt`, `golangci-lint`, context-aware functions
- **Testing:** `go test ./... -race`, 80%+ coverage in critical components
- **Package managers:** `go mod` for Go
- **Chesterton's Fence:** Before removing anything (like the Phase 76 `r=refresh` binding), articulate why it exists
- **Silent Fallbacks:** Let it crash rather than swallowing errors with `or {}`
- **LSP:** Prefer LSP over Grep/Read for code navigation (pyright, gopls, TypeScript)

## Sources

### Primary (HIGH confidence)
- bubbletea v2.0.5 source code: `exec.go` (ExecCommand interface, Exec flow, releaseTerminal/RestoreTerminal) [VERIFIED: read from GOMODCACHE]
- bubbletea v2.0.5 source code: `tea.go` lines 831-833 (execMsg handling), 1318-1360 (ReleaseTerminal/RestoreTerminal) [VERIFIED: read from GOMODCACHE]
- bubbles v2.1.0 source code: `textinput/textinput.go` (New, Update, View, Focus, Blur, SetWidth, SetValue, Value) [VERIFIED: read from GOMODCACHE]
- bubbles v2.1.0 source code: `textinput/styles.go` (Styles, StyleState, CursorStyle, DefaultStyles) [VERIFIED: read from GOMODCACHE]
- `internal/daemon/client.go` (CreateSession, KillSession, RenameSession signatures) [VERIFIED: codebase]
- `internal/daemon/types.go` (CreateRequest, RenameRequest, SessionInfo) [VERIFIED: codebase]
- `cmd_attach.go` (attachSession, stdinPump, wsOutputPump, lockedWriter) [VERIFIED: codebase]
- `cmd_attach_unix.go` (watchResize, SIGWINCH handler) [VERIFIED: codebase]
- `internal/pty/detect.go` (DetectCLIs, DetectedCLI struct) [VERIFIED: codebase]
- `internal/tui/*.go` (existing Phase 76 implementation -- model, update, view, keys, cmds, styles, help) [VERIFIED: codebase]
- `77-UI-SPEC.md` (locked design contract) [VERIFIED: codebase]

### Secondary (MEDIUM confidence)
- 76-RESEARCH.md (Bubble Tea v2 patterns, Lip Gloss v2 patterns) [VERIFIED via Phase 76 implementation]
- 76-PATTERNS.md (analog patterns for file structure and code conventions) [VERIFIED via Phase 76 implementation]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already in go.mod, versions verified via `go list -m`
- Architecture: HIGH -- tea.Exec flow verified by reading actual source code; attach logic verified in existing codebase
- Pitfalls: HIGH -- identified through code analysis (import cycles, key conflicts, state corruption)
- Validation: HIGH -- testing patterns established in Phase 76, reused here

**Research date:** 2026-04-15
**Valid until:** 2026-05-15 (stable -- Charm ecosystem GA, no breaking changes expected)
