---
phase: 163-read-only-guest-chat-posting-d-06-reconciliation
plan: "03"
subsystem: docs
tags: [documentation, testing, pitfalls, traceability, security, read-only, d-06]

requires:
  - phase: 163-01
    provides: backend RO chat-send gate removed; ErrChatReadOnly deleted; inject gate unchanged; test names (TestHandleChatSend_ROCanPost, TestHandleChatSend_ROCanPostInjectStillGated, TestChatSend_ROCanPost_RelayPath, TestChatSend_ROCanPost_WebPath)
  - phase: 163-02
    provides: frontend ChatPanel Send RO gate removed; 5 new vitest tests; Playwright SC-3 rewritten as ROCHAT-01/02

provides:
  - PITFALLS.md Pitfall 1 reconciled to D-06 rule (RO posts chat + presence/typing; @session inject / PTY write still gated)
  - PITFALLS.md cites Phase 163 as the reconciliation point
  - TESTING.md Section 2 Phase 163 delta note (Go +1 net function, vitest +5 tests, Playwright SC-3 rewrite; counts unchanged 366/132/9/509)
  - TESTING.md Section 4 CHAT-01 rows updated to reflect D-06 flips
  - TESTING.md Section 4 ROCHAT-01/ROCHAT-02/SEC-RO-01 traceability rows added
  - tests/check-traceability-paths.sh: passes

affects: []

tech-stack:
  added: []
  patterns:
    - "Documentation-only plan: two files updated (PITFALLS.md + TESTING.md); no code changes"
    - "TESTING.md standing rule applied: Section 2 delta note + Section 4 traceability rows for every test flip/add"

key-files:
  created: []
  modified:
    - .planning/research/PITFALLS.md
    - TESTING.md

key-decisions:
  - "D-06 paper trail complete: PITFALLS.md no longer instructs future phases to add a chat-post RO gate; reconciled rule documented with Phase 163 citation"
  - "SEC-RO-01 traceability row added to TESTING.md Section 4 — single test (TestHandleChatSend_ROCanPostInjectStillGated) proves both invariants: RO chat allowed AND inject blocked"
  - "CHAT-01 TESTING.md rows updated to reflect D-06 behavior; no rows deleted (CHAT-01 requirement still valid for RW path tests)"

requirements-completed: [ROCHAT-01, ROCHAT-02, SEC-RO-01]

duration: 3min
completed: 2026-06-28
status: complete
---

# Phase 163 Plan 03: D-06 Documentation Reconciliation + TESTING.md Registration Summary

**PITFALLS.md Pitfall 1 reconciled to D-06 (RO posts chat, @session inject / PTY write still gated, Phase 163 cited); TESTING.md Section 2 + Section 4 updated with Phase 163 delta note and ROCHAT-01/ROCHAT-02/SEC-RO-01 traceability rows; traceability path check passes.**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-06-28T20:30:27Z
- **Completed:** 2026-06-28T20:33:30Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

### Task 1: PITFALLS.md reconciliation

- Replaced "RO cap holders: read chat, cannot post" with the D-06 reconciled rule: RO clients ARE full chat participants (chat post + presence/typing allowed); cite Phase 163 as the reconciliation point
- Strengthened `@session` inject / PTY write guidance: `HandleInject` ErrReadOnly and MsgInput discard are explicitly called out as unchanged and release-blocking; regression proof via `TestHandleChatSend_ROCanPostInjectStillGated` (SEC-RO-01)
- Security Mistakes table: reworded from "allowing chat post" to "allowing `@session` inject or MsgInput PTY write" so the risk remains accurate post-D-06
- "Looks Done But Isn't" checklist: reworded RO item to new verification semantics — chat persists + broadcasts (no rejection); inject returns MsgInjectError + PTY write count = 0
- Pitfall-to-Phase Mapping: updated verification cell ("RO CAN post chat; @session inject → no PTY write") and added Phase 163 as reconciliation point
- Technical Debt table row "Client-side @session suppression for RO users": left intact — it refers to @session inject (still gated), not chat post

### Task 2: TESTING.md Section 2 + Section 4

**Section 2 (Suite Manifest) — Phase 163 delta note added:**
- Go EXTENDED in-place (no new files): hub_chatsend_test.go +1 net function (ROCanPostInjectStillGated); two relay/webserver files had test renames (0 net each)
- vitest EXTENDED in-place (no new files): ChatPanel.test.tsx +5 ROCHAT-01/02 tests (66 total in that file)
- Playwright EXTENDED in-place (no new files): chat-parity.spec.ts SC-3 rewritten as ROCHAT-01/02
- Counts unchanged: **366 Go / 132 vitest / 9 Playwright / 509 total**

**Section 4 (Traceability Map) — rows updated/added:**
- Updated 3 CHAT-01 rows (hub_chatsend_test.go, server_chatsend_test.go, webserver/server_chatsend_test.go): descriptions corrected from "ErrChatReadOnly / silently dropped" to D-06 behavior (RO persists + broadcasts)
- Updated PARITY-01 row (chat-parity.spec.ts): SC-3 description updated to reflect ROCHAT-01/02 rewrite
- Added **ROCHAT-01** rows: hub_chatsend_test.go (TestHandleChatSend_ROCanPost) + chat-parity.spec.ts (E2E: Send not disabled + message appears)
- Added **ROCHAT-02** rows: hub_chatsend_test.go (SEC-RO-01 guard) + server_inject_test.go + webserver/inject_test.go + capability_test.go (standing guards) + ChatPanel.test.tsx (inject gesture client-side guard)
- Added **SEC-RO-01** row: hub_chatsend_test.go (TestHandleChatSend_ROCanPostInjectStillGated — dual-invariant proof)

## Task Commits

1. **Task 1: Reconcile PITFALLS.md per D-06 SEC-RO-01** - `3c718656` (docs)
2. **Task 2: Update TESTING.md Section 2 + Section 4 for Phase 163** - `4943a981` (docs)

## Files Created/Modified

- `.planning/research/PITFALLS.md` — Pitfall 1 "How to avoid" reconciled; Security Mistakes table reworded; "Looks Done But Isn't" updated; Pitfall-to-Phase Mapping updated with Phase 163
- `TESTING.md` — Section 2 Phase 163 delta note added; Section 4 CHAT-01 rows updated; PARITY-01 row updated; ROCHAT-01/02 + SEC-RO-01 rows added

## Decisions Made

- D-06 paper trail is now complete across all three documentation layers: code comments (163-01), component comments (163-02), and research/test docs (163-03)
- ROCHAT-02 traceability explicitly references standing pre-163 guards (inject_test.go, server_inject_test.go, capability_test.go) so future phases can see the complete gate coverage at a glance
- SEC-RO-01 gets its own row separate from ROCHAT-02 because it proves a distinct claim (the scope of the change), not just the inject guard behavior

## Deviations from Plan

None — plan executed exactly as written.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. Documentation-only plan.

## Self-Check: PASSED

- `.planning/research/PITFALLS.md` modified: EXISTS
- `TESTING.md` modified: EXISTS
- Commit `3c718656` (Task 1): VERIFIED
- Commit `4943a981` (Task 2): VERIFIED
- `grep -n "cannot post" .planning/research/PITFALLS.md | grep -v "intentionally removed"`: EMPTY (no stale assertion)
- `grep -qi "Phase 163" .planning/research/PITFALLS.md`: PASSES
- `bash tests/check-traceability-paths.sh`: PASSES (OK: all traceability paths exist)
- `grep -qE "ROCHAT-01|ROCHAT-02|SEC-RO-01" TESTING.md`: PASSES

---
*Phase: 163-read-only-guest-chat-posting-d-06-reconciliation*
*Completed: 2026-06-28*
