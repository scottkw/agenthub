import type { IProgressState } from '@xterm/addon-progress'

/**
 * Phase 98 PRG-03 — cross-session progress aggregator.
 * Computes the mean of values from registry entries with state===1 (set),
 * then buckets the mean into one of 5 quartiles:
 *   [0,  0]   → 0  (no active progress; revert to base tray icon)
 *   (0,  25]  → 1  (25% glyph)
 *   (25, 50]  → 2  (50% glyph)
 *   (50, 75]  → 3  (75% glyph)
 *   (75, 100] → 4  (full glyph)
 *
 * Entries with state:0 (cleared), state:2 (error), state:3 (indeterminate),
 * or state:4 (paused) are EXCLUDED from the mean per Pitfall #2 — only
 * state:1 contributes. v3.2 ships state:1 + state:0 in the UI; state:2/3/4
 * are mapped to state:0 by the consumer (App.tsx) before reaching this
 * function. Locked Decision: mean (not median); see RESEARCH §"Locked Decisions".
 */
export function aggregateProgress(
  registry: Map<string, IProgressState>
): 0 | 1 | 2 | 3 | 4 {
  const values: number[] = []
  for (const s of registry.values()) {
    if (s.state === 1) values.push(s.value)
  }
  if (values.length === 0) return 0
  const mean = values.reduce((a, b) => a + b, 0) / values.length
  if (mean <= 0) return 0
  if (mean <= 25) return 1
  if (mean <= 50) return 2
  if (mean <= 75) return 3
  return 4
}
