---
phase: 129-write-concurrency-fix-dns-error-ux
reviewed: 2026-06-15T00:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - frontend/src/App.tsx
  - frontend/src/components/RemoteBrowseDNSWarning.tsx
  - frontend/src/wailsjs/wailsjs/go/models.ts
  - internal/daemon/relay_remote_files_test.go
  - internal/daemon/remote_files.go
  - internal/daemon/remote_files_test.go
  - internal/files/sandbox.go
  - internal/webserver/tailscale.go
  - internal/webserver/tailscale_test.go
findings:
  critical: 0
  warning: 4
  info: 5
  total: 9
status: issues_found
---

# Phase 129: Code Review Report

**Reviewed:** 2026-06-15
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Reviewed the Phase 129 write-concurrency fix (per-path mutex in `WriteFileAtomic`),
the Tailscale accept-dns UX surface (`isUnresolvableMagicDNS` classifier, injectable
`prefsFunc`, `RemoteBrowseDNSWarning`), and supporting tests. The package compiles
clean (`go build ./internal/...`) and passes `go vet` for all three packages.

No BLOCKERs found. The concurrency fix is structurally sound: the per-path mutex is
acquired before `os.OpenRoot` and released via `defer` on every error path, so the
mutex is never left held. The lock key is the absolute path, correctly avoiding
cross-Sandbox false contention. The DNS classifier uses `errors.As` correctly and
gates on a true `*net.DNSError`, and the actionable message is a fixed string that
leaks no hostname or cap token. The React banner is a static message (no XSS surface).

The findings below are robustness and consistency concerns. Two WARNINGs concern
real defect risk in the concurrency contract: a validator-classification gap that
can let a conflicting write through as a silent winner, and a hostname-substring
classifier that over-matches. The remainder are quality/consistency items.

## Warnings

### WR-01: `isUnresolvableMagicDNS` over-matches on substring, ignores host boundary

**File:** `internal/daemon/remote_files.go:298-306`
**Issue:** The MagicDNS check is `strings.Contains(baseURL, ".ts.net")`. This is a
substring test against the entire URL, not a hostname check. It matches paths,
query strings, and userinfo — e.g. `https://evil.example/.ts.net/x` or
`https://attacker.com/?q=.ts.net` would be classified as MagicDNS. More importantly
it matches any host that merely *contains* `.ts.net` as a substring, such as
`foo.ts.network` or `notreally.ts.net.evil.com`. Because the classifier only fires
on an actual `*net.DNSError` the security impact is low (worst case: a misleading
"enable accept-dns" message on a 502), but the message can therefore be emitted for
a non-Tailscale host whose DNS lookup happened to fail, which is the exact
"information / actionability" confusion DNS-02 was meant to prevent. The companion
banner string `.tailscale.net` has the same issue.
**Fix:** Parse the host and check the suffix on the hostname only:
```go
func isUnresolvableMagicDNS(err error, baseURL string) bool {
	var dnsErr *net.DNSError
	if !errors.As(err, &dnsErr) {
		return false
	}
	u, perr := url.Parse(baseURL)
	if perr != nil {
		return false
	}
	host := u.Hostname() // strips port + userinfo
	return strings.HasSuffix(host, ".ts.net") ||
		strings.HasSuffix(host, ".tailscale.net")
}
```

### WR-02: `WriteFileAtomic` validator re-check silently skips conflict detection when `root.Stat` errors for a non-ENOENT reason

**File:** `internal/files/sandbox.go:340-353`
**Issue:** The optimistic-concurrency re-check is `if fi, err := root.Stat(cleaned); err == nil { ... }`. The comment states that a Stat error "means another writer deleted it, which is fine." That reasoning only holds for `fs.ErrNotExist`. Any *other* Stat error (permission change, I/O error, a path component that became a non-directory) is also swallowed: the code falls through and renames unconditionally, defeating the single-winner guarantee for that path. A caller that asked for a specific validator and got a transient Stat failure will have its write applied as if the precondition passed — a silent last-writer-wins instead of the documented `ErrPreconditionFailed`. This is the one place in the concurrency fix where an error is converted into a silent success (contra CLAUDE.md §Silent Fallbacks).
**Fix:** Only treat the not-exist case as "skip the check"; surface other Stat errors:
```go
if fi, statErr := root.Stat(cleaned); statErr == nil {
	cur := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))
	if cur != validator {
		_ = root.Remove(tmp)
		return ErrPreconditionFailed
	}
} else if !errors.Is(statErr, fs.ErrNotExist) {
	_ = root.Remove(tmp)
	return fmt.Errorf("files: stat for precondition: %w", statErr)
}
```

### WR-03: Lock key derived from `filepath.Join(rootPath, cleaned)` is not the canonical on-disk path; symlinked directory components can split the lock for a single file

**File:** `internal/files/sandbox.go:292-293` (and `denylistCheck` canonicalization at 145-150)
**Issue:** The per-path mutex key is the *lexical* `filepath.Join(s.rootPath, cleaned)`. `denylistCheck` (lines 145-150) goes to some lengths to canonicalize the target's parent via `EvalSymlinks` before its `filepath.Rel` comparison, acknowledging that the directory portion of the path may contain symlinks. The lock key does not do this. If two Sandbox instances are rooted at paths that resolve to the same physical directory (e.g. one root passed through a symlink that `NewSandbox` resolves, another root that resolves to the same place but where `cleaned` traverses a differently-named symlinked subdir), two writers to the *same physical file* can acquire two different mutexes and race. This is a narrow case — `NewSandbox` resolves `rootPath` once, so the common path is safe — but the lock-key derivation is weaker than the denylist's own canonicalization standard and weaker than the security comment at lines 85-91 implies ("two Sandbox instances rooted at different directories writing a file with the same relative name do NOT contend" is true, but the converse — same physical file via different lexical keys — is not guaranteed).
**Fix:** Either document the assumption explicitly (single resolved root per physical dir, no in-tree symlinked dirs for write targets) as an accepted limitation, or derive the lock key from the canonicalized parent the same way `denylistCheck` already computes `canonAbs`. Given the per-path lock is the entire correctness mechanism for RACE-01, the documentation at lines 85-91 should at minimum state that the guarantee holds only for lexically-identical absolute paths.

### WR-04: `relay_remote_files_test.go` redefines builtin `min`, shadowing it package-wide for the test build

**File:** `internal/daemon/relay_remote_files_test.go:406-413`
**Issue:** The file defines `func min(a, b int) int` "for Go <1.21 compatibility." The module builds on go1.26 (`go version` = go1.26.4) and `go.mod`/toolchain target Go 1.21+. A package-level `min` shadows the builtin `min` for the *entire* `daemon_test` package, not just this file. Any other test in `daemon_test` that intends to call the generic builtin `min` (e.g. on `int64` or `float64`) will now fail to compile or silently bind to this int-only version. It is dead-compat code that introduces a package-wide footgun.
**Fix:** Delete the function and rely on the builtin:
```go
// remove lines 406-413 entirely; built-in min(a, b) works for the int slice expr.
```

## Info

### IN-01: `denylistCheck` fail-open on `filepath.Rel` error contradicts the "fail-closed" doc header

**File:** `internal/files/sandbox.go:115-122, 152-159`
**Issue:** The function doc (lines 115-122) advertises "fails closed (returns ErrProtectedSystemFile) on any ambiguity." But the actual `filepath.Rel` error path (lines 153-158) returns `nil` (fail-open) and the inline comment correctly explains this is the "not under $HOME" case. The doc header and the code disagree on the fail-open/fail-closed posture for the Rel-error branch. The code behavior is defensible (Rel only errors when paths are on different volumes, i.e. genuinely not under HOME) but the header overstates the guarantee.
**Fix:** Adjust the doc header to say it fails closed only for *symlinked-component ambiguity*, and fails open (correctly) when the target is provably not under $HOME.

### IN-02: `App.tsx` duplicates the `TailscaleHealth` shape as two inline literals instead of importing `webserver.TailscaleHealth`

**File:** `frontend/src/App.tsx:139-149, 509-519`
**Issue:** The generated `webserver.TailscaleHealth` type exists in `models.ts:273-300` with `acceptDns: boolean` (required). App.tsx instead hand-rolls the same shape twice — once for the `tailscaleHealth` state (line 139-149) and again for the `EventsOn('tailscale:health', ...)` handler (line 509-519) — both with `acceptDns?: boolean` (optional). The two inline copies can drift from the Go source of truth, and they already diverge from the generated type on the optionality of `acceptDns`. The optional-vs-required mismatch is the load-bearing distinction that `RemoteBrowseDNSWarning` relies on (`acceptDns === undefined` = prefs unavailable), so this is intentional, but maintaining it as two untyped literals invites silent drift if a field is added Go-side.
**Fix:** Import `webserver` from `models.ts` and derive: `type TSHealth = Omit<webserver.TailscaleHealth, 'acceptDns'> & { acceptDns?: boolean }`, used in both spots.

### IN-03: `RemoteBrowseDNSWarning` message string is duplicated against the Go daemon error string with no parity guard

**File:** `frontend/src/components/RemoteBrowseDNSWarning.tsx:41` and `internal/daemon/remote_files.go:261`
**Issue:** The banner message "Enable Tailscale DNS (accept-dns) to browse remote sessions" is a literal that exactly matches the daemon's 502 body literal at `remote_files.go:261` and the test constant at `remote_files_test.go:588`. Three copies, no shared constant or parity test. Per the project's cross-surface-parity standard, a drift here would silently desync the proactive banner from the reactive 502 message.
**Fix:** Not a code bug, but consider a parity test (or a comment cross-referencing the Go literal) so the two user-facing strings stay identical.

### IN-04: `pathLocks` is an unbounded, never-pruned process-lifetime map

**File:** `internal/files/sandbox.go:76-92`
**Issue:** The design note (lines 81-88) explicitly accepts unbounded growth (T-129-04), reasoning that each entry is ~8 bytes and bounded by distinct paths written in process lifetime. For the daemon's expected workload this is fine and was an accepted decision. Flagging only because the map keys are full absolute path strings (not 8 bytes — the string header plus backing bytes, tens to hundreds of bytes per distinct path), so the "negligible ~8 bytes" estimate in the comment understates real growth for a long-lived daemon writing many distinct paths. Still almost certainly acceptable; the comment's size estimate is just optimistic.
**Fix:** None required (accepted decision). Optionally correct the per-entry size estimate in the comment.

### IN-05: `checkHealth` probes prefs only when `Connected`, so `AcceptDNS` is always false for a node that is up but not in "Running" state

**File:** `internal/webserver/tailscale.go:62-79`
**Issue:** `AcceptDNS` is only set inside `if h.Connected`. For a node whose backend is "Starting"/"NeedsLogin" but whose prefs are readable, `AcceptDNS` reports false even if accept-dns is actually enabled. The `RemoteBrowseDNSWarning` only renders when `connected === true`, so the banner consequence is benign (no warning shown when not connected). This is consistent with the documented "safe zero value" intent, just noting the field is not a faithful reflection of the pref outside the Running state — anything reading `AcceptDNS` for a non-connected node gets a false negative.
**Fix:** None required given the only consumer gates on `connected`. Document that `AcceptDNS` is meaningful only when `Connected == true`.

---

_Reviewed: 2026-06-15_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
