# Requirements: AgentHub

**Defined:** 2026-03-26
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.6 Requirements

Requirements for v1.6 Terminal Fill Fix v2. Each maps to roadmap phases.

### Terminal Fill

- [ ] **FILL-01**: Terminal fills the full viewport width on initial tab activation for Claude CLI sessions
- [ ] **FILL-02**: Terminal fills the full viewport width on initial tab activation for Gemini CLI sessions
- [ ] **FILL-03**: Terminal fills the full viewport width on initial tab activation for OpenCode sessions
- [ ] **FILL-04**: Terminal fills the full viewport width on initial tab activation for Codex sessions (no regression)
- [ ] **FILL-05**: Switching tabs to a previously hidden terminal fills correctly without resize
- [ ] **FILL-06**: Fix works in both `wails dev` and production `wails build` modes

## Future Requirements

### Deferred

- **POLL-01**: WebSocket/SSE push for session status instead of polling
- **ARGS-06**: Quoted multi-word argument handling in GUI text field (shlex-level parsing)
- **TERM-05**: WebGL context loss fallback renderer

## Out of Scope

| Feature | Reason |
|---------|--------|
| WebGL renderer fallback | Separate concern, not related to initial fill |
| Terminal height fill | Only width is broken; height fills correctly |
| Agent-specific status heuristics | Deferred from v1.0; separate milestone |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| FILL-01 | — | Not started |
| FILL-02 | — | Not started |
| FILL-03 | — | Not started |
| FILL-04 | — | Not started |
| FILL-05 | — | Not started |
| FILL-06 | — | Not started |

**Coverage:**
- v1.6 requirements: 6 total
- Mapped to phases: 0 (awaiting roadmap)
- Unmapped: 6

---
*Requirements defined: 2026-03-26*
*Last updated: 2026-03-26*
