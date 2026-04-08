# Feature Research

**Domain:** Desktop app for AI coding CLI management — v1.11 new features
**Researched:** 2026-04-08
**Confidence:** HIGH (official docs + codebase verified)

---

## v1.11 Milestone: Local Network & UX Polish

### Context: What Already Exists

This is a SUBSEQUENT MILESTONE. The following are already built and are NOT scope for v1.11:

- Tailscale-only networking with Let's Encrypt TLS (server binds to Tailscale IP, FQDN-based URLs)
- Manual web server start in Settings modal (Settings > Web Server tab > Start Web Server button)
- Per-session web enable toggle (on/off per session in Daemon Manager panel)
- Settings as a modal dialog with two tabs: CLI Paths and Web Server
- Sidebar "New Tab" label with `PlusIcon`
- Claude Code detection via `exec.LookPath("claude")` with PATH augmentation
  for nvm/Volta/Homebrew (`/opt/homebrew/bin`, `~/.volta/bin`, `/usr/local/bin`)

---

## Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Local network fallback when Tailscale absent | Users without Tailscale still want web access from other devices on LAN; Tailscale is optional for many | MEDIUM | Re-introduces self-signed TLS (CA+leaf) scoped to local mode only; requires password since LAN is not zero-trust |
| Password protection in local mode | Self-signed cert cannot use network membership as access control; LAN is not Tailscale's zero-trust | LOW | Single random password per server start; shown in UI; HTTP Basic Auth middleware on web server |
| Nudge banner when running in local mode | User needs to know they are in a degraded security mode and how to upgrade | LOW | Persistent non-dismissable banner while local mode is active; links to Tailscale setup |
| Auto-start web server on daemon start | Having to manually start the web server each launch is friction for users who always want web access | LOW | Configurable boolean setting; if on, daemon calls StartWebServer during RunDaemon startup |
| Auto-enable web serving for new sessions | Per-session toggle defaults to off; users who always want sessions served must toggle each one manually | LOW | Configurable boolean setting; if on, CreateSession calls EnableSession immediately |
| Settings as sidebar tab | Settings modal interrupts workflow; other sidebar items (Home, Remote, Sessions) are persistent tabs — Settings should match | MEDIUM | Removes isOpen/onClose modal props; SettingsPanel becomes a full-height tab like DaemonManagerPanel |
| "New Session" label on sidebar | "New Tab" is ambiguous — the PlusIcon opens a new session modal, not a browser tab | LOW | Pure label and aria-label rename; no behavior change |
| Claude Code native install path detection | Native installer places binary at ~/.local/bin/claude which is not in PATH under launchd/Finder launches | LOW | Add ~/.local/bin to AugmentServicePath candidates in internal/daemon/path.go |

---

## Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Dual-mode web serving (Tailscale + local fallback) | AgentHub works everywhere — with or without Tailscale — unlike tools locked to a single networking model | MEDIUM | Mode auto-selected at server start based on Tailscale health; user sees which mode is active |
| Persistent non-blocking nudge for mode downgrade | Informs without blocking; user can still work while Tailscale setup happens in background | LOW | Top banner, not modal; stays visible but does not gate functionality |
| Zero-config auto-serve | Sessions immediately accessible over the web from the moment they are created, if server is running | LOW | Requires auto-start + auto-enable both enabled; when combined, the server starts itself and new sessions are served with no user action |

---

## Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Per-session password in local mode | Finer-grained access control | High complexity; per-session token system was removed in v1.2 for good reason | Single server-level password covers all sessions; per-session web-enable toggle is the per-session gate |
| User-configurable password in local mode | Power users want their own password | Credential management burden; users reuse weak passwords | Random generated password shown in UI; regenerates on each server start |
| Dismissable local-mode nudge banner | "Don't show again" reduces visual noise | Silently hides the security downgrade; users forget they are in unverified mode | Visually lightweight banner (not modal) so it is tolerable but remains informative |
| Auto-serve defaulting to on without explicit opt-in | Maximize convenience | Silently exposes all sessions over the network without user awareness | Present auto-serve as a setting the user enables; default off; show clear indicator when active |
| Settings as floating panel (not full tab) | Intermediate option between modal and tab | Creates a third navigation pattern inconsistent with everything else in the sidebar | Commit to sidebar tab pattern fully, consistent with Home, Remote, Sessions |
| Tailscale-to-local fallback at runtime | Graceful degradation if Tailscale disconnects mid-session | High complexity; race conditions with active browser clients; not a common scenario | Covered in future consideration; for v1.11, mode is selected at StartWebServer time only |

---

## Feature Dependencies

```
[Local network fallback]
    requires --> [Self-signed TLS (CA + leaf)]   (removed in v1.2; must be re-added, scoped to local mode)
    requires --> [Random password generation]     (in-memory; no persistence needed)
    requires --> [LAN IP binding]                 (net.InterfaceAddrs or 0.0.0.0)
    requires --> [HTTP Basic Auth middleware]      (on web server routes when in local mode)
    produces --> [Nudge banner]                   (displayed when local mode is active)

[Auto-start web server]
    requires --> [Mode selection logic]           (Tailscale health check at startup; picks Tailscale or local)
    enhances --> [Auto-enable sessions]           (natural pair; server must be running first)

[Auto-enable sessions]
    requires --> [Web server running]             (no-op if server not started)
    weak-dep --> [Auto-start web server]          (typically paired; either can be enabled independently)

[Settings as sidebar tab]
    replaces --> [Settings as modal dialog]       (modal pattern removed; tab pattern added)
    requires --> [Sidebar navigation wiring]      (onSettings callback behavior changes from modal open to tab switch)
    conflict --> [isOpen / onClose props]          (removed from SettingsPanel)

[Claude Code native path detection]
    enhances --> [AugmentServicePath]             (add ~/.local/bin to existing candidates list)
    standalone --> no dependency on other v1.11 features

["New Session" label rename]
    standalone --> pure string change in Sidebar.tsx
```

### Dependency Notes

- **Local network fallback requires self-signed TLS**: The CA+leaf infrastructure was deleted in v1.2 (Phases 15-17). It must be re-introduced using Go stdlib `crypto/x509` + `crypto/tls`. The existing `TLSConfig` override field in `webserver.Config` makes this a clean seam — pass a generated `*tls.Config` instead of Tailscale's `GetCertificate` hook.
- **Password ties to local mode only**: Tailscale mode retains zero-auth (network membership = access control). Password HTTP Basic Auth middleware only applies when the server binds to LAN IP with self-signed cert.
- **Auto-serve pair**: Auto-start and auto-enable are independent boolean settings but nearly always wanted together. Each can be enabled without the other; the combination produces the fully hands-off experience.
- **Settings-as-tab replaces modal entirely**: `SettingsPanel` currently returns null when `isOpen` is false. Converting to a tab removes the `isOpen`/`onClose` props entirely; the panel renders as a tab body unconditionally when the tab is active. The `onSettings` callback in `Sidebar` changes from calling `setSettingsOpen(true)` to switching the active panel state.

---

## MVP Definition for v1.11

### Launch With

- [ ] Local network fallback: self-signed TLS + single random password, binding to local network interface
- [ ] Nudge banner: persistent non-dismissable banner when in local mode, links to Tailscale setup
- [ ] Auto-start web server: boolean setting (default: off); if on, server starts at daemon boot
- [ ] Auto-enable sessions: boolean setting (default: off); if on, EnableSession called at CreateSession
- [ ] Settings as sidebar tab: modal removed, SettingsPanel rendered as persistent tab
- [ ] "New Session" label: rename sidebar label and aria-label from "New Tab" to "New Session"
- [ ] Claude Code native path: add ~/.local/bin (and Windows equivalent) to AugmentServicePath candidates

### Add After Validation (v1.11.x)

- [ ] Per-session auto-serve override in new-session modal (currently global setting; per-session override is a future refinement)
- [ ] Tailscale-to-local fallback at runtime (if Tailscale disconnects while server is running, graceful mode switch without dropping active browser sessions)

### Future Consideration (v2+)

- [ ] mDNS/Bonjour advertisement of local server for LAN device discovery without manual URL sharing
- [ ] Certificate pinning UI for browsers connecting to local-mode server (reduce cert warning friction)
- [ ] Persistent auto-serve preference saved across restarts (not just session lifetime)

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Local network fallback + password | HIGH | MEDIUM | P1 |
| Nudge banner (local mode) | MEDIUM | LOW | P1 |
| Auto-start web server | HIGH | LOW | P1 |
| Auto-enable sessions | HIGH | LOW | P1 |
| Settings as sidebar tab | MEDIUM | MEDIUM | P1 |
| "New Session" label rename | LOW | LOW | P1 (trivial; do with settings-as-tab phase) |
| Claude Code native path detection | MEDIUM | LOW | P1 |

All features are P1. This milestone is small and tightly scoped; no P2/P3 items exist.

**Priority key:**
- P1: Must have for v1.11 milestone to ship
- P2: Should have, add when possible
- P3: Nice to have, future consideration

---

## Implementation Notes by Feature

### Local Network Fallback + Password

**How other apps do it:**
- Bitwarden self-hosted: self-signed cert + user-configured password
- Caddy local HTTPS: auto-generates local CA, installs to trust store (invasive — not appropriate here)
- Typical LAN web apps: random generated token shown in UI, single challenge

**Recommended approach for AgentHub:**
1. On `StartWebServer`, check Tailscale health. If healthy: existing Tailscale+Let's Encrypt path unchanged. If unhealthy: generate self-signed CA+leaf (Go stdlib `crypto/x509` + `crypto/tls`), generate 16-char random alphanumeric password, bind to all interfaces or local LAN IP.
2. Password stored in-memory in `WebServer` struct; regenerated each time server starts.
3. Server adds Basic Auth middleware when in local mode. Challenge header: `WWW-Authenticate: Basic realm="AgentHub"`.
4. `BaseURL` in local mode uses machine's local IP (detected via `net.InterfaceAddrs`, preferring 192.168.x or 10.x range), not FQDN.
5. Web dashboard shows password prominently (copyable text field) when server is in local mode.
6. `WebServer` struct gains a `mode` field (`"tailscale"` or `"local"`) set at `Start()`.

**Self-signed cert generation:** Pure Go stdlib — no external deps. `crypto/rand` for key generation, `crypto/x509` for cert template, `crypto/tls` for in-memory `tls.Certificate`. Valid for 1 year.

**Confidence:** HIGH — all stdlib capabilities, well-documented pattern.

### Nudge Banner

**Expected behavior:**
- Displayed as a top banner in the desktop GUI when the web server is in local mode.
- Content: "Running in local network mode (Tailscale not found). Connection is not verified. Install Tailscale for trusted access." with a "Set up Tailscale" link.
- Non-dismissable while server is in local mode; disappears if server is stopped or restarted in Tailscale mode.
- Does NOT block any functionality; purely informational.

**Implementation:** Wails event emitted from daemon when local mode is active; frontend shows/hides banner based on event. Or: `GetWebServerStatus()` response includes a `mode` field; frontend polls status (already does this via 3s REST poll) and shows banner when mode is `"local"`.

**Confidence:** HIGH — existing REST polling pattern can carry the mode flag.

### Auto-Start Web Server

**Expected behavior:**
- Setting: `AutoStartWebServer bool` in daemon settings.
- When true: during `RunDaemon()` startup, after initial health check, daemon calls the same logic as `StartWebServer` IPC handler.
- Mode selection follows the same Tailscale health check path as manual start.
- If server fails to auto-start (e.g. port in use), daemon logs error but continues running; GUI shows no server running.

**Confidence:** HIGH — straightforward extension of existing RunDaemon startup sequence.

### Auto-Enable Sessions

**Expected behavior:**
- Setting: `AutoServeNewSessions bool` in daemon settings.
- When true: `CreateSession()` in daemon engine calls `ws.EnableSession(sessionID)` immediately after creating the session.
- No-op if web server is not running at the time of session creation.
- Sessions created before auto-serve is enabled are NOT retroactively enabled (avoids surprise exposure).

**Confidence:** HIGH — one additional call in CreateSession code path; already has the ws reference.

### Settings as Sidebar Tab

**Pattern in comparable desktop apps:**
- VS Code: Settings is a persistent tab, not a modal; survives navigation between other tabs.
- Warp terminal: Settings is a top-level navigable view accessed from the sidebar.
- Consensus: modals interrupt; tabs allow free navigation without losing context.

**Recommended approach for AgentHub:**
1. Add `'settings'` to the active panel union type in App.tsx (currently `'home' | 'remote' | 'sessions' | null`).
2. Change `onSettings` in Sidebar to emit a panel switch event rather than opening a modal.
3. Remove `isOpen`, `onClose` props from `SettingsPanel`; it renders unconditionally when active.
4. The Settings tab is closed by navigating to any other sidebar item (same pattern as Sessions, Remote).
5. Remove the footer `Close` button from SettingsPanel (navigation replaces it), or convert it to "Back" that switches to the previous panel.
6. `settings-overlay` and `settings-panel` modal CSS classes become tab-body CSS (no overlay, no fixed position).

**Confidence:** HIGH — established Wails/React tab pattern, mirrors existing DaemonManagerPanel behavior.

### Claude Code Native Path Detection

**Official install paths (from code.claude.com/docs/en/setup — HIGH confidence):**
- macOS/Linux native installer: `~/.local/bin/claude` (uninstall removes `~/.local/bin/claude` and `~/.local/share/claude`)
- Windows native installer: `%USERPROFILE%\.local\bin\claude.exe`
- macOS Homebrew cask: `/opt/homebrew/bin/claude` (already in AugmentServicePath candidates)
- npm (deprecated): wherever npm -g installs; typically `~/.nvm/.../bin` (already handled by nvmActiveBin)

**Current gap:** `~/.local/bin` is NOT in the current `AugmentServicePath` candidates list in `internal/daemon/path.go`. It is commonly absent from PATH when the daemon runs as a launchd service or when the app is launched from Finder/Dock, because shell init files (which add `~/.local/bin` to PATH via the install script) are not sourced.

**Fix:** Add `filepath.Join(home, ".local", "bin")` to the candidates slice. On all platforms (macOS, Linux, Windows), `os.UserHomeDir()` returns the correct home directory and `~/.local/bin` is the correct subdirectory for native Claude Code installs.

**Confidence:** HIGH — confirmed by official Claude Code uninstall documentation listing exact paths.

---

## Competitor Feature Analysis

| Feature | OpenCode | AgentHub v1.10 | AgentHub v1.11 Plan |
|---------|----------|----------------|---------------------|
| Local network access | Requires manual `opencode web` separate server; no GUI toggle | Tailscale-only | Automatic fallback to self-signed+password LAN mode |
| Auto-serve on session create | Proposed in GitHub issue #11997 (not shipped) | Manual toggle per session | Optional auto-enable setting |
| Settings navigation | Modal | Modal | Sidebar tab (persistent) |
| Claude Code detection | PATH only | PATH + common npm managers | PATH + npm managers + ~/.local/bin native path |

---

## Sources

- [Claude Code Advanced Setup](https://code.claude.com/docs/en/setup) — native install path `~/.local/bin/claude` confirmed by uninstall instructions (HIGH confidence)
- [Claude Code GitHub issue #10970](https://github.com/anthropics/claude-code/issues/10970) — `~/.local/bin` PATH issue for Homebrew users; confirms native vs Homebrew path distinction (MEDIUM confidence)
- [OpenCode GitHub issue #11997](https://github.com/anomalyco/opencode/issues/11997) — auto-start web server proposal with implementation notes (MEDIUM confidence; different product)
- AgentHub codebase read directly: `internal/pty/detect.go`, `internal/daemon/path.go`, `internal/webserver/server.go`, `frontend/src/components/SettingsPanel.tsx`, `frontend/src/components/Sidebar.tsx` (HIGH confidence)
- UX research: settings-as-tab pattern validated by VS Code, Warp terminal, and modal-vs-tab UX analysis (MEDIUM confidence)

---

*Feature research for: AgentHub v1.11 — Local Network Fallback, Auto-Serve, Settings-as-Tab, Label Rename, Native Path Detection*
*Researched: 2026-04-08*
