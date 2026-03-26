# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- ✅ **v1.2 Tailscale-Only Networking** — Phases 14-18 (shipped 2026-03-23)
- ✅ **v1.3 CLI + Daemon** — Phases 19-26 (shipped 2026-03-25)
- ✅ **v1.4 Unified Binary** — Phases 27-29 (shipped 2026-03-25)
- 🚧 **v1.5 Bug Fixes & CLI Args** — Phases 30-34 (in progress)

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

<details>
<summary>✅ v1.4 Unified Binary (Phases 27-29) — SHIPPED 2026-03-25</summary>

- [x] Phase 27: Unified Entrypoint (1/1 plans) — completed 2026-03-25
- [x] Phase 28: CLI Package Removal (1/1 plans) — completed 2026-03-25
- [x] Phase 29: Build System & Verification (1/1 plans) — completed 2026-03-25

</details>

### 🚧 v1.5 Bug Fixes & CLI Args (In Progress)

**Milestone Goal:** Fix terminal rendering and daemon performance regressions; add ability to pass custom arguments to agents from both CLI and GUI.

- [x] **Phase 30: Backend Args Wiring** - Thread args through all Go daemon layers (types → engine → API → client) (completed 2026-03-26)
- [x] **Phase 31: CLI Arg Passthrough** - Parse `--` separator in `cmdNew` and pass trailing tokens to session creation (completed 2026-03-26)
- [x] **Phase 32: Daemon Startup Performance** - Fix status polling latency and service-mode PATH resolution (completed 2026-03-26)
- [ ] **Phase 33: GUI Args Field** - Add args text field to new-session modal with per-agent memory and Wails binding update
- [ ] **Phase 34: Terminal Fill Fix** - Fix terminal viewport sizing on initial load for all CLIs

## Phase Details

### Phase 30: Backend Args Wiring
**Goal**: All Go daemon layers accept and forward `args []string` from API boundary to PTY so no args are silently dropped
**Depends on**: Phase 29 (v1.4 complete)
**Requirements**: ARGS-03
**Success Criteria** (what must be TRUE):
  1. `daemon.CreateRequest` JSON struct includes an `Args` field that survives HTTP serialization round-trip
  2. A session created via the daemon API with args receives those args at the PTY process invocation
  3. All existing callers (GUI, CLI) that pass no args continue to work without change
  4. Go tests cover the full IPC chain with a non-empty args slice
**Plans**: 1 plan
Plans:
- [x] 30-01-PLAN.md — Thread args through all 5 daemon layers + integration tests

### Phase 31: CLI Arg Passthrough
**Goal**: Users can pass extra flags to agents from the CLI using the `--` separator
**Depends on**: Phase 30
**Requirements**: ARGS-01
**Success Criteria** (what must be TRUE):
  1. `agenthub new claude /path -- --model claude-opus-4-5` starts a session with those extra flags visible in the PTY process arguments
  2. Args after `--` are passed as a `[]string` token array, not a raw shell string (no injection risk)
  3. `agenthub new claude /path` with no `--` continues to work as before
  4. Go tests cover the `cmdNew` `--` separator parsing with and without trailing args
**Plans**: 1 plan
Plans:
- [ ] 31-01-PLAN.md — Parse -- separator in runCLI, update cmdNew to forward extraArgs

### Phase 32: Daemon Startup Performance
**Goal**: Session status appears immediately after creation and service-mode agents resolve correctly in user PATH
**Depends on**: Phase 29 (v1.4 complete, independent of Phase 30-31)
**Requirements**: PERF-01, PERF-02, PERF-03
**Success Criteria** (what must be TRUE):
  1. Status indicator updates within 1 second of session creation (not after a 2-second blank period)
  2. `pollSessionStatus` makes its first HTTP call immediately on start, then polls at 500ms intervals
  3. Agents installed via nvm, volta, or Homebrew are found when the daemon runs as a launchd/systemd service
**Plans**: 2 plans
Plans:
- [x] 32-01-PLAN.md — Fix pollSessionStatus timing (poll-first, 500ms interval)
- [x] 32-02-PLAN.md — Add PATH augmentation for service-mode daemon

### Phase 33: GUI Args Field
**Goal**: Users can enter and persist extra arguments per agent in the new-session modal
**Depends on**: Phase 30
**Requirements**: ARGS-02, ARGS-04, ARGS-05
**Success Criteria** (what must be TRUE):
  1. New-session modal shows an args text field below the folder picker
  2. Args entered for an agent are pre-filled next time the same agent is selected
  3. User can clear the pre-filled args with a clear button (clears both UI and stored memory)
  4. Args are passed correctly to the session when the modal is submitted
  5. Wails TypeScript bindings reflect the updated `App.CreateSession` signature
**Plans**: TBD
**UI hint**: yes

### Phase 34: Terminal Fill Fix
**Goal**: Terminal fills the viewport correctly on first tab activation for all CLIs without requiring a window resize
**Depends on**: Phase 29 (v1.4 complete, independent of Phase 30-33)
**Requirements**: TERM-01, TERM-02, TERM-03, TERM-04
**Success Criteria** (what must be TRUE):
  1. Opening a Claude CLI session shows a full-viewport terminal on first activation (no manual resize needed)
  2. Opening a Gemini CLI session shows a full-viewport terminal on first activation (no manual resize needed)
  3. PTY sessions spawn with dimensions matching the container size, not hardcoded 80x24
  4. Switching tabs to a previously hidden terminal does not produce a 1-column or zero-height render
**Plans**: TBD
**UI hint**: yes

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-6 | v1.0 | 19/19 | Complete | 2026-03-19 |
| 7-13 | v1.1 | 13/13 | Complete | 2026-03-20 |
| 14-18 | v1.2 | 10/10 | Complete | 2026-03-23 |
| 19-26 | v1.3 | 15/15 | Complete | 2026-03-25 |
| 27-29 | v1.4 | 3/3 | Complete | 2026-03-25 |
| 30. Backend Args Wiring | v1.5 | 1/1 | Complete    | 2026-03-26 |
| 31. CLI Arg Passthrough | v1.5 | 0/1 | Complete    | 2026-03-26 |
| 32. Daemon Startup Performance | v1.5 | 2/2 | Complete    | 2026-03-26 |
| 33. GUI Args Field | v1.5 | 0/? | Not started | - |
| 34. Terminal Fill Fix | v1.5 | 0/? | Not started | - |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
*Full v1.2 details: .planning/milestones/v1.2-ROADMAP.md*
*Full v1.3 details: .planning/milestones/v1.3-ROADMAP.md*
*Full v1.4 details: .planning/milestones/v1.4-ROADMAP.md*
