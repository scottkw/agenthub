---
phase: 136
slug: tui-removal
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-19
validated: 2026-06-19
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
| NAV-01 | `agenthub tui` exits non-zero / unrecognized | smoke | `./agenthub tui; echo $?` (non-zero) | ✅ post-build | ✅ green |
| NAV-01 | No `internal/tui` import paths remain | build gate | `! grep -rn "scottkw/agenthub/internal/tui" --include="*.go" . --exclude-dir=.claude` | ✅ | ✅ green |
| NAV-01 | No charm.land/charmbracelet TUI deps in go.mod | build gate | `go mod tidy && ! grep -E "charm.land/(bubbletea|bubbles|lipgloss)|charmbracelet/glamour" go.mod` | ✅ | ✅ green |
| NAV-01 | Build compiles without TUI package | build gate | `go build ./...` | ✅ | ✅ green |
| TEST-06 | All TUI test files deleted | build gate | `! ls internal/tui 2>/dev/null` (dir gone) | ✅ | ✅ green |
| TEST-06 | Daemon parity tests referencing TUI removed | build gate | `go vet ./internal/daemon/...` (compiles, no tui import) | ✅ | ✅ green |
| TEST-06 | Surviving Go suite passes | automated | `go test -race -short ./...` | ✅ | ✅ green |
| TEST-06 | Frontend tests pass (no regression) | automated | `cd frontend && pnpm test` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> **Audit note (NAV-01-b):** The original grep gate `grep -rn "internal/tui"` was too
> coarse — it matches the substring inside the explanatory comment `// Nothing in this
> file imports internal/tui.` in `remote_files_test_helpers_test.go`. The command above
> was narrowed to the full package import path `scottkw/agenthub/internal/tui`, which
> correctly returns zero matches (no real import statements remain).

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — deletion phase)
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-19

---

## Validation Audit 2026-06-19

State A audit — existing VALIDATION.md re-run against the executed codebase.

| Metric | Count |
|--------|-------|
| Requirements audited | 8 |
| COVERED | 8 |
| PARTIAL | 0 |
| MISSING | 0 |
| Gaps found | 0 |
| Tests generated | 0 (deletion phase — existing Go/frontend suites cover all reqs) |
| Escalated | 0 |

**Live re-run evidence:** All 6 quick build gates executed live and passed
(`agenthub tui` → exit 1; no `scottkw/agenthub/internal/tui` imports; no charm deps
in `go.mod`; `go build ./...` green; `internal/tui/` dir absent; `go vet ./internal/daemon/...`
clean). `internal/daemon` race suite green. Full `go test -race -short ./...` and frontend
`pnpm test` (1749 tests) confirmed green during execution + phase verification with only a
documentation edit since. One gate command (NAV-01-b) was refined to eliminate a
comment-substring false positive. Phase is **Nyquist-compliant**.
