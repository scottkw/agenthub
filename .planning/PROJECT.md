# AgentHub

## What This Is

A cross-platform desktop app (macOS, Linux, Windows) for running AI coding CLIs — Claude Code, OpenCode, Codex, Gemini CLI, and others — in tabbed terminal sessions powered by xterm.js. Built with Go/Wails for the backend and React for the frontend. Every session can optionally be served over the web via TLS with authentication, accessible from any browser via URL or QR code, and bindable to a VPN interface (Tailscale-first, but any VPN supported). Live status indicators show whether each CLI is running, waiting, or errored.

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

### Active

- [ ] Build script (`build.sh`) for per-platform and all-platform compilation with macOS signing support
- [ ] UI/UX overhaul: settings modal declutter/restyle, web dashboard visual refresh
- [ ] Per-tab status bar replacing header overlay for web status/URL/controls
- [ ] Tab renaming with name propagation to web dashboard session names
- [ ] Larger toolbar buttons (current ones are too small)
- [ ] New-session modal with agent picker + folder browser (remembers last folder)
- [ ] Per-tab SHIFT+/SHIFT- font size adjustment
- [ ] Terminal fill: fix terminals not using full available space

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

## Current Milestone: v1.1 Polish & Build

**Goal:** Improve UI/UX across desktop and web, fix terminal sizing, and add a build script for cross-platform compilation with signing.

**Target features:**
- Build script with per-platform compilation and macOS signing
- UI/UX overhaul (settings modal, web dashboard, toolbar, new-session modal)
- Per-tab status bar, tab renaming, font size shortcuts
- Terminal full-window fill fix

## Context

Shipped v1.0 with ~8,100 LOC (6,400 Go + 1,700 JS/TS/CSS) across 149 files.
Tech stack: Go/Wails v2, React, xterm.js, nhooyr/websocket, go-pty, skip2/go-qrcode.
Status heuristics implemented for Claude CLI; other CLIs always show "running" (deferred).
9 items need human verification (TLS browser flows, visual elements, CI execution).

Spiritual successor to ccrs (Claude Code Remote Sessions) — a zsh script project that handled Tailscale setup, tmux sessions, ttyd web terminals, and remote peer discovery.

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

---
*Last updated: 2026-03-19 after v1.1 milestone started*
