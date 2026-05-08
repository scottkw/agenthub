import type { IProgressState } from '@xterm/addon-progress'

/**
 * Phase 98 PRG-03 — STUB. Wave 1 replaces with the verbatim mean-bucket implementation.
 * Returns 0 (no progress active) for any input until the implementation lands.
 */
export function aggregateProgress(
  _registry: Map<string, IProgressState>
): 0 | 1 | 2 | 3 | 4 {
  return 0
}
