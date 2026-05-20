import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  // Phase 120-06 Task 3 — relative base so the same built bundle works under
  // BOTH mount points: Wails desktop (served from `/`) and the webserver's
  // /app/ route (served to web-share viewers). Vite's default `/` base
  // emits absolute paths like `/assets/index-XYZ.css` which collide with
  // the webserver's existing `/assets/` mount (legacy xterm/webfs assets).
  // Relative paths resolve correctly under either mount point.
  base: './',
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    exclude: ['node_modules/**', 'dist/**', 'e2e/**', '**/*.spec.ts'],
  },
  resolve: {
    alias: {
      // Wails generates this directory at build time; stub it for tests
      './wailsjs/wailsjs/runtime/runtime': path.resolve(__dirname, 'src/wailsjs/runtime/runtime.js'),
      '../wailsjs/wailsjs/runtime/runtime': path.resolve(__dirname, 'src/wailsjs/runtime/runtime.js'),
      '../../wailsjs/wailsjs/runtime/runtime': path.resolve(__dirname, 'src/wailsjs/runtime/runtime.js'),
    },
  },
})
