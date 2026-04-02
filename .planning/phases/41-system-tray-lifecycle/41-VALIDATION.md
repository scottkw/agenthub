---
phase: 41
slug: system-tray-lifecycle
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-02
audited: 2026-04-02
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
| 41-01-01 | 01 | 1 | DMGR-01 | unit | `go test . -run TestBeforeCloseReturnsTrue -v` | ✅ tray_test.go | ✅ green |
| 41-01-02 | 01 | 1 | DMGR-01 | unit | `go test . -run TestHideWindowSessionsAlive -v` | ✅ tray_test.go | ✅ green |
| 41-01-03 | 01 | 1 | DMGR-02 | unit | `go test ./internal/daemon/... -run TestShutdownDaemon -v` | ✅ client_test.go | ✅ green |
| 41-01-04 | 01 | 1 | DMGR-02 | unit | `go test . -run TestTrayQuitNilClient -v` | ✅ tray_test.go | ✅ green |
| 41-01-05 | 01 | 1 | TRAY-03, TRAY-06 | unit | `go test . -run "TestTrayTooltip\|TestRefreshTrayStateNilClient" -v` | ✅ tray_test.go | ✅ green |
| 41-01-06 | 01 | 1 | TRAY-05 | manual | Inspect `build/darwin/Info.plist` | manual-only | ✅ verified |
| 41-01-07 | 01 | 1 | BRND-03 | unit | `go test . -run TestTrayIconAsset -v` | ✅ tray_test.go | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `tray_test.go` — `TestTrayIconAsset`, `TestTrayTooltip`, `TestTrayQuitNilClient`, `TestRefreshTrayStateNilClient` all present and green
- [x] `internal/daemon/client_test.go` — `TestShutdownDaemon` present and green

*All Wave 0 gaps resolved during execution.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Info.plist contains LSUIElement=true | TRAY-05 | Requires full `wails build` production build and visual inspection (no Dock icon) | 1. Run `wails build` 2. Inspect built Info.plist for `<key>LSUIElement</key><true/>` 3. Launch app — verify no Dock icon |

*NSStatusItem calls are untestable in unit tests — Cocoa requires a display server. Tests verify Go-side behavior through testable wrapper functions.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-04-02

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All 6 automated tests verified green. Wave 0 items were filled during phase execution (Plan 01 and Plan 02). No additional test generation needed.
