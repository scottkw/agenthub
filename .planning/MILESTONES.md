# Milestones

## v1.11 Local Network & UX Polish (Shipped: 2026-04-10)

**Phases completed:** 6 phases, 9 plans, 6 tasks

**Requirements:** 10/10 satisfied
**Commits:** 57 | **Timeline:** 2 days (2026-04-08 → 2026-04-10)
**Source changes:** 31 files, +1,651 / -980 lines

**Key accomplishments:**

- Claude Code native installer detection: `~/.local/bin` added as first AugmentServicePath candidate for Anthropic native installer discovery
- Settings converted from modal overlay to singleton sidebar tab (SettingsTab.tsx), consistent with Home/Remote/Sessions panels
- Web server auto-starts on daemon launch; new sessions auto-enabled for web serving in both Tailscale and local modes
- Local network fallback: self-signed TLS (P256 + IP SAN), HTTP Basic Auth with generated password, LAN IP selection, persistent nudge banner, password display in Settings
- Frontend webEnabled seeding chain restored across all 5 IPC layers (Go bindings → TypeScript types → React state) for correct StatusBar display
- Tech debt cleanup: deleted orphaned SettingsPanel/HealthModal components, fixed 11 stale test assertions

**Known tech debt (7 items, all non-blocking):**
- SUMMARY frontmatter missing `requirements_completed` for phases 59, 60, 61
- App.tsx `retryInit()` missing `webServerRunning` override (narrow race, no data loss)
- App.tsx stale closure risk in `createTab` useCallback (WR-01)
- App.tsx init/retryInit code duplication (WR-02)

---

## v1.10 Collapsible Sidebar Navigation (Shipped: 2026-04-08)

**Phases completed:** 2 phases, 3 plans, 4 tasks
**Requirements:** 11/11 satisfied
**Commits:** 21 | **Timeline:** 1 day (2026-04-08)
**Source changes:** 26 files, +3,243 / -205 lines

**Key accomplishments:**

- Collapsible left sidebar with Heroicons SVG icons (@heroicons/react 2.2.0) replacing toolbar action buttons
- App layout restructured to flex-row with Sidebar + app__content, collapsed (48px) and expanded (200px) modes
- All 5 navigation items wired: Home, Remote, Sessions, New Tab, Settings — each opens corresponding tab/panel
- Tab bar cleaned up: action buttons removed, retains session tabs only; dead CSS and obsolete tests removed
- Sidebar collapsed/expanded state persists via localStorage across app restarts

---

## v1.9 Remote Sessions & App Polish (Shipped: 2026-04-08)

**Phases completed:** 6 phases, 14 plans, 17 tasks
**Requirements:** 17/17 satisfied
**Commits:** 111 | **Timeline:** 2 days (2026-04-06 → 2026-04-07)
**Source changes:** 36 files, +3,499 / -97 lines

**Key accomplishments:**

- Standard macOS app menus (File, Edit, Window, Help) with Cmd+C/V clipboard in terminal tabs and build-time version injection via ldflags
- Tailscale peer discovery (`internal/tailnet`) with injectable deps, concurrent probe pool (cap 5), and daemon `GET /tailnet/peers` with 30s thundering-herd-safe cache
- Auto-update checker polling GitHub releases on startup + hourly, notification banner in WelcomeTab, and Help menu "Check for Updates" item
- Remote Sessions GUI panel with tailnet peer grouping, loading states, 30s auto-refresh, and one-click browser open via BrowserOpenURL
- CLI remote sessions: unified `agenthub list` with HOST column grouping, `agenthub attach hostname:session-id` via WSS relay with hostname banner
- Tailscale onboarding enhancement: platform-specific install commands with copy-to-clipboard, macOS auto-install via Homebrew with streaming progress, and numbered HTTPS cert setup guide

---

## v1.8 GitHub Distribution & CI/CD (Shipped: 2026-04-06)

**Phases completed:** 5 phases, 9 plans, 11 tasks

**Key accomplishments:**

- Go module path rewritten from `github.com/agenthub/agenthub` to `github.com/scottkw/agenthub` across go.mod and all 30 import sites in 16 .go files; race detector clean
- RELEASE_PLEASE_TOKEN configured and release-please created PR #1 (chore(main): release 1.8.0) on first push to main
- One-liner:
- Homebrew cask template and three-file WinGet manifests (schema 1.12.0) with sed-replaceable {{VERSION}} tokens matching Phase 46 artifact naming exactly
- distribute.yml GitHub Actions workflow that auto-updates scottkw/homebrew-agenthub cask formula on each release:published event using checksums.txt SHA256 extraction, nick-fields/retry, and TAP_DEPLOY_TOKEN PAT
- winget-releaser job added to distribute.yml with restrictive installer regex, plus populate-manifests.sh helper for one-time manual WinGet first submission
- WINGET_TOKEN PAT created with public_repo scope, stored as repo secret; scottkw/winget-pkgs fork verified; manifest submission deferred pending first release

---

## v1.7 Daemon UX & Branding (Shipped: 2026-04-03)

**Phases completed:** 8 phases, 10 plans, 20 tasks

**Key accomplishments:**

- 1024x1024 branded A logomark extracted from title logo, compiled into full 10-entry macOS ICNS (590KB), 6-frame Windows ICO, and 6 Linux PNGs via sips+iconutil+ImageMagick pipeline with post-build bundle injection
- Branded splash screen with StartHidden + OnDomReady lifecycle, static HTML bridge div, React SplashScreen overlay, and triple-path init dismissal with 3s fallback
- Machine hostname added to daemon session API — SessionInfo includes Hostname field populated at engine startup via os.Hostname()
- Web terminal status bar with session name, agent type, hostname display and 3-second REST-polled connection state indicator using TokyoNight theme
- Connection banner and detach message for CLI attach — shows session name, CLI type, hostname, and Ctrl-\ hint on stderr before raw mode
- DaemonManagerPanel tab in TabBar showing live session list with status dots, kill buttons, and web-serve toggles via ☰ button — zero new Go bindings
- Daemon POST /shutdown endpoint, two 18x18 monochrome tray icon PNGs embedded in tray.go, and LSUIElement=true in production Info.plist
- AgentHubMenuDelegate NSMenuDelegate for dynamic session menu, updateTray() for icon/tooltip state, startTrayPoller() 5s background refresh, and tray:focus-session frontend event handler
- Split refreshTrayState nil-client guard so tray shows error icon (trayIconErrorBytes) and updated tooltip when daemon fails to start
- Hostname field forwarded from daemon SessionInfo through App.go Wails binding, displayed as pill badge in DaemonManagerPanel with em dash fallback for empty values

---

## v1.6 Terminal Fill Fix v2 (Shipped: 2026-03-31)

**Phases completed:** 1 phases, 1 plans, 2 tasks

**Key accomplishments:**

- Replaced double-rAF one-shot fit with bounded rAF retry loop polling FitAddon.proposeDimensions() until cell dimensions are non-zero, fixing initial-load terminal fill for Claude, Gemini, and OpenCode CLIs

---

## v1.5 Bug Fixes & CLI Args (Shipped: 2026-03-26)

**Phases completed:** 5 phases, 6 plans, 8 tasks

**Key accomplishments:**

- Args wiring: `args []string` threaded through all 5 daemon IPC layers with integration tests proving args survive the full HTTP round-trip from JSON to PTY
- CLI passthrough: `agenthub new <agent> <path> -- <extra-args>` via `splitDashDash` helper
- Eliminated 2-second session status startup delay by restructuring pollSessionStatus to poll immediately then sleep 500ms between iterations
- TDD implementation of runtime PATH augmentation at daemon startup so service-mode agents (nvm, Volta, Homebrew) resolve via exec.LookPath without shell init files
- Thread cols/rows from React frontend through Wails/Go stack to PTY spawn with double-rAF initial fit timing

---

## v1.4 Unified Binary (Shipped: 2026-03-25)

**Phases completed:** 3 phases, 3 plans, 6 tasks

**Key accomplishments:**

- Merged cmd/agenthub-cli/ into root package: single agenthub binary dispatches no-args→GUI, flags→GUI, --help→usage(), commands→runCLI() with full migrated+new test suite
- Deleted cmd/agenthub-cli/ dead package (8 files, 1559 lines) and scrubbed its README.md row — repo now has zero references to the old standalone CLI binary
- Portable BASH_SOURCE path resolution in build-script tests and race-enabled CI workflow with ubuntu-latest build-script verification

---

## v1.3 CLI + Daemon (Shipped: 2026-03-25)

**Phases completed:** 8 phases, 15 plans, 23 tasks

**Key accomplishments:**

- Daemon architecture: SessionEngine extracted into `internal/daemon` with HTTP/JSON API over Unix socket, typed DaemonClient, 28 tests with -race
- Process separation: Sessions survive GUI close/reopen; RunDaemon/EnsureDaemon lifecycle; App reduced to thin DaemonClient shell
- Full CLI: 13 commands (new, list, kill, rename, attach, web start/stop/status, serve/unserve, health, qr, settings) with `--json` output
- Interactive attach: Full PTY proxy with raw I/O, detach key (Ctrl-\), SIGWINCH resize, Ctrl-C passthrough, scrollback replay, signal-safe terminal restore
- Service manager: `agenthub daemon install/uninstall/start/stop` via kardianos/service for launchd/systemd/Windows SCM
- Robustness: Windows named pipe dial fix + graceful GUI startup failure with error banner and retry

**Stats:**

- 101 commits, 38 files changed, +4,197/-295 lines, ~12,619 LOC (9,068 Go + 2,619 TS/TSX + 932 CSS)
- Timeline: 8 days (2026-03-17 → 2026-03-25)

**Tech debt (accepted):**

- ROADMAP.md plan checkboxes unchecked for 22-02, 23-02, 26-01 (code implemented and working)
- SUMMARY.md frontmatter missing requirements_completed field across all plan summaries
- All 6 Nyquist VALIDATION.md files are draft status
- Service manager live start/stop round-trip needs human verification on macOS

---

## v1.2 Tailscale-Only Networking (Shipped: 2026-03-23)

**Phases completed:** 5 phases, 10 plans

**Key accomplishments:**

- Tailscale health check infrastructure — detects installation, connection, and cert readiness with background polling via `local.Client{}`
- Let's Encrypt TLS via Tailscale daemon — replaced self-signed cert system with `GetCertificate` hook, FQDN-based URLs, CT disclosure flow
- Auth layer removal — deleted password auth, per-session tokens, and all auth middleware; tailnet membership = access control
- Dead code cleanup — removed generic VPN interface picker (`network.go`), `GetNetworkInterfaces`, and all orphaned frontend bindings
- Health modal with platform-specific guidance — three-state instructional UI (not installed / not connected / no certs) with Check Again button and auto-dismiss
- Tailscale status indicator in Settings panel replacing removed VPN interface picker

**Stats:**

- 64 commits, 74 files changed, ~8,846 LOC (5,364 Go + 2,550 TS/TSX + 932 CSS)
- Timeline: 6 days (2026-03-17 → 2026-03-23)
- Git range: `4e84151..6894faf`

**Tech debt (info-level only):**

- TailscaleHealth type defined inline in App.tsx and App.d.ts rather than imported from models.ts — manual sync needed if fields change
- NoCertsPanel `_platform` unused param — intentional, no platform-specific content in that panel

---

## v1.1 Polish & Build (Shipped: 2026-03-20)

**Phases completed:** 7 phases, 13 plans

**Key accomplishments:**

- Terminal layout baseline — CSS flex chain fix, enlarged toolbar buttons (38x38px), 42px tab bar
- Per-tab status bar replacing floating web-serving overlay with 3-state strip (inactive/off/on)
- Tabbed settings modal with inline Save Paths and single Close footer
- Per-tab font size adjustment via SHIFT+=/- shortcuts with per-tab state isolation
- New-session modal with agent picker, native OS folder browser, and last-folder memory
- Tab rename (double-click + right-click context menu) with name propagation to web dashboard
- Web dashboard visual redesign: card layout, status dots, CLI badges, TokyoNight palette
- Cross-platform build script (`build.sh`) with macOS signing/notarization pipeline

**Stats:**

- 88 files changed, ~9,956 LOC (6,541 Go + 2,622 TS/TSX + 793 CSS)
- Timeline: 4 days (2026-03-17 → 2026-03-20)
- Git range: `feat(07-01)` → `feat(13-02)`

**Known Gaps (accepted as tech debt):**

- TERM-01 partial: Initial-paint terminal fill timing race (xterm.js FitAddon) — fills after resize, tabled by user
- BUILD-01..04: Missing from 13-01-SUMMARY.md requirements_completed frontmatter (code verified working)
- DetectedCLI.DisplayName missing from TypeScript Wails stub (works at runtime)
- build_linux() uses go build instead of wails build inside Docker (binary produced, may lack wails metadata)
- macOS notarization untested end-to-end (codesign verified, notarytool needs real app-specific password)

---

## v1.0 MVP (Shipped: 2026-03-19)

**Phases completed:** 6 phases, 19 plans, 6 tasks

**Key accomplishments:**

- Cross-platform PTY process management with CLI auto-detection (Claude Code, Codex, Gemini CLI, OpenCode)
- WebSocket fan-out relay with binary framing protocol and bounded scrollback replay
- Wails desktop UI with tabbed xterm.js terminals, session naming, system tray persistence
- Embedded HTTPS web server with self-signed TLS (CA+leaf), bcrypt auth, per-session token links, VPN/Tailscale binding
- QR code generation for web-served sessions (desktop modal + web dashboard) with live status badges
- GitHub Actions CI matrix for macOS (signed/notarized), Linux (WebKitGTK 4.0/4.1), and Windows (NSIS)

**Stats:**

- 107 commits, 149 files, ~8,100 LOC (6,400 Go + 1,700 JS/TS/CSS)
- Timeline: 3 days (2026-03-17 → 2026-03-19)
- Git range: `feat(01-01)` → `feat(06-02)`

**Known Gaps:**

- STAT-02 partial: Status heuristics only for Claude CLI; codex/gemini/opencode always show "running"
- TERM-05 partial: @xterm/addon-clipboard installed but unused; native browser copy/paste works
- ParseWin32Input exported but not wired into relay/webserver input path (Windows only)
- 9 items require human verification (TLS flows, visual items, CI execution)

---
