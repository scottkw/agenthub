---
plan: 97-05
phase: 97-serialize-addon-save-session-ux
status: complete
tasks: 2/2
completed: 2026-05-07
---

# Plan 97-05 Summary

## What was built

**Task 1 — `(*App).SaveTerminalSession` Wails RPC + bindings:**
- `app.go`: added `saveFileDialogFunc` field on `*App` (function-injection pattern matching `serviceControlFunc`/`statusFunc` per PROJECT.md Key Decisions). Initialized to `runtime.SaveFileDialog` in `NewApp`.
- `app.go`: implemented `(*App).SaveTerminalSession(defaultDir, defaultName, content string) error` — opens native Save File dialog, treats empty path return as silent success per OpenFileDialog precedent at lines 815-829, writes with `os.WriteFile(path, []byte(content), 0o644)`, wraps errors with `SaveTerminalSession: dialog`/`SaveTerminalSession: write` triage prefixes.
- `frontend/src/wailsjs/go/main/App.d.ts`: replaced Plan 97-03 type stub comment with canonical Phase 92 hand-edit-pin form.
- `frontend/src/wailsjs/go/main/App.js`: added `SaveTerminalSession` Call() runtime stub mirroring SetImageConfig (Phase 96 pattern).
- `app_save_terminal_test.go`: replaced Plan 97-01 RED scaffold with table-driven GREEN test covering 4 sub-cases — cancel (silent success), write (success), IO error, dialog error.
- `internal/release/no_autosave_test.go`: unskipped `TestSER03_OnlySaveTerminalSessionInAppGo` — flips GREEN automatically now that `SaveTerminalSession` exists.

**Task 2 — SER-02 secrets-warning caption in Plugins UI:**
- `frontend/src/components/PluginsSection.tsx`: added 4th argument to `renderRow('serialize', ...)` with verbatim REQUIREMENTS SER-02 string: "Saved files include any secrets, tokens, or sensitive data printed in the session."
- Existing label and description unchanged.
- Renders as italic caption via existing `settings-panel__description--italic` class (Phase 93/96 pattern).
- `frontend/src/components/__tests__/PluginsSection.test.tsx`: flipped RED scaffold to GREEN with verbatim string assertion + positional check.

## Commits

- `6121f06` feat(97-05): implement (*App).SaveTerminalSession + Wails bindings + flip tests GREEN
- `65495d8` feat(97-05): add verbatim SER-02 secrets-warning caption to PluginsSection Serialize row

## Key files

### Created
- (none — Wave 0 scaffolds existed; Plan 97-05 flipped them GREEN)

### Modified
- `app.go` (+44 / SaveTerminalSession + saveFileDialogFunc field)
- `app_save_terminal_test.go` (+131 / table-driven test, RED → GREEN)
- `frontend/src/wailsjs/go/main/App.d.ts` (canonical Phase 92 form)
- `frontend/src/wailsjs/go/main/App.js` (+3 / Call() stub)
- `frontend/src/components/PluginsSection.tsx` (+3 / SER-02 caption)
- `frontend/src/components/__tests__/PluginsSection.test.tsx` (+15 / flip GREEN)
- `internal/release/no_autosave_test.go` (-2 / unskip dialog-call-site test)

## Verification

- `go test ./... -run TestSaveTerminalSession` GREEN
- `go test ./internal/release -run TestSER03` GREEN (all 3 sub-tests)
- `go build ./...` clean
- `pnpm tsc --noEmit` clean
- PluginsSection caption flips GREEN

## Operational note

Agent execution stalled in stream watchdog before SUMMARY.md could be written.
Both feature commits were complete and self-checks had passed.
Orchestrator recovered the dangling commits via `git fsck --unreachable`,
cherry-picked them onto main, and authored this SUMMARY.md from the agent's
final transcript and commit messages.

## Self-Check: PASSED
