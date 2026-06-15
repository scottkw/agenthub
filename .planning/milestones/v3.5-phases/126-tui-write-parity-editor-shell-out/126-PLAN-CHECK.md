# Phase 126 Plan Check

**Checked:** 2026-06-14
**Plans reviewed:** 126-01, 126-02, 126-03, 126-04 (4 plans, 9 tasks total)
**Against:** TUIW-01..TUIW-07, CONTEXT.md, RESEARCH.md, PATTERNS.md, CLAUDE.md

---

## PLAN CHECK PASSED (with warnings requiring pre-execution awareness)

No blockers that prevent execution. Four warnings documented below — two require explicit
acknowledgment before execution starts (W1, W2).

---

## Dimension 1: Requirement Coverage — PASS

| Requirement | Plan | Tasks | Covered |
|-------------|------|-------|---------|
| TUIW-01 | 01 | Task 1 + Task 2 | FilesClient 8-method interface + both compile-time guards + RemoteFilesClient write methods |
| TUIW-02 | 02 | Task 1 + Task 2 | `e` key → editFetchCmd → tea.ExecProcess → editWriteBackCmd |
| TUIW-03 | 02 | Task 1 | resolveEditor() chain `$EDITOR→$VISUAL→nano→vim→vi` + no-editor exact copy |
| TUIW-04 | 02 | Task 2 | editorExitMsg → tea.ClearScreen + unconditional loadDirCmd |
| TUIW-05 | 03 | Task 1 + Task 2 | d/r/m key branches, confirm modal, inline input, priority dispatch |
| TUIW-06 | 04 | Task 1 + Task 2 + Task 3 | `u` descope message + gh issue create + blocking human checkpoint |
| TUIW-07 | 04 | Task 1 | TestFiles_NoSyncFSCalls extended with broadened regex |

All 7 TUIW requirements have covering tasks. Full coverage.

---

## Dimension 2: Task Completeness — PASS

All 9 tasks have `<files>`, `<action>`, `<verify>`, and `<done>` fields. All auto tasks have
`<automated>` verify commands. The Plan 04 Task 3 checkpoint has the correct `<what-built>`,
`<how-to-verify>`, and `<resume-signal>` elements. Acceptance criteria are concrete and
machine-checkable.

---

## Dimension 3: Dependency Correctness — PASS

```
01 (wave 1, depends_on: [])
  → 02 (wave 2, depends_on: ["126-01"])
    → 03 (wave 3, depends_on: ["126-02"])
      → 04 (wave 4, depends_on: ["126-03"])
```

Serial chain. No cycles. No missing references. Files shared across plans
(files_cmds.go, files.go, update.go) are protected by the serial dependency chain —
Plan 02 finishes before Plan 03 touches the same files.

---

## Dimension 4: Key Links Planned — PASS

All critical wiring is explicitly specified with `from/to/via/pattern`:

- `files.go (e branch)` → `editFetchCmd` via tea.Cmd dispatch (Plan 02)
- `update.go (editorExitMsg)` → `tea.ClearScreen + loadDirCmd` via `tea.Batch` always (Plan 02)
- `editWriteBackCmd` → `client.WriteFile` via tea.Cmd closure (Plan 02)
- `files.go (d/r/m branches)` → confirm modal / inline filesNameInput state (Plan 03)
- `update.go (handleKey priority ladder)` → `handleFileDeleteConfirmKey` above Priority 5.5 (Plan 03)
- `deleteCmd/renameCmd/mkdirCmd` → `client.DeleteFile/RenameFile/MkdirFile` (Plan 03)
- `files.go (u branch)` → `m.files.err` with descope message, no write cmd (Plan 04)

---

## Dimension 5: Scope Sanity — PASS

| Plan | Tasks | Files | Wave | Status |
|------|-------|-------|------|--------|
| 01   | 2     | 3     | 1    | Within budget |
| 02   | 2     | 5     | 2    | Within budget |
| 03   | 2     | 6     | 3    | Within budget |
| 04   | 3*    | 2     | 4    | Within budget |

*Plan 04 has 2 auto tasks + 1 checkpoint:human-verify. Checkpoint is exempt from scope limits.

---

## Dimension 6: Verification Derivation — PASS

All `must_haves.truths` are user-observable behavioral statements (not implementation details):
"Pressing e on a selected file…", "Pressing u… shows…", etc. Artifacts include concrete
`contains:` patterns that can be grep-verified. Key links connect artifacts with specific
patterns for the verifier.

---

## Dimension 7: Context Compliance — PASS (with W1)

Locked decisions:
- `e` → suspend → $EDITOR → resume + ClearScreen + unconditional refresh: COVERED (Plans 02)
- No-editor inline error verbatim copy: COVERED (Plan 02 Task 1/2)
- d/r/m with confirmation and refresh: COVERED (Plan 03)
- u → upload descope message + GitHub issue: COVERED (Plan 04)
- FilesClient = exactly 8 methods, both implementers satisfy, no-sync gate: COVERED (Plans 01+04)

Deferred ideas excluded: TUI upload is correctly descoped (message + issue, no write attempt).

**W1 (WARNING — requires pre-execution acknowledgment):** CONTEXT.md and ROADMAP SC#1 say
"spawn `$EDITOR` with the file's sandbox-absolute path." Plan 02 explicitly deviates from this,
spawning the editor with a `/tmp/agenthub-edit-*` temp file path instead. This is documented
in Plan 02's objective NOTE with technical justification: `DaemonClient` has no
`GetSessionWorkDir` method (server-side only, api.go:85), and a literal absolute path breaks
remote parity. The milestone ARCHITECTURE.md §5.3/§7.3 ratified the temp-file design. Plan 02
instructs the verifier to check BEHAVIOR (edit → WriteFile-back → refresh for local AND remote),
not a literal path string.

This is NOT classified as scope reduction (Dimension 7b) because the deviation is explicit and
technically forced, not silent. But it IS a deviation from the literal locked decision text.

**Action required:** User should confirm awareness that `$EDITOR` will receive a temp file path
(`/tmp/agenthub-edit-*`) rather than the project-directory path. Practical consequence: the
editor's title bar shows a temp filename, autosave/swap files land in `/tmp`, and `:e %` in vim
expands to the temp path. If in-place editing is preferred for local sessions, a new
`GetSessionWorkDir` daemon API would be needed (out of scope for this phase).

---

## Dimension 7b: Scope Reduction — PASS

No silent scope reductions found. The only deviation from locked-decision text (the
"sandbox-absolute path" → temp-file substitution) is explicitly documented in Plan 02's
objective with full technical justification. Language like "v1", "static for now",
"future enhancement", or "not wired to" does not appear in plan actions.

---

## Dimension 7c: Architectural Tier Compliance — PASS

The Architectural Responsibility Map in RESEARCH.md assigns:
- `$EDITOR` spawn/suspend: TUI (Bubble Tea) tier → Plans 02 Task 2 assigns to TUI. Correct.
- Read/Write bytes: TUI → DaemonClient/RemoteFilesClient → Plans 01/02 assign to TUI client tier. Correct.
- Delete/rename/mkdir: TUI → DaemonClient/RemoteFilesClient → Plan 03 assigns to TUI client tier. Correct.
- Server sandbox enforcement: daemon server tier → not modified in these plans. Correct.
- No security-sensitive logic crossed tier boundaries.

---

## Dimension 8: Nyquist Compliance — PASS

VALIDATION.md exists (`126-VALIDATION.md`). All auto tasks have `<automated>` verify commands:

| Task | Plan | Wave | Automated Command | Status |
|------|------|------|-------------------|--------|
| 1 | 01 | 1 | `go build ./internal/tui/... && grep -q '...'` | present |
| 2 | 01 | 1 | `go test ./internal/tui/ -run 'TestRemoteFilesClient_(Write|Delete|Rename|Mkdir)'` | present |
| 1 | 02 | 2 | `go test ./internal/tui/ -run 'TestResolveEditor'` | present |
| 2 | 02 | 2 | `go test ./internal/tui/ -run 'TestHandleFilesKey_Edit|TestEditorExit_RefreshesUnconditionally'` | present |
| 1 | 03 | 3 | `go build ./internal/tui/... && go test ./internal/tui/ -run 'TestFilesOpCmd|TestFilesDelete|TestFilesRename|TestFilesMkdir'` | present |
| 2 | 03 | 3 | `go test ./internal/tui/ -run 'TestFilesDelete|TestFilesRename|TestFilesMkdir'` | present |
| 1 | 04 | 4 | `go test -race ./internal/tui/... ./internal/daemon/... -count=1 && gofmt -l internal/tui/ | (! grep .)` | present |
| 2 | 04 | 4 | `gh issue list --repo scottkw/agenthub --search "..." --json number,title --jq 'length > 0'` | present |

No watch-mode flags. No consecutive windows of 3 tasks without automated verify. Feedback
latency within 60 seconds. Wave 0 tests are created inline (TDD) within each plan's first task —
no separate Wave 0 plan required (the TDD pattern satisfies Wave 0 intent).

---

## Dimension 9: Cross-Plan Data Contracts — PASS

`filesOpMsg{sessionID, generation, op, err}` is defined in Plan 02 (files_edit.go) and consumed
by Plan 03 (deleteCmd/renameCmd/mkdirCmd return it; update.go case filesOpMsg handles it). Both
plans are in the same Go package `tui` — no duplicate type definition; sharing via package scope
is correct. Generation discipline is consistent: both Plan 02 and Plan 03 bump
`m.files.generation++` before dispatching write cmds and stamp the result msg.

---

## Dimension 10: CLAUDE.md Compliance — PASS

- Go conventions (gofmt, golangci-lint, ctx context.Context): All tasks explicitly run gofmt +
  golangci-lint and use context.WithTimeout per convention.
- Python venv: Not applicable (Go-only phase).
- pnpm: Not applicable.
- Cross-surface parity: Explicitly required; plans call this out as release-blocking.
- GitHub issues drive release planning: Plan 04 Task 2 + Task 3 enforce this for the upload gap.
- Colorblind user: Plan 03 Task 2 + PATTERNS.md explicitly enforce non-color danger signals
  (explicit "Delete"/"Cannot be undone" text, FgDanger as reinforcement only).
- No new Go modules: Confirmed (zero new dependencies; all stdlib or already-vendored bubbletea/v2).

---

## Dimension 11: Research Resolution — WARNING (W2)

**W2 (WARNING):** `126-RESEARCH.md` has a `## Open Questions` section without the `(RESOLVED)`
suffix. None of the three listed questions have inline `RESOLVED:` markers.

However, all three questions have substantive dispositions embedded in the research:
- Q1 (sandbox-absolute path vs temp-file): Decided by milestone ARCHITECTURE.md §5.3/§7.3;
  Plan 02 encodes the resolution. Research Summary (line 42) says "RESOLVED". The formal marker
  is missing but the resolution is documented and implemented.
- Q2 (remote cap files.write): Explicitly deferred to Phase 128 with clear rationale. This is
  a scope boundary, not an unresolved technical question.
- Q3 (recursive dir delete copy): Recommendation given; Plan 03 implements distinct copy for dirs.

Per strict Dimension 11, this is a formal gate failure. Per practical assessment, all questions
have answers that the plans implement. The risk from missing RESOLVED markers is that a future
reader might re-litigate these decisions.

**Fix:** Add `(RESOLVED)` to the section header and inline RESOLVED markers to each question
before execution to satisfy the gate. Low effort.

---

## Dimension 12: Pattern Compliance — PASS

All files in PATTERNS.md `## File Classification` have corresponding plan tasks that reference
the specified analogs:
- `files_client.go`: Plan 01 Task 1 reads `files_client.go` + `remote_files_client.go:40` + `daemon/client.go:513-661`
- `remote_files_client.go`: Plan 01 Task 2 reads existing read methods + daemon write methods
- `files_cmds.go`: Plans 02+03 read `loadDirCmd` (canonical analog) before implementing
- `files.go`: Plans 02+03 read `handleFilesKey` switch (exact analog)
- `update.go`: Plans 02+03 read `attachDoneMsg`, `handleKillConfirmKey`, `handleRenameKey` (exact analogs)
- `model.go/modal.go`: Plan 03 reads `modalKillConfirm` iota + `renderKillConfirmModal` (exact analogs)
- `files_test.go`: Plan 04 reads `TestFiles_NoSyncFSCalls` + `TestFiles_Phase121_Requirements` (exact analogs)

Shared patterns (FilesClient 8-method contract, generation discipline, nil-client guard,
CAP-LEAK invariant, colorblind-safe danger signal) are all addressed by the plans that create
files they apply to.

---

## Additional Issues Found

### W3 (WARNING): Plan 02 `files_modified` includes `files_cmds.go` but task actions do not modify it

Plan 02's `files_modified` frontmatter lists `internal/tui/files_cmds.go`, but neither Task 1
nor Task 2 in Plan 02 modifies `files_cmds.go`. Task 1 creates `files_edit.go` (the new file
for edit cmds/types), and Task 2 modifies `files.go` and `update.go`. The `files_cmds.go` entry
appears to be a vestigial carry-over from an earlier plan layout where edit cmds lived in
`files_cmds.go`. Plan 03 correctly claims `files_cmds.go` for its own deleteCmd/renameCmd/mkdirCmd.
This does not cause a correctness issue (the file is claimed by only one plan's actions), but it
creates a misleading frontmatter entry that could confuse the executor.

**Fix:** Remove `internal/tui/files_cmds.go` from Plan 02's `files_modified` list.

### W4 (WARNING): Cross-plan context `@` paths are incorrect

Plans 02, 03, and 04 load prior plans as context:
- Plan 02: `@.planning/phases/126-01-PLAN.md` (WRONG)
- Plan 03: `@.planning/phases/126-02-PLAN.md` (WRONG)
- Plan 04: `@.planning/phases/126-03-PLAN.md` (WRONG)

Actual paths are:
- `.planning/phases/126-tui-write-parity-editor-shell-out/126-01-PLAN.md`

These context loads will silently fail at execution time (file not found). The functional impact
is limited because each plan's `<interfaces>` section already copies the key method signatures
and patterns from prior plans. However, an executor that relies on the loaded context for
architectural decisions will be working with incomplete information.

**Fix:** Correct the paths to include the full subdirectory:
`@.planning/phases/126-tui-write-parity-editor-shell-out/126-0N-PLAN.md`

### W5 (WARNING): Plan 02 `<behavior>` test spec implies write-back skipped on `exitErr != nil`

The Plan 02 Task 2 `<behavior>` element states:
> "TestEditorExit_RefreshesUnconditionally: handling editorExitMsg with exitErr != nil STILL
> returns a batch containing a ClearScreen-yielding cmd AND a loadDir-yielding cmd;
> **with exitErr == nil same batch plus write-back**"

The phrasing "with exitErr == nil same batch plus write-back" implies write-back is NOT in the
batch when `exitErr != nil`. However, the `<action>` text, the `<done>` criteria ("always batches
tea.ClearScreen + write-back + loadDirCmd **regardless of exitErr**"), and the RESEARCH code
example (lines 331-342) all specify that `editWriteBackCmd` is appended unconditionally.

The test as specified will only assert ClearScreen+loadDirCmd on the error path — it will NOT
verify that write-back fires on `exitErr != nil`. An executor following the `<action>` will
implement unconditional write-back (correct), but the test won't catch a bug where write-back
is conditioned on `exitErr == nil`.

**Fix:** Amend Plan 02 Task 2 `<behavior>` to explicitly state:
> "with exitErr != nil: batch contains ClearScreen-yielding cmd, write-back cmd, AND loadDir-yielding cmd"
And add this assertion to `TestEditorExit_RefreshesUnconditionally`'s exitErr!=nil subtest.

---

## 5 Success Criteria Trace

| SC | Requirement | Plan | Verified by |
|----|-------------|------|-------------|
| SC#1: `e`→suspend→$EDITOR→resume+ClearScreen+unconditional refresh | TUIW-02/03/04 | 02 | `TestHandleFilesKey_Edit` + `TestEditorExit_RefreshesUnconditionally`; manual terminal UAT (deferred to verify-work) |
| SC#2: No-editor inline error exact copy | TUIW-03 | 02 | `TestHandleFilesKey_Edit` no-editor subcase; acceptance_criteria grep |
| SC#3: d/r/m confirm-delete/rename/mkdir with refresh | TUIW-05 | 03 | `TestFilesDelete`/`TestFilesRename`/`TestFilesMkdir` |
| SC#4: `u`→descope message + GitHub issue | TUIW-06 | 04 | `TestFilesUpload_Descoped` + `gh issue list` verify + human checkpoint |
| SC#5: FilesClient=8 methods, both clients satisfy, TestFiles_NoSyncFSCalls passes with write commands | TUIW-01/07 | 01+04 | compile-time guards in Plan 01 + extended gate in Plan 04 |

---

## Structured Issues

```yaml
issues:

  - plan: "126-RESEARCH.md"
    dimension: research_resolution
    severity: warning
    description: "## Open Questions section lacks (RESOLVED) suffix; no inline RESOLVED markers on questions 1-3"
    fix_hint: "Add '(RESOLVED)' to section header; add 'RESOLVED: [decision]' inline to each question before execution"

  - plan: "126-01"
    dimension: context_compliance
    severity: warning
    description: "CONTEXT/ROADMAP locked decision says 'spawn $EDITOR with the sandbox-absolute path'; Plan 02 spawns editor with a /tmp/agenthub-edit-* temp path instead. Not silent (Plan 02 documents this in its objective NOTE) but requires user acknowledgment."
    fix_hint: "User should confirm awareness before execution. If in-place editing is required for local sessions, a new GetSessionWorkDir daemon API would be needed (Phase 127+ scope). The temp-file design is technically forced for remote parity."

  - plan: "126-02"
    dimension: task_completeness
    severity: warning
    description: "files_modified frontmatter lists internal/tui/files_cmds.go but neither task in Plan 02 modifies that file (Task 1 creates files_edit.go, Task 2 modifies files.go + update.go)"
    task: null
    fix_hint: "Remove internal/tui/files_cmds.go from Plan 02 files_modified"

  - plan: "126-02"
    dimension: task_completeness
    severity: warning
    description: "<behavior> test spec implies editWriteBackCmd is NOT dispatched when exitErr != nil ('with exitErr == nil same batch plus write-back'), contradicting <action>/<done> which say write-back fires unconditionally. Test will not catch a regression where write-back is gated on exitErr == nil."
    task: 2
    fix_hint: "Amend <behavior> to assert write-back IS present in the exitErr != nil batch; add assertion to TestEditorExit_RefreshesUnconditionally"

  - plan: "126-02"
    dimension: key_links_planned
    severity: warning
    description: "Cross-plan context @ paths are wrong in Plans 02/03/04 — '@.planning/phases/126-01-PLAN.md' does not exist; actual path includes subdir '126-tui-write-parity-editor-shell-out/'. Context loads will silently fail at execution time."
    fix_hint: "Update all cross-plan @ references to @.planning/phases/126-tui-write-parity-editor-shell-out/126-0N-PLAN.md"
```

---

## Recommendation

**4 warnings found, 0 blockers.** Execution can proceed. Two warnings require explicit
acknowledgment before starting:

1. **W1 (sandbox-absolute path):** User should confirm the temp-file editing design is acceptable
   before Plan 02 executes. Editors will show `/tmp/agenthub-edit-*` in their title bar. The
   milestone architecture already ratified this.

2. **W2 (RESEARCH.md Open Questions):** Add `(RESOLVED)` markers to the three Open Questions
   before execution — takes 2 minutes and satisfies the formal gate.

W3 (vestigial files_cmds.go in Plan 02 frontmatter) and W4 (wrong @ paths) are quality/clarity
fixes that reduce confusion for the executor but do not prevent correct execution (the interfaces
sections carry the necessary context inline).

W5 (write-back test spec ambiguity) should be fixed in Plan 02 Task 2 to prevent a future test
coverage gap for the TUIW-04 unconditional-refresh requirement.
