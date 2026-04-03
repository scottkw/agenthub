---
phase: 44
slug: git-migration-to-github
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-03
---

# Phase 44 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (built-in) |
| **Config file** | none (standard go test) |
| **Quick run command** | `go build ./...` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 44-01-01 | 01 | 1 | GIT-02 | unit | `grep -r "github.com/agenthub/agenthub" --include="*.go" . \| wc -l` → 0 | ✅ | ⬜ pending |
| 44-01-02 | 01 | 1 | GIT-02 | unit | `head -1 go.mod` → `module github.com/scottkw/agenthub` | ✅ | ⬜ pending |
| 44-01-03 | 01 | 1 | GIT-02 | integration | `go build ./...` | ✅ | ⬜ pending |
| 44-01-04 | 01 | 1 | GIT-02 | integration | `go test -race ./...` | ✅ | ⬜ pending |
| 44-02-01 | 02 | 2 | GIT-01 | smoke | `git clone https://github.com/scottkw/agenthub /tmp/verify && git -C /tmp/verify log --oneline \| wc -l` | ❌ W0 | ⬜ pending |
| 44-02-02 | 02 | 2 | GIT-01 | smoke | `git -C /tmp/verify tag \| sort -V` | ❌ W0 | ⬜ pending |
| 44-02-03 | 02 | 2 | GIT-01 | smoke | `git -C /tmp/verify cat-file -t $(git -C /tmp/verify rev-parse v1.7)` → `tag` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Smoke verification commands for GIT-01 (clone, tag check, annotated tag object check) — these are shell commands run post-migration, not test files

*Existing go test infrastructure covers all GIT-02 requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CI secrets present in GitHub settings | GIT-01 | Secret values cannot be read back via API | Navigate to `github.com/scottkw/agenthub/settings/secrets/actions` — verify 7 secrets listed |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
