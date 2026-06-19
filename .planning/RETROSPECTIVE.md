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

## Milestone: v1.11 — Local Network & UX Polish

**Shipped:** 2026-04-10
**Phases:** 6 | **Plans:** 9 | **Commits:** 57

### What Was Built
- Claude Code native installer detection (`~/.local/bin` as first AugmentServicePath candidate) for Anthropic native installer discovery
- Sidebar "New Tab" label renamed to "New Session" with aria-label and vitest verification
- Settings converted from modal overlay to singleton sidebar tab (SettingsTab.tsx), consistent with Home/Remote/Sessions panels
- Web server auto-starts on daemon launch; new sessions auto-enabled for web serving in both Tailscale and local modes
- Local network fallback: self-signed TLS (P256 + IP SAN), HTTP Basic Auth with generated password, LAN IP selection excluding Tailscale CGNAT, persistent nudge banner, password display in Settings with click-to-copy
- Frontend webEnabled seeding chain restored across all 5 IPC layers (Go bindings → TypeScript types → React state)
- Tech debt cleanup: deleted orphaned SettingsPanel/HealthModal components, fixed 11 stale test assertions

### What Worked
- Milestone audit identified two real gaps (SERVE-02 frontend wiring, orphaned components) — gap closure phases (61, 62) shipped same day as audit
- Quick task (260409-vop) accelerated Phase 58+60 frontend work by resolving HealthModal flash and Settings modal→tab conversion ahead of formal phases
- Mode-aware server dispatch (Tailscale vs local) with daemon-lifetime password generation was clean architecture — single `AutoStartWebServer` entry point handles both paths
- All 6 phases Nyquist compliant from the start — second consecutive milestone with 100% Nyquist compliance
- 2-day timeline for 6 phases (9 plans) — consistent with v1.9/v1.10 velocity

### What Was Inefficient
- SUMMARY frontmatter missing `requirements_completed` for phases 59, 60, 61 — the systemic issue continues (now 8 milestones running)
- Quick task 260409-vop rewrote SettingsPanel→SettingsTab before Phase 58 formally planned it — caused downstream confusion (60-03-SUMMARY references "SettingsPanel.tsx" when SettingsTab.tsx was the actual file)
- Phase 61 was a 1-plan fix for a wiring gap that Phase 59 should have caught — the `WebEnabled` field missing from `app.go SessionInfo` was a type-chain oversight
- `retryInit()` missing `webServerRunning` override that `init()` has — near-identical functions diverged during Phase 61 implementation

### Patterns Established
- Mode-aware web server: `AutoStartWebServer(mode, password)` dispatches to `startTailscale()` or `startLocal()` based on Tailscale health check
- P256 self-signed certs with IP SAN for local network HTTPS — Chrome-compatible, no CA install needed
- HTTP Basic Auth middleware with `crypto/rand` base64url password, daemon-lifetime scope
- `LocalNetworkBanner` as conditional sibling to `app__content` — never inside terminal flex container
- Singleton tab pattern (find-or-add) now standard for all non-session tabs (Home, Remote, Sessions, Settings)

### Key Lessons
1. Quick tasks that overlap with planned phases create naming/reference confusion in downstream plans — either fold quick task work into the phase or clearly document the overlap
2. Type-chain wiring (Go struct → daemon API → Wails binding → TypeScript type → React state) needs an explicit checklist — omitting one layer creates subtle bugs that pass unit tests
3. Near-identical functions (`init`/`retryInit`) inevitably diverge — extract shared logic or use a single function with a retry flag
4. Milestone audits that run after all phases complete (not just mid-milestone) continue to find real gaps — Phases 61 and 62 were both audit-driven
5. SUMMARY frontmatter `requirements_completed` drift is confirmed across 8 milestones — at this point, the field should be auto-populated from VERIFICATION.md during phase completion

### Cost Observations
- Model mix: ~65% sonnet (execution), ~30% opus (audit/completion), ~5% haiku (synthesis)
- Sessions: ~4 sessions across 2 days
- Notable: Quick task 260409-vop provided a head start on frontend work, but created plan reference drift that required manual cleanup

---

## Milestone: v1.12 — UI/UX Polish

**Shipped:** 2026-04-11
**Phases:** 4 | **Plans:** 4 | **Commits:** 48

### What Was Built
- CSS flexbox centering for sidebar icons in collapsed 48px rail
- 8px symmetric terminal padding with dynamic background matching terminal theme
- 157 xterm-theme color schemes with live apply, localStorage persistence, and Settings > Appearance tab
- Web server URL actions: open in browser, copy to clipboard, inline QR code in Settings

### What Worked
- Research-driven phase ordering (risk-ascending, dependency-driven): CSS-only phases first, Go-touching phase last — zero regressions
- All 4 phases CSS/frontend-heavy, allowing rapid execution with immediate visual feedback
- Milestone audit passed on first try (after one minor doc cleanup commit) — 8/8 requirements satisfied with zero gaps
- Full Nyquist compliance across all 4 phases from the start
- xterm-theme library provided 157 themes with zero custom infrastructure needed

### What Was Inefficient
- Phase 65 (terminal theming) required 6 fix commits for WebGL texture atlas cache, Wails WebKit listbox behavior, and settings tab state persistence — more than the feature code itself
- SUMMARY frontmatter one-liner extraction continues to produce malformed output (phases 63 and 66 had blank or garbled one-liners)
- Phase 66 QR code needed Go backend method despite frontend-focused milestone — could have been pre-identified during research

### Patterns Established
- `clearTextureAtlas()` + `refresh()` pattern for WebGL theme switching — required for any xterm.js theme change
- `fs.readFileSync` for CSS source-inspection tests (not Vite `?raw` imports) — established as project-wide pattern
- QR cache lifetime tied to server lifecycle (not UI toggle state) — avoids unnecessary re-fetches

### Key Lessons
1. WebGL renderers cache glyph textures aggressively — theme changes require explicit atlas clearing, not just `options.theme` updates
2. Wails WebKit `<select>` elements behave differently from Chromium — `onInput` handler needed alongside `onChange` for reliable selection
3. CSS-only phases (63, 64) execute in minutes; phases touching Go backend (66) take 3-5x longer even for simple features
4. Small UI polish milestones (4 phases, 2 days) are efficient — focused scope with clear visual acceptance criteria
5. SUMMARY frontmatter one_liner extraction is still unreliable — confirmed across 13 milestones now; needs tooling fix

### Cost Observations
- Model mix: ~60% sonnet (execution), ~35% opus (audit/completion/retro), ~5% haiku (synthesis)
- Sessions: ~3 sessions across 2 days
- Notable: Phase 63 (CSS centering) completed in ~5 minutes — fastest phase in project history

---

## Milestone: v1.13 — Cross-Platform Fixes & UX

**Shipped:** 2026-04-12
**Phases:** 3 | **Plans:** 5 | **Commits:** 11

### What Was Built
- Cross-platform system tray: Linux D-Bus StatusNotifierItem (GNOME/KDE/XFCE) and Windows Shell_NotifyIcon Win32 API with shared menu helpers (tray_common.go)
- Comprehensive agent CLI discovery: daemon PATH augmentation extended for snap, flatpak, cargo, npm, pnpm, and Windows native installer paths
- Tailscale detection across platforms: Homebrew, system package manager, Windows default install location
- macOS Homebrew install command: single copyable `brew tap scottkw/agenthub && brew install --cask agenthub`
- Settings scrollable layout: single page with h3 section headers replacing sub-tab navigation

### What Worked
- All 3 phases fully independent (no cross-phase dependencies) — enabled parallel execution
- Build-tagged platform files (path_windows.go, path_other.go) followed existing codebase convention — no new patterns needed
- Shared tray_common.go extracted common logic (BuildMenuItems, TrayTooltip) so platform files are thin wrappers
- godbus/dbus/v5 was already an indirect dependency via Tailscale — promotion to direct added zero new deps
- Milestone audit passed on first try with 13/13 requirements satisfied at code level

### What Was Inefficient
- SUMMARY frontmatter `one_liner` field blank across all 5 summaries — milestone completion CLI extracted nothing useful (14th consecutive milestone with this issue)
- Phase 67 Plan 02 (Windows tray) completed in 3 minutes of AI time but requires human UAT on Windows — asymmetric verification effort
- 9 human UAT items accumulated (Linux/Windows tray + Settings visual) — cannot be automated from macOS development host

### Patterns Established
- D-Bus StatusNotifierItem + dbusmenu protocol for Linux system tray — freedesktop.org standard
- Shell_NotifyIcon + HWND_MESSAGE + WM_USER message loop for Windows tray
- Graceful degradation when StatusNotifierWatcher absent — export SNI service for auto-discovery hosts
- Runtime PNG-to-HICON conversion via GDI CreateDIBSection (no .ico files needed)

### Key Lessons
1. Platform tray implementations are thin when shared helpers handle menu construction — tray_common.go made Linux and Windows implementations small and testable
2. Build-tagged files are the right pattern for platform-specific PATH lists — runtime.GOOS checks would complicate testing
3. Human UAT items for cross-platform work are unavoidable when developing on a single platform — track them as tech debt, not blockers
4. Settings layout simplification (tabs → scroll) reduced code complexity while improving UX — fewer React components, fewer state transitions
5. SUMMARY frontmatter one_liner extraction is confirmed broken — needs tooling fix or deprecation after 14 milestones of empty output

### Cost Observations
- Model mix: ~70% sonnet (execution), ~25% opus (audit/completion), ~5% haiku
- Sessions: 2 sessions across 1 day
- Notable: Fastest 3-phase milestone — all phases completed same day, total ~30 minutes AI execution time

---

## Milestone: v1.14 — UI Polish

**Shipped:** 2026-04-14
**Phases:** 4 | **Plans:** 9 | **Commits:** 64

### What Was Built
- Sidebar icons fixed in horizontal position during collapse/expand via 48px icon slot margin
- OpenCode theme integration: managed tui.json, OPENCODE_TUI_CONFIG env injection, SIGUSR2 broadcast for live theme switching
- WCAG AA contrast: all 32 `#565f89` text declarations replaced with `#9aa5ce` across every GUI surface
- Theme picker curated from 157 to 138 themes via WCAG-derived readability criteria, localStorage fallback guard for stale names

### What Worked
- Phase 71 (OpenCode theming) had the highest complexity — 5 plans covering test stubs, env injection, integration testing, manual UAT, and gap closure (SIGUSR2 broadcast). The gap closure plan (71-05) was identified during verification, not after-the-fact audit
- CSS-only fix for sidebar icon stability (Phase 70) — clean, minimal, 5-minute execution
- WCAG contrast tests written RED-first (Phase 72) caught exact number of failing selectors before the fix
- Theme audit (Phase 73) used computed contrast ratios from research phase to build deterministic allowlist — no manual curation needed

### What Was Inefficient
- `gsd-tools audit-open` has a runtime bug (ReferenceError: output is not defined) — pre-close artifact audit had to be done manually
- SUMMARY frontmatter one_liner extraction continues to produce empty output — 15th milestone with this broken tooling
- Phase 70 roadmap checkbox marked `[ ]` despite being complete (same drift pattern as earlier milestones)

### Patterns Established
- Per-agent env injection via CreateRequest.Env for external CLI configuration (OpenCode tui.json)
- SIGUSR2 signal pipeline: frontend → Wails binding → daemon HTTP → engine broadcast → per-session signal
- ALLOWED_THEMES module pattern: sorted allowlist in separate file, imported by name only (not full theme objects)
- localStorage fallback guard: validate stored value against current allowlist, silent fallback to default

### Key Lessons
1. External CLI integration (OpenCode) is harder than internal features — requires managed config files, env injection, and process signals instead of direct API calls
2. WCAG contrast testing with computed ratios is automatable — RED-first test approach catches exact scope of changes needed
3. Theme curation criteria must be evidence-based (computed contrast ratios), not subjective — 138/157 pass rate validates the criteria aren't too aggressive
4. Signal-based live configuration updates (SIGUSR2) are the right pattern when the target process doesn't expose a config reload API
5. `daemonConfigDir()` duplication is an acceptable cost of Go's package boundary — internal packages genuinely cannot import main

### Cost Observations
- Model mix: ~70% sonnet (execution), ~25% opus (audit/completion), ~5% haiku
- Sessions: 2 sessions across 2 days
- Notable: 4-phase milestone completed in 2 days with 9 plans — steady velocity matching v1.13

---

## Milestone: v2.0 — Multi-Client, CLI UX & TUI Mode

**Shipped:** 2026-04-16
**Phases:** 5 | **Plans:** 16 | **Commits:** 116

### What Was Built
- Multi-client WebSocket fan-out: per-subscriber metadata, independent scrollback, read-only mode, max-wins resize arbitration, viewer count API
- CLI status bar: DECSTBM scroll-region bar with session context, live viewer count via MsgMeta push frames, --status-top flag, clean teardown
- TUI foundation: Bubble Tea v2 terminal UI with session list, heuristic status glyphs, web server status footer, help overlay, adaptive color
- TUI session operations: attach (tea.Exec suspend/resume), create modal, kill confirmation, inline rename — shared internal/attach/ package
- TUI remote & QR: unified local+remote session list with tailnet peer grouping, ASCII QR code overlay for session web URLs

### What Worked
- TDD discipline on Phase 74 (Hub extensions): RED tests first, GREEN implementation, clean refactor — caught resize arbitration edge cases early
- Shared internal/attach/ package extraction (Phase 77): CLI and TUI attach logic unified in one place, no duplication
- Phase ordering (74→75→76→77→78): each phase's output was directly consumed by the next — status bar viewer count needs Hub metadata, TUI needs status bar package, etc.
- 3-day execution for 5 phases/16 plans — fastest multi-phase milestone yet, enabled by well-defined requirements and phase boundaries
- MsgMeta protocol extension cleanly reused existing binary framing — one new frame type (0x04) integrated across relay, webserver, and CLI attach
- Bubble Tea v2 tea.Exec pattern for TUI→attach→TUI transitions worked perfectly first try — clean suspend/resume with no terminal state corruption

### What Was Inefficient
- SUMMARY frontmatter requirements_completed missing across 4 of 5 phases — metadata debt continues despite being a known issue since v1.1
- REQUIREMENTS.md checkboxes only checked for 4/23 items (TUI-03..06) despite all being satisfied — traceability table Pending/Complete status not updated during execution
- Phase 74 and 75 human UAT items (7 total) not resolved during milestone — live terminal/browser testing deferred
- summary-extract CLI tool produced poor one-liners for many plans — auto-extraction quality remains unreliable

### Patterns Established
- MsgMeta push frame pattern for broadcasting session metadata changes to all subscribers — reusable for future metadata (e.g., input locking)
- lockedWriter pattern for serializing concurrent writes to a shared terminal stdout — applicable anywhere multiple goroutines write to the same fd
- Priority-based key dispatch (5-level) in TUI: modal states > overlay states > main view — extensible for future TUI features
- FetchRemoteFn callback injection in TUI model — decouples tailnet discovery from TUI package, enables test isolation

### Key Lessons
1. Requirements traceability metadata (SUMMARY frontmatter, REQUIREMENTS.md checkboxes) needs automated tooling — manual updates continue to drift across every milestone
2. Existing binary framing protocols are cheap to extend — adding MsgMeta was a one-liner type constant plus handler, not a protocol redesign
3. Charm Bubble Tea v2 tea.Exec is the correct pattern for shelling out from TUI — alternatives (os.Exec, subprocess) would corrupt terminal state
4. Unified lists (local+remote in one view) are better UX than split panels in terminal UI — simpler mental model, less navigation
5. ASCII QR codes need terminal size guards — go-qrcode output can exceed small terminal viewports; 55×25 minimum prevents truncation

### Cost Observations
- Model mix: ~65% sonnet (execution), ~30% opus (planning/audit/completion), ~5% haiku
- Sessions: ~4 sessions across 3 days
- Notable: 16 plans in 3 days — highest daily throughput of any milestone (5.3 plans/day)

---

## Milestone: v2.1 — Bug Fixes & UX

**Shipped:** 2026-04-17
**Phases:** 4 | **Plans:** 8 | **Commits:** 91

### What Was Built
- Settings persistence: CLI paths and Tailscale path saved to daemon settings.json via Wails bindings with three-state Save button and native file/folder picker
- Tailscale 4-state health check cascade with platform-specific binary detection (Homebrew, Snap, Flatpak, Windows) and diagnostics checklist
- Banner vertical stacking with BannerStack container, independent dismiss handlers, and 200ms exit animation
- Start-minimized-to-tray with Settings > Behavior toggle, persisted preference, and domReady conditional WindowShow gate

### What Worked
- All 4 phases shared the daemon `daemonSettings` struct for persistence — single storage layer, no per-feature duplication
- Phase dependency chain (79→80→81→82) ensured each phase could build on the previous without integration surprises
- 4-state Tailscale health cascade replaced the binary installed/not-installed check — actionable diagnostics for every failure mode
- Non-optimistic toggle pattern (wait for API success before updating UI state) prevented stale toggle states
- `toggleLoaded` flash-gate pattern prevented visual flash on page load — reusable for any async-loaded toggle
- Milestone audit passed on first run (12/12 requirements, 14/14 integration points, 5/5 E2E flows)

### What Was Inefficient
- gsd-tools `summary-extract` one-liner auto-extraction continues to produce unusable output — had to manually curate accomplishments for MILESTONES.md (same issue since v1.14)
- gsd-tools `audit-open` command has a ReferenceError bug (`output is not defined`) — had to manually check for open artifacts
- Phase 80 Tailscale health check required decoupling `loadSettingsFromDisk()` to prevent startMinimized from interfering with CLIPaths loading — integration discovered during Phase 82 execution

### Patterns Established
- Three-state button pattern (idle → loading → confirmed → idle) for save actions — reusable for any async form submission
- `toggleLoaded` gate for async-loaded toggles — prevents off→on flash by deferring render until initial value resolves
- `domReady` conditional WindowShow — checks persisted preferences before showing main window
- BannerStack flex-column container with per-banner dismiss state — extensible for future notification types
- Platform-specific binary detection in separate `tailscale_paths.go` file following existing build-tagged pattern

### Key Lessons
1. Bug-fix milestones with tight scope (4 phases, 12 requirements) execute cleanly in 2 days — overhead is proportional to scope
2. Non-optimistic UI patterns (wait for server confirmation) are worth the extra complexity — they prevent impossible UI states
3. Shared persistence layers (`daemonSettings`) across phases need integration testing when multiple phases write to the same struct
4. gsd-tools summary-extract and audit-open need maintenance — both produced errors/garbage output at milestone close

### Cost Observations
- Model mix: ~60% sonnet (execution), ~35% opus (planning/audit/completion), ~5% haiku
- Sessions: ~3 sessions across 2 days
- Notable: 8 plans in 2 days — 4 plans/day, consistent with recent milestones

---

## Milestone: v3.0 — Session Lifecycle & TUI Polish

**Shipped:** 2026-04-19
**Phases:** 4 | **Plans:** 9 | **Commits:** ~40

### What Was Built
- Settings UI alignment: unified Paths table with single column headers, consistent 12px description text, shared section header CSS
- Session auto-close: exit detection via hub.Done() PTY EOF, pollSessionStatus stopped-state detection, session:exit Wails events, ExitToast and ExitCountdownBanner components, 5s countdown with Keep Open cancel, auto-close toggle in Settings
- Quit confirmation modal: app:quit-requested event from both window close and tray Quit, QuitConfirmModal with session count and colored status dots, Quit GUI Only with macOS native notification via UNUserNotificationCenter, Quit Everything with daemon shutdown
- TUI visual polish: TokyoNight hex palette with 22+ lipgloss LightDark adaptive tokens, two-pane sidebar+content layout, bordered session frames via injectBorderTitle(), per-agent colored badges for 6 CLIs, focus-aware key routing

### What Worked
- Phase dependency ordering (83→84→85→86) meant each phase had stable foundations; no cross-phase integration surprises during execution
- hub.Done() channel as exit signal guaranteed all PTY output was drained before exit detection — no truncation edge cases
- app:quit-requested event pattern unified both quit paths (window close + tray) into a single frontend handler — simple and testable
- Source-inspection tests continued to scale well — 28 QuitConfirmModal tests, 30 ExitToast/ExitCountdownBanner tests, all without Canvas/WebGL
- Milestone audit caught a real test regression (FLOW-01) and a TS interface gap (MISS-02) before shipping — both fixed in-session

### What Was Inefficient
- gsd-tools `summary-extract` one-liner extraction still produces unusable output — manually curated MILESTONES.md accomplishments again
- gsd-tools `audit-open` still has the `output is not defined` ReferenceError — manually verified no open artifacts
- SUMMARY frontmatter continues to lack `requirements_completed` field across all 9 SUMMARY files — systemic documentation gap
- REQUIREMENTS.md checkboxes never updated during phase execution — all 12 still unchecked despite being satisfied

### Patterns Established
- Circuit breaker pattern for pollSessionStatus — exits after N consecutive errors instead of looping for full deadline
- UNUserNotificationCenter cgo wrapper with build-tagged no-op stub — reusable for any macOS-specific native notification
- injectBorderTitle() for lipgloss frame titles — string-replaces border chars to inject styled text
- sidebarTabs mapping array (not tabID cast) — explicit index-to-tab mapping safe against iota reordering

### Key Lessons
1. Milestone audits with integration checking catch real bugs — the poll circuit breaker regression would have broken CI
2. TS interface drift from Go structs continues to be a recurring issue — needs automation or a build-step check
3. Two-day milestones (4 phases, 9 plans) execute cleanly when scope is well-defined from GitHub issues
4. cgo Objective-C wrappers in separate .m files (not inline cgo blocks) avoid duplicate symbol linker errors during go test

### Cost Observations
- Model mix: ~55% sonnet (execution/research), ~40% opus (planning/audit/completion), ~5% haiku
- Sessions: ~4 sessions across 2 days
- Notable: 9 plans in 2 days — consistent with v2.1 velocity (8 plans/2 days)

---

## Milestone: v3.1 — Security Hardening

**Shipped:** 2026-05-03
**Phases:** 4 (87-90) | **Plans:** 18 | **Closes:** GitHub Issue #35

### What Was Built
- Capability-based session authorization with read/write/admin tokens (Phase 87)
- WebSocket handshake security with byte-for-byte Origin allowlist; cross-site upgrades return 403 at handshake (Phase 88)
- Vendored xterm.js under `web/vendor/xterm/`; strict CSP across all embedded HTML routes (`script-src 'self'`, `connect-src 'self' wss://<host>`, `style-src 'self' 'unsafe-inline'` per D-09 amendment for xterm runtime style injection); zero CDN fetches verified across 1779 requests (Phase 89)
- Release pipeline hardening: SHA-pinned third-party Actions; build tools pinned via `tools.go`; release.yml split into validate → build-{macos,windows,linux} → sign-macos (required-reviewer gate) → publish; SLSA L2 attestations verified before codesigning (Phase 90)

### What Worked
- Capability tokens designed once (Phase 87) carried unchanged through Phases 88-90 — no protocol breakage during the security cascade
- WS Origin allowlist as byte-for-byte match (no subdomain wildcards, no regex) — kept the contract observable and testable
- Vendoring xterm.js under `web/vendor/xterm/` with a vendor_drift_test.go gate as a CI-enforced contract — set the pattern that v3.2 generalized for every addon
- Pipeline E2E proven through 5 rc cycles before v3.1.0 final tag — caught codesigning gate misconfig, CSP `unsafe-inline` carve-out, and Wails dev-mode origin handling before public release
- D-09 CSP amendment: `style-src 'self' 'unsafe-inline'` was added once xterm runtime style injection was discovered — minimal carve-out, never `script-src`

### What Was Inefficient
- Multiple rc cycles (rc1 → rc5) needed to stabilize the release pipeline — would have been faster to dry-run the full sign+publish chain on a throwaway tag first
- Phase 88 needed two Wails-origin patterns (production-darwin `wails://wails` + dev-mode) — caught late in UAT against rc3; could have been front-loaded by an explicit "all environments" enumeration in the discuss phase
- CSP discovery loop (the D-09 amendment) was reactive — xterm runtime style injection should have been audited in research before Phase 89 shipped the first strict CSP

### Patterns Established
- Capability-token middleware on relay WSS + REST (the v3.1 security-base pattern reused by v3.2 `/api/plugin-config`)
- Byte-for-byte Origin allowlist with explicit Wails-origin variants (production-darwin + dev-mode)
- Vendored-only-no-CDN discipline + `vendor_drift_test.go` (generalized in v3.2 for every `@xterm/addon-*`)
- D-09 CSP minimal-carve-out style: never `script-src`; `'unsafe-inline'` for styles only with a documented reason

### Key Lessons
- Security-hardening milestones have a long tail of UAT (cross-browser + Wails dev vs prod + Tailscale + local-network-fallback) — budget for it explicitly, don't expect to sign off at first rc
- Adding a CI gate for a contract is cheaper than enforcing the contract by convention — vendor_drift_test.go caught addon version drift that humans would miss
- Capability-string design (read/write/admin) is forward-compatible with later capability additions — v3.2 plugin-config capability fits the same model without protocol churn

### Cost Observations
- Sessions: ~6 sessions across ~13 days (2026-04-20 → 2026-05-03)
- Notable: long tail of rc-cycle UAT after code-complete; pipeline correctness depends on external integration verification

---

## Milestone: v3.2 — Plugin Suite

**Shipped:** 2026-05-12
**Phases:** 8 (92-99) | **Plans:** 44 | **Commits:** 269 | **Closes:** GitHub Issue #36

### What Was Built
- Plugin Settings Foundation: daemon `PluginSettings` source of truth, defaults-merge constructor, Wails RPC + `settings:plugins` runtime event, 8-toggle PluginsSection, v3.1→v3.2 `schemaVersion: 2` migration (Phase 92)
- Vendoring discipline + web parity: generalized `vendor_drift_test.go` CI gate for every `@xterm/addon-*`; vendored 3 already-shipping addons (webgl/unicode11/clipboard) for the web page; capability-gated `/api/plugin-config` REST + SSE; two-useEffect TerminalPanel hot-swap pattern; WebGLRecoveryBanner with context-loss / software-rasterized variants (Phase 93)
- Scrollback search (Cmd-F find bar) on desktop + web: regex/case/word toggles, persisted defaults, 200ms slide animation, `SetSearchConfig` sub-key RPC, `seededRef` one-shot pattern, 10k-line perf gate (Phase 94)
- Web-Links security hardening: strict scheme allowlist (`https`/`http`/`mailto`), Cmd-click macOS / Ctrl-click elsewhere, OSC 8 hover-href, `LinkConfirmPopover` for IDN + 30-entry typosquat list, Wails `BrowserOpenURL` desktop / `_blank`+`noopener,noreferrer` web (Phase 95)
- Inline images + CSP audit: sixel via `@xterm/addon-image`, `'wasm-unsafe-eval'` CSP amendment 2 (audited and minimal), 16 MB per-tab `storageLimit`, byte-fidelity multi-client relay test (Phase 96)
- Save Terminal As + OSC 9;4 progress: SerializeAddon → TabBar "Save Terminal As…" with secrets warning; ProgressAddon → per-tab `.tab__progress` underline + atomic `SetTrayProgress(quartile)` debounced 200ms (Phases 97 + 98)
- Release gate: `PluginToggleBanner` one-shot toasts for non-hot-swappable plugins, inline `<details>` disclosures persisting via sub-key RPCs immediately, cross-browser Playwright e2e (Chromium + Firefox + WebKit), GitHub Actions e2e workflow (Phase 99)

### What Worked
- Phase 92 ships ZERO addon-loading work — pure foundation (daemon struct + Wails RPC + event + settings UI shell + migration test). Lets Phase 93+ focus on hot-swap mechanics without entangling settings plumbing
- `vendor_drift_test.go` generalization from addon-fit-only → every `@xterm/addon-*` caught real version drift during Phase 93 — would have shipped mismatched bundles without it
- Two-useEffect TerminalPanel pattern: clean split between hot-swap addons (webgl/clipboard/web-links/search/image) and mount-only buffer-interpretation plugins (unicode11) — no flickers, no scrollback corruption
- Sub-key RPCs (`SetSearchConfig`/`SetWebLinksConfig`/`SetImageConfig`) persisting immediately without "Save Plugins" — fine-grained UX matches user expectations for disclosure toggles
- `LinkConfirmPopover` with risk-specific copy (osc8 / idn / typosquat) — defense-in-depth made visible to users, not just internal
- Phase 99 gap-closure (99-06) caught disclosure checkbox visibility regression via real-DOM render test — closed 99-UAT Tests 5/6 in one wave
- Cross-browser Playwright (Chromium + Firefox + WebKit) zero CSP violations on all three engines — true parity gate, not just "works in Chrome"
- 269 commits in 9 days — high velocity sustained by clean phase boundaries + parallel plan execution within phases

### What Was Inefficient
- 9 UAT scenarios deferred to v3.3 because raw shell session type doesn't exist in v3.2 — chafa sixel, OSC 9;4 progress, large scrollback paste, IDN URL test data all require a raw shell PTY which AgentHub doesn't offer. Should have surfaced the shell-session backlog dependency in discuss-phase BEFORE planning the milestone audit acceptance criteria
- 6 polish-grade tech debt items (mailto detection, IDN popover, find-bar Esc-after-toggle, find-bar slide-out, iTerm2 IIP, jsdom localStorage) discovered during human UAT walkthrough on 2026-05-11 — could have been caught earlier by an explicit cross-phase Wails-only UAT pass, not just per-phase UAT
- Phase 94 needed 2 gap-closure plans (94-06 + 94-07) to flip SC-2 and SRC-04 from PARTIAL to VERIFIED — initial planning underestimated find-bar animation + sub-key RPC plumbing
- VALIDATION.md frontmatter was authored at phase-start (Wave 0 contract) but never re-stamped at phase-end for 92/94/97/99 — process gap, not coverage gap, but it creates audit noise. Tooling could auto-flip the frontmatter when VERIFICATION.md status is `pass`/`complete`/`approved`
- Plan 95-06 (OSC 8 secondary provider) deferred to v3.3 because upstream `@xterm/addon-web-links` lacks the API — discovered only during the 95-01 spike. Could have been surfaced in 95-RESEARCH.md before planning

### Patterns Established
- Foundation phase pattern (Phase 92): ship plumbing + Settings UI shell + migration test, no feature loading. Decouples feature waves from settings plumbing
- Two-useEffect TerminalPanel hot-swap pattern (live-swap vs mount-only) — reusable for any future plugin
- `seededRef` one-shot useEffect for async-loaded UI seed (useRef(false) + early-return on seeded/null/mid-open) — re-usable for any PluginSettings-driven UI
- Sub-key RPCs that persist immediately (separate from bulk Save) — natural UX for disclosure-style sub-config
- Capability-gated `/api/plugin-config` REST + SSE pattern — reuses v3.1 capability middleware, scales to other multi-client config surfaces
- Cross-browser Playwright e2e as a release gate (Chromium + Firefox + WebKit zero CSP violations) — true parity gate
- Generalized vendor_drift_test.go as a load-bearing CI gate for every `@xterm/addon-*` — supply-chain contract

### Key Lessons
- A foundation phase that ships ZERO addon-loading work is high leverage — it isolates settings plumbing from feature mechanics. Worth the extra phase budget
- CSP carve-outs should be minimal and audited per addon (`'wasm-unsafe-eval'` only for the image addon's sixel decoder) — never blanket
- `storageLimit` for image addon (16 MB) is load-bearing — naïve 100 MB default × 8 tabs = OOM. Override upstream defaults whenever they assume single-tab usage
- Cmd-click on macOS / Ctrl-click elsewhere for link activation is non-negotiable — single-click never activates a link by default, no exceptions
- IDN + typosquat confirmation is a user-visible security feature, not internal — make the threat model legible
- Sub-key RPCs that persist immediately decouple disclosure sub-config from the main Save button — natural UX, no save-button confusion
- Long-tail UAT (iPad Safari, real Tailnet, raw shell PTY) needs a dedicated "what's blocked" inventory BEFORE planning the audit acceptance criteria, not after
- Cross-phase Wails-only UAT (a single end-to-end walkthrough of all plugins together) catches polish items that per-phase UAT misses

### Cost Observations
- Sessions: ~12 sessions across 9 days (2026-05-03 → 2026-05-12)
- Commits: 269 (highest single-milestone commit count to date)
- LOC delta: +22,697 across 117 source files
- Notable: highest velocity milestone in absolute terms; 44 plans in 9 days (4.9 plans/day) — exceeds v1.9 record (14 plans in 2 days = 7/day) on raw plans-per-day but at far larger plan size

---

## Milestone: v3.4 — File Browser (Read-Only) + TUI Parity

**Shipped:** 2026-05-21
**Phases:** 5 (118-122, including audit-driven Phase 122 insert) | **Plans:** 20/21 | **Commits:** 176 | **Closes:** GitHub Issues #62 + v3.4 slice of #64

### What Was Built
- TOCTOU-safe sandboxed filesystem API: new `internal/files/` package with `os.OpenRoot`-based `Sandbox` (Go 1.24+ kernel-level path resolution); HTTP `Handler` with Range support + 5 MB read cap + 0-byte short-circuit + darwin-filter; `FuzzSandboxPath` merge gate against 40+ payload corpus (Phase 118 FS-01..14)
- `files.read` capability bit + `HasPerm` whole-token comma-split + `requireFilesRead` middleware separated from `requireCapability` switch; session-owner default ON, viewer default OFF (Phase 118 FS-10..13)
- Webserver mounts `/api/files/*` under `requireFilesRead`; daemon socket auth-less by design; viewer-403 body contains `"files.read"`; cross-browser Playwright reports zero CSP violations (Phase 119 WEB-01..05)
- React `FileBrowserTab` (desktop + web-share): breadcrumb-bounded navigation, `react-markdown` + `remark-gfm` (NO `rehype-raw`), image previews via direct `<img src>`, `/` filter (NOT Cmd-F — preserves xterm.js scrollback search), Range-capable Download, `PermissionDeniedTakeover`, web-mode URL-param detection, 45-cell cross-browser Playwright merge gate (Phase 120 UI-01..14)
- TUI Files view: lipgloss two-pane + viewport, ALL filesystem I/O via `tea.Cmd` (static-grep gate enforces), `charmbracelet/glamour` for markdown, left-truncated status line, `/` filter, `?` help (Phase 121 TUI-01..10 local half)
- Cross-surface remote-session file browse parity (audit-driven Phase 122): daemon proxy + `RemoteCapStore`, `ExchangeJoinCodeAtURL` (303 redirect cap exchange), desktop GUI `RemoteJoinCodeModal` + `EnableWebSharingTakeover`, TUI `RemoteFilesClient` implements same `FilesClient` interface as daemon client — same `handleFilesKey` pipeline drives local AND remote (Phase 122 REMOTE-01..05)

### What Worked
- `os.OpenRoot` (Go 1.24+) as the sandbox primitive — Phase 118 RESEARCH explicitly cited CVE-2026-27976 / CVE-2026-43998 against the legacy `EvalSymlinks` + `Open` two-step. `FuzzSandboxPath` 60s fuzz against 40+ payloads found zero crashes
- Static-grep gates as merge enforcement (`TestHasPerm_NoStringsContains`, `TestRequireCapability_UnchangedByPhase118`, `TestFiles_NoSyncFSCalls`, Phase 122's 15 source-grep gates) — invariants enforced at CI time, not human review
- `FilesClient` interface in TUI (Phase 121-02) — enables Phase 122 `RemoteFilesClient` to slot in without touching `handleFilesKey` or the Update pipeline. Interface-first design paid off cleanly
- 3-observer cross-surface parity test (Phase 122-05): daemon-proxy Go + tui.RemoteFilesClient Go + Playwright HTTPS browser observe byte-identical responses against shared fixture — strongest possible automated evidence pending two-machine UAT
- 176 commits in 2 days — high velocity sustained by 4 distinct cross-phase parallelization opportunities (Phase 120 and 121 paralleled after Phase 119; Phase 122 plans paralleled across desktop GUI + TUI workstreams)
- Audit-driven mid-milestone phase insertion pattern (Phase 122) — Phase 121 local-only scope cut surfaced cross-surface remote-browse as a release-blocking gap. Same pattern proven in v3.3 Phases 107/108. Insertion happens in `<24h with full plan-execute-verify discipline

### What Was Inefficient
- Plan 122-01 first implementation had a structural gap — required 122-01-recovery plan to complete; recovery commit landed cleanly but added 4 extra commits (recovery + 3 follow-ups) on top of the original Plan 01 budget. Indicates the original Plan 01 spec under-specified the daemon proxy + cap-store boundary
- Phase 120 needed gap-closure Plan 06 after 120-VERIFICATION Human Verification #2 surfaced web-share parity gap (App.tsx didn't honor URL-param-driven web mode). Should have been caught in 120-RESEARCH or pre-merge cross-mode UAT
- 3 manual UATs deferred at audit close (TD-1 Wails click-path, TD-2 visual TokyoNight, TD-3 22-step two-machine tailnet) — release-eligible, but the v3.4 tag ships ahead of perceptual + tailnet evidence. Cross-surface byte-equivalence (3 observers) is the strongest automation; visual + perceptual UATs are inherently human
- Phase 121 had 7 review-fix follow-ons (CR-01 + WR-01..06) after initial Plan 03 merge — code review caught real issues (Ctrl+C in filter mode, generation counter for stale-result discard, viewport scroll on Up/Down). Some of these should have been pre-merge concerns

### Patterns Established
- **`os.OpenRoot`-based sandbox + 40+ payload `FuzzSandboxPath` corpus** as the filesystem-sandbox merge gate (load-bearing across any future filesystem-touching feature)
- **Static-grep gates as CI-time invariant enforcement** (`HasPerm_NoStringsContains`, `Files_NoSyncFSCalls`, capability-route-count grep) — invariants outlive human reviewers
- **Interface-first `FilesClient`** in TUI for swappable local vs remote backends — pattern reusable for any future surface that swaps daemon-direct vs network-mediated I/O
- **3-observer cross-surface parity test** (daemon-proxy Go + same-codebase-different-transport Go + browser Playwright) — strongest automated proof of network-stack byte-equivalence
- **Daemon-proxy + in-memory `RemoteCapStore` for cap material isolation** — keeps tailnet caps out of renderer process; only daemon holds them
- **Audit-driven mid-milestone phase insertion** reaffirmed (Phase 122 after Phase 121 local-only scope) — pattern now expected at every milestone close as standard sweep

### Key Lessons
- Go 1.24's `os.OpenRoot` is non-negotiable for filesystem sandboxes — the two-step `EvalSymlinks` + `Open` pattern is documented-exploitable. Any new filesystem-touching feature should default to `OpenRoot` going forward
- Static-grep gates pay for themselves on the first invariant-violation prevented. `HasPerm_NoStringsContains` blocks the `"no-files.read"` substring false-positive class; `Files_NoSyncFSCalls` blocks the Bubble Tea render-loop freeze class
- Interface-first design for cross-surface clients (`FilesClient` here, `Hub` in v2.0, `DaemonClient` since v1.3) pays off when a new transport surfaces — the consumer code doesn't change
- Cross-surface parity (GUI/TUI/web/CLI) is a release-blocking contract. Phase 121's local-only scope cut would have shipped v3.4 with a parity gap had the audit-driven insertion pattern not been treated as standard
- Cap material belongs in the daemon, not the renderer. Phase 122's `RemoteCapStore` enforces this trust boundary cleanly
- 0-byte file special-case via `http.ServeContent` (golang/go#54794) needs an explicit unit test — Go stdlib's default behavior returns 416, which is wrong for an empty file read

### Cost Observations
- Sessions: ~6 sessions across 2 days (2026-05-20 → 2026-05-21)
- Commits: 176 (high velocity; exceeds v3.3.1 by 10×)
- LOC delta: +42,159 / -1,963 across 197 files (incl. `.planning/`)
- Notable: 2-day shipping cycle is the second-fastest milestone in absolute time (after v1.6's single-phase day). 5 phases × 2 days = 2.5 phases/day, sustained by Phase 120 / 121 / 122 parallelization
- Audit-driven Phase 122 insertion completed end-to-end in <12 hours (plan → execute → verify) — pattern is now reliable enough to bake into milestone close as a sweep step

---

## Milestone: v3.5 — File Browser: Write Operations & Editor

**Shipped:** 2026-06-15
**Phases:** 6 (123-128) | **Plans:** 27/27 | **Commits:** 238 | **Closes:** GitHub Issues #63 + #64 (umbrella #24 pending GUI on-ramp)

### What Was Built
- TOCTOU-free write primitives on the v3.4 `os.OpenRoot` sandbox: atomic temp+sync+rename (no `O_TRUNC`), rename validates source AND destination, recursive delete, mkdir, multipart upload — shell-RC denylist on every write path + 60s `FuzzSandboxWrite` merge gate (0 crashes); auth-less daemon Unix-socket write routes; folds in TD-4 + TD-5 (Phase 123 FSW-01..12)
- Opt-in `files.write` capability (never default-on; web-share viewers need a further explicit grant) behind `requireFilesWrite` + CSRF Origin check, `schemaVersion: 4` migration (Phase 124 CAP-01..10)
- Vendored CodeMirror 6 editor (zero new CSP amendments; Monaco rejected for `worker-src blob:`) replacing v3.4 plain-text preview — syntax highlighting by extension, atomic Cmd/Ctrl+S with `If-Match`/412 conflict detection, dirty-state guard, full create/mkdir/delete/rename/cross-dir-move/single+multi-upload (drag-drop, XHR progress, 409/413) suite; 51/51 cross-browser write e2e zero CSP (Phase 125 EDIT-01..13)
- TUI write parity via `$EDITOR` shell-out (`e` key, suspend/`tea.ClearScreen`/resume) + `d`/`r`/`m` ops; `FilesClient` interface grew 4 → 8 methods so one pipeline drives local AND remote; upload descoped (#82) (Phase 126 TUIW-01..07)
- Dedicated web-share write security-hardening phase: denylist + symlink-escape / privilege-escalation / CSRF / concurrent-write audits; `SECURITY` artifact; fully automated (Phase 127 SEC-01..07)
- Remote tailnet peer write parity proven byte-identical by 3 independent observers (daemon-proxy Go + `tui.RemoteFilesClient` Go + Playwright HTTPS), 405 peer-outdated / 401 cap-expired mappings, Phase 122 read-regression guard (Phase 128 RMW-01..06)

### What Worked
- The v3.4 unifying contracts extended verbatim: `FilesClient` grew 4 → 8 methods, `FilesApiClient` switches local vs remote by `pathPrefix` — one write pipeline across GUI/TUI/web with no per-surface forks. Interface-first design (established v1.3 `DaemonClient`, v2.0 `Hub`, v3.4 `FilesClient`) paid off again
- Security-first phase ordering: write primitives + denylist + `FuzzSandboxWrite` (Phase 123) landed and fuzz-proven BEFORE any network-facing write surface was exposed (Phase 124+). Load-bearing foundation before exposure
- `files.write` never default-on, with a second explicit grant for web-share viewers — the most-exposed surface is opt-in twice. CSRF Origin check + `requireFilesWrite` mirror the proven v3.4 `requireFilesRead` separation
- CodeMirror 6 vendored with zero new CSP amendments — the research-ratified Monaco rejection (avoids `worker-src blob:`) kept the hard-won v3.1/v3.2 CSP posture intact
- 3-observer parity test reused from v3.4 Phase 122 for the write matrix — strongest automated evidence available pre-tailnet

### What Was Inefficient
- **The 98/100 automated integration score was blind to a 4-layer desktop-GUI remote-browse breakage** — the RMW/integration suite exercised the webserver/fixture surface but NEVER the relay loopback the Wails GUI actually uses. The deferred two-machine UAT (run the day of tag) uncovered a non-functional GUI remote path: relay routes never mounted (`58af6d6`), discovery probe rejecting cap-protected peers (`3508bd7`/#84), accept-dns prerequisite (#83), session-enumeration-vs-cap conflict (#86). A relay-surface regression gate now exists, but the gap should have been caught before tag
- The milestone tagged (v3.5, 2026-06-15) with the remote-browse GUI on-ramp NOT shippable — #86 + #83 deferred to v3.5.1. Data path proven live; the discover→list→pick UX is not. Umbrella #24 stays open
- Live desktop/TUI visual UATs (Phase 125 editor render + Tab/Cmd-V, Phase 126 `$EDITOR` suspend-resume, Phase 124 warning banner) deferred to a manual batch — logic + colorblind tokens source-verified, but live render is unverified at tag
- Nyquist `VALIDATION.md` frontmatter for Phases 123/125/126/127 reads `nyquist_compliant:false` despite green automated coverage — advisory bookkeeping debt

### Patterns Established
- **Relay-surface regression gate** (`internal/relay/server_files_test.go`, `internal/daemon/relay_remote_files_test.go`) — remote features MUST test the relay loopback the Wails GUI uses, not just the webserver/fixture surface. Direct response to the v3.5 blind spot
- **Security foundation before exposure** — write primitives + fuzz gate land before any network-facing write route; denylist enforced on every write path, not at the route layer
- **Double opt-in for the most-exposed surface** — `files.write` off by default for owner-grant, and a further explicit grant required for web-share viewers
- **Atomic write = temp + fsync + rename, never `O_TRUNC`** — and rename validates both endpoints; the sandbox write contract for any future filesystem mutation
- **`If-Match`/412 optimistic-concurrency for editor saves** — detects the AI-agent-edits-underneath-you race that auto-save would corrupt (auto-save ruled an explicit anti-feature)

### Key Lessons
- **Automated integration scores measure only the surfaces the tests drive.** A 98/100 PASS meant nothing for the relay loopback path because no test hit it. When a feature has multiple transport surfaces (webserver vs relay loopback vs daemon socket), each needs its own integration coverage — a high score on one is not evidence for another
- Deferred manual UATs are not a formality — the two-machine UAT was the ONLY thing that caught the remote-browse breakage. Physical-machine UATs that can't be automated must still gate the *feature* even if they don't gate the *tag*
- Extending a proven contract (v3.4 read stack → v3.5 write stack) is dramatically cheaper and safer than a parallel design — 4 → 8 methods, one pipeline, byte-identical parity reused wholesale
- CSP discipline compounds: choosing CodeMirror over Monaco at plan time (research-ratified) preserved zero-amendment CSP that took v3.1/v3.2 multiple phases to establish
- A milestone can ship its data path while a UX on-ramp stays deferred — but be explicit (umbrella #24 stays open, v3.5.1 scoped) rather than letting the tag imply the feature is usable

### Cost Observations
- Sessions: ~8 sessions across 2 days (2026-06-14 → 2026-06-15)
- Commits: 238 (sustained high velocity; matches v3.4-class output)
- LOC delta: ~14.9K source LOC across 86 files (excluding `.planning/`)
- Notable: 2-day shipping cycle again (6 phases / 2 days = 3 phases/day), but a full extra UAT+fix day on tag day (relay fixes `58af6d6`/`3508bd7`/`e45ccba`) — the deferred UAT cost was paid in a post-"complete" scramble, not avoided

---

## Milestone: v3.5.1 — Remote Browse Completion + Release-Gate Fix

**Shipped:** 2026-06-16
**Phases:** 2 (129-130) | **Plans:** 7 | **Requirements:** 11/11

### What Was Built
- #87 per-path single-winner `WriteFileAtomic` lock (package-level keyed `sync.Map` mutex) — deterministic If-Match release gate (RACE-01..03).
- #83 `accept-dns=false` actionable error + proactive `RemoteBrowseDNSWarning` banner (DNS-01..03).
- #86 tailnet-trusted metadata-only `GET /api/sessions/meta` discovery + honest per-peer panel states + relay-surface regression test (RB-01..05). Retired umbrella epic #24 — discover→list→pick→browse proven live on a two-machine tailnet.

### What Worked
- Resolving #87/#86 design decisions up front (before planning) kept both phases unambiguous.
- Wave-0 RED-first + mandatory relay-surface coverage (RB-05) — the v3.5 audit lesson — held: tests hit the GUI's actual loopback path, not just the webserver fixture.
- Two-machine live UAT was decisive: it caught 5 bugs that all green unit tests + a passing audit missed (DNS-banner mount gating, App.js runtime RPC binding absent, write-toggle non-rehydration, join-code not shown as text, "5-char" copy wrong). Mocked tests + tsc gave false confidence on the binding/render layer.

### What Was Inefficient
- The hand-maintained wailsjs bindings (`App.js` vs `App.d.ts`) drifted — plan added the type but not the runtime export; only a production build (not tests) surfaced it. Consider a check that `App.js` and `App.d.ts` exports match.
- wails CLI v2.10.2 vs repo-pinned v2.12.0: every `wails dev`/`build` rewrote go.mod down; required repeated reverts. Reconcile the CLI/pin.
- `wails build` produced a malformed ad-hoc signature ("damaged" on the 2nd Mac) — needed manual Developer ID re-sign for the UAT.

### Patterns Established
- Stale LSP "undefined symbol" diagnostics after multi-file edits / from removed agent worktrees are env artifacts — verify with real `go build`, not the editor.
- For colorblind UAT: state distinctions verified text-first at source (UI audit Color 4/4) + confirmed by the operator reading labels in live use.

### Key Lessons
- Live multi-surface UAT remains the only reliable catch for binding/render/gating gaps that unit tests mock away. Keep it in the loop before declaring a GUI on-ramp done.

## Milestone: v3.6 — Hub (Session Grid / Control Room)

**Shipped:** 2026-06-19 (tag v3.6, closes #78)
**Phases:** 5 (131-135) | **Plans:** 26 | **Requirements:** 39/39

### What Was Built
- Top-level **Hub** control-room surface: unified grid of live local + remote session cards, auto-grouped by working directory plus user-defined named groups (drag-to-assign, localStorage-persisted), live-count status filter bar + `/` search (Phase 131-132).
- Throttled per-card mini-previews (single shared poll interval, never a live xterm) and remote tailnet-peer sessions merged into the same grid (Phase 132).
- Colorblind-safe attention system: waiting/errored sessions float-to-top + pulse (amber `#e0af68`, BellAlertIcon + aria, never color alone), 1s-debounced FLIP reorder, collapsed-group badge, reduced-motion aware (Phase 133).
- Card→modal interaction with shared-element animation — interactive terminal modal + briefing modal (real terminal tail), plus a cap-gated relay WebSocket proxy for remote-session terminals (Phase 134).
- Accessibility hardening across the surface — keyboard operability, reduced-motion, colorblind-safe cues (Phase 135).

### What Worked
- The Wave-0 "close the Go data gap first" pattern (carried from prior milestones) caught the WorkDir/ExitCode/Duration/ViewerCount propagation gap before the frontend depended on it.
- Source-level / computed-style verification (the correct method for a colorblind owner) gave objective evidence for animation/color UATs (`hub-attn-pulse` active, exact hex, badge presence) that eyeballing could not.

### What Was Inefficient
- The live UATs stalled across **two prior passes** on a misdiagnosed symptom: a stale homebrew **v3.5.1 daemon** was holding the socket, so v3.6 fields (exit codes) never surfaced and exited sessions appeared to "vanish." Cost real time before the env was the actual fault. Lesson: when a daemon-backed UAT behaves impossibly, verify the **daemon binary identity** first.
- Driving terminal *input* for live UATs is genuinely hard: the wails-dev bridge has no PTY, and local-mode web-share blocks programmatic WS input (Basic-auth + CSRF-Origin). Owner-in-the-loop keystrokes (one `/model`) + automated Hub observation was the workable split.

### Patterns Established
- **Decouple "produce the state" from "observe the surface":** produce session states via the daemon socket (or one owner keystroke), observe the Hub on the `:34115` bridge via dev-browser. Sidesteps the no-PTY-on-bridge limitation for everything except literal terminal input.
- **Verify-by-composition for end-to-end UATs** whose atoms are each independently proven (ATTN-05 = live attention pipeline + unit-tested status-clear + Phase-134 modal pass) — a continuous manual click-through adds no coverage.
- **Don't fabricate sign-off to clear a gate.** A guardrail correctly blocked auto-stamping `human_verified`; the honest path was the owner's actual sign-off, recorded with accurate scope (automation/computation vs eyeballed).

### Key Lessons
- A milestone-close audit can surface a *real product bug*, not just doc gaps: CARD-08 error-exit was non-functional on macOS because the natural-exit path never called `cmd.Wait()` (go-pty has no unix `waitOnContext`). Fixed with `Session.ReapNaturalExit()` (TDD, `-race` clean) — finding it required actually trying to produce the state live.

### Cost Observations
- The bulk of the close-out cost was live-UAT environment archaeology (daemon identity, web-share auth layers), not the code. A 5-file daemon fix + 3 doc commits was the durable output.

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
| v1.11 | 57 | 6 | Local network fallback, auto-serve sessions, settings-as-tab, audit-driven gap closure (61, 62) |
| v1.12 | 48 | 4 | UI polish milestone, risk-ascending phase ordering, xterm-theme integration, WebGL atlas clearing pattern |
| v1.13 | 11 | 3 | Cross-platform tray (D-Bus SNI + Shell_NotifyIcon), shared tray helpers, build-tagged platform paths, settings scroll refactor |
| v1.14 | 64 | 4 | Per-agent env injection (OpenCode tui.json), SIGUSR2 signal broadcast, WCAG contrast audit, curated theme allowlist |
| v2.0 | 116 | 5 | Multi-client fan-out, DECSTBM status bar, Bubble Tea v2 TUI, shared attach package, unified local+remote list |
| v2.1 | 91 | 4 | Settings persistence, 4-state Tailscale health, banner stacking, start-minimized-to-tray, non-optimistic toggle pattern |
| v3.0 | ~40 | 4 | Session auto-close (hub.Done() exit signal), quit confirmation modal (event-based), TUI two-pane layout, circuit breaker poll pattern |
| v3.1 | ~50 | 4 | Capability tokens, byte-for-byte WS Origin allowlist, vendored xterm + strict CSP (D-09 style-src carve-out), SHA-pinned release pipeline + SLSA L2 |
| v3.2 | 269 | 8 | Foundation phase (zero feature loading), two-useEffect TerminalPanel hot-swap, `seededRef` one-shot pattern, generalized vendor_drift_test, cross-browser Playwright as release gate, sub-key RPCs for disclosure persistence |
| v3.5 | 238 | 6 | Relay-surface regression gate (integration scores blind to untested transport surfaces), security foundation before exposure, double opt-in for web-share writes, atomic temp+fsync+rename write contract, `If-Match`/412 editor saves, CodeMirror-over-Monaco for zero-CSP-amendment *(v3.3/v3.3.1/v3.4 rows pending backfill)* |

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
| v1.11 | 200+ (race-clean) | 268+ | ~43,000 | 7 (0 blockers; SUMMARY frontmatter gaps, retryInit asymmetry, stale closure risk, init/retryInit duplication) |
| v1.12 | 200+ (race-clean) | 270+ | ~43,700 | 4 cosmetic (SUMMARY frontmatter `provides` vs `requirements-completed`, VERIFICATION doc string) |
| v1.13 | 200+ (race-clean) | 270+ | ~43,000 | 11 (9 human UAT, 2 metadata; 0 blockers) |
| v1.14 | 200+ (race-clean) | 280+ | ~23,000 | 3 (0 blockers; SUMMARY one_liner extraction broken, ROADMAP checkbox drift, audit-open bug) |
| v2.0 | 280+ (race-clean) | 280+ | ~26,000 | 21 (0 blockers; 19 metadata drift, 2 scope deferrals) |
| v2.1 | 280+ (race-clean) | 456 | ~26,300 | 2 (0 blockers; info-level .catch patterns) |
| v3.0 | 300+ (race-clean) | 524 | ~30,000 | 2 cosmetic (autoCloseRef refresh, TUI stopped-session glyph) |
| v3.1 | 300+ (race-clean) | 524+ | ~30,000 | 3 (0 blockers; distribution 91-A/B/C deferred, D-09 CSP carve-out documented, Wails dev-mode origin pattern noted) |
| v3.2 | 300+ (race-clean) | 600+ | ~52,700 | 15 (0 blockers; 6 polish items, 9 UAT scenarios deferred to v3.3 — all blocked on shell-session feature, plus 7 quick-task ghosts) |
| v3.5 | 300+ (race-clean) + `FuzzSandboxWrite` | 600+ (51/51 cross-browser write e2e) | ~67,600 | 1 critical (remote-browse GUI — found by deferred UAT, partly fixed `58af6d6`/`3508bd7`, #86/#83 deferred v3.5.1) + operator UATs + Nyquist frontmatter (123/125/126/127) *(v3.3/v3.3.1/v3.4 rows pending backfill)* |

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
20. Quick tasks that overlap with planned phases create naming/reference confusion — fold into the phase or document the overlap explicitly (v1.11)
21. Type-chain wiring across 5+ IPC layers needs an explicit checklist — omitting one layer creates subtle state bugs that pass unit tests (v1.11)
20. SUMMARY frontmatter one_liner auto-extraction is unreliable across 10 milestones — needs tooling fix or manual curation at completion time (v1.1-v1.9)
21. Extending existing wire protocols (binary framing) is cheaper than designing new ones — MsgMeta added in v2.0 with one type constant and handler, no breaking changes (v2.0)
22. Shared package extraction (internal/attach/) should happen at the second consumer, not later — CLI+TUI attach unified cleanly because extraction was planned in the phase (v2.0)
23. Non-optimistic UI toggles (wait for server success before updating state) prevent impossible UI states — worth the async complexity (v2.1)
24. Shared persistence structs across phases need integration testing when multiple features write to the same storage layer (v2.1)
