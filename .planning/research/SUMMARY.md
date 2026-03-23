# Project Research Summary

**Project:** AgentHub v1.3 CLI + Daemon
**Domain:** Go CLI + background daemon + terminal attach/detach added to an existing Go/Wails desktop app
**Researched:** 2026-03-23
**Confidence:** HIGH

## Executive Summary

AgentHub v1.3 is a well-scoped architectural evolution of a mature desktop app. The core change is extracting session management out of the Wails GUI process into a persistent background daemon, then providing a CLI that is a peer client to the same daemon. The established pattern for this problem (used by Docker, gopls, tailscaled, and tmux) is: HTTP/JSON over a Unix domain socket (named pipe on Windows), a single binary that dispatches on os.Args, and a foreground daemon managed by the platform's native service manager. All three patterns are directly applicable and well-documented. The existing codebase has excellent internal package boundaries — `internal/relay`, `internal/pty`, `internal/webserver`, and `internal/status` are all daemon-ready today with no structural changes required.

The recommended approach is a five-phase migration that avoids big-bang rewrites. Phase 1 extracts `SessionEngine` from `App` without any process separation, establishing the module boundary and keeping all existing tests green. Phase 2 adds the HTTP/JSON IPC layer while the daemon is still in-process, validating the protocol before any process fork. Phase 3 forks the daemon into a separate process — the first phase where sessions genuinely outlive the GUI. Phase 4 adds CLI commands built on the validated `DaemonClient`. Phase 5 adds cross-platform service manager integration via `kardianos/service`. This incremental path means each phase is independently verifiable and rollback is cheap.

The highest-risk items are: (1) the architectural migration of session state from `App` to the daemon — if any state lives in two places, divergence bugs are nearly impossible to diagnose; (2) the PTY attach command, which requires correct terminal raw mode, SIGWINCH resize forwarding, signal proxying (Ctrl-C must pass through to the AI CLI, not terminate the attach process), and bulletproof terminal restore on every exit path; and (3) the Wails os.Args dispatch guard, which must correctly pass through internal Wails build arguments that are not user subcommands. These three items need explicit testing gates before moving to the next phase.

## Key Findings

### Recommended Stack

The v1.2 codebase already contains all low-level dependencies needed for v1.3. Only two new direct dependencies are required: `github.com/spf13/cobra@v1.9.1` for the CLI command tree, and `github.com/kardianos/service@v1.2.4` for cross-platform service registration. `golang.org/x/term` is already in the binary transitively and just needs promotion to a direct dependency. IPC uses stdlib `net.Listen("unix", ...)` plus the already-present `tailscale/go-winio` for Windows named pipes. No gRPC, no protobuf, no additional IPC library.

**Core technologies:**
- `github.com/spf13/cobra v1.9.1`: CLI command framework — industry standard (Kubernetes, Docker, GitHub CLI), best persistent-flag inheritance for nested command groups like `agenthub web start/stop/status`
- `github.com/kardianos/service v1.2.4`: Service manager integration — the only mature Go library handling launchd (macOS), systemd (Linux), and Windows Service from one unified API; released July 2025
- `golang.org/x/term` (promote from indirect): Terminal raw mode for attach — official Go extended library, already in the binary, `MakeRaw`/`Restore`/`GetSize` API confirmed at v0.41.0
- stdlib `net` + existing `tailscale/go-winio`: IPC transport — Unix socket on POSIX, named pipe on Windows, no additional dependencies
- stdlib `encoding/json` over socket: IPC protocol — sufficient for ~15 command types; Docker and tailscaled use this same pattern

### Expected Features

The v1.3 milestone has a clear four-group feature set aligned with the migration phases.

**Must have (table stakes):**
- Background daemon process with Unix socket IPC — the foundational prerequisite for everything else
- Shared session pool: GUI and CLI see identical sessions — broken product if missing; highest architectural risk
- `agenthub new`, `list`, `kill`, `rename` — session lifecycle core; users of any session manager expect these
- `agenthub attach <id>` with full PTY proxy (raw I/O, resize, Ctrl-C passthrough) — session persistence is useless without reconnect
- `agenthub detach` via configurable prefix key — attach without detach is a trap
- Terminal state restore on all exit paths — non-negotiable safety feature; broken terminal is the most visible failure mode
- `agenthub daemon install/uninstall/start/stop/status` — service lifecycle commands expected in any daemon-managed tool
- `agenthub web start/stop/status <id>`, `agenthub health`, `agenthub qr <id>` — parity with existing GUI features

**Should have (competitive):**
- Scrollback replay on reattach — infrastructure already exists in `relay/scrollback.go`; near-zero implementation cost; confirmed user expectation from tmux/shpool research
- Configurable detach prefix key — tmux's Ctrl-B conflicts with many editors; configurable prefix is a known quality-of-life win
- `--json` output on all list/status commands — standard in modern CLIs (gh, kubectl, docker); enables scripting
- Daemon auto-start from CLI if not running — CLI is unusable if users must manually start the daemon first

**Defer (v1.3.x after validation):**
- `agenthub serve <path>` / `agenthub unserve <id>` sugar commands
- `agenthub settings` read-only config inspection
- TUI session picker (fzf-style) — v2+
- Multiple simultaneous clients attached to one session — v2+
- Remote daemon over TCP — v2+

### Architecture Approach

The architecture is a daemon-centric model with two equal client types: the existing Wails GUI (thinned to a client) and the new CLI. The daemon owns all session state in a new `internal/daemon/` package containing `SessionEngine` (session logic extracted from `App`), `DaemonAPI` (HTTP handler on the Unix socket), and `DaemonClient` (typed Go client used by both GUI and CLI). All existing internal packages — `internal/relay/`, `internal/pty/`, `internal/status/`, `internal/webserver/` — are unchanged; they move from being constructed in `App` to being constructed in `daemon.SessionEngine`. The Wails GUI's `App` struct shrinks from owning ~8 fields of session state to owning exactly one: `*daemon.DaemonClient`. All Wails-bound method signatures on `App` stay identical; zero frontend changes are required in Phases 1-3.

**Major components:**
1. `internal/daemon/engine.go` — `SessionEngine`: owns `SessionRegistry`, `NativePTYBackend`, `HubManager`, status/tabNames maps; extracted from `App`
2. `internal/daemon/api.go` — `DaemonAPI`: HTTP/JSON handler on Unix socket; 15 routes covering session CRUD, web serving, health, attach WebSocket, settings, and SSE events
3. `internal/daemon/client.go` — `DaemonClient`: typed Go wrapper used by both GUI `App` and all CLI commands; the only new dependency either consumer introduces
4. `cmd/cli/` — cobra command tree: `new`, `list`, `attach`, `kill`, `rename`, `web`, `health`, `qr`, `settings`; all thin wrappers over `DaemonClient`
5. `cmd/daemon/main.go` — daemon subcommand + `kardianos/service` install/uninstall for launchd/systemd/Windows SCM

### Critical Pitfalls

1. **Session state in two places after extraction** — After Phase 2, `App` must hold zero authoritative session state. Any local map (`tabNames`, `sessionStatuses`) not migrated to the daemon creates divergence bugs that appear only in multi-client scenarios and are extremely hard to diagnose. Verification: GUI `list` output and CLI `list` output must be identical in all scenarios.

2. **Terminal left in raw mode after crash** — `defer term.Restore(...)` does not run on SIGKILL, `os.Exit`, or `log.Fatal`. Signal handlers for SIGTERM, SIGINT, and SIGHUP must explicitly call `term.Restore` before exiting. `log.Fatal` must never be called from within the raw mode attach path. This is the most user-visible failure: the shell appears frozen after an abnormal exit.

3. **Wails os.Args dispatch breaks the build** — Wails v2 invokes the compiled binary during code generation without user arguments. The `main()` dispatch guard must never panic or exit non-zero on `len(os.Args) == 1`. Guard Wails-internal arguments (e.g. `/tmp/wailsbindings`) with an `isWailsInternalArg` check and a `//go:build production` tag.

4. **Stale Unix socket blocks daemon restart** — After a crash or SIGKILL, the socket file remains on disk. The next startup gets `address already in use`. At startup: attempt connect; if connection refused, remove socket with `os.Remove`; then listen. Add a path length assertion (<104 chars on macOS) to catch cryptic `bind: invalid argument` errors for users with long usernames.

5. **PTY resize not propagated through the daemon proxy** — SIGWINCH is delivered to the CLI client process, not the daemon that owns the PTY master FD. The CLI must send resize frames through the IPC protocol; the daemon applies them via `pty.Resize()`. This is easy to miss because initial testing in a fixed-size terminal always works. Test explicitly by resizing the window during an active attach session.

6. **Daemon must not self-daemonize on macOS** — launchd expects to be the parent of the service process. A double-fork causes launchd to lose the PID and enter rapid-restart loops. Use `kardianos/service` throughout — it implements the correct foreground-run contract for each platform.

7. **Ctrl-C signal handling in raw mode** — In attach mode, Ctrl-C must be forwarded to the AI CLI as a PTY input byte (0x03), not handled as a Go SIGINT to terminate the attach process. Only SIGTERM and SIGHUP should trigger clean detach. The detach prefix state machine intercepts the detach sequence before bytes reach the PTY write path.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: SessionEngine Extraction

**Rationale:** Establishes the module boundary without any behavior change. The compiler enforces the interface before any process separation. All existing tests pass unchanged. This is the prerequisite for every subsequent phase.
**Delivers:** `internal/daemon/engine.go` with `SessionEngine` wrapping all session logic extracted from `App`; `App` delegates to `SessionEngine` directly (still in-process, no IPC yet)
**Addresses:** "Shared session pool" foundational requirement; "session state in two places" pitfall prevention starts here
**Avoids:** Big-bang rewrite risk; every phase after this builds on a tested module boundary

### Phase 2: DaemonAPI + DaemonClient (In-Process IPC)

**Rationale:** Validates the HTTP/JSON protocol over the Unix socket before adding process-separation complexity. The daemon is still in-process; bugs in JSON serialization or socket path handling are trivially reproducible. Existing tests run with a local in-process daemon.
**Delivers:** `internal/daemon/api.go` (HTTP handler), `internal/daemon/client.go` (typed client); `App` calls `DaemonClient` instead of holding `SessionEngine` directly; protocol fully validated
**Avoids:** Session state divergence (removes the last direct field ownership from `App`); stale socket pitfall (socket lifecycle handling implemented here); socket path >104 chars (assertion added here)

### Phase 3: Process Separation + Relay Migration

**Rationale:** First phase where sessions genuinely outlive the GUI window. The relay.Server moves from `App.startup` into the daemon. GUI connects to daemon on startup and retrieves the relay port. This is the highest-risk phase and must be gated by an explicit test: close the GUI, reopen it, confirm sessions are still listed and attachable.
**Delivers:** `cmd/daemon/main.go`; forked daemon process; `DaemonClient.ensureRunning()` with exponential backoff retry; GUI health events via SSE stream; GUI `App` struct holds only `*daemon.DaemonClient`
**Uses:** stdlib `net` (Unix socket), exponential backoff startup sync with 5s deadline (not `time.Sleep`)
**Avoids:** "Daemon as goroutine" anti-pattern (sessions still die with GUI); daemon startup race (fixed sleep is brittle on slow machines)

### Phase 4: CLI Commands

**Rationale:** All CLI commands are thin wrappers over `DaemonClient`, which is fully validated by Phase 3. The only complex command is `attach` — treat it as a mini-project. Build and test simpler commands first to validate the cobra tree and output formatting before tackling attach.
**Delivers:** Full `cmd/cli/` tree; `agenthub new/list/kill/rename/web/health/qr`; `agenthub attach` with PTY proxy, raw mode, SIGWINCH resize, signal proxying, scrollback replay, detach prefix state machine, terminal restore on all exit paths; Wails os.Args dispatch guard in `main.go`
**Implements:** cobra dispatch; `golang.org/x/term` promoted to direct dependency
**Avoids:** Terminal raw mode left on crash (signal handlers + no `log.Fatal` in raw path); PTY resize not propagated; Ctrl-C handled as shutdown instead of passthrough; Wails os.Args dispatch breakage

### Phase 5: Service Manager Integration

**Rationale:** `kardianos/service` wraps all platform complexity. Lower risk than Phases 3-4 but requires explicit testing on each platform because launchd, systemd, and Windows SCM have different failure modes.
**Delivers:** `agenthub daemon install/uninstall/start/stop/status`; launchd plist at `~/Library/LaunchAgents/`; systemd user unit; Windows service registration; daemon runs in foreground (no double-fork)
**Uses:** `github.com/kardianos/service v1.2.4`
**Avoids:** macOS launchd conflict (no self-daemonize); binary path hard-coded in plist (derive from `os.Executable()` at install time); `launchctl load` deprecated call (use `launchctl bootstrap`)

### Phase 6: Polish + v1.3.x Features

**Rationale:** After service manager is validated, add differentiator features that have near-zero implementation cost given the infrastructure from Phases 1-5.
**Delivers:** `--json` output on all list/status commands; `agenthub serve <path>` sugar; `agenthub settings` read-only inspection; configurable detach prefix key stored in config; UX improvements (attach banner showing session identity, detach key hint on connect)

### Phase Ordering Rationale

- **Phases 1-2 must precede Phase 3:** Process separation before the IPC protocol is proven creates multi-process debugging nightmares. Validate protocol in-process first.
- **Phase 3 gate:** Close GUI, verify sessions persist, reopen GUI, verify sessions appear. Do not proceed to Phase 4 until this passes.
- **Attach is isolated within Phase 4:** Build all simple CLI commands before `attach`. Attach has seven distinct correctness requirements (raw mode, SIGWINCH, Ctrl-C forwarding, detach prefix, scrollback replay, resize propagation, terminal restore) that each need explicit tests.
- **Service manager last:** The daemon binary must be stable before registering it with the OS service manager. A crashing daemon in launchd triggers rapid-restart loops that are annoying to diagnose.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Process Separation):** The relay port handoff between daemon and GUI (`GetRelayPort()` via daemon API) has no direct precedent in the codebase; the exact sequence for GUI startup, daemon detection, and relay port acquisition needs to be pinned during planning
- **Phase 5 (Service Manager):** Windows SCM behavior with `kardianos/service` is MEDIUM confidence; the library is well-maintained but Windows CI coverage is not confirmed in the existing project

Phases with standard patterns (skip research-phase):
- **Phase 1 (SessionEngine Extraction):** Pure Go refactor with direct codebase inspection; no external unknowns
- **Phase 2 (In-Process IPC):** HTTP/JSON over Unix socket is a fully documented pattern; `DaemonAPI` routes are enumerated in ARCHITECTURE.md
- **Phase 4 simple commands:** `new`, `list`, `kill`, `rename`, `web`, `health`, `qr` are thin wrappers; cobra patterns are well-established
- **Phase 4 attach:** Research is complete and detailed in STACK.md and ARCHITECTURE.md; implementation risk is execution, not unknowns

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All versions verified via pkg.go.dev; existing go.mod inspected directly; no speculative dependencies |
| Features | HIGH | Grounded in tmux/shpool/Docker CLI comparison; existing codebase read directly; `relay/scrollback.go` confirmed present |
| Architecture | HIGH | Direct codebase inspection of `app.go`, `internal/relay/`, `internal/pty/`; Docker/gopls/tailscaled precedent for HTTP-over-Unix-socket; migration path is incremental and reversible |
| Pitfalls | HIGH (POSIX/signal/PTY), MEDIUM (Wails-specific) | Signal/PTY/socket pitfalls verified against official docs and GitHub issues; Wails os.Args coexistence is MEDIUM — community discussion, limited official docs |

**Overall confidence:** HIGH

### Gaps to Address

- **Wails dev mode argument handling:** The `isWailsInternalArg` guard for `/tmp/wailsbindings` is documented in a community discussion, not official docs. Verify empirically during Phase 1 with `wails dev` to confirm the exact internal arg pattern before relying on it.
- **Windows CI coverage:** Windows named pipe IPC via `tailscale/go-winio` and service registration via `kardianos/service` on Windows SCM are noted in research but not confirmed with a CI run. Establish Windows CI in Phase 2 before Phase 5 makes it critical.
- **Relay port handoff sequence:** The exact GUI startup sequence (start daemon if needed → wait for socket → get relay port → start React) needs to be pinned during Phase 3 planning with respect to Wails lifecycle hooks.
- **Socket path length assertion:** The 104-character `sun_path` limit on macOS must be verified for `~/.config/agenthub/daemon.sock` with usernames up to ~30 characters. Add a startup assertion that panics with a clear message rather than the cryptic `bind: invalid argument`.

## Sources

### Primary (HIGH confidence)
- pkg.go.dev/github.com/spf13/cobra — v1.9.1 feature set, persistent flags, shell completion verified
- pkg.go.dev/github.com/kardianos/service — v1.2.4 platform support (launchd/systemd/Windows SCM) and API verified
- pkg.go.dev/golang.org/x/term — v0.41.0 MakeRaw/Restore/GetSize API confirmed
- pkg.go.dev/github.com/aymanbagabas/go-pty — v0.2.2 ReadWriteCloser + Resize API verified
- `/Users/ken/dev/agenthub/go.mod` — existing dependencies verified directly
- `/Users/ken/dev/agenthub/internal/relay/hub.go` — Hub/Subscriber fan-out compatibility with attach use case confirmed
- `/Users/ken/dev/agenthub/app.go` — current App struct field inventory for migration planning
- Docker daemon, tailscaled, gopls — HTTP/JSON over Unix socket pattern precedent (HIGH confidence)
- Eli Bendersky: Unix Domain Sockets in Go — `net.Listen("unix", ...)` pattern

### Secondary (MEDIUM confidence)
- github.com/wailsapp/wails/discussions/4175 — os.Args pattern in Wails v2 and production build tag approach
- github.com/wailsapp/wails/issues/1533 — `-appargs` flag conflict with argument flags
- iximiuz.com — Linux PTY attach/detach internals (raw mode + PTY proxy pattern, same as Docker attach)
- kardianos/service README — cross-platform service config examples
- Apple Developer: Creating Launch Daemons and Agents — launchd plist foreground-run contract
- shpool architecture (deepwiki.com/shell-pool/shpool) — session persistence patterns and scrollback replay behavior
- VictoriaMetrics: Graceful Shutdown in Go — signal handling and terminal restore patterns

### Tertiary (LOW confidence)
- None — all key technical claims have HIGH or MEDIUM sources

---
*Research completed: 2026-03-23*
*Ready for roadmap: yes*
