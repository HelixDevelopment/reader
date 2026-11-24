/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Browsers cannot call the Helix server cross-origin (it sends no CORS headers),
// so dev and preview proxy API paths to HELIX_API_PROXY. Native shells (Capacitor,
// Electron) load the built bundle and set the base URL in Settings instead.
const target = process.env.HELIX_API_PROXY
const proxy = target
  ? Object.fromEntries(['/v1', '/healthz', '/readyz'].map((p) => [p, { target, changeOrigin: true }]))
  : undefined

export default defineConfig({
  base: './',
  plugins: [react()],
  server: { port: 3000, proxy },
  preview: { port: 4173, proxy },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
  },
})
