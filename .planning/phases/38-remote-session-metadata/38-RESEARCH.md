# Phase 38: Remote Session Metadata - Research

**Researched:** 2026-04-01
**Domain:** Go daemon API — session metadata struct, `os.Hostname()`, test patterns
**Confidence:** HIGH

## Summary

Phase 38 is a focused, contained change to the Go daemon layer. The goal is to expose the machine hostname in every `SessionInfo` object returned by `GET /api/sessions` and `GET /api/sessions/{id}`. The hostname must be captured once at daemon startup via `os.Hostname()` and injected into `SessionInfo` when the engine builds the list. No frontend work is in scope.

The codebase is already well-structured for this change. `SessionInfo` in `internal/daemon/types.go` is a flat struct with JSON tags — adding a `Hostname` field is a one-line change. The engine's `ListSessions()` method in `internal/daemon/engine.go` builds each `SessionInfo` inline — the hostname value simply needs to be available there (either stored on the engine or looked up once).

The existing test infrastructure in `internal/daemon/api_test.go` and `internal/daemon/engine_test.go` already tests the `GET /sessions` path with JSON decode/field assertions; the new test follows the same table-driven or inline assertion pattern already used there.

**Primary recommendation:** Add `Hostname string` to `SessionInfo`, capture `os.Hostname()` once in `NewSessionEngine()` (stored as `engine.hostname`), populate it in `ListSessions()`, and add a single test asserting the field is non-empty in both the engine snapshot and the HTTP response.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RMTE-03 | Session metadata from daemon includes machine hostname (`os.Hostname()`) for remote identification | `SessionInfo` struct + `engine.ListSessions()` is the exact insertion point; `os.Hostname()` is stdlib, no deps |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os` (stdlib) | Go 1.26 | `os.Hostname()` — returns FQDN or short name depending on OS config | Standard library, cross-platform, no deps |
| `encoding/json` (stdlib) | Go 1.26 | JSON serialization of API responses (already in use) | Already used throughout `daemon` package |
| `testing` (stdlib) | Go 1.26 | Unit + HTTP integration tests (already in use) | All daemon tests use standard `testing` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| None | — | No third-party libraries needed | This phase is pure stdlib + existing project code |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `os.Hostname()` once at startup | Call `os.Hostname()` per request | Startup capture is correct: hostname is constant for lifetime of daemon; per-request would be wasted syscalls |
| Store hostname on engine | Pass hostname as parameter to `NewAPI` | Either works; storing on engine keeps the change co-located with other session metadata |

**Installation:** No new packages — stdlib only.

**Version verification:** `os.Hostname()` is part of Go 1 compatibility guarantee. No version concern.

## Architecture Patterns

### Recommended Change Structure
```
internal/daemon/
├── types.go      # Add Hostname field to SessionInfo
├── engine.go     # Store hostname at init, populate in ListSessions()
├── engine_test.go  # Add TestEngineListSessionsHostname
└── api_test.go   # Add TestAPIListSessionsHostname (HTTP JSON assertion)
```

No new files needed. All changes are additive to existing files.

### Pattern 1: Capture Once at Startup

**What:** Call `os.Hostname()` in `NewSessionEngine()`, store result in a `hostname string` field on the engine. Ignore error (return empty string on failure; tests assert non-empty so they will catch this on CI).

**When to use:** The hostname is constant for the process lifetime. Capturing once avoids per-call syscall overhead and keeps the value stable in tests.

**Example:**
```go
// internal/daemon/engine.go — NewSessionEngine
func NewSessionEngine() *SessionEngine {
    hostname, _ := os.Hostname()
    return &SessionEngine{
        hostname:        hostname,
        registry:        pty.NewSessionRegistry(),
        // ...existing fields...
    }
}
```

### Pattern 2: Populate Hostname in ListSessions

**What:** In `ListSessions()`, set `Hostname: e.hostname` on each `SessionInfo` built in the loop. The value is identical for every session — it is the host running the daemon, not something session-specific.

**When to use:** Always. Every session originates from the same machine; the field belongs on each item so consumers don't need a separate lookup.

**Example:**
```go
// internal/daemon/engine.go — ListSessions
result = append(result, SessionInfo{
    ID:        s.ID,
    CLI:       s.CLI,
    Name:      name,
    State:     state,
    CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
    Hostname:  e.hostname,
})
```

### Pattern 3: Existing Test Pattern (follow exactly)

**What:** The existing tests in `api_test.go` use `rawGet` + `json.Unmarshal` into typed structs, then assert fields. New hostname tests follow the same pattern.

**Example (modeled on `TestAPIGetSession`):**
```go
func TestAPIListSessionsHostname(t *testing.T) {
    _, _, socketPath := testDaemon(t)
    // Create a session so the list is non-empty
    rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"h-test","workDir":""}`)

    _, body := rawGet(t, socketPath, "/sessions")
    var sessions []SessionInfo
    if err := json.Unmarshal(body, &sessions); err != nil {
        t.Fatalf("decode sessions: %v", err)
    }
    if len(sessions) == 0 {
        t.Fatal("expected at least 1 session")
    }
    if sessions[0].Hostname == "" {
        t.Error("SessionInfo.Hostname is empty — want non-empty hostname")
    }
}
```

And the parallel engine test (modeled on `TestEngineListSessions`):
```go
func TestEngineListSessionsHostname(t *testing.T) {
    e := NewSessionEngine()
    id, err := e.CreateSession(context.Background(), "cat", "h-eng", "", nil, 0, 0, nil)
    if err != nil {
        t.Fatalf("CreateSession: %v", err)
    }
    t.Cleanup(func() { _ = e.KillSession(id) })

    sessions := e.ListSessions()
    if len(sessions) == 0 {
        t.Fatal("expected 1 session")
    }
    if sessions[0].Hostname == "" {
        t.Error("SessionInfo.Hostname empty — os.Hostname() must have failed or field not populated")
    }
}
```

### Anti-Patterns to Avoid

- **Do not call `os.Hostname()` every request:** Unnecessary syscall; hostname is stable.
- **Do not make Hostname a pointer (`*string`):** The field is always present; use a plain `string` consistent with other `SessionInfo` fields.
- **Do not add a new daemon API endpoint for hostname:** The requirement is to include it inline in session metadata, not as a separate `/hostname` endpoint.
- **Do not store hostname in the `SessionInfo` registry entry (`pty.Session`):** Hostname is infrastructure metadata from the daemon layer, not PTY session data. Keep it in the daemon engine layer.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Machine hostname | Custom `/etc/hostname` parsing, shell `hostname` command | `os.Hostname()` | stdlib cross-platform function; handles macOS, Linux, Windows correctly |

**Key insight:** `os.Hostname()` is a thin wrapper around `gethostname()` (POSIX) / `GetComputerName()` (Windows). It is the canonical Go approach; no alternative is needed.

## Common Pitfalls

### Pitfall 1: os.Hostname() Returns an Error
**What goes wrong:** On some systems (container without hostname set, unusual configurations), `os.Hostname()` can return an error. Silently ignoring with `_` means `Hostname` is an empty string, which would fail the test assertion "non-empty".
**Why it happens:** The function signature is `func Hostname() (name string, err error)`.
**How to avoid:** The test requirement says "non-empty" — this is correct behavior for CI and real machines. The daemon runs on a real host; `os.Hostname()` will succeed. However, if engine construction should be defensive, consider logging the error (not fatal). For this codebase, silent `_` discard is the existing pattern (see `cmd_daemon.go` signal handlers).
**Warning signs:** Empty hostname in test output indicates the daemon process has no hostname set (check the test environment).

### Pitfall 2: Forgetting to Update the DaemonClient
**What goes wrong:** `DaemonClient.ListSessions()` decodes into `[]SessionInfo`. If `SessionInfo` gains `Hostname`, the client automatically deserializes it with no code change needed — JSON unmarshaling ignores unknown fields by default, and known fields are populated automatically. No client code change is required.
**Why it happens:** Developers sometimes add a separate client method or response type.
**How to avoid:** Use the shared `SessionInfo` type — the client already decodes `GET /sessions` into `[]SessionInfo`. The field appears automatically.

### Pitfall 3: Breaking GET /sessions/{id}
**What goes wrong:** `handleGetSession` iterates `ListSessions()` and returns the matching `SessionInfo` directly. Because `ListSessions()` will now populate `Hostname`, the single-session endpoint also gains the field automatically — no separate code change needed.
**Why it happens:** Developers add the field to `ListSessions` but forget the single-session path.
**How to avoid:** Confirm both endpoints in tests or by code inspection. Since both use `ListSessions()` as the data source, a single change covers both.

## Code Examples

### os.Hostname() — Standard Usage
```go
// Source: Go stdlib os package documentation
hostname, err := os.Hostname()
if err != nil {
    // handle: extremely rare on real machines
    hostname = ""
}
// hostname is the short name (e.g., "macbook-pro") or FQDN depending on OS config
```

### Modified SessionInfo struct
```go
// internal/daemon/types.go
type SessionInfo struct {
    ID        string `json:"id"`
    CLI       string `json:"cli"`
    Name      string `json:"name"`
    State     string `json:"state"`
    CreatedAt string `json:"createdAt"`
    Hostname  string `json:"hostname"`  // ADD THIS LINE
}
```

### Modified SessionEngine struct
```go
// internal/daemon/engine.go
type SessionEngine struct {
    hostname string          // machine hostname, captured at startup — ADD THIS FIELD
    registry *pty.SessionRegistry
    backend  pty.SessionBackend
    manager  *relay.HubManager
    // ...existing fields unchanged...
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A — new field | Add `hostname` to `SessionInfo` | Phase 38 | Downstream: Phase 39 web status bar and CLI attach banner can read `hostname` from session list without a separate API call |

**Deprecated/outdated:** None — this is a net-new addition.

## Open Questions

1. **FQDN vs short hostname**
   - What we know: `os.Hostname()` returns the short name (e.g., `macbook-pro`) on macOS by default; a FQDN (e.g., `macbook-pro.local` or `macbook-pro.tail46d69a.ts.net`) is available via `net.LookupAddr` or Tailscale metadata
   - What's unclear: Phase 39 shows `macbook-pro.local` in the example display string — is `os.Hostname()` enough or do we need Tailscale FQDN?
   - Recommendation: Use `os.Hostname()` for Phase 38 as the requirement specifies it explicitly. Phase 39 can enrich or transform the value at display time if needed. Do not change the requirement for Phase 38.

## Environment Availability

Step 2.6: SKIPPED — Phase 38 is pure Go stdlib code change with no external tool dependencies beyond the existing Go toolchain (Go 1.26 confirmed present).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` stdlib |
| Config file | none — standard `go test` |
| Quick run command | `go test ./internal/daemon/... -run TestAPIListSessionsHostname\|TestEngineListSessionsHostname` |
| Full suite command | `go test ./internal/daemon/...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RMTE-03 | `GET /sessions` response includes non-empty `hostname` field | integration (HTTP) | `go test ./internal/daemon/... -run TestAPIListSessionsHostname` | Wave 0 — add to `api_test.go` |
| RMTE-03 | `SessionEngine.ListSessions()` populates `Hostname` field | unit | `go test ./internal/daemon/... -run TestEngineListSessionsHostname` | Wave 0 — add to `engine_test.go` |

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon/... -run TestAPIListSessionsHostname\|TestEngineListSessionsHostname`
- **Per wave merge:** `go test ./internal/daemon/...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/daemon/api_test.go` — add `TestAPIListSessionsHostname` (inline, no new file needed)
- [ ] `internal/daemon/engine_test.go` — add `TestEngineListSessionsHostname` (inline, no new file needed)

*(No new test files or framework setup needed — both go into existing test files)*

## Project Constraints (from CLAUDE.md)

| Directive | Application to this phase |
|-----------|--------------------------|
| Go: `go fmt`, `golangci-lint`, context-aware functions | Run `go fmt ./internal/daemon/...` after changes; new engine field requires no context |
| Node package manager: `pnpm` preferred | Not applicable — Go only phase |
| Python venv: never install globally | Not applicable — Go only phase |
| 80%+ coverage in critical components | Two new tests cover the new field; existing test count increases |
| LSP over Grep for code navigation | Use LSP to navigate to `SessionInfo`, `ListSessions`, `NewSessionEngine` definitions |
| Chesterton's Fence: articulate before removing | No removals in this phase |

## Sources

### Primary (HIGH confidence)
- Go stdlib `os` package — `os.Hostname()` cross-platform behavior (built-in knowledge, Go 1 compat guarantee, confirmed Go 1.26 in environment)
- `/Users/ken/dev/agenthub/internal/daemon/types.go` — `SessionInfo` struct definition (direct read)
- `/Users/ken/dev/agenthub/internal/daemon/engine.go` — `ListSessions()`, `NewSessionEngine()` implementation (direct read)
- `/Users/ken/dev/agenthub/internal/daemon/api.go` — route handlers, `handleListSessions` (direct read)
- `/Users/ken/dev/agenthub/internal/daemon/api_test.go` — existing test patterns (direct read)
- `/Users/ken/dev/agenthub/internal/daemon/engine_test.go` — existing engine test patterns (direct read)

### Secondary (MEDIUM confidence)
- REQUIREMENTS.md `RMTE-03` — explicit requirement for `os.Hostname()` (direct read)
- ROADMAP.md Phase 38 success criteria — "Go tests verify the hostname field is present and non-empty" (direct read)

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only, no uncertainty
- Architecture: HIGH — direct code inspection of insertion points; change is mechanical
- Pitfalls: HIGH — identified from code structure inspection, not speculation

**Research date:** 2026-04-01
**Valid until:** Stable indefinitely — stdlib `os.Hostname()` is Go 1 guaranteed; internal code structure only changes if engine.go is refactored
