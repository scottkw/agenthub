# AgentHub

## What This Is

A cross-platform desktop app (macOS, Linux, Windows) for running AI coding CLIs — Claude Code, OpenCode, Codex, Gemini CLI, and others — in tabbed terminal sessions powered by xterm.js. Built with Go/Wails for the backend and React for the frontend. Runs as a tray-resident app with a system tray icon, dynamic session menu, and daemon management panel — no dock/taskbar icon. Every session can be served over the web via Tailscale with browser-trusted Let's Encrypt TLS, accessible from any tailnet device via URL or QR code — no passwords, no tokens, no certificate setup. Remote sessions across tailnet peers are discoverable via both GUI (Remote Sessions panel with auto-refresh) and CLI (`agenthub list` with HOST column, `agenthub attach hostname:id` via WSS relay). Standard macOS app menus with Cmd+C/V clipboard in terminal tabs. Auto-update checker polls GitHub releases and shows notification banner with one-click download. Health checks detect Tailscale state and guide users through setup with platform-specific install commands, macOS auto-install via Homebrew, and post-install HTTPS cert configuration guide. Live status indicators show whether each CLI is running, waiting, or errored. Includes branded app icons, a splash screen with the title logo, a polished UI with tabbed settings, per-tab font sizing, new-session modal with agent picker, folder browser, and per-agent argument memory, tab renaming with web dashboard propagation, and a cross-platform build script with macOS signing support. CLI and GUI both support passing extra arguments to agents (`--` separator in CLI, text field in GUI). Distributed via GitHub releases (DMG, EXE+NSIS, tar.gz+deb), Homebrew cask, and WinGet.

## Core Value

One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## Requirements

### Validated

- ✓ Tabbed terminal UI with xterm.js for running multiple AI coding CLI sessions simultaneously — v1.0
- ✓ Support for major AI coding CLIs: Claude Code, OpenCode, Codex, Gemini CLI — v1.0
- ✓ Go-native PTY mode: built-in session persistence with no external dependencies — v1.0
- ✓ Web serving of terminal sessions via hosted xterm.js — v1.0
- ✓ Per-session toggle for web serving (on/off) — v1.0
- ✓ Self-signed TLS certificates (CA + leaf pattern) for all web connections — v1.0
- ✓ Web dashboard to browse all served sessions — v1.0 (password auth removed in v1.2 Phase 16)
- ✓ Per-session QR/URL access — v1.0 (shareable tokens removed in v1.2 Phase 16)
- ✓ QR code generation for all web-served sessions — v1.0
- ✓ VPN interface binding — Tailscale-only (generic VPN picker removed in v1.2 Phase 17) — v1.0
- ✓ Multi-platform: macOS, Linux, Windows — v1.0 (CI matrix, signing/notarization)
- ✓ Wails desktop shell with React frontend — v1.0
- ✓ Go backend serving both the desktop app and web interface on the same process — v1.0
- ✓ Live status indicators (running/waiting/idle/errored) per session — v1.0
- ✓ Build script (`build.sh`) for per-platform and all-platform compilation with macOS signing — v1.1
- ✓ Tabbed settings modal with inline Save Paths and single Close footer — v1.1
- ✓ Web dashboard visual redesign with card layout, status dots, CLI badges — v1.1
- ✓ Per-tab status bar replacing header overlay for web status/URL/controls — v1.1
- ✓ Tab renaming (double-click + right-click context menu) with web dashboard propagation — v1.1
- ✓ Larger toolbar buttons (38x38px, comfortable to click) — v1.1
- ✓ New-session modal with agent picker, native folder browser, and last-folder memory — v1.1
- ✓ Per-tab SHIFT+/SHIFT- font size adjustment — v1.1
- ✓ Terminal fill: CSS flex chain fixed, fills after resize — v1.1 (initial-paint timing gap fully resolved in v1.6 Phase 35)
- ✓ Tailscale health checks (installed, connected, certs enabled) with background polling — v1.2
- ✓ Health modal with platform-specific instructions (macOS/Linux/Windows) and Check Again auto-dismiss — v1.2
- ✓ Let's Encrypt TLS via Tailscale daemon (`GetCertificate` hook, FQDN-based URLs) — v1.2
- ✓ Certificate Transparency disclosure before first cert provisioning — v1.2
- ✓ Web server binds exclusively to Tailscale interface IP — v1.2
- ✓ Password auth, per-session tokens, and auth middleware removed — v1.2
- ✓ Web dashboard accessible without authentication to tailnet members — v1.2
- ✓ Self-signed certificate infrastructure removed (CA+leaf generation, tls.go) — v1.2
- ✓ Generic VPN interface binding code removed (Tailscale-only) — v1.2
- ✓ Dead code cleanup: network.go, GetNetworkInterfaces, and frontend binding stubs removed — v1.2
- ✓ Tailscale status indicator in Settings panel — v1.2
- ✓ SessionEngine extracted from App into `internal/daemon` package with HTTP/JSON protocol over Unix socket — v1.3
- ✓ Process separation: sessions persist across GUI close/reopen; RunDaemon/EnsureDaemon lifecycle; App reduced to thin DaemonClient shell — v1.3
- ✓ Standalone CLI binary with 13 commands (new, list, kill, rename, attach, web start/stop/status, serve/unserve, health, qr, settings) and daemon auto-start — v1.3
- ✓ Interactive terminal attach (`agenthub attach <id>`): full PTY proxy with raw I/O, detach key (Ctrl-\), resize propagation, Ctrl-C passthrough, scrollback replay, signal-safe terminal restore — v1.3
- ✓ Service manager integration: `agenthub daemon install/uninstall/start/stop` via kardianos/service for launchd/systemd/Windows SCM — v1.3
- ✓ Machine-readable CLI output: `--json` flag on list, web status, health, daemon status commands — v1.3
- ✓ Settings inspection: `agenthub settings` read-only command — v1.3
- ✓ Windows named pipe fix: CleanupStaleSocket uses winio.DialPipe for `\\.\pipe\...` paths — v1.3
- ✓ Graceful GUI startup failure: error banner with retry instead of panic on daemon failure — v1.3
- ✓ Unified entrypoint: root main.go dispatches GUI (no args), CLI (subcommand), and daemon modes — v1.4 Phase 27
- ✓ Dead `cmd/agenthub-cli/` package removed — 8 files (1,559 lines) deleted, all references scrubbed — v1.4 Phase 28
- ✓ Build system verified: portable build-script tests (35/35), CI race detector on all platforms, build-script CI step on ubuntu-latest — v1.4 Phase 29
- ✓ Backend args wiring: all 5 daemon IPC layers (types, engine, API, client, Wails binding) accept and forward `args []string` to PTY — v1.5 Phase 30
- ✓ CLI arg passthrough: `splitDashDash` helper + `cmdNew` updated to forward extra args via `--` separator to `CreateSession` — v1.5 Phase 31
- ✓ Daemon startup performance: immediate session status polling (500ms vs 2s) and PATH augmentation for service-mode agents (nvm, Volta, Homebrew) — v1.5 Phase 32
- ✓ Terminal fill fix v2: bounded rAF retry loop polling proposeDimensions() — fixes initial-load fill for all 4 CLIs — v1.6 Phase 35
- ✓ CLI `--` passthrough: `agenthub new <agent> <path> -- <extra-args>` forwards trailing tokens to agent PTY process — v1.5 Phase 31
- ✓ GUI args field: text field in new-session modal with per-agent localStorage persistence and clear button — v1.5 Phase 33
- ✓ Per-agent argument memory: last-used args pre-filled per agent, clearable — v1.5 Phase 33
- ✓ Platform icon sets: ICNS (macOS), ICO (Windows), multi-size PNGs (Linux/Wails) from branded logomark — v1.7 Phase 36
- ✓ Splash screen: WelcomeTab with title logo, tagline, version, install instructions; StartHidden + OnDomReady no-flash pattern — v1.7 Phase 37
- ✓ Session metadata includes machine hostname via os.Hostname() for remote identification — v1.7 Phase 38
- ✓ Web terminal status bar showing session name, agent type, hostname, and REST-polled connection state — v1.7 Phase 39
- ✓ CLI attach connection banner and detach message on stderr with session name, agent, hostname — v1.7 Phase 39
- ✓ Daemon Management Panel: in-GUI tab with session list, status dots, kill, and web-serve toggles — v1.7 Phase 40
- ✓ System tray icon with monochrome template, dynamic NSMenuDelegate menu, tooltip, and session list — v1.7 Phase 41
- ✓ Window hide-on-close: closing GUI hides window, daemon and tray remain active — v1.7 Phase 41
- ✓ LSUIElement: app hidden from Dock and Cmd+Tab, lives in menu bar only — v1.7 Phase 41
- ✓ Quit from tray: stops daemon, removes tray icon, fully exits application — v1.7 Phase 41
- ✓ Monochrome tray icon template adapting to light/dark macOS menu bar — v1.7 Phase 41
- ✓ Tray icon error state: shows disconnected icon and tooltip when daemon unreachable at startup — v1.7 Phase 42
- ✓ GUI hostname forwarding: Hostname field in App.go SessionInfo, displayed in DaemonManagerPanel — v1.7 Phase 43

- ✓ GitHub repository (scottkw/agenthub) with full Gitea history and all v1.0–v1.7 tags — v1.8 Phase 44
- ✓ Go module path rewritten to github.com/scottkw/agenthub across all imports — v1.8 Phase 44
- ✓ release-please auto-versioning with conventional commits and CHANGELOG generation — v1.8 Phase 45
- ✓ macOS signing removed from PR builds, retained only in release pipeline — v1.8 Phase 45
- ✓ Multi-platform release pipeline (macOS signed DMG, Windows EXE+NSIS, Linux tar.gz+deb, checksums.txt) — v1.8 Phase 46
- ✓ Homebrew cask tap (scottkw/homebrew-agenthub) with auto-update on release — v1.8 Phase 47
- ✓ Packaging templates (Homebrew cask template, WinGet 3-file manifests at schema 1.12.0) — v1.8 Phase 47
- ✓ distribute.yml auto-updates Homebrew tap + submits WinGet manifests on release — v1.8 Phases 47-48
- ✓ WinGet distribution infrastructure (WINGET_TOKEN, winget-pkgs fork, populate-manifests.sh) — v1.8 Phase 48

- ✓ Standard app menus (File, Edit, Window, Help) with keyboard shortcuts — v1.9 Phase 49
- ✓ Cmd+C/V clipboard operations in terminal tabs — v1.9 Phase 49
- ✓ Welcome screen version from release build (no hardcoded VERSION) — v1.9 Phase 49
- ✓ Welcome logo rounded corners — v1.9 Phase 49
- ✓ Tailscale peer discovery: `internal/tailnet` package with injectable deps, concurrent probe pool (cap 5), daemon `GET /tailnet/peers` with 30s cache — v1.9 Phase 50
- ✓ Auto-update checker: GitHub releases polling on startup + hourly, notification banner, Help menu trigger, one-click download — v1.9 Phase 51
- ✓ Remote Sessions GUI panel: tailnet peer grouping, loading states, 30s auto-refresh, one-click browser open — v1.9 Phase 52
- ✓ CLI remote session list: `agenthub list` shows local + remote sessions grouped by HOST column — v1.9 Phase 53
- ✓ CLI remote session attach: `agenthub attach hostname:session-id` connects via WSS relay over Tailscale HTTPS — v1.9 Phase 53
- ✓ Tailscale onboarding: platform-specific install commands with copy-to-clipboard, macOS auto-install via Homebrew, post-install HTTPS cert guide — v1.9 Phase 54

- ✓ Collapsible left sidebar with Heroicons SVG icons replacing top toolbar action buttons — v1.10 Phase 55
- ✓ Sidebar items: Home, Remote, Sessions, New Tab at top; Settings pinned to bottom — v1.10 Phase 55
- ✓ All sidebar icons from @heroicons/react (MIT-licensed open-source SVG icon set) — v1.10 Phase 55
- ✓ Sidebar toggle (Bars3Icon) and Sessions (ServerStackIcon) replacing Unicode icons — v1.10 Phase 55
- ✓ Sidebar collapsed/expanded state persisted in localStorage — v1.10 Phase 55
- ✓ Tab bar action buttons removed, retains session tabs only — v1.10 Phase 55
- ✓ Navigation wiring: all sidebar items (Home, Remote, Sessions, New Tab, Settings) open their corresponding tabs/panels — v1.10 Phase 56
- ✓ Tab bar cleanup: dead tab-bar CSS and obsolete tests removed after sidebar integration — v1.10 Phase 56

### Active

(No active requirements — planning next milestone)

## Current State

v1.10 shipped (2026-04-08). 11 milestones shipped (v1.0–v1.10), 56 phases completed, 103 plans total. Codebase: ~31K Go + ~10K TS/TSX. App runs as a tray-resident daemon with branded icons, splash screen, standard macOS menus, remote session discovery (GUI panel + CLI), auto-update notifications, guided Tailscale onboarding with auto-install, and collapsible left sidebar with Heroicons SVG icons — all navigation wired through sidebar.

### Out of Scope

- Mobile app — desktop and web access only; PWA via web serving covers mobile needs
- AI coding CLI installation — app checks for CLIs but doesn't install them
- Tailscale/VPN full management — app assists with installation and initial setup but doesn't manage ongoing Tailscale configuration
- End-to-end encryption beyond TLS — Tailscale's Let's Encrypt certs are sufficient
- User account system with registration — Tailscale network membership is the access control
- Cloud hosting or SaaS deployment — this is a local-first desktop app
- Plugin system for adding new CLIs — initial set is hardcoded, extensibility is future scope
- Session output search / replay — tools like agent-sessions serve this niche
- Split panes / tiling within a tab — each AI session gets its own tab
- Configurable session backend (tmux vs Go-native) — deferred to future milestone
- Real tmux mode with `tmux attach` — deferred to future milestone
- Per-session token expiry and revocation — removed: tokens deleted in v1.2
- Non-Tailscale VPN support — removed in v1.2; Tailscale-only networking
- Tab color coding per CLI type — deferred to future milestone
- Status heuristic patterns for non-Claude CLIs — deferred to future milestone
- Font/theme customization beyond size — per-tab font size covers the immediate need

## Context

Shipped v1.9 with ~31K Go + ~10K TS/TSX.
Tech stack: Go/Wails v2, React, xterm.js, nhooyr/websocket, go-pty, skip2/go-qrcode, tailscale.com/client/local, kardianos/service, Masterminds/semver, creativeprojects/go-selfupdate.
Architecture: Single `agenthub` binary — no args launches GUI (Wails), subcommands run CLI, `daemon` manages service. Background daemon (`internal/daemon`) owns all session state; GUI and CLI are both DaemonClient consumers over Unix socket (named pipe on Windows). Root package contains all CLI functions (unified in v1.4). Args thread through all 5 IPC layers to PTY. Frontend estimates terminal dimensions and passes cols/rows to backend at session creation. System tray uses native macOS cgo NSStatusBar (no third-party systray library) with NSMenuDelegate for dynamic menus. Remote session discovery via `internal/tailnet` package probes peers over Tailscale HTTPS.
Go test suite: 200+ tests race-clean across 8 packages (added internal/tailnet, internal/updater).
Frontend test suite: vitest source-inspection tests covering args field, terminal panel, modal, splash screen, daemon manager, remote sessions panel, health modal, and web status bar components.
Networking: Tailscale-only — Let's Encrypt certs via daemon, FQDN-based URLs, no auth layer.
Distribution: GitHub releases (DMG, EXE+NSIS, tar.gz+deb, checksums), Homebrew cask, WinGet. release-please auto-versioning.
Build script: `build.sh` compiles for macOS/Linux/Windows with optional macOS signing/notarization. CI runs race detector on all 4 platform legs + build-script tests on ubuntu-latest.
Known tech debt: WelcomeTab does not auto-dismiss (user-approved pivot to persistent tab); Linux/Windows tray stubs needed; dead installError state in App.tsx (~3 lines); CheckForUpdates TS binding exported but unused by frontend (by design — native menu callback).

## Constraints

- **Tech stack**: Go backend (Wails), React frontend, xterm.js for terminals
- **Single binary**: Wails compiles to a single distributable binary per platform
- **No cloud dependency**: Everything runs locally — no external services required
- **Cross-platform**: Must work on macOS, Linux, and Windows from day one

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go/Wails over Electron | Smaller binary, better performance, native Go ecosystem for PTY/tmux | ✓ Good — single binary ~15MB, fast startup |
| xterm.js for terminal rendering | Industry standard, well-maintained, used by VS Code terminal | ✓ Good — full ANSI/Unicode support, scrollback works |
| Go-native PTY only for v1 (tmux deferred to v2) | Reduce scope; Go-native covers all v1 use cases | ✓ Good — simplified architecture, no tmux dep |
| Same Go process serves desktop + web | Simpler architecture, single port management, shared session state | ✓ Good — session sharing works seamlessly |
| Self-signed TLS → Tailscale Let's Encrypt | Phase 15 switched to Tailscale certs; self-signed CA removed | ✓ Good — no CA install needed, browser-trusted by default |
| Tailscale network = access control | Phase 16 removed password + tokens; only tailnet members can reach the server | ✓ Good — simpler, no auth UI/middleware to maintain |
| go-pty (aymanbagabas) over creack/pty | Windows ConPTY support required from day one | ✓ Good — cross-platform PTY with single API |
| Binary framing protocol for WS relay | Distinguishes output/resize/input frames; enables scrollback replay | ✓ Good — clean separation of message types |
| Native macOS cgo NSStatusBar for tray | fyne.io/systray conflicts with Wails AppDelegate (duplicate symbol) | ⚠️ Revisit — platform-specific code, Linux/Windows stubs needed |
| In-process Unix socket before process separation | Phase 19 validates module boundary and protocol without fork complexity | ✓ Good — full test coverage of API contract; Phase 20 changed only socket path |
| CreateSession calls engine directly (not client) | onStatus callback wraps runtime.EventsEmit — callbacks can't serialize over HTTP | ✓ Good — clean exception, documented in code |
| Function injection for service control (`serviceControlFunc`) | Enables daemon-free unit testing without mocks or interfaces | ✓ Good — fast, deterministic tests |
| kardianos/service for cross-platform service management | Abstracts launchd/systemd/Windows SCM behind single Go API | ✓ Good — single codebase, platform-specific behavior via library |
| pollSessionStatus goroutine replaces onStatus callback | Callbacks can't serialize over HTTP in out-of-process daemon | ✓ Good — correct pattern for process separation |
| flag.NewFlagSet per CLI command (not package globals) | Avoids state pollution between test runs | ✓ Good — clean test isolation |
| Graceful startup with daemonErr + daemon:error event | Dual notification: event for real-time, field for polling | ✓ Good — no crash on daemon failure, retry works |
| ResizeObserver + requestAnimationFrame for fit() | Handles all layout changes, not just window resize | ✓ Good — fixed terminal height issues |
| ?raw source-inspection tests for xterm.js components | jsdom lacks Canvas/WebGL; runtime mocking xterm is fragile | ✓ Good — stable tests that verify code structure without DOM |
| JSX conditionals over CSS display toggle | Consistent pattern across StatusBar, SettingsPanel tabs | ✓ Good — cleaner React patterns, easier to test |
| build.sh with Docker for Linux cross-compile | No native Linux WebKitGTK headers on macOS | ✓ Good — portable Linux builds from any OS |
| ditto (not zip) for notarization archive | Preserves macOS extended attributes required by notarytool | ✓ Good — correct signing pipeline |
| `local.Client{}` zero-value for Tailscale daemon | Queries existing tailscaled via Unix socket; no tsnet, no embedded daemon | ✓ Good — minimal dependency, no second Tailscale node |
| In-process Unix socket before process separation | Phase 19 validates the module boundary and protocol without fork complexity | ✓ Good — full test coverage of API contract; Phase 20 changes only socket path |
| CreateSession calls engine directly (not client) | onStatus callback wraps runtime.EventsEmit — callbacks can't serialize over HTTP | ✓ Good — clean exception, documented in code |
| Function injection for health checks (`statusFunc`) | Enables daemon-free unit testing without mocks or interfaces | ✓ Good — fast, deterministic tests |
| `GetCertificate` hook (not cached CertPair) | Dynamic cert provisioning; certs always fresh from daemon | ✓ Good — no stale cert bugs, no disk writes |
| FQDN from `CertDomains()[0]` (not hardcoded) | Machine name auto-derived from Tailscale daemon | ✓ Good — zero configuration for users |
| CT disclosure via sentinel file | One-time acknowledgment persisted as `ct_disclosed` file | ✓ Good — simple, no database needed |
| Tailscale health gates web server startup | Server refuses to start without healthy Tailscale state | ✓ Good — clear error path, no partial failures |
| Safety dependency chain (health→TLS→auth removal→cleanup) | Each phase's deletion is safe only after the prior phase confirms the replacement works | ✓ Good — zero regressions across 5 phases |
| `args []string` threaded between workDir and onStatus params | Clean positional parameter ordering; `json:"args,omitempty"` for backward compat | ✓ Good — no wire format regression for nil callers |
| `splitDashDash` returns nil (not empty slice) when no `--` | Go idiom: nil means "not provided", empty means "provided but empty" | ✓ Good — clean distinction, no injection risk |
| Poll-first, sleep-after for `pollSessionStatus` | Eliminates artificial 2s blank period; 500ms interval is responsive without overhead | ✓ Good — immediate status feedback |
| Runtime PATH augmentation at daemon startup | Service-mode daemon can't source shell init files; prepend known install paths | ✓ Good — nvm/Volta/Homebrew agents found without config |
| Double-rAF for initial terminal fit | Wails WebView needs two animation frames for CSS layout commit before FitAddon measurement | ⚠️ Revisit — insufficient for 3/4 CLIs; replaced by rAF retry loop in v1.6 |
| Bounded rAF retry loop polling proposeDimensions() | Double-rAF fires once at ~32ms, misses slow CLI startups; retry loop polls until CharSizeService reports non-zero dimensions | ✓ Good — fixes all 4 CLIs, bounded at 20 attempts (~333ms) |
| Frontend cols/rows estimation at session creation | `Math.floor(clientWidth/charWidth)` estimates dimensions before xterm renders | ✓ Good — PTY spawns at correct size, no 80x24 default |
| Native macOS cgo NSStatusBar for tray (no fyne.io/systray) | fyne.io/systray conflicts with Wails AppDelegate — duplicate symbol linker error | ✓ Good — full control, no library conflicts; Linux/Windows stubs needed |
| NSMenuDelegate menuWillOpen: for dynamic tray menu | Always fresh at open time, no push-update polling needed for session list | ✓ Good — menu always reflects current state |
| StartHidden + OnDomReady for splash screen | Window hidden until WebView DOM ready, then runtime.WindowShow — no white flash | ✓ Good — canonical Wails no-flash pattern |
| Logo in frontend/public/ not src/assets/ | Stable /agenthub-title-logo.png URL without Vite content-hashing in dev and prod | ✓ Good — reliable for both static HTML splash and React |
| REST polling (3s) for web terminal status bar | Simpler than new WebSocket frame type; status bar is flex sibling to avoid FitAddon regression | ✓ Good — no relay protocol changes needed |
| WelcomeTab as persistent closeable tab (user-approved pivot) | Original spec was auto-dismiss on daemon connect; user preferred persistent welcome | ⚠️ Tech debt — auto-dismiss and 3s fallback not implemented |
| ObjC @implementation in separate .m file (not cgo blocks) | Go cgo blocks cause duplicate symbol linker errors during `go test` | ✓ Good — clean compilation for both build and test |
| Split nil-client guard in refreshTrayState | Single compound guard skipped tray update entirely on startup failure | ✓ Good — tray shows error icon when daemon unreachable |
| Post-build cp of pre-built ICNS into bundle | wails build produces 3-size ICNS (361KB); pre-built 10-entry iconfile.icns (590KB) needed for Retina | ✓ Good — full Retina coverage in production builds |
| Package-level appCtx for menu callback context | Avoids closure complexity in Wails menu API | ✓ Good — clean callback wiring |
| Injectable statusFunc for tailnet package | Mirrors webserver/tailscale.go pattern; enables daemon-free unit testing | ✓ Good — fast, deterministic tests |
| Full Mutex for tailnetCache.getOrRefresh | Prevents thundering herd on 30s cache expiry | ✓ Good — single concurrent refresh |
| Masterminds/semver for version comparison | Transitive dep of go-selfupdate; cleaner than constructing Release struct | ✓ Good — direct, minimal code |
| 5-second initial delay for update poller | Avoids startup race with frontend event subscription | ✓ Good — reliable event delivery |
| onOpen callback prop for RemoteSessionsPanel | Not direct BrowserOpenURL import; keeps component testable | ✓ Good — clean test isolation |
| loading+peers.length>0 shows data not spinner | Prevents 30s refresh flicker | ✓ Good — smooth UX |
| listOutput struct for CLI --json | Breaking change from flat array; enables local/remote grouping | ⚠️ Revisit — document migration for JSON consumers |
| goruntime alias for stdlib runtime | Avoids collision with wails/v2/pkg/runtime already imported as runtime | ✓ Good — clean namespace |
| cmd.Stderr = cmd.Stdout for brew streaming | Merges stderr into stdout pipe for single-goroutine output streaming | ✓ Good — simple, reliable |
| installProgress state in App.tsx not HealthModal | EventsOn subscriber pattern; HealthModal is pure display component | ✓ Good — clean separation |

---
## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-08 after v1.10 milestone*
