---
phase: 42
slug: tray-startup-failure-error-icon
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-02
audited: 2026-04-03
---

# Phase 42 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package |
| **Config file** | none (standard `go test`) |
| **Quick run command** | `go test . -run "TestRefreshTray" -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test . -run "TestRefreshTray" -v`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 42-01-01 | 01 | 1 | TRAY-03 | unit | `go test . -run TestRefreshTrayStateStartupFailure -v` | ✅ | ✅ green |
| 42-01-02 | 01 | 1 | TRAY-03 | unit | `go test . -run TestRefreshTrayStateNilClient -v` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `tray_test.go` — add `TestRefreshTrayStateStartupFailure` covering `{trayInit: true, client: nil}` path

*Existing `TestRefreshTrayStateNilClient` covers the `trayInit: false` path — keep as-is.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual error icon displayed in macOS menu bar | TRAY-03 | Requires macOS GUI rendering | Build app, kill daemon, launch app, observe tray icon shows error state |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-04-03

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
