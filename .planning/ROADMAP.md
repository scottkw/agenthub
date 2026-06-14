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
- 🚧 **v3.5 File Browser — Write Operations & Editor** — Phases 123-128 (in progress, closes Issues #63, #64 + umbrella #24)

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
<summary>🚧 v3.5 File Browser — Write Operations & Editor (Phases 123-128) — IN PROGRESS</summary>

- [x] **Phase 123: TD Cleanup + Write Sandbox Primitives + Daemon Routes** — FSW-01..FSW-12 (completed 2026-06-14)
- [x] **Phase 124: `files.write` Capability + Webserver Write Routes + Web-Share Opt-In** — CAP-01..CAP-10 (completed 2026-06-14)
- [ ] **Phase 125: React Editor (CodeMirror 6) — Desktop + Web** — EDIT-01..EDIT-13
- [ ] **Phase 126: TUI Write Parity (`$EDITOR` Shell-Out)** — TUIW-01..TUIW-07
- [ ] **Phase 127: Web-Share Write Security Hardening** — SEC-01..SEC-07
- [ ] **Phase 128: Remote Write Parity + Cross-Surface Integration** — RMW-01..RMW-06

</details>

## Phase Details

### Phase 123: TD Cleanup + Write Sandbox Primitives + Daemon Routes

**Goal:** The `internal/files/` sandbox has all write primitives (atomic write, rename, delete, mkdir, upload), the shell-RC denylist is enforced on all write paths, the two carried tech-debts (TD-4 and TD-5) are closed, and the daemon local-socket write routes are live — so every subsequent phase has a correct, trusted, fuzz-proven write API to build against.

**Depends on:** Phase 122 (v3.4 complete). No v3.5 prerequisites. Load-bearing security foundation — must land before any write endpoint is exposed on any surface.

**Requirements:** FSW-01, FSW-02, FSW-03, FSW-04, FSW-05, FSW-06, FSW-07, FSW-08, FSW-09, FSW-10, FSW-11, FSW-12

**Success Criteria** (what must be TRUE):

1. `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/...` reports zero crashes against a corpus that includes write-path traversal, rename-destination traversal (`oldRel=ok, newRel=../../.ssh/authorized_keys`), upload-filename injection (`../../../.bashrc`), and all v3.4 `FuzzSandboxPath` payloads extended to the write surface.
2. `curl --unix-socket ~/.agenthub/daemon.sock -X PUT 'http://localhost/api/files/write?session=<id>&path=hello.txt' -d 'hello'` succeeds with HTTP 200 and the file is durably written; a subsequent `GET /api/files/read` returns the identical content — confirming atomic temp+sync+rename semantics with no partial-file window.
3. A write attempt targeting `~/.bashrc`, `~/.ssh/authorized_keys`, or `~/.claude/CLAUDE.md` within a home-directory sandbox returns `403 Protected system file` — the shell-RC denylist is enforced on all five write methods (write, rename, delete, mkdir, upload).
4. `DaemonClient.ExchangeJoinCodeAtURL` correctly parses a `303 Location: ...?cap=<token>` response (TD-5 fixed): a desktop GUI can now acquire a remote session cap without silent failure, unblocking all remote-write testing in Phase 128.
5. `go test ./internal/files/... ./internal/daemon/...` is green with the race detector enabled; the five daemon write routes (`PUT /api/files/write`, `POST /api/files/upload`, `DELETE /api/files/delete`, `POST /api/files/rename`, `POST /api/files/mkdir`) are accessible on the local Unix socket with no authentication required (loopback trust, WEB-01 precedent).

**Plans:** 4/4 plans complete

Plans:
- [x] 123-01-PLAN.md — Sandbox write primitives (atomic write, rename, mkdir, delete) + shell-RC denylist + FuzzSandboxWrite (FSW-01..04, 06, 07)
- [x] 123-02-PLAN.md — TD cleanup: TD-5 ExchangeJoinCodeAtURL 303 parse + TD-4 WR-03/04/05 hardening (FSW-10, 11)
- [x] 123-03-PLAN.md — HTTP write handlers + 5 auth-less daemon routes + 50 MiB upload cap (FSW-05, 08, 12)
- [x] 123-04-PLAN.md — DaemonClient write methods (FSW-09)

**UI hint**: no

---

### Phase 124: `files.write` Capability + Webserver Write Routes + Web-Share Opt-In

**Goal:** The `files.write` capability bit exists, `requireFilesWrite` middleware (with CSRF Origin check) gates all five webserver write routes, `files.write` is opt-in for every token (a per-session "Enable file writes" toggle gates the owner cap; web-share viewers require a further explicit opt-in), and `schemaVersion: 4` migration is in place — so any surface that authenticates via the webserver can exercise write operations only after writes are explicitly enabled.

**Depends on:** Phase 123 (write sandbox primitives and daemon routes frozen).

**Requirements:** CAP-01, CAP-02, CAP-03, CAP-04, CAP-05, CAP-06, CAP-07, CAP-08, CAP-09, CAP-10

**Success Criteria** (what must be TRUE):

1. A cap token issued WITHOUT writes enabled (the default for both owner and viewer) returns HTTP 403 on all five webserver write routes — not 404, not 401; once writes are explicitly enabled for the session, the resulting `files.write`-bearing cap returns HTTP 2xx on all five routes.
2. A POST/PUT/DELETE request to a write route with an `Origin` header that does not match the server FQDN is rejected with HTTP 403 (CSRF Origin check); a request with no `Origin` header (desktop Wails fetch) passes vacuously — confirming the Phase 88 pattern is correctly applied to the write surface.
3. `TestHasPerm_NoStringsContains_Write` static-grep gate passes: no write-path code calls `strings.Contains(perms, "files.write")` — all permission checks use the `HasPerm` whole-token comma-split helper.
4. The web-share grant UI shows an explicit `files.write` opt-in toggle (default OFF), and toggling it on includes the string `"files.write"` in the issued viewer cap token; the home-directory write warning is visible in both GUI and TUI when `files.write` is active for a session whose cwd is `$HOME`.
5. `TestSettingsMigration_FilesWriteDefaultsFalse` passes: a settings file at `schemaVersion: 3` migrates to `schemaVersion: 4` with `FilesWrite: false` default; web-share `files.write` opt-in state persists across daemon restarts.

**Plans:** 5/5 plans complete

Plans:
- [x] 124-01-PLAN.md — files.write cap const + requireFilesWrite middleware (HasPerm + CSRF Origin) + 5 webserver route mounts + integration/static-grep tests (CAP-01,02,03,07,09)
- [x] 124-02-PLAN.md — per-session write opt-in state + schemaVersion 4 migration + cap-mint wiring + EvalSymlinks home-dir signal on SessionInfo/IssueCapabilities (CAP-04,06,08)
- [x] 124-03-PLAN.md — proxyRemoteFiles body+Content-Type forwarding fix + 5 remote write proxy routes (CAP-10)
- [x] 124-04-PLAN.md — GUI owner write toggle + Wails binding chain + viewer "Allow file editing" opt-in + home-dir warning banner (CAP-04,05,06)
- [x] 124-05-PLAN.md — TUI home-dir write warning line, cross-surface parity (CAP-06)

**UI hint**: yes

---

### Phase 125: React Editor (CodeMirror 6) — Desktop + Web

**Goal:** Users can open any text file in a CodeMirror 6 editor with syntax highlighting, save changes atomically via Cmd/Ctrl+S with conflict detection, and perform all write operations (create file, mkdir, delete, rename, cross-directory move, single and multi-file upload) from the `FileBrowserTab` — on both the desktop app and the web-share surface.

**Depends on:** Phase 123 (write API frozen) and Phase 124 (capability model live and webserver write routes accessible). This is the milestone centrepiece.

**Requirements:** EDIT-01, EDIT-02, EDIT-03, EDIT-04, EDIT-05, EDIT-06, EDIT-07, EDIT-08, EDIT-09, EDIT-10, EDIT-11, EDIT-12, EDIT-13

**Success Criteria** (what must be TRUE):

1. Opening a text file in the file browser and clicking the pencil (Edit) button mounts a CodeMirror 6 editor with syntax highlighting matching the file's extension (Go, TypeScript, Python, JSON, YAML, Markdown, Bash, HTML, CSS, and other common languages); the Edit button is absent for binary files and for callers without `files.write`; files > 500 KB show a large-file warning before entering edit mode; files approaching the 5 MB cap disable syntax highlighting with an in-editor notice.
2. Pressing Cmd/Ctrl+S saves the file atomically (temp file + sync + rename) with an `If-Match: <etag>` header; the editor header shows a three-state save indicator (idle / saving... / saved, ~1.5s transient); a dirty-state bullet/asterisk appears when the buffer differs from the last-saved snapshot; navigating away with unsaved changes triggers a "You have unsaved changes. Save or discard?" guard — no `beforeunload` (Wails blocks it), React-level guard only.
3. A concurrent-write collision (If-Match mismatch, HTTP 412) surfaces "This file was modified by another process" with three choices: [Force overwrite] / [Save as new file] / [Discard my changes] — the editor buffer is never silently discarded.
4. All write affordances (create file, mkdir, delete, rename, cross-directory move via "Move to…" picker, single file upload, multi-file upload with per-file progress, drag-and-drop into the directory listing) are visible and operable only when `canWrite` is true; a 409 name-collision on rename or upload shows "A file named X already exists. Replace it?" with Cancel as the default action.
5. Playwright cross-browser e2e (Chromium + Firefox + WebKit) passes all scenarios: local write-and-save, web-share write with a `files.write` cap, 403 without the cap, create file, mkdir, delete file, delete directory (recursive confirm with file count), rename, cross-directory move, single upload, multi-file upload, 412 conflict flow, binary-file no-edit, large-file guard — zero CSP violations in any browser; `vendor_drift_test.go` passes with CodeMirror packages version-matched.

**Plans:** 5/6 plans executed

Plans:
- [x] 125-01-PLAN.md — Server If-Match/412 + ETag + CodeMirror vendor-drift gate + Playwright WRITE_CAP fixture (Wave 0)
- [x] 125-02-PLAN.md — CodeMirror 6 install + Editor core (mount, Compartment toggle, language-by-extension, large-file/binary guards, Edit button, canWrite)
- [x] 125-03-PLAN.md — Save flow: Cmd/Ctrl+S + If-Match, three-state indicator, dirty state, unsaved guard, 412 conflict modal
- [x] 125-04-PLAN.md — Directory write affordances: create file, mkdir, rename, delete (recursive count), cross-directory move
- [x] 125-05-PLAN.md — Upload (single + multi, XHR per-file progress) + drag-and-drop
- [ ] 125-06-PLAN.md — Playwright cross-browser e2e (14 scenarios) + CSP-violation gate + desktop parity checkpoint

**UI hint**: yes

---

### Phase 126: TUI Write Parity (`$EDITOR` Shell-Out)

**Goal:** TUI users can edit files via `$EDITOR` shell-out, delete, rename, and create directories using keyboard shortcuts within the Files view — with full cross-surface parity against the GUI write operations (minus upload, which is formally descoped with an on-screen message).

**Depends on:** Phase 123 (`DaemonClient` write methods available, `FilesClient` interface extension ready). Can run in parallel with Phase 125 — the two phases share only the Phase 123 prerequisite and touch different files.

**Requirements:** TUIW-01, TUIW-02, TUIW-03, TUIW-04, TUIW-05, TUIW-06, TUIW-07

**Success Criteria** (what must be TRUE):

1. Pressing `e` on a selected file in the TUI Files view suspends the TUI, spawns the resolved `$EDITOR` (fallback chain: `$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi`) with the file's sandbox-absolute path, and resumes the TUI on editor exit — terminal state is cleanly restored via `tea.ClearScreen` and the directory listing refreshes unconditionally after every edit (no stale listing).
2. When no editor is resolvable, the TUI shows a clear inline error: "`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)." — not a crash or a silent no-op.
3. Pressing `d` on a selected file or directory shows a confirmation dialog (reusing the kill-session pattern); confirming deletes the entry recursively for directories. Pressing `r` opens inline rename; pressing `m` opens inline mkdir name input — both operations refresh the listing on completion.
4. Pressing `u` (upload) in the TUI Files view shows the on-screen message "Use desktop or web to upload files." — the one documented parity gap — and a follow-up GitHub issue is filed.
5. The `FilesClient` interface has exactly 8 methods (4 read + 4 write); both `*daemon.DaemonClient` and `*tui.RemoteFilesClient` satisfy the full interface; `TestFiles_NoSyncFSCalls` static-grep gate passes with write commands included — all write filesystem I/O routes through `tea.Cmd`, never synchronous in `Update`.

**Plans:** TBD

**UI hint**: no

---

### Phase 127: Web-Share Write Security Hardening

**Goal:** The web-share write surface has been security-audited end-to-end: symlink escapes return 403, the shell-RC denylist blocks all known bypass vectors, upload abuse is covered, capability escalation is impossible, concurrent-write races leave no partial files, and a Playwright e2e confirms the full web-share write flow with and without the `files.write` cap.

**Depends on:** Phase 124 (capability model and webserver write routes live) and Phase 125 (browser-facing write surface complete). This is the dedicated security audit phase for the most-exposed surface.

**Requirements:** SEC-01, SEC-02, SEC-03, SEC-04, SEC-05, SEC-06, SEC-07

**Success Criteria** (what must be TRUE):

1. A write or rename whose resolved target escapes the sandbox via a symlink returns HTTP 403 — not HTTP 200 — confirming the `os.OpenRoot` TOCTOU boundary holds on the write path as well as the read path.
2. Attempts to write, rename, or delete `~/.bashrc`, `~/.ssh/authorized_keys`, `~/.claude/CLAUDE.md`, and the daemon's own config directory within a home-directory sandbox all return `403 Protected system file` — the denylist cannot be bypassed by case variation, Unicode normalization, or path encoding.
3. `FuzzSandboxWrite` with the finalized corpus (rename-destination traversal, denylist-bypass attempts, upload-filename injection via `../` in multipart `FileHeader.Filename`) reports zero crashes; an over-cap upload (> 50 MiB) is rejected by `MaxBytesReader` before `ParseMultipartForm` with a clear error, not a truncated file.
4. The capability escalation audit confirms: no token lacking `files.write` reaches any write endpoint on any surface (daemon socket, webserver, remote proxy); `files.write` does not leak across sessions; findings are documented in a SECURITY artifact committed under `.planning/`.
5. Playwright web-share write e2e passes: a viewer granted `files.write` writes successfully; a viewer without it gets HTTP 403; a CSRF Origin-mismatch request (Origin header present, does not match FQDN) is rejected with HTTP 403 on POST/PUT/DELETE write routes.

**Plans:** TBD

**UI hint**: no

---

### Phase 128: Remote Write Parity + Cross-Surface Integration

**Goal:** Remote tailnet peer write operations (edit/save, upload, delete, rename, mkdir) work end-to-end from both the desktop GUI and the TUI, with write parity proven by 3 independent network-stack observers — mirroring the Phase 122 read-parity proof pattern. The milestone ships with a two-machine UAT checklist ready and no regression on Phase 122 remote read tests.

**Depends on:** All previous phases (123 through 127) complete. FSW-10 (TD-5) fixed in Phase 123 is a direct prerequisite — without it, the desktop GUI cannot acquire a remote cap.

**Requirements:** RMW-01, RMW-02, RMW-03, RMW-04, RMW-05, RMW-06

**Success Criteria** (what must be TRUE):

1. Remote write parity is proven by 3 independent observers — daemon-proxy Go, `tui.RemoteFilesClient` Go, and Playwright HTTPS browser — all producing byte-equivalent results for a write-then-read round trip against the same remote session, mirroring the Phase 122 read-parity proof.
2. The desktop GUI can perform edit/save, upload, delete, rename, cross-directory move, and mkdir on a remote tailnet peer session via the daemon proxy; the TUI can perform the same write operations (minus upload) via `RemoteFilesClient` over HTTPS (TLS 1.2+ pinned, cap token redacted from error messages).
3. A write attempt against a v3.4 remote peer (no write endpoints) returns HTTP 405 and the client surfaces the message "The remote session is running an older version of AgentHub that does not support file writes." — not a generic network error or an opaque 405.
4. If a remote cap expires mid-edit, the editor buffer is preserved and an "access expired" message is shown; any orphaned partial upload on the remote machine is cleaned up — no silent buffer loss, no stranded temp files.
5. The Phase 122 remote read test suite passes with zero regressions; a two-machine tailnet write UAT checklist is committed (Machine A web-share + Machine B GUI + Machine B TUI; cross-surface write parity + cap-expiry failure mode), closing umbrella Issue #24 when executed successfully.

**Plans:** TBD

**UI hint**: no

---

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
| 125 | v3.5 | 5/6 | In Progress|  |
| 126 | v3.5 | 0/TBD | Not started | - |
| 127 | v3.5 | 0/TBD | Not started | - |
| 128 | v3.5 | 0/TBD | Not started | - |

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
