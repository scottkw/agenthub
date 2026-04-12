---
phase: 68
slug: agent-tailscale-discovery-install-instructions
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-11
---

# Phase 68 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + vitest |
| **Config file** | `internal/agents/agents_test.go`, `frontend/vitest.config.ts` |
| **Quick run command** | `go test ./internal/agents/... && cd frontend && npx vitest run --reporter=verbose` |
| **Full suite command** | `go test ./... && cd frontend && npx vitest run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/agents/... && cd frontend && npx vitest run`
- **After every plan wave:** Run `go test ./... && cd frontend && npx vitest run`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 68-01-01 | 01 | 1 | DISC-01 | — | N/A | unit | `go test ./internal/agents/... -run TestAugmentServicePath` | ❌ W0 | ⬜ pending |
| 68-01-02 | 01 | 1 | DISC-02 | — | N/A | unit | `go test ./internal/agents/... -run TestChildProcessInheritsPath` | ❌ W0 | ⬜ pending |
| 68-01-03 | 01 | 1 | DISC-03 | — | N/A | unit | `go test ./internal/agents/... -run TestTailscaleDiscovery` | ❌ W0 | ⬜ pending |
| 68-02-01 | 02 | 1 | INST-01 | — | N/A | unit | `cd frontend && npx vitest run --reporter=verbose -t "install"` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/agents/path_test.go` — tests for augmented PATH discovery across package managers
- [ ] `frontend/src/components/__tests__/WelcomeTab.test.tsx` — test for updated install command

*Existing test infrastructure covers framework setup; only test files need creation.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Agent session starts via nvm-installed claude | DISC-01 | Requires nvm environment | Install claude via nvm, launch AgentHub, start session |
| Tailscale detected via Homebrew install | DISC-03 | Requires Homebrew Tailscale | Install Tailscale via brew, verify health check passes |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
