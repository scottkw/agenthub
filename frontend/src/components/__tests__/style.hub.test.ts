import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// Phase 131 — Hub CSS contract tests.
// Pattern: mirrors style.contrast.test.ts (reads raw CSS text, asserts structure).
// These tests verify Phase 131 design contract properties at the source level.
// The user is colorblind: hex constants are verified in source, never by eye.
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

// Phase 131 UAT follow-up — Open button (re-attach) must be styled, not bare.
// Guards against the CR-01 class of bug (TSX emits a class CSS never defines).
describe('Hub card Open button CSS contract (re-attach)', () => {
  it('defines .hub-card__row5 (actions row)', () => {
    expect(cssRaw).toContain('.hub-card__row5')
  })

  it('defines .hub-card__open button rule', () => {
    expect(cssRaw).toContain('.hub-card__open')
  })

  it('defines .hub-card__open:hover state', () => {
    expect(cssRaw).toContain('.hub-card__open:hover')
  })
})

// Phase 132 UAT fix — no Tailwind in this project, so Heroicon w-N/h-N classes are
// no-ops; Hub SVG icons must be sized explicitly or they render oversized.
describe('Hub icon sizing CSS contract (no Tailwind)', () => {
  it('sizes the group-sidebar collapse-toggle chevron svg', () => {
    expect(cssRaw).toContain('.hub__group-sidebar-toggle svg')
  })

  it('sizes the needs-input badge svg', () => {
    expect(cssRaw).toContain('.hub__group-sidebar-item__needs-input-badge svg')
  })

  it('sizes the card drag-handle / menu-btn svg', () => {
    expect(cssRaw).toContain('.hub-card__drag-handle svg')
    expect(cssRaw).toContain('.hub-card__menu-btn svg')
  })
})

describe('Hub CSS tokens — dark theme (default :root)', () => {
  it('declares --hub-bg custom property', () => {
    expect(cssRaw).toContain('--hub-bg')
  })

  it('declares --hub-surface custom property', () => {
    expect(cssRaw).toContain('--hub-surface')
  })

  it('declares --hub-accent custom property', () => {
    expect(cssRaw).toContain('--hub-accent')
  })

  it('declares --hub-destructive custom property', () => {
    expect(cssRaw).toContain('--hub-destructive')
  })

  it('declares --hub-dim-opacity custom property', () => {
    expect(cssRaw).toContain('--hub-dim-opacity')
  })

  it('dark theme --hub-bg has correct hex #14151b (comp dark surface)', () => {
    expect(cssRaw).toContain('--hub-bg: #14151b')
  })

  it('dark theme --hub-accent has correct hex #7aa2f7', () => {
    expect(cssRaw).toContain('--hub-accent: #7aa2f7')
  })
})

describe('Hub CSS tokens — light theme ([data-ui-theme="light"])', () => {
  it('[data-ui-theme="light"] block is present', () => {
    expect(cssRaw).toContain('[data-ui-theme="light"]')
  })

  it('light theme --hub-accent is declared inside [data-ui-theme="light"] block', () => {
    // Target the token-defining block opener (selector + ` {`), not a later
    // light-theme override rule (Phase 142 POL-02 added one earlier in the file).
    const lightIdx = cssRaw.indexOf('[data-ui-theme="light"] {')
    expect(lightIdx).toBeGreaterThan(-1)
    // Find the closing brace of the light block
    const blockEnd = cssRaw.indexOf('}', lightIdx + 1)
    const lightBlock = cssRaw.slice(lightIdx, blockEnd)
    expect(lightBlock).toContain('--hub-accent')
  })

  it('light theme --hub-destructive is declared inside [data-ui-theme="light"] block', () => {
    const lightIdx = cssRaw.indexOf('[data-ui-theme="light"] {')
    expect(lightIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', lightIdx + 1)
    const lightBlock = cssRaw.slice(lightIdx, blockEnd)
    expect(lightBlock).toContain('--hub-destructive')
  })

  it('HUB-04 WCAG AA comment for --hub-accent in light theme (#3d6fe8 on #ffffff = 4.5:1)', () => {
    // Source-level verification: hex constant + contrast comment must be present
    expect(cssRaw).toContain('#3d6fe8')
    expect(cssRaw).toContain('4.5:1')
  })

  it('HUB-04 WCAG AA comment for --hub-destructive in light theme (#c0394f on #ffffff = 4.7:1)', () => {
    expect(cssRaw).toContain('#c0394f')
    expect(cssRaw).toContain('4.7:1')
  })

  it('light theme --hub-bg has correct hex #f5f5f7', () => {
    expect(cssRaw).toContain('--hub-bg: #f5f5f7')
  })
})

describe('Hub layout — responsive grid (GRID-01)', () => {
  it('grid uses repeat(auto-fill, minmax(240px, 1fr)) (GRID-01)', () => {
    expect(cssRaw).toContain('repeat(auto-fill, minmax(240px, 1fr))')
  })

  it('.hub__card-row declares display: grid', () => {
    const hubCardRowIdx = cssRaw.indexOf('.hub__card-row')
    expect(hubCardRowIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', hubCardRowIdx)
    const block = cssRaw.slice(hubCardRowIdx, blockEnd)
    expect(block).toContain('display: grid')
  })

  it('.hub__card-row has gap: 8px', () => {
    const hubCardRowIdx = cssRaw.indexOf('.hub__card-row')
    expect(hubCardRowIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', hubCardRowIdx)
    const block = cssRaw.slice(hubCardRowIdx, blockEnd)
    expect(block).toContain('gap: 8px')
  })

  it('.hub__card-row has max-width: 1440px', () => {
    const hubCardRowIdx = cssRaw.indexOf('.hub__card-row')
    expect(hubCardRowIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', hubCardRowIdx)
    const block = cssRaw.slice(hubCardRowIdx, blockEnd)
    expect(block).toContain('max-width: 1440px')
  })

  it('.hub-card has min-width: 240px', () => {
    expect(cssRaw).toContain('min-width: 240px')
  })

  it('.hub-card has max-width: 360px', () => {
    expect(cssRaw).toContain('max-width: 360px')
  })
})

describe('Hub card dim state (CARD-08)', () => {
  it('.hub-card--dim rule is present', () => {
    expect(cssRaw).toContain('.hub-card--dim')
  })

  it('.hub-card--dim uses var(--hub-dim-opacity) for opacity', () => {
    const dimIdx = cssRaw.indexOf('.hub-card--dim')
    expect(dimIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', dimIdx)
    const block = cssRaw.slice(dimIdx, blockEnd)
    expect(block).toContain('var(--hub-dim-opacity)')
  })

  it('.hub-card--dim uses var(--hub-card-dim-bg) for background', () => {
    const dimIdx = cssRaw.indexOf('.hub-card--dim')
    expect(dimIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', dimIdx)
    const block = cssRaw.slice(dimIdx, blockEnd)
    expect(block).toContain('var(--hub-card-dim-bg)')
  })

  it('error-exit cards are NOT dimmed — comment present (CARD-08 correctness)', () => {
    expect(cssRaw).toContain('Error-exit cards are NOT dimmed')
  })
})

describe('Hub motion contract — reduced-motion guard (Pitfall 6)', () => {
  it('prefers-reduced-motion: no-preference media query is present', () => {
    expect(cssRaw).toContain('prefers-reduced-motion: no-preference')
  })

  it('hub-spin animation is declared inside the reduced-motion guard', () => {
    // The spin class and keyframes must appear only inside the media query guard
    const mediaIdx = cssRaw.indexOf('prefers-reduced-motion: no-preference')
    expect(mediaIdx).toBeGreaterThan(-1)
    // The hub-spin animation class must appear after the media query start
    const spinIdx = cssRaw.indexOf('hub-card__status-icon--spin')
    expect(spinIdx).toBeGreaterThan(mediaIdx)
  })

  it('hub-spin keyframes are declared', () => {
    expect(cssRaw).toContain('@keyframes hub-spin')
  })

  it('spin animation has 0.8s linear infinite timing', () => {
    expect(cssRaw).toContain('0.8s linear infinite')
  })
})

describe('Hub sidebar active state (Pitfall 8 / HUB-01)', () => {
  it('.sidebar__item--active rule is present', () => {
    expect(cssRaw).toContain('.sidebar__item--active')
  })

  it('.sidebar__item--active uses var(--hub-accent) for color', () => {
    const activeIdx = cssRaw.indexOf('.sidebar__item--active')
    expect(activeIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', activeIdx)
    const block = cssRaw.slice(activeIdx, blockEnd)
    expect(block).toContain('var(--hub-accent')
  })
})

describe('Hub group header (mirrors .remote-panel__peer-header)', () => {
  it('.hub__group-header is declared', () => {
    expect(cssRaw).toContain('.hub__group-header')
  })

  it('.hub__group-header has font-size: 11px', () => {
    const headerIdx = cssRaw.indexOf('.hub__group-header')
    expect(headerIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', headerIdx)
    const block = cssRaw.slice(headerIdx, blockEnd)
    expect(block).toContain('font-size: 11px')
  })

  it('.hub__group-header has font-weight: 600', () => {
    const headerIdx = cssRaw.indexOf('.hub__group-header')
    expect(headerIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', headerIdx)
    const block = cssRaw.slice(headerIdx, blockEnd)
    expect(block).toContain('font-weight: 600')
  })

  it('.hub__group-header has text-transform: uppercase', () => {
    const headerIdx = cssRaw.indexOf('.hub__group-header')
    expect(headerIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', headerIdx)
    const block = cssRaw.slice(headerIdx, blockEnd)
    expect(block).toContain('text-transform: uppercase')
  })

  it('.hub__group-header has letter-spacing: 0.08em', () => {
    const headerIdx = cssRaw.indexOf('.hub__group-header')
    expect(headerIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', headerIdx)
    const block = cssRaw.slice(headerIdx, blockEnd)
    expect(block).toContain('letter-spacing: 0.08em')
  })
})

describe('Hub colorblind-safe source comments (source-level UAT verification)', () => {
  it('has COLORBLIND-SAFE comment for running status dot dark hex #3b82f6', () => {
    expect(cssRaw).toContain('#3b82f6')
    expect(cssRaw).toContain('running')
  })

  it('has COLORBLIND-SAFE comment for idle status dark hex #22c55e', () => {
    expect(cssRaw).toContain('#22c55e')
  })

  it('has COLORBLIND-SAFE comment for stopped/done dark hex #565f89', () => {
    expect(cssRaw).toContain('#565f89')
  })

  it('has source-level COLORBLIND-SAFE comment block for light hex #1d4ed8 (running)', () => {
    expect(cssRaw).toContain('#1d4ed8')
    expect(cssRaw).toContain('7.1:1')
  })
})

// Phase 135 CSS contract — GAP-135-A focus-visible rings + GAP-135-E/F reduced-motion
describe('Phase 135 — GAP-135-A: focus-visible rings for Hub interactive elements', () => {
  it('A11Y-02: .hub-card:focus-visible rule is present (upgraded from :focus)', () => {
    expect(cssRaw).toContain('.hub-card:focus-visible')
  })

  it('A11Y-02: .hub-card:focus (bare, without -visible) outline rule is NOT present (Pitfall 2 guard)', () => {
    // The only :focus without -visible on hub-card must be :focus-within (structural, not outline)
    // Assert no standalone .hub-card:focus { outline rule exists
    expect(cssRaw).not.toMatch(/\.hub-card:focus\s*\{[^}]*outline/)
  })

  it('A11Y-02: .hub-filter__pill:focus-visible rule is present', () => {
    expect(cssRaw).toContain('.hub-filter__pill:focus-visible')
  })

  it('A11Y-02: .hub-card__open:focus-visible rule is present', () => {
    expect(cssRaw).toContain('.hub-card__open:focus-visible')
  })

  it('A11Y-02: .hub-modal__close:focus-visible rule is present', () => {
    expect(cssRaw).toContain('.hub-modal__close:focus-visible')
  })

  it('A11Y-02: .hub__group-sidebar-item:focus-visible rule is present', () => {
    expect(cssRaw).toContain('.hub__group-sidebar-item:focus-visible')
  })

  it('A11Y-02: all new :focus-visible rules use var(--hub-accent) token (colorblind constraint)', () => {
    // Extract the GAP-135-A grouped rule block — find it by locating .hub-filter__pill:focus-visible
    const pillIdx = cssRaw.indexOf('.hub-filter__pill:focus-visible')
    expect(pillIdx).toBeGreaterThan(-1)
    // Find the closing brace of the group rule
    const blockEnd = cssRaw.indexOf('}', pillIdx)
    const block = cssRaw.slice(pillIdx, blockEnd)
    expect(block).toContain('var(--hub-accent)')
  })

  it('A11Y-02: .hub-filter__search:focus is still present and unchanged (inputs keep :focus per WCAG 2.4.7)', () => {
    expect(cssRaw).toContain('.hub-filter__search:focus')
  })

  it('A11Y-02: .hub-modal__respond-input:focus is still present and unchanged (inputs keep :focus per WCAG 2.4.7)', () => {
    expect(cssRaw).toContain('.hub-modal__respond-input:focus')
  })
})

describe('Phase 135 — GAP-135-E/F: prefers-reduced-motion reduce blocks', () => {
  it('GAP-135-E: prefers-reduced-motion: reduce block targets .hub-card__status-icon--spin with animation: none', () => {
    expect(cssRaw).toMatch(/\.hub-card__status-icon--spin\s*\{\s*animation:\s*none/)
  })

  it('GAP-135-F: prefers-reduced-motion: reduce block targets .hub-card with transition: none', () => {
    // Must be inside a reduce block — find a reduce block containing .hub-card { transition: none
    expect(cssRaw).toMatch(/prefers-reduced-motion:\s*reduce[^@]*\.hub-card\s*\{\s*transition:\s*none/)
  })

  it('GAP-135-E/F: spin and card-hover reduce blocks appear AFTER the no-preference .hub-card 400ms transition override', () => {
    // The no-preference block sets .hub-card transition to 400ms ease (ATTN-03)
    const noPrefIdx = cssRaw.indexOf('border-color 400ms ease, box-shadow 400ms ease, background 100ms ease')
    expect(noPrefIdx).toBeGreaterThan(-1)

    // GAP-135-E spin reduce block must come after no-preference
    const spinReduceIdx = cssRaw.indexOf('.hub-card__status-icon--spin {\n    animation: none')
    expect(spinReduceIdx).toBeGreaterThan(noPrefIdx)

    // GAP-135-F card hover reduce block must also come after no-preference
    const cardTransitionNoneIdx = cssRaw.indexOf('.hub-card {\n    transition: none')
    expect(cardTransitionNoneIdx).toBeGreaterThan(noPrefIdx)
  })

  it('GAP-135-E/F: existing no-preference 400ms hub-card transition override is still present (not overwritten)', () => {
    expect(cssRaw).toContain('border-color 400ms ease, box-shadow 400ms ease, background 100ms ease')
  })
})

// Phase 133 CSS contract — attention pulse + badge (CR-02, CR-03, IN-01)
describe('Hub Phase 133 CSS contract', () => {
  it('CR-02: .hub-card--attention has opacity:1 (dim override invariant)', () => {
    // Find the .hub-card--attention rule and confirm opacity:1 is set
    const attnIdx = cssRaw.indexOf('.hub-card--attention {')
    expect(attnIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', attnIdx)
    const block = cssRaw.slice(attnIdx, blockEnd)
    expect(block).toContain('opacity: 1')
  })

  it('CR-03: .hub__group-sidebar-item__attn-badge--count rule is present', () => {
    expect(cssRaw).toContain('.hub__group-sidebar-item__attn-badge--count')
  })

  it('CR-03: attn-badge--count rule uses var(--hub-attn-badge-text) for color', () => {
    const countIdx = cssRaw.indexOf('.hub__group-sidebar-item__attn-badge--count')
    expect(countIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', countIdx)
    const block = cssRaw.slice(countIdx, blockEnd)
    expect(block).toContain('var(--hub-attn-badge-text)')
  })

  it('IN-01: .hub-card has position: relative (anchor for drag-handle/menu-btn)', () => {
    const cardIdx = cssRaw.indexOf('.hub-card {')
    expect(cardIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', cardIdx)
    const block = cssRaw.slice(cardIdx, blockEnd)
    expect(block).toContain('position: relative')
  })
})

// Phase 138 / CARD-03: Connection indicator CSS — remote card connected/available states.
// These assertions are RED until Plan 03 adds the CSS classes to style.css.
// COLORBLIND-SAFE: icon shape + text carry the state; color is reinforcement only.
describe('CARD-03: Connection indicator CSS (hub-card__conn)', () => {
  it('defines .hub-card__conn class', () => {
    expect(cssRaw).toContain('.hub-card__conn')
  })
  it('defines .hub-card__conn--connected modifier', () => {
    expect(cssRaw).toContain('.hub-card__conn--connected')
  })
  it('defines .hub-card__conn-icon class', () => {
    expect(cssRaw).toContain('.hub-card__conn-icon')
  })
})

// Phase 138 / CARD-04: Preserved grid CSS — anti-regression against accidental deletion.
// These must PASS against current style.css (they already exist).
describe('CARD-04: Preserved grid CSS (anti-regression)', () => {
  it('preserves .hub__card-row grid definition', () => {
    expect(cssRaw).toContain('.hub__card-row')
  })
  it('preserves .hub-card--attention class', () => {
    expect(cssRaw).toContain('.hub-card--attention')
  })
  it('preserves hub-card min-width 240px constraint', () => {
    expect(cssRaw).toContain('240px')
  })
})

// Phase 138 / CARD-04: Kill menu item CSS — destructive action styling.
// These assertions are RED until Plan 03 adds the CSS classes to style.css.
describe('CARD-04: Kill menu item CSS', () => {
  it('defines .hub-card__menu-item--destructive modifier', () => {
    expect(cssRaw).toContain('.hub-card__menu-item--destructive')
  })
  it('destructive color uses --hub-destructive custom property', () => {
    expect(cssRaw).toContain('--hub-destructive')
  })
})

// ============================================================
// Phase 142 comp-fidelity CSS tokens (GAP-04 anti-regression)
// Tokens verified against frontend/src/style.css on 2026-06-21.
// COLORBLIND-SAFE convention: hex literals are asserted at source, not by eye.
// ============================================================

describe('Phase 142 comp-fidelity CSS tokens (GAP-04 anti-regression)', () => {
  // (a) .hub-card block contains border-radius: 16px
  it('GAP-04: .hub-card block contains border-radius: 16px (pronounced rounded corners)', () => {
    // Find the .hub-card { rule block and assert border-radius is present
    const hubCardIdx = cssRaw.indexOf('.hub-card {')
    expect(hubCardIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', hubCardIdx)
    const block = cssRaw.slice(hubCardIdx, blockEnd)
    expect(block).toContain('border-radius: 16px')
  })

  // (b) session-type spine: .hub-card[data-agent="claude"] with border-left: 3px solid #7aa2f7
  // Colorblind-safe: hex pinned at source (periwinkle; also visible as shape cue via left border)
  it('GAP-04: .hub-card[data-agent="claude"] spine rule has border-left: 3px solid #7aa2f7', () => {
    // Find the claude data-agent spine rule
    const spineIdx = cssRaw.indexOf('.hub-card[data-agent="claude"]   { border-left: 3px solid #7aa2f7; }')
    expect(spineIdx).toBeGreaterThan(-1)
  })

  it('GAP-04: all 9 agent spine rules are present (claude, opencode, codex, gemini, cursor, aider, shell, unknown, agy)', () => {
    expect(cssRaw).toContain('.hub-card[data-agent="claude"]')
    expect(cssRaw).toContain('.hub-card[data-agent="opencode"]')
    expect(cssRaw).toContain('.hub-card[data-agent="codex"]')
    expect(cssRaw).toContain('.hub-card[data-agent="gemini"]')
    expect(cssRaw).toContain('.hub-card[data-agent="cursor"]')
    expect(cssRaw).toContain('.hub-card[data-agent="aider"]')
    expect(cssRaw).toContain('.hub-card[data-agent="shell"]')
    expect(cssRaw).toContain('.hub-card[data-agent="unknown"]')
    expect(cssRaw).toContain('.hub-card[data-agent="agy"]')
  })

  // (c) color-coded badge chip: .hub-card__badge and claude tint rule with color: #7aa2f7
  it('GAP-04: .hub-card__badge base rule is present', () => {
    expect(cssRaw).toContain('.hub-card__badge')
  })

  it('GAP-04: .hub-card[data-agent="claude"] .hub-card__badge chip tint uses color: #7aa2f7', () => {
    // Verify the exact source rule for the claude badge chip color
    const chipIdx = cssRaw.indexOf('.hub-card[data-agent="claude"]   .hub-card__badge { color: #7aa2f7;')
    expect(chipIdx).toBeGreaterThan(-1)
  })

  // (d) mini-preview window: height: 150px (note: height, not min-height — per interfaces block)
  it('GAP-04: .hub-card__preview block declares height: 150px (not min-height)', () => {
    const previewIdx = cssRaw.indexOf('.hub-card__preview {')
    expect(previewIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', previewIdx)
    const block = cssRaw.slice(previewIdx, blockEnd)
    expect(block).toContain('height: 150px')
    // Negative: must not be min-height only (the interface block specifies `height`)
    expect(block).not.toMatch(/min-height:\s*150px/)
  })
})

// ============================================================
// Phase 149-02: agy (Google Antigravity) agent badge color identity
// D-06: color #ff9e64 (TokyoNight orange) locked at source
// D-07: WCAG numbers honest — chip text on --hub-surface-elevated: dark 8.16:1 PASS, light 1.73:1 FAIL
//       (corrected in WR-02; earlier 8.72:1/2.03:1 measured against the wrong reference)
// D-08: all three color sites updated in lockstep (tab dot, card spine, card chip)
// D-09: key is 'agy' (binary name), NOT 'antigravity'
// COLORBLIND-SAFE: hex verified at source, not by eye (user is colorblind)
// ============================================================

describe('Phase 149-02 — agy agent badge color identity', () => {
  it('D-09: data-agent key is agy, NOT antigravity (key must match binary name)', () => {
    expect(cssRaw).not.toContain('data-agent="antigravity"')
  })

  it('D-06/D-08 Site 1: .tab__agent-badge--agy tab dot rule with #ff9e64 is present', () => {
    expect(cssRaw).toContain('.tab__agent-badge--agy')
    expect(cssRaw).toContain('.tab__agent-badge--agy')
    // The tab dot rule must use the locked agy color
    const tabIdx = cssRaw.indexOf('.tab__agent-badge--agy')
    expect(tabIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', tabIdx)
    const block = cssRaw.slice(tabIdx, blockEnd)
    expect(block).toContain('#ff9e64')
  })

  it('D-06/D-08 Site 2: .hub-card[data-agent="agy"] spine rule with border-left: 3px solid #ff9e64 is present', () => {
    const spineIdx = cssRaw.indexOf('.hub-card[data-agent="agy"]')
    expect(spineIdx).toBeGreaterThan(-1)
    const blockEnd = cssRaw.indexOf('}', spineIdx)
    const block = cssRaw.slice(spineIdx, blockEnd)
    expect(block).toContain('border-left: 3px solid #ff9e64')
  })

  it('D-06/D-08 Site 3: .hub-card[data-agent="agy"] .hub-card__badge chip rule with color: #ff9e64 is present', () => {
    // Use a regex to find the agy chip rule regardless of whitespace alignment
    const match = cssRaw.match(/\.hub-card\[data-agent="agy"\]\s+\.hub-card__badge\s*\{([^}]+)\}/)
    expect(match).not.toBeNull()
    expect(match![1]).toContain('color: #ff9e64')
  })

  it('D-06/D-08 Site 3: agy chip rule uses rgba(255, 158, 100, 0.45) border-color', () => {
    const match = cssRaw.match(/\.hub-card\[data-agent="agy"\]\s+\.hub-card__badge\s*\{([^}]+)\}/)
    expect(match).not.toBeNull()
    expect(match![1]).toContain('rgba(255, 158, 100, 0.45)')
  })

  it('D-07/WR-02: WCAG comment documents dark contrast 8.16:1 (PASS) against the actual chip surface', () => {
    // Chip text #ff9e64 on --hub-surface-elevated (#1c1e28); 8.16:1 must be present at source
    expect(cssRaw).toContain('8.16:1')
  })

  it('D-07/WR-02: WCAG comment documents light contrast 1.73:1 (FAIL) — honest, no false AA pass claim', () => {
    // Must document the actual light-mode failure against the real surface (#ececf0), honestly
    expect(cssRaw).toContain('1.73:1')
  })
})
