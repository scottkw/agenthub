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
| **Config file** | none (standard `go test`) |
| **Quick run command** | `go test ./... -short` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~6 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -short`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 41-01-01 | 01 | 1 | DMGR-01 | unit | `go test . -run TestBeforeCloseReturnsTrue -v` | ✅ tray_test.go | ⬜ pending |
| 41-01-02 | 01 | 1 | DMGR-01 | unit | `go test . -run TestHideWindowSessionsAlive -v` | ✅ tray_test.go | ⬜ pending |
| 41-01-03 | 01 | 1 | DMGR-02 | unit | `go test ./internal/daemon/... -run TestShutdownDaemon -v` | ❌ W0 | ⬜ pending |
| 41-01-04 | 01 | 1 | DMGR-02 | unit | `go test . -run TestTrayQuitShutdownsDaemon -v` | ❌ W0 | ⬜ pending |
| 41-01-05 | 01 | 1 | TRAY-03, TRAY-06 | unit | `go test . -run TestUpdateTray -v` | ❌ W0 | ⬜ pending |
| 41-01-06 | 01 | 1 | TRAY-05 | manual | Inspect `build/darwin/Info.plist` | manual-only | ⬜ pending |
| 41-01-07 | 01 | 1 | BRND-03 | unit | `go test . -run TestTrayIconAsset -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tray_test.go` — add `TestUpdateTray`, `TestTrayQuitShutdownsDaemon`, `TestTrayIconAsset`
- [ ] `internal/daemon/client_test.go` — add `TestShutdownDaemon`

*Existing infrastructure covers DMGR-01 (tests already exist).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Info.plist contains LSUIElement=true | TRAY-05 | Requires full `wails build` production build and visual inspection (no Dock icon) | 1. Run `wails build` 2. Inspect built Info.plist for `<key>LSUIElement</key><true/>` 3. Launch app — verify no Dock icon |

*NSStatusItem calls are untestable in unit tests — Cocoa requires a display server. Tests verify Go-side behavior through testable wrapper functions.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
