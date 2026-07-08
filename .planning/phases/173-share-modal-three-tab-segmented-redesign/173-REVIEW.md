---
phase: 173-share-modal-three-tab-segmented-redesign
reviewed: 2026-07-08T13:24:37Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - frontend/src/components/Hub/SessionShareModal.tsx
  - frontend/src/components/Hub/ShareSegmentedControl.tsx
  - frontend/src/components/SessionShare/shared.tsx
  - frontend/src/components/SessionShare/ShareLinkCard.tsx
  - frontend/src/components/SessionShare/TailnetTab.tsx
  - frontend/src/components/SessionShare/InternetReadOnlyTab.tsx
  - frontend/src/components/SessionShare/InternetFullAccessTab.tsx
  - frontend/src/style.css (phase-173 additions only)
findings:
  critical: 0
  warning: 7
  info: 4
  total: 11
status: issues_found
---

# Phase 173: Code Review Report

**Reviewed:** 2026-07-08T13:24:37Z
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Reviewed the three-tab segmented Share modal redesign: `SessionShareModal.tsx` (shell/state machine), `ShareSegmentedControl.tsx` (new tablist), `shared.tsx` (hoisted `HoldToConfirmButton`/`CodeDisplay`), `ShareLinkCard.tsx` (new reusable card), the three tab-body renderers (`TailnetTab`, `InternetReadOnlyTab`, `InternetFullAccessTab`), and the phase-173 CSS additions to `style.css`.

The security-critical boundary held up under scrutiny: the actual write-cap-minting control (`HoldToConfirmButton` inside `InternetFullAccessTab`) is gated on the real server-truth prop `session.funnelActive` (not any local/optimistic state), QR codes always encode the join-code exchange URL rather than the raw capability token (confirmed in both `ShareLinkCard` and `InternetFullAccessTab`), and the wall-off between read-only tabs and the public-write flow is structurally intact — neither `TailnetTab.tsx` nor `InternetReadOnlyTab.tsx` imports or references `HoldToConfirmButton` or any write-gate state. No hardcoded secrets, `eval`, `console.log`, or empty catch blocks were found in the reviewed files.

However, the new tab-availability state machine introduced in this phase (`ShareSegmentedControl`'s disabled-tab gating, the toggle's new textual state label, and the new roving-tabindex arrow-key navigation) has several correctness gaps, and the new `ShareLinkCard`/`InternetFullAccessTab` QR-caching logic can display a stale QR image after the underlying code rotates. None of these rise to Critical (the server-truth-gated mint path is unaffected), but several are genuine regressions/gaps introduced by this phase's refactor, not pre-existing carryover, and should be fixed before this ships.

## Warnings

### WR-01: Tab-availability gating keys off local `funnelOn`, which is never resynced from server truth

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:336`, `:359-367`, `:762`, `:768`
**Issue:** The new (phase-173) `ShareSegmentedControl` tab config disables the two Internet tabs with `disabled: !funnelOn || funnelDisabled` (lines 762/768). `funnelOn` is local component state, seeded once from `session.funnelActive` at mount (line 336) and only ever flipped by the modal's own `handleFunnelEnable`/`handleDisableFunnel` handlers (lines 395, 407, 467). There is no effect that reconciles `funnelOn` when `session.funnelActive` changes for any other reason (auto-expiry via the `FunnelRiskPanel` expiry options, another device disabling Funnel, daemon restart). The comment at line 331 claims "Plan 05's poll keeps it live via the sync effect" but no such effect exists anywhere in this file (`grep -n setFunnelOn` shows exactly three call sites, all from local user actions).

Before this phase, `SessionSharePanel.tsx` gated its Internet section visibility directly on the `funnelActive` prop (server truth), not a local mirror — this refactor introduced the staleness by adding a tab-disabled concept that keys off `funnelOn` instead. Net effect: if Funnel auto-expires or is disabled out-of-band while the modal is open, the "Enable internet sharing" toggle and the two Internet tabs remain visually enabled/checked (and the toggle's new text label reads "On" — see WR-02) even though the session is no longer exposed. The actual write-mint path is not affected (it re-checks `session.funnelActive` directly, see Summary), so this is a display/affordance bug, not an exploitable gap.
**Fix:** Either derive the tab-disabled condition directly from `session.funnelActive` (server truth) instead of `funnelOn`, or add an effect that resyncs `funnelOn` (and resets `tab`) whenever `session.funnelActive` transitions independently of the local handlers, e.g.:
```tsx
useEffect(() => {
  if (session.funnelActive === funnelOn) return
  setFunnelOn(session.funnelActive)
}, [session.funnelActive])
```

### WR-02: Toggle state label reads "On" during the uncommitted risk-panel-open state

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:28-31`, `:704`
**Issue:** `toggleStateLabel(funnelOn || riskPanelOpen, funnelDisabled)` is new in this phase (added specifically to satisfy D-07's colorblind-safety requirement that toggle state must be readable as text, not just knob position/color). But `riskPanelOpen` only means the user clicked the toggle and the `FunnelRiskPanel` confirm view is now showing — no `SetSessionFunnel` call has been made yet, and the session is not internet-exposed. During this window the text label nonetheless reads "On", which is exactly the kind of non-color-dependent state signal the colorblind owner is meant to rely on (per project convention: verify colorblind-safety at the source, and the whole point of D-07 was to give a textual ground truth). A user relying on the text (as intended) would believe internet sharing is already live when it is only pending confirmation/cancelable.
**Fix:** Give the pending state its own label, e.g. `funnelOn ? 'On' : riskPanelOpen ? 'Confirm…' : 'Off'`, or gate the "On" text strictly on `funnelOn` and let the checkbox `checked` attribute alone reflect the "about to enable" affordance.

### WR-03: Arrow-key navigation never moves DOM focus to the newly active tab

**File:** `frontend/src/components/Hub/ShareSegmentedControl.tsx:57-64`, `:88-96`
**Issue:** The component's own doc comment (lines 6-8) states the roving-tabindex/arrow-key contract is implemented "per the WAI-ARIA Authoring Practices Guide tablist pattern." That pattern requires that pressing ArrowLeft/ArrowRight both changes the active tab (automatic activation) AND moves actual DOM focus to the new tab button. Here, `moveSelection` (lines 57-64) only calls `onSelect(next.id)`; there is no `.focus()` call on the newly-active button's DOM node. After the parent re-renders with a new `active` prop, the previously-focused button gets `tabIndex={-1}` while DOM focus remains on it — the visible focus ring stays on a segment that is no longer marked `aria-selected`/`tabIndex=0`. Pressing Tab afterward will skip the (now `tabIndex=-1`) previously-focused button and jump past the segmented control entirely rather than landing on the actual active tab. Confirmed by the test suite (`ShareSegmentedControl.test.tsx`): every arrow-key test asserts only `onSelect` was called, never that focus moved.
**Fix:** Track a ref per tab button (or a single ref to the tablist and query by id) and call `.focus()` on the newly active tab inside `moveSelection`, e.g.:
```tsx
const btnRefs = useRef<Record<string, HTMLButtonElement | null>>({})
function moveSelection(delta: 1 | -1): void {
  ...
  if (next) {
    onSelect(next.id)
    btnRefs.current[next.id]?.focus()
  }
}
```

### WR-04: `ShareLinkCard` and `InternetFullAccessTab` cache the fetched QR image without invalidating it when the URL/code props change

**File:** `frontend/src/components/SessionShare/ShareLinkCard.tsx:59-90`; `frontend/src/components/SessionShare/InternetFullAccessTab.tsx:90-150`
**Issue:** `ShareLinkCard.handleToggleQR` caches the fetched QR PNG in `qrB64` state and only re-fetches `if (!qrB64)` (line 80). Nothing resets `qrB64`/`qrOpen`/`qrError` when the `url`/`code` props change. Since `ShareLinkCard` instances are re-used by position (not remounted) when `SessionShareModal` re-issues capabilities — e.g. toggling "Enable remote file browsing" calls `IssueCapabilities` again and replaces `cachedShare` with fresh `readURL`/`readCode`/`writeURL`/`writeCode` (`SessionShareModal.tsx:299-305`) — a QR already opened before the toggle continues to display the stale, now-superseded code's image after the toggle, while the adjacent `CodeDisplay` text correctly shows the new code. The same pattern is duplicated in `InternetFullAccessTab.handleToggleGateQR` (lines 133-150): disabling and re-arming the public-write gate without leaving the tab (component instance persists) leaves `gateQRb64` holding the previous, now-disabled write grant's QR image, which is shown immediately once the new grant arms if `showGateQR` was left `true`.
**Fix:** Add an effect (or reset inline) that clears the cached QR state whenever the identifying props change:
```tsx
useEffect(() => {
  setQrB64(null)
  setQrOpen(false)
  setQrError(null)
}, [url, code])
```
and the analogous reset keyed on `writeGateUrl`/`writeGateCode` in `InternetFullAccessTab`.

### WR-05: `HoldToConfirmButton`'s `disabled` gate is only checked at hold-start, not re-verified when the hold completes

**File:** `frontend/src/components/SessionShare/shared.tsx:56-71`; `frontend/src/components/SessionShare/InternetFullAccessTab.tsx:198-201`
**Issue:** `startHold()` guards with `if (disabled || holding) return` only at the moment the pointer/key goes down. The completion timeout (`setTimeout(..., HOLD_DURATION_MS)`) unconditionally calls `onConfirm()` after 3s regardless of whether `disabled` has since become `true` — e.g. if `funnelActive` flips false mid-hold (Funnel disabled from another device, or the 30s warm-up genuinely times out, exactly during the user's 3-second hold). This is explicitly the control called out in the review scope as "the sole affordance that can mint a public write capability," so its completion path should re-validate the gating condition, not only the start condition. Server-side authorization is presumably the true backstop, but the client should not fire the mint RPC (`SetSessionFunnelWrite`) once its own displayed precondition (`funnelActive && !warmingUp`) is already known to be false.
**Fix:** Re-check the current `disabled` value (via a ref updated each render, since `disabled` is a prop and the closure over it in the `setTimeout` callback is otherwise stale) before invoking `onConfirm`:
```tsx
const disabledRef = useRef(disabled)
disabledRef.current = disabled
...
timeoutRef.current = setTimeout(() => {
  clearTimers()
  setProgress(100)
  setHolding(false)
  if (!disabledRef.current) onConfirm()
}, HOLD_DURATION_MS)
```

### WR-06: LAN-password effect lacks the cancelled-guard pattern used by every other async effect in this file

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:176-182`
**Issue:** Every other async effect in this file (seeding, restart-reissue, warm-up completion) uses a `cancelled` flag to discard out-of-order responses. The LAN-password effect does not:
```tsx
useEffect(() => {
  if (webServerMode === 'local' && webServerRunning) {
    GetLocalNetworkPassword().then(setLanPassword).catch(() => setLanPassword(''))
  } else {
    setLanPassword('')
  }
}, [webServerMode, webServerRunning])
```
If `webServerMode`/`webServerRunning` flip quickly (e.g. local → tailscale → local), an in-flight `GetLocalNetworkPassword()` promise from the first `local` pass can resolve after the second `else` branch has already cleared `lanPassword`, re-populating it with a stale password display for a mode transition that no longer applies.
**Fix:** Add the same `cancelled` guard used elsewhere in the file:
```tsx
useEffect(() => {
  let cancelled = false
  if (webServerMode === 'local' && webServerRunning) {
    GetLocalNetworkPassword().then((pw) => { if (!cancelled) setLanPassword(pw) }).catch(() => { if (!cancelled) setLanPassword('') })
  } else {
    setLanPassword('')
  }
  return () => { cancelled = true }
}, [webServerMode, webServerRunning])
```

### WR-07: Copy-feedback timers (`setTimeout(..., 1500)`) are never cleared on unmount

**File:** `frontend/src/components/SessionShare/shared.tsx:150` (`CodeDisplay`); `frontend/src/components/SessionShare/ShareLinkCard.tsx:71` (`handleCopy`); `frontend/src/components/SessionShare/InternetFullAccessTab.tsx:127` (`handleCopy`)
**Issue:** All three "Copy" handlers call `setTimeout(() => setCopied(false), 1500)` with no ref/cleanup, and no effect return to clear them if the component unmounts before the timer fires (e.g. the user copies a link, then immediately switches tabs or closes the modal). React 18 no-ops a state update on an unmounted component rather than throwing, so this is not a crash risk, but it is a real "effect cleanup" gap explicitly called out in this review's focus areas, and is unlike the rest of the codebase's timer-cleanup discipline (every other timer in these same files — `HoldToConfirmButton`'s interval/timeout, the modal's warm-up timeout, the countdown interval — is properly tracked in a ref and cleared on unmount/re-arm).
**Fix:** Track the timeout in a ref and clear it on unmount, mirroring the pattern already used for `warmupTimeoutRef` in `SessionShareModal.tsx`.

## Info

### IN-01: `joinURLFor` and the QR-toggle-fetch-cache logic are duplicated verbatim across two files

**File:** `frontend/src/components/SessionShare/ShareLinkCard.tsx:26-29`, `:74-90`; `frontend/src/components/SessionShare/InternetFullAccessTab.tsx:29-32`, `:133-150`
**Issue:** `joinURLFor` is defined identically in both files (both explicitly say "Reused verbatim from SessionSharePanel.tsx" in their comments), and the QR toggle/fetch/cache logic (`handleToggleQR` vs `handleToggleGateQR`) is structurally identical apart from variable names. This is exactly the kind of duplication `shared.tsx` was created to eliminate for `HoldToConfirmButton`/`CodeDisplay`.
**Fix:** Hoist `joinURLFor` (and optionally a `useQRToggle(url, code)` hook) into `shared.tsx` so both consumers import one implementation — also closes WR-04 in one place instead of two.

### IN-02: Callback parameter shadows outer state variable of the same name

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:338`, `:485`, `:803`
**Issue:** The Funnel risk-panel's `expirySeconds` state (`const [expirySeconds, setExpirySeconds] = useState(3600)`, line 338) is shadowed by the `handleGateConfirm(expirySeconds: number)` parameter (line 485) and the inline callback `onGateConfirm={(expirySeconds) => void handleGateConfirm(expirySeconds)}` (line 803). Not a functional bug — both are correctly scoped and never cross-reference the outer value — but it is confusing to read two semantically distinct "expiry" values (the read-Funnel auto-expire preset vs. the write-gate's independent expiry) sharing an identifier, especially given this file already has a separate, differently-named `gateExpirySeconds` concept surfaced from `InternetFullAccessTab`.
**Fix:** Rename to `writeExpirySeconds` (or similar) in the callback signature and call site.

### IN-03: `prefersReducedMotion` detection is duplicated verbatim in three places

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:128-131`; `frontend/src/components/SessionShare/shared.tsx:34-37`; (pre-existing) `frontend/src/components/Hub/HubModal.tsx`
**Issue:** The same `typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches` expression is repeated in both new phase-173 files (in addition to its pre-existing copy in `HubModal.tsx`). Phase 173 was an opportunity to consolidate this into a shared hook rather than propagate the copy-paste further.
**Fix:** Extract a `usePrefersReducedMotion()` hook (e.g. in a shared `hooks/` module) for all three call sites to consume.

### IN-04: `CodeDisplay` retains hardcoded hex colors, inconsistent with this phase's own "no hardcoded hex" convention

**File:** `frontend/src/components/SessionShare/shared.tsx:160`, `:166`, `:170`
**Issue:** `CodeDisplay` renders with inline `style={{ color: '#9aa5ce', ... }}` / `background: '#16161e'` / `color: '#c0caf5'`, while every other phase-173 CSS addition in `style.css` (`.share-linkcard*`, `.share-seg*`) is explicitly annotated "All colors via var(--hub-*) tokens only." This is intentional per the phase context (D-09 explicitly says "PRESERVE as-is: HoldToConfirmButton, CodeDisplay"), so it is not a regression, but it does mean the modal now has one visibly inconsistent styling approach (raw hex inline vs. CSS custom properties) sitting directly inside the newly all-token-based `ShareLinkCard`.
**Fix:** Non-blocking; flag for a future cleanup pass to migrate `CodeDisplay` onto `--hub-*` tokens and CSS classes to match the rest of the redesign.

---

_Reviewed: 2026-07-08T13:24:37Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
