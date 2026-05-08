---
phase: 98-progress-addon-p2-cuttable
plan: "05"
subsystem: progress-addon
tags: [progress, web-parity, playwright, e2e, manual-uat, wave-4, cuttable-last]
status: partial
dependency_graph:
  requires:
    - "98-01 (wave-0: vendored addon + RED scaffolds)"
    - "98-02 (wave-1: aggregateProgress + SetTrayProgress RPC)"
    - "98-03 (wave-2: TerminalPanel hot-swap + App.tsx registry + tabProgress wiring)"
    - "98-04 (wave-3: TabBar.tsx per-tab .tab__progress element + style.css rules)"
  provides:
    - "web/terminal.html: <div id='progress-underline' aria-hidden='true'> above xterm container"
    - "web/assets/terminal.css: #progress-underline rule (position:fixed; transform:scaleX(0); #7aa2f7; 200ms ease-out; z-index:1000)"
    - "web/assets/terminal.js: ProgressAddon construction gated on pluginConfig.progress + onChange handler driving #progress-underline.style.transform"
    - "frontend/e2e/progress.spec.ts: 3 test.skip blocks mirroring Phase 95 web-links-live-toggle.spec.ts shape"
    - ".planning/phases/98-progress-addon-p2-cuttable/98-HUMAN-UAT.md: 3-scenario UAT runbook (pending human sign-off)"
    - "PRG-OFF gating invariant satisfied on web side (pluginConfig.progress gate in same file as constructor)"
  affects:
    - "Phase 99 release-gate: Playwright plumbing to unblock progress.spec.ts from test.skip"
    - "Phase 99 release-gate: cross-browser CSP e2e re-verify of #progress-underline"
tech_stack:
  added: []
  patterns:
    - "Web-side ProgressAddon UMD construction follows SerializeAddon.SerializeAddon pattern verbatim"
    - "position:fixed top:0 for web-page progress bar (vs position:absolute bottom:0 for desktop .tab__progress)"
    - "transform:scaleX(0..1) for GPU compositor animation — no layout reflow — mirrors desktop Pitfall #4 invariant"
    - "Playwright test.skip scaffold with numbered documented walk-through (Phase 95 pattern)"
key_files:
  created:
    - "frontend/e2e/progress.spec.ts (3 test.skip blocks: desktop per-tab, web-page underline, tray quartile)"
    - ".planning/phases/98-progress-addon-p2-cuttable/98-HUMAN-UAT.md (3-scenario UAT runbook)"
  modified:
    - "web/terminal.html (progress-underline div added above xterm container)"
    - "web/assets/terminal.css (#progress-underline rule with position:fixed + transform animation)"
    - "web/assets/terminal.js (ProgressAddon construction + onChange handler)"
key-decisions:
  - "position:fixed (not absolute) for #progress-underline — viewport-relative bar visible regardless of page scroll; mirrors RESEARCH Task 1 Sub-task B rationale"
  - "z-index:1000 ensures bar sits above xterm canvas layers without requiring changes to existing stacking context"
  - "Web page has no tab strip (Pitfall #10) — fixed top bar is the intentional web analog of per-tab .tab__progress desktop underline"
  - "3 test.skip blocks in progress.spec.ts mirror the 3 web-links-live-toggle.spec.ts blocks exactly (same import, same describe shape, same throw pattern)"
  - "Frontend dist built in worktree as Rule 3 fix (blocking: wails build needed frontend/dist to embed assets)"
patterns-established:
  - "Pattern: Web-side ProgressAddon construction mirrors SerializeAddon — gated on pluginConfig.progress, inside try/catch, silent on failure"
  - "Pattern: onChange handler guards against missing DOM element via `if (!bar) return` before style mutation"
  - "Pattern: Playwright test.skip scaffold authored at plan time (not deferred) so `pnpm exec playwright test` surfaces the documented walk, not silence"
requirements-completed: [PRG-02]
duration: ~35min
completed: "2026-05-08"
---

# Phase 98 Plan 05: Wave 4 Web Parity + Playwright Scaffold + Manual UAT Summary

**Web-side #progress-underline fixed bar wired to OSC 9;4 onChange via ProgressAddon UMD; Playwright e2e scaffold (3 test.skip blocks) parked for Phase 99 plumbing; 98-HUMAN-UAT.md runbook authored — awaiting human sign-off checkpoint.**

## Performance

- **Duration:** ~35 minutes
- **Started:** 2026-05-08T15:18:38Z
- **Completed:** 2026-05-08T15:XX:XXZ (checkpoint pending)
- **Tasks completed:** 2 of 3 (Task 3 is checkpoint:human-verify — returned to orchestrator)
- **Files modified:** 5 (3 web + 1 e2e + 1 UAT)

## Status: PARTIAL — UAT Checkpoint Pending

Plan 98-05 is autonomous: false. Tasks 1 and 2 executed fully and committed.
Task 3 (checkpoint:human-verify) was reached and the structured checkpoint has been
returned to the orchestrator. This SUMMARY reflects the partial-complete state.

After human UAT sign-off, Task 3 will be completed by the resume agent.

## Accomplishments

- Extended `web/terminal.html` with `<div id="progress-underline" aria-hidden="true">` above the xterm container (sibling of #webgl-recovery-banner)
- Added `#progress-underline` CSS rule in `web/assets/terminal.css` — position:fixed; top:0; scaleX(0) initial; #7aa2f7 accent; 200ms ease-out; z-index:1000; pointer-events:none
- Added ProgressAddon construction block to `web/assets/terminal.js` after the existing SerializeAddon block (lines 267-272 precedent) — gated on `pluginConfig.progress`, drives #progress-underline.style.transform on every onChange event
- All 3 PRG release tests GREEN (TestPRG_OffPath_NoProgressLogic, TestPRG_NewProgressAddonIsGated, TestPRG_SetTrayProgressUsage)
- `wails build -tags wailsassets` succeeds (web assets re-embedded into binary)
- Created `frontend/e2e/progress.spec.ts` with 3 test.skip blocks mirroring the 74-line Phase 95 web-links-live-toggle.spec.ts scaffold exactly — TypeScript compiles cleanly
- Authored `98-HUMAN-UAT.md` with 3 numbered scenarios + web parity addendum + final sign-off matrix

## Web-Side Underline Placement

The plan specified placing the #progress-underline element "above the xterm container" in document order. The actual placement:

```html
<div id="webgl-recovery-banner" hidden ...></div>
<!-- Phase 98 PRG-02 — web-side progress underline -->
<div id="progress-underline" aria-hidden="true"></div>
<div id="terminal">
  <!-- find-bar inside -->
</div>
```

This places it as a flex-sibling of #webgl-recovery-banner and #terminal in the body's flex column. The CSS `position: fixed; top: 0` removes it from the flex flow — it always anchors to the top of the viewport regardless of document structure.

## Pitfall #10 Note (Web Has No Tab Strip)

98-RESEARCH.md Pitfall #10 guided the placement decision: the web-served terminal page has no tab strip. Instead of a per-tab underline at the bottom of a tab, the web side uses a fixed-position bar at the TOP of the viewport — visible to the viewer of any individual session page regardless of how many sessions exist server-side.

## Playwright e2e Scaffold Scope

The 3 test.skip blocks in `frontend/e2e/progress.spec.ts` cover:
1. **OSC 9;4 → .tab__progress scaleX(0.47)** — desktop per-tab underline walk (PRG-02)
2. **#progress-underline transform from OSC 9;4 events** — web-served session walk (PRG-02 web-half)
3. **Cross-session aggregate tray quartile RPC at 200ms debounce** — tray glyph walk (PRG-03)

All parked at `test.skip` until Phase 99 release-gate Playwright plumbing lands (daemon RPC fixture + sessions fixture needed). The documented walk-throughs are the authoritative source for what the automated tests must eventually assert.

## Task Commits

1. **Task 1: Web parity #progress-underline DOM + CSS + ProgressAddon construction** — `9eb4c85` (feat)
2. **Task 2: Playwright e2e scaffold (test.skip; mirrors Phase 95 shape)** — `0fcd91c` (feat)
3. **UAT runbook: 98-HUMAN-UAT.md authored (pre-checkpoint)** — `dd820aa` (docs)

**Task 3 (checkpoint:human-verify):** Returned to orchestrator — awaiting human sign-off.

## Files Created/Modified

- `web/terminal.html` — progress-underline div inserted above #terminal (sibling of #webgl-recovery-banner)
- `web/assets/terminal.css` — #progress-underline rule: position:fixed top:0 height:2px background:#7aa2f7 transform:scaleX(0) transform-origin:left transition:200ms ease-out pointer-events:none z-index:1000
- `web/assets/terminal.js` — ProgressAddon construction block (gated on pluginConfig.progress) + onChange handler (state:1 → scaleX(value/100); else scaleX(0)) + if (!bar) return guard
- `frontend/e2e/progress.spec.ts` — 3 test.skip blocks (desktop underline; web underline; tray quartile)
- `.planning/phases/98-progress-addon-p2-cuttable/98-HUMAN-UAT.md` — 3-scenario UAT runbook with sign-off matrix

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Built frontend/dist to unblock wails build**
- **Found during:** Task 1 Sub-task F (build verification)
- **Issue:** The worktree had no `frontend/dist` directory — `wails build -tags wailsassets` failed with `assets_prod.go:10:12: pattern all:frontend/dist: no matching files found`. This is a pre-existing worktree condition, not introduced by Task 1's changes.
- **Fix:** Ran `cd frontend && pnpm install && pnpm build` to compile the frontend into dist/. Then re-ran `wails build -tags wailsassets` successfully.
- **Files modified:** `frontend/dist/` (generated; not committed — gitignored)
- **Verification:** wails build completed successfully: "Built '...agenthub.app' in 15.168s"
- **Committed in:** n/a (frontend/dist is gitignored)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The frontend build was a pre-existing worktree setup issue. No scope creep. All Task 1 web parity files were committed correctly.

## Known Stubs

None — all web parity functionality is implemented. The Playwright tests are intentional `test.skip` scaffolds (not stubs) — they document the walk rather than providing an empty test body.

## Threat Flags

No new threat surface beyond what the plan's threat model captures:
- T-98-04 (web vendor supply chain): mitigated by existing vendor_drift_test.go gate (unchanged)
- T-98-01 (untrusted CLI OSC 9;4 values): `if (!bar) return` guard + addon's [0,100] value clamp cover the web side

## Hand-off Note for Phase 99

- `progress.spec.ts` 3 test.skip blocks are the authoritative documented walk — Phase 99 Playwright plumbing should unskip them one by one
- PUI-02 italic caption for the Progress toggle is already done in Phase 98 Plan 01 (PluginsSection.tsx) — Phase 99 re-verifies through the cross-browser CSP e2e
- `#progress-underline` is position:fixed at z-index:1000 — Phase 99 CSP e2e should confirm no CSP violations from the element itself (it's pure CSS + DOM, no inline scripts or external resources)

## Self-Check

- [x] `web/terminal.html` contains `id="progress-underline"`
- [x] `web/assets/terminal.css` contains `#progress-underline {` rule
- [x] `web/assets/terminal.css` contains `transform: scaleX(0)` initial state
- [x] `web/assets/terminal.css` contains `background: #7aa2f7` TokyoNight accent
- [x] `web/assets/terminal.js` contains `new ProgressAddon.ProgressAddon()`
- [x] `web/assets/terminal.js` contains `pluginConfig.progress` gate
- [x] `web/assets/terminal.js` contains `getElementById('progress-underline')` DOM lookup
- [x] `frontend/e2e/progress.spec.ts` exists with 3 `test.skip(` function calls
- [x] `pnpm tsc --noEmit` clean (TypeScript compiles without errors)
- [x] PRG release tests GREEN: TestPRG_OffPath_NoProgressLogic, TestPRG_NewProgressAddonIsGated, TestPRG_SetTrayProgressUsage
- [x] `wails build -tags wailsassets` succeeds
- [x] Task 1 commit 9eb4c85 exists
- [x] Task 2 commit 0fcd91c exists
- [x] UAT commit dd820aa exists
- [x] 98-HUMAN-UAT.md authored with 3 scenarios + final sign-off matrix
- [x] SUMMARY.md created with status: partial

## Self-Check: PASSED
