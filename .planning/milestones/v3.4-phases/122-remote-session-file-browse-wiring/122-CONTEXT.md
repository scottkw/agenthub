# Phase 122: Remote-Session File Browse Wiring - Context

**Gathered:** 2026-05-21
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Both the desktop GUI's React FileBrowserTab AND the TUI's Files view work transparently against remote tailnet sessions — frontend detects a remote session and points at the remote machine's existing webserver `/api/files/*` routes (Phase 119) using the session's existing web-share cap token. The remote machine's daemon does the actual file work; frontend is just a thin pointer.

**Requirements:** REMOTE-01, REMOTE-02, REMOTE-03, REMOTE-04, REMOTE-05

</domain>

<decisions>
## Implementation Decisions

### Locked (user mental-model sign-off)
- **Mental model:** "remote machine's daemon does the file work; frontend just points at the right surface". The remote daemon's `/api/files/*` routes (Phase 118) AND the remote webserver's cap-gated mount (Phase 119) already exist. No new "talk to remote daemon" flow needs building.
- **Cap reuse:** Frontend uses the session's existing web-share cap (not a fresh mint). Owner-issued tokens already have `files.read` via Phase 118's `filesReadEnabled()` gate.
- **Web-share preconditioned:** If a remote session is NOT web-shared, file browse is unavailable — desktop GUI shows "Enable web sharing to browse this session's files"; TUI shows equivalent message.
- **No new components:** FileBrowserTab already accepts `(baseURL, capToken?)` per Phase 120-04. TUI Files view will be extended to support a remote URL + cap config.

### Claude's Discretion
- Where in App.tsx to add the remote-vs-local branch (likely the same tab-gate code at lines 1117-1134)
- TUI: extend `DaemonClient` with a remote HTTPS mode OR introduce a small `RemoteFilesClient` wrapper
- How desktop GUI reads the session's web-share URL + cap (likely from session metadata already tracked for web-share UI)

</decisions>

<code_context>
## Existing Code Insights

- `FileBrowserTab.tsx` already accepts `baseURL` + optional `capToken` — Phase 120-04 designed for this
- `App.tsx` (post Phase 120-06) detects web mode via `window.location.pathname.startsWith('/app/')` for the web-share viewer surface; desktop session-list-to-tab path needs analogous remote-detection branch
- Session model in frontend includes some remote/web-share metadata — inspect existing usage
- TUI `internal/tui/update.go:381-409` currently toasts "File browser not available for remote sessions" on the `f` key for remote — that toast gets removed
- TUI uses `DaemonClient` from `internal/daemon/client.go` which talks Unix socket; remote variant needs HTTPS + cap
- Phase 119 already proved the wire: remote webserver `/api/files/*` accepts `?cap=<token>` and returns identical FileEntry JSON shape

</code_context>

<specifics>
## Specific Ideas

**Desktop GUI changes (small):**
1. In App.tsx file-browser tab opener: if session is remote AND web-shared → pass `baseURL = session.webShareURL, capToken = session.webShareCap`
2. If session is remote AND NOT web-shared → show "Enable web sharing to browse" instead of opening tab
3. Local session path stays unchanged

**TUI changes (slightly larger):**
1. Extend `internal/tui/files_cmds.go` `tea.Cmd` factories to accept a `(baseURL, capToken)` config or take a configured client
2. Add a small HTTPS+cap client wrapper that implements the same `ListFiles/StatFile/ReadFile/HeadFile` interface as `DaemonClient`
3. When `f` pressed on remote session: build the remote client, open Files view; remove the v3.4 toast
4. Web-share preconditioned: if remote session isn't shared, show TUI equivalent message in status line

**Testing:**
- Vitest: App.tsx remote-vs-local tab-opener branch
- Playwright: Add cell to files-browser.spec.ts that simulates desktop GUI opening a remote session (different baseURL passed to FileBrowserTab)
- Go: TUI integration test with a remote-mocked webserver
- Cross-surface parity test: same scenario hits all three surfaces and observes identical behavior

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
