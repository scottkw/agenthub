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
result: FIXED (re-UAT) — after GAP-134-A fix (commit f84b10fe), clicking the local "claude 1" card opens the interactive modal and the terminal mounts live (Claude Code prompt visible). Original failure: clicking a LOCAL card opened the "Join Remote Session — Files" join-code modal because HubPanel used `session.hostname` as the local/remote discriminator (handleCardClick L354, modal render L484); local sessions carry os.Hostname() so every local session was misclassified as remote. Fix = provenance (remoteSessions prop). A second visual defect surfaced on open → GAP-134-B (header icons unsized). Grow-animation visual confirmation still pending a clean re-check after GAP-134-B.

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
status: fixed (re-UAT passed — modal opens, terminal mounts)
fix: commit f84b10fe — HubPanel now discriminates local-vs-remote by provenance (`remoteIdSet` from the remoteSessions prop) in both handleCardClick and the modal render, not by hostname. Added FE-ROUTE-01c (local session with a non-empty machine hostname → modal opens, no cap flow). Full suite 1700/1700, tsc clean. Confirmed in-app: clicking the local card opens the interactive modal with a live terminal.

### GAP-134-B: Modal header icons render unsized (balloon to fill the header strip)
status: fixed_pending_reuat
detail: The hub-modal header Heroicons (status, origin computer/globe, attn bell, close X) had no CSS sizing. Heroicons have no intrinsic width/height and this project has no Tailwind (w-N/h-N are no-ops — documented at .hub-card__attn-icon), so the icons ballooned to giant pale shapes overlapping the header.
fix: commit c19f538d — added explicit px sizes for `.hub-modal__status-icon` (20), `.hub-modal__origin-icon` (16), `.hub-modal__attn-icon` (16), `.hub-modal__close svg` (18), matching the hub-card convention; +4 CSS-contract tests. Suite green, tsc clean. Awaiting in-app re-check (HMR reload; reopen the modal).
