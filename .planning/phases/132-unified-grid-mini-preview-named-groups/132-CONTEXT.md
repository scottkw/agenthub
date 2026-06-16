# Phase 132: Unified Grid + Mini Preview + Named Groups - Context

**Gathered:** 2026-06-16
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Users can see throttled terminal output snapshots on every card, remote sessions alongside local ones, and organize sessions into named groups via a group sidebar.

Depends on: Phase 131 (cards, grid, HubPanel).

Requirements: CARD-07, GRID-03, GRID-07, GROUP-01, GROUP-02, GROUP-03, GROUP-04.

Locked design decisions (from v3.6 STATE.md Key Decisions — treat as binding):
- Mini preview = throttled snapshot of the session's recent output tail. NEVER a live xterm per card. Performance constraint is non-negotiable (CARD-07).
- Group membership key = session name + working directory. Survives session-id churn across restarts; unmatched sessions fall to a default lane (GROUP-04).
- Remote sessions reuse the locked Phase 122 design (daemon proxy + join-code exchange). No new remote-access architecture.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, the locked decisions above, and codebase conventions (extend the Phase 131 Hub components: SessionCard, SessionCardGrid, HubPanel; reuse existing remote-session plumbing).

</decisions>

<code_context>
## Existing Code Insights

Builds directly on Phase 131 Hub components (frontend/src/components/Hub/). Codebase context gathered during plan-phase research. Note: Phase 131 added a re-attach "Open" button (handleOpenSessionTab) on cards — preserve it.

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond ROADMAP success criteria — discuss phase skipped.

</specifics>

<deferred>
## Deferred Ideas

Structured "agent suggests options" multi-select (#78) is deferred to #93 — not this phase.

</deferred>
