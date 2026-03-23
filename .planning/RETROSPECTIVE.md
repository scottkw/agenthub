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

## Cross-Milestone Trends

### Process Evolution

| Milestone | Commits | Phases | Key Change |
|-----------|---------|--------|------------|
| v1.0 | 107 | 6 | Initial process — inside-out architecture, gap closure plans |
| v1.1 | ~50 | 7 | Source-inspection tests, JSX conditional pattern, layout-first ordering |
| v1.2 | 64 | 5 | Safety dependency chain, deletion-focused milestone, function injection testing |

### Cumulative Quality

| Milestone | Go Tests | JS Tests | LOC | Tech Debt Items |
|-----------|----------|----------|-----|-----------------|
| v1.0 | 53+ (race-clean) | 0 | ~8,100 | 14 (0 blockers) |
| v1.1 | 55+ (race-clean) | 73 | ~9,956 | 9 (0 blockers) |
| v1.2 | 55+ (race-clean) | 73+ | ~8,846 | 3 (0 blockers, info-only) |

### Top Lessons (Verified Across Milestones)

1. Interface-first design pays off — define contracts before implementations
2. Gap closure plans catch integration issues that initial planning misses
3. Phase ordering matters — foundation phases (layout, contracts) before feature phases prevents cascading issues
4. Source-inspection tests are a viable pattern when runtime testing is impractical (Canvas/WebGL, Wails bindings)
5. Deletion milestones benefit from strict dependency ordering — confirm replacements work before removing originals (v1.2)
