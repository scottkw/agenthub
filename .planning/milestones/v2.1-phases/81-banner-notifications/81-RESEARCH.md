# Phase 81: Banner Notifications - Research

**Researched:** 2026-04-16
**Domain:** React component architecture, CSS transitions, state lifting
**Confidence:** HIGH

## Summary

Phase 81 is a pure frontend refactor. No backend changes are needed. Two components need to be created or modified, state lifted from WelcomeTab to App level, and CSS transitions added for dismiss animation. The codebase already has a nearly complete blueprint for this work: the UI-SPEC.md (approved by gsd-ui-researcher) provides exact CSS values, color tokens, spacing, and markup structure. Everything is hand-coded BEM CSS with a Tokyo Night dark palette — no UI framework, no Tailwind, no shadcn.

The core work is three things: (1) create a new `BannerStack` wrapper div in `App.tsx` that holds both banners in a flex-direction:column container, (2) extract the update banner from `WelcomeTab` into a standalone `UpdateBanner` component at app level while lifting the `update` state and the `update:available`/`GetLastUpdateInfo` subscriptions to `App`, and (3) add an `onDismiss` prop to `LocalNetworkBanner` plus a dismiss X button (XMarkIcon from @heroicons/react/20/solid). Dismiss animation uses a `.banner-exit` CSS class toggled immediately, then the item is removed from state after a 200ms `setTimeout`.

The test suite currently passes 418 tests across 20 files. Tests for `LocalNetworkBanner` and `WelcomeTab` will need to be updated to reflect structural changes; new tests for `UpdateBanner` and the `BannerStack` integration in `App` will be added.

**Primary recommendation:** Lift update state to App, build BannerStack as a div with CSS (not a dedicated component file), keep LocalNetworkBanner as the existing file with a new `onDismiss` prop, and create a new `UpdateBanner.tsx` component.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Wrap all banners in a single flex-direction: column container (`BannerStack`) at the top of `.app`, above `.app__row`. Each banner is a row within the container with consistent gap.
- **D-02:** Cap visible banners at 3 — if more than 3 are active, the banner container gets a max-height and scrolls internally.
- **D-03:** Every banner gets a right-aligned X button for dismiss. Consistent placement at the far right of each banner row.
- **D-04:** LocalNetworkBanner becomes dismissible (currently has no dismiss control). Dismissing hides it for the current session; it reappears on next app launch or if conditions change.
- **D-05:** Banners animate on dismiss — fade out + height collapse over ~200ms using CSS transitions.
- **D-06:** Move the update banner out of WelcomeTab into the app-level banner stack alongside LocalNetworkBanner. All notification banners live in one stacking zone.
- **D-07:** Daemon error stays inline in the content area — it's a contextual state message with a retry button, not a notification banner.
- **D-08:** All banners share the same layout structure: icon + message + optional sub-text + optional CTA + dismiss X. Each banner type gets a unique accent color for its left border and icon (amber for warnings/LocalNetwork, blue for info/updates).

### Claude's Discretion
- Implementation of the BannerStack component (whether it's a dedicated component or just a wrapper div with CSS)
- Exact CSS transition timing and easing for fade + collapse
- How update state is lifted from WelcomeTab to App level (useState at App level vs context)
- Whether to create a shared Banner base component or keep individual banner components that follow the same CSS pattern
- Order of banners in the stack when multiple are active

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BAN-01 | When multiple notifications are active, they stack vertically instead of side-by-side | D-01 + D-02 + BannerStack container with flex-direction:column in style.css |
| BAN-02 | Each stacked notification remains independently dismissible | D-03 + D-04 + D-05: separate onDismiss callbacks per banner, CSS transition removes individual items |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Banner stacking layout | Frontend (React + CSS) | — | Pure presentation layer; no backend state needed |
| Banner dismiss state | Frontend App component | — | Session-local state, already pattern-matched to existing `useState` at App level |
| Update state lifting | Frontend App component | — | `update:available` event and `GetLastUpdateInfo()` already emitted from Go; just move the subscriber up the tree |
| Dismiss animation | CSS (`.banner-exit` class) | React `setTimeout` | CSS transitions handle visual; React handles state removal after transition completes |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2.4 | Component rendering, state management | Project standard [VERIFIED: package.json] |
| TypeScript | 5.9.3 | Type safety | Project standard [VERIFIED: package.json] |
| @heroicons/react | ^2.2.0 | Dismiss X icon (XMarkIcon from /20/solid) | Already installed; used for icon needs [VERIFIED: package.json] |
| Vitest | 4.1.0 | Test framework | Project standard [VERIFIED: package.json] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jsdom | 29.0.0 | DOM environment for tests | Test environment (already configured in vitest) [VERIFIED: package.json] |

### No new dependencies needed
All libraries are already installed. `@heroicons/react` provides the XMarkIcon. No new npm installs required.

## Architecture Patterns

### System Architecture Diagram

```
App.tsx (root)
  |
  |-- BannerStack (flex-column wrapper div, flex-shrink:0, max-height capped at 3 banners)
  |     |-- LocalNetworkBanner (if webServerMode='local' AND NOT localBannerDismissed)
  |     |     props: visible, tailscaleConnected, tailscaleBinaryFound, tailscaleDaemonUp,
  |     |            platformHint, onOpenURL, onDismiss
  |     |
  |     +-- UpdateBanner (if update !== null)
  |           props: update: UpdateInfo | null, onDismiss
  |
  +-- .app__row (sidebar + content — unchanged)

State owned by App:
  - webServerMode (existing)
  - tailscaleHealth (existing)
  - localBannerDismissed: boolean  (NEW — session-local, no persistence)
  - update: UpdateInfo | null      (LIFTED from WelcomeTab)

Event subscriptions moved to App:
  - EventsOn('update:available', ...) (MOVED from WelcomeTab)
  - GetLastUpdateInfo() on mount       (MOVED from WelcomeTab)
```

### Recommended Project Structure
```
frontend/src/
├── components/
│   ├── LocalNetworkBanner.tsx    # MODIFY: add onDismiss prop + dismiss X button
│   ├── UpdateBanner.tsx          # NEW: extracted from WelcomeTab, app-level props
│   ├── WelcomeTab.tsx            # MODIFY: remove update state + banner markup
│   └── __tests__/
│       ├── LocalNetworkBanner.test.tsx  # UPDATE: add dismiss tests
│       ├── UpdateBanner.test.tsx        # NEW: banner render + dismiss tests
│       └── App.test.tsx                 # UPDATE: add BannerStack integration tests
├── App.tsx                       # MODIFY: add BannerStack, lift update state
└── style.css                     # MODIFY: add .banner-stack, .banner-exit CSS
```

### Pattern 1: Dismiss Animation (D-05)
**What:** Apply `.banner-exit` CSS class on dismiss click, then remove from state after transition completes.
**When to use:** Both LocalNetworkBanner and UpdateBanner dismiss flows.

```typescript
// Source: UI-SPEC.md §Dismiss animation
// In App.tsx — handler for LocalNetworkBanner dismiss:
const handleDismissLocalBanner = useCallback(() => {
  setLocalBannerExiting(true)
  setTimeout(() => {
    setLocalBannerDismissed(true)
    setLocalBannerExiting(false)
  }, 200)
}, [])
```

CSS (in style.css):
```css
/* Source: UI-SPEC.md §Dismiss animation */
.banner-exit {
  opacity: 0;
  max-height: 0;
  overflow: hidden;
  padding-top: 0;
  padding-bottom: 0;
  transition: opacity 150ms ease, max-height 200ms ease, padding 200ms ease;
}
```

The banner receives `className={isExiting ? 'local-network-banner banner-exit' : 'local-network-banner'}`.

### Pattern 2: BannerStack Container (D-01, D-02)
**What:** A div wrapper (not a dedicated component file) rendered as the first child of `.app`.
**When to use:** Always rendered; returns early as empty fragment when 0 banners are active.

```tsx
// Source: UI-SPEC.md §BannerStack, Context.md D-01/D-02
// In App.tsx JSX:
{(showLocalBanner || update) && (
  <div className="banner-stack">
    {showLocalBanner && (
      <LocalNetworkBanner
        visible={true}
        ...
        onDismiss={handleDismissLocalBanner}
        className={localBannerExiting ? 'banner-exit' : undefined}
      />
    )}
    {update && (
      <UpdateBanner
        update={update}
        onDismiss={handleDismissUpdate}
        className={updateExiting ? 'banner-exit' : undefined}
      />
    )}
  </div>
)}
```

BannerStack CSS:
```css
/* Source: UI-SPEC.md §BannerStack */
.banner-stack {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  max-height: calc(3 * 53px);   /* D-02: cap at 3 banners */
  overflow-y: auto;
}
```

### Pattern 3: State Lifting for Update (D-06)
**What:** Move `update` state + subscriptions from `WelcomeTab` to `App`.
**When to use:** UpdateBanner now lives at App level; WelcomeTab no longer owns update state.

```typescript
// Source: WelcomeTab.tsx (existing pattern to move)
// In App.tsx — add these:
const [update, setUpdate] = useState<UpdateInfo | null>(null)

useEffect(() => {
  GetLastUpdateInfo()
    .then((info) => { if (info) setUpdate(info) })
    .catch(() => {})
  const offUpdate = EventsOn('update:available', (info: UpdateInfo) => {
    setUpdate(info)
  })
  return () => { offUpdate() }
}, [])
```

`WelcomeTab` removes this state and these subscriptions entirely.

### Pattern 4: UpdateInfo type
**What:** UpdateInfo is already defined inside WelcomeTab. It needs to be either extracted to a shared types file or re-declared at App level.
**Recommendation:** Declare it at App level (where it's consumed) and remove from WelcomeTab. No separate types file needed at this scale.

```typescript
// Move to App.tsx (or a shared location):
interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}
```

### Pattern 5: Dismiss X Button (D-03, D-08)
**What:** Right-aligned ghost button using XMarkIcon from @heroicons/react/20/solid.
**When to use:** Both banner types.

```tsx
// Source: UI-SPEC.md §LocalNetworkBanner, §Banner structure
import { XMarkIcon } from '@heroicons/react/20/solid'

// Inside banner JSX (far right of flex row):
<button
  type="button"
  className="local-network-banner__dismiss"
  aria-label="Dismiss local network notification"
  onClick={onDismiss}
>
  <XMarkIcon style={{ width: 16, height: 16 }} />
</button>
```

CSS for dismiss button (same pattern for both banner types):
```css
.local-network-banner__dismiss {
  margin-left: auto;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  color: #9aa5ce;
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  min-height: 24px;
  flex-shrink: 0;
}
.local-network-banner__dismiss:hover {
  color: #c0caf5;
  border-color: #3b4261;
}
.local-network-banner__dismiss:focus-visible {
  outline: 2px solid #7aa2f7;
  outline-offset: 2px;
}
```

### Anti-Patterns to Avoid
- **Putting dismiss state inside LocalNetworkBanner:** Dismiss needs to survive re-renders; state must live in App. [VERIFIED: existing pattern — all UI state in App.tsx]
- **Removing banner from DOM immediately on click:** CSS transition needs the element to be present; remove only after 200ms timeout.
- **Using `display: none` to hide banners:** Use opacity + max-height transitions for smooth animation, not display toggling.
- **Separate "dismissed" flag per banner type stored in localStorage:** D-04 explicitly says session-only, no persistence. Use `useState` only.
- **Passing `update` prop to WelcomeTab after extraction:** WelcomeTab should no longer know about update state at all.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Close/X icon | SVG string or unicode X | `XMarkIcon` from `@heroicons/react/20/solid` | Already installed, consistent 16px, accessible |
| Animation library | framer-motion, react-transition-group | CSS `.banner-exit` class + `setTimeout(200)` | Zero new dependencies; transitions handle the visual |

**Key insight:** React transition libraries add hundreds of KB for what is 5 lines of CSS + one setTimeout.

## Common Pitfalls

### Pitfall 1: WelcomeTab test breakage after state lift
**What goes wrong:** `WelcomeTab.test.tsx` currently asserts that `WelcomeTab` subscribes to `update:available`, imports `GetLastUpdateInfo`, renders `.update-banner` with `role="alert"`, etc. After extraction, all of these tests will fail.
**Why it happens:** Tests are structural/contract tests that inspect the component's raw source via `?raw` import.
**How to avoid:** Update `WelcomeTab.test.tsx` to remove the `update banner (UPD-02, UPD-03)` describe block; add equivalent tests in new `UpdateBanner.test.tsx`.
**Warning signs:** Any test that uses `import raw from '../../components/WelcomeTab.tsx?raw'` and checks for `update-banner` or `EventsOn`.

### Pitfall 2: App.test.tsx importing LocalNetworkBanner directly
**What goes wrong:** `App.test.tsx` asserts `raw.toContain('<LocalNetworkBanner')`. After the BannerStack refactor, the render location changes but the component name stays. This test should still pass as-is, but the conditional wrapping changes from `{webServerMode === 'local' && (<LocalNetworkBanner .../>)}` to a BannerStack wrapper. If tests check for the exact conditional, they may fail.
**How to avoid:** Run tests after each structural change. The `?raw` pattern checks for string presence, which is robust to wrapper changes.

### Pitfall 3: Exiting banner leaves gap before removal
**What goes wrong:** During the 200ms exit animation, `max-height: 0` collapses the banner row but if `padding` isn't also transitioned, a visual gap remains.
**Why it happens:** The element's padding contributes to layout height separately from `max-height`.
**How to avoid:** The `.banner-exit` transition must include `padding 200ms ease` alongside `max-height`. The UI-SPEC.md already specifies this correctly.

### Pitfall 4: `localBannerDismissed` not reset when conditions change
**What goes wrong:** D-04 says the banner should reappear if conditions change (e.g., webServerMode returns to 'local' after a mode switch). If dismissed state is a simple boolean, it will block re-display.
**How to avoid:** The dismiss state is `localBannerDismissed`. The banner shows when `webServerMode === 'local' && !localBannerDismissed`. When webServerMode changes away from 'local' and back, `localBannerDismissed` should reset. Implement by watching `webServerMode` in a `useEffect` and resetting `localBannerDismissed` when the mode transitions to non-local.

### Pitfall 5: Stacking order inconsistency
**What goes wrong:** If the order (LocalNetwork first, Update second) is implemented differently than the UI-SPEC, the visual hierarchy is reversed.
**How to avoid:** UI-SPEC §Interaction Contract specifies top-to-bottom order: LocalNetworkBanner first (system warning), UpdateBanner second (informational). Honor this ordering in the BannerStack JSX.

## Code Examples

### LocalNetworkBanner with onDismiss prop

```tsx
// Source: UI-SPEC.md §LocalNetworkBanner, canonical_refs LocalNetworkBanner.tsx
interface LocalNetworkBannerProps {
  visible: boolean
  tailscaleConnected: boolean
  tailscaleInstalled: boolean
  tailscaleBinaryFound: boolean
  tailscaleDaemonUp: boolean
  platformHint: string
  onOpenURL: (url: string) => void
  onDismiss?: () => void           // NEW
  className?: string               // NEW for banner-exit animation
}
```

### UpdateBanner component (extracted)

```tsx
// Source: WelcomeTab.tsx §update banner (existing markup to extract)
import React from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'

interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}

interface UpdateBannerProps {
  update: UpdateInfo
  onDismiss: () => void
  className?: string
}

export function UpdateBanner({ update, onDismiss, className }: UpdateBannerProps): React.ReactElement {
  return (
    <div className={`update-banner${className ? ' ' + className : ''}`} role="alert" aria-live="polite">
      <span className="update-banner__message">
        Update available:{' '}
        <span className="update-banner__version">{update.currentVersion}</span>
        {' '}<span className="update-banner__arrow">&rarr;</span>{' '}
        <span className="update-banner__version">{update.latestVersion}</span>
      </span>
      <div className="update-banner__actions">
        <button
          type="button"
          className="update-banner__btn--download"
          onClick={() => BrowserOpenURL(update.releaseURL)}
        >
          Download Update
        </button>
        <button
          type="button"
          className="update-banner__btn--dismiss"
          aria-label="Dismiss update notification"
          onClick={onDismiss}
        >
          Dismiss
        </button>
      </div>
    </div>
  )
}
```

Note: The existing `.update-banner__btn--dismiss` text button ("Dismiss") is kept. The UI-SPEC retains this existing dismiss button rather than replacing it with an X icon, for the update banner specifically.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Update banner inside WelcomeTab | UpdateBanner at App level | Phase 81 | Enables app-level stacking; WelcomeTab becomes display-only |
| LocalNetworkBanner without dismiss | LocalNetworkBanner with onDismiss prop | Phase 81 | BAN-02 compliance |
| Single banner possible (side-by-side layout bug) | BannerStack with flex-column | Phase 81 | BAN-01 fix |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `UpdateInfo` type can be moved to App.tsx scope without a shared types file | Architecture Patterns §4 | Low risk — if WelcomeTab also needs the type, import from App or create a shared types file |
| A2 | `className` prop passthrough is the right API for exit animation | Pattern 1 | If LocalNetworkBanner renders multiple roots conditionally, class must target each root separately |

## Open Questions

1. **Should `localBannerDismissed` reset when `webServerMode` toggles?**
   - What we know: D-04 says "reappears on next app launch or if conditions change"
   - What's unclear: "conditions change" — does toggling webServerMode off and back to 'local' count?
   - Recommendation: Yes, reset `localBannerDismissed = false` when `webServerMode` transitions to `'local'` from a non-local state. This matches the intent.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — this phase is a pure frontend code/CSS change with no new tooling, CLIs, or services required).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (test: { environment: 'jsdom', globals: true }) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test` |

All 418 tests currently passing. [VERIFIED: pnpm test output 2026-04-16]

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BAN-01 | Multiple active banners appear stacked vertically in a single flex-column container | structural | `cd frontend && pnpm test` | ❌ Wave 0 (App.test.tsx needs BannerStack assertions) |
| BAN-01 | BannerStack renders before .app__row | structural | `cd frontend && pnpm test` | ❌ Wave 0 |
| BAN-02 | Dismissing LocalNetworkBanner does not affect UpdateBanner | structural/behavioral | `cd frontend && pnpm test` | ❌ Wave 0 |
| BAN-02 | Each banner has its own dismiss control | structural | `cd frontend && pnpm test` | ❌ Wave 0 (LocalNetworkBanner.test.tsx needs dismiss tests) |
| BAN-02 | UpdateBanner renders dismiss button | structural | `cd frontend && pnpm test` | ❌ Wave 0 (UpdateBanner.test.tsx new file) |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test`
- **Per wave merge:** `cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/UpdateBanner.test.tsx` — new file, covers BAN-02 (update banner dismiss)
- [ ] `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` — extend with dismiss button tests (covers BAN-02)
- [ ] `frontend/src/components/__tests__/App.test.tsx` — extend with BannerStack integration tests (covers BAN-01)
- [ ] `frontend/src/components/__tests__/WelcomeTab.test.tsx` — remove update-banner describe block (currently tests will break after extraction)

Note: Test infrastructure (vitest, jsdom, config) is already fully established. No framework install needed.

## Security Domain

This phase has no security-relevant changes. It adds dismiss controls to notification banners and rearranges component structure. No authentication, cryptography, access control, or input validation concerns apply. `security_enforcement` section skipped.

## Sources

### Primary (HIGH confidence)
- `frontend/src/components/LocalNetworkBanner.tsx` — [VERIFIED: Read] Current component, full props interface, existing render branches
- `frontend/src/components/WelcomeTab.tsx` — [VERIFIED: Read] Update banner markup to extract, existing `UpdateInfo` interface, Wails subscriptions
- `frontend/src/App.tsx` lines 1–130, 580–700 — [VERIFIED: Read] State patterns, existing banner render location, tailscaleHealth shape
- `frontend/src/style.css` lines 56–62, 940–1005, 1441–1494 — [VERIFIED: Read] BEM CSS patterns, existing banner colors/spacing
- `.planning/phases/81-banner-notifications/81-UI-SPEC.md` — [VERIFIED: Read] Exact CSS values, spacing tokens, component inventory, dismiss animation spec
- `.planning/phases/81-banner-notifications/81-CONTEXT.md` — [VERIFIED: Read] Locked decisions D-01 through D-08
- `frontend/package.json` — [VERIFIED: Read] @heroicons/react ^2.2.0 installed, vitest 4.1.0
- `pnpm test output (2026-04-16)` — [VERIFIED: Bash] 418 tests passing baseline
- `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` — [VERIFIED: Read] Existing test patterns (createRoot + flushSync behavioral, ?raw structural)
- `frontend/src/components/__tests__/App.test.tsx` — [VERIFIED: Read] Existing App integration test patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries verified from package.json
- Architecture: HIGH — all patterns derived directly from existing codebase (no external research needed)
- Pitfalls: HIGH — derived from reading existing test suite and component structure
- CSS patterns: HIGH — exact values from UI-SPEC.md and style.css

**Research date:** 2026-04-16
**Valid until:** 2026-05-16 (stable React + BEM CSS — no fast-moving dependencies)
