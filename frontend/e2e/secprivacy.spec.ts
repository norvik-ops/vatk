import { test, expect } from './fixtures'

const FAKE_USER = { id: 'user-1', email: 'admin@example.com', display_name: 'Test Admin', roles: ['Admin'], role: 'Admin' }

async function authenticate(page: import('@playwright/test').Page) {
  await page.addInitScript((u) => { localStorage.setItem('vakt_user', JSON.stringify(u)) }, FAKE_USER)
  await page.route('**/api/v1/**', route =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [], pagination: { page: 1, limit: 25, total: 0, total_pages: 1 } }) })
  )
}

test.describe('SecPrivacy — DSR', () => {
  test.beforeEach(async ({ page }) => {
    await authenticate(page)
    await page.goto('/secprivacy/dsrs')
  })

  test('DSR list page renders', async ({ page }) => {
    await expect(page.getByText(/betroffenenanfragen|dsr/i)).toBeVisible()
  })

  test('can open create DSR dialog', async ({ page }) => {
    await page.getByRole('button', { name: /neue anfrage|erstellen|neu/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await expect(page.getByLabel(/name/i)).toBeVisible()
    await expect(page.getByLabel(/e-mail/i)).toBeVisible()
  })

  test('export button downloads CSV', async ({ page }) => {
    const downloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: /exportieren|export/i }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/dsr-export.*\.csv/)
  })
})
