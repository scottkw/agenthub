# Phase 85: Quit Confirmation Modal - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-19
**Phase:** 85-quit-confirmation-modal
**Areas discussed:** Zero-session behavior, Modal visual tone, Tray quit intercept, "Quit GUI only" UX

---

## Zero-Session Behavior

### Q1: When no sessions are active (count = 0), what should happen on quit?

| Option | Description | Selected |
|--------|-------------|----------|
| Skip modal, quit immediately | No sessions at risk, quit proceeds without interruption | |
| Always show modal | Modal always appears regardless of session count; two-choice still relevant since daemon may be running | ✓ |
| Show modal only for daemon | Skip if session count is 0 AND daemon has no other work | |

**User's choice:** Always show modal
**Notes:** Consistent behavior regardless of session state; daemon running is still a meaningful distinction.

### Q2: When session count is 0, should the modal show '0 active sessions' or an adjusted message?

| Option | Description | Selected |
|--------|-------------|----------|
| Same layout, show '0' | Always display '0 active sessions' — consistent structure | |
| Adjusted message | Swap count line for 'No active sessions' when 0; keep both exit buttons | ✓ |

**User's choice:** Adjusted message
**Notes:** Friendlier messaging when no sessions are at risk.

---

## Modal Visual Tone

### Q1: What visual tone should the quit confirmation modal have?

| Option | Description | Selected |
|--------|-------------|----------|
| Neutral confirmation | Match existing modal styling, session count displayed plainly, equal button styling | |
| Cautious with emphasis | Warning accent on session count, destructive styling on 'Quit All' | |
| Informational with context | Session list with names and status, clearly differentiated buttons | ✓ |

**User's choice:** Informational with context
**Notes:** User selected the preview showing individual session listing with agent name and status.

### Q2: Should 'Quit Everything' have destructive styling?

| Option | Description | Selected |
|--------|-------------|----------|
| Destructive accent on Quit All | Red/warning styling, standard UX pattern for destructive actions | ✓ |
| Equal styling | Both buttons styled the same, session list provides context | |

**User's choice:** Destructive accent on Quit All
**Notes:** Standard destructive action pattern.

---

## Tray Quit Intercept

### Q1: How should tray Quit route through the frontend modal?

| Option | Description | Selected |
|--------|-------------|----------|
| Wails event to frontend | Tray Quit emits event, frontend shows modal, user choice calls back to Go | ✓ |
| Show window first, then modal | Un-hide GUI window first, then trigger modal. More visual. | |
| You decide | Claude picks best technical approach | |

**User's choice:** Wails event to frontend (Recommended)
**Notes:** Consistent path — both window-close and tray-quit show the same modal.

### Q2: When tray emits quit event and window is hidden, should window auto-show?

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-show window with modal | Bring window to foreground with modal displayed | ✓ |
| Modal without window show | Keep window hidden, handle via native OS dialog from Go | |

**User's choice:** Auto-show window with modal
**Notes:** User sees the modal immediately with session context.

---

## "Quit GUI Only" UX

### Q1: When user picks 'Quit GUI Only', what happens to the GUI window?

| Option | Description | Selected |
|--------|-------------|----------|
| Hide to tray (current behavior) | Same as clicking close today — window hides, tray stays, daemon keeps running | |
| Fully close GUI process | GUI exits entirely, daemon keeps running, tray icon goes away | |
| Hide + brief toast confirmation | Hide to tray plus OS notification confirming sessions still running | ✓ |

**User's choice:** Hide + brief toast confirmation
**Notes:** User wants confirmation that the app is still running in the background.

### Q2: What type of confirmation — OS notification or in-app toast?

| Option | Description | Selected |
|--------|-------------|----------|
| OS notification | macOS Notification Center banner, visible after window hides | ✓ |
| In-app toast then hide | Brief in-app toast before window hides, no OS permission needed | |
| You decide | Claude picks best approach | |

**User's choice:** OS notification
**Notes:** Visible even after window hides; standard pattern for tray-minimized apps.

---

## Claude's Discretion

- Wails event naming convention for quit-requested event
- Modal component structure (new component vs extending existing patterns)
- OS notification implementation (Wails notification API or Go-native)
- Session list rendering and truncation in modal
- Cancel button placement and keyboard shortcuts
- Modal animation/transition

## Deferred Ideas

None — discussion stayed within phase scope
