---
phase: 147-in-app-help-page
verified: 2026-06-22T20:15:00Z
status: human_needed
score: 3/3 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Open the live Wails app and click the Help button in the sidebar. Verify the Help tab opens, Markdown content renders (headings, paragraphs, inline code spans), and the left section nav shows 'Getting Started' and 'Frequently Asked Questions'."
    expected: "Help tab opens. Both sections render with formatted content. Left nav is visible."
    why_human: "Wails native webview is inaccessible to Playwright/headless automation (DevTools disabled in production). M-14 in TESTING.md."
  - test: "Scroll the Help content pane slowly from Getting Started down to the FAQ section. Watch the left nav active item."
    expected: "The active indicator (aria-current + --active class) switches from 'Getting Started' to 'Frequently Asked Questions' as the FAQ section scrolls into the upper portion of the pane."
    why_human: "IntersectionObserver scroll-spy behaviour requires a live scrolling environment; jsdom's scrollIntoView stub covers click, not live scroll physics."
  - test: "Type 'DevTools' in the search box and wait ~200ms."
    expected: "At least one result appears with '<mark>' highlighted span around 'DevTools'. A 'Go to Frequently Asked Questions' jump button is visible. Clicking it scrolls the FAQ section into view."
    why_human: "Debounce timing and real scroll in native webview cannot be verified headlessly."
  - test: "Click the GitHub Issues link in the 'Report a bug' FAQ answer or trigger an empty-search state and click the 'report an issue on GitHub' button."
    expected: "The system default browser opens to https://github.com/scottkw/agenthub/issues. The Wails webview does NOT navigate away."
    why_human: "BrowserOpenURL dispatch to the host OS browser is untestable in jsdom (no real browser shell)."
---

# Phase 147: In-App Help Page Verification Report

**Phase Goal:** An in-app Help page provides documentation, an FAQ, search, and external links, reachable from the app navigation.
**Verified:** 2026-06-22T20:15:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A Help page is reachable from the app navigation (sidebar/menu) | VERIFIED | `Sidebar.tsx:299-305` renders a `<button aria-label="Help">` with `QuestionMarkCircleIcon` and `activePanel === '__help__'` active-state logic; `App.tsx:794-802` implements `handleOpenHelp` (find-or-add idempotent, mirrors `handleOpenSettings`); `App.tsx:1391` passes `onOpenHelp={handleOpenHelp}` to `<Sidebar>`; Tab type union in `TabBar.tsx:12` includes `'help'`; full Sidebar test suite with Help describe block passes (117/117 test files green). |
| 2 | It includes documentation content, an FAQ, search over that content, and external links | VERIFIED | `getting-started.md` has 7 scannable sections; `faq.md` has all 6 seed questions including DevTools, network sharing, remote file browse, logs location, updating, and bug reporting. `HelpContent.tsx` renders Markdown via `react-markdown` + `rehype-sanitize`; `HelpSearch.tsx` has debounced search with `<mark>` highlight and empty-state; external links are `BrowserOpenURL` buttons (no raw `<a href>`). `HelpTab.tsx:155-159` renders each section individually inside `<section id={id} className="help-content__section">`, making `getElementById('help-getting-started')` and `getElementById('help-faq')` resolve. Integration test `HelpTab.integration.test.tsx` renders `<HelpTab>` and asserts both section elements exist in the DOM, nav click calls `scrollIntoView` on the correct section, and a search result jump resolves a non-null element — this is the CR-01 fix evidence. All 117 vitest files pass (1862 tests). |
| 3 | Cross-surface parity satisfied or explicit carve-out documented | VERIFIED | `147-CONTEXT.md` D-03 explicitly records: "GUI only. CLI native `--help` satisfies CLI parity. Web-share viewers do not need the Help page." Web-share carve-out is implemented: `App.tsx:1576` gates `HelpTab` render with `{mode !== 'web' && ...}` (same pattern as `SettingsTab`, which also uses Wails RPC not available in web-share context). The cross-surface parity reconciliation is explicit and present in the planning record. |

**Score:** 3/3 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/HelpTab.tsx` | Container: state, debounce, search index, section rendering | VERIFIED | 165 lines; renders per-section `<section id={id}>` wrappers; 200ms debounce via `useRef`; `useMemo([])` search index; handles jump-to-section. |
| `frontend/src/components/HelpContent.tsx` | Markdown renderer with BrowserOpenURL links | VERIFIED | 92 lines; `react-markdown` + `remark-gfm` + `rehype-sanitize`; custom `a` → BrowserOpenURL button with `isSafeExternalHref` guard; `nodeToText` for safe aria-labels. |
| `frontend/src/components/HelpSearch.tsx` | Search input, snippet, mark highlight, empty state | VERIFIED | 232 lines; `HighlightedSnippet` uses plain-string split to `<mark className="help-search__mark">`; `extractSnippet` with surrogate-pair guard; visible label + clear button; empty-state with GitHub Issues link. |
| `frontend/src/components/HelpSectionNav.tsx` | Sticky nav with IntersectionObserver scroll-spy | VERIFIED | 90 lines; `IntersectionObserver` scoped to `contentPaneRef` root; `aria-current="true"` and `.help-nav__link--active` on active button; click scrolls + calls `onSectionChange`. |
| `frontend/src/content/help/getting-started.md` | Getting Started documentation content | VERIFIED | 27 lines; 7 sections (session creation, shell sessions, switching, file browser, sharing, settings); paragraph-split-friendly (blank-line separated). |
| `frontend/src/content/help/faq.md` | FAQ with 6 seed questions | VERIFIED | 26 lines; all 6 seed questions present; DevTools answer references `wails dev` and web-share; bug report answer links GitHub Issues; no keyboard-shortcuts.md created. |
| `frontend/src/components/__tests__/HelpTab.integration.test.tsx` | Real render-based integration test (CR-01 fix) | VERIFIED | 189 lines; renders `<HelpTab>` via `createRoot`; asserts `getElementById('help-getting-started')` and `getElementById('help-faq')` both non-null and are `<section>` elements; asserts nav click calls `scrollIntoView` on the correct section; asserts search result jump fires `scrollIntoView` (non-null target). PASSES in current suite. |
| `TESTING.md` | HELP-01 traceability (5 rows) + M-14 manual item | VERIFIED | §2 vitest count = 117 (was 112 + 4 + 1 integration); Total = 473; §4 has 5 HELP-01 rows (4 original + integration test); §5 Category H has M-14 with scroll-spy, search, external-link UAT steps. `bash tests/check-traceability-paths.sh` exits 0. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `Sidebar.tsx` `onOpenHelp` prop | `App.tsx` `handleOpenHelp` | `App.tsx:1391 onOpenHelp={handleOpenHelp}` | WIRED | Prop threaded from SidebarProps interface through Sidebar render call. |
| `handleOpenHelp` | `HELP_TAB` / tab state | `setTabs` + `setActiveId` in `App.tsx:800-801` | WIRED | Find-or-add pattern mirrors Settings. |
| `HELP_TAB` tab type | `HelpTab` render | `App.tsx:1576-1579` display-toggle block `activeId === HELP_TAB.id` | WIRED | `{mode !== 'web' && <div style={{ display: activeId === HELP_TAB.id ? 'flex' : 'none' ... }}><HelpTab /></div>}` |
| `HelpTab` sections | `getElementById` resolution | `HelpTab.tsx:155-159` renders `<section id={id}>` per `SECTION_META` entry | WIRED | CR-01 fix confirmed — section anchor IDs are rendered; integration test asserts both elements exist in DOM. |
| `HelpContent` links | `BrowserOpenURL` | `HelpContent.tsx:74` `onClick={() => { if (isSafeExternalHref(href)) BrowserOpenURL(href) }}` | WIRED | All `<a>` components intercepted; `https?:` scheme guard applied before dispatch. |
| `HelpSectionNav` scroll-spy | section DOM elements | `HelpSectionNav.tsx:54-55` `document.getElementById(section.id)` after sections are rendered | WIRED | Since CR-01 fix, `getElementById` resolves real elements and `observer.observe(el)` fires. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `HelpTab` | `gettingStartedMd`, `faqMd` | `?raw` Vite static import of bundled `.md` files | Yes — module constants, always non-empty | FLOWING |
| `HelpTab` | `searchIndex` | `useMemo([])` over `SECTION_META` splitting real markdown content | Yes — paragraph split of real content; integration test confirms `'session'` and `'Tailscale'` queries produce results | FLOWING |
| `HelpContent` | `markdown` prop | Passed from `SECTION_META[n].markdown` in `HelpTab.tsx:157` | Yes — actual markdown string from bundled files | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Integration test: section anchors exist in DOM | `cd frontend && pnpm test -- --reporter=verbose --run 2>&1 \| grep "section anchors"` | "HelpTab integration: section anchors render with expected ids (Phase 147)" — 4 tests PASS | PASS |
| Integration test: nav click scrolls correct section | `pnpm test` suite | "clicking the FAQ nav button calls scrollIntoView on the #help-faq section" — PASS | PASS |
| Integration test: search jump resolves non-null section | `pnpm test` suite | "typing a query then clicking a result scrolls a non-null section into view" — PASS | PASS |
| Full vitest suite | `cd frontend && pnpm test --run` | 117 test files, 1862 tests, 0 failures | PASS |
| Traceability path check | `bash tests/check-traceability-paths.sh` | Exit 0 — "OK: all traceability paths exist" | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| HELP-01 | 147-01, 147-02, 147-03, 147-04 | In-app Help page: docs, FAQ, search, external links (#69) | SATISFIED | All four components implemented, wired, and tested. Integration test confirms CR-01 fix. 5 traceability rows in TESTING.md. M-14 manual item added. REQUIREMENTS.md checkbox `[x]` for HELP-01 at line 83. |

### Anti-Patterns Found

No blockers found. The code-review BLOCKER CR-01 (section anchor IDs never rendered) was identified in `147-REVIEW.md` and fixed in commit `2d336d93` before this verification. The integration test (`HelpTab.integration.test.tsx`, commit `02ca6aa1`) was added specifically to prevent this regression.

Warnings from the code review (WR-01 through WR-05) remain as advisory items:
- WR-01 is addressed: HelpTab.integration.test.tsx now mounts the component.
- WR-02 (surrogate-pair edge case): addressed in HelpSearch.tsx with `clampToCodePoint`.
- WR-03 (aria-label on complex children): addressed in HelpContent.tsx with `nodeToText`.
- WR-04 (scheme validation): addressed with `isSafeExternalHref` / `SAFE_LINK_SCHEME` regex.
- WR-05 (result key): addressed — key is now `${r.sectionId}-${r.text.slice(0, 48)}`.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | No TBD/FIXME/XXX/stubs found in Help components | — | — |

### Human Verification Required

These items require a live Wails native-webview test because the production app has DevTools disabled and IntersectionObserver scroll physics, BrowserOpenURL dispatch, and the scroll-spy timing cannot be exercised headlessly. These are recorded as M-14 in TESTING.md §5 Category H.

#### 1. Help tab opens from sidebar

**Test:** Run the production (or `wails dev`) build. Click the `?` (Help) button in the sidebar bottom-nav.
**Expected:** A "Help" tab opens (or focuses if already open). The tab shows a two-column layout: sticky search bar at top, left nav with "Getting Started" and "Frequently Asked Questions", and rendered Markdown content on the right.
**Why human:** Wails native webview is inaccessible to Playwright or headless automation in the production build.

#### 2. Scroll-spy tracks active section

**Test:** With the Help tab open, scroll the right-hand content pane slowly downward past the "Frequently Asked Questions" heading.
**Expected:** The left nav active indicator switches from "Getting Started" to "Frequently Asked Questions" as the FAQ section enters the upper portion of the pane. The button gains `aria-current="true"` and the `--active` modifier class.
**Why human:** IntersectionObserver fires on real scroll geometry; jsdom's polyfill stubs the observer without triggering scroll-based callbacks.

#### 3. Debounced search highlights and jump works

**Test:** Type "DevTools" in the search input. Wait approximately 200ms.
**Expected:** One or more result snippets appear with the matched text wrapped in a highlighted `<mark>` element. A "Go to Frequently Asked Questions →" button is visible. Clicking it scrolls the FAQ section into view and clears the search query.
**Why human:** Real debounce timing and smooth scroll in the Wails webview require a live environment.

#### 4. External links open system browser

**Test:** Click the GitHub Issues link in the "How do I report a bug?" FAQ answer.
**Expected:** The system default browser (not the in-app webview) opens to `https://github.com/scottkw/agenthub/issues`. The Wails window remains at the Help tab and does not navigate away.
**Why human:** `BrowserOpenURL` dispatches to the host OS; jsdom has no real browser shell to open.

### Gaps Summary

No code gaps remain. All three observable truths are VERIFIED by codebase evidence and the passing test suite. The CR-01 BLOCKER identified in `147-REVIEW.md` was resolved before this verification: `HelpTab.tsx` now renders each section individually inside `<section id={id} className="help-content__section">` (lines 155-159), which makes `document.getElementById('help-getting-started')` and `document.getElementById('help-faq')` resolve, enabling the nav click, scroll-spy, and search jump to work correctly. The integration test confirms the fix is load-bearing (it would fail against the pre-fix concatenated render). Status is `human_needed` because the Wails native-webview UAT (M-14) cannot be automated headlessly — the automated evidence fully supports a pass pending those four live checks.

---

_Verified: 2026-06-22T20:15:00Z_
_Verifier: Claude (gsd-verifier)_
