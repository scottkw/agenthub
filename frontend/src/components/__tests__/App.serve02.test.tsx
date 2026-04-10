import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

// Source inspection tests for SERVE-02 webEnabled seeding in App.tsx.
// Verifies that init(), createTab(), and retryInit() all contain the
// webEnabled/sessionURLs seeding blocks restored in phase 61.

describe('SERVE-02: init() webEnabled seeding', () => {
  // Isolate the init function body for targeted assertions.
  const initBlock = raw.slice(
    raw.indexOf('async function init()'),
    raw.indexOf('void init()'),
  )

  it('init() contains s.webEnabled check inside sessions.forEach', () => {
    expect(initBlock).toContain('s.webEnabled')
  })

  it('init() calls GetWebServerURL inside seeding block', () => {
    expect(initBlock).toContain('GetWebServerURL()')
  })

  it('init() calls setWebEnabled with enabledMap', () => {
    expect(initBlock).toContain('setWebEnabled(enabledMap)')
  })

  it('init() calls setSessionURLs with urlMap', () => {
    expect(initBlock).toContain('setSessionURLs(urlMap)')
  })

  it('init() seeds webEnabled only when running is true', () => {
    // The seeding comment appears immediately before `if (running)`.
    // Verify the comment is present, then check that `if (running)` follows it.
    const seedingCommentPos = initBlock.indexOf('SERVE-02 restore')
    expect(seedingCommentPos).toBeGreaterThan(-1)
    const regionAfterComment = initBlock.slice(seedingCommentPos)
    expect(regionAfterComment).toContain('if (running)')
  })
})

describe('SERVE-02: createTab() webEnabled seeding', () => {
  // Isolate the createTab function body.
  const createTabBlock = raw.slice(
    raw.indexOf('const createTab = useCallback'),
    raw.indexOf('}, [tabCounter, webServerRunning])') + '}, [tabCounter, webServerRunning])'.length,
  )

  it('createTab() contains webServerRunning guard for seeding', () => {
    expect(createTabBlock).toContain('if (webServerRunning)')
  })

  it('createTab() sets webEnabled to true for new session', () => {
    expect(createTabBlock).toContain('setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))')
  })

  it('createTab() calls GetWebServerURL to seed sessionURL', () => {
    expect(createTabBlock).toContain('GetWebServerURL()')
  })

  it('createTab() calls setSessionURLs for new session URL', () => {
    expect(createTabBlock).toContain('setSessionURLs((prev) => ({ ...prev, [sessionId]:')
  })

  it('createTab() dependency array includes webServerRunning', () => {
    expect(createTabBlock).toContain('[tabCounter, webServerRunning]')
  })
})

describe('SERVE-02: retryInit() webEnabled seeding', () => {
  // Isolate the retryInit function body.
  // IMPORTANT: raw.indexOf('}, [])') finds the first occurrence (earlier callbacks),
  // so we must slice from retryInit's start position first, then find }, []) within that.
  const retryInitStart = raw.indexOf('const retryInit = useCallback')
  const retryInitSection = raw.slice(retryInitStart)
  const retryInitBlock = retryInitSection.slice(
    0,
    retryInitSection.indexOf('}, [])') + '}, [])'.length,
  )

  it('retryInit() contains s.webEnabled check inside sessions.forEach', () => {
    expect(retryInitBlock).toContain('s.webEnabled')
  })

  it('retryInit() calls GetWebServerURL inside seeding block', () => {
    expect(retryInitBlock).toContain('GetWebServerURL()')
  })

  it('retryInit() calls setWebEnabled with enabledMap', () => {
    expect(retryInitBlock).toContain('setWebEnabled(enabledMap)')
  })

  it('retryInit() calls setSessionURLs with urlMap', () => {
    expect(retryInitBlock).toContain('setSessionURLs(urlMap)')
  })

  it('retryInit() seeds webEnabled only when running is true', () => {
    // The seeding comment appears immediately before `if (running)`.
    const seedingCommentPos = retryInitBlock.indexOf('SERVE-02 restore')
    expect(seedingCommentPos).toBeGreaterThan(-1)
    const regionAfterComment = retryInitBlock.slice(seedingCommentPos)
    expect(regionAfterComment).toContain('if (running)')
  })
})

describe('SERVE-02: seeding blocks present in both paths (whole-file check)', () => {
  it('s.webEnabled appears at least twice (once in init, once in retryInit)', () => {
    const matches = raw.match(/s\.webEnabled/g)
    expect(matches).not.toBeNull()
    expect(matches!.length).toBeGreaterThanOrEqual(2)
  })

  it('GetWebServerURL is imported from wailsjs bindings', () => {
    expect(raw).toContain('GetWebServerURL')
    expect(raw).toContain("from './wailsjs/go/main/App'")
  })

  it('setSessionURLs is called in seeding blocks', () => {
    // At least three calls: init, retryInit, and the toggle handler
    const matches = raw.match(/setSessionURLs/g)
    expect(matches).not.toBeNull()
    expect(matches!.length).toBeGreaterThanOrEqual(3)
  })
})
