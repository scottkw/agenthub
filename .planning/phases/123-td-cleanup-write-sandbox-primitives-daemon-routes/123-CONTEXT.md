# Phase 123: TD Cleanup + Write Sandbox Primitives + Daemon Routes - Context

**Gathered:** 2026-06-14
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

The `internal/files/` sandbox has all write primitives (atomic write, rename, delete, mkdir, upload), the shell-RC denylist is enforced on all write paths, the two carried tech-debts (TD-4 and TD-5) are closed, and the daemon local-socket write routes are live — so every subsequent phase has a correct, trusted, fuzz-proven write API to build against.

**Depends on:** Phase 122 (v3.4 complete). No v3.5 prerequisites. Load-bearing security foundation — must land before any write endpoint is exposed on any surface.

**Requirements:** FSW-01, FSW-02, FSW-03, FSW-04, FSW-05, FSW-06, FSW-07, FSW-08, FSW-09, FSW-10, FSW-11, FSW-12

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

</decisions>

<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — discuss phase skipped. Refer to ROADMAP phase description and success criteria.

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
