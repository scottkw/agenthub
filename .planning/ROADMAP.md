# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- 🚧 **v1.1 Polish & Build** — Phases 7-13 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) — SHIPPED 2026-03-19</summary>

- [x] Phase 1: PTY Foundation (2/2 plans) — completed 2026-03-18
- [x] Phase 2: Session Registry + WebSocket Relay (2/2 plans) — completed 2026-03-18
- [x] Phase 3: Wails Desktop UI (3/3 plans) — completed 2026-03-18
- [x] Phase 4: Web Serving + TLS + Auth (4/4 plans) — completed 2026-03-18
- [x] Phase 5: QR Codes + Status Indicators (6/6 plans) — completed 2026-03-18
- [x] Phase 6: Distribution + Cross-Platform (2/2 plans) — completed 2026-03-19

</details>

### 🚧 v1.1 Polish & Build (In Progress)

**Milestone Goal:** Improve UI/UX across desktop and web, fix terminal sizing, and add a build script for cross-platform compilation with signing.

- [x] **Phase 7: Layout Baseline** — Terminal fill and toolbar sizing foundation for all subsequent UI work (completed 2026-03-19)
- [x] **Phase 8: Per-Tab Status Bar** — Permanent status strip replaces floating web-serving overlay (completed 2026-03-19)
- [x] **Phase 9: Settings Modal Overhaul** — Tabbed layout declutters and reorganizes settings (completed 2026-03-19)
- [x] **Phase 10: Per-Tab Font Size** — Keyboard shortcuts to adjust font size per terminal tab (completed 2026-03-19)
- [ ] **Phase 11: New-Session Modal** — Agent picker and native folder browser replace bare CLI picker
- [ ] **Phase 12: Tab Rename + Web Dashboard** — Tab renames propagate to web dashboard; dashboard visual refresh
- [ ] **Phase 13: Build Script** — One-command cross-platform compilation with macOS signing and notarization

## Phase Details

### Phase 7: Layout Baseline
**Goal**: Terminals fill all available space and toolbar buttons are easy to click
**Depends on**: Phase 6 (v1.0 shipped)
**Requirements**: TERM-01, UILAY-01
**Success Criteria** (what must be TRUE):
  1. Terminal content fills the full available vertical height in every tab with no blank dead space below the output
  2. Toolbar buttons are visually larger (36-44px hit target) and comfortable to click without precise aim
  3. Adding a new tab or switching tabs does not cause layout collapse or incorrect terminal sizing
**Plans:** 1/1 plans complete
Plans:
- [ ] 07-01-PLAN.md — Fix terminal flex chain, enlarge toolbar buttons, add test stubs

### Phase 8: Per-Tab Status Bar
**Goal**: Each tab has a permanent status strip at the bottom replacing the floating header overlay
**Depends on**: Phase 7
**Requirements**: UILAY-02, UILAY-03
**Success Criteria** (what must be TRUE):
  1. A fixed-height status bar is visible at the bottom of every tab showing web serving state, session URL, and controls
  2. The floating web-status header overlay is absent from the terminal content area
  3. The terminal content area fills the remaining space above the status bar with no dead space
  4. Status bar layout is correct on macOS, Linux, and Windows (WebView2)
**Plans:** 2/2 plans complete
Plans:
- [ ] 08-01-PLAN.md — Create StatusBar component, unit tests, and CSS rules
- [ ] 08-02-PLAN.md — Wire StatusBar into App.tsx, remove old overlay, clean up CSS

### Phase 9: Settings Modal Overhaul
**Goal**: Settings modal uses a tabbed layout that organizes options clearly without crowding
**Depends on**: Phase 7
**Requirements**: SETT-01, SETT-02
**Success Criteria** (what must be TRUE):
  1. Settings modal displays tabs (e.g., "CLI Paths" and "Web Serving") that switch between distinct option groups
  2. Each tab shows only its own settings — no options from other tabs are visible at the same time
  3. Modal has a single "Close" footer with improved visual styling and spacing
**Plans:** 1/1 plans complete
Plans:
- [ ] 09-01-PLAN.md — Add tab bar, conditional tab rendering, inline Save Paths, single Close footer

### Phase 10: Per-Tab Font Size
**Goal**: Users can adjust font size in any terminal tab using keyboard shortcuts, with size persisted per tab
**Depends on**: Phase 7, Phase 8
**Requirements**: TERM-02, TERM-03, TERM-04
**Success Criteria** (what must be TRUE):
  1. Pressing SHIFT+ in an active terminal tab increases the font size visibly
  2. Pressing SHIFT- in an active terminal tab decreases the font size visibly
  3. Holding SHIFT+= for several seconds does not inject plus characters into the terminal shell session
  4. After a font size change, the terminal correctly reflows to fill its container with no garbled or clipped output
  5. Each tab retains its own font size independently when switching between tabs
**Plans:** 1/1 plans complete
Plans:
- [ ] 10-01-PLAN.md — TDD: source-inspection tests + implement per-tab font size with key handler and state

### Phase 11: New-Session Modal
**Goal**: Creating a new session opens a full modal with agent picker, folder browser, and last-folder memory
**Depends on**: Phase 7
**Requirements**: SESS-01, SESS-02, SESS-03, SESS-04
**Success Criteria** (what must be TRUE):
  1. Clicking the + button opens a modal (not a dropdown) for creating a new session
  2. The modal lists available AI coding CLIs for selection as the agent
  3. The modal includes a button that opens a native OS folder browser to select the working directory
  4. The folder browser defaults to the last-used folder on subsequent opens, or home directory on first use
**Plans**: TBD

### Phase 12: Tab Rename + Web Dashboard
**Goal**: Tab renames propagate to the web dashboard session list, and the web dashboard has a refreshed visual design
**Depends on**: Phase 8, Phase 11
**Requirements**: UILAY-04, UILAY-05, WEBUI-01, WEBUI-02
**Success Criteria** (what must be TRUE):
  1. Double-clicking or right-clicking a tab label allows the user to rename that tab
  2. A renamed tab's name appears as the session name in the web dashboard (not the raw session ID)
  3. The web dashboard displays sessions in a visually improved layout with status color indicators and CLI badges
  4. New sessions created via the new-session modal appear with their chosen name in the web dashboard
**Plans**: TBD

### Phase 13: Build Script
**Goal**: A single build.sh script compiles the app for any platform, with macOS signing and notarization support
**Depends on**: Phase 12
**Requirements**: BUILD-01, BUILD-02, BUILD-03, BUILD-04, BUILD-05
**Success Criteria** (what must be TRUE):
  1. Running `build.sh --platform macos` produces a macOS binary without errors
  2. Running `build.sh --platform linux` produces a Linux binary without errors
  3. Running `build.sh --platform windows` produces a Windows binary without errors
  4. Running `build.sh --all` produces binaries for all three platforms sequentially
  5. Running `build.sh --platform macos --sign` produces a code-signed, notarized macOS build that passes `spctl --assess` verification
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. PTY Foundation | v1.0 | 2/2 | Complete | 2026-03-18 |
| 2. Session Registry + WebSocket Relay | v1.0 | 2/2 | Complete | 2026-03-18 |
| 3. Wails Desktop UI | v1.0 | 3/3 | Complete | 2026-03-18 |
| 4. Web Serving + TLS + Auth | v1.0 | 4/4 | Complete | 2026-03-18 |
| 5. QR Codes + Status Indicators | v1.0 | 6/6 | Complete | 2026-03-18 |
| 6. Distribution + Cross-Platform | v1.0 | 2/2 | Complete | 2026-03-19 |
| 7. Layout Baseline | 1/1 | Complete   | 2026-03-19 | - |
| 8. Per-Tab Status Bar | 2/2 | Complete   | 2026-03-19 | - |
| 9. Settings Modal Overhaul | 1/1 | Complete   | 2026-03-19 | - |
| 10. Per-Tab Font Size | 1/1 | Complete    | 2026-03-19 | - |
| 11. New-Session Modal | v1.1 | 0/? | Not started | - |
| 12. Tab Rename + Web Dashboard | v1.1 | 0/? | Not started | - |
| 13. Build Script | v1.1 | 0/? | Not started | - |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*v1.1 roadmap created: 2026-03-19*
