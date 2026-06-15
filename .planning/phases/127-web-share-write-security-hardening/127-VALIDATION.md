---
phase: 127
slug: web-share-write-security-hardening
status: draft
nyquist_compliant: false
wave_0_complete: false
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

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (planner fills) | — | 1 | SEC-02 | denylist bypass | .BASHRC + macOS config dir → 403 | unit | `go test ./internal/files/...` | ❌ W0 | ⬜ pending |
| (planner fills) | — | 2 | SEC-01/06 | symlink/fuzz | write-path symlink escape → 403; fuzz zero crashes | unit/fuzz | `go test ./internal/files/...` | ❌ W0 | ⬜ pending |
| (planner fills) | — | 3 | SEC-04/05 | escalation | audit table; concurrent write no partial | doc/unit | `go test ./internal/...` | ❌ W0 | ⬜ pending |
| (planner fills) | — | 4 | SEC-07 | CSRF | Origin-mismatch web-share → 403 | playwright | `pnpm exec playwright test files-write` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Denylist case-fold test (.BASHRC/.BashRC → 403) + macOS daemon-config-dir (`os.UserConfigDir()` path) → 403
- [ ] Write-path symlink-escape test (write/rename to symlink-escaping target → 403)
- [ ] FuzzSandboxWrite corpus top-up (rename-dest traversal, denylist-bypass, multipart FileHeader.Filename ../ )
- [ ] Capability escalation audit table → committed SECURITY artifact under .planning/
- [ ] Playwright web-share Origin-mismatch e2e cell

*Framework present — no install.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| (likely none — this phase is automated audit + tests + doc) | — | — | — |

*The escalation audit is a code-read documented in the SECURITY artifact; all enforcement is unit/e2e-testable.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
