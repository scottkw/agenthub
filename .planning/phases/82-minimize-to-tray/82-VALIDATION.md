---
phase: 82
slug: minimize-to-tray
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-16
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
| 82-01-01 | 01 | 1 | TRAY-01 | — | N/A | integration | `go test ./internal/daemon/...` | ❌ W0 | ⬜ pending |
| 82-01-02 | 01 | 1 | TRAY-02 | — | N/A | integration | `go test ./...` | ❌ W0 | ⬜ pending |
| 82-01-03 | 01 | 1 | TRAY-03 | — | N/A | integration | `go test ./internal/daemon/...` | ❌ W0 | ⬜ pending |

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

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
