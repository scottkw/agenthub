---
phase: 74
slug: multi-client-fan-out
status: audited
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-14
audited: 2026-04-14
---

# Phase 74 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing test infra |
| **Quick run command** | `go test ./internal/relay/... ./internal/daemon/... -count=1 -run 'TestHub_|TestAPI_'` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/relay/... ./internal/daemon/... -count=1 -run 'TestHub_|TestAPI_'`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 74-01-01 | 01 | 1 | MC-01 | — | N/A | integration | `go test ./internal/relay/ -run TestHub_TwoClientsFanOut` | ✅ | ✅ green |
| 74-01-02 | 01 | 1 | MC-02 | — | N/A | integration | `go test ./internal/relay/ -run TestHub_ReconnectScrollback` | ✅ | ✅ green |
| 74-01-03 | 01 | 1 | MC-03 | T-74-01 | ReadOnly client input discarded at server | integration | `go test ./internal/relay/ -run TestServer_ReadOnlyClientInputDiscarded` | ✅ | ✅ green |
| 74-01-04 | 01 | 1 | MC-04 | — | N/A | integration | `go test ./internal/daemon/ -run TestAPI_ListSessionsViewerCount` | ✅ | ✅ green |
| 74-01-05 | 01 | 1 | MC-05 | — | N/A | unit+integration | `go test ./internal/relay/ -run 'TestHub_ClientNameStored\|TestServer_ClientNameQueryParam'` | ✅ | ✅ green |
| 74-01-06 | 01 | 1 | MC-06 | — | N/A | unit | `go test ./internal/relay/ -run 'TestHub_ResizeMaxWins\|TestHub_ResizeClient'` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/relay/hub_test.go` — TestHub_ReadOnlyFlagStored, TestHub_ResizeMaxWinsPolicy, TestHub_ClientNameStored, TestHub_ResizeClientNoOpWhenDimensionsUnchanged, TestHub_ResizeClientUnsubscribeDoesNotShrink
- [x] `internal/relay/server_test.go` — TestServer_ReadOnlyClientInputDiscarded, TestServer_ReadOnlyClientReceivesOutput, TestServer_ClientNameQueryParam
- [x] `internal/daemon/api_test.go` — TestAPI_ListSessionsViewerCount

*All Wave 0 tests implemented and passing.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Two browser tabs see live output | MC-01 | Browser WebSocket requires real browser | Open two browser tabs to same session, verify both show output |
| Scrollback independence across tabs | MC-02 | Browser scroll position is client-side | Scroll one tab up, verify other tab stays at bottom |

---

## Validation Audit 2026-04-14

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All 6 requirements have automated test coverage. Test name corrections applied (VALIDATION.md draft had placeholder names that differed from actual implementations). All tests green under `go test ./internal/relay/... ./internal/daemon/... -count=1 -timeout 30s -short`.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-14
