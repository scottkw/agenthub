// Phase 93 Plan 05 — Playwright global-setup that boots the Go fixture binary,
// parses BASE_URL/CAP/ADMIN_URL from its stdout, and writes them to a temp
// JSON file the specs read via the e2e/fixture-env.ts loader.
//
// We launch the fixture ourselves (rather than via playwright.config.ts'
// webServer field) because Playwright's webServer plumbs a single URL only —
// it has no built-in path for capturing arbitrary KEY=VALUE lines from a
// fixture's stdout.
//
// The fixture writes lines like:
//   BASE_URL=https://127.0.0.1:PORT
//   CAP=<token>
//   ADMIN_URL=http://127.0.0.1:ADMIN_PORT
//   READY=1
// then continues running until SIGTERM.

import { spawn, spawnSync, type ChildProcessWithoutNullStreams } from 'node:child_process'
import { writeFileSync, mkdirSync, existsSync } from 'node:fs'
import { dirname, resolve } from 'node:path'

export interface FixtureEnv {
  baseURL: string
  cap: string
  adminURL: string
  pid: number
}

const FIXTURE_ENV_PATH = resolve(__dirname, '..', '.playwright', 'fixture-env.json')

let fixtureProc: ChildProcessWithoutNullStreams | null = null

export default async function globalSetup() {
  // Resolve repo root from this file's path: frontend/e2e/global-setup.ts → repo root is two levels up.
  const repoRoot = resolve(__dirname, '..', '..')

  // Build the fixture binary first so we can launch the resulting executable
  // directly. Spawning a built binary (rather than `go run`) gives us a clean
  // process tree where SIGTERM to the child PID actually reaps the server.
  const binPath = resolve(repoRoot, '.playwright', 'playwright-fixture')
  mkdirSync(dirname(binPath), { recursive: true })
  if (!existsSync(binPath) || process.env.PHASE_93_FIXTURE_REBUILD === '1') {
    const build = spawnSync(
      'go',
      ['build', '-tags=playwrightfixture', '-o', binPath, './cmd/playwright-fixture'],
      { cwd: repoRoot, stdio: 'inherit' }
    )
    if (build.status !== 0) {
      throw new Error(`go build playwright-fixture failed (exit=${build.status})`)
    }
  }

  fixtureProc = spawn(binPath, [], {
    cwd: repoRoot,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env },
  })

  const env = await new Promise<FixtureEnv>((resolveEnv, rejectEnv) => {
    let stdoutBuf = ''
    let stderrBuf = ''
    const timer = setTimeout(() => {
      rejectEnv(new Error(`fixture did not signal READY within 60s. stdout=${stdoutBuf} stderr=${stderrBuf}`))
    }, 60_000)

    let baseURL = ''
    let cap = ''
    let adminURL = ''

    fixtureProc!.stdout.on('data', (chunk: Buffer) => {
      stdoutBuf += chunk.toString()
      for (const line of stdoutBuf.split('\n')) {
        if (line.startsWith('BASE_URL=')) baseURL = line.slice('BASE_URL='.length).trim()
        if (line.startsWith('CAP=')) cap = line.slice('CAP='.length).trim()
        if (line.startsWith('ADMIN_URL=')) adminURL = line.slice('ADMIN_URL='.length).trim()
        if (line.startsWith('READY=1')) {
          clearTimeout(timer)
          resolveEnv({ baseURL, cap, adminURL, pid: fixtureProc!.pid! })
          return
        }
      }
    })
    fixtureProc!.stderr.on('data', (chunk: Buffer) => {
      stderrBuf += chunk.toString()
    })
    fixtureProc!.on('exit', (code, sig) => {
      clearTimeout(timer)
      rejectEnv(new Error(`fixture exited prematurely code=${code} sig=${sig}. stdout=${stdoutBuf} stderr=${stderrBuf}`))
    })
  })

  // Persist for specs.
  mkdirSync(dirname(FIXTURE_ENV_PATH), { recursive: true })
  writeFileSync(FIXTURE_ENV_PATH, JSON.stringify(env, null, 2))

  // Stash the pid in process.env so globalTeardown can kill it even after
  // module-scope state is cleared.
  process.env.PHASE_93_FIXTURE_PID = String(env.pid)
}
