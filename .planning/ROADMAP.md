# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- ✅ **v1.2 Tailscale-Only Networking** — Phases 14-18 (shipped 2026-03-23)
- ✅ **v1.3 CLI + Daemon** — Phases 19-26 (shipped 2026-03-25)
- ✅ **v1.4 Unified Binary** — Phases 27-29 (shipped 2026-03-25)
- ✅ **v1.5 Bug Fixes & CLI Args** — Phases 30-34 (shipped 2026-03-26)
- ✅ **v1.6 Terminal Fill Fix v2** — Phase 35 (shipped 2026-03-31)
- 🚧 **v1.7 Daemon UX & Branding** — Phases 36-41 (in progress)

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

<details>
<summary>✅ v1.6 Terminal Fill Fix v2 (Phase 35) — SHIPPED 2026-03-31</summary>

- [x] Phase 35: Terminal Fill Fix v2 (1/1 plans) — completed 2026-03-26

</details>

### 🚧 v1.7 Daemon UX & Branding (In Progress)

**Milestone Goal:** Make the daemon a first-class citizen with its own tray icon and management UI, add remote session indicators to web and CLI attach sessions, and establish app branding with proper icons and splash screen.

- [x] **Phase 36: App Icons & Branding Assets** - Generate platform icon sets (ICNS, ICO, PNGs) from the logomark; unblocks all visual work (completed 2026-03-31)
- [x] **Phase 37: Splash Screen** - Branded startup overlay using the title logo; dismisses when daemon connection is confirmed (completed 2026-04-01)
- [ ] **Phase 38: Remote Session Metadata** - Daemon exposes machine hostname in session metadata for remote identification
- [ ] **Phase 39: Remote Session Indicators** - Web terminal status bar and CLI attach banner showing session name, agent, hostname, and connection state
- [ ] **Phase 40: Daemon Management Panel** - React panel inside existing window for session list with status, kill, and web-serve controls
- [ ] **Phase 41: System Tray + Lifecycle** - Persistent tray icon with right-click menu, daemon state indicator, session list, window-hide-on-close, and LSUIElement

## Phase Details

### Phase 36: App Icons & Branding Assets
**Goal**: Properly branded platform icon sets exist for macOS, Windows, and Linux, and the title logo is available in the frontend asset tree for downstream use
**Depends on**: Nothing (first phase of v1.7)
**Requirements**: BRND-01
**Success Criteria** (what must be TRUE):
  1. The built macOS `.app` bundle shows the AgentHub logomark icon in Finder and the Dock (replaces generic placeholder)
  2. `AppIcon.icns` contains all 10 required size/density entries including 1024x1024@2x (Retina-ready)
  3. `icon.ico` contains at least 4 sizes (16, 32, 48, 256px) for Windows taskbar and installer
  4. Multi-size PNGs are present in `build/linux/` for Linux desktop integration and Wails embedding
  5. The full title logo PNG is copied into `frontend/src/assets/` for use by the splash screen
**Plans:** 1/1 plans complete
Plans:
- [x] 36-01-PLAN.md — Generate all icon assets, production build with ICNS injection, visual verification
**UI hint**: yes

### Phase 37: Splash Screen
**Goal**: Users see a branded splash screen during app startup that dismisses automatically when the daemon connection is confirmed, masking WebKit init latency
**Depends on**: Phase 36 (title logo in frontend/src/assets/)
**Requirements**: BRND-02
**Success Criteria** (what must be TRUE):
  1. A splash screen showing the full AgentHub title logo appears immediately on app launch with no visible white-flash before it
  2. The splash screen automatically dismisses once the daemon connection is confirmed and the main UI is ready
  3. If the daemon fails to connect, the splash screen still dismisses within 3 seconds (fallback timeout) so the error banner is visible
  4. The app window is hidden until the splash is ready to display (`StartHidden: true` + `OnDomReady` show pattern — no dock icon flash)
**Plans**: 1 plan
Plans:
- [x] 37-01-PLAN.md — Splash screen implementation (Go lifecycle + React overlay + tests + visual verification)
**UI hint**: yes

### Phase 38: Remote Session Metadata
**Goal**: The daemon includes machine hostname in session metadata so web and CLI clients can identify which host a session is running on
**Depends on**: Nothing (independent of tray and splash work)
**Requirements**: RMTE-03
**Success Criteria** (what must be TRUE):
  1. `GET /api/sessions` response includes a `hostname` field (populated via `os.Hostname()`) for each session
  2. Hostname is available in session metadata without any client configuration — it is populated automatically at daemon startup
  3. Go tests verify the hostname field is present and non-empty in the daemon API response struct
**Plans**: 1 plan
Plans:
- [ ] 37-01-PLAN.md — Splash screen implementation (Go lifecycle + React overlay + tests + visual verification)

### Phase 39: Remote Session Indicators
**Goal**: Remote users (web browser and CLI attach) can see the session name, agent type, host machine name, and connection state without guessing what they are connected to
**Depends on**: Phase 38 (hostname in session metadata)
**Requirements**: RMTE-01, RMTE-02
**Success Criteria** (what must be TRUE):
  1. The web terminal page shows a status bar above the terminal displaying the session name, agent type, and hostname (e.g., "claude-session | claude | macbook-pro.local")
  2. The web terminal status bar updates its connection state indicator within 3 seconds if the session goes offline
  3. The terminal viewport fills correctly after the status bar is added — `proposeDimensions()` row count is unchanged (no regression from v1.6)
  4. Running `agenthub attach <id>` prints a connection banner to stderr before the PTY stream: session name, agent, hostname, and the Ctrl-\ detach key reminder
  5. A "Detached." message is printed to stderr when the user exits an attach session
**Plans**: 1 plan
Plans:
- [ ] 37-01-PLAN.md — Splash screen implementation (Go lifecycle + React overlay + tests + visual verification)
**UI hint**: yes

### Phase 40: Daemon Management Panel
**Goal**: Users can view all active sessions with their status and perform kill/rename/web-serve operations from a panel inside the existing GUI window
**Depends on**: Nothing (can be validated independently of tray work)
**Requirements**: DMGR-03
**Success Criteria** (what must be TRUE):
  1. The daemon management panel is accessible within the existing GUI window (not a separate OS window)
  2. The panel lists all active sessions with their current status (running, waiting, idle, errored)
  3. User can kill any session from the panel without switching to the Sessions tab
  4. User can toggle web serving on/off for any session from the panel
  5. The panel uses only existing Wails bindings — no new Go IPC routes are added
**Plans**: 1 plan
Plans:
- [ ] 37-01-PLAN.md — Splash screen implementation (Go lifecycle + React overlay + tests + visual verification)
**UI hint**: yes

### Phase 41: System Tray + Lifecycle
**Goal**: AgentHub runs as a true tray-resident app — visible in the system tray with a right-click menu, hidden from the dock, and persisting when the window is closed
**Depends on**: Phase 40 (Daemon Manager tray menu item requires the panel to exist), Phase 36 (monochrome tray icon template requires branded icon assets)
**Requirements**: TRAY-01, TRAY-02, TRAY-03, TRAY-04, TRAY-05, TRAY-06, DMGR-01, DMGR-02, BRND-03
**Success Criteria** (what must be TRUE):
  1. The AgentHub icon appears in the macOS menu bar (system tray) and does NOT appear in the Dock or Cmd+Tab switcher
  2. Right-clicking the tray icon shows a menu with "Open AgentHub", active session names, and "Quit"
  3. Clicking "Open AgentHub" from the tray brings the GUI window to the foreground (shows it if hidden)
  4. Closing the GUI window (red traffic-light button) hides the window but leaves the tray icon and daemon running — sessions continue uninterrupted
  5. Clicking "Quit" from the tray menu stops the daemon, removes the tray icon, and fully exits the application
  6. The tray icon uses a monochrome template image that adapts correctly to both light and dark macOS menu bars
  7. The tray icon tooltip on hover shows the current active session count
  8. The tray icon switches to an error/disconnected visual state when the daemon is unreachable
**Plans**: 1 plan
Plans:
- [ ] 37-01-PLAN.md — Splash screen implementation (Go lifecycle + React overlay + tests + visual verification)
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
| 36. App Icons & Branding Assets | v1.7 | 1/1 | Complete    | 2026-04-01 |
| 37. Splash Screen | v1.7 | 1/1 | Complete   | 2026-04-01 |
| 38. Remote Session Metadata | v1.7 | 0/TBD | Not started | - |
| 39. Remote Session Indicators | v1.7 | 0/TBD | Not started | - |
| 40. Daemon Management Panel | v1.7 | 0/TBD | Not started | - |
| 41. System Tray + Lifecycle | v1.7 | 0/TBD | Not started | - |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
*Full v1.2 details: .planning/milestones/v1.2-ROADMAP.md*
*Full v1.3 details: .planning/milestones/v1.3-ROADMAP.md*
*Full v1.4 details: .planning/milestones/v1.4-ROADMAP.md*
*Full v1.5 details: .planning/milestones/v1.5-ROADMAP.md*
*Full v1.6 details: .planning/milestones/v1.6-ROADMAP.md*
