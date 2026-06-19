---
phase: 134-modal-interaction
plan: "07"
subsystem: frontend/relay-client-remote-seam
tags: [typescript, react, websocket, relay-client, tdd, cr-fix, remote-modal]
dependency_graph:
  requires:
    - "134-06: GET /api/relay/remote/{sessionID}/ws daemon route (the proxy this plan wires to)"
  provides:
    - "RelayClient optional remote seam: opts?.remote=true selects /api/relay/remote/{id}/ws URL"
    - "TerminalPanel remote prop: threads remote flag into RelayClient construction"
    - "HubInteractiveModal remote prop: forwards remote to TerminalPanel (CR-01 fix)"
    - "HubBriefingModal: remote tail from WS scrollback snapshot (CR-02 fix) + CR-03 leak/race/unmount fix"
  affects:
    - frontend/src/lib/relayClient.ts
    - frontend/src/lib/relayClient.test.ts
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/Hub/HubInteractiveModal.tsx
    - frontend/src/components/Hub/HubBriefingModal.tsx
tech_stack:
  added: []
  patterns:
    - "RelayClient 4th opts param: opts?.remote=true → /api/relay/remote/{id}/ws; default → /sessions/{id}/ws"
    - "TerminalPanel remote prop threaded into RelayClient construction as { remote }"
    - "Remote tail via short-lived proxied WS + onOutput chunk accumulation + client-side ANSI strip + last-N-lines"
    - "CR-03: clientRef + settled flag + clearTimeout + useEffect(() => () => clientRef.current?.close(), []) unmount cleanup"
    - "Cap token never in URL or React state (T-134-07-01 invariant)"
key_files:
  created: []
  modified:
    - frontend/src/lib/relayClient.ts
    - frontend/src/lib/relayClient.test.ts
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/Hub/HubInteractiveModal.tsx
    - frontend/src/components/Hub/HubBriefingModal.tsx
decisions:
  - "opts param on RelayClient (not a new subclass): minimizes diff; all existing callers unaffected by the optional 4th param"
  - "Tail snapshot timeout 3s at transport + 500ms after onOpen: 3s outer guards stalled connect; 500ms inner gives the peer time to replay the full scrollback before we close"
  - "extractTailLines client-side ANSI strip mirrors engine.go GetSessionTailLines (no new backend endpoint needed — the snapshot IS the tail)"
  - "settled guard + clientRef + useEffect cleanup: exact CR-03 fix from 134-REVIEW verbatim; applies to both local and remote send paths"
metrics:
  duration: "~5 minutes"
  completed: "2026-06-17"
  tasks: 3
  files_created: 0
  files_modified: 5
---

# Phase 134 Plan 07: Frontend Remote Wiring + CR-03 Fix Summary

**One-liner:** Added an optional `remote` seam to `RelayClient`/`TerminalPanel`/`HubInteractiveModal` that selects the daemon-proxy WS URL for remote sessions (CR-01), rewrote `HubBriefingModal`'s tail fetch to read the peer's scrollback snapshot via the proxied WS instead of the local-only `GetSessionTailLines` (CR-02), and applied the REVIEW's exact CR-03 settled-flag + clientRef + clearTimeout + unmount-cleanup fix to the briefing Send path.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Failing URL tests for RelayClient remote seam | 9ba5ee7f | frontend/src/lib/relayClient.test.ts |
| 1 (GREEN) | RelayClient opts.remote seam + URL unit test | a9eee5c6 | frontend/src/lib/relayClient.ts |
| 2 | Thread remote into TerminalPanel + HubInteractiveModal | e52779c6 | frontend/src/components/TerminalPanel.tsx, frontend/src/components/Hub/HubInteractiveModal.tsx |
| 3 | Remote tail via WS snapshot + CR-03 fix in HubBriefingModal | ae269b60 | frontend/src/components/Hub/HubBriefingModal.tsx |

## What Was Built

### Task 1: RelayClient `opts` seam (TDD)

- Added a 4th optional `opts?: { remote?: boolean }` parameter to the `RelayClient` constructor.
- When `opts?.remote` is true, builds `ws://127.0.0.1:${port}/api/relay/remote/${sessionId}/ws` (routes through the Plan 06 daemon proxy).
- When opts is omitted or `opts.remote` is false/undefined, builds `ws://127.0.0.1:${port}/sessions/${sessionId}/ws` (unchanged — all existing callers preserved).
- Cap token never appears in the constructed URL in either mode (T-134-07-01).
- `frontend/src/lib/relayClient.test.ts` extended with three FE-URL-01 behavioral tests using a `vi.stubGlobal('WebSocket', MockWebSocket)` stub that captures the constructed URL; all three assert URL form and absence of the cap token.

### Task 2: TerminalPanel + HubInteractiveModal remote prop

- `TerminalPanel` gains `remote?: boolean` prop; the existing `new RelayClient(relayPort, sessionId, {...})` call gets `{ remote }` as the 4th arg.
- `HubInteractiveModal` gains `remote?: boolean` prop forwarded to `TerminalPanel`.
- Fixes CR-01: when `HubPanel` (Plan 08) passes `remote={isRemote}`, the interactive terminal will route through the daemon proxy instead of the local relay for remote sessions.
- Both props are optional; no existing caller breaks.

### Task 3: HubBriefingModal — CR-02 + CR-03

**CR-02 (remote tail):**
- New `remote?: boolean` prop on `HubBriefingModalProps`.
- When `remote` is true: opens a short-lived `RelayClient` with `{ remote: true }`, accumulates `onOutput` payload bytes (MsgOutput frames — the peer replays the scrollback snapshot as the first frames on connect), then closes after 500ms post-`onOpen` (gives the peer time to flush all snapshot frames).
- Client-side `stripAnsi()` + `extractTailLines()` mirror `engine.go GetSessionTailLines` stripping and last-N-lines logic.
- 3s outer timeout guards stalled connections: `finish()` fires regardless, yielding an empty tail rather than hanging.
- When `remote` is false/undefined: unchanged `GetSessionTailLines(session.id, 20)` local path.

**CR-03 (leak/race/unmount fix):**
- `clientRef = useRef<RelayClient | null>(null)` tracks the in-flight send client.
- `settled` flag (local to the Promise executor) prevents post-abandon sendInput.
- `timer` handle (returned by `setTimeout(..., 5000)`) is stored so `clearTimeout(timer)` can be called on the happy-path settlement.
- Reject/timeout path calls `clientRef.current?.close()` before `reject()` — the WS and its 30s ping interval are torn down on timeout.
- `onOpen` guard: `if (settled) { client.close(); return }` — if `onOpen` fires after the 5s reject, the text is never sent.
- `useEffect(() => () => { clientRef.current?.close() }, [])` — unmount cleanup closes any in-flight client if the modal is dismissed mid-send.
- The remote send path passes `remote ? { remote: true } : undefined` to `RelayClient` so remote sessions route through the proxy.

## Verification

| Check | Result |
|-------|--------|
| `pnpm test --run relayClient` | 14/14 pass (FE-URL-01a/b + existing framing tests) |
| `pnpm test` (full suite) | 1689/1689 pass, 104/104 test files |
| `pnpm exec tsc --noEmit` | Clean (no errors) |

## TDD Gate Compliance

- RED: `test(134-07)` commit `9ba5ee7f` — FE-URL-01b failed (expected daemon-proxy URL, got local-direct URL)
- GREEN: `feat(134-07)` commit `a9eee5c6` — implementation makes all 14 relayClient tests pass
- REFACTOR: none required; the seam is minimal (5 lines in the constructor)

## Deviations from Plan

**1. [Rule 1 - Cleanup] Removed unused `parseServerFrame` import**
- **Found during:** Task 3 tsc check.
- **Issue:** Initial HubBriefingModal draft imported `parseServerFrame` from `relayClient`, but the remote tail path does not call it — the `RelayClient.onOutput` callback already delivers the decoded `MsgOutput` payload bytes (parsing happens inside RelayClient). TypeScript flagged the import as unused.
- **Fix:** Removed the import. The `onOutput` payloads are raw PTY bytes (already payload-stripped of the MSG_OUTPUT type byte by `parseServerFrame` inside RelayClient), which is exactly what `extractTailLines` needs.
- **Files modified:** `frontend/src/components/Hub/HubBriefingModal.tsx`
- **Commit:** ae269b60

## Threat Model Adherence

- T-134-07-01 (cap in React state/URL): mitigated — `opts.remote` is a boolean; constructed URLs contain no cap token; asserted in FE-URL-01b.
- T-134-07-02 (untrusted text late-delivery): mitigated — `settled` flag + `if (settled) { client.close(); return }` guard in `onOpen`; post-abandon text is never sent.
- T-134-07-03 (leaked RelayClient/WS + ping interval): mitigated — `clientRef` + reject-path `close()` + `useEffect` unmount cleanup; tail socket closed after 500ms post-connect.
- T-134-07-04 (XSS via rendered tail): mitigated — `stripAnsi()` strips ANSI; `extractTailLines` returns plain strings; rendered as `<pre>{tailLines.join('\n')}</pre>` (React auto-escapes; no `dangerouslySetInnerHTML`).
- T-134-07-SC (npm installs): N/A — no new frontend dependencies added.

## Known Stubs

None — the remote seam is fully wired to the Plan 06 daemon proxy. The discriminator value (`isRemote`) that drives the `remote` prop in `HubPanel` is supplied by Plan 08 (the HubPanel render update); the seam itself is complete.

## Manual / Deferred

- **Live two-peer terminal round-trip** (phase gate): interactive modal connects via daemon proxy; remote briefing shows real prompt; input reaches the PTY. Deferred to the phase gate with live tailnet peers.
- **`HubPanel.handleCardClick` isRemote discriminator** (Plan 08): passes `remote={isRemote}` to both modals; this plan adds the seam only.
- **HubBriefingModal behavioral tests** (WR-07): mock-RelayClient send/timeout/unmount + TAIL-01 WS snapshot tests deferred to Plan 08.

## Self-Check: PASSED
