---
phase: 166-funnel-frontend-help-guide
plan: "02"
subsystem: frontend-funnel-enable-flow
tags: [funnel, share-modal, risk-panel, tdd, react, colorblind-safe]
requires: [SetSessionFunnel-stub, funnelActive-field, phase-166-css]
provides: [FunnelRiskPanel, funnel-enable-flow, sharemodal-funnel-toggle, hubpanel-share-sync]
affects:
  - frontend/src/components/Hub/FunnelRiskPanel.tsx
  - frontend/src/components/Hub/SessionShareModal.tsx
  - frontend/src/components/Hub/HubPanel.tsx
tech-stack:
  added: []
  patterns: [presentational-subcomponent, two-step-commit-gesture, fail-closed-local-fallback, poll-sync-effect]
key-files:
  created:
    - frontend/src/components/Hub/FunnelRiskPanel.tsx
    - frontend/src/components/__tests__/FunnelRiskPanel.test.tsx
  modified:
    - frontend/src/components/Hub/SessionShareModal.tsx
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - frontend/src/components/Hub/HubPanel.tsx
key-decisions:
  - "Toggle ON never calls SetSessionFunnel — it opens the risk panel; only the explicit 'Enable internet share' CTA commits (D-02/FUI-01). Tests assert the two-step gesture."
  - "Local-fallback fails closed: webServerMode !== 'tailscale' disables the input (HTML disabled + aria-disabled + pointerEvents none) and blocks the binding (D-15)."
  - "FunnelRiskPanel is presentational (no Wails calls); it exposes only callbacks (onEnable/onCancel/onExpiryChange/onOpenHelp). Reuses existing accent CTA class (.link-confirm-popover__btn--continue) and ghost class (.hub-share-internet-section__disable)."
  - "warmingUp is a placeholder surfaced as data-funnel-warming on the toggle section; Plan 05 consumes it for the warm-up reveal."
  - "onOpenHelp is an OPTIONAL modal prop; the risk panel's Help link closes the modal then delegates. App-level tab navigation is intentionally NOT wired here (App.tsx is owned by Plan 166-03) — a deliberate seam."
  - "HubPanel sync effect keyed on [sessions, shareModalSession?.id] so its own setShareModalSession cannot re-trigger it (RESEARCH Pitfall 3)."
requirements-completed: [FUI-01, FUI-02, FUI-06]
coverage:
  - deliverable: "FunnelRiskPanel: risk statement + 5 expiry presets (3600 default) + Help link + actions"
    verification:
      - kind: test
        ref: frontend/src/components/__tests__/FunnelRiskPanel.test.tsx
        status: pass
    human_judgment: false
  - deliverable: "Toggle ON opens risk panel and does NOT call SetSessionFunnel (FUI-01/D-02)"
    verification:
      - kind: test
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#flipping the Funnel toggle ON opens the risk panel and does NOT call SetSessionFunnel"
        status: pass
    human_judgment: false
  - deliverable: "Explicit CTA commits SetSessionFunnel(id, true, expirySeconds) with the selected preset (FUI-02)"
    verification:
      - kind: test
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#changing the expiry preset commits the selected value"
        status: pass
    human_judgment: false
  - deliverable: "Local-fallback disables the toggle and blocks the binding (D-15)"
    verification:
      - kind: test
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#with webServerMode=\"local\" the Funnel toggle is disabled and SetSessionFunnel is never called"
        status: pass
    human_judgment: false
  - deliverable: "HubPanel keeps the modal session prop synced to the 3s poll"
    verification:
      - kind: command
        ref: "cd frontend && npx tsc --noEmit"
        status: pass
    human_judgment: false
  - deliverable: "No regressions across the frontend suite"
    verification:
      - kind: command
        ref: "cd frontend && npx vitest run — 134 files / 2228 tests pass"
        status: pass
    human_judgment: false
metrics:
  duration: "inline recovery"
  completed: "2026-06-30"
  tasks: 3
  files: 5
status: complete
---

# Phase 166 Plan 02: Funnel Enable Flow Summary

Wires the Funnel (internet share) enable flow into the Hub Share modal: a presentational `FunnelRiskPanel`, the toggle + two-step commit gesture inside `SessionShareModal`, the local-fallback fail-closed state, the Help cross-link seam, and the `HubPanel` sync effect that keeps the modal's session prop tracking the 3s poll (prerequisite for Plan 05's warm-up).

## Execution note

This plan was executed **inline by the orchestrator** rather than in a spawned executor subagent. Two consecutive `gsd-executor` dispatches stalled on the Claude API SSE stream-idle watchdog ("no progress for 600s", the known #2410 idle-timeout under large subagent context) — the first made zero commits, the second only left one orphan uncommitted test file (removed before re-execution). The frontend test script is `vitest run` (non-watch), so a hung test command was ruled out as the cause. Inline execution sidestepped the flaky subagent stream; full TDD discipline (RED → GREEN per task) and atomic commits were preserved.

## Tasks Completed

3 / 3

## Accomplishments

- **Task 1 — FunnelRiskPanel (`FunnelRiskPanel.tsx`):** New presentational component. Renders the verbatim risk statement (FUI-01), an "Auto-expire:" `<select>` with the fixed preset enum `{1800, 3600, 14400, 28800, 0}` defaulting to 3600 (FUI-02, D-05/D-07 — no custom-minutes input), a "Want tighter containment? See the Sharing Guide →" Help cross-link (FUI-06), and "Keep local only" / "Enable internet share" actions in the mandated Tab order (expiry → help → cancel → CTA). No Wails call — commit lives in the modal (D-02). 10/10 tests.
- **Task 2 — SessionShareModal wiring:** Added `SetSessionFunnel` import, `funnelActive: boolean` on `ShareSession`, an "Enable internet sharing" toggle following the existing browse-toggle markup, and `riskPanelOpen` / `expirySeconds` (default 3600) / `funnelError` / `warmingUp` state. Toggle ON → `setRiskPanelOpen(true)` (never a binding call); the CTA → `SetSessionFunnel(session.id, true, expirySeconds)` in try/catch (inline `--hub-destructive` error on failure, toggle stays OFF); "Keep local only" collapses with no call. Fails closed when `webServerMode !== 'tailscale'` (disabled input + `aria-disabled` + `pointerEvents:none` + "Internet sharing requires Tailscale" note). Help link closes the modal then calls `onOpenHelp`. 26/26 modal tests.
- **Task 3 — HubPanel sync effect:** `useEffect` keyed on `[sessions, shareModalSession?.id]` that re-sets `shareModalSession` from the live `sessions` prop when the modal is open, early-returning while closed. Propagates `funnelActive` flips into the modal so Plan 05's warm-up can complete (RESEARCH Pitfall 3).

## Verification Results

| Check | Result |
|-------|--------|
| `pnpm test -- FunnelRiskPanel` | 10/10 PASS |
| `pnpm test -- SessionShareModal` | 26/26 PASS (19 prior + 7 new) |
| `npx tsc --noEmit` | CLEAN |
| Full frontend suite (`vitest run`) | 134 files / 2228 tests PASS (no regressions) |

## Commits

| Hash | Message |
|------|---------|
| `57022812` | test(166-02): add failing FunnelRiskPanel contract (RED) |
| `f453b26d` | feat(166-02): add FunnelRiskPanel component (GREEN) |
| `3694bdfc` | test(166-02): add failing Funnel enable-flow contract to modal (RED) |
| `72d4ec5f` | feat(166-02): wire Funnel toggle + risk panel into SessionShareModal (GREEN) |
| `e589990a` | feat(166-02): add HubPanel shareModalSession sync effect |

## Deviations from Plan

**1. Inline execution instead of subagent** — see Execution note above. No scope change; the plan's task breakdown, TDD contract, and atomic-commit protocol were all honored.

**2. CTA/ghost button classes reused from siblings** — the plan left action-button classes to discretion (only `__actions` container mandated). Used the existing accent confirm pair `.link-confirm-popover__btn--continue` for the CTA (secondary anchor) and `.hub-share-internet-section__disable` for "Keep local only" (tertiary/ghost), rather than inventing new classes. No new CSS added.

## Threat Flags

- **T-166-03 (EoP, mitigated):** Two-step gesture enforced — toggle ON opens the panel; only the explicit CTA commits, shown on every enable with no don't-show-again. Test-asserted.
- **T-166-04 (EoP, mitigated):** Local-fallback fails closed (`webServerMode !== 'tailscale'` disables the toggle and blocks `SetSessionFunnel`). Test-asserted.
- **T-166-05 (Tampering, mitigated):** `expiresIn` comes from the fixed preset enum only; the UI cannot submit an arbitrary integer.

## Known Stubs

- **`onOpenHelp` App-level wiring:** the modal exposes the optional `onOpenHelp` callback and the risk panel invokes it, but HubPanel does not yet thread a tab-navigation handler from App (App.tsx is owned by Plan 166-03). The Help link currently closes the modal; actual navigation to the `help-sharing` section is a deliberate seam for a later plan.
- **`warmingUp` flag:** set on successful enable and surfaced as `data-funnel-warming` on the toggle section, but the warm-up reveal UX (Internet public section, polling, 30s timeout) is Plan 05.

## Next Step

Wave 1 sibling 166-03 (SessionCard badge + TabBar globe + App.tsx) can proceed; Wave 2 plan 166-05 consumes `warmingUp` + the synced session prop for the warm-up UX.

## Self-Check: PASSED

- [x] `FunnelRiskPanel.tsx` created and green (10/10)
- [x] `SessionShareModal.tsx` funnel toggle + risk panel + fail-closed wired (26/26)
- [x] `HubPanel.tsx` sync effect added, tsc clean
- [x] Commits `57022812`, `f453b26d`, `3694bdfc`, `72d4ec5f`, `e589990a` all in git log
- [x] Full frontend suite green (2228 tests) — no regressions
- [x] Two-step commit gesture + local-fallback fail-closed test-asserted
