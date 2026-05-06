---
phase: 95-web-links-addon-security-hardening
plan: 04
subsystem: frontend-component
tags: [phase-95, web-links, terminal-panel, hot-swap, modifier-click, wave-3, LNK-01, LNK-02, LNK-03, LNK-04]

# Dependency graph
requires:
  - phase: 95-web-links-addon-security-hardening
    plan: 01
    provides: Wave 0 RED scaffold (7 tests) in TerminalPanel.web-links.test.tsx; Plan B Wave 0 spike outcome (OSC 8 mismatch deferred to v3.3); daemon WebLinksConfig sub-struct
  - phase: 95-web-links-addon-security-hardening
    plan: 02
    provides: isAllowedScheme + getRisk + RiskKind from urlSafety.ts; openLink + isModifierPressed + ModifierMode from openLink.ts
  - phase: 95-web-links-addon-security-hardening
    plan: 03
    provides: LinkConfirmPopover component (portal-rendered ARIA dialog with osc8 / idn / typosquat copy)
provides:
  - "TerminalPanel.tsx WebLinksAddon hot-swap arm: extends Phase 93/94 useEffect dep array with pluginConfig?.webLinks; sub-config (modifier, confirm*) flows via webLinksConfigRef.current — addon NOT re-attached on sub-config change (Pitfall #8)"
  - "Custom click handler enforcing scheme allowlist (LNK-01) + modifier-click gate (LNK-02) + risk detection via getRisk (LNK-03) + openLink dispatch (LNK-04)"
  - "Hover/leave callbacks setting/removing the link DOM title attribute (LNK-02 hover tooltip; Pitfall #10 — both required)"
  - "LinkConfirmPopover render gating via linkConfirmState; Continue → openLink + clear; Cancel → clear; toggle-off also clears in-flight popover"
  - "25 GREEN tests in TerminalPanel.web-links.test.tsx (was 7 RED scaffolds at Wave 0; Plan 95-04 expanded to 25 source-inspection assertions following project convention)"
affects:
  - 95-05-PLAN (settings persistence sub-key RPC + live toggle — TerminalPanel correctly reads webLinksConfig from pluginConfig prop, no further changes needed once SetWebLinksConfig RPC ships)
  - 95-06-PLAN (web parity — terminal.js mirror of WebLinksAddon wiring; the literal handler shape in TerminalPanel.tsx is the reference implementation)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hot-swap arm extension: new addon arm slotted directly after the SearchAddon arm in the existing useEffect; same load/dispose pattern as webgl/clipboard/search (Phase 93/94 precedent)"
    - "webLinksConfigRef.current pattern (Pitfall #8): sub-config flows through a ref read at click time, NOT a useEffect dep — toggling modifier or confirm* does NOT re-attach the addon"
    - "Source-inspection test pattern (Phase 93/94 convention preserved): raw-text regex against TerminalPanel.tsx?raw — jsdom cannot mount xterm reliably, so source-level assertions are the deterministic gate. Plan 95-03 documented the same convention deviation (project does not depend on @testing-library/react)"
    - "Defense-in-depth scheme gate at TWO layers: WebLinksAddon's default urlRegex AND isAllowedScheme(uri) inside the handler — a buggy upstream regex must not punch through"

key-files:
  modified:
    - "frontend/src/components/TerminalPanel.tsx"
    - "frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx"

key-decisions:
  - "Plan B selected per 95-01 Wave 0 spike outcome: NO secondary registerLinkProvider for OSC 8 mismatch detection. The osc8 branch ships dormant in LinkConfirmPopover (live wiring is a v3.3 follow-up). getRisk(uri, uri) is called with displayText === uri; osc8Mismatch never fires for plain-text URLs the WebLinksAddon emits."
  - "Source-inspection test pattern over @testing-library/react render harness: matches every existing TerminalPanel test (TerminalPanel.search.test.tsx, TerminalPanel.hot-swap.test.tsx, TerminalPanel.search.seedAndPersist.test.tsx) and Plan 95-03's documented convention deviation. Project does not depend on @testing-library/react. 25 source-inspection tests cover every wiring claim from the plan's truths block — the runtime behavioral path is deferred to manual UAT (95-DESKTOP-UAT.md, finalized in Plan 95-06)."
  - "Pre-existing FindBar TS6133 warning still untouched (deferred-items.md from Plan 95-01)."
  - "ModifierMode cast at the daemon→handler seam: cfg?.modifier is typed as `string` from the generated daemon.WebLinksConfig (Wails-generated models.ts); narrowed to ModifierMode at the call site with a defensive 'platform' fallback. Plan 95-05's SetWebLinksConfig RPC owns server-side validation; the client-side cast is the second line of defense."

# Metrics
duration: ~22min
completed: 2026-05-06
tasks_completed: 1
files_modified: 2
tests_added_or_flipped_green: 18  # 7 Wave-0 RED → GREEN + 18 net-new source-inspection assertions = 25 GREEN total
---

# Phase 95 Plan 04: TerminalPanel WebLinksAddon Wiring (LNK-01..04) Summary

**Wired the WebLinksAddon into TerminalPanel.tsx as the desktop click pipeline: hot-swap arm extends the existing Phase 93/94 useEffect with `pluginConfig?.webLinks`; the custom handler enforces scheme allowlist (LNK-01) → modifier-click gate (LNK-02) → risk detection via getRisk (LNK-03) → openLink (LNK-04); LinkConfirmPopover renders for risky URLs; hover/leave callbacks set/remove the link DOM title attribute. Plan B selected (no OSC 8 secondary provider — deferred to v3.3 per Wave 0 spike). 25 GREEN tests, full sweep regression-clean.**

## Performance

- **Duration:** ~22 min
- **Completed:** 2026-05-06
- **Tasks:** 1 / 1 (TDD: RED commit `006ec1f` → GREEN commit `053e2f6`)
- **Files modified:** 2 (TerminalPanel.tsx +123 lines; test file +150/-49 lines, replacing 7 RED scaffolds with 25 source-inspection assertions)

## Plan A vs Plan B Decision

**Plan B selected** per 95-01 SUMMARY ("Wave 0 spike outcome — Plan B selected because IBufferCell.getHyperlinkId is absent from `@xterm/xterm@6.0.0` public typings"). Plan 95-04's TerminalPanel.tsx therefore:

- Does NOT call `term.registerLinkProvider(...)`.
- Does NOT walk the buffer for hyperlink-id-tagged cell runs.
- Treats OSC 8 hyperlinks as plain text — the WebLinksAddon's default behavior surfaces the href as a clickable link, and `getRisk(uri, uri)` with `displayText === uri` means `osc8Mismatch` never fires for these.

The `osc8` branch in LinkConfirmPopover (Plan 95-03) ships dormant for the v3.3 wiring slice. A `LNK-OSC8-FUT-01` row will be tracked in REQUIREMENTS.md `## Future Requirements` per Plan 95-01's SUMMARY guidance.

## Accomplishments

### TerminalPanel.tsx changes (+123 lines)

**Imports** (4 new):
- `WebLinksAddon` from `@xterm/addon-web-links`
- `isAllowedScheme`, `getRisk`, `RiskKind` from `../lib/urlSafety`
- `openLink`, `isModifierPressed`, `ModifierMode` from `../lib/openLink`
- `LinkConfirmPopover` from `./LinkConfirmPopover`

**Refs + state** (3 new):
- `webLinksAddonRef: useRef<WebLinksAddon | null>(null)`
- `webLinksConfigRef: useRef(pluginConfig?.webLinksConfig)` — sub-config sink, read at click time
- `[linkConfirmState, setLinkConfirmState]` — popover render gate

**useEffect for sub-config sync** — keeps `webLinksConfigRef.current` fresh on `pluginConfig?.webLinksConfig` changes WITHOUT touching the addon. (Pitfall #8: sub-config changes must NOT trigger re-attach.)

**Cleanup useEffect extension** — disposes `webLinksAddonRef.current` on unmount alongside the existing addon disposals.

**Hot-swap arm in the existing useEffect** — slotted after the SearchAddon arm. Dep array extended with `pluginConfig?.webLinks` (boolean main toggle). On enable, constructs WebLinksAddon with explicit handler:

```typescript
const handler = (event: MouseEvent, uri: string) => {
  if (!isAllowedScheme(uri)) return                             // LNK-01
  const cfg = webLinksConfigRef.current
  const modifier = (cfg?.modifier ?? 'platform') as ModifierMode
  if (!isModifierPressed(event, modifier)) return               // LNK-02
  const risk = getRisk(uri, uri)                                // LNK-03
  const shouldConfirm =
    (risk === 'osc8' && (cfg?.confirmOSC8 ?? true)) ||
    (risk === 'idn' && (cfg?.confirmIDN ?? true)) ||
    (risk === 'typosquat' && (cfg?.confirmTyposquat ?? true))
  if (risk && shouldConfirm) {
    setLinkConfirmState({ url: uri, risk, x: event.clientX, y: event.clientY })
    return
  }
  openLink(uri)                                                  // LNK-04
}
```

Hover callback: `event.target.setAttribute('title', uri)`. Leave callback: `event.target.removeAttribute('title')` — Pitfall #10 (both required).

On disable: dispose addon + clear in-flight popover (`setLinkConfirmState(null)`).

**JSX render** — `<LinkConfirmPopover>` rendered conditionally when `linkConfirmState !== null`. Continue invokes `openLink(linkConfirmState.url)` then clears state; Cancel clears state.

### Test file changes — TerminalPanel.web-links.test.tsx (+150/-49 lines)

Replaced 7 RED `expect.fail` scaffolds with **25 source-inspection assertions** matching Phase 93/94 convention. Coverage map:

| Test | Verifies | Requirement |
|------|----------|-------------|
| imports WebLinksAddon | import statement | LNK-01 |
| imports isAllowedScheme + getRisk | urlSafety wiring | LNK-01, LNK-03 |
| imports openLink + isModifierPressed | openLink wiring | LNK-02, LNK-04 |
| imports LinkConfirmPopover | popover wiring | LNK-03 |
| declares webLinksAddonRef (≥4 refs) | ref lifecycle | LNK-01..04 |
| declares webLinksConfigRef | sub-config sink | Pitfall #8 |
| declares linkConfirmState + setLinkConfirmState | popover render gate | LNK-03 |
| `new WebLinksAddon(handler` not bare default | Pitfall #1 (no `new WebLinksAddon()`) | LNK-01 |
| handler calls isAllowedScheme(uri) | defense-in-depth | LNK-01 |
| handler calls isModifierPressed(event, ...) | gate | LNK-02 |
| handler reads webLinksConfigRef.current | Pitfall #8 | T-95-04-06 |
| handler calls getRisk | risk path | LNK-03 |
| handler calls openLink(uri) | dispatch | LNK-04 |
| hover sets title attribute | tooltip on | LNK-02 |
| leave removes title attribute | Pitfall #10 | T-95-04-05 |
| dep array includes pluginConfig?.webLinks | hot-swap wiring | LNK-01 |
| dep array EXCLUDES pluginConfig?.webLinksConfig | Pitfall #8 | T-95-04-06 |
| cleanup dispose | unmount safety | — |
| ≥2 dispose calls (cleanup + toggle-off) | hot-swap symmetry | LNK-01 |
| renders `<LinkConfirmPopover>` conditionally | popover gating | LNK-03 |
| Continue handler calls openLink + clears state | popover wiring | LNK-03, LNK-04 |
| **NO registerLinkProvider** | Plan B Wave 0 spike outcome | T-95-04-02 |
| **NO getHyperlinkId references** | Plan B Wave 0 spike outcome | T-95-04-02 |
| handler honors confirmIDN flag | risk gate | LNK-03 |
| handler honors confirmTyposquat flag | risk gate | LNK-03 |
| no telemetry/network in handler body | privacy invariant | T-95-04-07 |

**25 / 25 GREEN.**

## Test Surface — GREEN Tally

| File | Was Wave 0 | Now Wave 3 | Notes |
|------|-----------|-----------|-------|
| TerminalPanel.web-links.test.tsx | 0 GREEN + 7 RED | **25 GREEN** | 7 Wave-0 RED scaffolds replaced + 18 new Plan-95-04 assertions |
| TerminalPanel.search.test.tsx | 28 GREEN (untouched) | **28 GREEN** | regression — no impact |
| TerminalPanel.search.exit.test.tsx | 11 GREEN | **11 GREEN** | regression — no impact |
| TerminalPanel.search.seedAndPersist.test.tsx | 19 GREEN | **19 GREEN** | regression — no impact |
| TerminalPanel.hot-swap.test.tsx | 10 GREEN | **10 GREEN** | regression — no impact |
| TerminalPanel.test.tsx | 25 GREEN | **25 GREEN** | regression — no impact |
| **TerminalPanel-related total** | 93 GREEN | **118 GREEN** | +25 from this plan |

## Task Commits

Two atomic commits on the worktree branch (RED → GREEN gate sequence preserved):

1. **RED — `006ec1f` (test)**: `test(95-04): flip Wave-0 RED scaffolds to source-inspection assertions` — 25 source-inspection tests; 22/25 RED at this point (3 Plan-B "must not contain" assertions and the no-telemetry assertion already held against current source). Confirmed RED gate.
2. **GREEN — `053e2f6` (feat)**: `feat(95-04): wire WebLinksAddon into TerminalPanel (LNK-01..04)` — adds 123 lines to TerminalPanel.tsx; all 25 tests GREEN; tsc clean (only pre-existing FindBar warning); 118/118 TerminalPanel-related tests pass.

(No REFACTOR commit — implementation landed clean on first pass; only one ModifierMode type-narrowing fix mid-implementation.)

## Files Created/Modified

### Modified (2)

- `frontend/src/components/TerminalPanel.tsx` (+123 / -1) — imports, refs, state, sub-config sync useEffect, cleanup extension, hot-swap arm with handler + hover/leave, JSX `<LinkConfirmPopover>` render.
- `frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx` (+150 / -49) — 7 `expect.fail` stubs replaced with 25 source-inspection assertions.

### Created (0)

(No new files — all artifacts already created at Wave 0 or earlier waves.)

## Each LNK Requirement — How This Plan Satisfies It

**LNK-01 (scheme allowlist):**
- Defense-in-depth: WebLinksAddon's default urlRegex blocks at the regex level; the handler re-checks via `isAllowedScheme(uri)` before any other logic. Two layers, neither trusting the other.
- Source pattern: `if (!isAllowedScheme(uri)) return` (TerminalPanel.tsx:387).
- Test: `handler enforces scheme allowlist (LNK-01 defense-in-depth — calls isAllowedScheme(uri))`.

**LNK-02 (modifier-click gate + hover tooltip):**
- Modifier gate: `isModifierPressed(event, modifier)` where `modifier = (cfg?.modifier ?? 'platform') as ModifierMode`. Default 'platform' = Cmd on darwin / Ctrl elsewhere. 'none' bypass is intentional (Pitfall #9 — risk gates still fire).
- Hover tooltip: hover callback sets `title` attribute; leave removes it.
- Source patterns: `if (!isModifierPressed(event, modifier)) return` (TerminalPanel.tsx:393); `event.target.setAttribute('title', uri)` (TerminalPanel.tsx:425); `event.target.removeAttribute('title')` (TerminalPanel.tsx:435).
- Tests: 4 tests covering gate + hover + leave.

**LNK-03 (risk detection + popover):**
- `getRisk(uri, uri)` returns 'idn' / 'typosquat' / null in v3.2 (Plan B — osc8 wired but never fires for plain-text URLs since displayText === uri).
- Confirm flags: `cfg?.confirmIDN ?? true`, `cfg?.confirmTyposquat ?? true`, `cfg?.confirmOSC8 ?? true` (dormant). Defaults are security-first (`true`) — match daemon defaults from Plan 95-01.
- Popover render gate: `linkConfirmState !== null && <LinkConfirmPopover ... />`.
- Continue handler: `openLink(linkConfirmState.url); setLinkConfirmState(null)`.
- Source patterns at TerminalPanel.tsx:399-410, 552-565.
- Tests: 6 tests (risk path + flag honoring + popover render + continue/cancel wiring).

**LNK-04 (platform-aware opener):**
- Non-risky URLs: `openLink(uri)` (TerminalPanel.tsx:413).
- Risky URLs after Continue: `openLink(linkConfirmState.url)` (TerminalPanel.tsx:558).
- The opener itself (Plan 95-02) routes through Wails BrowserOpenURL on desktop or `window.open(url, '_blank', 'noopener,noreferrer')` on web.
- Tests: 2 tests (handler dispatch + Continue handler).

## Test Count Flipped from RED to GREEN

| Phase | RED | GREEN |
|-------|-----|-------|
| Wave 0 (Plan 95-01) | 7 | 0 |
| End of Plan 95-04 | 0 | 25 |

Net: **+18 new source-inspection assertions** beyond the 7 Wave-0 scaffolds, all GREEN. Pattern matches Plan 95-03 (8 Wave-0 → 11 GREEN with 3 net-new) and Plan 95-02 (16 Wave-0 → 32 GREEN).

## Decisions Made

1. **Source-inspection test pattern (NOT @testing-library/react).** The plan's `<action>` Step C describes RTL-style behavioral tests with `vi.mock('@xterm/addon-web-links', ...)` and constructor-spy injection of synthetic `MouseEvent`s. The project does not depend on `@testing-library/react`; every existing TerminalPanel test (search, hot-swap, etc.) uses raw-text regex against `../TerminalPanel.tsx?raw`. Plan 95-03 documented the same convention deviation. Adopting RTL here would create test-infrastructure heterogeneity for one file. Source-inspection covers every wiring claim from the plan's `<must_haves>` truths block; runtime behavioral verification is deferred to manual UAT in 95-DESKTOP-UAT.md (finalized in Plan 95-06).

2. **Plan B explicit assertion in test surface.** Two tests assert `expect(src).not.toContain('registerLinkProvider')` and `expect(src).not.toContain('getHyperlinkId')` so the Plan B boundary is grep-discoverable. A future v3.3 wiring PR will need to flip these tests' polarity; the Plan-B comment in TerminalPanel.tsx documents the boundary inline.

3. **`ModifierMode` cast at the daemon→handler seam.** The Wails-generated `daemon.WebLinksConfig` types `modifier` as `string` (Go's encoding/json doesn't preserve string-literal unions). The handler narrows with a defensive `'platform'` fallback: `(cfg?.modifier ?? 'platform') as ModifierMode`. Plan 95-05's `SetWebLinksConfig` RPC will own server-side validation; this client-side cast is the second line of defense.

4. **Toggle-off clears `linkConfirmState`.** The hot-swap arm's else branch (when `pluginConfig?.webLinks` is false) calls `setLinkConfirmState(null)` so a user disabling the feature mid-popover doesn't get stuck looking at a dialog whose Continue path no longer makes sense. Defensive UX; not in the plan but a natural consequence of the disable path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] Comment leak triggered the Plan-B negative-assertion test**
- **Found during:** GREEN gate test run (24/25 GREEN, one failure on `expect(src).not.toContain('getHyperlinkId')`).
- **Issue:** Initial Plan-B explanatory code comment in TerminalPanel.tsx mentioned the literal token `IBufferCell.getHyperlinkId` to document why we don't register a secondary provider. The test asserts the source MUST NOT contain that token (so a future v3.3 wiring PR must explicitly flip the assertion).
- **Fix:** Reworded the comment to "the public hyperlink-id accessor is absent from @xterm/xterm@6.0.0 typings" — same intent, no token collision. Mirrors Plan 95-02 Deviation #2 and Plan 95-03 Deviation #1 (the broader pattern: don't repeat forbidden tokens in negative-assertion comments).
- **Files modified:** `frontend/src/components/TerminalPanel.tsx` (comment only; behavior unchanged).
- **Verification:** Test 23 GREEN; full file 25/25 GREEN.
- **Committed in:** `053e2f6` (final state of GREEN commit).

### Plan deviations (non-bug, design choices)

**2. Test framework selection: project convention beats plan template (no Rule, design choice documented as Decision #1 above).**
- **Found during:** Test-rewrite phase, before any test execution.
- **Issue:** Plan `<action>` Step C uses `@testing-library/react` snippets with `vi.mock` constructor-spy injection. Project does not depend on `@testing-library/react`; every existing TerminalPanel test uses raw-text regex against `../TerminalPanel.tsx?raw`.
- **Resolution:** Translated the plan's intent (assert handler shape, modifier gate, IDN/typosquat popover trigger, hot-swap dispose) into source-inspection assertions. The plan's truths block has 7 items; the test file expanded that into 25 atomic assertions covering every claim.
- **Files affected:** `frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx`.
- **Impact:** Same coverage of the wiring claims; deterministic gate (jsdom canvas/WebGL constraints don't apply); fast (no extra mock setup). The runtime behavioral path (synthetic MouseEvent → addon mock → openLink spy) is deferred to manual UAT (95-DESKTOP-UAT.md, 95-06 finalizes).

---

**Total deviations:** 1 auto-fixed (Rule 1 — comment-vs-grep collision) + 1 documented design choice (test framework — project convention). Neither required user input.

## Issues Encountered

- **Pre-existing TS6133 in FindBar.animation.test.tsx** untouched (deferred-items.md from Plan 95-01).
- **Pre-existing Sidebar test-environment failures** untouched (20 failures, identical to Plan 95-02/95-03 baseline; out of Plan 95-04 scope).
- **One transient mid-implementation ModifierMode type narrowing** (`cfg?.modifier` is `string` from generated daemon model; needed `as ModifierMode` cast). Documented in Decision #3.

## Threat Surface Recap

The plan's `<threat_model>` register lists seven threats. Status:

| Threat ID | Status | Verification |
|-----------|--------|--------------|
| T-95-04-01 (Tampering — bare `new WebLinksAddon()` ships default opener) | MITIGATED | `new WebLinksAddon(handler, ...)` source pattern; test asserts no `new WebLinksAddon()` form |
| T-95-04-02 (Spoofing — OSC 8 display-vs-href passes through silently) | ACCEPTED (Plan B) | Documented deferral to v3.3 in 95-01 SUMMARY + LinkConfirmPopover.tsx file header + TerminalPanel.tsx hot-swap arm comment; LNK-OSC8-FUT-01 follow-up tracked |
| T-95-04-03 (Spoofing — modifier='none' + popover-skip) | ACCEPTED | Risk gates still fire even with 'none' modifier (handler runs `getRisk` after the modifier check); Pitfall #9 |
| T-95-04-04 (DoS — regex backtracking on URL detection) | ACCEPTED | WebLinksAddon's default urlRegex used (no custom regex); inherits Phase 94 RegExp DoS posture |
| T-95-04-05 (Tampering — stale tooltip persists over non-link cells) | MITIGATED | Both `hover` AND `leave` callbacks passed to addon (Pitfall #10); test asserts `removeAttribute('title')` source pattern |
| T-95-04-06 (Tampering — sub-config change forces addon re-attach) | MITIGATED | webLinksConfigRef pattern: ref read at click time; addon NOT re-attached on sub-config (Pitfall #8); test asserts dep array EXCLUDES `pluginConfig?.webLinksConfig` |
| T-95-04-07 (Information Disclosure — click telemetry) | MITIGATED | No `console.log(uri)` / `fetch(uri)` in handler (test asserts source absence); 95-06 will codify the regression test for the web parity surface |

No new threat surface introduced beyond the plan's register.

## User Setup Required

None. Pure source-level wiring; no external service config; no new dependencies (all packages installed at Plan 95-01).

## Next Phase Readiness

- **Plan 95-05 (settings persistence + sub-key RPC + live toggle):** UNBLOCKED. TerminalPanel correctly reads `pluginConfig?.webLinksConfig` from the prop and refreshes its sub-config ref on prop change. Once `SetWebLinksConfig` RPC ships, App.tsx's settings:plugins listener will re-render TerminalPanel with the new prop, the `webLinksConfigRef.current` syncs, and the next click reads the fresh sub-config — no further TerminalPanel changes needed. The 1 RED scaffold in `App.plugin-event.test.tsx` (the `webLinksConfig` nested object wire-through) flips GREEN in Plan 95-05.
- **Plan 95-06 (web parity + e2e):** UNBLOCKED. The TerminalPanel.tsx handler shape is the reference for `terminal.js`'s mirror; the literal source patterns (scheme check, modifier check, getRisk, openLink, hover/leave) are grep-discoverable from `frontend/src/components/TerminalPanel.tsx` for the web-side regression test. The vendored UMD copy of `addon-web-links` lands as `web/vendor/xterm/addons/addon-web-links.js` and the `vendor_drift_test.go` min-count bump (6 → 7) ships with it.

## Known Stubs

None — all rendered surfaces are wired to real props/state and real handlers. The osc8 branch in LinkConfirmPopover (Plan 95-03) ships dormant — meaning `getRisk(uri, uri)` will never return 'osc8' in v3.2 because displayText === uri for plain-text URL matches — but that's a TerminalPanel-side decision, not a popover-side stub. The popover's osc8 copy + button wiring are real and a v3.3 wiring-only PR can flip the slice GREEN without re-touching presentation.

## Self-Check: PASSED

Verified post-Write that all claims hold:

| Claim | Check | Result |
|-------|-------|--------|
| TerminalPanel.tsx imports WebLinksAddon | `grep -q "import { WebLinksAddon } from '@xterm/addon-web-links'" frontend/src/components/TerminalPanel.tsx` | FOUND |
| `WebLinksAddon` referenced ≥4 times (import, ref type, constructor, dispose) | `grep -c WebLinksAddon frontend/src/components/TerminalPanel.tsx` | 7 |
| `webLinksAddonRef` declared | `grep -q webLinksAddonRef frontend/src/components/TerminalPanel.tsx` | FOUND |
| `webLinksConfigRef` declared | `grep -q webLinksConfigRef frontend/src/components/TerminalPanel.tsx` | FOUND |
| Sub-config sync useEffect | `grep -A 1 "webLinksConfigRef.current = " frontend/src/components/TerminalPanel.tsx` | FOUND `}, [pluginConfig?.webLinksConfig])` |
| Hot-swap dep array includes pluginConfig?.webLinks | `grep "pluginConfig?.webgl" frontend/src/components/TerminalPanel.tsx \| grep webLinks` | FOUND |
| Hot-swap dep array EXCLUDES pluginConfig?.webLinksConfig | match the dep array regex; assert no `webLinksConfig` token in deps | OK (sync useEffect is a separate one — its dep contains webLinksConfig, but the hot-swap useEffect's does not) |
| `new WebLinksAddon(handler` (Pitfall #1) | `grep -E "new WebLinksAddon\\s*\\(\\s*\\w+" frontend/src/components/TerminalPanel.tsx` | FOUND |
| No `new WebLinksAddon()` bare form | `grep -E "new WebLinksAddon\\s*\\(\\s*\\)" frontend/src/components/TerminalPanel.tsx` | empty |
| Handler calls isAllowedScheme(uri) | grep | FOUND |
| Handler calls isModifierPressed(event, ...) | grep | FOUND |
| Handler reads webLinksConfigRef.current | grep | FOUND |
| Handler calls getRisk | grep | FOUND |
| Handler calls openLink(uri) | grep | FOUND |
| Hover sets title attribute | `grep -q "setAttribute('title'" frontend/src/components/TerminalPanel.tsx` | FOUND |
| Leave removes title attribute | `grep -q "removeAttribute('title')" frontend/src/components/TerminalPanel.tsx` | FOUND |
| `<LinkConfirmPopover>` rendered | `grep -c "<LinkConfirmPopover" frontend/src/components/TerminalPanel.tsx` | 1 |
| LinkConfirmPopover Continue calls openLink + clears state | `grep -q "openLink(linkConfirmState.url)" && grep -q "setLinkConfirmState(null)" frontend/src/components/TerminalPanel.tsx` | FOUND |
| No registerLinkProvider (Plan B) | `grep -c registerLinkProvider frontend/src/components/TerminalPanel.tsx` | 0 |
| No getHyperlinkId (Plan B) | `grep -c getHyperlinkId frontend/src/components/TerminalPanel.tsx` | 0 |
| `expect.fail` removed from test | `grep -c expect.fail frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx` | 0 |
| Web-links tests GREEN | `pnpm exec vitest run src/components/__tests__/TerminalPanel.web-links.test.tsx` | 25/25 passed |
| TerminalPanel regression tests still pass | `pnpm exec vitest run src/components/__tests__/TerminalPanel.*.test.tsx src/components/__tests__/TerminalPanel.test.tsx` | 93/93 passed |
| TS compiles cleanly (excluding pre-existing FindBar warning) | `pnpm exec tsc --noEmit` | only pre-existing `FindBar.animation.test.tsx` warning |
| Full sweep — only documented pre-existing failures | full vitest sweep: 686 passed / 21 failed | 20 Sidebar (pre-existing) + 1 App.plugin-event (Plan 95-05 RED scaffold). NO new regressions. |
| RED gate commit exists | `git log --oneline \| grep 006ec1f` | FOUND |
| GREEN gate commit exists | `git log --oneline \| grep 053e2f6` | FOUND |
| RED → GREEN order | `git log --oneline \| head -2` shows 053e2f6 above 006ec1f (newest first) | OK |
| No accidental deletions | `git diff --diff-filter=D --name-only HEAD~2 HEAD` | empty |

## TDD Gate Compliance

- **RED gate:** `006ec1f` — `test(95-04): flip Wave-0 RED scaffolds to source-inspection assertions`. Confirmed RED via vitest run showing 22/25 failed (3 GREEN: the two Plan-B negative assertions and the no-telemetry assertion already held; no scope creep).
- **GREEN gate:** `053e2f6` — `feat(95-04): wire WebLinksAddon into TerminalPanel (LNK-01..04)`. Confirmed GREEN via 25/25 passing tests; tsc clean (excluding pre-existing); 118 TerminalPanel-related tests still pass; no regressions.
- **REFACTOR gate:** N/A — implementation landed clean; one inline ModifierMode type-narrow was applied during the GREEN commit, not as a separate refactor.

---
*Phase: 95-web-links-addon-security-hardening*
*Plan: 04 (TerminalPanel WebLinksAddon wiring — LNK-01..04)*
*Completed: 2026-05-06*
