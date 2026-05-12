---
status: complete
phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate
source: [99-01-SUMMARY.md, 99-02-SUMMARY.md, 99-03-SUMMARY.md, 99-04-SUMMARY.md, 99-05-SUMMARY.md]
started: 2026-05-12T04:08:00Z
updated: 2026-05-12T04:19:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: Kill any running AgentHub process. Run `wails build -tags wailsassets` (or `pnpm dev`). Launch AgentHub from scratch. App window opens, no panic/error in logs, the sessions list loads, and at least one terminal session can be created and renders its prompt.
result: pass

### 2. PUI-02 — Unicode 11 toggle banner
expected: Open Settings → Plugins. Toggle Unicode 11 OFF (it ships ON). Click Save Plugins. A one-shot BannerStack toast appears (above or near the top of the app) reading something like "Unicode 11 toggle applies to new sessions you create" — it has an × dismiss button and auto-dismisses around 6 seconds.
result: pass

### 3. PUI-02 — Inline Images toggle banner
expected: With the prior banner dismissed (or after it auto-dismissed), toggle Inline Images OFF. Click Save Plugins. A second BannerStack toast appears referencing Inline Images, same shape/dismiss behavior as the Unicode 11 toast.
result: pass

### 4. PUI-02 — Both toggles in one Save = two stacked banners
expected: Re-enable Unicode 11 AND Inline Images and Save (reset to known state). Then toggle BOTH OFF in one gesture and click Save Plugins once. TWO toasts stack simultaneously (one for unicode11, one for image), each with its own × dismiss and 6-second auto-dismiss timer.
result: pass

### 5. PUI-03 — Search disclosure
expected: In Settings → Plugins, the Search row has an inline expandable `<details>` element under the toggle. Expanding it reveals three checkboxes — Regex, Case sensitive, Whole word — bound to the Search plugin's default behavior. Other plugin rows that have no advanced config (WebGL, Unicode 11, Serialize, Clipboard, Progress) show no disclosure.
result: issue
reported: "No checkboxes... screenshot shows the Search defaults disclosure expanded with Regex / Case sensitive / Whole word as plain text labels but the <input type='checkbox'> controls themselves are missing. Same problem on the Web Links disclosure for Confirm OSC 8 / Confirm IDN / Confirm typosquat (only the modifier <select> renders correctly)."
severity: major

### 6. PUI-03 — Web Links disclosure
expected: The Web Links row has an inline `<details>` disclosure under its toggle. Expanding it reveals a `<select>` for the click modifier with options platform / cmd / ctrl / none, plus three checkboxes: Confirm OSC 8, Confirm IDN, Confirm typosquat.
result: issue
reported: "Screenshot confirms: modifier <select> dropdown ('Platform default (Cmd on macOS, Ctrl elsewhere)') renders correctly, but Confirm OSC 8 hyperlinks / Confirm IDN / Confirm typosquat patterns appear as label-only — no checkbox input rendered. Same bug pattern as Test 5."
severity: major

### 7. PUI-03 — Inline Images disclosure
expected: The Inline Images row has an inline `<details>` disclosure. Expanding it reveals a single number input (with "MB" suffix) labeled storage limit, accepting values in the range [1, 1000].
result: pass

### 8. PUI-04 — Disclosure changes persist without Save Plugins
expected: Expand the Web Links disclosure. Change the click modifier from its current value to a different one (e.g. platform → cmd). Do NOT click Save Plugins. Close and re-open Settings (or relaunch the app). The disclosure reflects the new modifier value — the sub-key RPC (SetWebLinksConfig) persisted immediately, no full-snapshot save was needed.
result: pass

### 9. SC-3 — Migration test green
expected: Run `go test ./internal/daemon/... -run TestSettingsMigration -count=1`. Both TestSettingsMigrationV3_1ToV3_2 and TestSettingsMigrationIdempotent PASS — the v3.1 fixture migrates to v3.2 with all 8 plugin booleans populated, schemaVersion: 2, and a second run is idempotent.
result: pass
auto_verified: "go test PASS - TestSettingsMigrationV3_1ToV3_2 (0.00s) + TestSettingsMigrationIdempotent (0.01s) — `ok github.com/scottkw/agenthub/internal/daemon 0.023s`"

### 10. SC-4 — Cross-browser CSP suite green
expected: From `frontend/`, run `pnpm exec playwright test web-csp.spec.ts`. All 3 browser projects (chromium, firefox, webkit) report 0 CSP violations during the attach/scroll session. Three passing tests total.
result: pass
auto_verified: "3 passed (15.2s) — chromium 3.9s + firefox 4.3s + webkit 4.1s; zero CSP violations across all three engines."

### 11. SC-4 — iPad Safari Tailscale UAT (real device)
expected: Execute the 5-scenario runbook in 99-iPad-UAT.md on a real iPad Safari over Tailscale (UAT-1 through UAT-5). Both screenshots captured (screenshots/99-iPad-UAT-3-zero-cdn.png and screenshots/99-iPad-UAT-4-zero-csp.png). Tester / Device / Date line filled in. All 5 sign-off checkboxes flipped to [x].
result: pass

## Summary

total: 11
passed: 9
issues: 2
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "Search disclosure renders three checkbox controls (Regex, Case sensitive, Whole word)"
  status: failed
  reason: "User reported: No checkboxes — screenshot shows the Search defaults disclosure expanded with option labels rendering as plain text but the <input type='checkbox'> controls themselves are missing."
  severity: major
  test: 5
  artifacts: []
  missing: []

- truth: "Web Links disclosure renders three checkbox controls (Confirm OSC 8, Confirm IDN, Confirm typosquat) alongside the working modifier <select>"
  status: failed
  reason: "User reported: Modifier <select> renders correctly but Confirm OSC 8 / Confirm IDN / Confirm typosquat labels show no checkbox input — same bug pattern as Test 5. Two screenshots confirm the regression spans both disclosures."
  severity: major
  test: 6
  artifacts: []
  missing: []
