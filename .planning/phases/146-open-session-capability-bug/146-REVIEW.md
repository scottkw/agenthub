---
phase: 146-open-session-capability-bug
reviewed: 2026-06-22T00:00:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - frontend/src/App.tsx
  - frontend/src/components/Hub/HubPanel.tsx
  - frontend/src/components/Hub/SessionCard.tsx
  - frontend/src/components/Hub/SessionCardGrid.tsx
  - frontend/src/components/__tests__/App.open-remote.test.tsx
  - frontend/src/lib/__tests__/remoteAdapter.test.ts
  - frontend/src/lib/remoteAdapter.ts
  - frontend/src/lib/remoteSession.ts
  - frontend/src/wailsjs/go/main/App.d.ts
  - internal/daemon/api.go
  - internal/daemon/mint_join_codes_test.go
  - internal/tailnet/sessions.go
  - internal/webserver/server.go
  - internal/webserver/sessions_meta_embed_test.go
  - internal/webserver/sessions_meta_test.go
findings:
  critical: 2
  warning: 3
  info: 2
  total: 7
status: issues_found
---

# Phase 146: Code Review Report

**Reviewed:** 2026-06-22
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

FIX-03 (#98) adds a per-session join-code minting path on the owner side
(`mintSessionJoinCodes` + `/api/sessions/meta` embed) and a viewer-side
exchange-then-open flow (`handleOpenRemoteSession`). Each individual layer is
clean and the unit tests pass — but the layers are **not connected end-to-end**,
and the security model that justifies handing out a read-write code is not
enforced where the threat model claims it is.

Two BLOCKERs:

1. **The feature is non-functional.** The join codes are produced by the owner
   webserver, fetched into `tailnet.ShareableSessionMeta`, then **dropped** at the
   `app.go::GetRemoteSessionsWithMeta` Go→frontend conversion (the Wails
   `RemoteSession` struct has no join-code fields and `app.go` was never modified
   this phase). `session.roJoinCode` is therefore always `undefined` in the
   frontend, so `handleOpenRemoteSession` always takes the D-03 "not shared"
   branch. "Open in browser" stays broken — the exact bug #98 set out to fix.
   Every unit test passes because no test crosses the `app.go` boundary.

2. **RW capability is broadcast to all tailnet peers.** `mintSessionJoinCodes`
   unconditionally mints BOTH a read-only and a read-write join code for every
   web-enabled session and embeds both in the open, no-cap `/api/sessions/meta`
   response. The `isPeerSelf` "only the owner gets RW" mitigation is purely
   client-side and advisory; the server's `/join/exchange` performs no identity
   check. Any tailnet peer that reads the raw discovery JSON obtains a working
   write/input token. The RESEARCH threat-model row claiming RW is gated by
   `isPeerSelf` does not match the implementation.

Plus a dead D-06 owner-RW path, a non-load-bearing `omitempty` inconsistency,
and minor doc/contract drift.

Note: `app.go` is not in the review file set but is the load-bearing integration
point; finding CR-01 is provable entirely from in-scope files (the frontend
consumes `roJoinCode`, the tailnet layer emits `ROJoinCode`, and the wiring
between them is verifiably absent).

## Critical Issues

### CR-01: Join codes never reach the frontend — "Open in browser" is dead on arrival

**File:** `frontend/src/App.tsx:1085` (consumer); root cause in `app.go:1207-1215` + struct `app.go:57-63` (out-of-scope but verified)
**Issue:**
The owner mints codes (`internal/daemon/api.go:1202` `mintSessionJoinCodes`),
the webserver embeds them (`internal/webserver/server.go:883-889`), and the
viewer's daemon decodes them into `tailnet.ShareableSessionMeta`
(`internal/tailnet/sessions.go:128-129`). But `GetRemoteSessionsWithMeta`
(`app.go:1207-1215`) constructs the Wails-exposed `RemoteSession` with only
`ID/Name/CLIType/Status/URL` — it never copies `s.ROJoinCode` / `s.RWJoinCode`,
and the Go `RemoteSession` struct (`app.go:57-63`) has no slot for them. `app.go`
was not touched in this phase (`git diff --name-only … | grep app.go` is empty).

Consequence: `adaptRemoteSession` reads `session.roJoinCode` / `session.rwJoinCode`
(`frontend/src/lib/remoteAdapter.ts:37-38`) from a value that is always
`undefined`. In `handleOpenRemoteSession` (`App.tsx:1085`):
```ts
if (!session.roJoinCode) {
  setSaveBanner({ kind: 'error', text: 'This session is not shared …' })
  return
}
```
This branch is taken for every remote session. The feature is 100% broken
end-to-end despite green unit tests (no test spans the `app.go` conversion).

**Fix:** Add the fields to the Wails struct and copy them in the conversion:
```go
// app.go
type RemoteSession struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    CLIType    string `json:"cliType"`
    Status     string `json:"status"`
    URL        string `json:"url"`
    ROJoinCode string `json:"roJoinCode,omitempty"`
    RWJoinCode string `json:"rwJoinCode,omitempty"`
}

// in GetRemoteSessionsWithMeta:
sessions = append(sessions, RemoteSession{
    ID: s.ID, Name: s.Name, CLIType: s.CLIType, Status: s.Status, URL: s.URL,
    ROJoinCode: s.ROJoinCode,
    RWJoinCode: s.RWJoinCode,
})
```
Add an integration test that drives `GetRemoteSessionsWithMeta` against a stub
peer serving `/api/sessions/meta` with codes and asserts the codes survive to the
returned `RemoteSession` — this boundary currently has zero coverage.

### CR-02: Read-write join code is handed to every tailnet peer; `isPeerSelf` gate is client-side only

**File:** `internal/daemon/api.go:1202-1262` (`mintSessionJoinCodes`), `internal/webserver/server.go:883-890` (embed), `frontend/src/App.tsx:1095-1097` (advisory gate)
**Issue:**
`mintSessionJoinCodes` always mints both an RO (`read`) and an RW
(`read,write`) code and registers both grants. `handleSessionsMeta`
(`server.go:883-889`) embeds **both** in the open, no-capability
`/api/sessions/meta` response, which is served to any caller that can reach the
tailnet bind IP. The only thing standing between a non-owner and a write-capable
token is this client-side line:
```ts
const code = session.rwJoinCode && isPeerSelf(session.hostname, tailscaleHealth)
  ? session.rwJoinCode
  : session.roJoinCode
```
`/join/exchange` (`internal/webserver/server.go:767`) and
`handleExchangeJoinCode` (`api.go:1303`) perform no identity check — they consume
whatever code is presented and return the corresponding token. A peer that simply
`curl`s `/api/sessions/meta` reads `rw_join_code` directly and exchanges it for a
write/input token, fully bypassing `isPeerSelf`. The RESEARCH threat-model row
("RO-to-RW escalation … D-05: rwJoinCode is used ONLY when isPeerSelf confirms
viewer is owner") describes a guarantee the code does not provide. Pre-146,
`/api/sessions/meta` contained no codes at all, so RW required an explicit
out-of-band share gesture; FIX-03 now auto-broadcasts RW to the whole tailnet.

**Fix:** Do not embed the RW code in the broadcast discovery payload. Options:
- Embed only `ro_join_code` in `/api/sessions/meta`; obtain RW through a path that
  actually authenticates the owner (e.g. the local Unix-socket
  `issueCapabilitiesForSession`, which the owner already reaches for re-attach),
  or
- Keep both fields only if the owner-identity check moves server-side into the
  exchange/mint path (verify the caller's tailnet identity matches the session
  owner before issuing an RW token).

At minimum, gate `mintSessionJoinCodes` so `rwCode` is empty unless the request
is provably from the session owner, and stop relying on `isPeerSelf` (a UI hint)
as the security boundary.

## Warnings

### WR-01: D-06 owner-RW selection is effectively dead code

**File:** `frontend/src/App.tsx:1068-1076` (`isPeerSelf`), `1095-1097` (selection)
**Issue:**
Sessions in `handleOpenRemoteSession` come exclusively from `remotePeers`
(adapted via `adaptAllRemoteSessions`), so `session.hostname` is always a
*remote* peer's hostname. `isPeerSelf` compares it to the viewer's OWN short
hostname (`tailscaleHealth.domain.split('.')[0]`). For the #98 scenario (one user,
two distinct Macs), the remote hostname never equals the local hostname, so
`isPeerSelf` always returns `false` and the RW branch is unreachable — owner
re-attach always gets RO, violating D-06's intent. (The RESEARCH §Open Questions
A2/UQ1 flagged exactly this format-mismatch risk; it was not resolved.) Combined
with CR-02 this is a wash for security, but it means the stated D-06 behavior is
not delivered and the `rwJoinCode`/`isPeerSelf` machinery is inert.
**Fix:** Either remove the dead RW selection path (and the `rwJoinCode` plumbing
that exists only to feed it) or derive ownership from a real signal — e.g. an
explicit `is_owner` field computed server-side, since hostname comparison cannot
distinguish "my other Mac" from "someone else's Mac" on a shared tailnet.

### WR-02: `omitempty` mismatch between the two meta structs masks degraded mode

**File:** `internal/webserver/server.go:59-60` vs `internal/tailnet/sessions.go:128-129`
**Issue:**
The producer (`sessionMetaItem`) serializes `ro_join_code`/`rw_join_code` with
NO `omitempty` (always present, empty string in degraded/nil-issuer mode), while
the consumer mirror (`ShareableSessionMeta`) uses `omitempty`. The comment at
`server.go:51-52` says fields are "Always serialized (no omitempty) so downstream
tests can assert their presence" — but `TestSessionsMeta_NilIssuer`
(`sessions_meta_test.go:117-122`) explicitly tolerates the field being absent OR
empty, so the no-`omitempty` choice buys nothing and the two structs disagree on
the wire contract. Not currently a runtime bug (empty string decodes fine), but
it is a latent inconsistency: a reader that distinguishes "field absent" from
"field empty" would behave differently against the two structs.
**Fix:** Make both consistent — use `omitempty` on both, or neither, and update
the `server.go:51-52` comment to match the actual test expectation.

### WR-03: No coverage for the exchange-failure → banner classification against real error strings

**File:** `frontend/src/App.tsx:1102-1107`; error source `internal/daemon/client_remote_files.go:145-147`
**Issue:**
`handleOpenRemoteSession` classifies failures via
`msg.includes('expired') || msg.includes('session-gone')`. This happens to match
the real strings the daemon produces (`"join exchange: expired"` /
`"join exchange: session-gone"` from the `/join?error=<kind>` 303 path), but the
match is by coincidence of substrings across a Go↔TS boundary with no contract
test. `App.open-remote.test.tsx` is source-inspection only (greps the file text);
it asserts the *literal* `'expired'`/`'session-gone'` appear in the source, not
that they line up with the daemon's emitted strings. A future reword on either
side (e.g. Go switching to the dead 410/404 status branch that emits
`"…not-found (status 404)"`) silently degrades every non-expiry failure to the
generic "Failed to open session" banner.
**Fix:** Centralize the error-kind tokens in one shared constant/contract, or add
a behavioral test that exercises `ExchangeJoinCodeAtURL`'s actual error output
through `handleOpenRemoteSession`'s classifier.

## Info

### IN-01: Cap token now transits the React layer, contradicting the documented contract

**File:** `frontend/src/App.tsx:1099-1100`; contract at `app.go:1232-1233`
**Issue:**
`ExchangeJoinCodeAtURL`'s docstring states the returned cap "must NEVER be passed
to the React frontend (it lives in the daemon's RemoteCapStore)." The new
`handleOpenRemoteSession` receives the token in JS (`const token = await
ExchangeJoinCodeAtURL(...)`) and interpolates it into a `BrowserOpenURL` URL. It
is not persisted in React state and a cap-in-URL is inherent to the open-in-
browser design, but the contract comment is now stale and the token does transit
the JS heap + the system browser address bar/history.
**Fix:** Update the `app.go:1232-1233` docstring to carve out the open-in-browser
case, and note the address-bar exposure is accepted (matches the QR/share flow).

### IN-02: Stale source-line reference in comment

**File:** `frontend/src/components/Hub/HubPanel.tsx:510`
**Issue:**
Comment cites "the tab grid guard (App.tsx:1535)" but the relevant App.tsx region
referenced by this phase ends around line 1429; line numbers drift and bare
line-number cross-references rot quickly.
**Fix:** Reference the symbol (e.g. "mirrors the HubModal relayPort>0 guard")
rather than an absolute line number.

---

_Reviewed: 2026-06-22_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
