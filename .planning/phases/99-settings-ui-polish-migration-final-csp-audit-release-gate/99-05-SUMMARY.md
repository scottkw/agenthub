---
phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate
plan: 05
status: task-1-complete-task-2-pending-human
requirements: []
---

# 99-05 SUMMARY — iPad Safari Tailscale UAT runbook (SC-4)

## What was built (Task 1 — autonomous)

Authored the 5-scenario manual UAT runbook at `.planning/phases/99-settings-ui-polish-migration-final-csp-audit-release-gate/99-iPad-UAT.md`. The runbook clones the Phase 93 iPad UAT shape verbatim (header, prereqs, numbered steps, blockquoted verbatim copy, sub-checks, PASS criteria, sign-off block) and extends it with the Phase 99 release-gate audits:

- **UAT-1** — All-plugins-enabled attach → render flow on iPad Safari over Tailscale (default state: 7 ON / Progress OFF). Includes the chafa-piped sixel image for IIP/sixel render verification.
- **UAT-2** — Scrollback → detach → re-attach → confirm scrollback intact (multi-client byte-fidelity, IMG-04).
- **UAT-3** — Zero-CDN audit via Safari Web Inspector remote debugging from dev Mac. Network tab filter `cdn.jsdelivr.net OR unpkg.com OR esm.sh OR fonts.googleapis.com` — must match zero requests. Screenshot to `screenshots/99-iPad-UAT-3-zero-cdn.png`.
- **UAT-4** — CSP zero-violation audit via Web Inspector Console. Zero messages matching `/csp|content security policy|refused to (execute|load|connect)/i`. Screenshot to `screenshots/99-iPad-UAT-4-zero-csp.png`.
- **UAT-5** — Second pass with ALL-8-ON (Progress flipped ON via Settings → Plugins → Save Plugins). Verifies progress underline + tray icon aggregate + re-runs UAT-3/UAT-4 with the all-8-ON state.

Sign-Off block lists 5 checkboxes (one per UAT) plus a final aggregation referencing `99-VALIDATION.md` and `/gsd-verify-work 99`. The bottom of the runbook has a Tester / Device / Date line for the human to fill in.

## Why a real iPad (not iOS Simulator)

ROADMAP SC-4 mandates real-device UAT verbatim. Safari WebKit on iOS has subtle CSP / WebAssembly / network differences from desktop WebKit that the simulator hides. The runbook calls this out explicitly in the prereqs.

## Pending: Task 2 — checkpoint:human-verify

Task 2 is a human-execution checkpoint. The runbook is authored and ready to run; the orchestrator cannot execute it because real-hardware UAT cannot be automated. The human runs the 5 scenarios on a real iPad over Tailscale, checks off all 5 sign-off boxes (`[x]`), commits the 2 screenshots under `screenshots/`, fills in the Tester / Device / Date line, and replies "approved" to resume.

**Resume signal:** the orchestrator advances to phase verification once all 5 UAT checkboxes are `[x]`, both screenshots exist, and the Tester / Device / Date line is filled in.

## Key files

- `.planning/phases/99-settings-ui-polish-migration-final-csp-audit-release-gate/99-iPad-UAT.md` (new, 124 lines) — the manual runbook for human execution.

## Commits

- `2113978` docs(99-05): author iPad Safari Tailscale UAT runbook (SC-4)

## Verification (Task 1 only)

- File exists.
- 5× `## UAT-` sections.
- 1× `## Sign-Off` block.
- "iOS Simulator is NOT a substitute" appears once.
- `99-VALIDATION.md` and `/gsd-verify-work 99` both referenced.
- Web Inspector procedure documented (Network tab CDN filter + Console tab CSP filter).
- 124 lines (>= 80 plan threshold).

## Self-Check: PASSED (Task 1) — Task 2 awaits human

Task 1 acceptance criteria fully satisfied. Task 2 checkpoint surfaced to user; no further orchestrator work until the human completes the UAT and signs off.

## Notes

This plan was executed inline by the orchestrator (Opus) rather than spawned as a Sonnet subagent — the plan's content is verbatim and the file write is straightforward, so no model handoff was needed. Sonnet daily rate limit had been hit during Wave 1; running inline avoided the wait.
