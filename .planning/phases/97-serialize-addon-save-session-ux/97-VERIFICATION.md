---
phase: 97-serialize-addon-save-session-ux
verified: 2026-05-08T00:16:56Z
status: human_needed
score: 10/10
overrides_applied: 0
human_verification:
  - test: "Scenario 1 — SER-OFF: Open AgentHub, toggle Serialize OFF in Settings, right-click terminal tab, click 'Save Terminal As…' — confirm banner appears and NO native dialog opens, no file written."
    expected: "Info banner 'Enable the Serialize plugin in Settings to save sessions.' appears. No native OS save dialog opens. No file is written to disk."
    why_human: "Live affordance behavior across Settings toggle + right-click context menu + banner stack rendering cannot be exercised in headless CI."
  - test: "Scenario 2 — SER-ON: Toggle Serialize ON, run a command that produces colored output (e.g. echo -e '\\033[1;31mError\\033[0m: test'), right-click tab, click 'Save Terminal As…', confirm native dialog opens, save to ~/Desktop/agenthub-uat-test.txt, inspect file in TextEdit/cat."
    expected: "Native Save dialog opens with title containing 'Save Terminal As…'. Default filename is tab-name+timestamp+.txt. Saved file is plain UTF-8 text with NO \\x1b[ escape sequences. File contains visible scrollback."
    why_human: "Native OS dialog cannot be exercised in headless CI; visual confirmation of dialog UX, file write integrity, and ANSI-strip correctness require human inspection."
  - test: "Scenario 3 — CANCEL: With Serialize ON, right-click tab, click 'Save Terminal As…', press Esc (or click Cancel). Repeat with Cancel button."
    expected: "Dialog closes. NO error toast appears. NO file is written at the default path."
    why_human: "Real native dialog cancellation behavior (Esc key, Cancel button) is OS-specific and cannot be exercised in headless CI."
  - test: "Scenario 4 — SER-02 CAPTION: Open Settings → Plugins. Locate 'Save terminal as text' Serialize row. Visually confirm the italic caption."
    expected: "Caption reads VERBATIM (italic styling, positioned directly under the toggle, not a hover tooltip): 'Saved files include any secrets, tokens, or sensitive data printed in the session.'"
    why_human: "Visual/text-rendered verification of italic styling and positioning requires human eye — automated source-scan asserts the literal string is in source, but only a human confirms rendering."
---

# Phase 97: Serialize Addon + Save-Session UX — Verification Report

**Phase Goal:** User can right-click a terminal tab, choose "Save Terminal As…", and export the visible scrollback as a `.txt` file via Wails save dialog — with explicit secrets-warning copy and zero auto-save / zero on-disk capture without an explicit user gesture.
**Verified:** 2026-05-08T00:16:56Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User right-clicks a terminal tab → chooses "Save Terminal As…" → Wails SaveFileDialog opens → confirms path → .txt file written containing full visible scrollback (text-only) | ? HUMAN | TabBar.tsx has "Save Terminal As…" menu item wired to onRequestSave; App.tsx handleRequestSave calls stripAnsi(fn()) then SaveTerminalSession; app.go SaveTerminalSession calls runtime.SaveFileDialog and os.WriteFile — all source-verified and unit-tested. End-to-end native dialog behavior requires human UAT. |
| 2 | Settings tooltip on Serialize toggle reads (verbatim or near-verbatim) secrets warning; toggle defaults to ON; serialize never auto-saves or auto-runs | VERIFIED | PluginsSection.tsx line 140: verbatim caption "Saved files include any secrets, tokens, or sensitive data printed in the session." confirmed. TestDefaultPluginSettings PASSES (Serialize==true). TestSER03_NoAutoSavePatterns + TestSER03_NoAutoSettingsField + TestSER03_OnlySaveTerminalSessionInAppGo all PASS. |
| 3 | No on-disk capture without explicit user action — regression test confirms no timer-driven serialization, no graceful-shutdown serialization, no auto-save settings field | VERIFIED | internal/release/no_autosave_test.go runs GREEN: TestSER03_NoAutoSavePatterns (0.37s PASS), TestSER03_NoAutoSettingsField (PASS), TestSER03_OnlySaveTerminalSessionInAppGo (PASS). .claude and .claire directories properly skipped. |

**Score:** 10/10 truths verified (2 programmatically, 1 pending human UAT gate)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/lib/stripAnsi.ts` | Pure helper exporting stripAnsi(input): string | VERIFIED | 26 lines; regex `/\x1b\[\??[0-9;]*[a-zA-Z]/g`; no DOM/network/logging. 12 unit tests PASS. |
| `frontend/src/lib/sanitizeFilename.ts` | Pure helper exporting sanitizeFilename(name): string | VERIFIED | 34 lines; 4-step pipeline (trim→collapse→allowlist→reserved-guard); returns 'session' fallback. 12 unit tests PASS. |
| `web/vendor/xterm/addons/addon-serialize.js` | Vendored UMD CJS bundle of @xterm/addon-serialize@0.14.0 | VERIFIED | File exists (minified UMD bundle, 1 line). Registered in web/embed.go line 11. |
| `web/vendor/xterm/VERSION` | 9 lines; 9th = @xterm/addon-serialize@0.14.0 | VERIFIED | 9 lines confirmed. Line 9 = `@xterm/addon-serialize@0.14.0`. |
| `internal/webserver/vendor_drift_test.go` | min-count guard bumped 8→9; addon-serialize in error list | VERIFIED | Line 34: `if len(pnpmVersions) < 9`. Line 35 error message includes addon-serialize. TestXtermVendorVersionsMatchPnpmLock PASSES. |
| `internal/daemon/plugin_settings_test.go` | TestDefaultPluginSettings asserts Serialize==true | VERIFIED | Line 36: `t.Error("expected Serialize=true (Phase 97 SER-02 lock-in)")`. TestDefaultPluginSettings PASSES. |
| `frontend/src/lib/__tests__/stripAnsi.test.ts` | All 6 RED scaffolds → GREEN real assertions | VERIFIED | 6 test cases covering SGR/ECH/cursor-moves/DEC modes/plain-text/round-trip — 12 tests PASS. |
| `frontend/src/lib/__tests__/sanitizeFilename.test.ts` | All 6 RED scaffolds → GREEN real assertions | VERIFIED | 6 test cases covering traversal/empty/leading-dot/Windows-reserved/whitespace/preserved-chars — 12 tests PASS. |
| `frontend/src/__tests__/App.saver.test.tsx` | All 9 RED scaffolds → GREEN source-scan assertions | VERIFIED | 11 tests PASS. Import checks, state/callback declarations, prop passes, banner copy. |
| `frontend/src/components/__tests__/TabBar.test.tsx` | "Save Terminal As…" menu-item scaffolds GREEN | VERIFIED | 9 tests PASS including Save Terminal As… item. |
| `frontend/src/components/__tests__/TerminalPanel.test.tsx` | All 9 SerializeAddon RED scaffolds GREEN | VERIFIED | 50 tests PASS including SerializeAddon hot-swap arm + placement guard. |
| `frontend/src/components/__tests__/PluginsSection.test.tsx` | SER-02 verbatim caption assertion GREEN | VERIFIED | 15 tests PASS including verbatim secrets-warning assertion. |
| `internal/release/no_autosave_test.go` | SER-03 negative-grep test GREEN; .claude/.claire in skipDirs | VERIFIED | 3 tests PASS; .claude and .claire added to skipDirs (a93e879 fix). |
| `app_save_terminal_test.go` | 4 table-driven sub-cases GREEN | VERIFIED | TestSaveTerminalSession: cancel, normal write, IO error, dialog error — all 4 PASS. |
| `app.go` | SaveTerminalSession + saveFileDialogFunc injection field | VERIFIED | Lines 65-75: saveFileDialogFunc field on *App + defaults to runtime.SaveFileDialog. Lines 846-868: SaveTerminalSession method with dialog, cancellation-silent-success, WriteFile 0o644, error wrapping. |
| `frontend/src/wailsjs/go/main/App.d.ts` | SaveTerminalSession declaration | VERIFIED | Line 146: `export function SaveTerminalSession(defaultDir: string, defaultName: string, content: string): Promise<void>` |
| `frontend/src/wailsjs/go/main/App.js` | SaveTerminalSession Call() stub | VERIFIED | Line 89: `export const SaveTerminalSession = (defaultDir, defaultName, content) => Call('main.App.SaveTerminalSession', [defaultDir, defaultName, content])` |
| `frontend/src/components/PluginsSection.tsx` | Verbatim SER-02 secrets-warning caption as 4th arg | VERIFIED | Line 140: `'Saved files include any secrets, tokens, or sensitive data printed in the session.'` — verbatim match. |
| `frontend/src/components/TerminalPanel.tsx` | SerializeAddon hot-swap arm + onRegisterSaver prop | VERIFIED | Line 526: `new SerializeAddon()` in hot-swap useEffect (line 351). Lines 529/535: register/unregister calls. Line 321: mount-cleanup unregister. |
| `frontend/src/components/TabBar.tsx` | "Save Terminal As…" (U+2026) menu-item + onRequestSave prop | VERIFIED | Line 185: "Save Terminal As…" with U+2026 confirmed. Line 181: `onRequestSave?.(contextMenu.tabId)`. |
| `web/embed.go` | //go:embed includes vendor/xterm/addons/addon-serialize.js | VERIFIED | Line 11: `//go:embed vendor/xterm/addons/addon-image.js vendor/xterm/addons/addon-serialize.js` |
| `web/terminal.html` | script tag after addon-image, before terminal.js | VERIFIED | Line 50-51: addon-image.js then addon-serialize.js. Line 65: terminal.js. Correct load order. |
| `web/assets/terminal.js` | initTerminal() constructs SerializeAddon with pluginConfig.serialize gate | VERIFIED | Line 267: `if (pluginConfig.serialize)` gate. Line 269: `new SerializeAddon.SerializeAddon()`. Parity-only, no web Save UI (by design). |
| `.planning/phases/97-serialize-addon-save-session-ux/97-HUMAN-UAT.md` | 4 manual UAT scenarios with sign-off checklist | VERIFIED | 125 lines. 4 scenarios: SER-OFF banner, SER-ON dialog+file, CANCEL silence, SER-02 caption. Sign-off blanks present (unsigned — UAT pending). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| frontend/package.json (@xterm/addon-serialize^0.14.0 in dependencies) | frontend/node_modules/@xterm/addon-serialize/lib/addon-serialize.js | pnpm install | WIRED | pnpm-lock.yaml resolves 0.14.0 with integrity hash; package.json line 19 in dependencies (not devDependencies) |
| web/vendor/xterm/VERSION (9th line) | internal/webserver/vendor_drift_test.go (min-count >= 9) | drift-test guard | WIRED | TestXtermVendorVersionsMatchPnpmLock PASSES |
| internal/release/no_autosave_test.go | frontend/src/**/*.{ts,tsx} + **/*.go (filepath.Walk) | regex scan — locks absence of auto-save patterns | WIRED | TestSER03_NoAutoSavePatterns PASSES; .claude/.claire in skipDirs |
| TerminalPanel hot-swap useEffect (positive arm) | App.tsx serializerRegistry | onRegisterSaver?.(sessionId, () => addon.serialize({ excludeModes: true })) | WIRED | Line 529 in TerminalPanel.tsx |
| TerminalPanel hot-swap useEffect (negative arm) + mount cleanup | App.tsx serializerRegistry (entry removal) | onRegisterSaver?.(sessionId, null) | WIRED | Lines 321 and 535 in TerminalPanel.tsx |
| TabBar Save menu item onClick | App.tsx handleRequestSave | onRequestSave?.(contextMenu.tabId) | WIRED | Line 181 in TabBar.tsx |
| App.tsx handleRequestSave | (*App).SaveTerminalSession Wails method | import { SaveTerminalSession } from './wailsjs/go/main/App' | WIRED | App.js line 89 Call() stub; App.d.ts line 146 type; App.tsx line 202 invocation |
| web/vendor/xterm/addons/addon-serialize.js (vendored) | /assets/xterm/addons/addon-serialize.js (HTTP) | web/embed.go //go:embed + FileServerFS route | WIRED | embed.go line 11 includes addon-serialize.js |
| web/terminal.html script tag | global SerializeAddon UMD symbol | browser script load + UMD factory assignment | WIRED | Lines 50-51 confirm load order: addon-image → addon-serialize → terminal.js |
| web/assets/terminal.js initTerminal() | term.loadAddon(new SerializeAddon.SerializeAddon()) | UMD global construction (parity only) | WIRED | Line 269: `new SerializeAddon.SerializeAddon()` inside `if (pluginConfig.serialize)` gate |
| (*App).SaveTerminalSession (Go) | wailsjs/go/main/App.{d.ts,js} bindings (TS) | Wails Call() bridge | WIRED | App.js Call('main.App.SaveTerminalSession', [...]) |
| PluginsSection Serialize renderRow | rendered italic caption under toggle | renderRow 4th argument | WIRED | Line 140 PluginsSection.tsx; PluginsSection.test.tsx verbatim assertion PASSES |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| App.tsx handleRequestSave | serializerRegistry[tab.sessionId] | TerminalPanel calls onRegisterSaver?.(sessionId, () => addon.serialize({...})) on hot-swap attach | Yes — SerializeAddon.serialize() on a live terminal buffer | FLOWING |
| app.go SaveTerminalSession | path from saveFileDialogFunc | runtime.SaveFileDialog (mocked in tests, real in production) | Yes — real dialog path or "" on cancel | FLOWING |
| PluginsSection.tsx Serialize row | 4th arg to renderRow() | Literal string constant in source | Literal (by design — it is fixed copy, not dynamic) | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| SER-03 no-auto-save patterns scan | `go test ./internal/release/... -count=1 -v` | All 3 tests PASS (0.37s) | PASS |
| SaveTerminalSession unit (4 sub-cases) | `go test . -run TestSaveTerminalSession -count=1 -v` | All 4 sub-cases PASS (0.00s) | PASS |
| TestDefaultPluginSettings Serialize==true | `go test ./internal/daemon/... -run TestDefaultPluginSettings -count=1` | PASS | PASS |
| Vendor drift test min-count==9 | `go test ./internal/webserver/... -run TestXtermVendorVersions -count=1` | PASS | PASS |
| stripAnsi + sanitizeFilename unit tests | `pnpm exec vitest run src/lib/__tests__/stripAnsi.test.ts src/lib/__tests__/sanitizeFilename.test.ts` | 2 files, 12 tests PASS | PASS |
| App.saver source-scan tests | `pnpm exec vitest run src/__tests__/App.saver.test.tsx` | 1 file, 11 tests PASS | PASS |
| TabBar context-menu tests | `pnpm exec vitest run src/components/__tests__/TabBar.test.tsx` | 1 file, 9 tests PASS | PASS |
| TerminalPanel SerializeAddon tests | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | 1 file, 50 tests PASS | PASS |
| PluginsSection verbatim caption | `pnpm exec vitest run src/components/__tests__/PluginsSection.test.tsx` | 1 file, 15 tests PASS | PASS |
| Full root package Go tests | `go test . -count=1` | PASS (37.9s) | PASS |
| TypeScript compilation | `cd frontend && pnpm tsc --noEmit` | Zero errors | PASS |
| Full frontend test suite | `pnpm exec vitest run` (all 49 files) | 48 passed, 1 failed (Sidebar.test.tsx — 20 pre-existing failures pre-dating Phase 97 at commit 46ddd2a) | PASS (Phase 97 tests clean) |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SER-01 | 97-01, 97-02, 97-03, 97-04, 97-05, 97-06 | User can right-click a terminal tab and choose "Save Terminal As…" to export the full visible scrollback as a .txt file via a Wails save dialog | SATISFIED (automated) + HUMAN UAT PENDING | Full pipeline wired: TabBar → App.tsx → stripAnsi + sanitizeFilename → SaveTerminalSession → os.WriteFile. All unit tests PASS. Native dialog + end-to-end file inspection require human UAT. |
| SER-02 | 97-01, 97-05, 97-06 | A Settings tooltip on the Serialize toggle warns explicitly that saved files include any secrets, tokens, or sensitive data printed in the session | SATISFIED | PluginsSection.tsx line 140: verbatim string present. TestDefaultPluginSettings: Serialize==true PASS. HUMAN UAT Scenario 4 required for visual italic styling confirmation. |
| SER-03 | 97-01, 97-05, 97-06 | User can enable/disable serialize support in Settings; serialize never auto-saves or auto-runs | SATISFIED | TestSER03_NoAutoSavePatterns, TestSER03_NoAutoSettingsField, TestSER03_OnlySaveTerminalSessionInAppGo — all PASS. No auto-save patterns found in codebase. .claude/.claire skipDirs fix (a93e879) prevents false positives from agent worktrees. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| web/assets/terminal.js | 271 | `} catch (e) { /* addon UMD may not be present — silent */ }` | Info | Web parity block silently swallows SerializeAddon load failures; by design per 97-RESEARCH §"Web Parity Scope" — the addon is inert on web in v3.2. Not a blocker. |

No blocker anti-patterns found in Phase 97 code.

### Human Verification Required

**Plan 97-06 has a HUMAN UAT gate. The 97-HUMAN-UAT.md runbook is authored but signoff has NOT yet been provided.** The following 4 scenarios must be run by a human tester on a real `wails build -tags wailsassets` build:

#### 1. SER-OFF — Save menu shows banner when Serialize is disabled

**Test:** In Settings → Plugins, toggle Serialize OFF. Right-click terminal tab. Click "Save Terminal As…".
**Expected:** Info banner appears ("Enable the Serialize plugin in Settings to save sessions." or similar). NO native Save dialog opens. No file written.
**Why human:** Live affordance behavior across Settings toggle + context menu + banner stack rendering cannot be exercised in headless CI.

#### 2. SER-ON — Native Save dialog opens, file is plain text, content matches scrollback

**Test:** Toggle Serialize ON. Run `echo -e "\033[1;31mError\033[0m: test"`. Right-click tab → "Save Terminal As…". Save to ~/Desktop/agenthub-uat-test.txt. Open file in TextEdit/cat.
**Expected:** Native dialog opens with title containing "Save Terminal As…". Default filename = tab-name+timestamp+.txt. File is plain UTF-8 text, NO `\x1b[` sequences, "Error: test" appears without color codes. File contains visible scrollback.
**Why human:** Native OS dialog + file write integrity + ANSI-strip correctness require human inspection.

#### 3. CANCEL — User-cancellation writes no file and shows no error

**Test:** With Serialize ON, right-click tab → "Save Terminal As…". Press Esc. Repeat with Cancel button.
**Expected:** Dialog closes. No error toast. No file at default path.
**Why human:** Real native dialog cancellation behavior (Esc key, Cancel button) is OS-specific and not exercisable in headless CI.

#### 4. SER-02 CAPTION — Verbatim secrets warning visible in Settings, italic styled

**Test:** Open Settings → Plugins. Find "Save terminal as text" row.
**Expected:** Directly under the toggle, in italic styling: "Saved files include any secrets, tokens, or sensitive data printed in the session." (verbatim, not a hover tooltip, not paraphrased).
**Why human:** Visual/text-rendered verification of italic styling and exact positioning requires human eye — source-scan asserts the literal string is in source but does not confirm rendering.

### Gaps Summary

No automated gaps. All 10 must-have truths are VERIFIED programmatically where testable. The sole pending item is the Plan 97-06 human checkpoint for which the runbook exists at `.planning/phases/97-serialize-addon-save-session-ux/97-HUMAN-UAT.md` but signoff has not been provided.

**Pre-existing test failures confirmed NOT attributable to Phase 97:** Sidebar.test.tsx — 20 failures verified to pre-date Phase 97 at commit 46ddd2a before phase work began.

---

_Verified: 2026-05-08T00:16:56Z_
_Verifier: Claude (gsd-verifier)_
