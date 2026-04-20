---
phase: 87
slug: capability-based-session-authorization
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-20
---

# Phase 87 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution of capability-based session authorization.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (backend) + Vitest (frontend) |
| **Config file** | Go: repo-root `go.mod`; Vitest: `frontend/vite.config.ts` |
| **Quick run command** | `go test ./internal/capability/... ./internal/session/... ./internal/web/... -run TestCapability -count=1` |
| **Full suite command** | `go test ./... -count=1 && cd frontend && pnpm test --run` |
| **Estimated runtime** | ~25 seconds quick / ~90 seconds full |

---

## Sampling Rate

- **After every task commit:** Run quick command (capability + session + web packages)
- **After every plan wave:** Run full suite (`go test ./... && pnpm test --run`)
- **Before `/gsd-verify-work`:** Full suite must be green + manual tailnet test per Success Criteria 1
- **Max feedback latency:** 25 seconds for quick, 90 seconds for full

---

## Per-Task Verification Map

*Populated by the planner as tasks are authored. Every task must map to either an `automated` command here or a Wave 0 dependency.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 87-XX-YY | XX | W | REQ-ID | T-87-NN | description | unit/integration | `go test ...` | ⬜ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Test infrastructure that must exist before Wave 1 tasks run. Planner populates based on new packages:

- [ ] `internal/capability/capability_test.go` — sign/verify round-trip, tamper detection, bound-session rejection, read-only enforcement, expiry
- [ ] `internal/capability/store_test.go` — signing key persistence + reload across restart (covers SEC-05)
- [ ] `internal/web/middleware_capability_test.go` — HTTP middleware rejects missing/invalid/wrong-session tokens
- [ ] `internal/web/relay_capability_test.go` — WebSocket upgrade rejects invalid capability; `MsgInput` rejected on read-only cap
- [ ] `internal/session/share_test.go` — explicit grant produces signed URL; un-shared session rejects listing
- [ ] Frontend: `frontend/src/lib/capability.test.ts` — token parsing/persistence + API header injection

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Un-granted tailnet peer cannot enumerate sessions | SEC-01 / SC-1 | Requires second tailnet machine; automated tests use in-process HTTP | From peer machine: `curl http://<host>.tailnet.ts.net:<port>/api/sessions` must return 401 |
| Capability-bearing URL opens in external browser | SEC-02 / SC-3 | End-to-end manual browser test | Grant share from CLI, paste URL in browser on second device, confirm session loads |
| Daemon restart preserves existing shared URLs | SEC-05 / SC-5 | Multi-process lifecycle test | Share session → note URL → kill+restart daemon → reload URL → must still authenticate |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags (`-watch`, `--watchAll`, etc.)
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
