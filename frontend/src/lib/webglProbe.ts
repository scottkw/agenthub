/**
 * isSoftwareWebGL — Phase 93 WGL-03.
 *
 * Returns true if the browser's WebGL implementation is software-rasterized
 * (SwiftShader, llvmpipe, ANGLE-software, ANGLE-SwiftShader). Used at
 * TerminalPanel mount to preempt WebGL on devices where the DOM renderer
 * will out-perform the GPU path (notably iPad Safari, GPU-blacklisted
 * corporate browsers, and Linux software-rasterizer fallback).
 *
 * Returns false on any error path (no WebGL, blocked context, exception)
 * — the caller falls through to the existing try/catch around WebglAddon
 * construction and the regular context-loss handler.
 *
 * Information-disclosure mitigation (T-93-WGL-03): the renderer string never
 * leaves this function. We return a boolean only; user-visible messages
 * never include "SwiftShader", "llvmpipe", "ANGLE", or any internal-detection
 * vocabulary.
 */
export function isSoftwareWebGL(): boolean {
  try {
    const canvas = document.createElement('canvas')
    const gl = (canvas.getContext('webgl') ||
      canvas.getContext('experimental-webgl')) as WebGLRenderingContext | null
    if (!gl) return false
    const renderer = gl.getParameter(gl.RENDERER) as string | null
    if (!renderer) return false
    return /SwiftShader|llvmpipe|ANGLE.*Software|ANGLE.*SwiftShader/i.test(renderer)
  } catch {
    return false
  }
}
