# AgentHub

## What This Is

A cross-platform desktop app (macOS, Linux, Windows) for running AI coding CLIs — Claude Code, OpenCode, Codex, Gemini CLI, and others — in tabbed terminal sessions powered by xterm.js. Built with Go/Wails for the backend and React for the frontend. Runs as a tray-resident app with a system tray icon, dynamic session menu, and daemon management panel — no dock/taskbar icon. Every session can be served over the web via Tailscale with browser-trusted Let's Encrypt TLS, accessible from any tailnet device via URL or QR code — no passwords, no tokens, no certificate setup. Falls back to local network serving with self-signed TLS and generated password when Tailscale is unavailable. Multiple WebSocket clients can connect to the same session simultaneously with independent scrollback, read-only mode, and max-wins PTY resize arbitration. CLI attach displays a persistent tmux-style status bar (session name, agent, hostname, viewer count, elapsed time) using DECSTBM scroll regions, with clean teardown and `--status-top` placement option. `agenthub tui` launches a full-screen Bubble Tea v2 terminal UI with session list (status glyphs, agent type, hostname, viewer count), full session lifecycle (attach with suspend/resume, create via modal, kill with confirmation, inline rename), remote tailnet peer sessions in a unified list, ASCII QR code overlay for session web URLs, web server status footer, and `?` help overlay. Remote sessions across tailnet peers are discoverable via GUI (Remote Sessions panel with auto-refresh), CLI (`agenthub list` with HOST column, `agenthub attach hostname:id` via WSS relay), and TUI (unified list with remote peer grouping). Standard macOS app menus with Cmd+C/V clipboard in terminal tabs. Auto-update checker polls GitHub releases and shows notification banner with one-click download. Health checks detect Tailscale state and guide users through setup with platform-specific install commands, macOS auto-install via Homebrew, and post-install HTTPS cert configuration guide. Live status indicators show whether each CLI is running, waiting, or errored. Terminal sessions support 138 curated color themes (WCAG-audited from 157 xterm-theme candidates) with live apply and persistence. Includes branded app icons, a splash screen with the title logo, a polished UI with collapsible sidebar navigation, tabbed settings with Appearance tab for theme selection, per-tab font sizing, new-session modal with agent picker, folder browser, and per-agent argument memory, tab renaming with web dashboard propagation, and a cross-platform build script with macOS signing support. Settings tab provides web server URL actions — open in browser, copy to clipboard, and QR code display. CLI and GUI both support passing extra arguments to agents (`--` separator in CLI, text field in GUI). Distributed via GitHub releases (DMG, EXE+NSIS, tar.gz+deb), Homebrew cask, and WinGet.

## Core Value

One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## Last Shipped Milestone: v3.4 File Browser (Read-Only) + TUI Parity (2026-05-21)

**Closes:** GitHub Issues #62 (read-only file browser) + v3.4 slice of #64 (TUI browse+preview parity). Umbrella epic #24 stays open across v3.4 + v3.5. 5 phases (118-122, including audit-driven Phase 122 insert), 48 REQ-IDs (FS-01..14, WEB-01..05, UI-01..14, TUI-01..10, REMOTE-01..05), 176 commits across 2 days (2026-05-20 → 2026-05-21). Audit status `tech_debt` — release-eligible after 3 user-acknowledged manual UATs (Phase 120 Wails click-path, Phase 121 visual TokyoNight, Phase 122 22-step two-machine tailnet). Tagged v3.4.

## Previous Milestone: v3.3.1 Bug Sweep (2026-05-19)

**Closes:** GitHub Issues #52 (Windows named pipe IPC, third-party PR #53 from Alexandre Castro rebased), #54 (chafa OSC/DA leak — web surface), #55 (WebGLRecoveryBanner missing), #56 (iPad tap-on-link scroll), #57 (Linux shell exit detection), #58 (TestPluginConfigStream_ExpiredCap_Returns401 CI flake), #60 (desktop relay OSC/DA absorption — discovered + filed + fixed in-milestone). 9 phases (109-117), 31 REQ-IDs (IPC-01..06, WEB-01..06, UI-01..04, PTY-01..04, TEST-01..06, PAPER-01..03), 17 commits across 5 days (2026-05-15 → 2026-05-19). Tagged v3.3.1. Subsequent v3.3.2 patch tag (2026-05-20) was a dependency-bump release (nfpm/v2, tailscale.com, golang.org/x/term, bubbletea/v2) tagged off-workflow with no roadmap.

## Earlier Milestone: v3.3 Shell Sessions & Polish (2026-05-17)

**Closes:** GitHub Issues #44 (shell agent) + #45 (Settings hyperlinked index). Absorbs v3.1 Phase 91 distribution-pipeline followups and v3.2 polish/UAT carry-over.

Raw shell sessions (bash/zsh/pwsh/system-default) as a first-class agent type across all three surfaces (GUI/TUI/CLI), with cross-platform discovery (`internal/pty/shells.go` — `$SHELL` + `/etc/shells` + Windows PowerShell paths), interactive (non-login) PTY spawn with WorkDir honored, status-heuristic exclusion (only `running`/`stopped`, never `waiting`/`error`), one-time web-share confirmation banner explaining arbitrary-command-execution risk, slate-cyan (#89ddff) agent badge. After first-user-test feedback, the multi-row shell picker collapsed mid-milestone to a single "Shell" entry across all surfaces (Phases 107/108), with Settings → Paths exposing a "Shell binary" field that the daemon resolves via `shellPath` (fallback chain: stored value → `$SHELL` → `DiscoverShells` → platform default), and clean-exit `-1 → 0` PTY-wait normalization that auto-closes shell tabs on any natural exit. Settings tab gains a sticky jump-link bar and autocomplete search (Issue #45). v3.2 polish closed: `mailto:` URLs route through the existing `LinkConfirmPopover` IDN chain, find-bar Esc/close dismissal after case-sensitive toggle, sixel-only IIP decision recorded. Phase 91 distribution-pipeline followups landed as workflow edits (release.yml PAT fallback so `release.published` auto-triggers `distribute.yml`; `distribute.yml` reads `$env:RELEASE_TAG` for both event types; `wingetcreate new`/`update` branching on `WINGET_FIRST_SUBMISSION`). 7 deferred v3.2 UATs executed end-to-end on physical iPad + Tailscale — 3 pass, 2 verified-in-code, 2 with bugs filed to v3.4 backlog (`scottkw/agenthub#54` chafa OSC leak, `#55` WebGLRecoveryBanner missing, `#56` iPad tap-on-link captured by xterm-helper-textarea).

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
- ✓ Tabbed settings modal with inline Save Paths and single Close footer — v1.1 (modal replaced by sidebar tab in v1.11 Phase 58)
- ✓ Web dashboard visual redesign with card layout, status dots, CLI badges — v1.1
- ✓ Per-tab status bar replacing header overlay for web status/URL/controls — v1.1
- ✓ Tab renaming (double-click + right-click context menu) with web dashboard propagation — v1.1
- ✓ Larger toolbar buttons (38x38px, comfortable to click) — v1.1
- ✓ New-session modal with agent picker, native folder browser, and last-folder memory — v1.1
- ✓ Per-tab SHIFT+/SHIFT- font size adjustment — v1.1
- ✓ Terminal fill: CSS flex chain fixed, fills after resize — v1.1 (initial-paint timing gap fully resolved in v1.6 Phase 35)
- ✓ Tailscale health checks (4-state: Not Installed / Daemon Stopped / Not Connected / Connected) with platform-specific binary detection and background polling — v1.2, upgraded v2.1 Phase 80
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
- ✓ Claude Code detection for ~/.local/bin (Anthropic native installer path) — v1.11 Phase 57
- ✓ Sidebar "New Tab" label renamed to "New Session" — v1.11 Phase 57
- ✓ Settings converted from modal overlay to sidebar tab (singleton pattern, consistent with Home/Remote/Sessions) — v1.11 Phase 58
- ✓ Web server auto-starts on daemon launch (no manual start required) — v1.11 Phase 59
- ✓ New sessions auto-enabled for web serving when web server running — v1.11 Phases 59+61
- ✓ Local network fallback: self-signed TLS (P256+IP SAN), HTTP Basic Auth with generated password, LAN IP selection excluding Tailscale CGNAT, mode-aware server dispatch, daemon fallback wiring, password display in Settings with copy, persistent nudge banner — v1.11 Phase 60
- ✓ Sidebar icon centering: flexbox alignment fix for collapsed 48px rail — v1.12 Phase 63
- ✓ Terminal padding: 8px inset with dynamic background matching terminal theme — v1.12 Phase 64
- ✓ Terminal theming: 157 xterm-theme color schemes, Settings > Appearance tab, live apply via useEffect, localStorage persistence — v1.12 Phase 65
- ✓ Web server link UX: Open in browser, copy URL to clipboard, inline QR code display for dashboard URL in Settings tab — v1.12 Phase 66

- ✓ Cross-platform system tray: Linux D-Bus StatusNotifierItem (GNOME/KDE/XFCE) and Windows Shell_NotifyIcon with shared menu helpers, dynamic session list, status icons, hide-on-close, and Quit lifecycle — v1.13 Phase 67
- ✓ Comprehensive agent CLI discovery: daemon PATH augmentation for snap, flatpak, cargo, npm, pnpm, and Windows native installer paths via build-tagged platform files — v1.13 Phase 68
- ✓ Tailscale detection across platforms: Homebrew, system package manager, and Windows default install location — v1.13 Phase 68
- ✓ macOS Homebrew install command: single copyable `brew tap scottkw/agenthub && brew install --cask agenthub` on Welcome screen — v1.13 Phase 68
- ✓ Settings scrollable layout: single page with h3 section headers (Appearance, Web Server, Paths) replacing sub-tab navigation — v1.13 Phase 69

- ✓ Sidebar icons stay in fixed horizontal position during collapse/expand — fixed 48px icon slot via margin: 0 14px — v1.14 Phase 70
- ✓ OpenCode honors selected terminal theme — managed tui.json with OPENCODE_TUI_CONFIG env injection, SIGUSR2 broadcast to active sessions — v1.14 Phase 71
- ✓ All GUI text meets WCAG AA 4.5:1 contrast — #565f89 replaced with #9aa5ce across all surfaces — v1.14 Phase 72
- ✓ Theme usability audit: curated allowlist of 138 readable themes (from 157), localStorage fallback guard — v1.14 Phase 73

- ✓ Multiple WebSocket clients connect to same session with live output simultaneously — v2.0 Phase 74
- ✓ Independent scrollback position per connected client — v2.0 Phase 74
- ✓ Read-only attach mode via `--readonly` flag (input suppressed, output received) — v2.0 Phase 74
- ✓ Connection count per session exposed via session metadata API — v2.0 Phase 74
- ✓ Client identity name at connection (`?client=name`) stored on subscriber — v2.0 Phase 74
- ✓ PTY resize arbitration: max-wins strategy prevents dimension thrash with multiple clients — v2.0 Phase 74
- ✓ CLI attach persistent tmux-style bottom bar with session name, agent type, hostname, detach hint, elapsed time — v2.0 Phase 75
- ✓ Status bar refreshes on timer without corrupting terminal output (DECSTBM scroll region) — v2.0 Phase 75
- ✓ Status bar suppressed when stdout is not a TTY — v2.0 Phase 75
- ✓ Status bar shows viewer count when multiple clients connected — v2.0 Phase 75
- ✓ Status bar shows connection state (connected/reconnecting/latency) for remote sessions — v2.0 Phase 75
- ✓ `--status-top` flag for top placement (bottom is default) — v2.0 Phase 75
- ✓ Status bar cleans up on detach/exit — clears bar line and restores terminal state — v2.0 Phase 75
- ✓ `agenthub tui` command launches Bubble Tea v2 terminal UI — v2.0 Phase 76
- ✓ Session list panel with status indicators, agent type, hostname, and viewer count — v2.0 Phase 76
- ✓ Web server status displayed in TUI footer/status area — v2.0 Phase 76
- ✓ Help overlay (`?` key) shows all keybindings for current view — v2.0 Phase 76
- ✓ TUI attach: suspend TUI, enter raw PTY attach, resume TUI on detach — v2.0 Phase 77
- ✓ TUI create session via modal (agent picker, working directory, extra args) — v2.0 Phase 77
- ✓ TUI kill session with confirmation dialog — v2.0 Phase 77
- ✓ TUI rename session via inline edit — v2.0 Phase 77
- ✓ Remote sessions panel in TUI shows tailnet peer sessions with hostname grouping — v2.0 Phase 78
- ✓ ASCII QR code display for session web URL in TUI — v2.0 Phase 78

- ✓ Settings persistence: CLI paths and Tailscale path saved to daemon `settings.json` via Wails bindings, survive app restarts — v2.1 Phase 79
- ✓ Save confirmation: three-state button (Save/Saving.../Saved!) with 1.5s transient feedback — v2.1 Phase 79
- ✓ Native file/folder picker: browse buttons on each path field via Wails `OpenFileDialog` with parent-directory default — v2.1 Phase 79
- ✓ Tailscale 4-state health check: Not Installed / Daemon Stopped / Not Connected / Connected with platform-specific binary detection (Homebrew, Snap, Flatpak, Windows) — v2.1 Phase 80
- ✓ Diagnostics checklist in Settings for Tailscale troubleshooting — v2.1 Phase 80
- ✓ Banner vertical stacking: BannerStack container with flex-column layout, independent dismiss handlers, 200ms exit animation — v2.1 Phase 81
- ✓ Dismissed-state reset on webServerMode change — v2.1 Phase 81
- ✓ Start minimized to tray: toggle in Settings > Behavior, persisted via daemon settings.json, `domReady` gate prevents window show — v2.1 Phase 82

- ✓ Settings UI alignment: unified Paths table with single column headers for CLI and Tailscale rows; consistent 12px description text and section header typography — v3.0 Phase 83
- ✓ Session auto-close: exit detection via `hub.Done()` PTY EOF, 5-second countdown with ExitToast and ExitCountdownBanner, Keep Open cancel, auto-close toggle in Settings — v3.0 Phase 84
- ✓ Quit confirmation modal: `app:quit-requested` event from window close and tray Quit; modal shows session count with colored dots; Quit GUI Only hides window with macOS notification; Quit Everything shuts daemon — v3.0 Phase 85
- ✓ TUI visual polish: TokyoNight hex palette with lipgloss LightDark adaptive tokens; two-pane sidebar+content layout; bordered session frames with titles; per-agent colored badges for 6 CLIs; focus-aware Tab/[/] navigation — v3.0 Phase 86

- ✓ Capability-based session authorization: per-session tokens with read/write/admin capability strings; cap-token middleware on relay WSS + REST endpoints — v3.1 Phase 87
- ✓ WebSocket handshake security: byte-for-byte Origin allowlist (no subdomain wildcards), Tailscale tailnet egress accepted, cross-origin rejected at upgrade time — v3.1 Phase 88
- ✓ Vendored xterm.js + strict CSP: all xterm assets vendored under `web/vendor/xterm/`; embedded HTML routes enforce `script-src 'self'`, `connect-src 'self' wss://<host>`, `style-src 'self' 'unsafe-inline'` (D-09 xterm runtime style injection); Chromium e2e zero CSP violations — v3.1 Phase 89
- ✓ v3.1 release gate: cross-browser CSP audit + WS Origin contract tests + local-network-fallback parity — v3.1 Phase 90

- ✓ Plugin Settings Foundation: daemon `PluginSettings` source-of-truth, defaults-merge constructor, Wails RPC + `settings:plugins` runtime event, 8-toggle PluginsSection, v3.1→v3.2 `schemaVersion: 2` migration — v3.2 Phase 92 (PLUG-01..03, PUI-01)
- ✓ Vendoring discipline + web parity: generalized `vendor_drift_test.go` CI gate; vendored webgl/unicode11/clipboard for web; capability-gated `/api/plugin-config` REST + SSE; two-useEffect TerminalPanel hot-swap pattern + WebGLRecoveryBanner — v3.2 Phase 93 (PLUG-04, WGL-01..04, U11-01..02, CLIP-01..02, WEB-01..03)
- ✓ Scrollback search (Cmd-F find bar): desktop + web find bar with regex/case/word toggles, persisted defaults, 200ms slide animation, `SetSearchConfig` sub-key RPC, `seededRef` one-shot pattern, 10k-line perf gate — v3.2 Phase 94 (SRC-01..05)
- ✓ Web-Links security hardening: scheme allowlist (`https`/`http`/`mailto`), Cmd-click on macOS / Ctrl-click elsewhere, OSC 8 hover-href display, IDN + 30-entry typosquat `LinkConfirmPopover`, `BrowserOpenURL` desktop / `_blank`+`noopener,noreferrer` web split — v3.2 Phase 95 (LNK-01..06)
- ✓ Inline images + CSP audit: sixel via `@xterm/addon-image`, `'wasm-unsafe-eval'` CSP amendment 2, 16 MB per-tab `storageLimit`, `SetImageConfig` sub-key RPC, byte-fidelity multi-client relay test — v3.2 Phase 96 (IMG-01..04)
- ✓ Save Terminal As: SerializeAddon → TabBar "Save Terminal As…" with secrets warning, mockable `saveFileDialogFunc` — v3.2 Phase 97 (SER-01..03)
- ✓ OSC 9;4 progress: ProgressAddon → per-tab `.tab__progress` underline + atomic `SetTrayProgress(quartile)` debounced 200ms; default OFF in v3.2 — v3.2 Phase 98 (PRG-01..03)
- ✓ Release gate (settings UI polish): `PluginToggleBanner` one-shot toasts, inline `<details>` disclosures persisting via sub-key RPCs immediately, cross-browser Playwright e2e (Chromium + Firefox + WebKit), GitHub Actions e2e workflow — v3.2 Phase 99 (PUI-02..04)

- ✓ Raw shell sessions as first-class agent type: cross-platform discovery (`internal/pty/shells.go` — bash/zsh/pwsh/powershell + `$SHELL` + `/etc/shells` + Windows PowerShell paths), interactive non-login PTY spawn with WorkDir → `os.UserHomeDir()` fallback, status-heuristic exclusion (only `running`/`stopped`) — v3.3 Phase 100 (SHELL-04, SHELL-05, SHELL-09)
- ✓ Shell session surfaces across GUI/CLI/TUI with slate-cyan #89ddff agent badge + web-share opt-in gate + one-time `ShellWebShareBanner` confirmation explaining arbitrary-command-execution risk — v3.3 Phase 101 (SHELL-01, SHELL-02, SHELL-03, SHELL-06, SHELL-07, SHELL-08)
- ✓ Single "Shell" entry across all three surfaces (GUI/TUI/CLI) with Settings → Paths "Shell binary" field; daemon resolves spawn binary exclusively via `e.shellPath` (fallback chain: stored value → `$SHELL` → `DiscoverShells` → platform default); clean-exit `-1 → 0` PTY-wait normalization with `autoCloseRef`-gated tab auto-close on any natural exit — v3.3 Phases 107/108 (SHELL-10, SHELL-11, SHELL-12, PARITY-01..04)
- ✓ Settings hyperlinked index: sticky jump-link bar with anchor links to each section header + autocomplete `SettingsSearch` filtering by label (static `SEARCH_INDEX`, scroll-margin-top anchors, native `a href=#…` smooth scroll) — v3.3 Phase 104 (SETUI-01, SETUI-02, SETUI-03)
- ✓ Web-Links polish — Cmd/Ctrl-click `mailto:` URLs route through `LinkConfirmPopover` IDN chain (`urlRegex` extended; IDN-in-mailto routes through `getRisk → 'idn' → popover`) — v3.3 Phase 102 (POLISH-01, POLISH-02)
- ✓ Find-bar polish: Esc / close dismissal after case-sensitive toggle click, 200ms slide-out animation parity, `Sidebar.test.tsx` localStorage polyfill, sixel-only IIP decision (iTerm2 IIP / OSC 1337 out-of-scope indefinitely) — v3.3 Phase 103 (POLISH-03..06)
- ✓ Distribution pipeline followups (Phase 91 backfill): `release.yml` PAT fallback (`${{ secrets.RELEASE_PUBLISH_TOKEN || secrets.GITHUB_TOKEN }}`) so `release.published` auto-triggers `distribute.yml`; `distribute.yml` reads `$env:RELEASE_TAG` env block; `wingetcreate new`/`update` branching on `WINGET_FIRST_SUBMISSION` — v3.3 Phase 106 (DIST-01, DIST-02, DIST-03; operator-pending: PAT + variable provisioning)
- ✓ Deferred v3.2 UAT re-run executed on physical iPad + Tailscale (7 scenarios): WebGL DOM fallback, iPad rasterizer banner, 10K-line scrollback perf, Web-Links iPad chain, chafa sixel fidelity, two-client image join, iPad 5-scenario release runbook — v3.3 Phase 105 (UAT-01..07; 3 pass + 2 verified-in-code + 2 issues filed to v3.4: #54/#55/#56)

- ✓ Windows daemon named-pipe IPC fix: cherry-picked + rebased from third-party PR #53 by Alexandre Castro (attribution preserved); CleanupStaleSocket uses winio.DialPipe for `\\.\pipe\...` paths — v3.3.1 Phase 109 (IPC-01..06; closes Issue #52)
- ✓ Web-served terminal OSC/DA absorption: 5-state `InputAbsorber` machine in `internal/relay/oscabsorb.go` consumes chafa's OSC 10/11 color-query + DA1 responses before they leak into shell stdin; 26 unit subtests + 6 integration tests; 4-line `server.go` wiring with no new deps — v3.3.1 Phase 111 (WEB-01..06; closes Issue #54 web surface)
- ✓ Desktop relay OSC/DA absorption parity: `InputAbsorber` lifted into `internal/relay/handleSession` so desktop Wails matches web behavior — v3.3.1 Phase 115 (closes Issue #60 — discovered during Phase 111 desktop parity UAT, filed + fixed in-milestone)
- ✓ WebGLRecoveryBanner rendering fix: `TerminalPanel.tsx` `onContextLoss` reorder — notify React state FIRST, then `queueMicrotask`-deferred dispose with try/catch; refutes initial closure-rot hypothesis (useState setters are identity-stable per React docs) — v3.3.1 Phase 112 (UI-01..04; closes Issue #55)
- ✓ iPad terminal touch-scroll: new `frontend/src/lib/touchScrollHandler.ts` translates single-finger touch Δy into `term.scrollLines(-lines)`; multi-touch bails for iOS pinch; sub-threshold (<8px) tap path untouched so OSC 8 web-links click handler keeps firing; `touchmove` registered `passive:false`; `touch-action: pan-y` on `.terminal-session-container` preserves pinch-zoom — v3.3.1 Phase 113 (closes Issue #56)
- ✓ Linux shell auto-close: Wait4-based PTY exit detector (`internal/pty/exit_linux.go`) + no-op stubs for other platforms; closes Issue #57 — v3.3.1 Phase 110 (PTY-01..04)
- ✓ CI test stability: `TestPluginConfigStream_ExpiredCap_Returns401` deflaked via Variant A rewrite — sign with deliberately wrong 32-byte 0xFF key (vs. previous base64-padding-bit flip which was a 6.25% no-op); exercises production `ErrInvalidSignature → 401` path; 100/100 stress pass — v3.3.1 Phase 114 (TEST-01..06; closes Issue #58)
- ✓ Paper-cuts cluster: TUI `lipgloss.Place` defensive guard + `agenthub attach` screen-clear + `cleanup.go` bounded-lifetime goroutine clarity — v3.3.1 Phase 117 (PAPER-01..03)
- ✓ Pre-existing test stability: `TestOpenCodeANSICapture` data race + 3 `ShellWebShareWarned_Default`-family default-value tests fixed — v3.3.1 Phase 116

- ✓ TOCTOU-safe sandboxed filesystem API: new `internal/files/` package with `os.OpenRoot`-based `Sandbox` (Go 1.24+ kernel-level path resolution, rejects 40+ payload `FuzzSandboxPath` corpus including path traversal, `%2e%2e%2f`, `%252e%252e%252f`, U+FF0F fullwidth slash, U+2024 one-dot-leader, null bytes, Windows reserved device names, alt data streams, 8.3 short names, UNC variants); HTTP `Handler` with Range support + 5 MB read cap + 0-byte short-circuit (golang/go#54794) + darwin-filter; `SessionEngine.sessionWorkDirs` map closes the WorkDir gap; `daemonSettings.FilesRead` + `schemaVersion: 3` migration — v3.4 Phase 118 (FS-01..14)
- ✓ `files.read` capability bit + `HasPerm` whole-token comma-split (NOT `strings.Contains` — substring would false-positive `"no-files.read"`); separate `requireFilesRead` middleware (NOT added to `requireCapability` switch — avoids breaking existing relay routes); session-owner cap includes `files.read` by default, web-share viewer cap excludes by default — v3.4 Phase 118 (FS-10..13)
- ✓ Webserver mounts `/api/files/*` under `requireFilesRead` (cap-gated for Tailscale-HTTPS web-share viewers); `SetFilesHandler` plumbed at both `AutoStartWebServer` and `handleWebServerStart` construction sites — daemon socket and webserver share the same `*files.Handler` instance; viewer without `files.read` → 403 with body `"files.read"` (NOT 404); missing cap → 401; non-GET/HEAD → 405. Cross-browser Playwright (Chromium + Firefox + WebKit) reports zero CSP violations — v3.4 Phase 119 (WEB-01..05)
- ✓ React `FileBrowserTab` (desktop + web-share): single-pane file list + side-by-side preview (NOT tree+list); `BreadcrumbBar` bounded at session cwd (cannot navigate above root); `react-markdown@10.1.0` + `remark-gfm@^4` for GFM tables/task lists (NO `rehype-raw` — XSS risk); image previews via `<img src="/api/files/read?...">` (NOT base64-in-state); `/` filter (parity with TUI; NOT Cmd-F — preserves xterm.js scrollback search); over-cap/binary refusal copy + Range-capable Download; ARIA semantics; 4-state `useFilesCapability` hook surfaces `PermissionDeniedTakeover`; web-mode detection module reads URL-param `session+cap` for `/app/...` web-share mount; Playwright cross-browser e2e merge gate (45 cells across Chromium + Firefox + WebKit) — v3.4 Phase 120 (UI-01..14)
- ✓ TUI Files view: lipgloss two-pane file list + viewport preview pane (`lipgloss.JoinHorizontal`, TokyoNight palette); ALL filesystem I/O via `tea.Cmd` returning `tea.Msg` (static-grep gate `TestFiles_NoSyncFSCalls` enforces); key-dispatch priority slot 5.5; `charmbracelet/glamour` promoted from indirect to direct dep for markdown; `truncateLeft` status line (`…/utils/helper.ts`) preserves leaf-end; `/` filter + Escape dismiss; `?` help overlay with Files keybindings; Backspace at cwd root is a no-op (never traverses above session root) — v3.4 Phase 121 (TUI-01..10 local half)
- ✓ Cross-surface remote-session file browse parity (audit-driven Phase 122 insert): daemon proxy `/api/files/remote/{sid}/{op}` + in-memory `RemoteCapStore`; `ExchangeJoinCodeAtURL` Wails helper (303 redirect cap exchange); desktop GUI `RemoteJoinCodeModal` + `EnableWebSharingTakeover` (verbatim "Enable web sharing to browse this session's files") + App.tsx remote tab gate with `pathPrefix='/api/files/remote/<sid>'`; TUI `RemoteFilesClient` (HTTPS+cap; TLS 1.2+ pinned; cap redacted from errors) implements same `FilesClient` interface as `*daemon.DaemonClient` — same `handleFilesKey` + `applyFilesListMsg` pipeline drives local AND remote; v3.4 toast "File browser not available for remote sessions" removed (grep = 0); cross-surface byte-equivalence proven by 3 independent observers (daemon-proxy Go + tui.RemoteFilesClient Go + Playwright HTTPS browser) — v3.4 Phase 122 (REMOTE-01..05, TUI-08 remote half)

- ✓ TOCTOU-free write primitives on the `os.OpenRoot` sandbox: atomic temp+sync+rename (no `O_TRUNC`), rename validates source AND destination, recursive delete, mkdir, multipart upload; shell-RC denylist on every write path + 60s `FuzzSandboxWrite` merge gate (0 crashes); auth-less daemon Unix-socket write routes; folds in TD-4 + TD-5 — v3.5 Phase 123 (FSW-01..12)
- ✓ Opt-in `files.write` capability (never default-on; web-share viewers require a further explicit grant) behind `requireFilesWrite` + CSRF Origin check, `schemaVersion: 4` migration — v3.5 Phase 124 (CAP-01..10)
- ✓ Vendored CodeMirror 6 editor (zero new CSP amendments; Monaco rejected for `worker-src blob:`) replacing v3.4 plain-text preview — syntax highlighting by extension, atomic Cmd/Ctrl+S with `If-Match`/412 conflict detection, dirty-state guard, full create/mkdir/delete/rename/cross-dir-move/single+multi-upload (drag-drop, XHR progress, 409/413) suite; 51/51 cross-browser write e2e zero CSP — v3.5 Phase 125 (EDIT-01..13)
- ✓ TUI write parity via `$EDITOR` shell-out (`e` key, suspend/`tea.ClearScreen`/resume) + `d`/`r`/`m` ops; `FilesClient` interface grew 4 → 8 methods so one pipeline drives local AND remote; TUI upload formally descoped (Issue #82) — v3.5 Phase 126 (TUIW-01..07)
- ✓ Web-share write security hardening: denylist + symlink-escape / privilege-escalation / CSRF / concurrent-write audits; `SECURITY` artifact; fully automated — v3.5 Phase 127 (SEC-01..07)
- ✓ Remote tailnet peer write parity proven byte-identical by 3 independent observers (daemon-proxy Go + `tui.RemoteFilesClient` Go + Playwright HTTPS); 405 peer-outdated / 401 cap-expired mappings; Phase 122 read-regression guard — v3.5 Phase 128 (RMW-01..06)

### Active

Requirements for v3.5.1 are scoped in `.planning/REQUIREMENTS.md` (remote-browse GUI on-ramp + release-gate fixes). Active items move to Validated as phases complete.

## Current Milestone: v3.5.1 Remote Browse Completion + Release-Gate Fix

**Goal:** Close the desktop-GUI remote-browse on-ramp and clear the release-gate blocker — retiring umbrella epic #24.

**Target features:**
- **#86** — Remote Sessions panel can discover→list→pick a peer's sessions, resolving the *enumerate-then-pick vs. cap-gated no-enumeration* conflict (the umbrella-#24 closer). Approach undecided at scoping — resolved in a design pass over three options:
  - (a) tailnet-trusted metadata-only discovery endpoint (accepts enumeration exposure on the already-trusted tailnet)
  - (b) panel lists *peers* + drives a join-code/URL per session; stop dropping empty-list peers (closest to existing join-code flow)
  - (c) keep enumeration locked; reframe as "paste a join code/URL" only
- **#83** — Actionable error when the client has `accept-dns=false`: detect unresolvable MagicDNS, surface *"Enable Tailscale DNS to browse remote sessions"*; optionally probe `accept-dns` state proactively
- **#87** — Deflake `TestWrite_TwoWritersIfMatchRace` / close the `WriteFileAtomic` If-Match TOCTOU window (release-gate blocker). Correctness decision in design pass: (a) per-path lock = true single-winner guarantee, vs (b) accept last-writer-wins + rewrite test to invariants-only

**Closes:** GitHub Issues #86, #83, #87 — retires umbrella epic #24 (remote-browse GUI on-ramp becomes usable).

**Scope decisions ratified at milestone scoping (2026-06-15):**
- Scope is the remote-browse cluster (#86 + #83) plus the release-gate-blocking flaky test (#87). #82 (TUI Files upload parity) deferred to a later milestone.
- #86 architecture (a/b/c) requires a design pass before its implementation phase — not pre-decided.
- The remote read/write *data path* is already proven live; only the GUI on-ramp (discover→list→pick) is broken.

**Phase numbering:** continues from v3.5 (last phase 128) — v3.5.1 starts at Phase 129.

**Process guard (from v3.5 audit):** the v3.5 98/100 automated integration score was blind to a 4-layer GUI remote-browse breakage because tests never drove the **relay loopback** the Wails GUI uses. A relay-surface regression gate now exists (`internal/relay/server_files_test.go`, `internal/daemon/relay_remote_files_test.go`) — all v3.5.1 remote work MUST exercise it, not just the webserver/fixture surface.

**Carry-forward operator one-time tasks (still required before next release):**

1. `RELEASE_PUBLISH_TOKEN` PAT (`Contents: read/write` on `scottkw/agenthub`) — `gh secret set RELEASE_PUBLISH_TOKEN`
2. `WINGET_FIRST_SUBMISSION=true` (one-time, first submission only) — `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Unset after winget-pkgs accepts first submission.

**Carry-forward manual UATs (still pending from v3.5):**

1. Two-machine tailnet write UAT (RMW-06) — data path proven live 2026-06-15; the remaining residue is the #86 GUI on-ramp, which v3.5.1 closes
2. Live desktop/TUI visual UATs — Phase 125 editor render + Tab/Cmd-V; Phase 126 `$EDITOR` suspend-resume; Phase 124 home-dir warning banner

## Current State

v3.5 File Browser — Write Operations & Editor shipped 2026-06-15, retiring the write-side half of the file-browser epic with full cross-surface parity. 6 phases (123-128), 55 requirements satisfied (FSW-01..12, CAP-01..10, EDIT-01..13, TUIW-01..07, SEC-01..07, RMW-01..06), 27 plans, 238 commits across 2 days (2026-06-14 → 2026-06-15; ~14.9K source LOC added across 86 files, excluding `.planning/`). Audit status `tech_debt` (accepted) — 55/55 reqs satisfied at code/test level, integration 98/100 PASS, no critical blockers; release-eligible after the operator-deferred manual UATs (two-machine tailnet write UAT + live desktop/TUI visual UATs). Closes GitHub Issues #63 + #64; umbrella epic #24 retires on the committed two-machine tailnet write UAT (RMW-06). The v3.4 `os.OpenRoot` sandbox gained TOCTOU-free write primitives (atomic temp+sync+rename, no `O_TRUNC`; rename validates both source AND destination; recursive delete; mkdir; multipart upload) with a shell-RC denylist enforced on every write path and a 60s `FuzzSandboxWrite` merge gate. A new opt-in `files.write` capability (never default-on — owner enables per session, web-share viewers require a further explicit grant) gates the webserver write routes behind `requireFilesWrite` + a CSRF Origin check, with `schemaVersion: 4` migration; the daemon Unix-socket surface stays auth-less by design. The milestone centrepiece is a vendored CodeMirror 6 editor (zero new CSP amendments) replacing the v3.4 plain-text preview — syntax highlighting by extension, atomic Cmd/Ctrl+S with `If-Match`/412 conflict detection, dirty-state + unsaved guard, and the full create/mkdir/delete/rename/move/upload affordance suite. TUI write parity via `$EDITOR` shell-out + `d`/`r`/`m` keys (upload formally descoped, Issue #82). The `FilesClient` interface grew 4 → 8 methods; one pipeline drives local AND remote writes, with remote tailnet peer parity proven byte-identical by 3 independent observers (daemon-proxy Go + `tui.RemoteFilesClient` Go + Playwright HTTPS browser). Cross-surface (GUI/TUI/CLI/web) parity remains a release-blocking contract.

<details>
<summary>Prior Current State — v3.4 File Browser (Read-Only) + TUI Parity (2026-05-21)</summary>

v3.4 File Browser (Read-Only) + TUI Parity shipped 2026-05-21. 5 phases (118-122), 48 requirements satisfied (FS-01..14, WEB-01..05, UI-01..14, TUI-01..10, REMOTE-01..05), 176 commits across 2 days. Audit status `tech_debt` — release-eligible after 3 user-acknowledged manual UATs. Audit-driven mid-milestone phase insertion pattern reaffirmed (Phase 122 inserted after Phase 121 surfaced cross-surface remote-browse parity gap; same pattern as v3.3 Phases 107/108). Sandboxed filesystem API is `os.OpenRoot`-based and fuzz-proven against 40+ payload corpus; `files.read` capability bit gates the network-facing webserver surface; daemon Unix-socket surface is auth-less by design (local trust boundary, WEB-01 decision). React `FileBrowserTab` + TUI Files view share the canonical `/api/files/*` contract; cross-surface remote-session parity proven byte-identical by 3 independent network-stack observers. Write operations, editor library, TUI `$EDITOR` shell-out, and syntax highlighting all deferred to v3.5. Cross-surface (GUI/TUI/CLI/web) parity remains release-blocking contract.

</details>

## Prior State Context

v3.3.1 Bug Sweep shipped 2026-05-19; v3.3.2 dependency-bump patch tagged 2026-05-20 (off-workflow). v3.3 Shell Sessions & Polish shipped 2026-05-17. 22 milestones shipped (v1.0–v3.3), 107 phases completed across the project (Phase 91 reserved for v3.1 distribution-pipeline followups was absorbed into v3.3 as Phase 106; v3.3 spans Phases 100-108, including audit-driven mid-milestone Phases 107 + 108). 133 commits in v3.3 (~5-day shipping cycle, 57 source files / +5,784 lines excluding `.planning/`). Closes GitHub Issues #44 (shell agent) + #45 (Settings hyperlinked index). Raw shell sessions now ship as a first-class agent type across all three surfaces (GUI/TUI/CLI) — single "Shell" entry, daemon-resolved `shellPath` (Settings → Paths "Shell binary" field with fallback chain `$SHELL` → `DiscoverShells` → platform default), interactive non-login PTY spawn with WorkDir honored, status-heuristic exclusion (only `running`/`stopped`), opt-in web-share with one-time arbitrary-command-execution confirmation banner, slate-cyan (#89ddff) agent badge, clean-exit auto-close on any natural exit. Settings tab gains sticky jump-link bar + autocomplete search. Web-Links closes `mailto:` + IDN gaps. v3.1 Phase 91 distribution-pipeline backfill landed as workflow edits (release.yml PAT fallback, `distribute.yml` `$env:RELEASE_TAG` + WINGET_FIRST_SUBMISSION branching) — code-complete, operator-pending PAT + variable provisioning before next release. The 9 v3.2-deferred UATs executed end-to-end (3 pass / 2 verified-in-code / 2 with bugs filed to v3.4 backlog: `scottkw/agenthub#54` chafa OSC leak, `#55` WebGLRecoveryBanner missing, `#56` iPad tap-on-link captured by xterm-helper-textarea — all pre-existing, not v3.3 regressions). Security posture from v3.1 + v3.2 carries forward unchanged: capability-based session authorization, byte-for-byte WS Origin allowlist, vendored-only-no-CDN discipline enforced by `vendor_drift_test.go` CI gate. Distribution via GitHub releases (DMG/EXE+NSIS/tar.gz+deb), Homebrew cask, WinGet, with release-please auto-versioning. Audit-driven mid-milestone phase insertion is now a proven pattern: Phase 107 added 2026-05-13 after code-complete audit surfaced shell-UX feedback + clean-exit bug; Phase 108 added 2026-05-16 after 101-UAT Test 3 surfaced release-blocking cross-surface parity gap. Cross-surface (GUI/TUI/CLI) parity is now treated as a release-blocking contract.

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
- Custom theme editor — 138 curated xterm-theme schemes cover the need; custom editing adds complexity
- Per-tab theme overrides — global theme sufficient; per-tab adds UI complexity
- Font family selection — web font loading race adds complexity; validate demand first
- Synchronized scrollback across clients — violates universal terminal-sharing contract; independent scrollback is expected
- Mouse-driven TUI navigation — interferes with AI agent mouse usage during attach
- Split-pane tiling in TUI — already out of scope for GUI; doubly complex for TUI
- Full xterm.js rendering in TUI — requires browser/WebView canvas; raw PTY attach is the correct pattern
- TUI daemon management (install/uninstall) — CLI subcommands already handle this; rare operation
- Rich color themes for status bar — functional, not decorative; ship one scheme, add configurability on demand

## Context

Shipped v3.3 on top of v3.2 baseline. v3.3 added +5,784 lines of source (57 files, excluding `.planning/`) across daemon shell PTY (`internal/pty/shells.go`, `engine.go resolveShellSpawn`), `internal/tui/modal.go` single-Shell collapse, CLI `cmd_cli.go` single-Shell `const cli = "shell"`, frontend `NewSessionModal.tsx` + `SettingsTab.tsx` shell binary picker + `SettingsJumpBar.tsx` + `SettingsSearch.tsx` + `ShellWebShareBanner.tsx` + `LinkConfirmPopover` mailto/IDN routing, and `.github/workflows/release.yml`+`distribute.yml` PAT/RELEASE_TAG/WINGET_FIRST_SUBMISSION edits. v3.2 baseline: ~21K Go (incl. 9K tests) + ~13K TS/TSX/CSS (incl. ~5K tests, +22.7K LOC added across v3.1+v3.2). Web vendor tree at `web/vendor/xterm/` carries xterm 6.0.x core + addon-fit + 8 plugin addons (addon-search, addon-image, addon-web-links, addon-serialize, addon-progress, addon-webgl, addon-unicode11, addon-clipboard) in same-origin same-version lockstep enforced by `vendor_drift_test.go`.
Tech stack: Go/Wails v2, React, xterm.js 6 + 8 vendored addons, xterm-theme@1.1.0, nhooyr/websocket, go-pty, skip2/go-qrcode, tailscale.com/client/local, kardianos/service, Masterminds/semver, creativeprojects/go-selfupdate, charmbracelet/bubbletea/v2, charmbracelet/lipgloss/v2, charmbracelet/bubbles/v2, @xterm/addon-* (search, image, web-links, serialize, progress, webgl, unicode11, clipboard), Playwright (chromium+firefox+webkit e2e).
Architecture: Single `agenthub` binary — no args launches GUI (Wails), subcommands run CLI, `tui` launches Bubble Tea terminal UI, `daemon` manages service. Background daemon (`internal/daemon`) owns all session state; GUI, CLI, and TUI are all DaemonClient consumers over Unix socket (named pipe on Windows). Root package contains all CLI functions (unified in v1.4). `internal/relay` Hub manages per-session subscriber fan-out with metadata, read-only enforcement, and max-wins resize arbitration. `internal/statusbar` provides DECSTBM scroll-region status bar for CLI attach. `internal/tui` provides Bubble Tea v2 terminal UI with session list, modals, and QR overlay. `internal/attach` provides shared attach logic used by both CLI and TUI. System tray uses native macOS cgo NSStatusBar, Linux D-Bus StatusNotifierItem, Windows Shell_NotifyIcon — all sharing menu helpers. Remote session discovery via `internal/tailnet` package probes peers over Tailscale HTTPS. OpenCode theme integration via managed tui.json and SIGUSR2 signal broadcast through daemon HTTP API.
Go test suite: 300+ tests race-clean across 11 packages (added internal/statusbar, internal/tui, internal/attach).
Frontend test suite: vitest source-inspection tests covering args field, terminal panel, modal, splash screen, daemon manager, remote sessions panel, health modal, web status bar, sidebar, settings tab (web link UX), WCAG contrast, and theme allowlist components.
Networking: Tailscale primary — Let's Encrypt certs via daemon, FQDN-based URLs, no auth layer. Local network fallback with self-signed TLS + generated password when Tailscale unavailable.
Distribution: GitHub releases (DMG, EXE+NSIS, tar.gz+deb, checksums), Homebrew cask, WinGet. release-please auto-versioning.
Build script: `build.sh` compiles for macOS/Linux/Windows with optional macOS signing/notarization. CI runs race detector on all 4 platform legs + build-script tests on ubuntu-latest.
Known tech debt: WelcomeTab does not auto-dismiss (user-approved pivot to persistent tab); dead installError state in App.tsx (~3 lines); CheckForUpdates TS binding exported but unused by frontend (by design — native menu callback). Stale closure on remotePeers in polling useEffect (WR-01); init/retryInit duplication (WR-02) — both flagged in Phase 62 code review. Linux/Windows tray requires human UAT on live desktop environments (9 items). Phases 74 and 75 have human UAT pending for live terminal/browser testing (7 items). Remote attach not yet supported from TUI (toast displayed — future phase). MC-05 client name stored server-side but not surfaced in SessionInfo API response. v3.0: autoCloseRef not refreshed on settings toggle mid-session (MISS-01, low); TUI shows stopped sessions with green glyph instead of filtering (MISS-03, cosmetic). v3.2 polish (carry-forward): P-1 `mailto:` URLs not detected as clickable; P-2 Cyrillic IDN spoof URLs silently inert (defensive-by-accident); P-3 find bar will not dismiss after case-sensitive toggle click; P-4 find bar slide-OUT animation missing on Esc/close; P-5 iTerm2 IIP image protocol does not render (sixel works); P-6 20 Sidebar.test.tsx tests fail under Vitest 4 + jsdom 29 (localStorage global), pre-existing test-env debt. 9 v3.2 UAT scenarios deferred to v3.3 — all blocked on raw shell session type, some additionally on physical iPad / real Tailscale Tailnet.

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
| Native macOS cgo NSStatusBar for tray | fyne.io/systray conflicts with Wails AppDelegate (duplicate symbol) | ✓ Good — resolved in v1.13: Linux uses D-Bus SNI, Windows uses Shell_NotifyIcon |
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
| P256 (not P521) for self-signed TLS | Chrome rejects P521 with cryptic error; P256 universally supported | ✓ Good — works in all browsers |
| Password lifetime = daemon lifetime | Generated once in runDaemonCore via crypto/rand, not per server start | ✓ Good — stable credential across restarts |
| Nudge banner as sibling to app__content | Never inside terminal flex container — avoids displacing terminal area | ✓ Good — clean layout separation |
| Settings-as-tab follows singleton pattern | find-or-add (not push) — consistent with DaemonManagerPanel | ✓ Good — no duplicate tabs |
| Enrichment in handleListSessions (not engine.go) | Keeps WebEnabled out of SessionEngine which has no web server reference | ✓ Good — clean module boundary |
| WebEnabled on app.go SessionInfo (not daemon.SessionInfo) | Maintains Wails API surface separation from daemon types | ✓ Good — clean type hierarchy |
| Mirror backend state in frontend (no ToggleWebServing in createTab) | Daemon already auto-enables; frontend is display-only for web state | ✓ Good — single source of truth |
| `.terminal-session-container` padding (not `.xterm`) | Padding on outer container avoids xterm.js FitAddon measurement interference | ✓ Good — clean separation, no resize regression |
| xterm-theme@1.1.0 for terminal themes | 157 iTerm2-compatible themes, MIT-licensed, well-maintained | ✓ Good — broad selection, no custom theme infrastructure needed |
| localStorage for theme persistence | Same pattern as sidebar collapse state; no backend round-trip needed | ✓ Good — instant restore on app launch |
| clearTextureAtlas + refresh for WebGL theme updates | WebGL renderer caches glyph textures; must clear atlas when theme colors change | ✓ Good — reliable live theme switching |
| QR cache cleared only on server stop (not hide/show) | Avoids re-fetch on repeated toggle; server URL doesn't change while running | ✓ Good — minimal network calls |
| fs.readFileSync for CSS inspection tests (not ?raw) | vitest/jsdom does not support Vite ?raw imports; readFileSync is established project pattern | ✓ Good — stable test approach |
| D-Bus StatusNotifierItem for Linux tray (not fyne.io/systray) | fyne.io/systray conflicts with Wails AppDelegate; D-Bus SNI is the freedesktop.org standard supported by GNOME/KDE/XFCE | ✓ Good — zero library conflicts, godbus/dbus/v5 already indirect dep |
| Shell_NotifyIcon Win32 API for Windows tray | Matches the low-level native approach used for macOS; no third-party library conflicts | ✓ Good — full control, HWND_MESSAGE for invisible window |
| Shared tray_common.go for BuildMenuItems + TrayTooltip | Platform tray files (macOS/Linux/Windows) share menu construction and tooltip logic | ✓ Good — DRY, single test file covers shared logic |
| Runtime PNG-to-HICON via GDI CreateDIBSection | Reuses existing PNG assets embedded in binary; no .ico files needed | ✓ Good — fewer build artifacts |
| Build-tagged path_windows.go / path_other.go for platform paths | Follows existing codebase convention (process_windows.go, socket_windows.go) | ✓ Good — clean compilation per platform |
| h3 section headers (not tabs) for Settings layout | Single scrollable page is simpler UX than sub-tab switching for ~3 sections | ✓ Good — fewer clicks, easier scanning |
| Fixed 48px icon slot via margin: 0 14px | Same center (24px) in both collapsed and expanded sidebar states — no position shift | ✓ Good — pure CSS fix, no JS needed |
| Managed tui.json for OpenCode theming | Write system theme to opencode config dir, inject OPENCODE_TUI_CONFIG env var per session | ✓ Good — OpenCode honors theme without upstream changes |
| SIGUSR2 broadcast for live OpenCode theme switching | Frontend → Wails → daemon HTTP → engine broadcast → per-session signal delivery | ✓ Good — closes live theme switching gap for OpenCode sessions |
| `daemonConfigDir()` duplicating `configDir()` | internal/daemon cannot import main package; mirrors same logic | ✓ Good — clean package boundary |
| #9aa5ce replacing #565f89 for all GUI text | WCAG AA 4.5:1 contrast on all dark backgrounds (sidebar, tab bar, settings, welcome, modals) | ✓ Good — measurable accessibility improvement |
| ALLOWED_THEMES in separate themes.ts module | Reusable sorted allowlist; SettingsTab imports names only, not full xterm-theme objects | ✓ Good — cleaner imports, single source of truth |
| WCAG-derived readability criteria for theme audit | fg:bg >= 3.0, cursor:bg >= 2.0, at most 3 important ANSI colors below 2.5 | ✓ Good — 138/157 themes pass, both light and dark options survive |
| localStorage fallback guard for stale theme names | App.tsx validates stored theme against allowlist before use; stale names fall back to Tomorrow_Night | ✓ Good — no broken state after theme removal |
| Per-subscriber metadata on Hub (not global state) | Each WebSocket subscriber carries its own cols/rows/readonly/clientName — no shared mutable state | ✓ Good — clean concurrent access, no locks needed beyond Hub.mu |
| Max-wins PTY resize arbitration | Largest terminal dimensions win; prevents smallest-client-dominates that degrades experience | ✓ Good — stable dimensions, no resize loops |
| MsgMeta binary frame type for push metadata | Extends existing binary framing protocol (MsgOutput/MsgInput/MsgResize) with type 0x04 for viewer count push | ✓ Good — no polling needed, real-time viewer count updates |
| DECSTBM scroll region for CLI status bar | OS-level terminal scroll region keeps status bar pinned; PTY output scrolls independently above | ✓ Good — no output corruption, clean separation |
| lockedWriter serializing PTY + bar stdout writes | Mutex-wrapped io.Writer prevents interleaved output between PTY data and status bar redraws | ✓ Good — no garbled terminal output |
| Bubble Tea v2 (not v1) for TUI | Charm v2 ecosystem (bubbletea/v2, lipgloss/v2, bubbles/v2) for latest API and BackgroundColorMsg support | ✓ Good — adaptive color themes, modern API |
| tea.Exec for TUI attach (suspend/resume pattern) | TUI suspends, raw PTY runs in foreground, TUI resumes on exit — standard Bubble Tea pattern | ✓ Good — clean terminal handoff, no state corruption |
| Extract internal/attach/ shared package | CLI and TUI both need attach logic; shared package avoids duplication | ✓ Good — single implementation, consistent behavior |
| Priority-based key dispatch in TUI update | 5-level: editing > kill confirm > new session modal > QR overlay > help > main view | ✓ Good — predictable key handling, no conflicts between modal states |
| Unified local+remote session list (not separate panels) | TUI shows local and remote sessions in one scrollable list with divider rows | ✓ Good — single mental model, matches CLI `agenthub list` grouping |
| go-qrcode ToSmallString for ASCII QR in TUI | Half-block characters (▀▄█) render QR at half-height; 55×25 terminal size guard | ✓ Good — readable in standard terminal sizes |
| Remote attach deferred (toast displayed) | TUI shows "not yet supported" for remote session attach — WSS relay attach is future scope | ✓ Good — clean scope boundary, no half-implementation |
| Two-pane sidebar + tab bar TUI layout | Sidebar (Home/Sessions/Remote/Settings) + horizontal tab bar + bordered content frames; focus-aware key routing (Tab toggles, [/] cycles) | ✓ Good — matches GUI navigation model, scales to future tabs |
| TokyoNight color palette for TUI | Hex true-color values via lipgloss LightDark adaptive; per-agent badge colors for 6 known CLIs | ✓ Good — consistent with GUI theme, degrades gracefully on 256-color terminals |
| Stable session list order via CreatedAt sort | `registry.List()` sorts by `CreatedAt` (oldest first) to prevent Go map iteration randomness | ✓ Good — deterministic display order on every refresh |
| `hub.Done()` channel as PTY-exit signal | Exit watcher goroutine blocks on `<-hub.Done()` — fires only after PTY Read returns EOF, ensuring all output is drained before exit detection | ✓ Good — no truncated output |
| `app:quit-requested` event (not direct quit) | Both `beforeClose` and `onTrayQuit` emit event to frontend; modal renders in React, user chooses exit mode | ✓ Good — single event path, testable |
| UNUserNotificationCenter for macOS notifications | Quit GUI Only sends native macOS notification via cgo Objective-C wrapper; no-op stub on Linux/Windows | ✓ Good — clean platform separation |
| 5-consecutive-error circuit breaker in pollSessionStatus | Exits promptly when daemon is gone but tolerates transient connection errors | ✓ Good — resilient polling without 300s timeout on dead socket |
| `injectBorderTitle()` for TUI frame titles | String-replaces top border characters to inject styled title; avoids lipgloss Border.TopLeft override limitations | ✓ Good — clean titles without border rendering hacks |
| `sidebarTabs` mapping array (not tabID cast) | Explicit mapping from sidebar index to tabID; avoids fragile iota-dependent integer cast | ✓ Good — safe against future tabID reordering |
| Settings persistence via daemon settings.json | CLI paths, Tailscale path, and startMinimized all share `daemonSettings` struct with `saveSettingsToDisk()` | ✓ Good — single persistence layer, no per-feature storage |
| Three-state Save button (idle/saving/saved) | Transient confirmation pattern — button shows "Saved!" for 1.5s then returns to idle | ✓ Good — no toast infrastructure needed |
| 4-state Tailscale health cascade | Not Installed → Daemon Stopped → Not Connected → Connected; checked in priority order | ✓ Good — replaces binary installed/not-installed with actionable diagnostics |
| BannerStack container for vertical stacking | Banners as flex-column siblings; independent dismiss via per-banner state in App.tsx | ✓ Good — clean separation, 200ms exit animation |
| Non-optimistic toggle for startMinimized | `setStartMinimized` only called after `await SetStartMinimized` succeeds — state reverts on failure | ✓ Good — user never sees stale toggle state |
| `domReady` conditional WindowShow | `app.go` checks startMinimized preference in `domReady()` before calling `WindowShow` | ✓ Good — window never appears when start-minimized enabled |
| Capability-based session tokens (read/write/admin) | Replaces flat per-session token with capability strings; cap-token middleware enforces on relay WSS + REST | ✓ Good — finer access control, read-only viewers are first-class |
| Byte-for-byte WS Origin allowlist (no subdomain wildcards) | Tailnet egress accepted, cross-origin rejected at upgrade; closes Issue #35 attack surface | ✓ Good — strict, observable, no regex foot-guns |
| Vendored xterm.js under web/vendor/xterm/ (no CDN) | All terminal assets served same-origin; CSP `script-src 'self'` honored | ✓ Good — supply-chain controlled, CSP-strict |
| Strict CSP with `style-src 'self' 'unsafe-inline'` (D-09) | xterm injects runtime styles into the page head; `'unsafe-inline'` for styles only, never scripts | ✓ Good — minimal carve-out, audited |
| Daemon `PluginSettings` is the single source of truth for plugin state | Wails RPC + `settings:plugins` event for desktop; `/api/plugin-config` REST + SSE for web; no per-client state | ✓ Good — multi-client coherent, settings.json round-trip |
| Defaults-merge constructor for v3.1→v3.2 settings.json migration | Naïve `json.Unmarshal` yields Go zero values; constructor merges defaults over loaded fields | ✓ Good — no addons-disabled-on-upgrade footgun |
| `schemaVersion: 2` written on first v3.2 load | Migration test asserts per-field for all 8 plugin booleans + 3 sub-configs | ✓ Good — explicit forward-compatibility contract |
| Generalized `vendor_drift_test.go` for every `@xterm/addon-*` | CI fails if `package.json` and `web/vendor/xterm/VERSION` disagree on any addon version | ✓ Good — load-bearing supply-chain gate, not just addon-fit |
| Two-useEffect TerminalPanel pattern for plugin hot-swap | One useEffect for live-swappable addons (webgl/clipboard/web-links/search/image), one mount-only for buffer-interpretation plugins (unicode11) | ✓ Good — clean separation between hot-swap and next-session-only plugins |
| `seededRef` one-shot useEffect for async-loaded UI seed | useRef(false) + early-return on seeded/null/mid-open; reusable for any PluginSettings-driven UI | ✓ Good — no UI disruption during open edit buffer |
| Find bar focus-conditioned Cmd-F (only when xterm has focus) | Browser find still works for non-terminal page text | ✓ Good — no shortcut hijack, respects platform expectations |
| `SetSearchConfig` / `SetWebLinksConfig` / `SetImageConfig` sub-key RPCs | Disclosure changes persist immediately without "Save Plugins" — separate from PluginSettings bulk save | ✓ Good — fine-grained, no save-button confusion |
| Web-Links scheme allowlist (https/http/mailto only) | `file://`, `javascript:`, custom protocols never made clickable by default | ✓ Good — phishing surface minimized |
| Cmd-click on macOS / Ctrl-click elsewhere for link activation | Single-click never activates a link; modifier-key default; configurable in Settings | ✓ Good — matches platform conventions, no accidental nav |
| `LinkConfirmPopover` for IDN/Punycode + typosquat | Portal-rendered ARIA dialog with risk-specific copy (osc8 / idn / typosquat); 30-entry typosquat list + Cyrillic codepoint metatest | ✓ Good — defense-in-depth, user-visible threat model |
| Wails `BrowserOpenURL` desktop / `_blank`+`noopener,noreferrer` web | Platform-aware opener; web side never current-tab-navigates | ✓ Good — no tab hijack via OSC 8 sequences |
| Plan B for OSC 8 hyperlink mismatch (deferred to v3.3) | upstream addon-web-links lacks secondary provider API; v3.2 detects mismatch but does not render warning popover | ⚠️ Revisit — partial LNK-03 coverage; full popover ships in v3.3 |
| `'wasm-unsafe-eval'` CSP amendment 2 (image addon) | addon-image's sixel decoder uses inline WASM; amendment is minimal and audited (no `worker-src`, no `blob:`) | ✓ Good — least-privilege CSP carve-out |
| `storageLimit: 16` MB (override upstream 100 MB default) | 8 tabs × 100 MB = OOM risk; 16 MB × 8 = 128 MB ceiling | ✓ Good — bounded memory, no surprise OOM |
| `PluginToggleBanner` one-shot toast for non-hot-swappable plugins | Unicode 11 + Image require new session; banner tells user explicitly after Save | ✓ Good — discoverable, no silent no-op |
| Cross-browser Playwright e2e (Chromium + Firefox + WebKit) | Zero CSP violations on all three engines; GitHub Actions e2e.yml committed | ✓ Good — true cross-browser parity gate |
| Atomic `SetTrayProgress(quartile)` (200ms debounced) | Per-tab progress aggregated to a single tray glyph at quartile granularity; idempotency guard | ✓ Good — no tray-icon thrash, predictable bucket |
| Progress addon default OFF in v3.2 (flips ON in v3.3) | P2 cuttable plugin; field validation before default-on flip | ✓ Good — conservative rollout, observable in production |
| Hand-edit `wailsjs/go/main/App.{d.ts,js}` (no `wails generate module`) | Project maintains hand-edited stubs with Call()-based convention; regen would break vite test aliasing | ✓ Good — preserves test harness, clean diff per phase |
| Single "Shell" entry across all three surfaces (GUI/TUI/CLI) | First-user-test feedback reversed Phase 101's multi-row design; 101-UAT Test 3 declared multi-row pattern on TUI/CLI a release-blocking parity gap | ✓ Good — Phases 107/108 collapsed to bare `cli="shell"`; daemon `engine.go:500-530` is single resolution point via `e.shellPath` |
| `shellPath` plumbing mirrors `shellWebShareWarned` line-for-line | Established settings-addition pattern: engine field + Get/Set methods + GET/PATCH HTTP routes + DaemonClient methods + Wails wrappers + TS bindings | ✓ Good — load-bearing across all settings additions; future settings should mirror it |
| `resolveDefaultShellPath` fallback chain resolved at `GetShellPath()`/`CreateSession` time | Preserves "clear field to use system default" semantics — empty stored value means "use platform default", not uninitialized state | ✓ Good — clean UX, no migration awkwardness |
| Auto-close on any natural exit (zero or non-zero); ExitToast for non-zero descoped | Tester decision during SHELL-12 runtime UAT: "I don't need to know error state anyway" — tab-close gesture is sufficient for shell sessions | ✓ Good for v3.3 — revisit in v3.4 only if ExitToast for non-zero becomes a needed signal |
| Sixel-only support; iTerm2 IIP (OSC 1337) out-of-scope indefinitely (POLISH-05) | Implementation effort vs user demand — sixel already handles the inline-image use case; IIP rendering is substantial | ✓ Good — decision recorded; v3.4+ revisit only on field demand |
| Audit-driven mid-milestone phase insertion (Phases 107/108) | Code-complete audits and UAT findings surfaced scope flips + release-blocking gaps that were not visible in the original roadmap | ✓ Good — accepts that requirements firmness improves as code lands; pattern is now standard |
| Cross-surface (GUI/TUI/CLI) parity is a release-blocking contract | 101-UAT Test 3 surfaced TUI/CLI still exposing multi-shell rows after Phase 107 GUI collapse — declared unacceptable v3.3 release state | ✓ Good — Phase 108 inserted as new release-blocking phase; future shell-affecting features must verify all three surfaces |
| `release.yml` PAT fallback (`${{ secrets.RELEASE_PUBLISH_TOKEN || secrets.GITHUB_TOKEN }}`) | Without PAT, `release.published` is muted to `distribute.yml`; fallback to `GITHUB_TOKEN` keeps the workflow functional but breaks the auto-trigger | ⚠️ Operator must provision `RELEASE_PUBLISH_TOKEN` before next release for `distribute.yml` auto-trigger |
| `distribute.yml` reads `$env:RELEASE_TAG` env block (not `github.event.release.tag_name`) | The latter is empty on `workflow_dispatch`; env-block approach works for both `release.published` and `workflow_dispatch` event paths | ✓ Good — single code path, both event types |
| `WINGET_FIRST_SUBMISSION` variable branches `wingetcreate new`/`update` | First submission to microsoft/winget-pkgs requires `new`; steady state is `update`. Variable-driven so operator sets once and unsets after acceptance | ⚠️ Operator must set `WINGET_FIRST_SUBMISSION=true` for first submission only, then unset |
| `SettingsJumpBar` + `SettingsSearch` static `SEARCH_INDEX` (no DOM scraping via `data-setting-label`) | Decoupled from render-time presence of conditional toggles; simpler maintenance | ✓ Good — clean separation; deep plugin sub-options excluded per spec (zero coupling to PluginsSection internals) |
| Native `a href=#…` anchors with `scroll-margin-top` (no JS smooth-scroll handlers) | Browser handles smooth scroll + anchor offset out of the box; no preventDefault foot-guns | ✓ Good — simpler, accessible by default |
| `os.OpenRoot` (Go 1.24+) for sandbox path resolution — not `filepath.EvalSymlinks` + `os.Open` | Two-step pattern has TOCTOU race exploited by CVE-2026-27976 (Zed) + CVE-2026-43998 (vm2); `OpenRoot` is kernel-level TOCTOU-free | ✓ Good — `FuzzSandboxPath` 60s fuzz reports zero crashes against 40+ payload corpus |
| `FuzzSandboxPath` merge gate against 40+ payload corpus | Path traversal payloads from PITFALLS.md research (`../`, `%2e%2e%2f`, `%252e%252e%252f`, U+FF0F, U+2024, null bytes, `CON`/`NUL`/`COM1`, `PROGRA~1`, `:hidden`, `\\?\C:\...`, UNC) | ✓ Good — exhaustive against known attack surface |
| `HasPerm` whole-token comma-split, NOT `strings.Contains` | Substring match would false-positive `"no-files.read"` against `"files.read"` requirement; static-grep gate `TestHasPerm_NoStringsContains` enforces | ✓ Good — clean separation, no foot-gun |
| Separate `requireFilesRead` middleware — not added to `requireCapability` switch | Avoids risk of breaking existing terminal relay routes; static gate `requireCapability` grep for `files.read` returns 0 | ✓ Good — load-bearing isolation |
| Daemon Unix-socket `/api/files/*` auth-less by design (WEB-01) | Local in-process trusted IPC; only network-facing webserver surface is cap-gated | ✓ Good — clean trust boundary, parallels Wails RPC trust model |
| Session-owner cap includes `files.read` by default; viewer cap excludes by default | Read-only viewers must be granted file access explicitly | ✓ Good — least-privilege default |
| 0-byte file via `/read` returns 200 with empty body (not 416) | Explicit unit test guards against `http.ServeContent` golang/go#54794 default behavior | ✓ Good — special-case correctness locked |
| HEAD on `/read` only (not on `/list` or `/stat`) | Stat already returns full metadata; HEAD on `/read` is the only preflight needed for inline-preview vs download decision | ✓ Good — surface minimization |
| `react-markdown@10.1.0` + `remark-gfm@^4`; NO `rehype-raw` | Raw HTML passthrough is XSS risk in preview pane; code-file syntax highlighting deferred to v3.5 with editor | ✓ Good — secure-by-default markdown |
| Image previews via `<img src="/api/files/read?...">` — NOT base64-in-state | 33% overhead + GC pressure for non-trivial files; SVG rendered as text (never as embedded SVG) | ✓ Good — performance and security |
| TUI `bubbles list.Model` + `viewport.Model` joined via `lipgloss.JoinHorizontal` — NOT `bubbles/filepicker` | filepicker is a selection-dialog primitive, not a browse pane | ✓ Good — correct primitive choice |
| TUI ALL filesystem I/O via `tea.Cmd` (returning `tea.Msg`) | Synchronous `os.ReadDir` freezes Bubble Tea render loop; static-grep gate `TestFiles_NoSyncFSCalls` enforces | ✓ Good — Bubble Tea idiom respected |
| TUI status line left-truncated (`…/utils/helper.ts`) | Preserves high-information leaf-end (filename) over root-end (cwd context) | ✓ Good — UX-correct truncation |
| `charmbracelet/glamour` promoted from indirect to direct dep | Markdown rendering parity between TUI Files view and desktop preview pane | ✓ Good — single source of truth for markdown rendering |
| `/` filter activation key (NOT Cmd-F) | Parity with TUI; Cmd-F is xterm.js scrollback search; collision would break user mental model | ✓ Good — surface-parity preserved |
| Phase 122 daemon proxy + `RemoteCapStore` (desktop GUI path) | Desktop GUI hits local daemon, which proxies to remote peer's Phase 119 routes; keeps cap material out of renderer process | ✓ Good — clean trust separation |
| Phase 122 TUI direct HTTPS (TUI path) | TUI has no daemon-hop need; `RemoteFilesClient` dials remote peer directly with cap, TLS 1.2+ pinned, cap redacted from errors | ✓ Good — minimal indirection for TUI |
| Phase 122 cross-surface parity proven by 3 independent observers | daemon-proxy Go + tui.RemoteFilesClient Go + Playwright HTTPS browser observe byte-identical responses against shared fixture | ✓ Good — strongest possible automated evidence pending two-machine UAT |
| Phase 122 mid-milestone insertion (audit-driven) | Phase 121 local-only scope cut surfaced cross-surface remote-browse as release-blocking gap; same pattern as v3.3 Phases 107/108 | ✓ Good — pattern reaffirmed as v3.4 release-blocking contract |
| Settings `schemaVersion: 3` migration via defaults-merge constructor | Same pattern as v3.2 `schemaVersion: 2`; new field `daemonSettings.FilesRead` | ✓ Good — load-bearing pattern; future settings should mirror |

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
*Last updated: 2026-06-16 — v3.5.1 Remote Browse Completion + Release-Gate Fix SHIPPED + archived (tag v3.5.1). 2 phases (129-130), 11/11 requirements (RACE-01..03, DNS-01..03, RB-01..05), 7 plans. Audit PASSED; cross-phase integration 11/11 wired. Closed #86 (tailnet-trusted metadata-only `/api/sessions/meta` discovery), #83 (accept-dns actionable error + proactive banner), #87 (per-path single-winner `WriteFileAtomic` lock). Retired umbrella epic #24 — desktop GUI remote-browse on-ramp (discover→list→pick→browse) proven live on a two-machine tailnet. 5 UAT-surfaced bugs fixed in-milestone (DNS-banner gating, App.js RPC binding, write-toggle re-hydration, join-code-as-text, "5-char"→8-char copy). Operator follow-ups before next tagged release: RELEASE_PUBLISH_TOKEN PAT + WINGET_FIRST_SUBMISSION (see STATE.md); wails CLI v2.10.2 vs repo-pin v2.12.0 mismatch + unsigned-build re-sign (audit tech_debt). Next: `/gsd:new-milestone`.*
