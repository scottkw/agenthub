---
phase: 81-banner-notifications
plan: 01
subsystem: frontend/components
tags: [banner, dismiss, ui-components, css]
dependency_graph:
  requires: []
  provides:
    - "LocalNetworkBanner with onDismiss/className props"
    - "UpdateBanner standalone component"
    - "Banner stack CSS (.banner-stack, .banner-exit, .local-network-banner__dismiss)"
  affects:
    - "frontend/src/components/LocalNetworkBanner.tsx"
    - "frontend/src/components/UpdateBanner.tsx"
    - "frontend/src/components/WelcomeTab.tsx"
    - "frontend/src/style.css"
tech_stack:
  added: []
  patterns:
    - "BEM CSS dismiss button pattern (.local-network-banner__dismiss)"
    - "CSS-only exit animation (.banner-exit with opacity/max-height/padding transitions)"
    - "Extracted component pattern (UpdateBanner lifted from WelcomeTab)"
key_files:
  created:
    - "frontend/src/components/UpdateBanner.tsx"
  modified:
    - "frontend/src/components/LocalNetworkBanner.tsx"
    - "frontend/src/components/WelcomeTab.tsx"
    - "frontend/src/style.css"
decisions:
  - "Kept text 'Dismiss' button on UpdateBanner (matching existing WelcomeTab pattern) rather than X icon"
  - "Used margin-left: auto on dismiss button for right-alignment within flex row"
  - "Exported UpdateInfo interface from UpdateBanner.tsx for App.tsx to import in Plan 02"
metrics:
  duration: "3m"
  completed: "2026-04-16"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 4
---

# Phase 81 Plan 01: Banner Component Layer Summary

Dismissible LocalNetworkBanner with XMarkIcon, standalone UpdateBanner extracted from WelcomeTab, and CSS rules for banner-stack layout and exit animation.

## Task Results

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Modify LocalNetworkBanner + create UpdateBanner + add CSS rules | f2ad687 | LocalNetworkBanner.tsx, UpdateBanner.tsx, style.css |
| 2 | Clean up WelcomeTab — remove lifted update state and banner markup | b7994f0 | WelcomeTab.tsx |

## Changes Made

### Task 1: Banner Components and CSS

**LocalNetworkBanner.tsx:**
- Added `XMarkIcon` import from `@heroicons/react/20/solid`
- Extended props interface with `onDismiss?: () => void` and `className?: string`
- Updated function signature to destructure new props
- All 4 return branches now use dynamic className with template literal for exit animation support
- All 4 return branches render dismiss X button (conditionally when onDismiss provided) with proper aria-label and BEM class

**UpdateBanner.tsx (new):**
- Standalone component extracted from WelcomeTab's update banner JSX
- Exports `UpdateInfo` interface and `UpdateBanner` component
- Props: `update: UpdateInfo`, `onDismiss: () => void`, `className?: string`
- Uses `BrowserOpenURL` from Wails runtime for download link
- Includes `role="alert"` and `aria-live="polite"` for accessibility

**style.css:**
- Added `.local-network-banner__dismiss` ghost button styles (transparent background, 24x24px min target, flex centered)
- Added hover state (color change, border reveal) and focus-visible outline
- Added `.banner-stack` container (flex column, max-height capped at 3 banners, overflow-y auto)
- Added `.banner-exit` animation class (opacity 0, max-height 0, padding collapse with transitions)
- Added `.banner-stack .update-banner` override (margin-bottom: 0)

### Task 2: WelcomeTab Cleanup

- Removed imports: `GetLastUpdateInfo`, `EventsOn`, `BrowserOpenURL`
- Removed `UpdateInfo` interface (now in UpdateBanner.tsx)
- Removed `update` state and `setUpdate` calls
- Removed update subscription `useEffect` (GetLastUpdateInfo + EventsOn)
- Removed update banner JSX block (29 lines)
- WelcomeTab now only manages version state

## Verification Results

- **Task 1:** All 418 tests pass (new props are optional, no contract breakage)
- **Task 2:** 406/418 tests pass; 12 failures in WelcomeTab.test.tsx "update banner (UPD-02, UPD-03)" describe block + 1 cascading version test failure -- all expected and documented in plan as "will be fixed in Plan 02's test updates"

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED

- [x] frontend/src/components/LocalNetworkBanner.tsx exists with onDismiss, className, XMarkIcon
- [x] frontend/src/components/UpdateBanner.tsx exists with UpdateBanner, UpdateInfo exports
- [x] frontend/src/components/WelcomeTab.tsx cleaned of all update-related code
- [x] frontend/src/style.css contains .banner-stack, .banner-exit, .local-network-banner__dismiss
- [x] Commit f2ad687 exists (Task 1)
- [x] Commit b7994f0 exists (Task 2)
