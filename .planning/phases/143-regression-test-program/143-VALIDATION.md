---
phase: 143
slug: regression-test-program
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-21
---

# Phase 143 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) + go test (backend) + Playwright (e2e) + bash (build-script & path-check) |
| **Config file** | `frontend/vitest config`, `frontend/playwright.config.ts`, CI in `.github/workflows/build.yml` + `e2e.yml` |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `go test -race -short ./... && cd frontend && pnpm test && pnpm exec playwright test && bash tests/build-script.test.sh && bash tests/check-traceability-paths.sh` |
| **Estimated runtime** | ~minutes (full), ~seconds (vitest quick) |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test` (or the relevant per-file vitest)
- **After every plan wave:** Run the full suite command above
- **Before `/gsd:verify-work`:** Full suite must be green AND the new path-check script must exit 0
- **Max feedback latency:** ~60 seconds (vitest quick)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {N}-01-01 | 01 | 1 | TEST-{XX} | — | N/A | unit/shell | `{command}` | ❌ W0 | ⬜ pending |

*Filled by the planner per task. Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/check-traceability-paths.sh` — new path-existence CI check (D-03)
- [ ] New vitest files closing the v4.0 gaps (D-08/D-09) — `hubGroupCounts`, `agentBadge`, sidebar item count, Phase-142 comp-fidelity tokens

*Planner refines this list against the gap-analysis in RESEARCH.md.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Native GUI / CLI flows, AirDrop'd signed-build checks, remote-peer UATs | TEST-04 | No PTY in :34115 wails-dev bridge; web-share WS blocks automated input; signing/AirDrop require human | Captured as the single `TESTING.md` manual regression checklist (11 items per RESEARCH.md) |

*These are intentionally manual and live in the TESTING.md checklist, not in CI.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
