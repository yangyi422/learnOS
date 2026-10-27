import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The Go API target is overridable so a local API can run on a port other than
// 8080 (on this workstation WSL2 forwards the deployed app onto 8080).
const apiTarget = process.env.VITE_API_TARGET || 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: '../internal/webassets/dist',
    emptyOutDir: true,
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    watch: {
      // Atomic file writes from editor/agent tooling leave short-lived temp dirs
      // inside the project; chokidar can hit EBUSY on them on Windows.
      ignored: ['**/.*.tmpdir/**', '**/.tmp/**', '**/test-results/**', '**/playwright-report/**'],
    },
    proxy: {
      '/api': apiTarget,
      '/healthz': apiTarget,
    },
  },
  preview: {
    host: '127.0.0.1',
    port: 4173,
  },
})
