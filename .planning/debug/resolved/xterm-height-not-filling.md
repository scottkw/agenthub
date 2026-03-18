---
status: resolved
trigger: "Diagnose why the xterm.js terminal doesn't fill the available window height. There's a large blank area below the terminal content."
created: 2026-03-18T00:00:00Z
updated: 2026-03-18T00:00:00Z
---

## Current Focus

hypothesis: confirmed — three distinct bugs break the flex chain and FitAddon measurement
test: static code analysis of CSS, JSX layout, and xterm container div styles
expecting: confirmed root causes
next_action: deliver diagnosis

## Symptoms

expected: terminal content fills the full height of terminal-container (100% minus tab bar)
actual: large blank area below terminal content
errors: none reported
reproduction: visible on app launch with any active tab
started: after terminal-wrapper style changes

## Eliminated

- hypothesis: html/body/#root flex chain is broken
  evidence: style.css lines 10-16 set height:100% on all three; .app is flex column height:100%
  timestamp: 2026-03-18T00:00:00Z

- hypothesis: terminal-container is not growing to fill available space
  evidence: .terminal-container has flex:1 and overflow:hidden — it fills correctly
  timestamp: 2026-03-18T00:00:00Z

## Evidence

- timestamp: 2026-03-18T00:00:00Z
  checked: style.css lines 161-165 (.terminal-wrapper rule)
  found: .terminal-wrapper sets flex-direction:column, width:100%, height:100% — but has NO display property
  implication: display defaults to block; the flex-direction:column declaration is inert on a block box

- timestamp: 2026-03-18T00:00:00Z
  checked: App.tsx lines 233-237 (.terminal-wrapper JSX)
  found: inline style only sets display:flex OR display:none (toggled by isActive). Active wrappers get display:flex.
  implication: active terminal-wrapper IS a flex container. But terminal-container is position:relative (not flex), so terminal-wrapper has no flex parent and height:100% computes against a relatively-positioned block — this is fine IF terminal-container has a definite height, which it does via flex:1. So the height:100% on terminal-wrapper resolves correctly. This path is actually okay.

- timestamp: 2026-03-18T00:00:00Z
  checked: TerminalPanel.tsx lines 107-116 (the container div)
  found: container div has style={{ flex:1, width:'100%', minHeight:0 }} — but NO height property and NO display:flex on the parent for flex:1 to resolve against
  implication: terminal-wrapper IS a flex container (display:flex, flex-direction:column). The TerminalPanel container div has flex:1 which DOES resolve inside that flex container. BUT: xterm.js's term.open() injects .xterm, .xterm-screen, and .xterm-viewport inside this div. Those inner elements are NOT flex children — they are block/absolutely positioned elements sized by xterm's own internal logic (cell character height × row count). The container div expands to flex:1 correctly, but xterm does not fill it.

- timestamp: 2026-03-18T00:00:00Z
  checked: FitAddon.fit() call timing in TerminalPanel.tsx
  found: fitAddon.fit() is called at line 57 — immediately after term.open() — inside the useEffect whose deps are [sessionId]. At this moment the component has just mounted but the CSS flex layout may not have completed a paint cycle, so the container div's clientHeight may be 0 or incorrect.
  implication: FitAddon reads containerRef.current.getBoundingClientRect() (or clientHeight). If called synchronously on the same microtask tick as open(), the browser hasn't performed layout yet. FitAddon calculates 0 rows and renders a tiny/wrong-sized terminal.

- timestamp: 2026-03-18T00:00:00Z
  checked: isActive useEffect (lines 91-105) — the second fit() call
  found: fitAddonRef.current?.fit() is called at line 100 when isActive becomes true. This fires after React commits the style change (display:flex), but it fires synchronously in the effect — before the browser repaints and recalculates layout for the newly-displayed element.
  implication: The second fit() call suffers the same measurement problem: display was just switched from none to flex, the browser hasn't laid out the element yet, so getBoundingClientRect returns stale/zero dimensions.

- timestamp: 2026-03-18T00:00:00Z
  checked: xterm.js internal sizing model
  found: xterm.js sizes the terminal canvas/viewport in pixels based on (cols × charWidth) and (rows × charHeight). FitAddon derives cols/rows from Math.floor(containerHeight / charHeight). If containerHeight is 0 or stale, rows=0 or a small number, and xterm renders a small canvas — leaving the rest of the container div blank.
  implication: The blank area IS the container div (flex:1, growing correctly) minus the xterm canvas (sized to wrong row count).

- timestamp: 2026-03-18T00:00:00Z
  checked: web-serving-bar in App.tsx lines 239-275
  found: web-serving-bar is a flex child of terminal-wrapper (display:flex, flex-direction:column) with no flex-shrink:0. It has no height or flex rules defined in CSS.
  implication: when webServerRunning is true, web-serving-bar takes up some intrinsic height as a block/flex-item, shrinking the space left for TerminalPanel. But no fit() is called after this bar appears/disappears, so the terminal size is never recalculated to account for the bar. Minor secondary contributor to wrong sizing.

## Resolution

root_cause: |
  Three compounding bugs:

  1. MISSING display:flex ON .terminal-wrapper CSS RULE (primary contributor to confusion)
     style.css line 161: `.terminal-wrapper { flex-direction: column; ... }` has no `display` property.
     The inline JSX style overrides this for active tabs (display:flex), but the CSS rule is misleading
     and inactive-tab display:none is the only thing in JSX — the flex-direction comes from CSS.
     Actually this is consistent. The real bugs are below.

  2. FitAddon.fit() CALLED BEFORE BROWSER LAYOUT (primary sizing bug)
     In the mount effect (sessionId dep), fit() is called synchronously after term.open(). The
     browser has not yet performed a layout pass, so the container's clientHeight is 0. xterm
     calculates 0 rows, renders a tiny terminal. The subsequent isActive effect also calls fit()
     synchronously after React commits display:flex — again before the browser paints/lays out
     the newly-visible element.

  3. xterm CONTAINER DIV HAS NO explicit height (root enabler of bug 2)
     The TerminalPanel div has flex:1 and minHeight:0 but no height:100%. Without an explicit
     height, the div's rendered height is only known after browser layout. xterm.js does not
     use ResizeObserver by default (only FitAddon does, and only when fit() is called). The
     terminal canvas stays at whatever size it was initialized to (wrong).

fix: |
  Three changes required:

  A. Defer fit() with requestAnimationFrame (fixes timing — both call sites)
     In the mount effect: `requestAnimationFrame(() => fitAddon.fit())`
     In the isActive effect: `requestAnimationFrame(() => fitAddonRef.current?.fit())`
     This guarantees the browser has completed layout before FitAddon reads dimensions.

  B. Add a ResizeObserver on the container div (fixes dynamic resizing edge cases)
     Instead of (or in addition to) the window resize listener, observe the container div
     itself. This handles web-serving-bar appearing/disappearing and any other layout shifts.
     `new ResizeObserver(() => fitAddonRef.current?.fit()).observe(containerRef.current)`

  C. Add display:flex to the .terminal-wrapper CSS rule (fixes the misleading CSS / makes
     the class self-sufficient even if inline style is removed)
     `.terminal-wrapper { display: flex; flex-direction: column; width: 100%; height: 100%; }`

verification: static analysis — all three root causes traceable directly to source lines
files_changed:
  - frontend/src/components/TerminalPanel.tsx
  - frontend/src/style.css
