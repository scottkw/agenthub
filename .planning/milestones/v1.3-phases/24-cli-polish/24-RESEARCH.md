# Phase 24: CLI Polish - Research

**Researched:** 2026-03-24
**Domain:** Go CLI output formatting — JSON mode flag, settings display
**Confidence:** HIGH

## Summary

Phase 24 adds two surgical enhancements to the existing CLI in `cmd/agenthub-cli/`:

1. **POLISH-01** (`--json` flag on list/status commands): Four commands — `agenthub list`, `agenthub web status`, `agenthub health`, and `agenthub daemon status` — currently emit human-readable tabwriter or key-value output. Each must gain a `--json` flag that switches to clean JSON with no interleaved plain text. The underlying data is already fully JSON-tagged via `daemon.SessionInfo`, `daemon.WebServerStatusResponse`, and `webserver.TailscaleHealth`. The daemon status subcommand does not yet exist.

2. **POLISH-02** (`agenthub settings` command): A new top-level command that reads current configuration values from the daemon (CLI path overrides via `GET /settings/cli-paths`) and system state (socket path, relay port, daemon reachability) and prints them in a human-readable key-value format. Read-only — no modifications.

The changes are confined to `cmd/agenthub-cli/main.go` and `cmd/agenthub-cli/cmd_daemon.go`. No internal package changes are needed for POLISH-01. POLISH-02 may need the daemon client to expose relay port and socket path accessors, but `GetCLIPaths()` and `GetRelayPort()` already exist on `DaemonClient`.

**Primary recommendation:** Use Go's `encoding/json` with `json.NewEncoder(out).Encode(v)` for JSON output. Parse `--json` with a simple `flag.NewFlagSet` per command, consistent with the existing arg-parsing style (no third-party flag library is used in this codebase). `agenthub daemon status` is a new subcommand that must be added to `cmdDaemon`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| POLISH-01 | All list/status commands support `--json` flag for machine-readable output | All four target commands are identified; underlying structs already have `json:` tags; `encoding/json` is already imported project-wide |
| POLISH-02 | User can view current settings from CLI (`agenthub settings`) | `DaemonClient.GetCLIPaths()` and `GetRelayPort()` already exist; `DefaultSocketPath()` returns the socket path; daemon reachability can be checked via `client.Health()` |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | stdlib | JSON encoding | Already used throughout the codebase; no new deps needed |
| `flag` | stdlib | Parsing `--json` flag per command | Consistent with Go stdlib patterns; no third-party flag lib exists in this project |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `text/tabwriter` | stdlib | Human-readable table output (existing) | Non-JSON mode only |
| `fmt` | stdlib | Key-value human output (existing) | Non-JSON mode only |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `flag.NewFlagSet` per command | `os.Args` slice scanning | `flag.NewFlagSet` is cleaner; but the existing codebase uses direct `args` slice scanning for subcommands — either approach is fine for a single boolean flag |
| `json.NewEncoder(out).Encode(v)` | `json.Marshal` + `fmt.Fprintf` | Encoder writes directly to `io.Writer` without intermediate buffer; preferred for testability |

**Installation:** No new packages needed.

---

## Architecture Patterns

### Existing Pattern: io.Writer Injection
All output-producing cmd functions take `out io.Writer` as a parameter. Tests pass `bytes.Buffer`; main passes `os.Stdout`. This pattern MUST be preserved for `--json` variants and the new `cmdSettings`.

```go
// Existing pattern (verified in main.go)
func cmdList(client *daemon.DaemonClient, out io.Writer) error { ... }
func cmdWebStatus(client *daemon.DaemonClient, out io.Writer) error { ... }
func cmdHealth(out io.Writer) error { ... }
```

### Existing Pattern: Flag Parsing
The codebase does NOT use a third-party CLI framework (cobra, urfave/cli, etc.). Commands receive raw `[]string` args. For the `--json` flag, use `flag.NewFlagSet`:

```go
// Pattern for adding --json to cmdList
func cmdList(client *daemon.DaemonClient, args []string, out io.Writer) error {
    fs := flag.NewFlagSet("list", flag.ContinueOnError)
    jsonOut := fs.Bool("json", false, "output as JSON")
    if err := fs.Parse(args); err != nil {
        return err
    }
    sessions, err := client.ListSessions()
    if err != nil {
        return fmt.Errorf("agenthub list: %w", err)
    }
    if *jsonOut {
        return json.NewEncoder(out).Encode(sessions)
    }
    // existing tabwriter output...
}
```

Note: `cmdList` currently takes no `args []string` parameter. Adding `--json` requires updating the signature and the call site in `main()`.

### New: daemon status Subcommand
`agenthub daemon status` does not exist. It must be added to `cmdDaemon` in `cmd_daemon.go`. The daemon client's `Health()` method (`GET /health`) is the right probe — it returns `{"status":"ok"}` when the daemon is running.

```go
case "status":
    return cmdDaemonStatus(client, out)
```

But `cmdDaemon` currently does NOT receive a `*daemon.DaemonClient` — it only receives `args []string` and `out io.Writer`. The `daemon status` subcommand needs client access. Two options:

**Option A:** Pass client into `cmdDaemon` (requires signature change and test update in `cmd_daemon_test.go`).

**Option B:** Have `main()` handle `daemon status` before dispatching to `cmdDaemon`, similar to how other commands work — but this breaks the daemon dispatch pattern since `daemon` is routed before `EnsureDaemon`.

**Option C (recommended):** Add a `clientFunc func() *daemon.DaemonClient` parameter to `cmdDaemon`, defaulting to nil for non-status subcommands. Or simpler: `cmdDaemon` accepts an optional `*daemon.DaemonClient` that is nil for install/uninstall/start/stop/run and non-nil only when the daemon is running.

Actually the cleanest approach given the existing code structure: handle `daemon status` in `main()` after the `EnsureDaemon` path. The `daemon` subcommand check in `main()` can be modified to fall through for `daemon status` specifically, since status requires a running daemon.

**Simplest implementation:** Move `daemon status` to the standard dispatch path in `main()`:

```go
// In main(), before the daemon-early-exit block:
if cmd == "daemon" && len(os.Args) > 2 && os.Args[2] == "status" {
    // fall through to EnsureDaemon + client setup
}
// Existing daemon block for install/uninstall/start/stop/run
if cmd == "daemon" { ... }
```

Then add `daemon status` as a case handled with the client. This avoids changing `cmdDaemon` signature.

### New: cmdSettings Function
```go
func cmdSettings(client *daemon.DaemonClient, out io.Writer) error {
    // Print socket path (known locally)
    // Print relay port (from daemon)
    // Print CLI path overrides (from daemon)
    // Print daemon reachability
}
```

Settings to display (from existing APIs):
- `socket-path`: `daemon.DefaultSocketPath()`
- `relay-port`: `client.GetRelayPort()`
- `cli-paths`: `client.GetCLIPaths()` — shows custom CLI executable overrides, empty map if none set

Human-readable output format (consistent with existing cmdHealth style):
```
socket-path  /Users/ken/Library/Application Support/agenthub/daemon.sock
relay-port   52341
cli-paths    (none)
```

Or with CLI path overrides set:
```
socket-path  /Users/ken/Library/Application Support/agenthub/daemon.sock
relay-port   52341
cli-paths    claude=/usr/local/bin/claude
             cursor=/opt/cursor/cursor
```

### Recommended Project Structure
No structural changes needed — all changes are in existing files:
```
cmd/agenthub-cli/
├── main.go          # modify cmdList, cmdWebStatus, cmdHealth signatures; add cmdSettings
├── cmd_daemon.go    # add "status" subcommand OR handle in main()
└── main_test.go     # add --json and cmdSettings tests
```

### Anti-Patterns to Avoid
- **Interleaved plain text with JSON**: The success criterion explicitly requires no interleaved text. When `--json` is set, ONLY valid JSON goes to stdout. Error messages go to stderr via `return fmt.Errorf(...)` (which main writes to stderr).
- **Changing existing human output format**: Non-`--json` output must remain unchanged. Tests like `TestCmdList_Empty` check for "ID" header — do not break these.
- **JSON to stderr**: All JSON output goes to `out` (stdout). Errors always go via `return error` (main writes to stderr).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON encoding | Custom string builder for JSON | `encoding/json` | Tag-based encoding handles escaping, null, nested types |
| Flag parsing | Manual `args[0] == "--json"` scanning | `flag.NewFlagSet` | Handles `--json=true`, `-json`, placement anywhere in args |

---

## Common Pitfalls

### Pitfall 1: Trailing newline in JSON output
**What goes wrong:** `json.Marshal` + `fmt.Fprintln` adds a `\n` after the JSON. `json.NewEncoder(out).Encode(v)` also adds exactly one `\n`. Both are jq-compatible.
**Why it happens:** Users often test with `agenthub list --json | jq .` — jq requires valid JSON, and a trailing newline is fine.
**How to avoid:** Use `json.NewEncoder(out).Encode(v)` — it adds exactly one trailing newline, consistent with jq expectations.

### Pitfall 2: Breaking existing tests when adding args parameter to cmdList
**What goes wrong:** `cmdList` currently takes no `args []string`. Adding it breaks `TestCmdList_Empty` and `TestCmdList_WithSessions` call sites.
**Why it happens:** Signature change without updating all callers.
**How to avoid:** Update all call sites in `main_test.go` at the same time as the signature change.

### Pitfall 3: daemon status requires EnsureDaemon first
**What goes wrong:** If `daemon status` is handled in the same early-exit block as `daemon run/install/uninstall`, the daemon client is never created.
**Why it happens:** The current `daemon` block exits before `EnsureDaemon` is called.
**How to avoid:** Handle `daemon status` in the normal dispatch path (after `EnsureDaemon`), not in the early `daemon` exit block.

### Pitfall 4: json.Encode on nil slice vs empty slice
**What goes wrong:** `client.ListSessions()` already guards against nil (returns `[]SessionInfo{}`), so `json.Encode` will produce `[]` not `null`. Verify this remains true.
**Why it happens:** `json.Marshal(nil)` produces `null`, not `[]`.
**How to avoid:** The existing `DaemonClient.ListSessions()` already handles this. No change needed there.

### Pitfall 5: TailscaleHealth field name casing in JSON
**What goes wrong:** `TailscaleHealth` has `json:"hasCerts"` (camelCase). The test in success criteria specifies `jq`-parseable output — camelCase is fine for jq, but make sure the struct is emitted directly, not remapped.
**How to avoid:** Pass `TailscaleHealth` directly to `json.NewEncoder(out).Encode(h)`. Do not re-wrap in a different struct.

---

## Code Examples

### JSON output for cmdList
```go
// Source: encoding/json stdlib + existing cmdList pattern
func cmdList(client *daemon.DaemonClient, args []string, out io.Writer) error {
    fs := flag.NewFlagSet("list", flag.ContinueOnError)
    jsonOut := fs.Bool("json", false, "output as JSON")
    if err := fs.Parse(args); err != nil {
        return err
    }
    sessions, err := client.ListSessions()
    if err != nil {
        return fmt.Errorf("agenthub list: %w", err)
    }
    if *jsonOut {
        return json.NewEncoder(out).Encode(sessions)
    }
    w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
    fmt.Fprintln(w, "ID\tNAME\tAGENT\tSTATUS")
    for _, s := range sessions {
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Name, s.CLI, s.State)
    }
    return w.Flush()
}
```

### JSON output for cmdHealth
```go
// Source: encoding/json stdlib + existing cmdHealth pattern
// TailscaleHealth already has json tags: installed, connected, hasCerts, ip, domain
func cmdHealth(args []string, out io.Writer) error {
    fs := flag.NewFlagSet("health", flag.ContinueOnError)
    jsonOut := fs.Bool("json", false, "output as JSON")
    if err := fs.Parse(args); err != nil {
        return err
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    h := webserver.CheckHealth(ctx)
    if *jsonOut {
        return json.NewEncoder(out).Encode(h)
    }
    fmt.Fprintf(out, "%-12s%v\n", "installed:", h.Installed)
    fmt.Fprintf(out, "%-12s%v\n", "connected:", h.Connected)
    fmt.Fprintf(out, "%-12s%v\n", "has-certs:", h.HasCerts)
    fmt.Fprintf(out, "%-12s%v\n", "ip:", h.IP)
    fmt.Fprintf(out, "%-12s%v\n", "domain:", h.Domain)
    return nil
}
```

### cmdSettings implementation sketch
```go
// Source: existing DaemonClient methods: GetCLIPaths(), GetRelayPort()
func cmdSettings(client *daemon.DaemonClient, out io.Writer) error {
    socketPath := daemon.DefaultSocketPath()
    fmt.Fprintf(out, "%-14s%s\n", "socket-path:", socketPath)

    port, err := client.GetRelayPort()
    if err != nil {
        fmt.Fprintf(out, "%-14s%s\n", "relay-port:", "(unavailable)")
    } else {
        fmt.Fprintf(out, "%-14s%d\n", "relay-port:", port)
    }

    paths, err := client.GetCLIPaths()
    if err != nil || len(paths) == 0 {
        fmt.Fprintf(out, "%-14s%s\n", "cli-paths:", "(none)")
    } else {
        first := true
        for name, path := range paths {
            if first {
                fmt.Fprintf(out, "%-14s%s=%s\n", "cli-paths:", name, path)
                first = false
            } else {
                fmt.Fprintf(out, "%-14s%s=%s\n", "", name, path)
            }
        }
    }
    return nil
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Human-only CLI output | `--json` flag for machine-readable mode | Phase 24 | Enables scripting, CI pipelines, shell composition |
| No settings introspection | `agenthub settings` read-only view | Phase 24 | Operators can verify config without opening GUI |

---

## Open Questions

1. **Should `agenthub daemon status` show service install state (launchd/systemd)?**
   - What we know: `kardianos/service` does not have a simple "is-installed" query; you'd have to attempt control operations to probe.
   - What's unclear: The success criteria says `agenthub daemon status --json` must work — but what fields should it contain?
   - Recommendation: Keep it simple — daemon status shows daemon process reachability (`{"running": true/false}`) via `client.Health()`. Service install state is out of scope for this phase.

2. **Does `agenthub health --json` need a different struct shape than TailscaleHealth?**
   - What we know: `TailscaleHealth` has `json:"hasCerts"` (camelCase). The success criteria only requires jq-parseable output.
   - Recommendation: Emit `TailscaleHealth` directly — camelCase JSON keys are standard and jq handles them fine.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib) |
| Config file | none — `go test ./...` |
| Quick run command | `cd /Users/ken/dev/agenthub && go test ./cmd/agenthub-cli/... -count=1` |
| Full suite command | `cd /Users/ken/dev/agenthub && go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| POLISH-01 | `agenthub list --json` emits valid JSON array | unit | `go test ./cmd/agenthub-cli/... -run TestCmdList_JSON -count=1` | ❌ Wave 0 |
| POLISH-01 | `agenthub web status --json` emits valid JSON object | unit | `go test ./cmd/agenthub-cli/... -run TestCmdWebStatus_JSON -count=1` | ❌ Wave 0 |
| POLISH-01 | `agenthub health --json` emits valid JSON object | unit | `go test ./cmd/agenthub-cli/... -run TestCmdHealth_JSON -count=1` | ❌ Wave 0 |
| POLISH-01 | `agenthub daemon status --json` emits valid JSON | unit | `go test ./cmd/agenthub-cli/... -run TestCmdDaemon_Status -count=1` | ❌ Wave 0 |
| POLISH-01 | JSON output is parseable (no interleaved text) | unit | `go test ./cmd/agenthub-cli/... -run TestJSON -count=1` | ❌ Wave 0 |
| POLISH-02 | `agenthub settings` prints socket-path, relay-port, cli-paths | unit | `go test ./cmd/agenthub-cli/... -run TestCmdSettings -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/agenthub-cli/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] Tests for `--json` flag on all four commands (in `main_test.go`)
- [ ] Tests for `cmdSettings` (in `main_test.go`)
- [ ] Tests for `cmdDaemon` `status` subcommand (in `cmd_daemon_test.go`)

*(No new framework install needed — all tests use Go stdlib `testing` package)*

---

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection — `cmd/agenthub-cli/main.go`, `main_test.go`, `cmd_daemon.go`, `cmd_daemon_test.go`
- `internal/daemon/types.go`, `client.go`, `engine.go`, `api.go`, `socket.go`
- `internal/webserver/tailscale.go`
- Go stdlib `encoding/json`, `flag` packages

### Secondary (MEDIUM confidence)
- Go `encoding/json` behavior (trailing newline from Encoder, null vs [] from nil slice) — verified by direct code reading of existing usage patterns

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only, no new deps, verified against go.mod
- Architecture: HIGH — entire cmd package read; all relevant types and call sites identified
- Pitfalls: HIGH — identified from direct code reading of affected signatures and test patterns

**Research date:** 2026-03-24
**Valid until:** 2026-04-24 (stable codebase, no fast-moving dependencies)
