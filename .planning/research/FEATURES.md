# Feature Research

**Domain:** Remote terminal session access, auto-update, Tailscale onboarding, and app polish for a Go/Wails desktop app
**Researched:** 2026-04-06
**Confidence:** HIGH (Tailscale LocalClient API verified via pkg.go.dev; Wails menu API verified; go-selfupdate API verified via pkg.go.dev; macOS menu conventions from Apple docs)

---

## v1.9 Milestone: Remote Sessions & App Polish

### Scope

This section covers only what is NEW in v1.9. Existing features already shipped:
tabbed terminal UI, web serving via Tailscale Let's Encrypt, CLI with 13 commands,
daemon architecture with Unix socket IPC, system tray, splash screen, GitHub distribution.

Focus areas:
1. **Remote session discovery and access** — enumerate tailnet peers, probe their AgentHub daemons, list/access remote sessions
2. **Auto-update** — check GitHub releases for newer version, notify user, open download link
3. **Tailscale install guidance** — improve existing health modal with direct install links
4. **Standard app menus** — File, Edit, Window, Help (Wails menu API)
5. **Welcome screen polish** — real build-time version, logo rounded corners

Prior milestone research (v1.8) preserved at bottom.

---

## Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Remote session listing per peer | If the app serves sessions over Tailscale, users expect to see what's running on other machines too | MEDIUM | `local.Client{}.Status()` returns `ipnstate.Status` with `Peer map` containing `PeerStatus{HostName, DNSName, TailscaleIPs}`. Probe each peer's AgentHub HTTP port asynchronously. Return cached results; refresh on interval. Already use `local.Client{}` for health checks in v1.2. |
| Remote sessions visible in GUI panel | Unified view of local + remote sessions is the core promise of a multi-machine workflow tool | MEDIUM | Extend existing DaemonManagerPanel tab. Add peer hostname column. Group by machine or show flat list with hostname. Link each remote session to its web terminal URL. |
| Update available notification | Any app distributed via GitHub Releases should notify when a newer version exists | LOW | Background goroutine on startup. Compare embedded build version to GitHub latest release tag. Surface as badge or banner in WelcomeTab or tray menu. Non-intrusive — do not block or pop modal. |
| Build-time version displayed in app | WelcomeTab currently shows hardcoded version string (v1.7 tech debt). Users notice when version is wrong after updating. | LOW | Inject via `-ldflags "-X main.Version=$(git describe --tags)"` in build.sh and CI. Expose via Wails binding. Fix WelcomeTab.tsx to read from backend. |
| Standard Edit menu (clipboard shortcuts) | On macOS, Cmd+C / Cmd+V / Cmd+Z silently fail in WebView without a registered Edit menu. This is a known macOS/WebView behavior that breaks terminal copy/paste. | LOW | Wails v2 provides `menu.EditMenu()` convenience method — one line. Without this, clipboard in xterm.js terminal and text fields is broken for many users. P1 blocker. |
| Standard Window menu | macOS users expect Cmd+M (minimize), Cmd+W (close) to work. Missing Window menu is a macOS convention violation. | LOW | Wails v2 provides `menu.WindowMenu()`. Standard macOS behavior. |
| Help menu with About | Standard macOS convention. Users look in Help for version info, documentation link, support. | LOW | Custom Help menu: "About AgentHub" (version dialog or modal), "Documentation" (opens GitHub README in browser), "Report Issue" (opens GitHub issues). |
| Tailscale install link in health modal | Health modal (v1.2) already has platform-specific instructions. Users need a direct download link, not just text instructions. | LOW | Add `https://tailscale.com/download` link per platform to existing health modal. macOS: Homebrew cask or App Store. Linux: `tailscale.com/install.sh`. Windows: MSI download link. |

---

## Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Zero-config remote peer discovery | No IP, no hostname, no port required — peers discovered automatically via Tailscale LocalClient peer enumeration | MEDIUM | `local.Client{}.Status()` → iterate `status.Peer` map → for each peer with `TailscaleIPs`, probe `http://<ip>:<agentHubPort>/api/sessions` with 1-2s timeout. Runs in background goroutines. Falls back gracefully on probe failure. This is fundamentally simpler than SSH tunneling, relay servers, or any credentials-based approach. |
| Unified remote+local session list | Single `agenthub list` command shows all reachable sessions across all tailnet peers, with hostname column | MEDIUM | CLI aggregates: local daemon API (existing) + HTTP probes to peer AgentHub daemons (new). `--local` flag for local-only. Hostname shown in `list` output table. |
| One-click access to remote sessions | Clicking a remote session in GUI opens its web terminal URL directly in the browser | LOW | Remote session items in panel carry the web URL (already served by peer's AgentHub). No new protocol — browser handles the connection through Tailscale. |

---

## Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Auto-replace binary on update (silent self-update) | Seamless "no action" update experience like Chrome | macOS Gatekeeper quarantines binaries downloaded programmatically; replacing a signed+notarized binary with a self-downloaded one silently breaks the app for users. Windows SmartScreen similar. | Notify of update + open GitHub releases page in browser. Homebrew/WinGet handle updates via their own channels for package-manager users. |
| Full Tailscale management UI | Convenience of not switching to Tailscale app | Out of scope (explicitly in PROJECT.md). Each Tailscale API change breaks the management UI. | Show health status (existing v1.2), add install links to health modal. Tailscale's own GUI handles ongoing management. |
| mDNS/Bonjour discovery alongside Tailscale | Find peers without Tailscale dependency | Product is Tailscale-native; adding mDNS introduces a parallel discovery path with different security model. Tailscale is the auth layer. | Tailscale peer list via LocalClient is the only discovery mechanism. |
| Polling all peers synchronously on every tray menu open | Always-fresh remote session count in tray menu | NSMenuDelegate `menuWillOpen:` is on the main thread; network latency on probe would freeze the menu. | Background goroutine refreshes peer probe cache on a 30s interval. Tray menu reads from cache. |
| Remote attach via new custom protocol | Rich bidirectional terminal over Tailscale without browser | WebSocket relay already exists and works for browsers. Building a second transport is duplicate work. | Remote CLI attach via existing WebSocket relay: `agenthub attach <peer>/<session-id>` routes through `wss://<peer-fqdn>/ws/<session-id>` — existing protocol, new routing only. |
| Tailscale Services API for port advertisement | Programmatic service discovery via Tailscale's own infrastructure | Tailscale Services is alpha (2025), not stable, not recommended for production use. LocalClient peer probing is simpler and already-used infrastructure. | HTTP probe pattern using existing `local.Client{}` and consistent AgentHub port. |

---

## Feature Dependencies

```
Remote Session Discovery
    └──requires──> Tailscale LocalClient.Status() [already used in v1.2 health checks]
                       └──requires──> Tailscale installed and connected [existing health gate]
    └──requires──> AgentHub daemon port consistent across installs [existing: same binary = same default port]

Remote Session GUI Panel
    └──requires──> Remote Session Discovery (probe results)
    └──requires──> Existing DaemonManagerPanel [v1.7 already built, extend it]

Remote Session CLI List
    └──requires──> Remote Session Discovery (probe results)
    └──requires──> Existing CLI list command [v1.3, extend with --remote flag]

Remote Attach via WebSocket
    └──requires──> Remote Session Discovery (know peer FQDN + session ID)
    └──requires──> Existing WebSocket relay on remote peer [v1.0, already serves browser sessions]
    └──enhances──> Remote Session CLI List

Auto-Update Notification (GUI)
    └──requires──> Build-time version injection [tech debt fix — prerequisite]
    └──requires──> GitHub Releases API access [new: go-selfupdate library]
    └──produces──> Version string + AssetURL for download link

Build-time Version Fix
    └──required by──> Auto-Update Notification (need real version to compare)
    └──required by──> Welcome screen version display (fix hardcoded WelcomeTab.tsx)
    └──blocks nothing if done first

Standard App Menus
    └──requires──> Wails menu.Menu struct passed in options [not currently set]
    └──no conflicts with existing features
    └──fixes──> clipboard in xterm.js on macOS (implicit dependency)

Tailscale Install Link
    └──enhances──> Existing health modal [v1.2]
    └──no new dependencies
```

### Dependency Notes

- **Remote session discovery builds on existing Tailscale LocalClient:** The `local.Client{}` zero-value is already used in `internal/daemon` for health checks (v1.2). `Status()` call adds peer enumeration with zero new dependencies.
- **Build-time version must be fixed before auto-update check:** Comparing `"0.0.0-dev"` or `"v1.7.0"` (hardcoded) against GitHub latest release produces wrong results. Fix version injection first, then wire auto-update.
- **Standard menus are additive:** No existing feature conflicts with adding Wails menus. The `options.Menu` field is currently unset. Adding it does not change existing behavior except fixing clipboard.
- **Remote attach can be deferred:** Remote session listing (discovery + GUI panel + CLI list) delivers primary value. Remote attach is a follow-on feature once listing is reliable.

---

## MVP Definition for v1.9

### Launch With (v1.9 — minimum to call milestone complete)

- [ ] **Build-time version injection** — ldflags in build.sh, VERSION exposed via Wails binding, WelcomeTab reads from backend. Prerequisite for everything else.
- [ ] **Remote session discovery** — background goroutine, `local.Client{}.Status()` peer enumeration, parallel HTTP probes, 30s refresh cache.
- [ ] **Remote session GUI panel** — extend DaemonManagerPanel with remote peers section; show peer hostname, session list, status, web terminal link.
- [ ] **Auto-update check** — background check via `creativeprojects/go-selfupdate`; show update available banner in WelcomeTab with "Download" button opening browser.
- [ ] **Standard app menus** — `menu.AppMenu()`, `menu.EditMenu()`, `menu.WindowMenu()`, custom File (New Session, Quit), custom Help (About, Documentation, Report Issue). Fixes macOS clipboard.
- [ ] **Tailscale install link improvement** — add direct download URLs to existing health modal per platform.
- [ ] **Welcome logo rounded corners** — CSS `border-radius` on logo image. Trivial polish.

### Add After Core Works (v1.9.x)

- [ ] **Remote session CLI list** — `agenthub list --remote` or unified list with hostname column. Depends on discovery being reliable.
- [ ] **Remote attach from CLI** — `agenthub attach <peer>/<id>` routing through remote peer's WebSocket relay. Higher complexity, lower urgency.

### Future Consideration (v2+)

- [ ] **Remote session QR from GUI** — fetch QR data from remote peer API. Web dashboard on remote machine already has QRs; low incremental value.
- [ ] **In-app Tailscale auto-install** — run brew/winget/apt commands in a new terminal tab. High complexity, low frequency use case.
- [ ] **Tailscale Services integration** — use Tailscale's alpha endpoint advertisement API instead of HTTP probing. Not stable; revisit when GA.
- [ ] **Peer-to-peer update push** — push binary updates to remote machines. Significant security surface; not aligned with current auth model.

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Standard Edit menu (fixes clipboard) | HIGH — broken clipboard in terminal is a P0 bug for many users | LOW — `menu.EditMenu()` one line | P1 |
| Build-time version injection | HIGH — prerequisite for update checker and accurate welcome screen | LOW — ldflags already used in CLI binary | P1 |
| Remote session discovery | HIGH — core milestone value proposition | MEDIUM — LocalClient + parallel HTTP probes + cache | P1 |
| Remote session GUI panel | HIGH — primary user-facing remote feature | MEDIUM — extend DaemonManagerPanel | P1 |
| Auto-update notification | HIGH — expected in any distributed desktop app | MEDIUM — go-selfupdate + Wails event + WelcomeTab badge | P1 |
| Standard Window menu | MEDIUM — convention violation without it | LOW — `menu.WindowMenu()` one line | P1 |
| Help menu | MEDIUM — users look here for version + docs | LOW — custom menu items, `BrowserOpenURL` | P1 |
| Tailscale install link | MEDIUM — reduces new user friction | LOW — add URLs to existing health modal | P2 |
| Remote session CLI list | MEDIUM — useful for power users | MEDIUM — aggregate local + remote in list command | P2 |
| Welcome logo rounded corners | LOW — cosmetic polish | LOW — CSS change | P2 |
| Remote attach from CLI | MEDIUM — useful but edge case | HIGH — WebSocket proxy routing | P3 |

**Priority key:**
- P1: Must have for v1.9 milestone to ship
- P2: Should have, include if time permits
- P3: Nice to have, defer to v2.0

---

## Key Implementation Decisions

### Remote Session Discovery — Design

**Discovery mechanism:** Call `local.Client{}.Status(ctx)` → iterate `status.Peer` (type `map[key.NodePublic]*ipnstate.PeerStatus`). Each `PeerStatus` has:
- `HostName string` — bare hostname (e.g. `"my-macbook"`)
- `DNSName string` — FQDN from MagicDNS (e.g. `"my-macbook.tail1234.ts.net"`)
- `TailscaleIPs []netip.Addr` — Tailscale IP addresses (e.g. `100.x.x.x`)
- `Online bool` — whether peer is online

**Probe strategy:** For each online peer, `GET http://<TailscaleIP>:<agentHubPort>/api/sessions` with 1-2s timeout. Run probes concurrently in goroutines. If probe fails (peer offline, no AgentHub), skip silently. Cache results. Refresh every 30s and on demand.

**Port convention:** AgentHub daemon HTTP port must be consistent (same binary = same default). Expose the port in settings so users who changed it can configure discovery target port.

**Security:** No new auth needed. Tailscale network membership is the auth layer (established v1.2). Remote AgentHub HTTP API already requires no separate auth for tailnet members.

### Auto-Update — Library Choice

**Use `creativeprojects/go-selfupdate`** (not `rhysd/go-github-selfupdate`). Reasons:
- Fork of rhysd with active maintenance
- Separates detection from installation: `DetectLatest()` returns `*Release{AssetURL, Version}` without touching the binary
- GitHub source provider built-in: `selfupdate.ParseSlug("scottkw/agenthub")`
- Cross-platform asset naming detection (matches `agenthub-{version}-{os}-{arch}.{ext}` convention already in use)

**Update flow:** Background goroutine at startup → `DetectLatest()` → if `latest.GreaterThan(currentVersion)` → emit Wails event with version + release URL → frontend shows badge in WelcomeTab + "Download" button → `runtime.BrowserOpenURL(releasePageURL)`.

**No silent binary replacement.** macOS Gatekeeper quarantines programmatically downloaded binaries. Homebrew and WinGet users update via `brew upgrade` / `winget upgrade`.

### Standard App Menus — Wails API

Wails v2 `menu` package provides:
- `menu.AppMenu()` — macOS Application menu (app name, Preferences, Hide, Quit)
- `menu.EditMenu()` — Edit with Cut/Copy/Paste/Select All/Undo/Redo — **fixes macOS WebView clipboard**
- `menu.WindowMenu()` — Minimize/Zoom/Close

Set via `options.App{Menu: appMenu}` in `wails.Run()`. Dynamic updates via `runtime.MenuUpdateApplicationMenu(ctx)`.

**File menu:** New Session (Cmd+N, emits Wails event to frontend), separator, Quit (Cmd+Q).

**Help menu:** About AgentHub (shows version modal or runtime.MessageDialog), Documentation (opens GitHub), Report Issue (opens GitHub issues).

### Version Injection

**Backend:** `-ldflags "-X github.com/scottkw/agenthub/cmd.Version=$(git describe --tags --always)"` in build.sh and CI. Expose via Wails method binding `GetVersion() string`.

**Frontend:** WelcomeTab.tsx calls `GetVersion()` on mount instead of using hardcoded constant. This also provides the version string for the auto-update comparison.

---

## Competitor Feature Analysis

| Feature | GitHub Desktop | VS Code Remote | Our Approach |
|---------|---------------|----------------|--------------|
| Auto-update | Silent download + restart prompt; shows version in About | Extension auto-update; editor update via prompt | Notify + open browser (avoids Gatekeeper complications) |
| Remote sessions | Not applicable | VS Code Tunnels requires microsoft.com auth | Zero-config via Tailscale peer enumeration — no auth, no accounts |
| App menus | Full standard macOS menus | Full standard macOS menus | Currently missing in v1.8; add in v1.9 via Wails menu API |
| Peer discovery | Not applicable | Server URL required | Automatic via `local.Client{}.Status()` |

---

## Sources

- [Tailscale local package — LocalClient.Status()](https://pkg.go.dev/tailscale.com/client/local) — `Status()` method verified; returns `*ipnstate.Status` with peer map (HIGH confidence)
- [ipnstate package — PeerStatus fields](https://pkg.go.dev/tailscale.com/ipn/ipnstate) — `HostName`, `TailscaleIPs`, `DNSName`, `Online` fields confirmed via GitHub source (HIGH confidence)
- [creativeprojects/go-selfupdate](https://pkg.go.dev/github.com/creativeprojects/go-selfupdate) — `DetectLatest()` API verified, separates detection from install (HIGH confidence)
- [Wails v2 menu package](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/menu) — `EditMenu()`, `AppMenu()`, `WindowMenu()` confirmed (HIGH confidence)
- [Tailscale Services feature](https://tailscale.com/docs/features/services) — alpha endpoint discovery, not used for v1.9 (MEDIUM confidence)
- [Tailscale install docs](https://tailscale.com/docs/install) — per-platform install methods for health modal links (HIGH confidence)
- [Apple macOS keyboard shortcuts](https://support.apple.com/en-us/102650) — standard menu shortcut conventions (HIGH confidence)

---

## Prior Milestone Research (v1.8 — GitHub Distribution & CI/CD)

The v1.8 research covered release-please auto-versioning, multi-platform release pipeline, Homebrew
cask tap (scottkw/homebrew-agenthub), WinGet submission infrastructure, artifact naming conventions,
and packaging templates. All v1.8 features shipped as of 2026-04-06.

Key decisions from v1.8 relevant to v1.9:
- Artifact naming: `agenthub-{version}-{os}-{arch}.{ext}` — used by go-selfupdate asset detection
- GitHub repo: `scottkw/agenthub` — used as `selfupdate.ParseSlug("scottkw/agenthub")`
- release-please PAT requirement — release tags don't trigger downstream workflows with GITHUB_TOKEN
- WinGet submission is async (hours to ~1 day for Microsoft moderation)
- ditto for macOS archive (preserves signing xattrs)

See git history for full v1.8 FEATURES.md content.

---

*Feature research for: AgentHub v1.9 — Remote Sessions, Auto-Update, Menus, and App Polish*
*Researched: 2026-04-06*
