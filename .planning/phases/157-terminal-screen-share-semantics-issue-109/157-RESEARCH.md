# Phase 157: Terminal Screen-Share Semantics (Issue #109) - Research

**Researched:** 2026-06-27
**Domain:** PTY relay grid arbitration (Go) + xterm.js viewer CSS-scaling (web + desktop React)
**Confidence:** HIGH (all symbols read at exact file:line in this session; no external packages)

## Summary

This is an internal-codebase phase: no new dependencies, no registry packages. Every fact
below is `[VERIFIED: codebase]` from a direct read of the named file:line. The phase replaces
the existing **max-wins** PTY-size arbiter (MC-06) with **host-authority** arbitration and makes
both viewers (web `terminal.js`, desktop `TerminalPanel.tsx`) *honor* a server-pushed grid size
and *CSS-downscale* it to fit, instead of self-sizing.

The codebase is unusually well-prepared for this change. The two distinguishing facts that make
Option B clean:

1. **Origin is already a first-class field.** `Subscriber.Origin` is set to `"local"` on the
   loopback relay path (`internal/relay/server.go:264`) and `"web"` on the webserver path
   (`internal/webserver/server.go:1072`). The arbiter can read `sub.Origin` directly — **no new
   distinction needs to be invented** (this resolves the CONTEXT "how is origin known" ambiguity
   decisively in favor of the existing field).
2. **The wire already carries a server→client resize frame.** `MsgResize` (0x02) and
   `MakeResizeFrame(cols, rows uint16)` exist (`internal/relay/protocol.go:14,44`). The desktop
   client *already parses* 0x02 (`relayClient.ts:140`) but deliberately **drops** it
   (`relayClient.ts:259`). The web client ignores all non-output frames
   (`terminal.js:1006`). So VIEW-04 is "stop ignoring 0x02," not "invent a protocol."

**Primary recommendation:** Centralize the policy in `Hub.ResizeClient` (gate on
`sub.Origin == "local"`, use `min` across local-origin subscribers, freeze when none), add a
`Hub.Rows()` method and a resize-frame broadcast, push the grid before scrollback replay on both
join paths, and gate viewer "honor + scale" behavior on guest-ness (web = always guest; desktop =
guest iff `remote` or `wsURL` is set). Replace the three MC-06 tests in `hub_test.go`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| PTY grid arbitration (which `(cols,rows)` the PTY runs at) | Relay hub (`internal/relay/hub.go`) | — | Single source of truth; both server read-pumps already funnel resize here via `ResizeClient` |
| Origin tagging (local host vs web guest) | Relay server (`server.go:264`) + Webserver (`server.go:1072`) | — | Already stamped at subscribe time on `Subscriber.Origin` |
| Reject web-origin resize | Relay hub arbiter (`ResizeClient`) | Webserver call site | Centralize in hub for one testable gate; call site is belt-and-suspenders |
| Broadcast host grid to guests | Relay hub (`broadcast`) | — | Hub owns the subscriber fan-out set |
| Honor server grid (`term.resize`) | Web viewer (`terminal.js`), Desktop viewer (`relayClient.ts`/`TerminalPanel.tsx`) | — | Each viewer owns its xterm instance |
| CSS downscale-to-fit | Web CSS (`terminal.css`), Desktop CSS (`style.css`) | — | Pixel scaling is a pure presentation concern |
| Host renders natively (never scaled/disturbed) | CLI host (`attach.go`), Desktop host (`TerminalPanel.tsx` non-guest) | — | Host drives the grid; scaling it would defeat authority |

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| VIEW-01 | PTY grid tracks host (local-origin); broadcast `MakeResizeFrame(ptyCols,ptyRows)` on host resize | `ResizeClient` rewrite (§Change-Layer 1) + new resize-frame broadcast |
| VIEW-02 | Web/remote `MsgResize` ignored by arbiter; loopback host is sole driver | Origin gate in `ResizeClient` + drop at webserver call site (§Change-Layer 2) |
| VIEW-03 | On join, push host grid before scrollback replay | `relay/server.go:295` + `webserver/server.go:1134` insertion points (§Change-Layer 3); requires new `Hub.Rows()` |
| VIEW-04 | Guests honor 0x02 → `term.resize`, stop self-sizing (web + desktop) | `terminal.js:1002` dispatch + `relayClient.ts:259` un-drop, guest-gated (§Change-Layers 4 & 6) |
| VIEW-05 | Guests CSS `transform: scale(s)`, `s=min(...)`, cap `s≤1.0`, recompute on resize + every `MsgResize` | Cell metrics already read in `fitTerminal`/touch handler; CSS approach (§Change-Layer 5) |

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Core model (Option B, locked upstream in Issue #109):** Host (local origin) is the single
  source of truth for the PTY grid. Guests `term.resize()` to the host grid and CSS-**downscale**
  to fit; cap `s ≤ 1.0` (downscale-only, never upscale). Host view is never disturbed (renders
  natively at the PTY grid).
- **D-01 — Host-not-connected:** **Freeze last host size.** When no local (host) subscriber is
  driving the PTY, retain the most recent host-reported `(cols,rows)` until a host reconnects.
  (NOT a fixed 120×32 default.)
- **D-02 — Multiple local hosts:** **Smallest-among-local arbitration** — PTY grid = `min` of
  host-origin subscriber sizes (swap `max`→`min`, scoped to **local origin only**). Keeps every
  native local host pixel-pristine; larger ones get padding. Common case (single local host) is
  moot.
- **D-03 — Guest input:** **Leave existing guest input behavior exactly as-is.** Sizing-only
  phase. The one invariant added: **guests never drive PTY resize** (web-origin `MsgResize`
  ignored). `MsgInput`/inject paths untouched.

### Claude's Discretion
- Exact scale-recompute hook wiring (ResizeObserver vs. window resize listener), CSS transform
  origin, and container layout for the scaled grid — implement to the cleanest result that honors
  `s ≤ 1.0` and no host disturbance.
- Test structure/naming for the host-authority hub tests, within the TESTING.md convention.

### Deferred Ideas (OUT OF SCOPE)
- Collaborative read-write guest typing as a *new* designed feature (D-03 keeps existing behavior).
- Relaxing "host view never disturbed" so local hosts also downscale (alternative to D-02's
  smallest-among-local) — not adopted.

## Project Constraints (from CLAUDE.md + AgentHub CLAUDE.md + TESTING.md)

- **Go:** `go fmt`, `golangci-lint`, context-aware functions; `go test -race -short ./...`.
- **JS/TS:** `camelCase`, ESLint + Prettier, TS types; `cd frontend && pnpm test` (vitest).
- **Silent fallbacks forbidden** — "let it crash"; do not convert hard failures to `or {}`.
- **Chesterton's Fence** — the `max`-wins arbiter and the `// resize frames... informational`
  drop at `relayClient.ts:259` exist for a reason (Phase 139 native-host sizing + avoiding a
  resize feedback loop); articulate before removing. This phase deliberately changes both — the
  rationale is the garble invariant in CONTEXT.
- **TESTING.md Standing Rule (repo CLAUDE.md):** every new/changed test file must update
  Section 2 (Suite Manifest counts), Section 4 (Traceability Map — path column is a repo-relative
  `.go`/`.ts`/`.tsx`/`.sh` path only, test names go in Notes), and Section 5 (Manual checklist)
  if a behavior cannot be automated. Run `bash tests/check-traceability-paths.sh` before commit.

## Standard Stack

No new packages. Existing, already-installed:

| Library | Version | Purpose | Source |
|---------|---------|---------|--------|
| `@xterm/xterm` | 6.0.0 (web vendored `web/vendor/xterm/VERSION`; desktop `frontend/package.json ^6.0.0`) | Terminal renderer; `term.resize(cols,rows)` API | [VERIFIED: codebase] |
| `@xterm/addon-fit` | 0.11.0 (both surfaces) | `FitAddon.fit()` self-sizing — **to be disabled on guests** | [VERIFIED: codebase] |
| `@xterm/addon-webgl` | 0.19.0 | WebGL canvas renderer (loaded on both) — relevant to scale-blur caveat | [VERIFIED: codebase] |
| `coder/websocket` | (Go, existing) | WS framing transport | [VERIFIED: codebase] |
| `charmbracelet/x/vt` | (Go, existing) | Headless VT for scrollback tail (consumes `Hub.Cols()`) | [VERIFIED: engine.go:780] |

**Version parity:** Web and desktop both run xterm **6.0.0** — the `term.resize()` and cell-metric
APIs are identical, so a single scaling approach ports across surfaces. [VERIFIED: codebase]

## The Wire: `MsgResize` (0x02) frame format

```go
// internal/relay/protocol.go:14
MsgResize byte = 0x02 // Terminal resize (cols, rows as big-endian uint16)

// internal/relay/protocol.go:44  — server → client resize push
func MakeResizeFrame(cols, rows uint16) []byte {
    return []byte{ MsgResize, byte(cols>>8), byte(cols), byte(rows>>8), byte(rows) }
}
// Frame: [0x02, cols_hi, cols_lo, rows_hi, rows_lo]  (5 bytes, big-endian)
```

**Critical naming hazard — two different resize frames:**
- `MsgResize` **0x02** = server→client ("here is the grid you must render"). Built by Go
  `MakeResizeFrame`. This is the frame VIEW-01/03/04 push and the viewers must honor.
- `MsgResize2` **0x11** = client→server ("I want this size"). Built by Go
  `attach.MakeClientResizeFrame` (`attach.go:151`), web `makeResizeFrame` (`terminal.js:25`),
  and TS `encodeResizeFrame`/`MSG_RESIZE2` (`relayClient.ts:5`). This is what the server
  read-pumps parse (`relay/server.go:332`, `webserver/server.go:1162`) and feed to
  `ResizeClient`. **VIEW-02 = ignore web-origin 0x11.**

`attach.go:146-152` documents the split explicitly: *"Do NOT use relay.MakeResizeFrame() — it uses
MsgResize (0x02) which the server ignores for client-originated resize."* That comment is correct
today and stays correct (0x02 remains server→client only). [VERIFIED: codebase]

TS decode of 0x02 already exists and is correct:
```ts
// relayClient.ts:140
case MSG_RESIZE: { if (data.length < 5) return {type:'unknown'}
  const cols = (data[1]<<8)|data[2]; const rows = (data[3]<<8)|data[4]
  return { type:'resize', cols, rows } }
```

## Architecture Patterns

### System Architecture (data flow)

```
                 HOST (drives grid)                         GUESTS (render + scale)
   ┌───────────────────────────────┐              ┌──────────────────────────────────┐
   │ Desktop host TerminalPanel     │              │ Web terminal.js  (ALWAYS guest)   │
   │  (remote=false, no wsURL)      │              │ Desktop TerminalPanel             │
   │  FitAddon.fit → onResize       │              │  (remote=true OR wsURL set)       │
   │  → sendResize(0x11)            │              └──────────────┬───────────────────┘
   └───────────────┬───────────────┘                             │ honor 0x02 → term.resize
   CLI attach host │ 0x11                                         │ recompute CSS scale(s≤1)
   (attach.go)     │                                              ▲
                   ▼                                              │ 0x02 broadcast (VIEW-01)
        relay/server.go read-pump  ── 0x11 ──┐         ┌──────────┘  + join push (VIEW-03)
        webserver read-pump (web) ── 0x11 ──┐│         │
                                            ▼▼         │
                              ┌─────────────────────────────────────┐
                              │ Hub.ResizeClient  (THE ARBITER)      │
                              │  • ignore sub.Origin!="local" (V-02) │
                              │  • ptyGrid = min(local subs) (D-02)  │
                              │  • freeze last host size (D-01)      │
                              │  • on change: resizeFn(PTY syscall)  │
                              │      + broadcast MakeResizeFrame(0x02)│
                              └──────────────────┬──────────────────┘
                                                 ▼ resizeFn
                                          real PTY (cols,rows)
```

File-to-tier mapping is in the Responsibility Map above; the diagram traces the resize data path.

### Pattern: unlock-before-IO (existing hub discipline — MUST follow)
`ResizeClient` releases `hub.mu` *before* calling `resizeFn` (`hub.go:254`) to avoid blocking the
broadcast drain on a PTY syscall. Any new broadcast added inside/after `ResizeClient` must also
run **outside** `hub.mu` (the `broadcast`/`BroadcastMeta` helpers acquire `mu` themselves —
calling them while holding `mu` self-deadlocks). [VERIFIED: hub.go:233-260, 290-301]

### Anti-Patterns to Avoid
- **Honoring 0x02 on the host.** The desktop host and CLI host must keep rendering natively. The
  CLI host already ignores 0x02 (only handles 0x01/0x20 — `attach.go:125-134`); the desktop host
  must keep dropping it. Gate the "honor" behavior on guest-ness (§Change-Layer 6).
- **Re-gridding guests via FitAddon.** Leaving `FitAddon.fit()` active on a guest re-derives
  `(cols,rows)` from pixels → defeats the whole phase. Disable it for guests.
- **Broadcasting from inside `hub.mu`.** Self-deadlock (see pattern above).
- **Letting a guest's 0x11 reach `ResizeClient` and mutate the grid.** That is the exact MC-06
  bug; the origin gate must drop it.

## Change-Layer 1 — Hub arbitration (`internal/relay/hub.go`)

### Symbols (exact)
- `Subscriber.Origin string` — `hub.go:62` (`"local"` relay loopback / `"web"` webserver). **This
  is the origin distinction VIEW-01/02 need — already present.** [VERIFIED]
- `Subscriber.Cols/Rows int` — `hub.go:56-58` ("Read/written under hub.mu (MC-06)").
- `Hub.ptyCols/ptyRows int` — `hub.go:92-94`.
- `Hub.ResizeClient(sub, cols, rows) error` — `hub.go:234-260` (the max-wins arbiter to replace).
- `Hub.Cols() int` — `hub.go:684` (returns `ptyCols`, **fallback 220** when `≤0`).
- `Hub.Rows()` — **DOES NOT EXIST.** Only `Cols()` exists. Confirmed by grep — `hub.Rows()` in
  CONTEXT/REQUIREMENTS is aspirational; **this phase must add `Rows()`.** [VERIFIED: no match in
  `internal/`]
- `Hub.broadcast` / `BroadcastMeta` — `hub.go:290`,`306` (non-blocking fan-out templates).
- `NewHub` — `hub.go:111`.

### Current logic (max-wins, MC-06) — `hub.go:234-260`
```go
func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
    h.mu.Lock()
    sub.Cols = cols; sub.Rows = rows
    maxCols, maxRows := 0, 0
    for s := range h.subscribers {            // ALL subscribers, no origin check
        if s.Cols > maxCols { maxCols = s.Cols }
        if s.Rows > maxRows { maxRows = s.Rows }
    }
    needResize := (maxCols>0||maxRows>0) && (maxCols!=h.ptyCols||maxRows!=h.ptyRows)
    if needResize { h.ptyCols=maxCols; h.ptyRows=maxRows }
    h.mu.Unlock()
    if needResize && h.resizeFn != nil { return h.resizeFn(maxCols, maxRows) }
    return nil
}
```

### Specified change (host-authority + min-among-local + freeze + broadcast)
```go
func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
    // VIEW-02: only the local host drives the grid. Web/remote guests are ignored
    // entirely (their reported size never enters the arbiter). Record nothing for them.
    if sub.Origin != "local" {
        return nil
    }
    h.mu.Lock()
    sub.Cols = cols; sub.Rows = rows

    // D-02: smallest-among-LOCAL. Iterate only local-origin subscribers.
    minCols, minRows := 0, 0
    for s := range h.subscribers {
        if s.Origin != "local" { continue }
        if s.Cols <= 0 || s.Rows <= 0 { continue } // not-yet-reported host
        if minCols == 0 || s.Cols < minCols { minCols = s.Cols }
        if minRows == 0 || s.Rows < minRows { minRows = s.Rows }
    }

    // D-01: freeze-last-host-size. If no local subscriber currently reports a size,
    // minCols/minRows stay 0 → do NOT overwrite ptyCols/ptyRows. The last host grid persists.
    needResize := (minCols > 0 && minRows > 0) && (minCols != h.ptyCols || minRows != h.ptyRows)
    if needResize { h.ptyCols = minCols; h.ptyRows = minRows }
    pc, pr := h.ptyCols, h.ptyRows
    h.mu.Unlock() // release BEFORE resizeFn + broadcast (unlock-before-IO discipline)

    if needResize {
        // VIEW-01: tell every guest the new authoritative grid.
        h.broadcastResize(uint16(pc), uint16(pr))
        if h.resizeFn != nil { return h.resizeFn(minCols, minRows) }
    }
    return nil
}

// VIEW-01 broadcast helper — mirrors BroadcastMeta non-blocking fan-out (hub.go:306).
func (h *Hub) broadcastResize(cols, rows uint16) {
    frame := MakeResizeFrame(cols, rows)
    h.mu.Lock(); defer h.mu.Unlock()
    for sub := range h.subscribers {
        select { case sub.Msgs <- frame: default: go sub.CloseSlow() }
    }
}
```

**Add `Hub.Rows()`** mirroring `Cols()` (`hub.go:684`). Pick a fallback consistent with the
headless VT default `emuRows = 50` (`engine.go:780`):
```go
func (h *Hub) Rows() int {
    h.mu.Lock(); defer h.mu.Unlock()
    if h.ptyRows <= 0 { return 50 }   // mirrors Cols()'s 220 fallback; 50 = engine emuRows
    return h.ptyRows
}
```

### D-01 edge case to verify in a test
When the host disconnects, `Unsubscribe` (`hub.go:199`) removes it from `h.subscribers` but does
**not** touch `ptyCols/ptyRows` — so the last host grid is already frozen for free. A subsequent
web-guest 0x11 returns early (origin gate) and cannot shrink it. The only path that *re-derives*
the grid is a local host reporting a size. **Test:** host sets 200×50, host unsubscribes, a web
guest sends 80×24 → `ptyCols/ptyRows` stay 200×50, `resizeFn` not called. (Contrast the OLD
`TestHub_ResizeClientUnsubscribeDoesNotShrink` which deliberately shrank to the remaining sub.)

### MC-06 tests to REPLACE (`internal/relay/hub_test.go`)
All three construct `Subscriber{}` with **no `Origin`** (empty string), so under the new gate they
would be ignored and the tests would fail — they must be rewritten with `Origin: "local"`:
- `TestHub_ResizeMaxWinsPolicy` — `hub_test.go:354`. Replace with `TestHub_ResizeHostAuthority_MinAmongLocal`: two `Origin:"local"` subs at 220×50 and 80×24 → PTY = **80×24** (min, not max).
- `TestHub_ResizeClientNoOpWhenDimensionsUnchanged` — `hub_test.go:403`. Keep semantics but stamp `Origin:"local"`.
- `TestHub_ResizeClientUnsubscribeDoesNotShrink` — `hub_test.go:439`. Repurpose as the **D-01 freeze** test (see edge case above); its current assertion (shrink-on-unsub) now *contradicts* D-01 and must be inverted.

New tests to ADD (Claude's discretion on naming): web-origin ignored (`Origin:"web"` sub at any
size never changes `ptyCols`); broadcast-on-host-resize (a subscribed guest receives a 0x02 frame
on `Msgs` after a host resize); `Rows()` fallback.

## Change-Layer 2 — Web-origin resize rejection (`webserver/server.go`, `relay/server.go`)

### Inbound resize call sites (both feed `ResizeClient`)
- `relay/server.go:332-337` — loopback host path (`sub.Origin == "local"`). **Keep** — this is the
  driver.
- `webserver/server.go:1162-1167` — web path (`sub.Origin == "web"`). This is the guest path.

```go
// webserver/server.go:1162
case relay.MsgResize2:
    if len(payload) >= 4 {
        cols := uint16(payload[0])<<8 | uint16(payload[1])
        rows := uint16(payload[2])<<8 | uint16(payload[3])
        _ = hub.ResizeClient(sub, int(cols), int(rows)) // MC-06: max-wins arbiter
    }
```

**Recommendation (defense in depth):** The hub origin gate (Change-Layer 1) already makes this a
no-op, which is the single authoritative enforcement point (and what the Go test asserts). For
clarity and to avoid a needless lock acquisition per guest keystroke-resize, **also** drop the call
at the web call site:
```go
case relay.MsgResize2:
    // VIEW-02: web/remote guests never drive the PTY grid (host-authority).
    // The hub arbiter also rejects non-local origin, but we drop here too so a
    // guest's resize never even reaches the lock. Guests conform via 0x02 push.
```
Leave `relay/server.go:336` unchanged (it is the only legitimate driver). Update the stale
`// MC-06: max-wins arbiter` comment there to reflect host-authority.

**Verify (Pitfall):** The remote-peer desktop-guest path proxies through
`/api/relay/remote/{id}/ws` (`relayClient.ts:224`) → the *peer's* webserver. From the peer's
view that connection is `Origin:"web"` (`webserver/server.go:1072`), so the same gate applies — a
remote desktop guest cannot drive the peer's PTY either. Consistent. [VERIFIED: codebase]

## Change-Layer 3 — Join-time grid push (VIEW-03)

### Insertion points (push BEFORE scrollback replay)
- **Relay path** `relay/server.go:294-299`:
```go
// Replay scrollback snapshot to bring the client up to date.
if snapshot := hub.ScrollbackSnapshot(); len(snapshot) > 0 {
    if err := conn.Write(ctx, websocket.MessageBinary, snapshot); err != nil { return }
}
```
- **Web path** `webserver/server.go:1133-1138` — identical shape, immediately after
  `RegisterPresence`/`NotifyPresence`.

**Specified change (both sites):** insert *before* the snapshot write:
```go
// VIEW-03: push the authoritative host grid so replayed raw bytes (which carry
// grid-specific wrap points + absolute cursor moves) land in a correctly-sized grid.
if c, r := hub.Cols(), hub.Rows(); c > 0 && r > 0 {
    if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeResizeFrame(uint16(c), uint16(r))); err != nil {
        return
    }
}
```
(In `relay/server.go` the package is `relay` so call `MakeResizeFrame` unqualified; in
`webserver/server.go` it is `relay.MakeResizeFrame`.)

### Ordering hazards (call out in plan)
1. **Resize MUST precede scrollback.** If scrollback replays into a default-sized guest grid
   first, the static screen garbles before the resize lands — this is exactly the
   "static-screen garble case" VIEW-03 fixes. Order is load-bearing.
2. **Subscribe-before-snapshot anti-race is preserved.** Both paths `Subscribe` first
   (`relay/server.go:279`, `webserver/server.go:1086`) so live frames queue in `Msgs` during the
   replay. The 0x02 push is a direct `conn.Write` (same channel as the snapshot), so it is ordered
   *before* the snapshot bytes and *before* any queued `Msgs` are drained by the write pump —
   correct. Do not route the join-push through `Msgs` (that would interleave it after queued
   output). Use a direct `conn.Write`, matching the existing snapshot write.
3. **`Cols()`/`Rows()` fallback (220×50) when no host has resized yet.** Pushing 220×50 to a guest
   that then scales down is harmless and correct (it is the same width the scrollback VT used —
   `engine.go:780`). Guard `c>0 && r>0` is always true given the fallbacks; keep it anyway for
   intent.

## Change-Layer 4 — Web viewer (`web/assets/terminal.js`)

Web is **always a guest** (the web-share surface never hosts a PTY; host is always desktop or CLI).
So the guest behavior is unconditional here. xterm **6.0.0**, WebGL addon loaded.

### Current self-sizing to REMOVE / change
- `terminal.js:168-171` — `new FitAddon(); loadAddon; fit()` — initial self-fit.
- `terminal.js:1032-1037` — `term.onResize → ws.send(makeResizeFrame(...))` (sends 0x11). Now
  ignored server-side, but should be removed so guest-driven resizes don't bounce.
- `terminal.js:1039-1042` — `window.addEventListener('resize', () => fitAddon.fit())` — replace
  with **scale recompute**, not re-fit.
- `terminal.js:990-992` — `ws.onopen` sends initial `makeResizeFrame(term.cols, term.rows)` (0x11)
  — drop (the server pushes 0x02 on join instead).

### Frame dispatch to ADD — `terminal.js:995-1007`
```js
ws.onmessage = function(evt) {
    var data = new Uint8Array(evt.data);
    if (data.length === 0) return;
    var msgType = data[0];
    var payload = data.slice(1);
    if (msgType === MsgOutput) {                    // 0x01
        term.write(new TextDecoder().decode(payload));
    } else if (msgType === 0x02 && payload.length >= 4) {  // VIEW-04: MsgResize
        var cols = (payload[0] << 8) | payload[1];
        var rows = (payload[2] << 8) | payload[3];
        term.resize(cols, rows);                    // honor host grid
        recomputeScale();                            // VIEW-05
    }
};
```
Add a `MsgResize = 0x02` const beside `MsgResize2 = 0x11` (`terminal.js:4`).

### Keep FitAddon loaded but do not auto-fit
`FitAddon` is also used to read proposed dimensions; safest is to **stop calling `fit()`** rather
than remove the addon (other code paths may reference cell metrics). The grid is now set only by
`term.resize()` from 0x02.

### xterm scale soundness (CSS `transform: scale` on the xterm container)
- The container holding `.xterm` (here `#terminal`, `terminal.css:54`) can take
  `transform: scale(s)`. Cursor, selection overlay, and the WebGL/canvas layers are all children
  of `.xterm` and scale together — visually consistent. [CITED: xterm.js renders all layers inside
  `.xterm`; ASSUMED for exact 6.0.0 DOM]
- **Blur caveat:** downscaling a WebGL/canvas bitmap via CSS uses the browser's image
  downsampler → mild softening. CONTEXT explicitly accepts this ("prefer a crisp renderer").
  Acceptable; no action required beyond keeping WebGL on.
- **Pointer-coordinate caveat (document):** xterm's mouse hit-testing computes cell coordinates
  from untransformed element geometry; under `transform: scale` mouse selection can be offset.
  For a downscale-only guest viewer this is a known, documented limitation (selection on a scaled
  guest may mis-target). Not a blocker for VIEW-01..05 (sizing/garble), but note it for UAT.
- **`transform-origin: top left`** so the grid pins to the corner and scales toward bottom-right
  (no negative offset / clipping).

## Change-Layer 5 — Web scaling CSS (`web/assets/terminal.css`)

### Container model
`#terminal { flex: 1; min-height: 0; width: 100%; position: relative; }` (`terminal.css:54`).
Recommended structure: keep `#terminal` as the **measurement container** (its
`clientWidth/clientHeight` = available viewport), and apply the transform to the **xterm element**
(`term.element`, the `.xterm` node xterm injects into `#terminal`).

```css
/* VIEW-05: downscale the host grid to fit the guest viewport. */
#terminal { overflow: hidden; }            /* clip any sub-pixel overhang */
#terminal .xterm {
    transform-origin: top left;
    /* scale set imperatively via element.style.transform = scale(s) */
}
```

### Scale computation (recompute on window resize AND every 0x02)
```js
function recomputeScale() {
    var core = term._core, dims = core && core._renderService && core._renderService.dimensions;
    if (!dims || !dims.css || !dims.css.cell) return;
    var cellW = dims.css.cell.width, cellH = dims.css.cell.height;
    if (!cellW || !cellH) return;
    var gridW = term.cols * cellW;     // host grid pixel size at THIS surface's font metrics
    var gridH = term.rows * cellH;
    var container = document.getElementById('terminal');
    var cw = container.clientWidth, ch = container.clientHeight;
    var s = Math.min(cw / gridW, ch / gridH);
    if (s > 1) s = 1;                  // VIEW-05: cap — downscale-only, never upscale
    term.element.style.transform = 'scale(' + s + ')';
}
window.addEventListener('resize', recomputeScale);   // replaces fitAddon.fit()
```

**Cell metrics source** is already used elsewhere in this file
(`term._core._renderService.dimensions.css.cell.height`, `terminal.js:182-187`) and in desktop
`fitTerminal` (`TerminalPanel.tsx:33`) — same private API, confirmed working at xterm 6.0.0.
[VERIFIED: codebase]

**Why measure cells, not the canvas:** `gridW = cols*cellW` is the *layout* size before transform;
measuring `term.element.getBoundingClientRect()` after a transform would feed back the scaled size
(oscillation). Use cell-metric math.

## Change-Layer 6 — Desktop viewer parity (`frontend/src/components/TerminalPanel.tsx`)

### Is desktop ever a GUEST? YES. (resolves success-criterion-5 — parity is REQUIRED, not a gap)
`TerminalPanel` takes `remote?: boolean` (`TerminalPanel.tsx:66`) and `wsURL?: string`
(`TerminalPanel.tsx:68`). It is instantiated as a guest in:
- `HubModal.tsx:233,245` and `HubInteractiveModal.tsx:78` with `remote={remote}` →
  `/api/relay/remote/{id}/ws` daemon-proxy to a **remote peer's** session (desktop user is a guest).
- `HubPanel.tsx:556` `remote={isRemote}`.
- `WebShareSessionView.tsx:74,85` with `wsURL={wsURL}` → web-share path.
And as the **host** in `App.tsx:1711` (no `remote`, no `wsURL`) for local sessions.

So a single desktop build is host for local sessions and guest for remote/web-share sessions.
Parity must be implemented; it cannot be waived. [VERIFIED: codebase]

### Current desktop self-sizing (host behavior — keep for host, gate off for guest)
- `relayClient.ts:259` — `// resize frames from server are informational; terminal resize is
  driven client-side` — the 0x02 frame is parsed (`relayClient.ts:140`) then **dropped**. There is
  **no `onResize` callback** in `RelayClientCallbacks` (`relayClient.ts:193`).
- `fitTerminal(term)` (`TerminalPanel.tsx:30-53`) — derives cols/rows from parent pixels and calls
  `term.resize`.
- `ResizeObserver` (`TerminalPanel.tsx:700`) — re-fits on every container resize.
- `term.onResize → client.sendResize(cols,rows)` (`TerminalPanel.tsx:305-307`) — sends 0x11.
- `onOpen → client.sendResize(term.cols, term.rows)` (`TerminalPanel.tsx:295`).

### Specified parity change
1. **Add an `onResize?` callback** to `RelayClientCallbacks` (`relayClient.ts:193`) and wire it in
   the dispatch where 0x02 is currently dropped (`relayClient.ts:259`):
   ```ts
   case 'resize':
       this.callbacks.onResize?.(frame.cols, frame.rows)
       break
   ```
   (Frame is already parsed to `{type:'resize',cols,rows}` — no decode work needed.)
2. **In `TerminalPanel`, compute `isGuest = remote || !!wsURL`.** When `isGuest`:
   - Wire `onResize: (cols, rows) => { term.resize(cols, rows); recomputeScale() }`.
   - **Do not** attach the `ResizeObserver→fitTerminal` (`TerminalPanel.tsx:700`); instead observe
     the container and call `recomputeScale()`.
   - **Do not** call `client.sendResize` from `onOpen`/`term.onResize` (host-only). Guests never
     drive resize (D-03 invariant).
   - Apply `term.element.style.transform = scale(s)` with the same min/cap math as web
     (Change-Layer 5), with desktop CSS in `frontend/src/style.css` (the TerminalPanel container
     rules live there — `style.css` referenced `TerminalPanel` per grep).
   When **not** guest (host): leave today's fit/sendResize behavior entirely unchanged.
3. **Keep `fitTerminal` for the host path only.** It is correct host authority behavior.

### Desktop vs web differences to account for
- Desktop uses React refs/effects and `ResizeObserver` (`TerminalPanel.tsx:700`); web uses a plain
  `window.resize` listener (`terminal.js:1040`). Use a `ResizeObserver` on the desktop container
  for the scale recompute (cleaner than window resize inside Wails).
- Desktop loads addons via effects (WebGL/clipboard hot-swap, `TerminalPanel.tsx:74,610`); the
  scale transform is orthogonal to addon load — apply on `term.element` after `term.open`.
- Desktop's `fitTerminal` already reads `core._renderService.dimensions.css.cell` — reuse that for
  `gridW/gridH`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Resize wire frame | A new JSON resize message | Existing `MakeResizeFrame`/0x02 + TS `MSG_RESIZE` decode | Already defined, tested (`protocol_test.go:34`), and parsed on TS side |
| Origin detection | A new "is this a host?" handshake | `Subscriber.Origin` (`"local"`/`"web"`) | Already stamped at subscribe on both paths |
| Non-blocking fan-out of the 0x02 frame | A bespoke loop | Mirror `BroadcastMeta` (`hub.go:306`) | Same slow-subscriber `CloseSlow` discipline, proven |
| Cell pixel metrics | Measuring DOM after transform | `term._core._renderService.dimensions.css.cell` | Avoids transform feedback oscillation; already used in repo |

**Key insight:** Nearly every primitive Option B needs already exists in the codebase from prior
phases; this phase is mostly *re-wiring policy*, not building infrastructure.

## Common Pitfalls

### Pitfall 1: `hub.Rows()` does not exist
CONTEXT and REQUIREMENTS say `hub.Rows()`; only `Cols()` exists (`hub.go:684`). **Add `Rows()`** or
VIEW-03 won't compile. Fallback 50 (= `engine.go:780 emuRows`). [VERIFIED]

### Pitfall 2: 0x02 vs 0x11 confusion
Pushing the wrong frame type silently no-ops (server drops client 0x02; clients ignore server
0x11). Guests must *receive* 0x02 and *send* (only host) 0x11. See "The Wire" section.

### Pitfall 3: Broadcasting while holding `hub.mu`
`broadcast`/`BroadcastMeta`/`broadcastResize` acquire `mu` themselves. Call them only after
`h.mu.Unlock()` (the `ResizeClient` discipline at `hub.go:254`). Holding `mu` self-deadlocks.

### Pitfall 4: Empty `Origin` in tests
Existing MC-06 tests build `Subscriber{}` with no `Origin` → ignored by the new gate. New/updated
hub tests MUST set `Origin:"local"` (or `"web"` for the rejection test). This is the #1 reason the
three MC-06 tests must be rewritten, not merely tweaked.

### Pitfall 5: Scale recompute order vs term.resize
On a 0x02 frame, call `term.resize(cols,rows)` **then** `recomputeScale()` — the scale depends on
the new `term.cols/rows`. Recomputing first uses stale dimensions.

### Pitfall 6: Transform feedback in scale math
Measure `gridW = cols * cellW` (layout), not `getBoundingClientRect()` (post-transform). The
latter oscillates. Container size from `clientWidth/clientHeight` is pre-transform and safe.

### Pitfall 7: Disturbing the host
The host (desktop non-guest, CLI) must never honor 0x02 or scale. Gate honor/scale on
`isGuest = remote || wsURL` (desktop) / always-true (web) / never (CLI already ignores 0x02).

## Runtime State Inventory

This is a sizing/protocol-behavior phase — not a rename/migration. No stored grid sizes, no
OS-registered state, no secrets. The one persistent-ish value is `Hub.ptyCols/ptyRows`, which is
**in-memory per session** and intentionally frozen across host disconnect (D-01) — no datastore.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — grid size is in-memory `Hub.ptyCols/ptyRows` only | none |
| Live service config | None | none |
| OS-registered state | None | none |
| Secrets/env vars | None | none |
| Build artifacts | Web `web/vendor/xterm/` is vendored (not rebuilt); desktop xterm via pnpm — no version bump | none |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test -race -short ./...` (366 `*_test.go` files) |
| Framework (web/desktop) | vitest (`cd frontend && pnpm test`, 127 files); Playwright e2e (8 specs) |
| Config file | `frontend/vitest.config.*`; Go has none (stdlib testing) |
| Quick run command | `go test -race -run TestHub ./internal/relay/` |
| Full suite command | `go test -race -short ./...` + `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| VIEW-01 | host resize → ptyGrid tracks host + 0x02 broadcast to subs | Go unit | `go test -run TestHub_ResizeHostAuthority ./internal/relay/` | ❌ Wave 0 (rewrite `hub_test.go:354`) |
| VIEW-01 | min-among-local (D-02) | Go unit | same | ❌ Wave 0 |
| VIEW-01 | freeze-last-host-size (D-01) on host unsubscribe | Go unit | `go test -run TestHub_ResizeFreeze ./internal/relay/` | ❌ Wave 0 (repurpose `hub_test.go:439`) |
| VIEW-02 | web-origin 0x11 never changes ptyGrid | Go unit | `go test -run TestHub_ResizeIgnoresWebOrigin ./internal/relay/` | ❌ Wave 0 |
| VIEW-02 | webserver read-pump drops resize (call-site) | Go integration | `go test ./internal/webserver/` | ⚠️ extend `server_test.go` |
| VIEW-03 | join pushes 0x02 before scrollback (both paths) | Go integration | `go test -run Resize ./internal/relay/ ./internal/webserver/` | ❌ Wave 0 |
| VIEW-04 | TS dispatch invokes `onResize` on 0x02 (un-drop) | vitest | `cd frontend && pnpm test relayClient` | ⚠️ extend `lib/relayClient.test.ts` |
| VIEW-04/05 | guest honors resize + applies capped scale; host does not | vitest | `cd frontend && pnpm test TerminalPanel` | ❌ Wave 0 (new `TerminalPanel.scale.test.tsx`) |
| VIEW-01..05 | Issue #109 screenshot scenario (host + smaller guest, no garble) | manual UAT | dev-browser two-surface | Section 5 manual item |

### Sampling Rate
- **Per task commit:** `go test -race -run TestHub ./internal/relay/` (sub-second).
- **Per wave merge:** `go test -race -short ./internal/relay/ ./internal/webserver/` + `pnpm test`.
- **Phase gate:** full suite green + `bash tests/check-traceability-paths.sh` exits 0.

### Wave 0 Gaps
- [ ] Rewrite `internal/relay/hub_test.go` MC-06 trio (lines 354/403/439) → host-authority/min/freeze + web-ignore + broadcast tests.
- [ ] Extend `internal/webserver/server_test.go` — web-origin resize dropped + join-push present.
- [ ] Extend `frontend/src/lib/relayClient.test.ts` — 0x02 → `onResize` callback fired.
- [ ] New `frontend/src/components/__tests__/TerminalPanel.scale.test.tsx` — guest honors resize + caps scale ≤1; host path unchanged.
- [ ] TESTING.md: Section 2 count deltas; Section 4 traceability rows for every new/changed file (path-only column); Section 5 new manual item **Category P — Terminal Screen-Share Garble (Issue #109)** (two-surface host+guest, cannot be unit-automated). Run `bash tests/check-traceability-paths.sh`.

## Security Domain

`security_enforcement` is not disabled in `.planning/config.json` → enabled.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | `MakeResizeFrame` takes `uint16`; cols/rows from a 5-byte frame are bounded to 0–65535 by construction. Viewers must clamp `term.resize` to xterm minimums (≥1) — `relayClient.ts:141` already guards `len<5`. |
| V4 Access Control | yes (unchanged) | VIEW-02 *tightens* control: guests can no longer influence the shared PTY grid (removes a cross-viewer DoS-ish vector where a tiny guest forced everyone to a tiny grid). RO/RW input gating (SEC-01) is untouched (D-03). |
| V6 Cryptography | no | — |

### Known Threat Patterns
| Pattern | STRIDE | Mitigation |
|---------|--------|-----------|
| Malicious guest sends 0x11 with extreme dims to garble/DoS all viewers | Tampering / DoS | Origin gate in `ResizeClient` drops non-local resize (VIEW-02) — the host alone sets the grid |
| Crafted 0x02 with `cols=0`/`rows=0` from a compromised server path | Tampering | Viewer guards: `payload.length>=4` + clamp to xterm min (resize(0,0) is invalid in xterm) |
| Integer overflow on `int(cols)` | — | Values are `uint16`-bounded (≤65535); `int` on all target platforms ≥32-bit — safe |

No new authn/crypto surface; the change is net-positive for access control.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| max-wins shared PTY grid (MC-06) | host-authority + guest CSS-downscale (Option B) | Phase 157 | Eliminates cross-viewer garble; guests get smaller-but-correct text |
| Guests self-size via FitAddon, drive PTY via 0x11 | Guests honor server 0x02, never drive resize | Phase 157 | Single real `(cols,rows)`; wrap points identical across viewers |
| Desktop drops server 0x02 (`relayClient.ts:259`) | Desktop guest honors 0x02 via new `onResize` callback | Phase 157 | Cross-surface parity (success criterion 5) |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | xterm 6.0.0 keeps all layers (canvas/cursor/selection) inside `.xterm`, so a single `transform: scale` on it scales them coherently | Change-Layer 4/5 | If a layer is portaled outside `.xterm`, it won't scale — verify visually in UAT; fallback is to wrap a parent div |
| A2 | `Rows()` fallback of 50 is the right default (mirrors `emuRows`) | Change-Layer 1 | Wrong default only matters before any host resize; cosmetic, self-corrects on first host size |
| A3 | Pointer/selection offset under CSS transform is acceptable for a guest viewer | Change-Layer 4 | If exact mouse-select on a scaled guest is required, needs a coordinate-mapping shim (out of current scope) |
| A4 | Desktop guest container CSS lives in `frontend/src/style.css` (TerminalPanel referenced there) | Change-Layer 6 | If rules are elsewhere, the scale CSS lands in the wrong file — confirm during planning |

## Open Questions

1. **Should the host path also broadcast 0x02 to *other local hosts* (multi-local-host D-02)?**
   - Known: D-02 says local hosts render natively (never scaled); the grid = `min` of them. The
     larger local host gets padding, not scaling.
   - Unclear: does the larger local host need a 0x02 to `term.resize` down to the min (so its
     xterm grid matches the PTY), or does it keep its native larger grid and just see padding?
   - Recommendation: broadcast 0x02 to all subscribers including local hosts; a local host honoring
     `term.resize(min)` keeps byte-for-byte parity without scaling (no disturbance to pixels, only
     grid). Confirm with the planner against "host view never disturbed" — likely fine since
     resizing the grid to min is not *scaling* and the common case is a single host (moot).

2. **Web `FitAddon` removal vs keep-but-don't-fit.** Recommendation: keep loaded, stop calling
   `fit()`. Confirm no other `terminal.js` code calls `fitAddon.fit()` beyond lines 171/1041
   (grep shows only those two).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | hub/server changes + tests | ✓ (repo builds) | per go.mod | — |
| pnpm + vitest | frontend tests | ✓ | per package.json | — |
| Playwright (chromium/firefox/webkit) | optional e2e | ✓ (8 specs exist) | — | manual dev-browser UAT |
| dev-browser skill | two-surface garble UAT (Issue #109 scenario) | ✓ (skill present) | — | manual human UAT |

No missing dependencies; no external services.

## Sources

### Primary (HIGH confidence — direct codebase reads this session)
- `internal/relay/hub.go` — Subscriber/Origin (62), ptyCols/Rows (92), ResizeClient (234-260), Cols (684), broadcast templates (290,306); **no `Rows()`**.
- `internal/relay/protocol.go` — MsgResize 0x02 (14), MakeResizeFrame (44), MsgResize2 0x11 (17).
- `internal/relay/server.go` — local origin stamp (264), read-pump MsgResize2 (332-337), scrollback replay (294-299).
- `internal/webserver/server.go` — web origin stamp (1072), read-pump MsgResize2 (1162-1167), scrollback replay (1133-1138).
- `internal/relay/hub_test.go` — MC-06 trio (354/403/439).
- `internal/attach/attach.go` — client resize 0x11 (146-152), output pump ignores 0x02 (125-134).
- `internal/daemon/engine.go` — Cols() consumer + emuRows=50 (756-800).
- `web/assets/terminal.js` — FitAddon (168-171), onmessage dispatch (995-1007), onResize/window-resize (1032-1042); `web/vendor/xterm/VERSION` (6.0.0); `web/assets/terminal.css:54`; `web/terminal.html`.
- `frontend/src/lib/relayClient.ts` — MSG_RESIZE decode (140), drop comment (259), callbacks (193), constructor remote/wsURL (215-227).
- `frontend/src/components/TerminalPanel.tsx` — fitTerminal (30-53), remote/wsURL props (66-70), onOpen/onResize sendResize (295,305), ResizeObserver (700); guest call sites: HubModal (233,245), HubInteractiveModal (78), HubPanel (556), WebShareSessionView (74,85); host: App.tsx (1711).
- `TESTING.md` — Suite Manifest §2, Traceability §4 format, Manual §5, Standing Convention §6; `tests/check-traceability-paths.sh`.

### Secondary (MEDIUM)
- xterm.js transform/scale layering behavior (general knowledge of xterm DOM structure) — flagged A1/A3.

## Metadata

**Confidence breakdown:**
- Arbiter / hub change: HIGH — every symbol read at exact line; Origin field already present.
- Wire/frame format: HIGH — `protocol.go` + TS decode both read.
- Web viewer changes: HIGH — dispatch + self-sizing lines located.
- Desktop parity: HIGH on guest-detection (props + call sites verified); MEDIUM on exact CSS file (A4).
- xterm scale soundness: MEDIUM — transform layering/pointer caveats are general (A1/A3).

**Research date:** 2026-06-27
**Valid until:** 2026-07-27 (stable internal codebase; no fast-moving deps)
