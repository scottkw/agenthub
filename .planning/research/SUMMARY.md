# Project Research Summary

**Project:** AgentHub v4.2 "Funnel Sharing & Polish"
**Domain:** Go/Wails v2 desktop app — Tailscale Funnel public internet sharing + cross-platform native notifications + Hub/web-share UX bug fixes
**Researched:** 2026-06-30
**Confidence:** HIGH (Funnel API verified against pinned v1.98.3 source on disk; architecture verified via direct code inspection of all named files; pitfalls grounded in project post-mortems)

---

## Executive Summary

AgentHub v4.2 adds Tailscale Funnel (one-click public-internet session sharing), awaiting-input native notifications, a Settings auto-switch toggle, and four v4.1 bug fixes to the existing Go/Wails v2 + React app. The centerpiece is Issue #107: Funnel exposes a session to anyone on the internet — not just tailnet members — gated by the existing join-code + capability-token system. Zero new Go dependencies are required for Funnel; every required API (SetServeConfig, GetServeConfig, CheckFunnelAccess, SetFunnel, IsFunnelOn) is present in the already-pinned tailscale.com v1.98.3. Only one new dependency is needed: github.com/gen2brain/beeep v0.11.2 for cross-platform native notifications.

The single highest-severity architectural rule for this milestone is atomicity of the Funnel backend: EnableFunnel/DisableFunnel lifecycle, the Funnel-aware FunnelBaseURL() method, the dual-origin requireAllowedOrigin fix, and the funnel-URL-emitting share-link builders must all ship in the same phase. A partial deployment where EnableFunnel() works but the Origin check is not yet updated causes every Funnel guest to 403 silently before the capability token is ever inspected — the symptom looks like a token bug but is an Origin header mismatch (Funnel arrives at port 443, Origin is https://hostname.ts.net with no port; BaseURL() returns https://hostname.ts.net:7443). Equally critical, teardown must fire on all four exit paths (user disables toggle, user disables web-share, session ends naturally, daemon shuts down cleanly); a missed teardown leaves a publicly-accessible Tailscale serve config alive with no session behind it.

The four v4.1 bug fixes (#112 plugin hot-swap, #115 Footer redundancy, #117 single-viewer kick, #118 external browser open) are independent of Funnel and of each other, with one exception: #117 requires both the relay buffer increase AND a viewer-disconnect UI to land in the same phase — shipping only the buffer change is documented as a recurring half-fix anti-pattern for this project. Funnel UAT must be verified from a machine OUTSIDE the tailnet; testing only from a tailnet device hides the Origin 403 bug (false-parity has bitten this project before).

---

## Key Findings

### Recommended Stack

No stack changes beyond one new module. tailscale.com v1.98.3 (already in go.mod) exposes the full Funnel control surface via tailscale.com/client/local — the same import already used in server.go. The local.Client is currently constructed and discarded inside startTailscale(); promoting it to a WebServer struct field (by value, zero-value usable, no constructor) is the entire infrastructure change required. All Funnel control flows through lc.SetServeConfig / lc.GetServeConfig. Wails v2.10.2 has no notification API (Wails v3-only); beeep v0.11.2 provides a no-CGO cross-platform path using osascript on macOS, COM API / PowerShell on Windows, and D-Bus / notify-send on Linux.

**Core technologies (additive only):**
- tailscale.com/client/local (already present): Funnel enable/disable via SetServeConfig/GetServeConfig; Funnel prereq check via ipn.CheckFunnelAccess; hostname resolution via StatusWithoutPeers
- tailscale.com/ipn (already present): ServeConfig struct, SetFunnel/IsFunnelOn helpers, CheckFunnelAccess/NodeCanFunnel/CheckFunnelPort prereq functions
- github.com/gen2brain/beeep v0.11.2 (NEW — only new dep): cross-platform native OS notifications, no CGO, single function call beeep.Notify(title, message, icon)

**Critical version notes:**
- Do NOT upgrade tailscale.com — Funnel APIs confirmed present in v1.98.3; a version bump is unnecessary and risks breaking the pinned go-webview2 constraint
- Do NOT attempt Wails v3 upgrade for notifications — v3 is alpha; beeep is the correct path
- Default Funnel port: 443 (gives a clean https://hostname.ts.net URL with no port component); fall back to 8443/10000 only if CheckFunnelPort(443) returns an error

### Expected Features

**Must have (table stakes):**
- Funnel toggle off-by-default — every comparable tool (ngrok, VS Code port-forwarding, Tailscale CLI) defaults to private; accidental public exposure is a showstopper
- Per-enable risk-acknowledgment dialog — blocks enabling; appears every time; no "don't show again"; contains risk statement, join-code gate explanation, TLS assurance, auto-expiry selector, and cross-link to Help article
- Persistent "INTERNET ACTIVE" visual indicator on Hub session card and session tab while Funnel is active — text + non-color icon, never color-only (release-blocking for colorblind owner)
- Funnel URL display with copy-to-clipboard immediately after enable
- One-click Funnel teardown from the Share modal
- Graceful, human-readable error when Funnel is not enabled in the tailnet (surface CheckFunnelAccess error strings verbatim)
- Awaiting-input notification fires once on running->waiting status transition; never re-fires during sustained waiting
- Notification user toggle in Settings (default: OFF)
- Footer "Share Session" button opens SessionShareModal (replaces dual Enable/Disable Web buttons, fixing #115 state drift)
- Settings toggle: "Switch to new session when created from Hub" (#116)
- Remote session opens in-app xterm.js tab, not external browser (#118)

**Should have (differentiators):**
- Auto-expiry selector in risk dialog (30 min / 1 hour / 4 hours / Session end) — no comparable tunneling tool offers this
- In-app Help article: "Sharing a Session Outside Your Tailnet" — Funnel path + device-share ACL alternative with copy-pasteable grants block and tailnet-wildcard-default gotcha callout
- Risk dialog cross-link to Help article
- Plugin-config / SSE hot-swap fix for web guests after Phase 159 redirect (#112)
- Multi-viewer Funnel sessions fix: relay buffer increase + viewer-disconnect UI, both in same phase (#117)
- Funnel cert warmup UX state ("Funnel is starting up..." for up to 30 seconds on first use)

**Defer to v4.3+:**
- Branded "AgentHub" notification sender on macOS (requires UNUserNotificationCenter CGO + entitlements + permission dialog)
- Device-share automation via Tailscale admin API (requires OAuth policy:write scope)
- ACL automation for tag:agenthub / autogroup:shared->tcp:7443 grant

### Architecture Approach

The Funnel integration touches four layers atomically: (1) WebServer struct promotion of local.Client + new EnableFunnel/DisableFunnel/FunnelBaseURL methods in server.go; (2) dual-origin allowlist in origin_mw.go and capability_mw.go; (3) Funnel-URL-emitting share-link builders in api.go at issueCapabilitiesForSession and handleExchangeJoinCode; (4) a funnelSessions map[string]bool reference-count on the API struct guarding DisableFunnel() calls (tear down only when the map is empty, to support multiple concurrent Funnel-enabled sessions). Notifications are wired entirely in app.go:pollSessionStatus — never from engine.go (which must remain Wails-import-free). The four v4.1 bug fixes each have confirmed root causes from direct code inspection.

**Major components modified:**

| Component | Change |
|-----------|--------|
| internal/webserver/server.go | Promote lc local.Client to struct field; add EnableFunnel/DisableFunnel/FunnelBaseURL |
| internal/webserver/origin_mw.go | Dual-URL allowlist: tailnet origin + Funnel origin (no port) |
| internal/webserver/capability_mw.go | originAllowedForWrite dual check |
| internal/daemon/api.go | funnelSessions map; handleSetSessionFunnel; funnel-aware URL builders; web-share teardown path |
| internal/relay/hub.go | Subscriber buffer 256->1024; new KickPersonKey method (#117) |
| app.go | SetSessionFunnel + KickSessionViewer bound methods; maybeNotifyWaiting with de-dup; notification platform dispatch |
| notification_windows.go (NEW), notification_linux.go (NEW) | Real platform notifications via beeep |
| frontend/src/components/Hub/SessionShareModal.tsx | Funnel toggle + risk-ack dialog; viewer list + Disconnect button (#117) |
| frontend/src/components/Hub/WebShareSessionView.tsx | Self-contained plugin-config fetch + SSE (#112); baseURL prop for remote-open (#118) |
| frontend/src/components/StatusBar.tsx | Replace Enable/Disable Web with single "Share Session" -> modal (#115) |
| frontend/src/components/SettingsTab.tsx | NotifyOnWaiting toggle + auto-switch toggle (#116) |
| frontend/src/content/help/sharing-guide.md (NEW) | Funnel + device-share help article |

### Critical Pitfalls

1. **Funnel teardown incomplete on non-happy-path exits** — The Tailscale serve config persists in the Tailscale daemon independent of the AgentHub process. Must wire teardown at all four sites: (a) user disables Funnel toggle, (b) user disables web-share for a Funnel-enabled session, (c) session onExit callback, (d) WebServer.Stop() / daemon shutdown. Verify via tailscale serve status after each trigger; integration tests must use real code paths, not mocked DisableFunnel stubs.

2. **Origin/BaseURL 403 before auth (integration landmine)** — Funnel guests arrive at port 443 with Origin: https://hostname.ts.net (no port). requireAllowedOrigin byte-matches ws.BaseURL() which returns https://hostname.ts.net:7443. Every guest 403s before the cap token is checked. Must ship FunnelBaseURL() and the dual-origin allowlist in the SAME PHASE as EnableFunnel(). UAT verification must come from a machine OUTSIDE the tailnet — testing only from a tailnet device hides this bug.

3. **Public-exposure indicator is color-only** — The Funnel-active indicator must include a persistent text label and a non-color icon. Never color-only. Verify CSS hex values in source, not by eye. Release-blocking defect for the colorblind project owner. Also verify prefers-reduced-motion is gated on all new animations.

4. **Notification spam — firing on state rather than transition** — pollSessionStatus runs every 500ms; waiting is a sustained state. Notification must fire only on the last != s.Status && s.Status == "waiting" transition. Default NotifyOnWaiting to OFF.

5. **#117 ships only Part A (buffer increase) without Part B (viewer-disconnect UI)** — The buffer increase reduces but does not eliminate kick-on-join. Without the Disconnect button in the Share modal there is no escape hatch. Both parts must be acceptance criteria of the same phase.

6. **Tests encoding the same wrong assumption** — Green CI has certified broken features before (Phase 150 shell-warning, Phase 155 false-parity). Origin allowlist tests must call EnableFunnel() — not set ws.funnelActive = true directly. Teardown tests must verify GetServeConfig returns empty — not assert a mock was called. Drive Funnel UAT from an external machine.

7. **ETag concurrency clobber on ServeConfig read-modify-write** — Always GetServeConfig first, modify the returned struct, preserve the ETag (tagged json:"-" so invisible in logs). Mutex-guard EnableFunnel/DisableFunnel. Never construct a &ipn.ServeConfig{} literal for SetServeConfig.

---

## Implications for Roadmap

Phase numbering continues from v4.1's Phase 164. Suggested phase structure:

### Phase 165: Funnel Backend (Atomic)

**Rationale:** The Funnel backend, Origin/BaseURL fix, and share-URL builders are a single atomic unit — shipping any subset causes silent 403s for every Funnel guest. This must be Phase 1 because all Funnel UI depends on it. External guests 403 before the cap token is checked if Origin is not fixed simultaneously.

**Delivers:**
- local.Client promoted to WebServer struct field (by value, zero-value usable)
- EnableFunnel(ctx, port) / DisableFunnel(ctx, port) / FunnelBaseURL() on WebServer
- Dual-origin allowlist in requireAllowedOrigin, allowedOrigins, originAllowedForWrite
- Funnel-URL-emitting issueCapabilitiesForSession and handleExchangeJoinCode (funnelBase when funnelSessions[sessionID] is true)
- funnelSessions reference-count map in API struct
- handleSetSessionFunnel daemon endpoint
- Teardown wired at all four sites: toggle-off, web-share-off, onExit, WebServer.Stop()
- ipn.CheckFunnelAccess preflight with verbatim error surfacing
- Fallback-mode guard: Funnel toggle disabled/hidden when Tailscale is not connected; funnelActive never set to true in fallback mode
- App.SetSessionFunnel Wails bound method

**Avoids:** Pitfalls 1 (teardown), 2 (Origin/BaseURL 403), 4 (ETag), 5 (no prereq check), 13 (wrong-assumption tests), 14 (fallback breakage)

**Research flag:** No additional research needed — all APIs verified against pinned v1.98.3 source.

**UAT gate (must verify from OUTSIDE tailnet):** tailscale serve status empty after all four teardown triggers; HTTP request with Origin: https://hostname.ts.net returns 200 (not 403); share URL has no :443 suffix; fallback-mode web-share unaffected with Tailscale not running.

---

### Phase 166: Funnel Frontend + Help Guide

**Rationale:** UI can only ship after Phase 165 backend is in place. Help article is a prerequisite for the risk-dialog cross-link; shipping them together avoids a dangling link.

**Delivers:**
- Funnel toggle + risk-acknowledgment dialog in SessionShareModal.tsx (per-enable, no "don't show again", auto-expiry selector, cross-link to Help article)
- Funnel URL display + copy-to-clipboard in Share modal
- Cert-warmup UX state ("Funnel is starting up...")
- Persistent "INTERNET ACTIVE" indicator on Hub session card and session tab (text + icon, colorblind-safe, prefers-reduced-motion gated)
- New frontend/src/content/help/sharing-guide.md — Funnel + device-share Help article (with grants ACL block, wildcard-default gotcha, revocation steps)
- HelpTab.tsx + HelpSectionNav.tsx updated with new "Sharing Guide" section

**Avoids:** Pitfalls 3 (color-only indicator), 6 (cert warmup UX), 15 (prefers-reduced-motion)

**Research flag:** No additional research needed.

**UAT gate:** Colorblind indicator verified at source-level (hex constants, not by eye); prefers-reduced-motion covered on all new animations; risk dialog appears on every Funnel enable (not remembered); "Funnel is starting up" state shown immediately after enable; Help article accessible via Help tab nav.

---

### Phase 167: Native Notifications (#110)

**Rationale:** Fully independent of Funnel. Can ship in any order after Phase 165; placing it here gives Funnel UAT time to stabilize before adding another moving part.

**Delivers:**
- beeep v0.11.2 added to go.mod
- notification_windows.go (beeep COM API) and notification_linux.go (beeep D-Bus / notify-send) complement existing notification_darwin.go
- maybeNotifyWaiting in app.go: transition-based de-dup (last != s.Status gate) + 60-second belt-and-suspenders de-dup map
- NotifyOnWaiting bool in Settings struct + settings serialization
- NotifyOnWaiting toggle in SettingsTab.tsx (Session Behavior section), default OFF
- "Script Editor" attribution on macOS documented in release notes (acceptable trade-off for v4.2)

**Avoids:** Pitfalls 7 (notification spam), 8 (Script Editor attribution), 9 (Linux CI failures — beeep errors non-fatal; notification logic tested with mocks, not beeep.Notify directly)

**Research flag:** No additional research needed.

**UAT gate:** running->waiting->sustained waiting x5 fires exactly once; 2-minute sustained-waiting verification; macOS notification appears ("Script Editor" attribution expected); Windows toast appears; Settings toggle OFF suppresses notification.

---

### Phase 168: Bug Fixes (#112, #115, #116, #117, #118)

**Rationale:** These issues are independent of Funnel and of each other. Batching reduces context-switch overhead. #117 is the most complex; it must ship with both Part A (buffer) and Part B (viewer-disconnect UI) in the same phase.

**Delivers:**

#115 — Footer state drift:
- StatusBar.tsx: replace Enable Web / Disable Web buttons with single "Share Session" button that opens SessionShareModal
- App.tsx: wire onOpenShareModal to current session at click time (not render time)

#116 — Auto-switch setting:
- New "Switch to new session when created from Hub" toggle in SettingsTab.tsx (default ON for backward compat)
- Hub session-creation flow respects the setting

#117 — Single-viewer kick (BOTH parts in same phase):
- internal/relay/hub.go: subscriber buffer 256->1024
- Hub.KickPersonKey(personKey string) method
- DELETE /sessions/{id}/viewers/{personKey} daemon endpoint
- App.KickSessionViewer(sessionID, personKey) Wails bound method
- Viewer list + Disconnect button in SessionShareModal.tsx

#118 — Remote-open external browser:
- WebShareSessionView.tsx: baseURL?: string prop; WS URL and all network calls use baseURL when provided (apiBaseURL = baseURL ?? window.location.origin)
- App.tsx: handleOpenRemoteSession creates in-app __websession__ tab instead of BrowserOpenURL

#112 — Plugin-config / SSE for web guests:
- WebShareSessionView.tsx: self-contained useEffect that fetches /api/plugin-config and subscribes to /api/plugin-config/stream when pluginConfig prop is null (browser context); Wails-context guard skips EventSource path in desktop app
- apiBaseURL for REST/EventSource is always window.location.origin — NOT the baseURL prop (different concerns)

**Avoids:** Pitfalls 10 (#117 half-fix), 11 (#112 CSP / wrong apiBaseURL), 12 (#118 broken REST calls for remote sessions)

**Research flag:** No additional research needed — root causes confirmed in direct code inspection.

**UAT gates:**
- #117: Two simultaneous viewers; one tabs away; Disconnect button terminates stuck viewer; viewer count updates within next poll cycle. TWO-MACHINE TAILNET UAT required.
- #118: Remote session opens in-app tab (not browser); terminal streams correctly; plugin config loads in in-app tab. TWO-MACHINE TAILNET UAT required.
- #112: Open web-share URL in real browser (not Wails WebView); DevTools Network shows /api/plugin-config 200 + /api/plugin-config/stream open EventSource; Console has no CSP errors.
- #115: Footer "Share Session" opens modal for current session; no independent state in footer.
- #116: Session created from Hub does not auto-switch when toggle is OFF.

---

### Phase Ordering Rationale

- Phase 165 before 166: backend atomicity — Funnel UI is broken without the Origin fix
- Phase 165 before 167/168: beeep and bug fixes are independent; ordering is for focus, not correctness
- Phase 166 and 167 are interchangeable in principle; 166 ships first because the risk-dialog cross-link needs the Help article
- Phase 168 bug fixes are parallelizable within the phase (each touches different files); #117 is most complex and should be planned first
- #117's Funnel multi-viewer issue (Origin check failure) may be resolved by Phase 165's dual-origin fix; verify in Phase 165 UAT before scoping Phase 168 #117 work

### Research Flags

All four phases have well-documented patterns and confirmed implementation paths. No phase in v4.2 requires a --research-phase sub-agent during planning:

- **Phase 165:** All Funnel APIs verified against pinned v1.98.3 source; all integration points confirmed in direct code inspection; no unknowns
- **Phase 166:** UI patterns follow existing SessionShareModal and Hub card extension points; Help system built in v4.0
- **Phase 167:** beeep API is a single function call; notification wiring point (pollSessionStatus) confirmed; existing notification_darwin.go provides the platform-file pattern
- **Phase 168:** All four root causes confirmed via direct code inspection with line numbers; fixes are targeted single-component changes

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All Funnel APIs verified against tailscale.com@v1.98.3 source on disk; beeep API verified from pkg.go.dev + GitHub source; Wails v2 runtime confirmed to have no notification API |
| Features | MEDIUM | Funnel UX patterns cross-referenced against ngrok, VS Code port-forwarding, Tailscale Funnel CLI docs; notification patterns from Warp and iTerm2 official docs |
| Architecture | HIGH | All named files inspected directly; line numbers confirmed for root causes of #112, #115, #117, #118; data flows traced end-to-end through Go backend and React frontend |
| Pitfalls | HIGH | Grounded in direct code inspection + project post-mortems from MEMORY.md (Phase 150 wrong-assumption tests, Phase 155 false-parity, colorblind indicator rule, two-machine tailnet UAT precedent) |

**Overall confidence: HIGH**

### Gaps to Address

- **Funnel cert warmup timing:** The 5-30 second window for Let's Encrypt cert provisioning on first-time Funnel use is documented but not measured empirically. If a readiness probe is implemented in Phase 166, the timeout and retry interval should be calibrated during execution.

- **NotifyOnWaiting default:** PITFALLS.md specifies default OFF; FEATURES.md does not state a default. Decision: **default OFF** — consistent with industry norm (Warp defaults notifications off) and avoids surprise spam on upgrade from v4.1.

- **beeep Windows COM API on AgentHub's tray-resident build:** Confirmed as working for tray-resident apps without AppUserModelId, but not yet UAT'd on AgentHub's actual Windows build. Flag for Phase 167 UAT on a real Windows machine.

- **#117 Funnel-origin dependency:** The likely root cause of second-viewer kick in Funnel sessions is an Origin check failure for the Funnel hostname. Phase 165's dual-origin fix may resolve the multi-viewer Funnel case without additional work in Phase 168. Verify in Phase 165 UAT; if confirmed, Phase 168 #117 scope reduces to buffer increase + viewer-disconnect UI only.

---

## Sources

### Primary (HIGH confidence)
- /Users/ken/go/pkg/mod/tailscale.com@v1.98.3/ipn/serve.go — ServeConfig struct, SetFunnel, CheckFunnelAccess/NodeCanFunnel/CheckFunnelPort, exact error strings, IsFunnelOn
- /Users/ken/go/pkg/mod/tailscale.com@v1.98.3/tailcfg/tailcfg.go — CapabilityHTTPS, NodeAttrFunnel, CapabilityFunnelPorts constants
- Direct code inspection (all files named in ARCHITECTURE.md): internal/webserver/server.go, origin_mw.go, capability_mw.go, internal/daemon/api.go, internal/relay/hub.go, app.go, notification_darwin.go, frontend/src/components/Hub/SessionShareModal.tsx, WebShareSessionView.tsx, StatusBar.tsx, App.tsx, HelpTab.tsx, HelpSectionNav.tsx
- pkg.go.dev/github.com/wailsapp/wails/v2@v2.10.2/pkg/runtime — confirmed no notification API in Wails v2
- Project MEMORY.md — colorblind indicator rule, wrong-assumption tests precedent (Phase 150), false-parity precedent (Phase 155), two-machine tailnet UAT precedent (Phase 122)

### Secondary (MEDIUM confidence)
- pkg.go.dev/tailscale.com@v1.98.3/client/local — SetServeConfig, GetServeConfig, Status, StatusWithoutPeers, QueryFeature signatures
- pkg.go.dev/github.com/gen2brain/beeep — Notify(title, message, icon any) error, v0.11.2 (Dec 2025)
- github.com/gen2brain/beeep/blob/master/notify_darwin.go — osascript approach, no CGO, no dock-icon dependency
- Tailscale Funnel documentation: https://tailscale.com/docs/features/tailscale-funnel
- Tailscale node sharing: https://tailscale.com/kb/1084/sharing
- Tailscale policy syntax / grants: https://tailscale.com/kb/1337/policy-syntax
- VS Code port forwarding: https://code.visualstudio.com/docs/debugtest/port-forwarding
- Warp desktop + agent notifications: https://docs.warp.dev/terminal/more-features/notifications/

### Tertiary (LOW confidence)
- tap-to-tmux (community notification de-dup pattern): https://github.com/flavio87/tap-to-tmux
- Claude Code + tmux notification pattern: https://software-dc.com/blog/4-claude-code-tmux-how-i-got-notifications-working

---
*Research completed: 2026-06-30*
*Ready for roadmap: yes*
