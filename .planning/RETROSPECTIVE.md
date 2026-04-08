# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — MVP

**Shipped:** 2026-03-19
**Phases:** 6 | **Plans:** 19 | **Commits:** 107

### What Was Built
- Cross-platform PTY process management with auto-detection of 4 AI coding CLIs
- WebSocket fan-out relay with binary framing and bounded scrollback replay
- Wails desktop UI: tabbed xterm.js terminals, session naming, system tray persistence
- Embedded HTTPS server with CA+leaf TLS, bcrypt auth, per-session token links, VPN binding
- QR code generation (desktop modal + web dashboard) with live session status badges
- GitHub Actions CI matrix: macOS (signing/notarization), Linux (WebKitGTK 4.0/4.1), Windows (NSIS)

### What Worked
- Inside-out architecture: PTY → relay → UI → web → features → distribution. Each phase had clear boundaries and testable deliverables
- Interface-first Go design: SessionBackend interface in Phase 1 meant Phase 2+ never touched core types
- Binary framing protocol designed once in Phase 2, used unchanged through Phase 6
- Gap closure plans (05-04 through 05-06) caught integration issues before milestone audit
- 3-day timeline from project init to all 19 plans complete

### What Was Inefficient
- Phase 3 Plan 3 (system tray) took 90min — fyne.io/systray conflicted with Wails AppDelegate, required rewrite to native cgo NSStatusBar
- ROADMAP.md checkboxes drifted from actual status (Phase 2/6 unchecked despite completion)
- Phase 5 needed 3 gap closure plans (05-04, 05-05, 05-06) — integration issues between relay framing + xterm.js fit() + dashboard auth not caught in initial planning
- Nyquist validation never run for any phase — all 6 VALIDATION.md files in draft status

### Patterns Established
- CA+leaf TLS pattern for local self-signed certs (browsers trust WSS after one-time CA install)
- Binary framing protocol: MsgOutput/MsgInput/MsgResize byte prefixes for WebSocket messages
- ResizeObserver + requestAnimationFrame for reliable xterm.js fit() calls
- Build tag pattern: darwin-specific cgo code with Linux/Windows stub files
- Conditional CI signing: builds succeed without secrets, signing/notarization added when configured

### Key Lessons
1. System tray libraries often conflict with desktop framework internals — always check for AppDelegate/main-loop conflicts before committing to a tray library
2. Web dashboard auth creates circular dependencies when login form is behind auth — always serve the HTML shell publicly and let client-side JS handle auth state
3. Status heuristic patterns need empirical data from each CLI — defer non-primary CLIs explicitly rather than shipping empty fallbacks
4. Gap closure plans are valuable — the 3 plans added to Phase 5 fixed real integration issues that would have been painful to find post-ship

### Cost Observations
- Model mix: ~60% sonnet (execution/research), ~30% opus (planning/review), ~10% haiku (synthesis)
- Sessions: ~15 sessions across 3 days
- Notable: Sonnet handled all 19 plan executions; opus used for roadmap, milestone audit, and completion

---

## Milestone: v1.1 — Polish & Build

**Shipped:** 2026-03-20
**Phases:** 7 | **Plans:** 13

### What Was Built
- Terminal layout baseline with CSS flex chain fix and enlarged toolbar buttons (38x38px)
- Per-tab status bar (3 states: inactive/off/on) replacing floating web-serving overlay
- Tabbed settings modal with inline Save Paths and single Close footer
- Per-tab font size adjustment via SHIFT+=/- with key suppression and per-tab state isolation
- New-session modal with agent picker, native OS folder browser, and localStorage last-folder memory
- Tab rename (double-click + right-click context menu) with name propagation to web dashboard via session resolver
- Web dashboard visual redesign: card layout, status dots, CLI badges, TokyoNight palette
- Cross-platform build script (`build.sh`) with macOS code signing and notarization pipeline

### What Worked
- Layout-first phase ordering (Phase 7) prevented false-positive flex issues in subsequent UI phases
- Source-inspection test pattern (?raw imports) scaled well — 73 tests covering xterm.js/Wails components without Canvas/WebGL
- JSX conditional rendering pattern (established Phase 8) reused consistently through Phase 9, keeping React patterns clean
- Phase dependency chain (7→8→10, 7→9, 7→11→12, 12→13) kept integration clean — each phase built on verified foundations
- Build script as final phase validated against the stable codebase — no retroactive fixes needed

### What Was Inefficient
- Phase 13 build.sh Plan 01 took 156 minutes — Docker cross-compile debugging (cross-wails image incompatibility, go-webview2 version conflict) consumed most of the time
- SUMMARY frontmatter inconsistency: 13-01-SUMMARY.md shipped with empty `requirements_completed` — BUILD-01..04 verified working but not recorded
- Phase 09 SUMMARY describes "two-tab layout" but implementation shipped three tabs — documentation drifted from reality
- DetectedCLI.DisplayName never added to TypeScript Wails stub (App.d.ts) — workaround in NewSessionModal works but type is stale

### Patterns Established
- Source-inspection tests via ?raw Vite imports — verify code structure without runtime DOM
- JSX conditionals (not CSS display toggle) for modal/tab content switching
- Per-tab state pattern: `Record<string, T>` keyed by sessionId in App.tsx for font sizes, tab names, etc.
- Session resolver pattern: closure wired once in StartWebServer, reads shared state at query time with correct mutex discipline
- attachCustomKeyEventHandler + return false for intercepting keyboard shortcuts without PTY character leak
- ditto (not zip) for macOS notarization archives — preserves extended attributes

### Key Lessons
1. Docker cross-compilation is the most time-expensive task — image incompatibilities and version conflicts are hard to debug. Budget extra time or pin exact versions.
2. SUMMARY frontmatter should be updated immediately when requirements are verified — retroactive updates are easy to forget.
3. xterm.js FitAddon initial-paint timing is fundamentally racy — multiple strategies (double-RAF, setTimeout, fonts.ready) all fail intermittently. Accept it or solve at a higher level (e.g., visible-only rendering).
4. TypeScript Wails stubs drift from Go structs — regenerating stubs should be a build step, not manual.

### Cost Observations
- Model mix: ~70% sonnet (execution/research/verification), ~25% opus (audit/review/completion), ~5% haiku (synthesis)
- Sessions: ~8 sessions across 2 days (v1.1 phases only)
- Notable: All 13 plan executions handled by sonnet; opus used for milestone audit and completion

---

## Milestone: v1.2 — Tailscale-Only Networking

**Shipped:** 2026-03-23
**Phases:** 5 | **Plans:** 10 | **Commits:** 64

### What Was Built
- Tailscale health check infrastructure — detects installation, connection, and cert readiness with background polling via `local.Client{}`
- Let's Encrypt TLS via Tailscale daemon — replaced self-signed cert system with `GetCertificate` hook and FQDN-based URLs
- Auth layer removal — deleted password auth, per-session tokens, and all auth middleware; tailnet = access control
- Dead code cleanup — removed generic VPN interface picker, `network.go`, and all orphaned frontend bindings
- Health modal — three-state instructional UI with platform-specific guidance, CT disclosure, Check Again auto-dismiss
- Tailscale status indicator in Settings panel replacing removed VPN interface picker

### What Worked
- Safety dependency chain (health→TLS→auth removal→cleanup) — each phase's deletion was safe only after the prior phase confirmed the replacement works. Zero regressions across 5 phases
- Function injection pattern (`statusFunc`) enabled daemon-free unit testing of Tailscale health checks without mocks or interfaces
- Milestone was a net-negative LOC change (removed more code than added) — codebase is simpler after shipping
- Phase 14 and 17 were fast (15min and 5min per plan) — well-scoped, compiler-guided work
- Audit passed 17/17 first attempt — strong requirements traceability throughout

### What Was Inefficient
- Phase 15 Plan 02 took 480s — `StartWebServer(port)` refactor had complex integration with CT disclosure, FQDN derivation, and health gating all in one plan
- Phase 16 Plans 01 and 02 both hit 445-480s — auth removal touched many files (9 and 8 files respectively) across backend and frontend
- Summary frontmatter `one_liner` field consistently null across all 10 summaries — CLI extraction produced no accomplishments for MILESTONES.md
- Nyquist validation partial for all 5 phases — wave_0_complete never reached true

### Patterns Established
- `local.Client{}` zero-value for Tailscale daemon queries — no constructor, no config, just call methods
- Health gate pattern: web server refuses to start without healthy Tailscale state
- CT disclosure via sentinel file (`ct_disclosed`) — one-time acknowledgment without database
- Props-down health state: App.tsx owns all health state, passes to HealthModal and SettingsPanel as props

### Key Lessons
1. Deletion milestones benefit from strict dependency ordering — removing auth before confirming TLS works would have been dangerous
2. Plans that touch 7+ files consistently take 400+ seconds — consider splitting large removal plans by subsystem
3. Summary frontmatter fields should be validated at write time — null `one_liner` across 10 summaries indicates a systematic gap
4. Tailscale `local.Client{}` is remarkably simple to integrate — the Go SDK's zero-value pattern eliminated all configuration boilerplate

### Cost Observations
- Model mix: ~70% sonnet (execution/research), ~25% opus (audit/review/completion), ~5% haiku (synthesis)
- Sessions: ~6 sessions across 6 days
- Notable: Phases 14 and 17 were the fastest in project history (compiler-guided deletion is predictable)

---

## Milestone: v1.3 — CLI + Daemon

**Shipped:** 2026-03-25
**Phases:** 8 | **Plans:** 15 | **Commits:** 101

### What Was Built
- Daemon architecture: SessionEngine extracted into `internal/daemon` with HTTP/JSON API over Unix socket
- Process separation: sessions persist across GUI close/reopen; daemon auto-starts from CLI
- Full CLI binary: 13 commands (new, list, kill, rename, attach, web start/stop/status, serve/unserve, health, qr, settings)
- Interactive PTY attach: raw I/O, detach key, resize propagation, Ctrl-C passthrough, scrollback replay, signal-safe terminal restore
- Service manager integration via kardianos/service for launchd/systemd/Windows SCM
- Windows named pipe fix and graceful GUI startup failure with error banner + retry

### What Worked
- In-process socket first (Phase 19), process separation second (Phase 20) — validated the entire API contract before adding fork complexity
- Function injection patterns: `serviceControlFunc`, `statusFunc` enabled unit testing without mocks or interfaces throughout
- Gap closure phases (25, 26) added after milestone audit caught real integration issues — audit-driven quality improvement
- Attach correctness properties defined upfront (7 properties → 7 tests) made Phase 22 testing systematic
- CLI command pattern (cmd functions return error, io.Writer injection) made all commands independently testable
- 8 days for 8 phases — consistent velocity despite increasing complexity (daemon, process separation, PTY proxy)

### What Was Inefficient
- ROADMAP.md plan checkboxes drifted again (22-02, 23-02, 26-01 unchecked despite completion) — recurring issue from v1.0
- Phase 22 attach tests (Plan 02) took 130 units — polling-based tests for live output required careful timing tuning
- Phase 23 service manager (Plan 01) took 104 units — kardianos/service integration required understanding platform-specific lifecycle nuances
- Nyquist validation still never completed for any phase — all VALIDATION.md files remain draft
- Summary one_liner extraction inconsistent — some phases returned "One-liner:" prefix instead of content

### Patterns Established
- Daemon architecture: SessionEngine + HTTP API + DaemonClient — clean separation of session state from client access
- EnsureDaemon pattern: auto-start daemon from any CLI command if socket not listening
- MsgResize2 (0x11) for client-to-server resize frames — disambiguates from server MsgResize (0x02)
- Detach prefix state machine for terminal escape sequences
- Dual error notification: Wails event (real-time) + stored field (polling) for daemon errors
- Nil-guard pattern for bound methods when daemon unavailable — return zero values matching existing error paths

### Key Lessons
1. In-process validation before process separation eliminates an entire class of debugging — protocol bugs surface immediately without fork/IPC complexity
2. ROADMAP.md checkbox drift is a systemic issue (3 milestones running) — needs automation or should be removed as a manual tracking mechanism
3. PTY proxy correctness requires property-based thinking — define the properties first, then test each one independently
4. kardianos/service abstracts well but KeepAlive/UserService flags matter — wrong defaults cause unexpected restart behavior
5. Gap closure phases after milestone audit are high-value — Phases 25 and 26 fixed real issues that would have shipped as bugs

### Cost Observations
- Model mix: ~65% sonnet (execution), ~30% opus (planning/audit/review/completion), ~5% haiku (synthesis)
- Sessions: ~10 sessions across 8 days
- Notable: Phases 24-26 were fast (2-15 units per plan) — well-scoped, focused work. Phase 19 Plan 01 was the largest (25min) due to full daemon package creation

---

## Milestone: v1.4 — Unified Binary

**Shipped:** 2026-03-25
**Phases:** 3 | **Plans:** 3 | **Tasks:** 6

### What Was Built
- Unified entrypoint: single `agenthub` binary dispatches GUI (no args), CLI (subcommands), and daemon modes from root main.go
- Dead code cleanup: deleted `cmd/agenthub-cli/` package (8 files, 1,559 lines) and scrubbed all references
- Build system hardened: portable BASH_SOURCE path resolution in tests, race detector in CI, build-script CI step on ubuntu-latest

### What Worked
- Three-phase dependency chain (dispatch→delete→verify) was the correct ordering — deleting before verification would have left CI in unknown state
- Phase 28 (deletion) completed in ~5 minutes — well-scoped, compiler-guided work with zero ambiguity
- Milestone audit passed first attempt: 12/12 requirements, 12/12 integration, 4/4 E2E flows — strong traceability throughout
- Entire milestone completed in a single day — focused refactoring milestones are fast when scope is clear

### What Was Inefficient
- v1.4 was essentially a cleanup milestone that could have been folded into v1.3 Phase 27 — separate milestone added overhead (requirements, roadmap, audit) for 3 phases of work
- ROADMAP.md still had unchecked plan checkbox for 29-01 despite completion — the recurring checkbox drift issue from v1.0/v1.3

### Patterns Established
- Unified binary dispatch: `len(os.Args)==1` or `HasPrefix("-")` → GUI; `--help/-h` → usage(); command → runCLI()
- BASH_SOURCE[0] for portable shell script path resolution in test suites

### Key Lessons
1. Cleanup milestones with <5 phases may not need full milestone ceremony — the overhead-to-value ratio is high for small refactoring work
2. ROADMAP.md plan checkbox drift is now confirmed across 4 milestones — this is a systemic process gap, not an oversight
3. Race detector in CI catches real issues — worth the ~30% test runtime overhead

### Cost Observations
- Model mix: ~60% sonnet (execution), ~35% opus (audit/completion), ~5% haiku (synthesis)
- Sessions: ~3 sessions in 1 day
- Notable: Fastest milestone by wall clock — focused scope with no feature additions

---

## Milestone: v1.5 — Bug Fixes & CLI Args

**Shipped:** 2026-03-26
**Phases:** 5 | **Plans:** 6 | **Commits:** 40

### What Was Built
- Backend args wiring: `args []string` threaded through all 5 daemon IPC layers (types, engine, API, client, Wails binding)
- CLI arg passthrough: `splitDashDash` helper + `cmdNew` updated to forward extra args via `--` separator
- Daemon startup fix: poll-first pattern (500ms vs 2s) + PATH augmentation for service-mode agents (nvm, Volta, Homebrew)
- GUI args field in new-session modal with per-agent localStorage persistence and clear button
- Terminal viewport fill: double-rAF fit timing + cols/rows threading from frontend to PTY spawn

### What Worked
- Two independent tracks (args: 30→31→33, fixes: 32+34) could be worked in parallel — dependency graph was well-structured
- TDD for `splitDashDash` and PATH augmentation caught edge cases early (nil vs empty slice distinction, path dedup)
- Milestone audit passed 12/12 first attempt — requirements were well-scoped and each phase had clear boundaries
- Small focused phases (1-2 plans each) executed quickly — most plans completed in under 5 minutes
- Double-rAF pattern resolved a timing issue that had been deferred since v1.1 — sometimes the right fix is to wait for context

### What Was Inefficient
- SUMMARY frontmatter `requirements-completed` field missing across 4 of 5 phases — recurring issue from v1.1-v1.4
- Phase 31 ROADMAP.md plan checkbox shows `[ ]` despite completion (31-01-PLAN.md) — the checkbox drift issue continues
- Nyquist validation partial for 4 of 5 phases — wave_0_complete never reached true (same pattern since v1.0)

### Patterns Established
- `splitDashDash` returns nil (not `[]string{}`) when no `--` present — Go idiom for "not provided" vs "provided but empty"
- Per-agent localStorage key pattern: `agenthub:args:{cliName}` — consistent with `agenthub:lastWorkDir`
- Double-rAF for Wails WebView CSS layout commit timing — necessary when single rAF is too early
- Frontend container dimension estimation: `Math.floor(clientWidth/charWidth)` for PTY initial size

### Key Lessons
1. Threading a parameter through N layers is mechanical but error-prone — integration tests at the boundary layer (HTTP round-trip) catch mismatches that unit tests miss
2. Poll-first, sleep-after is the correct pattern for responsive status displays — sleep-first introduces perceptible lag that makes the app feel broken
3. Service-mode daemons can't rely on shell init files (`.bashrc`, `.zshrc`) — runtime PATH augmentation is necessary for tool resolution
4. Double-rAF is a legitimate timing pattern for WebView-based apps where CSS layout commit is not synchronous with JS execution
5. SUMMARY frontmatter drift is now confirmed across 5 milestones — this is a tooling gap, not a human discipline issue

### Cost Observations
- Model mix: ~65% sonnet (execution), ~30% opus (audit/completion), ~5% haiku (synthesis)
- Sessions: ~4 sessions in 1 day
- Notable: Entire milestone completed in a single day — small focused bug-fix milestones execute fast

---

## Milestone: v1.6 — Terminal Fill Fix v2

**Shipped:** 2026-03-31
**Phases:** 1 | **Plans:** 1 | **Commits:** 14

### What Was Built
- Replaced double-rAF one-shot with bounded rAF retry loop polling `proposeDimensions()` until CharSizeService reports non-zero cell dimensions
- Fixes initial-load terminal fill for Claude, Gemini, and OpenCode CLIs (Codex already worked)
- WebSocket `onOpen` sends terminal dimensions immediately to close race where `fitTerminal()` fires before WS connects

### What Worked
- Root cause diagnosis was precise: `proposeDimensions()` returning `undefined` when CharSizeService hasn't measured cells was the canonical signal — no guesswork needed
- Single-phase milestone with clear success criteria made execution and verification straightforward
- UAT (4/4 passed) confirmed the fix across all CLIs without regression
- Research phase (35-RESEARCH.md) correctly identified the retry pattern before any code was written

### What Was Inefficient
- Milestone had unnecessary overhead for a single-phase bug fix — requirements, roadmap, audit ceremony for 1 phase
- Phase 34 (v1.5) attempted double-rAF which proved insufficient — the research phase in v1.6 could have been done before shipping v1.5
- REQUIREMENTS.md traceability table was never updated from "Not started" despite all checkboxes being checked
- VERIFICATION.md was never created (UAT.md was created instead) — audit flagged this as a gap

### Patterns Established
- `proposeDimensions() !== undefined` as canonical readiness check for FitAddon before calling `fit()`
- Bounded rAF retry loop: `requestAnimationFrame(() => fn(attempt + 1))` with `attempt < MAX_ATTEMPTS` guard
- `onOpen → sendResize(cols, rows)` to sync terminal dimensions on WebSocket connection

### Key Lessons
1. Double-rAF is not a general solution for WebView timing — it works for fast operations but fails when the dependency (CharSizeService font measurement) has variable latency
2. Polling with a readiness predicate (proposeDimensions check) is more robust than fixed-delay approaches for async initialization sequences
3. Single-bug milestones work but the full milestone ceremony (requirements, audit, roadmap) adds overhead disproportionate to scope
4. VERIFICATION.md vs UAT.md naming inconsistency caused a false audit gap — GSD tooling should recognize both

### Cost Observations
- Model mix: ~60% sonnet (execution/research), ~35% opus (audit/completion), ~5% haiku (synthesis)
- Sessions: ~4 sessions across 6 days
- Notable: Actual code work was ~8 minutes; most time spent on planning/research/verification ceremony

---

## Milestone: v1.7 — Daemon UX & Branding

**Shipped:** 2026-04-03
**Phases:** 8 | **Plans:** 10 | **Commits:** 88

### What Was Built
- Platform icon sets (ICNS/ICO/PNG) from 1024x1024 branded logomark with sips+iconutil+ImageMagick pipeline and post-build bundle injection
- Splash screen with StartHidden + OnDomReady lifecycle, static HTML bridge div, React WelcomeTab with branding
- Remote session indicators: web terminal status bar (REST-polled connection state) and CLI attach banner (session name, agent, hostname)
- Daemon Management Panel as in-GUI closeable tab with session list, status dots, kill, and web-serve toggles
- System tray with native cgo NSStatusBar, NSMenuDelegate for dynamic session menu, monochrome template icon, tooltip, error state
- Window hide-on-close lifecycle, LSUIElement (no dock/Cmd+Tab), quit-with-daemon-shutdown from tray
- Gap closures (Phases 42-43): tray startup-failure error icon and GUI hostname forwarding

### What Worked
- Milestone audit mid-development identified two real gaps (tray startup error icon, hostname forwarding) — gap closure phases (42, 43) shipped same day
- Native cgo approach for tray (from v1.0 lesson about systray conflicts) scaled well — NSMenuDelegate, icon state switching, tooltip all worked cleanly
- Props-down pattern (DaemonManagerPanel receives data/callbacks from App.tsx) kept new components testable with zero new Go bindings
- 4-day timeline for 8 phases (10 plans) — consistent velocity despite new platform-specific code (Objective-C, macOS menu bar)
- Parallel-capable phases (36+38, 37 waiting on 36 only) allowed efficient sequencing

### What Was Inefficient
- SUMMARY frontmatter missing `requirements_completed` field for Phases 36 and 37 — the systemic issue from v1.1 continues
- User-approved pivot on splash screen (persistent WelcomeTab instead of auto-dismiss) created audit gap — BRND-02 spec diverged from implementation
- Hardcoded VERSION in WelcomeTab.tsx — will drift on future releases
- Linux/Windows tray stubs are empty no-ops — platform coverage incomplete for tray functionality

### Patterns Established
- NSMenuDelegate `menuWillOpen:` for always-fresh dynamic menus — avoids push-update polling
- ObjC @implementation in separate `.m` files (not cgo blocks) to avoid duplicate symbol linker errors in `go test`
- Split nil-guard pattern: distinguish "not initialized" (skip) from "initialized but failed" (show error)
- Daemon POST /shutdown with response-flush-before-goroutine-exit for clean client disconnect
- 18x18 monochrome template icon generation with Go `image/draw` for pixel-exact letterforms

### Key Lessons
1. Native cgo platform code is maintainable when scoped to a single file with clear build tags — the tray implementation (tray_darwin.go + tray_helpers.m) is self-contained
2. Milestone audits that run mid-development (not just at the end) produce actionable gap closure phases that ship in the same milestone
3. User-approved pivots should be documented in REQUIREMENTS.md immediately — waiting creates audit confusion later
4. REST polling (3s interval) is sufficient for status indicators that don't need sub-second updates — simpler than extending WebSocket protocols
5. Props-down components with zero framework bindings (DaemonManagerPanel pattern) are the most testable — worth the prop-drilling cost

### Cost Observations
- Model mix: ~65% sonnet (execution/research), ~30% opus (audit/review/completion), ~5% haiku (synthesis)
- Sessions: ~6 sessions across 4 days
- Notable: Gap closure phases (42, 43) completed in 3min and 2min respectively — well-scoped, targeted fixes

---

## Milestone: v1.8 — GitHub Distribution & CI/CD

**Shipped:** 2026-04-06
**Phases:** 5 | **Plans:** 9

### What Was Built
- GitHub repository (scottkw/agenthub) with full Gitea history, all v1.0–v1.7 tags, and Go module path rewritten
- release-please auto-versioning with conventional commits and CHANGELOG.md generation
- Multi-platform release pipeline (macOS signed/notarized DMG, Windows EXE+NSIS, Linux tar.gz+deb, SHA256 checksums)
- Homebrew cask tap (scottkw/homebrew-agenthub) with auto-update distribute.yml workflow
- WinGet distribution infrastructure (winget-releaser CI job, WINGET_TOKEN PAT, winget-pkgs fork, populate-manifests.sh helper)
- Packaging templates (Homebrew cask template, 3-file WinGet manifests at schema 1.12.0)

### What Worked
- Phase dependency chain (migration→CI→release→Homebrew→WinGet) was strictly ordered — each phase built on verified outputs
- dev-browser automation for GitHub PAT creation saved manual steps and kept the workflow in-context
- gh CLI for secrets management was faster and more reliable than browser-based UI navigation
- Research phases for each phase caught key constraints early (winget-releaser regex, notarization delay retries, sparse-checkout for winget-pkgs)
- 3-day timeline for 5 phases — CI/CD infrastructure milestones execute quickly when dependencies are clear

### What Was Inefficient
- REQUIREMENTS.md traceability table showed "Pending" for phases 45-47 despite completion — checkbox/status updates weren't propagated
- ROADMAP.md plan checkboxes for phases 45-47 still unchecked despite summaries existing — the recurring drift issue
- Phase 48 Plan 02 Task 2 (manifest submission) deferred because no release exists yet — dependency on release pipeline not yet exercised
- Phase 47 VERIFICATION.md has 7 human_needed items that could have been automated (file existence checks, grep patterns)

### Patterns Established
- vedantmgoyal9/winget-releaser with restrictive `installers-regex` to match only the NSIS installer (not bare EXE)
- nick-fields/retry for checksums.txt download (handles notarization delay)
- Sparse-checkout for large upstream repos (winget-pkgs) — minimizes disk usage for manifest submission
- populate-manifests.sh: sed-based template population with v-prefix validation

### Key Lessons
1. CI/CD milestones are primarily about external integration (GitHub Actions, Homebrew tap, WinGet submission) — code changes are minimal but configuration correctness is critical
2. ROADMAP.md/REQUIREMENTS.md status drift is confirmed across 8 milestones now — the gsd-tools `roadmap update-plan-progress` command helps but isn't called consistently by executors
3. Human-action checkpoint plans should explicitly document their prerequisite state — Plan 48-02 Task 2 was blocked by "no release exists" which wasn't in the plan's depends_on
4. dev-browser automation for GitHub web UI tasks is effective for repetitive credential/setup workflows

### Cost Observations
- Model mix: ~65% sonnet (execution), ~30% opus (orchestration/completion), ~5% haiku
- Sessions: ~3 sessions across 3 days
- Notable: Phase 46 (release pipeline) was single-plan with highest complexity — release.yml touches 5 platform legs

---

## Milestone: v1.9 — Remote Sessions & App Polish

**Shipped:** 2026-04-08
**Phases:** 6 | **Plans:** 14 | **Commits:** 111

### What Was Built
- Standard macOS app menus (File, Edit, Window, Help) with Cmd+C/V clipboard in terminal tabs and build-time version injection via ldflags
- Tailscale peer discovery (`internal/tailnet`) with injectable deps, concurrent probe pool (cap 5), and daemon `GET /tailnet/peers` with 30s thundering-herd-safe cache
- Auto-update checker polling GitHub releases on startup + hourly, notification banner in WelcomeTab, Help menu "Check for Updates" item
- Remote Sessions GUI panel with tailnet peer grouping, loading states, 30s auto-refresh, and one-click browser open
- CLI remote sessions: unified `agenthub list` with HOST column, `agenthub attach hostname:session-id` via WSS relay
- Tailscale onboarding: platform-specific install commands with copy-to-clipboard, macOS auto-install via Homebrew with streaming progress, HTTPS cert setup guide

### What Worked
- Milestone audit passed first attempt: 17/17 requirements, 6/6 phases, 4/4 E2E flows — strong requirements traceability
- Injectable dependency patterns (statusFunc, detectFunc, discoverFunc) enabled comprehensive unit testing across all 3 new Go packages without live Tailscale/GitHub
- Props-down component pattern (RemoteSessionsPanel, HealthModal enhancements) kept components testable with zero new framework bindings
- 2-day timeline for 6 phases (14 plans) — fastest multi-feature milestone by velocity
- Parallel plan execution across phases worked well (e.g., Phase 51 plans 02+03 simultaneously, Phase 52 plans 01+02 simultaneously)
- All Nyquist validation compliant (6/6) — first milestone with 100% Nyquist compliance from the start

### What Was Inefficient
- SUMMARY frontmatter missing `requirements_completed` for 9 of 17 requirements (Phases 49, 51) — recurring systemic issue since v1.1
- Summary one_liner extraction still inconsistent — some returned prefixes ("One-liner:", "Task 1:") instead of clean descriptions, causing messy MILESTONES.md auto-generation
- Dead `installError` state in App.tsx — 3 lines of unused code from Phase 54 (benign but untidy)
- STATE.md accumulated merge conflict markers (<<<<<<< HEAD) — not caught during phase transitions

### Patterns Established
- `goruntime` alias for stdlib `runtime` when `wails/v2/pkg/runtime` already imported as `runtime`
- `cmd.Stderr = cmd.Stdout` for single-goroutine subprocess output streaming via Go pipes
- `loading+peers.length>0` pattern: show stale data during refresh instead of spinner flicker
- `fetchPeerSessionsWithClient` / `cmdAttachRemoteWithClient` injectable client pattern for httptest TLS in CLI tests
- `listOutput` struct for JSON output with grouped local/remote sections (breaking change from flat array)

### Key Lessons
1. Injectable dependency patterns (statusFunc, detectFunc, discoverFunc) are now the standard approach for all new Go packages — proven across 6 milestones
2. Parallel plan execution within and across phases significantly reduces wall-clock time — v1.9 was 2 days for work that would have taken 4+ sequentially
3. SUMMARY frontmatter one_liner extraction needs tooling fixes — the auto-generated MILESTONES.md accomplishments are consistently low quality
4. All Nyquist validation being compliant from the start (vs retroactive in v1.8) shows the validate-phase workflow is maturing
5. STATE.md merge conflicts from concurrent phase transitions need a cleanup step in the transition workflow

### Cost Observations
- Model mix: ~70% sonnet (execution), ~25% opus (audit/completion), ~5% haiku (synthesis)
- Sessions: ~4 sessions across 2 days
- Notable: Fastest multi-feature milestone — 6 phases in 2 days, consistent with increasing automation and parallelization

---

## Milestone: v1.10 — Collapsible Sidebar Navigation

**Shipped:** 2026-04-08
**Phases:** 2 | **Plans:** 3

### What Was Built
- Collapsible left sidebar with Heroicons SVG icons replacing top toolbar action buttons
- App layout restructured to flex-row (Sidebar + app__content)
- All 5 navigation items (Home, Remote, Sessions, New Tab, Settings) wired to corresponding tabs/panels
- Tab bar cleaned up — action buttons removed, retains session tabs only
- Sidebar state persistence via localStorage

### What Worked
- TDD approach: 13 RED tests for Sidebar written first, then implementation made them pass
- Source-inspection (?raw) pattern continued for Phase 56 nav wiring tests — avoids Wails runtime mocking
- Small milestone scope (2 phases, 1 day) executed cleanly with minimal overhead
- Milestone audit passed on first attempt — 11/11 requirements, 6/6 E2E flows

### What Was Inefficient
- Nyquist validation still partial (draft VALIDATION.md files for both phases) — systemic gap continues
- 56-01-SUMMARY.md used `provides` in dependency_graph instead of `requirements_completed` frontmatter — recurring SUMMARY drift

### Patterns Established
- Heroicons as the standard icon library for sidebar navigation (replacing Unicode characters)
- Sidebar as the primary navigation mechanism; tab bar for session management only

### Key Lessons
- UI restructuring milestones are fast when the component architecture is clean — Sidebar was a pure addition, TabBar cleanup was pure subtraction
- @heroicons/react provides a scalable icon system that avoids the Unicode icon inconsistency problems

### Cost Observations
- Model mix: ~60% sonnet (execution), ~35% opus (audit/completion), ~5% haiku (synthesis)
- Sessions: ~2 sessions in 1 day
- Notable: Smallest feature milestone yet — 2 phases, 3 plans, completed in a single day

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Commits | Phases | Key Change |
|-----------|---------|--------|------------|
| v1.0 | 107 | 6 | Initial process — inside-out architecture, gap closure plans |
| v1.1 | ~50 | 7 | Source-inspection tests, JSX conditional pattern, layout-first ordering |
| v1.2 | 64 | 5 | Safety dependency chain, deletion-focused milestone, function injection testing |
| v1.3 | 101 | 8 | In-process-first daemon validation, audit-driven gap closure phases, property-based PTY testing |
| v1.4 | ~20 | 3 | Unified binary dispatch, focused cleanup milestone, race detector in CI |
| v1.5 | 40 | 5 | Args threading through IPC layers, double-rAF terminal fix, poll-first status pattern |
| v1.6 | 14 | 1 | Bounded rAF retry loop replacing double-rAF, proposeDimensions() readiness gate |
| v1.7 | 88 | 8 | Native cgo tray, NSMenuDelegate dynamic menus, split nil-guard pattern, mid-milestone audit gap closure |
| v1.8 | ~18 | 5 | CI/CD infrastructure, GitHub Actions pipelines, package manager distribution, dev-browser automation |
| v1.9 | 111 | 6 | Remote session discovery (GUI+CLI), auto-update checker, Tailscale onboarding, app menus, parallel plan execution |
| v1.10 | 21 | 2 | Heroicons sidebar, flex-row layout restructure, tab bar cleanup, TDD-first UI milestone |

### Cumulative Quality

| Milestone | Go Tests | JS Tests | LOC | Tech Debt Items |
|-----------|----------|----------|-----|-----------------|
| v1.0 | 53+ (race-clean) | 0 | ~8,100 | 14 (0 blockers) |
| v1.1 | 55+ (race-clean) | 73 | ~9,956 | 9 (0 blockers) |
| v1.2 | 55+ (race-clean) | 73+ | ~8,846 | 3 (0 blockers, info-only) |
| v1.3 | 80+ (race-clean) | 73+ | ~12,619 | 4 (0 blockers, bookkeeping) |
| v1.4 | 194 (race-clean) | 73+ | ~12,771 | 4 (0 blockers, deferred) |
| v1.5 | 200+ (race-clean) | 80+ | ~13,400 | 7 (0 blockers, documentation/test gaps) |
| v1.6 | 200+ (race-clean) | 150 | ~73,000 | 3 (0 blockers, stale comments/docs) |
| v1.7 | 200+ (race-clean) | 150+ | ~15,100 | 4 (0 blockers; WelcomeTab auto-dismiss, hardcoded VERSION, Linux/Windows tray stubs, SUMMARY frontmatter) |
| v1.8 | 200+ (race-clean) | 150+ | ~41,000 | 3 (0 blockers; WinGet first submission deferred, ROADMAP checkbox drift, VERIFICATION human items) |
| v1.9 | 200+ (race-clean) | 160+ | ~41,000 | 4 (0 blockers; SUMMARY frontmatter gaps, dead installError state, unused CheckForUpdates TS binding, STATE.md merge markers) |
| v1.10 | 200+ (race-clean) | 268 | ~41,200 | 1 minor (SUMMARY frontmatter drift in 56-01) |

### Top Lessons (Verified Across Milestones)

1. Interface-first design pays off — define contracts before implementations (v1.0, v1.3)
2. Gap closure plans catch integration issues that initial planning misses (v1.0, v1.3)
3. Phase ordering matters — foundation phases (layout, contracts) before feature phases prevents cascading issues (v1.1, v1.3)
4. Source-inspection tests are a viable pattern when runtime testing is impractical (Canvas/WebGL, Wails bindings) (v1.1)
5. Deletion milestones benefit from strict dependency ordering — confirm replacements work before removing originals (v1.2)
6. In-process validation before process separation eliminates an entire class of debugging (v1.3)
7. Milestone audits that drive gap closure phases are high-value — they catch real bugs before shipping (v1.3)
8. SUMMARY frontmatter `requirements-completed` drift is systemic — needs tooling validation, not discipline (v1.1-v1.5)
9. Small focused bug-fix milestones (5 phases, 1 day) execute fast — overhead is minimal when scope is clear (v1.5)
10. Single-bug milestones add disproportionate ceremony overhead — consider folding into the previous milestone or using a lighter process (v1.6)
11. Polling with readiness predicates beats fixed-delay timing for async initialization — `proposeDimensions()` is more robust than double-rAF (v1.6)
12. Native cgo platform code is sustainable when scoped to single files with build tags — avoids library conflicts while maintaining full platform control (v1.0, v1.7)
13. Mid-milestone audits produce higher-quality gap closures than end-of-milestone audits — gaps are still fresh context (v1.7)
14. User-approved pivots need immediate REQUIREMENTS.md annotation — delayed documentation creates false audit gaps (v1.7)
15. CI/CD milestones are configuration-heavy, code-light — correctness depends on external integration verification, not unit tests (v1.8)
16. dev-browser automation is effective for GitHub web UI credential/setup workflows — saves manual context switching (v1.8)
17. Human-action checkpoint plans need explicit prerequisite state documentation — "no release exists" blocked Task 2 but wasn't in depends_on (v1.8)
18. Parallel plan execution within and across phases is the single biggest velocity lever — v1.9 shipped 14 plans in 2 days vs v1.3's 15 plans in 8 days (v1.9)
19. 100% Nyquist compliance from the start (not retroactive) is achievable and should be the default — v1.9 was first milestone to achieve this (v1.9)
20. SUMMARY frontmatter one_liner auto-extraction is unreliable across 10 milestones — needs tooling fix or manual curation at completion time (v1.1-v1.9)
