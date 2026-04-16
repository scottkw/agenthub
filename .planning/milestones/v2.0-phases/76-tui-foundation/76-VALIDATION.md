---
phase: 76
slug: tui-foundation
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-15
linked_at: 2026-04-15
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

*Linked to finalized plans 76-01, 76-02, 76-03. Task IDs follow `76-NN-TM` convention (plan NN, task M within the plan). Tests are created in Plan 76-02 Task 3 (same-wave with implementation since they reference real types) rather than as pure Wave 0 stubs — this is a deliberate, non-scope-reducing choice and the research-resolution note in this section's history.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 76-01-T1 | 76-01 | 1 | TUI-02 | — | `SessionInfo.Status` populated from heuristic detector under `statusMu.RLock` | unit + build | `go build ./internal/daemon/... && go test ./internal/daemon/... -count=1 -timeout 30s` | ✅ | ✅ green |
| 76-01-T2 | 76-01 | 1 | TUI-01, TUI-02 | — | Charm v2 deps installed; TUI package scaffolding compiles | build + vet | `go build ./internal/tui/... && go vet ./internal/tui/...` | ✅ | ✅ green |
| 76-02-T1 | 76-02 | 2 | TUI-01, TUI-09 | T-76-DOS | TUI `Init/Update` dispatch; daemon errors surface as `tea.Msg`, not panic; `?` toggles `showHelp` | build | `go build ./internal/tui/...` | ✅ | ✅ green |
| 76-02-T2 | 76-02 | 2 | TUI-02, TUI-08, TUI-09 | T-76-INJ, T-76-RES | Session rows rendered via Lip Gloss (no raw ANSI); footer shows web status+URL; help overlay centered bordered modal; resize handled via `WindowSizeMsg` | build + vet | `go build ./internal/tui/... && go vet ./internal/tui/...` | ✅ | ✅ green |
| 76-02-T3 | 76-02 | 2 | TUI-01, TUI-02, TUI-08, TUI-09 | T-76-INJ, T-76-DOS | Unit tests for `Update`, `View`, help overlay — covers all 4 requirements | unit | `go test ./internal/tui/... -count=1 -timeout 30s -v` | ✅ | ✅ green |
| 76-03-T1 | 76-03 | 3 | TUI-01 | — | `agenthub tui` launches cleanly on TTY; non-TTY prints diagnostic and exits 1 | build + vet | `go build -o /dev/null . && go vet .` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Coverage check:** TUI-01 (3 tasks), TUI-02 (4 tasks), TUI-08 (2 tasks), TUI-09 (3 tasks). Every requirement has ≥ 2 verifying tasks across build and unit dimensions.

---

## Wave 0 Requirements

Resolved by Plan 76-01 (Wave 1) — dependency install + package scaffolding happens in the first plan's first task. Unit test files are intentionally created in Plan 76-02 Task 3 (same wave as implementation) because they reference real model/view types created in that wave; pure Wave 0 stubs would fail to compile against not-yet-defined types.

- [x] Dependency install — 76-01-T1: `go get charm.land/bubbletea/v2@v2.0.5 charm.land/lipgloss/v2@v2.0.3 charm.land/bubbles/v2@v2.1.0` + `go mod tidy`
- [x] `internal/tui/` package directory created — 76-01-T2 (model.go, styles.go, keys.go, cmds.go)
- [x] `internal/tui/update_test.go` — 76-02-T3 covers TUI-01/02/08/09 state transitions
- [x] `internal/tui/view_test.go` — 76-02-T3 covers TUI-02/08 rendering
- [x] `internal/tui/help_test.go` — 76-02-T3 covers TUI-09 overlay

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (test stubs + `go get` + `go mod tidy`)
- [x] No watch-mode flags (always `-count=1`)
- [x] Feedback latency < 30s
- [x] Manual UAT items from this file are mirrored in `76-HUMAN-UAT.md` after execution
- [x] `nyquist_compliant: true` set in frontmatter after planner links all Task IDs

**Approval:** approved

---

## Validation Audit 2026-04-16

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

**Commands verified:**
- `go build ./internal/daemon/... && go test ./internal/daemon/...` — PASS
- `go build ./internal/tui/... && go vet ./internal/tui/...` — PASS
- `go build ./internal/tui/...` — PASS
- `go test ./internal/tui/... -count=1 -timeout 30s -v` — PASS (78 tests green)
- `go build -o /dev/null . && go vet .` — PASS

**Requirement coverage:** TUI-01 (3 tasks), TUI-02 (4 tasks), TUI-08 (2 tasks), TUI-09 (3 tasks) — all COVERED with automated tests.
