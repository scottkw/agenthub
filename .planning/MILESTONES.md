# Milestones

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
