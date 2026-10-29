import { existsSync } from 'node:fs'
import { join } from 'node:path'
import { defineConfig } from '@playwright/test'

// An explicit URL validates an already-running frontend, never a replacement server.
const externalBaseURL = process.env.PLAYWRIGHT_BASE_URL?.trim()
const windowsChrome = join(process.env.PROGRAMFILES || 'C:/Program Files', 'Google/Chrome/Application/chrome.exe')
const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ||
  (process.platform === 'win32' && existsSync(windowsChrome) ? windowsChrome : undefined)

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: 'list',
  use: {
    baseURL: externalBaseURL || 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    launchOptions: { executablePath },
  },
  webServer: externalBaseURL ? undefined : {
    command: 'npm run dev -- --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
})
