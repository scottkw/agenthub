# Architecture Research

**Domain:** Desktop app — bug fixes and CLI argument passthrough for AgentHub v1.5
**Researched:** 2026-03-25
**Confidence:** HIGH (direct codebase inspection of all affected files)

> This document supersedes the v1.3 architecture research. The daemon-centric
> architecture described there is now fully implemented in v1.4. This document
> focuses on the v1.5 integration points only.

---

## Current Architecture (v1.4 baseline)

```
┌──────────────────────────────────────────────────────────────────┐
│                      GUI Process (Wails)                          │
│  ┌────────────────────┐        ┌──────────────────────────────┐  │
│  │   React Frontend   │        │         app.go (App)         │  │
│  │  NewSessionModal   │←Wails→│  CreateSession(cli,name,dir) │  │
│  │  TerminalPanel     │ binds  │  (delegates to DaemonClient) │  │
│  │  (xterm.js + Fit)  │       └──────────────┬───────────────┘  │
│  └────────────────────┘                       │ HTTP/Unix socket  │
└───────────────────────────────────────────────┼──────────────────┘
                                                │
┌───────────────────────────────────────────────┼──────────────────┐
│                  Daemon Process               │                   │
│  ┌────────────────────────────────────────────▼───────────────┐  │
│  │                  daemon.API (HTTP mux)                      │  │
│  │  POST /sessions ───────────────────────────────────────┐   │  │
│  └───────────────────────────────────────────────────────-┼───┘  │
│  ┌──────────────────────────────────────────────────────────▼──┐  │
│  │                   daemon.SessionEngine                       │  │
│  │  CreateSession(ctx, cli, name, workDir, onStatus)           │  │
│  │    → backend.Create(pty.CreateRequest{CLI, Args, WorkDir})  │  │
│  └──────────────────────────────────────────┬────────────────--┘  │
│  ┌───────────────────────────────────────────▼──────────────────┐  │
│  │               pty.NativePTYBackend                           │  │
│  │  Create(req) → gopty.New() → cmd.Start(req.CLI, req.Args…)  │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘

CLI (same binary, different argv):
  agenthub new <agent> <path>
    → DaemonClient.CreateSession(cli, name, workDir)
    → POST /sessions  {cli, name, workDir}
```

---

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| `main.go` dispatch | Routes argv: no args → GUI, subcommand → CLI, `daemon` → service | `main.go` |
| `App` (Wails) | Thin binding shell; delegates all session ops to DaemonClient | `app.go` |
| `NewSessionModal` | Session creation UI: CLI picker, folder browser | `frontend/src/components/NewSessionModal.tsx` |
| `TerminalPanel` | xterm.js lifecycle; FitAddon timing | `frontend/src/components/TerminalPanel.tsx` |
| `cmdNew` | CLI handler for `agenthub new` | `cmd_cli.go` |
| `DaemonClient` | Typed HTTP client over Unix socket | `internal/daemon/client.go` |
| `daemon.API` | HTTP mux over Unix socket; deserializes requests | `internal/daemon/api.go` |
| `daemon.SessionEngine` | Owns all session state; calls backend.Create | `internal/daemon/engine.go` |
| `daemon.CreateRequest` | Wire type for POST /sessions | `internal/daemon/types.go` |
| `pty.CreateRequest` | Internal PTY spawn request (has `Args []string` already) | `internal/pty/backend.go` |
| `pty.NativePTYBackend` | Spawns process via go-pty using `req.CLI` and `req.Args` | `internal/pty/native.go` |
| `EnsureDaemon` | Spawns daemon subprocess + polls health; startup path | `internal/daemon/process.go` |

---

## Feature 1: CLI Argument Passthrough

### Integration Points

The PTY layer already supports args. `pty.CreateRequest.Args []string` exists (backend.go:13) and `NativePTYBackend.Create` already passes `req.Args` to the spawned process (native.go:41 — `p.CommandContext(childCtx, req.CLI, req.Args...)`). Zero changes needed below `SessionEngine`.

The gap is in the layers above: the wire type, engine signature, client, and UI all lack the `args` field.

### Call Chain: Current vs Target

```
Current GUI path:
  NewSessionModal.onConfirm(cli, workDir)
    → App.CreateSession(cli, name, workDir)
    → DaemonClient.CreateSession(cli, name, workDir)
    → POST /sessions  {cli, name, workDir}
    → engine.CreateSession(ctx, cli, name, workDir, nil)
    → backend.Create({CLI: path, Args: nil, WorkDir: dir})
    → exec(cli)   ← no args

Target GUI path:
  NewSessionModal.onConfirm(cli, workDir, args: string)
    → App.CreateSession(cli, name, workDir, args: []string)
    → DaemonClient.CreateSession(cli, name, workDir, args: []string)
    → POST /sessions  {cli, name, workDir, args: [...]}
    → engine.CreateSession(ctx, cli, name, workDir, args, nil)
    → backend.Create({CLI: path, Args: args, WorkDir: dir})
    → exec(cli, args...)   ← agent receives its arguments

Target CLI path:
  $ agenthub new claude /path/to/project -- --resume --continue
    → cmdNew parses positional args and "--" separator
    → DaemonClient.CreateSession("claude", "project", "/path", ["--resume","--continue"])
    → (same daemon path as GUI above)
```

### Components to Modify

| Component | File | Change |
|-----------|------|--------|
| `daemon.CreateRequest` | `internal/daemon/types.go` | Add `Args []string \`json:"args,omitempty"\`` |
| `daemon.SessionEngine.CreateSession` | `internal/daemon/engine.go` | Add `args []string` param; pass to `pty.CreateRequest{Args: args}` |
| `daemon.API.handleCreateSession` | `internal/daemon/api.go` | Pass `req.Args` to `engine.CreateSession` |
| `DaemonClient.CreateSession` | `internal/daemon/client.go` | Add `args []string` param to method and request struct construction |
| `App.CreateSession` (Wails) | `app.go` | Add `args []string` param to Wails-bound method |
| `NewSessionModal` | `frontend/src/components/NewSessionModal.tsx` | Add args text field, per-agent localStorage memory, clear button; update `onConfirm` signature |
| `App.tsx` | `frontend/src/App.tsx` | Update `handleCreate` to pass parsed args to `App.CreateSession` |
| `cmdNew` | `cmd_cli.go` | Parse `--` separator from args; pass trailing tokens to `CreateSession` |
| `wailsjs/` bindings | `frontend/src/wailsjs/go/main/App.js` | Auto-regenerated by `wails dev` / `wails build` when `App.CreateSession` signature changes |

**No changes needed:** `pty.backend.go`, `pty.native.go`, `pty.session.go` — PTY layer already supports args.

### Per-Agent Argument Memory

Pattern: extend the existing `LAST_DIR_KEY = 'agenthub:lastWorkDir'` localStorage pattern.

```
Key:   agenthub:lastArgs:<cliName>   (e.g., "agenthub:lastArgs:claude")
Value: raw args string as typed by user
Scope: per-CLI-name, not per-session
```

On CLI picker selection change in modal: load corresponding key from localStorage and pre-fill the text field. On confirm: save current text field value to localStorage. Clear button: sets field to empty string, removes localStorage key.

### Args Parsing in cmdNew

```go
// Find "--" separator in args
var agentArgs []string
for i, a := range args {
    if a == "--" {
        agentArgs = args[i+1:]
        args = args[:i]
        break
    }
}
// args now contains [agent, path]; agentArgs contains passthrough tokens
```

### Args Splitting in the GUI Text Field

Use a shell-word splitter, not `strings.Fields`. `strings.Fields` breaks on all whitespace, corrupting quoted arguments like `--system-prompt "do the thing"`.

Options:
- `github.com/google/shlex` — handles single/double quotes and backslash escapes; minimal, no transitive deps (MEDIUM confidence — widely used but not verified via Context7 in this session)
- Manual quote-aware split is acceptable for the limited flags AI CLIs actually use in practice

If shell-word splitting is deferred, the text field can document "space-separated flags without quoting" as a v1.5 limitation.

---

## Feature 2: Terminal Initial-Fit Fix

### Root Cause

In `TerminalPanel.tsx` the `isActive` useEffect (lines 103-128) fires when a tab becomes active:

```typescript
document.fonts.ready.then(() => {
  if (!cancelled) fit()
})
const ro = new ResizeObserver(fit)
ro.observe(container)
```

`document.fonts.ready` may already be resolved (resolved promise callbacks run as microtasks — synchronously before the next paint). When the callback fires, `fitAddon.fit()` reads `clientWidth`/`clientHeight` from the container. If the container's dimensions are not yet committed to the browser layout (transition from `display:none` to `display:flex`), `fit()` calculates wrong cols/rows.

For Claude and Gemini specifically: these CLIs immediately render their startup UI (prompts, banners, color output) before any user input arrives. If the initial PTY dimensions are wrong at that moment, the first render is corrupted and does not recover cleanly until a manual resize or window resize triggers another `fit()`.

The `ResizeObserver` fires on initial observation but only after the browser layout engine has settled — which is after the first paint, not at microtask time. On a fast machine the race is narrow; on a slow machine or a cold start with font loading, the race window is larger.

### Fix

Replace the direct `document.fonts.ready.then(fit)` call with a double-`requestAnimationFrame` deferral:

```typescript
document.fonts.ready.then(() => {
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      if (!cancelled) fit()
    })
  })
})
```

The double-rAF pattern (used by xterm.js internally) waits for two browser paint cycles. After both cycles, the browser has committed layout and the WebGL canvas has rendered its first frame. FitAddon's measurement of character cell dimensions against `clientWidth`/`clientHeight` is then reliable.

The `ResizeObserver` path is already correct and does not need to change — ResizeObserver callbacks fire after layout, not as microtasks.

### Components to Modify

| Component | File | Change |
|-----------|------|--------|
| `TerminalPanel` | `frontend/src/components/TerminalPanel.tsx` | Wrap the `document.fonts.ready` callback in double-rAF before calling `fit()` |

---

## Feature 3: Daemon Startup Latency

### Root Cause Analysis

There are two distinct latency sources users may be conflating:

**Source A: Daemon startup on first GUI open**

`EnsureDaemon` (process.go:58-91) spawns a subprocess and polls every 50ms up to 3 seconds. This is a one-time cost paid the first time the GUI opens after boot. If the daemon is installed as a service (`agenthub daemon install`) it starts at login and this cost is zero.

This is architecturally sound. No change needed here unless users report > 3s startup.

**Source B: Slow first status update after session creation**

`pollSessionStatus` (app.go:139-161) sleeps 2 seconds *before* the first poll:

```go
for time.Now().Before(deadline) {
    time.Sleep(2 * time.Second)   // ← sleeps BEFORE first check
    s, err := a.client.GetSessionStatus(sessionID)
    ...
}
```

This means the frontend receives no status event for at least 2 seconds after `CreateSession` returns. During those 2 seconds the tab shows no status indicator, which users perceive as "slow startup."

**Source C: PTY process spawn time**

`cmd.Start()` in `NativePTYBackend.Create` forks a process. This is typically <50ms on any platform. Not the bottleneck.

### Fix

Restructure `pollSessionStatus` to poll immediately, then use a shorter interval:

```go
func (a *App) pollSessionStatus(sessionID string) {
    var last string
    deadline := time.Now().Add(60 * time.Second)
    for time.Now().Before(deadline) {
        s, err := a.client.GetSessionStatus(sessionID)
        if err != nil {
            return
        }
        if s != last {
            last = s
            if a.ctx != nil && a.ctx.Value("frontend") != nil {
                runtime.EventsEmit(a.ctx, "session:status", map[string]string{
                    "sessionId": sessionID,
                    "status":    s,
                })
            }
            if s == string(status.StatusErrored) {
                return
            }
        }
        time.Sleep(500 * time.Millisecond)  // poll at end, not start
    }
}
```

### Components to Modify

| Component | File | Change |
|-----------|------|--------|
| `App.pollSessionStatus` | `app.go` | Move `time.Sleep` to end of loop; reduce interval from 2s to 500ms |

---

## Recommended Build Order

Dependencies drive the order. Each phase is independently testable.

### Phase 1: Wire Args Through the Backend Stack (Go only, no UI changes)

All changes are additive and backward-compatible — existing callers pass no args, which is identical to passing `nil`/empty slice.

1. `internal/daemon/types.go` — add `Args []string` to `CreateRequest`
2. `internal/daemon/engine.go` — add `args []string` param to `CreateSession`; pass to `pty.CreateRequest{Args: args}`
3. `internal/daemon/api.go` — pass `req.Args` to `engine.CreateSession`
4. `internal/daemon/client.go` — add `args []string` param to `CreateSession`

Validation: existing 194 tests pass; add daemon tests for `CreateRequest` round-trip with non-empty args.

### Phase 2: CLI Command Passthrough

Modify `cmd_cli.go`:
- `cmdNew` parses `--` separator; passes trailing tokens to `client.CreateSession`
- Update `usage()` and `--help` text

Validation: add test case to `cmd_cli_test.go` covering `-- --flag value` passthrough.

### Phase 3: Daemon Startup Latency Fix

Modify `app.go` — restructure `pollSessionStatus` as described above.

Validation: measure time from `CreateSession` call to first `session:status` event in frontend console; target < 1s.

### Phase 4: GUI Wails Binding + Modal UI

1. `app.go` — add `args []string` param to `App.CreateSession`
2. Run `wails dev` to regenerate `wailsjs/` bindings
3. `NewSessionModal.tsx` — add args text field, per-agent localStorage memory, clear button; update `onConfirm` signature
4. `App.tsx` — update `handleCreate` to pass args

Validation: vitest source-inspection tests for `NewSessionModal`; manual smoke test in dev with `claude -- --resume`.

### Phase 5: Terminal Initial-Fit Fix

Modify `TerminalPanel.tsx` — add double-rAF wrapper as described above.

Validation: manual test with Claude and Gemini — terminal must fill screen on first tab open without window resize.

---

## Data Flow

### Args Flow: GUI Path

```
NewSessionModal text field → onConfirm(cli, workDir, argsString)
  → App.tsx parseArgs(argsString) → string[]
  → App.CreateSession(cli, name, workDir, args)        [Wails call]
  → DaemonClient.CreateSession(cli, name, workDir, args) [Unix socket]
  → POST /sessions  {cli, name, workDir, args: [...]}
  → daemon.API decodes CreateRequest
  → engine.CreateSession(ctx, cli, name, workDir, args, nil)
  → backend.Create(pty.CreateRequest{CLI: path, Args: args, WorkDir: dir})
  → NativePTYBackend: p.CommandContext(ctx, req.CLI, req.Args...)
  → OS: exec(cli, args[0], args[1], ...)
```

### Args Flow: CLI Path

```
$ agenthub new claude /path -- --resume --continue
  → cmdNew: args=["claude","/path"], agentArgs=["--resume","--continue"]
  → DaemonClient.CreateSession("claude", "path", "/path", ["--resume","--continue"])
  → (same daemon path as GUI above)
```

### Terminal Fit Flow (after fix)

```
React: new tab created, isActive=true
  → isActive useEffect fires
  → document.fonts.ready.then(...)      ← may resolve immediately
      → rAF #1 queued                   ← deferred to next paint cycle
          → rAF #2 queued               ← deferred to paint cycle after that
              → fitAddon.fit()          ← container dimensions now stable
                  → term.onResize fires
                      → RelayClient.sendResize(cols, rows)
                          → daemon.backend.Resize(id, cols, rows)
                              → go-pty: ioctl(TIOCSWINSZ)
```

### Status Event Flow (after fix)

```
App.CreateSession called
  → client.CreateSession → POST /sessions → returns sessionID
  → go pollSessionStatus(sessionID)
      → immediately: GetSessionStatus → emit "session:status"  ← < 50ms
      → after 500ms: poll again if changed
      → ... (500ms intervals for 60s)
```

---

## Anti-Patterns

### Anti-Pattern 1: Shell-Splitting with strings.Fields

**What people do:** Split the GUI args text field with `strings.Fields(input)`.

**Why it's wrong:** Users pass quoted arguments like `--system-prompt "do the thing"`. `strings.Fields` splits on all whitespace, producing `["--system-prompt", "\"do", "the", "thing\""]`.

**Do this instead:** Use a shell-word splitter (`github.com/google/shlex` or equivalent) that handles single/double quotes and backslash escapes.

### Anti-Pattern 2: Fitting xterm.js Synchronously on Mount

**What people do:** Call `fitAddon.fit()` directly inside `useEffect` or in the `document.fonts.ready` microtask callback.

**Why it's wrong:** The browser has not committed layout dimensions at microtask time. FitAddon reads zero or stale `clientWidth`/`clientHeight`.

**Do this instead:** Defer `fit()` via `requestAnimationFrame` inside the `document.fonts.ready` callback. Two rAF calls ensure both layout and the WebGL canvas first-frame are complete.

### Anti-Pattern 3: Changing CreateSession Signature Across All Callers at Once

**What people do:** Add the `args` parameter and update all callers (types, engine, api, client, app, cmdNew, tests) in a single commit.

**Why it's wrong:** Large, hard-to-review diff; easy to miss a call site; test failures become hard to isolate.

**Do this instead:** Follow the build order: types first (additive, no caller changes), then engine/api/client (pass empty slice for now), then cmdNew, then GUI. Gate each phase on passing tests.

### Anti-Pattern 4: Polling Status Before the PTY Has Started

**What people do:** Call `GetSessionStatus` immediately after `CreateSession` returns.

**Why it's fine for v1.5:** `CreateSession` over Unix socket returns after the PTY process has started (the engine's `backend.Create` call completes before the HTTP response). The status watcher goroutine is running. An immediate poll will return `"running"` which is the correct initial state. No sleep needed before the first poll.

---

## Integration Points

### Internal Boundaries

| Boundary | Communication | Impact for v1.5 |
|----------|---------------|-----------------|
| GUI App → DaemonClient | Direct Go method call | `CreateSession` signature gains `args []string` — Wails regenerates JS bindings |
| DaemonClient → daemon.API | HTTP/JSON over Unix socket | `CreateRequest` gains `Args []string` — backward-compatible (omitempty) |
| daemon.API → SessionEngine | Direct Go method call (same process) | `CreateSession` signature gains `args []string` |
| SessionEngine → NativePTYBackend | `pty.CreateRequest` struct | Already has `Args []string` — no change |
| React → Wails binding | Wails-generated `wailsjs/` | Regenerated automatically when `App.CreateSession` changes |
| xterm.js → relay WebSocket | Binary-framed resize/input/output | Unchanged; only initial fit timing changes |

### File Change Summary

```
No new files needed. All v1.5 changes are modifications to existing files.

internal/daemon/types.go       MODIFIED — Args []string on CreateRequest
internal/daemon/engine.go      MODIFIED — args param on CreateSession
internal/daemon/api.go         MODIFIED — pass req.Args to engine
internal/daemon/client.go      MODIFIED — args param on CreateSession

app.go                         MODIFIED — args param on CreateSession; fix pollSessionStatus
cmd_cli.go                     MODIFIED — cmdNew parses -- separator

frontend/src/components/
  NewSessionModal.tsx           MODIFIED — args text field + per-agent memory
  TerminalPanel.tsx             MODIFIED — double-rAF for initial fit
frontend/src/App.tsx            MODIFIED — pass args to CreateSession

frontend/src/wailsjs/           AUTO-REGENERATED by wails dev/build
```

---

## Sources

- Codebase inspection (HIGH confidence): all integration points verified against working tree at v1.4 HEAD
- `internal/pty/backend.go:13` — `CreateRequest.Args []string` already present
- `internal/pty/native.go:41` — `p.CommandContext(childCtx, req.CLI, req.Args...)` already passes args
- `app.go:144` — `time.Sleep(2 * time.Second)` before first poll is the status latency source
- `frontend/src/components/TerminalPanel.tsx:115` — `document.fonts.ready.then(() => fit())` is the fit timing issue
- `frontend/src/components/NewSessionModal.tsx:4` — `LAST_DIR_KEY` localStorage pattern to extend for per-agent args

---

*Architecture research for: AgentHub v1.5 — bug fixes and CLI argument passthrough*
*Researched: 2026-03-25*
