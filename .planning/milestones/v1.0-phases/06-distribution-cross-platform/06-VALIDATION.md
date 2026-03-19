---
phase: 6
slug: distribution-cross-platform
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-18
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) |
| **Config file** | none — `go test ./...` discovers all `*_test.go` files |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | PLAT-01, PLAT-02, PLAT-03 | build smoke | CI matrix (`.github/workflows/build.yml`) | ❌ W0 | ⬜ pending |
| 06-01-02 | 01 | 1 | PLAT-02 | build smoke | CI `ubuntu-22.04` + `ubuntu-latest` with `-tags webkit2_41` | ❌ W0 | ⬜ pending |
| 06-02-01 | 02 | 1 | PLAT-01 | manual | `spctl --assess --type exec build/bin/agenthub.app` | Manual | ⬜ pending |
| 06-03-01 | 03 | 1 | PLAT-03 | unit | `go test ./internal/pty/... -v` (windows) | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `.github/workflows/build.yml` — CI matrix for macOS, Linux 22.04, Linux 24.04, Windows
- [ ] `tray_linux.go` — `//go:build linux` stub for `initTray`/`cleanupTray`
- [ ] `tray_windows.go` — `//go:build windows` stub for `initTray`/`cleanupTray`
- [ ] `build/darwin/Info.plist` — production plist with real `CFBundleIdentifier`
- [ ] `build/entitlements.plist` — hardened runtime entitlements for notarization

*These must be created before any cross-platform build can succeed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| macOS Gatekeeper passes signed .app | PLAT-01 | Requires actual macOS + Apple Developer cert | Run `spctl --assess --type exec` on built .app |
| Windows installer launches app correctly | PLAT-03 | Requires Windows machine with PTY | Install via NSIS, launch, create session, verify keyboard |
| Linux app runs on Ubuntu 22.04 | PLAT-02 | Requires Ubuntu 22.04 environment | Run binary, create session, verify terminal |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
