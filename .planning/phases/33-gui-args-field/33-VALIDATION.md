---
phase: 33
slug: gui-args-field
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 33 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest |
| **Config file** | frontend/vitest.config.ts |
| **Quick run command** | `cd frontend && npx vitest run --reporter=verbose` |
| **Full suite command** | `cd frontend && npx vitest run --reporter=verbose` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run --reporter=verbose`
- **After every plan wave:** Run `cd frontend && npx vitest run --reporter=verbose`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 33-01-01 | 01 | 1 | ARGS-02 | unit | `cd frontend && npx vitest run` | ❌ W0 | ⬜ pending |
| 33-01-02 | 01 | 1 | ARGS-04 | unit | `cd frontend && npx vitest run` | ❌ W0 | ⬜ pending |
| 33-01-03 | 01 | 1 | ARGS-05 | unit | `cd frontend && npx vitest run` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Test stubs for args field rendering, persistence, and clear behavior
- [ ] Existing vitest infrastructure covers framework needs

*Existing infrastructure covers framework installation.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Args text field visible below folder picker | ARGS-02 | Visual layout verification | Open new-session modal, confirm args field position |
| Args passed to session correctly | ARGS-04 | End-to-end Wails runtime | Create session with args, verify in daemon logs |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
