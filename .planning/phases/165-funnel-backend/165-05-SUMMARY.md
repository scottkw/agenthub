---
phase: 165-funnel-backend
plan: 05
subsystem: networking
tags: [tailscale, funnel, tls, loopback, http, webserver, tdd]

requires:
  - phase: 165-funnel-backend
    provides: "FNL-05 teardown (Stop/DisableFunnel lifecycle), EnableFunnel serve-config wiring, 165-04 gap-closure kill-path"

provides:
  - "Plain-HTTP loopback listener (127.0.0.1, ephemeral port) bound in startTailscale, serving ws.mux alongside the TLS listener"
  - "EnableFunnel proxy target is now http://127.0.0.1:<loopbackPort> — the dead https+insecure://<bindIP>:<tlsPort> target (165-04) is removed"
  - "Stop() restructured to close both TLS and loopback listeners atomically under one RLock snapshot"
  - "Rewritten TestEnableFunnel_ProxyTargetReachable: scheme==http guard, host==127.0.0.1, loopback port != TLS port, plain http.Client reachability dial"
  - "TESTING.md Section 2 note + Section 4 FNL-03 traceability + Section 5 M-34 updated to the loopback-HTTP fix"

affects: ["165-funnel-backend M-34 live UAT (off-tailnet re-test is the real FNL-03 acceptance gate)"]

tech-stack:
  added: []
  patterns:
    - "Co-located loopback HTTP listener as Funnel hop-2 proxy target: tailscaled owns all public TLS on hop 1; hop 2 is plain HTTP on kernel loopback (never leaves the machine)"
    - "Stop() listener snapshot pattern: RLock, snapshot both listeners, RUnlock, then close both — prevents deadlock and handles the two-listener case atomically"
    - "TDD RED/GREEN: write the failing test first (scheme/port assertions), commit test commit, then implement, confirm GREEN"

key-files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/webserver/funnel_test.go
    - TESTING.md

key-decisions:
  - "Loopback-HTTP target is safe ONLY because tailscaled and AgentHub are CO-LOCATED on the same host (kernel loopback never leaves the machine). If they ever split across tailnet nodes the target MUST become a WireGuard-tunneled tailnet-IP target. Recorded in code comments and threat model (T-165-18)."
  - "Enable-time binding was REJECTED in 165-05 design. The loopback listener is bound at startTailscale (start-time), not at EnableFunnel (enable-time). Enable-time would require double-bind guards on re-enable, a separate close path outside Stop(), and binding under the EnableFunnel Lock — more surface, weaker teardown reuse."
  - "Do NOT add any Host-based origin check. The dual-origin allowlist keys off the browser Origin header (preserved by tailscaled's proxy), not the Host header (which tailscaled may rewrite to 127.0.0.1:<loopbackPort>). Adding a Host==Origin check would cause a 403 on every Funnel request (WARNING-1 from design_decisions)."
  - "The unit test (TestEnableFunnel_ProxyTargetReachable) guards target SHAPE + loopback reachability only. Live M-34 (off-tailnet, no Tailscale client) is the real FNL-03 acceptance gate — the automated test alone is NOT sufficient to certify FNL-03 closed, given the 165-04 false-green precedent."

patterns-established:
  - "Co-location guard comment pattern: every site that depends on tailscaled+AgentHub co-location must have an explicit code comment naming the guard and the consequence of a future split."

requirements-completed: [FNL-03]

coverage:
  - id: D1
    description: "Plain-HTTP loopback listener (127.0.0.1, ephemeral port) added in startTailscale, serving ws.mux; Stop() closes it alongside the TLS listener"
    requirement: FNL-03
    verification:
      - kind: unit
        ref: "internal/webserver/funnel_test.go#TestEnableFunnel_ProxyTargetReachable (loopbackLn != nil assertion + port != tlsPort assertion)"
        status: pass
      - kind: unit
        ref: "go test ./internal/webserver/... -count=1 (full suite green)"
        status: pass
    human_judgment: false
  - id: D2
    description: "EnableFunnel proxy target is http://127.0.0.1:<loopbackPort> — scheme plain http, host 127.0.0.1, port == loopback listener port != TLS listener port"
    requirement: FNL-03
    verification:
      - kind: unit
        ref: "internal/webserver/funnel_test.go#TestEnableFunnel_ProxyTargetReachable (scheme, host, port assertions + anti-regression https/https+insecure guard)"
        status: pass
      - kind: unit
        ref: "internal/webserver/funnel_test.go#TestEnableFunnel_ProxyTargetReachable (plain http.Client GET returns 200)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Live M-34: external guest (off-tailnet device, no Tailscale client) loads the Funnel URL with HTTP 200 — the real end-to-end FNL-03 gate"
    requirement: FNL-03
    verification: []
    human_judgment: true
    rationale: "The SNI/ingress failure is only reproducible on a live tailnet with a real Tailscale Funnel-granted tailnet. Automated test guards target SHAPE + loopback reachability only; given the 165-04 false-green precedent (loopback self-signed cert answered any SNI), the live off-tailnet hop is the real proof."

duration: 7min
completed: 2026-06-30
status: complete
---

# Phase 165 Plan 05: Funnel Loopback-HTTP Gap-Closure Summary

**FNL-03 proxy-target layer closed: AgentHub adds a plain-HTTP loopback listener (127.0.0.1, ephemeral) as the Funnel hop-2 target, replacing the proven-dead https+insecure://<bindIP>:<tlsPort> target from 165-04**

## Performance

- **Duration:** 7 min
- **Started:** 2026-06-30T20:56:27Z
- **Completed:** 2026-06-30T21:03:11Z
- **Tasks:** 3 (Task 2 was TDD with RED + GREEN commits)
- **Files modified:** 3

## Accomplishments

- Added `loopbackListener net.Listener` field to WebServer struct; `startTailscale()` binds `net.Listen("tcp", "127.0.0.1:0")` (plain TCP/HTTP, ephemeral port) and serves `ws.mux` on it — no TLS, no cert
- `Stop()` restructured from `return ln.Close()` (single-listener early-return) to snapshot both listeners under one RLock and close both, with the loopback error discarded (preserves existing Stop semantics)
- `EnableFunnel` Step 5 now reads `ws.loopbackListener.Addr()` under the held Lock and sets Proxy to `"http://" + net.JoinHostPort("127.0.0.1", loopbackPort)` — the dead `https+insecure://<bindIP>:<tlsPort>` target and the `localPort` derivation from `ws.listener` are removed
- `TestEnableFunnel_ProxyTargetReachable` rewritten (TDD): RED committed first (scheme/port assertions fail against old https+insecure target), then GREEN implemented; asserts scheme==http (rejects https/https+insecure explicitly as 165-04 anti-regression), host==127.0.0.1, loopback port != TLS port, and dials with a PLAIN `http.Client` requiring a served response (connection-refused = dead target, fails the test)
- TESTING.md updated: Section 2 Phase 165-05 note, Section 4 FNL-03 row updated to loopback-HTTP shape, Section 5 M-34 updated with the loopback-HTTP fix and explicit statement that the automated test does not substitute for the live off-tailnet hop

## IMPORTANT — FNL-03 Verification Status

The unit test guards the **target SHAPE + loopback reachability** only. The true end-to-end FNL-03 gate is a **live M-34 re-test from an off-tailnet device** (machine with no Tailscale client, hitting `https://<hostname>.ts.net/sessions/<id>?cap=<token>`). This plan does NOT claim FNL-03 is fully verified — the orchestrator must run the live M-34 re-test after execution to confirm the Funnel 502 is closed.

## Task Commits

1. **Task 1: Add loopback HTTP listener + lifecycle** - `628cc94f` (feat)
2. **Task 2 RED: Failing test for loopback-HTTP proxy target** - `0cdf6e07` (test)
3. **Task 2 GREEN: Flip Funnel proxy target to loopback-HTTP** - `71f2caab` (feat)
4. **Task 3: Update TESTING.md** - `7a85d04d` (chore)

## Files Created/Modified

- `internal/webserver/server.go` — `loopbackListener` field, `startTailscale()` loopback bind + serve + error cleanup, `Stop()` two-listener close, `EnableFunnel` Step 5 loopback-HTTP target + co-location comment
- `internal/webserver/funnel_test.go` — `TestEnableFunnel_ProxyTargetReachable` rewritten (scheme/host/port/reachability assertions, plain http.Client, anti-regression guard vs https+insecure); `crypto/tls` import removed, `net/url` added
- `TESTING.md` — Section 2 Phase 165-05 note, Section 4 FNL-03 row updated, Section 5 M-34 updated

## Decisions Made

- **Loopback-HTTP target (co-location accepted-risk-with-guard):** Safe only because tailscaled and AgentHub are CO-LOCATED on the same host (kernel loopback, never on any wire). Recorded in code comments + threat model T-165-18. A future split MUST become a WireGuard-tunneled tailnet-IP target.
- **Start-time binding, not enable-time:** The loopback listener is bound in `startTailscale()`, not in `EnableFunnel()`. Enable-time would require double-bind guards on re-enable and binding under the EnableFunnel Lock.
- **No Host-based origin check added:** The dual-origin allowlist keys off the browser `Origin` header (set by the guest's browser, forwarded by tailscaled), not the `Host` header (which tailscaled may rewrite). Adding a Host==Origin check would cause 403 on every Funnel request (design_decisions WARNING-1).

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- FNL-03 proxy-target layer closed in code and tests; the loopback HTTP listener is live and reachable
- **Blocker for final FNL-03 sign-off:** live M-34 re-test from an off-tailnet device must PASS (orchestrator/operator runs this after this plan's commits are built and deployed)
- FNL-05 teardown (M-35 a/b/c), FNL-04 dual-origin allowlist, and M-36 fallback-mode are unaffected (no daemon or origin_mw changes in this plan)

---
*Phase: 165-funnel-backend*
*Completed: 2026-06-30*
