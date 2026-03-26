---
phase: 34
slug: terminal-fill-fix
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 34 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 (frontend) + go test (backend) |
| **Config file** | `frontend/vite.config.ts` (vitest), native go test |
| **Quick run command** | `cd frontend && npm test && go test ./internal/daemon/...` |
| **Full suite command** | `cd frontend && npm test && go test ./internal/daemon/... ./internal/pty/...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npm test && go test ./internal/daemon/... ./internal/pty/...`
- **After every plan wave:** Run `cd frontend && npm test && go test ./internal/daemon/... ./internal/pty/...`
- **Before `/gsd:verify-work`:** Full suite must be green + manual production binary validation
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 34-01-01 | 01 | 1 | TERM-04 | unit (source inspection) | `cd frontend && npm test` | ✅ `__tests__/TerminalPanel.test.tsx` | ⬜ pending |
| 34-01-02 | 01 | 1 | TERM-01/02 | unit (source inspection) | `cd frontend && npm test` | ❌ W0 | ⬜ pending |
| 34-01-03 | 01 | 1 | TERM-03 | unit | `go test ./internal/daemon/...` | ✅ `internal/daemon/engine_test.go` | ⬜ pending |
| 34-01-04 | 01 | 1 | TERM-03 | unit | `go test ./internal/daemon/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — add tests for TERM-04 double-rAF pattern check, TERM-01 initial-fit-not-synchronous check
- [ ] `internal/daemon/engine_test.go` — add test for TERM-03 default dimension guard (cols=0 → 80, rows=0 → 24)

*Existing infrastructure covers framework setup; only new test cases needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Terminal fills viewport on first tab activation (production binary) | TERM-01/02 | Requires Wails production build + visual confirmation | 1. `wails build -tags wailsassets` 2. Launch app 3. Create new Claude/Gemini session 4. Verify terminal fills without resize |
| Tab switch does not produce 1-column render | TERM-04 | Requires real browser layout engine | 1. Open multiple sessions 2. Switch tabs 3. Verify no collapsed terminal |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
