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
- 🚧 **v3.3.1 Bug Sweep** — Phases 109-114 (planning 2026-05-18, closes Issues #52, #54, #55, #56, #57, #58)

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

- [x] **Phase 87: Capability-Based Session Authorization** — Server-issued, signed capability tokens gate session listing, metadata, WebSocket access, and write permission. Tailnet membership is no longer sufficient. (UAT complete 2026-04-21)
- [x] **Phase 88: WebSocket Handshake Security** — Strict Origin allowlist (Tailscale FQDN, local-mode host, same-origin); cross-site upgrades return 403. Includes Wails desktop webview origin (`wails://wails`) — both production-darwin and dev-mode patterns. (UAT complete 2026-05-02 against rc3 + rc5 dmg)
- [x] **Phase 89: Vendored Terminal Assets + CSP** — xterm JS/CSS embedded in binary; strict CSP on all three HTML routes (`script-src 'self'`, `connect-src 'self' wss://<host>`, `style-src 'self' 'unsafe-inline'` per D-09). Zero CDN fetches verified across 1779 requests. (UAT complete 2026-05-02)
- [x] **Phase 90: Release Pipeline Hardening** — All third-party Actions SHA-pinned; build tools pinned via `tools.go`; release.yml split into validate → build-{macos,windows,linux} → sign-macos (gated by required-reviewer rule) → publish; SLSA L2 attestations verified before codesigning. (Pipeline E2E proven through 5 rc cycles before v3.1.0 final tag)

Distribution follow-ups deferred to a future milestone (see `.planning/deferred/91-distribution-pipeline-followups/91-CONTEXT.md`):
  - 91-A: Switch `release.yml` from `GITHUB_TOKEN` to PAT so `release.published` auto-triggers `distribute.yml`
  - 91-B: Fix `submit-winget` step's templating to handle `workflow_dispatch` events (use `RELEASE_TAG` env var instead of `github.event.release.tag_name`)
  - 91-C: One-time WinGet first-submission to `microsoft/winget-pkgs` (use `wingetcreate new`, not `update`)

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

Deferred to v3.3 (9 UAT scenarios + 6 polish items + shell-session backlog feature): see `.planning/milestones/v3.2-MILESTONE-AUDIT.md` and `.planning/v3.2-RELEASE-BLOCKERS.md`.

</details>

<details>
<summary>✅ v3.3 Shell Sessions & Polish (Phases 100-108) — SHIPPED 2026-05-17 (closes Issues #44 + #45)</summary>

- [x] Phase 100: Shell Session Backend & Discovery (4/4 plans) — completed 2026-05-13
- [x] Phase 101: Shell Session Surfaces & Web-Share Gating (4/4 plans) — completed 2026-05-13
- [x] Phase 102: Web-Links Polish — mailto + IDN (1/1 plan) — completed 2026-05-13
- [x] Phase 103: Find Bar Dismiss + Test-Env + IIP Polish (inline) — completed 2026-05-12
- [x] Phase 104: Settings Hyperlinked Index (1/1 plan) — completed 2026-05-12
- [x] Phase 105: Deferred v3.2 UAT Re-Run (runbook + UAT executed) — completed 2026-05-17 (3 pass + 4 issues filed to v3.4: #54/#55/#56)
- [x] Phase 106: Distribution Pipeline Followups (workflow edits) — completed 2026-05-13 (code-complete; operator: PAT + WinGet variable pending before next release)
- [x] Phase 107: Shell UX Collapse + Binary Path Picker + Clean-Exit Handling (4/4 plans) — completed 2026-05-13 (SHELL-12 runtime UAT signed off 5c58277)
- [x] Phase 108: TUI + CLI shell-entry collapse — parity with Phase 107 GUI (3/3 plans) — completed 2026-05-16 (TUI + CLI UAT smokes PASS 2026-05-17)

</details>

### 🚧 v3.3.1 Bug Sweep (Phases 109-114) — PLANNING 2026-05-18 (closes Issues #52, #54, #55, #56, #57, #58)

- [ ] Phase 109: Windows daemon named-pipe IPC (Issue #52, PR #53) — IPC-01..06
- [ ] Phase 110: Linux PTY natural-exit detection (Issue #57) — PTY-01..04
- [ ] Phase 111: Web bridge OSC/DA response consumption (Issue #54) — WEB-01..03
- [ ] Phase 112: WebGL recovery banner rendering (Issue #55) — UI-01, UI-02
- [ ] Phase 113: iPad terminal touch-scroll (Issue #56) — UI-03, UI-04
- [ ] Phase 114: CI test stability — webserver capability test flake (Issue #58) — TEST-01, TEST-02

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
| 109 | v3.3.1 | 0/1 | Planned | - |
| 110 | v3.3.1 | 0/0 | Not started | - |
| 111 | v3.3.1 | 0/0 | Not started | - |
| 112 | v3.3.1 | 0/0 | Not started | - |
| 113 | v3.3.1 | 0/0 | Not started | - |
| 114 | v3.3.1 | 0/0 | Not started | - |

### Phase 109: Windows daemon named-pipe IPC

**Goal:** AgentHub daemon, GUI, CLI, and TUI all work end-to-end on Windows 11 by listening on `\\.\pipe\agenthub-daemon` instead of failing to bind a Unix socket.

**Requirements:** IPC-01, IPC-02, IPC-03, IPC-04, IPC-05, IPC-06

**Depends on:** Nothing on the v3.3.1 critical path (highest-priority unblocker — Windows is currently 100% broken). Builds on v3.3 daemon HTTP/JSON API surface (Phase 19) and third-party PR #53 as the starting patch.

**Scope:** Adapt `internal/daemon` listener and `DaemonClient` dialer to use `winio` named pipes on Windows; Unix sockets on macOS/Linux. PR #53 evaluation task (rebase vs. re-apply) is mandatory, with `Co-Authored-By: im-alexandre` attribution either way. Cross-surface Windows verification (GUI + CLI + TUI) per v3.3 Phase 108 contract; macOS/Linux smoke for no regression.

**Plans:** 1 plan

Plans:
- [ ] 109-01-PLAN.md — Cherry-pick PR #53 (named-pipe IPC abstraction + Windows tests + kernel32 tray fix), preserve Alexandre Castro attribution, scaffold Windows UAT in VERIFICATION.md

### Phase 110: Linux PTY natural-exit detection

**Goal:** On Linux, a shell session whose child process exits cleanly causes the GUI tab / TUI list entry to auto-close — matching macOS behavior shipped in v3.3 SHELL-12.

**Requirements:** PTY-01, PTY-02, PTY-03, PTY-04

**Depends on:** v3.3 Phase 107 (SHELL-12 `autoCloseRef`-gated tab auto-close on `session:exit` with exit-code-0 normalization). This phase fixes the daemon-side blocker that prevents that frontend logic from firing on Linux.

**Scope:** Separate exit-detector goroutine that polls `syscall.Wait4(pid, &status, WNOHANG)` and explicitly closes the PTY to unblock the read loop, coordinated with go-pty's internal `waitOnContext`. Daemon-side fix in `internal/pty` / `internal/daemon/engine.go`. Affects GUI Linux + TUI Linux. Re-enable `TestListSessions_OnExitCallback_ReceivesNormalized` (currently `t.Skip()`'d on Linux).

**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 110 to break down)

### Phase 111: Web bridge OSC/DA response consumption

**Goal:** On the web-served terminal, OSC color-query (10/11) and Device Attributes (CSI c) responses are consumed by the web bridge and do not leak into shell stdin — `chafa --format=sixel <png>` produces a clean prompt on web, matching desktop.

**Requirements:** WEB-01, WEB-02, WEB-03

**Depends on:** v3.2 Phase 96 (image addon, sixel rendering on web) and Phase 93 (web vendoring + parity).

**Scope:** Web-only bug (desktop unaffected — Wails webview goes through xterm.js directly). Fix lives in the web bridge / relay code (`internal/webserver/relay.go` or equivalent). Cross-surface parity (chafa sixel on web vs. desktop) is the release gate. Regression test exercises OSC response consumption path.

**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 111 to break down)

### Phase 112: WebGL recovery banner rendering

**Goal:** When the terminal's WebGL context is lost, a `WebGLRecoveryBanner` renders inside `.banner-stack` and auto-dismisses after 8s — restoring the Phase 93 user-visible recovery flow.

**Requirements:** UI-01, UI-02

**Depends on:** v3.2 Phase 93 (WebGL recovery banner contract — banner exists but doesn't render; DOM fallback continues to work, masking the bug).

**Scope:** Pure frontend bug. Root cause suspected in `frontend/src/components/TerminalPanel.tsx:391-395` — likely `onContextLoss` closure rot (banner-state setter captured at mount time). Fix is local to `TerminalPanel.tsx` and adjacent banner state. Both desktop + web surfaces benefit from one fix (shared React component).

**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 112 to break down)

### Phase 113: iPad terminal touch-scroll

**Goal:** On iPad Safari and iPad Chrome, single-finger drag on the terminal area scrolls xterm scrollback — matching desktop wheel-scroll behavior.

**Requirements:** UI-03, UI-04

**Depends on:** v3.2 Phase 96/Phase 95 (web-links + image surface on iPad — touch-event semantics on terminal container are the same shared surface).

**Scope:** Pure frontend bug. Fix is `touchstart`/`touchmove` event wiring on the terminal container (or `touch-action: pan-y` CSS opt-in to delegate scroll to the browser's native touch-scroll). Web-only surface; desktop unaffected. Separate from Phase 112 — different root cause (touch capture vs. closure rot), different verification (physical iPad vs. desktop DevTools).

**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 113 to break down)

### Phase 114: CI test stability — webserver capability test flake

**Goal:** `TestPluginConfigStream_ExpiredCap_Returns401` passes deterministically (100/100 runs) on Linux CI under `-race -shuffle=on`, returning 401 (not 403), with the root cause identified and fixed (not papered over).

**Requirements:** TEST-01, TEST-02

**Depends on:** v3.2 Phase 92's `internal/webserver` capability/SSE plumbing.

**Scope:** Investigation-first phase. Likely root cause is test-state pollution across `internal/webserver` tests (testServer / EnableSession / SetSigningKey leaking under specific shuffle orderings). No rerun-pass hacks; the fix addresses why the test sometimes returns 403, not how to suppress the flake. Linux CI surface only.

**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 114 to break down)

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
