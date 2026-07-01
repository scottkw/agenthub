---
phase: 166-funnel-frontend-help-guide
verified: 2026-06-30T21:55:00Z
status: passed
score: 8/8 requirements verified
behavior_unverified: 0
overrides_applied: 0
gap_closure:
  - truth: "The risk dialog cross-links to the Help guide's contained-sharing alternative (FUI-06)"
    resolved_by: "commit fix(166): wire Funnel risk-panel Help cross-link to app navigation"
    fix: "Added onOpenHelp?: () => void to HubPanel Props; threaded onOpenHelp App.tsx → HubPanel → SessionShareModal (App passes () => handleOpenHelp('help-sharing')). App.handleOpenHelp now takes an optional sectionId and scrolls the Help tab to that section. Added a HubPanel integration test (HubPanel.test.tsx 'FUI-06: threads onOpenHelp through the Share modal risk-panel Help cross-link') that opens the real Share modal, flips the Funnel toggle, clicks the Help link, and asserts onOpenHelp fires — closing the exact unit-test blind spot the verifier flagged."
    verification: "cd frontend && npx tsc --noEmit (clean); npx vitest run (134 files / 2253 tests pass); pnpm run build (OK)."
gaps_original:
  - truth: "The risk dialog cross-links to the Help guide's contained-sharing alternative (FUI-06)"
    status: failed_then_fixed
    reason: "onOpenHelp prop is not wired from HubPanel to SessionShareModal. HubPanel renders <SessionShareModal> without passing onOpenHelp, so the 'See the Sharing Guide →' button in FunnelRiskPanel only closes the modal — it does NOT navigate to the Help tab. The unit test passes because it injects a mock onOpenHelp directly into SessionShareModal, but the real app integration is broken. RESOLVED — see gap_closure above."
    artifacts:
      - path: "frontend/src/components/Hub/HubPanel.tsx"
        issue: "Line 582-591: SessionShareModal rendered without onOpenHelp prop. HubPanel's Props interface (line 144-209) has no onOpenHelp slot."
      - path: "frontend/src/components/Hub/SessionShareModal.tsx"
        issue: "onOpenHelp is declared as optional (line 52) and used correctly at line 384 (handleOpenHelp), but receives undefined from HubPanel."
    missing:
      - "Add onOpenHelp?: () => void to HubPanel's Props interface"
      - "Thread onOpenHelp from App.tsx to HubPanel to SessionShareModal"
      - "App.tsx handleOpenHelp (line 814) additionally needs to scroll to help-sharing section, not just open the Help tab top-level — HelpTab.handleJumpToSection('help-sharing') or an initialSection prop"
deferred: []
human_verification:
  - test: "M-37: Live Funnel enable — TLS warm-up completes and the public URL opens from an off-tailnet device"
    expected: "After toggling Enable internet share and confirming, 'Starting up…' appears, then within ~30s the public URL is displayed and opens in an off-tailnet browser"
    why_human: "Requires a live tailnet with Funnel grant + a second device not on the tailnet. Cannot be automated."
  - test: "M-38: Live auto-expiry teardown at the chosen duration"
    expected: "When the chosen expiry elapses, the Funnel tears down automatically; the public URL stops responding; the internet badge clears on next poll"
    why_human: "Requires a live running Funnel and waiting for the expiry to trigger. Cannot be automated."
  - test: "M-39: Internet-exposure indicators appear while active and clear after disable"
    expected: "Hub card shows GlobeAltIcon + 'INTERNET' badge and session tab shows globe icon while Funnel is active; both clear within 3s after disable"
    why_human: "Requires a live daemon with funnelActive transitions flowing through the 3s poll. Automated tests mock this at component level."
  - test: "M-40: Local-fallback toggle is disabled when Tailscale is not the web-server mode"
    expected: "In a local web-server deployment, the Funnel toggle is greyed out and shows 'Internet sharing requires Tailscale'"
    why_human: "Requires a running app in local web-server mode. D-15 is test-asserted at unit level but needs live confirmation."
  - test: "Sharing guide content accuracy check — expiry durations"
    expected: "The sharing guide says '1 hour, 4 hours, 24 hours' but FunnelRiskPanel shows '30 minutes, 1 hour, 4 hours, 8 hours, Until I disable'. A human should confirm whether the guide should be updated to match the actual UI options."
    why_human: "Content accuracy judgment — the guide is factually incorrect about the max duration (says 24 hours, actual is 8 hours) and omits the 30-minute option."
---

# Phase 166: Funnel Frontend + Help Guide Verification Report

**Phase Goal:** Users can enable internet-sharing through a risk-aware UI flow, see a persistent non-color exposure indicator, and access an in-app guide covering both sharing paths.
**Verified:** 2026-06-30T21:55:00Z
**Status:** GAPS FOUND — 1 BLOCKER
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC1 | Funnel toggle shows risk dialog on every enable (no skip), with expiry selector and cross-link to Help | PARTIAL | Risk panel exists and fires every enable (tested); but the Help cross-link button closes the modal without navigating — `onOpenHelp` not wired from HubPanel |
| SC2 | Persistent colorblind-safe internet-exposure indicator on Hub card + session tab; verified at hex/source level | VERIFIED | `SessionCard.tsx:538-541` renders `GlobeAltIcon` + `<span>INTERNET</span>` conditional on `session.funnelActive`; `TabBar.tsx:241-248` renders globe icon with `aria-label="Internet exposed"`; dark `#43ddb2` / light `#0d7a5c` in `style.css:4654,4713` with COLORBLIND-SAFE comments naming GlobeAltIcon shape + text as state carriers |
| SC3 | Share modal shows Funnel URL with copy/QR; warm-up UX immediately after enable | VERIFIED | `SessionSharePanel.tsx:315-380` Internet (public) section; `SessionShareModal.tsx` warm-up state machine with 30s timeout; 17/17 panel + 31/31 modal tests pass |
| SC4 | One-click Funnel disable removes exposure and clears indicator immediately | VERIFIED | `handleDisableFunnel` clears timer then calls `SetSessionFunnel(id,false,0)`; test "clears the 30s timeout on disable" passes; indicator clears on next 3s poll |
| SC5 | Help tab includes Sharing Guide covering both Funnel path and device-share+ACL alternative, with ACL grant block and wildcard-default gotcha | VERIFIED | `sharing-guide.md` covers both paths; `autogroup:shared` → `tag:agenthub:tcp:7443` grant block; `*→*` gotcha explicitly called out; article registered in both SECTION_META (HelpTab.tsx:66) and SECTIONS (HelpSectionNav.tsx:14); 14/14 integration tests pass |

**Score: 4/5 ROADMAP SCs verified (SC1 partially failed on the cross-link navigation leg)**

### Per-Requirement Findings

| Req | Status | Evidence |
|-----|--------|----------|
| FUI-01 | VERIFIED | `FunnelRiskPanel.tsx` presentational component with verbatim risk statement; toggle ON → `setRiskPanelOpen(true)` only (no `SetSessionFunnel`); CTA calls `SetSessionFunnel`; 10/10 FunnelRiskPanel tests + 26/26+ SessionShareModal tests |
| FUI-02 | VERIFIED | `EXPIRY_OPTIONS` enum `[1800,3600,14400,28800,0]` (default 3600); `onExpiryChange` fires numeric value; committed to `SetSessionFunnel(id,true,expirySeconds)` via CTA |
| FUI-03 | VERIFIED | `SessionCard.tsx:538-541`: `.hub-internet-badge` with `GlobeAltIcon` (aria-hidden) + `<span class="hub-internet-badge__label">INTERNET</span>` driven by `session.funnelActive`; `TabBar.tsx:241-248`: `span.tab__internet-icon[aria-label="Internet exposed"]` + `GlobeAltIcon`; dark hex `#43ddb2` + light hex `#0d7a5c` in `style.css` with COLORBLIND-SAFE enforcement-only comments; `funnelActiveSessions` derived inline in `App.tsx:1508-1511` |
| FUI-04 | VERIFIED | `handleDisableFunnel` in `SessionShareModal.tsx:360-376`: clears warmupTimeoutRef first, calls `SetSessionFunnel(id,false,0)`, resets all funnel state; `SessionSharePanel.tsx:376-385`: single "Disable internet share" button, no confirm dialog (D-13); tested |
| FUI-05 | VERIFIED | `SessionSharePanel.tsx:313-390` Internet (public) section: warmup copy, timeout error, URL row with Copy URL (`ClipboardSetText`)/Open (`BrowserOpenURL`)/QR (`GetCapabilityQRCode`); `SessionShareModal.tsx` warm-up state machine; tested by 17 panel + multiple modal tests |
| FUI-06 | **FAILED (BLOCKER)** | `FunnelRiskPanel.tsx:82-88` "Want tighter containment? See the Sharing Guide →" button exists and calls `onOpenHelp()`; `SessionShareModal.tsx:381-385` `handleOpenHelp()` calls `handleClose()` + `onOpenHelp?.()` — BUT `HubPanel.tsx:582-591` renders `<SessionShareModal>` WITHOUT `onOpenHelp` prop; result: button clicks close the modal and navigate nowhere; unit test passes because it injects a mock directly into the component |
| HLP-01 | VERIFIED | `frontend/src/content/help/sharing-guide.md` covers both Funnel path (Option 1) and device-share + ACL path (Option 2); `HelpTab.tsx:66` + `HelpSectionNav.tsx:14` both register `help-sharing` entry |
| HLP-02 | VERIFIED | ACL grant block present (`src/dst: autogroup:shared → tag:agenthub:tcp:7443`); `*→*` wildcard-default gotcha at line 41; 4 Tailscale KB links as plain markdown; `grep -c '<a ' sharing-guide.md` = 0 (no raw anchor tags violating Wails CSP) |

**Score: 7/8 requirements verified**

---

## Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/wailsjs/go/main/App.d.ts` | funnelActive on SessionInfo + SetSessionFunnel export | VERIFIED | Line 23: `funnelActive: boolean`; line 212: `SetSessionFunnel(sessionID: string, enabled: boolean, expiresIn: number): Promise<void>` |
| `frontend/src/wailsjs/go/main/App.js` | SetSessionFunnel Call wrapper | VERIFIED | Line 135: `SetSessionFunnel = (sessionID, enabled, expiresIn) => Call('main.App.SetSessionFunnel', [sessionID, enabled, expiresIn])` |
| `frontend/src/style.css` | internet-badge tokens + all Phase-166 component classes | VERIFIED | `--hub-internet-badge-bg/text` in both dark/light; all 9+ classes present; COLORBLIND-SAFE count = 35 (was 33); motion guard correct |
| `frontend/src/components/__tests__/funnelBinding.contract.test.tsx` | 3-assertion contract test | VERIFIED | 3/3 pass (App.d.ts funnelActive, SetSessionFunnel signature, App.js Call wrapper) |
| `frontend/src/components/Hub/FunnelRiskPanel.tsx` | Presentational risk panel | VERIFIED | Risk statement, 5 expiry presets (default 3600), Help cross-link button, two action buttons |
| `frontend/src/components/Hub/SessionShareModal.tsx` | Funnel toggle + two-step gesture + fail-closed | VERIFIED | Toggle → panel → CTA pattern; D-15 fail-closed implemented; FUI-06 cross-link component present but not wired at app level |
| `frontend/src/components/SessionSharePanel.tsx` | Internet (public) section with URL/copy/QR/warmup/disable | VERIFIED | Full Internet section rendered; D-12 no-write-link enforced; 17/17 tests pass |
| `frontend/src/components/Hub/SessionCard.tsx` | Hub card internet badge | VERIFIED | Lines 538-541: conditional `.hub-internet-badge` with GlobeAltIcon + INTERNET text |
| `frontend/src/components/TabBar.tsx` | Session tab globe icon | VERIFIED | funnelActiveSessions prop; conditional `span.tab__internet-icon[aria-label="Internet exposed"]` |
| `frontend/src/App.tsx` | funnelActiveSessions derivation | VERIFIED | Lines 1508-1511: inline reduce over hubSessions passed to TabBar |
| `frontend/src/content/help/sharing-guide.md` | Sharing Guide article | VERIFIED | Both paths, ACL grant, wildcard gotcha, 4 KB links, no raw `<a>` |
| `frontend/src/components/HelpTab.tsx` | sharing-guide in SECTION_META | VERIFIED | Line 65-66: Phase 166-04 entry registered |
| `frontend/src/components/HelpSectionNav.tsx` | sharing-guide in SECTIONS | VERIFIED | Line 13-14: Phase 166-04 entry registered, in sync with HelpTab |
| `TESTING.md` | Section 2/4/5 Phase-166 rows + M-37..M-40 | VERIFIED | Suite manifest updated (132→134 vitest); 8 traceability rows (FUI-01..06 + HLP-01/02); M-37..M-40 manual items added; `check-traceability-paths.sh` exits 0 |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `App.tsx` | `TabBar.tsx` | `funnelActiveSessions` prop | WIRED | Lines 1508-1511 derive from `hubSessions.reduce`; passed to `<TabBar>` |
| `HubPanel.tsx` | `SessionShareModal.tsx` | `shareModalSession` poll-sync | WIRED | `useEffect` keyed on `[sessions, shareModalSession?.id]` keeps modal session live |
| `SessionShareModal.tsx` | `FunnelRiskPanel.tsx` | `riskPanelOpen` + callbacks | WIRED | Toggle → `setRiskPanelOpen(true)`; CTA via `handleFunnelEnable` |
| `SessionShareModal.tsx` | `SetSessionFunnel` (Wails) | explicit CTA only (D-02) | WIRED | `SetSessionFunnel(session.id, true, expirySeconds)` in `handleFunnelEnable` |
| `SessionShareModal.tsx` → `HubPanel.tsx` | `handleOpenHelp` (App.tsx) | `onOpenHelp` prop | **NOT WIRED** | HubPanel renders `<SessionShareModal>` without `onOpenHelp`; callback is undefined at runtime |
| `SessionShareModal.tsx` | `SessionSharePanel.tsx` | funnel props thread | WIRED | `funnelActive`, `funnelUrl`, `warmingUp`, `warmupTimedOut`, `onDisableFunnel` all threaded |
| `SessionSharePanel.tsx` | Wails IssueCapabilities/ClipboardSetText/BrowserOpenURL/GetCapabilityQRCode | on funnelActive flip | WIRED | URL re-issue, copy, open, QR all wired |
| `HelpTab.tsx` | `sharing-guide.md` | `?raw` import + `sharingMd` | WIRED | Line 14: `import sharingMd from '../content/help/sharing-guide.md?raw'`; used in SECTION_META |

---

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full frontend suite | `cd frontend && npx vitest run` | 134 files / 2252 tests PASS | PASS |
| TypeScript compile | `cd frontend && npx tsc --noEmit` | No output (clean) | PASS |
| Production build gate | `cd frontend && pnpm run build` | `built in 287ms` | PASS |
| funnelBinding contract test | Suite includes `funnelBinding.contract.test.tsx` 3/3 | PASS (seen in output) | PASS |
| Traceability paths script | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist` | PASS |
| D-12 no-write-link test | Suite includes `SessionSharePanel.test.tsx:269` | PASS (in suite) | PASS |
| D-15 local-fallback test | Suite includes `SessionShareModal.test.tsx:509` | PASS (in suite) | PASS |
| COLORBLIND-SAFE hex verified | `grep '#43ddb2' src/style.css` + `grep '#0d7a5c' src/style.css` | Both found as `--hub-internet-badge-text` | PASS |

---

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `SessionShareModal.tsx` | 272 | Stale comment: "placeholder flag consumed by Plan 05's warm-up UX" | Info | Comment is from Plan 02; warmingUp IS fully wired by Plan 05. Stale comment, not a stub. No impact. |
| `sharing-guide.md` | 11 | Content inaccuracy: says "1 hour, 4 hours, 24 hours" but UI shows "30 minutes, 1 hour, 4 hours, 8 hours" | Warning | Users reading the guide will see inaccurate max duration (24h vs 8h) and won't know about the 30-minute option. Should be corrected. |

No TBD/FIXME/XXX markers found in any Phase-166 modified files.

---

## FUI-06 Gap — Detail

**Root cause:** `onOpenHelp` is an optional prop on `SessionShareModal`. The component calls it correctly inside `handleOpenHelp()` (line 381-385). However, `HubPanel` (the only production render site of `SessionShareModal`, at line 582-591) does not pass `onOpenHelp`. The prop is therefore `undefined` at runtime.

**Consequence:** Clicking "Want tighter containment? See the Sharing Guide →" in the risk panel:
1. Calls `FunnelRiskPanel.onOpenHelp()` → `SessionShareModal.handleOpenHelp()`
2. `handleClose()` fires — modal closes
3. `onOpenHelp?.()` is a no-op (undefined)
4. User lands back on the Hub with no navigation to the Help tab

**Test gap:** `SessionShareModal.test.tsx:526-536` verifies the callback is called when `onOpenHelp` is provided via `renderModal({ onOpenHelp })`. This passes because the test injects the mock. It does NOT verify HubPanel threads the callback — that wiring is the integration gap.

**Fix required:**
1. Add `onOpenHelp?: () => void` to `HubPanel`'s Props interface
2. Pass it through from App.tsx's `handleOpenHelp` to HubPanel, then to `<SessionShareModal>`
3. Consider whether `handleOpenHelp` in App.tsx should also navigate to `help-sharing` section (currently opens Help tab at `help-getting-started` top). `HelpTab.handleJumpToSection` or an `initialSection` prop would be needed for direct scroll.

**Suggested override if intentionally deferred:**
```yaml
overrides:
  - must_have: "The risk dialog cross-links to the Help guide's contained-sharing alternative (FUI-06)"
    reason: "Cross-link button exists; Help navigation deferred to Phase 168 or post-launch polish"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

---

## Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| FUI-01 | 166-02 | Risk dialog on every Funnel enable, no skip | VERIFIED | FunnelRiskPanel + two-step gesture; tested |
| FUI-02 | 166-02 | Auto-expiry selector in risk dialog | VERIFIED | 5 presets [1800,3600,14400,28800,0], default 3600; tested |
| FUI-03 | 166-03 | Persistent colorblind-safe internet indicator | VERIFIED | GlobeAltIcon + INTERNET text + aria; source-verified hex |
| FUI-04 | 166-05 | One-click Funnel disable from Share UI | VERIFIED | Single button, no confirm, timer cleared; tested |
| FUI-05 | 166-05 | Public Funnel URL with copy/QR | VERIFIED | Internet (public) section with URL/copy/open/QR/warmup; tested |
| FUI-06 | 166-02 | Risk dialog cross-links to Help guide | FAILED | Button exists; `onOpenHelp` not wired from HubPanel to SessionShareModal |
| HLP-01 | 166-04 | Help article covering both sharing paths | VERIFIED | sharing-guide.md covers Funnel + device-share/ACL |
| HLP-02 | 166-04 | ACL grant block + wildcard-default gotcha + Tailscale doc links | VERIFIED | All present; no raw `<a>` tags |

---

## Human Verification Required

### 1. M-37: Live Funnel Enable + Off-Tailnet URL

**Test:** On a production build, enable internet sharing from the Share modal. Observe the "Starting up…" warm-up state, wait for the public URL to appear, then open it from a device not on your Tailscale network.
**Expected:** Warm-up completes within ~30s; public URL is accessible from off-tailnet browser; Join code gate is the only access control.
**Why human:** Requires a live Tailscale account with Funnel grant + production build + a second off-tailnet device.

### 2. M-38: Live Auto-Expiry Teardown

**Test:** Enable internet sharing with a short expiry (e.g., 30 minutes). Wait for the duration to elapse.
**Expected:** Funnel tears down automatically; public URL stops responding; internet badge clears on next Hub poll.
**Why human:** Requires a live running Funnel, real daemon event, and elapsed wall-clock time.

### 3. M-39: Indicators Appear + Clear with Funnel State

**Test:** Enable Funnel, observe Hub card badge and session tab icon appear. Disable, observe they clear.
**Expected:** Hub card shows GlobeAltIcon + "INTERNET" badge while active; session tab shows globe icon while active; both clear within 3s after disable.
**Why human:** Automated tests mock the funnelActive prop at component level; live behavior requires a running daemon.

### 4. M-40: Local-Fallback Toggle Disabled in Local Web-Server Mode

**Test:** Run the app with the web server in local (non-Tailscale) mode; open the Share modal.
**Expected:** Funnel toggle is greyed out and shows "Internet sharing requires Tailscale".
**Why human:** D-15 is unit-tested but live confirmation requires a running app in local web-server mode.

### 5. Sharing Guide Content Accuracy — Expiry Duration Discrepancy

**Test:** Read `sharing-guide.md` line 11 against the FunnelRiskPanel dropdown options.
**Expected:** Guide should list "30 minutes, 1 hour, 4 hours, 8 hours, or no auto-expiry" — not "1 hour, 4 hours, 24 hours".
**Why human:** Judgment call: whether to fix the guide (update MD content) or accept the inaccuracy.

---

## Gaps Summary

**1 BLOCKER — FUI-06 cross-link not wired**

The "Want tighter containment? See the Sharing Guide →" button in `FunnelRiskPanel` exists and the component contract is test-verified. However, the host component `HubPanel` renders `<SessionShareModal>` without passing `onOpenHelp`, so clicking the button only dismisses the modal. The Help tab is never opened.

Fix requires: (a) adding `onOpenHelp?: () => void` to HubPanel's Props, (b) threading it from App.tsx through HubPanel to SessionShareModal, and optionally (c) navigating to the `help-sharing` section specifically within HelpTab.

**1 WARNING — Sharing guide expiry duration inaccuracy**

`sharing-guide.md` line 11 lists "1 hour, 4 hours, 24 hours" but the actual FunnelRiskPanel options are "30 minutes, 1 hour, 4 hours, 8 hours, Until I disable." The guide omits the 30-minute preset and states an incorrect maximum (24h vs 8h). This is a documentation-only issue — no functional code is affected.

---

_Verified: 2026-06-30T21:55:00Z_
_Verifier: Claude (gsd-verifier)_
