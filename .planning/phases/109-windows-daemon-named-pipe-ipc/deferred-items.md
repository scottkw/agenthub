# Phase 109 — Deferred Items

Items discovered during Phase 109 execution that are out-of-scope for this plan (per the executor's scope-boundary rule: "Only auto-fix issues DIRECTLY caused by the current task's changes. Pre-existing warnings, linting errors, or failures in unrelated files are out of scope.").

## Pre-existing test failures on `main` (NOT introduced by Phase 109)

Discovered during Task 2 of Plan 109-01 while running the macOS smoke (`go test -race -short ./internal/daemon/`).

Three failures exist on `main` (`9cc1087`, "fix(109-01-PLAN): correct two shell-verify bugs flagged by plan-checker") BEFORE any Phase 109 cherry-pick lands:

1. **`TestAPIGetShellWebShareWarned_Default`** (`internal/daemon/api_test.go:1592`)
   `default value: got true, want false`
2. **`TestDaemonClient_GetSetShellWebShareWarned_RoundTrip`** (`internal/daemon/api_test.go:1638`)
   `initial value: got true, want false`
3. **`TestSetShellWebShareWarned_Default`** (`internal/daemon/engine_test.go:905`)
   `default GetShellWebShareWarned: got true, want false`

**Reproduction on plain `main`:**
```
git checkout main
go test -race -short -run 'TestAPIGetShellWebShareWarned_Default|TestDaemonClient_GetSetShellWebShareWarned_RoundTrip|TestSetShellWebShareWarned_Default' ./internal/daemon/
# All three fail
```

**Why deferred from Phase 109:** Touches `ShellWebShareWarned` default-value logic (introduced by `dbd95a7 feat(101-01): ShellWebShareWarned persistence + routes`, NOT by Phase 109 code). Phase 109 is a transport substitution (`net.Listen("unix", ...)` → `winio.ListenPipe(...)`); it does not touch the WebShareWarned default or its tests.

**Recommended next step:** File a follow-up bug on `scottkw/agenthub` titled along the lines of "ShellWebShareWarned default is `true`, tests expect `false`". Determine intent — is the default supposed to be `false` (and the live code wrong) or `true` (and the tests are stale)? Fix in a dedicated patch-release bug-sweep plan, not in Phase 109.

**Impact on Phase 109 verification:** The three failures pre-exist; they do not block Phase 109's IPC-01..06 work. Phase 109 verification (Task 3's full-suite `go test ./...`) will continue to surface these same three failures — they are not a Phase 109 regression. The plan's `go test -race -short ./internal/daemon/` and `go test -race -short ./...` verify-steps in Tasks 2 and 3 will be reported as "PASS modulo pre-existing failures already present on `main` (see `deferred-items.md`)".
