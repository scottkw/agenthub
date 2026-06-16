---
status: partial
phase: 131-hub-foundation-static-session-cards
source: [131-VERIFICATION.md]
started: 2026-06-16
updated: 2026-06-16
---

## Current Test

[awaiting human testing — requires a live built app with real sessions]

## Tests

### 1. Hub + Sessions coexistence
expected: Clicking "Hub" in the sidebar opens the Hub surface as a coexisting top-level tab; the existing Sessions/DaemonManager panel remains reachable and unchanged (HUB-01, HUB-02).
result: [pending]

### 2. Light/dark theme rendering
expected: Hub renders correctly in both light and dark themes; theme tokens apply to the rendered DOM (HUB-04). NOTE: user is colorblind — verify hex values in the Inspector against the UI-SPEC, not by eye. Source-level hex constants already verified.
result: [pending]

### 3. Responsive grid reflow
expected: The card grid reflows cleanly (no overlap/clipping) across narrow and wide viewport widths (GRID-01).
result: [pending]

### 4. Stopped dimming vs error-exit
expected: Stopped/exit-0 cards render visually dimmed with exit code shown; error-exit (non-zero) cards are NOT dimmed and show the exit code (CARD-08).
result: [pending]

### 5. Running spin animation + reduced-motion
expected: Running-card status icon spins; with prefers-reduced-motion the animation is suppressed (no spin) while the icon+label still convey status.
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps
