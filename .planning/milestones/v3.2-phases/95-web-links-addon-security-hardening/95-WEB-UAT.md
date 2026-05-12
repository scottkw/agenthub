---
status: deferred
deferred_to: v3.3
deferred_on: 2026-05-12
deferred_reason: "Runbook prints URLs into raw shell on Tailscale-served terminal page. v3.2 ships agent sessions only; shell session type on v3.3+ backlog (see v3.2-RELEASE-BLOCKERS.md)."
phase: 95-web-links-addon-security-hardening
type: web-uat
---

# Phase 95 — Web (Tailscale-served) UAT Runbook

> **🛑 DEFERRED to v3.3 (decided 2026-05-12)** — Runbook prints URLs into a raw shell on a Tailscale-served terminal page. AgentHub v3.2 ships **agent sessions only**; raw shell session type is on the v3.3+ backlog per `.planning/v3.2-RELEASE-BLOCKERS.md`. 95-VERIFICATION.md status is `deferred`. Re-open once shell sessions land.

Manual UAT runbook for Phase 95 web-links security hardening on the
**Tailscale-served terminal page**. Use the **dev-browser skill** (per
project-root `CLAUDE.md` workflow guidance) to automate the steps below
when running this from inside an agent context — do NOT ask the user to
manually verify browser behavior.

## Prerequisites

- agenthub running on a host with Tailscale active (the host emits the
  `https://<machine>.<tailnet>.ts.net/sessions/<id>?cap=…` share URLs).
- A second device on the same tailnet with a modern browser
  (Chrome / Safari / Firefox; iPad Safari is the canonical mobile
  target).
- A session shared via the share-URL flow (Settings → Sessions → Share).
- Plugin config defaults (Clickable web links = ON, Modifier = Platform,
  all confirm flags ON).

## 1. Vendored asset reachable

1. Open the share URL in the second-device browser.
2. Open DevTools → Network. Filter by `addon-web-links.js`.
3. **Expected:** status 200, content-type `application/javascript`,
   served from same origin (no CDN, no third-party domain). Path is
   `/assets/xterm/addons/addon-web-links.js`.

## 2. CSP zero-violation (LNK-01 / web-side)

1. DevTools → Console.
2. Reload the page; perform any other interactions in subsequent steps.
3. **Expected:** ZERO CSP violation messages anywhere in the console.
   The addon UMD is same-origin; CSP `script-src 'self'` is honored.

## 3. Scheme allowlist (LNK-01)

1. In the session, run `echo javascript:alert(1)`.
2. Hover the rendered text. **Expected:** the addon does NOT decorate as
   clickable (no underline, no pointer cursor).
3. Run `echo https://example.com`. Hover. **Expected:** decorated and
   clickable.

## 4. Modifier-click + hover tooltip (LNK-02)

1. Run `echo https://example.com`. Hover the URL.
2. **Expected:** the native browser tooltip (title attribute) shows
   `https://example.com`.
3. Single-click (no modifier). **Expected:** nothing happens.
4. Cmd-click (mac) / Ctrl-click (linux/win, including paired keyboards
   on iPad). **Expected:** `https://example.com` opens in a new browser
   tab. The new tab's URL bar shows `https://example.com`.

## 5. window.open with noopener (LNK-04)

1. After the new tab opens (step 4), open DevTools in that new tab.
2. Console: `window.opener`.
3. **Expected:** `null`. The `noopener,noreferrer` flag stripped the
   opener reference. NO `Referer` header is sent on the request that
   opened example.com (verify in DevTools → Network → request headers).

## 6. IDN / typosquat popover (LNK-03)

1. Same fixtures as the desktop UAT runbook:
   - `echo https://gооgle.com` (Cyrillic U+043E — copy verbatim).
   - `echo https://paypa1.com`.
2. Cmd-click / Ctrl-click each URL.
3. **Expected:** the web-side popover (plain DOM at
   `#link-confirm-popover`, role="dialog", aria-modal="true") appears
   anchored at the click point.
4. The popover **textContent** displays the URL as typed (no
   normalization to Punycode in the visible code element). The URL
   element is `<code id="link-confirm-url">` — verify via DevTools that
   it was set via `textContent` (NOT `innerHTML`). DevTools → Elements:
   the `<code>` should contain ONLY a text node, no child elements.
5. Click **Cancel**. **Expected:** popover hides (`hidden` attribute);
   no new tab opens.
6. Cmd-click / Ctrl-click again. Click **Continue**. **Expected:**
   `window.open` fires; new tab opens with the URL.
7. Press **Escape** while the popover is open. **Expected:** popover
   dismisses (Escape handler on document).

## 7. Live toggle propagation via SSE (LNK-05)

1. On the host (desktop agenthub): toggle Clickable web links **OFF** →
   Save.
2. On the web client (DevTools → Network → EventStream filter): observe
   the `/api/plugin-config/stream` SSE channel emit a `plugin-config`
   event within ~2 seconds.
3. In the web terminal (no page reload), echo a fresh
   `https://example.com`. Hover. **Expected:** the addon was disposed —
   the URL is NOT decorated as clickable. Cmd-click does nothing.
4. Toggle Clickable web links back **ON** → Save. Echo another URL;
   hover. **Expected:** the URL is clickable again. NO PAGE RELOAD
   required.

## 8. Sub-key toggle propagation (Pitfall #8)

1. With "Clickable web links" ON, change Modifier on the host from
   `Platform` to `None`; Save.
2. SSE event arrives on the web client.
3. In the web terminal (no reload), single-click an already-rendered
   `https://example.com`. **Expected:** opens in a new tab immediately
   (modifier no longer required).
4. **Verify NO re-attach:** the previously-rendered URL must already be
   single-clickable WITHOUT requiring a fresh hover; the addon was NOT
   re-attached on the sub-key change (the `currentWebLinksConfig`
   variable was updated and read at click time).

## 9. iPad Safari Tailscale (Phase 99 release gate)

This is the **canonical mobile target** per ROADMAP. Must run on a real
device.

1. On an iPad (any model, iOS 17+) with Tailscale active and connected
   to the same tailnet, open the share URL in Safari.
2. Verify the Safari address bar shows the Tailscale FQDN
   (`<machine>.<tailnet>.ts.net`), not localhost / IP.
3. Run through steps 1–7 above. iPad-specific notes:
   - With a paired keyboard (Magic Keyboard / Brydge), Cmd-click and
     Ctrl-click work as expected.
   - Without a paired keyboard, single-tap is the only available click;
     the modifier check fails so URLs do NOT navigate (this is the
     correct behavior — no accidental navigation from a misread tap).
   - The popover uses `position:fixed` — verify it stays within the
     viewport after edge-clipping mitigation triggers near the right
     and bottom edges.
4. Document any Safari-specific pitfalls in this file (append a section).

## Sign-off

- [ ] Vendored asset reachable (200, javascript content-type, same origin)
- [ ] CSP zero-violation
- [ ] Scheme allowlist (javascript: not clickable)
- [ ] Modifier-click + hover tooltip
- [ ] window.opener === null in opened tabs
- [ ] IDN / typosquat / Cyrillic popover (textContent only — no HTML injection)
- [ ] Live toggle propagates via SSE in < 2s without page reload
- [ ] Sub-key (Modifier) toggle propagates without addon re-attach
- [ ] iPad Safari Tailscale verified (with and without paired keyboard)

**Tester:** _________________________
**Date:** ___________________________
**Host build:** _____________________ (commit ____________)
**Client browser / OS:** ____________
**Tailnet name:** ___________________
**Pass / Fail:** ____________________

## dev-browser skill walkthrough (agent-driven UAT)

When this runbook is being executed by an agent (e.g. during
`/gsd:verify-work 95`), use the dev-browser skill verbatim:

```
1. dev-browser navigate <share-url>
2. dev-browser eval "document.getElementById('terminal').focus()"
3. dev-browser type "echo https://example.com\r"
4. dev-browser eval "window.__opens = []; var o = window.open;
                     window.open = function(u, t, f) { window.__opens.push([u,t,f]); return null; };"
5. dev-browser click <selector for the rendered URL DOM node> --modifier=Cmd
6. dev-browser eval "JSON.stringify(window.__opens)"
   Expected: [["https://example.com","_blank","noopener,noreferrer"]]
7. Continue through steps 6–8 of the manual runbook with the same
   eval-spy pattern.
```
