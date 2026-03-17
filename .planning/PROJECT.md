# AgentHub

## What This Is

A cross-platform desktop app (macOS, Linux, Windows) for running AI coding CLIs — Claude Code, OpenCode, Codex, Gemini CLI, and others — in tabbed terminal sessions powered by xterm.js. Built with Go/Wails for the backend and React for the frontend. Every session can optionally be served over the web via TLS with authentication, accessible from any browser via URL or QR code, and bindable to a VPN interface (Tailscale-first, but any VPN supported).

## Core Value

One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Tabbed terminal UI with xterm.js for running multiple AI coding CLI sessions simultaneously
- [ ] Support for major AI coding CLIs: Claude Code, OpenCode, Codex, Gemini CLI
- [ ] Configurable session backend: real tmux (when available) or Go-native PTY multiplexer
- [ ] Real tmux mode: sessions attachable from any external terminal via `tmux attach`
- [ ] Go-native PTY mode: built-in session persistence with no external dependencies
- [ ] Web serving of terminal sessions via hosted xterm.js process
- [ ] Per-session toggle for web serving (on/off)
- [ ] Self-signed TLS certificates for all web connections
- [ ] Web dashboard with password authentication to browse all served sessions
- [ ] Per-session shareable tokens/links for quick access
- [ ] QR code generation for all web-served sessions
- [ ] VPN interface binding — Tailscale-first, with support for other VPN interfaces
- [ ] Multi-platform: macOS, Linux, Windows
- [ ] Wails desktop shell with React frontend
- [ ] Go backend serving both the desktop app and web interface on the same process

### Out of Scope

- Mobile app — desktop and web access only
- AI coding CLI installation — app checks for CLIs but doesn't install them
- Tailscale/VPN installation or management — user handles VPN setup separately
- End-to-end encryption beyond TLS — TLS with self-signed certs is sufficient
- User account system with registration — password + tokens, not a full identity system
- Cloud hosting or SaaS deployment — this is a local-first desktop app
- Plugin system for adding new CLIs — initial set is hardcoded, extensibility is future scope

## Context

Spiritual successor to ccrs (Claude Code Remote Sessions) — a zsh script project that handled Tailscale setup, tmux sessions, ttyd web terminals, and remote peer discovery. AgentHub takes the same core workflows (session launch, web serving, remote access, QR codes) and reimplements them as a proper desktop application with a modern UI and multi-CLI support.

Key prior art from ccrs:
- ttyd-style web terminal serving bound to VPN interface
- Self-signed TLS and basic auth for web access
- QR code generation for session URLs
- tmux session management with attach/detach semantics
- Tailscale IP discovery and network binding

## Constraints

- **Tech stack**: Go backend (Wails), React frontend, xterm.js for terminals
- **Single binary**: Wails compiles to a single distributable binary per platform
- **No cloud dependency**: Everything runs locally — no external services required
- **Cross-platform**: Must work on macOS, Linux, and Windows from day one

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go/Wails over Electron | Smaller binary, better performance, native Go ecosystem for PTY/tmux | — Pending |
| xterm.js for terminal rendering | Industry standard, well-maintained, used by VS Code terminal | — Pending |
| Real tmux + Go-native PTY as configurable option | Maximum flexibility — tmux for power users, native PTY for zero-dep installs | — Pending |
| Same Go process serves desktop + web | Simpler architecture, single port management, shared session state | — Pending |
| Self-signed TLS (not Let's Encrypt) | Local-first app, no domain name, VPN provides network-level security | — Pending |
| Password + token auth model | Password for dashboard access, tokens for shareable per-session links | — Pending |

---
*Last updated: 2026-03-17 after project initialization*
