---
phase: 131
slug: hub-foundation-static-session-cards
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-16
---

# Phase 131 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend, 87 files / 1321 tests passing) + go test (backend) |
| **Config file** | frontend/vitest.config (existing); Go: standard `go test` |
| **Quick run command** | `cd frontend && pnpm vitest run <changed-files>` |
| **Full suite command** | `cd frontend && pnpm vitest run` (frontend); `go test ./...` (backend) |
| **Estimated runtime** | ~30–60 seconds frontend; backend varies |

---

## Sampling Rate

- **After every task commit:** Run quick command on changed test files
- **After every plan wave:** Run full suite for the layer touched
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

> Populated by the planner per task. Backend data-gap tasks (Wave 0 — ViewerCount, ExitCode, Duration, WorkDir on SessionInfo) verified via `go test` + binding regeneration check. Frontend card/grid/filter logic verified via vitest component + unit tests.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD by planner | — | — | — | — | — | — | — | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Backend `SessionInfo` extended with `ViewerCount`, `ExitCode`, `Duration`, `WorkDir`; Wails bindings regenerated (App.d.ts reflects new fields)
- [ ] Go unit coverage for the new field population path (engine.sessionWorkDirs → SessionInfo.WorkDir)
- [ ] Existing vitest infrastructure covers frontend card/grid/filter requirements

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Light/dark theme on-screen rendering | HUB-04 | Requires live app render; colorblind user verifies hex constants at source level, not by eye | Build app, open Hub, toggle theme; cross-check status/origin hex constants in style.css against UI-SPEC |
| Card grid responsive reflow | GRID-* | Visual layout requires live viewport | Resize window, confirm grid reflows without overlap |

*Remaining phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
