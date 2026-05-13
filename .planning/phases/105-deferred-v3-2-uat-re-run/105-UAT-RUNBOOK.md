---
phase: 105-deferred-v3-2-uat-re-run
type: uat-runbook
status: ready-for-human
requirements: [UAT-01, UAT-02, UAT-03, UAT-04, UAT-05, UAT-06, UAT-07]
---

# Phase 105 UAT Runbook — v3.2 Deferred UAT Re-Run

## Scope

These 7 UAT scenarios were deferred from v3.2 because they required shell sessions (Phase 100/101) and the link/find-bar polish (Phases 102/103). All prerequisites are now landed, so the UATs are unblocked.

**Why this phase is NOT auto-completed:** Each UAT is a physical-device interactive scenario (iPad + Tailscale, desktop Chrome DevTools, two-client image join, etc.). They cannot be run from an executor agent — they need a human to operate hardware, observe visual output, and judge fidelity.

---

## Required Environment

- **Desktop**: macOS dev machine with Chrome, DevTools open
- **iPad**: Physical iPad (Safari, not Chrome) on the same Tailscale network as the daemon
- **Tailscale**: Must be running on both daemon host and iPad
- **Shell sessions**: Available via `agenthub new shell` (Phase 100/101 — already shipped)
- **Test fixture binaries**: `chafa`, `curl`, basic shell utilities

---

## UAT-01 — WebGL context-loss → DOM fallback (Desktop Chrome)

**Goal**: Banner shows when WebGL context is forcibly lost, auto-dismisses after 8s, terminal falls back to DOM renderer.

**Steps**:
1. Open AgentHub on macOS desktop (Wails build)
2. Spawn a shell session (`agenthub new shell` or via GUI new-session modal)
3. Open browser/wails DevTools → Console
4. Run: `document.querySelector('canvas').getContext('webgl').getExtension('WEBGL_lose_context').loseContext()`
5. **Expected**: Recovery banner appears at top of terminal, terminal switches to DOM renderer, terminal content remains readable, banner auto-dismisses after 8 seconds.

**Pass criteria**: Banner shows + auto-dismisses 8s + terminal stays usable.

---

## UAT-02 — iPad software-rasterizer banner

**Goal**: When loading the web-served session on iPad Safari, a banner indicates software-rasterizer mode (iPad WebGL is software-rasterized on some hardware).

**Steps**:
1. On desktop: spawn a shell session, enable web serving (confirm the new ShellWebShareBanner appears one-time → "Enable web sharing")
2. Note the Tailscale URL
3. On iPad Safari: open the Tailscale URL
4. **Expected**: Software-rasterizer banner appears at top of terminal page

**Pass criteria**: Banner visible; iPad receives terminal output normally.

---

## UAT-03 — 10K scrollback regex search performance

**Goal**: Searching a 10,000-line scrollback with regex completes within frame-time budget (no main-thread jank).

**Steps**:
1. Spawn a shell session
2. Generate 10K lines: `for i in $(seq 1 10000); do echo "test line $i $RANDOM"; done`
3. Open DevTools → Performance → Record
4. Cmd-F to open find bar; enable regex toggle; type `\d{4}`
5. Stop recording
6. **Expected**: No frame >50ms during the search; total search completion <300ms

**Pass criteria**: Performance trace shows main thread responsive; search results render without dropped frames.

---

## UAT-04 — Web-Links full chain (iPad + Tailscale)

**Goal**: All 5 link-handling scenarios from Phase 95 work end-to-end on iPad Safari over Tailscale.

**Steps**:
1. Spawn shell session; enable web serving (confirm shell banner appears once)
2. Print test fixtures:
   ```
   cat /tmp/web-links-test.sh   # if exists; otherwise echo the fixtures manually
   echo "https://example.com"
   echo "https://пример.рф"      # IDN/Cyrillic — should trigger LinkConfirmPopover
   echo "https://paypa1.com"     # typosquat — should trigger popover
   echo "mailto:test@example.com"  # mailto — should be clickable now (POLISH-01)
   printf '\e]8;;https://evil.com\e\\Click here\e]8;;\e\\\n'  # OSC 8 mismatch
   ```
3. On iPad: open the Tailscale URL; navigate to the session
4. Cmd-click (iPad: long-press + Open Link OR keyboard Cmd-click with external keyboard) each URL
5. **Expected per URL**:
   - `https://example.com` → opens in new tab/browser
   - `https://пример.рф` → LinkConfirmPopover shows with Punycode form
   - `https://paypa1.com` → LinkConfirmPopover shows typosquat warning
   - `mailto:test@example.com` → mail app opens with addressed compose
   - OSC 8 mismatch → LinkConfirmPopover shows display-text vs href-mismatch warning

**Pass criteria**: All 5 paths behave as documented in UI-SPEC.

---

## UAT-05 — chafa visual fidelity (sixel / iTerm2 IIP decision)

**Goal**: Compare desktop and web rendering of `chafa` image output to verify visual fidelity parity.

**Steps**:
1. Spawn a shell session on desktop
2. Run `chafa --format=sixel <some-image.png>` (or `--format=symbols` if sixel unavailable)
3. Take screenshot of desktop rendering
4. Open same session on web (Tailscale URL); compare rendering
5. **Note**: Per Phase 103 POLISH-05 decision, AgentHub supports sixel only; iTerm2 IIP (OSC 1337) is explicitly out-of-scope. Test with sixel format only.

**Pass criteria**: Desktop and web renderings are visually equivalent (allowing for minor anti-alias differences).

---

## UAT-06 — Two-client mid-stream image join

**Goal**: When two web clients connect to the same session and an image is streaming, the second client receives byte-fidelity replay.

**Steps**:
1. Spawn shell session; enable web serving
2. Open the Tailscale URL on **two** browser windows (or one iPad + one desktop)
3. In the shell: start a long `chafa --format=sixel` render of a big image, OR a streaming command that emits sixel escapes over a few seconds
4. Mid-stream, refresh the second client (or open it fresh)
5. **Expected**: Second client renders the same final image as the first — no partial garbage, no missing rows

**Pass criteria**: Both clients show identical image output.

---

## UAT-07 — iPad 5-scenario runbook

**Goal**: Verify all 5 Phase 99 iPad-specific scenarios still work end-to-end.

**Steps** (run each on physical iPad Safari over Tailscale):

1. **Attach + chafa**: Open session, run `chafa` on a small image, verify sixel renders.
2. **Scrollback**: Generate 1000 lines, scroll up via iPad gestures, verify scrollback works.
3. **Zero-CDN**: DevTools (if available, otherwise inspect Network on a paired desktop) → confirm no external CDN requests.
4. **Zero-CSP-relaxation**: Confirm no CSP errors in console.
5. **All-8-plugins-ON Progress**: Enable every plugin via settings; verify Progress indicator shows correctly during plugin toggles.

**Pass criteria**: All 5 scenarios pass without regression.

---

## Test Fixture Sources

Some UATs reference fixtures that may need to be created:

- `/tmp/web-links-test.sh` (UAT-04): Was created during Phase 95 development. If absent, the inline `echo` commands above are sufficient.
- Test images for UAT-05/06: Any PNG/JPG in your home directory works.

---

## Sign-off

When each UAT passes, check the box in `.planning/REQUIREMENTS.md` and add a note to `.planning/phases/105-deferred-v3-2-uat-re-run/105-SUMMARY.md` (create it). When all 7 pass, the phase is complete.

If any UAT fails: file a bug, create a `.planning/quick/` task to address, do NOT block milestone close.

---

## Status (as of generation)

All 7 UATs are **ready for human execution**. No autonomous test can substitute. The autonomous v3.3 phase pipeline has put the prerequisites (shell sessions, web-links polish, find-bar polish) in place; these UATs verify the integration as a user would experience it.
