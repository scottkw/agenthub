// Phase 120-02 Task 3 — humanSize formatter.
// Power-of-1024 (binary) units; locale-independent (no Intl.NumberFormat — keeps
// test output stable across CI hosts). Negative / NaN / Infinity collapse to '—'.

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

export function humanSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '—'
  if (bytes === 0) return '0 B'
  if (bytes < 1024) return `${Math.floor(bytes)} B`

  let value = bytes
  let unitIdx = 0
  while (value >= 1024 && unitIdx < UNITS.length - 1) {
    value /= 1024
    unitIdx++
  }
  // One decimal place; truncate (not round half-up) to avoid '5.0 MB' → '5.1 MB'
  // jitter on the 5-MiB cap boundary check used downstream.
  const truncated = Math.floor(value * 10) / 10
  return `${truncated.toFixed(1)} ${UNITS[unitIdx]}`
}
