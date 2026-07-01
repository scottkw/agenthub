---
phase: 168-bug-fix-settings-polish
verified: 2026-07-01T23:09:58Z
status: human_needed
score: 2/6 must-haves fully verified (4 present + wired, behavior partially or fully unverified by automation)
behavior_unverified: 4
overrides_applied: 0
behavior_unverified_items:
  - truth: "SC1 (FIX-01): A web-share guest sees live plugin-config changes without page reload, with no CSP errors visible in DevTools Console."
    test: "Open the /app/ share URL in a real (non-Wails) browser; change a plugin-config value server-side; confirm the terminal updates live without reload; inspect DevTools Console for CSP errors."
    expected: "Config applies live (no reload) and the Console shows no CSP violations."
    why_human: "The SSE hot-swap state-transition mechanism itself IS behaviorally verified (WebShareSessionView.plugin-config.test.tsx mounts the real component and dispatches a fake SSE event, asserting the applied config changes) — but jsdom cannot evaluate real browser CSP enforcement or a live DevTools Console. Already tracked as TESTING.md M-43 (added by 168-07)."
  - truth: "SC2 (FIX-02): Multiple simultaneous viewers can connect to one shared session; a stuck viewer can be disconnected via the Share modal Disconnect button, and the Hub viewer count updates within the next poll cycle."
    test: "Two real browser clients open the same share link; confirm neither is kicked; click 'Disconnect all viewers' and confirm both drop and the Hub card's viewer count returns to 0 within a poll cycle."
    expected: "Both viewers coexist; Disconnect drops both; visible Hub viewer count reaches 0."
    why_human: "The backend mechanism (DisconnectWebViewers force-closing only Origin==\"web\" subscribers; two web subscribers coexisting without eviction) IS behaviorally verified by real Go tests (TestDisconnectWebViewers, TestHub_TwoWebOriginSubscribers_NoEviction) that construct live subscribers, broadcast frames, and assert receipt/closure under -race. The end-to-end two-real-browser UI observation (visible viewer-count drop within a poll cycle) cannot be exercised in jsdom/Go unit tests. Already tracked as TESTING.md M-42 (added by 168-07)."
  - truth: "SC3 (FIX-03): Opening a remote tailnet session from the Hub opens an in-app terminal tab (not an external browser window) that streams the terminal relay correctly."
    test: "On a live two-Mac tailnet, open a remote session from the Hub; confirm an in-app tab opens (no external browser window) and the terminal streams real PTY output; open a second different remote session and confirm two independent tabs with no cross-contamination."
    expected: "In-app tab opens per session; terminal streams live; two sessions never share cap/host state."
    why_human: "Code wiring is confirmed by direct source reading (handleOpenRemoteSession / handleModalExchange call openWebSessionTab, not BrowserOpenURL) and by App.open-remote.test.tsx (a real-mount click-path test on SessionCard's 'Open in browser' item plus source-inspection of the App.tsx routing). Real cross-machine PTY streaming correctness requires a live two-Mac tailnet, which no automated test in this repo can exercise (documented convention — the :34115 wails-dev bridge has no real tailnet peer). Already tracked as TESTING.md M-13 (reworded by 168-07 for the new in-app-tab behavior)."
  - truth: "SC4 (UX-02, partial — CR-02 sub-behavior): The footer 'Share Session' button opens the Share modal for the currently-active session with no independent state drift, INCLUDING when opened from a non-Hub session tab (Funnel warm-up resolution and 'Disconnect all viewers' visibility/count must stay live)."
    test: "Create a new session (default stayOnHubAfterCreate=OFF auto-switches to it); from that session's tab (not the Hub tab), click the footer 'Share Session' button; enable Funnel and confirm the warm-up UI resolves to 'live' (not stuck until the 30s timeout) once the daemon actually enables it; have a second viewer join and confirm the viewer count / 'Disconnect all viewers' button visibility update live while the modal stays open from the session tab."
    expected: "Funnel warm-up resolves promptly via live state, not the 30s fallback; viewer count and Disconnect-button visibility update without closing/reopening the modal."
    why_human: "This is the exact scenario code review finding CR-02 identified and fixed (168-REVIEW.md, fixed in commit c6255942): a second useEffect polls ListSessions() every 3s while shareModalSession is set and the Hub tab is not already polling. The fix is present and wired (frontend/src/App.tsx:1049-1070) and structurally sound on inspection (correct effect deps, cleanup via clearInterval), but per the executor's own TESTING.md note (line 34) NO dedicated test exercises this specific poll/live-sync behavior — CR-02's live-poll timing is only indirectly touched via SessionShareModal.test.tsx's Funnel warm-up suite through prop threading, not a focused App-level interval test. This is a state-consistency invariant (poll starts/stops correctly, feeds live server truth into an already-open modal) that presence/wiring checks cannot confirm, and it is not yet covered by an existing TESTING.md manual item — a NEW manual item is recommended (see Gaps Summary)."
human_verification:
  - test: "SC1 (FIX-01) live browser check — TESTING.md M-43"
    expected: "Live hot-swap with no CSP console errors on the /app/ surface."
    why_human: "Real browser CSP/DevTools inspection; not automatable."
  - test: "SC2 (FIX-02) live two-browser-viewer smoke test — TESTING.md M-42"
    expected: "Two viewers coexist; Disconnect drops both; Hub viewer count reaches 0 within a poll cycle."
    why_human: "Requires two real browser clients on a live share link."
  - test: "SC3 (FIX-03) live two-Mac remote-open in-app tab check — TESTING.md M-13 (reworded)"
    expected: "In-app tab opens (no external browser); terminal streams correctly; two sessions stay isolated."
    why_human: "Requires two real Macs on the same tailnet; no automated tailnet peer exists in CI."
  - test: "SC4 (UX-02) CR-02 live-sync check from a non-Hub session tab — NOT YET in TESTING.md manual checklist"
    expected: "Funnel warm-up resolves live (not via 30s timeout); viewer count / Disconnect button stay live while the Share modal is open from a session tab, not the Hub tab."
    why_human: "CR-02's fix (a scoped 3s poll effect) has zero automated coverage per the executor's own TESTING.md disclosure; this is a state-consistency invariant that needs a live run to confirm before treating the Critical finding as durably closed."
---

# Phase 168: Bug Fix & Settings Polish Verification Report

**Phase Goal:** Five web-share/Hub bugs are repaired and two Settings/Footer UX friction points eliminated, clearing Issues #112, #115, #116, #117, #118, and #121.
**Verified:** 2026-07-01T23:09:58Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A web-share guest sees live plugin-config changes without reload, no CSP errors in DevTools | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Core SSE hot-swap mechanism VERIFIED by a real-mount behavioral test (`WebShareSessionView.plugin-config.test.tsx` — dispatches a fake EventSource event and asserts the applied config updates without remount); live-browser CSP-console check needs human (TESTING.md M-43, already documented). |
| 2 | Multiple simultaneous viewers coexist; owner can disconnect all viewers; Hub count updates within a poll cycle | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Backend mechanism VERIFIED by real Go tests under `-race` (`TestDisconnectWebViewers`, `TestHub_TwoWebOriginSubscribers_NoEviction` — construct real subscribers, broadcast frames, assert receipt/closure); live two-browser UI confirmation needs human (TESTING.md M-42, already documented). |
| 3 | Opening a remote tailnet session opens an in-app tab (not external browser) and streams correctly | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Code wiring confirmed by direct source read (`handleOpenRemoteSession`/`handleModalExchange` call `openWebSessionTab`, never `BrowserOpenURL`) + `App.open-remote.test.tsx` (real-mount SessionCard click-path test + source-inspection); live cross-machine PTY streaming needs human (TESTING.md M-13, reworded by 168-07). |
| 4 | Footer "Share Session" button opens the Share modal for the active session with no independent state drift | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CR-01 fix (footer no-op for just-created session) VERIFIED — `hubSessions` seeded in `createTab`'s success path (App.tsx:822-839), covered by `App.createTab.hubSessionsSeed.test.tsx`. CR-02 fix (live-sync poll while modal open from a non-Hub tab, closing the Funnel-warm-up-hang/stale-viewer-count bug) is present and wired (App.tsx:1049-1070) but has **no dedicated test** — explicitly disclosed as untested in TESTING.md line 34 by the executor. This is a state-consistency invariant that needs a live check (no existing manual item covers it — recommend adding one). |
| 5 | "Stay on Hub after creating session" toggle in Settings → Session Behavior prevents auto-switch when ON | ✓ VERIFIED | `internal/daemon/engine_stayonhub_test.go`/`api_stayonhub_test.go` (default false, persists, round-trips) all PASS. Frontend gating logic directly read at App.tsx:806-812 — a simple deterministic `if (!stayOnHubAfterCreateRef.current) setActiveId(sessionId)` guard, correctly ordered after unconditional `setTabs` (D-10); confirmed by direct code reading plus `App.createTab.stayOnHub.test.tsx` source-inspection assertions on exact ordering. |
| 6 | A never-shared local session's Hub card reads 0 viewers (web-origin-only count) | ✓ VERIFIED | `internal/relay/hub_test.go` `TestRemoteViewerCount` and `internal/daemon/engine_test.go` `TestListSessions_ViewerCount` (plus `api_test.go` `TestAPI_ListSessionsViewerCount`) all PASS — real behavioral tests constructing mixed local/web subscribers and asserting the count excludes local-origin ones. `hub.RemoteViewerCount()` verified as the sole `engine.go` ListSessions call site (line 548); `SubscriberCount()`/`NotifyViewerCount` confirmed UNCHANGED (still used by `relay/server.go:506`), per the plan's prohibition. |

**Score:** 2/6 truths fully automated-verified; 4/6 present + wired with strong partial automated coverage, full observable truth requires a human/live check (3 of the 4 were pre-planned as manual-only in TESTING.md; 1 — the CR-02 sub-behavior of truth #4 — is a newly surfaced gap in manual coverage).

### Code Review Fix Verification (not just SUMMARY claims — checked against actual code + re-ran tests)

All 6 findings from 168-REVIEW.md (3 Critical, 3 Warning) were independently re-verified in the current codebase, not just trusted from 168-REVIEW-FIX.md's narrative:

| Finding | Fix commit | Verified in code | Verified by test |
|---|---|---|---|
| CR-01 footer Share button no-ops for just-created session | `e5391299` | `frontend/src/App.tsx:822-839` — `setHubSessions` appends a `SessionInfo` entry inside `createTab`'s success path, before `catch` | `App.createTab.hubSessionsSeed.test.tsx` PASS |
| CR-02 no live updates when modal opened from non-Hub tab | `c6255942` | `frontend/src/App.tsx:1049-1070` — scoped poll effect (`shareModalSession` set AND `activeId !== HUB_TAB.id`) | **No dedicated test** (disclosed in TESTING.md; see truth #4 above) |
| CR-03 missing `key` leaks state between remote tabs | `554f09b9` | `frontend/src/App.tsx:1714-1715` — `<WebShareSessionView key={wsSessionId} ...>` | `App.open-remote.test.tsx` (extended) PASS |
| WR-01 modal shows "sharing ON" on backend failure | `5820fa6d` | `handleShellWebShareConfirm` returns `Promise<boolean>`; modal only sets `shareEnabled(true)` on `ok` | `SessionShareModal.test.tsx` (extended, success + new failure-path test) PASS |
| WR-02 `engine.go` gofmt violation | `794ca22c` | `gofmt -l internal/daemon/engine.go` → empty (verified independently) | N/A (formatting) |
| WR-03 `types.go` gofmt violation | `1e070a94` | `gofmt -l internal/daemon/types.go` → empty (verified independently) | N/A (formatting) |

All 7 review-fix commits (`e5391299`, `c6255942`, `554f09b9`, `5820fa6d`, `794ca22c`, `1e070a94`, `1204d189`) are present in `git log` on the current branch, and the diffs match the claimed changes.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/hub.go: (*Hub).RemoteViewerCount()` | Web-origin-only viewer count | ✓ VERIFIED | Present at line 232; `TestRemoteViewerCount` passes |
| `internal/relay/hub.go: (*Hub).DisconnectWebViewers()` | Force-close web-origin subscribers | ✓ VERIFIED | Present at line 261; `TestDisconnectWebViewers`/`TestHub_TwoWebOriginSubscribers_NoEviction` pass under `-race` |
| `internal/daemon/engine.go` viewerCount call site | Uses `RemoteViewerCount()`, not `SubscriberCount()` | ✓ VERIFIED | Line 548 confirmed; `SubscriberCount()` untouched, still used by `relay/server.go` |
| `internal/daemon/api.go` disconnect + stay-on-hub routes | Daemon-local mux only | ✓ VERIFIED | `POST /sessions/{id}/disconnect-viewers` (line 154), `GET/PATCH /settings/stay-on-hub-after-create` (lines 127-128) registered on `a.mux`, not the guest-reachable webserver `/api/...` surface |
| `internal/daemon/client.go` + `app.go` bound methods | `DisconnectViewers`, `Get/SetStayOnHubAfterCreate` | ✓ VERIFIED | Present, delegate correctly |
| `frontend/src/components/Hub/WebShareSessionView.tsx: baseURL` prop + self-fetch/SSE | Web-guest live plugin-config | ✓ VERIFIED | Present at lines 29, 68-117; behavioral test passes |
| `frontend/src/App.tsx: openWebSessionTab(sessionId, baseURL, capToken)` | Per-tab param carry | ✓ VERIFIED | Extended signature at line 1156; per-tab `Tab.baseURL` present at line 233 |
| `frontend/src/components/SettingsTab.tsx` stay-on-hub toggle | In `id="settings-session-behavior"`, default OFF | ✓ VERIFIED | Lines 589 (section) / 616-641 (toggle) — correctly placed, NOT under `id="settings-behavior"` |
| `frontend/src/components/StatusBar.tsx` "Share Session" button | Single label, `onShareSession` prop | ✓ VERIFIED | Lines 7, 50, 53 |
| `frontend/src/components/Hub/SessionShareModal.tsx` "Disconnect all viewers" button | Reversible/ghost style, gated on `viewerCount > 0` | ✓ VERIFIED | Lines 276-282, 537-549; uses `.hub-share-internet-section__disable`, NOT destructive class |
| `TESTING.md` Suite Manifest / Traceability / manual checklist updates | All new test files registered; M-13 reworded; M-42/M-43 added | ✓ VERIFIED | Confirmed via grep; `bash tests/check-traceability-paths.sh` exits 0 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `engine.go` ListSessions | `hub.RemoteViewerCount()` | direct call | ✓ WIRED | Line 548 |
| App.tsx `handleOpenRemoteSession`/`handleModalExchange` | `openWebSessionTab` | in-app tab, no `BrowserOpenURL` | ✓ WIRED | Lines 1286, 1388; `BrowserOpenURL` only remains at `onOpenURL={BrowserOpenURL}` (help-link handler, unrelated to remote-session open) |
| App.tsx `createTab` | `setActiveId` | gated on `stayOnHubAfterCreateRef.current` | ✓ WIRED | Lines 806-812; tab always created (line 805) before the gate |
| StatusBar "Share Session" button | `openShareModalForActiveSession` | `onShareSession` prop | ✓ WIRED | App.tsx:1926, StatusBar.tsx:50 |
| SessionShareModal "Disconnect all viewers" | `DisconnectViewers(session.id)` RPC | `handleDisconnectViewers` | ✓ WIRED | SessionShareModal.tsx:280-282 → app.go:1034 → client.go:389-390 → api.go:1316-1324 |
| `<WebShareSessionView>` render site | remounts per remote session | `key={wsSessionId}` | ✓ WIRED | App.tsx:1714-1715 (CR-03 fix) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go build | `go build ./...` | clean, exit 0 | ✓ PASS |
| Go gofmt | `gofmt -l internal/daemon/engine.go internal/daemon/types.go internal/relay/hub.go internal/daemon/api.go internal/daemon/client.go app.go` | empty output | ✓ PASS |
| Go relay/daemon full package tests (`-race -short -count=1`) | `go test -race -short -count=1 ./internal/relay/... ./internal/daemon/...` | `ok` both packages | ✓ PASS |
| FIX-04 targeted Go tests | `go test ./internal/relay/... ./internal/daemon/... -run 'TestRemoteViewerCount\|TestListSessions_ViewerCount'` | PASS | ✓ PASS |
| FIX-02 targeted Go tests (`-race`) | `go test ./internal/relay/... -run 'TestDisconnectWebViewers\|TestHub_TwoWebOriginSubscribers_NoEviction' -race` | PASS | ✓ PASS |
| UX-01 targeted Go tests | `go test ./internal/daemon/... -run StayOnHub` | PASS (9 subtests) | ✓ PASS |
| FIX-02 daemon RPC tests | `go test ./internal/daemon/... -run DisconnectViewers` | PASS | ✓ PASS |
| Targeted vitest (8 files, phase + review-fix tests) | `pnpm vitest run WebShareSessionView.plugin-config App.open-remote App.createTab.stayOnHub SettingsTab.stay-on-hub-toggle StatusBar.shareSession SessionShareModal.disconnect App.createTab.hubSessionsSeed SessionShareModal` | 8 files / 104 tests PASS | ✓ PASS |
| Full frontend vitest suite | `pnpm vitest run` | 141 files / 2320 tests PASS | ✓ PASS |
| Frontend typecheck | `pnpm exec tsc --noEmit` | clean, exit 0 | ✓ PASS |
| Traceability checker | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist`, exit 0 | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| FIX-01 | 168-02 | Web-guest plugin-config self-fetch + SSE hot-swap | ✓ SATISFIED (mechanism); live CSP check human-pending | Code + behavioral test present; M-43 manual item |
| FIX-02 | 168-06 | Multi-viewer + owner disconnect | ✓ SATISFIED (mechanism); live 2-viewer check human-pending | Code + Go behavioral tests present; M-42 manual item |
| FIX-03 | 168-03 | Remote session opens in-app tab | ✓ SATISFIED (mechanism); live cross-machine check human-pending | Code + tests present; M-13 manual item (reworded) |
| FIX-04 | 168-01 | Hub viewer count excludes local subscribers | ✓ SATISFIED | Fully automated-verified |
| UX-01 | 168-04 | Stay-on-Hub-after-create toggle | ✓ SATISFIED | Fully automated-verified |
| UX-02 | 168-05 (+ 168-REVIEW-FIX CR-01/CR-02) | Footer Share Session button, no state drift | ⚠️ PARTIALLY SATISFIED | CR-01 sub-fix automated-verified; CR-02 sub-fix present/wired but untested — new human item recommended |

No orphaned requirements — REQUIREMENTS.md maps exactly FIX-01, FIX-02, FIX-03, FIX-04, UX-01, UX-02 to Phase 168 (all marked "Complete"), matching the union of every plan's `requirements:` frontmatter field exactly.

### Anti-Patterns Found

None. Scanned all files touched by plans 01-07 and the review-fix commits (`internal/relay/hub.go`, `internal/relay/hub_test.go`, `internal/daemon/engine.go`, `internal/daemon/types.go`, `internal/daemon/api.go`, `internal/daemon/client.go`, `app.go`, `frontend/src/components/Hub/WebShareSessionView.tsx`, `frontend/src/App.tsx`, `frontend/src/components/SettingsTab.tsx`, `frontend/src/components/StatusBar.tsx`, `frontend/src/components/Hub/SessionShareModal.tsx`, `frontend/src/components/Hub/HubPanel.tsx`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented" markers — none found (one benign pre-existing user-facing error-message string containing the words "Funnel not available" is not a debt marker).

## Human Verification Required

### 1. FIX-01 live browser plugin-config hot-swap + CSP check (TESTING.md M-43)

**Test:** Open the `/app/` share URL in a real (non-Wails) browser; change plugin-config server-side; confirm live hot-swap without page reload; inspect DevTools Console for CSP errors.
**Expected:** Config applies live; no CSP violations in the Console.
**Why human:** Real browser CSP enforcement + DevTools Console inspection cannot be exercised in jsdom.

### 2. FIX-02 live two-browser-viewer smoke test (TESTING.md M-42)

**Test:** Two real browser clients open the same share link; confirm neither is kicked; click "Disconnect all viewers"; confirm both drop and the Hub card's viewer count returns to 0 within a poll cycle.
**Expected:** Both viewers coexist; Disconnect drops both; visible Hub count reaches 0.
**Why human:** Requires two real browser clients and observation of the live Hub poll cycle.

### 3. FIX-03 live two-Mac remote-open in-app tab check (TESTING.md M-13, reworded)

**Test:** On a live two-Mac tailnet, open a remote session from the Hub; confirm an in-app tab opens (no external browser window) and streams real PTY output; open a second different remote session; confirm two independent, non-cross-contaminated tabs.
**Expected:** In-app tab per session; live streaming; correct per-tab isolation.
**Why human:** Requires two real Macs on the same tailnet — no automated tailnet peer exists in this repo's test environment.

### 4. UX-02 / CR-02 live-sync check for the footer Share button opened from a non-Hub tab (NOT YET in TESTING.md — recommended new manual item)

**Test:** Create a new session (default `stayOnHubAfterCreate=OFF` auto-switches to it, so you land on the session's own tab, not Hub). From that session tab, click the footer "Share Session" button. Enable Funnel and confirm the warm-up UI resolves to "live" via actual server state (not the 30s timeout fallback) once the daemon enables it. Have a second browser join as a viewer and confirm the modal's viewer count and "Disconnect all viewers" button visibility update live while the modal remains open from the session tab (not the Hub tab).
**Expected:** Funnel warm-up resolves promptly from live poll data; viewer count / Disconnect-button visibility track server truth without needing to close and reopen the modal or visit the Hub tab.
**Why human:** This is the exact scenario Critical finding CR-02 fixed (a scoped `ListSessions()` poll effect gated on `shareModalSession` being set and the Hub tab not already polling — `frontend/src/App.tsx:1049-1070`). The fix is present, wired, and structurally sound on code inspection, but the executor's own TESTING.md disclosure (line 34) states no dedicated automated test exercises this poll/live-sync behavior — it is only indirectly touched via `SessionShareModal.test.tsx`'s Funnel warm-up suite through prop threading, not a focused interval/timing test. Given this closes a Critical (not Warning) finding about a real user-visible hang (Funnel warm-up appearing stuck for 30s despite Funnel actually being live), a live confirmation is warranted before treating it as durably fixed, and a permanent regression-guard manual item should be added to TESTING.md's Category J/Q area alongside M-42/M-43.

## Gaps Summary

No BLOCKER-level gaps. All 6 requirements (FIX-01..04, UX-01, UX-02) have working, wired implementations; all 7 code-review-fix commits landed and were independently re-verified in the current source (not just trusted from 168-REVIEW-FIX.md); the full Go test suite (`-race -short`) and full frontend vitest suite (2320 tests) pass; `tsc --noEmit`, `gofmt -l`, `go vet` (via build), and `check-traceability-paths.sh` are all clean.

The phase is **not** blocked, but 4 of the 6 roadmap success criteria have an observable-truth component that automated tests cannot fully close:

- 3 of these (SC1/FIX-01, SC2/FIX-02, SC3/FIX-03) were correctly anticipated by the phase's own planning and are already tracked as manual-only items in TESTING.md (M-43, M-42, M-13) — this is expected, not a defect, for live-browser/live-network/live-cross-machine behavior.
- 1 (SC4/UX-02, specifically the CR-02 code-review-fix sub-behavior) is a genuine coverage gap the executor itself flagged in TESTING.md but did not turn into a manual checklist item. Recommend either (a) adding a TDD fake-timer regression test for the `App.tsx:1049-1070` poll effect in a follow-up phase, or (b) adding a new TESTING.md manual item (Category J or a new Category) covering the live-sync-from-non-Hub-tab scenario, and running it once before this phase is considered fully closed out.

---

_Verified: 2026-07-01T23:09:58Z_
_Verifier: Claude (gsd-verifier)_
