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
}

/**
 * TerminalPanel creates and owns one xterm.js Terminal instance.
 * All panels render simultaneously; inactive ones are hidden via display:none
 * to preserve the terminal buffer state without destroying the DOM node.
 */
export function TerminalPanel({ sessionId, isActive, relayPort }: TerminalPanelProps): React.ReactElement {
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
      fontSize: 14,
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
    fitAddon.fit()

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
  }, [sessionId])

  // Window resize: only fit the ACTIVE terminal (avoids hidden terminal sizing issues).
  useEffect(() => {
    if (!isActive) return

    const handleResize = () => {
      fitAddonRef.current?.fit()
    }

    window.addEventListener('resize', handleResize)
    // Fit immediately when this panel becomes active.
    fitAddonRef.current?.fit()

    return () => {
      window.removeEventListener('resize', handleResize)
    }
  }, [isActive])

  return (
    <div
      ref={containerRef}
      style={{
        display: isActive ? 'block' : 'none',
        width: '100%',
        height: '100%',
      }}
    />
  )
}
