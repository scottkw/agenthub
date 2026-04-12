---
phase: 66
slug: web-server-link-ux
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-11
audited: 2026-04-11
---

# Phase 66 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test / vitest |
| **Config file** | frontend/vitest.config.ts |
| **Quick run command** | `go test ./... -run TestGetWebServerQRCode -count=1` |
| **Full suite command** | `go test ./... -count=1 && cd frontend && pnpm test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -run TestGetWebServerQRCode -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 66-01-01 | 01 | 1 | WEB-01 | T-66-01 | N/A | source-inspection | `cd frontend && pnpm test -- --run -t "WEB-01"` | ✅ | ✅ green |
| 66-01-02 | 01 | 1 | WEB-02 | T-66-02 | N/A | source-inspection | `cd frontend && pnpm test -- --run -t "WEB-02"` | ✅ | ✅ green |
| 66-01-03 | 01 | 1 | WEB-03 | T-66-03 | N/A | source-inspection + integration | `cd frontend && pnpm test -- --run -t "WEB-03" && go test ./... -run TestGetWebServerQRCode -count=1` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Test File Inventory

| File | Framework | Tests | Requirements Covered |
|------|-----------|-------|---------------------|
| `frontend/src/components/__tests__/SettingsTab.web-link-ux.test.tsx` | vitest | 26 | WEB-01 (4), WEB-02 (6), WEB-03 (8), CSS (8) |
| `app_test.go` (`TestGetWebServerQRCode`, `TestGetWebServerQRCode_NoServer`) | go test | 2 | WEB-03 |

---

## Wave 0 Requirements

*No Wave 0 work needed — existing test infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Clicking URL opens default browser | WEB-01 | Wails `BrowserOpenURL` requires desktop runtime | Start app, enable web server, click Open button in Settings tab |
| Copy button writes to clipboard | WEB-02 | Wails `ClipboardSetText` requires desktop runtime | Start app, enable web server, click Copy button, paste elsewhere |
| QR code is scannable | WEB-03 | Requires mobile device camera | Start app, enable web server, scan displayed QR with phone |
| All actions work in Tailscale mode | WEB-01,02,03 | Requires Tailscale network connection | Enable Tailscale, start web server, verify URL/copy/QR use Tailscale address |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-04-11

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

**Notes:** All three requirements (WEB-01, WEB-02, WEB-03) have automated source-inspection tests covering implementation patterns. WEB-03 additionally has Go integration tests verifying PNG QR code generation end-to-end. All 28 tests pass green.
