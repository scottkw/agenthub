---
phase: 146-open-session-capability-bug
reviewed: 2026-06-22T00:00:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/App.open-remote.test.tsx
  - frontend/src/components/__tests__/SessionCard.share.test.tsx
  - frontend/src/components/Hub/SessionCard.tsx
  - frontend/src/components/RemoteJoinCodeModal.tsx
  - frontend/src/lib/__tests__/remoteAdapter.test.ts
  - frontend/src/lib/remoteAdapter.ts
  - frontend/src/lib/remoteSession.ts
  - frontend/src/wailsjs/go/main/App.d.ts
  - internal/daemon/api.go
  - internal/tailnet/sessions.go
  - internal/webserver/server.go
  - internal/webserver/sessions_meta_embed_test.go
  - internal/webserver/sessions_meta_test.go
findings:
  critical: 0
  warning: 3
  info: 4
  total: 7
status: issues_found
---

# Phase 146: Code Review Report

**Reviewed:** 2026-06-22
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

> NOTE: This file overwrites a stale REVIEW.md that reviewed the **superseded
> broadcast design** (`mintSessionJoinCodes`, `roJoinCode`/`rwJoinCode`,
> `isPeerSelf`, `SessionCardGrid.tsx`, `mint_join_codes_test.go`). That design was
> reversed to the out-of-band model (CONTEXT.md D-02/D-10). A repo-wide grep for
> `isPeerSelf|rwJoinCode|roJoinCode|ro_join_code|rw_join_code` returns zero hits in
> current production code, so the two prior BLOCKERs no longer apply. The findings
> below review the actually-submitted out-of-band implementation.

## Summary

FIX-03 (#98) removes the credential-broadcast mechanism (RO/RW join codes embedded
in the unauthenticated `/api/sessions/meta` payload) and replaces it with an
out-of-band open flow: the owner delivers a join code/link out of band, the viewer
pastes it into `RemoteJoinCodeModal`, and the browser opens at
`baseURL/sessions/{id}?cap=<token>`.

The four review priorities were checked:

1. **No credential leakage in discovery payload — PASS.** `sessionMetaItem`
   (server.go:50-56), `ShareableSessionMeta` (sessions.go:119-125), `RemoteSession`
   (remoteSession.ts:12-19, App.d.ts:105-112), and the adapter
   (`AdaptedRemoteSessionInfo`, remoteAdapter.ts:9-11) carry only
   `{id, name, cli_type, status, url}`. `handleSessionsMeta` (server.go:843-865) emits
   no token/grant/code. The inverted RB-03 test (sessions_meta_embed_test.go) and the
   exact-allowed-key-set test (sessions_meta_test.go:157-217) lock this down.

2. **Capability-URL construction — mostly correct, one correctness gap (WR-01).** The
   cap token is base64url (`A-Za-z0-9-_` + `.`), so the unencoded `?cap=` concatenation
   is injection-safe and matches the daemon's own construction (api.go:1163,
   server.go:797). The cap never enters React state — it lives in a local `const`.
   However, the open-session URL is built from `pending.id` rather than the exchanged
   cap's verified SID (WR-01).

3. **No dead broadcast code remains — PASS.** Grep confirms zero production hits for the
   broadcast identifiers. Type comments explicitly mark the removed fields.

4. **Local same-machine re-attach is untouched — PASS.** `handleOpenSessionTab`
   (App.tsx:1036-1050) is unchanged and remains wired to the local-only "Open" button
   (SessionCard.tsx:532-541, gated on `isLocal && session.state !== 'stopped'`). The
   regression-guard test (SessionCard.share.test.tsx:306-310) covers it.

No BLOCKER-class defects were found. Three WARNINGs and four INFO items follow.

## Warnings

### WR-01: open-session URL uses the clicked session id, not the exchanged cap's SID

**File:** `frontend/src/App.tsx:1142-1148`
**Issue:** The open-session branch builds the browser URL from `pending.id` (the session
the viewer clicked) but ignores the SID actually bound to the returned capability:

```js
const cap = await ExchangeJoinCodeAtURL(baseURL, code)
BrowserOpenURL(baseURL + '/sessions/' + pending.id + '?cap=' + cap)
```

Join codes are not session-scoped on the viewer side — the owner pastes whatever code
they were handed. If the owner shares a code minted for session B while the viewer
clicked "Open in browser" on session A, the URL becomes `/sessions/A?cap=<capForB>`. The
remote `requireCapability` cross-checks `claims.SID` against the path `{id}`
(server.go:603-604), so this is NOT an authorization bypass — it produces a 401/403
dead-end. But it is a silent correctness/UX failure: the user pasted a valid code and
gets a broken link with no explanation. Contrast the daemon's own exchange handler
(api.go:1260) and the webserver redirect (server.go:797), both of which build the URL
from the verified `claims.SID`, never from a caller-supplied id.
**Fix:** Derive the path id from the exchanged token's claims. `ExchangeJoinCodeAtURL`
already parses `/sessions/<id>?cap=<token>` from the Location header
(client_remote_files.go:161-171) and discards the id — return it (or the full path) and
open that, so the opened URL always matches the cap's bound session:

```js
const { sid, cap } = await ExchangeJoinCodeAtURL(baseURL, code)
BrowserOpenURL(baseURL + '/sessions/' + sid + '?cap=' + cap)
```

### WR-02: `App.open-remote.test.tsx` proves the open-session contract only by source-text inspection

**File:** `frontend/src/components/__tests__/App.open-remote.test.tsx:117-137`
**Issue:** The file's own header (lines 13-15, 139-148) correctly criticizes source-only
tests, yet the `handleModalExchange` open-session assertions are themselves pure
`raw.indexOf(...)` / `toContain('open-session')` / `toMatch(/\/sessions\/.*\?cap=/)`
string checks against `App.tsx?raw`. They pass as long as the substrings exist anywhere
in the sliced text — they would still pass with the WR-01 bug present (`pending.id`
produces a matching `/sessions/...?cap=` substring) and cannot detect a transposed
argument, a missing `await`, or the wrong base URL. The behavior-level assertions in this
file (lines 149-185) only exercise `SessionCard`'s button → handler wiring, never
`handleModalExchange` itself. So the actual open flow (exchange → correct URL →
BrowserOpenURL) has no behavioral coverage.
**Fix:** Add a behavior test that mocks `ExchangeJoinCodeAtURL` to resolve a known token
and `BrowserOpenURL` to a spy, drives the open-session intent through `handleModalExchange`
(extract it to a testable unit if needed), and asserts `BrowserOpenURL` was called with the
exact expected `baseURL + '/sessions/' + <expected-sid> + '?cap=' + <token>`. This covers
the path and would catch WR-01 as a regression.

### WR-03: `mapErrorMessage` collapses "not-found" into the wrong-code bucket, mis-directing recovery

**File:** `frontend/src/components/RemoteJoinCodeModal.tsx:38-50`; `internal/daemon/client_remote_files.go:118-121`
**Issue:** `ExchangeJoinCodeAtURL` maps an upstream 404 to the string
`"join code not-found (status 404)"` (client_remote_files.go:119-121). `mapErrorMessage`
checks `lower.includes('not-found')` in the same branch as `'invalid'` and returns
"Code invalid. Double-check the 8-character code (XXXX-XXXX)." But a 404 from the exchange
path corresponds to `ErrCodeNotFound` — a code that was never issued, already exchanged, or
GC'd (api.go:1234-1236). For a single-use code the common real cause is "already used /
stale," not "you typed it wrong." Telling the user to re-check the digits sends them down
the wrong recovery path (re-typing the same dead code) instead of "ask the owner for a fresh
code." The modal's JSDoc (lines 28-32) listing `'not-found'` as a wrong-code synonym is the
source of the conflation.
**Fix:** Fold not-found into the expired-style "ask the owner to generate a new code" copy,
which matches single-use semantics:

```js
if (lower.includes('expired') || lower.includes('not-found')) {
  return 'Code expired or already used. Ask the owner to generate a new code.'
}
if (lower.includes('invalid')) {
  return 'Code invalid. Double-check the 8-character code (XXXX-XXXX).'
}
```

## Info

### IN-01: code-length copy says "8-character" but the exchanger expects a 5-character code

**File:** `frontend/src/components/RemoteJoinCodeModal.tsx:48,133-143,150`
**Issue:** The modal body, placeholder, and error copy all say "8-character join code
(format XXXX-XXXX)". The Go binding's doc and tests describe and exercise a **5-character**
code (`ExchangeJoinCodeAtURL` doc, client_remote_files.go:43, and
`c.ExchangeJoinCodeAtURL(srv.URL, "ABCDE")` throughout client_test.go). "XXXX-XXXX" is 8
glyphs plus a hyphen. The mismatch is pre-existing (Phase 122 copy), but FIX-03 re-uses the
same component for the new open-session intent, so the inconsistent instruction now appears
on a second flow.
**Fix:** Reconcile the user-facing length/format string with the real join-code format
(check `capability/joincode.go` `JoinCodeManager.Issue`), then update both the open-session
and files body text.

### IN-02: duplicate doc comment block on `handleSessionsMeta`

**File:** `internal/webserver/server.go:833-842`
**Issue:** `handleSessionsMeta` carries two stacked doc headers — lines 833-838 and again
839-842 both begin "handleSessionsMeta handles GET /api/sessions/meta." The second block was
added during the RB-03 restore without removing the first; Go doc tooling attaches the whole
run, producing a redundant docstring. Cosmetic.
**Fix:** Collapse to a single comment block describing the cap-free contract.

### IN-03: stale "Daemon Manager panel" instruction in the files-intent modal copy

**File:** `frontend/src/components/RemoteJoinCodeModal.tsx:142-143`
**Issue:** The non-open-session body text tells the viewer the owner "generates it from the
Daemon Manager panel." Per the v4.0 redesign the Sessions/Remote pages were dropped and the
owner share gesture now lives on the Hub card Share button (SessionCard.tsx:542-552). If
"Daemon Manager panel" is no longer a current surface name, this instruction points the user
at a surface that may not exist.
**Fix:** Confirm the current owner-side share location and update the copy to name it.

### IN-04: `RemotePeerSessions`/`RemoteSession` defined in three places, drift-prone

**File:** `frontend/src/lib/remoteSession.ts:12-26`; `frontend/src/wailsjs/go/main/App.d.ts:105-119`; `frontend/src/wailsjs/wailsjs/go/main/App.d.ts:41`
**Issue:** `RemoteSession` and `RemotePeerSessions` are declared independently in
`lib/remoteSession.ts` and in the Wails stub `App.d.ts`. App.tsx imports
`RemotePeerSessions` from the Wails stub (line 40) but the lib helpers (`findRemoteSession`,
`remoteBaseURLFor`) are typed against the lib copy (line 51). Structural typing keeps this
compiling, but the declarations can drift — adding a field to one and not the other would
silently break the helper call sites. The `App.d.ts` header claims AUTO-GENERATED "DO NOT
edit manually," yet it carries hand-written "broadcast join-code fields REMOVED" comments,
indicating it is hand-maintained and thus prone to drift.
**Fix:** Have the Wails stub re-export the lib types (or vice versa) so there is a single
source of truth for the remote-session shape.

---

_Reviewed: 2026-06-22_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
