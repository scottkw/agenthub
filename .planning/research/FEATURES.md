# Feature Research

**Domain:** Terminal multiplexer desktop app with web serving for AI coding CLIs
**Researched:** 2026-03-20 (v1.2 Tailscale-only networking added; v1.1 and v1.0 preserved below)
**Confidence:** HIGH (Tailscale Go API verified via pkg.go.dev; existing codebase read directly)

---

## v1.2 Milestone: Tailscale-Only Networking

### Scope

This section covers only what is NEW in v1.2. Existing app ships: tabbed terminal sessions,
web serving with self-signed TLS, password/token auth, QR codes, VPN interface binding
(generic), and health status indicators. Research focus: Tailscale health checks, Let's
Encrypt cert provisioning via Tailscale, Tailscale-only networking, removal of password/token
auth and self-signed CA.

---

### Table Stakes (Users Expect These for v1.2)

Features that must work correctly for the milestone to feel complete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Tailscale health check: installed / daemon reachable | App must know if `tailscaled` is reachable before attempting to serve; raw TLS errors are not acceptable failure output | LOW | `local.Client.StatusWithoutPeers(ctx)` — if it returns an error, tailscaled is not running or Tailscale is not installed. These two states must be distinguished: binary absent vs. socket missing vs. daemon not running. |
| Tailscale health check: connected to tailnet | App must verify `BackendState == "Running"` before binding to Tailscale IP or requesting certs | LOW | `ipnstate.Status.BackendState` field; values are "NoState", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting", "Running". Only "Running" is safe to proceed. Show state-specific instructions for each non-Running state. |
| Tailscale health check: HTTPS certs enabled | `GetCertificate` will fail if HTTPS certs are not enabled in the tailnet admin console — this is a user action required in a web UI, not something the app can do | MEDIUM | `ipnstate.Status.CertDomains` — non-empty means certs are provisioned for this node. Empty means the user has not enabled HTTPS in their tailnet DNS settings (two sub-steps: enable MagicDNS, then enable HTTPS certificates). |
| Instructional modal for health check failures | Users who fail health checks need actionable steps, not raw errors | MEDIUM | Three distinct failure states need distinct instructions: (1) not installed — link to tailscale.com/download; (2) not connected — instructions to open Tailscale and connect; (3) certs not enabled — exact path in admin console (DNS page → HTTPS Certificates → Enable HTTPS). Modal must include a "Check Again" button that re-runs health checks without restarting the app. |
| Let's Encrypt cert via `local.Client.GetCertificate` | Replaces self-signed CA+leaf pattern; provides browser-trusted HTTPS on the `<hostname>.<tailnet>.ts.net` domain with automatic renewal | MEDIUM | `GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)` is the correct `tls.Config.GetCertificate` callback signature. Set it on the TLS config and tailscaled handles caching and renewal automatically. Stable API per pkg.go.dev. FQDN is `Status.Self.DNSName` (trim trailing dot). |
| Bind exclusively to Tailscale interface IP | Web server must only listen on the Tailscale CGNAT address (100.64.0.0/10), not 0.0.0.0, since auth is being removed | LOW | Existing `IsTailscaleIP()` and `ListInterfaces()` already detect Tailscale IPs. Auto-select the first interface where `IsTailscaleIP` is true; remove the user-facing interface picker dropdown. This binding is what makes auth removal safe. |
| Remove password auth and session tokens | When Tailscale is the trust boundary, password/token auth is redundant friction; deleting it simplifies the codebase materially | MEDIUM | Delete `auth.go`, `tokens.go`, all middleware (`dashboardAuth`, `sessionAuth`). Routes become open. The Tailscale network layer (node identity + ACLs) is the access control. Only web-enabled sessions are served, as before. |
| Remove self-signed CA infrastructure | Self-signed certs are incompatible with the Let's Encrypt model; keeping both creates dead code and confusing failure modes | LOW | Delete `tls.go` (CA generation, leaf cert generation). `NewWebServer` no longer calls `LoadOrCreateCA`. `Config.ConfigDir` field can be removed (was only needed for CA persistence). |
| Remove generic VPN interface selection UI | With Tailscale-only networking, there is no interface choice to present; the dropdown adds confusion | LOW | Frontend settings panel currently exposes an interface picker. Replace with a Tailscale status indicator showing connected IP, hostname, and health state. |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Per-OS instructional text in health check modal | Most tools show a raw error string; platform-specific guidance ("Open the Tailscale menu bar app" vs. "Run `sudo tailscale up`") is materially better UX | MEDIUM | Detect at runtime via `runtime.GOOS`. macOS: "Open the Tailscale menu bar app and click Connect". Linux: "Run `sudo tailscale up` in a terminal". Windows: "Open Tailscale from the system tray". Pass the OS hint to the React modal via Wails binding. |
| Auto-derive HTTPS URL from tailnet hostname | User never has to configure a domain; the app reads `Status.Self.DNSName` and constructs the shareable URL automatically | LOW | `DNSName` has a trailing dot (e.g. `hostname.tail12345.ts.net.`); trim it. Construct `https://<DNSName>:<port>`. Display as the shareable URL and use for QR code generation. This is a better URL than the raw Tailscale IP. |
| Health check on web-serve toggle | Re-run health checks at the moment the user enables web serving, not just at startup | LOW | Prevents confusing TLS errors if the user starts the app with Tailscale connected, then Tailscale disconnects, then they try to enable a session. Return to health check modal state rather than serving a broken TLS endpoint. |
| Certificate renewal handled transparently | Users never see cert expiry; no manual `tailscale cert` command needed | LOW | This is free behavior from `local.Client.GetCertificate` — tailscaled handles caching and auto-renewal. Explicitly better than the previous self-signed CA which required users to manually install a CA cert in their browser. Worth calling out in UI copy. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Fallback to self-signed certs when Tailscale certs fail | "Keep serving even if cert provisioning fails" | Silently serves untrusted content; browser shows cert warning that looks like an app bug; creates two parallel TLS code paths to maintain | Fail clearly with the health check modal. If certs are unavailable, web serving is disabled with a clear actionable message. No silent fallbacks. |
| "Try anyway" option to bypass health checks | Power users want to skip setup steps | Leads to confusing TLS errors that erode trust in the app's reliability; the three-step health check process is fast and should not be bypassable | Keep the "Check Again" button. If the check passes, proceed immediately. No bypass mode. |
| Keep password auth as optional extra security | "Tailscale + password = more secure" | Adds complexity to an auth model being intentionally simplified; Tailscale ACLs are the correct tool for fine-grained access control on a tailnet | Tailscale network-level trust is the boundary for v1.2. If per-user access control is needed, that is a future milestone with explicit scope. |
| Support non-Tailscale VPN interfaces alongside Tailscale | Preserves v1.0 behavior for non-Tailscale users | Splits the trust model; makes health checks ambiguous; doubles TLS cert management surface area; the generic VPN path used self-signed certs which are now being removed | v1.2 is explicitly Tailscale-only. The generic VPN path is removed. Document this as a breaking change for non-Tailscale users in release notes. |
| Tailscale Funnel integration for public internet access | Allow sessions to be publicly accessible via Tailscale Funnel | Funnel exposes services to the public internet, which is a different threat model from tailnet-only; mixing it with the trust-boundary-is-tailscale model creates security confusion | Defer to a future milestone with explicit scope definition and separate UX for "public" vs. "tailnet-only". |

---

### Feature Dependencies (v1.2)

```
[Tailscale installed / daemon reachable]
    └──required by──> [Tailscale connected check]
                          └──required by──> [HTTPS certs enabled check]
                                                └──required by──> [GetCertificate TLS config]
                                                                      └──required by──> [Bind to Tailscale IP]
                                                                                            └──required by──> [Remove password/token auth (safe)]

[Instructional modal]
    └──depends on──> [All three health check states] (must surface distinct guidance per failure)

[Auto-derive URL from DNSName]
    └──requires──> [Tailscale connected check] (DNSName is only populated in Status.Self when BackendState == "Running")

[Remove self-signed CA] ──safe only when──> [GetCertificate TLS config] (no other TLS source)
[Remove auth middleware] ──safe only when──> [Bind to Tailscale IP] (binding to 0.0.0.0 with no auth is a regression)
[Remove interface picker UI] ──replaces with──> [Tailscale status indicator]
```

#### Dependency Notes

- **Health checks are strictly ordered.** Cannot check cert enablement without confirming connection first; cannot confirm connection without confirming installation. Present all three as sequential steps in the modal.
- **Auth removal requires interface binding to be locked down first.** The current auth system is the only thing preventing arbitrary access from non-Tailscale IPs. Removing auth before the bind address is Tailscale-only is a security regression.
- **`GetCertificate` requires both MagicDNS and HTTPS certs enabled in admin console.** These are two separate toggles in the tailnet DNS settings. The health check for "certs enabled" covers both by checking `len(Status.CertDomains) > 0`.
- **`local.Client` package is the current API.** `tailscale.com/client/tailscale` (module-level functions) is deprecated; use `tailscale.com/client/local.Client` struct methods directly. `CertPair`, `CertPairWithValidity`, and `GetCertificate` are all marked stable API.

---

### MVP Definition (v1.2)

#### Must Ship

- [ ] Health check: `local.Client.StatusWithoutPeers()` error detection (tailscaled reachable)
- [ ] Health check: `BackendState == "Running"` (connected to tailnet)
- [ ] Health check: `len(CertDomains) > 0` (HTTPS certs enabled in tailnet)
- [ ] Instructional modal with per-state failure guidance and "Check Again" button
- [ ] `tls.Config.GetCertificate` wired to `local.Client.GetCertificate`
- [ ] Auto-select Tailscale IP as bind address; remove interface picker UI
- [ ] Auto-derive HTTPS URL from `strings.TrimSuffix(Status.Self.DNSName, ".")`
- [ ] Delete `auth.go`, `tokens.go`, `tls.go` and all auth middleware
- [ ] Remove `GET /ca.crt`, `POST /login`, `POST /api/sessions/{id}/token` routes
- [ ] Replace interface picker in settings UI with Tailscale status indicator

#### Add After Validation (v1.2.x)

- [ ] Per-OS instructional text in health check modal — add after confirming basic modal works on all three platforms
- [ ] Health check on web-serve toggle — add if users report confusion from mid-session Tailscale disconnects

#### Future Consideration (v2+)

- [ ] Tailscale Funnel integration for public access — requires separate scoping; different trust model
- [ ] Tailscale ACL-based per-session access control — only needed if multi-user tailnet sharing becomes a use case

---

### Feature Prioritization Matrix (v1.2)

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Health checks (all three states) | HIGH | LOW | P1 |
| Instructional modal | HIGH | MEDIUM | P1 |
| `GetCertificate` wiring | HIGH | LOW | P1 |
| Bind to Tailscale IP (auto-select) | HIGH | LOW | P1 |
| Remove password/token auth | HIGH | MEDIUM | P1 |
| Remove self-signed CA | MEDIUM | LOW | P1 |
| Auto-derive URL from DNSName | MEDIUM | LOW | P1 |
| Remove interface picker UI | MEDIUM | LOW | P1 |
| Per-OS modal instructions | MEDIUM | LOW | P2 |
| Health check on web-serve toggle | MEDIUM | LOW | P2 |

---

### Implementation Notes (v1.2)

**Health check API (Go)**

```go
import "tailscale.com/client/local"

client := &local.Client{}

// Check 1: tailscaled reachable
st, err := client.StatusWithoutPeers(ctx)
if err != nil {
    // tailscaled not running or not installed
    // Distinguish: check for binary via exec.LookPath("tailscale")
}

// Check 2: connected to tailnet
if st.BackendState != "Running" {
    // NeedsLogin, Stopped, Starting — show state-specific message
}

// Check 3: HTTPS certs enabled
if len(st.CertDomains) == 0 {
    // User must enable HTTPS in tailnet admin console (DNS page)
}

// FQDN for URL construction
fqdn := strings.TrimSuffix(st.Self.DNSName, ".") // e.g. "host.tail12345.ts.net"
```

**TLS config wiring (Go)**

```go
client := &local.Client{}
tlsCfg := &tls.Config{
    GetCertificate: client.GetCertificate, // handles caching + auto-renewal
}
ln, err := tls.Listen("tcp", tailscaleBindAddr, tlsCfg)
```

**What gets deleted from the existing codebase**

| File / Component | Reason |
|-----------------|--------|
| `internal/webserver/auth.go` | Password/session cookie management |
| `internal/webserver/tokens.go` | Per-session shareable tokens |
| `internal/webserver/tls.go` | Self-signed CA and leaf cert generation |
| `auth_test.go`, `tokens_test.go`, `tls_test.go` | Tests for deleted code |
| `GET /ca.crt` route | CA cert download endpoint |
| `POST /login` route | Login endpoint |
| `POST /api/sessions/{id}/token` route | Token creation endpoint |
| `dashboardAuth` middleware in `server.go` | All routes become open |
| `sessionAuth` middleware in `server.go` | Access control delegated to Tailscale |
| `ListInterfaces()` usage in settings UI | No longer needed for user selection |
| `Config.ConfigDir` field | Only needed for CA persistence |

**What gets added**

| Component | Notes |
|-----------|-------|
| `internal/tailscale/` (new package) | Wraps `local.Client` status checks; returns typed result with three states and FQDN |
| Health check modal in React frontend | Three-state UI with per-step guidance; "Check Again" button; optional per-OS text |
| `GetCertificate` in `server.go` | Replace `GenerateLeafCert` + `BuildTLSConfig` with `local.Client.GetCertificate` |
| Tailscale status indicator in settings UI | Shows connected IP, hostname, health state; replaces interface picker |

---

### Sources (v1.2)

- [Tailscale local client Go API — pkg.go.dev/tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) (HIGH — official package docs, stable API markers present)
- [ipnstate.Status fields — pkg.go.dev/tailscale.com/ipn/ipnstate](https://pkg.go.dev/tailscale.com/ipn/ipnstate) (HIGH — official package docs)
- [Enabling HTTPS — tailscale.com/kb/1153/enabling-https](https://tailscale.com/kb/1153/enabling-https) (HIGH — official Tailscale docs)
- [Tailscale TLS certs blog — tailscale.com/blog/tls-certs](https://tailscale.com/blog/tls-certs) (HIGH — official Tailscale blog)
- Existing codebase read directly: `internal/webserver/server.go`, `auth.go`, `tokens.go`, `tls.go`, `network.go`

---

## v1.1 Milestone: Polish & Build (Preserved)

This section covers features added in v1.1. v1.0 feature landscape is preserved below.

### Context: What v1.0 Already Ships

- Tabbed terminal UI (xterm.js), tab creation/close, inline rename via double-click
- Settings modal: CLI path overrides + web serving (single scrolling panel)
- Web serving: TLS, auth, per-session tokens, QR codes
- `web-serving-bar` overlay above terminal: Web On/Off toggle, URL, Copy Token Link, QR button
- Status dot per tab (running/waiting/idle/errored), status event subscription
- GitHub Actions CI matrix: macOS (signed+notarized), Linux (two webkit variants), Windows (NSIS)
- `ResizeObserver` + `requestAnimationFrame` driving `fitAddon.fit()` on active tab

---

### Table Stakes (Users Expect These for v1.1)

Features expected in a polished v1.1 given what v1.0 shipped. Missing these degrades perceived quality.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Build script (`build.sh`)** | Developers need one-command local builds for release; currently only possible via CI. | MEDIUM | Wails: `wails build -platform <target>`; macOS: `codesign` + `xcrun notarytool`; env-gated so unsigned local builds work without certs. Cannot cross-compile macOS from Linux/Windows (hard constraint: CGo + WebKit). |
| **Terminal full-fill fix** | Terminals that don't fill available space look broken. `web-serving-bar` overlay above the terminal takes height without being accounted for in the flex layout. | LOW | CSS fix: `.terminal-wrapper` flex column; status bar `flex-shrink: 0`; terminal `flex: 1; min-height: 0`. `ResizeObserver` fires `fitAddon.fit()` automatically. |
| **Larger toolbar buttons** | Current `+` and gear buttons are explicitly noted as too small in PROJECT.md. Touch/click targets should be 36–44px minimum (Apple HIG, Material guidelines). | LOW | CSS only. No logic changes. |
| **Per-tab status bar** | The `web-serving-bar` is conditional (only shows when web server running) and ad-hoc. A permanent status bar is expected in any app with per-tab state: always present, always shows current tab's status and controls. | MEDIUM | Replace `web-serving-bar` with permanent single-line strip below tab bar. Always rendered — height is stable, so terminal fit is deterministic. When web serving is off/unconfigured: show muted status text. |
| **Settings modal declutter** | Current panel is a single long scroll mixing CLI paths and web serving. Sectioned/tabbed settings modals are the standard for any desktop app with 2+ configuration domains. | LOW | CSS + React layout only; no backend changes. Two sections: "CLI Paths" and "Web Serving". Can use a tab strip at top of modal. Footer should be "Close" only — saves are per-field. |
| **Tab renaming → web dashboard** | `RenameSession` already works on desktop. The web dashboard session list shows session names. Users expect the rename to propagate — otherwise the dashboard shows stale names. | LOW | `RenameSession` Go method exists; `/api/sessions` response includes session name field; dashboard `renderSessions()` reads `session.name`. Likely already works — verify `json:"name"` tag on session struct. |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **New-session modal with agent picker + folder browser** | Replaces the bare CLI dropdown overlay. A proper modal with agent picker + folder browser (with "Browse" native OS dialog) is how IDEs handle new-terminal workflows (VS Code, JetBrains). Eliminates the "which folder is this agent running in?" confusion. | MEDIUM | Wails `runtime.OpenDirectoryDialog` provides native OS folder picker (one Go line, zero React file tree). Go `CreateSession` needs a `workDir string` parameter passed to PTY spawn `Cmd.Dir`. React modal: agent picker, directory field, Browse button, name field (pre-filled from folder basename), Create button. Last-used folder in `localStorage`. |
| **Per-tab SHIFT+/SHIFT- font size** | Lets each terminal have its own font size. Power users adjust per session (dense code view vs. comfortable reading). xterm.js supports per-instance `options.fontSize`; `attachCustomKeyEventHandler` intercepts Shift++ and Shift+- before they reach the PTY. After size change, `fitAddon.fit()` reflows cols/rows. | LOW | Intercept: `e.shiftKey && (e.key === '=' \|\| e.key === '+')` and `e.shiftKey && e.key === '-'`. Return `false` to prevent PTY passthrough. Bounds: 8–32px. Persist per-sessionId in `localStorage` as `agenthub.fontSize.<sessionId>`. |
| **Web dashboard visual refresh** | Current dashboard is functional but minimal: plain list items, no status colors, no CLI badge, no visual hierarchy. A card-based layout with status indicators, CLI type badge, and copy-link/QR actions makes the remote experience feel first-class. | MEDIUM | Pure HTML/CSS/JS in `web/dashboard.html` (no framework). Status and name already in `/api/sessions`. Cards replace `<li>` items. Status dot colors match desktop app convention. |

### Anti-Features (v1.1)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Folder browser built in React (custom file tree)** | Polished look | Massive scope: need Go FS APIs, permissions, pagination, cross-platform path separators. Fragile on large dirs. | Use `runtime.OpenDirectoryDialog` (Wails native OS dialog) — one line of Go, zero React tree code, native OS appearance. |
| **Settings as a separate page/route** | Cleaner than modal | Wails is a single-page app; routing adds complexity, back-navigation confusion. Settings are infrequent-access. | Tabbed/sectioned modal. Keyboard-dismissable. Same pattern as VS Code settings overlay. |
| **Font size stored server-side per session** | Persist across restarts | Font size is a local display preference, not session data. Backend doesn't need it. Synchronizing over WebSocket adds protocol complexity. | `localStorage` keyed by sessionId. Zero backend changes. Persists across app restarts. |
| **Auto-fit font size to fill terminal container** | "Fill the space perfectly" | FitAddon explicitly does not support this. Implementing font-size-to-fit requires iterative binary search, triggers layout thrash, and still only fits one dimension. This is a known xterm.js limitation. | Fixed user-controlled font size + fitAddon for cols/rows. This is the industry standard (VS Code, Hyper, Warp all work this way). |
| **macOS cross-compilation from Linux/Windows** | Single build host for all platforms | Hard constraint: Wails uses CGo + WebKit2GTK which cannot cross-compile to macOS. Apple notarization also requires a macOS host. | GitHub Actions macOS runner for macOS builds; `build.sh` auto-detects host platform and builds only the matching target. |
| **Global font size (all tabs)** | Simpler UX | Negates the per-tab differentiator. Some users want dense in one tab and readable in another. | Per-tab font size. A "reset all to default" button in settings can satisfy the "reset global" need. |

---

### Feature Dependencies (v1.1)

```
[build.sh]
    uses──> wails build -platform <target>
    uses──> codesign + xcrun notarytool  (macOS only, env-gated)
    requires──> APPLE_DEVELOPER_ID, NOTARIZATION_* env vars for signing

[Terminal full-fill fix]
    enables──> [Per-tab status bar]  (stable height required for deterministic fit)
    uses──> existing ResizeObserver + fitAddon pattern (no logic changes)

[Per-tab status bar]
    requires──> [Terminal full-fill fix]  (status bar height must be stable before fit)
    replaces──> web-serving-bar overlay
    enhances──> tab renaming display  (status bar shows current tab name)

[New-session modal]
    requires──> Go: OpenDirectoryDialog bound method (new)
    requires──> Go: CreateSession updated to accept workDir string (new param)
    enhances──> tab naming  (name pre-filled from folder basename)
    persists via──> localStorage: agenthub.lastWorkDir

[Per-tab font size SHIFT+/SHIFT-]
    requires──> [Terminal full-fill fix]  (font size change triggers refit)
    uses──> term.attachCustomKeyEventHandler (existing xterm.js API)
    requires──> fitAddon.fit() called after options.fontSize update
    persists via──> localStorage: agenthub.fontSize.<sessionId>

[Tab renaming → dashboard]
    depends on──> RenameSession Go method  (already exists)
    depends on──> /api/sessions returning name field  (verify json tag)

[Web dashboard visual refresh]
    depends on──> existing /api/sessions JSON endpoint  (status + name already present)
```

#### Dependency Notes (v1.1)

- **Status bar requires terminal full-fill fix first:** The status bar has a fixed pixel height that reduces the terminal container height. `fitAddon.fit()` must fire after the status bar mounts. The existing `ResizeObserver` handles this automatically if the status bar is inside the observed container hierarchy.
- **New-session modal requires backend change:** Current `CreateSession(cliName, defaultName string)` has no `workDir` parameter. This is a Go + TypeScript binding change — both backend and generated wailsjs bindings must be updated.
- **Font size change requires immediate refit:** After `term.options.fontSize = newSize`, cell pixel dimensions change. `fitAddon.fit()` must run immediately or PTY cols/rows will be stale and terminal content will reflow incorrectly.

---

### MVP Definition (v1.1)

#### Must Ship

All items from PROJECT.md Active list:

- [ ] `build.sh` — per-platform and all-platform with macOS signing support (env-gated)
- [ ] Terminal full-fill fix — terminals fill all available vertical space
- [ ] Per-tab status bar — replaces `web-serving-bar` overlay, permanent single-line strip
- [ ] Tab renaming propagation to web dashboard session names
- [ ] Larger toolbar buttons — CSS fix, 36–44px click area
- [ ] New-session modal with agent picker + folder browser (remembers last folder)
- [ ] Per-tab SHIFT+/SHIFT- font size adjustment
- [ ] Settings modal declutter/restyle

#### Stretch Goal (v1.1 if time allows)

- [ ] Web dashboard visual refresh — card layout, status colors, CLI badge

#### Defer (v1.2+)

- Tab color coding per CLI type (PROJECT.md explicit deferral)
- Status heuristics for non-Claude CLIs (PROJECT.md explicit deferral)
- Font size persistence via backend

---

### Feature Prioritization Matrix (v1.1)

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Terminal full-fill fix | HIGH | LOW | P1 |
| Larger toolbar buttons | MEDIUM | LOW | P1 |
| Per-tab status bar | HIGH | MEDIUM | P1 |
| Settings modal declutter | MEDIUM | LOW | P1 |
| Tab renaming → dashboard | MEDIUM | LOW | P1 |
| Per-tab SHIFT+/SHIFT- font size | MEDIUM | LOW | P1 |
| New-session modal + folder browser | HIGH | MEDIUM | P1 |
| Build script (`build.sh`) | HIGH | MEDIUM | P1 |
| Web dashboard visual refresh | MEDIUM | MEDIUM | P2 |

---

### Implementation Notes (v1.1)

**build.sh**
- Auto-detect host platform via `uname -s`; default to host-platform build only.
- `--platform all` flag: build for all platforms sequentially (macOS must run on macOS host).
- macOS signing: `codesign --deep --force --options runtime --sign "$APPLE_DEVELOPER_ID" ./build/bin/agenthub.app`
- macOS notarization: `xcrun notarytool submit <zip> --apple-id "$NOTARIZATION_APPLE_ID" --password "$NOTARIZATION_PWD" --team-id "$NOTARIZATION_TEAM_ID" --wait`
- All signing steps env-gated so unsigned local builds work without certs.
- Linux: document required `apt-get` deps (build-essential, pkg-config, libgtk-3-dev, libwebkit2gtk-4.1-dev).
- Reference: existing `.github/workflows/build.yml` for exact flag values.

**Terminal full-fill fix**
- `.terminal-wrapper`: `display: flex; flex-direction: column; flex: 1; min-height: 0`
- Status bar: `flex-shrink: 0` (fixed height, does not compress)
- Terminal container div: `flex: 1; min-height: 0` (takes remaining space)
- `ResizeObserver` already fires `fitAddon.fit()` on dimension change — no additional fit logic needed.

**Per-tab status bar**
- Left: status dot + label (running/waiting/idle/errored)
- Center: session name (read-only in status bar; rename via tab double-click)
- Right: web toggle button | URL link (when enabled) | copy token | QR
- Always rendered. When web server not configured/running: "Web serving off" muted text + "Configure" button that opens settings.

**New-session modal + folder browser**
- Go: add `OpenDirectoryDialog() (string, error)` bound method using `runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{Title: "Select Working Directory"})`.
- Go: update `CreateSession(cliName, name, workDir string)` — pass `workDir` to PTY spawn `exec.Cmd.Dir`.
- React: modal with agent picker (button group), directory text input, "Browse..." button (calls Go binding), session name input (auto-fills from `path.basename(dir)`), Create button.
- `localStorage` key `agenthub.lastWorkDir` — read on modal open, write on Create.

**Per-tab SHIFT+/SHIFT- font size**
- In `TerminalPanel.tsx`, after creating `term`:
  ```ts
  term.attachCustomKeyEventHandler((e) => {
    if (e.type !== 'keydown') return true
    if (e.shiftKey && (e.key === '=' || e.key === '+')) {
      term.options.fontSize = Math.min(32, (term.options.fontSize ?? 14) + 1)
      fitAddon.fit()
      return false
    }
    if (e.shiftKey && e.key === '-') {
      term.options.fontSize = Math.max(8, (term.options.fontSize ?? 14) - 1)
      fitAddon.fit()
      return false
    }
    return true
  })
  ```
- On terminal init, read `localStorage.getItem('agenthub.fontSize.' + sessionId)` and apply if set.
- Store on each change: `localStorage.setItem('agenthub.fontSize.' + sessionId, String(term.options.fontSize))`.

**Settings modal declutter**
- Add tab strip at top: "CLI Paths" | "Web Serving". Only one section visible at a time.
- Remove Save/Cancel footer — save is per-field (existing Set Password / UpdateCLIPath patterns stay).
- Single "Close" button in footer.
- CA cert block already uses `<details>` — keep; improve summary text.

**Tab renaming → dashboard**
- Verify Go session struct has `json:"name"` tag on `Name` field.
- Dashboard `/api/sessions` already returns session objects — confirm `name` is in JSON.
- No WebSocket push needed — dashboard has Refresh button + optional poll.

---

## v1.0 Feature Landscape (Preserved)

The sections below document the v1.0 feature research. They remain valid as the foundation.

---

## Table Stakes (v1.0)

Features users expect from any terminal host for AI coding CLIs. Missing = product feels broken.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Tabbed terminal sessions | Every modern terminal uses tabs; power users run many agents in parallel | Low | Tab rename, reorder, color-coding are secondary |
| Per-tab session naming | Users run claude/codex/gemini simultaneously and need to tell them apart | Low | Defaults to CLI name + working dir; user can rename |
| Session persistence across app restart | tmux-style: sessions survive app crash or restart | Medium | Requires PTY backend to maintain process when frontend disconnects |
| Scrollback buffer | Reviewing agent output is as important as interacting; adequate buffer required | Low | xterm.js; configure ≥10K lines |
| Resize / reflow on window resize | SIGWINCH propagation to PTY; agents with TUI output break on stale COLS/ROWS | Low | Standard PTY behavior wired through WebSocket |
| Copy/paste from terminal | Essential for grabbing file paths, command output, error messages | Low | Browser clipboard API; xterm.js handles with configuration |
| ANSI/color rendering | AI coding CLIs output rich ANSI color | Low | xterm.js native |
| Unicode/emoji support | Claude Code and others output emoji and Unicode symbols | Low | xterm.js Unicode11 addon |
| Launch a new session with a chosen CLI | Core UX entry point | Low | Depends on CLI detection |
| Detection of installed CLIs | App surfaces only installed, launchable CLIs | Low | PATH lookup at startup |
| Kill / close a session | Terminate stuck agents cleanly | Low | SIGHUP to PTY child, close tab |

---

## Differentiators (v1.0)

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Web serving per session via hosted xterm.js | "Walk away" workflow — start an agent, check progress from phone | High | Session persistence, TLS, auth | Self-hosted equivalent to Claude Code Remote Control (Anthropic, Feb 2026) |
| Per-session web toggle (on/off) | Not all sessions should be web-accessible | Low | Web serving | Single UI control per tab |
| Self-signed TLS for all web connections | Transport encryption without domain name or CA dependency | Medium | Web serving, VPN binding | CA+leaf cert pattern; browser trust via user-installed CA |
| VPN interface binding (Tailscale-first) | Exposes sessions only on VPN interface, not public internet | Medium | Web serving, TLS | Tailscale 100.x.x.x is canonical use case |
| QR code generation for session URLs | Fastest path from desktop to phone | Low | Web serving | Encodes full session URL with auth token |
| Web dashboard with password auth | Browse all web-served sessions from a browser | Medium | Web serving, TLS, auth | Password for dashboard; tokens for shareable per-session links |
| Per-session shareable tokens | Give a teammate a link without sharing master password | Medium | Web serving, auth | Scoped token per session |
| Go-native PTY mode (no dependencies) | Works on any machine without tmux | Medium | None | Default mode; sessions are internal to app process |
| Multi-CLI session status indicators | Per-tab: running/waiting/idle/errored | High | Session monitoring | Heuristic output parsing; Claude Code most parseable |
| System tray / menubar presence | Keep sessions alive when main window is closed | Medium | App lifecycle | Critical for "walk away" workflow |

---

## Anti-Features (v1.0)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Mobile app | Out of scope; web UI served from desktop app is the remote access mechanism | Use web-served xterm.js in any mobile browser |
| CLI installation / management | Version management, permissions, install liability | Detect via PATH; surface clear "not installed" message |
| Tailscale / VPN installation | Significant privilege requirements, separate concern | Read available VPN interfaces at runtime; document user responsibility |
| Let's Encrypt / ACME cert management | Requires domain + internet-accessible challenge endpoint | Self-signed TLS; VPN provides network trust boundary |
| User account / registration system | Single-user per installation; registration implies multi-tenancy | Password + token model |
| Cloud hosting / SaaS deployment | Contradicts local-first design | Sessions stay on user's machine; web serving over VPN only |
| Plugin system for new CLIs | Premature extensibility before validating core workflows | Hardcode initial CLI set; add more via code contributions |
| Session output search / replay | High complexity; agent-sessions niche product exists | Adequate scrollback buffer; full replay is future scope |
| Split panes / tiling within a tab | Increases terminal rendering complexity | Multiple tabs cover the use case |
| Multi-user concurrent session editing | Auth complexity; denial-of-service via web | Per-session tokens are observe-or-interact, not multi-write |

---

## Feature Dependencies (v1.0)

```
CLI detection
  └─> Launch session (session creation)
        └─> PTY backend (Go-native)
              └─> Session persistence
              └─> xterm.js terminal rendering (tab UI)
                    └─> Scrollback, resize, copy/paste, ANSI, Unicode

Web serving per session
  └─> Session must exist
  └─> Self-signed TLS
  └─> Password auth (dashboard)
        └─> Per-session shareable tokens
        └─> QR code generation
  └─> VPN interface binding (optional; enhances web serving)

System tray presence
  └─> Session persistence (sessions survive window close)

Multi-CLI status indicators
  └─> Session running (PTY backend)
  └─> Output pattern detection per CLI
```

---

## AI Coding CLI-Specific Considerations

1. **Long-running autonomous sessions.** AI agents run for minutes to hours. Session persistence and remote access are primary, not secondary.
2. **Multiple parallel agents.** One agent per feature branch / git worktree. Tab management and session identity are load-bearing.
3. **Approvals and interaction.** Claude Code pauses for user approval. Web interface must support full interaction, not read-only.
4. **Output volume.** Large volumes of structured output. Scrollback ≥10K lines, correct ANSI, Unicode non-negotiable.
5. **Agent state visibility.** Users want: "is this agent done, waiting, or still thinking?" Even simple heuristics are valuable.
6. **CLI version resilience.** Lean on `claude`, `codex`, `gemini`, `opencode` as PATH commands with optional path overrides.

---

## Sources

**v1.2 research:**
- [Tailscale local client Go API — pkg.go.dev/tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) (HIGH — official package docs, stable API markers)
- [ipnstate.Status fields — pkg.go.dev/tailscale.com/ipn/ipnstate](https://pkg.go.dev/tailscale.com/ipn/ipnstate) (HIGH — official package docs)
- [Enabling HTTPS — tailscale.com/kb/1153/enabling-https](https://tailscale.com/kb/1153/enabling-https) (HIGH — official Tailscale docs)
- [Tailscale TLS certs blog — tailscale.com/blog/tls-certs](https://tailscale.com/blog/tls-certs) (HIGH — official Tailscale blog)
- Existing codebase read directly: `internal/webserver/server.go`, `auth.go`, `tokens.go`, `tls.go`, `network.go`

**v1.1 research:**
- Wails v2 cross-platform build guide: https://wails.io/docs/guides/crossplatform-build/
- Wails v2 code signing guide: https://wails.io/docs/guides/signing/
- Wails v2 dialog API (OpenDirectoryDialog): https://wails.io/docs/reference/runtime/dialog/
- xterm.js Terminal API (options.fontSize, attachCustomKeyEventHandler): https://xtermjs.org/docs/api/terminal/classes/terminal/
- xterm.js ITerminalOptions (fontSize): https://xtermjs.org/docs/api/terminal/interfaces/iterminaloptions/
- xterm-addon-fit FitAddon behavior: https://www.npmjs.com/package/xterm-addon-fit
- Existing codebase read directly: App.tsx, SettingsPanel.tsx, TabBar.tsx, TerminalPanel.tsx, web/dashboard.html
- Existing CI workflow: .github/workflows/build.yml

**v1.0 research:**
- [ttyd official site](https://tsl0922.github.io/ttyd/) — web terminal table stakes
- [agent-deck README](https://github.com/asheshgoplani/agent-deck) — AI coding agent feature set
- [Claude Code Remote Control docs](https://code.claude.com/docs/en/remote-control) — remote session QR code workflow, Feb 2026
- [GoTTY GitHub](https://github.com/yudai/gotty) — multi-session limitations
- [Wails framework](https://wailsapp.io/) — single-binary constraint, tray support
- [Julia Evans: Getting a modern terminal setup](https://jvns.ca/blog/2025/01/11/getting-a-modern-terminal-setup/) — tab management UX norms

---
*Feature research for: AgentHub — v1.0 (2026-03-17), v1.1 (2026-03-19), v1.2 (2026-03-20)*
