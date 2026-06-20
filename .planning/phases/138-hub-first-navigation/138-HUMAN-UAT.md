---
status: partial
phase: 138-hub-first-navigation
source: [138-VERIFICATION.md]
started: 2026-06-20T23:25:00Z
updated: 2026-06-20T23:25:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. 3-item sidebar visible in live app
expected: Sidebar shows exactly Home, Hub, Settings — no Sessions, Remote, or New Session entry. Responsive collapse behavior intact.
result: [pending]

### 2. HubFilterBar is the sole New Session entry point
expected: No other button or affordance creates a new session; clicking HubFilterBar's button opens the new-session modal and the full modal → daemon creation path works.
result: [pending]

### 3. Remote card shows "Open in browser" and "Browse files"; Open forwards the real peer URL
expected: Overflow menu on a connected remote card shows both items; clicking "Open in browser" opens the real peer URL (not an empty page) in the system browser via BrowserOpenURL. (CR-01 fix — needs a live reachable peer.)
result: [pending]

### 4. Kill two-step confirm on a live local session
expected: Overflow menu on a live local session shows "Kill session"; first click shows "Confirm kill" / "This will stop the session"; second click terminates the session via the daemon.
result: [pending]

### 5. Remote card does NOT show Kill (CR-02) and does NOT show re-attach "Open" (WR-01)
expected: A remote card's overflow menu shows only "Open in browser" and "Browse files" — no "Kill session"; and no row-5 re-attach "Open" button renders on remote cards. (isLocal guard confirmed at source.)
result: [pending]

### 6. Colorblind-safe card indicators render with icon + text
expected: Local cards show ComputerDesktopIcon + "Local"; remote connected cards show LinkIcon + "Connected"; remote available cards show GlobeAltIcon + "Available". Every indicator pairs an icon shape with a text label — no color-only signals. (Hex/custom-property already verified at source; this confirms icons render alongside text.)
result: [pending]

### 7. Attention pulse, mini-preview, and grid reflow preserved (CARD-04)
expected: Hub cards still animate the attention pulse, show mini-preview text, and reflow responsively at 240–360px grid density after the new card affordances were added.
result: [pending]

## Summary

total: 7
passed: 0
issues: 0
pending: 7
skipped: 0
blocked: 0

## Gaps
