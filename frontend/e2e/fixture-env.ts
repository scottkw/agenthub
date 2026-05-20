// Phase 93 Plan 05 — fixture-env loader for Playwright specs.
// Reads .playwright/fixture-env.json (written by global-setup.ts) and
// returns the BASE_URL / CAP / VIEWER_CAP / SESSION_CWD / ADMIN_URL the
// test fixture exposed.
//
// Phase 120-05 extended: VIEWER_CAP (read-only, no files.read) and
// SESSION_CWD (absolute path to the seeded test tree) added so the
// files-browser.spec.ts suite can exercise the capability-denied path
// (UI-13) and correlate seeded fixtures with API responses.

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

export interface FixtureEnv {
  baseURL: string
  cap: string
  viewerCap: string
  sessionCwd: string
  adminURL: string
  pid: number
}

let cached: FixtureEnv | null = null

export function loadFixtureEnv(): FixtureEnv {
  if (cached) return cached
  const path = resolve(__dirname, '..', '.playwright', 'fixture-env.json')
  const data = readFileSync(path, 'utf8')
  cached = JSON.parse(data) as FixtureEnv
  return cached
}

export function sessionURL(env: FixtureEnv = loadFixtureEnv()): string {
  return `${env.baseURL}/sessions/playwright-test-session?cap=${encodeURIComponent(env.cap)}`
}

/**
 * filesApiURL builds an absolute URL to one of the /api/files/{list,stat,read}
 * routes, including session + cap query params. Used by the Phase 120-05
 * files-browser spec to exercise the real capability-gated API surface.
 *
 * @param env  Fixture env (auto-loaded if omitted)
 * @param op   One of 'list' | 'stat' | 'read'
 * @param path Relative path inside the seeded session cwd; omit / pass "." for the root
 * @param cap  Cap token to use; defaults to the owner cap (env.cap). Pass env.viewerCap for the denied path.
 */
export function filesApiURL(
  op: 'list' | 'stat' | 'read',
  path: string = '.',
  cap?: string,
  env: FixtureEnv = loadFixtureEnv(),
): string {
  const useCap = cap ?? env.cap
  const params = new URLSearchParams({
    session: 'playwright-test-session',
    path,
    cap: useCap,
  })
  return `${env.baseURL}/api/files/${op}?${params.toString()}`
}

/**
 * appUrl builds the /app/ entry URL with session + cap query params. Phase
 * 120-06 wired App.tsx to consult lib/webMode.detectMode() and, when the
 * pathname starts with `/app/`, skip the Wails RPC suite and source
 * session/cap from URL params + window.location.origin. The fixture binary
 * now embeds frontend/dist under -tags=playwrightfixture,wailsassets (see
 * cmd/playwright-fixture/assets_prod.go), so loading this URL in a browser
 * mounts a fully-functional file-browser tab — scenarios 13 + 14 in the
 * files-browser spec exercise that DOM path on all three browsers.
 */
export function appUrl(env: FixtureEnv = loadFixtureEnv()): string {
  return `${env.baseURL}/app/?session=playwright-test-session&cap=${encodeURIComponent(env.cap)}`
}

/** Same as appUrl but with the read-only viewer cap (no files.read perm). */
export function viewerAppUrl(env: FixtureEnv = loadFixtureEnv()): string {
  return `${env.baseURL}/app/?session=playwright-test-session&cap=${encodeURIComponent(env.viewerCap)}`
}
