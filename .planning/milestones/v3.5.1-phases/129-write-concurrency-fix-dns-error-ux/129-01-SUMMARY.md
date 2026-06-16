---
phase: 129-write-concurrency-fix-dns-error-ux
plan: "01"
subsystem: tests
tags: [tdd, red-phase, wave-0, race, dns, relay]
dependency_graph:
  requires: []
  provides:
    - TestCheckHealth_AcceptDNS (DNS-03 RED gate for Plan 129-03)
    - TestProxyRemoteFiles_AcceptDNSMessage (DNS-01/DNS-02 RED gate for Plan 129-02)
    - TestRemoteFiles_TwoWriterRace_RelaySurface (RACE-01 relay-surface RED gate for Plan 129-02)
    - newSandboxBackedRemotePeer (sandbox-backed upstream fixture for relay race test)
  affects:
    - internal/webserver/tailscale_test.go
    - internal/daemon/remote_files_test.go
    - internal/daemon/relay_remote_files_test.go
tech_stack:
  added: []
  patterns:
    - RED-first TDD wave (Wave 0 of 3)
    - Injectable prefsFunc mirroring existing statusFunc injection idiom (tailscale_test.go)
    - Sandbox-backed httptest.TLSServer for real WriteFileAtomic testing via HTTP
    - Relay loopback surface exercised through api.RelayHandler() (not webserver/fixture surface)
key_files:
  created: []
  modified:
    - internal/webserver/tailscale_test.go (TestCheckHealth_AcceptDNS added)
    - internal/daemon/remote_files_test.go (TestProxyRemoteFiles_AcceptDNSMessage added)
    - internal/daemon/relay_remote_files_test.go (newSandboxBackedRemotePeer + TestRemoteFiles_TwoWriterRace_RelaySurface added)
decisions:
  - "Wave 0 tests written against Plan 03 interface shape (injectable prefsFunc + AcceptDNS field); compile failure IS the DNS-03 RED signal"
  - "newSandboxBackedRemotePeer placed in relay_remote_files_test.go (not parity test) to keep relay-race concerns co-located"
  - "Relay race test uses -race detector to reliably trigger TOCTOU; goroutine scheduling makes it probabilistic without -race"
  - "Sub-case B (DNS-02) of TestProxyRemoteFiles_AcceptDNSMessage passes immediately; sub-case A (DNS-01) fails RED — this is correct (discrimination already works)"
metrics:
  duration: "~4 minutes"
  completed: "2026-06-15"
  tasks_completed: 3
  files_changed: 3
---

# Phase 129 Plan 01: Wave 0 RED Tests Summary

Three new test functions pin down Phase 129's behavior BEFORE any production code changes. Every test fails (RED) for the documented reason. Waves 1 and 2 (Plans 02 and 03) turn them green.

## One-liner

Wave 0 RED tests for RACE-01 relay-surface race, DNS-01/DNS-02 accept-dns message discrimination, and DNS-03 AcceptDNS field — all failing for documented reasons before production code exists.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | TestCheckHealth_AcceptDNS (DNS-03 RED) | 1744d05 | internal/webserver/tailscale_test.go |
| 2 | TestProxyRemoteFiles_AcceptDNSMessage (DNS-01/DNS-02 RED) | 9e785dd | internal/daemon/remote_files_test.go |
| 3 | TestRemoteFiles_TwoWriterRace_RelaySurface + sandbox-backed upstream (RACE-01 relay RED) | f72524b | internal/daemon/relay_remote_files_test.go |

## RED Signal Summary

| Test | RED Mechanism | Verified |
|------|--------------|---------|
| TestCheckHealth_AcceptDNS | Compile failure: `checkHealth` has wrong arity; `AcceptDNS` field missing from `TailscaleHealth` | go vet shows 3 errors |
| TestProxyRemoteFiles_AcceptDNSMessage (sub-case A) | Test FAIL: `proxyRemoteFiles` emits generic "remote unreachable" instead of DNS-specific message | `--- FAIL: sub-case A: DNS-01 FAIL` |
| TestProxyRemoteFiles_AcceptDNSMessage (sub-case B) | PASSES (correct): non-DNS error must NOT contain actionable string — discrimination already correct | `--- PASS: sub-case B` |
| TestRemoteFiles_TwoWriterRace_RelaySurface | Test FAIL under `-race`: TOCTOU allows both writers to win → `okCount=2` | RELAY-RED confirmed |

Note: Sub-case B of TestProxyRemoteFiles_AcceptDNSMessage passes because DNS-02 discrimination (absence check on non-DNS failures) is trivially true before the fix. The test as a whole is RED because sub-case A fails. This is correct per the plan spec.

## Pre-existing Baseline: TestWrite_TwoWritersIfMatchRace

**Confirmed RED with `-race` flag.** Observed counts (from multiple runs with `-count=100 -race`):

```
nilCount = 2; want exactly 1 successful writer
precondFailCount = 0; want exactly 1 ErrPreconditionFailed
```

Both goroutines win the race (both return nil from `WriteFileAtomic`) because the stat-check and rename are not atomic — the TOCTOU window allows both goroutines to pass the validator re-check before either commits the rename. This is the same root cause that `TestRemoteFiles_TwoWriterRace_RelaySurface` exercises on the relay surface.

Without `-race`, the test passes most of the time due to goroutine scheduling. The race detector reliably exposes the window. The per-path mutex (Plan 02) will close the window and make both tests pass deterministically.

## Interface Shape for Plan 03 (TestCheckHealth_AcceptDNS)

The test is written against the expected Plan 03 delivery:

```go
// Injectable prefs function (new parameter to checkHealth)
type prefsFunc func(ctx context.Context) (bool, error)  // returns (CorpDNS, error)

// Updated checkHealth signature
func checkHealth(ctx context.Context, fn statusFunc, customPath string, pf prefsFunc) TailscaleHealth

// New field on TailscaleHealth
type TailscaleHealth struct {
    // ... existing fields ...
    AcceptDNS bool `json:"acceptDns"` // DNS-03: CorpDNS from Tailscale prefs
}
```

Plan 03 only needs to add the parameter and field — the test already passes three table cases with the correct assertions.

## Deviations from Plan

None. Plan executed exactly as written.

Sub-case B behavior note: The plan spec says the test "FAILS now because proxyRemoteFiles emits only the generic message for all client.Do errors." This is correct at the test-function level — `TestProxyRemoteFiles_AcceptDNSMessage` reports FAIL because sub-case A fails. Sub-case B passing individually is expected and correct behavior (we're asserting absence of the DNS string, which is already absent before the fix).

## Known Stubs

None. All three tests use real infrastructure (actual DNS resolution, real files.Sandbox.WriteFileAtomic, real relay loopback through api.RelayHandler()).

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. Test-only changes.

T-129-02 (cap-token redaction) verified in TestProxyRemoteFiles_AcceptDNSMessage: `localCapToken` does not appear in either sub-case's 502 response body.

T-129-01 (relay surface blind spot) addressed: `TestRemoteFiles_TwoWriterRace_RelaySurface` exercises `api.RelayHandler()` — the actual GUI loopback path — not the webserver/fixture surface.

## Self-Check: PASSED

Files created/modified exist:
- internal/webserver/tailscale_test.go: contains `func TestCheckHealth_AcceptDNS`
- internal/daemon/remote_files_test.go: contains `func TestProxyRemoteFiles_AcceptDNSMessage`
- internal/daemon/relay_remote_files_test.go: contains `func TestRemoteFiles_TwoWriterRace_RelaySurface` and `func newSandboxBackedRemotePeer`

Commits verified:
- 1744d05: test(129-01): add TestCheckHealth_AcceptDNS RED for DNS-03
- 9e785dd: test(129-01): add TestProxyRemoteFiles_AcceptDNSMessage RED for DNS-01/DNS-02
- f72524b: test(129-01): add relay-surface race test + sandbox-backed upstream for RACE-01 RED
