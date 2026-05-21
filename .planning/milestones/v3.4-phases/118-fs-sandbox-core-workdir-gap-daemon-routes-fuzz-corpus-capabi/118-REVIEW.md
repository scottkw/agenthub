---
phase: 118
status: findings
critical_count: 0
warning_count: 4
info_count: 7
---

# Phase 118 Code Review — FS Sandbox Core + WorkDir Gap + Daemon Routes + Fuzz Corpus + Capability Bit

**Reviewed:** 2026-05-20
**Depth:** standard
**Files Reviewed:** 19
**Status:** findings

## Summary

Phase 118 lays the security-critical groundwork for the v3.4 file browser. The
implementation is generally careful — TOCTOU-safe sandbox built on Go 1.24+
`os.OpenRoot`, whole-token `HasPerm` semantics, explicit 0-byte short-circuit
ahead of `http.ServeContent`, defaults-merge migration for the new `filesRead`
setting, capability-bit gating with a separate `requireFilesRead` wrapper, and
a 40+ payload fuzz corpus.

No Critical findings (the load-bearing security boundaries — `HasPerm`
whole-token semantics, `os.OpenRoot` for traversal, owner/viewer perms split,
defaults-merge for `filesRead`, separation of `requireFilesRead` from
`requireCapability`, 0-byte Range mitigation, 5 MiB cap — are all implemented
correctly and tested). Findings are concentrated around a directory-listing
truncation false-positive, a few defense-in-depth gaps in the path validator,
and several maintainability concerns.

Files reviewed: `internal/files/{sandbox,handler,types,mime}.go` and tests,
`internal/capability/capability.go` and tests, `internal/webserver/capability_mw.go`
and `capability_test.go`, `internal/daemon/{engine,api,client,types,
engine_migration_test,plugin_settings}.go`, related tests, and the v3.2 fixture.

## Warnings

### WR-01: `Truncated` flag false-positives at exactly 10,000 entries

**File:** `internal/files/handler.go:135-143`

**Issue:** `dir.ReadDir(maxListEntries)` returns up to 10,000 entries. The
handler computes `Truncated: len(entries) == maxListEntries`. When a directory
contains exactly 10,000 entries (no more), `Truncated` is set to `true` even
though no further entries exist. The frontend will surface a misleading "more
entries available" message and may attempt a paginated read that finds nothing.

**Fix:** Read one extra entry to disambiguate, then trim:

```go
const probe = maxListEntries + 1
entries, err := dir.ReadDir(probe)
if err != nil && !errors.Is(err, io.EOF) {
    http.Error(w, "read directory failed", http.StatusInternalServerError)
    return
}
truncated := len(entries) > maxListEntries
if truncated {
    entries = entries[:maxListEntries]
}
result := FileListResponse{
    Entries:   make([]FileEntry, 0, len(entries)),
    Truncated: truncated,
}
```

Existing test `TestHandler_List_TruncatedAt10000` creates 10001 files and
happens to pass with the current implementation, but a 10000-file directory
will fail the contract.

### WR-02: `validateRelativePath` drive-letter check produces wrong error label for ADS-shaped paths

**File:** `internal/files/sandbox.go:156-159`

**Issue:** The drive-letter check fires when `len(p) >= 2 && p[1] == ':'`.
This is correct for `C:\foo`, but it also fires for legitimate-looking
filenames like `a:foo` (a one-character filename with a colon-suffix). Such
inputs are correctly rejected, but the error message is `"files: drive letter
rejected"` which is misleading — `a:foo` is not a drive letter. The dedicated
ADS check at line 166 (`strings.ContainsRune(p, ':')`) would never get a
chance to label this correctly because the drive-letter check fires first.

Because the colon-anywhere ban (line 166) already rejects all `:` payloads,
the drive-letter check at lines 156-159 is functionally redundant. It only
adds diagnostic value when the message is accurate.

**Fix:** Either tighten the drive-letter regex to require a single ASCII
letter followed by `:` (e.g., `unicode.IsLetter(rune(p[0])) && p[1] == ':'`),
or remove the special case entirely and let the colon-anywhere check produce
a uniform "colon rejected" message. Tests in `sandbox_test.go` only check
that rejection happens, not the message, so either change is test-safe.

### WR-03: Symlink-escape test asserts via wrong path shape — coverage is shallower than it appears

**File:** `internal/files/sandbox_test.go:210-229`

**Issue:** `TestSandbox_SymlinkEscapeBlocked` creates `root/escape →
outside_dir` and then calls `sb.Open("escape/secret")`. The test passes if
the open fails. But `os.Root.Open` distinguishes failure modes — it rejects
the symlink because it points *out of the root*. If the implementation were
silently broken (e.g., calling `os.Open(filepath.Join(s.rootPath, rel))`
instead of `root.Open`), the call could still fail with "no such file" because
`secret` doesn't actually exist on `root/escape/`. The test would pass for the
wrong reason.

**Fix:** Create the target file too (`outside/secret`) so the only path to
success is escape, then assert error:

```go
if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("leaked"), 0o644); err != nil {
    t.Fatal(err)
}
// ... existing symlink creation ...
f, err := sb.Open("escape/secret")
if err == nil {
    f.Close()
    t.Errorf("Open through escaping symlink succeeded; want error")
}
```

The fixture already creates `outside/secret` (line 217), so this is largely
defensive — but the assertion as written would still pass against a broken
implementation that returns ENOENT for the right reason (file not found via
the escape route). Consider adding a positive control that proves the
direct-path `outside/secret` IS readable to confirm the file exists.

### WR-04: Fuzz function contains tautological error check that triggers no real assertion

**File:** `internal/files/sandbox_test.go:330-332`

**Issue:** Inside `FuzzSandboxPath`:

```go
if !errors.Is(err, err) { // tautology — prevents linter complaining about unused err
    t.Fatalf("nil error wrapper")
}
```

`errors.Is(err, err)` is always true for any non-nil `err`. The comment
acknowledges it's a tautology. This is dead code that does nothing — if the
goal was to use `err` to suppress a linter warning, that's a smell; if the
goal was to verify the error is wrapped, the check is wrong. Either way, this
costs reader trust ("why is this here? is it load-bearing?") for no value.

**Fix:** Remove the tautology. The simple branch is:

```go
fp, err := sb.Open(rawPath)
if err != nil {
    // Rejection is fine; the fuzzer hunts panics, not over-rejection.
    return
}
defer fp.Close()
if _, err := fp.Stat(); err != nil {
    t.Errorf("accepted path %q stat failed: %v", rawPath, err)
}
```

`err` is naturally consumed by the `if err != nil` predicate — no
linter-appeasement gymnastics needed.

## Info

### IN-01: `HasPerm` does not handle whitespace around comma-separated tokens

**File:** `internal/capability/capability.go:44-54`

**Issue:** `HasPerm("read, files.read", "files.read")` returns false because
the second token is `" files.read"` (with leading space). Today only
`Sign`/owner-issuance produces these strings server-side, so whitespace never
enters the wire — but the function is exported and a future caller could pass
human-edited perms (e.g., from a config file) and silently miss the bit.

**Fix:** Trim each token before comparison:

```go
for _, t := range strings.Split(perms, ",") {
    if strings.TrimSpace(t) == perm {
        return true
    }
}
```

Add a test case `{"with-whitespace", "read, files.read", "files.read", true}`
to lock in.

### IN-02: `NewHandler` does not check `resolve` for nil at construction

**File:** `internal/files/handler.go:63-65`

**Issue:** The doc comment says "resolve must NOT be nil — a nil resolver is
a programming error caught by Go's nil-call panic on the first request."
Deferring nil panic to first request rather than constructor produces a
worse error path: production deployments crash on first user traffic instead
of failing fast at startup.

**Fix:** Add a nil check in the constructor:

```go
func NewHandler(resolve sandboxResolver) *Handler {
    if resolve == nil {
        panic("files: NewHandler requires non-nil resolver")
    }
    return &Handler{resolve: resolve}
}
```

### IN-03: `Stat` performs MIME sniff that consumes the file but no caller consumes the buffer afterwards

**File:** `internal/files/handler.go:198-205`

**Issue:** `Stat` calls `sniffMIME(f)` which advances the file offset, then
defensively seeks back to start. The Seek-to-start is documented as
"defense in depth" since `Stat` does not stream the body. The seek is a
syscall with no functional purpose in this code path — only defensive in
case a future caller adds streaming after the sniff. Reasonable, but worth
noting: if the file is on a network filesystem with slow seeks, this is
wasted I/O. Alternatively, drop the seek and update the comment to reflect
that the file is closed via `defer` immediately after.

**Fix (optional):** Remove the seek and rely on `defer f.Close()`:

```go
if mime == "" && !fi.IsDir() {
    mime = sniffMIME(f)
    // f is closed via defer; no further reads, so no need to rewind.
}
```

### IN-04: `Read` does not set `Content-Length` on 0-byte short-circuit path

**File:** `internal/files/handler.go:272-276`

**Issue:** The 0-byte branch sets `Last-Modified` and writes `WriteHeader(200)`
without explicitly setting `Content-Length: 0`. Go's stdlib will infer
Content-Length=0 because no bytes are written, but this only works if the
ResponseWriter is not in chunked mode. Test
`TestAPI_FilesRead_ZeroByteReturns200` allows either unset OR "0". Setting
it explicitly removes ambiguity for downstream caches and HEAD-mode clients:

```go
if fi.Size() == 0 {
    w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
    w.Header().Set("Content-Length", "0")
    w.WriteHeader(http.StatusOK)
    return
}
```

### IN-05: `extensionMIME` has Windows-specific extensions in the source-code branch with no rendering check

**File:** `internal/files/mime.go:32-33`

**Issue:** Extensions `.bat` and `.cmd` route to `text/plain; charset=utf-8`,
which is correct for the file-browser preview UX. But these can contain
arbitrary command-line content; if a future GUI ever passes the response body
through any kind of macro expansion / shell-format interpolation, this becomes
a vector. Today the frontend is documented as untrusted-source-code-only —
log a comment that these are safe BECAUSE the frontend treats them as plain
text, not because the content is benign.

**Fix:** Inline comment above the source-code branch warning that any future
template/interpolation step must NOT operate on these bodies. Cost: zero
runtime, +1 line of doc. No code change needed.

### IN-06: `daemonSettings` `FilesRead` defaults-merge happens at load time but not on first-run zero settings

**File:** `internal/daemon/engine.go:156-160`

**Issue:** `loadSettingsFromDisk` populates `FilesRead: &tr` before
`Unmarshal`. If `settings.json` doesn't exist (first run), the early `return`
on line 147 means `e.filesRead` stays `nil`. `filesReadEnabled()` returns
true for nil, so behavior is correct — but tests that construct a
`SessionEngine` literal (e.g., `engine_migration_test.go:45`) without calling
`loadSettingsFromDisk` will see `nil` and might mis-attribute behavior.

This is observable but not buggy. Worth adding a `nil` check in
`NewSessionEngine` for symmetry:

```go
e := &SessionEngine{
    ...
    sessionStatuses: make(map[string]status.SessionStatus),
}
defaultTrue := true
e.filesRead = &defaultTrue   // pre-load default; loadSettingsFromDisk may overwrite
e.loadSettingsFromDisk(cfgDir)
```

Then `e.filesRead` is never nil in normal operation. The nil-fallthrough
in `filesReadEnabled` would become a defense-in-depth, not the primary path.

### IN-07: `requireFilesRead` includes pre-emptive "claims not in context" branch that is documented as unreachable

**File:** `internal/webserver/capability_mw.go:105-112`

**Issue:** The branch
```go
claims, ok := capability.ClaimsFromContext(r.Context())
if !ok {
    http.Error(w, "files.read capability required", http.StatusForbidden)
    return
}
```
is documented as unreachable ("requireCapability always attaches claims on
the success path"). Dead code paths erode trust in the active paths — when
this branch fires (it can't today, but a future refactor might), the
operator sees the same body as the legitimate 403. Distinguishing them
would help debug a future regression:

**Fix (optional):** Return a different body, e.g.,
`"capability context missing"`, for the unreachable branch. Or upgrade to
an explicit `panic` since the documented contract is that the branch is
impossible — a panic surfaces the contract violation rather than silently
denying access.

---

## Notes on items confirmed correct

The following Phase-118-specific risks were checked and found correctly
implemented (no finding required):

- **HasPerm whole-token semantics:** `strings.Split(perms, ",")` with `==`
  comparison — Pitfall 4 mitigated. `TestHasPerm` exercises the
  `"no-files.read"` substring-false-positive case explicitly, and
  `TestHasPerm_NoStringsContains` source-inspects to forbid the regression.
- **0-byte Range short-circuit:** Explicit `fi.Size() == 0` check ahead of
  `http.ServeContent` — golang/go#54794 / FS-07 mitigated.
  `TestHandler_ZeroByteRead` covers both the no-Range and `Range: bytes=0-`
  case.
- **5 MiB preview cap with strict >:** Boundary at exactly 5 MiB is allowed
  per `TestHandler_Read_BoundaryAt5MiB`.
- **darwin-only `._` resource-fork filter:** Guarded by
  `runtime.GOOS == "darwin"`. Test skipped on non-darwin.
- **`sessionWorkDirs` map concurrency:** Initialized in `NewSessionEngine`;
  all accesses under `e.mu` (write in `CreateSession` line 333, delete in
  `KillSession` line 466, read in `GetSessionWorkDir` line 494).
- **`requireFilesRead` body contains "files.read":** Confirmed at both 403
  branches in capability_mw.go (lines 110, 114). Source-inspection test
  `TestRequireCapability_UnchangedByPhase118` pins that `requireCapability`
  does NOT mention `"files.read"` (Pitfall 4 separation invariant).
- **Owner gets files.read only when FilesRead nil-or-true; viewer never:**
  `issueCapabilitiesForSession` line 995-1000 — viewer perms hardcoded to
  `"read"`. Four api_test cases lock the matrix:
  `OwnerHasFilesRead_WhenSettingNil`, `ViewerNoFilesRead`,
  `OwnerNoFilesReadWhenDisabled`, `OwnerHasFilesReadWhenExplicitTrue`.
- **TOCTOU safety:** `NewSandbox` runs `EvalSymlinks` once at construction;
  per-request `os.OpenRoot` is atomic at the syscall level. The daemon path
  (api.go line 76) constructs a fresh Sandbox per request so symlink
  retargeting cannot persist across requests.
- **Defaults-merge for v3.2 → v3.4 upgrade:** `filesRead` pre-populated to
  `&true` before `json.Unmarshal`; explicit `false` is preserved
  (`TestSettingsMigration_FilesReadExplicitFalse`).
- **Fuzz corpus:** 40+ seeded payloads across traversal, encoded, Windows
  device names, ADS, null bytes, Unicode, long paths, mixed separators —
  matches the documented PITFALLS.md corpus.

## REVIEW COMPLETE
