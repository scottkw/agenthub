# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- ✅ **v1.2 Tailscale-Only Networking** — Phases 14-18 (shipped 2026-03-23)
- ✅ **v1.3 CLI + Daemon** — Phases 19-26 (shipped 2026-03-25)
- ✅ **v1.4 Unified Binary** — Phases 27-29 (shipped 2026-03-25)
- ✅ **v1.5 Bug Fixes & CLI Args** — Phases 30-34 (shipped 2026-03-26)
- ✅ **v1.6 Terminal Fill Fix v2** — Phase 35 (shipped 2026-03-31)
- ✅ **v1.7 Daemon UX & Branding** — Phases 36-43 (shipped 2026-04-03)
- ✅ **v1.8 GitHub Distribution & CI/CD** — Phases 44-48 (shipped 2026-04-06)
- ✅ **v1.9 Remote Sessions & App Polish** — Phases 49-54 (shipped 2026-04-08)
- ✅ **v1.10 Collapsible Sidebar Navigation** — Phases 55-56 (shipped 2026-04-08)
- ✅ **v1.11 Local Network & UX Polish** — Phases 57-62 (shipped 2026-04-10)
- ✅ **v1.12 UI/UX Polish** — Phases 63-66 (shipped 2026-04-11)
- ✅ **v1.13 Cross-Platform Fixes & UX** — Phases 67-69 (shipped 2026-04-12)
- ✅ **v1.14 UI Polish** — Phases 70-73 (shipped 2026-04-14)
- ✅ **v2.0 Multi-Client, CLI UX & TUI Mode** — Phases 74-78 (shipped 2026-04-16)
- ✅ **v2.1 Bug Fixes & UX** — Phases 79-82 (shipped 2026-04-17)
- ✅ **v3.0 Session Lifecycle & TUI Polish** — Phases 83-86 (shipped 2026-04-19)
- ✅ **v3.1 Security Hardening** — Phases 87-90 (shipped 2026-05-03, closes Issue #35)
- ✅ **v3.2 Plugin Suite** — Phases 92-99 (shipped 2026-05-12, closes Issue #36; Phase 91 deferred to a future milestone — see `.planning/deferred/91-distribution-pipeline-followups/`)
- ✅ **v3.3 Shell Sessions & Polish** — Phases 100-108 (shipped 2026-05-17, closes Issues #44 + #45)
- ✅ **v3.3.1 Bug Sweep** — Phases 109-117 (shipped 2026-05-19, closes Issues #52, #54, #55, #56, #57, #58, #60)
- ✅ **v3.4 File Browser (Read-Only) + TUI Parity** — Phases 118-122 (shipped 2026-05-21, closes Issues #62 + v3.4 slice of #64)
- ✅ **v3.5 File Browser — Write Operations & Editor** — Phases 123-128 (shipped 2026-06-15, closes Issues #63, #64; umbrella #24 pending two-machine UAT)
- ✅ **v3.5.1 Remote Browse Completion + Release-Gate Fix** — Phases 129-130 (shipped 2026-06-16, closes Issues #86, #83, #87; retired umbrella #24)
- 🚧 **v3.6 Hub (Session Grid / Control Room)** — Phases 131-135 (in progress, closes Issue #78)

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
- [x] Phase 34: PATH Augmentation (1/1 plans) — completed 2026-03-26

</details>

<details>
<summary>✅ v1.6 Terminal Fill Fix v2 (Phase 35) — SHIPPED 2026-03-31</summary>

- [x] Phase 35: Terminal Fill Retry Loop (1/1 plans) — completed 2026-03-31

</details>

<details>
<summary>✅ v1.7 Daemon UX & Branding (Phases 36-43) — SHIPPED 2026-04-03</summary>

- [x] Phase 36: App Icon (1/1 plans) — completed 2026-04-02
- [x] Phase 37: Splash Screen (1/1 plans) — completed 2026-04-02
- [x] Phase 38: Hostname in Session API (1/1 plans) — completed 2026-04-02
- [x] Phase 39: Web Terminal Status Bar (1/1 plans) — completed 2026-04-02
- [x] Phase 40: CLI Attach Banner (1/1 plans) — completed 2026-04-02
- [x] Phase 41: Daemon Manager Panel (1/1 plans) — completed 2026-04-02
- [x] Phase 42: Daemon Shutdown + Tray Icons (2/2 plans) — completed 2026-04-02
- [x] Phase 43: Tray Dynamic Session Menu (2/2 plans) — completed 2026-04-03

</details>

<details>
<summary>✅ v1.8 GitHub Distribution & CI/CD (Phases 44-48) — SHIPPED 2026-04-06</summary>

- [x] Phase 44: Go Module Path Rewrite (1/1 plans) — completed 2026-04-04
- [x] Phase 45: Release Please Setup (1/1 plans) — completed 2026-04-04
- [x] Phase 46: Release Artifacts & One-Liner (2/2 plans) — completed 2026-04-05
- [x] Phase 47: Homebrew + WinGet Manifests (2/2 plans) — completed 2026-04-05
- [x] Phase 48: Distribution Workflow (3/3 plans) — completed 2026-04-05

</details>

<details>
<summary>✅ v1.9 Remote Sessions & App Polish (Phases 49-54) — SHIPPED 2026-04-08</summary>

- [x] Phase 49: macOS App Menus (2/2 plans) — completed 2026-04-06
- [x] Phase 50: Tailscale Peer Discovery (3/3 plans) — completed 2026-04-06
- [x] Phase 51: Auto-Update Checker (2/2 plans) — completed 2026-04-06
- [x] Phase 52: Remote Sessions GUI (3/3 plans) — completed 2026-04-07
- [x] Phase 53: CLI Remote Sessions (2/2 plans) — completed 2026-04-07
- [x] Phase 54: Tailscale Onboarding (2/2 plans) — completed 2026-04-07

</details>

<details>
<summary>✅ v1.10 Collapsible Sidebar Navigation (Phases 55-56) — SHIPPED 2026-04-08</summary>

- [x] Phase 55: Sidebar Icons + Layout (2/2 plans) — completed 2026-04-08
- [x] Phase 56: Tab Bar Cleanup (1/1 plans) — completed 2026-04-08

</details>

<details>
<summary>✅ v1.11 Local Network & UX Polish (Phases 57-62) — SHIPPED 2026-04-10</summary>

- [x] Phase 57: Agent Discovery Paths (1/1 plans) — completed 2026-04-08
- [x] Phase 58: Settings Tab Conversion (1/1 plans) — completed 2026-04-08
- [x] Phase 59: Web Server Auto-Start (1/1 plans) — completed 2026-04-09
- [x] Phase 60: Local Network Fallback (3/3 plans) — completed 2026-04-09
- [x] Phase 61: Frontend webEnabled Seeding (2/2 plans) — completed 2026-04-09
- [x] Phase 62: Tech Debt Cleanup (1/1 plans) — completed 2026-04-10

</details>

<details>
<summary>✅ v1.12 UI/UX Polish (Phases 63-66) — SHIPPED 2026-04-11</summary>

- [x] Phase 63: Sidebar Icon Centering (1/1 plans) — completed 2026-04-10
- [x] Phase 64: Terminal Padding (1/1 plans) — completed 2026-04-11
- [x] Phase 65: Terminal Theming (1/1 plans) — completed 2026-04-11
- [x] Phase 66: Web Server Link UX (1/1 plans) — completed 2026-04-11

</details>

<details>
<summary>✅ v1.13 Cross-Platform Fixes & UX (Phases 67-69) — SHIPPED 2026-04-12</summary>

- [x] Phase 67: Cross-Platform System Tray (2/2 plans) — completed 2026-04-12
- [x] Phase 68: Agent & Tailscale Discovery + Install Instructions (2/2 plans) — completed 2026-04-12
- [x] Phase 69: Settings Scrollable Layout (1/1 plan) — completed 2026-04-12

</details>

<details>
<summary>✅ v1.14 UI Polish (Phases 70-73) — SHIPPED 2026-04-14</summary>

- [x] Phase 70: Sidebar Icon Position Stability (1/1 plans) — completed 2026-04-13
- [x] Phase 71: OpenCode Theming Fix (5/5 plans) — completed 2026-04-13
- [x] Phase 72: UI Contrast Improvement (2/2 plans) — completed 2026-04-14
- [x] Phase 73: Theme Usability Audit (1/1 plan) — completed 2026-04-14

</details>

<details>
<summary>✅ v2.0 Multi-Client, CLI UX & TUI Mode (Phases 74-78) — SHIPPED 2026-04-16</summary>

- [x] Phase 74: Multi-Client Fan-Out (3/3 plans) — completed 2026-04-15
- [x] Phase 75: CLI Status Bar (3/3 plans) — completed 2026-04-15
- [x] Phase 76: TUI Foundation (3/3 plans) — completed 2026-04-15
- [x] Phase 77: TUI Session Operations (4/4 plans) — completed 2026-04-15
- [x] Phase 78: TUI Remote & QR (3/3 plans) — completed 2026-04-15

</details>

<details>
<summary>✅ v2.1 Bug Fixes & UX (Phases 79-82) — SHIPPED 2026-04-17</summary>

- [x] Phase 79: Settings Persistence & Path Browsing (2/2 plans) — completed 2026-04-16
- [x] Phase 80: Tailscale Detection (2/2 plans) — completed 2026-04-16
- [x] Phase 81: Banner Notifications (2/2 plans) — completed 2026-04-16
- [x] Phase 82: Minimize to Tray (2/2 plans) — completed 2026-04-17

</details>

<details>
<summary>✅ v3.0 Session Lifecycle & TUI Polish (Phases 83-86) — SHIPPED 2026-04-19</summary>

- [x] Phase 83: Settings UI Alignment (1/1 plans) — completed 2026-04-19
- [x] Phase 84: Session Auto-Close (3/3 plans) — completed 2026-04-19
- [x] Phase 85: Quit Confirmation Modal (2/2 plans) — completed 2026-04-19
- [x] Phase 86: TUI Visual Polish (3/3 plans) — completed 2026-04-19

</details>

<details>
<summary>✅ v3.1 Security Hardening (Phases 87-90) — SHIPPED 2026-05-03 (v3.1.0, closes Issue #35)</summary>

- [x] **Phase 87: Capability-Based Session Authorization** — completed 2026-04-21
- [x] **Phase 88: WebSocket Handshake Security** — completed 2026-05-02
- [x] **Phase 89: Vendored Terminal Assets + CSP** — completed 2026-05-02
- [x] **Phase 90: Release Pipeline Hardening** — completed 2026-05-03

Distribution follow-ups deferred to a future milestone (see `.planning/deferred/91-distribution-pipeline-followups/91-CONTEXT.md`).

</details>

<details>
<summary>✅ v3.2 Plugin Suite (Phases 92-99) — SHIPPED 2026-05-12 (closes Issue #36)</summary>

> Phase 91 is reserved for the deferred v3.1 distribution-pipeline follow-ups (`.planning/deferred/91-distribution-pipeline-followups/`); v3.2 starts at Phase 92.

- [x] Phase 92: Plugin Settings Foundation (3/3 plans) — completed 2026-05-04
- [x] Phase 93: Vendoring Discipline + Web Parity for Already-Shipping Addons (5/5 plans) — completed 2026-05-04
- [x] Phase 94: Search Addon + Find Bar — Desktop + Web (7/7 plans incl. 2 gap-closure) — completed 2026-05-06
- [x] Phase 95: Web-Links Addon + Security Hardening (6/6 plans) — completed 2026-05-06
- [x] Phase 96: Image Addon + CSP Audit (6/6 plans) — completed 2026-05-07
- [x] Phase 97: Serialize Addon + Save-Session UX (6/6 plans) — completed 2026-05-08
- [x] Phase 98: Progress Addon (P2 — cuttable, default OFF) (5/5 plans) — completed 2026-05-08
- [x] Phase 99: Settings UI Polish + Migration + Final CSP Audit — Release Gate (6/6 plans incl. 1 gap-closure) — completed 2026-05-09

</details>

<details>
<summary>✅ v3.3 Shell Sessions & Polish (Phases 100-108) — SHIPPED 2026-05-17 (closes Issues #44 + #45)</summary>

- [x] Phase 100: Shell Session Backend & Discovery (4/4 plans) — completed 2026-05-13
- [x] Phase 101: Shell Session Surfaces & Web-Share Gating (4/4 plans) — completed 2026-05-13
- [x] Phase 102: Web-Links Polish — mailto + IDN (1/1 plan) — completed 2026-05-13
- [x] Phase 103: Find Bar Dismiss + Test-Env + IIP Polish (inline) — completed 2026-05-12
- [x] Phase 104: Settings Hyperlinked Index (1/1 plan) — completed 2026-05-12
- [x] Phase 105: Deferred v3.2 UAT Re-Run (runbook + UAT executed) — completed 2026-05-17
- [x] Phase 106: Distribution Pipeline Followups (workflow edits) — completed 2026-05-13
- [x] Phase 107: Shell UX Collapse + Binary Path Picker + Clean-Exit Handling (4/4 plans) — completed 2026-05-13
- [x] Phase 108: TUI + CLI shell-entry collapse — parity with Phase 107 GUI (3/3 plans) — completed 2026-05-16

</details>

<details>
<summary>✅ v3.3.1 Bug Sweep (Phases 109-117) — SHIPPED 2026-05-19 (closes Issues #52, #54, #55, #56, #57, #58, #60)</summary>

- [x] Phase 109: Windows daemon named-pipe IPC (Issue #52, PR #53) — IPC-01..06 (completed 2026-05-18)
- [x] Phase 110: Linux PTY natural-exit detection (Issue #57) — PTY-01..04 (completed 2026-05-18)
- [x] Phase 111: Web bridge OSC/DA response consumption — web surface (Issue #54 web side) — WEB-01..03 (completed 2026-05-18)
- [x] Phase 112: WebGL recovery banner rendering (Issue #55) — UI-01, UI-02 (completed 2026-05-18)
- [x] Phase 113: iPad terminal touch-scroll (Issue #56) — UI-03, UI-04 (completed 2026-05-18)
- [x] Phase 114: CI test stability — webserver capability test flake (Issue #58) — TEST-01, TEST-02 (completed 2026-05-18)
- [x] Phase 115: Desktop relay OSC/DA absorption (Issue #60) — WEB-04..06 (completed 2026-05-19)
- [x] Phase 116: Pre-existing test stability (4 tests) — TEST-03..06 (completed 2026-05-19)
- [x] Phase 117: Paper-cuts — TUI defensive + attach clear + WR-03 — PAPER-01..03 (completed 2026-05-19)

</details>

<details>
<summary>✅ v3.4 File Browser (Read-Only) + TUI Parity (Phases 118-122) — SHIPPED 2026-05-21 (closes Issues #62 + v3.4 slice of #64)</summary>

- [x] **Phase 118: FS Sandbox Core + WorkDir Gap + Daemon Routes + Fuzz Corpus + Capability Bit** — FS-01..FS-14 (completed 2026-05-20)
- [x] **Phase 119: WebServer Routes + `files.read` Capability Plumbing** — WEB-01..WEB-05 (completed 2026-05-20)
- [x] **Phase 120: FileBrowserTab.tsx (Desktop + Web)** — UI-01..UI-14 (completed 2026-05-20)
- [x] **Phase 121: TUI Files View** — TUI-01..TUI-10 (completed 2026-05-21)
- [x] **Phase 122: Remote-Session File Browse Wiring (Desktop GUI + TUI)** — REMOTE-01..REMOTE-05 (completed 2026-05-21, inserted mid-milestone after Phase 121 surfaced cross-surface remote-browse parity gap)

</details>

<!-- v3.4 phase details archived to milestones/v3.4-ROADMAP.md -->

<details>
<summary>✅ v3.5 File Browser — Write Operations & Editor (Phases 123-128) — SHIPPED 2026-06-15 (closes Issues #63, #64; umbrella #24 pending two-machine UAT)</summary>

- [x] **Phase 123: TD Cleanup + Write Sandbox Primitives + Daemon Routes** — FSW-01..FSW-12 (completed 2026-06-14)
- [x] **Phase 124: `files.write` Capability + Webserver Write Routes + Web-Share Opt-In** — CAP-01..CAP-10 (completed 2026-06-14)
- [x] **Phase 125: React Editor (CodeMirror 6) — Desktop + Web** — EDIT-01..EDIT-13 (completed 2026-06-14)
- [x] **Phase 126: TUI Write Parity (`$EDITOR` Shell-Out)** — TUIW-01..TUIW-07 (completed 2026-06-15)
- [x] **Phase 127: Web-Share Write Security Hardening** — SEC-01..SEC-07 (completed 2026-06-15)
- [x] **Phase 128: Remote Write Parity + Cross-Surface Integration** — RMW-01..RMW-06 (completed 2026-06-15)

</details>

<!-- v3.5 phase details archived to milestones/v3.5-ROADMAP.md -->

<details>
<summary>✅ v3.5.1 Remote Browse Completion + Release-Gate Fix (Phases 129-130) — SHIPPED 2026-06-16 (closes Issues #86, #83, #87; retired umbrella #24)</summary>

- [x] **Phase 129: Write Concurrency Fix + DNS Error UX** — RACE-01..03, DNS-01..03 (completed 2026-06-16)
- [x] **Phase 130: Remote Browse GUI On-Ramp** — RB-01..05 (completed 2026-06-16)

</details>

<!-- v3.5.1 phase details archived to milestones/v3.5.1-ROADMAP.md -->

### v3.6 Hub (Session Grid / Control Room) — Phases 131-135

- [x] **Phase 131: Hub Foundation + Static Session Cards** — HUB-01..04, CARD-01..06, CARD-08, GRID-01..02, GRID-04..06 (completed 2026-06-16)
- [ ] **Phase 132: Unified Grid + Mini Preview + Named Groups** — CARD-07, GRID-03, GRID-07, GROUP-01..04
- [ ] **Phase 133: Attention + Pulse** — ATTN-01..06
- [ ] **Phase 134: Modal Interaction** — MODAL-01..06
- [ ] **Phase 135: Accessibility Hardening** — A11Y-01..04

## Phase Details

### Phase 131: Hub Foundation + Static Session Cards

**Goal**: Users can navigate to a Hub surface and see all their sessions rendered as live, data-accurate cards in a responsive grouped grid with filter and search
**Depends on**: Backend data gap closed first (Wave 0): app.go SessionInfo + daemon.SessionInfo gain WorkDir, and app.go propagates ViewerCount/ExitCode/Duration/WorkDir — the ROADMAP "fields already exist" assumption was wrong (see 131-RESEARCH.md Critical Data Gap). Plus the existing sidebar navigation pattern.
**Requirements**: HUB-01, HUB-02, HUB-03, HUB-04, CARD-01, CARD-02, CARD-03, CARD-04, CARD-05, CARD-06, CARD-08, GRID-01, GRID-02, GRID-04, GRID-05, GRID-06
**Success Criteria** (what must be TRUE):

  1. User clicks "Hub" in the sidebar and sees a card grid showing all sessions, coexisting with the Sessions panel (not replacing it)
  2. Each card displays the session name (inline-editable), CLI badge, status indicator (shape + icon, not color alone), origin marker (local vs remote + peer host), viewer count, and uptime/duration+exit-code
  3. Stopped/exited cards render visually dimmed with their exit code shown; error-exit cards are not dimmed (they will get attention in Phase 133)
  4. Cards are auto-grouped by working directory; the status filter bar (All / Working / Needs input / Complete / Error / Idle) correctly filters the grid with live counts
  5. The `/` shortcut focuses the search field; typing filters cards by name, CLI, or host; "New session" on the Hub opens the existing create modal
  6. With no sessions, Hub shows an empty-state prompt to create one; the surface renders correctly in both light and dark themes

**Plans**: 5 plans (1 wave-0 backend data gap, 2 wave-1 card/control components, 1 wave-2 surface composition, 1 wave-3 integration + CSS)
Plans:
**Wave 1**

- [x] 131-01-PLAN.md — Wave 0: close the Go data gap (WorkDir on daemon+app SessionInfo; propagate ViewerCount/ExitCode/Duration/WorkDir; App.d.ts)
- [x] 131-02-PLAN.md — Wave 1: InlineSessionName + SessionCard (colorblind-safe status, badge, origin, viewer, uptime, dim)
- [x] 131-03-PLAN.md — Wave 1: HubFilterBar (live-count pills + search + New session) + HubEmptyState (two variants)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 131-04-PLAN.md — Wave 2: SessionCardGrid (group-by-workDir) + HubPanel (filter/search/shortcut/error composition)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 131-05-PLAN.md — Wave 3: TabBar/Sidebar/App.tsx wiring (coexisting Hub tab + poll) + Hub CSS (dark/light tokens, grid, dim, reduced-motion)

**UI hint**: yes

### Phase 132: Unified Grid + Mini Preview + Named Groups

**Goal**: Users can see throttled terminal output snapshots on every card, remote sessions alongside local ones, and organize sessions into named groups via a group sidebar
**Depends on**: Phase 131
**Requirements**: CARD-07, GRID-03, GRID-07, GROUP-01, GROUP-02, GROUP-03, GROUP-04
**Success Criteria** (what must be TRUE):

  1. Every card shows a mini terminal preview rendered from the session's recent output tail — a cheap snapshot, never a live xterm instance; the preview updates on a throttled interval and does not cause perf regression at scale
  2. Remote tailnet peer sessions appear in the same grid as local sessions; each remote card shows the peer hostname as its origin marker
  3. A collapsible group sidebar lists all working-directory groups with per-group running/total counts and a needs-input badge; selecting a group filters the grid to that group's cards
  4. User can create a named group and assign cards to it via drag-and-drop or a per-card "move to group" affordance; group definitions persist in localStorage across app restarts
  5. Group membership survives session-id churn: a session that restarts with the same name and working directory is re-matched to its group automatically; unmatched sessions appear in a default lane

**Plans**: 5 plans (1 wave-0 backend data gap, 2 wave-1 card/control components, 1 wave-2 surface composition, 1 wave-3 integration + CSS)
Plans:

- [x] 131-01-PLAN.md — Wave 0: close the Go data gap (WorkDir on daemon+app SessionInfo; propagate ViewerCount/ExitCode/Duration/WorkDir; App.d.ts)
- [x] 131-02-PLAN.md — Wave 1: InlineSessionName + SessionCard (colorblind-safe status, badge, origin, viewer, uptime, dim)
- [x] 131-03-PLAN.md — Wave 1: HubFilterBar (live-count pills + search + New session) + HubEmptyState (two variants)
- [x] 131-04-PLAN.md — Wave 2: SessionCardGrid (group-by-workDir) + HubPanel (filter/search/shortcut/error composition)
- [x] 131-05-PLAN.md — Wave 3: TabBar/Sidebar/App.tsx wiring (coexisting Hub tab + poll) + Hub CSS (dark/light tokens, grid, dim, reduced-motion)

**UI hint**: yes

### Phase 133: Attention + Pulse

**Goal**: Sessions needing attention float to the top and pulse visibly, with debounced non-jarring reordering, so users can identify blocked or errored sessions at a glance without relying on color
**Depends on**: Phase 131 (cards), Phase 132 (grid + groups)
**Requirements**: ATTN-01, ATTN-02, ATTN-03, ATTN-04, ATTN-05, ATTN-06
**Success Criteria** (what must be TRUE):

  1. A `waiting` session card or an `errored`/non-zero-exit session card displays a pulsing animated highlighted border plus a distinct attention icon — status is distinguishable without relying on color alone
  2. When cards overflow the viewport, attention cards sort above non-attention cards; the reordering is debounced (does not fire on every tick) and position changes animate smoothly without jarring jumps
  3. After a user resolves a `waiting` session from inside the modal, that card's pulse and attention state clear without a page reload
  4. A collapsed group header containing any attention card shows an attention badge; expanding the group reveals which card(s) need attention

**Plans**: 5 plans (1 wave-0 backend data gap, 2 wave-1 card/control components, 1 wave-2 surface composition, 1 wave-3 integration + CSS)
Plans:

- [x] 131-01-PLAN.md — Wave 0: close the Go data gap (WorkDir on daemon+app SessionInfo; propagate ViewerCount/ExitCode/Duration/WorkDir; App.d.ts)
- [x] 131-02-PLAN.md — Wave 1: InlineSessionName + SessionCard (colorblind-safe status, badge, origin, viewer, uptime, dim)
- [x] 131-03-PLAN.md — Wave 1: HubFilterBar (live-count pills + search + New session) + HubEmptyState (two variants)
- [x] 131-04-PLAN.md — Wave 2: SessionCardGrid (group-by-workDir) + HubPanel (filter/search/shortcut/error composition)
- [ ] 131-05-PLAN.md — Wave 3: TabBar/Sidebar/App.tsx wiring (coexisting Hub tab + poll) + Hub CSS (dark/light tokens, grid, dim, reduced-motion)

**UI hint**: yes

### Phase 134: Modal Interaction

**Goal**: Clicking any card opens a full interactive or briefing modal with a shared-element grow/shrink animation, and closing it returns focus cleanly to the originating card
**Depends on**: Phase 131 (cards), Phase 133 (attention state drives modal type)
**Requirements**: MODAL-01, MODAL-02, MODAL-03, MODAL-04, MODAL-05, MODAL-06
**Success Criteria** (what must be TRUE):

  1. Clicking a non-blocked session card expands it into a modal via a grow animation from the card's position; the modal mounts a full interactive TerminalPanel with the same RelayClient used by normal tabs — resize, copy/paste, and scrollback search all work
  2. Clicking a `waiting`/needs-input session card opens a briefing modal showing the real terminal tail (the prompt the agent printed) with a respond affordance; typing a response and submitting sends it to the session
  3. Closing any modal (Escape, close button, or clicking outside) plays a shrink-back animation and returns keyboard focus to the originating card
  4. For a remote session that requires a capability token, the modal interaction uses the existing Phase 122 join-code exchange path — no new remote-access architecture

**Plans**: 5 plans (1 wave-0 backend data gap, 2 wave-1 card/control components, 1 wave-2 surface composition, 1 wave-3 integration + CSS)
Plans:

- [x] 131-01-PLAN.md — Wave 0: close the Go data gap (WorkDir on daemon+app SessionInfo; propagate ViewerCount/ExitCode/Duration/WorkDir; App.d.ts)
- [x] 131-02-PLAN.md — Wave 1: InlineSessionName + SessionCard (colorblind-safe status, badge, origin, viewer, uptime, dim)
- [x] 131-03-PLAN.md — Wave 1: HubFilterBar (live-count pills + search + New session) + HubEmptyState (two variants)
- [ ] 131-04-PLAN.md — Wave 2: SessionCardGrid (group-by-workDir) + HubPanel (filter/search/shortcut/error composition)
- [ ] 131-05-PLAN.md — Wave 3: TabBar/Sidebar/App.tsx wiring (coexisting Hub tab + poll) + Hub CSS (dark/light tokens, grid, dim, reduced-motion)

**UI hint**: yes

### Phase 135: Accessibility Hardening

**Goal**: Every Hub interaction is fully operable by keyboard and safe for colorblind users — attention, status, and motion all carry non-color cues, and animations respect prefers-reduced-motion
**Depends on**: Phase 131, Phase 132, Phase 133, Phase 134 (validates the full surface)
**Requirements**: A11Y-01, A11Y-02, A11Y-03, A11Y-04
**Success Criteria** (what must be TRUE):

  1. All attention and status states are distinguishable without color: each state is uniquely identifiable by its icon shape and/or motion and/or position alone (verified at source level against hex constants — not by eye)
  2. A user can navigate the entire Hub by keyboard: Tab moves between cards, Enter/Space expands a card into its modal, Escape closes the modal and returns focus to the originating card, and the `/` search shortcut is reachable without a mouse
  3. With `prefers-reduced-motion: reduce` set in the OS, pulse and expand/collapse animations are replaced by a static highlighted border + icon — no motion occurs; all information previously conveyed by motion is conveyed by the static fallback
  4. While a modal is open, focus is trapped inside it — Tab cycles through modal controls only, and background cards are not reachable by keyboard

**Plans**: 5 plans (1 wave-0 backend data gap, 2 wave-1 card/control components, 1 wave-2 surface composition, 1 wave-3 integration + CSS)
Plans:

- [x] 131-01-PLAN.md — Wave 0: close the Go data gap (WorkDir on daemon+app SessionInfo; propagate ViewerCount/ExitCode/Duration/WorkDir; App.d.ts)
- [x] 131-02-PLAN.md — Wave 1: InlineSessionName + SessionCard (colorblind-safe status, badge, origin, viewer, uptime, dim)
- [ ] 131-03-PLAN.md — Wave 1: HubFilterBar (live-count pills + search + New session) + HubEmptyState (two variants)
- [ ] 131-04-PLAN.md — Wave 2: SessionCardGrid (group-by-workDir) + HubPanel (filter/search/shortcut/error composition)
- [ ] 131-05-PLAN.md — Wave 3: TabBar/Sidebar/App.tsx wiring (coexisting Hub tab + poll) + Hub CSS (dark/light tokens, grid, dim, reduced-motion)

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
| 35 | v1.6 | 1/1 | Complete | 2026-03-31 |
| 36-43 | v1.7 | 10/10 | Complete | 2026-04-03 |
| 44-48 | v1.8 | 9/9 | Complete | 2026-04-06 |
| 49-54 | v1.9 | 14/14 | Complete | 2026-04-08 |
| 55-56 | v1.10 | 3/3 | Complete | 2026-04-08 |
| 57-62 | v1.11 | 9/9 | Complete | 2026-04-10 |
| 63-66 | v1.12 | 4/4 | Complete | 2026-04-11 |
| 67-69 | v1.13 | 5/5 | Complete | 2026-04-12 |
| 70-73 | v1.14 | 9/9 | Complete | 2026-04-14 |
| 74-78 | v2.0 | 16/16 | Complete | 2026-04-16 |
| 79-82 | v2.1 | 8/8 | Complete | 2026-04-17 |
| 83 | v3.0 | 1/1 | Complete    | 2026-04-19 |
| 84 | v3.0 | 3/3 | Complete    | 2026-04-19 |
| 85 | v3.0 | 2/2 | Complete    | 2026-04-19 |
| 86 | v3.0 | 3/3 | Complete    | 2026-04-19 |
| 87-90 | v3.1 | 18/18 | Complete | 2026-05-03 |
| 92-99 | v3.2 | 44/44 | Complete | 2026-05-12 |
| 100-108 | v3.3 | 18/18 | Complete | 2026-05-17 |
| 109 | v3.3.1 | 1/1 | Complete   | 2026-05-18 |
| 110 | v3.3.1 | 1/1 | Complete   | 2026-05-18 |
| 111 | v3.3.1 | 1/1 | Complete   | 2026-05-18 |
| 112 | v3.3.1 | 1/1 | Complete   | 2026-05-18 |
| 113 | v3.3.1 | 1/1 | Complete   | 2026-05-18 |
| 114 | v3.3.1 | 1/1 | Complete   | 2026-05-18 |
| 115 | v3.3.1 | 1/1 | Complete   | 2026-05-19 |
| 116 | v3.3.1 | 1/1 | Complete   | 2026-05-19 |
| 117 | v3.3.1 | 1/1 | Complete   | 2026-05-19 |
| 118-122 | v3.4 | 20/21 | Complete | 2026-05-21 |
| 123 | v3.5 | 4/4 | Complete   | 2026-06-14 |
| 124 | v3.5 | 5/5 | Complete   | 2026-06-14 |
| 125 | v3.5 | 6/6 | Complete   | 2026-06-14 |
| 126 | v3.5 | 4/4 | Complete   | 2026-06-15 |
| 127 | v3.5 | 4/4 | Complete   | 2026-06-15 |
| 128 | v3.5 | 4/4 | Complete   | 2026-06-15 |
| 129 | v3.5.1 | 3/3 | Complete    | 2026-06-16 |
| 130 | v3.5.1 | 4/4 | Complete    | 2026-06-16 |
| 131 | v3.6 | 5/5 | Complete    | 2026-06-16 |
| 132 | v3.6 | 0/? | Not started | - |
| 133 | v3.6 | 0/? | Not started | - |
| 134 | v3.6 | 0/? | Not started | - |
| 135 | v3.6 | 0/? | Not started | - |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
*Full v1.2 details: .planning/milestones/v1.2-ROADMAP.md*
*Full v1.3 details: .planning/milestones/v1.3-ROADMAP.md*
*Full v1.4 details: .planning/milestones/v1.4-ROADMAP.md*
*Full v1.5 details: .planning/milestones/v1.5-ROADMAP.md*
*Full v1.6 details: .planning/milestones/v1.6-ROADMAP.md*
*Full v1.7 details: .planning/milestones/v1.7-ROADMAP.md*
*Full v1.8 details: .planning/milestones/v1.8-ROADMAP.md*
*Full v1.9 details: .planning/milestones/v1.9-ROADMAP.md*
*Full v1.10 details: .planning/milestones/v1.10-ROADMAP.md*
*Full v1.11 details: .planning/milestones/v1.11-ROADMAP.md*
*Full v1.12 details: .planning/milestones/v1.12-ROADMAP.md*
*Full v1.13 details: .planning/milestones/v1.13-ROADMAP.md*
*Full v1.14 details: .planning/milestones/v1.14-ROADMAP.md*
*Full v2.0 details: .planning/milestones/v2.0-ROADMAP.md*
*Full v2.1 details: .planning/milestones/v2.1-ROADMAP.md*
*Full v3.0 details: .planning/milestones/v3.0-ROADMAP.md*
*Full v3.1 details: .planning/milestones/v3.1-phases/*
*Full v3.2 details: .planning/milestones/v3.2-ROADMAP.md*
*Full v3.3 details: .planning/milestones/v3.3-ROADMAP.md*
*Full v3.3.1 details: .planning/milestones/v3.3.1-ROADMAP.md*
*Full v3.4 details: .planning/milestones/v3.4-ROADMAP.md*
*Full v3.5 details: .planning/milestones/v3.5-ROADMAP.md*
*Full v3.5.1 details: .planning/milestones/v3.5.1-ROADMAP.md*
