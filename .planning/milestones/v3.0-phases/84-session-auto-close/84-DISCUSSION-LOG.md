# Phase 84: Session Auto-Close - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-19
**Phase:** 84-session-auto-close
**Areas discussed:** Close timing & animation, Exit notification style, Output flush guarantee, Edge cases & opt-out

---

## Close Timing & Animation

### Q1: How long should the tab stay visible after the agent process exits?

| Option | Description | Selected |
|--------|-------------|----------|
| Brief delay (2-3s) | Enough to catch the last line of output, then auto-close. Keeps the UI tidy quickly. | |
| Moderate delay (5s) | Gives the user time to read a summary or error message before the tab disappears. | |
| Countdown with cancel | Show a visible countdown (e.g. 'Closing in 5s...') that the user can click to cancel, keeping the tab open. | ✓ |

**User's choice:** Countdown with cancel
**Notes:** User wants user control via cancel button during the countdown window.

### Q2: How should the tab disappear?

| Option | Description | Selected |
|--------|-------------|----------|
| Instant removal | Tab just disappears from the tab bar after the delay — same as manual close. | |
| Fade-out effect | Tab fades/dims before removal — gives a visual cue that it's about to close. | ✓ |
| You decide | Claude picks the approach that best fits the existing tab bar patterns. | |

**User's choice:** Fade-out effect

### Q3: How long should the countdown default be?

| Option | Description | Selected |
|--------|-------------|----------|
| 3 seconds | Quick but enough time to click cancel if needed. | |
| 5 seconds (Recommended) | Comfortable reading window — matches common toast durations. | ✓ |
| 10 seconds | Extra-generous for longer output. Might feel sluggish for quick tasks. | |

**User's choice:** 5 seconds

### Q4: Where should the countdown appear?

| Option | Description | Selected |
|--------|-------------|----------|
| Banner in terminal area | A bar at the top or bottom of the terminal content with 'Closing in Xs... [Keep Open]' | |
| Overlay on the tab itself | Small countdown badge or text on the tab bar tab | |
| Both tab + terminal | Tab shows visual indicator while terminal shows the actionable banner with cancel button | ✓ |

**User's choice:** Both tab + terminal

---

## Exit Notification Style

### Q1: What kind of notification when the agent exits?

| Option | Description | Selected |
|--------|-------------|----------|
| Toast popup (Recommended) | A brief floating notification showing '[Agent] exited' — visible even from other tabs. | |
| Inline banner only | Notification lives inside the terminal tab area only. | |
| Toast + inline banner | Toast ensures visibility from any tab; inline banner gives context in the exiting tab. | ✓ |

**User's choice:** Toast + inline banner

### Q2: What info should the exit notification include?

| Option | Description | Selected |
|--------|-------------|----------|
| Session name + agent type | e.g. 'Claude session "my-project" exited' | ✓ |
| Exit code | Show exit code (0 = clean, non-zero = error) for debugging context | ✓ |
| Session duration | How long the session was running (e.g. '42m 15s') | ✓ |
| Final status | Was it running/idle/waiting/errored at exit time | ✓ |

**User's choice:** All four — session name + agent type, exit code, session duration, final status

---

## Output Flush Guarantee

### Q1: How should we detect that output is fully flushed?

| Option | Description | Selected |
|--------|-------------|----------|
| PTY EOF detection (Recommended) | When the PTY read returns EOF, all output has been delivered. Start countdown only after EOF. | ✓ |
| Quiet period heuristic | Wait until no new output arrives for ~500ms after process exit. | |
| You decide | Claude picks the most reliable approach given the PTY/relay architecture. | |

**User's choice:** PTY EOF detection

### Q2: Should the terminal remain scrollable during the countdown?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, fully interactive (Recommended) | User can scroll, select, copy text from the terminal during the countdown. | ✓ |
| Read-only frozen view | Terminal content visible but not interactive. | |
| You decide | Claude picks based on implementation complexity. | |

**User's choice:** Yes, fully interactive

---

## Edge Cases & Opt-Out

### Q1: Should error exits behave differently from clean exits?

| Option | Description | Selected |
|--------|-------------|----------|
| Same behavior for all exits | Countdown + close regardless of exit code. | |
| Keep tab open on error (Recommended) | Non-zero exit code skips auto-close — tab stays open for review. | ✓ |
| Longer delay on error | Error exits get longer countdown but still auto-close. | |

**User's choice:** Keep tab open on error

### Q2: Should users be able to disable auto-close globally?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, settings toggle (Recommended) | Toggle in Settings to disable auto-close. Default: enabled. | ✓ |
| No opt-out needed | Cancel button during countdown is sufficient. | |
| Per-session choice | Pin sessions to prevent auto-close individually. | |

**User's choice:** Yes, settings toggle

### Q3: What happens when a session with active web viewers exits?

| Option | Description | Selected |
|--------|-------------|----------|
| Close tab, keep web serving briefly | GUI tab closes normally. Web viewers see final output + 'Agent exited'. Web stops after grace period. | ✓ |
| Same as no viewers | Tab auto-closes regardless. Web connections terminated on cleanup. | |
| You decide | Claude picks based on web server architecture. | |

**User's choice:** Close tab, keep web serving briefly

---

## Claude's Discretion

- Exit detection mechanism details (how backend detects natural process exit)
- Toast component implementation approach
- Fade-out animation duration and CSS
- Web serving grace period duration
- Settings toggle placement within Settings tab

## Deferred Ideas

None — discussion stayed within phase scope
