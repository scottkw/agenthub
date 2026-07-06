---
phase: 170-public-share-access-codes-reusable-share-lifetime-join-code-
reviewed: 2026-07-05T00:00:00Z
depth: standard
files_reviewed: 11
files_reviewed_list:
  - internal/capability/joincode.go
  - internal/capability/joincode_test.go
  - internal/daemon/api.go
  - internal/daemon/types.go
  - internal/daemon/api_test.go
  - internal/daemon/funnel_test.go
  - internal/webserver/join_test.go
  - frontend/src/components/SessionSharePanel.tsx
  - frontend/src/components/Hub/SessionShareModal.tsx
  - frontend/src/components/__tests__/SessionSharePanel.test.tsx
  - frontend/src/wailsjs/go/main/App.d.ts
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 170: Code Review Report

**Reviewed:** 2026-07-05
**Depth:** standard
**Files Reviewed:** 11
**Status:** issues_found

## Summary

FNL-08 adds a reusable, read-only, share-lifetime public join code for Tailscale-Funnel (internet-exposed) shares. The core security invariants the phase set out to hold — `rTok`-only binding (never `wTok`), single teardown chokepoint through `disableFunnelForSession`, TTL capped at `min(ExpiresIn, 8h)`, mutex discipline, no code leakage into logs — are correctly implemented at the mint site (`issueCapabilitiesForSession`) and teardown site, and are well covered by `funnel_test.go` and `join_test.go`. `IssueReusable`/`Revoke`/`Exchange` in `joincode.go` are sound: the single-mutex lookup+expiry+conditional-delete is TOCTOU-safe, and the reusable exemption is correctly scoped.

However, the idempotent-caching requirement (T-170-04, "a code already handed to a viewer must not silently rotate") collides with the existing grant-clearing behavior of the browse toggle. Toggling **remote file browsing** on a live Funnel share calls `ws.ClearGrants(id)`, which permanently invalidates the exact grant the cached public code (and public URL) depend on — while the idempotent cache refuses to re-mint. The result is a public share that silently dies mid-session with no UI recovery. That is the blocker below. Two supporting security/robustness warnings (a mint-after-teardown leak window, and `files.read` reaching anonymous internet viewers) and two info items round out the review.

No secret or join code is written to any log statement in the reviewed code — that requirement is satisfied.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Toggling remote file browsing permanently breaks the public read code and public URL of a live Funnel share

**File:** `internal/daemon/api.go:1766` (`handleSetSessionBrowse`) in conjunction with `internal/daemon/api.go:1457-1487` (`issueCapabilitiesForSession` idempotent cache)

**Issue:**
The public read code is minted once and cached idempotently, bound to the *first* read token `rTok1` (grant id `rgid1`), which was registered via `ws.AddGrant(sessionID, rgid1)`. The public URL shown in the UI (`funnelUrl`) also embeds `rTok1`.

When the owner toggles "Enable remote file browsing" on a session that is already Funnel-shared, `handleSetSessionBrowse` runs:

```go
a.engine.SetSessionBrowse(id, req.Enabled)
...
if ws != nil {
    ws.ClearGrants(id) // clears ALL grants for the session, including rgid1
}
```

`ClearGrants` permanently invalidates every previously-issued capability for the session — this is proven by `TestHandleWebServe_ToggleOffClearsGrants` (api_test.go:944), where a live HTTPS `probeGrant` returns `false` after `ClearGrants`. The frontend then re-issues caps (`handleBrowseToggle`, SessionShareModal.tsx:264), minting a fresh `rTok2`/`rgid2` — but:

1. Daemon side: the idempotent cache (`api.go:1466-1487`) sees `funnelReadCode[sessionID]` already set and returns the **same** `code1 -> rTok1`. `rgid1` is now cleared, so `Exchange(code1)` yields `rTok1`, which the webserver's grant check rejects. The distributed public join code is dead.
2. Frontend side: `handleBrowseToggle` updates `cachedShare` but never updates `funnelUrl` or `publicReadCode` (SessionShareModal.tsx:270-279). The warm-up effect that sets them is gated `if (funnelUrl) return` (SessionShareModal.tsx:375-376), so it never re-fires. The displayed public URL still embeds the now-dead `rTok1`.

Net effect: a normal, reachable owner action (enable/disable file browsing) silently and permanently breaks a public share link that may already be in the hands of remote viewers, with no error and no UI recovery path. This defeats the central deliverable of the phase.

**Fix:**
Treat the browse toggle as a rotation point for the public code on the daemon side — revoke and drop the cached entry so the next `issueCapabilitiesForSession` re-mints a fresh reusable code bound to the new `rTok`, then have the frontend adopt the re-issued values. In `handleSetSessionBrowse`, before/after `ClearGrants`:

```go
a.mu.Lock()
if code, ok := a.funnelReadCode[id]; ok {
    a.joinCodes.Revoke(code)
    delete(a.funnelReadCode, id)
    // keep funnelReadCodeTTL[id] so the re-mint reuses the same capped TTL
}
a.mu.Unlock()
```

and in `SessionShareModal.handleBrowseToggle`, after the re-issue, also refresh the Funnel-scoped state from the same response:

```ts
setFunnelUrl(resp.readUrl)
setPublicReadCode(resp.publicReadCode ?? null)
```

(Add a regression test: enable Funnel, mint public code, toggle browse on, assert `Exchange(publicReadCode)` still resolves to a read-only token whose grant is live — the current suite only asserts idempotency and revoke-on-teardown, never the browse-toggle interaction.)

## Warnings

### WR-01: Public read code can be minted *after* teardown already ran, leaking a live public code for up to 8h

**File:** `internal/daemon/api.go:1437-1487` (`issueCapabilitiesForSession`)

**Issue:**
`isFunnelSession` is read under `a.mu.RLock` at line 1437-1439 and the lock is released. The mint block re-acquires `a.mu.Lock` at line 1467. If `disableFunnelForSession` runs in that gap (auto-expiry timer fire, toggle-off, web-share-off, session exit — all call it), it revokes the code and deletes `funnelReadCode[sessionID]`, `funnelReadCodeTTL[sessionID]`, and `funnelSessions[sessionID]`. The mint block then observes `funnelReadCode[sessionID]` absent, defaults `ttl` to `funnelReadCodeMaxTTL` (8h, since `funnelReadCodeTTL` was also deleted), mints a fresh reusable code, and stores it into `funnelReadCode[sessionID]` — *after* the single teardown chokepoint has already run. That code is now orphaned: `funnelSessions` no longer marks the session as Funnel, but a live, internet-resolvable reusable public code persists until its 8h TTL. This violates the "a manually-disabled share must leave no live public entry point" contract in the `disableFunnelForSession` doc comment.

**Fix:**
Re-check funnel liveness *inside* the mint lock before minting, so a teardown that won the race wins definitively:

```go
if isFunnelSession {
    a.mu.Lock()
    if !a.funnelSessions[sessionID] { // teardown raced us — do not mint
        a.mu.Unlock()
    } else {
        cached, ok := a.funnelReadCode[sessionID]
        if !ok { /* mint + store as today */ }
        a.mu.Unlock()
        publicReadCode = cached
    }
}
```

### WR-02: Reusable public read code grants `files.read` to anonymous internet viewers when browse is ON at mint time

**File:** `internal/daemon/api.go:1409-1415`, `1466-1487`

**Issue:**
The public code is bound to `rTok`, whose perms are `read` (browse OFF) or `read,files.read` (browse ON, api.go:1411-1414). When browse is ON at first mint, the reusable public code — reachable by any anonymous viewer on the public internet who has the ~40-bit code — resolves to a token carrying `files.read`, i.e. read access to the session's working-directory sandbox. The phase's own scope test `TestFunnelPublicCode_ReadOnlyScope` (funnel_test.go:339) only asserts the token does **not** contain `write`/`files.write`; it explicitly tolerates `files.read` in both browse states. So "read-only" here means "no write," not "no filesystem exposure." For a tailnet share this matches D-05; for a *public internet* share it is a materially larger blast radius (directory contents exposed to the world, not just terminal output) and deserves an explicit decision rather than falling out of the browse toggle by default.

**Fix:**
Decide deliberately whether public Funnel codes may ever carry `files.read`. Safest option: strip `files.read` from the token used for the *public* reusable code regardless of the per-session browse toggle (mint the public code from a `read`-only claim set, independent of `browseEnabledFor`), and add a positive test asserting the public code's perms are exactly `read` even when browse is ON. If the exposure is intended, document it in the risk panel copy so the owner is warned that enabling browse also exposes files to the public link.

### WR-03: `handleBrowseToggle` refreshes `cachedShare` but not `funnelUrl`/`publicReadCode`, leaving the Internet section showing stale values

**File:** `frontend/src/components/Hub/SessionShareModal.tsx:264-288`

**Issue:**
Independent of the daemon grant issue in CR-01, the browse-toggle handler re-issues capabilities and updates only `cachedShare` (readURL/writeURL/readCode/writeCode). The Funnel-scoped `funnelUrl` and `publicReadCode` state are never updated here, and the warm-up effect that would refresh them is gated on `!funnelUrl` (line 375-376) so it will not re-run. The Internet (public) section therefore continues to render the pre-toggle URL and code even though the daemon has re-minted the underlying tokens. This is the frontend half of the CR-01 defect and needs its own fix even after the daemon side is corrected.

**Fix:**
In `handleBrowseToggle`, after a successful `IssueCapabilities`, also propagate the Funnel-scoped fields when the session is Funnel-active:

```ts
setCachedShare({ readURL: resp.readUrl, ... })
if (session.funnelActive) {
  setFunnelUrl(resp.readUrl)
  setPublicReadCode(resp.publicReadCode ?? null)
}
```

## Info

### IN-01: Re-enabling Funnel with a new `expiresIn` does not extend the already-minted public code's TTL

**File:** `internal/daemon/api.go:1671-1712` (`handleSetSessionFunnel` enable path)

**Issue:**
On a re-enable, `funnelReadCodeTTL[id]` is recomputed and stored (line 1684-1693), but the public code is minted lazily and cached idempotently, so `funnelReadCodeTTL` is only consumed on the *first* mint. A re-enable with a longer `expiresIn` updates the stored TTL but has no effect on the existing code, whose per-code TTL was fixed at first mint. This is a benign edge (the code still lives at least as long as its original TTL, and the Funnel session's own expiry timer is refreshed), but the stored-TTL update is effectively dead on the re-enable path and could mislead a future maintainer.

**Fix:** Either document that `funnelReadCodeTTL` is authoritative only at first mint, or on re-enable revoke+drop `funnelReadCode[id]` (as CR-01's fix would) so the next issuance re-mints with the updated TTL.

### IN-02: No rate limiting on the public `/join/exchange` endpoint; ~40-bit entropy + 8h window is the sole brute-force barrier

**File:** `internal/capability/joincode.go:81-105` (`IssueReusable`), consumed publicly via the webserver join handler (`internal/webserver/join_test.go`)

**Issue:**
The reusable code carries ~40 bits of entropy and is resolvable by unauthenticated callers from the public internet for up to 8h. The 8h `funnelReadCodeMaxTTL` cap keeps the expected-brute-force cost impractical for a home-bandwidth attacker, so this is not a blocker — but there is no visible per-IP/global rate limit on the public exchange endpoint, meaning the entropy and the TTL cap are the only defenses. Worth noting for the threat model.

**Fix:** Consider a modest rate limit (or exponential backoff) on the public `/join/exchange` handler as defense-in-depth for the internet-reachable reusable-code path.

---

_Reviewed: 2026-07-05_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
