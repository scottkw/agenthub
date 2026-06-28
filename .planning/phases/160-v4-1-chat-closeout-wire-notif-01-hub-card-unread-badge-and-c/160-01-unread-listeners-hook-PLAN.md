---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: 01
type: tdd
wave: 1
depends_on: []
files_modified:
  - frontend/src/components/Hub/useChatUnreadListeners.ts
  - frontend/src/components/Hub/useChatUnreadListeners.test.tsx
autonomous: true
requirements: [NOTIF-01]
must_haves:
  truths:
    - "A backgrounded session (modal closed) accrues an unread count when a MsgChat frame arrives over its relay WS."
    - "The session whose modal is open is excluded from background subscription (no double-count)."
    - "When relayPort is 0 or isActive is false, the hook opens zero RelayClient connections."
    - "Closing/unmounting the hook (or removing a session) closes that session's RelayClient (no leak)."
  artifacts:
    - frontend/src/components/Hub/useChatUnreadListeners.ts
    - frontend/src/components/Hub/useChatUnreadListeners.test.tsx
  key_links:
    - "useChatUnreadListeners reuses the EXPORTED accrueUnread(prev, msg, 'local') from ChatPanel.tsx (no re-implementation)."
    - "RelayClient onChat callback -> accrueUnread -> onUnreadChange(sessionId, count, hasMention) is the only outward signal."
  prohibitions:
    - "MUST NOT call onUnreadChange for the openModalSessionId (double-count guard)."
    - "MUST NOT construct a RelayClient when relayPort <= 0 (ws://127.0.0.1:0 fails)."
    - "MUST NOT send any frame to the server — listener is read-only (onOutput is a no-op)."
    - "MUST NOT use useState for per-session accrual — use useRef to avoid re-render storms."
---

<objective>
Create the background unread source for NOTIF-01: a custom React hook `useChatUnreadListeners` that opens one lightweight read-only relay WS subscription per BACKGROUNDED session (modal closed) and accrues unread chat counts per session. This is Part B of the NOTIF-01 fix — the closed-modal unread source that does not exist today (the only blocker in the v4.1 milestone audit).

Purpose: A session whose modal is closed has no `ChatPanel` mounted, so it has no WS subscription and cannot accrue unread. This hook provides that subscription so the Hub session-card badge can light up for backgrounded sessions.

Output: `useChatUnreadListeners.ts` (new hook) + `useChatUnreadListeners.test.tsx` (new vitest file). The hook is consumed by HubPanel in plan 160-02.

Note: TESTING.md registration of the new test file (vitest count bump + §4 traceability row) is handled centrally in plan 160-05 (sole TESTING.md owner), satisfying the repo standing convention at the phase level.
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-RESEARCH.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-PATTERNS.md
@frontend/src/components/Hub/ChatPanel.tsx
@frontend/src/components/Hub/HubPanel.tsx
@frontend/src/lib/relayClient.ts
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Write failing test for useChatUnreadListeners (RED)</name>
  <files>frontend/src/components/Hub/useChatUnreadListeners.test.tsx</files>
  <read_first>
    - 160-PATTERNS.md lines 251-279 (test mock setup: vi.mock RelayClient + vi.mock ./ChatPanel accrueUnread; renderHook/act)
    - 160-RESEARCH.md lines 316-341 (Common Pitfalls 1-6) and lines 421-431 (test map)
    - frontend/src/components/Hub/HubPanel.test.tsx (existing Wails-RPC mock + jsdom pattern to mirror)
    - frontend/src/components/Hub/ChatPanel.tsx lines 54-62 (UnreadState shape) and 185-194 (accrueUnread)
  </read_first>
  <behavior>
    - Opens one RelayClient per session where session.id !== openModalSessionId, relayPort > 0, and isActive === true.
    - Excludes openModalSessionId from the subscription set (no listener for the open-modal session).
    - On an onChat callback, calls onUnreadChange(sessionId, count, hasMention) with the accrued count for that session.
    - Constructs zero RelayClients when relayPort === 0 (Pitfall 2) or when isActive === false (Pitfall: idle-tab gate).
    - On unmount, calls close() on every RelayClient it opened (Pitfall 3: no leak on session removal).
  </behavior>
  <action>
    Create the vitest file exercising the hook's public contract via @testing-library/react renderHook + act. Mock `../../lib/relayClient` so RelayClient captures the callbacks object and exposes a spy `close`. Mock `./ChatPanel`'s accrueUnread as a pure increment so the test asserts on threading, not accrual math. Drive each behavior case (subscription set membership, openModalSessionId exclusion, relayPort=0 skip, isActive=false skip, onChat -> onUnreadChange, cleanup-closes-clients) from the behavior list. Reference the mock shape in 160-PATTERNS.md lines 256-271 exactly. The test imports the not-yet-created hook from `./useChatUnreadListeners`; the file failing to resolve/compile is an acceptable RED state for this task.
  </action>
  <verify>
    <automated>cd frontend && pnpm vitest run src/components/Hub/useChatUnreadListeners 2>&1 | grep -qiE 'fail|error|cannot find|no test' && echo RED-OK</automated>
  </verify>
  <acceptance_criteria>
    Test file exists and the suite is RED (hook not yet implemented). All five behavior cases are encoded as separate assertions/it-blocks.
  </acceptance_criteria>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement useChatUnreadListeners hook (GREEN)</name>
  <files>frontend/src/components/Hub/useChatUnreadListeners.ts</files>
  <read_first>
    - 160-PATTERNS.md lines 27-109 (full hook pattern: signature, sessionIdKey, isActive-gate effect, per-session RelayClient construction, useRef accrual)
    - 160-RESEARCH.md lines 86-118 (Part B architecture + key implementation notes) and 278-302 (RelayClient usage + callback shape)
    - frontend/src/components/Hub/HubPanel.tsx lines 59-119 (usePreviewPoller — the lifecycle analog to copy: sessionIdKey dep, isActive gate, per-session effect, cleanup)
    - frontend/src/lib/relayClient.ts lines 192-283 (RelayClient constructor + callbacks; onOutput is required, not optional)
  </read_first>
  <action>
    Implement the hook with signature `useChatUnreadListeners(sessions, relayPort, openModalSessionId, isActive, onUnreadChange)` returning void (160-PATTERNS.md lines 50-56). Import and reuse the exported `accrueUnread` and `UnreadState` type from `./ChatPanel` — do NOT re-implement accrual or mention detection. Use a `useRef<Map<string, UnreadState>>` for per-session accumulation so onChat callbacks never trigger re-renders (Pitfall 5). Use a single `useEffect` keyed on a stable `sessionIdKey = sessions.map(s => s.id).join(',')` plus isActive in the dep array (copy HubPanel line 73). Inside the effect: early-return when `!isActive || sessions.length === 0`; for each session where `session.id !== openModalSessionId && relayPort > 0`, construct a RelayClient with `onOutput` as an explicit no-op (Pitfall 4) and an `onChat` that reads the session's prior UnreadState from the ref (defaulting to count 0 / hasMention false), calls `accrueUnread(prev, message, 'local')` (desktop owner is always 'local'), writes it back to the ref, then calls `onUnreadChange(session.id, next.count, next.hasMention)`. Return a cleanup that closes every RelayClient opened by this effect run. Mirror the eslint-disable exhaustive-deps line from usePreviewPoller. The hook sends no frames to the server — it is a read-only loopback listener.
  </action>
  <verify>
    <automated>cd frontend && pnpm vitest run src/components/Hub/useChatUnreadListeners && pnpm exec tsc --noEmit 2>&1 | grep -v 'useChatUnreadListeners' | grep -c 'useChatUnreadListeners' | grep -qx 0 && echo OK</automated>
  </verify>
  <acceptance_criteria>
    useChatUnreadListeners.test.tsx is fully GREEN. Hook compiles under tsc with no errors in the new files. accrueUnread is imported from ChatPanel (not re-implemented). Per-session state uses useRef.
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Hub client -> local relay WS | Background listener connects to `ws://127.0.0.1:{relayPort}` (loopback only) |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-160-01 | Information Disclosure | useChatUnreadListeners WS | low | accept | Listener is read-only over loopback; sends no frames to server; reuses the same local relay path TerminalPanel already uses. No new attack surface (RESEARCH §Security Domain). |
| T-160-02 | Denial of Service | per-session RelayClient leak | low | mitigate | Effect cleanup closes every opened RelayClient on session removal / unmount (Pitfall 3); gated on isActive to avoid idle connections. |
</threat_model>

<verification>
- `cd frontend && pnpm vitest run src/components/Hub/useChatUnreadListeners` is GREEN.
- `cd frontend && pnpm exec tsc --noEmit` reports no errors in the two new files.
- accrueUnread is imported from ChatPanel; no duplicate accrual logic exists in the hook.
</verification>

<success_criteria>
The hook provides a working closed-modal unread source: per-backgrounded-session read-only WS subscription, useRef accrual, onUnreadChange signal, correct exclusion/gating/cleanup — all proven by the vitest suite.
</success_criteria>

<output>
Create `.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-01-SUMMARY.md` when done.
</output>
