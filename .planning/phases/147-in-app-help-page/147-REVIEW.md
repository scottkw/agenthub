---
phase: 147-in-app-help-page
reviewed: 2026-06-22T00:00:00Z
depth: standard
files_reviewed: 18
files_reviewed_list:
  - frontend/src/App.tsx
  - frontend/src/components/HelpContent.tsx
  - frontend/src/components/HelpSearch.tsx
  - frontend/src/components/HelpSectionNav.tsx
  - frontend/src/components/HelpTab.tsx
  - frontend/src/components/Sidebar.tsx
  - frontend/src/components/TabBar.tsx
  - frontend/src/components/__tests__/HelpContent.test.tsx
  - frontend/src/components/__tests__/HelpSearch.test.tsx
  - frontend/src/components/__tests__/HelpSectionNav.test.tsx
  - frontend/src/components/__tests__/HelpTab.test.tsx
  - frontend/src/components/__tests__/Sidebar.test.tsx
  - frontend/src/content/help/faq.md
  - frontend/src/content/help/getting-started.md
  - frontend/src/style.css
  - frontend/src/test-setup.ts
  - frontend/package.json
findings:
  critical: 1
  warning: 5
  info: 4
  total: 10
status: issues_found
---

# Phase 147: Code Review Report

**Reviewed:** 2026-06-22
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

Reviewed the in-app Help page feature: Markdown rendering (`HelpContent`), search
(`HelpSearch`/`HelpTab`), scroll-spy section nav (`HelpSectionNav`), sidebar/tab
wiring (`Sidebar`, `TabBar`, `App.tsx`), seed Markdown content, CSS, and the test
suite.

**Security posture is good.** Markdown is rendered through `react-markdown` +
`rehype-sanitize` with an extended-but-safe schema, no `dangerouslySetInnerHTML`,
and all anchors are intercepted into `BrowserOpenURL` buttons so nothing navigates
the Wails webview. The highlight path uses plain-string `<mark>` splitting, not raw
HTML injection. I found no XSS, injection, secret, or unsafe-deserialization issue.

**However, the feature's core navigation does not work.** The section nav, the
IntersectionObserver scroll-spy, and the search "Go to section" jump all depend on
DOM elements with `id="help-getting-started"` and `id="help-faq"`. Nothing in the
render path ever produces those IDs — there is no `rehype-slug` plugin and no custom
heading renderer that injects them. Every `document.getElementById(...)` call returns
`null`, so clicking a nav item, scrolling, or jumping from a search result is a
silent no-op. This is a BLOCKER. It is invisible to the test suite because
`HelpTab.test.tsx` only string-matches source files and never renders the component
(`createRoot`/`render` count for `HelpTab` is 0), so the green suite is false comfort.

## Critical Issues

### CR-01: Section anchors never exist — nav, scroll-spy, and jump-to-section are all dead

**File:** `frontend/src/components/HelpSectionNav.tsx:54,75`, `frontend/src/components/HelpTab.tsx:60-62,124`, `frontend/src/components/HelpContent.tsx:28-60`

**Issue:** Three independent code paths look up sections by element id:
- `HelpSectionNav` scroll-spy: `document.getElementById(section.id)` where `section.id` is `help-getting-started` / `help-faq` (lines 53-55).
- `HelpSectionNav` click handler: `document.getElementById(id)` then `scrollIntoView` (lines 75-77).
- `HelpTab.handleJumpToSection`: `document.getElementById(sectionId)` (line 124).

But `HelpContent` renders the concatenated Markdown with `react-markdown` and only
overrides `code` and `a`. The headings `## Getting Started` / `## Frequently Asked
Questions` render as bare `<h2>Getting Started</h2>` with **no `id` attribute**.
There is no `rehype-slug` in `rehypePlugins` (confirmed: `rehype-slug` is absent from
`package.json` and the entire `frontend/` tree), and no custom `h2` renderer.
Therefore:
- `getElementById('help-getting-started')` and `getElementById('help-faq')` always return `null`.
- The IntersectionObserver observes nothing (the `if (el) observer.observe(el)` guard is never true), so `onSectionChange` never fires from scrolling — the active-section indicator is frozen on the initial `help-getting-started`.
- Clicking a nav button or a search result's "Go to {section}" button calls `scrollIntoView` on `null` (guarded, so it just does nothing) — the page does not scroll.

Even if a slug plugin were added, the default GitHub slug for `## Getting Started`
is `getting-started`, **not** `help-getting-started`, so the IDs still would not
match without an explicit prefix. The leftover `.help-content__section` selector in
`style.css:1151` (with `scroll-margin-top`) is further evidence that an id/class-on-
section design was intended but never wired.

This is the central function of the feature (a two-section help page with a sticky
nav and scroll-spy) and it is entirely non-functional. No test covers it because
`HelpTab.test.tsx` is source-gate-only and never mounts the component.

**Fix:** Inject stable, prefixed IDs onto the section headings and observe those.
Add `rehype-slug` and a custom heading renderer that maps the first `<h2>` of each
section to the expected id, or — simpler and deterministic — render each section
separately and wrap it in an anchor element:

```tsx
// HelpTab.tsx — render sections individually with explicit anchor ids
{SECTION_META.map(({ id, markdown }) => (
  <section id={id} key={id} className="help-content__section">
    <HelpContent markdown={markdown} />
  </section>
))}
```

This guarantees `getElementById('help-getting-started')` / `'help-faq'` resolve, and
matches the already-present `.help-content__section { scroll-margin-top: 80px }` rule.
Then add a render-level test that mounts `HelpTab`, clicks a nav button, and asserts
`scrollIntoView` was called on the section element (or that the anchor exists), so
this class of regression cannot pass the suite again.

## Warnings

### WR-01: HelpTab test is source-gate-only — feature is shipped untested

**File:** `frontend/src/components/__tests__/HelpTab.test.tsx:19-54`

**Issue:** `HelpTab.test.tsx` only `readFileSync`s `App.tsx` and `style.css` and
asserts substring presence (`HELP_TAB`, `handleOpenHelp`, `--hub-search-highlight-bg`).
It never imports or renders `HelpTab`. The integration between `HelpContent`,
`HelpSectionNav`, `HelpSearch`, the search index, the debounce, and the section
anchors is therefore completely unverified — which is exactly why CR-01 shipped
green. Per the project's standing rule (`feedback_post_merge_gate_run_tsc`, and the
"vitest tolerates what the app rejects" memory), source-gate string matching is not
proof the feature works.

**Fix:** Add a render-based test that mounts `<HelpTab />`, asserts the section
anchors exist in the DOM, drives a search query through the 200ms debounce
(`vi.useFakeTimers`), and verifies a result click triggers `scrollIntoView` /
section navigation.

### WR-02: `extractSnippet` can split a multibyte/surrogate character and miscount snippet length

**File:** `frontend/src/components/HelpSearch.tsx:33-47`

**Issue:** `extractSnippet` uses `text.slice(start, end)` with character offsets
derived from `indexOf` and fixed `HALF = 60`. For the maintainer-authored ASCII
content this is fine today, but `slice` operates on UTF-16 code units, so a snippet
boundary that lands inside a surrogate pair (emoji, some CJK) produces a lone
surrogate (`�`) at the cut point. More importantly, the empty-query/no-match branch
`text.length > 120 ? text.slice(0, 120) + '…'` and the windowing math are duplicated
logic with magic numbers (`120`, `60`) that should be a single named constant. Low
real-world risk given current content, but it is a correctness foot-gun if help
content ever gains non-ASCII characters.

**Fix:** Hoist `SNIPPET_RADIUS`/`SNIPPET_MAX` constants, and either accept the ASCII
constraint explicitly in a comment or guard against splitting surrogate pairs (e.g.
nudge `start`/`end` off a low surrogate).

### WR-03: `aria-label` interpolates a React node, not text, for link labels

**File:** `frontend/src/components/HelpContent.tsx:47`

**Issue:** `aria-label={`${children} (opens in browser)`}` template-stringifies
`children`, which is a React node (array). When the link text is a plain string this
works, but if a Markdown link ever contains emphasis or inline code
(`[**bold**](url)`), `children` becomes an object/array and stringifies to
`[object Object]` (or `[object Object],...`), producing a broken screen-reader label.
The seed content happens to use plain-text link labels, so it is latent, but the
component is generic and the bug will surface the first time a styled link is added.

**Fix:** Derive a string label safely, e.g. compute the text content from `children`
when it is a string and fall back to the bare href:

```tsx
const labelText = typeof children === 'string' ? children : href ?? 'link'
// aria-label={`${labelText} (opens in browser)`}
```

### WR-04: External-link button has no guard for missing/relative href and no scheme validation

**File:** `frontend/src/components/HelpContent.tsx:42-55`

**Issue:** `onClick={() => href && BrowserOpenURL(href)}` opens whatever `href` the
Markdown supplied in the system browser. For the trusted bundled content this is safe,
but there is no allow-listing of schemes. If help Markdown ever interpolates a
non-`https` value (e.g. `file:`, `javascript:`, or a `mailto:` the user did not
expect), it is handed straight to `BrowserOpenURL`. `rehype-sanitize` will strip a
`javascript:` href from an `<a>` before your renderer sees it (defaultSchema's
protocol allow-list covers `href`), so the practical XSS risk is low — but the
component does no validation of its own and the comment promises "open in the system
browser" without bounding the scheme. Defense-in-depth is cheap here.

**Fix:** Validate the scheme before calling `BrowserOpenURL`:

```tsx
onClick={() => {
  if (href && /^https?:\/\//i.test(href)) BrowserOpenURL(href)
}}
```

### WR-05: Search-result `<li>` key uses array index, defeating React reconciliation

**File:** `frontend/src/components/HelpSearch.tsx:131`

**Issue:** `key={`${r.sectionId}-${i}`}` mixes a stable id with the array index `i`.
Because results are recomputed on every keystroke (different filtered subset), the
same paragraph can land at a different index across renders, so the key changes for
the same logical item — defeating reconciliation and remounting `<li>` nodes
unnecessarily (losing focus/selection state on the keyboard-navigable `role="option"`
items, which is a real a11y concern since they are `tabIndex={0}`). Two paragraphs
from the same section would also collide on `sectionId` without the index, which is
presumably why `i` was added — but the right fix is a key derived from content.

**Fix:** Key on stable content, e.g. a hash/prefix of `r.text` plus `sectionId`:
`key={`${r.sectionId}-${r.text.slice(0, 32)}`}`, or add a stable per-entry id when
building `searchIndex` in `HelpTab`.

## Info

### IN-01: Duplicate `SearchEntry` interface declared in two files

**File:** `frontend/src/components/HelpTab.tsx:19-23` and `frontend/src/components/HelpSearch.tsx:11-15`

**Issue:** `SearchEntry` is defined identically in both `HelpTab` and `HelpSearch`.
They are structurally compatible so it compiles, but the duplication will drift.
`HelpTab`'s comment even claims the type is "also exported so tests + other components
can import them," yet `HelpSearch` redeclares its own instead of importing it.

**Fix:** Export `SearchEntry` from one module (or a shared `types.ts`) and import it
in the other.

### IN-02: Markdown search-strip regexes are simplistic and can mis-strip content

**File:** `frontend/src/components/HelpTab.tsx:34-54`

**Issue:** `stripMd` uses greedy-ish single-pass regexes (e.g. `\*{1,3}([^*]+)\*{1,3}`
and `_{1,3}([^_]+)_{1,3}`). Underscores inside words (`snake_case`, file paths like
`Application_Support`) and stray asterisks will be partially consumed or mangled in
the search index. The content is maintainer-authored so the index "works," but the
plain-text snippets shown to users may contain odd artifacts. Documented as
"lightweight / good enough" — acceptable, but flagged so it is a conscious choice.

**Fix:** If snippet fidelity matters, derive search text from the rendered text
content (e.g. strip via a remark `toString`) rather than hand-rolled regexes.

### IN-03: Concatenated-markdown render duplicates the per-section parse work

**File:** `frontend/src/components/HelpTab.tsx:71,99-113`

**Issue:** `allMarkdown` concatenates both files for a single `HelpContent` render
(line 71), while `searchIndex` separately iterates `SECTION_META` and re-parses the
same source per section (lines 99-113). This is fine functionally, but the single
concatenated render is the very thing that makes per-section anchor IDs hard (see
CR-01). Rendering per-section (the CR-01 fix) also removes this redundancy.

**Fix:** Render each section from `SECTION_META` individually (see CR-01), which both
fixes the anchors and unifies the two iteration paths.

### IN-04: Leftover `.help-content__section` CSS selector with no matching element

**File:** `frontend/src/style.css:1151`

**Issue:** `.help-content__section` appears in the `scroll-margin-top` rule but no
component renders an element with that class. It is dead CSS today and corroborates
that the section-wrapper design (which would have provided the missing anchor IDs in
CR-01) was specified but not implemented.

**Fix:** When implementing the CR-01 fix, wrap each section in
`<section id=... className="help-content__section">` so this selector becomes live;
otherwise remove the dead rule.

---

_Reviewed: 2026-06-22_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
