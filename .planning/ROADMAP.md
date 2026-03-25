# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- ✅ **v1.2 Tailscale-Only Networking** — Phases 14-18 (shipped 2026-03-23)
- ✅ **v1.3 CLI + Daemon** — Phases 19-26 (shipped 2026-03-25)
- 🔄 **v1.4 Unified Binary** — Phases 27-29 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) — SHIPPED 2026-03-19</summary>

- [x] Phase 1: PTY Foundation (2/2 plans) — completed 2026-03-18
- [x] Phase 2: Session Registry + WebSocket Relay (2/2 plans) — completed 2026-03-18
- [x] Phase 3: Wails Desktop UI (3/3 plans) — completed 2026-03-18
- [x] Phase 4: Web Serving + TLS + Auth (4/4 plans) — completed 2026-03-18
- [x] Phase 5: QR Codes + Status Indicators (6/6 plans) — completed 2026-03-18
- [x] Phase 6: Distribution + Cross-Platform (2/2 plans) — completed 2026-03-19

</details>

<details>
<summary>✅ v1.1 Polish & Build (Phases 7-13) — SHIPPED 2026-03-20</summary>

- [x] Phase 7: Layout Baseline (1/1 plans) — completed 2026-03-19
- [x] Phase 8: Per-Tab Status Bar (2/2 plans) — completed 2026-03-19
- [x] Phase 9: Settings Modal Overhaul (1/1 plans) — completed 2026-03-19
- [x] Phase 10: Per-Tab Font Size (1/1 plans) — completed 2026-03-19
- [x] Phase 11: New-Session Modal (3/3 plans) — completed 2026-03-19
- [x] Phase 12: Tab Rename + Web Dashboard (3/3 plans) — completed 2026-03-20
- [x] Phase 13: Build Script (2/2 plans) — completed 2026-03-20

</details>

<details>
<summary>✅ v1.2 Tailscale-Only Networking (Phases 14-18) — SHIPPED 2026-03-23</summary>

- [x] Phase 14: Tailscale Health Check Infrastructure (2/2 plans) — completed 2026-03-20
- [x] Phase 15: Tailscale TLS + Interface Binding (2/2 plans) — completed 2026-03-20
- [x] Phase 16: Auth Layer Removal (2/2 plans) — completed 2026-03-20
- [x] Phase 17: Dead Code Cleanup (2/2 plans) — completed 2026-03-20
- [x] Phase 18: Frontend Health Modal + Status UI (2/2 plans) — completed 2026-03-22

</details>

<details>
<summary>✅ v1.3 CLI + Daemon (Phases 19-26) — SHIPPED 2026-03-25</summary>

- [x] Phase 19: Daemon Core / Engine + IPC (2/2 plans) — completed 2026-03-23
- [x] Phase 20: Process Separation (2/2 plans) — completed 2026-03-23
- [x] Phase 21: CLI Session + Web Commands (2/2 plans) — completed 2026-03-24
- [x] Phase 22: CLI Attach (2/2 plans) — completed 2026-03-24
- [x] Phase 23: Service Manager Integration (2/2 plans) — completed 2026-03-24
- [x] Phase 24: CLI Polish (2/2 plans) — completed 2026-03-24
- [x] Phase 25: Windows Named Pipe Dial Fix (1/1 plans) — completed 2026-03-24
- [x] Phase 26: Graceful GUI Startup Failure (2/2 plans) — completed 2026-03-24

</details>

### v1.4 Unified Binary (Phases 27-29)

- [x] **Phase 27: Unified Entrypoint** - Single main.go dispatches GUI, CLI commands, and daemon mode (completed 2026-03-25)
- [x] **Phase 28: CLI Package Removal** - `cmd/agenthub-cli/` deleted, all references scrubbed (completed 2026-03-25)
- [ ] **Phase 29: Build System & Verification** - build.sh, CI, and tests validated against unified binary

## Phase Details

### Phase 27: Unified Entrypoint
**Goal**: A single `agenthub` binary dispatches to GUI, all CLI commands, and daemon mode based on args
**Depends on**: Nothing (first phase of v1.4)
**Requirements**: ROUTE-01, ROUTE-02, ROUTE-03, CLI-01, CLI-02, CLI-03, CLI-04
**Success Criteria** (what must be TRUE):
  1. Running `agenthub` with no arguments launches the desktop GUI
  2. Running `agenthub <command>` executes the corresponding CLI action for all 13 commands
  3. Running `agenthub daemon` starts daemon mode (run/install/uninstall/start/stop subcommands work)
  4. Running `agenthub attach <id>` enters interactive PTY mode with raw I/O, detach key, and resize
  5. Running `agenthub --help` prints usage covering both GUI launch and all CLI subcommands
**Plans:** 1/1 plans complete
Plans:
- [x] 27-01-PLAN.md — Copy CLI source + tests to root package, wire unified dispatch in main.go

### Phase 28: CLI Package Removal
**Goal**: The `cmd/agenthub-cli/` package is fully deleted with no dangling references anywhere
**Depends on**: Phase 27
**Requirements**: CLEAN-01, CLEAN-02
**Success Criteria** (what must be TRUE):
  1. The `cmd/agenthub-cli/` directory does not exist in the repository
  2. No file in the repo (docs, CI workflows, build scripts, Go source) references `agenthub-cli`
  3. `go build ./...` succeeds with no import errors after the deletion
**Plans:** 1/1 plans complete
Plans:
- [x] 28-01-PLAN.md — Delete cmd/agenthub-cli/ and scrub references

### Phase 29: Build System & Verification
**Goal**: The build pipeline produces and validates a single unified binary on all platforms
**Depends on**: Phase 28
**Requirements**: BUILD-01, BUILD-02, BUILD-03
**Success Criteria** (what must be TRUE):
  1. `build.sh` compiles one `agenthub` binary that passes GUI, CLI, and daemon dispatch tests
  2. GitHub Actions CI runs the build and test suite against the unified binary with green status
  3. All existing test suites (28+ daemon tests, 16 CLI tests, 7 attach tests) pass with `-race` flag
**Plans:** 1 plan
Plans:
- [ ] 28-01-PLAN.md — Delete cmd/agenthub-cli/ and scrub references

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-6 | v1.0 | 19/19 | Complete | 2026-03-19 |
| 7-13 | v1.1 | 13/13 | Complete | 2026-03-20 |
| 14-18 | v1.2 | 10/10 | Complete | 2026-03-23 |
| 19-26 | v1.3 | 15/15 | Complete | 2026-03-25 |
| 27. Unified Entrypoint | v1.4 | 1/1 | Complete    | 2026-03-25 |
| 28. CLI Package Removal | v1.4 | 1/1 | Complete    | 2026-03-25 |
| 29. Build System & Verification | v1.4 | 0/? | Not started | - |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
*Full v1.2 details: .planning/milestones/v1.2-ROADMAP.md*
*Full v1.3 details: .planning/milestones/v1.3-ROADMAP.md*
