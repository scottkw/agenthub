---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: executing
stopped_at: Completed 06-01-PLAN.md
last_updated: "2026-03-19T13:02:19.240Z"
last_activity: 2026-03-18 — Plan 05-03 complete (QR modal + status badge dots wired into React desktop UI)
progress:
  total_phases: 6
  completed_phases: 5
  total_plans: 19
  completed_plans: 18
  percent: 93
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-17)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 1 — PTY Foundation

## Current Position

Phase: 5 of 6 (QR Codes + Status Indicators)
Plan: 3 of 3 in current phase
Status: In progress
Last activity: 2026-03-18 — Plan 05-03 complete (QR modal + status badge dots wired into React desktop UI)

Progress: [█████████░] 93%

## Performance Metrics

**Velocity:**
- Total plans completed: 1
- Average duration: 2 min
- Total execution time: 2 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-pty-foundation | 1 of 2 | 2 min | 2 min |

*Updated after each plan completion*
| Phase 01-pty-foundation P02 | 10min | 3 tasks | 14 files |
| Phase 02-session-registry-websocket-relay P01 | 3min | 2 tasks | 6 files |
| Phase 02-session-registry-websocket-relay P02 | 3min | 2 tasks | 5 files |
| Phase 03-wails-desktop-ui P01 | 13min | 2 tasks | 20 files |
| Phase 03-wails-desktop-ui P02 | 5min | 2 tasks | 10 files |
| Phase 03-wails-desktop-ui P03 | 90min | 2 tasks | 12 files |
| Phase 04-web-serving-tls-auth P02 | 3min | 2 tasks | 4 files |
| Phase 04-web-serving-tls-auth P01 | 15min | 2 tasks | 4 files |
| Phase 04-web-serving-tls-auth P03 | 4min | 2 tasks | 5 files |
| Phase 04-web-serving-tls-auth P04 | 30 | 3 tasks | 6 files |
| Phase 05-qr-codes-status-indicators P02 | 8min | 2 tasks | 4 files |
| Phase 05-qr-codes-status-indicators P01 | 15min | 2 tasks | 7 files |
| Phase 05-qr-codes-status-indicators P03 | 20min | 2 tasks | 4 files |
| Phase 05-qr-codes-status-indicators P05 | 5min | 1 tasks | 2 files |
| Phase 05-qr-codes-status-indicators P04 | 8 | 1 tasks | 2 files |
| Phase 05-qr-codes-status-indicators P06 | 5min | 1 tasks | 2 files |
| Phase 06-distribution-cross-platform P01 | 10min | 3 tasks | 7 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Use go-pty (aymanbagabas) not creack/pty — Windows ConPTY support required from day one
- [Roadmap]: win32-input-mode state-machine parser needed in Phase 1 — cannot be retrofitted
- [Roadmap]: CA cert pattern (not bare self-signed leaf) required for WSS — browsers silently reject untrusted wss://
- [Roadmap]: PLAT-01/02/03 assigned to Phase 6 but cross-platform validation is incremental each phase
- [Phase 01-pty-foundation]: SessionBackend is an interface so Plan 02 provides the platform implementation without touching Plan 01 types
- [Phase 01-pty-foundation]: DetectCLIs returns make([]DetectedCLI, 0) not nil — callers can range safely without nil check
- [Phase 01-pty-foundation]: Registry owns session lifetime: context cancellation does NOT remove sessions from registry
- [Phase 01-pty-foundation]: Do not combine Setpgid:true with go-pty — Setsid already creates new session (PGID==PID); combining causes EPERM on macOS
- [Phase 01-pty-foundation]: Close PTY master before cmd.Wait in killSession — prevents indefinite block when PTY slave is still referenced after child exits
- [Phase 01-pty-foundation]: win32input_parse.go has no build tag — stateless chunk parser compiles everywhere so unit tests run on all platforms
- [Phase 01-pty-foundation]: session.job stored as any — avoids Windows build tags in session.go; type assertion done in cleanup_windows.go
- [Phase 02-session-registry-websocket-relay]: Hub stores scrollback as framed bytes (MakeOutputFrame wrapped) so WebSocket clients receive identical bytes from live stream and replay without re-framing
- [Phase 02-session-registry-websocket-relay]: Scrollback.Append uses in-place copy-left on overflow (no extra allocation) to reduce GC pressure under high-throughput PTY output
- [Phase 02-session-registry-websocket-relay]: Hub.Shutdown uses sync.Once — allows Run to call it on return and external callers to call it safely without panic
- [Phase 02-session-registry-websocket-relay]: HubManager.Create is idempotent — returns existing hub if session already exists, preventing double-Run goroutines
- [Phase 02-session-registry-websocket-relay]: websocket.Accept uses InsecureSkipVerify:true — origin validation deferred to Phase 4 where CORS policy will be defined with known Electron origin
- [Phase 03-wails-desktop-ui]: os.DirFS stub (build tag !wailsassets) used instead of //go:embed — Go embed prohibits .. paths and does not follow symlinks; frontend/ is at repo root, cmd/agenthub/main.go can't reach it directly
- [Phase 03-wails-desktop-ui]: resizeFn callback injected into Hub at construction time — keeps relay package free of pty import cycle
- [Phase 03-wails-desktop-ui]: HubManager.Create accepts resizeFn parameter — App.CreateSession wires per-session resize to backend.Resize via closure
- [Phase 03-wails-desktop-ui]: App.ctx set to context.Background() in testApp helper — startup() not called in tests but backend.Create requires non-nil context
- [Phase 03-wails-desktop-ui]: Wails TypeScript stubs in wailsjs/ allow tsc compilation without running Go backend; wails dev regenerates them at runtime
- [Phase 03-wails-desktop-ui]: display:none for inactive TerminalPanel divs preserves xterm.js scrollback buffer — unmounting destroys it
- [Phase 03-wails-desktop-ui]: fitAddon.fit() only called on active terminal — hidden terminals return 0x0 sizing causing malformed resize frames
- [Phase 03-wails-desktop-ui]: Replaced fyne.io/systray with native macOS cgo NSStatusBar — fyne defines AppDelegate which conflicts with Wails AppDelegate (duplicate symbol linker error)
- [Phase 03-wails-desktop-ui]: Moved all Go files from cmd/agenthub/ to project root — Wails v2 requires main package co-located with wails.json
- [Phase 04-web-serving-tls-auth]: AuthManager sessions map uses string->time.Time for future expiry support
- [Phase 04-web-serving-tls-auth]: TokenStore uses bidirectional maps (tokenToSession + sessionToTokens) for O(1) lookup and O(n) bulk revocation
- [Phase 04-web-serving-tls-auth]: Leaf key never written to disk — CA key on disk is already a risk; leaf generated in-memory each launch
- [Phase 04-web-serving-tls-auth]: IPv4-only in ListInterfaces — VPN/Tailscale interfaces are always IPv4; keeps dropdown clean
- [Phase 04-web-serving-tls-auth]: clock skew buffer NotBefore=time.Now().Add(-time.Minute) prevents immediate rejection on machines with slight clock drift
- [Phase 04-web-serving-tls-auth]: OriginPatterns: ['*'] in websocket.Accept — sessionAuth middleware already validated the request
- [Phase 04-web-serving-tls-auth]: webEnabled map controls /sessions/{id} route separately from HubManager — session can be enabled before hub is created
- [Phase 04-web-serving-tls-auth]: Lazy WebServer init in App — webServer field starts nil; created on first call requiring it, avoids startup ordering issues
- [Phase 04-web-serving-tls-auth]: Password persisted as bcrypt hash to ~/.config/agenthub/web_password — survives restarts without storing plaintext
- [Phase 04-web-serving-tls-auth]: StartWebServer gates on IsWebPasswordSet() — cannot start web server without password set first
- [Phase 05-qr-codes-status-indicators]: Detector initial state is empty-sentinel so first Feed always fires onTransit
- [Phase 05-qr-codes-status-indicators]: Guard EventsEmit with ctx.Value(frontend) to prevent Wails runtime panic in tests
- [Phase 05-qr-codes-status-indicators]: statusMu is separate from main mu to avoid contention between Hub drain and status updates
- [Phase 05-qr-codes-status-indicators]: dashboardAuth protects the QR endpoint — consistent with /api/sessions auth level
- [Phase 05-qr-codes-status-indicators]: QR endpoint returns 404 for non-enabled sessions — consistent with session auth behavior
- [Phase 05-qr-codes-status-indicators]: Dashboard QR overlay implemented with pure CSS/JS — no external library required
- [Phase 05-qr-codes-status-indicators]: QR button visibility gated on both webEnabled[sessionId] and webServerRunning — prevents GetSessionQRCode calls when server is down
- [Phase 05-qr-codes-status-indicators]: Initial status seeded via GetSessionStatus per restored session — avoids blank dots on app reopen
- [Phase 05-qr-codes-status-indicators]: sessionStatuses cleaned up on tab close — prevents unbounded map growth
- [Phase 05-qr-codes-status-indicators]: requestAnimationFrame defers fit() until browser layout completes — containerRef.clientHeight non-zero before measuring
- [Phase 05-qr-codes-status-indicators]: ResizeObserver on containerRef replaces window resize listener — handles any layout change not just window resize
- [Phase 05-qr-codes-status-indicators]: Watch() uses relay.ParseFrame() guard to skip non-MsgOutput frames before feeding detector
- [Phase 05-qr-codes-status-indicators]: reANSI extended with OSC alternation [^\x07\x1b]*(?:\x07|\x1b\\) to strip Claude Code window title sequences before prompt matching
- [Phase 05-qr-codes-status-indicators]: GET /dashboard serves HTML without dashboardAuth wrapper — dashboard.html JS detects auth state by probing /api/sessions; HTML shell must be publicly accessible for login form to render
- [Phase 06-distribution-cross-platform]: build/darwin/ gitignore negation: use build/darwin/* + !Info.plist pattern to track plists while ignoring binary outputs
- [Phase 06-distribution-cross-platform]: entitlements.plist: network.client + network.server only — no get-task-allow (Apple rejects notarization with it)

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 3]: Linux WebKitGTK fragmentation — webkit2gtk-4.0 vs 4.1 (Ubuntu 22.04 vs 24.04) needs research during Phase 3 planning
- [Phase 4]: Per-OS CA cert trust installation UX underspecified — macOS Keychain, Linux NSS, Windows certutil; needs design pass before Phase 4 execution
- [Phase 5]: Per-CLI status indicator output patterns for Codex, Gemini CLI, OpenCode undocumented — empirical testing needed during Phase 5 planning

## Session Continuity

Last session: 2026-03-19T13:02:19.226Z
Stopped at: Completed 06-01-PLAN.md
Resume file: None
