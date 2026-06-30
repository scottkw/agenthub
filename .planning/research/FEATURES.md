# Feature Research

**Domain:** AgentHub v4.2 — Funnel Sharing, Notifications, and Hub/Share UX Polish
**Researched:** 2026-06-30
**Confidence:** MEDIUM (Tailscale Funnel docs + Warp/iTerm2 notification patterns cross-checked; VS Code port-forwarding UX verified via official docs; device-share ACL patterns from official Tailscale KB)

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist when a tool claims "share with anyone over the internet." Missing these = the feature feels dangerous or half-built.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Funnel toggle off-by-default** | Every public-sharing tool (VS Code port-forwarding, ngrok, Cloudflare Tunnel) defaults to private/closed. The user must take an intentional affirmative action to go public. Accidental public exposure is a showstopper. | LOW | Funnel toggle appears greyed/disabled in SessionShareModal until explicitly enabled. No "auto-Funnel" on session creation. |
| **Per-enable risk-acknowledgment dialog** | VS Code shows a caution message before making a port Public. ngrok shows an interstitial to every visitor. Industry consensus: the sharer must consciously accept the risk. | LOW | Modal dialog that blocks enabling; must click a clearly-labeled confirm button. Per-enable, not one-time-remembered (see Anti-features). Contents specified in detail below. |
| **Persistent "INTERNET" visual indicator while Funnel is active** | VS Code marks Public ports in the Ports panel. Tailscale Funnel `status` output shows active funnels. Users must never forget a session is currently internet-exposed. | LOW | Distinct badge on the Hub session card (e.g. globe icon, red/amber) and inside the active session tab. Must remain visible even if the share modal is closed. |
| **One-click Funnel teardown** | `tailscale funnel <port> off` is the documented pattern. Any tool that makes it hard to revoke public access is dangerous. | LOW | A single "Stop" / "Disable Funnel" button in the share modal (and optionally from the card badge). Triggers `SetServeConfig` removal + clears the Funnel indicator immediately. |
| **Graceful error when Funnel not enabled in tailnet** | Funnel requires `nodeAttrs: [{attr: ["funnel"]}]` in the tailnet policy file. If the tailnet hasn't added it, `AllowFunnel` will fail. Users need a clear, actionable error — not a raw Go error. | LOW | Detect the specific `AllowFunnel` error code. Show a dialog: "Funnel is not enabled for your tailnet. An admin must add the Funnel node attribute in the Tailscale admin console → Access Controls → Add Funnel to policy." |
| **Funnel URL displayed and copyable** | ngrok shows the public URL immediately on tunnel creation. Users need the URL to share with their recipient. | LOW | After Funnel enables, surface the `<hostname>.ts.net`-based URL (resolved from `tailscale status`) in the share modal. Copy-to-clipboard button. |
| **Click-to-focus on notification** | Warp, iTerm2, and macOS Notification Center universally support click-to-navigate. Users expect clicking an "awaiting input" notification to bring the app forward and open that session. | LOW | macOS: `UNUserNotificationCenterDelegate userNotificationCenter:didReceive:`. Windows: WinRT `ToastNotification.Activated`. Linux: libnotify `action-invoked`. All must navigate to the session by ID. |
| **Notification fires only on state transition** | The polling pattern (fire every 30s while still `waiting`) is universally annoying. Warp, tap-to-tmux, and Claude Code community integrations all use one-notification-per-transition. | LOW | Fire notification when status CHANGES to `waiting`/`needs-input`. Do NOT re-fire on each polling cycle while the status remains unchanged. Clear when status changes away. |
| **Notification user toggle in Settings** | Warp provides this in Settings → Features → Notifications. Users who run many concurrent sessions need opt-out. | LOW | New toggle in Settings → Session Behavior: "Notify when a session is awaiting input" (default ON). |
| **DND / Focus mode respect** | macOS DND prevents Notification Center banners from appearing. Both macOS and Windows have system DND. Notifications that ignore DND are a trust violation. | LOW | Use the OS notification API correctly (macOS: `UNUserNotificationCenter`; Windows: `ToastNotificationManager`). DND is handled by the OS when notifications go through the system channel. No custom logic needed. |
| **"Share Session" as the single affordance (not a drift-prone toggle)** | Having two separate controls (footer toggle + share modal) for the same underlying state is a classic UX split-brain bug. Users of well-designed apps expect one canonical entry point. | LOW | Rename footer "Enable Web" button to "Share Session". Wire it to open the existing `SessionShareModal` (not to directly toggle web-share state). The modal is the canonical control; the footer button is a shortcut to it. Depends on: `SessionShareModal.tsx` (v4.0). |
| **Settings option to not auto-switch to newly-created session** | Every major terminal emulator (iTerm2, WezTerm, Ghostty) and IDE (VS Code, JetBrains) provides a "focus new tab" setting that power users frequently disable when batch-creating tabs. Absence is a frustration for Hub-centric workflows. | LOW | New toggle in Settings → Session Behavior: "Switch to new session when created from Hub" (default ON for backward compat). Depends on: Hub session creation flow. |
| **Remote session opens in-app (not external browser)** | VS Code Remote and JetBrains Gateway both open remote sessions inside their own window, not in an external browser. Opening a browser window breaks the unified app experience and is particularly jarring for tray-resident apps. | MEDIUM | When user clicks a remote session card on the Hub, connect via WebSocket relay and render in a new in-app xterm.js tab (same as `agenthub attach hostname:id` flow). Depends on: WebSocket relay (v2.0), Hub session card component. |

---

### Differentiators (Competitive Advantage)

Features that set AgentHub apart in this space.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Funnel auto-expiry tied to join code TTL** | ngrok has hard session limits on free tier (2 hours). VS Code has no expiry. AgentHub's join codes are already TTL-bearing; surfacing this as "share expires in X" makes the risk explicit and time-bounded without extra backend work. | LOW | In the risk dialog, offer an "Auto-expire Funnel after" selector: 30 min / 1 hour / 4 hours / Session end (default: Session end = Funnel tears down when session is deleted). After expiry, tear down Funnel automatically and show a notification. |
| **Cross-link from risk dialog to Help guide** | VS Code and ngrok provide no in-context education about safer alternatives. AgentHub can uniquely link from the Funnel risk dialog directly to the "Sharing outside your tailnet" Help article covering the device-share alternative — turning the friction point into a teaching moment. | LOW | Link at bottom of risk dialog: "Want tighter containment? See Sharing a session with someone outside your tailnet →" — opens the Help tab to the device-share article. |
| **In-app Help article: sharing outside your tailnet** | No terminal multiplexer or AI coding tool ships guidance on Tailscale device-sharing ACLs. A concrete, copy-pasteable guide scoped to AgentHub's specific port is a genuine 1-of-1 in this space. | LOW | New Help markdown article (see Help Guide Structure below). Depends on: in-app Help page (HelpTab.tsx + HelpSectionNav.tsx, v4.0). |
| **Funnel-aware BaseURL and Origin validation** | The real landmine: when Funnel is active, all URLs (share links, capability token exchanges, Origin headers) must use the Funnel hostname, not the tailnet-only hostname. Without this fix, Funnel shares 403-fail before the capability token is ever checked. No comparable tool has this problem — they don't have a capability-token-gated perimeter. | HIGH | On Funnel enable, cache Funnel hostname from `tailscale status`. Pass it to `BaseURL()` and the Origin allowlist for the duration of the Funnel session. `issueCapabilitiesForSession` and `handleExchangeJoinCode` must emit the Funnel URL. Depends on: embedded Tailscale LocalClient, web server origin check, capability token system (v3.1). |
| **Plugin-config / SSE hot-swap fix for web guests** | Web-share guests lost live plugin-config and SSE hot-swap after v4.1 Phase 159's /app/ redirect. Fixing this ensures parity between desktop and web surfaces — a standing release norm. | MEDIUM | After the /app/ redirect, the /api/plugin-config REST and SSE endpoints are unreachable from web guests. Fix: proxy or re-expose them from the Funnel-accessible path, or embed plugin config in the initial serve response. Depends on: web server serve config, SSE infrastructure (v3.2). |
| **Multi-viewer Funnel sessions (bug fix: second viewer kicks first)** | Funnel sessions should support the same multi-viewer model as tailnet sessions. Currently a second WebSocket connection kicks the first viewer, who remains "connected" in the Hub with no way to disconnect. | MEDIUM | The max-wins PTY resize arbitration (v2.0) already supports multiple clients. Diagnose why the second Funnel WebSocket is not being added to the subscriber list — likely an Origin check failure that falls back to single-client behavior. Fix: ensure Funnel Origin is in the allowlist so the relay accepts multiple concurrent connections. |

---

### Anti-Features (Explicitly Avoid)

| Feature | Surface Appeal | Why Avoid | What to Do Instead |
|---------|---------------|-----------|-------------------|
| **One-time-remembered risk acknowledgment** | Reduces friction for repeat users who know they're going public. | Funnel exposure is acute and intentional. Users who leave a session Funnel-enabled and forget about it are the risk scenario. Per-enable confirmation keeps the action deliberate. The one-time pattern is appropriate for lower-stakes preferences (e.g. theme), not for public internet exposure of a live terminal. | Per-enable risk dialog, every time. Make it fast to dismiss (large confirm button) but impossible to accidentally trigger. |
| **Auto-Funnel on session creation** | Seems convenient for users who always want to share publicly. | Violates the off-by-default security norm that every tunneling tool upholds. Any accidental or scripted session creation would immediately become internet-exposed. | Keep Funnel toggleable only from the Share modal, never automatic. |
| **Funnel persisting after web-share is disabled** | Simpler teardown logic — only one state to check. | Funnel is a superset of web-share. Disabling web-share should implicitly disable Funnel (you can't have a Funnel without an underlying web-share). Lingering Funnel without web-share is a dangling public exposure. | On web-share disable, tear down Funnel first, then disable web-share. On session end, tear down Funnel. Three teardown triggers: (1) explicit Funnel disable, (2) web-share disable, (3) session deletion. |
| **Notification polling: re-fire while still waiting** | Easy to implement (just fire on each status poll that shows `waiting`). | If a session stays in `waiting` for 10 minutes and the poll interval is 30s, the user gets 20 notifications. Every comparable tool (Warp, tap-to-tmux) uses transition-based triggering to prevent this exact pattern. | Track the previous status per session. Fire notification only when status transitions TO `waiting`/`needs-input`. |
| **Notifications with no user toggle** | Simpler settings surface. | Some users run many concurrent Claude Code sessions and ambient `waiting` notifications are noise. Warp provides a toggle; iTerm2 provides a toggle. Absence of a toggle will generate user complaints. | Settings → Session Behavior toggle, default ON. |
| **Separate "Enable Web" state independent of Share modal** | Footer button quick-access feels convenient. | This is the existing bug #115: the footer toggle and the share modal can drift out of sync, creating a confusing state where the modal says one thing and the footer says another. | Unify: footer button is exclusively a shortcut to open the share modal. No independent state management. |
| **Device-share automation via Tailscale admin API** | Would make the "contained alternative" fully one-click from within AgentHub. | Requires an OAuth client with `policy_file` scope — a separate, heavier auth flow that adds a sensitive admin credential to AgentHub's config. Not justified for v4.2. | Document the device-share steps in the Help guide. Provide copy-pasteable ACL grants. The manual steps (Machine page → Share → accept invite) are a one-time setup per external collaborator. |

---

## Feature Dependencies

```
[LocalClient persisted on WebServer]
    +--required by--> [Funnel enable/disable toggle]
    +--required by--> [Funnel-aware BaseURL()]
    +--required by--> [Funnel hostname caching]

[Funnel backend (SetServeConfig/AllowFunnel)]
    +--required by--> [Funnel toggle in SessionShareModal]
    +--required by--> [Funnel teardown on web-share-off / session-end]
    +--required by--> [Funnel URL display]

[Funnel-aware BaseURL() + Origin allowlist]
    +--required by--> [Funnel share links work (don't 403)]
    +--required by--> [Multi-viewer Funnel fix]

[Risk acknowledgment dialog UI]
    +--required by--> [Cross-link to Help article] (link in dialog body)
    +--depends on---> [Help article existing]

[SessionShareModal.tsx (v4.0 existing)]
    +--extended by--> [Funnel toggle + risk dialog]
    +--extended by--> ["Share Session" footer button wiring (#115)]

[Hub session card (v4.0 existing)]
    +--extended by--> [Persistent INTERNET badge]
    +--extended by--> [In-app remote session tab (#118)]

[WebSocket relay (v2.0 existing)]
    +--reused by----> [In-app remote session tab (#118) — already handles agenthub attach hostname:id]

[HelpTab.tsx + HelpSectionNav.tsx (v4.0 existing)]
    +--extended by--> [New "Sharing outside your tailnet" Help article]

[Session status polling (existing daemon)]
    +--extended by--> [Awaiting-input notification trigger (transition detection)]

[In-app Help article]
    +--referenced by-> [Risk dialog cross-link]
```

### Dependency Notes

- **LocalClient must be persisted first**: everything Funnel-related depends on `WebServer` holding a `*tailscale.LocalClient` across request lifetimes, not constructing and discarding it per-request as today.
- **BaseURL/Origin fix is a correctness prerequisite for Funnel UX**: the Funnel toggle can exist in the UI without it, but sharing will silently 403. Must ship together.
- **Help article must exist before the risk dialog cross-link**: if the article is built in a later phase than the Funnel feature, the cross-link should be a deferred P2 addition.
- **Notification transition detection requires no new daemon infrastructure**: status is already polled; add a per-session "last-known-status" cache in the notification layer.
- **#115, #116, #118 are independent of Funnel**: the UX fixes have no Funnel dependency and can ship in any order.
- **Multi-viewer Funnel fix (#117) depends on Funnel-aware Origin allowlist**: the likely root cause is an Origin check failure for the Funnel hostname.

---

## Concrete Acceptance Criteria

### Risk Acknowledgment Dialog — Required Contents

The dialog blocks enabling Funnel until the user clicks confirm. It must contain ALL of the following:

**Title:** "Enable Funnel — Share over the Internet"

**Body (required elements):**
1. Plain-language statement of what Funnel does: "This will expose this session to the public internet. Anyone with the session URL can access it — your session is no longer limited to your tailnet."
2. The only gate: "Visitors must redeem a join code before viewing. The join code expires after [selected duration]."
3. Encryption reassurance (reduces false alarm fatigue): "Traffic is TLS-encrypted by Tailscale's Funnel relay."
4. The "not tailnet-only" callout framed as a risk: "This URL works from any device, anywhere in the world — not just Tailscale-connected devices."
5. Recommendation: "Recommended: keep this session read-only and use a short expiry."
6. Auto-expiry selector: "Disable Funnel automatically after: [30 min | 1 hour | 4 hours | Session end (default)]"
7. Cross-link: "Want tighter containment? See 'Sharing outside your tailnet' in Help →" (opens Help tab to device-share article)

**Buttons:** [Cancel] [Enable Funnel] — destructive action on the right, full-width or right-aligned.

**Do NOT include:** checkboxes for "don't show again" or "I understand" that suppress future dialogs.

**Trigger:** Every time the Funnel toggle is switched from off → on. Closing the dialog (Cancel) leaves Funnel disabled.

---

### In-App Help Guide Structure: "Sharing a Session with Someone Outside Your Tailnet"

This article belongs in the existing Help page (HelpTab.tsx). It must be scannable via the existing HelpSectionNav anchor system.

**Article title:** "Sharing a Session Outside Your Tailnet"

**Required sections (in order):**

**1. Two Paths (comparison overview — at top, above the fold)**

A two-column or bulleted quick-compare: Funnel (easier, no Tailscale required for recipient, public internet, gated by join code + TTL) vs. Device Sharing (recipient must have a Tailscale account, traffic stays within Tailscale network, more contained).

**2. Option A: Tailscale Funnel (Quickest)**

- One paragraph: "Funnel exposes your AgentHub web server to the public internet via Tailscale's relay. The recipient needs no Tailscale account — just the URL and a join code."
- Prerequisites listed: Funnel must be enabled in your tailnet (link to tailscale.com/kb/1223/tailscale-funnel), MagicDNS + HTTPS must be on.
- Steps: Open Hub → Share icon → toggle "Share over the internet (Funnel)" → copy the Funnel URL and send it with the join code.
- Limitation: Anyone with the URL can attempt to redeem a join code. Join codes are single-use and TTL-expiring.

**3. Option B: Device Sharing (More Contained)**

This is the section with acceptance-criteria-bearing detail.

**Prerequisites:**
- Recipient must have a Tailscale account (free tier is sufficient).
- You must be Owner, Admin, or IT admin in your tailnet to share a machine.

**Steps (numbered list, copy-paste friendly):**
1. Open the Tailscale admin console → Machines.
2. Find the machine running AgentHub. Click the `...` menu → Share.
3. Enter the recipient's email address or generate a shareable link (single-use, 30-day expiry).
4. The recipient accepts the invite in their Tailscale client.
5. Share your AgentHub URL with them using your machine's MagicDNS name: `https://<your-machine>.<tailnet>.ts.net:7443/`

**ACL grant (copy-pasteable — required verbatim in the article):**

By default, shared users may have wildcard access to your tailnet if you are using a legacy `acls` block with `*→*`. To restrict them to AgentHub only, add this to your tailnet policy file under `grants`:

```json
{
  "grants": [
    {
      "src": ["autogroup:shared"],
      "dst": ["tag:agenthub"],
      "ip": ["tcp:7443"]
    }
  ]
}
```

Also tag your AgentHub machine: open the admin console → Machines → your machine → Tags → add `agenthub`.

**The wildcard-default gotcha (required callout box):**

> **Warning:** If your tailnet policy file uses the legacy `acls` block with `"src": ["*"], "dst": ["*"]` (the default for new tailnets), shared users get that wildcard access too — not just AgentHub. Add the `grants` block above to restrict them. Tailscale recommends migrating from `acls` to `grants` for all new rules.

**Limitations:**
- Shared machines are quarantined: the recipient cannot initiate connections to other devices in your tailnet.
- MagicDNS names for shared machines require the fully-qualified form (`hostname.tailnet-name.ts.net`).
- Shared machine tags are not visible to the recipient's tailnet.

**4. Revocation**
- Funnel: toggle "Share over the internet" off in the Share modal, or it tears down automatically when the session ends.
- Device sharing: open admin console → Machines → shared machine → Sharing → revoke access.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Funnel backend (LocalClient persist + SetServeConfig/AllowFunnel) | HIGH | HIGH | P1 — foundation for all Funnel features |
| Funnel-aware BaseURL() + Origin allowlist | HIGH | MEDIUM | P1 — correctness gate; Funnel is broken without it |
| Funnel toggle + risk dialog in SessionShareModal | HIGH | LOW | P1 — requires P1 backend |
| Persistent INTERNET indicator (Hub card + session tab) | HIGH | LOW | P1 |
| Funnel URL display + copy | HIGH | LOW | P1 |
| Graceful Funnel-not-enabled error | HIGH | LOW | P1 |
| Funnel teardown lifecycle (3 triggers) | HIGH | MEDIUM | P1 |
| Auto-expiry selector | MEDIUM | LOW | P2 |
| Awaiting-input notifications (macOS/Windows/Linux) | HIGH | HIGH | P1 |
| Notification transition-based deduplication | HIGH | LOW | P1 — must ship with notifications |
| Notification user toggle in Settings | MEDIUM | LOW | P1 — must ship with notifications |
| Help article: Sharing outside tailnet | MEDIUM | LOW | P1 |
| Risk dialog cross-link to Help article | MEDIUM | LOW | P2 (P1 if Help article ships in same phase) |
| Footer "Share Session" → modal wiring (#115) | MEDIUM | LOW | P1 |
| Auto-switch setting (#116) | MEDIUM | LOW | P1 |
| In-app remote session tab (#118) | HIGH | MEDIUM | P1 |
| Multi-viewer Funnel fix (#117) | MEDIUM | MEDIUM | P2 (depends on Funnel Origin fix) |
| Plugin-config/SSE fix for web guests (#112) | MEDIUM | MEDIUM | P2 |

---

## Competitor / Comparable Tool Analysis

| Behavior | ngrok | VS Code Port Forwarding | Tailscale Funnel (direct CLI) | Warp Agent | AgentHub v4.2 approach |
|----------|-------|------------------------|-------------------------------|------------|----------------------|
| Default to private | Yes (tunnels are private by default) | Yes (Private visibility by default) | Yes (Funnel is opt-in; `tailscale serve` is tailnet-only) | N/A | Yes — Funnel off-by-default |
| Risk acknowledgment to sharer | None on free tier (warning goes to visitors) | Caution text in Ports panel | None in CLI (just runs) | N/A | Per-enable modal dialog |
| Public indicator while live | URL display in terminal output | "Public" badge in Ports panel | `tailscale funnel status` CLI | N/A | INTERNET badge on Hub card + session tab |
| One-click teardown | `ngrok http off` / close process | Right-click → Stop Forwarding | `tailscale funnel <port> off` | N/A | "Stop Funnel" button in share modal |
| Auto-expiry | 2 hours on free tier | None | None (persists with -bg) | N/A | Configurable: 30min/1h/4h/session-end |
| Notification trigger | N/A | N/A | N/A | User-input-required, errors, completion | Status → `waiting` transition |
| Notification deduplication | N/A | N/A | N/A | Not documented | Fire once on transition, not per poll |
| Notification user toggle | N/A | N/A | N/A | Settings → Features → Notifications | Settings → Session Behavior |
| Click-to-focus notification | N/A | N/A | N/A | Yes (click navigates to agent tab) | Yes (brings app forward, navigates to session) |
| In-app remote session | N/A | VS Code Remote renders in-app | JetBrains Gateway renders in-app | N/A | In-app xterm.js tab via WSS relay |
| Single share affordance | One URL+tunnel | Ports panel (one panel) | One command | N/A | "Share Session" → SessionShareModal (unified) |
| Auto-switch to new tab | N/A | No (stays on current editor) | N/A | N/A | User toggle in Settings |

**Key takeaways:**
- Every tunneling tool defaults to private. AgentHub must match this.
- ngrok puts the risk warning on the VISITOR side (interstitial page); VS Code puts it on the SHARER side (caution text). AgentHub should put it on the sharer side (risk dialog) — the visitor already passes through a join code gate.
- No tunneling tool offers auto-expiry; this is a genuine differentiator.
- Warp's agent notification model (completion / input-required / error) is the industry reference. AgentHub covers the `input-required` trigger.
- VS Code and JetBrains universally render remote sessions in-app. Opening a browser window for remote sessions is the odd-one-out behavior that #118 fixes.

---

## Sources

- Tailscale Funnel documentation: https://tailscale.com/docs/features/tailscale-funnel (MEDIUM — official docs, webfetch)
- Tailscale node sharing: https://tailscale.com/kb/1084/sharing (MEDIUM — official KB, webfetch)
- Tailscale policy syntax / grants: https://tailscale.com/kb/1337/policy-syntax (MEDIUM — official docs, webfetch)
- Warp desktop notifications: https://docs.warp.dev/terminal/more-features/notifications/ (MEDIUM — official docs, webfetch)
- Warp agent notifications: https://docs.warp.dev/agent-platform/capabilities/agent-notifications/ (MEDIUM — official docs, webfetch)
- VS Code port forwarding: https://code.visualstudio.com/docs/debugtest/port-forwarding (MEDIUM — official docs, webfetch)
- ngrok interstitial / free tier: https://ngrok.com/docs/pricing-limits/free-plan-limits (MEDIUM — official docs, websearch)
- tap-to-tmux (per-project deduplication pattern): https://github.com/flavio87/tap-to-tmux (LOW — community project)
- Claude Code + tmux notification pattern: https://software-dc.com/blog/4-claude-code-tmux-how-i-got-notifications-working (LOW — community blog)
- Project context: `.planning/PROJECT.md` v4.2 milestone section (HIGH — project source of truth)

---
*Feature research for: AgentHub v4.2 Funnel Sharing & Polish*
*Researched: 2026-06-30*
