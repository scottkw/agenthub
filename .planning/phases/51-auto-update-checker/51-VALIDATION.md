---
phase: 51
slug: auto-update-checker
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-07
---

# Phase 51 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `testing` stdlib |
| **Framework (Frontend)** | vitest v4.1.0 |
| **Config file** | `frontend/vite.config.ts` |
| **Quick run command (Go)** | `go test ./internal/updater/... -v` |
| **Quick run command (Frontend)** | `pnpm --dir frontend test` |
| **Full suite command** | `go test ./... && pnpm --dir frontend test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/updater/... && pnpm --dir frontend test`
- **After every plan wave:** Run `go test ./... && pnpm --dir frontend test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 51-01-01 | 01 | 1 | UPD-01 | unit | `go test ./internal/updater/ -run TestCheck_DevVersionSkip` | W0 (Plan 01 creates) | pending |
| 51-01-02 | 01 | 1 | UPD-01 | unit | `go test ./internal/updater/ -run TestCheck_NewerVersionFound` | W0 (Plan 01 creates) | pending |
| 51-01-03 | 01 | 1 | UPD-01 | unit | `go test ./internal/updater/ -run TestCheck_AlreadyLatest` | W0 (Plan 01 creates) | pending |
| 51-01-04 | 01 | 1 | UPD-01 | unit | `go test ./internal/updater/ -run TestCheck_DetectError` | W0 (Plan 01 creates) | pending |
| 51-01-05 | 01 | 1 | UPD-01 | unit | `go test ./internal/updater/ -run TestCheck_RateLimit` | W0 (Plan 01 creates) | pending |
| 51-01-06 | 01 | 1 | UPD-01 | unit | `go test ./internal/updater/ -run TestCheck_RateLimitBypass` | W0 (Plan 01 creates) | pending |
| 51-02-01 | 02 | 2 | UPD-01 | build | `go build -tags dev -o /dev/null .` | existing | pending |
| 51-02-02 | 02 | 2 | UPD-04 | build+vet | `go vet ./...` | existing | pending |
| 51-03-01 | 03 | 2 | UPD-02 | unit | `pnpm --dir frontend test` (WelcomeTab.test.tsx) | W0 (Plan 03 extends) | pending |
| 51-03-02 | 03 | 2 | UPD-03 | unit | `pnpm --dir frontend test` (BrowserOpenURL test) | W0 (Plan 03 extends) | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [x] `internal/updater/updater.go` — package with injectable detectFunc (Plan 01 creates via TDD RED phase)
- [x] `internal/updater/updater_test.go` — unit tests for all UPD-01 behaviors (Plan 01 creates via TDD RED phase)
- [x] `frontend/src/components/__tests__/WelcomeTab.test.tsx` — extend with banner tests (file exists, Plan 03 adds new `describe` block)

*Existing Go test infrastructure and vitest setup cover framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Banner appears in Welcome tab with correct styling | UPD-02 | Visual layout verification | 1. Build app 2. Set mock update available 3. Open Welcome tab 4. Verify banner shows versions and Download button |
| Download button opens system browser | UPD-03 | Requires OS browser launch | 1. Click Download button 2. Verify system browser opens to GitHub releases page |
| Help > Check for Updates triggers check | UPD-04 | Native Wails menu interaction cannot be automated in unit tests | 1. Open Help menu 2. Click "Check for Updates" 3. Verify check runs (observe banner or logs) |

Note on UPD-04 test coverage: The `CheckForUpdates()` bound method is tested indirectly via `go build` (compilation verifies method exists and type-checks). The menu callback wiring (`checkForUpdatesCallback` -> `appInstance.CheckForUpdates()`) is native Wails menu integration that cannot be unit-tested without a running Wails app. This is accepted as a Manual-Only verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved
