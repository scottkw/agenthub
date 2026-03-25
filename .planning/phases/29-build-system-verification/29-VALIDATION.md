---
phase: 29
slug: build-system-verification
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-25
---

# Phase 29 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` stdlib + bash (for build-script tests) |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test -race ./...` |
| **Full suite command** | `go test -race -count=1 ./... && bash tests/build-script.test.sh` |
| **Estimated runtime** | ~20 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./...`
- **After every plan wave:** Run `go test -race -count=1 ./... && bash tests/build-script.test.sh`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 29-01-01 | 01 | 1 | BUILD-01 | smoke | `bash tests/build-script.test.sh` | ✅ | ⬜ pending |
| 29-01-02 | 01 | 1 | BUILD-02 | integration | `go test -race ./...` (CI) | ✅ | ⬜ pending |
| 29-01-03 | 01 | 1 | BUILD-03 | automated | `go test -race -count=1 ./...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. The only prerequisite fix is the hardcoded absolute path in `tests/build-script.test.sh` (line 10) which must be converted to a relative path before CI can invoke it.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CI workflow runs green on all 4 matrix legs | BUILD-02 | Requires actual GitHub Actions run | Push branch, verify all matrix jobs pass |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 20s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
