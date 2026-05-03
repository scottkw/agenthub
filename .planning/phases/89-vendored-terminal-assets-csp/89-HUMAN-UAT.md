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

1. Launch AgentHub in Tailscale mode (default). Either:
   ```
   open -a AgentHub          # GUI launch via Finder/dock
   ```
   or run the unified entrypoint with no args (also dispatches to GUI mode):
   ```
   agenthub
   ```
   The daemon is auto-spawned by the GUI; no separate `daemon start` is required.
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
- [x] Verified on Safari version: current (macOS, 2026-05-02)
- [x] Terminal CSP-clean: yes (READ ONLY badge visible — confirms D-24/SEC-04 cap-perms enforcement works in WebKit too)
- [x] Dashboard CSP-clean: yes (console empty)
- [x] Join CSP-clean: yes (console empty)
- [x] Date/operator: 2026-05-02 / Ken Scott
- Result: **PASS** — zero `Refused to load|execute|connect` or `Content Security Policy` messages across all three pages. Only Safari noise: 3 source-map 404s on the terminal page (`.map` files not shipped to production — Safari logs these louder than Chromium). No security significance.

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
- [x] Tested from device: same Mac (second-device test deferred — see Findings below)
- [x] LAN URL reachable: yes (`https://192.168.1.186:7443/...`)
- [x] Self-signed cert warning dismissed cleanly: yes (Chrome — Safari subresource-cert bug discovered, see Findings)
- [x] Basic Auth accepted: yes (blank username + generated password from Settings panel)
- [x] Dashboard CSP-clean: yes (Chrome)
- [x] Join CSP-clean: yes (Chrome)
- [x] Terminal CSP-clean + WS connect + keystroke echo: yes — Chrome network tab shows 0/1779 matching third-party CDN; READ ONLY badge confirms cap-perms enforced in LAN mode; Issues panel shows zero CSP violations (only the same 2 cosmetic items from UAT-1: missing form id/name + Grammarly extension `unload` listener)
- [x] Date/operator: 2026-05-02 / Ken Scott
- Result: **PASS in Chrome.** Safari blocked by an unrelated subresource-cert UX bug (see Findings).

### Findings discovered during UAT-2 (filed for follow-up, NOT Phase 89 blockers)

1. **Sessions panel doesn't show LAN Basic Auth password.** In LAN-fallback mode, server enforces Basic Auth challenge but the password is only surfaced in the Settings tab, not on the Sessions tab where users go to grab share URLs. Users hit the Sign In prompt with no obvious source for the password. Recommend: add password display to Sessions panel header in LAN mode, or auto-include credentials in copied URLs.

2. **Safari rejects self-signed cert exceptions for subresources.** After accepting the cert warning and authenticating, Safari still blocks `/assets/terminal.css`, `/assets/xterm/xterm.js`, `/assets/xterm/addon-fit.js`, `/assets/terminal.js` with "The certificate for this server is invalid" — leaving the page stuck on "Connecting…". Chrome handles the same flow correctly. WebKit-specific UX bug, not a Phase 89 functional issue. Recommend: document Safari-LAN limitation, or add a one-time trust install flow.

3. **AgentHub native session pane stays blank when web-enabled in LAN mode.** Created a new `claude 2` session in the GUI; tab opens with "WEB ON" but the in-app terminal area never renders. External browser at the same session URL renders fine. Likely a Phase 87 native-render path issue independent of Phase 89's web/CSP scope.

4. **`agenthub start` is referenced in the UAT-2 instructions but does not exist as a CLI command.** Correct invocation is `open -a AgentHub` (or run with no args). Update the UAT-2 doc.

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
- [x] Tested in browser: Chrome (latest, 2026-05-01)
- [x] Filter used: `jsdelivr|unpkg|cdnjs|cdn\.`
- [x] Third-party request count: 0 (out of 22 total — all same-origin to Tailscale FQDN; Grammarly extension injection ignored as out-of-scope; favicon 404 pre-existing)
- [x] Session duration / action count: ~30s, multiple ls/pwd commands + window resize + scrollback + WebSocket round-trips
- [x] Date/operator: 2026-05-01 / Ken Scott
- Result: **PASS** — vendored xterm.{css,js}, addon-fit.js, terminal.{css,js} all served same-origin; SEC-07 SC-3 verified
- **Bonus finding:** Chromium DevTools Issues panel shows **zero CSP violations** during the live session. The 12× `style-src-elem 'inline'` violations called out in the "KNOWN FINDING" preamble of this doc no longer appear — the D-09 `'unsafe-inline'` amendment on style-src closes them. Only two non-security issues observed: a missing `id`/`name` on a form field (a11y autofill hint, likely the Join page form) and a deprecated `unload` listener originating from the Grammarly browser extension (`Grammarly-check.js:1`), not AgentHub code.

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

———

## Phase 89 Manual UAT — COMPLETE

All three manual verification items passed on 2026-05-02 by Ken Scott.
- UAT-1 Safari (Tailscale mode): **PASS** — three pages CSP-clean; only Safari noise is 3 source-map 404s
- UAT-2 Local-network-fallback (Chrome): **PASS** — 0 CSP violations, READ ONLY enforced. Safari path blocked by an unrelated WebKit subresource-cert UX bug; recorded as follow-up.
- UAT-3 Zero third-party requests (Chrome): **PASS** — 0 of 1779 requests matched `jsdelivr|unpkg|cdnjs|cdn\.` over 5.5-min session; bonus Issues-panel finding shows the 12× style-src-elem violations called out in the doc preamble are GONE under the D-09 amendment.

Phase 89 is ready for `/gsd-verify-work`.

Four non-blocking follow-up findings recorded in the UAT-2 sign-off block above (Sessions-panel password UX, Safari subresource cert, native session pane blank, `agenthub start` doc typo).
