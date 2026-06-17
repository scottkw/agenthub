---
phase: 134-modal-interaction
reviewed: 2026-06-17T00:00:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - frontend/src/App.tsx
  - frontend/src/components/Hub/HubBriefingModal.tsx
  - frontend/src/components/Hub/HubInteractiveModal.tsx
  - frontend/src/components/Hub/HubModal.tsx
  - frontend/src/components/Hub/HubPanel.tsx
  - frontend/src/components/Hub/SessionCard.tsx
  - frontend/src/components/Hub/SessionCardGrid.tsx
  - frontend/src/components/TerminalPanel.tsx
  - frontend/src/lib/relayClient.ts
  - frontend/src/style.css
  - internal/daemon/relay_remote_files.go
  - internal/daemon/remote_ws_proxy.go
  - internal/relay/server.go
findings:
  critical: 0
  warning: 4
  info: 3
  total: 7
status: issues_found
---

# Phase 134: Code Review Report

**Reviewed:** 2026-06-17
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

This is a re-review of Phase 134 after Plans 134-06/07/08 were written to fix the
three blockers (CR-01/CR-02/CR-03) and seven warnings from the prior review.

**The three prior blockers are FIXED and verified correct/complete:**

- **CR-01 (interactive modal attached the local relay with a remote id):** Resolved.
  `internal/daemon/remote_ws_proxy.go` adds a cap-gated reverse proxy at
  `GET /api/relay/remote/{sessionID}/ws`, mounted on the relay loopback surface
  (`relay_remote_files.go:74`). `RelayClient` (relayClient.ts:87-90) builds the
  proxy path when `opts.remote` is set; `TerminalPanel` threads the `remote` prop
  into its `RelayClient` (TerminalPanel.tsx:282-288); `HubInteractiveModal` →
  `HubModal` → `HubPanel` all compute and thread `isRemote`
  (HubPanel.tsx:484-493). The proxy dials the peer's already-cap-gated
  `wss://<baseURL>/sessions/{sid}/ws?cap=T` and copies opaque frames. Origin
  injection (Pitfall 1) and the request-context copy loop (Pitfall 4) are both
  handled, with passing integration tests (`remote_ws_proxy_test.go`
  WS-PROXY-01..06).

- **CR-02 (briefing tail/send used the local daemon for remote):** Resolved.
  `HubBriefingModal` (lines 82-130) reads the remote tail from a short-lived
  proxied `RelayClient` scrollback snapshot instead of `GetSessionTailLines`
  (which is local-only), and the send path passes `{ remote: true }`. Behavioral
  tests (`HubBriefingModal.test.tsx` TAIL-01a/b/c) confirm remote renders tail
  lines from the WS snapshot and never calls `GetSessionTailLines`.

- **CR-03 (briefing send WS leak + late-onOpen delivery + unmount leak):**
  Resolved for the **send** path. `clientRef` + `settled` flag + `clearTimeout`
  on the happy path + a `useEffect(() => () => clientRef.current?.close(), [])`
  unmount cleanup are all present (HubBriefingModal.tsx:139, 148-191). Behavioral
  tests CR-03-01a/b/c assert open→sendInput→close ordering, zero-sendInput on
  timeout, and the settled-guard suppression of a late onOpen. (But see WR-01
  below: the *tail* path reintroduces the same leak class CR-03 fixed.)

**Cap-gating verification (requested):** The cap token never enters any
client-visible URL. The webview opens `ws://127.0.0.1:{port}/api/relay/remote/{id}/ws`
with no cap (relayClient.ts:88). The handler looks the cap up server-side via
`a.remoteCaps.Get(sid)` (remote_ws_proxy.go:54) and places it only on the
**upstream** dial URL (`u.RawQuery = "cap=" + url.QueryEscape(capToken)`,
line 82). The scheme swap to `wss` is safe because `RemoteCapStore.Put` enforces
an `https://` baseURL with a non-empty host (remote_caps.go:76-81). Inbound Origin
is enforced at `websocket.Accept` via the shared loopback allowlist
(`relay.LoopbackOriginPatterns`), and the cross-site-Origin rejection test passes
(WS-PROXY-05). Dial-error close reasons are fixed literals; the cap is never
surfaced in a reason or log. This is correctly gated.

**WR fixes verified:** WR-01 (cancel callback resets `pendingModalSessionId`),
WR-02 (App guards `onRequestRemoteCap` against an in-flight modal), WR-03
(`terminalTheme` now required, unsafe `{} as ITheme` cast removed), WR-04 (real
`fontSize`/`onFontSizeChange` threaded), and WR-07 (real behavioral tests added)
are all in the current code. WR-05/WR-06 are intentionally deferred to Phase 135
(a11y) and documented in-code; not flagged here per scope.

**Remaining issues are NEW defects introduced by the fixes (all WARNING/INFO)** —
the most important is that the CR-03 unmount-leak fix was applied to the send path
but NOT the new remote-tail path, which opens a `RelayClient` (with a 30s ping
interval) that leaks if the modal unmounts during the tail window.

## Warnings

### WR-01: Remote tail `RelayClient` leaks on unmount — CR-03's own fix not applied to the tail path

**File:** `frontend/src/components/Hub/HubBriefingModal.tsx:82-130, 139`

**Issue:** CR-03 added `clientRef` + an unmount cleanup
(`useEffect(() => () => { clientRef.current?.close() }, [])`, line 139) to tear
down the **send** client. The new CR-02 remote-tail path opens a *separate*
`RelayClient` stored only in the local `tailClient` variable (line 87/100), which
`clientRef` never references. If the user dismisses the modal (Escape / Close /
click-outside) during the tail window — the 3s timeout or the 500ms
post-onOpen collection — the component unmounts but `tailClient` is never closed.
The underlying `WebSocket` and, once `onOpen` has fired, its 30s ping interval
(relayClient.ts:97-101) keep running detached. This is the exact leak class CR-03
was written to eliminate, reintroduced on the tail path. It is reachable on every
remote briefing the user opens-then-closes quickly.

**Fix:** Track the tail client in a ref and close it on unmount, or return the
close from the effect so React tears it down:
```tsx
useEffect(() => {
  if (!remote) {
    GetSessionTailLines(session.id, 20).then(setTailLines).catch(() => setTailLines([]))
    return
  }
  const chunks: Uint8Array[] = []
  let resolved = false
  const tailClient = new RelayClient(relayPort, session.id, { /* ... */ }, { remote: true })
  const finish = () => { if (resolved) return; resolved = true; tailClient.close(); setTailLines(extractTailLines(chunks, 20)) }
  const timeoutId = setTimeout(finish, 3000)
  // ...
  return () => { clearTimeout(timeoutId); tailClient.close() }  // unmount cleanup (mirrors CR-03 send fix)
}, [session.id, relayPort, remote])
```

### WR-02: `onRequestRemoteCap` WR-02 guard silently strands `pendingModalSessionId` with no user feedback

**File:** `frontend/src/App.tsx:1385-1391`, `frontend/src/components/Hub/HubPanel.tsx:353-363, 391-394`

**Issue:** The prior-review WR-02 fix added `if (joinModalForSession) return` to
App's `onRequestRemoteCap` so an in-flight file-browse modal is not overwritten.
But `HubPanel.handleCardClick` has *already* set `pendingModalSessionId` and
`pendingSourceRectRef` (HubPanel.tsx:357-359) before calling
`onRequestRemoteCap`. When App early-returns, no join modal opens, so the
`onClose` path that fires `capCancelledRef.current?.()` (App.tsx:1622-1626) never
runs. Result: `pendingModalSessionId` stays set indefinitely with zero user-visible
feedback — the card click appears to do nothing, and the stale pending id is dead
state. The WR-01 cancel callback only resets on a modal *close*, which cannot
happen here because no modal was ever opened.

**Fix:** Return a boolean (or accept a rejection) from `onRequestRemoteCap` so
HubPanel can reset its pending state when App declines, and surface a transient
toast ("Finish the current join-code first"). Minimal:
```tsx
// App: signal decline
onRequestRemoteCap={(s) => {
  if (joinModalForSession) { capCancelledRef.current?.(); return }  // reset HubPanel + no-op
  setJoinModalForSession({ ...s, intent: 'hub-modal' })
}}
```
(Calling the cancel ref on decline at least clears the stranded pending id.)

### WR-03: Remote-send read-only cap drops input silently behind an enabled Send button

**File:** `frontend/src/components/Hub/HubBriefingModal.tsx:255-266, 143-197`

**Issue:** For a remote session whose deposited cap grants read-only access, the
proxied send WS still connects (the cap is valid), `onOpen` fires, `sendInput`
runs, and the modal calls `onClose()` reporting success — but the peer silently
discards the input because the cap does not grant PTY write (relay
`handleSession` discards `MsgInput` for read-only subscribers, server.go:269).
The user sees "Sending…" → modal closes → no error, but nothing was delivered.
The in-code NOTE (lines 255-258) acknowledges this and defers a read-only
indicator to Phase 135. Per the project's cross-surface-parity and colorblind
constraints this is a real correctness/feedback gap, but it is explicitly
deferred and is not a security/data-loss issue — WARNING, not BLOCKER.

**Fix (Phase 135 or sooner):** Detect write-denied on the remote send path (e.g.
peer-side close code or a no-ack timeout) and render the existing
`hub-modal__error-banner` ("This session is read-only — response not sent")
instead of closing on success. A non-color affordance satisfies the colorblind
constraint.

### WR-04: Remote tail collection window can truncate a large scrollback snapshot

**File:** `frontend/src/components/Hub/HubBriefingModal.tsx:98-123`

**Issue:** The remote tail finishes 500ms after `onOpen` (line 115) or 3s after
mount (line 98), whichever comes first, then closes. The peer replays its
scrollback snapshot as one or more binary frames on subscribe (relay
`handleSession` writes the snapshot synchronously, server.go:238-242, then live
frames flow). For a large scrollback the snapshot may be chunked by the WS
transport; if not all chunks land within the 500ms window, `extractTailLines`
operates on a partial byte stream. Because the last partial chunk can split a
UTF-8 sequence or a line mid-stream, the rendered "last 20 lines" may be wrong or
show a mojibake first line. Unlike the local `GetSessionTailLines` (which returns
exact server-computed lines), this is a best-effort heuristic with a timing race.

**Fix:** Finish on a frame-quiescence signal rather than a fixed 500ms — e.g.
reset a short (150ms) idle timer on each `onOutput` and finish when output goes
quiet, capped by the 3s hard timeout. This collects the full snapshot regardless
of size while still bounding the wait.

## Info

### IN-01: Remote tail 3s timeout handle is not cleared on the happy path

**File:** `frontend/src/components/Hub/HubBriefingModal.tsx:98, 115-120`

**Issue:** `clearTimeout(timeoutId)` is only called in the tail `onClose` callback
(line 118). On the happy path (`onOpen` → 500ms → `finish()` → `tailClient.close()`
→ `onClose` fires → clearTimeout) it is eventually cleared, but `finish()` itself
does not clear it. The `resolved` guard makes a late 3s fire a harmless no-op, so
this is cosmetic, but the cleanup is asymmetric with the send path (which clears
its timer inside the success branch).

**Fix:** Call `clearTimeout(timeoutId)` inside `finish()` so all exit paths clear
the timer uniformly.

### IN-02: `extractTailLines` allocates a full merged buffer of all accumulated chunks

**File:** `frontend/src/components/Hub/HubBriefingModal.tsx:33-45`

**Issue:** `extractTailLines` concatenates every accumulated chunk into one
`Uint8Array` before decoding, then keeps only the last 20 lines. For a session
with a large scrollback snapshot this briefly holds the entire snapshot in memory
twice (chunks array + merged buffer). Not a correctness issue and out of the v1
performance scope, but worth noting since only the tail is needed — the chunk
accumulation could be bounded.

**Fix (optional):** Cap accumulated bytes (e.g. keep only the trailing ~64KB,
enough for 20 lines) to bound memory on very large snapshots.

### IN-03: `aria-label` for the briefing modal omits local/remote origin

**File:** `frontend/src/components/Hub/HubModal.tsx:130-132`

**Issue:** Carried forward from the prior review (IN-04). The dialog `aria-label`
is `Briefing: ${session.name} needs input` with no origin marker, while sighted
users see the Local/hostname header (lines 161-166). Given remote sessions now
behave differently (proxied tail, possible read-only send drop per WR-03), the
origin distinction is more relevant for screen-reader users, not less.

**Fix:** Include origin, e.g. `Briefing: ${session.name} on ${originText} needs input`.
Reasonable to fold into the Phase 135 a11y pass.

---

_Reviewed: 2026-06-17_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
