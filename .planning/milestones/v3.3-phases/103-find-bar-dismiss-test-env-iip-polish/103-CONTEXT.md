---
phase: 103-find-bar-dismiss-test-env-iip-polish
type: context
status: ready
mode: auto-generated
---

# Phase 103: Find Bar Dismiss + Test-Env + IIP Polish

**Gathered:** 2026-05-12
**Mode:** Auto-generated (REQUIREMENTS pre-answers grey areas)

<domain>
## Phase Boundary

Close 4 remaining v3.2 polish items that are not in the link path:

- **POLISH-03**: find-bar focus/event-propagation dismiss bug after case-sensitive toggle click
- **POLISH-04**: find-bar exit animation parity (200ms slide-out on Esc + close-button dismiss, same as entry)
- **POLISH-05**: iTerm2 IIP (OSC 1337) rendering investigation+decision — implement rendering OR explicitly document sixel-only support
- **POLISH-06**: Vitest 4 + jsdom 29 `localStorage` test-env regression (Sidebar.test.tsx 20 currently-failing tests)

**Note on POLISH-06**: A `frontend/src/test-setup.ts` localStorage polyfill was added in Phase 101-02 (Plan 101-02 deviation #2). All 20 Sidebar.test.tsx tests now pass (verified 2026-05-12). The persistent `ExperimentalWarning: localStorage is not available because --localstorage-file was not provided` line in vitest output is a Node 26 warning from `node:storage` API, not a test failure; the test-setup polyfill provides the global. Phase 103 may just need to document this and confirm.

**Note on POLISH-05**: This is an investigate-and-decide requirement. Acceptable outcomes:
1. Implement iTerm2 IIP rendering (likely substantial), OR
2. Document explicitly that AgentHub supports sixel only and IIP is out of scope.
Decision lives in a 103-IIP-DECISION.md or in the phase SUMMARY.
</domain>

<decisions>
## Implementation Decisions — Claude's Discretion

- **POLISH-03 fix**: Likely root cause is focus moving to a toggle button without an onKeyDown handler that bubbles Esc. Defensive fix: attach a `document.addEventListener('keydown')` for Escape that the FindBar component owns while mounted — OR ensure the container's `onKeyDown={handleContainerKeyDown}` continues to fire after toggle click. Investigate first, fix simplest path.
- **POLISH-04 exit animation**: FindBar already supports `exiting` prop and `.find-bar--exiting` class. The bug is likely that `onClose` immediately unmounts the bar; need to introduce a parent-side "exiting" intermediate state with `setTimeout(unmount, 200)`. Mirror in `web/assets/terminal.js`.
- **POLISH-05 decision (recommendation)**: Document sixel-only support. iTerm2 IIP is a major implementation lift (base64 decode → blob URL → HTML img in xterm.js custom renderer); not in the milestone scope per "Anti-Goals". Write a 103-IIP-DECISION.md that explicitly states this.
- **POLISH-06**: Confirm test-setup.ts polyfill in place; document the persistent Node warning as benign.

</decisions>

<code_context>
## Existing Code Insights

- `frontend/src/components/FindBar/FindBar.tsx` — accepts `exiting` prop, applies `.find-bar--exiting` CSS class
- `frontend/src/components/TerminalPanel.tsx` — parent owns FindBar mount/unmount
- `frontend/src/test-setup.ts` — Phase 101-02 polyfill (localStorage + sessionStorage)
- `frontend/vite.config.ts` — wires setupFiles
- Sidebar.test.tsx — 20 tests already passing as of Phase 101-02 (verified)

</code_context>

<specifics>
## Specific Ideas

1. **POLISH-03** — write RED test simulating click-toggle-then-Esc on the dismiss test file. Fix in FindBar.tsx — likely add a `keydown` document listener for Escape while mounted (with cleanup on unmount).
2. **POLISH-04** — TerminalPanel introduces `findBarExiting` state. `handleSearchClose` sets exiting=true and schedules `setTimeout(actualUnmount, 200)`. FindBar receives `exiting` prop and renders with `.find-bar--exiting`. Mirror in web/assets/terminal.js.
3. **POLISH-05** — write 103-IIP-DECISION.md: "AgentHub supports sixel only. iTerm2 IIP is out-of-scope per milestone Anti-Goals. Plugin documentation updated to reflect."
4. **POLISH-06** — write 103-06-NOTES.md confirming Sidebar tests pass; document the Node warning as benign.
</specifics>

<deferred>
## Deferred

- Full IIP implementation (deferred — see POLISH-05 decision)
- Sixel/IIP feature-parity in web (not in scope this milestone)
</deferred>
