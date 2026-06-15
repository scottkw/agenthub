---
phase: 127
slug: web-share-write-security-hardening
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-15
---

# Phase 127 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (internal/files fuzz + denylist; internal/webserver CSRF/cap) + Playwright (web-share Origin-mismatch e2e cell) |
| **Config file** | none (Go native); existing playwright config |
| **Quick run command** | `go test ./internal/files/... ./internal/webserver/...` |
| **Full suite command** | `go test -race ./internal/files/... ./internal/webserver/... && go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` + `pnpm exec playwright test files-write` |
| **Estimated runtime** | ~30-90s + 60s fuzz |

---

## Sampling Rate

- **After every task commit:** relevant package `go test`
- **After every plan wave:** `go test -race ./internal/files/... ./internal/webserver/...`
- **Before `/gsd:verify-work`:** full suite green; `FuzzSandboxWrite -fuzztime=60s` zero crashes; SECURITY artifact committed
- **Max feedback latency:** 90s (fuzz excluded)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Test File · Function | Automated Command | Status |
|---------|------|------|-------------|------------|-----------------|-----------|----------------------|-------------------|--------|
| 127-01 | 127-01-PLAN.md | 1 | SEC-02 | T-127-01/02 denylist bypass | `.BASHRC`/`.Bashrc`/`.SSH/...` + macOS `os.UserConfigDir()` config dir → `ErrProtectedSystemFile` | unit | `internal/files/write_test.go` · `TestDenylist_CaseVariation`, `TestDenylist_DaemonConfigDir` | `go test ./internal/files/ -run 'TestDenylist_CaseVariation\|TestDenylist_DaemonConfigDir' -count=1` | ✅ green |
| 127-02 | 127-02-PLAN.md | 2 | SEC-01 | T-127-03 symlink escape | write/rename/mkdir through escaping symlink → error; sentinel unmodified | unit | `internal/files/sandbox_test.go` · `TestSandbox_WritePathSymlinkEscapeBlocked` | `go test ./internal/files/ -run TestSandbox_WritePathSymlinkEscapeBlocked -count=1` | ✅ green |
| 127-02 | 127-02-PLAN.md | 2 | SEC-03 | T-127-04/05 upload abuse | `../` filename sanitized; >50 MiB → 413 before ParseMultipartForm; empty/dot/dotdot → 400 | unit | `internal/files/handler_test.go` · `TestHandlerUpload_FilenameSanitized`, `TestHandlerUpload_OverCap413`, `TestHandlerWrite_OverCap413`, `TestHandlerUpload_{Empty,Dot,DotDot}Filename400` | `go test ./internal/files/ -run 'TestHandlerUpload_FilenameSanitized\|TestHandlerUpload_OverCap413\|TestHandlerWrite_OverCap413\|TestHandlerUpload_.*Filename400' -count=1` | ✅ green |
| 127-01/02 | 127-01/02-PLAN.md | 2 | SEC-06 | T-127-03/04 fuzz | rename-dest traversal + denylist-bypass + multipart-filename seeds; 0 crashes, no escaped temp | fuzz | `internal/files/sandbox_test.go` · `FuzzSandboxWrite` (seeds L517-528; rename-dest at L560) | `go test ./internal/files/ -run '^$' -fuzz='FuzzSandboxWrite$' -fuzztime=8s -count=1` | ✅ green |
| 127-03 | 127-03-PLAN.md | 3 | SEC-04 | T-127-07/09 escalation | every write route 403 when `files.write` missing; SID-scoped; whole-token HasPerm; daemon loopback = accepted risk | unit + doc | `internal/webserver/capability_test.go` · `TestRequireFilesWrite` ("403 when files.write missing" × 5 routes); audit in `127-SECURITY.md` | `go test ./internal/webserver/ -run TestRequireFilesWrite -count=1` | ✅ green |
| 127-03 | 127-03-PLAN.md | 3 | SEC-05 | T-127-08 data integrity | two concurrent writers → exactly one win + one 412; interrupted write preserves original, no temp sibling | unit (-race) | `internal/files/write_test.go` · `TestWrite_TwoWritersIfMatchRace`, `TestWrite_InterruptedWritePreservesOriginal` | `go test -race ./internal/files/ -run 'TestWrite_TwoWritersIfMatchRace\|TestWrite_InterruptedWritePreservesOriginal' -count=1` | ✅ green |
| 127-04 | 127-04-PLAN.md | 4 | SEC-07 | T-127-06 CSRF | valid `files.write` cap + mismatched Origin → 403 (Origin check, not cap failure) | unit + e2e | `internal/webserver/capability_test.go` · `TestRequireFilesWrite` ("403 on mismatched Origin" × 5 routes) **[deterministic Go]**; `frontend/e2e/files-write.spec.ts:600` SEC-07 cell **[e2e]** | `go test ./internal/webserver/ -run TestRequireFilesWrite -count=1` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> **Audit note (2026-06-14):** SEC-07 CSRF rejection is covered by BOTH a deterministic Go-native test (the "403 on mismatched Origin" subtest in `TestRequireFilesWrite`, run green here across all 5 write routes) AND the Playwright e2e cell — coverage does not depend solely on the cross-browser Playwright run.

---

## Wave 0 Requirements

- [x] Denylist case-fold test (.BASHRC/.BashRC → 403) + macOS daemon-config-dir (`os.UserConfigDir()` path) → 403
- [x] Write-path symlink-escape test (write/rename to symlink-escaping target → 403)
- [x] FuzzSandboxWrite corpus top-up (rename-dest traversal, denylist-bypass, multipart FileHeader.Filename ../ )
- [x] Capability escalation audit table → committed SECURITY artifact under .planning/
- [x] Playwright web-share Origin-mismatch e2e cell

*Framework present — no install.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| (likely none — this phase is automated audit + tests + doc) | — | — | — |

*The escalation audit is a code-read documented in the SECURITY artifact; all enforcement is unit/e2e-testable.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-14 (retroactive audit)

---

## Validation Audit 2026-06-14

| Metric | Count |
|--------|-------|
| Requirements total | 7 (SEC-01..07) |
| Covered (automated) | 7 |
| Manual-only | 0 |
| Missing / Partial | 0 |
| Gaps | 0 |
| Resolved | 0 (no new tests needed) |
| Escalated | 0 |

Retroactive AUDIT-ONLY Nyquist validation of archived/shipped Phase 127: every SEC-01..07 requirement maps to a delivered automated test that was independently spot-run GREEN on `main` (`go test -count=1`, incl. `-race` for SEC-05 and an 8s `-fuzz` smoke for SEC-06); SEC-07 CSRF is covered by a deterministic Go test in addition to the Playwright e2e cell. No test/impl files were created or modified. Compliance flipped to `nyquist_compliant: true`.
