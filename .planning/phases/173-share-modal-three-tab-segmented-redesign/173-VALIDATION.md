---
phase: 173
slug: share-modal-three-tab-segmented-redesign
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-08
---

# Phase 173 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) — raw `react-dom/client` + `flushSync`; NO `@testing-library/react` in this codebase |
| **Config file** | `frontend/vitest.config.ts` (or root vite/vitest config) |
| **Quick run command** | `cd frontend && pnpm vitest run src/components/__tests__/SessionShareModal.test.tsx src/components/__tests__/SessionSharePanel.test.tsx` |
| **Full suite command** | `cd frontend && pnpm vitest run` then `cd frontend && pnpm tsc --noEmit && pnpm vite build` (tsc+build gate — vitest alone tolerates TS errors wails rejects) |
| **Estimated runtime** | ~30–60 seconds (targeted vitest); build gate adds ~1–2 min |

---

## Sampling Rate

- **After every task commit:** Run the quick run command (targeted share test files).
- **After every plan wave:** Run the full vitest suite.
- **Before `/gsd-verify-work`:** Full vitest suite green AND `tsc --noEmit && vite build` succeeds.
- **Max feedback latency:** ~60 seconds for targeted tests.

---

## Per-Task Verification Map

*Populated by the planner as tasks are assigned. Every task must map to an automated vitest assertion or a documented manual-only UAT item. Assertions must be attribute/text-based (aria-selected, aria-disabled, role, class presence, text labels) — NEVER hue/computed-color based (owner is colorblind; SM-07/SM-08).*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | SM-0X | — | N/A (layout/IA only) | unit | `pnpm vitest run …` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Update `SessionShareModal.test.tsx` — assert fixed control strip, segmented `role=tablist`/`role=tab`, `aria-selected`, `aria-disabled` on Internet tabs until confirmed, default tab = Internet·Read-only after confirm, reset-to-Tailnet on disable (SM-01/03/05/07).
- [ ] Update `SessionSharePanel.test.tsx` (or new per-tab test files) — assert `ShareLinkCard` structure, scope description attached beneath link, public-write flow present ONLY in Full-access tab, hold-to-confirm still gates arming (SM-04/06/08).
- [ ] If `SessionSharePanel.tsx` is deleted/split, redistribute its tests across new per-tab test files AND update `TESTING.md` Suite Manifest + Traceability map (repo standing rule).

*Existing infrastructure (vitest + raw react-dom) covers all phase requirements — no new framework install.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| No whole-dialog scroll / no reflow-on-toggle in a live build | SM-01, SM-02 | Layout reflow + bounded-height need a real rendered viewport; jsdom has no layout engine | Live-build the app, open Share modal, flip each toggle, confirm toggles never move and only the tab body region changes; confirm no whole-dialog scrollbar at normal viewport |
| Colorblind-safe distinction (⚠ glyph + inset ring, not hue) visually | SM-07 | Attribute assertions prove structure; visual affordance needs human/live check | Live build; confirm Full-access tab shows ⚠ glyph + inset ring when active; verify at source that ring is `box-shadow inset`, not a hue swap |
| `prefers-reduced-motion` hold-to-confirm fallback | SM-07 | Media-query behavior needs a real browser env | Live build with reduced-motion enabled; confirm hold-to-confirm degrades to plain confirm |

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
