import React, { useState, useEffect, useRef } from 'react'
import { ClipboardSetText } from '../../wailsjs/wailsjs/runtime/runtime'

// ---------------------------------------------------------------------------
// Phase 173 / D-09 — hoisted from SessionSharePanel.tsx (verbatim) so the new
// per-tab components (TailnetTab / InternetReadOnlyTab / InternetFullAccessTab)
// and ShareLinkCard can share a single source of truth. See 173-02-PLAN.md.
//
// Phase 171 / FNL-09 — hold-to-confirm public write consent gate (D-01/D-07/R1)
// ---------------------------------------------------------------------------
export const HOLD_DURATION_MS = 3000
export const HOLD_TICK_MS = 100

/**
 * HoldToConfirmButton — the sole affordance that can mint a public write
 * capability. A real <button> (native focus + keyboard operability, R1
 * accessibility requirement) driven by pointerdown/up/leave AND Space/Enter
 * keydown/keyup. Releasing before HOLD_DURATION_MS resets to 0% and issues
 * NO callback — onConfirm fires exactly once, only when the full duration
 * elapses while still held (matches the interaction contract; the timer,
 * not the release, is authoritative for completion).
 */
export function HoldToConfirmButton({
  disabled,
  onConfirm,
}: {
  disabled: boolean
  onConfirm: () => void
}): React.ReactElement {
  const [holding, setHolding] = useState(false)
  const [progress, setProgress] = useState(0)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const startRef = useRef(0)

  function clearTimers(): void {
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
      timeoutRef.current = null
    }
  }

  function startHold(): void {
    if (disabled || holding) return
    setHolding(true)
    setProgress(0)
    startRef.current = Date.now()
    intervalRef.current = setInterval(() => {
      const elapsed = Date.now() - startRef.current
      setProgress(Math.min(100, (elapsed / HOLD_DURATION_MS) * 100))
    }, HOLD_TICK_MS)
    timeoutRef.current = setTimeout(() => {
      clearTimers()
      setProgress(100)
      setHolding(false)
      onConfirm()
    }, HOLD_DURATION_MS)
  }

  function releaseHold(): void {
    if (!holding) return
    clearTimers()
    setHolding(false)
    setProgress(0)
  }

  useEffect(() => clearTimers, [])

  return (
    <button
      type="button"
      className="hub-funnel-write-gate__hold-btn"
      disabled={disabled}
      aria-disabled={disabled}
      onPointerDown={(e) => {
        try { e.currentTarget.setPointerCapture(e.pointerId) } catch { /* jsdom fallback */ }
        startHold()
      }}
      onPointerUp={releaseHold}
      onPointerLeave={releaseHold}
      onKeyDown={(e) => {
        if (e.key === ' ' || e.key === 'Enter') {
          e.preventDefault()
          startHold()
        }
      }}
      onKeyUp={(e) => {
        if (e.key === ' ' || e.key === 'Enter') {
          releaseHold()
        }
      }}
    >
      <span className="hub-funnel-write-gate__hold-fill" style={{ width: `${progress}%` }} />
      <span className="hub-funnel-write-gate__hold-label">
        {holding ? 'Holding… keep pressing' : 'Hold 3s to confirm'}
      </span>
    </button>
  )
}

// CodeDisplay renders a join code with a small Copy affordance. Used for both
// the read code and the write code in the share panel. The code is displayed
// as plain text so the owner can read it aloud or transcribe it to the joiner.
export function CodeDisplay({
  label,
  code,
}: {
  label: string
  code: string
}): React.ReactElement {
  const [copied, setCopied] = useState(false)
  async function handleCopyCode(): Promise<void> {
    try {
      await ClipboardSetText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // clipboard failure — code remains visible for manual copy
    }
  }
  return (
    <div
      className="session-share-panel__code-display"
      style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '4px 0 8px', fontSize: 13 }}
    >
      <span style={{ color: '#9aa5ce' }}>{label}</span>
      <code
        style={{
          fontFamily: 'monospace',
          fontWeight: 700,
          letterSpacing: '0.05em',
          background: '#16161e',
          padding: '2px 8px',
          borderRadius: 3,
          userSelect: 'all',
          color: '#c0caf5',
        }}
        data-testid="join-code-text"
      >
        {code}
      </code>
      <button
        type="button"
        className="daemon-panel__btn"
        onClick={() => void handleCopyCode()}
        aria-label={`Copy ${label.toLowerCase()} to clipboard`}
        style={{ fontSize: 11, padding: '2px 8px' }}
      >
        {copied ? 'Copied!' : 'Copy'}
      </button>
    </div>
  )
}
