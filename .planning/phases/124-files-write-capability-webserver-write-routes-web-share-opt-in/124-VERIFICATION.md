---
phase: 124-files-write-capability-webserver-write-routes-web-share-opt-in
verified: 2026-06-14T12:58:00Z
status: passed
score: 10/10
overrides_applied: 0
human_verification_completed: 2026-06-14
human_verification_result: >
  Validated 2026-06-14 (user requested validate-now). SC#5 migration observed LIVE — the real
  ~/Library/Application Support/agenthub/settings.json was migrated 3→4 by a live daemon run
  (filesWrite absent = false default). SC#5 persistence confirmed LIVE — wrote filesWrite:true,
  restarted daemon, value round-tripped (schemaVersion 4, filesWrite:true); original settings restored.
  SC#1/SC#2 webserver gating is covered by the 26-subtest TestRequireFilesWrite (real middleware stack).
  GUI + TUI home-dir warning colorblind treatment source-verified: ⚠ glyph + literal "Warning:" at
  HomeDirWriteWarning.tsx:45,47 and internal/tui/files.go:332; both surfaces gate on the same
  SessionInfo.HomeDir && FilesWrite server signal (cross-surface parity). Live on-screen GUI/TUI render
  deferred to milestone-end batch (Wails GUI not run headless); logic + tokens fully proven.
human_verification:
  - test: "Enable files.write for a $HOME-cwd session in the live GUI; confirm the amber HomeDirWriteWarning banner (with ⚠ glyph and 'Warning:' text) appears beneath the owner toggle."
    expected: "Banner renders with heading 'Warning: writes can affect your home directory' and body copy from CAP-06; dismiss button works; banner re-appears on re-enable."
    why_human: "Requires a running Wails desktop app with a $HOME-cwd session; cannot be driven by grep or unit tests."
  - test: "Enable files.write for a $HOME-cwd session in the live TUI; confirm the warning line is visible in the Files view status area."
    expected: "The line '⚠ Warning: cwd is $HOME — writes can affect dotfiles, SSH keys, and shell config. Protected files are blocked.' appears above the status line with amber amber styling."
    why_human: "Requires a running TUI with a live session; cannot be driven by unit tests."
  - test: "Toggle web-share 'Allow file editing' ON in a session with owner writes enabled, restart the daemon, then re-issue capabilities; confirm the default persists as false (opt-in, not persisted across daemon restart as enabled)."
    expected: "After daemon restart the 'Enable file writes' toggle is back to default OFF (per-session state is in-memory; the persisted FilesWrite default is false)."
    why_human: "Requires a daemon restart cycle with live GUI interaction."
---

# Phase 124: files.write Capability + Webserver Write Routes + Web-Share Opt-In — Verification Report

**Phase Goal:** files.write capability bit + requireFilesWrite middleware (CSRF Origin check) gating all five webserver write routes + opt-in for every token (per-session owner toggle; viewer further opt-in) + schemaVersion 4 migration.
**Verified:** 2026-06-14T12:58:00Z
**Status:** passed (live UAT 2026-06-14: migration + persistence confirmed; warnings source-verified)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Cap without files.write → 403 on all five webserver write routes; with files.write → 2xx | VERIFIED | `TestRequireFilesWrite` PASS: 5 routes x (403-missing, 2xx-present, Origin-mismatch-403, absent-Origin-pass, matching-Origin-pass) = 25 sub-tests + 1 priority sub-test all GREEN |
| 2 | Mismatched Origin → 403; absent Origin → vacuous pass | VERIFIED | Same test Origin sub-cases all PASS; `originAllowedForWrite` source confirmed: `if origin == "" { return true }` then strict BaseURL match |
| 3 | No write-path strings.Contains for files.write check | VERIFIED | `TestHasPerm_NoStringsContains_Write` PASS (exit 0); capability_mw.go uses `capability.HasPerm(claims.Perms, capability.PermFilesWrite)` |
| 4 | Web-share opt-in default OFF; ON with confirm → files.write in issued cap; home-dir write warning in GUI and TUI; both surfaces gate on same server signal | VERIFIED | SessionSharePanel tests (8/8 PASS): default-OFF, disabled-until-owner-ON, confirm-gates-cap; HomeDirWriteWarning tests (8/8 PASS): ⚠ glyph + "Warning:" literal; TUI tests (4/4 PASS): HomeDirWarning positive + 3 negative cases; parity condition identical (`SessionInfo.HomeDir && SessionInfo.FilesWrite` on both surfaces) |
| 5 | schemaVersion 3→4 migration with FilesWrite default false; per-session model not global | VERIFIED | `TestSettingsMigration_FilesWriteDefaultsFalse` PASS; no `filesWrite *bool` global field; `sessionWrites map[string]bool` per-session under `e.mu`; `filesWriteDefault bool` zero-value false |
| 6 | CAP-01: PermFilesWrite constant in capability package | VERIFIED | `internal/capability/capability.go:37`: `const PermFilesWrite = "files.write"` |
| 7 | CAP-02: requireFilesWrite is a third separate wrapper (not added to requireCapability) | VERIFIED | Separate function in `capability_mw.go:147`; `TestRequireCapability_UnchangedByPhase118` (pre-existing) remains green |
| 8 | CAP-07: Five write routes mounted via filesDispatch closure (no direct webserver→files coupling) | VERIFIED | `grep -c 'requireFilesWrite(filesDispatch' server.go` = 5; same closure reused; no second `filesDispatch :=` definition |
| 9 | CAP-10: Remote proxy forwards r.Body + Content-Type for write verbs; nil-body bug fixed; five routes registered | VERIFIED | `TestRemoteFilesWrite_ForwardsBody`, `TestRemoteFilesWrite_CallerCapStripped`, `TestRemoteFilesWrite_GetPassesNilBody` all PASS; `grep -c 'NewRequestWithContext(r.Context(), r.Method, upstreamURL, nil)' remote_files.go` = 0 |
| 10 | Phase 123 write engine (internal/files/) was NOT modified by this phase | VERIFIED | `git show <all-124-commits> --stat` has zero hits for `internal/files/`; most recent internal/files/ commits are Phase 123 (`fix(123):`, `feat(123-03):`) |

**Score:** 10/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/capability/capability.go` | PermFilesWrite constant | VERIFIED | Line 37: `const PermFilesWrite = "files.write"` with doc comment noting whole-token HasPerm semantics |
| `internal/webserver/capability_mw.go` | requireFilesWrite + originAllowedForWrite | VERIFIED | Lines 147, 187: both functions present with correct ordering (requireCapability → HasPerm → Origin) |
| `internal/webserver/server.go` | Five write route mounts behind requireFilesWrite | VERIFIED | Lines 512–516: all five routes wrap `requireFilesWrite(filesDispatch(...))` |
| `internal/webserver/capability_test.go` | TestRequireFilesWrite + TestHasPerm_NoStringsContains_Write | VERIFIED | Lines 604, 774: both tests exist and pass |
| `internal/daemon/plugin_settings.go` | CurrentSchemaVersion = 4 | VERIFIED | Line 9: `const CurrentSchemaVersion = 4` |
| `internal/daemon/engine.go` | filesWriteEnabledFor + per-session map + homeDir signal | VERIFIED | Lines 44–45 (fields), 545 (filesWriteEnabledFor), 553 (Unlocked variant), 563 (SetSessionFilesWrite), 573 (sessionCwdIsHome using EvalSymlinks) |
| `internal/daemon/api.go` | Cap mint appends files.write when per-session write ON; HomeDir in response | VERIFIED | Line 1075: `if a.engine.filesWriteEnabledFor(sessionID) { ownerPerms += "," + capability.PermFilesWrite }`; HomeDir populated at response construction |
| `internal/daemon/types.go` | SessionInfo.HomeDir/FilesWrite + IssueCapabilitiesResponse.HomeDir | VERIFIED | Lines 32–33 (SessionInfo), Line 144 (IssueCapabilitiesResponse.HomeDir) |
| `internal/daemon/engine_migration_test.go` | TestSettingsMigration_FilesWriteDefaultsFalse | VERIFIED | Test present and PASS |
| `internal/daemon/remote_files.go` | proxyRemoteFiles forwards r.Body + Content-Type for write verbs | VERIFIED | Line 213: `body = r.Body` for write verbs; nil-body call at line 169 removed (grep count = 0) |
| `internal/daemon/remote_files_test.go` | Remote write-proxy body-forwarding test | VERIFIED | TestRemoteFilesWrite_ForwardsBody + _CallerCapStripped + _GetPassesNilBody all PASS |
| `app.go` | SetSessionFilesWrite Wails binding | VERIFIED | Line 776: `func (a *App) SetSessionFilesWrite(sessionID string, enabled bool) error` |
| `internal/daemon/client.go` | DaemonClient.SetSessionFilesWrite | VERIFIED | Line 315: `func (c *DaemonClient) SetSessionFilesWrite(...)` |
| `frontend/src/components/HomeDirWriteWarning.tsx` | Colorblind-safe banner: ⚠ + "Warning:" | VERIFIED | Lines 45, 47: `⚠` glyph span + `<span>Warning: writes can affect your home directory</span>` |
| `frontend/src/components/SessionSharePanel.tsx` | "Allow file editing" opt-in row | VERIFIED | Lines 13–16 (ownerWriteEnabled prop), 174+ (write-optin row with aria-disabled) |
| `frontend/src/components/DaemonManagerPanel.tsx` | Owner "Enable file writes" toggle + banner mount | VERIFIED | Lines 303, 316, 324: toggle row; HomeDirWriteWarning mount conditional |
| `frontend/src/style.css` | --home-write-warning CSS modifier | VERIFIED | Line 1863: `.webgl-recovery-banner--home-write-warning` using pre-existing `#f59e0b` amber |
| `frontend/src/wailsjs/go/main/App.d.ts` | SetSessionFilesWrite export + homeDir: boolean | VERIFIED | Lines 177, 185: both present |
| `internal/tui/files.go` | Home-dir write warning line in renderFilesTab | VERIFIED | Line 332: `homeDirWriteWarning` const with verbatim copy; warning rendered on `HomeDir && FilesWrite` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `server.go` | `requireFilesWrite` | route mount wrapping filesDispatch | VERIFIED | 5 occurrences of `requireFilesWrite(filesDispatch(...))`|
| `capability_mw.go` | `capability.HasPerm` | whole-token permission check | VERIFIED | Line 156: `capability.HasPerm(claims.Perms, capability.PermFilesWrite)` |
| `api.go issueCapabilitiesForSession` | `engine.filesWriteEnabledFor` | per-session write check before appending files.write | VERIFIED | Lines 1073–1076: conditional append using per-session check |
| `engine.go homeDir signal` | `filepath.EvalSymlinks` | EvalSymlinks($HOME) before comparing to session cwd | VERIFIED | Line 573+: `cwdEqualsHome` calls `filepath.EvalSymlinks(home)` |
| `DaemonManagerPanel.tsx` | `SetSessionFilesWrite` Wails binding | owner toggle onChange | VERIFIED | `handleToggleFilesWrite` calls `SetSessionFilesWrite` and re-issues capabilities |
| `HomeDirWriteWarning.tsx` | `IssueCapabilitiesResponse.homeDir` | banner shown when homeDir && filesWrite enabled | VERIFIED | `share?.homeDir` prop used in DaemonManagerPanel mount condition |
| `tui/files.go renderFilesTab` | `SessionInfo.HomeDir && SessionInfo.FilesWrite` | lookup by m.files.sessionID in m.sessions | VERIFIED | Lines 372–379: loop over m.sessions, `HomeDir && FilesWrite` gate |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `DaemonManagerPanel.tsx` | `sessionWrites[id]` | `SetSessionFilesWrite` Wails → daemon engine `sessionWrites` map | Yes — engine per-session map written on toggle | FLOWING |
| `DaemonManagerPanel.tsx` | `share?.homeDir` | `IssueCapabilities` response `HomeDir` field ← `sessionCwdIsHome(id)` ← EvalSymlinks | Yes — computed live from cwd | FLOWING |
| `SessionSharePanel.tsx` | `ownerWriteEnabled` prop | passed from DaemonManagerPanel `sessionWrites[s.id]` | Yes — live engine state | FLOWING |
| `tui/files.go` | warning condition | `m.sessions[i].HomeDir && m.sessions[i].FilesWrite` ← `ListSessions` ← engine unlocked variants | Yes — server-computed signals | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Cap without files.write → 403; with → 2xx on all 5 routes | `go test ./internal/webserver/ -run TestRequireFilesWrite -count=1` | PASS (26 sub-tests) | PASS |
| Static-grep gate: no strings.Contains for files.write | `go test ./internal/webserver/ -run TestHasPerm_NoStringsContains_Write -count=1` | PASS | PASS |
| schemaVersion 3→4 migration FilesWrite defaults false | `go test ./internal/daemon/ -run TestSettingsMigration_FilesWriteDefaultsFalse -count=1` | PASS | PASS |
| HomeDirWriteWarning renders ⚠ + "Warning:" | `cd frontend && pnpm test -- --run HomeDirWriteWarning` | 8/8 PASS | PASS |
| SessionSharePanel viewer opt-in default OFF, gated, confirm gates cap | `cd frontend && pnpm test -- --run SessionSharePanel` | 8/8 PASS | PASS |
| TUI home-dir warning shown when HomeDir&&FilesWrite; absent otherwise | `go test ./internal/tui/ -run 'HomeDirWarning' -count=1` | 4/4 PASS | PASS |
| Remote write proxy forwards r.Body; nil-body bug fixed | `go test ./internal/daemon/ -run TestRemoteFilesWrite -count=1` | 3/3 PASS | PASS |
| Race-free: webserver + capability | `go test -race ./internal/webserver/ ./internal/capability/ -count=1` | PASS | PASS |
| Race-free: daemon | `go test -race ./internal/daemon/ -count=1` | PASS | PASS |
| Race-free: tui | `go test -race ./internal/tui/ -count=1` | PASS | PASS |
| go build ./internal/... | `go build ./internal/...` | clean (no output) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| CAP-01 | 124-01 | PermFilesWrite constant, gated via HasPerm (not strings.Contains) | SATISFIED | `capability.go:37` + `TestHasPerm_NoStringsContains_Write` PASS |
| CAP-02 | 124-01 | requireFilesWrite separate from requireCapability and requireFilesRead | SATISFIED | Third distinct wrapper in `capability_mw.go:147`; does not modify requireCapability |
| CAP-03 | 124-01 | CSRF Origin check on write methods; absent Origin passes vacuously | SATISFIED | `originAllowedForWrite` absent→true, present mismatch→false; `TestRequireFilesWrite` Origin sub-cases PASS |
| CAP-04 | 124-02, 124-04 | Per-session "Enable file writes" toggle (default OFF); owner cap only gets files.write when toggle ON | SATISFIED | `sessionWrites map[string]bool`; `SetSessionFilesWrite` binding chain; GUI toggle in DaemonManagerPanel |
| CAP-05 | 124-04 | Web-share viewer explicit opt-in with confirmation (default OFF) | SATISFIED | `SessionSharePanel` "Allow file editing" row: aria-disabled until owner ON; inline confirm; SessionSharePanel tests 8/8 PASS |
| CAP-06 | 124-04, 124-05 | Home-dir write warning on both GUI (⚠ + "Warning:") and TUI surfaces | SATISFIED | `HomeDirWriteWarning.tsx:45,47` + `files.go:332`; colorblind-safe glyph+text verified at source level; parity condition identical on both surfaces |
| CAP-07 | 124-01 | Webserver write routes via SetFilesHandlerProvider/filesDispatch pattern | SATISFIED | 5 mounts reuse same `filesDispatch` closure in `server.go` |
| CAP-08 | 124-02 | schemaVersion 4 migration; FilesWrite defaults false (opt-in for all) | SATISFIED | `plugin_settings.go:9` = 4; `FilesWrite bool` zero-value false; `TestSettingsMigration_FilesWriteDefaultsFalse` PASS |
| CAP-09 | 124-01 | Capability-denied integration tests: viewer without files.write → 403; with → 2xx | SATISFIED | `TestRequireFilesWrite` 26 sub-tests PASS; static-grep gate PASS |
| CAP-10 | 124-03 | Remote proxy write routes; r.Body forwarded for PUT/POST/PATCH | SATISFIED | 5 remote routes in `api.go:171-175`; nil-body call removed; `TestRemoteFilesWrite_ForwardsBody` PASS |

### Anti-Patterns Found

No blocker anti-patterns detected. Scan results:
- No `TBD`, `FIXME`, or `XXX` markers in any phase-touched file
- No `TODO`/`HACK`/`PLACEHOLDER` markers in any phase-touched file
- No `return null`/`return []`/placeholder text in rendering paths
- No `strings.Contains` paired with `"files.write"` (static-grep gate enforces this)
- No global `filesWrite *bool` field (per-session map confirmed)

### Human Verification Required

Three items require live-daemon UAT. These are documented as manual-only in `124-VALIDATION.md` (all automated coverage proves the logic; only the rendered UI and restart-persistence require human eyes/interaction).

#### 1. GUI Home-Dir Write Warning — Live App

**Test:** Run the Wails desktop app. Open a session whose cwd is `$HOME`. Enable the "Enable file writes" toggle.
**Expected:** The amber HomeDirWriteWarning banner appears beneath the toggle, showing the ⚠ glyph and the heading "Warning: writes can affect your home directory". Clicking dismiss closes it. Re-enabling shows it again.
**Why human:** Requires a running Wails desktop app with a $HOME-cwd session; cannot be driven by grep or unit tests.

#### 2. TUI Home-Dir Write Warning — Live TUI

**Test:** Run the TUI. Navigate to the Files view on a $HOME-cwd session with files.write active.
**Expected:** The line "⚠ Warning: cwd is $HOME — writes can affect dotfiles, SSH keys, and shell config. Protected files are blocked." appears above the status line in amber.
**Why human:** Requires a running TUI with a live session; no headless render test can verify terminal color output.

#### 3. Web-Share Opt-In Restart-Persistence

**Test:** Enable files.write for a session, toggle the web-share "Allow file editing" opt-in ON, then restart the daemon. Re-open the panel for that session.
**Expected:** Per-session write state (in-memory) resets to the persisted default (false) after restart, demonstrating the opt-in model is session-scoped and defaults false on restart.
**Why human:** Requires a full daemon restart cycle with live GUI interaction.

### Gaps Summary

No gaps. All 10 success criteria truths verified. All 10 CAP requirement IDs satisfied. No stubs, no placeholder text, no unresolved debt markers. All automated tests pass under `-race`. The three human verification items are documented manual-only behaviors from `124-VALIDATION.md` — they prove live rendering and UX, not logic (logic is proven by the automated test suite).

---

_Verified: 2026-06-14T12:58:00Z_
_Verifier: Claude (gsd-verifier)_
