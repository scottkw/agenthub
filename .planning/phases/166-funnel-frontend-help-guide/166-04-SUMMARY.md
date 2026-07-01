---
phase: 166-funnel-frontend-help-guide
plan: "04"
subsystem: frontend-help
tags: [help-article, navigation, tdd, HLP-01, HLP-02]
status: complete
completed_at: "2026-07-01T00:39:06Z"
duration: "~3 minutes"

dependency_graph:
  requires:
    - "frontend/src/content/help/chat.md (prose style analog)"
    - "frontend/src/components/HelpContent.tsx (markdown link → BrowserOpenURL conversion)"
  provides:
    - "frontend/src/content/help/sharing-guide.md (HLP-01/HLP-02 article)"
    - "help-sharing section in HelpTab SECTION_META + HelpSectionNav SECTIONS"
  affects:
    - "HelpTab section count: 3→4"
    - "HelpSectionNav nav button list"

tech_stack:
  added: []
  patterns:
    - "?raw markdown import via Vite"
    - "TDD RED/GREEN gate for component integration"
    - "SECTION_META + SECTIONS dual-array sync (Pitfall 5)"

key_files:
  created:
    - frontend/src/content/help/sharing-guide.md
  modified:
    - frontend/src/components/HelpTab.tsx
    - frontend/src/components/HelpSectionNav.tsx
    - frontend/src/components/__tests__/HelpTab.integration.test.tsx

decisions:
  - "Both SECTION_META (HelpTab) and SECTIONS (HelpSectionNav) updated in same commit to prevent nav/content desync (Pitfall 5)"
  - "Plain markdown links throughout article — HelpContent auto-converts to BrowserOpenURL; no raw <a> per Wails CSP constraint"
  - "ACL grant targets tag:agenthub:tcp:7443 with autogroup:shared src — narrowest possible grant for shared users"
  - "Wildcard-default gotcha included verbatim per UI-SPEC copywriting contract"

metrics:
  duration: "~3 minutes"
  completed_date: "2026-07-01"
  tasks: 2
  files: 4
---

# Phase 166 Plan 04: Sharing Guide Help Article Summary

Added the in-app "Sharing Outside Your Tailnet" Help article covering both the Tailscale Funnel path and the device-share + ACL path, registered in both Help nav arrays with a copy-pasteable ACL grant and the wildcard-default gotcha.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Author sharing-guide.md (both paths, ACL block, wildcard gotcha) | ee22a2d0 | frontend/src/content/help/sharing-guide.md |
| 2 (TDD) | Register help-sharing in BOTH nav arrays + integration test | 83692db2 (RED), b26131d7 (GREEN) | HelpTab.tsx, HelpSectionNav.tsx, HelpTab.integration.test.tsx |

## Verification Results

- `pnpm test -- HelpTab.integration`: 14/14 pass (section count 3→4, #help-sharing DOM presence, ordering chat→sharing→faq, nav button, nav click scrollIntoView)
- `npx tsc --noEmit`: clean (no errors)
- `grep -c 'tag:agenthub:tcp:7443' sharing-guide.md`: 1
- `grep -c 'autogroup:shared' sharing-guide.md`: 2
- `grep -c '*→*' sharing-guide.md`: 2 (wildcard gotcha present)
- `grep -c '<a ' sharing-guide.md`: 0 (no raw `<a>` tags)
- 4 Tailscale KB doc links as plain markdown (Funnel/ACL/sharing/tags)
- help-sharing entry present in both SECTION_META (HelpTab.tsx) and SECTIONS (HelpSectionNav.tsx) at same position

## TDD Gate Compliance

- RED gate: `test(166-04)` commit 83692db2 — 6 new tests fail, 8 existing pass
- GREEN gate: `feat(166-04)` commit b26131d7 — 14/14 pass
- REFACTOR: not required; code is clean as-is

## Deviations from Plan

None — plan executed exactly as written. Both nav arrays updated in sync per Pitfall 5. Test file structure issue (orphan `})`) caught and fixed before RED commit.

## Threat Surface Scan

No new network endpoints, auth paths, or trust boundary changes introduced. The article's external links (4 Tailscale KB URLs) pass through `HelpContent.tsx`'s existing `SAFE_LINK_SCHEME` validation before `BrowserOpenURL` — T-166-08 (spoofing via doc links) is covered by the existing HelpContent gate. T-166-09 (ACL grant too broad) is mitigated: article ships the narrow `autogroup:shared → tag:agenthub:tcp:7443` grant AND includes the explicit wildcard-default `*→*` gotcha with "verify your full ACL after editing" caveat.

## Known Stubs

None. The article is fully authored content with no placeholders.

## Self-Check: PASSED

All files exist on disk and all commits are present in git log:
- FOUND: frontend/src/content/help/sharing-guide.md
- FOUND: frontend/src/components/HelpTab.tsx (modified)
- FOUND: frontend/src/components/HelpSectionNav.tsx (modified)
- FOUND: frontend/src/components/__tests__/HelpTab.integration.test.tsx (modified)
- COMMIT ee22a2d0: docs(166-04): author sharing-guide.md
- COMMIT 83692db2: test(166-04): RED gate
- COMMIT b26131d7: feat(166-04): GREEN gate
