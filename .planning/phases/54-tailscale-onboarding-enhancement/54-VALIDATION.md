---
phase: 54
slug: tailscale-onboarding-enhancement
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-07
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
| 54-01-01 | 01 | 1 | TS-01 | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) | ⬜ pending |
| 54-01-02 | 01 | 1 | TS-01 | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) | ⬜ pending |
| 54-01-03 | 01 | 1 | TS-01 | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) | ⬜ pending |
| 54-02-01 | 02 | 1 | TS-02 | unit | `go test ./... -run TestAutoInstall` | ❌ W0 | ⬜ pending |
| 54-02-02 | 02 | 1 | TS-02 | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) | ⬜ pending |
| 54-03-01 | 03 | 1 | TS-03 | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) | ⬜ pending |
| 54-03-02 | 03 | 1 | TS-03 | unit (raw) | `cd frontend && pnpm test` | ✅ (extend HealthModal.test.tsx) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `app_test.go` — add `TestAutoInstallTailscale` covering: brew path resolution, error on non-darwin, goroutine event emission

*Existing HealthModal.test.tsx covers most requirements via raw import assertions; extend in-place.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Auto-install streams brew output in real-time | TS-02 | Requires real brew install + Wails event loop | Run app without Tailscale, click "Try Auto-Install", observe progress lines |
| Download links open in system browser | TS-01 | Requires Wails BrowserOpenURL runtime | Click download link in health modal, verify browser opens |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
