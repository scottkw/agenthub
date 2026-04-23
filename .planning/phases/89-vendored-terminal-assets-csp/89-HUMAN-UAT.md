# Phase 89 HUMAN-UAT — Manual Verification

**Phase:** 89-vendored-terminal-assets-csp
**Created:** 2026-04-22
**Status:** pending
**Covers:** SC-3 (zero third-party requests in live session) and SC-4 (Chromium + Safari in Tailscale + local-network-fallback modes)

———

## Why Manual?

Three verifications cannot be automated in CI:

1. **Safari compatibility.** chromedp is Chromium-only. Safari's CSP engine has historically differed on `connect-src 'self'` for wss:// (Research Q2 / MDN warning). D-09's explicit `wss://<host>` clause is a defense for this, but only a real Safari render proves it.
2. **Local-network-fallback mode.** The fallback path uses a self-signed HTTPS cert bound to a LAN IP, reachable only from a second device on the same network. CI cannot easily simulate this topology.
3. **Live-session network audit.** Proving zero requests to `cdn.jsdelivr.net` during a real attach/resize/scroll/detach session requires DevTools Network tab observation across a multi-second user journey — not something a headless test runs end-to-end.

———

## KNOWN FINDING — read before starting UAT

The automated `go test -tags=e2e` run on the dev Mac produced:

- `TestBrowserCSP_DashboardNoViolations` — **PASS**
- `TestBrowserCSP_JoinNoViolations` — **PASS**
- `TestBrowserCSP_TerminalNoViolations` — **FAIL** (12 × `style-src-elem 'inline'` violations, lineNumber:1)

The terminal page's violations originate from xterm.js injecting `<style>` elements at runtime (cursor/selection rendering), which the D-09 policy `style-src 'self'` blocks. This is a genuine conflict between the locked D-09/D-06 decisions (no `'unsafe-inline'`) and xterm.js's runtime behavior.

**UAT-1 Safari and UAT-3 live-session network audit are still meaningful** — Safari uses a different CSP engine (WebKit) and may or may not surface the same violations; UAT-3 audits network requests, not style-src violations. UAT-2 is likewise still meaningful.

The operator should proceed with all three UAT items, then report results. A gap-closure decision (add `'unsafe-inline'` to style-src, switch to a style-nonce/hash approach, or patch xterm) will follow based on UAT observations.

———

## UAT-1: Safari renders all three pages without CSP violations (Tailscale mode)

**Requirement:** SEC-08 SC-4 (Safari half)

**Steps:**

1. Start the agenthub daemon in Tailscale mode (default):
   ```
   agenthub start
   ```
2. Create or select a session, then click "Share" in the GUI → copy the capability-bearing URL.
3. Open Safari on this Mac (desktop Safari; iOS Safari is a nice-to-have, not required for v3.1).
4. Open Web Inspector → Console (Develop menu → Show Web Inspector → Console tab). If the Develop menu is hidden, enable it via Safari → Preferences → Advanced → Show Develop menu.
5. Paste the copied URL into the address bar and press Return.
6. Wait for the terminal to render. Observe the Console tab.
7. Interact with the terminal: type a command, resize the window, scroll back through output.
8. Navigate to `https://<tailscale-hostname>:<port>/dashboard`.
9. Navigate to `https://<tailscale-hostname>:<port>/join`.

**Expected:**

- Zero Console messages containing the literal text `Refused to load`, `Refused to execute`, `Refused to connect`, or `Content Security Policy`.
- Terminal renders the xterm UI and echoes keystrokes.
- Dashboard and join pages render without any console warnings beyond those present before Phase 89 (pre-existing favicon 404 etc. are not blockers).

**Sign-off:**
- [ ] Verified on Safari version: ___________ (e.g., 17.4)
- [ ] Terminal CSP-clean: yes / no
- [ ] Dashboard CSP-clean: yes / no
- [ ] Join CSP-clean: yes / no
- [ ] Date/operator: ___________

———

## UAT-2: Local-network-fallback HTTPS mode renders all three pages clean

**Requirement:** SEC-08 SC-4 (local-network-fallback half)

**Context:** Local-network-fallback is the path triggered when Tailscale is unavailable or disabled; the daemon binds to a LAN IP with a self-signed cert and prompts for Basic Auth.

**Steps:**

1. Disable Tailscale on this Mac (Tailscale menu → "Log out" or System Settings → Network → disable the Tailscale interface).
2. Restart the agenthub daemon — it should fall back to local-network HTTPS mode and display a LAN URL + randomly-generated Basic Auth password in the GUI.
3. From a second device on the same Wi-Fi network (phone, tablet, another laptop), open the displayed LAN URL (something like `https://192.168.1.42:12345/dashboard`) in Chrome or Safari on the second device.
4. Accept the self-signed certificate warning (click "Advanced → proceed anyway" in Chrome, or "Show Details → visit this website" in Safari).
5. Enter the Basic Auth credentials when prompted.
6. Open the device's developer tools Console (Chrome: DevTools → Console; Safari: the technique varies by platform — may require "Settings → Safari → Advanced → Web Inspector" on iOS and pairing to the Mac).
7. Open `/dashboard`, `/join`, and (after issuing a capability URL from the Mac GUI) `/sessions/{id}?cap=...`.

**Expected:**

- All three pages render.
- Zero CSP-violation messages in the console.
- WebSocket connection on the terminal page completes; commands echo.

**Sign-off:**
- [ ] Tested from device: ___________ (phone make/model, browser version)
- [ ] LAN URL reachable: yes / no
- [ ] Self-signed cert warning dismissed cleanly: yes / no
- [ ] Basic Auth accepted: yes / no
- [ ] Dashboard CSP-clean: yes / no
- [ ] Join CSP-clean: yes / no
- [ ] Terminal CSP-clean + WS connect + keystroke echo: yes / no
- [ ] Date/operator: ___________

———

## UAT-3: Live-session Network tab audit — zero third-party requests

**Requirement:** SEC-07 SC-3

**Context:** SC-3 mandates that a real attach/resize/scroll/detach session generates zero requests to `cdn.jsdelivr.net` or any other third-party origin. This is the definitive proof that vendoring + CSP + extraction closes Finding 4's runtime CDN exposure.

**Steps:**

1. Start agenthub in Tailscale mode; open a terminal session via Share URL in Chrome or Safari on this Mac.
2. Open DevTools → Network tab. Ensure "Preserve log" is CHECKED so navigation doesn't clear the log.
3. In the Filter bar at the top of the Network tab, enter: `jsdelivr|unpkg|cdnjs|cdn\.`  (regex filter — interpret as OR)
4. Reload the terminal page (Cmd+R).
5. Interact with the terminal for at least 20 seconds:
   - Type several commands (at least 10 keystrokes total)
   - Resize the window at least twice (trigger the xterm FitAddon)
   - Scroll back through output
   - Type more commands after scroll-back
   - Close the tab (triggers WS close / detach path)

**Expected:**

- The Network tab filter must show ZERO matching rows for the entire session.
- All rows (unfiltered) should be same-origin: the Tailscale FQDN over HTTPS + WSS. The only non-same-origin request permitted is the initial DNS resolution of the Tailscale hostname, which is handled by the OS resolver and doesn't appear in DevTools' Network tab.

**Sign-off:**
- [ ] Tested in browser: ___________ (Chrome/Safari + version)
- [ ] Filter used: `jsdelivr|unpkg|cdnjs|cdn\.`
- [ ] Third-party request count: ___________ (must be 0)
- [ ] Session duration / action count: ___________
- [ ] Date/operator: ___________

———

## Milestone Sign-Off

When all three UAT items are verified, append a completion block:

```
## Phase 89 Manual UAT — COMPLETE

All three manual verification items passed on {YYYY-MM-DD} by {operator}.
- UAT-1 Safari: PASS
- UAT-2 Local-network-fallback: PASS
- UAT-3 Zero third-party requests: PASS

Phase 89 is ready for /gsd:verify-work.
```
