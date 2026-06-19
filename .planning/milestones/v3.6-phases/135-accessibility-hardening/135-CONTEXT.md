# Phase 135: Accessibility Hardening - Context

**Gathered:** 2026-06-18
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Every Hub interaction is fully operable by keyboard and safe for colorblind users — attention, status, and motion all carry non-color cues, and animations respect prefers-reduced-motion.

Requirements: A11Y-01, A11Y-02, A11Y-03, A11Y-04.

This phase validates the full Hub surface built across Phases 131–134 (cards, grid + groups, attention + pulse, modal interaction). It is hardening, not new feature work: the existing components must be made fully keyboard-operable, colorblind-safe, and motion-respectful.

Success criteria (what must be TRUE):

  1. All attention and status states are distinguishable without color: each state is uniquely identifiable by its icon shape and/or motion and/or position alone (verified at source level against hex constants — not by eye).
  2. A user can navigate the entire Hub by keyboard: Tab moves between cards, Enter/Space expands a card into its modal, Escape closes the modal and returns focus to the originating card, and the `/` search shortcut is reachable without a mouse.
  3. With `prefers-reduced-motion: reduce` set in the OS, pulse and expand/collapse animations are replaced by a static highlighted border + icon — no motion occurs; all information previously conveyed by motion is conveyed by the static fallback.
  4. While a modal is open, focus is trapped inside it — Tab cycles through modal controls only, and background cards are not reachable by keyboard.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

### Standing project constraints (from user memory — apply, do not re-decide)
- **Colorblind-safe verification:** The user is colorblind. Verify all color-based criteria at the source level (hex constants in code), NOT by eye. Every state distinguished by color MUST also be distinguished by icon shape, motion, position, or text.
- **Cross-surface parity is release-blocking:** GUI/TUI/CLI must stay in sync. If an a11y affordance has a parity implication across surfaces, surface it — do not silently defer.

</decisions>

<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research. Key prior art: the Hub surface components delivered in Phases 131–134 (SessionCard, SessionCardGrid, HubPanel, HubFilterBar, attention/pulse logic, the card→modal shared-element animation). Phase 135 modifies these existing components rather than creating new surfaces.

</code_context>

<specifics>
## Specific Ideas

No specific requirements beyond the success criteria — discuss phase skipped. Refer to ROADMAP phase description and success criteria above.

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
