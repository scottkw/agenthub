---
phase: 134
slug: modal-interaction
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-17
---

# Phase 134 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (jsdom environment) |
| **Config file** | `frontend/vite.config.ts` — `test: { environment: 'jsdom', globals: true, setupFiles: ['./src/test-setup.ts'] }` |
| **Quick run command** | `cd frontend && pnpm test --run --reporter=verbose` |
| **Full suite command** | `cd frontend && pnpm test --run` |
| **Estimated runtime** | ~30 seconds (1638+ frontend tests) |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test --run --reporter=verbose 2>&1 | tail -20`
- **After every plan wave:** Run `cd frontend && pnpm test --run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | — | MODAL-01 | — | N/A | unit (source inspection) | `pnpm test --run HubModal` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | MODAL-01 | — | Open/menu buttons stopPropagation; card body calls onCardClick | unit (source inspection) | `pnpm test --run SessionCard` | ✅ extend | ⬜ pending |
| TBD | TBD | — | MODAL-02 | — | Escape/close/click-outside → onClose; focus returns to card | unit (source inspection) | `pnpm test --run HubModal` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | MODAL-03 | — | attention=false → interactive; attention=true → briefing | unit (source inspection) | `pnpm test --run HubModal` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | MODAL-03 | — | HubInteractiveModal mounts TerminalPanel with correct props | unit (source inspection) | `pnpm test --run HubInteractiveModal` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | MODAL-04 | — | Briefing renders GetSessionTailLines; Send disabled when empty | unit (source inspection) | `pnpm test --run HubBriefingModal` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | MODAL-05 | — | TerminalPanel isActive=true while open (fit after animationend) | unit (source inspection) | `pnpm test --run HubInteractiveModal` | ❌ W0 | ⬜ pending |
| TBD | TBD | — | MODAL-06 | — | Remote w/o cap → onRequestRemoteCap, not direct modal open | unit (source inspection) | `pnpm test --run HubPanel` | ✅ extend | ⬜ pending |
| TBD | TBD | — | CSS | — | overlay position:fixed/inset:0/z-index:200; grow/shrink keyframes; reduced-motion guard | CSS assertion (raw read) | `pnpm test --run style.hub` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky. Task IDs assigned by the planner; this map mirrors the RESEARCH.md Validation Architecture.*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/Hub/HubModal.test.tsx` — source-inspection tests for MODAL-01, MODAL-02, MODAL-03 routing
- [ ] `frontend/src/components/Hub/HubInteractiveModal.test.tsx` — MODAL-03, MODAL-05
- [ ] `frontend/src/components/Hub/HubBriefingModal.test.tsx` — MODAL-04
- [ ] CSS assertions appended to existing `style.hub.test.ts` (or new `style.hub.modal.test.ts`) — overlay z-index, keyframe existence, reduced-motion guards

*Established test pattern: all Hub tests use source-inspection (`?raw` import or `readFileSync(style.css)`) — no JSDOM mounting of xterm.js components (canvas APIs absent in jsdom). New tests follow this pattern.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Grow/shrink animation visual timing + focus return | MODAL-01, MODAL-02 | xterm.js + animation require a live webview; jsdom cannot render canvas/animations | In native `wails dev` window: click a non-blocked card → observe grow-from-card; type/copy/paste/scrollback-search in modal terminal; Escape → shrink-back + focus returns to originating card |
| Briefing respond round-trip (input reaches session) | MODAL-04 | Requires a live session in waiting/needs-input state | Drive a real waiting session; open briefing modal; type a response + Send; confirm the session receives the input |
| Remote cap-gated modal via join-code | MODAL-06 | Requires a real remote peer + cap exchange | Open a remote cap-protected session card; confirm join-code modal appears and modal opens after cap acquired |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
