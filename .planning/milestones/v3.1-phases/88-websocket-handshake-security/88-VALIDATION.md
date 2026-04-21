---
phase: 88
slug: websocket-handshake-security
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-21
---

# Phase 88 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.22) |
| **Config file** | none — `go.mod` drives it |
| **Quick run command** | `go test ./internal/webserver/... ./internal/relay/... -run TestSecurity -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15s quick / ~60s full |

---

## Sampling Rate

- **After every task commit:** Run quick command scoped to the touched package
- **After every plan wave:** Run full suite
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

*Filled in by planner — every task row must map to an SC (1–4), REQ-ID (SEC-06), and threat ref (T-88-XX).*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 88-01-01 | 01 | 1 | SEC-06 | T-88-01 | pending — planner fills | pending | pending | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/origin_mw_test.go` — new test file for origin middleware unit tests
- [ ] `internal/webserver/regression_test.go` (or append to existing) — source-grep guard for `"*"` and `ws.BaseURL()` markers (D-13 items 1 & 3)
- [ ] `internal/relay/regression_test.go` — source-grep guard for `InsecureSkipVerify: true` absence (D-13 item 2)
- [ ] `internal/webserver/server_test.go` — new integration test `TestSecurity_WebSocketRejectsCrossSiteOrigin` (inverts the security-review scaffolding scenario)
- [ ] `internal/relay/server_test.go` — new integration test covering loopback-only allowlist behavior

*Existing `httptest` + `net/http/httptest.NewServer` patterns in `internal/webserver/` cover infrastructure — no new test framework install.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tailscale-mode terminal page completes upgrade (SC-2 half A) | SEC-06 | Needs live tailnet FQDN + magicsock — httptest cannot reproduce | 1) `agenthub` up on a tailnet node; 2) open share link on another node's browser; 3) confirm terminal attaches (WS 101); 4) devtools shows `Origin: https://<host>.<tailnet>.ts.net` accepted |
| Local-HTTPS-fallback terminal page completes upgrade (SC-2 half B) | SEC-06 | Needs self-signed cert loaded in browser — E2E harness out of scope for v3.1 | 1) disable tailnet; 2) open share link in browser on same LAN; 3) accept self-signed cert; 4) confirm terminal attaches; 5) devtools shows `Origin: https://<host-ip>:<port>` accepted |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags (no `go test -watch`; no `-count=0`)
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
