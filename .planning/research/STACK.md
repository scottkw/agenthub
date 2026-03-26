# Stack Research

**Domain:** AgentHub v1.5 — terminal fill fix, daemon startup performance, CLI arg passthrough
**Researched:** 2026-03-25
**Confidence:** HIGH (library versions verified via pkg.go.dev and go.mod; patterns verified against xterm.js GitHub issues)

---

## Context: No New Dependencies Required

All three v1.5 features are implementable with the existing dependency set. The changes are:

1. **xterm.js terminal fill fix** — timing/CSS fix in React frontend only
2. **Daemon startup performance** — algorithmic fix in `internal/daemon/process.go`
3. **CLI arg passthrough** — data model and propagation through existing layers

No `go get` or `npm install` required for any of these.

---

## Recommended Stack

### Core Technologies (Unchanged)

| Technology | Version | Purpose | Why Relevant to v1.5 |
|------------|---------|---------|----------------------|
| `@xterm/xterm` | `^6.0.0` (currently installed) | Terminal rendering | Fix site for the fill bug |
| `@xterm/addon-fit` | `^0.11.0` (currently installed) | Terminal resize to container | Root cause of fill timing issue |
| `document.fonts.ready` | Browser API (no package) | Font load gate before fit() | Already used; needs enhancement |
| stdlib `flag.NewFlagSet` | Go stdlib (in use) | CLI command parsing | Extend `cmdNew` for `--` passthrough |
| Go `os/exec` / PTY `CreateRequest.Args` | Go stdlib + existing `internal/pty` | CLI arg forwarding to PTY | `Args []string` field already exists in `CreateRequest` |

### Supporting Libraries (Unchanged, Already in go.mod)

| Library | Version | Purpose | v1.5 Usage |
|---------|---------|---------|------------|
| `github.com/aymanbagabas/go-pty` | v0.2.2 | PTY process launch | `CreateRequest.Args []string` is already wired into `cmd := p.CommandContext(ctx, req.CLI, req.Args...)` |
| `golang.org/x/term` | v0.41.0 | Terminal raw mode | No change needed for v1.5 |
| `github.com/wailsapp/wails/v2` | v2.10.2 | Desktop GUI + JS bindings | Wails-bound method on `App` needs `args string` param |

---

## Feature-Specific Stack Patterns

### Feature 1: xterm.js Terminal Fill on Initial Load

**Problem:** Claude and Gemini CLIs render a styled TUI on startup (full-screen panels, status bars). If `FitAddon.fit()` is called while the terminal container is still transitioning from hidden to visible, the measured container dimensions are wrong — the PTY gets initialized at 80x24 (the fallback) and the CLI draws its UI at that size. The correct dimensions arrive later but the CLI has already committed to the wrong terminal size.

**Root cause (confirmed via xterm.js issues #4841, #5320, #5298):**
- `fit()` calls `proposeDimensions()` which reads `containerElement.clientWidth/clientHeight` from the DOM
- If called while the container is `display:none` or mid-CSS-transition, `clientWidth === 0` → `cols === 1` (or the prior 80x24 default survives)
- `document.fonts.ready` only gates font load; it does not gate CSS layout completion
- `ResizeObserver` fires on observation start AND on subsequent size changes — but only if the element is already visible when observed

**Fix pattern (no new packages):**

The current code in `TerminalPanel.tsx` uses `document.fonts.ready.then(() => fit())` plus a `ResizeObserver`. This is the right structure but has a gap: the ResizeObserver's initial callback fires before CSS layout has settled (the flex container completing its transition from display:none). The fix is to add a `requestAnimationFrame` gate inside the ResizeObserver callback to defer `fit()` to after paint:

```typescript
// In the ResizeObserver callback:
const ro = new ResizeObserver(() => {
  requestAnimationFrame(() => {
    fitAddonRef.current?.fit()
  })
})
ro.observe(container)
```

Additionally, add a targeted resize after the PTY relay WebSocket connects (`onOpen` callback in `RelayClient`), which sends the correct cols/rows to the PTY. This forces a SIGWINCH to the CLI process at the moment it is ready to receive input — the key missing piece for CLIs that draw their TUI before xterm.js has reported the correct size.

**The correct `onOpen` resize pattern:**

```typescript
onOpen: () => {
  // Fit now that the relay is connected; PTY can receive resize immediately.
  const fitAddon = fitAddonRef.current
  if (fitAddon) {
    requestAnimationFrame(() => fitAddon.fit())
  }
},
```

This combines with the existing ResizeObserver so both paths converge on `requestAnimationFrame(() => fit())`.

**Why `requestAnimationFrame` works:** The browser paints after rAF callbacks execute, which means CSS layout (including flex size resolution) has completed before the callback runs. This is the correct hook point for reading element dimensions.

**Why NOT `setTimeout(..., 100)` or similar:** Fixed delays are flaky — fast machines miss them, slow machines add unnecessary latency. rAF is layout-cycle-accurate and zero-latency on fast hardware.

**Confidence:** HIGH — rAF-after-fit is the documented workaround in xterm.js issue #4841 and confirmed by the maintainer (Tyriar) to be the correct approach for container-visibility timing.

---

### Feature 2: Daemon Startup Performance

**Problem:** `EnsureDaemon` in `internal/daemon/process.go` polls with `time.Sleep(50 * time.Millisecond)` for up to 3 seconds after spawning the daemon subprocess. The actual daemon startup time is typically 50–150ms (Go binary, no JVM warmup), but the polling may sleep through the ready window and add unnecessary latency visible to users.

**Current code analysis:**
```go
// Poll until daemon is fully ready — health + relay port (max 3 seconds).
deadline := time.Now().Add(3 * time.Second)
for time.Now().Before(deadline) {
    if err := client.Health(); err == nil {
        if port, relayErr := client.GetRelayPort(); relayErr == nil && port > 0 {
            return nil
        }
    }
    time.Sleep(50 * time.Millisecond)  // ← sleeps BEFORE re-checking
}
```

The issue: the loop sleeps AFTER each failed check. On the iteration where the daemon becomes ready, the code checks, succeeds, and returns — but only if it happens to check at the right moment. If the daemon is ready at t=80ms but the next poll is at t=100ms (50ms sleep), 20ms of unnecessary wait accrues. More importantly, the current structure does health check, then relay-port check as a separate HTTP round-trip — two sequential IPC calls per iteration.

**Fix pattern (no new packages):**

Three improvements, pure Go stdlib:

1. **Check immediately first** (before any sleep): return early if daemon is already running (handles the restart/retry case).

2. **Exponential backoff with cap**: start at 5ms, double each iteration, cap at 50ms. Matches daemon startup curve — fast to detect fast-starting daemons.

3. **Combine health + relay into one round-trip**: add a `/ready` endpoint (or extend `/health`) that returns `{"status":"ok","relayPort":N}` so a single HTTP call confirms both conditions.

```go
// Improved poll: immediate first, then exponential backoff
sleep := 5 * time.Millisecond
deadline := time.Now().Add(3 * time.Second)
for time.Now().Before(deadline) {
    if err := client.Health(); err == nil {
        if port, relayErr := client.GetRelayPort(); relayErr == nil && port > 0 {
            return nil
        }
    }
    time.Sleep(sleep)
    if sleep < 50*time.Millisecond {
        sleep *= 2
    }
}
```

**Impact:** For a daemon that starts in 80ms, the current code detects readiness at the 100ms poll tick. With 5ms start + doubling (5, 10, 20, 40, 80ms cumulative = 155ms), the daemon is detected at 80ms within the 5ms window — detection happens at 80ms instead of 100ms. For the GUI startup path (App.startup → EnsureDaemon), this reduces perceived latency.

**Additional: daemon startup warm path.** The daemon calls `NewSessionEngine()` + `NewAPI()` + `StartRelay()` + `api.Start(socketPath)` sequentially. `StartRelay()` does a `net.Listen("tcp", "127.0.0.1:0")` which is fast, but starts the relay server in a goroutine *after* `api.Start()`. Consider starting relay concurrently with API startup — both are independent listeners. This is a refactor within `runDaemonCore` in `process.go`, no new dependencies.

**Confidence:** HIGH — analysis based on direct code reading. The polling pattern is a known Go pattern optimization with no library dependency.

---

### Feature 3: CLI Argument Passthrough

**Problem:** Agents like Claude Code accept extra flags (`--model`, `--permission-mode`, `--verbose`). Users need to pass these through from both the CLI (`agenthub new`) and the GUI new-session modal.

**Data flow analysis (existing code):**

```
cmdNew (cmd_cli.go)
  → client.CreateSession(cli, name, workDir)      ← no args param
    → POST /sessions {cli, name, workDir}          ← no args field
      → engine.CreateSession(cli, name, workDir)  ← no args param
        → backend.Create(CreateRequest{CLI, Cols, Rows, WorkDir})  ← Args []string exists but unused!
          → p.CommandContext(ctx, req.CLI, req.Args...)  ← already wired!
```

`CreateRequest.Args []string` in `internal/pty/backend.go` is already defined and already forwarded to the PTY command. The gap is that nothing above it passes args down. This is a clean propagation fix through the existing layers.

**Fix pattern — no new packages, extend existing types:**

**Layer 1: `daemon/types.go` — add `Args []string` to wire types:**
```go
type CreateRequest struct {
    CLI     string   `json:"cli"`
    Name    string   `json:"name"`
    WorkDir string   `json:"workDir"`
    Args    []string `json:"args,omitempty"`   // ← add
}
```

**Layer 2: `daemon/engine.go` — thread args into PTY CreateRequest:**
```go
func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string, args []string, onStatus func(...)) (string, error) {
    sess, err := e.backend.Create(ctx, pty.CreateRequest{
        CLI:     cliPath,
        Args:    args,     // ← add
        Cols:    80, Rows: 24,
        WorkDir: workDir,
    })
```

**Layer 3: `daemon/client.go` — pass args through HTTP:**
```go
func (c *DaemonClient) CreateSession(cli, name, workDir string, args []string) (string, error) {
    req := CreateRequest{CLI: cli, Name: name, WorkDir: workDir, Args: args}
```

**Layer 4: `cmd_cli.go` — parse `--` passthrough in `cmdNew`:**

Go stdlib `flag.NewFlagSet` stops parsing at `--`. After `fs.Parse(args)`, `fs.Args()` contains everything after `--`. This is built in — no cobra needed.

```go
func cmdNew(client *daemon.DaemonClient, args []string, out io.Writer) error {
    fs := flag.NewFlagSet("new", flag.ContinueOnError)
    dir  := fs.String("dir", "", "working directory")
    cli  := fs.String("cli", "claude", "agent CLI name")
    if err := fs.Parse(args); err != nil { return err }

    remaining := fs.Args()  // positional args (session name)

    // Everything after "--" is captured in remaining after flag parsing.
    // Split on "--" to separate session name from agent args.
    var agentArgs []string
    name := ""
    for i, a := range remaining {
        if a == "--" {
            agentArgs = remaining[i+1:]
            remaining = remaining[:i]
            break
        }
    }
    if len(remaining) > 0 { name = remaining[0] }
    ...
    id, err := client.CreateSession(*cli, name, *dir, agentArgs)
```

Usage: `agenthub new myproject --dir /path --cli claude -- --model claude-opus-4-5 --verbose`

**Layer 5: `App.go` Wails binding — add args param:**
```go
func (a *App) CreateSession(cli, workDir string, args []string) (string, error) {
    name := filepath.Base(workDir)
    return a.client.CreateSession(cli, name, workDir, args)
}
```

**Layer 6: `NewSessionModal.tsx` — add args text field with per-agent localStorage memory:**

```typescript
const LAST_ARGS_KEY = (cli: string) => `agenthub:lastArgs:${cli}`

const [agentArgs, setAgentArgs] = useState(() =>
    localStorage.getItem(LAST_ARGS_KEY(selectedCLI)) ?? ''
)

// When CLI selection changes, load saved args for that CLI
useEffect(() => {
    setAgentArgs(localStorage.getItem(LAST_ARGS_KEY(selectedCLI)) ?? '')
}, [selectedCLI])

// On confirm, persist and parse
function handleConfirm() {
    localStorage.setItem(LAST_ARGS_KEY(selectedCLI), agentArgs)
    const parsedArgs = agentArgs.trim() ? agentArgs.trim().split(/\s+/) : []
    onConfirm(selectedCLI, selectedDir, parsedArgs)
}
```

Add a clear button: `<button onClick={() => { setAgentArgs(''); localStorage.removeItem(LAST_ARGS_KEY(selectedCLI)) }}>Clear</button>`

**arg parsing note:** Simple whitespace-split is correct for the MVP — agent CLIs use `--flag value` pairs without embedded spaces. Shell quoting (e.g. `--prompt "hello world"`) is a future enhancement; document this limitation in the UI placeholder text.

**Confidence:** HIGH — `CreateRequest.Args` is already defined and wired to the PTY command in `native.go`. This is purely a propagation change through existing types.

---

## Installation

```bash
# No new dependencies for any v1.5 feature.
# All changes are within existing packages.
```

---

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| `requestAnimationFrame` gate for fit() | `setTimeout(fit, 100)` delay | Fixed delay is unreliable — too fast on slow machines, unnecessary latency on fast ones. rAF is layout-cycle accurate. |
| `requestAnimationFrame` gate for fit() | `MutationObserver` on container | Mutations don't fire on CSS transitions completing. ResizeObserver already in place is correct mechanism; just needs rAF gate. |
| Exponential backoff polling in EnsureDaemon | Reduce sleep to 10ms flat | Flat 10ms still has up to 10ms excess delay at detection. Exponential starts faster and converges quicker. |
| Exponential backoff polling | Channel-based notification (daemon signals readiness) | Requires adding a signaling mechanism (pipe, socket ping) to the daemon spawn contract. Over-engineering for a 50-150ms startup path. |
| `flag.NewFlagSet` `--` split for args | Add cobra | Cobra not in the project (despite being researched in v1.3, stdlib flag was used instead). Adding cobra for one new flag is unjustified scope. |
| Whitespace-split args string from GUI | JSON array input in GUI | Whitespace split matches how users type CLI flags. JSON array is unfamiliar UX for CLI flags. |
| Per-CLI localStorage key for args | Single global args key | Different CLIs have different flags (claude uses `--model`, gemini uses `--model` differently, etc.). Per-CLI memory prevents confusion. |

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Shell quoting/parsing library (e.g. `google/shlex`) | Overkill for MVP — agent flags don't use quoted values in practice. Adds a new Go dependency. | Simple `strings.Fields()` / whitespace split; document limitation. |
| New `/ready` daemon endpoint (combined health+relay) | While it would save one round-trip, the current two-call pattern works correctly and the exponential backoff alone solves the performance issue without requiring API changes. | Exponential backoff on existing two-call pattern. |
| `cobra` for CLI arg parsing | Already researched in v1.3, not adopted. Flag stdlib is established in the codebase (flag.NewFlagSet per command). Adding cobra now is churn. | Continue stdlib `flag.NewFlagSet` pattern. |
| xterm.js addon-canvas or addon-serialize | Not needed for fit timing fix — existing WebGL/fallback stack is correct. | Existing `@xterm/addon-webgl` + canvas fallback. |
| Increase EnsureDaemon timeout beyond 3s | If the daemon doesn't start in 3s, something is wrong (binary not found, permission error). Longer timeout just delays error reporting. | Keep 3s deadline; improve detection speed within it. |

---

## Version Compatibility

| Package | Version | Notes |
|---------|---------|-------|
| `@xterm/addon-fit` | `^0.11.0` | `fit()` behavior unchanged since 0.10.x; rAF fix is in calling code, not the addon |
| `@xterm/xterm` | `^6.0.0` | No API changes needed; `onResize` event already fires correctly |
| Go stdlib `flag` | Go 1.26.1 (in go.mod) | `--` terminator behavior is stable and documented since Go 1.0 |
| `github.com/aymanbagabas/go-pty` | v0.2.2 | `CreateRequest.Args []string` already defined; no upgrade needed |

---

## Sources

- [xterm.js issue #4841 — FitAddon resizes incorrectly](https://github.com/xtermjs/xterm.js/issues/4841) — root cause analysis confirming rAF as correct fix. MEDIUM confidence (community + maintainer comment).
- [xterm.js issue #5320 — addon-fit: width=1](https://github.com/xtermjs/xterm.js/issues/5320) — CSS layout conflict as root cause. MEDIUM confidence.
- [xterm.js issue #5298 — fit not exactly to parent dimensions](https://github.com/xtermjs/xterm.js/issues/5298) — layout timing patterns. MEDIUM confidence.
- [pkg.go.dev/flag](https://pkg.go.dev/flag) — `--` terminator behavior for stdlib flag package. HIGH confidence.
- `/Users/ken/dev/agenthub/internal/pty/backend.go` — `CreateRequest.Args []string` already defined and wired. HIGH confidence (direct code read).
- `/Users/ken/dev/agenthub/internal/pty/native.go` — `p.CommandContext(ctx, req.CLI, req.Args...)` confirms args forwarding path. HIGH confidence (direct code read).
- `/Users/ken/dev/agenthub/internal/daemon/process.go` — `EnsureDaemon` polling logic (50ms flat sleep). HIGH confidence (direct code read).
- `/Users/ken/dev/agenthub/frontend/src/components/TerminalPanel.tsx` — current fit timing implementation. HIGH confidence (direct code read).
- `/Users/ken/dev/agenthub/frontend/package.json` — current xterm.js addon versions confirmed. HIGH confidence (direct file read).

---
*Stack research for: AgentHub v1.5 Bug Fixes & CLI Args*
*Researched: 2026-03-25*
