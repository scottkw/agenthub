# AgentHub — UI/UX Recommendations

Date: 2026-05-12
Scope: Wails desktop GUI (8 screenshots) + Bubble Tea TUI (4 screenshots) + frontend source (`frontend/src/style.css`, components)
Lens: impeccable design laws (product register) + accessibility heuristics
Status: **recommendations only — no code changed**

---

## Executive summary

The app already has a coherent identity: TokyoNight palette, monospace UI chrome, Heroicons sidebar, soft hairline borders, and dot-status indicators. That coherence is the strongest thing going for it — most of the recommendations below are about **leveraging that coherence harder**, not redesigning it. There is one systemic pattern (left-stripe-accent banners) that recurs eight times and violates an impeccable absolute ban; everything else is refinement.

The findings cluster around six themes:

1. **Side-stripe accent borders** as the universal banner pattern — replace.
2. **Status color semantics are confusing and theme-incoherent** — running=blue, idle=green is backwards; the dot colors are raw Tailwind, not TokyoNight.
3. **Tab bar wastes its highest-attention surface** — agent identity isn't expressed; numeric suffixes ("claude 1") imply multiplicity that doesn't exist; close-button "×" character clashes with the Heroicons system.
4. **Sidebar has no active-page indicator** — users have no signal for "where am I."
5. **Welcome tab is category-reflex SaaS** — centered logo + "GET STARTED" + install snippets. Generic.
6. **Contrast and accessibility gaps** — placeholder text and the per-tab status hint fail WCAG AA despite the README claim; status dots are color-only at 8px.

A short-list of "if you only do five things" lives at the bottom.

---

## 1. Critical findings

### 1.1 Absolute ban: side-stripe accent borders

`border-left: 3px solid …` is used as the universal "tone indicator" across the app. Locations found in `frontend/src/style.css`:

| Class | Accent color | Purpose |
|---|---|---|
| `.ct-disclosure` | `#7aa2f7` blue | Certificate Transparency banner |
| `.ct-disclosure--acknowledged` | `#9ece6a` green | acknowledged variant |
| `.update-banner` | `#7aa2f7` blue | Update available |
| `.local-network-banner` | `#f59e0b` amber | Local-network warning |
| `.webgl-recovery-banner` | `#7aa2f7` blue | WebGL context lost |
| `.banner--info` / `.banner--error` | blue / red | Save-feedback toasts |
| `.exit-toast__item--clean` / `--error` | green / red | Session-exit toast |
| `.exit-countdown-banner` | `#9ece6a` green | Auto-close countdown |
| `.link-confirm-popover` (no stripe but same family pattern) | — | — |

Side-stripe-as-affordance is the most-recognizable AI-design-tool tic of the past three years. It signals "AI generated banner." Eight instances ossify it as the system's voice.

**Replace pattern.** Pick one of these and apply consistently:

- **Option A — Tinted background + leading status pill.** Background goes to a 6% wash of the tone color; a small leading pill (`◆ INFO`, `◆ WARNING`) carries the semantic. Removes the stripe entirely; reads cleaner at any width.
- **Option B — Full 1px tone border around the banner.** No left-only stripe; the whole banner is enclosed by a low-saturation border in the tone color. Quieter than a 3px stripe, more deliberate, and reads as "a typed object" rather than "a generic alert."
- **Option C — Leading glyph in the tone color, no border at all.** Each banner gets a small Heroicons SVG (`InformationCircleIcon`, `ExclamationTriangleIcon`, `XCircleIcon`) in the tone color. The chrome around the banner is identical regardless of tone; the glyph carries the meaning. Most editorial; cheapest implementation.

Recommended: **Option C** for non-modal banners, **Option B** for persistent inline cards (the Certificate Transparency disclosure on Settings, which is a longer-lived object). Either way, the 3px stripe goes away.

### 1.2 Status dot semantics are inverted from convention

In `style.css` (lines 882–901, 1273–1276, 1455–1458):

```
.tab__status--running  { background: #3b82f6; }  /* BLUE   */
.tab__status--idle     { background: #22c55e; }  /* GREEN  */
.tab__status--waiting  { background: #f59e0b; }  /* AMBER  */
.tab__status--errored  { background: #ef4444; }  /* RED    */
```

Every other terminal/process tool on the planet uses **green = active/running** and **gray or muted = idle**. The current mapping inverts that — when a session is happily running, the dot is blue, and when it's idle (which the user reads as "alive but waiting"), the dot is green. Two issues compound:

1. **Cognitive mismatch.** Users coming from `pm2`, Docker Desktop, Activity Monitor, GitHub Actions, etc., will mis-read at-a-glance.
2. **Theme incoherence.** These four hex values are stock Tailwind (`#3b82f6`, `#22c55e`, `#f59e0b`, `#ef4444`). They sit in an otherwise tight TokyoNight system. Compare to TokyoNight's actual tokens already in the codebase: `#7aa2f7` (blue), `#9ece6a` (green), `#e0af68` (yellow), `#f7768e` (red).

**Recommendation:**
- Re-map to: **running = `#9ece6a` green**, **idle = `#565f89` muted blue-gray**, **waiting = `#e0af68` yellow**, **errored = `#f7768e` red**.
- For accessibility (color-only at 8px is below contrast guidance for chart marks), pair each dot with a glyph: solid circle for running, hollow ring for idle, half-fill for waiting, X for errored. Even at 8px the silhouette distinguishes them for users with red-green CVD.

### 1.3 Sidebar has no active-page indicator

`Sidebar.tsx` and `style.css` define `.sidebar__item` with hover and toggle states but no `--active` variant. The screenshots confirm: you cannot tell from the sidebar whether you're currently viewing Home, Remote, Sessions, or Settings — the visual answer lives only in the tab bar above.

**Recommendation:** Add a `.sidebar__item--active` modifier that:
- Sets `color: #c0caf5` (full-strength foreground).
- Adds a 2px leading rail (`box-shadow: inset 2px 0 0 #7aa2f7`), echoing the active-tab underline at the top of the tab bar. (Note: 2px inset rail is **not** a side-stripe ban violation — the ban targets decorative tone accents on cards/banners, not navigation rails. Navigation rails are a recognized pattern and read as "selection," not "alert.")
- Or alternatively, a soft background wash (`background: rgba(122, 162, 247, 0.08)`) which avoids the rail debate entirely.

### 1.4 WCAG AA failures despite README claim

The README says "All non-terminal GUI text meets WCAG AA 4.5:1 contrast ratio." Spot-checking against `#16161e`/`#1a1b26` backgrounds:

| Class | Color | BG | Contrast | AA pass? |
|---|---|---|---|---|
| `.tab-status-bar__hint` | `#545c7e` | `#16161e` | ~3.4:1 | **No** (italic 12px) |
| `.find-bar__input::placeholder` | `#414868` | `#16161e` | ~2.2:1 | No (placeholders get leeway but this is harsh) |
| `.new-session-modal__args-input::placeholder` | `#414868` | `#16161e` | ~2.2:1 | No |
| `.settings-web-server__copy-hint` | `#414868` | `#1e2030` | ~2.0:1 | **No** |

**Recommendation:** Lift the dimmest text token from `#414868` to `#7a88b0` (already used elsewhere in the codebase for `credential-value`) for placeholders and inline hints. Lift `#545c7e` to `#7a88b0` for the per-tab status hint. The README claim then matches reality.

---

## 2. Tab bar (the highest-attention surface)

This is the bar across the top of the GUI window. It's the first thing users scan and it carries the most state per pixel. Right now it underperforms.

### 2.1 Agent identity is invisible at a glance

The Sessions screenshot shows tabs labeled "claude 1 ×", "codex 2 ×", "gemini 3 ×", "opencode 4 ×" with the same blue running dot. Reading each is a typography exercise. The TUI gets this right — "per-agent colored badges" with TokyoNight-derived colors. The GUI tab bar throws that information away.

**Recommendation:** Each tab gets a 2px **leading agent-color rail** on the left edge of the tab content (not the bottom border — the bottom border is reserved for active-tab indication). Suggested mapping using TokyoNight tokens:

- Claude → `#7aa2f7` (blue)
- Codex → `#bb9af7` (purple)
- Gemini → `#e0af68` (yellow/gold)
- OpenCode → `#7dcfff` (cyan)

Or replace the status dot entirely with a small agent glyph (the same SVG monogram each CLI uses) tinted in the agent color, with the running/idle/etc. state expressed by glyph variant or border-color modulation.

### 2.2 The numeric suffix problem

"claude 1", "codex 2", "gemini 3", "opencode 4" reads like enumeration ("of N") when each is unique. Suggest:

- Default tab name = working-directory basename (e.g. `~/dev/temp` → `temp`) plus agent indicator from §2.1.
- Strip the numeric suffix unless there are actually two of the same CLI.
- Show the full path on hover (it's already shown in the agent welcome screens).

### 2.3 Active-tab + progress underline collide

Both `.tab--active` (`border-bottom: 2px solid #7aa2f7`) and `.tab__progress` (`background: #7aa2f7`, `height: 2px`, `bottom: 0`) are the same color and same position. On an inactive tab with active OSC 9;4 progress, the underline can read as "this tab is active" — a state error.

**Recommendation:** Either offset the progress underline upward (`bottom: 2px` instead of `0`) so it sits *above* the active-tab line on the active tab, or give the progress underline a distinct color — `#9ece6a` (TokyoNight green, signals "work happening") would also let it serve as an in-progress indicator visually distinct from "this tab is selected." Both signals then coexist cleanly.

### 2.4 Close-button "×" character clashes with the icon system

The sidebar uses Heroicons SVGs. The tab close button uses a Unicode "×" character. It looks like a font fallback; on some systems it renders differently than on others.

**Recommendation:** Use `XMarkIcon` from Heroicons (already imported into the project's icon set). One character of typography goes away and the system tightens.

---

## 3. Welcome tab (the first-run impression)

What you have today (`WelcomeTab.tsx`):

- Centered 280px logo
- "AI Coding Session Manager" tagline, `v3.0`
- "GET STARTED" all-caps section
- 3-row Installation grid (macOS / Linux / Windows code snippets)
- One "Links" line with a github URL

This is **category-reflex SaaS welcome**. If you covered up the logo, a viewer could guess: "developer tool, dark mode, install snippets, 'Get Started' heading." Nothing about the welcome tab argues this is a *better* tool than the next one in the category.

The first-order test fails ("dev tool → dark mode + install snippets"). The second-order also fails ("dev tool that's not generic-dark → centered logo + curated install block" is the next reflex tier).

### 3.1 Reframe as a control surface, not a brochure

The welcome tab is the only screen a brand-new user sees before they have anything to operate. It should answer one question very fast: "what can I do with this?"

**Recommendation — three options, pick one:**

- **Option A — Status-first welcome.** Three live tiles, asymmetric layout: "Agents detected" (lists the four CLIs as small chips with version), "Web server" (status + URL or "click to start"), "Tailnet peers" (count + hostnames). Below the tiles: a single primary CTA — "New session" — that opens the modal. Install snippets move to a `<details>` disclosure labeled "Install on another machine." This makes the welcome tab functional even after the user has used the app, instead of being purely a first-run artifact.

- **Option B — Editorial typographic welcome.** Lose the centered column entirely. A large left-aligned wordmark ("AgentHub") in a serif (Newsreader, IBM Plex Serif, or similar) paired with the existing monospace body. Below, two columns: left = "what this does" in 2–3 sentences of human-toned copy; right = the install grid as a small "On another machine?" sidebar. The version chip moves to the bottom-right footer. Reads like a docs-site landing — slow, intentional, confident.

- **Option C — Asymmetric hero with live preview.** Lose the centered layout. Left 40% = wordmark, one-sentence tagline, primary CTA. Right 60% = a small **live** session preview (the most recent terminal's last 6 lines, or a placeholder ANSI snippet animating once). This is the most ambitious option and requires building a non-interactive xterm thumbnail; it's also the one that makes the app feel **alive** the moment it opens.

For each option, drop the all-caps `GET STARTED` heading — it's the cliche signal. Use sentence-case headings or eyebrow labels in tinted neutral (already the system's voice elsewhere).

### 3.2 Install snippets specifics

Even if you keep the install grid:

- Center alignment of three rows with `text-align: right` labels and code blocks fights itself. Left-align the labels, left-align the code, give the labels a fixed `64px` width column. Reads as a table, not a centered list.
- Add a copy-to-clipboard icon-button on each `.welcome-tab__code` — the current state requires manual selection.

---

## 4. Sessions list

The Sessions tab today shows a flat list with: dot, name, agent chip, status histogram-looking glyph (which is actually the pixelated hostname privacy redaction in screenshots — real data shows a sparkline-ish thing? Confirm), "Web Off / Web On" toggle, "Kill" button.

### 4.1 Hierarchy issues

- Each row uses two CLI badges: the name "claude 1" already says claude, and there's a `[claude]` chip beside it. Redundant. Drop the chip if the agent color rail from §2.1 is adopted.
- "Web Off" and "Kill" buttons sit side-by-side with identical visual weight (both transparent + hairline border). "Kill" turns red on hover only. **Recommendation:** Group reversible toggles (Web on/off) visually distinct from destructive actions. Either a small divider between them, or move Kill into a `…` overflow menu — the same drawer that the TUI handles via the `d` keybinding. Reduces visual weight of every row and reduces accidental kills.

### 4.2 Empty state

There's no empty-state design captured in the screenshots. **Recommendation:** When sessions = 0, show: a soft ASCII illustration or single Heroicon (`ServerStackIcon`, the same icon as the sidebar entry — meta-recognizable), copy that says "No sessions yet. Pick a CLI to start." and a primary CTA — the New Session button. Resist the urge to use the "+1 Onboarding Card" pattern; one clear action is better than three.

### 4.3 No filter / sort / search

If a user has 15 sessions across two tailnet peers, the flat list breaks. **Recommendation (low priority for v3):** Add a single search box above the list that filters by name, agent, or hostname. Same input pattern as the Find Bar — same visual vocabulary.

### 4.4 The histogram-looking glyph after the name

In the screenshots there's a pixelated bar-graph-ish element next to each session name. If that's a privacy-redaction artifact, fine — ignore. If it's a real UI element (sparkline of activity? CPU? something), **it's not legible** at that size and color. Either grow it to a meaningful chart or remove it.

---

## 5. Agent tab status bar (bottom strip)

The bottom strip in each agent tab carries: `WEB ON` state pill, full URL (truncated), a row of small agent chip glyphs, "Disable Web" button, "QR" button. That's 5 distinct UI elements in 32px of vertical space.

### 5.1 Density is too high

The `.tab-status-bar__hint` italic 12px gray text fails AA (see §1.4) and competes with the URL.

**Recommendation:**

- Drop the agent chip row — the agent is already identified in the tab above and on Welcome. It's noise here.
- Promote the URL: it's the most actionable thing on this strip (users want to share-paste it). Give it `#7aa2f7` color, no underline, hover-underline. Add an inline copy icon-button at the right edge of the URL (24px square) so users don't have to triple-click to select.
- Group "Disable Web" + "QR" into a single small overflow `…` menu, since both are infrequent actions. Reduces the strip to: state pill + URL + (copy) + overflow. Two-thirds the visual load.

### 5.2 State pill typography

`WEB ON` is 11px uppercase bold with 0.05em tracking. Good. But the dot in front of "WEB" is the same `--state--on` green that conflicts with the status-dot remap in §1.2. After the remap, this becomes consistent again.

---

## 6. Settings

The single scrollable settings page is the right call — it beats nested settings trees for an app this size. But the visual density is flat.

### 6.1 Section dividers

`settings-panel__body h3` uses a 1px `#292e42` top border with 24px margin and 20px padding to delimit sections. That's adequate but every section looks identical in weight. **Recommendation:** Give section headers a small leading Heroicon (16×16, color `#9aa5ce`) — `PaintBrushIcon` for Appearance, `CloudIcon` for Web Server, `BellIcon` for Behavior, etc. The eye now jumps from icon to icon down the page; scanning a 12-section settings page takes seconds instead of effort.

### 6.2 Toggle row affordance

`.settings-panel__toggle-row` has `min-height: 44px` (good — touch target), cursor pointer, toggle thumb that translates 16px. The thumb-off color (`#565f89`) and track-off color (`#16161e` with `#292e42` border) read as "disabled" rather than "off." That's a common React-toggle mistake: off-state looks broken.

**Recommendation:** Off-state track = `#3b4261` (subtly lighter background, reads as "available"), thumb = `#9aa5ce` (mid-foreground, reads as "the thing that moves"). On-state stays as-is (`#7aa2f7` track, `#1a1b26` thumb). The control then clearly communicates "off but functional" → "on."

### 6.3 Certificate Transparency disclosure

`.ct-disclosure` uses the side-stripe pattern — covered in §1.1. After the remap: full 1px border in `#7aa2f7` at 20% alpha, with a small `InformationCircleIcon` in `#7aa2f7` at the leading edge.

### 6.4 Web Server status block

The TLS hostname and connection-status block is the densest part of Settings. Right now it's three lines of low-contrast neutral text with one green check. **Recommendation:** Promote it to a small **2-column key-value table** with `font-variant-numeric: tabular-nums` on the values — Status / Hostname / Port / Certificate. Same content, four times more scannable.

---

## 7. Remote Sessions

The screenshot shows a sparse layout: "Shows web-enabled sessions only" tiny meta line + the pixelated peer hostname + one session row. Functional but inert.

### 7.1 Peer hostname is the most important element on this view

It's currently rendered as the smallest text on the page (11px, uppercase, dim gray). **Recommendation:** Promote to 14px regular case, full-strength foreground, with a subtle 1px bottom-border separator below each peer group (already in the code via `.remote-panel__peer-header`'s `border-bottom`, but the text itself should be louder).

### 7.2 Open Session button stands out (good) but lacks context

Single primary button per row is correct. Add an inline metadata strip below the session name when hover or focus: `claude · started 2h ago · 1 viewer`. Two-row layout on hover; collapses back on blur. Gives users enough context to decide whether to attach.

### 7.3 Empty state

If a tailnet has zero peers running AgentHub, the panel should explain *why* there's nothing — "No peers running AgentHub found on tailnet `tailXXXX.ts.net`. Other machines need to be running AgentHub with web server enabled." This is harder to compose than to deliver, but the current view will leave new tailnet users wondering whether the feature is broken.

---

## 8. New Session modal

Functionally fine. Two polish items:

### 8.1 Agent picker is a vertical list of 4 buttons of identical appearance

These are the four most important options on the screen; they deserve more affordance. **Recommendation:** 2×2 grid of bigger selectors (80px tall), each with: a colored agent monogram (using the agent-color tokens from §2.1), the agent name in regular weight below, and a small version number underneath in dim color. Selection is the existing `--selected` border, but with the agent color instead of generic `#7aa2f7` — now the modal **feels different** depending on which agent you pick, which is what's actually happening downstream.

### 8.2 "Pick a working directory" affordance

The folder display row is fine, but the "Browse…" button shrinks to nothing visually. **Recommendation:** Give it a small `FolderOpenIcon` leading the label and lift the border to `#3b4261` (lighter than the input border, signals interactivity). Same change applies to the path-browse buttons in Settings.

---

## 9. TUI specific

The TUI is genuinely impressive already — tab metaphor, per-agent badges, bordered lipgloss frames, focus-aware nav. Three small notes:

### 9.1 Settings tab is read-only with a "run `agenthub settings` to edit" line

That's awkward — the user is in the TUI; they shouldn't have to drop to a CLI to flip a toggle. **Recommendation (medium priority):** Either ship a minimal in-TUI editor for the most-commonly-changed settings (theme picker, auto-close toggle, web server on/off) or remove the Settings tab from the TUI entirely and replace it with a `?` overlay that shows read-only config plus instructions. Half-implemented features are worse than absent ones.

### 9.2 Per-agent badges in Sessions list use brackets `[claude]`, `[codex]`, etc.

This is fine in the TUI vocabulary but it's also redundant with the per-agent badge color. **Recommendation (minor):** Drop the brackets — keep the color and word — to free a few characters of column space. `claude` instead of `[claude]`.

### 9.3 The full ASCII QR overlay is a delight

Keep it. Consider extending the same affordance into the GUI Session row: pressing `q` while hovering a session row shows an inline ASCII QR for share-by-line-of-sight. It would be a memorable cross-mode signature.

---

## 10. Cross-cutting polish

### 10.1 Border-radius inconsistency

The codebase uses 2px, 3px, 4px, 6px, 8px, 12px radii in different places. **Recommendation:** Standardize on three values — `--radius-sm: 4px` (buttons, chips, small inputs), `--radius-md: 6px` (cards, banners, session rows), `--radius-lg: 8px` (modals, overlays). Refactor existing values to the nearest token.

### 10.2 Typography pairing

Everything is monospace (`Cascadia Code` with fallbacks). This is the strongest "terminal app" signal but it also means there's no contrast between data (terminal output, paths, code) and chrome (labels, headings, button copy). Polished terminal apps (Warp, Ghostty, Zed) use a sans for chrome and reserve mono for data.

**Recommendation:** Introduce one sans-serif for non-data UI text — `Inter`, `Geist`, `SF Pro Text` (already on macOS), or `IBM Plex Sans`. Apply to: sidebar labels, settings section headers, button labels, modal titles, banner text. **Keep monospace** for: session names (they're filesystem paths), URLs, codes, the welcome install snippets, terminal output (already mono via xterm). The split makes chrome feel intentional and makes data feel like data.

### 10.3 Hover-state visibility

Most hover states shift by ≤4 luminance steps (`#16161e` → `#1e2030`). On a typical laptop screen the change is at the perception threshold. **Recommendation:** Boost hover-bg by 6–8 luminance steps (`#1e2030` → `#2a2f45`) and add a 100ms transition. Same with hover-border: lift to `#414868` instead of `#3b4261`. Interactions then feel responsive without becoming flashy.

### 10.4 Motion language

Existing transitions are 100–200ms `ease` — fine. The `.find-bar` and `.link-confirm-popover` use `cubic-bezier(0.16, 1, 0.3, 1)` — that's an ease-out-quint curve. Good. **Recommendation:** Adopt that curve project-wide via a single token (`--ease-out-quint`), and use it for sidebar toggle, modal open, banner enter/exit, and toggle thumb travel. The system then has one motion personality instead of several.

### 10.5 Window doesn't feel like a command center

In every screenshot the AgentHub window is small (about 700×500) floating in the middle of a desktop. For an app whose tagline is "AI Coding Session Manager," that's a tiny utility footprint. **Recommendation (out-of-scope but worth noting):** Set a larger default window size — `1100×720` minimum — and bias the layout to use the horizontal space. The current vertical-list layouts on Welcome and Sessions become two-column or three-column at wider widths.

### 10.6 Reduced-motion respect

The codebase has `@media (prefers-reduced-motion: reduce)` rules for `.find-bar`, `.webgl-recovery-banner`, `.banner`, and `.link-confirm-popover` — good. But not for `.sidebar` (toggle), `.tab__progress` (200ms growth), or the QR modal slide. **Recommendation:** One consolidated `@media (prefers-reduced-motion: reduce)` block at the top of `style.css` that disables all non-essential transitions/animations. Belt-and-suspenders accessibility.

---

## 11. Things to keep

Calling out wins so they don't get refactored away in the cleanup:

- **TokyoNight palette commitment.** The whole UI is in one coherent color system. Stay there. Don't drift toward "neutral SaaS."
- **Sidebar collapsed mode with fixed icon position.** The 48px icon slot that stays put during the width transition is a small piece of craft most apps miss.
- **Heroicons SVG everywhere (except the tab close).** Consistent icon system.
- **Banner stack with independent dismiss + 53px row height + 3-banner max-height.** Solves a problem most apps don't even acknowledge.
- **OSC 9;4 progress as per-tab underline + aggregate tray glyph.** Genuinely original.
- **`font-variant-numeric: tabular-nums` on countdowns.** Right call.
- **Session-exit handling with countdown + "Keep Open" toast.** Good UX pattern; well-executed.
- **The settings page being a single scroll instead of nested tabs.** Right call for this app size.
- **TUI parity with the GUI.** Most desktop apps don't have a TUI, let alone one this complete.

---

## 12. If you only do five things

In rough order of impact:

1. **Replace the side-stripe banner pattern** across all 8+ instances. Adopt the leading-icon approach (§1.1 Option C). High visibility, removes the strongest "AI tic" signal in the app.
2. **Re-map status dot colors** to TokyoNight tokens and fix the running=blue / idle=green inversion (§1.2). Pair with shape variants for CVD accessibility.
3. **Add a `.sidebar__item--active` indicator** (§1.3). Solves "where am I" instantly; one CSS rule.
4. **Surface agent identity in the tab bar** via a 2px leading color rail per agent (§2.1). Tabs become parseable at a glance instead of by reading.
5. **Re-do the Welcome tab as a status-first control surface** (§3.1 Option A). The first-run impression becomes a useful screen rather than a brochure.

Each of these is independently shippable and reversible. None require new dependencies. Items 1 and 2 are pure CSS; item 3 is one TSX prop; items 4–5 are component-scoped.

---

## Appendix A — Token recommendations

A consolidated token sheet to extract from `style.css` once the above lands:

```
/* Color — surface */
--surface-app:      #1a1b26;
--surface-chrome:   #16161e;
--surface-card:     #1e2030;
--surface-elev:     #292e42;

/* Color — foreground */
--fg-primary:       #c0caf5;
--fg-secondary:     #a9b1d6;
--fg-muted:         #9aa5ce;
--fg-dim:           #7a88b0;  /* replaces #414868 for placeholders/hints */

/* Color — accent */
--accent:           #7aa2f7;
--accent-hi:        #89b4fa;
--accent-wash:      rgba(122, 162, 247, 0.12);

/* Color — semantic (TokyoNight, not Tailwind) */
--ok:               #9ece6a;
--warn:             #e0af68;
--err:              #f7768e;
--idle:             #565f89;

/* Color — agents */
--agent-claude:     #7aa2f7;
--agent-codex:      #bb9af7;
--agent-gemini:     #e0af68;
--agent-opencode:   #7dcfff;

/* Radius */
--radius-sm:        4px;
--radius-md:        6px;
--radius-lg:        8px;

/* Motion */
--ease:             cubic-bezier(0.16, 1, 0.3, 1);
--dur-fast:         100ms;
--dur-base:         200ms;
```

This is not prescriptive — adopt the names that fit your team's existing conventions. The point is to centralize so that "the accent color" or "the radius for cards" is decided once, not re-decided in 40 selectors.

---

## Appendix B — Files referenced

- `frontend/src/style.css` (the design system, 2453 lines)
- `frontend/src/components/Sidebar.tsx`
- `frontend/src/components/WelcomeTab.tsx`
- `frontend/src/components/TabBar.tsx` *(not opened, inferred from CSS)*
- `frontend/src/components/SettingsTab.tsx` *(not opened, inferred from CSS)*
- `frontend/src/components/RemoteSessionsPanel.tsx` *(not opened, inferred from CSS)*
- 12 screenshots in `screenshots/`
- `README.md` for product context

No code modified.
