---
phase: 11
slug: new-session-modal
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-19
validated: 2026-03-20
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (via `vitest/config`) |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Full suite command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Estimated runtime** | ~5 seconds |

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
| 11-01-01 | 01 | 0 | SESS-01, SESS-02, SESS-03, SESS-04 | source-inspection | `pnpm test -- NewSessionModal` | ✅ | ✅ green |
| 11-02-01 | 02 | 1 | SESS-03 | source-inspection | `pnpm test -- NewSessionModal` | ✅ | ✅ green |
| 11-02-02 | 02 | 1 | SESS-03, SESS-04 | source-inspection | `pnpm test -- NewSessionModal` | ✅ | ✅ green |
| 11-03-01 | 03 | 1 | SESS-01, SESS-02, SESS-03, SESS-04 | source-inspection | `pnpm test -- NewSessionModal` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `frontend/src/components/__tests__/NewSessionModal.test.tsx` — 13 tests covering SESS-01, SESS-02, SESS-03, SESS-04

*All tests follow the project's source-inspection pattern (import file as `?raw`, assert text presence).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Native OS folder dialog opens | SESS-03 | Wails `runtime.OpenDirectoryDialog` requires native OS interaction; cannot be triggered in jsdom/vitest | 1. Click `+` button 2. Click "Browse" in modal 3. Verify native OS folder picker opens |
| Folder dialog defaults to last-used directory | SESS-04 | Requires native dialog with `DefaultDirectory` which can't be verified in test environment | 1. Pick a folder 2. Close modal 3. Reopen and browse again 4. Verify dialog opens to previously selected folder |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved

---

## Validation Audit 2026-03-20

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
