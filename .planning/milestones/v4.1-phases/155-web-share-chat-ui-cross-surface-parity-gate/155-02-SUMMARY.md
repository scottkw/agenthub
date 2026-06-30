---
phase: 155-web-share-chat-ui-cross-surface-parity-gate
plan: "02"
subsystem: frontend
tags: [chat, web-share, relay-client, terminal-panel, export, read-only, parity]
dependencies:
  requires: [155-01-SUMMARY.md]
  provides: [ChatPanel web-mode props, RelayClient wsURL override, TerminalPanel wsURL prop, Export button, RO cap suppression]
  affects: [frontend/src/lib/relayClient.ts, frontend/src/components/TerminalPanel.tsx, frontend/src/components/Hub/ChatPanel.tsx]
tech_stack:
  added: []
  patterns: [opt-in URL override (non-breaking), cap token opaque on client, Content-Disposition download via hidden anchor, server-authoritative perms resolution]
key_files:
  modified:
    - frontend/src/lib/relayClient.ts
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/Hub/ChatPanel.tsx
decisions:
  - "isReadOnly defaults to true (fail-safe): matches terminal.js precedent; cleared immediately for desktop (no capToken)"
  - "wsURL override evaluated before 127.0.0.1:${port} is built: port is 0 on web-share (Pitfall 1 prevention)"
  - "loadChatHistory extended with opts param: preserves exact desktop loopback behavior when opts absent"
  - "RO perms resolved from /api/sessions/{id}/info?cap= using whole-token membership of 'write' (no substring match)"
  - "data-chat-send present in both enabled and disabled states: Playwright selector contract must be stable regardless of RO status"
metrics:
  duration: "6 minutes"
  completed_date: "2026-06-26"
  tasks: 3
  files_modified: 3
status: complete
---

# Phase 155 Plan 02: Frontend Primitive Extensions — wsURL Override, Export Button, RO Suppression Summary

Additive prop extensions across relayClient.ts, TerminalPanel.tsx, and ChatPanel.tsx that enable the web-share surface: a wsURL override in RelayClient (short-circuiting the loopback URL), threading to TerminalPanel, and three new optional ChatPanel props (wsURL/apiBaseURL/capToken) plus an Export button and read-only cap suppression derived from the server's /info perms.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | RelayClient wsURL override + TerminalPanel wsURL prop | 3d23936b | relayClient.ts, TerminalPanel.tsx |
| 2 | ChatPanel web-mode props, loadChatHistory opts, Export button | 152f836a | ChatPanel.tsx |
| 3 | ChatPanel read-only cap suppression (PARITY-01 SC-3) | 55c80654 | ChatPanel.tsx |

## What Was Built

**Task 1 — RelayClient wsURL override + TerminalPanel wsURL prop:**

- `relayClient.ts`: constructor opts extended from `{ remote?: boolean }` to `{ remote?: boolean; wsURL?: string }`. When `opts.wsURL` is truthy, the URL is set verbatim before `127.0.0.1:${port}` is ever constructed (Pitfall 1: port is `0` on web-share). All existing behavior (binaryType, ping interval, onmessage frame switch) unchanged.
- `TerminalPanel.tsx`: `wsURL?: string` prop added alongside `remote?` with matching JSDoc. Destructured in component function; threaded into `new RelayClient(..., { remote, wsURL })`. Added to mount-effect dep array `[sessionId, wsURL]` so a web-mode remount reconnects with the override URL.

**Task 2 — ChatPanel web-mode props, loadChatHistory opts, Export button:**

- Three optional props added to `ChatPanelProps`: `wsURL?`, `apiBaseURL?`, `capToken?`. Non-breaking for desktop `HubInteractiveModal` caller (all optional).
- `loadChatHistory` extended with `opts?: { apiBaseURL?: string; capToken?: string }`. Derives base URL and appends `?cap=${encodeURIComponent(capToken)}` when present. Missing `?cap=` on web-share would return 401 and send the panel to error state (Pitfall 2 prevention).
- RelayClient construction now passes `{ wsURL }` as the opts arg; connect-effect dep array updated to include `wsURL`, `apiBaseURL`, `capToken`.
- Module-level `triggerExport(url: string)` helper: creates hidden `<a>`, sets `href`/`download=''`, appends, clicks, removes.
- Component-scoped `buildExportURL()` helper: `apiBaseURL ?? http://127.0.0.1:${relayPort}` + optional `?cap=`.
- Export button added to `chat-panel__header`: `ArrowDownTrayIcon` (18px), `data-chat-export` attribute, `aria-label`/`title="Export chat as Markdown"`, 36×36 click target. Present in both RO and RW modes (EXPORT-01).

**Task 3 — ChatPanel read-only cap suppression (PARITY-01 SC-3 defense-in-depth):**

- `isReadOnly` state added with fail-safe default `true` (matching `terminal.js` precedent).
- On mount (or when `capToken`/`sessionId` changes), resolves perms from `GET {apiBaseURL}/api/sessions/{id}/info?cap=`. Whole-token membership check: `perms.split(',').map(s => s.trim()).includes('write')` — matches `capability.HasPerm` semantics, no substring match. Any fetch error leaves `isReadOnly=true`. Desktop (no `capToken`) sets `isReadOnly=false` immediately.
- Send button: `data-chat-send` attribute (stable in both enabled/disabled states for Playwright), `disabled={!draft.trim() || isReadOnly}`, `aria-disabled={isReadOnly}`, `aria-label` switches to "Send message (read-only access)" when RO, reduced opacity + `cursor: not-allowed` visually.
- `handleSend`, `handleTextareaKeyDown` Enter path, and `handleInjectPointerDown` all early-return when `isReadOnly`.
- "Read only" label rendered below composer when `isReadOnly === true` (11px, `--hub-text-dim`, left-aligned).
- Server gating unchanged and authoritative (`HandleChatSend`/`HandleInject` bound to signed JWT claim); this task is defense-in-depth UX only.

## Verification

- `pnpm -C frontend exec tsc --noEmit` — clean (no type errors)
- `pnpm -C frontend test run src/components/Hub/` — 15 test files, 333 tests, all pass
- `pnpm -C frontend test run src/components/Hub/ChatPanel.test.tsx` — 39 tests pass (all existing desktop-behavior tests unchanged)
- `bash tests/check-traceability-paths.sh` — OK: all traceability paths exist

## Deviations from Plan

**None** — plan executed exactly as written.

The SOURCE-CONTRACT NOTE in Task 3 (UI-SPEC §2 describes RO detection via presence-roster `ReadOnly` field, but that field does NOT exist in `PresenceEntry` wire format) was correctly handled: RO is derived from `GET /api/sessions/{id}/info?cap=` perms as specified.

## Threat Model Coverage

| Threat ID | Disposition | Implementation |
|-----------|-------------|---------------|
| T-155-01 | mitigate | Client-side suppression: disabled Send, short-circuited inject, RO label derived from server /info perms. Server gate (HandleChatSend/HandleInject) unchanged and authoritative. |
| T-155-06 | mitigate | Cap never JWT-decoded client-side; forwarded opaquely as `?cap=${encodeURIComponent(token)}` in all web-surface calls. |
| T-155-05 | accept | Cap tokens in ?cap= appear in browser history; time-bounded JWTs, existing production pattern. No change. |
| T-155-SC | accept | No new npm packages added; no raw-HTML rehype plugin introduced. |

## Known Stubs

None — all props are wired at the primitive level. The WebShareSessionView caller (Plan 155-03) that computes and passes `wsURL`/`apiBaseURL`/`capToken` is the next plan.

## Self-Check: PASSED

- `frontend/src/lib/relayClient.ts` — exists and modified (commit 3d23936b)
- `frontend/src/components/TerminalPanel.tsx` — exists and modified (commit 3d23936b)
- `frontend/src/components/Hub/ChatPanel.tsx` — exists and modified (commits 152f836a, 55c80654)
- All three commits verified: `git log --oneline -5` shows 3d23936b, 152f836a, 55c80654
