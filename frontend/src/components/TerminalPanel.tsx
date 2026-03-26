import React, { useEffect, useRef } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebglAddon } from '@xterm/addon-webgl'
import { RelayClient } from '../lib/relayClient'

interface TerminalPanelProps {
  sessionId: string
  isActive: boolean
  relayPort: number
  fontSize: number
  onFontSizeChange: (delta: number) => void
}

/**
 * TerminalPanel creates and owns one xterm.js Terminal instance.
 * All panels render simultaneously; inactive ones are hidden via display:none
 * to preserve the terminal buffer state without destroying the DOM node.
 */
export function TerminalPanel({ sessionId, isActive, relayPort, fontSize, onFontSizeChange }: TerminalPanelProps): React.ReactElement {
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
      theme: { background: '#1a1b26' },
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
    const client = new RelayClient(relayPort, sessionId, {
      onOutput: (data) => term.write(data),
      onOpen: () => console.debug(`[RelayClient] connected session=${sessionId}`),
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
        fitAddonRef.current?.fit()
        return
      }

      // Cell dimensions not ready — schedule next rAF attempt
      if (attempt < MAX_ATTEMPTS) {
        rafId = requestAnimationFrame(() => tryFit(attempt + 1))
      } else {
        // Best-effort fallback after max attempts
        fitAddonRef.current?.fit()
      }
    }

    // Initial rAF: ensure display:none -> flex layout change is committed
    rafId = requestAnimationFrame(() => tryFit(0))

    // ResizeObserver handles all subsequent size changes (window resize, font size change)
    const ro = new ResizeObserver(() => fitAddonRef.current?.fit())
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
    fitAddonRef.current.fit()
  }, [fontSize])

  return (
    <div
      ref={containerRef}
      style={{
        flex: 1,
        width: '100%',
        minHeight: 0,
      }}
    />
  )
}
