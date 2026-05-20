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
import { writeFileSync, mkdirSync, existsSync, rmSync, cpSync, statSync, readdirSync } from 'node:fs'
import { dirname, resolve, join } from 'node:path'

export interface FixtureEnv {
  baseURL: string
  cap: string
  viewerCap: string
  sessionCwd: string
  adminURL: string
  pid: number
}

const FIXTURE_ENV_PATH = resolve(__dirname, '..', '.playwright', 'fixture-env.json')

let fixtureProc: ChildProcessWithoutNullStreams | null = null

export default async function globalSetup() {
  // Resolve repo root from this file's path: frontend/e2e/global-setup.ts → repo root is two levels up.
  const repoRoot = resolve(__dirname, '..', '..')

  // Phase 120-06 Task 3 — ensure the React bundle is built and copied into
  // cmd/playwright-fixture/dist/ so the fixture binary can embed it under
  // -tags wailsassets. `//go:embed` cannot escape its package, so we copy
  // frontend/dist into the fixture package rather than embedding across the
  // repo. Caching: skip the vite rebuild + dist-copy when frontend/dist/index.html
  // is newer than every frontend/src/** file.
  const frontendDir = resolve(repoRoot, 'frontend')
  const distDir = resolve(frontendDir, 'dist')
  const distIndex = resolve(distDir, 'index.html')

  function maxMtime(dir: string): number {
    let max = 0
    const stack: string[] = [dir]
    while (stack.length > 0) {
      const cur = stack.pop()!
      let entries: ReturnType<typeof readdirSync>
      try {
        entries = readdirSync(cur, { withFileTypes: true })
      } catch {
        continue
      }
      for (const ent of entries) {
        const p = join(cur, ent.name)
        if (ent.isDirectory()) {
          stack.push(p)
        } else {
          try {
            const m = statSync(p).mtimeMs
            if (m > max) max = m
          } catch {
            /* ignore unreadable file */
          }
        }
      }
    }
    return max
  }

  const srcMtime = maxMtime(resolve(frontendDir, 'src'))
  const distMtime = existsSync(distIndex) ? statSync(distIndex).mtimeMs : 0
  if (!existsSync(distIndex) || distMtime < srcMtime || process.env.PHASE_93_FIXTURE_REBUILD === '1') {
    const viteBuild = spawnSync('pnpm', ['exec', 'vite', 'build'], {
      cwd: frontendDir,
      stdio: 'inherit',
    })
    if (viteBuild.status !== 0) {
      throw new Error(
        `vite build failed (exit=${viteBuild.status}). Run \`cd frontend && pnpm build\` manually to diagnose.`,
      )
    }
  }
  if (!existsSync(distIndex)) {
    throw new Error(
      `frontend/dist/index.html missing after vite build. Investigate why pnpm build did not produce the SPA bundle.`,
    )
  }

  // Copy frontend/dist → cmd/playwright-fixture/dist for the embed (overwrites any stale copy).
  const fixturePkgDir = resolve(repoRoot, 'cmd', 'playwright-fixture')
  const fixtureDistDir = resolve(fixturePkgDir, 'dist')
  rmSync(fixtureDistDir, { recursive: true, force: true })
  cpSync(distDir, fixtureDistDir, { recursive: true })

  // Build the fixture binary so we can launch the resulting executable directly.
  // Spawning a built binary (rather than `go run`) gives us a clean process tree
  // where SIGTERM to the child PID actually reaps the server. Always rebuild
  // when we just refreshed the dist subtree — the embedded FS depends on it.
  const binPath = resolve(repoRoot, '.playwright', 'playwright-fixture')
  mkdirSync(dirname(binPath), { recursive: true })
  {
    const build = spawnSync(
      'go',
      ['build', '-tags=playwrightfixture,wailsassets', '-o', binPath, './cmd/playwright-fixture'],
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
    let viewerCap = ''
    let sessionCwd = ''
    let adminURL = ''

    fixtureProc!.stdout.on('data', (chunk: Buffer) => {
      stdoutBuf += chunk.toString()
      for (const line of stdoutBuf.split('\n')) {
        if (line.startsWith('BASE_URL=')) baseURL = line.slice('BASE_URL='.length).trim()
        // Match CAP= but NOT VIEWER_CAP= — exact prefix check.
        if (line.startsWith('CAP=')) cap = line.slice('CAP='.length).trim()
        if (line.startsWith('VIEWER_CAP=')) viewerCap = line.slice('VIEWER_CAP='.length).trim()
        if (line.startsWith('SESSION_CWD=')) sessionCwd = line.slice('SESSION_CWD='.length).trim()
        if (line.startsWith('ADMIN_URL=')) adminURL = line.slice('ADMIN_URL='.length).trim()
        if (line.startsWith('READY=1')) {
          clearTimeout(timer)
          resolveEnv({ baseURL, cap, viewerCap, sessionCwd, adminURL, pid: fixtureProc!.pid! })
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
