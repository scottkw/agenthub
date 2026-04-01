# Phase 39: Remote Session Indicators — Research

**Researched:** 2026-04-01
**Phase Goal:** Remote users (web browser and CLI attach) can see the session name, agent type, host machine name, and connection state without guessing what they are connected to

## Architecture Overview

Phase 39 has two independent workstreams that touch separate codebases:
1. **Web terminal status bar** — HTML/CSS/JS in `web/terminal.html` + Go webserver metadata endpoint
2. **CLI attach banner** — Go in `cmd_attach.go`

### Web Terminal Status Bar

**Current state:**
- `web/terminal.html` has a `#status` div (fixed position overlay, `display:none` by default) used only for disconnect messages
- `#terminal` div fills 100% width/height with xterm.js
- Session ID extracted from URL path `/sessions/{id}`
- WebSocket connects to relay for PTY I/O

**Required changes:**
1. Add a persistent status bar **above** the terminal as a flex sibling (not a fixed overlay)
2. Fetch session metadata via REST endpoint before/during terminal setup
3. Poll for connection state updates on a 3s interval (per STATE.md decision)

**Critical constraint — FitAddon regression prevention:**
- The `#terminal` div must remain flex:1 so FitAddon's `proposeDimensions()` calculates the correct row count after subtracting the status bar height
- The status bar must use `flex-shrink: 0` and a fixed height (matching the desktop `tab-status-bar` at 32px)
- Layout: `body { display: flex; flex-direction: column; }` → `#web-status-bar` (flex-shrink:0, 32px) + `#terminal` (flex:1)
- This matches the desktop pattern where `.terminal-wrapper` is a flex column with `.tab-status-bar` (flex-shrink:0) below `TerminalPanel` (flex:1)

**Metadata source — extending web session API:**
- Current `GET /api/sessions` returns `sessionListItem{id, name, cli_type, status}` — no hostname
- The `sessionResolver` callback returns `(name, cliType, status)` — no hostname
- Need to either:
  - (A) Add a `GET /api/sessions/{id}` endpoint to the webserver returning full metadata including hostname, OR
  - (B) Extend `sessionResolver` to return hostname as a 4th value and add `hostname` to `sessionListItem`

**Decision: Option B** — extend `sessionResolver` to 4 return values `(name, cliType, status, hostname)`. This is simpler than adding a new endpoint, and the hostname is already available in `SessionEngine.ListSessions()` → `SessionInfo.Hostname`. The webserver's `handleListSessions` and the new per-session terminal page metadata fetch both benefit.

**New endpoint for terminal page:**
- Add `GET /api/sessions/{id}/info` to webserver returning `{id, name, cli_type, status, hostname}` for a single session
- The terminal page fetches this once on load to populate the status bar
- The terminal page polls this every 3 seconds to update connection state
- If fetch fails → status bar shows "Disconnected" state

### CLI Attach Banner

**Current state:**
- `cmd_attach.go` `cmdAttach()` validates session exists via `client.ListSessions()` which returns `[]SessionInfo` with `{ID, CLI, Name, State, CreatedAt, Hostname}`
- All metadata needed for the banner is already available in the session info
- After validation, immediately enters raw mode and starts I/O pumps
- No banner or "Detached" message exists

**Required changes:**
1. After session lookup (line ~59), print banner to `os.Stderr` before entering raw mode
2. After `attachSession` returns, print "Detached." to `os.Stderr`
3. Banner format: `session-name | cli-type | hostname` + `Press Ctrl-\ to detach.`
4. Use `fmt.Fprintf(os.Stderr, ...)` — stderr because stdout is the PTY data stream

**Banner design:**
```
───────────────────────────────────
 claude-session | claude | macbook-pro.local
 Press Ctrl-\ to detach.
───────────────────────────────────
```

Stderr is correct because:
- stdout is wired to the PTY output stream (wsOutputPump writes to os.Stdout)
- stderr is free for out-of-band messages
- Same pattern as SSH connection banners

### Connection State Indicator (Web)

**Approach: REST polling at 3s intervals**
- Per STATE.md decision: "use REST polling (3s interval) not a new relay frame type"
- The terminal page already has a WebSocket for PTY I/O — but connection state should come from REST to decouple
- Poll `GET /api/sessions/{id}/info` every 3 seconds
- Display states: "Connected" (green dot), "Disconnected" (red dot), "Reconnecting..." (amber dot)
- WebSocket `onclose` immediately flips to "Disconnected" (no 3s delay for disconnect)
- REST poll confirms session is still alive on the server side

### Color Palette (matching desktop TokyoNight theme)

From existing `style.css`:
- Background: `#16161e` (status bar bg, matching desktop `.tab-status-bar`)
- Border: `#292e42`
- Text muted: `#565f89`
- Text: `#a9b1d6`
- Accent: `#7aa2f7` (blue)
- Green (connected): `#9ece6a`
- Red (disconnected): `#f7768e`
- Amber (reconnecting): `#e0af68`

## File Impact Analysis

| File | Change | Complexity |
|------|--------|------------|
| `web/terminal.html` | Add status bar HTML/CSS/JS, REST polling, flex layout | Medium |
| `internal/webserver/server.go` | Extend sessionResolver to 4 returns, add `/api/sessions/{id}/info` endpoint, add hostname to sessionListItem | Medium |
| `internal/daemon/api.go` | Update sessionResolver lambda to return hostname | Small |
| `cmd_attach.go` | Add banner print before raw mode, "Detached." after return | Small |
| `cmd_attach_test.go` | Add tests for banner and detached message | Small |
| `internal/webserver/server_test.go` | Update sessionResolver mock, test new endpoint | Small |

## Risk Assessment

1. **FitAddon regression (HIGH):** If the status bar doesn't use flex-shrink:0 or takes variable height, proposeDimensions() will miscalculate. Mitigation: fixed 32px height, flex-shrink:0, test with `window.resize` after status bar is visible.

2. **sessionResolver signature change (MEDIUM):** Changing from 3 to 4 return values breaks all callers. There are ~6 call sites (daemon api.go, app_test.go, cmd_cli_test.go, webserver server_test.go). All must be updated atomically.

3. **CORS/auth on new endpoint (LOW):** The webserver has no CORS restrictions (Tailscale network-level auth). New endpoint follows same pattern.

## Validation Architecture

### Automated Tests
- Go: `go test ./internal/webserver/... ./internal/daemon/...` — verify sessionResolver 4-arg, new endpoint returns hostname
- Go: `go test .` (root package) — verify CLI attach banner in stderr
- Frontend: N/A (web/terminal.html is vanilla JS, no React test framework)

### Manual Verification
- Open web terminal page → status bar visible with session name, agent, hostname
- Kill daemon → status bar shows "Disconnected" within 3 seconds
- Run `agenthub attach <id>` → banner printed to terminal
- Detach with Ctrl-\ → "Detached." printed

---
*Research completed: 2026-04-01*
