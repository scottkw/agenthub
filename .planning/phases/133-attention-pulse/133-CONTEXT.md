# Phase 133: Attention + Pulse - Context

**Gathered:** 2026-06-16
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Sessions needing attention float to the top and pulse visibly, with debounced non-jarring reordering, so users can identify blocked or errored sessions at a glance without relying on color.

Depends on: Phase 131 (cards), Phase 132 (grid + groups).

Requirements: ATTN-01, ATTN-02, ATTN-03, ATTN-04, ATTN-05, ATTN-06.

Locked constraints (binding):
- The user is COLORBLIND (release-blocking): attention must be conveyed by a pulsing animated border + a distinct attention icon + position (float-to-top), NEVER color alone.
- Pulse animation MUST respect prefers-reduced-motion (a non-animated attention treatment when reduced motion is requested), consistent with the Phase 131/132 reduced-motion guard pattern.
- Float-to-top reordering MUST be debounced (not on every poll tick) and position changes animate smoothly (no jarring jumps).

Sequencing note for ATTN-03 (modal-resolve clears attention):
- The Phase 134 modal does NOT exist yet. ATTN-03's clearing is STATUS-DRIVEN: attention state derives from session status (waiting / errored / non-zero exit); when the poll reports the status no longer needs attention, the pulse + attention state clear automatically — no page reload. The Phase 134 modal will simply be one trigger that causes that status change. Implement and test ATTN-03 as status-change-driven clearing (drivable now without the modal); do NOT couple it to a modal that doesn't exist yet.
</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Extend the Phase 131/132 Hub components (SessionCard attention treatment, SessionCardGrid float-to-top ordering, GroupSidebar attention badge on collapsed groups, hubStatus derivation). Reuse the established `--hub-*` tokens, reduced-motion guard, and colorblind-safe icon+text pattern.

</decisions>

<code_context>
## Existing Code Insights

Builds on Phase 131 (SessionCard, hubStatus.ts deriveHubStatus — 'waiting'/'errored'/'stopped-err' statuses already exist) and Phase 132 (SessionCardGrid named/workDir grouping, GroupSidebar with needs-input badge + per-group counts). Preserve the Phase 131 Open button and Phase 132 group/preview features. Codebase context gathered during plan-phase research.

</code_context>

<specifics>
## Specific Ideas

No specifics beyond ROADMAP success criteria — discuss skipped.

</specifics>

<deferred>
## Deferred Ideas

None — discuss skipped.

</deferred>
