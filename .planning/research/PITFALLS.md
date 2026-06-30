# Pitfalls Research — v4.2 Funnel Sharing & Polish

**Domain:** Adding Tailscale Funnel public sharing + cross-platform native notifications + Hub/web-share bug fixes to an existing Go/Wails desktop app (AgentHub)
**Researched:** 2026-06-30
**Confidence:** HIGH — all pitfalls grounded in direct code inspection of named files + project post-mortems documented in MEMORY.md

---

## Critical Pitfalls

### Pitfall 1: Funnel Teardown Incomplete on Non-Happy-Path Exits

**What goes wrong:**
`SetServeConfig` with `AllowFunnel` registers a node-level Tailscale serve config that persists in the Tailscale daemon independent of the AgentHub process. If AgentHub crashes, the daemon is killed via OS signal, the webserver is stopped but the Funnel teardown path is skipped (e.g., only the "user clicks disable" path calls `DisableFunnel`), or `onExit` for a session isn't wired to check `funnelSessions`, the Funnel remains active after the session is gone. The session's terminal is no longer connected, but the public Tailscale Funnel port is still open and responds to `https://hostname.ts.net/sessions/id?cap=...` — indefinitely, until the next `SetServeConfig` clears it or the user manually runs `tailscale serve reset`.

**Why it happens:**
The happy-path "user clicks disable Funnel toggle" is easy to remember. The three additional teardown sites are easy to miss: web-share-off (the existing `handleWebServe(id, false)` path must also check `funnelSessions`), session natural end (the `onExit` callback wired in `handleWebServerStart` / `AutoStartWebServer` must clear the session from `funnelSessions` and call `DisableFunnel()` if the map becomes empty), and daemon shutdown (a `defer ws.DisableFunnel()` or equivalent must fire in the webserver shutdown path). Developers test the toggle and think they're done.

**How to avoid:**
Enumerate all four teardown sites explicitly as checklist items in the phase plan:
1. `handleSetSessionFunnel(id, false)` — user disables toggle.
2. `handleWebServe(id, false)` — user disables web-share entirely for a session that had Funnel active; must remove `funnelSessions[id]` and call `DisableFunnel()` if map empty.
3. `onExit` callback (session death) — must mirror `handleWebServe` disable path.
4. `WebServer.Stop()` / daemon graceful shutdown — call `DisableFunnel()` unconditionally so the Tailscale serve config is cleared even if the process terminates cleanly.

For daemon crash (unclean exit): this cannot be fully prevented via code. Mitigate by documenting in the risk-acknowledgment dialog that Funnel is a Tailscale serve config, not an in-process state, and that users can clear a stuck config with `tailscale serve reset`. Do not promise teardown-on-crash in the UI.

Write a Go test that starts Funnel, kills the API handler mid-flight without calling Disable, and verifies `GetServeConfig` shows Funnel still active — then verify that `Stop()` calls `DisableFunnel()` after a restart.

**Warning signs:**
- UAT finds that after killing the daemon and restarting, `tailscale serve status` shows a lingering serve entry.
- A web-share-off test leaves `tailscale serve status` non-empty when it should be clear.
- The `onExit` path for session death is covered by a test that verifies `DisableFunnel` is called, but that test was skipped or mocked shallowly.

**Phase to address:**
The Funnel backend phase (first v4.2 phase covering `EnableFunnel`/`DisableFunnel`). The four teardown sites must be part of the success criteria, not an afterthought. Include a `tailscale serve status` shell check in UAT acceptance criteria.

---

### Pitfall 2: Public-Exposure Indicator Is Color-Only (Release-Blocking for Colorblind Owner)

**What goes wrong:**
A Funnel-active session is publicly accessible on the internet. The only persistent signal of this state is a visual indicator in `SessionShareModal` (and ideally on the Hub card). If that indicator communicates "internet-exposed" purely through color (e.g., a red/green dot, an orange badge), it is invisible to the project owner who is colorblind. This is a **release-blocking defect** per the established project norm. The same applies to the risk-acknowledgment dialog if its warning content is conveyed only through color cues.

**Why it happens:**
UI developers default to using color as the fastest signal for state. A red globe icon, an orange "LIVE" badge, or a warning dot — all are idiomatic but fail for colorblindness. The error is especially easy to make when adapting existing indicators (e.g., the web-share "WEB ON" green dot pattern) for Funnel.

**How to avoid:**
The Funnel "internet exposed" indicator must include ALL of:
- A persistent text label (e.g., "Internet Share ACTIVE", "FUNNEL ON") — not a tooltip, an always-visible label.
- A non-color icon with clear meaning (e.g., a globe icon + the text "Public", not just a colored dot).
- A distinctive shape or badge that differs from the tailnet-share state without relying on color.
- Verified against the actual hex constants in the component source, not by eye.

The risk-acknowledgment dialog must use bold/borders/icons for warnings, not color alone.

Check indicator accessibility at source level: read the CSS/JSX hex values and compare to the WCAG AA 4.5:1 requirement. Never eyeball it (per established project memory: "verify color-based UAT at source level, not by eye").

**Warning signs:**
- The PR diff for the Funnel indicator introduces a new CSS class that uses only `color`, `background-color`, or `border-color` to distinguish states.
- The risk-ack dialog uses `color: red` or similar color-only warning styling.
- No text label accompanies the state indicator — only an icon or dot.
- UAT for the indicator is described as "it turns orange when Funnel is on" — this is the wrong UAT criterion.

**Phase to address:**
The Funnel frontend phase (`SessionShareModal.tsx` Funnel toggle + risk dialog + persistent indicator). Add "verify indicator is not color-only" as an explicit acceptance criterion. Include Hub card indicator if the design adds one.

---

### Pitfall 3: Origin/BaseURL Funnel-Awareness Missing When Funnel Goes Live (Integration Landmine)

**What goes wrong:**
Funnel exposes the webserver at port 443 (no port in URL). The browser's `Origin` header is `https://hostname.ts.net`. The existing `requireAllowedOrigin` in `origin_mw.go` does a byte-for-byte match against `ws.BaseURL()`, which returns `https://hostname.ts.net:7443`. These do not match. Every Funnel guest gets a 403 response before the capability token is ever checked. The symptom looks like a cap token bug or an auth issue, not an Origin mismatch.

The same landmine hits two additional sites: `capability_mw.go`'s `originAllowedForWrite()` (write-capable web guests also fail), and the share URL builders in `api.go` (`issueCapabilitiesForSession` and `handleExchangeJoinCode`) which emit `:7443` URLs to Funnel guests — meaning the recipient's link opens the wrong host:port.

**Why it happens:**
The `BaseURL()` abstraction was designed for a single-URL world (tailnet FQDN + local-fallback). Funnel introduces a second valid origin on the same server. Developers who add `EnableFunnel()` and test that the serve config is applied correctly assume the URL routing will work — but the Origin check fires before routing and silently blocks all guests. The bug is hard to notice because `tailscale serve status` shows Funnel as active, the cert is valid, and the route resolves — but guests 403.

**How to avoid:**
The ARCHITECTURE.md dependency-ordering constraint is the key rule: **steps 1-4 (LocalClient on WebServer struct, EnableFunnel/DisableFunnel/FunnelBaseURL methods, dual Origin allowlist update, share URL builder update) must all land in the same phase as the Funnel toggle feature.** A partial deployment where `EnableFunnel()` works but Origin check is not updated causes silent 403s.

Implementation: keep `BaseURL()` returning the tailnet URL unchanged (it's used for local GUI display, tray, QR, settings copy). Add `FunnelBaseURL()` returning `https://hostname.ts.net` (no port). Update `requireAllowedOrigin` to check both. Update `allowedOrigins()` to return both. Update `originAllowedForWrite()` to use dual check. Update both URL builder sites to use `funnelBase` when `funnelSessions[sessionID]` is true.

UAT acceptance: with Funnel enabled, open the Funnel URL (`https://hostname.ts.net/sessions/id?cap=TOKEN`) from an **external machine** (not the tailnet). Confirm 200, not 403. Confirm the join-code flow produces a URL with no port. These are the exact surfaces that are silently broken if this pitfall occurs.

**Warning signs:**
- Web-share flow tested only from a tailnet device (Phase 155 false-parity pattern — this project has been bitten by verifying parity on the wrong surface).
- `requireAllowedOrigin` test only covers the existing `BaseURL()` form.
- Share URL is copied from the share modal and tested by clicking it, but the machine doing the test is still on the tailnet (Funnel 403 would not trigger because the tailnet URL also works).

**Phase to address:**
Same phase as the Funnel backend (the first v4.2 Funnel phase). This is not a separate "cleanup" phase — it must be atomic with Funnel enable. Verify on an external machine that is not on the tailnet.

---

### Pitfall 4: ETag Concurrency Clobber on ServeConfig Read-Modify-Write

**What goes wrong:**
`GetServeConfig` returns the current `ServeConfig` with an `ETag`. `SetServeConfig` must send this `ETag` as `If-Match` (the local SDK handles this automatically if the `ETag` field on the struct is preserved). If the code re-constructs a `ServeConfig` from scratch rather than modifying the returned struct, the `ETag` is lost. If another process (e.g., the user running `tailscale serve` CLI concurrently) modifies the config between the `Get` and the `Set`, the write clobbers the other change. The `ETag` field on `ipn.ServeConfig` is tagged `json:"-"` (not serialized), so it's invisible in JSON logging and debug output — developers lose it without realizing it.

**Why it happens:**
Developers constructing a new `ServeConfig{}` literal in `EnableFunnel` and `DisableFunnel` instead of calling `GetServeConfig` first and modifying the returned struct will silently omit the ETag. The omission is invisible until a concurrent modification triggers a 412 or a clobber.

**How to avoid:**
Always call `ws.lc.GetServeConfig(ctx)` immediately before each `ws.lc.SetServeConfig(ctx, sc)` call. Never cache the `ServeConfig` struct between enable and disable calls. Modify the returned struct in place (nil-guard it and use a new struct only when the returned value is nil). Guard `EnableFunnel`/`DisableFunnel` with a mutex so they cannot run concurrently. The STACK.md reference implementation shows the correct pattern.

**Warning signs:**
- `EnableFunnel` or `DisableFunnel` constructs `&ipn.ServeConfig{...}` with literal field values rather than calling `GetServeConfig` first.
- Error log shows "412 Precondition Failed" from `SetServeConfig`.
- Tests mock `GetServeConfig` to return nil — this is valid for the "first time" case but the test must also cover the "existing config present" case.

**Phase to address:**
Funnel backend phase. Include a test case for `EnableFunnel` called when a non-empty serve config already exists.

---

### Pitfall 5: No Prerequisite Check Before SetServeConfig (Opaque Failure Surfaced to User)

**What goes wrong:**
`SetServeConfig` with `AllowFunnel` fails if the tailnet hasn't enabled Funnel (missing `nodeAttrs→funnel`), HTTPS/MagicDNS is not enabled, or the requested port isn't in the policy-allowed set. The raw error from `SetServeConfig` is not human-readable. Showing it to the user provides no actionable guidance. The Funnel toggle silently resets to off with a generic "error enabling Funnel" message.

**Why it happens:**
`SetServeConfig` is the authoritative path, so developers call it and handle the error. They don't discover `ipn.CheckFunnelAccess(port, st.Self)` — a separate pre-flight function that returns exact, human-readable strings — until they encounter the opaque error in testing.

**How to avoid:**
Call `ipn.CheckFunnelAccess(funnelPort, st.Self)` before calling `SetServeConfig`. Surface the returned error string verbatim in the Funnel toggle UI:
- `"Funnel not available; HTTPS must be enabled. See https://tailscale.com/s/https."`
- `"Funnel not available; \"funnel\" node attribute not set. See https://tailscale.com/s/no-funnel."`
- `"port N is not allowed for funnel; allowed ports are: 443,8443,10000"`

The risk-acknowledgment dialog should only appear after `CheckFunnelAccess` passes — do not ask users to accept risk and then immediately fail.

**Warning signs:**
- `EnableFunnel()` calls `SetServeConfig` directly without a prior `CheckFunnelAccess` call.
- UAT is only performed on a tailnet that already has Funnel enabled — never tested on a tailnet without `funnel` nodeAttr.

**Phase to address:**
Funnel backend phase. Include an explicit test where `CheckFunnelAccess` returns an error; verify the error is surfaced in the response and that `SetServeConfig` is never called.

---

### Pitfall 6: Funnel Cert Provisioning Latency Surfaced as a Broken Feature

**What goes wrong:**
Tailscale Funnel requires a valid Let's Encrypt cert for the hostname. If the cert has never been provisioned (first-time Funnel user), the first few requests to `https://hostname.ts.net` may return TLS errors or 502 responses while the cert is being obtained (a 5-30 second window). Users who enable Funnel, immediately copy the URL, and send it to a recipient may give them a link that appears broken.

**Why it happens:**
The existing `GetCertificate` hook on AgentHub's webserver handles cert retrieval for the tailnet address and works instantly because the cert is already cached. Funnel goes through a second TLS termination at Tailscale's edge, which may need to provision the cert. There is no "cert is ready" signal from the API.

**How to avoid:**
After `SetServeConfig` succeeds, do NOT display the Funnel URL as immediately shareable. Show a "Funnel is starting up — allow up to 30 seconds for the first connection" state in the UI. Optionally probe `https://hostname.ts.net` with a short timeout and switch to "ready" when the probe succeeds. The risk-ack dialog text should mention this warmup latency.

**Warning signs:**
- UAT tests the Funnel URL "works" after a manual wait but the immediate-copy flow was never tested.
- The probe is tested only against a Funnel that was already active (cert already warm).
- No "warming up" UX state exists in `SessionShareModal` after `EnableFunnel` succeeds.

**Phase to address:**
Funnel frontend phase. The "starting up" UX state must be in the design spec, not retrofitted.

---

### Pitfall 7: Notification Spam — Firing on State Rather Than Transition Into State

**What goes wrong:**
The `waiting` status is persistent — a session remains in `waiting` state until the user inputs a response. The status polling loop in `app.go:pollSessionStatus` runs every 500ms. If the notification fires whenever `status == "waiting"`, the user receives a notification every 500ms (or every de-dup window of 60 seconds) for the duration of the waiting state — potentially dozens of notifications per session.

**Why it happens:**
The polling loop checks current state. The natural condition `if s.Status == "waiting"` is wrong; the correct condition is `if s.Status == "waiting" && last != "waiting"` (transition into waiting). Developers writing the polling hook add the notification call without considering that `waiting` is a sustained state.

**How to avoid:**
The notification must fire on the **transition** into `waiting`, not while in the `waiting` state. The `last` variable in `pollSessionStatus` already tracks the previous status. Correct check:

```
if s.Status != last {
    last = s.Status
    if s.Status == "waiting" {
        maybeNotifyWaiting(sessionID, name)
    }
}
```

The de-dup `notifiedWaiting` map with a 60-second window is belt-and-suspenders only. The transition check is the primary mechanism. The Settings toggle `NotifyOnWaiting` must default to **off**.

**Warning signs:**
- The condition in `pollSessionStatus` does not check `s.Status != last` before firing the notification.
- A test runs the notification path against a session already in `waiting` state (rather than a `running → waiting` transition).
- UAT for notifications runs for only a few seconds; it doesn't verify that no second notification fires after 10 minutes of sustained `waiting`.

**Phase to address:**
Notifications phase (#110). Include a test that simulates `running → waiting → waiting × 5` (sustained) and verifies notification fires exactly once.

---

### Pitfall 8: beeep on macOS Shows "Script Editor" as Notification Sender

**What goes wrong:**
`beeep.Notify()` on macOS uses `osascript` as its primary implementation. macOS attributes the notification to the `osascript` process, displayed in Notification Center as "Script Editor". Users see "Script Editor — [session] is awaiting input" with no apparent connection to AgentHub.

**Why it happens:**
AgentHub runs as `LSUIElement` (no Dock icon). The proper macOS notification path for background apps is `UNUserNotificationCenter` via CGO, which shows "AgentHub" as the sender but requires an entitlement, permission dialog, and signing changes. `beeep` uses `osascript` to avoid this complexity.

**How to avoid:**
Accept this trade-off for v4.2 — per STACK.md this is explicitly documented as acceptable. Do NOT attempt to switch to `UNUserNotificationCenter` in v4.2. Do NOT conditionally detect and use `terminal-notifier` (third-party binary, creates maintenance burden). Mitigate by putting the app name in the notification body: `title = "AgentHub"`, `message = "[sessionname] is awaiting input"`. This is what the ARCHITECTURE.md reference implementation already shows. If branded attribution matters in a future milestone, open a GitHub issue for the CGO path then.

**Warning signs:**
- A developer modifies `notification_darwin.go` to shell out to `terminal-notifier` conditionally.
- The notification test asserts the sender name is "AgentHub".
- The PR adds `UNUserNotificationCenter` CGO code without corresponding entitlement changes.

**Phase to address:**
Notifications phase (#110). Document the "Script Editor" attribution in release notes.

---

### Pitfall 9: Linux Notifications Fail Silently on Headless Installs — Tests Break on CI

**What goes wrong:**
`beeep` on Linux uses `org.freedesktop.Notifications` via D-Bus. On headless servers, CI runners (GitHub Actions ubuntu-latest), or SSH sessions, there is no D-Bus notification daemon. `beeep.Notify()` returns an error. If a notification test asserts that `beeep.Notify` returns nil, it fails on CI. If the error is propagated rather than swallowed, it disrupts the status polling loop.

**Why it happens:**
Developers test on a desktop Linux environment where D-Bus is running. CI is headless.

**How to avoid:**
The `sendNotification` platform wrapper must treat errors from `beeep.Notify` as non-fatal — log at DEBUG level, do not propagate. Tests of the notification feature must mock or stub the `sendNotification` call; test at the `maybeNotifyWaiting` logic layer (de-dup, transition detection, settings toggle), not at the `beeep.Notify` call site.

**Warning signs:**
- A test in the notifications package calls `beeep.Notify` directly and asserts on the error return.
- CI Linux notification tests pass locally but fail in GitHub Actions.
- `notification_linux.go` returns `beeep.Notify(...)` directly (returning the error) rather than swallowing it.

**Phase to address:**
Notifications phase (#110). CI gate includes `go test ./...` on the Linux runner.

---

### Pitfall 10: #117 Fix Ships Only the Buffer Increase, Not the Viewer-Disconnect UI

**What goes wrong:**
Increasing the relay Hub subscriber buffer from 256 to 1024 reduces kick-on-join frequency but does not eliminate it. Any background-tabbed browser can stop draining its WebSocket during a PTY-intensive output burst, and an arbitrarily large buffer will eventually fill. Without the viewer-disconnect UI (Part B of the #117 fix — a viewer list with a Disconnect button in `SessionShareModal`), there is no escape hatch when a viewer is stuck. The issue recurs later, reported as a new bug identical to #117.

**Why it happens:**
The buffer increase is a one-line diff. The viewer-disconnect UI requires a new Wails bound method, a new daemon endpoint (`DELETE /sessions/{id}/viewers/{personKey}`), a new `Hub.KickPersonKey` method, and frontend work. Under schedule pressure, Part A ships and Part B is deferred — the same "half-fix" pattern seen in prior milestones.

**How to avoid:**
Treat #117 as a two-part fix that ships in the same phase. Phase acceptance criteria must explicitly verify: (1) two viewers connected simultaneously; (2) one viewer's tab is backgrounded; (3) the other viewer sees correct viewer count; (4) an admin can disconnect the stuck viewer via the Share modal Disconnect button. Also verify that the Hub card's viewer count updates within the next poll cycle after `CloseSlow` fires (the async gap between `CloseSlow` and `hub.Unsubscribe` can leave a stale count).

**Warning signs:**
- The #117 PR diff shows only `hub.go` (buffer change) with no `SessionShareModal.tsx`, `app.go` (KickSessionViewer), or daemon endpoint changes.
- UAT tests "second viewer joins" but not "disconnect stuck viewer via Share modal".
- The viewer list UI shows viewers but the Disconnect button is absent or no-ops.

**Phase to address:**
Bug-fix phase (#117). Both parts must be in the acceptance criteria of the same phase.

---

### Pitfall 11: #112 Fix Introduces CSP Violation or Wrong apiBaseURL in Web Guests

**What goes wrong:**
The fix for #112 adds a `useEffect` in `WebShareSessionView.tsx` that creates a `new EventSource(url)` where `apiBaseURL = window.location.origin`. If a developer instead sets `apiBaseURL` to the `baseURL` prop (introduced by the `#118` fix for remote-open), the EventSource URL in a local web guest session will be wrong — pointing at the remote peer's URL rather than the serving host. This breaks plugin-config streaming.

Separately: if the `EventSource` URL does not match the `connect-src` directive in the CSP, the browser blocks it with a CSP violation (visible in DevTools as a console error, but the terminal continues to work, making the bug easy to miss).

**Why it happens:**
`#112` and `#118` both modify `WebShareSessionView.tsx` and introduce different `baseURL` concepts. Conflating them — using the remote `baseURL` prop as `apiBaseURL` for EventSource — is a natural merge mistake. CSP violations are silent from the server's perspective (no server log) and may not appear in the terminal behavior.

**How to avoid:**
Keep `apiBaseURL` for REST/EventSource calls strictly as `window.location.origin` — the origin of the page the browser loaded. The `baseURL` prop is exclusively for WS URL construction when opening a remote session. `apiBaseURL = baseURL ?? window.location.origin` is the correct pattern (per ARCHITECTURE.md) only when `WebShareSessionView` is loaded from the serving host and `baseURL` is the same origin. For the `#118` remote-open case, the component runs inside the Wails WebView where Wails RPC (not EventSource) provides plugin config.

Add a Wails-context guard: skip the EventSource path when running inside the Wails WebView (`typeof window.__wails !== 'undefined'` or equivalent). This prevents the EventSource from firing at all in the desktop app context, where the Wails event already provides plugin config.

UAT: open the Funnel URL in a real browser (not Wails WebView), open DevTools Network panel, verify `/api/plugin-config` returns 200 and `/api/plugin-config/stream` is an open EventSource. Check Console panel for CSP errors.

**Warning signs:**
- `apiBaseURL` in `WebShareSessionView` is set to the `baseURL` prop when it's defined.
- The `#112` fix is tested only in the Wails desktop app (where the EventSource path is never taken because `pluginConfig != null`).
- Browser DevTools shows `EventSource refused to connect` or a CSP `connect-src` error.

**Phase to address:**
Bug-fix phase (#112). UAT must use a real browser connected to the web-share URL.

---

### Pitfall 12: #118 Fix Breaks the window.location.host WS URL Assumption for Remote Sessions

**What goes wrong:**
`handleOpenRemoteSession` currently calls `BrowserOpenURL(url)`. The fix opens an in-app `__websession__` tab using `WebShareSessionView` with a `baseURL` prop set to the remote peer's URL. The WS URL construction must change from `wss://${window.location.host}/...` (the Wails WebView's host — the local server) to `wss://<remotePeerFQDN>:7443/...`. If `baseURL` is wired only for the WS URL but all `fetch` and `EventSource` calls in the component still use `window.location.origin`, those calls hit the local server (which doesn't have the remote session) and fail silently.

**Why it happens:**
Adding `baseURL` to fix the WS URL is a targeted change. Auditing every other network call in `WebShareSessionView` for the same fix is a broader change that requires understanding the full URL topology of the component in both contexts (Wails, local web guest, remote in-app tab).

**How to avoid:**
When `baseURL` is provided to `WebShareSessionView`, it must be used consistently as the base for ALL outgoing network calls from that component — WS URL, REST calls (`fetch`), EventSource — not just the WS URL. `apiBaseURL = baseURL ?? window.location.origin` is the correct pattern. Audit every `fetch`, `EventSource`, and `WebSocket` constructor in `WebShareSessionView` when adding `baseURL`.

Note: the remote cap-exchange is orchestrated by `handleOpenRemoteSession` in `App.tsx` before the component mounts — by the time `WebShareSessionView` renders, the cap token is already resolved and passed as a prop. The component itself does not need to perform the cap exchange. Confirm this is the case; if not, the cap exchange fetch must also use `apiBaseURL`.

UAT requires a real two-machine Tailscale setup (per established "two-machine tailnet" UAT precedent in v3.4 Phase 122).

**Warning signs:**
- Terminal relay works in the in-app remote tab but plugin config or chat features do not load.
- The `fetch` for plugin config uses `window.location.origin` when `baseURL` is defined.
- Remote session tests are only performed by connecting to a local session via the in-app tab path (not a real remote peer).

**Phase to address:**
Bug-fix phase (#118). Two-machine Tailscale UAT is required — do not accept a single-machine test.

---

### Pitfall 13: Tests Encoding the Same Wrong Assumption (Green CI on Broken Feature)

**What goes wrong:**
A test verifies that Funnel teardown fires on session end. The test mocks `DisableFunnel()` and asserts it was called when the `onExit` callback fires. But the test starts the session via a code path that doesn't wire `onExit` to the `funnelSessions` check. The test passes (mock is called via a direct call path), but the production `onExit` wiring is absent. CI is green; the feature is broken.

This is the exact failure pattern from Phase 150: the shell-warning gate matched bare agent names; test fixtures used bare names; CI passed; the real CLI uses `/bin/zsh` (fully qualified); the warning never fired in production.

Similarly for the Origin allowlist: a test that constructs a `WebServer` with `funnelActive = true` set directly (not via `EnableFunnel()`) can assert the dual-origin check works while the `EnableFunnel` → `funnelActive` wiring is broken.

**Why it happens:**
Unit tests at the function level are easy to write and fast. They don't test the wiring between functions. The same wrong assumption that exists in production code is easy to replicate in the test because the developer who writes both is reasoning from the same (incorrect) mental model.

**How to avoid:**
For Funnel teardown: write an integration test that goes through `handleSetSessionFunnel(id, true)` (enabling Funnel via the daemon API handler, not direct struct mutation), fires the `onExit` callback through the real session-end code path, and verifies that `GetServeConfig` returns an empty (or nil) serve config.

For Origin allowlist: write a test that calls `ws.EnableFunnel()` (not by setting `ws.funnelActive = true` directly), then issues an HTTP request with the Funnel origin header, and verifies 200.

Per project memory: "drive live with real daemon data" for features involving callback wiring and status heuristics.

**Warning signs:**
- Test setup sets `ws.funnelActive = true` directly instead of going through `ws.EnableFunnel()`.
- Teardown test asserts `mockDisableFunnel.called == true` but does not verify actual serve config state via `GetServeConfig`.
- All Funnel tests are unit tests; no integration test exercises the full IPC chain (frontend → Wails bound method → daemon API → webserver → Tailscale LocalClient).

**Phase to address:**
Every Funnel phase. Add to acceptance criteria: "integration tests use real code paths, not direct struct mutation." Run `bash tests/check-traceability-paths.sh` before closing the phase.

---

### Pitfall 14: Local-Network-Fallback Path Broken by Funnel Changes

**What goes wrong:**
AgentHub has a local-network fallback mode (no Tailscale) serving with self-signed TLS and HTTP Basic Auth. When Funnel code calls `ws.lc.StatusWithoutPeers(ctx)`, the `LocalClient` is zero-value usable but `StatusWithoutPeers` returns an error when the Tailscale daemon is not running. If `EnableFunnel` propagates this error in a way that disrupts the webserver's normal operation, fallback-mode users are broken. Additionally, if `ws.funnelActive` is ever incorrectly set in fallback mode, `requireAllowedOrigin` looks for an origin that can never arrive, breaking all web-share in fallback mode.

**Why it happens:**
Funnel code is developed and tested only on machines with Tailscale running. The fallback path is rarely tested after initial implementation. Changes to `WebServer` struct state can inadvertently affect the fallback path.

**How to avoid:**
`EnableFunnel` must return an explicit, user-surfaced error ("Tailscale is not connected — Funnel requires Tailscale") if `StatusWithoutPeers` fails, and must not modify `ws.funnelActive`. The Funnel toggle in `SessionShareModal` must be disabled/hidden when the Tailscale health check is not in the "Connected" state. `ws.funnelActive` must only ever be set to `true` inside `EnableFunnel` after a successful `SetServeConfig`. Include a CI test that initializes the webserver with a mock `LocalClient` that errors on `StatusWithoutPeers`; call `EnableFunnel`; verify `funnelActive` remains false.

**Warning signs:**
- Funnel toggle is enabled in the UI even when Tailscale health check shows "Not Connected".
- A test of the fallback mode webserver panics after the `local.Client` field promotion.
- `FunnelBaseURL()` returns a non-empty string in fallback mode.

**Phase to address:**
Funnel backend phase. Verify fallback mode explicitly: start the daemon with Tailscale not running, confirm web-share works (fallback), confirm Funnel toggle is disabled.

---

### Pitfall 15: `prefers-reduced-motion` Regression in Funnel Indicator or Risk Dialog

**What goes wrong:**
New UI components for the Funnel active indicator or the risk-acknowledgment dialog add entrance animations (fade-in, slide-in) or pulsing attention effects that are not gated on `prefers-reduced-motion: reduce`. This violates the established project accessibility norm (colorblind owner; reduced-motion is a release norm per PROJECT.md) and is a release-blocking defect.

**Why it happens:**
New components added in a hurry reuse animation patterns from the codebase without checking whether they respect `prefers-reduced-motion`. The Hub card "attention pulse" is already gated on `@media (prefers-reduced-motion: reduce)`. New components that add their own animations may not replicate this gate.

**How to avoid:**
Any CSS animation or transition added in v4.2 UI components must include:
```css
@media (prefers-reduced-motion: reduce) {
  animation: none;
  transition: none;
}
```
Verify this in the PR diff for: the risk-acknowledgment dialog modal entrance, the Funnel-active indicator (if it pulses), and any "warming up" spinner. Run `grep -r 'prefers-reduced-motion'` on new CSS files to verify coverage before accepting the phase.

**Warning signs:**
- New CSS files for Funnel components do not contain `prefers-reduced-motion`.
- The Funnel indicator uses the same attention-pulse class as Hub cards without overriding it in a reduced-motion context.

**Phase to address:**
Funnel frontend phase. Add `prefers-reduced-motion` verification to the accessibility acceptance criteria alongside the colorblind-safe check.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Buffer increase 256→1024 without viewer-disconnect UI | One-line fix, ships fast | Kick-on-join recurs under load; no user escape hatch | Never — Part B (viewer list + Disconnect) must ship in the same phase |
| beeep "Script Editor" attribution on macOS | Zero config, no entitlements | Users confused by "Script Editor" sender | Acceptable for v4.2; revisit if user feedback demands branded attribution |
| Not probing Funnel cert warmup before showing URL | URL shown instantly after SetServeConfig | Recipients hit TLS errors for up to 30 seconds on first Funnel use | Acceptable only if a "warming up" UX state is shown; not if URL displayed as immediately ready |
| Mocking DisableFunnel in teardown tests instead of asserting real serve config state | Tests faster to write | Tests encode the same wrong assumption; green CI on broken teardown (project has been bitten by this) | Never — Funnel teardown is a security property |
| Defaulting NotifyOnWaiting to ON | Feature visible immediately | Unexpected notification spam; users disable it before reading | Never — always default new notification features to OFF |
| Testing Origin allowlist with direct `ws.funnelActive = true` instead of via `EnableFunnel()` | Faster unit test | Misses `EnableFunnel` → state wiring bugs; CI green on broken production path | Never for the Origin allowlist test; acceptable only for pure middleware unit tests with explicit comment |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Tailscale ServeConfig | Construct a new `&ipn.ServeConfig{}` for every `SetServeConfig` call | Always `GetServeConfig` first; modify the returned struct; preserve the ETag field |
| Tailscale Funnel prerequisites | Call `SetServeConfig` and handle the error generically | Call `ipn.CheckFunnelAccess(port, st.Self)` first; surface the exact returned error string verbatim |
| beeep on Linux headless | Assert `beeep.Notify` returns nil in tests | Test notification logic with mocked `sendNotification`; treat beeep errors as non-fatal |
| `requireAllowedOrigin` with dual origins | Test only the tailnet origin path after adding Funnel support | Test both `ws.BaseURL()` and `ws.FunnelBaseURL()` paths; verify unknown origin is still rejected |
| `WebShareSessionView` in Wails vs. browser | Use `window.location.origin` for all URLs in both contexts | Guard the EventSource/REST path with a Wails-context check; EventSource is browser-context only |
| Funnel port 443 URL construction | Include `:443` in the Funnel base URL | Port 443 is implicit in HTTPS; `FunnelBaseURL()` must return `https://hostname` (no port) |
| `onExit` callback wiring | Wire `onExit` only in one session-start path | Audit all paths that start web-serving; every path that calls `EnableFunnel` needs an `onExit` that removes the session from `funnelSessions` |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| No risk-acknowledgment dialog before enabling Funnel | User exposes session to public internet without understanding the risk | Require explicit acknowledgment (checkbox + confirm button, not just a tooltip) before `EnableFunnel` IPC call is sent; do not skip this if Funnel was previously active |
| Join code TTL too long on a public Funnel endpoint | A leaked or forwarded join code can be reused by anyone on the internet | Recommend short-lived join codes (15-30 min) in the risk-ack dialog; consider defaulting join-code TTL shorter when Funnel mode is active |
| Write/file caps enabled by default on Funnel share | An internet user can submit PTY input or read/write files | Funnel share must default to read-only cap tokens; risk-ack dialog must explicitly warn that write caps grant terminal input to internet users |
| Funnel hostname logged at INFO level | Structured logs sent to aggregators expose the public URL of every Funnel session | Log `ws.funnelBaseURL` and Funnel URLs at DEBUG level only; redact in structured logs |
| Concurrent `EnableFunnel` / `DisableFunnel` without mutex | Two concurrent calls produce an ETag clobber or inconsistent `funnelSessions` state | Guard both methods with a mutex on the `WebServer`; do not allow concurrent calls |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Funnel URL shown as immediately shareable after SetServeConfig | Recipient gets TLS error for up to 30 seconds on first Funnel use | Show "Funnel starting up..." state; probe or add a brief delay before marking the URL as ready |
| Risk-ack dialog shown on every Funnel toggle re-enable | User must re-acknowledge risk every time they briefly disable and re-enable | Show the risk-ack dialog only on the first enable per session; subsequent enables use a lighter confirmation |
| Funnel indicator only visible inside the Share modal | User forgets a session is internet-exposed after closing the modal | Show a persistent indicator on the Hub session card and/or the tab status bar while Funnel is active |
| `NotifyOnWaiting` defaults to ON | User is surprised by unexpected desktop notifications; may disable before investigating | Default OFF; introduce the feature with a discoverable opt-in notice on first `waiting` event |
| `#115` Footer "Share Session" button opens modal for stale session | If the active session changes between Footer render and button click, the modal opens for the wrong session | Wire the Footer button to the currently-active session at click time, not at render time; confirm session ID at handler entry |

## "Looks Done But Isn't" Checklist

- [ ] **Funnel teardown — all four paths:** Verify `tailscale serve status` is empty after (a) user disables toggle, (b) user disables web-share, (c) session exits naturally, (d) daemon is cleanly stopped.
- [ ] **Origin allowlist — Funnel origin:** With Funnel enabled, issue an HTTP request with `Origin: https://hostname.ts.net` (no port) and verify 200, not 403. Use a real HTTP client, not a unit test bypassing middleware.
- [ ] **Share URL no-port form:** Verify the Funnel URL emitted by `issueCapabilitiesForSession` and `handleExchangeJoinCode` has no `:443` suffix (`https://hostname.ts.net/sessions/id?cap=TOKEN`).
- [ ] **Colorblind-safe indicator:** Verify the Funnel-active indicator has a text label that distinguishes the state without color. Read hex values in source; do not verify by eye. Confirm `prefers-reduced-motion` is gated on all animations.
- [ ] **Notification transition:** Verify notification fires exactly once when a session transitions `running → waiting`. Verify it does NOT fire again during sustained `waiting` state (leave a session waiting for 2 minutes).
- [ ] **Local fallback unaffected:** Start the daemon with Tailscale not running; verify web-share (fallback mode) serves correctly; verify Funnel toggle is disabled/hidden; verify `funnelActive` remains false.
- [ ] **#117 both parts:** Two simultaneous viewers connected; one tabs away; other sees correct viewer count; Disconnect button in Share modal terminates the stuck viewer; Hub viewer count updates within the next poll cycle.
- [ ] **#112 real browser:** Open a web-share URL in a real browser (not Wails WebView). Open DevTools Network panel — confirm `/api/plugin-config` and `/api/plugin-config/stream` return 200. Check Console panel for CSP errors.
- [ ] **#118 two machines:** On a two-machine Tailscale setup, open a remote session from the Hub; verify it opens an in-app tab (not an external browser window); verify the terminal relay streams correctly; verify plugin config loads in the in-app tab.
- [ ] **Join code TTL reminder:** Verify the risk-ack dialog mentions join code TTL and recommends a short value for Funnel shares.
- [ ] **TESTING.md updated:** New test files for Funnel, notifications, and bug fixes are in TESTING.md Section 2 (suite manifest) and Section 4 (traceability map). Run `bash tests/check-traceability-paths.sh`.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Funnel serve config left active after daemon crash | LOW | User runs `tailscale serve reset` in terminal. Document this command in the risk-ack dialog. |
| Origin 403 on Funnel URL (Origin allowlist not updated) | MEDIUM | Ship a patch release updating `requireAllowedOrigin` to include `FunnelBaseURL()`; disable Funnel toggle in UI until patch ships. |
| Notification spam (fires on state, not transition) | LOW | Ship a patch changing the condition in `pollSessionStatus`; no state migration needed. |
| ETag clobber on concurrent ServeConfig calls | MEDIUM | Add mutex guard around `EnableFunnel`/`DisableFunnel`; user can recover stuck serve config via `tailscale serve reset` + re-enable. |
| #117 fix shipped without viewer-disconnect UI | MEDIUM | Open a new issue for Part B (viewer list + Disconnect); leave Part A (buffer increase) in place as a partial mitigation; do not revert. |
| beeep Windows notification fails (COM API) | LOW | beeep falls back to PowerShell notification; if both fail, notification is silently dropped — acceptable for v4.2. |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Funnel teardown incomplete on all paths (P1) | Funnel backend phase (first v4.2) | `tailscale serve status` empty after all four teardown triggers; integration test via real code paths |
| Public-exposure indicator is color-only (P2) | Funnel frontend phase | Read CSS hex values in source; verify text label present; no color-only state distinction; `prefers-reduced-motion` gated |
| Origin/BaseURL 403 before auth (P3) | Funnel backend phase (same phase as EnableFunnel) | HTTP request from an external machine (not tailnet) with Funnel origin returns 200, not 403 |
| ETag concurrency clobber (P4) | Funnel backend phase | Test with pre-existing serve config; verify ETag preserved via mutex-guarded read-modify-write |
| No prerequisite check before SetServeConfig (P5) | Funnel backend phase | Test with mock that fails `CheckFunnelAccess`; verify error surfaced, `SetServeConfig` not called |
| Cert provisioning latency (P6) | Funnel frontend phase | "Starting up" UX state present in `SessionShareModal` after enable; risk-ack mentions warmup |
| Notification spam (P7) | Notifications phase (#110) | Test: `running → waiting → sustained waiting × 5` fires notification exactly once |
| beeep "Script Editor" attribution (P8) | Notifications phase (#110) | Document in release notes; notification title includes "AgentHub" |
| Linux headless notification failures (P9) | Notifications phase (#110) | CI test on Linux runner; `sendNotification` errors are non-fatal; test logic layer with mocks |
| #117 only Part A ships without Part B (P10) | Bug-fix phase (#117) | Two-viewer UAT + Disconnect button verified in the same phase |
| #112 CSP violation or wrong apiBaseURL (P11) | Bug-fix phase (#112) | Real browser DevTools: no CSP errors; `/api/plugin-config/stream` is open EventSource |
| #118 breaks WS URL or REST calls for remote sessions (P12) | Bug-fix phase (#118) | Two-machine Tailscale UAT; all network calls in in-app remote tab use correct `baseURL` |
| Tests encode same wrong assumption (P13) | All Funnel phases | Integration tests use real code paths, no direct struct mutation in test setup; `check-traceability-paths.sh` passes |
| Local fallback broken by Funnel changes (P14) | Funnel backend phase | Fallback mode UAT with Tailscale not running: web-share works, Funnel toggle disabled |
| `prefers-reduced-motion` regression (P15) | Funnel frontend phase | `grep -r 'prefers-reduced-motion'` covers all new CSS animation files |

## Sources

- Direct code inspection: `internal/webserver/server.go`, `internal/webserver/origin_mw.go`, `internal/webserver/capability_mw.go`, `internal/daemon/api.go`, `internal/relay/hub.go`, `app.go`, `notification_darwin.go`, `notification_other.go`, `frontend/src/components/Hub/WebShareSessionView.tsx`, `frontend/src/components/StatusBar.tsx`, `frontend/src/App.tsx`, `frontend/src/components/Hub/SessionShareModal.tsx`
- v4.2 STACK.md (2026-06-30): beeep platform behavior, `CheckFunnelAccess` error strings, ETag on `ServeConfig`, Funnel port policy
- v4.2 ARCHITECTURE.md (2026-06-30): root causes for #112, #115, #117, #118; anti-patterns; dependency ordering for Funnel-aware BaseURL; teardown sites
- Project MEMORY.md: "Tests can encode the same wrong assumption" (Phase 150 shell-warning gate), "verify color-based UAT at source level not by eye", "cross-surface parity is release-blocking", Phase 155 false-parity (verifying on wrong surface), "two-machine tailnet" UAT precedent (Phase 122)
- Tailscale `ipn/serve.go` v1.98.3: exact error strings from `CheckFunnelAccess` (lines 612-615); `ETag` field tagged `json:"-"` confirmed in source

---
*Pitfalls research for: AgentHub v4.2 — Tailscale Funnel public sharing, cross-platform native notifications, Hub/web-share bug fixes (#112, #115, #117, #118) added to existing Go/Wails/React app*
*Researched: 2026-06-30*
