# Phase 136: TUI Removal - Research

**Researched:** 2026-06-19
**Domain:** Go package deletion / dependency pruning / cross-package seam analysis
**Confidence:** HIGH

## Summary

Phase 136 is a pure deletion phase. The goal is to remove the `agenthub tui` command and all Bubble Tea infrastructure, narrowing the cross-surface parity contract to GUI/CLI/web. The central risk is not what to build — it is correctly mapping the TUI footprint so that shared code is not accidentally broken and the build stays green throughout.

The codebase has been fully audited (live tree only; `.claude/worktrees/**` excluded). The TUI footprint is well-contained. The `internal/tui` directory holds 32 files (18 source, 14 test). Exactly three external files import `internal/tui`: `cmd_tui.go`, `internal/daemon/remote_files_parity_test.go`, and `internal/daemon/remote_files_write_parity_test.go`. All charm.land/charmbracelet dependencies are exclusively consumed inside `internal/tui`; no shared code outside that package imports them.

The two daemon parity test files are the highest-complexity deletion. They test "TUI RemoteFilesClient vs daemon proxy must be byte-identical" — a TUI parity property that becomes moot after deletion. However, they also implicitly exercise the daemon proxy through `newDaemonAPIWithUpstreamCert` and `api.Handler()`. This daemon proxy behavior is fully covered by independent tests in `remote_files_test.go`, `relay_remote_files_test.go`, and `remote_files_write_test.go` — all of which have no TUI import. The parity tests may therefore be deleted outright rather than edited; the surviving daemon tests provide adequate coverage of daemon proxy correctness.

**Primary recommendation:** Delete `internal/tui/` and `cmd_tui.go` first (remove the package), then delete the three external files that import it (`cmd_tui.go` already scheduled, plus the two daemon parity test files). Remove the `case "tui":` dispatch from `main.go` and update `usage()` in `cmd_cli.go`. Run `go mod tidy` to drop the four charm.land direct dependencies. The build will be green immediately after the import references are removed, with no extraction or relocation work required.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| TUI command dispatch | CLI / main package | — | `case "tui":` switch in `main.go`; delegated to `cmdTUI()` in `cmd_tui.go` |
| TUI session list / modals / keys | `internal/tui` | — | Pure TUI; no non-TUI consumer |
| TUI attach flow | `internal/tui` | `internal/attach` (shared) | TUI attach.go wraps shared `internal/attach.AttachSession`; the shared package survives |
| TUI remote files client | `internal/tui` | daemon proxy | `RemoteFilesClient` and `FilesClient` interface are TUI-only; daemon proxy covers the GUI/web surface independently |
| Cross-surface parity proofs | `internal/daemon` (test) | `internal/tui` (oracle) | After deletion, daemon-only tests in `remote_files_test.go` cover proxy behavior; TUI oracle becomes unnecessary |

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NAV-01 | TUI surface removed entirely — `agenthub tui` no longer exists; Bubble Tea views, TUI-only shared code, and their tests deleted; cross-surface parity contract narrows to GUI/CLI/web | Full deletion map established; exact files enumerated below |
| TEST-06 | TUI tests are removed (not migrated) as part of TUI removal | 14 test files in `internal/tui`; 2 daemon parity test files delete; no TUI tests migrate |
</phase_requirements>

---

## Research Findings

### 1. Deletion Inventory

#### Files to delete outright (no survivors)

**Directory: `internal/tui/` — delete entire directory (32 files)**

Source files (18):
- `attach.go` — TUI-only attach wrapper using `internal/attach.AttachSession`
- `cmds.go` — Bubble Tea command factories (fetchSessions, fetchWebStatus, etc.)
- `files.go` — TUI Files view (two-pane list + preview, ansi.Strip, glamour render)
- `files_client.go` — `FilesClient` interface + compile-time guard that `*daemon.DaemonClient` satisfies it
- `files_cmds.go` — Bubble Tea commands for async file I/O
- `files_edit.go` — `$EDITOR` shell-out for TUI write parity
- `files.go` — Files view rendering
- `help.go` — TUI help overlay
- `joincode_prompt.go` — Join-code modal for remote session cap acquisition
- `keys.go` — Bubble Tea key map
- `modal.go` — New-session / kill-confirm / join-code modals
- `model.go` — TUI `Model` struct (all TUI state)
- `qr.go` — QR code overlay (lipgloss-rendered)
- `remote_files_client.go` — `RemoteFilesClient` struct, `NewRemoteFilesClientForTest` (exported for tests), sentinel errors `ErrRemotePeerNoWriteSupport` / `ErrRemoteCapExpired`
- `styles.go` — Lipgloss style definitions (TokyoNight palette)
- `tui.go` — `Run()` entrypoint; `newModel()`
- `update.go` — Bubble Tea `Update()` message handler (largest file)
- `view.go` — Bubble Tea `View()` render function

Test files (14) — delete outright per TEST-06:
- `attach_test.go`
- `files_edit_test.go`
- `files_integration_test.go`
- `files_ops_test.go`
- `files_test.go`
- `help_test.go`
- `integration_test.go`
- `joincode_prompt_test.go`
- `modal_test.go`
- `remote_files_client_test.go`
- `styles_test.go`
- `update_remote_test.go`
- `update_test.go`
- `view_test.go`

**Top-level file: `cmd_tui.go`**

Entire file is TUI-only: defines `cmdTUI()` which imports `internal/tui` and constructs `tui.ListRemoteGroup` / `tui.RemoteSessionEntry` / calls `tui.Run()`. Delete the whole file.

**Daemon parity test files (delete, not edit — see section 5 below):**
- `internal/daemon/remote_files_parity_test.go`
- `internal/daemon/remote_files_write_parity_test.go`

#### Files to edit (not delete)

**`main.go`** — Two edits:
1. Remove `case "tui":` / `err = cmdTUI(client)` from the `switch cmd` in `runCLI()` (lines 209-210). The `default:` case then handles `agenthub tui` with: `fmt.Fprintf(os.Stderr, "agenthub: unknown command %q\n...", cmd)` and `os.Exit(1)`, satisfying success criterion 1 ("exits with an error or is not recognized").
2. No import to remove from main.go itself — `internal/tui` is not imported there.

**`cmd_cli.go` — `usage()` function** — Remove the `  tui                                         Launch interactive terminal UI` line from the usage text (line 47). This is a docs-only edit; no import changes.

#### Files with comment-only TUI references (no code changes needed)

- `internal/daemon/client.go` line 516 — comment only: `// The FilesClient interface in internal/tui/files_client.go is NOT extended here`. Delete or update the comment; no import to change.
- `internal/attach/attach.go` lines 1-3 — package comment mentions TUI. Update comment to remove TUI reference; no import.

#### Frontend files with comment-only TUI references (optional cleanup)

- `frontend/src/lib/filesApi.ts` line 115 — comment: `internal/tui/remote_files_client.go (RMW-04 cross-surface parity contract)`. Update comment after deletion.
- `frontend/src/components/DaemonManagerPanel.tsx` line 389 — comment mentioning TUI. Update or delete comment.

> Note: `DaemonManagerPanel.tsx` is scheduled for deletion in Phase 138 (NAV-03 removes the Sessions page), so updating its comment in Phase 136 is optional.

---

### 2. Shared-Code Seams — Per-File Analysis

#### `internal/attach/attach.go` — SHARED, SURVIVES

`internal/attach` is imported by both `cmd_attach.go` (CLI) and `internal/tui/attach.go` (TUI). The package itself contains only generic I/O pump logic (`AttachSession`, `StdinPump`, `WsOutputPump`, `MakeClientResizeFrame`). It does NOT import `internal/tui`. Deleting the TUI has zero impact on `internal/attach` — it continues to serve the CLI attach path unchanged.

Action: None. Package survives as-is.

#### `internal/daemon/client.go` — COMMENT ONLY, NO CODE CHANGE

Line 516 is a comment referencing `internal/tui/files_client.go`. The file does not import `internal/tui`. The `DaemonClient` satisfies `FilesClient` via duck typing; that interface disappears with the TUI. The compile-time guard `var _ FilesClient = (*daemon.DaemonClient)(nil)` in `internal/tui/files_client.go` also disappears. This is fine — the interface served as the TUI's transport abstraction, not a daemon invariant.

Action: Update or delete the comment on line 516. No import or code change.

#### `internal/daemon/remote_files_parity_test.go` — DELETE OUTRIGHT

This file (`package daemon_test`) imports `internal/tui` to use `tui.NewRemoteFilesClientForTest` as a direct-client oracle proving the daemon proxy and TUI client observe byte-identical upstream responses. This is TUI parity evidence, not daemon correctness evidence.

The daemon proxy's behavior is fully covered independently by:
- `internal/daemon/remote_files_test.go` — `TestRemoteFiles_ListRoundTrip`, `TestRemoteFiles_NoCapRegistered_Returns404`, `TestRemoteFiles_Upstream403_PassesThrough`, `TestRemoteFiles_Upstream401_PassesThrough`, `TestRemoteFiles_StatRoundTrip`, `TestRemoteFiles_ReadRoundTrip`, `TestRemoteFiles_HeadOnRead`, `TestRemoteFiles_CallerCapStripped`, `TestRemoteFiles_PathQueryEncoding`, etc.
- `internal/daemon/relay_remote_files_test.go` — relay route coverage
- `internal/daemon/remote_caps_test.go` — cap deposit coverage

None of these files import `internal/tui`. Deleting `remote_files_parity_test.go` removes TUI parity proofs that are moot after TUI removal; it does not remove any surviving daemon correctness coverage.

Action: **Delete the entire file.**

#### `internal/daemon/remote_files_write_parity_test.go` — DELETE OUTRIGHT

Same analysis as above. This file imports `internal/tui` to use `tui.NewRemoteFilesClientForTest` as the direct-client oracle for write-parity proofs (`TestRemoteFilesWrite_CrossSurface`, `TestRemoteFilesWrite_CapLeakInvariant`). The daemon's write proxy is covered independently by `remote_files_test.go` functions `TestRemoteFilesWrite_ForwardsBody`, `TestRemoteFilesWrite_CallerCapStripped`, and `TestRemoteFilesWrite_GetPassesNilBody`.

Action: **Delete the entire file.**

---

### 3. Command Dispatch

The command dispatch is a hand-rolled switch in `main.go:runCLI()`. There is no cobra or urfave/cli framework. The relevant section:

```go
case "tui":
    err = cmdTUI(client)
```

This is the entirety of the registration. Remove these two lines. The `default:` arm of the same switch already handles unknown commands:

```go
default:
    fmt.Fprintf(os.Stderr, "agenthub: unknown command %q\nRun 'agenthub --help' for usage.\n", cmd)
    os.Exit(1)
```

After removing the `case "tui":` branch, `agenthub tui` will fall to `default:` and exit with code 1 and the message `agenthub: unknown command "tui"`. This satisfies success criterion 1.

**Recommendation: fully remove the case (no stub).** A stub that prints a "removed" message would be acceptable, but the `default:` arm's message is already accurate and leaves zero residue. Full removal is cleaner.

Also remove the `tui` line from `usage()` in `cmd_cli.go`.

---

### 4. go.mod / Dependency Cleanup

#### Direct dependencies removable after TUI deletion

All four charm.land direct dependencies are exclusively used inside `internal/tui`:

| Module | Used in | After deletion |
|--------|---------|----------------|
| `charm.land/bubbletea/v2 v2.0.6` | `internal/tui/*.go` (9 files) | Remove |
| `charm.land/bubbles/v2 v2.1.0` | `internal/tui/*.go` (7 files) | Remove |
| `charm.land/lipgloss/v2 v2.0.3` | `internal/tui/*.go` (6 files) | Remove |
| `github.com/charmbracelet/glamour v0.8.0` | `internal/tui/files.go` only | Remove |

#### `github.com/charmbracelet/x/ansi v0.11.7` — REMOVE

Exclusively used in `internal/tui/files.go`, `view.go`, and test files. Zero usage outside the TUI directory. [VERIFIED: grep search of live tree, no non-tui Go file imports this package]

#### `golang.org/x/term v0.43.0` — KEEP

Used by `cmd_tui.go` (to check TTY), but also by:
- `cmd_attach.go` — terminal raw mode
- `internal/statusbar/bar.go` — terminal width detection
- `internal/attach/attach_unix.go` — raw mode setup

`golang.org/x/term` MUST be kept. [VERIFIED: grep search]

#### Indirect dependencies likely dropped by `go mod tidy`

After removing the four direct charm.land + glamour + x/ansi deps, `go mod tidy` will prune any indirect deps that have no other requirer. The indirect dep list is large; do not hand-edit go.mod. Run `go mod tidy` and let it compute the correct closure.

Specific indirects that are charm-ecosystem-only and likely pruned:
- `github.com/charmbracelet/colorprofile`
- `github.com/charmbracelet/ultraviolet`
- `github.com/charmbracelet/x/term`
- `github.com/charmbracelet/x/termios`
- `github.com/charmbracelet/x/windows`
- `github.com/charmbracelet/lipgloss` (v0.12.1 indirect — legacy; check after tidy)
- `github.com/muesli/cancelreader`
- `github.com/muesli/reflow`
- `github.com/muesli/termenv`
- `github.com/atotto/clipboard`
- `github.com/aymanbagabas/go-osc52/v2`
- `github.com/alecthomas/chroma/v2` (used by glamour for syntax highlighting)
- `github.com/microcosm-cc/bluemonday` (used by glamour for HTML sanitization)
- `github.com/gorilla/css` (used by bluemonday)
- `github.com/aymerick/douceur` (used by bluemonday)
- `github.com/yuin/goldmark`, `github.com/yuin/goldmark-emoji` (used by glamour)
- `github.com/clipperhouse/displaywidth`, `github.com/clipperhouse/uax29/v2`
- `github.com/rivo/uniseg`
- `github.com/mattn/go-runewidth`
- `github.com/lucasb-eyer/go-colorful`
- `github.com/xo/terminfo`

> [ASSUMED] The exact set pruned by `go mod tidy` depends on whether any of these are pulled in transitively by other direct deps (e.g., Wails, goreleaser/nfpm). `go mod tidy` is authoritative; do not hand-edit.

**Mechanism: `go mod tidy`** — not manual go.mod editing. Run after all TUI files are deleted and `go build ./...` is green.

---

### 5. Test Impact (TEST-06)

#### Delete outright (TUI-only test coverage)

| File | Reason |
|------|--------|
| `internal/tui/attach_test.go` | TUI attach flow |
| `internal/tui/files_edit_test.go` | TUI `$EDITOR` shell-out |
| `internal/tui/files_integration_test.go` | TUI Files view integration |
| `internal/tui/files_ops_test.go` | TUI file operation key handlers |
| `internal/tui/files_test.go` | TUI Files view unit tests |
| `internal/tui/help_test.go` | TUI help overlay |
| `internal/tui/integration_test.go` | TUI model integration |
| `internal/tui/joincode_prompt_test.go` | TUI join-code modal |
| `internal/tui/modal_test.go` | TUI modals |
| `internal/tui/remote_files_client_test.go` | TUI `RemoteFilesClient` unit tests |
| `internal/tui/styles_test.go` | TUI lipgloss styles |
| `internal/tui/update_remote_test.go` | TUI remote session update handlers |
| `internal/tui/update_test.go` | TUI `Update()` message handler |
| `internal/tui/view_test.go` | TUI `View()` render |
| `internal/daemon/remote_files_parity_test.go` | TUI cross-surface parity (moot after TUI removal) |
| `internal/daemon/remote_files_write_parity_test.go` | TUI write cross-surface parity (moot) |

#### Edit (remove TUI assertions only) — NONE NEEDED

There are no daemon or CLI test files that mix TUI-specific assertions with surviving daemon/web behavior that cannot simply be deleted. The two daemon parity files test exclusively the TUI-vs-daemon-proxy parity contract; the daemon proxy's correctness is covered by other test files with no TUI import.

#### Non-TUI daemon tests that survive unchanged

- `internal/daemon/remote_files_test.go` — covers daemon proxy list/stat/read/write; no TUI import
- `internal/daemon/relay_remote_files_test.go` — covers relay mounting; no TUI import
- `internal/daemon/remote_caps_test.go` — covers cap deposit/registration; no TUI import

These are all in `package daemon` (not `package daemon_test`), so the `internal/tui` import cycle constraint from the deleted parity tests does not affect them.

---

### 6. Verification Approach

Exact commands to prove all four success criteria:

```bash
# SC1: agenthub tui exits with an error / is not recognized
./agenthub tui
# Expected: exits non-zero with "agenthub: unknown command \"tui\"" on stderr

# SC2: internal/tui package deleted, no import paths remaining
grep -r "internal/tui" . --include="*.go" --exclude-dir=".claude"
# Expected: zero matches

grep -r "charm\.land\|bubbletea\|charmbracelet/glamour\|charmbracelet/x/ansi" . --include="*.go" --exclude-dir=".claude"
# Expected: zero matches (after go mod tidy removes them from go.sum as well)

ls internal/tui 2>&1
# Expected: "No such file or directory"

# SC3: Go tests pass
go test -race -short ./...
# Expected: all pass; no "tui" in test output

cd frontend && pnpm test
# Expected: all pass (frontend has no TUI Go dependency)

# SC4: go build clean
go build ./...
# Expected: exit 0 with no output

# Full CI parity check
go test -race -short ./internal/...    # mirrors CI Windows path
go test -race -short ./...             # mirrors CI mac/linux path
```

**Additional verification — go.mod cleaned:**

```bash
go mod tidy
git diff go.mod go.sum
# Expected: charm.land/* lines removed from require block
```

---

### 7. Ordering / Landmines

**Recommended deletion order:**

**Step 1 — Remove the package and its direct consumer first:**

Delete `internal/tui/` (entire directory) AND `cmd_tui.go` simultaneously. These two deletions are co-dependent: `cmd_tui.go` imports `internal/tui`, so deleting either alone leaves a broken import. Delete both in the same commit or at minimum the same `git add` batch.

**Step 2 — Edit `main.go` and `cmd_cli.go`:**

Remove `case "tui": err = cmdTUI(client)` from `main.go` and the `tui` line from `usage()` in `cmd_cli.go`. These are straightforward edits; they do not have import dependencies.

After Step 2, `go build ./...` should be green. Verify before proceeding.

**Step 3 — Delete the daemon parity test files:**

Delete `internal/daemon/remote_files_parity_test.go` and `internal/daemon/remote_files_write_parity_test.go`. These import `internal/tui`, so the build will fail if they remain after Step 1. They can be deleted simultaneously with Step 1 if the executor prefers one big atomic change, or as a separate step after verifying the build from Steps 1-2.

After Step 3, `go test -race -short ./...` should be green. Verify.

**Step 4 — go mod tidy:**

Run `go mod tidy` to remove the charm.land direct deps and any orphaned indirect deps. This is safe only after `go build ./...` and `go test ./...` are both green (otherwise tidy may incorrectly add removed packages back as needed).

**Step 5 — Comment and README cleanup:**

Update comments in:
- `internal/daemon/client.go` line 516
- `internal/attach/attach.go` package comment
- `frontend/src/lib/filesApi.ts` line 115
- `README.md` (many references — see section 1 of this research; update in a single commit)

These are cosmetic only and do not affect the build. They can be deferred to a polish pass within the same phase.

**Landmines:**

1. **Delete `cmd_tui.go` BEFORE or simultaneously with `internal/tui/`**, never after. If `internal/tui/` is deleted first, the build breaks with `cannot find module providing package github.com/scottkw/agenthub/internal/tui`.

2. **Delete the daemon parity test files BEFORE running `go test ./...`**. They will fail to compile with `cannot find module providing package internal/tui` after Step 1.

3. **Run `go mod tidy` AFTER the build is green**, not before. Running it while the code still references charm.land packages will not remove them.

4. **`opencode-tui.json` in `internal/daemon/engine.go`** is NOT related to the AgentHub TUI. It is a config file for the OpenCode CLI's own TUI (its schema URL is `opencode.ai/tui.json`). Do NOT touch `engine.go` or its tests as part of this phase.

5. **`golang.org/x/term`** appears in `cmd_tui.go` but also in `cmd_attach.go`, `internal/statusbar/bar.go`, and `internal/attach/attach_unix.go`. Do not add it to the "remove from go.mod" list. `go mod tidy` handles this correctly.

---

## Architecture Patterns

### Project Structure After Phase 136

```
internal/
├── attach/          # Survives — shared CLI attach logic (no TUI import)
├── daemon/          # Survives — prune 2 parity test files
│   ├── remote_files_parity_test.go        # DELETE
│   └── remote_files_write_parity_test.go  # DELETE
├── tui/             # DELETE ENTIRE DIRECTORY
└── [all other packages unchanged]
```

### Command Dispatch Pattern

The project uses a hand-rolled switch in `main.go` (not cobra/urfave). No framework-level deregistration is needed — removing the `case "tui":` arm is the complete registration removal.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Removing unused go.mod entries | Manual go.mod editing | `go mod tidy` | Tidy correctly computes the transitive closure; hand-editing misses indirect orphans or incorrectly removes deps shared with other direct requires |
| Verifying no import paths remain | Shell grep scripted check | `go build ./...` | The compiler is authoritative; grep can miss alias imports or vendored copies |

---

## Common Pitfalls

### Pitfall 1: Deleting `internal/tui` before `cmd_tui.go`
**What goes wrong:** Build fails immediately with `cannot find package "github.com/scottkw/agenthub/internal/tui"` from the still-present `cmd_tui.go`.
**Why it happens:** `cmd_tui.go` is a top-level main package file that imports the deleted package.
**How to avoid:** Delete `internal/tui/` and `cmd_tui.go` in the same atomic operation (single `git rm -r` + `git rm`).
**Warning signs:** Any attempt to `go build` after deleting `internal/tui/` but before removing `cmd_tui.go`.

### Pitfall 2: Running `go test ./...` before deleting daemon parity test files
**What goes wrong:** `remote_files_parity_test.go` and `remote_files_write_parity_test.go` fail to compile with `cannot find package internal/tui`.
**Why it happens:** These test files are in `package daemon_test` (external test package) and import `internal/tui`.
**How to avoid:** Delete all three external importers of `internal/tui` before running any test or build command.
**Warning signs:** `FAIL github.com/scottkw/agenthub/internal/daemon [build failed]` in test output.

### Pitfall 3: Treating `opencode-tui.json` references as TUI-related
**What goes wrong:** Editor accidentally includes `internal/daemon/engine.go` in scope and modifies `ensureOpenCodeTUIConfig` or related test references.
**Why it happens:** The string "tui" appears in `engine.go` referencing OpenCode's own configuration file format.
**How to avoid:** Scope all deletion to `internal/tui/` directory and files explicitly importing that package. Do not grep-and-replace "tui" across the entire codebase.
**Warning signs:** Any changes to `engine.go`, `engine_test.go`, or `opencode_ansi_test.go`.

### Pitfall 4: Running `go mod tidy` too early
**What goes wrong:** `go mod tidy` may incorrectly re-add a dependency if any source file still references it, making the cleanup appear to have failed.
**Why it happens:** `tidy` works from the current source tree. If any TUI file remains, charm.land deps are still "needed."
**How to avoid:** Verify `go build ./...` is green (all TUI files deleted, all import references removed) before running `go mod tidy`.

### Pitfall 5: Believing `golang.org/x/term` is TUI-only
**What goes wrong:** Removing `golang.org/x/term` from go.mod breaks `cmd_attach.go`, `statusbar/bar.go`, and `attach_unix.go`.
**Why it happens:** `cmd_tui.go` imports `x/term`, creating a false impression it is TUI-only.
**How to avoid:** `go mod tidy` will keep it correctly. Do not manually remove it.

---

## Package Legitimacy Audit

This phase performs no new package installs. All packages involved (charm.land/*, charmbracelet/*) are being REMOVED, not added. No package legitimacy check required.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Indirect deps pruned by `go mod tidy` after TUI deletion (listed in section 4) will not be pulled in transitively by other direct deps | go.mod / dependency cleanup | Low — `go mod tidy` is authoritative; the list is informational, not prescriptive. No harm if some survive via other requirers. |

**All other claims in this research were verified directly against the live codebase by grep, file read, and import tracing.**

---

## Environment Availability

Step 2.6: SKIPPED — Phase 136 is a pure deletion/cleanup phase. No external tools, services, runtimes, or databases are required beyond the standard Go toolchain already present.

```bash
go version  # verified: go 1.26.3 (from go.mod)
```

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package |
| Config file | `go.mod` (Go version pinned) |
| Quick run command | `go test -race -short ./internal/daemon/... ./...` |
| Full suite command | `go test -race -short ./...` |
| Frontend suite | `cd frontend && pnpm test` |
| CI test command | `go test -race -short ./...` (mac/linux); `go test -race -short ./internal/...` (Windows) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NAV-01 | `agenthub tui` exits non-zero | smoke | `./agenthub tui; echo $?` | ✅ (post-build) |
| NAV-01 | No `internal/tui` import paths in codebase | build gate | `go build ./...` | ✅ |
| NAV-01 | No charm.land deps in go.mod | build gate | `go mod tidy && git diff go.mod` | ✅ |
| TEST-06 | All TUI test files deleted | build gate | `go test -race -short ./...` (must not compile tui) | ✅ |
| TEST-06 | Surviving test suite passes | automated | `go test -race -short ./...` | ✅ |
| TEST-06 | Frontend tests pass (no regression) | automated | `cd frontend && pnpm test` | ✅ |

### Sampling Rate
- **Per commit:** `go build ./...`
- **Phase gate:** `go test -race -short ./...` green + `cd frontend && pnpm test` green + `agenthub tui` exits non-zero

### Wave 0 Gaps
None — existing test infrastructure covers all phase requirements. No new test files are written in this phase (TEST-06 is deletion, not addition).

---

## Security Domain

Phase 136 is a deletion phase with no new attack surface. The security impact is uniformly positive:

- `RemoteFilesClient` (direct HTTPS + cap token over Tailscale) is removed. The cap-leak invariant enforced in the TUI (T-122-04-01, T-126-01) is no longer needed because the code that leaks it is gone. The daemon proxy path (which already enforces the same invariant independently) survives unchanged.
- No new authentication, session management, cryptography, or input validation paths are introduced.

ASVS categories: not applicable for a deletion phase.

---

## Sources

### Primary (HIGH confidence — verified against live codebase)
- Live codebase grep: all `import "github.com/scottkw/agenthub/internal/tui"` occurrences confirmed by direct file read
- `internal/tui/` directory listing — 32 files verified by `find`
- `go.mod` — direct dependency list read in full
- `internal/daemon/remote_files_test.go` — daemon proxy test coverage confirmed by function listing
- `main.go` — command dispatch switch read in full
- `cmd_cli.go` — `usage()` function read in full
- `internal/attach/attach.go` — import list verified (no `internal/tui` import)
- `internal/daemon/client.go` — import list verified (line 516 is comment only)

### Secondary (MEDIUM confidence)
- `go.sum` — charm.land entries confirmed present (will be removed by `go mod tidy`)
- README.md — TUI references enumerated for cleanup

---

## Metadata

**Confidence breakdown:**
- Deletion inventory: HIGH — every file verified by direct read or grep
- Shared-code seams: HIGH — import analysis done per-file
- go.mod cleanup: HIGH for direct deps; MEDIUM for indirect dep list (go mod tidy is authoritative)
- Test impact: HIGH — all daemon tests with TUI imports identified; daemon correctness coverage without TUI confirmed

**Research date:** 2026-06-19
**Valid until:** 2026-07-19 (stable codebase; valid until any significant refactor of `internal/daemon` or command dispatch)
