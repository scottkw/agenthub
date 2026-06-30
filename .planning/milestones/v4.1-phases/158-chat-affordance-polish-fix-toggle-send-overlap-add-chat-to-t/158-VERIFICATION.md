---
phase: 158-chat-affordance-polish-fix-toggle-send-overlap-add-chat-to-t
verified: 2026-06-27T18:10:00Z
status: passed
score: 6/11 must-haves verified
behavior_unverified: 5
overrides_applied: 0
behavior_unverified_items:

  - truth: "When the chat drawer is open in the Hub interactive modal, the chat toggle button relocates clear of the 360px drawer (right:372px) so it no longer overlaps or obscures the composer Send/Inject button."
    test: "Open a live session in the Hub interactive modal, open the chat drawer, and inspect the rendered pixel position of the toggle button relative to the composer Send/Inject button."
    expected: "The toggle button sits visually clear of the Send/Inject button (no overlap) and the drawer occupies the right 360px of the modal."
    why_human: "JSDOM performs no layout. The source-gate proves the CSS rule's presence and offset (right:372px) but cannot measure rendered pixel geometry. A real browser with a layout engine is required."

  - truth: "Because the rule is unscoped (matches the shared .hub-modal__chat-toggle classes), it also corrects the identical latent overlap on the web-share surface."
    test: "Open a web-share session in a browser, open the chat drawer, and visually confirm the toggle relocates clear of the composer on the web-share surface."
    expected: "Same relocation behavior as the Hub modal — toggle not overlapping Send/Inject on the web-share surface."
    why_human: "Runtime visual verification on the web-share surface cannot be measured in JSDOM. The rule is unscoped and structurally applies, but the rendered outcome needs a live browser."

  - truth: "From a live session opened in a direct terminal TAB (not the Hub card modal), a chat toggle button is present and toggles a working ChatPanel drawer."
    test: "Open a live session in a terminal tab (not via Hub card modal), click the chat toggle, send a message, and confirm the ChatPanel drawer receives and displays it."
    expected: "Chat toggle present, drawer opens in overlay mode, messages flow through the relay loopback path."
    why_human: "JSDOM mocks both TerminalPanel and ChatPanel. The working ChatPanel drawer with actual messages requires a live daemon + relay + WebView render."

  - truth: "The tab drawer is overlay mode (D-02): position:absolute over the terminal's right edge; the PTY is NEVER resized when the drawer opens/closes."
    test: "With a live session in a terminal tab, open and close the chat drawer while observing the terminal grid — confirm no reflow/garble occurs and the terminal columns/rows do not change."
    expected: "Terminal content is stable (no reflow/garble) before, during, and after drawer open/close. PTY sendResize is never triggered by the chat toggle."
    why_human: "The isActive-not-bound-to-chatOpen invariant is verified by Test 3 (prop forwarding). The actual PTY sendResize suppression requires a live PTY — JSDOM mocks TerminalPanel and cannot observe real resize events."

  - truth: "Cross-surface parity (release-blocking, upstream PARITY-01): the GUI terminal tab now matches the Hub interactive modal and the web-share view — same ChatPanel, same toggle, same ChatBadge."
    test: "Compare chat affordance behavior side-by-side on all three surfaces (GUI terminal tab, Hub interactive modal, web-share) with a live session: open drawer, send message, confirm unread badge, mention badge."
    expected: "All three surfaces show the same toggle button, same ChatPanel overlay, same ChatBadge behavior — functionally identical chat affordance."
    why_human: "Visual and behavioral parity across three surfaces cannot be confirmed from JSDOM alone. Requires live rendering on each surface."
human_verification:

  - test: "M-29 — CHAT-FIX-01: Toggle/Send non-overlap visual check"
    expected: "With a live session open in the Hub interactive modal, open the chat drawer and confirm the chat toggle button sits clear of (does not overlap/obscure) the composer Send/Inject button. Clicking the toggle still closes the drawer."
    why_human: "JSDOM performs no layout — rendered pixel overlap between position:absolute elements cannot be measured in vitest. Source-gate only proves the CSS rule's presence and offset (right:372px)."

  - test: "M-30 — CHAT-PARITY-01: Terminal-tab chat affordance (overlay, no PTY resize, StatusBar preserved, cross-surface parity)"
    expected: "Open a live session in a direct terminal TAB (not the Hub card modal). A chat toggle is present. Opening the chat drawer shows a working ChatPanel overlay. The terminal is NOT resized (no reflow/garble). The StatusBar remains visible below the drawer. Unread badge accrues while drawer is closed. Cross-surface parity confirmed: GUI tab matches Hub modal and web-share."
    why_human: "Requires a live daemon + PTY + WebView render. JSDOM cannot verify overlay geometry, no-resize invariant with a real PTY, or visual cross-surface parity."
---

# Phase 158: Chat Affordance Polish Verification Report

**Phase Goal:** Two chat-affordance defects found during v4.1 UAT are resolved: (1) the chat toggle no longer covers the composer's Send button when the drawer is open (CHAT-FIX-01), and (2) the chat affordance (toggle + ChatPanel) is reachable from the raw session terminal tab, not only the Hub interactive modal and web-share view — closing a cross-surface parity gap (CHAT-PARITY-01).
**Verified:** 2026-06-27T18:10:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

#### Plan 01 Truths (CHAT-FIX-01)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | When the chat drawer is open, the chat toggle relocates to right:372px clearing the 360px drawer and the composer Send/Inject button | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS rule `.chat-panel--open ~ .hub-modal__chat-toggle { right: 372px }` confirmed in style.css:6059; source-gate 3/3 pass. Rendered pixel geometry requires live browser (M-29). |
| 2 | The relocated toggle remains the affordance that closes the drawer (clickable, labelled "Close chat") | ✓ VERIFIED | No JS/markup change; click handler unchanged; TerminalChatHost.test.tsx Test 2 verifies aria-label flips to "Close chat" after toggle; HubInteractiveModal.test.tsx 16/16 still pass. |
| 3 | When the drawer is closed, the toggle stays at right:12px — no visual change to closed state | ✓ VERIFIED | `chatToggleOverlap.test.ts` assertion (c) passes: base `.hub-modal__chat-toggle` rule still contains `right: 12px`; source confirmed in style.css:6032. |
| 4 | The fix is a single additive CSS rule — no markup or JS change | ✓ VERIFIED | Only style.css, chatToggleOverlap.test.ts (new), TESTING.md modified in Plan 01. Confirmed by commit 415be4b7 (`git log` shows CSS + test + docs only). No .tsx files changed. |
| 5 | Unscoped rule also corrects the identical latent overlap on the web-share surface (parity-positive) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS rule is unscoped (no `.hub-modal__body--interactive` parent). WebShareSessionView.tsx uses same class names — structural match confirmed. Visual correction on web-share surface requires live browser verification. |

#### Plan 02 Truths (CHAT-PARITY-01)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 6 | From a live session in a direct terminal TAB, a chat toggle is present and toggles a working ChatPanel drawer | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | TerminalChatHost.tsx exists + wired in App.tsx:1711; behavioral tests 11/11 pass (toggle present, open/close toggles ChatPanel `open` prop). Live relay + PTY required for "working drawer" confirmation (M-30). |
| 7 | Tab drawer is overlay mode (D-02): position:absolute over terminal; PTY is NEVER resized on open/close | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `.terminal-chat-host { position: relative }` in style.css:513 provides containing block; unscoped `.chat-panel` (position:absolute) applies inside. Test 3 verifies `isActive` forwarded from props (not chatOpen — D-02 invariant). Actual PTY no-resize requires live PTY + WebView (M-30). |
| 8 | Cross-surface parity (release-blocking, upstream PARITY-01): GUI tab now matches Hub modal and web-share | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | TerminalChatHost imports same ChatPanel/ChatBadge/ChatBubbleLeftRightIcon. Structural parity verified; HubInteractiveModal.test.tsx 16/16 + WebShareSessionView tests pass (no regression). Live visual cross-surface parity requires M-30. |
| 9 | ChatPanel REUSED not forked: tab imports existing ChatPanel/ChatBadge; Hub-modal and web-share paths unchanged | ✓ VERIFIED | TerminalChatHost.tsx:7-8 imports `ChatPanel` from `./ChatPanel` and `ChatBadge` from `./ChatBadge`. No ChatPanel logic duplicated in the file. HubInteractiveModal.test.tsx 16/16 PASS (no regression). |
| 10 | Terminal tab keeps full terminal behavior: TerminalChatHost forwards every TerminalPanel prop the tab previously passed | ✓ VERIFIED | TerminalChatHost.tsx forwards: sessionId, isActive, relayPort, fontSize, onFontSizeChange, theme, pluginConfig, onWebGLContextLost, onRegisterSaver, onProgressChange, remote. Test 3 (11 tests, 3 prop-forwarding assertions) pass. `tsc --noEmit` 0 errors. |
| 11 | Per-tab StatusBar remains visible and is NOT overlaid by the drawer (host wraps only the terminal) | ✓ VERIFIED | App.tsx:1733-1740: StatusBar and ExitCountdownBanner are siblings of TerminalChatHost inside `.terminal-wrapper`, not children of the host. TerminalChatHost.tsx does not render StatusBar or ExitCountdownBanner. |

**Score:** 6/11 truths verified (5 present + wired, behavior not exercised by tests)

### Deferred Items

None — all truths are either verified or routed to human verification. No truths are addressed in later phases.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/style.css` | Additive sibling-combinator rule `.chat-panel--open ~ .hub-modal__chat-toggle { right: 372px }` | ✓ VERIFIED | Rule confirmed at line 6059; base rule `right: 12px` at line 6032 preserved. `.terminal-chat-host` containing-block rule at line 513. Reduced-motion entry at line 6965. |
| `frontend/src/components/Hub/chatToggleOverlap.test.ts` | Vitest source-gate asserting relocation rule exists (3 assertions) | ✓ VERIFIED | File exists (1703 bytes). 3/3 tests pass: (a) selector present, (b) right:372px in relocation block, (c) right:12px in base block. Uses `readFileSync` pattern consistent with `style.hub.test.ts`. |
| `TESTING.md` | Section 2 manifest entry + Section 4 CHAT-FIX-01 + CHAT-PARITY-01 traceability rows + Section 5 M-29 + M-30 | ✓ VERIFIED | All 4 entries confirmed. `bash tests/check-traceability-paths.sh` exits 0. |
| `frontend/src/components/Hub/TerminalChatHost.tsx` | Overlay host component (TerminalPanel + always-mounted ChatPanel + toggle) | ✓ VERIFIED | File exists (5030 bytes, 122 lines). Substantive: imports ChatPanel, ChatBadge, TerminalPanel; owns chatOpen/unread state; renders in correct DOM order (ChatPanel before toggle). |
| `frontend/src/components/Hub/TerminalChatHost.test.tsx` | Behavioral vitest: toggle present, click toggles ChatPanel open prop, props forwarded, DOM order | ✓ VERIFIED | File exists (7455 bytes). 11/11 tests pass across 4 describe blocks. |
| `frontend/src/App.tsx` | Imports TerminalChatHost; tabs.map renders it in place of bare TerminalPanel; StatusBar as sibling | ✓ VERIFIED | Import at line 59. `<TerminalChatHost` at line 1711 with all props. StatusBar at line 1737 as sibling outside host. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `.chat-panel--open` class | `.hub-modal__chat-toggle` relocation | CSS general-sibling combinator | ✓ WIRED | Rule confirmed in style.css:6059. ChatPanel renders BEFORE the toggle in TerminalChatHost (DOM order test passes). |
| `TerminalChatHost` props | `TerminalPanel` props | All props forwarded through TerminalChatHost | ✓ WIRED | onWebGLContextLost / onRegisterSaver / onProgressChange forwarding verified by Test 3. isActive forwarded from props (not chatOpen — D-02). tsc --noEmit 0 errors. |
| `App.tsx` terminal tab | `TerminalChatHost` | Import + JSX replacement | ✓ WIRED | Import at App.tsx:59; `<TerminalChatHost` at App.tsx:1711. |
| `TerminalChatHost` | `StatusBar` (stays outside) | StatusBar as sibling in `.terminal-wrapper` | ✓ WIRED | App.tsx:1737 shows StatusBar as sibling; TerminalChatHost.tsx does not render StatusBar. |

### Data-Flow Trace (Level 4)

Not applicable. This phase adds CSS positioning logic and a UI host component that delegates data flow to the existing (unchanged) ChatPanel and TerminalPanel implementations. No new data-fetching paths or dynamic data rendering were introduced. TerminalChatHost owns only `chatOpen/unreadCount/hasMention` state — no network calls or data sources.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| CHAT-FIX-01 CSS rule presence (3 source-gate assertions) | `pnpm -C frontend exec vitest run src/components/Hub/chatToggleOverlap.test.ts` | 3/3 PASS | ✓ PASS |
| CHAT-PARITY-01 TerminalChatHost behavioral tests (11 tests across 4 describe blocks) | `pnpm -C frontend exec vitest run src/components/Hub/TerminalChatHost.test.tsx` | 11/11 PASS | ✓ PASS |
| style.hub.modal.test.ts (pre-existing test fixed by commit 1fec183c) | `pnpm -C frontend exec vitest run src/components/__tests__/style.hub.modal.test.ts` | 18/18 PASS | ✓ PASS |
| Traceability paths | `bash tests/check-traceability-paths.sh` | OK: all traceability paths exist | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| CHAT-FIX-01 | 158-01 | Toggle/Send overlap bug fix | ✓ SATISFIED | CSS rule + source-gate verified; M-29 registered for visual confirmation |
| CHAT-PARITY-01 | 158-02 | Terminal-tab chat affordance; upstream PARITY-01, D-02 | ✓ SATISFIED (automated) / ⚠️ HUMAN NEEDED (visual/live) | TerminalChatHost wired + 11 behavioral tests pass; M-30 registered for live UAT |
| PARITY-01 (upstream) | REQUIREMENTS.md Phase 155 | Cross-surface chat parity (release-blocking) | ⚠️ EXTENDED BY PHASE 158 | PARITY-01 was completed in Phase 155 for Hub modal + web-share. Phase 158 extends it to the terminal tab. REQUIREMENTS.md traceability table does not yet list Phase 158 — see note below. |

**Requirements coverage note:** CHAT-FIX-01 and CHAT-PARITY-01 are Phase 158-specific identifiers that appear in ROADMAP.md (Phase 158 section) and TESTING.md (Section 4 traceability rows) but are NOT present in REQUIREMENTS.md's traceability table, which ends at Phase 157 (VIEW-01..05). This is a documentation gap — Phase 158 was added as a polish/bug-fix pass after the v4.1 requirements baseline was locked. CHAT-PARITY-01 is downstream of PARITY-01 (REQUIREMENTS.md Phase 155); CHAT-FIX-01 is a new UAT-discovered defect with no pre-declared requirement. Neither gap blocks the implementation goal, but REQUIREMENTS.md should be updated to reflect Phase 158's contribution.

### Anti-Patterns Found

No TBD, FIXME, or XXX markers found in any files modified by Phase 158. No stubs detected. No empty handlers.

The pre-existing `style.hub.modal.test.ts` fragility (where Plan 158-01's new `@media (prefers-reduced-motion: reduce)` block at the end of style.css caused `lastIndexOf` to match the wrong block) was identified in the 158-02 SUMMARY and subsequently fixed by commit `1fec183c`. The test now uses `ruleMatch.find((r) => r.includes('animation: none'))` (selector-targeted matching), is green (18/18 pass), and is no longer a failure.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No debt markers found | — | — |

### Human Verification Required

#### 1. M-29 — CHAT-FIX-01: Toggle/Send Non-Overlap Visual Check

**Test:** Open a live session in the Hub interactive modal. Open the chat drawer. Observe whether the chat toggle button sits clear of (does not overlap/obscure) the composer's Send/Inject button. Then click the toggle and confirm the drawer closes.

**Expected:** The toggle button is positioned visually left of the drawer's left edge (the drawer is 360px wide; the toggle should appear at ~372px from the right edge of the modal, leaving 12px clear of the drawer). The composer's Send/Inject button is fully visible and not obscured. Clicking the (relocated) toggle closes the drawer.

**Why human:** JSDOM performs no layout. The source-gate in `chatToggleOverlap.test.ts` proves the CSS rule `right: 372px` is present and the base `right: 12px` is preserved, but cannot measure rendered pixel geometry. A live browser with a real layout engine is required.

#### 2. M-30 — CHAT-PARITY-01: Terminal-Tab Chat Affordance (Overlay, No PTY Resize, StatusBar, Cross-Surface Parity)

**Test:** Open a live session in a direct terminal tab (not via the Hub card modal). Confirm: (a) a chat toggle button is present in the terminal tab; (b) clicking it opens a ChatPanel drawer in overlay mode (the terminal occupies the full tab, the drawer sits on top of the right portion); (c) the terminal does NOT reflow or garble when the drawer opens/closes; (d) the StatusBar remains visible below the drawer; (e) an unread badge accrues on the toggle while the drawer is closed and a message arrives. Cross-surface parity: compare the tab chat affordance to the Hub modal and web-share surfaces — they should behave identically.

**Expected:** Chat toggle present in terminal tab. Overlay drawer opens without PTY resize (no terminal reflow/garble). StatusBar visible below drawer. Unread badge shows count while drawer closed. Affordance behavior matches Hub modal and web-share (PARITY-01 / release-blocking).

**Why human:** The TerminalChatHost behavioral tests (11/11 PASS) exercise toggle/prop-forwarding in JSDOM with mocked TerminalPanel and ChatPanel. Confirming overlay geometry (position:absolute behavior), PTY no-resize invariant (no real PTY in JSDOM), StatusBar visibility, and cross-surface visual parity all require a live daemon + PTY + WebView render.

---

## Gaps Summary

No implementation gaps found. All artifacts exist, are substantive, and are wired correctly. Both source-gate and behavioral tests pass. The 5 PRESENT_BEHAVIOR_UNVERIFIED truths are runtime/visual behaviors that JSDOM cannot exercise — these are correctly routed to manual UAT items M-29 and M-30 (per the context note: "intentionally deferred to manual UAT items M-29 and M-30 — these are expected human_verification items, not gaps").

---

_Verified: 2026-06-27T18:10:00Z_
_Verifier: Claude (gsd-verifier)_
