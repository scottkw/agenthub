# Phase 129: Write Concurrency Fix + DNS Error UX - Context

**Gathered:** 2026-06-15
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss) + resolved open design decision

<domain>
## Phase Boundary

The release gate passes deterministically and users see actionable errors when Tailscale DNS prerequisites are missing.

Two self-contained fixes (independent of Phase 130):
1. **Write concurrency (#87):** Fix the flaky `TestWrite_TwoWritersIfMatchRace` that gates releases — the `WriteFileAtomic` If-Match contract must be deterministic, consistent across code/comments/proxy, and never produce interleaved content or leftover temp files.
2. **DNS error UX (#83):** When a remote browse fails because the client has `accept-dns=false` (unresolvable MagicDNS), surface a specific actionable message instead of an opaque 502; distinguish this case from other remote-unreachable failures; probe `accept-dns` state proactively.

Requirements: RACE-01, RACE-02, RACE-03, DNS-01, DNS-02, DNS-03.

</domain>

<decisions>
## Implementation Decisions

### RESOLVED — #87 If-Match concurrency contract (user decision, 2026-06-15)

**(a) Per-path lock → true single-winner guarantee.**

- Serialize concurrent writes to the same path so exactly one writer wins.
- The losing writer receives a clean conflict (412 Precondition Failed / If-Match mismatch), not a silent overwrite.
- This matches standard If-Match optimistic-concurrency semantics: the second writer's stale ETag must fail.
- **RACE-02 consequence:** code, inline comments, AND the remote-write proxy must all assert the single-winner contract. Any lingering "last-writer-wins (WR-02)" comments must be corrected to reflect single-winner — no mismatch between the asserted guarantee and the documentation.
- `TestWrite_TwoWritersIfMatchRace` must assert the single-winner outcome and pass 100/100 (no goroutine-scheduling dependence).

### Claude's Discretion (DNS UX + mechanics)

DNS error-UX implementation choices (where the `accept-dns` probe lives, exact message wording within the actionable-message intent, how the unresolvable-MagicDNS condition is detected vs other failures) are at Claude's discretion guided by ROADMAP success criteria and codebase conventions. Determine during plan-phase research.

</decisions>

<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research. Known touch-points from prior milestone work:
- `WriteFileAtomic` and the If-Match path live in the FS write primitives (Phase 123, FSW-*).
- The remote-write proxy (Phase 128, RMW-*) must agree with the chosen contract.
- DNS / remote-unreachable failures surface through the daemon's remote-browse path; relay-loopback surface (`internal/relay/`, `internal/daemon/relay_remote_files_test.go`) is the GUI's path and must be exercised by tests.

</code_context>

<specifics>
## Specific Ideas

- The release `validate` gate is the deterministic-pass target for RACE-01.
- DNS message intent: name the fix, e.g. "Enable Tailscale DNS (accept-dns) to browse remote sessions."
- The actionable message must surface only when the `accept-dns=false` / unresolvable-MagicDNS condition is actually the cause (DNS-02) — not as a blanket catch-all.

</specifics>

<deferred>
## Deferred Ideas

None for this phase.

</deferred>
