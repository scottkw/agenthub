// Phase 120 — human-readable size formatter.
//
// NOTE (Wave 2 parallel execution): the canonical implementation ships in
// Plan 02 (Wave 1) with full unit tests. This file provides the minimal
// implementation Plan 03's FileRow consumes; the signature is byte-identical
// to Plan 02's spec so the merge is trivial.

/**
 * Format a byte count as a short human-readable string.
 * Uses power-of-1024 (binary) units with abbreviations B/KB/MB/GB/TB.
 * - 0 → "0 B"
 * - NaN, negative, Infinity → "—"
 */
export function humanSize(bytes: number): string {
  if (!Number.isFinite(bytes) || Number.isNaN(bytes) || bytes < 0) {
    return '—'
  }
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let value = bytes
  while (value >= 1024 && i < units.length - 1) {
    value = value / 1024
    i++
  }
  // Whole bytes: no decimal. Larger units: one decimal.
  if (i === 0) return `${Math.floor(value)} B`
  return `${value.toFixed(1)} ${units[i]}`
}
