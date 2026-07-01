---
phase: 167-native-notifications
verified: 2026-07-01T07:08:33Z
status: human_needed
score: 10/11 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "M-41 — per-platform manual notification delivery (macOS/Windows/Linux)"
    expected: "On each of macOS, Windows, and Linux: enable the toggle, hide the window to tray (QuitGUIOnly-style, not full quit), drive a session into `waiting`, confirm exactly ONE OS-native notification appears identifying session name + agent type while the window is hidden. Repeat with the toggle OFF and confirm no notification appears. macOS must show real 'AgentHub' attribution (native UNUserNotificationCenter path, not a generic/Script-Editor label)."
    why_human: "Real OS notification centers require a live desktop session on each of the three target OSes; no CI runner (build.yml) has one on any platform. The trigger logic, de-dup, cold-start baseline, body format, and disabled-toggle no-op are all unit-proven through the injected sendNotificationFunc seam (Plan 03) and the code paths for macOS UNUserNotificationCenter / beeep (Windows, Linux) are proven to compile and call the correct APIs on all three GOOS targets — but no automated test can observe an actual OS notification banner rendering."
---

# Phase 167: Native Notifications Verification Report

**Phase Goal:** Users who opt in receive a single native OS notification the moment a session transitions to awaiting-input state, on macOS, Windows, and Linux.
**Verified:** 2026-07-01T07:08:33Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A non-waiting→`waiting` transition produces a real, visible native OS notification on macOS, Windows, and Linux, including when the app window is hidden in the tray (ROADMAP SC1 / NTF-01) | ⚠️ NEEDS HUMAN (code path present + wired + unit tested; real on-screen delivery unautomatable) | `refreshTrayState` (app.go:1443-1460) calls `a.maybeNotifyWaiting(sessions)` in the connected branch every 5s tick, independent of window visibility — the tray poller runs regardless of tray-hidden state. `sendNotificationFunc` defaults to the real `sendNotification` (app.go:128). Native macOS path retained (`notification_darwin.go`, `tray_objc_darwin.m` — real `UNUserNotificationCenter` + per-session `requestWithIdentifier:`); Windows/Linux wrappers call `beeep.Notify` (`notification_windows.go`, `notification_linux.go`). `go build`/`go vet` clean on darwin natively (CGO_ENABLED=1) and cross-compiles clean for `GOOS=windows` and `GOOS=linux`. No automated test asserts an actual OS-rendered banner on any platform — deferred to manual **M-41** (TESTING.md Category U), registered and cross-referenced in VALIDATION.md's Manual-Only Verifications table. |
| 2 | A session held in `waiting` fires exactly once per transition — no repeat notifications while it remains `waiting` (ROADMAP SC2 / NTF-02) | ✓ VERIFIED | `TestMaybeNotifyWaiting` passes (`go test . -run TestMaybeNotifyWaiting -v`); edge-detection logic at app.go:163-189 fires only when `known && prev != waiting && current == waiting`. |
| 3 | Cold-start baseline: the first poll tick after launch captures current statuses silently — no notification burst for sessions already `waiting` at launch | ✓ VERIFIED | `TestMaybeNotifyWaiting_FirstTickNoNotify` passes; `firstRun` branch (app.go:167-178) baselines without calling `sendNotificationFunc`. |
| 4 | The notification text includes the session name and agent type (ROADMAP SC3 / NTF-03) | ✓ VERIFIED | `TestMaybeNotifyWaiting_BodyFormat` + `TestDisplayNameForCLI` (7 subtests) pass; body built as `"%s (%s) is waiting for your input."` with `displayNameForCLI(s.CLI)` (app.go:139-154, 180). |
| 5 | A Settings toggle enables/disables notifications, defaults to off, and suppresses all notifications when off (ROADMAP SC4 / NTF-04) | ✓ VERIFIED | `TestMaybeNotifyWaiting_DisabledNoop` passes (backend no-op when `!a.notifyOnWaiting.Load()`); `SettingsTab.notify-toggle.test.tsx` (15 tests, all pass) covers default-off render, load via `GetNotifyOnWaiting`, save via `SetNotifyOnWaiting`, and rejection handling. |
| 6 | `NotifyOnWaiting` persists to `settings.json` (0600) and survives a daemon restart; defaults false when absent, no schema-version bump | ✓ VERIFIED | `TestNotifyOnWaiting_Default`, `_Persists`, `_RoundTrip`, `_NoSchemaBump` all pass (`go test ./internal/daemon/ -race -run NotifyOnWaiting -v`); `CurrentSchemaVersion` unchanged; `notifyOnWaiting,omitempty` confirmed in engine.go:115. |
| 7 | `GET`/`PATCH /settings/notify-on-waiting` REST routes + `DaemonClient` round-trip over the Unix socket | ✓ VERIFIED | `TestAPIGetNotifyOnWaiting_Default`, `TestAPIPatchNotifyOnWaiting_FlipsTrue`, `TestAPIPatchNotifyOnWaiting_BadBody`, `TestDaemonClient_GetSetNotifyOnWaiting_RoundTrip` all pass; routes registered at api.go:125-126. |
| 8 | `sendNotification(identifier, title, body string)` has an identical signature across darwin/windows/linux; native macOS attribution retained (not replaced by beeep); Windows/Linux wrappers log-and-swallow delivery errors | ✓ VERIFIED | All three files declare the 3-arg signature; `tray_objc_darwin.m` uses the passed `identifier` in `requestWithIdentifier:` (no longer hardcoded); `notification_windows.go`/`notification_linux.go` each call `beeep.Notify` once and `log.Printf` on error without returning it. `notification_other.go` confirmed deleted. `go.mod` pins `github.com/gen2brain/beeep v0.11.2`. |
| 9 | The toggle is physically located under `#settings-behavior` (NOT `#settings-session-behavior`), per the LOCKED user correction | ✓ VERIFIED | SettingsTab.tsx: toggle JSX (lines ~505-527) sits between `<h3 id="settings-behavior">` (line 475) and `<h3 id="settings-session-behavior">` (line 531). Source-level placement guard test in `SettingsTab.notify-toggle.test.tsx` passes. |
| 10 | Settings search surfaces the notify toggle and targets `settings-behavior` | ✓ VERIFIED | `SettingsSearch.tsx:30` — `{ label: 'Notify me when a session is awaiting input', target: 'settings-behavior' }`. |
| 11 | M-41 manual item + Category U + traceability rows registered in TESTING.md; `check-traceability-paths.sh` exits 0 | ✓ VERIFIED | `bash tests/check-traceability-paths.sh` → `OK: all traceability paths exist` (exit 0); `TESTING.md` contains `### Category U — Native Notifications (NTF)`, **M-41**, and traceability rows for NTF-01..04 mapping to `app_test.go`, `engine_notify_test.go`, `api_notify_test.go`, `SettingsTab.notify-toggle.test.tsx`. |

**Score:** 10/11 truths verified (1 routed to human verification — real cross-platform OS notification delivery, which is inherently unautomatable in CI)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/daemon/engine.go` | `daemonSettings.NotifyOnWaiting` + engine field + Get/Set | ✓ VERIFIED | Field, engine field, load/save wiring, `GetNotifyOnWaiting`/`SetNotifyOnWaiting` all present (lines 48, 115, 214, 250, 1107-1120) |
| `internal/daemon/api.go` | GET/PATCH `/settings/notify-on-waiting` routes + handlers | ✓ VERIFIED | Routes registered lines 125-126; handlers lines 874-887 |
| `internal/daemon/client.go` | `DaemonClient.GetNotifyOnWaiting`/`SetNotifyOnWaiting` | ✓ VERIFIED | Lines 166-179 |
| `internal/daemon/engine_notify_test.go`, `api_notify_test.go` | Unit tests | ✓ VERIFIED | All 8 tests pass |
| `notification_windows.go`, `notification_linux.go` | beeep wrappers | ✓ VERIFIED | Both present, log-and-swallow, `sendNotification(identifier, title, body string)` |
| `notification_darwin.go`, `tray_objc_darwin.m` | Identifier threaded to native macOS path | ✓ VERIFIED | Both updated; `notification_other.go` confirmed deleted |
| `go.mod`/`go.sum` | `github.com/gen2brain/beeep v0.11.2` | ✓ VERIFIED | Pinned |
| `app.go` | `displayNameForCLI`, `maybeNotifyWaiting`, App fields, `refreshTrayState` wiring, Get/SetNotifyOnWaiting, startup load | ✓ VERIFIED | All present and wired (see truths table) |
| `app_test.go` | `TestMaybeNotifyWaiting_*`, `TestDisplayNameForCLI` | ✓ VERIFIED | All 6 tests pass |
| 4 wailsjs App binding files | `GetNotifyOnWaiting`/`SetNotifyOnWaiting` | ✓ VERIFIED | `grep -rl 'GetNotifyOnWaiting' frontend/src/wailsjs/` returns exactly 4 files |
| `frontend/src/components/SettingsTab.tsx` | Toggle in Behavior section | ✓ VERIFIED | Confirmed placement + state/load/save wiring |
| `frontend/src/components/SettingsSearch.tsx` | Search index entry | ✓ VERIFIED | Entry present, `target: 'settings-behavior'` |
| `SettingsTab.notify-toggle.test.tsx` | Render/default/load/save coverage | ✓ VERIFIED | 15/15 tests pass |
| `TESTING.md` | Category U, M-41, Suite Manifest note, traceability rows | ✓ VERIFIED | All present; path checker exits 0 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `SetNotifyOnWaiting` (engine) | `saveSettingsToDisk` | mutates `e.notifyOnWaiting` under `e.mu.Lock` then saves | ✓ WIRED | engine.go:1117-1120 |
| `loadSettingsFromDisk` | `e.notifyOnWaiting` | assigns `s.NotifyOnWaiting` | ✓ WIRED | engine.go:214 |
| `refreshTrayState` | `maybeNotifyWaiting` | called after `a.ListSessions()` in connected branch | ✓ WIRED | app.go:1456-1457 |
| `maybeNotifyWaiting` | `sendNotificationFunc` | calls with per-session identifier `agenthub.session-waiting.<id>` | ✓ WIRED | app.go:181 |
| `startup` | `a.notifyOnWaiting.Store` | reads `client.GetNotifyOnWaiting()` on daemon connect | ✓ WIRED | app.go:230-234 |
| `SetNotifyOnWaiting` (App) | `client.SetNotifyOnWaiting` | stores atomic then persists via daemon client | ✓ WIRED | app.go:727-733 |
| `SettingsTab.tsx` toggle `onChange` | `SetNotifyOnWaiting` (wailsjs) | `handleToggleNotifyOnWaiting` calls the binding, updates state on success | ✓ WIRED | SettingsTab.tsx:378-388 |
| `SettingsTab.tsx` mount `useEffect` | `GetNotifyOnWaiting` (wailsjs) | initial checked state loaded | ✓ WIRED | SettingsTab.tsx:211-214 |
| `tray_objc_darwin.m` `sendNotification` | `requestWithIdentifier:` | uses passed identifier, not hardcoded string | ✓ WIRED | tray_objc_darwin.m:160-174 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Daemon settings persistence (default/persist/round-trip/no-schema-bump) | `go test ./internal/daemon/ -race -run NotifyOnWaiting -v` | 4/4 pass | ✓ PASS |
| REST + client round-trip | `go test ./internal/daemon/ -race -run Notify -v` | 8/8 pass (incl. unrelated pre-existing "Notify" tests) | ✓ PASS |
| App-layer trigger + display name | `go test . -run 'TestMaybeNotifyWaiting|TestDisplayNameForCLI' -v` | 6/6 pass (12 subtests) | ✓ PASS |
| Frontend toggle | `cd frontend && npx vitest run SettingsTab.notify-toggle.test.tsx` | 15/15 pass | ✓ PASS |
| Native darwin build | `go build ./... && go vet ./...` (CGO_ENABLED=1, native) | clean | ✓ PASS |
| Cross-compile windows | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...` | clean | ✓ PASS |
| Cross-compile linux | `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...` | clean | ✓ PASS |
| Frontend type-check | `cd frontend && npx tsc --noEmit` | clean | ✓ PASS |
| Full Go suite | `go test -race -short ./...` | all packages ok | ✓ PASS |
| Full frontend suite | `cd frontend && npx vitest run` | 135 files / 2268 tests pass | ✓ PASS |
| Traceability path checker | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist` | ✓ PASS |
| Real OS notification banner rendering (3 platforms) | — (requires live desktop session) | not run | ? SKIP — routed to human verification (M-41) |

### Probe Execution

No probes declared for this phase and no conventional `scripts/*/tests/probe-*.sh` files found. Skipped.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|--------------|------------|-------------|--------|----------|
| NTF-01 | 167-02, 167-03 | Native OS notification on `waiting` transition, macOS/Windows/Linux, tray-hidden | ✓ SATISFIED (code); real delivery → human verification | Trigger wired in tray poller; delivery primitives present on all 3 platforms and build clean |
| NTF-02 | 167-03 | Fires once per transition, no repeats | ✓ SATISFIED | `TestMaybeNotifyWaiting`, `TestMaybeNotifyWaiting_FirstTickNoNotify` pass |
| NTF-03 | 167-03 | Notification identifies session name + agent type | ✓ SATISFIED | `TestMaybeNotifyWaiting_BodyFormat`, `TestDisplayNameForCLI` pass |
| NTF-04 | 167-01, 167-03, 167-04 | Settings toggle, default off | ✓ SATISFIED | Persistence, REST, client, App binding, and UI toggle all present and tested |

No orphaned requirements — REQUIREMENTS.md maps exactly NTF-01..04 to Phase 167, and all four appear in at least one plan's `requirements` frontmatter field (167-01: NTF-04; 167-02: NTF-01; 167-03: NTF-01/02/03/04; 167-04: NTF-04).

### Anti-Patterns Found

None. Scanned all phase-modified files (`internal/daemon/engine.go`, `api.go`, `client.go`, `engine_notify_test.go`, `api_notify_test.go`, `notification_windows.go`, `notification_linux.go`, `notification_darwin.go`, `tray_objc_darwin.m`, `app.go`, `app_test.go`, `frontend/src/components/SettingsTab.tsx`, `SettingsSearch.tsx`, `SettingsTab.notify-toggle.test.tsx`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` and stub patterns — no matches related to this phase's work.

### Human Verification Required

### 1. M-41 — Cross-platform on-screen notification delivery (macOS/Windows/Linux)

**Test:** On each of macOS, Windows, and Linux: enable the notify-on-waiting toggle in Settings → Behavior, hide the window to the system tray (QuitGUIOnly-style, not a full quit), drive a session into the `waiting` state (e.g. trigger a `[y/n]` prompt), and observe the OS notification center.

**Expected:** Exactly ONE native OS notification appears, identifying the session name and agent type, while the app window is hidden in the tray. On macOS the notification must show real "AgentHub" attribution (the native `UNUserNotificationCenter` path, not a generic label). Repeating with the toggle OFF must produce no notification.

**Why human:** Real OS notification centers require a live desktop session on each of the three target operating systems; no CI runner in `build.yml` has one on any platform. This is the only remaining check — all trigger logic, de-duplication, cold-start baseline suppression, body formatting, and the disabled-toggle no-op are unit-proven via the injected `sendNotificationFunc` seam, and the delivery code compiles/links correctly on all three GOOS targets (confirmed via native darwin build and windows/linux cross-compiles in this verification). This item is already tracked as **M-41** in `TESTING.md` (Category U) and was not skipped or hidden by the executing plans — all four SUMMARY.md files explicitly flag it as `human_judgment: true` and defer to this manual check.

### Gaps Summary

No gaps found. All 10 codebase-verifiable truths pass with real (not stubbed) implementations: the daemon settings layer mirrors the proven `StartMinimized` pattern exactly and is unit-tested for default/persist/round-trip/no-schema-bump; the cross-platform notification primitive retains the real native macOS attribution path (per an explicit LOCKED decision rejecting beeep's "Script Editor" fallback) while adding genuine beeep-backed Windows/Linux wrappers with correct log-and-swallow error handling; the GUI-side edge-detection trigger (`maybeNotifyWaiting`) is exercised by 6 passing unit tests covering exactly-once-per-transition, cold-start baseline suppression, body format, the disabled-toggle no-op, and map pruning; and the Settings UI toggle is correctly placed per the LOCKED user correction, defaults off, and round-trips through the Wails binding with 15 passing component tests. The single remaining item — the phase goal's literal claim that a user "receives a native OS notification" — is a runtime, cross-platform OS-integration behavior that cannot be observed by any automated check in this environment; the phase's own artifacts (VALIDATION.md, all four SUMMARY.md files, TESTING.md) already correctly identify and register this as manual item M-41 rather than silently claiming it as tested. This routes the phase to `human_needed` rather than `passed`, per the verification decision tree (Step 9) — human sign-off on M-41 across macOS/Windows/Linux is the only outstanding step before the phase goal can be certified as fully achieved.

---

_Verified: 2026-07-01T07:08:33Z_
_Verifier: Claude (gsd-verifier)_
