/**
 * computeGuestScale — pure scale helper for the guest terminal viewer.
 *
 * VIEW-05: derives the CSS scale factor that fits the host grid (gridW × gridH
 * pixels) inside the guest container (containerW × containerH pixels).
 *
 * Rules:
 *  • s = min(containerW / gridW, containerH / gridH)
 *  • cap: s is never > 1.0 (downscale-only; never upscale a guest view)
 *  • guard: if gridW or gridH is ≤ 0, returns 1 (safe no-op; avoids divide-by-zero)
 */
export function computeGuestScale(
  containerW: number,
  containerH: number,
  gridW: number,
  gridH: number,
): number {
  if (gridW <= 0 || gridH <= 0) return 1
  const s = Math.min(containerW / gridW, containerH / gridH)
  return s > 1 ? 1 : s
}

/**
 * DEFAULT_GUEST_MIN_SCALE — readability floor for the guest terminal viewer
 * (BUG-01 / #128). At a 14px base font, ~0.7 scale renders ~10px effective
 * text — the floor below which the grid would keep shrinking to unreadable
 * sizes on a narrow phone viewport. Single source of truth shared by
 * TerminalPanel.recomputeScale and this file's tests.
 */
export const DEFAULT_GUEST_MIN_SCALE = 0.7

/**
 * computeGuestViewport — floor-aware guest scale helper (BUG-01 / #128).
 *
 * Extends computeGuestScale with a readability floor: below `minScale` the
 * guest viewer stops shrinking further and instead signals the caller to
 * enable horizontal scroll (overflowX: true), keeping text legible. The
 * never-upscale invariant (VIEW-04/05) is preserved — naturalCapped is
 * always ≤ 1 before the floor comparison.
 *
 * Rules:
 *  • natural = min(containerW / gridW, containerH / gridH)
 *  • naturalCapped = min(natural, 1)  — downscale-only, never upscale
 *  • naturalCapped >= minScale → { scale: naturalCapped, overflowX: false }
 *  • naturalCapped <  minScale → { scale: minScale, overflowX: true }
 *  • guard: gridW ≤ 0 or gridH ≤ 0 → { scale: 1, overflowX: false }
 */
export function computeGuestViewport(
  containerW: number,
  containerH: number,
  gridW: number,
  gridH: number,
  minScale: number = DEFAULT_GUEST_MIN_SCALE,
): { scale: number; overflowX: boolean } {
  if (gridW <= 0 || gridH <= 0) return { scale: 1, overflowX: false }
  const natural = Math.min(containerW / gridW, containerH / gridH)
  const naturalCapped = natural > 1 ? 1 : natural
  if (naturalCapped >= minScale) {
    return { scale: naturalCapped, overflowX: false }
  }
  return { scale: minScale, overflowX: true }
}
