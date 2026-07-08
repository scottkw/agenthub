---
phase: 173-share-modal-three-tab-segmented-redesign
verified: 2026-07-08T14:20:00Z
status: gaps_found
score: 6/8 must-haves verified
behavior_unverified: 1
overrides_applied: 0
gaps:
  - truth: "SM-07 — keyboard-accessible: real tablist/tab with roving tabindex + visible focus"
    status: failed
    reason: "ShareSegmentedControl's arrow-key handler (moveSelection) changes the `active` tab id via onSelect but never moves actual DOM focus to the newly-active tab button. The WAI-ARIA roving-tabindex pattern the component's own docstring claims to implement (lines 6-8) requires focus to follow selection. Confirmed by direct source inspection: zero `.focus(` calls exist anywhere in ShareSegmentedControl.tsx. Net effect: after an arrow-key press, DOM focus remains on the button that just became tabIndex=-1 (no longer reachable via Tab), and a subsequent Tab press skips the entire tablist rather than landing on the actual active tab. No test in ShareSegmentedControl.test.tsx asserts focus movement (only onSelect calls) — the gap is real, not merely untested."
    artifacts:
      - path: "frontend/src/components/Hub/ShareSegmentedControl.tsx"
        issue: "moveSelection() (lines 57-64) calls onSelect(next.id) only; no ref-based .focus() call on the newly active tab button"
    missing:
      - "Add a per-tab button ref map (or single tablist ref + query) and call .focus() on the newly active tab inside moveSelection, per the WAI-ARIA APG tablist pattern (matches REVIEW.md WR-03's suggested fix)"
      - "Add a regression test asserting document.activeElement moves to the new tab's button after ArrowRight/ArrowLeft"
    related_review_finding: "WR-03 (173-REVIEW.md)"
  - truth: "SM-07 — colorblind-safe On/Off/N-A text labels are the ground-truth signal for toggle state"
    status: partial
    reason: "toggleStateLabel(funnelOn || riskPanelOpen, funnelDisabled) at SessionShareModal.tsx:704 reads 'On' the instant the user clicks the internet toggle and the transient FunnelRiskPanel confirm view opens (riskPanelOpen=true) — before SetSessionFunnel has been called and before the session is actually internet-exposed. A colorblind user relying on the text label (the explicit purpose of this SM-07 affordance) would read 'On' during a window where sharing is only pending/cancelable, not committed. Confirmed by direct source inspection at SessionShareModal.tsx:687/693/695/704."
    artifacts:
      - path: "frontend/src/components/Hub/SessionShareModal.tsx"
        issue: "toggleStateLabel call at line 704 conflates funnelOn (committed) with riskPanelOpen (pending, uncommitted) into a single 'On' label"
    missing:
      - "Give the pending state its own label (e.g. funnelOn ? 'On' : riskPanelOpen ? 'Confirm…' : 'Off'), or gate the 'On' text strictly on funnelOn"
    related_review_finding: "WR-02 (173-REVIEW.md)"
human_verification:
  - test: "Open the Share modal live (rendered viewport), toggle each of the three controls, and confirm the toggle rows never move/reflow and the whole dialog never scrolls (only the tab-body region does)"
    expected: "Control strip stays pinned; only the region below the divider swaps shape; no whole-dialog scrollbar appears at a normal viewport height"
    why_human: "jsdom has no layout engine — the DOM-order-stability test (SM-01) and the CSS structure (SM-02: max-height + single .hub-share-modal__tabpanel scroll region) are both confirmed at the source/structural level, but the actual rendered reflow/scroll behavior needs a live browser (per 173-VALIDATION.md Manual-Only Verifications, already flagged by the executor)"
  - test: "In a live build, confirm the Full-access tab's active-state ring is a box-shadow inset (not a background hue swap) and is visually distinguishable to a colorblind viewer, alongside the ⚠ glyph"
    expected: "Active Full-access segment shows the ⚠ glyph in its sub label and an inset ring; no reliance on a background-color hue change alone"
    why_human: "Owner is colorblind (project memory) — verified at the source level here (style.css:6629-6636 uses `box-shadow: inset 0 0 0 1px var(--hub-destructive)`, not a background/hue change; ShareSegmentedControl.tsx:70 always prefixes the danger sub label with '⚠ '), but the live rendered affordance should still get a human/live-build check per 173-VALIDATION.md"
  - test: "With prefers-reduced-motion enabled at the OS level, open the Full-access tab's gate and confirm the hold-to-confirm control degrades to a single plain confirm click (no timed fill bar)"
    expected: "A plain 'Confirm' button appears instead of the 3s hold-fill bar; clicking it once arms the write gate"
    why_human: "Media-query behavioral branching is unit-tested via a matchMedia mock (HoldToConfirmButton.test.tsx, passing), but a real prefers-reduced-motion OS/browser environment should be spot-checked per 173-VALIDATION.md"
behavior_unverified_items:
  - truth: "SM-02 — bounded height: no state scrolls the whole dialog; scroll confined to .hub-share-modal__tabpanel within max-height: calc(100vh - 64px)"
    test: "Live-render the Share modal at a short viewport with the Internet·Full-access tab's armed result showing (tallest content state) and confirm only .hub-share-modal__tabpanel scrolls, never the whole .hub-share-modal__body"
    expected: "The modal card never exceeds calc(100vh - 64px); only the tabpanel region scrolls; the control strip + segmented control stay visible and fixed"
    why_human: "jsdom has no layout/paint engine so overflow/scroll-region behavior cannot be asserted computationally; CSS source is correct (.hub-share-modal__tabpanel: flex:1/min-height:0/overflow-y:auto; .hub-share-modal max-height unchanged) but the live behavior is explicitly called out as manual-only in 173-VALIDATION.md"
---

# Phase 173: Share modal three-tab segmented redesign — Verification Report

**Phase Goal:** Reorganize the session Share modal (#129) from a single growing/overflowing column into a stable frame — a fixed toggle control strip + a three-tab segmented access panel (Tailnet·Private / Internet·Read-only / Internet·⚠ Full-access) whose panels swap instead of stacking. Wall off the public-write/command-execution flow entirely inside the Full-access tab, replace the inline-injected Funnel risk panel with a transient confirm view, unify the four ad-hoc link rows into one reusable ShareLinkCard. Frontend-only UX/IA — no change to the sharing capability model, tokens, TTL semantics, Funnel teardown, or the 3s hold-to-confirm gate. Colorblind-safe (⚠ glyph + inset ring, not hue; On/Off/N-A text labels) and keyboard-accessible (role=tablist/tab, roving tabindex, :focus-visible).

**Verified:** 2026-07-08T14:20:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP SM-01..SM-08 contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SM-01 — fixed control strip; toggling never reflows/pushes on-screen content | ✓ VERIFIED | `hub-share-modal__rule` divider separates fixed toggles from the swappable region (SessionShareModal.tsx:714-733); DOM-order-stability test `SessionShareModal — SM-01: fixed control strip does not reflow on toggle` passes (confirmed live: 97/97 tests green across all 8 phase-173 test files) |
| 2 | SM-02 — bounded height; no state scrolls the whole dialog; scroll confined to one region within `max-height: calc(100vh - 64px)` | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS structurally correct: `.hub-share-modal` `max-height: calc(100vh - 64px)` unchanged (style.css:6413); `.hub-share-modal__tabpanel` is the designed single scroll region (`flex:1; min-height:0; overflow-y:auto`, style.css:6465-6472) with a code comment documenting the outer `__body` overflow as a safety net only. Live reflow/scroll behavior needs a rendered viewport (jsdom has no layout engine) — see Human Verification |
| 3 | SM-03 — three-tab segmented control that swaps the panel body | ✓ VERIFIED | `role="tablist"` + 3×`role="tab"` (ShareSegmentedControl.tsx:67-101); shell dispatches `tab==='tailnet'/'internet-ro'/'internet-fa'` to three distinct components (SessionShareModal.tsx:780-810); test `renders role="tablist" with three role="tab" segments, exactly one aria-selected` passes |
| 4 | SM-04 — public-write/command-execution flow lives ONLY in the Full-access tab | ✓ VERIFIED | `TailnetTab.tsx` and `InternetReadOnlyTab.tsx` contain neither `hub-funnel-write-gate` nor `HoldToConfirmButton` (grep-confirmed, zero matches); `InternetFullAccessTab.tsx` hosts the entire danger→gate→armed flow verbatim (lines 152-286); negative wall-off tests in `TailnetTab.test.tsx`/`InternetReadOnlyTab.test.tsx` assert `.hub-funnel-write-gate` and `.hub-funnel-write-gate__hold-btn` are `null`, and pass |
| 5 | SM-05 — Internet tabs disabled/aria-disabled until internet risk confirmed; risk ack is a transient confirm view; default tab after confirm = Internet·Read-only; disabling internet resets to Tailnet | ✓ VERIFIED (with a non-blocking residual gap — see Warnings) | `FunnelRiskPanel` renders as the transient confirm view REPLACING the segmented-control region (SessionShareModal.tsx:735-749), not injected above links; the confirm/reset effect (lines 359-367) sets `tab='internet-ro'` on confirm and resets to `'tailnet'` on disable; test `Internet tabs are aria-disabled until confirmed; confirming defaults to Internet·Read-only; disabling resets to Tailnet` passes. **However**, WR-01 (173-REVIEW.md) is independently confirmed: the tab-disabled condition keys off local `funnelOn` (line 336, only 3 `setFunnelOn` call sites, all user-driven — grep-confirmed, no resync effect exists) rather than `session.funnelActive` (server truth); if Funnel is disabled/expires externally while the modal stays open, the Internet tabs stay incorrectly enabled. This is a genuine phase-173-introduced regression (pre-phase code gated visibility directly on the `funnelActive` prop) but is narrow in scope (requires an external state change while the modal is open) and does not affect the security-critical write-mint gate (which reads `session.funnelActive` directly in `InternetFullAccessTab`, unaffected) |
| 6 | SM-06 — one reusable ShareLinkCard used by all tailnet + internet rows | ✓ VERIFIED (per locked scope) | `ShareLinkCard` used by `TailnetTab` (2 instances: Read-Only + Full Access) and `InternetReadOnlyTab` (1 instance: public URL). **Note:** the phase's own locked CONTEXT.md decision (D-06) explicitly scoped ShareLinkCard adoption to "Read-Only, Full Access, Public URL" — 3 of the 4 originally-identified ad-hoc rows (173-RESEARCH.md/173-DESIGN-129.md both list a 4th: the public-write gate result row). `InternetFullAccessTab.tsx`'s write-gate result row (lines 214-260) still hand-rolls its own url-row/copy/open/QR markup, structurally duplicating `ShareLinkCard`'s layout+logic rather than reusing it (flagged as IN-01 in 173-REVIEW.md, correctly rated Info-severity, not a defect — a deliberate, documented scope decision, not an oversight) |
| 7 | SM-07 — colorblind-safe + keyboard-operable: ⚠ glyph + inset ring not hue, On/Off/N-A text labels, real tablist/tab with roving tabindex + visible focus, prefers-reduced-motion fallback | ✗ FAILED | **Colorblind-safe cues: VERIFIED.** `⚠` glyph prefixed onto the danger sub-label unconditionally (ShareSegmentedControl.tsx:70); danger ring is `box-shadow: inset 0 0 0 1px var(--hub-destructive)` (style.css:6629-6631), never a background/hue swap (grep-confirmed: zero hex literals in `.share-seg*`/`.share-linkcard*` rules). **`prefers-reduced-motion` fallback: VERIFIED** — `HoldToConfirmButton` renders a plain single-click confirm branch (shared.tsx:86-101), test passes. **Roving tabindex + visible focus: FAILED** — see gap #1 below (WR-03): arrow-key navigation never moves DOM focus to the newly-active tab (zero `.focus(` calls in ShareSegmentedControl.tsx). **On/Off/N-A text labels: PARTIALLY FAILED** — see gap #2 below (WR-02): the label reads "On" during the pending/unconfirmed risk-panel-open window, misleading the exact colorblind-safety ground-truth this affordance exists to provide |
| 8 | SM-08 — modal widened to `width: min(520px, calc(100vw - 48px))`; capabilities/tokens/TTL/Funnel-teardown/3s hold-gate unchanged; test suites updated with attribute-based non-hue assertions | ✓ VERIFIED | `width: min(520px, calc(100vw - 48px))` present (style.css:6406), old `min(480px...)` gone; `SessionShareModal.test.tsx`/`.disconnect.test.tsx` updated and green; `TESTING.md` Suite Manifest reconciled (142→147, +6/-1) and `bash tests/check-traceability-paths.sh` exits 0; `tsc --noEmit` and `vite build` both succeed (independently re-run, clean) |

**Score:** 6/8 truths verified (SM-01, SM-03, SM-04, SM-05, SM-06, SM-08); 1 failed (SM-07); 1 present-but-behavior-unverified (SM-02)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/style.css` (phase-173 additions) | Width bump, single-scroll tabpanel class, `.share-seg*`/`.share-linkcard*` families, new `--hub-*` token in both theme blocks | ✓ VERIFIED | All present; `--hub-share-seg-active-bg` in both `:root` (4722) and `[data-ui-theme="light"]` (4790) |
| `frontend/src/components/SessionShare/shared.tsx` | Exported `CodeDisplay` + `HoldToConfirmButton`, single source of truth | ✓ VERIFIED | Both exported; `SessionSharePanel.tsx` deleted (no dangling local copies); reduced-motion branch added additively |
| `frontend/src/components/Hub/ShareSegmentedControl.tsx` | `role=tablist` component, roving tabindex, arrow-nav, danger glyph | ⚠️ VERIFIED w/ defect | Renders correctly and is wired; roving-tabindex focus-move is missing (gap #1) |
| `frontend/src/components/SessionShare/ShareLinkCard.tsx` | Reusable link row; QR targets join URL | ✓ VERIFIED | Used by 3 of 4 original ad-hoc rows per locked D-06 scope (see truth #6) |
| `frontend/src/components/SessionShare/{TailnetTab,InternetReadOnlyTab,InternetFullAccessTab}.tsx` | Three tab-body renderers; SM-04 wall-off | ✓ VERIFIED | Wall-off confirmed structurally + by passing negative tests |
| `frontend/src/components/Hub/SessionShareModal.tsx` | Shell: tab state machine, segmented control + dispatch, transient confirm view, On/Off/N-A labels | ⚠️ VERIFIED w/ defects | Structurally correct and wired; WR-01 (stale funnelOn resync) and WR-02 (misleading pending-state label) are real, confirmed gaps |
| `TESTING.md` | Suite Manifest + Traceability reconciled | ✓ VERIFIED | `check-traceability-paths.sh` exits 0; count matches `find` (147) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `SessionShareModal.tsx` | `ShareSegmentedControl` | `active={tab} onSelect={setTab} tabs=[...]` | ✓ WIRED | Confirmed at lines 753-772 |
| `SessionShareModal.tsx` | `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab` | conditional render on `tab` inside `.hub-share-modal__tabpanel` | ✓ WIRED | Confirmed at lines 780-810; all RPC handlers stay in the shell (no bindings imported by tabs) |
| `ShareLinkCard`/`InternetFullAccessTab` | `GetCapabilityQRCode` | `joinURLFor(url, code)` — join-code exchange URL, not raw token | ✓ WIRED | Confirmed via source read in both files; T-173-02 mitigation intact |
| `ShareSegmentedControl` (arrow key) | DOM focus | *(expected)* `.focus()` on the new active tab button | ✗ NOT WIRED | Confirmed absent — see gap #1 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 8 phase-173 vitest files green | `pnpm vitest run <8 files> --reporter=dot` | 8 files / 97 tests passed | ✓ PASS |
| `tsc --noEmit` clean | `pnpm exec tsc --noEmit` | no output, exit 0 | ✓ PASS |
| `vite build` succeeds | `pnpm exec vite build` | `✓ built in 667ms` | ✓ PASS |
| `TESTING.md` traceability paths all exist | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist`, exit 0 | ✓ PASS |
| No debt markers (TBD/FIXME/XXX) in phase-173 files | `grep -nE "TBD\|FIXME\|XXX" <7 files>` | no matches | ✓ PASS |
| No hex literals in `.share-seg*`/`.share-linkcard*` CSS | `awk`+`grep -E "#[0-9a-fA-F]{3,6}"` over those rule blocks | no matches | ✓ PASS |
| `SessionSharePanel.tsx`/test deleted, no dangling imports | `test ! -f ...` + grep | both deleted, no stray imports found | ✓ PASS |
| `.focus(` call exists in `ShareSegmentedControl.tsx` (roving-tabindex contract) | `grep -n "\.focus("` | no matches (exit 1) | ✗ FAIL — confirms gap #1 |

### Requirements Coverage

SM-01 through SM-08 are tracked in ROADMAP.md's Phase 173 entry only (not REQUIREMENTS.md's traceability table, as expected — this phase has no REQUIREMENTS.md entries, confirmed by `grep -n "Phase 173" .planning/REQUIREMENTS.md` returning nothing). All 8 were verified directly against the codebase above; no orphaned requirements.

### Anti-Patterns Found

None of the blocking kind (no TBD/FIXME/XXX, no `console.log`, no `eval`, no empty catch blocks, no hardcoded hex in new CSS). The code-review pass (173-REVIEW.md) found 0 Critical findings and 7 Warning + 4 Info findings, none of which involve debt markers.

## Non-Blocking Warnings (surfaced for visibility, not gating this verdict)

- **WR-01** (SessionShareModal.tsx): `funnelOn` never resyncs from `session.funnelActive` after mount — a phase-173-introduced regression (pre-phase code gated Internet-section visibility directly off the `funnelActive` prop). Narrow real-world trigger (external Funnel state change while the modal stays open); the security-critical write-mint path is unaffected. Recommend fixing before the next hardening pass on this surface.
- **SM-06 scope note**: `InternetFullAccessTab`'s write-gate result row duplicates `ShareLinkCard`'s layout/logic rather than reusing it. This is a sanctioned, locked scope decision from `173-CONTEXT.md` (D-06 explicitly names only 3 of the 4 originally-identified ad-hoc rows), not an oversight — but it does mean the ROADMAP's literal "four ad-hoc link rows" / "used by all tailnet + internet rows" framing is only 3/4 satisfied. Rated Info (IN-01) by the code reviewer, not Warning.
- **WR-04/WR-05/WR-06/WR-07** (173-REVIEW.md): QR-cache staleness on prop change, HoldToConfirmButton's disabled-recheck-at-completion gap, LAN-password effect missing a cancelled-guard, and uncleared copy-feedback timers. Independently confirmed these are **pre-existing behaviors carried over verbatim** from `SessionSharePanel.tsx` (not phase-173 regressions) — consistent with D-08's "unchanged behavior" preservation goal. Legitimate tech debt, but out of this phase's goal scope; recommend filing as a follow-up issue rather than blocking this phase.

## Gaps Summary

The phase substantially achieves its goal: the modal is now a fixed control strip + swappable three-tab panel, the public-write/command-execution flow is provably walled off (tested negative assertions), the transient confirm view replaces the old inline-injected risk panel, ShareLinkCard unifies 3 of the 4 original ad-hoc link rows (4th excluded by a locked, documented scope decision), the modal is widened per spec, and all capability/token/TTL/teardown/hold-gate behavior is verifiably unchanged. `tsc`, `vite build`, the full targeted vitest run (97/97), and the TESTING.md traceability gate are all green.

However, two concrete, source-confirmed defects in the SM-07 keyboard/colorblind-accessibility requirement block a clean pass:

1. **Roving tabindex is incomplete** — `ShareSegmentedControl`'s arrow-key handler never moves DOM focus to the newly active tab (zero `.focus()` calls exist in the file), breaking the WAI-ARIA APG tablist contract the component's own docstring claims to follow. A keyboard user pressing an arrow key then Tab will skip the entire control.
2. **The On/Off state label can mislead the colorblind ground-truth signal** — it reads "On" during the pending/unconfirmed risk-panel window, before the user has actually committed to enabling internet sharing.

Both are explicitly named in the ROADMAP's SM-07 acceptance text ("roving tabindex" and "On/Off/N-A text labels") — this is not scope creep or an edge case, it is the literal requirement not being fully met by the new code. Both fixes are small, well-scoped, and already have suggested patches in 173-REVIEW.md (WR-02/WR-03).

Additionally, SM-02's live no-whole-dialog-scroll/no-reflow behavior cannot be verified without a rendered browser viewport (jsdom limitation) — this was already correctly flagged as manual-only in 173-VALIDATION.md and is carried forward as a human-verification item, not a gap.

---

_Verified: 2026-07-08T14:20:00Z_
_Verifier: Claude (gsd-verifier)_
