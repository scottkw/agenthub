---
phase: 57
slug: quick-wins
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-08
---

# Phase 57 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test / vitest |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./internal/daemon/...` |
| **Full suite command** | `go test ./... && cd frontend && pnpm test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/...`
- **After every plan wave:** Run `go test ./... && cd frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 57-01-01 | 01 | 1 | DET-01 | unit | `go test ./internal/daemon/ -run TestAugmentServicePath` | ✅ | ⬜ pending |
| 57-02-01 | 02 | 1 | UI-01 | unit | `cd frontend && pnpm test -- --grep "New Session"` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Windows native installer path | DET-01 | No Windows CI | Install claude via Anthropic installer on Windows, verify detection |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
