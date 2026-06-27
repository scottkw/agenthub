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
