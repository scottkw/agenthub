---
phase: 132
slug: unified-grid-mini-preview-named-groups
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-16
validated: 2026-06-19
---

# Phase 132 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) + go test (backend) |
| **Config file** | frontend/vite.config.ts (existing); Go: standard `go test` |
| **Quick run command** | `cd frontend && pnpm vitest run <changed-files>` |
| **Full suite command** | `cd frontend && pnpm vitest run` ; `go test ./...` |
| **Estimated runtime** | ~30–60s frontend; backend varies |

---

## Sampling Rate

- **After every task commit:** Run quick command on changed test files
- **After every plan wave:** Run full suite for the layer touched
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

> Populated by the planner per task. Wave 0 = `GetSessionTailLines` RPC (daemon route + client + app.go + App.d.ts) sourced from the existing relay scrollback ring buffer; verified via `go test` + binding-stub grep. Frontend mini-preview/group-sidebar/named-group/remote-grid logic verified via vitest. localStorage persistence via vitest (jsdom polyfill in test-setup.ts).

> Reconstructed by validate-phase audit 2026-06-19 from phase SUMMARYs (the planner left this map as a TBD placeholder). Each requirement mapped to its executed test file.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 132-W0-go | — | 0 | CARD-07 (tail RPC) | — | strips relay framing + ANSI before return | go unit | `go test ./internal/daemon/... -count=1` | ✅ | ✅ green |
| 132-W0-bind | — | 0 | CARD-07 (tail RPC) | — | N/A | go build + binding | `go build ./... && grep -q GetSessionTailLines frontend/src/wailsjs/go/main/App.d.ts` | ✅ | ✅ green |
| 132-mini | 01/02 | 1 | CARD-07 | — | React escapes tail text | vitest | `cd frontend && npx vitest run src/components/Hub/MiniPreview.test.tsx` | ✅ | ✅ green |
| 132-poller | 02 | 1 | CARD-07 | — | single shared poll interval | vitest | `cd frontend && npx vitest run src/components/Hub/HubPanel.test.tsx` | ✅ | ✅ green |
| 132-card | 01 | 1 | CARD-07 | — | preview placement on card | vitest | `cd frontend && npx vitest run src/components/Hub/SessionCard.test.tsx` | ✅ | ✅ green |
| 132-grid | 03 | 2 | GRID-02 (grouped grid) | — | N/A | vitest | `cd frontend && npx vitest run src/components/Hub/SessionCardGrid.test.tsx` | ✅ | ✅ green |
| 132-sidebar | 03 | 2 | GRID-03, GROUP-01 | — | N/A | vitest | `cd frontend && npx vitest run src/components/Hub/GroupSidebar.test.tsx` | ✅ | ✅ green |
| 132-groups | 04 | 2 | GROUP-01..04 | — | name:::workDir key survives id churn | vitest | `cd frontend && npx vitest run src/lib/hubGroups.test.ts` | ✅ | ✅ green |
| 132-remote | 05 | 3 | GRID-07 (unified remote) | — | remote sessions excluded from tail poll | vitest | `cd frontend && npx vitest run src/lib/remoteAdapter.test.ts` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `GetSessionTailLines(id, n)` RPC: daemon HTTP route + daemon client method + Wails-bound app.go method + App.d.ts stub; strips relay framing (0x01) + ANSI before returning plain-text lines (per RESEARCH §scrollback)
- [ ] Go unit coverage for the tail-extraction + strip path (reuse engine_test.go strip patterns)
- [ ] Existing vitest infrastructure (+ jsdom localStorage polyfill) covers frontend requirements

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Mini-preview perf at scale (no regression) | CARD-07 | Throughput/jank requires live app with many sessions | Build app, open Hub with 10+ sessions, confirm no jank; one shared throttle (not per-card) |
| Drag-and-drop card → group | GROUP-04 | Native HTML5 DnD pointer gesture | Drag a card onto a group in the sidebar; confirm membership + localStorage persists across restart |
| Remote peer card in grid | GRID-03 | Requires a live tailnet peer | Confirm remote session appears with peer hostname origin |

*Remaining behaviors have automated verification.*

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

State A audit. Map reconstructed from SUMMARYs (planner left a TBD placeholder), then
re-run live: 9 entries all green — frontend mapped files **368/368** (shared run with
131/133), `go test ./internal/daemon/...` **ok**, `go build ./...` **ok**, `App.d.ts`
contains `GetSessionTailLines`, `tsc --noEmit` clean.

| Metric | Count |
|--------|-------|
| Requirements audited | 9 entries (CARD-07, GRID-02/03/07, GROUP-01..04) |
| COVERED | 9 |
| PARTIAL / MISSING | 0 |
| Gaps found | 0 |

Manual-only items unchanged (mini-preview perf at scale, native DnD gesture, live remote
peer). GROUP-01..04 + GRID-02/03 additionally confirmed live this date — see
132-HUMAN-UAT.md "Live Re-test 2026-06-19".
