# Phase 125 — UI Review

**Audited:** 2026-06-14
**Baseline:** 125-UI-SPEC.md (the approved design contract)
**Screenshots:** not captured (no dev server on :3000/:5173/:8080 — code-only audit)
**Verdict:** ADVISORY / non-blocking. Ships, but with one BLOCKER-class defect (missing CSS — the Phase 124 regression class) that will render the editor header + several new affordances unstyled.

---

## Pillar Scores

| Pillar | Score | Key Finding |
|--------|-------|-------------|
| 1. Copywriting | 4/4 | Every locked EDIT-07/08/09/10/11 string is verbatim; safe-default focus on all four destructive modals confirmed at source |
| 2. Visuals | 2/4 | ~17 emitted classNames have NO matching CSS rule — editor header, save indicator, dirty marker, row-actions, move-tree, inline input all render unstyled (Phase-124-class regression) |
| 3. Color | 3/4 | Zero new hexes in named CSS; but new components hardcode TokyoNight hexes inline + the destructive row-delete relies on an undefined class for its red |
| 4. Typography | 3/4 | Inherits 13/16/11px scale; UploadQueuePanel + drop overlay set sizes via inline `fontSize` literals rather than the shared scale, and use an `h3` where the modal CSS only styles `h2` |
| 5. Spacing | 3/4 | Header/modal spacing inherits shipped values where the class exists; new components use inline `px` literals (gap 4/6/8/12) — on-grid but off-system |
| 6. Experience Design | 4/4 | All save/conflict/delete/collision/upload states carry icon+text; aria-live status regions present; safe-default focus everywhere; buffer never silently dropped |

**Overall: 19/24**

---

## Top 3 Priority Fixes

1. **~17 emitted CSS classes are undefined in `style.css` (BLOCKER — Phase 124 regression class).** The editor header chrome, three-state save indicator, dirty marker, per-row action cluster, inline name input, and move-tree all emit BEM classNames that have zero matching CSS rule. They render as default block/inline elements: the dirty `●` has no accent color, the save indicator has no layout, the row-action cluster does NOT hide-until-hover (the `:hover`/`:focus-within` reveal the SPEC §1 requires is never defined), and the `preview-name` ellipsis chain breaks because it is now nested in an unstyled `preview-name-group`. **Fix:** add the missing rules to `style.css` (see full list below) before the next visual UAT.

2. **Per-row Delete button has no red without `file-browser__btn--destructive`, which is undefined (colorblind-adjacent).** `FileRowActions` delete emits `file-browser__btn--destructive` with no inline fallback, so it renders as a plain icon button. The `TrashIcon` glyph still carries the meaning (colorblind contract technically holds), but the destructive affordance loses its visual weight and diverges from the modal delete buttons, which DO get red via inline style. **Fix:** define `.file-browser__btn--destructive { color: #f7768e }` (icon) and let the modal buttons reuse it instead of inline hexes.

3. **UploadQueuePanel + UploadDropOverlay are built almost entirely from inline styles with hardcoded hexes, bypassing the BEM/CSS system the SPEC mandates (design-language drift).** SPEC §Design System says reuse `.new-session-modal*`; the panel attaches `new-session-modal__overlay` (undefined) and `new-session-modal__title` on an `h3` (CSS only styles `__header h2`), then overrides everything with inline `style={{…}}` carrying `#292e42`, `#7aa2f7`, `#9ece6a`, `#f7768e`, `#c0caf5`, `#9aa5ce`. Functionally fine, but it is not the shared system and will not track theme changes. **Fix:** move these to `.file-browser__upload-*` BEM rules consuming the existing tokens.

---

## Detailed Findings

### Pillar 1: Copywriting (4/4)
All locked strings verified verbatim at source:
- Unsaved guard (`UnsavedChangesModal.tsx:79-124`): title `Unsaved changes`, body `You have unsaved changes. Save or discard?`, `Save` / `Discard` / `Keep editing` — default focus on `Keep editing` (`keepEditingRef`, line 93). PASS.
- Conflict 412 (`ConflictModal.tsx:108-159`): heading `This file was modified by another process`, body verbatim, `Force overwrite` / `Save as new file` / `Discard my changes` — default focus on `Discard my changes` (`discardRef`, line 122), `Force overwrite` is last and never focused. PASS.
- Delete (`DeleteConfirmModal.tsx:71-75`): file + recursive-dir variants, `Delete "{name}" and all {N} files inside? This cannot be undone.` verbatim; `Cancel` holds default focus (`cancelBtnRef`, line 125). PASS.
- Collision 409 (`CollisionConfirmModal.tsx:91-99`): `File already exists` / `A file named "{name}" already exists. Replace it?`; `Cancel` default focus. PASS.
- Move (`MoveToPickerModal.tsx:206,261`): `Move "{name}" to…` / `Move here`. PASS.
- Editor strings: `Save` / `Cancel` (EditorHeader), `Saving…` / `Saved` (SaveIndicator:47,59), `Modified` aria-label (EditorHeader:57), large-file `This is a large file ({N} MB). Edits may be slow.` (Editor:132), `Syntax highlighting disabled for large files.` (Editor:250). PASS.
- Save-error copy `Couldn't save the file. Your changes are still here — try again.` (`useFilesWrite.ts:129`). Generic op-failure `Couldn't complete that. Try again.` (FileBrowserTab). PASS.
- Upload: `Uploading {N} files` / `Uploading {name}` (UploadQueuePanel:48-51), `Done`, `Failed — try again`, over-cap skip copy, `Drop files to upload here`. PASS.

No generic-label leakage. Strongest pillar.

### Pillar 2: Visuals (2/4)
**This is the Phase-124-regression class — emitted-but-undefined CSS. Confirmed against `frontend/src/style.css` (the only global stylesheet; `FindBar/style.css` is scoped, `dist/` is a build artifact).**

Classes EMITTED by the new components with ZERO matching rule in `style.css`:

| Class | Emitted by | Visual consequence |
|-------|-----------|--------------------|
| `file-browser__editor-notice` | Editor.tsx:130 | Large-file warning banner unstyled (no padding/border/bg) |
| `file-browser__editor-notice--info` | Editor.tsx:249 | Syntax-disabled caption unstyled |
| `file-browser__editor-mount` | Editor.tsx:257 | CM6 mount relies on inline flex; class is dead but mount survives via inline style |
| `file-browser__preview-name-group` | EditorHeader.tsx:53 | Unstyled `div` inside the flex header — breaks the `preview-name` `flex:1 1 auto` ellipsis chain (now nested, not a direct flex child) |
| `file-browser__dirty-marker` | EditorHeader.tsx:56 | The `●` has NO `#7aa2f7` accent color and no spacing — renders as default text-color bullet |
| `file-browser__preview-path` | EditorHeader.tsx:72 | Path subline gets no 11px muted caption styling |
| `file-browser__preview-header-actions` | EditorHeader.tsx:79 | Right cluster (indicator+Save+Cancel) has no flex/gap |
| `file-browser__save-indicator` | SaveIndicator.tsx:35 | No layout for the status region |
| `file-browser__save-indicator-icon(+--saving/--saved/--error)` | SaveIndicator.tsx:45,57,69 | No color/sizing/spin animation — the `--saving` spinner will not spin |
| `file-browser__save-indicator-text` | SaveIndicator.tsx:47,59 | No styling |
| `file-browser__editor-error` | FileBrowserTab.tsx:1248 | Inline save-error bar unstyled (no red border, no padding) |
| `file-browser__inline-name-input` | InlineNameInput.tsx:115 | Create/rename input has no styling — likely default browser input |
| `file-browser__list-row--inline-input` | InlineNameInput.tsx:109 | Row variant unstyled |
| `file-browser__row-actions` | FileRow.tsx:135 | **No `:hover`/`:focus-within` reveal rule exists — SPEC §1 requires the action cluster to be revealed on row hover/focus. Without it, the cluster is either always visible or always at default flow position. The defining behavioral contract is unimplemented in CSS.** |
| `file-browser__btn--destructive` | FileRow.tsx:183 | Row delete button has no red (see Fix #2) |
| `file-browser__move-tree-node` / `__move-tree-row` / `--selected` / `--disabled` | MoveToPickerModal.tsx:106-113 | Move picker tree rows have no selected-highlight, no disabled dimming, no hover — selection feedback is invisible (selected target indistinguishable) |

Classes that ARE defined and reused correctly: `file-browser__preview-header`, `file-browser__preview-name`, `file-browser__preview` (29 hits), `file-browser__btn` family (`--primary` x3, `--secondary`, `--icon`), all `quit-modal*` (overlay/header/body/footer/close/subtitle/btn--cancel/--quit-gui/--quit-all), `file-browser__error`.

Note: `quit-modal__title` (used on the modal `h2`s) has no own rule, but `.quit-modal__header h2` styles it via descendant selector — so modal titles ARE styled. Not a gap. `new-session-modal__title` on an `h3` (UploadQueuePanel) is NOT covered by `.new-session-modal__header h2` — minor, masked by inline styles.

Visual hierarchy and focal point are otherwise sound (single primary Save CTA per surface; icon buttons all carry `aria-label`). Score capped at 2 because a behavioral SPEC requirement (hover-reveal row actions) and the entire editor-header/save-indicator visual layer are undefined.

### Pillar 3: Color (3/4)
- Named CSS: zero new hexes introduced (confirmed — no new `.file-browser__*` color rules were added at all, which is the flip side of the missing-CSS finding).
- CM6 theme (`Editor.tsx:43-92`): all hexes (`#1a1b26`, `#c0caf5`, `#7aa2f7`, `#16161e`, `#9aa5ce`, `#292e42`, `#1e2030`, `#283457`) are SPEC-declared TokyoNight tokens. PASS — colorblind-safe, on-palette.
- Inline hardcoded hexes in components: UploadQueuePanel, UploadDropOverlay, DeleteConfirmModal (`#f7768e`), CollisionConfirmModal (`#e0af68` border + `#f7768e`) all inline-apply SPEC tokens. On-palette, but not via the shared system — they will not follow a future theme retune. Minor drift.
- Accent `#7aa2f7` reserved correctly: Save primary, Move here primary, upload progress fill, drop-overlay border, dirty bullet (intended — though the bullet color never lands because `dirty-marker` is unstyled and EditorHeader sets no inline color). Destructive `#f7768e` reserved for delete/replace/force-overwrite.
- **Colorblind contract: PASS at source.** Every state carries a glyph + literal text independent of color (verified state-by-state in Pillar 6). Color is decoration in every case. The one nuance: the dirty `●` bullet's accent color does not render (missing CSS), but the glyph + `aria-label="Modified"` still carry the signal, so the contract holds.

Score 3 (not 4) for the inline-hex drift and the dirty-marker color not landing.

### Pillar 4: Typography (3/4)
- EditorHeader, modals inherit the 13/16/11px scale via existing classes. CM6 body is 13px monospace (Editor.tsx:50). PASS.
- UploadQueuePanel/UploadDropOverlay set `fontSize: 11/12/13/14` via inline literals (UploadDropOverlay:80 uses 14px/600 for the prompt — on-scale but inline). `h3.new-session-modal__title` does not inherit the modal heading rule (CSS targets `h2`). Off-system, on-scale. Score 3.

### Pillar 5: Spacing (3/4)
- Where defined classes are reused (preview-header `8px 16px` / gap 12px; quit-modal footer; btn `6px 12px`), spacing matches the grandfathered shipped values per SPEC. PASS.
- New components use inline `gap`/`padding` literals: UploadQueuePanel rows `padding: '8px 0'`, gap 2/4/8; DeleteConfirmModal icon `marginRight: 6`; MoveToPickerModal `paddingLeft: 20`. All multiples of 2/4 (on-grid) but applied inline rather than through the spacing system. Score 3.

### Pillar 6: Experience Design (4/4)
- **Colorblind state coverage (release-blocking) — all PASS at source:**
  - Dirty: `●` glyph + `aria-label="Modified"` + `title="Unsaved changes"` (EditorHeader:54-62).
  - Save idle: renders nothing (absence = clean). saving: `ArrowPathIcon` + `Saving…`. saved: `CheckCircleIcon` + `Saved`. error: `ExclamationTriangleIcon` (glyph only here; literal text in EditorErrorBar — SPEC-intended). (SaveIndicator.tsx:38-72.)
  - Save indicator is `role="status" aria-live="polite" aria-atomic="true"` (SaveIndicator:32-36). PASS.
  - Conflict 412: `ExclamationTriangleIcon` + heading text (ConflictModal:100-109).
  - Delete: `TrashIcon` + verb `Delete` + count; recursive variant uses `ExclamationTriangleIcon` + `{N} files` (DeleteConfirmModal:98-144).
  - Collision 409: `ExclamationTriangleIcon` + literal filename (CollisionConfirmModal:84-99).
  - Upload per-file: `N%` text / `CheckCircleIcon`+`Done` / `ExclamationTriangleIcon`+`Failed — try again` / over-cap glyph+copy (UploadQueuePanel:126-200), wrapped in `role="status" aria-live="polite"`.
  - Binary/no-edit: Edit affordance absent entirely for `isBinary`/dir (FileRow:141) — absence is the signal. PASS.
- **Safe-default focus on every destructive modal:** Keep editing / Discard my changes / Cancel / Cancel — all verified via `ref`+`useEffect` focus (Pillar 1). Destructive buttons never default-focused. PASS.
- State machine: loading (`Loading…` in move-tree), error (EditorErrorBar non-takeover so buffer stays copyable), empty (`Empty` in move-tree), disabled (`acting` guards disable all modal buttons during async; Save disabled while `isSaving`; Move here disabled until target chosen). Confirmation for all four destructive ops. PASS.
- Buffer never silently discarded (ConflictModal comments + every path preserves buffer). PASS.

The experience-design / interaction logic is the strongest part of the implementation — the gap is purely that its CSS skin is missing.

---

## Cross-Surface Parity
Not directly verifiable without a running desktop + web-share pair (no dev server). At source level the components are surface-agnostic React with no Wails-specific branches, so parity is structurally sound. The SPEC §7 Wails watch-items (Tab indentation, Cmd-V double-paste in WebView) cannot be confirmed from code alone — flag for live UAT. `needs_human_review: true` for the two WebView keymap items.

## Registry Safety
No `components.json`; project does not use shadcn or any third-party UI registry. CodeMirror 6 is a vendored npm dep gated by `vendor_drift_test.go`, not a registry block. Registry audit: not applicable, skipped.

---

## Files Audited
- `.planning/phases/125-react-editor-codemirror-6-desktop-web/125-UI-SPEC.md`
- `frontend/src/components/Editor.tsx`
- `frontend/src/components/FileBrowser/EditorHeader.tsx`
- `frontend/src/components/FileBrowser/SaveIndicator.tsx`
- `frontend/src/components/FileBrowser/UploadQueuePanel.tsx`
- `frontend/src/components/FileBrowser/UploadDropOverlay.tsx`
- `frontend/src/components/FileBrowser/InlineNameInput.tsx`
- `frontend/src/components/FileBrowser/FileRow.tsx`
- `frontend/src/components/FileBrowser/BreadcrumbBar.tsx`
- `frontend/src/components/FileBrowser/modals/ConflictModal.tsx`
- `frontend/src/components/FileBrowser/modals/DeleteConfirmModal.tsx`
- `frontend/src/components/FileBrowser/modals/CollisionConfirmModal.tsx`
- `frontend/src/components/FileBrowser/modals/UnsavedChangesModal.tsx`
- `frontend/src/components/FileBrowser/modals/MoveToPickerModal.tsx`
- `frontend/src/components/FileBrowserTab.tsx` (error-bar rendering)
- `frontend/src/lib/useFilesWrite.ts` (save-error copy)
- `frontend/src/style.css` (CSS class definition audit)
