---
phase: 167-native-notifications
verified: 2026-07-01T12:35:00Z
status: human_needed
score: 11/11 must-haves verified
behavior_unverified: 0
overrides_applied: 0
postverify_fix:
  commit: 96c27ff9
  summary: "Test-isolation regression closed inline during --gaps-only execution: added `engine.notifyOnWaiting = false` to testDaemon()'s reset block (internal/daemon/api_test.go:48). Confirmed hermetic — `go clean -testcache && go test -short ./internal/daemon/` passes with the real on-disk settings.json still holding notifyOnWaiting:true. Score advances 10/11 → 11/11; sole remaining item is the M-41 human live-delivery re-test, which reclassifies the phase status from gaps_found to human_needed."
re_verification:
  previous_status: human_needed
  previous_score: 10/11
  gaps_closed:
    - "M-41 darwin crash regression hardened: bundle-id guard (hasValidBundleIdentifier / hasAppBundleID) + @try/@catch added to sendNotification in tray_objc_darwin.m and notification_darwin.go; confirmed present in code, confirmed compiling (go build .), confirmed passing (TestHasAppBundleID_FalseWhenUnbundled, TestSendNotification_NoBundleReturnsCleanly), confirmed registered in TESTING.md (Suite Manifest note, NTF-01 traceability row, Category U M-41 update) with check-traceability-paths.sh exiting 0"
  gaps_remaining:
    - "M-41 live on-screen notification delivery on a signed production build (macOS/Windows/Linux) — unchanged from the previous verification, inherently unautomatable, still routed to human verification"
  regressions:
    - "internal/daemon package: TestAPIGetNotifyOnWaiting_Default, TestAPIPatchNotifyOnWaiting_BadBody, TestDaemonClient_GetSetNotifyOnWaiting_RoundTrip now FAIL on this checkout (go test -race -short ./internal/daemon/) — NOT failing in the previous verification's 8/8 pass claim. Root cause confirmed by isolation: testDaemon() (internal/daemon/api_test.go) never resets engine.notifyOnWaiting after NewSessionEngine() loads the real on-disk settings.json, even though the surrounding code comment explicitly requires 'every field that loadSettingsFromDisk touches must appear in this reset block. Future engine fields require an addition here.' Phase 167-01 added NotifyOnWaiting to loadSettingsFromDisk (engine.go:214) but never added the corresponding engine.notifyOnWaiting = false line to this reset block. This dev machine's real settings.json currently has \"notifyOnWaiting\":true (set via manual UAT of the toggle sometime between the previous verification run and this one), which leaks into the test process and flips the 3 tests' expected-false assertions to fail. Moving the real settings.json aside makes all tests pass again; restoring it reproduces the failure — causality confirmed, not an environment fluke."
gaps:
  - truth: "GET/PATCH /settings/notify-on-waiting REST routes + DaemonClient round-trip over the Unix socket (NTF-04)"
    status: resolved
    resolved_by: "commit 96c27ff9 — added engine.notifyOnWaiting = false to testDaemon() reset block (internal/daemon/api_test.go:48); daemon suite now hermetic regardless of real settings.json state"
    original_status_was: failed
    reason: "The regression suite backing this truth is not hermetic: testDaemon()'s reset block (internal/daemon/api_test.go, lines ~41-48) omits engine.notifyOnWaiting, so NewSessionEngine()'s unconditional read of the real ~/Library/Application Support/agenthub/settings.json leaks into every api_notify_test.go test. On this checkout, with the real settings file at notifyOnWaiting:true (a plausible and current state — the toggle was manually exercised for UAT), 3 of 4 api_notify_test.go tests fail. This is a genuine, machine-state-dependent test-isolation bug in code shipped by this phase, not an environment fluke — verified by moving the real settings.json aside (tests pass) and restoring it (tests fail again, reproducibly)."
    artifacts:
      - path: "internal/daemon/api_test.go"
        issue: "testDaemon() resets engine.startMinimized, engine.shellWebShareWarned, engine.shellWebShareWarningEnabled, engine.shellPath, engine.autoCloseSession, engine.pluginSettings, engine.cliPaths but NOT engine.notifyOnWaiting — despite the field being added to loadSettingsFromDisk in engine.go:214 by this same phase (167-01), and despite the surrounding comment explicitly stating this obligation."
    missing:
      - "Add `engine.notifyOnWaiting = false` to the reset block in testDaemon() (internal/daemon/api_test.go, alongside the other engine.* resets at lines 43-48)."
      - "Re-run `go test -race -short ./internal/daemon/ -run NotifyOnWaiting` and `go test -race -short ./...` to confirm hermetic pass regardless of the real on-disk settings.json state."
deferred: []
human_verification:
  - test: "M-41 — per-platform manual notification delivery (macOS/Windows/Linux), re-run on a SIGNED PRODUCTION BUILD after the 167-05 hardening"
    expected: "On each of macOS, Windows, and Linux: enable the toggle, hide the window to tray (QuitGUIOnly-style, not full quit), drive a session into `waiting`, confirm exactly ONE OS-native notification appears identifying session name + agent type while the window is hidden. Repeat with the toggle OFF and confirm no notification appears. macOS must show real 'AgentHub' attribution (native UNUserNotificationCenter path, not a generic/Script-Editor label). This is the same M-41 item from the initial verification, now re-scoped to also confirm the 167-05 bundle-id-guard + @try/@catch hardening did not regress delivery on a real signed build (only the fail-safe, log-and-swallow, unbundled-process path was proven headlessly by 167-05's TestHasAppBundleID_FalseWhenUnbundled / TestSendNotification_NoBundleReturnsCleanly — the live delivery path itself still requires a human, on a signed build, per TESTING.md Category U)."
    why_human: "Real OS notification centers require a live desktop session on each of the three target OSes, on a signed/bundled production build (not `wails dev`, which is now deliberately a no-op path); no CI runner (build.yml) has one on any platform, and `go test` never pumps the macOS main dispatch queue so the async native path cannot be exercised headlessly even on darwin CI."
---

# Phase 167: Native Notifications Verification Report

**Phase Goal:** Deliver cross-platform native "session is waiting for input" notifications — daemon NotifyOnWaiting setting, cross-platform sendNotification (native macOS + beeep Win/Linux), GUI edge-detection trigger via tray poll, and a user-facing Settings Behavior toggle (default OFF).
**Verified:** 2026-07-01T12:35:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap-closure plan 167-05 (M-41 darwin crash hardening)

## Goal Achievement

### Re-Verification Focus: 167-05 M-41 Crash Hardening

The 167-05 gap-closure plan claimed to harden the darwin native notification path against the M-41 unbundled-process crash. This was independently re-verified against the actual code, not the SUMMARY narrative:

| Claim (167-05-SUMMARY.md) | Verification | Result |
|---|---|---|
| `hasValidBundleIdentifier()` C helper added in `tray_objc_darwin.m`, returns 1/0 based on `[[NSBundle mainBundle] bundleIdentifier]` | Read `tray_objc_darwin.m:161-163` directly | ✓ CONFIRMED — exact implementation present |
| `sendNotification` guards synchronously **before** `dispatch_async`, log-and-swallow (`NSLog` + `return`) when no valid bundle id | Read `tray_objc_darwin.m:177-181` | ✓ CONFIRMED — guard precedes `dispatch_async`, no dispatch reached when unbundled |
| `@try/@catch` wraps the `UNUserNotificationCenter` authorization + request-add blocks (defense in depth) | Read `tray_objc_darwin.m:186-208` | ✓ CONFIRMED — both nested blocks wrapped, `NSLog` on catch |
| Valid-bundle path unchanged (feature preserved) | Diff review — no logic removed inside the `hasValidBundleIdentifier()==1` branch, only wrapped in `@try` | ✓ CONFIRMED |
| `notification_darwin.go` exposes `hasAppBundleID()`, Go `sendNotification` wrapper early-returns with `log.Printf` when false, before any `C.CString` allocation | Read `notification_darwin.go:21-51` | ✓ CONFIRMED |
| `notification_darwin_test.go` created (`//go:build darwin`) with `TestHasAppBundleID_FalseWhenUnbundled` + `TestSendNotification_NoBundleReturnsCleanly` | Read the file directly | ✓ CONFIRMED — both tests present, correctly scoped |
| Both new tests pass on darwin | Ran `go clean -testcache && go test -race -short -run 'HasAppBundleID|SendNotification_NoBundle' . -v` (native arm64 darwin) | ✓ CONFIRMED — 2/2 PASS, exit 0 |
| `go build .` compiles cleanly (cgo) | Ran `go build .` natively (CGO_ENABLED=1) | ✓ CONFIRMED — clean |
| TESTING.md registration: Suite Manifest count bump, NTF-01 traceability row, Category U M-41 note update | Grepped TESTING.md for the relevant lines | ✓ CONFIRMED — 370→371 Go / 516→517 total note present; traceability row present (line 312); Category U M-41 note updated with the 167-05 hardening + signed-build re-run requirement (line 636) |
| `bash tests/check-traceability-paths.sh` passes | Ran directly | ✓ CONFIRMED — exit 0, "OK: all traceability paths exist" |
| Commits `f5caa4b4`, `cebf30eb` exist and match the claimed diffs | `git show --stat` on both | ✓ CONFIRMED — file sets and diffs match the SUMMARY's description exactly |

**Conclusion:** The 167-05 M-41 crash-hardening work is real, present, correctly wired, and independently verified — not a SUMMARY-only claim. This closes the code-level regression cleanly.

### Observable Truths (Full Phase Re-Check)

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | A non-waiting→`waiting` transition produces a real, visible native OS notification on macOS, Windows, Linux, including tray-hidden (NTF-01) | ⚠️ NEEDS HUMAN (code path present + wired + unit tested; real on-screen delivery unautomatable; now additionally hardened against the M-41 crash) | Unchanged from initial verification, plus 167-05 hardening confirmed above. `refreshTrayState` → `maybeNotifyWaiting` → `sendNotificationFunc` chain intact (app.go). |
| 2 | Fires exactly once per transition, no repeats while `waiting` (NTF-02) | ✓ VERIFIED | `TestMaybeNotifyWaiting` passes (re-ran: `go test . -run TestMaybeNotifyWaiting -v` → PASS) |
| 3 | Cold-start baseline: no notification burst for sessions already `waiting` at launch | ✓ VERIFIED | `TestMaybeNotifyWaiting_FirstTickNoNotify` passes |
| 4 | Notification text includes session name + agent type (NTF-03) | ✓ VERIFIED | `TestMaybeNotifyWaiting_BodyFormat`, `TestDisplayNameForCLI` pass |
| 5 | Settings toggle enables/disables, defaults off, suppresses all when off (NTF-04) | ✓ VERIFIED | `TestMaybeNotifyWaiting_DisabledNoop` passes; `SettingsTab.notify-toggle.test.tsx` 15/15 pass (re-ran directly) |
| 6 | `NotifyOnWaiting` persists to `settings.json` (0600), survives restart, defaults false, no schema bump | ✓ VERIFIED | `TestNotifyOnWaiting_Default/_Persists/_RoundTrip/_NoSchemaBump` all pass (re-ran) |
| 7 | `GET`/`PATCH /settings/notify-on-waiting` REST routes + `DaemonClient` round-trip | ✗ **FAILED (regression, re-verification finding)** | `go test -race -short -run 'NotifyOnWaiting' ./internal/daemon/ -v` → 3 of 4 `api_notify_test.go` tests **FAIL** on this checkout: `TestAPIGetNotifyOnWaiting_Default`, `TestAPIPatchNotifyOnWaiting_BadBody`, `TestDaemonClient_GetSetNotifyOnWaiting_RoundTrip` all report `got true, want false`. Root cause isolated and confirmed (see Gaps Summary): `testDaemon()`'s engine-reset block never resets `engine.notifyOnWaiting`, so the real on-disk `~/Library/Application Support/agenthub/settings.json` (currently `"notifyOnWaiting":true` on this machine) leaks into the test. This is a code-level test-isolation bug shipped by Phase 167-01, not an environment artifact — confirmed by removing/restoring the real settings file and observing the tests flip from pass to fail and back. |
| 8 | `sendNotification(identifier, title, body string)` signature identical across darwin/windows/linux; native macOS attribution retained; Win/Linux log-and-swallow | ✓ VERIFIED (unchanged, plus hardening) | All three files confirmed; darwin path now additionally bundle-id-guarded per 167-05 |
| 9 | Toggle physically under `#settings-behavior` (LOCKED correction) | ✓ VERIFIED | Unchanged — confirmed via file read |
| 10 | Settings search surfaces the toggle, targets `settings-behavior` | ✓ VERIFIED | Unchanged |
| 11 | M-41 + Category U + traceability registered in TESTING.md; `check-traceability-paths.sh` exits 0 | ✓ VERIFIED | Re-confirmed; now also includes the 167-05 hardening note and NTF-01 row for `notification_darwin_test.go` |

**Score:** 10/11 truths verified as before, **but truth #7's supporting test suite is currently reproducibly failing on this checkout** — a new regression discovered during re-verification that did not exist (or was not exposed) at the time of the initial 10/11 pass.

### Required Artifacts (delta from initial verification)

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `tray_objc_darwin.m` | `hasValidBundleIdentifier()` + guarded `sendNotification` + `@try/@catch` | ✓ VERIFIED | Confirmed by direct read; matches plan exactly |
| `notification_darwin.go` | `hasAppBundleID()` + guarded Go wrapper | ✓ VERIFIED | Confirmed by direct read |
| `notification_darwin_test.go` | 2 headless regression tests, darwin-tagged | ✓ VERIFIED | Present, both tests pass natively |
| `TESTING.md` | Manifest count bump, NTF-01 row, Category U update | ✓ VERIFIED | All three present |
| `internal/daemon/api_test.go` (`testDaemon()`) | Hermetic engine-state reset for every field `loadSettingsFromDisk` touches (self-documented obligation) | ✗ **STALE / INCOMPLETE** | Missing `engine.notifyOnWaiting = false` reset line — the one omission that breaks NTF-04's REST-route test hermeticity |

### Key Link Verification (delta)

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `tray_objc_darwin.m` `sendNotification` | `hasValidBundleIdentifier()` | synchronous guard before `dispatch_async` | ✓ WIRED | Confirmed: guard runs first, `return` on false, no dispatch reached |
| `notification_darwin.go` `sendNotification` | `hasAppBundleID()` (cgo) | early-return before `C.CString` | ✓ WIRED | Confirmed |
| `testDaemon()` (api_test.go) | `NewSessionEngine()` disk-load | reset block | ⚠️ **PARTIAL** | Resets 6 other fields but not `notifyOnWaiting`, so the real on-disk settings leak through for this one field |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Darwin bundle-id guard tests | `go clean -testcache && go test -race -short -run 'HasAppBundleID\|SendNotification_NoBundle' . -v` | 2/2 pass | ✓ PASS |
| Native darwin build | `go build .` (CGO_ENABLED=1, native arm64) | clean | ✓ PASS |
| Cross-compile windows | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...` | clean | ✓ PASS |
| Cross-compile linux | `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...` | clean | ✓ PASS |
| Traceability path checker | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist`, exit 0 | ✓ PASS |
| Full Go suite | `go test -race -short ./...` | **FAIL** — 3 tests fail in `internal/daemon` (see above); all other packages ok | ✗ **FAIL** |
| Notify-scoped Go tests, isolated | `go test -race -short -run 'NotifyOnWaiting' ./internal/daemon/ -v` | 3 FAIL, 4 PASS | ✗ **FAIL** |
| Root-cause isolation | moved real `~/Library/Application Support/agenthub/settings.json` aside, re-ran the same 3 failing tests | 4/4 PASS | Confirms root cause precisely |
| Root-cause isolation (restore) | restored the real settings.json, re-ran | 3/4 FAIL again (same 3) | Confirms reproducibility |
| App-layer trigger + display name | `go test . -run 'TestMaybeNotifyWaiting\|TestDisplayNameForCLI' -v` | 6/6 pass | ✓ PASS |
| Frontend toggle | `npx vitest run src/components/__tests__/SettingsTab.notify-toggle.test.tsx` | 15/15 pass | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| NTF-01 | 167-02, 167-03, 167-05 | Native OS notification on `waiting` transition, all 3 platforms, tray-hidden, now crash-hardened on darwin | ✓ SATISFIED (code); real delivery → human verification | Trigger wired; delivery primitives present and hardened; darwin crash regression closed |
| NTF-02 | 167-03 | Fires once per transition | ✓ SATISFIED | Tests pass |
| NTF-03 | 167-03 | Session name + agent type in body | ✓ SATISFIED | Tests pass |
| NTF-04 | 167-01, 167-03, 167-04 | Settings toggle, default off, persists, REST round-trip | ⚠️ **PARTIALLY SATISFIED** — daemon persistence layer (engine_notify_test.go) and frontend toggle are solid; the REST-route layer (api_notify_test.go / DaemonClient round-trip) is currently failing on this checkout due to the test-isolation gap above | `TestNotifyOnWaiting_*` (engine layer) pass; `TestAPI*NotifyOnWaiting*` (REST layer) fail |

No orphaned requirements — REQUIREMENTS.md maps exactly NTF-01..04 to Phase 167, all four covered across the five plans' `requirements` frontmatter fields (confirmed: 167-01→NTF-04, 167-02→NTF-01, 167-03→NTF-01/02/03/04, 167-04→NTF-04, 167-05→NTF-01/02/03/04).

### Anti-Patterns Found

None in the 167-05 changed files (`tray_objc_darwin.m`, `notification_darwin.go`, `notification_darwin_test.go`, `TESTING.md`) — scanned for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`, none found.

The `internal/daemon/api_test.go` test-isolation gap is not a debt-marker anti-pattern (no TODO/FIXME present) — it is a silent omission that violates the file's own explicitly documented contract ("Future engine fields require an addition here"). Classified as a 🛑 Blocker because it produces a currently-reproducible failing test run on this exact checkout, not a hypothetical future risk.

### Human Verification Required

### 1. M-41 — Cross-platform on-screen notification delivery, re-run on a SIGNED PRODUCTION BUILD

**Test:** On each of macOS, Windows, and Linux: enable the notify-on-waiting toggle in Settings → Behavior, hide the window to the system tray (QuitGUIOnly-style, not a full quit), drive a session into the `waiting` state, and observe the OS notification center.

**Expected:** Exactly ONE native OS notification appears identifying the session name and agent type while the window is hidden in the tray. macOS shows real "AgentHub" attribution (native `UNUserNotificationCenter`, not a generic label). Toggle OFF produces no notification. This re-run additionally confirms the 167-05 bundle-id-guard + `@try/@catch` hardening did not regress real delivery on a signed build (the hardening was only unit-proven for the fail-safe/unbundled path — `wails dev` is now deliberately inert).

**Why human:** Real OS notification centers require a live desktop session on a signed/bundled production build on each of the three target OSes; no CI runner has one; `go test` never pumps the macOS main dispatch queue, so the async delivery path cannot be exercised headlessly even on darwin CI.

### Gaps Summary

**Two distinct findings from this re-verification:**

1. **167-05's stated goal (M-41 darwin crash hardening) is fully and independently confirmed.** The bundle-id guard, `@try/@catch` defense-in-depth, headless regression test, and TESTING.md registration all exist exactly as claimed in 167-05-SUMMARY.md, verified by direct code read and by executing the tests/build myself (not by trusting the SUMMARY). This part of the re-verification passes cleanly.

2. **A new regression was discovered during full-suite re-verification, unrelated to 167-05's own changes:** `internal/daemon`'s `testDaemon()` helper (in `api_test.go`, added/extended across earlier Phase-167 plans, specifically 167-01) never resets `engine.notifyOnWaiting` after `NewSessionEngine()` unconditionally loads the real on-disk `settings.json`. The surrounding code comment in that exact reset block explicitly states "Future engine fields require an addition here OR a refactor" — and `notifyOnWaiting` was the one Phase-167 field never added. On this development machine, the real settings file currently has `"notifyOnWaiting":true` (very plausibly set via manual UAT of the toggle between the initial verification run and this one), which leaks into `api_notify_test.go`'s tests and causes 3 of 4 to fail with `got true, want false`. This was confirmed as a genuine, reproducible, machine-state-dependent bug — not a fluke of my environment — by moving the real settings file aside (all 4 tests pass) and restoring it (3 of 4 fail again, identically). This is a real code-quality gap in the phase's own test-isolation contract, and it means the previous verification's "8/8 pass" claim for this test group was contingent on a disk-state condition that no longer holds, and will not hold for any developer who has ever toggled the setting locally before running `go test ./...`.

The fix is a one-line addition (`engine.notifyOnWaiting = false` in the `testDaemon()` reset block) plus a re-run of the full daemon suite to confirm hermeticity. This does not require re-touching the darwin native path, the frontend, or the M-41 human-verification item — it is isolated to the test harness.

**Overall status: gaps_found.** Per the verification decision tree, a currently-failing regression test (rule 1) takes precedence over the routing of M-41 to human verification (rule 2), even though M-41 itself remains an open, unavoidably-human item exactly as before. Both should be addressed: close the `testDaemon()` reset gap (small, mechanical fix), then proceed to the M-41 live signed-build UAT before the phase can be marked `passed`.

---

_Verified: 2026-07-01T12:35:00Z_
_Verifier: Claude (gsd-verifier)_
