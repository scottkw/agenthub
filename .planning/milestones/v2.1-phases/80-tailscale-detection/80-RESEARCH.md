# Phase 80: Tailscale Detection — Research

**Researched:** 2026-04-16
**Status:** Complete

## Current Architecture

### TailscaleHealth struct (`internal/webserver/tailscale.go`)
- 2-state model: `Installed` (daemon socket reachable) + `Connected` (BackendState == "Running")
- Also tracks: `HasCerts`, `IP`, `Domain`
- Injectable `statusFunc` for testability — `checkHealth(ctx, fn)` accepts fake status functions
- `CheckHealth(ctx)` is the public API, creates `local.Client` and calls `StatusWithoutPeers`

### CLI Detection (`internal/pty/detect.go`)
- `DetectCLIs()` uses `exec.LookPath()` — PATH-only detection
- `tailscale` is already in `knownCLIs` list
- No platform-specific path augmentation in this package

### PATH Augmentation (`internal/daemon/path.go`)
- `AugmentServicePath()` already prepends: `/opt/homebrew/bin`, `/usr/local/bin`, `/snap/bin`, `/var/lib/flatpak/exports/bin`, `~/.local/share/flatpak/exports/bin`
- Windows: `C:\Program Files\Tailscale` via `path_windows.go`
- Build tag pattern: `path_windows.go` (`//go:build windows`) + `path_other.go` (`//go:build !windows`)
- Called once at startup before any LookPath

### Frontend (`SettingsTab.tsx`, `LocalNetworkBanner.tsx`)
- `tailscaleStatusClass()` returns `ok`/`warn`/`error` based on `installed` + `connected`
- `tailscaleStatusText()` returns "Connected"/"Not Connected"/"Not detected"
- `LocalNetworkBanner` has 3-state logic: connected / installed-not-connected / not-installed
- Settings already has Tailscale path override row with Browse button (Phase 79)

### Health Poller (`app.go:760`)
- 10s ticker, emits `tailscale:health` on state change
- `GetTailscaleStatus()` wraps `CheckHealth` with 5s timeout

## Key Decisions from CONTEXT.md

1. **D-01:** Expand to 4-state: Not Installed → Installed (daemon stopped) → Running (disconnected) → Connected. Add `DaemonUp bool`.
2. **D-02:** Single `CheckHealth(ctx)` cascades through all 4 states.
3. **D-03:** Hardcoded well-known paths per platform for binary detection.
4. **D-04:** Trust `tailscale.com/client/local` library default socket paths. No multi-socket probing.
5. **D-05:** Stepped checklist with green/yellow/red indicators. Grayed out after first failure.
6. **D-06:** "Daemon stopped" shows text-only platform-specific instructions. No action buttons.
7. **D-07:** Settings path override — same Phase 79 pattern.

## Implementation Approach

### Backend Changes

**1. Extend TailscaleHealth struct:**
- Add `DaemonUp bool` field
- Add `PlatformHint string` field (for frontend platform-specific instructions)

**2. Rewrite checkHealth cascade:**
- Step 1: Binary detection — try custom path from settings first, then well-known paths, then PATH
- Step 2: If binary found → probe daemon socket via `local.Client.StatusWithoutPeers`
- Step 3: If daemon responds → check `BackendState`
- Step 4: If connected → check certs
- Error at step 2 = Installed but daemon stopped (`DaemonUp: false`)
- Non-"Running" state at step 3 = Running but disconnected

**3. Platform-specific binary paths:**
- New file `internal/webserver/tailscale_paths.go` with build tags OR runtime `GOOS` check
- Paths from D-03 as constants/variables
- Detection order: custom path → well-known hardcoded → PATH (exec.LookPath)

**4. Settings integration:**
- Read custom Tailscale path from daemon client (already has `GetCLIPaths`/`UpdateCLIPath`)
- Pass to `CheckHealth` or have it read from a package-level config

### Frontend Changes

**5. Update SettingsTab.tsx:**
- Update `tailscaleStatusClass()` to handle `daemonUp` boolean — 4 states
- Update `tailscaleStatusText()` for 4 states
- Add collapsible `<details>` checklist below status when not "Connected"

**6. Update LocalNetworkBanner.tsx:**
- Distinguish "daemon stopped" from "not installed" for more specific nudge text

### Testing

**7. Extend tailscale_test.go:**
- Test all 4 health states
- Test binary detection with well-known paths (mock filesystem)
- Test custom path override takes precedence

**8. Update detect_test.go if detection moves/coordinates**

## Risks and Considerations

- **Binary detection vs daemon detection:** Currently `Installed` means "daemon socket reachable." Phase 80 splits this into "binary exists" (file on disk) and "daemon running" (socket reachable). The `local.Client` library error when daemon is down vs not installed both return errors — need to distinguish.
- **Platform hint delivery:** Frontend needs to know the platform for instruction text. Options: (a) Go backend includes `PlatformHint` in TailscaleHealth, (b) frontend detects via user agent. Option (a) is cleaner since Go already knows `runtime.GOOS`.
- **Custom path integration:** `GetCLIPaths()` goes through the daemon client. If daemon isn't connected, can't read custom paths. May need local fallback (read from config file directly).
- **macOS app bundle path:** `/Applications/Tailscale.app/Contents/MacOS/Tailscale` — note this is the GUI app binary, not the CLI. The CLI may be at a different path or symlinked. Need to verify which binary is the correct one for `tailscale status` commands.

## Validation Architecture

### Critical Path Tests
1. Binary detection on each platform finds all well-known paths
2. 4-state cascade produces correct TailscaleHealth for each scenario
3. Custom path override takes precedence over auto-detect
4. Frontend renders correct status text and dot color for all 4 states
5. Checklist shows correct pass/fail/gray states

### Edge Cases
- Tailscale installed but binary in unexpected location (only PATH has it)
- Daemon running but network unreachable (timeout scenario)
- Custom path set to invalid binary
- Multiple Tailscale installations (Homebrew + app bundle)

---

*Phase: 80-tailscale-detection*
*Research completed: 2026-04-16*
