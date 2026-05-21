/**
 * @vitest-environment jsdom
 *
 * Phase 122-03 Task 3 — FileBrowserTab 401-on-remote → EnableWebSharingTakeover.
 *
 * The full FileBrowserTab DOM interaction is heavy (capability probe, list
 * fetch, preview fetch — all useEffects with abort controllers). Existing
 * FileBrowserTab tests (.no-base64, .no-rehype-raw, .singleton) follow a
 * source-inspection pattern; we do the same here so the contract change is
 * pinned without re-implementing a render harness.
 *
 * Behaviour pinned:
 *   - FileBrowserTabProps.pathPrefix?: string is declared
 *   - FileBrowserTabProps.onReenterJoinCode?: () => void is declared
 *   - pathPrefix is forwarded to FilesApiClient constructor
 *   - The error switch routes 401 + isRemote → EnableWebSharingTakeover
 *   - 403 still routes to PermissionDeniedTakeover (no regression)
 *   - 404 still routes to EmptyDirectoryState OR not-found error (no regression)
 *   - The 'enable-web-sharing' ListError discriminator is added to the union
 */

import { describe, it, expect } from 'vitest'
import raw from '../../FileBrowserTab.tsx?raw'

describe('FileBrowserTab — Phase 122-03 props extension', () => {
  it('declares pathPrefix?: string on FileBrowserTabProps', () => {
    expect(raw).toMatch(/pathPrefix\?:\s*string/)
  })

  it('declares onReenterJoinCode?: () => void on FileBrowserTabProps', () => {
    expect(raw).toMatch(/onReenterJoinCode\?:\s*\(\)\s*=>\s*void/)
  })

  it('forwards pathPrefix to the FilesApiClient constructor', () => {
    // The client constructor must receive pathPrefix in its config.
    expect(raw).toMatch(/new FilesApiClient\(\s*\{[^}]*pathPrefix/)
  })
})

describe('FileBrowserTab — Phase 122-03 401-isRemote handler', () => {
  it('imports EnableWebSharingTakeover', () => {
    expect(raw).toMatch(
      /import\s*\{\s*EnableWebSharingTakeover\s*\}\s*from\s*['"][^'"]*EnableWebSharingTakeover['"]/,
    )
  })

  it("adds an 'enable-web-sharing' ListError discriminator to the union", () => {
    // The ListError type union must include the new discriminator
    expect(raw).toMatch(/'enable-web-sharing'/)
  })

  it('routes 401 + isRemote to the enable-web-sharing path', () => {
    // The pattern: when isUnauthorized() AND isRemote (+ onReenterJoinCode), set
    // listError to 'enable-web-sharing'. The exact line shape can vary; we
    // verify the conjunction is present in the error handling.
    expect(raw).toMatch(/isUnauthorized\(\)/)
    // The remote branch references isRemote in the unauthorized handler
    const unauthIdx = raw.indexOf('isUnauthorized()')
    expect(unauthIdx).toBeGreaterThan(-1)
    const slice = raw.slice(unauthIdx, unauthIdx + 800)
    expect(slice).toMatch(/isRemote/)
    expect(slice).toMatch(/enable-web-sharing/)
  })

  it('renders EnableWebSharingTakeover when listError === enable-web-sharing', () => {
    expect(raw).toMatch(/listError\s*===\s*['"]enable-web-sharing['"]/)
    expect(raw).toContain('<EnableWebSharingTakeover')
  })
})

describe('FileBrowserTab — Phase 122-03 no regression', () => {
  it('403 (forbidden) still has a PermissionDeniedTakeover-equivalent branch', () => {
    // The 403/files.read takeover continues to exist via capState='denied'
    // (handled at the capability probe level — unchanged from Phase 120-04).
    expect(raw).toContain('PermissionDeniedTakeover')
  })

  it('404 (not-found) still has its existing not-found branch', () => {
    expect(raw).toMatch(/isNotFound\(\)/)
    expect(raw).toMatch(/'not-found'/)
  })
})
