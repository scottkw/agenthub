---
phase: 82
slug: minimize-to-tray
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-16
audited: 2026-04-17
---

# Phase 82 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + vitest |
| **Config file** | vitest.config.ts (frontend), go test (backend) |
| **Quick run command** | `cd frontend && npx vitest run --reporter=verbose 2>&1 | tail -20` |
| **Full suite command** | `go test ./... && cd frontend && npx vitest run` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 82-01-01 | 01 | 1 | TRAY-01 | — | N/A | integration | `go test ./internal/daemon/... -run TestStartMinimized -v` | ✅ | ✅ green |
| 82-01-02 | 01 | 1 | TRAY-02 | — | N/A | integration | `go test ./internal/daemon/... -run "TestAPIGetStartMinimized\|TestAPISetStartMinimized\|TestAPISetStartMinimizedInvalidBody" -v` | ✅ | ✅ green |
| 82-01-03 | 01 | 1 | TRAY-03 | — | N/A | integration | `go test ./internal/daemon/... -run TestStartMinimizedPersistence -v` | ✅ | ✅ green |
| 82-02-01 | 02 | 2 | TRAY-01 | — | N/A | unit | `cd frontend && npx vitest run src/components/__tests__/SettingsTab.start-minimized.test.tsx --reporter=verbose` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements. Go test and vitest are already configured.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Window hidden on launch | TRAY-02 | Requires visual inspection of OS window state | Enable toggle, restart app, verify no window shown, only tray icon visible |
| Tray click opens window | TRAY-02 | Requires OS tray interaction | Click tray icon after hidden start, verify window appears |
| Setting survives restart | TRAY-03 | Requires full app restart cycle | Toggle on, quit app, relaunch, verify still enabled in settings |

---

## Nyquist Audit (2026-04-17)

Gaps filled by nyquist-auditor:

| Gap | Requirement | File Created / Modified | Tests | Status |
|-----|-------------|------------------------|-------|--------|
| TRAY-01 frontend | Settings tab start-minimized toggle | `frontend/src/components/__tests__/SettingsTab.start-minimized.test.tsx` (created) | 22 tests — bindings, state, mount, handler, JSX structure, CSS | green |
| TRAY-02 API | GET/PATCH /settings/start-minimized handlers | `internal/daemon/api_test.go` (appended 3 functions) | TestAPIGetStartMinimized, TestAPISetStartMinimized, TestAPISetStartMinimizedInvalidBody | green |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** 2026-04-17 — all gaps filled, all tests green
