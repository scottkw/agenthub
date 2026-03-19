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

## Cross-Milestone Trends

### Process Evolution

| Milestone | Commits | Phases | Key Change |
|-----------|---------|--------|------------|
| v1.0 | 107 | 6 | Initial process — inside-out architecture, gap closure plans |

### Cumulative Quality

| Milestone | Go Tests | JS Tests | LOC | Tech Debt Items |
|-----------|----------|----------|-----|-----------------|
| v1.0 | 53+ (race-clean) | 0 | ~8,100 | 14 (0 blockers) |

### Top Lessons (Verified Across Milestones)

1. Interface-first design pays off — define contracts before implementations
2. Gap closure plans catch integration issues that initial planning misses
