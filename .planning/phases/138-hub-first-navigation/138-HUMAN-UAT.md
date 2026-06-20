---
status: partial
phase: 138-hub-first-navigation
source: [138-VERIFICATION.md]
started: 2026-06-20T23:25:00Z
updated: 2026-06-20T23:25:00Z
---

## Current Test

[awaiting remote-peer UATs — see pending items 3/4/5/6-remote]

> Live UAT run 2026-06-20 via Playwright + system Chrome against the `wails dev`
> bridge (http://localhost:34115). A throwaway "Shell/bin/zsh" local session
> ("shell 1") was created to render a card; it is LEFT RUNNING so the user can
> exercise the deferred live items (#4 Kill two-step) on it. Screenshots:
> /tmp/uat-138-hub.png (empty Hub), /tmp/uat-138-card.png (local card + menu).

## Tests

### 1. 3-item sidebar visible in live app
expected: Sidebar shows exactly Home, Hub, Settings — no Sessions, Remote, or New Session entry. Responsive collapse behavior intact.
result: PASS — live DOM: button.sidebar__item = ["Home","Hub","Settings"] (count 3); no aria-label "Sessions"/"Remote"/"New Session". Confirmed in screenshot.

### 2. HubFilterBar is the sole New Session entry point
expected: No other button or affordance creates a new session; clicking HubFilterBar's button opens the new-session modal and the full modal → daemon creation path works.
result: PASS — HubFilterBar `.hub-filter__new-session` present; `.hub__header` removed (0 matches). New-session modal → daemon creation path worked (created "shell 1"). NOTE: a contextual empty-state `.hub__empty-cta` "New session" CTA renders ONLY while zero sessions exist (not the removed persistent chrome duplicate); disappears once a session exists.

### 3. Remote card shows "Open in browser" and "Browse files"; Open forwards the real peer URL
expected: Overflow menu on a connected remote card shows both items; clicking "Open in browser" opens the real peer URL (not an empty page) in the system browser via BrowserOpenURL. (CR-01 fix — needs a live reachable peer.)
result: [pending] — requires a live reachable remote peer (not available in this environment). CR-01 fix verified at source + unit test (adapter carries url).

### 4. Kill two-step confirm on a live local session
expected: Overflow menu on a live local session shows "Kill session"; first click shows "Confirm kill" / "This will stop the session"; second click terminates the session via the daemon.
result: [pending] — "Kill session" item IS present on the live local card (confirmed). Two-step confirm flow + actual termination left for the user to click on the "shell 1" session. Two-step UI logic is unit-tested.

### 5. Remote card does NOT show Kill (CR-02) and does NOT show re-attach "Open" (WR-01)
expected: A remote card's overflow menu shows only "Open in browser" and "Browse files" — no "Kill session"; and no row-5 re-attach "Open" button renders on remote cards. (isLocal guard confirmed at source.)
result: PARTIAL — local-card inverse confirmed LIVE: the local "shell 1" card shows "Kill session" + re-attach "Open" and does NOT show "Open in browser"/"Browse files". The remote-card direction needs a live peer; isLocal guard verified at source + unit test.

### 6. Colorblind-safe card indicators render with icon + text
expected: Local cards show ComputerDesktopIcon + "Local"; remote connected cards show LinkIcon + "Connected"; remote available cards show GlobeAltIcon + "Available". Every indicator pairs an icon shape with a text label — no color-only signals. (Hex/custom-property already verified at source; this confirms icons render alongside text.)
result: PARTIAL — LOCAL origin confirmed LIVE: `.hub-card__origin` = "Local" WITH an svg icon; local card has NO `.hub-card__conn` chip (correct). Remote Connected/Available chip (LinkIcon/GlobeAltIcon + text) needs a live peer; verified at source + unit test.

### 7. Attention pulse, mini-preview, and grid reflow preserved (CARD-04)
expected: Hub cards still animate the attention pulse, show mini-preview text, and reflow responsively at 240–360px grid density after the new card affordances were added.
result: PASS (grid + mini-preview) — `.hub__card-row` is display:grid, columns 272px (within 240–360px band); mini-preview terminal text renders on the card (screenshot). Attention pulse not forced (idle session); `.hub-card--attention` CSS is unit-verified (style.hub.test.ts).

## Summary

total: 7
passed: 3
partial: 2
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps

None. No defects found in live UAT. Items 3, 4, and the remote halves of 5/6
remain pending only because they require a live reachable remote peer (and, for
#4, a deliberate user click-through) — to be completed in a dedicated
`/gsd:verify-work 138` session with a peer connected. The "shell 1" session was
left running for that follow-up.
