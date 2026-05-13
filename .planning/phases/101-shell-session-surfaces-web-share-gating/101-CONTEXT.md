---
phase: 101-shell-session-surfaces-web-share-gating
type: context
status: ready
mode: auto-generated
---

# Phase 101: Shell Session Surfaces & Web-Share Gating - Context

**Gathered:** 2026-05-12
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped — research-complete pattern; ROADMAP + REQUIREMENTS pre-answer the gray areas)

<domain>
## Phase Boundary

User can pick a shell (bash, zsh, pwsh, or "system default") as a first-class "agent" everywhere AgentHub already surfaces agents:
- **GUI new-session modal** — shell appears alongside the 6 AI CLIs in the agent picker
- **CLI** — `agenthub new shell <path>` and/or `--shell=bash|zsh|pwsh` flag
- **TUI** — new-session modal flow

Shell sessions visually distinguished via a distinct agent badge color in the GUI tab bar and TUI session list (consistent with the existing 6-CLI palette).

Web serving for shell sessions is **opt-in only** — when the web server is running, newly-created shell sessions are NOT auto-enabled for web serving (overrides the agent-session default). On the first toggle-ON, a one-time confirmation banner explains the security implications (arbitrary command execution).

**Scope:** SHELL-01, SHELL-02, SHELL-03, SHELL-06, SHELL-07, SHELL-08 — 6 requirements across 3 surfaces.

**Out of scope:** Backend daemon shell spawning (Phase 100), status heuristic exclusion (Phase 100), custom shell binary path picker (rejected — system-discovery sufficient).
</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting `workflow.skip_discuss` semantics. The ROADMAP phase goal, success criteria, REQUIREMENTS.md SHELL-01..SHELL-08, codebase conventions, and Phase 100's already-built `GET /shells` API are the authoritative inputs.

Specifically deferred to the planner agent:
- **Modal UX:** How shell selection appears in the agent picker (separator? grouped? inline?). Mirror existing 6-CLI pattern unless that's awkward.
- **Badge color:** Pick a hex color for the shell badge that visually distinguishes shells from the 6 existing CLI badges. Consider terminal-green or slate-gray as natural shell-y choices that don't collide with brand colors.
- **CLI argv shape:** `agenthub new shell <path>` (positional) vs `--shell=bash` flag — research existing `new` command shape and follow that pattern.
- **TUI flow:** Mirror the existing new-session modal; add shell entries to the agent list.
- **Web-share confirmation persistence:** Backend setting (preferred — persists across reinstalls) vs. frontend localStorage. Decision: use the existing settings store (backend) so the warning is per-user/per-machine, not per-browser-context.
- **"System default" semantics:** Resolves to `$SHELL` on POSIX or `pwsh.exe` → `powershell.exe` fallback on Windows (matches Plan 100-01's synthetic-default behavior).

</decisions>

<code_context>
## Existing Code Insights

**Already built (Phase 100):**
- `internal/pty/shells.go::DiscoverShells()` — returns `[]DetectedShell{Name, Path, Source}` cross-platform
- `internal/daemon/engine.go::CreateSession` — resolves abstract names (`shell`, `bash`, `zsh`, `pwsh`, `powershell`) → absolute path + non-login interactive argv
- `internal/daemon/engine.go::isShellSession` + `resolveShellSpawn` — shell vs CLI dispatch
- `GET /shells` HTTP endpoint + `DaemonClient.ListShells()` — list available shells for UI consumption

**Surfaces to modify (Phase 101):**
- **Frontend (Wails React):** `frontend/src/` — new-session modal (likely `frontend/src/components/NewSession*`), tab badge rendering, web-share toggle confirmation
- **CLI:** `cmd/agenthub/` — `new` subcommand (extend or add `shell` sub-subcommand)
- **TUI:** `internal/tui/` — new-session modal (mirror existing CLI agent list)
- **Settings/persistence:** wherever the existing per-session web-share defaults live

Pattern reference: Phase 100's PATTERNS.md captures the AI-CLI shape that shells now mirror. For UI patterns, look at how the existing 6 CLIs (claude, opencode, etc.) are rendered in the agent picker and tab badges.

</code_context>

<specifics>
## Specific Ideas

1. **Agent picker:** Add a "Shell" entry (or one entry per discovered shell) to the existing agent picker. Use `GET /shells` to populate dynamically rather than hardcoding.
2. **Badge color:** Terminal-green (`#10b981` Tailwind emerald-500) or slate-gray — must visually distinguish from existing 6 CLI badges. Decision deferred to UI-SPEC research.
3. **Web-share gating logic:**
   - When `CreateSession` returns for `is_shell == true`, frontend reads existing per-session `web_share_enabled` default and overrides to `false` if shell
   - When user toggles web-share ON for first shell session, show one-time modal/banner
   - Persist "user has been warned" flag in backend settings (e.g., `settings.shell_web_share_warned: true`)
4. **CLI shape:** `agenthub new shell [<path>] [--shell=bash|zsh|pwsh]` — positional path optional (defaults to CWD), `--shell` selects which discovered shell to spawn (defaults to "system default").
5. **TUI:** Extend new-session modal agent list with shell entries (one per discovered shell, plus "system default").

</specifics>

<deferred>
## Deferred Ideas

- Custom shell binary path picker (explicitly out-of-scope per REQUIREMENTS Anti-Goals)
- Shell session command-history persistence (not in this milestone)
- Per-shell theme preferences (not in this milestone)

</deferred>
