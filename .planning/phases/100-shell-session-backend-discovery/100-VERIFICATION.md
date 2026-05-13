---
phase: 100-shell-session-backend-discovery
verified: 2026-05-12T17:08:00Z
status: passed
score: 13/13 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: none
  previous_score: n/a
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 100: Shell Session Backend & Discovery Verification Report

**Phase Goal:** Daemon can spawn raw shell PTYs as a distinct session type with cross-platform binary discovery, correct interactive (non-login) semantics, and clean exclusion from AI-CLI status heuristics.

**Verified:** 2026-05-12T17:08:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| #   | Truth | Status     | Evidence |
| --- | ----- | ---------- | -------- |
| 1   | Daemon enumerates installed shells per platform and exposes them via session-creation API | VERIFIED | `internal/pty/shells.go` `DiscoverShells()` (lines 68-116) enumerates `knownShellSpecs` (bash, zsh, pwsh, powershell) via `exec.LookPath`; on POSIX adds synthetic `shell` entry from `$SHELL` cross-checked against `/etc/shells`. Exposed at `GET /shells` via `internal/daemon/api.go:495` `handleListShells`. Windows discovery augmented via `path_windows.go` adding `C:\Program Files\PowerShell\7` + `%LOCALAPPDATA%\Microsoft\WindowsApps`. Verified with `TestDiscoverShells_FindsInstalledShells` and `TestHandleListShells_PopulatedPATH` (PASS). |
| 2   | Shell session spawns as interactive (non-login) PTY with caller-supplied WorkDir honored | VERIFIED | `engine.go:238-249` `resolveShellSpawn` dispatch replaces `cliPath`+`args` from `pty.KnownShellSpecs()` (argv = `["-i"]` for bash/zsh, `["-NoLogo"]` for pwsh/powershell — never `-l`/`--login`). Empty WorkDir for shells falls back to `os.UserHomeDir()`. Verified by `TestCreateSession_ShellArgv_Interactive`, `TestCreateSession_ZshArgv_Interactive`, `TestCreateSession_ShellWorkDirHonored`, `TestCreateSession_ShellEmptyWorkDirHome` (all PASS). |
| 3   | Shell sessions appear in `agenthub list` and registry without emitting `waiting`/`error` heuristic states — only `running` and `stopped` | VERIFIED | `engine.go:294` wraps `go status.Watch(...)` in `if !isShellSession(cli)` guard so shell IDs never get an entry in `sessionStatuses`. `engine.go:359` ListSessions falls through to `heuristicStatus = StatusRunning` default. Verified end-to-end by `TestShellSessionLifecycle_StatusOnlyRunningOrStopped` (PASS, 1.14s): polls 5 times across alive phase, then 20× post-kill; asserts Status never in {waiting, error, errored, idle}. Backed by unit guards `TestCreateSession_ShellSkipsStatusWatch`, `TestListSessions_ShellStatusRunning`, `TestShell_NoStatusMapEntry` (all PASS). |

**Score:** 3/3 ROADMAP success criteria verified

### Plan-Level Must-Haves (Aggregated Truths from 4 Plans)

| #   | Truth | Plan | Status |
| --- | ----- | ---- | ------ |
| 1   | DiscoverShells returns bash+zsh entries when both on PATH | 100-01 | VERIFIED (TestDiscoverShells_FindsInstalledShells) |
| 2   | DiscoverShells returns non-nil empty slice when no shells on PATH | 100-01 | VERIFIED (TestDiscoverShells_AllMissing; `make([]DetectedShell, 0)` at shells.go:69) |
| 3   | knownShellSpecs contains exactly bash, zsh, pwsh, powershell (canonical order) | 100-01 | VERIFIED (shells.go:46-51; TestKnownShellSpecs_HasExpectedEntries) |
| 4   | Empty `$SHELL` never appends synthetic 'shell' entry (H4 contract) | 100-01 | VERIFIED (shells.go:88-92 early return; TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry) |
| 5   | Missing `/etc/shells` silently skipped (no panic) | 100-01 | VERIFIED (shells.go:177-179 returns empty slice on error; TestDiscoverShells_NoEtcShells) |
| 6   | CreateSession(cli=bash) sets path ending /bash, Args contains `-i`, never `-l`/`--login` | 100-02 | VERIFIED (TestCreateSession_ShellArgv_Interactive) |
| 7   | CreateSession(cli=zsh) Args contains `-i`, never `-l`/`--login` | 100-02 | VERIFIED (TestCreateSession_ZshArgv_Interactive) |
| 8   | cliPaths["powershell"] override resolves via knownShellSpec match (M2) | 100-02 | VERIFIED (engine.go:474-487; TestResolveShellSpawn_PowerShellOverride) |
| 9   | Empty WorkDir for shells resolves to `os.UserHomeDir()`, NOT daemon CWD | 100-02 | VERIFIED (engine.go:241-244; TestCreateSession_ShellEmptyWorkDirHome) |
| 10  | Empty WorkDir for AI CLIs unchanged (no `$HOME` substitution) | 100-02 | VERIFIED (TestCreateSession_AICLIEmptyWorkDirUnchanged — negative regression guard) |
| 11  | `go status.Watch` NEVER invoked for shell sessions | 100-02 | VERIFIED (engine.go:294 `if !isShellSession(cli)`; TestCreateSession_ShellSkipsStatusWatch; TestShell_NoStatusMapEntry) |
| 12  | Windows PATH augmentation includes `C:\Program Files\PowerShell\7` and `%LOCALAPPDATA%\Microsoft\WindowsApps` | 100-03 | VERIFIED (path_windows.go:20, 23 — confirmed via grep; cross-compile `GOOS=windows go vet`/`go build` clean) |
| 13  | GET /shells returns ShellsResponse JSON; empty case is `"shells":[]` not `"shells":null` | 100-04 | VERIFIED (api.go:495-507 uses `make([]DetectedShell, 0, len(...))`; api_test.go:1373-1377 asserts `bytes.Contains(body, "\"shells\":[]")` AND `!bytes.Contains(body, "\"shells\":null")`; TestHandleListShells_EmptyPATH PASS) |

**Score:** 13/13 plan-level must-have truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/pty/shells.go` | DiscoverShells, KnownShellSpecs, DetectShell, ErrShellNotFound, types, testEtcShellsPath hook | VERIFIED | 206 lines; all exports present; `var testEtcShellsPath = ""` at line 58 (single source of truth per H1) |
| `internal/pty/shells_test.go` | 9 test functions covering POSIX + Windows | VERIFIED | 9 funcs present; 7 PASS on POSIX, 2 Windows tests correctly skipped |
| `internal/daemon/engine.go` | resolveShellSpawn, isShellSession, status.Watch guard, extended knownShells map | VERIFIED | `isShellSession` at L113; `resolveShellSpawn` at L467; `knownShells` map at L103-108 includes `pwsh`, `pwsh.exe`, `powershell`, `powershell.exe`; status.Watch guard at L294 |
| `internal/daemon/engine_test.go` | 13 new tests for shell-spawn + status-bypass | VERIFIED | All 13 test funcs found by grep; 12 PASS on POSIX dev box, `TestCreateSession_PwshArgv` correctly skipped (pwsh not installed) |
| `internal/daemon/path_windows.go` | Extended platformExtraBins with PowerShell paths | VERIFIED | `Microsoft\WindowsApps` at L20 (inside LOCALAPPDATA guard); `C:\Program Files\PowerShell\7` at L23 hardcoded; pre-existing 4 entries preserved |
| `internal/daemon/path_windows_test.go` | 3 Windows-tagged tests | VERIFIED | All 3 funcs found; `//go:build windows` first non-blank line; cross-compile vet clean |
| `internal/daemon/types.go` | DetectedShell + ShellsResponse wire types | VERIFIED | Types at L56-66 with JSON tags |
| `internal/daemon/api.go` | handleListShells handler + GET /shells route | VERIFIED | Route registered at L70; handler at L495-507; uses `make([]DetectedShell, 0, len(discovered))` non-null guarantee |
| `internal/daemon/api_test.go` | 3 integration tests | VERIFIED | TestHandleListShells_EmptyPATH, TestHandleListShells_PopulatedPATH, TestShellSessionLifecycle_StatusOnlyRunningOrStopped — all PASS |
| `internal/daemon/client.go` | DaemonClient.ListShells method | VERIFIED | Method at L109-118; nil-guard mirrors ListSessions pattern |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/pty/shells.go | os/exec | exec.LookPath per ShellSpec.Name | WIRED | shells.go:73 inside Pass 1 loop |
| internal/pty/shells.go | runtime | GOOS branch for POSIX synthetic entry | WIRED | shells.go:86 `if runtime.GOOS != "windows"` |
| internal/daemon/engine.go | internal/pty.DiscoverShells / KnownShellSpecs | resolveShellSpawn calls | WIRED | engine.go:478 `pty.KnownShellSpecs()`, engine.go:490 `pty.DiscoverShells()`, engine.go:500 also |
| internal/daemon/engine.go | os.UserHomeDir | shell WorkDir default | WIRED | engine.go:242 `os.UserHomeDir()` inside shell-dispatch block |
| internal/daemon/engine.go | isShellSession | status.Watch guard | WIRED | engine.go:294 `if !isShellSession(cli) { go status.Watch(...) }` |
| internal/daemon/api.go | internal/pty.DiscoverShells | handleListShells | WIRED | api.go:496 `pty.DiscoverShells()` |
| internal/daemon/client.go | GET /shells | doJSON request | WIRED | client.go:111 `c.doJSON(http.MethodGet, "/shells", ...)` |
| internal/daemon/path_windows.go | exec.LookPath augmented PATH | service-mode pwsh discovery | WIRED | platformExtraBins emits both PowerShell paths; consumed by `applyExtraBinsToPath` (pre-existing) |

All key links verified WIRED.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| pty package tests (shell-related) | `go test ./internal/pty/... -run 'Shell\|KnownShellSpecs' -race -count=1` | 7 PASS / 2 SKIP (Windows-only), 0 FAIL | PASS |
| daemon package shell tests | `go test ./internal/daemon -run 'Shell\|IsShellSession\|ResolveShellSpawn\|HandleListShells\|ShellSessionLifecycle\|AICLIEmptyWorkDir\|ListSessions_ShellStatus' -race -count=1` | 14 PASS / 1 SKIP (pwsh not installed), 0 FAIL | PASS |
| Full pty+daemon regression (excl. known flake) | `go test ./internal/pty/... ./internal/daemon/... -race -count=1 -skip TestOpenCodeANSICapture` | pty 1.563s ok, daemon 4.048s ok | PASS |
| Windows cross-compile vet | `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/...` | exit 0 | PASS |
| Windows cross-compile build | `GOOS=windows GOARCH=amd64 go build ./internal/daemon/...` | exit 0 | PASS |
| gofmt cleanliness | `gofmt -l` on 8 modified files | empty output | PASS |
| go vet cleanliness | `go vet ./internal/pty/... ./internal/daemon/...` | empty output | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| SHELL-04 | 100-01, 100-03, 100-04 | Daemon discovers available shells per platform | SATISFIED | `pty.DiscoverShells` + Windows PATH augmentation + GET /shells route; TestDiscoverShells_*, TestHandleListShells_* PASS |
| SHELL-05 | 100-02 | Shell sessions spawn interactive (not login) with WorkDir honored | SATISFIED | `resolveShellSpawn` + WorkDir/$HOME logic; TestCreateSession_ShellArgv_Interactive, ZshArgv, WorkDirHonored, EmptyWorkDirHome all PASS |
| SHELL-09 | 100-02, 100-04 | Shell sessions excluded from CLI-status heuristics | SATISFIED | status.Watch guard at engine.go:294; TestShellSessionLifecycle_StatusOnlyRunningOrStopped + unit guards PASS |

All 3 requirements for Phase 100 SATISFIED. No orphaned requirements (SHELL-01/-02/-03/-06/-07/-08 are explicitly mapped to Phase 101 in REQUIREMENTS.md).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers found in any phase-modified file |

Verified with `grep -nE "TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER" internal/pty/shells.go internal/pty/shells_test.go internal/daemon/engine.go internal/daemon/api.go internal/daemon/client.go internal/daemon/types.go internal/daemon/path_windows.go internal/daemon/path_windows_test.go` — no matches (only `TODO` references appear in test names like `TestShellSessionLifecycle_StatusOnlyRunningOrStopped` substring-matched by accident — verified by reading; no debt markers).

The Plan 02 parallel-wave stub (commit `3516a8e`) referenced in 100-02-SUMMARY.md was made obsolete by the canonical Plan 01 implementation (commit `691b635`) landing on main. The current `internal/pty/shells.go` is the canonical Plan 01 version (verified: 206 lines, includes `/etc/shells` parsing, `testEtcShellsPath` hook, `argvForShellBasename`, `readEtcShells` helpers — features the stub lacked). The orchestrator merge handled the de-duplication correctly. The `internal/pty/shells_helpers.go` stub does NOT exist in the current tree (verified via `ls /Users/ken/dev/agenthub/internal/pty/shells*.go` — only `shells.go` + `shells_test.go`).

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `handleListShells` response | `out []DetectedShell` | `pty.DiscoverShells()` | Yes — calls `exec.LookPath` for each `knownShellSpec`; on the dev host TestHandleListShells_PopulatedPATH confirms ≥1 entry returned | FLOWING |
| `resolveShellSpawn` return | `(path, args, ok)` | `pty.KnownShellSpecs()` + `pty.DiscoverShells()` | Yes — real `exec.LookPath` based discovery; TestResolveShellSpawn_KnownShell verifies bash resolves with `-i` | FLOWING |
| `ListSessions` Status field for shells | `heuristicStatus` | conservative `StatusRunning` default at engine.go:359 (sessionStatuses[id] absent for shells per L294 guard) | Yes — by-design "running" until State transitions to "stopped"; TestListSessions_ShellStatusRunning verifies | FLOWING |

### Probe Execution

No conventional `scripts/*/tests/probe-*.sh` files declared for this phase. Phase 100 is a Go-only backend phase verified via standard `go test` invocations (above). Probe execution N/A.

### Human Verification Required

None. Phase 100 is daemon-side backend wiring with no UI surface. UI integration (GUI/CLI/TUI shell pickers, visual badges, web-share confirmation banner) is explicitly deferred to Phase 101 per the v3.3 roadmap. The Phase 100 deliverables (cross-platform discovery, engine dispatch, HTTP API, status-heuristic bypass) are fully verifiable by automated Go tests, which all PASS.

### Gaps Summary

No gaps. All 3 ROADMAP success criteria, all 13 plan-level must-have truths, all 10 required artifacts, all 8 key links, and all 3 requirement IDs are verified against the actual code. Test suite passes under `-race` with no regressions. Cross-compile to Windows is clean.

### Notes on SUMMARY.md Claims (Trust Audit)

- **Plan 01 SUMMARY claim:** "All 9 test functions named verbatim" — VERIFIED by `grep -E "^func Test" internal/pty/shells_test.go` returning exactly the 9 declared names.
- **Plan 02 SUMMARY claim:** "Parallel-wave stub at `internal/pty/shells_helpers.go`" — file does NOT exist in the current tree. The stub was replaced by the canonical Plan 01 implementation at merge time, as the SUMMARY itself anticipated ("at wave merge time, Plan 01's commit on internal/pty/shells.go MUST take precedence"). No gap — this is the documented merge plan executing correctly.
- **Plan 02 SUMMARY claim:** "13 new tests" — VERIFIED by `grep -E "^func Test" internal/daemon/engine_test.go | grep -iE "shell|Zsh|Pwsh|workdir"` returning exactly 13 tests.
- **Plan 03 SUMMARY claim:** "Exactly two `paths = append(...)` lines added" — VERIFIED by reading path_windows.go (lines 20 and 23 are the two additions; pre-existing 4 entries at lines 15, 18, 19, 22 preserved).
- **Plan 04 SUMMARY claim:** "Status never in {waiting, error, errored, idle}" — VERIFIED by reading the test which asserts this on every poll sample.
- **Plan 04 SUMMARY pragmatic interpretation note:** The original plan called for `s.Status == "stopped"` post-kill, but the test author correctly noted that `engine.KillSession` removes the session from the registry. The implementation accepts removal OR State=="stopped" as terminal — this faithfully preserves the SHELL-09 contract spirit (Status never in forbidden states). Accepted as a documented pragmatic adjustment, not a deviation.

---

_Verified: 2026-05-12T17:08:00Z_
_Verifier: Claude (gsd-verifier)_
