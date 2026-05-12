---
phase: 100-shell-session-backend-discovery
plan: 03
subsystem: daemon/path-augmentation
tags: [windows, powershell, path, daemon, service-mode, shell-04]
requires: []
provides:
  - "Windows PATH augmentation now includes PowerShell 7 install paths"
  - "exec.LookPath('pwsh') succeeds in service-mode daemon on Windows"
affects:
  - "Plan 01 DiscoverShells() — pwsh is now discoverable on Windows service-mode"
  - "Plan 04 GET /shells integration test on Windows CI will return a pwsh entry"
tech-stack:
  added: []
  patterns:
    - "Backtick raw-string literals for Windows absolute paths (existing convention)"
    - "Conditional env-var-dependent entry inside LOCALAPPDATA guard"
    - "Build-tag-gated test file (//go:build windows) with cross-compile vet verification on macOS"
key-files:
  created:
    - internal/daemon/path_windows_test.go
  modified:
    - internal/daemon/path_windows.go
decisions:
  - "PowerShell\\7 entry hardcoded (not env-dependent) — matches existing Tailscale convention"
  - "WindowsApps entry kept inside LOCALAPPDATA conditional — graceful skip when env var unset"
  - "Local containsString helper instead of slices.Contains — keeps test file stdlib-only and flat"
metrics:
  duration: "~7 minutes"
  completed: "2026-05-12T23:36:26Z"
  tasks: 2
  files_changed: 2
requirements:
  - SHELL-04
---

# Phase 100 Plan 03: Windows PowerShell PATH Augmentation Summary

Extended `internal/daemon/path_windows.go::platformExtraBins` to include `C:\Program Files\PowerShell\7` and `%LOCALAPPDATA%\Microsoft\WindowsApps`, making `pwsh.exe` discoverable via `exec.LookPath` in service-mode Windows daemons that do not inherit the user's PATH.

## What Changed

### `internal/daemon/path_windows.go` (modified, +2 lines)

Exactly two `paths = append(...)` lines added:

```diff
 	if local := os.Getenv("LOCALAPPDATA"); local != "" {
 		paths = append(paths, filepath.Join(local, "pnpm"))
 		paths = append(paths, filepath.Join(local, "Programs", "nodejs"))
+		paths = append(paths, filepath.Join(local, "Microsoft", "WindowsApps"))
 	}
 	paths = append(paths, `C:\Program Files\Tailscale`)
+	paths = append(paths, `C:\Program Files\PowerShell\7`)
 	return paths
```

- `Microsoft\WindowsApps` is appended **inside** the `LOCALAPPDATA` conditional so it is silently skipped when the env var is empty (graceful degradation — matches the existing pnpm/nodejs pattern).
- `C:\Program Files\PowerShell\7` is appended **outside** any conditional as a hardcoded absolute path using a backtick raw-string literal — matches the existing `C:\Program Files\Tailscale` convention.
- All four pre-existing entries (`APPDATA\npm`, `LOCALAPPDATA\pnpm`, `LOCALAPPDATA\Programs\nodejs`, `C:\Program Files\Tailscale`) are preserved verbatim in their original order. Regression guarded by `TestPlatformExtraBins_PreservesExistingEntries`.

### `internal/daemon/path_windows_test.go` (new, 85 lines)

Build-tag-gated test file (`//go:build windows`) with three test functions plus a local `containsString` helper:

1. **`TestPlatformExtraBins_WindowsIncludesPowerShell`** — asserts both new entries are returned when `LOCALAPPDATA` is set.
2. **`TestPlatformExtraBins_PreservesExistingEntries`** — regression guard for the four pre-existing entries.
3. **`TestPlatformExtraBins_LocalAppDataEmpty`** — asserts WindowsApps is skipped when `LOCALAPPDATA` is empty while `PowerShell\7` is still emitted.

Tests use `t.Setenv` for env isolation (no global state mutation). Local `containsString` helper avoids `slices.Contains` generics dependency.

## POSIX path_other.go Confirmation

`internal/daemon/path_other.go` is **untouched**. Last modification predates this plan (commit `d072fee` from phase 68-01). Verified via `git log --oneline -5 -- internal/daemon/path_other.go` showing no commits from this plan, and the file's contents still return `nil` for all non-Windows platforms — exactly as required by the success criteria.

## Test Execution Gating

Per the `<parallel_execution>` note in the orchestrator prompt, Windows-only tests cannot be executed on the macOS dev box. The verification chain used here is:

| Gate | Command | Result |
|------|---------|--------|
| Cross-compile vet | `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/...` | exit 0 |
| Cross-compile build | `GOOS=windows GOARCH=amd64 go build ./internal/daemon/...` | exit 0 |
| gofmt (both files) | `gofmt -l internal/daemon/path_windows.go internal/daemon/path_windows_test.go` | empty output |
| Content presence | `grep PowerShell\|WindowsApps\|Tailscale` on path_windows.go | all present |
| Diff size | `git diff --numstat` on path_windows.go | `2 0` (2 insertions, 0 deletions) |

**Actual test execution is deferred to Windows CI.** On a Windows runner, the canonical command is:

```
go test ./internal/daemon -run TestPlatformExtraBins -race -count=1
```

This is consistent with verification.md noting that path_windows.go changes are gated by the Windows CI runner. Empirically, the change is also validated by Plan 04's `GET /shells` integration test on Windows CI returning a `pwsh` entry — that downstream signal proves end-to-end discoverability.

## Validation Map Footnote

This plan does not appear in `100-VALIDATION.md`'s per-task validation table because that file focuses on the discovery/spawn/status surfaces. The `path_windows.go` PATH augmentation is validated empirically by:

1. **Static gates (this plan):** cross-compile vet, gofmt, content greps.
2. **Windows CI unit tests (post-merge):** `TestPlatformExtraBins_*` from Task 1.
3. **Windows CI integration test (Plan 04):** `GET /shells` returns a `pwsh` entry — proves `exec.LookPath` finds pwsh.exe through the augmented PATH.

## Commits

| Task | Type | Hash | Description |
|------|------|------|-------------|
| 1 | test | `aad44f1` | Add Windows-tagged tests for PowerShell PATH augmentation (RED) |
| 2 | feat | `6e9f34a` | Extend platformExtraBins with PowerShell install paths (GREEN) |

REFACTOR gate not exercised — implementation is two clean `append()` calls, no cleanup needed.

## TDD Gate Compliance

- ✅ RED gate: `test(100-03): add Windows-tagged tests…` commit precedes implementation.
- ✅ GREEN gate: `feat(100-03): extend Windows platformExtraBins…` commit follows tests.
- ✅ Behavior added is covered by tests written first (TestPlatformExtraBins_WindowsIncludesPowerShell asserts the two new entries; the implementation appends exactly those two entries).
- ⚪ REFACTOR gate: not needed — two-line diff has no structural complexity to clean up.

## Deviations from Plan

None — plan executed exactly as written.

The plan called for backtick raw-string literals for hardcoded Windows paths (matching the existing Tailscale convention) and for the WindowsApps entry to sit inside the LOCALAPPDATA conditional. Both were applied as specified.

## RESEARCH.md Pitfall 1 Mitigation

The plan implements RESEARCH.md "Pitfall 1" (Windows service-mode PATH does not inherit user PATH) and Assumption A5 (pwsh.exe lives in `C:\Program Files\PowerShell\7` and/or `%LOCALAPPDATA%\Microsoft\WindowsApps`) exactly as documented. No deviation from the research-phase mitigation.

## Threat Model Compliance

The plan's `<threat_model>` lists T-100-04 (Spoofing — pwsh.exe path resolution) with disposition `mitigate`. The mitigation rationale is:

- `C:\Program Files\PowerShell\7` is administrator-writable only on stock Windows (NTFS default ACLs preserved). Daemon-process privilege equals discovered-binary privilege.
- `%LOCALAPPDATA%\Microsoft\WindowsApps` is user-writable, but Microsoft Store install paths are gated by Store package signing.
- `applyExtraBinsToPath` in `path.go` **appends** (not prepends) these entries to PATH, so user PATH entries earlier in PATH take precedence — `exec.LookPath` will prefer a user-installed pwsh.exe over our appended candidates.

This matches the threat model exactly. No new threat surface introduced beyond what the threat register already accepts.

## Threat Flags

None. No new network endpoints, auth paths, file-access patterns outside the documented PATH augmentation, or schema changes were introduced. The two new PATH entries are within the existing `platformExtraBins` trust boundary documented in the threat register.

## Known Stubs

None. No placeholder values, empty-array UI fallbacks, or TODO markers introduced.

## Self-Check: PASSED

- ✅ `internal/daemon/path_windows.go` exists and contains `PowerShell`, `WindowsApps`, `Tailscale`, `Programs`, `pnpm`, `npm`.
- ✅ `internal/daemon/path_windows_test.go` exists with all three required test functions and `//go:build windows` as the first non-blank line.
- ✅ Commit `aad44f1` exists in git log.
- ✅ Commit `6e9f34a` exists in git log.
- ✅ `internal/daemon/path_other.go` not modified by this plan.
- ✅ `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/...` exits 0.
- ✅ `GOOS=windows GOARCH=amd64 go build ./internal/daemon/...` exits 0 (parallel_execution note).
- ✅ `gofmt -l` produces no output on either file.
