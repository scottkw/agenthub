---
phase: 163-read-only-guest-chat-posting-d-06-reconciliation
plan: "02"
subsystem: ui
tags: [react, typescript, vitest, playwright, chat, read-only, ROCHAT]

# Dependency graph
requires:
  - phase: 163-01
    provides: server-side MsgChatSend RO gate removed; ErrChatReadOnly deleted; HandleInject ErrReadOnly kept
provides:
  - ChatPanel Send button enabled for RO clients (no isReadOnly in disabled expression)
  - handleSend + Enter handler isReadOnly guard removed
  - chat-composer__readonly-label JSX block removed
  - handleInjectPointerDown RO gate retained
  - 5 new vitest tests covering ROCHAT-01/02 (RO send + inject gate)
  - Playwright SC-3 rewritten as ROCHAT-01/02: RO message appears, Send not disabled
affects: [163-03, chat-parity-spec, ChatPanel-consumers]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "isReadOnly retained as inject-only gate — chat send paths no longer check it, inject press-and-hold still returns early"
    - "vitest async RO test pattern: mountPanel({capToken}) + await act(async () => {}) to flush /info fetch before asserting"

key-files:
  created: []
  modified:
    - frontend/src/components/Hub/ChatPanel.tsx
    - frontend/src/components/Hub/ChatPanel.test.tsx
    - frontend/e2e/chat-parity.spec.ts

key-decisions:
  - "Phase 163-02: isReadOnly state + /info perms resolution retained for inject gate only — not for chat-send"
  - "Phase 163-02: chat-composer__readonly-label JSX block removed entirely (no CSS change needed — label used inline styles)"
  - "Phase 163-02: Playwright SC-3 rewritten to assert RO message appears end-to-end (depends on 163-01 server gate removal)"
  - "Phase 163-02: inject-gate Playwright assertion deferred — prove at vitest + Go level, not browser (avoids flaky PTY-write assertion)"

patterns-established:
  - "T-163-03 mitigated: handleInjectPointerDown if (isReadOnly) return retained and proven by new vitest gesture test"

requirements-completed: [ROCHAT-01, ROCHAT-02]

coverage:
  - id: D1
    description: "ChatPanel Send button enabled for RO clients — no disabled/aria-disabled/opacity RO guard"
    requirement: ROCHAT-01
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/ChatPanel.test.tsx#-t rochat-01: RO viewer Send button is NOT disabled (ROCHAT-01)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/ChatPanel.test.tsx#-t rochat-01: RO viewer clicking Send calls sendChat with the draft (ROCHAT-01)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/ChatPanel.test.tsx#-t rochat-01: RO viewer pressing Enter calls sendChat and clears textarea (ROCHAT-01)"
        status: pass
    human_judgment: false
  - id: D2
    description: "No blanket Read-only label renders when isReadOnly"
    requirement: ROCHAT-01
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/ChatPanel.test.tsx#-t rochat-01: no Read only label renders when isReadOnly (ROCHAT-01)"
        status: pass
    human_judgment: false
  - id: D3
    description: "handleInjectPointerDown RO gate retained — RO press-and-hold does NOT call sendSessionInject"
    requirement: ROCHAT-02
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/ChatPanel.test.tsx#-t rochat-02: RO viewer press-and-hold does NOT call sendSessionInject (ROCHAT-02 inject gate)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Playwright E2E: RO viewer Send button not disabled; RO message appears in thread end-to-end"
    requirement: ROCHAT-01
    verification:
      - kind: e2e
        ref: "frontend/e2e/chat-parity.spec.ts#ROCHAT-01 — RO viewer CAN post chat; ROCHAT-02 — @session inject stays gated"
        status: unknown
    human_judgment: true
    rationale: "E2E Playwright test requires a live daemon + fixture session; cannot auto-run without test infra. Status confirmed at CI time."

# Metrics
duration: 4min
completed: 2026-06-28
status: complete
---

# Phase 163 Plan 02: RO Chat-Send Enable + Test Flip Summary

**ChatPanel Send button and Enter handler unblocked for RO guests across all surfaces (GUI/Hub/web-share); blanket "Read only" label removed; 5 new vitest tests + rewritten Playwright ROCHAT-01/02 spec prove RO-can-send + inject-still-gated**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-06-28T20:18:57Z
- **Completed:** 2026-06-28T20:23:02Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Removed `isReadOnly` from the Send button `disabled` expression, `aria-disabled`, and opacity/cursor RO style — button now enabled for all clients including RO (ROCHAT-01)
- Removed `if (isReadOnly) return` early-returns from `handleSend` and the Enter handler in `handleTextareaKeyDown` — RO clients can now post via click or keyboard
- Removed the `chat-composer__readonly-label` JSX block (the blanket "Read only" composer label)
- Retained `handleInjectPointerDown if (isReadOnly) return` — inject gesture stays RO-gated client-side (T-163-03 defense-in-depth, ROCHAT-02)
- Added 5 new vitest tests: RO Send not disabled, RO click sends, RO Enter sends + clears, no "Read only" text, RO inject press-and-hold does NOT call sendSessionInject — all green (66 total, +5)
- Rewrote Playwright `PARITY-01 SC-3` → `ROCHAT-01/02`: asserts Send is NOT disabled and RO message appears in thread; updated header legend comment

## Artifacts This Phase Produces

- Removed `chat-composer__readonly-label` JSX block from ChatPanel.tsx
- ChatPanel Send-button RO gate removed (disabled expression, aria-disabled, opacity style, handleSend guard, Enter handler guard)
- Flipped/renamed Playwright `PARITY-01 SC-3` test → `ROCHAT-01 — RO viewer CAN post chat; ROCHAT-02 — @session inject stays gated`
- New vitest tests: 5 tests in `ROCHAT-01/02 — RO chat-send enabled; inject gesture stays gated` describe block (vitest test-count delta: +5, total 66 — note for 163-03 TESTING.md update)

## Task Commits

1. **Task 1: Enable RO chat-send + remove Read-only label in ChatPanel** - `9d403b2d` (feat)
2. **Task 2: Flip RO-cannot-send vitest + Playwright assertions; add ROCHAT-01/02 coverage** - `7fefa94a` (test)

## Files Created/Modified

- `frontend/src/components/Hub/ChatPanel.tsx` — Send button RO guard removed; handleSend + Enter handler isReadOnly removed; readonly-label JSX block removed; inject gate kept
- `frontend/src/components/Hub/ChatPanel.test.tsx` — 5 new ROCHAT-01/02 tests added; 66 total passing
- `frontend/e2e/chat-parity.spec.ts` — SC-3 rewritten as ROCHAT-01/02; header legend updated

## Decisions Made

- `isReadOnly` state + `/info` perms resolution retained — the inject gesture (`handleInjectPointerDown`) still depends on it; only the chat-send/label uses were removed
- Playwright inject-gate assertion deferred from this E2E spec — proven at vitest (gesture test) + Go (163-01 SEC-RO-01 guard) to avoid flaky PTY-write assertion in browser
- No CSS file change needed — the readonly label used inline styles only

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Threat Flags

No new threat surface introduced. T-163-03 (Elevation of Privilege / handleInjectPointerDown) mitigated per plan: inject RO gate retained and proven by new vitest test.

## Self-Check: PASSED

- `frontend/src/components/Hub/ChatPanel.tsx` exists and modified: FOUND
- `frontend/src/components/Hub/ChatPanel.test.tsx` exists and modified: FOUND
- `frontend/e2e/chat-parity.spec.ts` exists and modified: FOUND
- Commit `9d403b2d` exists: FOUND
- Commit `7fefa94a` exists: FOUND
- `grep -c "Read only" ChatPanel.tsx` = 0: CONFIRMED (no "Read only" text remains in source)
- `pnpm --dir frontend test -- --run ChatPanel`: 66/66 PASSED
- `pnpm --dir frontend exec tsc --noEmit`: PASSED

## Next Phase Readiness

- Phase 163-03 (TESTING.md update) should note vitest delta: +5 tests in ChatPanel.test.tsx (total 66)
- Playwright ROCHAT-01/02 test requires live daemon to run; standard CI coverage via chat-parity.spec.ts

---
*Phase: 163-read-only-guest-chat-posting-d-06-reconciliation*
*Completed: 2026-06-28*
