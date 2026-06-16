---
phase: 130
slug: remote-browse-gui-on-ramp
audited: 2026-06-16
baseline: 130-UI-SPEC.md (approved 2026-06-15)
screenshots: not captured (no dev server detected on ports 3000, 5173, 8080)
---

# Phase 130 — UI Review

**Audited:** 2026-06-16
**Baseline:** 130-UI-SPEC.md (approved, status=approved)
**Screenshots:** Not captured — no dev server detected on ports 3000, 5173, or 8080. Audit is code-only.

---

## Pillar Scores

| Pillar | Score | Key Finding |
|--------|-------|-------------|
| 1. Copywriting | 3/4 | "Browse files" button label uses lowercase 'f'; spec mandates "Browse Files". Error state copy ("Could not load sessions") is absent from the component. |
| 2. Visuals | 3/4 | All three per-peer states are text-labeled and structurally distinct; focus-visible outline mandated by spec is missing on `.remote-panel__btn`. |
| 3. Color | 4/4 | Zero new hex values; all four new CSS classes use only existing TokyoNight tokens. Colorblind contract honored: "Unreachable" text is the primary signal, #f7768e is reinforcement only. |
| 4. Typography | 4/4 | New classes use only 11px/600 and 13px/400 from the declared scale. No out-of-spec sizes or weights introduced in Phase 130 rules. |
| 5. Spacing | 3/4 | New Phase 130 classes are spec-compliant (2px/4px/8px). Pre-existing panel values (gap: 10px, padding: 10px 12px, gap: 6px, margin-bottom: 20px) fall outside the 4-point scale but are inherited from Phase 52 — not Phase 130 regressions. |
| 6. Experience Design | 2/4 | Error state entirely absent: `catch` block silently clears loading, no `remoteError` state, no error branch in `RemoteSessionsPanel`. The spec requires "Could not load sessions" heading + body. RPC poll error is invisible to the user. |

**Overall: 19/24**

---

## Top 3 Priority Fixes

1. **Missing error state in `RemoteSessionsPanel`** — When `GetRemoteSessionsWithMeta()` throws (network drop, Wails RPC failure), the catch block at `App.tsx:913` sets `remoteLoading(false)` but leaves `remotePeers` empty and renders nothing. The user sees a blank panel with no explanation. Fix: add `const [remoteError, setRemoteError] = useState(false)` in App.tsx, set it in the catch block, clear it on success; add an `error?: boolean` prop to `RemoteSessionsPanel` and render `<div class="remote-panel__error-title">Could not load sessions</div>` + `<div class="remote-panel__error-body">An error occurred loading remote sessions. Check your tailnet connection.</div>` when `error` is true and `peers.length === 0`.

2. **Missing `focus-visible` outline on `.remote-panel__btn`** — The UI-SPEC Accessibility Contract §7 mandates `outline: 2px solid #7aa2f7; outline-offset: 2px` on all interactive elements. No `:focus-visible` rule exists for `.remote-panel__btn` in `style.css`, and the global CSS reset does not remove browser defaults, but the Wails WKWebView renders inside an Electron-like shell where default focus rings vary by platform. Every other interactive widget in the app (`.file-browser__btn`, `.find-bar__btn`, `.link-confirm-popover__btn`) has an explicit focus-visible rule. Fix: add `.remote-panel__btn:focus-visible { outline: 2px solid #7aa2f7; outline-offset: 2px; }` after line 1666 in `style.css`.

3. **"Browse Files" button label casing mismatch** — The UI-SPEC Copywriting Contract (line 114) specifies the primary CTA label as **"Browse Files"** (capital F). The implementation renders `Browse files` (lowercase f) at `RemoteSessionsPanel.tsx:103`. This was pre-existing from Phase 122 and carried forward unchanged. The test suite (`RemoteSessionsPanel.test.tsx:197`) encodes the wrong casing and would need updating too. Fix: change the button text at `RemoteSessionsPanel.tsx:103` from `Browse files` to `Browse Files`, and update `title="Browse files"` to `title="Browse Files"` at line 100; update test assertion at `RemoteSessionsPanel.test.tsx:197` from `'Browse files'` to `'Browse Files'`.

---

## Detailed Findings

### Pillar 1: Copywriting (3/4)

**Passing:**
- "Open Session" — matches spec exactly (`RemoteSessionsPanel.tsx:95`).
- "Probing peers..." — matches spec exactly (`RemoteSessionsPanel.tsx:40`).
- "No remote peers found" — matches spec exactly (`RemoteSessionsPanel.tsx:49`).
- "No tailnet peers are running AgentHub." — matches spec exactly (`RemoteSessionsPanel.tsx:50`).
- "Shows shareable sessions" — matches spec, old copy "Shows web-enabled sessions only" is gone (verified: `grep -c 'Shows web-enabled sessions only'` → 0).
- "No shareable sessions" — matches spec (`RemoteSessionsPanel.tsx:69`).
- "This peer has no sessions with web-sharing enabled." — matches spec (`RemoteSessionsPanel.tsx:71`).
- "Unreachable" — matches spec (`RemoteSessionsPanel.tsx:63`).
- `aria-label={`Open ${s.name} in browser`}` — matches spec pattern (`RemoteSessionsPanel.tsx:94`).
- `aria-label={`Browse files on ${s.name}`}` — matches spec Accessibility Contract §3 (`RemoteSessionsPanel.tsx:101`).

**Failures:**
- WARNING: `RemoteSessionsPanel.tsx:103` — button label `Browse files` does not match the UI-SPEC Copywriting Contract which specifies **"Browse Files"** (capital F). Pre-existing from Phase 122; Phase 130 did not correct it.
- WARNING: Error state copy entirely absent. The spec Copywriting Contract requires:
  - Heading: **"Could not load sessions"**
  - Body: **"An error occurred loading remote sessions. Check your tailnet connection."**
  Neither string exists in `RemoteSessionsPanel.tsx` (grep returns 0 hits). This is not a Phase 122 carry-over — it is a new Phase 130 requirement from the spec. The panel has no error rendering path at all.

No generic labels (Submit, Click Here, OK, Cancel, Save) detected.

---

### Pillar 2: Visuals (3/4)

**Passing — colorblind-safety contract:**
- All three per-peer states are structurally and textually distinct:
  - Unreachable: `<div class="remote-panel__peer-unreachable">Unreachable</div>` — text badge is the primary signal; `#f7768e` is reinforcement only. No state is color-only.
  - Reachable-zero-sessions: "No shareable sessions" title + body — text label under peer header.
  - Reachable-with-sessions: session rows with action buttons.
- Status dots retain `title={s.status}` attribute at `RemoteSessionsPanel.tsx:85` (tooltip text for colorblind users).
- Action buttons retain aria-labels with session name at lines 93–94, 101.
- Loading region has `role="status"` + `aria-label="Loading remote peers"` at line 38.
- Focal hierarchy: session rows and their CTAs are the visually dominant elements in the populated state; per-peer meta/state text is secondary. Layout matches spec diagram.
- `prefers-reduced-motion` fallback implemented: spinner collapses to `content: '…'` pseudoelement at `style.css:1730–1743`, mirroring the `file-browser__spinner` pattern exactly.

**Failures:**
- WARNING: No `.remote-panel__btn:focus-visible` rule exists in `style.css`. The UI-SPEC Accessibility Contract §7 mandates this rule be added if missing. The spec plan (Plan 04, Task 1 action) explicitly says "Add if missing". The implementation did not add it. Every other interactive button component in the codebase (`.file-browser__btn:focus-visible` at line 3466, `.find-bar__btn:focus-visible` at line 2713, `.link-confirm-popover__btn:focus-visible` at line 2805) has this rule. Keyboard-navigating users on platforms where browser default focus rings are suppressed will have no visible focus indicator on "Open Session" / "Browse Files" buttons.

---

### Pillar 3: Color (4/4)

**Passing — all checks:**
- New Phase 130 CSS classes (`1694–1743`) use exactly these existing tokens: `#f7768e`, `#1e2030`, `#9aa5ce`, `#c0caf5`. Zero new hex values introduced.
- Existing action button tokens verified: `#7aa2f7` (Open Session bg), `#89b4fa` (hover), `#9ece6a` (Browse Files bg), `#b6dd86` (hover), `#1a1b26` (button text). All pre-existing from Phase 122.
- WCAG AAA comment preserved at `style.css:1680–1682`: `#9ece6a fg on #1a1b26 bg ≈ 9.5:1 contrast`.
- Status dot colors (`#3b82f6`, `#22c55e`, `#f59e0b`, `#ef4444`) are pre-existing from Phase 52.
- Spinner track `#292e42` / top arc `#7aa2f7` — matches spec exactly.
- 60/30/10 distribution: dominant `#1a1b26` surface, secondary `#16161e` session rows, accent blue/green limited to action buttons. Contract met.
- Colorblind mandate: confirmed. Text labels carry every state distinction; color is secondary reinforcement throughout.

No hardcoded hex outside the TokyoNight token set detected in Phase 130 additions.

---

### Pillar 4: Typography (4/4)

**Passing — all checks:**
- Phase 130 new classes introduce exactly two sizes: **11px** (`.remote-panel__peer-unreachable`) and **13px** (`.remote-panel__peer-empty-sessions`, `.remote-panel__peer-empty-sessions-body`). Both are in the declared type scale.
- Phase 130 new classes introduce exactly two weights: **600** (`.remote-panel__peer-unreachable`, `-title`) and **400** (inherited in body, explicitly 600 for title). No out-of-spec weights.
- `.remote-panel__peer-empty-sessions-title` inherits `font-size: 13px` from its parent `.remote-panel__peer-empty-sessions` — the spec does not assign a separate role/size to this element, so inheritance is correct.
- Pre-existing sizes (11px peer-header, 13px session name, 11px CLI badge, 14px empty-title) are unchanged and match the spec's Typography table.
- Font family inherits `"Cascadia Code"`, `"MesloLGS NF"`, monospace from body — no override introduced.

Note: The spec's typography table lists `.remote-panel__empty-title` under "Section label (empty / error headings) | 14px | 600 | 1.3". The existing `empty-title` rule has `font-size: 14px; font-weight: 600` (correct) but no `line-height: 1.3` (minor gap, pre-existing from Phase 52, not a Phase 130 regression).

---

### Pillar 5: Spacing (3/4)

**Phase 130 new classes — passing:**
- `.remote-panel__peer-unreachable`: `gap: 4px` (xs), `padding: 2px 8px` (established badge convention per spec footnote), `margin-bottom: 8px` (sm). All justified by spec.
- `.remote-panel__peer-empty-sessions`: `padding: 8px 0` (sm vertical). Correct.
- `.remote-panel__peer-empty-sessions-title`: `margin-bottom: 4px` (xs). Correct.

**Pre-existing panel values — not Phase 130 regressions, but documented:**
- `.remote-panel__session-list { gap: 2px }` — The UI-SPEC spacing table says `sm | 8px | Gap between session rows (.remote-panel__session-list gap)` but the actual pre-Phase-130 value (verified via `git show 000b613:frontend/src/style.css`) was already `gap: 2px`. The spec documentation is incorrect; the implementation correctly preserved the existing value.
- `.remote-panel__peer { gap: 2px; margin-bottom: 20px }` — 20px is not on the declared 4-point scale (4/8/16/24). Pre-existing from Phase 52.
- `.remote-panel__session-row { gap: 10px; padding: 10px 12px }` — 10px and 12px are not on the declared scale. Pre-existing.
- `.remote-panel__actions { gap: 6px }` — 6px not on declared scale. Pre-existing.
- `.remote-panel__btn { padding: 4px 10px }` — 10px not on declared scale. Pre-existing.
- `.remote-panel__peer-header { padding-bottom: 8px }` — Spec claims this should be `md=16px` but pre-Phase-130 value was already `8px`. Spec documentation error; implementation correctly preserved the existing value.

The 3/4 score reflects that Phase 130 itself adds only spec-compliant spacing, but the pre-existing panel has multiple off-scale values that the UI-SPEC failed to document accurately. Phase 130 did not introduce or worsen any spacing violation.

---

### Pillar 6: Experience Design (2/4)

**Passing:**
- Loading state: `loading && peers.length === 0` renders spinner + "Probing peers..." with `role="status"` (`RemoteSessionsPanel.tsx:35–44`). Correct.
- Panel-level empty state: `!loading && peers.length === 0` renders "No remote peers found" (`RemoteSessionsPanel.tsx:45–54`). Correct.
- Per-peer states: all three branches (unreachable / reachable-empty / reachable-with-sessions) are rendered. No peer is silently dropped from the UI (`RemoteSessionsPanel.tsx:60–110`).
- Poll interval: 30s (`App.tsx:918`), cleanup on unmount (`App.tsx:919–922`). No memory leak.
- `prefers-reduced-motion` fallback: implemented (`style.css:1730–1743`).
- Pick flow: `onBrowseFiles(s.id, s.name)` → `handleBrowseFilesRemote` → join-code cap or direct `FileBrowserTab` open (`App.tsx:984+`). Preserved.

**Failures:**
- BLOCKER: Error state is missing. `App.tsx:913–915`:
  ```typescript
  } catch {
    if (!cancelled) setRemoteLoading(false)
  }
  ```
  The catch block silently discards the error. There is no `remoteError` state variable (confirmed: `grep -c 'remoteError' frontend/src/App.tsx` → 0). `RemoteSessionsPanel` has no `error` prop (confirmed: `grep -c 'error' frontend/src/components/RemoteSessionsPanel.tsx` → 0). When the Wails RPC fails (Tailscale daemon down, network timeout, Go panic), `peers` stays empty and `loading` is false. The panel renders the "No remote peers found" empty state — which is semantically wrong, misleading the user into thinking there are no tailnet peers when in fact the query failed. The UI-SPEC Copywriting Contract requires a distinct error state with "Could not load sessions" heading and body text. This is a distinct user task flow (diagnosis: network issue vs. no peers) that is currently conflated.

- WARNING: No disabled state on action buttons. The UI-SPEC Interaction Contract table specifies `opacity: 0.5; cursor: not-allowed` for disabled button state ("add if not already present"). No `:disabled` or `.--disabled` rule exists on `.remote-panel__btn`. In the current implementation buttons are never programmatically disabled, but the spec anticipated adding this. Low immediate impact (no button is currently disabled) but inconsistent with the documented contract and the pattern used in `.file-browser__btn`.

---

## Registry Safety

No `components.json` found — shadcn not initialized. Registry audit skipped per protocol. No third-party blocks to audit.

---

## Files Audited

- `frontend/src/components/RemoteSessionsPanel.tsx` (116 lines — full read)
- `frontend/src/style.css` (lines 1516–1743, remote-panel section)
- `frontend/src/App.tsx` (lines 893–923, 1003–1060, 1292, 1354–1355 — remote sessions poll, error handling, panel wiring)
- `.planning/phases/130-remote-browse-gui-on-ramp/130-UI-SPEC.md` (full read)
- `.planning/phases/130-remote-browse-gui-on-ramp/130-04-PLAN.md` (full read)
- `.planning/phases/130-remote-browse-gui-on-ramp/130-04-SUMMARY.md` (full read)
- `.planning/phases/130-remote-browse-gui-on-ramp/130-01-SUMMARY.md`, `130-02-SUMMARY.md`, `130-03-SUMMARY.md` (full read)
- `frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx` (grep audit — casing assertions)
- Git history: `frontend/src/style.css` and `frontend/src/components/RemoteSessionsPanel.tsx` at commit `000b613` (pre-Phase-130 baseline)
