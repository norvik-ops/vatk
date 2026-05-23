import { test, expect } from './fixtures'

const FAKE_USER = { id: 'user-1', email: 'admin@example.com', display_name: 'Test Admin', roles: ['Admin'], role: 'Admin' }

test.describe('Assets (SecPulse)', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript((u) => { localStorage.setItem('vakt_user', JSON.stringify(u)) }, FAKE_USER)
  })

  test('shows empty state when no assets exist', async ({ page }) => {
    await page.route('**/api/v1/secpulse/assets**', route =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { page: 1, limit: 25, total: 0, total_pages: 1 } }),
      })
    )
    await page.route('**/api/v1/**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    )

    await page.goto('/secpulse/assets')
    await expect(page.locator('[data-testid="empty-state"]').or(page.locator('text=Kein Asset').or(page.locator('text=No assets')))).toBeVisible({ timeout: 8000 })
  })

  test('opens create asset dialog', async ({ page }) => {
    await page.route('**/api/v1/secpulse/assets**', route =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { page: 1, limit: 25, total: 0, total_pages: 1 } }),
      })
    )
    await page.route('**/api/v1/**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    )

    await page.goto('/secpulse/assets')
    await page.click('button:has-text("Neu"), button:has-text("New"), button:has-text("Asset")')
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 3000 })
    await expect(page.locator('input[id="asset-name"]').or(page.locator('label:has-text("Name") + input, label:has-text("Name") ~ * input'))).toBeVisible()
  })
})
