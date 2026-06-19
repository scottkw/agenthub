---
phase: 136
slug: tui-removal
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-19
---

# Phase 136 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

This is a **deletion phase**. No new behavior is added; validation proves that
removal left the surviving build, test matrix, and surfaces green with zero TUI
residue. There are **no Wave 0 test stubs** — TEST-06 is deletion, not addition.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package (+ frontend `pnpm test`) |
| **Config file** | `go.mod` (Go version pinned); frontend `package.json` |
| **Quick run command** | `go build ./...` |
| **Full suite command** | `go test -race -short ./...` |
| **Frontend suite** | `cd frontend && pnpm test` |
| **CI test command** | `go test -race -short ./...` (mac/linux); `go test -race -short ./internal/...` (Windows) |
| **Estimated runtime** | ~60–120 seconds (Go race suite) |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `go test -race -short ./...`
- **Before `/gsd:verify-work`:** `go test -race -short ./...` green + `cd frontend && pnpm test` green + `agenthub tui` exits non-zero
- **Max feedback latency:** ~120 seconds

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this maps the phase requirements to their
proving commands. Every requirement has an automated build/test gate.

| Req | Behavior | Test Type | Automated Command | File Exists | Status |
|-----|----------|-----------|-------------------|-------------|--------|
| NAV-01 | `agenthub tui` exits non-zero / unrecognized | smoke | `./agenthub tui; echo $?` (non-zero) | ✅ post-build | ⬜ pending |
| NAV-01 | No `internal/tui` import paths remain | build gate | `! grep -rn "internal/tui" --include=*.go . --exclude-dir=.claude` | ✅ | ⬜ pending |
| NAV-01 | No charm.land/charmbracelet TUI deps in go.mod | build gate | `go mod tidy && ! grep -E "charm.land/(bubbletea|bubbles|lipgloss)|charmbracelet/glamour" go.mod` | ✅ | ⬜ pending |
| NAV-01 | Build compiles without TUI package | build gate | `go build ./...` | ✅ | ⬜ pending |
| TEST-06 | All TUI test files deleted | build gate | `! ls internal/tui 2>/dev/null` (dir gone) | ✅ | ⬜ pending |
| TEST-06 | Daemon parity tests referencing TUI removed | build gate | `go vet ./internal/daemon/...` (compiles, no tui import) | ✅ | ⬜ pending |
| TEST-06 | Surviving Go suite passes | automated | `go test -race -short ./...` | ✅ | ⬜ pending |
| TEST-06 | Frontend tests pass (no regression) | automated | `cd frontend && pnpm test` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements.* No new test files are
written — TEST-06 is deletion, not addition. The Go and frontend suites already
exist and run in CI.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `agenthub tui` is removed from CLI help/usage | NAV-01 | Human-read of usage text | Run `./agenthub --help` / `./agenthub` and confirm no `tui` subcommand listed |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (none — deletion phase)
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
