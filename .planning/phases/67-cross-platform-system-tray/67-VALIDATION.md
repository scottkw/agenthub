---
phase: 67
slug: cross-platform-system-tray
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-11
---

# Phase 67 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — uses Go standard testing |
| **Quick run command** | `go test -run TestTray -count=1 ./...` |
| **Full suite command** | `go test -count=1 ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -run TestTray -count=1 ./...`
- **After every plan wave:** Run `go test -count=1 ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 67-01-01 | 01 | 1 | TRAY-01, TRAY-02 | — | N/A | unit | `go test -run TestTrayIcon -v ./...` | ❌ W0 | ⬜ pending |
| 67-01-02 | 01 | 1 | TRAY-03 | — | N/A | unit | `go test -run TestTrayMenu -v ./...` | ❌ W0 | ⬜ pending |
| 67-01-03 | 01 | 1 | TRAY-04 | — | N/A | unit | `go test -run TestTrayTooltip -v ./...` | ✅ | ⬜ pending |
| 67-02-01 | 02 | 1 | TRAY-05 | — | N/A | unit | `go test -run TestBeforeClose -v ./...` | ✅ | ⬜ pending |
| 67-02-02 | 02 | 1 | TRAY-06 | — | N/A | unit | `go test -run TestTrayQuit -v ./...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tray_common_test.go` — tests for shared tray logic (tooltip, menu data, icon conversion)
- [ ] Platform-specific test stubs — build-tagged tests for Linux D-Bus and Windows Win32

*Existing tray_test.go (darwin) covers macOS tests and serves as pattern reference.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Linux tray icon visible in GNOME/KDE/XFCE | TRAY-01 | Requires display server + desktop environment | Launch app on Linux with supported DE, verify icon appears in system tray |
| Windows tray icon visible in notification area | TRAY-02 | Requires Windows display server | Launch app on Windows, verify icon appears in notification area |
| Tray icon state changes on daemon disconnect | TRAY-04 | Requires running daemon + visual verification | Stop daemon while app is running, verify icon changes to error state |
| Hide-on-close keeps tray icon | TRAY-05 | Requires window manager interaction | Close app window, verify tray icon remains and sessions are alive |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
