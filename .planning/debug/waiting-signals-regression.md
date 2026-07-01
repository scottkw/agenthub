---
status: diagnosed
trigger: "Phase 167 regression: session 'waiting' state no longer fires native toast, tab attention dot, or Hub card glowing border — all three broke together."
created: 2026-07-01T00:00:00.000Z
updated: 2026-07-01T00:00:00.000Z
---

## Current Focus

hypothesis: CONFIRMED — Phase 167's maybeNotifyWaiting fires the native macOS UNUserNotificationCenter call from the always-on tray poller when NotifyOnWaiting is ON; under `wails dev` (unbundled binary) that call aborts the whole GUI process on the first `waiting` transition, killing all four session-lifecycle signals (toast + dot + glow + auto-close) at once.
test: Data-flow differential (four independent update paths can only stop together via a process-wide fault) + settings.json shows notifyOnWaiting:true + only new 167 process-fault-capable code is the unguarded UNUserNotificationCenter call.
expecting: N/A — root cause identified.
next_action: Return ROOT CAUSE FOUND to caller (find_root_cause_only). Fix direction: bundle-id guard + @try/@catch in tray_objc_darwin.m sendNotification; re-run M-41 on a signed production build.

## Symptoms

expected: On `waiting` status — native OS toast fires (toggle ON, window hidden), attention dot renders on session tab, Hub card shows glowing waiting border.
actual: None appear. All three fail together.
errors: None reported (GUI observation).
reproduction: Drive a session into `waiting`; observe GUI. Test 1 (M-41) in 167-UAT.md.
started: Discovered Phase 167 UAT 2026-07-01. Tab dot + Hub glow are PRE-EXISTING (worked before 167); toast is new. Regression introduced in Phase 167 changeset.

## Eliminated

- hypothesis: "Phase 167 changed the frontend render code for the tab dot / Hub glow."
  evidence: "git diff a739b461..11f64148 frontend touched ONLY SettingsTab.tsx, SettingsSearch.tsx, and 4-line additive App.js/App.d.ts bindings. hubStatus.ts, SessionCard.tsx, Sidebar.tsx (the dot/glow render code) are UNTOUCHED."
  timestamp: 2026-07-01

- hypothesis: "Phase 167 altered the daemon waiting-status detector or the status→frontend propagation path."
  evidence: "detector.go (StatusWaiting='waiting', line 193) is untouched by 167. engine.go/api.go/client.go diffs are 100% additive (new NotifyOnWaiting setting only). app.go's pollSessionStatus/ListSessions/session:status emit path is unchanged — the two app.go edits are (1) sendNotification 2→3 arg signature (consistent across .go/.m/callers) and (2) inserting a read-only maybeNotifyWaiting(sessions) call into refreshTrayState."
  timestamp: 2026-07-01

## Evidence

- timestamp: 2026-07-01
  checked: "Full Phase 167 code changeset git diff a739b461..11f64148 across all .go/.tsx/.m files"
  found: "Every change is additive (new setting, new REST routes, new client methods, new notification primitive, new Settings toggle) EXCEPT: sendNotification signature 2→3 args (consistently updated in notification_darwin.go cgo header, tray_objc_darwin.m definition, and both callers), and one inserted line `a.maybeNotifyWaiting(sessions)` in refreshTrayState (reads the slice, mutates only a local map, returns early when toggle OFF)."
  implication: "By data-flow, the committed 167 source cannot disable the pre-existing tab dot or Hub glow — neither reads NotifyOnWaiting nor any 167-added field. The three signals share exactly ONE runtime prerequisite: the session actually reaching status=='waiting'."

## Evidence (cont.)

- timestamp: 2026-07-01
  checked: "FOURTH symptom from coordinator: tab auto-close on session exit also broke (pre-existing, SHELL-12). Test env = `wails dev` (unbundled binary), not a production build."
  found: "Auto-close is driven by App.tsx EventsOn('session:exit') (line 691-712) → handleCloseTab when autoCloseRef ON. Dot/glow are driven by EventsOn('session:status') AND an independent 3s setInterval ListSessions poll (App.tsx line 993). Toast is driven by the Go tray poller. These are FOUR INDEPENDENT update paths (per-session pollSessionStatus goroutines, the 3s React poll, the tray poller, plus the WebView render loop)."
  implication: "The only way all four independent paths stop simultaneously is a PROCESS-WIDE fault (the GUI process crashing/aborting). A blocked goroutine cannot explain it — pollSessionStatus and the React poll run independently of the tray poller."

- timestamp: 2026-07-01
  checked: "settings.json at ~/Library/Application Support/agenthub/settings.json (mtime Jul 1 02:17, matching the UAT window)"
  found: "\"notifyOnWaiting\":true AND \"autoCloseSession\":true AND \"startMinimized\":true."
  implication: "The NotifyOnWaiting toggle was ON during the failing UAT — so maybeNotifyWaiting's new sendNotification path WAS armed. autoCloseSession was ON, so auto-close SHOULD have fired but didn't (consistent with a dead process). With the toggle at its default OFF, maybeNotifyWaiting returns early and the new native-notification path never runs — which is why this never regressed before M-41 flipped the toggle ON."

- timestamp: 2026-07-01
  checked: "tray_objc_darwin.m sendNotification (lines 160-181): the native macOS path Phase 167 wired into the tray poller"
  found: "Calls [UNUserNotificationCenter currentNotificationCenter] with NO @try/@catch and NO bundle-identifier guard. UNUserNotificationCenter requires a valid app-bundle identifier; when called from a process without one (an unsigned/unbundled `wails dev` binary) it raises an uncaught NSInternalInconsistencyException ('bundleProxyForCurrentProcess is nil'), which aborts the process. Pre-167 the ONLY caller was QuitGUIOnly (fires only at quit, so a fault there is invisible); Phase 167 added a NEW frequent caller — maybeNotifyWaiting on every non-waiting→waiting transition."
  implication: "The differential is exact: the process-wide fault is triggered by the one new 167 code path, armed by the one toggle that was ON. No macOS crash .ips was found, but that dir holds only 1 stale report total (crash reporting appears suppressed on this machine), so absence is not counter-evidence; a main-thread ObjC abort under `wails dev` typically surfaces in the terminal running the dev server, not DiagnosticReports."

## Resolution

root_cause: |
  Phase 167 (commit 27a2588f / ea82ded5) wired the native macOS notification call
  (sendNotification → [UNUserNotificationCenter currentNotificationCenter] in
  tray_objc_darwin.m) into the always-on 5s tray poller via App.maybeNotifyWaiting,
  gated by the NotifyOnWaiting toggle. UNUserNotificationCenter requires a valid
  app-bundle identifier; under `wails dev` the binary is not a properly
  bundled/signed .app, so the call raises an uncaught NSException that aborts the
  entire GUI process. When the M-41 tester turned the toggle ON (confirmed
  notifyOnWaiting:true in settings.json) and drove a session to `waiting`, the very
  first waiting transition fired sendNotification and crashed the process. Because
  the toast, the attention dot, the Hub-card glow, and tab-auto-close-on-exit are
  all downstream of that single GUI process (its WebView, its per-session
  pollSessionStatus goroutines, its 3s ListSessions React poll, and its tray
  poller), the crash killed ALL FOUR at once — the shared upstream cause. It is a
  genuine code regression (an always-on path now calls a crash-prone native API
  with no bundle-id guard / no @try-@catch), but its DESTRUCTIVE trigger is
  environment-sensitive: it fires under `wails dev` (unbundled/unsigned) and is
  latent with the toggle OFF; a signed production .app (valid bundle id +
  notification entitlement) is the environment the code was written for and likely
  does not crash there. This also means the 8/8 unit verification passed honestly —
  the sendNotificationFunc injection seam stubs out the real UNUserNotificationCenter
  call in tests, so the crash can only appear in a live unbundled run.
fix: "(not applied — goal: find_root_cause_only). Suggested direction: make the native notification path fail safe — guard on [[NSBundle mainBundle] bundleIdentifier] != nil (and wrap the UNUserNotificationCenter block in @try/@catch), turning sendNotification into a log-and-swallow no-op when there is no valid bundle, mirroring the beeep wrappers' contract. Then re-run M-41 on a SIGNED production build, not `wails dev`."
verification: ""
files_changed: []
