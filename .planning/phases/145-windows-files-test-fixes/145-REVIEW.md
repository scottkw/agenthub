---
phase: 145-windows-files-test-fixes
reviewed: 2026-06-22T00:00:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - internal/files/handler_test.go
  - internal/files/write_test.go
  - internal/files/concurrent_read_windows_test.go
  - internal/files/concurrent_read_unix_test.go
  - frontend/src/lib/__tests__/useFilesCapability.test.tsx
  - .github/dependabot.yml
  - go.mod
findings:
  critical: 0
  warning: 1
  info: 4
  total: 5
status: issues_found
---

# Phase 145: Code Review Report

**Reviewed:** 2026-06-22
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Phase 145 is a test-and-dependency-hardening phase: it makes the `internal/files`
suite pass on Windows CI without regressing macOS/Linux, pins `go-webview2` to a
wails-v2.10.2-compatible version, and adds a Dependabot ignore so the pin is not
auto-bumped. No production code in `internal/files` was modified.

I verified the load-bearing facts adversarially rather than trusting the summary:

- **Build-tagged helper pair is correct.** `concurrent_read_windows_test.go`
  (`//go:build windows`) and `concurrent_read_unix_test.go` (`//go:build !windows`)
  partition the build space exhaustively and non-overlappingly. Both export an
  identical signature `readFilePlatformSafe(path string) ([]byte, error)`, which is
  the symbol consumed at `write_test.go:174`. Exactly one definition compiles per
  platform — no duplicate-symbol or undefined-symbol risk.
- **Security assertions preserved.** The `$HOME` redirect in
  `TestHandlerUpload_FilenameSanitized` and `TestDenylist_NonHomeRootedUnaffected`
  points `$HOME`/`USERPROFILE` at a *separate* `t.TempDir()` so the sandbox root is
  genuinely outside home. The original assertions are intact and arguably stronger:
  the sanitization test still asserts the file lands at `root/.bashrc` AND still
  asserts nothing was written to the parent-dir escape target
  (`handler_test.go:733-740`). The denylist remains enforced everywhere it was
  before — `TestDenylist_HomeRooted`, `_CaseVariation`, and `_DaemonConfigDir` are
  untouched and still assert `ErrProtectedSystemFile`.
- **React cleanup is correct.** `afterEach` unmounts every tracked root inside
  `act()`, triggering both effect cleanups (`cancelled = true`), and the new
  `probeWrite` stub neutralizes the web-share async path that previously leaked a
  late `setState` past jsdom teardown. The tests call `useFilesCapability(client,
  'sid')` with `filesWriteSignal` left `undefined`, which is exactly the web-share
  branch that fires `probeWrite` (useFilesCapability.ts:108-134) — so the stub is
  load-bearing, not decorative.
- **Pin is sound.** `go.mod` and `go.sum` both reference `go-webview2 v1.0.19` only;
  no stray `v1.0.22` remains in either file, and `go.sum` carries both the module
  and `/go.mod` hash lines. wails is at `v2.10.2`. The Dependabot ignore targets the
  correct module path and documents the rationale.

The one Warning below is a latent correctness gap in the new Windows helper's
resource handling, not a regression in the phase's stated goal. The Info items are
maintainability nits.

## Warnings

### WR-01: Windows `readFilePlatformSafe` leaks the raw handle if `os.NewFile` path is unreachable / partial-failure window

**File:** `internal/files/concurrent_read_windows_test.go:27-41`
**Issue:** `syscall.CreateFile` returns a raw `Handle`. It is only wrapped in an
`*os.File` (whose `Close` the `defer` will run) on the line *after* the error check.
The current code is safe for the two states it handles (CreateFile error → return;
CreateFile success → wrap + defer Close). However, the pattern is fragile: any future
edit that adds a fallible step between the successful `CreateFile` and the
`os.NewFile(uintptr(h), ...)` wrap (for example, a `path` re-validation, a seek, or a
second syscall) would leak the OS handle, because nothing closes `h` until it is
adopted by an `*os.File`. In a test that opens this handle 200+ times per run inside a
tight reader loop, a leaked handle also blocks the very POSIX-semantics rename the
helper exists to permit, which would manifest as a flaky, hard-to-diagnose Windows-only
failure rather than a clean error. Defense-in-depth: bind the cleanup to the raw handle
immediately.
**Fix:**
```go
h, err := syscall.CreateFile(
    pathp,
    syscall.GENERIC_READ,
    syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
    nil,
    syscall.OPEN_EXISTING,
    syscall.FILE_ATTRIBUTE_NORMAL,
    0,
)
if err != nil {
    return nil, err
}
// Adopt the handle into an *os.File immediately so Close is guaranteed even if
// any future fallible step is inserted before the read.
f := os.NewFile(uintptr(h), path)
defer f.Close()
return io.ReadAll(f)
```
(This is a minimal reordering — `os.NewFile` itself does not fail — but it documents
the invariant and makes the helper robust to future edits. Severity is Warning, not
Critical, because the code as written today does not actually leak.)

## Info

### IN-01: `dependabot.yml` ignore lacks an explicit `versions`/`update-types` qualifier — silently pins ALL future updates

**File:** `.github/dependabot.yml:32`
**Issue:** `ignore: - dependency-name: "github.com/wailsapp/go-webview2"` with no
`versions` or `update-types` qualifier suppresses *every* Dependabot update for that
module indefinitely, including a future security patch in the `v1.0.x` line. The
comment says "Bump only together with a compatible wails/v2 upgrade," which is the
intent, but the broad ignore also masks patch-level CVEs in the pinned branch. This is
acceptable for an indirect, build-time-only dependency, but worth a note so a reviewer
re-evaluates it when wails is next upgraded (the comment should be the trigger to
*remove* this ignore, not just edit the pin).
**Fix:** Consider scoping to major/minor only so security patches still surface:
```yaml
    ignore:
      - dependency-name: "github.com/wailsapp/go-webview2"
        update-types: ["version-update:semver-major", "version-update:semver-minor"]
```
Or leave as-is and add "remove this entry on the next wails bump" to the phase exit
notes / TESTING.md convention.

### IN-02: New test files not reflected in TESTING.md traceability map

**File:** `internal/files/concurrent_read_windows_test.go`, `internal/files/concurrent_read_unix_test.go`
**Issue:** The repo CLAUDE.md standing rule (and TESTING.md Section 6) requires that
"every future phase that adds, renames, or removes tests must update TESTING.md."
Phase 145 adds two new `*_test.go` files. They are platform-helper files rather than
standalone requirement coverage, but the Suite Manifest count ("346 `*_test.go` files")
and, if either maps to a v4.0 requirement, the Traceability Map (Section 4) should be
checked. The traceability path-check (`tests/check-traceability-paths.sh`) only
validates that mapped paths *exist* — it will not catch a missing row for a newly added
file. Confirm whether these helpers need a manifest count bump.
**Fix:** Run `bash tests/check-traceability-paths.sh`, update the Suite Manifest file
count in TESTING.md Section 2, and add traceability rows if these helpers cover a v4.0
requirement (FSW-01 concurrency invariant is the candidate).

### IN-03: `mountedRoots` module-level array is shared mutable test state — fine today, brittle under parallelism

**File:** `frontend/src/lib/__tests__/useFilesCapability.test.tsx:22`
**Issue:** `mountedRoots` is a module-scoped array mutated by `renderHook` (push) and
cleared in `afterEach`. vitest runs tests within a file serially by default, so this is
correct now. But if this file is ever switched to `test.concurrent` or the suite gains
`{ pool: 'threads', isolate: false }` semantics, the shared array would interleave roots
across tests and `afterEach` could unmount another test's still-live root. Low risk
given current config; flagged so a future concurrency change does not silently corrupt
cleanup.
**Fix:** No change required now. If concurrency is introduced, move `mountedRoots` into
a per-test fixture or use vitest's `onTestFinished` to scope cleanup to the owning test.

### IN-04: `TestHandlerUpload_FilenameSanitized` relies on `t.TempDir()` placement assumptions documented in a comment, not asserted

**File:** `internal/files/handler_test.go:696-702`
**Issue:** The comment explains that on Windows `t.TempDir()` lands under
`%USERPROFILE%`, which is why `$HOME` must be redirected. This reasoning is sound, but
the test does not assert the precondition it depends on (that `newHandler`'s sandbox
root is now outside the redirected `fakeHome`). If a future Go release changes
`t.TempDir()` placement, or `newHandler` is refactored to root under `$HOME`, the test
could pass for the wrong reason (or the denylist could fire and produce a confusing 403).
The redirect makes the test robust regardless, so this is informational.
**Fix:** Optionally add a one-line guard after `setHomeEnv` confirming the sandbox root
is not under `fakeHome` (mirrors the `filepath.Rel` + `HasPrefix("..")` pattern already
used in `TestDenylist_DaemonConfigDir`), making the test's correctness self-evident
rather than comment-dependent.

---

_Reviewed: 2026-06-22_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
