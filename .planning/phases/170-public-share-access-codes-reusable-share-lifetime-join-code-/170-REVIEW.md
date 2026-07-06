---
phase: 170-public-share-access-codes-reusable-share-lifetime-join-code-
reviewed: 2026-07-06T01:45:41Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - frontend/src/components/Hub/SessionShareModal.tsx
  - frontend/src/components/SessionSharePanel.tsx
  - frontend/src/components/__tests__/SessionSharePanel.test.tsx
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/wailsjs/go/models.ts
  - internal/capability/joincode.go
  - internal/capability/joincode_test.go
  - internal/daemon/api.go
  - internal/daemon/api_test.go
  - internal/daemon/funnel_test.go
  - internal/daemon/types.go
  - internal/webserver/join_test.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 170: Code Review Report

**Reviewed:** 2026-07-06T01:45:41Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Phase 170 adds reusable, share-lifetime public join codes (FNL-08) on top of the
single-use join-code machinery. The `JoinCodeManager` reusable/TTL/revoke work
(`joincode.go`) is clean, correct, and well-tested: the mutex-under-Exchange
TOCTOU argument holds, single-use vs reusable is a single conditional delete, and
expiry deletes for both classes. The webserver-boundary reusable contract
(`join_test.go`) and the funnel teardown revocation suite (`funnel_test.go`) are
thorough.

The defects live at the **integration boundary between the reusable public code
and the D-15 grant-revocation system**. The reusable code is deliberately pinned
to the *first* read token and never rotated (T-170-04), but the token it points
to depends on a grant that a normal owner gesture — toggling "Enable remote file
browsing" — wipes via `ws.ClearGrants`. The result is a silent, unrecoverable
lockout of every existing public viewer. Two further issues concern the code
outliving or under-living its share: a TOCTOU window that can orphan a code past
teardown, and the 8h backstop TTL killing the code with no re-mint path for
"Until I disable" shares.

No `<structural_findings>` block was provided, so this report is entirely
narrative findings.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Toggling "remote file browsing" permanently breaks the live reusable public code

**File:** `internal/daemon/api.go:1466-1487` (mint/cache), `internal/daemon/api.go:1773-1782` (`handleSetSessionBrowse` → `ClearGrants`), `frontend/src/components/Hub/SessionShareModal.tsx:264-288` (`handleBrowseToggle` re-issue)

**Issue:**
The reusable public read code is minted once from the read token `rTok` of the
*first* `issueCapabilitiesForSession` call for a Funnel session and cached in
`a.funnelReadCode[sessionID]`. By design (T-170-04) it is never rotated on
subsequent calls — every later call returns the cached code, which still resolves
(via `Exchange`) to that original `rTok` and therefore that original grant ID
(`G1`).

The browse toggle destroys the grant that code depends on:

1. Funnel enabled, caps issued → public code `P` minted, bound to `rTok_1`
   (grant `G1`); `ws.AddGrant(sid, G1)`. Owner shares `P` out-of-band.
2. Owner flips "Enable remote file browsing". Frontend `handleBrowseToggle` →
   `SetSessionBrowse` → daemon `handleSetSessionBrowse` calls
   `ws.ClearGrants(id)` (api.go:1781), deleting the entire grant set for the
   session — including `G1`.
3. `handleBrowseToggle` then re-issues caps → `issueCapabilitiesForSession`
   mints a *new* `rTok_2` with a *new* grant `G2` and `AddGrant(G2)`, but the
   cached public code `P` is **not** rotated (still bound to `rTok_1`/`G1`), and
   because `a.funnelReadCode[sid]` is already set there is **no re-mint path**.
4. Any public viewer now redeeming `P` is sent to `/sessions/{sid}?cap=rTok_1`.
   `requireCapability` (`internal/webserver/capability_mw.go:61`) calls
   `isGrantActive(sid, G1)` → false → **403 "capability has been revoked."**

The reusable code is silently and permanently dead for every holder, which is
the exact failure mode T-170-04 exists to prevent ("a code already handed to a
viewer must not silently break"). The frontend compounds it: `handleBrowseToggle`
refreshes `cachedShare` but never updates `publicReadCode` state, so the UI keeps
displaying the now-broken code with no error. Recovery requires a full Funnel
off→on cycle.

This only bites the browse path because it is the only re-issue trigger that
calls `ClearGrants`; a plain re-issue (modal reopen, warm-up) accumulates grants
and leaves `G1` intact. That is why the existing idempotency test
(`funnel_test.go:388` `TestIssueCapabilities_FunnelPublicCode_Idempotent`, and the
`browse_on` case of `TestFunnelPublicCode_ReadOnlyScope`, which sets browse
*before* the first issue) does not catch it — no test toggles browse *after* the
public code is minted.

**Fix:** The reusable public code must survive a grant clear. Options, in
preference order:

1. Preserve the public code's grant across `ClearGrants` — track the public
   code's grant ID separately and re-`AddGrant` it after the clear (or have
   `handleSetSessionBrowse` re-add it) so `rTok_1`/`G1` stays active for the
   code's lifetime.
2. Re-bind the cached code to the new token without changing the code string.
   The `JoinCodeManager` map is keyed by code, so a `Rebind(code, newToken)`
   method can swap the stored token while keeping the string stable; call it from
   `issueCapabilitiesForSession` whenever a Funnel public code already exists and
   a fresh `rTok` was just minted.

Add a regression test that mints the public code, toggles browse **after** mint,
then drives the real `/join/exchange` → `/sessions/{id}?cap=...` path and asserts
the request is NOT 403 (grant still active).

## Warnings

### WR-01: TOCTOU — a public code can be minted after Funnel teardown and outlive the share

**File:** `internal/daemon/api.go:1436-1487`

**Issue:**
`issueCapabilitiesForSession` reads `isFunnelSession` under one `a.mu.RLock`
(lines 1437-1439), releases it, then re-acquires `a.mu.Lock` for the mint block
(lines 1466-1485). The doc comment claims the lock is held "across `IssueReusable`
... to close the TOCTOU window," but the funnel-membership check and the mint are
under **separate** lock acquisitions. Interleaving:

1. Thread A reads `isFunnelSession = true`, releases RLock.
2. Thread B (`disableFunnelForSession`, e.g. expiry timer or user disable) takes
   the lock, revokes any cached code, deletes `funnelReadCode[sid]`,
   `funnelSessions[sid]`, unlocks. Share is torn down.
3. Thread A takes the lock, sees `funnelReadCode[sid]` absent, mints a **new**
   reusable code (TTL up to 8h), stores it in `funnelReadCode[sid]`, unlocks.

The session is now Funnel-disabled but holds a live reusable public code that no
future teardown will revoke (teardown already ran). Because
`disableFunnelForSession` does not `ClearGrants`, the token behind that code stays
valid too — so a code that was supposed to "die with the share" resolves for up to
8h after teardown. This defeats the T-170-02 "dies with the share" invariant.

**Fix:** Perform the funnel-membership check and the mint under a single
uninterrupted `a.mu.Lock`. Re-read `a.funnelSessions[sessionID]` inside the mint
critical section and only mint when it is still `true`; otherwise return
`publicReadCode == ""`.

### WR-02: "Until I disable" shares lose the reusable code at the 8h backstop with no re-mint

**File:** `internal/daemon/api.go:1466-1484` (mint gate), `internal/daemon/api.go:108` + `api.go:1687-1693` (TTL selection)

**Issue:**
When `ExpiresIn == 0` ("Until I disable"), no funnel expiry timer is armed
(api.go:1696 gates on `req.ExpiresIn > 0`), but the reusable code TTL is set to
`funnelReadCodeMaxTTL` (8h). After 8h the code expires and `Exchange` deletes it —
yet the share (and its cap-bearing public URL) stays up indefinitely. Crucially,
`a.funnelReadCode[sid]` still holds the now-dead code string, so the mint gate
`cached, ok := a.funnelReadCode[sessionID]` sees `ok == true` and
`issueCapabilitiesForSession` **never re-mints**. The flagship long-lived public
share therefore loses its reusable join code permanently at the 8h mark, returning
the dead string to the UI (viewers get 410-then-404) with no recovery short of a
Funnel off→on cycle.

**Fix:** Treat an expired cached code as absent at the mint gate — track its expiry
(or probe resolvability) and re-mint into `a.funnelReadCode[sid]` if it has lapsed.
Alternatively, for `ExpiresIn == 0`, arm a refresh that rotates the code before the
8h backstop. Add a clock-injected test covering an `ExpiresIn==0` session past 8h
asserting a fresh, resolvable code is returned.

### WR-03: Redundant double cap-issuance re-adds grants on every Funnel modal open

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:207-227` (seeding effect) and `:374-396` (warm-up completion effect)

**Issue:**
For a session already `funnelActive` when the modal opens, both the server-truth
seeding effect (fires because `shareEnabled` is true) and the warm-up completion
effect (fires because `session.funnelActive && !funnelUrl`) call
`IssueCapabilities(session.id)` on the same render pass. Each runs
`issueCapabilitiesForSession`, which mints two fresh tokens and `AddGrant`s two new
grant IDs server-side. The grant set therefore grows by two on every modal open
(and again on every browse toggle), and only `ClearGrants`/session-end prunes it —
a slow unbounded accumulation of stale grants in `ws.grants[sid]`, plus a redundant
round-trip. Not a correctness bug on its own, but it widens the CR-01/WR-01 surface
(more live tokens per session than the UI reflects).

**Fix:** Deduplicate issuance — gate the seeding effect out when the session is
Funnel-active (let the warm-up effect own issuance for Funnel sessions), or share a
single in-flight issuance promise between the two effects.

## Info

### IN-01: Doc comment overstates the TOCTOU guarantee

**File:** `internal/daemon/api.go:1457-1465`

**Issue:** The mint-block comment states "a.mu is held across IssueReusable ... to
close the TOCTOU window where two concurrent callers could each mint a distinct
code." It closes the *two-concurrent-minters* window, but not the
*funnel-disabled-mid-issue* window (see WR-01). The comment reads as a broader
guarantee than the code provides.

**Fix:** Narrow the comment to the concurrency it actually covers, or implement the
single-critical-section fix in WR-01 so the comment becomes true.

### IN-02: `IssueReusable` duplicates the code-generation body of `Issue`

**File:** `internal/capability/joincode.go:66-105`

**Issue:** `Issue` and `IssueReusable` share the identical 8-char base32
code-generation body (`rand.Read` → `EncodeToString` → dashed split); only the
stored `joinEntry` differs. The duplication is small and each is individually
correct, but two copies of the entropy/format logic can drift.

**Fix:** Extract a private `newCode() (string, error)` helper and have both call
it, keeping the entry construction in each caller.

---

_Reviewed: 2026-07-06T01:45:41Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
