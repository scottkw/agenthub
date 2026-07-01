---
phase: 166-funnel-frontend-help-guide
plan: "05"
subsystem: frontend-funnel-share-surface
tags: [funnel, warm-up, share-panel, tdd, react, testing-md]
requires: [FunnelRiskPanel, funnel-enable-flow, hubpanel-share-sync, phase-166-css]
provides: [funnel-warmup-state-machine, internet-public-section, funnel-disable, testing-md-166]
affects:
  - frontend/src/components/Hub/SessionShareModal.tsx
  - frontend/src/components/SessionSharePanel.tsx
  - TESTING.md
tech-stack:
  added: []
  patterns: [warm-up-state-machine, poll-driven-effect, timer-ref-cleanup, presentational-section]
key-files:
  created: []
  modified:
    - frontend/src/components/Hub/SessionShareModal.tsx
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - frontend/src/components/SessionSharePanel.tsx
    - frontend/src/components/__tests__/SessionSharePanel.test.tsx
    - TESTING.md
key-decisions:
  - "Warm-up rides the existing 3s Hub poll (HubPanel shareModalSession sync from Plan 02) — no new 2s poll. Effect keyed on [session.funnelActive, funnelUrl]."
  - "Funnel URL is obtained by RE-ISSUING IssueCapabilities after funnelActive flips true (daemon swaps the cap base to the Funnel base once live — RESEARCH), stored as funnelUrl."
  - "30s warm-up timeout held in a single useRef, cleared on completion, disable, AND unmount (Pitfall 4). Disable clears the timer FIRST so a late fire cannot flip warmupTimedOut post-disable."
  - "Only the read-only Funnel URL is ever rendered in the Internet section — never a public write link (D-12). Test asserts the write cap token is absent from the section."
  - "Disable is single-click, no confirmation (D-13) — the persistent internet-exposure indicator is the compensating control."
  - "Effect gated on !funnelUrl so it fires exactly once AND also covers reopening a modal for an already-Funnel-active session."
requirements-completed: [FUI-04, FUI-05]
coverage:
  - deliverable: "Warm-up state after enable + IssueCapabilities re-issue on funnelActive flip → live URL"
    verification:
      - kind: test
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#re-issues IssueCapabilities and reveals the public URL when funnelActive flips true"
        status: pass
    human_judgment: false
  - deliverable: "30s warm-up timeout error"
    verification:
      - kind: test
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#surfaces the timeout error after 30s without funnelActive"
        status: pass
    human_judgment: false
  - deliverable: "One-click disable calls SetSessionFunnel(id, false, 0) + timer cleanup"
    verification:
      - kind: test
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#clears the 30s timeout on disable — no late timeout fire"
        status: pass
    human_judgment: false
  - deliverable: "Internet (public) read-only URL + Copy URL/Open/QR; no write link (D-12)"
    verification:
      - kind: test
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx#does NOT render a public write link in the Internet section (D-12)"
        status: pass
    human_judgment: false
  - deliverable: "TESTING.md manifest + traceability + M-37..M-40"
    verification:
      - kind: command
        ref: "bash tests/check-traceability-paths.sh (exit 0)"
        status: pass
    human_judgment: false
  - deliverable: "Real build gate — tsc && vite build (not just vitest)"
    verification:
      - kind: command
        ref: "cd frontend && npx tsc --noEmit && pnpm run build"
        status: pass
    human_judgment: false
  - deliverable: "No regressions across the frontend suite"
    verification:
      - kind: command
        ref: "cd frontend && npx vitest run — 134 files / 2252 tests pass"
        status: pass
    human_judgment: false
  - deliverable: "Live Funnel warm-up + off-tailnet URL (production build)"
    verification:
      - kind: manual
        ref: "TESTING.md M-37"
        status: pending
    human_judgment: true
  - deliverable: "Live auto-expiry teardown / indicators clear / local-fallback disable"
    verification:
      - kind: manual
        ref: "TESTING.md M-38, M-39, M-40"
        status: pending
    human_judgment: true
metrics:
  duration: "inline recovery"
  completed: "2026-06-30"
  tasks: 3
  files: 5
status: complete
---

# Phase 166 Plan 05: Funnel Share Surface Completion Summary

Completes the Funnel share surface: the warm-up UX after enable, the "Internet (public)" read-only URL section (copy + open + QR) revealed once TLS/`funnelActive` is live, one-click disable, and the TESTING.md regression-convention updates + real build gate.

## Execution note

Executed **inline by the orchestrator** after the spawned `gsd-executor` stalled on the same Claude API SSE stream-idle watchdog that hit Plan 166-02 (intermittent #2410 idle timeout — the executor made zero commits and left no partial work). Inline execution preserved full TDD (RED → GREEN) and atomic commits. The panel (Task 2) was implemented before the modal (Task 1) because the modal's warm-up DOM assertions render through the panel's Internet section; commits are ordered accordingly.

## Tasks Completed

3 / 3

## Accomplishments

- **Task 2 — SessionSharePanel Internet (public) section:** New props `funnelActive` / `funnelUrl` / `warmingUp` / `warmupTimedOut` / `onDisableFunnel`. Renders `.hub-share-internet-section` when the Funnel is engaged: "Starting up… (TLS warming up)" during warm-up; "Public URL (read-only):" + the funnelUrl + Copy URL (`ClipboardSetText`, 1500ms reset) / Open (`BrowserOpenURL`) / QR (`GetCapabilityQRCode(funnelUrl)`) once live; "Connection timed out. Try disabling and re-enabling." on timeout; and a single-click "Disable internet share". Only the read-only URL is ever shown — never a write link (D-12). 17/17 panel tests.
- **Task 1 — SessionShareModal warm-up state machine:** Added `warmupTimedOut` + `funnelUrl` state and a `warmupTimeoutRef`. `handleFunnelEnable` now arms a 30s timeout after `SetSessionFunnel`. A `useEffect` keyed on `[session.funnelActive, funnelUrl]` re-issues `IssueCapabilities` when the daemon flips `funnelActive` true (delivered by Plan 02's HubPanel poll-sync), stores the Funnel-base `readUrl` as `funnelUrl`, clears warm-up, and clears the timer. `handleDisableFunnel` clears the timer first, then calls `SetSessionFunnel(id, false, 0)` (FNL-05 teardown) and resets all funnel state. An unmount effect clears any pending timer (Pitfall 4). All five funnel props are threaded to the rendered SessionSharePanel. 31/31 modal tests (incl. fake-timer timeout via `React.act`).
- **Task 3 — TESTING.md + build gate:** Suite Manifest updated (vitest 132→134, total 511→513) with a Phase-166 note; Section 4 traceability rows for FUI-01..06 + HLP-01/02 (repo-relative `.tsx` paths only); Section 5 manual items M-37 (live warm-up + off-tailnet URL, production build), M-38 (auto-expiry teardown), M-39 (indicators appear/clear), M-40 (local-fallback disable). `check-traceability-paths.sh` exits 0.

## Verification Results

| Check | Result |
|-------|--------|
| `pnpm test -- SessionSharePanel` | 17/17 PASS |
| `pnpm test -- SessionShareModal` | 31/31 PASS |
| `npx tsc --noEmit` | CLEAN |
| `pnpm run build` (vite production) | ✓ built |
| `bash tests/check-traceability-paths.sh` | OK (exit 0) |
| Full frontend suite (`vitest run`) | 134 files / 2252 tests PASS (no regressions) |

## Commits

| Hash | Message |
|------|---------|
| `2583a29f` | test(166-05): add failing Internet (public) section contract (RED) |
| `9c42f7c7` | feat(166-05): add Internet (public) section to SessionSharePanel (GREEN) |
| `75c49238` | feat(166-05): add warm-up state machine + disable to SessionShareModal (GREEN) |
| `49ff9b3c` | docs(166-05): register Phase-166 tests + M-37..M-40 in TESTING.md |

## Deviations from Plan

**1. Inline execution + panel-before-modal order** — see Execution note. No scope change; TDD and atomic commits honored.

**2. Fake-timer test uses `React.act`** — the 30s warm-up timeout DOM assertion needed `await React.act(async () => vi.advanceTimersByTimeAsync(...))` because React 19's scheduler commits via MessageChannel (not faked by vitest fake timers), so `flushSync` alone did not flush the timer-driven state update. This is the standard React-18/19 fake-timer pattern, not a behavioral change.

## Threat Flags

- **T-166-10 (Info Disclosure, mitigated):** Internet section renders only the read-only Funnel `readUrl`; no public write link. Test asserts the write cap token is absent from the section (D-12).
- **T-166-11 (DoS, mitigated):** Warm-up rides the existing 3s poll with a 30s timeout + error; no busy-loop; timer cleared on success/disable/unmount (Pitfall 4).
- **T-166-12 (Tampering, mitigated):** Disable calls `SetSessionFunnel(id,false,0)` → Phase-165 four-path teardown; indicator reconciles on next poll.

## Known Stubs

- **Already-active-on-open URL:** the warm-up effect fires for a reopened modal whose session is already Funnel-active (gated on `!funnelUrl`), so the live URL is fetched then too — not a stub, but noted since Plan 02 seeds `funnelOn` from `session.funnelActive` without fetching the URL.
- **Funnel usable only alongside web-share:** the Internet section lives inside SessionSharePanel, which the modal renders only when "Share the session" is ON and caps are cached — matching the daemon's requirement that Funnel exposes an already-running web share. Enabling Funnel with web-share OFF is not a supported flow (consistent with the Plan 02/05 design).

## Manual Verification Required

M-37..M-40 (live, production build) — warm-up completes + off-tailnet URL opens (M-37), auto-expiry tears down (M-38), indicators appear/clear (M-39), local-fallback disables the toggle (M-40). These require a live tailnet + Funnel grant + an off-tailnet device and cannot be automated.

## Next Step

Phase 166 execution complete (all 5 plans). Ready for phase verification (`/gsd-verify-work 166`) and the live M-37..M-40 UAT on a production build.

## Self-Check: PASSED

- [x] `SessionSharePanel.tsx` Internet section (URL/copy/open/QR/warmup/timeout/disable) — 17/17
- [x] `SessionShareModal.tsx` warm-up state machine + disable + timer cleanup — 31/31
- [x] `TESTING.md` manifest + traceability + M-37..M-40; check-traceability-paths.sh OK
- [x] Commits `2583a29f`, `9c42f7c7`, `75c49238`, `49ff9b3c` all in git log
- [x] `tsc --noEmit && pnpm run build` succeeds (real build gate)
- [x] Full frontend suite green (2252 tests) — no regressions
- [x] No public write link in the Internet section (D-12) test-asserted
