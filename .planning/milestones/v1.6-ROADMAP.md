# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- ✅ **v1.2 Tailscale-Only Networking** — Phases 14-18 (shipped 2026-03-23)
- ✅ **v1.3 CLI + Daemon** — Phases 19-26 (shipped 2026-03-25)
- ✅ **v1.4 Unified Binary** — Phases 27-29 (shipped 2026-03-25)
- ✅ **v1.5 Bug Fixes & CLI Args** — Phases 30-34 (shipped 2026-03-26)
- 🚧 **v1.6 Terminal Fill Fix v2** — Phase 35 (in progress)

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

<details>
<summary>✅ v1.5 Bug Fixes & CLI Args (Phases 30-34) — SHIPPED 2026-03-26</summary>

- [x] Phase 30: Backend Args Wiring (1/1 plans) — completed 2026-03-26
- [x] Phase 31: CLI Arg Passthrough (1/1 plans) — completed 2026-03-26
- [x] Phase 32: Daemon Startup Performance (2/2 plans) — completed 2026-03-26
- [x] Phase 33: GUI Args Field (1/1 plans) — completed 2026-03-26
- [x] Phase 34: Terminal Fill Fix (1/1 plans) — completed 2026-03-26

</details>

### v1.6 Terminal Fill Fix v2 (In Progress)

**Milestone Goal:** All four CLIs fill the terminal viewport on initial load without requiring a window resize, in both dev and production builds.

- [x] **Phase 35: Terminal Fill Fix v2** - Diagnose and fix initial-load terminal fill for all CLIs (completed 2026-03-26)

## Phase Details

### Phase 35: Terminal Fill Fix v2
**Goal**: All four CLIs (Claude, Gemini, OpenCode, Codex) fill the terminal viewport correctly on first tab activation, without requiring a window resize
**Depends on**: Nothing (standalone fix)
**Requirements**: FILL-01, FILL-02, FILL-03, FILL-04, FILL-05, FILL-06
**Success Criteria** (what must be TRUE):
  1. Opening a Claude CLI session shows a terminal that fills the full panel width immediately, with no blank right-side gap
  2. Opening a Gemini CLI session shows a terminal that fills the full panel width immediately, with no blank right-side gap
  3. Opening an OpenCode session shows a terminal that fills the full panel width immediately, with no blank right-side gap
  4. Switching to any previously hidden terminal tab fills the full panel width without requiring a resize event
  5. The fix behaves identically in `wails dev` and `wails build` production mode
**Plans:** 1/1 plans complete

Plans:
- [x] 35-01-PLAN.md — rAF retry loop replacing double-rAF one-shot for initial terminal fit

**UI hint**: yes

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-6 | v1.0 | 19/19 | Complete | 2026-03-19 |
| 7-13 | v1.1 | 13/13 | Complete | 2026-03-20 |
| 14-18 | v1.2 | 10/10 | Complete | 2026-03-23 |
| 19-26 | v1.3 | 15/15 | Complete | 2026-03-25 |
| 27-29 | v1.4 | 3/3 | Complete | 2026-03-25 |
| 30-34 | v1.5 | 6/6 | Complete | 2026-03-26 |
| 35 | v1.6 | 1/1 | Complete   | 2026-03-26 |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
*Full v1.2 details: .planning/milestones/v1.2-ROADMAP.md*
*Full v1.3 details: .planning/milestones/v1.3-ROADMAP.md*
*Full v1.4 details: .planning/milestones/v1.4-ROADMAP.md*
*Full v1.5 details: .planning/milestones/v1.5-ROADMAP.md*
