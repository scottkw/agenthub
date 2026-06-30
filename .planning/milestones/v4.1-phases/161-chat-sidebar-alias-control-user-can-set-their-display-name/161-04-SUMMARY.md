---
phase: 161-chat-sidebar-alias-control-user-can-set-their-display-name
plan: "04"
subsystem: testing
tags: [playwright, vitest, e2e, alias, chat, cross-surface, regression]

requires:
  - phase: 161-03
    provides: alias control in ChatPanel header (.chat-panel__alias-label), validateAlias, handleAliasCommit

provides:
  - ALIAS-UI-01 cross-surface e2e proof (alias set on web client A propagates to web client B presence roster)
  - TESTING.md consolidated for all Phase 161 tests (Suite Manifest §2, Traceability §4, Manual §5)
  - Live-UAT regression fix: ChatPanel.currentAlias derives from live roster entry, not frozen MsgSelf snapshot (603b6e0b)
  - Human-verified: live desktop-owner alias propagation, web pre-fill of Tailscale computed name, RO-enabled control

affects:
  - phase-162
  - phase-163
  - gsd-verify-work (UAT routing via coverage block)

tech-stack:
  added: []
  patterns:
    - "Playwright alias-propagation test: two-RW-context harness with waitForHubSubscribers(4), assert roster avatar title only (no past-message relabeling per RESEARCH Pitfall 3)"
    - "Live-roster-key pattern: ChatPanel.currentAlias = participants[selfIdentity.personKey]?.alias ?? selfIdentity.alias for live tracking (not frozen connect-time snapshot)"

key-files:
  created: []
  modified:
    - frontend/e2e/chat-parity.spec.ts
    - TESTING.md
    - frontend/src/components/Hub/ChatPanel.tsx
    - frontend/src/components/Hub/ChatPanel.test.tsx

key-decisions:
  - "currentAlias priority: live presence roster entry for self.personKey takes precedence over frozen selfIdentity.alias; selfIdentity.alias is fallback seed only (pre-roster web pre-fill window)"
  - "No past-message relabeling asserted in ALIAS-UI-01 e2e per RESEARCH Pitfall 3 (per-message snapshot is expected behavior, not a bug)"
  - "TESTING.md delta deferred to Plan 161-04 so Plans 161-01..03 had disjoint file sets during parallel execution"

patterns-established:
  - "Presence-roster-as-truth pattern: for self-identity display, always resolve from the live broadcasted roster entry rather than the connect-time handshake frame"

requirements-completed: [ALIAS-UI-01, ALIAS-UI-02]

coverage:
  - id: D1
    description: "ALIAS-UI-01 e2e: alias set on web client A propagates to web client B presence roster (.chat-presence avatar title)"
    requirement: ALIAS-UI-01
    verification:
      - kind: e2e
        ref: "frontend/e2e/chat-parity.spec.ts#ALIAS-UI-01 — alias set on client A propagates to client B presence roster"
        status: pass
    human_judgment: false
  - id: D2
    description: "TESTING.md consolidated for all Phase 161 tests: §2 Suite Manifest delta, §4 Traceability rows (ALIAS-UI-01/02), §5 Category S manual items (M-32/M-33)"
    requirement: ALIAS-UI-01
    verification:
      - kind: other
        ref: "bash tests/check-traceability-paths.sh exits 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "Live-UAT regression fix: ChatPanel header tracks live roster alias for self, not frozen MsgSelf snapshot (ALIAS-UI-02)"
    requirement: ALIAS-UI-02
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/ChatPanel.test.tsx#currentAlias tracks roster for desktop personKey after alias change"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/ChatPanel.test.tsx#currentAlias tracks roster for web personKey after alias change"
        status: pass
      - kind: manual_procedural
        ref: "Phase 161-04 live UAT — desktop alias change updated header label live (603b6e0b)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live-verified: desktop alias propagates to web guest roster; web pre-fill shows Tailscale computed name; RO control is not disabled"
    requirement: ALIAS-UI-01
    verification:
      - kind: manual_procedural
        ref: "Phase 161-04 human-verify checkpoint — APPROVED by user"
        status: pass
    human_judgment: true
    rationale: "Live Tailnet + real web-share WS; cannot be automated (web-share WS blocks automated input per memory reference_live_uat_daemon_gotchas)"

duration: 25min
completed: "2026-06-28"
status: complete
---

# Phase 161 Plan 04: Cross-Surface Proof + TESTING.md Consolidation Summary

**Playwright e2e proves alias set on one web client propagates to the other's presence roster (ALIAS-UI-01); TESTING.md consolidated for all Phase 161 tests; live-UAT regression fix for frozen MsgSelf header (603b6e0b) discovered and committed; cross-surface parity verified live against a real Tailnet web guest.**

## Performance

- **Duration:** ~25 min (continuation from human-verify checkpoint)
- **Started:** 2026-06-28T10:45:00Z (original agent)
- **Completed:** 2026-06-28T16:55:00Z (continuation agent including gate run)
- **Tasks:** 3 (Task 1: e2e, Task 2: TESTING.md, Task 3: human-verify APPROVED)
- **Files modified:** 4

## Accomplishments

- Extended `frontend/e2e/chat-parity.spec.ts` with a new `ALIAS-UI-01` test: two-RW-context Playwright harness drives client A's alias control (`.chat-panel__alias-label`), commits the new alias, and asserts client B's `.chat-presence` roster avatar title reflects the change. Test passes on Chromium, Firefox, and WebKit (27/27 total e2e).
- Consolidated `TESTING.md` for all Phase 161 tests: §2 Suite Manifest Phase 161 delta (6 extended test files across Plans 01–04), §4 Traceability rows for ALIAS-UI-01/02 (Go and TypeScript paths), §5 Category S (Phase 161) manual items M-32/M-33 for live-only behaviors.
- Human-verify checkpoint APPROVED live: desktop alias propagation updates header, message-author, and roster; web guest pre-fill shows Tailscale computed name; web alias propagates cross-surface; RO control accessible.
- Live-UAT regression found and fixed (603b6e0b): `ChatPanel.currentAlias` previously froze at the connect-time MsgSelf snapshot; now derives from the live presence roster entry for `selfIdentity.personKey`, falling back to `selfIdentity.alias` only as the pre-roster seed.

## Task Commits

1. **Task 1: Extend chat-parity.spec.ts with ALIAS-UI-01 alias-propagation e2e** - `b1b888a3` (test)
2. **Task 2: Consolidate TESTING.md (Suite Manifest, Traceability, Manual) + path check** - `ad728fd8` (docs)
3. **Task 3: Human-verify live cross-surface alias propagation + web pre-fill** - APPROVED (no code commit; regression fix in 603b6e0b)

**UAT regression fix (161-03 attribution):** `603b6e0b` fix(161-03): header tracks live roster alias for self, not frozen MsgSelf

## Files Created/Modified

- `frontend/e2e/chat-parity.spec.ts` — +72 lines: new ALIAS-UI-01 alias-propagation test reusing two-RW Playwright harness
- `TESTING.md` — +32 lines: Phase 161 §2 delta note; §4 ALIAS-UI-01/02 traceability rows; §5 Category S M-32/M-33 manual items
- `frontend/src/components/Hub/ChatPanel.tsx` — currentAlias derives from live roster entry for selfIdentity.personKey (603b6e0b)
- `frontend/src/components/Hub/ChatPanel.test.tsx` — +2 regression tests: desktop + web personKey roster-tracking (603b6e0b)

## Decisions Made

- **currentAlias priority chain:** `participants[selfIdentity.personKey]?.alias ?? selfIdentity.alias`. The personKey is `local:local` on desktop and the `tailnetID:web` key on web-share (matches the server's roster broadcast key). `selfIdentity.alias` is kept as fallback for the window before the roster first includes self (web pre-fill).
- **No past-message relabeling in e2e:** ALIAS-UI-01 asserts only roster propagation. ChatMessage snapshots are intentional (RESEARCH Pitfall 3) — asserting them would make the test brittle and over-assert non-requirements.
- **TESTING.md delta batched into Plan 04:** Plans 161-01..03 left TESTING.md untouched so their parallel Wave-1 file sets were disjoint. Plan 04 owns the full delta.

## Deviations from Plan

### Auto-fixed Issues (during live UAT — Rule 1: Bug)

**1. [Rule 1 - Bug] ChatPanel.currentAlias frozen on connect-time MsgSelf snapshot; header did not update after alias change**

- **Found during:** Task 3 (human-verify checkpoint — live Wails app)
- **Issue:** `currentAlias` was derived from `selfIdentity.alias` (set once from MsgSelf 0x37 on WS connect). After the user changes their alias, the server re-broadcasts the presence roster (NotifyPresence) but does NOT re-emit MsgSelf. The header "chatting as: «name»" label therefore stayed on the initial hostname default even though the message-author name and roster roster entry updated correctly.
- **Fix:** `ChatPanel.currentAlias` now reads `participants[selfIdentity.personKey]?.alias ?? selfIdentity.alias`. The personKey-based lookup resolves from the live, re-broadcast roster so alias changes reflect immediately in the header.
- **Files modified:** `frontend/src/components/Hub/ChatPanel.tsx`, `frontend/src/components/Hub/ChatPanel.test.tsx`
- **Verification:** 2 new regression tests pass (desktop + web personKey); all 61 ChatPanel unit tests pass; live Wails app re-confirmed header updates on alias change.
- **Committed in:** `603b6e0b` (separate fix commit on main, attributed to 161-03)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug: frozen header alias after change)
**Impact on plan:** Bug fix required for ALIAS-UI-02 correctness; no scope creep.

### Deferred Out-of-Scope Items (user decision — NOT fixed in this phase)

The following surfaced during live UAT but were explicitly deferred by the user to a new phase:

1. **Long `authorID` (nodekey:…) renders un-truncated** in the message secondary label on the web-share surface, causing horizontal scroll (`ChatMessage.tsx`). Pre-existing cosmetic issue.
2. **Feature request:** make the chat window resizable width-wise.

These are pre-existing issues unrelated to the aliasing work. Neither prevents the Phase 161 goal from being achieved.

## Issues Encountered

None beyond the Rule 1 regression fix documented above. Gate ran clean (Go tests, vitest 2170/2170, tsc, Playwright 27/27, traceability path check).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 161 complete: ALIAS-UI-01 and ALIAS-UI-02 both proven (automated + live-verified).
- Phase 162 (Settings Polish — Terminal Plugins jump link) is independent; can proceed immediately.
- Phase 163 (Read-Only Guest Chat Posting, D-06 reconciliation) depends on ChatPanel understanding of RO cap; alias control correctly leaves RO alias-set enabled (no `isReadOnly` guard on `handleAliasCommit`), consistent with D-06.
- The two deferred cosmetic items (`authorID` truncation + resizable chat width) should be filed as GitHub issues or added to a future polish phase.

## Self-Check

- [x] `frontend/e2e/chat-parity.spec.ts` exists and contains ALIAS-UI-01 test
- [x] `TESTING.md` contains §5 Category S with M-32/M-33 items
- [x] Commit `b1b888a3` exists (test 161-04 e2e)
- [x] Commit `ad728fd8` exists (docs 161-04 TESTING.md)
- [x] Commit `603b6e0b` exists (fix 161-03 live regression)
- [x] Full phase gate: Go OK, vitest 2170/2170, tsc clean, Playwright 27/27, traceability OK

## Self-Check: PASSED

---
*Phase: 161-chat-sidebar-alias-control-user-can-set-their-display-name*
*Completed: 2026-06-28*
