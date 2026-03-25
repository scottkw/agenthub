# Requirements: AgentHub

**Defined:** 2026-03-25
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.4 Requirements

Requirements for unified binary milestone. Each maps to roadmap phases.

### Command Routing

- [x] **ROUTE-01**: User can run `agenthub` with no args to launch GUI
- [x] **ROUTE-02**: User can run `agenthub <command>` to execute any CLI command
- [x] **ROUTE-03**: User can run `agenthub daemon` to start daemon mode

### CLI Integration

- [x] **CLI-01**: All 13 CLI commands (new, list, kill, rename, attach, serve, unserve, web, health, qr, settings, daemon) work from unified binary
- [x] **CLI-02**: `--json` flag works on applicable commands from unified binary
- [x] **CLI-03**: Interactive attach (raw PTY, detach key, resize) works from unified binary
- [x] **CLI-04**: `agenthub --help` shows both GUI and CLI usage

### Cleanup

- [x] **CLEAN-01**: `cmd/agenthub-cli/` directory fully removed
- [x] **CLEAN-02**: No references to `agenthub-cli` remain in docs, CI, or build scripts

### Build System

- [ ] **BUILD-01**: `build.sh` produces single binary that handles GUI + CLI + daemon
- [ ] **BUILD-02**: GitHub Actions CI builds and tests unified binary
- [ ] **BUILD-03**: All existing tests pass against unified binary (daemon tests, CLI tests, attach tests)

## Future Requirements

None — v1.4 is a focused refactoring milestone.

## Out of Scope

| Feature | Reason |
|---------|--------|
| New CLI commands | v1.4 is merge-only, no new functionality |
| Headless CLI-only build target | Fully removing cmd/agenthub-cli per user decision |
| Tab completion / shell integration | Future milestone |
| Man page generation | Future milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| ROUTE-01 | Phase 27 | Complete |
| ROUTE-02 | Phase 27 | Complete |
| ROUTE-03 | Phase 27 | Complete |
| CLI-01 | Phase 27 | Complete |
| CLI-02 | Phase 27 | Complete |
| CLI-03 | Phase 27 | Complete |
| CLI-04 | Phase 27 | Complete |
| CLEAN-01 | Phase 28 | Complete |
| CLEAN-02 | Phase 28 | Complete |
| BUILD-01 | Phase 29 | Pending |
| BUILD-02 | Phase 29 | Pending |
| BUILD-03 | Phase 29 | Pending |

**Coverage:**
- v1.4 requirements: 12 total
- Mapped to phases: 12
- Unmapped: 0

---
*Requirements defined: 2026-03-25*
*Last updated: 2026-03-25 — traceability filled after roadmap creation*
