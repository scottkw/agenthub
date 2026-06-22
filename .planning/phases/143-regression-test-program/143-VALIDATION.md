---
phase: 143
slug: regression-test-program
status: planned
nyquist_compliant: true
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
| 143-01-01 | 01 | 1 | TEST-03 | — | N/A | unit (vitest) | `cd frontend && pnpm test -- hubGroupCounts agentBadge` | ❌ W0 | ⬜ pending |
| 143-01-02 | 01 | 1 | TEST-03 | — | N/A | unit (vitest) | `cd frontend && pnpm test -- Sidebar` | ❌ W0 | ⬜ pending |
| 143-01-03 | 01 | 1 | TEST-03 | — | N/A | unit (vitest) | `cd frontend && pnpm test -- style.hub` | ❌ W0 | ⬜ pending |
| 143-02-01 | 02 | 1 | TEST-01 | T-143-02 | path-map integrity | shell | `bash tests/check-traceability-paths.sh` | ❌ W0 | ⬜ pending |
| 143-02-02 | 02 | 1 | TEST-02 | T-143-02 | gate hosts path-check | config grep | `grep -q 'bash tests/check-traceability-paths.sh' .github/workflows/build.yml` | ❌ W0 | ⬜ pending |
| 143-03-01 | 03 | 2 | TEST-01, TEST-04, TEST-05 | — | N/A | doc + shell | `test -f TESTING.md && bash tests/check-traceability-paths.sh` | ❌ W0 | ⬜ pending |
| 143-03-02 | 03 | 2 | TEST-05 | — | N/A | doc | `test -f CLAUDE.md && grep -q 'TESTING.md' CLAUDE.md` | ❌ W0 | ⬜ pending |
| 143-04-01 | 04 | 3 | TEST-02 | T-143-01 | confirm-before-mutate | checkpoint:decision | (human gate) | n/a | ⬜ pending |
| 143-04-02 | 04 | 3 | TEST-02 | T-143-01, T-143-02 | required-check gate | gh api GET | `gh api repos/scottkw/agenthub/branches/main/protection --jq '.required_status_checks.checks \| length'` → 5 | n/a | ⬜ pending |
| 143-04-03 | 04 | 3 | TEST-02 | T-143-01 | gate blocks failing PR | checkpoint:human-verify | (human smoke-test) | n/a | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

No Wave 0 scaffolding (per RESEARCH.md Validation Architecture → "Wave 0 Gaps: None"). The gap-closure tests (GAP-01..04) are written as GREEN tests against existing stable logic, not RED-then-GREEN against new implementation. The path-check script (`tests/check-traceability-paths.sh`) is itself a Wave 1 deliverable (Plan 02), not a prerequisite scaffold — it tolerates an absent TESTING.md (exit 0) so it is green before Plan 03 authors the map.

The "❌ W0" markers above mean "file does not exist yet at planning time" — each is created by its own task within Wave 1, not by a separate scaffolding wave.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Native GUI / CLI flows, AirDrop'd signed-build checks, remote-peer UATs (M-01..M-11) | TEST-04 | No PTY in :34115 wails-dev bridge; web-share WS blocks automated input; signing/AirDrop require human + two machines | Captured as the single `TESTING.md` manual regression checklist (11 items per RESEARCH.md), authored in Plan 03 |
| Gate blocks a failing PR; admin doc-push still lands | TEST-02 | Requires opening a live draft PR and a live admin push to observe GitHub's merge-button state | Plan 04 Task 3 checkpoint:human-verify smoke-test |

*These are intentionally manual and live in the TESTING.md checklist / the Plan 04 checkpoint, not in CI.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or are checkpoint/human tasks with explicit verification steps
- [x] Sampling continuity: no 3 consecutive auto tasks without automated verify
- [x] Wave 0 covers all MISSING references (none required; documented above)
- [x] No watch-mode flags
- [x] Feedback latency < 60s (vitest quick)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planned 2026-06-21
