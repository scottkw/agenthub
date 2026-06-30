---
phase: 154-desktop-chat-ui
plan: "06"
subsystem: frontend-chat-ui
tags: [chat, composer, inject, mention, overlay, tdd, testing]
dependency_graph:
  requires: [154-01, 154-02, 154-04, 154-05]
  provides: [composer, overlay-drawer, unread-badge, testing-registration]
  affects: [ChatPanel.tsx, HubInteractiveModal.tsx, SessionCard.tsx, style.css, TESTING.md]
tech_stack:
  added: [react-textarea-autosize (composer auto-grow)]
  patterns:
    - draftRef liveRef pattern (600ms inject timer reads non-stale draft)
    - clientRef pattern (RelayClient accessible from event handlers without stale closure)
    - D-02 overlay mode (ChatPanel position:absolute over terminal, no PTY resize on toggle)
    - D-09 always-mounted ChatPanel (unread accrues while drawer is closed)
    - D-08 press-and-hold inject (600ms setTimeout + setPointerCapture + CSS scaleX fill)
    - Pitfall 7 guard (Enter ALWAYS routes to sendChat, NEVER to inject)
key_files:
  modified:
    - frontend/src/components/Hub/ChatPanel.tsx
    - frontend/src/components/Hub/ChatPanel.test.tsx
    - frontend/src/components/Hub/HubInteractiveModal.tsx
    - frontend/src/components/Hub/HubInteractiveModal.test.tsx
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/SessionCard.test.tsx
    - frontend/src/style.css
    - TESTING.md
decisions:
  - Enter always routes to sendChat never to inject (Pitfall 7 — strictly separate code paths, no shared logic)
  - draftRef.current synced inline during render for inject timer closure (avoids stale state in 600ms timeout)
  - comment text must not contain CSS class names used in source-inspection tests (hub-modal__terminal-col comment fix)
  - Overlay mode confirmed: D-02 position:absolute drawer over terminal, no flex push, no PTY resize
metrics:
  duration: "~70 minutes (including context reload from summary)"
  completed: "2026-06-26"
  tasks_completed: 3
  files_modified: 8
status: complete
---

# Phase 154 Plan 06: Composer + Overlay Integration + TESTING.md Summary

Composer, HubInteractiveModal overlay integration, SessionCard unread badge, and full Phase 154 TESTING.md registration via TDD (RED → GREEN for both TDD tasks).

## What Was Built

### Task 1: ChatPanel Composer (TDD)

**RED commit:** `7875f2d5` — 5 failing tests for inject/composer/sec-03 groups

**GREEN commit:** `44a0c60f` — full composer implementation:

- `react-textarea-autosize` (minRows=1, maxRows=6) replaces the placeholder slot
- `handleTextareaKeyDown`: Enter (no Shift, no popover) → `clientRef.current?.sendChat(draftRef.current)` → clear draft; Shift+Enter → default newline (no preventDefault); ArrowUp/Down navigates popover; Enter in popover selects; Escape/Tab close popover
- `handleDraftChange`: detects `@` trigger (lastIndexOf, no whitespace in fragment) → opens MentionPopover; mirrors draft to `draftRef.current` inline during render (liveRef pattern for inject timer closure)
- Press-and-hold inject (D-08): `handleInjectPointerDown` starts 600ms `setTimeout` that calls `sendSessionInject(draftRef.current)` + clears draft; `resetInjectHold` clears timer on pointerUp/pointerCancel
- CRITICAL guard: Enter NEVER reaches the inject path — `handleTextareaKeyDown` and the inject pointer handler are strictly separate code paths (Pitfall 7)
- Inject button CSS: `::before` fill animation `scaleX(0→1)` over 600ms; `--holding` modifier flips label color; `prefers-reduced-motion` disables animation while timer still fires
- SEC-03: no `rehype-raw` added; existing `rehype-sanitize` in ChatMessage guards XSS

### Task 2: HubInteractiveModal Overlay + SessionCard Badge (TDD)

**RED commit:** `fd941b46` — 7 failing overlay tests + SessionCard ChatBadge tests scaffolded

**GREEN commit:** `63e46d3f` — overlay implementation:

**HubInteractiveModal.tsx:**
- `chatOpen`, `unreadCount`, `hasMention` state added
- `hub-modal__body--interactive` becomes `position:relative` containing block
- TerminalPanel remains full-bleed with NO column wrapper — `isActive={open}` (modal-open prop) unchanged regardless of `chatOpen` (D-02: no PTY resize on chat toggle)
- ChatPanel always-mounted with `open={chatOpen}` (D-09: unread accrues while closed)
- `hub-modal__chat-toggle` button: `ChatBubbleLeftRightIcon` + `ChatBadge`; aria-label flips "Open chat"/"Close chat"

**SessionCard.tsx:**
- `unreadCount?` and `hasChatMention?` props added
- `ChatBadge` rendered in card header (count=0 renders null, no DOM node)

**style.css:**
- `.hub-modal__body--interactive { position: relative }` override added
- `.hub-modal__body--interactive .chat-panel` overlay rules: `position:absolute; top/right/bottom:0; width:360px; z-index:5; translateX(100%)` → `translateX(0)` when open
- `.hub-modal__chat-toggle`: 44px min touch target, bottom-right float, `border-radius:22px` pill shape

### Task 3: TESTING.md Registration

**Commit:** `c04aed0a`

- Section 2: Go 362→365 (+3 Phase 154-01 Go files), vitest 120→125 (+5 Phase 154 files), total 490→498
- Section 4: 14 new traceability rows covering CHAT-01..04, MENTION-01, NOTIF-01, SEC-03 with Phase 154 test files
- Section 5: Category L (M-20..M-23) — inject animation, overlay no-resize, day separator scroll, Enter-never-injects live verification
- `bash tests/check-traceability-paths.sh` exits 0 (all paths verified on disk)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Comment text contained CSS class name checked by source-inspection test**

- **Found during:** Task 2 GREEN, first test run
- **Issue:** The comment `"NO hub-modal__terminal-col wrapper"` in HubInteractiveModal.tsx contained the string `hub-modal__terminal-col`, causing the source-inspection test `expect(raw).not.toContain('hub-modal__terminal-col')` to fail
- **Fix:** Rephrased comment to `"no column-shrink wrapper"` — preserves the design intent without embedding the prohibited string
- **Files modified:** `frontend/src/components/Hub/HubInteractiveModal.tsx`
- **Commit:** included in `63e46d3f`

## TDD Gate Compliance

Both TDD tasks completed the full RED → GREEN cycle:

| Task | RED commit | GREEN commit | Gate |
|------|-----------|-------------|------|
| Task 1 (Composer) | `7875f2d5` — 5 failing tests | `44a0c60f` — all 39 pass | PASS |
| Task 2 (Overlay) | `fd941b46` — 5 failing overlay tests | `63e46d3f` — all 113 pass | PASS |

## Final Verification

- `pnpm exec vitest run` — **125 files / 2053 tests, all pass**
- `pnpm exec tsc --noEmit` — **clean**
- `go test -race -short ./internal/relay/... ./internal/webserver/...` — **ok** (relay 4.3s, webserver 9.9s)
- `bash tests/check-traceability-paths.sh` — **exit 0** (OK: all traceability paths exist)
- `grep -rn "rehype-raw" frontend/src` — **no chat-code matches** (SEC-03 satisfied)

## Self-Check: PASSED

All files exist and all commits are present in git log.
