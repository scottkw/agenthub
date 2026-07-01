# Phase 167: Native Notifications - Research

**Researched:** 2026-07-01
**Domain:** Cross-platform native OS notifications, Go daemon/GUI process architecture, Wails settings plumbing
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Settings toggle placement (USER CORRECTION — overrides NTF-04 / success-criterion-4 wording)**
- The enable/disable toggle goes in the **Behavior** section of Settings
  (`<h3 id="settings-behavior">Behavior</h3>`, SettingsTab.tsx:443, the TRAY-01 section),
  **NOT** the "Session Behavior" section (`settings-session-behavior`, line 471).
- NTF-04 and ROADMAP success criterion #4 say "Session Behavior" — that wording is
  superseded. Requirement intent (a Settings toggle, default off) is unchanged; only the
  section changes to **Behavior**.
- Follow the existing SettingsTab toggle pattern already used in the Behavior section
  (e.g. TRAY-01 / start-minimized toggles).

**Default state**
- Notifications default **OFF** (NTF-04). User opts in — avoids surprise notifications on
  upgrade from v4.1 and first-run OS permission prompts. Consistent with industry norm (Warp).

**De-duplication**
- Fire exactly **once per `running → waiting` (non-waiting → waiting) transition** (NTF-02).
  A session held in `waiting` for 5 minutes triggers no additional notifications.
- `maybeNotifyWaiting` de-dup lives in `app.go` (per v4.2 architecture notes).

**Notification content**
- Text includes **session name + agent type** so the user knows which session needs
  attention (NTF-03).

**macOS attribution**
- beeep on macOS attributes to "Script Editor" — **accepted trade-off for v4.2**. The Title
  field uses "AgentHub" to aid identification. Revisit only if user feedback demands branded
  attribution.
- **RESEARCH FINDING that bears on this decision — see Open Question 1 below.** The
  codebase already has a native, properly-attributed macOS notification path
  (`notification_darwin.go`, Phase 85) that does NOT have the "Script Editor" problem.
  Reusing it for macOS (and only adding beeep for Windows/Linux) is strictly better than the
  locked decision's literal wording and introduces no regression. Flagged, not silently
  applied — see Open Questions.

**Library / platform wrappers**
- New Go dependency: `github.com/gen2brain/beeep v0.11.2` (cross-platform, no CGO).
- New files: `notification_windows.go`, `notification_linux.go` (beeep platform wrappers).
- Tray-hidden delivery is a hard requirement (NTF-01) — notification must fire even when the
  GUI window is minimized to the system tray.

**Setting field**
- `NotifyOnWaiting bool` added to daemon Settings. **CORRECTION**: the actual struct is
  `daemonSettings` in `internal/daemon/engine.go:111`, not `internal/daemon/types.go` (that
  file has no `Settings` struct at all — see Architecture Patterns).

### Claude's Discretion

- Exact notification body wording (must include session name + agent type per NTF-03).
- Whether to reuse the existing native macOS notification path vs. introducing beeep for
  macOS too (research recommends reuse — see Open Question 1).
- Internal de-dup data structure and where exactly in `app.go`'s polling loop the check runs.
- Unique notification identifier scheme (needed to fix a real bug in the existing macOS
  code — see Common Pitfalls).

### Deferred Ideas (OUT OF SCOPE)

- Click-to-focus / notification actions — out of scope for v4.2.
- Branded macOS attribution (replace "Script Editor") — revisit in a future milestone.
- Notification sound configuration — not in scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NTF-01 | Native OS notification on macOS/Windows/Linux when a session enters `waiting`, INCLUDING when the GUI window is hidden (tray-resident). | Architecture Patterns (trigger point = Go-side background ticker in `app.go`, independent of window visibility — see System Architecture Diagram); Standard Stack (beeep for Windows/Linux, existing native path for macOS). |
| NTF-02 | Fire once per transition into `waiting`, not repeatedly while it stays `waiting`. | Pattern 2 (edge-triggered de-dup in `maybeNotifyWaiting`); Common Pitfalls #1 (cold-start seeding); Code Examples. |
| NTF-03 | Notification identifies which session needs attention (session name + agent type). | `internal/pty/detect.go` `DisplayName` table; Code Examples (`displayNameForCLI` helper + body format). |
| NTF-04 | Toggle in Settings (default OFF, user opts in). | Architecture Patterns Pattern 3 (Settings plumbing end-to-end); Code Examples (SettingsTab.tsx Behavior-section mirror). |
</phase_requirements>

## Summary

The daemon already computes a heuristic `waiting` status per session via
`internal/status/detector.go`, and that detector is **already edge-triggered** — its
`onTransit` callback fires only when the classified status changes, not on every PTY frame.
However, the callback is wired only *inside the daemon process* (`engine.go:470`); the
separate GUI process (`app.go`, `App` struct) does not receive it directly. Since Phase 85
("App is a thin Wails-binding shell — all session state lives in the daemon process"), the
GUI polls the daemon over a Unix socket. The only pattern that already fires a background
task on a fixed interval **independent of window visibility** is `startTrayPoller` (5s
ticker started in `startup()`, alive until Wails shutdown, refreshing tray icon/tooltip via
`a.ListSessions()`). This is the correct home for the notification trigger: extend
`refreshTrayState()` to compare each session's `Status` against a per-session
previously-observed value, and fire a notification exactly on the non-waiting→waiting edge.
This satisfies NTF-01 (fires regardless of window visibility, since the poller process keeps
running while hidden in tray) and NTF-02 (edge detection, not level detection) without
touching the daemon.

The codebase already has a **working, previously-shipped notification primitive**:
`sendNotification(title, body string)`, implemented natively for macOS via
`UNUserNotificationCenter` (cgo, `notification_darwin.go` + `tray_objc_darwin.m`, Phase 85,
proven live for the "Quit GUI Only" flow) and as a no-op stub for all other platforms
(`notification_other.go`, `//go:build !darwin`). The locked architecture (STATE.md) adds
`github.com/gen2brain/beeep v0.11.2` and two new platform files
(`notification_windows.go`, `notification_linux.go`). The natural reading of "two new files,
not three" is: **keep the existing native darwin implementation unchanged, and replace the
no-op `!darwin` stub with real beeep-backed implementations split by GOOS.** This avoids
introducing beeep's macOS "Script Editor" attribution problem at all (see Open Question 1)
while still satisfying the letter of the locked dependency addition for Windows/Linux.

One genuine bug must be fixed to reuse the existing macOS path safely: its
`UNNotificationRequest` uses a **hardcoded identifier** (`"agenthub.quit-gui-only"`). If two
sessions transition to `waiting` in quick succession, the second `sendNotification` call
would replace/cancel the first under that identifier. The C function and its Go wrapper need
an identifier parameter (e.g., a `"agenthub.session-waiting.<sessionID>"` string) threaded
through.

Settings persistence follows an exact existing template
(`StartMinimized`/`GetStartMinimized`/`SetStartMinimized`) spanning
`internal/daemon/engine.go` (`daemonSettings` struct + engine field + getter/setter),
`internal/daemon/api.go` (REST route pair), `internal/daemon/client.go` (Go client method
pair), `app.go` (Wails-bound method pair), and `frontend/src/components/SettingsTab.tsx`
(toggle JSX + load/save hooks) — all confirmed present and grep-verified.

**Primary recommendation:** Add `NotifyOnWaiting bool` (plain bool, `omitempty`, matching the
`FilesWrite` precedent — no schema-version bump needed) to `daemonSettings`; extend
`refreshTrayState()` in `app.go` with an edge-triggered `maybeNotifyWaiting` check driven by
`a.ListSessions()`; reuse `sendNotification` (fixing its hardcoded identifier) for macOS and
add beeep-backed `notification_windows.go` / `notification_linux.go`; mirror the
`startMinimized` toggle exactly in the Behavior section of `SettingsTab.tsx`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Status detection (`waiting` classification) | Daemon (background process) | — | Already implemented in `internal/status/detector.go`; edge-triggered, PTY-output-driven, process-independent of GUI. |
| Transition trigger / de-dup | GUI process (Go, `app.go`) | — | `App` is the only process with beeep/CGO notification code; must poll the daemon and diff status itself since the daemon→GUI `onStatus` callback path was removed in Phase 85's process-separation refactor. |
| Notification dispatch (OS-level) | GUI process (Go, `app.go` + `notification_*.go`) | OS/native APIs (UNUserNotificationCenter, beeep→dbus/notify-send/WinRT toast) | GUI process is the one guaranteed to run continuously (tray-resident) regardless of window state; OS APIs are the actual delivery mechanism. |
| Settings toggle (UI) | Frontend Server / React (SettingsTab.tsx) | — | Standard Settings-tab toggle pattern; no server-side rendering involved (Wails webview is local). |
| Settings persistence | API / Backend (daemon `daemonSettings`) | — | Single source of truth on disk (`settings.json`), consistent with every other daemon setting (StartMinimized, AutoCloseSession, etc.). |
| Session name + agent-type resolution | Daemon (`SessionInfo.Name`/`.CLI`) → GUI (`app.go` display formatting) | — | Daemon already exposes `Name` and `CLI` in `SessionInfo`; GUI process is closest to the notification dispatch call and owns display-name mapping (mirrors `internal/pty/detect.go`'s `DisplayName` table, which is Go-side and already canonical). |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/gen2brain/beeep` | v0.11.2 (latest published; released 2025-12-11) [VERIFIED: Go module proxy] | Cross-platform desktop notifications for Windows + Linux | Pure-Go/no-CGO on Windows (WinRT COM via `go-toast`, PowerShell/win32 fallback) and Linux (D-Bus via `godbus/dbus`, `notify-send` fallback); actively maintained by `gen2brain` (established Go OSS author); this project already links `godbus/dbus/v5` indirectly (via `tailscale.com`), so no net-new system dependency class is introduced. |

### Supporting (already present in the codebase — reused, not new)
| Component | Location | Purpose | When to Use |
|-----------|----------|---------|-------------|
| `sendNotification(title, body string)` | `notification_darwin.go` (cgo, `UNUserNotificationCenter`) + `notification_other.go` (no-op stub) | Existing native macOS notification primitive, shipped in Phase 85 for "Quit GUI Only" | Reuse for macOS in this phase (recommended — see Open Question 1); requires an identifier-parameter fix (Common Pitfalls #2). |
| `internal/pty.knownCLIs` (`detect.go:25-31`) | Go | CLI-name → human `DisplayName` table (e.g. `"claude"` → `"Claude Code"`) | Source of truth for agent-type display text (NTF-03). Note: `DetectCLI`/`DetectCLIs` do a live `PATH` lookup — do NOT call them from the notification path; mirror the table as a small local lookup instead (see Code Examples). |
| `startTrayPoller` / `refreshTrayState` | `app.go:1317-1355` | 5s background ticker, alive for the app's whole lifetime (`ctx` cancelled only at Wails shutdown), already fetches `a.ListSessions()` every tick | Hook point for the notification trigger — do not add a second ticker. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reusing daemon's `status.Watch` `onTransit` callback directly | Have the daemon push status changes to the GUI (e.g. via a long-lived SSE/WS connection) | Rejected: the daemon↔GUI split was a deliberate Phase 85 architecture decision ("callbacks can't serialize over HTTP in out-of-process daemon" — PROJECT.md); adding a push channel is a much larger change than the existing 5s-poll pattern already used for tray state, health, and updates. Polling is also what `pollSessionStatus`/`refreshTrayState` already do project-wide. |
| beeep for macOS too (literal reading of STATE.md) | Keep native `UNUserNotificationCenter` path for macOS, beeep only for Windows/Linux | Recommended (see Open Question 1) — strictly better UX (real "AgentHub" attribution vs. beeep's osascript-fallback "Script Editor" attribution), reuses proven Phase-85 code, and matches "two new files" (`notification_windows.go`, `notification_linux.go`) exactly as STATE.md lists them (no `notification_darwin.go` rewrite mentioned). |
| A new field type `*bool` (nil = default) | Plain `bool` with `json:"notifyOnWaiting,omitempty"` | Plain `bool` matches the `FilesWrite` precedent exactly (Phase 124 / CAP-08): zero-value `false` IS the correct default-off behavior, so no defaults-merge/schema-version bump is needed. `*bool` is only needed when the default is `true` (e.g. `ShellWebShareWarningEnabled`), which is not the case here (NTF-04 requires default OFF). |

**Installation:**
```bash
go get github.com/gen2brain/beeep@v0.11.2
```

**Version verification:** `go list -m -versions github.com/gen2brain/beeep` was run against the
real Go module proxy (`proxy.golang.org`) during this research session and returned
`v0.10.0 v0.11.0 v0.11.1 v0.11.2` — confirming v0.11.2 is the latest published version, not
stale training-data knowledge. `go list -m -json github.com/gen2brain/beeep@v0.11.2` further
confirmed: published 2025-12-11, Go version requirement 1.21.5 (well under the project's
`go 1.26.3`), sourced from `github.com/gen2brain/beeep` tag `v0.11.2` [VERIFIED: Go module
proxy].

## Package Legitimacy Audit

> The automated `package-legitimacy check` seam only supports `npm|pypi|crates` ecosystems —
> Go modules are not covered. Verification below was performed manually against the
> authoritative Go module proxy and GitHub, per the same evidentiary standard.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/gen2brain/beeep` | Go module proxy (proxy.golang.org) | v0.11.2 published 2025-12-11; project has released since v0.10.0 (multi-year history) | N/A (Go modules have no download counter) | `github.com/gen2brain/beeep` — real, active, git-tagged (`v0.11.2` resolves to a real commit `ab78edd6...`) | OK | Approved — [VERIFIED: Go module proxy] |
| `git.sr.ht/~jackmordaunt/go-toast` | sourcehut (indirect, pulled in transitively by beeep on Windows) | Established sourcehut project used by beeep's Windows 10/11 toast path | N/A | `git.sr.ht/~jackmordaunt/go-toast` | OK (indirect, not directly imported by this project) | Approved — transitive dependency, standard for beeep's Windows path per source inspection |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────── Daemon process (headless, long-lived) ───────────────────────────────┐
│                                                                                                        │
│  PTY output bytes                                                                                     │
│       │                                                                                                │
│       ▼                                                                                                │
│  status.Detector.Feed()  (internal/status/detector.go)                                                 │
│       │  classify() → running/idle/waiting/errored, EDGE-TRIGGERED (only fires onTransit on change)   │
│       ▼                                                                                                │
│  engine.sessionStatuses[id] = s   (internal/daemon/engine.go:472, under e.statusMu)                    │
│       │                                                                                                │
│       ▼                                                                                                │
│  GET /sessions  (internal/daemon/api.go handleListSessions) ──── HTTP over Unix domain socket ────┐    │
└──────────────────────────────────────────────────────────────────────────────────────────────────┼────┘
                                                                                                       │
┌──────────────────────────────── GUI process (Wails app, tray-resident) ───────────────────────────┼────┐
│                                                                                                      ▼   │
│  startTrayPoller goroutine (app.go:1320)  ── 5s ticker, alive until Wails shutdown ──                    │
│  (runs REGARDLESS of window visibility — this is the tray-hidden guarantee for NTF-01)                   │
│       │                                                                                                  │
│       ▼                                                                                                  │
│  refreshTrayState()  (app.go:1339)                                                                       │
│       │  sessions := a.ListSessions()   ← already fetched every tick for tray icon/tooltip               │
│       ▼                                                                                                  │
│  maybeNotifyWaiting(sessions)   [NEW]                                                                     │
│       │  for each session: compare sessions[i].Status vs a.lastWaitingStatus[id]                          │
│       │  if prev != "waiting" && current == "waiting" && a.notifyOnWaiting.Load():                        │
│       │      body := fmt.Sprintf("%s (%s) is waiting for your input.", s.Name, displayNameForCLI(s.CLI))  │
│       │      a.sendNotificationFunc("agenthub.session-waiting."+s.ID, "AgentHub", body)                   │
│       ▼                                                                                                    │
│  sendNotification(id, title, body)                                                                         │
│       ├── darwin: notification_darwin.go → tray_objc_darwin.m UNUserNotificationCenter (native, cgo)       │
│       ├── windows: notification_windows.go → beeep.Notify (WinRT toast / PowerShell fallback)               │
│       └── linux:   notification_linux.go   → beeep.Notify (D-Bus / notify-send fallback)                    │
│                                                                                                              │
│  ── Settings toggle (independent flow) ──                                                                   │
│  SettingsTab.tsx Behavior section (#settings-behavior, line 443)                                             │
│       │  onChange → SetNotifyOnWaiting(next)  (Wails binding)                                                 │
│       ▼                                                                                                        │
│  App.SetNotifyOnWaiting(val bool)  [NEW, mirrors App.SetStartMinimized]                                        │
│       │  a.notifyOnWaiting.Store(val); a.client.SetNotifyOnWaiting(val)                                        │
│       ▼ (Unix socket, PATCH /settings/notify-on-waiting)                                                       │
│  engine.SetNotifyOnWaiting(val)  → daemonSettings.NotifyOnWaiting → settings.json (0600)                       │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
notification_darwin.go        # UNCHANGED except: sendNotification gains an id parameter (Common Pitfalls #2)
notification_windows.go       # NEW — beeep-backed sendNotification(id, title, body string)
notification_linux.go         # NEW — beeep-backed sendNotification(id, title, body string)
notification_other.go         # DELETE — replaced by the two files above (matrix is darwin/windows/linux only, see build.yml)
tray_objc_darwin.m            # EDIT — sendNotification C function takes an identifier param instead of the hardcoded "agenthub.quit-gui-only"
app.go                        # EDIT — App struct gains notifyOnWaiting atomic.Bool, lastWaitingStatus map[string]string, sendNotificationFunc field;
                               #        startup() loads initial value; refreshTrayState() calls maybeNotifyWaiting(); new bound methods GetNotifyOnWaiting/SetNotifyOnWaiting
internal/daemon/engine.go     # EDIT — daemonSettings.NotifyOnWaiting bool `json:"notifyOnWaiting,omitempty"`; e.notifyOnWaiting field; Get/SetNotifyOnWaiting methods (mirror StartMinimized exactly)
internal/daemon/api.go        # EDIT — GET/PATCH /settings/notify-on-waiting routes + handlers (mirror start-minimized exactly)
internal/daemon/client.go     # EDIT — DaemonClient.GetNotifyOnWaiting/SetNotifyOnWaiting (mirror exactly)
frontend/src/components/SettingsTab.tsx   # EDIT — new toggle in Behavior section (id="settings-behavior"), state hooks mirroring startMinimized
frontend/src/components/SettingsSearch.tsx # EDIT — add one SEARCH_INDEX entry, target: 'settings-behavior'
```

### Pattern 1: Edge-triggered status detection is ALREADY built — do not re-detect in the daemon
**What:** `internal/status/detector.go`'s `Detector.Feed()` only calls `onTransit` when
`next != d.current` (line 161). The daemon-side `status.Watch` callback wired in
`engine.go:470-477` therefore already fires exactly once per transition (including
non-waiting→waiting). The GUI-side de-dup work in this phase is a SEPARATE, ADDITIONAL edge
detection over the polled `SessionInfo.Status` string — it exists because the daemon's
edge-triggered callback is process-local and does not cross the Unix-socket boundary to the
GUI (Phase 85 removed that callback path: `app.go:269` "emits Wails events (replaces the
onStatus callback used in earlier phases)").
**When to use:** Confirm this while planning — do not attempt to add a new daemon→GUI push
channel; use the existing 5s poll.
**Example:**
```go
// Source: internal/status/detector.go:157-167 (existing, unmodified)
func (d *Detector) Feed(raw []byte) {
	stripped := StripANSI(raw)
	d.tail = appendTail(d.tail, stripped, maxTailBytes)
	next := d.classify()
	if next != d.current {
		d.current = next
		if d.onTransit != nil {
			d.onTransit(d.sessionID, next)
		}
	}
}
```

### Pattern 2: GUI-side edge detection over polled status (the actual NTF-02 de-dup)
**What:** `maybeNotifyWaiting` keeps a `map[string]string` of the last-observed `Status` per
session ID, updated once per 5s tick from `a.ListSessions()`. It notifies only when the
current tick's status is `"waiting"` AND the previous tick's status for that session ID was
anything else.
**When to use:** Called from `refreshTrayState()` right after `sessions = a.ListSessions()`
(app.go:1352) — no extra HTTP round trip, reuses the slice tray state already fetches every
tick.
**Example:** see Code Examples section below (full function).

### Pattern 3: Settings plumbing — exact existing template (StartMinimized)
**What:** Every daemon-persisted boolean setting in this codebase follows the identical
5-layer shape: `daemonSettings` struct field → engine field + `Get*`/`Set*` methods (which
call `saveSettingsToDisk()` under `e.mu.Lock()`) → two REST routes in `api.go` → two
`DaemonClient` methods in `client.go` → two Wails-bound methods on `App` in `app.go` →
React state + `useEffect` load + handler in `SettingsTab.tsx`.
**When to use:** Copy this shape exactly for `NotifyOnWaiting`; do not invent a new
persistence mechanism.
**Example (existing StartMinimized, to mirror verbatim):**
```go
// Source: internal/daemon/engine.go:1088-1101
func (e *SessionEngine) GetStartMinimized() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.startMinimized
}

func (e *SessionEngine) SetStartMinimized(val bool) {
	e.mu.Lock()
	e.startMinimized = val
	e.saveSettingsToDisk()
	e.mu.Unlock()
}
```
```go
// Source: internal/daemon/api.go:856-870
func (a *API) handleGetStartMinimized(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"startMinimized": a.engine.GetStartMinimized()})
}

func (a *API) handleSetStartMinimized(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartMinimized bool `json:"startMinimized"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	a.engine.SetStartMinimized(req.StartMinimized)
	w.WriteHeader(http.StatusNoContent)
}
```
```go
// Source: internal/daemon/client.go:151-164
func (c *DaemonClient) GetStartMinimized() (bool, error) {
	var resp map[string]bool
	if err := c.doJSON(http.MethodGet, "/settings/start-minimized", nil, &resp); err != nil {
		return false, err
	}
	return resp["startMinimized"], nil
}

func (c *DaemonClient) SetStartMinimized(val bool) error {
	return c.doJSON(http.MethodPatch, "/settings/start-minimized",
		map[string]bool{"startMinimized": val}, nil)
}
```
```tsx
// Source: frontend/src/components/SettingsTab.tsx:105-109, 185-189, 332-344, 443-468
const [startMinimized, setStartMinimized] = useState(false)
const [toggleLoaded, setToggleLoaded] = useState(false)
const [toggleSaving, setToggleSaving] = useState(false)
const [toggleError, setToggleError] = useState<string | null>(null)

useEffect(() => {
  GetStartMinimized().then(val => {
    setStartMinimized(val)
    setToggleLoaded(true)
  }).catch(() => setToggleLoaded(true))
}, [])

async function handleToggleMinimized() {
  const next = !startMinimized
  setToggleSaving(true)
  setToggleError(null)
  try {
    await SetStartMinimized(next)
    setStartMinimized(next)
  } catch (err) {
    setToggleError('Could not save preference — ' + (err instanceof Error ? err.message : String(err)))
  } finally {
    setToggleSaving(false)
  }
}
```

### Anti-Patterns to Avoid
- **Adding a second ticker for notifications:** `startTrayPoller` already polls every 5s and
  already fetches the full session list. A separate notification-specific ticker would be
  redundant, double the HTTP-over-socket traffic, and risk the two tickers racing on
  `a.lastWaitingStatus`.
- **Calling `pty.DetectCLI`/`pty.DetectCLIs` from the notification path:** these do a live
  `exec.LookPath` PATH scan — slow and semantically wrong (a session already running under a
  CLI whose binary later moved/is removed from PATH should still get a correct display name).
  Use a small static lookup mirroring `knownCLIs` instead (see Code Examples).
- **Re-using the hardcoded macOS notification identifier `"agenthub.quit-gui-only"` for
  session-waiting notifications:** guarantees notification replacement/loss when 2+ sessions
  become `waiting` close together. Must be per-session (see Common Pitfalls #2).
- **Querying `GetNotifyOnWaiting()` over the socket on every 5s tick:** unnecessary I/O.
  Cache the flag in an `atomic.Bool` on `App`, loaded once at `startup()` and updated
  whenever the user flips the Settings toggle (mirrors the existing `lastTrayQuartile
  atomic.Int32` pattern on the same struct).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Windows/Linux native notification delivery | Custom `syscall`/COM bindings for Windows toast, custom D-Bus client for Linux | `github.com/gen2brain/beeep` | Handles the WinRT COM API / PowerShell fallback / win32-Beep fallback matrix on Windows and the D-Bus / `notify-send` fallback matrix on Linux; hand-rolling either is a multi-week undertaking with deep platform-API and packaging (AUMID/shortcut) pitfalls. |
| macOS native notification delivery | New cgo/AppleScript code | Existing `notification_darwin.go` (`UNUserNotificationCenter`, Phase 85, live-verified) | Already built, already proven live for a shipped feature (Quit GUI Only, D-10/D-11). Do not introduce beeep's osascript path when a better-attributed native path already exists in the same repo. |
| Agent-type display names | A duplicate switch statement in Go | `internal/pty/detect.go`'s `knownCLIs` table (mirror as a small const map/switch local to `app.go`, since the live table does PATH lookups unsuitable for this use case) | Keeps agent display names consistent with what the frontend's `agentBadgeModifier` and the CLI-detection UI already show; avoids drift between "Claude Code" (detect.go) and some ad-hoc "claude" string in a notification body. |

**Key insight:** Every piece of this phase — status detection, background-independent
polling, native notification delivery, and settings persistence — already has a proven
building block in the codebase from a prior phase. The work is almost entirely "wire these
four existing patterns together," not "build something new." The only genuinely new logic is
the ~15-line edge-detection function (`maybeNotifyWaiting`) and the identifier fix to the
existing macOS notification code.

## Common Pitfalls

### Pitfall 1: Cold-start notification storm for sessions already `waiting` at GUI launch
**What goes wrong:** On the very first `refreshTrayState()` tick after the GUI process
starts (or reconnects to an already-running daemon), `a.lastWaitingStatus` is empty. Every
session that happens to already be in `waiting` state at that moment (e.g., left overnight,
or the daemon kept running headless after a previous GUI quit) will look like a
non-waiting→waiting transition on tick 1, firing a notification burst for stale state, not
new information.
**Why it happens:** The de-dup map has no prior baseline the first time it runs; treating
"unknown" as "not waiting" makes every already-waiting session look like a fresh transition.
**How to avoid:** On the very first tick after `notifyOnWaiting` becomes true and the map is
empty/uninitialized, populate `lastWaitingStatus` from the current session list WITHOUT
firing notifications (baseline capture), then notify on every subsequent tick's edges only.
This is a **recommended** behavior, not explicitly locked in CONTEXT.md — flagged in Open
Questions for planner/user confirmation, since it changes user-visible behavior at cold
start.
**Warning signs:** Multiple notifications fire simultaneously right after the app is
launched/reopened from tray, for sessions the user never touched during this session.

### Pitfall 2: Hardcoded macOS notification identifier collapses concurrent notifications
**What goes wrong:** `tray_objc_darwin.m`'s existing `sendNotification` C function creates a
`UNNotificationRequest` with a fixed identifier: `requestWithIdentifier:@"agenthub.quit-gui-only"`.
If reused as-is for session-waiting notifications, two sessions transitioning to `waiting`
within a short window will collide on this identifier — `UNUserNotificationCenter` treats a
second `addNotificationRequest` with the same identifier as *replacing* the pending/delivered
one, so the first notification can silently disappear or never be shown.
**Why it happens:** The identifier was fine when `sendNotification` had exactly one caller
(`QuitGUIOnly`, which only ever fires once per quit action) but is unsafe once it's reused for
a class of events that can fire multiple times per session.
**How to avoid:** Extend the C function signature to `void sendNotification(const char
*identifier, const char *title, const char *body)`, thread an identifier through
`notification_darwin.go`'s `sendNotification`, and use a per-session identifier
(`"agenthub.session-waiting." + sessionID`) for the new call sites while `QuitGUIOnly` keeps
using its own fixed identifier (`"agenthub.quit-gui-only"`) since it is a singleton
notification by design.
**Warning signs:** During manual/live testing, only the LAST of several rapidly-transitioning
sessions' notifications appears on macOS.

### Pitfall 3: `pollSessionStatus` is per-session and time-boxed — it does not cover this feature
**What goes wrong:** It is tempting to reuse `app.go`'s existing `pollSessionStatus`
goroutine (started per-`CreateSession` call, `app.go:279`) for notification detection, since
it already emits `"session:status"` events on change. This will NOT satisfy NTF-01/NTF-02:
(1) it runs for only 300s (5 minutes) after session creation, then stops — a session that
transitions to `waiting` after that window is silently missed; (2) it is scoped to a single
session and does nothing for sessions created before the GUI attached (e.g., daemon kept
running after a GUI restart); (3) it emits a Wails frontend event, which requires the
renderer/webview to be mounted — while the renderer keeps running when the window is merely
*hidden* (tray-minimized, per the existing `QuitGUIOnly`/`WindowHide` design), this is a
fragile assumption to build a hard OS-notification requirement on.
**Why it happens:** `pollSessionStatus`'s doc comment ("Poll session status for up to 60s")
is now stale — the actual `deadline` is `300 * time.Second` — making the scope of coverage
easy to misjudge by reading comments alone.
**How to avoid:** Use `startTrayPoller`/`refreshTrayState` instead (see Pattern 1/2) — it is
a pure Go background loop with no dependency on the webview or a specific session's recent
creation time.
**Warning signs:** A session that has been open for over 5 minutes never gets a notification
even with the toggle on.

### Pitfall 4: `daemonSettings.SchemaVersion` bump is unnecessary and would be a regression source
**What goes wrong:** Adding a bump to `CurrentSchemaVersion` (currently `4`,
`internal/daemon/plugin_settings.go:9`) "to be safe" when adding `NotifyOnWaiting` would
trigger the defaults-merge/re-save path (`needsUpgradeWrite`) on every existing user's next
daemon start, unnecessarily rewriting `settings.json` and risking unrelated migration bugs
for a field that needs none.
**Why it happens:** `SchemaVersion` bumps are used elsewhere in this codebase (e.g. the
`Plugins` defaults-merge, Pitfall #14 per the code comments) for fields that need a non-zero
default merged in on load. `NotifyOnWaiting`'s correct default (`false`) already equals Go's
zero value for `bool`, so a plain `omitempty` field round-trips correctly with NO merge logic
— exactly the `FilesWrite` precedent (`engine.go:105-110`, `118`, `223`, `251`).
**How to avoid:** Add the field as `NotifyOnWaiting bool \`json:"notifyOnWaiting,omitempty"\``
with no schema-version change and no defaults-merge entry, mirroring `FilesWrite` exactly.
**Warning signs:** `TestSettingsMigration_*`-style tests (if a new one is added) failing on
an unrelated schema-version assertion, or every existing user's `settings.json` being
rewritten on first v4.2 daemon start for no functional reason.

### Pitfall 5: Headless Linux (CI, minimal server installs) has no notification backend
**What goes wrong:** `beeep.Notify` on Linux requires either a running D-Bus session bus with
a notification daemon (`org.freedesktop.Notifications`) or the `notify-send` binary on PATH.
Neither is guaranteed on a headless CI runner (`ubuntu-latest`/`ubuntu-22.04` in
`build.yml`) or a minimal/server Linux install. Calling `beeep.Notify` there returns an error
(does not panic), but a naive integration test that asserts "notification delivered" would
fail/hang in CI.
**Why it happens:** beeep's Linux backend is display-session-dependent by design — desktop
notifications are inherently a desktop-session feature.
**How to avoid:** (1) `sendNotification`'s beeep-backed wrappers must log-and-swallow the
`error` return from `beeep.Notify` — never propagate it up as a user-facing failure, matching
the existing darwin `sendNotification`'s `void`-returning, best-effort contract. (2) Do not
write an automated test that asserts real OS notification delivery; gate that behavior behind
a manual UAT item (see Validation Architecture / TESTING.md M-41).
**Warning signs:** CI failures or flaky tests where none should exist if this logic is
accidentally asserted against real beeep calls instead of the injected `sendNotificationFunc`
seam.

## Code Examples

### `maybeNotifyWaiting` — the core NTF-02 edge-detection logic (new)
```go
// Source: new code for app.go, following the existing atomic-field pattern
// (lastTrayQuartile atomic.Int32, app.go:86) and function-injection pattern
// (saveFileDialogFunc, app.go:97; refreshTrayStateFunc, app.go:102) already
// used in this file for testability.

// displayNameForCLI mirrors internal/pty/detect.go's knownCLIs table without
// doing a live PATH lookup (a session's CLI may no longer be on PATH by the
// time it transitions to waiting; the display name must still resolve).
func displayNameForCLI(cli string) string {
	switch cli {
	case "claude":
		return "Claude Code"
	case "codex":
		return "OpenAI Codex"
	case "gemini":
		return "Gemini CLI"
	case "opencode":
		return "OpenCode"
	case "agy":
		return "Google Antigravity"
	default:
		return cli // shells and unknown CLIs: show the raw name
	}
}

// maybeNotifyWaiting fires a native notification for every session whose
// Status transitioned from non-"waiting" to "waiting" since the previous
// call. Must be called from a single goroutine (the tray-poller ticker) —
// a.lastWaitingStatus is not otherwise synchronized.
func (a *App) maybeNotifyWaiting(sessions []SessionInfo) {
	if !a.notifyOnWaiting.Load() {
		return
	}
	firstRun := a.lastWaitingStatus == nil
	if firstRun {
		a.lastWaitingStatus = make(map[string]string, len(sessions))
	}
	seen := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		seen[s.ID] = true
		prev, known := a.lastWaitingStatus[s.ID]
		a.lastWaitingStatus[s.ID] = s.Status
		if firstRun {
			continue // baseline capture only — see Common Pitfalls #1
		}
		if s.Status == string(status.StatusWaiting) && known && prev != string(status.StatusWaiting) {
			body := fmt.Sprintf("%s (%s) is waiting for your input.", s.Name, displayNameForCLI(s.CLI))
			a.sendNotificationFunc("agenthub.session-waiting."+s.ID, "AgentHub", body)
		}
	}
	for id := range a.lastWaitingStatus {
		if !seen[id] {
			delete(a.lastWaitingStatus, id) // prune sessions no longer present
		}
	}
}
```

### Wiring into `refreshTrayState` (edit)
```go
// Source: app.go:1339-1355, extended
func (a *App) refreshTrayState() {
	if !a.trayInit {
		return
	}
	if a.client == nil {
		a.updateTray(nil, false)
		return
	}
	connected := a.client.Health() == nil
	var sessions []SessionInfo
	if connected {
		sessions = a.ListSessions()
		a.maybeNotifyWaiting(sessions) // NEW
	}
	a.updateTray(sessions, connected)
}
```

### beeep-backed `notification_linux.go` (new)
```go
//go:build linux

package main

import (
	"log"

	"github.com/gen2brain/beeep"
)

// sendNotification sends a Linux desktop notification via D-Bus (falling
// back to notify-send). Best-effort: errors are logged, never surfaced —
// mirrors the darwin implementation's contract (Common Pitfalls #5).
func sendNotification(identifier, title, body string) {
	if err := beeep.Notify(title, body, nil); err != nil {
		log.Printf("notification: beeep.Notify failed: %v", err)
	}
}
```

### beeep-backed `notification_windows.go` (new)
```go
//go:build windows

package main

import (
	"log"

	"github.com/gen2brain/beeep"
)

func sendNotification(identifier, title, body string) {
	if err := beeep.Notify(title, body, nil); err != nil {
		log.Printf("notification: beeep.Notify failed: %v", err)
	}
}
```

### Fixed darwin `sendNotification` — identifier threaded through (edit)
```go
// Source: notification_darwin.go, extended with an identifier parameter
// (fixes Common Pitfalls #2). C signature and tray_objc_darwin.m must be
// updated in lockstep — see void sendNotification(const char *identifier,
// const char *title, const char *body) in the .m file.
func sendNotification(identifier, title, body string) {
	cid := C.CString(identifier)
	ctitle := C.CString(title)
	cbody := C.CString(body)
	C.sendNotification(cid, ctitle, cbody)
	C.free(unsafe.Pointer(cid))
	C.free(unsafe.Pointer(ctitle))
	C.free(unsafe.Pointer(cbody))
}
```

## State of the Art

| Old Approach (Phase 85, still live) | Current Approach (this phase) | When Changed | Impact |
|--------------------------------------|-------------------------------|---------------|--------|
| `sendNotification(title, body string)` — single fixed identifier, macOS-only, non-macOS is a no-op | `sendNotification(identifier, title, body string)` — per-call identifier, real delivery on all 3 platforms | Phase 167 | Existing `QuitGUIOnly` call site needs its call signature updated too (pass its own fixed identifier); Windows/Linux users get their first-ever native notification from this app. |
| GUI process has no notion of "was this session already waiting" | GUI process tracks per-session last-observed status across ticks | Phase 167 | New in-memory state (`lastWaitingStatus`) on `App` — not persisted, resets on GUI restart (acceptable: see Pitfall 1's baseline-capture recommendation). |

**Deprecated/outdated:**
- The doc-comment on `pollSessionStatus` ("Poll session status for up to 60s") is stale — the
  real deadline is 300s. Not in scope to fix here, but do not rely on the comment when
  reasoning about coverage window (Common Pitfalls #3).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Reusing the existing native `UNUserNotificationCenter` macOS path (instead of beeep for macOS) is the intended reading of STATE.md's "two new files" plan, not a deviation from it. | Summary, Standard Stack, Open Questions | If the user actually wants beeep uniformly across all 3 platforms (e.g., for delivery-mechanism consistency or lower cgo maintenance surface), the planner should route macOS through beeep too and accept the Script Editor attribution exactly as CONTEXT.md's locked decision states. Low risk either way — both are functional; this is a UX/attribution choice, not a correctness one. |
| A2 | Suppressing notifications on the very first `maybeNotifyWaiting` tick (baseline capture, Pitfall 1) is the desired cold-start behavior. | Common Pitfalls #1, Code Examples | If the user actually wants a notification for every already-`waiting` session immediately on GUI launch/reconnect (e.g., "catch me up on what's waiting"), the baseline-capture suppression would need to be removed. This is a genuine product-behavior choice not addressed in CONTEXT.md. |
| A3 | The notification body format `"{SessionName} ({AgentDisplayName}) is waiting for your input."` satisfies NTF-03's "session name + agent type" requirement. | Code Examples | CONTEXT.md leaves exact wording to Claude's discretion; low risk — any format including both pieces of information satisfies the requirement text. |
| A4 | `beeep.Notify`'s Windows path (`go-toast`, struct-field based) does not shell out to PowerShell with unescaped string interpolation of the session name (which is user-controlled via `RenameSession`). | Security Domain, Common Pitfalls | Based on a WebFetch read of beeep's `notify_windows.go` source (current master, not necessarily byte-identical to the v0.11.2 tag). If a future beeep version (or the pinned v0.11.2 specifically) does use unescaped string interpolation somewhere in its Windows fallback chain, a maliciously-named session could inject PowerShell — low likelihood given the struct-based `toast.Notification` API observed, but not verified against the exact pinned tag's source. |

## Open Questions (RESOLVED 2026-07-01 — see 167-CONTEXT.md dated decision blocks)

> **RESOLVED:** Q1 → **reuse the native macOS `UNUserNotificationCenter` path** (beeep for Win/Linux only); Q2 → **silent cold-start baseline capture** (do not notify already-`waiting` sessions at launch). Both answers are locked in `167-CONTEXT.md` and implemented by plans 167-02/167-03. A third decision resolved alongside these: accept beeep's default Windows/Linux attribution for v4.2 (branded AUMID/app_name polish deferred).

1. **Should macOS use beeep at all, or keep its existing native `UNUserNotificationCenter` path?**
   - What we know: The codebase already has a proven, live, correctly-macOS-attributed
     notification path (`notification_darwin.go`, Phase 85). STATE.md's architecture notes
     list exactly two NEW files (`notification_windows.go`, `notification_linux.go`) — not
     three — which is consistent with keeping `notification_darwin.go` unchanged (beyond the
     identifier fix). CONTEXT.md's locked decision text explicitly says "beeep on macOS
     attributes to 'Script Editor' — accepted trade-off," which reads as if beeep WAS
     intended for macOS too.
   - What's unclear: Whether the CONTEXT.md author knew about the existing native macOS
     implementation when writing that trade-off note, or was reasoning purely from beeep's
     generic cross-platform behavior.
   - Recommendation: Use the existing native macOS path (strictly better outcome, zero
     regression, satisfies the literal "two new files" list). Flag this choice explicitly to
     the user during `/gsd-plan-phase` or `/gsd-discuss-phase` follow-up so the deviation from
     the literal CONTEXT.md wording is an informed, confirmed choice rather than a silent
     override.

2. **Cold-start behavior: notify for sessions already `waiting` when the GUI (re)connects, or baseline-capture silently?**
   - What we know: NTF-02 only specifies "once per transition," which is ambiguous about
     transitions that happened before the GUI process started observing.
   - What's unclear: Whether "already waiting at launch" should notify (useful — "catch up")
     or stay silent (avoids a notification burst on every GUI restart for long-lived waiting
     sessions).
   - Recommendation: Default to silent baseline-capture (Pitfall 1) as the safer, less
     surprising choice; document as Claude's-discretion in the plan.

## Environment Availability

| Dependency | Required By | Available (this dev machine, macOS) | Version | Fallback |
|------------|--------------|----------------------|---------|----------|
| Go module proxy access | `go get github.com/gen2brain/beeep@v0.11.2` | ✓ | — | — |
| `UNUserNotificationCenter` framework | macOS notification delivery | ✓ (framework already linked in `tray.go`/`notification_darwin.go`) | macOS 10.14+ | — |
| D-Bus session bus / `notify-send` | Linux notification delivery (beeep) | N/A on this dev machine (macOS); **not guaranteed on CI runners** (`ubuntu-latest`, `ubuntu-22.04` in `build.yml`) | — | Graceful no-op with logged error (Pitfall 5) — no automated test may assert real delivery on Linux CI. |
| WinRT toast COM API / PowerShell | Windows notification delivery (beeep) | N/A (macOS dev machine); no Windows CI runner available for manual verification during this research session | — | Graceful no-op with logged error if unavailable (e.g., legacy Windows or locked-down toast policy); live verification is a manual UAT item (M-41). |

**Missing dependencies with no fallback:** none — every platform path has a documented
graceful-degradation (log-and-continue) fallback per the CONTEXT.md requirement ("Note
failure modes when the notification backend is absent and how to fail gracefully").

**Missing dependencies with fallback:** Linux/Windows live notification delivery cannot be
verified from this macOS research session or from headless CI — both require a manual, live,
on-screen UAT pass per platform (see Validation Architecture / TESTING.md M-41 below).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (daemon/app layer) + `vitest` (SettingsTab toggle) |
| Config file | none — project-root `go test`, `frontend/vitest.config.ts` (existing) |
| Quick run command | `go test -race -short ./... -run Notify` and `cd frontend && pnpm test -- SettingsTab` |
| Full suite command | `go test -race -short ./...` and `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NTF-01 | Notification fires cross-platform, tray-hidden-independent | manual (real OS notification cannot be asserted headlessly) | — | ❌ Wave 0 (manual UAT only) |
| NTF-02 | Fires exactly once per non-waiting→waiting transition; no repeat while held in `waiting` | unit | `go test -run TestMaybeNotifyWaiting ./...` (new, on `App`, using the injected `sendNotificationFunc` seam — no real OS call) | ❌ Wave 0 |
| NTF-02 | Cold-start baseline suppression (Pitfall 1) | unit | same test file, `TestMaybeNotifyWaiting_FirstTickNoNotify` | ❌ Wave 0 |
| NTF-03 | Notification body includes session name + agent display name | unit | same test file, `TestMaybeNotifyWaiting_BodyFormat`; plus `TestDisplayNameForCLI` table test | ❌ Wave 0 |
| NTF-04 | Toggle off (default) suppresses all notifications | unit | same test file, `TestMaybeNotifyWaiting_DisabledNoop` | ❌ Wave 0 |
| NTF-04 | Settings toggle persists across daemon restart, defaults false | unit | `internal/daemon/engine_notify_test.go` (new, mirrors `engine_shell_warn_test.go`'s pattern for `ShellWebShareWarningEnabled`) — default/persist/API-GET/API-PATCH/client-roundtrip | ❌ Wave 0 |
| NTF-04 | SettingsTab toggle renders in Behavior section, loads/saves | unit | `frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx` (new, mirrors `SettingsTab.shell-warn-toggle.test.tsx`) | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -race -short ./... -run Notify` and the targeted vitest file.
- **Per wave merge:** Full `go test -race -short ./...` + full `pnpm test`.
- **Phase gate:** Full suite green before `/gsd-verify-work`; manual UAT (M-41, three
  platforms) tracked separately per the existing TESTING.md convention for GUI/OS-level
  behavior that CI cannot assert.

### Wave 0 Gaps
- [ ] `app_test.go` — add `TestMaybeNotifyWaiting_*` table tests using the
      `sendNotificationFunc` injection seam (App-struct field, function-injection pattern —
      NEW seam needed, mirrors `saveFileDialogFunc`).
- [ ] `internal/daemon/engine_notify_test.go` — new file, mirrors
      `internal/daemon/engine_shell_warn_test.go` shape for `NotifyOnWaiting`.
- [ ] `internal/daemon/api_notify_test.go` (or extend `api_test.go`) — GET/PATCH
      `/settings/notify-on-waiting` handler tests, mirrors the `start-minimized` route tests.
- [ ] `frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx` — new file,
      mirrors `SettingsTab.shell-warn-toggle.test.tsx`.
- [ ] `tray_objc_darwin_test.go` / darwin build-tagged test (if one exists for the tray ObjC
      layer) — verify the identifier-parameterized C signature compiles; likely covered by
      the existing `-race -short ./...` darwin CI leg rather than a dedicated test.

*Gap for NTF-01 (actual cross-platform on-screen notification delivery, including
tray-hidden): cannot be automated — requires a real OS notification center. This is
explicitly acknowledged in CONTEXT.md ("expect a manual UAT (M-NN) item"). See TESTING.md
addition below.*

### TESTING.md Manual Checklist Addition (for the planner to add verbatim, per the Standing Convention)
Next available manual item ID is **M-41** (grepped: no `M-41`/`M-42`/`M-43` exist in
`TESTING.md` as of this research). Recommended category: new `### Category U — Native
Notifications (NTF)` under Section 5, with at minimum:
- **M-41** Cross-platform notification delivery on `→ waiting` transition, including
  tray-hidden (NTF-01/02/03): on each of macOS, Windows, and Linux, enable the toggle, hide
  the window to tray (`QuitGUIOnly`-style, not full quit), drive a session into `waiting`,
  and confirm exactly one OS-native notification appears identifying the session name +
  agent type — even though the window is hidden. Repeat with the toggle OFF and confirm no
  notification appears (NTF-04).
  - *Why not automatable:* real OS notification centers require a live desktop session;
    CI runners (`build.yml`) do not have one on any of the three platforms.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | no | Notification feature has no auth surface — daemon Unix-socket trust boundary is unchanged. |
| V3 Session Management | no | N/A. |
| V4 Access Control | no | Settings toggle is local-machine-only (Wails-bound method → local daemon socket); no new remote-accessible surface. |
| V5 Input Validation | yes | Session `Name` (user-controlled via `RenameSession`) flows into the notification body text on all 3 platforms. See Known Threat Patterns below. |
| V6 Cryptography | no | N/A — no new secrets/crypto surface. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| AppleScript/osascript injection via a maliciously-renamed session on macOS | Tampering | **Not applicable if the recommended approach (Open Question 1) is taken** — the native `UNUserNotificationCenter` path constructs an `NSString` directly (Objective-C), never shells out to `osascript`, so there is no string-interpolation-into-a-script-language risk. If beeep IS used for macOS instead (literal CONTEXT.md reading), beeep's macOS fallback (`osascript -e 'display notification %q with title %q'`) uses Go's `%q` (Go-string quoting, NOT AppleScript-string quoting) — this should be verified against the exact v0.11.2 source before shipping if that path is chosen, since Go's `%q` escaping does not guarantee AppleScript-safe quoting. |
| PowerShell/shell injection via session name on Windows | Tampering | Verified (MEDIUM confidence, [CITED: github.com/gen2brain/beeep/blob/master/notify_windows.go], not the exact pinned tag) that beeep's Windows 10/11 path uses `go-toast`'s `toast.Notification{Title: title, Body: message, ...}` struct fields (COM API), not string-interpolated PowerShell — no injection risk observed. |
| D-Bus / notify-send argument injection via session name on Linux | Tampering | beeep's D-Bus path passes the title/body as separate D-Bus method-call arguments (not concatenated into a command string); its `notify-send` fallback (if beeep uses `exec.Command` with an argument slice, not a shell string) is similarly safe. Not independently re-verified against the exact `notify_linux.go` source in this session — low risk, standard practice for this class of library, but worth a spot-check during implementation if time allows. |
| Notification-body information disclosure (session name may contain sensitive project/path info) | Information Disclosure | Accepted risk, consistent with the rest of the app's UX — session names are already visible in the Hub UI, tab bar, and tray tooltip; a notification surfacing the same name is not a new disclosure vector. Out of scope per CONTEXT.md ("Deferred Ideas" does not list this, and NTF-03 explicitly requires the name be shown). |

## Sources

### Primary (HIGH confidence)
- `internal/status/detector.go` (repo, read in full) — edge-triggered `onTransit` semantics.
- `internal/daemon/engine.go` (repo) — `CreateSession`, `daemonSettings`, `GetStartMinimized`/`SetStartMinimized`, `loadSettingsFromDisk`/`saveSettingsToDisk`.
- `internal/daemon/api.go`, `internal/daemon/client.go` — REST route + client method pairs for existing settings.
- `internal/daemon/types.go` (repo, read in full — confirmed NO `Settings` struct exists there; STATE.md's canonical-ref pointer is corrected in this document).
- `app.go` (repo, read in full relevant sections) — `App` struct, `startup`, `pollSessionStatus`, `refreshTrayState`, `startTrayPoller`, `QuitGUIOnly`/`sendNotification` call site.
- `notification_darwin.go`, `notification_other.go`, `tray_objc_darwin.m` (repo) — existing native notification implementation.
- `frontend/src/components/SettingsTab.tsx`, `SettingsJumpBar.tsx`, `SettingsSearch.tsx` (repo) — exact Behavior-section toggle pattern to mirror.
- `.planning/milestones/v3.0-phases/85-quit-confirmation-modal/85-RESEARCH.md` and `85-01-PLAN.md` (repo history) — origin and rationale of the existing macOS notification implementation.
- `go list -m -versions` / `go list -m -json github.com/gen2brain/beeep@v0.11.2` [VERIFIED: Go module proxy] — confirmed v0.11.2 is latest, published 2025-12-11, real tagged commit.
- `.github/workflows/build.yml` (repo) — confirmed build matrix is exactly darwin/universal, linux/amd64 ×2, windows/amd64 (no other GOOS targets).
- `TESTING.md` (repo, read Sections 1-6) — Suite Manifest, Traceability Map format, Manual Checklist format (M-37..M-40 precedent), Standing Convention.

### Secondary (MEDIUM confidence)
- [gen2brain/beeep GitHub repo + pkg.go.dev](https://github.com/gen2brain/beeep) [CITED] — `Notify(title, message string, icon any) error` signature, per-platform backend summary (Linux D-Bus→notify-send, macOS terminal-notifier→osascript, Windows COM→PowerShell/win32).
- [gen2brain/beeep notify_darwin.go](https://github.com/gen2brain/beeep/blob/master/notify_darwin.go) [CITED, master branch — not the exact v0.11.2 tag] — confirms macOS fallback chain (terminal-notifier → `osascript -e 'display notification ...'`), the mechanism behind the CONTEXT.md-documented "Script Editor" attribution.
- [gen2brain/beeep notify_windows.go](https://github.com/gen2brain/beeep/blob/master/notify_windows.go) [CITED, master branch] — confirms Windows 10/11 path uses `go-toast`'s struct-based `toast.Notification` API (COM), not string-interpolated PowerShell.

### Tertiary (LOW confidence)
- General knowledge of D-Bus `org.freedesktop.Notifications` and Windows toast-notification AUMID requirements for unpackaged apps — [ASSUMED], not independently verified against beeep's exact implementation in this session; flagged in Assumptions Log / Environment Availability as requiring manual UAT confirmation.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — beeep version/existence verified against the real Go module proxy; existing native macOS path fully read and confirmed live-shipped (Phase 85).
- Architecture: HIGH — every wiring point (daemon settings, REST routes, client methods, Wails bindings, SettingsTab toggle, tray poller) was located and read directly in the current codebase, not inferred.
- Pitfalls: HIGH for the identifier-collision bug and the `pollSessionStatus` scope trap (both directly observed in code); MEDIUM for the cold-start baseline-capture recommendation (a genuine product-behavior judgment call, flagged as Open Question 2).
- Security: MEDIUM — Windows COM-based safety confirmed via source read (not the pinned tag exactly); Linux D-Bus/notify-send safety asserted from general library-design knowledge, not independently re-verified line-by-line.

**Research date:** 2026-07-01
**Valid until:** 30 days (stable domain — Go stdlib + a pinned third-party library version; the only fast-moving risk is a future beeep release changing platform-fallback behavior, mitigated by the version pin).
