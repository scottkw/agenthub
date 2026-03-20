---
phase: 9
slug: settings-modal-overhaul
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-19
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.1.0 |
| **Config file** | `frontend/vite.config.ts` (test: { environment: 'jsdom' }) |
| **Quick run command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test -- SettingsPanel` |
| **Full suite command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Estimated runtime** | ~1 second |

---

## Sampling Rate

- **After every task commit:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **After every plan wave:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | SETT-01 | unit | `pnpm test -- SettingsPanel` | ✅ | ✅ green |
| 09-01-02 | 01 | 1 | SETT-01 | unit | `pnpm test -- SettingsPanel` | ✅ | ✅ green |
| 09-01-03 | 01 | 1 | SETT-01 | unit | `pnpm test -- SettingsPanel` | ✅ | ✅ green |
| 09-01-04 | 01 | 1 | SETT-02 | unit | `pnpm test -- SettingsPanel` | ✅ | ✅ green |
| 09-01-05 | 01 | 1 | SETT-02 | unit | `pnpm test -- SettingsPanel` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tab visual styling matches design tokens | SETT-02 | CSS appearance cannot be verified in jsdom | Visual inspection: tab underline is accent blue (#7aa2f7), inactive text is dim (#565f89) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-03-20

---

## Validation Audit 2026-03-20

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

12 tests passing across SETT-01 (8 tab tests) and SETT-02 (4 styling/footer tests). All requirements fully covered by automated tests in `frontend/src/components/__tests__/SettingsPanel.test.tsx`.
