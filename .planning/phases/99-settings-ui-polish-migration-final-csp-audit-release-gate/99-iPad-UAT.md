# Phase 99 — iPad Safari Manual UAT (v3.2 Release Gate)

> Manual UAT script for the runtime behaviors that headless Playwright cannot reproduce on iPad Safari over Tailscale. Run during `/gsd-verify-work 99` before v3.2 release sign-off.

## Prerequisites

- Real iPad with Safari, joined to the same Tailnet as the dev Mac running AgentHub. **iOS Simulator is NOT a substitute** (per ROADMAP SC-4 — Safari WebKit on iOS has subtle CSP / WebAssembly / network differences from desktop WebKit that the simulator hides).
- Dev Mac with Safari ≥ 16 and "Develop" menu enabled (Safari → Settings → Advanced → Show Develop menu in menu bar).
- **iPad must be USB-connected to the dev Mac** for Web Inspector remote debugging (Settings → Safari → Advanced → Web Inspector ON on iPad).
- AgentHub built with `wails build -tags wailsassets` and running. Web server enabled. At least one terminal session created and capability-shared via the share panel.
- The Tailscale URL of the form `https://<tailnet-fqdn>.ts.net/sessions/<id>?cap=<token>`.
- **Default plugin state for UAT-1 through UAT-4:** 7 ON / Progress OFF (the v3.2 default).
- **For UAT-5:** all 8 plugins ON (toggle Progress ON in Settings → Plugins → Save Plugins → reload web session URL).

---

## UAT-1: All-Plugins-Enabled Attach → Render Flow (SC-4)

1. On iPad Safari, open the Tailscale URL.
2. Within 5 seconds the terminal should attach and render its scrollback (capability cookie set, WebSocket connected, xterm DOM mounted).
3. Type `echo hello` and press Enter — observe the echo appears in the terminal.
4. From the dev Mac, in the running session, run:
   ```
   chafa --format=iterm2 -s 40x20 https://upload.wikimedia.org/wikipedia/commons/thumb/4/47/PNG_transparency_demonstration_1.png/280px-PNG_transparency_demonstration_1.png
   ```
   (or any local PNG via `chafa --format=iterm2 chart.png`).
5. Observe the inline image renders in the iPad terminal pane (sixel/IIP via @xterm/addon-image).
6. From the dev Mac, run a CLI emitting OSC 9;4 progress, e.g.:
   ```
   printf '\033]9;4;1;25\007'   # OSC 9;4 state=1 value=25
   sleep 1
   printf '\033]9;4;1;75\007'   # state=1 value=75
   sleep 1
   printf '\033]9;4;0;0\007'    # state=0 (clear)
   ```
   Confirm: NO progress underline appears (Progress plugin is OFF in UAT-1's default state).

**PASS criteria:**
- Terminal attached within 5 seconds. ✅
- Echo command displays correctly. ✅
- Inline image renders inside the iPad terminal pane. ✅
- No progress underline (Progress OFF). ✅

---

## UAT-2: Scrollback → Detach → Re-attach → Persistence (IMG-04, multi-client byte-fidelity)

1. From UAT-1's session, type `seq 1 200` and press Enter — produces 200 lines of scrollback.
2. Scroll up in the iPad terminal — confirm lines 1-200 are visible in scrollback.
3. Tap the browser back button (or close the Safari tab) to detach.
4. Re-open the Tailscale URL.
5. The session re-attaches; scrollback should be fully populated (lines 1-200 + the chafa image from UAT-1 still rendered as inline image during replay).

**PASS criteria:**
- Scrollback intact across detach/re-attach. ✅
- The inline sixel image rendered correctly during scrollback replay (multi-client byte-fidelity preserved). ✅

---

## UAT-3: Zero-CDN Audit (Phase 89 SEC-08 / Phase 99 release gate)

1. On the dev Mac with iPad USB-connected: Safari → Develop menu → "<iPad name>" → select the open Tailscale URL tab. Web Inspector opens.
2. Switch to the **Network** tab in Web Inspector.
3. Apply a domain filter: type `cdn.jsdelivr.net OR unpkg.com OR esm.sh OR fonts.googleapis.com` in the filter box.
4. From the dev Mac, drive a full session: type commands, scroll, render an image (re-run UAT-1 chafa).
5. Observe the Network tab — the filter should match **zero requests**.
6. Take a screenshot of the Network tab showing zero matches and save to `screenshots/99-iPad-UAT-3-zero-cdn.png`.

**PASS criteria:**
- Zero requests to any CDN domain during full session. ✅
- Screenshot captured. ✅

---

## UAT-4: CSP Zero-Violation Audit (SC-4 — load-bearing for v3.2 release)

1. Continue from UAT-3 with Web Inspector still open.
2. Switch to the **Console** tab.
3. Clear the console.
4. From the dev Mac, drive a full session: attach, type, scroll, render image, detach, re-attach.
5. Observe the console — there must be **zero messages** matching the regex `/csp|content security policy|refused to (execute|load|connect)/i`.
6. Take a screenshot of the empty console + the timestamp range covering the full UAT-4 session, save to `screenshots/99-iPad-UAT-4-zero-csp.png`.

**PASS criteria:**
- Zero CSP violation messages in iPad Safari console during full session. ✅
- Screenshot captured. ✅

---

## UAT-5: Second Pass — All 8 Plugins ON (Progress flipped ON)

1. On the dev Mac, open AgentHub Settings → Plugins.
2. Toggle **Progress (OSC 9;4)** ON. Click **Save Plugins**.
3. (Plan 99-01 PUI-02 will fire BannerStack toasts for unicode11/image if those were toggled too — for this UAT, only Progress is changed, so no banner is expected.)
4. On the iPad, reload the Tailscale URL (the SSE plugin-config stream should propagate the change automatically, but a hard reload is the most reliable way to confirm).
5. From the dev Mac, run the OSC 9;4 sequence from UAT-1 step 6.
6. Observe in the iPad terminal: the tab strip (or progress underline area) should now show a progress underline at the value reported by OSC 9;4 (25%, then 75%, then cleared).
7. While progress is non-zero, the tray icon on the dev Mac should show an aggregate quartile glyph (verify on dev Mac, not iPad).
8. Re-run UAT-3 and UAT-4 with all-8-ON state — confirm zero CDN requests and zero CSP violations still hold.

**PASS criteria:**
- Progress underline visible on the tab during OSC 9;4 emission. ✅
- Tray icon on dev Mac reflects aggregate quartile. ✅
- Re-run UAT-3 and UAT-4 still pass with all-8-ON. ✅

---

## Sign-Off

- [ ] UAT-1 PASS
- [ ] UAT-2 PASS
- [ ] UAT-3 PASS (screenshot captured at `screenshots/99-iPad-UAT-3-zero-cdn.png`)
- [ ] UAT-4 PASS (screenshot captured at `screenshots/99-iPad-UAT-4-zero-csp.png`)
- [ ] UAT-5 PASS

Once all 5 UATs PASS:
1. Mark `99-VALIDATION.md` § Validation Sign-Off all checkboxes complete.
2. Update `99-VERIFICATION.md` (created by `/gsd-verify-work 99`) with the iPad UAT screenshots and tester sign-off line.
3. Run `/gsd-verify-work 99` to finalize phase verification.
4. v3.2 release gate clears once `/gsd-verify-work 99` reports green.

---

*Tester:* _______________________  *Device:* iPad ___________ (model + iOS version)  *Date:* __________
