---
phase: 121
status: findings
critical_count: 1
warning_count: 6
info_count: 5
files_reviewed: 10
files_reviewed_list:
  - internal/tui/files.go
  - internal/tui/files_cmds.go
  - internal/tui/files_test.go
  - internal/tui/files_integration_test.go
  - internal/tui/model.go
  - internal/tui/keys.go
  - internal/tui/update.go
  - internal/tui/view.go
  - internal/tui/help.go
  - go.mod
---

# Phase 121: Code Review Report

**Reviewed:** 2026-05-20T00:00:00Z
**Depth:** standard (with cross-file tracing into `internal/files`, `internal/daemon`, and `charm.land/bubbles/v2/textinput`)
**Files Reviewed:** 10
**Status:** issues_found

## Summary

The Phase 121 TUI Files view implementation is largely well-structured: the async-only FS rule is enforced via tea.Cmd factories, context.WithTimeout guards all three commands, the Backspace-at-root no-op closes the obvious traversal seam, the filter-mode cascade correctly handles TUI-PITFALL-2, and the merge-gate test (`TestFiles_NoSyncFSCalls`) is present and effective. The structural pre-pass (test coverage matrix, integration test wiring) is solid.

That said, there is **one BLOCKER** — Ctrl+C is silently swallowed when the filter input is active, breaking the universal "Ctrl+C always quits" terminal contract. There are also several WARNINGs around resize handling, layout math, and stale-result races, plus a handful of Info items.

## Critical Issues

### CR-01: Ctrl+C is swallowed by filter textinput; user cannot quit while filtering — BLOCKER

**File:** `internal/tui/files.go:352-376`
**Issue:** When `m.files.filterActive == true`, the dispatcher routes every key except `esc` and `enter` straight into `m.files.filterInput.Update(msg)`. The bundled `charm.land/bubbles/v2/textinput` Update (verified at `charm.land/bubbles/v2@v2.1.0/textinput/textinput.go:580-648`) has **no Ctrl+C handling**; ctrl+c falls into the `default` branch (`insertRunesFromUserInput([]rune(msg.Text))`), which inserts nothing visible but also returns no quit command. Net effect: while the user is typing a filter, pressing `Ctrl+C` does NOT terminate the TUI — they must press Esc first to leave filter mode, then Ctrl+C. This violates the standard terminal-app contract (every TUI in this codebase honors Ctrl+C unconditionally — see Quit binding in `keys.go:32-35`). The same applies to `Q`, but `Q` is reasonable to swallow as a literal filter char; `Ctrl+C` is not, because it has no legitimate filter use.

Note this is *not* covered by Priority 5.5: `handleFilesKey` is the dispatcher, and the filter-mode early branch returns before `key.Matches(msg, m.keys.Quit)` is reached on lines 474-475.

**Fix:**
```go
// Filter-mode capture (Pitfall TUI-PITFALL-2).
if m.files.filterActive {
    switch s {
    case "esc":
        // …existing clear-and-blur path…
    case "enter":
        // …existing accept path…
    case "ctrl+c":
        return m, tea.Quit
    }
    var cmd tea.Cmd
    m.files.filterInput, cmd = m.files.filterInput.Update(msg)
    m.files.selected = 0
    return m, cmd
}
```
Add a covering test parallel to `TestHandleFilesKey_FilterActive_EscClears` asserting that `ctrl+c` returns `tea.Quit` even when the filter is active.

## Warnings

### WR-01: Filter textinput width is never recomputed on terminal resize

**File:** `internal/tui/files.go:63-71`, `internal/tui/update.go:24-27`
**Issue:** `newFilesModel` sets `filterInput.SetWidth(listW - 4)` exactly once, at the moment `f` is pressed. The `tea.WindowSizeMsg` handler in `update.go:24-27` updates `m.width`/`m.height` but never propagates the new pane geometry into `m.files.filterInput`. After a horizontal resize the filter input is still sized for the original list pane — too narrow for the new pane, or worse, wider than the new pane (cursor positions fall off-pane). The preview viewport handles this correctly because `renderFilesPreviewPane` re-applies `SetWidth/SetHeight` on every render; the filter input has no equivalent re-sync.
**Fix:** Either (a) re-compute and apply `filterInput.SetWidth(...)` inside `renderFilesListPane` on every render the same way the preview viewport is re-sized, or (b) add a case to `Update` that propagates `tea.WindowSizeMsg` into `m.files.filterInput` when `m.activeTabID() == tabFiles`. Option (a) is more robust because it also covers tab switching back to Files after a resize.

### WR-02: Separator column in renderFilesTab has a trailing newline → potential phantom row

**File:** `internal/tui/files.go:292-296`
**Issue:** `sepCol := lipgloss.NewStyle().Foreground(...).Render(strings.Repeat("│\n", bodyH))` produces `bodyH` `│\n` pairs — that is `bodyH` separator lines plus a trailing empty line. When `lipgloss.JoinHorizontal(lipgloss.Top, listPane, sepCol, previewPane)` aligns three columns, the column with a trailing newline is one line taller than the others. This either (a) creates a blank row at the bottom of the Files tab, pushing the status line down by one or (b) gets silently clipped — either way it's a fragile dependency on lipgloss-internal padding behavior. The same pattern is used in `view.go:69-70` for the main sidebar separator, which suggests it's an accepted house style, but the depth doubles for taller terminals.
**Fix:** Drop the trailing newline:
```go
sepCol := lipgloss.NewStyle().
    Foreground(m.styles.BorderNormal).
    Render(strings.TrimRight(strings.Repeat("│\n", bodyH), "\n"))
```
Or build a slice and `strings.Join(... , "\n")`.

### WR-03: Reset-during-fetch race can apply a stale parent-dir listing over a fresh root listing

**File:** `internal/tui/files.go:492-517`, `internal/tui/update.go:381-410`
**Issue:** The stale-msg filter on `applyFilesListMsg` only discards messages from a *different* session ID. If the user opens Files for session A, navigates into `subdir/`, then presses `f` again on the same session to reset, `newFilesModel` is rebuilt and a fresh `loadDirCmd(".", ...)` is dispatched. If the previously in-flight `loadDirCmd("subdir", ...)` completes between the two events (network jitter, slow disk inside the sandbox), it lands on the freshly-reset model with `sessionID == sessionA`, passes the stale check, and sets `m.files.cwd = "subdir"` plus stale entries. The subsequent root-listing reply then overwrites it — but for the duration of the race the UI shows the wrong cwd. With heavier post-Plan-02 backends (e.g. remote Files in v4.x) this widens to a real correctness defect.
**Fix:** Add a monotonic "generation" counter on `filesModel` and stamp every outgoing `loadDirCmd` / `headFileCmd` / `readFileCmd` with the current generation; bump on every reset (newFilesModel) and every Enter/Backspace. In `applyFilesListMsg` / `applyFilesHeadMsg` / `applyFilesReadMsg` discard messages whose generation < current. Pattern follows the same shape as session-cwd guard but covers intra-session reordering.

### WR-04: filteredEntries() uses `entries[:0:0]` slice header but reads aliased data — fragile if append is ever made in-place

**File:** `internal/tui/files.go:330-342`
**Issue:** `out := fm.entries[:0:0]` creates a zero-length, zero-capacity header that still points at `&fm.entries[0]`. Today's `append` correctly allocates new backing storage on the first growth (cap=0), so the result is safe. But this pattern is unusual and brittle: a future maintainer who "optimises" by changing it to `out := fm.entries[:0]` (cap unchanged) would silently corrupt `fm.entries` because every filter match would overwrite the first N slots of the source array. The receiver is also value-typed (`fm filesModel`), so `fm.entries` is itself a copy of the slice header — but the backing array is shared. This is a footgun, not a current bug.
**Fix:** Use an explicit allocation that documents intent:
```go
out := make([]daemon.FileEntry, 0, len(fm.entries))
```
The allocation cost is negligible (entries are capped to a few hundred by the daemon's truncation limit).

### WR-05: Up/Down routed to viewport when previewFocused is a no-op in the bundled viewport

**File:** `internal/tui/files.go:394-415`
**Issue:** When `m.files.previewFocused == true`, Up/Down arrow keys are dispatched to `m.files.preview.Update(msg)`. The bundled `charm.land/bubbles/v2/viewport` only scrolls on PgUp/PgDn/Home/End/MouseWheel (verified in viewport.go); Up/Down arrow keys are not in its default KeyMap. So pressing Up while previewFocused does literally nothing — neither cursor-move (list is unfocused) nor viewport-scroll (viewport ignores Up). Users will report "the arrow keys are dead in preview mode".
**Fix:** Either (a) explicitly translate Up/Down to viewport line-scroll while previewFocused, or (b) document the limitation and rely on PgUp/PgDn — but then the hint bar ("Up/Down  Enter Open  Backspace Up …" at `view.go:730`) is misleading. Recommend (a):
```go
case key.Matches(msg, m.keys.Up):
    if m.files.previewFocused {
        m.files.preview.LineUp(1)
        return m, nil
    }
    // …existing list-cursor path…
```

### WR-06: Status line `pathBudget` ignores ANSI rendering and entry-count width

**File:** `internal/tui/files.go:235-271`
**Issue:** `pathBudget := w - 40` reserves 40 columns for the `• N entries(…) • i/N` tail. But the tail's actual width depends on `N`: a directory with 9999 entries renders as ` • 9999 entries • 9999/9999` = ~26 chars + the `(truncated)` flag = 36 chars. Close to 40 but not quite, AND the daemon's truncation cap is configurable so a future bump to 10k+ entries could push the tail past 40 chars and cause the leading path to render past `w`, with the lipgloss `.Width(w)` then clipping the path from the right (losing the leaf segment — the opposite of what `truncateLeft` was designed to preserve). The magic 40 is also untested.
**Fix:** Compute the tail string first, then subtract its actual width:
```go
tail := fmt.Sprintf(" • %d entries%s • %d/%d", n, trunc, sel, n)
pathBudget := w - lipgloss.Width(tail)
if pathBudget < 10 {
    pathBudget = 10
}
pathPart := truncateLeft(displayPath, pathBudget)
body := pathPart + tail
```

## Info

### IN-01: Hint bar advertises "Backspace Up" but Left also navigates up; right arrow is silently unbound

**File:** `internal/tui/view.go:729-731`, `internal/tui/files.go:385`
**Issue:** `handleFilesKey` treats both `backspace` and `left` as "up one directory" (line 385), but the hint bar only mentions Backspace. The help overlay (`help.go:70`) does say "Backspace, Left" — so the help is correct and the hint bar is the inconsistent one. Also, `right` arrow has no binding — given vim users will expect `h`/`l` to map to left/right and Backspace, the unbound `l` is at minimum a documentation gap.
**Fix:** Either drop the `left` mapping (rely on the documented Backspace) or update the hint bar to "Backspace/← Up". Same review on whether `h`/`l` vim semantics deserve a binding.

### IN-02: `filterInput.CharLimit = 128` is undocumented and inconsistent with other inputs

**File:** `internal/tui/files.go:71`
**Issue:** `m.dirInput`/`m.argsInput` use `CharLimit = 256` (`update.go:646, 652`). The filter input uses 128. No comment explains why. Either align to the same constant or document the rationale (e.g. "filter prefix length is bounded by realistic filenames").
**Fix:** `fi.CharLimit = 256` to match, or add a comment explaining the difference.

### IN-03: Magic numbers `5 * 1024 * 1024`, `5*time.Second`, `10*time.Second` lack a shared source-of-truth

**File:** `internal/tui/files.go:485`, `internal/tui/files_cmds.go:65, 85, 106`
**Issue:** The 5 MiB preview cap is hardcoded both client-side (TUI) and server-side (daemon — per the comment at files.go:484). Same for the 5s / 10s timeouts. If the daemon's limit is ever bumped, the TUI silently clips below the daemon's new cap, and vice versa. A shared constant package or a configuration knob would prevent drift.
**Fix:** Move `previewSizeCap` to `internal/files/limits.go` (or wherever the daemon-side cap lives) and import from both layers. For timeouts, consider exposing them as package-level vars so the integration test can shrink them deterministically.

### IN-04: `_ = listH // reserved for Plan 02 list-pane sizing` is dead code after Plan 02 shipped

**File:** `internal/tui/files.go:81`
**Issue:** Plan 02 has shipped (per the file-coverage tests and dispatcher being complete). The `_ = listH` placeholder is no longer "reserved"; either the param should be used or removed. As-is it is a fossil comment.
**Fix:** Either use `listH` to size the embedded viewport list pane (it's used implicitly via `renderFilesListPane` re-computing innerH on every render — so it's truly unused at construction), or drop the parameter from `newFilesModel`'s signature.

### IN-05: `TestFiles_NoSyncFSCalls` regex would not catch `os.ReadFile` or `os.WriteFile`

**File:** `internal/tui/files_test.go:790`
**Issue:** The regex `\bos\.(ReadDir|Open|OpenFile|Stat)\b` is good but does NOT match `os.ReadFile`, `os.WriteFile`, `os.Lstat`, or `os.Create` — all of which would also be synchronous-FS regressions inside the Update path. The merge gate is narrower than its docstring implies ("Update path must not call synchronous FS").
**Fix:** Broaden the regex to catch all `os.*File*` and `os.*stat*` variants, e.g.:
```go
re := regexp.MustCompile(`\bos\.(ReadFile|WriteFile|ReadDir|Open|OpenFile|Create|Stat|Lstat|Remove|RemoveAll|Mkdir|MkdirAll)\b`)
```

## Structural Findings (fallow)

_No structural pre-pass payload was provided to this review (no `<structural_findings>` block in the prompt). This section is intentionally empty._

---

_Reviewed: 2026-05-20T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

## REVIEW COMPLETE
