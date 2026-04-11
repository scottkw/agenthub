import React, { useEffect, useRef } from 'react'
import { Terminal } from '@xterm/xterm'
import type { ITheme } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebglAddon } from '@xterm/addon-webgl'
import { RelayClient } from '../lib/relayClient'

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
}

/**
 * TerminalPanel creates and owns one xterm.js Terminal instance.
 * All panels render simultaneously; inactive ones are hidden via display:none
 * to preserve the terminal buffer state without destroying the DOM node.
 */
export function TerminalPanel({ sessionId, isActive, relayPort, fontSize, onFontSizeChange, theme }: TerminalPanelProps): React.ReactElement {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const clientRef = useRef<RelayClient | null>(null)

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
    const unicode11 = new Unicode11Addon()
    term.loadAddon(fitAddon)
    term.loadAddon(unicode11)
    term.unicode.activeVersion = '11'   // TERM-03: emoji + CJK + box-drawing

    // Attempt WebGL renderer; fall back gracefully on context loss.
    try {
      const webglAddon = new WebglAddon()
      webglAddon.onContextLoss(() => {
        console.warn(`[TerminalPanel] WebGL context lost for session ${sessionId}; using canvas fallback`)
        webglAddon.dispose()
      })
      term.loadAddon(webglAddon)
    } catch (err) {
      console.warn(`[TerminalPanel] WebGL unavailable for session ${sessionId}:`, err)
    }

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
      term.dispose()
      termRef.current = null
      fitAddonRef.current = null
      clientRef.current = null
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  // onFontSizeChange intentionally omitted: stable callback captured once per session
  }, [sessionId])

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
  // refresh(0, rows-1) forces a full repaint so hidden panels update when revealed.
  useEffect(() => {
    if (!termRef.current) return
    termRef.current.options.theme = theme
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
