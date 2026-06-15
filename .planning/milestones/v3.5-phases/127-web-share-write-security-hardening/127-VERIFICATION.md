---
phase: 127-web-share-write-security-hardening
verified: 2026-06-14T18:00:00Z
status: passed
score: 7/7
overrides_applied: 0
re_verification: false
---

# Phase 127: Web-Share Write Security Hardening Verification Report

**Phase Goal:** The web-share write surface has been security-audited end-to-end: symlink escapes return 403, the shell-RC denylist blocks all known bypass vectors, upload abuse is covered, capability escalation is impossible, concurrent-write races leave no partial files, and a Playwright e2e confirms the full web-share write flow with and without the `files.write` cap.

**Verified:** 2026-06-14T18:00:00Z
**Status:** passed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SEC-01: Write/rename/mkdir through symlink escaping sandbox root returns non-nil error; nothing created outside root | VERIFIED | `TestSandbox_WritePathSymlinkEscapeBlocked` PASS (exit 0); all three methods (WriteFileAtomic, Rename, Mkdir) return error; sentinel unmodified; positive control confirms outside is writable |
| 2 | SEC-02: Denylist blocks ~/.bashrc, ~/.ssh/authorized_keys, ~/.claude/CLAUDE.md, daemon config dir; not bypassable by case variation | VERIFIED | `TestDenylist_CaseVariation` PASS (.BASHRC, .Bashrc, .SSH/authorized_keys, .Claude/CLAUDE.md all return ErrProtectedSystemFile); `TestDenylist_DaemonConfigDir` PASS (os.UserConfigDir()/agenthub/settings.json blocked); denylistCheck uses strings.ToLower at lines 137 and 146 of sandbox.go; os.UserConfigDir appears 3 times in sandbox.go |
| 3 | SEC-03: Upload filename injection sanitized; over-cap (>50 MiB) rejected before ParseMultipartForm | VERIFIED | `TestHandlerUpload_FilenameSanitized` PASS; `TestHandlerUpload_OverCap413` PASS (413, file not on disk); `TestHandlerWrite_OverCap413` PASS (413); `TestHandlerUpload_EmptyFilename400` / `DotFilename400` / `DotDotFilename400` all PASS |
| 4 | SEC-04: SECURITY artifact committed covering per-surface capability-escalation audit (webserver requireFilesWrite+HasPerm, auth-less daemon socket as accepted risk, remote proxy cap-strip, cross-session scoping) | VERIFIED | 127-SECURITY.md exists (170 lines); covers requireFilesWrite, HasPerm, daemon loopback (accepted risk AR-127-01), remote proxy cap-strip, cross-session SID scoping, CSRF Origin inversion; all 7 surfaces enumerated; threats_open: 0 |
| 5 | SEC-05: Concurrent-write race yields exactly one success + one ErrPreconditionFailed; interrupted write leaves original intact and no temp sibling | VERIFIED | `TestWrite_TwoWritersIfMatchRace` PASS under -race; `TestWrite_InterruptedWritePreservesOriginal` PASS under -race |
| 6 | SEC-06: FuzzSandboxWrite with finalized corpus (case-variation + multipart seeds) reports zero crashes | VERIFIED | Executor ran 60s fuzz gate (105,964 executions / 72,377 executions second run) — 0 crashes both runs; case-variation seeds (.BASHRC, .Bashrc) confirmed in sandbox_test.go lines 525-526; no crash corpus files under testdata/fuzz (directory does not exist) |
| 7 | SEC-07: Playwright e2e — viewer w/ cap writes 200, without cap gets 403, CSRF Origin-mismatch with valid cap gets 403 | VERIFIED | SEC-07 cell at files-write.spec.ts:600 uses `env.writeCap` (valid cap) + Origin: https://evil.example.com; asserts 403 (CSRF rejection, not cap failure); executor confirmed 51 tests green across Chromium/Firefox/WebKit |

**Score:** 7/7 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/files/sandbox.go` | denylistCheck with case-fold + os.UserConfigDir | VERIFIED | strings.ToLower at lines 137, 146; os.UserConfigDir present 3 times; no internal/daemon import |
| `internal/files/write_test.go` | TestDenylist_DaemonConfigDir + case-variation + SEC-05 data-integrity tests | VERIFIED | TestDenylist_CaseVariation at line 613; TestDenylist_DaemonConfigDir at line 661; TestWrite_TwoWritersIfMatchRace at line 1153; TestWrite_InterruptedWritePreservesOriginal at line 1253 |
| `internal/files/sandbox_test.go` | TestSandbox_WritePathSymlinkEscapeBlocked + FuzzSandboxWrite seeds | VERIFIED | TestSandbox_WritePathSymlinkEscapeBlocked present (count=2 for declaration+call); .BASHRC seed at line 525 |
| `.planning/phases/127-web-share-write-security-hardening/127-SECURITY.md` | Capability-escalation audit >= 60 lines | VERIFIED | 170 lines; covers all 7 surfaces; STRIDE register 10 threats/0 open; accepted risks log 3 entries; ASVS L1 mapping |
| `frontend/e2e/files-write.spec.ts` | CSRF Origin-mismatch e2e cell | VERIFIED | "evil.example.com" appears 3 times (comment, test name, header value); uses env.writeCap for CSRF proof |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| denylistCheck | os.UserConfigDir | local derivation in sandbox.go | VERIFIED | grep finds 3 occurrences; no internal/daemon import |
| WriteFileAtomic/Rename/Mkdir/Delete | denylistCheck | single chokepoint | VERIFIED | All four methods confirmed to call denylistCheck (pre-existing wiring, unchanged) |
| TestSandbox_WritePathSymlinkEscapeBlocked | os.OpenRoot boundary | symlink → error, nothing outside root | VERIFIED | Test PASS confirms boundary holds for all 3 write methods |
| SEC-07 e2e cell | originAllowedForWrite (capability_mw.go) | PUT with valid cap + evil.example.com Origin → 403 | VERIFIED | Test cell uses env.writeCap; asserts 403 proving CSRF check not cap check |
| TestWrite_TwoWritersIfMatchRace | WriteFileAtomic validator re-check | two goroutines, same If-Match validator | VERIFIED | PASS under -race |

---

### Data-Flow Trace (Level 4)

Not applicable — this phase is test-only and audit-documentation. No new dynamic rendering artifacts introduced.

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| SEC-01: Write-path symlink escape blocked | `go test ./internal/files/ -run TestSandbox_WritePathSymlinkEscapeBlocked -count=1 -v` | exit 0, PASS | PASS |
| SEC-02: Denylist case-fold + daemon config dir | `go test ./internal/files/ -run 'TestDenylist' -count=1 -v` | exit 0, all subtests PASS (TestDenylist_HomeRooted, TestDenylist_NonHomeRootedUnaffected, TestDenylist_CaseVariation, TestDenylist_DaemonConfigDir) | PASS |
| SEC-03: Over-cap 413 + filename sanitization | `go test ./internal/files/ -run 'TestHandlerUpload_OverCap413\|TestHandlerWrite_OverCap413\|TestHandlerUpload_FilenameSanitized' -count=1 -v` | exit 0, all PASS | PASS |
| SEC-05: Concurrent write race + interrupted write | `go test -race ./internal/files/ -run 'TestWrite_TwoWritersIfMatchRace\|TestWrite_InterruptedWritePreservesOriginal' -count=1 -v` | exit 0, both PASS | PASS |
| Full internal/files suite with race detector | `go test -race ./internal/files/ -count=1` | exit 0, ok in 6.456s | PASS |
| Build clean (scoped to ./internal/...) | `go build ./internal/...` | exit 0 | PASS |

Note: `go build ./...` fails due to a stray untracked `security-review/` directory at the repo root that pre-dates Phase 127 and is not tracked by git. The phase notes explicitly document this and scope the build check to `./internal/...`. This is a pre-existing issue, not introduced by Phase 127.

---

### Probe Execution

No `scripts/*/tests/probe-*.sh` files declared or found for this phase.

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SEC-01 | 127-02-PLAN.md | Write-path symlink-escape → 403 | SATISFIED | TestSandbox_WritePathSymlinkEscapeBlocked PASS |
| SEC-02 | 127-01-PLAN.md | Shell-RC denylist + daemon config dir + case variation | SATISFIED | TestDenylist_CaseVariation + TestDenylist_DaemonConfigDir PASS; strings.ToLower in sandbox.go |
| SEC-03 | 127-02-PLAN.md | Upload abuse: filename injection sanitized, over-cap rejected | SATISFIED | TestHandlerUpload_FilenameSanitized + OverCap413 + EmptyFilename400 PASS |
| SEC-04 | 127-03-PLAN.md | Capability-escalation audit documented in SECURITY artifact | SATISFIED | 127-SECURITY.md exists, 170 lines, all surfaces covered, threats_open: 0 |
| SEC-05 | 127-03-PLAN.md | Concurrent-write and interrupted-write leave no partial file | SATISFIED | TestWrite_TwoWritersIfMatchRace + TestWrite_InterruptedWritePreservesOriginal PASS under -race |
| SEC-06 | 127-01-PLAN.md + 127-02-PLAN.md | FuzzSandboxWrite corpus finalized, 0 crashes | SATISFIED | Executor confirmed 0 crashes in two 60s runs; case-variation seeds present in sandbox_test.go |
| SEC-07 | 127-04-PLAN.md | Playwright e2e: cap OK=200, no-cap=403, CSRF Origin-mismatch=403 | SATISFIED | SEC-07 cell at files-write.spec.ts:600; executor confirmed 51 tests green cross-browser |

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | — | — | — |

No TBD/FIXME/XXX markers found in any modified file. No placeholder returns, empty handlers, or stub implementations found.

---

### Human Verification Required

None. All success criteria are verifiable programmatically or via committed test results. The Playwright cross-browser run (51 tests) was confirmed by the executor; re-running it is expensive and not required per the phase notes — the committed SEC-07 cell existence and the executor's documented cross-browser pass are sufficient evidence.

---

### Gaps Summary

No gaps. All 7 SEC-ID requirements are satisfied with running tests on `main`.

**Key findings:**
- All 7 SEC requirements satisfied with test evidence on the live codebase
- All 7 ROADMAP success criteria verified
- 4 net-new production code lines in sandbox.go (strings.ToLower + os.UserConfigDir); all other changes are test-only or documentation
- The stray `security-review/` directory at repo root breaks `go build ./...` but is untracked, pre-dates Phase 127, and is noted in the phase notes as a known issue; `go build ./internal/...` is clean
- Daemon loopback socket correctly documented as ACCEPTED RISK (WEB-01 loopback trust), not a gap

---

_Verified: 2026-06-14T18:00:00Z_
_Verifier: Claude (gsd-verifier)_
