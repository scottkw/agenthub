# Phase 168: Bug Fix & Settings Polish - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-01
**Phase:** 168-bug-fix-settings-polish
**Areas discussed:** Viewer-count rule (FIX-04), Disconnect UX (FIX-02), Stay-on-Hub toggle (UX-01), Footer "Share Session" (UX-02)

---

## Viewer-count rule (FIX-04 / #121)

### What the count includes
| Option | Description | Selected |
|--------|-------------|----------|
| Only Origin=='web' | Count web-origin subscribers only; excludes local-loopback app connections. A never-shared local session reads 0. | ✓ |
| Everything except a hardcoded local list | Keep counting all subscribers but subtract known-internal client names. Fragile. | |

### Connections vs distinct persons
| Option | Description | Selected |
|--------|-------------|----------|
| Raw web connections | Each web-origin WebSocket = 1; two tabs = 2. Straight len() over Origin=='web'. | ✓ |
| Distinct web persons (PersonKey) | Collapse by PersonKey; two tabs from one person = 1. | |

**User's choice:** Only Origin=='web', raw connection count.
**Notes:** User qualified — "#1 is fine as long as they are actual connections and not counting web, chat, Hub preview, etc. all as separate connections." Verified this holds: a web guest's terminal + chat share one WS (`WebShareSessionView.tsx:57`), plugin-config SSE is tracked outside the hub, and Hub preview is local-origin. So one guest tab = 1 connection; the caveat is satisfied by the single-WS-per-guest design.

---

## Disconnect UX (FIX-02 / #117)

### Disconnect scope
| Option | Description | Selected |
|--------|-------------|----------|
| Single "Disconnect all viewers" button | One button force-closes all web-origin connections. No per-viewer UI. | ✓ |
| Per-viewer list with individual disconnect | Live roster, each viewer its own Disconnect button. Heavier. | |

### Rejoin after disconnect
| Option | Description | Selected |
|--------|-------------|----------|
| Just drop connections | Force-close sockets only; cap stays valid so a stuck viewer can reconnect. | ✓ |
| Disconnect + revoke cap | Also invalidate the capability; kicked viewers need a fresh join code. | |

**User's choice:** Single "Disconnect all viewers" button; drop connections only (no revoke).
**Notes:** The multi-viewer-kick half of #117 was flagged as a research item — confirm whether Phase 165's dual-origin fix already resolved it before scoping relay work.

---

## Stay-on-Hub toggle (UX-01 / #116)

### Default state
| Option | Description | Selected |
|--------|-------------|----------|
| Default OFF | Preserves today's auto-switch behavior; toggle opts into staying on Hub. | ✓ |
| Default ON | Sessions stay on Hub by default; opt into auto-switch. | |

### Scope (clarified mid-discussion)
| Option | Description | Selected |
|--------|-------------|----------|
| Only Hub-created sessions | Toggle governs the Hub "New session" path only. | ✓ (collapses to this) |
| All session creation | Toggle suppresses auto-switch for every path. | |

### On-create focus when ON
| Option | Description | Selected |
|--------|-------------|----------|
| Stay on Hub, session appears as a card | Skip setActiveId; still create the tab; stay on Hub. | ✓ |
| Stay on Hub, don't even create the tab | Daemon-only create; build the tab on demand from the card. | |

**User's choice:** Default OFF; when ON, skip setActiveId but still create the tab (stay on Hub); gates only the Hub-create path.
**Notes:** User asked where else sessions can be created besides the Hub. Investigation showed the GUI "New session" button exists only on HubPanel (`App.tsx:1534`) → `createTab` → `setActiveId` (the sole auto-switch), and CLI-created sessions never run through `createTab` (they never auto-switch the GUI). So "all creation" and "Hub-created only" collapse to the same single gate — no `fromHub` flag needed.

---

## Footer "Share Session" button (UX-02 / #115)

### Button role
| Option | Description | Selected |
|--------|-------------|----------|
| Always opens Share modal | Removes footer's direct ToggleWebServing; modal is single source of truth. | ✓ |
| Opens modal to enable, toggles off directly | Keeps a quick-off path but reintroduces two share-state code paths. | |

### When active tab isn't a session
| Option | Description | Selected |
|--------|-------------|----------|
| Hidden when no active session | Button only renders on a real shareable local session. | ✓ |
| Disabled (greyed) when no active session | Always present but non-interactive off a session. | |

**User's choice:** Always opens the Share modal; hidden when the active tab isn't a shareable local session.

---

## Claude's Discretion

- Exact new-method / endpoint names (`RemoteViewerCount`, disconnect endpoint shape), button styling, and precise wiring of the disconnect action.
- Whether the "Disconnect all viewers" button shows always or only when there are web viewers to disconnect (prefer only when `viewerCount > 0`).
- FIX-01 and FIX-03 approaches were locked as "determined by requirement" (self-fetch plugin-config + SSE; reroute remote-open to the in-app web-session tab) — not discussed as gray areas.

## Deferred Ideas

- Per-viewer disconnect list / viewer-roster UI — beyond bug-fix scope.
- Disconnect + cap revocation as one action — overlaps with "disable web sharing".
- FIX-05 / #120 Tailscale detection — already split into Phase 169.
- Reviewed-not-folded todo: "Help Guide — document Tailscale Funnel admin prerequisites" (belongs to Funnel Help / Phase 166, not this phase).
