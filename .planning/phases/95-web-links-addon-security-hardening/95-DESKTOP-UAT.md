# Phase 95 — Desktop UAT Runbook

Manual UAT runbook for Phase 95 web-links security hardening on the
**desktop (Wails) build**. Complements the automated test suite — the
behavior paths verified here are jsdom-untestable (xterm canvas + xterm
hover decorations + OS browser launch via `BrowserOpenURL`).

## Prerequisites

- Build a desktop binary with embedded assets:
  ```bash
  wails build -tags wailsassets
  ```
- Install / open the resulting macOS / Linux / Windows binary.
- Open one terminal session in agenthub.
- Settings → Plugins: confirm "Clickable web links" is enabled (default).
- Settings → Plugins → Web links: defaults are `Modifier=Platform`,
  Confirm-OSC8/IDN/Typosquat = ON.

## 1. Scheme allowlist (LNK-01)

1. In an active session, run:
   ```sh
   printf 'visit javascript:alert(1) and file:///etc/passwd\n'
   ```
2. Hover both URLs. **Expected:** NEITHER is decorated by the addon as a
   clickable link (xterm core may visually render the text but the addon's
   underline / cursor change MUST be absent).
3. Run `echo https://example.com`. **Expected:** the URL is underlined and
   the cursor changes to a pointer on hover.

## 2. Modifier-click + hover tooltip (LNK-02)

1. Run `echo https://example.com`.
2. Hover the URL. **Expected:** the native browser-like tooltip (DOM
   `title` attribute) shows `https://example.com`.
3. Single-click (no modifier). **Expected:** nothing happens.
4. **macOS:** Cmd-click. **Linux / Windows:** Ctrl-click. **Expected:**
   `https://example.com` opens in the **OS default browser** via the Wails
   runtime opener — NEVER inside the Wails WebView.
5. Move the mouse off the URL. **Expected:** the tooltip is removed
   (`removeAttribute('title')`); hovering empty terminal cells shows no
   stale tooltip text (Pitfall #10).

## 3. IDN / Cyrillic spoof popover (LNK-03)

1. Open a fresh terminal pane (or scroll above the prior URLs) so the
   addon picks up any settings change.
2. Paste the literal Cyrillic URL:
   ```sh
   echo https://gооgle.com
   ```
   **Important:** the two `о` chars MUST be Cyrillic small `o` (U+043E),
   NOT Latin `o` (U+006F). Copy from this line directly — do NOT retype.
3. Cmd-click (mac) / Ctrl-click (linux/win) the URL.
4. **Expected:** a popover appears anchored at the click point, captioned
   `Confirm link destination`, with the IDN risk copy ("This link contains
   internationalized characters that can spoof familiar domains."). The
   URL shown in the popover code element is the **Cyrillic form**
   (textContent — NOT normalized to Latin / NOT escaped to Punycode in the
   display).
5. Click **Cancel**. **Expected:** popover dismisses; no browser
   navigation; focus returns to the terminal.
6. Cmd-click the same URL again. Click **Continue**. **Expected:** the
   OS default browser opens — most modern browsers will then display the
   resolved Punycode form (`xn--ggle-...`) in the address bar; that is
   correct behavior.
7. Press **Escape** while the popover is open. **Expected:** popover
   dismisses (Pitfall #3 — Escape on document, not just on the dialog).

## 4. Typosquat popover (LNK-03)

1. Run `echo visit https://paypa1.com (note the digit 1, not letter l)`.
2. Cmd-click / Ctrl-click the URL.
3. **Expected:** popover with typosquat copy ("This domain matches a
   known impersonation pattern."). URL textContent shows `paypa1.com`.

## 5. (Plan A only) OSC 8 mismatch popover (LNK-03)

1. Run:
   ```sh
   printf '\e]8;;https://evil.example\e\\click here\e]8;;\e\\\n'
   ```
   (Bash heredoc-safe variant; substitute `printf` into a script file if
   needed.)
2. Hover the displayed text "click here". **Expected:** tooltip shows
   `https://evil.example` (the resolved href, NOT the display text).
3. Cmd-click / Ctrl-click. **Expected:** popover with osc8 copy ("This
   link displays one address but points to another...").

## 5b. (Plan B only — currently selected) OSC 8 known limitation

OSC 8 hyperlinks with display-vs-href divergence are **NOT detected** in
v3.2 per the Wave 0 spike outcome (95-RESEARCH §"Wave 0 Spike Outcome"):
`IBufferCell.getHyperlinkId` is not in `@xterm/xterm@6.0.0` public typings,
so we cannot discover the hyperlink-id-tagged buffer cell range from the
addon-web-links click handler. Tracked as `LNK-OSC8-FUT-01` for v3.3.

In v3.2, OSC 8 hyperlinks are still surfaced — the hover tooltip shows the
**resolved href** (xterm core's default behavior), and the popover triggers
ONLY if the href itself is a Cyrillic / typosquat domain (not because of
the display-vs-href divergence). This is a known and accepted gap.

## 6. Live toggle (LNK-05/LNK-06)

1. With a session open and a URL visible (`echo https://example.com`),
   open Settings → Plugins.
2. Toggle "Clickable web links" **OFF**. Click Save.
3. Without restarting the session, type any keystroke in the terminal
   (e.g. press Enter). **Expected:** the previously-rendered URL loses
   its underline on the next render; new URLs do NOT become clickable.
4. Toggle back **ON**; Save. Type more output (`echo https://example.com`).
   **Expected:** URLs are clickable again. NO session restart required.

## 7. Sub-key toggling without re-attach (Pitfall #8)

1. With "Clickable web links" ON, navigate to Settings → Plugins → Web
   links sub-config.
2. Change Modifier from "Platform" to "None"; Save.
3. Without restarting the session, single-click on a previously-rendered
   `https://example.com`. **Expected:** the URL opens immediately (no
   modifier required).
4. **Verify NO addon re-attach:** echo a fresh URL; hover it. **Expected:**
   the URL is clickable WITHOUT requiring a page event between Save and
   hover (confirms `webLinksConfigRef` / `currentWebLinksConfig` was read
   at click time, NOT a React useEffect dep that would have re-attached
   the addon).

## 8. Window.opener defense (LNK-04)

1. Cmd-click / Ctrl-click any URL to open it in the OS default browser.
2. In the new browser window's DevTools console, run `window.opener`.
   **Expected:** `null` (the noopener flag stripped any opener reference;
   the Wails opener uses `BrowserOpenURL` which routes through OS shell).

## Sign-off

- [ ] Scheme allowlist verified (javascript: and file:// not clickable)
- [ ] Modifier-click + hover tooltip verified (title set on hover, removed on leave)
- [ ] IDN / Cyrillic popover verified (with U+043E codepoints intact in the popover URL)
- [ ] Typosquat popover verified (paypa1.com)
- [ ] OSC 8 verified (Plan A) **OR** known-limitation acknowledged (Plan B — currently selected)
- [ ] Live toggle ON ⇄ OFF verified without session restart
- [ ] Sub-key toggle (Modifier) verified without addon re-attach
- [ ] window.opener === null in opened browser tabs

**Tester:** _________________________
**Date:** ___________________________
**Build:** wails build -tags wailsassets (commit: ____________)
**OS / version:** ___________________
**Pass / Fail:** ____________________
