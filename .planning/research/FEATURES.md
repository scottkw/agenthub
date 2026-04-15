# Feature Research

**Domain:** Multi-client WebSocket session sharing, tmux-style CLI status bar, and near-GUI-parity TUI mode for AgentHub v2.0
**Researched:** 2026-04-14
**Confidence:** HIGH for multi-client and status bar (established patterns with verified prior art); MEDIUM for TUI mode (approach depends on PTY-in-TUI integration complexity)

---

## Context: What Already Exists (v1.14 baseline)

This is a SUBSEQUENT MILESTONE. The following are already built and are NOT scope for v2.0:

- Single-client WebSocket relay with binary framing (output/resize/input frame types)
- Scrollback replay on connect (server sends buffered history on new connection)
- CLI `agenthub attach <id>` with raw PTY proxy, detach key (Ctrl-\), resize propagation, Ctrl-C passthrough, signal-safe terminal restore
- Web terminal status bar showing session name, agent type, hostname, REST-polled connection state
- CLI attach shows connection banner and detach message on stderr — NO persistent status bar
- GUI: tabbed terminals, collapsible sidebar, settings tab, remote sessions panel, 138 theme presets with live apply
- Daemon over Unix socket; GUI and CLI are both DaemonClient consumers
- Remote session discovery and WSS relay for tailnet peers (`agenthub attach hostname:id`)
- Binary framing protocol distinguishes output/resize/input frames

---

## Feature Area 1: Multi-Client Session Connections (GitHub #13)

### What Users Expect from Multi-Client Sharing

In established terminal multiplexers (tmux, GNU screen, tmate), attaching multiple clients to the same session means:
- All clients see the same live output simultaneously — the PTY stdout is broadcast to every attached connection
- Each client maintains its own scrollback position — one client scrolling back does not affect another client's view
- All clients share the same PTY input — any client can type, all see the result (collaborative mode)
- Read-only observer mode is available via a flag (`tmux attach -r`) — watches without ability to send input; useful for demos and pair-programming observation
- New clients joining mid-session receive a scrollback replay of recent history, then stream live output from that point forward

The key UX invariant: **independent scrollback per client, shared live output**. This is universally expected.

### Table Stakes (Multi-Client)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Broadcast live output to all connected clients | Core of multi-client sharing — all viewers see the same live terminal | MEDIUM | Requires hub/fan-out: session engine maintains a list of connected sinks; PTY reader goroutine writes to all sinks. Already partially implied by binary framing protocol. |
| Independent scrollback per client | tmux/screen convention — scrolling back on one client does not affect others | LOW | Scrollback position is maintained client-side by xterm.js and the CLI attach viewport; server only replays history on connect. No server-side scroll cursor needed. |
| Scrollback replay for new joiner | Users joining mid-session expect to see recent history before the live stream | LOW | Already implemented for single-client. Extension: all new client connections get the same replay buffer before live streaming begins. |
| Read-only attach mode | Observer/demo use case — watch without being able to type | LOW | `agenthub attach --readonly <id>` flag. Server closes the input direction; client still receives output. No new protocol frames needed — just reject input frames from read-only connections. |
| Connection count visible in GUI | Users want to know how many clients are viewing a session | LOW | Status bar or session list shows e.g. "2 viewers". Daemon tracks connected client count per session. Already available via a counter in the session engine. |

### Differentiators (Multi-Client)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Per-client named identity in connection list | Pair-programming or demo use: see "MacBook Pro — CLI", "iPad — web" as watchers | MEDIUM | Client sends optional name at handshake (e.g. `?client=macbook`). Session engine tracks client metadata. Daemon API exposes per-session client list. GUI can show watchers list in session detail. |
| Input locking (host takes exclusive control) | Host can lock input to prevent observers from typing even in non-read-only mode | HIGH | Complex state machine; likely a v2.x feature. Defer. |

### Anti-Features (Multi-Client)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Synchronized scrollback across all clients | "Everyone sees the same scroll position" | Violates the core contract — independent scrollback is what users expect from terminal sharing; forces one user's navigation onto others | Keep scrollback independent (standard behavior) |
| PTY resize negotiated across all clients | "Everyone sees the same terminal size" | Different client terminal sizes create conflict — tmux resolves this by using the smallest attached client's dimensions, causing unwanted shrinking for larger-screen clients | Let the owning PTY maintain its size; web clients respect their own viewport; document the limitation |

---

## Feature Area 2: tmux-Style CLI Attach Status Bar (GitHub #8)

### What Users Expect from a tmux-Style Status Bar

tmux's status bar is the canonical reference. Users familiar with tmux expect:
- A persistent one-line bar fixed at the bottom of the terminal (never scrolls away)
- Left side: session name in brackets, e.g. `[my-session]`
- Right side: hostname, current time/date, agent type, elapsed time since attach
- Center or left: keyboard hint for the primary action (detach key)
- Visual separator between the status bar and terminal content (background color difference)
- The bar does NOT flicker or disappear when terminal output scrolls
- The bar updates on a timer (typically every 1-15 seconds for time display)
- The bar is drawn using terminal control sequences (ANSI) directly to the terminal

The "persistent overlay" pattern is well-established: reserve the last terminal row, use ANSI cursor positioning to redraw it in-place, and restore the cursor to the content area. GNU screen's `hardstatus` and tmux's `status` both follow this model.

### Table Stakes (CLI Status Bar)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Persistent bottom bar that survives terminal scrolling | The core feature — a status bar that doesn't scroll away | MEDIUM | Implemented by: (1) reserving the last row via terminal manipulation, OR (2) using terminal alternate screen mode with a reserved row. The simpler approach for a pass-through PTY: use ANSI escape sequences to reposition cursor and redraw the last line on a goroutine tick. Go's `golang.org/x/term` or direct ANSI writes to stderr work here. |
| Session name, agent type, hostname | Matches existing CLI attach banner content — users already see these on connect; they expect them to persist | LOW | All three values already available in the attach command's session metadata from the daemon. |
| Detach key hint | `[Ctrl-\] detach` is the minimum users need to see — prevents confusion about how to exit | LOW | Static string, rendered in the status bar right-hand section. |
| Elapsed session time | How long the session has been running (not time since attach — time since session was created) | LOW | Session creation timestamp already in session metadata. Format: `1h23m` or `42m`. Refreshed by the status bar ticker goroutine. |
| Status bar updates without corrupting terminal output | The bar refresh must not interleave with PTY output and produce garbled display | HIGH | This is the hard part. The status bar goroutine must synchronize with the PTY output relay. Two approaches: (a) alternate screen trick — not compatible with all agents, (b) use ANSI save/restore cursor + redraw only when safe. The typical pattern is to redraw on a timer from a separate goroutine using `\033[s` (save cursor), position to last row, write, `\033[u` (restore cursor). Works in practice but can produce rare visual glitches on very high-throughput output. |
| Disabled in non-interactive / non-TTY mode | If stdout is not a terminal (piped), the status bar must be suppressed | LOW | Check `term.IsTerminal(int(os.Stdin.Fd()))` before enabling the status bar goroutine. Already implied by the existing raw PTY mode guard. |
| Cleanup on detach / exit | On Ctrl-\ or normal exit, clear the status bar line and restore normal terminal state | LOW | Defer to signal handler / cleanup path already in the attach command. Clear the last row with spaces and restore cursor positioning. |

### Differentiators (CLI Status Bar)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Connection state indicator in status bar | Shows "connected" vs "reconnecting" vs "relay latency Xms" live | MEDIUM | WebSocket connection state already tracked in attach; surface it in the status bar. Particularly valuable for remote Tailscale sessions where latency matters. |
| Client count in status bar | "3 viewers" — lets the attaching user know others are watching | LOW | Daemon already tracks connected clients; add it to the session metadata polled or pushed to CLI attach. |
| Configurable status bar position (top vs bottom) | Some users prefer top (like some tmux configs) | LOW | Simple positional parameter — use bottom as default; add `--status-top` flag for preference. |

### Anti-Features (CLI Status Bar)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Full tmux-style window list in status bar | "Show all session names like tmux does" | AgentHub sessions are not windows in a multiplexer; showing all sessions makes the bar complex and wastes space | Show only the current session; session list is in the GUI/TUI |
| Rich color themes for status bar | "Make it look like Starship prompt" | Adds configuration complexity; the status bar is functional, not decorative in this context | Ship one well-chosen color scheme (distinct from terminal background — a dark gray or colored band works); add configurable colors only if user demand is confirmed |
| Mouse support in status bar | "Click to detach or switch session" | Mouse event handling in raw PTY mode is complex; breaks workflows that rely on terminal mouse for the AI agent running inside | Keyboard-only for status bar; mouse is for the AI agent |

---

## Feature Area 3: Near-GUI Parity TUI Mode (GitHub #7)

### What Users Expect from "Near-GUI Parity TUI"

Users who want a TUI alternative to the desktop GUI are typically:
- Working over SSH and cannot run the desktop app on the remote machine
- Preferring to stay in the terminal even when at a local machine
- Using headless servers where a GUI is unavailable

"Near-GUI parity" means the TUI covers the same functional surface as the GUI: creating sessions, listing them, attaching to them, killing them, renaming them, and accessing settings like web server status and theme. It does NOT mean pixel-perfect visual reproduction.

Reference TUIs with similar scope (lazydocker, k9s, lazygit) share common UX patterns:
- **Two-panel layout**: left panel is a scrollable list of items; right panel shows detail/output for the selected item
- **Keyboard-first navigation**: arrow keys or j/k to move list selection, Enter to attach/drill in, q or Esc to go back, ? for help overlay
- **Persistent bottom status bar** or top bar showing current mode, context, and key hints
- **Modal dialogs** for create/rename/confirm-kill operations (pop up, capture input, dismiss)
- **Live refresh**: list updates automatically (polling or event-driven) to reflect session state changes
- **Help overlay**: pressing ? shows all keybindings for the current view

### Table Stakes (TUI Mode)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Session list panel | Core navigation: see all sessions, their agent, status (running/waiting/idle/errored), hostname | LOW | Maps directly to `agenthub list` output. Reuses daemon IPC. Rendered as a scrollable list with status indicators. |
| Attach to session from list | Primary action: select a session and press Enter to attach | MEDIUM | Bubbletea suspends its own rendering, hands the terminal over to the raw PTY proxy (same code as `agenthub attach`), then resumes TUI when user detaches. This is the Bubbletea `tea.ExecProcess` or equivalent suspend pattern. |
| Detach from session and return to TUI list | Round-trip: attach → work → detach → back to list | MEDIUM | Ctrl-\ to detach must return the user to the TUI list, not drop them to the shell. Requires the attach code to return cleanly to the TUI event loop. |
| Create new session | Users need to be able to launch new AI agent sessions without leaving the TUI | MEDIUM | Modal form: agent picker (list), working directory (text input with path completion), optional extra args. On confirm, calls daemon CreateSession IPC. |
| Kill session from list | Remove a session — with confirmation | LOW | Confirmation modal ("Kill session 'my-session'? [y/N]"). Calls daemon KillSession IPC. |
| Rename session | Tab renaming expected by GUI users | LOW | Inline edit or modal text input. Calls daemon RenameSession IPC. |
| Session status indicators | Replicate GUI running/waiting/idle/errored dots with color in the list | LOW | Map existing status values to colored characters or lipgloss-styled text. |
| Web server status in TUI | Know whether the web server is running and what URL it's serving | LOW | Footer or status panel showing web server URL (Tailscale or local). |
| Remote sessions panel | List sessions on tailnet peers (same as GUI remote sessions panel) | MEDIUM | Calls same tailnet peer probing code used by GUI. Displayed as a second list or grouped section in the session list. |
| Help overlay | Standard TUI convention — ? shows keybindings | LOW | Static screen or scrollable modal showing all keybindings. |
| Keyboard shortcuts matching CLI conventions | q/Esc to quit/back, j/k or arrows to navigate, Enter to select, ? for help | LOW | Standard TUI conventions — users who use lazygit, k9s will find them familiar. |

### Differentiators (TUI Mode)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Embedded terminal viewport for session preview | See a live preview of session output in the right panel without full attach | HIGH | Requires embedding a terminal emulator (bubbleterm or equivalent) inside the Bubbletea viewport. This is the hardest feature in the TUI scope. Bubbleterm (`taigrr/bubbleterm`) provides PTY-in-bubbletea support but is early-stage. Full xterm.js rendering is not available in a terminal environment. ANSI pass-through via a viewport component is feasible but may produce rendering artifacts. Mark as a stretch goal. |
| Theme selection in TUI settings panel | Change the theme without opening the desktop GUI | MEDIUM | TUI settings view mirrors GUI settings: show theme list, select to preview (in a color swatch), confirm to save. Writes to same config store as GUI. |
| Start/stop web serving per session from TUI | Toggle web serving without the desktop GUI | LOW | Calls existing daemon web start/stop IPC. List action or sidebar toggle. |
| QR code display in TUI | Show QR code for a session URL in the terminal | LOW | ASCII QR code (text/block character rendering) rather than an image. Several Go libraries support text-mode QR output. |

### Anti-Features (TUI Mode)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Full xterm.js rendering inside TUI | "True terminal inside a terminal" | xterm.js requires a browser/WebView canvas — it cannot render in a terminal environment. Attempting to embed it creates double-encoding of ANSI sequences and garbled output. | Raw PTY attach (suspend TUI, hand off to raw mode, resume on detach) — this is the correct pattern and what tmux/screen do |
| Mouse-driven TUI navigation | "Click to select sessions" | Mouse event handling in a TUI requires raw input mode changes that interfere with the AI agent's own mouse usage during attach | Keyboard-only navigation for the TUI shell; mouse support inside the session is handled by the attached agent |
| Split-pane tiling in TUI | "Show two sessions side by side" | Already out of scope per PROJECT.md for GUI; doubly complex for TUI where terminal dimensions are more constrained | Single-session attach with quick switch via detach → reselect in list |
| TUI replaces the daemon management panel | "Full daemon management from TUI" | Install/uninstall/start/stop of the system service is rare; adding to TUI adds surface area without proportional value | CLI subcommands (`agenthub daemon install/start/stop`) already handle this |

---

## Feature Dependencies

```
Multi-Client Broadcast
    depends on --> existing binary framing relay protocol
    depends on --> existing scrollback buffer (already per-session ring buffer)
    requires --> session engine: replace single-sink with multi-sink fan-out
    requires --> session engine: per-connection write tracking (independent)
    enables --> connection count display (GUI, CLI status bar, TUI)
    enables --> read-only attach mode (input suppression per connection)

Read-Only Attach Mode
    depends on --> Multi-Client Broadcast (needs multi-sink to distinguish RO connections)
    requires --> new connection metadata field: readonly bool
    requires --> server rejects input frames from read-only connections

CLI Status Bar
    depends on --> existing CLI attach command (wraps it)
    depends on --> existing session metadata (name, agent, hostname, created_at)
    requires --> ANSI cursor-control status bar goroutine
    requires --> timer/ticker for elapsed time refresh
    enhances --> multi-client (can show viewer count in bar)

Connection Count in Daemon API
    depends on --> Multi-Client Broadcast (count only meaningful when multi-client exists)
    enables --> CLI status bar viewer count display
    enables --> GUI session list viewer count badge
    enables --> TUI session list viewer count

TUI Mode
    depends on --> existing daemon IPC (all CRUD operations already available)
    depends on --> existing CLI attach command (suspend TUI, delegate to raw PTY attach)
    depends on --> existing tailnet peer probing (for remote sessions panel)
    requires --> bubbletea + lipgloss + bubbles as new dependencies
    requires --> session list model (polls daemon list endpoint)
    requires --> modal components (create, rename, kill-confirm)
    enhances --> CLI status bar (TUI returns to list after detach; status bar shows during attach)

TUI Embedded Preview (stretch)
    depends on --> TUI Mode (must exist first)
    depends on --> bubbleterm or equivalent PTY-in-TUI library
    conflicts with --> raw PTY attach (cannot be active simultaneously)
    risk --> HIGH complexity, library maturity unknown
```

### Dependency Notes

- **Multi-client is the foundation**: both the CLI status bar (viewer count) and TUI mode (session list shows viewers) benefit from multi-client. Implement multi-client before exposing viewer count in those surfaces.
- **CLI status bar is independent of TUI**: the status bar is a feature of the existing `attach` command and does not require TUI mode. Ship it standalone if TUI mode is delayed.
- **TUI mode delegates to existing attach**: the TUI does not reimplement the PTY proxy. It suspends itself, hands control to the raw attach code, then resumes. This is the correct pattern — the bubbletea `ExecProcess` API is designed for exactly this.
- **Embedded preview is NOT required for TUI MVP**: listing sessions, attaching, creating, and killing sessions are the MVP. The embedded preview is a stretch goal that can follow.

---

## MVP Definition for v2.0

### Launch With (v2.0 core)

Multi-client:
- [ ] Multi-sink fan-out broadcast to all connected WebSocket clients — core contract
- [ ] Independent scrollback per client (no server-side change needed; replay-on-connect already works)
- [ ] Connection count tracked in daemon and exposed via session metadata API
- [ ] Read-only attach mode via `--readonly` flag on CLI attach

CLI status bar:
- [ ] Persistent ANSI bottom bar with session name, agent, hostname, detach hint, elapsed time
- [ ] Status bar goroutine that refreshes time every 10 seconds without corrupting terminal output
- [ ] Status bar suppressed when stdout is not a TTY
- [ ] Clean teardown on detach/exit (clear bar line, restore cursor)

TUI mode:
- [ ] `agenthub tui` command launches Bubbletea TUI
- [ ] Session list panel with status indicators (running/waiting/idle/errored), agent, hostname, viewer count
- [ ] Select session to attach (suspends TUI, raw PTY attach, resumes on detach)
- [ ] Create new session modal (agent picker, working directory, extra args)
- [ ] Kill session with confirmation
- [ ] Rename session
- [ ] Web server status in footer/status panel
- [ ] Remote sessions panel (tailnet peer probing)
- [ ] Help overlay (? key)

### Add After Validation (v2.x)

- [ ] Connection count display in GUI session list badge — adds visibility without blocking v2.0
- [ ] Per-client identity tracking (client name in connection list) — useful for pair programming
- [ ] Configurable status bar position (top vs bottom) — user preference
- [ ] TUI theme selection panel — mirrors GUI settings
- [ ] ASCII QR code display in TUI — low complexity, nice to have
- [ ] Embedded terminal preview in TUI — stretch goal; assess bubbleterm maturity after MVP ships

### Future Consideration (v2.x+)

- [ ] Input locking (host-exclusive control in multi-client) — complex state machine
- [ ] Configurable status bar color theme — low priority
- [ ] TUI start/stop web serving per session — add when TUI settings panel is built

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Multi-client fan-out broadcast | HIGH | MEDIUM | P1 |
| Connection count in daemon API | MEDIUM | LOW | P1 |
| Read-only attach mode | MEDIUM | LOW | P1 |
| CLI status bar (name/agent/host/time) | HIGH | MEDIUM | P1 |
| Status bar corruption-free refresh | HIGH | HIGH | P1 (required to make bar viable) |
| TUI session list + status | HIGH | MEDIUM | P1 |
| TUI attach (suspend/resume pattern) | HIGH | MEDIUM | P1 |
| TUI create/kill/rename | MEDIUM | MEDIUM | P1 |
| TUI remote sessions panel | MEDIUM | MEDIUM | P1 |
| TUI web server status display | LOW | LOW | P1 |
| TUI help overlay | MEDIUM | LOW | P1 |
| GUI viewer count badge | MEDIUM | LOW | P2 |
| Per-client identity tracking | LOW | MEDIUM | P2 |
| TUI theme selection | LOW | MEDIUM | P2 |
| ASCII QR in TUI | LOW | LOW | P2 |
| Embedded terminal preview in TUI | MEDIUM | HIGH | P3 |
| Input locking (multi-client) | LOW | HIGH | P3 |

**Priority key:**
- P1: Required for v2.0 milestone
- P2: Ship when P1 complete, before or alongside v2.1
- P3: Future milestone, do not block on

---

## Complexity Notes by Feature Area

### Multi-Client (Medium overall)

The underlying architecture change is replacing the single-sink write path in the session engine with a concurrent fan-out to a slice of registered sinks. This is a well-understood pattern. The hard part is concurrent write safety: each sink write must be non-blocking (if a slow client blocks the goroutine, fast clients are starved) so a goroutine-per-client with a buffered channel is preferred over a synchronous loop. The existing ring buffer for scrollback replay already provides the "replay on connect" behavior — the only change is supporting more than one live subscriber.

### CLI Status Bar (Medium overall; one hard sub-problem)

The mechanical implementation (ANSI positioning, goroutine tick) is straightforward. The hard sub-problem is avoiding visual corruption when PTY output and status bar redraw interleave. The standard approach is:
1. Write to stderr (status bar) while PTY output goes to stdout — this relies on the terminal merging two streams correctly, which is generally true in practice
2. Use ANSI save cursor (`\033[s`) before redrawing the bar, restore cursor (`\033[u`) after — this keeps the content cursor in the right place
3. Accept that very high-throughput output (e.g. `cat /dev/urandom | xxd`) may momentarily displace the bar — it self-corrects on the next tick

This is exactly what tools like `shox` and `bottombar` do, and it's acceptable in practice.

### TUI Mode (Medium-High overall; one hard sub-problem)

The majority of TUI features map directly onto existing daemon IPC calls — session list, create, kill, rename, web server status, remote sessions. These are straightforward Bubbletea models calling existing Go functions.

The hard sub-problem is the attach suspend/resume cycle. Bubbletea provides `tea.ExecProcess` (or `Program.Suspend()` + restore in v2) to hand control to a subprocess and resume after it exits. Using this to call the existing raw PTY attach code (not a subprocess — it's in-process) requires careful terminal state management: restore normal mode before entering raw PTY mode, and re-enter TUI mode cleanly when detaching. This is achievable but requires testing across macOS, Linux, and Windows (Windows ConPTY adds complexity).

The embedded terminal preview is the hardest sub-problem and is explicitly deferred to a stretch goal.

---

## Sources

- tmux status bar structure and format strings: https://tao-of-tmux.readthedocs.io/en/latest/manuscript/09-status-bar.html
- tmux multi-client session sharing and read-only mode: https://hamvocke.com/blog/remote-pair-programming-with-tmux/
- wemux multi-user tmux: https://github.com/zolrath/wemux
- Bubbletea framework: https://github.com/charmbracelet/bubbletea
- bubbleterm PTY-in-TUI: https://pkg.go.dev/github.com/taigrr/bubbleterm
- lazydocker TUI patterns: https://github.com/jesseduffield/lazydocker
- lazytui component-based TUI: https://github.com/DokaDev/lazytui
- shox terminal status bar: https://github.com/liamg/shox
- CLI UX best practices: https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays
- WebSocket broadcast architecture: https://websockets.readthedocs.io/en/stable/topics/broadcast.html
- AgentHub PROJECT.md read directly (HIGH confidence)

---

*Feature research for: AgentHub v2.0 — Multi-Client Sessions, CLI Status Bar, TUI Mode*
*Researched: 2026-04-14*
