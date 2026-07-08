---
phase: 173-share-modal-three-tab-segmented-redesign
plan: 04
subsystem: ui
tags: [react, share-modal, ui, qr, clipboard]

# Dependency graph
requires:
  - phase: 173-01
    provides: ".share-linkcard* CSS classes (dark + light theme tokens, --hub-* only, no inline hex)"
  - phase: 173-02
    provides: "CodeDisplay hoisted to frontend/src/components/SessionShare/shared.tsx"
provides:
  - "ShareLinkCard — the one reusable link row (title · truncated URL · Copy/Open/QR · join code · scope description beneath) replacing the four ad-hoc hand-laid link rows in SessionSharePanel.tsx"
  - "Per-card independent copied/QR-open/QR-b64/QR-error state (encapsulation improvement over the panel's single shared qrError slot)"
affects: [173-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ShareLinkCard consumes joinURLFor(url, code) reused verbatim from SessionSharePanel.tsx — QR always encodes the join-code exchange URL, never the raw capability token"

key-files:
  created:
    - frontend/src/components/SessionShare/ShareLinkCard.tsx
    - frontend/src/components/__tests__/ShareLinkCard.test.tsx
  modified: []

key-decisions:
  - "joinURLFor copied verbatim into ShareLinkCard.tsx rather than exported/shared from SessionSharePanel.tsx — SessionSharePanel.tsx is slated for replacement by the plan-06 shell, so adding a new cross-import there would create a dependency destined to be deleted"
  - "QR/Copy/Open buttons reuse the existing .daemon-panel__btn class (unchanged) inside the new .share-linkcard__actions wrapper — no new button styling introduced"

patterns-established:
  - "One ShareLinkCard component renders the identical layout for every access tier (Tailnet Read-Only, Tailnet Full Access, Internet public URL) — callers pass title/url/code/description only"

requirements-completed: [SM-06]

coverage:
  - id: D1
    description: "ShareLinkCard renders title, CSS-truncated URL with full URL in title=, and Copy/Open/QR action buttons"
    requirement: "SM-06"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareLinkCard.test.tsx#ShareLinkCard — structure contract (SM-06) > renders the title text"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareLinkCard.test.tsx#ShareLinkCard — structure contract (SM-06) > renders .share-linkcard__url with title= equal to the full URL (CSS-truncated, not JS-truncated)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareLinkCard.test.tsx#ShareLinkCard — structure contract (SM-06) > renders Copy, Open, and QR action buttons"
        status: pass
    human_judgment: false
  - id: D2
    description: "Join code (shared CodeDisplay) and scope description render inside .share-linkcard__desc, positioned directly beneath the link — fixes the orphaned scope paragraph defect"
    requirement: "SM-06"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareLinkCard.test.tsx#ShareLinkCard — structure contract (SM-06) > renders the join code (CodeDisplay) and the scope description inside .share-linkcard__desc, positioned after the link"
        status: pass
    human_judgment: false
  - id: D3
    description: "Copy button copies the URL; Open button calls BrowserOpenURL with the URL"
    requirement: "SM-06"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareLinkCard.test.tsx#ShareLinkCard — action wiring > clicking Copy calls ClipboardSetText with the full URL"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareLinkCard.test.tsx#ShareLinkCard — action wiring > clicking Open calls BrowserOpenURL with the URL"
        status: pass
    human_judgment: false
  - id: D4
    description: "QR button fetches GetCapabilityQRCode targeting the join-code exchange URL (joinURLFor), never the raw capability token — Information Disclosure mitigation T-173-02"
    requirement: "SM-06"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/ShareLinkCard.test.tsx#ShareLinkCard — action wiring > clicking QR calls GetCapabilityQRCode with a target containing /join?code= (join-URL, not the raw capability token)"
        status: pass
    human_judgment: false

# Metrics
duration: 4min
completed: 2026-07-08
status: complete
---

# Phase 173 Plan 04: ShareLinkCard Summary

**One reusable `ShareLinkCard` component (title · CSS-truncated URL · Copy/Open/QR · join code · scope description directly beneath) replacing the four ad-hoc hand-laid link rows in `SessionSharePanel.tsx`, with the QR always encoding the join-code exchange URL rather than the raw capability token.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-07-08T07:07:00-05:00 (approx)
- **Completed:** 2026-07-08T07:09:30-05:00
- **Tasks:** 2
- **Files modified:** 2 (both new)

## Accomplishments
- Built `ShareLinkCard.tsx`: a single component rendering `.share-linkcard` with `.share-linkcard__top` (title, CSS-truncated URL with full URL in `title=`, Copy/Open/QR actions), `.share-linkcard__join` (shared `CodeDisplay`), and `.share-linkcard__desc` (scope description) — the unified shape for all three tiers (Tailnet Read-Only, Tailnet Full Access, Internet public)
- Each card instance owns independent `copied`/`qrOpen`/`qrB64`/`qrError` state — a deliberate small encapsulation improvement over `SessionSharePanel.tsx`'s single shared `qrError` slot (RESEARCH Assumption A1)
- QR target derivation (`joinURLFor`) reused verbatim from `SessionSharePanel.tsx` — encodes `https://host/join?code=<code>`, never the capability token, closing the Information-Disclosure threat (T-173-02)
- Migrated the old inline `style={{color:'#9aa5ce'}}` scope-paragraph styling into the `.share-linkcard__desc` CSS class (plan 01) — no inline hex anywhere in the new component
- Wrote a 7-test contract suite (raw `react-dom/client` + `flushSync`, matching the codebase's established pattern) covering structure (title/URL/title-attr/actions), DOM-order (desc beneath join beneath top), and action wiring (Copy/Open/QR), including an explicit assertion that the raw capability token never appears in the QR fetch target

## Task Commits

Each task was committed atomically:

1. **Task 1: Create ShareLinkCard.tsx (reusable link row, per-card Copy/Open/QR)** - `920f9d1a` (feat)
2. **Task 2: ShareLinkCard.test.tsx — structure + QR-target + desc-beneath contract** - `c5f38c06` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `frontend/src/components/SessionShare/ShareLinkCard.tsx` - New reusable link-row component: title/URL/actions, `CodeDisplay` join-code, scope description, per-card QR/copy state, `joinURLFor` reused verbatim
- `frontend/src/components/__tests__/ShareLinkCard.test.tsx` - 7-test structure/DOM-order/action-wiring contract suite; no computed-color assertions

## Decisions Made
- Copied `joinURLFor` verbatim into `ShareLinkCard.tsx` rather than exporting it from `SessionSharePanel.tsx` for cross-import — `SessionSharePanel.tsx` is the file plan 06's shell is expected to retire/replace, so a new dependency on it would be short-lived by design. The plan's prohibition ("MUST NOT re-derive QR target URL logic") is satisfied because the logic itself (`new URL(capURL)` → `${protocol}//${host}/join?code=${code}`) is copied verbatim, not reimplemented differently.
- Reused the existing `.daemon-panel__btn` class for Copy/Open/QR buttons inside the new `.share-linkcard__actions` wrapper rather than introducing new button styling — consistent with the plan's "no new plumbing" constraint.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' automated verify commands passed on the first attempt (`tsc --noEmit` clean, `pnpm vitest run` 7/7 green on the new test file). Ran the full frontend gate as an additional check beyond the plan's per-task verification: `tsc --noEmit` clean overall, `pnpm vitest run` 145 files / 2381 tests all passing (no regressions).

## Issues Encountered
The first test-writing pass used `act(async () => flushSync(...))` around click handlers, which triggered a spurious "The current testing environment is not configured to support act(...)" stderr warning (tests still passed). Switched to the codebase's established `await flushSync(() => btn.click())` pattern (matching `SessionSharePanel.test.tsx`'s precedent) — warning gone, same 7/7 pass result.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `ShareLinkCard` is ready to be wired into the plan-06 shell (`SessionShareModal`'s per-tab components) for all three tiers — Tailnet Read-Only, Tailnet Full Access, and Internet public URL — replacing the four hand-laid rows in `SessionSharePanel.tsx`.
- Contract established for future callers: `title`, `url`, `code`, `description`, optional `codeLabel` — no session-id/context plumbing required since `GetCapabilityQRCode` takes only the target URL string.
- No blockers.

---
*Phase: 173-share-modal-three-tab-segmented-redesign*
*Completed: 2026-07-08*

## Self-Check: PASSED

- FOUND: frontend/src/components/SessionShare/ShareLinkCard.tsx
- FOUND: frontend/src/components/__tests__/ShareLinkCard.test.tsx
- FOUND commit: 920f9d1a
- FOUND commit: c5f38c06
