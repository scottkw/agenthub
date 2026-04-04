# Requirements: AgentHub

**Defined:** 2026-04-03
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

## v1.8 Requirements

Requirements for GitHub Distribution & CI/CD milestone. Each maps to roadmap phases.

### Git Migration

- [x] **GIT-01**: Repository mirrored to GitHub (scottkw/agenthub) with full history and all tags preserved
- [x] **GIT-02**: Go module path updated from github.com/agenthub/agenthub to github.com/scottkw/agenthub with all imports rewritten

### Release Automation

- [x] **REL-01**: release-please.yml workflow creates Release PRs with auto-versioned CHANGELOG.md from conventional commits
- [ ] **REL-02**: release.yml workflow builds multi-platform artifacts on tag push (macOS signed/notarized DMG, Windows EXE + NSIS installer, Linux amd64 tar.gz + deb)
- [x] **REL-03**: Existing build.yml modified to remove macOS signing (moved to release-only), retaining tests and race detector
- [ ] **REL-04**: SHA256 checksums file generated and attached to each GitHub Release

### Distribution

- [ ] **DIST-01**: Homebrew cask tap repo (scottkw/homebrew-agenthub) with cask formula installable via `brew tap scottkw/agenthub && brew install --cask agenthub`
- [ ] **DIST-02**: distribute.yml workflow auto-updates Homebrew tap with new version and SHA256 on each release
- [ ] **DIST-03**: WinGet manifest submitted to microsoft/winget-pkgs (manual first submission, then automated via distribute.yml)
- [ ] **DIST-04**: Packaging templates in repo (packaging/homebrew/agenthub.rb.template, packaging/winget/manifests/)

## Future Requirements

### Extended Distribution

- **EDIST-01**: Linux ARM64 builds (tar.gz + deb)
- **EDIST-02**: Linux AppImage packaging
- **EDIST-03**: Scoop manifest for Windows
- **EDIST-04**: Submission to homebrew/homebrew-cask core tap
- **EDIST-05**: Windows EV code signing

## Out of Scope

| Feature | Reason |
|---------|--------|
| GoReleaser | Incompatible with Wails build system — no .app bundle or signing awareness |
| Linux ARM64 builds | Deferred — requires dedicated ARM runner; low priority for initial release |
| AppImage packaging | Deferred — tar.gz and deb cover initial Linux needs |
| Scoop manifest | Low ROI vs WinGet for Windows users |
| Core homebrew-cask submission | Requires established user base; personal tap has identical install UX |
| VERSION injection in WelcomeTab.tsx | Existing tech debt; separate concern from release automation |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| GIT-01 | Phase 44 | Complete |
| GIT-02 | Phase 44 | Complete |
| REL-01 | Phase 45 | Complete |
| REL-03 | Phase 45 | Complete |
| REL-02 | Phase 46 | Pending |
| REL-04 | Phase 46 | Pending |
| DIST-01 | Phase 47 | Pending |
| DIST-02 | Phase 47 | Pending |
| DIST-04 | Phase 47 | Pending |
| DIST-03 | Phase 48 | Pending |

**Coverage:**
- v1.8 requirements: 10 total
- Mapped to phases: 10
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-03*
*Last updated: 2026-04-03 after roadmap creation*
