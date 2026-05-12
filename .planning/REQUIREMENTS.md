# Requirements: AgentHub v3.3 — Shell Sessions & Polish

**Defined:** 2026-05-12
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

**Milestone Goal:** Land raw shell sessions (bash/zsh/pwsh) as a first-class agent type — unblocking the ~9 deferred v3.2 UAT scenarios that require raw PTY — then close v3.2 polish/tech-debt and re-run the deferred UAT batches end-to-end.

**Closes GitHub Issues:** #44 (shell agent), #45 (Settings hyperlinked index). Carry-forward: v3.2 polish P-1..P-6 from `v3.2-RELEASE-BLOCKERS.md`; 9 deferred v3.2 UAT scenarios; Phase 91 distribution-pipeline followups from v3.1.

## v3.3 Requirements

Requirements for the v3.3 release. Each maps to roadmap phases (continuing numbering from Phase 99 → Phase 100).

### SHELL — Raw shell session type (Issue #44)

- [ ] **SHELL-01**: User can select a shell (bash, zsh, pwsh, or "system default") in the new-session modal agent picker
- [ ] **SHELL-02**: User can launch a shell session from CLI with `agenthub new shell <path>` (or equivalent `--shell=bash|zsh|pwsh` flag)
- [ ] **SHELL-03**: User can launch a shell session from TUI via the existing new-session modal flow
- [ ] **SHELL-04**: Daemon discovers available shells per platform (macOS: `/bin/bash`, `/bin/zsh`; Linux: `$SHELL`, `/etc/shells` entries; Windows: `pwsh.exe`, `powershell.exe`)
- [ ] **SHELL-05**: Shell sessions spawn as interactive (not login) with the user's selected working directory honored
- [ ] **SHELL-06**: Shell sessions display a distinct agent badge color in GUI tab and TUI list (consistent with the existing 6-CLI palette)
- [ ] **SHELL-07**: Shell sessions do NOT auto-enable web serving when the web server is running (opt-in only — overrides the agent-session default)
- [ ] **SHELL-08**: User sees a one-time confirmation banner when first enabling web serving for a shell session, explaining that shells expose arbitrary command execution
- [ ] **SHELL-09**: Shell sessions are excluded from CLI-status heuristics (only `running` / `stopped` indicators — no fake `waiting` / `error` state)

### SETUI — Settings hyperlinked index (Issue #45)

- [ ] **SETUI-01**: User sees a sticky jump-link bar at the top of the Settings tab with anchor links to each section header (Plugins, Appearance, Web Server, Behavior, Paths)
- [ ] **SETUI-02**: Clicking a jump-link smoothly scrolls the Settings tab to that section
- [ ] **SETUI-03**: User can type into an autocomplete search box at the top of Settings to filter and jump to specific settings by label

### POLISH — v3.2 polish closure (P-1..P-6 from `v3.2-RELEASE-BLOCKERS.md`)

- [ ] **POLISH-01**: User can Cmd/Ctrl-click `mailto:` URLs in terminal output and the system mail client opens (P-1 — closes spec gap in Web-Links scheme allowlist)
- [ ] **POLISH-02**: User sees the `LinkConfirmPopover` when a URL contains non-ASCII (Cyrillic / IDN homograph) hostname characters, with the Punycode form displayed alongside the display form (P-2)
- [ ] **POLISH-03**: User can press Esc or click the close button to dismiss the find bar AFTER clicking the case-sensitive toggle (P-3 — focus/event-propagation fix)
- [ ] **POLISH-04**: Find bar slides out with the same 200ms exit animation it uses on entry, on both Esc and close-button dismiss (P-4 — apply `.find-bar--exiting` with delayed unmount in `FindBar.tsx` + `web/assets/terminal.js`)
- [ ] **POLISH-05**: iTerm2 inline-image protocol (IIP / OSC 1337) renders images correctly with the Image plugin enabled — OR — documentation explicitly states sixel-only support (P-5 — investigate and decide)
- [ ] **POLISH-06**: All 20 currently-failing `Sidebar.test.tsx` tests pass under Vitest 4 + jsdom 29 (P-6 — `localStorage` global setup via vitest setupFile)

### UAT — Deferred v3.2 UAT re-run (now unblocked by SHELL-01..09)

- [ ] **UAT-01**: Phase 93 WebGL context-loss → DOM fallback verified in desktop Chrome via `WEBGL_lose_context.loseContext()` (banner shows + 8s auto-dismiss)
- [ ] **UAT-02**: Phase 93 iPad Safari software-rasterizer banner verified on physical iPad + Tailscale-served session
- [ ] **UAT-03**: Phase 94 10,000-line scrollback regex search performance verified (DevTools main-thread frame timing within budget)
- [ ] **UAT-04**: Phase 95 LNK-01..05 full chain verified on iPad Safari + Tailscale (Cmd-click / IDN popover / typosquat / OSC 8 / `mailto:`)
- [ ] **UAT-05**: Phase 96 `chafa --format=iterm2` (or sixel) desktop+web visual-fidelity comparison verified using a shell session
- [ ] **UAT-06**: Phase 96 two-client mid-stream image join verified (byte-fidelity replay across simultaneous WSS subscribers)
- [ ] **UAT-07**: Phase 99 iPad 5-scenario runbook verified on physical iPad + Tailscale (attach+chafa, scrollback, zero-CDN, zero-CSP, all-8-ON Progress)

### DIST — Phase 91 distribution-pipeline followups (carried from v3.1)

- [ ] **DIST-01**: `release.yml` publishes via a non-`GITHUB_TOKEN` credential (PAT or GitHub App token) so `release.published` automatically triggers `distribute.yml` (91-A)
- [ ] **DIST-02**: `distribute.yml`'s submit-winget step reads `RELEASE_TAG` env var (works on both `release:published` and `workflow_dispatch`; fixes empty `$version` and double-dash installer URL) (91-B)
- [ ] **DIST-03**: WinGet first submission uses `wingetcreate new` for the initial microsoft/winget-pkgs submission rather than `update` (91-C)

## Future Requirements

Deferred to v3.4+. Tracked but not in current roadmap.

### Networking & infra (Issue #9, #42)

- **NET-01**: Let's Encrypt certs for non-Tailscale public IP serving (Issue #42 — needs DNS + port-forward detection)
- **NET-02**: Embed Headscale + Tailscale client with cloud-VPS docker-compose stack (Issue #9)

### File management (Issue #24)

- **FILE-01**: File browser tab with remote upload/download/preview/edit (Issue #24 — scoped to session working directory)

### Mobile & multi-agent (Issue #30, #10)

- **MOB-01**: iOS/Android wrapper apps (PWA + Capacitor or React Native) (Issue #30)
- **ORCH-01**: Inter-session communication via Claude Peers MCP server (Issue #10)

### Admin control (Issue #47)

- **ADMIN-01**: Central admin policy enforcement for restricting app behaviors (Issue #47 — MDM-style)

## Out of Scope

Explicitly excluded for v3.3. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Custom shell binary path picker | System-discovery (SHELL-04) is sufficient; custom paths add UX complexity and rare-use surface |
| Shell login mode (`-l` / `--login`) toggle | Interactive non-login covers 95% of use; login-shell semantics deferred until users request |
| Per-shell theme override | Global theme works for all session types; per-tab override remains out-of-scope (carried from v1.12) |
| Shell session status heuristics (waiting/error) | Shells have no AI-agent state model; SHELL-09 explicitly excludes fake state |
| Multi-shell side-by-side in one tab | Split panes are out-of-scope for the project; one shell per tab |
| Issue #48 (Add update check) | Already shipped in v1.9 Phase 51; closed as resolved 2026-05-12 |
| In-app auto-install of updates | Current download-link UX is sufficient; auto-install across signed/notarized binaries is a separate research effort |
| Shell history sync across sessions | Each shell uses its own native history (`~/.bash_history`, `~/.zsh_history`); cross-session sync is shell-specific and out-of-scope |
| Settings autocomplete fuzzy-match across plugin sub-options | SETUI-03 covers section-label search only; deep-search into every disclosed sub-config defers to a future Settings overhaul |

## Traceability

Which phases cover which requirements. Populated by roadmap creation 2026-05-12.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SHELL-01 | Phase 101 | Pending |
| SHELL-02 | Phase 101 | Pending |
| SHELL-03 | Phase 101 | Pending |
| SHELL-04 | Phase 100 | Pending |
| SHELL-05 | Phase 100 | Pending |
| SHELL-06 | Phase 101 | Pending |
| SHELL-07 | Phase 101 | Pending |
| SHELL-08 | Phase 101 | Pending |
| SHELL-09 | Phase 100 | Pending |
| SETUI-01 | Phase 104 | Pending |
| SETUI-02 | Phase 104 | Pending |
| SETUI-03 | Phase 104 | Pending |
| POLISH-01 | Phase 102 | Pending |
| POLISH-02 | Phase 102 | Pending |
| POLISH-03 | Phase 103 | Pending |
| POLISH-04 | Phase 103 | Pending |
| POLISH-05 | Phase 103 | Pending |
| POLISH-06 | Phase 103 | Pending |
| UAT-01 | Phase 105 | Pending |
| UAT-02 | Phase 105 | Pending |
| UAT-03 | Phase 105 | Pending |
| UAT-04 | Phase 105 | Pending |
| UAT-05 | Phase 105 | Pending |
| UAT-06 | Phase 105 | Pending |
| UAT-07 | Phase 105 | Pending |
| DIST-01 | Phase 106 | Pending |
| DIST-02 | Phase 106 | Pending |
| DIST-03 | Phase 106 | Pending |

**Coverage:**
- v3.3 requirements: 28 total
- Mapped to phases: 28 ✓
- Unmapped: 0

**Phase distribution:**
- Phase 100 (Shell Backend & Discovery): SHELL-04, SHELL-05, SHELL-09 (3 reqs)
- Phase 101 (Shell Surfaces & Web-Share Gating): SHELL-01, SHELL-02, SHELL-03, SHELL-06, SHELL-07, SHELL-08 (6 reqs)
- Phase 102 (Web-Links Polish — mailto + IDN): POLISH-01, POLISH-02 (2 reqs)
- Phase 103 (Find Bar + Test-Env + IIP Polish): POLISH-03, POLISH-04, POLISH-05, POLISH-06 (4 reqs)
- Phase 104 (Settings Hyperlinked Index): SETUI-01, SETUI-02, SETUI-03 (3 reqs)
- Phase 105 (Deferred v3.2 UAT Re-Run): UAT-01..07 (7 reqs)
- Phase 106 (Distribution Pipeline Followups): DIST-01, DIST-02, DIST-03 (3 reqs)

---
*Requirements defined: 2026-05-12*
*Last updated: 2026-05-12 — traceability populated by roadmap creation (28/28 mapped across Phases 100-106)*
