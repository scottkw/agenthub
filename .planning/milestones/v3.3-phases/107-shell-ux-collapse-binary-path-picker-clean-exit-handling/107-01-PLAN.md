---
phase: 107
plan: "107-01"
type: execute
status: pending
wave: 0
depends_on: []
requirements: [SHELL-11]
files_modified:
  - internal/daemon/engine.go
  - internal/daemon/api.go
  - internal/daemon/client.go
  - app.go
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - internal/daemon/engine_test.go
  - internal/daemon/api_test.go
autonomous: true
must_haves:
  truths:
    - "Daemon persists `shellPath` field across restart through settings.json round-trip."
    - "GET /settings/shell-path returns the resolved default when the user has not set anything (never empty)."
    - "PATCH /settings/shell-path with a valid executable path persists it and returns 204."
    - "PATCH /settings/shell-path with a non-existent or non-executable path returns HTTP 400 with a human-readable body."
    - "CreateSession for `cli=\"shell\"` uses the persisted shellPath when set; falls back to platform default when unset."
    - "Wails layer exposes `GetShellPath()` / `SetShellPath(path)` callable from the frontend."
  artifacts:
    - path: internal/daemon/engine.go
      provides: "ShellPath settings field + Get/SetShellPath methods + resolveShellSpawn override branch"
      contains: "shellPath"
    - path: internal/daemon/api.go
      provides: "GET/PATCH /settings/shell-path routes + handlers with executable validation"
      contains: "handleGetShellPath"
    - path: internal/daemon/client.go
      provides: "GetShellPath()/SetShellPath() DaemonClient methods"
      contains: "GetShellPath"
    - path: app.go
      provides: "App.GetShellPath()/App.SetShellPath(path) Wails wrappers"
      contains: "GetShellPath"
    - path: frontend/src/wailsjs/go/main/App.d.ts
      provides: "TS bindings for GetShellPath/SetShellPath"
      contains: "GetShellPath"
    - path: frontend/src/wailsjs/go/main/App.js
      provides: "Runtime Call() bindings for GetShellPath/SetShellPath"
      contains: "GetShellPath"
  key_links:
    - from: app.go GetShellPath/SetShellPath
      to: client.go GetShellPath/SetShellPath
      via: a.client.GetShellPath()
    - from: client.go GetShellPath/SetShellPath
      to: api.go /settings/shell-path
      via: doJSON HTTP call
    - from: api.go handlers
      to: engine.go Get/SetShellPath
      via: a.engine.SetShellPath(req.Path)
    - from: engine.resolveShellSpawn
      to: engine.shellPath field
      via: "override resolution before falling back to discovery"
---

<objective>
SHELL-11 backend: plumb a persisted `shellPath` setting through daemon -> HTTP API -> DaemonClient -> Wails app facade -> TS bindings, with executable-validation on PATCH. Wired into shell-session spawn so CreateSession honors the user's chosen binary. Mirrors the existing `shellWebShareWarned` plumbing pattern verbatim and pairs with 107-03 (frontend Settings field) in wave 1.

Purpose: Give the user a single source of truth for "which shell binary do new shell sessions launch". Auto-discovery is no longer sufficient (per CONTEXT.md: SHELL-11 reverses the original "Custom shell binary path picker — out of scope" decision after first-user test).

Output: Backend persistence + RPC surface. No UI yet (107-03 handles that). Existing CLI `agenthub new shell --shell=NAME` and modal flow continue to work; the new shellPath is consumed as an additional override layer on top of `cliPaths[cli]` discovery.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-CONTEXT.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-UI-SPEC.md

@internal/daemon/engine.go
@internal/daemon/api.go
@internal/daemon/client.go
@app.go

<interfaces>
Existing analogue we are mirroring — copy this verbatim, substituting `shellPath`/`Path` for `shellWebShareWarned`/`Value`:

From internal/daemon/engine.go (settings struct):
```go
type daemonSettings struct {
    CLIPaths            map[string]string `json:"cliPaths,omitempty"`
    StartMinimized      bool              `json:"startMinimized,omitempty"`
    ShellWebShareWarned bool              `json:"shellWebShareWarned,omitempty"`
    AutoCloseSession    *bool             `json:"autoCloseSession,omitempty"`
    Plugins             PluginSettings    `json:"plugins"`
    SchemaVersion       int               `json:"schemaVersion"`
}
```

From internal/daemon/engine.go (engine fields ~L37-41):
```go
startMinimized      bool
shellWebShareWarned bool
autoCloseSession    *bool
pluginSettings      PluginSettings
```

From internal/daemon/engine.go (~L583-599) — exact pattern to mirror:
```go
func (e *SessionEngine) GetShellWebShareWarned() bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.shellWebShareWarned
}
func (e *SessionEngine) SetShellWebShareWarned(val bool) error {
    e.mu.Lock()
    e.shellWebShareWarned = val
    e.saveSettingsToDisk()
    e.mu.Unlock()
    return nil
}
```

From internal/daemon/api.go (~L73-74) — route registration pair:
```go
a.mux.HandleFunc("GET /settings/shell-web-share-warned", a.handleGetShellWebShareWarned)
a.mux.HandleFunc("PATCH /settings/shell-web-share-warned", a.handleUpdateShellWebShareWarned)
```

From internal/daemon/api.go (~L545-564) — handler pair:
```go
func (a *API) handleGetShellWebShareWarned(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]bool{"value": a.engine.GetShellWebShareWarned()})
}
func (a *API) handleUpdateShellWebShareWarned(w http.ResponseWriter, r *http.Request) {
    var req struct { Value bool `json:"value"` }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    if err := a.engine.SetShellWebShareWarned(req.Value); err != nil {
        http.Error(w, fmt.Sprintf("persist: %v", err), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

From internal/daemon/client.go (~L142-155):
```go
func (c *DaemonClient) GetShellWebShareWarned() (bool, error) {
    var resp map[string]bool
    if err := c.doJSON(http.MethodGet, "/settings/shell-web-share-warned", nil, &resp); err != nil {
        return false, err
    }
    return resp["value"], nil
}
func (c *DaemonClient) SetShellWebShareWarned(val bool) error {
    return c.doJSON(http.MethodPatch, "/settings/shell-web-share-warned",
        map[string]bool{"value": val}, nil)
}
```

From app.go (~L421-440):
```go
func (a *App) GetShellWebShareWarned() bool { /* …if a.client == nil return false */ }
func (a *App) SetShellWebShareWarned(v bool) error { /* …if a.client == nil return nil */ }
```

From engine.go resolveShellSpawn (~L487-540) — the function we must extend so the new shellPath override is honored. It currently consults `e.ResolveCLI(cli)` which looks at `e.cliPaths`. The new shellPath is consulted for `cli == "shell"` (the bare system-default key) and ONLY when `cliPaths["shell"]` is unset (cliPaths still wins if both are set, preserving per-binary overrides like `cliPaths["bash"]`).
</interfaces>

</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add shellPath field + Get/SetShellPath methods + settings round-trip</name>
  <files>internal/daemon/engine.go, internal/daemon/engine_test.go</files>
  <behavior>
    - daemonSettings struct gains `ShellPath string \`json:"shellPath,omitempty"\`` field.
    - SessionEngine gains `shellPath string` field beside `shellWebShareWarned`.
    - loadSettingsFromDisk copies `s.ShellPath` into `e.shellPath` (mirror line 165 pattern).
    - saveSettingsToDisk includes `ShellPath: e.shellPath` (mirror line 190 pattern).
    - GetShellPath() returns current value, OR — when empty — returns a resolved platform default by calling a new internal helper `resolveDefaultShellPath()` that: (1) reads `$SHELL` env; (2) on POSIX falls back to `pty.DiscoverShells()` first entry whose Name=="shell" (the synthetic system-default); (3) hard-fallback `/bin/zsh` on darwin, `/bin/bash` on linux, `pwsh.exe` on windows. NEVER returns empty string.
    - SetShellPath(path string) error: when path is empty, clears the field (allowing "use system default"); otherwise validates via `os.Stat` (exists) AND `info.Mode()&0111 != 0` (executable bit). On validation failure return `fmt.Errorf("path %q does not exist or is not executable", path)`. On success, set field + saveSettingsToDisk under mutex.
    - Tests in engine_test.go (mirror TestSetShellWebShareWarned_Default + _Persists patterns at lines 897-940):
      * TestGetShellPath_DefaultResolvesPlatformDefault — fresh engine returns non-empty string matching `os.Getenv("SHELL")` OR one of the hardcoded platform defaults.
      * TestSetShellPath_RejectsMissingPath — `SetShellPath("/no/such/path")` returns error; `e.shellPath` unchanged.
      * TestSetShellPath_RejectsNonExecutable — create tempfile with 0644 perms, SetShellPath returns error.
      * TestSetShellPath_AcceptsExecutable — point at `/bin/sh` (POSIX); confirm `e.shellPath` updated and round-trips via new engine load.
      * TestSetShellPath_EmptyClears — set to "/bin/sh", then SetShellPath(""), confirm GetShellPath falls back to platform default.
  </behavior>
  <action>
    Open internal/daemon/engine.go. (1) Insert `ShellPath string \`json:"shellPath,omitempty"\`` into daemonSettings struct between `AutoCloseSession` and `Plugins`. (2) Insert `shellPath string` engine field beside `shellWebShareWarned` (~L38). (3) In loadSettingsFromDisk after `e.shellWebShareWarned = s.ShellWebShareWarned` add `e.shellPath = s.ShellPath`. (4) In saveSettingsToDisk's struct literal add `ShellPath: e.shellPath,` after `ShellWebShareWarned`. (5) Add new exported method `GetShellPath() string` that locks mu.RLock, reads e.shellPath, if empty calls resolveDefaultShellPath() and returns its result. (6) Add new private method `resolveDefaultShellPath() string` that consults `$SHELL`, then `pty.DiscoverShells()` looking for Name=="shell", then platform hardcode (`runtime.GOOS` switch). (7) Add new exported method `SetShellPath(path string) error` with the validation described above. (8) Extend resolveShellSpawn: at the very top, after the `if !isShellSession(cli)` early-return, add a new "(0) Settings shellPath override" branch BEFORE the existing (1) cliPaths branch: when `cli == "shell"` AND `e.cliPaths["shell"]` is empty AND `e.shellPath` is non-empty, treat e.shellPath as the override and look up its basename against pty.KnownShellSpecs() to derive argv. If the basename doesn't match a known shell spec, fall through to the existing branches (do NOT error — the binary may be a custom shell that still accepts -i). Document this with a comment citing SHELL-11 and "preserves per-binary cliPaths[bash] overrides taking precedence over the catch-all shellPath".

    Write engine_test.go tests using existing `newTestEngine(t *testing.T)` helper if present, or follow TestSetShellWebShareWarned_Persists' direct construction pattern. The "round-trip" test creates engine1, sets shellPath, closes engine1, creates engine2 from same configDir, asserts GetShellPath returns the set value. Use `t.TempDir()` + override `os.UserConfigDir` via `os.Setenv("XDG_CONFIG_HOME", tmp)` (POSIX) — mirror the existing test setup pattern verbatim from TestSetShellWebShareWarned_Persists.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/daemon/ -run 'TestSetShellPath|TestGetShellPath' -v -count=1</automated>
  </verify>
  <done>
    All five new tests pass. `go vet ./internal/daemon/` clean. `go build ./...` clean.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: HTTP routes + DaemonClient methods + Wails app facade + TS bindings</name>
  <files>internal/daemon/api.go, internal/daemon/api_test.go, internal/daemon/client.go, app.go, frontend/src/wailsjs/go/main/App.d.ts, frontend/src/wailsjs/go/main/App.js</files>
  <behavior>
    - GET /settings/shell-path returns `{"value": "<resolved path string>"}` with 200. Body never has an empty `value` (engine resolves default).
    - PATCH /settings/shell-path with body `{"value":"/bin/zsh"}` returns 204; subsequent GET returns the new value.
    - PATCH /settings/shell-path with body `{"value":"/no/such/path"}` returns 400 with plain-text body containing "does not exist or is not executable" (verbatim from engine error).
    - PATCH /settings/shell-path with body `{"value":""}` returns 204 (clears override; subsequent GET returns resolved default).
    - DaemonClient.GetShellPath() returns the resolved string; SetShellPath(path) returns nil on success, error wrapping the daemon's 400 body on failure.
    - App.GetShellPath() returns "" on nil client (parity with GetShellWebShareWarned); App.SetShellPath(v) returns nil on nil client.
    - frontend/src/wailsjs/go/main/App.d.ts declares `GetShellPath(): Promise<string>` and `SetShellPath(v: string): Promise<void>`.
    - App.js exports the matching `Call('main.App.GetShellPath', [])` / `Call('main.App.SetShellPath', [v])` bindings.
  </behavior>
  <action>
    (1) internal/daemon/api.go: register the two routes between lines 73-74 (after the shell-web-share-warned pair) — keep grouping logical. Add `handleGetShellPath(w, r)` returning `map[string]string{"value": a.engine.GetShellPath()}`. Add `handleUpdateShellPath(w, r)` decoding `{ Value string }`; pass to `a.engine.SetShellPath(req.Value)`; on error return `http.Error(w, err.Error(), http.StatusBadRequest)` (NOT 500 — validation failure is a client error). On success, 204. Place both handlers immediately after handleUpdateShellWebShareWarned (~L564). (2) internal/daemon/client.go: add GetShellPath()/SetShellPath() mirroring lines 142-155 but using `map[string]string` for the response shape. (3) app.go: add App.GetShellPath()/App.SetShellPath() mirroring lines 421-440 verbatim — same nil-client guards, same error swallowing. (4) Update frontend/src/wailsjs/go/main/App.d.ts: add `export function GetShellPath(): Promise<string>` and `export function SetShellPath(v: string): Promise<void>` immediately after the existing SetShellWebShareWarned declaration (~L34). Update App.js: add `export const GetShellPath = () => Call('main.App.GetShellPath', [])` and `export const SetShellPath = (v) => Call('main.App.SetShellPath', [v])` immediately after the existing SetShellWebShareWarned binding (~L16).

    (5) Write internal/daemon/api_test.go tests using the existing httptest pattern (look for the existing TestHandleGetShellWebShareWarned* tests in the file as the template):
      * TestHandleGetShellPath_ReturnsDefault — GET returns 200 with non-empty value.
      * TestHandleUpdateShellPath_ValidPath_Persists — PATCH /bin/sh returns 204; follow-up GET returns "/bin/sh".
      * TestHandleUpdateShellPath_InvalidPath_Returns400 — PATCH "/no/such/path" returns 400, body contains "does not exist".
      * TestHandleUpdateShellPath_Empty_ClearsOverride — PATCH "/bin/sh" then PATCH "" both return 204; GET after second returns resolved default (non-empty, not "/bin/sh").
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/daemon/ -run 'TestHandleGetShellPath|TestHandleUpdateShellPath' -v -count=1 && go build ./...</automated>
  </verify>
  <done>
    Four api_test.go tests pass. `go build ./...` succeeds. App.d.ts and App.js export both new symbols (grep -c "GetShellPath" yields exactly 2 in each file: declaration/export + Call binding line).
  </done>
</task>

</tasks>

<verification>
- `go test ./internal/daemon/ -count=1` — full daemon suite green; the new 9 tests pass and no SHELL-01..09 / PLUG-01..04 / SET-01..02 tests regress.
- `go build ./...` — clean (catches App.d.ts/App.js drift since Wails-build typechecks bindings).
- Manual smoke (dev-browser optional, not required for this plan): start daemon, `curl -X PATCH localhost:PORT/settings/shell-path -d '{"value":"/bin/sh"}'` returns 204; `curl localhost:PORT/settings/shell-path` returns the new value.
</verification>

<success_criteria>
- Settings persistence parity: shellPath round-trips through engine restart exactly like shellWebShareWarned (per "Critical invariants" in user prompt).
- No regression of SHELL-01..09: existing `agenthub new shell --shell=bash` CLI continues to work because resolveShellSpawn's branch (0) only fires when `cli == "shell"` (bare key); per-binary `bash`/`zsh`/`pwsh`/`powershell` paths go through the existing branches unchanged.
- Default fallback chain order: `$SHELL` → `pty.DiscoverShells()` Name=="shell" entry → platform hardcode. Resolution happens at GetShellPath() / CreateSession() time, NOT at settings-load (so clearing the field returns to "use system default" semantics).
- Validation runs daemon-side on PATCH; frontend (107-03) receives 400 + plain-text error body to surface verbatim.
</success_criteria>

<output>
After completion, create `.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-01-SUMMARY.md` covering: shellPath plumbing, resolveDefaultShellPath fallback chain, validation contract, files touched, test count, follow-ups for 107-03 (frontend will consume GetShellPath() on mount and SetShellPath() on Save Paths click).
</output>
