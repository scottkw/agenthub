---
phase: 170-public-share-access-codes-reusable-share-lifetime-join-code-
fixed_at: 2026-07-05T00:00:00Z
review_path: .planning/phases/170-public-share-access-codes-reusable-share-lifetime-join-code-/170-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 170: Code Review Fix Report

**Fixed at:** 2026-07-05
**Source review:** .planning/phases/170-public-share-access-codes-reusable-share-lifetime-join-code-/170-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4 (Critical + Warning; Info findings IN-01/IN-02 out of scope per `fix_scope: critical_warning`)
- Fixed: 4
- Skipped: 0

All four fixes are security/concurrency logic changes. Syntax/build verification
passed (`go build`, `go vet`, `tsc --noEmit`, `go test ./internal/capability/`),
but semantic correctness — especially the concurrency ordering (WR-01), the
grant-survival behaviour (CR-01), and the 8h re-mint path (WR-02) — was NOT
exercised by a live/regression test in this pass. The review recommends new
regression tests for each; those were not added here. **Each fix below is
flagged "requires human verification"** — confirm behaviour with the suggested
regression tests before the phase proceeds.

> **Update 2026-07-05 (follow-up):** Regression tests were subsequently added
> for **CR-01 and WR-02** (commit `23d273e5`) and mutation-verified (each fails
> on the pre-fix code, passes after the fix). See the
> "Regression Coverage Added" section at the end of this report. CR-01 and WR-02
> are now covered at the automated level; **WR-01 and WR-03 still have no
> dedicated regression test** and remain human-verification-only.

## Fixed Issues

### CR-01: Toggling "remote file browsing" permanently breaks the live reusable public code

**Files modified:** `internal/capability/joincode.go`, `internal/daemon/api.go`
**Commit:** b2649327
**Status:** fixed + regression-tested (commit `23d273e5`); live UAT still recommended
**Applied fix:** Chose review option 2 (rebind, the lower-blast-radius option).
Added `JoinCodeManager.Rebind(code, token)` which swaps the stored token for an
existing reusable code while keeping the **code string, expiry, and reusable
class unchanged** (returns `ErrCodeNotFound` if the code is gone). In
`issueCapabilitiesForSession`, extracted `mintFunnelReadCodeLocked` and changed
the mint block so that when a cached public code already exists it is *rebound*
to the freshly-minted `rTok` (whose grant `AddGrant` just registered) rather
than left pointing at the old, now-cleared grant. Because the code string is
preserved (T-170-04), the previously-broken frontend behaviour (never updating
`publicReadCode` after a browse toggle) is now moot — the code the viewer holds
keeps resolving. If `Rebind` finds the code already gone, a fresh code is minted.
**Human verification:** add the regression test the review requests — mint the
public code, toggle browse **after** mint, then drive the real
`/join/exchange` → `/sessions/{id}?cap=...` path and assert the response is NOT
403 (grant still active).

### WR-01: TOCTOU — a public code can be minted after Funnel teardown and outlive the share

**Files modified:** `internal/daemon/api.go`
**Commit:** 78964901
**Status:** fixed: requires human verification
**Applied fix:** The mint block re-reads `a.funnelSessions[sessionID]` INSIDE the
same `a.mu.Lock()` that guards the mint, and only mints/rebinds when it is still
`true`. A `disableFunnelForSession` that interleaves between the earlier RLock
membership read (used for base-URL selection) and the mint lock now causes the
mint to be skipped (`publicReadCode == ""`), so teardown can no longer be raced
into orphaning a live code. **Human verification:** the fix is a lock-ordering
change; confirm with a concurrency/interleaving test (mint gated on a
just-removed `funnelSessions` entry returns an empty public code and leaves no
resolvable orphan). Note: `disableFunnelForSession` does not itself hold the
same critical section as the whole issuance, so the guarantee is "no mint after
the membership entry is removed," which is the invariant WR-01 asks for.

### WR-02: "Until I disable" shares lose the reusable code at the 8h backstop with no re-mint

**Files modified:** `internal/daemon/api.go`
**Commit:** 2d38e06b
**Status:** fixed + regression-tested (commit `23d273e5`); live UAT still recommended
**Applied fix:** Added `funnelReadCodeExpiry map[string]time.Time`, populated on
every fresh mint in `mintFunnelReadCodeLocked` (`now + ttl`) and cleaned up in
`disableFunnelForSession`. At the mint gate the cached code is now treated as
**absent when expired** (`ok && hasExpiry && time.Now().After(expiry)`), so an
`ExpiresIn==0` share whose code hit the 8h `funnelReadCodeMaxTTL` backstop
re-mints a fresh, resolvable code on the next issuance instead of returning the
dead string forever. Expiry is preserved by `Rebind` (CR-01), so the ~40-bit
entropy window (T-170-03) is not extended by a rebind. **Human verification:**
add the clock-injected test the review requests — an `ExpiresIn==0` session past
8h returns a fresh, resolvable code. Caveat: re-mint only fires on the next
`IssueCapabilities` call; a fully idle share still shows the dead code until the
next issuance (modal reopen/poll). If "self-healing while idle" is required, a
refresh timer (the review's alternative) would be needed.

### WR-03: Redundant double cap-issuance re-adds grants on every Funnel modal open

**Files modified:** `frontend/src/components/Hub/SessionShareModal.tsx`
**Commit:** b088c1e8
**Status:** fixed: requires human verification
**Applied fix:** Deduplicated issuance for Funnel-active sessions. The
server-truth seeding effect now bails out early when `session.funnelActive` is
true, ceding issuance to the warm-up completion effect, which was extended to
seed `cachedShare` (read/write URLs + single-use codes) from its **same**
`IssueCapabilities` response. Result: one issuance per modal open for Funnel
sessions instead of two, halving the per-open grant-set growth. Plain
(non-Funnel) shares are unchanged — the seeding effect still owns their
issuance. **Human verification:** confirm in the running app that opening the
Funnel share modal issues capabilities exactly once (network panel) and that
both the single-use RO/Full-access codes and the public code render correctly.

## Skipped Issues

None — all in-scope findings were fixed.

Out of scope (Info tier, not addressed per `fix_scope: critical_warning`):
- IN-01: Doc comment overstates the TOCTOU guarantee. (Note: WR-01 partially
  addresses the underlying gap; the comment at api.go ~1457 may now warrant a
  follow-up wording tweak.)
- IN-02: `IssueReusable` duplicates the code-generation body of `Issue`.

## Regression Coverage Added

Added 2026-07-05 in commit `23d273e5`, after the fix pass, to close the
"no regression tests" gap the review flagged. No new test files were created
(tests were appended to the existing, already-registered suites), so TESTING.md
needs no new rows. Each test was **mutation-verified**: it fails on the pre-fix
code and passes after the fix.

| Finding | Test | File | Mutation proof |
|---------|------|------|----------------|
| CR-01 | `TestJoinCodeManager_Rebind` | `internal/capability/joincode_test.go` | Rebind swaps token; preserves code string, reusable class, and expiry (not extended) |
| CR-01 | `TestJoinCodeManager_Rebind_UnknownCode` | `internal/capability/joincode_test.go` | Absent code → `ErrCodeNotFound` |
| CR-01 | `TestIssueCapabilities_BrowseToggleRebindsPublicCode` | `internal/daemon/funnel_test.go` | Removing the Rebind branch → test fails (public code resolves to a revoked grant → viewers 403) |
| WR-02 | `TestIssueCapabilities_ExpiredPublicCodeRemints` | `internal/daemon/funnel_test.go` | Reverting the gate to `if !ok` (drop expiry check) → test fails (stale code returned instead of re-mint) |

The daemon-layer tests reuse the existing `funnel_test` harness
(`testDaemon` + `makeFunnelTestWebServer` + `probeGrant`). CR-01's daemon test
asserts grant liveness via a live `/api/sessions/{id}/info` request (200 = grant
active, 403 = revoked). WR-02's test drives the daemon re-mint gate by forcing
`funnelReadCodeExpiry` into the past — the join-code manager's own clock cannot
be advanced from the `daemon` package (`SetClockForTest` is `capability`
-internal), so the manager-level TTL sweep is covered separately by the
pre-existing `TestJoinCodeManager_ReusableExpiresAfterTTL`.

**Still human-verification-only:** WR-01 (deterministic concurrency/interleaving
test is hard without a race harness) and WR-03 (frontend single-issuance dedupe,
best confirmed in the running app's network panel).

---

_Fixed: 2026-07-05_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
