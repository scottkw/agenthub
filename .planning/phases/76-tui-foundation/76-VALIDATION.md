---
phase: 76
slug: tui-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-15
---

# Phase 76 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `76-RESEARCH.md` → `## Validation Architecture`.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib, go1.26.2) |
| **Config file** | none — Go's built-in test runner |
| **Quick run command** | `go test ./internal/tui/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -count=1 -timeout 120s` |
| **Estimated runtime** | ~15s quick / ~60s full |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/tui/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -count=1 -timeout 120s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

*Populated during planning — planner links each plan task to this map. Minimum coverage targets below; executor fills actual Task IDs after PLAN.md files exist.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD (Wave 0) | 76-0X | 0 | — | — | N/A | install | `go get charm.land/bubbletea/v2@v2.0.5 charm.land/lipgloss/v2@v2.0.3 charm.land/bubbles/v2@v2.1.0` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | TUI-01 | — | Launches without panic; non-TTY fallback | unit + manual UAT | `go test ./internal/tui/... -run TestProgramInit -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | TUI-02 | T-76-INJ | Session names rendered via Lip Gloss (no raw ANSI) | unit | `go test ./internal/tui/... -run TestView_SessionList -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | TUI-08 | — | Footer shows web server on/off + URL | unit | `go test ./internal/tui/... -run TestView_Footer -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | TUI-09 | — | `?` toggles help overlay; `Esc`/`?` closes | unit | `go test ./internal/tui/... -run TestHelpOverlay -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*The planner MUST update Task IDs and Plan refs when PLAN.md files are finalized. The checker will enforce that each TUI-XX requirement has at least one row here.*

---

## Wave 0 Requirements

Files that must exist before the first implementation task runs. Each satisfies a MISSING reference in the map above.

- [ ] `internal/tui/` package directory created
- [ ] `internal/tui/tui_test.go` — stubs for `TestProgramInit` (TUI-01)
- [ ] `internal/tui/update_test.go` — stubs for `TestUpdate_*` covering state transitions (TUI-01/02/08/09)
- [ ] `internal/tui/view_test.go` — stubs for `TestView_SessionList`, `TestView_Footer` (TUI-02/08)
- [ ] `internal/tui/help_test.go` — stubs for `TestHelpOverlay` (TUI-09)
- [ ] Dependency install: `go get charm.land/bubbletea/v2@v2.0.5 charm.land/lipgloss/v2@v2.0.3 charm.land/bubbles/v2@v2.1.0`
- [ ] `go mod tidy` after install

---

## Manual-Only Verifications

TUI rendering fidelity cannot be fully covered by headless unit tests. The following behaviors require live-terminal UAT:

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Alt-screen enter/exit cleanliness (scrollback preserved) | TUI-01 | Depends on real terminal emulator behavior | Run `agenthub tui` in a real terminal; press `q`; confirm prior shell scrollback returns intact |
| Adaptive color rendering on dark vs light background | UI-SPEC §Color | `tea.BackgroundColorMsg` only fires against a real terminal | Run `agenthub tui` in a dark-bg terminal and a light-bg terminal; verify contrast in selected row and accent glyphs |
| Unicode glyph rendering (status dots `●`, help close `×`) | TUI-02, TUI-09 | Font substitution varies per terminal/OS | Visual inspection across macOS Terminal, iTerm2, and at least one Linux terminal (or fallback note in HUMAN-UAT) |
| Resize handling (no tearing/garbage) | TUI-01 | SIGWINCH delivery requires real TTY | Run TUI; resize the terminal window smaller/larger; verify layout reflows cleanly |
| Help overlay centering on odd terminal dimensions | TUI-09 | Exact centering depends on runtime size | Resize to 61×11, 80×24, 120×40; verify help overlay stays centered and bordered |
| Sub-minimum size fallback (< 60×10) | UI-SPEC §Minimum | Needs live resize | Resize to 40×8; verify graceful "Terminal too small" message; resize back; verify recovery |
| Non-TTY fallback (piped stdout) | TUI-01 | Non-TTY environments vary | Run `agenthub tui | cat`; verify graceful exit with diagnostic message, not panic |

Each of the above MUST be listed as a human UAT item in `76-HUMAN-UAT.md` after implementation.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (test stubs + `go get` + `go mod tidy`)
- [ ] No watch-mode flags (always `-count=1`)
- [ ] Feedback latency < 30s
- [ ] Manual UAT items from this file are mirrored in `76-HUMAN-UAT.md` after execution
- [ ] `nyquist_compliant: true` set in frontmatter after planner links all Task IDs

**Approval:** pending
