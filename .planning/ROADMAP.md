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
- ✅ **v3.6 Hub (Session Grid / Control Room)** — Phases 131-135 (shipped 2026-06-19, closes Issue #78)
- ✅ **v4.0 Hub-First Consolidation & UI/UX Overhaul** — Phases 136-150 (shipped 2026-06-23, closes #51, #65, #68, #69, #96, #97, #98, #100, #101; #99 not-planned; 151 cancelled)
- 🚧 **v4.1 Session Chat** — Phases 151-163 (active, started 2026-06-25, closes #79, #108, #109)

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

<details>
<summary>✅ v3.6 Hub (Session Grid / Control Room) (Phases 131-135) — SHIPPED 2026-06-19 (closes Issue #78)</summary>

- [x] Phase 131: Hub Foundation + Static Session Cards (5/5 plans) — completed 2026-06-16
- [x] Phase 132: Unified Grid + Mini Preview + Named Groups (5/5 plans) — completed 2026-06-16
- [x] Phase 133: Attention + Pulse (5/5 plans) — completed 2026-06-17
- [x] Phase 134: Modal Interaction (8/8 plans) — completed 2026-06-18
- [x] Phase 135: Accessibility Hardening (3/3 plans) — completed 2026-06-19

</details>

<!-- v3.6 phase details archived to milestones/v3.6-ROADMAP.md -->

<details>
<summary>✅ v4.0 Hub-First Consolidation & UI/UX Overhaul (Phases 136-150) — SHIPPED 2026-06-23 (closes #51, #65, #68, #69, #96, #97, #98, #100, #101; #99 not-planned)</summary>

- [x] Phase 136: TUI Removal (2/2 plans) — completed 2026-06-19
- [x] Phase 137: Share Modal & Cap Model (3/3 plans) — completed 2026-06-20
- [x] Phase 138: Hub-First Navigation (4/4 plans) — completed 2026-06-20
- [x] Phase 139: Card Rendering & Tab Strip (4/4 plans) — completed 2026-06-21
- [x] Phase 140: UI-Spec Gate (2/2 plans) — completed 2026-06-21
- [x] Phase 141: Redesign Implementation (9/9 plans) — completed 2026-06-21
- [x] Phase 142: Hub & Settings Redesign Polish (4/4 plans) — completed 2026-06-21
- [x] Phase 143: Regression Test Program (4/4 plans) — completed 2026-06-22
- [x] Phase 144: Daemon Styled-Tail Race Fix (1/1 plan) — completed 2026-06-22
- [x] Phase 145: Windows Files Test Fixes (3/3 plans) — completed 2026-06-22
- [x] Phase 146: Open Session Capability Bug (5/5 plans) — completed 2026-06-22
- [x] Phase 147: In-App Help Page (4/4 plans) — completed 2026-06-23
- [x] Phase 148: Session Tab Chevron (1/1 plan) — completed 2026-06-23
- [x] Phase 149: Google Antigravity Agent (3/3 plans) — completed 2026-06-23
- [x] Phase 150: Shell-Sharing Warning Toggle (3/3 plans) — completed 2026-06-23
- [~] Phase 151: Terminal Font Zoom Shortcuts — CANCELLED 2026-06-23 (already shipped via Phase 134; #99 closed)

</details>

<!-- v4.0 phase details archived to milestones/v4.0-ROADMAP.md -->

### v4.1 Session Chat (Phases 151-163) — ACTIVE

- [x] **Phase 151: Message Schema + ChatStore** - Daemon-side JSONL chat store with history replay and Markdown export endpoints (completed 2026-06-25)
- [x] **Phase 152: Relay Protocol + Identity + Presence** - Frame-type extension, TailnetID/alias attribution, presence and typing indicators (completed 2026-06-26)
- [x] **Phase 153: @session PTY Bridge** - Sanitized one-way PTY injection with RW-cap gate and confirm step (completed 2026-06-26)
- [x] **Phase 154: Desktop Chat UI** - ChatPanel.tsx inside session modal with mentions, badges, safe Markdown rendering (completed 2026-06-26)
- [x] **Phase 155: Web-Share Chat UI + Cross-Surface Parity Gate** - Web SPA chat wiring, Markdown export, Playwright parity verification (completed 2026-06-27)
- [x] **Phase 156: Install Links & Distribution** - Linux install.sh, correct winget ID, winget catalog first-submission automation (completed 2026-06-27)
- [x] **Phase 157: Terminal Screen-Share Semantics (Issue #109)** - Host-authority PTY arbiter, guests honor server resize + CSS scale-to-fit, fixes cross-viewer terminal garble (completed 2026-06-27)
- [x] **Phase 158: Chat Affordance Polish** - Toggle/Send overlap fix (CHAT-FIX-01) + terminal-tab chat parity via TerminalChatHost (CHAT-PARITY-01); UAT-discovered during v4.1 (completed 2026-06-27)
- [x] **Phase 159: Web-Share Chat Parity** - Redirect shared `/sessions/{id}?cap=` → chat-capable `/app/` SPA so remote web guests get chat; closes the real PARITY-01 gap (155 verified only `/app/`); pre-unblocks Tailscale Funnel milestone (added 2026-06-27) (completed 2026-06-27)
- [x] **Phase 160: v4.1 Chat Closeout** - NOTIF-01 Hub-card unread-badge wiring + 153/154/156 tech-debt closeout per milestone audit; **+ terminal bottom empty-space (xterm row-quantization, pre-existing/global — deferred from 159 UAT 2026-06-27)** (added 2026-06-27) (completed 2026-06-28)
- [x] **Phase 161: Chat-Sidebar Alias Control** - User can set their chat display name from the shared ChatPanel sidebar (GUI tab, Hub modal, AND web-share guest via the 159 redirect) — surfaces the already-built Phase 152 alias backend (`MsgAliasSet`/AliasStore) that never got a UI; cross-surface parity by shared component (added 2026-06-27) (completed 2026-06-28)
- [x] **Phase 162: Settings Polish — Terminal Plugins jump link (#108)** - Move the "Plugins" Settings jump link to last position and rename to "Terminal Plugins" (label + section header), anchor id stable; independent of chat work (added 2026-06-27) (completed 2026-06-28)
- [x] **Phase 163: Read-Only Guest Chat Posting (D-06 reconciliation)** - RO-cap guests can POST chat (all surfaces via shared ChatPanel) while `@session` inject + PTY input stay RO-gated; reverses the SEC-01 RO chat-send gate (Phase 154) per user decision 2026-06-27. Surfaced by Phase 159 live UAT (Test 3). Code+tests done & verified (10/10 must-haves); human UAT 2/2 pass (ROCHAT-01/02 live e2e chromium-green + SEC-RO-01 via 163-SECURITY.md, threats_open: 0) (execution 2026-06-28) (completed 2026-06-29)
- [ ] **Phase 164: Web-Share Chat Layout Polish — message-header overflow fix + resizable chat width** - CHAT-LAYOUT-01: web peer's raw `authorID` (nodekey:…) renders un-truncated in the `.chat-msg__tailnet-id` header label → horizontal scroll on web-share `/app/`; designed WEBCHAT-06 ellipsis doesn't engage on that surface. CHAT-LAYOUT-02: resizable chat width. Both in shared ChatPanel/ChatMessage. Discovered during Phase 161 UAT (added 2026-06-28)

> **Closeout ordering note (2026-06-27, updated 2026-06-28):** Phase 160 ("Chat Closeout") closes the *originally-scoped* v4.1 chat gaps (NOTIF-01 + 153/154/156 tech debt). Phases 161 (alias UI), 162 (#108), 163 (RO-can-chat), and 164 (web-share chat layout polish) are scope added after 160; the milestone-close audit (`/gsd-audit-milestone`) now runs after **164** (was 163; 164 added 2026-06-28 from Phase 161 UAT). 160 keeps its number to avoid breaking committed Phase-160 references in the 159 plan/research.

### Progress: v4.1 Session Chat

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 151. Message Schema + ChatStore | 3/3 | Complete    | 2026-06-25 |
| 152. Relay Protocol + Identity + Presence | 6/6 | Complete    | 2026-06-26 |
| 153. @session PTY Bridge | 3/3 | Complete    | 2026-06-26 |
| 154. Desktop Chat UI | 6/6 | Complete   | 2026-06-26 |
| 155. Web-Share Chat UI + Cross-Surface Parity Gate | 6/6 | Complete    | 2026-06-27 |
| 156. Install Links & Distribution | 3/3 | Complete   | 2026-06-27 |
| 157. Terminal Screen-Share Semantics (Issue #109) | 5/5 | Complete    | 2026-06-27 |
| 158. Chat Affordance Polish (CHAT-FIX-01, CHAT-PARITY-01) | 2/2 | Complete    | 2026-06-27 |
| 159. Web-Share Chat Parity (WEBCHAT-01..06) | 5/5 | Complete — live UAT M-31 done 2026-06-27 (3/4 pass; Test 3 RO-chat gap → Phase 163) | 2026-06-27 |
| 160. v4.1 Chat Closeout (NOTIF-01 + tech debt) | 5/5 | Complete    | 2026-06-28 |
| 161. Chat-Sidebar Alias Control (ALIAS-UI-01/02) | 4/4 | Complete   | 2026-06-28 |
| 162. Settings Polish — Terminal Plugins jump link (#108) | 1/1 | Complete   | 2026-06-28 |
| 163. Read-Only Guest Chat Posting (ROCHAT-01/02, SEC-RO-01) | 3/3 | Complete — verified 10/10; human UAT 2/2 pass (ROCHAT live e2e + SEC-RO-01 163-SECURITY.md) | 2026-06-29 |
| 164. Web-Share Chat Layout Polish (CHAT-LAYOUT-01/02) | 1/2 | In Progress|  |

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
*Full v3.6 details: .planning/milestones/v3.6-ROADMAP.md*
*Full v4.0 details: .planning/milestones/v4.0-ROADMAP.md*

### Phase 158: Chat affordance polish: fix toggle/Send overlap + add chat to terminal tab

**Goal:** Two chat-affordance defects found during v4.1 UAT are resolved: (1) the chat toggle no longer covers the composer's Send button when the drawer is open, and (2) the chat affordance (toggle + ChatPanel) is reachable from the raw session terminal tab, not only the Hub interactive modal and web-share view — closing a cross-surface parity gap.
**Requirements**: CHAT-FIX-01 (toggle/Send overlap), CHAT-PARITY-01 (terminal-tab chat affordance; upstream PARITY-01, D-02)
**Depends on:** Phase 157
**Plans:** 2/2 plans complete

**Success Criteria** (what must be TRUE):

  1. With the chat drawer open in the Hub interactive modal, the chat toggle button does not overlap or obscure the composer's Send/Inject button — the toggle relocates clear of the 360px drawer (e.g. left of the drawer edge) while `chat-panel--open`, and remains the affordance that closes the drawer. Verified visually against the UAT screenshot scenario.
  2. From a live session opened in a direct terminal **tab** (not via the Hub card modal), the user can open the same chat drawer — a chat toggle is present on the tab and toggles a working `ChatPanel` (overlay mode, no PTY resize, consistent with D-02). Cross-surface parity: GUI tab now matches the Hub modal and web-share surfaces.
  3. The existing Hub-modal and web-share chat paths are unchanged (no regression); the bug fix is CSS-scoped and the tab integration reuses `ChatPanel` without forking it.

**Context (root cause, confirmed during UAT):**

- BUG: `.hub-modal__chat-toggle` is `position:absolute; bottom:12px; right:12px; z-index:6` (style.css:6012). The drawer `.chat-panel` is `width:360px; right:0; z-index:5`. Toggle (higher z-index) floats over the composer's Send button at the drawer's bottom-right when open. Likely fix: `right:372px` (or equivalent) under `.chat-panel--open`.
- PARITY: chat toggle + `ChatPanel` are mounted only in `HubInteractiveModal.tsx:83-101` and `WebShareSessionView.tsx`. The raw terminal tab (`App.tsx:1701-1737`: `TerminalPanel` + `StatusBar`) has neither. Add the affordance there using the same overlay pattern.

Plans:
**Wave 1**

- [x] 158-01-PLAN.md — CSS bug fix: relocate chat toggle clear of the open drawer so it no longer covers Send (CHAT-FIX-01) [wave 1]

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 158-02-PLAN.md — Terminal-tab chat parity: extract TerminalChatHost (TerminalPanel + overlay ChatPanel + toggle), wire into App.tsx tab (CHAT-PARITY-01) [wave 2, depends 158-01]

### Phase 159: Web-Share Chat Parity — remote web guests get chat

**Goal:** A remote collaborator who opens a web-share link can use session chat — the same ChatPanel, toggle, unread/mention badge, and presence as the desktop tab and Hub modal. Closes the REAL cross-surface parity gap: Phase 155 verified PARITY-01 only on the `/app/` React SPA, but the share flow hands out `/sessions/{id}?cap=` (the raw vanilla-JS `terminal.js` viewer, which has no chat UI and silently discards chat frames 0x30–0x34). Release-blocking parity; also a prerequisite for the upcoming Tailscale Funnel session-sharing milestone (web chat must work for funnel-shared sessions).

**Approach (decided 2026-06-27):** Redirect the shared `/sessions/{id}?cap=TOKEN` link to the chat-capable `/app/?session={id}&cap={token}` React SPA (WebShareSessionView), rather than re-building chat in vanilla JS. The SPA already carries ChatPanel (Phase 155) + the Phase 157 host-authority resize/scale semantics (157-04). Server change in `internal/webserver/server.go` (the `/sessions/{id}` route / `handleTerminalPage`). Planning MUST: (a) honor Chesterton's Fence — establish why the lightweight raw viewer exists before retiring it for shared sessions, and decide whether any non-chat path still needs it; (b) preserve cap-token + session-id through the redirect (URL-encoded), for both read and write caps; (c) keep RO caps as full chat participants (D-06).

**Requirements**: WEBCHAT-01 (remote guest reaches a chat-capable surface via redirect), WEBCHAT-02 (cross-surface parity verified on the ACTUALLY-SHARED link, not `/app/` directly), PARITY-01 (upstream — finally honored on the shared surface)
**Depends on:** Phase 158
**Plans:** 1/1 plans complete

Plans:

- [x] 159-01-PLAN.md — Redirect handleTerminalPage (/sessions/{id}?cap= → /app/ SPA, 302) + Go tests + TESTING.md regression registration (M-31)

**Root-cause evidence (Explore, 2026-06-27):** share URL built at `internal/daemon/api.go:755-760` (`/sessions/{id}?cap=`); `web/assets/terminal.js:1-26` defines only 0x01/0x02/0x10/0x11/0x12 — no chat; the `/app/` route at `internal/webserver/server.go:568-580` is open and reachable but never linked from the share flow; `handleWSSRelay` (`server.go:1047-1065`) already relays MsgChatSend/Typing/Presence to web guests — only the raw viewer's missing UI drops them.

### Phase 160: v4.1 Chat Closeout — NOTIF-01 unread badge + 153/154/156 tech debt

**Goal:** Close the remaining v4.1 chat gaps so the milestone can ship. (1) **NOTIF-01** — the Hub session-card unread badge is dead-wired: `SessionCard` renders `<ChatBadge>` but `SessionCardGrid` never threads `unreadCount`/`hasChatMention`, and unread accrual lives in `ChatPanel`, which only mounts while the modal is open — so a backgrounded session has no unread source and the card badge can never light. Thread the prop AND provide a session-scoped unread signal for closed-modal sessions. (2) Clear the minor tech debt flagged in the v4.1 milestone audit: 153 IN-02/IN-04, 154 NOTIF-02 traceability row, 156 WR-01/02/03.

**Requirements**: NOTIF-01 (Hub-card unread badge — finally wired), plus tech-debt closeout (153 IN-02/IN-04, 154 NOTIF-02, 156 WR-01/02/03)
**Depends on:** Phase 159
**Plans:** 5/5 plans complete

Plans:

- [x] 160-01-PLAN.md — NOTIF-01 background unread source: `useChatUnreadListeners` hook (read-only relay WS per backgrounded session) + test [NOTIF-01]
- [x] 160-02-PLAN.md — NOTIF-01 prop threading: lift unread out of modal, thread `unreadBySessionId` through SessionCardGrid to both card sites, wire HubPanel `unreadMap` + reset-on-open + hook [NOTIF-01]
- [x] 160-03-PLAN.md — IN-02 regression test: control-only inject → zero PTY writes (`TestInject_ControlOnlyInput`) [IN-02]
- [x] 160-04-PLAN.md — install.sh hardening: WR-01 `grep -F`, WR-03 root `mkdir -p`, + install-sh.test.sh assertions [WR-01, WR-03]
- [x] 160-05-PLAN.md — docs/traceability closeout: IN-04 doc comment, WR-02 Run Command, TESTING.md §2/§4 registration [IN-04, NOTIF-02, WR-02]

**Audit reference:** `.planning/v4.1-MILESTONE-AUDIT.md` (NOTIF-01 = the dead-wiring BLOCKER; Phase 154 VERIFICATION falsely claimed it was wired "via existing session callback" — that callback does not exist).

### Phase 161: Chat-Sidebar Alias Control — user can set their display name

**Goal:** A user can set their own chat display name (alias) from within the chat experience, and it immediately becomes their author name in chat messages and their name in the presence roster, across every chat surface. The control lives in the **shared `ChatPanel` sidebar** (e.g. inline-editable "you" entry in the presence roster, or a "chatting as: <name> ✏️" affordance), NOT in Settings — so the desktop terminal tab, the Hub interactive modal, AND the web-share guest (which reaches `ChatPanel` via the Phase 159 redirect) all get the control from one implementation. Closes the long-standing gap where the Phase 152 alias backend shipped with no way for a user to set their alias.

**Approach (decided 2026-06-27):** Surface the **already-built Phase 152 backend** — do NOT rebuild it. `MsgAliasSet` (0x34), `AliasStore` (`~/.config/agenthub/aliases.json`), `ValidateAlias` (≤32 runes, no control chars), and the `alias` field on `ChatMessage`/`PresenceEntry` already exist; both the relay (desktop owner, `local:local`) and webserver (web guest, `tailnetID:web`) server paths already accept `MsgAliasSet`, persist via `AliasSetFn`, and re-broadcast presence. `encodeAliasSetFrame()` already exists in `relayClient.ts` but is only called in tests. Planning MUST: (a) decide whether the desktop-owner path sends `MsgAliasSet` over the existing relay client (preferred — reuses the wire path, no new Wails bindings) or needs a `GetAlias`/`SetAlias` Wails binding (`app.go` + engine) — confirm against live client wiring; (b) make the control clearly communicate the alias is a **global** display name (per `personKey`), not per-session; (c) keep the shared `ChatPanel` un-forked so all surfaces inherit it (cross-surface parity by construction); (d) show the user's current/default alias (Tailnet computed name for guests, owner default for desktop) as the pre-filled value.

**Requirements**: ALIAS-UI-01 (user sets their alias from the shared chat sidebar; available on GUI tab, Hub modal, and web-share guest — cross-surface parity via shared `ChatPanel`), ALIAS-UI-02 (the set alias persists via the Phase 152 `AliasStore`/`MsgAliasSet` path and immediately updates the user's chat author name + presence-roster name for all participants; respects `ValidateAlias`)
**Depends on:** Phase 159 (web-share guests reach `ChatPanel` only via the 159 redirect; also Phase 155 `ChatPanel` + Phase 152 alias backend)
**Plans:** 4/4 plans complete

Plans:

- [x] 161-01-PLAN.md — Backend self-identity frame (MsgSelf 0x37 on both server paths for web pre-fill) [Wave 1]
- [x] 161-02-PLAN.md — relayClient wire layer (sendAliasSet + MsgSelf parse/onSelf) [Wave 1]
- [x] 161-03-PLAN.md — Shared ChatPanel alias header control (validateAlias mirror, RO-enabled, pre-fill) [Wave 2]
- [x] 161-04-PLAN.md — Cross-surface alias-propagation e2e + TESTING.md consolidation + live human-verify [Wave 3]

**Backend-ready evidence (Explore, 2026-06-27):** `internal/daemon/alias_store.go` (AliasStore, aliases.json), `internal/relay/protocol.go:84` (`MsgAliasSet` 0x34) + `:95/:260` (`PresenceEntry.alias`/`ChatMessage.alias`), relay dispatch `internal/relay/server.go:357-372`, webserver dispatch `internal/webserver/server.go:1186-1201`, `frontend/src/lib/relayClient.ts:94-99` (`encodeAliasSetFrame`, currently test-only). GAP: no Wails `GetAlias`/`SetAlias` binding, no UI calls `encodeAliasSetFrame`, no Settings/sidebar control. Web surface has no Settings page — which is exactly why the chat sidebar (shared component) is the right home over a Settings Profile section.

### Phase 162: Settings Polish — Terminal Plugins jump link (#108)

**Goal:** Resolve GitHub issue #108. The Settings "Plugins" jump link currently sits *first* in the sticky jump bar while its section renders *last* — clicking the first link jumps to the bottom. And the label "Plugins" misleads (these are terminal/shell plugins). Fix both: move the jump link to the **last** position (matching render order) and rename it to **"Terminal Plugins"** (jump-bar label + section header), keeping the `settings-plugins` anchor id stable so scroll-spy/jump behavior keeps working. Shared React frontend, so the fix applies to both GUI and web surfaces (cross-surface parity).

**Approach:** Per #108: in `frontend/src/components/SettingsJumpBar.tsx:21-29` move `{ label: 'Plugins', id: 'settings-plugins' }` from first to last in `SETTINGS_JUMP_LINKS` and rename label to `Terminal Plugins` (update/remove the lines 11-14 block comment documenting the intentional first-position placement); in `frontend/src/components/PluginsSection.tsx:289` rename the `<h3 id="settings-plugins">` text to `Terminal Plugins` (keep the id). `SettingsTab.tsx` already renders `PluginsSection` last — no reorder there. Update affected tests (SettingsTab/SettingsSearch/jump-bar specs) that assert label text or link order.

**Requirements**: SETTINGS-UI-01 (Plugins jump link moved to last + renamed "Terminal Plugins" on label and section header; `settings-plugins` anchor unchanged; jump/scroll-spy still works; affected tests updated) — closes #108
**Depends on:** Nothing (independent of chat work; pure Settings-page change)
**Plans:** 1/1 plans complete

Plans:

- [x] 162-01-PLAN.md — Move plugins jump link to last + rename "Terminal Plugins" (label + h3), keep `settings-plugins` anchor, update hyperlinked-index test, run frontend gate

**Source:** GitHub issue #108 (label: bug). Affected: `SettingsJumpBar.tsx:21-29` (+ comment 11-14), `PluginsSection.tsx:289`, `SettingsTab.tsx:904-905` (no change needed), plus label/order assertions in Settings specs.

### Phase 163: Read-Only Guest Chat Posting (D-06 reconciliation)

**Goal:** RO-cap guests (web-share, Hub modal, desktop tab) can **post chat messages** and appear in presence/typing, while `@session` inject and PTY/terminal input remain RO-gated. Reverses the SEC-01 RO chat-send gate (Phase 154) and reconciles the conflict between **D-06** ("RO clients are full chat participants", Phase 152) and **SEC-01/PITFALLS** ("RO read chat, cannot post", Phase 154/155). Surfaced by Phase 159 live UAT (`159-UAT.md` Test 3 / Gaps); user-decided 2026-06-27: *"RO users can participate in chat but cannot @ the session to inject a prompt."*

**Approach (decided 2026-06-27):**

- **Server:** remove the RO gate in `internal/relay/hub.go` `HandleChatSend` (`ErrChatReadOnly`) + the webserver `MsgChatSend` read-pump path. **KEEP** `HandleInject` (`ErrReadOnly`) and the `MsgInput` PTY gate untouched — `@session` inject and terminal input stay RO-blocked.
- **Frontend:** `frontend/src/components/Hub/ChatPanel.tsx` — enable the chat composer Send when `isReadOnly`; **KEEP** the `@session` press-and-hold inject gesture gated for RO; remove/reword the `chat-composer__readonly-label` "Read only".
- **Tests:** flip the Phase 155 PARITY-01 SC-3 assertions that assert RO-cannot-post; add coverage asserting RO **can** post chat but an RO `@session` attempt yields **zero PTY writes** (regression guard).
- **Docs:** reconcile D-06 vs SEC-01; update `.planning/research/PITFALLS.md` (l.21).
- **Security:** run `/gsd-secure-phase 163` after execution to confirm only `MsgChatSend` was loosened (inject + PTY still gated).

**Requirements**: ROCHAT-01 (RO can post chat across all surfaces via the shared `ChatPanel`), ROCHAT-02 (RO `@session` inject + PTY input remain gated — regression guard), SEC-RO-01 (security re-review confirms only `MsgChatSend` was loosened)
**Depends on:** Phase 159 (redirect lands RO web guests on the chat surface) + Phases 154/155 (code being changed)
**Plans:** 3/3 plans complete

**UI hint:** yes (ChatPanel composer)

Plans:

**Wave 1** *(server + frontend in parallel — no shared files)*

- [x] 163-01-PLAN.md — Server: remove RO chat-send gate in hub.HandleChatSend (single source for relay + webserver paths) + delete ErrChatReadOnly + flip Go RO tests + dual-invariant regression guard (RO chats, inject still ErrReadOnly/zero PTY) [ROCHAT-01, ROCHAT-02, SEC-RO-01] [wave 1]
- [x] 163-02-PLAN.md — Frontend: enable RO Send + remove "Read only" label in shared ChatPanel (GUI tab / Hub modal / web-share guest) while keeping @session inject gesture RO-gated; flip vitest + Playwright SC-3 assertions [ROCHAT-01, ROCHAT-02] [wave 1]

**Wave 2** *(blocked on Wave 1 — cites final test names + counts)*

- [x] 163-03-PLAN.md — Docs: reconcile PITFALLS.md D-06 vs SEC-01 + register flips in TESTING.md Section 2/4 (repo-relative paths; run check-traceability-paths.sh) [ROCHAT-01, ROCHAT-02, SEC-RO-01] [wave 2, depends 163-01 + 163-02]

### Phase 164: Web-Share Chat Layout Polish — message-header overflow fix + resizable chat width

**Goal:** Fix two web-share chat layout issues discovered during Phase 161 live UAT, both in the shared `ChatPanel`/`ChatMessage` so all surfaces (GUI tab, Hub modal, web-share guest) benefit:

1. **CHAT-LAYOUT-01 (bug — message-header overflow):** A web peer's raw `authorID` (e.g. `nodekey:456d9361bab4eb7…`) renders un-truncated as the `.chat-msg__tailnet-id` secondary label, forcing horizontal scroll on the web-share `/app/` surface. The designed WEBCHAT-06 ellipsis (`text-overflow: ellipsis; flex-shrink: 1000` on `.chat-msg__tailnet-id`, `min-width: 0` on `.chat-msg__header`) does NOT engage on that surface — root-cause the unconstrained-width ancestor in the web-share layout and make the header truncate. Ideally a web peer's secondary label should show a friendly Tailscale name rather than the raw node public key (server-side `authorID` concern — scope during planning).
2. **CHAT-LAYOUT-02 (feature — resizable chat width):** Make the chat window resizable width-wise.

**Requirements**: CHAT-LAYOUT-01 (header-overflow truncation / friendly tailnet label), CHAT-LAYOUT-02 (resizable chat width) — finalize during /gsd-plan-phase
**Depends on:** Phase 163
**Plans:** 1/2 plans executed

Source: discovered during Phase 161 UAT (alias-control live verification); deferred here by user decision 2026-06-28. CHAT-LAYOUT-01 is pre-existing (`ChatMessage.tsx` was untouched by Phase 161).

**Wave 1**

- [x] 164-01-PLAN.md — CHAT-LAYOUT-01: header-overflow fix — constrain getRowStyle virtualizer row width (root cause) + render last-6-char authorID fingerprint as the secondary label (shared ChatMessage/ChatPanel)

**Wave 2** *(blocked on Wave 1 — shared ChatPanel.tsx/test files)*

- [ ] 164-02-PLAN.md — CHAT-LAYOUT-02: resizable chat width — left-edge drag handle, localStorage persistence + clamp (~280–640px), single --chat-panel-width custom property across all three surfaces, toggle offset tracks live width, D-02 overlay preserved

---

## Phase Details: v4.1 Session Chat

### Phase 151: Message Schema + ChatStore

**Goal**: The daemon can durably persist, replay, and clean up a session's chat thread.
**Depends on**: Nothing (no existing chat infrastructure; Phase 150 complete)
**Requirements**: PERSIST-01, PERSIST-02, PERSIST-03
**Success Criteria** (what must be TRUE):

  1. The daemon writes each chat message to a JSONL file at `~/.config/agenthub/chats/<sessionID>.jsonl`; after a daemon restart the file is intact and `GET /api/chat/:id/history` returns the full message history.
  2. A participant joining the chat after a restart or late connection receives the complete message history in a single response — no messages are missing.
  3. Calling `KillSession` on a session removes its JSONL file; no orphaned chat files remain after deletion.
  4. After 10,000 messages the hard cap is enforced and `AppendMessage` rejects further writes — the store does not grow unbounded.
  5. The REST export endpoint returns a Markdown document that round-trips every `ChatMessage` field (author, alias, timestamp, body) without data loss.

**Plans**: 3/3 plans complete
**Wave 1**

- [x] 151-01-PLAN.md — ChatMessage schema (protocol.go) + ChatStore core: JSONL persistence, restart-load, Messages() replay, 10k cap, sessionID path-traversal hardening, concurrent-write -race safety

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 151-02-PLAN.md — ChatStore Export (Markdown) + Delete (teardown); SessionEngine chatStores map wired into CreateSession + KillSession (no orphans)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 151-03-PLAN.md — REST history + export endpoints (relay loopback for desktop; capability-gated on webserver for web); TESTING.md regression registration

### Phase 152: Relay Protocol + Identity + Presence

**Goal**: Every participant is identified and their live presence and typing status propagate in real time across the relay.
**Depends on**: Phase 151
**Requirements**: IDENT-01, IDENT-02, PRESENCE-01, PRESENCE-02
**Success Criteria** (what must be TRUE):

  1. Each incoming WebSocket connection is stamped with a `TailnetID` (from `lc.WhoIs` for web clients; `"local"` for the Wails desktop owner) and an alias before any message is stored; both values are visible to all other participants.
  2. A participant can set or change their alias; the updated name propagates to all connected clients within one relay round-trip.
  3. All currently-connected participants see each other's presence (connected / disconnected) update in real time when someone joins or leaves.
  4. A typing indicator appears for a participant within 500 ms of them starting to compose and auto-clears if they stop typing for 5 seconds or disconnect abruptly — the indicator is never stored in the JSONL log.
  5. The desktop Wails owner and a same-machine browser client appear as two distinct, correctly-labelled presence entries — no silent identity merge.

**Plans**: 6/6 plans complete

- [x] 152-01-PLAN.md — Wire-protocol foundation: frame constants 0x30-0x34, presence/typing/alias payload structs + encoders, ValidateAlias (Wave 1)
- [x] 152-02-PLAN.md — Daemon AliasStore: composite-key JSON persistence surviving restart (Wave 1)
- [x] 152-03-PLAN.md — Hub presence roster (refcount collapse) + typing TTL via time.AfterFunc (Wave 2)
- [x] 152-04-PLAN.md — relayClient.ts presence/typing parse + typing/alias encoders (Wave 2)
- [x] 152-05-PLAN.md — Relay path: engine AliasStore wiring + owner identity stamping + alias/typing dispatch (Wave 3)
- [x] 152-06-PLAN.md — Web path: WhoIs identity stamping + cross-surface parity + TESTING.md update (Wave 4)

### Phase 153: @session PTY Bridge

**Goal**: RW-capable participants can inject a prompt into the agent PTY via `@session`; the injection is sanitized and requires deliberate confirmation.
**Depends on**: Phase 152
**Requirements**: MENTION-02, MENTION-03, SEC-01, SEC-02
**Success Criteria** (what must be TRUE):

  1. Sending `@session <text>` as a RW-cap holder causes exactly that text to appear in the session PTY stdin; the chat thread shows a "→ injected into terminal" indicator for all participants.
  2. A RO-cap holder who attempts a `@session` send receives a server-side rejection; no PTY write occurs regardless of any client-side suppression or direct WS frame injection.
  3. A message containing C0 control characters, embedded newlines, or terminal escape sequences (CSI, OSC) is sanitized at the daemon handler — only printable text plus exactly one trailing newline reaches PTY stdin; a unit test corpus covers newline injection, null bytes, and CSI sequences.
  4. The `@session` injection path requires a deliberate confirm step; a single accidental keypress or Enter-on-autocomplete does not trigger a PTY write.

**Plans**: 3/3 plans complete
**Wave 1**

- [x] 153-01-PLAN.md — Protocol verbs + SanitizePTYText sanitizer + SEC-02 corpus (SEC-02)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 153-02-PLAN.md — Hub.HandleInject + relay read-pump + engine wiring + relay tests (MENTION-02, MENTION-03, SEC-01)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 153-03-PLAN.md — Web-share read-pump + adversarial RO-JWT test + TESTING.md (SEC-01)

### Phase 154: Desktop Chat UI

**Goal**: The desktop GUI shows a fully functional chat panel inside the session modal with safe Markdown rendering and unread notifications.
**Depends on**: Phase 153
**Requirements**: CHAT-01, CHAT-02, CHAT-03, CHAT-04, MENTION-01, NOTIF-01, NOTIF-02, SEC-03
**Success Criteria** (what must be TRUE):

  1. A user types a message, presses Enter to send, and sees it appear in the thread with their alias, tailnet ID, and a HH:MM timestamp; hovering the timestamp reveals the full ISO-8601 datetime; Shift+Enter inserts a newline without sending.
  2. Typing `@` opens an autocomplete popover listing current session participants plus the pinned `@session` entry; the user can navigate with arrow keys and confirm with Enter.
  3. Messages that `@mention` the current user are visually distinct from regular messages; both the chat-toggle button and the Hub session card display an unread badge count when new messages arrive while the panel is not in focus.
  4. Pasting `<script>alert(1)</script>` or `<img src=x onerror=alert(1)>` into a message renders completely inert — no script executes and no `onerror` attribute is present in the DOM on either surface.
  5. Day separators appear between messages from different calendar days and remain anchored to the top of the visible viewport as the user scrolls through history.

**Plans**: 6/6 plans complete

**Wave 1** *(parallel — disjoint subsystems)*

- [x] 154-01-PLAN.md — Server-side MsgChatSend (0x31) dispatch wiring: ChatSendPayload + Hub.HandleChatSend (SEC-01 RO gate, no PTY write) + relay & webserver read-pump cases + Go tests
- [x] 154-02-PLAN.md — npm packages (@tanstack/react-virtual, react-textarea-autosize) + relayClient.ts chat/inject frame constants, ChatMessage type, encoders, dispatching callbacks

**Wave 2** *(blocked on 154-02)*

- [x] 154-03-PLAN.md — ChatMessage.tsx (avatar rows, @mention 3-signal, inject indicator, safe Markdown SEC-03) + ChatDaySeparator.tsx

**Wave 3** *(blocked on 154-03 — shared style.css)*

- [x] 154-04-PLAN.md — ChatBadge.tsx (unread + @mention glyph) + MentionPopover.tsx (@session pinned, filterable, keyboard-nav)

**Wave 4** *(blocked on 154-03/154-04)*

- [x] 154-05-PLAN.md — ChatPanel.tsx drawer: own RelayClient subscription, virtualizer + sticky day separators (CHAT-04), late-join scrollback, empty/loading states, unread accrual (D-09)

**Wave 5** *(blocked on 154-05)*

- [x] 154-06-PLAN.md — Composer (CHAT-03 auto-grow, Enter/Shift+Enter, mention popover, press-and-hold inject D-08) + HubInteractiveModal overlay integration (drawer floats over the terminal, no resize — D-02 revised for Issue #109) + SessionCard badge + TESTING.md registration

**UI hint**: yes

### Phase 155: Web-Share Chat UI + Cross-Surface Parity Gate

**Goal**: The web-share surface delivers the identical chat experience and Markdown export is available on both surfaces.
**Depends on**: Phase 154
**Requirements**: EXPORT-01, PARITY-01
**Success Criteria** (what must be TRUE):

  1. A participant on the web-share browser sees the same thread, presence indicators, typing indicators, unread badge, and `@mention` highlights as the desktop GUI owner — no feature drift between surfaces verified by Playwright e2e.
  2. Clicking "Export" on either the desktop or web-share surface downloads a `.md` file with YAML frontmatter and the full message thread in GitHub-compatible Markdown.
  3. A RO-cap web-share viewer cannot post messages or trigger `@session` injection — the server rejects both actions regardless of client behavior; a RW-cap web-share viewer can do both.
  4. The `@session` injection path works identically from both surfaces: same relay frame type, same daemon handler, same "→ injected into terminal" indicator in the thread.

**Plans**: 6/6 plans complete
**UI hint**: yes

**Wave 1** *(parallel — disjoint Go vs frontend subsystems)*

- [x] 155-01-PLAN.md — ChatStore.Export() YAML frontmatter + Go export-route tests + TESTING.md (EXPORT-01)
- [x] 155-02-PLAN.md — Frontend primitives: RelayClient/TerminalPanel/ChatPanel `wsURL`/`apiBaseURL`/`capToken` + Export button + RO-cap suppression (PARITY-01, EXPORT-01)

**Wave 2** *(blocked on 155-02)*

- [x] 155-03-PLAN.md — WebShareSessionView wrapper + App.tsx web-mode session tab + vitest (PARITY-01)

**Wave 3** *(blocked on 155-01 + 155-03)*

- [x] 155-04-PLAN.md — Playwright fixture chat wiring + chat-parity.spec.ts gate + TESTING.md finalize (PARITY-01, EXPORT-01)

**Gap closure** *(verification gaps_found — PARITY-01 broadcast + SC-3 history-load)*

- [x] 155-05-PLAN.md — BLOCKER 1: diagnose→fix→prove broadcast non-delivery; SC-1 broadcast + unread badge green on 3 browsers (PARITY-01) [wave 1]
- [x] 155-06-PLAN.md — BLOCKER 2: SC-3 WS warm-up fix + full chat-parity suite green on chromium/firefox/webkit (PARITY-01) [wave 2, depends 155-05]

### Phase 156: Install Links & Distribution

**Goal**: The Welcome screen install instructions are accurate and the Linux curl + Windows winget distribution paths work end-to-end.
**Depends on**: Nothing (orthogonal to chat phases; can run in parallel or after)
**Requirements**: INSTALL-01, INSTALL-02, INSTALL-03
**Success Criteria** (what must be TRUE):

  1. Following the Linux curl command on the Welcome screen executes `scripts/install.sh`, detects the correct architecture, verifies the SHA256 checksum from `checksums.txt`, and installs the `agenthub` binary to a standard path — verified on a clean Linux box and documented in the TESTING.md manual checklist.
  2. The Welcome screen shows `winget install scottkw.agenthub` (correct package ID) and links to `github.com/scottkw/agenthub` — no placeholder URLs or wrong IDs visible anywhere on the screen.
  3. The `distribute.yml` workflow executes the first-submission code path correctly when `WINGET_FIRST_SUBMISSION=true` and `WINGET_TOKEN` are provisioned; the repo-side automation is complete and verifiable by a dry-run — the phase ships even while Microsoft's catalog PR review is externally pending.

**Plans**: 3/3 plans complete
**Wave 1**

- [x] 156-01-PLAN.md — Fix the three Welcome screen install strings + vitest source-gate (INSTALL-01/02)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 156-02-PLAN.md — POSIX scripts/install.sh (arch detect → SHA256-verify → install) + shellcheck CI gate (INSTALL-01)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 156-03-PLAN.md — winget first-submission dry-run + operator runbook + M-26 (INSTALL-03)

**UI hint**: yes

### Phase 157: Terminal Screen-Share Semantics (Issue #109)

**Goal**: Eliminate the cross-viewer PTY grid-size garble (overlapping/doubled characters when viewers have different window sizes) by adopting "screen-share semantics" (Option B, decision locked in Issue #109): the host's terminal is the single source of truth for the PTY grid; guests render at the host's grid size and CSS-scale to fit their viewport, never driving the PTY. Full Option B scope (all six change-layers).
**Depends on**: Nothing hard — orthogonal to the chat phases (Phase 154's chat drawer is overlay mode and no longer resizes the PTY). Touches relay/webserver PTY arbitration + the web and desktop terminal viewers.
**Requirements**: VIEW-01, VIEW-02, VIEW-03, VIEW-04, VIEW-05
**Success Criteria** (what must be TRUE):

  1. With a host (local origin) and a smaller-windowed guest connected, the guest renders the host's exact grid with no garble (no overlapping/doubled characters) — verified against the Issue #109 screenshot scenario.
  2. A guest's window resize never changes the PTY size; only the host's resize does (host-authority) — verified by asserting the arbiter ignores web-origin `MsgResize` (replaces the MC-06 max-wins behavior).
  3. A freshly-joined guest receives the host's grid size (`MsgResize` pushed before scrollback replay), so static (non-repainting) screens render correctly on join.
  4. Guests honor server-pushed `MsgResize` (0x02) → `term.resize(cols,rows)` and CSS-scale the host grid to fit their viewport, capped at scale ≤ 1.0 (downscale-only, never upscale); the host view is never disturbed.
  5. The desktop `TerminalPanel` achieves cross-surface parity for the guest path, or the gap is explicitly documented if hosts are always desktop / guests always web (cross-surface parity is a release-blocking project rule).
  6. The existing MC-06 max-wins hub tests are replaced with host-authority tests; the TESTING.md suite manifest + traceability map are updated for every new/changed test file.

**Open decisions** (resolve at discuss/plan time — from the Issue #109 thread): host-not-connected behavior (freeze last host size vs. a fixed default like 120×32); multiple local subscribers (max-among-local vs. latest-wins); read-write guest model (types into the host-sized grid via `MsgInput`/inject but cannot resize it).
**Plans**: 5/5 plans complete

- [x] 157-01-PLAN.md — Hub arbiter rewrite (host-authority + min-among-local + freeze + web-origin gate + broadcastResize + Rows()) + hub-test replacement (VIEW-01, VIEW-02)
- [x] 157-02-PLAN.md — Server call sites: join-time 0x02 push before scrollback (both surfaces) + drop web-origin resize + server tests (VIEW-02, VIEW-03)
- [x] 157-03-PLAN.md — Web guest viewer: honor 0x02 → term.resize + CSS downscale-to-fit (VIEW-04, VIEW-05)
- [x] 157-04-PLAN.md — Desktop guest parity: RelayClient onResize + isGuest-gated honor/scale + vitest (VIEW-04, VIEW-05)
- [x] 157-05-PLAN.md — TESTING.md regression-convention update (manifest §2 + traceability §4 + manual §5) + path-check (VIEW-01..05)

**UI hint**: yes (terminal viewer rendering)
