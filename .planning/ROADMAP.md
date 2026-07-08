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
- ✅ **v4.1 Session Chat** — Phases 151-164 (shipped 2026-06-29, closes #79, #108, #109)
- **v4.2 Funnel Sharing & Polish** — Phases 165-176 (closes #107, #110, #112, #115, #116, #117, #118, #119, #120, #121, #123, #124, #125, #126, #127, #128, #129; **reopened 2026-07-05**: 165-169 done, +170 public read code / +171 public full-access sharing added after live UAT — Funnel access methods expanded beyond read-only; **expanded again 2026-07-08** after 172/173 UI polish: +174 dependency updates & Dependabot hygiene, +175 web-share/remote-viewer/windowing bug fixes, +176 platform & hardening bug fixes — sweeping the outstanding bug backlog + Dependabot PRs before ship)

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

<details>
<summary>✅ v4.1 Session Chat (Phases 151-164) — SHIPPED 2026-06-29 (closes #79, #108, #109)</summary>

- [x] Phase 151: Message Schema + ChatStore (3/3 plans) — completed 2026-06-25
- [x] Phase 152: Relay Protocol + Identity + Presence (6/6 plans) — completed 2026-06-26
- [x] Phase 153: @session PTY Bridge (3/3 plans) — completed 2026-06-26
- [x] Phase 154: Desktop Chat UI (6/6 plans) — completed 2026-06-26
- [x] Phase 155: Web-Share Chat UI + Cross-Surface Parity Gate (6/6 plans) — completed 2026-06-27
- [x] Phase 156: Install Links & Distribution (3/3 plans) — completed 2026-06-27
- [x] Phase 157: Terminal Screen-Share Semantics (Issue #109) (5/5 plans) — completed 2026-06-27
- [x] Phase 158: Chat Affordance Polish (CHAT-FIX-01, CHAT-PARITY-01) (2/2 plans) — completed 2026-06-27
- [x] Phase 159: Web-Share Chat Parity (WEBCHAT-01..06) (5/5 plans) — completed 2026-06-27
- [x] Phase 160: v4.1 Chat Closeout (NOTIF-01 + tech debt) (5/5 plans) — completed 2026-06-28
- [x] Phase 161: Chat-Sidebar Alias Control (ALIAS-UI-01/02) (4/4 plans) — completed 2026-06-28
- [x] Phase 162: Settings Polish — Terminal Plugins jump link (#108) (1/1 plan) — completed 2026-06-28
- [x] Phase 163: Read-Only Guest Chat Posting (ROCHAT-01/02, SEC-RO-01) (3/3 plans) — completed 2026-06-29
- [x] Phase 164: Web-Share Chat Layout Polish (CHAT-LAYOUT-01/02) (2/2 plans) — completed 2026-06-28

</details>

<!-- v4.1 phase details archived to milestones/v4.1-ROADMAP.md -->

**v4.2 Funnel Sharing & Polish (Phases 165-168)**

- [x] **Phase 165: Funnel Backend** - Atomic Funnel lifecycle: LocalClient promotion, EnableFunnel/DisableFunnel, Funnel-aware Origin allowlist + BaseURL + share-URL builders, four-path teardown, CheckFunnelAccess preflight, auto-expiry enforcement (completed 2026-06-30)
- [x] **Phase 166: Funnel Frontend + Help Guide** (5/5 plans) — completed 2026-06-30, verified 8/8 - Risk-acknowledgment dialog, auto-expiry selector, colorblind-safe internet-exposure indicator, Funnel URL display, one-click disable, in-app Sharing Guide Help article
- [x] **Phase 167: Native Notifications** - beeep cross-platform notification on waiting-state transition, de-dup guard, Settings toggle (default off) (7/7 plans executed 2026-07-01, incl. 167-05/06/07 M-41 crash-hardening + instrumentation + permission-denied hint) — CLOSED complete 2026-07-01; code-verified 11/11, M-41 live on-screen delivery DEFERRED to release-time UAT on signed macOS/Windows/Linux builds (inherently unautomatable)
- [x] **Phase 168: Bug Fix & Settings Polish** - Fix #112 (web-guest plugin-config SSE), #117 (multi-viewer kick + disconnect UI), #118 (remote-open in-app tab), #115 (Footer Share modal), #116 (Hub auto-switch setting), #121 (phantom viewer count — pairs with #117) (9/9 plans executed 2026-07-02, incl. 168-08 UX-02 footer-pill drift + 168-09 FIX-03 remote-tab RC-A/B/C gap-closures) — CLOSED complete 2026-07-02; verified 6/6 must-haves, all live UAT 4/4 PASS (FIX-01/02/03 + UX-02 confirmed on a two-Mac tailnet production build)
- [x] **Phase 169: Tailscale Detection Fix** - Fix #120 (Tailscale reports "installed but not Connected" on non-admin macOS accounts where `macsys` `sameuserproof` is unreadable) — honest permission-aware detection (`permProbeFunc` + `PermissionLimited`, never a false Connected) + Settings "Permission Limited" state. Executed + verified 2026-07-05 (2/2 plans); M-45 live non-admin macsys acceptance deferred (env-only).
- [x] **Phase 170: Public Share Access Codes (read)** - Reusable, share-lifetime join code for the INTERNET (PUBLIC) link so recipients who can't scan the QR / paste the full URL can join read-only with a short code (FNL-08). Added 2026-07-05 from live UAT. Planned 2026-07-05 (4 plans / 4 waves). **COMPLETE 2026-07-06** — executed 4/4 + code-verified 14/14 + M-46 live off-tailnet UAT PASSED. Live UAT surfaced + fixed a blocker (commit 5a92ddae): the public URL/Copy/Open/QR pointed at the ephemeral `/sessions/{id}?cap=` cap link (401'd once the grant rotated/daemon restarted) instead of the reusable `/join?code=<publicReadCode>` entry point.
- [x] **Phase 171: Public Full-Access (Read-Write) Sharing** - Opt-in public read-write Funnel sharing behind a hard consent gate + single-use write code + shorter expiry; supersedes today's accidental write-cap rebasing (FNL-09). Added 2026-07-05. **SPEC-FIRST**: `/gsd-spec-phase 171` → discuss → secure → plan. Executed 4/4 + automated verification PASSED 2026-07-07 (build/full go-race suite/2353 frontend tests green; closed-write-perimeter invariant confirmed at source — deny-before-gate 403, expiry clamp 3600, Perms hardcoded read,write no files.write; colorblind-safe indicators verified at source). **PENDING live M-47 off-tailnet public-write UAT** — the RCE-severity acceptance gate — via `/gsd-verify-work 171`. (completed 2026-07-08)

## Phase Details

### Phase 165: Funnel Backend

**Goal**: The daemon can activate and fully tear down Tailscale Funnel with correct Origin/BaseURL awareness, so an internet-external guest can reach a Funnel-enabled session.
**Depends on**: Nothing (first v4.2 phase)
**Requirements**: FNL-01, FNL-02, FNL-03, FNL-04, FNL-05, FNL-06, FNL-07
**Success Criteria** (what must be TRUE):

  1. A Funnel share URL emitted by the daemon has no port suffix (`https://hostname.ts.net/sessions/id?cap=TOKEN`, not `:7443`)
  2. An HTTP request from a machine outside the tailnet carrying `Origin: https://hostname.ts.net` receives 200, not 403
  3. `tailscale serve status` shows an empty config after each of the four teardown triggers: user disables Funnel toggle, user disables web-share, session ends naturally, daemon stops cleanly
  4. When Funnel prerequisites are not met, `EnableFunnel` returns a human-readable error string matching `ipn.CheckFunnelAccess` output and never calls `SetServeConfig`
  5. Web-share in local-network fallback mode (Tailscale not running) continues unaffected and `funnelActive` remains false

**Plans**: 5/5 plans complete

- [x] 165-05-PLAN.md

- [x] 165-01-PLAN.md — webserver Funnel primitives: LocalClient promotion, funnelClient seam, EnableFunnel/DisableFunnel/FunnelBaseURL/ClearLingeringFunnel, dual-origin allowlist (FNL-02/04/06 + Stop teardown)
- [x] 165-02-PLAN.md — daemon half: funnelSessions/funnelExpiry maps, handleSetSessionFunnel endpoint, five-trigger teardown helper, Funnel-aware URL builders, auto-expiry, startup clear, SessionInfo.funnelActive (FNL-01/03/05/07)
- [x] 165-03-PLAN.md — Wails on-ramp: App.SetSessionFunnel + DaemonClient.SetSessionFunnel + SessionInfo.FunnelActive mirror (FNL-01)
- [x] 165-04-PLAN.md — gap closure (live UAT): fix Funnel 502 (proxy target localhost→bindIP, https+insecure) + tear down Funnel on explicit kill path; real-reachability + kill-path regression tests (FNL-03/FNL-05)

### Phase 166: Funnel Frontend + Help Guide

**Goal**: Users can enable internet-sharing through a risk-aware UI flow, see a persistent non-color exposure indicator, and access an in-app guide covering both sharing paths.
**Depends on**: Phase 165
**Requirements**: FUI-01, FUI-02, FUI-03, FUI-04, FUI-05, FUI-06, HLP-01, HLP-02
**Success Criteria** (what must be TRUE):

  1. Clicking the Funnel toggle shows a risk-acknowledgment dialog on every enable (no "don't show again") — containing risk statement, auto-expiry duration selector, and a cross-link to the Help guide
  2. While a session is internet-exposed, the Hub card and session tab display a persistent text-label indicator (e.g. "INTERNET ACTIVE" + globe icon) that conveys state without relying on color alone; verified at hex/source level
  3. The Share modal shows the Funnel URL with copy-to-clipboard and QR code; a "starting up..." UX state is shown immediately after enable while the TLS connection warms up
  4. One-click Funnel disable from the Share modal removes the Funnel exposure and clears the indicator immediately
  5. The Help tab includes a "Sharing Guide" article covering both the Funnel path and the device-share + ACL alternative, with a copy-pasteable ACL grant block and the wildcard-default (`*→*`) gotcha called out explicitly

**Plans**: 5 plans

Plans:
**Wave 1**

- [x] 166-01-PLAN.md — Wave 0 blocker: hand-authored Wails stubs (SetSessionFunnel + SessionInfo.funnelActive) + import-contract test + all Phase-166 CSS tokens/classes
- [x] 166-02-PLAN.md — Funnel enable flow: inline risk panel (every enable) + auto-expiry selector + Help cross-link + local-fallback disable + HubPanel session sync
- [x] 166-03-PLAN.md — Colorblind-safe internet-exposure indicator: Hub card globe+"INTERNET" badge + session-tab globe icon + App.tsx funnelActiveSessions
- [x] 166-04-PLAN.md — Sharing Guide help article (both paths + ACL grant + wildcard gotcha) registered in both Help nav arrays

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 166-05-PLAN.md — Funnel URL display + warm-up UX + one-click disable + TESTING.md manifest/traceability/M-37..M-40 + tsc && vite build gate

**UI hint**: yes

### Phase 167: Native Notifications

**Goal**: Users who opt in receive a single native OS notification the moment a session transitions to awaiting-input state, on macOS, Windows, and Linux.
**Depends on**: Nothing (independent of Funnel)
**Requirements**: NTF-01, NTF-02, NTF-03, NTF-04
**Success Criteria** (what must be TRUE):

  1. When a session transitions from any non-waiting state to `waiting`, a native OS notification fires on macOS, Windows, and Linux — including when the app window is hidden in the system tray
  2. A session held in `waiting` state for 5 minutes triggers no additional notifications (fires exactly once per `running → waiting` transition)
  3. The notification text includes the session name and agent type so the user knows which session needs attention
  4. A Settings → Session Behavior toggle enables/disables notifications; the toggle defaults to off and suppresses all notifications when off

  *(Note: toggle placement moved to the **Behavior** section per the LOCKED user correction in 167-CONTEXT.md — success-criterion-4 "Session Behavior" wording is superseded; intent unchanged.)*

**Plans**: 7/7 plans complete

Plans:
**Wave 1**

- [x] 167-01-PLAN.md — Daemon settings persistence: `NotifyOnWaiting` through engine → api → client (mirrors StartMinimized) (NTF-04)
- [x] 167-02-PLAN.md — Cross-platform notification primitives: beeep v0.11.2 Windows/Linux wrappers + per-call identifier threaded through the native macOS path; delete notification_other.go (NTF-01)

**Wave 2** *(blocked on Wave 1)*

- [x] 167-03-PLAN.md — App-layer trigger: `maybeNotifyWaiting` edge-detect + cold-start baseline in the tray poller, `displayNameForCLI`, Get/SetNotifyOnWaiting bound + wailsjs bindings (NTF-01/02/03/04)

**Wave 3** *(blocked on Wave 2)*

- [x] 167-04-PLAN.md — Settings Behavior-section toggle (default off) + search entry + TESTING.md Category U / M-41 / traceability (NTF-04)

**Gap closure** *(M-41 regression — unguarded native notification crash)*

- [x] 167-05-PLAN.md — Harden the darwin native notification path: bundle-id guard + @try/@catch + Go log-and-swallow so the always-on tray poller can no longer abort the GUI process under wails dev; headless bundle-id regression test + TESTING.md registration (NTF-01/02/03/04)

**Gap closure** *(M-41 live-delivery blocker — native toast never displays on a signed build)*

- [x] 167-06-PLAN.md — Instrument every native notification path (attempt/not-granted/auth-error/delivery logging), request UNUserNotificationCenter authorization proactively on toggle-enable, register a willPresentNotification foreground-presentation delegate, emit a `notification:permission-denied` event on denial, add maybeNotifyWaiting edge-fire + cache-load logging; headless Go tests for the authorization seam + TESTING.md registration (NTF-01/04)
- [x] 167-07-PLAN.md — Frontend permission-denied hint in the Settings Behavior section wired to the `notification:permission-denied` event (System Settings → Notifications remediation) + vitest coverage + TESTING.md registration (NTF-04) *(depends on 167-06)*

**UI hint**: yes

### Phase 168: Bug Fix & Settings Polish

**Goal**: Five web-share/Hub bugs are repaired and two Settings/Footer UX friction points eliminated, clearing Issues #112, #115, #116, #117, #118, and #121.
**Depends on**: Phase 165 (Phase 165's dual-origin fix may resolve the Funnel multi-viewer kick component of #117 — verify in Phase 165 UAT before scoping #117 relay buffer + disconnect work here)
**Requirements**: FIX-01, FIX-02, FIX-03, FIX-04, UX-01, UX-02
**Success Criteria** (what must be TRUE):

  1. A web-share guest opening a session URL in a real browser (not the Wails WebView) sees live plugin-config changes without page reload, with no CSP errors visible in DevTools Console
  2. Multiple simultaneous viewers can connect to one shared session; a stuck viewer can be disconnected via the Share modal Disconnect button, and the Hub viewer count updates within the next poll cycle
  3. Opening a remote tailnet session from the Hub opens an in-app terminal tab (not an external browser window) that streams the terminal relay correctly
  4. The footer "Share Session" button opens the Share modal for the currently-active session with no independent state drift
  5. A "Stay on Hub after creating session" toggle in Settings → Session Behavior prevents auto-switching to a newly Hub-created session when enabled
  6. A never-shared local session's Hub card reads 0 viewers — the count excludes the app's own internal WebSocket subscribers (Terminal/Chat/status watcher) and reflects only real remote/shared viewers (#121; touches the same `ViewerCount`/relay-hub plumbing as #117, so fix alongside FIX-02)

**Plans**: 7/7 plans complete
Plans:
**Wave 1**

- [x] 168-01-PLAN.md — FIX-04: Hub.RemoteViewerCount() (web-origin only) + engine viewerCount swap [wave 1]
- [x] 168-02-PLAN.md — FIX-01: web-guest plugin-config self-fetch + SSE + baseURL prop seam [wave 1]

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 168-03-PLAN.md — FIX-03: reroute remote-session open to in-app web-session tab (per-tab cap/host) [wave 2]

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 168-04-PLAN.md — UX-01: "Stay on Hub after creating a session" setting + toggle + createTab gate [wave 3]

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 168-05-PLAN.md — UX-02: footer "Share Session" button opens lifted Share modal, gated on tab type [wave 4]
- [x] 168-06-PLAN.md — FIX-02: Hub.DisconnectWebViewers() + owner-only RPC + Share-modal Disconnect button [wave 4]

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 168-07-PLAN.md — Regression-suite wiring: TESTING.md manifest/traceability + M-13 reword + manual M-NN [wave 5]

**Gap closure** *(UAT/verification failures)*

- [x] 168-08-PLAN.md — UX-02/#115 gap: footer Share pill web-share drift (modal toggle notifies App.webEnabled) [gap closure]
- [x] 168-09-PLAN.md — FIX-03 gap: remote web-session tab — daemon-proxy transport (RC-A) + full-height wrapper (RC-B) + "Open in tab" relabel (RC-C) [gap closure]

**UI hint**: yes

### Phase 169: Tailscale Detection Fix

**Goal**: On a non-admin macOS account, Tailscale connection state is reported correctly ("Connected" rather than "installed but not connected"), clearing Issue #120.
**Depends on**: None (orthogonal to the Funnel/web-share/Hub work — different subsystem, macOS-specific).
**Requirements**: FIX-05
**Success Criteria** (what must be TRUE):

  1. On a non-admin macOS account where the `macsys` `sameuserproof` file is unreadable (root:admin 0640), Tailscale detection reports an accurate state — "daemon running, status unreadable (permission-limited on non-admin macsys)" — instead of a false "installed but not connected"
  2. On accounts where the SDK read succeeds, behavior is unchanged
  3. The UI surfaces actionable guidance (grant admin, or use the Homebrew build) for the permission-limited state

> **Re-plan needed (2026-07-02).** Original CLI-`status`-fallback approach invalidated by Phase 169 code review (CR-01): both the SDK read and a spawned non-setuid `tailscale` CLI run as the same OS user and hit the identical `0640 root:admin` `sameuserproof` gate that `tailscaled` sets by design; only the macsys GUI bypasses it via in-process `SetCredentials`. Re-scoped to honest permission-aware detection (EACCES-vs-daemon-down). See 169-REVIEW.md.

**Plans**: 2 plans (re-planned 2026-07-02 — honest permission-aware detection, supersedes the invalidated CLI fallback)

**Wave 1**

- [x] 169-01-PLAN.md — Backend: revert the invalidated CLI-status fallback (D-01) + add `permProbeFunc` file-probe seam, `PermissionLimited` field, and honest darwin/macsys EACCES detection with liveness confirm + unit tests (FIX-05)

**Wave 2** *(blocked on Wave 1 — frontend reads the new field)*

- [x] 169-02-PLAN.md — Frontend guidance + docs: SettingsTab distinct "Permission Limited" state + actionable copy (D-05) + vitest, and TESTING.md reconcile (Suite Manifest, traceability, Category W / M-45 macsys-Standalone rewording per IN-02) (FIX-05)

**UI hint**: no

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 165. Funnel Backend | 5/5 | Complete   | 2026-06-30 |
| 166. Funnel Frontend + Help Guide | 5/5 | Complete (verified 8/8) | 2026-06-30 |
| 167. Native Notifications | 7/7 | Complete   | 2026-07-01 |
| 168. Bug Fix & Settings Polish | 9/9 | Complete   | 2026-07-02 |
| 169. Tailscale Detection Fix | 2/2 | Complete (verified; M-45 live deferred) | 2026-07-05 |
| 170. Public Share Access Codes (read) | 4/4 | ✅ M-46 live UAT PASSED (2026-07-06) | — |
| 171. Public Full-Access (RW) Sharing | 4/4 | Complete    | 2026-07-08 |
| 172. Hub-card layout & badge refinement | 1/1 | Complete | 2026-07-08 |
| 173. Share modal three-tab segmented redesign | 8/8 | Complete (UAT verified 3/3 — SM-02/SM-07 via live dev-browser harness) | 2026-07-08 |

### Phase 170: Public Share Access Codes (read)

**Goal:** Funnel/internet shares surface a reusable, share-lifetime join code in the Share modal's INTERNET (PUBLIC) section, so a recipient who cannot scan the QR or paste the full capability URL can join **read-only** with a short code — closing the UAT dead-end where typing the public URL lands on a code-entry page with no code available.

**Requirements**: FNL-08

**Depends on:** Phase 166 (Funnel frontend), Phase 165 (Funnel backend)

**Design constraints (locked in discussion 2026-07-05):**

- Add **per-code TTL + reusable (non-single-use)** semantics to `JoinCodeManager` (today all codes are 5-min single-use — wrong for a public share meant to last the auto-expiry window and serve multiple viewers).
- The public code's lifetime is **tied to the funnel auto-expiry** (dies exactly when the share does).
- **Read-only scope only** — the reusable code resolves to the funnel *read* cap; it must never map to the write cap.
- Keep **40-bit crypto/rand** entropy (2⁴⁰ over an ≤8h window is not brute-forceable even without rate-limiting).
- **Supplement, not replace** the existing self-contained cap-URL + QR flow.

**Plans:** 4/4 plans complete

Plans:

**Wave 1**

- [x] 170-01-PLAN.md — Reusable join-code primitive: `IssueReusable`/`Revoke`/`reusable` flag on `JoinCodeManager` + conditional-delete `Exchange` + new `internal/webserver/join_test.go` public double-exchange proof (FNL-08)

**Wave 2** *(blocked on Wave 1)*

- [x] 170-02-PLAN.md — Daemon Funnel public read-code: mint-once-cached read-only code in `issueCapabilitiesForSession`, 8h-capped TTL from `handleSetSessionFunnel`, `Revoke` in `disableFunnelForSession`, `PublicReadCode` on `IssueCapabilitiesResponse` + funnel_test.go scope/idempotent/teardown tests (FNL-08)

**Wave 3** *(blocked on Wave 2)*

- [x] 170-03-PLAN.md — Frontend: models.ts sync + reusable `<CodeDisplay>` row in the INTERNET (PUBLIC) section + `publicReadCode` threaded through the modal warm-up/disable + component test (FNL-08)

**Wave 4** *(blocked on Waves 1-3)*

- [x] 170-04-PLAN.md — TESTING.md reconcile: Suite Manifest counts + FNL-08 traceability rows + M-46 live off-tailnet manual item + `check-traceability-paths.sh` (FNL-08)

### Phase 171: Public Full-Access (Read-Write) Sharing

**Goal:** Add opt-in **public read-write** Funnel sharing behind a distinct hard consent gate, surfacing a full-access public URL + write code + QR with unmistakable, colorblind-safe consent — and **supersede today's *accidental* public write**: `issueCapabilitiesForSession` already rebases BOTH read and write caps to the public Funnel base by cap-issue timing, and the Funnel serve config exposes the whole mux with no read-only downgrade, so public write is currently reachable but unintentional/unlabeled. This phase replaces that with a deliberate, gated flow (net security improvement).

**Requirements**: FNL-09

**Depends on:** Phase 170, Phase 166, Phase 165

**⚠ SPEC-FIRST — do NOT go straight to /gsd-plan-phase.** Route: `/gsd-spec-phase 171` (clarify WHAT + the "internet RCE, behind what gates" question) → `/gsd-discuss-phase 171` → `/gsd-secure-phase` (threat model) → plan. Public read-write = internet-reachable command execution on the host; the consequence of a leaked link is RCE, not just observation.

**Design starting point (to be confirmed in spec/discuss):**

- A **distinct second gate** beyond "enable internet sharing" — a typed acknowledgment ("I understand this grants command execution to anyone with the link"), not a one-click toggle.
- **Single-use write codes only** (never reusable — a leaked reusable write code is far worse than a leaked read one). Note this diverges from Phase 170's reusable *read* code.
- **Shorter default/max expiry** for write shares.
- Distinct, unmissable UI treatment (text + icon + color, colorblind-safe) so a public-write link is never confused with public-read.
- Threat model must assert: **no public write path except through the new gate** (closes the accidental rebasing).

**Plans:** 4/4 plans complete

Plans:

**Wave 1**

- [x] 171-01-PLAN.md — Capability + webserver enforcement primitives: `IssueSingleUseWithTTL`, `RemoveGrant`, `SetRWGate`/`isRWGated` + `rwGated` map, gate-aware `originAllowedForWrite`, and the critical `TestHandleWSSRelay_WriteCap_RequiresGate` (FNL-09)

**Wave 2** *(blocked on Wave 1)*

- [x] 171-02-PLAN.md — Daemon RW-gate lifecycle: `handleSetSessionFunnelWrite` (terminal-only D-05, expiry clamp R5), `revokeFunnelWriteLocked`/`disableFunnelWriteForSession` all-trigger teardown, D-04 write-rebase removal, `SessionInfo.FunnelWriteActive`, client + Wails binding (FNL-09)

**Wave 3** *(blocked on Wave 2)*

- [x] 171-03-PLAN.md — Frontend: `.hub-funnel-write-gate` Danger section (≥3s hold-to-confirm, keyboard parity, warm-up gating, post-gate/used states) + colorblind-safe `.hub-fullaccess-badge`/tab icon (label+icon+shape distinct) + modal wiring (FNL-09)

**Wave 4** *(blocked on Waves 1-3)*

- [x] 171-04-PLAN.md — TESTING.md reconcile (Suite Manifest + FNL-09 traceability + M-47 live off-tailnet public-write UAT) + Sharing Guide FULL ACCESS section (FNL-09)

### Phase 172: Hub-card layout & badge refinement

**Goal:** Consolidate the Hub session card's three inconsistent metadata treatments (`Running`/`Local` icon+text, `/bin/zsh` outlined pill, `INTERNET` filled green pill on its own row) into ONE consistent chip row (agent · origin · exposure) with tighter vertical rhythm — while deliberately KEEPING the INTERNET chip the one prominent colored/filled chip (security-exposure signal that must stay unmissable + colorblind-safe per user_colorblind). Making the other chips quieter/outlined makes INTERNET pop MORE by contrast. Frontend-only (Hub card component + style.css); no backend.
**Requirements**: TBD (design-polish phase; user flagged 2026-07-05, critique captured in commit 4402b44e)
**Depends on:** None (frontend-only, independent of Funnel phases 170/171)
**Plans:** 1/1 plans complete

**Approach note:** User wanted 2-3 throwaway HTML mockups (frontend-design skill / /gsd-sketch) to compare chip-row treatments BEFORE touching code. Done — Sketch 001 (`.planning/sketches/001-hub-card-chip-row/`), WINNER = Variant B (7px rounded-rect quiet chips, dedicated exposure line, fully-muted origin). Plan builds to Variant B.

Plans:

- [x] 172-01-PLAN.md — Consolidate the Hub card into one chip row (agent · origin quiet chips + dedicated INTERNET/FULL ACCESS exposure line + muted uptime·viewers·Connected meta line); style.css both themes + SessionCard.tsx restructure + vitest structure coverage + TESTING.md reconcile (D-01..D-07)

### Phase 173: Share modal three-tab segmented redesign

**Goal:** Reorganize the session Share modal (GitHub #129) from a single growing column — where each toggle *injects* a block that pushes everything down until the dialog overflows and scrolls — into a **stable frame**: a fixed toggle control strip + a three-tab **segmented access panel** (Tailnet · Private / Internet · Read-only / Internet · ⚠ Full access) whose panels **swap instead of stacking**. Wall off the public-write / command-execution flow entirely inside the Full-access tab, replace the inline-injected Funnel risk panel with a **transient confirm view**, and unify the four ad-hoc link rows into one reusable `ShareLinkCard`. Frontend-only UX/IA + polish — **no change** to the sharing capability model, tokens issued, TTL semantics, Funnel teardown, or the 3s hold-to-confirm safety gate. Colorblind-safe (⚠ glyph + inset ring, not hue; On/Off/N-A text labels per user_colorblind) and keyboard-accessible (`role=tablist`/`tab`, roving tabindex, `:focus-visible`).
**Requirements**: SM-01 (fixed control strip — toggles pinned, toggling never reflows/pushes on-screen content), SM-02 (bounded height — no state scrolls the whole dialog; any scroll confined to one region within `max-height: calc(100vh - 64px)`), SM-03 (three-tab segmented control Tailnet·Private / Internet·Read-only / Internet·⚠ Full-access that swaps the panel body), SM-04 (public-write / command-execution flow — warning → enable → hold-to-confirm → armed summary → disable — lives ONLY in the Full-access tab), SM-05 (Internet tabs disabled/`aria-disabled` until internet risk confirmed; risk ack is a transient confirm view not injected above links; default tab after confirm = Internet·Read-only; disabling internet resets to Tailnet), SM-06 (one reusable `ShareLinkCard`: title · truncated URL · Copy/Open/QR · join code · scope description attached directly beneath — used by all tailnet + internet rows), SM-07 (colorblind-safe + keyboard-operable: ⚠ glyph + inset ring not hue, On/Off/N-A text labels, real `tablist`/`tab` with roving tabindex + visible focus, `prefers-reduced-motion` fallback on hold-to-confirm), SM-08 (modal widened to `width: min(520px, calc(100vw - 48px))`, still clamps on narrow viewports; capabilities/tokens/TTL/Funnel-teardown/3s hold-gate unchanged; `SessionShareModal.test.tsx` + `SessionSharePanel.test.tsx` updated with attribute-based, non-hue assertions)
**Depends on:** None (frontend-only UX/IA; reorganizes the already-shipped share surfaces from Funnel phases 165/170/171 — no unfinished-phase dependency)
**Plans:** 8/8 plans complete
**Status:** ✅ Complete — UAT verified 3/3 (2026-07-08). All 3 human-verification items (SM-02 scroll confinement, SM-07 colorblind-safe inset ring, SM-07 prefers-reduced-motion hold fallback) passed via live dev-browser component harness; VERIFICATION.md status = passed.

**Approach note:** GitHub #129 is a full design spec — the chosen three-tab segmented layout was selected from four reviewed candidates and an interactive mockup was built + reviewed before the spec was written. Components: `ShareSegmentedControl` (new), `ShareLinkCard` (new/extracted), `SessionShareModal.tsx` shell (owns tab/confirm state), `SessionSharePanel.tsx` refactored into `TailnetTab` / `InternetReadOnlyTab` / `InternetFullAccessTab`; preserve `HoldToConfirmButton` + `CodeDisplay` + `FunnelRiskPanel` (latter becomes the transient confirm view). CSS extends `frontend/src/style.css` — migrate inline `style={}` layout to classes. RESEARCH corrected stale spec refs: reuse existing `riskPanelOpen`/`funnelOn` state (not new `internetEnabled`/`internetConfirmed`); tokens are `--hub-*` (both theme blocks), not the spec's literals; `.hub-share-modal` at style.css ~`:6391`. 5 waves: (1) CSS foundation + shared-component hoist, (2) ShareSegmentedControl + ShareLinkCard, (3) three tab bodies, (4) shell wiring + modal tests, (5) TESTING.md reconcile + gate.

Plans:
**Wave 1**

- [x] 173-01-PLAN.md — CSS foundation: width bump min(520px), single inner scroll region, `.share-seg*`/`.share-linkcard*` classes + new `--hub-*` token in both themes (SM-02/03/06/07/08)
- [x] 173-02-PLAN.md — Hoist CodeDisplay + HoldToConfirmButton to shared module + add prefers-reduced-motion plain-confirm fallback (SM-07/08)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 173-03-PLAN.md — New `ShareSegmentedControl` (role=tablist, roving tabindex, arrow-nav, ⚠ danger glyph) + a11y tests (SM-03/07)
- [x] 173-04-PLAN.md — New reusable `ShareLinkCard` (title·URL·Copy/Open/QR·join·desc, QR→join URL) + tests (SM-06)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 173-05-PLAN.md — Refactor `SessionSharePanel` into TailnetTab/InternetReadOnlyTab/InternetFullAccessTab; wall off public-write in Full-access tab + SM-04 negative test (SM-04/06)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 173-06-PLAN.md — Shell wiring: tab state machine (reuse riskPanelOpen/funnelOn) + transient confirm view + On/Off/N-A labels + delete SessionSharePanel + modal tests (SM-01/05/07/08)

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 173-07-PLAN.md — TESTING.md Suite Manifest + Traceability reconcile + full vitest/build gate (SM-08)

**Gap closure** *(verifier score 6/8 → close two SM-07 defects; independent wave 1)*

- [x] 173-08-PLAN.md — Roving-tabindex focus-follow in ShareSegmentedControl (WR-03) + pending-aware Internet toggle label (WR-02) + funnelOn server-truth resync (WR-01) + 2 regression tests + TESTING.md reconcile (SM-07/SM-05)

### Phase 174: Dependency Updates & Dependabot Hygiene

**Goal:** Bring the dependency tree current without risking the Windows build, the Funnel feature, or CI — merge the low-risk Dependabot bumps behind a full build+test gate, and formally DEFER the three high-risk upgrades via `.github/dependabot.yml` ignore rules + PR-close rationale so they stop re-opening.
**Requirements**: DEP-01 (merge low-risk Dependabot PRs, each verified green: go build/vet/test + `tsc && vite build` + deb/rpm packaging still builds — CI actions #114 attest-build-provenance 4.1.0→4.1.1, #113 setup-go 6.4.0→6.5.0, #103 action-gh-release 3.0.0→3.0.1, #85 pnpm/action-setup 6.0.8→6.0.9; Go modules #89 coder/websocket 1.8.14→1.8.15 [gate on webserver/relay tests], #106 golang.org/x/term 0.43.0→0.44.0, #105 goreleaser/nfpm/v2 2.46.3→2.47.0), DEP-02 (defer high-risk upgrades with documented rationale + `dependabot.yml` ignore entries + close the PRs citing this phase — #104 wailsapp/wails v2 2.10.2→2.12.0 [coupled to the pinned go-webview2 v1.0.19 Windows-build constraint; bump only with a coordinated, tested webview2 upgrade], #88 tailscale.com 1.98.3→1.100.0 [entire Funnel feature built + live-UAT'd on 1.98.3; revisit post-ship with full off-tailnet re-UAT], #102 actions/checkout v6→v7 [major; evaluate in a branch])
**Depends on:** None (independent — dependency/CI hygiene; no code dependency on 173/175/176)
**Plans:** 2/3 plans executed

Plans:
**Wave 1**

- [x] 174-01-PLAN.md — DEP-01 CI-action SHA bumps (#114 attest-provenance, #113 setup-go, #103 gh-release, #85 pnpm) + close PRs

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 174-02-PLAN.md — DEP-01 Go-module bumps (#89 coder/websocket, #106 x/term, #105 nfpm/v2) each gated + close PRs

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 174-03-PLAN.md — DEP-02 defer wails/tailscale/checkout via surgical dependabot.yml ignores + close #104/#88/#102

### Phase 175: Web-share, Remote-viewer & Windowing Bug Fixes

**Goal:** Fix the outstanding web-share / remote-viewer / windowing bugs that degrade the guest and shared-session experience — a mobile guest can read the terminal, a remote viewer learns when the owner ends the session, an exited shared session cleans up its own tab, and host/guest session-open never lands in a dead empty window.
**Requirements**: BUG-01 (#128 — web-share terminal legible on mobile: the 80-col grid no longer downscales to an unreadable size on a phone viewport), BUG-02 (#125 — remote viewer sees a clear disconnect notice when the owner ends/stops the shared session; no silent dead terminal), BUG-03 (#126 — exiting from inside a shared session auto-closes its tab, matching unshared-session behavior), BUG-04 (#119 — host card interaction + guest session-open produce a working session view, no broken/empty MDI windows with no recovery; **re-verify against the Phase 168-03 in-app-tab fix FIRST** and scope only the residual gap)
**Depends on:** None (168 web-share/remote-open plumbing already shipped; independent of 174/176)
**Plans:** 0 plans

Plans:

- [ ] TBD (run /gsd-plan-phase 175 to break down)

### Phase 176: Platform & Hardening Bug Fixes

**Goal:** Close the remaining cross-cutting platform and hardening bugs — the Linux GUI launches and renders, the `/app/` route carries a CSP header, and the Hub card mini-preview wraps long lines correctly.
**Requirements**: BUG-05 (#124 — Linux GUI launches without the macOS-role-menu segfault and the webview renders with no DMABUF freeze; both fixable in `main.go`), BUG-06 (#123 — the `/app/` route serves a Content-Security-Policy header, currently none; carried over from the Phase 168-02 #123 follow-up), BUG-07 (#127 — Hub card mini-preview wraps long lines correctly instead of stacking one character per row / styled-tail preview wrapping)
**Depends on:** None (independent of 174/175)
**Plans:** 0 plans

Plans:

- [ ] TBD (run /gsd-plan-phase 176 to break down)

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
*Full v4.1 details: .planning/milestones/v4.1-ROADMAP.md*
