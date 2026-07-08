---
phase: 173-share-modal-three-tab-segmented-redesign
plan: 08
reviewed: 2026-07-08T09:30:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - frontend/src/components/Hub/ShareSegmentedControl.tsx
  - frontend/src/components/Hub/SessionShareModal.tsx
  - frontend/src/components/__tests__/ShareSegmentedControl.test.tsx
  - frontend/src/components/__tests__/SessionShareModal.test.tsx
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 173 Plan 08: Gap-Closure Code Review Report

**Reviewed:** 2026-07-08T09:30:00Z
**Depth:** standard
**Files Reviewed:** 4 (delta only — commits `5f6be853..HEAD`)
**Status:** issues_found (no blockers; the two closed gaps are correctly fixed)

## Summary

This is a surgical delta review of 173-08, not a re-review of Phase 173. Scope: the `btnRefs`/`.focus()` roving-tabindex fix in `ShareSegmentedControl.tsx`, the pending-aware Internet toggle label + `funnelOn`↔`session.funnelActive` resync effect in `SessionShareModal.tsx`, and the two new regression test suites.

Both source-confirmed gaps (WR-03 focus-follow, WR-02 pending label) are fixed correctly and match their stated intent:

- `moveSelection` now calls `btnRefs.current[next.id]?.focus()` immediately after `onSelect(next.id)`, keyed on `next.id` (not the stale `active` prop) as the plan required. `next` is always drawn from `enabledTabs`, so `.focus()` is never called on a `disabled` button — no null-ref or focus-escape risk. Ran the two touched test files live (`pnpm exec vitest run ShareSegmentedControl.test.tsx SessionShareModal.test.tsx`): 60/60 pass. `tsc --noEmit` is clean.
- The Internet toggle-state text (`funnelDisabled ? 'N/A' : funnelOn ? 'On' : riskPanelOpen ? 'Confirm…' : 'Off'`) correctly gates `'On'` strictly on committed `funnelOn`; the pending/uncommitted window reads `'Confirm…'`, never `'On'`. Checkbox `checked`/`aria-checked` intentionally still track `funnelOn || riskPanelOpen`, matching the plan's explicit "text-only" scoping.
- The new `funnelOn` resync `useEffect` (deps `[session.funnelActive]` only, `funnelOn` deliberately excluded) does not stomp `handleFunnelEnable`'s optimistic `setFunnelOn(true)` and does not infinite-loop (guarded by the `session.funnelActive === funnelOn` early return, and the dependency array only re-fires on an actual prop **value** change, not every render). Traced the effect ordering against the pre-existing `prevFunnelOnRef` tab-reset effect — they compose correctly (resync flips `funnelOn`, which then triggers the existing tab-reset effect on the next commit) with no duplicate reset, as documented.

One real, if narrow, correctness gap remains in the resync design itself (see WARN-01) — the effect can only correct drift when `session.funnelActive`'s *value* changes; it cannot correct drift caused by the local state alone moving away from an *unchanged* server-truth prop.

## Warnings

### WR-01: `funnelOn` resync effect cannot recover from a failed `handleDisableFunnel` call — the "reconciles on next poll" comment stays false in that path

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:339-355` (new resync effect) interacting with the pre-existing, untouched `handleDisableFunnel` (`frontend/src/components/Hub/SessionShareModal.tsx:475-491`)

**Issue:** The new resync effect is:
```js
useEffect(() => {
  if (session.funnelActive === funnelOn) return
  setFunnelOn(session.funnelActive)
}, [session.funnelActive])
```
This only re-fires when the **value** of `session.funnelActive` changes between renders — by design, so it doesn't stomp `handleFunnelEnable`'s optimistic update (correct, and verified by the new WR-01 regression test for the out-of-band *disable* case, which does change the prop value).

But `handleDisableFunnel` (untouched by this plan) unconditionally does `setFunnelOn(false)` even when `SetSessionFunnel(session.id, false, 0)` throws:
```js
try {
  await SetSessionFunnel(session.id, false, 0)
} catch {
  // Best-effort teardown — still clear local state; the indicator reconciles on next poll.
}
setFunnelOn(false)
```
If that call fails (network drop, daemon error), `session.funnelActive` stays `true` on the server and the prop **never changes value** — it was `true`, remains `true`. Because the new resync effect only reacts to value changes, it never re-fires to correct `funnelOn` back to `true`. The result: the modal now shows the Internet toggle as `Off`, both Internet tabs `aria-disabled`, and the active tab reset to Tailnet — while the session is, in fact, still Funnel-exposed server-side. The comment's promise ("the indicator reconciles on next poll") remains unfulfilled for this specific failure path even after this gap-closure lands, and 173-08's own threat model (T-173-08-02, "stale display" mitigation) is only half-closed: it fixes the *external-disable-while-modal-open* direction (tested) but not the *local-disable-API-call-fails* direction (untested, unfixed).

This is a display-only issue (the write-mint gate in `InternetFullAccessTab` reads `session.funnelActive` directly per the threat model, so no capability is actually revoked or over-granted) but it is the wrong direction of error for a security-adjacent affordance: it makes a still-internet-exposed session look safely Off.

**Fix:** Either (a) have `handleDisableFunnel`'s catch branch leave `funnelOn` unchanged (don't optimistically clear on failure — mirrors how `handleFunnelEnable`'s catch already avoids creating a false-positive by setting `funnelOn` back to the value consistent with `session.funnelActive`), or (b) drive `funnelOn` display purely off `session.funnelActive` and use a separate `pendingDisable` flag for the optimistic transition, so a failed disable falls back to server truth instead of freezing on the optimistic guess:
```js
async function handleDisableFunnel(): Promise<void> {
  if (warmupTimeoutRef.current) { clearTimeout(warmupTimeoutRef.current); warmupTimeoutRef.current = null }
  try {
    await SetSessionFunnel(session.id, false, 0)
    setFunnelOn(false)
  } catch {
    // Leave funnelOn as-is (server truth still active); surface an inline error instead
    // of silently understating exposure.
    setFunnelError('Failed to disable internet sharing. Please try again.')
    return
  }
  setFunnelUrl(null)
  setPublicReadCode(null)
  setWarmingUp(false)
  setWarmupTimedOut(false)
}
```

## Info

### IN-01: Per-tab ref callback is a fresh closure every render (unnecessary detach/reattach churn)

**File:** `frontend/src/components/Hub/ShareSegmentedControl.tsx:78-80`

**Issue:**
```jsx
ref={(el) => {
  btnRefs.current[t.id] = el
}}
```
This inline arrow function is recreated on every render of `ShareSegmentedControl`. React treats a changed ref-callback identity as "detach then reattach" even when the underlying DOM node hasn't changed (same `key={t.id}`), so every re-render of the tab bar calls each tab's ref callback with `null` and then the element again. Functionally harmless here (assignment into a plain object, no cleanup logic to double-fire), but it's unnecessary churn and a minor anti-pattern.

**Fix:** Memoize per-tab ref callbacks, e.g. build them once with `useCallback` in a small helper keyed by `t.id`, or use a `useMemo`'d map of stable callbacks:
```jsx
const setBtnRef = React.useCallback(
  (id: ShareTab) => (el: HTMLButtonElement | null) => { btnRefs.current[id] = el },
  [],
)
// ...
ref={setBtnRef(t.id)}
```
(Note `setBtnRef(t.id)` itself still allocates a new closure per render unless further memoized per-id; a `Map<ShareTab, RefCallback>` built once via `useMemo` would fully stabilize it. Low priority — three tabs, no measurable cost — flagged for completeness only.)

### IN-02: Internet toggle-state label duplicates `toggleStateLabel`'s On/Off/N-A logic instead of extending it

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:28-31` (`toggleStateLabel`) vs `frontend/src/components/Hub/SessionShareModal.tsx:731`

**Issue:** The Share and Browse toggle rows still call the shared `toggleStateLabel(checked, disabled)` helper (lines 632, 672), but the Internet row now inlines its own three/four-way ternary (`funnelDisabled ? 'N/A' : funnelOn ? 'On' : riskPanelOpen ? 'Confirm…' : 'Off'`) rather than extending the helper with an optional "pending" branch. This is explicitly permitted by the plan (either approach was acceptable) and doesn't cause a behavioral bug, but it leaves two divergent implementations of the same On/Off/N-A contract in one file — a future edit to `toggleStateLabel`'s copy (e.g. localizing the strings, or changing `'N/A'` to something else) will silently not apply to the Internet row.

**Fix:** Consider folding the pending case into `toggleStateLabel` for consistency, e.g. `toggleStateLabel(checked, disabled, pending?)`, and call it uniformly from all three rows. Non-blocking; purely a maintainability note.

---

_Reviewed: 2026-07-08T09:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
