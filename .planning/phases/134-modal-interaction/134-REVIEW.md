---
phase: 134-modal-interaction
reviewed: 2026-06-17T00:00:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - frontend/src/components/Hub/HubModal.tsx
  - frontend/src/components/Hub/HubInteractiveModal.tsx
  - frontend/src/components/Hub/HubBriefingModal.tsx
  - frontend/src/components/Hub/SessionCard.tsx
  - frontend/src/components/Hub/SessionCardGrid.tsx
  - frontend/src/components/Hub/HubPanel.tsx
  - frontend/src/App.tsx
  - frontend/src/style.css
  - frontend/src/components/Hub/HubModal.test.tsx
  - frontend/src/components/Hub/HubInteractiveModal.test.tsx
  - frontend/src/components/Hub/HubBriefingModal.test.tsx
  - frontend/src/components/Hub/SessionCard.test.tsx
  - frontend/src/components/Hub/HubPanel.test.tsx
  - frontend/src/components/__tests__/style.hub.modal.test.ts
findings:
  critical: 3
  warning: 7
  info: 4
  total: 14
status: issues_found
---

# Phase 134: Code Review Report

**Reviewed:** 2026-06-17
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Phase 134 wires a card-click → modal gesture for the Hub, routing attention sessions
to a briefing modal (terminal tail + respond-to-PTY) and non-attention sessions to an
interactive modal (live `TerminalPanel`). The animation, focus-return, Escape handling,
and card-click/Open/menu `stopPropagation` coexistence are largely correct and well
documented.

The serious problems are concentrated on the **remote-session path**. The MODAL-06 cap
gate runs the Phase 122 join-code exchange, but that exchange only deposits a cap into
the local daemon's `RemoteCapStore` for the *file-browse proxy* (`/api/files/remote/{sid}`).
Neither the interactive `TerminalPanel`'s WebSocket nor the briefing modal's
`RelayClient` / `GetSessionTailLines` has any remote-proxy equivalent — both connect to
the **local** relay (`ws://127.0.0.1:{port}/sessions/{id}/ws`) and the **local** daemon
(`e.manager.Get(id)`) using a session id that only exists on the *remote* peer. The
result: after the user completes a join-code exchange (cost: a real round-trip + token
deposit), the modal opens onto a session the local relay does not know about. That is a
broken feature shipping behind a security-looking gate, and it is the headline blocker.

Secondary concerns: `RelayClient`/WebSocket leaks in the briefing send path on timeout
and on unmount; an untrusted-text-after-abandon delivery window; a stuck
`pendingModalSessionId` when the join-code modal is dismissed; and a test suite that for
the modal components is almost entirely `?raw` source-string `toContain` assertions with
no behavioral coverage of the new interaction.

## Critical Issues

### CR-01: Interactive modal attaches the local relay using a remote session id — terminal never connects

**File:** `frontend/src/components/Hub/HubInteractiveModal.tsx:41-49`, `frontend/src/components/Hub/HubPanel.tsx:340-368,450-460`, `frontend/src/lib/relayClient.ts:86`

**Issue:** `HubInteractiveModal` mounts `TerminalPanel` with `relayPort` (the **local**
daemon relay port from `GetRelayPort()`) and `session.id`. `RelayClient` builds
`ws://127.0.0.1:${port}/sessions/${sessionId}/ws` and the local relay resolves sessions
via `e.manager.Get(id)` (internal/relay/server.go:48, engine.go:549). For a remote
(tailnet-peer) session — `adaptAllRemoteSessions` sets `hostname = peer.hostname` and a
peer-local `id` — that id does not exist on the local relay, so the WebSocket attaches to
nothing. The MODAL-06 gate (`handleCardClick`) deliberately lets remote sessions reach
this modal *after* a join-code exchange, but the join-code cap is only consumed by the
file-browse proxy (`/api/files/remote/{sid}`), not by the relay WS path. There is no
remote-relay-proxy route. Net effect: completing the cap exchange opens an interactive
terminal modal that can never connect for any remote session.

**Fix:** Either (a) restrict the interactive modal to local sessions and route remote
attention/interactive intents to the existing remote-open flow (`BrowserOpenURL` /
`handleOpenRemoteSession`), or (b) thread a remote base URL + the deposited cap into
`TerminalPanel`/`RelayClient` and add a relay proxy equivalent to
`/api/files/remote/{sid}`. Until a remote WS path exists, gate the interactive branch:
```tsx
// HubPanel.handleCardClick — do not open the interactive modal for remote sessions
const isRemote = !!session.hostname && session.hostname !== ''
if (isRemote) {
  // remote interactive terminal has no relay-proxy route yet (CR-01)
  handleOpenRemoteSession(remoteBaseURLFor(...))   // or block with a toast
  return
}
```

### CR-02: Briefing modal tail + send use the local daemon/relay for remote sessions — always empty, never delivered

**File:** `frontend/src/components/Hub/HubBriefingModal.tsx:34-38,52-62`, `internal/daemon/engine.go:548-551`

**Issue:** A remote session in attention state (`waiting`/`errored`) routes to
`HubBriefingModal`. The tail fetch calls `GetSessionTailLines(session.id, 20)`, which on
the Go side does `hub, ok := e.manager.Get(id)` and returns `[]string{}` when the id is
unknown (engine.go:550-551) — every remote session shows "No recent output available."
The Send flow constructs `new RelayClient(relayPort, session.id, …)` against the **local**
relay; the WS for an unknown id will not deliver input to the remote PTY (and, per the
relay's behavior for unknown ids, the send is silently dropped). So the briefing modal's
two core functions — show context, send a response — are both inert for remote sessions,
yet the modal presents a fully-enabled Send button implying success. Combined with CR-01,
no remote session is actually serviceable by either modal branch.

**Fix:** Gate the briefing modal to local sessions, or implement a remote tail/send proxy.
Minimal interim: detect `session.hostname !== ''` and render an explicit "Open on
{hostname}" affordance instead of the tail/respond UI, so the user is not given a
non-functional Send button.

### CR-03: Briefing send leaks the WebSocket on timeout and delivers untrusted input after the user abandons

**File:** `frontend/src/components/Hub/HubBriefingModal.tsx:47-73`

**Issue:** `handleSend` constructs a `RelayClient` inside the Promise executor and only
calls `client.close()` inside `onOpen`. Two defects:
1. **Leak on timeout:** if the socket never reaches `onOpen` within 5s, the Promise
   rejects but `client.close()` is never called — the underlying `WebSocket` (and its
   connection attempt) is never torn down. There is no `reject`-path cleanup.
2. **Late delivery of untrusted text:** if `onOpen` fires *after* the 5s reject (slow
   relay), `client.sendInput(responseText + '\n')` still runs and writes the user's text
   to the PTY even though the UI already showed "Failed to send" and the user may have
   moved on. There is no flag suppressing the post-timeout send.
3. **Unmount leak:** the component has no cleanup; if the modal unmounts while a send is
   in flight (Escape/close), the `RelayClient` and its 30s ping interval keep running.

This is the "untrusted-text-to-PTY" path called out for special attention — the bound
(`maxLength={4096}`) limits payload size but does nothing about *when* / *whether* the
text is delivered relative to the user's intent.

**Fix:** Track the client in a ref and a `settled` flag; clean up on every exit path.
```tsx
const clientRef = useRef<RelayClient | null>(null)
// in executor:
let settled = false
const timer = setTimeout(() => { if (!settled) { settled = true; clientRef.current?.close(); reject(new Error('timeout')) } }, 5000)
const client = new RelayClient(relayPort, session.id, {
  onOutput: () => {},
  onOpen: () => {
    if (settled) { client.close(); return }   // do not send after abandon
    client.sendInput(responseText + '\n')
    setTimeout(() => { settled = true; clearTimeout(timer); client.close(); resolve() }, 100)
  },
  onClose: () => {},
})
clientRef.current = client
// useEffect(() => () => clientRef.current?.close(), [])  // unmount cleanup
```

## Warnings

### WR-01: Dismissing the join-code modal strands `pendingModalSessionId` (no reset on cancel)

**File:** `frontend/src/components/Hub/HubPanel.tsx:340-368`, `frontend/src/App.tsx:1600-1607`

**Issue:** `handleCardClick` sets `pendingModalSessionId` and asks App to open
`RemoteJoinCodeModal`. `handleCapAcquired` clears it only on a *successful* exchange. If
the user closes the modal (`onClose` → `setJoinModalForSession(null)`), HubPanel is never
notified, so `pendingModalSessionId` and `pendingSourceRectRef` stay set indefinitely.
There is no functional deadlock (a later click re-requests), but the stale pending id is
dead state and a latent bug if any future code keys off "is a modal pending."

**Fix:** Thread an `onRequestRemoteCap` cancel/teardown back to HubPanel, or have
`handleCapAcquired` be paired with a `handleCapCancelled` that App calls from the join
modal's `onClose`, resetting `pendingModalSessionId`/`pendingSourceRectRef`.

### WR-02: `onRequestRemoteCap` overwrites any in-flight `joinModalForSession` (incl. file-browse intent)

**File:** `frontend/src/App.tsx:1380,1600-1607`, `frontend/src/App.tsx:1106-1134`

**Issue:** The Hub's `onRequestRemoteCap` does
`setJoinModalForSession({ …, intent: 'hub-modal' })` unconditionally. If a file-browse
join modal (`intent` undefined) is already open for a different session, this silently
replaces it; `handleModalExchange` then routes by the *new* intent and the original
file-browse request is lost. The two entry points share one `joinModalForSession` slot
with no guard.

**Fix:** Guard the setter when a modal is already open (`if (joinModalForSession) return`
or queue), or include the originating session id in a uniqueness check before replacing.

### WR-03: `theme={terminalTheme ?? ({} as ITheme)}` defeats type safety and can render an unthemed terminal

**File:** `frontend/src/components/Hub/HubPanel.tsx:456`

**Issue:** `({} as ITheme)` is an unsafe assertion that fabricates an empty object as a
valid `ITheme`. `terminalTheme` in App is already guaranteed non-null (App.tsx:271-272
falls back to a real theme), so this fallback is dead — but if `terminalTheme` is ever
genuinely undefined, the interactive terminal mounts with an empty theme object rather
than failing loudly. The cast hides that contract.

**Fix:** Make `terminalTheme` required on `HubPanelProps` (App always supplies it) and
drop the cast, or pass through `undefined` and let `TerminalPanel` apply its own default.

### WR-04: `fontSize={14}` hardcoded in the modal ignores per-session font size and `onFontSizeChange`

**File:** `frontend/src/components/Hub/HubPanel.tsx:455`, `frontend/src/components/Hub/HubInteractiveModal.tsx:46`

**Issue:** The interactive modal mounts `TerminalPanel` with a literal `fontSize={14}` and
`HubInteractiveModal` defaults `onFontSizeChange` to a no-op (`?? (() => {})`). The main
terminal tabs honor `fontSizes[sessionId]` and a working zoom handler (App.tsx:1549-1550).
In the modal, ⌘+/⌘- zoom is silently swallowed and the user's existing per-session font
preference is ignored. Magic number duplicates `DEFAULT_FONT_SIZE`.

**Fix:** Thread the real `fontSize`/`onFontSizeChange` down from App (the modal already
has `relayPort`/`theme`/`pluginConfig` plumbed the same way), or document the intentional
read-only-zoom decision. At minimum use `DEFAULT_FONT_SIZE` instead of a bare `14`.

### WR-05: Escape `stopImmediatePropagation` on `document` can swallow Escape for other global handlers

**File:** `frontend/src/components/Hub/HubModal.tsx:100-109`, `frontend/src/components/Hub/SessionCard.tsx:190-200`, `frontend/src/components/Hub/HubPanel.tsx:249-259`

**Issue:** The modal registers a `document` `keydown` listener that calls
`e.stopImmediatePropagation()` on Escape to prevent the originating card's menu from
double-firing. Because both the modal's and the card's listeners are on `document`,
listener-registration order decides which runs first. The card's menu listener only
mounts when `menuOpen` is true, so the documented case is covered, but
`stopImmediatePropagation` also blocks *any* other `document`-level Escape handler
registered after the modal (current or future — e.g. a global command palette). This is
broad collateral suppression for a narrow goal.

**Fix:** Scope the guard. Attach the modal's Escape handler to the dialog element (with
the dialog focused via the focus-trap), or use a shared "topmost overlay" coordinator
rather than `stopImmediatePropagation` on `document`. At minimum, document that the modal
intentionally suppresses all subsequent global Escape handlers while open.

### WR-06: No focus trap inside the modal — Tab escapes to the Hub behind the overlay

**File:** `frontend/src/components/Hub/HubModal.tsx:123-186`

**Issue:** The dialog sets `role="dialog"` + `aria-modal="true"` and returns focus to the
card on unmount (MODAL-02), but there is no focus *trap*: Tab/Shift-Tab can move focus out
of the dialog to the still-rendered Hub cards underneath (they are `tabIndex={0}`). For an
`aria-modal` dialog this is an accessibility defect — keyboard and screen-reader users can
interact with obscured background controls.

**Fix:** Add a focus trap (cycle Tab within the dialog's focusable elements) or mark
background content `inert`/`aria-hidden` while the modal is open.

### WR-07: Modal component tests are source-string assertions with zero behavioral coverage

**File:** `frontend/src/components/Hub/HubModal.test.tsx:1-49`, `frontend/src/components/Hub/HubInteractiveModal.test.tsx:1-40`, `frontend/src/components/Hub/HubBriefingModal.test.tsx:1-37`, `frontend/src/components/Hub/HubPanel.test.tsx:734-761`

**Issue:** Every test for `HubModal`, `HubInteractiveModal`, and `HubBriefingModal`, plus
the MODAL-06 HubPanel block, is a `?raw` import with `expect(raw).toContain('…')` /
`indexOf` ordering checks. These assert that *source text* contains a string — they pass
even if the logic is inverted, the handler is never wired, or (as in CR-01/CR-02) the
feature is fundamentally broken for remote sessions. e.g. HubBriefingModal.test.tsx:14
only checks the literal `'RelayClient'` appears; it never sends, never asserts delivery,
never exercises the timeout/leak path. The MODAL-06 test (HubPanel.test.tsx:742) asserts
`onRequestRemoteCap?.({` precedes `setModalState` in the *string*, not that an uncapped
remote session is actually blocked at runtime. This is a false sense of safety for the
exact paths that carry the blockers above.

**Fix:** Add behavioral tests: mock `RelayClient` and assert the briefing send opens →
`sendInput` → `close` ordering and the timeout cleanup; mock `GetSessionTailLines` and
assert tail rendering/empty/loading; render `HubPanel` and assert a remote-without-cap
card click does NOT call `setModalState` (spy `onRequestRemoteCap`) while a local click
does. xterm can be mocked to allow mounting `HubModal` in jsdom.

## Info

### IN-01: `relayPort === 0` is treated as a valid port for the modal

**File:** `frontend/src/components/Hub/HubPanel.tsx:450`

**Issue:** The modal renders when `relayPort !== undefined`, but App passes
`relayPort ?? undefined` from `relayPort: number | null` where the un-initialized value
can be `0` in some code paths (the terminal tab grid guards with `relayPort > 0` at
App.tsx:1535). The modal omits the `> 0` guard, so a transient `0` would build
`ws://127.0.0.1:0/...`.

**Fix:** Mirror the tab grid guard: `relayPort !== undefined && relayPort > 0`.

### IN-02: `HubInteractiveModalProps.onFontSizeChange` is declared but never supplied by the shell

**File:** `frontend/src/components/Hub/HubInteractiveModal.tsx:20,46`, `frontend/src/components/Hub/HubModal.tsx:176-183`

**Issue:** `HubModal` renders `HubInteractiveModal` without passing `onFontSizeChange`, so
the prop is always its no-op default. Dead surface that suggests zoom is supported when it
is not (see WR-04).

**Fix:** Remove the optional prop or wire it through from App.

### IN-03: `handleCapAcquired` deps omit `pendingSourceRectRef` reads but include `sessions`/`remoteSessions` — re-registers callback each poll

**File:** `frontend/src/components/Hub/HubPanel.tsx:353-373`

**Issue:** `handleCapAcquired` depends on `[pendingModalSessionId, sessions, remoteSessions]`;
`sessions`/`remoteSessions` change references on every 3s/30s poll, so the callback identity
churns and the `useEffect` at line 371 re-invokes `onRegisterCapAcquired` every poll. App
stores it in a ref (`capAcquiredRef.current = fn`), so this is harmless today, but it is
needless churn and fragile if the registration ever does real work.

**Fix:** Read sessions from a ref inside the callback, or accept the id and let App resolve
the session, keeping the callback stable.

### IN-04: `ariaLabel` for briefing uses session name without indicating remote origin

**File:** `frontend/src/components/Hub/HubModal.tsx:119-121`

**Issue:** Minor copywriting/accessibility gap: the dialog `aria-label` is
`Briefing: ${session.name} needs input` with no origin (Local/hostname), while the visible
header shows the origin marker. Screen-reader users lose the local/remote distinction that
sighted users get — relevant given remote sessions behave differently (CR-01/CR-02).

**Fix:** Include origin in the aria-label, e.g. `Briefing: ${session.name} on ${originText} needs input`.

---

_Reviewed: 2026-06-17_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
