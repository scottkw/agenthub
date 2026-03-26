# Phase 30: Backend Args Wiring - Research

**Researched:** 2026-03-25
**Domain:** Go daemon IPC chain — types, engine, API handler, HTTP client, Wails app binding
**Confidence:** HIGH

## Summary

The gap is fully understood from reading the code. `pty.CreateRequest.Args []string` already exists and is correctly forwarded to `gopty.Cmd` in `NativePTYBackend.Create` (line 41 of `native.go`). Every layer *above* that drops args on the floor:

1. `daemon.CreateRequest` (types.go) — no `Args` field
2. `SessionEngine.CreateSession` (engine.go) — signature `(ctx, cli, name, workDir string)`, no args parameter
3. `API.handleCreateSession` (api.go) — decodes `CreateRequest`, calls `engine.CreateSession` without args
4. `DaemonClient.CreateSession` (client.go) — signature `(cli, name, workDir string)`, no args parameter
5. `App.CreateSession` (app.go) — Wails-bound method `(cli, name, workDir string)`, delegates to client

The fix is mechanical and additive at each layer. No existing logic changes — only new plumbing.

**Primary recommendation:** Add `Args []string` to `daemon.CreateRequest`, thread it through all five layers in order, update existing callers with zero-value nil/empty slices so behavior is unchanged, then add a table-driven integration test that verifies the full IPC chain with a non-empty args slice.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ARGS-03 | Args propagate through daemon layers (types → engine → API → client → PTY) | Full gap map identified; each of the 5 layers documented with exact file/line changes needed |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | stdlib | HTTP JSON serialisation/deserialisation | Already used throughout; `Args []string` round-trips as `["--flag","value"]` |
| `github.com/aymanbagabas/go-pty` | already in go.mod | PTY process spawn with `Args...` spread | Already handles args — bottom layer is done |

### Supporting
None — this phase is pure plumbing inside existing packages, no new dependencies.

**Installation:** No new packages required.

## Architecture Patterns

### Full IPC chain (current state vs. target)

```
Caller                     Gap              Target
------                     ---              ------
App.CreateSession          no args param    CreateSession(cli, name, workDir string, args []string)
  └─ DaemonClient.CreateSession   no args  CreateSession(cli, name, workDir string, args []string)
       └─ HTTP POST /sessions               body: {"cli","name","workDir","args":[...]}
            └─ handleCreateSession          decode CreateRequest{Args}
                 └─ engine.CreateSession    no args param → CreateSession(ctx, cli, name, workDir string, args []string)
                      └─ pty.CreateRequest  ALREADY HAS Args []string ✓
                           └─ gopty.Cmd     req.Args... spread ✓
```

### Pattern: Additive, backward-compatible signature extension

Each layer gets `args []string` appended to its signature. All existing callers (GUI passing nil/empty, CLI passing nil/empty) compile without change.

**Example — engine.CreateSession signature change:**
```go
// Before
func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string, onStatus func(string, status.SessionStatus)) (string, error)

// After
func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string, args []string, onStatus func(string, status.SessionStatus)) (string, error)
```

**Example — pty.CreateRequest construction in engine.go (the critical wire):**
```go
// Before
sess, err := e.backend.Create(ctx, pty.CreateRequest{
    CLI:     cliPath,
    Cols:    80,
    Rows:    24,
    WorkDir: workDir,
})

// After
sess, err := e.backend.Create(ctx, pty.CreateRequest{
    CLI:     cliPath,
    Args:    args,
    Cols:    80,
    Rows:    24,
    WorkDir: workDir,
})
```

**Example — daemon.CreateRequest type addition:**
```go
// types.go — add Args field
type CreateRequest struct {
    CLI     string   `json:"cli"`
    Name    string   `json:"name"`
    WorkDir string   `json:"workDir"`
    Args    []string `json:"args,omitempty"`
}
```

`omitempty` on a `[]string` emits `null`/absent when empty — harmless at the receiving end since `nil` and `[]string{}` are equivalent in Go. Alternatively, omit `omitempty` to always emit `"args":[]` — either is correct, `omitempty` is slightly cleaner.

**Example — DaemonClient.CreateSession:**
```go
// Before
func (c *DaemonClient) CreateSession(cli, name, workDir string) (string, error) {
    req := CreateRequest{CLI: cli, Name: name, WorkDir: workDir}

// After
func (c *DaemonClient) CreateSession(cli, name, workDir string, args []string) (string, error) {
    req := CreateRequest{CLI: cli, Name: name, WorkDir: workDir, Args: args}
```

**Example — App.CreateSession (Wails binding):**
```go
// Before
func (a *App) CreateSession(cli, name, workDir string) (string, error) {
    id, err := a.client.CreateSession(cli, name, workDir)

// After
func (a *App) CreateSession(cli, name, workDir string, args []string) (string, error) {
    id, err := a.client.CreateSession(cli, name, workDir, args)
```

### Nil/empty args safety

`gopty.Cmd` receives `req.Args...`. When `args` is nil or empty, this spreads zero arguments — identical to the current behaviour. No guard needed.

### Anti-Patterns to Avoid

- **Converting nil to empty slice in every layer:** Unnecessary. `nil` and `[]string{}` both spread to zero arguments. Only convert once, at the PTY layer if the backend ever stores args.
- **Adding args to `SessionInfo`:** Out of scope. Args are write-once at creation time, not needed for listing.
- **Parsing/splitting the args string in this phase:** ARGS-06 (deferred). Phase 30 only wires an already-split `[]string`. Phase 31 (CLI) and 33 (GUI) handle parsing.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON round-trip for `[]string` | Custom serialiser | stdlib `encoding/json` | `[]string` marshals to `["a","b"]` natively |
| Process arg forwarding | Manual `exec.Command` construction | `gopty.Cmd` spread `req.Args...` | Already done at PTY layer |

## Common Pitfalls

### Pitfall 1: Forgetting the `app.go` Wails binding signature
**What goes wrong:** `DaemonClient.CreateSession` gets args, but `App.CreateSession` does not — the Wails TypeScript binding still has the old 3-argument signature, so the frontend can never pass args.
**Why it happens:** `app.go` is a thin shell and easy to forget when following the internal chain.
**How to avoid:** Success criteria explicitly require the Wails-bound method to accept args.
**Warning signs:** Compilation succeeds but `App.CreateSession` still has 3 parameters.

### Pitfall 2: Existing test callers fail to compile
**What goes wrong:** `engine_test.go` and `client_test.go` both call `CreateSession` without args — adding a required parameter breaks compilation.
**Why it happens:** Go function signatures are strict; no default parameters.
**How to avoid:** All existing call sites must be updated to pass `nil` as the args argument.
**Warning signs:** `go build ./...` fails immediately after signature change.

### Pitfall 3: `handleCreateSession` not forwarding args to engine
**What goes wrong:** `daemon.CreateRequest.Args` is decoded correctly but the `handleCreateSession` handler calls `engine.CreateSession(..., nil)` by mistake.
**Why it happens:** It is the most error-prone hand-off — the decoded `req.Args` must be explicitly threaded.
**How to avoid:** The integration test (client → API → engine → PTY) will catch this by verifying the spawned process received the expected argument.

### Pitfall 4: JSON `null` vs `[]` for empty args
**What goes wrong:** `omitempty` on `[]string` causes the field to be absent when nil — a strict JSON decoder on a future receiver might reject missing field.
**Why it happens:** Go's `omitempty` treats nil slice as zero value.
**How to avoid:** Both `null` and absent field decode to `nil` in Go; current client uses `json.NewDecoder` which is lenient. `omitempty` is fine. Document this decision.

## Code Examples

Verified from reading source files in this repo:

### PTY layer — already wires args (no change needed)
```go
// Source: internal/pty/native.go:41
cmd := p.CommandContext(childCtx, req.CLI, req.Args...)
```

### Engine call site — must wire args into CreateRequest
```go
// Source: internal/daemon/engine.go:48 (current, before fix)
sess, err := e.backend.Create(ctx, pty.CreateRequest{
    CLI:     cliPath,
    Cols:    80,
    Rows:    24,
    WorkDir: workDir,
})
```

### Test helper pattern — existing testDaemon fixture
```go
// Source: internal/daemon/api_test.go:17
func testDaemon(t *testing.T) (*API, *DaemonClient, string) {
    engine := NewSessionEngine()
    api := NewAPI(engine)
    socketPath := shortSocketPath(t, "api.sock")
    api.Start(socketPath)
    t.Cleanup(func() { api.Stop() })
    client := NewDaemonClient(socketPath)
    time.Sleep(10 * time.Millisecond)
    return api, client, socketPath
}
```

The new args test should use `testDaemon` as its foundation — no new infrastructure needed.

### Verifying args reach the process — test strategy
Use `cat` as a stand-in (already the convention in existing tests). To verify args are forwarded, use `/bin/sh -c 'printf "%s\n" "$@" -- "$@"'` pattern, or more directly: spawn `echo` with known args and read the PTY output. The simplest reliable approach is:

```go
// Spawn "cat" with args "--foo" "bar" and verify the PTY process command
// can be inspected, OR spawn "sh" with "-c" "echo $0 $@" -- "testarg"
// and read a line from the session to confirm.
```

Simpler and more robust: use the existing `testDaemon` helper and call `client.CreateSession("cat", "test", "", []string{"--extra"})` — the test only needs to verify that:
1. No error is returned (args are accepted at every layer)
2. The session ID is non-empty (session was created successfully)

A deeper test (verifying args actually reached the process) can be done by spawning `sh` with args that write to stdout and reading PTY output, but the phase success criteria only require IPC chain coverage, not byte-level verification.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./internal/daemon/... -run TestArgs -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ARGS-03 | `daemon.CreateRequest` JSON includes `Args` field | unit | `go test ./internal/daemon/... -run TestArgsRoundTrip -v` | No — Wave 0 |
| ARGS-03 | Session created with args doesn't error through full IPC chain | integration | `go test ./internal/daemon/... -run TestClientCreateSessionWithArgs -v` | No — Wave 0 |
| ARGS-03 | Engine.CreateSession with args forwards to pty.CreateRequest | unit | `go test ./internal/daemon/... -run TestEngineCreateSessionWithArgs -v` | No — Wave 0 |
| ARGS-03 | Existing callers with no args continue to work | regression | `go test ./...` | Existing tests cover this after callers are updated |

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon/... -v`
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/daemon/engine_test.go` — add `TestEngineCreateSessionWithArgs` verifying args reach pty.CreateRequest (via a custom mock backend or by observing no-error with `cat`)
- [ ] `internal/daemon/client_test.go` — add `TestClientCreateSessionWithArgs` using `testDaemon` helper, calls `client.CreateSession("cat","t","",[]string{"--extra"})`, asserts non-empty ID

*(No new test files required — add functions to existing test files)*

## Environment Availability

Step 2.6: SKIPPED — this phase is pure Go source code changes with no external dependencies beyond what is already in go.mod.

## Sources

### Primary (HIGH confidence)
- `/Users/ken/dev/agenthub/internal/pty/backend.go` — `CreateRequest.Args []string` confirmed present
- `/Users/ken/dev/agenthub/internal/pty/native.go` — `req.Args...` confirmed forwarded to `gopty.Cmd`
- `/Users/ken/dev/agenthub/internal/daemon/types.go` — `CreateRequest` confirmed missing `Args` field
- `/Users/ken/dev/agenthub/internal/daemon/engine.go` — `CreateSession` confirmed missing args parameter
- `/Users/ken/dev/agenthub/internal/daemon/api.go` — `handleCreateSession` confirmed passes only `req.CLI, req.Name, req.WorkDir`
- `/Users/ken/dev/agenthub/internal/daemon/client.go` — `CreateSession` confirmed missing args parameter
- `/Users/ken/dev/agenthub/app.go` — `App.CreateSession` confirmed missing args parameter
- `/Users/ken/dev/agenthub/.planning/STATE.md` — decision confirmed: "gap is in the 5 layers above it"

## Metadata

**Confidence breakdown:**
- Gap identification: HIGH — read every file in the IPC chain directly
- Fix approach: HIGH — additive, no ambiguity in Go's type system
- Test strategy: HIGH — existing `testDaemon` and `cat` conventions are proven in the codebase
- Wails binding implication: HIGH — `App.CreateSession` is the Wails-bound surface, must include args for frontend to use them

**Research date:** 2026-03-25
**Valid until:** 2026-06-25 (stable — pure Go stdlib patterns, no third-party library decisions)
