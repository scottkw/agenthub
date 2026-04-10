---
phase: 61
slug: serve02-frontend-fix
status: complete
nyquist_compliant: true
wave_0_complete: true
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
| 61-01-01 | 01 | 1 | SERVE-02 | — | N/A | integration | `go build -tags wailsassets ./...` | ✅ | ✅ green |
| 61-01-02 | 01 | 1 | SERVE-02 | — | N/A | type-check | `cd frontend && npx tsc --noEmit` | ✅ | ✅ green |
| 61-01-03 | 01 | 1 | SERVE-02 | — | N/A | manual | StatusBar visual verification | N/A | ✅ green |
| 61-01-03-auto | 01 | 1 | SERVE-02 | — | N/A | unit (source-string) | `npx vitest run src/components/__tests__/App.serve02.test.tsx` | ✅ | ✅ green |

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved

---

## Validation Audit 2026-04-10

| Metric | Count |
|--------|-------|
| Gaps found | 1 |
| Resolved | 1 |
| Escalated | 0 |

Tests added: `frontend/src/components/__tests__/App.serve02.test.tsx` (18 tests covering webEnabled seeding in init, createTab, retryInit).
