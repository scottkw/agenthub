---
phase: 170-public-share-access-codes-reusable-share-lifetime-join-code
verified: 2026-07-05T20:25:00Z
status: passed
score: 9/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "M-46 live off-tailnet reusable-code join + share-lifetime teardown"
    expected: "On a real Tailscale-Funnel-granted machine, enable Funnel for a session, copy the 'Public join code (reusable):' value from the Share modal INTERNET (PUBLIC) section. From an off-tailnet device, open the public URL, land on /join, enter the code, confirm a read-only join. From a SECOND off-tailnet device, enter the SAME code and confirm it ALSO joins read-only (proving reusability). Then disable the internet share and confirm the code no longer resolves (proving share-lifetime teardown)."
    why_human: "Requires a real Funnel-granted tailnet plus at least one (ideally two) off-tailnet devices — cannot be exercised by the automated test suite, which the codebase itself acknowledges (TESTING.md M-46, 170-VALIDATION.md Manual-Only Verifications). This is the actual live UAT the phase goal describes ('a recipient who cannot scan the QR ... can join read-only with a short code') and has not yet been executed live in this session."
---

# Phase 170: Public Share Access Codes (read) Verification Report

**Phase Goal:** Funnel/internet shares surface a reusable, share-lifetime join code in the Share modal's INTERNET (PUBLIC) section, so a recipient who cannot scan the QR or paste the full capability URL can join **read-only** with a short code — closing the UAT dead-end where typing the public URL lands on a code-entry page with no code available.

**Verified:** 2026-07-05T20:25:00Z
**Status:** passed
**Re-verification:** No — initial verification (fresh retry after a prior verifier attempt died on a transient API error before writing any report; no prior VERIFICATION.md existed)

> **M-46 CLOSED LIVE 2026-07-06.** The one item routed to human verification — the live off-tailnet reusable-code join + share-lifetime teardown — was executed live on a real Funnel tailnet and PASSED all 3 conditions (read-only join on two off-tailnet devices with the same reusable code; no connection after Disable internet share). Live UAT surfaced a blocker the code-verification could not (the public URL pointed at the ephemeral `/sessions/{id}?cap=` cap link, not the reusable `/join?code=<publicReadCode>` entry point) → fixed in commit 5a92ddae (frontend-only, RED→GREEN TDD, full gate green). See 170-UAT.md. Status advanced human_needed → passed.

## Goal Achievement

### Observable Truths

Must-haves merged from all 4 plans' frontmatter (ROADMAP.md Phase 170 section carries a narrative Goal + Design Constraints list rather than a separate machine-readable `success_criteria` array, so plan-level `must_haves` are the authoritative source per Step 2's Option-C-adjacent fallback; all design constraints are covered below).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `IssueReusable` mints a code that survives repeated `Exchange` calls until its per-code TTL elapses or `Revoke` is called | VERIFIED | `internal/capability/joincode.go:93-105` implements `IssueReusable`; `Exchange` (`:138-154`) skips the success-delete when `entry.reusable`. `TestJoinCodeManager_IssueReusable_MultiExchange` — ran live, PASS. |
| 2 | `Revoke(code)` makes a previously-valid reusable code return `ErrCodeNotFound` on the next `Exchange` | VERIFIED | `Revoke` at `joincode.go:112-116` (`delete(m.codes, code)` under lock). `TestJoinCodeManager_Revoke` — ran live, PASS. |
| 3 | A reusable code past its per-call TTL returns `ErrCodeExpired` (and is deleted), same as single-use codes | VERIFIED | `Exchange`'s expiry branch (`joincode.go:146-149`) is unconditional for both classes. `TestJoinCodeManager_ReusableExpiresAfterTTL` — ran live, PASS. |
| 4 | The existing single-use `Issue`/`Exchange` contract is unchanged — all 6 pre-existing joincode tests pass unmodified | VERIFIED | Read `joincode_test.go` in full — all 6 original tests present, byte-for-byte logic unchanged, only 3 new tests appended. Orchestrator-confirmed full `go test -short ./...` green; independently re-ran the reusable-specific subset live. |
| 5 | The public `/join/exchange` HTTP handler resolves a reusable read code twice in a row (two 303 redirects to `/sessions/{id}?cap=…`, never `/join?error=`) | VERIFIED | `internal/webserver/join_test.go` — `TestJoinExchange_ReusableCodeSurvivesTwoExchanges` and the negative-guard `TestJoinExchange_SingleUseCodeStillFailsOnSecondExchange` — both ran live, PASS. `handleJoinExchange` in `server.go` shares the same `JoinCodeManager` instance via `ws.SetJoinCodes`, confirmed by reading the handler. |
| 6 | Funnel-enable mints a `PublicReadCode` via `IssueCapabilities`; a repeat call while the share is active returns the identical code (idempotent, cached per session) | VERIFIED | `internal/daemon/api.go:1466-1487` — cache-check (`a.funnelReadCode[sessionID]`) before minting, held under `a.mu`. `TestIssueCapabilities_FunnelPublicCode_Idempotent` + `TestIssueCapabilitiesForSession_FunnelPublicReadCode` — both ran live, PASS. |
| 7 | The `PublicReadCode` resolves (via `Exchange`) to a token whose `Perms` contains only read scopes — never `write`/`files.write` — in both browse OFF and browse ON states (the critical read-only-enforcement threat) | VERIFIED | `api.go:1457-1465,1474`: mint call is `a.joinCodes.IssueReusable(rTok, ttl)` — `rTok` only, never `wTok`, confirmed by reading the full `issueCapabilitiesForSession` body (`rTok`/`wTok` are separately signed at `:1409-1422`, only `rTok` is passed to `IssueReusable`). `TestFunnelPublicCode_ReadOnlyScope` (subtests `browse_off`/`browse_on`) + `TestIssueCapabilitiesForSession_FunnelPublicReadCode`'s `Verify`+`HasPerm` assertions — ran live, PASS. |
| 8 | `disableFunnelForSession` revokes the cached public code and clears the cache on every in-process teardown trigger (toggle-off, web-share-off, session-exit, auto-expiry timer) | VERIFIED | `api.go:1741-1745` — `Revoke` + double `delete` inside the single `disableFunnelForSession` chokepoint, called from all 4 in-process trigger sites per the doc comment (`:1719-1726`) and confirmed by reading `handleSetSessionFunnel`'s toggle-off branch. `TestFunnelTeardown_AllTriggers` (5 subtests, including `4_daemon_stop` which intentionally does not assert revocation since it bypasses this chokepoint by design — documented, not a gap) + `TestFunnelAutoExpiry_RevokesPublicReadCode` — ran live, PASS. |
| 9 | The public code's per-code TTL is `min(ExpiresIn, 8h)`; `ExpiresIn==0` caps at the 8h backstop | VERIFIED (code read) / flagged by plan authors as `human_judgment: true` for the literal-8h-constant assertion | `api.go:1680-1691` implements exactly `min(ExpiresIn, funnelReadCodeMaxTTL)` with `funnelReadCodeMaxTTL = 8 * time.Hour` (`:108`). No test asserts the literal 8h constant directly (would require reaching into an unexported field) — 170-02-SUMMARY.md itself flags this (coverage D3, `human_judgment: true`). Code inspection confirms the arithmetic matches the must-have exactly; counted as VERIFIED here on code-reading grounds (not a runtime behavior gap — it's a straightforward `if`/`min` computation, not a state-transition/cancellation invariant per Step 3's behavior-dependent-truth test). |
| 10 | A non-Funnel (ordinary tailnet/local) session returns an empty `PublicReadCode` | VERIFIED | `api.go:1466` — `publicReadCode` only set inside `if isFunnelSession`; zero value `""` otherwise. `TestIssueCapabilities_NonFunnelSession_EmptyPublicReadCode` — ran live, PASS. |
| 11 | The Share modal's INTERNET (PUBLIC) section renders a reusable join-code row (via `<CodeDisplay>`) below the public URL row, whenever the Funnel is live and the public code is present | VERIFIED | `SessionSharePanel.tsx:369-371` — `{publicReadCode && (<CodeDisplay label="Public join code (reusable):" code={publicReadCode} />)}` placed directly after the URL row (`:339-368`), inside the `funnelActive && funnelUrl && !warmingUp` branch (`:337`). |
| 12 | The public code's label signals reusability, differing from the single-use "Join code:" wording used by RO/Full-Access sections | VERIFIED | Label is literally `"Public join code (reusable):"` vs. `"Join code:"` at `:257` and `:308` (grep-confirmed both). |
| 13 | `SessionShareModal` captures `resp.publicReadCode` from the same warm-up `IssueCapabilities` re-issue that already sets `funnelUrl`, and clears it on disable | VERIFIED | `SessionShareModal.tsx:380-383` — `setFunnelUrl(resp.readUrl)` and `setPublicReadCode(resp.publicReadCode ?? null)` in the same effect, same response object, no second RPC. `:418` — `setPublicReadCode(null)` alongside `setFunnelUrl(null)` in `handleDisableFunnel`. |
| 14 | `models.ts`/`App.d.ts` mirror the new `publicReadCode` field so `resp.publicReadCode` is defined at runtime in a real wails/vite build, not just under vitest | VERIFIED | `frontend/src/wailsjs/go/main/App.d.ts:212` carries `publicReadCode: string` — and `SessionShareModal.tsx` imports `IssueCapabilities` from `../../wailsjs/go/main/App` (confirmed by reading the import block), i.e. the actual type the call site resolves. `frontend/src/wailsjs/wailsjs/go/models.ts:39,52` also carries the field (secondary generated file). Orchestrator-confirmed `tsc --noEmit` clean + `vite build` succeeds. |

**Score:** 9/9 truths verified, 0 present-but-behavior-unverified. (Table condenses to 9 rows above — several closely related must-haves from the same plan were merged where they share one piece of evidence; 14 individual assertions collectively covered, all VERIFIED.)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/capability/joincode.go` | `reusable` field, `IssueReusable`, `Revoke`, conditional-delete `Exchange` | VERIFIED | Read in full; matches plan/summary exactly (lines 21-24, 93-105, 112-116, 138-154). |
| `internal/capability/joincode_test.go` | Reusable multi-exchange, revoke, TTL-expiry tests + 6 preserved originals | VERIFIED | Read in full; 6 originals untouched, 3 new tests appended (note: SUMMARY lists 4 new tests including `Revoke_UnknownCodeIsNoOp`, all present). |
| `internal/webserver/join_test.go` | New file — reusable double-exchange at public HTTP layer | VERIFIED | New file confirmed present; both named tests ran live and passed. |
| `internal/daemon/api.go` | `funnelReadCode`/`funnelReadCodeTTL` maps, `funnelReadCodeMaxTTL` const, mint-cache-revoke wiring | VERIFIED | All present at documented line ranges; wiring traced end-to-end (mint → cache → response → revoke). |
| `internal/daemon/types.go` | `PublicReadCode string json:"publicReadCode"` on `IssueCapabilitiesResponse` | VERIFIED | Present at line 191 with doc comment. |
| `internal/daemon/funnel_test.go` + `api_test.go` | Read-only-scope, idempotent-mint, teardown-revocation tests | VERIFIED | All named tests present and ran live: `TestFunnelPublicCode_ReadOnlyScope`, `TestIssueCapabilities_FunnelPublicCode_Idempotent`, `TestFunnelAutoExpiry_RevokesPublicReadCode`, `TestFunnelTeardown_AllTriggers`, `TestIssueCapabilities_NonFunnelSession_EmptyPublicReadCode`, `TestIssueCapabilitiesForSession_FunnelPublicReadCode` — all PASS. |
| `frontend/src/wailsjs/wailsjs/go/models.ts` | `publicReadCode` field on `IssueCapabilitiesResponse` class + constructor | VERIFIED | Lines 39, 52. |
| `frontend/src/wailsjs/go/main/App.d.ts` | `publicReadCode` on the hand-authored interface (deviation from plan's "do not hand-edit App.d.ts" assumption, correctly self-corrected) | VERIFIED | Line 212 — this is the type actually resolved at the `IssueCapabilities()` call site used by the app; without this the plan's literal instruction would have left a silent `undefined` at runtime despite `models.ts` being synced. Executor caught and fixed this itself (170-03-SUMMARY.md Deviations section); code-read confirms the fix is real and necessary. |
| `frontend/src/components/SessionSharePanel.tsx` | `publicReadCode` prop + `<CodeDisplay>` row in Internet (public) block | VERIFIED | Lines 100, 129, 369-371. |
| `frontend/src/components/Hub/SessionShareModal.tsx` | `publicReadCode` state threaded from warm-up effect to panel prop | VERIFIED | Lines 323, 383, 418, 646. |
| `frontend/src/components/__tests__/SessionSharePanel.test.tsx` | Positive + negative assertions for the reusable code row | VERIFIED | Ran live: `pnpm vitest run SessionSharePanel` — 19/19 tests pass, including both new FNL-08 cases. |
| `TESTING.md` | Suite Manifest counts, FNL-08 traceability rows (5, superset of the 4 named), M-46 manual item | VERIFIED | `bash tests/check-traceability-paths.sh` ran live → `OK: all traceability paths exist`. All 5 FNL-08 rows present with file-path-only path columns. M-46 present under "Category X — Public Share Access Codes (FNL-08)" with why-not-automatable + source lines. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `internal/webserver/server.go` `handleJoinExchange` | `internal/capability` `JoinCodeManager.Exchange` | `ws.SetJoinCodes` shared instance | WIRED | Confirmed by reading `handleJoinExchange` (`server.go:~1072-1125`) — pulls `ws.joinCodes` under `ws.mu.RLock()`, calls `.Exchange(code)`, redirects on success. Same manager instance the daemon wires via `SetJoinCodes` (per `170-02-SUMMARY.md`, referenced in `api.go` construction, not independently re-verified beyond the passing integration tests). |
| `internal/daemon/api.go` `issueCapabilitiesForSession` | `internal/capability` `IssueReusable` | Direct call, `rTok`-only | WIRED | Line 1474: `a.joinCodes.IssueReusable(rTok, ttl)` — confirmed `rTok` (not `wTok`) is the only token variable in scope at that call site. |
| `internal/daemon/api.go` `disableFunnelForSession` | `internal/capability` `Revoke` | Direct call | WIRED | Line 1742: `a.joinCodes.Revoke(code)`. |
| `frontend SessionShareModal.tsx` warm-up effect | `frontend SessionSharePanel.tsx` `publicReadCode` prop | React state → prop | WIRED | `setPublicReadCode` (state) → `publicReadCode={publicReadCode}` at the panel invocation (line 646) → destructured prop rendered in `<CodeDisplay>`. |
| `frontend App.d.ts` / `models.ts` type | Runtime `resp.publicReadCode` at the `IssueCapabilities()` call site | Type declaration → field access | WIRED | Confirmed the import path (`../../wailsjs/go/main/App`) resolves to the file that was actually patched (`App.d.ts`), not just the secondary `models.ts` file — this is the correct fix location, verified by reading the import statement directly rather than trusting the SUMMARY's narrative alone. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `SessionSharePanel.tsx` public code row | `publicReadCode` prop | `SessionShareModal.tsx` state, set from `resp.publicReadCode` (real daemon `IssueCapabilities` RPC response, not a static/hardcoded value) | Yes — traced end-to-end: daemon mints a real crypto/rand code → JSON field → Wails RPC → React state → prop → render | FLOWING |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| FNL-08 | 170-01, 170-02, 170-03, 170-04 (all 4 plans declare it) | Reusable, share-lifetime, read-only public join code, supplementing (not replacing) the cap-URL/QR | SATISFIED | All code-level truths above verified; REQUIREMENTS.md line 21 carries the full FNL-08 text and is marked `[x]` (checked/done) in its own checklist rendering. |

**Note (doc-hygiene, not a code gap):** `.planning/REQUIREMENTS.md` line 92's tracking table still shows `| FNL-08 | Phase 170 | Planned |` even though line 21's checklist entry is `[x]` (done) and ROADMAP.md marks Phase 170 `[x] ... (completed 2026-07-06)`. This is a stale status label in a summary table, not a missing requirement — FNL-08 is fully accounted for and its acceptance text is present. Flagged as a minor WARNING for doc reconciliation, not a phase-goal blocker.

No orphaned requirements found — FNL-08 is the only requirement mapped to Phase 170 in REQUIREMENTS.md, and all 4 plans declare it.

### Anti-Patterns Found

None. Scanned all files modified across the 4 plans (`internal/capability/joincode.go`, `joincode_test.go`, `internal/webserver/join_test.go`, `internal/daemon/api.go`, `types.go`, `api_test.go`, `funnel_test.go`, `frontend/.../models.ts`, `SessionSharePanel.tsx`, `SessionShareModal.tsx`, `App.d.ts`, `SessionSharePanel.test.tsx`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` — zero matches (the only `XXXX` hits were pre-existing doc-comment format notation `XXXX-XXXX`, not debt markers).

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Reusable code survives 2 unit-level Exchange calls + Revoke + TTL-expiry | `go test ./internal/capability/... -run 'TestJoinCodeManager_(IssueReusable_MultiExchange\|Revoke$\|ReusableExpiresAfterTTL)' -v` | 3/3 PASS | PASS |
| Reusable code survives 2 public-HTTP `/join/exchange` calls; single-use still fails on 2nd | `go test ./internal/webserver/... -run TestJoinExchange -v` | 2/2 PASS | PASS |
| Daemon mint/cache/revoke/read-only-scope/idempotency/non-Funnel-empty | `go test ./internal/daemon/... -run 'TestFunnelPublicCode_ReadOnlyScope\|TestIssueCapabilities_FunnelPublicCode_Idempotent\|TestIssueCapabilities_NonFunnelSession_EmptyPublicReadCode\|TestIssueCapabilitiesForSession_FunnelPublicReadCode\|TestFunnelTeardown_AllTriggers\|TestFunnelAutoExpiry' -v` | 6 test groups, all subtests PASS | PASS |
| Frontend renders/hides the public code row correctly | `cd frontend && pnpm vitest run SessionSharePanel` | 19/19 tests PASS | PASS |
| TESTING.md traceability paths valid | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist` | PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files or phase-declared probes found for this phase (grep of PLAN/SUMMARY files for `probe-*.sh` returned no matches). Step 7c: SKIPPED (no probes declared or discovered).

## Human Verification Required

### 1. M-46 — Live off-tailnet reusable-code join + share-lifetime teardown

**Test:** On a Tailscale-connected machine with Funnel granted (production build), enable Funnel for a session, copy the "Public join code (reusable):" value from the Share modal's INTERNET (PUBLIC) section. From an off-tailnet device, open the public URL, land on the `/join` code-entry page, enter the code, and confirm a read-only join. From a SECOND off-tailnet device, enter the SAME code and confirm it ALSO joins read-only. Then disable the internet share and confirm the code no longer resolves.
**Expected:** Both off-tailnet devices join read-only using the same code (proving reusability); after disabling the share, re-entering the code fails (proving share-lifetime teardown).
**Why human:** Requires a real Funnel-granted tailnet plus off-tailnet devices — this is exactly the live, cross-network UAT the phase goal describes, and it is explicitly deferred as a manual item (M-46) by the phase's own test plan (170-04). It has not been executed live during this verification pass. This is the single most important behavior the phase promises (a recipient who cannot scan the QR/paste the URL can still join) and remains unproven end-to-end outside of the automated proxy tests (which do prove every component in isolation and at each interface boundary, but not the full live cross-device flow).

## Gaps Summary

No code-level gaps found. Every must-have truth derived from the 4 plans' frontmatter — including the security-critical read-only-only enforcement (public code is minted from `rTok` exclusively, never `wTok`, in both browse-off and browse-on states) — is implemented exactly as claimed and independently re-verified by reading the actual source (not just trusting SUMMARY prose) and by re-running the relevant named tests live (all passed). The one item routed to human verification is the phase's own explicitly-deferred live cross-device UAT (M-46), which by design cannot be automated and has not yet been executed. A minor doc-hygiene item (REQUIREMENTS.md status-table label still says "Planned" for FNL-08) is noted but does not block phase completion.

---
*Verified: 2026-07-05T20:25:00Z*
*Verifier: Claude (gsd-verifier)*
