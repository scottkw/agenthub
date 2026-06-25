---
phase: 140
slug: ui-spec-gate
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-20
---

# Phase 140 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

**Phase type: documentation / decision-gate.** Phase 140 produces a UI-spec contract
(`140-UI-SPEC.md`) and a GitHub issue #93 triage comment. It changes no source code — git
confirms only `.planning/` markdown files were modified. There is no executable behavior to
exercise with a test framework. Both requirements are verified by content/state assertions
that are inherently manual-only (a planning-doc grep would be brittle; a GitHub-state check
requires a live network call). The actual redesign implementation and its automated tests
belong to Phase 141 (Redesign Implementation) and Phase 142 (Regression Test Program).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | N/A — no code under test (documentation phase) |
| **Config file** | none |
| **Quick run command** | N/A |
| **Full suite command** | N/A |
| **Estimated runtime** | N/A |

---

## Sampling Rate

- **After every task commit:** N/A — no automated suite for a documentation phase
- **After every plan wave:** N/A
- **Before `/gsd:verify-work`:** N/A
- **Max feedback latency:** N/A

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 140-01-01 | 01 | 1 | RDS-01 | T-140-01 / — | N/A — local markdown write, no attack surface | manual (doc content) | see Manual-Only #1 | N/A | ✅ green |
| 140-01-02 | 01 | 1 | RDS-01 | T-140-01 / — | N/A | manual (doc content) | see Manual-Only #1 | N/A | ✅ green |
| 140-02-01 | 02 | 1 | CARRY-02 | — | N/A — read/comment on GitHub issue | manual (external state) | see Manual-Only #2 | N/A | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*No automated test files: the deliverables are a planning document and an external GitHub
comment, neither of which is appropriate to assert against in the application test suite.*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements — none are automatable as code tests.
No test framework install or stub files are needed for a documentation/decision-gate phase.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| UI-spec records Direction 01 "Refined Native" with accent lock, conflict reconciliation, and recolor-only restyle depth | RDS-01 | Deliverable is a planning markdown artifact, not executable behavior; asserting on doc prose in the app test suite would be brittle and out of place | `f=.planning/phases/140-ui-spec-gate/140-UI-SPEC.md; test -f "$f" && grep -q "Direction 01 — Refined Native" "$f" && grep -q "#7aa2f7" "$f" && grep -q "#3d6fe8" "$f" && grep -i "#7C8CFF" "$f" \| grep -qi reject && for id in D-08 D-09 D-10 D-11 D-12 D-13; do grep -q "$id" "$f"; done && grep -qi "recolor only" "$f"` — all clauses must pass |
| Issue #93 carries the v4.0 triage comment (all 7 #78 items re-deferred, in-scope subset = none) and remains OPEN | CARRY-02 | Deliverable is an external GitHub comment; verification requires a live `gh` API call — non-deterministic, not a unit/integration test | `gh issue view 93 --json state -q .state` returns `OPEN`; `gh issue view 93 --comments \| grep -qi "v4.0 triage"`; comment enumerates: usage metrics, projects, avatars, agent suggests, session detail, tweaks panel, pixel-faithful |

---

## Validation Sign-Off

- [x] All tasks have manual verification documented (no automatable code in this phase)
- [x] Sampling continuity: N/A — documentation phase, no automated suite
- [x] Wave 0 covers all MISSING references — none exist
- [x] No watch-mode flags
- [x] Feedback latency: N/A
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-06-20

---

## Validation Audit 2026-06-20

| Metric | Count |
|--------|-------|
| Requirements audited | 2 (RDS-01, CARRY-02) |
| Automated-test gaps found | 0 |
| Resolved (tests generated) | 0 |
| Escalated to manual-only | 2 |

Reconstructed from artifacts (State B): no prior VALIDATION.md existed; two SUMMARY files
present. Both requirements classified manual-only — Phase 140 is a documentation/decision
gate with no source code. Both verifications already proven green in `140-VERIFICATION.md`
(14/14 must-have truths). No test files generated.
