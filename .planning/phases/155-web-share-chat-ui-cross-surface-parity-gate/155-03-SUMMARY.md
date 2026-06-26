---
phase: "155"
plan: "03"
subsystem: frontend
tags: [web-share, chat-ui, parity-gate, react, typescript]
dependency_graph:
  requires: ["155-02"]
  provides: ["WebShareSessionView component", "web-session Tab type", "App.tsx web-mode bootstrap", "PARITY-01 vitest coverage"]
  affects: ["frontend/src/App.tsx", "frontend/src/components/TabBar.tsx", "frontend/src/components/Hub/WebShareSessionView.tsx", "TESTING.md"]
tech_stack:
  added: []
  patterns: ["TerminalPanel + ChatPanel overlay (D-02 web-share)", "find-or-focus tab pattern (__websession__ prefix)", "wsURL prop threading to both terminal and chat children"]
key_files:
  created:
    - frontend/src/components/Hub/WebShareSessionView.tsx
    - frontend/src/components/Hub/WebShareSessionView.test.tsx
  modified:
    - frontend/src/components/TabBar.tsx
    - frontend/src/App.tsx
    - TESTING.md
decisions:
  - "isActive={true} constant on web-share (no grow animation, unlike modal's animated isActive={open})"
  - "openWebSessionTab called after handleOpenFileBrowser so session tab ends up as active tab (setActiveId ordering)"
  - "wsURL derived from window.location at render time — hermetic to host:port configured by vitest (tests use window.location.host not hardcoded 'localhost')"
  - "web-session excluded from the relayPort > 0 terminal loop guard (web-share port is 0)"
metrics:
  duration: "5 minutes"
  completed: "2026-06-26"
  tasks: 3
  files: 5
status: complete
requirements: [PARITY-01]
---

# Phase 155 Plan 03: WebShareSessionView Component Summary

WebShareSessionView wrapper component (TerminalPanel + ChatPanel overlay configured for webserver WS/REST endpoints) wired into App.tsx as the primary web-mode session tab.

## What Was Built

### Task 1: WebShareSessionView component (commit 9d6d4892)

New `frontend/src/components/Hub/WebShareSessionView.tsx` — a thin wrapper component that mirrors `HubInteractiveModal` in overlay structure:

- Props: `{ sessionId, capToken, relayPort, theme?, pluginConfig? }`
- Computes `wsURL = wss://{window.location.host}/sessions/{id}/ws?cap={encoded}` and `apiBaseURL = window.location.origin` internally
- Threads `wsURL` to **both** `TerminalPanel` and `ChatPanel` (Pitfall 6 guard: forgetting wsURL on TerminalPanel leaves terminal on ws://127.0.0.1:0 while chat works)
- Forwards `apiBaseURL` and `capToken` to `ChatPanel` for history/export fetches
- Uses `isActive={true}` (constant — no grow animation, unlike modal's `isActive={open}`)
- Root class `hub-modal__body hub-modal__body--interactive` and toggle class `hub-modal__chat-toggle` are verbatim copies of HubInteractiveModal so Playwright parity selectors work on both surfaces (PARITY-01)
- Unread badge state (`chatOpen`/`unreadCount`/`hasMention` + `handleUnreadChange`) copied from HubInteractiveModal

TypeScript: `pnpm -C frontend exec tsc --noEmit` clean.

### Task 2: App.tsx web-mode bootstrap + Tab type + render branch (commit 34e46b28)

- `TabBar.tsx`: extended `Tab.type` union with `'web-session'`
- `App.tsx`: added `openWebSessionTab(sessionId)` useCallback — find-or-focus with `__websession__${sessionId}` stable ID, pushes `{ type: 'web-session', name: 'Session', ... }` tab and calls `setActiveId`
- `App.tsx`: updated web-mode bootstrap effect — opens file browser first (background, existing call), then `openWebSessionTab` last (active/foreground) so the session tab ends up active on mount. Cap cannot be decoded client-side; file tab always opens and `PermissionDeniedTakeover` handles missing `files.read` (RESEARCH Pattern 5a — backward compatible)
- `App.tsx`: added `WebShareSessionView` render branch (`activeId.startsWith('__websession__')`) — renders with `relayPort={relayPort ?? 0}`, `capToken={webParams.capToken ?? ''}`, and `terminalTheme`/`pluginConfig` from existing App.tsx state
- `App.tsx`: excluded `'web-session'` from the `relayPort > 0` terminal loop guard (web-share relayPort is 0; wsURL inside WebShareSessionView overrides the sentinel)

Build: `pnpm -C frontend run build` green (Vite + Rolldown).

### Task 3: WebShareSessionView vitest + TESTING.md registration (commit 98ada10e)

`frontend/src/components/Hub/WebShareSessionView.test.tsx` — 11 tests:

1. Renders without throwing given minimal props
2. Root element carries `hub-modal__body--interactive` CSS class (parity gate)
3. Chat toggle button carries `hub-modal__chat-toggle` class (parity gate)
4. `wsURL = wss://{host}/sessions/{id}/ws?cap={encoded}` forwarded to TerminalPanel (Pitfall 6 guard)
5. Same `wsURL` forwarded to ChatPanel (Pitfall 6: both children need it)
6. Percent-encodes special characters in sessionId and capToken
7. wsURL shape matches `wss://...?cap=` pattern; TerminalPanel and ChatPanel receive identical URLs
8. `capToken` forwarded to ChatPanel
9. `apiBaseURL = window.location.origin` forwarded to ChatPanel
10. `sessionId` forwarded to both TerminalPanel and ChatPanel
11. Chat toggle interaction flips ChatPanel `open` prop

TESTING.md:
- Section 2: vitest count 125 → 126; Total 498 → 499
- Section 4: PARITY-01 traceability row pointing at `frontend/src/components/Hub/WebShareSessionView.tsx`
- `bash tests/check-traceability-paths.sh` exits 0

## Verification

- `pnpm -C frontend exec tsc --noEmit` — clean (exit 0)
- `pnpm -C frontend run build` — green (exit 0, chunk-size warnings are pre-existing)
- `pnpm -C frontend test run src/components/Hub/` — 344/344 tests pass (16 test files)
- `bash tests/check-traceability-paths.sh` — exits 0 ("OK: all traceability paths exist")

## Deviations from Plan

None — plan executed exactly as written.

One deviation note: test assertions used `window.location.host` and `window.location.origin` dynamically (not hardcoded `localhost`) because vitest configures jsdom with `http://localhost:3000` as the default origin, not `http://localhost`. This makes the tests hermetic to vitest's configured port — a correctness improvement, not a deviation from intent.

## Known Stubs

None — WebShareSessionView is fully wired. The component constructs real wsURL and apiBaseURL values from `window.location` at render time. In web-share context (App.tsx bootstrap), `webParams.sessionId` and `webParams.capToken` are read from the live URL query params.

## Threat Flags

None — no new trust boundaries introduced. `WebShareSessionView` only constructs a WS URL and passes it to children that already handle the webserver WS upgrade (`requireCapability` HMAC-verified `?cap=` + `requireAllowedOrigin` — unchanged, per T-155-04 in the plan's threat model). The cap token flows through existing Phase 154 patterns (`capToken?: string` on ChatPanel).

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| `frontend/src/components/Hub/WebShareSessionView.tsx` exists | FOUND |
| `frontend/src/components/Hub/WebShareSessionView.test.tsx` exists | FOUND |
| `155-03-SUMMARY.md` exists | FOUND |
| Commit 9d6d4892 (Task 1) | FOUND |
| Commit 34e46b28 (Task 2) | FOUND |
| Commit 98ada10e (Task 3) | FOUND |
