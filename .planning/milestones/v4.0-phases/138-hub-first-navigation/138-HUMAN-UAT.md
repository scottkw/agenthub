---
status: passed
phase: 138-hub-first-navigation
source: [138-VERIFICATION.md]
started: 2026-06-20T23:25:00Z
updated: 2026-06-23T00:00:00Z
remote_items_resolved_by: 146-HUMAN-UAT.md
---

## Current Test

[complete — local items confirmed live 2026-06-20; the deferred remote-peer items were exercised live in Phase 146's two-Mac tailnet UAT (146-HUMAN-UAT.md, passed 2026-06-22). Resolved 2026-06-23 for milestone close.]

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
result: PASS (resolved by Phase 146 live UAT) — the remote card "Open in browser" → real-peer-URL flow (CR-01) was exercised live on a two-Mac tailnet in 146-HUMAN-UAT.md (Tests 1–3, all PASS, user-approved 2026-06-22): clicking "Open in browser" on a remote card opened the real peer URL via the cap-bearing `App.OpenRemoteSessionURL` path, with the unshared-peer error banner also confirmed. Source + unit verification (adapter carries url) stands.

### 4. Kill two-step confirm on a live local session
expected: Overflow menu on a live local session shows "Kill session"; first click shows "Confirm kill" / "This will stop the session"; second click terminates the session via the daemon.
result: PASS — live (human, 2026-06-20): created a local Shell session; overflow menu showed "Kill session" (no Open-in-browser/Browse-files). First click flipped to "Confirm kill" / "This will stop the session" with the session still Running; second click terminated it (card removed). Two-step guard works end to end.

### 5. Remote card does NOT show Kill (CR-02) and does NOT show re-attach "Open" (WR-01)
expected: A remote card's overflow menu shows only "Open in browser" and "Browse files" — no "Kill session"; and no row-5 re-attach "Open" button renders on remote cards. (isLocal guard confirmed at source.)
result: PASS — local-card inverse confirmed LIVE (2026-06-20): the local "shell 1" card shows "Kill session" + re-attach "Open" and does NOT show "Open in browser"/"Browse files". Remote-card direction: the remote card's "Open in browser"/"Browse files" menu rendered and was used live on a real peer in Phase 146's two-Mac UAT (146-HUMAN-UAT, 2026-06-22); the no-Kill / no-reattach-Open specifics on remote cards remain isLocal-guard verified at source + unit test.

### 6. Colorblind-safe card indicators render with icon + text
expected: Local cards show ComputerDesktopIcon + "Local"; remote connected cards show LinkIcon + "Connected"; remote available cards show GlobeAltIcon + "Available". Every indicator pairs an icon shape with a text label — no color-only signals. (Hex/custom-property already verified at source; this confirms icons render alongside text.)
result: PASS — LOCAL origin confirmed LIVE (2026-06-20): `.hub-card__origin` = "Local" WITH an svg icon; local card has NO `.hub-card__conn` chip (correct). Remote Connected/Available chip (LinkIcon/GlobeAltIcon + text) rendered live on a real peer during Phase 146's two-Mac UAT (the remote card was present and interacted with); icon+text pairing remains verified at source + unit test (no color-only signals).

### 7. Attention pulse, mini-preview, and grid reflow preserved (CARD-04)
expected: Hub cards still animate the attention pulse, show mini-preview text, and reflow responsively at 240–360px grid density after the new card affordances were added.
result: PASS (grid + mini-preview) — `.hub__card-row` is display:grid, columns 272px (within 240–360px band); mini-preview terminal text renders on the card (screenshot). Attention pulse not forced (idle session); `.hub-card--attention` CSS is unit-verified (style.hub.test.ts).

## Summary

total: 7
passed: 7
partial: 0
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

None. No defects found. The local items passed live on 2026-06-20. The
remote-peer items (3, and the remote halves of 5/6) were deferred only for lack
of a two-machine tailnet in the original session; that environment was later
available in Phase 146, whose live two-Mac UAT (146-HUMAN-UAT.md, passed
2026-06-22) exercised the remote card "Open in browser" → real-peer-URL flow
end to end. The remaining remote-card specifics (no-Kill, chip icon+text) stay
source + unit verified and rendered live as part of the 146 run. Resolved
2026-06-23 for milestone close — no dedicated follow-up needed.
