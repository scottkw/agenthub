import { describe, it, expect } from 'vitest'
import { computeGuestScale } from './terminalScale'

describe('computeGuestScale — VIEW-05 scale math', () => {
  // Container fits the grid (would-be scale > 1.0): cap to 1.0
  it('returns 1.0 when container is larger than grid (downscale-only cap)', () => {
    // container 800×600, grid 400×300 → s = min(2.0, 2.0) = 2.0 → capped to 1.0
    expect(computeGuestScale(800, 600, 400, 300)).toBe(1)
  })

  // Container is exactly half the grid size in both dimensions
  it('returns 0.5 when container is half the grid in both axes', () => {
    // container 400×300, grid 800×600 → s = min(0.5, 0.5) = 0.5
    expect(computeGuestScale(400, 300, 800, 600)).toBe(0.5)
  })

  // Width-bound: container height is more generous than width
  it('returns the width-bound scale when width is the tighter constraint', () => {
    // container 400×600, grid 800×300 → s = min(0.5, 2.0) = 0.5
    expect(computeGuestScale(400, 600, 800, 300)).toBe(0.5)
  })

  // Height-bound: container width is more generous than height
  it('returns the height-bound scale when height is the tighter constraint', () => {
    // container 600×400, grid 300×800 → s = min(2.0, 0.5) = 0.5
    expect(computeGuestScale(600, 400, 300, 800)).toBe(0.5)
  })

  // Zero gridW guard (avoid divide-by-zero)
  it('returns 1 when gridW is 0 (safe no-op)', () => {
    expect(computeGuestScale(800, 600, 0, 300)).toBe(1)
  })

  // Zero gridH guard
  it('returns 1 when gridH is 0 (safe no-op)', () => {
    expect(computeGuestScale(800, 600, 400, 0)).toBe(1)
  })

  // Both grid dims zero
  it('returns 1 when both grid dims are 0', () => {
    expect(computeGuestScale(800, 600, 0, 0)).toBe(1)
  })

  // Negative gridW guard (defensive: negative is treated as ≤ 0)
  it('returns 1 when gridW is negative', () => {
    expect(computeGuestScale(800, 600, -1, 300)).toBe(1)
  })

  // Exact fit (scale = 1.0 exactly)
  it('returns exactly 1.0 when container equals grid size', () => {
    expect(computeGuestScale(800, 600, 800, 600)).toBe(1)
  })

  // Non-integer scale
  it('returns a non-integer scale factor for non-even ratios', () => {
    // container 300×200, grid 400×300 → min(0.75, 0.666…) ≈ 0.666…
    const s = computeGuestScale(300, 200, 400, 300)
    expect(s).toBeCloseTo(2 / 3, 5)
    expect(s).toBeLessThanOrEqual(1)
  })
})
