import { describe, it, expect } from 'vitest'
import type { IProgressState } from '@xterm/addon-progress'
import { aggregateProgress } from '../aggregateProgress'

function set(value: number): IProgressState { return { state: 1, value } }
function remove(): IProgressState { return { state: 0, value: 0 } }

// Phase 98 PRG-03 — Wave 0 RED scaffold.
// All test cases fail at Wave 0 because the stub returns 0 always.
// Wave 1 (Plan 02 Task 1) replaces the stub body with the mean-bucket
// implementation and turns these GREEN.
describe('aggregateProgress', () => {
  it('returns 0 for an empty registry', () => {
    expect(aggregateProgress(new Map())).toBe(0)
  })
  it('returns 0 when every entry is state:0 (cleared)', () => {
    const m = new Map<string, IProgressState>([['a', remove()], ['b', remove()]])
    expect(aggregateProgress(m)).toBe(0)
  })
  it('returns 1 for a single state:1 at 5%', () => {
    const m = new Map<string, IProgressState>([['a', set(5)]])
    expect(aggregateProgress(m)).toBe(1)
  })
  it('returns 3 for mean(50, 75) = 62.5', () => {
    const m = new Map<string, IProgressState>([['a', set(50)], ['b', set(75)]])
    expect(aggregateProgress(m)).toBe(3)
  })
  it('returns 4 for value 100', () => {
    const m = new Map<string, IProgressState>([['a', set(100)]])
    expect(aggregateProgress(m)).toBe(4)
  })
  it('respects bucket boundary 25 → quartile 1', () => {
    const m = new Map<string, IProgressState>([['a', set(25)]])
    expect(aggregateProgress(m)).toBe(1)
  })
  it('respects bucket boundary 50 → quartile 2', () => {
    const m = new Map<string, IProgressState>([['a', set(50)]])
    expect(aggregateProgress(m)).toBe(2)
  })
  it('respects bucket boundary 75 → quartile 3', () => {
    const m = new Map<string, IProgressState>([['a', set(75)]])
    expect(aggregateProgress(m)).toBe(3)
  })
  it('ignores state:0 entries when computing the mean', () => {
    // Mean over [50] (state:1 only) — registry has one cleared entry which is excluded.
    const m = new Map<string, IProgressState>([['a', set(50)], ['b', remove()]])
    expect(aggregateProgress(m)).toBe(2)
  })
})
