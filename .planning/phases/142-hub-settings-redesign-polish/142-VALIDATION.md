---
phase: 142
slug: hub-settings-redesign-polish
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-21
validated: 2026-06-21
---

# Phase 142 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `pnpm --dir frontend test run <file>` |
| **Full suite command** | `pnpm --dir frontend test run && pnpm --dir frontend exec tsc --noEmit` |
| **Estimated runtime** | ~30 seconds |

> Note: a green vitest run is NOT proof the app builds — `tsc --noEmit` (or `wails dev`'s `tsc && vite build`) must also pass. POL-04's repaint fix is verified **manually in the native `wails dev` window** (the `:34115` bridge has no PTY).

---

## Sampling Rate

- **After every task commit:** Run `pnpm --dir frontend test run <touched file>`
- **After every plan wave:** Run `pnpm --dir frontend test run && pnpm --dir frontend exec tsc --noEmit`
- **Before `/gsd:verify-work`:** Full suite + tsc must be green; POL-04 manual native check passes
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-T1 | 142-01 | 1 | POL-02 | T-142-01 | N/A (test) | unit | `pnpm --dir frontend test run src/components/__tests__/SettingsTab.appearance-theme.test.tsx` | ✅ | ✅ green |
| 01-T2 | 142-01 | 1 | POL-05/03/04 | T-142-01 | N/A (test) | unit + source-gate | `pnpm --dir frontend test run src/components/__tests__/Sidebar.test.tsx` | ✅ | ✅ green |
| 01-T3 | 142-01 | 1 | POL-05 | T-142-01 | N/A (test) | unit | `pnpm --dir frontend test run src/components/Hub/HubPanel.test.tsx` | ✅ | ✅ green |
| 02-T1 | 142-02 | 2 | POL-04 | T-142-02/03 | Repaint timing only; no byte-handling change | source-gate + tsc | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/__tests__/Sidebar.test.tsx` | ✅ | ✅ green |
| 02-T2 | 142-02 | 2 | POL-04 | T-142-02 | N/A | manual (wails dev, PTY) | n/a — native human check | n/a | 🔲 manual-pending |
| 03-T1 | 142-03 | 2 | POL-05 | T-142-04/05/06 | Group input trim; client-only localStorage | unit + tsc | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/Hub/HubPanel.test.tsx` | ✅ | ✅ green |
| 03-T2 | 142-03 | 2 | POL-05 | T-142-04/06 | React text-escaped group names; trim guard | unit + tsc + source-gate | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/__tests__/Sidebar.test.tsx` | ✅ | ✅ green |
| 04-T1 | 142-04 | 3 | POL-01 | T-142-08 | More of same already-rendered tail | source-gate + tsc + manual (narrow-width visual) | `pnpm --dir frontend exec tsc --noEmit` + grep `hub-card__preview` (style.css:5366) | ✅ | ✅ green (auto) · 🔲 visual manual-pending |
| 04-T2 | 142-04 | 3 | POL-02 | T-142-07 | Colorblind-safe icon+text; persistence untouched | unit + tsc | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/__tests__/SettingsTab.appearance-theme.test.tsx` | ✅ | ✅ green |
| 04-T3 | 142-04 | 3 | POL-03 | T-142-SC | N/A | source-gate + tsc | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/__tests__/Sidebar.test.tsx` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · 🔲 manual-pending*

---

## Wave 0 Requirements

- [x] Existing vitest infrastructure covers most phase requirements (e.g. `SettingsTab.appearance-theme.test.tsx`).
- [x] Tests migrated as source moved: `GroupSidebar.test.tsx` → `Sidebar.test.tsx` (POL-05; old file confirmed removed), `SettingsTab.appearance-theme.test.tsx` asserts `role="switch"` + `aria-checked` (POL-02).
- [x] HubPanel.test.tsx asserts GroupSidebar side-panel absent (`hub__body` has `hub__grid-scroll`, no `.hub__group-sidebar`) + group-count prop API (Plan 01 Task 3).
- [x] Source-gate tests present in Sidebar.test.tsx: POL-03 `PlusIcon` in HubFilterBar/HubEmptyState; POL-04 `pendingThemeRef` + `fitTerminal` after `clearTextureAtlas` in TerminalPanel.tsx; POL-05 `.sidebar__group-list`/`.sidebar__group-item` CSS rules.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Terminal repaints cleanly after theme + tab switch | POL-04 | xterm.js WebGL repaint requires a real PTY + visible native webview; `:34115` bridge has no PTY | In `wails dev`: open a session, run output, switch theme, switch tabs and back, and switch theme cross-tab — terminal shows no garble (Plan 02 Task 2 blocking checkpoint) |
| Card icons clear of content at all widths | POL-01 | Visual reflow at narrow card widths | Resize Hub grid; confirm menu/handle never overlap name/status/preview (Plan 04 Task 1) |

*Colorblind-safe state (POL-02) verified at the hex/source level (grep) + icon-shape/text-label source assertion, never by eye — owner is colorblind.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (POL-04 native check is the only manual gate; backed by a `pendingThemeRef` source gate)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (Sidebar/HubPanel/SettingsTab tests + POL-03/04 source gates in Plan 01)
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-21

---

## Validation Audit 2026-06-21

| Metric | Count |
|--------|-------|
| Automated rows verified green | 8/8 (POL-01 auto, POL-02 ×2, POL-03, POL-04 source-gate, POL-05 ×3) |
| Manual-pending rows | 2 (POL-04 native terminal-repaint check; POL-01 narrow-width visual reflow) |
| Gaps found | 0 — strategy was already nyquist-compliant; this audit confirmed every automated command exists and passes |

**Verification basis:** Full frontend suite green (108 files / 1771 tests) + `tsc --noEmit` exit 0.
142-referenced files re-run in isolation: SettingsTab.appearance-theme + Sidebar + HubPanel = 3 files / 107 tests green. GroupSidebar.test.tsx confirmed removed; `.hub-card__preview` present at style.css:5366; POL-03/04/05 source gates present.

**Remaining manual checks** (documented manual-only, not blockers — require native `wails dev` PTY + visual reflow, neither assertable in jsdom; owner is colorblind so color is source-verified):
1. POL-04 — terminal repaints cleanly after theme + tab switch (run output, switch theme, switch tabs/back, cross-tab theme switch → no garble).
2. POL-01 — card menu/handle never overlap name/status/preview at narrow Hub grid widths.
