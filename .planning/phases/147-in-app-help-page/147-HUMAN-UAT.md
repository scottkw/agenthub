---
status: partial
phase: 147-in-app-help-page
source: [147-VERIFICATION.md]
started: 2026-06-23T01:14:50Z
updated: 2026-06-23T01:14:50Z
---

## Current Test

[awaiting human testing — run in the production Wails native webview]

## Tests

These 4 items require the live Wails native webview (M-14 in TESTING.md §5 Category H). They cannot be confirmed headlessly: DevTools is disabled in production, `BrowserOpenURL` only dispatches to the host OS in the native webview, and scroll-spy depends on real scroll geometry.

### 1. Help tab opens from sidebar
expected: Clicking the Help item (question-mark icon) in the sidebar opens the Help tab with a two-column layout (left section nav + content pane). Sidebar Help item shows active state.
result: [pending]

### 2. Scroll-spy tracks the active section
expected: Scrolling the content pane updates the active section indicator in the left nav (IntersectionObserver fires on real scroll geometry); the active item shows aria-current and the active class.
result: [pending]

### 3. Debounced search highlights + jump works
expected: Typing in the search box (after ~200ms debounce) shows highlighted snippet results; an empty query shows the empty state; clicking a result smooth-scrolls to that section and clears the search.
result: [pending]

### 4. External links open the system browser
expected: Clicking an external link button (docs/repo/issues) in the Help content opens it in the system default browser via `BrowserOpenURL` — NOT inside the Wails webview.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0

## Notes

- Automated verification passed 3/3 must-haves. The CR-01 dead-navigation blocker was fixed (commit 2d336d93) and is covered by a real render-based integration test (`HelpTab.integration.test.tsx`) that fails against the broken code and passes against the fix.
- Items 1–3 are partially driveable in a regular browser against `wails dev` (localhost:34115); item 4 (`BrowserOpenURL` → OS browser) genuinely requires the native production webview.
