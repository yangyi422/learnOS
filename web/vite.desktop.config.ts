import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Desktop build for the Wails v3 shell. It reuses the same Vue 3 sources as the
// web app but writes to desktop/frontend/dist so it can never clear or overwrite
// the Go-embedded web build in internal/webassets/dist.
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: '../desktop/frontend/dist',
    emptyOutDir: true,
    target: 'chrome120',
  },
})
