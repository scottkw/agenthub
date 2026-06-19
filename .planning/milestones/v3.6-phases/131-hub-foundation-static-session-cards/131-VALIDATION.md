---
phase: 131
slug: hub-foundation-static-session-cards
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-16
validated: 2026-06-19
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

> Backend data-gap tasks (Wave 0 — ViewerCount, ExitCode, Duration, WorkDir on SessionInfo) verified via `go test` + binding regeneration check. Frontend card/grid/filter logic verified via vitest component + unit tests.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 131-01-01 | 01 | 0 | CARD-04/05/06, GRID-02 | — | N/A | go unit | `go test ./internal/daemon/... -count=1 -run ListSessions` | ✅ | ✅ green |
| 131-01-02 | 01 | 0 | CARD-04/05/06, GRID-02 | — | N/A | go build + binding | `go build ./... && grep -q "workDir" frontend/src/wailsjs/go/main/App.d.ts` | ✅ | ✅ green |
| 131-02-01 | 02 | 1 | CARD-01 | T-131-02 (XSS) | React escapes session name on render | vitest | `pnpm vitest run src/components/Hub/InlineSessionName.test.tsx` | ❌ W1 | ✅ green |
| 131-02-02 | 02 | 1 | CARD-01..06, CARD-08 | T-131-02 (XSS) | React escapes name/host/workDir | vitest | `pnpm vitest run src/components/Hub/SessionCard.test.tsx` | ❌ W1 | ✅ green |
| 131-03-01 | 03 | 1 | GRID-04/05/06, HUB-03 | — | N/A | vitest | `pnpm vitest run src/components/Hub/HubFilterBar.test.tsx` | ❌ W1 | ✅ green |
| 131-03-02 | 03 | 1 | GRID-04/05/06, HUB-03 | — | N/A | vitest | `pnpm vitest run src/components/Hub/HubEmptyState.test.tsx` | ❌ W1 | ✅ green |
| 131-04-01 | 04 | 2 | GRID-02/04/05/06 | — | N/A | vitest | `pnpm vitest run src/components/Hub/SessionCardGrid.test.tsx` | ❌ W2 | ✅ green |
| 131-04-02 | 04 | 2 | GRID-04/05/06, HUB-03 | — | N/A | vitest | `pnpm vitest run src/components/Hub/HubPanel.test.tsx` | ❌ W2 | ✅ green |
| 131-05-01 | 05 | 3 | HUB-01/02, GRID-01 | — | Hub coexists; daemon gate untouched | vitest + tsc | `pnpm vitest run src/components/__tests__/Sidebar.test.tsx src/components/__tests__/App.hub.test.tsx && pnpm exec tsc --noEmit` | ❌ W3 | ✅ green |
| 131-05-02 | 05 | 3 | HUB-04, GRID-06, CARD-08 | — | colorblind-safe hex at source; reduced-motion guard | vitest CSS-contract | `pnpm vitest run src/components/__tests__/style.hub.test.ts` | ❌ W3 | ✅ green |

*(All commands run from `frontend/` except the Go commands which run from repo root.)*

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Backend `SessionInfo` extended with `ViewerCount`, `ExitCode`, `Duration`, `WorkDir`; Wails bindings regenerated (App.d.ts reflects new fields)
- [x] Go unit coverage for the new field population path (engine.sessionWorkDirs → SessionInfo.WorkDir)
- [x] Existing vitest infrastructure covers frontend card/grid/filter requirements

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Light/dark theme on-screen rendering | HUB-04 | Requires live app render; colorblind user verifies hex constants at source level, not by eye | Build app, open Hub, toggle theme; cross-check status/origin hex constants in style.css against UI-SPEC |
| Card grid responsive reflow | GRID-* | Visual layout requires live viewport | Resize window, confirm grid reflows without overlap |

*Remaining phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved (validate-phase audit 2026-06-19)

---

## Validation Audit 2026-06-19

Retroactive audit of the executed phase (State A). All 10 task entries cross-referenced
against implementation summaries and re-run live: frontend mapped files **368/368 green**
(shared run with 132/133), `go test ./internal/daemon/...` **ok**, `go build ./...` **ok**,
`App.d.ts` contains `workDir`, `tsc --noEmit` clean. Statuses flipped `⬜ pending → ✅ green`.

| Metric | Count |
|--------|-------|
| Tasks audited | 10 |
| COVERED | 10 |
| PARTIAL / MISSING | 0 |
| Gaps found | 0 |

Manual-only items unchanged (HUB-04 by-eye theme — N/A, colorblind user verifies hex at
source; GRID responsive reflow). Light/dark hex + a broad set of card/grid items were
additionally confirmed live this date — see 131-HUMAN-UAT.md "Live Re-test 2026-06-19".
