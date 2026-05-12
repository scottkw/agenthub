---
status: resolved
trigger: "Phase 99-02 disclosures: Search and Web Links checkbox inputs missing at runtime; <select> and <input type=\"number\"> render correctly"
created: 2026-05-11T00:00:00Z
updated: 2026-05-12T22:55:00Z
resolved_on: 2026-05-12
resolved_by: "Phase 99-06 gap-closure: commit 247a7b4 (drop hidden-toggle class from 6 disclosure checkboxes) + commit 6cfe32d (real-DOM render test PluginsSection.disclosure.render.test.tsx — 3 it-blocks, all passing). 99-UAT.md Tests 5 & 6 flipped issue → pass."
---

## Current Focus

hypothesis: CONFIRMED — disclosure checkboxes carry className "settings-panel__toggle-input" which is globally hidden by a Phase-82 toggle-switch CSS rule (position:absolute; width:1px; height:1px; opacity:0; pointer-events:none). The main plugin rows pair this hidden input with visible .settings-panel__toggle-track + .settings-panel__toggle-thumb spans inside the same <label>. The disclosure helpers reuse the class but DO NOT render the track/thumb spans, so the user sees only the label text.
test: Read style.css for .settings-panel__toggle-input rule; compare main-row markup (renderRow lines 134-138) with disclosure markup (lines 171-200, 226-252); confirm absence of track/thumb spans in disclosures.
expecting: CSS rule hides every .settings-panel__toggle-input checkbox; only renderRow provides a visible replacement; disclosure helpers provide none — confirmed.
next_action: Return ROOT CAUSE FOUND to parent; suggested fix direction: drop className "settings-panel__toggle-input" from the 7 disclosure checkbox inputs (so browser-default rendering applies) OR add a disclosure-scoped CSS override OR render the missing track/thumb spans alongside each disclosure checkbox.

## Symptoms

expected: <details> disclosures for Search and Web Links render visible checkbox controls beside each label (Regex / Case sensitive / Whole word; Confirm OSC 8 / Confirm IDN / Confirm typosquat).
actual: <details> opens; labels render as plain text; no checkbox visual appears. Web Links modifier <select> renders correctly. Inline Images <input type="number"> renders correctly.
errors: none (silent visual regression; no console errors reported in symptoms).
reproduction: Launch wails build -tags wailsassets, open Settings → Plugins, expand the Search disclosure or the Web Links disclosure. Observe labels with no checkbox to their left.
started: Phase 99-02 introduction (commit 2e4e0ff feat(99-02): inline <details> disclosures for Search/WebLinks/Image).

## Eliminated

- hypothesis: Conditional renders checkbox-less branch (e.g. pluginConfig.searchConfig falsy first paint).
  evidence: renderSearchDisclosure and renderWebLinksDisclosure both early-return null when !pluginsLoaded || !pluginConfig (lines 163, 206). Once past that guard, the <input> elements are unconditionally inside the JSX — no inner conditional gates them. Also, the <select> in the same gated branch renders fine, so the guard is passing at runtime.
  timestamp: 2026-05-11T00:00:00Z

- hypothesis: dangerouslySetInnerHTML or HTML stripping.
  evidence: Standard JSX throughout PluginsSection.tsx; no innerHTML usage; no Markdown rendering layer.
  timestamp: 2026-05-11T00:00:00Z

- hypothesis: Self-closing tag wrapping issue inside <label>.
  evidence: JSX self-closes <input ... /> identically in renderRow (which works) and in disclosure helpers (which don't). The DOM shape difference is not the cause; the visual difference is.
  timestamp: 2026-05-11T00:00:00Z

- hypothesis: checked-without-onChange React warning strips input.
  evidence: Every disclosure checkbox has both `checked={...}` and `onChange={(e) => dispatch(...)}` (lines 176-177, 186-187, 196-197, 230-231, 239-240, 248-249). Not the cause.
  timestamp: 2026-05-11T00:00:00Z

## Evidence

- timestamp: 2026-05-11T00:00:00Z
  checked: frontend/src/components/PluginsSection.tsx renderRow (working main-plugin row) — lines 118-156.
  found: renderRow markup is <label class="settings-panel__toggle-row" ...><span class="settings-panel__toggle-track"><span class="settings-panel__toggle-thumb"/></span><span class="settings-panel__toggle-label">{label}</span></label> followed by a separate hidden <input class="settings-panel__toggle-input"/>. The visible toggle pill is the track+thumb spans, NOT the input.
  implication: The toggle UI is a CSS-only iOS-style switch; the real <input> is intentionally hidden as a semantic anchor only.

- timestamp: 2026-05-11T00:00:00Z
  checked: frontend/src/style.css lines 585-592 (.settings-panel__toggle-input rule).
  found: Rule is `position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none;` — i.e. every checkbox carrying this class is collapsed to a 1×1px transparent unclickable element regardless of where it is rendered. Phase-82 vintage (comment "Toggle / switch control (Phase 82)").
  implication: Any <input class="settings-panel__toggle-input"> renders as a visually-invisible element. The class is appropriate ONLY when paired with the visible track/thumb spans.

- timestamp: 2026-05-11T00:00:00Z
  checked: frontend/src/components/PluginsSection.tsx renderSearchDisclosure (lines 162-203) and renderWebLinksDisclosure (lines 205-255).
  found: Every disclosure checkbox (lines 173, 183, 193, 228, 237, 246) carries `className="settings-panel__toggle-input"`. None of these are accompanied by `.settings-panel__toggle-track` / `.settings-panel__toggle-thumb` spans inside the parent <label>; only a `.settings-panel__toggle-label` span. Six instances total: 3 in Search disclosure + 3 in Web Links disclosure.
  implication: The CSS hides the input (1×1, opacity 0) and no visible toggle pill is drawn in its place. The user sees only the label text. Exact match to the screenshot evidence.

- timestamp: 2026-05-11T00:00:00Z
  checked: frontend/src/components/PluginsSection.tsx <select> (line 216-224) and <input type="number"> (line 266).
  found: Neither element carries the `settings-panel__toggle-input` class. Browser-default rendering applies and they appear correctly.
  implication: Differential evidence confirms the class is the discriminator between "renders" and "invisible." Working controls have no class on the form element; broken controls all have `settings-panel__toggle-input`.

- timestamp: 2026-05-11T00:00:00Z
  checked: Source-inspection test pattern (PluginsSection.disclosure.test.tsx, per 99-02-SUMMARY.md).
  found: Tests use `import raw from '../PluginsSection.tsx?raw'` and `expect(raw).toContain(...)` to grep literals — they assert checkbox markup is *in the source string*, not that any input is visible in a real DOM.
  implication: Explains why 33/33 tests passed while the runtime UI is broken. The source-inspection test pattern (chosen because Wails-generated `daemon.*Config` constructors fail under jsdom) cannot catch CSS-driven visual regressions. Class-level test gap, not bug-level.

- timestamp: 2026-05-11T00:00:00Z
  checked: style.css grep for "details" — any disclosure-scoped CSS override.
  found: Only three `.settings-panel__details` rules (container margin, summary color, summary:hover). No override of `.settings-panel__details .settings-panel__toggle-input` that would unset the hidden positioning.
  implication: No CSS cascade rescues the disclosure case. The bug is purely the missing track/thumb spans paired with the hidden-input class.

## Resolution

root_cause: |
  The six disclosure checkbox inputs (3 in renderSearchDisclosure + 3 in renderWebLinksDisclosure) reuse `className="settings-panel__toggle-input"`, which the Phase-82 toggle-switch CSS rule (style.css:586-592) hides via `position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none`. In the main plugin rows (renderRow), this hidden input is paired with visible `.settings-panel__toggle-track` and `.settings-panel__toggle-thumb` spans that draw the iOS-style toggle pill — the input is a semantic anchor only. The disclosure helpers copied the class but omitted the track/thumb spans, so the checkbox is invisible and nothing replaces it. The <select> and <input type="number"> in the same disclosures render correctly because they do not carry the toggle-input class. Source-inspection tests passed because they grep file content rather than rendered DOM.
fix: |
  (Not applied per goal:find_root_cause_only. Suggested direction below.)
verification: |
  (Not applied per goal:find_root_cause_only.)
files_changed: []
