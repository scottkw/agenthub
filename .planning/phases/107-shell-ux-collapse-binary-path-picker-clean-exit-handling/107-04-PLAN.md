---
phase: 107
plan: "107-04"
type: execute
status: pending
wave: 1
depends_on: ["107-02"]
requirements: [SHELL-12]
files_modified:
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/App.shellExit.test.tsx
autonomous: true
must_haves:
  truths:
    - "A `session:exit` event with `exitCode === 0` triggers an immediate `handleCloseTab(sessionId)` call AND does NOT add the session to `sessionExits` state."
    - "A `session:exit` event with `exitCode !== 0` adds the session to `sessionExits` state (existing ExitToast path) and does NOT call `handleCloseTab`."
    - "ExitToast renders nothing for exit-code-0 sessions (the `exits` prop never contains an entry for them)."
    - "When the auto-closing tab is the active tab, focus shifts to the adjacent tab (or Welcome if it was the only one) — existing handleCloseTab logic, asserted explicitly to lock the invariant."
    - "Exit-code 0 sessions never trigger the 5-second countdown — the entire countdown branch is skipped."
    - "The auto-close-session setting (D-11) becomes irrelevant for exit-code 0 (always closes immediately); it still gates non-zero-exit behavior — actually, this plan does NOT change non-zero behavior at all, so the user prompt's only behavioral delta is on exit-code 0."
  artifacts:
    - path: frontend/src/App.tsx
      provides: "session:exit handler branching on exitCode === 0 → handleCloseTabRef.current?.(sessionId) early-return"
      contains: "data.exitCode === 0"
    - path: frontend/src/components/__tests__/App.shellExit.test.tsx
      provides: "Vitest suite locking UI-SPEC §4 SHELL-12 5-assertion test contract"
  key_links:
    - from: App.tsx session:exit handler
      to: handleCloseTabRef.current
      via: "early-return branch: data.exitCode === 0 → close immediately"
    - from: handleCloseTab adjacency logic (~L703-708)
      to: setActiveId(next?.id ?? null)
      via: "existing left-tab focus shift, asserted by new test"
---

<objective>
SHELL-12 frontend: branch the `session:exit` event handler on `data.exitCode === 0` so clean exits close the tab immediately (skipping ExitToast and the 5-second countdown), while non-zero exits retain the existing ExitToast behavior. Wave 1 (consumes 107-02's normalized exit codes — daemon now guarantees that `data.exitCode` is the user-meaningful value, never -1).

Purpose: First-user test reported "exited with error" toast firing when typing `exit` in zsh — the user expected the tab to just disappear (like AI-CLI sessions do on natural completion). 107-02 fixes the daemon side (no more -1 leaking through); this plan closes the loop on the frontend by treating exit-code 0 as "task done, no toast needed."

Output: One small but precise change to the session:exit handler in App.tsx + a new Vitest suite asserting the 5 invariants from UI-SPEC §4 SHELL-12. No new components. No ExitToast.tsx changes (it just won't be rendered for clean exits).
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-CONTEXT.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-UI-SPEC.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-02-SUMMARY.md

@frontend/src/App.tsx
@frontend/src/components/ExitToast.tsx
@frontend/src/components/__tests__/App.exit.test.tsx

<interfaces>
Existing session:exit handler we are modifying (App.tsx ~L537-595):

```typescript
const offExit = EventsOn(
  'session:exit',
  (data: {
    sessionId: string
    exitCode: number
    sessionName: string
    cli: string
    duration: number
    finalStatus: string
  }) => {
    // Always record the exit (shows toast)
    const exitState: ExitState = {
      sessionId: data.sessionId,
      sessionName: data.sessionName,
      cli: data.cli,
      exitCode: data.exitCode,
      duration: data.duration,
      finalStatus: data.finalStatus,
      countdown: data.exitCode === 0 ? 5 : -1,
      cancelled: false,
    }
    setSessionExits(prev => ({ ...prev, [data.sessionId]: exitState }))

    // Only start auto-close countdown for clean exits (D-10) when enabled (D-11)
    if (data.exitCode === 0) {
      if (autoCloseRef.current) {
        const timer = setInterval(() => { /* ... countdown logic ... */ }, 1000)
        countdownTimers.current[data.sessionId] = timer
      } else {
        setSessionExits(prev => ({
          ...prev,
          [data.sessionId]: { ...prev[data.sessionId], countdown: -1 },
        }))
      }
    }
  }
)
```

Per UI-SPEC §2 SHELL-12, replace with this branching shape:

```typescript
if (data.exitCode === 0) {
  // Close tab immediately — no toast, no countdown
  void handleCloseTabRef.current?.(data.sessionId)
  // Do NOT call setSessionExits for this session
  return
}
// Existing behavior (non-zero only): record exit state, show ExitToast
const exitState: ExitState = { /* ... */ }
setSessionExits(prev => ({ ...prev, [data.sessionId]: exitState }))
// No countdown for non-zero exits — this matches the existing pattern where
// the countdown branch was already gated on `data.exitCode === 0`, so removing
// it leaves non-zero behavior unchanged.
```

Existing handleCloseTab adjacency logic (App.tsx ~L687-721) — we are RELYING on this, not changing it. The new test must assert that when handleCloseTab is called for the active tab, setActiveId is invoked with the adjacent tab id (or null when only one tab remained). This locks the SHELL-12 §"Focus shift on active-tab close" invariant.

Existing test patterns in App.exit.test.tsx — use as scaffolding. The test mocks EventsOn (`vi.mock('../../wailsjs/runtime/runtime', () => ({ EventsOn: ... }))`) and synthesizes session:exit events. Mirror its mock harness exactly.
</interfaces>

</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Branch session:exit handler on exitCode === 0 + RED-first vitest suite</name>
  <files>frontend/src/App.tsx, frontend/src/components/__tests__/App.shellExit.test.tsx</files>
  <behavior>
    Per UI-SPEC §4 SHELL-12 test contract — five assertions in the new test file:
    1. `session:exit` event with `exitCode: 0` does NOT add an entry to `sessionExits` state. (Verify by rendering ExitToast at the App's normal mount point and asserting nothing with role="alert"/the ExitToast container renders for the dispatched sessionId.)
    2. `session:exit` event with `exitCode: 0` calls `handleCloseTab` with the session id. (Verify by spying on KillSession — handleCloseTab calls KillSession — OR by asserting the tab is removed from the rendered tab list within waitFor.)
    3. `session:exit` event with `exitCode: 1` adds an entry to `sessionExits` state and does NOT call `handleCloseTab`. (Verify the ExitToast renders the session's name AND the tab remains in the rendered list.)
    4. `ExitToast` receives no entries for exit-code-0 sessions. (Implicit from #1; assert explicitly by querying for ExitToast's identifying element.)
    5. When the active tab closes, `activeId` shifts to the adjacent tab id (or `'welcome'` if it was the only non-Welcome tab). (Render with two tabs in pre-seeded state; dispatch exit-code-0 for the active tab; assert the other tab becomes active in the DOM — e.g., its `aria-selected="true"` or the active-tab class.)

    Implementation behavior — single targeted edit to App.tsx:
    - In the EventsOn('session:exit', ...) handler at line 537-595:
      * Insert at the top of the handler body (before the existing `const exitState: ExitState = ...`): an `if (data.exitCode === 0) { void handleCloseTabRef.current?.(data.sessionId); return }` early-return.
      * Remove the now-dead `countdown: data.exitCode === 0 ? 5 : -1` line — it can become a constant `-1` since the surviving branch only handles non-zero. Replace with `countdown: -1` for clarity.
      * Remove the entire `if (data.exitCode === 0) { ... }` countdown block (lines ~561-593) — unreachable after the early-return.
    - Do NOT touch: handleCloseTab body, autoCloseRef wiring (still useful for other code paths), ExitToast import, ExitToast render at ~L1236.
  </behavior>
  <action>
    (1) Create frontend/src/components/__tests__/App.shellExit.test.tsx using App.exit.test.tsx as the template. Same vi.mock structure for `'../../wailsjs/runtime/runtime'` (EventsOn handler capture), `'../../wailsjs/go/main/App'` (ListSessions/KillSession/etc. mocks), and any other modules App.tsx pulls.

       Test scaffolding:
       - Mock EventsOn to capture handlers per event name: `let handlers: Record<string, (data: any) => void> = {}; const EventsOn = vi.fn((name, fn) => { handlers[name] = fn; return () => {} })`.
       - Mock ListSessions to return one running shell session. Mock KillSession to resolve.
       - Render `<App />` and `await waitFor` until the tab list shows the session.
       - In each test, call `act(() => handlers['session:exit']({ sessionId: 'sess-1', exitCode: 0|1, sessionName: 'zsh', cli: 'shell', duration: 12, finalStatus: 'running' }))`.

       Write all five assertions. Run the test once and confirm it FAILS against the current App.tsx (proves the early-return branch is not yet in place).

    (2) Edit frontend/src/App.tsx session:exit handler at ~L537-595. Replace the handler body with the locked shape from UI-SPEC §2:
       ```typescript
       const offExit = EventsOn(
         'session:exit',
         (data: { sessionId: string; exitCode: number; sessionName: string; cli: string; duration: number; finalStatus: string }) => {
           // SHELL-12: clean exit (code 0) closes the tab immediately, no toast,
           // no countdown. The daemon (107-02) normalizes PTY -1 → 0 so we can
           // trust this branch covers all natural-exit cases including shell EOF.
           if (data.exitCode === 0) {
             void handleCloseTabRef.current?.(data.sessionId)
             return
           }
           // Non-zero exit: record state and show ExitToast (existing behavior).
           const exitState: ExitState = {
             sessionId: data.sessionId,
             sessionName: data.sessionName,
             cli: data.cli,
             exitCode: data.exitCode,
             duration: data.duration,
             finalStatus: data.finalStatus,
             countdown: -1,
             cancelled: false,
           }
           setSessionExits(prev => ({ ...prev, [data.sessionId]: exitState }))
         }
       )
       ```

       Note: deleting the existing countdown block also removes references to `countdownTimers.current[data.sessionId]` (the setInterval / clearInterval pair) inside that block. The `countdownTimers` ref is still defined and used by handleCloseTab (~L716-719) for cleanup. Leave that cleanup in place — it's a no-op when no timer was set, and removing it would break any future countdown reintroduction. If linting flags the now-unused autoCloseRef inside the listener scope, leave the import and ref intact (other code paths consume it; only the body usage at this site is removed). Run `pnpm typecheck` after the edit; fix any unused-import warnings only by leaving imports that are still referenced elsewhere.

    (3) Re-run the test file — all five assertions should pass. Also re-run App.exit.test.tsx to confirm no regression. (If App.exit.test.tsx had assertions specifically about exit-code-0 toast behavior, they will need updating — note that as a follow-up in the summary if encountered; per CONTEXT.md the existing exit-code-0 path is now considered a bug, so updating its assertions is correct.)
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub/frontend && pnpm test --run src/components/__tests__/App.shellExit.test.tsx src/components/__tests__/App.exit.test.tsx src/components/__tests__/ExitToast.test.tsx</automated>
  </verify>
  <done>
    Five new test assertions pass. App.exit.test.tsx still passes (any exit-code-0 assertions updated to reflect new behavior). ExitToast.test.tsx still passes (the component itself is unchanged). `grep -c "data.exitCode === 0" frontend/src/App.tsx` returns 1 (only the new early-return branch). The countdown block is gone — `grep -c "countdown: data.exitCode === 0" frontend/src/App.tsx` returns 0.
  </done>
</task>

</tasks>

<verification>
- `cd frontend && pnpm test --run` — full Vitest suite green.
- `cd frontend && pnpm typecheck` — no TypeScript errors. Any now-unused imports (e.g., if `countdownTimers` becomes write-only) flagged and either kept (still used by handleCloseTab cleanup) or removed surgically.
- `cd frontend && pnpm build` — clean production build (no dead-code-elimination warnings about the removed countdown block).
- Manual UAT after 107-01 + 107-02 + 107-03 also ship (dev-browser optional, not required for this plan): spawn a shell session, type `exit`, confirm the tab vanishes without any toast appearing. Spawn another shell session with a forced non-zero exit (e.g., `exit 2`) and confirm the existing ExitToast still appears.
</verification>

<success_criteria>
- "Critical invariant: Exit-code 0 ≠ exited with error" honored: the ExitToast never renders for exit-code 0, even briefly. The early-return pattern guarantees `setSessionExits` is never called for clean exits.
- "Critical invariant: Tab auto-close UX" honored: handleCloseTab's existing adjacency logic shifts focus left (or to Welcome). Asserted explicitly in test #5.
- No regression of the non-zero exit path: dispatching exitCode=1 still adds to sessionExits and renders ExitToast (test #3).
- Pairs cleanly with 107-02: daemon guarantees `data.exitCode` is the user-meaningful value (no leaking -1), so the frontend branch on `=== 0` is sound. The plan does NOT independently re-normalize -1 — that's daemon-side per CONTEXT.md §SHELL-12.
</success_criteria>

<output>
After completion, create `.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-04-SUMMARY.md` covering: the early-return branch added, the countdown block removed, the 5 new test assertions, any App.exit.test.tsx adjustments made, and confirmation that ExitToast.tsx is untouched. Flag for follow-up: the `autoCloseRef` / `countdownTimers` infrastructure remains in the file for future use; consider a hygiene cleanup pass if neither is referenced elsewhere after this phase ships.
</output>
