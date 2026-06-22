/**
 * Phase 141 — Redesign Implementation: colorblind-safe theme-token guardrails.
 *
 * WHY THIS FILE EXISTS
 * --------------------
 * The user is COLORBLIND. Color correctness for the Refined-Native recolor
 * (Phase 141) cannot be verified by eye — it is verified at the SOURCE level by
 * asserting against the hex constants / token declarations in style.css.
 *
 * The original 141-VALIDATION.md contract leaned on brittle line-anchored
 * `sed -n 'A,Bp' style.css | grep` commands. Phase 142 subsequently edited
 * style.css, so those line ranges went STALE. This file replaces those checks
 * with structural (selector / declaration / comment-aware) parsing that survives
 * future edits — it NEVER references a fixed line number.
 *
 * Requirements covered:
 *   - RDS-02  — chosen redesign applied across surviving surfaces: no raw hex
 *               left behind in a migrated surface outside the D-03 fence.
 *   - RDS-04  — colorblind-safe semantics in BOTH light and dark themes:
 *               (a) every --hub-* token declared in :root is also declared in
 *                   [data-ui-theme="light"] and vice-versa (theme parity), so a
 *                   surface can never silently fall back to a dark hex in light
 *                   mode (the failure a colorblind user cannot self-detect);
 *               (b) motion is governed by @media (prefers-reduced-motion ...),
 *                   and the S-07 share-modal motion is gated.
 */

import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

// Resolve style.css relative to THIS test file (src/__tests__/ -> src/style.css)
// so the path survives regardless of the vitest cwd.
const __dirname = dirname(fileURLToPath(import.meta.url))
const STYLE_PATH = resolve(__dirname, '..', 'style.css')
const rawCss = readFileSync(STYLE_PATH, 'utf8')

// --- helpers -------------------------------------------------------------

const HEX_RE = /#[0-9a-fA-F]{3}(?:[0-9a-fA-F]{3})?\b/g

/** Strip /* ... *\/ block comments (multi-line aware) so hex inside comments
 *  and the doc text are never treated as live style. Returns a string of the
 *  same line count (comment bodies replaced with spaces) so line numbers in
 *  failure messages stay meaningful. */
function stripBlockComments(css: string): string {
  let out = ''
  let i = 0
  let inComment = false
  while (i < css.length) {
    if (!inComment && css[i] === '/' && css[i + 1] === '*') {
      inComment = true
      i += 2
      continue
    }
    if (inComment && css[i] === '*' && css[i + 1] === '/') {
      inComment = false
      i += 2
      continue
    }
    if (inComment) {
      // preserve newlines so line numbers are stable
      out += css[i] === '\n' ? '\n' : ' '
    } else {
      out += css[i]
    }
    i++
  }
  return out
}

/** Extract the body text of the FIRST top-level rule whose selector list
 *  matches `selectorMatch`. Brace-balanced; nested at-rules counted. Returns
 *  null if not found. Located by SELECTOR, never by line number. */
function extractBlockBody(css: string, selectorMatch: RegExp): string | null {
  const m = selectorMatch.exec(css)
  if (!m) return null
  // find the opening brace after the selector match
  let i = css.indexOf('{', m.index)
  if (i === -1) return null
  let depth = 0
  const start = i + 1
  for (; i < css.length; i++) {
    if (css[i] === '{') depth++
    else if (css[i] === '}') {
      depth--
      if (depth === 0) return css.slice(start, i)
    }
  }
  return null
}

/** Pull every custom-property NAME (--foo) declared directly in a block body. */
function declaredCustomProps(blockBody: string): Set<string> {
  const names = new Set<string>()
  const re = /(--[a-zA-Z0-9-]+)\s*:/g
  let m: RegExpExecArray | null
  while ((m = re.exec(blockBody)) !== null) names.add(m[1])
  return names
}

const cleanCss = stripBlockComments(rawCss)
const cleanLines = cleanCss.split('\n')

/**
 * The selector prefixes that Phase 141 migrated (per 141-RESEARCH.md
 * §Per-Surface Hex Inventory: sidebar, Welcome tab, tab bar / status bar /
 * terminal container, File Browser + Editor chrome, Settings, Share Modal).
 * These are the ONLY surfaces RDS-02 promised to fully tokenize. Other
 * surfaces (.new-session-modal, .qr-modal, .daemon-panel, .banner, etc.) were
 * explicitly OUT of Phase 141 scope and keep their pre-existing hex, so they
 * must NOT be flagged here.
 */
const MIGRATED_SURFACE_SELECTORS = [
  '.sidebar',
  '.welcome-tab',
  '.tab-bar',
  '.tab-status-bar',
  '.terminal-container',
  '.file-browser',
  '.settings-panel',
  '.settings-jump-bar',
  '.settings-search',
  '.hub-share-modal',
]

/**
 * D-03 FENCE — permitted residual hex. These are semantic colors that are
 * colorblind-safe by reinforcement (icon + text label carry the meaning); the
 * contract (141-VALIDATION.md "Permitted residual hex" + 141-RESEARCH.md §D-03
 * Boundary Confirmation) forbids migrating them to theme tokens.
 *
 * If the fence legitimately needs to grow, add the selector substring here and
 * note WHY (semantic identifier / status reinforcement) — the failure message
 * below tells you exactly which selector to consider.
 */
const D03_FENCE_SELECTOR_SUBSTRINGS = [
  '.tab__agent-badge--', // per-agent semantic identifier
  '.tab-status-bar__state--', // WEB ON/OFF/NOT-RUNNING status; text label is primary
]

// For each LINE of the cleaned CSS, record the nearest preceding selector /
// at-rule header so we can attribute a hex occurrence to its enclosing rule
// structurally (no line numbers hard-coded).
function nearestSelectorForLine(lineIdx: number): string {
  for (let j = lineIdx; j >= 0; j--) {
    const t = cleanLines[j]
    // a selector / at-rule header line contains '{' and is not a property or
    // custom-property declaration line
    if (t.includes('{') && !/^\s*(--|[a-z-]+\s*:)/.test(t)) {
      return t.trim()
    }
  }
  return '<file scope>'
}

// Identify the token-definition blocks structurally so hex inside them (the
// legitimate token VALUES) is never flagged as a migration miss.
const rootBody = extractBlockBody(cleanCss, /(^|\n)\s*:root\s*\{/)
const lightBody = extractBlockBody(cleanCss, /\[data-ui-theme="light"\]\s*\{/)

// Compute the [start,end) line span of the two token blocks so the hex scan
// can skip them. Located via their unique selector text, not fixed numbers.
function blockLineSpan(headerRe: RegExp): [number, number] {
  const headerLine = cleanLines.findIndex((l) => headerRe.test(l))
  if (headerLine === -1) return [-1, -1]
  let depth = 0
  let started = false
  for (let j = headerLine; j < cleanLines.length; j++) {
    for (const ch of cleanLines[j]) {
      if (ch === '{') {
        depth++
        started = true
      } else if (ch === '}') depth--
    }
    if (started && depth === 0) return [headerLine, j]
  }
  return [headerLine, cleanLines.length - 1]
}
const rootSpan = blockLineSpan(/^\s*:root\s*\{/)
const lightSpan = blockLineSpan(/^\s*\[data-ui-theme="light"\]\s*\{/)

function inTokenBlock(lineIdx: number): boolean {
  return (
    (lineIdx >= rootSpan[0] && lineIdx <= rootSpan[1]) ||
    (lineIdx >= lightSpan[0] && lineIdx <= lightSpan[1])
  )
}

// ---------------------------------------------------------------------------

describe('Phase 141 theme tokens — colorblind source-level guardrails', () => {
  it('locates the :root and [data-ui-theme="light"] token blocks structurally', () => {
    // Sanity: if the parser cannot find the blocks, every downstream assertion
    // is meaningless — fail loudly rather than pass vacuously.
    expect(rootBody, ':root block not found in style.css').not.toBeNull()
    expect(lightBody, '[data-ui-theme="light"] block not found in style.css').not.toBeNull()
    expect(rootSpan[0]).toBeGreaterThanOrEqual(0)
    expect(lightSpan[0]).toBeGreaterThanOrEqual(0)
  })

  // -------- RDS-02: no raw hex left in a migrated surface (outside D-03) -----
  it('RDS-02: migrated surfaces contain no raw hex outside the D-03 fence (colorblind: source-verified)', () => {
    const offenders: string[] = []

    for (let i = 0; i < cleanLines.length; i++) {
      if (inTokenBlock(i)) continue // legitimate token VALUE definitions

      const line = cleanLines[i]
      const matches = line.match(HEX_RE)
      if (!matches) continue

      const selector = nearestSelectorForLine(i)

      // only police surfaces Phase 141 promised to migrate
      const inScope = MIGRATED_SURFACE_SELECTORS.some((s) => selector.includes(s))
      if (!inScope) continue

      // skip the D-03 fence (semantic colors that must stay raw)
      const fenced = D03_FENCE_SELECTOR_SUBSTRINGS.some((s) => selector.includes(s))
      if (fenced) continue

      offenders.push(
        `style.css:${i + 1}  ${selector}  ->  ${matches.join(', ')}  (raw: ${line.trim()})`,
      )
    }

    expect(
      offenders,
      [
        'RDS-02 VIOLATION — raw hex found in a Phase-141 migrated surface that is',
        'NOT in the D-03 fence. Each must be migrated to a var(--hub-*) token (the',
        'user is colorblind; an un-tokenized hex will not flip in light theme and',
        'they cannot self-detect it). If a listed selector is a LEGITIMATE semantic',
        'color (icon/text carries the meaning), add its selector substring to',
        'D03_FENCE_SELECTOR_SUBSTRINGS with a rationale.',
        '',
        ...offenders,
      ].join('\n'),
    ).toEqual([])
  })

  // -------- RDS-04: theme-token parity (no light-theme orphans) -------------
  it('RDS-04: every --hub-* token is declared in BOTH :root and [data-ui-theme="light"]', () => {
    expect(rootBody).not.toBeNull()
    expect(lightBody).not.toBeNull()

    const rootProps = declaredCustomProps(rootBody as string)
    const lightProps = declaredCustomProps(lightBody as string)

    const missingInLight = [...rootProps].filter((p) => !lightProps.has(p)).sort()
    const missingInRoot = [...lightProps].filter((p) => !rootProps.has(p)).sort()

    expect(
      { missingInLight, missingInRoot },
      [
        'RDS-04 PARITY VIOLATION — a custom property is declared for one palette',
        'but not the other. The orphaned palette silently inherits the wrong',
        'value, which a colorblind user cannot detect by eye.',
        `  Declared in :root but missing in [data-ui-theme="light"]: ${JSON.stringify(missingInLight)}`,
        `  Declared in [data-ui-theme="light"] but missing in :root: ${JSON.stringify(missingInRoot)}`,
      ].join('\n'),
    ).toEqual({ missingInLight: [], missingInRoot: [] })
  })

  // -------- RDS-04: reduced-motion guards -----------------------------------
  it('RDS-04: motion is governed by @media (prefers-reduced-motion ...)', () => {
    const guardCount = (cleanCss.match(/@media\s*\(prefers-reduced-motion/g) || []).length
    expect(
      guardCount,
      'RDS-04 — expected at least one @media (prefers-reduced-motion ...) guard block',
    ).toBeGreaterThan(0)

    // The contract requires BOTH the no-preference (animate) and reduce (static)
    // arms of the established static-first motion pattern to be present.
    expect(cleanCss).toMatch(/@media\s*\(prefers-reduced-motion:\s*no-preference\)/)
    expect(cleanCss).toMatch(/@media\s*\(prefers-reduced-motion:\s*reduce\)/)
  })

  it('RDS-04 / S-07: share-modal enter/exit animation is gated by prefers-reduced-motion', () => {
    // The share-modal keyframes (hub-share-modal-in/out) are assigned to the
    // --entering/--exiting phase classes; that assignment MUST live inside a
    // no-preference guard, and a reduce arm must neutralise it. (141-RESEARCH
    // §Motion Contract; S-07.)
    const noPrefArm = extractBlockBody(
      cleanCss,
      /@media\s*\(prefers-reduced-motion:\s*no-preference\)\s*\{\s*\.hub-share-modal--entering/,
    )
    expect(
      noPrefArm,
      'S-07 — .hub-share-modal--entering animation must be inside a ' +
        '@media (prefers-reduced-motion: no-preference) block',
    ).not.toBeNull()
    expect(noPrefArm as string).toMatch(/animation:\s*hub-share-modal-in/)

    // And a reduce arm that disables the animation for reduced-motion users.
    const reduceArm = extractBlockBody(
      cleanCss,
      /@media\s*\(prefers-reduced-motion:\s*reduce\)\s*\{\s*\.hub-share-modal\b/,
    )
    expect(
      reduceArm,
      'S-07 — a @media (prefers-reduced-motion: reduce) arm must neutralise ' +
        '.hub-share-modal motion',
    ).not.toBeNull()
    expect(reduceArm as string).toMatch(/animation:\s*none/)
  })
})
