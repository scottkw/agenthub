# Phase 128: Remote Write Parity + Cross-Surface Integration - Context

**Gathered:** 2026-06-15
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Remote tailnet peer write operations (edit/save, upload, delete, rename, mkdir) work end-to-end from both the desktop GUI and the TUI, with write parity proven by 3 independent network-stack observers — mirroring the Phase 122 read-parity proof pattern. The milestone ships with a two-machine UAT checklist ready and no regression on Phase 122 remote read tests.

**Depends on:** All previous phases (123-127) complete. FSW-10 (TD-5) fixed in Phase 123 (ExchangeJoinCodeAtURL 303 parse) is a direct prerequisite — without it, the desktop GUI cannot acquire a remote cap.

**Requirements:** RMW-01, RMW-02, RMW-03, RMW-04, RMW-05, RMW-06

**This is the FINAL phase of v3.5 — closes umbrella Issue #24 when the two-machine UAT is executed.**
</domain>

<decisions>
## Implementation Decisions

### Locked (from ROADMAP success criteria)
- SC1: remote write parity proven by 3 independent observers — daemon-proxy Go, tui.RemoteFilesClient Go, and Playwright HTTPS browser — all byte-equivalent for a write-then-read round trip against the same remote session (mirror Phase 122 read-parity proof).
- SC2: desktop GUI does edit/save, upload, delete, rename, cross-dir move, mkdir on a remote tailnet peer session via the daemon proxy; TUI does the same minus upload via RemoteFilesClient over HTTPS (TLS 1.2+ pinned, cap token redacted from error messages).
- SC3: write against a v3.4 remote peer (no write endpoints) → HTTP 405 + client surfaces "The remote session is running an older version of AgentHub that does not support file writes." (not a generic network error or opaque 405).
- SC4: remote cap expires mid-edit → editor buffer preserved + "access expired" message; orphaned partial upload on remote cleaned up — no silent buffer loss, no stranded temp files.
- SC5: Phase 122 remote read test suite passes with zero regressions; a two-machine tailnet write UAT checklist committed (Machine A web-share + Machine B GUI + Machine B TUI; cross-surface write parity + cap-expiry failure mode) — closes Issue #24 when executed.

### Claude's Discretion
The 3-observer parity test harness structure (mirror Phase 122); the 405-detection + friendly-message wiring; the cap-expiry mid-edit handling; the UAT checklist format.
</decisions>

<code_context>
## Existing Code Insights

Gathered during plan-phase research. Anchors: the Phase 122 read-parity proof harness (3 observers — find it), the daemon remote-files write proxy (124 CAP-10), tui.RemoteFilesClient write methods (126-01), the ExchangeJoinCodeAtURL 303 fix (123/TD-5), the GUI remote-write path (124-04 desktop), the existing remote read tests (122), and the prior two-machine UAT checklist precedent (122).

</code_context>

<specifics>
## Specific Ideas

Refer to ROADMAP Phase 128 success criteria. This is largely INTEGRATION + PARITY-PROOF + a committed UAT checklist — most write plumbing exists (123-127); this phase proves it works remotely end-to-end across surfaces, handles the v3.4-peer 405 + cap-expiry failure modes, and confirms zero regression on Phase 122 remote reads. The two-machine UAT is a committed checklist executed by the operator (closes Issue #24).

</specifics>

<deferred>
## Deferred Ideas

The two-machine tailnet UAT execution itself is operator-run (requires 2 physical machines) — the deliverable is the committed checklist + the automated 3-observer parity proof.

</deferred>
