---
status: partial
phase: 134-modal-interaction
source: [134-VERIFICATION.md]
started: 2026-06-17T11:10:00Z
updated: 2026-06-17T11:10:00Z
---

## Current Test

Tests 1–4 PASS (after GAP-134-A/B/C fixes + detector #95). Test 6 (reduced motion) and Test 5 (remote two-machine) remaining.

## Tests

### 1. Grow animation visual (MODAL-01)
expected: Modal grows from the clicked card's center position with a smooth ~220ms scale animation; the card is the visual origin (not screen center).
result: PASS (re-UAT after GAP-134-A/B/C fixes) — clicking the local "claude 1" card opens the interactive modal; the modal grows from the card's position (~220ms, user confirmed); terminal mounts live (Claude Code prompt visible); header renders cleanly (normal-size status/origin/close icons, user confirmed). Three defects were found and fixed along the way: GAP-134-A (local card misrouted to remote join modal — provenance fix f84b10fe), GAP-134-B (header icons unsized/ballooned — CSS fix c19f538d), GAP-134-C (header showed globe+hostname for a local session — provenance fix 9e591846).

### 2. Shrink animation + focus return (MODAL-02)
expected: Closing via Escape, X button, and click-outside all shrink the modal back toward the originating card (~180ms); keyboard focus returns to the card that was clicked.
result: PASS — all three close paths (Escape, × button, click-outside) shrink + close cleanly; focus returns to the originating card (Enter/Space reopens it). User confirmed.

### 3. Interactive terminal functional check (MODAL-03/05)
expected: Full interactive terminal renders and accepts input; window resize reflows without jank or 0-column dims; copy/paste and scrollback search work.
result: PASS — command input/output, copy/paste, and scrollback all work. Minor: window resize causes a brief flash while the terminal repaints (xterm reflow repaint; cosmetic, non-blocking). → OBS-1 (polish candidate).

### 4. Briefing modal round-trip (MODAL-04)
expected: For a waiting session, briefing view shows the real terminal tail; respond textarea auto-focuses; typing + Send Response delivers input to the PTY and closes the modal; session leaves waiting state.
result: PASS (core mechanic) — required fixing the status detector first (#95) so the card flips to "Needs input"; once it did, the card showed the attention border + bell (Phase 133 ATTN also validated), clicking opened the briefing modal, header read "Local" + computer icon (GAP-134-C confirmed), and typing a menu-selection number + Send Response fed back to the agent (round-trip works). Two findings: (1) the tail text is garbled (collapsed spacing + raw kitty/mouse escapes) — pre-existing rendering tech debt, same root as the mini-preview → filed #96; (2) the briefing offers free-text response only, not a selectable rendering of the agent's menu options — this is the locked v3.6 design, deferred to #93 (agents don't emit parseable choice data). Neither is a 134 regression; core MODAL-04 round-trip verified.

### 5. Remote two-machine tailnet test (MODAL-06)
expected: On Machine A, clicking a remote (Machine B) card with no cap cached shows the join-code modal; after the exchange the Hub modal auto-opens; interactive terminal executes commands on Machine B; remote briefing tail shows real Machine B output; Send Response delivers to Machine B's PTY; font-size zoom works in the remote interactive modal.
result: BLOCKED by GAP-134-A (needs second tailnet machine regardless).

### 6. Reduced-motion behavior (A11Y-03)
expected: With macOS "Reduce Motion" enabled, the modal appears/disappears instantly — no scale or fade, no flash of invisible content.
result: BLOCKED by GAP-134-A (cannot open modal to observe motion).

## Out-of-scope observations (pre-existing, NOT Phase 134 regressions)

- **Two "New session" buttons** stacked top-right: `hub__header` button (added 131-04) + `HubFilterBar` button (added 131-03). Pre-existing since Phase 131. TODO: file a GitHub issue.
- **Garbled mini-preview text / briefing tail** (collapsed spacing + raw kitty/mouse escapes like `(B[<u[>1u`): regex ANSI-strip can't reconstruct cursor-positioned spacing and misses private CSI sequences. Phase 132 (CARD-07) + Phase 134 (MODAL-04). → **filed #96**.
- **Status shows "Running" not "Waiting"** for a session at a select-menu prompt: detector only matched `[y/n]`, not Claude Code select-menus. → **fixed this session (#95)**; confirmed live (card flips to "Needs input").
- **Briefing free-text response only** (no rendered menu selection): locked v3.6 design, deferred to **#93**.

## Summary

total: 6
passed: 4
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps

### GAP-134-A: Local session card click opens remote join-code modal (MODAL-01 broken for local sessions)
status: fixed (re-UAT passed — modal opens, terminal mounts)
fix: commit f84b10fe — HubPanel now discriminates local-vs-remote by provenance (`remoteIdSet` from the remoteSessions prop) in both handleCardClick and the modal render, not by hostname. Added FE-ROUTE-01c (local session with a non-empty machine hostname → modal opens, no cap flow). Full suite 1700/1700, tsc clean. Confirmed in-app: clicking the local card opens the interactive modal with a live terminal.

### GAP-134-B: Modal header icons render unsized (balloon to fill the header strip)
status: fixed_pending_reuat
detail: The hub-modal header Heroicons (status, origin computer/globe, attn bell, close X) had no CSS sizing. Heroicons have no intrinsic width/height and this project has no Tailwind (w-N/h-N are no-ops — documented at .hub-card__attn-icon), so the icons ballooned to giant pale shapes overlapping the header.
fix: commit c19f538d — added explicit px sizes for `.hub-modal__status-icon` (20), `.hub-modal__origin-icon` (16), `.hub-modal__attn-icon` (16), `.hub-modal__close svg` (18), matching the hub-card convention; +4 CSS-contract tests. Suite green, tsc clean. Confirmed in-app: header renders with normal-size icons.

### GAP-134-C: Modal header origin marker mislabels local sessions as remote (globe icon + machine hostname)
status: fixed (re-UAT passed)
detail: HubModal computed `isLocal` from `session.hostname`; local sessions carry os.Hostname(), so the header showed the GlobeAltIcon + machine name instead of the ComputerDesktopIcon + "Local". Same hostname-vs-provenance bug class as GAP-134-A. The computer-vs-globe icon is a colorblind-safe non-color cue, so the wrong icon is a real defect.
fix: commit 9e591846 — `isLocal = !remote` (provenance prop already passed to HubModal). Added source-inspection tests. Suite green, tsc clean.
