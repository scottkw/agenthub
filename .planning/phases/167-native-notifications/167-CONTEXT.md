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

### macOS attribution
- beeep on macOS attributes to "Script Editor" — **accepted trade-off for v4.2**. The Title
  field uses "AgentHub" to aid identification. Revisit only if user feedback demands branded
  attribution.

### Library / platform wrappers
- New Go dependency: `github.com/gen2brain/beeep v0.11.2` (cross-platform, no CGO).
- New files: `notification_windows.go`, `notification_linux.go` (beeep platform wrappers).
- Tray-hidden delivery is a hard requirement (NTF-01) — notification must fire even when the
  GUI window is minimized to the system tray.

### Setting field
- `NotifyOnWaiting bool` added to daemon Settings (`internal/daemon/types.go`).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Status transition source (where `waiting` is set)
- `internal/daemon/engine.go` (~line 428, status pkg) — the running/idle/waiting/errored
  status detector. The `→ waiting` transition is the notification trigger point.

### Settings UI
- `frontend/src/components/SettingsTab.tsx` — Behavior section at line 443
  (`id="settings-behavior"`); reuse the existing toggle pattern there.

### App-layer wiring
- `app.go` — `maybeNotifyWaiting` de-dup + platform notification dispatch.
- `internal/daemon/types.go` — `Settings` struct (add `NotifyOnWaiting bool`).

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
