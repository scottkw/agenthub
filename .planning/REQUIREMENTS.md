# Requirements: AgentHub

**Defined:** 2026-03-25
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.5 Requirements

Requirements for v1.5 Bug Fixes & CLI Args. Each maps to roadmap phases.

### Terminal Fix

- [ ] **TERM-01**: Terminal fills correctly on initial load for Claude CLI sessions (no resize needed)
- [ ] **TERM-02**: Terminal fills correctly on initial load for Gemini CLI sessions (no resize needed)
- [ ] **TERM-03**: PTY sessions spawn at appropriate initial dimensions instead of hardcoded 80x24
- [ ] **TERM-04**: Double-rAF deferral on `fit()` ensures layout is committed before terminal sizing

### Daemon Performance

- [ ] **PERF-01**: Session status appears immediately after session creation (no artificial delay)
- [ ] **PERF-02**: `pollSessionStatus` first poll runs without 2-second sleep
- [ ] **PERF-03**: Service-mode daemon resolves agent CLIs in user PATH (nvm, volta, Homebrew paths)

### CLI Arguments

- [ ] **ARGS-01**: User can pass extra arguments to an agent via `agenthub new <agent> -- --flag value`
- [ ] **ARGS-02**: User can enter extra arguments in the GUI new-session modal text field
- [ ] **ARGS-03**: Args propagate through daemon layers (types → engine → API → client → PTY)
- [ ] **ARGS-04**: Per-agent argument memory: last-used args pre-filled in GUI modal
- [ ] **ARGS-05**: User can clear or edit pre-filled args before session creation

## Future Requirements

### Deferred

- **POLL-01**: WebSocket/SSE push for session status instead of polling
- **ARGS-06**: Quoted multi-word argument handling in GUI text field (shlex-level parsing)
- **TERM-05**: WebGL context loss fallback renderer

## Out of Scope

| Feature | Reason |
|---------|--------|
| Agent-specific status heuristics | Deferred from v1.0; separate milestone |
| Tab color coding per CLI type | Deferred from v1.2; separate milestone |
| Configurable poll intervals | Over-engineering for v1.5; fix the sleep first |
| Agent installation/management | App checks for CLIs but doesn't install them |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| TERM-01 | — | Pending |
| TERM-02 | — | Pending |
| TERM-03 | — | Pending |
| TERM-04 | — | Pending |
| PERF-01 | — | Pending |
| PERF-02 | — | Pending |
| PERF-03 | — | Pending |
| ARGS-01 | — | Pending |
| ARGS-02 | — | Pending |
| ARGS-03 | — | Pending |
| ARGS-04 | — | Pending |
| ARGS-05 | — | Pending |

**Coverage:**
- v1.5 requirements: 12 total
- Mapped to phases: 0
- Unmapped: 12 ⚠️

---
*Requirements defined: 2026-03-25*
*Last updated: 2026-03-25 after initial definition*
