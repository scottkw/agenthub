# Phase 167: Native Notifications - Context

**Gathered:** 2026-07-01
**Status:** Ready for planning
**Source:** Locked decisions carried from v4.2 STATE.md + one user correction (2026-07-01)

<domain>
## Phase Boundary

Users who opt in receive a single native OS notification the moment a session transitions
to awaiting-input (`waiting`) state, on macOS, Windows, and Linux — including when the app
window is hidden in the system tray. Requirements NTF-01..NTF-04.

Out of scope: notification actions/click-to-focus, notification sounds config, per-session
notification opt-in, and any notification trigger other than the `→ waiting` transition.
</domain>

<decisions>
## Implementation Decisions (LOCKED)

### Settings toggle placement (USER CORRECTION — overrides NTF-04 / success-criterion-4 wording)
- The enable/disable toggle goes in the **Behavior** section of Settings
  (`<h3 id="settings-behavior">Behavior</h3>`, SettingsTab.tsx:443, the TRAY-01 section),
  **NOT** the "Session Behavior" section (`settings-session-behavior`, line 471).
- NTF-04 and ROADMAP success criterion #4 say "Session Behavior" — that wording is
  superseded. Requirement intent (a Settings toggle, default off) is unchanged; only the
  section changes to **Behavior**.
- Follow the existing SettingsTab toggle pattern already used in the Behavior section
  (e.g. TRAY-01 / start-minimized toggles).

### Default state
- Notifications default **OFF** (NTF-04). User opts in — avoids surprise notifications on
  upgrade from v4.1 and first-run OS permission prompts. Consistent with industry norm (Warp).

### De-duplication
- Fire exactly **once per `running → waiting` (non-waiting → waiting) transition** (NTF-02).
  A session held in `waiting` for 5 minutes triggers no additional notifications.
- `maybeNotifyWaiting` de-dup lives in `app.go` (per v4.2 architecture notes).

### Notification content
- Text includes **session name + agent type** so the user knows which session needs
  attention (NTF-03).

### macOS attribution (RESOLVED 2026-07-01 — SUPERSEDES the earlier "beeep/Script Editor" decision)
- **Reuse the existing native macOS path**, do NOT use beeep on macOS. AgentHub already ships
  `notification_darwin.go` (Phase 85, `UNUserNotificationCenter`, verified live for "Quit GUI
  Only") which gives real **"AgentHub"** name + icon attribution. beeep's macOS osascript
  fallback ("Script Editor") is explicitly REJECTED.
- **Required fix to reuse it:** the existing `sendNotification` hardcodes its macOS notification
  identifier (`"agenthub.quit-gui-only"`). Thread a **per-session identifier** through the C
  function + Go wrapper so concurrent session-waiting notifications don't silently replace each
  other. Update the existing `QuitGUIOnly` call site to pass its own fixed identifier.
- Signature evolves: `sendNotification(title, body)` → `sendNotification(identifier, title, body)`.

### Windows/Linux attribution (RESOLVED 2026-07-01)
- **Accept beeep's default attribution** on Windows/Linux for v4.2. The notification fires
  reliably (NTF-01 met); the displayed app-name label may be generic (Windows toasts are
  attributed by AUMID; Linux is DE-dependent). Branded Windows AUMID registration / Linux
  `app_name` polish is **deferred to a future milestone** — do NOT add it in this phase.

### Library / platform wrappers
- New Go dependency: `github.com/gen2brain/beeep v0.11.2` (no CGO; Windows = WinRT toast via
  go-toast w/ PowerShell fallback, Linux = D-Bus w/ notify-send fallback). VERIFIED latest via
  Go module proxy (published 2025-12-11).
- New files: `notification_windows.go`, `notification_linux.go` (beeep wrappers, GOOS-split).
  `notification_other.go` is DELETED (matrix is darwin/windows/linux only, per build.yml).
  macOS keeps its existing `notification_darwin.go` (no rewrite).
- Both new wrappers MUST **log-and-swallow** the `beeep.Notify` error return — never surface a
  user-facing failure. Headless Linux (CI, minimal server installs) has no notification backend
  and will error; that must be a no-op, not a crash (Pitfall 5 in RESEARCH.md).
- Tray-hidden delivery is a hard requirement (NTF-01) — satisfied because the trigger lives in
  the GUI's always-on tray poller (see trigger point below), independent of window visibility.

### Cold-start baseline (RESOLVED 2026-07-01)
- On the FIRST poll tick after GUI launch, **silently baseline-capture** current session
  statuses — do NOT fire notifications for sessions already `waiting` at launch. Only fire for
  non-waiting → `waiting` transitions that occur AFTER the baseline is established.

### Setting field (CORRECTED per RESEARCH.md — there is no `Settings` struct in types.go)
- Add `NotifyOnWaiting bool` to the daemon settings struct `daemonSettings`
  (`internal/daemon/engine.go:111`). Mirror the existing `StartMinimized` end-to-end pattern
  exactly: `GetStartMinimized`/`SetStartMinimized` spanning engine.go → api.go → client.go →
  app.go → SettingsTab.tsx.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Trigger point (CORRECTED per RESEARCH.md — daemon detector does NOT cross the socket)
- The daemon's `status.Detector` (`internal/status/detector.go:157-167`) is edge-triggered but
  process-local — Phase 85 removed the daemon→GUI push path, so it cannot drive GUI-side
  notifications. **The correct trigger is a new GUI-side edge check inside the existing
  `startTrayPoller`/`refreshTrayState` 5s ticker (`app.go:1317-1355`)**, which already runs
  independent of window visibility (satisfies NTF-01 tray-hidden) and already calls
  `a.ListSessions()` every tick. Diff each session's status vs the previous tick to detect the
  non-waiting → `waiting` edge; this IS the once-per-transition de-dup (NTF-02).
- Do NOT reuse `pollSessionStatus` (app.go:279-330) — it is per-session and only runs ~300s
  after creation (wrong tool).

### Agent-type display name (NTF-03)
- Mirror the existing `knownCLIs` display-name table in `internal/pty/detect.go` as a small
  static lookup ("claude" → "Claude Code", etc.). Do NOT call `DetectCLI` (it does a live PATH
  scan).

### Settings UI
- `frontend/src/components/SettingsTab.tsx` — **Behavior** section at line 443
  (`id="settings-behavior"`); reuse the existing toggle pattern there. NOT Session Behavior.

### App-layer wiring
- `app.go` — `maybeNotifyWaiting` de-dup + platform notification dispatch, driven from the tray
  poller edge check above.
- `internal/daemon/engine.go:111` — `daemonSettings` struct (add `NotifyOnWaiting bool`; there
  is NO `Settings` struct in types.go).

### Regression suite convention
- `TESTING.md` — new test files must be registered (Suite Manifest §2, Traceability §4);
  run `bash tests/check-traceability-paths.sh` before committing. Add M-NN manual items for
  any behavior that can't be automated (native OS notification delivery is GUI/OS-level).

</canonical_refs>

<specifics>
## Specific Ideas

- Notification delivery on real OSes cannot be asserted in headless CI — expect a manual
  UAT (M-NN) item for on-screen notification appearance across macOS/Windows/Linux,
  including the tray-hidden case.
</specifics>

<deferred>
## Deferred Ideas

- Click-to-focus / notification actions — out of scope for v4.2.
- Branded macOS attribution (replace "Script Editor") — revisit in a future milestone.
- Notification sound configuration — not in scope.
</deferred>

---

*Phase: 167-native-notifications*
*Context gathered: 2026-07-01 (locked-decision capture; discuss-phase skipped — decisions pre-answered in STATE.md)*
