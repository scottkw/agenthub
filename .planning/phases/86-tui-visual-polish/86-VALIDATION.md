---
phase: 86
slug: tui-visual-polish
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-19
---

# Phase 86 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing package (stdlib) |
| **Config file** | none — standard `go test` |
| **Quick run command** | `cd /Users/ken/dev/agenthub && go test ./internal/tui/... -count=1` |
| **Full suite command** | `cd /Users/ken/dev/agenthub && go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/tui/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 86-01-01 | 01 | 1 | TUI-04 | — | N/A | unit | `go test ./internal/tui/... -run TestStyles_TokyoNight -count=1` | ❌ W0 | ⬜ pending |
| 86-01-02 | 01 | 1 | TUI-04 | — | N/A | unit | `go test ./internal/tui/... -run TestAgentBadgeColor -count=1` | ❌ W0 | ⬜ pending |
| 86-02-01 | 02 | 1 | TUI-01 | — | N/A | unit | `go test ./internal/tui/... -run TestView_SessionFrame -count=1` | ❌ W0 | ⬜ pending |
| 86-02-02 | 02 | 1 | TUI-01 | — | N/A | unit | `go test ./internal/tui/... -run TestInjectBorderTitle -count=1` | ❌ W0 | ⬜ pending |
| 86-03-01 | 03 | 2 | TUI-02 | — | N/A | unit | `go test ./internal/tui/... -run TestUpdate_TabFocusToggle -count=1` | ❌ W0 | ⬜ pending |
| 86-03-02 | 03 | 2 | TUI-02 | — | N/A | unit | `go test ./internal/tui/... -run TestUpdate_TabCycle -count=1` | ❌ W0 | ⬜ pending |
| 86-03-03 | 03 | 2 | TUI-02 | — | N/A | unit | `go test ./internal/tui/... -run TestUpdate_SidebarNavigation -count=1` | ❌ W0 | ⬜ pending |
| 86-03-04 | 03 | 2 | TUI-02 | — | N/A | unit | `go test ./internal/tui/... -run TestView_Sidebar -count=1` | ❌ W0 | ⬜ pending |
| 86-04-01 | 04 | 2 | TUI-03 | — | N/A | unit | `go test ./internal/tui/... -run TestView_AgentBadge -count=1` | ❌ W0 | ⬜ pending |
| 86-04-02 | 04 | 2 | TUI-03 | — | N/A | unit | `go test ./internal/tui/... -run TestStatusGlyph -count=1` | ✅ view_test.go | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/tui/styles_test.go` — new file: `TestStyles_TokyoNight`, `TestAgentBadgeColor`
- [ ] `internal/tui/view_test.go` — add `TestView_SessionFrame`, `TestView_Sidebar`, `TestView_AgentBadge`, `TestInjectBorderTitle`
- [ ] `internal/tui/update_test.go` — add `TestUpdate_TabFocusToggle`, `TestUpdate_TabCycle`, `TestUpdate_SidebarNavigation`

*Existing infrastructure covers test framework — no new dependencies needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual color rendering matches TokyoNight palette | TUI-04 | Terminal color rendering depends on terminal emulator | Run `go run . tui`, visually compare sidebar/content colors against GUI |
| Tab bar underline indicator visible | TUI-02 | Depends on terminal ANSI underline support | Run `go run . tui`, verify active tab is visually distinct |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
