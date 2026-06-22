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
- 🚧 **v4.0 Hub-First Consolidation & UI/UX Overhaul** — Phases 136-150 (in progress)

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

### v4.0 Hub-First Consolidation & UI/UX Overhaul (In Progress)

**Milestone Goal:** Collapse AgentHub onto a single Hub-centric surface — retire the TUI and the Sessions/Remote sidebar pages, fold their controls into per-card Share modals and indicators, ship the Claude redesign, fix the tab strip, and stand up a formal regression-test program.

- [x] **Phase 136: TUI Removal** — Delete `agenthub tui`, Bubble Tea views, TUI-only shared code, and TUI tests; cross-surface parity contract narrows to GUI/CLI/web (completed 2026-06-19)
- [x] **Phase 137: Share Modal & Cap Model** — Per-card Share modal with session/file-browse toggles, dual RO/RW cap issuance; security-sensitive backend change isolated for review (completed 2026-06-20)
- [x] **Phase 138: Hub-First Navigation** — Remove Sessions page, Remote page, sidebar New Session item; add local/remote and connected/available indicators on cards; sidebar collapses to Home/Hub/Settings (completed 2026-06-20)
- [x] **Phase 139: Card Rendering & Tab Strip** — Headless VT render for mini-preview/briefing-modal tail (#96); browser-style shrink-then-scroll tab strip (completed 2026-06-21)
- [x] **Phase 140: UI-Spec Gate** — Redesign direction chosen after browser review; #93 backlog triaged (completed 2026-06-21)
- [x] **Phase 141: Redesign Implementation** — gap-closure plans 141-06..09 restyled to the comp (fonts, palette, radii, light/dark toggle); render-compare human-approved 2026-06-21 (false pass closed)
- [x] **Phase 142: Hub & Settings Redesign Polish** — Hub card icon overlap + preview sizing; Settings theme slider; comp-styled New-session buttons; terminal garble on theme/tab switch; Hub groups IA (move out of secondary side panel) (completed 2026-06-21)
- [x] **Phase 143: Regression Test Program** — Automated suite consolidated with CI gate; manual checklist established; standing convention documented (completed 2026-06-22)

## Phase Details

### Phase 136: TUI Removal
**Goal**: The `agenthub tui` command and all Bubble Tea infrastructure is deleted; the codebase is cleaner and all cross-surface parity obligations now apply only to GUI/CLI/web
**Depends on**: Phase 135 (v3.6 complete)
**Requirements**: NAV-01, TEST-06
**Success Criteria** (what must be TRUE):
  1. `agenthub tui` exits with an error or is not recognized — the TUI surface no longer launches
  2. `internal/tui` package, Bubble Tea views, TUI key handlers, TUI file-client wiring, and TUI-specific subcommand dispatch are deleted from the codebase
  3. All TUI tests are deleted (not migrated); the Go and frontend test suites pass with no TUI references
  4. `go build ./...` and the full CI test matrix are green with no TUI import paths remaining
**Plans**: 2 plans
- [x] 136-01-PLAN.md — Delete internal/tui package + cmd_tui.go + dispatch/usage + daemon parity tests; go mod tidy (NAV-01, TEST-06)
- [x] 136-02-PLAN.md — README/comment cleanup + full Go race/frontend suite gate + `agenthub tui` non-zero smoke (NAV-01, TEST-06)

### Phase 137: Share Modal & Cap Model
**Goal**: Each Hub session card has a Share button opening a per-session Share modal; the cap model issues one "share" action minting separate read-only and read/write tokens; file-browse permission inherits from the presented token
**Depends on**: Phase 136
**Requirements**: SHARE-01, SHARE-02, SHARE-03, SHARE-04, SHARE-05, SHARE-06
**Success Criteria** (what must be TRUE):
  1. Clicking Share on a local session card opens a modal with a "Share the session" toggle; toggling it on reveals two distinct share links/codes (one read-only, one read/write), each copyable with a QR code
  2. Toggling "Enable remote file browsing" in the Share modal grants file-browse access that inherits the permission level of the code the visitor presents (RO code → read-only browse, RW code → read/write browse)
  3. When the web server is in local-network mode, the LAN Basic Auth password is visible in the Share modal
  4. Every per-session web-share capability previously available in the Sessions page (on/off, URL, QR, password) is available in the Share modal with no regression in behavior
  5. Share controls are absent (hidden or disabled) on remote peer cards — a user cannot re-share a session they do not own
**Plans**: 3 plans
- [x] 137-01-PLAN.md — Wave 0 test scaffolding: retire global-flag + FilesRead-migration Go tests; add D-03/D-04 browse-matrix + browse-aware route tests + whole-token grep gate; new SessionCard.share + SessionShareModal frontend tests; update SessionSharePanel tests (SHARE-01..06)
- [x] 137-02-PLAN.md — Backend cap-model (security core): engine.go field collapse to sessionBrowse; api.go D-03/D-04 perm injection; POST /sessions/{id}/browse + client + Wails binding; SessionInfo.BrowseEnabled; toggle-off ClearGrants (SHARE-03, SHARE-05)
- [x] 137-03-PLAN.md — Frontend: simplify SessionSharePanel (strip CAP-05); new SessionShareModal (two toggles, LAN password, homeDir warning, server-truth lifecycle); SessionCard Share button + colorblind-safe remote-disabled state; wire into Hub (SHARE-01, SHARE-02, SHARE-04, SHARE-05, SHARE-06)
**UI hint**: yes

### Phase 138: Hub-First Navigation
**Goal**: The sidebar contains exactly Home, Hub, and Settings; Sessions and Remote pages are removed; all session-creation entry points resolve to the Hub's filter-bar button; cards indicate local vs remote origin and, for remote cards, connection state
**Depends on**: Phase 137
**Requirements**: NAV-02, NAV-03, NAV-04, NAV-05, CARD-01, CARD-02, CARD-03, CARD-04
**Success Criteria** (what must be TRUE):
  1. The sidebar renders exactly three items — Home, Hub, Settings — with no Sessions, Remote, or New Session entries
  2. The Hub `.hub__header` title bar and its duplicate New Session button are gone; the HubFilterBar button is the only way to start a new session from the GUI
  3. Every session card clearly indicates whether it is local or remote (via icon, label, or badge — never color alone)
  4. Remote cards additionally indicate whether the user is currently connected to that session vs it being merely available — the distinction is conveyed without relying on color alone
  5. Cards accommodate the new Share button and local/remote/connected indicators without losing attention pulse, mini-preview, or responsive grid reflow
**Plans**: 4 plans
- [x] 138-01-PLAN.md — Wave 0 test scaffolding: rewrite Sidebar/App.nav/App.hub/style.hub tests to the 3-item sidebar + header-less Hub + panel removal; extend SessionCard.share tests with CARD-02/03/04 RED cases (NAV-02..05, CARD-01..04)
- [x] 138-02-PLAN.md — Relocate RemoteSession/RemotePeerSessions types to lib/remoteSession.ts; thread isRemote/isConnected props HubPanel→Grid→Card; provenance-based origin (CARD-02) + colorblind-safe Connected/Available chip + CSS (CARD-03) (CARD-02, CARD-03, CARD-04)
- [x] 138-03-PLAN.md — Card overflow affordances: Kill (two-step confirm), Open-in-browser, Browse-files (CONTEXT D1/D2/D3) guarded against card-click modal; remove .hub__header duplicate New Session button (CARD-01); per-peer unreachable/empty hint (CONTEXT D4) (CARD-01, CARD-04)
- [x] 138-04-PLAN.md — Collapse sidebar to Home/Hub/Settings; clean App.tsx routing (consts/handlers/polls/branches) keeping the Hub remote poll; wire parity handlers into HubPanel; trim TabBar Tab.type; delete both panels + orphan test (NAV-02..05)
**UI hint**: yes

### Phase 139: Card Rendering & Tab Strip
**Goal**: Mini-preview cards and briefing-modal tails render agent output legibly via a headless VT emulator; the tab strip shrinks and scrolls browser-style so all tabs remain reachable regardless of count
**Depends on**: Phase 136
**Requirements**: CARD-05, TAB-01, TAB-02, TAB-03
**Success Criteria** (what must be TRUE):
  1. Mini-preview cards display Claude Code (and other agent) output with correct column spacing and no leaked escape sequences — a headless VT emulator is used for scrollback rendering, not a regex ANSI strip
  2. The briefing-modal tail shows the same legible output under the same rendering path
  3. When more tabs are open than fit the window width, tabs shrink proportionally down to a sensible minimum width (browser-style flex-shrink behavior)
  4. When tabs overflow the available width, visible scroll affordances (chevrons and/or a visible scrollbar) let the user reach every tab
  5. Tab close, rename, and progress-underline affordances remain functional and accessible at minimum tab width
**Plans**: 4 plans
- [x] 139-01-PLAN.md — Wave 0 scaffolding: failing Go styled-tail tests + frontend vtColor/MiniPreview/TabBar tests + A2 headless-xterm verification (CARD-05, TAB-01..03)
- [x] 139-02-PLAN.md — Browser-style tab strip: flex-shrink to icon-only floor + scroll-position-aware chevrons; rename/close/progress survive the floor (TAB-01, TAB-02, TAB-03)
- [x] 139-03-PLAN.md — CARD-05 backend: charmbracelet/x/vt headless VT grid; GetSessionStyledTailLines engine+route+client+Wails binding (n-clamp both layers) (CARD-05)
- [x] 139-04-PLAN.md — CARD-05 frontend: vtColor ITheme mapper + MiniPreview StyledSpan render (xterm-free) + briefing-modal local styled-span/remote headless-xterm tail (CARD-05)
**UI hint**: yes

### Phase 140: UI-Spec Gate
**Goal**: A redesign direction is formally chosen from `./agenthub-v4.0-redesign` after browser review and documented; the #93 backlog is triaged and the in-scope subset identified
**Depends on**: Phase 138
**Requirements**: RDS-01, CARRY-02
**Success Criteria** (what must be TRUE):
  1. A specific redesign direction (or documented explicit mix) is recorded in a UI spec artifact after the standalone HTML comps are reviewed in a browser
  2. The chosen direction is reconciled against Hub-first structure decisions (structural decisions win conflicts); any comp elements that conflict with the already-shipped navigation structure are called out and resolved in the spec
  3. Issue #93 is reviewed; items pulled into v4.0 scope are listed and their delivery assigned to Phase 141; the remainder are re-deferred with #93 updated accordingly
**Plans**: 2 plans
  - [x] 140-01-PLAN.md — Author the UI-spec artifact (Direction 01 Refined Native, accent lock, conflict reconciliation, recolor-only)
  - [x] 140-02-PLAN.md — Post #93 v4.0 triage comment (all 7 #78 items re-deferred; in-scope subset for Phase 141: none); keep #93 open

### Phase 141: Redesign Implementation
**Goal**: The chosen redesign visual language is applied across all surviving surfaces with correct colorblind-safe semantics, prefers-reduced-motion support, and internally consistent Hub GroupSidebar ARIA
**Depends on**: Phase 140
**Requirements**: RDS-02, RDS-03, RDS-04, CARRY-01
**Success Criteria** (what must be TRUE):
  1. All surviving surfaces (Welcome, Hub, terminal/session, File Browser, Editor, Settings) render the chosen redesign visual language
  2. The redesign does not re-introduce the removed Sessions or Remote sidebar pages or a standalone New Session sidebar item — Hub-first structure is preserved throughout
  3. Colorblind-safe semantics and prefers-reduced-motion compliance are maintained across all restyled surfaces in both light and dark themes (verified at hex-constant level, not by eye)
  4. The Hub GroupSidebar ARIA model is internally consistent — either the `listbox`/`option` roles with roving-tabindex are implemented correctly, or both are replaced with a plain focusable control list; no mismatched role/interaction pattern remains (#97)
**Plans**: 9 plans (5 original + 4 gap-closure)
- [x] 141-01-PLAN.md — New --hub-text-dim token (both themes) + Wave 0 RED test updates (GroupSidebar ARIA, StatusBar D-11, SessionShareModal S-07 smoke) (RDS-02, RDS-04, CARRY-01, RDS-03)
- [x] 141-02-PLAN.md — Tokenize sidebar (S-01 Welcome) + tab bar/status bar/terminal (S-03); D-03 fences kept; motion guards (RDS-02, RDS-04)
- [x] 141-03-PLAN.md — Tokenize File Browser + Editor chrome (S-04/S-05) + Settings (S-06); motion guards (RDS-02, RDS-04)
- [x] 141-04-PLAN.md — Add .hub-share-modal* CSS (S-07 gap) + lift inline hex from SessionShareModal.tsx (RDS-02, RDS-04)
- [x] 141-05-PLAN.md — CARRY-01 GroupSidebar ARIA fix + D-11 "Sessions tab" copy reword (CARRY-01, RDS-03)
- [x] 141-06-PLAN.md — [gap] Vendor Plus Jakarta Sans + JetBrains Mono woff2 (no CDN); re-value --hub-* tokens to comp palette + font/radii/type tokens; keep blue accent (RDS-02, RDS-04)
- [x] 141-07-PLAN.md — [gap] Apply font/radii/type tokens across all surviving surfaces; add style.redesign source gate (RDS-02, RDS-03, RDS-04)
- [x] 141-08-PLAN.md — [gap] Wire persisted Light/Dark toggle in Settings → Appearance (sets data-ui-theme on <html>) (RDS-04, CARRY-01)
- [x] 141-09-PLAN.md — [gap] Rendered app-vs-comp comparison per surface + vendor/hex gates + human visual checkpoint (RDS-02, RDS-03, RDS-04, CARRY-01)
**UI hint**: yes

### Phase 142: Hub & Settings Redesign Polish
**Goal**: The post-redesign UAT findings are resolved — Hub cards and Settings/theme controls match the comp's interaction quality, the terminal repaints cleanly across theme/tab switches, and Hub group navigation no longer relies on a secondary side-by-side panel
**Depends on**: Phase 141
**Requirements**: POL-01, POL-02, POL-03, POL-04, POL-05
**Success Criteria** (what must be TRUE):
  1. Hub card header icons (⋮ menu, ☰ handle) no longer overlap other card elements at any card width, and the in-card preview is sized to be legible/useful (POL-01)
  2. The Settings → Appearance Light/Dark control is a single slider/toggle switch (not two separate buttons), keeping persistence + the colorblind-safe state indication (POL-02)
  3. Both Hub "New session" buttons (top-right and empty-state) are styled to match the comp's sidebar "New Session" affordance (POL-03)
  4. A terminal session repaints correctly with no garbling after a theme switch or tab switch (root cause identified; regression source confirmed/fixed) (POL-04)
  5. Hub group navigation is restructured out of the secondary side-by-side panel — groups surface in the main sidebar under Hub (per the comp) or an agreed better pattern; no two collapsible side panels sit side by side (POL-05)
**Plans**: 4 plans
- [x] 142-01-PLAN.md — Wave 0 test scaffolding: POL-02 settings toggle test → role=switch; POL-05 Sidebar group sub-list tests (render/select/drag) + HubPanel GroupSidebar-absent + counts-callback; POL-03/POL-04 source gates (POL-02, POL-03, POL-04, POL-05)
- [x] 142-02-PLAN.md — POL-04 terminal repaint hardening: isActive-guarded theme effect + pendingThemeRef deferral + fitTerminal-after-atlas-clear; native wails dev human verify (autonomous: false) (POL-04)
- [x] 142-03-PLAN.md — POL-05 group nav restructure: extract count helpers, lift groupDefs/activeGroupId to App, remove GroupSidebar side-panel, nested Sidebar group sub-list with CARRY-01 ARIA + drag-to-assign + inline create; delete GroupSidebar (POL-05)
- [x] 142-04-PLAN.md — POL-01 card gutter + 88px preview; POL-02 single role=switch icon+text toggle; POL-03 comp-styled + New Session buttons; all tokenized with light overrides + motion contract (POL-01, POL-02, POL-03)
- [x] 142-05-SUMMARY.md — post-UAT comp-fidelity gap closure (unplanned): New Session accent pill, session-type colored spine+chip (lib/agentBadge.ts), 16px corners, taller/denser preview (tail 4→12, 150px, 10px), card IA restructure (centered name, color-coded origin, Open+Share pair, styled Share). Commits 25e2e7b9, 5930ec2f
**UI hint**: yes

### Phase 143: Regression Test Program
**Goal**: A formal, labeled automated regression suite runs in CI as a merge gate; a single maintained manual checklist replaces scattered per-phase UAT logs; a standing convention is documented for all future phases
**Depends on**: Phase 142
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05
**Success Criteria** (what must be TRUE):
  1. The automated regression suite (Go + vitest + Playwright) is consolidated under a labeled structure with a requirement-to-test traceability map, and runs in CI as a merge gate
  2. Automated coverage gaps for the Hub and for cross-surface GUI/CLI/web flows are closed — the suite tests the v4.0 surface, not just pre-v4.0 code paths
  3. A single maintained manual regression checklist document exists covering all current release-critical behaviors, replacing the scattered per-phase UAT logs
  4. A standing convention is documented requiring every future phase to add its regression tests to the appropriate group (automated vs manual-intervention)
**Plans**: 4 plans
- [x] 143-01-PLAN.md — Close v4.0 coverage gaps GAP-01..04 (hubGroupCounts, agentBadge, sidebar 3-item-with-groups, Phase 142 CSS tokens) [TEST-03]
- [x] 143-02-PLAN.md — Path-existence CI check + build.yml step [TEST-01, TEST-02]
- [x] 143-03-PLAN.md — TESTING.md (manifest + traceability map + gh api command + 11-item manual checklist + convention) + repo CLAUDE.md pointer [TEST-01, TEST-04, TEST-05]
- [x] 143-04-PLAN.md — Apply branch protection on main via gh api (checkpoint-gated) [TEST-02]

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
| 83 | v3.0 | 1/1 | Complete | 2026-04-19 |
| 84 | v3.0 | 3/3 | Complete | 2026-04-19 |
| 85 | v3.0 | 2/2 | Complete | 2026-04-19 |
| 86 | v3.0 | 3/3 | Complete | 2026-04-19 |
| 87-90 | v3.1 | 18/18 | Complete | 2026-05-03 |
| 92-99 | v3.2 | 44/44 | Complete | 2026-05-12 |
| 100-108 | v3.3 | 18/18 | Complete | 2026-05-17 |
| 109 | v3.3.1 | 1/1 | Complete | 2026-05-18 |
| 110 | v3.3.1 | 1/1 | Complete | 2026-05-18 |
| 111 | v3.3.1 | 1/1 | Complete | 2026-05-18 |
| 112 | v3.3.1 | 1/1 | Complete | 2026-05-18 |
| 113 | v3.3.1 | 1/1 | Complete | 2026-05-18 |
| 114 | v3.3.1 | 1/1 | Complete | 2026-05-18 |
| 115 | v3.3.1 | 1/1 | Complete | 2026-05-19 |
| 116 | v3.3.1 | 1/1 | Complete | 2026-05-19 |
| 117 | v3.3.1 | 1/1 | Complete | 2026-05-19 |
| 118-122 | v3.4 | 20/21 | Complete | 2026-05-21 |
| 123 | v3.5 | 4/4 | Complete | 2026-06-14 |
| 124 | v3.5 | 5/5 | Complete | 2026-06-14 |
| 125 | v3.5 | 6/6 | Complete | 2026-06-14 |
| 126 | v3.5 | 4/4 | Complete | 2026-06-15 |
| 127 | v3.5 | 4/4 | Complete | 2026-06-15 |
| 128 | v3.5 | 4/4 | Complete | 2026-06-15 |
| 129 | v3.5.1 | 3/3 | Complete | 2026-06-16 |
| 130 | v3.5.1 | 4/4 | Complete | 2026-06-16 |
| 131 | v3.6 | 5/5 | Complete | 2026-06-16 |
| 132 | v3.6 | 5/5 | Complete | 2026-06-16 |
| 133 | v3.6 | 5/5 | Complete | 2026-06-17 |
| 134 | v3.6 | 8/8 | Complete | 2026-06-18 |
| 135 | v3.6 | 3/3 | Complete | 2026-06-19 |
| 136. TUI Removal | v4.0 | 2/2 | Complete   | 2026-06-19 |
| 137. Share Modal & Cap Model | v4.0 | 3/3 | Complete    | 2026-06-20 |
| 138. Hub-First Navigation | v4.0 | 4/4 | Complete    | 2026-06-20 |
| 139. Card Rendering & Tab Strip | v4.0 | 4/4 | Complete    | 2026-06-21 |
| 140. UI-Spec Gate | v4.0 | 2/2 | Complete    | 2026-06-21 |
| 141. Redesign Implementation | v4.0 | 9/9 | Complete   | 2026-06-21 |
| 142. Hub & Settings Redesign Polish | v4.0 | 4/4 | Complete    | 2026-06-21 |
| 143. Regression Test Program | v4.0 | 4/4 | Complete    | 2026-06-22 |
| 144. Daemon Styled-Tail Race Fix | v4.0 | 1/1 | Complete    | 2026-06-22 |
| 145. Windows Files Test Fixes | v4.0 | 3/3 | Complete    | 2026-06-22 |
| 146. Open Session Capability Bug | v4.0 | —/— | Re-planning | 2026-06-22 |
| 147. In-App Help Page | v4.0 | 0/0 | Not planned | — |
| 148. Session Tab Chevron | v4.0 | 0/0 | Not planned | — |
| 149. Google Antigravity Agent | v4.0 | 0/0 | Not planned | — |
| 150. Shell-Sharing Warning Toggle | v4.0 | 0/0 | Not planned | — |

### Phase 144: Daemon Styled-Tail Race Fix
**Goal**: `GetSessionStyledTailLines` (the headless-VT styled-tail render) passes `go test -race` on all platforms — no data race between the drain goroutine's read and the emulator close
**Depends on**: Phase 143
**Requirements**: FIX-01
**GitHub Issue**: #100
**Success Criteria** (what must be TRUE):
  1. `go test -race ./internal/daemon/` passes — styled-tail tests (TestGetSessionStyledTailLines_*, TestHandleGetSessionStyledTailLines) no longer report a data race
  2. Mini-preview and briefing-modal tails still render correctly (no #96 regression — spacing preserved, private-CSI escapes absorbed)
  3. CI "Run Go tests (all platforms, race detector)" is green for the daemon package across all four build matrix jobs
**Plans**: 1 plan
- [x] 144-01-PLAN.md — Strip query sequences pre-Write + remove drain goroutine (synchronous VT drive), broadened query fixture, TESTING.md FIX-01 traceability

### Phase 145: Windows Files Test Fixes
**Goal**: `internal/files` tests pass on Windows CI — filename sanitization, denylist path-rooting, and atomic-write concurrency behave correctly under Windows path semantics
**Depends on**: Phase 143
**Requirements**: FIX-02
**GitHub Issue**: #101
**Success Criteria** (what must be TRUE):
  1. `TestHandlerUpload_FilenameSanitized`, `TestDenylist_NonHomeRootedUnaffected`, and `TestWriteFileAtomic_ConcurrentReadNeverPartial` pass on `windows/amd64, windows-latest`
  2. The same tests still pass on macOS and Linux (no regression)
  3. The Windows build matrix job is green
**Plans**: 3 plans
- [x] 145-01-PLAN.md — Redirect $HOME in the two home-rooting tests (handler upload sanitize + non-home denylist)
- [x] 145-02-PLAN.md — Build-tagged readFilePlatformSafe (FILE_SHARE_DELETE) for the concurrent-read test
- [x] 145-03-PLAN.md — Update TESTING.md (manifest/traceability/manual) + confirm Windows CI matrix green

### Phase 146: Open Session Capability Bug
**Goal**: The "Open Session" button opens the live session instead of landing on a "capability required" web page
**Depends on**: Phase 143
**Requirements**: FIX-03
**GitHub Issue**: #98
**Success Criteria** (what must be TRUE):
  1. Clicking "Open Session" (Hub card / re-attach) opens the live session, not a "capability required" error page
  2. The capability is issued or reused correctly for the open-in-browser flow
  3. Behaves correctly across GUI + web per cross-surface parity
**Plans**: 4 plans (out-of-band redesign 2026-06-22)
- [ ] 146-01-PLAN.md — Wave 0 TDD RED: inverted RB-03 cap-free meta test + out-of-band open behavior test
- [ ] 146-02-PLAN.md — Go backend: remove broadcast (mint/issuer/fields), restore RB-03, reconcile Go tests
- [ ] 146-03-PLAN.md — Frontend: remove broadcast threading + wire out-of-band open (modal intent, handler, card)
- [ ] 146-04-PLAN.md — TESTING.md regression-convention compliance + full-suite/build-gate verification
> First execution (broadcast "Mechanism B", plans 146-00..03) was superseded after the
> code-review/verify gates found it dead-on-arrival + broadcasting RW to the tailnet. Design
> reversed to out-of-band code/link delivery (see 146-CONTEXT.md D-02/D-04/D-09..D-12,
> 146-RESEARCH.md, superseded-broadcast/). Re-planned 2026-06-22.

### Phase 147: In-App Help Page
**Goal**: An in-app Help page provides documentation, an FAQ, search, and external links, reachable from the app navigation
**Depends on**: Phase 143
**Requirements**: HELP-01
**GitHub Issue**: #69
**Success Criteria** (what must be TRUE):
  1. A Help page is reachable from the app navigation (sidebar/menu)
  2. It includes documentation content, an FAQ, search over that content, and external links (docs/repo/issues)
  3. Present on the appropriate surfaces per cross-surface parity
**Plans**: Not planned yet
- [ ] TBD (run /gsd-plan-phase 147 to break down)

### Phase 148: Session Tab Chevron
**Goal**: Session tabs show a down-chevron indicator signaling the context menu is available, improving discoverability
**Depends on**: Phase 143
**Requirements**: TAB-04
**GitHub Issue**: #68
**Success Criteria** (what must be TRUE):
  1. Each session tab displays a down-chevron affordance indicating a menu
  2. Activating the chevron opens the existing tab context menu
  3. Visual treatment matches the v4.0 redesign and works in light/dark themes
**Plans**: Not planned yet
- [ ] TBD (run /gsd-plan-phase 148 to break down)

### Phase 149: Google Antigravity Agent
**Goal**: Google Antigravity CLI is selectable as a supported agent and launches correctly
**Depends on**: Phase 143
**Requirements**: AGENT-01
**GitHub Issue**: #65
**Success Criteria** (what must be TRUE):
  1. Google Antigravity appears in the agent selection UI
  2. Launching a session with it starts the Antigravity CLI correctly (binary detection / path handling)
  3. Available across surfaces (GUI/CLI/web) per cross-surface parity
**Plans**: Not planned yet
- [ ] TBD (run /gsd-plan-phase 149 to break down)

### Phase 150: Shell-Sharing Warning Toggle
**Goal**: A Settings toggle enables/disables the shell-session web-sharing warning and can re-enable it after the first acknowledgment
**Depends on**: Phase 143
**Requirements**: SET-01
**GitHub Issue**: #51
**Success Criteria** (what must be TRUE):
  1. Settings has a toggle controlling the shell web-share warning
  2. Turning it off suppresses the warning; turning it on restores it (even after a prior one-time acknowledgment)
  3. The setting persists across restarts (backed by daemon settings)
**Plans**: Not planned yet
- [ ] TBD (run /gsd-plan-phase 150 to break down)

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
