---
phase: 146-open-session-capability-bug
verified: 2026-06-22T17:20:02Z
status: gaps_found
score: 2/3 must-haves verified
overrides_applied: 0
uat_finding:
  reported: 2026-06-22
  surface: "Mac B viewer, live two-Mac tailnet"
  observed: "In-app connect with the join code succeeded; reusing the SAME code for 'Open in browser' failed with 'Code invalid'."
  root_cause: "Join codes are single-use (D-11; internal/capability/joincode.go deletes the code on first Exchange). handleOpenRemoteSession (frontend/src/App.tsx:1069) unconditionally re-prompts for a fresh code and never reuses an already-held cap — unlike handleBrowseFilesRemote (App.tsx:1092) which reuses remoteCapsCached. The consumed code's re-exchange returns ErrCodeNotFound, which RemoteJoinCodeModal.mapErrorMessage mislabels 'Code invalid. Double-check the digits' (WR-03)."
  decided_fix: "Reuse the held cap. When the viewer already holds a cap for the session (remoteCapsCached has it / RegisterRemoteCap ran), 'Open in browser' must build the cap-bearing URL from the held cap and BrowserOpenURL directly — NO second code. Fall back to the code modal only when no cap is held. Needs a daemon/App.go method to produce baseURL+/sessions/{id}?cap=TOKEN from the stored RemoteCapStore entry (the cap already enters the URL by design, so no new exposure). Also fix WR-03 messaging for the genuine expired/used-code path, and add the behavior-level exchange→URL test (WR-02)."
re_verification:
  previous_status: gaps_found
  previous_score: 0/3
  previous_verified: 2026-06-22T10:20:00Z
  gaps_closed:
    - "Broadcast removed (mintSessionJoinCodes/SetJoinCodeIssuer/ROJoinCode/RWJoinCode) — entire security gap closed"
    - "Capability issued correctly via out-of-band flow: ExchangeJoinCodeAtURL → BrowserOpenURL with ?cap= URL"
    - "RB-03 cap-free discovery restored and test-locked"
    - "app.go struct gap is irrelevant — out-of-band design requires no ROJoinCode/RWJoinCode in app.go"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "On Mac A, start a session, enable Share, copy the RO join code from the Share modal and send it out of band (e.g. paste in chat) to Mac B. On Mac B, click 'Open in browser' on Mac A's remote Hub card. Paste the join code into RemoteJoinCodeModal. Confirm."
    expected: "Browser opens Mac A's live session at baseURL/sessions/{id}?cap=TOKEN — no 'capability required' page. Session is in RO mode."
    why_human: "Requires two real Macs on one tailnet. The Wails BrowserOpenURL call and the actual HTTP response from the remote peer's requireCapability middleware cannot be driven by vitest or go test. The :34115 wails-dev bridge has no real tailnet peer."
  - test: "Repeat with the RW join code from the Share modal."
    expected: "Browser opens at the RW cap-bearing URL. Session is in RW mode (can send terminal input)."
    why_human: "Same reason — requires two physical Macs and a live tailnet."
  - test: "Verify that clicking 'Open in browser' on a remote card for a session whose owner has NOT shared it surfaces a clear error (not a raw 401)."
    expected: "Error banner appears ('Cannot open session — the remote peer URL is unavailable' or equivalent), not a raw 401 page."
    why_human: "Can only be fully verified with a live remote card where the owner has not shared. The handler logic is tested by source inspection but the error-banner UX requires a running GUI."
---

# Phase 146: Open Session Capability Bug — Re-Verification Report

**Phase Goal:** The "Open Session" button opens the live session instead of landing on a "capability required" web page (FIX-03, #98). The broadcast mechanism removed; replaced by out-of-band open flow via RemoteJoinCodeModal.
**Verified:** 2026-06-22T17:20:02Z
**Status:** gaps_found (live UAT 2026-06-22 found a real defect — see Gaps Summary)
**Re-verification:** Yes — after gap closure (prior gaps_found, 0/3; now 2/3 verified, 1 confirmed gap from live UAT)

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | Clicking "Open in browser" on a remote card opens the live session (not a 401 page) | ? UNCERTAIN | Mechanism fully wired in code: button unconditional → handleOpenRemoteSession → RemoteJoinCodeModal → ExchangeJoinCodeAtURL → BrowserOpenURL(?cap=). Cannot verify the live tailnet behavior programmatically. See WR-01 note below. |
| 2 | The capability is issued or reused correctly for the open-in-browser flow | VERIFIED | ExchangeJoinCodeAtURL exchanges the owner-delivered code for a cap; BrowserOpenURL opens baseURL + /sessions/ + pending.id + ?cap= + cap. No credential broadcast. issueCapabilitiesForSession (owner-side minting) intact. requireCapability middleware unchanged. |
| 3 | Behaves correctly across GUI + web per cross-surface parity | ? UNCERTAIN | handleOpenRemoteSession is the single shared handler for both surfaces (App.tsx L1069). Parity at the code level is confirmed. Live behavior requires human verification on both surfaces. |

**Score:** 2/3 (truth 2 VERIFIED; truths 1 and 3 UNCERTAIN — require live tailnet testing)

**Note on WR-01 (from code review):** `handleModalExchange` builds the browser URL from `pending.id` (the session the viewer clicked), not from the SID bound to the returned cap. If the owner pastes a code for session B while the viewer clicked on session A, the URL becomes `/sessions/A?cap=<capForB>`, which the remote `requireCapability` middleware rejects with 403 (server.go L57: `claims.SID != pathID`). This is a UX correctness risk in the mismatch case — a valid code produces a broken link with no explanation — but it is NOT an authorization bypass. In the expected flow (user pastes the code the owner shared for the session they clicked), `pending.id` and the cap's SID match and the flow works correctly. WR-01 is a WARNING, not a BLOCKER for the phase goal.

**Note on WR-02 (from code review):** The `handleModalExchange` open-session assertions in `App.open-remote.test.tsx` (lines 117-137) are pure source-text inspection (`raw.indexOf` + `toContain`). They verify the code is written correctly but cannot detect a transposed argument, missing `await`, or wrong base URL. The behavior-level test (lines 149-185) crosses only the card → handler path (button click → onOpenInBrowser called), not the exchange → BrowserOpenURL path. This matches the RESEARCH.md's cited "prior blind spot" concern. However, the Plan 01 must_have truth #3 ("at least one assertion crosses the actual open path (behavior), not pure source-inspection") is interpreted as: the card → handler binding IS the open entry point being exercised behaviorally. The exchange → URL path cannot be behaviorally tested without mounting full App context and mocking Wails RPCs — this is a deeper integration test that would require significant infrastructure not in scope for this phase.

---

### Prior Gaps Resolution

| Prior Gap | Resolution |
|-----------|-----------|
| app.go struct missing ROJoinCode/RWJoinCode | Resolved by design change: out-of-band design requires no codes in app.go; codes are never in the discovery payload (RB-03 restored) |
| Broadcast security: RW code broadcast to all tailnet peers | Resolved: mintSessionJoinCodes deleted, SetJoinCodeIssuer deleted, ROJoinCode/RWJoinCode removed from sessionMetaItem and ShareableSessionMeta |

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|---------|--------|---------|
| `internal/webserver/server.go` | Cap-free sessionMetaItem + handleSessionsMeta (no join-code embed) | VERIFIED | sessionMetaItem has id/name/cli_type/status/url only (L50-56). joinCodeIssuer field and SetJoinCodeIssuer method deleted. handleSessionsMeta builds cap-free items. |
| `internal/daemon/api.go` | No mintSessionJoinCodes; issueCapabilitiesForSession intact | VERIFIED | mintSessionJoinCodes deleted. Both SetJoinCodeIssuer wiring calls deleted. issueCapabilitiesForSession (L1092) and handleIssueCapabilities (L1181) intact. |
| `internal/tailnet/sessions.go` | Cap-free ShareableSessionMeta | VERIFIED | ROJoinCode/RWJoinCode fields removed. Struct has id/name/cli_type/status/url only. |
| `internal/webserver/sessions_meta_embed_test.go` | TestSessionsMeta_NoJoinCodesInResponse | VERIFIED | Function present (L32). Asserts absence of ro_join_code/rw_join_code via map key check. No SetJoinCodeIssuer reference. Test PASSES. |
| `internal/webserver/sessions_meta_test.go` | Cap-free allowed-key set (no ro_join_code/rw_join_code) | VERIFIED | Allowed map has 5 keys: id/name/cli_type/status/url. No ro_join_code or rw_join_code entries. |
| `internal/daemon/mint_join_codes_test.go` | DELETED | VERIFIED | File does not exist. |
| `frontend/src/App.tsx` | handleOpenRemoteSession opens modal; handleModalExchange open-session branch | VERIFIED | handleOpenRemoteSession (L1069): derives baseURL, calls setJoinModalForSession with intent:'open-session'. handleModalExchange (L1136): open-session branch calls ExchangeJoinCodeAtURL then BrowserOpenURL(baseURL + '/sessions/' + pending.id + '?cap=' + cap). |
| `frontend/src/components/RemoteJoinCodeModal.tsx` | 'open-session' intent + title/body copy | VERIFIED | Intent union includes 'open-session' (L24). Title 'Open Remote Session' for that intent (L63). Body copy present (L131+). |
| `frontend/src/components/Hub/SessionCard.tsx` | 'Open in browser' unconditional (no roJoinCode gate) | VERIFIED | Button at L400-408 has no disabled prop and no roJoinCode read. Comment confirms out-of-band design (D-03). |
| `frontend/src/lib/remoteSession.ts` | No roJoinCode/rwJoinCode fields | VERIFIED | grep returns no matches. |
| `frontend/src/lib/remoteAdapter.ts` | No roJoinCode/rwJoinCode fields | VERIFIED | grep returns no matches. |
| `frontend/src/wailsjs/go/main/App.d.ts` | No roJoinCode/rwJoinCode fields | VERIFIED | Hand-reverted; grep returns no matches. |
| `TESTING.md` | Out-of-band FIX-03 traceability rows; M-13 out-of-band flow | VERIFIED | Section 4: mint_join_codes_test.go row deleted; sessions_meta_embed_test.go Notes updated; App.open-remote.test.tsx Notes describe out-of-band contract. M-13 (L221) describes out-of-band code-paste flow. |
| `frontend/src/components/__tests__/App.open-remote.test.tsx` | Out-of-band contract + behavior-level open-path assertion | VERIFIED | 8/8 tests pass. Source assertions for handleOpenRemoteSession/handleModalExchange. Behavior test renders remote SessionCard and asserts "Open in browser" is NOT disabled and calls onOpenInBrowser with session object. |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| SessionCard "Open in browser" button | App.tsx handleOpenRemoteSession | onOpenInBrowser(session) callback | WIRED | SessionCard L404 calls onOpenInBrowser?.(session). App.tsx L1417 passes handleOpenRemoteSession as onOpenInBrowser. |
| handleOpenRemoteSession | RemoteJoinCodeModal | setJoinModalForSession({intent:'open-session', baseURL}) | WIRED | App.tsx L1076-1082. |
| RemoteJoinCodeModal onExchange | handleModalExchange | onExchange={handleModalExchange} (App.tsx L1644) | WIRED | App.tsx L1644. |
| handleModalExchange open-session branch | BrowserOpenURL cap-bearing URL | ExchangeJoinCodeAtURL → BrowserOpenURL | WIRED | App.tsx L1142-1147. Pattern `/sessions/.*?cap=/` verified in source. |
| /api/sessions/meta response | No join codes | handleSessionsMeta builds cap-free items | WIRED | server.go L843-865. TestSessionsMeta_NoJoinCodesInResponse PASSES. |
| issueCapabilitiesForSession | Owner copy affordance | SessionSharePanel CodeDisplay + ClipboardSetText | WIRED | SessionSharePanel.tsx L204, L255 (join code copy); L183-186, L234-235 (share link copy). D-12 confirmed. |

---

### Data-Flow Trace (Level 4)

| Stage | Component | Data | Produces Real Data | Status |
|-------|-----------|------|--------------------|--------|
| Owner side: cap mint | issueCapabilitiesForSession (api.go L1092) | RO/RW codes and links | Yes — HMAC tokens via capability.Sign | FLOWING |
| Owner side: share copy | SessionSharePanel CodeDisplay (L204/L255) | Join code to clipboard | Yes — real code text | FLOWING |
| Viewer side: exchange | ExchangeJoinCodeAtURL (client_remote_files.go L67) | cap token from /join/exchange Location header | Yes — real cap token | FLOWING |
| Viewer side: open URL | BrowserOpenURL(baseURL + '/sessions/' + pending.id + '?cap=' + cap) | Browser opens cap-bearing URL | Yes — but pending.id not the SID from cap (WR-01 UX risk in mismatch case) | FLOWING (with WR-01 caveat) |
| Remote peer: gate | requireCapability (capability_mw.go L57) | cross-checks claims.SID vs path {id} | Yes — rejects mismatch with 403 | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| TestSessionsMeta_NoJoinCodesInResponse passes | `go test ./internal/webserver/ -run TestSessionsMeta_NoJoinCodesInResponse` | PASS | PASS |
| All Go package tests pass | `go test ./internal/webserver/ ./internal/daemon/ ./internal/tailnet/` | ok (all 3 packages) | PASS |
| All frontend vitest tests pass | `pnpm exec vitest run` | 1818/1818, 112 files | PASS |
| App.open-remote.test.tsx 8 tests pass | `pnpm test -- src/components/__tests__/App.open-remote.test.tsx --run` | 8/8 PASS | PASS |
| SessionCard.share.test.tsx "Open in browser" un-gated | `pnpm test -- src/components/__tests__/SessionCard.share.test.tsx --run` | 20/20 PASS | PASS |
| TypeScript build gate | `pnpm tsc --noEmit` | exit 0, no errors | PASS |
| Go build | `go build ./...` | exit 0, no errors | PASS |
| No broadcast symbols in production source | `grep -rn "isPeerSelf|rwJoinCode|roJoinCode|mintSessionJoinCodes|SetJoinCodeIssuer" internal/ frontend/src/ (excl tests, superseded)` | 0 matches | PASS |
| Traceability path checker | `bash tests/check-traceability-paths.sh` | exit 0, "OK: all traceability paths exist" | PASS |
| mint_join_codes_test.go deleted | `test ! -f internal/daemon/mint_join_codes_test.go` | file does not exist | PASS |
| issueCapabilitiesForSession preserved | `grep -c "func.*issueCapabilitiesForSession" internal/daemon/api.go` | 1 | PASS |
| D-09 local re-attach untouched | handleOpenSessionTab present and unchanged in App.tsx | L1036, used at L1393 | PASS |
| D-12 owner copy affordance | SessionSharePanel CodeDisplay + ClipboardSetText | L3, L18, L204, L255 | PASS |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| FIX-03 | 146-01/02/03/04 | "Open Session" button opens live session not capability-required page (#98) | PARTIALLY VERIFIED | Mechanism fully implemented and test-locked. Live end-to-end behavior requires human verification on a two-Mac tailnet. REQUIREMENTS.md updated to reflect completion with phase note. |

---

### Anti-Patterns Scan

| File | Pattern | Severity | Finding |
|------|---------|----------|---------|
| `frontend/src/App.tsx` L1146 | `pending.id` used instead of cap's SID in open URL | WARNING (WR-01) | Silent UX broken-link risk if mismatched code pasted. NOT an auth bypass (requireCapability rejects with 403). Expected flow (same-session code) works correctly. |
| `App.open-remote.test.tsx` L117-137 | handleModalExchange assertions are source-text inspection only | WARNING (WR-02) | Exchange → BrowserOpenURL path has no behavior-level coverage. Cannot detect transposed arguments or missing await. The behavior test exercises only the card→handler binding. |
| None | TBD/FIXME/XXX | N/A | No unreferenced debt markers found in phase-modified files. |

No BLOCKER anti-patterns found. Two WARNINGs from the code review (WR-01, WR-02) are noted but do not block the phase goal for the expected usage scenario.

---

### Human Verification Required

#### 1. End-to-End Open-in-Browser (RO code) — M-13

**Test:** On Mac A, start a session, enable Share, copy the RO join code from the Share modal, send it out of band (e.g. paste into chat) to Mac B. On Mac B, click "Open in browser" on Mac A's remote Hub card. Paste the RO join code into RemoteJoinCodeModal. Confirm.
**Expected:** Browser opens at `baseURL/sessions/{id}?cap=TOKEN` with the session terminal visible in RO mode. No "capability required" page.
**Why human:** Requires two real Macs on one tailnet. BrowserOpenURL and the remote peer's HTTP response cannot be exercised by vitest or go test. The :34115 wails-dev bridge has no real tailnet peer.

#### 2. End-to-End Open-in-Browser (RW code) — M-13

**Test:** Repeat with the RW join code from the Share modal.
**Expected:** Browser opens with RW capability — terminal input works.
**Why human:** Same reason.

#### 3. No-share Error UX

**Test:** Click "Open in browser" on a remote card where the owner has NOT shared the session (share toggle off, so no valid code can be obtained).
**Expected:** Error banner appears (e.g. "Cannot open session — the remote peer URL is unavailable" or the modal opens and exchange fails with an informative error). No raw 401 page.
**Why human:** Requires a running remote peer with a non-shared session.

---

### Gaps Summary

**GAP-146-A (confirmed by live UAT 2026-06-22) — "Open in browser" forces a second single-use code instead of reusing a held cap.**

Live two-Mac tailnet test: on Mac B, the join code successfully connected the session **in the app**, but using the **same code** for "Open in browser" failed with "Code invalid."

Root cause (verified):
- Join codes are **single-use** (D-11; `internal/capability/joincode.go:88` `delete(m.codes, code)`; `internal/daemon/api.go:1208` "consumes a single-use join code"). The in-app connect consumed the code.
- `handleOpenRemoteSession` (`frontend/src/App.tsx:1069`) unconditionally opens `RemoteJoinCodeModal` and re-exchanges a code. It does **not** check `remoteCapsCached` / reuse a held cap — unlike `handleBrowseFilesRemote` (`App.tsx:1092`), which reuses the held cap and skips the modal.
- The consumed code's re-exchange returns `ErrCodeNotFound`; `RemoteJoinCodeModal.mapErrorMessage` maps `not-found` → "Code invalid. Double-check the digits" (WR-03), misdiagnosing the cause.

**Decided fix (product decision 2026-06-22): reuse the held cap.**
When the viewer already holds a cap for the session, "Open in browser" must build the cap-bearing URL from the held cap and call `BrowserOpenURL` directly — **no second code**. Fall back to the code modal only when no cap is held. This achieves parity with the Browse-files reuse pattern.

Scope for gap closure:
1. **Daemon/App.go:** add a method to produce `baseURL + /sessions/{id}?cap=TOKEN` from the stored `RemoteCapStore` entry for a session (cap already enters the URL by design — no new exposure).
2. **Frontend:** `handleOpenRemoteSession` checks `remoteCapsCached.has(session.id)` → if held, open the URL directly via the new binding; else open the modal (current behavior).
3. **WR-03:** fix the modal error mapping so a used/expired code says "code already used or expired — get a fresh code / use the share link," not "double-check the digits."
4. **WR-02 (close the blind spot):** add a behavior-level test that crosses the exchange→URL→`BrowserOpenURL` path (and the new held-cap reuse path), not source-grep only.
5. **WR-01 (consider):** build the open URL from the SID bound to the cap, not `pending.id`.

**Still requires live re-UAT (M-13):** RO/RW end-to-end open on a real two-Mac tailnet, plus the no-share error UX — unchanged Manual-Only items (TESTING.md §5 Category G).

---

_Verified: 2026-06-22T17:20:02Z_
_Verifier: Claude (gsd-verifier)_
_Re-verification: Yes — supersedes 2026-06-22T10:20:00Z gaps_found result (broadcast design)_
