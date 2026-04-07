---
phase: 50
slug: tailscale-peer-discovery
status: audited
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-07
audited: 2026-04-07
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
| **Full suite command** | `go test -race -count=1 ./internal/tailnet/... ./internal/daemon/...` |
| **Estimated runtime** | ~7 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./internal/tailnet/...`
- **After every plan wave:** Run `go test -race -count=1 ./internal/tailnet/... ./internal/daemon/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 50-01-01 | 01 | 1 | REM-01 | unit | `go test -race ./internal/tailnet/...` | ✅ | ✅ green |
| 50-02-01 | 02 | 2 | REM-01 | integration | `go test -race ./internal/daemon/...` | ✅ | ✅ green |

**Test file coverage:**
- `internal/tailnet/tailnet_test.go` — 12 tests: DiscoverPeers (3), ProbePeer (4), ProbeAll (2), Integration (1), Public wrappers (2)
- `internal/daemon/api_test.go` — TestHandleTailnetPeers with 3 sub-tests: cached peers, empty array, client method

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing Go test infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live Tailscale peer discovery | REM-01 | Requires real tailnet with multiple peers | Start AgentHub on 2+ tailnet machines, verify /tailnet/peers returns both |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-07

---

## Validation Audit 2026-04-07

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All 15 automated tests (12 tailnet + 3 daemon sub-tests) pass with `-race` flag. No gaps detected.
