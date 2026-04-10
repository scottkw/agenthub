---
phase: 61
slug: serve02-frontend-fix
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-09
---

# Phase 61 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend), go test (backend) |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `cd frontend && npx vitest run --reporter=verbose 2>&1 \| head -80` |
| **Full suite command** | `cd frontend && npx vitest run && cd .. && go test ./...` |
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
| 61-01-01 | 01 | 1 | SERVE-02 | — | N/A | integration | `go build -tags wailsassets ./...` | ✅ | ⬜ pending |
| 61-01-02 | 01 | 1 | SERVE-02 | — | N/A | type-check | `cd frontend && npx tsc --noEmit` | ✅ | ⬜ pending |
| 61-01-03 | 01 | 1 | SERVE-02 | — | N/A | manual | StatusBar visual verification | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| StatusBar shows correct web toggle state | SERVE-02 | UI state requires visual confirmation | Create session with web server running, verify toggle shows enabled; restore app, verify toggle state persists |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
