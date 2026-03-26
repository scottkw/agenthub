---
phase: 35
slug: terminal-fill-fix-v2
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 35 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 |
| **Config file** | `frontend/vite.config.ts` |
| **Quick run command** | `cd frontend && npx vitest run` |
| **Full suite command** | `cd frontend && npx vitest run && cd .. && go test ./internal/daemon/... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run`
- **After every plan wave:** Run `cd frontend && npx vitest run && cd .. && go test ./internal/daemon/... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 35-01-01 | 01 | 1 | FILL-01..04 | unit (source inspection) | `cd frontend && npx vitest run` | ✅ extend TerminalPanel.test.tsx | ⬜ pending |
| 35-01-02 | 01 | 1 | FILL-01..04 | unit (source inspection) | `cd frontend && npx vitest run` | ✅ extend TerminalPanel.test.tsx | ⬜ pending |
| 35-01-03 | 01 | 1 | FILL-05 | unit (source inspection) | `cd frontend && npx vitest run` | ✅ existing test | ⬜ pending |
| 35-01-04 | 01 | 1 | FILL-06 | manual (prod binary) | `wails build -tags wailsassets` | ❌ manual only | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — add FILL-01..06 source inspection tests:
  - `tryFit` function present in isActive effect
  - `proposeDimensions()` call present
  - `MAX_ATTEMPTS` constant present
  - `cancelled` flag still present
  - `cancelAnimationFrame(rafId)` present in cleanup
  - `[isActive]` dependency array unchanged

*Existing infrastructure covers framework setup; only new test cases needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Fix works in production `wails build` binary | FILL-06 | WebView timing differs from Vite dev server; requires real binary | 1. `wails build -tags wailsassets` 2. Launch binary 3. Open each CLI tab 4. Verify terminal fills full width immediately |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
