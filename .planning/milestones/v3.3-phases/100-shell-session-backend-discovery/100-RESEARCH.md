# Phase 100: Shell Session Backend & Discovery — Research

**Researched:** 2026-05-12
**Domain:** Cross-platform PTY shell spawn + session-type dispatch + status-heuristic exclusion (Go backend, no UI work)
**Confidence:** HIGH

## Summary

Phase 100 is a tightly-scoped backend phase that adds a new "shell" agent type to the existing AgentHub daemon. Three changes are required, all in Go: (1) a cross-platform shell-discovery enumerator alongside the existing `internal/pty/detect.go` AI-CLI discovery; (2) a per-CLI-name resolution branch in `engine.CreateSession` that maps the abstract names `bash`, `zsh`, `pwsh`, and `shell` (system default) to concrete absolute paths plus the correct interactive-non-login argv; and (3) a single-line guard in `engine.CreateSession` that skips `go status.Watch(...)` for shell-type sessions so the AI-CLI heuristic detector never runs against shell output. The `SessionInfo.Status` field already defaults to `"running"` when no detector entry exists [VERIFIED: internal/daemon/engine.go:308], so SHELL-09 falls out for free once `status.Watch` is bypassed — no changes to `cmd_cli.go`, `internal/tui/view.go`, or `app.go` status consumers are required for this phase.

Working-directory plumbing is already in place end-to-end: `CreateRequest.WorkDir` flows from HTTP body → `engine.CreateSession` → `pty.CreateRequest.WorkDir` → `cmd.Dir` on the `gopty.Cmd` [VERIFIED: internal/pty/native.go:42]. SHELL-05 needs only to ensure the new shell-resolution branch does not strip or override it. The PTY backend already injects `TERM=xterm-256color` and `COLORTERM=truecolor` [VERIFIED: native.go:46], which is exactly what an interactive shell expects — no env tweaks needed beyond passing the shell argv flags.

**Primary recommendation:** Land four small files: `internal/pty/shells.go` (cross-platform shell enumeration + argv table), `internal/pty/shells_test.go` (table-driven enumeration tests with PATH/`/etc/shells` mocking), an edit to `internal/daemon/engine.go` adding `resolveShellSpawn(cli)` + a `cli=="shell" || isShell(cli)` guard around the `go status.Watch(...)` call, and a daemon HTTP endpoint `GET /shells` returning the discovered list. No new types in `daemon/types.go` beyond a `ShellsResponse` and an optional `SessionInfo.Type` field (deferred to Phase 101 if SHELL-09 is enforced at the dispatch layer instead).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cross-platform shell discovery (SHELL-04) | Daemon / Go backend (`internal/pty`) | — | All existing CLI discovery lives in `internal/pty/detect.go`; shell discovery is the same shape and same caller (daemon). No frontend responsibility. |
| Interactive non-login PTY spawn (SHELL-05) | Daemon / Go backend (`internal/daemon/engine.go` + `internal/pty/native.go`) | — | Spawn argv shaping must live in the session-type dispatch (engine), not in the generic PTY backend (which is and should stay shell-agnostic). |
| Working-directory honor (SHELL-05) | Daemon / Go backend (`pty.CreateRequest.WorkDir` → `cmd.Dir`) | API layer (already passes `req.WorkDir` through unchanged) | Plumbing is end-to-end; this phase only verifies it isn't accidentally dropped by the shell-resolution branch. |
| Status-heuristic exclusion (SHELL-09) | Daemon / Go backend (`engine.CreateSession`) | — | Cleanest hook is at the spawn site where `go status.Watch(...)` is invoked. Skipping it means `sessionStatuses[id]` stays empty and `ListSessions` falls through to its conservative `"running"` default [VERIFIED: engine.go:308]. |
| Discovery API exposure | Daemon HTTP API (`internal/daemon/api.go`) | — | One new route `GET /shells` mirrors `GET /settings/cli-paths` shape. Frontend/CLI/TUI consumers ship in Phase 101. |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/aymanbagabas/go-pty` | v0.2.2 | Cross-platform PTY (`creack/pty` on Unix, ConPTY on Windows) | Already in use across the entire project [VERIFIED: go.mod:10]. Latest tag is v0.2.2 published 2024-01-05 [CITED: pkg.go.dev/github.com/aymanbagabas/go-pty]. No upgrade needed. |
| Go stdlib `os/exec` (`exec.LookPath`) | go1.22+ | PATH-based binary discovery | Already used by `internal/pty/detect.go:37`. Honors `PATHEXT` on Windows so `pwsh` resolves to `pwsh.exe` automatically. |
| Go stdlib `runtime.GOOS` | go1.22+ | Platform-conditional discovery branching | Standard idiom; already used at `internal/pty/native.go:62`. |
| Go stdlib `bufio` + `os.Open` | go1.22+ | `/etc/shells` parsing on Linux | Trivial line scan with comment stripping. No third-party parser warranted. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `internal/status` package | (existing) | Heuristic detector — bypassed for shell sessions | Read only; this phase wires the bypass, not the detector. |
| `internal/pty` package | (existing) | PTY backend + session registry | Extended with `shells.go` (new file). |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom `/etc/shells` parser | `github.com/twpayne/go-shell` | go-shell is a tiny abstraction over `$SHELL`/`$ComSpec` and lacks `/etc/shells` enumeration [VERIFIED: github.com/twpayne/go-shell/blob/master/shell_windows.go — only inspects env vars]. Not worth a dep for ~20 LOC of file reading. |
| New session-type field at API layer | Continue using the existing `CLI string` field as session-type discriminator | A `Type` field is cleaner long-term but expands the wire protocol surface. Phase 100 acceptance only requires that `cli=="shell"` (or shell basenames `bash`/`zsh`/`pwsh`) trigger the new branch — wire-format change can be deferred. **Recommend: keep `CLI string`; treat the value `shell` or any registered shell basename as a session-type discriminator.** |
| Use `tcsh`/`fish`/`ksh` etc. | Just `bash`/`zsh`/`pwsh` + system default | Out-of-scope per REQUIREMENTS.md line 87 — exotic shells deferred. |

**Installation:** Nothing new — `go-pty v0.2.2` already in `go.mod`.

**Version verification:** `go list -m -versions github.com/aymanbagabas/go-pty` returned `v0.1.0 v0.1.1 v0.2.0 v0.2.1 v0.2.2` on 2026-05-12 [VERIFIED: command output]. Current pin `v0.2.2` is the latest.

## Architecture Patterns

### System Architecture Diagram

```
                ┌─────────────────────────────────────────────┐
                │              Phase 100 scope                │
                └─────────────────────────────────────────────┘

  HTTP / Wails caller                      Daemon (internal/daemon)
  ───────────────────                      ─────────────────────────
  POST /sessions                           api.handleCreateSession
  { cli: "shell" | "bash"                       │
    | "zsh" | "pwsh",                           ▼
    workDir: "/path",                      engine.CreateSession
    ... }                                       │
                                                ├──► (NEW) resolveShellSpawn(cli)
                                                │       │
                                                │       └──► returns {Path, Args}
                                                │            using pty.DiscoverShells()
                                                │            (NEW internal/pty/shells.go)
                                                │
                                                ▼
                                          backend.Create(pty.CreateRequest{
                                            CLI:     "/bin/zsh",    ← absolute path
                                            Args:    ["-i"],         ← interactive, non-login
                                            WorkDir: req.WorkDir,    ← passed through
                                            Env:     nil,            ← no opencode tweak
                                            ...
                                          })
                                                │
                                                ▼
                                          internal/pty/native.go
                                          gopty.New() → cmd.Dir = req.WorkDir
                                                │
                                                ▼
                                          ┌─ POSIX: bash -i / zsh -i
                                          │   (interactive, NOT -l/--login)
                                          └─ Windows: pwsh.exe -NoLogo
                                              (ConPTY via go-pty)

  Status heuristic path                    SHELL-09 GUARD
  ─────────────────────                    ──────────────
  engine.CreateSession line 245:           if !isShellSession(cli) {
    go status.Watch(hub, id, cli, ...)         go status.Watch(hub, id, cli, ...)
                                             }
                                          ▼
                                          sessionStatuses[id] never populated
                                          for shell sessions →
                                          ListSessions falls through to
                                          conservative "running" default
                                          (engine.go:308)
                                          → SessionInfo.Status = "running"
                                          → switches to "stopped" when
                                            backend transitions State
                                            (engine.go:295)

  Discovery surface (consumed by Phase 101)
  ────────────────────────────────────────
  GET /shells       (NEW)
    → daemon.ShellsResponse {
        shells: [
          { name: "bash",   path: "/bin/bash",   displayName: "bash" },
          { name: "zsh",    path: "/bin/zsh",    displayName: "zsh" },
          { name: "shell",  path: "/bin/zsh",    displayName: "system default" }
        ]
      }
```

### Component Responsibilities

| File | Responsibility | New / Modified |
|------|----------------|----------------|
| `internal/pty/shells.go` | `DiscoverShells() []DetectedShell` — platform-conditional enumeration | NEW |
| `internal/pty/shells_test.go` | Table-driven tests with PATH + `/etc/shells` fixtures | NEW |
| `internal/daemon/engine.go` | `resolveShellSpawn(cli) (path, args)` + `isShellSession(cli) bool` helper + guard around `go status.Watch` | MODIFIED |
| `internal/daemon/engine_test.go` | Add `TestCreateSession_ShellArgv_Interactive`, `TestCreateSession_ShellWorkDirHonored`, `TestCreateSession_ShellSkipsStatusWatch` using existing `spyBackend` | MODIFIED |
| `internal/daemon/api.go` | Register `GET /shells` route → `a.handleListShells` | MODIFIED |
| `internal/daemon/types.go` | Add `ShellsResponse` + `DetectedShell` JSON types | MODIFIED |
| `internal/daemon/client.go` | Add `ListShells()` client method | MODIFIED |
| (Phase 101) `frontend/`, `cmd_cli.go`, `internal/tui/` | Surface work — NOT this phase | — |

### Recommended Project Structure

No new packages or directories. Two new files in `internal/pty/`:

```
internal/pty/
├── shells.go          # NEW — DiscoverShells, ShellSpec table, argv shaping
├── shells_test.go     # NEW — fixture-based discovery tests
├── detect.go          # existing — AI-CLI discovery (knownCLIs)
├── detect_test.go     # existing — pattern to mirror
└── ...
```

### Pattern 1: Mirror `detect.go`'s known-list pattern

The existing AI-CLI discovery uses a package-level `knownCLIs` slice + `DetectCLIs() / DetectCLI(name)` API [VERIFIED: internal/pty/detect.go:25-68]. Shell discovery should mirror this shape exactly for consistency.

```go
// internal/pty/shells.go
package pty

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ShellSpec is a known shell that AgentHub can spawn.
type ShellSpec struct {
	Name        string   // canonical name used by the API ("bash", "zsh", "pwsh")
	DisplayName string   // human-readable label
	Argv        []string // additional args for interactive-non-login (e.g. ["-i"])
}

// DetectedShell is a ShellSpec whose binary is present on this system.
type DetectedShell struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Path        string   `json:"path"`
	Argv        []string `json:"argv"`
}

// knownShellSpecs is the authoritative shell list.
// Cross-platform shape: bash/zsh on POSIX, pwsh + (Windows-only) powershell.exe.
var knownShellSpecs = []ShellSpec{
	{Name: "bash", DisplayName: "bash", Argv: []string{"-i"}},
	{Name: "zsh",  DisplayName: "zsh",  Argv: []string{"-i"}},
	{Name: "pwsh", DisplayName: "PowerShell", Argv: []string{"-NoLogo"}},
}

// DiscoverShells returns the subset of knownShellSpecs whose binary is found
// on the system, in priority order. On POSIX it also reads /etc/shells and
// adds a "system default" entry derived from $SHELL.
func DiscoverShells() []DetectedShell {
	result := []DetectedShell{}
	// Pass 1: known specs via PATH
	for _, spec := range knownShellSpecs {
		path, err := exec.LookPath(spec.Name)
		if err != nil {
			continue
		}
		result = append(result, DetectedShell{
			Name:        spec.Name,
			DisplayName: spec.DisplayName,
			Path:        path,
			Argv:        spec.Argv,
		})
	}
	// Pass 2: Windows-only fallback — powershell.exe (Windows PowerShell 5.x)
	// only when pwsh.exe was not found.
	if runtime.GOOS == "windows" {
		havePwsh := false
		for _, s := range result {
			if s.Name == "pwsh" {
				havePwsh = true
				break
			}
		}
		if !havePwsh {
			if p, err := exec.LookPath("powershell.exe"); err == nil {
				result = append(result, DetectedShell{
					Name:        "powershell",
					DisplayName: "Windows PowerShell",
					Path:        p,
					Argv:        []string{"-NoLogo"},
				})
			}
		}
	}
	// Pass 3: POSIX — /etc/shells supplements + $SHELL system default
	if runtime.GOOS != "windows" {
		if sysShell := systemDefaultShell(); sysShell != nil {
			result = append(result, *sysShell)
		}
	}
	return result
}
```
**Source:** Mirrors the verified pattern in `internal/pty/detect.go:25-68`.

### Pattern 2: Resolve shell argv inside `engine.CreateSession`

```go
// internal/daemon/engine.go (additions)

// resolveShellSpawn maps an abstract shell name to (concrete path, argv).
// Returns ("", nil, false) if cli is not a shell name.
func (e *SessionEngine) resolveShellSpawn(cli string) (string, []string, bool) {
	// Caller's custom override (Settings → Paths) wins.
	if path := e.ResolveCLI(cli); path != cli {
		// Has a custom override — figure argv from the basename.
		for _, spec := range pty.KnownShellSpecs() { // exported helper
			if spec.Name == cli {
				return path, spec.Argv, true
			}
		}
	}
	// Otherwise, look it up in the live discovery list.
	for _, sh := range pty.DiscoverShells() {
		if sh.Name == cli || (cli == "shell" && sh.Name == systemDefaultName()) {
			return sh.Path, sh.Argv, true
		}
	}
	return "", nil, false
}

// isShellSession returns true if cli refers to a shell agent type.
// Used to bypass status.Watch for SHELL-09.
func isShellSession(cli string) bool {
	switch cli {
	case "shell", "bash", "zsh", "pwsh", "powershell":
		return true
	}
	return false
}
```

### Pattern 3: Status-heuristic guard

```go
// internal/daemon/engine.go, inside CreateSession, replacing line 245:
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
// For shell sessions, sessionStatuses[id] is never written.
// ListSessions (engine.go:308) falls through to its "running" default,
// and transitions to "stopped" when the PTY state changes (engine.go:295).
```

### Anti-Patterns to Avoid

- **Adding a new `SessionType` field to `pty.CreateRequest`:** The PTY backend should stay agent-agnostic; session-type semantics belong in the engine. Resolve to `{CLI, Args}` at the engine level and pass through the existing fields.
- **Adding "shell" to `internal/status/detector.go`'s `PatternsForCLI`:** Even with an empty PatternSet, `Watch` still subscribes, holds a Hub channel, and runs a goroutine. The guard at the call site avoids those resources entirely.
- **Modifying `cmd_cli.go` `cmdList` to special-case shell sessions:** Unnecessary — `cmdList` already prints `s.State` (running/stopped), not `s.Status` [VERIFIED: cmd_cli.go:140]. SHELL-09 is already satisfied for the CLI surface once the engine guard lands.
- **Spawning `bash -l` or `zsh -l`:** Login shells re-source `/etc/profile`, `~/.profile`, `~/.bash_profile`, `~/.zprofile` and produce slow startup + login banner spam. Out-of-scope per REQUIREMENTS.md line 86.
- **Forwarding `req.Args` to shells:** The HTTP-level `CreateRequest.Args` is meant for CLI passthrough (`agenthub new claude -- --model X`). If a shell session sets `Args`, prepend (not replace) the shell's own argv: `append(shellArgs, req.Args...)`. Safer: ignore `req.Args` for shells in Phase 100 and revisit if a use case appears.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Find executable on PATH | Manual `os.Stat` over `os.Getenv("PATH")` split | `exec.LookPath` | Stdlib already handles `PATHEXT` on Windows, OS-specific separators, and respects `os.PathListSeparator`. |
| Cross-platform PTY | Native `unix.OpenPty()` / Windows `CreatePseudoConsole` | `go-pty` (already in use) | Already vendored; ConPTY requires special process attributes that `os/exec` cannot set [CITED: Go issue #62708 surfaced in WebSearch]. |
| Detect "system default" shell | Inspect `/etc/passwd`, `getent`, etc. | `os.Getenv("SHELL")` then fall back to `/bin/sh` | What every standard tool does (login(1), su(1), tmux). `/etc/passwd` parsing is brittle and requires running-user lookup. |
| Argv quoting on Windows | Manual quote escaping | `gopty.Cmd.Args` (go-pty handles it) | go-pty wraps Windows process-creation flags so the standard `cmd.Args` slice works. Don't pre-quote. |
| Status detection bypass | Adding empty PatternSet | Skip `go status.Watch(...)` entirely with `isShellSession` guard | Saves goroutine + channel buffer; cleaner semantics ("we never tried to classify"). |

**Key insight:** Almost every part of this phase already has a stdlib or in-repo idiom. The phase's risk is structural (where to put the new code), not algorithmic.

## Common Pitfalls

### Pitfall 1: `exec.LookPath("pwsh")` on Windows misses Microsoft Store + Program Files installs
**What goes wrong:** PowerShell 7 (`pwsh.exe`) installed via Microsoft Store lives under `%LOCALAPPDATA%\Microsoft\WindowsApps\pwsh.exe` and may not be on the default `%PATH%` for service-mode daemons. Program Files install path is `C:\Program Files\PowerShell\7\pwsh.exe`. The existing daemon already augments PATH at startup for service-mode AI CLIs [VERIFIED: PROJECT.md key decision "Runtime PATH augmentation at daemon startup"].
**Why it happens:** Service-mode processes can't source the user's shell init files; the existing `path_windows.go` build-tagged file prepends known install dirs.
**How to avoid:** Extend `internal/daemon/path_windows.go` (or wherever PATH augmentation lives — see `path.go:15`) to include `C:\Program Files\PowerShell\7` and `%LOCALAPPDATA%\Microsoft\WindowsApps`. Verify the existing file's structure first; the project pattern is "comprehensive agent CLI discovery" [VERIFIED: PROJECT.md v1.13 Phase 68].
**Warning signs:** SHELL-04 returns empty `pwsh` entry on Windows when run as a service. UAT: install PowerShell 7 via Microsoft Store, run daemon as a service, verify `GET /shells` lists `pwsh`.

### Pitfall 2: `/etc/shells` missing or empty on minimal Linux containers
**What goes wrong:** Alpine-based or distroless containers may ship without `/etc/shells`. A naive parser that errors on missing file would return zero shells even when `/bin/bash` and `/bin/zsh` are on PATH.
**Why it happens:** `/etc/shells` is part of `shadow` or `etc-files` package; containers strip it.
**How to avoid:** Treat `/etc/shells` as supplementary, not authoritative. Pass 1 (`exec.LookPath` over `knownShellSpecs`) is the primary source. Pass 3 (`/etc/shells`) is a no-op if the file is missing — silently skip. Acceptance per Blockers/Concerns in STATE.md: "Phase 100 acceptance limits scope to bash/zsh/pwsh/system-default; exotic shells deferred."
**Warning signs:** Discovery returns zero shells in CI when running inside a slim container. Test fixture: `t.TempDir()` + `os.Setenv("PATH", ...)` without writing `/etc/shells` — must still return bash/zsh.

### Pitfall 3: Setpgid+Setsid EPERM on macOS regression
**What goes wrong:** Modifying the PTY backend to add new sysproc attributes can re-introduce the `EPERM` failure noted in `native.go:48-50`.
**Why it happens:** `go-pty` already sets `Setsid: true`; combining with `Setpgid: true` causes EPERM on macOS.
**How to avoid:** Phase 100 must NOT touch `internal/pty/native.go`'s sysproc attribute block. All shell-specific logic lives in `engine.go` (argv shaping) and a new `shells.go` (discovery). The backend stays generic.
**Warning signs:** macOS CI fails with "operation not permitted" when running shell-spawn integration tests.

### Pitfall 4: `req.WorkDir = ""` falls through to current working directory of the daemon
**What goes wrong:** Caller passes empty `workDir`. `go-pty`'s `cmd.Dir = ""` means "use parent's CWD" — i.e., wherever the daemon was started. For service-mode daemons this is typically `/` or `C:\Windows\System32`.
**Why it happens:** Existing behavior — `engine.go:226` and `native.go:42` pass `req.WorkDir` straight through.
**How to avoid:** Document in the API that empty `workDir` means "use `$HOME` for shells, daemon CWD for AI CLIs" — OR — require frontend/CLI surface (Phase 101) to always pass a concrete path. **Recommendation:** in Phase 100, add a default in `resolveShellSpawn`: if `req.WorkDir == ""` for shells, substitute `os.UserHomeDir()`. AI CLIs preserve existing behavior.
**Warning signs:** User opens a shell session with no folder picked and lands in `/` — surprising and breaks shell history files.

### Pitfall 5: ConPTY does not parse argv shell-style
**What goes wrong:** Treating a shell's argv as a quoted shell command (e.g., passing `"-i -l"` as a single string) breaks. Each token must be a separate element of `Args`.
**Why it happens:** `go-pty` on Windows uses ConPTY which receives the joined command line and Microsoft's parser; on POSIX it uses `execve` (no shell). In both cases, individual argv slots are sacred.
**How to avoid:** Always use `Argv []string{"-i"}` not `Argv []string{"-i -l"}`. The `ShellSpec.Argv` field in this research is correctly typed as `[]string`.
**Warning signs:** `pwsh.exe` opens with error "Unknown parameter: -i -l".

### Pitfall 6: Stale CLI path override sneaks shells into the AI-CLI custom-path map
**What goes wrong:** A user-configured custom path like `cliPaths["claude"] = "/bin/zsh"` would, after this phase, route both AI-CLI spawn AND status-heuristic decisions through the wrong code path.
**Why it happens:** The daemon already has the defensive `knownShells` filter [VERIFIED: engine.go:97-100] specifically to drop these. **This pitfall is already mitigated.**
**How to avoid:** Verify the `knownShells` map in `engine.go:97` covers `pwsh`/`powershell` (it currently has `sh`/`bash`/`zsh`/`fish`/`csh`/`tcsh`/`dash`/`ksh`). Decide whether to add `pwsh` — leaning yes — but note that on Windows the basename includes `.exe` so `filepath.Base("/path/to/pwsh.exe")` yields `pwsh.exe`, which won't match a bare `"pwsh"` key. Either normalize (strip `.exe`) or extend the map with both forms.
**Warning signs:** A user with a stale `claude → /bin/zsh` override in settings.json gets shell-argv treatment for their Claude session after upgrade.

## Code Examples

Verified patterns from the existing codebase that Phase 100 should mirror.

### Pattern: Test argv injection without launching a real PTY

```go
// Source: internal/daemon/engine_test.go:241-257 (spyBackend)
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

Apply this pattern for new tests:

```go
// New: TestCreateSession_ShellArgv_Interactive
func TestCreateSession_ShellArgv_Interactive(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	_, err := e.CreateSession(context.Background(), "bash", "tab", "/home/user", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession(bash): %v", err)
	}

	// Assert CLI resolved to absolute path
	if !strings.HasSuffix(spy.lastReq.CLI, "/bash") {
		t.Errorf("CLI = %q, want absolute path ending in /bash", spy.lastReq.CLI)
	}
	// Assert -i in args, no -l
	hasI, hasL := false, false
	for _, a := range spy.lastReq.Args {
		if a == "-i" { hasI = true }
		if a == "-l" || a == "--login" { hasL = true }
	}
	if !hasI { t.Errorf("Args missing -i: %v", spy.lastReq.Args) }
	if hasL  { t.Errorf("Args has login flag (must be non-login): %v", spy.lastReq.Args) }
	// Assert workDir honored
	if spy.lastReq.WorkDir != "/home/user" {
		t.Errorf("WorkDir = %q, want %q", spy.lastReq.WorkDir, "/home/user")
	}
}
```

### Pattern: PATH-mocked discovery test

```go
// Source: internal/pty/detect_test.go:12-40 (mirror for shells)
func TestDiscoverShells_FindsInstalledShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses shell script stubs not executable on Windows")
	}
	dir := t.TempDir()
	for _, name := range []string{"bash", "zsh"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
			t.Fatalf("writing stub %s: %v", name, err)
		}
	}
	t.Setenv("PATH", dir)

	result := DiscoverShells()

	names := map[string]bool{}
	for _, sh := range result {
		names[sh.Name] = true
	}
	if !names["bash"] { t.Error("bash not discovered") }
	if !names["zsh"]  { t.Error("zsh not discovered") }
}
```

### Pattern: Status-watch bypass assertion

```go
// New: TestCreateSession_ShellSkipsStatusWatch
func TestCreateSession_ShellSkipsStatusWatch(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	id, _ := e.CreateSession(context.Background(), "bash", "sh-tab", "", nil, 80, 24, nil, nil)

	// Drive a Feed that would normally produce an "errored" classification...
	// Then verify GetSessionStatus returns "running" (the conservative default).
	got := e.GetSessionStatus(id)
	if got != "running" {
		t.Errorf("shell session status = %q, want %q", got, "running")
	}

	// Also verify ListSessions reports Status="running" and never "waiting"/"errored"
	sessions := e.ListSessions()
	for _, s := range sessions {
		if s.ID == id && s.Status != "running" {
			t.Errorf("ListSessions Status = %q, want %q", s.Status, "running")
		}
	}
}
```

## Runtime State Inventory

Phase 100 is **not** a rename/refactor/migration phase — it is purely additive. No runtime state needs auditing.

Explicit confirmation of each category:

| Category | Status |
|----------|--------|
| Stored data | None — no databases, ChromaDB, or persistent stores impacted. SessionRegistry is in-memory only. |
| Live service config | None — settings.json gains no new keys in Phase 100 (custom shell-path override is out-of-scope per REQUIREMENTS.md line 85). |
| OS-registered state | None — daemon service registration unchanged. |
| Secrets/env vars | None — no new secrets. `TERM` / `COLORTERM` env injection already in place for all sessions. |
| Build artifacts | None — no module renames, no `*.egg-info`-style artifacts in Go projects. |

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All build/test | ✓ | 1.22+ (existing) | — |
| go-pty | PTY spawn | ✓ | v0.2.2 (in go.mod) | — |
| `/bin/bash` | Test fixture + macOS shell discovery | ✓ on dev machine (macOS Darwin 25.4) | — | Test uses `t.TempDir()` + stub; production discovery uses `exec.LookPath` |
| `/bin/zsh` | Test fixture + macOS shell discovery | ✓ on dev machine | — | Same as above |
| `pwsh.exe` | Windows discovery UAT | Unverified on macOS dev box (Windows CI/UAT only) | — | Tests must be GOOS-conditional; CI matrix already includes Windows leg [VERIFIED: PROJECT.md "race detector on all 4 platform legs"] |
| `/etc/shells` | Linux discovery supplementation | ✓ on macOS dev (Darwin has /etc/shells); Linux varies | — | Treated as optional; missing file is silent skip |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None blocking Phase 100. Windows-specific verification requires running on the Windows CI leg (already part of every PR per project conventions).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (verified by `go.mod` and existing 300+ tests) |
| Config file | none — Go convention (no pytest.ini analog) |
| Quick run command | `go test ./internal/pty/... ./internal/daemon/... -run Shell -race -count=1` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| SHELL-04 | Discover bash + zsh via PATH | unit | `go test ./internal/pty -run TestDiscoverShells_FindsInstalledShells -race` | ❌ Wave 0 — `internal/pty/shells_test.go` |
| SHELL-04 | Skip missing shells | unit | `go test ./internal/pty -run TestDiscoverShells_SkipsMissing -race` | ❌ Wave 0 |
| SHELL-04 | `/etc/shells` supplements POSIX discovery (with fixture file) | unit | `go test ./internal/pty -run TestDiscoverShells_EtcShellsFixture -race` | ❌ Wave 0 |
| SHELL-04 | Empty / missing `/etc/shells` is silent skip | unit | `go test ./internal/pty -run TestDiscoverShells_NoEtcShells -race` | ❌ Wave 0 |
| SHELL-04 | Windows pwsh.exe discovery via PATHEXT | unit (Windows-tagged) | `go test ./internal/pty -run TestDiscoverShells_Windows -race` (GOOS=windows only) | ❌ Wave 0 |
| SHELL-04 | `GET /shells` returns discovered list | integration | `go test ./internal/daemon -run TestHandleListShells -race` | ❌ Wave 0 |
| SHELL-05 | bash argv is `-i` (interactive, non-login) | unit (`spyBackend`) | `go test ./internal/daemon -run TestCreateSession_ShellArgv_Interactive -race` | ❌ Wave 0 |
| SHELL-05 | zsh argv is `-i` | unit (`spyBackend`) | `go test ./internal/daemon -run TestCreateSession_ZshArgv_Interactive -race` | ❌ Wave 0 |
| SHELL-05 | pwsh argv is `-NoLogo` (no `-l`/`--login`) | unit (`spyBackend`) | `go test ./internal/daemon -run TestCreateSession_PwshArgv -race` | ❌ Wave 0 |
| SHELL-05 | WorkDir passed through unchanged to backend | unit (`spyBackend`) | `go test ./internal/daemon -run TestCreateSession_ShellWorkDirHonored -race` | ❌ Wave 0 |
| SHELL-05 | Empty WorkDir defaults to `$HOME` for shells | unit | `go test ./internal/daemon -run TestCreateSession_ShellEmptyWorkDirHome -race` | ❌ Wave 0 |
| SHELL-05 | Real PTY smoke test: spawn shell, read prompt, kill | integration | `go test ./internal/pty -run TestShellSpawn_RealPTY -race` (POSIX only) | ❌ Wave 0 |
| SHELL-09 | `go status.Watch` not invoked for shell sessions | unit (`spyBackend` + status assertion) | `go test ./internal/daemon -run TestCreateSession_ShellSkipsStatusWatch -race` | ❌ Wave 0 |
| SHELL-09 | `ListSessions` returns Status="running" for shell session | unit | `go test ./internal/daemon -run TestListSessions_ShellStatusRunning -race` | ❌ Wave 0 |
| SHELL-09 | `ListSessions` returns Status="stopped" after shell exit | integration | `go test ./internal/daemon -run TestListSessions_ShellStatusStopped -race` | ❌ Wave 0 |
| SHELL-09 | `sessionStatuses[id]` map is never written for shell session | unit (reach into engine state) | `go test ./internal/daemon -run TestShell_NoStatusMapEntry -race` | ❌ Wave 0 |
| SHELL-09 | Windows / Linux parity smoke (no `waiting`/`errored` emitted) | manual UAT | n/a — spawn shell on each OS, run `agenthub list` for 30s, confirm Status never flips | Manual |

### Sampling Rate
- **Per task commit:** `go test ./internal/pty/... ./internal/daemon/... -run Shell -race -count=1`
- **Per wave merge:** `go test ./internal/pty/... ./internal/daemon/... -race -count=1`
- **Phase gate:** `go test ./... -race -count=1` green on macOS dev box + GitHub Actions CI matrix (macOS/Linux/Windows)

### Wave 0 Gaps
- [ ] `internal/pty/shells.go` — new file (`DiscoverShells`, `ShellSpec`, `DetectedShell`)
- [ ] `internal/pty/shells_test.go` — new file (table-driven discovery tests + GOOS-conditional Windows test)
- [ ] `internal/daemon/engine_test.go` — extend with shell-specific spawn tests + status-bypass test
- [ ] `internal/daemon/api_test.go` — extend with `TestHandleListShells`
- [ ] `internal/daemon/types.go` — add `ShellsResponse` type
- [ ] No framework install needed — Go stdlib `testing` is already in use across all packages

### What's testable in CI vs needs manual UAT

| Test Category | CI Viable? | Notes |
|---------------|-----------|-------|
| Discovery (PATH + `/etc/shells` mocking) | ✓ Linux + macOS legs | `t.TempDir()` + `t.Setenv("PATH", ...)` pattern from `detect_test.go` |
| Windows pwsh.exe discovery | ✓ Windows CI leg | Build-tagged test; rely on `pwsh.exe` being installed in GitHub Actions Windows runner image (it is, default-installed) |
| Argv shape (`spyBackend`) | ✓ all platforms | No real PTY; assertions on `lastReq.Args` and `lastReq.WorkDir` |
| Real PTY shell spawn (integration) | ✓ POSIX CI (Linux + macOS) | Existing pattern `TestEngineCreateSession` uses `/bin/cat` — substitute `/bin/sh` and verify prompt byte returns |
| Real PTY shell spawn (Windows ConPTY) | ⚠ Windows CI feasible but fragile | go-pty Windows tests exist; recommend skipping initial release and gating on Windows manual UAT |
| Status-heuristic bypass | ✓ all platforms | Assert on `e.GetSessionStatus(id)` and `e.sessionStatuses[id]` map state |
| `agenthub list` does not show `waiting`/`errored` for shells | ✓ via integration test on `engine.ListSessions()` | No need to invoke CLI; the data path is what matters |
| Cross-platform real shell behavior (e.g., zsh sources `~/.zshrc`?) | ✗ manual UAT | "Did the prompt actually appear?" — eyeballing required |

## Project Constraints (from CLAUDE.md)

The project CLAUDE.md (at `/Users/ken/dev/CLAUDE.md`, user-level) imposes:

| Constraint | How Phase 100 Honors It |
|------------|-------------------------|
| Go: `go fmt` + `golangci-lint`, context-aware functions (`ctx context.Context`) | `CreateSession(ctx context.Context, ...)` already conforms; new helpers add no new external surface |
| Cross-platform: must work on macOS, Linux, Windows from day one | Discovery uses `runtime.GOOS` branches; CI matrix already includes all 3 |
| Testing: 80%+ coverage in critical components, integration tests for daemon | Wave 0 tests cover unit + integration for all three requirements |
| Make beliefs pay rent (explicit predictions before significant actions) | Plans should predict argv exactly before writing the resolve function |
| Premature abstraction: need 3 real examples | Three shells (bash, zsh, pwsh) is exactly 3 — the abstraction is warranted |
| Silent fallbacks: `or {}` converts hard failures into silent corruption | Empty argv slice for an unknown shell name should be an error, not a silent empty spawn |
| RULE 0 (catastrophic failures): STOP completely | N/A for Phase 100 — well-contained additive work |
| NEVER `kill node.exe` | N/A — Go-only changes |
| Use LSP for code navigation | Apply during plan execution |

No project-root `./CLAUDE.md` exists in `/Users/ken/dev/agenthub/`; the constraints above come from `/Users/ken/dev/CLAUDE.md` (user-level, inherited).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Windows PTY via `os/exec` + manual ConPTY | go-pty (`creack/pty` on Unix, ConPTY on Windows) | Project v1.0 decision [VERIFIED: PROJECT.md key decision row] | One API surface, Windows-ConPTY-correct |
| Generic VPN interface binding | Tailscale-only | v1.2 | Not directly relevant to Phase 100, but illustrates project's "one supported path" philosophy — apply same lens: support bash/zsh/pwsh, defer fish/tcsh |
| Login-shell startup (`-l`) | Interactive non-login (`-i`) for daemon-spawned shells | Per REQUIREMENTS.md SHELL-05 + out-of-scope row 86 | Faster startup, no banner spam, no profile re-source surprises |

**Deprecated/outdated:**
- `cmd.exe` on Windows — out of scope for this phase. PowerShell (pwsh / powershell.exe) is the supported Windows shell. Document this in the public API so users don't expect `cmd.exe`.
- Older go-pty versions (v0.1.x) — current pin v0.2.2 is the latest; no upgrade pressure.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | PowerShell 7's `pwsh.exe` install paths on Windows (`C:\Program Files\PowerShell\7\pwsh.exe` and `%LOCALAPPDATA%\Microsoft\WindowsApps`) | Pitfall 1 | If WindowsApps path requires UAC elevation to access from service-mode daemon, discovery may still fail. Confirm before locking in. **Mitigation:** rely primarily on `exec.LookPath("pwsh")` honoring `PATHEXT`; treat hardcoded paths as supplement, not primary. |
| A2 | `bash -i` and `zsh -i` are sufficient for interactive non-login semantics on all POSIX systems including macOS (Catalina+ default shell zsh) | Pattern 2 + Pitfall 1 | If macOS Catalina+'s zsh has unusual default behavior (e.g., requires `ZDOTDIR` to find user config), prompts may not render correctly. **Verification:** real-PTY smoke test in CI validates this empirically. |
| A3 | `pwsh -NoLogo` is the correct argv for interactive non-login PowerShell (no `-NoExit` since we want the shell to be the foreground process, not exit) | Pattern 2 | If `-NoExit` is actually needed to keep pwsh alive in a ConPTY context, sessions would close immediately. **Verification:** Windows CI / manual UAT. |
| A4 | The existing `engine.go:308` conservative `"running"` default is sufficient to satisfy SHELL-09 without modifying `ListSessions` further | Architecture Diagram + Pattern 3 | If any consumer treats `Status=="running"` differently from "shell" — unlikely per code search, all consumers use `s.State` or accept any non-stopped status — guard is sufficient. **Verification:** new test `TestListSessions_ShellStatusRunning`. |
| A5 | Windows-mode daemon already has PATH augmentation for PowerShell, OR can be extended trivially | Pitfall 1 | If `path_windows.go` does not include PowerShell paths, SHELL-04 may return empty on Windows service-mode. **Verification:** read `internal/daemon/path_windows.go` during Wave 0 planning. |
| A6 | `req.Args` (HTTP-level CreateRequest.Args, used by `agenthub new claude -- --model X`) is NOT used for shell sessions in Phase 100; if a future use surfaces, prepend shell argv to req.Args | Anti-Patterns | If Phase 101's CLI surface (`agenthub new shell <path> -- <extra>`) wants to forward `<extra>` as shell init commands, Phase 100 must reserve a way to do that. **Recommendation:** for now, accept `req.Args` as appended-after-shell-argv. |

## Open Questions

1. **Should `cli="shell"` (system default) be a distinct entry in `GET /shells` response, or computed client-side from `$SHELL`?**
   - What we know: the daemon already inspects env vars in similar contexts (OPENCODE_TUI_CONFIG). `$SHELL` is a per-user var that the daemon will see on POSIX.
   - What's unclear: on Windows, `$SHELL` is usually unset; "system default" doesn't have a clean answer. Could fall back to `pwsh.exe` if present, else `powershell.exe`.
   - Recommendation: include a synthetic `{name:"shell", displayName:"system default", path: <resolved>}` entry in the API response. Phase 101 surface code can then render it uniformly without OS branching.

2. **Should the `Type` field be added to `daemon.SessionInfo` now (Phase 100) or in Phase 101?**
   - What we know: `isShellSession(cli)` works as a discriminator using existing fields. SessionInfo additions are API surface changes.
   - What's unclear: Phase 101 needs to render a distinct badge color (SHELL-06) — does it need a separate Type field, or can it inspect `cli ∈ {bash,zsh,pwsh,powershell,shell}`?
   - Recommendation: defer to Phase 101 planning. Phase 100 can ship without it; if Phase 101 finds the discriminator awkward, add `Type` as an additive field with backward-compat default `Type=""` treated as AI CLI.

3. **Should `/etc/shells` parsing produce additional `DetectedShell` entries beyond `knownShellSpecs`?**
   - What we know: REQUIREMENTS.md line 19 mentions "Linux: `$SHELL`, `/etc/shells` entries" implying yes. But out-of-scope row 85 says custom paths are out.
   - What's unclear: are "/etc/shells entries" meant to surface as discovered shells (e.g., `/bin/dash`) or just inform the "system default" picker?
   - Recommendation: in Phase 100, parse `/etc/shells` ONLY to validate that the system-default `$SHELL` value is a real interactive shell (sanity check). Do NOT surface `dash`/`fish`/`tcsh` as user-selectable shells — out of scope per the requirements doc.

4. **Cleanup behavior: when `engine.KillSession` runs for a shell session, does anything else need to be skipped?**
   - What we know: `KillSession` already deletes `sessionStatuses[id]` (a no-op when empty), `tabNames[id]`, and `sessionCLIs[id]`. All safe.
   - Recommendation: no special handling needed. Verify with `TestKillSession_ShellCleanup`.

## Sources

### Primary (HIGH confidence)
- `internal/daemon/engine.go` lines 200-282 (`CreateSession`), 285-341 (`ListSessions`), 308 (Status default), 245 (`status.Watch` call site) — verified in this session
- `internal/pty/detect.go` lines 25-68 (`knownCLIs`, `DetectCLIs`, `DetectCLI`) — pattern to mirror for shell discovery
- `internal/pty/native.go` lines 34-97 (PTY backend `Create`), 42 (`cmd.Dir = req.WorkDir`), 46 (TERM env) — verified working-directory and env plumbing
- `internal/status/detector.go` full file — verified the heuristic engine bypass is structurally sound at the engine call site
- `internal/daemon/engine_test.go` lines 41-100 (engine tests), 241-257 (`spyBackend`) — test harness pattern verified
- `internal/pty/detect_test.go` full file — PATH-mocking pattern verified
- `internal/daemon/types.go` full file — JSON wire format
- `internal/daemon/api.go` lines 60-95 (routes), 376-413 (`handleCreateSession`) — verified API surface
- `cmd_cli.go` line 140 (`agenthub list` prints `s.State` not `s.Status`) — verified CLI surface unaffected
- `internal/tui/view.go` line 800-811 (`statusGlyph`) — verified TUI default branch handles shell sessions correctly
- `.planning/PROJECT.md` lines 268, 295, 322 — verified project decisions: go-pty rationale, runtime PATH augmentation, single source of truth for state
- `.planning/STATE.md` lines 50-87 — verified phase decomposition and concerns
- `.planning/REQUIREMENTS.md` lines 16-25, 85-90 — verified requirement definitions and out-of-scope
- `go.mod` line 10 — verified `go-pty v0.2.2` pin
- `go list -m -versions github.com/aymanbagabas/go-pty` — verified v0.2.2 is the latest

### Secondary (MEDIUM confidence)
- [pkg.go.dev/github.com/aymanbagabas/go-pty](https://pkg.go.dev/github.com/aymanbagabas/go-pty) — go-pty Cmd struct documentation (Dir, Args, Env support)
- [github.com/aymanbagabas/go-pty](https://github.com/aymanbagabas/go-pty) — referenced Go issues #62708 / #62710 for ConPTY rationale
- [twpayne/go-shell shell_windows.go](https://github.com/twpayne/go-shell/blob/master/shell_windows.go) — confirms minimal SHELL/ComSpec pattern (rejected as too thin a dep)
- [Microsoft Learn: about_Pwsh](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_pwsh) — `-NoLogo` flag suppresses banner
- [ss64.com/ps/pwsh.html](https://ss64.com/ps/pwsh.html) — pwsh.exe flag reference

### Tertiary (LOW confidence)
- WebSearch results on bash interactive non-login semantics — consistent with widely-known shell behavior; no contradiction found

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in use in the project, verified versions
- Architecture: HIGH — code paths read directly from current files; no inference
- Pitfalls: MEDIUM — Pitfall 1 (Windows pwsh paths) and Pitfall 4 (empty WorkDir default) assume behavior that should be reverified during Wave 0 by reading `path_windows.go` and the existing service-mode PATH augmentation code

**Research date:** 2026-05-12
**Valid until:** 2026-06-12 (30 days — Go ecosystem moves slowly, go-pty release cadence is yearly)

## RESEARCH COMPLETE
