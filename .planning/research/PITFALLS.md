# Pitfalls Research

**Domain:** Multi-client terminal sharing, CLI status bar, TUI mode — added to existing Go/Wails daemon app
**Researched:** 2026-04-14
**Confidence:** HIGH (architecture verified against actual codebase)

---

## Critical Pitfalls

### Pitfall 1: Subscribe-Before-Snapshot Race (Scrollback Divergence)

**What goes wrong:**
A late-joining client takes a scrollback snapshot before subscribing to live frames. Frames produced between snapshot time and subscribe time are silently dropped. The client sees a frozen partial history, then live output jumps forward — creating an apparent "gap" or duplicate depending on implementation.

**Why it happens:**
The natural implementation order is: (1) take snapshot, (2) send snapshot, (3) subscribe. But between step 1 and step 3, the PTY drain goroutine continues writing. Any frames produced during that window never arrive for the new client.

**How to avoid:**
Subscribe first, snapshot second — the pattern already implemented in `relay/server.go`. Never invert this order. The subscribe channel is buffered (256 frames) to absorb the window; the snapshot send must complete before the write pump starts draining the channel. If the snapshot is split into multiple WebSocket writes, call Subscribe once, then stream all snapshot chunks before entering the write pump loop.

**Warning signs:**
- Client sees terminal state that "skips" lines on connection
- Race detector reports read/write conflict in hub.broadcast and ScrollbackSnapshot
- Integration tests pass serially but fail under `-count=100`

**Phase to address:** Multi-client WebSocket fan-out phase (Phase 1 of milestone)

---

### Pitfall 2: Multiple Clients Sending Resize — PTY Gets Hammered with Conflicting Dimensions

**What goes wrong:**
Each attached client sends `MsgResize2` frames when its terminal window changes. The PTY accepts the last `TIOCSWINSZ` call, so whichever client resizes most recently wins. A second client with a smaller window then triggers a resize that wraps lines in the first client's view. All clients end up with corrupted visual state because the PTY renders to the "wrong" dimensions for most of them.

**Why it happens:**
A PTY has exactly one window size (`struct winsize`). With N clients all issuing resize events, they overwrite each other non-deterministically. The PTY emits `SIGWINCH` to the child process on each change; Claude Code and other TUI agents redraw immediately — producing interleaved redraws visible to other clients.

**How to avoid:**
Implement a resize arbitration policy before wiring multi-client input. Options ranked by correctness:
1. **Minimum-of-attached policy** (tmux default): track all client sizes, resize PTY to `min(cols)×min(rows)`. Re-evaluate on each client connect/resize/disconnect.
2. **Primary-client policy**: designate one client (first attacher or explicit `--primary` flag) as the sole resize authority; other clients are view-only.
3. **Largest-fits policy**: use `max(cols)×max(rows)` and let smaller clients clip. Simplest to implement but breaks scroll position on small clients.

The Hub already has a single `resizeFn` — extend it to accept the subscriber pointer alongside the dimensions so the manager can track per-client sizes.

**Warning signs:**
- Terminal output wraps at unexpected column widths when two CLI clients are attached simultaneously
- Claude Code or other agents emit rapid redraws under multi-attach test
- `SIGWINCH` flood visible in strace/dtrace on the daemon process

**Phase to address:** Multi-client resize arbitration sub-task within Phase 1

---

### Pitfall 3: Concurrent PTY stdin Writes from Multiple Input Clients

**What goes wrong:**
`hub.WriteInput` passes bytes directly to the PTY's `writer io.Writer`. With N clients all sending `MsgInput` frames concurrently, multiple goroutines call `writer.Write` simultaneously. The underlying OS PTY device does not guarantee atomicity for writes larger than `PIPE_BUF` (512 bytes on macOS, 4096 on Linux). Result: interleaved keystrokes — the AI agent receives garbled input and its state machine corrupts.

**Why it happens:**
The current single-client assumption means the write pump is the only writer. Multi-client breaks that assumption. `hub.WriteInput` has no mutex; `io.Writer` is not documented as safe for concurrent use.

**How to avoid:**
Two approaches:
1. **Serialize via channel**: Replace `hub.WriteInput` with a channel-based input queue. A single goroutine drains the channel and writes to PTY stdin. Zero mutex contention; natural backpressure.
2. **Mutex guard**: Add a `sync.Mutex` to Hub protecting `writer.Write` calls. Simpler but blocks all input senders while one write is in progress.

For this codebase, the channel approach fits better — it pairs naturally with the existing broadcast channel pattern.

Additionally: decide whether all clients can send input (collaborative mode) or only a designated primary can (view-only for secondary clients). Collaborative mode requires the serialization above; view-only mode gates `MsgInput` handling at the server level.

**Warning signs:**
- Agent receives unexpected characters or corrupted command sequences under multi-client test
- Race detector flags concurrent write on the PTY file descriptor
- Daemon logs show `write /dev/ptmx: interrupted system call` errors

**Phase to address:** Multi-client input arbitration, Phase 1 of milestone

---

### Pitfall 4: Status Bar Scroll Region Not Set — Bottom Line Gets Overwritten by Normal Output

**What goes wrong:**
A raw-mode status bar draws itself at the bottom of the terminal using cursor positioning (`\033[<rows>;1H`). PTY output arriving at `stdout` then scrolls the screen, pushing the status bar up or overwriting it. The status bar "floats" rather than staying pinned to the bottom.

**Why it happens:**
ANSI terminals scroll from the top of the screen to the bottom by default. Without a scroll region (`DECSTBM`, `\033[1;<rows-1>r`), all output including PTY passthrough scrolls over the entire screen height, including the last row reserved for the status bar.

**How to avoid:**
On status bar activation:
1. Query terminal height via `term.GetSize`.
2. Emit `\033[1;<rows-1>r` to restrict the scroll region to all rows except the last.
3. Draw the status bar at `\033[<rows>;1H` (absolute, outside scroll region).
4. On every SIGWINCH: re-query size, emit new `DECSTBM`, redraw bar.
5. On detach/exit: emit `\033[r` (reset scroll region) + `\033[<rows>;1H\033[2K` (clear the status line) before restoring terminal state.

Critical: the scroll region must be set before raw mode output begins passing through, and must be reset before `term.Restore` is called. Failing to reset leaves the host terminal's scroll region permanently broken for the rest of the session.

**Warning signs:**
- Status bar "jumps" up when the AI agent produces multi-line output
- After detach, the user's shell prompt appears only in the top portion of the terminal
- `echo test` in the shell after detach scrolls only part of the screen

**Phase to address:** CLI status bar implementation, Phase 2 of milestone

---

### Pitfall 5: SIGWINCH Received During Status Bar Redraw — Torn Render

**What goes wrong:**
SIGWINCH arrives while the status bar goroutine is mid-write (cursor move + content + clear-to-eol sequence). The handler immediately tries to redraw at a new size. Two partial writes race on `os.Stdout` — the terminal receives interleaved escape sequences and displays garbage or a partially-moved cursor.

**Why it happens:**
Go signal handlers run in their own goroutines, not on the writing goroutine. If both the normal status update ticker and the SIGWINCH handler write to stdout concurrently, there is no mutual exclusion.

**How to avoid:**
Route all status bar writes through a single goroutine (the "renderer goroutine"). SIGWINCH sends a message to that goroutine via a non-blocking channel; the ticker sends another. The renderer owns stdout writes exclusively. This is the same pattern as Bubble Tea's event loop — one writer for the terminal surface.

Do not use `sync.Mutex` around individual `os.Stdout.Write` calls — the protection boundary must be the complete render sequence (cursor-save → move → write → cursor-restore), not individual writes.

**Warning signs:**
- Status bar shows partial escape sequences (visible `^[` or `[m` in bar text)
- Terminal cursor ends up at a random position after resize
- Stress test with rapid terminal resizes produces visible artifacts

**Phase to address:** CLI status bar implementation, Phase 2 of milestone

---

### Pitfall 6: Raw Mode Not Restored on Panic or Signal Exit — Terminal Left Broken

**What goes wrong:**
`term.MakeRaw` changes the terminal's termios settings. If the process exits via `panic`, unhandled `SIGTERM`, or `os.Exit` called in a library, `defer term.Restore(...)` does not run. The user's terminal is left in raw mode: no echo, no line editing, Ctrl-C doesn't generate SIGINT. The session appears "dead" and requires typing `reset` blindly.

**Why it happens:**
Go `defer` does not execute on `os.Exit` or unhandled signal termination (which calls `runtime.throw`). Panic without `recover` does call deferred functions but some TUI frameworks have buggy panic handlers that skip terminal restore (documented in ratatui/ratatui#1005).

**How to avoid:**
1. Install a `signal.Notify` handler for `SIGTERM`/`SIGINT` that explicitly calls `term.Restore` before `os.Exit`.
2. Wrap the attach entry point in a `recover` block that restores terminal state before re-panicking.
3. For the TUI mode (Bubble Tea), verify that `p.Run()` always restores terminal on `ctrl+c` and on panic — check the bubbletea release notes for the running version.
4. If adding alt-screen (`\033[?1049h`), pair every enable with a deferred disable (`\033[?1049l`).

The current `cmd_attach.go` uses `defer term.Restore(...)` which handles normal exit and explicit detach. Extend the same pattern to the status bar and TUI paths.

**Warning signs:**
- `stty -a` shows `icanon` or `echo` disabled after process exit
- User reports terminal "frozen" after kill or crash
- CI test that kills the attach process mid-run leaves tty in bad state

**Phase to address:** CLI status bar Phase 2; TUI mode Phase 3 of milestone

---

### Pitfall 7: Slow WebSocket Client Blocks PTY Drain Goroutine (Head-of-Line Blocking)

**What goes wrong:**
The broadcast loop in `hub.go` holds `mu.Lock()` while iterating subscribers. If a slow client's channel is full and `CloseSlow` is spawned asynchronously, the lock is still held for the entire iteration. A very slow client can cause brief lock contention that stalls the drain goroutine, causing the PTY read buffer to fill up and back-pressure into the PTY driver.

**Why it happens:**
The current `broadcast` function is correct for normal cases — it uses non-blocking channel send and spawns `CloseSlow` in a goroutine. However the mutex is held across the full iteration. With many subscribers, this window grows linearly.

**How to avoid:**
This is acceptable at the expected scale (less than 10 simultaneous clients per session). Document the limit. If more than 10 simultaneous clients per session becomes a requirement, switch to a lock-free subscriber list using `sync.Map` or copy-on-write slice under a RWMutex, releasing the lock before the channel send.

For the current milestone (multi-client support), the existing pattern is sufficient. Add a per-session subscriber count limit (e.g., 8 clients) to bound the iteration cost.

**Warning signs:**
- PTY output visible latency exceeding 100ms during stress test with 5+ clients
- Mutex contention in `go tool pprof` mutex profile
- `CloseSlow` goroutines accumulating without completing

**Phase to address:** Multi-client fan-out, Phase 1 of milestone (bound subscriber count as guard)

---

### Pitfall 8: Bubble Tea Fights the PTY Passthrough — Double Raw Mode, Alt Screen Conflict

**What goes wrong:**
The TUI mode runs Bubble Tea which enters raw mode and optionally alt-screen. Inside the TUI, a terminal panel streams PTY output from a daemon session. The PTY output contains its own ANSI escape sequences (including cursor moves, color codes, alt-screen toggles from Claude Code). Bubble Tea's renderer tries to own the terminal surface while the raw PTY output also writes to it. Result: display corruption, cursor teleportation, nested alt-screen switches that break the outer TUI.

**Why it happens:**
Bubble Tea's renderer uses `\033[?25l` (hide cursor), absolute cursor positioning, and full-screen repaints. PTY output from Claude Code contains its own `\033[?1049h` (enter alt screen) and cursor positioning. When both write to the same fd, they fight for control.

**How to avoid:**
Two viable approaches:
1. **Virtual terminal emulation** (preferred): Use a VT100/xterm parser (e.g., `github.com/charmbracelet/x/ansi`) to decode the PTY stream into a virtual screen buffer. Render the virtual screen contents as a Bubble Tea viewport component. PTY escape sequences never reach the real terminal — they are interpreted into a cell grid that Bubble Tea renders safely.
2. **Separate PTY passthrough pane**: Suspend Bubble Tea's renderer while a session is "focused", pass raw PTY output directly to the terminal, and restore the TUI when the user switches tabs. This is effectively `tmux detach` semantics and is more complex to implement cleanly.

Do NOT write raw PTY bytes directly into a Bubble Tea `viewport.Model` — the viewport does not parse ANSI sequences, and escape codes will display as literal characters.

**Warning signs:**
- TUI terminal pane shows `\033[1;1H` or `\033[2J` as literal text
- After switching sessions in TUI mode, cursor is at wrong position
- Claude Code's TUI (alternate screen) causes the outer TUI to blank out

**Phase to address:** TUI mode, Phase 3 of milestone — needs explicit design decision before implementation

---

### Pitfall 9: Daemon Client HTTP Polling in Bubble Tea — Blocking Commands Leak on Exit

**What goes wrong:**
Bubble Tea commands (`tea.Cmd`) run as goroutines and send results back via messages. If a command makes a blocking HTTP call to the daemon (Unix socket, short timeout) and the user quits the TUI while the call is in-flight, the goroutine blocks until the HTTP client's context times out. Multiple abandoned goroutines accumulate during rapid navigation, slowing shutdown or preventing clean exit.

**Why it happens:**
Bubble Tea programs manage `tea.Cmd` goroutines internally but have no cancellation mechanism for in-flight commands when `p.Quit()` is called. If the command doesn't respect context cancellation, it keeps running.

**How to avoid:**
1. Always pass a `context.Context` derived from the program's lifecycle to daemon HTTP calls. Store the cancel function in the model and call it in `Update` when `tea.QuitMsg` is received.
2. Use short timeouts on daemon polling commands (1–2s) — the daemon is local, so slow responses indicate a real problem rather than network latency.
3. For streaming PTY output, use a `tea.Cmd` that blocks on a channel (fed by a goroutine reading from the WebSocket), not on direct WebSocket reads — this allows clean cancellation by closing the channel.

**Warning signs:**
- TUI takes more than 2s to exit after Ctrl-C
- `goroutine` dump during exit shows multiple HTTP transport goroutines blocked
- Unit tests hang when `p.Run()` is called in a test and the test times out before the program exits

**Phase to address:** TUI mode, Phase 3 of milestone

---

### Pitfall 10: Hub.WriteInput Races with Hub.Shutdown — Write to Closed PTY

**What goes wrong:**
When a session ends (PTY exits, `hub.Shutdown()` called), the WebSocket server goroutine for a connected client may still receive input frames and call `hub.WriteInput`. The PTY `writer` is a closed file descriptor at this point — `Write` returns `EPIPE` or `EIO`. In the current code this error is silently discarded (`_ = hub.WriteInput(payload)`). In a multi-client scenario, multiple goroutines racing on this path amplify the issue.

**Why it happens:**
PTY lifetime and WebSocket client lifetime are decoupled. The client does not know the PTY has exited until the hub's `done` channel closes. Between PTY exit and the client seeing the close signal, input frames continue to arrive.

**How to avoid:**
Check `hub.closed` before writing input. The existing `hub.done` channel provides the signal — add a `select` case in `handleSession`'s read pump that exits on `<-hub.Done()`. The existing write pump already handles this; mirror the pattern in the read pump:

```go
select {
case <-hub.Done():
    return
default:
    _ = hub.WriteInput(payload)
}
```

This is a correctness fix, not just a performance concern — silent `EPIPE` suppression hides real session lifecycle bugs.

**Warning signs:**
- Race detector reports data race on hub's `closed` field
- Daemon logs show `write /dev/ptmx: broken pipe` after session exit
- Test with session kill while client is typing shows no error but input is lost silently

**Phase to address:** Multi-client WebSocket, Phase 1 of milestone

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Accept any resize from any client without arbitration | Simpler server, no per-client state | PTY width thrashes; all clients see wrong wrap | Never for multi-client |
| Raw PTY bytes direct into Bubble Tea viewport | Zero parsing code | Escape codes displayed as literal text; display corruption | Never |
| Single global mutex for all Hub operations | Simple locking | Lock held during fan-out iteration blocks PTY drain | Acceptable for < 10 clients; document limit |
| Polling daemon over Unix socket from TUI render loop | Simple implementation | Missed frames, high latency, goroutine leak on exit | Never — use `tea.Cmd` with context |
| Skip scroll region setup for status bar | Fewer escape sequences | Status bar scrolls away on any output | Never |
| Omit SIGWINCH handler for status bar | Less code | Status bar permanently wrong size after resize | Never |
| Render PTY output as plain text in TUI viewport | No VT parser dependency | ANSI sequences appear as garble | Never for real terminal output |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| coder/websocket + multi-writer | Spawning one goroutine per subscriber that writes to the shared conn | Single write pump goroutine per connection, fed by channel — already the pattern in server.go |
| Bubble Tea + existing DaemonClient | Calling `client.ListSessions()` synchronously in `Update()` | Wrap in `tea.Cmd` returning a message; always use context with timeout |
| PTY resize + SIGWINCH + status bar | Forwarding SIGWINCH directly to both the resize watcher and the status bar | Route SIGWINCH to one handler that updates PTY size then triggers status bar redraw |
| `term.MakeRaw` + alt-screen | Entering alt-screen without checking current screen state | Always pair with explicit restore on every exit path including panic |
| Scrollback snapshot + live subscribe | Snapshot before subscribe | Subscribe first, then snapshot — already correct in relay/server.go; never regress this |
| Hub input write + hub shutdown | Writing input after PTY EOF | Check `hub.Done()` before `WriteInput`; treat EPIPE as a non-error closed signal |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Holding `mu` during channel sends in broadcast | Microsecond stalls in PTY drain | Release lock before iterating sends, or use copy-on-write subscriber slice | More than 8 simultaneous clients per session |
| Status bar redraw on every PTY output frame | CPU spike, flicker | Redraw only on tick (e.g., 500ms) and on SIGWINCH | Any session with high-bandwidth output (compilation) |
| VT parser allocates per-frame in TUI viewport | Memory pressure in long sessions | Use streaming VT parser with fixed screen buffer; reset on clear-screen | Sessions running more than 1 hour |
| Bubble Tea full re-render on every daemon poll message | UI flicker, high CPU | Use `tea.Tick` at 1s intervals, not per-output-frame polling | Any session with active output |
| Scrollback buffer raw bytes exceeding 256 KiB | Late-join replay takes more than 200ms | Keep 256 KiB default; consider compressed snapshots for long sessions | Sessions more than 30 minutes with verbose output |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Accepting input from all multi-client connections without gate | Any observer can inject keystrokes into the AI agent | Explicit input policy: collaborative (all write) or view-only (primary writes); enforce at Hub level |
| No per-connection rate limit on input frames | Malicious or buggy client can flood PTY stdin | Cap input processing at 64 KB/s per connection in the read pump |
| Status bar leaks session metadata to shell history | Session names/hostnames visible in terminal scrollback | Write status bar only to the reserved last line; never write to scrollback-visible area |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Status bar uses same colors as terminal theme | Bar invisible on some themes | Use reverse video (`\033[7m`) or a fixed neutral palette independent of theme |
| Status bar width not capped to terminal width | Bar content wraps to next line, consuming two rows | Truncate status text to `cols-1` characters; use `\033[K` to clear remainder |
| TUI mode switches to alt-screen, user loses scrollback | No way to scroll back through session history | Either stay on main screen with scroll region, or provide an explicit scrollback view inside the TUI |
| Detach from status-bar attach leaves scroll region set | User's subsequent commands only scroll in a subregion | Always emit `\033[r` on detach, even on SIGTERM/SIGKILL recovery paths |
| Multi-client attach: second client sees garbled history | Scrollback contains ANSI sequences the second client's terminal resets on connect | Strip cursor-positioning sequences from scrollback on replay; keep only printable and color/style sequences |

---

## "Looks Done But Isn't" Checklist

- [ ] **Status bar scroll region:** Verify `stty size` changes during attach cause bar to stay at bottom, not scroll up — test by resizing terminal window while attached
- [ ] **Terminal restore:** Run `stty -a` after `agenthub attach` exits via Ctrl-\\, SIGTERM, and simulated panic — all three must leave `icanon` and `echo` enabled
- [ ] **Multi-client scrollback:** Connect client A, produce 500 lines of output, connect client B — verify B sees complete history without gaps or duplicates
- [ ] **Resize arbitration:** Connect two clients with different window sizes — verify PTY columns match the chosen policy (min/primary) and neither client sees wrapped lines at wrong width
- [ ] **Input gate:** In view-only mode, verify keystrokes from secondary client do not reach PTY stdin (check with `cat` session that echoes input)
- [ ] **Hub.Done() in read pump:** Kill a session, verify secondary client disconnects within 1s and does not log EPIPE errors
- [ ] **Bubble Tea exit:** Verify `p.Run()` returns in less than 500ms after Ctrl-C in TUI mode; check for goroutine leaks with `runtime.NumGoroutine()`
- [ ] **VT parser correctness:** Run a Claude Code session inside TUI viewport for 5 minutes; verify no literal escape sequences appear as text in the rendered panel

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Subscribe-after-snapshot shipped | MEDIUM | Add integration test reproducing the race, fix ordering, verify with `-count=100 -race` |
| Terminal left in raw mode by status bar crash | LOW | Add signal handler calling `term.Restore` before status bar code ships |
| PTY resize thrash from multi-client | MEDIUM | Add per-subscriber size tracking to Hub struct; implement min-of-attached policy |
| Bubble Tea/PTY display corruption | HIGH | Requires VT parser integration or architectural pivot to passthrough mode; design upfront |
| Goroutine leak in TUI daemon polling | LOW | Add context cancellation in `tea.Cmd` wrappers; detectable with pprof before release |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Subscribe-before-snapshot race | Phase 1: Multi-client WebSocket | Integration test: 3 clients join at different times, compare output byte-for-byte |
| PTY resize arbitration | Phase 1: Multi-client WebSocket | Test: two clients with different sizes; PTY cols matches policy |
| Concurrent PTY stdin writes | Phase 1: Multi-client WebSocket | Race detector under multi-client input stress test |
| Slow client head-of-line blocking | Phase 1: Multi-client WebSocket | Subscriber count limit + pprof mutex profile |
| Hub.WriteInput after shutdown | Phase 1: Multi-client WebSocket | Race detector; session kill while client types |
| Status bar scroll region | Phase 2: CLI status bar | Manual: resize terminal during attach, verify bar stays pinned |
| SIGWINCH torn render | Phase 2: CLI status bar | Stress: rapid terminal resizes with bar active |
| Raw mode not restored | Phase 2 + Phase 3 | `stty -a` after crash/kill on all exit paths |
| Bubble Tea + PTY conflict | Phase 3: TUI mode | Design decision logged before Phase 3 starts; VT parser or passthrough chosen |
| Daemon polling goroutine leak | Phase 3: TUI mode | Goroutine count before/after `p.Run()` returns |

---

## Sources

- `internal/relay/hub.go`, `server.go`, `scrollback.go` — actual subscribe-before-snapshot pattern in codebase
- `cmd_attach.go`, `cmd_attach_unix.go` — current raw mode + SIGWINCH + defer restore pattern
- [coder/websocket race condition issues](https://github.com/coder/websocket/issues/168) — confirmed race conditions in earlier versions; concurrent writes safe in current version
- [gorilla/websocket concurrent write panic](https://github.com/gorilla/websocket/issues/913) — documents "only one writer" requirement (applies to gorilla; coder/websocket serializes internally)
- [tmux window-size smallest policy](https://man7.org/linux/man-pages/man1/tmux.1.html) — canonical multi-client resize strategy reference
- [DECSTBM scroll region VT510 reference](https://vt100.net/docs/vt510-rm/chapter4.html) — authoritative escape sequence for reserving status bar row
- [Ratatui panic handler raw mode bug](https://github.com/ratatui/ratatui/issues/1005) — documented failure of TUI framework panic handler to restore raw mode
- [Bubble Tea injecting messages from outside program loop](https://github.com/charmbracelet/bubbletea/issues/25) — goroutine/channel pattern for streaming external data
- [bubbleterm PTY integration package](https://pkg.go.dev/github.com/taigrr/bubbleterm) — VT parser approach for embedding terminal in Bubble Tea
- [Claude Code TUI CPR leak in raw mode](https://github.com/anthropics/claude-code/issues/17787) — confirms CPR sequences from PTY programs leak to display without proper interception

---
*Pitfalls research for: multi-client terminal sharing, CLI status bar, TUI mode on Go/Wails daemon*
*Researched: 2026-04-14*
