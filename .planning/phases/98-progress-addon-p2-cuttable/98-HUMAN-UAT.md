---
phase: 98
type: human-uat
created: 2026-05-08
signed_off: 2026-05-11
tester: Ken Scott
build: v3.2-dev @ c6c6a81
requirements: [PRG-01, PRG-02, PRG-03]
plans: [98-01, 98-02, 98-03, 98-04, 98-05]
status: approved
result: 3 desktop scenarios pass, web parity addendum pass, static OFF-path invariant GREEN
---

# Phase 98 Human UAT — Progress Addon (PRG-02 per-tab underline, PRG-03 tray glyph, PRG-01 cuttability)

> Run these AFTER Plans 98-01 through 98-05 are complete and the standard
> test gates are green (`go test ./... -count=1`, `cd frontend && pnpm test --run`,
> `cd frontend && pnpm tsc --noEmit`).

## Setup

```bash
# Build the desktop app with vendored web assets.
wails build -tags wailsassets

# Open the built app.
open build/bin/AgentHub.app   # macOS
# or run the binary directly on Linux/Windows
```

Confirm the app launches without errors and at least one terminal session is available.

---

## Scenario 1 — PRG-02 Desktop: Per-tab progress underline visible from OSC 9;4 events

**Why manual:** Live OSC 9;4 → CSS transform pipeline — automated tests cover the source-scan invariants and the scaleX transform logic (vitest) but cannot exercise the full cross-component event chain at runtime (ProgressAddon onChange → onProgressChange prop → App.tsx tabProgress registry → TabBar.tsx .tab__progress CSS). The 200ms debounce and smooth CSS transition require human visual confirmation.

**Setup:** Open AgentHub. Open Settings → Plugins. Confirm the "Progress indicator" toggle shows as **OFF** by default (v3.2 default; the italic caption reads "Default OFF in v3.2 — flips ON in v3.3 after field validation."). Toggle Progress **ON**. Click Save. Wait for the three-state save indicator to reach "saved".

**Verify:**
1. Open a new terminal tab (or use an existing one).
2. In the terminal, run: `bash tests/fixtures/osc94-progress-fixture.sh`
   (If the fixture is unavailable, manually emit via: `printf '\033]9;4;1;25\007'; sleep 1; printf '\033]9;4;1;50\007'; sleep 1; printf '\033]9;4;1;75\007'; sleep 1; printf '\033]9;4;1;100\007'; sleep 1; printf '\033]9;4;0;0\007'`)
3. Observe the active tab in the tab strip:
   - **Expected:** A thin (#7aa2f7 TokyoNight-accent) underline appears at the bottom of the active tab's tab-strip entry.
   - **Expected:** The underline grows smoothly from ~25% → ~50% → ~75% → ~100% width as each step fires (~1s apart due to the fixture's sleep).
   - **Expected:** The underline disappears smoothly (collapses to scaleX(0)) on the final clear emit.
4. Open a second terminal tab. Confirm the underline does NOT appear on the second tab while the fixture runs in the first (per-tab isolation).
5. Animation should be smooth (no jank, no flicker, no layout reflow).

Pass: [x]   Fail: [ ]   Notes: Injection path adapted — tab fixture-as-CLI override (Settings → CLI Paths) used instead of in-tab `bash …` since AgentHub tabs spawn agent CLIs, not a shell. All five expected behaviors confirmed by tester on macOS.

---

## Scenario 2 — PRG-03: Cross-session aggregate tray glyph quartile transitions

**Why manual:** System tray icon rendering is OS-level — no headless e2e framework can observe or assert on the macOS menu bar icon / Windows notification area icon. Quartile transition timing (200ms debounce) and Go-side idempotency gating require visual confirmation.

**Setup:** Continue from Scenario 1 (Progress toggle ON). Open 3 terminal tabs.

**Verify:**
1. In tabs 1 and 2, run the OSC 9;4 fixture (or manually emit progress sequences). Leave tab 3 idle (no progress).
2. As the active sessions' progress values increase, observe the system tray icon:
   - **Expected (macOS):** The menu bar icon cycles through quartile glyphs as the aggregate mean crosses 25/50/75/100 thresholds. The glyphs are distinct (visually different from the base AgentHub icon).
   - **Expected:** The tray icon does NOT rapid-swap during the bursting phase between quartile boundaries (the 200ms debounce + Go-side idempotency gate prevents flicker).
3. Let the fixtures finish (or emit `\033]9;4;0;0\007` in tabs 1 and 2).
   - **Expected:** The tray icon reverts to the base AgentHub icon once all active progress is cleared.
4. Re-open the OSC 9;4 fixture in only one tab. Note which quartile the single-tab aggregate maps to.

Pass: [x]   Fail: [ ]   Notes: Adapted with fixture-as-CLI override (2 fixture tabs + 1 idle non-fixture tab). Quartile cycling, no rapid flicker, revert-to-base confirmed by tester on macOS.

---

## Scenario 3 — PRG-01: Cuttability OFF-toggle smoke

**Why manual:** The OFF-toggle path requires live verification that addon disposal propagates correctly through the React component tree (TerminalPanel disposes → onProgressChange emits state:0 → App.tsx clears tabProgress → TabBar.tsx .tab__progress collapses). Automated tests assert the source-scan invariant (TestPRG_OffPath_NoProgressLogic) and the off-path static structure (TestPRG_NewProgressAddonIsGated), but cannot exercise the runtime disposal chain.

**Setup:** Continue from Scenario 1/2 (Progress currently ON, at least one terminal showing an underline from a recent fixture run).

**Verify:**
1. While at least one `.tab__progress` underline is visible (or has been visible recently), open Settings → Plugins.
2. Toggle Progress **OFF**. Click Save. Wait for the "saved" indicator.
3. **Expected:** Every active `.tab__progress` element disappears — underlines collapse to scaleX(0) (invisible) immediately on toggle, not on next session reload.
4. **Expected:** The tray icon reverts to the base AgentHub icon (or remains base if it was already there).
5. Re-run the OSC 9;4 fixture in any terminal tab.
   - **Expected:** NO underline appears in the tab strip (the ProgressAddon has been disposed; onChange is unsubscribed).
   - **Expected:** The tray icon does NOT update (SetTrayProgress is not called when the addon is off).
6. Open browser DevTools (or check Console.app on macOS): confirm no JS errors related to ProgressAddon disposal.
7. Run the static OFF-path invariant test:
   ```bash
   go test ./internal/release -run TestPRG_OffPath_NoProgressLogic -count=1
   ```
   **Expected:** GREEN (the static invariant is intact — no polling patterns exist).

Pass: [x]   Fail: [ ]   Notes: OFF-toggle immediately collapses active underlines and reverts tray glyph (tester-confirmed). Re-running fixture with Progress OFF produced no underline and no tray update. `go test ./internal/release -run TestPRG_OffPath_NoProgressLogic -count=1` → GREEN (0.07s). DevTools check skipped — production Wails build (`-tags wailsassets`) ships without DevTools; not a regression.

---

## Web Parity Addendum — Web-Served Session (Optional)

> This step is only required if a Tailscale-served or LAN-accessible session URL
> is available. Skip on isolated dev machines.

**Setup:** Start the agenthub daemon with web sharing enabled. Obtain a session URL with a capability token.

**Verify:**
1. Navigate to the session URL in a browser (Chrome or Firefox recommended).
2. With Progress toggle ON (from Scenario 1 setup), emit OSC 9;4 sequences in the session.
   - **Expected:** A thin `#7aa2f7` progress bar appears at the very top of the browser viewport (position: fixed; top: 0), growing from scaleX(0) toward scaleX(1) as values increase.
   - **Expected:** The bar disappears smoothly on the clear emit.
3. Open DevTools → Console. Confirm no CSP violations or JS errors related to ProgressAddon.
4. The web-served page has no tab strip (Pitfall #10 in 98-RESEARCH.md) — the fixed-position top bar is the intentional web-side analog.

Pass: [x]   Fail: [ ]   Notes: Verified via Tailscale-served session URL on a fresh `/tmp/osc94-web-uat.sh` fixture-CLI tab (long-lived variant to accommodate URL grab + browser open). Top progress bar grew through 25→50→75→100 and collapsed smoothly. DevTools Console clean — no CSP violations, no ProgressAddon errors.

---

## Final Sign-Off

- [x] Scenario 1 (per-tab underline) passes on macOS
- [x] Scenario 2 (tray glyph quartile transitions) passes on macOS
- [x] Scenario 3 (OFF-toggle cuttability smoke) passes on macOS
- [x] (Optional) Web parity addendum verified
- [ ] (Optional) Verified on Linux: ___________
- [ ] (Optional) Verified on Windows: ___________
- [x] No CSP violations in browser DevTools (web-served session)
- [x] `go test ./internal/release -run TestPRG_OffPath_NoProgressLogic -count=1` is GREEN

**Tester:** Ken Scott
**Date:** 2026-05-11
**Build:** AgentHub `v3.2-dev @ c6c6a81` (HEAD of main; wails build -tags wailsassets)

**Notes / issues observed:**

```
Injection path adapted for AgentHub's product surface:
- The runbook as authored assumes "open a terminal tab and run `bash …`",
  but AgentHub sessions spawn registered agent CLIs (claude / codex / opencode)
  — there is no general shell session type.
- Claude Code's `!` prefix runs commands in the background and does NOT pass
  raw OSC 9;4 escape bytes through to the parent pty's stdout (so the bytes
  never reach AgentHub's ProgressAddon).
- Workaround used: temporary CLI override in Settings → CLI Paths pointing the
  `opencode` slot at the fixture script. Sessions spawned with that "agent"
  run the fixture, which emits OSC 9;4 directly into the tab pty where
  ProgressAddon catches it. For the web parity addendum the override was
  re-pointed at /tmp/osc94-web-uat.sh (long-lived variant) to give time to
  grab the session URL and open it in a browser.

Out-of-scope observations (not Phase 98 regressions, not blocking):

1. Fast-exiting fixture-CLI sessions trigger an "exited with error · running
   · Exit code: -1 · Duration: 4s" toast in AgentHub. Pre-existing fast-exit /
   go-pty ProcessState race documented in engine.go:271-273. Phase 98 did not
   touch session-exit handling; symptom only surfaces because we used a
   4-second script as a "CLI". Not a PRG-01/02/03 finding.

2. Production Wails build (`-tags wailsassets`) ships without DevTools, so the
   runbook step "open browser DevTools (or Console.app)" is not exercisable
   from inside the desktop app. The browser-side web-parity DevTools check
   covered the equivalent verification.

Backlog item suggested:
- Add a first-class "shell session" type (user-selectable bash/zsh/pwsh) as a
  v3.3+ feature, with appropriate security gating for web-shared shells.
  Would obsolete the fixture-as-CLI override hack and unlock other UX wins
  (git, log tailing, curl progress observation, etc.). Deferred from v3.2
  release gate.
```

---

## Cuttability Note

If any scenario fails, do NOT mark this UAT as approved. Describe the failure and the
affected requirement (PRG-01/PRG-02/PRG-03) so a gap-closure plan can be authored via
`/gsd-plan-phase 98 --gaps`.

Wave 4 (Plan 98-05: web parity + e2e + this UAT runbook) is the explicitly cuttable
wave for Phase 98. Dropping Wave 4 leaves the desktop fully functional:
- PRG-02 per-tab underline: complete (Waves 1-3)
- PRG-03 tray glyph: complete (Waves 1-2)
- PRG-OFF regression tests: GREEN regardless

The vendored `web/vendor/xterm/addons/addon-progress.js` asset (Wave 0) stays whether
or not Wave 4 lands — vendor_drift_test.go stays GREEN in all cases.
