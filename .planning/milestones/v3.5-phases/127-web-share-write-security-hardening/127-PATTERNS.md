# Phase 127: Web-Share Write Security Hardening - Pattern Map

**Mapped:** 2026-06-14
**Files analyzed:** 5 gap-fix targets (2 production-code, 2 test, 1 e2e) + 1 net-new doc
**Analogs found:** 6 / 6 (every gap has a concrete existing analog)

> This is an AUDIT-and-HARDEN phase. The write surface is already built and tested (123-125).
> Only the gap-fix files below should be created/modified. Do NOT touch `WriteFileAtomic`,
> `Rename`, `Mkdir`, `Delete`, `validateRelativePath`, the `write.go` handlers, or
> `requireFilesWrite`/`originAllowedForWrite` — they are correct and tested (PITFALLS §1).

## Net-New CODE Changes (the only two)

Both land in **one function**: `denylistCheck` in `internal/files/sandbox.go:96-148`.
1. **macOS daemon-config-dir hole** — add `os.UserConfigDir()`-derived prefix to the protected set.
2. **Case-insensitivity** — case-fold the base name and dir-prefix comparison.

Everything else is additive tests, one e2e cell, and a markdown doc.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/files/sandbox.go` (`denylistCheck` only) | utility (security boundary) | transform/validate | itself — CR-02 EvalSymlinks block already in `denylistCheck` (sandbox.go:114-146); `daemonConfigDir` (engine.go:62) | exact (same fn, extend in place) |
| `internal/files/write_test.go` (extend / add `TestDenylist_DaemonConfigDir` + case-variation) | test | transform/validate | `TestDenylist_HomeRooted` (write_test.go:478) | exact |
| `internal/files/sandbox_test.go` (`TestSandbox_WritePathSymlinkEscapeBlocked`) | test | file-I/O | `TestSandbox_SymlinkEscapeBlocked` (sandbox_test.go:217) | exact (mirror read→write) |
| `internal/files/sandbox_test.go` (`FuzzSandboxWrite` seed top-ups) | test (fuzz) | transform | `FuzzSandboxWrite` (sandbox_test.go:367) | exact (add `f.Add(...)` lines) |
| `frontend/e2e/files-write.spec.ts` (Origin-mismatch scenario) | test (e2e) | request-response | standalone APIRequestContext smoke (spec.ts:570); `originAllowedForWrite` (capability_mw.go:187) | exact |
| `.planning/.../127-SECURITY.md` (new) | doc | — | `83-SECURITY.md`, `74-SECURITY.md`, `60-SECURITY.md` | role-match (prior SECURITY artifacts) |

## Pattern Assignments

### `internal/files/sandbox.go` — `denylistCheck` (utility, transform/validate)

**Analog:** itself. The function already builds a canonical `$HOME`-relative path and matches
a base-name set + dir-prefix set. Extend BOTH the prefix set (GAP 1) and the comparison (GAP 2).

**Existing structure to extend** (sandbox.go:124-147):
```go
rel, err := filepath.Rel(home, canonAbs)
if err != nil || strings.HasPrefix(rel, "..") {
    return nil // not under $HOME — denylist does not apply
}
// Shell RC files — exact base-name match.
base := filepath.Base(canonAbs)
switch base {
case ".bashrc", ".zshrc", ".profile", ".bash_profile",
    ".zprofile", ".zshenv", ".bash_login":
    return ErrProtectedSystemFile
}
// Directory-prefix protections (forward-slash normalised).
relSlash := filepath.ToSlash(rel)
for _, dir := range []string{".ssh/", ".claude/", ".config/agenthub/"} {
    if relSlash == strings.TrimSuffix(dir, "/") || strings.HasPrefix(relSlash, dir) {
        return ErrProtectedSystemFile
    }
}
return nil
```

**GAP 1 fix — daemon config dir derivation.** The daemon derives its config dir from
`os.UserConfigDir()` (engine.go:62-70):
```go
func daemonConfigDir() string {
    base, err := os.UserConfigDir()       // macOS → ~/Library/Application Support
    if err != nil { base = os.TempDir() }
    dir := filepath.Join(base, "agenthub")
    _ = os.MkdirAll(dir, 0700)
    return dir
}
```
The denylist must use the SAME derivation. Compute the `$HOME`-relative form of
`filepath.Join(os.UserConfigDir(), "agenthub")` and add it to the dir-prefix slice
(in `filepath.ToSlash` form). Keep the literal `.config/agenthub/` for cross-platform
copied trees (belt-and-suspenders). Do NOT import the daemon package — replicate the
two-line derivation locally (engine.go's own comment says "internal packages cannot
import main"; same applies — keep `internal/files` dependency-free of `internal/daemon`).
`[VERIFIED on host: os.UserConfigDir() = /Users/ken/Library/Application Support]`

**GAP 2 fix — case-fold.** All protected names are ASCII, so `strings.ToLower` is safe and
sufficient for the high-value macOS/Windows case vector. Apply to `base` before the `switch`
(switch on `strings.ToLower(base)`) and to `relSlash` before the prefix loop
(compare `strings.ToLower(relSlash)` against already-lowercase prefixes).

**Unicode NFC — RESOLVED ASSUMPTION (overrides RESEARCH A1):** RESEARCH assumed
`golang.org/x/text` was NOT in go.mod and recommended skipping NFC. **It IS present:**
`go.mod:140 golang.org/x/text v0.37.0 // indirect`. NFC normalization via
`golang.org/x/text/unicode/norm` is therefore available at zero new-dependency cost
(it would promote an indirect dep to direct). Planner decision: ASCII case-fold closes
the realistic vector and all protected names are ASCII, so NFC remains LOW-residual and
OPTIONAL. If the planner wants belt-and-suspenders, `norm.NFC.String(rel)` before
comparison is now a legitimate, in-tree option — not a new dependency. Document whichever
choice is made in 127-SECURITY.md's residual-risk register.

**Fail-closed convention (preserve):** the function returns `ErrProtectedSystemFile` on
match and `nil` otherwise; the CR-02 EvalSymlinks-the-parent block (sandbox.go:114-122)
already canonicalizes symlinked path components before the `Rel` compare. Do not weaken it.

---

### `internal/files/write_test.go` — denylist tests (test, transform/validate)

**Analog:** `TestDenylist_HomeRooted` (write_test.go:478-576).

**Fake-HOME + sandbox-at-HOME setup pattern** (write_test.go:478-502):
```go
fakeHome := t.TempDir()
setHomeEnv(t, fakeHome)                 // helper at write_test.go:467
// seed prerequisite files so rename/delete reach the denylist, not ENOENT
os.WriteFile(filepath.Join(fakeHome, ".bashrc"), []byte("# shell rc"), 0o644)
os.MkdirAll(filepath.Join(fakeHome, ".ssh"), 0o700)
sb, err := files.NewSandbox(fakeHome)   // root AT $HOME — the dangerous case
```

**Per-target × per-method table-test pattern** (write_test.go:504-557): iterate a
`denyTargets` slice across `WriteFileAtomic` / `Mkdir` / `Delete` / `Rename-into`,
asserting `isProtected(err)` (helper at write_test.go:605). Mirror this exactly.

**New: `TestDenylist_DaemonConfigDir`** — guard the platform-specific path with `runtime.GOOS`
(PITFALLS Pitfall 2: never hardcode `~/Library/Application Support`). Derive the expected
relative path from `os.UserConfigDir()` the same way the fix does. Because the test sets
`HOME` to a tempdir via `setHomeEnv`, also set the matching platform config-dir env so the
derivation lands under the fake HOME — or compute the target relative to whatever
`os.UserConfigDir()` returns and skip if it isn't under the fake HOME. Assert
`isProtected(err)` for write/rename/delete of `<configdir>/agenthub/settings.json`.

**New: case-variation cases** — add `.BASHRC`, `.Bashrc`, `.SSH/authorized_keys` to a
`denyTargets`-style slice and assert `isProtected`. (On case-insensitive volumes these
resolve to the same inode; the fix must catch them lexically regardless of FS.)

**Negative control (preserve):** `TestDenylist_NonHomeRootedUnaffected` (write_test.go:581)
must still pass — a `.bashrc` literally inside a non-$HOME sandbox stays writable. The
case-fold/config-dir additions must not over-match outside $HOME.

---

### `internal/files/sandbox_test.go` — `TestSandbox_WritePathSymlinkEscapeBlocked` (test, file-I/O)

**Analog:** `TestSandbox_SymlinkEscapeBlocked` (sandbox_test.go:217-248) — the READ-path test.

**Mirror these load-bearing elements** (from the read test):
```go
if runtime.GOOS == "windows" { t.Skip("symlink creation requires admin on Windows") }
sb, root := newTestSandbox(t)
outside := t.TempDir()
secretPath := filepath.Join(outside, "secret")
os.WriteFile(secretPath, []byte("leaked"), 0o644)
// POSITIVE CONTROL (WR-03): prove outside/secret is really readable, so the
// negative assertion can't pass merely via ENOENT masking a broken impl.
data, _ := os.ReadFile(secretPath) // must == "leaked"
linkPath := filepath.Join(root, "escape")
if err := os.Symlink(outside, linkPath); err != nil { t.Skipf("symlink unsupported: %v", err) }
```

**Write-path assertions to add** (the new coverage): through the `escape` symlink, each must
return a non-nil error AND create nothing outside root:
```go
sb.WriteFileAtomic("escape/pwned", []byte("x"))  // want err
sb.Rename("a.txt", "escape/pwned")               // want err  (a.txt seeded by newTestSandbox)
sb.Mkdir("escape/sub")                           // want err
// after each: stat the outside path and assert it was NOT created/modified
```

**Why it holds (no code change needed):** every write method does `os.OpenRoot(s.rootPath)`
then `root.OpenFile`/`root.Rename`/`root.Mkdir` (sandbox.go:222-237 for write; the same
`os.OpenRoot` + native-`root.*` pattern in Rename/Mkdir/Delete). `os.Root` rejects escaping
symlinks atomically. This test makes SC1 explicit — it does not add protection.

**Optional handler-level 403:** if the planner wants the HTTP mapping proven, assert the
write handler returns 403 (the `os.Root` rejection maps via `writeWriteError`, write.go:288).

---

### `internal/files/sandbox_test.go` — `FuzzSandboxWrite` seed top-ups (test, fuzz)

**Analog:** `FuzzSandboxWrite` (sandbox_test.go:367-441). Add `f.Add(...)` lines alongside
the existing write-specific seeds (sandbox_test.go:434-440):
```go
// existing write-specific seeds end at:
f.Add("..%2f..%2f.bashrc")
// --- ADD (Phase 127, SEC-06): case-variation + multipart-filename vectors ---
f.Add("../../.BASHRC")
f.Add("../../.Bashrc")
f.Add(".SSH/authorized_keys")
f.Add("../../../etc/passwd")   // multipart-filename-style injection
```
**Merge gate unchanged:** `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/`
→ 0 crashes. The corpus already covers rename-dest traversal + canonical denylist-bypass;
these top-ups finalize it against the case vector the GAP 2 fix introduces.

---

### `frontend/e2e/files-write.spec.ts` — Origin-mismatch scenario (test, e2e/request-response)

**Analog:** the standalone APIRequestContext smoke (spec.ts:570-588) for the request shape,
and `originAllowedForWrite` (capability_mw.go:187-198) for the asserted behavior.

**Request-shape pattern to copy** (spec.ts:570-584):
```ts
const env = loadFixtureEnv()
const ctx = await playwrightRequest.newContext({ ignoreHTTPSErrors: true })
try {
  const params = new URLSearchParams({
    session: 'playwright-test-session',
    path: `origin-test-${Date.now()}.txt`,
    cap: env.writeCap,                 // VALID files.write cap
  })
  const url = `${env.baseURL}/api/files/write?${params.toString()}`
  const resp = await ctx.put(url, {
    headers: {
      'Content-Type': 'application/octet-stream',
      Origin: 'https://evil.example.com',   // NEW: mismatched Origin
    },
    data: 'csrf attempt',
  })
  expect(resp.status(), 'mismatched Origin must 403 even with valid write cap').toBe(403)
} finally { await ctx.dispose() }
```
**Behavior anchor:** `originAllowedForWrite` returns false when a present Origin ≠
`ws.BaseURL()` (capability_mw.go:193-197); `requireFilesWrite` runs the Origin check AFTER
the cap check (capability_mw.go:160-167), so the 403 here proves CSRF rejection on a
fully-capable token. Existing scenarios 1-3 (write-OK / mount / 403-no-cap, spec.ts:89/110/128)
must stay green cross-browser. Spec runs serial (`test.describe.configure({ mode: 'serial' })`,
spec.ts:34) — keep the new cell consistent with that.

---

### `.planning/phases/127-web-share-write-security-hardening/127-SECURITY.md` (doc)

**Analogs (structure precedent):**
- `.planning/milestones/v3.0-phases/83-settings-ui-alignment/83-SECURITY.md`
- `.planning/milestones/v2.0-phases/74-multi-client-fan-out/74-SECURITY.md`
- `.planning/milestones/v1.11-phases/60-local-network-fallback/60-SECURITY.md`

Read one for the exact heading shape before authoring. Content is fully enumerated in
RESEARCH.md's "Capability-Escalation Audit" table (RESEARCH lines 131-143) — the audit is a
finite, already-verified set of code paths; transcribe it with the per-surface enforcement
matrix below, the denylist threat model, and a residual-risk register. No auditor-agent spawn
needed (RESEARCH Open Q2). Filename per RESEARCH Open Q1 recommendation: `127-SECURITY.md` in
the phase dir.

## Shared Patterns

### Capability enforcement (source for the SECURITY artifact's per-surface matrix)

**Webserver write gate** — `requireFilesWrite` (capability_mw.go:147-170):
wraps `requireCapability` (HMAC + SID + grant + session-enabled) → then
`HasPerm(claims.Perms, PermFilesWrite)` (capability_mw.go:156) → then `originAllowedForWrite`
(capability_mw.go:164). Order is load-bearing: 401 (cap) → 403 (perm) → 403 (origin).
`HasPerm` is whole-token `strings.Split`, NOT `strings.Contains` — a perm string
`"no-files.write"` does not match (RESEARCH line 140). `PermFilesWrite` referenced at
capability_mw.go:156.

**CSRF Origin inversion** — `originAllowedForWrite` (capability_mw.go:187-198):
absent Origin passes vacuously (desktop Wails sends none); present Origin must byte-match
`ws.BaseURL()`; empty BaseURL with present Origin fails closed. This is the INVERSE of the
WS-upgrade `requireAllowedOrigin`. The SECURITY artifact must state this inversion explicitly.

**Remote proxy cap-strip** — `proxyRemoteFiles` (remote_files.go:175-229):
strips any caller-supplied `?cap` (case-insensitive, remote_files.go:204), force-sets
`session` from the path (prevents session-confusion, remote_files.go:215), injects the
stored cap (remote_files.go:216), forwards `r.Body` for PUT/POST/PATCH (remote_files.go:224-226).
The REMOTE peer's `requireFilesWrite` is the actual enforcer — defense in depth across the
tailnet. Document as SECURE.

**Daemon Unix-socket / named-pipe write routes** — NO cap enforcement, by design (loopback
trust, WEB-01 precedent). The SECURITY artifact must record this as an ACCEPTED, DOCUMENTED
risk, not a finding (RESEARCH line 143). It is not a web-share surface.

**Owner-token opt-in** — `files.write` is added to the owner token ONLY when
`filesWriteEnabledFor(sid)` (engine.go:545; issuance at api.go:1070-1079 per RESEARCH);
the viewer token is bare `"read"`. Never default-on.

### Denylist (the security boundary all four write methods call)

`denylistCheck` (sandbox.go:96-148) is invoked from `WriteFileAtomic` (sandbox.go:219),
`Rename` (both src+dest), `Mkdir`, `MkdirAll`, `Delete`. It is the single chokepoint —
the two GAP fixes here protect every write method at once. Returns `ErrProtectedSystemFile`;
fail-closed on symlinked-parent ambiguity via the CR-02 EvalSymlinks block (sandbox.go:114-122).

## No Analog Found

None. Every Phase 127 gap-fix has an exact in-repo analog (same file/function for the code
changes, sibling test for each new test, prior-milestone artifacts for the doc).

## Metadata

**Analog search scope:** `internal/files/`, `internal/webserver/`, `internal/daemon/`,
`frontend/e2e/`, `.planning/milestones/*/SECURITY.md`
**Files scanned:** sandbox.go, write.go, sandbox_test.go, write_test.go, engine.go,
capability_mw.go, remote_files.go, files-write.spec.ts, go.mod
**Key verification this session:** `golang.org/x/text v0.37.0` IS in go.mod (go.mod:140) —
overrides RESEARCH Assumption A1; NFC is now an in-tree option, not a new dependency.
**Pattern extraction date:** 2026-06-14
