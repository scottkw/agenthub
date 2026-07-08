---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
reviewed: 2026-07-08T00:00:00Z
depth: standard
files_reviewed: 16
files_reviewed_list:
  - app.go
  - app_poll_test.go
  - internal/relay/hub.go
  - internal/relay/scrollback_altscreen_test.go
  - internal/relay/server.go
  - internal/webserver/server.go
  - internal/webserver/session_ended_test.go
  - frontend/src/lib/terminalScale.ts
  - frontend/src/lib/terminalScale.test.ts
  - frontend/src/components/TerminalPanel.tsx
  - frontend/src/components/__tests__/TerminalPanel.scale.test.tsx
  - frontend/src/lib/relayClient.ts
  - frontend/src/lib/relayClient.test.ts
  - frontend/src/components/SessionEndedBanner.tsx
  - frontend/src/components/__tests__/SessionEndedBanner.test.tsx
  - frontend/src/style.css
findings:
  critical: 2
  warning: 2
  info: 2
  total: 6
status: issues_found
---

# Phase 175: Code Review Report

**Reviewed:** 2026-07-08
**Depth:** standard (with vendored-dependency tracing for the concurrency-sensitive BUG-04 emulator work, per explicit focus-area request)
**Files Reviewed:** 16
**Status:** issues_found

## Summary

BUG-01 (guest readability floor), BUG-02 (WS close reason + disconnect banner), and
BUG-03 (diagnostic logging, deadline left unchanged per the DISPROVED live-diagnosis
verdict) are all implemented cleanly. `computeGuestViewport` is correct and
well-tested at its boundaries; `SessionEndedBanner` genuinely never renders the raw
`CloseEvent.reason` into the DOM (verified against a hostile-string test); both WS
write pumps now close with a fixed, non-leaking `StatusNormalClosure` +
`"session ended"` and cannot double-close or leak the read-pump goroutine.

BUG-04's new lazy per-hub `charmbracelet/x/vt` live emulator (`internal/relay/hub.go`)
is where this review found real problems. Tracing `feedLiveEmulator`/
`EnsureLiveEmulator`/`RenderSnapshot` against the actual vendored `xvt` source
(`github.com/charmbracelet/x/vt@v0.0.0-20260615092313-b57e5e6d29bb`) surfaced two
BLOCKER-level correctness/availability issues that the plan's own unit test
(`TestScrollbackAltScreenReplay`) does not exercise, because that test never invokes
`Hub.ResizeClient` and therefore never triggers either defect. Both are detailed
below with full call-chain evidence.

## Critical Issues

### CR-01: Live VT emulator is built once at the Cols()/Rows() fallback and never resized — reconnect preamble is permanently mis-dimensioned

**File:** `internal/relay/hub.go:873-891` (`EnsureLiveEmulator`), `internal/relay/hub.go:364-407` (`ResizeClient`), `internal/relay/hub.go:939-958` (`Cols`/`Rows` fallback defaults)

**Issue:**
`EnsureLiveEmulator` constructs the emulator exactly once, sized from whatever
`h.Cols()`/`h.Rows()` return *at that instant* (`hub.go:880`):

```go
cols, rows := h.Cols(), h.Rows()
emu := xvt.NewEmulator(cols, rows)
```

`Cols()`/`Rows()` fall back to hard-coded defaults (220×50) until the host-authority
arbiter (`ResizeClient`) has recorded at least one positive local-origin size
(`hub.go:942-946`, `hub.go:954-957`). But `EnsureLiveEmulator()` is called
synchronously, *before* the WS read pump goroutine even starts, at both call sites:

- `internal/relay/server.go:279-321` — `hub.Subscribe(sub)` → …→ `hub.EnsureLiveEmulator()` — all happens before `go func() { … conn.Read(ctx) … }()` is spawned.
- `internal/webserver/server.go:1419-1492` — same ordering.

The desktop owner's own `TerminalPanel` only sends its real terminal size via a
`MsgResize2` (0x11) frame from `onOpen` *after* the WebSocket handshake completes —
which requires a full client round-trip that cannot possibly complete before the
server has already returned from `EnsureLiveEmulator()`. Since `EnsureLiveEmulator`
is idempotent and *never re-derives or re-sizes* the emulator after construction
(confirmed: `h.liveEmu.Resize(...)` is never called anywhere in `hub.go` — the
vendored `xvt.Emulator` does expose a `Resize(width, height int)` method that this
code never calls), **every hub's live emulator is permanently built at the 220×50
fallback grid**, not the host's real terminal size (typically ~80–200 × ~24–50).

Because `RenderSnapshot()` (`hub.go:907-933`) renders from this permanently
mis-sized emulator while the *same connection* has just been told the *correct*
current grid via a separate `MakeResizeFrame(hub.Cols(), hub.Rows())` write
(`server.go:297-301`, `webserver/server.go:1469-1473`), a reconnecting/late-joining
client's xterm.js is sized to the real grid but receives replayed content generated
at a different (220×50) internal width/height. For any full-screen TUI this
reintroduces exactly the symptom BUG-04 (#119) was written to fix: cursor-position
escapes and line-wrap boundaries computed for a 220-column buffer, replayed into a
guest's xterm.js sized to the host's actual (usually much narrower) grid, can
misplace or garble content.

This is not caught by `TestScrollbackAltScreenReplay`
(`internal/relay/scrollback_altscreen_test.go`) because that test never calls
`hub.ResizeClient`, so the hub's `ptyCols`/`ptyRows` never leave the same 220×50
fallback the emulator was built with — the mismatch is invisible in that harness.

**Fix:** Feed real resize events into the live emulator wherever the host-authority
grid changes, mirroring the existing unlock-before-IO discipline already used for
`broadcastResize`/`resizeFn`:

```go
// in ResizeClient, after the existing broadcastResize/resizeFn IO block:
if needResize {
    h.broadcastResize(uint16(pc), uint16(pr))
    h.emuMu.Lock()
    if h.liveEmu != nil {
        h.liveEmu.Resize(pc, pr)
    }
    h.emuMu.Unlock()
    if h.resizeFn != nil {
        return h.resizeFn(minCols, minRows)
    }
}
```

Add a regression test that calls `hub.ResizeClient` with a size different from the
220×50 fallback before asserting on `RenderSnapshot()`'s content/geometry.

---

### CR-02: `feedLiveEmulator` runs synchronously inside `Hub.Run()`'s single drain goroutine — a blocking `Emulator.Write` freezes PTY draining/broadcast for the entire session, contradicting the code's own safety claim

**File:** `internal/relay/hub.go:435-456` (`Run`), `internal/relay/hub.go:458-470` (`feedLiveEmulator`), `internal/relay/hub.go:107-120` (doc comment)

**Issue:** The `Hub` struct doc comment asserts:

> `emuMu` is a SEPARATE lock from `mu` (T-175-04-02): a slow/stuck emulator write
> must never stall the PTY-drain/broadcast loop or contend with subscriber fan-out.

and `Run()`'s inline comment repeats the claim:

> guarded by its own `emuMu` (never `hub.mu`), so a slow/stuck emulator write can
> never stall this drain loop.

This is inaccurate. `feedLiveEmulator` is called **synchronously, in-line, in the
same goroutine as the PTY read loop** (`hub.go:449`, inside `Run()`'s `for` loop —
there is no `go feedLiveEmulator(...)`). The separate `emuMu` lock only prevents
*lock contention* with `Subscribe`/`Unsubscribe`/`broadcast` (which use `h.mu`); it
does nothing to prevent `Run()` itself from blocking if `h.liveEmu.Write(clean)`
blocks.

`h.liveEmu.Write` (vendored `xvt.Emulator.Write`) can genuinely block: the emulator's
query-response channel (`Emulator.pw`) is a bare `io.PipeWriter` from `io.Pipe()`
(unbuffered — a `Write` blocks until something calls `Read` on the paired
`io.PipeReader`). Nothing in this codebase ever calls `Emulator.Read()` on the live
emulator (confirmed by grep — only `Write`, `Render`, `IsAltScreen` are called), so
*any* PTY byte sequence that triggers one of the vendored emulator's response
writers (DA1/DA2, DSR/CPR/DECXCPR, DECRQM, OSC 10/11/12 "?" queries) will hang
forever unless it was already stripped by `liveEmuQueryStripPattern` before
reaching `Write`.

The current regex (hand-duplicated from `internal/daemon/engine.go`'s
`queryStripPattern`, `hub.go:151-160`) does cover every query-response code path
present in the pinned `xvt` version as of this review — but this is a *fragile*
guarantee for two reasons this codebase has already been burned by once:

1. This exact class of bug (an unstripped query sequence deadlocking a bare
   `io.Pipe`-backed emulator `Write`) is the documented root cause of issue
   **#96/#100**, which is why `queryStripPattern` exists in `engine.go` at all.
2. `GetSessionStyledTailLines` (the precedent this code explicitly mirrors,
   `engine.go:773-802`) only risks blocking a single **on-demand RPC-serving
   goroutine** if the strip pattern ever misses something. `feedLiveEmulator` is
   now on the **hot path of the one-and-only per-hub `Run()` goroutine** — a hang
   there stalls PTY draining, scrollback appends, *and* broadcast to every
   currently-connected subscriber (host included), for the life of the process (no
   timeout, no recovery). The blast radius of the same fragile mitigation is
   categorically larger than the pattern it was copied from.

A future `xvt` version bump that adds a new query-response CSI/OSC/DCS handler (or a
TUI that emits one of the not-yet-covered families such as DECRQSS, XTGETTCAP, OSC 4
palette query) would silently reintroduce a full-session hang with no compile-time
or test-time signal, because the doc comments assert this cannot happen.

**Fix:** Either (a) run `feedLiveEmulator` in its own goroutine per hub, fed via a
bounded channel, so a stuck `Write` cannot block `Run()`'s PTY read loop; or (b) add
a dedicated goroutine that drains `h.liveEmu.Read()` into `io.Discard` for the
hub's lifetime as defense-in-depth against any strip-pattern gap; and correct the
doc comments so they describe the actual guarantee (no lock contention with
subscriber fan-out) rather than an absolute stall-immunity claim that is not true
today.

## Warnings

### WR-01: `RenderSnapshot()`'s emulator-derived preamble geometry is untested against a real resize sequence

**File:** `internal/relay/scrollback_altscreen_test.go:40-116`

**Issue:** `TestScrollbackAltScreenReplay` is the only automated coverage of
`RenderSnapshot()`'s content-reconstruction behavior, and it never calls
`hub.ResizeClient`, so it cannot detect CR-01 (the emulator is built and stays at
the 220×50 fallback throughout the test, matching `hub.Cols()`/`hub.Rows()`
coincidentally). The `go test -race` run cited in the 175-04 SUMMARY as proof of
"no regressions" therefore provides no signal on the geometry-mismatch defect.

**Fix:** Add a test that calls `hub.ResizeClient(localSub, 100, 30)` (or any size
different from the 220×50 fallback) before subscribing a second client and
asserting on `RenderSnapshot()`'s output geometry — this will fail today per CR-01
and should be the regression guard for its fix.

### WR-02: `TerminalPanel`'s guest disconnect banner fires for the RelayClient's `onClose`, including cleanup-time `client.close()` calls issued after the component starts unmounting

**File:** `frontend/src/components/TerminalPanel.tsx:352-374` (RelayClient construction), `frontend/src/components/TerminalPanel.tsx:385-458` (mount-effect cleanup)

**Issue:** The mount-effect cleanup calls `client.close()` synchronously
(`TerminalPanel.tsx:388`), which calls `this.ws.close()` in `relayClient.ts`. Per
the WebSocket spec, `onclose` fires *asynchronously* after the close handshake
completes — after React has already run the rest of the cleanup (terminal
disposal, ref nulling, etc.) and the component may already be unmounted. When that
deferred `onclose` fires, `onClose: (code, reason) => { … setSessionEnded(...) }`
(`TerminalPanel.tsx:360-363`) still runs against the closure's `setSessionEnded`
from a component that is no longer mounted. React 18 silently no-ops state updates
on unmounted function components (no crash, no console warning), so this is not
user-visible today, but it is a latent footgun: any future change that moves
`sessionEnded` state to a ref-backed side effect (e.g., an analytics ping, a
`console.debug` call site added later) would fire post-unmount without anyone
noticing the ordering hazard.

**Fix:** Guard the `onClose` handler with a mounted-ref check (a common pattern
already used elsewhere for async callbacks in this codebase), or explicitly no-op
`callbacks.onClose` from within the cleanup function before calling `client.close()`.

## Info

### IN-01: Doc-comment/behavior mismatch compounds CR-02's risk for future maintainers

**File:** `internal/relay/hub.go:114-117`, `internal/relay/hub.go:446-449`

**Issue:** Already covered in detail under CR-02, but called out separately here
because it is a maintainability hazard independent of whether CR-02 is fixed via
option (a) or (b): the comments are written as an unconditional safety guarantee
("can never stall … loop") rather than a scoped one ("never contends for `hub.mu`
with subscriber operations"). A future reviewer trusting this comment at face value
would not think to add a timeout or drain goroutine when extending this code.

**Fix:** See CR-02's fix — reword the comments to describe only the lock-contention
guarantee that actually holds.

### IN-02: `RemoteViewerCount`/`ResizeClient` local-origin loop duplicates min-among-local logic already present elsewhere

**File:** `internal/relay/hub.go:375-387`

**Issue:** Not a defect, just noted in passing while tracing `ResizeClient` for
CR-01: the min-among-local iteration is `O(subscribers)` per resize event, which is
fine functionally (performance is out of scope for this review) but worth flagging
if `ResizeClient` is touched again for the CR-01 fix — the new `h.liveEmu.Resize`
call should go in the same already-unlocked `needResize` block to avoid adding a
third separate `h.mu`/`h.emuMu` round-trip.

**Fix:** No action required beyond folding the CR-01 fix into the existing
`needResize` block as shown in that finding's fix snippet.

---

_Reviewed: 2026-07-08_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
