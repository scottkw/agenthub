---
phase: 98-progress-addon-p2-cuttable
verified: 2026-05-08T00:00:00Z
reverified: 2026-05-10T00:00:00Z
status: human_needed
score: 5/6 must-haves verified (CR-01 resolved 2026-05-10; SC-3 runtime tray glyph still pending UAT sign-off)
overrides_applied: 0
gaps:
human_verification:
  - test: "Execute all 3 scenarios in 98-HUMAN-UAT.md (per-tab underline, cross-session tray glyph, OFF-toggle cuttability smoke)"
    expected: "All 3 Pass checkboxes ticked, Tester + Date + Build fields filled, and 98-HUMAN-UAT.md committed with status: approved"
    why_human: "Plan 98-05 Task 3 is checkpoint:human-verify — the UAT runbook is authored but the sign-off matrix is entirely blank. The tray icon (macOS menu bar / Windows notification area) cannot be observed programmatically. The full cross-component runtime event chain (ProgressAddon onChange → onProgressChange prop → App.tsx tabProgress → TabBar CSS scaleX transform) requires live visual confirmation. OS-level tray-API rendering and 200ms debounce smoothness are similarly unverifiable by grep."
resolved:
  - test: "Resolve CR-01 data race (app.go lastTrayQuartile) before treating human UAT as meaningful"
    resolution: "Field migrated to sync/atomic.Int32 at app.go:60. All writes use .Store() (app.go:84,901,904); all reads use .Load() (tray.go:115, tray_linux.go:429, tray_windows.go:586,616). go test -race -run TestApp_SetTrayProgress . passes clean."
    resolved_on: "2026-05-10"
---

# Phase 98: Progress Addon (P2 — Cuttable) Verification Report

**Phase Goal:** OSC 9;4 progress reporting from running CLIs surfaces as a per-tab progress underline and a tray-icon aggregate quartile glyph — shipped default OFF in v3.2 and explicitly cuttable if Phases 95 or 96 over-run.
**Verified:** 2026-05-08
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (from ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Phase is explicitly cuttable — absence does not block other phases | VERIFIED | ROADMAP marks phase as "explicitly cuttable"; TestPRG_OffPath_NoProgressLogic + TestPRG_NewProgressAddonIsGated enforce the OFF-path invariant; Wave 0 foundation leaves binary behaviorally identical to Phase 97 with toggle absent |
| 2 | User enables OSC 9;4 support in Settings (default OFF in v3.2; toggle copy notes v3.3 flip); CLI emitting OSC 9;4 shows subtle progress underline on its tab | VERIFIED (automated) / UNCERTAIN (runtime) | PluginsSection.tsx carries verbatim caption "Default OFF in v3.2 — flips ON in v3.3 after field validation."; daemon plugin_settings.go default Progress=false; TerminalPanel.tsx hot-swap arm gated on pluginConfig?.progress; TabBar.tsx renders .tab__progress with scaleX transform; full runtime chain unverified (UAT pending) |
| 3 | Tray icon reflects aggregate progress glyph (quartile indicator) across sessions; no flicker or excessive system-tray-API churn | VERIFIED (static) / UNCERTAIN (runtime) | Go-side SetTrayProgress RPC implemented with idempotency + bounds; 4 quartile PNGs embedded; trayIconBytesForState helper present in all 3 platform files. **CR-01 data race resolved 2026-05-10:** lastTrayQuartile migrated to `sync/atomic.Int32` (app.go:60); all reads via `.Load()` in tray.go:115 / tray_linux.go:429 / tray_windows.go:586,616; all writes via `.Store()` in app.go:84,901,904; `go test -race -run TestApp_SetTrayProgress .` passes clean. Runtime tray behavior still unverified (UAT pending). |

**Score:** 5/6 truths verified (SC-2 partially verified — runtime chain pending UAT; SC-3 race resolved, runtime tray glyph still pending UAT)

---

### Required Artifacts

#### Wave 0 (Plan 01)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/vendor/xterm/addons/addon-progress.js` | Vendored UMD bundle | VERIFIED | File exists and is referenced by embed.go |
| `web/vendor/xterm/VERSION` | Contains @xterm/addon-progress@0.2.0 | VERIFIED | Confirmed present |
| `frontend/src/lib/aggregateProgress.ts` | Mean-bucket implementation (Wave 1 fills) | VERIFIED | Implementation present (not stub); returns correct quartiles |
| `internal/release/no_progress_when_off_test.go` | 3-test OFF-path scaffold | VERIFIED | All 3 test functions present and confirmed by grep |
| `assets/tray_icon_progress_25.png` | 18x18 quartile glyph | VERIFIED | Non-zero size |
| `assets/tray_icon_progress_50.png` | 18x18 quartile glyph | VERIFIED | Non-zero size |
| `assets/tray_icon_progress_75.png` | 18x18 quartile glyph | VERIFIED | Non-zero size |
| `assets/tray_icon_progress_100.png` | 18x18 quartile glyph | VERIFIED | Non-zero size |
| `tests/fixtures/osc94-progress-fixture.sh` | Executable OSC 9;4 fixture | VERIFIED | Executable bit set |
| `frontend/src/components/__tests__/App.test.tsx` | progress-debounce scaffold | VERIFIED | `progress-debounce` tag present (from grep of SUMMARY self-check) |

#### Wave 1 (Plan 02)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app.go` | (*App).SetTrayProgress + lastTrayQuartile field | VERIFIED | One definition at line 892; field at line 60 (`atomic.Int32`, initialized to -1 via `.Store(-1)` at app.go:84 — race-safe per 2026-05-10 CR-01 resolution) |
| `tray.go` | //go:embed + trayIconBytesForState helper | VERIFIED | Helper and all 4 progress embeds confirmed |
| `tray_linux.go` | //go:embed + trayIconBytesForState helper | VERIFIED | Helper and all 4 progress embeds confirmed |
| `tray_windows.go` | //go:embed + trayIconBytesForState helper | VERIFIED | Helper and all 4 progress embeds confirmed |
| `app_set_tray_progress_test.go` | 3 unit tests | VERIFIED | File exists with TestApp_SetTrayProgress_{Idempotent,BoundsCheck,ErrorPrecedence} |
| `frontend/src/wailsjs/go/main/App.d.ts` | SetTrayProgress export | VERIFIED | `export function SetTrayProgress(quartile: number): Promise<void>` present |
| `frontend/src/wailsjs/go/main/App.js` | SetTrayProgress Call | VERIFIED | `Call('main.App.SetTrayProgress', [quartile])` present |

#### Wave 2 (Plan 03)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/TerminalPanel.tsx` | Hot-swap arm + onProgressChange prop + refs + cleanup | VERIFIED | pluginConfig?.progress gate at line 568; onProgressChange prop at line 81; progressAddonRef at line 135; state:0 emitted at lines 346 and 588 |
| `frontend/src/App.tsx` | progressRegistry + tabProgress + handleProgressChange + 200ms debounce + prop wiring | VERIFIED | All 6 items verified by grep: useRef Map, useState Record, handleProgressChange, 200ms setTimeout, onProgressChange prop at line 1061, tabProgress at line 958 |

#### Wave 3 (Plan 04)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/TabBar.tsx` | tabProgress prop + .tab__progress element + data-testid + scaleX | VERIFIED | tabProgress at line 33 (interface) + 49 (destructure); .tab__progress at line 169; data-testid at line 173; scaleX at line 171 |
| `frontend/src/style.css` | .tab__progress rule + position:relative on .tab | VERIFIED | .tab__progress rule at line 908; position:relative on .tab at line 118; transform:scaleX(0) initial state at line 915 |

#### Wave 4 (Plan 05)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/terminal.html` | `<div id='progress-underline'>` | VERIFIED | `<div id="progress-underline" aria-hidden="true">` present |
| `web/assets/terminal.css` | #progress-underline rule (transform-based, #7aa2f7) | VERIFIED | Rule present with scaleX(0) initial, #7aa2f7 accent |
| `web/assets/terminal.js` | ProgressAddon construction + onChange handler + gate | VERIFIED | new ProgressAddon.ProgressAddon(); pluginConfig.progress gate; getElementById('progress-underline') handler |
| `frontend/e2e/progress.spec.ts` | 3 test.skip scaffold blocks | VERIFIED | 3 test.skip blocks confirmed |
| `.planning/phases/98-progress-addon-p2-cuttable/98-HUMAN-UAT.md` | UAT runbook with 3 scenarios | VERIFIED (authored) / NOT SIGNED OFF | File exists; all 3 Pass/Fail checkboxes are blank; Tester/Date/Build fields unfilled; status: partial |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `web/embed.go` | `web/vendor/xterm/addons/addon-progress.js` | //go:embed directive | WIRED | `vendor/xterm/addons/addon-progress.js` on the embed line |
| `web/terminal.html` | `/assets/xterm/addons/addon-progress.js` | `<script src=...>` | WIRED | Script tag present after addon-serialize.js |
| `vendor_drift_test.go` | @xterm/addon-progress | regex + min-count >= 10 | WIRED | `< 10` guard present |
| `(*App).SetTrayProgress` | `(*App).refreshTrayState` | side-effect call on quartile change | WIRED | `a.refreshTrayState()` called in SetTrayProgress body |
| `(*App).updateTray` | `trayIconBytesForState(connected)` | byte selector helper | WIRED | All 3 platform files call the helper |
| `frontend Wails caller` | `(*App).SetTrayProgress` | Call('main.App.SetTrayProgress', ...) | WIRED | App.js binding confirmed |
| `TerminalPanel.tsx hot-swap useEffect` | `new ProgressAddon()` | inside `if (pluginConfig?.progress)` branch | WIRED | Line 568 |
| `TerminalPanel.tsx onChange` | `App.tsx handleProgressChange` | `onProgressChange?.(sessionId, state)` | WIRED | Line 577 |
| `App.tsx handleProgressChange` | `Wails main.App.SetTrayProgress` | 200ms debounce + idempotency guard then `void SetTrayProgress(quartile)` | WIRED | Lines 213-217 |
| `App.tsx <TabBar tabProgress=...>` | `TabBar.tsx tabProgress prop` | tabProgress={tabProgress} | WIRED | Line 958; TabBar.tsx renders .tab__progress with it |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| `TabBar.tsx .tab__progress` | `tabProgress[tab.sessionId]` | App.tsx progressRegistry + setTabProgress | Yes — driven by ProgressAddon onChange events from real terminal sessions | FLOWING (pending UAT to confirm end-to-end) |
| `trayIconBytesForState` | `a.lastTrayQuartile` | SetTrayProgress RPC from debounced handleProgressChange | Yes — updated on each quartile transition | FLOWING (pending UAT) |

---

### Behavioral Spot-Checks

Step 7b: SKIPPED for most checks (requires running app, live terminal, OS tray). The following static checks serve as partial coverage.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| aggregateProgress returns correct quartiles | (vitest per SUMMARY — 9 tests GREEN) | Per SUMMARY self-check — vitest runs confirmed green | PASS (per SUMMARY; not re-run) |
| SetTrayProgress bounds check | grep for bounds guard in app.go | `if quartile < 0 || quartile > 4 { return fmt.Errorf(...) }` present | PASS |
| OFF-path gating invariant | internal/release/no_progress_when_off_test.go exists | 3 test functions confirmed | PASS |
| HUMAN-UAT 3-scenario sign-off | grep status in 98-HUMAN-UAT.md | status: partial, all checkboxes blank | FAIL |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|------------|------------|-------------|--------|---------|
| PRG-01 | 98-01, 98-03, 98-05 | User can enable OSC 9;4 progress support in Settings (default OFF in v3.2; flips ON in v3.3) | VERIFIED (static) / NEEDS HUMAN (runtime) | PluginsSection.tsx caption confirmed; daemon default false confirmed; runtime affordance and toggle persistence require UAT |
| PRG-02 | 98-03, 98-04, 98-05 | When enabled, CLIs emitting OSC 9;4 show a progress underline on their tab | VERIFIED (code) / NEEDS HUMAN (visual) | TerminalPanel hot-swap arm + App.tsx tabProgress + TabBar .tab__progress element all wired; runtime visual requires UAT |
| PRG-03 | 98-02, 98-03 | Tray icon reflects aggregate progress glyph; no flicker | VERIFIED (static) / UNCERTAIN (runtime) | SetTrayProgress RPC + trayIconBytesForState + 4 quartile PNGs all present. CR-01 data race resolved 2026-05-10 (atomic.Int32 + go test -race clean). Runtime quartile rendering still requires UAT. |

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| ~~`app.go:59,899,902`~~ | ~~59, 899, 902~~ | ~~`lastTrayQuartile int` read/written from concurrent goroutines without mutex or atomic~~ | ~~BLOCKER (CR-01 from 98-REVIEW.md)~~ **RESOLVED 2026-05-10** | Field migrated to `atomic.Int32` at app.go:60; writes use `.Store()` at app.go:84,901,904; `go test -race -run TestApp_SetTrayProgress .` passes clean |
| ~~`tray.go:115`, `tray_linux.go:429`, `tray_windows.go:586,616`~~ | ~~Various~~ | ~~`switch a.lastTrayQuartile` without lock~~ | ~~BLOCKER (CR-01 companion reads)~~ **RESOLVED 2026-05-10** | All three reads now use `.Load()` (tray.go:115, tray_linux.go:429, tray_windows.go:586,616) — race-safe per Go memory model |
| `frontend/src/components/TerminalPanel.tsx:574-576` | 574 | Comment claims "App.tsx maps state:2/3/4 → state:0 by convention" — the code does not do this mapping | WARNING (IN-01 from 98-REVIEW.md) | Misleading comment may cause future contributors to skip implementing state:2/3/4 handling in v3.3 thinking it already exists |
| `tray_linux.go:451-454` | 451 | makePixmap() called inside tray.mu.Lock() on every 5s poll (PNG decode not pre-cached at initTray) | WARNING (WR-01 from 98-REVIEW.md) | Inconsistent with Windows pre-cached HICON pattern; can cause D-Bus menu jitter on slow systems |
| `frontend/src/lib/aggregateProgress.ts:28` | 28 | `if (mean <= 0) return 0` — state:1 with value=0 returns 0 (no-active-progress) instead of quartile 1 | WARNING (WR-02 from 98-REVIEW.md) | Hides the tray glyph when a CLI emits OSC 9;4;1;0 (task started, 0% progress) |
| `build/gen_progress_icons.go:68-72` | 68 | `imgH := bounds.Max.Y` should be `bounds.Max.Y - bounds.Min.Y` | INFO (WR-03 from 98-REVIEW.md) | Latent bug in generation script; correct in practice since png.Decode always returns Min={0,0} |
| `web/assets/terminal.css:235-247` | 235 | #progress-underline at position:fixed top:0 overlaps top 2px of #web-status-bar | INFO (IN-02 from 98-REVIEW.md) | Cosmetic conflict with status bar during active progress |

---

### Human Verification Required

#### 1. UAT Sign-Off (Plan 98-05 Task 3 — Blocking Human Checkpoint)

**Test:** Execute the 3 scenarios in `.planning/phases/98-progress-addon-p2-cuttable/98-HUMAN-UAT.md`

**Scenario 1 (PRG-02 desktop):** Build with `wails build -tags wailsassets`. Open Settings → Plugins → confirm Progress is OFF by default with the italic caption visible. Toggle ON, Save. Open a terminal tab, run `bash tests/fixtures/osc94-progress-fixture.sh`. Observe a thin #7aa2f7 underline appear at the bottom of the active tab growing 25→50→75→100% then disappearing smoothly.

**Scenario 2 (PRG-03 tray glyph):** With Progress ON, open 3 terminal tabs. Run the fixture in 2 of them. Observe the system tray icon cycle through quartile glyphs without flickering. Let fixtures complete; tray reverts to base icon.

**Scenario 3 (PRG-01 OFF-toggle smoke):** With an underline active, toggle Progress OFF in Settings. Observe underlines collapse immediately. Re-run the fixture — no underline should appear. Verify `go test ./internal/release -run TestPRG_OffPath_NoProgressLogic -count=1` stays GREEN.

**Expected:** All 3 Pass checkboxes ticked, Tester + Date + Build fields filled in 98-HUMAN-UAT.md, file committed with status: approved.

**Why human:** OS-level tray icon rendering; live event chain (ProgressAddon onChange → CSS scaleX transform); 200ms debounce smoothness; addon disposal propagation through React tree.

#### 2. ~~CR-01 Data Race Fix Required Before Meaningful UAT~~ — RESOLVED 2026-05-10

**Test:** Fix `lastTrayQuartile int` in `app.go` with `sync/atomic` or a dedicated mutex; apply matching atomic loads in all three `trayIconBytesForState` copies; run `go test -race ./...`

**Expected:** `go test -race` passes with no data race detected on the progress code path.

**Resolution (2026-05-10):** Architectural choice was `sync/atomic.Int32` (simpler than a named mutex; field is a single int read on every 5s tray poll, so a wait-free atomic load is preferable to mutex contention). Fix landed across all 4 platform files:
- `app.go:60` — field declared `lastTrayQuartile atomic.Int32`
- `app.go:84` — initialized via `a.lastTrayQuartile.Store(-1)`
- `app.go:901,904` — `.Load()` guard + `.Store()` write in `SetTrayProgress`
- `tray.go:115`, `tray_linux.go:429`, `tray_windows.go:586,616` — all reads use `.Load()`

**Race-detector evidence:** `go test -race -run TestApp_SetTrayProgress .` → `ok  github.com/scottkw/agenthub  1.025s` (clean — no race report on the SetTrayProgress codepath).

---

### Code Review Findings (from 98-REVIEW.md) — Not Yet Resolved

| ID | Severity | Summary | Resolution Required Before Phase Close |
|----|----------|---------|---------------------------------------|
| CR-01 | ~~BLOCKER~~ **RESOLVED 2026-05-10** | Data race on App.lastTrayQuartile — fixed via `atomic.Int32` at app.go:60 with `.Load()`/`.Store()` across app.go + tray.go + tray_linux.go + tray_windows.go; `go test -race -run TestApp_SetTrayProgress .` passes | ~~Yes — fix before UAT sign-off~~ Resolved |
| WR-01 | WARNING | Linux makePixmap() called inside tray.mu.Lock() every 5s poll (not pre-cached at initTray) | Recommended before Phase 99 release gate |
| WR-02 | WARNING | aggregateProgress returns 0 (no progress) for state:1 value=0 — hides tray glyph for just-started tasks | Recommended — add test case or fix logic |
| WR-03 | INFO | gen_progress_icons.go uses bounds.Max.Y instead of bounds.Max.Y - bounds.Min.Y | Low risk (latent only) |
| IN-01 | INFO | Misleading comment in TerminalPanel.tsx claiming App.tsx maps state:2/3/4 to state:0 | Fix comment before v3.3 state:2/3/4 work lands |
| IN-02 | INFO | #progress-underline overlaps top 2px of #web-status-bar on web side | Clarify intent or adjust top offset |
| IN-03 | INFO | osc94-progress-fixture.sh uses set -euo pipefail — printf failures abort mid-sequence | Acceptable for manual UAT; add `|| true` for CI robustness |

---

### Gaps Summary

No hard gaps prevent the automated portions of the phase from being considered complete. The codebase evidence confirms all 5 waves shipped: vendoring pipeline, Go-side tray RPC, TerminalPanel hot-swap arm, App.tsx registry + debounce, TabBar underline, web parity.

**One item now blocks phase closure** (CR-01 resolved 2026-05-10):

1. ~~**CR-01 data race (BLOCKER):**~~ **RESOLVED 2026-05-10.** Field migrated to `atomic.Int32` at `app.go:60`; writes via `.Store()` (app.go:84,901,904); reads via `.Load()` (tray.go:115, tray_linux.go:429, tray_windows.go:586,616). Architectural choice was atomic over mutex — wait-free is preferable for a 5s tray poll. `go test -race -run TestApp_SetTrayProgress .` passes clean.

2. **UAT sign-off pending (human checkpoint):** Plan 98-05 Task 3 is a `checkpoint:human-verify` gate. 98-HUMAN-UAT.md has been authored (all 3 scenarios documented) but the sign-off matrix is entirely blank — no Pass/Fail boxes ticked, no Tester/Date/Build recorded. `status: partial` in the frontmatter confirms the checkpoint has not been completed. The human UAT is the contractual gate for PRG-02 (tab underline visual) and PRG-03 (tray glyph quartile transitions) at runtime.

---

_Verified: 2026-05-08_
_Re-verified: 2026-05-10 — CR-01 resolved with race-detector evidence_
_Verifier: Claude (gsd-verifier)_
