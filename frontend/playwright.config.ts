import { defineConfig, devices } from '@playwright/test'
import path from 'path'

const demoURL = process.env.E2E_DEMO_URL

export const AUTH_FILE = path.join(import.meta.dirname, '.playwright/auth.json')

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    trace: 'on-first-retry',
  },

  projects: demoURL
    ? [
        // One-time setup: call demo/start, log in, save storageState
        {
          name: 'setup-demo',
          testMatch: '**/smoke-setup.ts',
          use: { baseURL: demoURL },
        },
        // Smoke tests: run against live demo with saved auth
        {
          name: 'chromium-demo',
          use: {
            ...devices['Desktop Chrome'],
            baseURL: demoURL,
            storageState: AUTH_FILE,
          },
          testMatch: '**/smoke.spec.ts',
          dependencies: ['setup-demo'],
        },
      ]
    : [
        // Mocked unit tests — run against local dev server
        { name: 'chromium', use: { ...devices['Desktop Chrome'] }, testIgnore: ['**/smoke*.ts'] },
        { name: 'firefox', use: { ...devices['Desktop Firefox'] }, testIgnore: ['**/smoke*.ts'] },
        { name: 'webkit', use: { ...devices['Desktop Safari'] }, testIgnore: ['**/smoke*.ts'] },
      ],

  // Only start Vite dev server for mocked tests; smoke tests hit the live demo
  webServer: demoURL
    ? undefined
    : {
        command: 'npm run dev',
        url: 'http://localhost:5173',
        reuseExistingServer: !process.env.CI,
        timeout: 30_000,
      },
})
