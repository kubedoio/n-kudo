import { test, expect } from '@playwright/test'

test.describe('VM Lifecycle', () => {
  test('user can navigate to sites page', async ({ page }) => {
    await page.goto('/admin/tenants')
    
    // Verify tenants page loaded
    await expect(page.locator('h1')).toContainText('Tenants')
  })
})
