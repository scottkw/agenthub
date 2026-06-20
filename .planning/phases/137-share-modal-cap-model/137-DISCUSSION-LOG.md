# Phase 137: Share Modal & Cap Model - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-20
**Phase:** 137-share-modal-cap-model
**Areas discussed:** RW-browse write gating, RO code reads filesystem, Toggle scope vs. globals, Modal reuse & trigger

---

## RW-browse write gating

| Option | Description | Selected |
|--------|-------------|----------|
| Inherit fully from RW code | RW code → files.read + files.write automatically; single browse toggle is the only gate; no owner write opt-in, no viewer confirm. | ✓ |
| Keep viewer confirm (CAP-05) | RW code grants files.read; files.write still needs the viewer to confirm "Allow file editing". | |
| Keep both existing gates | Browse enables read-browse; write still needs owner per-session opt-in AND viewer confirm. | |

**User's choice:** Inherit fully from RW code.
**Notes:** Justification — the RW code already grants full terminal read+write (complete session control), so files.write is not a real escalation. Removes the CAP-05 two-gate and the separate per-session files.write opt-in (SetSessionFilesWrite).

### Follow-up: default & home-dir warning

| Option | Description | Selected |
|--------|-------------|----------|
| Default OFF, keep CAP-06 warning | Browse defaults OFF per session; home-dir write warning shown when cwd is $HOME. | ✓ |
| Default OFF, drop CAP-06 warning | Defaults OFF, no special home-dir warning. | |
| Default ON when sharing | Sharing the session also enables browsing by default. | |

**User's choice:** Default OFF, keep CAP-06 warning.

---

## RO code reads filesystem

| Option | Description | Selected |
|--------|-------------|----------|
| Read-only filesystem browse | RO code → files.read (list/stat/read, no writes). Matches SHARE-03; RO link now exposes file contents. | ✓ |
| Browse requires RW code only | Browse toggle only grants file access to RW-code holders; RO stays terminal-view-only. | |

**User's choice:** Read-only filesystem browse.
**Notes:** RO-code holders gaining read-only filesystem access is intended and accepted; owner accepts the exposure by enabling browse.

---

## Toggle scope vs. globals

| Option | Description | Selected |
|--------|-------------|----------|
| Global stays as master kill-switch | Per-session browse only takes effect if global files-browsing setting is also ON (AND-gate). | |
| Per-session only; remove global | Per-session browse toggle is the single source of truth; global setting removed from Settings. | ✓ |
| Per-session only; keep global hidden/legacy | Per-session authoritative; leave global flag internally but stop surfacing it. | |

**User's choice:** Per-session only; remove global.
**Notes:** Per-session toggle becomes sole driver of files.read (both codes) and files.write (RW code) injection; subsumes SetSessionFilesWrite and filesReadEnabled.

### Follow-up: persistence

| Option | Description | Selected |
|--------|-------------|----------|
| Ephemeral — match web-serve | In-memory; daemon restart resets sharing/browsing to OFF; modal seeds from server truth. | ✓ |
| Persist across restarts | Written to daemon config and restored on restart (cap-reissue-on-startup). | |

**User's choice:** Ephemeral — match web-serve.

---

## Modal reuse & trigger

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse + simplify SessionSharePanel | Wrap existing panel in modal, strip dead CAP-05 two-gate UI, wire single browse toggle. | ✓ |
| Rebuild modal from scratch | New component; SessionSharePanel retired; re-implement lifecycle. | |

**User's choice:** Reuse + simplify SessionSharePanel.

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated Share button | Visible Share button/icon on the card (SHARE-01 literal). | ✓ |
| Overflow menu item | Add "Share" to existing EllipsisHorizontal overflow menu. | |
| You decide at UI-spec time | Defer placement to Phase 140 UI-spec gate. | |

**User's choice:** Dedicated Share button.

### SHARE-06: remote peer cards

| Option | Description | Selected |
|--------|-------------|----------|
| Hidden entirely | No Share button on remote peer cards. | |
| Visible but disabled + tooltip | Greyed-out Share button with tooltip; must be colorblind-safe (greyed + lock icon + tooltip). | ✓ |

**User's choice:** Visible but disabled + tooltip.
**Notes:** Disabled state must not rely on color alone (colorblind-owner release norm).

---

## Claude's Discretion

- Exact `SessionEngine` mechanism for collapsing `SetSessionFilesWrite`/`filesReadEnabled` into the per-session browse state.
- Cap-reissue-on-toggle plumbing through `issueCapabilitiesForSession`.
- Migration/cleanup of the removed global Settings control.
- Final modal copy/labels and disabled-state iconography (subject to Phase 140 UI-spec).

## Deferred Ideas

- Persisting share/browse state across daemon restarts with cap-reissue-on-startup (chose ephemeral).
- Card visual redesign, local/remote + connected/available indicators, mini-preview/tail VT render fix — CARD-01..05 / RDS in Phases 138-140.
