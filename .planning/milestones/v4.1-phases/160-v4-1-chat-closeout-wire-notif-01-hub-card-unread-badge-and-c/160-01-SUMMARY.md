---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: "01"
subsystem: frontend/Hub
tags: [notif-01, chat, unread, hook, tdd]
status: complete

dependency_graph:
  requires: []
  provides:
    - useChatUnreadListeners hook (background WS unread source for NOTIF-01)
  affects:
    - HubPanel (plan 160-02 will consume this hook)
    - SessionCard unread badge (downstream via HubPanel state)

tech_stack:
  added: []
  patterns:
    - useEffect with stable sessionIdKey dep (copied from usePreviewPoller in HubPanel.tsx)
    - useRef<Map> for per-session state accumulation (avoids re-render storms)
    - vi.fn(function() {...}) constructor mock (arrow functions cannot be used with `new`)
    - createRoot + act from react (project test pattern — @testing-library/react not installed)

key_files:
  created:
    - frontend/src/components/Hub/useChatUnreadListeners.ts
    - frontend/src/components/Hub/useChatUnreadListeners.test.tsx
  modified: []

decisions:
  - "Use useRef<Map<string, UnreadState>> for per-session accumulation, not useState, to avoid re-render storms on every MsgChat frame (per plan spec and Pitfall 5)"
  - "Import accrueUnread from ChatPanel — not re-implemented (per plan prohibition)"
  - "Test uses createRoot+act from react instead of @testing-library/react renderHook: package not in devDependencies; all 5 behavior cases still covered"
  - "Constructor mock uses function keyword not arrow function: arrow functions cannot be used with `new` in JavaScript"

metrics:
  duration: "~5 minutes"
  completed: "2026-06-27"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 0
---

# Phase 160 Plan 01: Unread Listeners Hook Summary

Background WS unread source for NOTIF-01: read-only relay subscription per backgrounded session, accruing unread counts without double-counting the open-modal session.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Write failing test for useChatUnreadListeners (RED) | 525410d9 | frontend/src/components/Hub/useChatUnreadListeners.test.tsx |
| 2 | Implement useChatUnreadListeners hook (GREEN) | 197192a9 | frontend/src/components/Hub/useChatUnreadListeners.ts, .test.tsx (updated) |

## Verification

- `pnpm vitest run src/components/Hub/useChatUnreadListeners` — 6/6 tests GREEN
- `pnpm exec tsc --noEmit` — zero errors in the two new files
- `accrueUnread` imported from ChatPanel (grep confirms — no re-implementation)
- Per-session state uses `useRef<Map<string, UnreadState>>` (not useState)

## Deviations from Plan

### Test Framework Deviation (Rule 3 — auto-fixed)

**Found during:** Task 1 (RED)
**Issue:** Plan specified `@testing-library/react` `renderHook` + `act` for the test. That package is not in the project's `devDependencies` and was not available.
**Fix:** Rewrote the test using the project's established pattern: `createRoot` + `act` from `react` directly (same technique as `HubPanel.test.tsx`). A thin `HookRunner` wrapper component calls the hook under test. All 5 plan behavior cases are covered.
**Files modified:** `useChatUnreadListeners.test.tsx`
**Commit:** 197192a9

### Constructor Mock Requires Function Keyword (Rule 1 — auto-fixed)

**Found during:** Task 2 (GREEN → first run)
**Issue:** PATTERNS.md showed the RelayClient mock with an arrow function implementation (e.g., `vi.fn().mockImplementation((...) => ({ ... }))`). Arrow functions cannot be used as constructors; calling `new RelayClient(...)` threw `is not a constructor`.
**Fix:** Changed mock to `vi.fn(function(this: any, ...) { this.callbacks = callbacks; this.close = vi.fn() })`. Also updated test assertions to use `mock.instances` (not `mock.results[i].value`) since constructor `this` is tracked in `mock.instances`.
**Files modified:** `useChatUnreadListeners.test.tsx`
**Commit:** 197192a9

## Known Stubs

None — the hook is fully implemented. The `onUnreadChange` callback is the only outward signal; HubPanel (plan 160-02) will wire it to unreadMap state.

## Threat Surface Scan

No new network endpoints or auth paths introduced. The hook connects only to `ws://127.0.0.1:{relayPort}/sessions/{id}/ws` — the same local loopback relay path already used by `TerminalPanel` and `ChatPanel`. No new threat surface beyond what the plan's threat model already registered (T-160-01 accepted, T-160-02 mitigated by cleanup).

## Self-Check

**Files exist:**
- `frontend/src/components/Hub/useChatUnreadListeners.ts` — FOUND
- `frontend/src/components/Hub/useChatUnreadListeners.test.tsx` — FOUND

**Commits exist:**
- `525410d9` (RED test) — FOUND
- `197192a9` (GREEN implementation) — FOUND

## Self-Check: PASSED
