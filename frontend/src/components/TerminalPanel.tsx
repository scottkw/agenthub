import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Terminal } from '@xterm/xterm'
import type { ITheme, IDisposable } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { ImageAddon } from '@xterm/addon-image'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebglAddon } from '@xterm/addon-webgl'
import { ClipboardAddon } from '@xterm/addon-clipboard'
import { SearchAddon } from '@xterm/addon-search'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { SerializeAddon } from '@xterm/addon-serialize'
import { RelayClient } from '../lib/relayClient'
import { isSoftwareWebGL } from '../lib/webglProbe'
import { isXtermFocused } from '../lib/isXtermFocused'
import { FindBar, type FindBarSearchOptions } from './FindBar/FindBar'
import { SetSearchConfig } from '../wailsjs/go/main/App'
import { daemon } from '../wailsjs/go/models'
import { isAllowedScheme, getRisk, type RiskKind } from '../lib/urlSafety'
import { openLink, isModifierPressed, type ModifierMode } from '../lib/openLink'
import { LinkConfirmPopover } from './LinkConfirmPopover'
type PluginSettings = daemon.PluginSettings

// Custom fit that uses full container width (no hardcoded scrollbar deduction).
// FitAddon.fit() always subtracts DEFAULT_SCROLL_BAR_WIDTH (14px) even when the
// scrollbar is hidden via CSS and takes 0px — causing a permanent right-side gap.
function fitTerminal(term: Terminal): void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const core = (term as any)._core
  const dims = core._renderService.dimensions
  if (dims.css.cell.width === 0 || dims.css.cell.height === 0) return

  const parent = term.element?.parentElement
  if (!parent) return

  // Use clientWidth/Height minus padding for an unambiguous content-box size.
  // getComputedStyle().width can return border-box values with box-sizing:border-box,
  // which would include the container padding and make cols/rows too large.
  const parentStyle = window.getComputedStyle(parent)
  const parentW = parent.clientWidth - parseFloat(parentStyle.paddingLeft) - parseFloat(parentStyle.paddingRight)
  const parentH = parent.clientHeight - parseFloat(parentStyle.paddingTop) - parseFloat(parentStyle.paddingBottom)

  const cols = Math.max(2, Math.floor(parentW / dims.css.cell.width))
  const rows = Math.max(1, Math.floor(parentH / dims.css.cell.height))

  if (term.rows !== rows || term.cols !== cols) {
    core._renderService.clear()
    term.resize(cols, rows)
  }
}

interface TerminalPanelProps {
  sessionId: string
  isActive: boolean
  relayPort: number
  fontSize: number
  onFontSizeChange: (delta: number) => void
  theme: ITheme
  // Phase 93 PLUG-03/WGL-01/CLIP-01: pluginConfig is consumed in the hot-swap
  // useEffect to live-attach/dispose WebGL and Clipboard addons. Unicode 11
  // is honored at session init only (next-session semantics — UI-SPEC).
  pluginConfig?: PluginSettings | null
  // Phase 93 WGL-02/WGL-03: fired when WebGL falls back to DOM (context-loss)
  // or when software-rasterizer is detected at startup (preempted).
  onWebGLContextLost?: (reason: 'context-loss' | 'software-rasterized') => void
  /**
   * Phase 97 SER-01: register/unregister this panel's serialize() closure
   * with App.tsx's saver registry. Called inside the hot-swap useEffect
   * positive arm with a closure that captures the SerializeAddon ref;
   * called with null in the negative arm AND in mount-useEffect cleanup
   * to prevent registry memory leaks (Pitfall #6).
   */
  onRegisterSaver?: (sessionId: string, fn: (() => string) | null) => void
}

/**
 * TerminalPanel creates and owns one xterm.js Terminal instance.
 * All panels render simultaneously; inactive ones are hidden via display:none
 * to preserve the terminal buffer state without destroying the DOM node.
 */
export function TerminalPanel({
  sessionId,
  isActive,
  relayPort,
  fontSize,
  onFontSizeChange,
  theme,
  pluginConfig,
  onWebGLContextLost,
  onRegisterSaver,
}: TerminalPanelProps): React.ReactElement {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const clientRef = useRef<RelayClient | null>(null)
  // Phase 93 WGL-01 / CLIP-01: addon refs for hot-swap useEffect.
  const webglAddonRef = useRef<WebglAddon | null>(null)
  const clipboardAddonRef = useRef<ClipboardAddon | null>(null)
  // Phase 94 SRC-01..04: SearchAddon lifecycle + FindBar state. The addon
  // ref is hot-swapped from the same useEffect as webgl/clipboard (specific-
  // key dep array — Pitfall #1). The onDidChangeResults subscription is
  // disposed alongside the addon. The debounce timer is canceled on close +
  // unmount (Pitfall #10 — cancel-on-close prevents zombie searches).
  const searchAddonRef = useRef<SearchAddon | null>(null)
  const searchResultsDisposableRef = useRef<IDisposable | null>(null)
  const debounceTimerRef = useRef<number | null>(null)
  // Phase 95 LNK-01..04: WebLinksAddon lifecycle. Hot-swapped from the same
  // useEffect as webgl/clipboard/search (specific-key dep array — Pitfall #1).
  // The sub-config (modifier, confirm*) flows via webLinksConfigRef so
  // changing those does NOT re-attach the addon (Pitfall #8).
  const webLinksAddonRef = useRef<WebLinksAddon | null>(null)
  const webLinksConfigRef = useRef(pluginConfig?.webLinksConfig)
  // Phase 96 IMG-01/IMG-02: ImageAddon ref. Construction is NEXT-SESSION-ONLY
  // (mount useEffect, NOT hot-swap useEffect) — toggling pluginConfig.image
  // in Settings does NOT re-attach on already-open terminals. The italic
  // "Applies to new sessions you create." caption in PluginsSection is the
  // user-facing affordance for this constraint.
  const imageAddonRef = useRef<ImageAddon | null>(null)
  // Phase 97 SER-01: SerializeAddon ref. Construction is in the HOT-SWAP
  // useEffect (NOT mount) — Serialize is a pure buffer-walker with no
  // buffer-state implications, so it can be attached/detached at runtime.
  const serializeAddonRef = useRef<SerializeAddon | null>(null)

  // Phase 94 SRC-01/02: FindBar UI state. Owned at TerminalPanel level so
  // SearchAddon (also at this level) and FindBar share a single source of
  // truth. searchOptions are seeded from pluginConfig?.searchConfig at
  // mount only (Pitfall #2 — mid-open re-sync would surprise the user).
  const [findBarOpen, setFindBarOpen] = useState(false)
  // Phase 94 WR-01 / SC-4 — exit animation. While `findBarExiting` is true,
  // the FindBar receives an `exiting` prop that toggles the .find-bar--exiting
  // CSS modifier (translateY(-8px) over 200ms + opacity over 150ms — UI-SPEC
  // §Animation line 200). The actual unmount is delayed by 200ms via
  // findBarExitTimerRef so the transition completes. Re-opening during exit
  // cancels the pending timer (Pitfall #10 — no zombie state).
  const [findBarExiting, setFindBarExiting] = useState(false)
  const findBarExitTimerRef = useRef<number | null>(null)
  const [findBarFocusSeq, setFindBarFocusSeq] = useState(0)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchOptions, setSearchOptions] = useState<FindBarSearchOptions>(() => ({
    regex: pluginConfig?.searchConfig?.regex ?? false,
    caseSensitive: pluginConfig?.searchConfig?.caseSensitive ?? false,
    wholeWord: pluginConfig?.searchConfig?.wholeWord ?? false,
  }))
  // Phase 94-07 WR-02 / SC-2 (gap closure) — first-load seed for searchOptions.
  // The lazy initializer above runs ONLY on first render, when pluginConfig
  // is typically still null (App.tsx loads it async via GetPluginSettings).
  // This ref guards a one-shot useEffect that seeds searchOptions exactly
  // once when pluginConfig.searchConfig FIRST becomes non-null AND the
  // find bar is NOT currently open. Pitfall #2 invariant preserved:
  // mid-open re-seeds would surprise the user (e.g. SSE-pushed change
  // from another window clobbers a toggle they just clicked).
  const seededRef = useRef(false)
  useEffect(() => {
    if (seededRef.current) return
    if (!pluginConfig?.searchConfig) return
    if (findBarOpen) return // Pitfall #2 — never re-seed mid-open.
    setSearchOptions({
      regex: !!pluginConfig.searchConfig.regex,
      caseSensitive: !!pluginConfig.searchConfig.caseSensitive,
      wholeWord: !!pluginConfig.searchConfig.wholeWord,
    })
    seededRef.current = true
  }, [pluginConfig?.searchConfig, findBarOpen])
  const [matchInfo, setMatchInfo] = useState<{ index: number; count: number }>({ index: -1, count: 0 })

  // Phase 95 LNK-03: link-confirmation popover state. Set by the
  // WebLinksAddon click handler when getRisk(displayText, href) returns a
  // non-null kind AND the matching pluginConfig.webLinksConfig.confirm*
  // flag is true. Continue → openLink + clear; Cancel → clear.
  const [linkConfirmState, setLinkConfirmState] = useState<
    { url: string; risk: RiskKind; x: number; y: number } | null
  >(null)

  // Keep webLinksConfigRef in sync with sub-config changes WITHOUT triggering
  // an addon re-attach (Pitfall #8 — sub-config is read at click time).
  useEffect(() => {
    webLinksConfigRef.current = pluginConfig?.webLinksConfig
  }, [pluginConfig?.webLinksConfig])

  // Create the terminal and relay client once per sessionId.
  useEffect(() => {
    if (!containerRef.current) return

    const term = new Terminal({
      scrollback: 10000,         // TERM-04
      allowProposedApi: true,    // required for unicode11
      cursorBlink: true,
      fontFamily: '"Cascadia Code", "MesloLGS NF", "Fira Code", monospace',
      fontSize,
      theme,
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)

    // Phase 93 U11-01: Unicode 11 honors next-session-only semantics. The
    // pluginConfig?.unicode11 flag is read at session init; toggling it in
    // Settings does NOT mutate already-open terminals (UI-SPEC § Interaction
    // Contract — italic caption "Applies to new sessions you create."
    // explains this affordance to users). Default true if pluginConfig
    // hasn't loaded yet (preserves Phase 92 always-on behavior).
    if (pluginConfig?.unicode11 !== false) {
      const unicode11 = new Unicode11Addon()
      term.loadAddon(unicode11)
      term.unicode.activeVersion = '11'   // TERM-03: emoji + CJK + box-drawing
    }

    // Phase 96 IMG-01/IMG-02: ImageAddon is NEXT-SESSION-ONLY.
    // Lives in the MOUNT useEffect (alongside Unicode 11), NOT the
    // hot-swap useEffect — toggling pluginConfig.image in Settings
    // does NOT re-attach on already-open terminals. The italic
    // "Applies to new sessions you create." caption in PluginsSection
    // is the user-facing affordance for this constraint.
    //
    // enableSizeReports: false is MANDATORY (96-RESEARCH §"Pitfall 8":
    // CSI 14/16/18 t reports would leak terminal pixel dimensions to
    // the running CLI as keyboard input — privacy + security regression).
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
        // WASM instantiation failure is non-critical; sixel/IIP escapes
        // pass through harmlessly as printable garbage. No banner.
        // 96-RESEARCH §"Claude's Discretion / fall back gracefully".
        console.warn('Phase 96 IMG-01: ImageAddon construction failed', e)
      }
    }

    // WebGL + Clipboard addons load via the hot-swap useEffect below
    // (Phase 93 WGL-01/CLIP-01). Initial load is the same code path as
    // hot-swap because pluginConfig?.webgl/clipboard are dep-array keys.

    term.open(containerRef.current)
    // Don't fit here — hidden panels have zero dimensions.
    // The isActive effect handles fitting when a panel becomes visible.

    // Intercept SHIFT+= and SHIFT+- for font size control; return false to suppress PTY injection.
    term.attachCustomKeyEventHandler((ev: KeyboardEvent): boolean => {
      if (ev.type !== 'keydown') return true
      if (ev.shiftKey && ev.key === '=') { onFontSizeChange(+1); return false }
      if (ev.shiftKey && ev.key === '-') { onFontSizeChange(-1); return false }
      return true
    })

    termRef.current = term
    fitAddonRef.current = fitAddon

    // Connect relay client — one per terminal (TERM-01 independent sessions).
    // onOpen sends the current terminal dimensions to the PTY. This is critical:
    // fitTerminal() runs before the WS connects, so the onResize event from fit()
    // is silently dropped (WS not yet open). Without this, the CLI process never
    // learns the correct terminal size and renders to the wrong width.
    const client = new RelayClient(relayPort, sessionId, {
      onOutput: (data) => term.write(data),
      onOpen: () => {
        client.sendResize(term.cols, term.rows)
      },
      onClose: () => console.debug(`[RelayClient] disconnected session=${sessionId}`),
    })
    clientRef.current = client

    // Wire terminal input to relay (TERM-05: paste support via terminal.onData).
    const disposeData = term.onData((data) => client.sendInput(data))

    // Wire terminal resize to relay.
    const disposeResize = term.onResize(({ cols, rows }) => {
      client.sendResize(cols, rows)
    })

    return () => {
      disposeData.dispose()
      disposeResize.dispose()
      client.close()
      // Phase 93: dispose hot-swap addons before disposing the terminal
      // itself to avoid orphaned references. dispose() on the addon detaches
      // the render backend / clipboard handler — Terminal.dispose() then
      // tears down the buffer. Order matters only for cleanliness.
      if (webglAddonRef.current) {
        webglAddonRef.current.dispose()
        webglAddonRef.current = null
      }
      if (clipboardAddonRef.current) {
        clipboardAddonRef.current.dispose()
        clipboardAddonRef.current = null
      }
      // Phase 94 SRC-01..04: tear down SearchAddon + onDidChangeResults
      // subscription + pending debounce (HMR + unmount safety —
      // RESEARCH §"Pitfall #4" + §"Pitfall #10").
      if (searchResultsDisposableRef.current) {
        searchResultsDisposableRef.current.dispose()
        searchResultsDisposableRef.current = null
      }
      if (searchAddonRef.current) {
        searchAddonRef.current.dispose()
        searchAddonRef.current = null
      }
      // Phase 95 LNK-01..04: dispose WebLinksAddon on unmount so the
      // addon's link provider releases the buffer-scan subscription.
      if (webLinksAddonRef.current) {
        try { webLinksAddonRef.current.dispose() } catch { /* ignore */ }
        webLinksAddonRef.current = null
      }
      // Phase 96 IMG-01/IMG-02: dispose ImageAddon on unmount. Mirror
      // Phase 95 web-links cleanup style (try/catch + null the ref).
      if (imageAddonRef.current) {
        try { imageAddonRef.current.dispose() } catch { /* ignore */ }
        imageAddonRef.current = null
      }
      // Phase 97 SER-01: dispose serializeAddon AND flush the saver registry
      // entry on unmount (Pitfall #6 — leaving a stale closure behind would
      // mean handleRequestSave invokes a disposed addon).
      if (serializeAddonRef.current) {
        serializeAddonRef.current.dispose()
        serializeAddonRef.current = null
      }
      onRegisterSaver?.(sessionId, null)
      if (debounceTimerRef.current !== null) {
        window.clearTimeout(debounceTimerRef.current)
        debounceTimerRef.current = null
      }
      // Phase 94 WR-01: clear any pending exit-animation unmount so the
      // timer doesn't fire after the panel is gone (would no-op safely
      // because state setters on unmounted components are silent in React
      // 18+, but the ref state machine wants a clean teardown).
      if (findBarExitTimerRef.current !== null) {
        window.clearTimeout(findBarExitTimerRef.current)
        findBarExitTimerRef.current = null
      }
      term.dispose()
      termRef.current = null
      fitAddonRef.current = null
      clientRef.current = null
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  // onFontSizeChange + pluginConfig?.unicode11 intentionally omitted: mount
  // effect runs once per session; unicode11 is read at init (next-session
  // semantics, UI-SPEC § Interaction Contract).
  }, [sessionId])

  // Phase 93 hot-swap useEffect (WGL-01 + CLIP-01 + WGL-02 + WGL-03).
  // Lives AFTER the mount useEffect so termRef.current is set before this
  // first runs. Dep array keys are SPECIFIC fields (Pitfall #1 — putting
  // the whole pluginConfig object would re-run on every save even if the
  // relevant flags didn't change). Unicode 11 is intentionally NOT in this
  // dep array — it's next-session-only.
  useEffect(() => {
    const term = termRef.current
    if (!term) return

    // WebGL hot-swap (WGL-01) + software-rasterizer preemption (WGL-03)
    //                       + context-loss recovery (WGL-02)
    if (pluginConfig?.webgl) {
      if (!webglAddonRef.current) {
        if (isSoftwareWebGL()) {
          // Software-rasterized WebGL detected at startup — DOM renderer
          // is preemptively used; one-shot persistent toast informs user.
          onWebGLContextLost?.('software-rasterized')
        } else {
          try {
            const webglAddon = new WebglAddon()
            webglAddon.onContextLoss(() => {
              webglAddon.dispose()
              webglAddonRef.current = null
              onWebGLContextLost?.('context-loss')
            })
            term.loadAddon(webglAddon)
            webglAddonRef.current = webglAddon
          } catch (err) {
            // WebGL context creation failed — silent (no toast); user
            // explicitly enabled WebGL. Browser console still surfaces err.
            console.warn(`[TerminalPanel] WebGL unavailable for session ${sessionId}:`, err)
          }
        }
      }
    } else {
      // Toggle OFF — dispose addon if loaded. The Terminal's buffer
      // (scrollback) survives; only the render backend detaches.
      if (webglAddonRef.current) {
        webglAddonRef.current.dispose()
        webglAddonRef.current = null
      }
    }

    // Clipboard hot-swap (CLIP-01)
    if (pluginConfig?.clipboard) {
      if (!clipboardAddonRef.current) {
        const clipAddon = new ClipboardAddon()
        term.loadAddon(clipAddon)
        clipboardAddonRef.current = clipAddon
      }
    } else {
      if (clipboardAddonRef.current) {
        clipboardAddonRef.current.dispose()
        clipboardAddonRef.current = null
      }
    }

    // Phase 94 SRC-01..04: SearchAddon hot-swap. Loading is symmetric with
    // webgl/clipboard — single useEffect coordinates all three. The
    // onDidChangeResults subscription is created once when the addon
    // attaches and disposed when the addon detaches. Default highlight
    // limit (1000) — RESEARCH §"SearchAddon API Contract".
    if (pluginConfig?.search) {
      if (!searchAddonRef.current) {
        const searchAddon = new SearchAddon()
        term.loadAddon(searchAddon)
        searchAddonRef.current = searchAddon
        searchResultsDisposableRef.current = searchAddon.onDidChangeResults((e) => {
          setMatchInfo({ index: e.resultIndex, count: e.resultCount })
        })
      }
    } else {
      if (searchResultsDisposableRef.current) {
        searchResultsDisposableRef.current.dispose()
        searchResultsDisposableRef.current = null
      }
      if (searchAddonRef.current) {
        searchAddonRef.current.dispose()
        searchAddonRef.current = null
      }
      // Search disabled mid-session: close the bar + clear local state.
      setFindBarOpen(false)
      setSearchQuery('')
      setMatchInfo({ index: -1, count: 0 })
    }

    // Phase 95 LNK-01..04: WebLinksAddon hot-swap arm.
    // Sub-config (modifier, confirm*) is read at click time via
    // webLinksConfigRef.current so toggling sub-config does NOT re-attach
    // the addon (Pitfall #8). Only the boolean main toggle
    // (pluginConfig?.webLinks) drives load/dispose.
    //
    // Plan B (Wave 0 spike outcome — 95-RESEARCH §"Wave 0 Spike Outcome"):
    // OSC 8 mismatch detection deferred to v3.3 because the public
    // hyperlink-id accessor is absent from @xterm/xterm@6.0.0 typings. We
    // do NOT register a secondary link provider; v3.2 ships IDN +
    // typosquat detectors only. The osc8 branch in LinkConfirmPopover
    // ships dormant for the v3.3 wiring slice.
    if (pluginConfig?.webLinks) {
      if (!webLinksAddonRef.current) {
        const handler = (event: MouseEvent, uri: string) => {
          // LNK-01 defense-in-depth: handler re-validates the scheme even
          // though the addon's default regex already rejects non-allowlisted
          // schemes. A buggy upstream regex must NEVER punch through.
          if (!isAllowedScheme(uri)) return
          const cfg = webLinksConfigRef.current
          // LNK-02: modifier-click gate. Default 'platform' = Cmd on darwin
          // / Ctrl elsewhere. Modifier='none' bypass is intentional escape
          // hatch (Pitfall #9 — risk gates still fire below).
          // The daemon-typed `modifier` is `string`; narrow to ModifierMode
          // (the daemon's defaultPluginSettings + SetWebLinksConfig RPC
          // validation pin the value to one of these four literals — Plan
          // 95-05 enforces, here we accept and fall back defensively).
          //
          // WR-03: use truthy fallback (||) — matching web/assets/terminal.js
          // — so an empty-string `modifier` (corrupted settings.json edge
          // case) falls back to 'platform' instead of breaking the click
          // gate silently. Desktop and web must behave identically here
          // (UI-SPEC web parity mandate). WR-02's API-boundary validation
          // is the primary defense; this is the secondary belt-and-braces.
          const modifier = (cfg?.modifier || 'platform') as ModifierMode
          if (!isModifierPressed(event, modifier)) return
          // LNK-03: risk detection. For plain-text URL matches the addon
          // emits, displayText === uri, so osc8Mismatch never fires here —
          // OSC 8 mismatch is a v3.3 follow-up (Plan B; see header).
          const risk = getRisk(uri, uri)
          const shouldConfirm =
            (risk === 'osc8' && (cfg?.confirmOSC8 ?? true)) ||
            (risk === 'idn' && (cfg?.confirmIDN ?? true)) ||
            (risk === 'typosquat' && (cfg?.confirmTyposquat ?? true))
          if (risk && shouldConfirm) {
            setLinkConfirmState({ url: uri, risk, x: event.clientX, y: event.clientY })
            return
          }
          // LNK-04: platform-aware opener (Wails BrowserOpenURL on desktop;
          // window.open with noopener,noreferrer on web).
          openLink(uri)
        }
        const addon = new WebLinksAddon(handler, {
          // urlRegex undefined → use the addon's default scheme-restricted
          // regex; defense-in-depth re-checks scheme inside the handler (LNK-01).
          hover: (event: MouseEvent, uri: string) => {
            // Native title attribute is the simplest accessible tooltip;
            // xterm exposes the link element via this callback's event.target.
            if (event.target instanceof HTMLElement) {
              event.target.setAttribute('title', uri)
            }
          },
          leave: (event: MouseEvent) => {
            // Pitfall #10: BOTH hover AND leave callbacks are required.
            // Without leave, a stale tooltip persists over non-link cells
            // after the mouse moves off the link.
            if (event.target instanceof HTMLElement) {
              event.target.removeAttribute('title')
            }
          },
        })
        term.loadAddon(addon)
        webLinksAddonRef.current = addon
      }
    } else {
      // Toggle OFF — dispose addon. Terminal buffer (scrollback) survives;
      // only the link provider detaches.
      if (webLinksAddonRef.current) {
        try { webLinksAddonRef.current.dispose() } catch { /* ignore */ }
        webLinksAddonRef.current = null
      }
      // Clear any in-flight confirm popover so users don't get a stuck
      // dialog after disabling the feature.
      setLinkConfirmState(null)
    }

    // Phase 97 SER-01 hot-swap arm. Mirrors clipboard/webgl shape. Serialize is
    // hot-swap-friendly (pure buffer-walker, no buffer-state implications) —
    // distinct from Image and Unicode 11 which are mount-only per 97-PATTERNS.md
    // §"Hot-swap addon arm contract". When attaching, register the addon's
    // serialize() closure with App.tsx's saver registry so TabBar's right-click
    // "Save Terminal As…" can reach it.
    if (pluginConfig?.serialize) {
      if (!serializeAddonRef.current) {
        const serializeAddon = new SerializeAddon()
        term.loadAddon(serializeAddon)
        serializeAddonRef.current = serializeAddon
        onRegisterSaver?.(sessionId, () => serializeAddon.serialize({ excludeModes: true }))
      }
    } else {
      if (serializeAddonRef.current) {
        serializeAddonRef.current.dispose()
        serializeAddonRef.current = null
        onRegisterSaver?.(sessionId, null) // Pitfall #6 — flush stale closure
      }
    }
  }, [pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search, pluginConfig?.webLinks, pluginConfig?.serialize, onWebGLContextLost, onRegisterSaver, sessionId])

  // Fit when this panel becomes active, and track container size changes.
  useEffect(() => {
    if (!isActive || !containerRef.current) return

    const container = containerRef.current
    let cancelled = false
    let rafId: number | undefined
    const MAX_ATTEMPTS = 20  // ~333ms at 60fps; covers slow CLI startup delays

    const tryFit = (attempt: number) => {
      if (cancelled) return

      // proposeDimensions() returns undefined when css.cell.width === 0
      // (CharSizeService hasn't measured font yet — zero cell dims from display:none open())
      const dims = fitAddonRef.current?.proposeDimensions()
      if (dims !== undefined) {
        fitTerminal(termRef.current!)
        return
      }

      // Cell dimensions not ready — schedule next rAF attempt
      if (attempt < MAX_ATTEMPTS) {
        rafId = requestAnimationFrame(() => tryFit(attempt + 1))
      } else {
        // Best-effort fallback after max attempts
        fitTerminal(termRef.current!)
      }
    }

    // Initial rAF: ensure display:none -> flex layout change is committed
    rafId = requestAnimationFrame(() => tryFit(0))

    // ResizeObserver handles all subsequent size changes (window resize, font size change)
    const ro = new ResizeObserver(() => { if (termRef.current) fitTerminal(termRef.current) })
    ro.observe(container)

    return () => {
      cancelled = true
      if (rafId !== undefined) cancelAnimationFrame(rafId)
      ro.disconnect()
    }
  }, [isActive])

  // Apply font size changes from the controlled prop.
  useEffect(() => {
    if (!termRef.current || !fitAddonRef.current) return
    termRef.current.options.fontSize = fontSize
    fitTerminal(termRef.current)
  }, [fontSize])

  // Apply theme changes from the controlled prop (THM-03).
  // clearTextureAtlas() forces the WebGL renderer to rebuild its glyph cache
  // with the new colors — without this, WebGL panels keep the old palette.
  useEffect(() => {
    if (!termRef.current) return
    termRef.current.options.theme = theme
    termRef.current.clearTextureAtlas()
    termRef.current.refresh(0, termRef.current.rows - 1)
  }, [theme])

  // Phase 94 SRC-01: focus-conditioned Cmd-F (Mac) / Ctrl-F (Win/Linux)
  // window keydown listener. The isXtermFocused() guard mitigates T-94-03
  // by letting browser-native find pass through when focus is on a sibling
  // (sidebar / modal / settings). When the bar is already open, re-pressing
  // Cmd-F bumps focusSeq to re-focus the search input (UI-SPEC §"Opening
  // the Find Bar — when already open").
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent): void {
      if (!pluginConfig?.search) return
      const isMac = navigator.platform.toUpperCase().includes('MAC')
      const modifier = isMac ? e.metaKey : e.ctrlKey
      if (!modifier || e.key.toLowerCase() !== 'f') return
      if (!isXtermFocused(containerRef.current)) return
      e.preventDefault()
      // Phase 94 WR-01: re-opening during exit cancels the pending unmount
      // timer (Pitfall #10 — no zombie state). Drop the .find-bar--exiting
      // modifier so the bar re-appears at-rest, no flicker.
      if (findBarExitTimerRef.current !== null) {
        window.clearTimeout(findBarExitTimerRef.current)
        findBarExitTimerRef.current = null
      }
      setFindBarExiting(false)
      setFindBarOpen(true)
      setFindBarFocusSeq((s) => s + 1)
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [pluginConfig?.search])

  // Phase 94 SRC-01..04: FindBar callback handlers. The 100ms input debounce
  // (RESEARCH §"Pattern 4") coalesces keystrokes; toggle changes re-search
  // instantly from local state then fire-and-forget SetPluginSettings
  // (RESEARCH §"Pattern 3"; Pitfall #2 — no mid-open re-sync from props).
  const handleSearchQueryChange = useCallback((q: string) => {
    setSearchQuery(q)
    if (debounceTimerRef.current !== null) {
      window.clearTimeout(debounceTimerRef.current)
    }
    debounceTimerRef.current = window.setTimeout(() => {
      debounceTimerRef.current = null
      if (q === '') {
        searchAddonRef.current?.clearDecorations()
        setMatchInfo({ index: -1, count: 0 })
        return
      }
      // Phase 94 SRC-02 + SRC-04 reconciliation: pass decorations: {} so
      // SearchAddon._fireResults fires the onDidChangeResults event (it gates on
      // !!opts.decorations). The empty object registers invisible decoration
      // overlays — no per-theme color overrides — so the active match still
      // highlights via xterm core's selection (theme.selectionBackground) and
      // the 138-theme invariant holds. The forbidden color keys are listed
      // and source-inspected by FindBar.themeMatrix.test.tsx.
      // Discovered during Plan 94-05 chromedp e2e (web parity wave).
      searchAddonRef.current?.findNext(q, { ...searchOptions, decorations: {} as never })
    }, 100)
  }, [searchOptions])

  const handleSearchOptionsChange = useCallback((opts: FindBarSearchOptions) => {
    setSearchOptions(opts)
    // Phase 94 SRC-02 + SRC-04: see decorations comment in handleSearchQueryChange.
    if (searchQuery) searchAddonRef.current?.findNext(searchQuery, { ...opts, decorations: {} as never })
    // Phase 94-07 WR-03 (gap closure) — write ONLY the searchConfig sub-key.
    // Previously we constructed a full PluginSettings from the App-level
    // prop and called SetPluginSettings, which raced PluginsSection's
    // stale edit buffer (PluginsSection only fetches GetPluginSettings on
    // mount and does NOT subscribe to settings:plugins). The new
    // SetSearchConfig RPC mutates only e.pluginSettings.SearchConfig
    // under the engine mutex — PluginsSection's unsaved boolean edits
    // can no longer be clobbered by a find-bar persistence call.
    //
    // The Phase 93 PLUG-04 listener fires from the daemon side, so the
    // settings:plugins SSE event still pushes to web subscribers — SRC-05
    // web parity unchanged. App.SetSearchConfig also re-emits the Wails
    // "settings:plugins" runtime event so the desktop App.tsx listener
    // continues to receive a frame on every change.
    SetSearchConfig(new daemon.SearchConfig(opts)).catch(() => {
      /* silent — Settings panel surfaces persistence errors */
    })
  }, [searchQuery])

  const handleSearchNext = useCallback(() => {
    // Phase 94 SRC-02 + SRC-04: see decorations comment in handleSearchQueryChange.
    if (searchQuery) searchAddonRef.current?.findNext(searchQuery, { ...searchOptions, decorations: {} as never })
  }, [searchQuery, searchOptions])

  const handleSearchPrev = useCallback(() => {
    // Phase 94 SRC-02 + SRC-04: see decorations comment in handleSearchQueryChange.
    if (searchQuery) searchAddonRef.current?.findPrevious(searchQuery, { ...searchOptions, decorations: {} as never })
  }, [searchQuery, searchOptions])

  const handleSearchClose = useCallback(() => {
    if (debounceTimerRef.current !== null) {
      window.clearTimeout(debounceTimerRef.current)
      debounceTimerRef.current = null
    }
    searchAddonRef.current?.clearDecorations()
    setSearchQuery('')
    setMatchInfo({ index: -1, count: 0 })
    // Phase 94 WR-01 / SC-4 — play the 200ms exit transition before unmount.
    // CSS .find-bar--exiting (frontend/src/style.css:2180-2184): transform
    // 200ms ease + opacity 150ms ease. Match the longer (200ms) duration so
    // we unmount only after the slide-up completes. UI-SPEC §"Closing the
    // Find Bar" lines 304-311 (animation-first then unmount sequence).
    setFindBarExiting(true)
    if (findBarExitTimerRef.current !== null) {
      window.clearTimeout(findBarExitTimerRef.current)
    }
    findBarExitTimerRef.current = window.setTimeout(() => {
      findBarExitTimerRef.current = null
      setFindBarExiting(false)
      setFindBarOpen(false)
      // Return focus to the xterm helper textarea (xterm's internal input
      // sink). UI-SPEC §"Closing the Find Bar > Return focus to terminal".
      containerRef.current
        ?.querySelector<HTMLTextAreaElement>('.xterm-helper-textarea')
        ?.focus()
    }, 200)
  }, [])

  return (
    <div
      ref={containerRef}
      className="terminal-session-container"
      style={{
        flex: 1,
        width: '100%',
        minHeight: 0,
        backgroundColor: theme.background ?? '#1a1b26',
      }}
    >
      {(findBarOpen || findBarExiting) && pluginConfig?.search && (
        <FindBar
          query={searchQuery}
          onQueryChange={handleSearchQueryChange}
          matchCount={matchInfo.count}
          currentMatchIndex={matchInfo.index}
          searchOptions={searchOptions}
          onSearchOptionsChange={handleSearchOptionsChange}
          onNext={handleSearchNext}
          onPrev={handleSearchPrev}
          onClose={handleSearchClose}
          focusSeq={findBarFocusSeq}
          exiting={findBarExiting}
        />
      )}
      {linkConfirmState && (
        <LinkConfirmPopover
          url={linkConfirmState.url}
          risk={linkConfirmState.risk}
          x={linkConfirmState.x}
          y={linkConfirmState.y}
          onContinue={() => {
            openLink(linkConfirmState.url)
            setLinkConfirmState(null)
          }}
          onCancel={() => setLinkConfirmState(null)}
        />
      )}
    </div>
  )
}
