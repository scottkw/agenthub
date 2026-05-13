import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
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
