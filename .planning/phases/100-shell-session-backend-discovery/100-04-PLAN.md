---
phase: 100-shell-session-backend-discovery
plan: 04
type: execute
wave: 2
depends_on:
  - 100-01
  - 100-02
files_modified:
  - internal/daemon/types.go
  - internal/daemon/api.go
  - internal/daemon/api_test.go
  - internal/daemon/client.go
autonomous: true
requirements:
  - SHELL-04
  - SHELL-05
  - SHELL-09
user_setup: []

must_haves:
  truths:
    - "GET /shells returns 200 with a ShellsResponse body listing discovered shells"
    - "GET /shells body is shape {\"shells\": [{\"name\": ..., \"displayName\": ..., \"path\": ..., \"argv\": [...]}, ...]}"
    - "When no shells are installed (PATH empty), GET /shells returns {\"shells\": []} (non-null empty array, not null)"
    - "DaemonClient.ListShells() returns the same []DetectedShell slice as GET /shells body"
    - "Creating a shell session via the existing POST /sessions API (cli=bash) and then GET /sessions returns Status=running for that session (SHELL-09 end-to-end)"
    - "After the shell session's PTY exits, GET /sessions returns Status=stopped for that session (SHELL-09 end-to-end transition)"
  artifacts:
    - path: internal/daemon/types.go
      provides: "DetectedShell + ShellsResponse JSON wire types"
      contains: "type ShellsResponse struct"
    - path: internal/daemon/api.go
      provides: "handleListShells handler + route registration GET /shells"
      contains: "handleListShells"
    - path: internal/daemon/api_test.go
      provides: "TestHandleListShells_EmptyPATH + TestHandleListShells_PopulatedPATH + TestShellSessionLifecycle_StatusOnlyRunningOrStopped"
      contains: "TestHandleListShells"
    - path: internal/daemon/client.go
      provides: "(c *DaemonClient).ListShells() ([]DetectedShell, error)"
      contains: "func (c *DaemonClient) ListShells"
  key_links:
    - from: internal/daemon/api.go
      to: internal/pty.DiscoverShells
      via: "handleListShells calls pty.DiscoverShells() and converts to daemon.DetectedShell"
      pattern: "pty\\.DiscoverShells\\("
    - from: internal/daemon/client.go
      to: internal/daemon/api.go (GET /shells route)
      via: "doJSON GET request to /shells path"
      pattern: "\"/shells\""
    - from: internal/daemon/api_test.go (TestShellSessionLifecycle)
      to: internal/daemon/engine.go (CreateSession + ListSessions)
      via: "exercise full create→list→exit cycle for shell sessions"
      pattern: "(CreateSession|ListSessions)"
---

<objective>
Expose Plan 01's `pty.DiscoverShells()` over the daemon's HTTP API as `GET /shells` and add a corresponding `DaemonClient.ListShells()` method. Add an integration test exercising the full create→list→exit lifecycle for a shell session to prove SHELL-09 end-to-end (status only flips running→stopped, never waiting/error).

Purpose: Closes the SHELL-04 wire-format surface (Phase 101 GUI/CLI/TUI surfaces will consume this). Also provides an end-to-end SHELL-09 integration verification on top of Plan 02's unit-level guards.

Output: Four modified files. Adds JSON wire types, one new HTTP route, one new client method, and three new integration tests.

Task structure (per H3 fix — splits the original combined RED+GREEN task into a true RED→GREEN pair so the failing-test signal is preserved as a separate commit). The RED task adds types + tests and confirms the build fails referencing the not-yet-implemented handler/client. The GREEN task lands the handler + route + client method and asserts every test passes. We accept the slightly uglier intermediate commit because the assurance value of a true RED gate on the SHELL-09 end-to-end lifecycle assertion is high.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md
@.planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md
@.planning/phases/100-shell-session-backend-discovery/100-VALIDATION.md
@internal/daemon/types.go
@internal/daemon/api.go
@internal/daemon/api_test.go
@internal/daemon/client.go
@internal/pty/shells.go

<interfaces>
Existing daemon/types.go patterns to mirror:

  // CLIPathsResponse maps CLI name to custom path override.
  type CLIPathsResponse map[string]string

  // StatusResponse is the response body for GET /sessions/{id}/status.
  type StatusResponse struct {
      Status string `json:"status"`
  }

  // SessionInfo (existing) — note: does NOT embed pty.Session; fields copied with JSON tags.
  type SessionInfo struct {
      ID string `json:"id"`
      Name string `json:"name"`
      CLI string `json:"cli"`
      Hostname string `json:"hostname,omitempty"`
      State string `json:"state"`
      Status string `json:"status,omitempty"`
      // ... other fields ...
  }

Existing api.go patterns to mirror:

  // Route registration (api.go ~L60-95):
  a.mux.HandleFunc("GET /settings/cli-paths", a.handleGetCLIPaths)

  // Read-only handler (api.go ~L478):
  func (a *API) handleGetCLIPaths(w http.ResponseWriter, r *http.Request) {
      paths := a.engine.GetCLIPaths()
      if paths == nil { paths = map[string]string{} }
      writeJSON(w, http.StatusOK, paths)
  }

Existing api_test.go fixtures to reuse:

  // testDaemon (api_test.go ~L26-47):
  func testDaemon(t *testing.T) (*API, *DaemonClient, string) {
      // Skips Windows; creates engine with t.TempDir() configDir + empty cliPaths.
      // Returns API, client, socket path.
  }

  // rawGet (api_test.go ~L57-67):
  func rawGet(t *testing.T, socketPath, path string) (int, []byte) {
      // HTTP GET over Unix socket; returns status + body.
  }

Existing client.go patterns to mirror:

  // GetCLIPaths (client.go L97-104):
  func (c *DaemonClient) GetCLIPaths() (map[string]string, error) {
      var paths map[string]string
      if err := c.doJSON(http.MethodGet, "/settings/cli-paths", nil, &paths); err != nil {
          return nil, err
      }
      return paths, nil
  }

Plan 01 exports (consumed here):

  // internal/pty/shells.go
  type DetectedShell struct {
      Name string `json:"name"`
      DisplayName string `json:"displayName"`
      Path string `json:"path"`
      Argv []string `json:"argv"`
  }
  func DiscoverShells() []DetectedShell  // returns non-nil; can be empty

  // Plan 01 H4 contract (relied on by TestHandleListShells_EmptyPATH):
  // When $SHELL env is empty, DiscoverShells appends no synthetic "shell" entry,
  // regardless of /etc/shells contents. Locked by Plan 01's
  // TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1a: Add ShellsResponse wire types + failing tests (RED — build fails referencing handler/client method)</name>
  <files>internal/daemon/types.go, internal/daemon/api_test.go</files>
  <read_first>
    - internal/daemon/types.go (where to insert the new types — after CLIPathsResponse at L49-50)
    - internal/daemon/api_test.go (testDaemon at L26-47, rawGet at L57-67, TestAPIGetCLIPaths at L269-282)
    - internal/daemon/api.go (route table in registerRoutes around L60-95; handleGetCLIPaths at ~L478 — DO NOT edit in this task, just understand the shape)
    - .planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md sections for types.go, api_test.go
    - .planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md "Architectural Responsibility Map" row "Discovery API exposure" + "Open Questions" #1 (system-default entry inclusion)
    - internal/pty/shells.go (Plan 01 output, esp. the empty-$SHELL contract from H4)
  </read_first>
  <behavior>
    Add JSON wire types (DetectedShell, ShellsResponse) to types.go AND add three integration tests to api_test.go. The tests reference `a.handleListShells` (via the route table — they make HTTP calls to GET /shells, which 404s without the route) and `client.ListShells()` (which does not yet exist as a method on DaemonClient). Both gaps produce a build failure: `client.ListShells undefined`.

    The route 404 alone is not a build failure — it's a runtime failure that would mask "RED" as "not red enough." Anchoring the RED gate on `client.ListShells` is the cleanest way to assert "the integration tests cannot pass without Task 1b's implementation."

    Test names (must match VALIDATION.md verbatim):

    1. TestHandleListShells_EmptyPATH (SHELL-04 wire format):
       - Setup: t.Setenv("PATH", t.TempDir()) BEFORE calling testDaemon (so discovery sees empty PATH)
       - On POSIX, also t.Setenv("SHELL", "") to suppress the synthetic "shell" entry
         (relies on Plan 01 H4 contract: TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry locks "empty SHELL → no synthetic entry")
       - Call testDaemon to get socket path; rawGet "/shells"
       - Assert status == 200
       - Assert body unmarshal into ShellsResponse succeeds
       - Assert response.Shells != nil (non-nil empty slice, NOT null) — verify via raw JSON check: body contains `"shells":[]` or `"shells": []`, NOT `"shells":null`
       - Assert len(response.Shells) == 0

    2. TestHandleListShells_PopulatedPATH (SHELL-04 wire format, dev-host smoke):
       - No PATH override — use the host's real PATH (on dev macOS, bash + zsh are present)
       - rawGet "/shells", expect 200
       - Unmarshal into ShellsResponse
       - Assert len(response.Shells) >= 1 (at least one shell on PATH on any reasonable dev host)
       - For each entry: assert Name != "" AND Path != "" AND len(Argv) >= 1 AND DisplayName != ""
       - Assert at least one entry's Name is in the set {"bash", "zsh", "pwsh", "powershell", "shell"} (defensive — protects against schema drift)

    3. TestShellSessionLifecycle_StatusOnlyRunningOrStopped (SHELL-09 end-to-end, POSIX only):
       - Skip on Windows: `if runtime.GOOS == "windows" { t.Skip("requires POSIX shell") }`
       - Skip if bash not on PATH: `if _, err := exec.LookPath("bash"); err != nil { t.Skip("bash not installed") }`
       - testDaemon to get the client
       - Create a shell session via client.CreateSession (or call engine.CreateSession directly if no client method exists for it — verify existing test pattern in api_test.go)
       - Use CreateSession args: ctx, cli="bash", name="lifecycle-test", workDir="", args=[]string{} OR nil, cols=80, rows=24
       - Wait briefly (100ms) for the session to be registered
       - Call client.ListSessions() (or rawGet /sessions if that's the existing pattern)
       - Find the session by ID; assert s.Status == "running" (SHELL-09: never "waiting"/"error"/"errored"/"idle")
       - Repeat the status check 5 times over 1 second (200ms interval) — assert s.Status stays "running" throughout (no flickering to waiting/error)
       - Call client.KillSession(id) (or engine.KillSession)
       - Poll for up to 2 seconds at 100ms interval, calling ListSessions each time; assert eventually s.Status == "stopped"
       - Assert s.Status NEVER takes the values "waiting", "error", "errored", or "idle" at any point during the test

    The tests will fail to compile because:
    - `client.ListShells()` is referenced but the method does not exist (added in Task 1b)
    - `ShellsResponse` is referenced — declared in types.go in THIS task, so this part will compile

    The build failure point is therefore narrowly: `undefined: (*DaemonClient).ListShells` (or equivalent Go error wording). This is the RED gate.
  </behavior>
  <action>
    Make two edits in a single commit:

    1. internal/daemon/types.go — Add after CLIPathsResponse (around L50):
       ```go
       // DetectedShell is the JSON-serialisable representation of a discovered shell.
       // Mirrors internal/pty.DetectedShell; duplicated to keep daemon wire types
       // decoupled from internal/pty's Go API (see SessionInfo at L4-16 for the same
       // pattern — wire types are not pty.* embeds).
       type DetectedShell struct {
           Name        string   `json:"name"`
           DisplayName string   `json:"displayName"`
           Path        string   `json:"path"`
           Argv        []string `json:"argv"`
       }

       // ShellsResponse is the response body for GET /shells.
       type ShellsResponse struct {
           Shells []DetectedShell `json:"shells"`
       }
       ```

    2. internal/daemon/api_test.go — Append the three test functions described in <behavior>. Reuse testDaemon and rawGet fixtures verbatim.

       For TestShellSessionLifecycle_StatusOnlyRunningOrStopped, the test must drive an actual session create + kill cycle. If api_test.go already has client.CreateSession / client.KillSession patterns, mirror them. If not, use the engine-direct pattern from engine_test.go and skip the HTTP layer for the lifecycle assertions (still using the testDaemon fixture for setup). Either approach is acceptable — choose whichever matches the existing dominant pattern in api_test.go.

       Each test must reference `client.ListShells()` at least once (for tests 1 and 2 this is direct; for test 3 it's optional but encouraged for symmetry — alternately call rawGet "/shells" in test 3 to anchor the SHELL-04 wire surface). Tests 1 and 2 MUST call `client.ListShells()` so the RED gate engages on the undefined symbol.

       Add imports as needed: "encoding/json", "os/exec", "runtime", "strings", "time", "bytes".

    Do NOT edit internal/daemon/api.go or internal/daemon/client.go in this task — that's Task 1b's job. The build failure caused by undefined `client.ListShells` is the RED signal.

    Run `goimports -w internal/daemon/types.go internal/daemon/api_test.go` and `gofmt -w` on both.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go build ./internal/daemon/... 2>&1 | grep -qE 'undefined:[[:space:]]*(daemon\.)?\(?\*?DaemonClient\)?\)?\.ListShells|c\.ListShells undefined|undefined:[[:space:]]*ListShells' && echo "RED-as-expected (build fails: client.ListShells undefined)" || (echo "FAIL: build did not fail with the expected undefined-symbol error for client.ListShells. Build output:"; go build ./internal/daemon/... 2>&1; exit 1)</automated>
  </verify>
  <acceptance_criteria>
    - internal/daemon/types.go contains `type DetectedShell struct`
    - internal/daemon/types.go contains `type ShellsResponse struct`
    - internal/daemon/types.go DetectedShell has JSON tags `name`, `displayName`, `path`, `argv` — verify by:
      `grep -c 'json:"\(name\|displayName\|path\|argv\)"' internal/daemon/types.go` returns >= 4
      (BSD-grep-safe alternative to backtick-heavy patterns per L1 advisory)
    - internal/daemon/api_test.go contains `func TestHandleListShells_EmptyPATH`
    - internal/daemon/api_test.go contains `func TestHandleListShells_PopulatedPATH`
    - internal/daemon/api_test.go contains `func TestShellSessionLifecycle_StatusOnlyRunningOrStopped`
    - internal/daemon/api_test.go references `client.ListShells()` or `.ListShells(` in at least 2 test bodies (verify: `grep -c '\.ListShells(' internal/daemon/api_test.go` returns >= 2)
    - `go build ./internal/daemon/...` FAILS with an undefined-symbol error referencing `ListShells` (the RED gate — proves Task 1b is the GREEN-providing implementation)
    - api.go and client.go are UNCHANGED in this task (verify via `git diff --stat internal/daemon/api.go internal/daemon/client.go` shows no modifications)
  </acceptance_criteria>
  <done>RED phase committed: types.go has the wire types, api_test.go has three failing-to-compile tests that reference `client.ListShells`. Build fails with `undefined: ListShells` (or equivalent). The next commit (Task 1b) lands the implementation and turns this green.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 1b: Add handleListShells route + DaemonClient.ListShells (GREEN — all three Task 1a tests pass)</name>
  <files>internal/daemon/api.go, internal/daemon/client.go</files>
  <read_first>
    - internal/daemon/api.go (route table in registerRoutes around L60-95; handleGetCLIPaths at ~L478 — the analog handler to mirror)
    - internal/daemon/client.go (GetCLIPaths at L97-104 — the analog client method to mirror)
    - internal/daemon/types.go (Task 1a output — DetectedShell + ShellsResponse types)
    - internal/daemon/api_test.go (Task 1a output — see what's tested)
    - .planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md sections for api.go, client.go
    - internal/pty/shells.go (Plan 01 output)
  </read_first>
  <behavior>
    After this task, Task 1a's three tests pass:
    - TestHandleListShells_EmptyPATH: GET /shells returns 200 with `{"shells":[]}` on wire (non-null empty array) when PATH is empty and (POSIX) $SHELL is empty.
    - TestHandleListShells_PopulatedPATH: GET /shells returns 200 with a populated Shells slice on a real dev host.
    - TestShellSessionLifecycle_StatusOnlyRunningOrStopped: create-list-kill cycle for a bash session shows Status only running→stopped.

    The full daemon test suite passes with no regressions.
  </behavior>
  <action>
    Make two edits in a single commit:

    1. internal/daemon/api.go — Two sub-edits in this file:

       (a) Register the route in registerRoutes near the cli-paths route (around L68):
           a.mux.HandleFunc("GET /shells", a.handleListShells)

       (b) Add the handler near handleGetCLIPaths (around L478):
           ```go
           // handleListShells returns the daemon's view of installed shells per
           // pty.DiscoverShells. Read-only; no engine state mutated.
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
       Note the `make([]DetectedShell, 0, len(...))` and the defensive Argv copy: ensures JSON output is `"shells":[]` not `"shells":null` and that callers cannot mutate the package-level knownShellSpecs table via the response.

       Add `"github.com/scottkw/agenthub/internal/pty"` import if not already present.

    2. internal/daemon/client.go — Add method (place near GetCLIPaths at L97):
       ```go
       // ListShells returns the daemon's discovery of installed shells.
       func (c *DaemonClient) ListShells() ([]DetectedShell, error) {
           var resp ShellsResponse
           if err := c.doJSON(http.MethodGet, "/shells", nil, &resp); err != nil {
               return nil, err
           }
           return resp.Shells, nil
       }
       ```

    Run `goimports -w internal/daemon/api.go internal/daemon/client.go` and `gofmt -w` on both. Run `go vet ./internal/daemon/...`.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/daemon -run 'TestHandleListShells|TestShellSessionLifecycle' -race -count=1 -v && go test ./internal/daemon -race -count=1</automated>
  </verify>
  <acceptance_criteria>
    - internal/daemon/api.go contains `func (a *API) handleListShells`
    - internal/daemon/api.go contains `"GET /shells"` route registration (verify: `grep -c 'GET /shells' internal/daemon/api.go` returns >= 1)
    - internal/daemon/api.go handleListShells uses `make([]DetectedShell, 0,` (non-null empty slice guarantee)
    - internal/daemon/client.go contains `func (c *DaemonClient) ListShells`
    - `go test ./internal/daemon -run TestHandleListShells_EmptyPATH -race -count=1` exits 0
    - `go test ./internal/daemon -run TestHandleListShells_PopulatedPATH -race -count=1` exits 0 (on dev box with bash+zsh)
    - `go test ./internal/daemon -run TestShellSessionLifecycle_StatusOnlyRunningOrStopped -race -count=1` exits 0 on POSIX
    - `go test ./internal/daemon -race -count=1` exits 0 (no regression to existing daemon tests)
    - `go vet ./internal/daemon/...` exits 0
    - `gofmt -l internal/daemon/api.go internal/daemon/client.go` produces no output
    - Raw-JSON empty-shells body assertion: when discovery returns empty, the on-wire bytes contain `"shells":[]` and NOT `"shells":null` (the assertion lives in TestHandleListShells_EmptyPATH, using `bytes.Contains(body, []byte("\"shells\":[]"))`; if BSD-grep escaping causes issues on macOS shells, use Go-side bytes.Contains in the test rather than grep at the shell layer)
  </acceptance_criteria>
  <done>api.go and client.go committed. GET /shells returns 200 with ShellsResponse JSON. DaemonClient.ListShells() works end-to-end via Unix socket. SHELL-09 verified end-to-end: shell sessions transition only running→stopped, never waiting/error. No existing daemon tests regress. The Task 1a RED commit now reads as a properly motivated failing-test snapshot in `git log`.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| HTTP / Unix-socket caller → GET /shells | The new route exposes the daemon's view of installed shells. No state is mutated. Information disclosure scope: existing CLI-paths route already discloses similar information; shells discovery is analogous (binary paths the daemon knows about). |
| HTTP / Unix-socket caller → POST /sessions (cli=bash) | Pre-existing route; Plan 02 expanded its behavior to spawn shells. Plan 04 only tests the lifecycle; no new attack surface added here. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-100-01b | Information Disclosure | GET /shells response leaks installed shell paths | accept | Listing installed shell binary paths is equivalent in sensitivity to the existing GET /settings/cli-paths route. Daemon API access already implies host trust (Unix-socket / capability-token gated). No new disclosure surface. |
| T-100-09 | Tampering | Caller mutates ShellsResponse.Shells slice via returned pointer | mitigate | Argv is defensively copied via `append([]string(nil), s.Argv...)` in handleListShells. Plan 01's DiscoverShells also returns a fresh slice per call. No shared mutable state escapes the handler. |
| T-100-S10 | Denial of Service | GET /shells called in a hot loop triggers PATH scan thrash | accept | Each call performs ~4 exec.LookPath calls plus an optional small file read. Bounded, no network, no allocations beyond the response slice. If profiling shows this is a hot path in Phase 101 (GUI new-session modal opens frequently), add a cache then. Phase 100 ships without caching. |
</threat_model>

<verification>
After both tasks complete:

```
# Plan-scoped gate (matches VALIDATION.md sampling rate):
go test ./internal/daemon -run 'TestHandleListShells|TestShellSessionLifecycle' -race -count=1

# Full daemon regression:
go test ./internal/daemon -race -count=1

# Cross-package phase gate:
go test ./internal/pty/... ./internal/daemon/... -race -count=1

# Full suite (final phase gate):
go test ./... -race -count=1

# Vet + format:
go vet ./internal/daemon/...
gofmt -l internal/daemon/  # only path_windows.go from Plan 03 should appear (cross-compile-only)
```

Wire-format manual smoke (against a running daemon):
```
curl --unix-socket ~/.config/agenthub/daemon.sock http://daemon/shells
# Expect: {"shells":[{"name":"bash","displayName":"bash","path":"/bin/bash","argv":["-i"]}, ...]}
# When PATH has no shells, body must be {"shells":[]} — NOT {"shells":null}
```

Validation map IDs satisfied: TBD-04-api-route, TBD-09-list-stopped (end-to-end SHELL-09 lifecycle).
</verification>

<success_criteria>
- Four files modified across two tasks: types.go + api_test.go (Task 1a, RED), api.go + client.go (Task 1b, GREEN)
- Task 1a leaves a build-failing commit referencing `client.ListShells undefined` (preserves RED signal per H3)
- Task 1b's commit turns the build green; all three tests pass
- `go test ./internal/daemon -race -count=1` exits 0 (no regressions; all new tests pass)
- `go test ./internal/pty/... ./internal/daemon/... -race -count=1` exits 0 (cross-package phase gate)
- GET /shells returns 200 with shape `{"shells": [...]}`; empty case is `{"shells":[]}` not `{"shells":null}`
- DaemonClient.ListShells() round-trips successfully over Unix socket
- SHELL-09 end-to-end: shell session Status only takes values "running" or "stopped" across full lifecycle (verified across 5 polling samples + post-kill transition)
- Phase 100 complete: all three requirements (SHELL-04 discovery + wire surface, SHELL-05 spawn semantics, SHELL-09 status exclusion) satisfied
</success_criteria>

<output>
After completion, create `.planning/phases/100-shell-session-backend-discovery/100-04-SUMMARY.md` documenting:
- GET /shells route signature, request/response shape (JSON sample)
- DaemonClient.ListShells() Go signature
- Test coverage: TestHandleListShells_EmptyPATH (empty PATH), TestHandleListShells_PopulatedPATH (dev-host real PATH), TestShellSessionLifecycle_StatusOnlyRunningOrStopped (SHELL-09 end-to-end)
- Confirmation of `"shells":[]` non-null empty-array wire guarantee
- Phase 100 closeout summary: all 3 requirement IDs covered across Plans 01-04 with traceability table
- Note on RED→GREEN split (H3): Task 1a's commit intentionally leaves `client.ListShells undefined` as the RED gate; Task 1b's commit lands the implementation. `git log` reads as a clean TDD pair.
- Any deviations from RESEARCH.md (e.g., if Open Question #1 system-default entry inclusion changed in implementation) with rationale
- Note for Phase 101: SessionInfo.Type field NOT added in Phase 100 per Open Question #2; cli field value (bash/zsh/pwsh/powershell/shell) is the session-type discriminator
</output>
</content>
</invoke>