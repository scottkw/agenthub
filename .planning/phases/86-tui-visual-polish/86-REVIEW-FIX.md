---
phase: 86-tui-visual-polish
fixed_at: 2026-04-19T14:30:00Z
review_path: .planning/phases/86-tui-visual-polish/86-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 86: Code Review Fix Report

**Fixed at:** 2026-04-19T14:30:00Z
**Source review:** .planning/phases/86-tui-visual-polish/86-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2
- Fixed: 2
- Skipped: 0

## Fixed Issues

### WR-01: Remote Tab Uses Wrong Index Space for Selection Highlighting

**Files modified:** `internal/tui/view.go`
**Commit:** 5d4ca0e
**Applied fix:** Changed `renderRemoteTab` to pass `-1` as the `idx` argument to `renderRemoteSessionRow` instead of `len(rows)`. Since the Remote tab is a standalone view not driven by `m.unifiedList` navigation, selection highlighting does not apply. Passing `-1` ensures the `idx == m.selected` comparison in `renderRemoteSessionRow` never matches, eliminating spurious selection highlighting on remote session rows.

### WR-02: Fragile Implicit Coupling Between Sidebar Index and tabID Values

**Files modified:** `internal/tui/update.go`
**Commit:** 2478e1c
**Applied fix:** Introduced an explicit `sidebarTabs` mapping array (`var sidebarTabs = [...]tabID{tabHome, tabSessions, tabRemote, tabSettings}`) that decouples sidebar item ordering from `tabID` iota values. Replaced `tabID(m.sidebarFocus)` with `sidebarTabs[m.sidebarFocus]` in the `enter` case, and replaced the magic number `3` with `len(sidebarTabs)-1` in the `down`/`j` case for sidebar bounds checking.

---

_Fixed: 2026-04-19T14:30:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
