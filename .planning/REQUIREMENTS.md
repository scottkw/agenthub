# Requirements: AgentHub — v4.0 Hub-First Consolidation & UI/UX Overhaul

**Defined:** 2026-06-19
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v4.0 Requirements

Requirements for the v4.0 milestone. Each maps to a roadmap phase (Phase 136+).

### Navigation Restructure (NAV)

- [ ] **NAV-01**: The TUI surface is removed entirely — `agenthub tui` no longer exists; Bubble Tea views, TUI-only shared code, and their tests are deleted. Cross-surface parity contract narrows to GUI/CLI/web.
- [ ] **NAV-02**: The "+ New Session" sidebar item is removed; users create sessions from the Hub.
- [ ] **NAV-03**: The "Sessions" sidebar page (`DaemonManagerPanel`) is removed.
- [ ] **NAV-04**: The "Remote" sidebar page is removed (the Hub already unifies local + remote sessions).
- [ ] **NAV-05**: The sidebar contains exactly Home / Hub / Settings.

### Share Modal (SHARE)

- [ ] **SHARE-01**: Each Hub session card has a "Share" button that opens a per-session Share modal.
- [ ] **SHARE-02**: The Share modal has a "Share the session" toggle (replacing "Web On"); when on, it reveals two share links/codes — one read-only and one read/write.
- [ ] **SHARE-03**: The Share modal has an "Enable remote file browsing" toggle; file-browse permission inherits from the share code the visitor presents (read-only code → read-only browse; read/write code → read/write browse).
- [ ] **SHARE-04**: Share links/codes are copyable, each has a QR code, and the LAN Basic Auth password surfaces in the modal when the web server runs in local mode.
- [ ] **SHARE-05**: The Share modal carries forward every per-session web-share capability the removed Sessions page provided (web-serve on/off, URLs, QR, password) with no regression, including the existing cap/URL/QR lifecycle behavior (off→on cache-clear, stale-URL cleanup, server-truth seeding).
- [ ] **SHARE-06**: Sharing controls are unavailable (hidden or disabled) on remote peer cards — a user cannot re-share a session they do not own.

### Hub Card (CARD)

- [ ] **CARD-01**: The Hub `.hub__header` (the "Hub" title bar and its duplicate "New session" button) is removed; the `HubFilterBar` "New Session" button is the sole top-of-page creation entry point.
- [ ] **CARD-02**: Each card indicates whether its session is local or remote.
- [ ] **CARD-03**: Remote cards indicate whether the session is merely available or one the user is currently connected to — conveyed colorblind-safe (icon/text/shape, never color alone).
- [ ] **CARD-04**: The session card is resized/redesigned to accommodate the Share button and indicators while preserving attention pulse/float-to-top, mini-preview, grid density, and responsive reflow.
- [ ] **CARD-05**: Mini-preview cards and the briefing-modal tail render agent (TUI-style) output legibly (#96) — correct column spacing and no leaked escape sequences (headless VT render of scrollback, not regex ANSI strip).

### UI/UX Redesign (RDS)

- [ ] **RDS-01**: A redesign direction (or an explicit mix) from `./agenthub-v4.0-redesign` is chosen and documented at UI-spec time, after browser review of the standalone HTML.
- [ ] **RDS-02**: The chosen redesign is implemented across all surviving surfaces (Welcome, Hub, terminal/session, File Browser, Editor, Settings).
- [ ] **RDS-03**: The redesign is reconciled with the Hub-first structure (no Sessions/Remote sidebar pages; creation on the Hub); structural decisions (NAV/SHARE/CARD) win conflicts with the older comps.
- [ ] **RDS-04**: The redesign honors colorblind-safe semantics and `prefers-reduced-motion` throughout (light + dark).

### Tab Strip (TAB)

- [ ] **TAB-01**: Open tabs shrink as their count grows (browser-style), down to a sensible minimum width.
- [ ] **TAB-02**: When tabs overflow the window width, a visible side-scroll affordance (scroll chevrons and/or a visible scrollbar) lets the user reach every tab.
- [ ] **TAB-03**: Tab close, rename, and progress-underline affordances remain functional at the minimum tab width.

### Regression Testing (TEST)

- [ ] **TEST-01**: An automated regression suite (Go + vitest + Playwright) is consolidated and labeled, with a requirement→test traceability map.
- [ ] **TEST-02**: The automated regression suite runs in CI as a merge gate.
- [ ] **TEST-03**: Automated coverage gaps are closed for the Hub and for cross-surface GUI/CLI/web flows.
- [ ] **TEST-04**: A single maintained manual (human-intervention) regression checklist exists, replacing the scattered per-phase UAT logs.
- [ ] **TEST-05**: A standing convention requires every future phase to add its regression tests to the appropriate group (automated vs. human-intervention).
- [ ] **TEST-06**: TUI tests are removed (not migrated) as part of the TUI removal.

### v3.6 Carry-Overs (CARRY)

- [ ] **CARRY-01**: The Hub GroupSidebar ARIA model is made internally consistent (#97) — either the listbox roving-tabindex / `aria-activedescendant` pattern is implemented, or the `listbox`/`option` roles are dropped in favor of a plain focusable control list.
- [ ] **CARRY-02**: The deferred #78 Hub-fidelity backlog (#93) is triaged at planning; the in-scope subset is delivered and the remainder is explicitly re-deferred (with #93 updated).

## Future Requirements

Deferred beyond v4.0. Tracked, not in this roadmap.

### Deferred #78 Hub fidelity (#93 — triage at planning, most likely re-deferred)

- Per-session usage metrics on cards (tokens / spend / context-window %) — overlaps #67 Agent Dashboard
- Formal "projects" model (first-class projects replacing working-directory grouping)
- Member / collaborator avatars + presence
- Structured "agent suggests" briefings (structured decision data vs. live terminal tail)
- Session-detail / chat-thread page
- Tweaks panel (design-tooling)

### Existing backlog (not pulled into v4.0)

- #79 Session Chat · #10 Intersession orchestration · #49 Split panes · #69 In-app Help page · #65 Antigravity CLI agent · #50 Local Gemma agent · #68 Tab down-chevron · #66 Marketing screenshot automation · #59 Contributor page · #51 Shell-sharing-warning toggle · #67 Agent Dashboard

## Out of Scope

| Feature | Reason |
|---------|--------|
| TUI surface in any form | Removed in v4.0 (NAV-01); `agenthub tui` retired, parity is GUI/CLI/web |
| TUI Hub parity (#82) | Moot — TUI removed; issue closed not-planned |
| Mobile app (#30) | Desktop + web access only; PWA via web serving covers mobile |
| Redesign-direction faithful/pixel port of fields the backend can't supply | Hub-first + real-data adaptation wins; pixel fidelity tracked under #93 |

## Traceability

Which phases cover which requirements. Filled during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| NAV-01..05 | TBD | Pending |
| SHARE-01..06 | TBD | Pending |
| CARD-01..05 | TBD | Pending |
| RDS-01..04 | TBD | Pending |
| TAB-01..03 | TBD | Pending |
| TEST-01..06 | TBD | Pending |
| CARRY-01..02 | TBD | Pending |

**Coverage:**
- v4.0 requirements: 31 total (NAV 5, SHARE 6, CARD 5, RDS 4, TAB 3, TEST 6, CARRY 2)
- Mapped to phases: 0 (roadmap pending)
- Unmapped: 31 ⚠️ (filled by roadmapper)

---
*Requirements defined: 2026-06-19 — v4.0 Hub-First Consolidation & UI/UX Overhaul*
*Last updated: 2026-06-19 (initial definition)*
