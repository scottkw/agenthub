# Phase 100: Shell Session Backend & Discovery — Pattern Map

**Mapped:** 2026-05-12
**Files analyzed:** 9 (2 new, 6 modify, 1 possibly modify)
**Analogs found:** 9 / 9 (every file has an exact-quality in-repo analog)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/pty/shells.go` (NEW) | utility (discovery) | request-response (sync enumeration) | `internal/pty/detect.go` | exact |
| `internal/pty/shells_test.go` (NEW) | test | request-response | `internal/pty/detect_test.go` | exact |
| `internal/daemon/engine.go` (MODIFY) | service (session dispatch) | request-response | self (same file, existing `CreateSession` body) | exact |
| `internal/daemon/engine_test.go` (MODIFY) | test | request-response | self (`spyBackend` at L241 + `TestCreateSession_OpenCodeEnv` at L262) | exact |
| `internal/daemon/types.go` (MODIFY) | model (JSON DTO) | request-response | self (existing `CLIPathsResponse` / `UpdateCLIPathRequest`) | exact |
| `internal/daemon/api.go` (MODIFY) | controller (HTTP route) | request-response | self (`handleGetCLIPaths` at L478 + route table L60-95) | exact |
| `internal/daemon/api_test.go` (MODIFY) | test (integration) | request-response | self (`TestAPIGetCLIPaths` at L269) | exact |
| `internal/daemon/client.go` (MODIFY) | client (HTTP wrapper) | request-response | self (`GetCLIPaths` at L98) | exact |
| `internal/daemon/path_windows.go` (POSSIBLY MODIFY) | config (PATH augmentation) | startup config | self (existing `platformExtraBins`) | exact |

---

## Pattern Assignments

### `internal/pty/shells.go` (NEW — utility, discovery)

**Analog:** `internal/pty/detect.go` (verified 68-line file, mirror exactly)

**Package + imports pattern** (detect.go L1-6):
```go
package pty

import (
	"errors"
	"os/exec"
)
```

For `shells.go` the planner should ADD `runtime`, `bufio`, `os`, `strings` only if `/etc/shells` parsing lands in Wave 0 (Research §Pattern 1). Bare-minimum imports for the recommended scope (PATH discovery + GOOS branching):
```go
package pty

import (
	"os/exec"
	"runtime"
)
```

**Spec-struct pattern** (detect.go L11-22, MIRROR EXACTLY for shells):
```go
// CLISpec describes a known CLI tool that AgentHub can launch.
type CLISpec struct {
	Name        string
	DisplayName string
}

// DetectedCLI is a CLISpec whose binary was found on PATH.
type DetectedCLI struct {
	Name        string
	DisplayName string
	Path        string
}
```

Shells need one extra field — `Argv []string` — because shells require argv flags (`-i`, `-NoLogo`) where AI CLIs don't. New shape:
```go
type ShellSpec struct {
	Name        string
	DisplayName string
	Argv        []string
}
type DetectedShell struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Path        string   `json:"path"`
	Argv        []string `json:"argv"`
}
```
Note JSON tags: `detect.go`'s `DetectedCLI` has none because the CLI list is not yet wire-exposed; `DetectedShell` IS wire-exposed via `GET /shells`, so tags are required.

**Known-list pattern** (detect.go L24-30, MIRROR EXACTLY):
```go
var knownCLIs = []CLISpec{
	{Name: "claude", DisplayName: "Claude Code"},
	{Name: "codex", DisplayName: "OpenAI Codex"},
	{Name: "gemini", DisplayName: "Gemini CLI"},
	{Name: "opencode", DisplayName: "OpenCode"},
}
```
Apply as `knownShellSpecs` per Research §Pattern 1 (bash/zsh/pwsh).

**Discovery-loop pattern** (detect.go L32-48 — the core pattern to copy verbatim):
```go
func DetectCLIs() []DetectedCLI {
	result := make([]DetectedCLI, 0)
	for _, spec := range knownCLIs {
		path, err := exec.LookPath(spec.Name)
		if err != nil {
			continue
		}
		result = append(result, DetectedCLI{
			Name:        spec.Name,
			DisplayName: spec.DisplayName,
			Path:        path,
		})
	}
	return result
}
```
Two semantic preservation points:
1. `make([]T, 0)` — non-nil empty slice (NOT `var result []T`). The detect.go author explicitly chose this; mirror it. Test `TestDetectCLIs_AllMissing` at detect_test.go L57-69 enforces non-nil.
2. `continue` on `LookPath` error — silent skip is the project pattern for binary discovery.

**Lookup-by-name pattern** (detect.go L50-68 — copy for `DetectShell(name)`):
```go
var ErrCLINotFound = errors.New("CLI not found")

func DetectCLI(name string) (*DetectedCLI, error) {
	for _, spec := range knownCLIs {
		if spec.Name != name {
			continue
		}
		path, err := exec.LookPath(spec.Name)
		if err != nil {
			return nil, ErrCLINotFound
		}
		return &DetectedCLI{...}, nil
	}
	return nil, ErrCLINotFound
}
```
Apply as `DetectShell(name string) (*DetectedShell, error)` with `ErrShellNotFound`.

**GOOS-branch pattern** (already in repo at `native.go:62`, but inline-conditional in detect.go is absent — this is NEW shape for shells.go). The closest sibling pattern is `native.go:62`:
```go
// native.go:60-69 — runtime.GOOS branch shape
if err := p.Resize(req.Cols, req.Rows); err != nil {
	if runtime.GOOS == "windows" {
		log.Printf("[warn] pty resize on Windows after Start: %v", err)
	} else {
		cancel()
		_ = p.Close()
		return nil, fmt.Errorf("initial resize: %w", err)
	}
}
```
For `shells.go`: use `if runtime.GOOS == "windows" { … fallback to powershell.exe … }` and `if runtime.GOOS != "windows" { … $SHELL / /etc/shells supplement … }` blocks AFTER the main `knownShellSpecs` loop. See Research §Pattern 1 lines 207-232.

---

### `internal/pty/shells_test.go` (NEW — test)

**Analog:** `internal/pty/detect_test.go` (90-line file, mirror exactly)

**Package + imports pattern** (detect_test.go L1-8):
```go
package pty

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)
```

**PATH-mock fixture pattern** (detect_test.go L12-40 — this is the load-bearing test pattern):
```go
func TestDetectCLIs_FindsInstalledCLIs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell script stubs not executable on Windows")
	}
	dir := t.TempDir()

	// Write a stub "claude" executable.
	stubPath := filepath.Join(dir, "claude")
	if err := os.WriteFile(stubPath, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}

	t.Setenv("PATH", dir)

	result := DetectCLIs()

	var found bool
	for _, cli := range result {
		if cli.Name == "claude" {
			found = true
			if cli.Path == "" {
				t.Error("expected non-empty Path for claude")
			}
		}
	}
	if !found {
		t.Error("expected DetectCLIs to find claude, but it was not in results")
	}
}
```

Critical setup details to copy verbatim:
- `t.TempDir()` for an isolated PATH dir
- Stub file mode `0755` (executable)
- Stub content `#!/bin/sh\necho ok\n` is enough — `exec.LookPath` only checks `+x`, never executes
- `t.Setenv("PATH", dir)` — auto-restored on test exit
- Windows skip with explicit reason

**Empty-PATH assertion pattern** (detect_test.go L43-53):
```go
func TestDetectCLIs_SkipsMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	result := DetectCLIs()
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}
```

**Non-nil-slice assertion pattern** (detect_test.go L57-69):
```go
func TestDetectCLIs_AllMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	result := DetectCLIs()
	if result == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected zero entries, got %d", len(result))
	}
}
```
**Apply this verbatim for `TestDiscoverShells_AllMissing`** — Research's Pitfall 2 (`/etc/shells` missing on slim containers) makes the non-nil guarantee load-bearing.

**Known-list assertion pattern** (detect_test.go L73-90):
```go
func TestKnownCLIs_HasExpectedEntries(t *testing.T) {
	expected := []string{"claude", "codex", "gemini", "opencode"}
	if len(knownCLIs) != len(expected) {
		t.Fatalf("expected %d known CLIs, got %d", len(expected), len(knownCLIs))
	}
	nameSet := make(map[string]bool, len(knownCLIs))
	for _, spec := range knownCLIs {
		nameSet[spec.Name] = true
	}
	for _, name := range expected {
		if !nameSet[name] {
			t.Errorf("expected knownCLIs to contain %q", name)
		}
	}
}
```
**Apply for `TestKnownShellSpecs_HasExpectedEntries`** with `expected := []string{"bash", "zsh", "pwsh"}`.

**Windows-specific test pattern (NEW shape — no existing analog):** Research §"Per-Phase Requirements → Test Map" SHELL-04 row notes Windows pwsh.exe discovery needs a build-tagged GOOS test. There is no existing build-tagged test in `internal/pty/` to mirror — recommend gating with `if runtime.GOOS != "windows" { t.Skip(...) }` at the top of the test function (the inverse of detect_test.go L13-14 pattern), NOT a `//go:build windows` file split.

---

### `internal/daemon/engine.go` (MODIFY — service)

**Analog:** self (the file being modified). Three insertion points to verify against existing patterns in the same file.

**Insertion point 1: `resolveShellSpawn` helper** (NEW function, place above `CreateSession` at L200). No existing analog for "argv resolver" but the closest shape pattern is `ResolveCLI` at L386-395:
```go
// engine.go:386-395 — pattern for "look up by name, return resolved value"
func (e *SessionEngine) ResolveCLI(name string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if path, ok := e.cliPaths[name]; ok {
		return path
	}
	return name
}
```
Apply: `resolveShellSpawn(cli string) (path string, args []string, ok bool)` — Research §Pattern 2 (lines 244-261) gives the full body. Note the `ok bool` return signals "not a shell" without an error type (consistent with `ResolveCLI`'s "return name as-is" fallthrough).

**Insertion point 2: argv + env application inside `CreateSession`** (modify L214-227). Current code:
```go
// engine.go:214-227 — current CreateSession body to modify
// Per-agent environment configuration.
var env []string
if cli == "opencode" && e.opencodeTUIConfig != "" {
	env = append(env, "OPENCODE_TUI_CONFIG="+e.opencodeTUIConfig)
}

sess, err := e.backend.Create(ctx, pty.CreateRequest{
	CLI:     cliPath,
	Args:    args,
	Env:     env,
	Cols:    cols,
	Rows:    rows,
	WorkDir: workDir,
})
```

The `cli == "opencode"` block is THE precedent for "per-CLI-name customization at this layer." For shells, the planner inserts a parallel block:
```go
// After cliPath := e.ResolveCLI(cli) at L205,
// BEFORE the env block at L215:
if path, shellArgs, isShell := e.resolveShellSpawn(cli); isShell {
	cliPath = path
	args = shellArgs       // Research Anti-Patterns: do NOT merge req.args for shells in Phase 100
	if workDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			workDir = home  // Pitfall 4 mitigation
		}
	}
}
```

**Insertion point 3: status.Watch guard** (modify L245-252). Current code:
```go
// engine.go:245-252 — wrap this go-call in an if-guard
go status.Watch(hub, id, cli, func(sid string, s status.SessionStatus) {
	e.statusMu.Lock()
	e.sessionStatuses[sid] = s
	e.statusMu.Unlock()
	if onStatus != nil {
		onStatus(sid, s)
	}
})
```
Apply Research §Pattern 3 (lines 277-291): wrap in `if !isShellSession(cli) { ... }`. The conservative-default at L308 (`heuristicStatus := string(status.StatusRunning)`) absorbs the absent map entry — this is the SHELL-09 mechanism that "falls out for free."

**`knownShells` map extension (engine.go L97-100):** The defensive map already exists:
```go
// engine.go:94-100 — existing defensive filter
var knownShells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "fish": true,
	"csh": true, "tcsh": true, "dash": true, "ksh": true,
}
```
Per Research Pitfall 6: add `"pwsh": true`, `"pwsh.exe": true`, `"powershell": true`, `"powershell.exe": true`. Note the `.exe` form is necessary because on Windows `filepath.Base("/path/to/pwsh.exe")` returns `pwsh.exe`, not `pwsh`.

**Sibling `isShellSession` helper:** No existing helper of this exact shape. Closest analog is the inline `if cli == "opencode"` check on engine.go L216. Add as a package-level helper for testability:
```go
func isShellSession(cli string) bool {
	switch cli {
	case "shell", "bash", "zsh", "pwsh", "powershell":
		return true
	}
	return false
}
```

---

### `internal/daemon/engine_test.go` (MODIFY — test)

**Analog:** self. The `spyBackend` harness at L241-257 is the load-bearing fixture and `TestCreateSession_OpenCodeEnv` at L262-320 is the role-model test.

**spyBackend harness pattern** (engine_test.go L239-257 — reuse verbatim, no changes needed):
```go
// spyBackend records the CreateRequest from the most recent Create call.
// Used by Wave 0 tests to assert on env injection without launching a real PTY.
type spyBackend struct {
	lastReq pty.CreateRequest
}

func (s *spyBackend) Create(_ context.Context, req pty.CreateRequest) (*pty.Session, error) {
	s.lastReq = req
	return &pty.Session{
		ID:        "spy-id",
		CLI:       req.CLI,
		State:     pty.StateRunning,
		CreatedAt: time.Now(),
	}, nil
}

func (s *spyBackend) Resize(string, int, int) error { return nil }
func (s *spyBackend) Kill(string) error             { return nil }
func (s *spyBackend) List() []*pty.Session          { return nil }
```

**Test fixture wiring pattern** (engine_test.go L262-270 — copy verbatim, change cli arg):
```go
func TestCreateSession_OpenCodeEnv(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	_, err := e.CreateSession(context.Background(), "opencode", "test-oc", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(opencode): %v", err)
	}
```

**Args-on-spy assertion pattern** (no exact existing analog — the OpenCode test asserts on `Env`, not `Args`). Closest pattern is the OpenCode env-iteration at engine_test.go L273-287:
```go
// L273-287 — pattern for iterating a captured field on spy
var found bool
for _, entry := range spy.lastReq.Env {
	if strings.HasPrefix(entry, "OPENCODE_TUI_CONFIG=") {
		found = true
		wantEnv := "OPENCODE_TUI_CONFIG=" + e.opencodeTUIConfig
		if entry != wantEnv {
			t.Errorf("env var = %q, want %q", entry, wantEnv)
		}
		break
	}
}
if !found {
	t.Errorf("CreateSession(opencode): expected OPENCODE_TUI_CONFIG in Env, got %v", spy.lastReq.Env)
}
```
Apply the same iteration shape on `spy.lastReq.Args` per Research §Code Examples lines 380-407.

**WorkDir assertion pattern (no exact existing test — synthesized from spy field access):**
```go
if spy.lastReq.WorkDir != "/home/user" {
	t.Errorf("WorkDir = %q, want %q", spy.lastReq.WorkDir, "/home/user")
}
```

**Status-bypass assertion pattern** (no exact existing test for this shape — use engine_test.go's `TestEngineGetSessionStatus` at L138-159 as the closest reference):
```go
// L138-159 — pattern for asserting session-status via engine accessor
s := e.GetSessionStatus("nonexistent-id")
if s != "running" {
	t.Errorf("unknown session status: got %q, want %q", s, "running")
}
```

For shell-skip test, combine `spyBackend` + `e.GetSessionStatus(id)` + a read of `e.sessionStatuses` map under `e.statusMu.RLock()` (see L384-388 pattern below from `TestSessionCLIs_TrackedAndCleanedUp`).

**Engine-internal-state inspection pattern** (engine_test.go L384-388 — useful for `TestShell_NoStatusMapEntry`):
```go
// Reach into engine internals to verify map state
e.mu.RLock()
cli, ok := e.sessionCLIs[id]
e.mu.RUnlock()
```
Apply with `e.statusMu.RLock()` + `e.sessionStatuses[id]` membership check.

---

### `internal/daemon/types.go` (MODIFY — model)

**Analog:** self. The existing `CLIPathsResponse` at L49-50 and `StatusResponse` at L38-41 are the wire-DTO templates.

**Simple-map DTO pattern** (types.go L49-50):
```go
// CLIPathsResponse maps CLI name to custom path override.
type CLIPathsResponse map[string]string
```

**Wrapper-struct DTO pattern** (types.go L38-41):
```go
// StatusResponse is the response body for GET /sessions/{id}/status.
type StatusResponse struct {
	Status string `json:"status"`
}
```

**Apply for ShellsResponse:** the shell discovery returns a structured list (not a map), so use a wrapper struct. Add to types.go (recommended placement: directly after `CLIPathsResponse` at L50):
```go
// DetectedShell is the JSON-serialisable representation of a discovered shell.
// Mirror of internal/pty.DetectedShell — duplicated to keep daemon's JSON
// wire types decoupled from internal/pty's Go API surface (per the project
// pattern of NOT importing internal/pty types into wire responses; see
// SessionInfo at L4-16 which copies pty.Session fields rather than embedding).
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
**Justification for type duplication:** `SessionInfo` (types.go L4-16) does NOT embed `pty.Session` — it copies fields with JSON tags. Maintaining this separation prevents accidental wire-format coupling to internal types. Document the parallel in a comment.

---

### `internal/daemon/api.go` (MODIFY — controller)

**Analog:** self. Two insertion points, both with direct in-file precedents.

**Route-registration pattern** (api.go L60-95 — add one line in `registerRoutes`):
```go
// api.go:68 — closest sibling registration
a.mux.HandleFunc("GET /settings/cli-paths", a.handleGetCLIPaths)
```
Apply: `a.mux.HandleFunc("GET /shells", a.handleListShells)` — recommended placement: near the discovery-adjacent routes, e.g. directly above L68 or grouped with the new shell routes anticipated for Phase 101.

**Read-only handler pattern** (api.go L478-484 — copy verbatim):
```go
func (a *API) handleGetCLIPaths(w http.ResponseWriter, r *http.Request) {
	paths := a.engine.GetCLIPaths()
	if paths == nil {
		paths = map[string]string{}
	}
	writeJSON(w, http.StatusOK, paths)
}
```
Apply as:
```go
func (a *API) handleListShells(w http.ResponseWriter, r *http.Request) {
	shells := pty.DiscoverShells()
	writeJSON(w, http.StatusOK, ShellsResponse{Shells: convertShells(shells)})
}
```
Note: `pty.DiscoverShells` already returns a non-nil slice (mirroring `DetectCLIs` at detect.go L35), so no nil-guard is needed — but adapting `[]pty.DetectedShell` → `[]daemon.DetectedShell` requires a tiny converter (parallel to the manual `SessionInfo` construction at engine.go L327-338).

**Non-engine handler precedent:** This is one of the few handlers that does NOT delegate to the engine (it calls `pty.*` directly). The closest precedent is the engine-bypass discovery pattern — verify by reading `api.go` around its existing direct-call sites. If the planner prefers engine-mediation (for testability with spyBackend / DI), add `engine.DiscoverShells()` as a thin wrapper that calls `pty.DiscoverShells()`.

---

### `internal/daemon/api_test.go` (MODIFY — test)

**Analog:** self. `TestAPIGetCLIPaths` at L269-282 is the direct template.

**Test-daemon fixture pattern** (api_test.go L26-47 — reuse verbatim, do not reimplement):
```go
func testDaemon(t *testing.T) (*API, *DaemonClient, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	engine := NewSessionEngine()
	engine.configDir = t.TempDir()
	engine.cliPaths = make(map[string]string)
	engine.startMinimized = false
	api := NewAPI(engine)
	socketPath := shortSocketPath(t, "api.sock")
	if err := api.Start(socketPath); err != nil {
		t.Fatalf("api.Start: %v", err)
	}
	t.Cleanup(func() { api.Stop() })
	client := NewDaemonClient(socketPath)
	time.Sleep(10 * time.Millisecond)
	return api, client, socketPath
}
```
**Critical:** macOS Unix-socket path-length limit (103 chars) — use `shortSocketPath`, NOT `filepath.Join(t.TempDir(), "api.sock")`.

**rawGet helper pattern** (api_test.go L57-67 — reuse verbatim):
```go
func rawGet(t *testing.T, socketPath, path string) (int, []byte) {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{DialContext: dialUnix(socketPath)}}
	resp, err := client.Get("http://daemon" + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}
```

**Read-only endpoint test pattern** (api_test.go L269-282 — copy and adapt):
```go
func TestAPIGetCLIPaths(t *testing.T) {
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/settings/cli-paths")
	if status != 200 {
		t.Errorf("GET /settings/cli-paths: want 200, got %d", status)
	}
	var paths map[string]string
	if err := json.Unmarshal(body, &paths); err != nil {
		t.Fatalf("decode cli paths: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty map, got %v", paths)
	}
}
```
Apply as `TestHandleListShells` — replace path with `/shells`, decode into `ShellsResponse`, assert `len(resp.Shells) >= 0` (the discovery result depends on the test host PATH; on dev macOS bash + zsh exist, on CI Linux runner the result varies — keep assertions loose unless using PATH-mock).

For tighter assertion, the planner can override `PATH` with `t.Setenv("PATH", t.TempDir())` BEFORE calling `testDaemon(t)` so discovery returns empty deterministically. (`exec.LookPath` reads PATH at call time, not at process start, so this works.)

---

### `internal/daemon/client.go` (MODIFY — client)

**Analog:** self. `GetCLIPaths` at L97-104 is the direct template.

**Read-only-fetch client method** (client.go L97-104):
```go
// GetCLIPaths returns the current CLI path override map.
func (c *DaemonClient) GetCLIPaths() (map[string]string, error) {
	var paths map[string]string
	if err := c.doJSON(http.MethodGet, "/settings/cli-paths", nil, &paths); err != nil {
		return nil, err
	}
	return paths, nil
}
```

Apply as:
```go
// ListShells returns the shells discovered on the daemon's PATH.
func (c *DaemonClient) ListShells() ([]DetectedShell, error) {
	var resp ShellsResponse
	if err := c.doJSON(http.MethodGet, "/shells", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Shells, nil
}
```
Note `doJSON` is already the project's standard JSON-over-Unix-socket helper. No new transport code needed.

---

### `internal/daemon/path_windows.go` (POSSIBLY MODIFY — config)

**Analog:** self. Single function `platformExtraBins` (23-line file).

**Build-tag + extension pattern** (path_windows.go full file):
```go
//go:build windows

package daemon

import (
	"os"
	"path/filepath"
)

func platformExtraBins() []string {
	var paths []string
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		paths = append(paths, filepath.Join(appdata, "npm"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		paths = append(paths, filepath.Join(local, "pnpm"))
		paths = append(paths, filepath.Join(local, "Programs", "nodejs"))
	}
	paths = append(paths, `C:\Program Files\Tailscale`)
	return paths
}
```

Per Research Pitfall 1: extend the function with two entries:
```go
paths = append(paths, `C:\Program Files\PowerShell\7`) // pwsh 7.x install
if local := os.Getenv("LOCALAPPDATA"); local != "" {
	paths = append(paths, filepath.Join(local, "Microsoft", "WindowsApps")) // Microsoft Store install
}
```
The existing env-var pattern (`os.Getenv("APPDATA")` / `os.Getenv("LOCALAPPDATA")` with non-empty guard) is the canonical shape — mirror it.

**Verify before editing:** Research lists this as A5 ("Windows-mode daemon already has PATH augmentation for PowerShell, OR can be extended trivially") with the assumption flag. The file above CONFIRMS the assumption — no PowerShell paths are currently included, so the planner MUST add them in Phase 100 to satisfy SHELL-04 on Windows.

---

## Shared Patterns

### Concurrency idiom: lock + check + release
**Source:** `engine.go:388-394` (`ResolveCLI`)
**Apply to:** All new engine accessor methods (`resolveShellSpawn`, any future `DiscoverShells` engine wrapper).
```go
e.mu.RLock()
defer e.mu.RUnlock()
if path, ok := e.cliPaths[name]; ok {
	return path
}
return name
```
For helpers that DO NOT touch engine state (`isShellSession`, `resolveShellSpawn` if it only reads `pty.DiscoverShells()` and the cli-paths map), keep the lock window minimal — acquire RLock only around the `e.cliPaths[name]` lookup, drop it before calling `pty.DiscoverShells()` (which does file-system I/O via `exec.LookPath` and must not run under a mutex).

### JSON wire response helper
**Source:** `api.go:483` (and ~30 other call sites)
**Apply to:** `handleListShells` and any new daemon handlers.
```go
writeJSON(w, http.StatusOK, paths)
```
`writeJSON` is the project's standard response writer — never marshal + write directly.

### Error wrapping with %w
**Source:** `engine.go:229`
**Apply to:** Any new error paths in `engine.go` shell-resolution branch.
```go
return "", fmt.Errorf("create session: %w", err)
```
Use `%w` (not `%v`) so callers can `errors.Is` / `errors.As`.

### Settings file isolation in tests
**Source:** `engine_test.go:219-221` (`TestEngineResolveCLI`) and `api_test.go:32-35` (`testDaemon`)
**Apply to:** All new tests that instantiate `NewSessionEngine()`.
```go
e := NewSessionEngine()
e.configDir = t.TempDir()
e.cliPaths = make(map[string]string)
```
Without this, `NewSessionEngine` reads the developer's actual `~/.config/agenthub/settings.json` and pollutes test results. **Mandatory** for all new engine tests.

### exec.LookPath honors PATHEXT on Windows
**Source:** `detect.go:37` (used implicitly across the codebase)
**Apply to:** `shells.go` discovery — DO NOT add `.exe` suffix manually; `exec.LookPath("pwsh")` resolves to `pwsh.exe` automatically on Windows.

### GOOS branching at runtime, not build tag
**Source:** `native.go:62` (PTY resize Windows fallback)
**Apply to:** `shells.go` — use `if runtime.GOOS == "windows"` inline branches rather than splitting into `shells_unix.go` + `shells_windows.go`. The discovery surface is small enough that a single file with `runtime.GOOS` checks reads cleaner. **Exception:** path_windows.go already uses build tags — for Windows-specific PATH augmentation extensions, keep that pattern.

---

## No Analog Found

All files in this phase have direct in-repo analogs. **Zero files require RESEARCH.md fallback patterns.**

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| — | — | — | None. |

---

## Metadata

**Analog search scope:**
- `/Users/ken/dev/agenthub/internal/pty/` (full directory listing — 18 files)
- `/Users/ken/dev/agenthub/internal/daemon/` (full directory listing — 35 files)
- Files read in full: `internal/pty/detect.go`, `internal/pty/detect_test.go`, `internal/pty/native.go`, `internal/daemon/engine.go`, `internal/daemon/types.go`, `internal/daemon/path.go`, `internal/daemon/path_windows.go`, `internal/daemon/path_other.go`, `internal/daemon/engine_test.go` (full — 546 lines)
- Files read in part: `internal/daemon/api.go` (L1-150 routes + L460-540 handlers), `internal/daemon/api_test.go` (L1-160 fixtures + L269-300 CLI-paths test), `internal/daemon/client.go` (L96-110 GetCLIPaths)
- Grep scans: `handleGetCLIPaths`, `mux.HandleFunc`, `func Test`, `func.*Client`, status.Watch signature

**Files scanned (read or grep-touched):** 14
**Pattern extraction date:** 2026-05-12

---

## PATTERN MAPPING COMPLETE
