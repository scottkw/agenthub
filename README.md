# AgentHub

<p align="center">
  <img src="docs/agenthub-title-logo.png" alt="AgentHub" width="400">
</p>

AgentHub runs AI coding CLIs — Claude Code, Codex, Gemini CLI, OpenCode, Google Antigravity — and raw shell sessions (bash, zsh, PowerShell) in persistent terminal sessions managed by a background daemon. As of **v4.0** the desktop GUI is **Hub-first**: a single top-level Hub shows every local and remote session as a live, colorblind-safe card you can filter, group, share, and expand into a terminal. Google Antigravity (`agy`) is integrated as a selectable agent and was verified live (interactive REPL, authenticated session) in v4.0; it remains in limited/waitlist rollout, so it appears in the picker once `agy` is installed. Two surfaces share one set of sessions: a Wails desktop GUI and the `agenthub` CLI (the previous Bubble Tea TUI was retired in v4.0 — cross-surface parity is now GUI/CLI/web). Sessions survive GUI restarts and are controllable from either surface. Agent CLIs are discovered automatically across common install locations (nvm, Volta, Homebrew, snap, flatpak, cargo, pipx, and native installers).

Sessions can be shared over the web on a per-session basis: browser-trusted TLS over Tailscale when available, with a self-signed TLS + password-auth fallback for the local network. Multiple clients can connect to the same session simultaneously with independent scrollback, optional read-only mode, and stable PTY resize arbitration. CLI `attach` displays a tmux-style status bar with session context and a live viewer count. Remote sessions on other tailnet machines are discoverable from both surfaces. As of **v4.1**, every session also carries a human-to-human **chat** side channel (tailnet identity + alias, presence/typing, daemon-persisted, Markdown export) with a one-way `@session`→PTY-prompt bridge to the agent — at full parity across the desktop GUI and the web-share surface.

The terminal core is extended by a curated, vendored xterm.js plugin suite — GPU-accelerated rendering with software-fallback detection, Cmd-F scrollback search with regex/case/word toggles, clickable web links with strict scheme allowlist and IDN/typosquat confirmation, inline sixel images, Unicode 11 width tables, "Save Terminal As…" via the serialize addon, OSC 52 system clipboard, and OSC 9;4 progress reporting (per-tab progress underlines plus a tray quartile glyph) — all user-controlled from Settings → Plugins. Sessions support 138 WCAG-audited color themes with live switching across the AI CLIs (including OpenCode, via SIGUSR2 broadcast); all non-terminal GUI text meets WCAG AA 4.5:1 contrast.

Sessions auto-close when the agent process exits — 5-second countdown, toast notification, and Keep Open cancel — and closing the GUI prompts a quit confirmation showing active session count. The system tray is native on each platform: NSStatusBar on macOS, D-Bus StatusNotifierItem on Linux, and Shell_NotifyIcon on Windows. Auto-update notifications keep you on the latest release. Built with Go/Wails and React.

## Latest Release

**v4.1 — Session Chat** (2026-06-28) — adds a per-session, human-to-human chat side channel inside AgentHub, so collaborators connected to a session talk to each other (and pipe prompts to the agent) without leaving for Slack/Discord.

- **Per-session chat** — Each session has its own chat thread, attached to the session and reachable from the GUI session tab, the Hub interactive modal, and the web-share guest surface. Enter sends, Shift+Enter inserts a newline; the composer auto-grows and renders Markdown safely (sanitized, no raw HTML/scripts). Day separators stick to the top while scrolling.
- **Identity & presence** — Participants are identified by their tailnet ID plus a self-chosen **alias** (settable/changeable right from the chat sidebar), both visible to everyone. Live presence (connected/disconnected) and debounced typing indicators show who's there and who's composing.
- **`@session` agent bridge** — Typing `@` opens an autocomplete over the session's participants plus a pinned `@session` target. `@session <text>` injects the message into the agent's PTY as a prompt (one-way — the agent's reply stays in the terminal), gated to read/write holders, sanitized before injection, and confirmed with a deliberate press-and-hold.
- **Durability & export** — Threads are persisted by the daemon for the session's life (survive daemon/app restart, full late-join scrollback) and deleted when the session is deleted. Any thread can be downloaded as a Markdown file from both surfaces.
- **Read-only guests can chat** — Read-only capability holders are full chat participants (post messages, set an alias) while `@session` inject and raw PTY input stay read-only-gated, enforced server-side.
- **Cross-surface parity (release-blocking)** — Every chat feature behaves identically on the desktop GUI and the web-share browser; the actually-shared link (`/sessions/{id}?cap=`) redirects to the chat-capable surface, and an unread badge surfaces on the chat toggle and the Hub session card.
- **Terminal screen-share fidelity** ([#109](https://github.com/scottkw/agenthub/issues/109)) — the host's terminal is the single source of truth for the PTY grid; guests render at the host's grid size and CSS-scale to fit their viewport (downscale-only), eliminating cross-viewer character garble when windows differ in size.
- **Settings & install polish** — the Settings "Plugins" jump link moved to last position and renamed **"Terminal Plugins"** ([#108](https://github.com/scottkw/agenthub/issues/108)); the Welcome screen install commands now use the correct winget package id (`winget install scottkw.agenthub`) and a working Linux `install.sh`.

Closes [#79](https://github.com/scottkw/agenthub/issues/79), [#108](https://github.com/scottkw/agenthub/issues/108), [#109](https://github.com/scottkw/agenthub/issues/109). [Release notes](https://github.com/scottkw/agenthub/releases/tag/v4.1).

**v4.0 — Hub-First Consolidation & UI/UX Overhaul** (2026-06-23) — collapses AgentHub onto a single Hub-centric surface, retires the TUI, ships the v4.0 redesign, and stands up a formal regression-test program.

- **Hub-first navigation** — the top-level **Hub** is now the home for sessions: a working-directory-grouped grid of live session cards (status, agent badge, origin, throttled mini-preview) with a status filter bar, `/` search, named groups, and a card→modal interaction. The separate **Sessions** and **Remote** sidebar pages and the **+ New Session** sidebar item are gone — session creation and remote sessions both live on the Hub. The sidebar is now **Home / Hub / Help / Settings**.
- **TUI retired** — the `agenthub tui` Bubble Tea surface and its code were removed; the cross-surface parity contract narrows to **GUI / CLI / web**.
- **Per-card Share modal** ([#78](https://github.com/scottkw/agenthub/issues/78) follow-through) — each Hub card has a Share modal with two toggles: *Share the session* (reveals read-only + read/write capability links) and *Enable remote file browsing* (browse permission derives from the presented share code). Remote "Open in browser" reuses the held web-share capability instead of minting a second single-use code ([#98](https://github.com/scottkw/agenthub/issues/98)).
- **v4.0 visual redesign** — Plus Jakarta Sans + JetBrains Mono typography, a refreshed color palette/radii/type-scale, and a persisted **Light / Dark** theme toggle, plus a Chrome-style shrink-then-scroll tab strip with a discoverability chevron ([#68](https://github.com/scottkw/agenthub/issues/68)) and WebGL repaint hardening.
- **In-app Help page** ([#69](https://github.com/scottkw/agenthub/issues/69)) — a searchable Help surface in the sidebar with documentation, FAQ, and external links.
- **Google Antigravity agent** ([#65](https://github.com/scottkw/agenthub/issues/65)) — `agy` is a selectable agent across GUI/CLI/web with auto-detection (incl. Windows `%LOCALAPPDATA%\agy\bin`), a status detector, and a distinct agent-color identity; verified live (interactive REPL) in v4.0.
- **Shell web-share warning toggle** ([#51](https://github.com/scottkw/agenthub/issues/51)) — a Settings toggle controls the one-time shell-session web-sharing warning (re-enableable), firing consistently across both share surfaces; shell detection is path-aware.
- **Formal regression-test program** — a consolidated automated suite (Go + vitest + Playwright) with requirement→test traceability, a path-check CI script, a maintained manual checklist, and branch protection on `main` (4 build jobs + Playwright required). Also fixes a daemon styled-tail data race ([#100](https://github.com/scottkw/agenthub/issues/100)) and Windows files test failures ([#101](https://github.com/scottkw/agenthub/issues/101)); the Hub mini-preview/briefing tail now renders through a headless VT emulator ([#96](https://github.com/scottkw/agenthub/issues/96)).

Closes [#51](https://github.com/scottkw/agenthub/issues/51), [#65](https://github.com/scottkw/agenthub/issues/65), [#68](https://github.com/scottkw/agenthub/issues/68), [#69](https://github.com/scottkw/agenthub/issues/69), [#96](https://github.com/scottkw/agenthub/issues/96), [#97](https://github.com/scottkw/agenthub/issues/97), [#98](https://github.com/scottkw/agenthub/issues/98), [#100](https://github.com/scottkw/agenthub/issues/100), [#101](https://github.com/scottkw/agenthub/issues/101). [Release notes](https://github.com/scottkw/agenthub/releases/tag/v4.0).

**v3.5.1 — Remote Browse Completion + Release-Gate Fix** (2026-06-16) — completes the desktop-GUI remote-browse on-ramp (discover → list → pick → browse a tailnet peer's files) and clears the flaky release gate, retiring the remote file-browser epic.

- **Remote-browse GUI on-ramp** ([#86](https://github.com/scottkw/agenthub/issues/86), [#24](https://github.com/scottkw/agenthub/issues/24)) — the Remote Sessions panel now discovers a tailnet peer, lists its shareable sessions, and opens one in the File Browser end-to-end over the relay loopback. A new tailnet-trusted, **metadata-only** `GET /api/sessions/meta` endpoint returns shareable-session metadata only — never capabilities or content — preserving the Phase 87/88 no-enumeration security model. Proven live on two machines.
- **Honest remote panel states** — a reachable peer with shareable sessions is never shown as "No remote peers found"; reachable-but-empty and unreachable peers surface correct, text-labeled states (colorblind-safe, color used only as reinforcement).
- **Actionable Tailscale DNS error** ([#83](https://github.com/scottkw/agenthub/issues/83)) — when a remote browse fails because the client has `accept-dns=false`, you get a specific actionable message ("Enable Tailscale DNS (accept-dns) to browse remote sessions") and a proactive warning banner, instead of an opaque 502.
- **Deterministic write release gate** ([#87](https://github.com/scottkw/agenthub/issues/87)) — `WriteFileAtomic` serializes same-path concurrent writers via a per-path lock for a true single-winner `If-Match` contract; the flaky `TestWrite_TwoWritersIfMatchRace` is now deterministic.
- **Relay-surface regression coverage** — the discover → pick → browse path is tested through the relay loopback the GUI actually uses, guarding against the v3.5-class blind spot where only the webserver/fixture surface was tested.

Closes [#86](https://github.com/scottkw/agenthub/issues/86), [#83](https://github.com/scottkw/agenthub/issues/83), [#87](https://github.com/scottkw/agenthub/issues/87), and retires umbrella epic [#24](https://github.com/scottkw/agenthub/issues/24). [Release notes](https://github.com/scottkw/agenthub/releases/tag/v3.5.1).

**v3.5 — File Browser: Write Operations & Editor** (2026-06-15) — create, edit, delete, rename, and upload files from any session, plus an in-app code editor, across desktop, web-share, and TUI.

- **Sandboxed write primitives** ([#63](https://github.com/scottkw/agenthub/issues/63)) — create / edit / delete / rename / upload scoped to each session's working directory on the same TOCTOU-safe `os.OpenRoot` sandbox as read, with atomic writes and fuzz-gated path-traversal coverage.
- **Per-session file-write capability (off by default)** — an opt-in `files.write` bit gates the write routes (CSRF-protected); web-share viewers opt in through an explicit two-gate consent model, never silently.
- **CodeMirror 6 editor** (desktop + web) — open, edit, and save with optimistic-concurrency conflict detection (`If-Match`/ETag), dirty markers, and Tab-to-indent.
- **TUI write parity** — `$EDITOR` shell-out (suspend/resume) plus delete / rename / mkdir keybindings.
- **Security hardening** — denylist, symlink / privilege-escalation / CSRF / concurrent-write coverage with a dedicated security audit.
- **Remote tailnet write data path** — the daemon-proxy write path across surfaces is implemented and **proven end-to-end live on two machines** (read + write). Includes fixes so the desktop GUI's relay loopback mounts the remote file routes and tailnet discovery recognizes capability-protected peers.
- **Clearer share-link scopes** — the Sessions-tab links now state that the View-Only link is session-only (no file access) and the Full Access link grants full session control *plus* file browsing.

**Known limitation at v3.5 (resolved in v3.5.1):** the desktop GUI remote-*browse* on-ramp — discovering and listing a peer's sessions to pick one — was deferred ([#86](https://github.com/scottkw/agenthub/issues/86)); at v3.5 remote file access required a join code / capability URL. This was completed in **v3.5.1** (see above).

Closes the write slice of [#63](https://github.com/scottkw/agenthub/issues/63). ([#24](https://github.com/scottkw/agenthub/issues/24) remote file browser was completed in v3.5.1.) [Release notes](https://github.com/scottkw/agenthub/releases/tag/v3.5).

**v3.4.2 — Dependency maintenance** (2026-06-14) — patch release rolling forward direct dependencies and the transitive bumps they pull in. Consolidates six Dependabot updates: `tailscale.com` 1.96.5 → 1.98.3, `golang.org/x/sys` 0.43.0 → 0.45.0, `golang.org/x/term` 0.42.0 → 0.43.0, `github.com/aymanbagabas/go-pty` 0.2.2 → 0.2.3, `github.com/Masterminds/semver/v3` 3.4.0 → 3.5.0, and the `actions/checkout` CI action 6.0.2 → 6.0.3. No user-visible behavior changes from v3.4.1. [Release notes](https://github.com/scottkw/agenthub/releases/tag/v3.4.2).

**v3.4.1 — Test reliability** (2026-06-14) — patch release. Deflakes a class of CI tests that asserted hub subscriber/viewer counts (and async goroutine state) after a fixed delay, racing the asynchronous subscription and intermittently failing the release gate. Adds a shared `WaitFor` poll-until-condition test helper. No user-visible behavior changes from v3.4. Closes [#80](https://github.com/scottkw/agenthub/issues/80). [Release notes](https://github.com/scottkw/agenthub/releases/tag/v3.4.1).

**v3.4 — File Browser (Read-Only) + TUI Parity** (2026-06-13) — browse, preview, and download files from any session across all three surfaces, including remote tailnet sessions.

- **Sandboxed read-only file browser** ([#62](https://github.com/scottkw/agenthub/issues/62)) — list, preview, and download files scoped to each session's working directory. TOCTOU-safe `os.OpenRoot`-based sandbox (Go 1.24+ kernel-level path resolution), 5 MB read cap with HTTP Range support, fuzz-gated against 40+ path-traversal payloads.
- **Capability-gated web-share access** — a new `files.read` capability bit gates `/api/files/*`; session owners hold it by default, web-share viewers do not (opt-in). Viewers without it get an explicit 403, never a silent failure.
- **Desktop + web-share UI** — React `FileBrowserTab` with single-pane file list + side-by-side preview, GFM markdown rendering, inline image previews, `/`-activated type-ahead filter, a breadcrumb bounded at the session root, and resizable list/preview columns.
- **TUI parity** ([#64](https://github.com/scottkw/agenthub/issues/64)) — full Files view in the Bubble Tea TUI: two-pane list + preview with glamour markdown, `/` filter, and async-only filesystem I/O that never blocks the render loop.
- **Remote-session file browse** — browse files on sessions hosted by other tailnet machines from desktop and TUI via a daemon proxy + join-code capability exchange; byte-equivalent results proven across Go and browser clients.
- **Post-milestone fixes** — Wails webview CORS for `/api/files/*`, clickable breadcrumb root, size/mtime population in the list, a native save dialog for desktop downloads, and resizable Size/Modified columns with persistence.

Closes [#62](https://github.com/scottkw/agenthub/issues/62) and the TUI-parity slice of [#64](https://github.com/scottkw/agenthub/issues/64).

**v3.3.2 — Dependency maintenance** (2026-05-19) — patch release rolling forward direct dependencies and transitive security-relevant bumps. No user-visible behavior changes from v3.3.1. [Release notes](https://github.com/scottkw/agenthub/releases/tag/v3.3.2).

**v3.3.1 — Bug Sweep** (2026-05-19) — patch release closing all known bugs in v3.3.

- **Windows daemon named-pipe IPC** ([#52](https://github.com/scottkw/agenthub/issues/52)) — daemon now listens on `\\.\pipe\agenthub-daemon` on Windows; GUI / CLI / TUI all work on a fresh Windows install. Credit: [@im-alexandre](https://github.com/im-alexandre) (PR #53).
- **OSC color-query + DA1 leak fixed on both relay surfaces** ([#54](https://github.com/scottkw/agenthub/issues/54), [#60](https://github.com/scottkw/agenthub/issues/60)) — `chafa --format=sixel`, `vim`, `neovim` (truecolor probe), `mc` and other capability-probing programs no longer leave junk on the shell prompt. Streaming `InputAbsorber` state machine filters OSC 10/11 + DA1 envelopes on both the web-share path and the daemon-direct (desktop / CLI attach) path.
- **WebGL recovery banner now renders** ([#55](https://github.com/scottkw/agenthub/issues/55)) — `WebGLRecoveryBanner` appears with 8s auto-dismiss when the WebGL context is lost; DOM fallback continues working.
- **iPad touch-scroll for terminal scrollback** ([#56](https://github.com/scottkw/agenthub/issues/56)) — single-finger drag scrolls xterm scrollback on iPad Safari + iPad Chrome; multi-touch still triggers native pinch-zoom; tap-on-link (OSC 8) preserved.
- **Linux shell auto-close** ([#57](https://github.com/scottkw/agenthub/issues/57)) — SHELL-12 auto-close (5s countdown + toast on agent exit) now works on Linux, matching macOS behavior. Cross-platform parity achieved via platform-specific PTY exit detector.
- **CI test stability** ([#58](https://github.com/scottkw/agenthub/issues/58)) — `TestPluginConfigStream_ExpiredCap_Returns401` deflaked; root cause was a base64-padding-bit no-op in the test-side capability mutation.
- **Plus** — pre-existing test debt repaired (`TestOpenCodeANSICapture` data race + 3 default-value tests), TUI defensive guard against zero-dimension panics, `agenthub attach` clears local terminal on entry, and a clarified bounded-lifetime contract on the `killSession` Wait goroutine.

Download v4.0: [Releases page](https://github.com/scottkw/agenthub/releases/tag/v4.0) — macOS DMG (signed + notarized), Windows installer / standalone, Linux deb / tar.gz.

## Features

### Hub (v4.0)
- **Session grid** — The top-level **Hub** shows every session as a live card (name, agent badge, colorblind-safe status, origin, viewer count, uptime, throttled mini terminal preview), auto-grouped by working directory with user-defined named groups (drag-to-assign, persisted)
- **Local + remote in one place** — Local sessions and remote tailnet-peer sessions render side by side in the same grid, with local/remote and connected/available indicators (icon + text, never color alone)
- **Filter, search, create** — A status filter bar with live counts (All / Working / Needs input / Complete / Error / Idle), `/`-focused search, and a New Session button — the Hub is the sole session-creation entry point
- **Attention surfacing** — Waiting/errored sessions float to the top of their group and pulse (motion + icon + position, reduced-motion aware), with the Hub mini-preview/briefing tail rendered through a headless VT emulator for accurate styled output
- **Card → modal** — Click a card to open an interactive terminal modal (or a briefing modal driven by the real terminal tail for attention sessions), including a cap-gated relay proxy for remote-session terminals
- **Per-card Share modal** — Two toggles per card: *Share the session* (reveals read-only + read/write capability links) and *Enable remote file browsing* (permission derives from the presented share code). A re-enableable warning fires before web-sharing a raw shell session

### Session Chat (v4.1)
- **Per-session thread** — A chat side channel attached to each session, reachable from the GUI session tab, the Hub interactive modal, and the web-share guest surface (one shared component → cross-surface parity by construction). Enter sends, Shift+Enter newlines; the composer auto-grows and renders Markdown safely (sanitized — no raw HTML or scripts); day separators stick to the top while scrolling
- **Identity, alias & presence** — Each participant shows a tailnet ID plus a self-chosen alias (set/change it from the chat sidebar), both visible to all; live connected/disconnected presence and debounced, never-stored typing indicators (server-side TTL clears them on abrupt disconnect)
- **`@session` agent bridge** — `@`-autocomplete over the session's participants plus a pinned `@session` target; `@session <text>` injects into the agent's PTY as a one-way prompt (reply stays in the terminal), gated to read/write holders, sanitized (C0/escape-sequence stripped, newlines collapsed), and guarded by a deliberate press-and-hold; a "→ injected into terminal" indicator marks injected messages
- **Read-only participants** — Read-only capability holders are full chat participants (post, set alias) while `@session` inject and raw PTY input remain read-only-gated server-side
- **Durability & export** — Daemon-persisted for the session's life (survives restart, full late-join scrollback, hard per-session message cap, deleted with the session); download any thread as Markdown from both surfaces
- **Notifications** — In-app unread badge on the chat toggle and the Hub session card; `@mentions` of your alias are visually distinct and highlighted in the stream
- **Resizable & polished** — The chat drawer is an overlay that never resizes the terminal; it's width-resizable via a left-edge drag handle (persisted, clamped) and truncates long author identities cleanly in the narrow panel

### Terminal & Sessions
- **Collapsible sidebar** — Left sidebar with Heroicons SVG icons for navigation: Home, Hub (top); Help, Settings (bottom). Toggle between collapsed (icons only, 48px) and expanded (icons + labels, 200px) via hamburger button; icons stay in fixed horizontal position during transitions; state persists in localStorage. *(v4.0: the separate Remote/Sessions pages and the New Session item were folded into the Hub.)*
- **Tabbed terminals** — Run multiple AI coding sessions side-by-side with full xterm.js terminals (ANSI 256-color, Unicode, emoji, 10K+ line scrollback, full-width viewport fill)
- **Background daemon** — Sessions live in a standalone daemon process; closing the GUI hides the window while sessions and the system tray remain active
- **CLI auto-detection** — Detects Claude Code, Codex, Gemini CLI, OpenCode, and Google Antigravity (`agy`) on startup — including when launched from Finder/Dock (augments PATH with `~/.local/bin`, Homebrew, nvm, Volta, snap, flatpak, cargo, pipx, and platform-specific install locations on macOS, Linux, and Windows); supports custom CLI path overrides. Google Antigravity is currently waitlist-gated; detection is fully wired and the agent appears in the picker when `agy` is installed.
- **New session modal** — Select a CLI and pick a working directory with a native folder browser; remembers your last-used directory
- **CLI argument passing** — Pass extra arguments to CLIs with `--` separator syntax (e.g., `agenthub new claude ~/dir -- --arg1`); arguments are remembered per CLI
- **Terminal theming** — 138 curated color themes (WCAG-audited for readability across all 4 CLIs); select in Settings > Appearance, applies live to all open sessions including OpenCode (via SIGUSR2 broadcast), persists across restarts with localStorage fallback guard for removed themes
- **Terminal padding** — 8px inset around terminal content with dynamic background matching the active theme
- **Per-tab font size** — Zoom the active terminal in/out with the standard macOS shortcuts `Shift-Cmd-+` / `Shift-Cmd--` (range 6–32px); the terminal refits cleanly after each change
- **Tab management** — Rename tabs by double-clicking or right-click context menu
- **Session auto-close** — When an agent process exits, its tab shows a 5-second countdown with an inline banner and fixed-position toast notification; tab auto-closes after countdown unless "Keep Open" is clicked; non-zero exits skip auto-close and show error state; toggle auto-close behavior in Settings > Session Behavior
- **Live status indicators** — Colored dots per tab: running (green), waiting (yellow), idle (gray), errored (red)
- **Standard app menus** — File, Edit, Window, Help menus with keyboard shortcuts; Cmd+C/V clipboard in terminal tabs
- **Welcome tab** — Branded splash screen with version info, platform-specific installation instructions, and getting-started guide

### Plugin Suite (v3.2)
- **8 curated xterm.js addons, vendored same-origin** — `addon-webgl`, `addon-unicode11`, `addon-search`, `addon-web-links`, `addon-image`, `addon-serialize`, `addon-clipboard`, `addon-progress`. All bundled under `web/vendor/xterm/addons/` with a CI-enforced version-parity gate (`vendor_drift_test.go`); no runtime CDN fetches; strict CSP-compatible
- **Settings → Plugins** — 8 enable/disable toggles in the order: WebGL, Unicode 11, Search, Web Links, Inline Images, Serialize, Clipboard, Progress. Three plugins (Search, Web Links, Inline Images) include inline `<details>` disclosures for per-plugin sub-config (regex/case/word defaults; modifier-key + risk policy; image storage limit). Non-hot-swappable plugins (Unicode 11, Inline Images) show "Applies to new sessions you create" inline plus a one-shot toast after Save
- **WebGL renderer** — GPU-accelerated rendering with automatic DOM fallback on context loss (banner toast with 8-second auto-dismiss) and proactive software-rasterizer detection (SwiftShader / llvmpipe / ANGLE-software / iPad Safari fall back at startup)
- **Scrollback search (Cmd-F)** — Focus-conditioned find bar on desktop + web with regex / case-sensitive / whole-word toggles, persisted defaults, next-match (Enter / Cmd-G), previous-match (Shift-Enter / Cmd-Shift-G), match count display, 200ms slide animation matching the BannerStack vocabulary, theme-aware highlight, and a 10,000-line scrollback perf budget
- **Web Links** — Clickable URLs with a strict scheme allowlist (`https`, `http`, `mailto` only — never `file://`, `javascript:`, or custom protocols). Cmd-click activation on macOS / Ctrl-click on Linux + Windows by default; single-click never activates a link. OSC 8 hyperlink href is shown in the hover tooltip. A risk-aware confirmation popover catches IDN/Punycode spoofs and a 30-entry typosquat list before navigation. Desktop routes through Wails `BrowserOpenURL`; web opens in a new tab with `_blank` + `noopener,noreferrer` (current-tab navigation is never possible)
- **Inline images (sixel)** — Render sixel-encoded images directly in the terminal via `@xterm/addon-image`. CSP includes a minimal, audited `'wasm-unsafe-eval'` carve-out (the addon's sixel decoder uses inline WASM). `storageLimit` defaults to 16 MB per-tab (overrides the upstream 100 MB default to prevent OOM with many concurrent tabs); configurable in the Settings disclosure. Multi-client byte-fidelity is preserved across the relay
- **Unicode 11 width tables** — Correct wide-character and emoji handling. Buffer-interpretation plugin: applies to new sessions only (toggling on a live session would re-flow existing scrollback); web-served clients share the daemon-side setting so multi-client scrollback wrap stays identical across viewers
- **Save Terminal As…** — Right-click a tab to save its current scrollback as a `.txt` file via a native `SaveFileDialog`. Backed by `@xterm/addon-serialize`; explicit secrets-warning tooltip; no auto-save / no on-disk capture without an explicit gesture
- **OSC 52 clipboard** — System clipboard read/write driven by OSC 52 escape sequences emitted by the running CLI. On web-served sessions, write access honors the capability bound to the session token — read-only viewers cannot have OSC 52 writes affect their clipboard via the terminal channel
- **OSC 9;4 progress (default OFF)** — Terminals emitting OSC 9;4 progress sequences (long-running task percent) surface as a subtle per-tab progress underline plus an aggregate tray quartile glyph (debounced at 200ms, atomic). Optional in v3.2; defaults flip ON in a future release after field validation
- **Daemon source of truth + hot-swap** — Plugin state lives in the daemon's `settings.json` under `PluginSettings` (with `schemaVersion: 2` migration from v3.1). Desktop receives a Wails `settings:plugins` runtime event for hot-swappable plugins; web clients subscribe to a capability-gated `/api/plugin-config` REST + SSE stream. No app restart needed
- **Cross-browser e2e CSP audit** — Playwright runs Chromium + Firefox + WebKit with zero CSP violations enforced in CI (`.github/workflows/e2e.yml`)

### Multi-Client Sessions
- **Simultaneous connections** — Multiple WebSocket clients can connect to the same session and receive live output simultaneously
- **Independent scrollback** — Each connected client maintains its own scrollback position without affecting other viewers
- **Read-only mode** — Attach with `--readonly` flag to observe a session without sending input (`agenthub attach --readonly <id>`)
- **Viewer count** — Session metadata API and CLI `agenthub list` show the current viewer count per session
- **Client identity** — Clients can provide a name at connection (e.g., `agenthub attach --client=macbook <id>`)
- **Resize arbitration** — Max-wins strategy: PTY dimensions stabilize to the largest active client, preventing resize thrashing

### Remote Sessions
- **Tailscale peer discovery** — Automatically discovers AgentHub instances running on other machines in your tailnet
- **Unified in the Hub (v4.0)** — Remote tailnet-peer sessions appear as cards in the Hub alongside local sessions (the separate Remote Sessions panel was removed in v4.0), with connected/available indicators
- **CLI remote list** — `agenthub list` shows local and remote sessions grouped by HOST column
- **CLI remote attach** — `agenthub attach hostname:session-id` connects to remote sessions via WSS relay over Tailscale HTTPS
- **Open in browser** — A remote Hub card's "Open in browser" opens the real peer URL, reusing the held web-share capability (no second join code)

### Auto-Update
- **Update checker** — Polls GitHub releases on startup and hourly for new versions
- **Notification banner** — In-app banner when an update is available with one-click download; multiple banners stack vertically with independent dismiss controls
- **Help menu trigger** — Manual check via Help > Check for Updates

### System Tray
- **Cross-platform tray** — macOS (native cgo NSStatusBar), Linux (D-Bus StatusNotifierItem for GNOME/KDE/XFCE), Windows (Shell_NotifyIcon Win32 API) — all sharing menu construction via common helpers
- **Menu bar icon** — Monochrome template icon adapts to light/dark mode (macOS); embedded PNG icon (Linux/Windows)
- **Session menu** — Dynamic menu listing all active sessions; click to activate
- **Session count tooltip** — Shows active session count (e.g., "AgentHub — 3 sessions")
- **Error state** — Tray icon switches to error state when daemon is unreachable
- **Quit confirmation** — Closing the window or selecting Quit from the tray menu presents a modal showing active session count with colored status dots and three options: Keep Running (dismiss), Quit GUI Only (hide window, send macOS notification, daemon stays running), or Quit Everything (stop daemon and exit)
- **Start minimized** — Optional "Start minimized to system tray" toggle in Settings > Behavior; when enabled, the app launches hidden with only the tray icon visible — preference persists across restarts
- **Dock hiding** — App hides from Dock and Cmd+Tab via LSUIElement (macOS)

### CLI
- **Full CLI** — `agenthub new`, `list`, `kill`, `rename`, `attach`, `web`, `health`, `qr`, `settings`
- **Interactive attach** — `agenthub attach <id>` for full PTY proxy with raw I/O, resize propagation, Ctrl-C passthrough, scrollback replay, and configurable detach key (default Ctrl-\\, set with `--detach-key=`)
- **Status bar** — Persistent tmux-style bottom bar during attach showing session name, agent type, hostname, detach hint, elapsed time, and live viewer count; refreshes without corrupting output (DECSTBM scroll region); suppressed when stdout is not a TTY; `--status-top` flag for top placement; clean teardown on detach
- **Connection banner** — Attach displays session name, CLI type, and hostname before entering raw mode
- **Machine-readable output** — `--json` flag on list, web status, health, and daemon status commands
- **Daemon management** — `agenthub daemon install/uninstall/start/stop` registers with platform service managers (launchd, systemd, Windows SCM)

### Shell sessions
AgentHub supports raw PTY shell sessions alongside AI CLI sessions (Claude Code, Codex, Gemini CLI, OpenCode). Both surfaces — the GUI new-session modal and the CLI `new shell` subcommand — expose this as exactly one entry labelled "Shell". There is no per-surface picker for the spawned binary; the daemon resolves it from a single Settings value.

- **GUI** — The New Session modal shows one static "Shell" row beneath the detected AI CLIs. Pressing Create launches a raw PTY using the binary configured in Settings → Paths.
- **CLI** — `agenthub new shell [<path>]`. No selection flag — the spawned binary is whatever Settings → Paths resolves to (the same value used by the GUI). The optional positional path sets the working directory; omit it to launch in `$HOME`. Extra tokens after `--` are NOT forwarded to shell sessions (a stderr warning is emitted if present, matching the GUI's no-args behavior).

**Binary selection.** Open the desktop app, go to Settings → Paths, and set the shell binary path. If no shellPath is configured, the daemon falls back to `$SHELL` (or the platform default — `zsh` on macOS, `bash` on Linux, `powershell.exe` on Windows).

**Cross-surface parity.** Both surfaces use the same shellPath value. Change it once in Settings → Paths and the new choice applies to every shell session you start, regardless of which surface launched it.

### Web Serving
- **Auto-serve** — Web server starts automatically on daemon launch; new sessions are web-served by default
- **Dual-mode networking** — Tailscale mode (Let's Encrypt TLS, zero-config security) when available; local network fallback (self-signed TLS + HTTP Basic Auth with generated password) when Tailscale is unavailable. Automatically upgrades from local to Tailscale mode when Tailscale connects after startup
- **Per-session toggle** — Enable/disable web access per session from GUI or CLI (`agenthub serve/unserve`)
- **Web dashboard** — Dark-themed dashboard with session cards, live status dots, CLI badges, QR code thumbnails, and direct connect links
- **Web terminal status bar** — Live session info with name, CLI type, hostname, and three-state connection indicator (connecting/connected/disconnected)
- **QR codes** — Every web-served session gets a scannable QR code in the desktop app and CLI
- **Health checks** — 4-state Tailscale health cascade (binary found → daemon running → connected → certs ready) across Homebrew, system package managers, Snap, Flatpak, and Windows default paths (`agenthub health` CLI command)
- **Nudge banner** — Context-aware in-app banner with 4-state detection: recommends Tailscale installation when binary not found; shows daemon-stopped instructions (platform-specific) when binary exists but daemon isn't running; shows "upgrading to Tailscale..." when Tailscale connects and the server is restarting; each banner is independently dismissible with fade-out animation

### Security
- **Capability-based session authorization** — Session listing, metadata, and WebSocket access require server-issued, HMAC-signed capability tokens bound to a specific session ID. Tailnet membership is no longer sufficient on its own; explicit grant is required to share each session. Cross-session capability use is rejected with 403.
- **Server-bound read-only enforcement** — Read-only is a property of the capability claims, not a client-supplied query string. Clients reconnecting without `?readonly` cannot bypass; the relay rejects `MsgInput` from read-only subscribers regardless of how the connection was opened.
- **No auto-expose** — Creating a new session while the web server is running does not automatically expose it. Sessions become reachable only after explicit grant via the GUI/CLI; the daemon then returns a signed capability-bearing URL.
- **Signing-key rotation panic button** — "Regenerate Signing Key" in Settings invalidates every outstanding capability across all sessions in one click; useful if a share link is suspected leaked.
- **Strict WebSocket Origin allowlist** — Cross-site WebSocket hijacking is blocked at the handshake. Allowlist covers Tailscale FQDN (`<host>.<tailnet>.ts.net:<port>`), local-mode bind host (`<lan-ip>:<port>`), and the Wails desktop webview (production `wails://wails`, dev `wails://wails.localhost:<port>`, Windows `http://wails.localhost`). All other origins return 403.
- **Vendored terminal assets + Content-Security-Policy** — xterm.js, addons, and themes are embedded in the binary at `web/vendor/xterm/`. Zero runtime CDN fetches. All three embedded HTML routes (`/dashboard`, `/join`, `/sessions/{id}`) enforce a strict CSP: `script-src 'self'`, `connect-src 'self' wss://<host>`, `style-src 'self' 'unsafe-inline'` (the last only to accommodate xterm's runtime style injection — see Phase 89 D-09).
- **Signed + notarized releases with SLSA L2 build provenance** — All third-party GitHub Actions are 40-character SHA-pinned (`scripts/grep-gate.sh` enforces this in CI). Build-tool versions (Wails, nfpm) pinned via `tools.go`. The release pipeline is split into `validate` → `build-{macos,windows,linux}` (no secrets) → `sign-macos` (gated by a required-reviewer rule, holds notarization credentials) → `publish`. Sigstore build-provenance attestation is verified BEFORE codesigning runs, so a compromised build job cannot inject a malicious binary into the signing flow.
- **Reproducibility** — `release.yml` and `distribute.yml` produce SHA256 checksums alongside artifacts; every published release includes a `checksums.txt`. Dependabot is configured for both `gomod` and `github-actions` ecosystems with no auto-merge — every dependency change goes through manual review.

### Settings
- **Settings as sidebar tab** — Persistent Settings tab accessible from the sidebar (not a modal), consistent with the Home / Hub / Help surfaces
- **Single scrollable page** — All settings on one page organized by section headers (Appearance, Web Server, Paths, Behavior, Session Behavior, Terminal Plugins) with visual dividers — no sub-tabs. *(v4.1: the "Plugins" jump link moved to last and was renamed "Terminal Plugins" — [#108](https://github.com/scottkw/agenthub/issues/108).)*
- **Terminal Plugins section** — 8 enable/disable toggles for the v3.2 plugin suite (WebGL, Unicode 11, Search, Web Links, Inline Images, Serialize, Clipboard, Progress) with per-plugin descriptions, "Applies to new sessions you create" inline captions for non-hot-swappable plugins, one-shot toast after Save, and inline `<details>` disclosures exposing per-plugin sub-config for Search (regex/case/word defaults), Web Links (modifier-key + risk-confirmation policy), and Inline Images (storage limit). Sub-key RPCs persist disclosure changes immediately — no "Save Plugins" required for sub-config
- **Appearance section** — A persisted **Light / Dark** UI theme toggle (v4.0 redesign) plus a terminal theme selector with 138 curated color schemes; the selected terminal theme applies live to all terminals and persists in localStorage
- **Web Server section** — Start/stop web server with mode-aware status display; URL actions (open in browser, copy to clipboard, inline QR code); local network password with click-to-copy
- **Behavior section** — "Start minimized to system tray" toggle with non-optimistic save, loading state, and error feedback
- **Session Behavior section** — "Auto-close tab on exit" toggle controls whether session tabs auto-close when the agent process exits; and a re-enableable "Shell web-share warning" toggle (v4.0) controls the one-time warning shown before web-sharing a raw shell session, firing consistently across both share surfaces; preferences persist via daemon settings
- **Paths section** — Override auto-detected CLI paths per agent; each path has a native browse button that opens a file picker; save confirmation shows a green "Saved!" indicator for 1.5 seconds
- **Tailscale status indicator** — 4-state color-coded dot (Connected / Not Connected / Daemon Stopped / Not Installed) with collapsible diagnostics checklist showing binary detection, daemon status, connection state, and TLS readiness; platform-specific troubleshooting instructions for macOS, Linux, and Windows
- **Certificate Transparency disclosure** — Acknowledgment flow for CT log requirements

### Platform
- **Cross-platform** — macOS (universal, signed + notarized), Linux (Ubuntu 22.04 + 24.04), Windows (NSIS installer)
- **Single binary** — `agenthub` launches GUI; `agenthub <command>` runs CLI
- **Custom app icons** — Branded AgentHub logomark across all platforms
- **Build script** — `build.sh` for local cross-platform builds with optional macOS code signing

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                         Clients                               │
│  ┌──────────────────┐  ┌────────────────────────────────┐   │
│  │  GUI (Wails)      │  │ CLI (agenthub <cmd>)           │   │
│  │  React+xterm      │  │ attach/list/new                │   │
│  └────────┬──────────┘  └──────────────┬─────────────────┘   │
│           │          DaemonClient       │                      │
│           └────────────┬───────────────┘                      │
├────────────────────────┼─────────────────────────────────────┤
│              Unix Socket / Named Pipe                         │
├──────────────────────┼────────────────────────────────────────┤
│  ┌───────────────────┴────────────────────────────────────┐  │
│  │                Daemon (background process)              │  │
│  │  ┌──────────┐  ┌──────────┐  ┌─────────────────────┐  │  │
│  │  │ Session  │  │ WebSocket │  │    Web Server        │  │  │
│  │  │ Engine   │  │ Relay Hub │  │  (Tailscale or      │  │  │
│  │  │ (go-pty) │  │ (fan-out, │  │   Local TLS)        │  │  │
│  │  │          │  │ multi-    │  │                     │  │  │
│  │  │          │  │ client)   │  │                     │  │  │
│  │  └──────────┘  └──────────┘  └─────────────────────┘  │  │
│  │  ┌──────────┐  ┌──────────┐  ┌─────────────────────┐  │  │
│  │  │  Status  │  │ QR Code  │  │   Service Manager   │  │  │
│  │  │ Detector │  │ Generator│  │                     │  │  │
│  │  └──────────┘  └──────────┘  └─────────────────────┘  │  │
│  │  ┌──────────┐  ┌──────────┐                            │  │
│  │  │ Tailnet  │  │ Update   │                            │  │
│  │  │ Peers    │  │ Checker  │                            │  │
│  │  └──────────┘  └──────────┘                            │  │
│  └────────────────────────────────────────────────────────┘  │
│                      HTTP/JSON API                            │
└──────────────────────────────────────────────────────────────┘
```

**Go packages:**

| Package | Purpose |
|---------|---------|
| `internal/daemon` | Session engine, HTTP/JSON API, Unix socket server, DaemonClient |
| `internal/pty` | PTY process management, CLI detection |
| `internal/relay` | Binary framing protocol, scrollback buffer, WebSocket fan-out hub with multi-client support, per-subscriber metadata, read-only enforcement, max-wins resize arbitration |
| `internal/status` | Heuristic status detection (running/waiting/idle/errored) |
| `internal/statusbar` | DECSTBM scroll-region status bar for CLI attach with rune-safe formatting, viewer count, connection state, terminal injection prevention |
| `internal/attach` | Shared attach logic for CLI — ANSI-safe border-title injection, allowlist attach-status guard, error-propagating AttachSession |
| `internal/tailnet` | Tailscale peer discovery, concurrent probe pool, cached peer list |
| `internal/updater` | GitHub release polling, semantic version comparison, update notifications |
| `internal/webserver` | HTTPS server (Tailscale or local self-signed TLS), dashboard, health checks, Basic Auth |
| `web/` | Embedded HTML assets (dashboard + terminal pages) |

**Frontend (`frontend/`):**

| Component | Purpose |
|-----------|---------|
| `App.tsx` | Root layout, daemon client, session management, sidebar + content flex layout |
| `Sidebar.tsx` | Collapsible navigation sidebar with Heroicons: Home, Hub, Help, Settings |
| `Hub/HubPanel.tsx` | Hub session grid — cards, working-directory + named groups, status filter bar, search, attention surfacing, card→modal, per-card Share modal |
| `Hub/SessionShareModal.tsx` | Per-card Share modal: session-share toggle (RO/RW capability links) + remote file-browse toggle |
| `TabBar.tsx` | Tab strip with status dots, rename, close, overflow chevron (session tabs only — no action buttons) |
| `TerminalPanel.tsx` | xterm.js terminal with WebSocket relay, per-tab font size, theme support |
| `NewSessionModal.tsx` | CLI selector, working directory picker, argument input |
| `HelpTab.tsx` | In-app Help page — searchable docs/FAQ, section nav, external links |
| `WelcomeTab.tsx` | Branded welcome screen with installation instructions |
| `StatusBar.tsx` | Per-tab web-serving controls |
| `ExitToast.tsx` | Fixed-position toast notification for session exits — clean/error variants, countdown display, Keep Open and dismiss buttons |
| `ExitCountdownBanner.tsx` | Inline countdown banner in terminal area — "Agent exited cleanly. Tab closes in Ns." with Keep Open button |
| `QuitConfirmModal.tsx` | Quit confirmation modal — session list with colored status dots, three exit options (Keep Running, Quit GUI Only, Quit Everything) |
| `SettingsTab.tsx` | Settings as sidebar tab: single scrollable page with section headers (Plugins, Behavior, Session Behavior, Appearance, Web Server, Paths); start-minimized toggle, auto-close toggle, shell web-share warning toggle, Light/Dark theme toggle + terminal theme selector, web server controls with URL actions (open/copy/QR), CLI path overrides with native browse buttons, local network password |
| `LocalNetworkBanner.tsx` | 4-state context-aware nudge banner with independent dismiss: not-installed, daemon-stopped, not-connected, and upgrade-in-progress states with platform-specific instructions |
| `UpdateBanner.tsx` | Standalone update notification banner with version info, download button, and dismiss control |
| `QRModal.tsx` | QR code display for web-served sessions |

## Installation

### macOS (Homebrew)

```bash
brew tap scottkw/agenthub
brew install --cask agenthub
```

### Windows (WinGet)

```powershell
winget install scottkw.agenthub
```

> WinGet availability depends on the first submission being accepted by Microsoft. Check [Releases](https://github.com/scottkw/agenthub/releases) for direct download in the meantime.

### GitHub Releases

Download the latest release for your platform from [Releases](https://github.com/scottkw/agenthub/releases):

| Platform | File | Notes |
|----------|------|-------|
| macOS (universal) | `agenthub-v*-darwin-universal.dmg` | Signed and notarized |
| Windows | `agenthub-v*-windows-amd64-installer.exe` | NSIS installer |
| Windows | `agenthub-v*-windows-amd64.exe` | Standalone executable |
| Linux (deb) | `agenthub-v*-linux-amd64.deb` | Ubuntu/Debian package |
| Linux (tar.gz) | `agenthub-v*-linux-amd64.tar.gz` | Generic archive |

All releases include a `checksums.txt` with SHA256 hashes for verification.

## Prerequisites

- **Go** 1.22+ ([go.dev/dl](https://go.dev/dl/))
- **Node.js** 18+ and **pnpm** ([pnpm.io](https://pnpm.io/installation))
- **Wails CLI** v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- **Tailscale** (optional) — enables zero-config web serving with browser-trusted TLS; without it, local network mode uses self-signed TLS + password auth ([tailscale.com](https://tailscale.com))

### Platform-specific

**macOS:**
- Xcode Command Line Tools: `xcode-select --install`

**Linux (Ubuntu/Debian):**
```bash
# Ubuntu 24.04 / Debian 13+
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev

# Ubuntu 22.04
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev
```

**Windows:**
- [WebView2 runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (included in Windows 11; install manually on Windows 10)
- A C compiler — MSYS2 with MinGW-w64 or TDM-GCC

## Development

```bash
# Clone
git clone https://github.com/scottkw/agenthub.git
cd agenthub

# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Run in dev mode (hot-reload frontend, live Go rebuild)
wails dev
```

Dev mode opens the desktop app with Vite HMR for the frontend and automatic Go rebuild on save.

### Running tests

```bash
# Go tests (with race detector)
go test -race ./...

# Frontend tests
cd frontend && pnpm test
```

## Building

### Using `build.sh` (recommended)

```bash
# Build for the current platform
./build.sh --platform macos    # macOS universal binary (.app)
./build.sh --platform linux    # Linux amd64 via Docker
./build.sh --platform windows  # Windows amd64 via cross-compile

# Build all platforms
./build.sh --all

# Build + sign and notarize macOS (requires Apple Developer credentials)
./build.sh --platform macos --sign
```

### Manual builds

#### Local build (current platform)

```bash
wails build -tags wailsassets
```

Output: `build/bin/agenthub` (or `agenthub.exe` on Windows, `agenthub.app` on macOS)

#### macOS (universal binary)

```bash
wails build -platform darwin/universal -tags wailsassets
```

#### Linux

```bash
# Ubuntu 24.04 (WebKitGTK 4.1)
wails build -tags webkit2_41,wailsassets

# Ubuntu 22.04 (WebKitGTK 4.0)
wails build -tags wailsassets
```

#### Windows

```bash
# Standard build
wails build -tags wailsassets

# With NSIS installer
wails build -nsis -tags wailsassets
```

### CI/CD

GitHub Actions automates building, releasing, and distributing AgentHub:

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `build.yml` | Push/PR | 4-runner matrix build with race detector (no signing) |
| `release.yml` | Tag push (v*) | Multi-platform release builds with macOS signing/notarization (gated by required-reviewer on the `release` environment) |
| `distribute.yml` | Release published | Updates Homebrew tap + submits WinGet manifest PR |

**Build matrix:**

| Runner | Platform | Notes |
|--------|----------|-------|
| `macos-latest` | `darwin/universal` | Signing + notarization in release.yml |
| `ubuntu-latest` | `linux/amd64` | WebKitGTK 4.1 (`-tags webkit2_41`) |
| `ubuntu-22.04` | `linux/amd64` | WebKitGTK 4.0 |
| `windows-latest` | `windows/amd64` | NSIS installer + WebView2 embedded |

## Usage

### Desktop (GUI)

1. **Launch AgentHub** — run `agenthub` with no arguments to open the GUI
2. **Navigate via sidebar** — Home / Hub / Help / Settings; toggle collapsed/expanded with the hamburger icon
3. **Open the Hub** — the Hub shows every local and remote session as a live card, grouped by working directory; filter by status, search with `/`, and organize into named groups
4. **Create a session** — click New Session on the Hub to open the new session modal; select a CLI, working directory, and optional arguments
5. **Use the terminal** — click a card to open the interactive terminal modal (or a briefing modal for sessions needing input); new sessions are automatically web-served
6. **Manage & share** — from a Hub card, kill the session or open its Share modal to reveal read-only / read-write links and enable remote file browsing
7. **Remote sessions** — sessions on other tailnet machines appear as Hub cards; "Open in browser" opens the real peer URL
8. **System tray** — close the window to hide; use the tray menu to switch sessions or quit

### CLI

```bash
# Session management
agenthub new claude-code ~/project           # Create a new session
agenthub new claude-code ~/project -- --arg  # Pass extra CLI arguments
agenthub list                                # List all sessions
agenthub list --json                         # Machine-readable output
agenthub attach <id>                         # Attach to session (Ctrl-\ to detach)
agenthub attach hostname:<id>                # Attach to remote session via Tailscale
agenthub kill <id>                           # Terminate a session
agenthub rename <id> "my session"            # Rename a session
agenthub attach --readonly <id>              # Read-only attach (observe without input)
agenthub attach --client=macbook <id>        # Attach with client identity name

# Web serving
agenthub web start                    # Start the Tailscale web server
agenthub web stop                     # Stop the web server
agenthub web status                   # Check web server state
agenthub serve <id>                   # Enable web access for a session
agenthub unserve <id>                 # Disable web access
agenthub health                       # Tailscale health check
agenthub qr <id>                      # Show session QR code in terminal

# Daemon management
agenthub daemon install               # Register as login service
agenthub daemon uninstall             # Remove service registration
agenthub daemon start                 # Start the daemon service
agenthub daemon stop                  # Stop the daemon service
agenthub daemon status                # Check daemon status

# Configuration
agenthub settings                     # Show current settings
```

### Status indicators

Each tab shows a colored status dot:

| Color | Status | Meaning |
|-------|--------|---------|
| Green | Running | CLI is actively producing output |
| Yellow | Waiting | CLI has shown a prompt and is waiting for input |
| Gray | Idle | No recent output activity |
| Red | Errored | CLI process has exited with a non-zero code |

Status detection uses heuristic output patterns for **Claude Code** and **Google Antigravity** (`agy`). Other CLIs show "running" until their patterns are catalogued.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Desktop framework | [Wails v2](https://wails.io) |
| Backend | Go 1.22+ |
| Frontend | React 19, TypeScript, Vite |
| Terminal | [xterm.js](https://xtermjs.org) v6 + vendored addons (`addon-webgl`, `addon-unicode11`, `addon-search`, `addon-web-links`, `addon-image`, `addon-serialize`, `addon-clipboard`, `addon-progress`) |
| Terminal themes | [xterm-theme](https://www.npmjs.com/package/xterm-theme) — 138 curated schemes (WCAG-audited from 157 candidates) |
| PTY | [go-pty](https://github.com/aymanbagabas/go-pty) (cross-platform) |
| WebSocket | [nhooyr/websocket](https://github.com/coder/websocket) |
| QR codes | [go-qrcode](https://github.com/skip2/go-qrcode) |
| TLS | Tailscale Let's Encrypt via `GetCertificate`; self-signed P256 for local network mode |
| Peer discovery | [tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) |
| Auto-update | [go-selfupdate](https://github.com/creativeprojects/go-selfupdate), [Masterminds/semver](https://github.com/Masterminds/semver) |
| Service manager | [kardianos/service](https://github.com/kardianos/service) |
| Cross-browser e2e | [Playwright](https://playwright.dev) (Chromium + Firefox + WebKit) — CSP zero-violation gate in CI |
| CI | GitHub Actions (4-runner matrix + Playwright e2e) |

## License

See [LICENSE](LICENSE) for details.
