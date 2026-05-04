import React, { useEffect, useRef } from 'react'
import { Terminal } from '@xterm/xterm'
import type { ITheme } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebglAddon } from '@xterm/addon-webgl'
import { ClipboardAddon } from '@xterm/addon-clipboard'
import { RelayClient } from '../lib/relayClient'
import { isSoftwareWebGL } from '../lib/webglProbe'
import type { daemon } from '../wailsjs/go/models'
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
}: TerminalPanelProps): React.ReactElement {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const clientRef = useRef<RelayClient | null>(null)
  // Phase 93 WGL-01 / CLIP-01: addon refs for hot-swap useEffect.
  const webglAddonRef = useRef<WebglAddon | null>(null)
  const clipboardAddonRef = useRef<ClipboardAddon | null>(null)

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
  }, [pluginConfig?.webgl, pluginConfig?.clipboard, onWebGLContextLost, sessionId])

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
    />
  )
}
