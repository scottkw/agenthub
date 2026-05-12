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
- 🚧 **v3.3 Shell Sessions & Polish** — Phases 100-106 (in progress, closes Issues #44 + #45, absorbs Phase 91 deferred work + v3.2 polish/UAT carry-over)

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

Distribution follow-ups deferred to a future milestone (see `.planning/deferred/91-distribution-pipeline-followups/91-CONTEXT.md`) — absorbed by v3.3 Phase 106:
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

### 🚧 v3.3 Shell Sessions & Polish (In Progress)

**Milestone Goal:** Land raw shell sessions (bash/zsh/pwsh) as a first-class agent type — unblocking the ~9 deferred v3.2 UAT scenarios that require raw PTY — then close v3.2 polish/tech-debt and re-run the deferred UAT batches end-to-end. Closes GitHub Issues #44 (shell agent) and #45 (Settings hyperlinked index). Absorbs Phase 91 v3.1 distribution-pipeline follow-ups.

**Phase numbering:** Continues from v3.2's last phase (99). v3.3 spans Phases 100-106. The deferred Phase 91 directory remains at `.planning/deferred/91-distribution-pipeline-followups/`; its work is absorbed into Phase 106 (fresh v3.3 phase, NOT a renumber of 91).

- [ ] **Phase 100: Shell Session Backend & Discovery** — Daemon-side shell PTY plumbing: cross-platform shell discovery, interactive (non-login) PTY spawn with working-directory honor, exclusion from CLI-status heuristics.
- [ ] **Phase 101: Shell Session Surfaces & Web-Share Gating** — User-facing shell selection across GUI/CLI/TUI plus distinct agent badge color, web-share-disabled-by-default override, one-time arbitrary-execution confirmation banner.
- [ ] **Phase 102: Web-Links Polish — mailto + IDN** — Close the v3.2 LNK spec gap: detect `mailto:` URLs (P-1) and admit non-ASCII hostnames so the `LinkConfirmPopover` fires for Cyrillic/IDN homographs (P-2).
- [ ] **Phase 103: Find Bar Dismiss + Test-Env + IIP Polish** — Find-bar Esc/close after case-sensitive toggle (P-3), find-bar slide-out animation (P-4), iTerm2 IIP investigation+decision (P-5), Vitest 4 + jsdom 29 localStorage test-env fix (P-6).
- [ ] **Phase 104: Settings Hyperlinked Index** — Issue #45: sticky jump-link bar at the top of Settings with anchor links + smooth scroll to each section header, plus an autocomplete search box.
- [ ] **Phase 105: Deferred v3.2 UAT Re-Run** — Execute the 9 UAT scenarios deferred from v3.2 (now unblocked by shell sessions): WebGL context-loss, iPad rasterizer banner, 10K-line scrollback perf, full LNK chain on iPad, chafa sixel/IIP fidelity, two-client mid-stream image join, iPad 5-scenario runbook.
- [ ] **Phase 106: Distribution Pipeline Followups** — Absorb the deferred Phase 91 bucket: PAT-credentialed `release.yml` (91-A), `RELEASE_TAG` env var in `distribute.yml` (91-B), one-time `wingetcreate new` for first WinGet submission (91-C).

## Phase Details

### Phase 100: Shell Session Backend & Discovery
**Goal**: Daemon can spawn raw shell PTYs as a distinct session type with cross-platform binary discovery, correct interactive (non-login) semantics, and clean exclusion from AI-CLI status heuristics.
**Depends on**: Nothing inside v3.3 (foundation phase; depends only on v3.2 daemon code)
**Requirements**: SHELL-04, SHELL-05, SHELL-09
**Success Criteria** (what must be TRUE):
  1. Daemon enumerates installed shells per platform (macOS: `/bin/bash`, `/bin/zsh`; Linux: `$SHELL` + `/etc/shells` entries; Windows: `pwsh.exe`, `powershell.exe`) and exposes them via session-creation API.
  2. A shell session spawned via the daemon API runs as an interactive, non-login PTY with the caller-supplied working directory honored.
  3. Shell sessions appear in `agenthub list` and the session registry without ever emitting `waiting` or `error` heuristic states — only `running` and `stopped`.
**Plans**: TBD

Plans:
- [ ] 100-01: TBD

### Phase 101: Shell Session Surfaces & Web-Share Gating
**Goal**: User can pick a shell as a first-class "agent" everywhere (GUI new-session modal, CLI `agenthub new shell`, TUI new-session flow), see it visually distinguished, and only enable web serving for it through an explicit one-time confirmation step.
**Depends on**: Phase 100 (daemon spawns + discovery API exists)
**Requirements**: SHELL-01, SHELL-02, SHELL-03, SHELL-06, SHELL-07, SHELL-08
**Success Criteria** (what must be TRUE):
  1. User selects bash / zsh / pwsh / "system default" in the GUI new-session modal agent picker and the session opens in a tab.
  2. User launches a shell session from CLI (`agenthub new shell <path>` or equivalent `--shell` flag) and from the TUI new-session modal.
  3. Shell sessions render a distinct agent badge color in the GUI tab bar and TUI session list (consistent with the existing 6-CLI palette).
  4. When the web server is running, newly-created shell sessions are NOT auto-enabled for web serving (overrides the agent-session default).
  5. The first time a user toggles web serving ON for a shell session, a one-time confirmation banner explains that shells expose arbitrary command execution; subsequent toggles do not re-prompt.
**Plans**: TBD

Plans:
- [ ] 101-01: TBD

**UI hint**: yes

### Phase 102: Web-Links Polish — mailto + IDN
**Goal**: Close the v3.2 Phase 95 spec gap so the Web-Links plugin matches its documented behavior: `mailto:` URLs are clickable and IDN/Cyrillic-homograph URLs trigger the existing `LinkConfirmPopover`.
**Depends on**: Phase 100 (UAT reproduction requires a shell session to print test URLs) — implementation itself is independent of shell work but cross-checking against `/tmp/web-links-test.sh` fixture needs raw PTY.
**Requirements**: POLISH-01, POLISH-02
**Success Criteria** (what must be TRUE):
  1. User Cmd-clicks (macOS) / Ctrl-clicks (elsewhere) a `mailto:noreply@example.com` URL in terminal output and the system mail client opens.
  2. When a URL contains non-ASCII hostname characters (e.g. `https://gооgle.com` with Cyrillic `о` U+043E), it renders as a clickable link, and on activation the `LinkConfirmPopover` displays both the display form and the resolved Punycode form before any navigation.
  3. The fix is applied symmetrically across the desktop (`frontend/src/components/TerminalPanel.tsx`) and web (`web/assets/terminal.js`) link matchers.
**Plans**: TBD

Plans:
- [ ] 102-01: TBD

### Phase 103: Find Bar Dismiss + Test-Env + IIP Polish
**Goal**: Close the four remaining v3.2 polish items that are not in the link path: find-bar focus/event-propagation dismiss bug, find-bar exit animation, iTerm2 IIP rendering investigation+decision, and Vitest 4 + jsdom 29 `localStorage` test-env regression.
**Depends on**: Phase 100 (POLISH-05 IIP repro fixture requires a shell session to print `ESC ] 1337` sequences)
**Requirements**: POLISH-03, POLISH-04, POLISH-05, POLISH-06
**Success Criteria** (what must be TRUE):
  1. After clicking the find-bar case-sensitive toggle, pressing Esc OR clicking the close button dismisses the find bar (both on desktop `FindBar.tsx` and `web/assets/terminal.js`).
  2. The find bar slides out with the same 200ms animation it uses on entry (`.find-bar--exiting` class applied with delayed unmount) on both Esc and close-button dismiss.
  3. iTerm2 IIP (OSC 1337) is EITHER demonstrated to render correctly with the Image plugin enabled OR explicitly documented as sixel-only support (Phase 96 SUMMARY + `web/vendor/xterm/addons/addon-image/README` updated with the decision).
  4. All 20 currently-failing `Sidebar.test.tsx` tests pass under Vitest 4 + jsdom 29 via a setupFile that exposes `localStorage` as a global; no source code under `frontend/src/components/` is touched.
**Plans**: TBD

Plans:
- [ ] 103-01: TBD

**UI hint**: yes

### Phase 104: Settings Hyperlinked Index
**Goal**: User can navigate the Settings tab via a sticky jump-link bar with section anchors and an autocomplete search box (Issue #45).
**Depends on**: Nothing inside v3.3 (independent UI work; can ship in parallel with shell phases)
**Requirements**: SETUI-01, SETUI-02, SETUI-03
**Success Criteria** (what must be TRUE):
  1. A sticky jump-link bar at the top of the Settings tab shows anchor links to each section header (Plugins, Appearance, Web Server, Behavior, Paths).
  2. Clicking any jump-link smoothly scrolls the Settings tab to that section without losing the sticky bar.
  3. Typing into the autocomplete search box at the top of Settings filters matching setting labels and, on selection, jumps to the corresponding section.
**Plans**: TBD

Plans:
- [ ] 104-01: TBD

**UI hint**: yes

### Phase 105: Deferred v3.2 UAT Re-Run
**Goal**: Execute and sign off the 9 v3.2 UAT scenarios that were deferred to v3.3 — all now unblocked by shell sessions (Phase 100/101) and the link/find-bar polish (Phases 102/103).
**Depends on**: Phase 101 (shell sessions surfaced to GUI/web/CLI for UAT printing of test fixtures), Phase 102 (LNK chain re-run requires mailto + IDN fixes landed), Phase 103 (full polish closure for find-bar UAT-3 and IIP/sixel decision before Phase 96 fidelity test)
**Requirements**: UAT-01, UAT-02, UAT-03, UAT-04, UAT-05, UAT-06, UAT-07
**Success Criteria** (what must be TRUE):
  1. Phase 93 UAT-1 verified in desktop Chrome: `WEBGL_lose_context.loseContext()` invocation in DevTools triggers the WebGL recovery banner with 8s auto-dismiss.
  2. Phase 93 UAT-2 verified on physical iPad Safari over a Tailscale-served session: software-rasterizer banner appears.
  3. Phase 94 UAT-3 verified: 10,000-line scrollback regex search completes within the documented main-thread frame-time budget (DevTools Performance capture attached).
  4. Phase 95 UAT-4 verified on iPad Safari + Tailscale: full LNK-01..05 chain (Cmd-click activation, IDN popover, typosquat popover, OSC 8 hover-href, `mailto:`) operates end-to-end using a shell session to print the fixtures.
  5. Phase 96 UAT-5 + UAT-6 verified using a shell session: `chafa --format=iterm2` (or sixel fallback per Phase 103 decision) renders identically across desktop and web; two-client mid-stream join replays the image with byte-fidelity over the WSS relay.
  6. Phase 99 UAT-7 verified on physical iPad + Tailscale: all 5 scenarios in `.planning/phases/99-*/99-iPad-UAT.md` (attach+chafa, scrollback, zero-CDN, zero-CSP, all-8-ON Progress) pass.
**Plans**: TBD

Plans:
- [ ] 105-01: TBD

### Phase 106: Distribution Pipeline Followups
**Goal**: Close the deferred Phase 91 bucket carried from v3.1 — release pipeline auto-triggers distribute.yml on a real `release.published` event, WinGet manifest submission succeeds end-to-end without manual template fixups, and the first microsoft/winget-pkgs submission uses `wingetcreate new`. Absorbs `.planning/deferred/91-distribution-pipeline-followups/` (the deferred directory is archived as part of this phase).
**Depends on**: Nothing inside v3.3 (CI/release-pipeline work, isolated from product code; can run anywhere in the sequence — sequenced late so the release gate captures all v3.3 product changes)
**Requirements**: DIST-01, DIST-02, DIST-03
**Success Criteria** (what must be TRUE):
  1. A test release tag pushed via `release.yml` (using the PAT/GitHub-App credential, not `GITHUB_TOKEN`) automatically fires `release.published`, which triggers `distribute.yml` end-to-end — no manual `workflow_dispatch` needed.
  2. `distribute.yml`'s submit-winget step reads `RELEASE_TAG` from env on both `release:published` and `workflow_dispatch`; the resulting installer URL has no double-dash and `$version` is populated.
  3. The first WinGet submission to microsoft/winget-pkgs is made via `wingetcreate new` (validated locally before pushing the PR) and the deferred `.planning/deferred/91-distribution-pipeline-followups/` directory is moved into the Phase 106 archive once the PR lands.
**Plans**: TBD

Plans:
- [ ] 106-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 100 → 101 → 102 → 103 → 104 → 105 → 106. Phase 104 (Settings hyperlinked index) is independent and may run in parallel with the shell phases if planning chooses; Phase 106 (distribution) is also independent and may run anywhere in the sequence.

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
| 100. Shell Session Backend & Discovery | v3.3 | 0/TBD | Not started | - |
| 101. Shell Session Surfaces & Web-Share Gating | v3.3 | 0/TBD | Not started | - |
| 102. Web-Links Polish — mailto + IDN | v3.3 | 0/TBD | Not started | - |
| 103. Find Bar Dismiss + Test-Env + IIP Polish | v3.3 | 0/TBD | Not started | - |
| 104. Settings Hyperlinked Index | v3.3 | 0/TBD | Not started | - |
| 105. Deferred v3.2 UAT Re-Run | v3.3 | 0/TBD | Not started | - |
| 106. Distribution Pipeline Followups | v3.3 | 0/TBD | Not started | - |

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
