---
phase: 40
slug: daemon-management-panel
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-01
---

# Phase 40 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend), go test (backend — not needed this phase) |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `cd frontend && npx vitest run --reporter=verbose src/components/__tests__/DaemonManagerPanel.test.tsx` |
| **Full suite command** | `cd frontend && npx vitest run --reporter=verbose` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 40-01-01 | 01 | 1 | DMGR-03 | unit | `cd frontend && npx vitest run src/components/__tests__/DaemonManagerPanel.test.tsx` | ❌ W0 | ⬜ pending |
| 40-01-02 | 01 | 1 | DMGR-03 | integration | `cd frontend && npx vitest run --reporter=verbose` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/DaemonManagerPanel.test.tsx` — stubs for DMGR-03 panel behaviors

*Existing vitest infrastructure covers all phase requirements. No new framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Panel visually accessible in GUI | DMGR-03 (SC-1) | Requires running Wails app | Open app → click Sessions button → verify panel appears |
| Kill button removes session | DMGR-03 (SC-3) | Requires live daemon | Create session → open panel → click Kill → verify session removed |
| Web toggle reflects state | DMGR-03 (SC-4) | Requires Tailscale web server | Start web server → open panel → toggle web on/off |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
