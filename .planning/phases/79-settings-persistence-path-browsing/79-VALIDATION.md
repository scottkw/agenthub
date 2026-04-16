---
phase: 79
slug: settings-persistence-path-browsing
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-16
---

# Phase 79 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — uses standard Go testing |
| **Quick run command** | `go test ./internal/daemon/... -run TestSettings -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/... -run TestSettings -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 79-01-01 | 01 | 1 | SET-01 | — | N/A | unit | `go test ./internal/daemon/... -run TestSettingsPersistence -count=1` | ❌ W0 | ⬜ pending |
| 79-01-02 | 01 | 1 | SET-02 | — | N/A | unit | `go test ./internal/daemon/... -run TestTailscalePathPersistence -count=1` | ❌ W0 | ⬜ pending |
| 79-02-01 | 02 | 1 | SET-03 | — | N/A | manual | UI visual confirmation | N/A | ⬜ pending |
| 79-02-02 | 02 | 1 | SET-04 | — | N/A | integration | `go test ./... -run TestOpenFileDialog -count=1` | ❌ W0 | ⬜ pending |
| 79-02-03 | 02 | 1 | SET-05 | — | N/A | manual | UI picker populates field | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/settings_test.go` — stubs for SET-01, SET-02 persistence tests
- [ ] Existing Go test infrastructure covers framework needs

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Save confirmation visible | SET-03 | Visual UI feedback (toast/flash) | Click Save, verify visible indicator appears before button returns to idle |
| Picker populates field | SET-05 | Native OS dialog interaction | Click browse, select file, verify path appears in input |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
