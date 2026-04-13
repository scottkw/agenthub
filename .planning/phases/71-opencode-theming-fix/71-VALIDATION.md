---
phase: 71
slug: opencode-theming-fix
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-13
---

# Phase 71 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `testing` (stdlib) |
| **Framework (JS)** | Vitest ^4.1.0 |
| **Config file** | `frontend/vitest.config.ts` + `go.mod` (no extra config) |
| **Quick run command** | `go test ./internal/daemon/ -count=1 -short && cd frontend && pnpm test --run` |
| **Full suite command** | `go test ./... -count=1 -short && cd frontend && pnpm test --run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/ -count=1 -short`
- **After every plan wave:** Run `go test ./... -count=1 -short && cd frontend && pnpm test --run`
- **Before `/gsd-verify-work`:** Full suite must be green + manual visual UAT
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 71-01-01 | 01 | 0 | THM-05 | — | Wave 0 test stubs created | unit (Go) | `go test ./internal/daemon/ -count=1 -run TestCreateSession_OpenCodeEnv` | ❌ W0 | ⬜ pending |
| 71-01-02 | 01 | 0 | THM-05 | — | Wave 0 test stub for managed tui.json | unit (Go) | `go test ./internal/daemon/ -count=1 -run TestOpenCodeTUIConfig` | ❌ W0 | ⬜ pending |
| 71-01-03 | 01 | 0 | THM-05 | — | Pre-existing test fixed (4 → 5 CLIs) | unit (Go) | `go test ./internal/status/ -count=1 -run TestKnownCLIs_HasExpectedEntries` | ✅ | ⬜ pending |
| 71-02-01 | 02 | 1 | THM-05 | — | Managed opencode-tui.json written at configDir | unit (Go) | `go test ./internal/daemon/ -count=1 -run TestOpenCodeTUIConfig` | ✅ W0 | ⬜ pending |
| 71-02-02 | 02 | 1 | THM-05 | — | CreateSession injects OPENCODE_TUI_CONFIG for opencode CLI only | unit (Go) | `go test ./internal/daemon/ -count=1 -run TestCreateSession_OpenCodeEnv` | ✅ W0 | ⬜ pending |
| 71-02-03 | 02 | 1 | THM-05 | T-71-01 | No env var injected for non-opencode CLIs | unit (Go) | `go test ./internal/daemon/ -count=1 -run TestCreateSession_OpenCodeEnv` | ✅ W0 | ⬜ pending |
| 71-03-01 | 03 | 2 | THM-05 | — | Empirical ANSI capture confirms system theme emits palette indices | integration (Go) | `go test ./internal/daemon/ -count=1 -run TestOpenCodeANSICapture -timeout 30s` | ❌ W0 | ⬜ pending |
| 71-04-01 | 04 | 2 | THM-05 | — | Theme change repaints active OpenCode session (manual UAT) | manual | Human in /gsd-verify-work | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/engine_test.go` — stub `TestCreateSession_OpenCodeEnv` for THM-05 (env var injection)
- [ ] `internal/daemon/engine_test.go` — stub `TestOpenCodeTUIConfig` for THM-05 (managed JSON file creation)
- [ ] `internal/status/detector_test.go` — fix `TestKnownCLIs_HasExpectedEntries` (pre-existing failure: expects 4, has 5)
- [ ] `internal/daemon/opencode_ansi_test.go` — stub `TestOpenCodeANSICapture` (integration test that spawns opencode in PTY, captures escape sequences, asserts ANSI palette indices)

*Rationale: Wave 0 adds stubs so red → green transitions track implementation during Wave 1+.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Theme switch repaints active OpenCode session to match selected palette | THM-05 (SC-1) | Visual/perceptual — color pixel comparison in xterm.js canvas is flaky across GPUs | 1. Launch app, open OpenCode session; 2. Settings > Appearance → switch theme; 3. Verify OpenCode panel colors change without restart |
| New OpenCode session starts with current theme applied | THM-05 (SC-2) | Visual | 1. Set theme X; 2. Create new OpenCode session; 3. Verify colors match theme X immediately |
| All four agents visually consistent under same theme | THM-05 (SC-3) | Cross-CLI visual comparison | 1. Open sessions for claude, codex, gemini, opencode; 2. Verify the four terminal panels render matching palette for same theme |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
