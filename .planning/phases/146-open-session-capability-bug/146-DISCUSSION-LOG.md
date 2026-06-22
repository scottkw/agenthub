# Phase 146: Open Session Capability Bug - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-22
**Phase:** 146-open-session-capability-bug
**Areas discussed:** Remote cap source, Permission level, Scope

---

## Remote cap source

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-fetch via tailnet trust | Local node mints a cap from the remote owner over the authenticated tailnet connection; fully automatic | |
| Reuse existing share cap | Open only works when shared; reuse the web-share cap, mirroring remote-file-browse | ✓ (gate) |
| Join-code modal | Reuse Phase 122 join-code modal to obtain the cap | |

**User's choice:** Accepted Claude's recommended blend — **shared-gated + auto-delivered**: gate Open on the Share toggle (the safety of "reuse existing share cap"), but deliver the cap automatically over the authenticated tailnet connection (the seamlessness of auto-fetch), with no manual join code.
**Notes:** User was unsure which path to take and asked for a recommendation. Rationale accepted: pure auto-fetch adds a new security surface (too much for a bug-fix phase); join-code adds friction for opening one's own session across one's own machines; shared-gated + auto-delivery reuses the proven Phase 122/137 cap model. The exact delivery mechanism (auto-fetch endpoint vs. cap-bearing discovery response) is intentionally left to the research step as a feasibility question — only the intent is locked.

---

## Permission level

| Option | Description | Selected |
|--------|-------------|----------|
| Read-write | Opened session always grants write (interact immediately) | |
| Read-only | Opened session is view-only by default | |
| Match the source | Inherit the share's level; default RW for owner re-attach when both exist | ✓ |

**User's choice:** Match the source — inherit the share's permission (RO share → RO; RW share → RW), no silent escalation; prefer read-write for owner re-attach where both are available.
**Notes:** Aligns with the #98 report ("Nothing ever opens inside the app itself" — user wants to actually use the session).

---

## Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Remote open-in-browser only | Fix exactly the #98 affordance (Hub card "Open in browser", remote) | ✓ |
| Remote + local Open parity | Also unify the local "Open" button path | |

**User's choice:** Remote open-in-browser only — tightest fix that closes #98. Research/verification confirms the local "Open" button and GUI-vs-web paths aren't separately broken (same shared handler), so parity is covered without expanding the build.
**Notes:** —

## Claude's Discretion

- Exact UI treatment for the "session not shared yet → share first" case (disabled control + tooltip vs. redirect to Share modal), as long as it replaces the raw `capability required` 401 dead-end.

## Deferred Ideas

- General "open any tailnet peer's session without sharing it first" / auto-mint-on-request capability — its own phase; new security surface.
- Native in-app remote attach (opening a remote session inside the app instead of the browser) — not part of #98.
