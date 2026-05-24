import { test as setup } from '@playwright/test'
import fs from 'fs'
import path from 'path'
import { AUTH_FILE } from '../playwright.config'

setup('demo-login', async ({ page, baseURL }) => {
  const base = baseURL ?? 'http://localhost:5173'

  const res = await fetch(`${base}/api/v1/demo/start`, { method: 'POST' })
  if (!res.ok) throw new Error(`demo/start schlug fehl: ${res.status}`)
  const { admin_email, admin_password } = await res.json() as {
    admin_email: string
    admin_password: string
  }

  await page.goto('/login')
  await page.locator('#email').fill(admin_email)
  await page.locator('#password').fill(admin_password)
  await page.getByRole('button', { name: /anmelden|sign in/i }).click()
  await page.waitForURL('/', { timeout: 15_000 })

  fs.mkdirSync(path.dirname(AUTH_FILE), { recursive: true })
  await page.context().storageState({ path: AUTH_FILE })
})
