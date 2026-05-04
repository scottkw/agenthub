// Phase 93 Plan 05 — Playwright global-teardown that kills the Go fixture
// process spawned by global-setup.ts.
//
// Reads PHASE_93_FIXTURE_PID from process.env (written during globalSetup) and
// sends SIGTERM. The fixture handles SIGTERM with a clean shutdown.

export default async function globalTeardown() {
  const pidStr = process.env.PHASE_93_FIXTURE_PID
  if (!pidStr) return
  const pid = Number(pidStr)
  if (!Number.isFinite(pid) || pid <= 0) return
  try {
    process.kill(pid, 'SIGTERM')
  } catch {
    // already gone
  }
}
