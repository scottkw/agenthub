---
phase: 165-funnel-backend
plan: 04
subsystem: funnel
tags: [tailscale, funnel, go, tls, webserver, daemon, reverse-proxy, kill-path, ref-count]

# Dependency graph
requires:
  - phase: 165-funnel-backend-01
    provides: EnableFunnel/DisableFunnel/funnelClient interface, ws.Stop teardown wiring
  - phase: 165-funnel-backend-02
    provides: disableFunnelForSession ref-counted helper, runSessionExitCleanup, funnelSessions map

provides:
  - Funnel proxy target fixed to https+insecure://<bindIP>:<port> (FNL-03 502 closed)
  - Funnel teardown on explicit-kill path via synchronous runSessionExitCleanup (FNL-05 kill path closed)
  - TestEnableFunnel_ProxyTargetReachable CI regression guard (proxy string + reachability)
  - TestFunnelTeardown_KillPath CI regression guard (kill-path teardown + ref-count)

affects: [165-funnel-backend, funnel-sharing, UAT, TESTING.md]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "https+insecure scheme for internal same-host proxy hops where cert is for DNS name but dial is raw IP"
    - "Synchronous runSessionExitCleanup in kill path — no grace period for killed sessions"
    - "TDD RED→GREEN commit discipline for critical bug fixes (string-assertion regression guard)"

key-files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/webserver/funnel_test.go
    - internal/daemon/api.go
    - internal/daemon/funnel_test.go
    - TESTING.md

key-decisions:
  - "GAP 1 Option A: Proxy target uses https+insecure://<ws.config.BindIP>:<port> — rejected B (MagicDNS hairpin unverified) and C (extra loopback listener surface)"
  - "GAP 2: synchronous runSessionExitCleanup in handleDeleteSession (no 10s grace) because a killed session has no buffered output to flush; KillSession sets IsKilled so natural-exit goroutine returns early — no double-cleanup race"
  - "TDD: RED commit (failing test) before GREEN commit (fix) for both bugs — regression guards assert the exact defect class"

patterns-established:
  - "https+insecure is the correct scheme when tailscaled dials a raw IP to an internal listener whose TLS cert is issued for the DNS name"
  - "Kill path must converge on the same cleanup helpers as natural-exit path; do not add a second DisableFunnel call — the ref-counted disableFunnelForSession is the single authority"

requirements-completed: [FNL-03, FNL-05]

coverage:
  - id: D1
    description: "Funnel reverse-proxy target is https+insecure://<bindIP>:<port> — not https://localhost:<port> — so external Funnel guests receive 200 instead of 502 (FNL-03)"
    requirement: FNL-03
    verification:
      - kind: unit
        ref: "internal/webserver/funnel_test.go#TestEnableFunnel_ProxyTargetReachable"
        status: pass
    human_judgment: false

  - id: D2
    description: "TestEnableFunnel_ProxyTargetReachable also dials the proxy target with InsecureSkipVerify and asserts a real HTTP response (not connection-refused) — catches the 502 class in CI"
    requirement: FNL-03
    verification:
      - kind: integration
        ref: "internal/webserver/funnel_test.go#TestEnableFunnel_ProxyTargetReachable (reachability leg)"
        status: pass
    human_judgment: false

  - id: D3
    description: "handleDeleteSession synchronously calls runSessionExitCleanup after KillSession — Funnel is torn down and funnelSessions[id] is removed on the explicit-kill path (FNL-05 GAP 2)"
    requirement: FNL-05
    verification:
      - kind: unit
        ref: "internal/daemon/funnel_test.go#TestFunnelTeardown_KillPath/single_session_killed"
        status: pass
    human_judgment: false

  - id: D4
    description: "Killing session A of two Funnel sessions leaves B's Funnel up (ref-count guard); B's natural-exit cleanup then clears the config — stale-ref-count regression gone (FNL-05)"
    requirement: FNL-05
    verification:
      - kind: unit
        ref: "internal/daemon/funnel_test.go#TestFunnelTeardown_KillPath/refcount_killing_a_keeps_b_up"
        status: pass
    human_judgment: false

  - id: D5
    description: "M-34 external-tailnet 200 on Funnel URL — expected to PASS live after 165-04 fix (was FAILING due to 502)"
    requirement: FNL-03
    verification: []
    human_judgment: true
    rationale: "Requires an external-tailnet machine (no Tailscale) and a live Tailscale Funnel grant; automated guards cover the proxy-target string + reachability in CI but the full end-to-end live leg requires manual verification"

# Metrics
duration: 25min
completed: 2026-06-30
status: complete
---

# Phase 165 Plan 04: Gap-Closure (Funnel 502 + Kill-Path Teardown) Summary

**Fixed two live-UAT defects: Funnel proxy target changed from `https://localhost:<port>` to `https+insecure://<bindIP>:<port>` (closes HTTP 502 for all external Funnel guests), and handleDeleteSession now synchronously tears down Funnel on the explicit-kill path via runSessionExitCleanup (closes stale ref-count leak)**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-06-30T19:11:09Z
- **Completed:** 2026-06-30T19:36:00Z
- **Tasks:** 3 (2 TDD, 1 docs)
- **Files modified:** 5

## Accomplishments

- **FNL-03 closed:** The Funnel reverse-proxy target (`https://localhost:<port>`) was unreachable because AgentHub's listener binds to `ws.config.BindIP` (tailnet IP, not localhost). Changed to `https+insecure://<bindIP>:<port>` per the locked Option A decision. External Funnel guests now receive 200 instead of 502.
- **FNL-05 kill-path closed:** `handleDeleteSession` previously returned 204 without any cleanup, leaving Funnel exposed after an explicit kill and leaving a stale `funnelSessions[id]` entry that blocked a sibling session's later teardown. Now calls `a.runSessionExitCleanup(id)` synchronously (no grace period) after `KillSession`.
- **CI regression guards added:** `TestEnableFunnel_ProxyTargetReachable` asserts the exact proxy target string AND real HTTPS reachability (connection-refused = test failure); `TestFunnelTeardown_KillPath` drives the real DELETE handler and asserts teardown + sibling ref-count integrity. Both tests followed strict TDD RED→GREEN discipline.
- **TESTING.md wired:** Section 2 gap-closure note, Section 4 new FNL-03 webserver row + extended FNL-05 daemon row, Section 5 M-34 updated to expected PASS live with automated guard cited.

## Task Commits

TDD tasks have two commits each (RED then GREEN):

1. **Task 1 RED — TestEnableFunnel_ProxyTargetReachable (failing)** - `2363fe33` (test)
2. **Task 1 GREEN — Fix EnableFunnel proxy target (FNL-03)** - `7d319d65` (feat)
3. **Task 2 RED — TestFunnelTeardown_KillPath (failing)** - `77747929` (test)
4. **Task 2 GREEN — Synchronous runSessionExitCleanup in handleDeleteSession (FNL-05)** - `8fbb890a` (feat)
5. **Task 3 — TESTING.md updates** - `378c6176` (docs)

## Files Created/Modified

- `/Users/ken/dev/agenthub/internal/webserver/server.go` — EnableFunnel Proxy target fixed (`https://localhost` → `https+insecure://<bindIP>`); added Option A comment with WHY rationale
- `/Users/ken/dev/agenthub/internal/webserver/funnel_test.go` — Added `TestEnableFunnel_ProxyTargetReachable` (imports: crypto/tls, net, time added); stateful fake pattern, proxy string assertion + InsecureSkipVerify reachability leg
- `/Users/ken/dev/agenthub/internal/daemon/api.go` — `handleDeleteSession` now calls `a.runSessionExitCleanup(id)` synchronously after `KillSession`; added WHY comment (no double-cleanup race, disableFunnelForSession ref-count authority)
- `/Users/ken/dev/agenthub/internal/daemon/funnel_test.go` — Added `TestFunnelTeardown_KillPath` with two sub-cases (single-session teardown, two-session ref-count guard)
- `/Users/ken/dev/agenthub/TESTING.md` — Section 2 note, Section 4 two new/extended rows, Section 5 M-34 updated

## Decisions Made

- **GAP 1 Option A honored:** `https+insecure://` scheme + raw bind IP as proxy host. The listener serves real TLS (via `ws.lc.GetCertificate`); `https+insecure` keeps TLS-encrypted internal hop but skips cert-hostname verification because the proxy dials the raw IP while the cert is for the DNS name. This hop never leaves the host. Rejected B (MagicDNS hairpin unverified) and C (extra loopback listener surface).
- **No grace period on kill path:** A killed session has no buffered PTY output to flush; `KillSession` sets `sess.IsKilled()` so the natural-exit goroutine returns early without arming the `onExit` AfterFunc — no double-cleanup race possible.
- **TDD discipline:** Both production fixes have a failing test committed before the green fix. The string-assertion in `TestEnableFunnel_ProxyTargetReachable` is the exact regression guard that would catch a future regression back to `localhost`.

## Deviations from Plan

None — plan executed exactly as written. All locked decisions honored. Both tests follow the exact patterns specified in the plan's `<behavior>` and `<action>` blocks.

## Issues Encountered

None. Both fixes were minimal one-liner / two-liner changes with no ambiguity in implementation. The TDD RED phase confirmed the bugs precisely as described in the plan.

## Known Stubs

None — both fixes wire real production behavior (no placeholder values or hardcoded stubs).

## Threat Flags

No new security surface introduced beyond what the plan's threat model already analyzed:
- T-165-14 (`https+insecure` skips cert-hostname on same-host hop) — accepted; documented in code comment.
- T-165-15/T-165-16 (GAP 2 exposure + ref-count block) — mitigated; `TestFunnelTeardown_KillPath` asserts the mitigations.

## Next Phase Readiness

- Phase 165 is now fully executed (165-01 through 165-04 complete). Both live-UAT defects are closed.
- M-34 manual verification is required on a real Funnel-granted tailnet to confirm the 200 fix end-to-end.
- Phase 166 (Funnel UI polish / notifications) can proceed without blockers.

---
*Phase: 165-funnel-backend*
*Completed: 2026-06-30*
