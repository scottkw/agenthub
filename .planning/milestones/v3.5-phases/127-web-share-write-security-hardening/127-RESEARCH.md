# Phase 127: Web-Share Write Security Hardening - Research

**Researched:** 2026-06-14
**Domain:** Security audit + targeted hardening of an already-built Go (`os.OpenRoot`-sandboxed) filesystem write surface exposed over a Tailscale web-share capability boundary
**Confidence:** HIGH (all claims verified against live source in `internal/files/`, `internal/webserver/`, `internal/daemon/`)

## Summary

Phase 127 is an **AUDIT-and-HARDEN** phase, not a build phase. The write surface was built and tested across Phases 123-125 (all source files exist on disk: `internal/files/write.go`, `internal/files/sandbox.go` with the denylist + atomic write + native `root.Rename`, `internal/webserver/capability_mw.go` with `requireFilesWrite` + CSRF Origin check, and `frontend/e2e/files-write.spec.ts` with 14 e2e scenarios). The overwhelming majority of SEC-01..SEC-07 is **already covered** by existing code and tests. The research task was to verify each success criterion against the real code and isolate the genuine gaps.

**The audit found three real GAPS and two SMALL gaps; everything else is ALREADY-COVERED:**

1. **GAP (SC2/SEC-02) — macOS daemon-config-dir denylist hole.** The denylist in `sandbox.go:denylistCheck` hardcodes `.config/agenthub/` as the daemon config dir. On macOS the actual daemon config dir is `~/Library/Application Support/agenthub/` (`os.UserConfigDir()` returns `~/Library/Application Support`, **verified on this host**). A home-rooted session on macOS can therefore write/rename/delete the daemon's own `settings.json` — the denylist never matches. SC2 explicitly lists "daemon config dir → 403". This is the single most important finding.
2. **GAP (SC2/SEC-02) — denylist does not normalize case or Unicode.** `denylistCheck` compares base names case-sensitively (`switch base { case ".bashrc" ... }`) and does no NFC/NFD normalization. On case-insensitive macOS/Windows filesystems, `.BASHRC` resolves to the same inode but bypasses the exact-match denylist. Path-encoding (`%2e`) is already neutralized upstream by `validateRelativePath` (no URL-decoding happens in the sandbox; the HTTP layer decodes once and `%252e` stays literal), so the encoding vector is largely closed — but case and Unicode are open.
3. **GAP (SC1/SEC-01) — no write-path symlink-escape test.** `TestSandbox_SymlinkEscapeBlocked` exercises `sb.Open` (the READ path) only. There is no test asserting that `WriteFileAtomic`/`Rename`/`Mkdir` through a symlink-escaping path returns an error → 403. The protection almost certainly holds (all write methods use `os.OpenRoot` + native `root.*`), but SC1 requires the explicit test and it does not exist.
4. **SMALL gap (SC4/SEC-04) — no committed SECURITY artifact.** SC4 requires the capability-escalation audit findings to live in a committed `.planning/` SECURITY artifact. None exists. The audit itself is mostly reading existing code paths (all of which are correct) and writing them up.
5. **SMALL gap (SC5/SEC-07) — no CSRF Origin-mismatch *e2e* scenario.** The CSRF Origin-mismatch is comprehensively **unit-tested** for all five write routes (`capability_test.go:690` "403 on mismatched Origin"). The Playwright `files-write.spec.ts` has scenarios 1-3 (web-share write OK / web-share mount / 403-without-cap) but no Origin-mismatch browser scenario. SEC-07 asks for the e2e cell.

**Primary recommendation:** Treat this phase as ~70% audit/test/doc and ~30% net-new code. The only net-new *production* code is the denylist hardening (fix the macOS config-dir lookup + add case/Unicode normalization). Everything else is adding tests (write-path symlink, denylist-bypass, fuzz-seed top-ups), one e2e scenario, and writing the SECURITY artifact.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **SC1:** symlink-escape on write/rename → HTTP 403 (os.OpenRoot TOCTOU boundary holds on write path).
- **SC2:** write/rename/delete of `~/.bashrc`, `~/.ssh/authorized_keys`, `~/.claude/CLAUDE.md`, daemon config dir → 403 Protected system file; denylist not bypassable by case variation, Unicode normalization, or path encoding.
- **SC3:** FuzzSandboxWrite finalized corpus (rename-dest traversal, denylist-bypass, upload-filename `../` injection) zero crashes; over-cap (>50 MiB) rejected by MaxBytesReader before ParseMultipartForm with clear error (not truncated file).
- **SC4:** capability escalation audit — no token lacking files.write reaches any write endpoint on any surface (daemon socket, webserver, remote proxy); files.write doesn't leak across sessions; findings in a committed SECURITY artifact.
- **SC5:** Playwright web-share write e2e — viewer with files.write writes OK; viewer without → 403; CSRF Origin-mismatch on POST/PUT/DELETE → 403.

### Claude's Discretion
Whether to use the gsd-security-auditor pattern; how to structure the SECURITY artifact; which additional fuzz seeds/test cases to add. Confirm existing mitigations hold; only add code where a gap is found.

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SEC-01 | Write-path symlink-escape test: write/rename/mkdir whose resolved target escapes sandbox returns 403, not 200 | GAP — only READ-path symlink test exists (`sandbox_test.go:217`). Need write-path test; protection itself holds via `os.OpenRoot` (verified: all write methods open root + use `root.*`). |
| SEC-02 | Shell-RC denylist tests: write/rename into `~/.bashrc`, `~/.ssh/authorized_keys`, `~/.claude/CLAUDE.md`, daemon config → 403 Protected system file | PARTIAL — `TestDenylist_HomeRooted` (`write_test.go:478`) covers `.bashrc`/`.ssh`/`.claude`. GAPS: (a) macOS daemon config dir (`~/Library/Application Support/agenthub`) not in denylist; (b) case/Unicode bypass not tested or defended. |
| SEC-03 | Upload abuse: multipart `../` filename sanitized; over-cap (>50 MiB) rejected by MaxBytesReader; no zip-slip | ALREADY-COVERED — `Upload` uses `filepath.Base` + empty/`.`/`..`/separator guard (`write.go:158-168`); `MaxBytesReader` before `ParseMultipartForm` (`write.go:131`); no archive extraction exists. Optionally add a multipart-filename fuzz seed. |
| SEC-04 | Capability-escalation audit: no token lacking files.write reaches any write endpoint on any surface; no cross-session leak; SECURITY artifact | PARTIAL — all enforcement paths verified correct (see Capability-Escalation Audit table). GAP: the committed SECURITY artifact does not exist yet. |
| SEC-05 | Data-integrity tests: concurrent-write race (two writers + If-Match) + atomic-rename failure paths leave no corrupt/partial file; original preserved | PARTIAL — `TestWriteFileAtomic_ConcurrentReadNeverPartial` + `TestWriteFileAtomic_ValidatorRecheck` + `TestWrite_IfMatch_*` exist. GAP if SC wants an explicit two-writers-racing-If-Match test and an interrupted-write/original-preserved assertion. Note: SC5 is not in the CONTEXT locked list but IS a phase requirement. |
| SEC-06 | FuzzSandboxWrite corpus finalized: rename-dest traversal, denylist-bypass, upload-filename-injection; 0 crashes merge gate | ALREADY-COVERED (mostly) — `FuzzSandboxWrite` (`sandbox_test.go:367`) seeds rename-dest traversal (`f.Add("a.txt", rawPath)` body), `.ssh/authorized_keys`, `.bashrc`, `.claude/CLAUDE.md`, `..%2f..%2f.bashrc`, temp-name collision. GAPS: no case-variation (`.BASHRC`) seed, no Unicode-normalized denylist seed, no explicit multipart-filename seed (upload sanitization is unit-tested, not fuzzed). |
| SEC-07 | Playwright web-share write e2e: viewer w/ files.write writes OK; viewer without → 403; CSRF Origin-mismatch rejected | PARTIAL — `files-write.spec.ts` scenarios 1-3 cover write-OK + mount + 403-without-cap. CSRF Origin-mismatch is **unit-tested** (`capability_test.go:690`) but has no e2e cell. GAP: add Origin-mismatch Playwright scenario. |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Sandbox confinement (path validation, symlink escape, denylist) | `internal/files` (Sandbox) | — | `os.OpenRoot` is the terminal security boundary; denylist is a method-level guard on all write methods. Tier-correct: never in the HTTP/middleware layer. |
| Capability gate (`files.write` HasPerm) | `internal/webserver` (`requireFilesWrite`) | — | Webserver tier only. The daemon socket is auth-less by design (loopback trust, WEB-01). Correct. |
| CSRF Origin check | `internal/webserver` (`originAllowedForWrite`) | — | State-changing-verb concern; lives in `requireFilesWrite`. Correct. |
| Cross-session SID scoping | `internal/webserver` (`requireCapability`) | — | `claims.SID != pathID → 403`. Write routes inherit it via the `requireFilesWrite → requireCapability` wrap. |
| Remote write cap forwarding | `internal/daemon` (`proxyRemoteFiles`) | remote peer `requireFilesWrite` | Proxy strips caller-supplied `cap` and injects the stored cap; the *remote* peer enforces `files.write` + Origin. Defense in depth across the tailnet. |
| e2e contract verification | `frontend/e2e` (Playwright) | Go unit/integration | Browser-level proof that the cap + Origin contract holds end-to-end on the web-share surface. |

## Standard Stack

No new packages. This phase is entirely Go stdlib (`testing`, `testing/fuzz`, `net/http/httptest`, `os`, `path/filepath`) plus the existing Playwright/Vitest frontend tooling already vendored for Phase 125.

| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| Go `testing` + native fuzzing | go1.26.4 (verified `go version` on host) | Unit tests, `FuzzSandboxWrite` merge gate | Already the project's test framework; fuzz corpus is an established merge gate (FSW-07). |
| Playwright | already installed (Phase 120/125) | web-share write e2e scenarios | `files-write.spec.ts` already exists; add scenarios, do not introduce new tooling. |
| `os.UserConfigDir` / `os.UserHomeDir` | go stdlib | Correct cross-platform daemon-config-dir derivation for the denylist fix | The daemon already uses `os.UserConfigDir()` in `engine.go:daemonConfigDir`; the denylist must use the SAME derivation rather than hardcoding `.config/agenthub/`. |

**Installation:** None.

## Package Legitimacy Audit

Not applicable — this phase installs no external packages. All work is Go stdlib tests, one Playwright scenario in an existing spec, and a markdown SECURITY artifact.

## Runtime State Inventory

Not a rename/refactor/migration phase. Omitted.

## What Is ALREADY-COVERED (cite the code/test)

> This is the heart of the research per the objective: do not re-plan what already holds.

| Success criterion area | Already-covered by | Evidence |
|------------------------|--------------------|----------|
| `os.OpenRoot` TOCTOU boundary on write methods | `WriteFileAtomic`, `Rename`, `Mkdir`, `MkdirAll`, `Delete` all `os.OpenRoot(s.rootPath)` then use `root.OpenFile`/`root.Rename`/`root.Mkdir`/`root.MkdirAll`/`root.RemoveAll` | `sandbox.go:222-400` — no hand-rolled `os.Rename(absOld,absNew)`; uses native `root.Rename` (sandbox.go:340). |
| Rename validates BOTH source and destination | `Rename` calls `validateAndClean(oldRel)` AND `validateAndClean(newRel)` + denylist on both | `sandbox.go:320-341`. `TestRename_DestinationTraversalRejected` (`write_test.go:283`), `TestRename_SourceTraversalRejected` (`write_test.go:311`). |
| Atomic write (temp + Sync + rename; never O_TRUNC in place) | `WriteFileAtomic` writes sibling `.agenthub-tmp-<rand>` via `O_EXCL`, `f.Sync()`, `root.Rename` | `sandbox.go:214-285`. `TestWriteFileAtomic_ConcurrentReadNeverPartial` (`write_test.go:145`). |
| MaxBytesReader before ParseMultipartForm (over-cap → 413, not truncated) | `Upload` sets `r.Body = http.MaxBytesReader(...)` then `ParseMultipartForm`; distinguishes 413 via `errors.As(&maxErr)` | `write.go:131-143`. `TestUpload_IN05_MalformedMultipart_Returns400` + `TestMaxUploadBytes_Is50MiB` (`write_test.go:742,974`). Write body also capped (`write.go:89`). |
| Multipart filename `../` sanitized | `safeName := filepath.Base(header.Filename)` + reject `""`/`.`/`..`/`/`/`\` | `write.go:158-168`. |
| Shell-RC / SSH / Claude denylist (home-rooted) | `denylistCheck` on `.bashrc`,`.zshrc`,`.profile`,`.bash_profile`,`.zprofile`,`.zshenv`,`.bash_login`, and prefixes `.ssh/`,`.claude/`,`.config/agenthub/` | `sandbox.go:96-148`. `TestDenylist_HomeRooted` + `TestDenylist_NonHomeRootedUnaffected` (`write_test.go:478,581`). Called from all four write methods (`sandbox.go:219,329,332,352,372,391`). |
| Denylist symlink fix (CR-02 fail-closed) | `denylistCheck` EvalSymlinks the existing parent before `filepath.Rel`; macOS `/var/folders`→`/private/...` handled | `sandbox.go:114-131` + the macOS EvalSymlinks(home) step at `sandbox.go:110`. |
| Path-encoding bypass (`%2e`, overlong) | `validateRelativePath` rejects null bytes, ADS colon, device names, drive letters, UNC, traversal; HTTP layer decodes once so `%252e` stays literal | `sandbox.go:452-501`. Fuzz seeds `%2e%2e%2fetc%2fpasswd`, `%252e...` (`sandbox_test.go:262`). |
| `files.write` capability gate (HasPerm, not strings.Contains) | `requireFilesWrite` → `requireCapability` → `HasPerm(claims.Perms, PermFilesWrite)` | `capability_mw.go` (requireFilesWrite). `HasPerm` is `strings.Split`-based whole-token (`capability.go:51`). |
| CSRF Origin check on write verbs | `originAllowedForWrite`: absent Origin passes vacuously (desktop Wails), present must byte-match `BaseURL()`, empty BaseURL fails closed | `capability_mw.go` (originAllowedForWrite). Unit-tested for all 5 routes: 403-on-mismatch / vacuous-pass-on-absent / pass-on-match (`capability_test.go:690-739`). |
| Cross-session scoping | `requireCapability` rejects `claims.SID != pathID` with 403 | `capability_mw.go` requireCapability. |
| Owner token opt-in for files.write (never default) | `issueCapabilitiesForSession`: `ownerPerms += ",files.write"` ONLY `if a.engine.filesWriteEnabledFor(sessionID)`; viewer token is bare `"read"` | `api.go:1070-1079`. |
| Remote proxy: strips caller cap, forwards body, remote peer enforces | `proxyRemoteFiles` strips any caller `?cap`, injects stored token, forwards `r.Body` for PUT/POST/PATCH; remote `requireFilesWrite` enforces | `remote_files.go:197-226`. |
| Method routing (405 on wrong verb) | Go 1.22+ method-prefix mux: `PUT /api/files/write` etc. | `server.go:512-521`, `api.go:152-156`. `TestFilesRoutes_*Returns405*` (`files_routes_test.go:232-255`). |
| HEAD /write canWrite probe (200 with cap / 403 without) | `Write` short-circuits HEAD after session resolve; middleware fires first | `write.go:62-67`. `TestFilesRoutes_HeadWrite_With/WithoutFilesWrite` (`files_routes_test.go:267,286`). |
| If-Match / 412 TOCTOU re-stat | `Write` checks If-Match pre-write; `WriteFileAtomic` re-checks validator immediately before rename | `write.go:77-104`, `sandbox.go:256-276`. `TestWrite_IfMatch_*`, `TestWriteFileAtomic_ValidatorRecheck` (`write_test.go:644-907`). |
| e2e web-share write OK + 403-without-cap | `files-write.spec.ts` scenarios 1, 3 (+ 14 scenarios total cross-browser) | `frontend/e2e/files-write.spec.ts:89,128`. |

## The GAPS (what this phase must actually do)

### GAP 1 (NET-NEW CODE) — macOS daemon-config-dir denylist hole [SEC-02]
**What's wrong:** `denylistCheck` (`sandbox.go:142`) protects the literal relative prefix `.config/agenthub/`. The actual daemon config dir comes from `daemonConfigDir()` (`engine.go:62`) which calls `os.UserConfigDir()`. **Verified on this host:** `os.UserConfigDir()` → `/Users/ken/Library/Application Support` on macOS (NOT `~/.config`). So on macOS the daemon's own `settings.json` lives at `~/Library/Application Support/agenthub/settings.json`, which the denylist never matches. A home-rooted session can overwrite daemon settings.
**Fix:** Derive the daemon config dir inside `denylistCheck` from `os.UserConfigDir()` (relative to `$HOME`) and add that relative prefix to the protected set — covering macOS (`Library/Application Support/agenthub/`), Linux (`.config/agenthub/`), and Windows (`AppData/Roaming/agenthub/`). Keep the literal `.config/agenthub/` as belt-and-suspenders for cross-platform copied trees. `[VERIFIED: os.UserConfigDir on host = ~/Library/Application Support]`
**Test:** Extend `TestDenylist_HomeRooted` (or add `TestDenylist_DaemonConfigDir`) to set `os.UserConfigDir` expectations and assert 403 for the platform-correct config path. The test must run on the host OS — guard with `runtime.GOOS` rather than hardcoding a path.

### GAP 2 (NET-NEW CODE) — denylist case/Unicode normalization [SEC-02]
**What's wrong:** Base-name comparison is exact and case-sensitive (`switch base { case ".bashrc" ... }`, `sandbox.go:134`). On case-insensitive macOS/Windows volumes, `.BASHRC` / `.Bashrc` resolve to the same file but bypass the denylist. No Unicode NFC/NFD normalization either.
**Fix:** Normalize the base name and the relative-slash path with `strings.ToLower` (the denylist names are all ASCII, so `ToLower` is safe and sufficient for case) and `golang.org/x/text/unicode/norm`-free NFC via... — note: stdlib has no NFC normalizer. Recommendation: apply `unicode`-aware comparison only if a vendored `x/text` is already in go.mod; otherwise do case-folding (`strings.ToLower`) which closes the high-value macOS/Windows case vector, and document the residual NFC/NFD risk as LOW (it requires a filesystem that stores a decomposed form AND a denylist entry containing a composable character — none of the protected names contain non-ASCII). **Confirm go.mod for `golang.org/x/text` before recommending NFC.** `[ASSUMED]` that `x/text` is not currently a dependency — planner must verify.
**Test:** denylist-bypass cases `.BASHRC`, `.Bashrc`, `.SSH/authorized_keys` → 403.

### GAP 3 (NET-NEW TEST) — write-path symlink-escape test [SEC-01]
**What's wrong:** `TestSandbox_SymlinkEscapeBlocked` (`sandbox_test.go:217`) only tests `sb.Open` (read). No test asserts `WriteFileAtomic`/`Rename`/`Mkdir` through an escaping symlink fails.
**Fix:** Add `TestSandbox_WritePathSymlinkEscapeBlocked` mirroring the read test: create `escape -> <outside dir>` inside root, attempt `sb.WriteFileAtomic("escape/pwned", ...)`, `sb.Rename("a.txt","escape/pwned")`, `sb.Mkdir("escape/sub")` — each must return a non-nil error and create nothing outside root. Add a handler-level test asserting the HTTP write returns 403 (path-validation/`os.Root` rejection maps via `writeWriteError`). The protection holds (all methods use `root.*`); the test makes SC1 explicit.

### GAP 4 (NET-NEW DOC) — SECURITY artifact [SEC-04]
**What's wrong:** No `127-SECURITY.md` (or similar) exists. Precedent: `83-SECURITY.md`, `74-SECURITY.md`, `60-SECURITY.md` exist in prior phases — follow that naming/structure.
**Fix:** Author `.planning/phases/127-web-share-write-security-hardening/127-SECURITY.md` documenting the capability-escalation audit (the table below), with a per-surface enforcement matrix (daemon socket / webserver / remote proxy), the denylist threat model, and the residual-risk register (microscopic TOCTOU window in If-Match re-stat, documented in `sandbox.go:198-203`; NFC/NFD denylist residual). Commit it.

### GAP 5 (NET-NEW TEST) — CSRF Origin-mismatch e2e scenario [SEC-07]
**What's wrong:** Origin-mismatch is unit-tested (`capability_test.go:690`) but `files-write.spec.ts` has no Origin-mismatch browser cell.
**Fix:** Add a Playwright scenario to `files-write.spec.ts`: issue a `files.write` cap, then `request.put('/api/files/write?...', { headers: { Origin: 'https://evil.example.com' } })` and assert 403. (Use the standalone `APIRequestContext` pattern already at spec line 570.)

### SMALL gaps (fuzz seed top-ups + optional data-integrity test) [SEC-06, SEC-05]
- **SEC-06:** Add `f.Add(".BASHRC")`, `f.Add(".Bashrc")`, and a multipart-style `f.Add("../../../etc/passwd")` filename seed to `FuzzSandboxWrite`. The corpus already covers rename-dest traversal and the canonical denylist-bypass paths; these top-ups finalize it against the case-variation vector introduced by the GAP 2 fix. Merge gate stays `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` → 0 crashes.
- **SEC-05 (phase requirement, not in CONTEXT locked list):** Existing `TestWriteFileAtomic_ConcurrentReadNeverPartial` + `TestWriteFileAtomic_ValidatorRecheck` cover most of it. If SEC-05 is in scope, add `TestWrite_TwoWritersIfMatchRace` (two goroutines, both with the same If-Match validator; exactly one 200, one 412; file is one writer's complete content, never interleaved) and an interrupted-write test (force a write error mid-`WriteFileAtomic`; assert original file content preserved and no `.agenthub-tmp-*` left behind). Confirm SEC-05 scope at plan time — CONTEXT.md's locked list (SC1-SC5) maps to SEC-01,02,03,04,07; SEC-05 and SEC-06 are phase requirements but SC5-as-listed is the e2e (SEC-07).

## Capability-Escalation Audit (source for the SECURITY artifact)

| Surface | Reachable without `files.write`? | Enforcement | Verdict |
|---------|----------------------------------|-------------|---------|
| Webserver `PUT/POST/DELETE /api/files/{write,upload,delete,rename,mkdir}` | No | `requireFilesWrite` → `requireCapability` (HMAC+SID+grant+session-enabled) → `HasPerm(files.write)` → Origin check | SECURE (`server.go:512-521`, `capability_mw.go`) |
| Webserver `HEAD /api/files/write` (canWrite probe) | No | Same `requireFilesWrite`; handler short-circuits HEAD after middleware | SECURE — returns 403 without cap (`files_routes_test.go:286`) |
| Daemon Unix socket / named pipe write routes | **Yes (by design)** | None — loopback trust boundary (WEB-01 precedent); only in-process GUI/TUI reach it | ACCEPTED RISK — document explicitly. Not a web-share surface. |
| Remote proxy `/api/files/remote/{sid}/{write,...}` | No | Proxy strips caller `?cap`, injects stored token; **remote peer's** `requireFilesWrite` enforces cap + Origin | SECURE (`remote_files.go:197-226`) |
| Cross-session (`files.write` on session A used for session B) | No | `requireCapability` rejects `claims.SID != pathID` (403) | SECURE — SID-scoped tokens |
| Token tampering / wrong perm string (`"no-files.write"`) | No | `HasPerm` whole-token `strings.Split` (not `strings.Contains`) | SECURE (`capability.go:51`) |
| Viewer token default | N/A | Viewer token is bare `"read"`; `files.write` added to owner token ONLY when `filesWriteEnabledFor(sid)` | SECURE — opt-in (`api.go:1070-1079`) |

**Key insight:** the only surface that reaches write endpoints without a `files.write` perm is the daemon local socket — and that is the intentional loopback-trust boundary inherited from the v3.4 read surface (WEB-01). The SECURITY artifact must state this explicitly as an accepted, documented risk, not a finding.

## Common Pitfalls

### Pitfall 1: "Re-implementing" already-correct code
**What goes wrong:** Treating this like a build phase and rewriting `WriteFileAtomic`/`Rename`/denylist.
**How to avoid:** The code is correct and tested. Touch `denylistCheck` ONLY (for GAPS 1-2). Everything else is additive tests + one e2e + one doc.
**Warning sign:** A plan task that edits `write.go` handlers or the `Rename`/`WriteFileAtomic` bodies.

### Pitfall 2: Hardcoding a macOS path in the config-dir denylist test
**What goes wrong:** A test that asserts `~/Library/Application Support/agenthub` fails on Linux CI.
**How to avoid:** Derive from `os.UserConfigDir()` in both the fix and the test; guard platform-specific assertions with `runtime.GOOS`.

### Pitfall 3: Over-reaching on Unicode normalization
**What goes wrong:** Pulling in `golang.org/x/text` (a new dependency) for NFC normalization the threat model barely needs — all protected names are ASCII.
**How to avoid:** Case-fold with `strings.ToLower` (stdlib, ASCII-safe). Document NFC/NFD as a LOW residual risk. Only add `x/text` if it is ALREADY in go.mod.

### Pitfall 4: Confusing "unit-tested" with "e2e-tested" for CSRF
**What goes wrong:** Concluding SEC-07 CSRF is unmet because the e2e doesn't have it, and re-implementing the Origin check.
**How to avoid:** The Origin check IS implemented and unit-tested for all 5 routes. SEC-07 needs only the e2e *cell* added, not new middleware.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain (fuzzing) | FuzzSandboxWrite, all Go tests | ✓ | go1.26.4 (verified) | — |
| Playwright + browsers | SEC-07 e2e scenario | ✓ (Phase 120/125 established) | as vendored | — |
| Writable `$HOME` for denylist tests | SEC-02 tests use a fake HOME via env/temp | ✓ | — | tests set HOME to a tempdir (see `write_test.go:478` pattern) |

No missing dependencies.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + native fuzzing (go1.26.4); Playwright for e2e |
| Config file | none for Go; `frontend/playwright.config.ts` for e2e |
| Quick run command | `go test ./internal/files/... ./internal/webserver/... ./internal/daemon/...` |
| Full suite command | above + `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` + `cd frontend && pnpm playwright test files-write.spec.ts` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SEC-01 | write-path symlink escape → error/403 | unit | `go test ./internal/files/ -run TestSandbox_WritePathSymlinkEscapeBlocked` | ❌ Wave 0 |
| SEC-02 | denylist: macOS config dir → 403 | unit | `go test ./internal/files/ -run TestDenylist` | ⚠️ exists, extend |
| SEC-02 | denylist: case `.BASHRC` → 403 | unit | `go test ./internal/files/ -run TestDenylist` | ❌ Wave 0 |
| SEC-03 | upload filename `../`; over-cap 413 | unit | `go test ./internal/files/ -run TestUpload` | ✅ |
| SEC-04 | escalation audit; SECURITY artifact | doc + existing tests | `go test ./internal/webserver/ -run TestFilesRoutes` | ⚠️ artifact ❌ |
| SEC-05 | two-writer If-Match race; original preserved on failure | unit | `go test ./internal/files/ -run TestWrite` | ⚠️ partial, extend |
| SEC-06 | FuzzSandboxWrite 0 crashes (with case/unicode seeds) | fuzz | `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` | ⚠️ exists, top-up seeds |
| SEC-07 | e2e: write-OK / 403-no-cap / Origin-mismatch | e2e | `pnpm playwright test files-write.spec.ts` | ⚠️ scenarios 1,3 ✅; Origin-mismatch ❌ |

### Sampling Rate
- **Per task commit:** `go test ./internal/files/... ./internal/webserver/...`
- **Per wave merge:** full suite incl. fuzz + Playwright `files-write.spec.ts`
- **Phase gate:** full suite green; `FuzzSandboxWrite` 0 crashes; SECURITY artifact committed.

### Wave 0 Gaps
- [ ] `internal/files/sandbox_test.go` — `TestSandbox_WritePathSymlinkEscapeBlocked` (SEC-01)
- [ ] `internal/files/write_test.go` — extend `TestDenylist_HomeRooted` for daemon config dir + case-variation (SEC-02); optional two-writer-race + interrupted-write tests (SEC-05)
- [ ] `internal/files/sandbox_test.go` — `FuzzSandboxWrite` seed top-ups (`.BASHRC`, multipart-style filename) (SEC-06)
- [ ] `frontend/e2e/files-write.spec.ts` — Origin-mismatch scenario (SEC-07)
- [ ] `.planning/phases/127-web-share-write-security-hardening/127-SECURITY.md` — capability-escalation audit artifact (SEC-04)
- Production code: `internal/files/sandbox.go` `denylistCheck` only — config-dir derivation + case-fold (SEC-02)

## Security Domain

### Applicable ASVS Categories (per CLAUDE.md: security = ASVS)

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Architecture | yes | Sandbox boundary (`os.OpenRoot`) + capability tier separation; documented in SECURITY artifact |
| V2 Authentication | yes | HMAC-signed capability tokens (`requireCapability`); SID + grant + session-enabled checks |
| V4 Access Control | yes | `files.write` perm gate (`HasPerm`); cross-session SID scoping; opt-in token issuance |
| V5 Validation/Sanitization | yes | `validateRelativePath` (traversal/ADS/device/UNC/null), `filepath.Base` on upload filename, denylist |
| V12 Files & Resources | yes | `MaxBytesReader` 50 MiB cap before `ParseMultipartForm`; atomic temp+rename; no archive extraction (no zip-slip surface) |
| V13 API/Web Service | yes | CSRF Origin check on state-changing verbs; method-prefix 405 routing |
| V6 Cryptography | partial | `crypto/rand` for temp-file suffix (let-it-crash on CSPRNG failure, `sandbox.go:232`) — no hand-rolled crypto |

### Known Threat Patterns for Go sandboxed-write-over-web-share

| Pattern | STRIDE | Standard Mitigation | Status |
|---------|--------|---------------------|--------|
| Path traversal on write/rename destination | Tampering / Elevation | `validateAndClean` on BOTH paths + native `root.Rename` | COVERED |
| Symlink-escape on write (write-TOCTOU) | Tampering / Elevation | `os.OpenRoot` terminal boundary | COVERED (test GAP — SEC-01) |
| Shell-RC / SSH-key / agent-config overwrite | Elevation (persistence) | `denylistCheck` on all write methods | COVERED for `.config` Linux; GAP macOS config dir + case (SEC-02) |
| Multipart filename `../` injection | Tampering | `filepath.Base` + name guard | COVERED |
| Upload DoS (disk/memory exhaustion) | DoS | `MaxBytesReader` before parse | COVERED |
| Zip-slip via archive extraction | Tampering | No extraction implemented | COVERED (by absence) |
| CSRF from a tailnet browser | Tampering / Spoofing | Origin byte-match on write verbs | COVERED (e2e cell GAP — SEC-07) |
| Capability escalation (no-perm / cross-session / token tamper) | Elevation | `HasPerm` whole-token + SID scope + opt-in issuance | COVERED (artifact GAP — SEC-04) |
| Concurrent-write lost update / partial file | Tampering / data integrity | If-Match 412 + atomic temp+rename + validator re-stat | COVERED (explicit two-writer test GAP — SEC-05) |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `golang.org/x/text` is NOT currently in go.mod, so NFC normalization should be avoided in favor of ASCII case-fold | GAP 2 | If it IS present, NFC could be added cheaply; if absent and added anyway, introduces a new dependency contrary to vendoring discipline. Planner must verify go.mod. |
| A2 | SEC-05 (data-integrity tests) is in phase scope even though CONTEXT.md's locked SC list maps to SEC-01,02,03,04,07 | SEC-05 row | If SEC-05 is descoped, the two-writer-race/interrupted-write tests are optional. Confirm at plan time. |
| A3 | The microscopic stat→rename TOCTOU window in If-Match re-check (documented `sandbox.go:198-203`) is an accepted residual, not a phase deliverable | residual-risk register | If product wants it eliminated, requires kernel `renameat2` support unavailable in Go stdlib — would be a deferred item, not this phase. |

## Open Questions

1. **SECURITY artifact filename/location**
   - What we know: prior phases use `NN-SECURITY.md` in the phase dir (`83-SECURITY.md`, `74-SECURITY.md`, `60-SECURITY.md`); CONTEXT says "under .planning/".
   - What's unclear: phase-dir `127-SECURITY.md` vs a milestone-level artifact.
   - Recommendation: `.planning/phases/127-web-share-write-security-hardening/127-SECURITY.md` (matches precedent + CONTEXT).

2. **gsd-security-auditor pattern (Claude's discretion)**
   - What we know: CONTEXT leaves this to discretion.
   - Recommendation: the audit is a finite, fully-enumerated set of code paths (table above); a structured manual audit written into `127-SECURITY.md` is sufficient. No need to spawn a separate auditor agent unless the planner wants an independent review pass.

3. **Unicode NFC/NFD scope for the denylist**
   - What we know: all protected names are ASCII; the realistic bypass is case (macOS/Windows), not decomposition.
   - Recommendation: ship ASCII case-fold; document NFC/NFD as LOW residual. Revisit only if a non-ASCII protected name is ever added.

## Recommended Plan Decomposition

Aligns with the objective's suggested ordering; sized to the actual gaps:

- **Plan 127-01 — Denylist hardening + tests (SEC-02, SEC-06 seeds).** Fix `denylistCheck` config-dir derivation (`os.UserConfigDir`-based) + ASCII case-fold. Extend `TestDenylist_*` for macOS config dir + `.BASHRC`. Add fuzz seeds. *Only net-new production code in the phase.*
- **Plan 127-02 — Write-path symlink test + fuzz finalize (SEC-01, SEC-06).** `TestSandbox_WritePathSymlinkEscapeBlocked` (unit + handler 403). Confirm `FuzzSandboxWrite` 0 crashes with new seeds.
- **Plan 127-03 — Capability-escalation audit + SECURITY artifact (SEC-04, + optional SEC-05 tests).** Write `127-SECURITY.md` from the audit table; add two-writer-race + interrupted-write tests if SEC-05 in scope. Commit artifact.
- **Plan 127-04 — Web-share e2e gap (SEC-07).** Add Origin-mismatch scenario to `files-write.spec.ts`; confirm scenarios 1/3 still green cross-browser.

## State of the Art

| Old Approach | Current Approach (in this codebase) | When | Impact |
|--------------|-------------------------------------|------|--------|
| Hand-rolled `os.Rename(absOld,absNew)` with prefix-check (PITFALLS.md §1 workaround) | Native `root.Rename(oldClean,newClean)` — TOCTOU-safe, both paths relative to root | Go 1.25+ added `Root.Rename`/`Root.MkdirAll` | The PITFALLS.md "Rename not on os.Root" caveat is OBSOLETE for this codebase — the native methods exist and are used (`sandbox.go:340,380`). Do not plan the workaround. |

**Deprecated/outdated guidance:**
- PITFALLS.md §1/§Integration "`os.Root.Rename` does not exist / use `os.Rename` with double-validated abs paths" — superseded; the code uses native `root.Rename`. The double-validation (both paths) is still done, but via `validateAndClean` before native rename, not via the abs-path workaround.

## Sources

### Primary (HIGH confidence — live source, verified this session)
- `internal/files/sandbox.go` — denylist (`denylistCheck` :96-148), `WriteFileAtomic` :214, native `Rename` :320, `Mkdir`/`MkdirAll`/`Delete`, `validateRelativePath` :452
- `internal/files/write.go` — handlers, `MaxBytesReader` :89/:131, `filepath.Base` upload guard :158, `writeWriteError` :288
- `internal/files/write_test.go` — `TestDenylist_HomeRooted` :478, `TestWriteFileAtomic_ConcurrentReadNeverPartial` :145, `TestWrite_IfMatch_*`, `TestUpload_*`
- `internal/files/sandbox_test.go` — `TestSandbox_SymlinkEscapeBlocked` :217 (read-only), `FuzzSandboxWrite` :367
- `internal/webserver/capability_mw.go` — `requireFilesWrite`, `originAllowedForWrite`, `requireCapability`
- `internal/webserver/capability_test.go` — Origin-mismatch unit tests :690-739
- `internal/webserver/files_routes_test.go` — 403/405/HEAD-probe tests
- `internal/webserver/server.go` :502-521 — route registration
- `internal/daemon/api.go` :152-175 (routes), :1023-1079 (`issueCapabilitiesForSession`)
- `internal/daemon/remote_files.go` :175-226 — `proxyRemoteFiles` cap-strip + body-forward
- `internal/daemon/engine.go` :60-115 — `daemonConfigDir` via `os.UserConfigDir`
- `frontend/e2e/files-write.spec.ts` — scenarios 1-14
- **Host verification:** `os.UserConfigDir()` = `/Users/ken/Library/Application Support` on this macOS host (ran a one-off Go program); `go version` = go1.26.4
- Prior SECURITY artifacts for structure: `.planning/milestones/v3.0-phases/83-settings-ui-alignment/83-SECURITY.md`, `v2.0-phases/74-.../74-SECURITY.md`, `v1.11-phases/60-.../60-SECURITY.md`

### Secondary (MEDIUM confidence)
- `.planning/research/PITFALLS.md`, `ARCHITECTURE.md`, `SUMMARY.md` — milestone research (note: `os.Root.Rename` gap claim is now obsolete vs the shipped code)
- `.planning/REQUIREMENTS.md` SEC-01..SEC-07

## Metadata

**Confidence breakdown:**
- Already-covered inventory: HIGH — every claim cites a specific source line read this session.
- macOS config-dir gap: HIGH — `os.UserConfigDir()` value verified on host; denylist hardcodes `.config/agenthub/`.
- Case/Unicode gap: HIGH for case (exact-match `switch` confirmed); MEDIUM for the recommendation to skip NFC (depends on go.mod — flagged A1).
- Symlink write-test gap: HIGH — grep confirms only read-path symlink test exists.
- e2e Origin-mismatch gap: HIGH — spec grep confirms no Origin scenario.

**Research date:** 2026-06-14
**Valid until:** 2026-07-14 (stable — internal codebase, no fast-moving external deps)
