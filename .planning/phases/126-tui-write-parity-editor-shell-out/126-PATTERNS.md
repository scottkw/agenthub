# Phase 126: TUI Write Parity (`$EDITOR` Shell-Out) - Pattern Map

**Mapped:** 2026-06-14
**Files analyzed:** 7 modified (no new files required)
**Analogs found:** 7 / 7 (all in-tree — this is a pure composition phase)

> **Guardrail up front (TUIW-01 blocker):** The milestone ARCHITECTURE.md §4.1
> sketched the `FilesClient` write methods with **`error`-only** returns. That is
> WRONG. The live Phase 123 `*daemon.DaemonClient` methods return **value structs**
> (`files.FileWriteResponse` / `files.FileOpResponse`). The interface MUST match the
> existing implementer or `*daemon.DaemonClient` will silently fail to satisfy
> `FilesClient` (compile error: `wrong type for method DeleteFile`). The exact
> signatures are in the Shared Patterns section below — copy them verbatim from
> `internal/daemon/client.go:513-661`, NOT from the architecture doc.

---

## File Classification

| Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---------------|------|-----------|----------------|---------------|
| `internal/tui/files_client.go` | interface/contract | request-response | `*daemon.DaemonClient` write methods (`client.go:513-661`) | exact (interface must mirror impl) |
| `internal/tui/remote_files_client.go` | service (transport client) | request-response (HTTPS+cap) | its own read methods (`ListFiles`/`ReadFile`) + DaemonClient write methods | exact (mirror read shape, copy body construction from daemon) |
| `internal/tui/files_cmds.go` | command factory (`tea.Cmd`) | file-I/O + request-response (async) | `loadDirCmd`/`readFileCmd`/`headFileCmd` (`files_cmds.go:87-147`) | exact |
| `internal/tui/files.go` | controller (key dispatch) | event-driven (keypress) | `handleFilesKey` switch (`files.go:445-...`) | exact |
| `internal/tui/update.go` | controller (Update reducer) | event-driven (msg) | `attachDoneMsg` handler (`update.go:81`), `attachCmd`/`tea.Exec` (`update.go:366`), `handleKillConfirmKey` (`update.go:669`), `handleRenameKey` (`update.go:639`) | exact |
| `internal/tui/model.go` / `modal.go` | model state + view | UI state | `modalKillConfirm` iota (`model.go:15-23`), kill/rename state fields (`model.go:148-156`), `renderKillConfirmModal` (`modal.go:163`) | exact |
| `internal/tui/files_test.go` | test | static-grep gate | `TestFiles_NoSyncFSCalls` (`files_test.go:850`) | exact (extend in place) |

---

## Pattern Assignments

### `internal/tui/files_client.go` (interface, request-response)

**Analog:** the existing 4-method interface (`files_client.go:23-28`) + the live DaemonClient write signatures.

**Current interface to extend** (`files_client.go:23-28`):
```go
type FilesClient interface {
	ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error)
	StatFile(ctx context.Context, sessionID, relPath string) (files.FileEntry, error)
	ReadFile(ctx context.Context, sessionID, relPath string) (data []byte, mime string, err error)
	HeadFile(ctx context.Context, sessionID, relPath string) (size int64, mime string, mtime time.Time, err error)
}
```

**Action:** Add the 4 write methods (see Shared Patterns → "FilesClient 8-method contract" for the exact lines). Do NOT add `UploadFile` (descoped, TUIW-06). Add a compile-time guard mirroring the one already in `remote_files_client.go:40`:
```go
// remote_files_client.go:40 (existing — copy this shape for the daemon guard)
var _ FilesClient = (*RemoteFilesClient)(nil)
// ADD in files_client.go (NEW — cheapest TUIW-01 enforcement; needs a daemon import):
var _ FilesClient = (*daemon.DaemonClient)(nil)
```

---

### `internal/tui/remote_files_client.go` (transport client, request-response over HTTPS+cap)

**Analog:** its own read methods + `*daemon.DaemonClient` write methods.

The TUI talks DIRECTLY to the remote webserver (no daemon proxy — `remote_files_client.go:23-26`), so the Phase 124 `proxyRemoteFiles` body-forwarding fix is NOT on this path. Each new write method must construct its own request body, mirroring the daemon method, but point at `c.filesURL(op, sid, rel)`.

**Existing read-method shape to mirror** — `ReadFile` (`remote_files_client.go:164-183`):
```go
func (c *RemoteFilesClient) ReadFile(ctx context.Context, sid, rel string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("read", sid, rel), nil)
	if err != nil {
		return nil, "", fmt.Errorf("remote files read: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("remote files read: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("remote files read: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	...
}
```

**Body construction to copy from the daemon side** — note WriteFile uses `octet-stream`, RenameFile uses a JSON `renameRequest`, Delete/Mkdir use `nil` body:
- `WriteFile`: `http.MethodPut`, `bytes.NewReader(data)`, `Content-Type: application/octet-stream` → decode `files.FileWriteResponse` (daemon `client.go:513-533`).
- `DeleteFile`: `http.MethodDelete`, `nil` body → decode `files.FileOpResponse` (daemon `client.go:582-601`).
- `RenameFile`: `http.MethodPost`, JSON `{oldRel,newRel}`, `Content-Type: application/json` → `files.FileOpResponse` (daemon `client.go:603-637`). Note the local `renameRequest` struct at `client.go:603-607`.
- `MkdirFile`: `http.MethodPost`, `nil` body, path via `filesURL` → `files.FileOpResponse` (daemon `client.go:642-661`).

**CAP-LEAK invariant (T-122-04-01, SECURITY V7)** — interpolate ONLY `(statusCode, body)` into error strings, NEVER the full URL (which carries `cap=`). Every existing read method already does this; `redactCapFromURL` (`remote_files_client.go:106`) exists for any path that must embed a URL. Use `NewRemoteFilesClientForTest` (`remote_files_client.go:82`) for `httptest.NewTLSServer` round-trip tests.

---

### `internal/tui/files_cmds.go` (command factory, async file-I/O + request-response)

**Analog:** `loadDirCmd` (`files_cmds.go:87-104`) — the canonical gate-safe `tea.Cmd` shape.

**Pattern to clone** (nil-guard + context timeout + generation stamping + result msg):
```go
func loadDirCmd(client FilesClient, sid, relPath string, gen uint64) tea.Cmd {
	return func() tea.Msg {
		if isNilFilesClient(client) {                         // files_cmds.go:23 helper — handles typed-nil
			return filesListMsg{sessionID: sid, generation: gen, relPath: relPath, err: errNilClient}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		entries, truncated, err := client.ListFiles(ctx, sid, relPath)
		return filesListMsg{sessionID: sid, generation: gen, relPath: relPath, entries: entries, truncated: truncated, err: err}
	}
}
```

**Action — add:** `editFetchCmd` (ReadFile → write temp file), `editWriteBackCmd` (read temp → WriteFile → `defer os.Remove`), `deleteCmd`, `renameCmd`, `mkdirCmd`, plus new msg types (`filesEditReadyMsg`, `editorExitMsg`, `filesOpMsg`). Use `context.WithTimeout` per CLAUDE.md (ctx-aware funcs). The `editWriteBackCmd` reference implementation is in RESEARCH.md §"Code Examples" (lines 311-326) — all `os.CreateTemp`/`os.ReadFile`/`os.Remove` go INSIDE the `func() tea.Msg` closure (async by construction → gate-safe; see Shared Patterns → No-Sync Gate).

---

### `internal/tui/files.go` (controller, event-driven keypress)

**Analog:** the non-filter switch in `handleFilesKey` (`files.go:484-...`).

**Existing branch shape to mirror** (the backspace/nav case at `files.go:490-498`) — note the `generation++` discipline before dispatching a `tea.Cmd`:
```go
case s == "backspace" || s == "left":
	if m.files.cwd == "" || m.files.cwd == "." {
		return m, nil
	}
	parent := parentDir(m.files.cwd)
	m.files.loading = true
	m.files.generation++ // WR-03: supersede any in-flight load
	return m, loadDirCmd(m.files.client, m.files.sessionID, parent, m.files.generation)
```

**Action — add cases for `e`/`d`/`r`/`m`/`u`** in the non-filter switch. The `e` branch reference (resolveEditor guard + selection bounds + `joinDir(cwd, ansi.Strip(name))` + `generation++`) is in RESEARCH.md §"Code Examples" lines 289-305. Key helpers already present:
- `joinDir(base, name)` (`files.go:411-416`) — forward-slash sandbox-relative path join. SECURITY V5: send relative paths ONLY, never absolute or `../`.
- `m.files.filteredEntries()` (`files.go:421-439`) — visible entries (ansi-stripped names).
- `m.files.err = errors.New(...)` → renders in status line via `renderFilesStatusLine` (`files.go:283-288`). Use this for the no-editor error and upload-descope message (RESEARCH A4).

**No-editor error copy (verbatim, locked):** `` "`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)." ``
**Upload-descope copy (TUIW-06):** `"Use desktop or web to upload files."`

---

### `internal/tui/update.go` (Update reducer + suspend/resume + modal/inline handlers)

#### `$EDITOR` shell-out — `tea.Exec` / `attachCmd` (`update.go:366-369`)

The Attach branch is the proven suspend-resume template:
```go
cmd := &attachCmd{client: m.client, sessionID: s.ID}
return m, tea.Exec(cmd, func(err error) tea.Msg {
	return attachDoneMsg{err: err}
})
```
`attachCmd` (`attach.go:17-116`) hand-rolls raw mode ONLY because PTY attach needs byte-level I/O. **The editor does NOT** — use `tea.ExecProcess(exec.Command(editor, tmpPath), onExit)` instead (simpler; see RESEARCH §"Don't Hand-Roll"). The `onExit` callback returns an `editorExitMsg`.

#### Resume-side handler — `attachDoneMsg` (`update.go:81-88`)

This is the template for `editorExitMsg`: fire follow-up cmds after the suspended process returns, surface a toast on error.
```go
case attachDoneMsg:
	cmds := []tea.Cmd{fetchSessions(m.client), fetchWebStatus(m.client)}
	if msg.err != nil {
		m.toast = fmt.Sprintf("Attach error: %s", msg.err)
		m.toastKind = toastError
		m.toastExp = time.Now().Add(3 * time.Second)
	}
	return m, tea.Batch(cmds...)
```
**For `editorExitMsg`:** `tea.Batch(tea.ClearScreen, editWriteBackCmd(...), loadDirCmd(...))` ALWAYS (TUIW-04 — unconditional refresh regardless of `exitErr`), bumping `m.files.generation` before `loadDirCmd`. Reference impl in RESEARCH §"Code Examples" lines 331-342. Add `case editorExitMsg:` and `case filesOpMsg:` to the `Update` switch (existing files msg cases at `update.go:122-129`).

#### Delete-confirm — `handleKillConfirmKey` (`update.go:669-691`)

```go
func (m Model) handleKillConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "y":
		return m.executeKill()
	case s == "n", s == "esc":
		m.modal = modalNone
		m.killTarget = nil
		return m, nil
	case s == "enter":
		if m.killFocusYes { return m.executeKill() }
		m.modal = modalNone; m.killTarget = nil
		return m, nil
	case s == "left", s == "right", s == "h", s == "l", s == "tab":
		m.killFocusYes = !m.killFocusYes
		return m, nil
	}
	return m, nil
}
```
**Action:** clone as `handleFileDeleteConfirmKey`; the `y`/`enter`-on-Yes branch dispatches `deleteCmd` instead of `executeKill`.

#### Rename/Mkdir inline input — `handleRenameKey` (`update.go:639-667`)

```go
func (m Model) handleRenameKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "enter":
		name := strings.TrimSpace(m.editInput.Value())
		if name == "" { /* toast "Name cannot be empty"; return m, nil */ }
		m.editing = false
		if name == m.editOriginal { return m, nil }   // no-op guard
		return m, renameSession(m.client, m.editSessionID, name)
	case s == "esc":
		m.editing = false
		return m, nil
	default:
		var cmd tea.Cmd
		m.editInput, cmd = m.editInput.Update(msg)
		return m, cmd
	}
}
```
**Action:** clone for Files rename (`r`, prefill name, dispatch `renameCmd`) and mkdir (`m`, empty, dispatch `mkdirCmd`). RESEARCH recommends a DEDICATED `filesNameInput` field on `filesModel` (not reusing `editInput`) to avoid colliding with session-rename state.

#### Priority dispatch ladder — `handleKey` (`update.go:208-247`)

```go
// Priority 1: Inline rename    -> m.editing
// Priority 2: Kill confirm     -> m.modal == modalKillConfirm
// Priority 3: New session       Priority 3.5: Join-code prompt
// Priority 5: Help overlay
// Priority 5.5: Files tab       -> m.activeTabID() == tabFiles -> handleFilesKey
```
**Action:** add the file-delete-confirm and Files-inline-input checks ABOVE the Priority 5.5 `handleFilesKey` dispatch (so a stray `y`/Enter can't leak to tab-cycling), exactly as kill-confirm (Priority 2) sits above. Mirror `TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp`.

---

### `internal/tui/model.go` + `modal.go` (state + view)

**modalState iota** (`model.go:15-23`): add `modalFileDeleteConfirm`. **State fields** (`model.go:148-156` show `killTarget`/`killFocusYes`/`editing`/`editInput`/`editOriginal`): add a `fileDeleteTarget` (relPath + isDir) and a dedicated `filesNameInput textinput.Model` on `filesModel`.

**Delete-confirm render** — clone `renderKillConfirmModal` (`modal.go:163-...`):
```go
question := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgDanger).
	Render(fmt.Sprintf("Kill session %q?", name))
detail := lipgloss.NewStyle().Foreground(m.styles.FgMuted).
	Render("This will terminate the running process.")
```
**Action:** `renderFileDeleteConfirmModal` with copy `Delete "name"?` + `This cannot be undone.` (dir variant: `Delete directory "name" and all contents?`). COLORBLIND constraint (MEMORY + RESEARCH): the explicit "Delete"/"Cannot be undone" TEXT carries the danger signal — `FgDanger` is reinforcement only, exactly like `homeDirWriteWarning` (`files.go:332`) uses a `⚠`+"Warning:" glyph, not color alone.

---

### `internal/tui/files_test.go` (test, static-grep gate)

**The No-Sync gate** — `TestFiles_NoSyncFSCalls` (`files_test.go:850-878`):
```go
files := []string{"files.go", "files_cmds.go"}
re := regexp.MustCompile(`\bos\.(ReadDir|Open|OpenFile|Stat)\b`)
commentLine := regexp.MustCompile(`^\s*//`)
// ... fails on any non-comment match in the listed files
```
**What the regex forbids:** `os.ReadDir`, `os.Open`, `os.OpenFile`, `os.Stat` — and ONLY those. It does NOT match `os.CreateTemp`, `os.ReadFile`, `os.WriteFile`, `os.Remove`, `os.Create`, `os.Getenv`, `exec.LookPath`, `exec.Command`.
**How write commands stay gate-safe:** the editor temp-file I/O (`os.CreateTemp`/`os.ReadFile`/`os.Remove`) lives inside `tea.Cmd` closures in `files_cmds.go` — async, never in `Update`, and not matched by the current regex either way. `client.WriteFile`/`DeleteFile`/etc. are network calls (not `os.*`) so they never match — but they STILL must run inside `tea.Cmd`s to avoid blocking the loop.
**TUIW-07 action (per the chosen reading):** keep `files.go`/`files_cmds.go` pure; optionally broaden the regex to also forbid `os.Create|Remove|ReadFile|WriteFile` and confirm those verbs appear only inside `func() tea.Msg {…}` closures (RESEARCH Pitfall 4 / A2). Also add `TestFiles_Phase126_Requirements` traceability matrix mirroring `TestFiles_Phase121_Requirements` (`files_test.go:1086`) — the project's established per-phase convention.

---

## Shared Patterns

### FilesClient 8-method contract (TUIW-01) — COPY VERBATIM

**Source:** `internal/daemon/client.go:513-661` (the live, frozen Phase 123 signatures).
**Apply to:** `files_client.go` interface AND `remote_files_client.go` impl. Both `*daemon.DaemonClient` and `*RemoteFilesClient` must satisfy all 8.

```go
type FilesClient interface {
	// Read (existing — DO NOT CHANGE — files_client.go:24-27)
	ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error)
	StatFile(ctx context.Context, sessionID, relPath string) (files.FileEntry, error)
	ReadFile(ctx context.Context, sessionID, relPath string) (data []byte, mime string, err error)
	HeadFile(ctx context.Context, sessionID, relPath string) (size int64, mime string, mtime time.Time, err error)
	// Write (new — MUST match DaemonClient return types: response structs, NOT error-only)
	WriteFile(ctx context.Context, sessionID, relPath string, data []byte) (files.FileWriteResponse, error)
	DeleteFile(ctx context.Context, sessionID, relPath string) (files.FileOpResponse, error)
	RenameFile(ctx context.Context, sessionID, oldRel, newRel string) (files.FileOpResponse, error)
	MkdirFile(ctx context.Context, sessionID, relPath string) (files.FileOpResponse, error)
}
```
**Warning sign of getting this wrong:** `*daemon.DaemonClient does not implement FilesClient (wrong type for method DeleteFile)` ⇒ the interface used `error`-only returns (the stale ARCHITECTURE.md §4.1 sketch). The compile-time guard `var _ FilesClient = (*daemon.DaemonClient)(nil)` catches it at build time. **Do NOT add `UploadFile`** (descoped, TUIW-06).

### Editor resolution chain (TUIW-03)

**Source:** RESEARCH §"Pattern 2"; stdlib `os.Getenv` + `exec.LookPath`. New helper (lives in `files_cmds.go` or a new `files_edit.go` — both gate-safe, `exec.LookPath` is not in the regex).
```go
func resolveEditor() string {
	for _, cand := range []string{os.Getenv("EDITOR"), os.Getenv("VISUAL"), "nano", "vim", "vi"} {
		if cand == "" { continue }
		if p, err := exec.LookPath(cand); err == nil { return p }
	}
	return ""
}
```
Honor the locked order `$EDITOR → $VISUAL → nano → vim → vi` (CONTEXT) even though many tools check `$VISUAL` first.

### Generation discipline (WR-03)

**Source:** `files.go:497`, `files_cmds.go:87` — bump `m.files.generation++` BEFORE dispatching any `loadDirCmd`/refresh, stamp it on the result msg, and discard messages whose `generation < current` (`files.go:644`, `:683`, `:725`). Apply to every new write `tea.Cmd` and the post-edit refresh so a stale in-flight request cannot clobber the listing.

### Nil-client guard

**Source:** `isNilFilesClient` (`files_cmds.go:23-34`) + `errNilClient` (`files_cmds.go:16`). Every new write `tea.Cmd` must guard with `isNilFilesClient(client)` before dispatching (handles the typed-nil case `testModel()` injects).

### CAP-LEAK invariant (SECURITY V7, T-122-04-01)

**Source:** every `RemoteFilesClient` read method (`remote_files_client.go:120-212`) — error strings interpolate ONLY `(statusCode, body)`, never the URL. `redactCapFromURL` (`:106`) for any URL-embedding path. Apply to all 4 new remote write methods.

### Colorblind-safe danger signal (MEMORY + CLAUDE.md)

**Source:** `homeDirWriteWarning` (`files.go:332`) — `⚠ Warning:` glyph+text carries meaning; color is reinforcement only. Apply to the delete-confirm modal: explicit "Delete"/"Cannot be undone" text, not red alone.

---

## No Analog Found

None. Every mechanism this phase needs already exists in-tree (interface extension, transport client, `tea.Cmd` factory, key dispatch, suspend/resume via `tea.Exec`, confirm modal, inline input, static-grep gate). This is a pure composition phase — the risk is in wiring (priority dispatch, generation bumping, unconditional refresh, no-sync gate placement, the interface return-type match), not in any novel component.

---

## Metadata

**Analog search scope:** `internal/tui/` (files_client, remote_files_client, files, files_cmds, update, model, modal, attach, files_test), `internal/daemon/client.go` (write method signatures).
**Files scanned:** 9 source files + 1 test file (all VERIFIED against live source at the cited line numbers).
**Pattern extraction date:** 2026-06-14
