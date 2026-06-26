---
phase: "154"
plan: "04"
subsystem: frontend
status: complete
tags: [chat-ui, badge, mention-popover, colorblind-safe, accessibility, tdd]
dependency_graph:
  requires: ["154-03"]
  provides: [ChatBadge, MentionPopover]
  affects: [SessionCard, HubInteractiveModal, ChatPanel]
tech_stack:
  added: []
  patterns: [BEM CSS, TDD (RED/GREEN), createRoot+act test pattern, global keydown listener]
key_files:
  created:
    - frontend/src/components/Hub/ChatBadge.tsx
    - frontend/src/components/Hub/ChatBadge.test.tsx
    - frontend/src/components/Hub/MentionPopover.tsx
    - frontend/src/components/Hub/MentionPopover.test.tsx
  modified:
    - frontend/src/style.css
decisions:
  - "ChatBadge uses role=status (not role=img) for live region semantics; badge is a sibling of the toggle button"
  - "MentionPopover accepts style/className passthrough for parent-controlled positioning (bottom:100% is CSS default; parent can override)"
  - "Escape listener registered in MentionPopover useEffect — closes popover even when focus is in the textarea (SessionCard keyboard-dismissable pattern)"
  - "Participant activeIndex mapping: 0=@session, i+1=participant[i] — index contract documented in JSDoc for ChatPanel wiring in plan 154-06"
metrics:
  duration: "~4 minutes"
  completed: "2026-06-26"
  tasks_completed: 2
  files_created: 4
  files_modified: 1
---

# Phase 154 Plan 04: ChatBadge + MentionPopover Summary

ChatBadge and MentionPopover built as self-contained, unit-tested components ready for ChatPanel wiring in plan 154-06. ChatBadge encodes the NOTIF-01/D-10 badge contract (@ glyph as non-color mention signal); MentionPopover encodes MENTION-01/D-07 (@session always pinned first regardless of filter).

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | ChatBadge — count + colorblind-safe @mention glyph state | `40497a7b` | ChatBadge.tsx, ChatBadge.test.tsx, style.css |
| 2 | MentionPopover — @session pinned, filterable, keyboard-navigable | `9d943603` | MentionPopover.tsx, MentionPopover.test.tsx |

## What Was Built

### ChatBadge (NOTIF-01, D-10)

`ChatBadge` is an 18px filled circle badge component with two states:

- **Normal unread** (`hasMention=false`): renders the count number as text. `aria-label="N unread message(s)"`.
- **@mention unread** (`hasMention=true`): renders `@` glyph replacing the count. `aria-label="N unread messages, including a mention"`. Also applies `chat-badge--mention` class.
- **count=0**: renders `null` (no DOM node).

The `@` glyph is the **primary non-color channel** for the mention state (D-10 mandate). The `--hub-accent` background color is reinforcement only. The `aria-label` containing "mention" is a second non-color channel for screen readers.

### MentionPopover (MENTION-01, D-07)

`MentionPopover` is a `role="listbox"` autocomplete popover with two sections:

- **Section 1 ("Agent")**: @session row with `CommandLineIcon` + "Inject into terminal" description. Always first, always rendered — `filter` prop never affects this row (D-07 invariant).
- **Divider** (`mention-popover__divider`).
- **Section 2 (participants)**: `participants` prop filtered by `filter` (case-insensitive substring on alias).

Active item is controlled by `activeIndex` prop (0 = @session, i+1 = participant[i]). The active item receives `mention-popover__item--active` class and `aria-selected=true`. Arrow/Enter index management is owned by the ChatPanel composer (plan 154-06).

Escape closes the popover via a global `keydown` listener installed in `useEffect` (SessionCard keyboard-dismissable pattern from PATTERNS.md).

### Security (T-154-07, T-154-08)

- T-154-07: @session is in its own "Agent" section structurally separated from participant rows — a participant cannot alias as "@session" and be mistaken for the agent target.
- T-154-08: All aliases render as React text nodes (auto-escaped, no `dangerouslySetInnerHTML`).

### CSS Added to style.css

Two new BEM blocks added:

- `.chat-badge`, `.chat-badge--mention` — 18px filled circle badge.
- `.mention-popover`, `.mention-popover__section-label`, `.mention-popover__item`, `.mention-popover__item--session`, `.mention-popover__item--active`, `.mention-popover__item--participant`, `.mention-popover__alias`, `.mention-popover__desc`, `.mention-popover__divider`, `.mention-popover__session-icon`, `.mention-popover__avatar`, `.mention-popover__tailnet-id` — full BEM set per UI-SPEC §8.

## Test Results

```
Test Files  2 passed (2)
      Tests  10 passed (10)
```

- `ChatBadge.test.tsx`: 4 tests — null at 0, count text, singular aria-label, @ glyph + aria-label + class at mention.
- `MentionPopover.test.tsx`: 6 tests — sections + participants, filter never removes @session, activeIndex highlighting, @session click, alice click, Escape closes.

`pnpm exec tsc --noEmit`: clean.

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED (failing tests) | verified via pnpm test run failing | PASS |
| GREEN (passing impl) | `40497a7b`, `9d943603` | PASS |
| REFACTOR | No refactor needed (clean on first pass) | N/A |

## Deviations from Plan

**1. [Rule 3 - Blocking] @testing-library/react not installed**
- **Found during:** Task 1 (ChatBadge.test.tsx RED phase)
- **Issue:** The test file initially imported from `@testing-library/react` which is not in the project's package.json.
- **Fix:** Rewrote tests to use `createRoot` + `act` from `react-dom/client` — the same pattern used by all other test files in the codebase (ChatMessage.test.tsx, ChatDaySeparator.test.tsx, etc.).
- **Files modified:** ChatBadge.test.tsx (rewritten), MentionPopover.test.tsx (written with correct pattern from the start)
- **Commit:** part of `40497a7b`

## Known Stubs

None. Both components are self-contained presentational components that receive all data via props. No stubs, no hardcoded empty values.

## Threat Flags

None. No new trust boundaries introduced beyond what the plan's threat model covers (T-154-07 and T-154-08 both mitigated in implementation).

## Self-Check: PASSED

- [x] `frontend/src/components/Hub/ChatBadge.tsx` — exists
- [x] `frontend/src/components/Hub/ChatBadge.test.tsx` — exists
- [x] `frontend/src/components/Hub/MentionPopover.tsx` — exists
- [x] `frontend/src/components/Hub/MentionPopover.test.tsx` — exists
- [x] `frontend/src/style.css` — modified with chat-badge + mention-popover BEM classes
- [x] Commit `40497a7b` — exists (feat: ChatBadge)
- [x] Commit `9d943603` — exists (feat: MentionPopover)
- [x] 10/10 tests green
- [x] tsc --noEmit clean
