# Roadmap: AgentHub

## Overview

AgentHub is built from the inside out: PTY process management first, then the session registry and WebSocket relay that wires everything together, then the local desktop UI, then the remote web serving layer, then the differentiating features (QR codes, status indicators), and finally the distribution pipeline. Each phase delivers a coherent, verifiable capability that the next phase builds on.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: PTY Foundation** - Cross-platform PTY process management, CLI detection, session lifecycle (completed 2026-03-18)
- [ ] **Phase 2: Session Registry + WebSocket Relay** - In-memory session state, fan-out hub, WebSocket protocol
- [ ] **Phase 3: Wails Desktop UI** - Tabbed xterm.js terminal, session naming, system tray, full local UX
- [ ] **Phase 4: Web Serving + TLS + Auth** - Embedded HTTPS server, self-signed TLS, dashboard, token auth, VPN binding
- [ ] **Phase 5: QR Codes + Status Indicators** - QR code generation, session status heuristics, per-session URL display
- [ ] **Phase 6: Distribution + Cross-Platform** - GitHub Actions CI matrix, macOS notarization, Linux/Windows build validation

## Phase Details

### Phase 1: PTY Foundation
**Goal**: A Go binary can spawn AI coding CLIs in a proper PTY on macOS, Linux, and Windows; read/write PTY I/O; resize correctly; detect installed CLIs; and kill sessions cleanly on shutdown with no orphan processes.
**Depends on**: Nothing (first phase)
**Requirements**: CLI-01, CLI-02, TERM-06, TERM-07, SESS-01
**Success Criteria** (what must be TRUE):
  1. Running the binary and typing into Claude Code / Gemini CLI produces correct interactive terminal output (real PTY, not a pipe — isatty() passes)
  2. Resizing the terminal window propagates SIGWINCH to the child process and the CLI reflows its output correctly
  3. Closing the session sends SIGHUP and kills the entire process group — no orphan CLI processes remain after session close
  4. App startup scans PATH and identifies which of Claude Code, Codex, Gemini CLI, OpenCode are available
  5. Session state persists in memory (Go-native PTY backend) — process continues running after the conceptual "window close" event is triggered in test code
**Plans:** 2/2 plans complete

Plans:
- [ ] 01-01-PLAN.md — Scaffold Go module, define interfaces, implement CLI detection and session registry
- [ ] 01-02-PLAN.md — Implement NativePTYBackend (spawn, resize, kill), win32-input parser, smoke-test binary

### Phase 2: Session Registry + WebSocket Relay
**Goal**: An in-memory SessionRegistry tracks all sessions; a WebSocket fan-out hub broadcasts PTY output to N connected clients; the binary framing protocol is defined and tested; connection lifecycle (connect, disconnect, reconnect) is handled correctly.
**Depends on**: Phase 1
**Requirements**: SESS-03
**Success Criteria** (what must be TRUE):
  1. Two WebSocket clients connected to the same session both receive the same PTY output simultaneously without data loss or corruption
  2. A client that disconnects and reconnects receives the current session state and resumes receiving output without restarting the PTY process
  3. Input sent from any connected client reaches the PTY process and produces the expected output visible to all clients
**Plans:** 2/2 plans complete

Plans:
- [ ] 02-01-PLAN.md — Binary framing protocol, bounded scrollback buffer, per-session fan-out Hub
- [ ] 02-02-PLAN.md — HubManager, HTTP/WebSocket server, integration tests proving SESS-03

### Phase 3: Wails Desktop UI
**Goal**: Users can open the AgentHub desktop app, launch AI coding CLI sessions in named tabs, interact with them via a fully-functional xterm.js terminal (ANSI color, Unicode, scrollback, copy/paste, resize), and close the window without killing their sessions.
**Depends on**: Phase 2
**Requirements**: TERM-01, TERM-02, TERM-03, TERM-04, TERM-05, CLI-03, SESS-02
**Success Criteria** (what must be TRUE):
  1. User can open multiple tabs, each running an independent session, and switch between them without losing terminal state
  2. User can rename any tab — the new name persists across session reattachment and is visible in the tab bar
  3. Terminal renders Claude Code's full color UI output (ANSI 256-color, emoji, box-drawing characters) without corruption
  4. User can scroll back through 10,000+ lines of output using the scrollbar or keyboard shortcuts
  5. User can copy text from the terminal and paste it back in; the app window can be closed to the system tray and sessions remain alive, resumable on reopen
**Plans:** 3 plans

Plans:
- [ ] 03-01-PLAN.md — Wails scaffold, Go App struct with bound methods, relay resize wiring, React frontend skeleton
- [ ] 03-02-PLAN.md — React frontend: xterm.js terminal, relay client, tab management, settings panel
- [ ] 03-03-PLAN.md — System tray integration, build verification, visual checkpoint

### Phase 4: Web Serving + TLS + Auth
**Goal**: Users can toggle web serving on any session, access it from a remote browser over HTTPS (with TLS that the browser trusts), authenticate via a dashboard password, share per-session token links, and bind the web server to a specific network interface including auto-detected Tailscale.
**Depends on**: Phase 3
**Requirements**: WEB-01, WEB-02, WEB-03, WEB-04, WEB-05, WEB-06, NET-01, NET-02, NET-03
**Success Criteria** (what must be TRUE):
  1. Toggling web serving on a session produces an HTTPS URL; a remote browser can load that URL, accept the CA cert (installed via in-app guidance), and connect to the session over WSS without a browser security error
  2. The web dashboard at the root URL requires a password to access and lists all currently web-served sessions
  3. A per-session token link works without dashboard login — the token grants access to exactly that one session
  4. Remote browser interaction with the session is fully bidirectional — the remote user can type commands and see output, not just observe
  5. User can select a specific network interface (or let Tailscale auto-detect) from the desktop app, and the web server binds to that interface's IP only
**Plans**: TBD

### Phase 5: QR Codes + Status Indicators
**Goal**: Every web-served session has a QR code visible in the desktop app and on the web dashboard; each tab shows a live status indicator (running, waiting for input, idle, errored) based on heuristic output parsing.
**Depends on**: Phase 4
**Requirements**: QR-01, QR-02, STAT-01, STAT-02
**Success Criteria** (what must be TRUE):
  1. When a session has web serving enabled, a QR code appears in the desktop UI and on the web dashboard — scanning it with a phone opens the session URL
  2. Each tab in the desktop app shows a status badge that updates without manual refresh: "running" when the CLI is actively producing output, "waiting" when it has shown a prompt and is idle, and "errored" when the process has exited non-zero
**Plans**: TBD

### Phase 6: Distribution + Cross-Platform
**Goal**: AgentHub builds cleanly on macOS, Linux, and Windows via a GitHub Actions CI matrix; macOS builds are signed and notarized; Linux builds handle WebKitGTK version variants; Windows builds produce a usable installer. Each platform produces a working binary that passes the Phase 3 and Phase 4 success criteria on that platform.
**Depends on**: Phase 5
**Requirements**: PLAT-01, PLAT-02, PLAT-03
**Success Criteria** (what must be TRUE):
  1. GitHub Actions CI runs on macOS-latest, ubuntu-latest, and windows-latest; all three jobs pass without manual intervention
  2. macOS build is signed and notarized — Gatekeeper allows it to open without a warning dialog
  3. Linux build runs on Ubuntu 22.04 and 24.04 (WebKitGTK 4.0 and 4.1 variants both tested)
  4. Windows build produces an installer; the app launches, PTY sessions work with correct keyboard input (win32-input-mode handled), and self-signed TLS certs can be installed in the Windows cert store
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. PTY Foundation | 2/2 | Complete   | 2026-03-18 |
| 2. Session Registry + WebSocket Relay | 2/2 | Complete |  2026-03-18 |
| 3. Wails Desktop UI | 0/3 | Not started | - |
| 4. Web Serving + TLS + Auth | 0/? | Not started | - |
| 5. QR Codes + Status Indicators | 0/? | Not started | - |
| 6. Distribution + Cross-Platform | 0/? | Not started | - |
