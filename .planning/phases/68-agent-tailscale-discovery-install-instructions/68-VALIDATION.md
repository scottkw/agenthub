---
phase: 68
slug: agent-tailscale-discovery-install-instructions
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-11
audited: 2026-04-12
---

# Phase 68 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + vitest |
| **Config file** | `internal/agents/agents_test.go`, `frontend/vitest.config.ts` |
| **Quick run command** | `go test ./internal/daemon/... && cd frontend && npx vitest run --reporter=verbose` |
| **Full suite command** | `go test ./... && cd frontend && npx vitest run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/... && cd frontend && npx vitest run`
- **After every plan wave:** Run `go test ./... && cd frontend && npx vitest run`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 68-01-01 | 01 | 1 | DISC-01 | T-68-01 | Hardcoded candidate list, no user input | unit | `go test ./internal/daemon/... -run "TestAugmentServicePath_Cargo\|TestAugmentServicePath_FlatpakUser\|TestAugmentServicePath_AddsExistingDirs\|TestAugmentServicePath_AddsLocalBin\|TestAugmentServicePath_SkipsNonexistent"` | ✅ | ✅ green |
| 68-01-02 | 01 | 1 | DISC-02 | T-68-01 | os.Setenv modifies process PATH for exec.LookPath | unit | `go test ./internal/daemon/... -run "TestAugmentServicePath_AddsExistingDirs\|TestAugmentServicePath_PrependsNotAppends"` | ✅ | ✅ green |
| 68-01-03 | 01 | 1 | DISC-03 | T-68-02 | Windows Tailscale in protected system dir | unit | `go test ./internal/daemon/... -run TestPlatformExtraBins_NonWindows` | ✅ | ✅ green |
| 68-02-01 | 02 | 1 | INST-01 | T-68-04 | Display-only text, no execution | unit | `cd frontend && npx vitest run --reporter=verbose -t "install"` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No Wave 0 test scaffolding was needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Agent session starts via nvm-installed claude | DISC-01 | Requires nvm environment | Install claude via nvm, launch AgentHub, start session |
| Tailscale detected via Homebrew install | DISC-03 | Requires Homebrew Tailscale | Install Tailscale via brew, verify health check passes |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-12

---

## Validation Audit 2026-04-12

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

### Coverage Notes

- **DISC-01**: 5 tests cover PATH augmentation across package managers (cargo, flatpak, volta, local bin, skip nonexistent). All pass.
- **DISC-02**: Covered by `TestAugmentServicePath_AddsExistingDirs` and `_PrependsNotAppends` — these verify `os.Setenv("PATH", ...)` modifies the process environment, which Go's `exec.LookPath` reads. A separate child-process inheritance test is unnecessary since Go guarantees environment inheritance.
- **DISC-03**: Tailscale paths are part of the candidate list (`/usr/local/bin`, `/opt/homebrew/bin` on macOS; system PATH on Linux; `C:\Program Files\Tailscale` via `platformExtraBins` on Windows). `TestPlatformExtraBins_NonWindows` verifies the platform split. Windows-specific paths are build-tag guarded and compile-verified.
- **INST-01**: `WelcomeTab.test.tsx` asserts the full `brew tap scottkw/agenthub && brew install --cask agenthub` command string via raw source import.
