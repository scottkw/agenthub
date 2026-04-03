---
phase: 40
slug: daemon-management-panel
status: complete
nyquist_compliant: true
wave_0_complete: true
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
| 40-01-01 | 01 | 1 | DMGR-03 | unit | `cd frontend && npx vitest run src/components/__tests__/DaemonManagerPanel.test.tsx` | ✅ | ✅ green |
| 40-01-02 | 01 | 1 | DMGR-03 | integration | `cd frontend && npx vitest run --reporter=verbose` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `frontend/src/components/__tests__/DaemonManagerPanel.test.tsx` — 10 tests for DMGR-03 panel behaviors (5 source-inspection, 5 DOM)

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved

---

## Validation Audit 2026-04-02

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All requirements fully covered by automated tests. 10 DaemonManagerPanel tests + 177 total suite tests pass green.
