# Feature Research

**Domain:** Terminal multiplexer desktop app with web serving for AI coding CLIs
**Researched:** 2026-03-20 (v1.2 Tailscale-only networking added; v1.1 and v1.0 preserved below)
**Confidence:** HIGH (Tailscale Go API verified via pkg.go.dev; existing codebase read directly)

---

## v1.2 Milestone: Tailscale-Only Networking

### Scope

This section covers only what is NEW in v1.2. Existing app ships: tabbed terminal sessions,
web serving with self-signed TLS, password/token auth, QR codes, VPN interface binding
(generic), and health status indicators. Research focus: Tailscale health checks, Let's
Encrypt cert provisioning via Tailscale, Tailscale-only networking, removal of password/token
auth and self-signed CA.

---

### Table Stakes (Users Expect These for v1.2)

Features that must work correctly for the milestone to feel complete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Tailscale health check: installed / daemon reachable | App must know if `tailscaled` is reachable before attempting to serve; raw TLS errors are not acceptable failure output | LOW | `local.Client.StatusWithoutPeers(ctx)` — if it returns an error, tailscaled is not running or Tailscale is not installed. These two states must be distinguished: binary absent vs. socket missing vs. daemon not running. |
| Tailscale health check: connected to tailnet | App must verify `BackendState == "Running"` before binding to Tailscale IP or requesting certs | LOW | `ipnstate.Status.BackendState` field; values are "NoState", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting", "Running". Only "Running" is safe to proceed. Show state-specific instructions for each non-Running state. |
| Tailscale health check: HTTPS certs enabled | `GetCertificate` will fail if HTTPS certs are not enabled in the tailnet admin console — this is a user action required in a web UI, not something the app can do | MEDIUM | `ipnstate.Status.CertDomains` — non-empty means certs are provisioned for this node. Empty means the user has not enabled HTTPS in their tailnet DNS settings (two sub-steps: enable MagicDNS, then enable HTTPS certificates). |
| Instructional modal for health check failures | Users who fail health checks need actionable steps, not raw errors | MEDIUM | Three distinct failure states need distinct instructions: (1) not installed — link to tailscale.com/download; (2) not connected — instructions to open Tailscale and connect; (3) certs not enabled — exact path in admin console (DNS page → HTTPS Certificates → Enable HTTPS). Modal must include a "Check Again" button that re-runs health checks without restarting the app. |
| Let's Encrypt cert via `local.Client.GetCertificate` | Replaces self-signed CA+leaf pattern; provides browser-trusted HTTPS on the `<hostname>.<tailnet>.ts.net` domain with automatic renewal | MEDIUM | `GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)` is the correct `tls.Config.GetCertificate` callback signature. Set it on the TLS config and tailscaled handles caching and renewal automatically. Stable API per pkg.go.dev. FQDN is `Status.Self.DNSName` (trim trailing dot). |
| Bind exclusively to Tailscale interface IP | Web server must only listen on the Tailscale CGNAT address (100.64.0.0/10), not 0.0.0.0, since auth is being removed | LOW | Existing `IsTailscaleIP()` and `ListInterfaces()` already detect Tailscale IPs. Auto-select the first interface where `IsTailscaleIP` is true; remove the user-facing interface picker dropdown. This binding is what makes auth removal safe. |
| Remove password auth and session tokens | When Tailscale is the trust boundary, password/token auth is redundant friction; deleting it simplifies the codebase materially | MEDIUM | Delete `auth.go`, `tokens.go`, all middleware (`dashboardAuth`, `sessionAuth`). Routes become open. The Tailscale network layer (node identity + ACLs) is the access control. Only web-enabled sessions are served, as before. |
| Remove self-signed CA infrastructure | Self-signed certs are incompatible with the Let's Encrypt model; keeping both creates dead code and confusing failure modes | LOW | Delete `tls.go` (CA generation, leaf cert generation). `NewWebServer` no longer calls `LoadOrCreateCA`. `Config.ConfigDir` field can be removed (was only needed for CA persistence). |
| Remove generic VPN interface selection UI | With Tailscale-only networking, there is no interface choice to present; the dropdown adds confusion | LOW | Frontend settings panel currently exposes an interface picker. Replace with a Tailscale status indicator showing connected IP, hostname, and health state. |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Per-OS instructional text in health check modal | Most tools show a raw error string; platform-specific guidance ("Open the Tailscale menu bar app" vs. "Run `sudo tailscale up`") is materially better UX | MEDIUM | Detect at runtime via `runtime.GOOS`. macOS: "Open the Tailscale menu bar app and click Connect". Linux: "Run `sudo tailscale up` in a terminal". Windows: "Open Tailscale from the system tray". Pass the OS hint to the React modal via Wails binding. |
| Auto-derive HTTPS URL from tailnet hostname | User never has to configure a domain; the app reads `Status.Self.DNSName` and constructs the shareable URL automatically | LOW | `DNSName` has a trailing dot (e.g. `hostname.tail12345.ts.net.`); trim it. Construct `https://<DNSName>:<port>`. Display as the shareable URL and use for QR code generation. This is a better URL than the raw Tailscale IP. |
| Health check on web-serve toggle | Re-run health checks at the moment the user enables web serving, not just at startup | LOW | Prevents confusing TLS errors if the user starts the app with Tailscale connected, then Tailscale disconnects, then they try to enable a session. Return to health check modal state rather than serving a broken TLS endpoint. |
| Certificate renewal handled transparently | Users never see cert expiry; no manual `tailscale cert` command needed | LOW | This is free behavior from `local.Client.GetCertificate` — tailscaled handles caching and auto-renewal. Explicitly better than the previous self-signed CA which required users to manually install a CA cert in their browser. Worth calling out in UI copy. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Fallback to self-signed certs when Tailscale certs fail | "Keep serving even if cert provisioning fails" | Silently serves untrusted content; browser shows cert warning that looks like an app bug; creates two parallel TLS code paths to maintain | Fail clearly with the health check modal. If certs are unavailable, web serving is disabled with a clear actionable message. No silent fallbacks. |
| "Try anyway" option to bypass health checks | Power users want to skip setup steps | Leads to confusing TLS errors that erode trust in the app's reliability; the three-step health check process is fast and should not be bypassable | Keep the "Check Again" button. If the check passes, proceed immediately. No bypass mode. |
| Keep password auth as optional extra security | "Tailscale + password = more secure" | Adds complexity to an auth model being intentionally simplified; Tailscale ACLs are the correct tool for fine-grained access control on a tailnet | Tailscale network-level trust is the boundary for v1.2. If per-user access control is needed, that is a future milestone with explicit scope. |
| Support non-Tailscale VPN interfaces alongside Tailscale | Preserves v1.0 behavior for non-Tailscale users | Splits the trust model; makes health checks ambiguous; doubles TLS cert management surface area; the generic VPN path used self-signed certs which are now being removed | v1.2 is explicitly Tailscale-only. The generic VPN path is removed. Document this as a breaking change for non-Tailscale users in release notes. |
| Tailscale Funnel integration for public internet access | Allow sessions to be publicly accessible via Tailscale Funnel | Funnel exposes services to the public internet, which is a different threat model from tailnet-only; mixing it with the trust-boundary-is-tailscale model creates security confusion | Defer to a future milestone with explicit scope definition and separate UX for "public" vs. "tailnet-only". |

---

### Feature Dependencies (v1.2)

```
[Tailscale installed / daemon reachable]
    └──required by──> [Tailscale connected check]
                          └──required by──> [HTTPS certs enabled check]
                                                └──required by──> [GetCertificate TLS config]
                                                                      └──required by──> [Bind to Tailscale IP]
                                                                                            └──required by──> [Remove password/token auth (safe)]

[Instructional modal]
    └──depends on──> [All three health check states] (must surface distinct guidance per failure)

[Auto-derive URL from DNSName]
    └──requires──> [Tailscale connected check] (DNSName is only populated in Status.Self when BackendState == "Running")

[Remove self-signed CA] ──safe only when──> [GetCertificate TLS config] (no other TLS source)
[Remove auth middleware] ──safe only when──> [Bind to Tailscale IP] (binding to 0.0.0.0 with no auth is a regression)
[Remove interface picker UI] ──replaces with──> [Tailscale status indicator]
```

#### Dependency Notes

- **Health checks gate web server startup:** A healthy Tailscale state is required before binding to the interface IP. The health check chain is sequential: installed → connected → certs enabled. Each must pass before the next is checked.
- **Auth removal is safe only after interface binding:** Removing password auth while still binding to 0.0.0.0 would expose sessions on the local network. The Tailscale-interface-only binding is the prerequisite that makes auth removal safe.
- **Self-signed CA removal is safe only after GetCertificate hook is tested:** If the GetCertificate callback has a bug, there is no fallback — the server won't start. Confirm it works end-to-end before deleting tls.go.

---

### MVP Definition (v1.2)

#### Launch With

- [x] Health check: installed, connected, certs enabled — core safety gate
- [x] Instructional modal with per-OS text and Check Again button
- [x] Let's Encrypt cert via `local.Client.GetCertificate`
- [x] Tailscale-only interface binding
- [x] Remove password/token auth and self-signed CA infrastructure

#### Defer

- Tailscale Funnel integration — different threat model, separate milestone
- Per-user Tailscale ACL configuration UI — out of scope for v1.2

---

### Feature Prioritization Matrix (v1.2)

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Health check chain | HIGH | LOW | P1 |
| Instructional modal | HIGH | MEDIUM | P1 |
| GetCertificate TLS hook | HIGH | MEDIUM | P1 |
| Tailscale-only binding | HIGH | LOW | P1 |
| Remove auth + self-signed CA | MEDIUM | MEDIUM | P1 (cleanup enables simpler codebase) |
| Per-OS instructions | MEDIUM | LOW | P1 |
| CT disclosure modal | MEDIUM | LOW | P2 |

---

## v1.3 Milestone: CLI + Daemon

### Scope

This section covers only what is NEW in v1.3. Existing app ships all v1.2 features. The core
architectural change: session management moves from the GUI process into a persistent background
daemon. The GUI and a new CLI become two equal client types that attach to the same daemon session
pool. Research basis: tmux/screen patterns, shpool architecture, kardianos/service for cross-platform
service registration, Go Unix domain socket IPC, and PTY raw mode / resize passthrough.

---

### Table Stakes (Users Expect These for v1.3)

Features that must work for the milestone to feel complete. Grounded in what tmux, screen, shpool,
and Docker daemon users consider non-negotiable in any session manager.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| `agenthub new [agent] [path]` | Any session manager must support creating sessions from CLI without a GUI | LOW | Forwards to daemon over Unix socket. Returns session ID. Prints session name on success. Should accept `--name` flag for custom naming. Mirrors GUI new-session modal. |
| `agenthub list` (alias `ls`) | Sessions must be enumerable; core workflow is "what is running?" | LOW | Returns table: ID, name, agent, working dir, status, web URL if serving. Machine-readable `--json` flag useful for scripting. |
| `agenthub attach <id-or-name>` | Session persistence is useless without a way to reconnect | HIGH | Enters raw PTY proxy mode: stdin → daemon socket → PTY; PTY output → stdout. Must handle resize (SIGWINCH → send resize event to daemon). Must restore terminal state on exit. This is the hardest feature in the milestone. |
| `agenthub detach` | Attach without detach is a trap; user must be able to leave session running | LOW | Send detach message to daemon via the attach connection. Configurable prefix key (default `ctrl+b d`, matching tmux convention). Restores normal terminal mode. Session keeps running in daemon. |
| `agenthub kill <id-or-name>` | Sessions must be terminable from CLI | LOW | Sends SIGTERM to the session's child process via daemon. Daemon removes session from pool. Confirm with a `--force` flag for SIGKILL. |
| `agenthub rename <id-or-name> <new-name>` | Named sessions are table stakes; renaming is expected | LOW | Updates session name in daemon registry. Propagates to GUI (GUI polls or receives push notification). |
| Background daemon process | The whole point: sessions must survive GUI close and terminal disconnect | MEDIUM | Single daemon per user, not per session. Listens on Unix socket at well-known path (e.g. `$XDG_RUNTIME_DIR/agenthub.sock` on Linux, `$TMPDIR/agenthub.sock` on macOS). Auto-started by CLI if not running. |
| Daemon auto-start on login | Users expect their session manager to be there when they open a terminal | MEDIUM | Platform service registration: launchd plist on macOS, systemd user unit on Linux, Windows Service on Windows. `kardianos/service` package provides a unified Go API across all three. Registered on first `agenthub daemon install`. |
| `agenthub daemon start/stop/status` | Service lifecycle commands are expected in any daemon-managed tool (Docker, Redis, etc.) | LOW | Wraps platform service manager commands. `status` reports whether daemon is running, its PID, and socket path. |
| Shared session pool: GUI + CLI see same sessions | If GUI and CLI show different sessions, the product is broken | HIGH | Daemon owns all sessions. GUI connects to daemon over Unix socket (or migrates to HTTP IPC — see architecture). GUI's current in-process session store becomes a client view of daemon state. This is the largest architectural change in v1.3. |
| Graceful terminal state restore on detach/exit | Raw PTY mode changes terminal settings; failure to restore leaves terminal broken | MEDIUM | Call `term.Restore(fd, oldState)` on any exit path: normal detach, ctrl+c, daemon disconnect, signal. Must use `defer` and handle panics. This is the classic footgun in PTY tools. |
| `agenthub web start/stop/status <id>` | Per-session web serving is an existing GUI feature that must be reachable from CLI | LOW | Maps to existing web server toggle. `status` prints URL + QR text if serving. Existing webserver package already handles this; CLI just invokes it via daemon RPC. |
| `agenthub health` | CLI should surface the same Tailscale health state the GUI shows | LOW | Calls existing health check logic. Returns structured status: installed, connected, certs. Exits non-zero on failure for scripting. |
| `agenthub qr <id>` | QR code sharing is a core AgentHub feature; must be CLI-accessible | LOW | Prints QR code as Unicode block characters (existing `skip2/go-qrcode` can output to terminal). Falls back to printing URL if session is not web-served. |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Configurable detach prefix key | tmux's C-b default conflicts with many editors and terminal apps; configurable prefix is a known quality-of-life win | LOW | Stored in config file. Default: `ctrl+b` then `d` (tmux convention). Alternative suggestion: `ctrl+]` (no conflicts, used by shpool). Applied in CLI attach handler before bytes are forwarded to daemon. |
| GUI and CLI are interchangeable session clients | Most session managers treat GUI as primary and CLI as secondary; making them truly equal enables headless server use cases | HIGH | Requires daemon to own all session state. GUI becomes a display client, not the session owner. This is architecturally ambitious but enables: SSH into machine, `agenthub list`, attach to session the GUI was managing. |
| `agenthub serve <path>` / `agenthub unserve <id>` | Shorthand for new-session + web-start in one command; enables one-line "serve this project" workflow | LOW | Sugar over `new` + `web start`. Prints URL and QR immediately. Common enough workflow to deserve first-class support. |
| `agenthub settings` | CLI settings inspection (read-only) closes the gap between GUI and CLI discoverability | LOW | Prints current settings as JSON or table. Not a full TUI settings editor — just inspection. GUI remains the write interface. |
| `--json` output on all list/status commands | Enables scripting and automation (piping to jq, CI integration) | LOW | Standard in modern CLIs (gh, kubectl, docker). Use `encoding/json`. Add to `list`, `status`, `health`, `web status`. |
| Scrollback replay on attach | When reattaching to a session, show recent output so the user knows where they left off | MEDIUM | Existing `relay/scrollback.go` already implements in-memory VT100 scrollback buffer. On attach, daemon sends the scrollback buffer before entering live relay mode. Shpool uses the same pattern — confirmed as a user expectation. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Full TUI session manager (tmux-style pane layout) | "Why not make it like tmux?" | Out of scope for v1.3; AgentHub tabs are managed in the GUI, not via a TUI pane system; adding pane layout to the CLI would be a parallel product, not an enhancement | Each AI agent gets one session, one tab. The GUI is the multiplexer. The CLI is the automation interface. |
| Multiple clients attached to one session simultaneously | Collaborative terminal sharing (tmux's killer feature) | Complex to implement correctly (output fan-out, input arbitration, resize conflicts); not a stated requirement; adds protocol complexity without a clear AgentHub use case | One active attach at a time. The GUI can observe sessions (web view is already the read-only path). |
| Daemon per session (zmx model) | Isolates crashes — if one session's daemon crashes, others survive | Adds process management overhead; complicates the shared session pool (each GUI tab would need to track multiple daemon PIDs); cross-session commands (`list`, `rename`) require a coordinator anyway | Single daemon model (shpool/tmux approach). Make the daemon resilient rather than eliminating it. |
| Automatic session naming from git branch or cwd | "Name sessions after my project automatically" | Heuristic-based naming causes collisions and surprises; users immediately start overriding it | Require explicit `--name` or default to `agent-<id>`. Support `agenthub rename` for post-hoc naming. |
| Session recording/replay (full asciinema-style) | "Record my Claude Code sessions" | Significant storage and privacy concerns; orthogonal to session management; better served by a dedicated tool | Scrollback buffer for reattach is sufficient. Full recording is out of scope. |
| Remote daemon (TCP socket) | Access sessions over the network without SSH | Security model complexity (auth, TLS); the existing web-serving feature covers the read-only remote access case; the GUI covers local management | Web serving is the remote access path. CLI is local-only (Unix socket). SSH + `agenthub attach` covers advanced cases. |
| Interactive TUI for `agenthub list` | "Show a fzf-style picker" | Adds a TUI dependency (bubbletea/lipgloss); the simple table output is faster and more scriptable; the GUI is the correct rich-UI surface | Plain table output with `--json` flag for scripting. Pipe to `fzf` externally if desired. |

---

### Feature Dependencies (v1.3)

```
[Background daemon process]
    └──required by──> [Unix socket IPC]
                          └──required by──> [agenthub new/list/kill/rename/web/health/qr]
                                                └──required by──> [agenthub attach]
                                                                      └──required by──> [agenthub detach]

[Shared session pool: GUI + CLI]
    └──requires──> [Daemon owns all sessions] (GUI migrates from in-process store to daemon client)
                       └──required by──> [GUI connects to daemon socket]

[agenthub attach]
    └──requires──> [PTY proxy: raw I/O relay]
    └──requires──> [Terminal state save/restore]
    └──requires──> [Resize event forwarding (SIGWINCH)]
    └──enhanced by──> [Scrollback replay on attach]

[Daemon auto-start on login]
    └──requires──> [agenthub daemon install] (registers with platform service manager)
    └──uses──> [kardianos/service] for launchd/systemd/Windows Service abstraction

[agenthub serve <path>]
    └──sugar over──> [agenthub new] + [agenthub web start]
    └──requires──> [agenthub new] and [agenthub web start/stop/status]

[Configurable detach prefix key]
    └──requires──> [agenthub attach] (prefix key is parsed during attach raw mode)
    └──stored in──> [config file] (existing config system)

[Scrollback replay on attach]
    └──already implemented──> [relay/scrollback.go] (in-memory VT100 buffer exists)
    └──requires──> [Daemon sends scrollback on attach handshake]
```

### Dependency Notes

- **Daemon is the prerequisite for everything:** All CLI commands route through the Unix socket to the daemon. The daemon must exist before any CLI command is useful. Auto-start logic (start daemon if not running, then run the command) must be rock-solid.
- **Shared session pool requires architectural migration:** The GUI currently owns sessions in-process. Moving them to the daemon means the GUI must become a client. This is the highest-risk change in v1.3 and should be the first phase, with all other CLI commands built on top.
- **PTY attach is the hardest feature:** Raw terminal mode, SIGWINCH handling, terminal state restore, and prefix key parsing must all work correctly. A single missed `defer term.Restore()` leaves the user's terminal broken. Treat this as a mini-project within the milestone.
- **Scrollback replay is low-cost:** `relay/scrollback.go` already exists. The daemon just needs to send the buffer at the start of an attach session. This is a differentiator that costs almost nothing given the existing infrastructure.
- **kardianos/service for service management:** The `github.com/kardianos/service` package provides a single Go API for launchd (macOS), systemd (Linux), and Windows Service. This is the right choice — do not write platform-specific service registration code.

---

### MVP Definition (v1.3)

#### Launch With (Phase 1: Daemon foundation)

- [ ] Background daemon with Unix socket IPC
- [ ] Daemon auto-start from CLI if not running
- [ ] Shared session pool: GUI migrates to daemon-as-backend

#### Launch With (Phase 2: Core CLI commands)

- [ ] `agenthub new`, `list`, `kill`, `rename` — session lifecycle
- [ ] `agenthub web start/stop/status` — web serving control
- [ ] `agenthub health` — Tailscale health from CLI
- [ ] `agenthub qr <id>` — QR code output

#### Launch With (Phase 3: Attach/detach)

- [ ] `agenthub attach <id>` — full PTY proxy with raw I/O, resize, ctrl-c passthrough
- [ ] `agenthub detach` via configurable prefix key
- [ ] Terminal state restore on all exit paths
- [ ] Scrollback replay on reattach (already implemented in relay/scrollback.go)

#### Launch With (Phase 4: Service manager)

- [ ] `agenthub daemon install/uninstall/start/stop/status`
- [ ] launchd plist (macOS), systemd user unit (Linux), Windows Service

#### Add After Validation (v1.3.x)

- [ ] `agenthub serve <path>` / `agenthub unserve <id>` — sugar commands
- [ ] `agenthub settings` — read-only config inspection
- [ ] `--json` output on all list/status commands

#### Future Consideration (v2+)

- TUI session picker with fzf-style interface
- Multiple simultaneous clients attached to one session
- Remote daemon over TCP

---

### Feature Prioritization Matrix (v1.3)

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Background daemon + Unix socket | HIGH | MEDIUM | P1 |
| Shared session pool (GUI→daemon migration) | HIGH | HIGH | P1 |
| `agenthub new/list/kill/rename` | HIGH | LOW | P1 |
| `agenthub attach` with PTY proxy | HIGH | HIGH | P1 |
| `agenthub detach` with prefix key | HIGH | LOW | P1 |
| Terminal state restore | HIGH | MEDIUM | P1 (safety-critical) |
| `agenthub web start/stop/status` | MEDIUM | LOW | P1 |
| `agenthub health` | MEDIUM | LOW | P1 |
| Daemon auto-start on login (kardianos/service) | MEDIUM | MEDIUM | P1 |
| `agenthub qr <id>` | MEDIUM | LOW | P2 |
| Scrollback replay on attach | MEDIUM | LOW | P2 (infrastructure exists) |
| Configurable detach prefix | MEDIUM | LOW | P2 |
| `--json` output flags | LOW | LOW | P2 |
| `agenthub serve <path>` sugar | LOW | LOW | P3 |
| `agenthub settings` inspect | LOW | LOW | P3 |

---

### Competitor / Reference Feature Analysis (v1.3)

| Feature | tmux | shpool | Docker CLI | AgentHub v1.3 |
|---------|------|--------|------------|---------------|
| Session create | `tmux new -s name` | `shpool attach name` (creates if absent) | `docker run` | `agenthub new [agent] [path]` |
| Session list | `tmux ls` | `shpool list` | `docker ps` | `agenthub list` |
| Attach | `tmux attach -t name` | `shpool attach name` | `docker attach` | `agenthub attach <id>` |
| Detach | `ctrl+b d` (configurable) | `ctrl+]` (fixed) | `ctrl+p ctrl+q` | `ctrl+b d` (configurable) |
| Kill | `tmux kill-session -t name` | `shpool kill name` | `docker kill` | `agenthub kill <id>` |
| Rename | `tmux rename-session -t old new` | not supported | `docker rename` | `agenthub rename <id> <name>` |
| Daemon auto-start | server auto-starts on first tmux cmd | daemon must be started manually | dockerd managed by systemd | `agenthub daemon install` + kardianos/service |
| JSON output | not native | not supported | `--format json` | `--json` flag |
| Scrollback on reattach | full scrollback | in-memory VT100 replay | not applicable | relay/scrollback.go replay |
| GUI co-existence | not applicable | not applicable | not applicable | GUI + CLI share daemon session pool |
| Web serving | not applicable | not applicable | not applicable | `agenthub web start/stop/status` |

---

## v1.5 Milestone: Bug Fixes and CLI Args

### Scope

This section covers only what is NEW in v1.5. Existing app ships all v1.4 features. Two items
are bug fixes (regressions): terminal not filling on initial load, and slow agent startup
introduced by daemon mode. Three items are new features: CLI arg passthrough via `--`, GUI
args text field in the new-session modal, and per-agent argument memory. Research basis:
direct codebase analysis of the full call chain, xterm.js FitAddon behavior with hidden panels,
PTY size initialization patterns from VTE/libvte and VS Code terminal, and established CLI
convention for `--` argument separator.

---

### Table Stakes (Users Expect These for v1.5)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Terminal fills available space on first open | Every terminal app (VS Code, iTerm2, Warp) sizes to container on launch; a 80x24 terminal inside a full-screen window looks broken | MEDIUM | Root cause identified: `fit()` is deferred until the panel becomes active (`isActive` effect), but the relay has already sent PTY size 80x24 to the agent at session creation time. Claude Code and Gemini CLI render their TUI layout on first output at 80 columns and never reflow correctly after resize. Current code already handles `document.fonts.ready` + `ResizeObserver` — the gap is the PTY-side initial size. Fix: spawn with a wider default (e.g., 220x50) rather than 80x24. The relay resize frame sent after `fit()` runs will then only be a minor adjustment rather than a dramatic layout change. |
| Agent startup is fast | Users expect a session to be live within 1-2s of hitting Create; any visible spinner longer than that creates doubt | MEDIUM | Daemon persistence is working (EnsureDaemon is already a no-op when running). The regression is per-session creation latency. Likely cause: `exec.LookPath` is called at exec time inside go-pty for the default case (non-overridden CLI names are passed as-is, so the OS PATH scan happens on every `cmd.Start()`). Pre-resolving and caching CLI paths at daemon startup eliminates repeated filesystem scans. Secondary suspect: socket round-trip + HTTP overhead for the POST /sessions request. Profile before assuming. |
| CLI arg passthrough (`agenthub new <agent> <path> -- <args>`) | Standard POSIX/GNU convention used by dotnet run, docker run, npm run. Claude Code accepts `--model`, `--permission-mode`; Gemini CLI accepts `--model`, `--yolo`; OpenCode accepts prompts as positional args. Power users and script authors need this. | LOW | Plumbing already exists: `pty.CreateRequest.Args []string` is defined but unused. The entire call chain (cmdNew → DaemonClient.CreateSession → POST /sessions → engine.CreateSession → backend.Create) only needs `Args []string` added to `daemon.CreateRequest` (additive JSON change — backwards-compatible). `cmdNew` currently uses positional args without a FlagSet; refactor to `flag.NewFlagSet` which naturally terminates at `--`, making `fs.Args()` the passthrough args. |
| Args field in GUI new-session modal | GUI users expect the same capability as CLI users. VS Code launch profiles, iTerm2 profiles, and Warp workflows all support per-launch arguments. | LOW-MEDIUM | `NewSessionModal` currently calls `onConfirm(cli, workDir)` — extend to `onConfirm(cli, workDir, args string)`. Single-line text input. Passed to `App.CreateSession` which passes it to `DaemonClient.CreateSession`. Args string is space-split into `[]string` before sending to daemon; quoted strings are not supported in v1.5 (see anti-features). |
| Per-agent argument memory with pre-fill and clear | The last-folder memory in v1.1 established this UX pattern. Users running `claude --model claude-opus-4-5` repeatedly expect the field to pre-fill. | LOW | localStorage keyed by CLI name: `agenthub:lastArgs:<cli>`. Pre-fill the args field when an agent is selected in the modal. Clear button adjacent to the field. No backend involvement — purely frontend. Matches the `LAST_DIR_KEY` localStorage pattern already in `NewSessionModal.tsx`. |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Correct initial PTY columns (no narrow-layout flash) | Claude Code and Gemini CLI render TUI layouts on first output. Competitors (ccmanager, others) have the same 80-column default bug and have not fixed it. An app that opens correctly on first paint is noticeably better. | MEDIUM | Spawn with cols=220, rows=50 as the default. 220 is wide enough that no common AI CLI TUI layout will wrap. The relay will adjust to actual dimensions within one `fit()` cycle (typically <100ms after the panel renders). This is a one-line change in `engine.CreateSession` but requires coordinated testing across Claude Code and Gemini CLI to confirm the layout improvement. |
| Per-agent arg memory (CLI-specific) | Other terminal apps have profile-level defaults; storing args per-CLI-type is a lightweight version that requires zero configuration UI. | LOW | Key by CLI name in localStorage. The per-agent scope means `claude` and `gemini` args are remembered independently, which is the right behavior since they have different flag vocabularies. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Shell-interpolated args in GUI | "Pass args as a shell string, let the shell expand it" | Cross-platform: Windows has no POSIX shell. Shell injection risk. The PTY is already a shell; double-interpolation confuses users. Paths with spaces break unless quoted correctly. | Accept a raw args string, split on whitespace, pass `[]string` to exec directly. In v1.5 this is sufficient for all documented AI CLI flags. Quoted-string support can be added in v1.6 if users need it. |
| Persistent args stored in daemon or settings file | "Remember my args forever, survive reinstall" | Daemon settings are currently in-memory only. Adding disk persistence introduces schema migrations, config file location decisions, and upgrade complexity. | Use Wails WebView localStorage, which persists across app launches within the same machine, same user. No daemon changes. Same pattern as `lastWorkDir`. |
| Per-session arg editing after spawn | "Let me change the model flag on a running session" | PTY processes don't support re-exec with new args. This would require kill + recreate, destroying scrollback and session state. | Args are creation-time only. New args require a new session (open another tab). |
| Complex shell quoting in args field | "Support `--prompt 'multi word message'` in the GUI" | Requires a full POSIX tokenizer (shlex). Adds implementation complexity and testing surface. Not needed for the v1.5 use cases (model flags, boolean flags). | Defer quoted-string support to v1.6. In v1.5, document that the args field is whitespace-split. |
| Automatic daemon warm-up / pre-spawning | "Pre-start sessions before I ask for them" | Daemon already persists across GUI close/reopen. The startup latency is per-session exec time, not daemon startup. Pre-spawning wastes resources for sessions the user may not open. | Fix the exec.LookPath caching to reduce per-session creation latency directly. |

---

### Feature Dependencies (v1.5)

```
[CLI arg passthrough: agenthub new -- <args>]
    requires --> [Args []string field in daemon.CreateRequest]
    requires --> [Args []string in DaemonClient.CreateSession]
    requires --> [Args []string in engine.CreateSession]
    requires --> [Args []string in pty.CreateRequest -- already exists, currently unused]
    requires --> [cmdNew refactored to use flag.NewFlagSet with -- termination]

[GUI args field in NewSessionModal]
    requires --> [Args []string field in daemon.CreateRequest -- same as above]
    requires --> [App.CreateSession(cli, name, workDir, args string)]
    enhances --> [Per-agent argument memory]

[Per-agent argument memory]
    requires --> [GUI args field -- no point pre-filling if there is no field]
    enhances --> [GUI args field -- pre-fills on agent selection]
    independent-of --> [CLI arg passthrough -- CLI users set args inline]

[Terminal initial-fit fix]
    requires --> [Wider default PTY size in engine.CreateSession (220x50 instead of 80x24)]
    independent-of --> [CLI arg passthrough and per-agent memory]
    note --> [Frontend fit() timing already correct; the regression is the PTY spawn size]

[Slow agent startup fix]
    requires --> [Profiling to confirm root cause before changing code]
    independent-of --> [All other v1.5 features -- different code path entirely]
    likely-fix --> [Cache exec.LookPath result for each CLI at daemon startup]
```

### Dependency Notes

- **Args passthrough requires one protocol change:** `daemon.CreateRequest` needs `Args []string`. This is an additive JSON field (missing = nil = no extra args). No version negotiation needed. All three layers (CLI, App.go Wails binding, daemon API handler) need to thread the field through.
- **GUI args and per-agent memory should be one phase:** They are tightly coupled (pre-fill only makes sense with the field present) and collectively amount to ~40 lines of frontend code.
- **Terminal fit fix is independent and should be first:** It is the most visible regression and the fix is a one-line change in Go plus validation. Ship it first. The xterm.js `isActive` + `ResizeObserver` + `document.fonts.ready` chain is already correct; only the PTY spawn size needs changing.
- **Startup regression requires profiling before fixing:** The cause is inferred (exec.LookPath repeated on each spawn) but not yet measured. A phase that adds timing instrumentation and confirms the root cause before implementing the fix reduces the risk of fixing the wrong thing.

---

### MVP Definition (v1.5)

All five items are the milestone. There is no sub-MVP — two are regressions reported by users
and three are directly related new features.

#### Launch With

- [ ] **Terminal fill fix** — visible regression; makes product look broken on first open for Claude and Gemini CLIs
- [ ] **Slow agent startup fix** — regression from daemon separation; affects every agent on every session create
- [ ] **CLI arg passthrough** — `agenthub new <agent> <path> -- <extra-args>`; core new feature for scripting users
- [ ] **GUI args text field** — parity with CLI; required so GUI users have the same capability
- [ ] **Per-agent argument memory** — polish add-on; bundle with GUI args field (same phase, low cost)

#### Defer

- Shell-aware (shlex-style) quoted-string tokenization for the GUI args field — simple space-split is sufficient for v1.5 use cases
- Args stored in a daemon-side persistent settings file — localStorage is sufficient
- Per-session argument editing after spawn — requires kill+recreate, different feature

---

### Feature Prioritization Matrix (v1.5)

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Terminal fill fix (wider PTY spawn size) | HIGH — visible regression | LOW — one-line change + validation | P1 |
| Slow agent startup fix | HIGH — regression affects all agents | MEDIUM — profile first, then fix | P1 |
| CLI arg passthrough (`--`) | HIGH — scripting use case | LOW — plumbing exists | P1 |
| GUI args text field | HIGH — parity with CLI | LOW — modal extension | P1 |
| Per-agent arg memory | MEDIUM — convenience | LOW — localStorage only | P1 (bundle with GUI args) |

---

### How Comparable Tools Handle These Problems

#### Terminal Initial Sizing

| Tool | Approach | Result |
|------|----------|--------|
| VS Code integrated terminal | Queries layout dimensions synchronously before PTY spawn; sends SIGWINCH after attach if size changed | No 80-column flash in normal case; known issue only when panel is hidden at spawn (GitHub #98399) |
| iTerm2 | Sets PTY window size (TIOCSWINSZ via `ptySetSize`) before spawning child process | Correct from first byte; this is the VTE canonical pattern |
| Warp | WebView-based; waits for layout render, then spawns PTY with measured cols/rows | No flash; tradeoff is spawn latency after first paint |
| ccmanager (direct competitor) | Same 80x24 default bug; relies on SIGWINCH from subsequent resize | Users report wrapped layouts on first open; unfixed as of 2025 |
| AgentHub v1.4 | Spawns 80x24; ResizeObserver triggers fit() + relay resize after isActive | Regression: agent paints initial TUI at 80 cols; subsequent resize does not cause layout reflow in Claude Code or Gemini CLI |

The VTE/libvte canonical pattern is to set PTY size _before_ spawning the child process so it reads correct dimensions on startup. In AgentHub's architecture, the frontend does not know its pixel dimensions at session-creation time (the session is created before the terminal panel renders). The practical fix is to spawn with a generous default (220x50) so that no AI CLI TUI layout wraps at spawn time. The relay resize that arrives after `fit()` becomes a minor adjustment rather than a full layout reflow.

#### Daemon Startup Latency

| Tool | Per-Session Approach |
|------|---------------------|
| VS Code | Extension host pre-warms; terminal process created before user opens panel |
| iTerm2 | Pre-forks shell processes into a pool |
| AgentHub v1.4 | Daemon persists; per-session latency is exec time + socket round-trip |

The latency in AgentHub is not daemon startup (already solved by persistence) — it is per-session exec time. The inferred bottleneck is `exec.LookPath` being called at spawn time inside go-pty for each new session (Go issue #36768 confirms this is a known pattern). Pre-resolving CLI paths at daemon startup and caching them eliminates repeated filesystem scans.

#### CLI Arg Passthrough Convention

| Tool | Convention | Example |
|------|------------|---------|
| dotnet run | `--` separator | `dotnet run -- --myapp-flag value` |
| docker run | Args after image name | `docker run image cmd --flag` |
| npm run | `--` separator | `npm run start -- --port 3001` |
| concurrently | `-P` / `--passthrough-arguments` + `--` | `concurrently -P -- npm start -- --flag` |
| agenthub new (v1.5) | `--` separator | `agenthub new claude /path -- --model claude-opus-4-5` |

The `--` convention is the POSIX/GNU standard. Go's `flag.NewFlagSet` with `ContinueOnError` stops parsing at `--` and makes remaining args available via `fs.Args()`. `cmdNew` currently uses raw positional args without a FlagSet; refactoring it to use `flag.NewFlagSet` (consistent with `cmdList`, `cmdWebStatus`, etc.) makes `--` termination free.

---

### Sources

- AgentHub codebase (direct read): `/Users/ken/dev/agenthub/` — HIGH confidence (primary source)
- xterm.js FitAddon with `display:none`: [GitHub xtermjs/xterm.js #3029](https://github.com/xtermjs/xterm.js/issues/3029) — MEDIUM confidence
- VS Code terminal 80-column wrap bug: [GitHub microsoft/vscode #98399](https://github.com/microsoft/vscode/issues/98399) — MEDIUM confidence
- VTE `set_size` before spawn canonical pattern: web search result from VTE/libvte documentation — MEDIUM confidence
- Go `exec.LookPath` caching: [golang/go #36768](https://github.com/golang/go/issues/36768) — MEDIUM confidence
- CLI `--` passthrough convention: [GNU Coding Standards](https://www.gnu.org/prep/standards/html_node/Command_002dLine-Interfaces.html) — HIGH confidence
- dotnet run `--` convention: [Microsoft Learn CLI design guidance](https://learn.microsoft.com/en-us/dotnet/standard/commandline/design-guidance) — HIGH confidence
- concurrently passthrough: [concurrently docs](https://github.com/open-cli-tools/concurrently/blob/main/docs/cli/passthrough-arguments.md) — HIGH confidence
- Claude Code CLI flags: [Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference) — HIGH confidence
- OpenCode CLI: [opencode.ai/docs/cli](https://opencode.ai/docs/cli/) — MEDIUM confidence

---

## Sources (All Milestones)

- tmux man page and wiki: https://github.com/tmux/tmux/wiki/Getting-Started
- shpool architecture: https://deepwiki.com/shell-pool/shpool
- zmx session persistence: https://lobste.rs/s/fvdh2d/zmx_session_persistence_for_terminal
- kardianos/service Go package: https://pkg.go.dev/github.com/kardianos/service
- creack/pty PTY library: https://pkg.go.dev/github.com/creack/pty (SIGWINCH, InheritSize, raw mode patterns)
- Unix domain sockets in Go: https://eli.thegreenplace.net/2019/unix-domain-sockets-in-go/
- Zellij session management: https://zellij.dev/tutorials/session-management/
- Existing codebase: `/Users/ken/dev/agenthub/internal/relay/` (scrollback.go already implemented)

---

*Feature research for: AgentHub v1.3 CLI + Daemon, v1.5 Bug Fixes and CLI Args*
*Researched: 2026-03-25*
