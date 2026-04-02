---
phase: 41
slug: system-tray-lifecycle
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-02
---

# Phase 41 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./... -run TestTray -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~6 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 6 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 41-01-01 | 01 | 1 | BRND-03 | unit | `go test . -run TestTrayIconAsset -v` | ❌ W0 | ⬜ pending |
| 41-01-02 | 01 | 1 | TRAY-05 | manual | Inspect `build/darwin/Info.plist` | N/A | ⬜ pending |
| 41-01-03 | 01 | 1 | DMGR-02 | unit | `go test ./internal/daemon/... -run TestShutdownDaemon -v` | ❌ W0 | ⬜ pending |
| 41-01-04 | 01 | 1 | DMGR-02 | unit | `go test . -run TestTrayQuitShutdownsDaemon -v` | ❌ W0 | ⬜ pending |
| 41-01-05 | 01 | 1 | TRAY-02, TRAY-04 | unit | `go test . -run TestUpdateTray -v` | ❌ W0 | ⬜ pending |
| 41-01-06 | 01 | 1 | TRAY-03, TRAY-06 | unit | `go test . -run TestUpdateTray -v` | ❌ W0 | ⬜ pending |
| 41-01-07 | 01 | 1 | DMGR-01 | unit | `go test . -run TestBeforeCloseReturnsTrue -v` | ✅ tray_test.go | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tray_test.go` — add `TestUpdateTray`, `TestTrayQuitShutdownsDaemon`, `TestTrayIconAsset`
- [ ] `internal/daemon/client_test.go` — add `TestShutdownDaemon`

*Existing infrastructure covers DMGR-01 (`TestBeforeCloseReturnsTrue`, `TestHideWindowSessionsAlive`).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| AgentHub icon visible in macOS menu bar | TRAY-01 | Requires display server + NSStatusItem | Run `wails build`, launch app, verify icon in menu bar |
| No Dock icon with LSUIElement | TRAY-05 | Requires production build + macOS UI | Run `wails build`, launch app, check Dock and Cmd+Tab |
| Light/dark menu bar adaptation | BRND-03 | Visual verification | Toggle macOS appearance, verify icon visibility |
| Tray menu opens with correct items | TRAY-02 | Requires NSMenu + display | Right-click tray icon, verify "Open AgentHub", sessions, "Quit" |
| Session click focuses GUI tab | TRAY-04 | End-to-end integration | Click session name in tray menu, verify tab switches |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 6s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
