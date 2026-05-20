import { describe, it, expect } from 'vitest'
import { humanSize } from '../humanSize'

// Phase 120-02 Task 3 — humanSize formatter unit tests.
// Power-of-1024 (binary) units; locale-independent; '—' for NaN/negative.
describe('humanSize', () => {
  it('returns "0 B" for 0', () => {
    expect(humanSize(0)).toBe('0 B')
  })
  it('returns "1 B" for 1', () => {
    expect(humanSize(1)).toBe('1 B')
  })
  it('returns "999 B" for 999', () => {
    expect(humanSize(999)).toBe('999 B')
  })
  it('returns "1.0 KB" for 1024', () => {
    expect(humanSize(1024)).toBe('1.0 KB')
  })
  it('returns "1.5 KB" for 1536', () => {
    expect(humanSize(1536)).toBe('1.5 KB')
  })
  it('returns "5.0 MB" for 5 MiB', () => {
    expect(humanSize(5 * 1024 * 1024)).toBe('5.0 MB')
  })
  it('returns "5.0 MB" for just over 5 MiB (rounds to 1 decimal)', () => {
    expect(humanSize(5 * 1024 * 1024 + 1)).toBe('5.0 MB')
  })
  it('returns "1.0 GB" for 1 GiB', () => {
    expect(humanSize(1024 * 1024 * 1024)).toBe('1.0 GB')
  })
  it('returns "—" for NaN', () => {
    expect(humanSize(NaN)).toBe('—')
  })
  it('returns "—" for negative numbers', () => {
    expect(humanSize(-1)).toBe('—')
  })
  it('returns "—" for Infinity', () => {
    expect(humanSize(Infinity)).toBe('—')
  })
  it('returns "1.0 TB" for 1 TiB', () => {
    expect(humanSize(1024 * 1024 * 1024 * 1024)).toBe('1.0 TB')
  })
})
