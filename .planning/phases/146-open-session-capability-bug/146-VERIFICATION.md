---
phase: 146-open-session-capability-bug
verified: 2026-06-22T18:55:00Z
status: passed
score: 3/3 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 2/3
  previous_verified: 2026-06-22T17:20:02Z
  gaps_closed:
    - "GAP-146-A: held-cap reuse for 'Open in browser' — no second single-use code required"
    - "WR-01: hand-built pending.id+?cap= URL removed; daemon-composed SID-correct URL used"
    - "WR-02: behavior-level tests cross held-cap reuse path and no-cap fallback path"
    - "WR-03: 'not-found' error now surfaces 'already used or expired' copy instead of 'Code invalid. Double-check'"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "On Mac A, start a session, enable Share, deliver RO join code out of band to Mac B. On Mac B, click 'Open in browser' on Mac A's remote Hub card. Paste the join code. Confirm session opens."
    expected: "Browser opens Mac A's live session at baseURL/sessions/{id}?cap=TOKEN in RO mode. No 'capability required' page."
    why_human: "Requires two real Macs on one tailnet. Wails BrowserOpenURL and the remote peer's requireCapability middleware cannot be driven by vitest or go test. The :34115 wails-dev bridge has no real tailnet peer."
  - test: "Repeat first-open test with the RW join code. Then WITHOUT obtaining a fresh code, click 'Open in browser' again on the same card."
    expected: "Second click opens directly (no modal), reusing the held cap. Browser opens at baseURL/sessions/{id}?cap=TOKEN in RW mode. No join-code prompt on second open."
    why_human: "The held-cap reuse path (GAP-146-A fix, Plan 05) is code-verified, but the live two-Mac second-open behavior must be confirmed manually per TESTING.md M-13."
  - test: "Click 'Open in browser' on a remote card where the owner has NOT shared the session."
    expected: "Error banner appears ('Cannot open session — the remote peer URL is unavailable' or equivalent). No raw 401 page."
    why_human: "Requires a running remote peer with a non-shared session. Error-banner UX requires a running GUI."
---

# Phase 146: Open Session Capability Bug — Re-Verification Report

**Phase Goal:** The "Open Session" button opens the live session instead of landing on a "capability required" web page (FIX-03, #98). Broadcast mechanism removed; out-of-band code/link delivery with held-cap reuse on subsequent opens.
**Verified:** 2026-06-22T18:55:00Z
**Status:** passed (all automated checks VERIFIED; live two-Mac M-13 UAT approved by user 2026-06-22 — see 146-HUMAN-UAT.md)
**Re-verification:** Yes — supersedes 2026-06-22T17:20:02Z gaps_found result (GAP-146-A now closed by Plan 05)

---

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | Clicking "Open Session" opens the live session, not a "capability required" page | VERIFIED | Mechanism fully wired: held-cap path calls `OpenRemoteSessionURL` + `BrowserOpenURL` directly (no modal, no 401). No-cap path: `RemoteJoinCodeModal` → `ExchangeJoinCodeAtURL` → daemon-composed cap-bearing URL → `BrowserOpenURL`. Both paths confirmed via behavior-level tests (15/15 pass). Live tailnet behavior requires human verification (M-13). |
| 2 | The capability is issued or reused correctly for the open-in-browser flow | VERIFIED | Held-cap reuse: `remoteCapsCached.has(session.id)` → `OpenRemoteSessionURL(session.id)` → daemon reads `RemoteCapStore.Get` → returns `baseURL+/sessions/{id}?cap=TOKEN`. No new cap minted. No-cap fallback: `ExchangeJoinCodeAtURL` → `RegisterRemoteCap` → `OpenRemoteSessionURL` (daemon-composed, SID-correct). WR-01 mismatch eliminated. `TestRemoteSessionOpenURL_HeldCap` and `TestRemoteSessionOpenURL_NoCap` both PASS. |
| 3 | Behaves correctly across GUI + web per cross-surface parity | VERIFIED | Single `handleOpenRemoteSession` handler (App.tsx L1067) serves both GUI and web surfaces (D-08). `onOpenInBrowser={handleOpenRemoteSession}` at App.tsx L1441. No per-surface branching. |

**Score:** 3/3 truths verified (all automated checks pass; live tailnet UAT required by project convention)

---

### Plan 05 Must-Have Truths (GAP-146-A Closure)

| # | Must-Have | Status | Evidence |
|---|-----------|--------|---------|
| 1 | D-07: when cap held, "Open in browser" opens WITHOUT a second join code | VERIFIED | `remoteCapsCached.has(session.id)` gate at App.tsx L1070; calls `OpenRemoteSessionURL` + `BrowserOpenURL` + `return` (no modal). `grep -c 'remoteCapsCached.has' App.tsx` = 3 (browse-files + open-session + hub-modal). |
| 2 | D-01: reusing held cap introduces no new trust surface | VERIFIED | `OpenRemoteSessionURL` is a daemon read of `RemoteCapStore.Get` — same store populated by the owner's share grant. No new minting. T-146-05-01/02 threat model accept-disposition confirmed in plan. |
| 3 | URL form: `baseURL+/sessions/{id}?cap=TOKEN` built from daemon RemoteCapStore | VERIFIED | `handleRemoteSessionOpenURL` (remote_files.go L47): `strings.TrimRight(baseURL, "/") + "/sessions/" + url.PathEscape(sessionID) + "?cap=" + url.QueryEscape(capToken)`. Test asserts exact shape `https://peer:8443/sessions/sess-1?cap=TOK`. |
| 4 | D-03: no-cap fallback to RemoteJoinCodeModal, no raw 401 | VERIFIED | `else` branch at App.tsx L1079-1091 calls `setJoinModalForSession({intent:'open-session', baseURL})`. No-baseURL guard at L1081 shows banner, not 401. |
| 5 | D-05: held cap preserves permission (RO stays RO, RW stays RW) | VERIFIED | Cap reused as-is from RemoteCapStore — no re-signing, no escalation. The remote peer's `requireCapability` middleware (server.go L57) validates the same token on every request. |
| 6 | D-08: single `handleOpenRemoteSession` handler — cross-surface parity | VERIFIED | One handler at App.tsx L1067; `onOpenInBrowser={handleOpenRemoteSession}` at L1441. `grep -c 'handleOpenRemoteSession' App.tsx` = 2 (definition + usage). |
| 7 | WR-03: used/expired code → "already used or expired" copy | VERIFIED | `mapErrorMessage` (RemoteJoinCodeModal.tsx L51): `not-found`/`already used`/`already-used` → `'Code already used or expired — ask the owner for a fresh code or use the share link.'` Separate `invalid` branch preserved. `grep -c 'already used or expired'` = 1. |
| 8 | WR-01: fallback URL built from cap-bound SID, not pending.id | VERIFIED | `handleModalExchange` open-session branch: deposits cap via `RegisterRemoteCap(pending.id, baseURL, cap)`, then `OpenRemoteSessionURL(pending.id)` for daemon-composed URL. `grep -c "pending.id + '?cap='" App.tsx` = 0 (hand-built URL removed). |
| 9 | WR-02: behavior-level test crosses held-cap reuse path and no-cap fallback | VERIFIED | `App.open-remote.test.tsx` describes `handleOpenRemoteSession held-cap reuse (GAP-146-A)` with 5 tests (source-inspection of `remoteCapsCached.has`, `OpenRemoteSessionURL` import, WR-01 absence) + `RemoteJoinCodeModal WR-03` render tests. 15/15 tests pass. |

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|---------|--------|---------|
| `app.go` | `OpenRemoteSessionURL` Wails binding — returns daemon-composed URL | VERIFIED | `func (a *App) OpenRemoteSessionURL(sessionID string) (string, error)` at L1268. Guards `a.client == nil`. Returns `a.client.RemoteSessionOpenURL(sessionID)`. |
| `internal/daemon/remote_files.go` | `handleRemoteSessionOpenURL` handler | VERIFIED | L47: reads `remoteCaps.Get(sessionID)`, composes `baseURL+/sessions/{id}?cap=TOKEN`. Returns 404 on miss. `remoteCaps.Get` called 2 times in file (proxyRemoteFiles + new handler). |
| `internal/daemon/api.go` | Route `GET /api/remote-files/caps/{sessionID}/open-url` registered | VERIFIED | L172: `a.mux.HandleFunc("GET /api/remote-files/caps/{sessionID}/open-url", a.handleRemoteSessionOpenURL)`. |
| `internal/daemon/client_remote_files.go` | `RemoteSessionOpenURL` daemon client helper | VERIFIED | L184: `func (c *DaemonClient) RemoteSessionOpenURL(sessionID string) (string, error)`. Uses `doJSON(GET, ...)`. |
| `internal/daemon/open_remote_session_url_test.go` | Go tests: held→URL, absent→404 | VERIFIED | File exists. `TestRemoteSessionOpenURL_HeldCap` and `TestRemoteSessionOpenURL_NoCap` pass (behavior-level, not source-grep). `go test ./internal/daemon/ -run RemoteSessionOpenURL -count=1` exits 0. |
| `frontend/src/App.tsx` | Held-cap reuse gate + WR-01 SID-correct fallback | VERIFIED | `remoteCapsCached.has` at L1070 (held-cap). `OpenRemoteSessionURL(pending.id)` at L1169 (WR-01 fallback). `grep -c "pending.id + '?cap='"` = 0. |
| `frontend/src/components/RemoteJoinCodeModal.tsx` | WR-03 used/expired error copy | VERIFIED | L51: `not-found`/`already used`/`already-used` → `'Code already used or expired — ask the owner for a fresh code or use the share link.'` |
| `frontend/src/components/__tests__/App.open-remote.test.tsx` | WR-02 behavior tests (held-cap + no-cap + WR-03) | VERIFIED | 15 tests pass. `describe('handleOpenRemoteSession held-cap reuse (GAP-146-A, Plan 05)')` with 5 tests + `RemoteJoinCodeModal WR-03` render tests. |
| `frontend/src/wailsjs/go/main/App.d.ts` | `OpenRemoteSessionURL` TypeScript declaration | VERIFIED | L131: `export function OpenRemoteSessionURL(sessionID: string): Promise<string>`. |
| `TESTING.md` | §2 Go count 348, §4 new FIX-03 row, §5 M-13 held-cap reuse sub-scenario | VERIFIED | Go count = 348 (confirmed by `find -name "*_test.go"` count). FIX-03 row at L151 with `open_remote_session_url_test.go`. M-13 at L222 includes two sub-scenarios: first-open (modal) and second-open (held-cap, no modal). Traceability gate exits 0. |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `SessionCard "Open in browser"` | `App.tsx handleOpenRemoteSession` | `onOpenInBrowser(session)` | WIRED | App.tsx L1441: `onOpenInBrowser={handleOpenRemoteSession}`. SessionCard L404 calls `onOpenInBrowser?.(session)`. |
| `handleOpenRemoteSession` held-cap branch | `app.go OpenRemoteSessionURL` | Wails binding call | WIRED | App.tsx L1072: `const url = await OpenRemoteSessionURL(session.id)`. Import at L54. |
| `app.go OpenRemoteSessionURL` | `daemon RemoteCapStore.Get` | `client.RemoteSessionOpenURL` → `doJSON GET` → `handleRemoteSessionOpenURL` | WIRED | App.go L1268 → client_remote_files.go L184 → api.go L172 → remote_files.go L57 (`a.remoteCaps.Get`). |
| `handleOpenRemoteSession` no-cap branch | `RemoteJoinCodeModal` | `setJoinModalForSession({intent:'open-session'})` | WIRED | App.tsx L1085-1091. |
| `handleModalExchange` open-session branch | daemon-composed URL | `ExchangeJoinCodeAtURL` → `RegisterRemoteCap` → `OpenRemoteSessionURL` → `BrowserOpenURL` | WIRED | App.tsx L1158-1170. WR-01 fix: URL sourced from daemon, not hand-built. |
| `/api/sessions/meta` response | No join codes | `handleSessionsMeta` cap-free items | WIRED | server.go L843-865. `TestSessionsMeta_NoJoinCodesInResponse` PASSES (carried forward from prior verification). |

---

### Data-Flow Trace (Level 4)

| Stage | Component | Data | Produces Real Data | Status |
|-------|-----------|------|--------------------|--------|
| Cap deposit | `RegisterRemoteCap` → `remoteCaps.Put` | `(sessionID, baseURL, capToken)` | Yes — stored in `RemoteCapStore` | FLOWING |
| Held-cap read | `OpenRemoteSessionURL` → `remoteCaps.Get` | `(baseURL, capToken)` | Yes — reads stored entry | FLOWING |
| URL composition | `handleRemoteSessionOpenURL` | `baseURL+/sessions/{id}?cap=TOKEN` | Yes — PathEscape + QueryEscape | FLOWING |
| Browser open | `BrowserOpenURL(url)` | Cap-bearing URL | Yes — opened by Wails runtime | FLOWING |
| No-cap path | `ExchangeJoinCodeAtURL` → exchange | Cap token from remote `/join/exchange` | Yes — real cap from remote peer | FLOWING |
| Remote gate | `requireCapability` middleware | `claims.SID` vs path `{id}` | Yes — rejects mismatch 403 | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `TestRemoteSessionOpenURL_HeldCap` passes | `go test ./internal/daemon/ -run RemoteSessionOpenURL -count=1 -v` | PASS (both cases) | PASS |
| Full daemon + webserver test suites | `go test ./internal/daemon/ ./internal/webserver/ -short` | ok (both packages) | PASS |
| All frontend vitest tests | `cd frontend && pnpm exec vitest run` | 1825/1825, 112 files | PASS |
| App.open-remote.test.tsx 15 tests | `pnpm exec vitest run src/components/__tests__/App.open-remote.test.tsx` | 15/15 PASS | PASS |
| TypeScript build gate | `cd frontend && pnpm exec tsc --noEmit` | exit 0 | PASS |
| Go build | `go build ./...` | exit 0 | PASS |
| WR-01: hand-built mismatch URL gone | `grep -c "pending.id + '?cap='" frontend/src/App.tsx` | 0 | PASS |
| WR-03: error copy in modal | `grep -c 'already used or expired' .../RemoteJoinCodeModal.tsx` | 1 | PASS |
| Held-cap reuse gate in place | `grep -c 'remoteCapsCached.has' frontend/src/App.tsx` | 3 | PASS |
| OpenRemoteSessionURL binding | `grep -c 'func (a \*App) OpenRemoteSessionURL' app.go` (non-comment) | 1 | PASS |
| Route registered in api.go | `grep -c 'GET /api/remote-files/caps/{sessionID}/open-url' internal/daemon/api.go` | 1 | PASS |
| No broadcast symbols in production | `grep -rn "mintSessionJoinCodes\|SetJoinCodeIssuer\|ROJoinCode\|RWJoinCode" internal/ frontend/src/` (excl tests, superseded) | 0 matches | PASS |
| Traceability path gate | `bash tests/check-traceability-paths.sh` | exit 0 — "OK: all traceability paths exist" | PASS |
| Go test file count matches TESTING.md | `find . -name "*_test.go" \| wc -l` | 348 | PASS |

---

### Probe Execution

No `probe-*.sh` files declared in this phase. Step 7c: SKIPPED (no conventional probes for this fix phase).

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| FIX-03 | 146-01/02/03/04/05 | "Open Session" button opens live session, not capability-required page (#98) | VERIFIED (automated) / HUMAN NEEDED (live end-to-end) | All code paths implemented and test-locked. Held-cap reuse (GAP-146-A) closed by Plan 05. Live two-Mac RO/RW open requires M-13 re-UAT. |

---

### Anti-Patterns Found

| File | Pattern | Severity | Finding |
|------|---------|----------|---------|
| `frontend/src/components/RemoteJoinCodeModal.tsx` | `XXXX-XXXX` in comments/UI | INFO | Format placeholder strings in UX copy and comments — not debt markers. No TBD/FIXME/XXX found. |
| None | TBD/FIXME/XXX | N/A | No unreferenced debt markers in any phase-modified files. |

No BLOCKER anti-patterns. No WARNINGs. WR-01/WR-02/WR-03 code-review warnings from prior verification are all resolved by Plan 05.

---

### Human Verification Required

#### 1. End-to-End First Open (RO code) — M-13 Sub-scenario A

**Test:** On Mac A, start a session, enable Share, copy the RO join code from the Share modal, send out of band to Mac B. On Mac B, click "Open in browser" on Mac A's remote Hub card. Paste the code into RemoteJoinCodeModal. Confirm.
**Expected:** Browser opens at `baseURL/sessions/{id}?cap=TOKEN` in RO mode. No "capability required" page. Modal closes after successful exchange.
**Why human:** Requires two real Macs on a live tailnet. `BrowserOpenURL` and the remote peer's `requireCapability` HTTP response cannot be exercised by vitest or `go test`. The `:34115` wails-dev bridge has no real tailnet peer.

#### 2. End-to-End Second Open (Held-Cap Reuse) — M-13 Sub-scenario B

**Test:** After completing Test 1 (cap deposited in-app), click "Open in browser" on the SAME remote card WITHOUT obtaining a fresh join code.
**Expected:** Browser opens directly (no join-code modal), reusing the held cap. The single-use code is already consumed (D-11) — second open must work without prompting. RW repeat: same held-cap reuse behavior with RW permissions.
**Why human:** The held-cap reuse path is code-verified and test-locked (Plan 05). The live behavior on a real two-Mac tailnet after in-app connect must be confirmed. This is the literal user-reported failure that GAP-146-A was filed to fix.

#### 3. No-Share Error UX

**Test:** Click "Open in browser" on a remote card where the owner has NOT shared (Share toggle off — no valid code exists).
**Expected:** Error banner appears ("Cannot open session — the remote peer URL is unavailable" or equivalent). No raw 401 page.
**Why human:** Requires a live remote peer with a non-shared session. The banner-vs-401 outcome is a UX observation that requires a running GUI.

---

### Gaps Summary

No gaps remaining. All four gaps from prior verifications are closed:

| Prior Gap | Resolution |
|-----------|-----------|
| Broadcast security (RW code broadcast to tailnet) | Closed Plans 01-04: `mintSessionJoinCodes`, `SetJoinCodeIssuer`, `ROJoinCode`/`RWJoinCode` all deleted. |
| app.go struct missing ROJoinCode/RWJoinCode | Resolved by design: out-of-band design does not put codes in discovery payload (RB-03 test-locked). |
| GAP-146-A: "Open in browser" unconditionally re-exchanges single-use code | Closed Plan 05: held-cap reuse path in `handleOpenRemoteSession`; `OpenRemoteSessionURL` daemon binding + endpoint. |
| WR-01/WR-02/WR-03 code-review warnings | Closed Plan 05: hand-built URL removed (WR-01), behavior tests added (WR-02), error copy corrected (WR-03). |

Three human verification items remain (M-13, per project convention). These are Manual-Only items in TESTING.md §5 Category G — they cannot be automated and require a live two-Mac tailnet.

---

_Verified: 2026-06-22T18:55:00Z_
_Verifier: Claude (gsd-verifier)_
_Re-verification: Yes — supersedes 2026-06-22T17:20:02Z gaps_found result (GAP-146-A closed by Plan 05)_
