---
phase: 154-desktop-chat-ui
plan: "03"
subsystem: frontend/chat-ui
status: complete
tags: [chat, markdown, xss, colorblind-safe, tdd]
completed: "2026-06-26"
duration: "9 minutes"

dependency_graph:
  requires: ["154-02"]
  provides: ["ChatMessage", "ChatDaySeparator", "formatDaySeparator", "tailnetIdToHue", "formatHHMM", "chat-msg__*", "chat-day-sep__*"]
  affects: ["154-05", "154-06"]

tech_stack:
  added: []
  patterns:
    - "react-markdown v10 (no className prop) wrapped in div.chat-msg__body"
    - "rehype-sanitize default schema — belt-and-suspenders against stored unsafe content"
    - "deterministic hue hash (polynomial hash % 360) for avatar color from tailnet ID"
    - "Three-signal colorblind-safe @mention: accent bar (::before), background tint, @you chip glyph"
    - "TDD: RED (failing test commit) → GREEN (implementation commit)"

key_files:
  created:
    - frontend/src/components/Hub/ChatMessage.tsx
    - frontend/src/components/Hub/ChatMessage.test.tsx
    - frontend/src/components/Hub/ChatDaySeparator.tsx
    - frontend/src/components/Hub/ChatDaySeparator.test.tsx
  modified:
    - frontend/src/style.css

decisions:
  - "React import removed: react-jsx transform handles JSX automatically; noUnusedLocals requires dropping bare import React"
  - "chat-day-sep::no-sticky: position:sticky intentionally omitted from ChatDaySeparator — parent virtualizer applies it to avoid transform vs. sticky conflict (Pitfall 1)"
  - "rehype-sanitize default schema: no tightening or loosening; default GH schema is sufficient + react-markdown v10 raw-HTML disabled"

metrics:
  duration: "9 minutes"
  completed: "2026-06-26"
  tasks_completed: 2
  files_changed: 5
  tests_added: 38
  tests_passing: 38
---

# Phase 154 Plan 03: Chat Message Row Components Summary

Pure render components for the chat thread — ChatMessage.tsx and ChatDaySeparator.tsx — with SEC-03 safe Markdown, colorblind-safe @mention treatment (three independent signals), session-inject system row, and day separator formatting, all proven by 38 unit tests.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| 1 (RED) | ChatMessage failing tests | 74652c3a | ChatMessage.test.tsx |
| 1 (GREEN) | ChatMessage implementation + CSS | 4cf8dfc1 | ChatMessage.tsx, style.css |
| 2 (RED) | ChatDaySeparator failing tests | 83067c5b | ChatDaySeparator.test.tsx |
| 2 (GREEN) | ChatDaySeparator implementation | 04e696ef | ChatDaySeparator.tsx |
| fix | Remove unused React import (tsc noUnusedLocals) | 22a842ef | all 4 files |

## Verification Results

```
pnpm test run src/components/Hub/ChatMessage.test.tsx src/components/Hub/ChatDaySeparator.test.tsx
  Test Files: 2 passed
  Tests:      38 passed

pnpm exec tsc --noEmit
  TSC PASS (clean)

grep -rn "rehype-raw" frontend/src/components/Hub/
  PASS: no rehype-raw in Hub components (SEC-03 guard)
```

## Requirements Addressed

| Requirement | Coverage |
|-------------|----------|
| CHAT-02 | Consecutive collapse: isFirstInGroup=false omits header, applies chat-msg__body--consecutive |
| CHAT-04 | ChatDaySeparator with formatDaySeparator (Today/Yesterday/locale-short) |
| NOTIF-02 | @mention-of-me three-signal: accent bar (chat-msg--mention::before), tint, @you chip |
| SEC-03 | react-markdown + rehype-sanitize; raw HTML plugin absent; two XSS payload tests pass |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Unused React import caused TypeScript noUnusedLocals error**
- **Found during:** Post-implementation TSC clean run
- **Issue:** `import React from 'react'` in all four files triggered TS6133 under `noUnusedLocals: true`. With `jsx: react-jsx`, the JSX transform does not require the React default import.
- **Fix:** Removed the bare `import React from 'react'` from ChatMessage.tsx, ChatMessage.test.tsx, ChatDaySeparator.tsx, ChatDaySeparator.test.tsx.
- **Files modified:** All four new files
- **Commit:** 22a842ef

None — plan executed as written apart from the TypeScript import fix above.

## SEC-03 Compliance Evidence

The two XSS assertions in ChatMessage.test.tsx verify:
1. A script-tag payload in `content` produces `container.querySelector('script') === null`
2. An img event-handler payload in `content` produces zero `[onerror]` attribute matches

`rehype-raw` is absent from all Hub component imports. The raw-HTML rehype plugin is not installed and was not added.

## TDD Gate Compliance

| Gate | Commit | Type |
|------|--------|------|
| RED (Task 1) | 74652c3a | `test(154-03)` |
| GREEN (Task 1) | 4cf8dfc1 | `feat(154-03)` |
| RED (Task 2) | 83067c5b | `test(154-03)` |
| GREEN (Task 2) | 04e696ef | `feat(154-03)` |

Both tasks followed the RED → GREEN commit sequence. No REFACTOR pass was needed.

## Known Stubs

None. Components are pure presentational renders with no data source wiring (by design — they receive props from the parent ChatPanel, built in plan 154-05).

## Threat Flags

None. This plan implements SEC-03 mitigations (safe Markdown). No new network endpoints, auth paths, or trust boundaries introduced.

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| ChatMessage.tsx exists | FOUND |
| ChatDaySeparator.tsx exists | FOUND |
| ChatMessage.test.tsx exists | FOUND |
| ChatDaySeparator.test.tsx exists | FOUND |
| 74652c3a commit exists | FOUND |
| 4cf8dfc1 commit exists | FOUND |
| 83067c5b commit exists | FOUND |
| 04e696ef commit exists | FOUND |
| 22a842ef commit exists | FOUND |
| 38/38 tests pass | PASS |
| TSC clean | PASS |
| no rehype-raw in Hub | PASS |
