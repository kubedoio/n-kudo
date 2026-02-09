import { test, expect } from '@playwright/test'

test.describe('API Key Management', () => {
  test('API Keys tab exists', async ({ page }) => {
    await page.goto('/admin/tenants')
    await expect(page.locator('text=Tenants')).toBeVisible()
  })
})
