# Feature Landscape

**Domain:** Terminal multiplexer desktop app with web serving for AI coding CLIs
**Researched:** 2026-03-17

---

## Table Stakes

Features users expect from any terminal host for AI coding CLIs. Missing = product feels broken or incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Tabbed terminal sessions | Every modern terminal (iTerm2, Windows Terminal, VS Code) uses tabs; power users run many agents in parallel | Low | Tab rename, reorder, color-coding are secondary expectations within this feature |
| Per-tab session naming | Tabs need identification — users run claude/codex/gemini simultaneously and need to tell them apart at a glance | Low | Defaults to CLI name + working dir; user can rename |
| Session persistence across app restart | tmux-style: if the app crashes or is restarted, sessions survive | Medium | Requires PTY backend to maintain process even when frontend disconnects |
| Scrollback buffer | Reviewing what the agent did is core workflow; reading agent output is as important as interacting with it | Low | xterm.js provides this; configure adequate buffer depth (10K+ lines) |
| Resize / reflow on window resize | SIGWINCH propagation to PTY; agents that print progress bars or TUI output (like Claude Code) break on stale COLS/ROWS | Low | Standard PTY behavior, but must be wired through websocket correctly |
| Copy/paste from terminal | Essential for grabbing file paths, command output, error messages | Low | Browser clipboard API (Ctrl+C/Ctrl+V); xterm.js handles this with configuration |
| ANSI/color rendering | AI coding CLIs output rich ANSI color; broken colors = broken output readability | Low | xterm.js handles this natively |
| Unicode/emoji support | Claude Code and other CLIs output emoji and Unicode symbols in progress displays | Low | xterm.js Unicode11 addon required |
| Launch a new session with a chosen CLI | The core UX entry point — pick from a menu of supported CLIs, launch in a new tab | Low | Depends on: CLI detection feature |
| Detection of installed CLIs | App should know which of the supported CLIs are installed and surfaced as launchable | Low | PATH lookup at startup, re-checked on demand |
| Kill / close a session | Users need to terminate stuck agents cleanly | Low | Send SIGHUP to PTY child, close tab |

---

## Differentiators

Features that set AgentHub apart from both generic terminals (iTerm2, Windows Terminal) and raw web terminal tools (ttyd, GoTTY). Users of AI coding CLIs will value these but don't currently expect them from a single tool.

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Web serving per session via hosted xterm.js | "Walk away" workflow — start an agent, step away from desk, check progress on phone or another machine | High | Session persistence, TLS, auth | Claude Code's own Remote Control launched Feb 2026 via Anthropic servers; AgentHub provides a self-hosted, all-CLI equivalent |
| Per-session web toggle (on/off) | Not all sessions should be web-accessible; toggle prevents accidental exposure | Low | Web serving | Single UI control per tab |
| Self-signed TLS for all web connections | VPN provides network-level trust; self-signed TLS adds transport encryption without a domain name or CA dependency | Medium | Web serving, VPN binding | Must handle browser "untrusted cert" UX (user-add cert flow or persistent exception) |
| VPN interface binding (Tailscale-first) | Exposes sessions only on the VPN interface, keeping them off the public internet without firewall rules | Medium | Web serving, TLS | Tailscale100.x.x.x is the canonical use case; also supports arbitrary interface selection for other VPNs |
| QR code generation for session URLs | Fastest path from desktop to phone — no copy-pasting URLs; validated pattern from ccrs predecessor | Low | Web serving | Encodes the full session URL with auth token; display in-app and in web dashboard |
| Web dashboard with password auth | Browse all web-served sessions from a browser; centralized access point | Medium | Web serving, TLS, auth | Password protects the dashboard; per-session tokens provide shareable links without exposing the dashboard password |
| Per-session shareable tokens | Give a teammate a link to observe (or interact with) a running session without sharing your master password | Medium | Web serving, auth | Token should be scoped, time-limited, and revocable |
| Real tmux backend mode | Power users who live in tmux want `tmux attach` semantics — attach to the same session from any terminal independently of the app | Medium | PTY abstraction layer | Optional; requires tmux installed. Go-native PTY mode is the default. |
| Go-native PTY mode (no dependencies) | Works on any machine without tmux; sessions persist natively in the Go backend | Medium | None | Default mode for fresh installs; sessions are internal to the app process |
| Multi-CLI session status indicators | Show per-tab whether the agent is running (processing), waiting for user input, idle, or errored — like agent-deck's ●/◐/○/✕ system | High | Session monitoring | Requires parsing agent-specific output patterns. Complexity varies by CLI; Claude Code is most heuristically parseable |
| Working directory context per tab | Display the cwd the agent was launched from; helps when running multiple agents across different projects | Low | Tab metadata | Captured at launch time, updated if agent changes dir (optional) |
| System tray / menubar presence | Keep sessions alive in the background when the main window is closed; access from tray | Medium | App lifecycle management | Critical for "walk away" workflow on desktop — app should not terminate sessions when window is closed |

---

## Anti-Features

Features to explicitly NOT build. Each one has been considered and rejected for a concrete reason.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Mobile app | Out of scope (PROJECT.md); web UI served from the desktop app is the remote access mechanism | Use the web-served xterm.js UI in any mobile browser |
| CLI installation / management | Increases scope dramatically; version management, update handling, permissions are each a project; adds liability if an install goes wrong | Detect installed CLIs via PATH; surface a clear "not installed" message with a link to the CLI's docs |
| Tailscale / VPN installation or management | VPN setup is a separate concern with significant privilege requirements (kernel extensions, system services) | Read available VPN interfaces at runtime; document that VPN setup is the user's responsibility |
| Let's Encrypt / ACME cert management | Requires a domain name and an internet-accessible HTTP-01 or DNS-01 challenge endpoint; this is a local-first app with no guaranteed domain | Self-signed TLS is sufficient; VPN provides the network trust boundary |
| User account / registration system | AgentHub is single-user per installation; a registration system implies multi-tenancy and identity management | Password + token model handles all required auth; "forgot password" = edit config |
| Cloud hosting / SaaS deployment mode | Adds a hosted backend, data residency, billing, and support burden; contradicts the local-first design | Sessions stay on the user's machine; web serving exposes them over VPN only |
| Plugin system for new CLIs | Adding extensibility infrastructure before validating core workflows is premature; plugin ABIs require versioning and compatibility guarantees | Hardcode the initial CLI set (Claude Code, Codex, Gemini CLI, OpenCode); add more via code contributions |
| Session output search / replay | Valuable but high complexity — requires structured log storage, indexing, and a query UI; tools like agent-sessions (native macOS app) already serve this niche | Provide adequate scrollback buffer; full replay/search is a future scope item |
| MCP server management | Agent-deck targets this niche with deep tmux integration; it requires understanding per-CLI MCP config formats and socket lifecycle management | Out of scope for v1; point users to agent-deck for MCP orchestration |
| Multi-user concurrent editing of a session | Increases auth complexity significantly (who owns a session? who can write?); opens denial-of-service vectors via web | Per-session tokens are view-or-interact, not multi-write; observe vs. control is the correct split |
| Notification push to phone | Telegram/Slack integrations (as in agent-deck conductors) are complex to configure reliably and introduce external service dependencies | QR code + browser tab as the phone interface is sufficient for the "check in" use case |
| Split panes / tiling within a tab | Increases terminal rendering complexity significantly; the use case is covered by simply opening multiple tabs | Each AI coding session gets its own tab; split panes are a distraction from the multi-tab model |

---

## Feature Dependencies

```
CLI detection
  └─> Launch session (session creation)
        └─> PTY backend (tmux or Go-native)
              └─> Session persistence
              └─> xterm.js terminal rendering (tab UI)
                    └─> Scrollback, resize, copy/paste, ANSI, Unicode

Web serving per session
  └─> Session must exist (session creation)
  └─> Self-signed TLS
  └─> Password auth (dashboard)
        └─> Per-session shareable tokens
        └─> QR code generation
  └─> VPN interface binding (optional; enhances web serving)

System tray presence
  └─> Session persistence (sessions must survive window close)

Multi-CLI status indicators
  └─> Session running (PTY backend)
  └─> Output pattern detection per CLI (CLI-specific)
```

---

## MVP Recommendation

The minimum product that validates the core value proposition ("one app, multiple AI agents, web-accessible from anywhere"):

**Must have in v1:**
1. Tabbed xterm.js terminal with session naming and ANSI/Unicode rendering
2. Launch sessions for Claude Code, Codex, Gemini CLI, OpenCode (PATH detection)
3. Go-native PTY backend with session persistence
4. Per-session web serving toggle with self-signed TLS
5. Web dashboard with password auth and per-session shareable token links
6. QR code generation for web-served session URLs
7. VPN interface selection (Tailscale interface detection as default)
8. System tray / menubar icon to keep sessions alive with window closed

**Defer to v2:**
- Real tmux backend (adds complexity; Go-native PTY covers the use case)
- Multi-CLI status indicators (heuristic parsing is fragile; ship it once the rest is stable)
- Per-session token expiry / revocation (useful, but MVP tokens can be long-lived)
- Tab color coding (polish, not function)
- Font/theme customization (xterm.js defaults are usable; theming is a polish layer)

**Never build (anti-features above).**

---

## AI Coding CLI-Specific Considerations

These are needs specific to users of Claude Code, Codex, Gemini CLI, and OpenCode — distinct from generic terminal users:

1. **Long-running autonomous sessions.** AI agents run for minutes to hours. Session persistence and "walk away" remote access are not niceties — they are the primary reason to use AgentHub over a bare terminal.

2. **Multiple parallel agents.** A common workflow is running one agent per feature branch / git worktree simultaneously. Tab management and clear session identity (name, working dir) are load-bearing.

3. **Approvals and interaction.** Claude Code in particular pauses for user approval. The web interface must support full interaction (not read-only) so users can approve changes from phone or secondary machine without returning to the desktop.

4. **Output volume.** AI coding CLIs produce large volumes of structured output (file diffs, tool call logs, progress bars). Adequate scrollback (≥10K lines), correct ANSI rendering, and Unicode support are non-negotiable.

5. **Context about the agent's state.** Users want to know at a glance: "is this agent done, waiting, or still thinking?" Status indicators address this; even a simple heuristic (last output line matches known "waiting for input" patterns) is valuable.

6. **CLI version matters.** Claude Code, Codex, and Gemini CLI are actively developed. The app should not hard-code CLI invocation paths in ways that break on version updates. Lean on `claude`, `codex`, `gemini`, `opencode` as PATH commands with optional path overrides in config.

---

## Sources

- [ttyd official site](https://tsl0922.github.io/ttyd/) — feature reference for web terminal table stakes (HIGH confidence: official source)
- [agent-deck README](https://github.com/asheshgoplani/agent-deck) — AI coding agent session manager feature set (HIGH confidence: official source)
- [Claude Code Remote Control docs](https://code.claude.com/docs/en/remote-control) — remote session QR code workflow, Feb 2026 (HIGH confidence: official Anthropic source)
- [GoTTY GitHub](https://github.com/yudai/gotty) — multi-session limitations, basic auth (HIGH confidence: official source)
- [Wails framework](https://wailsapp.io/) — single-binary constraint, system tray support (HIGH confidence: official source)
- [Wetty GitHub](https://github.com/butlerx/wetty) — Node.js web terminal, xterm.js integration pattern (HIGH confidence: official source)
- [DevOps.com: Claude Code Remote Control](https://devops.com/claude-code-remote-control-keeps-your-agent-local-and-puts-it-in-your-pocket/) — "walk away" use case validation (MEDIUM confidence: secondary source)
- [2026 Guide to CLI Coding Tools — Tembo](https://www.tembo.io/blog/coding-cli-tools-comparison) — AI coding CLI landscape, user expectations (MEDIUM confidence: third-party analysis)
- [agent-sessions GitHub](https://github.com/jazzyalex/agent-sessions) — session search/replay as a distinct niche product (MEDIUM confidence: existence validates anti-feature decision)
- [Julia Evans: Getting a modern terminal setup](https://jvns.ca/blog/2025/01/11/getting-a-modern-terminal-setup/) — tab management, color, UX norms (MEDIUM confidence: practitioner reference)
