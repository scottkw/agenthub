---
phase: 74
slug: multi-client-fan-out
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-14
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
| 74-01-01 | 01 | 1 | MC-01 | — | N/A | integration | `go test ./internal/relay/ -run TestHub_TwoClientsFanOut` | TBD | ⬜ pending |
| 74-01-02 | 01 | 1 | MC-02 | — | N/A | unit | `go test ./internal/relay/ -run TestHub_IndependentScrollback` | TBD | ⬜ pending |
| 74-01-03 | 01 | 1 | MC-03 | T-74-01 | ReadOnly client input discarded at hub | unit | `go test ./internal/relay/ -run TestHub_ReadOnlyClientInputDiscarded` | ❌ W0 | ⬜ pending |
| 74-01-04 | 01 | 1 | MC-04 | — | N/A | integration | `go test ./internal/daemon/ -run TestAPI_ListSessionsViewerCount` | ❌ W0 | ⬜ pending |
| 74-01-05 | 01 | 1 | MC-05 | — | N/A | unit | `go test ./internal/relay/ -run TestHub_ClientNameStoredOnSubscriber` | ❌ W0 | ⬜ pending |
| 74-01-06 | 01 | 1 | MC-06 | — | N/A | unit | `go test ./internal/relay/ -run TestHub_ResizeMaxWinsPolicy` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/relay/hub_test.go` — stubs for TestHub_ReadOnlyClientInputDiscarded, TestHub_ResizeMaxWinsPolicy, TestHub_ClientNameStoredOnSubscriber
- [ ] `internal/daemon/api_test.go` — stub for TestAPI_ListSessionsViewerCount

*Existing test infrastructure covers framework and fixtures.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Two browser tabs see live output | MC-01 | Browser WebSocket requires real browser | Open two browser tabs to same session, verify both show output |
| Scrollback independence across tabs | MC-02 | Browser scroll position is client-side | Scroll one tab up, verify other tab stays at bottom |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
