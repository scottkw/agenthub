# Milestones

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
