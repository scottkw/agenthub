---
status: deferred
deferred_to: v3.3
deferred_on: 2026-05-12
deferred_reason: "Aggregates 95-DESKTOP-UAT.md + 95-WEB-UAT.md results; both source runbooks require raw shell PTY to exercise URL printing + hover/click. AgentHub v3.2 ships agent sessions only; shell session type deferred to v3.3+ (see v3.2-RELEASE-BLOCKERS.md)."
phase: 95-web-links-addon-security-hardening
source: [95-VERIFICATION.md, 95-DESKTOP-UAT.md, 95-WEB-UAT.md]
started: 2026-05-06T18:55:00Z
updated: 2026-05-12T22:55:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Cross-OS modifier-click + hover tooltip (LNK-02 / SC-2)
expected: Cmd-click on macOS / Ctrl-click on Linux/Windows opens URL via Wails BrowserOpenURL; single-click does nothing; hover tooltip shows resolved href.
runbook: 95-DESKTOP-UAT.md §2
result: [pending]

### 2. Cyrillic spoof popover (LNK-03 / SC-3)
expected: `echo https://gооgle.com` (Cyrillic U+043E) → Cmd-click triggers popover with idn copy + full resolved URL; Continue → BrowserOpenURL; Cancel dismisses.
runbook: 95-DESKTOP-UAT.md §3
result: [pending]

### 3. Web window.opener defense (LNK-04 / SC-4)
expected: Tailscale-served terminal — Cmd-click https URL opens new tab; DevTools `window.opener === null`; original tab does not navigate.
runbook: 95-WEB-UAT.md §5
result: [pending]

### 4. Live toggle without session restart (LNK-06 / SC-5)
expected: With session running + URL visible — disable web-links in Settings → links lose underline; re-enable → links return underlined; no session restart needed.
runbook: 95-DESKTOP-UAT.md §6 + 95-WEB-UAT.md §7
result: [pending]

### 5. iPad Safari Tailscale walkthrough (Phase 99 release gate co-verification)
expected: iPad Safari + paired keyboard — LNK-01..05 chain all fire; popover renders; window.open spawns new tab; no console errors.
runbook: 95-WEB-UAT.md §9
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps
