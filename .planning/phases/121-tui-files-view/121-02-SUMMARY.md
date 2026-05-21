---
phase: 121-tui-files-view
plan: 02
subsystem: tui
tags: [tui, files, key-dispatch, preview, glamour, render]
requires: [phase-121/121-01]
provides:
  - applyFilesListMsg / applyFilesHeadMsg / applyFilesReadMsg helpers
  - full handleFilesKey dispatch (nav + filter + preview-focus)
  - parentDir / joinDir / filesModel.filteredEntries
  - renderFilesTab two-pane layout + renderFilesListPane / renderFilesPreviewPane / renderFilesStatusLine
  - per-tab buildHelpContent ("Files" group)
  - per-tab renderHintBar (Files-specific hints)
affects:
  - internal/tui/files.go (full Plan-02 behaviour replaces Plan-01 stubs)
  - internal/tui/update.go (msg-switch dispatches the 3 Plan-02 msgs)
  - internal/tui/help.go (switch on activeTabID for keybinding groups)
  - internal/tui/view.go (renderHintBar switches on activeTabID)
tech-stack:
  added: []
  patterns:
    - "Strict filter-mode key cascade (Pitfall TUI-PITFALL-2 — Backspace stays in textinput)"
    - "Echo-field sessionID staleness guard in every msg handler (T-121-04)"
    - "ansi.Strip applied to FileEntry.Name before any render or path construction (T-121-06)"
    - "Preview classification: HEAD-first decision tree (overcap > binary > markdown > text)"
    - "Preview reset on cwd change (T-121-09) — viewport reused, content cleared"
key-files:
  modified:
    - internal/tui/files.go
    - internal/tui/files_test.go
    - internal/tui/update.go
    - internal/tui/help.go
    - internal/tui/view.go
decisions:
  - "previewSizeCap constant (5 MiB) lives in files.go alongside the apply-helpers — keeps the policy adjacent to its enforcement site."
  - "Markdown detection uses suffix-OR-mime — daemon mime sniffing may miss .md files served with text/plain by older proxies, but the suffix is authoritative."
  - "Glamour render failure falls back to plain text (not previewErr) — bad markdown still has user-readable content; only ReadFile errors become previewErr."
  - "parentDir uses path.Dir (forward-slash) — daemon side normalises to '/' regardless of host OS, so filepath.Dir would break on Windows."
  - "PgUp/PgDn route to viewport when previewFocused — list jumps 10 entries when list-focused (no per-page calc; cheap and predictable)."
  - "renderFilesListPane reserves a bottom row for the filter input whenever filterActive OR a non-empty value persists — keeps the input visually anchored even when the user has stepped out of filter-input mode."
metrics:
  completed: 2026-05-20T00:00:00Z
  duration: ~15 min wall-clock
  tasks_completed: 3
  files_changed: 5
requirements:
  - TUI-01
  - TUI-03
  - TUI-04
  - TUI-05
  - TUI-06
  - TUI-08
  - TUI-09
---

# Phase 121 Plan 02: handleFilesKey (nav + filter + Tab focus) + preview classification with glamour + status line + help overlay + hint bar Summary

Replaced the Plan-01 stubs with the full Files view behaviour: navigation (Up/Down/PgUp/PgDn/Enter/Backspace), inline filter (`/` + Esc with strict Backspace cascade), preview-focus toggle (Tab), preview classification via HEAD-then-READ (text → raw, markdown → glamour, binary → "Use desktop or web to preview", over-cap → "Too large…"), session-cwd-relative status line with left-truncation, Files-specific help overlay group, and Files-specific hint bar.

## What landed

### Task 1 — apply-helpers + Update wiring (commits `7ecd32f` RED + `866c496` GREEN)

- **`applyFilesListMsg`** in `internal/tui/files.go`:
  - Staleness guard: `msg.sessionID != m.files.sessionID` → no-op (T-121-04).
  - Mutates `cwd`, `entries`, `truncated`, `selected = 0`, `loading = false` on success.
  - Resets preview pane (viewport `SetContent("")`, previewKind/previewErr/previewMime/previewLoading cleared) on every directory change so old preview content doesn't leak across navigation (T-121-09).
  - On error: substring-checks `"session not found"` and translates to a friendly `errors.New("Session no longer running")` — the files tab stays open with a clear stale-session indicator instead of auto-closing.
- **`applyFilesHeadMsg`** decision tree:
  - `err != nil` → `previewKind = previewErr`, content = `"Error: …"`.
  - `size > previewSizeCap` (5 MiB) → `previewKind = previewOverCap`, content = `"Too large to preview, use desktop or web to download"`.
  - `!strings.HasPrefix(mime, "text/")` → `previewKind = previewBinary`, content = `"Use desktop or web to preview"`.
  - Otherwise → set `previewLoading = true`, dispatch `readFileCmd`.
- **`applyFilesReadMsg`** classification:
  - Suffix-OR-mime markdown detection (`.md` / `.markdown` / `text/markdown`).
  - Markdown → `glamour.Render` with style `"dark"`/`"light"` from `m.hasDark`; render error falls back to plain text (still user-visible).
  - Non-markdown text → `previewKind = previewText`, raw bytes as content.
- **`Update` switch** extended with three new cases routing to the helpers; no other `Update` behaviour changed.

### Task 2 — full handleFilesKey dispatch (commits `11d5438` RED + `b508297` GREEN)

- **`handleFilesKey`** replaced the Plan-01 stub with a 132-line cascade:
  - **Filter-mode capture FIRST**: Esc clears + blurs, Enter commits + blurs, every other key forwards to `filterInput.Update(msg)` and resets `selected` to 0 so the cursor lands on the first match. Backspace here deletes a char inside the textinput — it can never navigate up (Pitfall TUI-PITFALL-2, asserted by `TestHandleFilesKey_FilterActive_BackspaceDoesNotNavigate`).
  - **Non-filter mode**:
    - `/` → activate filter.
    - Backspace / Left → root no-op (TUI-03) when `cwd == "" || cwd == "."`, else `loadDirCmd(parentDir(cwd))`.
    - Up / Down → if `previewFocused` route to viewport, else move cursor with bounds.
    - PgUp / PgDn → same focus-aware routing; non-preview jumps 10 entries.
    - Enter → `loadDirCmd` on dir, `headFileCmd` on file (sets `previewLoading = true`).
    - Tab → toggle `previewFocused`.
    - `?` → open help.
    - `[` / `]` → cycle tabs (Pitfall TUI-PITFALL-7 — must work even inside Files).
    - `Q` / `ctrl+c` → quit.
    - Any other key → swallowed (no fall-through).
- **`parentDir`** uses `path.Dir` (forward-slash daemon paths), strips trailing slash, returns `""` for root / `"."` / `"/"`.
- **`joinDir`** returns `name` alone when base is `""` or `"."`, else `path.Join`.
- **`filesModel.filteredEntries`** lowercases the trimmed filter value and matches case-insensitively against `ansi.Strip(entry.Name)` — defends T-121-06 (filename ANSI-escape spoofing).

### Task 3 — two-pane render + status line + help + hint bar (commits `130240c` RED + `bf79618` GREEN)

- **`renderFilesListPane(w, h, focused)`**:
  - Empty state: centred "No files" message.
  - Populated: visible window (clamps selected into view), `>` cursor prefix, `/` suffix for dirs, `BgSelected`/`FgSelected` highlight for the cursor row, `truncate` for overflow.
  - Bottom-anchored filter row (active = `"/ " + filterInput.View()` in FgAccent; static = `"/ " + value` in FgMuted) when either `filterActive` OR a persisted value exists.
  - Wrapped in `wrapInFrame` with `" Files "` title and BorderAccent when focused.
- **`renderFilesPreviewPane(w, h, focused)`**:
  - Sets viewport `SetWidth/SetHeight` to match inner pane on every render (resize-safe).
  - Frame title varies by `previewKind` (` Markdown ` / ` Binary ` / ` Too large ` / ` Error ` / ` Preview `).
  - Loading state replaces body with centred "Loading…".
- **`renderFilesStatusLine(w)`**:
  - Error mode: full-width `truncate("Error: "+err, w)` in StatusErrored.
  - Normal mode: `displayPath` (`/` for empty cwd, `./` prefix for non-rooted), left-truncated to `max(10, w-40)` via `truncateLeft` (Phase 121 status-line helper), followed by `• N entries[(truncated)] • i/N` in FgMuted.
- **`renderFilesTab(cw, ch)`**:
  - 40 % list / 60 % preview split via `lipgloss.JoinHorizontal` + 1-char `│`-column separator.
  - Status line composed below via `lipgloss.JoinVertical`.
- **`help.go::buildHelpContent`** now switches on `m.activeTabID()`:
  - tabFiles → "Files" group (Up/Down, PgUp/PgDn, Enter, Backspace/Left, Tab, /, Esc).
  - default → "Sessions" group with the new `f Open files view` binding alongside the existing keys.
- **`view.go::renderHintBar`** switches on `activeTabID()`:
  - tabFiles → `"Up/Down  Enter Open  Backspace Up  / Filter  Tab Focus  ? Help  Q Quit"`.
  - default → unchanged Sessions hint.

## Verification

```
go build ./...                       # exit 0
go vet ./internal/tui/               # no diagnostics
go test ./internal/tui/... -race     # ok 1.261s
```

Whole-plan grep gates:

| Check | Expected | Actual |
|-------|----------|--------|
| `filesListMsg`/`HeadMsg`/`ReadMsg` cases in `Update()` | 3 | 3 |
| `glamour.Render` mentions in `files.go` | 1 | 1 |
| `ansi.Strip` mentions in `files.go` | ≥ 2 | 4 |
| `case tabFiles:` in `help.go` | 1 | 1 |
| `lipgloss.JoinHorizontal` in `files.go` | ≥ 1 | 1 |
| Synchronous FS in `files.go`/`files_cmds.go` (`os.ReadDir|Open|Stat|OpenFile`) | 0 | 0 |
| `handleFilesKey` body line count | ≥ 60 | 132 |
| `m.files.filterActive` references inside `handleFilesKey` | ≥ 3 | 4 |
| STATE.md / ROADMAP.md modified | 0 | 0 (worktree mode) |

**31 new tests added** (9 Task 1 + 14 Task 2 + 8 Task 3) — all pass alongside the 30+ pre-existing TUI tests. Total `--- PASS` count in `go test -v`: **143** (no skipped / no failing).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Test gate mismatch] `glamour.Render` count gate hit a comment mention**

- **Found during:** Task 1 GREEN verification.
- **Issue:** Plan verify gate requires `grep -c 'glamour.Render' internal/tui/files.go` to be exactly 1, but my initial doc comment for `applyFilesReadMsg` said "goes through `glamour.Render`" — pushing the count to 2.
- **Fix:** Reworded the comment to "rendered via the glamour markdown renderer" so only the actual call site matches the grep. No code-behaviour change.
- **Files modified:** `internal/tui/files.go`.
- **Commit:** rolled into Task 1 GREEN (`866c496`).

### Plan compliance notes (NOT deviations)

- **TUI-PITFALL-7 (`[` / `]` cycling)** — verified by `TestHandleFilesKey_BracketKeysCycleTabs`. The dispatch uses `m.keys.PrevTab` / `m.keys.NextTab` (KeyMap-driven, not hardcoded), keeping a single source of truth for the cycle bindings.
- **TUI-PITFALL-6 (re-press resets `filesModel`)** — already covered by Plan 01's `TestFiles_OpenFromSessions_ResetsModel`; Plan 02 does NOT introduce same-session-reuse optimisations.
- **Filter selection clamp** — when Esc clears the filter, `m.files.selected` is clamped against the un-filtered entry count; when Enter commits the filter, it's clamped against the filtered count. Both branches handle the edge case where the cursor was on a now-invisible entry.

No architectural changes, no checkpoint hits, no package-install failures.

## Threat Model Coverage

| Threat ID | Disposition | Plan 02 status |
|-----------|-------------|----------------|
| T-121-04 (stale-msg confusion) | mitigate | Done — all 3 apply-helpers compare `msg.sessionID == m.files.sessionID`. |
| T-121-06 (ANSI-escape spoofing in filenames) | mitigate | Done — `ansi.Strip` called in `filteredEntries`, `renderFilesListPane`, and the Enter dispatch (3 sites). |
| T-121-08 (Backspace ambiguity — filter vs navigate-up) | mitigate | Done — filter-mode branch processes the press via `textinput.Update` and returns BEFORE any navigate-up code path. Verified by dedicated test. |
| T-121-09 (preview retains stale content across navigation) | mitigate | Done — `applyFilesListMsg` calls `preview.SetContent("")` + resets `previewKind/previewErr/previewMime/previewLoading` on every cwd change. |
| T-121-10 (malicious markdown payload) | accept | glamour v0.8.0 produces ANSI-only output; never executes embedded HTML/JS. Read access is gated by the daemon's sandbox. |

All previous (Plan 01) threats remain mitigated; no new surface introduced.

## Known Stubs

None. Plan 01's documented stubs are all resolved:

| File | Symbol | Plan-01 status | Plan-02 status |
|------|--------|----------------|----------------|
| `internal/tui/files.go` | `renderFilesTab` | stub returning "Loading…" | Full two-pane render |
| `internal/tui/files.go` | `handleFilesKey` | swallowed most keys | Full dispatch (132 lines) |
| `internal/tui/files_cmds.go` | `filesErrMsg` | declared but unused | Still declared but unused — reserved for Plan 03 preview-render error envelope. Not a stub in the user-visible sense; the type contract is stable. |
| `internal/tui/files.go` | `var _ = glamour.WithAutoStyle` anchor | go-mod-tidy anchor | Removed — `glamour.Render` is now a real call site. |

The `filesErrMsg` retention is intentional: Plan 03's preview-render test suite will likely emit it for synthetic failure paths, and removing it would force a no-behaviour Plan-01 reverse merge. Leaving it in is cheaper than churning the message-type surface.

## Self-Check: PASSED

Verified file existence:

- `internal/tui/files.go` — FOUND (updated; replaces Plan-01 stubs)
- `internal/tui/update.go` — FOUND (extended with 3 Plan-02 msg cases)
- `internal/tui/help.go` — FOUND (switch on activeTabID added)
- `internal/tui/view.go` — FOUND (renderHintBar switches on activeTabID)
- `internal/tui/files_test.go` — FOUND (31 new tests appended)
- `.planning/phases/121-tui-files-view/121-02-SUMMARY.md` — FOUND (this file)

Verified commits in branch history (`git log --oneline 51ce3618..HEAD`):

- `bf79618` — feat(121-02): two-pane renderFilesTab + status line + help + hint bar (Task 3 GREEN) — FOUND
- `130240c` — test(121-02): add failing tests for renderFilesTab + status + help + hints (Task 3 RED) — FOUND
- `b508297` — feat(121-02): full handleFilesKey dispatch + parentDir/joinDir/filteredEntries (Task 2 GREEN) — FOUND
- `11d5438` — test(121-02): add failing tests for handleFilesKey + parentDir + joinDir (Task 2 RED) — FOUND
- `866c496` — feat(121-02): wire filesListMsg/HeadMsg/ReadMsg into Update + apply-helpers (Task 1 GREEN) — FOUND
- `7ecd32f` — test(121-02): add failing tests for applyFiles*Msg helpers (Task 1 RED) — FOUND

Verified success criteria from execution prompt:

- [x] Tasks committed atomically with `feat(121-02:)` / `test(121-02:)` prefixes
- [x] SUMMARY.md at `.planning/phases/121-tui-files-view/121-02-SUMMARY.md`
- [x] handleFilesKey implements: Up/Down/PgUp/PgDn (cursor), Enter (enter dir), Backspace/Left (up — root-bounded), `/` (filter activate), Esc (filter clear+dismiss), Tab (focus toggle list↔preview), `?` (help overlay)
- [x] Backspace at root is a NO-OP (`TestHandleFilesKey_Backspace_AtRoot_NoOp` + `_Dot_NoOp`)
- [x] Preview classification: HEAD first → if size > 5 MB → "Too large to preview…"; if non-text → "Use desktop or web to preview"; if .md → `glamour.Render`; else raw text
- [x] Status line: session-cwd-relative path (left-truncated `…/utils/helper.ts`), file count, selection position
- [x] Help overlay in tabFiles mode shows Files keybindings (↑/↓, PgUp/PgDn, Enter, Backspace, /, ?, Esc, q)
- [x] Session-killed message handling: "session not found" → "Session no longer running" status line; tab stays open
- [x] `go test ./internal/tui/...` passes
- [x] No synchronous `os.ReadDir` / `os.Open` / `os.Stat` inside Update (grep gate = 0)
- [x] No modifications to `STATE.md` / `ROADMAP.md`

Plan 02 ready for Plan 03's integration test suite + UAT.
