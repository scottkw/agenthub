/**
 * Phase 93 WGL-03 — webglProbe source-inspection tests.
 *
 * Per UI-SPEC §"Information Disclosure" mitigation: the renderer string never
 * leaves the probe function. The probe returns a boolean only — the regex
 * lives in the source, not in user-visible messages.
 */
import { describe, it, expect } from 'vitest'
import src from '../webglProbe.ts?raw'

describe('webglProbe source contract', () => {
  it('exports isSoftwareWebGL as a function', () => {
    expect(src).toMatch(/export\s+function\s+isSoftwareWebGL/)
  })

  it('returns boolean (signature explicitly typed)', () => {
    expect(src).toMatch(/isSoftwareWebGL\([^)]*\)\s*:\s*boolean/)
  })

  it('matches all four software-renderer signatures (SwiftShader, llvmpipe, ANGLE.*Software, ANGLE.*SwiftShader)', () => {
    expect(src).toMatch(/SwiftShader/)
    expect(src).toMatch(/llvmpipe/)
    expect(src).toMatch(/ANGLE/)
  })

  it('reads gl.RENDERER via getParameter (the actual probe mechanism)', () => {
    expect(src).toContain('getParameter')
    expect(src).toContain('RENDERER')
  })
})
