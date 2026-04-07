---
phase: 50-tailscale-peer-discovery
plan: "01"
subsystem: tailnet
tags: [go, tailscale, peer-discovery, tdd]
dependency_graph:
  requires: []
  provides: [internal/tailnet package, Peer type, DiscoverPeers, ProbePeer, DiscoverAndProbe]
  affects: [50-02-PLAN.md (daemon route), future GUI and CLI consumers]
tech_stack:
  added: []
  patterns: [injectable-statusFunc, errgroup.SetLimit, httptest.NewTLSServer+rewriteTransport]
key_files:
  created:
    - internal/tailnet/tailnet.go
    - internal/tailnet/tailnet_test.go
  modified: []
decisions:
  - Use injectable statusFunc pattern (mirrors internal/webserver/tailscale.go)
  - Use rewriteTransport to redirect probePeer tests to httptest.NewTLSServer without changing port logic
  - No InsecureSkipVerify — Tailscale LE certs trusted by system CAs via FQDN
  - Probe port 7443 only (non-default ports deferred to v2+)
metrics:
  duration_seconds: 952
  completed_date: "2026-04-07"
  tasks_completed: 1
  tasks_total: 1
  files_created: 2
  files_modified: 0
---

# Phase 50 Plan 01: Tailscale Peer Discovery Package Summary

**One-liner:** Injectable-dependency tailnet package with DiscoverPeers/ProbePeer/DiscoverAndProbe, backed by 12 race-clean tests using errgroup.SetLimit(5) and httptest.NewTLSServer redirect transport.

## What Was Built

Created `internal/tailnet` — a pure Go package for discovering AgentHub instances on the tailnet.

### Package: `internal/tailnet/tailnet.go`

| Symbol | Type | Purpose |
|--------|------|---------|
| `DefaultProbePort` | `const int = 7443` | Matches SettingsPanel.tsx default |
| `Peer` | `struct` | Online tailnet peer with json tags |
| `statusFunc` | `type func` | Injectable Tailscale status function |
| `probeFunc` | `type func` | Injectable probe function |
| `discoverPeers` | `func` | Testable inner function (injected statusFunc) |
| `DiscoverPeers` | `func` | Public: queries local tailscaled |
| `probePeer` | `func` | Testable inner function (injected *http.Client) |
| `ProbePeer` | `func` | Public: 2s timeout, system-CA TLS |
| `probeAll` | `func` | Concurrent probing capped at 5 goroutines |
| `DiscoverAndProbe` | `func` | Composes discovery + probing |

### Test Coverage: `internal/tailnet/tailnet_test.go`

12 test functions, all passing with `-race` flag:

| Test | What It Verifies |
|------|-----------------|
| `TestDiscoverPeers_OnlineOnly` | Only online peers returned; all fields mapped correctly |
| `TestDiscoverPeers_Empty` | Empty peer map returns `[]Peer{}` (not nil) |
| `TestDiscoverPeers_Error` | statusFunc error propagated, nil peers returned |
| `TestProbePeer_Found` | 200 OK → returns true |
| `TestProbePeer_NotFound` | 404 → returns false |
| `TestProbePeer_Timeout` | 500ms deadline → returns false (2s internal timeout) |
| `TestProbePeer_DNSNameDotStrip` | DNSName trailing dot stripped from URL path |
| `TestProbeAll_Concurrent` | 10 peers, max 5 concurrent goroutines via errgroup.SetLimit |
| `TestProbeAll_FiltersNonResponders` | Only responding peers returned |
| `TestDiscoverAndProbe_Integration` | discoverPeers + probeAll composition verified |
| `TestDiscoverPeers_Public` | Public wrapper compiles and runs (live daemon optional) |
| `TestProbePeer_Public` | Unreachable host returns false within 3s |

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Mirror injectable-statusFunc from webserver/tailscale.go | Consistent project pattern; enables unit tests without live daemon |
| `rewriteTransport` RoundTripper for probe tests | Redirects probePeer (which builds its own URL from DNSName+port) to httptest.NewTLSServer without modifying production logic |
| No `InsecureSkipVerify` | Tailscale Let's Encrypt certs are FQDN-only and publicly trusted; using skip-verify hides connectivity problems |
| Probe port 7443 only | Matches SettingsPanel.tsx default; non-default port discovery is v2+ |
| `strings.TrimSuffix(peer.DNSName, ".")` | PeerStatus.DNSName documented as "ends with a dot" — must strip for URL construction |
| `make([]Peer, 0, len(status.Peer))` | Ensures never-nil slice even when no online peers |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Deferred Items

Pre-existing flaky test logged: `TestHub_SlowClientDisconnected` in `internal/relay` fails intermittently (~1/3 runs) under system load. This test is independent of the tailnet package and existed before this plan. Logged in `.planning/phases/50-tailscale-peer-discovery/deferred-items.md`.

## Self-Check: PASSED
