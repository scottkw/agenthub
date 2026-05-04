# Phase 93 — iPad Safari Manual UAT

> Manual UAT script for the runtime behaviors that headless Playwright cannot reproduce on iPad Safari over Tailscale. Run during `/gsd-verify-work 93` before phase sign-off.

## Prerequisites
- iPad with Safari, joined to the same Tailnet as the dev Mac running AgentHub
- AgentHub running with web-server enabled, at least one terminal session created and shared with a capability link
- The Tailscale URL of the form `https://<tailnet-fqdn>.ts.net/sessions/<id>?cap=<token>`

## UAT-1: WebGL Software-Rasterizer Preemption (WGL-03)

1. Open the Tailscale URL in iPad Safari (the iPad reports software-rasterized WebGL via ANGLE Metal Renderer).
2. Within 5 seconds, an in-page banner above the terminal scrollback should appear with the EXACT copy:
   > Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience.
3. Confirm:
   - Banner has a 53px height (visually one line of text + small padding)
   - Banner has a thin blue (`#7aa2f7`) accent strip on the LEFT edge
   - Banner has a × button on the RIGHT edge that dismisses the banner when tapped
   - Banner does NOT auto-dismiss (wait 30 seconds; banner persists until tapped)
   - Banner only shows ONCE per session (reload the page; banner does NOT reappear in the same Safari tab — verified via sessionStorage)
4. Type a few commands in the terminal; confirm output renders correctly via the DOM renderer (no GPU artifacts, no missing characters).

PASS criteria: All four sub-checks confirmed.

## UAT-2: WebGL Context Loss → DOM Fallback (WGL-02)

> Run on a desktop browser (Chrome) where WebGL is hardware-accelerated. iPad Safari already preempts to DOM, so context-loss runtime testing requires a hardware-accelerated browser.

1. Open the Tailscale URL in Chrome on the dev Mac.
2. Confirm the WebGL renderer is active (no software-preemption banner appears at startup).
3. Open DevTools Console.
4. Run:
   ```
   const canvas = document.querySelector('#terminal canvas')
   const gl = canvas.getContext('webgl2') || canvas.getContext('webgl')
   gl.getExtension('WEBGL_lose_context').loseContext()
   ```
5. Within 1 second, the in-page banner should appear with the EXACT copy:
   > Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact.
6. Confirm:
   - Banner styling matches UAT-1 (53px, blue accent, × button)
   - Scrollback is intact (any output before the context loss is still visible)
   - No auto-retry: confirm via DevTools Network tab that NO new request to `/assets/xterm/addons/addon-webgl.js` is fired after the loss
   - Banner auto-dismisses after 8 seconds (set a 10-second timer and confirm the banner disappears between 7–9 seconds)
   - Reloading the page: banner does NOT reappear in the same browser tab (sessionStorage one-shot per session)

PASS criteria: All five sub-checks confirmed.

## UAT-3: Hot-Swap Across Open Desktop Terminals (WGL-01)

> Run on the desktop AgentHub app.

1. Open the AgentHub Wails app; create two terminal sessions in different tabs.
2. Open Settings → Plugins; confirm the WebGL toggle is ON.
3. Toggle WebGL OFF; press "Save Plugins"; confirm the button cycles "Saving…" → "Saved!".
4. Switch to each terminal tab; confirm:
   - Output continues to render correctly (DOM renderer engaged silently)
   - Scrollback is intact in both tabs
   - No flicker, no banner, no in-app notification (silent hot-swap per UI-SPEC §"Web plugin-config live update" applied to desktop)
5. Toggle WebGL ON; Save; confirm the renderer hot-swaps back without flicker or scrollback loss.

PASS criteria: All confirmations pass on BOTH tabs.

## UAT-4: Unicode 11 Italic Caption (U11-01)

1. Open Settings → Plugins.
2. Visually confirm the row labelled "Unicode 11 widths" has — directly under its existing description ("Correct cell widths for emoji and CJK characters using the Unicode 11 width tables.") — a SECOND italicized paragraph reading EXACTLY:
   > Applies to new sessions you create.
3. Confirm the italic paragraph color matches the existing description color (`#9aa5ce`, a muted text tone — not pure white).
4. Toggle Unicode 11 OFF; press Save; switch to an existing terminal tab. Confirm output renders identically (already-open terminals NOT affected per the italic affordance).
5. Create a NEW session; confirm Unicode 11 width tables are NOT applied (test with an emoji like `echo "📌"` — the cursor advance should reflect the toggle state).
6. Toggle Unicode 11 back ON; Save; create another new session; confirm the emoji cell width is correct.

PASS criteria: Italic caption verbatim, correct color, next-session-only behavior confirmed.

## UAT-5: Web Page Zero-CDN Real-Network Audit (WEB-02 manual)

1. Open Chrome on the dev Mac. Open DevTools → Network.
2. Filter for: `cdn.jsdelivr.net OR unpkg.com OR esm.sh`.
3. Visit the Tailscale URL and run a full session: attach, type a few commands, scroll up to read history, scroll back down, type more, detach.
4. Confirm: zero matches in the filtered network log.

PASS criteria: Zero CDN requests across the full session.

## Sign-Off

- [ ] UAT-1 PASS
- [ ] UAT-2 PASS
- [ ] UAT-3 PASS
- [ ] UAT-4 PASS
- [ ] UAT-5 PASS

Once all 5 UATs PASS, mark `93-VALIDATION.md` § Validation Sign-Off all checkboxes and proceed with `/gsd-verify-work 93`.
