# AgentHub

## What This Is

A cross-platform desktop app (macOS, Linux, Windows) for running AI coding CLIs — Claude Code, OpenCode, Codex, Gemini CLI, and others — in tabbed terminal sessions powered by xterm.js. Built with Go/Wails for the backend and React for the frontend. Every session can optionally be served over the web via TLS with authentication, accessible from any browser via URL or QR code, and bindable to a VPN interface (Tailscale-first, but any VPN supported). Live status indicators show whether each CLI is running, waiting, or errored. Includes a polished UI with tabbed settings, per-tab font sizing, new-session modal with agent picker and folder browser, tab renaming with web dashboard propagation, and a cross-platform build script with macOS signing support.

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
- ✓ Web dashboard with password authentication to browse all served sessions — v1.0
- ✓ Per-session shareable tokens/links for quick access — v1.0
- ✓ QR code generation for all web-served sessions — v1.0
- ✓ VPN interface binding — Tailscale-first, with support for other VPN interfaces — v1.0
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
- ⚠️ Terminal fill: CSS flex chain fixed, fills after resize — initial-paint timing gap remains (tabled) — v1.1

### Active

**Current Milestone: v1.2 Tailscale-Only Networking**

**Goal:** Simplify networking to Tailscale-only — use its Let's Encrypt certs, remove self-signed TLS and password/token auth, add health checks with user-friendly guidance.

**Target features:**
- Tailscale-only networking (remove generic VPN interface support)
- Tailscale health checks (installed, connected, certs enabled) with instructional modal
- Let's Encrypt certs via Tailscale for web server and remote sessions
- Remove password auth, per-session tokens, and self-signed cert infrastructure

### Out of Scope

- Mobile app — desktop and web access only; PWA via web serving covers mobile needs
- AI coding CLI installation — app checks for CLIs but doesn't install them
- Tailscale/VPN installation or management — user handles VPN setup separately
- End-to-end encryption beyond TLS — TLS with self-signed certs is sufficient
- User account system with registration — password + tokens, not a full identity system
- Cloud hosting or SaaS deployment — this is a local-first desktop app
- Plugin system for adding new CLIs — initial set is hardcoded, extensibility is future scope
- Session output search / replay — tools like agent-sessions serve this niche
- Split panes / tiling within a tab — each AI session gets its own tab
- Configurable session backend (tmux vs Go-native) — deferred to future milestone
- Real tmux mode with `tmux attach` — deferred to future milestone
- Per-session token expiry and revocation — deferred to future milestone
- Tab color coding per CLI type — deferred to future milestone
- Status heuristic patterns for non-Claude CLIs — deferred to future milestone
- Font/theme customization beyond size — per-tab font size covers the immediate need

## Context

Shipped v1.1 with ~9,956 LOC (6,541 Go + 2,622 TS/TSX + 793 CSS).
Tech stack: Go/Wails v2, React, xterm.js, nhooyr/websocket, go-pty, skip2/go-qrcode.
Frontend test suite: 73 vitest tests (source-inspection pattern for xterm.js/Wails constraints).
Go test suite: race-clean, webserver tests with resolver coverage.
Status heuristics implemented for Claude CLI; other CLIs always show "running" (deferred).
Build script: `build.sh` compiles for macOS/Linux/Windows with optional macOS signing/notarization.

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
| Self-signed TLS with CA+leaf pattern | Local-first, no domain; CA pattern lets browsers trust leaf certs | ✓ Good — WSS works without browser errors after CA install |
| Password + token auth model | Password for dashboard, tokens for shareable per-session links | ✓ Good — simple, effective for single-user |
| go-pty (aymanbagabas) over creack/pty | Windows ConPTY support required from day one | ✓ Good — cross-platform PTY with single API |
| Binary framing protocol for WS relay | Distinguishes output/resize/input frames; enables scrollback replay | ✓ Good — clean separation of message types |
| Native macOS cgo NSStatusBar for tray | fyne.io/systray conflicts with Wails AppDelegate (duplicate symbol) | ⚠️ Revisit — platform-specific code, Linux/Windows stubs needed |
| ResizeObserver + requestAnimationFrame for fit() | Handles all layout changes, not just window resize | ✓ Good — fixed terminal height issues |
| ?raw source-inspection tests for xterm.js components | jsdom lacks Canvas/WebGL; runtime mocking xterm is fragile | ✓ Good — stable tests that verify code structure without DOM |
| JSX conditionals over CSS display toggle | Consistent pattern across StatusBar, SettingsPanel tabs | ✓ Good — cleaner React patterns, easier to test |
| build.sh with Docker for Linux cross-compile | No native Linux WebKitGTK headers on macOS | ✓ Good — portable Linux builds from any OS |
| ditto (not zip) for notarization archive | Preserves macOS extended attributes required by notarytool | ✓ Good — correct signing pipeline |

---
*Last updated: 2026-03-20 after v1.1 milestone*
