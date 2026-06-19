---
phase: 133
slug: attention-pulse
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-16
validated: 2026-06-19
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

> Reconstructed by validate-phase audit 2026-06-19 from phase SUMMARYs (the planner left this map as a TBD placeholder). No Wave-0 backend gap — attention derives from existing status fields. All frontend CSS + TS.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 133-status | 01 | 1 | ATTN-01 | — | derives from existing status/exitCode | vitest util | `cd frontend && npx vitest run src/lib/hubStatus.test.ts` | ✅ | ✅ green |
| 133-card | 01/02 | 1 | ATTN-01, ATTN-02 | — | non-color cue (border + BellAlertIcon) | vitest | `cd frontend && npx vitest run src/components/Hub/SessionCard.test.tsx` | ✅ | ✅ green |
| 133-grid | 03 | 1 | ATTN-02, ATTN-04 | — | debounced FLIP reorder | vitest | `cd frontend && npx vitest run src/components/Hub/SessionCardGrid.test.tsx` | ✅ | ✅ green |
| 133-panel | 03 | 1 | ATTN-03, ATTN-05 | — | status-driven clear, no reload | vitest | `cd frontend && npx vitest run src/components/Hub/HubPanel.test.tsx` | ✅ | ✅ green |
| 133-badge | 04 | 1 | ATTN-06 | — | collapsed-group badge gate | vitest | `cd frontend && npx vitest run src/components/Hub/GroupSidebar.test.tsx` | ✅ | ✅ green |
| 133-css | 02 | 1 | ATTN-01, A11Y-03 | — | pulse keyframe + reduced-motion static fallback (colorblind-safe at source) | vitest CSS-contract | `cd frontend && npx vitest run src/components/__tests__/style.hub.test.ts` | ✅ | ✅ green |

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — no backend gap)
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved (validate-phase audit 2026-06-19)

---

## Validation Audit 2026-06-19

State A audit. Map reconstructed from SUMMARYs (planner left a TBD placeholder), then
re-run live: 6 entries all green — frontend mapped files **368/368** (shared run with
131/132), `tsc --noEmit` clean. No backend gap (attention derives from existing status).

| Metric | Count |
|--------|-------|
| Requirements audited | ATTN-01..06 (+ A11Y-03 CSS) |
| COVERED | 6 entries |
| PARTIAL / MISSING | 0 |
| Gaps found | 0 |

Manual-only items unchanged (pulse animation by-eye — N/A, colorblind user verifies hex +
icon at source; float-to-top smoothness; collapsed-group badge). These remain genuinely
operator-only — they require a live waiting/errored attention session the external
`wails dev` bridge cannot produce. See 133-HUMAN-UAT.md "Live Re-test 2026-06-19".
