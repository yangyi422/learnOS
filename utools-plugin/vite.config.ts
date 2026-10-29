import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { copyFileSync, existsSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const rootDir = dirname(fileURLToPath(import.meta.url))

// Files uTools loads next to index.html. They live at the plugin root (readable,
// un-bundled, per the uTools preload rules) and are copied into dist on build.
const RUNTIME_FILES = ['plugin.json', 'preload.js', 'logo.png']

export default defineConfig({
  base: './',
  plugins: [
    vue(),
    {
      name: 'lifeos-copy-runtime',
      apply: 'build',
      closeBundle() {
        const outDir = resolve(rootDir, 'dist')
        for (const file of RUNTIME_FILES) {
          const from = resolve(rootDir, file)
          if (!existsSync(from)) {
            console.warn(`[lifeos] missing runtime file: ${file}`)
            continue
          }
          copyFileSync(from, resolve(outDir, file))
        }
      },
    },
  ],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsDir: 'assets',
  },
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.test.ts'],
    restoreMocks: true,
  },
})
