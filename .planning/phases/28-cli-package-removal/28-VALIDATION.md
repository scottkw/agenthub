---
phase: 28
slug: cli-package-removal
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-25
---

# Phase 28 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package |
| **Config file** | none (go test uses go.mod) |
| **Quick run command** | `go build ./...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 28-01-01 | 01 | 1 | CLEAN-01 | smoke | `ls cmd/agenthub-cli 2>/dev/null && exit 1 \|\| exit 0` | N/A | ✅ green |
| 28-01-02 | 01 | 1 | CLEAN-01 | build | `go build ./...` | N/A | ✅ green |
| 28-01-03 | 01 | 1 | CLEAN-02 | smoke | `grep -r "agenthub-cli" . --include="*.go" --include="*.sh" --include="*.yml" --include="*.md" --exclude-dir=.planning --exclude-dir=.claude` | N/A | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. This phase deletes tests, it does not add them.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved

## Validation Audit 2026-03-25

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All 3 verification commands executed and returned green. No auditor agent needed.
