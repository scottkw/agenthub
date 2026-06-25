# Phase 147: In-App Help Page - Context

**Gathered:** 2026-06-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver an in-app **Help page** (HELP-01, GitHub #69) in the AgentHub desktop GUI providing:
1. **Documentation** — "Getting Started / Basic Operation" content (session creation, shells, session switching, file browser, web share, settings overview, keyboard shortcuts).
2. **FAQ** — seeded with the common questions from #69 (DevTools-in-prod, network sharing, remote file browse, where sessions/logs live, updating, reporting bugs).
3. **Search** — live, debounced filter over both doc and FAQ content with matched-term highlighting and context snippets.
4. **External links** — GitHub repo and the marketing/product site, opening in the user's default external browser.

The page is reachable from the main app navigation as a new top-level surface.

**In scope:** the GUI Help page (navigation entry, content rendering, search, external links) for the Wails desktop app.
**Out of scope:** remote-fetched/dynamic content (Option B), a web-share-viewer Help surface, a dedicated `agenthub help` CLI command, FAQ auto-generation from GitHub issues, and any content beyond the seeded sections (maintainer authors/expands content over time).
</domain>

<decisions>
## Implementation Decisions

### Navigation placement
- **D-01:** Help becomes a **4th sidebar item** — sidebar order is now **Home / Hub / Settings / Help**. This deliberately reopens the Phase-138 (NAV-02..05) decision to keep the sidebar at exactly three items, because Help is a genuine top-level surface and the sidebar is the most discoverable entry point.
- **D-02:** Help opens as its own **special tab** (pattern: `__help__`), mirroring how Settings (`__settings__`) and Hub (`__hub__`) are modeled in `App.tsx` — not a modal or popover.

### Cross-surface scope (parity reconciliation)
- **D-03:** **GUI only.** The cross-surface parity contract for v4.0 is GUI/CLI/web (TUI dropped). The CLI's existing native `--help`/help output satisfies the CLI side of parity — no new `agenthub help` content command is in scope. Web-share viewers are scoped to a single shared session and do not need the Help page.
- **D-04:** Issue #69's "TUI parity is release-blocking" section is **obsolete** and explicitly superseded by this decision — the TUI was removed in v4.0. No TUI Help and no follow-up TUI issue is required.

### Content source & format
- **D-05:** **Option A — bundled Markdown** files committed in the repo and rendered at runtime. Chosen for offline operation and zero new network failure modes; content updates ship with app releases (acceptable for v1). Remote fetch (Option B) is explicitly deferred.
- **D-06:** Content is **maintainer-authored / hand-curated**. FAQ is seeded from the #69 suggested set and reviewed by the maintainer — NOT auto-scraped from closed GitHub issues.

### Search behavior
- **D-07:** **Rich search.** Single search input pinned at the top; debounced live filtering over both documentation and FAQ **body** content. Matched terms are highlighted (e.g., `<mark>`) inside ~1–2 line context snippets, each with a "jump to section" affordance.
- **D-08:** **Empty state** when no matches: a "No results for '<query>'" message that points to GitHub issues for known topics (per #69).
- **D-09:** This is richer than the existing `SettingsSearch` (which matches section labels only). `SettingsSearch` is a structural analog for the debounce + jump-to-anchor mechanics, not a drop-in — the body-level snippet/highlight indexing is new.

### Layout & accessibility (per #69, locked)
- **D-10:** Layout = **left section-nav** (Getting Started, FAQ, …) + **right content pane**, search input pinned at the top. Scannable content: short paragraphs, headings, code/keyboard styling.
- **D-11:** Respect existing **theme tokens** (light/dark) — no bespoke palette. Must honor colorblind-safe + `prefers-reduced-motion` release norms.
- **D-12:** Accessibility: search input has a visible/associated **label**; sections use correct heading hierarchy (h1 → h2 → h3); external links carry an `aria-label` indicating they open externally. Search + section nav are keyboard accessible.

### External links
- **D-13:** GitHub repo and website links open in the **external default browser** via the established `BrowserOpenURL` Wails runtime pattern (as used in `SettingsTab.tsx`). They do NOT open inside the app webview.

### Claude's Discretion
- **Markdown rendering approach** — choice between a runtime renderer library (e.g. react-markdown) vs. build-time Markdown→HTML, including the XSS-safe strategy for injecting `<mark>` highlight spans into rendered content. Left to research/planning; must be sanitization-safe.
- **Search index granularity** — whether to index per-section, per-paragraph, or per-heading-block for snippet extraction; pick whatever yields good snippet quality.
- **Exact icon** for the Help sidebar item (a heroicons outline icon consistent with the existing sidebar set, e.g. QuestionMarkCircleIcon).
- **Keyboard-shortcuts doc content** — include only shortcuts that are actually documented/implemented; omit the section if none exist.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirement & issue source
- `.planning/REQUIREMENTS.md` — HELP-01 definition (Phase 147).
- GitHub issue **#69** (`scottkw/agenthub`) — full feature spec: content sections, search/highlight behavior, layout/UX notes, accessibility, content-source options, and acceptance criteria. The authoritative content/AC source. NOTE: its "Cross-Surface → TUI" section is obsolete (see D-03/D-04).

### Milestone scope context
- `.planning/PROJECT.md` — v4.0 milestone scope; TUI-dropped decision and the GUI/CLI/web parity contract that drives D-03/D-04; colorblind-safe + reduced-motion + theme-token release norms.
- `.planning/ROADMAP.md` §"Phase 147" — goal and success criteria.

### Cross-surface parity norm
- `CLAUDE.md` (repo root) + memory note "Cross-surface parity is release-blocking" — parity is release-blocking; the reconciliation in D-03/D-04 must be explicit, not silent.

### Regression-test convention
- `TESTING.md` (repo root) — new test files must be registered (Suite Manifest §2, Traceability §4); run `bash tests/check-traceability-paths.sh` before committing. Add an M-NN manual item only if a behavior can't be automated.
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/components/Sidebar.tsx` — add the 4th item here (Home/Hub/Settings/Help). Settings lives in `sidebar__bottom`; Help placement relative to it is a layout choice. Needs a new `onOpenHelp` prop + `activePanel === '__help__'` active state.
- `frontend/src/App.tsx` — special-tab routing model (`__welcome__`, `__settings__`, `__hub__`). Add a `HELP_TAB` (`__help__`) + `handleOpenHelp` callback following the existing Settings pattern.
- `frontend/src/components/SettingsSearch.tsx` — structural analog for debounced search + native hash-jump to section anchors (`scroll-margin-top` to clear sticky bars). Body-level snippet/highlight indexing is NOT present here and is new work (D-09).
- `frontend/src/components/SettingsJumpBar.tsx` — pattern for a sticky left/top section-nav with anchor links; analog for the Help left section-nav (D-10).
- `BrowserOpenURL` from `../wailsjs/wailsjs/runtime/runtime` (used in `SettingsTab.tsx:609`) — external-browser link opening (D-13).

### Established Patterns
- Special tabs are non-session `Tab` objects with a `type` discriminator and an `id` like `__settings__`; the sidebar button toggles them and reflects active state via `activePanel`. Help follows this exactly.
- Settings tab already demonstrates a searchable, anchor-navigated, theme-token-styled content surface — the closest structural precedent for the Help page.

### Integration Points
- Sidebar: new `onOpenHelp` prop wired from `App.tsx`; active indicator via `activePanel === '__help__'`.
- App tab state: `HELP_TAB` constant + open handler; render a new `HelpTab`/`HelpPage` component when the help tab is active.
- Bundled Markdown content: a new content directory under `frontend/src/` (or assets) — exact location is a planning detail; must be importable/embeddable so it works offline.
</code_context>

<specifics>
## Specific Ideas

- Sidebar must read **Home / Hub / Settings / Help** after this phase.
- FAQ seed set is exactly the questions enumerated in #69 (DevTools-in-prod → use `wails dev`/web-share to Chrome; network sharing; remote file browse; where sessions/logs are stored; updating AgentHub; reporting bugs → GitHub issues link).
- Search empty state should point users to GitHub issues for known topics.
- "DevTools doesn't open in production" FAQ aligns with the known project constraint (Wails DevTools disabled in prod) — answer should match that established behavior.
</specifics>

<deferred>
## Deferred Ideas

- **Remote/dynamic Help content (Option B)** — fetch content from website/GitHub at runtime to update without a release. Deferred; revisit if content churn justifies it.
- **`agenthub help` content CLI command** — a terminal echo of the same Help content beyond the framework's native `--help`. Out of scope for v1 (CLI native help satisfies parity).
- **Web-share-viewer Help surface** — exposing Help to browser viewers of a shared session. Out of scope.
- **FAQ auto-sourced from closed GitHub issues** — generating/curating FAQ entries from issue history. Not in scope; content stays maintainer-curated.

</deferred>

---

*Phase: 147-in-app-help-page*
*Context gathered: 2026-06-22*
