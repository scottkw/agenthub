---
phase: 104-settings-hyperlinked-index
type: context
status: ready
mode: auto-generated
---

# Phase 104: Settings Hyperlinked Index

**Mode:** Auto-generated.

<domain>
## Phase Boundary

User can navigate the Settings tab via:
- **SETUI-01**: Sticky jump-link bar at top with anchor links to each section header (the existing h3s in SettingsTab.tsx: Behavior, Session Behavior, Appearance, Web Server, Security, Paths — plus Plugins from PluginsSection.tsx)
- **SETUI-02**: Clicking a jump-link smoothly scrolls to the target section
- **SETUI-03**: Autocomplete search box at top of Settings filters/jumps to specific settings by label

Scope: 3 requirements. Single phase. Issue #45.

Out of scope:
- Fuzzy-match into plugin sub-options (SETUI-03 covers section labels + top-level setting labels only)
- Settings overhaul beyond the index/search
</domain>

<decisions>
## Implementation Decisions — Claude's Discretion

- **JumpBar**: New component `frontend/src/components/SettingsJumpBar.tsx`. Sticky via `position: sticky; top: 0; z-index: 1`. List of `<a href="#section-id">` anchors. Optional active-section highlight via IntersectionObserver (nice-to-have, not required).
- **Section IDs**: Add `id="settings-{slug}"` to each h3 in SettingsTab.tsx + PluginsSection.tsx. Slugs: `behavior`, `session-behavior`, `appearance`, `web-server`, `security`, `paths`, `plugins`.
- **Smooth scroll**: Native CSS `scroll-behavior: smooth` on the settings container, OR JS `element.scrollIntoView({ behavior: 'smooth' })` on click.
- **Autocomplete**: Simple substring filter over a static list of setting labels (collected during the Phase 104 implementation pass). Render dropdown of matches; on click, scroll to that setting's anchor. Use a `data-setting-label="..."` attribute on each setting row so labels are discoverable from a single source.
- **Sticky offset**: Account for jump-bar height when scrolling to anchors (use `scroll-margin-top` CSS on each section header).
</decisions>

<code_context>
## Existing Code Insights

- `frontend/src/components/SettingsTab.tsx` — owns 6 h3 section headers
- `frontend/src/components/PluginsSection.tsx` — owns the 7th (Plugins)
- No existing autocomplete/combobox component — build inline or simple dropdown
</code_context>

<specifics>
## Specific Ideas

1. New file `SettingsJumpBar.tsx` — sticky strip with 7 anchor links.
2. New file `SettingsSearch.tsx` — input + dropdown list of matching labels.
3. Add `id` attributes to each h3.
4. Add `data-setting-label` on each settings row that's searchable.
5. Test: jump-bar renders with 7 links; click jumps to anchor; search filters list; click result scrolls.
</specifics>

<deferred>
## Deferred
- Active-section highlight via IntersectionObserver
- Deep search into plugin sub-options
- Keyboard arrow navigation in autocomplete
</deferred>
