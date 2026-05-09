---
phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate
plan: 02
status: complete
requirements: [PUI-03, PUI-04]
---

# 99-02 SUMMARY — PluginsSection inline disclosures (PUI-03, PUI-04)

## What was built

Three inline `<details>` disclosures rendered under the Search / Web Links / Inline Images rows in the existing `PluginsSection` component. Each disclosure exposes the plugin's runtime configuration (search defaults, link click behavior, image storage limit) without a per-row save button — sub-key RPCs dispatch immediately on change, honoring the PUI-04 anti-race contract first established in Phase 94-07 WR-03.

The `renderRow` helper signature gained an optional 5th `disclosure?: React.ReactNode` argument; the three rows that need an advanced section (`search`, `webLinks`, `image`) pass their respective render-helper output. The remaining five rows (webgl, unicode11, serialize, clipboard, progress) pass nothing and behave exactly as before.

## Key files

- `frontend/src/components/PluginsSection.tsx` (modified) — adds `imageStorageDebounceRef` + cleanup `useEffect`; extends `renderRow` with optional `disclosure` arg; introduces `renderSearchDisclosure`, `renderWebLinksDisclosure`, `renderImageDisclosure` helpers; wires them into the matching `renderRow` calls. Imports `SetSearchConfig`, `SetWebLinksConfig`, `SetImageConfig` from the Wails App bindings.
- `frontend/src/components/__tests__/PluginsSection.disclosure.test.tsx` (new) — 9 source-inspection vitest assertions covering the 3 disclosure markers, verbatim summary copy, sub-key RPC dispatch literals, modifier `<select>` options, [1,1000] storageLimit clamp, the SetPluginSettings count contract, and the 500ms debounce wrapper.

## Disclosure shapes

- **Search defaults** — 3 checkboxes (Regex / Case sensitive / Whole word) bound to `pluginConfig.searchConfig`. Each `onChange` constructs a `new daemon.SearchConfig({ ...sc, [field]: value })` and dispatches `SetSearchConfig(next)` with a silent catch.
- **Link click behavior** — `<select>` for the modifier (`platform` / `cmd` / `ctrl` / `none`) plus 3 checkboxes (`confirmOSC8` / `confirmIDN` / `confirmTyposquat`). Each control dispatches `SetWebLinksConfig(new daemon.WebLinksConfig(...))` immediately on change.
- **Storage limit** — single `<input type="number" min={1} max={1000} step={1}>` with " MB" suffix, bound to `pluginConfig.imageConfig.storageLimit`. Local state updates immediately for input responsiveness; the daemon RPC `SetImageConfig(new daemon.ImageConfig({ storageLimit: v }))` is wrapped in a 500ms debounce via `imageStorageDebounceRef`. The cleanup `useEffect` clears the timer on unmount.

## Why sub-key RPCs (not SetPluginSettings)

Phase 94-07 WR-03 documented the race between full-snapshot saves (Save Plugins button) and per-field changes: a user could toggle a sub-field in a disclosure, then click Save Plugins, and either (a) lose the disclosure change if the snapshot was taken before it, or (b) overwrite a concurrent daemon-side mutation. The anti-race contract is that disclosure changes go through dedicated sub-key RPCs (`SetSearchConfig`, `SetWebLinksConfig`, `SetImageConfig`) which the daemon serializes through a single mutex (engine.go:497-563), and the Save Plugins button only reflects boolean enable/disable changes.

Source-inspection test 8 enforces the contract by counting `SetPluginSettings` occurrences — exactly 2 (import + handleSavePlugins call). Adding a third would fail the test.

## 500ms debounce rationale

A naive number input (no debounce) would dispatch `SetImageConfig` on every keystroke. Typing "150" produces three RPCs: `1`, `15`, `150`. Each crosses the Wails IPC boundary, holds the daemon mutex briefly, and persists to settings.json. With clamping to `[1, 1000]`, the intermediate values are also valid. Threat T-99-05 from the plan's threat model identifies this as a DoS surface; the 500ms debounce is the mitigation. Local state updates immediately so the UI is still responsive.

## Source-inspection test pattern

The test file uses `import raw from '../PluginsSection.tsx?raw'` followed by `expect(raw).toContain(...)` and regex assertions on the file string. This avoids the createRoot render path because Wails-generated `daemon.*Config` constructors call `convertValues` which expects Go-side context and throws under jsdom. The precedent is `PluginsSection.test.tsx` (PUI-01 source-inspection tests).

## Commits

- `2e4e0ff` feat(99-02): inline `<details>` disclosures for Search/WebLinks/Image (PUI-03/PUI-04)
- `b6c8760` test(99-02): source-inspection asserts for PUI-03 disclosures + PUI-04 anti-race

## Verification

- `pnpm tsc --noEmit` exits 0.
- `pnpm test -- --run PluginsSection` → 33/33 pass (24 PluginsSection + 9 disclosure).
- Acceptance criteria greps satisfied: 3× `settings-panel__details`, 1× each verbatim summary, 3× `new daemon.SearchConfig`, 4× `new daemon.WebLinksConfig`, 2× `new daemon.ImageConfig`, all 4 modifier select options, `min={1}` / `max={1000}` present, `imageStorageDebounceRef` declared + cleared + set, `SetPluginSettings` count = 2.
- PUI-04 anti-race contract: enforced by source-inspection test (`expect(matches.length).toBe(2)`).

## Self-Check: PASSED

- All 9 must_haves[truths] from PLAN are satisfied (verified by tests + source inspection).
- Both artifacts exist with min_lines met (test file is 53 lines).
- No regression in pre-existing `PluginsSection.test.tsx` (24 tests still green).

## Notes

The plan's Test 9 regex `setTimeout\([^)]+,\s*500\)` was too strict — it forbade any `)` between `setTimeout(` and `, 500)`, but our debounce body contains nested parens (the `.catch(() => {})` chain). Updated to `setTimeout\([\s\S]+?,\s*500\)` which still asserts the 500ms parameter without forbidding nested calls. Documented in the test file inline.

The plan also authored two JSDoc strings that referenced `SetPluginSettings` in prose, which would have inflated the regex count and broken Test 8 (`expect(matches.length).toBe(2)`). Trimmed both: the file's existing JSDoc now says "Wails Get/Set plugin-settings bindings"; the new disclosure JSDoc says "never the full-snapshot save RPC" instead of "NOT SetPluginSettings". Both rewrites preserve the documentation intent without re-introducing the bare token.

This plan was executed inline by the orchestrator (Opus) rather than spawned as a Sonnet subagent — because Wave 1's spawned agents had hit the daily Sonnet rate limit and there was no need to wait for the reset to make progress on a small, prescriptive plan.
