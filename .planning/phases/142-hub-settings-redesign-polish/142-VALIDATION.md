---
phase: 142
slug: hub-settings-redesign-polish
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-21
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
| 01-T1 | 142-01 | 1 | POL-02 | T-142-01 | N/A (test) | unit (RED) | `pnpm --dir frontend test run src/components/__tests__/SettingsTab.appearance-theme.test.tsx` | ✅ (update) | ⬜ pending |
| 01-T2 | 142-01 | 1 | POL-05/03/04 | T-142-01 | N/A (test) | unit + source-gate (RED) | `pnpm --dir frontend test run src/components/__tests__/Sidebar.test.tsx` | ✅ (extend) | ⬜ pending |
| 01-T3 | 142-01 | 1 | POL-05 | T-142-01 | N/A (test) | unit (RED) | `pnpm --dir frontend test run src/components/Hub/HubPanel.test.tsx` | ✅ (update) | ⬜ pending |
| 02-T1 | 142-02 | 2 | POL-04 | T-142-02/03 | Repaint timing only; no byte-handling change | source-gate + tsc | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/__tests__/Sidebar.test.tsx` | ✅ (Plan 01) | ⬜ pending |
| 02-T2 | 142-02 | 2 | POL-04 | T-142-02 | N/A | manual (wails dev, PTY) | n/a — native human check | n/a | ⬜ pending |
| 03-T1 | 142-03 | 2 | POL-05 | T-142-04/05/06 | Group input trim; client-only localStorage | unit + tsc | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/Hub/HubPanel.test.tsx` | ✅ (Plan 01) | ⬜ pending |
| 03-T2 | 142-03 | 2 | POL-05 | T-142-04/06 | React text-escaped group names; trim guard | unit + tsc + source-gate | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/__tests__/Sidebar.test.tsx` | ✅ (Plan 01) | ⬜ pending |
| 04-T1 | 142-04 | 3 | POL-01 | T-142-08 | More of same already-rendered tail | source-gate + tsc + manual (narrow-width visual) | `pnpm --dir frontend exec tsc --noEmit` + grep `hub-card__preview` | ✅ style.css | ⬜ pending |
| 04-T2 | 142-04 | 3 | POL-02 | T-142-07 | Colorblind-safe icon+text; persistence untouched | unit + tsc | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/__tests__/SettingsTab.appearance-theme.test.tsx` | ✅ (Plan 01) | ⬜ pending |
| 04-T3 | 142-04 | 3 | POL-03 | T-142-SC | N/A | source-gate + tsc | `pnpm --dir frontend exec tsc --noEmit && pnpm --dir frontend test run src/components/__tests__/Sidebar.test.tsx` | ✅ (Plan 01) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · Planner fills this map per task.*

---

## Wave 0 Requirements

- [ ] Existing vitest infrastructure covers most phase requirements (e.g. `SettingsTab.appearance-theme.test.tsx`).
- [ ] Update/migrate tests when source moves: `GroupSidebar.test.tsx` → `Sidebar.test.tsx` (POL-05, done in Plan 01 Task 2; old file deleted in Plan 03 Task 2), `SettingsTab.appearance-theme.test.tsx` → `[role="switch"][aria-checked]` (POL-02, Plan 01 Task 1).
- [ ] HubPanel.test.tsx updated to assert GroupSidebar side-panel absent + onGroupCountsChange (Plan 01 Task 3).
- [ ] Source-gate tests for POL-03 (PlusIcon) and POL-04 (pendingThemeRef) added to Sidebar.test.tsx (Plan 01 Task 2).

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

**Approval:** planned 2026-06-21
