---
phase: 96
plan: 04
subsystem: image-addon-frontend-integration
tags: [phase-96, image, frontend, terminal-panel, plugins-section, wave-2]
dependency_graph:
  requires:
    - phase-96-plan-01 (Wave 0 RED scaffolds + ImageConfig hand-edit)
    - phase-93-u11-01 (Unicode 11 mount-useEffect arm — structural analog)
    - phase-95-lnk-04 (web-links cleanup pattern — try/dispose mirror)
  provides:
    - ImageAddon construction in TerminalPanel.tsx MOUNT useEffect (next-session-only)
    - imageAddonRef declaration + dispose in mount-useEffect cleanup
    - enableSizeReports:false regression guard (Pitfall #8 — CSI Response Pollution)
    - storageLimit pass-through with default 16 MB cap
    - Italic 'Applies to new sessions you create.' caption under Image row in PluginsSection
    - 6 Wave 0 RED scaffolds flipped GREEN (5 TerminalPanel + 1 PluginsSection)
  affects:
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/PluginsSection.tsx
    - frontend/src/components/__tests__/TerminalPanel.test.tsx
    - frontend/src/components/__tests__/PluginsSection.test.tsx
tech-stack:
  added: []
  patterns:
    - "Phase 93 U11-01 mount-useEffect arm (next-session-only addon construction)"
    - "Phase 95 LNK-04 try/dispose cleanup mirror"
    - "Verbatim italic-caption-string sharing across next-session-only addons (UX consistency)"
key-files:
  created: []
  modified:
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/components/PluginsSection.tsx
    - frontend/src/components/__tests__/TerminalPanel.test.tsx
    - frontend/src/components/__tests__/PluginsSection.test.tsx
    - .planning/phases/96-image-addon-csp-audit/deferred-items.md
decisions:
  - "enableSizeReports:false hardcoded at construction (Pitfall #8 regression guard) — overrides plan 96-RESEARCH §Pitfall 8 'ship default true' guidance because Plan 96-04 must_haves explicitly mandate :false. Rationale: privacy-first default; if a real CLI image-rendering use case demands size reports, a future plan can flip with informed consent rather than accept the silent leak by default."
  - "Italic caption string IDENTICAL to Phase 93 unicode11 caption ('Applies to new sessions you create.') — UX consistency intentional; both addons share the same next-session-only semantic and therefore the same affordance copy."
metrics:
  duration: ~15 min
  completed_date: 2026-05-07
  tasks_completed: 2
  files_created: 0
  files_modified: 5
  commits: 2
requirements: [IMG-01, IMG-02]
---

# Phase 96 Plan 04: Frontend ImageAddon Integration Summary

**One-liner:** Wired `@xterm/addon-image` into `TerminalPanel`'s MOUNT useEffect (next-session-only invariant — explicitly NOT in the hot-swap useEffect) with `enableSizeReports: false` (Pitfall #8 CSI 14/16/18 t pixel-leak regression guard) and `storageLimit: pluginConfig?.imageConfig?.storageLimit ?? 16`, mirrored the Phase 95 web-links try/dispose cleanup pattern, and added the verbatim italic `'Applies to new sessions you create.'` caption under the Image row of `PluginsSection` — flipping all 6 Wave 0 RED scaffolds GREEN.

## What shipped

### Task 1 — TerminalPanel mount-useEffect arm + cleanup (commit `90ab869`)

**`frontend/src/components/TerminalPanel.tsx`:**

- New addon import in alphabetical position: `import { ImageAddon } from '@xterm/addon-image'` (between addon-fit and addon-unicode11).
- New mount-only ref alongside Phase 95 web-links ref:
  ```typescript
  const imageAddonRef = useRef<ImageAddon | null>(null)
  ```
- New construction block inside the `[sessionId]`-keyed MOUNT useEffect, **immediately after the Unicode 11 block** (positional invariant — RESEARCH §"Pattern 1: TerminalPanel Mount useEffect"):
  ```typescript
  if (pluginConfig?.image !== false) {
    try {
      const storageLimit = pluginConfig?.imageConfig?.storageLimit ?? 16
      const imageAddon = new ImageAddon({
        storageLimit,
        enableSizeReports: false,
      })
      term.loadAddon(imageAddon)
      imageAddonRef.current = imageAddon
    } catch (e) {
      console.warn('Phase 96 IMG-01: ImageAddon construction failed', e)
    }
  }
  ```
  - Gated by `pluginConfig?.image !== false` so default-true behavior is preserved when `pluginConfig` hasn't loaded yet (matches `defaultPluginSettings.Image = true`).
  - `enableSizeReports: false` is the Pitfall #8 regression guard. Without it, the addon registers `getWinSizePixels` / `getCellSizePixels` / `getWinSizeChars` window-options handlers; xterm.js then RESPONDS to CSI 14/16/18 t queries with terminal pixel dimensions, which leak to the running CLI as keyboard input.
  - `storageLimit ?? 16` honors AgentHub's 16 MB tab-OOM-safety default (overrides addon-image's 128 MB upstream default) while remaining configurable through Plan 96-02's sub-key RPC surface.
  - try/catch with console.warn — WASM instantiation failure is non-critical; sixel/IIP escapes pass through harmlessly as printable garbage (Discretion in 96-RESEARCH).
- New cleanup arm parallel to Phase 95 web-links inside the mount useEffect's cleanup function:
  ```typescript
  if (imageAddonRef.current) {
    try { imageAddonRef.current.dispose() } catch { /* ignore */ }
    imageAddonRef.current = null
  }
  ```
- **NOT modified:** the hot-swap useEffect (line 289-455). Its dep array does NOT include `pluginConfig?.image` or `pluginConfig?.imageConfig`. Its body does NOT reference `imageAddonRef` or construct `new ImageAddon(...)`. This is the next-session-only invariant — toggling Image in Settings does NOT re-attach on already-open terminals; the italic caption is the user-facing affordance.

**`frontend/src/components/__tests__/TerminalPanel.test.tsx`:**

5 RED `expect.fail` scaffolds replaced with real source-scan assertions over the file's existing `import raw from '../TerminalPanel.tsx?raw'` channel:

| Test | Assertion |
|------|-----------|
| imports ImageAddon from @xterm/addon-image | `raw.toContain("import { ImageAddon } from '@xterm/addon-image'")` |
| declares imageAddonRef | regex `/const\s+imageAddonRef\s*=\s*useRef<ImageAddon\s*\|\s*null>\(null\)/` |
| enableSizeReports: false (Pitfall #8 guard) | `raw.toContain('new ImageAddon(')` AND `raw.toContain('enableSizeReports: false')` |
| storageLimit pass-through with default | `raw.toContain('pluginConfig?.imageConfig?.storageLimit ?? 16')` |
| MOUNT-useEffect, NOT hot-swap (next-session-only invariant) | (a) ImageAddon construction appears within 2000 chars after `new Unicode11Addon(` — proves mount-useEffect placement; (b) iterates every dep array `}, [...])` regex match and asserts NONE reference `pluginConfig?.image` or `pluginConfig?.imageConfig` |

### Task 2 — PluginsSection italic caption (commit `f89274c`)

**`frontend/src/components/PluginsSection.tsx`:**

The existing `image` renderRow call gains the 4th `caption` argument with the literal verbatim string:

```typescript
{renderRow('image', 'Inline images',
  'Render images sent via sixel or the iTerm2 inline image protocol directly inside the terminal.',
  'Applies to new sessions you create.')}
```

The `renderRow` helper signature already supported the optional 4th `caption` parameter (Phase 92 — line 78-114); no signature change needed. The string is identical character-for-character to the Phase 93 unicode11 caption at line 130 — UX consistency is intentional because both addons are next-session-only.

**`frontend/src/components/__tests__/PluginsSection.test.tsx`:**

The IMG-01 RED `expect.fail` scaffold replaced with a positional source-scan assertion:

```typescript
const matches = raw.match(/Applies to new sessions you create\./g) || []
expect(matches.length).toBeGreaterThanOrEqual(2)  // unicode11 + image

const inlineImagesIdx = raw.indexOf("'Inline images'")
const captionAfterImage = raw.indexOf('Applies to new sessions you create.', inlineImagesIdx)
expect(captionAfterImage - inlineImagesIdx).toBeLessThan(400)
```

Two-layer assertion: (a) the literal string appears at least twice in the source (unicode11 + image — UX consistency check); (b) it appears within 400 chars after the `'Inline images'` label, proving it is the 4th argument of the image renderRow call rather than floating elsewhere.

## Verification results

```text
$ cd frontend && pnpm test src/components/__tests__/TerminalPanel.test.tsx
Test Files  1 passed (1)
     Tests  41 passed (41)
       (was 36 passed + 5 failed RED scaffolds at Plan 96-04 start)

$ cd frontend && pnpm test src/components/__tests__/PluginsSection.test.tsx
Test Files  1 passed (1)
     Tests  14 passed (14)
       (was 13 passed + 1 failed RED scaffold at Plan 96-04 start)

$ cd frontend && pnpm tsc --noEmit
src/components/FindBar/__tests__/FindBar.animation.test.tsx(15,47): error TS6133: 'beforeEach' is declared but its value is never read.
       (1 pre-existing TS6133, not introduced by Plan 96-04 — see deferred-items.md)
```

**Source-level invariant scans:**

```text
$ grep -c "import { ImageAddon } from '@xterm/addon-image'" frontend/src/components/TerminalPanel.tsx
1
$ grep -c "imageAddonRef" frontend/src/components/TerminalPanel.tsx
6
$ grep -c "new ImageAddon(" frontend/src/components/TerminalPanel.tsx
1
$ grep -c "enableSizeReports: false" frontend/src/components/TerminalPanel.tsx
1
$ grep -c "pluginConfig?.imageConfig?.storageLimit ?? 16" frontend/src/components/TerminalPanel.tsx
1
$ grep -c "imageAddonRef.current.dispose" frontend/src/components/TerminalPanel.tsx
1
$ grep -c "Applies to new sessions you create" frontend/src/components/PluginsSection.tsx
2
```

All acceptance criteria green.

## Truths mapped to tests

| Truth | Verified By |
|-------|-------------|
| ImageAddon imported | grep + `IMG-01/IMG-02` test 1 (TerminalPanel.test.tsx) |
| imageAddonRef parallel to other addon refs | regex test 2 |
| `enableSizeReports: false` (Pitfall #8 guard) | test 3 |
| `storageLimit ?? 16` pass-through | test 4 |
| MOUNT-useEffect placement, NOT hot-swap | test 5 (positional + dep-array invariant) |
| MOUNT-useEffect cleanup disposes addon | grep `imageAddonRef.current.dispose` (1 match) |
| Italic caption present, UX-consistent with unicode11 | PluginsSection.test.tsx IMG-01 caption test (matches.length >= 2 + positional check within 400 chars of 'Inline images') |

## Deviations from Plan

### Resolved naming/structure deviations

**1. [Rule 1 — Source-scan channel] PluginsSection.test.tsx uses `import raw from '...?raw'` — not `readFileSync`**

- **Found during:** Task 2 — reading the existing PluginsSection.test.tsx for the source-scan convention.
- **Issue:** Plan 96-04 Task 2 action step B specifies `readFileSync(resolve(__dirname, '../PluginsSection.tsx'), 'utf8')`, but the actual file uses Vite's `?raw` import suffix (line 2: `import raw from '../PluginsSection.tsx?raw'`). Mixing the two conventions in one file would have introduced a needless second source-read pathway.
- **Fix:** Used the existing `raw` import (already in scope at the top of the file). The assertions are otherwise identical to the plan's specified shape.
- **Files modified:** `frontend/src/components/__tests__/PluginsSection.test.tsx`
- **Commit:** `f89274c`

This is a structural alignment with the file's existing convention, not a behavioral change. Both `readFileSync` and `?raw` produce the same source string.

### Auth gates

None.

## Deferred Issues

**1. `Sidebar.test.tsx` — 20 pre-existing failures (logged in `deferred-items.md`):**

The full-suite `pnpm test` run reports 20 failing tests in `frontend/src/components/__tests__/Sidebar.test.tsx`. Stash round-trip on commit `90ab869` reproduces them with all Plan 96-04 changes stashed — confirming pre-existing. Failures appear to be a React 19 / jsdom `root.unmount()` cleanup issue during `afterEach`. Out of scope for Phase 96; deferred for a separate housekeeping triage.

**2. Pre-existing TS6133 in `FindBar.animation.test.tsx`** — already documented in Plan 96-01 SUMMARY's deferred-items section. Unchanged by Plan 96-04.

## Key decisions

- **`enableSizeReports: false` overrides 96-RESEARCH §Pitfall 8 default-`true` guidance:** Plan 96-04 must_haves explicitly mandate `enableSizeReports: false` as the Pitfall #8 regression guard. Rationale: privacy-first default — CSI 14/16/18 t pixel-dimension responses leaking to the running CLI is a measurable info-disclosure surface. If a future image-rendering tool genuinely requires size reports (e.g. some advanced sixel renderers query window pixel dims to scale), a follow-up plan can flip the flag with informed consent rather than ship the leak by default.
- **Italic caption text shared verbatim with Unicode 11:** Both addons are next-session-only; sharing the exact same caption string ('Applies to new sessions you create.') reinforces the affordance pattern in users' minds and prevents copy drift.
- **Mount-useEffect placement is structurally enforced by the IMG-01 invariant test:** Test 5 in TerminalPanel.test.tsx scans every `useEffect` dep array via the `}, [ ... ])` regex pattern and asserts NONE reference `pluginConfig?.image` or `pluginConfig?.imageConfig`. A future maintainer who tries to "fix" the next-session-only behavior by adding the field to the hot-swap useEffect dep array will hit this invariant test and be forced to read the comment explaining why.

## Threat model coverage

| Threat ID | Disposition | Mitigation Verified |
|-----------|-------------|---------------------|
| T-96-04-01 (Info Disclosure: CSI pixel-dim leak) | mitigated | `enableSizeReports: false` set at construction; source-scan test asserts the literal string presence (Test 3). |
| T-96-04-02 (Tampering: future maintainer adds image to hot-swap dep) | mitigated | Test 5 asserts no useEffect dep array references `pluginConfig?.image` / `pluginConfig?.imageConfig`. Source-scan invariant. |
| T-96-04-03 (DoS: WASM failure crashes terminal) | mitigated | Construction wrapped in try/catch with console.warn — sixel/IIP escapes pass through harmlessly. |
| T-96-04-04 (Repudiation: invisible next-session-only affordance) | mitigated | Italic caption rendered verbatim under Image row; matches existing unicode11 affordance for UX consistency. |

## Self-Check: PASSED

Files modified exist (verified via `git diff --name-only HEAD~2 HEAD`):
- FOUND: frontend/src/components/TerminalPanel.tsx
- FOUND: frontend/src/components/PluginsSection.tsx
- FOUND: frontend/src/components/__tests__/TerminalPanel.test.tsx
- FOUND: frontend/src/components/__tests__/PluginsSection.test.tsx
- FOUND: .planning/phases/96-image-addon-csp-audit/deferred-items.md

Commits exist:
- FOUND: 90ab869 (feat(96-04): wire ImageAddon into TerminalPanel mount useEffect)
- FOUND: f89274c (feat(96-04): add italic next-session-only caption under Image row)

Plan is parallel-safe with Plan 96-05 — no file overlap (96-05 touches `internal/daemon/` Wails bindings + relay byte-fidelity; 96-04 touches frontend React only).
