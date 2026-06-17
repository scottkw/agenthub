---
phase: 134-modal-interaction
verified: 2026-06-17T11:05:00Z
status: human_needed
score: 18/18
overrides_applied: 0
human_verification:
  - test: "Open a local session card and observe the modal expand"
    expected: "Modal grows from the card's center position with a smooth 220ms scale animation; the card that was clicked is the visual origin"
    why_human: "Shared-element grow animation cannot be verified by grep or unit tests — requires visual observation in the Wails native webview"
  - test: "Close the modal (Escape, X button, and click-outside — all three paths)"
    expected: "Modal shrinks back toward the originating card position with a 180ms shrink animation; focus returns to the card that was originally clicked"
    why_human: "Animation direction, origin point accuracy, and focus-return UX require live observation"
  - test: "Open a non-attention session modal; run a command; resize the window"
    expected: "Full interactive terminal renders correctly; typing works; resize triggers terminal reflow without layout jank; copy/paste and scrollback search work"
    why_human: "MODAL-03/05 terminal interaction quality (MODAL-05: resize/copy/paste/scrollback) cannot be confirmed without a live PTY"
  - test: "Open a waiting/attention session briefing modal"
    expected: "Terminal tail lines are displayed in the briefing view (not 'No recent output available'); the respond textarea auto-focuses; typing a response and clicking Send Response delivers input to the session and closes the modal"
    why_human: "MODAL-04 round-trip requires a live daemon session in the waiting state; tail content accuracy needs real data"
  - test: "Live two-machine tailnet test (MODAL-06 remote path): on Machine A, open the Hub and click a remote session card; complete the join-code exchange; observe the modal open"
    expected: "Join-code modal appears (cap not yet acquired); after entering the join code the Hub modal opens for the remote session; typing in the interactive terminal executes on Machine B; for a waiting remote session the briefing tail shows real output from Machine B's PTY and Send Response delivers input"
    why_human: "Requires two live Tailscale-connected machines; cap-gated WS proxy correctness end-to-end cannot be automated without real tailnet peers"
  - test: "Test reduced-motion: enable 'Reduce motion' in macOS Accessibility settings, then open and close a modal"
    expected: "Modal appears and disappears instantly with no animation (opacity/transform instant, animation: none applied)"
    why_human: "prefers-reduced-motion behavior requires the OS accessibility setting to be toggled and observed visually"
---

# Phase 134: Card → Modal Interaction — Verification Report

**Phase Goal:** Clicking any card opens a full interactive or briefing modal with a shared-element grow/shrink animation, and closing it returns focus cleanly to the originating card.
**Verified:** 2026-06-17T11:05:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Clicking a card body fires `onCardClick` with session + bounding rect | VERIFIED | `SessionCard.tsx:248,253` — article onClick calls `onCardClick?.(session, e.currentTarget.getBoundingClientRect())`; Open button and menu button both call `e.stopPropagation()` (lines 276, 396); source-inspection tests pass |
| 2 | `SessionCardGrid` threads `onCardClick` through both render paths | VERIFIED | `SessionCardGrid.tsx:246,289` — `onCardClick={onCardClick}` present in both the named-group and workDir-group render paths |
| 3 | `HubModal` renders overlay + dialog with `role="dialog"`, `aria-modal`, grow/shrink phase machine | VERIFIED | `HubModal.tsx:84,139-154` — `phase` state `'entering'|'open'|'exiting'`; `hub-modal--${phase}` className; `onAnimationEnd` advances phases; `role="dialog"` and `aria-modal="true"` present |
| 4 | `transformOrigin` is set from the source card's bounding rect so grow animation originates from the card | VERIFIED | `HubModal.tsx:123,149` — `transformOrigin` computed as `${sourceRect.left + sourceRect.width/2}px ${sourceRect.top + sourceRect.height/2}px`; applied as inline style on the panel |
| 5 | Escape, close button, and click-outside all trigger the shrink/close path; focus returns on unmount | VERIFIED | `HubModal.tsx:110-120` — `document.addEventListener('keydown')` with `e.stopImmediatePropagation()` on Escape; overlay `onClick={handleClose}`; panel `onClick={(e) => e.stopPropagation()}`; `cardFocusRef` captures `document.activeElement` on mount and calls `.focus()` in cleanup |
| 6 | `HubModal` routes to `HubInteractiveModal` (non-attention) or `HubBriefingModal` (attention) | VERIFIED | `HubModal.tsx:79,188-206` — `isAttentionStatus(deriveHubStatus(session))` drives the branch; both components imported and rendered |
| 7 | `HubInteractiveModal` mounts `TerminalPanel` with `isActive` bound to the `'open'` phase (fit-safe timing) | VERIFIED | `HubInteractiveModal.tsx:46-55` — `isActive={open}` where `open` = `isOpen` prop; `HubModal.tsx:199` passes `isOpen={phase === 'open'}`; no second `RelayClient` constructed; tests pass |
| 8 | `HubBriefingModal` fetches real terminal tail via `GetSessionTailLines` (local) and renders it | VERIFIED | `HubBriefingModal.tsx:146-149` — `GetSessionTailLines(session.id, 20).then(setTailLines)`; TAIL-01c behavioral test confirms local path |
| 9 | `HubBriefingModal` sends response via `RelayClient.sendInput` inside `onOpen` (race-safe) with 4096-char guard | VERIFIED | `HubBriefingModal.tsx:186-204` — `sendInput` called only inside `onOpen`; `maxLength={4096}` on textarea; CR-03-01a behavioral test confirms ordering |
| 10 | Send is disabled when response textarea is empty | VERIFIED | `HubBriefingModal.tsx:161` — `sendDisabled = sending \|\| responseText.trim() === ''`; MODAL-04 test "Send button disabled when response text is empty" passes |
| 11 | Modal CSS: overlay z-index 200, grow/shrink keyframes inside no-preference guard, reduced-motion fallback, token-only colors | VERIFIED | `style.css:5101-5360` — `.hub-modal-overlay { z-index: 200 }`; four `@keyframes hub-modal-*` at root scope; animation assignments inside `prefers-reduced-motion: no-preference`; `prefers-reduced-motion: reduce` block sets `animation: none`; `hub-modal__send-btn` uses `var(--hub-accent)`; `hub-modal__tail` uses `var(--hub-preview-bg)`; CSS contract test (14 assertions) passes |
| 12 | `HubPanel` opens modal on local card click; gates uncapped remote clicks to `onRequestRemoteCap` | VERIFIED | `HubPanel.tsx:353-363` — `handleCardClick` checks `isRemote && !remoteCapsCached?.has(session.id)` before setting `modalState`; `<HubModal>` render gated on `relayPort > 0`; FE-ROUTE-01a/b behavioral tests pass |
| 13 | After successful cap exchange, Hub modal auto-opens for the pending remote session | VERIFIED | `HubPanel.tsx:367-381` — `handleCapAcquired` callback matches `sessionId === pendingModalSessionId`, finds session, opens modal; registered with `App.tsx` via `onRegisterCapAcquired` |
| 14 | `App.tsx` supplies `relayPort/terminalTheme/pluginConfig` and the intent-discriminated cap-acquired dispatch | VERIFIED | `App.tsx:1381-1401` — `relayPort`, `terminalTheme`, `pluginConfig`, `remoteCapsCached`, `onRequestRemoteCap`, `onRegisterCapAcquired`, `onRegisterCapCancelled`, `fontSizes`, `onFontSizeChange` all passed to `<HubPanel>`; `handleModalExchange` branches on `intent`; file-browse path preserved |
| 15 | `GET /api/relay/remote/{sessionID}/ws` is mounted and cap-gated; proxy copies frames bidirectionally on request context | VERIFIED | `remote_ws_proxy.go:40-114` — 131 lines; cap lookup at line 54; `websocket.Accept` with loopback origin allowlist; upstream dial injects `Origin`; copy goroutines use `r.Context()`; `relay_remote_files.go:74` mounts route; all 6 WS-PROXY tests pass under `-race` |
| 16 | `relay.LoopbackOriginPatterns` exported and reused for inbound accept | VERIFIED | `internal/relay/server.go:363` — `func LoopbackOriginPatterns(host string) []string { return loopbackOriginPatterns(host) }`; used at `remote_ws_proxy.go:65` |
| 17 | `RelayClient` builds daemon-proxy URL when `remote: true`; cap token never enters React state or URL | VERIFIED | `relayClient.ts:88` — `/api/relay/remote/${sessionId}/ws` path selected; no cap param; `relayClient.test.ts` FE-URL-01a/b behavioral URL tests pass |
| 18 | `isRemote` discriminator threaded from `HubPanel` → `HubModal` → leaf components; `terminalTheme` required (no unsafe cast); real `fontSize`/`onFontSizeChange` threaded; `relayPort > 0` guard; WR-01/WR-02/WR-03/WR-04 all fixed | VERIFIED | `HubPanel.tsx:481-493` — `isRemote` computed and passed as `remote={isRemote}`; no `({} as ITheme)` cast present (grep returns 0); `fontSize={fontSizes?.[modalState.session.id] ?? DEFAULT_FONT_SIZE}`; `relayPort > 0` guard at line 480; `handleCapCancelled` at line 391; WR-02 guard at `App.tsx:1392` |

**Score:** 18/18 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/Hub/HubModal.tsx` | Modal shell: overlay, phases, Escape/click-outside, focus return, routing | VERIFIED | 210 lines; `role="dialog"`, `aria-modal`, `transformOrigin`, `isAttentionStatus`, `cardFocusRef` all present |
| `frontend/src/components/Hub/HubInteractiveModal.tsx` | TerminalPanel host (MODAL-03/05) | VERIFIED | 58 lines; mounts `TerminalPanel` with `isActive={open}`, `remote` seam threaded |
| `frontend/src/components/Hub/HubBriefingModal.tsx` | Tail + respond + Send (MODAL-04) | VERIFIED | 291 lines; `GetSessionTailLines`, `RelayClient`, `sendInput` in `onOpen`, `maxLength={4096}`, `clientRef` + `settled` guard, unmount cleanup |
| `frontend/src/components/Hub/SessionCard.tsx` | `onCardClick` + stopPropagation (MODAL-01) | VERIFIED | `onCardClick` declared; `getBoundingClientRect` in handler; 2x `e.stopPropagation()` confirmed |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | `onCardClick` threaded through both paths | VERIFIED | `onCardClick={onCardClick}` at lines 246 and 289 |
| `frontend/src/components/Hub/HubPanel.tsx` | Modal state, cap gate, `HubModal` render | VERIFIED | `modalState` state; `handleCardClick` with remote gate; `<HubModal>` render with `isRemote`; `relayPort > 0` guard |
| `frontend/src/App.tsx` | Modal props + cap-acquired/cancelled wiring | VERIFIED | All 9 new/updated props passed to HubPanel; `capAcquiredRef`/`capCancelledRef`; `intent` discriminator in `handleModalExchange` |
| `frontend/src/style.css` | All `.hub-modal*` rules + keyframes + reduced-motion | VERIFIED | `.hub-modal-overlay` z-index 200; 4 keyframes at root; `prefers-reduced-motion: no-preference` guard; `reduce` fallback |
| `internal/daemon/remote_ws_proxy.go` | `handleRemoteSessionWS` + `copyWS` | VERIFIED | 131 lines; both functions present; cap lookup, origin injection, context-correct copy loop |
| `internal/daemon/relay_remote_files.go` | Route `GET /api/relay/remote/{sessionID}/ws` | VERIFIED | Line 74 mounts the route; no `FilesCORS` wrapper |
| `internal/relay/server.go` | `LoopbackOriginPatterns` exported | VERIFIED | Line 363; delegates to unexported `loopbackOriginPatterns` |
| `internal/daemon/remote_ws_proxy_test.go` | WS-PROXY-01..06 integration tests | VERIFIED | 362 lines; all 6 tests named and present; all pass under `-race` |
| `frontend/src/lib/relayClient.ts` | `remote` seam selecting daemon-proxy URL | VERIFIED | Line 88; `opts?.remote` flag; behavioral URL tests pass |
| `frontend/src/components/TerminalPanel.tsx` | `remote` prop threaded into `RelayClient` construction | VERIFIED | Line 66 prop; line 288 passed to `new RelayClient` as `{ remote }` |
| `frontend/src/components/Hub/HubModal.test.tsx` | Source-inspection tests for HubModal | VERIFIED | Tests pass; asserts `role="dialog"`, `transformOrigin`, `cardFocusRef`, `Escape`, `isAttentionStatus` routing |
| `frontend/src/components/Hub/HubInteractiveModal.test.tsx` | Source-inspection + behavioral tests | VERIFIED | Tests pass |
| `frontend/src/components/Hub/HubBriefingModal.test.tsx` | Source-inspection + CR-03-01 + TAIL-01 behavioral tests | VERIFIED | 13 tests pass; mocked RelayClient confirms send ordering, timeout cleanup, remote tail |
| `frontend/src/components/__tests__/style.hub.modal.test.ts` | CSS contract assertions | VERIFIED | 14 tests pass; z-index 200, 4 keyframes, no-preference guard, reduced-motion, token colors |
| `frontend/src/components/Hub/HubPanel.test.tsx` | FE-ROUTE-01a/b behavioral tests | VERIFIED | Remote-without-cap gate and local modal-open behavioral tests pass |
| `frontend/src/lib/relayClient.test.ts` | FE-URL-01 URL behavioral tests | VERIFIED | Local and remote URL forms asserted; cap token absent from both |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `SessionCardGrid.tsx` | `SessionCard.onCardClick` | `onCardClick={onCardClick}` in both render paths | VERIFIED | Lines 246, 289 |
| `SessionCard.tsx` article onClick | `onCardClick` callback | `getBoundingClientRect` passed to handler | VERIFIED | Lines 248, 253 |
| `HubPanel.tsx handleCardClick` | `onRequestRemoteCap` (App.tsx) | remote-without-cap gate returns early | VERIFIED | Lines 354-362 |
| `App.tsx handleModalExchange` | `HubPanel onCapAcquired` | `capAcquiredRef.current?.(pending.id)` | VERIFIED | `App.tsx:1133`; `HubPanel.tsx:391` `handleCapCancelled` + `capCancelledRef` |
| `HubPanel.tsx` | `<HubModal>` | `modalState` render | VERIFIED | Lines 480-496 |
| `HubModal.tsx` | `HubInteractiveModal` / `HubBriefingModal` | `isAttentionStatus` routing | VERIFIED | Lines 79, 188-206 |
| `HubModal.tsx` panel | `sourceRect` | `transformOrigin` inline style from card center | VERIFIED | Line 123, 149 |
| `HubInteractiveModal` | `TerminalPanel` / `RelayClient` remote seam | `remote` prop forwarded | VERIFIED | `HubInteractiveModal.tsx:54`; `TerminalPanel.tsx:288` |
| `HubBriefingModal` | WS scrollback snapshot | `onOutput` accumulation for remote tail | VERIFIED | Lines 109-135; TAIL-01a/b/c tests pass |
| `HubBriefingModal handleSend` | `settled` guard + `clientRef` + `clearTimeout` | CR-03 leak/race fix | VERIFIED | Lines 170, 76, 200 |
| `relay_remote_files.go` mux | `handleRemoteSessionWS` | `GET /api/relay/remote/{sessionID}/ws` route | VERIFIED | Line 74 |
| `handleRemoteSessionWS` | `RemoteCapStore.Get(sid)` | cap lookup before dial | VERIFIED | `remote_ws_proxy.go:54` |
| `handleRemoteSessionWS` upstream Dial | peer `/sessions/{sid}/ws` | `wss://` + `?cap=T` + injected `Origin` header | VERIFIED | Lines 80-96; WS-PROXY-04 test confirms `Origin` injection |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| `HubBriefingModal` | `tailLines` (local) | `GetSessionTailLines(session.id, 20)` Wails RPC | Yes — Go engine reads PTY scrollback buffer | FLOWING |
| `HubBriefingModal` | `tailLines` (remote) | `RelayClient` + `extractTailLines` from WS snapshot frames | Yes — peer replays PTY scrollback on connect | FLOWING |
| `HubInteractiveModal` | live terminal output | `TerminalPanel` → `RelayClient` → relay/proxy WS | Yes — direct PTY connection via relay or daemon proxy | FLOWING |
| `HubPanel.tsx` | `modalState` | `handleCardClick` from DOM `getBoundingClientRect` | Yes — real `SessionInfo` + real DOMRect | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| 1699 frontend tests pass | `pnpm exec vitest run` | 1699/1699 PASS, 104 test files | PASS |
| Modal component tests (HubModal, HubInteractiveModal, HubBriefingModal) | `vitest run HubModal HubInteractiveModal HubBriefingModal` | 31/31 PASS | PASS |
| CSS contract tests | `vitest run style.hub.modal` | 14/14 PASS | PASS |
| HubPanel behavioral tests (FE-ROUTE-01a/b) | `vitest run HubPanel` | 43/43 PASS | PASS |
| WS proxy tests WS-PROXY-01..06 | `go test ./internal/daemon/ -run RemoteSessionWS -race` | 6/6 PASS (12.26s incl. LongLived) | PASS |
| Go build | `go build ./...` | No output (clean) | PASS |
| TypeScript | `pnpm exec tsc --noEmit` | No output (clean) | PASS |

### Probe Execution

No `probe-*.sh` files declared or conventionally present for this phase. Step 7c: SKIPPED (no probe scripts).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| MODAL-01 | 134-01,02,04,05 | Clicking a card expands into modal via grow animation | VERIFIED | `SessionCard` → `HubPanel` → `HubModal` end-to-end wired; CSS keyframes present; human UAT needed for visual confirm |
| MODAL-02 | 134-01,04 | Closing shrinks back + restores focus | VERIFIED | `handleClose` sets `phase='exiting'`; `cardFocusRef` captures and restores focus; human UAT for visual confirm |
| MODAL-03 | 134-01,03,04,05,06,07,08 | Non-blocked sessions → full interactive terminal (TerminalPanel + RelayClient) | VERIFIED | `HubInteractiveModal` mounts `TerminalPanel`; `isActive` gated on `'open'` phase; remote seam threaded |
| MODAL-04 | 134-01,03,05,07,08 | Waiting/needs-input sessions → briefing view with real tail + respond | VERIFIED | `HubBriefingModal` fetches `GetSessionTailLines` (local) or WS snapshot (remote); `sendInput` race-safe; `maxLength=4096` |
| MODAL-05 | 134-01,03,06,07 | Modal is fully functional: resize, copy/paste, scrollback search | VERIFIED (partial) | `TerminalPanel` is the same component used by normal tabs (inherited capabilities); human UAT required for resize/copy/paste/scrollback |
| MODAL-06 | 134-01,05,06,07,08 | Remote session cap-gate: join-code exchange + cap-gated WS proxy | VERIFIED | Cap gate in `HubPanel.handleCardClick`; daemon proxy `GET /api/relay/remote/{sid}/ws`; `RelayClient` remote seam; all 6 WS-PROXY tests pass; human live-UAT required for real tailnet round-trip |

**A11Y coverage note (REQUIREMENTS.md):**
- A11Y-01 (color-safe): SATISFIED — `HubModal` status icons use shape+label; attention badge uses `BellAlertIcon` + text
- A11Y-02 (keyboard: Enter/Space expand, Escape closes): PARTIAL — cards are `tabIndex=0` with `onKeyDown` for Enter/Space (partial); Escape on modal works; focus trap (A11Y-04) not implemented — both explicitly deferred to Phase 135 per REQUIREMENTS.md traceability (A11Y-02 Phase 135 Pending, A11Y-04 Phase 135 Pending)
- A11Y-03 (reduced-motion): VERIFIED — CSS `prefers-reduced-motion: reduce` block sets `animation: none`; CSS test passes

**Confirmed deferrals (not gaps):**
- WR-05 (broad document-level Escape suppression): deferred to Phase 135 with in-code comment at `HubModal.tsx:102-114`
- WR-06 (focus trap, A11Y-04): deferred to Phase 135 with in-code comment at `HubModal.tsx:134-137`
- WR-03 from REVIEW (read-only cap silent-drop indicator): acknowledged in code at `HubBriefingModal.tsx:275-278`; requires colorblind-safe non-color cue, deferred to Phase 135 per project constraint
- WR-04 from REVIEW (remote tail truncation with large scrollback): ADDRESSED — idle-timer approach (150ms quiescence) replaces fixed 500ms window; WR-04 note retained in comments as informational

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `HubBriefingModal.tsx` | 275-278 | NOTE comment: read-only cap silent drop | INFO | Documented carry-forward to Phase 135; not a functional regression — deferred per signed-off constraint |

No `TBD`, `FIXME`, or `XXX` markers found in any phase-modified files.

No hardcoded hex colors found in `.hub-modal*` CSS declaration blocks. No empty `return null` stubs.

### Human Verification Required

#### 1. Grow Animation Visual Confirmation (MODAL-01)

**Test:** Open a session card by clicking it in the Hub grid.
**Expected:** The modal expands from the card's center position — not from screen center or a fixed point. The grow animation duration is approximately 220ms with an ease-out curve.
**Why human:** `transformOrigin` is set correctly in code but the visual perception of the animation origin (card position vs. center) requires live observation in the Wails native webview.

#### 2. Shrink Animation + Focus Return (MODAL-02)

**Test:** Close the modal using Escape, the X button, and clicking the overlay (three separate tests).
**Expected:** All three paths trigger a shrink animation back toward the originating card. After unmount, keyboard focus is on the card that was clicked to open the modal.
**Why human:** Animation direction and focus-return UX require live observation; jsdom tests mock the DOM and cannot verify visual animation or real focus state.

#### 3. Interactive Terminal Functional Check (MODAL-03/05)

**Test:** Open a running (non-attention) session modal. Type a command and press Enter. Resize the app window. Try copy, paste, and scrollback search.
**Expected:** Terminal renders and accepts input (MODAL-03). Resize causes the terminal to reflow correctly without computing 0-column dimensions (Pitfall 1 guard on `isActive={phase === 'open'}`). Copy/paste and scrollback search work (MODAL-05).
**Why human:** Live PTY I/O and terminal resize behavior cannot be tested in jsdom (xterm requires canvas APIs).

#### 4. Briefing Modal Round-Trip (MODAL-04)

**Test:** Put a session into the `waiting` state (agent prompts for input). Click its card in the Hub. Observe the briefing view. Type a response and click Send Response.
**Expected:** The briefing view displays the real terminal tail (the prompt the agent printed). The respond textarea auto-focuses. Clicking Send Response delivers the text to the session's PTY and closes the modal. The session leaves the `waiting` state.
**Why human:** Requires a live daemon session in the `waiting` state and real PTY round-trip.

#### 5. Remote Session Live UAT — Two-Machine Tailnet Test (MODAL-06)

**Test:** On Machine A (Hub visible), click a remote session card that belongs to Machine B (no cap yet cached). Complete the join-code exchange. Observe the Hub modal open for the remote session. Type a command. For a `waiting` remote session, observe the briefing tail and Send Response.
**Expected:** The join-code modal appears on click (cap not cached). After exchanging the join code, the Hub modal opens automatically (cap-acquired auto-open path). Interactive terminal shows live output from Machine B's PTY. For a briefing session on Machine B, the tail shows real output from the remote peer (not empty/`No recent output available`). Send Response delivers input to Machine B's PTY. Font-size zoom in the remote interactive modal works.
**Why human:** Requires two live Tailscale-connected machines. The daemon-side WS proxy (WS-PROXY-01..06) is integration-tested but the real tailnet end-to-end path (HMAC cap verification, real peer PTY, relay scrollback replay) requires live infrastructure.

#### 6. Reduced-Motion Behavior (A11Y-03)

**Test:** Enable "Reduce Motion" in macOS System Settings → Accessibility → Display. Open and close a Hub modal.
**Expected:** The modal appears and disappears instantly with no scale or fade animation. No flash of invisible content.
**Why human:** `prefers-reduced-motion: reduce` CSS is verified at the source level but the actual OS system setting must be toggled and behavior observed.

---

## Gaps Summary

No gaps were identified. All 18 must-have truths are VERIFIED in the codebase. The phase goal — clicking any card opens a full interactive or briefing modal with a shared-element grow/shrink animation, and closing it returns focus cleanly to the originating card — is architecturally complete in code.

Outstanding items are:

1. **Six human UAT items** (live animation, PTY interaction, remote two-machine tailnet) that cannot be automated without a running Wails app and real tailnet infrastructure.
2. **Three confirmed deferrals** to Phase 135 (a11y): focus trap / A11Y-04, broad Escape suppression refinement / WR-05, and colorblind-safe read-only indicator / WR-03. These are signed-off, documented in-code, and do not block the phase goal.

The REVIEW.md finding WR-01 (tail path leak) was resolved in the code before this verification ran — the `useEffect` cleanup in `HubBriefingModal.tsx:139-143` closes the `tailClient` and clears its timers on unmount.

---

_Verified: 2026-06-17T11:05:00Z_
_Verifier: Claude (gsd-verifier)_
