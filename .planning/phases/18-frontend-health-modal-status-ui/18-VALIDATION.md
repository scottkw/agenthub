---
phase: 18
slug: frontend-health-modal-status-ui
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-22
---

# Phase 18 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 (existing) |
| **Config file** | `frontend/vite.config.ts` — `test.environment: 'jsdom'` |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm test` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test`
- **After every plan wave:** Run `cd frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 18-01-01 | 01 | 1 | HEALTH-04 | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 18-01-02 | 01 | 1 | HEALTH-04 | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 18-01-03 | 01 | 1 | HEALTH-04 | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 18-01-04 | 01 | 1 | HEALTH-04 | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 18-02-01 | 02 | 1 | HEALTH-05 | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 18-02-02 | 02 | 1 | HEALTH-04 | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/HealthModal.test.tsx` — `?raw` source-inspection tests for HEALTH-04 panels (NotInstalled, NotConnected, NoCerts), platform conditionals (HEALTH-05), CT disclosure, onCheckAgain
- [ ] Updates to `frontend/src/components/__tests__/SettingsPanel.test.tsx` — add tests for `tailscaleHealth` prop and "Tailscale Status" label
- [ ] Updates to `frontend/src/components/__tests__/App.test.tsx` — verify `GetTailscaleStatus` call, `Environment` call, `tailscale:health` subscription, `HealthModal` in JSX

*Existing vitest infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Modal visually blocks interaction when unhealthy | HEALTH-04 | CSS overlay z-index stacking requires visual check | Launch app with Tailscale stopped; verify modal overlay prevents clicking tabs/settings |
| Platform-specific text reads correctly | HEALTH-05 | Content accuracy beyond structure check | Read each platform's instruction text on the respective OS |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
