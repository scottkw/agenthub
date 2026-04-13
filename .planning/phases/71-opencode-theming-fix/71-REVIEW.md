---
phase: 71-opencode-theming-fix
reviewed: 2026-04-13T20:38:07Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - internal/daemon/engine.go
  - internal/daemon/engine_test.go
  - internal/daemon/opencode_ansi_test.go
  - internal/pty/detect_test.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 71: Code Review Report

**Reviewed:** 2026-04-13T20:38:07Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Phase 71 implements per-agent env injection to force OpenCode's "system" theme via a managed `opencode-tui.json` file and `OPENCODE_TUI_CONFIG` env var. The implementation is clean and well-scoped: the env var is correctly guarded to opencode-only sessions, the managed config file uses a hardcoded constant (no injection risk), and the `daemonConfigDir()` function mirrors the existing `configDir()` pattern in `app.go`.

The code changes are small, focused, and follow existing project patterns. Two warnings found regarding silent error suppression in `ensureOpenCodeTUIConfig` and a minor goroutine leak in test infrastructure. Three informational items noted for potential improvements. No critical/blocking issues.

The SC-1 gap (live theme switching) is documented in 71-04-SUMMARY.md and explicitly out of scope for this review.

## Warnings

### WR-01: Silent failure on os.WriteFile in ensureOpenCodeTUIConfig

**File:** `internal/daemon/engine.go:54`
**Issue:** `_ = os.WriteFile(path, content, 0644)` silently discards write errors. If the write fails (disk full, permission denied after directory creation, read-only filesystem), the function returns a path to a file that does not exist or has stale content. The caller (`NewSessionEngine`) stores this path in `e.opencodeTUIConfig` and later injects it as `OPENCODE_TUI_CONFIG=<path>`. OpenCode would then receive an env var pointing to a missing or empty file, and silently fall back to its default theme -- defeating the entire purpose of the feature.

This is distinguished from the `_ = os.MkdirAll` on line 43, which is acceptable because `daemonConfigDir` mirrors `app.go:configDir()` which uses the same pattern, and directory creation failure is immediately visible when the subsequent WriteFile fails.

The research document (Pitfall 4, "Common Pitfalls") correctly identifies that the file is intentionally overwritten on every launch, but does not address the case where the write itself fails.

**Fix:** Log the error or return it to the caller. Since `ensureOpenCodeTUIConfig` is called at engine init (not in a hot path), logging is the minimal change:
```go
func ensureOpenCodeTUIConfig(dir string) string {
	path := filepath.Join(dir, "opencode-tui.json")
	content := []byte("{\"$schema\":\"https://opencode.ai/tui.json\",\"theme\":\"system\"}\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		// Log but don't fail startup -- opencode sessions will use their default theme.
		fmt.Fprintf(os.Stderr, "agenthub: warning: could not write managed opencode config: %v\n", err)
		return "" // empty path prevents OPENCODE_TUI_CONFIG injection
	}
	return path
}
```
Returning `""` on error is safe because `CreateSession` already guards with `e.opencodeTUIConfig != ""` (line 89). This turns a silent corruption into an observable degradation.

### WR-02: Goroutine leak in TestCreateSession_OpenCodeEnv via Hub.Run

**File:** `internal/daemon/engine_test.go:263`
**Issue:** `TestCreateSession_OpenCodeEnv` calls `e.CreateSession()` which internally calls `e.manager.Create(id, sess, sess, resizeFn)` and `go status.Watch(hub, ...)`. The spy backend returns a `*pty.Session` with nil `pty` field. When `Hub.Run()` calls `sess.Read()`, it gets an immediate error and shuts down -- but the `status.Watch` goroutine blocks forever on `sub.Msgs` channel because the hub is already shut down and never closes subscriber channels.

Each call to `CreateSession` in this test leaks one goroutine (three total: opencode, claude, codex). This is unlikely to cause flaky tests today since Go's test runner doesn't check for leaked goroutines, but it could mask real leaks if a goroutine leak detector (like `goleak`) is added later, and it's contrary to the test's stated design goal of "no goroutine leaks" (from 71-01-SUMMARY.md).

**Fix:** Either (a) have `spyBackend` return a session that doesn't trigger Hub creation (would require refactoring CreateSession), or (b) add cleanup that kills the spy sessions:
```go
t.Cleanup(func() { _ = e.KillSession("spy-id") })
```
This calls `e.manager.Remove("spy-id")` which shuts down the hub and unblocks the Watch goroutine. Note: all three sub-tests (opencode, claude, codex) use separate engines, so each needs its own cleanup.

## Info

### IN-01: Duplicated configDir logic between app.go and daemon package

**File:** `internal/daemon/engine.go:37-45`
**Issue:** `daemonConfigDir()` duplicates `app.go:configDir()` (lines 316-324). The implementation is identical: `os.UserConfigDir()` with `os.TempDir()` fallback, joined with "agenthub", `MkdirAll` with 0700. This is documented as intentional in 71-02-SUMMARY.md ("internal packages cannot import main; 6-line function, acceptable duplication"). The code comment on line 36 also explains this.

**Note:** This is informational only -- the duplication is justified. If the config directory logic ever changes (e.g., XDG_CONFIG_HOME override), both copies need updating. Consider extracting to an internal utility package in a future refactor if more functions need the config directory.

### IN-02: opencode_ansi_test.go diagnostic test has no hard assertions

**File:** `internal/daemon/opencode_ansi_test.go:125-135`
**Issue:** `TestOpenCodeANSICapture` uses `t.Logf` for all findings, including the critical case where assumption A1 is violated (line 126). This is documented as intentional (71-03-SUMMARY.md: "diagnostic test approach rather than hard assertion") because the go-pty test environment lacks an OSC responder, making the system theme fall back to OpenCode's default. The test serves as a baseline and logs its findings for human analysis.

**Note:** This is appropriate for this phase. The test is self-skipping in CI and short mode. If future work resolves the OSC responder limitation (e.g., a mock responder), this test could be upgraded to a hard assertion.

### IN-03: Test file uses three separate engine+spy instances instead of resetting

**File:** `internal/daemon/engine_test.go:258-316`
**Issue:** `TestCreateSession_OpenCodeEnv` creates three separate `NewSessionEngine()` + `spyBackend` pairs (for opencode, claude, codex). Each `NewSessionEngine()` call writes the managed tui.json file and creates all subsystems. This is not wrong, but it's heavier than necessary -- the test could reset the spy between calls on a single engine.

**Note:** The current approach is clearer to read and avoids shared-state issues. The overhead is negligible (3 engine inits, ~1ms each). Flagging only because the pattern creates the goroutine leaks noted in WR-02; a single engine with cleanup would reduce the leak surface.

---

_Reviewed: 2026-04-13T20:38:07Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
