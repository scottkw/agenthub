---
status: partial
phase: 141-redesign-implementation
source: [141-VERIFICATION.md]
started: 2026-06-21T00:00:00Z
updated: 2026-06-21T00:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Light theme visual rendering
expected: Toggle to light mode (`[data-ui-theme=light]`) and verify all restyled surfaces (Welcome, Hub, terminal/session, File Browser, Editor, Settings, Share Modal) repaint correctly using the `--hub-*` token overrides — no unstyled/black-on-black or default-colored panels. Colorblind note: color *correctness* was verified at hex-constant level in code (RDS-04); this item is a structural repaint check, not a color-discrimination check.
result: [pending]

### 2. GroupSidebar visual layout after CARRY-01 ARIA refactor
expected: In the Hub, the group sidebar items render WITHOUT browser-default button chrome (no native border/background — WR-01 fix); tabbing with the keyboard shows the 2px `--hub-accent` focus ring on each group button (WR-02 fix); when the sidebar is collapsed, the "Groups" heading is visually hidden but the list remains ARIA-labelled (WR-03 fix, `.sr-only`).
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
