---
phase: 100-shell-session-backend-discovery
plan: 04
subsystem: internal/daemon
tags: [shell-discovery, http-api, daemon-client, SHELL-04, SHELL-05, SHELL-09, tdd, wave-2]
requires:
  - "Plan 100-01: pty.DiscoverShells + DetectedShell (canonical, on main)"
  - "Plan 100-02: engine shell-spawn dispatch + status.Watch bypass (canonical, on main)"
provides:
  - "HTTP route: GET /shells → ShellsResponse"
  - "wire types: daemon.DetectedShell, daemon.ShellsResponse"
  - "client method: (*DaemonClient).ListShells() ([]DetectedShell, error)"
  - "SHELL-04 satisfied at the wire layer (Phase 101 GUI/CLI/TUI surfaces consume this)"
  - "SHELL-09 end-to-end verification (status only running→stopped across full lifecycle)"
affects:
  - "Phase 101 (GUI/CLI/TUI shell pickers): now have a daemon API to enumerate shells"
  - "Phase 100 closeout: all three requirements (SHELL-04 + SHELL-05 + SHELL-09) covered"
tech-stack:
  added: []
  patterns:
    - "Read-only HTTP handler mirroring handleGetCLIPaths (api.go:478)"
    - "DaemonClient method mirroring GetCLIPaths (client.go:97)"
    - "Wire-type duplication of pty.DetectedShell into daemon.DetectedShell (mirrors SessionInfo:pty.Session pattern)"
    - "Non-null empty-slice JSON guarantee via make([]T, 0, n)"
    - "Defensive Argv copy in handler so callers cannot mutate package-level knownShellSpecs"
    - "RED→GREEN TDD split with intentional `client.ListShells undefined` build failure as RED gate"
key-files:
  created: []
  modified:
    - internal/daemon/types.go
    - internal/daemon/api.go
    - internal/daemon/api_test.go
    - internal/daemon/client.go
decisions:
  - "Wire-type duplication (not pty.DetectedShell embed): daemon-side DetectedShell is a separate struct with JSON tags, mirroring the SessionInfo:pty.Session pattern. Keeps wire DTOs decoupled from internal Go API."
  - "RED gate anchored on `client.ListShells undefined` build failure: a 404 from a missing route is a runtime failure (would look green at compile time), so Task 1a's RED is asserted at the compiler via three call sites to the not-yet-existing DaemonClient.ListShells method."
  - "Defensive Argv copy in handleListShells: `append([]string(nil), s.Argv...)` so callers cannot mutate pty's package-level knownShellSpecs table through the response (T-100-09 mitigation)."
  - "Non-null empty-slice JSON guarantee: handler uses `make([]DetectedShell, 0, len(...))` so empty discovery serialises as `{\"shells\":[]}`, not `{\"shells\":null}` — locked by `bytes.Contains(body, []byte(\"\\\"shells\\\":[]\"))` assertion in TestHandleListShells_EmptyPATH."
  - "Lifecycle test interprets `s.Status == \"stopped\"` pragmatically: the engine's KillSession removes the session from the registry entirely, and the heuristic-status default is `\"running\"` (no watcher writes for shells per Plan 02). Test asserts SHELL-09's actual spirit: across the full lifecycle, Status only ever takes values in {running, stopped} and never appears in {waiting, error, errored, idle}. Termination is detected via either State==stopped or removal-from-registry."
metrics:
  duration: "~3min"
  completed: "2026-05-12"
  tasks: 2
  files_modified: 4
  commits: 2
---

# Phase 100 Plan 04: GET /shells + DaemonClient.ListShells + SHELL-09 lifecycle — Summary

Closes Phase 100 by exposing Plan 01's `pty.DiscoverShells()` over the daemon's
HTTP API as `GET /shells`, adding a corresponding `DaemonClient.ListShells()`
client method, and adding an integration test that exercises the full shell
session lifecycle (create → list → kill) to prove SHELL-09 end-to-end on top of
Plan 02's unit-level guards.

## Wire surface

### GET /shells (read-only)

Route registered at `internal/daemon/api.go:69`:

```go
a.mux.HandleFunc("GET /shells", a.handleListShells)
```

Handler (`internal/daemon/api.go:478-498`):

```go
func (a *API) handleListShells(w http.ResponseWriter, r *http.Request) {
    discovered := pty.DiscoverShells()
    out := make([]DetectedShell, 0, len(discovered))
    for _, s := range discovered {
        out = append(out, DetectedShell{
            Name:        s.Name,
            DisplayName: s.DisplayName,
            Path:        s.Path,
            Argv:        append([]string(nil), s.Argv...),
        })
    }
    writeJSON(w, http.StatusOK, ShellsResponse{Shells: out})
}
```

Response JSON shape (populated case):

```json
{
  "shells": [
    {"name": "bash",       "displayName": "bash",       "path": "/bin/bash",  "argv": ["-i"]},
    {"name": "zsh",        "displayName": "zsh",        "path": "/bin/zsh",   "argv": ["-i"]},
    {"name": "powershell", "displayName": "Windows PowerShell", "path": "/usr/local/bin/powershell", "argv": ["-NoLogo"]}
  ]
}
```

Empty case (deterministic non-null array):

```json
{"shells": []}
```

The `"shells":null` shape is impossible — verified at the byte level in
`TestHandleListShells_EmptyPATH` via `bytes.Contains(body, []byte("\"shells\":[]"))`
and `!bytes.Contains(body, []byte("\"shells\":null"))`.

### DaemonClient.ListShells (`internal/daemon/client.go:107-117`)

```go
func (c *DaemonClient) ListShells() ([]DetectedShell, error) {
    var resp ShellsResponse
    if err := c.doJSON(http.MethodGet, "/shells", nil, &resp); err != nil {
        return nil, err
    }
    if resp.Shells == nil {
        resp.Shells = []DetectedShell{}
    }
    return resp.Shells, nil
}
```

The nil-guard mirrors `ListSessions`'s defensive `if sessions == nil { sessions = []SessionInfo{} }`
pattern at `client.go:60-62` so client callers never need to nil-check the result.

## New wire types (`internal/daemon/types.go`)

Placed directly after `CLIPathsResponse` at the existing wire-type cluster:

```go
type DetectedShell struct {
    Name        string   `json:"name"`
    DisplayName string   `json:"displayName"`
    Path        string   `json:"path"`
    Argv        []string `json:"argv"`
}

type ShellsResponse struct {
    Shells []DetectedShell `json:"shells"`
}
```

The duplication of `pty.DetectedShell`'s fields (rather than embedding) mirrors
the established `SessionInfo` ↔ `pty.Session` pattern at `types.go:4-16` — wire
DTOs stay decoupled from internal Go API surfaces.

## Test coverage map (3 new tests)

| Test | Requirement | What it locks |
|------|-------------|---------------|
| `TestHandleListShells_EmptyPATH` | SHELL-04 (wire format, non-null empty-array guarantee) | With `PATH=t.TempDir()` and `SHELL=""` (relies on Plan 01 H4 contract suppressing the synthetic entry), GET /shells returns 200 with body containing literal `"shells":[]` and never `"shells":null`. `client.ListShells()` round-trips a non-nil zero-length slice. |
| `TestHandleListShells_PopulatedPATH` | SHELL-04 (dev-host smoke) | On the real host PATH, at least one shell is discovered and every entry has non-empty Name + Path + DisplayName + len(Argv) ≥ 1. At least one entry's Name is in {bash, zsh, pwsh, powershell, shell} (defends against schema drift). Client round-trip returns same length. |
| `TestShellSessionLifecycle_StatusOnlyRunningOrStopped` | SHELL-09 (end-to-end) | POSIX-only; skipped if bash absent from PATH. Creates a bash session via `client.CreateSession`. Polls 5 times over ~1s while alive — asserts Status is `"running"` on every sample (never waiting/error/errored/idle). Calls `client.KillSession`, polls up to 2s — asserts Status stays in {running, stopped} and the session eventually either disappears from the registry or transitions to State="stopped". Finally re-calls `client.ListShells()` to confirm the lifecycle did not corrupt engine state. |

All 3 tests pass under `-race`:

```
=== RUN   TestHandleListShells_EmptyPATH
--- PASS: TestHandleListShells_EmptyPATH (0.02s)
=== RUN   TestHandleListShells_PopulatedPATH
--- PASS: TestHandleListShells_PopulatedPATH (0.02s)
=== RUN   TestShellSessionLifecycle_StatusOnlyRunningOrStopped
--- PASS: TestShellSessionLifecycle_StatusOnlyRunningOrStopped (1.14s)
PASS
ok  github.com/scottkw/agenthub/internal/daemon  2.220s
```

## TDD gate compliance (RED → GREEN)

Per Plan 04 H3 split — the RED phase intentionally leaves a build-failing
commit so the failing-test signal is preserved as a separate snapshot in
`git log`.

| Gate  | Commit  | Subject |
|-------|---------|---------|
| RED   | `2de44f9` | `test(100-04): add ShellsResponse types + failing /shells tests (RED)` |
| GREEN | `6277456` | `feat(100-04): wire GET /shells + DaemonClient.ListShells (GREEN)` |

The RED gate is anchored on `client.ListShells undefined` — the test file
references the method at three call sites, so `go test -c ./internal/daemon`
fails with exactly that compiler error on commit `2de44f9`:

```
internal/daemon/api_test.go:1392:24: client.ListShells undefined (type *DaemonClient has no field or method ListShells)
internal/daemon/api_test.go:1456:30: client.ListShells undefined (type *DaemonClient has no field or method ListShells)
internal/daemon/api_test.go:1572:22: client.ListShells undefined (type *DaemonClient has no field or method ListShells)
```

This is the "RED-as-expected" signal asserted by the plan's
`<verify><automated>` block — chosen over the "route-404 at runtime" approach
because a 404 is a runtime failure that would mask "build green" as "tests
red," whereas an `undefined` compile error is unambiguously red.

Note: `go build ./internal/daemon/...` is clean even on the RED commit because
Go's `build` target does not compile `_test.go` files. The plan's RED verify
command was therefore executed against `go test -c` rather than `go build`.
This is identical to the documented Plan 01 RED-verify substitution (see
`100-01-SUMMARY.md` deviation note 1) — a Go build-tooling fact, not a plan
deviation.

## Phase 100 closeout — requirement traceability

| Req ID    | Plan(s) covering | Where verified |
|-----------|------------------|----------------|
| SHELL-04  | Plan 01 (library), Plan 04 (wire) | `internal/pty/shells_test.go` (TestDiscoverShells_*, TestKnownShellSpecs_HasExpectedEntries) + `internal/daemon/api_test.go` (TestHandleListShells_EmptyPATH, TestHandleListShells_PopulatedPATH) |
| SHELL-05  | Plan 02 (engine) | `internal/daemon/engine_test.go` (TestCreateSession_ShellArgv_Interactive, TestCreateSession_ZshArgv_Interactive, TestCreateSession_PwshArgv, TestCreateSession_ShellWorkDirHonored, TestCreateSession_ShellEmptyWorkDirHome) |
| SHELL-09  | Plan 02 (unit-level), Plan 04 (end-to-end) | `internal/daemon/engine_test.go` (TestCreateSession_ShellSkipsStatusWatch, TestListSessions_ShellStatusRunning, TestShell_NoStatusMapEntry) + `internal/daemon/api_test.go` (TestShellSessionLifecycle_StatusOnlyRunningOrStopped) |

Plan 03 (Windows pwsh PATH augmentation) is a cross-compile-only deliverable
on `internal/daemon/path_windows.go`; verified in Plan 03's SUMMARY.

## Verification results

```
$ go test ./internal/daemon -run 'TestHandleListShells|TestShellSessionLifecycle' -race -count=1
ok  github.com/scottkw/agenthub/internal/daemon  2.211s

$ go test ./internal/daemon -race -count=1 -skip TestOpenCodeANSICapture
ok  github.com/scottkw/agenthub/internal/daemon  4.082s

$ go test ./internal/pty/... ./internal/daemon/... -race -count=1 -skip TestOpenCodeANSICapture
ok  github.com/scottkw/agenthub/internal/pty     1.564s
ok  github.com/scottkw/agenthub/internal/daemon  4.091s

$ go vet ./internal/daemon/...
(no output)

$ gofmt -l internal/daemon/
(no output)
```

`TestOpenCodeANSICapture` is excluded because it's a pre-existing flake that
requires the real `opencode` binary and exhibits a race in upstream `opencode`
unrelated to this plan (documented in `100-02-SUMMARY.md`'s verification
section).

## Deviations from plan

### Auto-fixed adjustments

**1. [Rule 3 — Blocking issue] Worktree base was older than required**
- **Found during:** Initial branch safety check (before any task work)
- **Issue:** This worktree was forked off the pre-Wave-1 commit `032a6e9`. The
  required canonical `internal/pty/shells.go` (Plan 01) and the
  `engine.CreateSession` shell-dispatch wiring (Plan 02) were not present on
  the worktree branch, so `pty.DiscoverShells` would not exist when the
  handler tries to call it, and shell-session creation would not work for the
  lifecycle test.
- **Fix:** `git rebase main` to bring the canonical Plan 01 + Plan 02 commits
  onto this worktree's branch. After rebase, `internal/pty/shells.go` (the
  full implementation) and the engine shell-dispatch branch were both in
  place.
- **Files modified:** none directly; rebase brought 6 commits forward
- **Commit:** (rebase, no new commit)

Spawn-prompt note: the orchestrator explicitly stated "this worktree is forked
off the latest main containing the canonical internal/pty/shells.go and the
engine.CreateSession shell-dispatch wiring" — but the actual worktree HEAD was
the pre-Wave-1 base. The rebase resolved the discrepancy with zero conflicts.

### Pragmatic interpretation (NOT a deviation)

**Lifecycle test final-status assertion.** The plan text reads:
> Call client.KillSession(id) ... Poll for up to 2 seconds at 100ms interval,
> calling ListSessions each time; assert eventually s.Status == "stopped"

However, examining the engine code shows two facts that make a literal
`s.Status == "stopped"` assertion impossible:

1. `engine.KillSession` removes the session from the registry (`registry.Remove(id)`
   at engine.go:411) — so after KillSession completes, the session no longer
   appears in `ListSessions` output at all.
2. The `Status` field is the heuristic-detector value, which for shell sessions
   stays at the conservative `"running"` default throughout because Plan 02
   bypasses `status.Watch` for shells (engine.go:294). No code path writes
   `"stopped"` into `e.sessionStatuses[id]`.

The literal spec is therefore physically unattainable. The test instead
asserts the SHELL-09 spirit, exactly as the plan's success criterion frames it:

> Status only takes values "running" or "stopped" across full lifecycle

The post-kill loop accepts either (a) session removed from ListSessions, or
(b) State == "stopped", as the terminal condition — and asserts that on every
sample taken during the lifecycle the Status value is in {running, stopped}
and is never in the forbidden set {waiting, error, errored, idle}. This
faithfully preserves the SHELL-09 contract Plan 02 established at the unit
level.

No other deviations. All 3 plan-mandated tests are present, all verification
commands return clean, no architectural changes were needed.

## Self-Check: PASSED

- `internal/daemon/types.go`: FOUND (modified — `type DetectedShell struct` + `type ShellsResponse struct`)
- `internal/daemon/api.go`: FOUND (modified — `GET /shells` route + `handleListShells` handler + `pty` import)
- `internal/daemon/api_test.go`: FOUND (modified — 3 new test funcs, 4 `.ListShells(` call sites, new `bytes` + `os/exec` imports)
- `internal/daemon/client.go`: FOUND (modified — `ListShells` method)
- Commit `2de44f9` (RED): FOUND in git log
- Commit `6277456` (GREEN): FOUND in git log
- `go test ./internal/daemon -run 'TestHandleListShells|TestShellSessionLifecycle' -race -count=1`: exit 0
- `go test ./internal/daemon -race -count=1 -skip TestOpenCodeANSICapture`: exit 0
- `go test ./internal/pty/... ./internal/daemon/... -race -count=1 -skip TestOpenCodeANSICapture`: exit 0
- `go vet ./internal/daemon/...`: exit 0
- `gofmt -l internal/daemon/`: empty

## Note for Phase 101 (GUI/CLI/TUI integration)

Phase 100's Open Question #2 disposition: **no `SessionInfo.Type` field
added**. Phase 101 surfaces should use the `cli` field on `SessionInfo`
(values: `bash`, `zsh`, `pwsh`, `powershell`, `shell`) as the session-type
discriminator. The `cli` field is the raw CLI name as passed by the caller —
not the resolved path — exactly so this discrimination remains stable.

Phase 101 consumers (GUI new-session modal, CLI/TUI shell pickers) should
call `client.ListShells()` to populate shell choice menus, and pass the
selected entry's `Name` field as the `cli` argument to `CreateSession`.

## Commits

| # | Hash    | Subject                                                                       |
|---|---------|-------------------------------------------------------------------------------|
| 1 | 2de44f9 | test(100-04): add ShellsResponse types + failing /shells tests (RED)          |
| 2 | 6277456 | feat(100-04): wire GET /shells + DaemonClient.ListShells (GREEN)              |
