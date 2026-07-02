---
phase: 168-bug-fix-settings-polish
verified: 2026-07-02T13:00:07Z
status: human_needed
score: 5/6 must-haves fully verified (1 present + wired, live end-to-end confirmation still pending)
behavior_unverified: 1
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: "2/6 fully automated-verified, 4/6 present+wired needing human confirmation"
  gaps_closed:
    - "SC1 (FIX-01): live browser plugin-config hot-swap + CSP check — human UAT PASS (168-UAT.md Test 1, 2026-07-02)"
    - "SC2 (FIX-02): live two-browser-viewer smoke test — human UAT PASS (168-UAT.md Test 2, 2026-07-02)"
    - "SC4 (UX-02) footer pill web-share drift (the exact bug live UAT Test 4 caught) — root-caused and fixed by gap-closure plan 168-08: SessionShareModal.onShareEnabledChange callback + App.tsx setWebEnabled wiring, with real mounted-component regression tests (ON/OFF notify + un-warned no-notify) and an App-wiring source-inspection test, all passing"
  gaps_remaining:
    - "SC3 (FIX-03): live two-Mac remote-open in-app tab check — UAT reported 'blocked, no second Mac right now' (168-UAT.md Test 3); still unverified live, unchanged from prior pass"
    - "SC4 (UX-02) D4: the 168-08 fix itself has full unit/regression coverage of every individual link in the causal chain (modal toggle -> callback -> App wiring -> StatusBar render) but no single test mounts App.tsx end-to-end to observe the footer pill flip live on a real daemon session — the exact integration point that already produced one live-UAT surprise. A fresh live re-check (rerun 168-UAT.md Test 4) is recommended before treating this as durably closed. This is *new evidence needed*, not a regression: the underlying callback-fires and wiring-exists facts are both proven by real tests."
  regressions: []
behavior_unverified_items:
  - truth: "SC4 (UX-02): Footer StatusBar pill shows 'WEB ON'/'WEB OFF' matching actual server web-share state on every modal toggle path (footer-opened and Hub-card-opened, warned and un-warned) — no drift, per the gap fixed by plan 168-08."
    test: "On a live daemon session with a shell CLI: dismiss the one-time shell-web-share warning once, then from the Share modal toggle 'Share the session' ON — confirm the footer pill flips to 'WEB ON' immediately (not stuck on 'WEB OFF'); toggle OFF — confirm it returns to 'WEB OFF'. Repeat opening the modal from a Hub card (not just the footer) to cover both open paths."
    expected: "Footer pill always matches the modal's live share state; no stale label survives a toggle."
    why_human: "SessionShareModal.test.tsx behaviorally proves the modal-level callback fires with the correct value on real toggle clicks, and StatusBar.shareSession.test.tsx proves (a) via source-inspection that App.tsx wires that callback verbatim to setWebEnabled, and (b) via a separate mounted-StatusBar test that the pill text correctly reflects the webEnabled prop. No single test mounts App.tsx itself (an established, pre-existing test-suite limitation, not introduced by this fix) to observe the full click-to-pill-flip chain in one render tree on a live daemon. Given this exact seam already produced a live-UAT-only-detectable bug once (168-UAT.md Test 4), a fresh live re-check is the appropriate closing evidence rather than trusting the fix's plausibility."
human_verification:
  - test: "SC3 (FIX-03) live two-Mac remote-open in-app tab check (TESTING.md M-13)"
    expected: "In-app tab opens (no external browser window); terminal streams real PTY output; two independently-opened remote sessions never cross-contaminate."
    why_human: "Requires two real Macs on the same tailnet. 168-UAT.md Test 3 explicitly reports 'blocked, no second Mac right now' — this is carried forward unresolved from the prior verification pass, not a new finding."
  - test: "SC4 (UX-02) live re-check of the 168-08 footer-pill-drift fix (recommend adding as a new TESTING.md manual item, e.g. M-44, alongside M-42/M-43)"
    expected: "Footer pill tracks live web-share state through a real toggle on a live daemon session, from both the footer-opened and Hub-card-opened modal paths, with the shell warning already dismissed."
    why_human: "This is the exact scenario 168-UAT.md Test 4 found broken. The fix (168-08) is proven correct at every individual seam by real mounted-component tests, but the full chain has not been re-confirmed live end-to-end since the fix landed. Note: 168-08-PLAN.md's own <verification> section defers this exact check to a live-UAT re-run and does not add a permanent TESTING.md manual checklist entry for it — recommend adding one so this regression class has standing coverage."
---

# Phase 168: Bug Fix & Settings Polish Verification Report

**Phase Goal:** Five web-share/Hub bugs are repaired and two Settings/Footer UX friction points eliminated, clearing Issues #112, #115, #116, #117, #118, and #121.
**Verified:** 2026-07-02T13:00:07Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plan 168-08 (UX-02 / #115 footer pill drift), following live UAT that closed SC1/SC2 and surfaced the SC4 gap.

## Goal Achievement

### Observable Truths (ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A web-share guest sees live plugin-config changes without reload, no CSP errors in DevTools (FIX-01) | ✓ VERIFIED | Mechanism verified by `WebShareSessionView.plugin-config.test.tsx` (real-mount SSE event dispatch); live confirmation now closed by human UAT — 168-UAT.md Test 1: PASS ("Find in scrollback" plugin toggle disabled live in the guest browser, no reload, no CSP violations in DevTools Console). |
| 2 | Multiple simultaneous viewers coexist; owner can disconnect all viewers; Hub count updates within a poll cycle (FIX-02) | ✓ VERIFIED | Backend mechanism verified by real Go tests under `-race` (`TestDisconnectWebViewers`, `TestHub_TwoWebOriginSubscribers_NoEviction`); live confirmation now closed by human UAT — 168-UAT.md Test 2: PASS. |
| 3 | Opening a remote tailnet session opens an in-app tab (not external browser) and streams correctly (FIX-03) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Code wiring confirmed by direct source read (`handleOpenRemoteSession`/`handleModalExchange` call `openWebSessionTab`, never `BrowserOpenURL`) + `App.open-remote.test.tsx`. Live two-Mac confirmation attempted in 168-UAT.md Test 3 but **blocked** ("no second Mac right now") — unresolved, unchanged from the prior verification pass. |
| 4 | Footer "Share Session" button opens the Share modal for the active session with no independent state drift (UX-02) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CR-01 (footer no-op for just-created session) and CR-03 (cross-tab state leak) sub-fixes remain ✓ VERIFIED (unchanged, regression-tested, no files touched by 168-08). The footer-pill-drift sub-bug that live UAT (Test 4) caught — pill stuck on "WEB OFF" while actively shared, once the shell warning was dismissed — is now fixed by gap-closure plan **168-08**: `SessionShareModal.onShareEnabledChange` callback (fires after a successful `ToggleWebServing`, both ON and OFF) wired in `App.tsx` to `setWebEnabled`. Verified fresh in this pass: real code present and correct (see Fresh Code Verification below), 4 new/extended behavioral+source-inspection tests pass, full 141-file/2324-test vitest suite green, `tsc --noEmit` clean, traceability script passes. No test mounts App.tsx end-to-end to observe the live pill flip — routed to human re-check (see Human Verification). |
| 5 | "Stay on Hub after creating session" toggle prevents auto-switch when ON (UX-01) | ✓ VERIFIED (regression-checked, unchanged) | Not touched by 168-08. `internal/daemon/engine_stayonhub_test.go`/`api_stayonhub_test.go` and `App.createTab.stayOnHub.test.tsx` unaffected; full suite still green. |
| 6 | A never-shared local session's Hub card reads 0 viewers (FIX-04) | ✓ VERIFIED (regression-checked, unchanged) | Not touched by 168-08. `TestRemoteViewerCount`, `TestListSessions_ViewerCount`, `TestAPI_ListSessionsViewerCount` unaffected. |

**Score:** 5/6 truths at ✓ VERIFIED status (including 2 newly closed by human UAT this pass); 1/6 (truth #4, the 168-08 sub-behavior only) present + wired with strong per-seam automated coverage but no full live end-to-end confirmation yet. Truth #3 (FIX-03) remains blocked on hardware availability, unchanged from the prior pass — not a new gap.

### Fresh Code Verification — 168-08 Gap Closure (Footer Pill Web-Share Drift, UX-02 / #115)

Focus of this verification pass, checked directly against the current codebase (not trusted from 168-08-SUMMARY.md):

| Item | File:Line | Verified |
|---|---|---|
| `onShareEnabledChange?: (sessionId: string, enabled: boolean) => void` prop declared | `frontend/src/components/Hub/SessionShareModal.tsx:70` | ✓ Present, optional, correctly typed |
| Prop destructured in component signature | `SessionShareModal.tsx:104` | ✓ Present |
| Invoked inside `handleShareToggle`'s success path, after `await ToggleWebServing` resolves, for both ON and OFF | `SessionShareModal.tsx:243-254` (`onShareEnabledChange?.(session.id, next)` at line 254, after `await ToggleWebServing(session.id, next)` at line 244 and `setShareEnabled(next)`/cache-clear) | ✓ Confirmed — not invoked inside the warning-guard early return (lines 239-242), matching the plan's no-double-set requirement with `handleShellWebShareConfirm` |
| App.tsx single `<SessionShareModal>` render wires the callback | `frontend/src/App.tsx:1945-1958` — `onShareEnabledChange={(sessionId, enabled) => setWebEnabled((prev) => ({ ...prev, [sessionId]: enabled }))}` | ✓ Confirmed — single render site (grep confirms exactly one `<SessionShareModal` in App.tsx), correct setter shape |
| Regression test: warned-path ON notifies | `frontend/src/components/__tests__/SessionShareModal.test.tsx:710-730` | ✓ PASS (real mount, real click, asserts `ToggleWebServing('sess-1', true)` and `onShareEnabledChange('sess-1', true)` called exactly once, no warning banner) |
| Regression test: warned-path OFF notifies | `SessionShareModal.test.tsx:732-749` | ✓ PASS (real mount from already-shared state, real click, asserts OFF args on both mocks) |
| Regression test: un-warned first-time shell path does NOT notify | `SessionShareModal.test.tsx:751-770` | ✓ PASS (asserts banner shown, `ToggleWebServing` NOT called, `onShareEnabledChange` NOT called — proves no double-set with `handleShellWebShareConfirm`) |
| App-wiring source-inspection test | `frontend/src/components/__tests__/StatusBar.shareSession.test.tsx:178-184` | ✓ PASS (greps the `<SessionShareModal` render block in `App.tsx?raw` for co-located `onShareEnabledChange` and `setWebEnabled`) |
| TESTING.md Suite Manifest note + 2 traceability rows | `TESTING.md:34, 215, 216` | ✓ Present, matches standing convention |
| `check-traceability-paths.sh` | — | ✓ `OK: all traceability paths exist` |
| Commits present in git log, diffs match claims | `2129a423` (fix), `a1b609ff` (test), `6ede69a2` (docs) | ✓ Confirmed via `git show --stat` — diffs are exactly the claimed +17/+94/-0 lines, nothing extraneous |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/hub.go: (*Hub).RemoteViewerCount()` | Web-origin-only viewer count | ✓ VERIFIED (unchanged) | Not touched by 168-08 |
| `internal/relay/hub.go: (*Hub).DisconnectWebViewers()` | Force-close web-origin subscribers | ✓ VERIFIED (unchanged) | Not touched by 168-08 |
| `internal/daemon/engine.go` viewerCount call site | Uses `RemoteViewerCount()` | ✓ VERIFIED (unchanged) | Not touched by 168-08 |
| `frontend/src/components/Hub/WebShareSessionView.tsx` | Web-guest live plugin-config | ✓ VERIFIED (unchanged) | Not touched by 168-08; live-confirmed by 168-UAT Test 1 |
| `frontend/src/components/SettingsTab.tsx` stay-on-hub toggle | Default OFF | ✓ VERIFIED (unchanged) | Not touched by 168-08 |
| `frontend/src/components/StatusBar.tsx` "Share Session" button | Single label | ✓ VERIFIED (unchanged) | Not touched by 168-08 |
| `frontend/src/components/Hub/SessionShareModal.tsx` `onShareEnabledChange` prop | Fires on ON/OFF after successful toggle | ✓ VERIFIED (new, 168-08) | Line 70 (decl), 104 (destructure), 254 (invocation) |
| `frontend/src/App.tsx` `<SessionShareModal>` wiring | `onShareEnabledChange` → `setWebEnabled` | ✓ VERIFIED (new, 168-08) | Lines 1945-1958 |
| `TESTING.md` Suite Manifest / Traceability | 168-08 note + 2 rows | ✓ VERIFIED (new, 168-08) | Lines 34, 215, 216 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `SessionShareModal.handleShareToggle` success path | `onShareEnabledChange(session.id, next)` | direct call after `await ToggleWebServing` resolves | ✓ WIRED | `SessionShareModal.tsx:254`; behaviorally proven for both ON and OFF by real click tests |
| App.tsx `<SessionShareModal>` render | `setWebEnabled` | `onShareEnabledChange` prop callback | ✓ WIRED | `App.tsx:1955-1957`; confirmed by source-inspection test (App.tsx not fully mountable — established, pre-existing test-suite convention) |
| `webEnabled[sessionId]` | StatusBar pill label | `!!webEnabled[tab.sessionId]` prop → "WEB ON"/"WEB OFF" | ✓ WIRED | Confirmed by pre-existing `StatusBar.shareSession.test.tsx` tests (lines 65-83) that mount StatusBar directly with each `webEnabled` value and assert the rendered label |
| (full chain) modal click → footer pill on a live App.tsx instance | — | — | ⚠️ NOT DIRECTLY TESTED | No test mounts App.tsx end-to-end; each individual link above is independently proven. Routed to human verification. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Targeted vitest (168-08 files) | `pnpm vitest run SessionShareModal StatusBar.shareSession --run` | 3 files / 57 tests PASS | ✓ PASS |
| Full frontend vitest suite | `pnpm vitest run` | 141 files / 2324 tests PASS (up from 2320 — +4 new tests, 0 new files, matches plan's "net 0 new files" claim) | ✓ PASS |
| Frontend typecheck | `pnpm exec tsc --noEmit` | clean, exit 0 | ✓ PASS |
| Traceability checker | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist`, exit 0 | ✓ PASS |
| Backend/relay regression (no files touched by 168-08) | `git diff --stat <prior-commit> HEAD -- internal/ app.go` | empty diff | ✓ CONFIRMED unaffected |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| FIX-01 | 168-02 (+168-07) | Web-guest plugin-config self-fetch + SSE hot-swap | ✓ SATISFIED | Code + behavioral test + human UAT PASS (Test 1) |
| FIX-02 | 168-06 (+168-07) | Multi-viewer + owner disconnect | ✓ SATISFIED | Code + Go behavioral tests + human UAT PASS (Test 2) |
| FIX-03 | 168-03 (+168-07) | Remote session opens in-app tab | ⚠️ SATISFIED (mechanism); live cross-machine check still blocked | Code + tests present; 168-UAT Test 3 blocked (no second Mac) |
| FIX-04 | 168-01 (+168-07) | Hub viewer count excludes local subscribers | ✓ SATISFIED | Fully automated-verified, unchanged |
| UX-01 | 168-04 (+168-07) | Stay-on-Hub-after-create toggle | ✓ SATISFIED | Fully automated-verified, unchanged |
| UX-02 | 168-05, 168-REVIEW-FIX (CR-01/CR-02), **168-08** | Footer Share Session button, no state drift | ⚠️ SATISFIED (mechanism, freshly closed); live end-to-end re-check recommended | CR-01/CR-03 automated-verified (unchanged); footer-pill-drift bug found by live UAT is now fixed by 168-08 with full per-seam test coverage; full-chain live re-check pending |

No orphaned requirements — REQUIREMENTS.md maps exactly FIX-01, FIX-02, FIX-03, FIX-04, UX-01, UX-02 to Phase 168 (all marked "Complete"), matching the union of every plan's `requirements:` frontmatter field exactly (168-01 through 168-08).

Note: the phase goal statement says "clearing Issues #112, #115, #116, #117, #118, and #121" — these are still OPEN on GitHub as of this verification (checked via `gh issue view`). No plan task in this phase closes GitHub issues via `gh issue close`; issue closure is a separate, manual, post-verification project-management action in this project's established workflow (consistent with prior phases/milestones), not a code-verifiable artifact. Not treated as a gap.

### Anti-Patterns Found

None. Scanned the 168-08-touched files (`frontend/src/components/Hub/SessionShareModal.tsx`, `frontend/src/App.tsx`, the two extended test files, `TESTING.md`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented" markers — none found. All prior-phase files (01-07 scope) re-confirmed clean by the unchanged-diff check above.

## Human Verification Required

### 1. FIX-03 live two-Mac remote-open in-app tab check (TESTING.md M-13) — carried forward, unresolved

**Test:** On a live two-Mac tailnet, open a remote session from the Hub; confirm an in-app tab opens (no external browser window) and streams real PTY output; open a second different remote session; confirm two independent, non-cross-contaminated tabs.
**Expected:** In-app tab per session; live streaming; correct per-tab isolation.
**Why human:** Requires two real Macs on the same tailnet. 168-UAT.md Test 3 reports "blocked, no second Mac right now" — genuinely untestable in this environment, not a defect finding.

### 2. UX-02 / 168-08 footer-pill-drift fix — live end-to-end re-check (recommend new TESTING.md manual item, e.g. M-44)

**Test:** On a live daemon session with a shell CLI, dismiss the one-time shell-web-share warning once. From the Share modal (opened either via the footer "Share Session" button or a Hub card), toggle "Share the session" ON — confirm the footer pill immediately reads "WEB ON" (not stuck on "WEB OFF"). Toggle OFF — confirm it returns to "WEB OFF". Repeat via the other open path (footer vs. Hub card) to cover both.
**Expected:** Footer pill always matches live server web-share state; no stale label after either toggle direction or either open path.
**Why human:** This is the exact scenario 168-UAT.md Test 4 found broken in the prior pass. The 168-08 fix is proven correct at every individual seam (modal→callback: real click test; callback→App wiring: source-inspection of the trivial one-line setter; App state→pill label: existing StatusBar render test) — but no test mounts App.tsx end-to-end, so the full click-to-pill-flip chain has not been observed live since the fix landed. Given this precise integration point already produced one live-UAT-only-detectable bug, closing evidence should be a fresh live confirmation rather than trusting per-seam proofs to compose correctly. Recommend making this a permanent TESTING.md manual checklist item (M-44) alongside M-42/M-43 so this regression class has standing coverage going forward — 168-08-PLAN.md's own `<verification>` section deferred this check but did not add a permanent entry.

## Gaps Summary

No BLOCKER-level gaps. All 6 requirements (FIX-01..04, UX-01, UX-02) have working, wired, tested implementations. The gap-closure plan 168-08 correctly root-caused and fixed the exact footer-pill-drift bug that live UAT surfaced (168-UAT.md Test 4), with real behavioral regression tests (not just source presence) proving the modal-level callback fires correctly on both ON and OFF transitions, and a source-inspection test proving the trivial App-level wiring exists. The fix is a small, well-scoped diff (17 lines of implementation) that does not touch the CR-02 poll, the seed effects, or `handleShellWebShareConfirm`, matching the plan exactly. `tsc --noEmit`, the full 141-file/2324-test vitest suite, and `check-traceability-paths.sh` are all clean; no regressions were introduced (Go/backend files are untouched by this plan).

Two items remain open for human/live confirmation, carried into this phase's closing UAT round:

1. **FIX-03 (SC3)** — blocked purely on hardware availability (no second Mac), unchanged from the prior verification pass. Not a code defect.
2. **UX-02 footer-pill-drift fix (168-08)** — code is present, correct, and covered by real component-level tests at every seam, but the full end-to-end chain (click on a live daemon session → visible pill flip) has not been re-observed live since the fix landed. Given this exact seam already fooled a prior "looks correct" code-review assessment once (the original CR-02 review marked the modal wiring "structurally sound" while the notify-callback didn't exist at all), a fresh live re-check is the appropriate bar before calling UX-02 durably closed. Recommend adding a permanent TESTING.md manual item (M-44) for this scenario.

Neither item blocks phase progression on its own technical merits (the underlying code changes are sound and tested), but both require a human/live pass before the phase can be marked fully `passed`.

---

_Verified: 2026-07-02T13:00:07Z_
_Verifier: Claude (gsd-verifier)_
