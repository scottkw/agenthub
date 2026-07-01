---
phase: 167
slug: native-notifications
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-01
---

# Phase 167 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (daemon/app layer) + `vitest` (SettingsTab toggle) |
| **Config file** | none for Go (`go test`); `frontend/vitest.config.ts` (existing) |
| **Quick run command** | `go test -race -short ./... -run Notify` and `cd frontend && pnpm test -- SettingsTab` |
| **Full suite command** | `go test -race -short ./...` and `cd frontend && pnpm test` |
| **Estimated runtime** | ~60–90 seconds (Go race suite) + ~20s (vitest slice) |

---

## Sampling Rate

- **After every task commit:** Run `go test -race -short ./... -run Notify` and the targeted vitest file
- **After every plan wave:** Run `go test -race -short ./...` + full `pnpm test`
- **Before `/gsd-verify-work`:** Full suite must be green; manual M-41 (3 platforms) tracked separately
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| notify-dedup | — | 1 | NTF-02 | — | N/A | unit | `go test -run TestMaybeNotifyWaiting ./...` | ❌ W0 | ⬜ pending |
| notify-coldstart | — | 1 | NTF-02 | — | N/A | unit | `go test -run TestMaybeNotifyWaiting_FirstTickNoNotify ./...` | ❌ W0 | ⬜ pending |
| notify-body | — | 1 | NTF-03 | — | N/A | unit | `go test -run 'TestMaybeNotifyWaiting_BodyFormat|TestDisplayNameForCLI' ./...` | ❌ W0 | ⬜ pending |
| notify-disabled | — | 1 | NTF-04 | — | Toggle-off suppresses all notifications | unit | `go test -run TestMaybeNotifyWaiting_DisabledNoop ./...` | ❌ W0 | ⬜ pending |
| notify-setting-persist | — | 1 | NTF-04 | — | Default false; persists across daemon restart | unit | `go test ./internal/daemon/ -run Notify` | ❌ W0 | ⬜ pending |
| notify-toggle-ui | — | 1 | NTF-04 | — | Renders in Behavior section; loads/saves | unit | `cd frontend && pnpm test -- SettingsTab.notify-toggle` | ❌ W0 | ⬜ pending |
| notify-identifier-fix | — | 1 | NTF-01 | V5 (session name) | Per-session macOS identifier; concurrent notifications don't overwrite | compile/darwin CI | `go test -race -short ./...` (darwin leg) | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `app_test.go` — add `TestMaybeNotifyWaiting_*` table tests using a NEW `sendNotificationFunc` injection seam (App-struct field, mirrors `saveFileDialogFunc`) so no real OS call fires in tests
- [ ] `internal/daemon/engine_notify_test.go` — new; mirrors `engine_shell_warn_test.go` for `NotifyOnWaiting` (default false / persist / round-trip)
- [ ] `internal/daemon/api_notify_test.go` (or extend `api_test.go`) — GET/PATCH notify-on-waiting handler tests, mirrors `start-minimized` route tests
- [ ] `frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx` — new; mirrors `SettingsTab.shell-warn-toggle.test.tsx`
- [ ] `TestDisplayNameForCLI` — table test for the static agent-type display-name lookup (mirrors `knownCLIs`)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cross-platform on-screen notification delivery on `→ waiting`, including tray-hidden | NTF-01/02/03 | Real OS notification centers require a live desktop session; CI runners (`build.yml`) have none on any of the 3 platforms | **M-41** (new, Section 5 Category U): on each of macOS / Windows / Linux — enable the toggle, hide the window to tray (QuitGUIOnly-style, not full quit), drive a session into `waiting`, confirm exactly ONE OS-native notification appears identifying session name + agent type while the window is hidden. Repeat with toggle OFF → confirm no notification (NTF-04). macOS must show "AgentHub" attribution (native path). |

*Planner MUST add M-41 to TESTING.md Section 5 (new `### Category U — Native Notifications (NTF)`) and a Traceability row per the Standing Convention; run `bash tests/check-traceability-paths.sh` before committing.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-07-01 (gsd-plan-checker PASS)
