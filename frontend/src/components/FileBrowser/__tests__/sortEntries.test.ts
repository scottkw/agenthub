import { describe, it, expect } from 'vitest'
import { sortEntries, defaultSortDir } from '../sortEntries'
import type { FileEntry } from '../../../lib/filesApi'

// Helper — build a minimal FileEntry quickly.
function ent(
  name: string,
  opts: Partial<FileEntry> = {},
): FileEntry {
  return {
    name,
    size: 0,
    mtime: '',
    mode: 0,
    isDir: false,
    isSymlink: false,
    isBinary: false,
    ...opts,
  }
}

describe('sortEntries — directories-sticky rule', () => {
  it('keeps directories at the top on name asc', () => {
    const input = [
      ent('alpha.txt'),
      ent('zeta', { isDir: true }),
      ent('beta.txt'),
      ent('a-dir', { isDir: true }),
    ]
    const out = sortEntries(input, 'name', 'asc')
    expect(out.map((e) => e.name)).toEqual([
      'a-dir',
      'zeta',
      'alpha.txt',
      'beta.txt',
    ])
  })

  it('keeps directories at the top on name desc (dirs still on top)', () => {
    const input = [
      ent('alpha.txt'),
      ent('zeta', { isDir: true }),
      ent('beta.txt'),
      ent('a-dir', { isDir: true }),
    ]
    const out = sortEntries(input, 'name', 'desc')
    // Dirs stay on top; within dirs reversed (zeta then a-dir);
    // within files reversed (beta.txt then alpha.txt).
    expect(out.map((e) => e.name)).toEqual([
      'zeta',
      'a-dir',
      'beta.txt',
      'alpha.txt',
    ])
  })

  it('keeps directories at the top on size asc even when a dir would sort below files', () => {
    const input = [
      ent('big.bin', { size: 1_000_000 }),
      ent('emptyDir', { isDir: true, size: 0 }),
      ent('tiny.txt', { size: 10 }),
    ]
    const out = sortEntries(input, 'size', 'asc')
    expect(out[0].name).toBe('emptyDir') // dir always first
    expect(out.slice(1).map((e) => e.name)).toEqual(['tiny.txt', 'big.bin'])
  })
})

describe('sortEntries — sort key behavior', () => {
  it('sorts files by size descending when sortKey=size sortDir=desc', () => {
    const input = [
      ent('small', { size: 100 }),
      ent('big', { size: 9_999 }),
      ent('mid', { size: 500 }),
    ]
    const out = sortEntries(input, 'size', 'desc')
    expect(out.map((e) => e.name)).toEqual(['big', 'mid', 'small'])
  })

  it('sorts files by name ascending within group', () => {
    const input = [
      ent('charlie.txt'),
      ent('alpha.txt'),
      ent('bravo.txt'),
    ]
    const out = sortEntries(input, 'name', 'asc')
    expect(out.map((e) => e.name)).toEqual([
      'alpha.txt',
      'bravo.txt',
      'charlie.txt',
    ])
  })

  it('sinks mtime="" entries under asc and rises them under desc within their group', () => {
    const ascInput = [
      ent('has-mtime.txt', { mtime: '2026-05-20T10:00:00Z' }),
      ent('no-mtime.txt', { mtime: '' }),
      ent('older.txt', { mtime: '2026-05-19T10:00:00Z' }),
    ]
    const asc = sortEntries(ascInput, 'modified', 'asc')
    // ascending: empty mtime sinks to BOTTOM
    expect(asc[asc.length - 1].name).toBe('no-mtime.txt')

    const desc = sortEntries(ascInput, 'modified', 'desc')
    // descending: empty mtime rises to TOP of group
    expect(desc[0].name).toBe('no-mtime.txt')
  })

  it('case-insensitive name compare puts "Apple.txt" before "banana.txt"', () => {
    const input = [
      ent('banana.txt'),
      ent('Apple.txt'),
    ]
    const out = sortEntries(input, 'name', 'asc')
    expect(out.map((e) => e.name)).toEqual(['Apple.txt', 'banana.txt'])
  })
})

describe('sortEntries — purity contract', () => {
  it('returns a NEW array (input not mutated)', () => {
    const input = [
      ent('b.txt'),
      ent('a.txt'),
    ]
    const inputCopy = [...input]
    const out = sortEntries(input, 'name', 'asc')
    // input array reference is preserved (we did not return the same array)
    expect(Object.is(input, out)).toBe(false)
    // input is unchanged
    expect(input.map((e) => e.name)).toEqual(inputCopy.map((e) => e.name))
  })
})

describe('defaultSortDir', () => {
  it("returns 'asc' for 'name'", () => {
    expect(defaultSortDir('name')).toBe('asc')
  })
  it("returns 'desc' for 'size'", () => {
    expect(defaultSortDir('size')).toBe('desc')
  })
  it("returns 'desc' for 'modified'", () => {
    expect(defaultSortDir('modified')).toBe('desc')
  })
})
