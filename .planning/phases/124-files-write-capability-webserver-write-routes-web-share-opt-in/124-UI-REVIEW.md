# Phase 124 — UI Review

**Audited:** 2026-06-14
**Baseline:** 124-UI-SPEC.md (approved design contract)
**Screenshots:** not captured (no dev server on :3000/:5173/:8080 — code-only audit)
**Registry audit:** skipped (no `components.json`, no shadcn — UI-SPEC declares no registries)
**Verdict:** ADVISORY — does not block. Cross-surface parity and colorblind contract both PASS at source level.

---

## Pillar Scores

| Pillar | Score | Key Finding |
|--------|-------|-------------|
| 1. Copywriting | 4/4 | All locked copy (CAP-04/05/06) is verbatim, including the multi-line warning and the smart-quote helptext |
| 2. Visuals (hierarchy/consistency) | 3/4 | Reuses toggle/banner/share components correctly, but `session-share-panel__write-optin`, `__write-confirm`, and `__link-row--locked` have NO CSS definitions — they render unstyled |
| 3. Color | 4/4 | Zero new hexes; warning is amber `#f59e0b` not destructive red; colorblind reinforcement-only confirmed at source |
| 4. Typography | 4/4 | Inherits 13px/12px scale; no new sizes; heading 13px/600, body 13px/400 match the contract exactly |
| 5. Spacing | 3/4 | Banner/toggle spacing reused verbatim, but several inline-style spacing values (`46px`, `12px`) bypass the BEM/token system as one-off literals |
| 6. Experience Design (states/parity) | 4/4 | Default-OFF on both toggles, saving/error/disabled states present, two-gate WR-01 link gating correct, GUI↔TUI parity on identical server signal |

**Overall: 22/24**

---

## Top 3 Priority Fixes

1. **Missing CSS for the write opt-in and locked-link rows (WARNING).** `SessionSharePanel.tsx` emits `session-share-panel__write-optin`, `session-share-panel__write-confirm`, `session-share-panel__write-confirm-body`, `session-share-panel__write-confirm-actions`, `session-share-panel__link-row--locked`, and `session-share-panel__url--locked`, but `grep` of `frontend/src/style.css` (and all `.css`) returns NONE of these selectors. The toggle itself inherits `settings-panel__toggle-row`, so it is styled — but the confirmation block, the disabled-gate dimming (only an inline `opacity: 0.6` survives), and the locked-placeholder row have no dedicated rules. The inline-confirmation styling the spec says to "reuse existing inline confirmation styling" was never wired to an actual class. *Fix:* add the `session-share-panel__write-*` and `--locked` rules to `style.css` (the SPEC Surface 2 says reuse existing inline-confirmation styling — define those selectors or apply the existing class names).

2. **Inline-style spacing literals bypass the token system (WARNING).** `DaemonManagerPanel.tsx:340` sets the helptext margin to `'0 0 4px 46px'` and `:344` repeats `46px`; the LAN-creds block (`:224-267`) is entirely inline-styled with `8px 12px`, hardcoded `#1e2030`/`#3b4261`/`#16161e`. The SPEC's Spacing Scale only sanctions multiples of 4 via classes; `46px` is an un-tokened magic value (toggle track 36px + 10px gap, computed by hand) and the colors here are not in the Phase 124 palette table. The helptext should be a `.settings-panel__toggle-helptext` rule in `style.css`, not an inline style. (LAN-creds is pre-existing from an earlier phase, not introduced here — note but do not block.) *Fix:* move the helptext margin/size/color into a CSS class so the 46px alignment lives in one place.

3. **Error-state color is applied via inline `#f7768e`, not a class (WARNING).** `DaemonManagerPanel.tsx:344` styles the write-error paragraph with inline `color: '#f7768e'`. The SPEC Surface 1 says reuse `settings-panel__error` styling — the class is applied, but it is then overridden with an inline destructive hex. If `.settings-panel__error` already carries the correct color this inline override is redundant; if it does not, the error copy depends on a hardcoded hex. *Fix:* drop the inline color and let `.settings-panel__error` own it.

---

## Detailed Findings

### Pillar 1: Copywriting (4/4)

Every locked string matches the contract verbatim. Verified against the Copywriting Contract table:

- Owner toggle label `Enable file writes` — `DaemonManagerPanel.tsx:338` ✓
- Owner helptext `Lets this session create, edit, delete, rename, and upload files in its working directory. Off by default.` — `:341` ✓ (exact)
- Web-share label `Allow file editing` — `SessionSharePanel.tsx:218` ✓
- CAP-05 confirm body `This will allow the recipient to create, edit, delete, rename, and upload files in this session's working directory.` — `:223` ✓ (uses `&apos;` HTML entity for the apostrophe — renders identically)
- GUI warning heading `Warning: writes can affect your home directory` — `HomeDirWriteWarning.tsx:47` ✓
- GUI warning body — `:51` ✓ verbatim including `(~/.zshrc, ~/.ssh, ~/.claude)` and `Protected system files are always blocked.`
- TUI warning `⚠ Warning: cwd is $HOME — writes can affect dotfiles, SSH keys, and shell config. Protected files are blocked.` — `files.go:332` ✓ verbatim, externalised as the `homeDirWriteWarning` const
- Error state `Couldn't save the write setting. Try again.` — `DaemonManagerPanel.tsx:130` ✓

The empty/disabled state matches the contract's "absence is the empty state" model — no write affordance is rendered while disabled. The locked-link placeholders (`Enable file writes to generate a write link` / `Enable "Allow file editing" above to generate a write link`, `SessionSharePanel.tsx:282-284`) are NOT in the locked-copy table but are sensible additions for the two-gate UX; they use curly quotes consistently. No generic "Submit/OK/Save" labels found. No issue.

Note: the contract lists a transient `Saved` success token (~1.5s). The owner toggle does NOT render a `Saved` confirmation — it relies on the thumb position flipping as the implicit success signal. This is a minor deviation from the contract's stated success affordance but not a copy defect; the spec's own Detected Patterns table calls `Saved` a "nicety." Flagged as WARNING-level, not a copy failure.

### Pillar 2: Visuals (3/4)

Component reuse is correct in structure: `HomeDirWriteWarning.tsx` builds on `webgl-recovery-banner` + the new `--home-write-warning` modifier (verified present at `style.css:1863`), with the `local-network-banner__icon` glyph span and the `webgl-recovery-banner__dismiss` XMarkIcon (16px) exactly as specified. Both toggles reuse `settings-panel__toggle-row/__toggle-track/__toggle-thumb/__toggle-input` (verified `style.css:680-728`). Zero new CSS files; the `--home-write-warning` modifier and two `__home-write-*` helpers are the only additions, namespaced to avoid collision with the `__shell-*` Phase 101 helpers. This satisfies "introduce zero new color hexes and zero new CSS files."

The score is held at 3 by the missing-CSS finding (Top Fix #1): the write opt-in row, its inline confirmation, and the locked-link placeholder reference six selectors with no definitions in `style.css`. Visually, the confirmation block and locked row will inherit only default/parent layout — no border, padding, or visual separation distinguishing the confirmation from the surrounding panel. Visual hierarchy of the consent gate (the most security-sensitive interaction in the phase) is therefore weaker than the contract intends. The toggle and banner themselves are fine; the gap is the surrounding chrome.

Focal point and hierarchy on the warning banner are good: amber left-border + glyph + 600-weight heading + 400 body gives a clear two-level read.

### Pillar 3: Color (4/4)

Zero new hexes introduced. Verified:

- Warning border + glyph `#f59e0b` (`style.css:1864`, `:1707`) — amber, matching `local-network-banner`, NOT the destructive `#f7768e`. Correct per the contract's explicit "cautionary not destructive" rule.
- Toggle ON track/thumb `#7aa2f7` / thumb `#1a1b26` (`style.css:715-721`) — accent reserved to ON-state + focus outline (`:1835`), not body text or warning.
- TUI `StatusWaiting` = `#8c6c3e` / `#e0af68` (`styles.go:62`) used for the warning line (`files.go:380`).

**Colorblind contract (release-blocking) — PASS at source level**, per the colorblind verification rule (verify the glyph/text tokens in code, not by eye):

- GUI banner: `⚠` glyph (`HomeDirWriteWarning.tsx:45`) + literal `Warning:` text (`:47`). Color is reinforcement only.
- TUI warning: `⚠` + `Warning:` baked into the `homeDirWriteWarning` const (`files.go:332`); `files_test.go:1026-1032` asserts the glyph, the literal `Warning:` token, AND the verbatim `Warning: cwd is $HOME` are present — a regression guard exists.
- Toggle state: thumb position (`translateX(16px)`) is the primary non-color signal (`style.css:719`), and `role="switch"` + `aria-checked` are present on both inputs (`DaemonManagerPanel.tsx:328-329`, `SessionSharePanel.tsx:208-209`).

Minor note (not scored down): the LAN-creds inline block (`DaemonManagerPanel.tsx:228-263`) uses `#1e2030`/`#3b4261` which are not in the Phase 124 color table — but this is pre-existing Phase-87/P-3 code reused as-is, outside this phase's scope.

### Pillar 4: Typography (4/4)

No new sizes. Banner heading `13px / 600 / 1.5` (`style.css:1871-1874`) matches the contract's Emphasis role. Banner body `13px / 400 / 1.5` (`:1878-1881`) matches Body. Toggle label `13px / 400 / 1.5` (`:724-727`) matches Label. Helptext `12px / 1.4` (inline at `DaemonManagerPanel.tsx:340`) matches the Caption role. TUI emphasis is via `StatusWaiting` foreground, not bold-only — honoring the contract's "bold alone is not a reliable signal" note. No issue.

### Pillar 5: Spacing (3/4)

Reused values are correct: banner padding `12px 16px` (`style.css:1799`), banner `margin-bottom: 24px` (`:1800`, collapsing to 0 in `banner-stack`), heading icon-to-text `gap: 4px` (`:1870`), toggle-row `gap: 10px` + `min-height: 44px` grandfathered exception (`:690-692`) — all match the Spacing Scale and its declared Exceptions.

Held at 3 by inline-style spacing literals (Top Fix #2): `DaemonManagerPanel.tsx:340/344` hardcode `margin: '0 0 4px 46px'`. `46px` is an off-grid hand-computed alignment value (track 36 + gap 10) living inline rather than in a class, which means the helptext indent will silently drift if the toggle dimensions ever change. The `4px` bottom margins are on-grid but should be tokenized. This is a maintainability/consistency gap, not a visible-at-rest defect.

### Pillar 6: Experience Design (4/4)

State coverage is strong:

- **Default-OFF (both toggles):** owner toggle initial state is the empty `sessionWrites` map → `!!sessionWrites[s.id]` is `false` (`DaemonManagerPanel.tsx:48,331`); migration default `FilesWrite: false` confirmed server-side (`engine.go:108,192`). Web-share opt-in `useState(false)` (`SessionSharePanel.tsx:55`). Both verified OFF.
- **Saving state:** `settings-panel__toggle-row--saving` + `pointerEvents:none; opacity:0.6` + `disabled` during in-flight `SetSessionFilesWrite` (`:322-332`). Matches contract.
- **Error state:** `Couldn't save the write setting. Try again.` rendered in `settings-panel__error` (`:128-131,343-347`). A second, distinct error path handles the post-toggle `IssueCapabilities` failure (IN-01) with `Links may be stale — try toggling off and on to refresh.` — robust failure handling beyond what the contract required.
- **Disabled gate:** web-share opt-in is `aria-disabled` + `opacity:0.6` + `pointerEvents:none` + `disabled` on the input until `ownerWriteEnabled` (`SessionSharePanel.tsx:198-213`). The `handleWriteOptinToggle` also hard-guards `if (!ownerWriteEnabled) return` (`:87`). Two-layer gate.
- **Two-gate link disclosure (WR-01):** `surfaceWriteLink = ownerWriteEnabled && allowFileEditing` (`:62`); the Full Access Link URL/token + write-QR are only rendered when both gates hold, otherwise a locked placeholder (`:250-287`). Toggling either gate off immediately collapses the write QR and clears its b64 (`:90-98`). This correctly implements the WR-01 fix — the `files.write` link is never disclosed before explicit per-share consent.
- **Confirmation flow:** toggling ON opens inline confirm with verbatim CAP-05 body; Confirm activates, Cancel reverts to OFF (`:101-109,220-242`). No new modal component — matches contract.
- **Banner dismissal:** per-session-per-enable, re-shows on re-enable (`:55,92-93,355-358`); NOT timer-dismissed. Matches the "standing caution" contract.

**Cross-surface parity (release-blocking) — PASS.** Both surfaces gate the home-dir warning on the SAME server-side signal: GUI uses `sessionWrites[s.id] && s.homeDir` where `s.homeDir` comes from `SessionInfo.HomeDir` (the ListSessions-derived field, `DaemonManagerPanel.tsx:355` with the explicit comment that it deliberately uses `s.homeDir` over the staleable per-capability `share?.homeDir`). TUI gates on `m.sessions[i].HomeDir && m.sessions[i].FilesWrite` from the same `daemon.SessionInfo` (`files.go:378-379`), populated from `engine.go:466-467`. Both collapse to the single source of truth in `engine.go`. The condition `HomeDir && FilesWrite` is identical on both surfaces, so the GUI banner ⇔ TUI line equivalence the contract requires holds. `files_test.go` guards the TUI side.

---

## Files Audited

- `.planning/phases/124-.../124-UI-SPEC.md` (contract)
- `frontend/src/components/HomeDirWriteWarning.tsx`
- `frontend/src/components/SessionSharePanel.tsx`
- `frontend/src/components/DaemonManagerPanel.tsx`
- `frontend/src/style.css` (lines 680-728 toggles, 1794-1882 banner + `--home-write-warning` modifier)
- `internal/tui/files.go` (warning line + parity condition)
- `internal/tui/styles.go` (StatusWaiting token)
- `internal/daemon/engine.go`, `internal/daemon/api.go` (HomeDir/FilesWrite source-of-truth, parity verification)
- `internal/tui/files_test.go` (colorblind-signal regression guard)
