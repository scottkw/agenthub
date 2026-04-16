# Phase 80: Tailscale Detection - Context

**Gathered:** 2026-04-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Broaden Tailscale binary detection to cover all common installation methods (Homebrew, system package manager, Snap, Flatpak, Windows default) and make connection-state reporting reliable and granular across macOS, Linux, and Windows. Two requirements: TS-01 (detection breadth) and TS-02 (connection state reliability).

</domain>

<decisions>
## Implementation Decisions

### Detection Granularity
- **D-01:** Expand `TailscaleHealth` from 2-state to 4-state model: Not Installed → Installed (daemon stopped) → Running (disconnected) → Connected. Add `DaemonUp bool` field to the struct.
- **D-02:** Single `CheckHealth(ctx)` call cascades through all 4 states: binary detection → daemon socket probe → connection status → cert check. One call, one result.

### Platform-Specific Paths
- **D-03:** Use well-known hardcoded paths per platform for binary detection:
  - macOS: `/Applications/Tailscale.app/Contents/MacOS/Tailscale`, `/opt/homebrew/bin/tailscale`, `/usr/local/bin/tailscale`
  - Linux: `/usr/bin/tailscale`, `/usr/sbin/tailscale`, `/snap/bin/tailscale`, `/var/lib/flatpak/exports/bin/tailscale`, `~/.local/share/flatpak/exports/bin/tailscale`
  - Windows: `C:\Program Files\Tailscale\tailscale.exe`, `C:\Program Files (x86)\Tailscale\tailscale.exe`, Registry `HKLM\SOFTWARE\Tailscale IPN`
- **D-04:** Trust the `tailscale.com/client/local` library default socket paths for daemon communication. No multi-socket probing — the library handles platform differences.

### Failure UX
- **D-05:** Health modal uses a stepped checklist with green/yellow/red indicators for all 4 states. First failing step shows platform-specific instructions. Steps after a failure are grayed out.
- **D-06:** "Daemon stopped" state shows platform-specific text instructions only — no action buttons. macOS: "Open Tailscale from Applications/menu bar". Linux: "sudo systemctl start tailscaled". Windows: "Open Tailscale from Start menu/system tray".

### Settings Override
- **D-07:** Settings > Paths includes a Tailscale CLI path field (empty = auto-detect). If user sets a custom path, detection uses it first, then falls back to auto-detect. Follows the Phase 79 path persistence + browse button pattern.

### Claude's Discretion
- Exact order of well-known path checking (PATH first vs hardcoded first)
- Whether to cache the detected binary path after first successful detection
- Internal error handling and timeout strategy for the cascaded health check
- How to structure platform-specific path lists (build tags vs runtime GOOS check)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Tailscale Detection
- `internal/webserver/tailscale.go` — Current `CheckHealth()` and `TailscaleHealth` struct (to be extended with `DaemonUp`)
- `internal/webserver/tailscale_test.go` — Existing health check tests (inject pattern)
- `internal/pty/detect.go` — Current `DetectCLIs()` with PATH-only detection (Tailscale is in `knownCLIs`)
- `internal/pty/detect_test.go` — CLI detection tests

### Path Infrastructure
- `internal/daemon/path.go` — `AugmentServicePath()` with well-known directories
- `internal/daemon/path_windows.go` — Windows-specific extra paths (already has `C:\Program Files\Tailscale`)

### Frontend Integration
- `app.go` §`GetTailscaleStatus` — Wails binding that calls `CheckHealth()`
- `app.go` §`startHealthPoller` — Background health polling loop
- `frontend/src/components/SettingsTab.tsx` — Settings panel (Phase 79 added path persistence + browse)
- `frontend/src/components/LocalNetworkBanner.tsx` — Consumes Tailscale health state

### Requirements
- `.planning/REQUIREMENTS.md` §TS-01, §TS-02 — Detection breadth and connection state requirements

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `webserver.CheckHealth()` / `checkHealth()` — injectable `statusFunc` pattern for testability; extend rather than replace
- `internal/daemon/path.go` `AugmentServicePath()` — already has snap/flatpak/homebrew dirs; platform path list can inform detection
- `internal/daemon/path_windows.go` `platformExtraBins()` — Windows path pattern to follow
- `app.go` `startHealthPoller()` — 10s background polling with change-detection; will automatically pick up new health fields
- Phase 79 Settings path persistence + browse buttons — same UX pattern for Tailscale path override

### Established Patterns
- Build tags for platform-specific code (`path_windows.go` with `//go:build windows`)
- Injectable function types for testability (`statusFunc` in both `tailscale.go` and `tailnet.go`)
- Health state emitted via Wails events (`tailscale:health`)

### Integration Points
- `TailscaleHealth` struct consumed by: frontend health modal, `startHealthPoller`, `GetTailscaleStatus` binding, `StartWebServerTailscale`
- Settings path fields: Phase 79 added `GetCLIPaths`/`SaveCLIPaths` — Tailscale path override integrates here
- `DetectCLIs()` in `internal/pty/detect.go` — Tailscale detection should move to or coordinate with the new broader detection

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 80-tailscale-detection*
*Context gathered: 2026-04-16*
