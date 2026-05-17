---
phase: 100-shell-session-backend-discovery
plan: 02
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/daemon/engine.go
  - internal/daemon/engine_test.go
autonomous: true
requirements:
  - SHELL-05
  - SHELL-09
user_setup: []

must_haves:
  truths:
    - "When CreateSession is called with cli='bash', the backend receives an absolute path ending in /bash and Args containing '-i' (interactive)"
    - "When CreateSession is called with cli='bash', Args never contains '-l' or '--login' (non-login per SHELL-05)"
    - "When CreateSession is called with cli='zsh', Args contains '-i' and never '-l'/'--login'"
    - "When CreateSession is called with cli='pwsh', Args contains '-NoLogo' and never '-l'/'--login'"
    - "When CreateSession is called with cli='powershell' (override path set), Args contains '-NoLogo' and resolveShellSpawn matches via knownShellSpecs (M2: 'powershell' is now a first-class spec in Plan 01)"
    - "When CreateSession is called with cli='shell', the resolved path matches $SHELL (POSIX) or pwsh.exe/powershell.exe (Windows)"
    - "Caller-supplied non-empty WorkDir reaches the backend unchanged (cmd.Dir == WorkDir)"
    - "Empty WorkDir for shell sessions resolves to os.UserHomeDir() — NOT daemon CWD"
    - "Empty WorkDir for AI CLI sessions retains existing behavior (no $HOME substitution)"
    - "go status.Watch is NEVER invoked for shell sessions (cli in {shell, bash, zsh, pwsh, powershell})"
    - "engine.sessionStatuses[id] has no entry for shell sessions"
    - "engine.GetSessionStatus(id) returns 'running' for live shell sessions (conservative default at engine.go:308)"
    - "engine.GetSessionStatus(id) returns 'stopped' for shell sessions after PTY exit (existing State transition path)"
    - "engine.go knownShells map (L97) includes pwsh, pwsh.exe, powershell, powershell.exe (Pitfall 6 mitigation)"
  artifacts:
    - path: internal/daemon/engine.go
      provides: "resolveShellSpawn(cli) (path, args, ok); isShellSession(cli) bool; defaultShellWorkDir() string; status.Watch guard wrapping line ~245; extended knownShells map"
      contains: "func (e *SessionEngine) resolveShellSpawn"
    - path: internal/daemon/engine_test.go
      provides: "TestCreateSession_ShellArgv_Interactive, TestCreateSession_ZshArgv_Interactive, TestCreateSession_PwshArgv, TestCreateSession_ShellWorkDirHonored, TestCreateSession_ShellEmptyWorkDirHome, TestCreateSession_ShellSkipsStatusWatch, TestListSessions_ShellStatusRunning, TestShell_NoStatusMapEntry"
      contains: "func TestCreateSession_ShellArgv_Interactive"
  key_links:
    - from: internal/daemon/engine.go
      to: internal/pty.DiscoverShells
      via: "resolveShellSpawn calls pty.DiscoverShells() and pty.KnownShellSpecs()"
      pattern: "pty\\.(DiscoverShells|KnownShellSpecs|DetectShell)"
    - from: internal/daemon/engine.go
      to: os.UserHomeDir
      via: "shell session WorkDir default"
      pattern: "os\\.UserHomeDir\\("
    - from: internal/daemon/engine.go (status.Watch call site)
      to: isShellSession
      via: "if-guard wraps go status.Watch(...) call"
      pattern: "if !isShellSession\\(cli\\)"
---

<objective>
Wire the daemon's `engine.CreateSession` to handle shell sessions: (1) resolve the abstract cli name (`shell`, `bash`, `zsh`, `pwsh`, `powershell`) into a concrete absolute path + non-login interactive argv, (2) substitute `$HOME` when caller supplies empty WorkDir for shells, and (3) skip the AI-CLI status heuristic `go status.Watch(...)` for shell sessions so `SessionInfo.Status` only ever resolves to `"running"` or `"stopped"` (per SHELL-09).

Purpose: Closes SHELL-05 (interactive non-login PTY spawn with WorkDir honor) and SHELL-09 (status heuristic exclusion). Together with Plan 01 (discovery library — now exposing `powershell` as a first-class knownShellSpec per M2) and Plan 04 (HTTP route), this completes the daemon side of Phase 100.

Output: Two modified files (`engine.go`, `engine_test.go`). Zero new files. No changes to `internal/pty/native.go` (per Pitfall 3 — must not touch sysproc attributes).
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
@internal/daemon/engine.go
@internal/daemon/engine_test.go
@internal/pty/shells.go

<interfaces>
<!-- engine.go relevant signatures. Source verified at /Users/ken/dev/agenthub/internal/daemon/engine.go. -->

CreateSession signature (engine.go:204):
```go
func (e *SessionEngine) CreateSession(
    ctx context.Context,
    cli, name, workDir string,
    args []string,
    cols, rows int,
    onStatus func(string, status.SessionStatus),
    onExit func(string, int),
) (string, error)
```

Existing per-CLI customization site (engine.go:214-217) — the opencode branch is the precedent:
```go
var env []string
if cli == "opencode" && e.opencodeTUIConfig != "" {
    env = append(env, "OPENCODE_TUI_CONFIG="+e.opencodeTUIConfig)
}
```

Existing status.Watch call site (engine.go:245):
```go
go status.Watch(hub, id, cli, func(sid string, s status.SessionStatus) {
    e.statusMu.Lock()
    e.sessionStatuses[sid] = s
    e.statusMu.Unlock()
    if onStatus != nil { onStatus(sid, s) }
})
```

Existing ListSessions conservative default (engine.go:308):
```go
heuristicStatus := string(status.StatusRunning) // conservative default
e.statusMu.RLock()
if hs, ok := e.sessionStatuses[s.ID]; ok {
    heuristicStatus = string(hs)
}
e.statusMu.RUnlock()
```

Existing knownShells defensive map (engine.go:97-100):
```go
var knownShells = map[string]bool{
    "sh": true, "bash": true, "zsh": true, "fish": true,
    "csh": true, "tcsh": true, "dash": true, "ksh": true,
}
```

ResolveCLI signature (engine.go:386):
```go
func (e *SessionEngine) ResolveCLI(name string) string {
    e.mu.RLock(); defer e.mu.RUnlock()
    if path, ok := e.cliPaths[name]; ok { return path }
    return name
}
```

spyBackend harness (engine_test.go:241-257):
```go
type spyBackend struct{ lastReq pty.CreateRequest }
func (s *spyBackend) Create(_ context.Context, req pty.CreateRequest) (*pty.Session, error) {
    s.lastReq = req
    return &pty.Session{ID: "spy-id", CLI: req.CLI, State: pty.StateRunning, CreatedAt: time.Now()}, nil
}
func (s *spyBackend) Resize(string, int, int) error { return nil }
func (s *spyBackend) Kill(string) error             { return nil }
func (s *spyBackend) List() []*pty.Session          { return nil }
```

Engine test isolation (mandatory per PATTERNS.md § "Settings file isolation in tests"):
```go
e := NewSessionEngine()
e.configDir = t.TempDir()
e.cliPaths = make(map[string]string)
```

pty.DetectedShell exported by Plan 01 (relied on here). Per M2, knownShellSpecs now has 4 entries (bash, zsh, pwsh, powershell):
```go
type DetectedShell struct {
    Name string `json:"name"`
    DisplayName string `json:"displayName"`
    Path string `json:"path"`
    Argv []string `json:"argv"`
}
func DiscoverShells() []DetectedShell
func KnownShellSpecs() []ShellSpec  // returns [bash, zsh, pwsh, powershell] per M2
func DetectShell(name string) (*DetectedShell, error)
var ErrShellNotFound = errors.New("shell not found")
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Extend engine_test.go with failing shell-spawn + status-bypass tests (RED)</name>
  <files>internal/daemon/engine_test.go</files>
  <read_first>
    - internal/daemon/engine_test.go (full file — spyBackend at L241-257, TestCreateSession_OpenCodeEnv at L262-320, TestEngineGetSessionStatus, TestSessionCLIs_TrackedAndCleanedUp at L384-388 for internal-state-probe pattern)
    - .planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md § "internal/daemon/engine_test.go (MODIFY — test)"
    - .planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md § "Code Examples" → TestCreateSession_ShellArgv_Interactive and TestCreateSession_ShellSkipsStatusWatch
    - .planning/phases/100-shell-session-backend-discovery/100-VALIDATION.md (verify test names match TBD-05-* and TBD-09-* entries verbatim)
    - internal/pty/shells.go (Plan 01 output — see what's exported, esp. M2's 4-entry knownShellSpecs)
  </read_first>
  <behavior>
    Add the following test functions to internal/daemon/engine_test.go. All must use the existing `spyBackend` harness and the mandatory engine isolation idiom (`e.configDir = t.TempDir(); e.cliPaths = make(map[string]string)`).

    Test names (must match VALIDATION.md verbatim — Plan checker greps these via -run):

    1. TestCreateSession_ShellArgv_Interactive (SHELL-05):
       - Skip on Windows (relies on /bin/bash existing).
       - Inject spy backend; call CreateSession(ctx, "bash", "tab", "/tmp", nil, 80, 24, nil, nil)
       - Assert spy.lastReq.CLI ends with "/bash" (absolute path)
       - Assert spy.lastReq.Args contains "-i"
       - Assert spy.lastReq.Args does NOT contain "-l" and does NOT contain "--login"
       - Assert spy.lastReq.WorkDir == "/tmp"

    2. TestCreateSession_ZshArgv_Interactive (SHELL-05):
       - Same shape as #1, cli="zsh", expect path ends with "/zsh" and Args has "-i", not "-l"/"--login".

    3. TestCreateSession_PwshArgv (SHELL-05):
       - Skip if `exec.LookPath("pwsh")` returns error (pwsh not installed — common on macOS dev box).
       - Call CreateSession(ctx, "pwsh", ...); assert spy.lastReq.Args contains "-NoLogo"; never "-l"/"--login".

    4. TestCreateSession_ShellWorkDirHonored (SHELL-05):
       - Skip on Windows.
       - CreateSession(ctx, "bash", "tab", "/home/user/project", ...); assert spy.lastReq.WorkDir == "/home/user/project".

    5. TestCreateSession_ShellEmptyWorkDirHome (SHELL-05, Pitfall 4 mitigation):
       - Skip on Windows.
       - CreateSession(ctx, "bash", "tab", "", ...);
       - home, _ := os.UserHomeDir()
       - Assert spy.lastReq.WorkDir == home AND home != "" AND home != "/" AND home != "."

    6. TestCreateSession_AICLIEmptyWorkDirUnchanged (SHELL-05 negative case — protects against regression):
       - CreateSession(ctx, "claude", "tab", "", ...);
       - Assert spy.lastReq.WorkDir == "" (existing AI-CLI behavior: do NOT substitute $HOME for non-shell sessions).

    7. TestCreateSession_ShellSkipsStatusWatch (SHELL-09):
       - Skip on Windows.
       - CreateSession(ctx, "bash", "tab", "", ...); get id.
       - Sleep 50ms to give any unintended goroutine time to write to sessionStatuses.
       - Read e.GetSessionStatus(id) — must equal "running".
       - Also probe internal map state under e.statusMu.RLock():
         ```go
         e.statusMu.RLock()
         _, exists := e.sessionStatuses[id]
         e.statusMu.RUnlock()
         if exists { t.Errorf("expected no entry in sessionStatuses for shell session, but found one") }
         ```

    8. TestListSessions_ShellStatusRunning (SHELL-09):
       - Skip on Windows.
       - CreateSession(ctx, "bash", "tab", "", ...); call e.ListSessions().
       - Find the SessionInfo matching id; assert s.Status == "running".
       - Assert s.Status != "waiting" AND s.Status != "error" AND s.Status != "errored" AND s.Status != "idle".

    9. TestShell_NoStatusMapEntry (SHELL-09 defensive):
       - Skip on Windows.
       - Create three shell sessions (bash, zsh, system "shell"); create one AI-CLI session ("claude").
       - For each shell-session id: verify sessionStatuses[id] absent.
       - For the claude id: verify sessionStatuses entry will exist OR the goroutine has not yet populated it (this is timing-dependent; relax to "the absence-or-presence for shells is consistent" — the load-bearing assertion is shell IDs have no entry; don't assert anything about claude).

    10. TestIsShellSession_AllShellNames (unit test for helper):
        - Table-driven test: {"shell": true, "bash": true, "zsh": true, "pwsh": true, "powershell": true, "claude": false, "opencode": false, "": false, "Bash": false (case-sensitive), "/bin/bash": false (basename match is in resolveShellSpawn, not isShellSession)}
        - Assert isShellSession(name) matches expected for each.

    11. TestResolveShellSpawn_KnownShell (unit test for helper):
        - Skip on Windows for bash/zsh probes.
        - Call e.resolveShellSpawn("bash"); assert ok=true, path ends with "/bash", args contains "-i".
        - Call e.resolveShellSpawn("claude"); assert ok=false.

    12. TestResolveShellSpawn_SystemDefault (POSIX-only):
        - t.Setenv("SHELL", "/bin/zsh") (or similar real path); ensure /bin/zsh exists via exec.LookPath skip.
        - Call e.resolveShellSpawn("shell"); assert ok=true, path == "/bin/zsh", args contains "-i".

    13. TestResolveShellSpawn_PowerShellOverride (M2 lock-in — POSIX OK because we test the override branch, not actual exec):
        - Set e.cliPaths["powershell"] = "/usr/local/bin/pwsh-stub" (any non-existent path is fine — the override branch matches on basename, not file existence).
        - Call e.resolveShellSpawn("powershell"); assert ok=true (matched via M2: knownShellSpecs now contains "powershell").
        - Assert returned path == "/usr/local/bin/pwsh-stub" (override honored).
        - Assert returned args contains "-NoLogo" (argv from the "powershell" knownShellSpec entry).
        - This test locks the M2 contract: a `cliPaths["powershell"]` override resolves cleanly without falling through to discovery.

    For probes #7 and #9, the test reaches into engine internals (e.statusMu, e.sessionStatuses). This is the documented pattern (engine_test.go:384-388). Do NOT add new exported accessors just for tests.
  </behavior>
  <action>
    Append the 13 test functions described in <behavior> to `internal/daemon/engine_test.go`. Each test:
    - Begins with `spy := &spyBackend{}` and `e := NewSessionEngine(); e.configDir = t.TempDir(); e.cliPaths = make(map[string]string); e.backend = spy` (mandatory isolation).
    - Skips on Windows if it depends on POSIX shells (use `if runtime.GOOS == "windows" { t.Skip("requires POSIX shell binaries") }`).
    - Skips pwsh-specific tests via `if _, err := exec.LookPath("pwsh"); err != nil { t.Skip("pwsh not installed") }`.

    Add imports if not already present: "os", "os/exec", "runtime", "strings", "time".

    Do NOT yet modify engine.go — the test file must reference `e.resolveShellSpawn` and `isShellSession` which do not yet exist, causing build failure. This is the RED phase.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go build ./internal/daemon/... 2>&1 | grep -qE 'undefined:[[:space:]]*(daemon\.)?(resolveShellSpawn|isShellSession)' && echo "RED-as-expected (build fails with undefined-symbol error for the not-yet-implemented helpers)" || (echo "FAIL: build did not fail referencing the unimplemented helpers. Build output:"; go build ./internal/daemon/... 2>&1; exit 1)</automated>
  </verify>
  <acceptance_criteria>
    - File internal/daemon/engine_test.go contains func TestCreateSession_ShellArgv_Interactive
    - File contains func TestCreateSession_ZshArgv_Interactive
    - File contains func TestCreateSession_PwshArgv
    - File contains func TestCreateSession_ShellWorkDirHonored
    - File contains func TestCreateSession_ShellEmptyWorkDirHome
    - File contains func TestCreateSession_AICLIEmptyWorkDirUnchanged
    - File contains func TestCreateSession_ShellSkipsStatusWatch
    - File contains func TestListSessions_ShellStatusRunning
    - File contains func TestShell_NoStatusMapEntry
    - File contains func TestIsShellSession_AllShellNames
    - File contains func TestResolveShellSpawn_KnownShell
    - File contains func TestResolveShellSpawn_SystemDefault
    - File contains func TestResolveShellSpawn_PowerShellOverride (M2)
    - Every new test invokes `e.configDir = t.TempDir()` (verifiable via grep)
    - `go build ./internal/daemon/...` fails with `undefined:` error for `resolveShellSpawn` or `isShellSession`
  </acceptance_criteria>
  <done>engine_test.go contains all 13 new tests; build fails referencing the unimplemented helpers (RED phase confirmed).</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement engine.go shell-spawn + status-bypass (GREEN — all Plan 01+02 tests pass)</name>
  <files>internal/daemon/engine.go</files>
  <read_first>
    - internal/daemon/engine.go (current state; especially L94-100 knownShells, L200-260 CreateSession, L285-341 ListSessions, L386-395 ResolveCLI)
    - internal/pty/shells.go (Plan 01 output — DiscoverShells, KnownShellSpecs returning [bash,zsh,pwsh,powershell] per M2, DetectedShell types)
    - .planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md § "internal/daemon/engine.go (MODIFY — service)"
    - .planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md § "Pattern 2" + "Pattern 3" + "Pitfall 4" + "Pitfall 6"
    - internal/daemon/engine_test.go (Task 1 output — see what's tested)
  </read_first>
  <behavior>
    After this task, every test from Task 1 passes (GREEN). Specifically:
    - cli="bash"/"zsh"/"pwsh"/"powershell" routes through resolveShellSpawn to set cliPath = absolute path and args = ShellSpec.Argv
    - cli="shell" resolves to the system default via pty.DiscoverShells (matches the synthetic Name="shell" entry)
    - cliPaths["powershell"] override resolves via the override branch matching knownShellSpec "powershell" (M2 ripple — no fallthrough to discovery needed)
    - Empty WorkDir for any shell session becomes os.UserHomeDir() (silently no-op if home unavailable)
    - Empty WorkDir for AI CLI sessions remains "" (no regression)
    - go status.Watch(...) is wrapped in `if !isShellSession(cli) { ... }` so shell sessions never invoke the heuristic detector
    - knownShells map (L97) gains pwsh, pwsh.exe, powershell, powershell.exe so stale "claude -> pwsh.exe" CLI overrides are filtered on load
    - User-supplied req.Args (CreateRequest.Args) for shells is IGNORED in Phase 100 (per RESEARCH.md Anti-Patterns + Assumption A6). AI CLIs continue to pass args through unchanged.
  </behavior>
  <action>
    Modify `internal/daemon/engine.go` with these surgical edits. Do NOT touch unrelated code.

    1. Extend `knownShells` map (L94-100) to include pwsh + .exe forms:
       ```go
       var knownShells = map[string]bool{
           "sh": true, "bash": true, "zsh": true, "fish": true,
           "csh": true, "tcsh": true, "dash": true, "ksh": true,
           "pwsh": true, "pwsh.exe": true,
           "powershell": true, "powershell.exe": true,
       }
       ```

    2. Add package-level helper `isShellSession` (place near `knownShells` for thematic grouping):
       ```go
       // isShellSession returns true if cli refers to a shell-type session
       // (vs an AI CLI). Used to bypass status.Watch (SHELL-09) and to gate
       // shell-specific argv/workdir resolution.
       func isShellSession(cli string) bool {
           switch cli {
           case "shell", "bash", "zsh", "pwsh", "powershell":
               return true
           }
           return false
       }
       ```

    3. Add method `resolveShellSpawn` on `*SessionEngine` (place near `ResolveCLI` at L386). Per M2, `pty.KnownShellSpecs()` now includes a `powershell` entry — the override branch matches uniformly without a special-case pwsh↔powershell fallback:
       ```go
       // resolveShellSpawn maps an abstract shell name (shell|bash|zsh|pwsh|powershell) to
       // (absolute path, interactive non-login argv, ok=true). Returns
       // ("", nil, false) when cli is not a shell name.
       //
       // Custom path overrides from e.cliPaths take precedence over PATH discovery
       // (mirrors ResolveCLI semantics). Since Plan 01's knownShellSpecs (per M2)
       // contains both `pwsh` and `powershell` as first-class specs, the override
       // branch resolves cliPaths["powershell"] cleanly via knownShellSpecs lookup —
       // no special pwsh↔powershell name-mismatch fallback is needed.
       func (e *SessionEngine) resolveShellSpawn(cli string) (string, []string, bool) {
           if !isShellSession(cli) {
               return "", nil, false
           }

           // Per-cli override via Settings (custom shell binary picker is out-of-scope
           // for Phase 100, but ResolveCLI may already hold an override from prior versions).
           override := e.ResolveCLI(cli) // returns cli itself if no override
           if override != cli {
               // Try to match by basename against known shell specs to get argv.
               // M2: knownShellSpecs now contains "powershell" alongside "pwsh",
               // so cliPaths["powershell"] resolves on the first name check below.
               base := filepath.Base(override)
               for _, spec := range pty.KnownShellSpecs() {
                   if spec.Name == cli || spec.Name == strings.TrimSuffix(base, ".exe") || spec.Name == base {
                       argv := append([]string(nil), spec.Argv...)
                       return override, argv, true
                   }
               }
               // Override path basename doesn't match any known shell — bail to discovery.
           }

           // Live discovery via pty.DiscoverShells (Plan 01).
           for _, sh := range pty.DiscoverShells() {
               if sh.Name == cli {
                   argv := append([]string(nil), sh.Argv...)
                   return sh.Path, argv, true
               }
           }

           // Safety net: if cli="pwsh" but the host only has powershell.exe
           // discoverable (rare — Windows 5.x only, no PowerShell 7 installed),
           // accept the canonical "powershell" entry. M2 made "powershell" a
           // first-class spec but a caller specifying cli="pwsh" still wants
           // a PowerShell session — this preserves API consistency.
           if cli == "pwsh" {
               for _, sh := range pty.DiscoverShells() {
                   if sh.Name == "powershell" {
                       argv := append([]string(nil), sh.Argv...)
                       return sh.Path, argv, true
                   }
               }
           }

           return "", nil, false
       }
       ```

    4. Modify `CreateSession` body (starting at L204). After `cliPath := e.ResolveCLI(cli)` (L205) and BEFORE the `cols/rows` clamping, insert the shell-resolution branch:
       ```go
       cliPath := e.ResolveCLI(cli)

       // Shell-session dispatch: SHELL-04/05.
       // Replaces cliPath, args, and (when empty) workDir for shell-type cli values.
       // AI CLI sessions skip this branch entirely.
       if path, shellArgs, isShell := e.resolveShellSpawn(cli); isShell {
           cliPath = path
           args = shellArgs // per Anti-Patterns: ignore caller-supplied req.Args for shells in Phase 100
           if workDir == "" {
               if home, err := os.UserHomeDir(); err == nil && home != "" {
                   workDir = home
               }
               // If UserHomeDir fails, fall through with workDir == "" (existing behavior).
               // This is rare (no $HOME, no /etc/passwd entry) and not worth erroring on.
           }
       }
       ```

       Note: AI-CLI sessions (cli="claude", etc.) reach `isShell == false`, so cliPath/args/workDir are unchanged — TestCreateSession_AICLIEmptyWorkDirUnchanged guards this.

    5. Modify the `go status.Watch(...)` call at L245 — wrap in the SHELL-09 guard:
       ```go
       // SHELL-09: shells have no AI-agent state model. Skip status.Watch so
       // sessionStatuses[id] stays empty and ListSessions falls through to its
       // conservative "running" default (engine.go ListSessions branch).
       if !isShellSession(cli) {
           go status.Watch(hub, id, cli, func(sid string, s status.SessionStatus) {
               e.statusMu.Lock()
               e.sessionStatuses[sid] = s
               e.statusMu.Unlock()
               if onStatus != nil {
                   onStatus(sid, s)
               }
           })
       }
       ```

    6. Add imports if not already present: `"os"`, `"path/filepath"`, `"strings"`, and `"github.com/scottkw/agenthub/internal/pty"` (which should already be imported). Run `goimports -w internal/daemon/engine.go`.

    7. Run `go fmt ./internal/daemon/...` and `go vet ./internal/daemon/...`.

    No other changes. Do NOT modify `internal/pty/native.go` (Pitfall 3). Do NOT add a `Type` field to `SessionInfo` (deferred to Phase 101 per Open Question #2 in RESEARCH.md).
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/daemon -run 'Shell|IsShellSession|ResolveShellSpawn|ListSessions_ShellStatus|AICLIEmptyWorkDir' -race -count=1 -v && go test ./internal/daemon -race -count=1 -run 'TestEngine|TestCreateSession_OpenCode'</automated>
  </verify>
  <acceptance_criteria>
    - File internal/daemon/engine.go contains func isShellSession
    - File contains `func (e *SessionEngine) resolveShellSpawn(`
    - knownShells map contains entry `"pwsh": true` (verify via: grep -c '"pwsh":' internal/daemon/engine.go returns >= 1)
    - knownShells map contains entry `"powershell": true`
    - knownShells map contains entry `"pwsh.exe": true`
    - knownShells map contains entry `"powershell.exe": true`
    - engine.go (excluding comments) contains `if !isShellSession(cli) {` immediately before `go status.Watch(` (verify via grep -B1 'go status.Watch' internal/daemon/engine.go | grep -q 'if !isShellSession')
    - `go test ./internal/daemon -run TestCreateSession_ShellArgv_Interactive -race -count=1` exits 0
    - `go test ./internal/daemon -run TestCreateSession_ZshArgv_Interactive -race -count=1` exits 0
    - `go test ./internal/daemon -run TestCreateSession_ShellWorkDirHonored -race -count=1` exits 0
    - `go test ./internal/daemon -run TestCreateSession_ShellEmptyWorkDirHome -race -count=1` exits 0
    - `go test ./internal/daemon -run TestCreateSession_AICLIEmptyWorkDirUnchanged -race -count=1` exits 0
    - `go test ./internal/daemon -run TestCreateSession_ShellSkipsStatusWatch -race -count=1` exits 0
    - `go test ./internal/daemon -run TestListSessions_ShellStatusRunning -race -count=1` exits 0
    - `go test ./internal/daemon -run TestShell_NoStatusMapEntry -race -count=1` exits 0
    - `go test ./internal/daemon -run TestIsShellSession_AllShellNames -race -count=1` exits 0
    - `go test ./internal/daemon -run TestResolveShellSpawn_KnownShell -race -count=1` exits 0
    - `go test ./internal/daemon -run TestResolveShellSpawn_PowerShellOverride -race -count=1` exits 0 (M2 contract)
    - Existing test `go test ./internal/daemon -run TestCreateSession_OpenCodeEnv -race -count=1` still exits 0 (no regression to AI CLI dispatch)
    - `go vet ./internal/daemon/...` exits 0
    - `gofmt -l internal/daemon/engine.go` is empty
  </acceptance_criteria>
  <done>engine.go and engine_test.go committed. All shell-spawn and status-bypass tests pass under -race, including M2's TestResolveShellSpawn_PowerShellOverride. Existing TestCreateSession_OpenCodeEnv passes unchanged (AI CLI dispatch unaffected). SHELL-05 and SHELL-09 satisfied at the engine layer.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Session API caller → engine.CreateSession | The HTTP/Wails caller specifies `cli` and `workDir`. After this plan, certain `cli` values (`bash`/`zsh`/`pwsh`/`powershell`/`shell`) cause the daemon to spawn shell binaries with interactive argv. This expands the command-execution surface compared to AI-CLI-only behavior. |
| engine.cliPaths override map → spawn | A stale or attacker-controlled `cliPaths["claude"] = "/bin/sh"` setting could route shell binaries through AI-CLI argv (no `-i`, no shell-specific defaults). Plan mitigates via extended `knownShells` map filtering on settings load. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-100-01 | Elevation of Privilege | `engine.CreateSession` shell dispatch | mitigate | Per RESEARCH.md T-100-01: daemon spawns ONLY binaries returned by `pty.DiscoverShells()` (which uses `exec.LookPath` against `knownShellSpecs` allowlist — see Plan 01 threat model). Caller-supplied argv (`req.Args`) is IGNORED for shell sessions in this phase (per Anti-Patterns in RESEARCH.md and the `args = shellArgs` line in Task 2 — overwrites, doesn't merge). No caller-supplied path injection: cli is matched against the static allowlist (`shell|bash|zsh|pwsh|powershell`) via `isShellSession`. |
| T-100-02 | Information Disclosure | `workDir` traversal | accept | Per RESEARCH.md T-100-02: daemon honors but does not validate WorkDir (matches existing AI-CLI session behavior). Daemon API access implies host trust. Empty WorkDir defaults to `$HOME` for shells (safer than daemon CWD which can be `/` in service mode); for AI CLIs, existing behavior is preserved. |
| T-100-06 | Tampering | Stale CLI path override (Pitfall 6) | mitigate | Extended `knownShells` map (L97) now includes `pwsh`, `pwsh.exe`, `powershell`, `powershell.exe` so a stale `cliPaths["claude"] = "/bin/zsh"` or `cliPaths["claude"] = "C:\\...\\pwsh.exe"` is dropped on settings load (existing filter in `loadSettingsFromDisk` keys on basename). |
| T-100-07 | Denial of Service | Status-watch goroutine leak | mitigate | Guard at `go status.Watch` call site ensures no goroutine is spawned for shell sessions — saves goroutine + channel buffer per session. Plan 01's `DiscoverShells` is called O(1) per CreateSession (acceptable). |
| T-100-08 | Information Disclosure | shell argv contains user-supplied args | mitigate | `args = shellArgs` line overwrites caller-supplied `req.Args` for shell sessions (Phase 100 Anti-Pattern compliance). No argv injection possible through the HTTP API for shells. AI CLI sessions retain existing `req.Args` pass-through (existing behavior; existing threat model). |
</threat_model>

<verification>
After both tasks complete:

```bash
# Plan-scoped quick gate (matches VALIDATION.md sampling rate):
go test ./internal/daemon -run 'Shell|IsShellSession|ResolveShellSpawn|ListSessions_ShellStatus|AICLIEmptyWorkDir' -race -count=1

# Regression gate (existing daemon tests):
go test ./internal/daemon -race -count=1

# Cross-package quick gate (run before Plan 04 starts):
go test ./internal/pty/... ./internal/daemon/... -race -count=1

# Format / vet / lint:
go vet ./internal/daemon/...
gofmt -l internal/daemon/engine.go internal/daemon/engine_test.go  # both empty

# Pattern preservation:
grep -B1 'go status.Watch(' internal/daemon/engine.go | grep -q 'if !isShellSession(cli)'  # SHELL-09 guard wired
```

Validation map IDs satisfied: TBD-05-bash-argv, TBD-05-zsh-argv, TBD-05-pwsh-argv, TBD-05-powershell-override (M2), TBD-05-workdir-pass, TBD-05-workdir-default-home, TBD-09-no-watch, TBD-09-list-running, TBD-09-no-status-map.
</verification>

<success_criteria>
- Two files modified: `internal/daemon/engine.go`, `internal/daemon/engine_test.go`
- All 13 new tests from Task 1 pass under -race
- No existing daemon test regresses (`go test ./internal/daemon -race -count=1` exits 0)
- Cross-platform: tests POSIX-only behavior are properly skipped on Windows; pwsh-specific tests skip when pwsh is unavailable
- SHELL-05: shell sessions spawn interactive (-i / -NoLogo), non-login (no -l/--login), with WorkDir honored, $HOME fallback for shells only
- SHELL-05 (M2): cliPaths["powershell"] override resolves via knownShellSpecs match without falling through to discovery
- SHELL-09: `go status.Watch` not invoked for shell sessions; `ListSessions` returns Status="running" for live shell, Status="stopped" after exit; sessionStatuses map has no shell entries
- Plan 04 (HTTP route) can depend on these changes being in place to write integration tests that exercise the full create→list path for shell sessions
</success_criteria>

<output>
After completion, create `.planning/phases/100-shell-session-backend-discovery/100-02-SUMMARY.md` documenting:
- New helpers added (signatures of `isShellSession`, `resolveShellSpawn`) and their location in engine.go
- Diff summary of `CreateSession` modifications (shell-resolution branch, status.Watch guard)
- Extended `knownShells` map keys (pwsh + .exe forms)
- Test coverage map: which new test verifies which requirement (highlight M2's TestResolveShellSpawn_PowerShellOverride)
- Confirmation that AI-CLI dispatch path is unchanged (TestCreateSession_OpenCodeEnv green, TestCreateSession_AICLIEmptyWorkDirUnchanged green)
- Any deviations from RESEARCH.md/PATTERNS.md with rationale
</output>
</content>
</invoke>