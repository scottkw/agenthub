---
phase: 66
slug: web-server-link-ux
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-11
---

# Phase 66 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test / vitest |
| **Config file** | none — existing test infrastructure |
| **Quick run command** | `go test ./... -run TestWebServer -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -run TestWebServer -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 66-01-01 | 01 | 1 | WEB-01 | — | N/A | integration | `go test ./... -run TestGetWebServerQRCode -count=1` | ❌ W0 | ⬜ pending |
| 66-01-02 | 01 | 1 | WEB-02 | — | N/A | manual | Browser open verified via Wails runtime | ❌ W0 | ⬜ pending |
| 66-01-03 | 01 | 1 | WEB-03 | — | N/A | manual | Clipboard copy verified via Wails runtime | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Clicking URL opens default browser | WEB-01 | Wails `BrowserOpenURL` requires desktop runtime | Start app, enable web server, click URL in Settings tab |
| Copy button writes to clipboard | WEB-02 | Wails `ClipboardSetText` requires desktop runtime | Start app, enable web server, click copy button, paste elsewhere |
| QR code is scannable | WEB-03 | Requires mobile device camera | Start app, enable web server, scan displayed QR with phone |
| All actions work in Tailscale mode | WEB-01,02,03 | Requires Tailscale network connection | Enable Tailscale, start web server, verify URL/copy/QR use Tailscale address |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
