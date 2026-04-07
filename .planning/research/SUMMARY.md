# Project Research Summary

**Project:** AgentHub v1.9 — Remote Sessions & App Polish
**Domain:** Go/Wails desktop app — tailnet peer discovery, auto-update, Tailscale install assistance, standard app menus
**Researched:** 2026-04-06
**Confidence:** HIGH

## Executive Summary

AgentHub v1.9 is a focused polish and capability milestone on top of a well-established Go/Wails v2 codebase. The core architecture — daemon, Unix socket IPC, Wails GUI, native cgo tray, xterm.js terminals over Tailscale — is already proven. This milestone adds four discrete, bounded capabilities: tailnet peer discovery with a remote session panel, auto-update notification, Tailscale install guidance improvements, and standard macOS app menus. All four fit cleanly into the existing architecture without requiring new frameworks or significant structural changes. The only new direct dependency is `creativeprojects/go-selfupdate@v1.5.2` for the update checker; everything else is already in `go.mod`.

The recommended approach prioritizes hard prerequisites first: fix the hardcoded version string (`-ldflags` injection) before implementing the update checker, and add standard app menus (which fix broken macOS clipboard) before or alongside any other work. Remote session discovery follows the same injectable `statusFunc` pattern already used in the health check system, probing peers via their existing Tailscale HTTPS web server `/api/sessions` endpoint — no new server-side protocol is needed. The update checker uses detect-only mode (no binary self-replacement), surfacing a download link and opening the browser for user-controlled installation.

The two highest-priority risks are both architectural decisions that must be settled before writing code: (1) the daemon API is Unix-socket-only — remote peer probing must target the web server HTTPS endpoint, not the daemon port; and (2) in-process binary self-replacement on macOS breaks notarization and Gatekeeper — the update path must be "notify and open browser," not silent patching. Both risks are fully understood and the mitigations are clear and low-cost to implement correctly.

## Key Findings

### Recommended Stack

The existing stack is unchanged. One new dependency is needed. `creativeprojects/go-selfupdate@v1.5.2` (MIT, released Dec 2025) wraps the GitHub Releases API call and version comparison in a single clean function (`DetectLatest()`), separating detection from installation. It is the actively maintained successor to `rhysd/go-github-selfupdate` and supports macOS universal binary asset naming, which matches the v1.8 release pipeline.

**Core technologies (new additions only):**
- `creativeprojects/go-selfupdate@v1.5.2`: version detection — cleanly separates detection from binary replacement; `DetectLatest()` API verified on pkg.go.dev; active maintenance (Dec 2025)
- `tailscale.com/client/local` (already at v1.96.3): peer enumeration — `local.Client{}.Status()` returns full `PeerStatus` with `HostName`, `TailscaleIPs`, `Online`, `DNSName`; zero new imports
- `github.com/wailsapp/wails/v2/pkg/menu` (already in go.mod): app menus — `AppMenu()`, `EditMenu()`, `WindowMenu()` helpers verified; must guard macOS-only helpers with `runtime.GOOS == "darwin"`

**What not to add:**
- `google/go-github`: full GitHub API client (~50K LOC) for a single releases check — use go-selfupdate instead
- `rhysd/go-github-selfupdate`: predecessor with infrequent maintenance since 2022
- `go-selfupdate UpdateSelf()/UpdateTo()` on macOS: replaces signed binary in-place, breaking notarization
- `fyne.io/systray` or any third-party tray library: confirmed duplicate symbol linker error with Wails AppDelegate (v1.7 history)
- `tsnet`: creates a second Tailscale node; wrong for reading host's existing tailnet state
- Shell scripts for Tailscale detection: `exec.LookPath` + direct `exec.Command("brew", ...)` is safer and more reliable

### Expected Features

**Must have for v1.9 (table stakes):**
- Remote session listing per tailnet peer — core value proposition of multi-machine workflow tool
- Build-time version injection — prerequisite for update checker and accurate Welcome screen; current hardcoded string is tech debt
- Standard Edit menu — broken macOS clipboard (Cmd+C/Cmd+V/Cmd+Z in xterm.js) is a P0 bug without it
- Standard Window and Help menus — convention violations on macOS without them
- Update available notification — expected in any desktop app distributed via GitHub Releases
- Remote session panel in GUI — unified view of local + remote sessions grouped by peer hostname
- Tailscale install link improvements — reduce new user friction with per-platform copyable commands and direct download URLs

**Should have (ship if time permits):**
- Remote session CLI list (`agenthub list --remote`) — power user feature, medium complexity
- Welcome logo rounded corners — CSS `border-radius`, trivial cosmetic polish

**Defer to v1.9.x or v2+:**
- Remote attach from CLI via WebSocket relay — higher complexity, lower urgency; GUI panel delivers the core value
- In-app Tailscale auto-install via subprocess — fails on TTY check, requires sudo; copyable commands are the right approach
- Tailscale Services API integration — alpha as of 2025, not stable
- Peer-to-peer update push — significant security surface, out of scope

### Architecture Approach

The v1.9 additions follow the existing layered pattern without structural change: new `internal/tailnet` package (peer discovery + probe) feeds new daemon API routes (`GET /tailnet/peers`, `GET /tailnet/sessions`), which are exposed via new `DaemonClient` typed methods, consumed by new Wails-bound methods on `App{}`, and rendered in new React components (`RemoteSessionsPanel.tsx`, `UpdateBanner.tsx`). The update checker runs in the GUI process — not the daemon — because update notifications require a user session and the daemon may run as a system service. App menus are a pure `main.go` change: pass `Menu: buildAppMenu(app)` in `options.App{}`.

**Major components (new in v1.9):**
1. `internal/tailnet` — `PeerDiscovery` calls `local.Client{}.Status()` via injectable `statusFunc` (for CI testability without live Tailscale); `ProbePeer` does HTTP GET to remote `/api/sessions` with 2s timeout and `InsecureSkipVerify` for IP-addressed tailnet probe
2. `internal/updater` — `CheckLatest()` via `go-selfupdate`; emits Wails event `update:available`; runs in GUI goroutine on startup; TTL-gated to once per hour with ETag caching
3. Daemon API routes — `GET /tailnet/peers` and `GET /tailnet/sessions` with 30s result cache (`sync.Mutex` + `time.Time`); 5 concurrent goroutine probe pool
4. App Menu — `buildAppMenu()` in `main.go`; macOS-only `AppMenu()` / `EditMenu()` guarded by `runtime.GOOS == "darwin"`; callbacks emit Wails events (`menu:new-session`, `menu:check-updates`)
5. `RemoteSessionsPanel.tsx` — polls `GetTailnetPeers()` on mount and every 30s; shows loading spinner; groups by peer hostname; "Open" button calls `runtime.BrowserOpenURL` to peer's Tailscale HTTPS web terminal URL

### Critical Pitfalls

1. **Wails app menu + existing cgo tray conflict** — Incorrect `AppMenu()`/`EditMenu()` ordering silently breaks all keyboard shortcuts in xterm.js. Required order: `NewMenu()` → `AppMenu.Append(menu.AppMenu())` → `AppMenu.Append(menu.EditMenu())` → then custom submenus. Test Cmd+C/Cmd+V in terminal immediately after adding the menu.

2. **Remote daemon API is Unix-socket-only** — The daemon control API is not accessible from remote peers. Remote peer probing must target the web server's existing Tailscale HTTPS `/api/sessions` endpoint, not any daemon port. This is a design decision that must be settled before any discovery code is written; retrofitting is HIGH cost.

3. **In-process binary replacement fails on macOS and Windows** — macOS Gatekeeper rejects replacement of a notarized inner binary; Windows locks the running `.exe`. The update flow must be: detect → notify → open GitHub releases page in browser. Never call `UpdateSelf()` or `UpdateTo()` on macOS or Windows.

4. **Peer probe blocking UI / no timeout** — `local.Client{}.Status()` returns all ever-seen tailnet peers; `Online: true` means connected to control plane only, not that AgentHub is running. Probes must be concurrent (`sync.WaitGroup` + goroutine pool capped at ~5), each with `http.Client{Timeout: 2s}`. Run entirely in background goroutine; emit results via Wails events as they arrive.

5. **GitHub Releases API rate limiting** — 60 unauthenticated requests/hour per IP; shared NAT exhausts quota quickly. Gate checks with a persisted last-check timestamp; skip if checked within 1 hour. Use `If-None-Match` ETag. Handle 429 and all non-200 responses silently (log only, no error dialog).

## Implications for Roadmap

Based on the dependency graph from ARCHITECTURE.md and the pitfall-to-phase mapping from PITFALLS.md:

### Phase 1: Standard App Menus + Version Injection
**Rationale:** Both are self-contained with no external dependencies on other v1.9 work. Version injection is the explicit prerequisite for the update checker. The Edit menu is a P0 UX bug fix (broken clipboard in terminal). These ship independently and deliver immediate user value on day one of the milestone.
**Delivers:** Working macOS keyboard shortcuts in xterm.js terminal; correct version display in Welcome tab; File/Edit/Window/Help menus
**Addresses:** Build-time version injection, standard Edit/Window/Help menus, Welcome screen version display, logo rounded corners
**Avoids:** Pitfall 1 (menu/tray conflict) — test Cmd+C/Cmd+V post-implementation before marking done; Pitfall 8 (VERSION blank in dev) — add `"dev"` fallback in Go code

### Phase 2: internal/tailnet Package (Peer Discovery + Probe)
**Rationale:** This is a pure Go package with no UI dependencies; it builds and fully tests independently. The injectable `statusFunc` pattern mirrors the existing `internal/webserver/tailscale.go` and enables CI testing without a live Tailscale daemon. This is the start of the critical path (C → E → F in ARCHITECTURE.md build order).
**Delivers:** Tested `DiscoverPeers()` and `ProbePeer()` functions with full coverage; defines `PeerInfo` and `RemoteSessionInfo` types
**Implements:** `internal/tailnet/discovery.go`, `internal/tailnet/probe.go`
**Avoids:** Pitfall 2 (blocking probe / no timeout) — concurrent goroutines + 2s timeout enforced from the start; Pitfall 3 (wrong probe target) — probe web server HTTPS, not daemon port

### Phase 3: internal/updater Package + Update Notification UI
**Rationale:** Depends on Phase 1 (needs real version string for comparison). Independent of peer discovery. Builds `internal/updater`, adds `UpdateBanner.tsx`, wires the Wails binding and startup goroutine. TTL caching must be implemented here, not deferred.
**Delivers:** Non-blocking update check at startup; `UpdateBanner` in Welcome tab with version + download link; TTL-gated to once per hour with ETag caching
**Implements:** `internal/updater/updater.go`, `frontend/src/components/UpdateBanner.tsx`, `App.CheckForUpdate()` binding
**Avoids:** Pitfall 4 (binary replacement on macOS) — detect only, open browser; Pitfall 5 (rate limiting) — 1-hour TTL + ETag + silent 429 handling

### Phase 4: Daemon API Routes + Remote Session GUI Panel
**Rationale:** Depends on Phase 2 (needs `internal/tailnet`). Adds daemon routes, `DaemonClient` typed methods, and `RemoteSessionsPanel.tsx`. The 30s result cache on the daemon side is mandatory, not optional. This is the final step of the critical path and the primary user-facing deliverable of v1.9.
**Delivers:** Remote sessions visible in GUI grouped by peer hostname; 30s background refresh; loading spinner; "Open" button launches web terminal in browser; local and remote sessions clearly separated
**Implements:** `GET /tailnet/peers`, `GET /tailnet/sessions` daemon routes; `RemoteSessionsPanel.tsx`; `App.GetTailnetPeers()` Wails binding
**Avoids:** Pitfall 2 (blocking startup) — run discovery in goroutine, show loading state; Pitfall 3 (wrong API) — Unix socket for local control only, web server HTTPS for remote probe

### Phase 5: Tailscale Install Guidance Enhancement
**Rationale:** Smallest scope; enhances the existing health modal with direct download links and copyable install commands per platform. No subprocess execution. Can slot anywhere in the schedule.
**Delivers:** Per-platform install commands with copy-to-clipboard in health modal; direct download URLs for macOS, Linux, Windows
**Avoids:** Pitfall 6 (brew subprocess) — show copyable text only, no `exec.Command("brew", "install", ...)`

### Phase 6: Remote Session CLI List (v1.9.x — if time permits)
**Rationale:** Depends on Phase 4 being reliable. Extends the existing `agenthub list` command to aggregate local + remote sessions. Lower urgency since the GUI panel delivers the same value for most users.
**Delivers:** `agenthub list --remote` or unified list with hostname column
**Implements:** Extend existing CLI list command via `DaemonClient.GetTailnetSessions()`

### Phase Ordering Rationale

- Phases 1 and 2 are independent — build in parallel. Neither blocks the other.
- Phase 3 must follow Phase 1 (needs real version string from ldflags).
- Phase 4 must follow Phase 2 (needs `internal/tailnet` package to exist).
- Phase 5 is independent — slot wherever it fits in the schedule.
- Phase 6 depends on Phase 4 being stable and proven; treat as a follow-on within the v1.9.x window.
- Critical path: Phase 2 → Phase 4. Ship Phases 1 and 3 on the fast track.

### Research Flags

Phases with standard, well-documented patterns (no additional research needed):
- **Phase 1 (Menus + Version):** Wails menu API fully verified on pkg.go.dev; ldflags injection is standard Go build practice; injectable statusFunc pattern proven in existing codebase
- **Phase 5 (Tailscale install guidance):** Enhancement to existing modal; no new patterns or APIs

Phases that may need targeted validation during implementation:
- **Phase 3 (Updater):** Validate that `go-selfupdate ParseSlug("scottkw/agenthub")` correctly matches v1.8 release artifact naming convention before wiring; confirm `ChecksumValidator` integration with existing `checksums.txt` file
- **Phase 4 (Remote sessions):** TLS probe with `InsecureSkipVerify` on tailnet IPs is architecturally sound but confirm that Tailscale Let's Encrypt certs are FQDN-only (no IP SANs) before treating `InsecureSkipVerify` as required

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All APIs verified on pkg.go.dev; single new dependency with clean track record; existing stack inspected directly in codebase |
| Features | HIGH | Feature set is well-scoped; table stakes vs. defer lines are clear; competitor analysis consistent with approach |
| Architecture | HIGH | Existing codebase inspected directly (api.go, client.go, types.go, app.go, main.go, tailscale.go, server.go); new components follow proven existing patterns |
| Pitfalls | HIGH | All pitfalls verified against live code (Unix socket binding, relay binding to 127.0.0.1) and official docs (Wails menu ordering, GitHub rate limits, macOS Gatekeeper) |

**Overall confidence:** HIGH

### Gaps to Address

- **`go-selfupdate` asset naming match:** The update checker's `ParseSlug("scottkw/agenthub")` relies on artifact names matching `agenthub_{goos}_{goarch}.{ext}`. Confirm v1.8 release artifacts use exactly this naming convention before finalizing the updater. If naming differs, either rename artifacts in the release pipeline or configure a custom `AssetSuffix`.
- **`InsecureSkipVerify` scope for tailnet probes:** The architecture recommends `InsecureSkipVerify: true` for IP-addressed tailnet probes because Let's Encrypt certs are FQDN-based. Confirm Tailscale does not include IP SANs in the cert; if it does, standard TLS verification can be used instead.
- **Windows and Linux tray stubs:** Adding app menus must not break `tray_linux.go` / `tray_windows.go` stubs. Verify compilation on all three OS targets after Phase 1; the build matrix should cover all platforms before merging.

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/tailscale.com/client/local` — `Status()`, `WhoIs()` methods; `PeerStatus` fields: HostName, TailscaleIPs, DNSName, Online, Active, OS
- `pkg.go.dev/tailscale.com/ipn/ipnstate` — full `PeerStatus` struct fields confirmed via GitHub source
- `pkg.go.dev/github.com/creativeprojects/go-selfupdate` — `DetectLatest()` API, v1.5.2 (Dec 2025), GitHub source provider, `ChecksumValidator`
- `pkg.go.dev/github.com/wailsapp/wails/v2/pkg/menu` — `AppMenu()`, `EditMenu()`, `WindowMenu()`, `AddText`, `AddSeparator`, `AddSubmenu` builder API
- `tailscale.com/docs/install` — per-platform install commands (brew cask, winget, install.sh)
- `formulae.brew.sh/cask/tailscale-app` — `tailscale-app` cask name confirmed
- `docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api` — 60 req/hr unauthenticated limit
- Project codebase direct inspection — `internal/daemon/api.go`, `client.go`, `types.go`, `app.go`, `main.go`, `internal/webserver/tailscale.go`, `internal/webserver/server.go`

### Secondary (MEDIUM confidence)
- `wails.io/docs/reference/menus/` — menu ordering requirements; confirmed via search (direct fetch 403)
- `wails.io/docs/reference/runtime/menu/` — `MenuSetApplicationMenu`, `MenuUpdateApplicationMenu` runtime API
- `github.com/wailsapp/wails/issues/3865` — menu + systray conflict in Wails v3 alpha (awareness note; v1.9 uses v2)
- `github.com/wailsapp/wails/pull/3847` — correct macOS AppMenu/EditMenu ordering fix in Wails
- `github.com/golang/go/issues/21997` — Windows `os.Rename()` fails on running binary

### Tertiary (LOW confidence — needs validation during implementation)
- Tailscale Let's Encrypt cert scope (FQDN-only vs. IP SAN) — inferred from architecture; confirm before finalizing probe implementation
- v1.8 release artifact naming convention vs. go-selfupdate asset detection — confirm before finalizing updater implementation

---
*Research completed: 2026-04-06*
*Ready for roadmap: yes*
