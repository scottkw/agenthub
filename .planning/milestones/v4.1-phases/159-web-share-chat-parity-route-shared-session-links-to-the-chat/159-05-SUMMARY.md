---
phase: 159-web-share-chat-parity-route-shared-session-links-to-the-chat
plan: 05
gap_closure: true
status: complete
requirements: [WEBCHAT-06]
completed: 2026-06-27
---

# Plan 159-05 Summary — Chat author header truncation (WEBCHAT-06)

## Self-Check: PASSED

## What was built

Cosmetic chat-rendering fix surfaced during 159 live UAT. In the narrow chat panel, a
long author name (`kens-personal-macbook-air` — the tailnet hostname used as the fallback
identity before an alias is set) wrapped character-by-character into a tall 3-line stack
next to the avatar, while the raw `(nodekey:…)` overflowed the panel edge.

Root cause: `.chat-msg__header` is a non-wrapping flex row with two long text items
(`.chat-msg__alias` + `.chat-msg__tailnet-id`) and no truncation, so the name wrapped and
the nodekey overflowed.

## Change (CSS-only)

- `frontend/src/style.css`:
  - `.chat-msg__header` — `min-width: 0` so flex children can shrink.
  - `.chat-msg__alias` — `white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    min-width: 0; flex-shrink: 1` → long names truncate on one line.
  - `.chat-msg__tailnet-id` — same truncation + `flex-shrink: 1000` so the long nodekey
    gives up space first, preserving the alias.
- `frontend/src/components/__tests__/style.hub.test.ts` — +3 source-gate tests asserting
  the truncation properties (project convention; the user is colorblind → verify at source).

## Verification

- vitest: 85 pass in style.hub.test.ts (incl. 3 new).
- Live (fixture + dev-browser, long name injected): `.chat-msg__alias` height = 16px
  (one line; was 3-line wrap), computed `white-space: nowrap`, `text-overflow: ellipsis`.

## Not changed (reported to user)

- Empty band at the bottom of the window: the height chain (html/body/#root/.app/.app__row/
  .app__content/.terminal-container) is correct (all 100%/flex), so the gap is the xterm
  terminal quantizing to whole character rows — pre-existing and global (present in desktop
  screenshots too), not a chat/web-share issue. Deferred pending user decision.

## Files
- created: 159-05-SUMMARY.md
- modified: frontend/src/style.css, frontend/src/components/__tests__/style.hub.test.ts, TESTING.md, .planning/REQUIREMENTS.md
