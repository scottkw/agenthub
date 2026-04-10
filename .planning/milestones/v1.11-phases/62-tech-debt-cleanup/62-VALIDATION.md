---
phase: 62
slug: tech-debt-cleanup
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-10
---

# Phase 62 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 |
| **Config file** | `frontend/vite.config.ts` |
| **Quick run command** | `cd frontend && npx vitest run` |
| **Full suite command** | `cd frontend && npx vitest run` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run`
- **After every plan wave:** Run `cd frontend && npx vitest run`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 2 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 62-01-01 | 01 | 1 | T1: SettingsPanel.tsx deleted | T-62-01 / accept | N/A — dead code deletion | shell | `test ! -f frontend/src/components/SettingsPanel.tsx` | ✅ | ✅ green |
| 62-01-01 | 01 | 1 | T1: HealthModal.tsx deleted | T-62-01 / accept | N/A — dead code deletion | shell | `test ! -f frontend/src/components/HealthModal.tsx` | ✅ | ✅ green |
| 62-01-01 | 01 | 1 | T1: HealthModal.test.tsx deleted | T-62-01 / accept | N/A — dead code deletion | shell | `test ! -f frontend/src/components/__tests__/HealthModal.test.tsx` | ✅ | ✅ green |
| 62-01-01 | 01 | 1 | T1: health-modal CSS removed | — | N/A | shell | `! grep -q 'health-modal' src/style.css` | ✅ | ✅ green |
| 62-01-01 | 01 | 1 | T1: settings-overlay CSS removed | — | N/A | shell | `! grep -q 'settings-overlay' src/style.css` | ✅ | ✅ green |
| 62-01-02 | 01 | 1 | T3: All vitest tests pass | — | N/A | unit | `cd frontend && npx vitest run` | ✅ | ✅ green |
| 62-01-02 | 01 | 1 | T4: NAV-04 label says "New Session" | — | N/A | source-inspection | `grep 'New Session sidebar button' frontend/src/components/__tests__/App.nav.test.tsx` | ✅ | ✅ green |
| 62-01-02 | 01 | 1 | T5: NET-01..NET-05 marked [x] | — | N/A | shell | `grep '\[x\].*NET-0[1-5]' .planning/REQUIREMENTS.md \| wc -l` (expect 5) | ✅ | ✅ green |
| 62-01-02 | 01 | 1 | A1: App.test.tsx source-inspection | — | N/A | unit | `cd frontend && npx vitest run src/components/__tests__/App.test.tsx` | ✅ | ✅ green |
| 62-01-02 | 01 | 1 | A2: App.nav.test.tsx navigation wiring | — | N/A | unit | `cd frontend && npx vitest run src/components/__tests__/App.nav.test.tsx` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 2s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-10
