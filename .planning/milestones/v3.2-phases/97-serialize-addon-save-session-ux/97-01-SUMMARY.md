---
phase: 97
plan: "01"
subsystem: frontend-addon-infra
tags: [phase-97, serialize, scaffolding, vendor-unblock, wave-0, red-scaffold, green-now]
dependency_graph:
  requires: [phase-96]
  provides: [addon-serialize-vendored, wave-0-scaffolds, SER-03-regression-lock]
  affects: [vendor_drift_test, plugin_settings_test, frontend-test-scaffolds]
tech_stack:
  added: ["@xterm/addon-serialize@0.14.0"]
  patterns: [vendored-UMD-bundle, RED-scaffold-expect-fail, GREEN-now-regression-lock, filepath-Walk-regex-scan]
key_files:
  created:
    - frontend/src/lib/__tests__/stripAnsi.test.ts
    - frontend/src/lib/__tests__/sanitizeFilename.test.ts
    - frontend/src/__tests__/App.saver.test.tsx
    - internal/release/no_autosave_test.go
    - app_save_terminal_test.go
    - web/vendor/xterm/addons/addon-serialize.js
  modified:
    - frontend/package.json
    - frontend/pnpm-lock.yaml
    - web/vendor/xterm/VERSION
    - internal/webserver/vendor_drift_test.go
    - internal/daemon/plugin_settings_test.go
    - frontend/src/components/__tests__/TabBar.test.tsx
    - frontend/src/components/__tests__/TerminalPanel.test.tsx
    - frontend/src/components/__tests__/PluginsSection.test.tsx
    - frontend/src/components/FindBar/__tests__/FindBar.animation.test.tsx
decisions:
  - "@xterm/addon-serialize@0.14.0 promoted to runtime dependency (zero transitive deps); UMD CJS bundle vendored byte-identically to web/vendor/xterm/addons/addon-serialize.js"
  - "vendor_drift_test.go min-count bumped 8→9 in lockstep with VERSION 9th line — drift gate stays GREEN throughout Wave 0"
  - "SER-03 regression test (no_autosave_test.go) extends node_modules skip list to include frontend/node_modules (pnpm workspace path) to prevent false-positive matches from @types/react and react-dom dev packages"
  - "TestDefaultPluginSettings SER-02 citation: error message updated + 8-line comment block added above existing assertion (no assertion logic changes — assertion pre-existed from Phase 96)"
metrics:
  duration: "~32 minutes"
  completed: "2026-05-07"
  tasks: 2
  files_created: 9
  files_modified: 9
---

# Phase 97 Plan 01: Wave 0 Foundation Summary

One-liner: Vendored @xterm/addon-serialize@0.14.0 UMD bundle with drift-gate lockstep, SER-03 GREEN-NOW regression lock, and 10 RED scaffold targets unblocking Plans 97-02 through 97-06.

## What Was Built

### Task 1: Runtime dep promotion + vendor bundle + drift gate + SER-02 lock

**Dependency promotion** — `@xterm/addon-serialize@^0.14.0` added to `frontend/package.json` `dependencies` block (alphabetical between `@xterm/addon-search` and `@xterm/addon-unicode11`). `pnpm-lock.yaml` resolves at `0.14.0` with zero transitive runtime dependencies.

**Vendor bundle** — `web/vendor/xterm/addons/addon-serialize.js` is a byte-identical copy of `frontend/node_modules/@xterm/addon-serialize/lib/addon-serialize.js` (UMD CJS bundle; assigns `t.SerializeAddon=e()` on globalThis). The `.mjs` ESM variant was not used — same-origin `<script>` tags require the UMD build. `cmp -s` confirms byte identity.

**VERSION update** — `web/vendor/xterm/VERSION` now has 9 lines; 9th = `@xterm/addon-serialize@0.14.0`. The version-parity assertion in `TestXtermVendorVersionsMatchPnpmLock` auto-feeds from both files; no manual assertion code needed.

**Drift-test min-count bump** — `internal/webserver/vendor_drift_test.go` line 34: `< 8` → `< 9`; error message updated to list `addon-serialize` and cite `Phase 97 SER-03`. `go test ./internal/webserver/ -run TestXtermVendorVersions` passes GREEN.

**SER-02 default lock** — `TestDefaultPluginSettings` existing assertion at lines 28-29 (`!s.Serialize`) updated: error message now cites "Phase 97 SER-02 lock-in"; 8-line comment block added explaining the toggle semantics. No struct changes to `plugin_settings.go` (field and default already existed from Phase 96).

### Task 2: Wave 0 RED+GREEN scaffolds

**GREEN-NOW (SER-03 invariant lock):**

`internal/release/no_autosave_test.go` — two tests pass immediately:
- `TestSER03_NoAutoSavePatterns`: filepath.Walk over Go + TS/TSX/JS source files; 8 forbidden regex patterns (setInterval/setTimeout/BeforeQuit/OnShutdown scheduling serialize(), plus auto-save/auto-export/auto-capture/save-on-X vocabulary). Scans 0.4s. Forever-defense: any future auto-save wiring will fail CI.
- `TestSER03_NoAutoSettingsField`: reads `plugin_settings.go` directly; asserts no `json:"autoSave"` / `json:"autoExport"` / `json:"autoCapture"` / `json:"saveOnX"` fields exist.

**RED-SKIP scaffold (SER-03 dialog-call-site):**
- `TestSER03_OnlySaveTerminalSessionInAppGo` in same file: `t.Skip` until Plan 97-05 lands `(*App).SaveTerminalSession`. Plan 97-05 unskips by removing the `t.Skip` line; the body then asserts exactly one `Save*(Session|Terminal|Scrollback)` method in `app.go` named `SaveTerminalSession`.

**RED-SKIP scaffold (Wails RPC save):**
- `app_save_terminal_test.go`: `TestSaveTerminalSession` `t.Skip` scaffold with 4 sub-cases documented in comment (cancel path, normal write, IO error, dialog setup error). Plan 97-05 implements.

**RED frontend scaffolds (expect.fail — each cites downstream Plan):**

| File | Tests | Unblocks |
|------|-------|---------|
| `frontend/src/lib/__tests__/stripAnsi.test.ts` | 6 RED | Plan 97-02 |
| `frontend/src/lib/__tests__/sanitizeFilename.test.ts` | 6 RED | Plan 97-02 |
| `frontend/src/__tests__/App.saver.test.tsx` | 9 RED | Plans 97-03, 97-04 |
| `TabBar.test.tsx` (extended) | 3 RED | Plan 97-04 |
| `TerminalPanel.test.tsx` (extended) | 9 RED | Plan 97-04 |
| `PluginsSection.test.tsx` (extended) | 1 RED | Plan 97-05 |

Total: 43 Plan 97-XX markers across 6 frontend test files (VALIDATION.md minimum: ≥20).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-existing unused `beforeEach` import in FindBar.animation.test.tsx**
- **Found during:** Task 1 — `pnpm tsc --noEmit` reported TS6133 on a pre-existing file
- **Issue:** `FindBar.animation.test.tsx` imported `beforeEach` from vitest but never used it; this caused `pnpm tsc --noEmit` to exit non-zero, blocking the plan's acceptance criterion
- **Fix:** Removed `beforeEach` from the import destructure on line 15
- **Files modified:** `frontend/src/components/FindBar/__tests__/FindBar.animation.test.tsx`
- **Commit:** 782361b

**2. [Rule 1 - Bug] SER-03 regression test false-positives from frontend/node_modules**
- **Found during:** Task 2 — `go test ./internal/release/...` reported violations in `@types/react`, `react-dom`, and `playwright` dev package files
- **Issue:** The skip list included `"node_modules"` (top-level) but not `"frontend/node_modules"` (pnpm workspace path); `@types/react` index.d.ts and react-dom dev bundles contain `auto-save` text in non-project code
- **Fix:** Extended `skipDirs` map to include `"frontend/node_modules": true`
- **Files modified:** `internal/release/no_autosave_test.go`
- **Commit:** 9489ddb

## Known Stubs

None — Wave 0 is infrastructure-only (vendoring + scaffolds). No UI or runtime behavior shipped in this plan.

## Threat Flags

None — plan's threat model analyzed: addon-serialize has zero CSP-relevant constructs (no Worker/blob/eval); `vendor_drift_test.go` gate enforces version parity in CI; `no_autosave_test.go` provides forever-defense against auto-save regressions.

## Self-Check

Files created/modified exist on disk:
- `web/vendor/xterm/addons/addon-serialize.js` — FOUND
- `web/vendor/xterm/VERSION` (9 lines) — FOUND
- `internal/webserver/vendor_drift_test.go` (< 9 guard) — FOUND
- `internal/daemon/plugin_settings_test.go` (SER-02 citation) — FOUND
- `frontend/src/lib/__tests__/stripAnsi.test.ts` — FOUND
- `frontend/src/lib/__tests__/sanitizeFilename.test.ts` — FOUND
- `frontend/src/__tests__/App.saver.test.tsx` — FOUND
- `internal/release/no_autosave_test.go` — FOUND
- `app_save_terminal_test.go` — FOUND

Commits: 782361b (Task 1), 9489ddb (Task 2)

## Self-Check: PASSED
