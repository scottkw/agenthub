---
phase: 121
verified: 2026-05-20T00:00:00Z
status: human_needed
score: 5/5 must-haves verified
requirements_covered: 9/10
overrides_applied: 0
human_verification:
  - test: "Press 'f' in TUI on a local session, verify lipgloss-bordered Files view appears with TokyoNight palette, list pane left + preview pane right"
    expected: "Border in TokyoNight accent color; layout is two-pane horizontal (40/60 split) with status line below"
    why_human: "Visual rendering of lipgloss borders and TokyoNight palette colors is a perceptual check; user is colorblind so verify TokyoNight constants are wired (renderFilesTab uses m.styles.BorderAccent/BorderNormal — source-level palette wiring is verified, but per-eye lipgloss output requires a sighted spot-check)"
  - test: "Navigate into a subdirectory with Enter, then press Backspace at root cwd"
    expected: "Backspace at root cwd is a no-op (no error, no traversal above sandbox); navigation works smoothly without flicker"
    why_human: "Asynchronous tea.Cmd round-trip behavior under real terminal timing — automated tests verify the no-op return, but the user-perceived smoothness of nav through a real socket needs eyes on it"
  - test: "Open a 6 MB text file via 'f' → Enter on the file"
    expected: "Preview shows 'Too large to preview, use desktop or web to download' — file body is NOT transferred"
    why_human: "Need to confirm message text renders correctly inside the preview frame at typical terminal widths"
  - test: "Press '/' then type 'helper' to filter; press Esc"
    expected: "Filter pill appears at bottom of list pane while typing; Esc clears both filter value and active state; list returns to unfiltered"
    why_human: "Real-time filter UX feel (cursor stays on first match, typing is responsive) is not captured by snapshot tests"
  - test: "Press 'f' on a remote (tailnet) session"
    expected: "Toast appears: 'File browser not available for remote sessions in v3.4' — tabFiles does NOT open"
    why_human: "TUI-08 was deliberately scoped to refuse remote sessions for v3.4 (Plan 01 decision, documented in toast string). Need explicit user confirmation this descoping is accepted before release — see Requirements Gap below."
---

# Phase 121: TUI Files View Verification Report

**Phase Goal:** TUI users can browse and preview files for any session using keyboard navigation inside a lipgloss-bordered file browser pane — with the same sandboxed cwd constraint, type-ahead filter, and text/markdown preview available in the desktop and web surfaces.

**Verified:** 2026-05-20
**Status:** human_needed
**Re-verification:** No — initial verification after Phase 121 closure

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1   | `f` opens lipgloss-bordered Files view, list left + preview right, TokyoNight palette | VERIFIED | `internal/tui/update.go:381-409` (FilesOpen handler opens tabFiles + dispatches loadDirCmd); `files.go:316-328` (lipgloss.JoinHorizontal 40/60 split with sepCol); `files.go:124-138, 215-227` (wrapInFrame uses m.styles.BorderAccent/BorderNormal — TokyoNight palette source). Lipgloss/palette wiring verified at source level; perceptual confirmation in human_verification |
| 2   | Up/Down/PgUp/PgDn cursor, Enter enter dir, Backspace/Left up, Backspace at root no-op | VERIFIED | `files.go:431-509` (full nav dispatch with explicit `if m.files.cwd == "" \|\| m.files.cwd == "."` root-no-op guard at line 433); `TestHandleFilesKey_Backspace_AtRoot_NoOp` + `_Dot_NoOp` PASS |
| 3   | Text ≤5MB raw preview; .md → glamour ANSI; binary → "Use desktop or web to preview"; over-cap → "Too large to preview, use desktop or web to download" | VERIFIED | `files.go:582-660` (applyFilesHeadMsg decision tree + applyFilesReadMsg with `glamour.Render`); `previewSizeCap = 5 * 1024 * 1024` at line 538; exact refusal strings at lines 600, 607 |
| 4   | `/` filter (current dir only); Esc clears+dismisses; status line: cwd-relative path (left-truncated), file count, selection position | VERIFIED | `files.go:392-398` (Esc clears filter+blurs); `files.go:426-429` (/ activates filter); `files.go:362-381` (filteredEntries filters current dir only); `files.go:283-296` (renderFilesStatusLine: tail-aware pathBudget WR-06 fix); `TestRenderFilesStatusLine_*` PASS |
| 5   | All FS I/O via tea.Cmd (NO synchronous os.ReadDir/os.Open in Update); `?` help in tabFiles mode; key-dispatch priority above main view, below kill-confirm/new-session/QR/help | VERIFIED | Grep `\bos\.(ReadDir\|Open\|OpenFile\|Stat)\b` on files.go+files_cmds.go returns 0 matches; `TestFiles_NoSyncFSCalls` enforces this as a merge gate; `help.go:63-74` (tabFiles switch shows Files keybindings); `update.go:148-155` (Priority 5.5 sits between Priority 5 help and Priority 6 tab-cycling, below modals 1-4) |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/tui/files.go` | filesModel + handleFilesKey + apply*Msg helpers + renderFilesTab | VERIFIED | 661 lines, all symbols present, no stubs |
| `internal/tui/files_cmds.go` | loadDirCmd, readFileCmd, headFileCmd with context.WithTimeout | VERIFIED | 126 lines, 3× `context.WithTimeout` (1× 5s in loadDir, 1× 10s in readFile, 1× 5s in headFile), `errNilClient` sentinel for nil-DaemonClient |
| `internal/tui/files_test.go` | Coverage matrix + no-sync-FS guard + path-truncation + dispatch-priority tests | VERIFIED | 1055 lines, 10/10 TUI-NN subtests in TestFiles_Phase121_Requirements PASS |
| `internal/tui/files_integration_test.go` | End-to-end DaemonClient over Unix socket | VERIFIED | 378 lines, `//go:build !windows`, `TestFiles_Integration_LocalSessionEndToEnd` PASS race-clean |
| `internal/tui/update.go` | Priority 5.5 dispatch + filesListMsg/HeadMsg/ReadMsg cases | VERIFIED | Priority 5.5 at lines 148-155; 3 msg-switch cases at lines 108-115 |
| `internal/tui/help.go` | tabFiles-specific keybinding group | VERIFIED | `case tabFiles:` at line 64 lists Up/Down, PgUp/PgDn, Enter, Backspace/Left, Tab, /, Esc |
| `internal/tui/keys.go` | FilesOpen, FilterStart, FilterEsc, FilesFocusToggle KeyMap entries | VERIFIED | All 4 bindings present and used in handleContentKey + handleFilesKey |

### Key Link Verification

| From | To  | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Sessions tab 'f' key | tabFiles + loadDirCmd | handleContentKey:FilesOpen | WIRED | update.go:381-409 — opens tab, builds filesModel with sized panes, dispatches loadDirCmd on local entry; toast on remote |
| handleKey dispatcher | handleFilesKey | Priority 5.5 cascade | WIRED | update.go:153 — `if m.activeTabID() == tabFiles { return m.handleFilesKey(msg) }` between Priority 5 (help) and Priority 6 (tab-cycling) |
| tea.Cmd factories | DaemonClient.{ListFiles,ReadFile,HeadFile} | context.WithTimeout | WIRED | files_cmds.go:65-119 — each factory checks `client == nil`, runs 5s/10s context timeout, returns echo-field msg (sessionID + relPath + generation) |
| filesListMsg | applyFilesListMsg | Update switch | WIRED | update.go:108-109 — Update routes msg to apply helper; helper updates entries + resets preview (T-121-09) |
| filesHeadMsg → readFileCmd | applyFilesHeadMsg | tea.Cmd chain | WIRED | files.go:614 — HEAD result for text/* mime triggers readFileCmd with generation propagated for WR-03 |
| applyFilesReadMsg → glamour.Render | Markdown preview | suffix+mime check | WIRED | files.go:636-657 — `.md`/`.markdown`/`text/markdown` → glamour render with hasDark style; fallback to plain text on render error |
| filterInput width | terminal resize | renderFilesListPane every-render | WIRED | files.go:116-124 (WR-01 fix) — `m.files.filterInput.SetWidth(filterW)` re-applied each render |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| renderFilesListPane | m.files.entries | filesListMsg from loadDirCmd → DaemonClient.ListFiles → daemon /api/files/list | YES | FLOWING — integration test asserts entries `a.txt`, `b.md`, `sub/` appear in real round-trip |
| renderFilesPreviewPane | m.files.preview (viewport content) | filesReadMsg from readFileCmd → DaemonClient.ReadFile → daemon /api/files/read | YES | FLOWING — integration test asserts `preview.View()` contains `"alpha"` after Enter on `a.txt` |
| renderFilesStatusLine | m.files.cwd + len(entries) + selected | applyFilesListMsg mutation of cwd/entries/selected | YES | FLOWING — Backspace/Enter navigation updates cwd, verified by integration test (`sub` → root round-trip) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full TUI test suite race-clean | `go test ./internal/tui/... -race -count=1 -timeout 120s` | `ok github.com/scottkw/agenthub/internal/tui 1.291s` | PASS |
| Integration test (Unix socket DaemonClient) | `go test -run TestFiles_Integration_LocalSessionEndToEnd ./internal/tui/ -race -count=1 -v` | PASS in 0.02s | PASS |
| TUI-XX traceability matrix | `go test -run '^TestFiles_Phase121_Requirements$' ./internal/tui/ -v` | 10/10 TUI-NN subtests PASS | PASS |
| No-sync-FS static guard | `grep -E '\bos\.(ReadDir\|Open\|OpenFile\|Stat)\b' internal/tui/files.go internal/tui/files_cmds.go` | 0 matches | PASS |
| context.WithTimeout count in files_cmds.go | `grep -c "context.WithTimeout" internal/tui/files_cmds.go` | 3 | PASS |
| Project-wide tests (excl. unrelated security-review package layout failure) | `go test ./... -short -count=1` | All phase-relevant packages OK | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| TUI-01 | 121-01, 121-02 | files.go sub-model + lipgloss two-pane + TokyoNight | SATISFIED | TestFiles_TabID_Distinct + TestRenderFilesTab_BasicLayout PASS |
| TUI-02 | 121-01 | "Files" reachable per-session via Sessions list; opens scoped to cwd | SATISFIED | TestFiles_OpenFromSessions_LocalEntry + _RemoteEntry_ShowsToast + _EmptyList_NoOp + _ResetsModel PASS |
| TUI-03 | 121-01, 121-02 | Up/Down/PgUp/PgDn/Enter/Backspace nav with Backspace-at-root no-op | SATISFIED | TestHandleFilesKey_Backspace_AtRoot_NoOp + _Dot_NoOp + _NonRoot_DispatchesParent + Down_* PASS |
| TUI-04 | 121-02 | Text 5MB cap + glamour markdown + binary refusal + over-cap refusal | SATISFIED | TestApplyFilesHeadMsg_OverCap_RefusalMessage + _Binary_RefusalMessage + _Text_DispatchesRead + TestApplyFilesReadMsg_TextSetsContent + _MarkdownSuffix_UsesGlamour PASS |
| TUI-05 | 121-02 | `/` filter + Esc clears+dismisses; current-dir only | SATISFIED | TestHandleFilesKey_Slash_ActivatesFilter + _FilterActive_EscClears + _BackspaceDoesNotNavigate + _CtrlCQuits PASS |
| TUI-06 | 121-01, 121-02 | Status line: cwd-relative path (left-truncated) + count + selection | SATISFIED | TestTruncateLeft + TestRenderFilesStatusLine_TruncatedFlag + _ErrorShown + _LeftTruncation + TestFiles_PathTruncation_StatusLine PASS |
| TUI-07 | 121-01, 121-03 | ALL FS I/O via tea.Cmd (sync os.ReadDir in Update = merge-gate fail) | SATISFIED | TestFiles_NoSyncFSCalls static-grep guard + TestLoadDirCmd_DispatchesAsync PASS; 0 grep hits |
| **TUI-08** | 121-02 | TUI Files view works against local AND remote (tailnet) sessions | **PARTIAL** | TestFiles_OpenFromSessions_RemoteEntry_ShowsToast + TestFiles_Integration_LocalSessionEndToEnd PASS. **Local works fully; remote refused with toast "File browser not available for remote sessions in v3.4" (update.go:388). Coverage matrix entry exists but only asserts the refusal, not the implementation of remote-Tailscale-HTTPS path.** See "Requirements Gaps" below. |
| TUI-09 | 121-02 | Help overlay (?) updated with file-browser keybindings | SATISFIED | TestBuildHelpContent_FilesActive_ShowsFilesGroup + _SessionsActive_ShowsSessionsGroup PASS; help.go:64-74 renders Files group |
| TUI-10 | 121-01, 121-03 | Key-dispatch priority correctly above main view, below kill-confirm/new-session/QR/help | SATISFIED | TestFiles_HandleKey_DispatchPriority + _BelowKillConfirm + TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp PASS; Priority 5.5 placement in update.go:148-155 verified |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| _none_ | — | — | — | Zero TBD/FIXME/XXX/TODO/HACK/placeholder/stub-return matches in files.go, files_cmds.go, help.go |

### Code Review Closure (121-REVIEW.md findings)

| Finding | Type | Status | Evidence |
| ------- | ---- | ------ | -------- |
| CR-01 (Ctrl+C swallowed in filter mode) | BLOCKER | CLOSED | files.go:407-414 intercepts `ctrl+c` before forwarding to textinput; TestHandleFilesKey_FilterActive_CtrlCQuits asserts tea.QuitMsg returned |
| WR-01 (filterInput width not recomputed on resize) | WARNING | CLOSED | files.go:116-124 re-applies SetWidth every render |
| WR-02 (sepCol trailing newline → phantom row) | WARNING | CLOSED | files.go:326 wraps with `strings.TrimRight(..., "\n")` |
| WR-03 (reset-during-fetch race) | WARNING | CLOSED | Generation counter on filesModel (files.go:48-54, 438, 504, 508); all apply*Msg helpers discard `msg.generation < m.files.generation` (lines 551, 587, 626); HEAD→read propagates generation through msg.generation (line 614); TestApplyFilesListMsg_StaleGenerationDiscarded PASS |
| WR-04 (filteredEntries entries[:0:0] footgun) | WARNING | CLOSED | files.go:373 uses explicit `make([]daemon.FileEntry, 0, len(fm.entries))` |
| WR-05 (Up/Down no-op when previewFocused) | WARNING | CLOSED | files.go:448, 459 explicit ScrollUp(1)/ScrollDown(1) when previewFocused |
| WR-06 (status line pathBudget magic 40) | WARNING | CLOSED | files.go:289-294 computes tail string first and subtracts `lipgloss.Width(tail)` from pane width |
| IN-01..IN-05 | Info | Not all addressed | Info-level only; not required for phase closure |

All 1 BLOCKER + 6 WARNINGs from 121-REVIEW.md are closed in production code with covering tests.

### Requirements Gaps

**TUI-08 partial coverage:** The requirement text says "TUI Files view works against local AND remote (tailnet) sessions; uses the same daemon-local HTTP API for local and Tailscale HTTPS for remote (no relay frames)." The implementation refuses remote sessions with the toast string `"File browser not available for remote sessions in v3.4"` (update.go:388). The coverage matrix (TUI-08) lists `TestFiles_OpenFromSessions_RemoteEntry_ShowsToast` (asserts the refusal) and `TestFiles_Integration_LocalSessionEndToEnd` (covers local only). There is NO code path that invokes Tailscale HTTPS for remote file browsing in this phase.

This deviation is **explicitly documented** in:
1. Plan 01 SUMMARY's Task 3 description (entry kind branching to toast for remote).
2. The toast text itself, which is dated "in v3.4" — signaling explicit v3.4-scope refusal.

Per user memory ("Cross-surface parity is release-blocking — GUI/TUI/CLI must stay in sync; never defer a parity gap without explicit user sign-off"), this requires explicit user sign-off before release closure. Recommended dispositions (pick one before merge):

- **Option A — Update REQUIREMENTS.md:** Mark TUI-08 as "local-only in v3.4; remote-Tailscale Files deferred to v3.5" with a GitHub issue filed (matches the TUI-EDIT-03 "formally descoped with follow-up issue filed" pattern).
- **Option B — Promote to a gap:** Reopen Phase 121 (or open Phase 122) to add a remote-Tailscale-HTTPS path that reuses the same handleFilesKey + applyFilesListMsg/HeadMsg/ReadMsg pipeline against a different DaemonClient endpoint.

Either disposition is acceptable to the verifier; the BLOCKER vs WARNING classification is **WARNING** because (a) the descoping is documented in user-visible UI (the toast), (b) it has a covering test, and (c) the spirit-of-parity rule has a documented descope precedent (TUI-EDIT-03). It is included in the `human_verification` section above so the user can make the call.

### Human Verification Required

See `human_verification:` frontmatter section above. 5 items requiring human eyes on:

1. Visual TokyoNight palette and lipgloss border rendering.
2. Navigation smoothness through real Unix-socket round trips.
3. Over-cap refusal string rendering at typical terminal widths.
4. Filter UX feel (typing responsiveness, cursor reset on first match).
5. **Sign-off on TUI-08 descoping** for remote sessions (see Requirements Gaps).

### Gaps Summary

No goal-blocking gaps. All 5 ROADMAP success criteria are satisfied with verifiable evidence in code, all critical+warning review findings (CR-01, WR-01..WR-06) are closed in production source with covering tests, and 9/10 TUI-NN requirements are fully satisfied. The remaining TUI-10 → TUI-09 → TUI-08 cascade is:

- TUI-08 is **partially** satisfied: local works fully, remote refused with v3.4-dated toast. This is a documented descope, not an implementation defect. Final disposition depends on user sign-off (cross-surface parity rule).

Phase 121 is **fundamentally complete** at the code level. The verifier's status is `human_needed` (not `passed`) because (a) visual lipgloss/palette rendering is perceptual and (b) the TUI-08 descope needs explicit user acceptance per the cross-surface-parity user-memory rule.

---

_Verified: 2026-05-20_
_Verifier: Claude (gsd-verifier)_
