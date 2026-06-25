# Phase 150: Shell-Sharing Warning Toggle - Context

**Gathered:** 2026-06-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Add a daemon-backed Settings toggle that controls whether the shell-session
web-share security warning appears. The toggle can re-enable the warning even
after the user previously acknowledged the one-time banner.

**Scope note (expanded during discussion — exceeds a literal reading of SET-01/#51):**
Phase 137 split sharing into two surfaces, and the shell warning only follows
one of them. To make the toggle meaningful, this phase ALSO wires the shell
warning into the **Hub Share modal** (the primary v4.0 share surface), not just
the legacy per-tab StatusBar path. This closes a cross-surface parity gap
(release-blocking per the standing parity rule). SET-01 wording and the Phase
150 ROADMAP goal should be read as including this parity fix.

**In scope:**
- New daemon-backed setting governing the shell web-share warning (default ON)
- Settings UI toggle (Session Behavior section) with a confirm-on-disable dialog
- Wire the warning into BOTH share entry points for shell sessions: the Hub
  Share modal ON-path AND the existing StatusBar toggle path
- Re-arm semantics: toggling the setting OFF→ON resets the one-time `warned` flag

**Out of scope:**
- Retiring/consolidating the legacy StatusBar per-tab web toggle (separate concern)
- Changing the HomeDirWriteWarning (D-09) — different warning, different threat
- Any change to the warning's copy/threat model itself
</domain>

<decisions>
## Implementation Decisions

### Toggle Model (warning ↔ one-time acknowledgment interplay)
- **D-01:** Introduce a NEW daemon-backed setting (the "warning enabled" master
  switch). It is SEPARATE from the existing one-time `shellWebShareWarned` flag.
  Both flags coexist.
- **D-02:** The shell warning fires iff: `session is a shell` AND
  `warningEnabled == true` AND `shellWebShareWarned == false`. This preserves
  the Phase 101 one-time "acknowledge once → don't show again" behavior inside
  the enabled state.
- **D-03:** **Re-arm semantics.** Flipping the Settings toggle OFF→ON MUST reset
  `shellWebShareWarned` to `false`, so the warning shows again on the next shell
  web-share even if it was previously acknowledged (satisfies Success Criterion 2).
- **D-04:** Behavior matrix:
  - `OFF → ON`: warning re-armed → shows next shell web-share
  - `ON`, never acknowledged: shows once, user clicks OK
  - `ON`, already acknowledged: stays suppressed (one-time ack honored)
  - `OFF`: never warns (regardless of `warned`)

### Placement & Label
- **D-05:** Toggle lives in the **Session Behavior** section of Settings
  (`SettingsTab.tsx:413`), directly below the Auto-close-on-exit toggle —
  grouped with other per-session behavior switches.
- **D-06:** Label: **"Warn before web-sharing a shell session."** Reuse the
  established colorblind-safe `role=switch` toggle pattern (matches Auto-close,
  Appearance — no color-only state signal).

### Disable UX
- **D-07:** Turning the toggle **OFF** prompts a confirmation dialog
  ("Disable the shell web-share security warning?" → Cancel / Disable),
  because it weakens a security guardrail. Turning it **ON** is instant
  (no confirmation). This is a deliberate exception to the silent-toggle
  pattern used elsewhere in Settings.

### Default State
- **D-08:** Default **ON** (warning enabled) on fresh install / first run —
  preserves today's safe behavior where a new user is warned the first time
  they web-share a shell.

### Cross-Surface Parity (scope expansion)
- **D-09:** Wire the shell warning into the Hub **Share modal** ON-path
  (`SessionShareModal.handleShareToggle`) for shell sessions. Today that handler
  calls `ToggleWebServing` directly with no shell-warning interception — only
  the legacy StatusBar path (`App.tsx handleToggleWeb`) shows the warning. After
  this phase, both surfaces respect the same `warningEnabled && !warned` gate.
- **D-10:** Reuse the existing `ShellWebShareBanner` component and the
  interception/race-mitigation logic already in `App.tsx` (synchronous local
  `warned` set before awaiting the persist RPC). Don't fork the warning UI.

### Claude's Discretion
- Exact confirm-dialog component/wording (reuse existing modal/dialog primitives).
- The new setting's exact field/RPC name (mirror the `ShellWebShareWarned`
  naming convention — `Get/SetShellWebShareWarningEnabled` is the natural analog).
- Whether the Share-modal warning renders as the existing banner inline or as a
  modal-appropriate variant — researcher/planner to choose, behavior must match.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirement & roadmap
- `.planning/REQUIREMENTS.md` — SET-01 (line ~91). NOTE: phase scope expanded
  beyond the literal wording (see D-09); update SET-01 + Phase 150 ROADMAP goal
  to reflect the Share-modal parity fix.
- `.planning/ROADMAP.md` §"Phase 150: Shell-Sharing Warning Toggle" (line ~635)
- GitHub issue #51 (scottkw/agenthub) — "Add a flag in Settings to
  enable/disable the shell session sharing warning"

### Existing warning mechanism (Phase 101 SHELL-07/08)
- `frontend/src/components/ShellWebShareBanner.tsx` — the warning UI component to reuse
- `frontend/src/App.tsx:853-910` — `handleToggleWeb` interception +
  `handleShellWebShareConfirm` (the warned-flag race mitigation pattern)
- `internal/daemon/engine.go:45,110,195,222,993-1001` — `shellWebShareWarned`
  field, persistence (load/save), Get/Set accessors
- `internal/daemon/api.go:112-113,725-741` — GET/PATCH
  `/settings/shell-web-share-warned` HTTP handlers
- `internal/daemon/client.go:166-178` — `Get/SetShellWebShareWarned` daemon client

### Sharing surfaces (Phase 137 SHARE-01..06)
- `frontend/src/components/Hub/SessionShareModal.tsx:183-199` —
  `handleShareToggle` (primary surface; currently NO shell warning — to be wired)
- `frontend/src/components/StatusBar.tsx` — legacy per-tab web toggle (warning path)

### Settings UI patterns
- `frontend/src/components/SettingsTab.tsx:413` — Session Behavior section
  (placement) + Auto-close toggle pattern (`autoCloseSession`, lines 106-110,
  180-183, 312) to mirror for load/save/saving/error state
- `frontend/src/components/SettingsSearch.tsx` — toggle labels are indexed here;
  keep in sync when adding the new toggle label
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ShellWebShareBanner` component — the warning UI; reuse as-is for both surfaces.
- Auto-close-on-exit toggle (`SettingsTab.tsx`) — exact analog for the new
  toggle's state machine: `Get/Set` RPC, loaded/saving/error flags, role=switch.
- `shellWebShareWarned` daemon plumbing — the new `warningEnabled` setting
  follows the identical engine field → save/load → api handler → client → Wails
  binding chain.

### Established Patterns
- Daemon-backed settings persist via the engine's settings struct
  (`engine.go:110` JSON tags) — new setting added there persists across restarts
  (Success Criterion 3).
- Colorblind-safe toggles use `role=switch` with text/state cues, never
  color-only (D-06 standing constraint).
- Warning race mitigation: set local `warned` synchronously before awaiting the
  persist RPC (`App.tsx:886-890`) — apply the same pattern in the Share modal.

### Integration Points
- `SessionShareModal.handleShareToggle` (line 186) — insert shell-warning
  interception before/around the `ToggleWebServing` call for shell sessions.
- `SettingsTab` Session Behavior section — new toggle + confirm-on-disable dialog.
- New daemon setting endpoint mirrors `/settings/shell-web-share-warned`.
- `SettingsSearch` index — add the new toggle label.
</code_context>

<specifics>
## Specific Ideas

- Toggle label is fixed: "Warn before web-sharing a shell session."
- Confirm-on-disable copy direction: "Disable the shell web-share security
  warning?" with Cancel / Disable actions (final wording at Claude's discretion).
- `SHELL_CLIS` membership (already used in `App.tsx`) is the canonical "is this a
  shell session?" test — reuse it in the Share modal interception.
</specifics>

<deferred>
## Deferred Ideas

- **Retire/consolidate the legacy StatusBar per-tab web toggle.** SHARE-02 says
  the Share modal toggle "replaces Web On," yet the StatusBar toggle still
  exists. Whether to remove it is a separate cleanup, not this phase. Note for
  roadmap backlog.
- **Web-surface behavior** for the warning was not raised — this phase targets
  the desktop GUI share surfaces (Share modal + StatusBar). Remote/web-share
  visitors don't initiate shell sharing, so out of scope here.

</deferred>

---

*Phase: 150-shell-sharing-warning-toggle*
*Context gathered: 2026-06-23*
