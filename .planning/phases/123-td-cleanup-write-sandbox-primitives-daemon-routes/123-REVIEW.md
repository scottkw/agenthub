---
phase: 123
slug: td-cleanup-write-sandbox-primitives-daemon-routes
depth: standard
status: findings
files_reviewed: 6
findings:
  critical: 2
  warning: 6
  info: 3
  total: 11
reviewed: 2026-06-14
diff_base: 64f10b8
resolution: fixed
resolution_note: >
  All 2 critical + 6 warning findings fixed (CR-01, CR-02, WR-01..WR-06) plus IN-03
  (sentinel errors). Commits 1ea4d65, ffe1ea6, c601efc, ba0a4c0, and WR-06 fix.
  go test -race green (files + daemon), gofmt + vet clean. IN-01 (duplicate
  renameRequest struct) and IN-02 (magic constants) deferred — Info-only, minor tech debt.
---

# Phase 123 Code Review — Write Sandbox Primitives + Daemon Routes

**Depth:** standard
**Files reviewed (6):** internal/files/sandbox.go, internal/files/write.go, internal/files/types.go, internal/daemon/api.go, internal/daemon/client.go, internal/daemon/client_remote_files.go

Security-critical phase (write surface over auth-less daemon socket). Reviewer cleared traversal/symlink-escape on the `os.Root` + `validateRelativePath` layer and confirmed `RemoveAll(".")` does not wipe the sandbox root under Go 1.26. The material findings are below.

---

## Critical / BLOCKER

### CR-01: Empty/dot upload filename writes file content onto a directory (500s + stray temp files)
**File:** `internal/files/write.go:68-111`

`safeName := filepath.Base(header.Filename)` returns `"."` when `header.Filename == ""`, so `target := filepath.Join(dir, ".")` collapses to `dir`, and `WriteFileAtomic(dir, data)` attempts to rename a regular temp file onto an existing directory → opaque 500 + stray `.agenthub-tmp-*`. Trivially craftable over the auth-less socket.

**Fix:** Reject empty/dot/separator filenames before building the target:
```go
safeName := filepath.Base(header.Filename)
if safeName == "." || safeName == ".." || safeName == "" || strings.ContainsAny(safeName, `/\`) {
    http.Error(w, "invalid upload filename", http.StatusBadRequest)
    return
}
```

### CR-02: `denylistCheck` can fail OPEN under a symlinked sandbox root (lexical target vs canonicalized home)
**File:** `internal/files/sandbox.go:83-116`

`home` is `EvalSymlinks`-resolved but `abs := filepath.Join(s.rootPath, cleaned)` is built lexically. If any component of `rootPath` is a symlink not collapsed by the single construction-time `EvalSymlinks`, `filepath.Rel(home, abs)` returns a `..`-prefixed path and the function returns `nil` — silently skipping the shell-RC / `.ssh` / `.claude` denylist. The denylist is the only guard for `~/.ssh/authorized_keys` when the session workdir is under `$HOME`; a fail-open here is security-relevant and contradicts CLAUDE.md "Silent Fallbacks / let it crash".

**Fix:** Canonicalize the target's existing parent the same way `home` is canonicalized before computing the relation, and treat the `$HOME`-rooted case as fail-closed. Add a guard asserting `rootPath == EvalSymlinks(rootPath)`.

---

## Warnings

### WR-01: Write/Delete/Mkdir default `path` to `"."` → nonsensical root ops with opaque 500s
**File:** `internal/files/write.go:44,122,174` (via `relPath`). A missing/empty `path` on a write verb is a client error → return 400, don't reuse the read-side `"."` default.

### WR-02: OS errors (missing parent, rename-onto-dir, ENOTEMPTY) map to 500 instead of 4xx
**File:** `internal/files/write.go:190-199`. `writeWriteError` should branch `errors.Is(err, fs.ErrNotExist)` → 404, `errors.Is(err, fs.ErrExist)` → 409, keep 500 only for genuinely unexpected errors.

### WR-03: `rand.Read` error ignored when generating temp-file suffix
**File:** `internal/files/sandbox.go:186-188`. Surface a failed CSPRNG read (`return fmt.Errorf("files: generate temp suffix: %w", err)`) per let-it-crash.

### WR-04: `ExchangeJoinCodeAtURL` trusts upstream `Location` host without validation (MITM cap-token swap risk)
**File:** `internal/daemon/client_remote_files.go:133-155`. With `InsecureSkipVerify: true` on the transport, a tailnet MITM can return `Location: https://attacker/...?cap=<attacker-token>`. After parsing `loc`, if absolute, assert `u.Host == parsed.Host` and scheme match before accepting the cap.

### WR-05: Error-kind parsing uses prefix-anchored `TrimPrefix` on a `Contains` match → garbage kinds for absolute error URLs
**File:** `internal/daemon/client_remote_files.go:137-144`. Parse `loc` once and read `u.Query().Get("error")` instead of string-prefix surgery.

### WR-06: PUT `/api/files/write` reads body with unbounded `io.ReadAll` — no size cap (DoS)
**File:** `internal/files/write.go:45`. `Upload` wraps body in `http.MaxBytesReader(50 MiB)`; `Write` does not. Same handler is mounted on relay TCP (`api.go:236`) and the webserver, so loopback trust does not justify unbounded allocation. Wrap `r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)` and map overflow → 413.

---

## Info

### IN-01: Duplicate `renameRequest` struct in `files` and `daemon` packages (`write.go:131-135`, `client.go:596-599`) — drift risk.
### IN-02: Magic `50ms` / `maxAttempts=3` Windows rename retry constants (`sandbox.go:222-240`) — promote to named constants.
### IN-03: `isValidationError` classifies 403 vs 500 by `msg[:7] == "files: "` string prefix (`write.go:204-213`) — brittle; use sentinel errors (`errors.Is`) so future `fmt.Errorf("files: ...")` OS wraps aren't misclassified as traversal rejections. (Resolves WR-02 robustly.)

---

## Cleared (do not re-flag)
`os.Root` + `validateRelativePath` reject absolute/`..`/drive/UNC/ADS/null/Windows-device paths cross-platform; `RemoveAll(".")` does not wipe root (Go 1.26, verified); atomic temp+Sync+Rename sound; `Rename` validates both paths. `remote_files.go`/`remote_caps.go` proxy handlers are out of phase-123 scope.
