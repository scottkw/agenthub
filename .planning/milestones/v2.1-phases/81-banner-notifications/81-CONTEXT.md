# Phase 81: Banner Notifications - Context

**Gathered:** 2026-04-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix banner stacking so multiple active notification banners stack vertically (one above the other) with independent dismiss controls. Unify the two existing banner systems (LocalNetworkBanner at app level, update-banner inside WelcomeTab) into a single stacking container at the app level. Two requirements: BAN-01 (vertical stacking) and BAN-02 (independent dismiss).

</domain>

<decisions>
## Implementation Decisions

### Stacking Layout
- **D-01:** Wrap all banners in a single flex-direction: column container (`BannerStack`) at the top of `.app`, above `.app__row`. Each banner is a row within the container with consistent gap.
- **D-02:** Cap visible banners at 3 — if more than 3 are active, the banner container gets a max-height and scrolls internally.

### Dismiss Behavior
- **D-03:** Every banner gets a right-aligned X button for dismiss. Consistent placement at the far right of each banner row.
- **D-04:** LocalNetworkBanner becomes dismissible (currently has no dismiss control). Dismissing hides it for the current session; it reappears on next app launch or if conditions change.
- **D-05:** Banners animate on dismiss — fade out + height collapse over ~200ms using CSS transitions.

### Banner Unification
- **D-06:** Move the update banner out of WelcomeTab into the app-level banner stack alongside LocalNetworkBanner. All notification banners live in one stacking zone.
- **D-07:** Daemon error stays inline in the content area — it's a contextual state message with a retry button, not a notification banner.

### Visual Consistency
- **D-08:** All banners share the same layout structure: icon + message + optional sub-text + optional CTA + dismiss X. Each banner type gets a unique accent color for its left border and icon (amber for warnings/LocalNetwork, blue for info/updates).

### Claude's Discretion
- Implementation of the BannerStack component (whether it's a dedicated component or just a wrapper div with CSS)
- Exact CSS transition timing and easing for fade + collapse
- How update state is lifted from WelcomeTab to App level (useState at App level vs context)
- Whether to create a shared Banner base component or keep individual banner components that follow the same CSS pattern
- Order of banners in the stack when multiple are active

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Current Banner Components
- `frontend/src/components/LocalNetworkBanner.tsx` — Current LocalNetworkBanner (to be made dismissible, moved into BannerStack)
- `frontend/src/components/WelcomeTab.tsx` §update-banner — Current update banner (to be extracted and moved to app-level)

### App Layout
- `frontend/src/App.tsx` §580-592 — Where LocalNetworkBanner is currently rendered (above `.app__row`)
- `frontend/src/App.tsx` §653-689 — Inline daemon error panel (stays as-is per D-07)

### CSS
- `frontend/src/style.css` §1441-1485 — `.local-network-banner` styles
- `frontend/src/style.css` §940-1005 — `.update-banner` styles
- `frontend/src/style.css` §56-62 — `.app` flex column layout

### Tests
- `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` — Existing banner tests (need updating for dismiss)
- `frontend/src/components/__tests__/App.test.tsx` — App integration tests (need updating for BannerStack)

### Requirements
- `.planning/REQUIREMENTS.md` §BAN-01, §BAN-02 — Vertical stacking and independent dismiss requirements
- GitHub issue #28 — Original bug report: banners cramped side-by-side

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `LocalNetworkBanner` component: Already has the icon + message + sub-text + CTA pattern; needs dismiss X added
- `.local-network-banner` CSS: BEM-style naming established; extend for shared banner base class
- `update-banner` in WelcomeTab: Has dismiss functionality via `setUpdate(null)`; logic to be lifted to App level

### Established Patterns
- BEM CSS naming: `.local-network-banner__icon`, `.local-network-banner__message`, etc.
- Flex layout: `.app` is already `flex-direction: column` — BannerStack fits naturally as a child before `.app__row`
- State management: React useState at App level for UI state (e.g., `webServerMode`, `tailscaleHealth`)

### Integration Points
- `App.tsx` renders `LocalNetworkBanner` — replace with `BannerStack` containing both banners
- `WelcomeTab` update state (`update` / `setUpdate`) — needs to be lifted to App or passed via props/context
- `startHealthPoller` in App — already emits tailscale health state; banner visibility derives from this

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 81-banner-notifications*
*Context gathered: 2026-04-16*
