---
phase: 144
slug: daemon-styled-tail-race-fix
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-21
---

# Phase 144 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (race detector) |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test -race ./internal/daemon/ -run 'TestGetSessionStyledTailLines'` |
| **Full suite command** | `go test -race ./internal/daemon/` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd:verify-work`:** Full suite must be green (race detector clean)
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-----------|--------|
| 144-01-01 | 01 | 1 | FIX-01 | — | N/A | unit (race) | `go test -race ./internal/daemon/ -run 'TestGetSessionStyledTailLines'` | ✅ | ⬜ pending |
| 144-01-02 | 01 | 1 | FIX-01 | — | N/A | unit (regression) | `go test ./internal/daemon/ -run 'TestGetSessionStyledTailLines'` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] No new framework — `go test -race` already in CI ("Run Go tests (all platforms, race detector)")
- [ ] Optional new regression fixture `TestGetSessionStyledTailLines_AllQueriesNoHang` — broadened query-sequence scrollback proving Write never blocks after the strip (per RESEARCH.md open question A2). If added, it lives in the existing daemon test file (no new file) OR if a new file is created, add a TESTING.md traceability row for FIX-01 + run `bash tests/check-traceability-paths.sh`.

*Existing infrastructure covers all phase requirements; the four issue-named tests already assert the behavior under `-race`.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Mini-preview / briefing-modal tail renders correctly in the live GUI | FIX-01 (SC#2) | Visual render in native Wails webview | Build app, open a session with colored/TUI output, confirm mini-preview + briefing modal show styled tail with spacing preserved (no #96 "Loading…" hang). Covered indirectly by `TestGetSessionStyledTailLines_*` rendering assertions — manual check is confirmatory only. |

*Primary verification is automated via the race + rendering tests; manual check is confirmatory.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (broadened AllQueriesNoHang fixture in existing engine_test.go)
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-06-21
