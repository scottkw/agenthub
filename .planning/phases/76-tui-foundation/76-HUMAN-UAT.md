---
status: partial
phase: 76-tui-foundation
source: [76-VERIFICATION.md]
started: 2026-04-15T14:05:00Z
updated: 2026-04-15T14:05:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Alt-screen enter/exit cleanliness
expected: Alt-screen enters cleanly, prior shell scrollback returns intact after quit
command: `agenthub tui` → navigate with j/k → press `q`
result: [pending]

### 2. Adaptive color rendering
expected: Adaptive colors render with correct contrast — selected row highlights, accent glyphs visible on both backgrounds
command: Run in a dark-background terminal AND in a light-background terminal
result: [pending]

### 3. Unicode glyph rendering
expected: Unicode glyphs U+25CF, U+25CB render correctly without font substitution boxes
command: `agenthub tui` — visually inspect status glyphs (filled and hollow circles) and help close hint
result: [pending]

### 4. SIGWINCH resize reflow
expected: Layout reflows cleanly on every resize with no tearing or garbage characters
command: `agenthub tui` — resize window smaller and larger while open
result: [pending]

### 5. Help overlay centering across sizes
expected: Help overlay remains centered and fully bordered at all tested sizes
command: `agenthub tui` → press `?` — test at 61x11, 80x24, 120x40
result: [pending]

### 6. Sub-minimum size fallback
expected: Graceful "Terminal too small (need 60x10)" message appears; resize back restores full UI
command: `agenthub tui` → resize terminal to below 60x10
result: [pending]

### 7. Non-TTY piped fallback
expected: Prints "agenthub tui requires a terminal. Redirect to a TTY or use agenthub list instead" to stderr and exits 1
command: `agenthub tui | cat`
result: [pending]

## Summary

total: 7
passed: 0
issues: 0
pending: 7
skipped: 0
blocked: 0

## Gaps
