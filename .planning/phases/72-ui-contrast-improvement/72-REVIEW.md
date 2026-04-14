---
phase: 72-ui-contrast-improvement
reviewed: 2026-04-14T12:05:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - frontend/src/components/__tests__/style.contrast.test.ts
  - frontend/src/style.css
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
status: issues_found
---

# Phase 72: Code Review Report

**Reviewed:** 2026-04-14T12:05:00Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

This review covers the WCAG AA contrast improvement work: a systematic replacement of the low-contrast `#565f89` text color (ratio ~2.8:1) with the compliant `#9aa5ce` (ratio ~6.6-7.4:1) across 32 CSS rules in `style.css`, plus a new regression test suite in `style.contrast.test.ts`.

The CSS changes are clean and mechanical -- every instance of `#565f89` has been replaced, and no instances remain. The replacement color `#9aa5ce` passes WCAG AA (>= 4.5:1) against all three background colors used in the UI. The sRGB linearization and contrast ratio calculations in the test file are correct per IEC 61966-2-1.

The intentional preservation of `#414868` on `.tab-status-bar__state--inactive` (a deliberately-dim indicator) is properly guarded by a positive assertion test.

Two minor items found.

## Warnings

### WR-01: Test coverage gap -- regression tests cover only 13 of 32 changed selectors

**File:** `frontend/src/components/__tests__/style.contrast.test.ts:38-98`
**Issue:** The "no failing #565f89 text color" regression tests cover 12 selectors (plus 1 positive assertion), but `style.css` had 32 selectors changed from `#565f89` to `#9aa5ce`. The following 19 changed selectors have no regression test preventing reintroduction of the low-contrast color:

- `.daemon-panel__empty` (line 1102)
- `.daemon-panel__count` (line 1123)
- `.daemon-panel__cli` (line 1172)
- `.daemon-panel__hostname` (line 1180)
- `.remote-panel__loading` (line 1249)
- `.remote-panel__empty-title` (line 1275)
- `.remote-panel__empty-body` (line 1281)
- `.remote-panel__peer-header` (line 1297)
- `.remote-panel__peer-meta` (line 1304)
- `.remote-panel__cli` (line 1353)
- `.local-network-banner__sub` (line 1419)
- `.settings-web-server__password-label` (line 1451)
- `.settings-web-server__credential-hint` (line 1457)
- `.settings-web-server__action-btn` (line 1531)
- `.new-session-modal__close` (line 573)
- `.new-session-modal__folder-display` (line 643)
- `.new-session-modal__btn--close` (line 720)
- `.update-banner__arrow` (line 919)
- `.update-banner__btn--dismiss` (line 948)
- `.settings-panel__btn--cancel` (line 428)

If any of these are later reverted to `#565f89` (e.g., during a merge conflict resolution), the regression test suite would not catch it.

**Fix:** Add test cases for the uncovered selectors. A data-driven approach avoids boilerplate:

```typescript
const mustNotUse565f89 = [
  '.tab', '.tab__close', '.tab-status-bar', '.tab-status-bar__state--off',
  '.settings-panel__body\\s+h3', '.settings-panel__description',
  '.settings-panel__empty', '.settings-panel__table\\s+th',
  '.settings-panel__url', '.settings-panel__btn--cancel',
  '.welcome-tab__version', '.welcome-tab__heading',
  '.new-session-modal__section-label', '.new-session-modal__close',
  '.new-session-modal__folder-display', '.new-session-modal__btn--close',
  '.daemon-panel__empty', '.daemon-panel__count',
  '.daemon-panel__cli', '.daemon-panel__hostname',
  '.remote-panel__loading', '.remote-panel__empty-title',
  '.remote-panel__empty-body', '.remote-panel__peer-header',
  '.remote-panel__peer-meta', '.remote-panel__cli',
  '.local-network-banner__sub',
  '.settings-web-server__password-label',
  '.settings-web-server__credential-hint',
  '.settings-web-server__action-btn',
  '.update-banner__arrow', '.update-banner__btn--dismiss',
]

describe('UI-01: no #565f89 text color in any changed selector', () => {
  for (const sel of mustNotUse565f89) {
    it(`${sel} does not use #565f89`, () => {
      const re = new RegExp(`${sel.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*\\{[^}]*color:\\s*#565f89`)
      expect(css).not.toMatch(re)
    })
  }
})
```

## Info

### IN-01: sRGB linearization threshold uses 0.04045 (IEC 61966-2-1) rather than WCAG 2.0's 0.03928

**File:** `frontend/src/components/__tests__/style.contrast.test.ts:10`
**Issue:** The WCAG 2.0 spec references a threshold of 0.03928, while the IEC 61966-2-1 standard uses 0.04045. The test uses 0.04045. Both are considered correct -- the WCAG value was a rounding artifact, and 0.04045 is the more precise number. The practical difference is negligible (affects only a handful of color values near the threshold, with sub-0.001 luminance impact). No action needed; this note is for documentation purposes only.
**Fix:** None required. A comment noting the intentional choice could prevent future "fix" attempts:

```typescript
// 0.04045 per IEC 61966-2-1 (more precise than WCAG 2.0's 0.03928).
return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
```

---

_Reviewed: 2026-04-14T12:05:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
