---
phase: 75
slug: cli-status-bar
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-14
---

# Phase 75 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test toolchain |
| **Quick run command** | `go test ./internal/cli/statusbar/...` |
| **Full suite command** | `go test ./internal/cli/... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/cli/statusbar/...`
- **After every plan wave:** Run `go test ./internal/cli/... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 75-01-01 | 01 | 1 | SB-01 | — | N/A | unit | `go test ./internal/cli/statusbar/ -run TestBarRender` | ❌ W0 | ⬜ pending |
| 75-01-02 | 01 | 1 | SB-02 | — | N/A | unit | `go test ./internal/cli/statusbar/ -run TestScrollRegion` | ❌ W0 | ⬜ pending |
| 75-02-01 | 02 | 1 | SB-03 | — | N/A | unit | `go test ./internal/cli/statusbar/ -run TestTTYDetection` | ❌ W0 | ⬜ pending |
| 75-02-02 | 02 | 1 | SB-04 | — | N/A | unit | `go test ./internal/cli/statusbar/ -run TestViewerCount` | ❌ W0 | ⬜ pending |
| 75-03-01 | 03 | 2 | SB-05 | — | N/A | integration | `go test ./internal/cli/... -run TestAttachStatusBar` | ❌ W0 | ⬜ pending |
| 75-03-02 | 03 | 2 | SB-06 | — | N/A | unit | `go test ./internal/cli/statusbar/ -run TestCleanup` | ❌ W0 | ⬜ pending |
| 75-03-03 | 03 | 2 | SB-07 | — | N/A | unit | `go test ./internal/cli/statusbar/ -run TestSignalHandling` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/cli/statusbar/bar_test.go` — stubs for SB-01, SB-02, SB-06
- [ ] `internal/cli/statusbar/tty_test.go` — stubs for SB-03
- [ ] `internal/cli/statusbar/viewer_test.go` — stubs for SB-04
- [ ] `internal/cli/statusbar/integration_test.go` — stubs for SB-05, SB-07

*Existing Go test infrastructure covers framework needs — no new tooling required.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual scroll region correctness | SB-02 | Terminal rendering can only be verified visually | 1. Run `agenthub attach <session>` 2. Generate output exceeding terminal height 3. Verify status bar stays fixed at bottom, output scrolls normally |
| Clean terminal restore on detach | SB-06 | Terminal state restoration requires visual inspection | 1. Attach to session 2. Press Ctrl-\\ 3. Verify no leftover bar artifacts, cursor at correct position |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
