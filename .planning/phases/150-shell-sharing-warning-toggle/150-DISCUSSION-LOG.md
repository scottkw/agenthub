# Phase 150: Shell-Sharing Warning Toggle - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-23
**Phase:** 150-shell-sharing-warning-toggle
**Areas discussed:** Toggle ↔ warned interplay, Placement & label, Confirm on disable, Default state, Scope (Share-modal parity)

---

## Toggle Model (warning ↔ one-time acknowledgment)

| Option | Description | Selected |
|--------|-------------|----------|
| Re-arm, keep one-time ack | New `enabled` setting separate from existing `warned`; warning shows iff `shell && enabled && !warned`; OFF→ON resets `warned`. Preserves Phase 101 behavior. | ✓ |
| Always warn when ON | Single-flag model; ON warns every shell web-share, drops the one-time acknowledgment entirely. | |

**User's choice:** Re-arm, keep one-time ack
**Notes:** Preserves the Phase 101 "acknowledge once → don't show again" behavior while satisfying Success Criterion 2 (turning ON restores the warning even after a prior acknowledgment).

---

## Placement & Label

| Option | Description | Selected |
|--------|-------------|----------|
| Security section | Alongside the key-rotation panic button (SettingsTab.tsx:675). | |
| Session Behavior section | Next to the Auto-close-on-exit toggle (SettingsTab.tsx:413). | ✓ |

**User's choice:** Session Behavior section
**Notes:** Label: "Warn before web-sharing a shell session." Reuse colorblind-safe role=switch pattern.

---

## Confirm on Disable

| Option | Description | Selected |
|--------|-------------|----------|
| Silent toggle | Flip off instantly like other Settings toggles. | |
| Confirm before disabling | Confirmation dialog on OFF; instant on ON. | ✓ |

**User's choice:** Confirm before disabling
**Notes:** Deliberate exception to the silent-toggle pattern because disabling weakens a security guardrail (defense-in-depth, Phase 87 SEC-01).

---

## Default State

| Option | Description | Selected |
|--------|-------------|----------|
| ON (warning enabled) | Preserves current safe first-run behavior. | ✓ |
| OFF (warning disabled) | No warning unless user opts in. | |

**User's choice:** ON (warning enabled)

---

## Scope — Share-modal parity

Surfaced during discussion: Phase 137 split sharing into two surfaces; the shell
warning only fires via the legacy StatusBar path, not the primary Hub Share modal.

| Option | Description | Selected |
|--------|-------------|----------|
| Toggle + wire warning into Share modal | Add toggle AND surface the warning in the Share modal ON-path for shells; closes cross-surface parity gap. | ✓ |
| Toggle only (literal SET-01) | Just the toggle governing the existing StatusBar-path warning; leaves parity gap open. | |
| File the gap as its own phase | Toggle-only now; separate issue/phase for the Share-modal gap. | |

**User's choice:** Toggle + wire warning into Share modal
**Notes:** Expands phase beyond a literal reading of SET-01/#51. SET-01 wording and the Phase 150 ROADMAP goal should be updated to reflect the parity fix. Cross-surface parity is release-blocking per the standing rule.

---

## Claude's Discretion

- Exact confirm-dialog component and wording (reuse existing dialog primitives).
- New setting's field/RPC name (mirror `ShellWebShareWarned` naming).
- Share-modal warning rendering (inline banner vs modal variant) — behavior must match.

## Deferred Ideas

- Retire/consolidate the legacy StatusBar per-tab web toggle (SHARE-02 said the
  modal toggle "replaces Web On," yet the StatusBar toggle remains). Separate cleanup.
- Web-surface behavior — out of scope; remote visitors don't initiate shell sharing.
