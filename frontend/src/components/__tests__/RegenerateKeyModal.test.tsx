/**
 * RegenerateKeyModal — copy customization tests (Phase 150 gap-closure, SET-01).
 *
 * Regression context: Phase 150-02 reused RegenerateKeyModal as the
 * confirm-on-disable dialog for the shell web-share warning toggle, but the
 * component had hardcoded "Regenerate Signing Key?" / "Invalidate All Links"
 * copy. The disable-warning surface therefore rendered a dialog claiming it
 * would invalidate ALL shared links — a misleading, action-mismatched prompt
 * (found in live UAT). These tests pin both the default (regen-key) copy AND
 * the parameterized copy so the component can be safely reused.
 */
import React from 'react'
import { describe, it, expect, vi } from 'vitest'
import { render, act } from '@testing-library/react'
import { RegenerateKeyModal } from '../RegenerateKeyModal'

const noop = async (): Promise<void> => {}

describe('RegenerateKeyModal copy', () => {
  it('renders the default signing-key copy when no copy props are supplied (regression guard for the Security surface)', () => {
    const { container } = render(
      <RegenerateKeyModal isOpen onConfirm={noop} onCancel={() => {}} />
    )
    const text = container.textContent ?? ''
    expect(text).toContain('Regenerate Signing Key?')
    expect(text).toContain('invalidates ALL shared links')
    expect(text).toContain('Keep Links')
    expect(text).toContain('Invalidate All Links')
  })

  it('renders supplied copy props instead of the signing-key defaults', () => {
    const { container } = render(
      <RegenerateKeyModal
        isOpen
        onConfirm={noop}
        onCancel={() => {}}
        title="Disable shell web-share warning?"
        body="AgentHub won't show the one-time security reminder before web-sharing a shell session."
        confirmLabel="Disable warning"
        cancelLabel="Keep warning"
      />
    )
    const text = container.textContent ?? ''
    expect(text).toContain('Disable shell web-share warning?')
    expect(text).toContain('one-time security reminder')
    expect(text).toContain('Disable warning')
    expect(text).toContain('Keep warning')
    // The misleading signing-key copy must NOT leak into a non-signing-key use.
    expect(text).not.toContain('Regenerate Signing Key')
    expect(text).not.toContain('Invalidate All Links')
    expect(text).not.toContain('invalidates ALL shared links')
  })

  it('confirm button invokes onConfirm regardless of label', async () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined)
    const { container } = render(
      <RegenerateKeyModal
        isOpen
        onConfirm={onConfirm}
        onCancel={() => {}}
        confirmLabel="Disable warning"
      />
    )
    const buttons = Array.from(container.querySelectorAll<HTMLButtonElement>('button'))
    const confirmBtn = buttons.find((b) => b.textContent?.includes('Disable warning'))
    expect(confirmBtn).toBeTruthy()
    await act(async () => { confirmBtn!.click() })
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })
})
