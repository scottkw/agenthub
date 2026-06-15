---
phase: 123
slug: td-cleanup-write-sandbox-primitives-daemon-routes
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-14
---

# Phase 123 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib testing + go fuzz) |
| **Config file** | none — Go module native |
| **Quick run command** | `go test ./internal/files/... ./internal/daemon/...` |
| **Full suite command** | `go test -race ./internal/files/... ./internal/daemon/...` |
| **Estimated runtime** | ~30-90 seconds (fuzz adds 60s when `-fuzz` enabled) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/files/... ./internal/daemon/...`
- **After every plan wave:** Run `go test -race ./internal/files/... ./internal/daemon/...`
- **Before `/gsd:verify-work`:** Full suite must be green; `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/...` reports zero crashes
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (planner fills) | — | — | FSW-01..12 | — | sandbox write traversal rejected | fuzz/unit | `go test -race ./internal/files/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/files/sandbox_write_test.go` — write-primitive unit tests (write, rename, delete, mkdir, upload)
- [ ] `internal/files/fuzz_write_test.go` — `FuzzSandboxWrite` harness mirroring `FuzzSandboxPath`
- [ ] `internal/daemon/*_write_test.go` — daemon write-route handler tests

*Framework already present (go test) — no install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| curl PUT over real `~/.agenthub/daemon.sock` | FSW (daemon route) | requires a live daemon + real session | Start daemon, create session, run the success-criterion curl, confirm 200 + read-back |

*Most behaviors have automated verification; the live-socket curl is the documented manual smoke.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
