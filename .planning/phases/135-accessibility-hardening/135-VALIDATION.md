---
phase: 135
slug: accessibility-hardening
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-18
validated: 2026-06-19
---

# Phase 135 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 |
| **Config file** | frontend/vite.config.ts (test key) |
| **Quick run command** | `cd frontend && npx vitest run <file>` |
| **Full suite command** | `cd frontend && npx vitest run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick command on affected test file
- **After every plan wave:** Run full suite command
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 135-01-T1 | 01 | 1 | A11Y-02 | T-135-01-01 | focus rings reveal only on-screen state | source-inspection (readFileSync) | `cd frontend && npx vitest run src/components/__tests__/style.hub.test.ts` | Yes (style.hub.test.ts) | ✅ green |
| 135-01-T2 | 01 | 1 | A11Y-03 | T-135-01-01 | static motion fallbacks | source-inspection (readFileSync) | `cd frontend && npx vitest run src/components/__tests__/style.hub.test.ts` | Yes (style.hub.test.ts) | ✅ green |
| 135-02-T1 | 02 | 1 | A11Y-02 | T-135-02-01 | aria-pressed same path as onClick | DOM-render (createRoot+getAttribute) | `cd frontend && npx vitest run src/components/Hub/HubFilterBar.test.tsx` | Yes | ✅ green |
| 135-02-T2 | 02 | 1 | A11Y-02 | T-135-02-02 | keyboard reaches same client state as click | DOM-render (createRoot+keydown) | `cd frontend && npx vitest run src/components/Hub/GroupSidebar.test.tsx` | Yes | ✅ green |
| 135-03-T1 | 03 | 1 | A11Y-01, A11Y-02 | T-135-03-02 | scoped Escape, no global suppression | source-inspection (`?raw`) | `cd frontend && npx vitest run src/components/Hub/HubModal.test.tsx` | Yes | ✅ green |
| 135-03-T2 | 03 | 1 | A11Y-04 | T-135-03-01 | inert paired with mandatory cleanup | source-inspection (`?raw`) | `cd frontend && npx vitest run src/components/Hub/HubModal.test.tsx` | Yes | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Nyquist Dimension 8:** every task has an `<automated>` verify command. No 3 consecutive tasks lack automated verification. jsdom 29 does NOT implement `inert` focus suppression and cannot exercise CSS media queries — A11Y-03 and A11Y-04 are therefore validated by source-inspection (`readFileSync`/`?raw`), the established pattern in `style.hub.test.ts` and `HubModal.test.tsx`. This is a deliberate, documented test-strategy choice, not a coverage gap.

---

## Wave 0 Requirements

No separate Wave 0 scaffolding plan is needed: every test target file already exists.

| Test file | Status | Used by |
|-----------|--------|---------|
| `frontend/src/components/__tests__/style.hub.test.ts` | EXISTS (Phase 131-132 hub CSS contract) — extended in 135-01 | 135-01 (A11Y-02 focus-visible, A11Y-03 reduce blocks) |
| `frontend/src/components/Hub/HubFilterBar.test.tsx` | EXISTS (has `renderFilterBar` helper) — extended in 135-02 | 135-02 T1 (aria-pressed) |
| `frontend/src/components/Hub/GroupSidebar.test.tsx` | EXISTS (DOM-render harness) — extended in 135-02 | 135-02 T2 (keyboard items) |
| `frontend/src/components/Hub/HubModal.test.tsx` | EXISTS (`?raw` pattern, stopImmediatePropagation test replaced) — extended in 135-03 | 135-03 (WR-05, A11Y-01 mirror, A11Y-04 inert) |

No `<automated>MISSING — Wave 0 must create…</automated>` references exist in any plan. All verify commands target existing files.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live keyboard Tab-trap (inert focus barrier actually blocks Tab to background cards) | A11Y-04 | jsdom 29 does not implement `inert` focus suppression; only a real WebView2/WKWebView enforces it | In `wails dev`: open the Hub, click a card to open its modal, press Tab repeatedly — focus must cycle only within the modal and never reach a background card; press Escape — focus returns to the originating card. (Source-level inert set/unset is auto-verified in 135-03; this confirms the runtime barrier.) |
| Visual focus-ring + reduced-motion by eye | A11Y-02, A11Y-03 | NOT REQUIRED — user is colorblind; verify ring color and motion gating at SOURCE against `var(--hub-accent)` and `prefers-reduced-motion: reduce`, never by eye | Covered by 135-01 source assertions. No by-eye check. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — all test files exist)
- [x] No watch-mode flags (`vitest run`, not `vitest`)
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved (planner)

---

## Validation Audit 2026-06-19

Retroactive audit of the executed phase (State A). All 6 task entries cross-referenced
against implementation summaries and live test runs. The four mapped test files were
executed: **150/150 tests passed** in 1.55s (style.hub.test.ts, HubFilterBar.test.tsx,
GroupSidebar.test.tsx, HubModal.test.tsx). Each requirement is covered by
behavior-targeting assertions (focus-visible ×15, prefers-reduced-motion ×7,
aria-pressed ×4, keydown Enter/Space ×6, inert/Escape/STATUS_CONFIG ×21). Statuses
flipped from plan-time `⬜ pending` to `✅ green`.

| Metric | Count |
|--------|-------|
| Requirements audited | 6 |
| COVERED | 6 |
| PARTIAL | 0 |
| MISSING | 0 |
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

No automated-coverage gaps. One pre-existing manual-only item remains by design
(live keyboard Tab-trap under real WebView — jsdom 29 cannot exercise `inert` focus
suppression).
