---
phase: 36
slug: app-icons-branding-assets
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-31
---

# Phase 36 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | bash + ImageMagick identify + iconutil --list |
| **Config file** | none — CLI tools already installed |
| **Quick run command** | `identify build/appicon.png && iconutil -l build/darwin/AppIcon.icns 2>/dev/null` |
| **Full suite command** | `bash scripts/verify-icons.sh` (created in Wave 0) |
| **Estimated runtime** | ~3 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick verify (identify + file checks)
- **After every plan wave:** Run full verification script
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 36-01-01 | 01 | 1 | BRND-01 | file+identify | `identify build/appicon.png \| grep "1024x1024"` | ❌ W0 | ⬜ pending |
| 36-01-02 | 01 | 1 | BRND-01 | iconutil | `iconutil -l build/darwin/AppIcon.icns 2>/dev/null \| wc -l` | ❌ W0 | ⬜ pending |
| 36-01-03 | 01 | 1 | BRND-01 | identify | `identify build/windows/icon.ico \| wc -l` (>=4 sizes) | ❌ W0 | ⬜ pending |
| 36-01-04 | 01 | 1 | BRND-01 | file | `ls build/linux/icon_*.png \| wc -l` (>=4 files) | ❌ W0 | ⬜ pending |
| 36-01-05 | 01 | 1 | BRND-01 | file | `test -f frontend/src/assets/agenthub-title-logo.png` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `scripts/verify-icons.sh` — verification script checking all icon formats and sizes
- [ ] Existing `identify` (ImageMagick) and `iconutil` (macOS built-in) confirmed available

*Existing CLI tools cover all verification needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| macOS .app icon visible in Finder/Dock | BRND-01 | Requires GUI display to verify visual rendering | Run `wails build`, open Finder, check .app icon visually |
| Icon visual quality (no artifacts, correct colors) | BRND-01 | Requires human visual inspection | Open each generated icon file and verify brand colors/quality |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
