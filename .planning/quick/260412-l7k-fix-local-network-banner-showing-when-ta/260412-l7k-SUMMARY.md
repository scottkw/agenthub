---
phase: quick
plan: 260412-l7k
subsystem: daemon, frontend
tags: [tailscale, local-mode, banner, auto-upgrade, settings]
dependency_graph:
  requires: []
  provides: [daemon-local-to-tailscale-upgrade, context-aware-banner, tailscale-path-setting]
  affects: [internal/daemon, frontend/src/App.tsx, frontend/src/components]
tech_stack:
  added: []
  patterns: [background-goroutine-upgrader, frontend-upgrade-polling, context-aware-ui]
key_files:
  created: []
  modified:
    - internal/daemon/api.go
    - internal/daemon/process.go
    - internal/daemon/process_test.go
    - frontend/src/App.tsx
    - frontend/src/components/LocalNetworkBanner.tsx
    - frontend/src/components/__tests__/LocalNetworkBanner.test.tsx
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/components/__tests__/SettingsTab.test.tsx
decisions:
  - "RestartWebServer implemented as stop+start (not idempotent like AutoStartWebServer) to force mode upgrade"
  - "upgradeToTailscale goroutine only launched when daemon starts in local-mode fallback (not when Tailscale is already healthy at startup)"
  - "Frontend polls GetWebServerMode every 3s for up to 10 attempts (30s) after Tailscale connects — matches 15s backend polling interval with margin"
  - "Tailscale path in Settings is informational only (no input field) since Tailscale is accessed via Go library, not CLI binary"
metrics:
  duration: ~15 minutes
  completed: 2026-04-12
  tasks: 2
  files_modified: 8
---

# Quick Task 260412-l7k: Fix Local Network Banner Showing When Tailscale Connected

**One-liner:** Backend goroutine auto-upgrades web server from local to Tailscale mode with context-aware frontend banner and mode re-polling on Tailscale health events.

## What Was Done

### Task 1: Backend auto-upgrade from local to Tailscale mode

Added `RestartWebServer` method to `API` that stops the current web server and starts a new one — unlike `AutoStartWebServer` (which is idempotent), this always replaces the running server to force mode upgrades.

Added `upgradeToTailscale` goroutine in `process.go` that polls Tailscale health every 15 seconds. When it detects a fully healthy Tailscale state (Connected + HasCerts + IP), it calls `RestartWebServer` to switch from local to Tailscale mode. It exits after a successful upgrade or context cancellation.

The goroutine is launched in `runDaemonCore` only when the daemon falls back to local mode (i.e., Tailscale was not available at startup).

### Task 2: Frontend context-aware banner, mode re-polling, Tailscale path setting

**LocalNetworkBanner:** Added `tailscaleConnected` prop. When true, renders "upgrading to Tailscale..." with no CTA button. When false, renders original "Install Tailscale" messaging with CTA.

**App.tsx:** Added `upgradePollerRef` to track the polling interval. When `tailscale:health` fires with a connected+healthy state and `webServerMode` is `local`, starts polling `GetWebServerMode()` every 3 seconds (up to 10 attempts / 30 seconds). When mode changes to `tailscale`, updates state and clears the interval. Interval is also cleared on unmount. Passes `tailscaleConnected` derived from `tailscaleHealth` to `LocalNetworkBanner`.

**SettingsTab:** Added Tailscale status info block in the Paths section showing connection status (connected with domain/ip, installed-but-not-connected, or not detected) with a description explaining that no path configuration is needed since AgentHub uses the Go library to communicate with Tailscale.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | dfcabc2 | feat(quick-260412-l7k): backend auto-upgrade from local to Tailscale mode |
| 2 | e768272 | feat(quick-260412-l7k): context-aware banner, mode re-polling, Tailscale path in settings |

## Test Results

- `go test ./internal/daemon/ -v -count=1`: 70 tests, all PASS
- `npx vitest run --reporter=verbose`: 346 tests, all PASS
- `npx tsc --noEmit`: clean (no type errors)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — no new network endpoints, auth paths, or schema changes introduced. `RestartWebServer` is only callable internally (same trust level as `AutoStartWebServer`). T-quick-02 mitigation satisfied: goroutine exits after successful upgrade and respects context cancellation.

## Self-Check: PASSED

- `internal/daemon/api.go`: contains `RestartWebServer` — FOUND
- `internal/daemon/process.go`: contains `upgradeToTailscale` — FOUND
- `internal/daemon/process_test.go`: contains `TestRestartWebServer_StopsAndStarts` and `TestUpgradeToTailscale_ExitsOnCancel` — FOUND
- `frontend/src/components/LocalNetworkBanner.tsx`: contains `tailscaleConnected` — FOUND
- `frontend/src/components/SettingsTab.tsx`: contains `local daemon socket` — FOUND
- Commit dfcabc2 — FOUND
- Commit e768272 — FOUND
