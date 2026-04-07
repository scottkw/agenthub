---
phase: 50
slug: tailscale-peer-discovery
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-07
---

# Phase 50 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — standard Go test tooling |
| **Quick run command** | `go test -race ./internal/tailnet/...` |
| **Full suite command** | `go test -race -count=1 ./internal/tailnet/... ./internal/webserver/...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./internal/tailnet/...`
- **After every plan wave:** Run `go test -race -count=1 ./internal/tailnet/... ./internal/webserver/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 50-01-01 | 01 | 1 | REM-01 | unit | `go test -race ./internal/tailnet/...` | ❌ W0 | ⬜ pending |
| 50-01-02 | 01 | 1 | REM-01 | unit | `go test -race ./internal/tailnet/...` | ❌ W0 | ⬜ pending |
| 50-02-01 | 02 | 2 | REM-01 | integration | `go test -race ./internal/webserver/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/tailnet/tailnet_test.go` — stubs for DiscoverPeers, ProbePeer
- [ ] Test fixtures with mock StatusFunc and httptest TLS server

*Existing Go test infrastructure covers tooling requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live Tailscale peer discovery | REM-01 | Requires real tailnet with multiple peers | Start AgentHub on 2+ tailnet machines, verify /tailnet/peers returns both |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
