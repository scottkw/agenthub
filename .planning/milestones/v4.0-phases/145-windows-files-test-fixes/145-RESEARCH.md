# Phase 145: Windows Files Test Fixes - Research

**Researched:** 2026-06-22
**Domain:** Go test platform portability — Windows path semantics, atomic rename, denylist home-root detection
**Confidence:** HIGH

## Summary

Three tests in `internal/files` fail on the `windows/amd64, windows-latest` CI runner while passing on macOS and Linux. Each failure has a distinct root cause. This research identifies the exact source-level mechanism for each failure and gives a concrete per-test fix recommendation (test-fix vs production-fix) with code evidence.

**Primary finding:** All three failures are TEST-SIDE bugs where POSIX-only assumptions are encoded in the test setup or assertions. The production code in `sandbox.go`/`write.go`/`handler.go` is correct cross-platform. Each test requires a targeted fix that makes the test OS-aware without weakening the security property it verifies.

**Primary recommendation:** Fix the three tests to account for Windows semantics (platform-gated sandbox root, `FILE_SHARE_DELETE` reader pattern, and a Windows-proof skip guard). No production code changes are required for the core logic.

---

## Project Constraints (from CLAUDE.md)

- Python: `uv` or `pip`, always in a virtualenv
- Go: `go fmt`, `golangci-lint`, context-aware functions
- Node: `pnpm` preferred
- TESTING.md standing convention: every phase that adds, renames, or removes tests MUST update TESTING.md Sections 2, 4, and 5 (`bash tests/check-traceability-paths.sh` before committing)
- Safety Rule 0: STOP on unknown or catastrophic failures; do not attempt further actions without user confirmation

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIX-02 | `internal/files` tests pass on Windows CI — filename sanitization and denylist path-rooting respect Windows path semantics (#101) | Per-test root cause identified; concrete fixes specified below with line-level code evidence |
</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Atomic file rename | OS / Kernel (Windows NtSetInformationFile) | `internal/files/sandbox.go` (caller) | The rename is a kernel operation; the Go layer wraps it |
| Denylist home-root detection | `internal/files/sandbox.go` (denylistCheck) | Test setup (setHomeEnv) | denylistCheck computes the detection; tests must replicate the OS environment correctly |
| Filename sanitization | `internal/files/write.go` Upload handler (filepath.Base) | validateAndClean (second layer) | filepath.Base is the first sanitization; validateAndClean catches any residual |
| CI verification | `build.yml` Windows matrix job | Developer cross-compilation (`GOOS=windows go vet`) | Windows runner is the only environment where failures manifest |

---

## Per-Test Root Cause Analysis and Fix Recommendations

### Test 1: `TestHandlerUpload_FilenameSanitized` (handler_test.go:695)

**What the test does:**
Uploads a multipart form with `filename="../../.bashrc"`, expects HTTP 200 and the file to land at `root/.bashrc` inside the sandbox. Uses `newHandler(t)` which roots the sandbox at `t.TempDir()`.

**What breaks on Windows CI:**

On the GitHub Actions `windows-latest` runner (Windows Server 2025, as of September 2025), `os.TempDir()` returns `C:\Users\runneradmin\AppData\Local\Temp` and `os.UserHomeDir()` returns `%USERPROFILE%` = `C:\Users\runneradmin`. Therefore `t.TempDir()` produces a path UNDER `%USERPROFILE%`, making the sandbox root home-rooted from `denylistCheck`'s perspective.

Code path:
1. `newHandler(t)` → `files.NewSandbox(t.TempDir())` → `rootPath = EvalSymlinks(C:\Users\runneradmin\AppData\Local\Temp\TestXxx\...)`
2. Upload handler calls `sb.WriteFileAtomic(".bashrc", data)` (after `filepath.Base("../../.bashrc") == ".bashrc"`)
3. `WriteFileAtomic` calls `denylistCheck(".bashrc")`
4. `denylistCheck` computes `home = EvalSymlinks(os.UserHomeDir()) = C:\Users\runneradmin`
5. `abs = filepath.Join(rootPath, ".bashrc")` — this is under `C:\Users\runneradmin\...`
6. `filepath.Rel(home, abs)` returns a path that does NOT start with `..` (it's a valid relative path under home)
7. `base = strings.ToLower(filepath.Base(abs)) = ".bashrc"` — matches the denylist
8. `denylistCheck` returns `ErrProtectedSystemFile`
9. Handler returns HTTP 403; test expects 200 → **FAIL**

**Why `filepath.Base` is not the issue:**
On Windows, `filepath.Base("../../.bashrc")` correctly returns `.bashrc` because Windows `IsPathSeparator` recognizes both `/` and `\`. [VERIFIED: Go stdlib source `/opt/homebrew/Cellar/go/1.26.4/libexec/src/os/path_windows.go:19`] The sanitization logic is correct. The failure is purely in the test environment setup.

**Fix recommendation: TEST-FIX**

The test must not use a sandbox rooted under `$HOME`. The `TestHandlerUpload_FilenameSanitized` test should create its sandbox in a directory that is provably NOT under `os.UserHomeDir()`. Options:

Option A (preferred): Use `os.MkdirTemp("", ...)` with an explicit base directory guaranteed to be outside `$HOME`, similar to how the test itself could set `HOME` to a fake dir via `setHomeEnv`. This is the safest approach.

Option B: Add a skip guard — if `t.TempDir()` is under `os.UserHomeDir()`, choose a different temp location (platform-gated logic in the helper).

The cleanest fix is to extract a helper `newHandlerOutsideHome(t)` that either (a) creates a temp dir in `os.TempDir()` when that is outside `$HOME`, or (b) explicitly sets HOME to a fake value (like `setHomeEnv(t, fakeHome)` does in the denylist tests) so that the test sandbox is guaranteed to be non-home-rooted.

**Code evidence (`write_test.go:466-473`):** The denylist tests already have the correct pattern:
```go
func setHomeEnv(t *testing.T, dir string) {
    t.Helper()
    t.Setenv("HOME", dir)
    if runtime.GOOS == "windows" {
        t.Setenv("USERPROFILE", dir)
    }
}
```
The fix should either reuse this helper or add a `newHandlerWithNonHomeSandbox(t)` helper that calls `setHomeEnv(t, someOtherDir)` to redirect `$HOME` away from the test's sandbox root.

---

### Test 2: `TestDenylist_NonHomeRootedUnaffected` (write_test.go:581)

**What the test does:**
Verifies that writing `.bashrc` in a sandbox rooted OUTSIDE `$HOME` succeeds (denylist should not fire for non-home-rooted sandboxes).

**What breaks on Windows CI:**

The test has a skip guard at lines 588-591:
```go
home, _ := os.UserHomeDir()
if home != "" && strings.HasPrefix(root, home) {
    t.Skipf("tmpdir %q is under $HOME %q; cannot run non-home test", root, home)
}
```

On Windows CI, `t.TempDir()` returns a path like `C:\Users\runneradmin\AppData\Local\Temp\TestDenylist_NonHome...`. `os.UserHomeDir()` returns `C:\Users\runneradmin`.

`strings.HasPrefix` is case-sensitive. If the OS returns `root` with any path component in different case than `home` (e.g., short-name 8.3 paths like `RUNNER~1`, or symlink-induced canonicalization differences), the prefix check DOES NOT fire and the test proceeds.

When the test then calls `sb.WriteFileAtomic(".bashrc", ...)`:
- `denylistCheck` runs
- `home = EvalSymlinks(os.UserHomeDir())` — resolves to the canonical home path
- `canonAbs` is built from `s.rootPath` (already `EvalSymlinks`-resolved) + `.bashrc`
- If the paths are consistently canonicalized, `filepath.Rel(home, canonAbs)` returns a path starting with `AppData\Local\Temp\...` (no `..` prefix)
- Denylist fires, returns `ErrProtectedSystemFile`
- Test expects `nil` → **FAIL**

Even if the skip guard fires (consistent case), this test is **fragile on Windows**: it relies on `t.TempDir()` being outside `$HOME`, which is not guaranteed on Windows where `%TEMP%` is typically `%USERPROFILE%\AppData\Local\Temp`.

**Alternative failure mode:** On Windows, `t.TempDir()` is ALWAYS under `%USERPROFILE%` in the default GitHub Actions runner configuration. Even if the skip fires, the test effectively cannot run its assertion on Windows CI — it always skips.

**Fix recommendation: TEST-FIX**

The test must not depend on `t.TempDir()` being outside `$HOME`. The fix has two parts:

1. **Make the sandbox root provably non-home-rooted** by redirecting `$HOME` away from the sandbox using `setHomeEnv(t, fakeHome)` where `fakeHome` is a separate temp directory distinct from the sandbox root.

2. **Replace `strings.HasPrefix` with a Windows-safe path prefix check** (case-insensitive) or just eliminate the skip guard entirely once part 1 makes the sandbox provably non-home-rooted.

Fixed pattern:
```go
func TestDenylist_NonHomeRootedUnaffected(t *testing.T) {
    // Use a fake $HOME that is separate from the sandbox root.
    fakeHome := t.TempDir()
    sandboxRoot := t.TempDir()
    setHomeEnv(t, fakeHome)  // directs denylistCheck away from sandboxRoot

    sb, err := files.NewSandbox(sandboxRoot)
    // ... rest of test unchanged ...
}
```

This is safe and correct: `denylistCheck` reads `os.UserHomeDir()` at call time, which will return `fakeHome`. Since `sandboxRoot` is not under `fakeHome`, `filepath.Rel(fakeHome, canonAbs)` will start with `..`, denylist returns `nil`, and `WriteFileAtomic(".bashrc")` succeeds. The security property (non-home sandbox allows `.bashrc`) is verified correctly.

---

### Test 3: `TestWriteFileAtomic_ConcurrentReadNeverPartial` (write_test.go:145)

**What the test does:**
Runs a reader goroutine (`os.ReadFile(targetPath)` in a tight loop) concurrently with a writer loop (200 iterations of `sb.WriteFileAtomic("concurrent.txt", content)`). Asserts that reads never return empty or partial (mixed) content.

**What breaks on Windows CI:**

**The Windows sharing-violation mechanism:**

On POSIX systems, `rename(2)` is atomic and works even when the destination file has open read handles. On Windows, the behavior depends on how the destination file was opened and what rename API is used.

**How `writeAtomicRename` works (sandbox.go:396-414):**
```go
func writeAtomicRename(root *os.Root, tmp, dst string) error {
    const maxAttempts = 3
    attempts := maxAttempts
    if runtime.GOOS != "windows" {
        attempts = 1
    }
    // ...retry loop with 50ms sleep...
    return root.Rename(tmp, dst)
}
```

`root.Rename(tmp, dst)` calls `renameat` in `root_windows.go:376`, which calls `windows.Renameat`.

**`windows.Renameat` implementation (Go stdlib `internal/syscall/windows/at_windows.go:364-436`):**
```go
func Renameat(...) error {
    err := NtOpenFile(&h, SYNCHRONIZE|DELETE, objAttrs, ...,
        FILE_SHARE_DELETE|FILE_SHARE_READ|FILE_SHARE_WRITE, ...)
    // ...
    renameInfoEx := FILE_RENAME_INFORMATION_EX{
        Flags: FILE_RENAME_REPLACE_IF_EXISTS | FILE_RENAME_POSIX_SEMANTICS,
        ...
    }
    err = NtSetInformationFile(h, ..., &renameInfoEx, ..., FileRenameInformationEx)
    if err == nil { return nil }
    // fallback to FILE_RENAME_INFORMATION with ReplaceIfExists
}
```

The key detail: `FILE_RENAME_POSIX_SEMANTICS` is the atomic rename-over-open-file path. It allows the rename to succeed even when the destination file has open handles — BUT only if those open handles were opened with `FILE_SHARE_DELETE`. [CITED: rust-lang/rust#123985, POSIX delete semantics on Windows]

**How `os.ReadFile` opens files on Windows:**

`os.ReadFile` → `os.OpenFile` → `openFileNolog` in `os/file_windows.go:158`:
```go
r, err := syscall.Open(path, flag|syscall.O_CLOEXEC, syscallMode(perm))
```

`syscall.Open` on Windows uses `FILE_SHARE_READ | FILE_SHARE_WRITE` WITHOUT `FILE_SHARE_DELETE` (confirmed: `syscall/syscall_windows.go:395`: `sharemode := uint32(FILE_SHARE_READ | FILE_SHARE_WRITE)`).

**The failure chain:**
1. Reader goroutine calls `os.ReadFile(targetPath)` which opens the destination file WITHOUT `FILE_SHARE_DELETE`
2. Writer calls `root.Rename(tmp, dst)` → `windows.Renameat`
3. `NtSetInformationFile` with `FILE_RENAME_POSIX_SEMANTICS` FAILS because the destination file is held open without `FILE_SHARE_DELETE`
4. Fallback to `FILE_RENAME_INFORMATION` with `ReplaceIfExists: true` ALSO FAILS for the same reason
5. `writeAtomicRename` retries up to 3 times with 50ms delay, but in a tight reader loop the file is almost always open
6. After all retries fail, `WriteFileAtomic` returns an error — `t.Errorf("WriteFileAtomic iteration %d: %v", i, err)` fires → **FAIL**

Note: The test failure is likely `WriteFileAtomic iteration N: files: rename temp: ...` (a non-nil error from `writeAtomicRename`), NOT the partial-read assertion. The retry loop mitigates but does not eliminate the sharing violation because the reader goroutine holds the file open continuously.

**Alternative failure mode:** Even if a rename succeeds with POSIX semantics on Windows Server 2025, the pre-existing file handle held by the reader continues to reference the OLD inode (POSIX behavior). The reader sees the old complete content. So any rename that DOES succeed is actually correct — the test's atomicity assertion would pass even on Windows IF the rename succeeded. The failure is the non-nil error from failed rename attempts, which the writer logs via `t.Errorf`.

**Fix recommendation: TEST-FIX**

The test must not use `os.ReadFile` (which opens without `FILE_SHARE_DELETE`) to race against `WriteFileAtomic`. The fix is to make the reader use a Windows-aware open that includes `FILE_SHARE_DELETE`, so the POSIX semantics path can succeed.

On Windows, the reader should open with `os.OpenFile` plus a custom flag via `syscall.O_CLOEXEC` and the Windows-specific share mode, or — more practically — the test should use `os.OpenFile` → `io.ReadAll` → `Close` where the open uses the correct share flags.

The cleanest approach that does not require importing `syscall` directly into the test is to write a platform-gated helper:

```go
// readFilePlatformSafe reads the file in a way that does not block
// atomic renames on Windows (opens with FILE_SHARE_DELETE).
// On non-Windows it is equivalent to os.ReadFile.
func readFilePlatformSafe(path string) ([]byte, error) {
    // On Windows, os.Open uses FILE_SHARE_READ|FILE_SHARE_WRITE
    // without FILE_SHARE_DELETE, blocking atomic rename-over-open.
    // We must open with os.O_RDONLY + the Windows share-delete flag
    // so NtSetInformationFile with FILE_RENAME_POSIX_SEMANTICS can proceed.
    return os.ReadFile(path) // POSIX: fine
    // Windows-specific variant shown in the build-tagged file below.
}
```

With build-tagged helpers `concurrent_read_unix_test.go` and `concurrent_read_windows_test.go`:

`concurrent_read_windows_test.go` (GOOS=windows):
```go
//go:build windows

package files_test

import (
    "io"
    "os"
    "syscall"
)

func readFilePlatformSafe(path string) ([]byte, error) {
    // Open with FILE_SHARE_DELETE so os.Root.Rename with POSIX semantics
    // can replace the destination file while this handle is open.
    // syscall.FILE_SHARE_DELETE = 0x4
    h, err := syscall.Open(path, syscall.O_RDONLY, 0)
    // NOTE: syscall.Open still doesn't set FILE_SHARE_DELETE.
    // Use syscall.CreateFile directly:
    pathp, _ := syscall.UTF16PtrFromString(path)
    h2, err := syscall.CreateFile(
        pathp,
        syscall.GENERIC_READ,
        syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
        nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
    if err != nil {
        return nil, err
    }
    f := os.NewFile(uintptr(h2), path)
    defer f.Close()
    return io.ReadAll(f)
}
```

`concurrent_read_unix_test.go` (non-Windows):
```go
//go:build !windows

package files_test

import "os"

func readFilePlatformSafe(path string) ([]byte, error) {
    return os.ReadFile(path)
}
```

Then in `TestWriteFileAtomic_ConcurrentReadNeverPartial`, replace:
```go
data, err := os.ReadFile(targetPath)  // current — breaks on Windows
```
with:
```go
data, err := readFilePlatformSafe(targetPath)  // platform-safe
```

This is a TEST-FIX only: the production `WriteFileAtomic` is correct. The existing `writeAtomicRename` retry loop was a reasonable mitigation but the retry cannot overcome a reader that continuously holds the file open without `FILE_SHARE_DELETE`.

**Alternative simpler approach:** Instead of platform-gated file open, the test can accept that on Windows the concurrent-read atomicity invariant is verified differently — the reader goroutine can be changed to not hold the file open continuously (e.g., add a brief sleep between reads). However, this weakens the test. The build-tagged helper approach is more principled.

---

## Standard Stack

This phase is a pure Go test-fix. No new dependencies are added.

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os` (stdlib) | go1.26.3 | File operations | Already used |
| `syscall` (stdlib) | go1.26.3 | Windows CreateFile with custom share flags | For the Windows-specific concurrent-read helper |
| `runtime` (stdlib) | go1.26.3 | GOOS detection | Already used in `setHomeEnv` |
| `testing` (stdlib) | go1.26.3 | Test infrastructure | Already used |

## Package Legitimacy Audit

No external packages are installed in this phase. All code changes use Go stdlib only.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
Test goroutine (writer)              Test goroutine (reader)
     |                                     |
     v                                     v
WriteFileAtomic("concurrent.txt")    readFilePlatformSafe(path)
     |                                     |
     v                                     v
writeAtomicRename(root, tmp, dst)    [POSIX] os.ReadFile → no FILE_SHARE_DELETE
     |                                     |
     v                                     v
windows.Renameat → NtSetInfoFile     [Windows] syscall.CreateFile with
   with FILE_RENAME_POSIX_SEMANTICS      FILE_SHARE_DELETE
                                          |
     <---- rename succeeds if reader -----+
           has FILE_SHARE_DELETE
```

### Recommended Project Structure

No new directories. Changes are within `internal/files/`:
```
internal/files/
├── write_test.go             # Tests 1+2: fix setHomeEnv usage + skip guard
├── concurrent_read_unix_test.go    # NEW: os.ReadFile wrapper for non-Windows
└── concurrent_read_windows_test.go # NEW: syscall.CreateFile wrapper for Windows
```

OR (simpler alternative for Test 3, if build tags are undesirable):

```
internal/files/
├── write_test.go             # All three fixes inline; Test 3 uses a platform-gated helper
└── platform_test.go          # NEW: readFilePlatformSafe build-tagged helper
```

### Pattern 1: Windows-Safe HOME Isolation in Tests
**What:** Use `setHomeEnv(t, fakeHome)` to redirect `os.UserHomeDir()` away from the actual test sandbox, ensuring the sandbox is provably non-home-rooted.
**When to use:** Any test that creates a sandbox and needs to control whether the denylist fires.
**Example:**
```go
// Source: write_test.go:466-473 (existing setHomeEnv helper)
fakeHome := t.TempDir()
sandboxRoot := t.TempDir()
setHomeEnv(t, fakeHome)  // HOME=fakeHome; USERPROFILE=fakeHome on Windows
sb, _ := files.NewSandbox(sandboxRoot)
// denylistCheck now considers sandboxRoot to be outside fakeHome → denylist won't fire
```

### Pattern 2: Windows FILE_SHARE_DELETE for Test Readers
**What:** Open files for reading with `FILE_SHARE_DELETE` so `writeAtomicRename` with POSIX semantics can succeed.
**When to use:** Any test that reads a file concurrently with `WriteFileAtomic`.
**Example:**
```go
// Source: windows.Renameat in internal/syscall/windows/at_windows.go:375
// The rename opens the source with FILE_SHARE_DELETE|READ|WRITE,
// and uses FILE_RENAME_POSIX_SEMANTICS. Readers must also hold FILE_SHARE_DELETE
// for the rename to succeed while the read handle is open.
```

### Anti-Patterns to Avoid
- **Using `strings.HasPrefix` for Windows path prefix checks:** Case-sensitive; breaks on Windows where `USERPROFILE` and `TEMP` paths may differ in capitalization due to 8.3 name resolution or symlinks. Use `filepath.Rel` + `strings.HasPrefix(rel, "..")` (already done in `denylistCheck`).
- **Relying on `t.TempDir()` being outside `$HOME` on Windows:** Always false on GitHub Actions Windows runner; `%TEMP%` is under `%USERPROFILE%`.
- **Using `os.ReadFile` as a concurrent reader against `WriteFileAtomic` on Windows:** Opens without `FILE_SHARE_DELETE`; blocks the POSIX rename path.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Windows FILE_SHARE_DELETE open | Custom WinAPI wrapper | `syscall.CreateFile` (stdlib) | Already in Go stdlib; direct WinAPI call is the right tool |
| HOME isolation in tests | Custom env cleanup logic | `t.Setenv` (stdlib) | Auto-restored after test; already used by `setHomeEnv` |
| Cross-platform path prefix | `strings.HasPrefix` | `filepath.Rel` + `..` check | Already correct in `denylistCheck`; tests should match |

---

## Common Pitfalls

### Pitfall 1: Windows `%TEMP%` is Under `%USERPROFILE%`
**What goes wrong:** Tests that assume `t.TempDir()` produces a path outside `$HOME` will fail or be skipped on Windows because `%TEMP%` defaults to `%USERPROFILE%\AppData\Local\Temp`.
**Why it happens:** POSIX systems typically use `/tmp` which is never under `$HOME`. Windows has a per-user temp directory by default.
**How to avoid:** Use `setHomeEnv(t, fakePath)` to control what `os.UserHomeDir()` returns during the test rather than relying on the physical temp location.
**Warning signs:** Test passes on macOS (uses `/var/folders/...`) and Linux (uses `/tmp/...`) but fails or skips on Windows.

### Pitfall 2: `os.ReadFile` Blocks Windows POSIX Rename
**What goes wrong:** A goroutine using `os.ReadFile` or `os.Open` (without `FILE_SHARE_DELETE`) while another goroutine calls `WriteFileAtomic` → `root.Rename`. The rename fails with sharing violation.
**Why it happens:** `syscall.Open` on Windows (used by `os.OpenFile`) sets `FILE_SHARE_READ|FILE_SHARE_WRITE` only. `windows.Renameat` requests `FILE_RENAME_POSIX_SEMANTICS` but it requires all extant handles on the destination to have been opened with `FILE_SHARE_DELETE`.
**How to avoid:** In Windows tests that race a reader with `WriteFileAtomic`, use `syscall.CreateFile` with `FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE`.
**Warning signs:** `TestWriteFileAtomic_ConcurrentReadNeverPartial` fails on Windows with `files: rename temp: ...` (non-nil error from `writeAtomicRename`), NOT the partial-read assertion.

### Pitfall 3: Case-Sensitive Path Prefix on Windows
**What goes wrong:** `strings.HasPrefix(path, home)` returns false on Windows when path components differ in case (e.g., `RUNNER~1` vs `runneradmin`).
**Why it happens:** Windows filesystem is case-insensitive but `strings.HasPrefix` is case-sensitive. Path canonicalization via `filepath.EvalSymlinks` helps but is not guaranteed to produce uniform case.
**How to avoid:** Use `filepath.Rel(home, path)` and check if the result starts with `..`. The `denylistCheck` in `sandbox.go` already does this correctly. Tests should use the same pattern or use `setHomeEnv` to control the environment.
**Warning signs:** Skip guard using `strings.HasPrefix(root, home)` doesn't fire on Windows when it should.

### Pitfall 4: Build Tags Required for Windows-Only Test Helpers
**What goes wrong:** If `concurrent_read_windows_test.go` uses `syscall.CreateFile` without a `//go:build windows` tag, `go vet` fails on non-Windows platforms (`syscall.CreateFile` is undefined on non-Windows).
**Why it happens:** Go's `syscall` package is platform-specific.
**How to avoid:** Always pair Windows-only test files with `//go:build windows` and a `//go:build !windows` companion file. Both must define the same exported function name.
**Warning signs:** `GOOS=linux go vet ./internal/files/...` fails after adding the Windows test helper.

### Pitfall 5: The Retry Loop in `writeAtomicRename` Is Not Sufficient Alone
**What goes wrong:** The existing 3-attempt retry with 50ms delay in `writeAtomicRename` does not prevent `TestWriteFileAtomic_ConcurrentReadNeverPartial` from failing.
**Why it happens:** The reader goroutine is in a tight loop that holds the file open for microseconds between calls. With 200 writer iterations and 3 retries (150ms window per write), there is a high probability the reader re-opens the file during every retry window.
**How to avoid:** Fix the reader in the test (not the production code) to use `FILE_SHARE_DELETE`. The retry loop is still valuable for production use (e.g., antivirus scanners on Windows), but it cannot overcome a tight-loop reader without proper share flags.

---

## Code Examples

### Example 1: Existing `setHomeEnv` Helper (already in write_test.go:466)
```go
// Source: internal/files/write_test.go:466-473
func setHomeEnv(t *testing.T, dir string) {
    t.Helper()
    t.Setenv("HOME", dir)
    if runtime.GOOS == "windows" {
        t.Setenv("USERPROFILE", dir)
    }
}
```
Use this helper in all three failing tests.

### Example 2: Fixed `TestDenylist_NonHomeRootedUnaffected`
```go
func TestDenylist_NonHomeRootedUnaffected(t *testing.T) {
    // Create a fake $HOME that is distinct from the sandbox root.
    // This makes the test portable: on Windows, t.TempDir() is under
    // %USERPROFILE%, but by setting HOME/USERPROFILE to a different dir,
    // denylistCheck will not consider the sandbox home-rooted.
    fakeHome := t.TempDir()
    sandboxRoot := t.TempDir()
    setHomeEnv(t, fakeHome)

    sb, err := files.NewSandbox(sandboxRoot)
    if err != nil {
        t.Fatalf("NewSandbox: %v", err)
    }

    // Writing ".bashrc" in a non-home sandbox must succeed.
    if err := sb.WriteFileAtomic(".bashrc", []byte("# rc in non-home sandbox")); err != nil {
        t.Errorf("WriteFileAtomic(.bashrc) in non-home sandbox: %v (want nil)", err)
    }
}
```

### Example 3: Windows-Safe Concurrent Reader Helper
```go
// concurrent_read_windows_test.go
//go:build windows

package files_test

import (
    "io"
    "os"
    "syscall"
)

// readFilePlatformSafe reads path with FILE_SHARE_DELETE so WriteFileAtomic's
// POSIX-semantics rename (via windows.Renameat + FILE_RENAME_POSIX_SEMANTICS)
// can replace the file while this read handle is open.
func readFilePlatformSafe(path string) ([]byte, error) {
    pathp, err := syscall.UTF16PtrFromString(path)
    if err != nil {
        return nil, err
    }
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
    f := os.NewFile(uintptr(h), path)
    defer f.Close()
    return io.ReadAll(f)
}
```

```go
// concurrent_read_unix_test.go
//go:build !windows

package files_test

import "os"

// readFilePlatformSafe reads the file. On non-Windows, os.ReadFile is sufficient
// because rename(2) is atomic and does not require FILE_SHARE_DELETE semantics.
func readFilePlatformSafe(path string) ([]byte, error) {
    return os.ReadFile(path)
}
```

### Example 4: Fixed `TestHandlerUpload_FilenameSanitized` (sketch)
```go
func TestHandlerUpload_FilenameSanitized(t *testing.T) {
    // Set HOME to a fake dir so the sandbox (under t.TempDir()) is not
    // considered home-rooted by denylistCheck. On Windows, t.TempDir()
    // lives under %USERPROFILE%, so without this redirect the upload of
    // ".bashrc" would trigger the denylist and return 403.
    fakeHome := t.TempDir()
    setHomeEnv(t, fakeHome)

    h, root := newHandler(t)
    // ... rest of test unchanged ...
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `os.Rename` on Windows (MoveFileW, no overwrite) | `windows.Renameat` with `FILE_RENAME_POSIX_SEMANTICS` via `NtSetInformationFile` | Go 1.24 (os.Root introduction) | Atomic overwrite in same filesystem; requires reader has FILE_SHARE_DELETE |
| `syscall.Open` without `FILE_SHARE_DELETE` (Windows default) | Still the default for `os.OpenFile`; `windows.Openat` (used by os.Root) includes FILE_SHARE_DELETE | Go 1.24 | Tests using `os.ReadFile` still open without FILE_SHARE_DELETE |
| `windows-latest` = Windows Server 2022 | `windows-latest` = Windows Server 2025 | September 2025 GA | POSIX rename semantics fully supported; FILE_SHARE_DELETE requirement remains |

**Deprecated/outdated:**
- Using `strings.HasPrefix` for Windows path containment checks: deprecated in favor of `filepath.Rel` + `..` prefix detection.
- `filepath.HasPrefix` (deprecated since Go 1.0): never use; use `filepath.Rel` instead.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — invoked via `go test` |
| Quick run command | `go test -race -short -run "TestHandlerUpload_FilenameSanitized\|TestDenylist_NonHomeRootedUnaffected\|TestWriteFileAtomic_ConcurrentReadNeverPartial" ./internal/files/` |
| Full suite command | `go test -race -short ./internal/files/` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FIX-02 | `TestHandlerUpload_FilenameSanitized` passes on Windows | Go unit | `go test -race -short -run TestHandlerUpload_FilenameSanitized ./internal/files/` | Yes — `internal/files/handler_test.go:695` |
| FIX-02 | `TestDenylist_NonHomeRootedUnaffected` passes on Windows | Go unit | `go test -race -short -run TestDenylist_NonHomeRootedUnaffected ./internal/files/` | Yes — `internal/files/write_test.go:581` |
| FIX-02 | `TestWriteFileAtomic_ConcurrentReadNeverPartial` passes on Windows | Go unit + race | `go test -race -short -run TestWriteFileAtomic_ConcurrentReadNeverPartial ./internal/files/` | Yes — `internal/files/write_test.go:145` |
| FIX-02 | Full `internal/files` suite passes with race detector | Go unit + race | `go test -race -short ./internal/files/` | Yes — all existing files |

### Sampling Rate
- **Per commit:** `go test -race -short ./internal/files/`
- **Phase gate:** Full suite green (`go test -race -short ./internal/...`) before push to CI
- **CI gate:** Windows matrix job (`build (agenthub, windows/amd64, windows-latest)`) must be green

### Wave 0 Gaps
- [ ] `internal/files/concurrent_read_windows_test.go` — covers FIX-02 (new file, Windows build-tagged helper)
- [ ] `internal/files/concurrent_read_unix_test.go` — covers FIX-02 (new file, non-Windows companion)

*(Existing test files are modified, not moved or renamed. The two new helpers are the only new files.)*

### Local Pre-CI Verification

Since the developer is on macOS and cannot reproduce Windows failures locally:

```bash
# Cross-compile check: ensure Windows-specific test files compile
GOOS=windows GOARCH=amd64 go vet ./internal/files/

# Cross-compile check: ensure all test files compile for Windows
GOOS=windows GOARCH=amd64 go test -c ./internal/files/ -o /dev/null

# Run full files suite locally (macOS) to ensure no regressions
go test -race -short ./internal/files/

# Run race detector (macOS) — covers Linux/macOS atomicity
go test -race -count=3 -run TestWriteFileAtomic_ConcurrentReadNeverPartial ./internal/files/
```

These checks provide confidence before burning a CI run but do NOT prove Windows correctness. The Windows matrix job is the only ground truth. Push to CI after local checks pass.

---

## Security Domain

Security enforcement is enabled (not explicitly disabled in config).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | yes | `denylistCheck` in sandbox.go — this phase must not weaken it |
| V5 Input Validation | yes | `validateRelativePath` + `validateAndClean` — unchanged by this phase |
| V6 Cryptography | no | — |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Denylist bypass via home-root confusion | Tampering | `denylistCheck` uses `filepath.Rel` + `..` check; fixes must not weaken this |
| Filename traversal (`../../.bashrc`) | Tampering | `filepath.Base` (primary) + `validateAndClean` (secondary); unchanged in this phase |
| Concurrent write with partial-read | Tampering / DoS | Atomic rename via `writeAtomicRename`; test fix does not change production logic |

**Critical security constraint:** The test fixes for Tests 1 and 2 use `setHomeEnv(t, fakeHome)` to REDIRECT `$HOME`, not to disable the denylist. The denylist itself is exercised by `TestDenylist_HomeRooted` and `TestDenylist_CaseVariation` (which correctly set HOME to the sandbox root). Phase 145 must NOT add any changes that cause the denylist to fire less aggressively in production.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.26.x | `os.Root`, `windows.Renameat` | On CI and local | 1.26.3 (go.mod) | — |
| Windows Server 2025 runner | Final verification | CI only | `windows-latest` | No local fallback; use `GOOS=windows go vet` for early cross-compile check |
| `syscall.CreateFile` | Windows-specific test helper | Go stdlib | always | — |

**Missing dependencies with no fallback:**
- None. The Windows runner is accessible only via CI push.

**Missing dependencies with fallback:**
- Windows-only test execution: use `GOOS=windows GOARCH=amd64 go vet/go test -c` for compile-time check before pushing.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `windows-latest` on GitHub Actions is Windows Server 2025 as of late 2025 | Standard Stack / Pitfall 2 | Low risk — Windows Server 2022 also has FILE_RENAME_POSIX_SEMANTICS and the same FILE_SHARE_DELETE requirement |
| A2 | `%TEMP%` on the GitHub Actions Windows runner is under `%USERPROFILE%` | Test 1 and Test 2 root cause analysis | If `%TEMP%` is on a separate drive (e.g., D:\), `t.TempDir()` would be outside `$HOME` and the skip guard might fire correctly. But the `setHomeEnv` fix is still the right pattern regardless. |
| A3 | `FILE_RENAME_POSIX_SEMANTICS` requires reader handles to have `FILE_SHARE_DELETE` | Test 3 root cause | HIGH: If POSIX semantics work without FILE_SHARE_DELETE on Windows Server 2025, the failure would be different (partial read, not rename error). The rust-lang/rust#123985 issue and related discussions confirm this requirement. [CITED: rust-lang/rust#123985] |

---

## Open Questions (RESOLVED)

1. **Is the existing `writeAtomicRename` retry loop still needed after the test fix?**
   - What we know: The retry was added as a mitigation for sharing violations from production readers (antivirus, indexers). The test fix makes the TEST reader safe; production readers are outside our control.
   - RESOLVED: Keep the retry loop unchanged — see plan 145-02 threat model T-145-02. It does not affect correctness; it only improves production resilience, and the test fix is independent of whether we keep or remove it.

2. **Should Tests 1 and 2 use `setHomeEnv` or instead find a genuinely non-home temp location?**
   - What we know: `setHomeEnv` modifies the environment for the test, which is what the existing denylist tests do. Using a fake home is the established pattern.
   - RESOLVED: Use `setHomeEnv` — plan 145-01 does exactly this. `t.Setenv` (used by `setHomeEnv`) restores the env after the test, so there is no cross-test interference.

3. **Should `newHandler` be replaced by a `newHandlerOutsideHome` helper, or should the fix be inline?**
   - What we know: `newHandler` is used by 35+ tests; changing it globally would affect all of them.
   - RESOLVED: Inline redirect — plan 145-01 Task 1 redirects `$HOME` via `setHomeEnv` inline inside `TestHandlerUpload_FilenameSanitized` before `newHandler(t)`. This supersedes the earlier `newHandlerOutsideHome(t)` helper idea: the inline approach is equivalent, requires fewer changes, and avoids adding a second handler helper that only one test would ever use.

---

## Sources

### Primary (HIGH confidence)
- Go stdlib `$GOROOT/src/internal/syscall/windows/at_windows.go` — verified `windows.Renameat` uses `FILE_SHARE_DELETE|READ|WRITE` and `FILE_RENAME_POSIX_SEMANTICS`; `windows.Openat` uses `FILE_SHARE_READ|WRITE|DELETE`
- Go stdlib `$GOROOT/src/syscall/syscall_windows.go:395` — verified `syscall.Open` (used by `os.OpenFile`) sets only `FILE_SHARE_READ|FILE_SHARE_WRITE` (no DELETE)
- Go stdlib `$GOROOT/src/os/path_windows.go:19` — verified `IsPathSeparator` recognizes both `/` and `\` on Windows; `filepath.Base("../../.bashrc")` returns `.bashrc` correctly
- Go stdlib `$GOROOT/src/os/file_windows.go:158` — verified `openFileNolog` calls `syscall.Open` (not `windows.Openat`)
- Go stdlib `$GOROOT/src/os/root_windows.go:376` — verified `renameat` calls `windows.Renameat`
- Go stdlib `$GOROOT/src/os/file.go:609` — verified `os.UserHomeDir()` returns `%USERPROFILE%` on Windows
- `internal/files/sandbox.go` — verified `denylistCheck` uses `filepath.Rel` + `..` check; case-folds via `strings.ToLower`; `EvalSymlinks` on both home and target
- `internal/files/write_test.go:466-473` — verified `setHomeEnv` helper exists and sets both `HOME` and `USERPROFILE`

### Secondary (MEDIUM confidence)
- [rust-lang/rust#123985](https://github.com/rust-lang/rust/issues/123985) — confirms `FILE_RENAME_POSIX_SEMANTICS` requires reader to have `FILE_SHARE_DELETE`
- [GitHub Actions runner images](https://github.com/actions/runner-images/issues/12677) — `windows-latest` migrated to Windows Server 2025 by September 2025
- [golang/go#32088](https://github.com/golang/go/issues/32088) — background on Go `syscall.Open` not setting `FILE_SHARE_DELETE` by default

### Tertiary (LOW confidence)
- Search results confirming `%TEMP% = %USERPROFILE%\AppData\Local\Temp` on GitHub Actions Windows runner — not directly verified against a live runner log

---

## Metadata

**Confidence breakdown:**
- Per-test root cause analysis: HIGH — verified from Go stdlib source code on the installed Go 1.26.4
- Windows OS semantics (FILE_SHARE_DELETE requirement): HIGH — confirmed by Go stdlib source + rust issue citation
- CI runner environment (`%TEMP%` under `%USERPROFILE%`): MEDIUM — confirmed by general Windows knowledge and search results, not a live runner log

**Research date:** 2026-06-22
**Valid until:** 2026-12-22 (stable OS semantics; Go stdlib behavior unlikely to change)
