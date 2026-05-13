---
phase: 105-deferred-v3-2-uat-re-run
type: phase-summary
status: human_needed
requirements: [UAT-01..07]
---

# Phase 105 Summary — Deferred v3.2 UAT Re-Run

## What this phase delivered

- **105-UAT-RUNBOOK.md** — step-by-step runbook for the 7 deferred UAT scenarios, ready for the user to execute on physical hardware (desktop + iPad + Tailscale).

## What requires the user

All 7 UATs are physical-device interactive scenarios. They cannot be autonomously verified. The autonomous v3.3 pipeline put their prerequisites in place:

| UAT | Depends on | Verified by autonomous? |
|-----|-----------|-------------------------|
| UAT-01 (WebGL loss) | v3.0 Phase 93 | No — needs Chrome DevTools |
| UAT-02 (iPad SW raster banner) | v3.0 Phase 93 | No — needs physical iPad |
| UAT-03 (10K scrollback perf) | v3.0 Phase 94 | No — needs DevTools profiling |
| UAT-04 (Web-links chain) | Phase 102 ✓ (mailto + IDN) | Partial — code paths exercised in vitest |
| UAT-05 (chafa fidelity) | Phase 100/101 ✓ (shell sessions) | No — needs visual comparison |
| UAT-06 (two-client image join) | Phase 96 | No — needs two concurrent clients |
| UAT-07 (iPad 5-scenario) | Phases 100/101/102/103 ✓ | No — needs physical iPad |

## Hand-off

The user should:
1. Schedule a UAT session (estimated 1-2 hours for all 7 scenarios)
2. Follow 105-UAT-RUNBOOK.md
3. Check off each UAT in REQUIREMENTS.md as it passes
4. File bugs for any failures (will become quick tasks in v3.3 follow-up, OR rolled into v3.4)

## Status

**Phase status:** human_needed
**Phase complete only when:** User executes UAT-RUNBOOK and confirms all 7 pass.

For milestone audit purposes: this phase is "code-complete" — all prerequisites in place, runbook authored, ready for human UAT. The audit step can choose to either:
- Accept partial completion and mark milestone v3.3 complete with UATs pending
- Block milestone close until UATs pass

Recommended: accept partial; track UATs as a follow-up checkpoint, not a blocker.
