# Pitfalls Research

**Domain:** Go/Wails desktop app — remote tailnet session discovery, auto-update, Tailscale install assistance, app menus
**Researched:** 2026-04-06
**Confidence:** HIGH (architecture is fully readable; pitfalls verified against live code and official docs)

---

## Critical Pitfalls

### Pitfall 1: Wails App Menu Conflicts with Existing Native CGO Tray

**What goes wrong:**
Adding a Wails `options.App.Menu` while the existing native CGO `NSStatusBar` tray is already running causes duplicate ObjC class registration or silent menu handler failures. The app already uses a `.m` file (`tray_objc.m`) with an `NSMenuDelegate` for the tray. Wails also installs its own `NSMenu` for the app menu bar. If `wails.Run()` is given a `Menu:` option, the two ObjC menu systems may conflict — particularly if both try to set the `NSApplication` delegate or respond to the same `menuWillOpen:` selector.

The Wails docs explicitly warn that on macOS, `AppMenu.Append(menu.AppMenu())` must be the first call, followed immediately by `menu.EditMenu()`, and that order matters for correct placement. Missing `EditMenu` silently disables all standard text shortcuts (Cmd+C, Cmd+V, Cmd+Z) in the xterm.js terminal and any other text areas.

**Why it happens:**
Developers add a `Menu:` field to `options.App` and verify it visually, but don't test keyboard shortcuts in the terminal. The existing tray code's `NSMenuDelegate` is on a different `NSMenu` instance, but shared `NSApplication` delegate state can interfere.

**How to avoid:**
- Add the Wails menu in `main.go` inside `runGUI()` as a `Menu:` option field — not via cgo.
- Use the exact macOS ordering: `NewMenu()` → `AppMenu.Append(menu.AppMenu())` → `AppMenu.Append(menu.EditMenu())` before any custom submenus.
- Test Cmd+C/Cmd+V inside an xterm.js terminal after adding the app menu.
- Do not attempt to register a second `NSMenuDelegate` in ObjC for the app menu bar; let Wails manage the menu bar and keep the tray's delegate isolated to its `NSStatusItem` menu.

**Warning signs:**
- Standard keyboard shortcuts (Cmd+C, Cmd+V, Cmd+Z) stop working in terminals after adding the menu.
- `wails build` completes but the menu items fire no callbacks.
- Linker errors mentioning duplicate symbol `@implementation` for menu-related ObjC classes.

**Phase to address:** Whichever phase implements standard app menus.

---

### Pitfall 2: Tailnet Peer Port Scanning — Blocking the Main Thread and False Positives

**What goes wrong:**
Discovering which tailnet peers are running an AgentHub daemon requires probing each peer's HTTP API port. If this probe runs synchronously on the Wails startup goroutine or blocks the daemon API handler, the app freezes. Worse, if the probe has no timeout, a single offline peer that Tailscale still lists as a known node causes the entire discovery to hang indefinitely.

A secondary failure: Tailscale's `local.Client.Status()` returns all peers that have ever been in the tailnet, including offline ones. The `PeerStatus.Online` field indicates whether the node is currently connected to the Tailscale control plane — but a node can be "online" in Tailscale's view yet have its AgentHub daemon stopped. Both checks are necessary.

**Why it happens:**
Developers call `local.Client{}.Status()`, iterate all peers, and issue HTTP probes assuming that `Online: true` means the daemon is reachable. They forget timeouts, and they issue probes from a goroutine that blocks the UI response path.

**How to avoid:**
- Run peer discovery in a background goroutine, never on the Wails event loop or daemon API goroutine.
- Use `http.Client{Timeout: 2 * time.Second}` for each probe — never the default zero-timeout client.
- Filter peers with `peer.Online` before probing, but treat `Online` as a hint, not a guarantee.
- Issue probes concurrently with `sync.WaitGroup` and a worker pool capped at ~5 goroutines.
- Emit results back to the frontend via `runtime.EventsEmit` as they arrive, not as a single blocking list.
- Cache results with a TTL (e.g., 30s) so re-opening the remote panel does not re-probe on every render.

**Warning signs:**
- Remote session panel takes >5s to appear.
- App freezes on startup when one tailnet peer is offline.
- Probe goroutine leak visible in `runtime.NumGoroutine()` growing over time.

**Phase to address:** Tailnet peer discovery phase.

---

### Pitfall 3: Remote Daemon HTTP API Is Unix-Socket-Only — No TCP Endpoint Exists for Remote Peers

**What goes wrong:**
The existing daemon serves its control API exclusively over a Unix socket (`internal/daemon/api.go`). The web server binds to the Tailscale IP with HTTPS (Let's Encrypt via Tailscale daemon). For remote session access, the app needs to call the remote daemon's HTTP API over the tailnet — but the daemon API is a Unix socket, not a TCP port. There is no remote-accessible HTTP endpoint for the daemon API on peer machines.

Developers assume they can open `http://<tailscale-ip>:<daemon-port>/sessions` and get the remote session list. This endpoint does not exist.

**Why it happens:**
The daemon API and the web server are separate. The web server (Tailscale HTTPS, port 443) serves the xterm.js terminal UI — not a JSON session list API. The JSON API is Unix socket only. It is easy to conflate the two.

**How to avoid:**
Add a new lightweight HTTP/JSON endpoint exposed on the web server (not the Unix socket) specifically for remote peer discovery. For example, `GET /api/sessions` on the existing Tailscale HTTPS web server. This endpoint is already behind Tailscale network-only binding, so it is safe to expose without additional auth. The remote GUI calls this via the peer's Tailscale FQDN URL (e.g., `https://peer.tail.ts.net/api/sessions`).

Do not try to expose or tunnel the Unix socket remotely.

**Warning signs:**
- Remote discovery code references `daemon.NewDaemonClient` with a remote IP — this will never work.
- Connection refused errors when probing remote peers on the daemon's port.
- Confusion between "relay port" (TCP, localhost only) and "web server port" (Tailscale HTTPS, accessible on tailnet).

**Phase to address:** Tailnet peer discovery and remote session API phase. This is a design decision that must be made before writing any discovery code.

---

### Pitfall 4: Auto-Update Binary Replacement Fails on macOS App Bundle and Windows

**What goes wrong:**
The distributed binary on macOS is a `.app` bundle, not a bare executable. Self-update libraries like `go-selfupdate` target the running executable path (`os.Executable()`), which inside a `.app` bundle resolves to `AgentHub.app/Contents/MacOS/agenthub`. Replacing only that file leaves the bundle in an inconsistent state — the bundle's `Info.plist` version string, the icon resources, and any framework links are unchanged. The user's macOS sees a version mismatch. Gatekeeper may also block the replaced inner binary on macOS Sequoia, which tightened override controls.

On Windows, the running `.exe` is locked by the OS. The standard rename trick (rename old binary, write new binary to original path) fails with "Access is denied" when using `os.Rename()` on the running executable. The release pipeline ships an NSIS installer; self-updating by patching a standalone exe bypasses the installer and leaves registry entries stale.

**Why it happens:**
Self-update libraries document the Unix rename pattern and "target the running executable" as the default. macOS bundle structure is not accounted for. Windows file locking is platform-specific and only surfaces at runtime.

**How to avoid:**
- For macOS: implement "notify and open in browser" rather than in-place bundle patching. Point the user to the GitHub release page or the Homebrew cask update command (`brew upgrade --cask agenthub`). The Homebrew cask tap already auto-updates on release (v1.8 milestone).
- For Windows: point to the GitHub release download page or trigger the NSIS installer download. Do not attempt to replace a running `.exe` on Windows.
- For Linux: bare binary or `.deb` — `go-selfupdate` rename pattern works here. Direct binary replacement is safe on Linux.
- Auto-update checker should only check and notify; the actual update mechanism differs per platform.
- Use `filepath.EvalSymlinks(os.Executable())` before determining the update target path.

**Warning signs:**
- `os.Executable()` returns a path inside `/var/folders/` (temp sandbox) on macOS rather than the `.app` bundle path.
- Tests of the update path pass on Linux CI but fail silently on macOS test runs.
- Windows CI builds succeed but `os.Rename()` returns "Access is denied" in a smoke test.

**Phase to address:** Auto-update checker/installer phase.

---

### Pitfall 5: GitHub Releases API Rate Limiting Breaks Unauthenticated Update Checks

**What goes wrong:**
The GitHub Releases REST API (`https://api.github.com/repos/scottkw/agenthub/releases/latest`) has a rate limit of 60 requests/hour for unauthenticated requests, keyed by the originating IP. If multiple users share a NAT (corporate network, university), they share the same quota. Multiple AgentHub instances checking at startup exhaust the limit quickly, and all instances receive HTTP 429 — which, if not handled, surfaces as a misleading error dialog.

**Why it happens:**
Update checks are easy to write against the GitHub API without authentication. The rate limit feels fine in development (single IP, rare checks). The failure only appears in multi-user scenarios or when the check interval is too aggressive (e.g., on every app launch).

**How to avoid:**
- Check at most once per session startup and no more than once per hour — persist the last-check timestamp to disk.
- Handle HTTP 429 and all non-200 responses gracefully: log and silently skip, never show an error to the user.
- Send `If-None-Match` with the cached ETag value; a 304 Not Modified response does not count against rate limits.
- No authentication token is needed or appropriate here — just backoff and caching.

**Warning signs:**
- HTTP 403 or 429 responses from `api.github.com` in logs.
- Users on shared networks see "update check failed" errors at startup.
- Update checker fires on every window focus event rather than with a time gate.

**Phase to address:** Auto-update checker phase.

---

### Pitfall 6: Tailscale Install Assistance — `brew install` Subprocess Requires a Terminal

**What goes wrong:**
Attempting to run `brew install tailscale` as a subprocess from inside the GUI and stream its output to the app fails because:
1. Homebrew detects that it is not running in an interactive terminal (no TTY) and suppresses progress bars; the `NONINTERACTIVE=1` flag helps with prompts but does not solve the sudo requirement.
2. Tailscale service setup requires sudo — prompting for a password from inside a Wails WebView context has no safe mechanism.
3. On Apple Silicon, Homebrew lives at `/opt/homebrew/bin/brew`; on Intel it is `/usr/local/bin/brew`. If the daemon's PATH augmentation does not include both, `exec.LookPath("brew")` fails silently.

The correct pattern for Tailscale install assistance is: detect whether Tailscale is installed and connected (already done in the health check system), then display the appropriate platform-specific installation command for the user to copy and run — not attempt to run it silently in the background.

**Why it happens:**
It is tempting to give users a one-click install. The Tailscale install assistance feature sounds like "run brew install on their behalf." The sudo and TTY constraints make this unsafe and unreliable.

**How to avoid:**
- Show a copyable shell command, not a "click to install" button that runs a subprocess.
- For macOS: offer `brew install tailscale` or the Tailscale download URL.
- For Linux: show `curl -fsSL https://tailscale.com/install.sh | sh` or the appropriate package manager command.
- For Windows: show the Tailscale MSI download URL.
- Use `os/exec` at most to run `brew --version` to detect if Homebrew is installed, not to install packages.
- The existing health check modal (v1.2) already handles platform-specific guidance; v1.9 enhances that, not replaces it with automation.

**Warning signs:**
- Subprocess running brew hangs waiting for TTY input.
- `exec.Command("brew", "install", "tailscale").Run()` appears to succeed but Tailscale service is not started.
- `exec.LookPath("brew")` returns `""` when GUI is launched from Finder even on a machine with Homebrew installed.

**Phase to address:** Tailscale install assistance phase.

---

### Pitfall 7: Remote Session Attach Requires Web Server WebSocket — Not the Relay Port

**What goes wrong:**
The existing attach mechanism (`cmd_attach.go`) connects to the local daemon's relay server over `127.0.0.1:<relay-port>` via WebSocket. The relay port is ephemeral and localhost-only. Remote attach must go through the Tailscale HTTPS web server's WebSocket endpoint — which is already serving terminals over Tailscale HTTPS for browser access. Developers who naively copy the local relay client and point it at the remote IP and relay port get a connection refused, because the relay server binds to `127.0.0.1` by design (`api.go` line 66).

**Why it happens:**
The relay port and the web server port are both in the daemon, both do WebSocket. Developers conflate "attach uses WebSocket" with "just point the WebSocket client at the remote web server." But the relay is an internal loopback-only server; the web server is the externally accessible endpoint.

**How to avoid:**
- The remote GUI panel that shows peer sessions should link users to the existing web terminal URL — the Tailscale HTTPS URL already works in a browser for remote access. This is the zero-new-code path for remote viewing.
- For CLI remote attach, proxy through the web server's WebSocket endpoint, which already speaks the binary relay protocol — rather than inventing a new remote relay protocol.
- Map out the data path before writing code: local relay stays unchanged; remote path reuses the existing web WebSocket endpoint.
- Expose a session discovery API endpoint on the web server; use the existing WebSocket for remote attach.

**Warning signs:**
- Remote attach code references `relay-port` endpoint or `relay.NewServer` for cross-machine connections.
- Tests pass locally (because relay is on loopback) but time out against a real remote peer.
- Connection refused on the remote machine's relay port.

**Phase to address:** Remote session attach phase; must be designed before implementation begins.

---

### Pitfall 8: VERSION Embedding — Wails Dev vs. Production Build Mismatch

**What goes wrong:**
The WelcomeTab currently hard-codes the version string (noted as tech debt in PROJECT.md). The correct approach is to inject the version at build time via `-ldflags "-X main.Version=..."`. In dev mode (`wails dev`), these ldflags are not applied, so the version shows as empty string or a placeholder. If the frontend reads the version via a Wails binding (`GetVersion()` → `app.Version`), the dev vs. production split causes confusion: tests see `""`, production sees `"v1.9.0"`.

**Why it happens:**
Developers fix the hard-coded string with a Go variable but forget to set it for `wails dev`. CI builds use `build.sh` which passes ldflags; local dev uses `wails dev` which does not.

**How to avoid:**
- Expose `GetVersion()` as a Wails binding that returns the ldflags-injected variable, falling back to `"dev"` when empty.
- Document the fallback explicitly so the frontend renders `"dev"` in development rather than blank.
- Update `build.sh` and the CI release workflow to pass `-X main.Version=$(git describe --tags --abbrev=0)` consistently.
- Add a frontend test that verifies the `GetVersion()` binding exists and returns a non-empty string.

**Warning signs:**
- Welcome screen shows blank version in production build.
- `wails dev` and `wails build` produce different version display behavior with no documented fallback.
- `go test ./...` passes but version binding is never exercised.

**Phase to address:** Welcome screen polish phase.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Show all peers regardless of `Online` field | Simpler discovery code | Users see stale peers that fail to connect | Never — always filter on `PeerStatus.Online` |
| Sequential peer probe with no timeout | Simple sequential code | Discovery hangs N×2s if N peers are offline | Never — always concurrent with timeout |
| Hard-code update channel to unauthenticated GitHub API | Zero config | Rate limit failures on shared networks | Acceptable only if check is TTL-gated client-side (1/hr + ETag caching) |
| Show "checking for updates..." on every launch | Reassures user | Triggers 429 at scale; annoying if not dismissed quickly | Never — gate behind 1-hour TTL |
| Copy local attach code for remote attach | Fast to ship | Wrong protocol target; fails at first real remote test | Never — design the remote data path before coding |
| Bundle Tailscale install as subprocess | "One click" experience | Fails on TTY check, requires sudo, breaks on non-Homebrew paths | Never — copyable commands are safer and more reliable |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Tailscale `local.Client.Status()` | Trust `Online: true` means AgentHub daemon is reachable | `Online` = connected to control plane only; still HTTP-probe before treating as available |
| Tailscale `local.Client.Status()` | Include exit nodes and relay nodes in peer list | Filter to peers with `TailscaleIPs` set and `Online: true`; skip subnet routers |
| GitHub Releases API | Call on every startup event | Cache last-check timestamp to disk; skip if checked within 1 hour |
| GitHub Releases API | Show error dialog on API failure | Silently log; update check is best-effort |
| Wails app menu + existing cgo tray | Add `Menu:` to Wails options and assume no conflict | Verify that `NSMenuDelegate` methods on the tray's `NSStatusItem` menu do not interfere with the app menu bar delegate; test keyboard shortcuts |
| Wails app menu on macOS | Append custom items before `AppMenu()` and `EditMenu()` | Always: `NewMenu()` → `AppMenu.Append(menu.AppMenu())` → `AppMenu.Append(menu.EditMenu())` → then custom items |
| Remote daemon API | Call Unix socket endpoint from remote machine | Expose a read-only JSON sessions endpoint on the Tailscale HTTPS web server instead |
| `os.Executable()` for update target | Use the raw path for binary replacement | Run `filepath.EvalSymlinks(os.Executable())` and decide per-platform strategy before writing any replacement code |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Sequential peer probe with no timeout | Discovery panel takes N×2s where N = offline peers | Concurrent probes, `http.Client{Timeout: 2s}`, worker pool | Immediately if any peer is offline |
| Uncached peer probe on every panel open | Repeated 2s probes each time user opens remote tab | Cache result with 30s TTL | Immediately if user opens panel rapidly |
| GitHub API check on every app focus | Multiple 429 responses per day | 1-hour TTL + ETag conditional requests | 2+ users on same NAT, or aggressive focus cycling |
| Streaming brew output with no timeout | GUI freezes if brew hangs waiting for TTY | Do not stream brew output; copy-to-clipboard pattern only | Immediately on any Homebrew install attempt from GUI |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Exposing daemon Unix socket API over TCP for remote access | Any tailnet peer gains full session control (create/kill/rename) | Never expose daemon Unix socket API over TCP; expose only read-only session list via Tailscale HTTPS web server |
| Downloading auto-update binary over plain HTTP | MITM can deliver malicious binary | Always download from `https://github.com/scottkw/agenthub/releases/`; verify SHA256 from `checksums.txt` |
| Trusting release asset filename for platform detection | Wrong platform binary if redirect is malicious | Parse OS/arch from `os.GOOS`/`os.GOARCH` in Go; verify against checksums.txt before any execution |
| Silently running downloaded binary before checksum check | Arbitrary code execution if download is corrupted | Verify SHA256 from `checksums.txt` before exec |
| TLS skip-verify for tailnet peer connections | Connection to impersonated peer | Use standard `http.Client` which validates TLS; never use `InsecureSkipVerify` |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Show all tailnet peers including offline ones | User clicks a peer, gets connection error with no explanation | Only show peers where `Online: true` AND HTTP probe succeeded; show last-seen for recently offline peers |
| "Update available" modal that blocks workflow | Interrupts coding session | Non-blocking notification in toolbar/tray; user dismisses and updates later |
| "Checking for update..." spinner that never resolves | User thinks app is broken | 5s timeout on update check; show nothing if check fails |
| Tailscale install instructions as a plain text block | User must manually transcribe the brew command | Each command in a code block with a copy-to-clipboard button |
| Remote session panel mixes local and remote sessions | Confusing which sessions are local vs. remote | Group by machine hostname with clear visual separation |
| Version "v1.9.0" in Welcome tab that does not match the running binary | Erodes trust; looks sloppy | Use `GetVersion()` binding so displayed version is always the actual build version |

---

## "Looks Done But Isn't" Checklist

- [ ] **App menus:** Menus appear visually but Cmd+C/Cmd+V broken in terminal — verify `menu.EditMenu()` appended and ordering is correct
- [ ] **App menus:** Menu items have click handlers registered but callbacks fire on wrong goroutine — Wails menu callbacks run on AppKit thread; use `runtime.*` functions only
- [ ] **Peer discovery:** Shows peer list but includes offline machines — verify `PeerStatus.Online` filter and HTTP probe before display
- [ ] **Peer discovery:** Works on dev machine but hangs in a real tailnet — verify concurrent probes with timeouts, not sequential
- [ ] **Remote session list:** Lists sessions but clicking "attach" does nothing — verify remote attach target is web server WebSocket, not localhost relay
- [ ] **Auto-update checker:** Shows "update available" on every launch — verify TTL cache is persisted to disk, not just in-memory
- [ ] **Auto-update download:** Download succeeds but binary not executable after write — verify `os.Chmod(path, 0755)` after writing new binary on Unix
- [ ] **Auto-update macOS:** Binary placed at `os.Executable()` path but version not reflected in Finder — full `.app` bundle must be replaced, not just the inner binary; prefer "open download page" approach
- [ ] **Tailscale install guidance:** Shows instructions but no copy button — ensure copyable code blocks in UI
- [ ] **VERSION in Welcome tab:** Shows correct version in `wails build` but blank in `wails dev` — verify fallback to `"dev"` when ldflags are not set
- [ ] **Windows build:** Self-update path compiled but untested — verify Windows-specific `os.Rename()` failure is caught and falls back to "open download page" notification
- [ ] **Linux/Windows tray stubs:** Adding app menu does not crash Linux/Windows builds — verify `tray_linux.go` and `tray_windows.go` stubs compile after any new tray-related changes

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| App menu breaks keyboard shortcuts | LOW | Reorder `AppMenu()` and `EditMenu()` appends; rebuild |
| App menu + cgo tray conflict causes crash | HIGH | Remove Wails Menu from options; re-implement as custom ObjC menu or scope to only the Wails menu bar |
| Remote API design wrong (Unix socket exposed over TCP) | HIGH | Add new endpoint to web server; update all callers; existing daemon API unchanged |
| Sequential peer probes freeze UI | MEDIUM | Add goroutine pool and timeout; no protocol changes needed |
| Auto-update corrupts binary on rollback failure | HIGH | Provide user with manual download URL; add rollback verification step |
| VERSION blank in production | LOW | Fix ldflags in `build.sh`; add fallback to `"dev"` in Go code |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Wails menu + cgo tray conflict | App menus phase | Cmd+C/Cmd+V works in terminal after menu added; no crash on startup |
| Tailnet peer probe blocking / no timeout | Peer discovery phase | Discovery completes in <3s even with 2 offline peers; no goroutine leak in tests |
| Remote daemon API misuse | Peer discovery design step | Remote peer endpoint uses web server HTTPS URL, not a TCP daemon port |
| Auto-update binary replacement failure | Auto-update phase | macOS shows download link; Linux actually replaces binary; Windows shows download link |
| GitHub API rate limiting | Auto-update phase | Check interval TTL-gated; 429 response produces silent log, not error dialog |
| Tailscale install subprocess failure | Tailscale install guidance phase | No `exec.Command("brew install")` in codebase; only copyable text shown |
| Remote attach wrong protocol target | Remote attach phase | Attach uses web server WebSocket URL, not relay port |
| VERSION blank in Welcome tab | Welcome screen polish phase | `wails dev` shows "dev"; `wails build` shows tag from ldflags |

---

## Sources

- Tailscale Go SDK `tailscale.com/client/tailscale` — `LocalClient.Status()`, `PeerStatus.Online` — [pkg.go.dev](https://pkg.go.dev/tailscale.com/client/tailscale)
- Tailscale ipnstate package — `PeerStatus` structure and `Online` field — [pkg.go.dev](https://pkg.go.dev/tailscale.com/ipn/ipnstate)
- Wails v2 Menus documentation — macOS EditMenu ordering requirement — [wails.io](https://wails.io/docs/reference/menus/)
- Wails v2 GitHub Issue #3865 — menu and systray conflict in Wails v3 alpha — [github.com/wailsapp/wails](https://github.com/wailsapp/wails/issues/3865)
- Wails v2 PR #3847 — fixed macOS menu example with correct AppMenu/EditMenu ordering — [github.com/wailsapp/wails](https://github.com/wailsapp/wails/pull/3847)
- GitHub API rate limits documentation — 60 req/hr unauthenticated — [docs.github.com](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
- go-selfupdate (creativeprojects) — rollback failure behavior, Windows exe notes — [github.com/creativeprojects/go-selfupdate](https://github.com/creativeprojects/go-selfupdate)
- golang/go Issue #21997 — Windows rename vs. running binary behavior — [github.com/golang/go](https://github.com/golang/go/issues/21997)
- Homebrew non-interactive discussion — `NONINTERACTIVE=1` and TTY constraints — [github.com/orgs/Homebrew/discussions/3199](https://github.com/orgs/Homebrew/discussions/3199)
- Project codebase: `internal/daemon/api.go` (Unix socket only, relay binds to `127.0.0.1`), `tray.go` (native cgo NSStatusBar), `main.go` (wails.Run options, no Menu field currently), `internal/daemon/types.go` (SessionInfo struct)

---
*Pitfalls research for: Go/Wails desktop app v1.9 — remote tailnet sessions, auto-update, Tailscale install, app menus*
*Researched: 2026-04-06*
