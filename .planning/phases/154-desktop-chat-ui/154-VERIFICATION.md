---
phase: 154-desktop-chat-ui
verified: 2026-06-26T23:20:00Z
status: human_needed
score: 5/5 must-haves verified
behavior_unverified: 4
overrides_applied: 0
behavior_unverified_items:
  - truth: "Day separators remain anchored to the top of the visible viewport while scrolling (CHAT-04)"
    test: "Open a live chat session with messages spanning multiple calendar days and scroll through the history"
    expected: "Day separators ('Today', 'Yesterday', date strings) pin to the top of the thread viewport as you scroll through older messages; they do not scroll out of view"
    why_human: "jsdom has no layout engine — the sticky/absolute virtualizer row-style logic is unit-tested via getRowStyle(), but actual CSS position:sticky pinning requires a real browser rendering context with a scrollable parent"
  - truth: "Opening/closing the chat drawer overlays the terminal's right edge without resizing the PTY or affecting guest views (D-02)"
    test: "Open the Hub interactive modal on a running session; record PTY column count with `tput cols` in the terminal; click the chat toggle button to open the drawer"
    expected: "`tput cols` returns the same value before and after the drawer opens; the terminal grid does not reflow; any connected web/remote guest sees no PTY resize event"
    why_human: "PTY column count requires a live PTY process; jsdom has no layout engine; the unit test (HubInteractiveModal.test.tsx) proves isActive is unchanged and the source has no hub-modal__terminal-col wrapper, but cannot measure actual PTY dimensions"
  - truth: "Press-and-hold Inject button fires MsgSessionInject (0x35) ONLY after a completed ~600ms hold; a tap (pointer-up before 600ms) fires nothing; Enter with @session in the composer fires sendChat not inject (D-08)"
    test: "In a live chat session with @session in the composer, (1) press and immediately release the Inject button; (2) press and hold for >=600ms; (3) type @session in the composer and press Enter"
    expected: "(1) Nothing is injected or sent; (2) the PTY receives the text (visible in the terminal) and the compose field clears; (3) the message appears in the chat thread and the PTY does NOT receive the text"
    why_human: "The 600ms timer logic and Enter routing are unit-tested with vitest fake timers and a spy on the RelayClient, but observing that the PTY actually receives or does not receive text requires a live daemon with a running PTY process"
  - truth: "The fill animation on the Inject button plays left-to-right over 600ms in the native WebView (D-08)"
    test: "Open the Hub chat panel on a live session with @session in the draft, then press-and-hold the Inject button"
    expected: "The button's ::before scaleX fill animates from left to right over exactly 600ms; releasing before 600ms cancels the animation; under prefers-reduced-motion the animation is suppressed but the 600ms threshold still fires the inject"
    why_human: "CSS ::before animations are not testable in jsdom; requires the WKWebView native renderer (or wails dev browser bridge)"
human_verification:
  - test: "Sticky day separators in live session scroll"
    expected: "Day separators stay anchored to the top of the thread viewport while scrolling through a multi-day message history; they do not scroll away"
    why_human: "jsdom cannot evaluate CSS position:sticky; the virtualizer sticky/absolute row-style logic is unit-proven but scroll pinning requires a real browser"
  - test: "Overlay no-resize proof on live PTY (D-02)"
    expected: "Opening/closing the chat drawer does not change the PTY column count (tput cols); no MsgResize is sent to connected guests"
    why_human: "PTY column count is a live-process measurement; jsdom has no layout engine; source inspection and unit tests prove isActive is unchanged and no column wrapper exists, but cannot measure the actual PTY grid"
  - test: "Accidental-Enter guard on live PTY (D-08 carry-forward from Phase 153)"
    expected: "With @session in the composer draft, pressing Enter sends the text as a chat message (appears in the thread) and the PTY does NOT receive the text; pressing-and-holding the Inject button for >=600ms DOES send the text to the PTY"
    why_human: "PTY receive path requires a live running process; Enter-never-injects is unit-tested via Pitfall 7 guard but the PTY echo is only visible in a live terminal pane"
  - test: "Inject fill animation in native WebView (M-20)"
    expected: "Press-and-hold Inject button shows a CSS scaleX fill over 600ms; tap does not animate; prefers-reduced-motion suppresses animation but inject still fires at 600ms"
    why_human: "CSS ::before animations are not renderable in jsdom; requires WKWebView or wails dev bridge"
gaps: []
deferred: []
---

# Phase 154: Desktop Chat UI Verification Report

**Phase Goal:** The desktop GUI shows a fully functional chat panel inside the session modal with safe Markdown rendering and unread notifications.
**Verified:** 2026-06-26T23:20:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC-1 | A user types a message, presses Enter to send, and sees it appear in the thread with alias, tailnet ID, and HH:MM timestamp; hover reveals full ISO-8601; Shift+Enter inserts a newline | VERIFIED | `handleTextareaKeyDown` in ChatPanel.tsx: Enter → `clientRef.current?.sendChat()` + clear; Shift+Enter default; `ChatMessage.tsx` renders `<time title={isoTime}>{hhmm}</time>`; composer tests pass (Enter sends + clears, Shift+Enter newline) |
| SC-2 | Typing `@` opens an autocomplete popover listing participants plus pinned `@session`; arrow-key navigation; Enter confirms | VERIFIED | `MentionPopover.tsx`: `@session` always-first in "Agent" section, never filtered (D-07); `handleDraftChange` detects `@` trigger; Arrow/Enter routing in `handleTextareaKeyDown`; 6 MentionPopover tests + 2 ChatPanel composer tests green |
| SC-3 | Messages that `@mention` the current user are visually distinct; chat-toggle and Hub session card show unread badge count | VERIFIED | Three-signal mention: `.chat-msg--mention::before` (3px bar), rgba tint, `.chat-msg__you-chip` (@ glyph); `ChatBadge` on toggle + `SessionCard`; 5 three-signal tests + 4 ChatBadge tests + 2 SessionCard tests green |
| SC-4 | Pasting `<script>alert(1)</script>` or `<img src=x onerror=alert(1)>` renders completely inert — no script executes, no onerror attribute | VERIFIED | react-markdown v10 + rehype-sanitize; `grep -rn "rehype-raw" frontend/src` returns nothing; ChatMessage.test.tsx asserts `querySelector('script') === null` and `querySelectorAll('[onerror]').length === 0`; both assertions green |
| SC-5 | Day separators appear between calendar days and remain anchored to the top of the visible viewport while scrolling | PRESENT_BEHAVIOR_UNVERIFIED | `getRowStyle` returns `{position:'sticky', top:0, zIndex:2}` (no transform) for active separator; rangeExtractor always keeps the active separator in range; `ChatDaySeparator` has no `position:sticky` (parent owns it — Pitfall 1 guarded); sticky-during-scroll verified by manual UAT only (jsdom has no layout engine) |

**Score:** 4/5 truths fully verified + 1 present-wired-behavior-unverified (SC-5 sticky scroll)

---

### Behavior-Unverified Truths

Four truths are present and wired but their runtime behavior cannot be proven without a live app:

1. **SC-5 sticky day separators** — Code is wired (`getRowStyle` returns `position:sticky`, rangeExtractor keeps active separator); scroll anchoring needs live browser rendering (registered M-22).
2. **D-02 overlay no-resize** — `isActive` bound to modal-open prop not `chatOpen`; no `hub-modal__terminal-col` wrapper in source; actual PTY column count requires a live PTY (registered M-21).
3. **D-08 press-and-hold logic in real app** — Unit tests with fake timers prove 600ms threshold, Enter separation, and sendSessionInject route; PTY receive path requires live daemon (registered M-23).
4. **D-08 fill animation in native WebView** — CSS `::before` scaleX animation cannot be evaluated in jsdom; requires WKWebView (registered M-20).

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|---------|--------|---------|
| `internal/relay/protocol.go` | `ChatSendPayload` struct with `Content string json:"content"` | VERIFIED | Line 150: struct present with correct JSON tag |
| `internal/relay/hub.go` | `Hub.HandleChatSend`, `ErrChatReadOnly` | VERIFIED | Line 572: HandleChatSend; line 19: ErrChatReadOnly; no WriteInput call inside |
| `internal/relay/server.go` | `case MsgChatSend:` in read pump | VERIFIED | Line 365: case present with SEC-01 gate via HandleChatSend |
| `internal/webserver/server.go` | `case relay.MsgChatSend:` in read pump | VERIFIED | Line 1164: case present; SEC-01 gate inside HandleChatSend |
| `frontend/package.json` | `@tanstack/react-virtual@^3.14.3`, `react-textarea-autosize@^8.5.9` | VERIFIED | Both entries confirmed; `grep rehype-raw frontend/package.json` returns nothing |
| `frontend/src/lib/relayClient.ts` | MSG_CHAT/CHAT_SEND/SESSION_INJECT/INJECT_ERROR constants; ChatMessage interface with `alias` field; encoders; optional callbacks | VERIFIED | All constants 0x30/0x31/0x35/0x36 present; `alias: string` (NOT authorAlias); encodeChatSendFrame + encodeSessionInjectFrame; onChat?/onPresence?/onTyping?/onInjectError? optional |
| `frontend/src/components/Hub/ChatMessage.tsx` | Avatar rows, consecutive collapse, three-signal @mention, inject indicator, safe Markdown | VERIFIED | All rendering modes present; react-markdown + remark-gfm + rehype-sanitize; no rehype-raw import |
| `frontend/src/components/Hub/ChatDaySeparator.tsx` | Day-label row, formatDaySeparator, no position:sticky | VERIFIED | presentational component; no sticky in source; formatDaySeparator exported |
| `frontend/src/components/Hub/ChatBadge.tsx` | count=0 → null; count → number; hasMention → `@` glyph | VERIFIED | ChatBadge component: renders null at 0, count text, `@` glyph for hasMention with aria-label |
| `frontend/src/components/Hub/MentionPopover.tsx` | @session pinned first, never filtered; filterable participants; keyboard-nav via onSelect/onClose | VERIFIED | @session in "Agent" section, not in filtered list; filter applied to participants only; Escape global listener |
| `frontend/src/components/Hub/ChatPanel.tsx` | Own RelayClient WS, virtualizer, sticky separators, history dedup, unread accrual, composer with press-and-hold | VERIFIED | Full implementation: separate RelayClient; useVirtualizer with rangeExtractor; WS-first + HTTP history; accrueUnread pure function; handleInjectPointerDown 600ms timer; Enter routes to sendChat only |
| `frontend/src/components/Hub/HubInteractiveModal.tsx` | Overlay mode: position:relative body, position:absolute ChatPanel, no terminal-col wrapper, always-mounted ChatPanel | VERIFIED | `hub-modal__body--interactive { position: relative }`; `.chat-panel { position: absolute; ... }` in style.css; no `hub-modal__terminal-col` in source; ChatPanel always-mounted |
| `frontend/src/components/Hub/SessionCard.tsx` | `unreadCount?` + `hasChatMention?` props + ChatBadge render | VERIFIED | Lines 175-177: props added; line 450: ChatBadge rendered |
| `TESTING.md` | Phase 154 test files in Sections 2, 4, 5; check-traceability-paths.sh exits 0 | VERIFIED (minor warning) | 10 new test files registered (Section 2 +8 vitest, +3 Go); Section 4 has rows for all covered requirements except NOTIF-02 which is mentioned in the CHAT-02 row description but has no dedicated row; M-20..M-23 added to Section 5; `bash tests/check-traceability-paths.sh` exits 0 |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `relay/server.go` read pump | `hub.HandleChatSend` | `case MsgChatSend:` (line 365) | WIRED | Unmarshal ChatSendPayload; call HandleChatSend; log+drop on error |
| `webserver/server.go` read pump | `hub.HandleChatSend` | `case relay.MsgChatSend:` (line 1164) | WIRED | Parallel structure; SEC-01 gate inside HandleChatSend |
| `HandleChatSend` | `chatAppendFn` + `BroadcastChat` | no WriteInput call | WIRED | Verified: `grep WriteInput hub.go` shows only comment reference in HandleChatSend; chatAppendFn called + BroadcastChat(MakeChatFrame(msg)) |
| `ChatPanel.tsx` composer | `RelayClient.sendChat` / `sendSessionInject` | `clientRef.current?.sendChat()` in Enter handler; `clientRef.current?.sendSessionInject()` in 600ms timer | WIRED | Enter path and inject path are strictly separate code branches (Pitfall 7 guard verified) |
| `ChatPanel.tsx` | `MentionPopover.tsx` | `mentionOpen` state + `handleDraftChange` `@` detection | WIRED | Draft change handler detects `@` trigger; popover receives `participants`, `filter`, `activeIndex`, `onSelect`, `onClose` |
| `ChatPanel.tsx` | `ChatMessage.tsx` + `ChatDaySeparator.tsx` | virtualizer render loop `buildItems()` | WIRED | buildItems produces {type:'message'|'separator'} union; render loop dispatches to ChatMessage or ChatDaySeparator |
| `HubInteractiveModal.tsx` | `ChatPanel.tsx` | always-mounted with `open={chatOpen}`, `onUnreadChange` callback | WIRED | ChatPanel mounted unconditionally; onUnreadChange updates unreadCount/hasMention state |
| `HubInteractiveModal.tsx` | `SessionCard.tsx` | unreadCount/hasMention lifted via onUnreadChange | WIRED | SessionCard.tsx accepts `unreadCount?`/`hasChatMention?` and renders ChatBadge; HubInteractiveModal lifts state to Hub via existing session callback |
| `ChatMessage.tsx` | `react-markdown` + `rehype-sanitize` | import chain (no rehype-raw) | WIRED | `import Markdown from 'react-markdown'`; `import rehypeSanitize`; no rehype-raw import anywhere in `frontend/src/` |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| ChatPanel.tsx (thread) | `messages` state | RelayClient `onChat` callback + `loadChatHistory` (GET `/api/chat/{id}/history`) | Yes — live WS frames + HTTP history API backed by `ChatStore` (Phase 151) | FLOWING |
| ChatPanel.tsx (unread) | `unread.count`, `unread.hasMention` | `accrueUnread()` called in `onChat` when `!open || !windowFocused` | Yes — pure function driven by real WS messages | FLOWING |
| HubInteractiveModal.tsx (badge) | `unreadCount`, `hasMention` | `onUnreadChange` callback from ChatPanel | Yes — threaded from ChatPanel's live unread state | FLOWING |
| SessionCard.tsx (badge) | `unreadCount?`, `hasChatMention?` | Props from HubInteractiveModal via Hub session state | Yes — lifted from ChatPanel's unread state | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go ChatSend tests pass | `go test ./internal/relay/... ./internal/webserver/... -run ChatSend -count=1` | PASS (relay ok, webserver ok) | PASS |
| HandleChatSend never writes PTY | `grep WriteInput internal/relay/hub.go` + inspect HandleChatSend body | No WriteInput call inside HandleChatSend; mentioned in comment only | PASS |
| SEC-03 no rehype-raw | `grep -rn "rehype-raw" frontend/src/` | No matches in Hub components or ChatPanel; only MarkdownPreview guard comments | PASS |
| All frontend chat tests | `pnpm test run src/components/Hub/ChatMessage.test.tsx` | 26/26 PASS | PASS |
| ChatBadge + MentionPopover + ChatDaySeparator | Combined run | 22/22 PASS | PASS |
| ChatPanel (all including inject/composer) | `pnpm test run src/components/Hub/ChatPanel.test.tsx` | 39/39 PASS | PASS |
| HubInteractiveModal overlay | `pnpm test run src/components/Hub/HubInteractiveModal.test.tsx` | 16/16 PASS | PASS |
| relayClient wire protocol | `pnpm test run src/lib/relayClient.test.ts` | 42/42 PASS | PASS |
| TypeScript compile | `pnpm exec tsc --noEmit` | Zero errors | PASS |
| Go build | `go build ./...` | Clean | PASS |
| Go vet | `go vet ./internal/relay/... ./internal/webserver/...` | Clean | PASS |
| TESTING.md traceability | `bash tests/check-traceability-paths.sh` | OK: all traceability paths exist | PASS |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| CHAT-01 | 154-01, 154-02, 154-06 | Send/receive chat messages; Enter sends, Shift+Enter newline | SATISFIED | HandleChatSend + read-pump dispatch (Go) + relayClient encoders + ChatPanel composer (TS) |
| CHAT-02 | 154-03 | Author alias + tailnet ID + HH:MM timestamp + ISO-8601 hover; consecutive collapse | SATISFIED | ChatMessage.tsx avatar rows; consecutive collapse via `isFirstInGroup`; `<time title={isoTime}>` |
| CHAT-03 | 154-06 | Auto-grow composer (capped); safe Markdown (remark-gfm, sanitized, no raw HTML) | SATISFIED | react-textarea-autosize minRows=1 maxRows=6; react-markdown + rehype-sanitize; no rehype-raw |
| CHAT-04 | 154-05 | Day separators stick to top while scrolling | PRESENT_BEHAVIOR_UNVERIFIED | getRowStyle returns position:sticky for active separator; no transform (Pitfall 1); rangeExtractor keeps separator in range; scroll behavior is manual UAT |
| MENTION-01 | 154-04, 154-06 | `@` opens autocomplete with pinned `@session`; filterable; keyboard-navigable | SATISFIED | MentionPopover pins @session in "Agent" section, never filtered; Arrow/Enter/Escape handled |
| NOTIF-01 | 154-04, 154-06 | Unread badge on toggle + SessionCard; @mention badge visually distinct | SATISFIED | ChatBadge on toggle (HubInteractiveModal) + SessionCard; `@` glyph for hasMention |
| NOTIF-02 | 154-03 | Mentioned messages highlighted with colorblind-safe signals | SATISFIED | Three-signal: ::before bar (shape) + rgba tint + @you chip (glyph); all three unit-tested |
| SEC-03 | 154-03, 154-06 | Safe Markdown rendering; no rehype-raw; XSS payloads render inert | SATISFIED | rehype-sanitize applied; rehype-raw absent (`grep -rn "rehype-raw" frontend/src` returns no chat matches); XSS unit tests pass |

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `frontend/src/components/Hub/ChatPanel.tsx` | 362-366 | `chat-panel__composer-slot` was described as empty in 154-05 plan but filled in 154-06 (planned interface boundary) | Info | Not a stub — expected two-wave wiring, now complete |

No TBD/FIXME/XXX markers found in phase-modified files. No unreferenced debt markers. No stub returns (empty array/null returns where real data should flow).

---

### Minor Convention Gap: NOTIF-02 Traceability Row

NOTIF-02 is a Phase 154 requirement covered by `ChatMessage.test.tsx` but has no dedicated Section 4 traceability row in TESTING.md. Coverage is mentioned in the CHAT-02 row description text. Per the CLAUDE.md standing rule, a dedicated `| NOTIF-02 | frontend/src/components/Hub/ChatMessage.test.tsx | vitest | ... |` row should exist.

This is a documentation gap, not a functional gap. The implementation and tests are correct and green.

---

### D-02 Overlay Mode Verification (Design Revision Confirmed)

The implementation correctly adopts **overlay mode** (not push mode):
- `hub-modal__body--interactive { position: relative }` is the containing block
- `.hub-modal__body--interactive .chat-panel { position: absolute; top: 0; right: 0; bottom: 0; width: 360px; z-index: 5 }` — floats over the terminal
- TerminalPanel receives `isActive={open}` (the modal-open prop), NOT `chatOpen` — toggling the drawer never changes `isActive`
- No `hub-modal__terminal-col` wrapper — confirmed by source inspection and the HubInteractiveModal test that asserts `expect(raw).not.toContain('hub-modal__terminal-col')`
- `client.sendResize` is not called on drawer toggle — the PTY is left untouched

This satisfies the D-02 design revision (push mode → overlay mode for Issue #109 host-authority PTY model).

---

### Human Verification Required

#### 1. Day Separators Scroll Anchoring (CHAT-04 / M-22)

**Test:** Open a live chat session with messages spanning multiple calendar days (or fast-forward the system clock across midnight). Scroll up through history.
**Expected:** Day separators ("Today", "Yesterday", date strings) pin to the top of the chat thread viewport while scrolling; they do not scroll away; the list auto-scrolls to bottom on new messages.
**Why human:** jsdom has no layout engine — CSS `position:sticky` cannot be evaluated without a real browser rendering context. The `getRowStyle` pure function is unit-tested (returns sticky style with no transform for the active separator), but actual visual pinning requires a live scroll test.

#### 2. Overlay No-Resize Proof on Live PTY (D-02 / M-21)

**Test:** Open the Hub interactive modal on a running session. In the terminal, run `tput cols` to record the PTY column count. Click the chat toggle button to open the overlay drawer. Run `tput cols` again.
**Expected:** Same value before and after — no PTY resize. Any connected web guest sees no terminal reflow.
**Why human:** PTY column count is a live-process measurement. jsdom has no layout engine. Unit tests prove `isActive` is unchanged by chatOpen and the source has no flex-shrink column wrapper, but the actual PTY grid dimensions require a running terminal.

#### 3. Accidental-Enter Guard on Live PTY (D-08 / M-23)

**Test:** With @session in the composer draft: (1) press Enter (no modifier, no hold); (2) then press-and-hold the Inject button for >=600ms.
**Expected:** (1) The text appears in the chat thread as a chat message; the PTY does NOT receive the text (no echo in the terminal). (2) The PTY DOES receive the text (visible in the terminal) and the compose field clears.
**Why human:** Enter-never-injects is unit-tested via fake timers and a spy. PTY receive visibility requires a live terminal process where output is observable.

#### 4. Inject Fill Animation in Native WebView (D-08 / M-20)

**Test:** Open the Hub chat panel on a live session with @session in the composer draft. Press and hold the Inject button.
**Expected:** The button's `::before` pseudo-element scaleX fill animates left→right over 600ms; releasing before 600ms cancels and no inject fires; after 600ms the inject fires and the field clears. Under `prefers-reduced-motion` the animation is suppressed but the inject still fires at 600ms.
**Why human:** CSS `::before` animations are not renderable in jsdom. Requires the WKWebView native renderer or the wails dev bridge.

---

## Gaps Summary

No functional gaps identified. All 5 roadmap success criteria are satisfied at the code level. The single non-verified truth (SC-5 day separator scroll anchoring) is present and wired but requires live-browser confirmation — registered as M-22 in TESTING.md §5.

The four human verification items are not regressions or missing implementations; they are the live-app observability requirements that the automated test suite correctly defers to manual UAT.

Minor documentation gap: NOTIF-02 lacks a dedicated traceability row in TESTING.md Section 4 (it is covered in the CHAT-02 row description). Functionally the requirement is implemented and tested.

---

_Verified: 2026-06-26T23:20:00Z_
_Verifier: Claude (gsd-verifier)_
