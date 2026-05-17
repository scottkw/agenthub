---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
fixed_at: 2026-05-13T22:15:00Z
review_path: .planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 107: Code Review Fix Report

**Fixed at:** 2026-05-13T22:15:00Z
**Source review:** `.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 4 (CR-01, WR-01, WR-02, WR-03)
- Fixed: 4
- Skipped: 0

---

## Fixed Issues

### CR-01: autoCloseRef not consulted in SHELL-12 exit handler

**Files modified:** `frontend/src/App.tsx`, `frontend/src/components/__tests__/App.shellExit.test.tsx`
**Commit:** `401eec5`
**Applied fix:** Wrapped the `handleCloseTabRef.current?.(data.sessionId)` + `return` block inside `if (autoCloseRef.current)` in the `session:exit` EventsOn callback. When `autoCloseRef.current` is `true` (default), the tab closes immediately as before. When `false` (user opted out of auto-close), control falls through to the `setSessionExits` call so the ExitToast is shown — honoring the user's preference. Updated five existing test assertions in `App.shellExit.test.tsx` to match the new guard structure (the old shape checked for unconditional close; the new shape checks for guarded close). Added three new CR-01 assertions that lock the `autoCloseRef.current` guard shape permanently.

---

### WR-01: SetShellPath accepts directory paths

**Files modified:** `internal/daemon/engine.go`, `internal/daemon/engine_settings_test.go`, `internal/daemon/api_test.go`
**Commit:** `d137d9b`
**Applied fix:** Added `if info.IsDir() { return fmt.Errorf("path %q is a directory, not an executable", path) }` between the `os.Stat` call and the execute-bit check in `SetShellPath`. Added `TestSetShellPathValidation` in `engine_settings_test.go` covering: directory rejected with "is a directory" message, non-existent path rejected, non-executable file rejected, valid executable accepted, and empty-string clear accepted and returning non-empty platform default. Also added `TestHandleUpdateShellPath_Directory_Returns400` in `api_test.go` to verify the 400 at the HTTP API layer.

---

### WR-02: False "Saved!" indicator when shell-path save fails

**Files modified:** `frontend/src/components/SettingsTab.tsx`, `frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx`
**Commit:** `75f9822`
**Applied fix:** Introduced a `let shellPathOk = true` flag before the inner try/catch for `SetShellPath`. The catch block sets `shellPathOk = false` in addition to setting `shellPathError`. The `setSaved(true)` / `setTimeout` calls are now guarded by `if (shellPathOk)` so they only fire when the shell-path save actually succeeded. Added `WR-02` test (failure shows inline error, no "Saved!" button visible) and `WR-02b` test (success shows "Saved!" button, no error).

---

### WR-03: handleUpdateShellPath missing MaxBytesReader

**Files modified:** `internal/daemon/api.go`, `internal/daemon/api_test.go`
**Commit:** `2259ad1`
**Applied fix:** Added `r.Body = http.MaxBytesReader(w, r.Body, 8192)` as the first line of `handleUpdateShellPath`, matching the pattern used by `handleSetPluginSettings`, `handleSetSearchConfig`, `handleSetWebLinksConfig`, and `handleSetImageConfig`. Added `TestHandleUpdateShellPath_OversizedBody` verifying that a body of 8300 bytes is rejected (response is not 204).

---

## Skipped Issues

None — all in-scope findings were fixed.

---

## Verification

**Go:** `go test ./internal/daemon/... -race -count=1 -skip TestOpenCodeANSICapture` — PASS (all tests green)

**TypeScript:** `pnpm tsc --noEmit` — PASS (no errors)

**Frontend tests:** `pnpm test -- --run` — PASS (59 test files, 901 tests)

**gofmt:** `gofmt -l ./internal/daemon/` — PASS (no output)

**go vet:** `go vet ./internal/daemon/...` — PASS (no output)

---

_Fixed: 2026-05-13T22:15:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
