---
status: complete
phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate
source: [99-01-SUMMARY.md, 99-02-SUMMARY.md, 99-03-SUMMARY.md, 99-04-SUMMARY.md, 99-05-SUMMARY.md, 99-06-SUMMARY.md]
started: 2026-05-12T04:08:00Z
updated: 2026-05-12T22:45:00Z
sign_off: "v3.2 release gate: 10/11 tests pass; Test 11 (iPad runbook) blocked on future shell-session feature, deferred to v3.3. v3.2 ships on automated coverage (vitest disclosure render tests + go settings migration tests + playwright cross-browser CSP suite)."
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
result: pass
auto_verified: "Fixed in 99-06 (commit 247a7b4) — dropped `settings-panel__toggle-input` class from all 6 disclosure checkbox inputs. Gap-closure render test `PluginsSection.disclosure.render.test.tsx` (commit 6cfe32d) renders the real DOM via createRoot+jsdom and asserts (1) 6 checkbox inputs exist inside `.settings-panel__details`, (2) NONE carry the hidden `.settings-panel__toggle-input` class, (3) the 8 main-row toggles STILL do (differential guardrail). All 3 it-blocks pass; RED→GREEN evidence in commit message."

### 6. PUI-03 — Web Links disclosure
expected: The Web Links row has an inline `<details>` disclosure under its toggle. Expanding it reveals a `<select>` for the click modifier with options platform / cmd / ctrl / none, plus three checkboxes: Confirm OSC 8, Confirm IDN, Confirm typosquat.
result: pass
auto_verified: "Same fix as Test 5 — 99-06 commit 247a7b4 removed the hidden class from the 3 Web Links confirmation checkboxes (Confirm OSC 8 / Confirm IDN / Confirm typosquat). Render test in commit 6cfe32d covers all 6 disclosure checkboxes including these three."

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
result: blocked
blocked_by: future-feature-terminal-sessions
reason: "Runbook UAT-1/UAT-2/UAT-5 emit chafa inline images and OSC 9;4 progress sequences from a shell PTY. AgentHub v3.2 ships agent sessions only — raw shell session type is deferred to v3.3+ per `.planning/v3.2-RELEASE-BLOCKERS.md` 'Backlog (from Phase 98 sign-off)'. UAT-3 + UAT-4 (zero-CDN, zero-CSP audits) are session-type-agnostic and remain runnable, but per 2026-05-12 decision the entire iPad runbook is deferred together to keep the gate atomic. Re-open in v3.3 once shell sessions land."
prior_state: "Was marked pass in earlier session (false-positive — no screenshots, no Tester/Device/Date filled in 99-iPad-UAT.md). Corrected 2026-05-12."

## Summary

total: 11
passed: 10
issues: 0
pending: 0
skipped: 0
blocked: 1

resolved_gaps:
  - test: 5
    closed_by: "99-06 fix (247a7b4) + render test (6cfe32d) — all 11 disclosure tests pass"
  - test: 6
    closed_by: "99-06 fix (247a7b4) + render test (6cfe32d) — same code path as Test 5"

deferred:
  - test: 11
    deferred_to: "v3.3 — once raw shell session type ships (per v3.2-RELEASE-BLOCKERS.md backlog)"
    reason: "iPad runbook UAT-1/2/5 require shell PTY for chafa + OSC 9;4 emission. AgentHub v3.2 ships agent sessions only."

## Gaps

- truth: "Search disclosure renders three checkbox controls (Regex, Case sensitive, Whole word)"
  status: failed
  reason: "User reported: No checkboxes — screenshot shows the Search defaults disclosure expanded with option labels rendering as plain text but the <input type='checkbox'> controls themselves are missing."
  severity: major
  test: 5
  root_cause: "The 6 disclosure checkbox inputs (3 in renderSearchDisclosure, 3 in renderWebLinksDisclosure) reuse className='settings-panel__toggle-input'. That class is a Phase-82 iOS-toggle-switch hider (style.css:586-592 collapses it to 1×1px opacity 0) intended as the semantic anchor for a paired visible .settings-panel__toggle-track + .settings-panel__toggle-thumb pair (rendered in renderRow). The disclosure helpers copied the hidden-input class but omitted the track/thumb spans, so each checkbox collapses to an invisible 1×1px element with no visible substitute. The disclosure <select> and number input do NOT carry that class and render correctly — confirming the class is the discriminator. Source-inspection tests pass because they grep for the 'checkbox' literal in the file string, not the rendered DOM or computed styles."
  artifacts:
    - path: "frontend/src/components/PluginsSection.tsx"
      issue: "renderSearchDisclosure (lines 162-203) puts className='settings-panel__toggle-input' on all 3 checkboxes without rendering the .settings-panel__toggle-track / __toggle-thumb visible counterparts"
    - path: "frontend/src/style.css"
      issue: "Lines 586-592 globally hide .settings-panel__toggle-input — no .settings-panel__details scope override exists"
    - path: "frontend/src/components/__tests__/PluginsSection.disclosure.test.tsx"
      issue: "Source-inspection test pattern (import raw + expect(raw).toContain) cannot catch CSS-driven visual regressions — gap in test strategy"
  missing:
    - "Pick a render strategy for disclosure checkboxes — option (a) drop the toggle-input class to fall back to native checkbox rendering (smallest diff, best UX inside a <details> config block), or (b) render matching track/thumb spans + --checked class to get iOS pill UI, or (c) add a .settings-panel__details-scoped CSS override that unsets the hiding"
    - "Add a real-DOM render test (vitest + @testing-library/react) or Playwright check for disclosure checkbox visibility so the same bug cannot pass tests again"
  debug_session: ".planning/debug/99-disclosure-checkboxes-missing.md"

- truth: "Web Links disclosure renders three checkbox controls (Confirm OSC 8, Confirm IDN, Confirm typosquat) alongside the working modifier <select>"
  status: failed
  reason: "User reported: Modifier <select> renders correctly but Confirm OSC 8 / Confirm IDN / Confirm typosquat labels show no checkbox input — same bug pattern as Test 5. Two screenshots confirm the regression spans both disclosures."
  severity: major
  test: 6
  root_cause: "Same root cause as Test 5. renderWebLinksDisclosure (PluginsSection.tsx:205-255) applies className='settings-panel__toggle-input' to the 3 confirmation checkboxes (lines 228, 237, 246) without the visible track/thumb counterparts, so each is hidden by the Phase-82 global rule at style.css:586-592. The modifier <select> at line 216 lacks that class and renders correctly — same differential evidence."
  artifacts:
    - path: "frontend/src/components/PluginsSection.tsx"
      issue: "renderWebLinksDisclosure (lines 205-255) puts className='settings-panel__toggle-input' on the 3 confirmation checkboxes (228, 237, 246) without the visible track/thumb counterparts"
  missing:
    - "Fix lands in the same patch as Test 5 — same render path. Choose option (a) drop the class (recommended), (b) render track/thumb spans, or (c) scoped CSS override."
  debug_session: ".planning/debug/99-disclosure-checkboxes-missing.md"
