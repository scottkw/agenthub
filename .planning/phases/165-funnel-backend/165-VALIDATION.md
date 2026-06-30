---
phase: 165
slug: funnel-backend
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-30
---

# Phase 165 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./internal/webserver/... ./internal/daemon/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~TBD seconds (planner/Wave 0 to confirm package paths) |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** TBD seconds

---

## Per-Task Verification Map

*Populated by the planner — every task maps to a requirement and an automated `go test` command. See RESEARCH.md `## Validation Architecture` for the sampling targets (four teardown sites, dual-origin allowlist, ETag preserve-on-modify, CheckFunnelAccess preflight, fallback-mode guard, FNL-07 expiry timer).*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | — | — | FNL-01..07 | — | — | unit | `go test ...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `funnelClient` interface seam (3 methods) so CI tests run without a live tailscaled daemon — mirrors the existing `statusFunc`/`prefsFunc` injection pattern in `tailscale.go`
- [ ] Test files registered in TESTING.md per the standing regression-test convention

*Planner to enumerate concrete test files once task breakdown is set.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| External-tailnet 200-not-403 with `Origin: https://hostname.ts.net` | FNL-04 | Requires a real machine outside the tailnet hitting a live Funnel URL | From a non-tailnet machine, open the emitted Funnel share URL; confirm 200 (not 403) and join succeeds |
| `tailscale serve status` empty after all four teardown triggers | FNL-05 | Asserts live Tailscale daemon serve-config state | After each of: toggle-off, web-share-off, session end, daemon stop — run `tailscale serve status`; confirm empty |
| Fallback-mode web-share unaffected (Tailscale not running) | — | Requires Tailscale stopped on host | Stop Tailscale, start web-share; confirm it works and `funnelActive` stays false |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < TBD s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
