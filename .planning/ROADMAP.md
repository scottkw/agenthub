# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- ✅ **v1.2 Tailscale-Only Networking** — Phases 14-18 (shipped 2026-03-23)
- ✅ **v1.3 CLI + Daemon** — Phases 19-26 (shipped 2026-03-25)
- ✅ **v1.4 Unified Binary** — Phases 27-29 (shipped 2026-03-25)
- ✅ **v1.5 Bug Fixes & CLI Args** — Phases 30-34 (shipped 2026-03-26)
- ✅ **v1.6 Terminal Fill Fix v2** — Phase 35 (shipped 2026-03-31)
- ✅ **v1.7 Daemon UX & Branding** — Phases 36-43 (shipped 2026-04-03)
- 🚧 **v1.8 GitHub Distribution & CI/CD** — Phases 44-48 (in progress)

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

<details>
<summary>✅ v1.1 Polish & Build (Phases 7-13) — SHIPPED 2026-03-20</summary>

- [x] Phase 7: Layout Baseline (1/1 plans) — completed 2026-03-19
- [x] Phase 8: Per-Tab Status Bar (2/2 plans) — completed 2026-03-19
- [x] Phase 9: Settings Modal Overhaul (1/1 plans) — completed 2026-03-19
- [x] Phase 10: Per-Tab Font Size (1/1 plans) — completed 2026-03-19
- [x] Phase 11: New-Session Modal (3/3 plans) — completed 2026-03-19
- [x] Phase 12: Tab Rename + Web Dashboard (3/3 plans) — completed 2026-03-20
- [x] Phase 13: Build Script (2/2 plans) — completed 2026-03-20

</details>

<details>
<summary>✅ v1.2 Tailscale-Only Networking (Phases 14-18) — SHIPPED 2026-03-23</summary>

- [x] Phase 14: Tailscale Health Check Infrastructure (2/2 plans) — completed 2026-03-20
- [x] Phase 15: Tailscale TLS + Interface Binding (2/2 plans) — completed 2026-03-20
- [x] Phase 16: Auth Layer Removal (2/2 plans) — completed 2026-03-20
- [x] Phase 17: Dead Code Cleanup (2/2 plans) — completed 2026-03-20
- [x] Phase 18: Frontend Health Modal + Status UI (2/2 plans) — completed 2026-03-22

</details>

<details>
<summary>✅ v1.3 CLI + Daemon (Phases 19-26) — SHIPPED 2026-03-25</summary>

- [x] Phase 19: Daemon Core / Engine + IPC (2/2 plans) — completed 2026-03-23
- [x] Phase 20: Process Separation (2/2 plans) — completed 2026-03-23
- [x] Phase 21: CLI Session + Web Commands (2/2 plans) — completed 2026-03-24
- [x] Phase 22: CLI Attach (2/2 plans) — completed 2026-03-24
- [x] Phase 23: Service Manager Integration (2/2 plans) — completed 2026-03-24
- [x] Phase 24: CLI Polish (2/2 plans) — completed 2026-03-24
- [x] Phase 25: Windows Named Pipe Dial Fix (1/1 plans) — completed 2026-03-24
- [x] Phase 26: Graceful GUI Startup Failure (2/2 plans) — completed 2026-03-24

</details>

<details>
<summary>✅ v1.4 Unified Binary (Phases 27-29) — SHIPPED 2026-03-25</summary>

- [x] Phase 27: Unified Entrypoint (1/1 plans) — completed 2026-03-25
- [x] Phase 28: CLI Package Removal (1/1 plans) — completed 2026-03-25
- [x] Phase 29: Build System & Verification (1/1 plans) — completed 2026-03-25

</details>

<details>
<summary>✅ v1.5 Bug Fixes & CLI Args (Phases 30-34) — SHIPPED 2026-03-26</summary>

- [x] Phase 30: Backend Args Wiring (1/1 plans) — completed 2026-03-26
- [x] Phase 31: CLI Arg Passthrough (1/1 plans) — completed 2026-03-26
- [x] Phase 32: Daemon Startup Performance (2/2 plans) — completed 2026-03-26
- [x] Phase 33: GUI Args Field (1/1 plans) — completed 2026-03-26
- [x] Phase 34: Terminal Fill Fix (1/1 plans) — completed 2026-03-26

</details>

<details>
<summary>✅ v1.6 Terminal Fill Fix v2 (Phase 35) — SHIPPED 2026-03-31</summary>

- [x] Phase 35: Terminal Fill Fix v2 (1/1 plans) — completed 2026-03-26

</details>

<details>
<summary>✅ v1.7 Daemon UX & Branding (Phases 36-43) — SHIPPED 2026-04-03</summary>

- [x] Phase 36: App Icons & Branding Assets (1/1 plans) — completed 2026-04-01
- [x] Phase 37: Splash Screen (1/1 plans) — completed 2026-04-01
- [x] Phase 38: Remote Session Metadata (1/1 plans) — completed 2026-04-01
- [x] Phase 39: Remote Session Indicators (2/2 plans) — completed 2026-04-01
- [x] Phase 40: Daemon Management Panel (1/1 plans) — completed 2026-04-02
- [x] Phase 41: System Tray + Lifecycle (2/2 plans) — completed 2026-04-02
- [x] Phase 42: Tray Startup-Failure Error Icon (1/1 plans) — completed 2026-04-03
- [x] Phase 43: GUI Hostname Forwarding (1/1 plans) — completed 2026-04-03

</details>

### v1.8 GitHub Distribution & CI/CD (In Progress)

**Milestone Goal:** Move AgentHub from Gitea to GitHub with automated multi-platform release builds, Homebrew tap, and WinGet distribution.

- [x] **Phase 44: Git Migration to GitHub** - Mirror Gitea repo to GitHub with full history, all tags, and updated Go module path (completed 2026-04-04)
- [x] **Phase 45: release-please + CI Signing Removal** - Automated versioning via release-please.yml; remove macOS signing from build.yml (completed 2026-04-04)
- [x] **Phase 46: Release Build Pipeline** - Tag-triggered multi-platform release.yml with macOS signing/notarization and SHA256 checksums (completed 2026-04-04)
- [ ] **Phase 47: Homebrew Tap + Packaging Templates** - Tap repo, cask formula, packaging templates, and Homebrew leg of distribute.yml
- [ ] **Phase 48: WinGet Distribution** - WinGet manifests, manual first submission, and WinGet leg of distribute.yml

## Phase Details

### Phase 44: Git Migration to GitHub
**Goal**: The GitHub repository scottkw/agenthub exists with complete Gitea history, all v1.0-v1.7 tags intact, and the Go module path updated so builds reference the correct canonical import path
**Depends on**: Nothing (first phase of v1.8)
**Requirements**: GIT-01, GIT-02
**Success Criteria** (what must be TRUE):
  1. `git clone https://github.com/scottkw/agenthub` succeeds and contains the full commit history from v1.0 through v1.7
  2. All v1.0-v1.7 release tags (e.g., v1.0.0, v1.7.0) are visible on the GitHub Releases page or via `git tag` after clone
  3. The Go module path in `go.mod` reads `github.com/scottkw/agenthub` and all internal imports are rewritten to match
  4. `go build ./...` succeeds with no import path errors after the module path change
  5. Existing CI secrets from Gitea are confirmed present in GitHub repository settings (7 macOS signing secrets)
**Plans**: 2 plans
Plans:
- [x] 44-01-PLAN.md -- Go module path rewrite (go.mod + 16 .go files)
- [x] 44-02-PLAN.md -- GitHub repo creation, mirror push, secrets migration

### Phase 45: release-please + CI Signing Removal
**Goal**: Merging conventional-commit PRs to main automatically creates a Release PR with updated CHANGELOG.md and version bump; macOS signing no longer runs on every PR build, saving notarization quota and runner minutes
**Depends on**: Phase 44 (GitHub repo with tags as baseline for release-please)
**Requirements**: REL-01, REL-03
**Success Criteria** (what must be TRUE):
  1. After merging a conventional-commit PR to main, a Release PR appears automatically with an updated CHANGELOG.md and bumped version
  2. The Release PR shows the correct next SemVer (e.g., v1.8.0) derived from commit type (feat/fix/chore)
  3. `build.yml` no longer runs macOS code signing or notarization steps — PR builds complete without touching Apple APIs
  4. `build.yml` still runs the full test matrix (Go race detector, build-script tests) and produces build artifacts for validation
  5. `release-please-config.json` and `.release-please-manifest.json` are committed to the repo and track the current version
**Plans**: 2 plans
Plans:
- [x] 45-01-PLAN.md -- Create release-please workflow + config files, remove signing from build.yml
- [x] 45-02-PLAN.md -- Configure PAT secret, verify end-to-end Release PR creation

### Phase 46: Release Build Pipeline
**Goal**: Merging a Release PR (created by release-please) produces a GitHub Release with multi-platform signed artifacts and a SHA256 checksums file that users and package managers can verify
**Depends on**: Phase 45 (release-please creates the tags that trigger release.yml)
**Requirements**: REL-02, REL-04
**Success Criteria** (what must be TRUE):
  1. Merging a release-please Release PR triggers `release.yml` automatically and produces artifacts for macOS (signed/notarized DMG), Windows (NSIS installer + bare EXE), and Linux (amd64 tar.gz + deb)
  2. The macOS DMG is signed and notarized — macOS Gatekeeper allows installation without a security override prompt
  3. A `checksums.txt` file containing SHA256 hashes for all artifacts is attached to the GitHub Release alongside the binaries
  4. Artifact filenames follow a consistent naming convention (e.g., `agenthub-v1.8.0-darwin-universal.dmg`) that downstream package managers can rely on
  5. The GitHub Release page shows all artifacts available for download immediately after the workflow completes
**Plans**: 1 plan
Plans:
- [x] 46-01-PLAN.md -- Create release.yml with macOS/Windows/Linux build jobs and publish job

### Phase 47: Homebrew Tap + Packaging Templates
**Goal**: macOS users can install AgentHub via `brew install --cask agenthub` using the scottkw/homebrew-agenthub tap; each new release automatically updates the cask formula; packaging templates for both Homebrew and WinGet are committed to the main repo
**Depends on**: Phase 46 (artifact names and macOS DMG URL format locked by release.yml)
**Requirements**: DIST-01, DIST-02, DIST-04
**Success Criteria** (what must be TRUE):
  1. `brew tap scottkw/agenthub && brew install --cask agenthub` installs a working AgentHub app on a clean macOS machine
  2. After a new GitHub Release is published, the Homebrew cask formula in scottkw/homebrew-agenthub is updated automatically (version and SHA256) without manual intervention
  3. The `packaging/homebrew/agenthub.rb.template` file in the main repo contains a complete, renderable cask formula template with placeholder tokens for version and SHA256
  4. The `packaging/winget/manifests/` directory in the main repo contains the three-file WinGet manifest set (version, installer, locale) matching WinGet schema 1.12.0
  5. The `distribute.yml` Homebrew job includes retry logic for the macOS asset SHA256 download to handle the notarization delay after release
**Plans**: TBD

### Phase 48: WinGet Distribution
**Goal**: Windows users can install AgentHub via `winget install AgentHub.AgentHub`; the package identity is established in microsoft/winget-pkgs via a manual first submission, and subsequent releases are submitted automatically
**Depends on**: Phase 46 (Windows NSIS installer artifact available on GitHub Release), Phase 47 (packaging/winget/ templates committed to repo)
**Requirements**: DIST-03
**Success Criteria** (what must be TRUE):
  1. `winget install AgentHub.AgentHub` installs AgentHub on a Windows machine (confirms manual first submission was accepted by Microsoft)
  2. The distribute.yml WinGet job runs `winget-releaser` after each new release and automatically submits a manifest PR to microsoft/winget-pkgs
  3. The WINGET_TOKEN secret (classic PAT with public_repo scope) is stored in GitHub repository settings and used by distribute.yml
  4. `winget validate` passes against the submitted manifests before the first manual PR is opened
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-6 | v1.0 | 19/19 | Complete | 2026-03-19 |
| 7-13 | v1.1 | 13/13 | Complete | 2026-03-20 |
| 14-18 | v1.2 | 10/10 | Complete | 2026-03-23 |
| 19-26 | v1.3 | 15/15 | Complete | 2026-03-25 |
| 27-29 | v1.4 | 3/3 | Complete | 2026-03-25 |
| 30-34 | v1.5 | 6/6 | Complete | 2026-03-26 |
| 35 | v1.6 | 1/1 | Complete | 2026-03-31 |
| 36-43 | v1.7 | 10/10 | Complete | 2026-04-03 |
| 44. Git Migration to GitHub | v1.8 | 2/2 | Complete    | 2026-04-04 |
| 45. release-please + CI Signing Removal | v1.8 | 2/2 | Complete    | 2026-04-04 |
| 46. Release Build Pipeline | v1.8 | 1/1 | Complete    | 2026-04-04 |
| 47. Homebrew Tap + Packaging Templates | v1.8 | 0/? | Not started | - |
| 48. WinGet Distribution | v1.8 | 0/? | Not started | - |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
*Full v1.2 details: .planning/milestones/v1.2-ROADMAP.md*
*Full v1.3 details: .planning/milestones/v1.3-ROADMAP.md*
*Full v1.4 details: .planning/milestones/v1.4-ROADMAP.md*
*Full v1.5 details: .planning/milestones/v1.5-ROADMAP.md*
*Full v1.6 details: .planning/milestones/v1.6-ROADMAP.md*
*Full v1.7 details: .planning/milestones/v1.7-ROADMAP.md*
*Full v1.8 details: .planning/milestones/v1.8-ROADMAP.md*
