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
  // One decimal place, truncated (Math.floor) rather than rounded. Phase 120
  // WR-05: the previous comment claimed this guarded a downstream 5-MiB
  // boundary check, but the 5-MiB cap is enforced server-side in bytes (HTTP
  // 413), never compared against this formatted string — so that rationale
  // was incorrect. The truncation IS deliberate, but the real reason is
  // display stability: it gives a single canonical rendering per byte count
  // (no Math.round half-even ambiguity across JS engines / CI hosts) and
  // avoids ever upgrading a "5.0 MB" display to "5.1 MB" for a value that is
  // strictly less than 5 MiB, which would mislead users into thinking they
  // had crossed the cap. The cost is a consistent low bias of up to 0.1 unit
  // (e.g. 5.99 MB renders as "5.9 MB"); accept that as the trade for
  // deterministic UI snapshots.
  const truncated = Math.floor(value * 10) / 10
  return `${truncated.toFixed(1)} ${UNITS[unitIdx]}`
}
