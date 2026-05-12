---
phase: 98-progress-addon-p2-cuttable
plan: "01"
subsystem: progress-addon
tags: [progress, vendoring, wave-0, scaffolds, red-tests, tray-icons]
dependency_graph:
  requires: []
  provides:
    - "web/vendor/xterm/addons/addon-progress.js (vendored UMD)"
    - "web/vendor/xterm/VERSION (10-line manifest)"
    - "frontend/src/lib/aggregateProgress.ts (stub)"
    - "internal/release/no_progress_when_off_test.go (3-test scaffold)"
    - "assets/tray_icon_progress_{25,50,75,100}.png (18x18 quartile glyphs)"
    - "tests/fixtures/osc94-progress-fixture.sh (UAT fixture)"
  affects:
    - "web/embed.go (//go:embed extended)"
    - "web/terminal.html (script tag added)"
    - "internal/webserver/vendor_drift_test.go (min-count 9→10)"
    - "frontend/package.json + pnpm-lock.yaml"
    - "frontend/src/components/PluginsSection.tsx (v3.3-flip caption)"
    - "frontend/src/components/__tests__/*.test.tsx (RED scaffolds)"
tech_stack:
  added:
    - "@xterm/addon-progress@0.2.0 (npm dependency + vendored UMD)"
  patterns:
    - "Phase 97 vendoring pipeline (7-step: pnpm add → cp → VERSION → embed.go → terminal.html → drift-test → verify)"
    - "Phase 97 SER-03 negative-regression test scaffold (filepath.WalkDir + regexp.MustCompile)"
    - "image/draw + image/png stdlib for generated PNG assets"
    - "source-inspection ?raw vitest pattern for all frontend RED scaffolds"
key_files:
  created:
    - "web/vendor/xterm/addons/addon-progress.js"
    - "frontend/src/lib/aggregateProgress.ts"
    - "frontend/src/lib/__tests__/aggregateProgress.test.ts"
    - "internal/release/no_progress_when_off_test.go"
    - "tests/fixtures/osc94-progress-fixture.sh"
    - "build/gen_progress_icons.go"
    - "assets/tray_icon_progress_25.png"
    - "assets/tray_icon_progress_50.png"
    - "assets/tray_icon_progress_75.png"
    - "assets/tray_icon_progress_100.png"
  modified:
    - "frontend/package.json"
    - "frontend/pnpm-lock.yaml"
    - "web/vendor/xterm/VERSION"
    - "web/embed.go"
    - "web/terminal.html"
    - "internal/webserver/vendor_drift_test.go"
    - "frontend/src/components/PluginsSection.tsx"
    - "frontend/src/components/__tests__/PluginsSection.test.tsx"
    - "frontend/src/components/__tests__/TerminalPanel.test.tsx"
    - "frontend/src/components/__tests__/TabBar.test.tsx"
    - "frontend/src/components/__tests__/App.test.tsx"
decisions:
  - "Used lib/addon-progress.js (CJS UMD) not .mjs — consistent with Phase 97 SER-01 precedent for vendor copy"
  - "aggregateProgress.ts is a stub (returns 0) at Wave 0; Wave 1 fills the mean-bucket implementation"
  - "All RED scaffolds use source-inspection (?raw) pattern consistent with existing codebase convention"
  - "itoa2 helper added (not reusing itoa from no_autosave_test.go) to avoid duplicate declaration in same package"
  - "gen_progress_icons.go uses //go:build ignore so it is never compiled into the binary"
metrics:
  duration: "~40 minutes"
  completed_date: "2026-05-08"
  tasks: 3
  files_created: 10
  files_modified: 11
  commits: 3
---

# Phase 98 Plan 01: Wave 0 Foundation Summary

**One-liner:** Vendored @xterm/addon-progress@0.2.0 through the full Phase 93 pipeline, authored 20 RED scaffolds across 5 test files, generated 4 TokyoNight quartile PNG assets, and shipped the v3.3-flip italic caption — wails build green, cuttability invariant intact.

## Vendoring Pipeline Completion

All 7 steps of the Phase 93/97 vendoring discipline are complete:

| Step | File | Status |
|------|------|--------|
| A. pnpm add | frontend/package.json + pnpm-lock.yaml | Done — `"@xterm/addon-progress": "0.2.0"` in dependencies |
| B. UMD copy | web/vendor/xterm/addons/addon-progress.js | Done — byte-identical to node_modules copy |
| C. VERSION manifest | web/vendor/xterm/VERSION | Done — 10 lines, addon-progress@0.2.0 appended |
| D. embed.go | web/embed.go | Done — `vendor/xterm/addons/addon-progress.js` on line 11 |
| E. terminal.html | web/terminal.html | Done — script tag after addon-serialize.js |
| F. drift-test bump | internal/webserver/vendor_drift_test.go | Done — min-count 9→10, addon-progress in provenance |
| G. wails build | `wails build -tags wailsassets` | VERIFIED GREEN |

## PluginsSection v3.3-Flip Caption

The Progress toggle row now carries the verbatim italic caption:

```
'Default OFF in v3.2 — flips ON in v3.3 after field validation.'
```

This satisfies PRG-01 visible-affordance. No CSS change was needed — `settings-panel__description--italic` class is already styled by Phase 93.

## RED Scaffolds Authored (Wave 0 → future waves flip GREEN)

### Frontend (source-inspection ?raw pattern)

| File | Count | Tags | Flips GREEN at |
|------|-------|------|----------------|
| PluginsSection.test.tsx | 3 GREEN | v3.3-flip caption | Already GREEN |
| aggregateProgress.test.ts | 7 RED + 2 GREEN | (9 total) | Wave 1 (Plan 02) |
| TerminalPanel.test.tsx | 5 RED | progress-hot-swap, progress-onchange-forward | Wave 2 (Plan 03) |
| TabBar.test.tsx | 3 RED | progress-underline, progress-transform | Wave 3 (Plan 04) |
| App.test.tsx | 5 RED | progress-debounce | Wave 2 (Plan 03) |

### Go backend

| File | Test | Status | Flips at |
|------|------|--------|----------|
| no_progress_when_off_test.go | TestPRG_OffPath_NoProgressLogic | GREEN | (stays green — no-op guard) |
| no_progress_when_off_test.go | TestPRG_NewProgressAddonIsGated | GREEN | (stays green — no-op guard) |
| no_progress_when_off_test.go | TestPRG_SetTrayProgressUsage | RED | Wave 1 (Plan 02 Task 2) |

## `wails build -tags wailsassets` Verification

Build succeeded without errors. The frontend was built first (`pnpm build`) then wails build ran clean.

## Cuttability Invariant

With every later wave dropped (Waves 1–4), the binary is behaviorally identical to Phase 97 except:
- The Progress toggle is visible with the v3.3-flip italic caption
- The addon UMD is vendored (web parity)
- Zero addon construction anywhere
- Zero OSC 9;4 handler anywhere
- Zero SetTrayProgress callsite anywhere

The `TestPRG_OffPath_NoProgressLogic` and `TestPRG_NewProgressAddonIsGated` tests enforce these invariants in CI.

## OSC 9;4 Shell Fixture

`tests/fixtures/osc94-progress-fixture.sh` is an executable bash script that emits 4 progress updates (25/50/75/100%) then clears. Used for manual UAT and future Playwright e2e seeding.

## Tray Quartile PNGs

Four 18×18 RGBA PNG glyphs generated by `build/gen_progress_icons.go`:

| File | Bar width | Color |
|------|-----------|-------|
| tray_icon_progress_25.png | 5px | #7aa2f7 (TokyoNight accent) |
| tray_icon_progress_50.png | 9px | #7aa2f7 |
| tray_icon_progress_75.png | 13px | #7aa2f7 |
| tray_icon_progress_100.png | 18px (full) | #7aa2f7 |

All verified 18×18 by `file` command. Ready for Wave 1 `//go:embed` directives.

## Deviations from Plan

### None — plan executed exactly as written.

The only minor deviation: `itoa2` was added to `no_progress_when_off_test.go` (named distinctly from `itoa` in `no_autosave_test.go`) to avoid a duplicate-declaration error in the same `release_test` package. This is a correctness requirement, not a plan deviation.

## Known Stubs

| File | Stub | Resolved at |
|------|------|-------------|
| frontend/src/lib/aggregateProgress.ts | Returns 0 for all inputs | Wave 1 (Plan 02 Task 1) |

The stub correctly satisfies the 2 test cases where the expected result is 0 (empty registry, all-cleared). The other 7 test cases are RED as expected.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| supply_chain | web/vendor/xterm/addons/addon-progress.js | Vendored UMD at trust boundary — mitigated by vendor_drift_test.go gate (min-count now 10, includes addon-progress in provenance). CI fails red if pnpm-lock.yaml and VERSION disagree. |

## Self-Check

- [x] Task 1 commit aa958f4 exists: `feat(98-01): vendor @xterm/addon-progress@0.2.0 through full pipeline`
- [x] Task 2 commit c5b9c28 exists: `feat(98-01): PluginsSection v3.3-flip caption + Wave 0 RED frontend scaffolds`
- [x] Task 3 commit a8b0ba1 exists: `feat(98-01): backend RED scaffold + OSC 9;4 fixture + 4 tray PNG quartile glyphs`
- [x] web/vendor/xterm/addons/addon-progress.js exists
- [x] web/vendor/xterm/VERSION has 10 lines ending with @xterm/addon-progress@0.2.0
- [x] web/embed.go includes addon-progress.js
- [x] web/terminal.html has script tag for addon-progress.js
- [x] internal/webserver/vendor_drift_test.go uses `< 10`
- [x] frontend/src/lib/aggregateProgress.ts exists (stub)
- [x] frontend/src/lib/__tests__/aggregateProgress.test.ts has 9 test cases
- [x] PluginsSection.tsx has v3.3-flip caption
- [x] All 4 PNG files exist as 18×18 images
- [x] OSC 9;4 fixture is executable bash
- [x] no_progress_when_off_test.go defines 3 tests (2 GREEN, 1 RED)

## Self-Check: PASSED
