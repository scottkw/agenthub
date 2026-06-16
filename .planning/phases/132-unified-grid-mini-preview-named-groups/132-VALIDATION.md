---
phase: 132
slug: unified-grid-mini-preview-named-groups
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-16
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

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD by planner | — | — | — | — | — | — | — | — | ⬜ pending |

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

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
