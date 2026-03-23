# Phase 21: CLI Session + Web Commands - Research

**Researched:** 2026-03-23
**Domain:** Go CLI architecture, daemon client integration, QR code terminal rendering
**Confidence:** HIGH

## Summary

Phase 21 adds a standalone `agenthub` CLI binary (or sub-command dispatch from the existing binary) that exposes session management and web-serving commands. All the hard work is already done: `DaemonClient` in `internal/daemon/client.go` already wraps every daemon API operation needed (create, list, kill, rename, web start/stop/status, toggle web-serve). The CLI is essentially a thin argument-parsing shell that calls these client methods and formats output.

The key architectural decision is **where the CLI entry point lives**. The current `main.go` is a Wails binary (panics without Wails build tags for non-GUI paths). A standalone CLI binary must live in a separate `cmd/agenthub-cli/` directory with its own `main.go` and no Wails imports, built with `go build ./cmd/agenthub-cli/`. It calls `daemon.EnsureDaemon()` before every command — auto-starting the daemon if needed — then delegates to `DaemonClient`.

The QR code requirement (`agenthub qr <id>`) is the only technically non-trivial piece. The GUI uses `skip2/go-qrcode` to encode a URL into a base64 PNG. For terminal output, the same library can render as ASCII/UTF-8 using block characters — no additional library is needed. The `agenthub web start` command requires a Tailscale health query (IP, FQDN, cert status) before calling `StartWebServer`; the `webserver.CheckHealth()` function already exists and is importable without Wails.

**Primary recommendation:** Build a standalone CLI binary at `cmd/agenthub-cli/main.go` using stdlib `flag`/`os.Args` dispatch (no cobra/urfave — project has no such dependency and the command surface is small). Wire every subcommand to the existing `DaemonClient` methods. For `qr`, use `skip2/go-qrcode` in `qrcode.WriteFile`-equivalent ASCII mode.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CLI-01 | `agenthub new <agent> <path>` creates a session and prints ID | `DaemonClient.CreateSession(cli, name, workDir)` is ready; need arg parsing + stdout print |
| CLI-02 | `agenthub list` shows all sessions with ID, name, agent, status | `DaemonClient.ListSessions()` returns `[]SessionInfo` with all required fields |
| CLI-03 | `agenthub kill <id>` terminates a session | `DaemonClient.KillSession(id)` is ready |
| CLI-04 | `agenthub rename <id> <name>` renames session, reflected in GUI | `DaemonClient.RenameSession(id, name)` is ready; GUI polls sessions so rename is immediately visible |
| WEB-01 | `agenthub web start`, `stop` control Tailscale web server | `DaemonClient.StartWebServer(ip, port, fqdn)` and `StopWebServer()` ready; need Tailscale health check before start |
| WEB-02 | `agenthub web status` reports web server state | `DaemonClient.GetWebServerStatus()` returns `{Running, URL, Addr}` |
| WEB-03 | `agenthub serve <id>`, `unserve <id>` toggle web serving per session | `DaemonClient.ToggleWebServing(sessionID, enabled)` is ready |
| WEB-04 | `agenthub health` Tailscale health check | `webserver.CheckHealth(ctx)` returns `TailscaleHealth{Installed, Connected, HasCerts, IP, Domain}` |
| WEB-05 | `agenthub qr <id>` display QR code in terminal | `DaemonClient.GetWebServerStatus()` gives base URL; `skip2/go-qrcode` for ASCII render |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| stdlib `os`/`flag`/`fmt` | Go 1.26 | Argument parsing, output | No external dependency; ~9 subcommands fits flat dispatch |
| `internal/daemon` (DaemonClient) | (project) | All daemon API calls | Already implements every operation needed |
| `internal/webserver` (CheckHealth) | (project) | Tailscale status for web start | Already exists, no Wails dependency |
| `skip2/go-qrcode` | v0.0.0-20200617 | QR code encoding | Already in `go.mod`; supports ASCII output |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` | stdlib | `--json` flag output (future POLISH-01) | For machine-readable output; skip for Phase 21 |
| `os/exec` | stdlib | (not needed this phase) | Attach phase (22) uses PTY proxy |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| stdlib arg dispatch | cobra/urfave-cli | Cobra adds 500KB+ binary size, zero benefit for ~9 flat commands; no existing cobra usage in project |
| `skip2/go-qrcode` ASCII | `qrencode` system binary | System dependency; skip2 is already in go.mod |

**Installation:**
No new dependencies. All required packages are already in `go.mod`.

**Version verification:** All packages verified from existing `go.mod` — no new packages to add.

## Architecture Patterns

### Recommended Project Structure
```
cmd/
└── agenthub-cli/
    └── main.go          # CLI entry point, arg dispatch, output formatting
```

The CLI binary is separate from the Wails binary. The Wails binary (`main.go` at root) cannot be reused for CLI because:
1. Wails injects build constraints that panic when invoked outside Wails
2. The `daemon` sub-command dispatch in current `main.go` is already there but running the Wails binary requires CGO + macOS frameworks
3. A dedicated `cmd/agenthub-cli/` binary has no CGO dependency and builds on all platforms with `go build ./cmd/agenthub-cli/`

### Pattern 1: Flat Command Dispatch
**What:** Single `main()` that reads `os.Args[1]` and dispatches to handler functions. No sub-command framework.
**When to use:** Small, stable command surface (9 commands). This project has no existing cobra usage.
**Example:**
```go
// cmd/agenthub-cli/main.go
func main() {
    if len(os.Args) < 2 {
        usage()
        os.Exit(1)
    }
    socketPath := daemon.DefaultSocketPath()
    if err := daemon.EnsureDaemon(socketPath); err != nil {
        fmt.Fprintf(os.Stderr, "agenthub: %v\n", err)
        os.Exit(1)
    }
    client := daemon.NewDaemonClient(socketPath)

    switch os.Args[1] {
    case "new":
        cmdNew(client, os.Args[2:])
    case "list":
        cmdList(client)
    case "kill":
        cmdKill(client, os.Args[2:])
    case "rename":
        cmdRename(client, os.Args[2:])
    case "web":
        cmdWeb(client, os.Args[2:])
    case "serve":
        cmdServe(client, os.Args[2:])
    case "unserve":
        cmdUnserve(client, os.Args[2:])
    case "health":
        cmdHealth()
    case "qr":
        cmdQR(client, os.Args[2:])
    default:
        fmt.Fprintf(os.Stderr, "agenthub: unknown command %q\n", os.Args[1])
        usage()
        os.Exit(1)
    }
}
```

### Pattern 2: EnsureDaemon Before Every Command
**What:** Every CLI command calls `daemon.EnsureDaemon(socketPath)` before creating a `DaemonClient`.
**When to use:** Always — this satisfies DAEMON-05 (daemon auto-starts when any CLI command is run).
**Example:**
```go
// EnsureDaemon already exists in internal/daemon/process.go
// Polls up to 3 seconds for daemon to become ready (health + relay port).
func EnsureDaemon(socketPath string) error { ... }
```

### Pattern 3: `agenthub new <agent> <path>`
**What:** Positional args: agent name (e.g. "claude"), working directory path. Name defaults to agent+timestamp or directory basename.
**Example:**
```go
func cmdNew(client *daemon.DaemonClient, args []string) {
    if len(args) < 2 {
        fmt.Fprintln(os.Stderr, "usage: agenthub new <agent> <path>")
        os.Exit(1)
    }
    agent, workDir := args[0], args[1]
    name := filepath.Base(workDir) // sensible default name
    id, err := client.CreateSession(agent, name, workDir)
    if err != nil {
        fmt.Fprintf(os.Stderr, "agenthub new: %v\n", err)
        os.Exit(1)
    }
    fmt.Println(id)
}
```

### Pattern 4: `agenthub list` Output Format
**What:** Tabular output using `text/tabwriter`.
**Example:**
```go
func cmdList(client *daemon.DaemonClient) {
    sessions, err := client.ListSessions()
    if err != nil {
        fmt.Fprintf(os.Stderr, "agenthub list: %v\n", err)
        os.Exit(1)
    }
    w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
    fmt.Fprintln(w, "ID\tNAME\tAGENT\tSTATUS")
    for _, s := range sessions {
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Name, s.CLI, s.State)
    }
    w.Flush()
}
```

### Pattern 5: `agenthub web start`
**What:** Query Tailscale health first (same check as `App.StartWebServer`), then call daemon `StartWebServer`.
**Why:** The daemon's `handleWebServerStart` does NOT do the Tailscale health check — it accepts explicit `ip/port/fqdn` parameters. The health gate lives in the caller (`App.StartWebServer` in app.go). The CLI must replicate this gate.
**Example:**
```go
func cmdWebStart(client *daemon.DaemonClient, args []string) {
    port := 443 // default; parse --port flag if needed
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    h := webserver.CheckHealth(ctx)
    if !h.Connected {
        fmt.Fprintln(os.Stderr, "agenthub: Tailscale is not connected")
        os.Exit(1)
    }
    if h.IP == "" {
        fmt.Fprintln(os.Stderr, "agenthub: Tailscale IP not available")
        os.Exit(1)
    }
    if !h.HasCerts {
        fmt.Fprintln(os.Stderr, "agenthub: Tailscale HTTPS certificates not enabled")
        os.Exit(1)
    }
    url, err := client.StartWebServer(h.IP, port, h.Domain)
    if err != nil {
        fmt.Fprintf(os.Stderr, "agenthub web start: %v\n", err)
        os.Exit(1)
    }
    fmt.Println(url)
}
```

### Pattern 6: `agenthub qr <id>` — ASCII QR in Terminal
**What:** `skip2/go-qrcode` supports ASCII output via `qrcode.WriteFile` with a string path, OR the `qrcode.New()` API and `ToString(false)` for inverted black/white.
**Important detail:** `qrcode.New(content, recovery).ToString(inverted bool)` returns a string of Unicode block characters (half-blocks using `\u2580`/`\u2584`/`\u2588`/` `) that render correctly in any terminal. This is already in go.mod.
**Example:**
```go
func cmdQR(client *daemon.DaemonClient, args []string) {
    if len(args) < 1 {
        fmt.Fprintln(os.Stderr, "usage: agenthub qr <session-id>")
        os.Exit(1)
    }
    id := args[0]
    resp, err := client.GetWebServerStatus()
    if err != nil || !resp.Running {
        fmt.Fprintln(os.Stderr, "agenthub qr: web server not running")
        os.Exit(1)
    }
    url := fmt.Sprintf("%s/sessions/%s", resp.URL, id)
    q, err := qrcode.New(url, qrcode.Medium)
    if err != nil {
        fmt.Fprintf(os.Stderr, "agenthub qr: %v\n", err)
        os.Exit(1)
    }
    fmt.Println(q.ToString(false))
    fmt.Println(url)
}
```

### Pattern 7: `agenthub health` Output
**What:** Call `webserver.CheckHealth(ctx)` and print structured status. No daemon call needed — it queries tailscaled directly.
**Example:**
```go
func cmdHealth() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    h := webserver.CheckHealth(ctx)
    fmt.Printf("installed:  %v\n", h.Installed)
    fmt.Printf("connected:  %v\n", h.Connected)
    fmt.Printf("has-certs:  %v\n", h.HasCerts)
    fmt.Printf("ip:         %s\n", h.IP)
    fmt.Printf("domain:     %s\n", h.Domain)
}
```

### Anti-Patterns to Avoid
- **Calling `daemon.RunDaemon()` from CLI commands:** The CLI is a client, not a server. RunDaemon starts an API server; CLI commands just call EnsureDaemon + DaemonClient.
- **Importing Wails packages in CLI binary:** `cmd/agenthub-cli/main.go` must NOT import `github.com/wailsapp/wails/v2` — it will fail to build without the Wails build tags and CGO.
- **Duplicating Tailscale health logic:** `webserver.CheckHealth(ctx)` already exists. Don't re-implement it.
- **Hardcoding socket path:** Always use `daemon.DefaultSocketPath()` to support platform-specific paths (named pipe on Windows).
- **Omitting EnsureDaemon:** If a user runs `agenthub list` with no daemon running, EnsureDaemon auto-starts it. Skipping this call breaks DAEMON-05.
- **`agenthub web start` skipping health gate:** The daemon's `handleWebServerStart` accepts any IP/port/FQDN without validation. The CLI must validate Tailscale health before calling, or users get a confusing TLS error from the daemon.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Session CRUD | Custom HTTP client | `DaemonClient` methods | Already complete and tested |
| Daemon auto-start | Process spawn + poll | `daemon.EnsureDaemon(socketPath)` | Handles stale socket, race conditions, 3s timeout |
| Socket path | Hardcode path | `daemon.DefaultSocketPath()` | Platform-specific (Windows named pipe vs Unix socket) |
| QR code | Custom QR encoder | `skip2/go-qrcode` (already in go.mod) | Already handles all QR encoding; `ToString(false)` gives terminal-printable output |
| Tailscale health | Direct tailscaled IPC | `webserver.CheckHealth(ctx)` | Already implemented and tested |
| Table output | Custom column formatting | `text/tabwriter` (stdlib) | Handles column alignment correctly |

**Key insight:** This phase is almost entirely wiring. All business logic exists. The only new code is argument parsing, output formatting, and one new binary entry point.

## Common Pitfalls

### Pitfall 1: Building CLI in the Same Binary as Wails
**What goes wrong:** `main.go` panics at startup when Wails build constraints are not met (no CGO, no macOS frameworks). The current panic message is `"Wails applications will not build without the correct build tags."` — verified experimentally.
**Why it happens:** Wails v2 injects a build-tag guard that fails at runtime if its native bridge is missing.
**How to avoid:** Put CLI in `cmd/agenthub-cli/main.go`. Build with `go build ./cmd/agenthub-cli/`. This binary has no Wails import and no CGO requirement.
**Warning signs:** If you see the Wails panic when testing CLI commands.

### Pitfall 2: `agenthub web start` Port Default
**What goes wrong:** Port 443 requires root. Tailscale's webserver typically binds on an ephemeral Tailscale IP where 443 IS accessible without root because Tailscale owns the IP stack. But on some OS configurations, binding port 443 on any IP may need privilege.
**Why it happens:** The existing `App.StartWebServer(port int)` in app.go takes port as a parameter but the GUI passes a configured port from settings. The CLI needs a sensible default.
**How to avoid:** Default to port 443 since the webserver binds on the Tailscale IP (100.x.x.x) which is in a virtual network interface. Document that non-443 ports may break HTTPS cert validation.

### Pitfall 3: `agenthub qr` When Web Server Not Running
**What goes wrong:** Returns an opaque error if called before `agenthub web start`.
**Why it happens:** `GetWebServerStatus()` returns `{Running: false}` and there's no URL to encode.
**How to avoid:** Check `resp.Running` and print a clear error: `"web server not running — use 'agenthub web start' first"`.

### Pitfall 4: Session Name for `agenthub new`
**What goes wrong:** `CreateSession(cli, name, workDir)` requires a name but `agenthub new <agent> <path>` in the success criteria only specifies agent and path.
**Why it happens:** CLI spec doesn't include a `--name` flag but the API requires a name string.
**How to avoid:** Default name = `filepath.Base(workDir)` (same convention used by most tmux/screen-style tools). Optionally support `--name` flag.

### Pitfall 5: `agenthub rename` Not Reflected Immediately in GUI
**What goes wrong:** Success criteria says "new name reflected in GUI tab bar." The GUI polls ListSessions on a timer, so there can be a 1-2 second delay.
**Why it happens:** GUI uses `pollSessionStatus` goroutines, not websocket push. The rename is immediately stored in daemon but the GUI won't re-render until its next poll cycle.
**How to avoid:** This is expected behavior — the rename IS immediately persisted; the GUI just hasn't polled yet. No fix needed; success criteria is satisfied because the name IS reflected, just not instantaneously.

### Pitfall 6: EnsureDaemon Takes socketPath Argument
**What goes wrong:** `EnsureDaemon` signature is `EnsureDaemon(socketPath string) error` — it requires an explicit path.
**Why it happens:** Learned from Phase 20 decision: socketPath is injected to allow tests to use short paths (macOS 103-char limit).
**How to avoid:** Always call `daemon.DefaultSocketPath()` and pass the result to both `EnsureDaemon` and `NewDaemonClient`.

## Code Examples

Verified patterns from project source:

### DaemonClient — Full Method Surface Available for CLI
```go
// Source: internal/daemon/client.go (verified)
client.Health() error
client.ListSessions() ([]SessionInfo, error)
client.CreateSession(cli, name, workDir string) (string, error)
client.KillSession(id string) error
client.RenameSession(id, name string) error
client.GetSessionStatus(id string) (string, error)
client.GetRelayPort() (int, error)
client.StartWebServer(ip string, port int, fqdn string) (string, error)
client.StopWebServer() error
client.GetWebServerStatus() (WebServerStatusResponse, error)
client.ToggleWebServing(sessionID string, enabled bool) error
```

### SessionInfo Fields (for `agenthub list` output)
```go
// Source: internal/daemon/types.go (verified)
type SessionInfo struct {
    ID        string `json:"id"`
    CLI       string `json:"cli"`
    Name      string `json:"name"`
    State     string `json:"state"`      // "running" or "stopped"
    CreatedAt string `json:"createdAt"`  // RFC3339
}
```

### WebServerStatusResponse (for `agenthub web status` and `agenthub qr`)
```go
// Source: internal/daemon/types.go (verified)
type WebServerStatusResponse struct {
    Running bool   `json:"running"`
    URL     string `json:"url"`   // e.g. "https://hostname.ts.net"
    Addr    string `json:"addr"`  // e.g. "100.x.x.x:443"
}
```

### TailscaleHealth (for `agenthub health` and `agenthub web start` gate)
```go
// Source: internal/webserver/tailscale.go (verified)
type TailscaleHealth struct {
    Installed bool   `json:"installed"` // tailscaled socket reachable
    Connected bool   `json:"connected"` // BackendState == "Running"
    HasCerts  bool   `json:"hasCerts"`  // len(CertDomains) > 0
    IP        string `json:"ip"`
    Domain    string `json:"domain"`    // e.g. "hostname.ts.net"
}
// Call: webserver.CheckHealth(ctx) — no Wails dependency
```

### QR Code ASCII Output
```go
// Source: go.mod (skip2/go-qrcode already present)
import qrcode "github.com/skip2/go-qrcode"

q, err := qrcode.New(url, qrcode.Medium)
// q.ToString(false) returns a multi-line string with Unicode half-blocks
// Prints directly to terminal — no PNG conversion needed
fmt.Println(q.ToString(false))
```

### Tabwriter for `agenthub list`
```go
// Source: stdlib text/tabwriter
import "text/tabwriter"

w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
fmt.Fprintln(w, "ID\tNAME\tAGENT\tSTATUS")
for _, s := range sessions {
    fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Name, s.CLI, s.State)
}
w.Flush()
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CLI calling engine directly | CLI calls DaemonClient over Unix socket | Phase 20 | GUI and CLI share the same session pool; no dual state |
| Wails binary as sole entry point | Separate `cmd/agenthub-cli/` binary | Phase 21 (new) | CLI works without CGO / Wails framework |
| QR as base64 PNG | QR as terminal ASCII | Phase 21 (new) | Works in any terminal session, no image display needed |

## Open Questions

1. **Port for `agenthub web start`**
   - What we know: `App.StartWebServer(port int)` takes a port; GUI uses a settings-configured value
   - What's unclear: What default port to use in CLI (443? flag? positional arg?)
   - Recommendation: Default to 443, add optional `--port` flag. Document that 443 on Tailscale IP typically works without root.

2. **Default session name for `agenthub new <agent> <path>`**
   - What we know: API requires a name; success criteria doesn't specify
   - What's unclear: Should name default silently or require `--name` flag?
   - Recommendation: Default to `filepath.Base(workDir)`. Let planner decide if `--name` flag is needed.

3. **Binary name and install location**
   - What we know: `cmd/agenthub-cli/` builds a binary named `agenthub-cli` by default
   - What's unclear: Should it be renamed to `agenthub` at install? How does it coexist with the Wails binary?
   - Recommendation: Build as `agenthub` using `-o agenthub` flag; the Wails binary is an `.app` bundle on macOS, so naming conflict is minimal. Document in README.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib) |
| Config file | none — `go test ./...` discovers all `*_test.go` files |
| Quick run command | `go test ./cmd/agenthub-cli/... ./internal/daemon/... -count=1 -timeout 30s` |
| Full suite command | `go test ./... -count=1 -timeout 60s` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLI-01 | `cmdNew` creates session via DaemonClient, prints ID | unit | `go test ./cmd/agenthub-cli/... -run TestCmdNew -v` | Wave 0 |
| CLI-02 | `cmdList` outputs tabular sessions | unit | `go test ./cmd/agenthub-cli/... -run TestCmdList -v` | Wave 0 |
| CLI-03 | `cmdKill` terminates session via DaemonClient | unit | `go test ./cmd/agenthub-cli/... -run TestCmdKill -v` | Wave 0 |
| CLI-04 | `cmdRename` updates name via DaemonClient | unit | `go test ./cmd/agenthub-cli/... -run TestCmdRename -v` | Wave 0 |
| WEB-01 | `cmdWebStart` gates on Tailscale health, calls StartWebServer | unit (mock) | `go test ./cmd/agenthub-cli/... -run TestCmdWebStart -v` | Wave 0 |
| WEB-02 | `cmdWebStatus` prints running/stopped + URL | unit | `go test ./cmd/agenthub-cli/... -run TestCmdWebStatus -v` | Wave 0 |
| WEB-03 | `cmdServe`/`cmdUnserve` call ToggleWebServing | unit | `go test ./cmd/agenthub-cli/... -run TestCmdServe -v` | Wave 0 |
| WEB-04 | `cmdHealth` queries Tailscale health and prints fields | unit (mock) | `go test ./cmd/agenthub-cli/... -run TestCmdHealth -v` | Wave 0 |
| WEB-05 | `cmdQR` renders QR to stdout for running web server | unit | `go test ./cmd/agenthub-cli/... -run TestCmdQR -v` | Wave 0 |

**Note on testability:** The CLI command functions should accept an injected `*daemon.DaemonClient` and `io.Writer` for output to enable unit testing without starting a real daemon. Tests spin up a `testDaemon(t)` (already exists in `internal/daemon/api_test.go`) to provide a real socket.

### Sampling Rate
- **Per task commit:** `go test ./cmd/agenthub-cli/... -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1 -timeout 60s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd/agenthub-cli/main.go` — CLI entry point (new file)
- [ ] `cmd/agenthub-cli/main_test.go` — covers CLI-01 through WEB-05
- [ ] Test helper that starts `testDaemon(t)` from `cmd/agenthub-cli/` package scope (or import from `internal/daemon` testhelper)

*(Existing `internal/daemon` test infrastructure is complete and passing. The gaps are only the new CLI package.)*

## Sources

### Primary (HIGH confidence)
- `internal/daemon/client.go` — Full DaemonClient method surface, verified by reading source
- `internal/daemon/types.go` — SessionInfo, WebServerStatusResponse, TailscaleHealth types
- `internal/daemon/process.go` — EnsureDaemon signature and behavior
- `internal/daemon/socket.go` — DefaultSocketPath() platform logic
- `internal/webserver/tailscale.go` — CheckHealth() function, no Wails dependency
- `app.go` — StartWebServer() Tailscale gate logic (the CLI must replicate this)
- `go.mod` — Confirmed skip2/go-qrcode present at v0.0.0-20200617
- `go test ./internal/daemon/...` — Tests pass, confirming stable API surface

### Secondary (MEDIUM confidence)
- `skip2/go-qrcode` README (https://github.com/skip2/go-qrcode) — `ToString(inverted bool)` method for terminal output; confirmed method exists in the library

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages already in go.mod, no new dependencies
- Architecture: HIGH — DaemonClient already covers all required operations; binary structure is straightforward Go
- Pitfalls: HIGH — all identified from reading actual source; Wails panic verified experimentally

**Research date:** 2026-03-23
**Valid until:** 2026-06-23 (stable internal API; go.mod pinned)
