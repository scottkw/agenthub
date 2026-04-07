---
phase: 54
slug: tailscale-onboarding-enhancement
status: audited
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-07
audited: 2026-04-07 (re-audited)
---

# Phase 54 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.1.0 / go test |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm test && cd /Users/ken/dev/agenthub && go test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test`
- **After every plan wave:** Run `cd frontend && pnpm test && go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 54-01-01 | 01 | 1 | TS-01 | unit (raw) | `cd frontend && pnpm test` | ✅ HealthModal.test.tsx | ✅ green |
| 54-01-02 | 01 | 1 | TS-01 | unit (raw) | `cd frontend && pnpm test` | ✅ HealthModal.test.tsx | ✅ green |
| 54-01-03 | 01 | 1 | TS-01 | unit (raw) | `cd frontend && pnpm test` | ✅ HealthModal.test.tsx | ✅ green |
| 54-02-01 | 02 | 1 | TS-02 | unit (go) | `go test -run TestAutoInstallTailscale .` | ✅ app_test.go | ✅ green |
| 54-02-02 | 02 | 1 | TS-02 | unit (raw) | `cd frontend && pnpm test` | ✅ HealthModal.test.tsx | ✅ green |
| 54-03-01 | 03 | 1 | TS-03 | unit (raw) | `cd frontend && pnpm test` | ✅ HealthModal.test.tsx | ✅ green |
| 54-03-02 | 03 | 1 | TS-03 | unit (raw) | `cd frontend && pnpm test` | ✅ HealthModal.test.tsx | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `app_test.go` — `TestAutoInstallTailscale` with 2 subtests (findBrew path resolution, method callable) — PASS

*All Wave 0 requirements fulfilled during Plan 01 execution.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Auto-install streams brew output in real-time | TS-02 | Requires real brew install + Wails event loop | Run app without Tailscale, click "Try Auto-Install", observe progress lines |
| Download links open in system browser | TS-01 | Requires Wails BrowserOpenURL runtime | Click download link in health modal, verify browser opens |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-07

---

## Validation Audit 2026-04-07

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

**Test suite results:**
- Frontend: 232 tests passed (11 test files), 5.19s
- Go: `TestAutoInstallTailscale` 2/2 subtests passed, 0.018s
- TS-01 coverage: 7 tests (brew/winget/curl commands, clipboard, onOpenURL, download links, CopyableCommand)
- TS-02 coverage: 4 tests (darwin gate, onAutoInstall prop, progress output, running state) + 2 Go subtests
- TS-03 coverage: 5 tests (MagicDNS, admin DNS link, numbered steps, CT disclosure, Check Again)

## Validation Audit 2026-04-07 (re-audit)

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

**Test suite results:**
- Frontend: 248 tests passed (12 test files), 5.68s
- Go: `TestAutoInstallTailscale` 2/2 subtests passed, 0.019s
- All TS-01, TS-02, TS-03 test blocks present in HealthModal.test.tsx
- No regressions detected — nyquist_compliant remains true
