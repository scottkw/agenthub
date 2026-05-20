// Phase 120 — pure sort comparator for FileListPane.
//
// Contract (per 120-UI-SPEC §"Sort interaction" + 120-03-PLAN Task 1):
//   1. Directories ALWAYS sort to the top of the list, regardless of sortKey/
//      sortDir. (v3.4 has no toggle for this; it is a fixed UI contract.)
//   2. Within {directories} and within {files}, apply the active sort key.
//      - 'name'     — case-insensitive first pass; tie-break case-sensitive.
//      - 'size'     — numeric compare; tie-break by name asc.
//      - 'modified' — string compare on RFC3339 mtime; mtime=="" entries
//                     sink under 'asc' and rise under 'desc'; tie-break name asc.
//   3. sortDir === 'desc' reverses ordering within each group (but does NOT
//      move directories below files).
//   4. Returns a NEW array — input is never mutated.

import type { FileEntry } from '../../lib/filesApi'
import type { SortKey, SortDir } from '../../lib/filesTypes'

/**
 * Return the default sort direction for a freshly-clicked column header.
 * - name → 'asc' (alphabetical reads top-to-bottom)
 * - size → 'desc' (largest first feels more useful at-a-glance)
 * - modified → 'desc' (newest first)
 */
export function defaultSortDir(key: SortKey): SortDir {
  return key === 'name' ? 'asc' : 'desc'
}

function compareByName(a: FileEntry, b: FileEntry): number {
  // Case-insensitive first pass, locale-independent ordering.
  const an = a.name.toLowerCase()
  const bn = b.name.toLowerCase()
  if (an < bn) return -1
  if (an > bn) return 1
  // Tie-break case-sensitive so "FOO" and "foo" have a stable order.
  if (a.name < b.name) return -1
  if (a.name > b.name) return 1
  return 0
}

function compareBySize(a: FileEntry, b: FileEntry): number {
  if (a.size !== b.size) return a.size - b.size
  return compareByName(a, b)
}

function compareByMtime(a: FileEntry, b: FileEntry): number {
  const aEmpty = a.mtime === ''
  const bEmpty = b.mtime === ''
  // Empty mtime treated as "oldest" — sinks to bottom under asc, which means
  // it naturally rises to top of group when the entire group is reversed for
  // desc. So under the bare ascending comparator: empties go LAST.
  if (aEmpty && !bEmpty) return 1
  if (!aEmpty && bEmpty) return -1
  if (aEmpty && bEmpty) return compareByName(a, b)
  // RFC3339 strings sort correctly lexicographically.
  if (a.mtime < b.mtime) return -1
  if (a.mtime > b.mtime) return 1
  return compareByName(a, b)
}

export function sortEntries(
  entries: FileEntry[],
  sortKey: SortKey,
  sortDir: SortDir,
): FileEntry[] {
  // Partition into dirs vs files (directories-sticky always-on in v3.4).
  const dirs: FileEntry[] = []
  const files: FileEntry[] = []
  for (const e of entries) {
    ;(e.isDir ? dirs : files).push(e)
  }

  const comparator = (a: FileEntry, b: FileEntry): number => {
    switch (sortKey) {
      case 'name':
        return compareByName(a, b)
      case 'size':
        return compareBySize(a, b)
      case 'modified':
        return compareByMtime(a, b)
    }
  }

  dirs.sort(comparator)
  files.sort(comparator)

  if (sortDir === 'desc') {
    // Reverse the within-group ordering. For modified, the comparator places
    // empty-mtime entries last under asc, so reversal naturally rises them
    // to the top of their group as required by the UI spec.
    dirs.reverse()
    files.reverse()
  }

  return [...dirs, ...files]
}
