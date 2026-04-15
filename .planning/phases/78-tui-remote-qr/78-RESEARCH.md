# Phase 78: TUI Remote & QR - Research

**Researched:** 2026-04-15
**Domain:** Go TUI (Bubble Tea v2) -- remote session integration + QR code overlay
**Confidence:** HIGH

## Summary

Phase 78 extends the Bubble Tea TUI (Phases 76-77) with two features: (1) remote tailnet peer sessions displayed as a grouped section within the existing session list, and (2) an ASCII QR code modal overlay triggered by the `q` key for any session with a web URL.

Both features have well-established data paths. Remote session fetching already exists in `cmd_cli.go` (`cmdList`) and `app.go` (`GetRemoteSessions`) -- the TUI must reuse the same `client.ListTailnetPeers()` + `fetchPeerSessions()` pattern, NOT create a new data path. QR code generation uses `github.com/skip2/go-qrcode` which is already in `go.mod` and already used by `cmdQR` in `cmd_cli.go`. No new dependencies are required.

The primary complexity is structural: the session list must evolve from a flat `[]daemon.SessionInfo` to a unified `[]listEntry` slice that interleaves local sessions, peer divider rows, and remote sessions. Navigation (j/k) must skip divider rows. The QR overlay follows the exact same modal pattern established by kill-confirm and new-session modals in Phase 77.

**Primary recommendation:** Implement in 2 plans -- Plan 1 covers the unified list model, remote session fetching, and divider row rendering. Plan 2 covers the QR overlay, `q` key reassignment, and help/hint bar updates.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TUI-07 | Remote sessions panel shows tailnet peer sessions with same grouping as GUI | Unified list model with `listEntry` entries, `fetchRemoteSessions` tea.Cmd reusing `client.ListTailnetPeers()` + `fetchPeerSessions()`, divider rows grouped by peer hostname -- exact grouping parity with CLI `cmdList` and GUI `RemoteSessionsPanel.tsx` |
| TUI-10 | ASCII QR code display for session web URL | QR overlay modal using `go-qrcode` `ToSmallString(false)`, triggered by `q` key, URL constructed from `webStatus.URL + "/sessions/" + session.ID` for local or `CLIRemoteSession.URL` field for remote |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Remote session discovery | API / Backend (daemon) | -- | `ListTailnetPeers()` is a daemon API call; peer probe is network I/O |
| Remote session fetching | Client (TUI tea.Cmd) | API / Backend (peer HTTPS) | TUI fires async tea.Cmd that probes peer HTTPS endpoints directly (same as CLI) |
| Session list rendering | Client (TUI View) | -- | Pure rendering logic in Bubble Tea View() |
| QR code generation | Client (TUI) | -- | `qrcode.New().ToSmallString()` is CPU-only, synchronous, <1ms |
| Session URL construction | Client (TUI) | API / Backend (web status) | Local URL from daemon web status; remote URL from peer probe response |
| Key dispatch / overlay lifecycle | Client (TUI Update) | -- | Bubble Tea model state machine |

## Standard Stack

### Core (already installed -- no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.0.5 | TUI framework (Model-View-Update) | Already used since Phase 76 [VERIFIED: go.mod] |
| `charm.land/lipgloss/v2` | v2.0.3 | Terminal styling and layout | Already used since Phase 76 [VERIFIED: go.mod] |
| `charm.land/bubbles/v2` | v2.1.0 | Key binding, textinput components | Already used since Phase 76 [VERIFIED: go.mod] |
| `github.com/skip2/go-qrcode` | v0.0.0-20200617195104-da1b6568686e | QR code generation with `ToSmallString` half-block rendering | Already used by `cmdQR` in `cmd_cli.go` [VERIFIED: go.mod] |

### Supporting (already used -- no additions)

| Library | Version | Purpose | When Used |
|---------|---------|---------|-----------|
| `github.com/scottkw/agenthub/internal/tailnet` | in-tree | `Peer` type, `DefaultProbePort` (7443), peer discovery | Remote session fetch |
| `github.com/scottkw/agenthub/internal/daemon` | in-tree | `DaemonClient`, `SessionInfo`, `WebServerStatusResponse` | All daemon communication |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `github.com/skip2/go-qrcode` | `github.com/mdp/qrterminal/v3` | qrterminal has explicit terminal rendering but go-qrcode is already in go.mod and `ToSmallString` produces equivalent output -- no reason to add another dependency |

**Installation:** No installation needed. All dependencies are already in `go.mod`.

## Architecture Patterns

### System Architecture Diagram

```
User Input (j/k/q/Enter/...)
        |
        v
  +-------------+    tea.KeyPressMsg     +-----------------+
  | Bubble Tea  | --------------------> | Update()        |
  | Runtime     |                       | handleKey()     |
  +-------------+                       | handleMainKey() |
        ^                               +-----------------+
        |                                   |          |
   tea.View                           tea.Cmd(s)   state mutation
        |                                   |
        v                                   v
  +-------------+               +---------------------+
  | View()      |               | fetchRemoteSessions |----> ListTailnetPeers()
  | renderFull()|               | (async goroutine)   |         (daemon API)
  +-------------+               +---------------------+
        |                               |            \
   renders from                         |             v
   m.unifiedList                        |      fetchPeerSessions()
   m.qrSession                          |         (per-peer HTTPS probe)
   m.qrContent                          |
        |                               v
        v                      remoteSessionsMsg
  +------------------+              |
  | Session List     |<-------------+
  | [local sessions] |
  | [-- divider --]  |         +-------------------+
  | [remote sessions]|         | QR Overlay        |
  +------------------+         | qrcode.New(url)   |
                               | .ToSmallString()  |
                               +-------------------+
```

### Recommended Project Structure

No new files beyond internal/tui/ package. All new code lives in existing files or new files within the same package:

```
internal/tui/
  model.go      # Extended: listEntry, listEntryKind, peerDivider, sessionRef, new model fields
  cmds.go       # Extended: fetchRemoteSessions tea.Cmd
  update.go     # Extended: remoteSessionsMsg handler, QR overlay key handling, unified list rebuild
  view.go       # Extended: renderDividerRow, renderRemoteSessionRow, renderQROverlay, header count
  keys.go       # Modified: q -> QR, Q -> quit, update Quit binding
  help.go       # Modified: help text updated per UI-SPEC
  modal.go      # Extended (or new qr.go): renderQROverlay
  *_test.go     # Extended: tests for all new behavior
cmd_remote.go   # Existing: CLIRemoteSession type reused by TUI (already in package main)
```

**Critical note:** `CLIRemoteSession` and `fetchPeerSessions` live in `package main` (cmd_remote.go, cmd_cli.go). The TUI package (`internal/tui`) cannot import `package main`. Two options:
1. **Move types to a shared internal package** (e.g., `internal/remote/`) -- cleanest but larger diff
2. **Duplicate the types and fetch logic in internal/tui** -- simpler, types are small (6-field struct)
3. **Pass a fetch function from cmd_tui.go** -- the TUI model accepts a callback that cmd_tui.go provides, wrapping the main-package functions

Option 3 is recommended: inject a `fetchRemoteFn func(ctx context.Context) []listRemoteGroup` callback into the Model at construction time. This avoids both import cycles and code duplication. The callback is set in `cmd_tui.go` which lives in `package main` and can call `fetchPeerSessions` directly.

### Pattern 1: Unified List Entry Model

**What:** Replace `[]daemon.SessionInfo` with `[]listEntry` for the rendered list
**When to use:** When the session list must contain heterogeneous row types (local, remote, divider)
**Example:**

```go
// Source: UI-SPEC.md Data Model Changes section
type listEntryKind int

const (
    entryLocal listEntryKind = iota
    entryRemote
    entryDivider
)

type listEntry struct {
    kind    listEntryKind
    session *daemon.SessionInfo        // non-nil when kind == entryLocal
    remote  *remoteSessionEntry        // non-nil when kind == entryRemote
    divider *peerDivider               // non-nil when kind == entryDivider
}

type remoteSessionEntry struct {
    ID       string
    Name     string
    CLIType  string
    Status   string
    Hostname string
    FQDN     string
    URL      string // pre-built: https://{fqdn}:{port}/sessions/{id}
}

type peerDivider struct {
    Hostname     string
    SessionCount int
}
```

[VERIFIED: internal/tui/model.go for current Model structure; UI-SPEC.md for target structure]

### Pattern 2: Navigation Skipping Dividers

**What:** j/k movement skips `entryDivider` rows
**When to use:** Any time `m.selected` is incremented/decremented
**Example:**

```go
func (m Model) moveDown() Model {
    for i := m.selected + 1; i < len(m.unifiedList); i++ {
        if m.unifiedList[i].kind != entryDivider {
            m.selected = i
            return m
        }
    }
    return m // no selectable entry below
}
```

[ASSUMED: exact implementation -- UI-SPEC says "j/k skips over divider rows"]

### Pattern 3: QR Overlay as Modal State

**What:** QR overlay uses the same modal pattern as kill-confirm and new-session
**When to use:** When `q` is pressed on a session with a URL
**Example:**

```go
// In handleMainKey:
case key.Matches(msg, m.keys.QR):
    entry := m.unifiedList[m.selected]
    url := m.sessionURL(entry)
    if url == "" {
        m.toast = "Web serving not enabled for this session"
        m.toastKind = toastInfo
        m.toastExp = time.Now().Add(2 * time.Second)
        return m, nil
    }
    q, err := qrcode.New(url, qrcode.Medium)
    if err != nil {
        m.toast = fmt.Sprintf("QR code generation failed: %s", err)
        m.toastKind = toastError
        m.toastExp = time.Now().Add(3 * time.Second)
        return m, nil
    }
    m.qrSession = &sessionRef{ID: id, Name: name, IsRemote: isRemote, URL: url}
    m.qrContent = q.ToSmallString(false)
    m.qrURL = url
    return m, nil
```

[VERIFIED: cmdQR in cmd_cli.go uses identical qrcode.New + ToSmallString pattern]

### Anti-Patterns to Avoid

- **Creating a new data path for remote sessions:** The CLI already has `client.ListTailnetPeers()` + `fetchPeerSessions()`. Reuse it. Do NOT add a new daemon endpoint.
- **Styling QR code output with lipgloss colors:** `ToSmallString(false)` uses ANSI reset codes internally. Applying foreground/background corrupts half-block rendering. Use `lipgloss.Place()` for centering only. [VERIFIED: UI-SPEC Anti-Pattern #2]
- **Blocking Update loop with HTTP calls:** All network I/O (remote session probe) must run in `tea.Cmd` goroutines, never in `Update()` directly. [VERIFIED: UI-SPEC Anti-Pattern #3]
- **Regenerating QR on every View() call:** Generate once in Update, store in `m.qrContent`. [VERIFIED: UI-SPEC Anti-Pattern #7]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| QR code generation | Custom QR encoder | `github.com/skip2/go-qrcode` + `ToSmallString(false)` | Already in go.mod, battle-tested, half-block rendering is compact |
| Tailnet peer discovery | Custom peer scanner | `client.ListTailnetPeers()` daemon API | Already implemented in Phase 50, handles DNS resolution |
| Peer session fetch | Custom HTTP client setup | `fetchPeerSessions()` pattern from cmd_remote.go | Already handles TLS, timeouts, JSON decode |
| Border title injection | Custom string manipulation | `injectBorderTitle()` in view.go | Already implemented in Phase 76, ANSI-safe |
| Modal overlay centering | Custom positioning | `lipgloss.Place()` | Already used by help, kill-confirm, new-session modals |

**Key insight:** Phase 78 composes existing pieces -- it does NOT introduce new infrastructure. The remote data path, QR library, and overlay rendering pattern all exist.

## Common Pitfalls

### Pitfall 1: Package Main Import Cycle
**What goes wrong:** Attempting to import `CLIRemoteSession` or `fetchPeerSessions` from `package main` into `internal/tui`.
**Why it happens:** These types/functions were defined in `cmd_remote.go` and `cmd_cli.go` (package main) for CLI use. Go forbids importing package main from other packages.
**How to avoid:** Inject a fetch callback into the TUI Model. `cmd_tui.go` (package main) provides the callback that wraps `client.ListTailnetPeers()` + `fetchPeerSessions()`. Define `remoteSessionEntry` as a TUI-local type mirroring CLIRemoteSession's fields.
**Warning signs:** Compilation error "import cycle not allowed" or "use of internal package not allowed".

### Pitfall 2: QR Code Color Corruption
**What goes wrong:** QR code becomes unreadable (garbled blocks, lost contrast).
**Why it happens:** Wrapping `ToSmallString` output in `lipgloss.NewStyle().Foreground(...)` overrides the ANSI reset codes embedded in the half-block characters.
**How to avoid:** Render the QR code string as-is inside the overlay. Apply `lipgloss.Place()` for centering. Apply styles only to the border, title, URL line, and hint -- never to the QR block itself.
**Warning signs:** QR code looks solid-colored or scanner cannot read it.

### Pitfall 3: Unified List Index Mismatch
**What goes wrong:** Operations (attach, kill, rename, QR) act on wrong session because `m.selected` indexes into `unifiedList` but operations expect the old flat session index.
**Why it happens:** The transition from `[]daemon.SessionInfo` to `[]listEntry` requires updating every place that uses `m.selected` to index into `m.sessions`.
**How to avoid:** Every operation must first resolve `m.unifiedList[m.selected]` to get the correct `listEntry`, then dispatch based on `entry.kind`. Kill/rename must check `kind != entryRemote`.
**Warning signs:** Pressing `d` on a divider row panics; pressing Enter on a remote session tries to attach locally.

### Pitfall 4: Remote Fetch Blocking the Tick Loop
**What goes wrong:** Remote probe (5-10s per peer) delays local session refresh.
**Why it happens:** If `fetchRemoteSessions` is batched synchronously with `fetchSessions`.
**How to avoid:** `fetchRemoteSessions` must be a separate `tea.Cmd` that fires independently. On `tickMsg`, batch it alongside `fetchSessions` and `fetchWebStatus` -- each runs in its own goroutine. The remote result arrives via `remoteSessionsMsg` without blocking local data.
**Warning signs:** Session list freezes for 5-10 seconds on each tick cycle.

### Pitfall 5: Selection Jump on Unified List Rebuild
**What goes wrong:** After remote data arrives and `unifiedList` is rebuilt, the cursor jumps to wrong row.
**Why it happens:** `m.selected` is an integer index into `unifiedList`. When remote data changes, divider rows shift indices.
**How to avoid:** When rebuilding `unifiedList`, remember the currently selected entry (by ID + kind), rebuild the list, then scan for the same entry and restore `m.selected`. Clamp to bounds if not found.
**Warning signs:** Cursor visibly jumps to a different session after every 2-second tick.

### Pitfall 6: `q` Key Conflict During Transition
**What goes wrong:** `q` still quits instead of opening QR overlay.
**Why it happens:** The Quit key binding in `keys.go` currently includes `"q"` alongside `"ctrl+c"`.
**How to avoid:** Phase 78 must update the Quit binding to `"Q", "ctrl+c"` (removing lowercase `q`), and add a new QR binding for `"q"`. The `handleKey` dispatch must check QR overlay state before main view processing.
**Warning signs:** Pressing `q` exits the TUI instead of showing QR.

## Code Examples

### Verified: QR Code Generation (from cmd_cli.go cmdQR)

```go
// Source: cmd_cli.go lines 318-334 [VERIFIED: codebase]
func cmdQR(client *daemon.DaemonClient, args []string, out io.Writer) error {
    resp, err := client.GetWebServerStatus()
    if err != nil || !resp.Running {
        return fmt.Errorf("web server not running")
    }
    url := fmt.Sprintf("%s/sessions/%s", resp.URL, args[0])
    q, err := qrcode.New(url, qrcode.Medium)
    if err != nil {
        return fmt.Errorf("agenthub qr: %w", err)
    }
    fmt.Fprint(out, q.ToSmallString(false))
    fmt.Fprintln(out, url)
    return nil
}
```

### Verified: Remote Session Fetch Pattern (from cmd_cli.go cmdList)

```go
// Source: cmd_cli.go lines 103-128 [VERIFIED: codebase]
peers, _ := client.ListTailnetPeers()
if len(peers) > 0 {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    groupMap := make(map[string][]CLIRemoteSession)
    for _, p := range peers {
        fqdn := strings.TrimSuffix(p.DNSName, ".")
        peerSessions, _ := fetchPeerSessions(ctx, fqdn, tailnet.DefaultProbePort)
        for i := range peerSessions {
            peerSessions[i].Hostname = p.Hostname
            peerSessions[i].FQDN = fqdn
        }
        if len(peerSessions) > 0 {
            groupMap[p.Hostname] = append(groupMap[p.Hostname], peerSessions...)
        }
    }
}
```

### Verified: Modal Overlay Pattern (from modal.go renderKillConfirmModal)

```go
// Source: internal/tui/modal.go lines 99-163 [VERIFIED: codebase]
bordered := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(m.styles.BorderNormal).
    Width(overlayWidth-2).
    Padding(1, 2).
    Background(m.styles.BgModal).
    Render(content)

title := lipgloss.NewStyle().
    Bold(true).
    Foreground(m.styles.BorderAccent).
    Render(" QR: session-name ")

bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)

return lipgloss.Place(m.width, m.height,
    lipgloss.Center, lipgloss.Center, bordered)
```

### Verified: Session URL Construction

For **local sessions**: `url = fmt.Sprintf("%s/sessions/%s", m.webStatus.URL, session.ID)` [VERIFIED: cmd_cli.go cmdQR line 327]

For **remote sessions**: URL is pre-built in `fetchRemoteSessions` as `fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, port, item.ID)` [VERIFIED: app.go line 569]

The TUI must replicate this: local sessions construct URL from `m.webStatus.URL` + session ID; remote sessions carry the URL in their `remoteSessionEntry.URL` field.

### QR Code Dimensions (Measured)

```
// Source: empirical measurement with go-qrcode [VERIFIED: ran qrtest.go]
URL length 45 chars -> QR: 41 cols x 21 rows (Version 4)
URL length 65 chars -> QR: 45 cols x 23 rows (Version 5)
URL length 82 chars -> QR: 45 cols x 23 rows (Version 5)

// Overlay total with border + padding + URL + hint:
// Width: max(qr_cols + 6, url_len + 6, 50) clamped to min(term_cols - 4, 80)
// Height: qr_rows + 6 (border=2 + pad=2 + url=1 + hint=1)
// Worst case: 45 + 6 = 51 wide, 23 + 6 = 29 tall -> fits 55x25 minimum
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Flat `[]SessionInfo` list | Unified `[]listEntry` with dividers | Phase 78 | Session list becomes heterogeneous -- all indexing must use listEntry |
| `q` = Quit | `q` = QR overlay, `Q` = Quit | Phase 78 | Help overlay, hint bar, and key dispatch all need updating |

**No deprecated patterns -- all existing Phase 76/77 patterns are current and reused as-is.**

## Data Flow: Remote Sessions

```
2-second tick
    |
    v
tea.Batch(fetchSessions, fetchWebStatus, fetchRemoteSessions, nextTick)
    |                                         |
    v                                         v
sessionsMsg (local)                  remoteSessionsMsg (remote groups)
    |                                         |
    +--------------------+--------------------+
                         |
                         v
               rebuildUnifiedList()
    [local session 1]
    [local session 2]
    [-- divider: hostname-A (2 sessions) --]
    [remote session from A]
    [remote session from A]
    [-- divider: hostname-B (1 session) --]
    [remote session from B]
                         |
                         v
               m.unifiedList = result
               restore m.selected by entry identity
```

## Data Flow: QR Overlay

```
User presses 'q' on selected session
    |
    v
Is m.unifiedList[m.selected] web-served?
    |           |
   NO          YES
    |           |
    v           v
Toast:        Compute URL
"Web serving  (local: webStatus.URL + /sessions/ + id)
not enabled"  (remote: entry.URL)
              |
              v
           qrcode.New(url, qrcode.Medium)
              |           |
           ERROR        SUCCESS
              |           |
              v           v
           Toast:      m.qrSession = ref
           "QR code    m.qrContent = q.ToSmallString(false)
           generation  m.qrURL = url
           failed"     (overlay rendered in View)
```

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Option 3 (inject fetch callback into Model) is the cleanest approach for the package-main import cycle | Architecture Patterns | Low -- alternatives exist (move types or duplicate) |
| A2 | Selection restoration by entry ID+kind is sufficient to prevent cursor jump on list rebuild | Pitfall 5 | Medium -- may need fallback to "clamp to nearest" |
| A3 | Remote fetch can run in parallel with local fetch in tea.Batch without contention on m.client | Pitfall 4 | Low -- DaemonClient uses HTTP with connection pooling, safe for concurrent use |

## Open Questions

1. **Remote attach from TUI**
   - What we know: UI-SPEC says Enter on a remote session attaches via `hostname:id` syntax. `cmdAttachRemote` in `cmd_attach.go` handles this flow.
   - What's unclear: The current TUI `attachCmd` only handles local sessions (dials `ws://127.0.0.1:{port}`). Remote attach needs WSS to the peer's FQDN.
   - Recommendation: Extend `attachCmd` to accept a remote flag and WSS URL. Or inject a remote-attach callback similar to the remote-fetch callback. This is a functional extension of the attach flow from Plan 77-02.

2. **Sorting order of peer groups**
   - What we know: CLI `cmdList` iterates a map (undefined order). GUI sorts by hostname implicitly via React key ordering.
   - What's unclear: Should TUI sort peer groups alphabetically by hostname?
   - Recommendation: Sort alphabetically. This is deterministic and matches user expectations.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified -- all libraries are in go.mod, no new CLI tools or services needed).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package |
| Config file | none -- Go test discovery is automatic |
| Quick run command | `go test ./internal/tui/... -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TUI-07a | Remote sessions appear in unified list | unit | `go test ./internal/tui/... -run TestUpdate_RemoteSessionsMsg -count=1 -v` | Wave 0 |
| TUI-07b | Divider rows are not selectable (j/k skip) | unit | `go test ./internal/tui/... -run TestUpdate_NavigationSkipsDividers -count=1 -v` | Wave 0 |
| TUI-07c | Kill/rename blocked on remote sessions (toast) | unit | `go test ./internal/tui/... -run TestUpdate_KillRemoteBlocked -count=1 -v` | Wave 0 |
| TUI-07d | Header count shows "N local, M remote" | unit | `go test ./internal/tui/... -run TestView_HeaderRemoteCount -count=1 -v` | Wave 0 |
| TUI-07e | Divider row rendering matches UI-SPEC format | unit | `go test ./internal/tui/... -run TestView_DividerRow -count=1 -v` | Wave 0 |
| TUI-10a | QR overlay opens on `q` press | unit | `go test ./internal/tui/... -run TestUpdate_QROpen -count=1 -v` | Wave 0 |
| TUI-10b | QR overlay closes on Esc | unit | `go test ./internal/tui/... -run TestUpdate_QRClose -count=1 -v` | Wave 0 |
| TUI-10c | QR overlay shows toast when no URL | unit | `go test ./internal/tui/... -run TestUpdate_QRNoURL -count=1 -v` | Wave 0 |
| TUI-10d | QR rendered content contains valid QR string | unit | `go test ./internal/tui/... -run TestView_QROverlayContent -count=1 -v` | Wave 0 |
| TUI-10e | Terminal too small shows toast | unit | `go test ./internal/tui/... -run TestUpdate_QRTerminalTooSmall -count=1 -v` | Wave 0 |
| TUI-10f | `q` no longer quits, `Q` and Ctrl+C quit | unit | `go test ./internal/tui/... -run TestUpdate_QuitKeyReassignment -count=1 -v` | Wave 0 |
| TUI-10g | Help overlay updated with QR entry | unit | `go test ./internal/tui/... -run TestHelp_QRBinding -count=1 -v` | Wave 0 |
| INTEGRATION | Remote session grouping matches CLI output | manual | Run `agenthub tui` and `agenthub list` on same machine with peers | N/A (requires tailnet) |

### Sampling Rate

- **Per task commit:** `go test ./internal/tui/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] All test functions listed above -- they do not exist yet (Wave 0 creates them alongside implementation)
- [ ] Test helpers for `remoteSessionsMsg` and unified list construction -- extend `testModel()` in `update_test.go`
- [ ] No framework install needed -- Go testing is built-in

*(Existing test infrastructure covers all TUI unit test needs -- 50 tests pass currently)*

## Files Likely Modified / Created

### Modified Files

| File | Changes |
|------|---------|
| `internal/tui/model.go` | Add `listEntry`, `listEntryKind`, `peerDivider`, `remoteSessionEntry`, `sessionRef` types. Add `remoteSessions`, `unifiedList`, `qrSession`, `qrContent`, `qrURL`, `fetchRemoteFn` fields to Model. Add `remoteSessionsMsg` message type. |
| `internal/tui/cmds.go` | Add `fetchRemoteSessions` tea.Cmd that calls `m.fetchRemoteFn`. |
| `internal/tui/update.go` | Handle `remoteSessionsMsg`. Add `rebuildUnifiedList()`. Update `handleKey` dispatch (QR overlay priority between new-session and help). Update `handleMainKey` for `q` (QR), `d`/`r` (block on remote). Add `handleQRKey`. Update navigation to skip dividers. |
| `internal/tui/view.go` | Update `renderSessionList` to iterate `unifiedList`. Add `renderDividerRow`. Add `renderRemoteSessionRow`. Update `renderHeader` for "N local, M remote" count. Update `renderWebStatus` and `renderHintBar` for new key labels. |
| `internal/tui/keys.go` | Change Quit binding from `"q", "ctrl+c"` to `"Q", "ctrl+c"`. Add QR binding for `"q"`. |
| `internal/tui/help.go` | Update help content: add `q QR code / URL` to Sessions group, change quit to `Q, Ctrl+C`. |
| `internal/tui/modal.go` | Add `renderQROverlay()` following kill-confirm pattern. |
| `internal/tui/tui.go` | Update `newModel` to accept and store `fetchRemoteFn`. Update `Init` to batch `fetchRemoteSessions`. |
| `cmd_tui.go` | Construct `fetchRemoteFn` callback wrapping `client.ListTailnetPeers()` + `fetchPeerSessions()`. Pass to `tui.Run`. |
| `internal/tui/update_test.go` | Add ~10 new tests for remote sessions, QR overlay, key reassignment, navigation skipping. |
| `internal/tui/view_test.go` | Add ~5 new tests for divider rendering, QR overlay content, header count, remote row rendering. |
| `internal/tui/help_test.go` | Update existing tests for new help content. |

### Potentially Created Files

| File | Purpose |
|------|---------|
| `internal/tui/qr.go` | QR overlay rendering (could go in modal.go instead -- planner's discretion) |

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A (TUI runs locally, auth via daemon socket) |
| V3 Session Management | no | N/A |
| V4 Access Control | yes | Remote sessions are read-only from TUI; kill/rename blocked with toast |
| V5 Input Validation | yes | URL constructed from daemon data, not user input; QR content is URL-only |
| V6 Cryptography | no | N/A (TLS handled by existing peer probe in cmd_remote.go) |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious peer returning crafted session data | Tampering | `fetchPeerSessions` already validates JSON structure; TUI truncates display strings |
| URL injection via remote session URL field | Tampering | URL is constructed server-side from FQDN + port + session ID -- not from user input |
| QR code containing non-URL content | Information Disclosure | QR content is always `webStatus.URL + /sessions/ + id` -- never user-controlled |

## Sources

### Primary (HIGH confidence)
- `internal/tui/*.go` -- all 16 TUI package files read and analyzed
- `cmd_cli.go` -- `cmdList`, `cmdQR`, `listRemoteGroup` type [VERIFIED: codebase]
- `cmd_remote.go` -- `CLIRemoteSession`, `fetchPeerSessions` [VERIFIED: codebase]
- `app.go` -- `GetRemoteSessions`, `RemotePeerSessions`, `RemoteSession` types [VERIFIED: codebase]
- `internal/tailnet/tailnet.go` -- `Peer` type, `DefaultProbePort` [VERIFIED: codebase]
- `internal/daemon/types.go` -- `SessionInfo`, `WebServerStatusResponse` [VERIFIED: codebase]
- `go.mod` -- all dependency versions [VERIFIED: go.mod]
- `78-UI-SPEC.md` -- all UI contract decisions [VERIFIED: read in full]
- QR code dimensions -- empirical measurement via `go run` [VERIFIED: ran test program]

### Secondary (MEDIUM confidence)
- `github.com/skip2/go-qrcode` source code in module cache -- `ToSmallString` implementation [VERIFIED: read source]
- `frontend/src/components/RemoteSessionsPanel.tsx` -- GUI grouping pattern [VERIFIED: codebase]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already in go.mod, all patterns already established
- Architecture: HIGH -- UI-SPEC is detailed and locked; existing patterns are clear
- Pitfalls: HIGH -- identified from codebase analysis (package import cycle, QR rendering, index mismatch)

**Research date:** 2026-04-15
**Valid until:** 2026-05-15 (stable -- all dependencies pinned, no fast-moving APIs)
