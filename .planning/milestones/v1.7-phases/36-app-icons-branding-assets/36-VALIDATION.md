---
phase: 36
slug: app-icons-branding-assets
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-31
audited: 2026-04-02
---

# Phase 36 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | bash + ImageMagick identify + iconutil |
| **Config file** | none — CLI tools already installed |
| **Quick run command** | `identify build/appicon.png && test -s build/darwin/iconfile.icns` |
| **Full suite command** | `bash scripts/verify-icons.sh` |
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
| 36-01-01 | 01 | 1 | BRND-01 | file+identify | `identify build/appicon.png \| grep "1024x1024"` | ✅ | ✅ green |
| 36-01-02 | 01 | 1 | BRND-01 | file+size | `test -s build/darwin/iconfile.icns && test $(stat -f%z build/darwin/iconfile.icns) -gt 100000` | ✅ | ✅ green |
| 36-01-03 | 01 | 1 | BRND-01 | identify | `identify build/windows/icon.ico \| wc -l` (>=4 sizes) | ✅ | ✅ green |
| 36-01-04 | 01 | 1 | BRND-01 | file | `ls build/linux/{16,32,48,128,256,512}x*.png \| wc -l` (6 files) | ✅ | ✅ green |
| 36-01-05 | 01 | 1 | BRND-01 | file | `test -f frontend/src/assets/agenthub-title-logo.png` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `scripts/verify-icons.sh` — verification script checking all icon formats and sizes (13 checks, all pass)
- [x] Existing `identify` (ImageMagick) and `iconutil` (macOS built-in) confirmed available

*Existing CLI tools cover all verification needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| macOS .app icon visible in Finder/Dock | BRND-01 | Requires GUI display to verify visual rendering | Run `wails build`, open Finder, check .app icon visually |
| Icon visual quality (no artifacts, correct colors) | BRND-01 | Requires human visual inspection | Open each generated icon file and verify brand colors/quality |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved

---

## Validation Audit 2026-04-02

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

**Notes:**
- All 13 automated checks in `scripts/verify-icons.sh` pass (13/13 green)
- All 5 requirement verification tasks covered by automated commands
- Bundle ICNS size mismatch (590KB source vs 362KB bundle) is expected — post-build injection is a documented manual step after each `wails build`
- 2 manual-only verifications remain (visual GUI inspection) — appropriate for branding/icon quality
