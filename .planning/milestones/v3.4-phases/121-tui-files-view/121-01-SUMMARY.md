---
phase: 121-tui-files-view
plan: 01
subsystem: tui
tags: [tui, files, scaffolding, tea-cmd, key-dispatch]
requires: [phase-118/FS-03, phase-118/FS-05, phase-118/FS-06]
provides:
  - tabFiles tabID
  - filesModel sub-model + previewKind enum
  - loadDirCmd / readFileCmd / headFileCmd tea.Cmd factories
  - filesListMsg / filesReadMsg / filesHeadMsg / filesErrMsg message types
  - truncateLeft helper (path status-line)
  - KeyMap.FilesOpen / FilterStart / FilterEsc / FilesFocusToggle
  - Priority 5.5 key dispatch slot
  - Open-from-Sessions wiring ('f' on local → tabFiles, on remote → toast)
affects:
  - internal/tui/model.go (tabID iota extended, Model.files field added)
  - internal/tui/keys.go (KeyMap extended)
  - internal/tui/update.go (handleKey priority cascade, handleContentKey switch)
  - internal/tui/view.go (renderContentPane switch, truncateLeft helper)
tech-stack:
  added:
    - github.com/charmbracelet/glamour@v0.8.0 (promoted from transitive to direct)
  patterns:
    - tea.Cmd factory with context.WithTimeout (mirrors fetchSessions in cmds.go)
    - Sub-model embedded in Model + per-press reset (Pitfall TUI-PITFALL-6)
    - Echo-field message types (sessionID + relPath) for stale-msg detection
key-files:
  created:
    - internal/tui/files.go
    - internal/tui/files_cmds.go
    - internal/tui/files_test.go
  modified:
    - go.mod
    - go.sum
    - internal/tui/model.go
    - internal/tui/keys.go
    - internal/tui/update.go
    - internal/tui/view.go
decisions:
  - "tabFiles iota placed at end of existing tabID block (after tabSettings) — preserves ordering of existing sidebar/tab IDs."
  - "Priority 5.5 inserted between help (5) and tab-cycling (6) — modal priorities (1-4) still beat Files dispatch."
  - "filesModel reset on every 'f' keypress (no same-session optimisation) per Pitfall TUI-PITFALL-6."
  - "Glamour anchored via 'var _ = glamour.WithAutoStyle' in files.go — keeps go mod tidy happy while leaving renderer wiring for Plan 02."
  - "truncateLeft snaps to a path-segment boundary when one exists inside the kept window — width may be < maxWidth as a result (test adjusted accordingly)."
metrics:
  completed: 2026-05-20T23:49:20Z
  duration: ~10 min wall-clock
  tasks_completed: 3
  files_changed: 9
requirements:
  - TUI-01
  - TUI-02
  - TUI-06
  - TUI-07
  - TUI-10
---

# Phase 121 Plan 01: tabFiles + filesModel + tea.Cmd factories + key dispatch + open-from-Sessions wiring Summary

Laid the structural foundation for the TUI Files view: tabFiles tab ID, embedded filesModel sub-model, three context-bounded tea.Cmd factories wrapping `DaemonClient.{ListFiles, ReadFile, HeadFile}`, a left-truncating path helper, Priority 5.5 key dispatch insertion, and the open-from-Sessions wiring (`f` on local → opens tabFiles + loadDirCmd; `f` on remote → toast "File browser not available for remote sessions in v3.4").

## What landed

### Task 1 — Scaffold (commits `a12e850` RED + `30cb4da` GREEN)

- **Glamour promoted to direct dependency** in `go.mod` at `v0.8.0` (was transitive). Anchored by `var _ = glamour.WithAutoStyle` in `internal/tui/files.go` so `go mod tidy` retains it before Plan 02 starts using it for real.
- **`tabFiles` iota** added to `internal/tui/model.go` after `tabSettings`. `tabName` extended to return "Files".
- **`Model.files filesModel`** field added under a `// File browser state (Phase 121)` marker.
- **`internal/tui/files.go`** (new file):
  - `previewKind` int + 6 constants (`previewEmpty`, `previewText`, `previewMarkdown`, `previewBinary`, `previewOverCap`, `previewErr`).
  - `filesModel` struct with the full Plan-01/02 field surface (sessionID, cwd, entries, truncated, selected, loading, err, filterActive, filterInput, preview, previewKind, previewMime, previewLoading, previewErr, previewFocused).
  - `newFilesModel(sid, listW, listH, previewW, previewH) filesModel` constructor — initialises textinput.Model (prompt `"/ "`, width `listW-4`, char limit 128) and viewport.Model with the requested dimensions. `loading: true` by default; Plan 02 flips it on `filesListMsg` arrival.
  - `(m Model) renderFilesTab(cw, ch int) string` STUB — wraps `"Loading…"` in the bordered frame. Plan 02 replaces with the two-pane render.
  - `(m Model) handleFilesKey(msg tea.KeyPressMsg)` STUB — handles `Q` / `ctrl+c` (quit), `?` (help), `[` / `]` (tab cycle), `esc` (no-op). All other keys swallowed. Plan 02 replaces with full dispatch.
- **`KeyMap` extended** in `internal/tui/keys.go`: `FilesOpen` (`f`), `FilterStart` (`/`), `FilterEsc` (`esc`), `FilesFocusToggle` (`tab`). All four populated in `defaultKeyMap()`.
- **`truncateLeft(s string, maxWidth int) string`** added in `internal/tui/view.go` next to `truncate`. Defensive against zero/negative widths; snaps to path-segment boundary when one lies inside the kept window.
- **`renderContentPane`** extended with `case tabFiles:` → `m.renderFilesTab(cw, contentHeight)`.
- **Tests added** (3): `TestTruncateLeft` (7 sub-tests including multibyte + edge widths), `TestFiles_TabID_Distinct`, `TestFiles_KeyMap_BindsF`.

### Task 2 — tea.Cmd factories (commits `077ee4e` RED + `24a422a` GREEN)

- **`internal/tui/files_cmds.go`** (new file):
  - `errNilClient` sentinel — returned by all three factories when `client == nil` (testModel() path).
  - `filesListMsg` / `filesReadMsg` / `filesHeadMsg` / `filesErrMsg` — all carry `sessionID` + `relPath` echo fields so Plan 02's Update handler can detect stale results from a different session (T-121-04).
  - `loadDirCmd(client, sid, relPath) tea.Cmd` — 5s `context.WithTimeout`, calls `client.ListFiles`, returns `filesListMsg`.
  - `readFileCmd(client, sid, relPath) tea.Cmd` — 10s timeout (5 MiB body transfer), calls `client.ReadFile`, returns `filesReadMsg`.
  - `headFileCmd(client, sid, relPath) tea.Cmd` — 5s timeout, calls `client.HeadFile` and drops the modtime return (status line uses relPath, not mtime).
- **Tests added** (4): `TestLoadDirCmd_NilClient_ReturnsErrSentinel`, `TestReadFileCmd_NilClient_ReturnsErrSentinel`, `TestHeadFileCmd_NilClient_ReturnsErrSentinel`, `TestLoadDirCmd_DispatchesAsync`.

### Task 3 — Key-dispatch priority + Sessions wiring (commits `ca15a02` RED + `d3d6f94` GREEN)

- **`handleKey` Priority 5.5** in `internal/tui/update.go` — sits between Priority 5 (help overlay) and Priority 6 (tab cycling):
  ```go
  if m.activeTabID() == tabFiles {
      return m.handleFilesKey(msg)
  }
  ```
- **`handleContentKey` FilesOpen case** — placed after the existing `QR` case (consistent with the QR/Kill/Rename pattern of "needs an active list entry"):
  - Empty `unifiedList` → no-op.
  - Remote entry → toast "File browser not available for remote sessions in v3.4" (`toastInfo`, 2s).
  - Local entry → reset `m.files = newFilesModel(sid, listW=cw*40/100, paneH-2, previewW=cw-listW-1, paneH-2)`, open tab, return `loadDirCmd(client, sid, ".")`.
  - Pane dimensions clamped to ≥ 1 to avoid lipgloss panics on tiny terminals.
- **Tests added** (6): DispatchPriority, DispatchPriority_BelowKillConfirm, OpenFromSessions_LocalEntry, OpenFromSessions_RemoteEntry_ShowsToast, OpenFromSessions_EmptyList_NoOp, OpenFromSessions_ResetsModel.

## Verification

```
go build ./...               # exit 0
go vet ./internal/tui/       # no diagnostics
go test ./internal/tui/... -count=1 -race
ok  github.com/scottkw/agenthub/internal/tui  1.245s
```

Whole-plan grep gates:

| Check | Expected | Actual |
|-------|----------|--------|
| `filesModel` mentions in `files.go` | ≥ 3 | 5 |
| `context.WithTimeout` in `files_cmds.go` | == 3 | 3 |
| Synchronous FS in `files.go` / `files_cmds.go` (`os.ReadDir|Open|Stat|OpenFile`) | == 0 | 0 |
| `glamour` in `go.mod` | ≥ 1 | 1 (direct) |
| `STATE.md` / `ROADMAP.md` modified | 0 | 0 (worktree mode) |

13 new tests added; all pass alongside the 30+ existing TUI tests.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] TestTruncateLeft width-equals-maxWidth assertion was overly strict**

- **Found during:** Task 1 GREEN verification.
- **Issue:** The plan's behavior contract said `truncateLeft("a/b/c/d/utils/helper.ts", 18)` should return "a string of width 18 starting with `…/` ending with `helper.ts`." But the implementation snaps to path-segment boundaries (also required by the plan's action), which means width = 17 in this case (`…/utils/helper.ts`). Strict equality on rune-width vs. preserving the leaf segment are mutually exclusive — leaf preservation wins because that's the user-visible value.
- **Fix:** Relaxed the test to assert `startsWith == "…/" && endsWith == "helper.ts"` without an exact-width predicate. The width-counted assertion is retained for the other sub-tests where snapping is not in play (multibyte case).
- **Files modified:** `internal/tui/files_test.go`.
- **Commit:** `30cb4da` (combined with Task 1 GREEN — RED was already committed with the strict assertion at `a12e850`, so the relaxation rides with the implementation rather than a separate "test fix" commit).

**2. [Rule 1 — Bug] TestFiles_OpenFromSessions_ResetsModel had a setup gap**

- **Found during:** Task 3 GREEN verification.
- **Issue:** Test pressed `'f'` twice. After the first press the active tab is `tabFiles`, so the second `'f'` goes through Priority 5.5 → `handleFilesKey` (stub), NOT through `handleContentKey`. The stub swallows `'f'` without resetting, so `cwd="subdir"` was preserved and the assertion fired. The reset behaviour (Pitfall 6) is specifically scoped to opening from the Sessions tab — re-pressing `'f'` while already in `tabFiles` should NOT re-trigger a reload (that would defeat user navigation).
- **Fix:** Test now switches back to the Sessions tab before the second `'f'` press, which is the realistic flow.
- **Files modified:** `internal/tui/files_test.go`.
- **Commit:** `d3d6f94`.

No architectural changes required. No package-install failures. No checkpoint hits.

## Threat Model Coverage

| Threat ID | Disposition | Plan 01 mitigation status |
|-----------|-------------|----------------------------|
| T-121-01 (DoS via slow daemon I/O) | mitigate | Done — all 3 Cmds use `context.WithTimeout`. |
| T-121-02 (client-side path-sandbox tampering) | mitigate | Done — TUI passes `(sid, relPath)` straight through; no client-side validation. |
| T-121-03 (cap-token leakage) | accept | N/A — TUI talks daemon socket only. |
| T-121-04 (stale msg confusion) | mitigate | Carrier added — all msg types include `sessionID` + `relPath`. Plan 02 will consume. |
| T-121-05 (preview pane DoS) | mitigate | Carrier added — single shared `viewport.Model` in `filesModel`. Plan 02 implements `SetContent("")` on nav. |
| T-121-06 (ANSI-escape spoofing in filenames) | mitigate | Not yet surfaced — Plan 01 does not render entries. |
| T-121-07 (nil DaemonClient panic) | mitigate | Done — all 3 factories check `client == nil` and return `errNilClient`. |
| T-121-SC (glamour promotion supply-chain) | mitigate | Done — `go get @v0.8.0`; no new third-party code (was already in module cache). |

No new threat surface introduced beyond what was anticipated in the plan's threat register.

## Known Stubs

These are intentional Plan-01 stubs that Plan 02 will replace:

| File | Symbol | Reason | Resolved in |
|------|--------|--------|-------------|
| `internal/tui/files.go` | `renderFilesTab` returns `"Loading…"` only | Plan 02 implements two-pane (list \| preview) render | 121-02 |
| `internal/tui/files.go` | `handleFilesKey` swallows most keys | Plan 02 implements filter/nav/preview dispatch | 121-02 |
| `internal/tui/files_cmds.go` | `filesErrMsg` type declared but not emitted | Plan 02 will emit for preview-render failures | 121-02 |
| `internal/tui/files.go` | `var _ = glamour.WithAutoStyle` anchor | Plan 02 will replace with real renderer wiring | 121-02 |

These stubs are required to land Plan 01's structural seams without entangling Plan 02's behavior — they are not bugs.

## Self-Check: PASSED

Verified file existence:

- `internal/tui/files.go` — FOUND
- `internal/tui/files_cmds.go` — FOUND
- `internal/tui/files_test.go` — FOUND
- `.planning/phases/121-tui-files-view/121-01-SUMMARY.md` — FOUND (this file)

Verified commits in branch history (`git log --oneline -8`):

- `d3d6f94` — feat(121-01): wire Priority 5.5 key dispatch + 'f' opens tabFiles (GREEN) — FOUND
- `ca15a02` — test(121-01): add failing tests for priority dispatch + open-from-Sessions (RED) — FOUND
- `24a422a` — feat(121-01): add tea.Cmd factories for ListFiles/ReadFile/HeadFile (GREEN) — FOUND
- `077ee4e` — test(121-01): add failing tests for files_cmds.go factories (RED) — FOUND
- `30cb4da` — feat(121-01): scaffold tabFiles + filesModel + truncateLeft (GREEN) — FOUND
- `a12e850` — test(121-01): add failing tests for tabFiles + truncateLeft + KeyMap (RED) — FOUND

Verified success criteria:

- [x] Tasks committed atomically with `feat(121-01:)` / `test(121-01:)` prefixes
- [x] SUMMARY.md at `.planning/phases/121-tui-files-view/121-01-SUMMARY.md`
- [x] `glamour@v0.8.0` promoted to direct dep in `go.mod`
- [x] `tabFiles` iota added to `model.go`
- [x] `filesModel` struct in `internal/tui/files.go` (NEW file)
- [x] tea.Cmd factories in `internal/tui/files_cmds.go` (NEW file): `loadDirCmd`, `readFileCmd`, `headFileCmd` — all use `context.WithTimeout`
- [x] `truncateLeft` helper for status line path truncation
- [x] Key-dispatch priority slot inserted between help overlay (priority 5) and tab cycling (priority 6)
- [x] "Open from Sessions list" wiring: `f` key on local session → opens Files tab; on remote → toast
- [x] `go test ./internal/tui/...` passes (race-clean)
- [x] `go build ./internal/...` clean
- [x] No modifications to `STATE.md` / `ROADMAP.md`
