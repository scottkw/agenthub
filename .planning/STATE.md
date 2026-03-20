---
gsd_state_version: 1.0
milestone: v1.2
milestone_name: Tailscale-Only Networking
status: unknown
stopped_at: Completed 15-02-PLAN.md
last_updated: "2026-03-20T19:48:49.427Z"
progress:
  total_phases: 5
  completed_phases: 2
  total_plans: 4
  completed_plans: 4
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-20)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 15 — tailscale-tls-interface-binding

## Current Position

Phase: 15 (tailscale-tls-interface-binding) — EXECUTING
Plan: 2 of 2

## Performance Metrics

**Velocity:**

- Total plans completed: 0 (v1.2)
- Average duration: ~25 min (from v1.1 baseline)
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| — | — | — | — |

*Updated after each plan completion*
| Phase 14-tailscale-health-check-infrastructure P01 | 15 | 2 tasks | 4 files |
| Phase 14-tailscale-health-check-infrastructure P02 | 10 | 2 tasks | 2 files |
| Phase 15-tailscale-tls-interface-binding P01 | 215 | 2 tasks | 4 files |
| Phase 15 P02 | 480 | 2 tasks | 7 files |

## Accumulated Context

### Decisions

Decisions logged in PROJECT.md Key Decisions table.
Key v1.2 constraints affecting all phases:

- Use `tailscale.com/client/local` zero-value `local.Client{}` — queries existing tailscaled daemon via Unix socket; no tsnet, no embedded daemon
- Build order is a safety dependency chain: health checks → TLS + IP binding → auth removal → dead code deletion
- Cert pattern: `tls.Config{GetCertificate: lc.GetCertificate}` only; never cache CertPair at startup
- FQDN: always derive from `lc.CertDomains(ctx)[0]`; zero hardcoded `.ts.net` strings in URL construction
- Binary size: measure delta after `go get tailscale.com@v1.96.3` in Phase 14; fallback to `github.com/tailscale/tscert` documented if binary exceeds ~25 MB
- [Phase 14-01]: statusFunc type defined in tailscale.go (not test file) to avoid duplicate type in same-package tests
- [Phase 14-01]: Connected==(BackendState=='Running') only; all 5 non-Running states correctly yield Connected=false
- [Phase 14-02]: GetTailscaleStatus uses context.Background() (not a.ctx) so callable before Wails fully initialises
- [Phase 14-02]: Struct equality (h != last) prevents event storms when Tailscale state is stable
- [Phase 14-02]: startHealthPoller goroutine bound to Wails ctx for lifecycle alignment with app shutdown
- [Phase 15-01]: Use InsecureSkipVerify for fresh token-test clients (these test auth logic, not TLS trust)
- [Phase 15-01]: Tailscale GetCertificate hook in Start(); TLSConfig field in Config for test isolation
- [Phase 15-02]: StartWebServer(port int) gates on Tailscale health — IP and FQDN derived from daemon, not user config
- [Phase 15-02]: CT disclosure persisted as sentinel file ct_disclosed; HasCTDisclosure reads os.Stat
- [Phase 15-02]: TestGetSessionQRCode bypasses StartWebServer Tailscale gate via direct WebServer construction with in-memory TLS

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 15]: Requires live manual verification against a real tailscaled daemon with MagicDNS + HTTPS certs enabled — user-environment dependency, not a code issue
- [Phase 14]: macOS App Store Tailscale may not expose CLI on PATH; use `StatusWithoutPeers` as primary "is installed" signal; treat connection error as not installed/not running

## Session Continuity

Last session: 2026-03-20T19:48:49.424Z
Stopped at: Completed 15-02-PLAN.md
Resume file: None
