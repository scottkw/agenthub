# Phase 81: Banner Notifications - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-16
**Phase:** 81-banner-notifications
**Areas discussed:** Stacking layout, Dismiss behavior, Banner unification, Visual consistency

---

## Stacking Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Flex column container | Wrap all banners in a single flex-direction: column container at the top of .app | ✓ |
| Absolute overlay stack | Position banners as an absolute/fixed overlay at the top, stacking downward | |
| You decide | Claude picks the best approach | |

**User's choice:** Flex column container
**Notes:** Recommended option — natural vertical stacking with consistent gap

| Option | Description | Selected |
|--------|-------------|----------|
| Show all | Display every active banner, no cap | |
| Cap at 3 with scroll | Max 3 visible, container scrolls if more | ✓ |
| You decide | Claude picks based on realistic banner count | |

**User's choice:** Cap at 3 with scroll
**Notes:** None

---

## Dismiss Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Right-aligned X | X button at far right of each banner row | ✓ |
| Inline dismiss text | Small 'Dismiss' text link alongside banner actions | |
| You decide | Claude picks the dismiss control style | |

**User's choice:** Right-aligned X
**Notes:** Consistent with existing update-banner dismiss pattern

| Option | Description | Selected |
|--------|-------------|----------|
| Make it dismissible | Add X to LocalNetworkBanner, hides for session | ✓ |
| Keep non-dismissible | LocalNetworkBanner stays always-visible in local mode | |
| Dismissible with memory | Persist dismiss across sessions until state changes | |

**User's choice:** Make it dismissible (session-scoped)
**Notes:** Reappears on next app launch or if conditions change

| Option | Description | Selected |
|--------|-------------|----------|
| Instant removal | Banner disappears immediately | |
| Fade + collapse | Fade out and collapse height over ~200ms | ✓ |
| You decide | Claude picks based on existing animation patterns | |

**User's choice:** Fade + collapse
**Notes:** ~200ms CSS transition

---

## Banner Unification

| Option | Description | Selected |
|--------|-------------|----------|
| Unify at app level | Move update banner to same top-level container as LocalNetworkBanner | ✓ |
| Keep separate | Update banner stays inside WelcomeTab, fix CSS only | |
| You decide | Claude picks the approach that best fixes #28 | |

**User's choice:** Unify at app level
**Notes:** All notification banners live in one stacking zone — directly fixes GitHub #28

| Option | Description | Selected |
|--------|-------------|----------|
| Keep inline | Daemon error stays in content area as contextual state message | ✓ |
| Move to banner stack | Treat daemon error as a banner too | |
| You decide | Claude decides based on UX patterns | |

**User's choice:** Keep inline
**Notes:** Daemon error is contextual — appears in content area when no sessions exist, has retry button

---

## Visual Consistency

| Option | Description | Selected |
|--------|-------------|----------|
| Shared structure, unique accents | All banners use same layout, unique accent colors per type | ✓ |
| Keep distinct styles | Each banner keeps its own visual design | |
| You decide | Claude unifies CSS based on existing design language | |

**User's choice:** Shared structure, unique accents
**Notes:** Amber for warnings/LocalNetwork, blue for info/updates

---

## Claude's Discretion

- BannerStack component implementation approach
- CSS transition timing and easing
- Update state lifting strategy
- Shared Banner base component vs individual components with shared CSS
- Banner ordering in the stack

## Deferred Ideas

None — discussion stayed within phase scope
