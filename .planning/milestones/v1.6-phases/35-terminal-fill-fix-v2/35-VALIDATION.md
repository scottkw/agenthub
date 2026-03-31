---
phase: 35
slug: terminal-fill-fix-v2
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-26
audited: 2026-03-31
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
| **Estimated runtime** | ~3 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run`
- **After every plan wave:** Run `cd frontend && npx vitest run && cd .. && go test ./internal/daemon/... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 3 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 35-01-01 | 01 | 1 | FILL-01..04 | unit (source inspection) | `cd frontend && npx vitest run` | ✅ TerminalPanel.test.tsx | ✅ green |
| 35-01-02 | 01 | 1 | FILL-01..04 | unit (source inspection) | `cd frontend && npx vitest run` | ✅ TerminalPanel.test.tsx | ✅ green |
| 35-01-03 | 01 | 1 | FILL-05 | unit (source inspection) | `cd frontend && npx vitest run` | ✅ TerminalPanel.test.tsx | ✅ green |
| 35-01-04 | 01 | 1 | FILL-06 | manual (prod binary) | `wails build -tags wailsassets` | ❌ manual only | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements.*

---

## Requirement Coverage Detail

| Requirement | Test Description | Test File | Status |
|-------------|-----------------|-----------|--------|
| FILL-01 | `MAX_ATTEMPTS = 20`, `tryFit`, `proposeDimensions()` | TerminalPanel.test.tsx:86-98 | COVERED |
| FILL-02 | `requestAnimationFrame` count ≥2, retry scheduling | TerminalPanel.test.tsx:99-101 | COVERED |
| FILL-03 | `cancelled = true`, `cancelAnimationFrame(rafId)` | TerminalPanel.test.tsx:103-109 | COVERED |
| FILL-04 | Absence of `rafId2`, `document.fonts.ready` | TerminalPanel.test.tsx:112-118 | COVERED |
| FILL-05 | `[isActive]` dependency, `new ResizeObserver`, `fitTerminal()` | TerminalPanel.test.tsx:120-131 | COVERED |
| FILL-06 | Production binary manual verification | Manual | MANUAL-ONLY |

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Fix works in production `wails build` binary | FILL-06 | WebView timing differs from Vite dev server; requires real binary | 1. `wails build -tags wailsassets` 2. Launch binary 3. Open each CLI tab 4. Verify terminal fills full width immediately |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 3s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-03-31

---

## Validation Audit 2026-03-31

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
| Total requirements | 6 |
| Automated coverage | 5 (FILL-01..05) |
| Manual-only | 1 (FILL-06) |
| Test suite | 152 tests, 8 files, all GREEN |
