// Phase 93 Plan 05 — Playwright config for the web parity / CSP / hot-swap
// e2e suite.
//
// The Go fixture (cmd/playwright-fixture, build-tag=playwrightfixture) is
// launched by ./e2e/global-setup.ts which also writes BASE_URL / CAP /
// ADMIN_URL to .playwright/fixture-env.json. Specs read those values via
// ./e2e/fixture-env.ts.
//
// We do NOT use Playwright's `webServer` field because that field expects a
// single URL; our fixture exposes two (HTTPS app + plain-HTTP admin) plus a
// capability token, which is cleaner to pass through a JSON file written by
// globalSetup.

import { defineConfig, devices } from '@playwright/test'
import { resolve } from 'node:path'

export default defineConfig({
  testDir: './e2e',
  testMatch: /.*\.spec\.ts$/,
  fullyParallel: false, // single fixture, sequential specs
  workers: 1,
  retries: 0,
  reporter: [['list']],
  globalSetup: resolve(__dirname, 'e2e', 'global-setup.ts'),
  globalTeardown: resolve(__dirname, 'e2e', 'global-teardown.ts'),
  use: {
    ignoreHTTPSErrors: true, // self-signed cert from the fixture
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
