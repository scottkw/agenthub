---
phase: 94-search-addon-find-bar-desktop-web
reviewed: 2026-05-05T00:00:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - frontend/src/components/FindBar/FindBar.tsx
  - frontend/src/components/FindBar/style.css
  - frontend/src/components/TerminalPanel.tsx
  - frontend/src/components/PluginsSection.tsx
  - frontend/src/lib/isXtermFocused.ts
  - frontend/src/style.css
  - frontend/src/wailsjs/go/models.ts
  - internal/daemon/plugin_settings.go
  - internal/daemon/engine.go
  - web/embed.go
  - web/terminal.html
  - web/assets/terminal.js
  - web/assets/terminal.css
  - internal/webserver/testdata/findbar_perf_fixture.txt
findings:
  critical: 0
  warning: 4
  info: 6
  total: 10
status: needs-attention
---

# Phase 94: Code Review Report

**Reviewed:** 2026-05-05
**Depth:** standard
**Files Reviewed:** 14
**Status:** needs-attention

## Summary

Adversarial review of the Phase 94 Search Addon + Find Bar implementation across desktop (React) and web (plain DOM) surfaces. The implementation is generally high quality — the focus-conditioning gate (T-94-03), debounce + cancel-on-close (T-94-04), defaults-merge load path (T-94-02), and same-origin vendoring (T-94-01/05) are all correctly implemented and well-tested.

**No BLOCKER-class defects were found.** Specifically:
- No XSS surfaces (no `innerHTML` of user input; only static SVG / textContent for query echo)
- No SQL/command injection vectors
- No hardcoded secrets
- No authentication bypasses (search is purely client-side; daemon receives only SearchConfig booleans via the existing capability-gated SetPluginSettings)
- The decorations:`{}` reconciliation is correctly applied at every findNext / findPrevious site on both surfaces
- Cancel-on-close correctly clears both the debounce timer and the addon decorations on both surfaces
- The mount-cleanup pyramid in `TerminalPanel.tsx` correctly disposes the SearchAddon, the onDidChangeResults subscription, and the debounce timer on unmount

The 4 WARNINGs below identify maintainability and correctness gaps that should be fixed but are not data-loss / security risks:
1. UI-SPEC §"Animation" 200ms slide-in/slide-out is **not implemented** — the `.find-bar--entering` / `.find-bar--exiting` CSS classes exist but no code applies them. Bar appears instantly.
2. Desktop `searchOptions` lazy-init from `pluginConfig?.searchConfig` runs once at first render when `pluginConfig` is typically `null` (loaded asynchronously). Persisted user toggle defaults are **not honored on first session open** — bar opens with all-toggles-OFF until manually flipped.
3. PluginsSection ↔ FindBar persistence race: T-94-02 is documented as accepted, but the find bar's own `handleSearchOptionsChange` writes the entire pluginConfig back via `SetPluginSettings`. If the user has unsaved boolean-toggle edits in PluginsSection at the same time, those edits **will be silently overwritten** by the find bar's persistence call (find bar is reading the App-level prop, not the PluginsSection local edit buffer).
4. `decorations: {} as never` cast bypasses TypeScript safety — if the upstream `@xterm/addon-search` ISearchOptions type changes, source-inspection tests will still pass while real runtime breaks.

## Warnings

### WR-01: UI-SPEC slide-in/slide-out animation is not implemented

**Files:**
- `frontend/src/components/FindBar/FindBar.tsx:48-238`
- `frontend/src/style.css:2176-2184` (CSS classes defined)
- `web/assets/terminal.js:440-472` (showFindBar / hideFindBar)
- `web/assets/terminal.css:115` (transition rule defined but state never changes)

**Issue:**
UI-SPEC §"Animation" mandates a 200ms slide-in (`translateY(-100%)` → `translateY(0)`) on open and an asymmetric 150ms exit (`translateY(0)` → `translateY(-8px)` + opacity fade) on close. The CSS defines `.find-bar--entering` and `.find-bar--exiting` modifier classes, but **neither React FindBar.tsx nor web/assets/terminal.js ever applies these classes**. On both surfaces the bar appears instantly at `translateY(0); opacity: 1`. Dimension 2 (Visuals) of the UI-SPEC checker sign-off claims "find bar layout, dimensions, border, shadow, animation timing, and all icon button states fully specified" — implementation does not honor "animation timing." The 94-VALIDATION.md `FindBar.visual` test is a snapshot test and only asserts the transition CSS rule exists, not that classes are actually applied to the element during open/close. UAT instruction in 94-VALIDATION.md "200 ms slide-in / slide-out feels right" is a manual check that has not been run.

**Fix:**
On desktop (FindBar.tsx) — apply `.find-bar--entering` on first render via a `useState(true)` mount flag, then drop it in a `requestAnimationFrame`/`setTimeout(0)` so the browser sees the class flip and runs the transition. For exit, add an exit-animation state in TerminalPanel that delays the unmount by 150-200ms while applying `.find-bar--exiting`.

On web (terminal.js) — in `showFindBar`, set `el.classList.add('find-bar--entering')` BEFORE removing `hidden`, then RAF the removal of the entering class. In `hideFindBar`, add `find-bar--exiting`, wait 200ms via setTimeout, then `el.hidden = true` and remove the class.

```tsx
// FindBar.tsx — sketch
const [entering, setEntering] = useState(true)
useEffect(() => {
  // Drop the modifier on the next frame so the transition fires.
  const id = requestAnimationFrame(() => setEntering(false))
  return () => cancelAnimationFrame(id)
}, [])
const className = `find-bar${entering ? ' find-bar--entering' : ''}`
```

Alternatively, document this as an accepted scope reduction in 94-VERIFICATION.md (currently absent) and remove the unused `.find-bar--entering` / `.find-bar--exiting` CSS rules so the next maintainer doesn't assume they're wired.

---

### WR-02: searchOptions never sync from pluginConfig prop after first render

**File:** `frontend/src/components/TerminalPanel.tsx:99-103`

**Issue:**
The local `searchOptions` state is initialized via lazy initializer:

```tsx
const [searchOptions, setSearchOptions] = useState<FindBarSearchOptions>(() => ({
  regex: pluginConfig?.searchConfig?.regex ?? false,
  caseSensitive: pluginConfig?.searchConfig?.caseSensitive ?? false,
  wholeWord: pluginConfig?.searchConfig?.wholeWord ?? false,
}))
```

`useState(() => ...)` runs **only on the first render**. In real usage, `pluginConfig` is loaded asynchronously by `App.tsx` via `GetPluginSettings()` and is `null` on the first render of any TerminalPanel mounted before the load resolves. When pluginConfig later arrives with persisted user choices (say `caseSensitive: true`), `searchOptions` is **never re-seeded** — the bar opens with all-OFF defaults despite user's saved preferences.

The 94-03 plan rationale ("Pitfall #2 — never re-sync mid-open" — preserve toggle state during an open session against an SSE-pushed change from another window) is a legitimate concern AFTER the bar has been opened, but it incorrectly precludes the legitimate first-load seeding case. SRC-02 requires "Defaults are persisted via `SearchConfig` in daemon settings and **loaded at find-bar mount**" — UI-SPEC line 27.

**Reproduction:** Toggle case-sensitive ON, restart the desktop app, press Cmd-F. Toggle is OFF instead of ON.

**Fix:**
Track a "has the searchOptions ever been seeded from a non-null pluginConfig" flag, or seed lazily inside the Cmd-F open handler instead of useState's initializer:

```tsx
const seededRef = useRef(false)
useEffect(() => {
  if (seededRef.current) return
  if (!pluginConfig?.searchConfig) return
  setSearchOptions({
    regex: !!pluginConfig.searchConfig.regex,
    caseSensitive: !!pluginConfig.searchConfig.caseSensitive,
    wholeWord: !!pluginConfig.searchConfig.wholeWord,
  })
  seededRef.current = true
}, [pluginConfig?.searchConfig])
```

This honors the persisted defaults on first load AND respects "no mid-open re-sync" (only fires once before the bar is ever opened).

---

### WR-03: PluginsSection edits silently clobbered by FindBar persistence

**Files:**
- `frontend/src/components/TerminalPanel.tsx:414-426` (handleSearchOptionsChange)
- `frontend/src/components/PluginsSection.tsx:42-55` (handleSavePlugins) + `:64-72` (toggle)

**Issue:**
PluginsSection holds a private local `pluginConfig` state populated by `GetPluginSettings()` at mount and mutated by toggle clicks. The Save Plugins button writes the **entire** PluginsSection-local `pluginConfig` back via `SetPluginSettings(pluginConfig)`. Meanwhile, `TerminalPanel.handleSearchOptionsChange` constructs `new daemon.PluginSettings({ ...pluginConfig, searchConfig: opts })` from the **App-level** prop and calls `SetPluginSettings(next)` directly.

Race scenario:
1. User opens Settings → Plugins → flips WebGL OFF (PluginsSection local `pluginConfig.webgl = false`, not saved yet — Save button still says "Save Plugins").
2. User opens find bar in another tab and toggles case-sensitive ON → desktop `handleSearchOptionsChange` fires → reads App-level prop (still `webgl: true` because PluginsSection hasn't saved) → writes `{ webgl: true, ..., searchConfig: { caseSensitive: true } }` to disk.
3. App receives `settings:plugins` event with the new state → updates the App-level prop.
4. User goes back to Settings, clicks "Save Plugins" → PluginsSection writes its private state with `webgl: false` → searchConfig (from PluginsSection's stale snapshot taken at step 1, where it was all-false) **overwrites** the case-sensitive toggle the user just flipped on.

This is a real data-loss vector for the user's preferences, even if the surface area is narrow. The 94-03 SUMMARY notes this race as "T-94-02 (toggle race vs PluginsSection) — accept — UNCHANGED — pre-existing race; daemon settings:plugins event re-syncs," but PluginsSection deliberately does NOT consume the daemon settings:plugins event mid-edit (see lines 30-40, only initial `GetPluginSettings()`). The "re-sync" mitigation does not apply.

**Fix:**
Two acceptable paths:
1. **Make PluginsSection consume the settings:plugins event** to refresh its local state when the find bar persists changes. Risks resetting unsaved boolean-toggle edits — UX trade-off.
2. **Make `handleSearchOptionsChange` write only the searchConfig sub-key**, not the full PluginSettings, and add a daemon-side `SetSearchConfig(SearchConfig)` RPC. Most surgical; preserves PluginsSection's edit buffer semantics.

Document the chosen mitigation in 94-VERIFICATION.md or escalate the threat-model entry from "accept" to "mitigate."

---

### WR-04: `decorations: {} as never` cast bypasses TypeScript safety net

**Files:**
- `frontend/src/components/TerminalPanel.tsx:410, 417, 430, 435`
- `web/assets/terminal.js:373-380` (no type system, but same fragility)

**Issue:**
Four findNext/findPrevious sites in TerminalPanel.tsx pass `{ ...searchOptions, decorations: {} as never }`. The `as never` cast is required because `@xterm/addon-search` ISearchOptions declares `matchOverviewRuler` and `activeMatchColorOverviewRuler` as required `string` fields when `decorations` is present. The cast forces the compiler to accept an empty object.

This deliberately defeats the type system for an empirically-discovered runtime contract (the `_fireResults(e){this._resultTracker.fireResultsChanged(!!e?.decorations)}` gate in addon-search 0.16). Risks:

1. **Upstream API change:** if a future addon-search version validates the `decorations` field at runtime (throws on missing required keys), the cast hides the breakage from `tsc` and `vitest`'s jsdom can't construct a real Terminal/SearchAddon to catch it. The chromedp e2e is the only catch — but chromedp is build-tagged `e2e` and not part of the default lane.
2. **`as never` is the strongest type-system override** — strictly worse than `as any` because it conveys "this is impossible" while it is in fact "this is empirically required." A future maintainer reading `as never` may assume the call is dead code and remove it.
3. The 94-05 summary documents the discovery thoroughly, but the production code only has a one-line comment pointing at FindBar.themeMatrix.test.tsx.

**Fix:**
Define a typed wrapper at module scope so the cast is named and documented in one place:

```tsx
// addon-search 0.16's onDidChangeResults event fires only when opts.decorations
// is truthy (`_fireResults` gate). Empty object is truthy + has no per-theme
// color keys, preserving the 138-theme invariant via theme.selectionBackground.
// Discovered empirically via Plan 94-05 chromedp e2e (94-05-SUMMARY.md §"Deviation 1").
type FindNextOpts = ISearchOptions
function findNextOpts(o: FindBarSearchOptions): FindNextOpts {
  return { ...o, decorations: {} as ISearchOptions['decorations'] }
}
```

This centralizes the cast and makes the lifetime of the workaround visible to grep / future code review.

## Info

### IN-01: `navigator.platform` is deprecated

**Files:**
- `frontend/src/components/TerminalPanel.tsx:374` — `navigator.platform.toUpperCase().includes('MAC')`
- `frontend/src/components/FindBar/FindBar.tsx:79` — `e.metaKey || e.ctrlKey` (OK — uses both)
- `web/assets/terminal.js:492, 547` — `navigator.platform.toUpperCase().indexOf('MAC') >= 0`

**Issue:**
`Navigator.platform` is deprecated per MDN; modern browsers may return reduced/spoofed strings. The current strategy works in practice but is on borrowed time.

**Fix:** Prefer feature-detection via `e.metaKey || e.ctrlKey` (FindBar.tsx already does this for Cmd-G / Ctrl-G — accepts either modifier with no platform branching). Apply the same pattern to TerminalPanel's window-level Cmd-F handler and to terminal.js's keydown handlers — accept either metaKey or ctrlKey for the F key as well, dropping the platform check entirely. Browser-native Cmd-F still works on the unfocused fallthrough.

### IN-02: Web SearchAddon errors silently swallowed

**File:** `web/assets/terminal.js:436, 458, 497-503, 508-515, 533-535`

**Issue:**
Six try/catch blocks around `searchAddonHandle.findNext / findPrevious / clearDecorations` swallow exceptions silently (`catch (e) {}`). This is defensive but masks real bugs. A future change that breaks a findNext call (say, by passing a stale Terminal reference) will appear as "search just doesn't work" with no diagnostic in the browser console.

**Fix:** Log to console at warn level: `} catch (e) { console.warn('[FindBar]', e); }`. Preserves the silent-degradation behavior at runtime but surfaces the cause for debugging.

### IN-03: Multiple `navigator.platform` calls per keystroke

**File:** `web/assets/terminal.js:492, 547`

**Issue:**
Each keydown event recomputes `navigator.platform.toUpperCase().indexOf('MAC') >= 0`. For high-frequency typing into the find-bar input, this is invoked on every keystroke. Trivial perf cost, but easily hoisted.

**Fix:** Compute `var isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;` once at the top of the IIFE.

### IN-04: PluginsSection comment misleading about Phase 92 contract

**File:** `frontend/src/components/PluginsSection.tsx:18`

**Issue:**
The component header states "Phase 92 contract: TerminalPanel does NOT consume pluginConfig — the pipeline is wired but inert. Phase 93 wires consumption." This is stale — Phase 93 wired consumption and Phase 94 added SearchConfig consumption. The comment may confuse future maintainers about whether pluginConfig is live.

**Fix:** Update the comment to: "Phase 92/93/94 contract: TerminalPanel consumes pluginConfig live for webgl/clipboard/search hot-swap; unicode11 is next-session-only."

### IN-05: `findBarFocusSeq` accumulates indefinitely

**File:** `frontend/src/components/TerminalPanel.tsx:97, 380`

**Issue:**
`setFindBarFocusSeq((s) => s + 1)` is called on every Cmd-F press while the terminal is focused. Over a long session, this number can grow large. Not a correctness bug — JS Number safe-integer range is 2^53 — but a code smell.

**Fix:** Reset to 0 in `handleSearchClose`, or use a boolean toggle pattern instead.

### IN-06: Web `var` declarations in modern code

**File:** `web/assets/terminal.js` (throughout — `var` is used throughout instead of `let`/`const`)

**Issue:**
The file uses ES5 `var` declarations exclusively. Modern browsers (the only target for the served terminal page) fully support `let`/`const`. `var` hoisting can lead to subtle scoping bugs (e.g. closures over loop variables). The IIFE `for (var i = 0; ...) { (function(id, key) { ... })(toggleSpecs[i][0], toggleSpecs[i][1]); }` IIFE-wrap on lines 526-538 is a workaround that wouldn't be needed with `let i`.

**Fix:** Migrate the Phase 94-added find-bar block to `let`/`const` and replace the IIFE-wrap with `for (const [id, key] of toggleSpecs) { ... }`. Out of scope for an isolated Phase 94 fix; flag for a future cleanup pass.

---

_Reviewed: 2026-05-05_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
