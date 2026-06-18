import { describe, it, expect, afterEach, vi } from 'vitest'
import React, { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { RemoteJoinCodeModal } from './RemoteJoinCodeModal'

// GAP-134-E: the join-code modal title must reflect the cap's intent. The Phase 122
// file-browse flow says "— Files"; the Phase 134 hub-modal flow opens the interactive/
// briefing terminal and must NOT be mislabelled "Files".

let container: HTMLDivElement | null = null
let root: Root | null = null

afterEach(() => {
  if (root) act(() => root!.unmount())
  container?.remove()
  container = null
  root = null
})

function renderModal(intent?: 'files' | 'hub-modal'): HTMLDivElement {
  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
  act(() => {
    root!.render(
      React.createElement(RemoteJoinCodeModal, {
        remoteSession: { id: 's1', name: 'claude 1', hostname: 'peer-host' },
        intent,
        onExchange: vi.fn().mockResolvedValue(undefined),
        onClose: vi.fn(),
      }),
    )
  })
  return container
}

describe('RemoteJoinCodeModal — title reflects intent (GAP-134-E)', () => {
  it('hub-modal intent shows the generic title (NOT "Files")', () => {
    const c = renderModal('hub-modal')
    const title = c.querySelector('.remote-join-modal__title')
    expect(title?.textContent?.trim()).toBe('Join Remote Session')
    expect(title?.textContent).not.toContain('Files')
  })

  it('files intent shows the "— Files" title', () => {
    const c = renderModal('files')
    const title = c.querySelector('.remote-join-modal__title')
    expect(title?.textContent?.trim()).toBe('Join Remote Session — Files')
  })

  it('defaults to the generic title when intent is omitted', () => {
    const c = renderModal(undefined)
    const title = c.querySelector('.remote-join-modal__title')
    expect(title?.textContent?.trim()).toBe('Join Remote Session')
  })
})
