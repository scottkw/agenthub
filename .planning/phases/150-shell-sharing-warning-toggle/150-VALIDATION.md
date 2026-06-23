---
phase: 150
slug: shell-sharing-warning-toggle
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-23
---

# Phase 150 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (daemon) + vitest (frontend) |
| **Config file** | `frontend/vitest.config.ts`; Go uses standard `go test ./...` |
| **Quick run command** | `cd frontend && pnpm vitest run <changed test>` / `go test -race -short ./internal/daemon/...` |
| **Full suite command** | `cd frontend && pnpm vitest run && pnpm tsc --noEmit` ; `go test -race -short ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run the relevant quick command (go test for daemon, vitest for frontend)
- **After every plan wave:** Run the full suite for the touched surface
- **Before `/gsd:verify-work`:** Full suite must be green (vitest + `tsc --noEmit` + `go test -race ./...`)
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

> Derived from the D-04 behavior matrix + RESEARCH.md "Validation Architecture".

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 150-01-01 | 01 | 1 | SET-01 | T-150-01 | `*bool` field, nil->true default ON; engine builds | unit (build) | `go build ./internal/daemon/...` | ❌ W0 | ⬜ pending |
| 150-01-02 | 01 | 1 | SET-01 | T-150-01,02,04 | default ON, persist round-trip, atomic OFF->ON re-arm, OFF-no-rearm, API get/patch, client round-trip | unit | `go test -race -short ./internal/daemon/... -run ShellWebShareWarningEnabled` | ❌ W0 | ⬜ pending |
| 150-02-01 | 02 | 2 | SET-01 | T-150-05,06,07 | toggle renders in Session Behavior; confirm-on-disable gates OFF; instant ON; tsc clean | source + type | `cd frontend && pnpm tsc --noEmit` | ❌ W0 | ⬜ pending |
| 150-02-02 | 02 | 2 | SET-01 | T-150-05 | confirm-on-disable / cancel / instant-ON paths; search index byte-match | unit | `cd frontend && pnpm vitest run SettingsTab.shell-warn-toggle SettingsSearch` | ❌ W0 | ⬜ pending |
| 150-03-01 | 03 | 2 | SET-01 | T-150-09 | StatusBar gate adds warningEnabled; re-arm re-sync; prop threading | source | `cd frontend && grep warningEnabled gate in App.tsx` | partial | ⬜ pending |
| 150-03-02 | 03 | 2 | SET-01 | T-150-08,10,11 | Share-modal shell/non-shell/disabled interception; shared warned authority | unit | `cd frontend && pnpm vitest run SessionShareModal App.shellWebShare` | extend | ⬜ pending |
| 150-03-03 | 03 | 2 | SET-01 | — | TESTING.md registration; traceability paths valid | script | `bash tests/check-traceability-paths.sh` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

> Concrete from RESEARCH.md. Created by the executor before the implementation in the same task (TDD `tdd="true"` tasks write the failing test first).

- [ ] `internal/daemon/engine_shell_warn_test.go` — default ON, persist round-trip, OFF->ON re-arm resets shellWebShareWarned, OFF does not re-arm (mirror engine_settings_test.go) — created in 150-01 Task 2
- [ ] `internal/daemon/api_shell_warn_test.go` — GET default true, PATCH flips, DaemonClient round-trip (mirror api_test.go:1588-1660) — created in 150-01 Task 2
- [ ] `frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx` — toggle state machine + confirm-on-disable + cancel + instant-ON — created in 150-02 Task 2
- [ ] Extend `frontend/src/components/__tests__/SessionShareModal.test.tsx` — shell vs non-shell vs disabled interception; banner without ToggleWebServing — created in 150-03 Task 2
- [ ] Extend `frontend/src/components/__tests__/App.shellWebShare.test.tsx` — warningEnabled=false StatusBar suppression — created in 150-03 Task 2

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live web-share of a real shell session fires the warning banner on the actual share surface | SET-01 | Live PTY web-share cannot be driven through the automated bridge (no PTY) — TESTING.md §5 | Web-share a shell session via Hub Share modal and via StatusBar; confirm banner appears when enabled+unacked, absent when disabled (new M-NN in 150-03 Task 3) |
| Persistence across restart | SET-01 | Requires full daemon restart with on-disk settings.json | Toggle OFF, restart the app, confirm the warning stays suppressed; toggle ON, restart, confirm it fires again (new M-NN in 150-03 Task 3) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** ready for execution
