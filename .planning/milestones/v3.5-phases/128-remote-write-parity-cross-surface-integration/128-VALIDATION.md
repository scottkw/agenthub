---
phase: 128
slug: remote-write-parity-cross-surface-integration
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-15
---

# Phase 128 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (daemon remote proxy + RemoteFilesClient via httptest.TLSServer; 3-observer parity harness) + vitest (FilesApiError 405/401) + Playwright (HTTPS browser observer) |
| **Quick run command** | `go test ./internal/daemon/... ./internal/tui/...` |
| **Full suite command** | `go test -race ./internal/... && (cd frontend && pnpm test)` + Playwright remote scenarios |
| **Estimated runtime** | ~60-90s + e2e |

## Sampling Rate

- After every task commit: relevant package test
- After every wave: `go test -race ./internal/...`
- Before verify: 3-observer parity green; Phase 122 remote-read suite zero regressions; two-machine UAT checklist committed
- Max latency: 90s

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Secure Behavior | Test | Command | Status |
|---------|------|------|-------------|-----------------|------|---------|--------|
| (planner) | 128-01 | 1 | RMW-04 | v3.4 peer 405 → friendly msg (Go+TS) | unit | `go test ./internal/tui/...` + `pnpm test` | ⬜ |
| (planner) | 128-02 | 1 | RMW-05 | cap-expiry → buffer preserved + "access expired"; upload abort cleans queue | unit | `go test ./internal/...` + `pnpm test` | ⬜ |
| (planner) | 128-03 | 2 | RMW-01/02/03 | 3-observer write-parity byte-equivalent | unit/e2e | `go test ./internal/daemon/...` + playwright | ⬜ |
| (planner) | 128-04 | 2 | RMW-06 | Phase 122 read regression zero; UAT checklist committed | regression/doc | `go test ./internal/daemon/ -run Remote` | ⬜ |

## Wave 0 Requirements

- [ ] 405 version-gate test (v3.4 peer fixture returns 405 → friendly message) — Go + TS
- [ ] cap-expiry 401 "access expired" test + upload-abort queue cleanup
- [ ] 3-observer write-parity harness (daemon-proxy Go + RemoteFilesClient Go + Playwright HTTPS), backed by a real Sandbox/t.TempDir() peer fixture
- [ ] Phase 122 remote-read regression run
- [ ] Two-machine UAT checklist committed (operator-deferred)

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Instructions |
|----------|-------------|------------|--------------|
| Two-machine tailnet write UAT | RMW-06 | requires 2 physical tailnet machines | Machine A web-share + Machine B GUI + Machine B TUI; cross-surface write parity + cap-expiry; closes Issue #24 |

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 deps
- [ ] Wave 0 covers MISSING refs
- [ ] `nyquist_compliant: true`

**Approval:** pending
