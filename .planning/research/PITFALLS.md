# Pitfalls Research

**Domain:** Go/Wails desktop app — dual-mode networking (Tailscale + local fallback), auto-serve sessions, modal-to-tab conversion, CLI path detection for multiple install methods
**Researched:** 2026-04-08
**Confidence:** HIGH (architecture fully readable from live codebase; external claims verified against official Claude Code docs and Go standard library)

---

## Critical Pitfalls

### Pitfall 1: Dual-Mode Web Server — Two Listeners, One Config Struct

**What goes wrong:**
The current `webserver.Config` embeds Tailscale-specific fields: `BindIP` (Tailscale 100.x.x.x address), `FQDN` (MagicDNS hostname for Let's Encrypt), and a `GetCertificate` hook from `local.Client`. Reintroducing local-network serving by adding an `if !tailscale { useLocalCert }` branch inside `Start()` creates a single method that now silently selects two incompatible code paths. The failure mode is subtle: `Start()` succeeds but `BaseURL()` returns the wrong scheme/host for the active mode, QR codes encode the wrong URL, and the StatusBar shows an unreachable link.

A second failure: the existing `server.go` computes `BaseURL()` using `ws.config.FQDN`. If Tailscale mode returns a `.ts.net` hostname and local mode returns a LAN IP, but both go through the same `BaseURL()` function, one mode will always produce the wrong URL — and the bug will not appear in unit tests that only exercise one code path.

**Why it happens:**
The existing Config struct and Start() method are Tailscale-only by design (v1.2 removed all self-signed infrastructure). When reintroducing local mode, developers modify the existing struct in-place rather than introducing a discriminated union or a mode flag, and the two modes share fields that have incompatible meanings.

**How to avoid:**
- Add an explicit `Mode` field (or a separate `LocalConfig` struct) so callers cannot accidentally supply a `FQDN` for local mode or omit a password for local mode.
- Implement `Start()` as two private methods (`startTailscale`, `startLocal`) with a dispatcher — no conditional branching inside a single method.
- Make `BaseURL()` return a string that is computed lazily from the actual listener address AND the active mode, not from the config struct.
- Unit-test both modes independently: Tailscale tests inject a mock `TLSConfig`; local tests use a real self-signed cert generated in-memory.
- Add a compile-time check that prevents `Config.FQDN` from being non-empty when `Mode == Local`.

**Warning signs:**
- `BaseURL()` returns an FQDN-style URL when the server was started in local mode.
- QR code contains the Tailscale hostname even though Tailscale is not connected.
- `IsWebServerRunning()` returns `true` but `GetWebServerURL()` returns `""` — the two modes disagree on "running" state.

**Phase to address:** Local network fallback phase (first v1.11 phase). The mode abstraction must be locked in before anything else touches the web server.

---

### Pitfall 2: Self-Signed Certificate — P521 Curve Rejected by Browsers

**What goes wrong:**
Go's `crypto/tls/generate_cert.go` generates ECDSA keys with curve P521 by default. P521 is not supported in Chrome, Chromium, or Firefox — the browser returns a cryptic `remote error: tls: illegal parameter` error with no indication that the curve is the cause. Users see a connection error, not a certificate warning, so the "click through the warning" workaround does not apply.

Previously (v1.0), AgentHub's self-signed CA+leaf generation also used P256, but that code was deleted in v1.2 and is not available for copy-paste reference.

**Why it happens:**
Go examples and generate_cert.go documentation do not prominently flag curve support limitations. Developers generate a cert, test it in curl (which accepts P521), and declare it working — without testing in Chrome.

**How to avoid:**
- Generate self-signed certificates using P256 (`elliptic.P256()`), not P521.
- Generate an in-memory CA + leaf pair (two separate certs) rather than a single self-signed cert: some TLS clients reject certs that act as both CA and leaf.
- Test the generated cert by opening it in Chrome immediately during development — do not rely on curl or Go's own TLS client.
- Use `x509.Certificate{IsCA: true, ...}` for the CA cert and a separate leaf cert signed by that CA.
- Store only the in-memory cert (no disk writes); regenerate on each daemon start to avoid stale keys.

**Warning signs:**
- Browser shows `ERR_SSL_VERSION_OR_CIPHER_MISMATCH` or `ERR_BAD_SSL_CLIENT_AUTH_CERT` instead of the expected "certificate is not trusted" warning.
- `tls: illegal parameter` appears in Go server logs.
- Curl succeeds but Chrome fails — this is the P521 signature.

**Phase to address:** Local network fallback phase. Cert generation is the first implementation step; must get the curve right before building anything on top.

---

### Pitfall 3: Password Authentication — Shared State Between Daemon and Web Server

**What goes wrong:**
The local-mode password needs to be:
1. Generated once per daemon start (not re-rolled on each web server start/stop cycle).
2. Accessible to the web server middleware for request validation.
3. Retrievable by the GUI so it can display it to the user.
4. NOT stored in plain text in a config file world-readable at `~/.config/agenthub/`.

The common failure is generating the password inside `StartWebServer()` and storing it only in the `WebServer` struct. When the GUI calls `GetWebServerURL()` or the daemon API returns the URL, the password is not included. Then developers add the password to the URL as a query param (`?pw=...`) — which lands in browser history, server logs, and QR codes that are screen-captured.

A second failure: if the password is rolled on each `StopWebServer/StartWebServer` cycle, existing browser sessions are invalidated without user notification.

**Why it happens:**
Password generation is simple (`crypto/rand`), so it gets added at the most convenient call site rather than the architecturally correct one. Auth middleware is an afterthought.

**How to avoid:**
- Generate the password once in `runDaemonCore()` at daemon startup — it lives as long as the daemon process, not as long as any individual web server instance.
- Pass the password to `WebServer` at construction time (`NewWebServer(cfg, manager, password string)`), not at start time.
- Never embed the password in the URL. Use HTTP Basic Auth or a custom header — Basic Auth is widely supported by browsers (they prompt the user) and by curl.
- Store the password in memory only; never write it to disk.
- Expose `GetLocalPassword()` as a daemon API endpoint so the GUI can show it in the Settings tab without embedding it in every URL.
- Document that rolling the password requires a daemon restart (not just a web server restart) so UX copy is accurate.

**Warning signs:**
- Password appears in `sessionURLs` values stored in React state (it should not be in the URL).
- `GetWebServerURL()` returns different values before and after `StopWebServer/StartWebServer`.
- Auth check is implemented as URL query param comparison rather than header comparison.

**Phase to address:** Local network fallback phase — password design must be part of the initial architecture, not added after the fact.

---

### Pitfall 4: Auto-Serve — Existing Sessions Are Not Retroactively Served

**What goes wrong:**
Auto-serve means "all new sessions are automatically web-enabled when created." If the implementation only changes `CreateSession` to call `EnableSession(id)` after creation, then sessions that exist when the user upgrades (or sessions created before the web server starts) are never auto-served. The user creates a session, the web server starts late, and the session is not served.

A second failure: the current `webEnabled` map in `WebServer` and the `webEnabled` state in React are separate. If auto-serve calls `EnableSession` at the daemon level but the React state is not seeded from the daemon on load, the toggle in the `StatusBar` shows OFF even though the session is being served. Users click it off, accidentally unserving their session.

**Why it happens:**
Auto-serve is typically implemented as "new sessions default to enabled." The existing session restoration on app launch (`init()` → `ListSessions()`) was written for a world where web-enabled state is per-user-toggle. It does not know to treat "all sessions are served" as the default.

**How to avoid:**
- Track auto-serve as a daemon-level setting: when enabled, `ListSessions()` response should include a `webEnabled` field per session that the daemon authoratively knows — not the frontend guessing from local state.
- On `init()`, after `ListSessions()`, also call `GetWebEnabled(sessionId)` for each session (or return it in `ListSessions` response) and seed `webEnabled` state from those values.
- Do not rely on the React state as the source of truth for web-enabled state across window hide/show cycles.
- The nudge banner for Tailscale should not appear for sessions that are already served over Tailscale — distinguish "local mode auto-served" from "Tailscale mode auto-served."

**Warning signs:**
- `StatusBar` shows web toggle OFF for a session that is actually being served.
- Sessions created before the web server starts are never auto-served even when auto-serve is on.
- Restoring the app window after it was hidden shows all sessions as web-disabled even though the daemon has them web-enabled.

**Phase to address:** Auto-serve phase. Must extend `ListSessions` response to include web-enabled state before any frontend auto-serve work.

---

### Pitfall 5: Settings Modal-to-Tab Conversion — State Lifecycle Mismatch

**What goes wrong:**
The current `SettingsPanel` uses `isOpen` as its lifecycle gate: it loads web server state in a `useEffect([isOpen])` hook, returns `null` when `isOpen` is false, and resets local state (selectedPort, serverLoading, ctDisclosed) on each open. Converting it to a persistent sidebar tab means:

1. The component is always mounted — the `useEffect([isOpen])` pattern no longer works. Web server state will not reload when the tab is re-focused.
2. `if (!isOpen) return null` is the current null-check against stale operations. Remove it and async callbacks (handleToggleServer) can fire against unmounted-equivalent state.
3. The Settings tab will appear in `tabs` state as a Tab object, but it has no `sessionId`. Tab-keyed logic (webEnabled, sessionURLs, fontSizes) will not break the Settings tab, but stale state across the settings tab and terminal tabs could expose subtle bugs.
4. The sidebar currently triggers `onSettings` which calls `setShowSettings(true)`. This will need to change to a tab navigation call — similar to `handleOpenDaemonManager`. If the old `setShowSettings` call sites (including the "no CLIs found" path in `handleAddTab`) are not all updated, Settings can open as both a modal overlay and a tab simultaneously.

**Why it happens:**
The modal pattern (mount/unmount on open) is fundamentally different from the tab pattern (always mounted, focus-based visibility). Developers convert the container (`showSettings` → tab navigation) but leave the component's internal lifecycle assumptions unchanged.

**How to avoid:**
- Replace `useEffect([isOpen])` with `useEffect([activeId === SETTINGS_TAB.id])` — load state when tab becomes active, not when a modal opens.
- Remove the `if (!isOpen) return null` guard and replace it with CSS visibility (already the pattern for terminal tabs: `display: isActive ? 'flex' : 'none'`).
- Search all call sites of `setShowSettings(true)` (there are at least two: sidebar `onSettings` and `handleAddTab` when no CLIs are detected) and replace each with the tab navigation pattern.
- Add a `SETTINGS_TAB` constant alongside `WELCOME_TAB`, `DAEMON_MANAGER_TAB`, `REMOTE_SESSIONS_TAB` — the Settings tab is a singleton tab just like the others.
- Verify that `handleSettingsClose` (which calls `IsWebServerRunning()`) is replaced or removed — the tab pattern has no "close" action that needs cleanup.

**Warning signs:**
- Settings opens as an overlay AND a tab simultaneously (old `setShowSettings` path not updated).
- Web server state in Settings tab is stale after the user starts/stops the server from another trigger.
- `handleAddTab`'s "no CLIs" path still calls `setShowSettings(true)` after migration, causing a broken overlay with no backdrop.

**Phase to address:** Settings-as-tab phase. This is a pure frontend refactor; all call sites must be audited before the `SettingsPanel` internals are changed.

---

### Pitfall 6: Claude Code Path Detection — Native Installer at ~/.local/bin Is Not on Default PATH in Service Mode

**What goes wrong:**
The Anthropic native installer places the Claude Code binary at `~/.local/bin/claude` (macOS/Linux) or `%USERPROFILE%\.local\bin\claude.exe` (Windows). `~/.local/bin` is not on the default `PATH` in all shell configurations — and critically, it is NOT added to PATH by macOS launchd or Linux systemd user service units, which is exactly where the AgentHub daemon runs.

The existing `AugmentServicePath()` already handles nvm, Volta, and Homebrew paths. Adding `~/.local/bin` to the augmentation candidates is the correct fix. However, there is a second failure: if `DetectCLIs()` is called before `AugmentServicePath()` runs (e.g., in a test harness), `~/.local/bin/claude` will not be found. The order dependency is invisible because tests set up their own PATH.

**Why it happens:**
`~/.local/bin` is a relatively new convention (XDG Base Directory Specification). Older tools and shell configs do not include it. The native installer does add it to `~/.bashrc` or `~/.zshrc` for interactive shells — but service-mode processes do not source those files.

**How to avoid:**
- Add `filepath.Join(home, ".local", "bin")` to the `candidates` list in `AugmentServicePath()`.
- For Windows, add `filepath.Join(userProfile, ".local", "bin")` — this requires detecting `USERPROFILE` env var, not `HOME`.
- Additionally, check for the Homebrew cask location: `brew --cask claude-code` installs the binary differently from the native installer (typically as a symlink in `/opt/homebrew/bin/` or `/usr/local/bin/`). The existing Homebrew path augmentation in `AugmentServicePath()` already covers this case — no change needed there.
- Test `DetectCLIs()` after `AugmentServicePath()` runs. Add a test that stubs `~/.local/bin` by prepending a temp dir to PATH and verifies `DetectCLIs` returns Claude Code.
- The fix for `AugmentServicePath` is one line; the important discipline is ensuring it runs before `DetectCLIs` in the daemon startup sequence.

**Warning signs:**
- Claude Code is installed and works in the terminal, but AgentHub shows it as "not detected."
- `exec.LookPath("claude")` returns `""` in a subprocess but `which claude` in a terminal returns a path.
- macOS: `claude` is at `~/.local/bin/claude` but AgentHub was launched from the Dock (not a terminal), so PATH does not include `~/.local/bin`.

**Phase to address:** Claude Code detection fix phase. One-line change to `AugmentServicePath` plus a test; low risk but must be verified against both install methods (native and Homebrew).

---

### Pitfall 7: Nudge Banner — Infinite Re-Render and Stale Tailscale State

**What goes wrong:**
The nudge banner ("You're on local network — connect Tailscale for a better experience") needs to:
- Appear when `tailscaleHealth.installed === false` or `tailscaleHealth.connected === false`.
- Disappear when Tailscale connects (health poller fires a `tailscale:health` event).
- Be dismissible per-session (or globally) by the user.

The failure modes are:
1. If the banner is rendered inside the `terminal-wrapper` div (alongside TerminalPanel and StatusBar), adding it will break the flex layout that governs terminal fill — a regression that has been fixed twice already (v1.1 CSS flex chain fix, v1.6 rAF retry loop). The terminal height will shrink by the banner height with no compensating adjustment.
2. If dismiss state is kept only in React local state, every window hide/show cycle resets it — the user dismisses the banner, closes and reopens the app, and it reappears.
3. If the banner listens for `tailscale:health` events but the event fires before the component subscribes (the 5-second startup delay in the health poller is a known pattern), the banner never disappears even after Tailscale connects.

**Why it happens:**
Banner UI is typically slotted inside the nearest parent container. The terminal flex layout is fragile (see PROJECT.md tech debt notes: "double-rAF for initial terminal fit" and "bounded rAF retry loop"). Adding height to that container without accounting for it in the `rows` calculation breaks terminal fill silently.

**How to avoid:**
- Render the nudge banner as a sibling to `app__content`, not inside the `terminal-wrapper`. A fixed-position or sticky banner at the top of the main content area avoids all flex layout interactions.
- Store dismiss state in `localStorage` (like the sidebar collapsed state) so it persists across window hide/show.
- The banner dismissal should be session-scoped (stored as `dismissed-local-nudge: true`) not tab-scoped.
- If the banner triggers a health recheck, gate it with the same TTL as the normal health poller — do not create a second concurrent polling loop.

**Warning signs:**
- Terminal height shrinks after the nudge banner appears.
- rAF retry loop fires extra iterations because `proposeDimensions()` returns stale values after the banner changes layout height.
- Banner reappears on every app launch even after the user dismissed it.

**Phase to address:** Local network fallback phase — the banner is part of the local fallback feature. Its layout placement must be considered alongside the flex chain, not as an afterthought.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Embed password in session URL query param | Easy to share via copy | Password in browser history, logs, QR images | Never — use HTTP Basic Auth or a header |
| Roll password on every `StartWebServer` call | Simple state management | Browser sessions invalidated without warning | Never — tie password to daemon lifetime |
| Regenerate self-signed cert on every `StartWebServer` | Simpler code | New cert on each start means browsers prompt for trust re-acceptance repeatedly | Never — generate once per daemon start, cache in memory |
| Skip self-signed cert in-memory and write to disk | Persistent across restarts | World-readable key file in `~/.config/agenthub/`; stale cert if hostname changes | Never — keep in memory only |
| Leave `setShowSettings(true)` call sites in handleAddTab | Avoids touching unrelated code | Settings opens as modal overlay after modal→tab migration | Never — all call sites must migrate together |
| Reuse existing `webEnabled` React state for auto-serve default | No new API needed | React state diverges from daemon's authoritative state across hide/show cycles | Never — daemon must be authoritative |
| Use P521 curve in self-signed cert (Go default) | Zero extra code | Silent browser TLS handshake failure (not a certificate warning) | Never — always use P256 |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `crypto/tls` self-signed cert | Use default `generate_cert.go` curve (P521) | Explicitly pass `elliptic.P256()` to `ecdsa.GenerateKey` |
| `crypto/tls` self-signed cert | Generate one cert that is both CA and leaf | Generate CA cert + leaf cert signed by CA; store CA for client trust, serve leaf |
| Local web server bind address | Bind to `0.0.0.0` to be reachable on LAN | Bind to the specific LAN IP, not `0.0.0.0` — avoids accidentally serving on Tailscale interface too |
| Dual-mode web server | Check `if tailscale { } else { }` inside `Start()` | Separate code paths at construction time via `Config.Mode` field; `Start()` is mode-unaware |
| `AugmentServicePath()` + Claude native install | Only augment Homebrew and nvm paths | Also prepend `~/.local/bin` (macOS/Linux) and `%USERPROFILE%\.local\bin` (Windows) |
| Settings modal → tab | Leave `isOpen`-triggered `useEffect` in place | Replace with `activeId === SETTINGS_TAB.id` trigger; remove `if (!isOpen) return null` guard |
| Nudge banner in `terminal-wrapper` | Slot banner inside terminal flex container | Render banner as sibling of `app__content`, never inside `terminal-wrapper` |
| HTTP Basic Auth for local password | Roll password per web server start | Password lives with daemon lifetime; `NewWebServer` receives password at construction |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Two concurrent health pollers (existing + nudge banner) | CPU usage creeps up; Tailscale API hammered every few seconds | Banner reads from existing `tailscaleHealth` state, does not start its own poll | Immediately if banner adds a `setInterval` for its own health check |
| Self-signed cert regenerated on each web server start/stop | 50-200ms cert generation delay each time | Generate once at daemon startup; pass to `NewWebServer` | Every start/stop cycle |
| Auto-serve calls `ToggleWebServing` per session at daemon start | N IPC calls on startup for N existing sessions | Daemon tracks "auto-serve enabled" flag natively; no per-session call needed at startup | Grows linearly with session count |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Embed password in session URL | Browser history, logs, shared URLs expose password | HTTP Basic Auth header only; password never in URL |
| Write self-signed key to disk at `~/.config/agenthub/` | World-readable private key; key persists after password is changed | In-memory only; regenerate on daemon start |
| Bind local web server to `0.0.0.0` instead of specific LAN IP | Server reachable on all interfaces including Tailscale | Detect LAN IP explicitly; bind to that address |
| Use same password for all sessions | One compromised session URL exposes all sessions | Single password protects the dashboard; per-session auth is over-engineering |
| Show password in plain text in Settings tab without masking | Screen sharing / shoulder surfing | Default masked; toggle to reveal; never log password |
| Accept connections without validating `Origin` header in local mode | CSRF from a malicious local web page | Add `Origin` check in WebSocket `AcceptOptions` for local mode (unlike Tailscale mode which relies on network-level auth) |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Auto-serve enables web serving without starting the web server | Sessions appear served but no URL is shown; toggle is ON but server is off | Auto-serve and web server start are atomic: starting the server auto-enables all new sessions |
| Nudge banner appears even when user is intentionally using local mode | Feels like a bug report rather than a feature | Banner is dismissible and stays dismissed (localStorage); not shown again unless Tailscale status changes from "was connected" to "not connected" |
| Settings tab does not indicate which mode the web server is in | User cannot tell if they're in Tailscale mode or local mode | Settings "Web Server" tab shows active mode explicitly: "Tailscale (Let's Encrypt)" or "Local Network (self-signed)" |
| "New Tab" label in sidebar misleads users expecting browser behavior | Users expect a browser tab; this creates a new terminal session | Rename to "New Session" (the v1.11 label rename requirement) |
| Local mode password shown once then inaccessible | User forgets password; no recovery without restarting daemon | Password always retrievable from Settings > Web Server tab (masked, with reveal toggle) |
| QR code in local mode encodes an FQDN URL (Tailscale HTTPS) | User scans QR code on mobile; connection fails because Tailscale is not connected | QR code URL must match current active mode; local mode QR uses LAN IP + port |

---

## "Looks Done But Isn't" Checklist

- [ ] **Self-signed cert:** Cert generated and TLS listener starts — verify in Chrome (not curl) that connection shows "not trusted" warning (not a cryptic TLS error)
- [ ] **Self-signed cert curve:** Cert generated — verify with `openssl x509 -in cert.pem -text | grep 'Public Key Algorithm'` that it says `id-ecPublicKey` with `prime256v1`, not `secp521r1`
- [ ] **Local web server bind:** Server starts — verify it is NOT reachable from the Tailscale IP (`curl https://100.x.x.x:7443` should fail) when in local mode
- [ ] **Password auth:** Server starts with password — verify a request without the `Authorization: Basic ...` header returns `401`, not `200`
- [ ] **Auto-serve:** New session created — verify `IsWebEnabled(sessionId)` returns true AND StatusBar shows URL without user clicking the toggle
- [ ] **Auto-serve on restore:** App window re-shown after hide — verify sessions that were auto-served still show as web-enabled in StatusBar
- [ ] **Settings as tab:** Settings tab opens — verify opening Settings via sidebar does NOT show a modal overlay simultaneously
- [ ] **Settings as tab:** "No CLIs found" path — verify `handleAddTab` opens the Settings tab (not a modal overlay) when no CLIs are detected
- [ ] **Settings tab lifecycle:** Tab is always mounted — verify web server state loads when tab becomes active (not just once at mount)
- [ ] **Claude Code native install:** `claude` installed at `~/.local/bin/claude` — verify DetectCLIs returns it when app is launched from Finder (not terminal)
- [ ] **Nudge banner layout:** Banner visible — verify terminal fill is unaffected (no height shrinkage in the terminal panel)
- [ ] **Nudge banner persist:** Banner dismissed — verify it does not reappear after window hide/show cycle
- [ ] **QR code mode consistency:** In local mode, QR button clicked — verify QR encodes a LAN IP URL, not a `.ts.net` URL
- [ ] **"New Session" label:** Sidebar rendered — verify collapsed state shows icon only (no label), expanded state shows "New Session" (not "New Tab")

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Wrong TLS curve (P521) in production | LOW | Change `elliptic.P521()` to `elliptic.P256()` in cert generation; daemon restart regenerates cert |
| Password embedded in URL (in production) | HIGH | Change auth mechanism to Basic Auth header; invalidate all existing shared URLs; bump a minor version with release notes |
| Dual-mode code paths tangled in Start() | MEDIUM | Extract `startTailscale()` and `startLocal()` private methods; no external API change needed |
| Settings opens as both modal and tab | LOW | Find all `setShowSettings(true)` call sites; replace each with tab navigation; remove modal rendering from JSX |
| Auto-serve state diverges from daemon | MEDIUM | Extend `ListSessions` response to include per-session `webEnabled` field; seed React state from that on init |
| Nudge banner breaks terminal layout | LOW | Move banner outside `terminal-wrapper` to sibling position; no Go changes needed |
| Claude Code not detected after native install | LOW | Add `~/.local/bin` to `AugmentServicePath`; daemon restart re-runs detection |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Dual-mode Config struct ambiguity | Local network fallback (first phase) | Both modes have explicit `Mode` field; `BaseURL()` correct for each mode |
| P521 TLS curve rejected by browsers | Local network fallback (cert generation step) | Chrome opens local web terminal with "not trusted" warning (not TLS error) |
| Password in URL | Local network fallback (auth design step) | No `pw=` query param anywhere; Basic Auth header used; `401` returned without credentials |
| Password state vs. daemon lifetime | Local network fallback (password architecture step) | Same password survives `StopWebServer/StartWebServer` cycle |
| Auto-serve state not seeded from daemon | Auto-serve phase | StatusBar shows correct state after app window re-show |
| Settings modal + tab conflict | Settings-as-tab phase | Zero `setShowSettings` calls remain in App.tsx after migration |
| Settings tab lifecycle (`isOpen` → `activeId`) | Settings-as-tab phase | Web server state refreshes when tab is re-focused |
| Claude native install not detected | Claude Code detection phase | Claude Code at `~/.local/bin` found when app launched from Finder |
| Nudge banner breaks terminal flex layout | Local network fallback (banner rendering step) | Terminal fill unchanged after banner appears; no extra rAF iterations |

---

## Sources

- Anthropic Claude Code official docs — installation paths: `~/.local/bin/claude` (native), `~/.local/share/claude` (data), Homebrew and WinGet as alternatives — [code.claude.com/docs/en/setup](https://code.claude.com/docs/en/setup) — HIGH confidence (official docs, verified 2026-04-08)
- Go `crypto/tls` — P521 curve not supported in Chrome/Firefox: `remote error: tls: illegal parameter` — [github.com/golang/go/issues/19901](https://github.com/golang/go/issues/19901) — HIGH confidence (official Go issue)
- Go `crypto/tls` — CA+leaf single-cert scheme rejected by some TLS clients — [go.dev/src/crypto/tls/generate_cert.go](https://go.dev/src/crypto/tls/generate_cert.go) — HIGH confidence (official Go source)
- Project codebase: `internal/webserver/server.go` (Config struct, Start(), BaseURL()), `internal/webserver/tailscale.go` (CheckHealth, TailscaleHealth), `internal/daemon/path.go` (AugmentServicePath, known PATH candidates), `internal/pty/detect.go` (DetectCLIs, LookPath only), `frontend/src/App.tsx` (showSettings state, setShowSettings call sites, webEnabled React state), `frontend/src/components/SettingsPanel.tsx` (isOpen lifecycle pattern), `frontend/src/components/Sidebar.tsx` (New Tab label) — HIGH confidence (live codebase)
- PROJECT.md — v1.2 decision log: "Self-signed certificate infrastructure removed"; v1.1 flex chain fix; v1.6 rAF retry loop — HIGH confidence (project history)

---
*Pitfalls research for: Go/Wails desktop app v1.11 — local network fallback, auto-serve, settings-as-tab, label rename, Claude Code detection*
*Researched: 2026-04-08*
