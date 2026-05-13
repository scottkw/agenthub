---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
type: context
status: ready
mode: autonomous-direct (research complete from 2026-05-13 v3.3 audit + first-user-test feedback)
gathered: 2026-05-13
---

# Phase 107: Shell UX Collapse + Binary Path Picker + Clean-Exit Handling

**Gathered:** 2026-05-13
**Mode:** Autonomous-direct (gray areas pre-answered from audit + dev-test feedback; discuss step intentionally skipped per `feedback_skip_discuss_when_research_complete.md`)

<domain>

## Phase Boundary

Three closely-related shell-UX deltas surfaced by first-user testing of Phase 100/101 on 2026-05-13:

1. **SHELL-10** — collapse new-session modal shell rows to a single "Shell" entry (reverses SHELL-01's multi-row design).
2. **SHELL-11** — expose a "Shell binary" path field in Settings → Paths (flips an Out-of-Scope item; auto-discovery is no longer "sufficient").
3. **SHELL-12** — clean-exit handling: normalize PTY exit-code `-1 → 0` in the ExitToast emission path AND auto-close the tab on a clean exit (matches the user's expectation that a `exit`-typed shell should disappear like an AI CLI tool does on natural completion).

All three are corrections to v3.3 v1; the underlying daemon + frontend plumbing from Phases 100/101 is already in place and reusable.

</domain>

<decisions>

## Implementation Decisions — Pre-Answered

### SHELL-10 (single Shell entry)

- **The single "Shell" row uses the configured `paths.shell` value, NOT a runtime picker.** No per-session shell choice; if the user wants `bash` instead of `zsh` they change it in Settings.
- **Label:** "Shell" (no `— bash` suffix; the configured path is shown as the detail line under the row, e.g., `/bin/zsh`).
- **Removed:** the `sortedShells.map(...)` loop in `NewSessionModal.tsx:154-172` collapses to a single `<button>` element.
- **Removed:** the `shellsLoading` skeleton (replaced by a single static row that's always present; if `paths.shell` is unset, the row uses `$SHELL` fallback and shows the resolved path in the detail line).
- **Keep:** the existing daemon `pty.DiscoverShells()` API and `GET /shells` route — they're still useful for the Settings → Paths field's "pick from discovered" affordance (SHELL-11) and for the CLI `agenthub new shell --shell=bash` flag which still accepts overrides.
- **Drop:** the `shell:NAME` prefix scheme in `selectedAgent`. With one row, the agent id is just `shell` (no per-binary subtype).

### SHELL-11 (Settings → Paths shell binary field)

- **Field name:** "Shell binary" (sits next to existing "Claude Code", "OpenCode", etc. fields).
- **Default value (if unset):** resolved at daemon-side on first read — `$SHELL` env var → `/etc/passwd` lookup → `/bin/zsh` (macOS), `/bin/bash` (Linux), `pwsh.exe` (Windows).
- **Persistence:** add `shellPath` field to daemon settings struct (`internal/daemon/engine.go` settings struct), mirroring the existing `paths.X` pattern (Claude Code, OpenCode, etc.). New HTTP routes `GET/PATCH /settings/shell-path` mirroring the existing `/settings/shell-web-share-warned` pattern.
- **Validation:** path must (a) exist, (b) be executable (`os.Stat` + `mode&0111 != 0`), (c) be in the discovered-shells list OR have been explicitly typed by the user. Validation runs daemon-side on PATCH; on failure return 400 with a stderr-friendly message.
- **UI affordance:** text input field with a dropdown of `DiscoverShells()` results as suggestions (similar to the existing Settings → Paths fields). Manual typing allowed for custom paths.
- **Anti-Goal still respected:** No "login shell" toggle, no per-tab shell override, no shell history sync. These remain Out-of-Scope.

### SHELL-12 (clean-exit handling)

- **Bug location:** `internal/daemon/engine.go:333-335` already normalizes `-1 → 0` on the *natural-exit* path, but the *ExitToast emission path* at `engine.go:377-398` reads `s.ExitCode()` directly without that normalization. Apply the same `if ec == -1 { ec = 0 }` guard at line 383-384.
- **Auto-close on exit-code 0:** the frontend tab-management code receives the session-exit event. On exit-code 0, automatically dispatch the existing tab-close handler (skip the ExitToast render entirely). On non-zero, keep the existing ExitToast behavior.
- **Status field:** when the daemon emits the exit event with exit-code 0, also flip the session's `state` to `stopped` (currently it stays "running" — see screenshot: status "running" while body says "exited with error"). This is consistent with SHELL-09 contract (`running` / `stopped` only).
- **Tab-close UX:** if the tab is the active tab when it closes, switch focus to the tab to its left (or Welcome if it was the only one). Match the existing "agent finished" auto-close behavior for AI CLIs.

</decisions>

<code_context>

## Existing Code Insights

### SHELL-10 modify
- `frontend/src/components/NewSessionModal.tsx` — collapse `sortedShells.map(...)` loop into single static row; remove `SHELL_PREFIX` parsing.
- `frontend/src/App.tsx:1217` — keep `shells={detectedShells}` prop for the Settings → Paths suggestions (now used by SettingsTab, not the modal).
- `cmd_cli.go:cmdNewShell` — CLI flag `--shell=NAME` still useful for power users; keep but ensure it bypasses the Settings `paths.shell` default.

### SHELL-11 new
- `internal/daemon/engine.go` — add `shellPath string` to settings struct (~L38 area); load/save round-trip (~L165, L190); `GetShellPath()` / `SetShellPath()` methods (~L583, L593).
- `internal/daemon/api.go` — `GET /settings/shell-path` + `PATCH /settings/shell-path` mirroring `/settings/shell-web-share-warned` pattern (~L73-74 + handlers ~L545+).
- `internal/daemon/client.go` — `GetShellPath()` / `SetShellPath()` methods (~L142).
- `app.go` — Wails wrappers `GetShellPath()` / `SetShellPath()` (mirror lines 421-440).
- `frontend/src/components/SettingsTab.tsx` — locate the Paths section (Phase 104 added `id="settings-paths"`) and add the new field row.
- `frontend/src/components/PathsSection.tsx` (if it exists) OR inline in SettingsTab — use the existing field-row pattern.

### SHELL-12 modify
- `internal/daemon/engine.go:377-398` — apply `-1 → 0` guard in the ExitToast emission path.
- `internal/daemon/engine.go:294-310` — when shell exits cleanly (exit-code 0), set session state to `stopped` (currently the SHELL-09 status.Watch bypass leaves state untouched).
- `frontend/src/App.tsx` — find the session-exit event handler; branch on `exitCode === 0` → close tab silently; else → existing ExitToast behavior.
- `frontend/src/components/ExitToast.tsx:46-50` — no change to render logic; just stop being rendered for clean exits.

### Testing pattern
- Daemon: `internal/daemon/engine_test.go` already has shell-spawn tests (Phase 100). Add `TestSessionExit_ZeroCode_AutoClose` (asserts exit-code 0 → state=stopped + no ExitToast event), `TestSessionExit_NonZeroCode_ShowsToast`.
- Frontend: `frontend/src/components/__tests__/App.shellExit.test.tsx` (new) — assert tab closes on exit-code 0, toast renders on exit-code ≠ 0.
- Settings UI: extend `frontend/src/components/__tests__/SettingsTab.hyperlinked-index.test.tsx` OR new `SettingsTab.shellPath.test.tsx` — assert field renders, dropdown shows discovered shells, validation flags invalid paths.

</code_context>

<specifics>

## Specific Ideas

1. **Plan 107-01 (backend SHELL-11):** add `shellPath` settings field + GET/PATCH routes + DaemonClient methods + Wails wrappers + validation. Mirror the `shellWebShareWarned` plumbing pattern verbatim.
2. **Plan 107-02 (frontend SHELL-10 + SHELL-11):** collapse modal to single "Shell" row reading `paths.shell` via new Wails RPC; add Settings → Paths "Shell binary" field with discovered-shells dropdown.
3. **Plan 107-03 (SHELL-12 daemon-side):** normalize `-1 → 0` in ExitToast emission path; flip state to `stopped` on clean shell exit.
4. **Plan 107-04 (SHELL-12 frontend):** branch on exit-code 0 in App.tsx session-exit handler → close tab silently; preserve ExitToast for non-zero.

Parallelization: Plans 107-01 and 107-03 are both daemon-only and can run in wave 0 in parallel. Plans 107-02 and 107-04 are both frontend and depend on the daemon outputs of waves 0; they can run in wave 1 in parallel.

</specifics>

<deferred>

## Deferred Ideas

- **Login shell toggle (`-l` / `--login`):** explicitly Out-of-Scope per REQUIREMENTS.md.
- **Per-tab shell override:** Out-of-Scope; users change the Settings field, not per-session.
- **Shell history sync:** Out-of-Scope; each shell uses its native history.
- **Custom shell argv override:** Out-of-Scope; the daemon's `["-i"]` argv is sufficient.
- **CLI `agenthub new shell --shell=NAME` deprecation:** the flag still works for power users who want to override the Settings default per-invocation; deprecation is a v3.4+ consideration.

</deferred>

<test_plan>

## Test Plan Summary

- **SHELL-10:** Vitest test for NewSessionModal asserts exactly one "Shell" row, no per-binary rows; existing AI-CLI rows unchanged.
- **SHELL-11:** Daemon RoundTripJSON test + API integration test (GET default + PATCH valid path + PATCH invalid path returns 400) + Vitest for Settings field render + Wails RPC binding test.
- **SHELL-12:** Daemon test asserts exit-code 0 → state=stopped + no toast event; exit-code -1 normalized to 0; non-zero shows toast. Frontend test asserts tab auto-close on 0; ExitToast renders on non-zero.

All tests run in the existing harness (`go test ./internal/...` + `pnpm test`). No new test infrastructure.

</test_plan>
