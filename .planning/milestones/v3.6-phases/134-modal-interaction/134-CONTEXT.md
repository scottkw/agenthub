# Phase 134: Modal Interaction - Context

**Gathered:** 2026-06-17
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Clicking any Hub card opens a full interactive or briefing modal with a shared-element grow/shrink animation, and closing it returns focus cleanly to the originating card.

Success criteria (what must be TRUE):
1. Clicking a non-blocked session card expands it into a modal via a grow animation from the card's position; the modal mounts a full interactive TerminalPanel with the same RelayClient used by normal tabs — resize, copy/paste, and scrollback search all work.
2. Clicking a `waiting`/needs-input session card opens a briefing modal showing the real terminal tail (the prompt the agent printed) with a respond affordance; typing a response and submitting sends it to the session.
3. Closing any modal (Escape, close button, or clicking outside) plays a shrink-back animation and returns keyboard focus to the originating card.
4. For a remote session that requires a capability token, the modal interaction uses the existing Phase 122 join-code exchange path — no new remote-access architecture (MODAL-06).

Requirements: MODAL-01..06.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

### Locked decisions carried from STATE.md / prior phases
- **Briefing modal data source:** Driven by real terminal tail (actual prompt the agent printed). Structured "agent suggests options" multi-select (#78) is deferred to #93 — agents don't emit that data today.
- **Remote modal interaction:** Reuses locked Phase 122 design (daemon proxy + join-code exchange). No new remote-access architecture (MODAL-06).
- **Re-attach Open button must be preserved:** Phase 131 added an "Open" button on Hub cards + Sessions rows (commit 08fc2be). The card-click→modal interaction must COEXIST with that button, not regress it (see memory `project_phase134_reattach_button`).
- **Colorblind-safe constraint** (user is colorblind): any status/affordance conveyed in the modal must carry non-color cues. Release-blocking; verify at source level (hex constants), not by eye. Full a11y validation is Phase 135.

</code_context>
<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research. Key anchors to research:
- Existing `TerminalPanel` + `RelayClient` used by normal session tabs (must be reused, not reimplemented).
- Hub card component (`SessionCard`), grid (`SessionCardGrid`), and `HubPanel` from Phases 131-133.
- Attention/`deriveStatus` logic from Phase 133 (drives modal type: interactive vs briefing).
- Phase 122 remote join-code/cap exchange path.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — discuss phase skipped. Refer to ROADMAP phase description and success criteria above.

</specifics>

<deferred>
## Deferred Ideas

- Structured "agent suggests options" multi-select in briefing modal → deferred to issue #93.

</deferred>
