---
status: partial
phase: 134-modal-interaction
source: [134-VERIFICATION.md]
started: 2026-06-17T11:10:00Z
updated: 2026-06-17T11:10:00Z
---

## Current Test

Test 1 FAILED — blocked by a Phase 134 routing gap (see Gaps). Tests 2–6 blocked until the gap is fixed.

## Tests

### 1. Grow animation visual (MODAL-01)
expected: Modal grows from the clicked card's center position with a smooth ~220ms scale animation; the card is the visual origin (not screen center).
result: FAIL — clicking a LOCAL session card ("claude 1" on this host) opened the "Join Remote Session — Files" join-code modal instead of the interactive/briefing modal. Root cause: HubPanel uses `session.hostname` as the local/remote discriminator (handleCardClick L354, modal render L484), but local sessions carry the machine hostname (os.Hostname()), so every local session is misclassified as remote. The code's own usePreviewPoller comments forbid the hostname check; provenance (sessions vs remoteSessions prop) is the correct discriminator. Introduced in 134-05/134-08. Behavioral tests missed it because they build local sessions with hostname:''. → GAP-134-A.

### 2. Shrink animation + focus return (MODAL-02)
expected: Closing via Escape, X button, and click-outside all shrink the modal back toward the originating card (~180ms); keyboard focus returns to the card that was clicked.
result: BLOCKED by GAP-134-A (cannot open the modal for a local session).

### 3. Interactive terminal functional check (MODAL-03/05)
expected: Full interactive terminal renders and accepts input; window resize reflows without jank or 0-column dims; copy/paste and scrollback search work.
result: BLOCKED by GAP-134-A.

### 4. Briefing modal round-trip (MODAL-04)
expected: For a waiting session, briefing view shows the real terminal tail; respond textarea auto-focuses; typing + Send Response delivers input to the PTY and closes the modal; session leaves waiting state.
result: BLOCKED by GAP-134-A.

### 5. Remote two-machine tailnet test (MODAL-06)
expected: On Machine A, clicking a remote (Machine B) card with no cap cached shows the join-code modal; after the exchange the Hub modal auto-opens; interactive terminal executes commands on Machine B; remote briefing tail shows real Machine B output; Send Response delivers to Machine B's PTY; font-size zoom works in the remote interactive modal.
result: BLOCKED by GAP-134-A (needs second tailnet machine regardless).

### 6. Reduced-motion behavior (A11Y-03)
expected: With macOS "Reduce Motion" enabled, the modal appears/disappears instantly — no scale or fade, no flash of invisible content.
result: BLOCKED by GAP-134-A (cannot open modal to observe motion).

## Out-of-scope observations (pre-existing, NOT Phase 134 regressions)

- **Two "New session" buttons** stacked top-right: `hub__header` button (added 131-04) + `HubFilterBar` button (added 131-03). Pre-existing since Phase 131. Candidate for a GitHub issue.
- **Garbled mini-preview text** (raw ANSI/mouse-tracking escapes like `(B[<u[>1u`): MiniPreview renders `GetSessionTailLines` output without stripping control sequences. Phase 132 (CARD-07). Candidate for a GitHub issue.
- **Status shows "Running" not "Waiting"** for a session at a prompt: daemon status-detector heuristic (`internal/daemon/engine.go`), known pre-existing tuning gap. Already a known candidate for a separate issue.

## Summary

total: 6
passed: 0
issues: 1
pending: 0
skipped: 0
blocked: 5

## Gaps

### GAP-134-A: Local session card click opens remote join-code modal (MODAL-01 broken for local sessions)
status: fixed_pending_reuat
fix: commit f84b10fe — HubPanel now discriminates local-vs-remote by provenance (`remoteIdSet` from the remoteSessions prop) in both handleCardClick and the modal render, not by hostname. Added FE-ROUTE-01c (local session with a non-empty machine hostname → modal opens, no cap flow). Full suite 1700/1700, tsc clean. Awaiting re-UAT in the app (reload the Wails dev window, click the local card again).
