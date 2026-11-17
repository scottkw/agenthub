// Phase 173 UAT harness — isolated live mount of the REAL Share-modal frame DOM
// + real ShareSegmentedControl + real InternetFullAccessTab (armed, tallest state)
// against the real style.css. Lets dev-browser verify the 3 human-verification
// items (SM-02 scroll confinement, SM-07 colorblind-safe ring, SM-07
// prefers-reduced-motion hold fallback) that jsdom cannot. Temporary; safe to delete.
//
// scenario is chosen via ?scenario=scroll|ring|reducedmotion (default scroll).
// For reducedmotion we override window.matchMedia to force reduce=true BEFORE the
// component reads it at render — this is exactly the lever the OS setting pulls
// from HoldToConfirmButton's perspective (it branches on matchMedia(...).matches).
import React from 'react'
import { createRoot } from 'react-dom/client'
import './src/style.css'
import { ShareSegmentedControl, type ShareTab } from './src/components/Hub/ShareSegmentedControl'
import { InternetFullAccessTab } from './src/components/SessionShare/InternetFullAccessTab'

const params = new URLSearchParams(window.location.search)
const scenario = params.get('scenario') ?? 'scroll'

if (scenario === 'reducedmotion') {
  // Force prefers-reduced-motion: reduce for the component's JS branch.
  const realMatchMedia = window.matchMedia.bind(window)
  window.matchMedia = (q: string): MediaQueryList => {
    if (q.includes('prefers-reduced-motion')) {
      return {
        matches: q.includes('reduce'),
        media: q,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      } as unknown as MediaQueryList
    }
    return realMatchMedia(q)
  }
}

// Faithful copy of the modal's control-strip toggle rows (SessionShareModal.tsx:616-733).
function ControlStrip(): React.ReactElement {
  return (
    <>
      <label className="settings-panel__toggle-row settings-panel__toggle-row--checked">
        <input type="checkbox" className="settings-panel__toggle-input" role="switch" aria-checked readOnly checked aria-label="Share the session" />
        <span className="settings-panel__toggle-track"><span className="settings-panel__toggle-thumb" /></span>
        <span className="settings-panel__toggle-label">Share the session</span>
        <span className="settings-panel__toggle-state">On</span>
      </label>
      <div className="hub-share-modal__toggle-wrap" aria-disabled={false}>
        <label className="settings-panel__toggle-row">
          <input type="checkbox" className="settings-panel__toggle-input" role="switch" aria-checked={false} readOnly aria-label="Enable remote file browsing" />
          <span className="settings-panel__toggle-track"><span className="settings-panel__toggle-thumb" /></span>
          <span className="settings-panel__toggle-label">Enable remote file browsing</span>
          <span className="settings-panel__toggle-state">Off</span>
        </label>
      </div>
      <div className="hub-funnel-toggle-section" aria-disabled={false}>
        <label className="settings-panel__toggle-row settings-panel__toggle-row--checked">
          <input type="checkbox" className="settings-panel__toggle-input" role="switch" aria-checked readOnly checked aria-label="Enable internet sharing" />
          <span className="settings-panel__toggle-track"><span className="settings-panel__toggle-thumb" /></span>
          <span className="settings-panel__toggle-label">Enable internet sharing</span>
          <span className="settings-panel__toggle-state">On</span>
        </label>
      </div>
    </>
  )
}

// Renders the REAL modal frame DOM: overlay > dialog(.hub-share-modal) > header
// + .hub-share-modal__body { control strip + rule + segbar + .hub-share-modal__tabpanel }.
// The Full-access tab is active (danger ring under test) and armed (tallest content).
function Harness(): React.ReactElement {
  const [tab, setTab] = React.useState<ShareTab>('internet-fa')
  // A future expiry so the countdown + full armed result renders.
  const expiresAt = Math.floor(Date.now() / 1000) + 1800
  return (
    <div className="hub-modal-overlay hub-modal-overlay--open" style={{ position: 'static' }}>
      <div role="dialog" aria-modal="true" aria-label="Share demo-session" className="hub-share-modal hub-share-modal--open">
        <div className="hub-share-modal__header">
          <span className="hub-share-modal__title">Share — demo-session</span>
          <button type="button" className="hub-modal__close" aria-label="Close share modal">✕</button>
        </div>
        <div className="hub-share-modal__body">
          <ControlStrip />
          <hr className="hub-share-modal__rule" />
          <ShareSegmentedControl
            active={tab}
            onSelect={setTab}
            tabs={[
              { id: 'tailnet', main: 'Tailnet', sub: 'Private' },
              { id: 'internet-ro', main: 'Internet', sub: 'Read-only' },
              { id: 'internet-fa', main: 'Internet', sub: 'Full access', danger: true },
            ]}
          />
          <div className="hub-share-modal__tabpanel">
            <InternetFullAccessTab
              funnelActive={true}
              warmingUp={false}
              writeGateUrl="https://demo-session.tail1234.ts.net/app/write"
              writeGateCode="ABC-123-XYZ"
              writeGateExpiresAt={expiresAt}
              writeGateUsed={false}
            />
          </div>
        </div>
      </div>
    </div>
  )
}

// The reducedmotion scenario needs the gate OPEN (not armed) so the HoldToConfirmButton
// is on screen. Render the tab alone in gate-open state instead of the armed frame.
function ReducedMotionHarness(): React.ReactElement {
  return (
    <div className="hub-share-modal hub-share-modal--open" style={{ position: 'static', maxHeight: 'none' }}>
      <div className="hub-share-modal__body">
        <div className="hub-share-modal__tabpanel">
          <InternetFullAccessTab funnelActive={true} warmingUp={false} />
        </div>
      </div>
    </div>
  )
}

// 'reducedmotion' (matchMedia forced reduce) and 'motion' (baseline, no override)
// both render the idle Full-access tab so the HoldToConfirmButton branch is reachable.
const idleTabScenario = scenario === 'reducedmotion' || scenario === 'motion'
const root = createRoot(document.getElementById('root')!)
root.render(idleTabScenario ? <ReducedMotionHarness /> : <Harness />)
