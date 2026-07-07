---
phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard
plan: 03
subsystem: ui
tags: [react, heroicons, funnel, rw-gate, hold-to-confirm, colorblind-safe]

requires:
  - phase: 171-02
    provides: "handleSetSessionFunnelWrite/disableFunnelWriteForSession RPC surface (SetSessionFunnelWrite/DisableSessionFunnelWrite), SessionInfo.FunnelWriteActive (no omitempty), hand-synced Wails TS bindings (App.d.ts/App.js/models.ts) — the exact daemon surface this plan's UI consumes"
provides:
  - "Danger section (.hub-funnel-write-gate) in SessionSharePanel.tsx: locked risk-forward consent copy (contains 'command execution' + 'anyone with the link' verbatim), 15m/30m/1h expiry select (no 'Until I disable'), a real <button> hold-to-confirm control (pointerdown/up/leave + Space/Enter keydown/keyup, 3000ms) that fires onGateConfirm(expirySeconds) exactly once on completion and issues zero RPCs on early release"
  - "Post-gate result block: public write URL (Copy/Open/QR) + <CodeDisplay label=\"Single-use write code:\"> + live mm:ss countdown (destructive-colored under 60s) + 'Disable public write' button; a writeGateUsed prop collapses the URL/code rows to 'Write code used — one writer connected' while keeping the countdown + disable button"
  - "SessionShareModal.tsx wiring: hold-completion -> SetSessionFunnelWrite(session.id, expirySeconds); Disable button -> DisableSessionFunnelWrite; session.funnelWriteActive (existing hubSessions poll) is the authoritative true->false signal that clears local write-gate state (Disable and Expiry converge on the same collapse-to-Idle path, client-side countdown is visual only)"
  - ".hub-fullaccess-badge (SessionCard) + .tab__fullaccess-icon (TabBar): LockOpenIcon + 'FULL ACCESS' + notched clip-path badge geometry, rendered after the read .hub-internet-badge/.tab__internet-icon (read-then-write order), gated solely on funnelWriteActive so RW teardown leaves the read indicator untouched"
  - "App.tsx funnelWriteActiveSessions derivation from the existing hubSessions 3s poll (mirrors funnelActiveSessions, no new interval), passed to TabBar"
  - "--hub-fullaccess-badge-bg/-text CSS tokens (dark #f7768e / light #c0394f, alias-by-value of --hub-destructive) and the full .hub-funnel-write-gate*/.hub-fullaccess-badge*/.tab__fullaccess-icon component CSS, with prefers-reduced-motion handling that keeps the hold-fill's geometry functional (transition removed, JS-driven width steps preserved)"
affects: [171-04]

tech-stack:
  added: []
  patterns:
    - "Hold-to-confirm gesture: a single setTimeout(3000ms) is the sole authority for completion (not pointerup) — pointerup/pointerleave/keyup before it fires only clear the timers and reset visual progress to 0%, they never call onConfirm. A parallel setInterval(100ms) reads Date.now() - startRef.current purely to drive the visual fill percentage; both timers are always cleared together via one clearTimers() helper (mirrors the codebase's existing ChatPanel inject-hold precedent but adds the JS-driven percentage tick, required because the fill motion is functionally load-bearing under prefers-reduced-motion, not decorative)."
    - "Authoritative-poll-clears-local-state: SessionShareModal watches session.funnelWriteActive (not a local timer) transition true->false via a ref-compared useEffect to clear the write-gate's local URL/code/expiry/used state — the same shape as the existing warm-up-completion effect for the read Funnel path, generalized to a teardown-detection direction instead of an activation-detection direction."
    - "Danger section mirrors the read .hub-share-internet-section's funnelEngaged gating exactly (same visibility condition) but the *inner* hold control has its own narrower disabled condition (funnelActive && !warmingUp) — visibility and interactivity are gated independently, per the UI-SPEC's resolved Open Question 1."

key-files:
  created: []
  modified:
    - frontend/src/style.css
    - frontend/src/components/SessionSharePanel.tsx
    - frontend/src/components/Hub/SessionShareModal.tsx
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/TabBar.tsx
    - frontend/src/App.tsx
    - frontend/src/components/__tests__/SessionSharePanel.test.tsx
    - frontend/src/components/__tests__/SessionCard.share.test.tsx
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - frontend/src/components/__tests__/SessionShareModal.disconnect.test.tsx

key-decisions:
  - "writeGateUsed is a controlled prop with no live backend driver this phase — SessionInfo exposes funnelWriteActive only (stays true across a redemption until disable/expiry), not a distinct 'redeemed' signal; inventing a heuristic (e.g. off viewerCount deltas) risked a false-positive 'code used' state, so it was deliberately left undriven and documented as a gap. Live single-use verification is 171-04's manual M-47 UAT."
  - "The Danger section's post-gate URL row renders writeGateUrl as-is (no /join?code= derivation) — unlike the read section's FNL-08 reusable-code rewrite, this cap-embedded URL is the direct analog of the existing Full Access Link block (same file), which also renders its raw cap URL alongside a separate CodeDisplay code row."
  - "Consent-copy literal-phrase test asserts case-insensitively ('anyone with the link' vs the rendered 'Anyone with the link…') since the locked UI-SPEC copy capitalizes the sentence-initial word; case-insensitive substring match is the correct interpretation of 'literally contains' here, not a relaxation of the check."
  - "Hold-to-confirm completion is timer-authoritative, not release-authoritative: the 3000ms setTimeout fires onGateConfirm even though the test never dispatches pointerup, exactly matching the Interaction Contract's 'Hold completes (3000ms reached while still pressed)' — release is only ever a cancellation path, never a completion trigger."

coverage:
  - id: D1
    description: "Releasing the hold before 3s issues zero SetSessionFunnelWrite calls; fill resets to 0%, label reverts to 'Hold 3s to confirm' (R1 gate-completeness edge)"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx#R1: releasing the hold before 3s issues nothing"
        status: pass
    human_judgment: false
  - id: D2
    description: "Completing the >=3s hold fires exactly one onGateConfirm(expirySeconds) and reveals the write URL/code/countdown/disable block"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx#R1: completing the >=3s hold fires exactly one onGateConfirm"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx#renders the post-gate result block"
        status: pass
    human_judgment: false
  - id: D3
    description: "Keyboard equivalent (Space/Enter) drives the same hold timer; early keyup issues nothing, a full hold fires once"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx#keyboard equivalent"
        status: pass
    human_judgment: false
  - id: D4
    description: "The RW indicator differs from the read indicator by label, icon import, and shape (clip-path notch vs pill radius) — colorblind-safe, verified at source"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionCard.share.test.tsx#R7: renders .hub-fullaccess-badge with LockOpenIcon and literal label"
        status: pass
      - kind: other
        ref: "grep -n 'hub-fullaccess-badge-text\\|hub-internet-badge-text' frontend/src/style.css (different hex in both theme blocks); grep -n 'clip-path' frontend/src/style.css"
        status: pass
    human_judgment: false
  - id: D5
    description: "FULL ACCESS badge/tab-icon coexist with the read indicator (read-then-write order) and clear independently of funnelActive (RW teardown keeps the read badge)"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionCard.share.test.tsx#coexistence: both badges render"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionCard.share.test.tsx#RW teardown keeps the read badge"
        status: pass
    human_judgment: false
  - id: D6
    description: "The expiry selector offers only 15m/30m/1h (default 15m), no 'Until I disable'"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx#R1: completing the >=3s hold fires exactly one onGateConfirm (asserts default 900)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Consent-gate copy contains the mandated literal phrases 'command execution' and 'anyone with the link' (SPEC Prohibition #4) and the hold control is disabled until funnelActive && !warmingUp"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx#consent-copy compliance"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionSharePanel.test.tsx#warm-up gating"
        status: pass
    human_judgment: false
  - id: D8
    description: "Full frontend gate: tsc --noEmit clean, vite build succeeds, and the whole vitest suite is green (no regression from the new required funnelWriteActive field or the new Danger section/badge/icon)"
    requirement: FNL-09
    verification:
      - kind: other
        ref: "cd frontend && pnpm exec tsc --noEmit && pnpm build"
        status: pass
      - kind: unit
        ref: "cd frontend && pnpm test -- --run (142 files / 2353 tests)"
        status: pass
    human_judgment: false

duration: 21min
completed: 2026-07-07
status: complete
---

# Phase 171 Plan 03: Danger Section (Hold-to-Confirm Public Write Gate) + FULL ACCESS Indicators Summary

**A real `<button>` hold-to-confirm gesture (pointer + keyboard, 3000ms, timer-authoritative completion) gates `SetSessionFunnelWrite`, with a colorblind-safe FULL ACCESS badge (LockOpenIcon + notched clip-path, distinct from the read INTERNET pill) that coexists with and clears independently of the existing read share.**

## Performance

- **Duration:** 21 min
- **Started:** 2026-07-07T16:30:30-05:00 (approx., immediately after 171-02)
- **Completed:** 2026-07-07T16:51:12-05:00
- **Tasks:** 3
- **Files modified:** 10 (0 new files — 2 pre-existing test files were extended in place, 2 additional test fixture files fixed for a required-field build break)

## Accomplishments

- `.hub-funnel-write-gate` Danger section in `SessionSharePanel.tsx`: locked risk-forward warning copy (contains "command execution" and "anyone with the link" verbatim, SPEC Prohibition #4), a 15m/30m/1h expiry `<select>` (D-11/R5, no "Until I disable"), and a real `<button>` hold-to-confirm control — pointerdown/up/leave AND Space/Enter keydown/keyup drive the same 3000ms timer. The 3000ms `setTimeout` (not `pointerup`) is authoritative for completion: early release only clears timers and resets the visual fill to 0% with zero RPC calls (R1); a full hold fires `onGateConfirm(expirySeconds)` exactly once. Post-gate renders the public write URL (Copy/Open/QR, mirroring the existing Full Access Link pattern) + `<CodeDisplay label="Single-use write code:">` + a live "Expires in mm:ss" countdown (destructive-colored under 60s) + "Disable public write". A `writeGateUsed` prop collapses the URL/code rows to "Write code used — one writer connected" while the countdown and disable button stay visible.
- `SessionShareModal.tsx`: wires hold-completion to `SetSessionFunnelWrite(session.id, expirySeconds)` and the Disable button to `DisableSessionFunnelWrite`; `session.funnelWriteActive` (the existing hubSessions 3s poll, no new interval) is the authoritative true→false signal that clears local write-gate state — Disable and Expiry converge on the same collapse-to-Idle path, matching the Interaction Contract's steps 7/8. `ShareSession` gained `funnelWriteActive`.
- `.hub-fullaccess-badge` (SessionCard) and `.tab__fullaccess-icon` (TabBar): `LockOpenIcon` + literal "FULL ACCESS" label + notched `clip-path` badge geometry (not the read badge's pill `border-radius`) — colorblind-safe per D-09, verified at the source level (different class, icon import, label string, and CSS shape rule, never color alone). Rendered after the read `.hub-internet-badge`/`.tab__internet-icon` (read-then-write order, D-10) and gated solely on `funnelWriteActive`, so an RW-only teardown leaves the read indicator in place. `App.tsx` derives `funnelWriteActiveSessions` from the existing `hubSessions` poll, mirroring `funnelActiveSessions` exactly.
- `--hub-fullaccess-badge-bg`/`-text` CSS tokens (dark `#f7768e` / light `#c0394f`, alias-by-value of `--hub-destructive`, deliberately a different hue from the read badge's teal) plus the full `.hub-funnel-write-gate*`/`.hub-fullaccess-badge*`/`.tab__fullaccess-icon` component CSS block, including the `prefers-reduced-motion` handling that keeps the hold-fill's width change functional (JS-driven, discrete) while removing only the CSS transition easing.

## Task Commits

Each task was committed atomically:

1. **Task 1: CSS tokens + Danger-section + FULL ACCESS badge/icon styles** - `2b9af74f` (feat)
2. **Task 2: Danger section + hold-to-confirm gate + modal wiring** - `40331fa2` (feat)
3. **Task 3: FULL ACCESS badge (SessionCard) + tab icon (TabBar) + App poll derivation** - `1ff76e3e` (feat)

## Files Created/Modified

- `frontend/src/style.css` - `--hub-fullaccess-badge-bg`/`-text` tokens (dark+light) + `.hub-funnel-write-gate*`/`.hub-fullaccess-badge*`/`.tab__fullaccess-icon` component CSS
- `frontend/src/components/SessionSharePanel.tsx` - `HoldToConfirmButton` + `.hub-funnel-write-gate` Danger section (warning, expiry select, hold control, post-gate result/used states, disable button)
- `frontend/src/components/Hub/SessionShareModal.tsx` - write-gate state (`writeGateUrl`/`writeGateCode`/`writeGateExpiresAt`/`writeGateUsed`) + `SetSessionFunnelWrite`/`DisableSessionFunnelWrite` wiring + authoritative-poll teardown effect; `ShareSession.funnelWriteActive`
- `frontend/src/components/Hub/SessionCard.tsx` - `.hub-fullaccess-badge` render block (after `.hub-internet-badge`)
- `frontend/src/components/TabBar.tsx` - `.tab__fullaccess-icon` render (after `.tab__internet-icon`) + `funnelWriteActiveSessions` prop
- `frontend/src/App.tsx` - `funnelWriteActiveSessions` poll derivation, passed to `TabBar`
- `frontend/src/components/__tests__/SessionSharePanel.test.tsx` - early-release/complete-hold/keyboard/used-state/consent-copy/warm-up-gating tests
- `frontend/src/components/__tests__/SessionCard.share.test.tsx` - R7 distinct-indicator + coexistence + RW-teardown-keeps-read-badge tests
- `frontend/src/components/__tests__/SessionShareModal.test.tsx`, `SessionShareModal.disconnect.test.tsx` - `funnelWriteActive: false` added to the `makeSession` fixtures (Rule 1 build fix)

## Decisions Made

- `writeGateUsed` left as an undriven controlled prop (see key-decisions above) — documented rather than faked, since the only plausible client-side heuristic (viewer-count deltas) cannot reliably distinguish "this specific single-use write code was redeemed" from any other viewer connecting via the read link or Full Access Link.
- Post-gate write URL rendered as-is (no `/join?code=` rewrite), matching the existing Full Access Link block's pattern rather than the read section's FNL-08 reusable-code derivation, since the write cap URL is a direct grant-bound link, not a share-lifetime reusable one.
- Consent-copy test asserts case-insensitively against the locked UI-SPEC copy's sentence-initial "Anyone" capitalization.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added `funnelWriteActive: false` to 2 `SessionShareModal` test fixtures**
- **Found during:** Task 2 (frontend `tsc --noEmit` gate)
- **Issue:** `ShareSession` gained a required `funnelWriteActive: boolean` field (mirroring `SessionInfo`'s existing no-omitempty rule); `tsc --noEmit` broke in `SessionShareModal.test.tsx`'s `makeSession` helper and `SessionShareModal.disconnect.test.tsx`'s fixture, both of which construct the session object without it.
- **Fix:** Added `funnelWriteActive` (optional override + `?? false` default) to `SessionShareModal.test.tsx`'s `ModalOpts`/`makeSession`, and a plain `funnelWriteActive: false` field to `SessionShareModal.disconnect.test.tsx`'s fixture.
- **Files modified:** `frontend/src/components/__tests__/SessionShareModal.test.tsx`, `frontend/src/components/__tests__/SessionShareModal.disconnect.test.tsx`
- **Verification:** `cd frontend && pnpm exec tsc --noEmit` clean; `pnpm test -- SessionShareModal --run` — 42 tests pass.
- **Committed in:** `40331fa2` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - build-breakage fix)
**Impact on plan:** Directly required for the plan's own `tsc --noEmit` gate to pass after adding the required `ShareSession.funnelWriteActive` field. No scope creep beyond what was necessary.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The full frontend gate is green: `tsc --noEmit` clean, `pnpm build` succeeds, and the whole vitest suite (142 files / 2353 tests, +16 net new this plan) passes.
- Colorblind-safety verified at source: `--hub-fullaccess-badge-text` (`#f7768e`/`#c0394f`) resolves to a different hex than `--hub-internet-badge-text` in both theme blocks; `LockOpenIcon` vs `GlobeAltIcon` are distinct icon imports; `"FULL ACCESS"` vs `"INTERNET"` are distinct literal labels; `.hub-fullaccess-badge` uses `clip-path`, never the pill's `border-radius`.
- 171-04 (TESTING.md reconciliation, Sharing Guide RW section, 171-SECURITY.md synthesis) can now cite this plan's exact test names/counts — no new test FILES were created this plan (both `SessionSharePanel.test.tsx` and `SessionCard.share.test.tsx` were extended in place), so the Suite Manifest file counts are unchanged; only new traceability rows are needed.
- Known gap for 171-04's manual UAT: `writeGateUsed`'s live detection (single-use-code-redeemed → "Write code used" collapse) has no automated driver this phase — this is exactly the kind of behavior 171-04's M-47 live off-tailnet UAT (hold gate → redeem from a real off-tailnet device → owner UI reflects "used") is designed to prove end-to-end.
- No blockers for 171-04.

---
*Phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard*
*Completed: 2026-07-07*

## Self-Check: PASSED

All 9 created/modified files found on disk; all 3 task commit hashes (2b9af74f, 40331fa2, 1ff76e3e) found in git log.
