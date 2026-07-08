# Phase 175: Web-share, Remote-viewer & Windowing Bug Fixes - Research

**Researched:** 2026-07-08
**Domain:** Go relay/webserver (WebSocket terminal streaming) + React/xterm.js frontend (guest terminal rendering) + Wails desktop event bridge
**Confidence:** HIGH (all four bugs traced to exact `file:line` locations by reading the live `v4.2-funnel-sharing` branch; BUG-03's root cause is the one MEDIUM-confidence item — code-grounded but not live-confirmed)

## Summary

All four bugs were traced to precise code locations by reading the current branch, not by re-deriving from the GitHub issue text. One critical correction to the phase brief: **BUG-01's issue text points at `web/assets/terminal.js`, but that file is dead code as of Phase 159.** `GET /sessions/{id}` now unconditionally 302-redirects to `/app/?session=...&cap=...` (`internal/webserver/server.go:1296-1304`), so every real mobile guest lands on the React SPA (`WebShareSessionView` → `TerminalPanel.tsx`), not the vanilla-JS viewer. The React surface has its own byte-for-byte port of the same VIEW-05 downscale bug (`frontend/src/lib/terminalScale.ts` `computeGuestScale`, called from `TerminalPanel.tsx:172-190`), and that is the surface that must be fixed. TESTING.md's Category P (M-27/M-28) still documents `terminal.js` as "the web viewer" — that documentation is now stale and should be reconciled as part of this phase per the repo's standing Regression Test Convention.

BUG-02 (no disconnect notice) and the relay layer behind BUG-04 (blank window on reconnect) both stem from the same design gap: the WebSocket write pumps in `internal/webserver/server.go` and `internal/relay/server.go` treat `hub.Done()` (session end) as a silent `return` — no close reason is ever sent, and the JS clients (`web/assets/terminal.js`, `frontend/src/lib/relayClient.ts`) don't even read the `CloseEvent`'s `code`/`reason` fields. Fixing BUG-02 is a small, mechanical change (send a close reason, read it on the client, render a banner using the existing `WebGLRecoveryBanner`-style pattern).

BUG-04's residual scope (after Phase 168-03) was already root-caused in detail by the project owner directly in GitHub issue #119 — a rare case where the issue thread *is* the research. I independently verified every code citation in that analysis against the live branch and confirmed it is accurate: the "MDI window" complaint is the intentional Phase-134 `HubModal` preview (not a bug), already reachable via a tab affordance for remote sessions since Phase 168-03 (#118); the real bug is that `hub.ScrollbackSnapshot()` replays a truncated raw-byte ring (256 KiB, `internal/relay/scrollback.go:6`) that loses the alt-screen-enter escape sequence for long-running full-screen TUIs (Claude, Gemini-CLI, OpenCode), leaving the reconnecting viewer's xterm.js instance rendering garbage into the wrong buffer with no self-healing repaint.

BUG-03 (shared session's tab doesn't auto-close) is the one bug where static reading did **not** find a share-conditional branch — the auto-close code path is byte-identical whether or not the session was shared. The strongest code-grounded hypothesis is a **general** bug that happens to correlate with sharing: `app.go`'s exit-detection poller (`pollSessionStatus`) gives up permanently after a fixed 5-minute window from session *creation* (`app.go:386`, `deadline := time.Now().Add(300 * time.Second)`), with no re-arm and no user-visible signal. A realistic "create → share → deliver join code → verify from a guest device → exit" workflow easily exceeds 5 minutes, while an "exit immediately" comparison does not — explaining the observed correlation without sharing itself being the cause. This needs a live, timed reproduction to confirm before the plan commits to a fix.

**Primary recommendation:** Treat this phase as four independent, surgical fixes. Do not attempt a unifying refactor — the four bugs live in different files/layers (CSS-scale math, WS close semantics, VT-emulator screen replay, and a Wails-bridge poll timeout) and each has a narrow, well-understood fix site.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Mobile terminal legibility (BUG-01) | Browser / Client (React SPA) | — | Pure CSS/xterm-scale math in `frontend/src/lib/terminalScale.ts` + `TerminalPanel.tsx`; no server change needed. Dead-code sibling in `web/assets/terminal.js` is out of the reachable request path. |
| Disconnect notice (BUG-02) | API / Backend (WS close reason) | Browser / Client (render banner) | Server must emit a close code/reason on `hub.Done()`; both JS clients must read `CloseEvent.code`/`.reason` and render UI. Split responsibility, both sides required. |
| Shared-session tab auto-close (BUG-03) | Frontend Server / Desktop shell (Wails Go process, `app.go`) | Browser / Client (React event handler, already correct) | The exit-*detection* poller lives in the Wails Go process (`app.go`), not the daemon or the React frontend; the React side (`App.tsx`) already reacts correctly to whatever event it receives. |
| Blank/empty window on reconnect (BUG-04 residual) | API / Backend (relay VT-emulator replay) | Browser / Client (xterm.js render, unaffected once the server sends correct bytes) | Root cause is server-side scrollback replay losing terminal-mode state; the fix is entirely in `internal/relay` + wherever the two WS handlers (`internal/webserver/server.go`, `internal/relay/server.go`) build the on-connect preamble. |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BUG-01 | Web-share terminal legible on mobile (#128) | Confirmed fix site is `frontend/src/lib/terminalScale.ts` (`computeGuestScale`) + `TerminalPanel.tsx` `recomputeScale` (guest path), NOT `web/assets/terminal.js` (dead code post-Phase-159 redirect). See "Bug 1" section below for exact lines and fix approach. |
| BUG-02 | Remote viewer sees a disconnect notice on session end (#125) | Confirmed both WS write-pump sites (`internal/webserver/server.go:1590-1605`, `internal/relay/server.go:425-437`) close silently with no reason, and both JS clients discard `CloseEvent.code`/`.reason`. See "Bug 2" section for the exact 4-site fix (2 Go, 2 JS) and a reusable banner pattern already in the codebase. |
| BUG-03 | Exiting from inside a shared session auto-closes its tab (#126) | Confirmed the client-side auto-close mechanism (`session:exit` Wails event → `App.tsx` `handleCloseTab`) is identical for shared/unshared sessions. Primary suspect identified: `app.go:386`'s fixed 5-minute exit-poll deadline with no re-arm. Flagged MEDIUM confidence — needs live timed repro (see Open Questions). |
| BUG-04 | Host/guest session-open never lands in a dead empty window (#119) | Confirmed via GitHub issue #119's own root-cause comment (verified against live code): the "MDI window" complaint = Phase-134 `HubModal`, already given a tab affordance by 168-03/FIX-03 — no work needed there. Residual bug = raw scrollback-ring replay losing alt-screen state on reconnect, in `internal/webserver/server.go:1484-1489` AND `internal/relay/server.go:311-316` (both sites, byte-identical pattern). Fix primitive (`charmbracelet/x/vt` `Emulator.Render()`/`IsAltScreen()`) already vendored and used elsewhere in the codebase (`engine.go:761` `GetSessionStyledTailLines`). |
</phase_requirements>

## Standard Stack

No new external libraries are needed for any of the four fixes — everything is built from libraries already vendored in this codebase.

### Core (already in use, no version change needed)

| Library | Version (from go.mod / package.json) | Purpose | Why no new dependency |
|---------|---------|---------|--------------|
| `github.com/coder/websocket` | v1.8.15 `[VERIFIED: go.mod]` | Server-side WS framing; `conn.Close(code, reason)` already used elsewhere (`internal/webserver/server.go:1409`, `websocket.StatusPolicyViolation`) — BUG-02's fix reuses this exact call shape with a new status/reason. | Already bumped to 1.8.15 in Phase 174 (DEP-01); no further action. |
| `github.com/charmbracelet/x/vt` | `v0.0.0-20260615092313-b57e5e6d29bb` `[VERIFIED: go.mod]` | Headless VT emulator already used for Hub-card styled-tail previews (`engine.go:761` `GetSessionStyledTailLines`). Exposes `IsAltScreen()`, `CursorPosition()`, `Render()` (ANSI-encoded screen snapshot) — exactly the primitives BUG-04's fix needs. | Confirmed present in the module cache at `~/go/pkg/mod/github.com/charmbracelet/x/vt@...`; public API inspected directly (see Bug 4 section). |
| `@xterm/xterm` | `^6.0.0` `[VERIFIED: frontend/package.json]` | Guest terminal rendering (`TerminalPanel.tsx`, and the dead `web/assets/terminal.js`). `term.element.style.transform = 'scale(...)'` is the existing VIEW-05 mechanism BUG-01 must extend, not replace. | No addon changes needed; `FitAddon` is already loaded-but-unused by design (guest never drives grid size — VIEW-04). |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Extending `computeGuestScale`'s existing downscale-cap math (BUG-01) | A CSS media-query-only fix (font-size media queries in `terminal.css`/inline styles) | Rejected direction: the grid is scaled via JS-computed `transform: scale()`, not native font-size — a pure-CSS media query can't coordinate with the JS-computed cell metrics without also touching `recomputeScale`/`computeGuestScale`. Any fix must go through the existing scale pipeline. |
| A new binary WS message type for "session ended" (BUG-02) | WS close code + reason string (native `CloseEvent`) | Chosen: the browser's native `WebSocket.onclose` already carries `code`/`reason` (MDN-confirmed, reason capped at 123 UTF-8 bytes) and the server already has a working `conn.Close(code, reason)` call site to copy. A new `Msg*` frame type (e.g. `0x38`) would work too but adds unneeded protocol surface for a one-shot terminal event that naturally rides the close handshake. |
| Live per-hub VT emulator + `Render()` snapshot on connect (BUG-04) | Bump `DefaultScrollbackBytes` (256 KiB → several MiB) | The project owner's own issue-#119 analysis (verified against code) explicitly calls the byte-bump a "cheap interim mitigation, not a real fix" — it only narrows the reproduction window and does not fix idle alt-screen apps past the new limit. Not recommended as the phase's actual fix; may be worth a 1-line safety-net alongside the real fix, planner's discretion. |

**Installation:** None — no `npm install` / `go get` needed. All fixes are code changes to existing files.

## Package Legitimacy Audit

**Not applicable.** This phase installs no new external packages. All four fixes reuse libraries already present in `go.mod` / `frontend/package.json`, verified above via `go.mod`/`package.json` inspection (not `npm view`/`pip index`, since no new package names are being introduced).

## Architecture Patterns

### System Architecture Diagram — how a guest reaches a terminal, and where each bug sits

```
                 ┌─────────────────────────────────────────────────────────┐
                 │  Phone / browser opens share link  GET /sessions/{id}   │
                 └───────────────────────────┬─────────────────────────────┘
                                              │
                            internal/webserver/server.go:1296
                            handleTerminalPage() — UNCONDITIONAL 302
                                              │
                                              ▼
                 ┌─────────────────────────────────────────────────────────┐
                 │  GET /app/?session={id}&cap={token}   (React SPA)       │
                 │  frontend/src/components/Hub/WebShareSessionView.tsx    │
                 └───────────────────────────┬─────────────────────────────┘
                                              │ passes wsURL, apiBaseURL
                                              ▼
                 ┌─────────────────────────────────────────────────────────┐
                 │  frontend/src/components/TerminalPanel.tsx              │
                 │   isGuest = remote || !!wsURL   (line 324)              │
                 │   BUG-01 lives here: recomputeScale() (172-190) calls   │
                 │   computeGuestScale() (lib/terminalScale.ts) — grid     │
                 │   downscale-to-fit, never upscale, no floor/reflow.     │
                 │   BUG-02 (client half) lives here: onClose is a no-op   │
                 │   console.debug (line 336) — no UI signal.              │
                 └───────────────────────────┬─────────────────────────────┘
                                              │ wss://.../sessions/{id}/ws?cap=
                                              ▼
                 ┌─────────────────────────────────────────────────────────┐
                 │  internal/webserver/server.go handleSession (~1350-1605)│
                 │   1417 hub.Subscribe(sub)                                │
                 │   1469 push authoritative resize frame (VIEW-03)         │
                 │   1485 conn.Write(hub.ScrollbackSnapshot())  <-- RAW      │
                 │        BYTE REPLAY — BUG-04 residual: alt-screen enter   │
                 │        may have scrolled out of the 256 KiB ring.        │
                 │   1590-1605 write pump: case <-hub.Done(): return        │
                 │        <-- BUG-02 (server half): silent close, no reason │
                 └───────────────────────────┬─────────────────────────────┘
                                              │ same PTY, same Hub
                                              ▼
                 ┌─────────────────────────────────────────────────────────┐
                 │  internal/relay/hub.go Run() (363-380)                  │
                 │   reads PTY in a loop; io.EOF -> defer h.Shutdown() (419)│
                 │   Same Hub instance is ALSO used by:                    │
                 │    - internal/relay/server.go (desktop attach / HubModal│
                 │      "remote" proxy path) — IDENTICAL raw-replay bug    │
                 │      at server.go:311-316, IDENTICAL silent-close bug   │
                 │      at server.go:425-437 (BUG-04 + BUG-02 second site) │
                 │    - the natural-exit watcher goroutine (engine.go:495) │
                 │      that BUG-03's owner-side auto-close depends on     │
                 └───────────────────────────┬─────────────────────────────┘
                                              │ <-hub.Done()
                                              ▼
                 ┌─────────────────────────────────────────────────────────┐
                 │  internal/daemon/engine.go:495-519 exit watcher          │
                 │   sess.SetState(pty.StateStopped); onExit(id, exitCode) │
                 └───────────────────────────┬─────────────────────────────┘
                                              │ HTTP poll (500ms, app.go)
                                              ▼
                 ┌─────────────────────────────────────────────────────────┐
                 │  app.go:383-426 pollSessionStatus() (Wails Go process)  │
                 │   BUG-03 primary suspect: 300s hard deadline from       │
                 │   session CREATE time (line 386), no re-arm, silent     │
                 │   give-up — orphans the exit-watch for long-lived       │
                 │   sessions (share workflows correlate with longer       │
                 │   lifetimes).                                          │
                 │   On success: runtime.EventsEmit(a.ctx,"session:exit")  │
                 └───────────────────────────┬─────────────────────────────┘
                                              │ Wails EventsOn (desktop only)
                                              ▼
                 ┌─────────────────────────────────────────────────────────┐
                 │  frontend/src/App.tsx:711-750 offExit handler           │
                 │   exitCode===0 && autoCloseRef.current                  │
                 │     -> handleCloseTabRef.current(sessionId)  (886-920)  │
                 │        -> await ToggleWebServing(id,false) (if webEnabled)│
                 │        -> KillSession(id)                                │
                 │        -> setTabs(filter out id)                         │
                 └─────────────────────────────────────────────────────────┘
```

### Recommended file map (no new directories needed)

```
frontend/src/
├── lib/terminalScale.ts          # BUG-01: extend computeGuestScale (pure fn, already unit-tested)
├── components/TerminalPanel.tsx  # BUG-01: recomputeScale (172-190); BUG-02 client: onClose (336)
├── lib/relayClient.ts            # BUG-02 client: ws.onclose (297-300) must capture CloseEvent
├── components/WebGLRecoveryBanner.tsx  # BUG-02 UI: pattern to mirror for a "session ended" banner
└── App.tsx                       # BUG-03: session:exit handler (711-750), handleCloseTab (886-920)

app.go                            # BUG-03: pollSessionStatus (383-426), emitExitEvent (429-449)

internal/
├── relay/
│   ├── hub.go                    # Shutdown()/Run() (363-421) — shared by both WS surfaces
│   ├── scrollback.go             # 256 KiB ring (BUG-04 root cause)
│   ├── protocol.go               # frame type consts — MsgMeta (0x20) or new type for BUG-02/04
│   └── server.go                 # desktop-attach / remote-proxy WS handler (BUG-02 + BUG-04 site 2)
├── webserver/server.go           # web-guest WS handler (BUG-01 redirect@1296, BUG-02+04 site 1)
└── daemon/engine.go               # GetSessionStyledTailLines (761) — xvt usage precedent for BUG-04

web/assets/terminal.js            # DEAD CODE (unreachable since Phase 159) — see BUG-01 note
web/terminal.html                 # DEAD CODE (embedded, no route serves it anymore)
```

### Pattern: server-authoritative resize before replay (VIEW-03) — reuse for BUG-04

Both WS handlers already follow a "push authoritative state before replaying history" pattern for
terminal *size*; BUG-04's fix extends the same pattern to terminal *screen contents*.

```go
// Source: internal/webserver/server.go:1466-1489 (also internal/relay/server.go:294-316, identical)
// VIEW-03: push authoritative host grid BEFORE scrollback replay so that
// replayed raw bytes land in a correctly-sized terminal grid. Direct conn.Write
// (not sub.Msgs) guarantees ordering before any queued live output.
if c, r := hub.Cols(), hub.Rows(); c > 0 && r > 0 {
    if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeResizeFrame(uint16(c), uint16(r))); err != nil {
        return
    }
}
...
// Replay scrollback snapshot to bring the client up to date.
if snapshot := hub.ScrollbackSnapshot(); len(snapshot) > 0 {
    if err := conn.Write(ctx, websocket.MessageBinary, snapshot); err != nil {
        return
    }
}
```

The recommended BUG-04 fix inserts a **VT-emulator-derived screen snapshot** between these two
steps (or replaces the raw `ScrollbackSnapshot()` write), using the same emulator already proven
in production for Hub-card previews:

```go
// Source: internal/daemon/engine.go:761-793 (GetSessionStyledTailLines) — existing precedent.
// Confirms: (1) MsgOutput framing bytes must be stripped before feeding the emulator,
// (2) a queryStripPattern regex removes terminal-query/in-band-resize sequences that would
// otherwise block emu.Write (Issue #96 — already solved here), (3) xvt.NewEmulator(cols, rows)
// is cheap enough to construct per-request.
emu := xvt.NewEmulator(cols, emuRows)
clean := queryStripPattern.ReplaceAll(stripped, nil)
emu.Write(clean)
// New for BUG-04 (not yet in the codebase — confirmed available on Emulator, checked directly
// against the vendored module source at
// ~/go/pkg/mod/github.com/charmbracelet/x/vt@v0.0.0-20260615092313-b57e5e6d29bb/emulator.go):
//   emu.IsAltScreen() bool         // emulator.go:490
//   emu.CursorPosition() uv.Position  // emulator.go:211
//   emu.Render() string            // emulator.go:140 — ANSI-encoded screen snapshot w/ styles+links
```

### Anti-Patterns to Avoid

- **Fixing BUG-01 in `web/assets/terminal.js` only.** That file is unreachable in production
  (`internal/webserver/server.go:1296-1304` always redirects `/sessions/{id}` to `/app/`). A fix
  there would pass any test that hits the raw HTML file directly but do nothing for real mobile
  guests. Fix `frontend/src/lib/terminalScale.ts` / `TerminalPanel.tsx` first; treat any
  `terminal.js` change as optional parity cleanup (or flag for removal), not the primary fix.
- **Adding a client-side reconnect loop as a side effect of BUG-02.** Neither
  `web/assets/terminal.js` nor `frontend/src/lib/relayClient.ts` reconnects after any close today
  (confirmed: no `setTimeout`/retry call sites near either `connect()`/`ws.onclose`). BUG-02 only
  asks for a *notice*, not auto-reconnect. Adding reconnect logic is a larger, separate feature —
  don't fold it into this phase's scope without an explicit CONTEXT.md decision.
  Explicitly listed as a note in issue #125, not a requirement.
  Note: unconditionally reconnecting after a same-code "session ended" close would also be wrong
  behavior — the session is genuinely gone.
- **Assuming `disableFunnelForSession` is the BUG-03 culprit.** It's tempting given Funnel's own
  teardown chokepoint pattern, but `internal/daemon/api.go:1975-2009` shows it's a fast, lock-only
  no-op for any session that was never in the `funnelSessions` map (i.e., every plain-web-share,
  non-Funnel session — the case in the #126 repro). Don't spend a plan task "fixing" this path
  without first confirming the session was Funnel-active.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Reconstructing a terminal screen from a byte log (BUG-04) | A custom "replay the tail N KB and hope" heuristic, or a bigger ring buffer as the *real* fix | `charmbracelet/x/vt`'s `Emulator` (already vendored, already used for Hub-card previews at `engine.go:761`) | It already solves the two hard problems this bug needs solved: (1) stripping terminal-query bytes that would otherwise deadlock a naive VT parser (Issue #96, solved via `queryStripPattern`), and (2) tracking alt-screen-vs-main-screen mode + cursor position, which a raw byte tail structurally cannot recover once truncated. |
| A "session ended" signal channel | A new polling endpoint, a new SSE stream, or a custom binary frame type | The WS connection's own `CloseEvent.code`/`.reason` (native browser API) | The connection is *already* closing at the exact moment the signal needs to fire — piggybacking on the close handshake needs zero new plumbing on either side, just populating fields that are currently left at their zero values. |
| Mobile-appropriate terminal scaling | A second xterm.js instance, a canvas-based custom renderer, or reflowing PTY columns from the client | Extend the existing `computeGuestScale` pure function (already unit-tested in `terminalScale.test.ts`) with a floor + reflow/scroll fallback | The scale math is already correct and tested for the ≥~50% case; the gap is purely "what happens below the readability floor," which is an additive change to one pure function, not a rewrite. |

**Key insight:** every one of the four bugs has an existing, working, *partially correct*
mechanism already in the codebase (scale-to-fit, WS close, exit-watch poll, VT emulator for
previews). None of the four fixes should be architected as new subsystems — they are extensions
or corrections of code that is already 80-95% right.

## Common Pitfalls

### Pitfall 1: Fixing the dead `web/assets/terminal.js` surface for BUG-01
**What goes wrong:** A plan that greps the GitHub issue text and finds `terminal.js:983` (VIEW-05)
edits that file, ships it, and the mobile bug persists because real traffic never reaches it.
**Why it happens:** The issue was filed citing the pre-Phase-159 architecture; Phase 159
(`WEBCHAT-01`) redirected `/sessions/{id}` to `/app/` afterward and nobody updated the issue text
or `TESTING.md` Category P.
**How to avoid:** Confirm the redirect at `internal/webserver/server.go:1296-1304` before writing
any BUG-01 task; target `frontend/src/lib/terminalScale.ts` + `TerminalPanel.tsx`.
**Warning signs:** A fix that only touches `web/` and passes `node --check` (the existing gate for
that vendored asset per Category P) but has no vitest coverage.

### Pitfall 2: Sending the close reason via `sub.Msgs` instead of `conn.Close()` (BUG-02)
**What goes wrong:** `sub.Msgs` is a buffered channel drained by the write pump; if you push a
"session ended" frame onto it and then let the normal `hub.Done()` case do `conn.CloseNow()`
(abrupt), you risk the frame never being flushed (channel send races the pump's own exit) and you
get no `code`/`reason` on the client's `CloseEvent` either way.
**Why it happens:** `sub.Msgs` is the right channel for *live* broadcast frames but the write pump
already special-cases `hub.Done()` as an unconditional `return` (`internal/webserver/server.go:1599`,
`internal/relay/server.go:433`) — pushing to `sub.Msgs` from inside that case doesn't get read.
**How to avoid:** Call `conn.Close(<code>, "<reason>")` directly in the `case <-hub.Done():` branch,
mirroring the existing `conn.Close(websocket.StatusPolicyViolation, "too slow")` pattern already in
this file (`server.go:1409`). The deferred `conn.CloseNow()` (line 1431) becomes a harmless no-op
after an explicit `Close()`.
**Warning signs:** A test that asserts the frame was *sent* (channel receive) but not that the
client actually observed `CloseEvent.code !== 1005` (no status received) on connection teardown.

### Pitfall 3: Alt-screen fix that doesn't handle idle sessions (BUG-04)
**What goes wrong:** A fix that only serializes the emulator's screen "at connect time" but the
emulator itself was never fed the PTY stream continuously (i.e., only reconstructed from
`ScrollbackSnapshot()` on demand) reproduces the exact same truncation bug one layer deeper.
**Why it happens:** The natural first instinct is "run the emulator over the replayed bytes right
before responding to a new connection" — but if the *source* bytes are still the truncated 256 KiB
ring, the emulator will still miss the alt-screen-enter sequence.
**How to avoid:** Per the issue-#119 analysis (verified against code): maintain a **live**,
per-hub VT emulator that consumes every PTY frame continuously (hook into `Hub.broadcast`/`Run()`,
not `ScrollbackSnapshot()`), so its `IsAltScreen()`/cursor/grid state is always correct regardless
of ring truncation. Only serialize *that* live emulator's `Render()` output on connect.
**Warning signs:** The fix still reads `hub.ScrollbackSnapshot()` anywhere in the new code path.

### Pitfall 4: Treating BUG-03 as proven before a live repro
**What goes wrong:** Committing to the "5-minute poll deadline" fix (`app.go:386`) without first
confirming it's the actual mechanism risks shipping a fix for the wrong bug (the tab still won't
close if the real cause is, e.g., a stale `sessionExits`/`webEnabled` React-state closure).
**Why it happens:** Static analysis found no share-conditional branch in the traced control flow,
but that doesn't rule out a timing race invisible to static reading.
**How to avoid:** Wave 0 of the plan should include a timed live reproduction: create a session,
immediately exit it (baseline, expect <30s, tab closes); create another, share it, wait
>5 minutes (simulate the realistic share workflow), exit it, observe whether the tab closes. If
the >5-minute case reproduces the bug and the baseline doesn't, the `app.go:386` deadline is
confirmed as the root cause. If a <5-minute shared-session exit *also* reproduces the bug, the
deadline theory is wrong and the plan needs a different diagnostic pass (add temporary logging to
`handleCloseTab`/`ToggleWebServing`/`KillSession` to find where the flow actually stalls).
**Warning signs:** A plan that writes the `app.go:386` fix directly into Wave 1 with no diagnostic
task first.

## Bug-by-bug detail

### Bug 1 — BUG-01 (#128): mobile terminal legibility

**Files/lines:**
- `frontend/src/lib/terminalScale.ts:12-21` — `computeGuestScale(containerW, containerH, gridW, gridH)`, pure function, already unit-tested (`terminalScale.test.ts`, 8 cases). `s = min(cw/gridW, ch/gridH)`, capped `s ≤ 1` (never upscale, never has a floor).
- `frontend/src/components/TerminalPanel.tsx:172-190` — `recomputeScale()`, the guest-path caller; reads `dims.css.cell` from xterm's private `_core._renderService`, calls `computeGuestScale`, applies `term.element.style.transform = 'scale(s)'`.
- `frontend/src/components/TerminalPanel.tsx:324` — `const isGuest = remote || !!wsURL` — this is exactly the flag that makes a `/app/` web-share connection (which always sets `wsURL`, see `WebShareSessionView.tsx:94`) take the downscale-only path.
- `frontend/src/App.tsx:80` — `const DEFAULT_FONT_SIZE = 14` (matches the dead `terminal.js:166 fontSize: 14`).
- `frontend/index.html:5` — `<meta name="viewport" content="width=device-width, initial-scale=1.0">` already present (no `user-scalable=no`, so pinch-zoom is not blocked — but the issue reporter says it's still unreadable without it).
- **Dead code, do not fix here:** `web/assets/terminal.js:983-1009` (`recomputeScale`, identical VIEW-05 math) and `:166` (`fontSize: 14`). Confirmed unreachable: `internal/webserver/server.go:1296-1304` (`handleTerminalPage`) issues an unconditional `http.Redirect(..., http.StatusFound)` to `/app/?session=...&cap=...` for every `GET /sessions/{id}` request; `web/terminal.html` (embedded via `web/embed.go:5`) has no other route pointing at it.

**Current behavior confirmed by reading code:** On a narrow phone viewport (e.g. ~375-414 CSS px
wide), the host's PTY grid is commonly 80+ columns at a 14px monospace font (~9px cell width),
giving a grid pixel width of ~700-750px. `computeGuestScale` computes `s ≈ 0.5-0.55` and the
*entire* grid (text, cursor, everything) is shrunk via CSS `transform: scale()`. There is no floor
on `s` and no alternate layout (horizontal scroll, higher base font, fewer visible columns) — the
grid just keeps shrinking as the viewport narrows, per the `Math.min(...)` formula with only an
upper cap.

**Recommended fix mechanism:** This is a guest-only, downscale-only design (VIEW-04/05, Phase 157)
that intentionally never lets the guest resize the host's real PTY. The fix must stay within that
constraint (never send a resize to the host) while making small viewports usable. Two
non-mutually-exclusive options, both implementable as extensions to `computeGuestScale` /
`recomputeScale` without touching the PTY-size contract:
1. **Scale floor + horizontal scroll fallback:** below some minimum readable `s` (e.g. ~0.75, in
   CSS-cell terms roughly ≥10-11px effective font), stop shrinking further and instead let
   `#terminal`'s equivalent container in `TerminalPanel.tsx`'s CSS scroll horizontally
   (`touch-action: pan-x pan-y`, remove `overflow: hidden` for that state) — the existing
   `attachTouchScroll`-equivalent vertical touch-scroll pattern in `web/assets/terminal.js:182-229`
   is a useful reference for touch-gesture handling even though that file itself is dead code.
2. **Raise the base font size on narrow viewports** (a `matchMedia`/`window.innerWidth` check
   feeding a larger `fontSize` into the `Terminal` constructor options before scale is computed) so
   the same `s` produces a larger effective pixel size.
Both keep `computeGuestScale`'s "never upscale" invariant for the desktop case untouched — the
change should be gated so it only alters behavior below a defined viewport/scale threshold.

**Testability:** `computeGuestScale` is a pure function with existing vitest coverage
(`frontend/src/lib/terminalScale.test.ts`) — any floor logic added is directly unit-testable with
new cases (e.g. "s does not go below 0.6" or "returns a reflow flag below the floor"). The visual
outcome (does 7px text actually read as "legible") is not unit-assertable and needs a live/UAT
check; per project memory ("Browser UAT via isolated component harness"), mount `TerminalPanel`
alone in a throwaway harness at a narrow viewport (dev-browser) rather than trying to drive the
full Wails-coupled app shell.

### Bug 2 — BUG-02 (#125): no disconnect notice on session end

**Files/lines (server, both WS handlers, byte-identical pattern):**
- `internal/webserver/server.go:1590-1605` — write pump; `case <-hub.Done(): return` (line 1599)
  falls through to the deferred `conn.CloseNow()` at line 1431 — no code/reason.
- `internal/relay/server.go:425-437` — identical write pump; `case <-hub.Done(): return` (line 433).
- Precedent for the fix shape already exists in the same files:
  `internal/webserver/server.go:1409` — `conn.Close(websocket.StatusPolicyViolation, "too slow")`
  (used for `CloseSlow`, a different trigger, but proves `conn.Close(code, reason)` is the
  established idiom in this codebase).
- `internal/relay/hub.go:363-380` (`Run()`, `defer h.Shutdown()` on PTY EOF) and `:413-421`
  (`Shutdown()`, `close(h.done)`) — confirms `hub.Done()` fires on **both** natural PTY exit and
  explicit `KillSession` → `manager.Remove` (`internal/daemon/engine.go:611-617`) → `hub.Shutdown()`
  (`internal/relay/manager.go:50-56`) — i.e., this single fix point covers "owner ends the
  session" AND "session exits naturally" (both cases from the BUG-02 repro steps).
- `internal/relay/protocol.go:12-22` (frame-type consts) / `:61-65` (`MetaPayload`, currently only
  `ViewerCount`) — either a new close-code convention (simplest) or an extended `MetaPayload`/new
  `Msg*` byte (0x38+ is unused; 0x30-0x3F reserved for chat/presence per the comment at line 76)
  are both viable; the WS-close-reason route needs no new frame type at all.

**Files/lines (client, both surfaces, byte-identical pattern):**
- `frontend/src/lib/relayClient.ts:297-300` — `this.ws.onclose = () => { this._clearPing();
  callbacks.onClose?.() }` — the `CloseEvent` parameter is never captured.
- `frontend/src/components/TerminalPanel.tsx:336` — `onClose: () => console.debug(...)` — the only
  consumer of `RelayClient`'s `onClose` callback, and it does nothing user-visible.
- `web/assets/terminal.js:1047-1050` — `ws.onclose = function() { connected = false;
  updateStatusBar(sessionMeta, false); }` — only flips a small status-dot color
  (`web/assets/terminal.css` `.status-dot--disconnected`); no banner/overlay. (This file is
  reachable only if a guest hits the WS endpoint directly bypassing the page redirect — low
  priority relative to the React surface, but trivial to fix in the same pass since the mechanism
  is identical.)

**Current behavior confirmed by reading code:** When the owner ends a session (kill or natural
exit), the server-side write pump for every connected WS client (web guest, in-app remote-peer
tab, desktop `HubModal` reopen) hits `hub.Done()` and does a bare `return`, triggering an abrupt,
reason-less close. Neither JS client reads `CloseEvent.code`/`.reason` even though the browser
already exposes them (confirmed via MDN: `close` event's `CloseEvent.code`/`.reason`, `reason` is
capped at 123 UTF-8 bytes and the server may set arbitrary values via
`conn.Close(code, reason)`). Result: the guest's terminal just stops updating with zero
explanation — exactly the "silent dead terminal" the issue describes.

**Recommended fix mechanism:**
1. Server: in the `case <-hub.Done():` branch of both write pumps, call
   `conn.Close(websocket.StatusNormalClosure, "session ended")` (or a small custom application
   status in the 4000-4999 private-use range, e.g. `4000`, if the planner wants a code the client
   can distinguish from an ordinary normal-closure) instead of an unadorned `return`. The deferred
   `conn.CloseNow()` becomes a no-op after the explicit close (confirmed idiom-compatible with the
   existing `CloseSlow` pattern).
2. Client (`relayClient.ts`): change `this.ws.onclose = () => {...}` to
   `this.ws.onclose = (evt) => { ...; callbacks.onClose?.(evt.code, evt.reason) }`, and extend the
   `onClose` callback type to accept `(code?: number, reason?: string) => void`.
3. Client (`TerminalPanel.tsx`): render a banner (mirror the existing
   `WebGLRecoveryBanner.tsx` pattern — `role="status" aria-live="polite"`, text label, no
   color-only signal per the user's colorblind-safety convention) reading "Session ended — the
   owner stopped this session" (or similar) instead of `console.debug`.
4. Optionally mirror steps 1-2 in `web/assets/terminal.js`'s `ws.onclose` for symmetry, low cost.

**Testability:** The server half is fully unit-testable using the existing test harness pattern in
`internal/webserver/inject_test.go:95-140` (real `WebServer` + `relay.HubManager` + TLS WS dial) —
call `manager.Get(sessionID)` then `hub.Shutdown()` mid-test and assert the WS client receives a
`websocket.CloseError` with the expected code/reason (the `coder/websocket` client exposes this via
a typed error from `conn.Read`). The client half (banner render) is a standard React/vitest
component test — mock `RelayClient`'s `onClose` callback and assert the banner text renders. No
live UAT strictly required for either half, though a live two-machine check (per TESTING.md
Category G precedent) is good confirmation for the phase's final gate.

### Bug 3 — BUG-03 (#126): shared session's tab doesn't auto-close on exit

**Files/lines (confirmed mechanism, identical for shared and unshared sessions):**
- `internal/daemon/engine.go:495-519` — natural-exit watcher goroutine: `<-hub.Done()` →
  (unless `sess.IsKilled()`) `sess.SetState(pty.StateStopped)` → `onExit(id, exitCode)`. No
  branch on web-share state anywhere in this function.
- `internal/daemon/api.go:755-759` — the `onExit` callback registered at session-create time is
  purely about the **server-side D-12 web-share grace period** (`time.AfterFunc(10*time.Second,
  ...)` → `runSessionExitCleanup`, which disables web-serving + clears grants); it is unrelated to
  the desktop app's own tab-close detection.
- `app.go:366-376` — `CreateSession` spawns `go a.pollSessionStatus(id)` **once**, only for
  sessions this exact Wails process created.
- `app.go:383-426` — `pollSessionStatus`: polls `ListSessions()` every 500ms; on `s.State ==
  "stopped"` calls `a.emitExitEvent` and returns. **Line 386:**
  `deadline := time.Now().Add(300 * time.Second) // extended to 5min for long-running agents` —
  a fixed, non-renewing deadline measured from session *creation*, not from last activity. If the
  deadline passes before the session exits, the loop silently returns (line 387's
  `for time.Now().Before(deadline)` becomes false) with **no event ever emitted** — the frontend
  never learns the session exited.
- `app.go:429-449` — `emitExitEvent` → `runtime.EventsEmit(a.ctx, "session:exit", {...})`.
- `frontend/src/App.tsx:711-750` — `EventsOn('session:exit', ...)`: for `exitCode === 0` and
  `autoCloseRef.current` true (default), calls `handleCloseTabRef.current?.(data.sessionId)`
  immediately, no branch on share state.
- `frontend/src/App.tsx:886-920` — `handleCloseTab`: `if (webEnabled[id] && !sessionExits[id])`
  awaits `ToggleWebServing(id, false)` (both errors *and* success are handled — the `try/catch`
  swallows failures) before `KillSession(id)` and `setTabs(filter)`. A genuinely **hanging**
  (never-resolving) promise here — not an error — is the only way this await could block tab
  removal indefinitely; confirmed the underlying `http.Client` (`internal/daemon/client.go:37`) has
  **no `Timeout:` set**, so a server-side hang would propagate as a client-side hang with no
  recovery.
- `internal/daemon/api.go:1361-1390` (`handleWebServe`) → unconditionally calls
  `a.disableFunnelForSession(r.Context(), id)` (line 1388) even for non-Funnel sessions — but
  `internal/daemon/api.go:1975-2009` shows this is a fast, lock-protected map-delete for any
  session not in `a.funnelSessions`, and only calls `ws.DisableFunnel(ctx)` (the one call that
  could plausibly block on a Tailscale IPC) when `remaining == 0` **and** the session was actually
  Funnel-active. **This path is not a plausible hang for a plain (non-Funnel) web-share session**
  — ruled out as the general-case cause, but worth re-checking if the live repro specifically used
  a Funnel share.

**Current behavior confirmed by reading code:** No share-conditional branch exists anywhere in this
chain — sharing does not, by itself, change how exit is detected or how the tab is closed.

**Most likely explanation (MEDIUM confidence, code-grounded, not live-confirmed):** `app.go:386`'s
fixed 300-second exit-poll window, measured from session creation with no re-arm, silently expires
for any session (shared or not) that lives longer than 5 minutes before exiting. A realistic
"create a session → share it → deliver a join code out of band → have a guest connect and verify →
exit" manual test sequence (exactly the #126 repro steps) plausibly exceeds 5 minutes, while an
immediate "create → exit" baseline does not — this produces the exact correlation the issue
reports without sharing itself being causal. This is presented as a hypothesis, not a conclusion —
see Open Questions for the required confirmation step before a plan commits to this fix.

**Recommended fix mechanism (pending live confirmation):** Either remove the fixed deadline (poll
until the session disappears from `ListSessions()` or the daemon connection drops — the `!found`
branch at `app.go:420-423` already handles session-removed-externally) or re-arm the deadline on
each successful poll tick that shows continued activity, and/or surface a user-visible signal (log
line, or a lightweight "still watching" heartbeat) if the watch does time out, so the failure mode
is at least debuggable instead of silent.

**Testability:** This is Wails-process logic (`app.go`), which is outside the vitest suite by
design and not easily unit-testable without extracting `pollSessionStatus`'s deadline logic into a
pure, injectable-clock helper first (consider this a possible Wave-0 refactor task: extract the
deadline math into a small testable function, e.g. `shouldContinuePolling(createdAt, now,
maxWindow) bool`). The live-repro/diagnostic step is mandatory regardless (see Pitfall 4) — no
regression test can be written correctly until the root cause is confirmed live.

### Bug 4 — BUG-04 (#119): host/guest session-open windowing

**Problem 1 (MDI-vs-tab) — already resolved, no code change needed this phase:**
- `frontend/src/components/Hub/SessionCard.tsx:594-604` — the primary "Open" button (row 5) is
  gated `isLocal && onOpenSession && session.state !== 'stopped'` — local-only by design.
- `frontend/src/components/Hub/SessionCard.tsx:437-461` — for remote cards, a kebab-menu item
  labeled **"Open in tab"** (prop still named `onOpenInBrowser` from its pre-168-03 history, but
  now wired to the in-app path) is available unconditionally (`isRemote`), calling
  `handleOpenRemoteSession` (per the 168-03 summary) which now opens an **in-app tab**
  (`openWebSessionTab`), not an external browser window — this is exactly FIX-03/#118, already
  shipped and unit-tested (`App.open-remote.test.tsx`, 21/21 passing per the 168-03 summary).
- `frontend/src/components/Hub/HubPanel.tsx:420-439` — clicking the card *body* (not the Open
  button/menu) for either a local or remote session opens `HubModal` (`remote={isRemote}`,
  `HubPanel.tsx:568-575`) — this is the **intentional** Phase-134 preview-modal design, confirmed
  correct by the project owner's own issue-#119 comment and independently verified here. No
  "MDI-style window bug" exists in this half — it is by-design UX, not a defect.
- **No code changes recommended for Problem 1.** If the planner wants to close the residual
  discoverability gap the owner flagged ("whoever picks up #118 explicitly includes the guest
  viewing a remote session path"), that's already done — a remote card's kebab menu has "Open in
  tab" today. Confirm this live as part of verification rather than re-implementing it.

**Problem 2 (blank/empty window on reconnect) — the real residual bug, requires a fix:**
- `internal/relay/scrollback.go:6-39` — `DefaultScrollbackBytes = 256 * 1024`; `Scrollback.Append`
  discards oldest bytes at an arbitrary byte boundary once the buffer exceeds 256 KiB.
- `internal/webserver/server.go:1484-1489` and `internal/relay/server.go:311-316` — both WS
  handlers replay `hub.ScrollbackSnapshot()` (raw bytes, whatever survived the ring) directly to a
  newly-connecting client. Reached by: web guests (`/app/` → `WebShareSessionView` → `wsURL`), the
  in-app remote-peer tab opened by FIX-03/168-03, and the desktop `HubModal` (both `remote:true`
  proxy and `remote:false` local-reattach cases) — **all four connection paths share this exact
  code**, so the bug reproduces for the host's own reopen of a local card AND for a guest.
- Confirmed root cause (verified against vendored source, not just the issue text): a full-screen
  TUI (Claude, Gemini-CLI, OpenCode) emits `ESC[?1049h` (alt-screen enter) once, then only
  differential repaints. If the ring has wrapped past that one enter sequence (any session with
  >256 KiB of output — trivial for a long-running coding-agent session), the replayed bytes are
  alt-screen *content* with no mode-switch marker, so xterm.js paints it into the wrong (main)
  buffer, and the cut point can also land mid-escape-sequence. An idle TUI (no `SIGWINCH`, guest is
  read-only and doesn't drive PTY size) never self-heals with a live repaint. Shell sessions (main
  buffer, plain scrolling text) and Codex/Antigravity (apparently force a full repaint on connect)
  don't reproduce — matching the issue's own agent-specific repro table exactly.
- Fix primitive already vendored and precedented: `github.com/charmbracelet/x/vt`
  `Emulator.IsAltScreen()` (`emulator.go:490`), `.CursorPosition()` (`:211`), `.Render()` (`:140`,
  "renders a snapshot of the terminal screen as a string with styles and links encoded as ANSI
  escape codes") — confirmed present by reading the vendored module source directly at
  `~/go/pkg/mod/github.com/charmbracelet/x/vt@v0.0.0-20260615092313-b57e5e6d29bb/emulator.go`.
  `internal/daemon/engine.go:761-793` (`GetSessionStyledTailLines`) is a working precedent for
  feeding PTY output through this emulator safely (including the `queryStripPattern` regex that
  prevents `emu.Write` from deadlocking on terminal-query bytes, per Issue #96/#100).

**Recommended fix mechanism:** Maintain a live, continuously-fed VT emulator per `Hub` (not
reconstructed on-demand from the truncated ring) so its alt-screen/cursor/grid state is always
correct. On new-connection replay, write the emulator's `Render()` output (optionally preceded by
an explicit `?1049h` mode-set when `IsAltScreen()` is true) instead of, or in addition to, the raw
`ScrollbackSnapshot()`. This must be applied at **both** WS handler sites
(`internal/webserver/server.go` and `internal/relay/server.go`) since they share the bug but are
separate call sites.

**Testability:** A backend test can subscribe to a `Hub` after feeding it enough synthetic PTY
output to wrap the 256 KiB ring past an `ESC[?1049h` sequence, then assert a fresh
`ScrollbackSnapshot()`-replacement (or the new emulator-derived preamble) correctly reconstructs
alt-screen mode + cursor position — this is a pure Go unit test, no live daemon needed (mirrors the
existing `internal/relay/scrollback_test.go` / `hub_*_test.go` style). A live two-client
confirmation (idle Claude session, reconnect, confirm no blank window) remains valuable as the
phase's UAT gate, per the issue's own suggested quick-confirm recipe (resize the host terminal to
force a `SIGWINCH` repaint; if that instantly fills the blank window, the diagnosis is right).

## Runtime State Inventory

Not applicable — this is a bug-fix phase with no rename/refactor/migration in scope. No stored
data, service config, OS-registered state, secrets, or build artifacts carry any renamed
identifier. (Skipped per the trigger condition in the research protocol: none of BUG-01..04 involve
renaming, rebranding, or moving persisted state.)

## Environment Availability

Skipped — this phase modifies existing Go/TypeScript code only; it introduces no new external
tool, service, or runtime dependency beyond what's already required to build/test this repo (Go
toolchain, Node/pnpm, and — for BUG-04's live UAT gate only — a real Tailscale tailnet + a second
device, both already required and available per the v4.2 milestone's existing live-UAT practice).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (standard library `testing`), project convention per root CLAUDE.md |
| Framework (frontend) | `vitest` (confirmed: `frontend/package.json`, existing suite ~2300+ tests per 168-03 summary) |
| Config file | `frontend/vitest.config.ts` (existing); no Go test config needed (standard `go test ./...`) |
| Quick run command (Go, package-scoped) | `go test ./internal/relay/... ./internal/webserver/... ./internal/daemon/...` |
| Quick run command (frontend, file-scoped) | `cd frontend && pnpm vitest run <TestFile>` |
| Full suite command (Go) | `go build ./... && go vet ./... && go test ./...` |
| Full suite command (frontend) | `cd frontend && pnpm exec tsc --noEmit && pnpm vitest run && pnpm exec vite build` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BUG-01 | `computeGuestScale`-derived floor/reflow keeps text above a readability threshold at narrow viewport widths | unit | `cd frontend && pnpm vitest run terminalScale` | ✅ `frontend/src/lib/terminalScale.test.ts` (extend existing cases) |
| BUG-01 | Visual legibility on an actual narrow viewport | manual/live UAT | dev-browser isolated `TerminalPanel` harness at ~375px width, or live phone against a real share link | ❌ new harness — see Category P precedent in TESTING.md |
| BUG-02 | WS close on `hub.Done()` carries a non-zero code + non-empty reason | unit (Go) | `go test ./internal/webserver/... -run TestSessionEnd` (new) | ❌ Wave 0 — extend `inject_test.go`-style harness |
| BUG-02 | Client renders a disconnect banner on `onClose(code, reason)` | unit (vitest) | `cd frontend && pnpm vitest run TerminalPanel` | ✅ `frontend/src/components/__tests__/TerminalPanel*.test.tsx` (extend) |
| BUG-03 | Exit-poll deadline logic (once extracted to a pure fn) behaves correctly for long-lived sessions | unit (Go) | `go test ./... -run TestShouldContinuePolling` (new, pending extraction) | ❌ Wave 0 — requires the extraction refactor noted in Bug 3 Testability |
| BUG-03 | Live timed repro (share workflow >5min vs immediate exit) | manual-only | N/A — live daemon + real timing | ❌ new manual checklist item (TESTING.md Section 5) |
| BUG-04 | Reconnect after ring-wrap-past-alt-screen-enter reconstructs correct buffer mode/cursor | unit (Go) | `go test ./internal/relay/... -run TestScrollbackAltScreenReplay` (new) | ❌ Wave 0 — new test file, mirrors `hub_chatsend_test.go` harness style |
| BUG-04 | Live two-client confirmation (idle Claude session, reconnect, no blank) | manual-only | N/A — live daemon, real agent CLI | ❌ new manual checklist item (TESTING.md Section 5), or extend Category P |

### Sampling Rate
- **Per task commit:** the relevant quick-run command above (Go package or vitest file scope).
- **Per wave merge:** full suite (`go build && go vet && go test ./...` AND
  `tsc --noEmit && vitest run && vite build`) — matches the pattern already used across Phase 168's
  plans (see 168-03-SUMMARY.md's verification commands).
- **Phase gate:** full suite green, plus the manual/live-UAT items above, before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `internal/webserver/session_ended_test.go` (or similar) — covers BUG-02 server-side close reason, using the `inject_test.go:95-140` harness pattern.
- [ ] `frontend/src/lib/terminalScale.test.ts` — extend with floor/reflow cases for BUG-01.
- [ ] `internal/relay/scrollback_altscreen_test.go` (or similar) — covers BUG-04's ring-wrap-past-alt-screen-enter reconstruction.
- [ ] A pure, injectable-clock extraction of `app.go:386`'s deadline logic — needed before BUG-03 can have any automated coverage at all; until then BUG-03 is manual-only.
- [ ] TESTING.md Category P (M-27/M-28) reconciliation — currently describes `web/assets/terminal.js` as "the web viewer," which is stale post-Phase-159; required by this repo's standing Regression Test Convention whenever a phase touches behavior a manual-checklist item already documents.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth-mechanism change in this phase — capability tokens (JWT-based caps) are unchanged. |
| V3 Session Management | Marginal | BUG-02's WS-close-reason string must never leak internal detail (mirrors the existing `IN-01` convention at `internal/webserver/server.go:1574-1579`: log detail server-side, send only a stable, generic reason like `"session ended"` to the client — never echo raw Go errors). |
| V4 Access Control | No | No capability/permission-check change. BUG-04's VT-emulator screen replay must respect the same read/write perms already enforced before any WS frame is written (no new code path bypasses `sub.ReadOnly`). |
| V5 Input Validation | No | No new user input surface introduced by any of the four fixes. |
| V6 Cryptography | No | No cryptographic material touched. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Information disclosure via an overly detailed WS close reason (BUG-02) | Information Disclosure | Reuse the existing `IN-01` convention (`internal/webserver/server.go:1574-1579`): a fixed, generic reason string (e.g. `"session ended"`), never the underlying Go error text. `CloseEvent.reason` is attacker-controllable to render as-is in a client-side banner — treat it as a fixed enum on the server, never raw text derived from user/session data (session *names* are user-set — do not interpolate them into the close reason without the same `dangerouslySetInnerHTML`-avoidance discipline already used elsewhere, e.g. `HubModal.tsx:67` "session.name ... rendered as React text content — no dangerouslySetInnerHTML"). |
| VT-emulator resource exhaustion via a maliciously long-lived idle session (BUG-04) | Denial of Service | The existing `GetSessionStyledTailLines` precedent already bounds the emulator to `cols × 50 rows`; any new live-per-hub emulator should have an equivalently bounded memory footprint (not unbounded scrollback) — reuse `emu.SetScrollbackSize` rather than an unbounded internal buffer. |
| Read-only capability bypass via a new relay frame type (if BUG-02 is implemented as a new `Msg*` byte instead of a WS-close reason) | Elevation of Privilege | Not applicable if the close-reason approach (recommended) is used — no new frame type, no new dispatch branch to gate. If the planner chooses the `MsgMeta`/new-frame-type alternative instead, it must NOT be dispatched through any code path that also handles `MsgInput`/`MsgSessionInject` (which are `sub.ReadOnly`-gated per `internal/webserver/server.go:1507,1568`) — a server→client-only notification frame has no read/write distinction to make, but keep it structurally separate from the client→server dispatch switch to avoid future confusion. |

## Sources

### Primary (HIGH confidence — read directly from the live `v4.2-funnel-sharing` branch)
- `web/assets/terminal.js`, `web/terminal.html`, `web/assets/terminal.css`, `web/embed.go` — raw web viewer (confirmed dead code post-Phase-159).
- `frontend/src/lib/terminalScale.ts`, `frontend/src/lib/terminalScale.test.ts`, `frontend/src/components/TerminalPanel.tsx`, `frontend/src/lib/relayClient.ts`, `frontend/src/components/Hub/WebShareSessionView.tsx`, `frontend/src/components/Hub/HubModal.tsx`, `frontend/src/components/Hub/HubPanel.tsx`, `frontend/src/components/Hub/SessionCard.tsx`, `frontend/src/components/WebGLRecoveryBanner.tsx`, `frontend/src/App.tsx`, `frontend/index.html`.
- `app.go` (root, Wails-bound `App` struct).
- `internal/relay/hub.go`, `internal/relay/server.go`, `internal/relay/scrollback.go`, `internal/relay/protocol.go`, `internal/relay/manager.go`.
- `internal/webserver/server.go`.
- `internal/daemon/engine.go`, `internal/daemon/api.go`, `internal/daemon/client.go`, `internal/daemon/types.go`.
- `internal/webserver/inject_test.go` (test-harness pattern precedent).
- `go.mod`, `frontend/package.json` (dependency version confirmation).
- `~/go/pkg/mod/github.com/charmbracelet/x/vt@v0.0.0-20260615092313-b57e5e6d29bb/emulator.go` (vendored module source — confirmed `IsAltScreen`, `CursorPosition`, `Render` public API).
- `.planning/phases/168-bug-fix-settings-polish/168-03-PLAN.md` and `168-03-SUMMARY.md` (prior-phase FIX-03 implementation record).
- `TESTING.md` (Category G, P — existing manual-checklist precedent and conventions).
- `gh issue view 119/125/126/128 --repo scottkw/agenthub` (including the `gh issue view 119 --comments` root-cause analysis authored by the project owner, independently verified line-by-line against the live branch in this research session).

### Secondary (MEDIUM confidence)
- [MDN: WebSocket close event / CloseEvent.code](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/close_event) — confirms the native `code`/`reason` fields and the 123-byte UTF-8 reason cap, used to validate the recommended BUG-02 mechanism.
- [MDN: WebSocket.close() method](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/close) — confirms server-settable code/reason semantics.

### Tertiary (LOW confidence — general web-search context, not project-specific)
- General "responsive font sizing" web-search results (dev.to/Medium articles on CSS `vw`/`rem` scaling) — used only as generic background; the actual fix recommendation is derived from the project's own existing `computeGuestScale` mechanism, not from these articles.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | BUG-03's root cause is the `app.go:386` 5-minute exit-poll deadline, not a share-specific code branch | Bug 3 detail | If wrong, the Wave-0 diagnostic task (live timed repro) will show no correlation with session lifetime, and the plan needs a fallback diagnostic pass (temporary logging around `handleCloseTab`/`ToggleWebServing`/`KillSession`) instead of a direct fix. Explicitly flagged MEDIUM confidence and gated behind a required repro step — not presented as fact. |
| A2 | A WS-close-reason (native `CloseEvent`) is sufficient for BUG-02, vs. a dedicated new frame type | Bug 2 detail, Standard Stack "Alternatives Considered" | If the planner or a future phase wants a richer disconnect payload (e.g. structured reason codes for different UI treatments — "owner ended" vs "network error" vs "server restarting"), the close-reason string alone may prove too limited and a frame-type approach would need revisiting. Low risk given issue #125's scope is a single generic notice. |
| A3 | The recommended BUG-04 fix (live per-hub VT emulator feeding `Render()`) is buildable within a single phase's scope | Bug 4 detail | This is the most architecturally involved of the four fixes (new persistent per-hub state, wired into `Hub.Run()`'s hot broadcast loop) and carries the most integration risk (must not block or slow the PTY drain loop — `Hub.broadcast`'s existing non-blocking-send + `CloseSlow` pattern must be preserved). If a full live emulator proves too invasive for one phase, the "cheap interim mitigation" (bump `DefaultScrollbackBytes`) explicitly flagged as NOT a real fix by the issue's own author is the documented fallback — but should not be presented as satisfying BUG-04 without an explicit scope-reduction decision. |

**If this table is empty:** N/A — see rows above.

## Open Questions

1. **Does BUG-03 actually correlate with session lifetime, or with something else share-specific?**
   - What we know: the client-side auto-close mechanism is code-identical for shared/unshared
     sessions; the `app.go:386` 5-minute poll deadline is a plausible, code-grounded, but unproven
     explanation for the observed correlation.
   - What's unclear: whether the original #126 repro (which doesn't state elapsed time) actually
     took >5 minutes, or whether some other timing/race condition (e.g. a stale React closure over
     `webEnabled`/`sessionExits`) is the real cause.
   - Recommendation: Wave 0 of the plan should run the timed live repro described in Pitfall 4
     before committing to a fix; do not write the `app.go:386` fix directly into Wave 1.

2. **Should BUG-02's disconnect banner distinguish "owner ended the session" from "network/relay
   error"?**
   - What we know: issue #125 asks for a generic "session ended / you've been disconnected"
     notice; the WS-close-reason mechanism recommended here can carry a distinguishing string if
     desired (e.g. `"session-ended"` vs `"server-shutdown"`), but the current codebase has no
     precedent for multiple distinct close reasons on this connection type.
   - What's unclear: whether product intent (not yet captured in a CONTEXT.md for this phase) wants
     a single generic notice or differentiated messaging.
   - Recommendation: default to a single generic notice (matches the issue's literal ask); if
     CONTEXT.md/discuss-phase surfaces a desire for differentiated messaging, the close-reason
     string is already extensible without a frame-type change.

3. **Is the live per-hub VT emulator (BUG-04) safe to run for every session unconditionally, or
   should it be lazy (constructed only when a viewer connects)?**
   - What we know: `GetSessionStyledTailLines` constructs an emulator on-demand per Hub-card-preview
     request (not persistent); a *live*, continuously-fed emulator is a new persistent-state pattern
     for this codebase.
   - What's unclear: the CPU/memory cost of feeding every PTY byte through a second consumer
     (existing scrollback ring + new live emulator) for every active session, including ones with
     zero connected viewers.
   - Recommendation: the planner should consider constructing the live emulator lazily (only for
     sessions that have ever been web-shared, or only from the first connect onward, replaying
     `ScrollbackSnapshot()` once to bootstrap it) rather than for every session unconditionally, to
     bound the resource cost. This is a design decision for the plan, not something research can
     resolve without a product-intent input.
