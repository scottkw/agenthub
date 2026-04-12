# Phase 69: Settings Scrollable Layout - Research

**Researched:** 2026-04-11
**Domain:** React frontend component refactor (SettingsTab.tsx, style.css)
**Confidence:** HIGH

## Summary

Phase 69 replaces the current tabbed sub-navigation inside the Settings panel with a single continuous scrollable page. The current `SettingsTab` component renders three sub-tabs (CLI Paths, Web Server, Appearance) via a `settings-panel__tabs` button bar plus conditional `{activeTab === 'x' && ...}` content gating. The refactor removes the tab switcher UI and renders all three groups sequentially with visible section headers as separators, in one `overflow-y: auto` scroll container.

This is a pure frontend refactor. No backend calls change. No Go code changes. All existing state, event handlers, and Wails bindings remain identical — only the JSX structure and CSS change. The existing `settingsActiveTab` state in `App.tsx` (lifted to persist across unmounts) becomes obsolete and can be removed as part of cleanup.

The test suite uses source-inspection (`?raw` imports and `readFileSync`) rather than DOM rendering, which makes the existing tests brittle against the exact patterns being changed. New tests need to assert the scrollable layout contract (no tab buttons, section headers present, all content groups present simultaneously).

**Primary recommendation:** Remove `settings-panel__tabs` + tab buttons; render all three content groups unconditionally under named section headers in a single `settings-panel__body` container; add a new `settings-panel__section-header` CSS class following the existing `welcome-tab__heading` visual pattern.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SETT-01 | Settings tab replaced with single scrollable page containing all settings groups | Remove conditional `activeTab` gating; render Appearance + Web Server + Paths groups sequentially in one scrollable div |
| SETT-02 | Each settings group has a visible section header (Appearance, Web Server, Paths, etc.) | Add `<h2>` or `<div>` section headers with new CSS class; `settings-panel__body h3` stub already exists in CSS |
| SETT-03 | Existing settings functionality preserved (theme picker, web actions, save paths) | All handlers, state variables, and Wails calls are unchanged; only conditional rendering removed |
</phase_requirements>

## Current Architecture (Verified)

### SettingsTab.tsx — Key Structural Facts [VERIFIED: codebase read]

**Props that drive the tab UI (become obsolete):**
- `activeTab: 'cli-paths' | 'web-server' | 'appearance'` — drives which tab is shown
- `onActiveTabChange: (tab: ...) => void` — lifted setter from App.tsx

**State in App.tsx that becomes obsolete:**
```typescript
// App.tsx line 96 — can be removed after refactor
const [settingsActiveTab, setSettingsActiveTab] = useState<'cli-paths' | 'web-server' | 'appearance'>('cli-paths')
```

**Content currently gated by activeTab (must all become unconditional):**
1. `activeTab === 'cli-paths'` — CLI path table + Save Paths button
2. `activeTab === 'web-server'` — Tailscale status, CT disclosure, port, start/stop button, URL row, LAN credentials
3. `activeTab === 'appearance'` — Terminal theme selector

**Tab switcher to remove:**
```tsx
// SettingsTab.tsx lines 214-239 — remove entire block
<div className="settings-panel__tabs" role="tablist">
  <button ...>CLI Paths</button>
  <button ...>Web Server</button>
  <button ...>Appearance</button>
</div>
```

### style.css — Relevant Classes [VERIFIED: codebase read]

**Classes to remove (tab nav):**
- `.settings-panel__tabs` — tab row container
- `.settings-panel__tab-btn` — individual tab button
- `.settings-panel__tab-btn:hover`
- `.settings-panel__tab-btn--active`

**Classes that stay (body + content):**
- `.settings-tab` — outer wrapper (`display: flex; flex-direction: column; flex: 1; overflow-y: auto`)
- `.settings-panel__body` — scrollable content (`flex: 1; overflow-y: auto; padding: 20px`)
- `.settings-panel__body h3` — already defined (`font-size: 13px; text-transform: uppercase; letter-spacing: 0.08em; color: #565f89; margin-bottom: 12px`) — **currently unused in JSX**
- All field group, button, table, input, and web-server classes — unchanged

**Section header CSS already exists:** `.settings-panel__body h3` is defined in the CSS (lines 375-381) but never used. This means the planner can use `<h3>` elements for section headers with zero CSS additions needed.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2.4 | Component rendering | Project standard [VERIFIED: package.json] |
| TypeScript | 5.9.3 | Type safety | Project standard [VERIFIED: package.json] |
| vitest | 4.1.0 | Testing | Project standard [VERIFIED: package.json] |

### No New Dependencies
This phase requires zero new npm packages. All needed tools (React, CSS, heroicons, Wails bindings) already exist. [VERIFIED: codebase read]

## Architecture Patterns

### Recommended Refactor Structure

**Before:** Tab-gated conditional rendering
```tsx
// Current pattern — remove this
{activeTab === 'appearance' && (
  <div className="settings-panel__field-group">
    ...theme selector...
  </div>
)}
```

**After:** Sequential unconditional sections with headers
```tsx
// New pattern — render all groups always
<div className="settings-panel__body">
  {/* Appearance section */}
  <h3>Appearance</h3>
  <div className="settings-panel__field-group">
    ...theme selector...
  </div>

  {/* Web Server section */}
  <h3>Web Server</h3>
  ...web server content...

  {/* Paths section */}
  <h3>Paths</h3>
  ...CLI paths table...
</div>
```

The `<h3>` elements use the pre-existing `.settings-panel__body h3` CSS rule — no new CSS class needed. [VERIFIED: style.css line 375]

### Section Order Recommendation

Suggested order (SETT-02 requires headers; order is Claude's discretion):
1. **Appearance** — theme picker (simple, visual — good first impression)
2. **Web Server** — Tailscale status, CT disclosure, start/stop (most commonly adjusted)
3. **Paths** — CLI path overrides (power user / rarely changed)

This is Claude's discretion — the planner should confirm or choose.

### Props Interface Change

Remove from `SettingsTabProps`:
- `activeTab: 'cli-paths' | 'web-server' | 'appearance'`
- `onActiveTabChange: (tab: 'cli-paths' | 'web-server' | 'appearance') => void`

Remove from `App.tsx`:
- `const [settingsActiveTab, setSettingsActiveTab] = ...` (line 96)
- `activeTab={settingsActiveTab}` prop (line 578)
- `onActiveTabChange={setSettingsActiveTab}` prop (line 579)

### Visual Design Pattern [VERIFIED: style.css, WelcomeTab.tsx]

The project uses this heading pattern in `WelcomeTab`:
- Font size: 11px
- Text transform: uppercase
- Letter spacing: 0.1em
- Color: `#565f89` (muted/secondary)
- Font weight: 600
- Margin bottom: 10px

The `.settings-panel__body h3` stub matches this pattern closely (`font-size: 13px`, `text-transform: uppercase`, `letter-spacing: 0.08em`, `color: #565f89`). Using `<h3>` tags activates this existing stub cleanly.

### Optional: Section Spacing Enhancement

If the planner wants visual separation between sections, add a bottom margin or top border to `.settings-panel__body h3` (or a `.settings-panel__section + .settings-panel__section` sibling rule). This is cosmetic and can be handled by adding a small rule like:

```css
.settings-panel__body h3 {
  /* existing rules... */
  margin-top: 24px;  /* add top spacing for sections after the first */
  padding-top: 20px;
  border-top: 1px solid #292e42;
}
.settings-panel__body h3:first-child {
  margin-top: 0;
  padding-top: 0;
  border-top: none;
}
```

This is additive CSS — does not break existing tests.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Scroll container | Custom JS scroll tracking | Native CSS `overflow-y: auto` | Already applied to `.settings-panel__body` — nothing to add |
| Section anchors | JS scroll-to-section | Skip entirely — not required by SETT-01/02/03 | Out of scope |
| Sticky headers | Intersection Observer | Skip — not required | Out of scope |

## Common Pitfalls

### Pitfall 1: Breaking Existing Source-Inspection Tests
**What goes wrong:** The existing tests in `SettingsTab.test.tsx` assert presence of tab-specific patterns like `"activeTab === 'appearance'"`, `settings-panel__tab-btn`, and `aria-selected`. Removing the tab UI will break these tests.
**Why it happens:** Tests were written to verify the tab-based UI is correctly implemented — they now verify the wrong thing.
**How to avoid:** Update test file in the same wave as the component refactor. New tests should assert: (1) no `settings-panel__tabs` div, (2) section headers present (Appearance, Web Server, Paths), (3) all content groups present unconditionally (theme select, CT disclosure, CLI table all visible in raw source simultaneously).
**Warning signs:** `pnpm test` fails on `SettingsTab.test.tsx` describe blocks after component edit.

### Pitfall 2: Forgetting to Remove Lifted State from App.tsx
**What goes wrong:** `settingsActiveTab` state and props remain in `App.tsx`, causing TypeScript errors after removing them from the `SettingsTabProps` interface.
**Why it happens:** The activeTab state was lifted to App.tsx (line 96) to persist across unmounts — it must be cleaned up in both places.
**How to avoid:** Remove all three: the `useState` declaration, the `activeTab={...}` prop, and the `onActiveTabChange={...}` prop in App.tsx simultaneously with the interface change.
**Warning signs:** TypeScript compile error: "Property 'activeTab' does not exist on type 'IntrinsicAttributes & SettingsTabProps'".

### Pitfall 3: Double Scroll (Nested overflow-y)
**What goes wrong:** Both `.settings-tab` and `.settings-panel__body` have `overflow-y: auto`. This is fine because `.settings-tab` is `flex-direction: column` and `.settings-panel__body` is the flex child that actually scrolls.
**Why it happens:** The outer `.settings-tab` `overflow-y: auto` is a safety fallback that shouldn't conflict.
**How to avoid:** No action needed — the existing CSS already handles this correctly. Removing the tab bar means more content fits in the body, making scrolling work naturally.
**Warning signs:** Content gets clipped without scrollbar. If observed, check that `.settings-panel__body` retains `flex: 1; overflow-y: auto`.

### Pitfall 4: style.settings.test.ts Asserting Retained Tab Classes
**What goes wrong:** `style.settings.test.ts` currently asserts `css.toContain('.settings-panel__tabs')` (line 40). After removing that CSS block, this test will fail.
**Why it happens:** The test was written to verify the tab nav CSS exists. After the refactor it should verify it does NOT exist.
**How to avoid:** Invert the test: assert `.settings-panel__tabs` is NOT in the CSS after refactor. Also assert the new section header CSS is present.

## Test Infrastructure

### Current State [VERIFIED: pnpm test run]
- **Framework:** vitest 4.1.0
- **Config:** `frontend/vite.config.ts` (vitest config inline)
- **Test command:** `cd frontend && pnpm test`
- **Suite status:** 323 tests passing in 18 test files
- **Relevant test files:**
  - `frontend/src/components/__tests__/SettingsTab.test.tsx` — will need updates
  - `frontend/src/components/__tests__/style.settings.test.ts` — will need updates
  - `frontend/src/components/__tests__/SettingsTab.web-link-ux.test.tsx` — likely OK (tests WEB-01/02/03 functionality, not tab structure)

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SETT-01 | No tab switcher UI visible; single scroll container | source-inspection | `cd frontend && pnpm test` | Wave 0 — update existing |
| SETT-02 | Section headers present (Appearance, Web Server, Paths) | source-inspection | `cd frontend && pnpm test` | Wave 0 — add assertions |
| SETT-03 | All content groups present unconditionally in source | source-inspection | `cd frontend && pnpm test` | Wave 0 — add assertions |

### Wave 0 Gaps
- [ ] `SettingsTab.test.tsx` — update describe blocks to remove tab-assertion tests, add scrollable-layout assertions
- [ ] `style.settings.test.ts` — update assertions: `.settings-panel__tabs` should NOT be present; section header h3 pattern should be present

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (inline vitest config) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite (323+ tests) green before `/gsd-verify-work`

## Environment Availability

Step 2.6: SKIPPED — no external dependencies. Pure frontend refactor touching one component file, one CSS file, and test files only.

## Security Domain

This phase makes no changes to authentication, data handling, cryptography, or network communication. All existing security properties (CT disclosure flow, Tailscale status gating, web server start/stop guard) are preserved unchanged. No ASVS categories apply to a layout refactor.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Modal-based settings | Inline sidebar tab | Prior phase (UI-02) | Settings lives at `__settings__` tab ID in the TabBar |
| Sub-tabs for organization | Scrollable sections (this phase) | Phase 69 | Simpler navigation, all settings visible without clicking |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Section order: Appearance, Web Server, Paths | Architecture Patterns | Low — SETT-02 only requires headers exist, not a specific order; planner can choose any order |
| A2 | `<h3>` tags are sufficient for SETT-02 section headers | Architecture Patterns | Low — existing CSS stub targets `h3`; a `<div>` with a dedicated class would also work |

## Open Questions

1. **Section order preference**
   - What we know: SETT-02 requires visible headers; no order specified
   - What's unclear: User preference for order (Appearance first vs. Web Server first)
   - Recommendation: Appearance, Web Server, Paths — most commonly used first

## Sources

### Primary (HIGH confidence)
- Codebase read: `frontend/src/components/SettingsTab.tsx` — full component source
- Codebase read: `frontend/src/App.tsx` — full app source, lifted state at line 96
- Codebase read: `frontend/src/style.css` — all settings CSS classes verified
- Codebase read: `frontend/src/components/__tests__/SettingsTab.test.tsx` — existing test patterns
- Codebase read: `frontend/src/components/__tests__/style.settings.test.ts` — CSS test patterns
- Codebase run: `pnpm test` — 323 tests passing baseline confirmed

## Metadata

**Confidence breakdown:**
- Current component structure: HIGH — read full source
- CSS class inventory: HIGH — read full style.css, ran grep verification
- Test impact analysis: HIGH — read all affected test files
- Section order recommendation: LOW — no user preference stated (marked ASSUMED)

**Research date:** 2026-04-11
**Valid until:** Stable — no external dependencies, pure codebase refactor
