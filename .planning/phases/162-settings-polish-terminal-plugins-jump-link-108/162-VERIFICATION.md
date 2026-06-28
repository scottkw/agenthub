---
phase: 162-settings-polish-terminal-plugins-jump-link-108
verified: 2026-06-28T13:10:00Z
status: human_needed
score: 5/6 must-haves verified
behavior_unverified: 1
overrides_applied: 0
re_verification: false
behavior_unverified_items:
  - truth: "Clicking the 'Terminal Plugins' jump link scrolls to the plugins section because its href anchor still resolves to id='settings-plugins'."
    test: "Open Settings in the live app (GUI or web-share surface), click the 'Terminal Plugins' jump link in the sticky bar, observe scroll destination."
    expected: "The viewport scrolls to (and stops at) the 'Terminal Plugins' section header, with the header visible below the sticky jump bar (not hidden behind it)."
    why_human: "Anchor/id integrity is statically verified (href='#settings-plugins' targets id='settings-plugins'). The actual scroll transition — that the browser navigates to the correct section and scroll-margin-top clears the sticky bar — is a runtime state change that grep/presence checks cannot observe."
human_verification:
  - test: "Click 'Terminal Plugins' jump link in running app (GUI and web-share surface)"
    expected: "Viewport scrolls to the 'Terminal Plugins' section header; header appears below the sticky jump bar, not behind it; the section is the last visible section (renders last)."
    why_human: "Live browser scroll behavior from hash navigation — statically verifiable anchor wiring is confirmed, but the runtime scroll outcome requires a live surface."
  - test: "Search for 'terminal plugins' in the Settings search box"
    expected: "A result labeled 'Terminal Plugins' appears, pointing to the settings-plugins anchor; clicking it scrolls to the section correctly."
    why_human: "SettingsSearch derives from SETTINGS_JUMP_LINKS (statically verified), but the search UI rendering and result navigation are runtime behaviors."
  - test: "Verify cross-surface parity — open Settings in web-share view and confirm the jump bar shows 'Terminal Plugins' last"
    expected: "Jump bar on the web-share surface shows the same 7 links in the same order, 'Terminal Plugins' last. Clicking it scrolls to the section on the web surface."
    why_human: "Shared React component ensures code-level parity, but the web-share rendering path (Wails webview vs browser) can differ at runtime."
---

# Phase 162: Settings Polish — Terminal Plugins Jump Link (#108) Verification Report

**Phase Goal:** Resolve GitHub issue #108. The Settings "Plugins" jump link currently sits first in the sticky jump bar while its section renders last — clicking the first link jumps to the bottom. The label "Plugins" misleads (these are terminal/shell plugins). Fix both: move the jump link to the LAST position (matching render order) and rename it to "Terminal Plugins" (jump-bar label + section header), keeping the `settings-plugins` anchor id stable so scroll-spy/jump behavior keeps working. Shared React frontend, so the fix applies to both GUI and web surfaces (cross-surface parity).

**Verified:** 2026-06-28T13:10:00Z
**Status:** human_needed — all static must-haves verified; live scroll/search behavior requires UAT
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | The Settings sticky jump bar shows "Terminal Plugins" as the LAST link, matching the on-screen render order (PluginsSection renders last). | VERIFIED | `SETTINGS_JUMP_LINKS[6]` = `{ label: 'Terminal Plugins', id: 'settings-plugins' }` — confirmed via node probe and direct read of `SettingsJumpBar.tsx:28`. Array has exactly 7 entries; plugins entry is at index 6 (last). |
| 2 | The plugins section header (h3) reads "Terminal Plugins". | VERIFIED | `PluginsSection.tsx:289`: `<h3 id="settings-plugins">Terminal Plugins</h3>` — exact byte match confirmed via grep. |
| 3 | Clicking the "Terminal Plugins" jump link scrolls to the plugins section because its href anchor still resolves to id="settings-plugins". | PRESENT_BEHAVIOR_UNVERIFIED | Static anchor integrity is verified: `SettingsJumpBar` renders `href="#settings-plugins"` from `link.id`, and `PluginsSection` carries `id="settings-plugins"` on the h3. CSS `scroll-margin-top` (tested) clears the sticky bar. The live scroll transition is a runtime browser behavior — see Human Verification. |
| 4 | The `settings-plugins` anchor id is unchanged (byte-for-byte) on both the jump-link entry and the section h3. | VERIFIED | Jump link: `id: 'settings-plugins'` at `SettingsJumpBar.tsx:28`. Section h3: `id="settings-plugins"` at `PluginsSection.tsx:289`. Test fixture: `id: 'settings-plugins'` at `SettingsTab.hyperlinked-index.test.tsx:15`. All three match exactly. |
| 5 | SettingsSearch auto-reflects the new "Terminal Plugins" label because SEARCH_INDEX spreads SETTINGS_JUMP_LINKS — no separate edit needed; this gives cross-surface parity (single shared React component). | VERIFIED | `SettingsSearch.tsx:2` imports `SETTINGS_JUMP_LINKS`; `SettingsSearch.tsx:26` spreads it: `...SETTINGS_JUMP_LINKS.map((l) => ({ label: l.label, target: l.id }))`. No hardcoded "Plugins" or "Terminal Plugins" label exists in `SettingsSearch.tsx`. Label propagation is automatic. |
| 6 | The frontend gate passes — tsc typecheck, vite build, and the full vitest suite — with the hyperlinked-index spec updated to expect "Terminal Plugins". | VERIFIED | `pnpm build` (tsc + vite): exit 0, build complete in 572ms. `pnpm test` (vitest): 2170 passed, 132 files, 0 failures. `bash tests/check-traceability-paths.sh`: exit 0, "OK: all traceability paths exist". |

**Score:** 5/6 truths verified (1 present, behavior-unverified — live scroll)

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/SettingsJumpBar.tsx` | SETTINGS_JUMP_LINKS reordered, "Terminal Plugins" at index 6, comment updated | VERIFIED | Confirmed: 7-entry array, last entry `{ label: 'Terminal Plugins', id: 'settings-plugins' }`. No bare `label: 'Plugins'` remains. Block comment updated to describe render-order-matching (no stale "listed first" prose found). Commit 20de5475. |
| `frontend/src/components/PluginsSection.tsx` | `<h3 id="settings-plugins">Terminal Plugins</h3>` | VERIFIED | `PluginsSection.tsx:289` contains exact string. `id` preserved. "Save Plugins" button copy unchanged at line 334. Commit 39b3e9e3. |
| `frontend/src/components/__tests__/SettingsTab.hyperlinked-index.test.tsx` | Fixture `label: 'Terminal Plugins'`; full vitest suite green | VERIFIED | Line 15: `{ label: 'Terminal Plugins', id: 'settings-plugins', file: 'PluginsSection.tsx', raw: pluginsRaw }`. `expectedTargets` includes `'#settings-plugins'` (unchanged). Suite passes: 2170/2170. Commit e5310d0b. |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `SettingsJumpBar` rendered `<a href="#settings-plugins">` | `PluginsSection` `<h3 id="settings-plugins">` | Browser hash navigation; CSS `scroll-margin-top` on `.settings-panel__body h3` | VERIFIED (static) | Href comes from `link.id` = `'settings-plugins'`; h3 id = `"settings-plugins"` — match is byte-for-byte. CSS rule verified by test (hyperlinked-index spec checks `scroll-margin-top: Npx` on `.settings-panel__body h3` and `position: sticky` on `.settings-jump-bar`). |
| `SettingsSearch.SEARCH_INDEX` | `SETTINGS_JUMP_LINKS` | Spread: `...SETTINGS_JUMP_LINKS.map(...)` | VERIFIED | `SettingsSearch.tsx:26`. No separate hard-edit. Label change propagates automatically. |

---

### Data-Flow Trace (Level 4)

Not applicable. This phase modifies static string constants and array order in a client-side component. There is no dynamic data source — `SETTINGS_JUMP_LINKS` is a compile-time constant, not fetched from an API or database. No Level 4 trace needed.

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| SETTINGS_JUMP_LINKS: 7 entries, last is Terminal Plugins, no bare Plugins label | `node -e "..."` (Task 1 verify script) | "OK order+label" — IDs in order ending with `settings-plugins`; labels ending with `Terminal Plugins`; no bare `'Plugins'` | PASS |
| PluginsSection h3 contains Terminal Plugins with id intact | `node -e "...regex test..."` | "OK h3" | PASS |
| Frontend tsc + vite build | `cd frontend && pnpm build` | exit 0; "built in 572ms" | PASS |
| Full vitest suite | `cd frontend && pnpm test` | 2170 passed, 132 files, 0 failures | PASS |
| Traceability check | `bash tests/check-traceability-paths.sh` | exit 0; "OK: all traceability paths exist" | PASS |

---

### Probe Execution

No probes declared in PLAN.md or SUMMARY.md. No `scripts/*/tests/probe-*.sh` files found for this phase. Section skipped.

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SETTINGS-UI-01 | 162-01-PLAN.md | Plugins jump link moved to last + renamed "Terminal Plugins" on label and section header; `settings-plugins` anchor unchanged; jump/scroll-spy still works; affected tests updated — closes #108 | SATISFIED | All 5 static must-haves VERIFIED. SETTINGS-UI-01 is defined inline in ROADMAP.md Phase 162 entry. Note: this requirement ID does not appear in `.planning/REQUIREMENTS.md` (which is scoped to v4.1 Session Chat features); it was added to ROADMAP.md directly as an orthogonal bug fix. The omission from REQUIREMENTS.md is an informational gap — the requirement is fulfilled. |

**Orphaned requirements:** None. REQUIREMENTS.md (v4.1 Session Chat scope) maps no additional IDs to Phase 162. SETTINGS-UI-01 is defined solely in ROADMAP.md for this phase.

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | — | — | No TBD/FIXME/XXX markers, no stubs, no empty returns, no hardcoded-empty data in the three modified files. The `return null` guards at PluginsSection.tsx:163,203,252 are pre-existing render-helper guards (conditional on plugin data loaded), not stubs. |

---

### Human Verification Required

#### 1. Jump Link Click-to-Scroll (GUI surface)

**Test:** Open AgentHub Settings tab in the running GUI app. Confirm the sticky jump bar shows "Terminal Plugins" as the last (7th) link. Click it.
**Expected:** The viewport scrolls to the "Terminal Plugins" section header. The header is fully visible below the jump bar (not hidden behind it). The section renders last in the Settings body.
**Why human:** Static anchor/id integrity is confirmed. The live scroll transition — browser hash navigation, CSS smooth scroll, scroll-margin-top clearing the sticky bar — requires a running app with a real scrollable viewport.

#### 2. Settings Search for "terminal plugins"

**Test:** Open Settings, type "terminal" in the search box, observe results.
**Expected:** A result labeled "Terminal Plugins" appears. Clicking it scrolls to the section correctly (same scroll behavior as Truth 3 above).
**Why human:** SEARCH_INDEX spread from SETTINGS_JUMP_LINKS is statically verified. The rendered search UI and result navigation are runtime behaviors.

#### 3. Cross-Surface Parity — Web-Share Surface

**Test:** Open Settings in the web-share surface (or wails dev browser). Confirm the jump bar shows "Terminal Plugins" as the last link. Click it.
**Expected:** Same behavior as the GUI surface — "Terminal Plugins" last, scrolls to the correct section, search shows "Terminal Plugins" as a result.
**Why human:** Shared React component ensures code-level parity. The web-share rendering path uses the same component but runs in a different browser context (Wails webview vs. plain Chrome); runtime parity must be confirmed.

---

### Gaps Summary

No gaps. All static must-haves are VERIFIED:
- SETTINGS_JUMP_LINKS reordered with "Terminal Plugins" last (index 6 of 7)
- PluginsSection h3 renamed to "Terminal Plugins" with `id="settings-plugins"` preserved
- Anchor `href="#settings-plugins"` in jump bar targets `id="settings-plugins"` on the h3
- SettingsSearch propagates the label automatically via SETTINGS_JUMP_LINKS spread
- Frontend gate fully green: tsc + vite build, 2170 vitest tests, traceability check

One truth is `PRESENT_BEHAVIOR_UNVERIFIED` (live scroll to the section on click) — wiring is correct, behavior requires live UAT. Awaiting human verification in 162-UAT.md.

---

_Verified: 2026-06-28T13:10:00Z_
_Verifier: Claude (gsd-verifier)_
