---
phase: 67-cross-platform-system-tray
plan: 01
subsystem: tray
tags: [linux, dbus, system-tray, sni, dbusmenu]
dependency_graph:
  requires: []
  provides: [linux-tray, shared-tray-helpers]
  affects: [tray_common.go, tray_linux.go, tray.go, tray_test.go]
tech_stack:
  added: [github.com/godbus/dbus/v5 (promoted to direct)]
  patterns: [D-Bus StatusNotifierItem, dbusmenu protocol, ARGB32 pixmap conversion, graceful degradation]
key_files:
  created:
    - tray_common.go
    - tray_common_test.go
    - tray_linux.go
    - tray_linux_test.go
  modified:
    - tray.go (removed trayTooltip — now in tray_common.go)
    - tray_test.go (removed TestTrayTooltip — moved to tray_common_test.go)
    - go.mod (godbus/dbus/v5 promoted from indirect to direct)
decisions:
  - "Always include both separators in BuildMenuItems (Open|sep|sessions|sep|Quit) — gives 4 items with no sessions, matching test spec"
  - "godbus/dbus/v5 promoted to direct dependency — already in go.mod as indirect via Tailscale"
  - "Graceful degradation when StatusNotifierWatcher absent — log warning, do not set disabled=true, SNI service still exported for auto-discovery hosts"
  - "T-67-02 bounds check in Event handler — id < 1 || id > len(items) returns early"
metrics:
  duration_minutes: 15
  completed_date: "2026-04-12"
  tasks_completed: 2
  files_created: 4
  files_modified: 3
---

# Phase 67 Plan 01: Shared Tray Helpers + Linux D-Bus StatusNotifierItem Summary

**One-liner:** Extracted shared tray helpers to platform-neutral tray_common.go and replaced the Linux no-op stub with a full D-Bus StatusNotifierItem + dbusmenu implementation using godbus/dbus/v5.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Extract shared tray helpers | 652a2b8 | tray_common.go, tray_common_test.go, tray.go, tray_test.go |
| 2 (RED) | Add failing Linux tests | cb345ff | tray_linux_test.go |
| 2 (GREEN) | Implement Linux D-Bus tray | ed34143 | tray_linux.go, go.mod |

## What Was Built

### Task 1: Shared Tray Helpers

Created `tray_common.go` (no build tag — available on all platforms) with:
- `trayTooltip(n int) string` — moved from darwin-only `tray.go`
- `MenuItemKind` type and `MenuItemAction`/`MenuItemSeparator` constants
- `MenuItem` struct with Kind, Label, SessionID, Index fields
- `BuildMenuItems(sessions []SessionInfo) []MenuItem` — canonical menu structure for all platforms: Open AgentHub | separator | [sessions] | separator | Quit

Created `tray_common_test.go` with 4 tests covering tooltip formatting and menu structure.

Removed `trayTooltip` from `tray.go` (darwin) and `TestTrayTooltip` from `tray_test.go` — both moved to platform-neutral files.

### Task 2: Linux D-Bus StatusNotifierItem Tray

Replaced the 4-line no-op stub in `tray_linux.go` with a full D-Bus SNI implementation:

**Key components:**
- `pngToARGB32Pixmap()` — decodes embedded 18x18 PNG to ARGB32 byte array for D-Bus `IconPixmap` property (`a(iiay)` format)
- `buildDbusMenuLayout()` — converts `[]MenuItem` to `*dbusMenuNode` with sequential IDs and string property maps for dbusmenu protocol
- `StatusNotifierItemExporter` — exports `org.kde.StatusNotifierItem` interface with `Activate`, `ContextMenu`, `SecondaryActivate`, `Scroll` methods
- `DbusMenuExporter` — exports `com.canonical.dbusmenu` interface with `GetLayout`, `Event`, `GetGroupProperties`, `AboutToShow`, `AboutToShowGroup`, `EventGroup`
- `linuxTray` struct — holds D-Bus connection, bus name, icon pixmaps, menu items, mutex, app reference, and `disabled` flag
- `initTray()` — connects to session bus via `dbus.SessionBusPrivate()`, requests bus name `org.kde.StatusNotifierItem-{pid}-1`, exports SNI at `/StatusNotifierItem` and dbusmenu at `/MenuBar`, registers with StatusNotifierWatcher
- `updateTray()` — swaps icon pixmap (connected/error), rebuilds menu, increments revision, emits `NewIcon`, `NewToolTip`, and `LayoutUpdated` signals
- `cleanupTray()` — closes D-Bus connection
- `setDockVisible()` — no-op (Linux has no Dock equivalent)

**Security:** T-67-02 bounds check in `Event` handler validates `id` is in `[1, len(items)]` before dispatching.

**Graceful degradation:** If D-Bus session bus, auth, Hello, name acquisition, or object export fails — logs a warning and sets `disabled=true`. If `StatusNotifierWatcher` registration fails — logs a warning but keeps the SNI service exported (some tray hosts auto-discover without watcher).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] BuildMenuItems always includes first separator**
- **Found during:** Task 2 (TestBuildMenuItemsEmpty failing — expected 4 items, got 3)
- **Issue:** Original implementation only added the first separator when sessions were present. The plan spec said 4 items with no sessions (Open, sep, sep, Quit).
- **Fix:** Changed `BuildMenuItems` to always include the initial separator after "Open AgentHub", giving the correct 4-item structure with no sessions.
- **Files modified:** tray_common.go
- **Commit:** 652a2b8 (fix applied before commit)

**2. [Rule 3 - Blocking] dbus.Exporter type doesn't exist in godbus v5.2.0**
- **Found during:** Task 2 first compile attempt
- **Issue:** Initial implementation used `dbus.Exporter(dbus.NewDefaultSignalHandler())` which is not a valid godbus v5 API call.
- **Fix:** Removed the introspection export call — not required for basic SNI functionality. Used `conn.ExportAll()` (which exists) for the SNI and dbusmenu objects.
- **Files modified:** tray_linux.go
- **Commit:** ed34143 (fix applied before commit)

**3. [Rule 2 - Missing critical functionality] godbus promoted to direct dependency**
- **Found during:** Task 2 implementation
- **Issue:** godbus was indirect — plan explicitly says to promote it to direct.
- **Fix:** Ran `go mod tidy` after adding the import, which promoted it automatically.
- **Files modified:** go.mod
- **Commit:** ed34143

## Known Stubs

None — all interface methods are implemented. The `setDockVisible` no-op is intentional (Linux has no Dock).

## Threat Flags

None — no new network endpoints or auth paths introduced. D-Bus session bus is per-user with Unix socket auth (T-67-01 accepted).

## Self-Check: PASSED

- tray_common.go: FOUND
- tray_common_test.go: FOUND
- tray_linux.go: FOUND
- tray_linux_test.go: FOUND
- Commit 652a2b8: FOUND
- Commit cb345ff: FOUND
- Commit ed34143: FOUND
- `go vet ./...` (darwin): PASSED
- `GOOS=linux go vet ./...`: PASSED
- `GOOS=linux go build ./...`: PASSED
- All darwin tests pass: PASSED
