---
phase: 133
slug: attention-pulse
status: draft
nyquist_compliant: false
wave_0_complete: true
created: 2026-06-16
---

# Phase 133 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) |
| **Config file** | frontend/vite.config.ts (existing) |
| **Quick run command** | `cd frontend && pnpm vitest run <changed-files>` |
| **Full suite command** | `cd frontend && pnpm vitest run` |
| **Estimated runtime** | ~30–60s |

---

## Sampling Rate

- **After every task commit:** Run quick command on changed test files
- **After every plan wave:** Run full suite
- **Before `/gsd:verify-work`:** Full suite green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

> Populated by the planner per task. NO Wave-0 backend gap this phase — attention derives from EXISTING status fields (deriveHubStatus already returns waiting/errored/stopped-err). All work is frontend CSS + TS, verified via vitest (component + util) + style.hub CSS-contract assertions for the pulse keyframes / attention border / reduced-motion guard. ATTN-03 (status-driven clear) tested by simulating a status change in a unit test (no modal needed).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD by planner | — | — | — | — | — | — | — | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] No backend gap — attention derives from existing SessionInfo status/exitCode (confirmed in RESEARCH). Existing vitest infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Pulse animation appearance | ATTN-01 | Live animation; colorblind user verifies via source hex + icon, not by eye | Build app, set a session to waiting/errored; confirm pulsing border + BellAlertIcon; toggle prefers-reduced-motion → static border |
| Debounced float-to-top + FLIP reorder smoothness | ATTN-02 | Animation smoothness + debounce timing require live observation | Trigger a status change; confirm card floats to top of its group after ~1s debounce with a smooth (non-jarring) FLIP transition |
| Collapsed-group attention badge | ATTN-06 | Requires collapsed sidebar + a real attention session | Collapse the group sidebar with an attention session present; confirm the attention badge shows on the collapsed group |

*Remaining behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (none — no backend gap)
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
