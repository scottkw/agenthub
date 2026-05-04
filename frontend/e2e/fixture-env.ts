// Phase 93 Plan 05 — fixture-env loader for Playwright specs.
// Reads .playwright/fixture-env.json (written by global-setup.ts) and
// returns the BASE_URL / CAP / ADMIN_URL the test fixture exposed.

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

export interface FixtureEnv {
  baseURL: string
  cap: string
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
