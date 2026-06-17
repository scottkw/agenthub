# Requirements: AgentHub — v3.6 Hub (Session Grid / Control Room)

**Defined:** 2026-06-16
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Primary issue:** #78 (Session Grid). **Companion:** #76 (closed) folded in for live-session behaviors not covered by #78.

## Scope Decisions (ratified 2026-06-16)

- **Bind to real data, adapt the look.** Cards render only fields `SessionInfo` actually supplies (id, name, CLI, status, hostname, webEnabled, viewerCount, createdAt, exitCode, duration) plus per-session working directory. #78's "projects" are realized as working-directory grouping. No mock/placeholder fields.
- **Both modals, by state.** Blocked (`waiting`/needs-input) sessions open a briefing modal **driven by the real terminal tail** (not the structured "agent suggests options" list from #78 — agents don't emit that; deferred to #93). All other sessions expand to a full interactive terminal modal (#76).
- **Hub coexists** with the existing Sessions (DaemonManagerPanel) view — it does not replace it.
- **Attention granularity v1 = `waiting` + `errored`/non-zero exit** (both free from the existing status detector). Terminal-bell / OSC-9 and "activity-while-unobserved" signals deferred.
- **Card previews are throttled snapshots** (recent output tail), never a heavyweight live xterm per card — that does not scale. Only the modal mounts a live interactive terminal.
- **Group membership keys off session name (+ working directory)** to survive ephemeral session-id churn across restarts; unmatched sessions fall to a default lane for manual reassignment.
- **CLI parity** is satisfied by the existing `agenthub list` status column (already surfaces `waiting`/`errored`) — no new CLI work this milestone.
- **TUI Hub parity deferred** to a later milestone with explicit user sign-off — tracked on issue **#82**. Cross-surface parity remains a release-blocking contract; this is a signed-off deferral, not a silent gap.

## v3.6 Requirements

### Hub Surface & Navigation (HUB)

- [x] **HUB-01**: User can open the Hub from a "Hub" item in the left sidebar
- [x] **HUB-02**: Hub is a top-level surface alongside Home / Remote / Sessions / Settings, coexisting with (not replacing) the existing Sessions panel
- [x] **HUB-03**: When no sessions exist, Hub shows an empty state prompting the user to create one
- [x] **HUB-04**: Hub renders correctly in both light and dark themes

### Session Card (CARD)

- [x] **CARD-01**: Each card shows the session name, inline-editable consistent with TabBar rename behavior
- [x] **CARD-02**: Each card shows the CLI/agent badge using the existing per-CLI color/badge mapping
- [x] **CARD-03**: Each card shows a status indicator conveyed by shape + icon + motion (not color alone)
- [x] **CARD-04**: Each card shows an origin marker — local vs remote, with the peer hostname for remote sessions
- [x] **CARD-05**: Each card shows the viewer count when the session is web-shared
- [x] **CARD-06**: Each card shows uptime while running, or duration + exit code once stopped
- [x] **CARD-07**: Each card shows a mini terminal preview of the session's recent output tail
- [x] **CARD-08**: Stopped/exited cards render dimmed with exit code and no pulse, unless the exit was an error (→ attention)

### Grid, Grouping, Filter & Search (GRID)

- [x] **GRID-01**: Cards render in a responsive grid that reflows by viewport width with sensible min/max card sizes
- [x] **GRID-02**: Cards are auto-grouped by working directory (the real-data analog of #78 "projects")
- [x] **GRID-03**: A collapsible group sidebar shows per-group running/total counts and a needs-input badge; selecting a group filters the grid to it
- [x] **GRID-04**: A status filter bar (All / Working / Needs input / Complete / Error / Idle) filters cards with live counts
- [x] **GRID-05**: A functional search field filters cards by name/CLI/host, activated by the `/` shortcut
- [x] **GRID-06**: A "New session" action on the Hub opens the existing create flow
- [x] **GRID-07**: The grid includes both local daemon sessions and remote tailnet/web-shared peer sessions in one unified view

### Attention (ATTN)

- [x] **ATTN-01**: A session with status `waiting`, or `errored`/non-zero exit, is flagged as needing attention
- [x] **ATTN-02**: An attention card shows a pulsing animated highlighted border plus an attention icon
- [x] **ATTN-03**: When cards overflow the viewport, attention cards sort to the top
- [ ] **ATTN-04**: Reordering on status change is debounced and position changes are animated (non-jarring)
- [ ] **ATTN-05**: Resolving a `waiting` session inside its modal clears that card's attention state
- [x] **ATTN-06**: A collapsed group containing an attention card shows an attention badge on its header

### Card → Modal Interaction (MODAL)

- [ ] **MODAL-01**: Clicking a card expands it into a modal via a shared-element-style grow animation
- [ ] **MODAL-02**: Closing the modal shrinks it back into the originating card's position and restores focus
- [ ] **MODAL-03**: For non-blocked sessions, the modal mounts a full interactive terminal (the same `TerminalPanel` + input-wired `RelayClient` used by normal tabs)
- [ ] **MODAL-04**: For `waiting`/needs-input sessions, the modal opens a briefing view driven by the real terminal tail (the prompt the agent printed) with a respond affordance
- [ ] **MODAL-05**: The modal session is fully functional — resize, copy/paste, and scrollback search all work
- [ ] **MODAL-06**: For a remote session requiring a capability, the modal interaction uses the existing remote-open / join-code exchange path (locked Phase 122 design)

### Named Groups (GROUP)

- [x] **GROUP-01**: User can create named groups
- [x] **GROUP-02**: User can assign cards to a group via drag-and-drop or a per-card "move to group" affordance
- [x] **GROUP-03**: Group definitions and membership persist locally (localStorage, consistent with existing layout-state persistence)
- [x] **GROUP-04**: Group membership keys off session name (+ working directory) so it survives session-id churn across restarts; unmatched sessions fall to a default lane for manual reassignment

### Accessibility (A11Y)

- [x] **A11Y-01**: Attention and status are conveyed by motion + icon + position, never by color alone (colorblind-safe)
- [ ] **A11Y-02**: Cards are keyboard-focusable; Enter/Space expands; Escape closes the modal and returns focus to the originating card
- [x] **A11Y-03**: Pulse and expand/collapse animations honor `prefers-reduced-motion`, falling back to a static highlighted border + icon
- [ ] **A11Y-04**: The modal traps focus while open

## Future Requirements (deferred)

Tracked but not in the v3.6 roadmap.

### Deferred #78 fidelity → issue #93

- **Per-session usage metrics** — token count, spend (USD), context-window % on cards (overlaps #67; backend does not track today)
- **Formal projects model** — first-class projects with code/color/desc, replacing working-directory auto-grouping
- **Member/collaborator avatars + presence** — single-user/tailnet model today
- **Structured "agent suggests" briefings** — context timeline + multi-select options + recommended choice (requires agent-emitted structured decision data)
- **Session detail / chat-thread page** — the linked per-session message thread with tool-output blocks (specified but unbuilt in the #78 mock)
- **Tweaks panel** — design-tooling density/modal/accent live-editing panel

### TUI Hub parity → issue #82

- Attention indicator + float-to-top + named-group support in the TUI unified session list (signed-off deferral)

### Richer attention signals (future)

- Terminal bell (`BEL` / `OSC 9`) → new `session:attention` event from the relay/detector
- "Activity while unobserved" → new output on a non-active card (gate behind a setting)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Live interactive xterm per card | Won't scale to dozens of cards; throttled snapshot previews instead, live terminal only in the modal |
| Replacing the Sessions (DaemonManagerPanel) view | Hub coexists; replacement is a separate UX decision |
| Pixel-faithful #78 port with mock data | Chose real-data binding; mock fidelity tracked in #93 |
| Backend usage/cost/token tracking | Out of this milestone; overlaps #67, tracked in #93 |
| Bespoke Hub color palette | Reuse existing app theme tokens (light/dark) — no new palette |

## Traceability

Populated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| HUB-01 | Phase 131 | Complete |
| HUB-02 | Phase 131 | Complete |
| HUB-03 | Phase 131 | Complete |
| HUB-04 | Phase 131 | Complete |
| CARD-01 | Phase 131 | Complete |
| CARD-02 | Phase 131 | Complete |
| CARD-03 | Phase 131 | Complete |
| CARD-04 | Phase 131 | Complete |
| CARD-05 | Phase 131 | Complete |
| CARD-06 | Phase 131 | Complete |
| CARD-07 | Phase 132 | Complete |
| CARD-08 | Phase 131 | Complete |
| GRID-01 | Phase 131 | Complete |
| GRID-02 | Phase 131 | Complete |
| GRID-03 | Phase 132 | Complete |
| GRID-04 | Phase 131 | Complete |
| GRID-05 | Phase 131 | Complete |
| GRID-06 | Phase 131 | Complete |
| GRID-07 | Phase 132 | Complete |
| ATTN-01 | Phase 133 | Complete |
| ATTN-02 | Phase 133 | Complete |
| ATTN-03 | Phase 133 | Complete |
| ATTN-04 | Phase 133 | Pending |
| ATTN-05 | Phase 133 | Pending |
| ATTN-06 | Phase 133 | Complete |
| MODAL-01 | Phase 134 | Pending |
| MODAL-02 | Phase 134 | Pending |
| MODAL-03 | Phase 134 | Pending |
| MODAL-04 | Phase 134 | Pending |
| MODAL-05 | Phase 134 | Pending |
| MODAL-06 | Phase 134 | Pending |
| GROUP-01 | Phase 132 | Complete |
| GROUP-02 | Phase 132 | Complete |
| GROUP-03 | Phase 132 | Complete |
| GROUP-04 | Phase 132 | Complete |
| A11Y-01 | Phase 135 | Complete |
| A11Y-02 | Phase 135 | Pending |
| A11Y-03 | Phase 135 | Complete |
| A11Y-04 | Phase 135 | Pending |

**Coverage:**
- v3.6 requirements: 39 total (HUB ×4, CARD ×8, GRID ×7, ATTN ×6, MODAL ×6, GROUP ×4, A11Y ×4)
- Mapped to phases: 39/39 ✓
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-16*
*Last updated: 2026-06-16 — traceability table populated by roadmapper*
