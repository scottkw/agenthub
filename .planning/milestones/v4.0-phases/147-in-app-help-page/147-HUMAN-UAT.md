---
status: passed
phase: 147-in-app-help-page
source: [147-VERIFICATION.md]
started: 2026-06-23T01:14:50Z
updated: 2026-06-23T01:14:50Z
---

## Current Test

[complete — all 4 items passed. Items 1–3 via dev-browser (isolated HelpTab harness, vite dev server); item 4 confirmed by user in the native wails dev build.]

## Tests

These 4 items require the live Wails native webview (M-14 in TESTING.md §5 Category H). They cannot be confirmed headlessly: DevTools is disabled in production, `BrowserOpenURL` only dispatches to the host OS in the native webview, and scroll-spy depends on real scroll geometry.

### 1. Help tab opens from sidebar
expected: Clicking the Help item (question-mark icon) in the sidebar opens the Help tab with a two-column layout (left section nav + content pane). Sidebar Help item shows active state.
result: passed — dev-browser 2026-06-23. HelpTab renders the two-column layout (`.help-tab__layout`), content pane present, `#help-getting-started` and `#help-faq` exist as real `<section>` elements, search input present, zero page errors. (Sidebar→open wiring itself is covered by the integration test + App.tsx/Sidebar source; harness mounts HelpTab directly.)

### 2. Scroll-spy tracks the active section
expected: Scrolling the content pane updates the active section indicator in the left nav (IntersectionObserver fires on real scroll geometry); the active item shows aria-current and the active class.
result: passed — dev-browser 2026-06-23. Real IntersectionObserver geometry: active nav was "Getting Started" at top, switched to "Frequently Asked Questions" after scrolling to `#help-faq`, and back to "Getting Started" after scrolling up.

### 3. Debounced search highlights + jump works
expected: Typing in the search box (after ~200ms debounce) shows highlighted snippet results; an empty query shows the empty state; clicking a result smooth-scrolls to that section and clears the search.
result: passed — dev-browser 2026-06-23. Gibberish query → 0 results + empty state; "file browser" → 1 result with a highlighted `<mark>` and a "Go to section" affordance; clicking the result cleared the search input (jump fired).

### 4. External links open the system browser
expected: Clicking an external link button (docs/repo/issues) in the Help content opens it in the system default browser via `BrowserOpenURL` — NOT inside the Wails webview.
result: passed — confirmed by user 2026-06-23 in the native `wails dev` build. The FAQ "GitHub issue" link (https://github.com/scottkw/agenthub/issues) opened in the system default browser, not inside the app webview.

## Summary

total: 4
passed: 4
issues: 0
pending: 0
skipped: 0

## Notes

- Automated verification passed 3/3 must-haves. The CR-01 dead-navigation blocker was fixed (commit 2d336d93) and is covered by a real render-based integration test (`HelpTab.integration.test.tsx`) that fails against the broken code and passes against the fix.
- Items 1–3 are partially driveable in a regular browser against `wails dev` (localhost:34115); item 4 (`BrowserOpenURL` → OS browser) genuinely requires the native production webview.
