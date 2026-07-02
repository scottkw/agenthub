---
phase: 168-bug-fix-settings-polish
verified: 2026-07-02T15:53:47Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: "5/6 fully verified, 1 present+wired pending live confirmation"
  gaps_closed:
    - "SC3 (FIX-03): live two-Mac remote-open in-app tab — was blocked on hardware ('no second Mac'); now CONFIRMED live PASS on a two-Mac tailnet production build 2026-07-02 (168-UAT.md Test 3) after gap-closure plan 168-09 (RC-A daemon-proxy transport, RC-B .terminal-wrapper full-height, RC-C 'Open in tab' relabel — commits 2deff5f5, 3b3fea34)"
    - "SC4 (UX-02): footer-pill web-share drift — the 168-08 fix's full click-to-pill-flip chain was carried as behavior-unverified pending a live re-check; now CONFIRMED live PASS on a production build 2026-07-02 (168-UAT.md Test 4), both toggle directions, both open paths, warned path"
  gaps_remaining: []
  regressions: []
---

# Phase 168: Bug Fix & Settings Polish Verification Report

**Phase Goal:** Five web-share/Hub bugs are repaired and two Settings/Footer UX friction points eliminated, clearing Issues #112, #115, #116, #117, #118, and #121.
**Verified:** 2026-07-02T15:53:47Z
**Status:** passed
**Re-verification:** Yes — regenerated after gap-closure plan 168-09 (FIX-03 remote web-session tab) landed and after 168-UAT.md completed with all 4 live tests passing, closing the two previously-open human-verification items (SC3, SC4).

## Goal Achievement

### Observable Truths (ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A web-share guest sees live plugin-config changes without reload, no CSP errors in DevTools (FIX-01) | ✓ VERIFIED | Mechanism verified by `WebShareSessionView.plugin-config.test.tsx` (real-mount SSE dispatch); live-confirmed by human UAT — 168-UAT.md Test 1 PASS ("Find in scrollback" plugin toggle disabled live in the guest browser, no reload, no CSP violations in DevTools Console). |
| 2 | Multiple simultaneous viewers coexist; owner can disconnect all; Hub count updates within a poll cycle (FIX-02) | ✓ VERIFIED | Backend mechanism verified by real Go tests under `-race` (`TestDisconnectWebViewers`, `TestHub_TwoWebOriginSubscribers_NoEviction`); live-confirmed by human UAT — 168-UAT.md Test 2 PASS (two browser viewers coexist, "Disconnect all viewers" drops both, count returns to 0). |
| 3 | Opening a remote tailnet session opens an in-app tab (not external browser) and streams correctly (FIX-03) | ✓ VERIFIED | 168-09 root-cause fixes verified directly in the current codebase (RC-A daemon-proxy transport, RC-B full-height wrapper, RC-C relabel — see Fresh Code Verification below); 58 targeted tests pass; live-confirmed by human UAT — 168-UAT.md Test 3 PASS on a two-Mac tailnet production build 2026-07-02 ("Open in tab opened the remote session in a tab and the Session works as expected" — full-height PTY stream, no 403 blank, no dead band). |
| 4 | Footer "Share Session" button opens the Share modal for the active session with no independent state drift (UX-02) | ✓ VERIFIED | CR-01 (footer no-op) and CR-03 (cross-tab leak) sub-fixes regression-tested; the footer-pill-drift sub-bug is fixed by 168-08 (`SessionShareModal.onShareEnabledChange` → `App.tsx setWebEnabled`, proven at every seam by real component tests); live-confirmed by human UAT — 168-UAT.md Test 4 PASS on a production build 2026-07-02 (footer pill flips WEB ON/WEB OFF tracking the modal toggle, both directions, both open paths, shell warning dismissed — no stale-label drift). |
| 5 | "Stay on Hub after creating session" toggle prevents auto-switch when ON (UX-01) | ✓ VERIFIED | `internal/daemon/engine_stayonhub_test.go` / `api_stayonhub_test.go` and `App.createTab.stayOnHub.test.tsx`; full suite green; untouched by 168-08/168-09. |
| 6 | A never-shared local session's Hub card reads 0 viewers (FIX-04) | ✓ VERIFIED | `TestRemoteViewerCount`, `TestListSessions_ViewerCount`, `TestAPI_ListSessionsViewerCount`; untouched by 168-08/168-09. |

**Score:** 6/6 truths ✓ VERIFIED. The two previously-open items (SC3 blocked on hardware, SC4 behavior-unverified) are now both backed by completed live human UAT plus code + automated coverage. 0 behavior-unverified.

### Fresh Code Verification — 168-09 Gap Closure (FIX-03 Remote Web-Session Tab, SC3 / #118)

Checked directly against the current codebase (not trusted from 168-09-SUMMARY.md):

| Item | File:Line | Verified |
|---|---|---|
| RC-A: `remote?: boolean` prop declared on WebShareSessionView | `frontend/src/components/Hub/WebShareSessionView.tsx:30` | ✓ Present, optional |
| RC-A: `const useProxy = remote === true` transport switch | `WebShareSessionView.tsx:68` | ✓ Present |
| RC-A: TerminalPanel gets `remote={useProxy}` and `wsURL={useProxy ? undefined : wsURL}` (daemon proxy, not direct cross-origin peer wss) | `WebShareSessionView.tsx:154-155` | ✓ Confirmed — proxy path when remote, direct wsURL preserved for native web-guest |
| RC-A follow-up: ChatPanel + chat toggle gated behind `{!useProxy && …}` (chat hidden on remote tabs) | `WebShareSessionView.tsx:164` | ✓ Confirmed — native path keeps chat |
| RC-A: App threads `remote={isRemoteWebTab}` on the `__websession__` WebShareSessionView | `frontend/src/App.tsx:1703, 1727` | ✓ Confirmed |
| RC-B: `__websession__` branch wrapped in `<div className="terminal-wrapper" style={{ display: 'flex' }}>` (full-height flex-column chain, parity with normal terminal tabs) | `App.tsx:1715` | ✓ Confirmed |
| RC-C: SessionCard remote menu item text "Open in tab" (stale "Open in browser" gone) + in-app `WindowIcon` (external-link glyph dropped) | `frontend/src/components/Hub/SessionCard.tsx:414-415, 20` | ✓ Confirmed — grep finds no residual "Open in browser" |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/hub.go: RemoteViewerCount()` | Web-origin-only viewer count | ✓ VERIFIED | FIX-04, unchanged |
| `internal/relay/hub.go: DisconnectWebViewers()` | Force-close web-origin subscribers | ✓ VERIFIED | FIX-02, unchanged |
| `frontend/src/components/Hub/WebShareSessionView.tsx` | Web-guest live plugin-config + remote transport switch | ✓ VERIFIED | FIX-01 (live Test 1); FIX-03 `remote`/`useProxy` (168-09) |
| `frontend/src/App.tsx` `__websession__` branch | `.terminal-wrapper` wrapper + `remote={isRemoteWebTab}` | ✓ VERIFIED | 168-09 RC-B |
| `frontend/src/components/Hub/SessionCard.tsx` remote menu item | "Open in tab" + WindowIcon | ✓ VERIFIED | 168-09 RC-C |
| `frontend/src/components/SettingsTab.tsx` stay-on-hub toggle | Default OFF | ✓ VERIFIED | UX-01, unchanged |
| `frontend/src/components/Hub/SessionShareModal.tsx` `onShareEnabledChange` | Fires on ON/OFF after successful toggle | ✓ VERIFIED | UX-02 / 168-08 |
| `frontend/src/App.tsx` `<SessionShareModal>` wiring | `onShareEnabledChange` → `setWebEnabled` | ✓ VERIFIED | UX-02 / 168-08 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| WebShareSessionView (remote) | daemon WS proxy `ws://127.0.0.1:{relayPort}/api/relay/remote/{id}/ws` | `useProxy` → TerminalPanel `remote:true`, no `wsURL` | ✓ WIRED | Proven by `WebShareSessionView.test.tsx` remote-branch tests; live-confirmed Test 3 (real PTY bytes, no 403) |
| `__websession__` render | full-height pane | `.terminal-wrapper` flex-column wrapper | ✓ WIRED | Source-verified `App.tsx:1715`; live-confirmed Test 3 (no dead band) |
| SessionShareModal toggle | `App.webEnabled` | `onShareEnabledChange` → `setWebEnabled` | ✓ WIRED | Component tests; live-confirmed Test 4 (footer pill tracks toggle) |
| `webEnabled[sessionId]` | StatusBar pill label | `!!webEnabled[tab.sessionId]` → "WEB ON"/"WEB OFF" | ✓ WIRED | StatusBar render tests; live-confirmed Test 4 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| 168-09 targeted vitest | `pnpm vitest run WebShareSessionView.test App.open-remote SessionCard.share` | 3 files / 58 tests pass | ✓ PASS |
| Full frontend vitest suite | `pnpm vitest run` | 141 files / 2329 tests pass | ✓ PASS |
| Frontend typecheck | `pnpm exec tsc --noEmit` | clean, exit 0 | ✓ PASS |
| Traceability checker | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist`, exit 0 (BSD `grep -P` warning is a known macOS quirk, non-fatal) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| FIX-01 | 168-02 (+07) | Web-guest plugin-config self-fetch + SSE hot-swap | ✓ SATISFIED | Code + behavioral test + human UAT Test 1 PASS |
| FIX-02 | 168-06 (+07) | Multi-viewer + owner disconnect | ✓ SATISFIED | Code + Go race tests + human UAT Test 2 PASS |
| FIX-03 | 168-03, **168-09** | Remote session opens in-app tab, streams | ✓ SATISFIED | 168-09 RC-A/RC-B/RC-C code + 58 tests + human UAT Test 3 PASS (live two-Mac) |
| FIX-04 | 168-01 (+07) | Hub viewer count excludes local subscribers | ✓ SATISFIED | Go tests, unchanged |
| UX-01 | 168-04 (+07) | Stay-on-Hub-after-create toggle | ✓ SATISFIED | Go + frontend tests, unchanged |
| UX-02 | 168-05, 168-REVIEW-FIX, **168-08** | Footer Share button, no state drift / pill drift | ✓ SATISFIED | Code + component tests + human UAT Test 4 PASS (live) |

No orphaned requirements — REQUIREMENTS.md maps exactly FIX-01..04, UX-01, UX-02 to Phase 168, matching the union of every plan's `requirements:` frontmatter.

Note: the goal statement says "clearing Issues #112, #115, #116, #117, #118, #121". Issue closure on GitHub is a manual post-verification project-management step in this project's established workflow (no plan task runs `gh issue close`), not a code-verifiable artifact — not treated as a gap, consistent with the prior pass. Out-of-scope enhancement/bug findings raised during UAT (#125 no-disconnect-notice, #126 shared-tab auto-close, direct on-card "Open in tab" button suggestion, #107 read-write comment) were filed separately and explicitly do not gate this phase.

### Anti-Patterns Found

None. Scanned the 168-09-touched files (`WebShareSessionView.tsx`, `App.tsx`, `SessionCard.tsx`, the extended test files, `TESTING.md`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented" — none found. Prior-phase files re-confirmed clean; full suite green.

## Human Verification Required

None outstanding. Both previously-open items are now closed by completed live human UAT (168-UAT.md, status: complete, 4/4 pass, 0 issues):

- **SC3 (FIX-03)** — was carried forward as blocked on a second Mac. Now PASS live on a two-Mac tailnet production build (Test 3), validating 168-09's D1 (daemon-proxy PTY stream) and D2 (full-height render) human-judgment items.
- **SC4 (UX-02)** — was behavior-unverified pending a full-chain live re-check. Now PASS live on a production build (Test 4), confirming the footer pill tracks the modal toggle end-to-end with no drift. TESTING.md manual item M-44 was added for standing coverage of this regression class.

## Gaps Summary

No gaps. All 6 success criteria (FIX-01..04, UX-01, UX-02) have working, wired, tested implementations, and every behavior-dependent criterion (SC3 remote streaming, SC4 pill-flip state transition) is backed by completed live human UAT in addition to automated coverage. 168-09 correctly root-caused the FIX-03 remote-tab failure into two independent defects (RC-A direct cross-origin wss 403'd by the peer Origin allowlist → rerouted through the daemon proxy; RC-B missing `.terminal-wrapper` full-height chain → wrapped) plus the RC-C stale label, and all three are verified present in the current codebase. `tsc --noEmit`, the full 141-file/2329-test vitest suite, and `check-traceability-paths.sh` are all clean. Phase goal achieved — ready to proceed.

---

_Verified: 2026-07-02T15:53:47Z_
_Verifier: Claude (gsd-verifier)_
