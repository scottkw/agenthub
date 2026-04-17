---
phase: 80
slug: tailscale-detection
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-16
---

# Phase 80 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + vitest 4.1.0 |
| **Config file** | frontend/vitest.config.ts (frontend); none — Go test is built-in (backend) |
| **Quick run command** | `go test ./internal/webserver/... && cd frontend && npx vitest run` |
| **Full suite command** | `go test ./... && cd frontend && npx vitest run` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/... ./internal/pty/...`
- **After every plan wave:** Run `go test ./...` + `cd frontend && npx vitest run`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 80-01-01 | 01 | 1 | TS-01 | T-80-01, T-80-04 | Custom path stat-only, no exec | unit | `go test ./internal/webserver/ -run TestDetectTailscaleBinary` | ✅ | ✅ green |
| 80-01-02 | 01 | 1 | TS-01, TS-02 | T-80-03 | 5s context timeout on daemon probe | unit | `go test ./internal/webserver/ -run "TestCheckHealth_(BackendState\|FullyHealthy\|DaemonStopped\|CustomPath\|BinaryNotFound)"` | ✅ | ✅ green |
| 80-01-03 | 01 | 1 | TS-02 | — | N/A | unit | `go test ./internal/webserver/ -run TestCheckHealth_NotRunning` | ✅ | ✅ green |
| 80-02-01 | 02 | 2 | TS-01, TS-02 | T-80-06 | Wails IPC only, no remote exposure | source | `cd frontend && npx vitest run src/components/__tests__/App.test.tsx` | ✅ | ✅ green |
| 80-02-02 | 02 | 2 | TS-01, TS-02 | T-80-05 | platformHint is local-only | source | `cd frontend && npx vitest run src/components/__tests__/SettingsTab.test.tsx` | ✅ | ✅ green |
| 80-02-03 | 02 | 2 | TS-01, TS-02 | T-80-06 | No action buttons in daemon-stopped (D-06) | unit | `cd frontend && npx vitest run src/components/__tests__/LocalNetworkBanner.test.tsx` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/webserver/tailscale_test.go` — extended with 4-state health check tests (11 test functions)
- [x] `frontend/src/components/__tests__/SettingsTab.test.tsx` — 13 tests in TS-01/TS-02 block
- [x] `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` — 3 daemon-stopped tests added
- [x] `frontend/src/components/__tests__/App.test.tsx` — 5 source-inspection tests for new props

*Existing infrastructure covers test framework requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tailscale status displays correctly in Settings UI | TS-02 | Requires visual browser verification | Open Settings > Web Server; verify dot color and text match state |
| Platform-specific instructions render correctly | TS-02 | Platform-dependent UI text | Check "Show diagnostics" checklist on each platform |
| Tailscale path override works via Browse button | TS-01 | Requires file dialog interaction | Settings > Paths > Browse for tailscale binary |

---

## Test Coverage Detail

### Backend (Go) — 11 test functions in tailscale_test.go

| Test | Requirement | Covers |
|------|-------------|--------|
| TestCheckHealth_NotRunning | TS-02 | Daemon error returns correct state |
| TestCheckHealth_BackendState (4 subtests) | TS-02 | All BackendState values mapped correctly |
| TestCheckHealth_CertDomains (2 subtests) | TS-02 | Cert domain detection |
| TestCheckHealth_FullyHealthy | TS-01, TS-02 | Full cascade passes |
| TestCheckHealth_BinaryNotFound | TS-01 | Not-found path (SKIP on dev machines with tailscale) |
| TestCheckHealth_DaemonStopped | TS-01, TS-02 | Binary found, daemon unreachable |
| TestCheckHealth_CustomPathPrecedence | TS-01 | Custom path overrides auto-detect |
| TestDetectTailscaleBinary_CustomPath | TS-01 | Custom path returns correctly |
| TestDetectTailscaleBinary_InvalidCustomPath | TS-01 | Invalid custom falls through |
| TestDetectTailscaleBinary_Empty | TS-01 | Empty path uses auto-detect |

### Frontend (Vitest) — 21 tests across 3 files

**SettingsTab.test.tsx** (13 in TS-01/TS-02 block):
- binaryFound/daemonUp/platformHint type fields
- Daemon Stopped status text
- Show diagnostics collapsible
- 4 checklist steps (Binary detected, Daemon running, Connected, TLS certs)
- Platform instructions (macOS, Linux, Windows)
- Auto-detect placeholder hint

**LocalNetworkBanner.test.tsx** (3 new):
- Daemon-stopped message + macOS hint + no buttons (D-06)
- Linux daemon instruction
- Windows daemon instruction

**App.test.tsx** (5 new):
- tailscaleBinaryFound, tailscaleDaemonUp, platformHint props passed
- binaryFound, daemonUp in state type

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-17

---

## Validation Audit 2026-04-17

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

**Notes:**
- Original VALIDATION.md was a draft with incomplete task map (4 of 6 tasks) and pending statuses
- All 6 tasks now mapped with correct automated commands and ✅ green status
- 11 Go tests + 21 frontend tests provide full requirement coverage for TS-01 and TS-02
- TestCheckHealth_BinaryNotFound SKIP on dev machines (tailscale installed); cascade tested indirectly by DaemonStopped
