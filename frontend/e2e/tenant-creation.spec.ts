import { test, expect } from '@playwright/test'

test.describe('Tenant Creation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin/tenants')
  })

  test('user can create a new tenant', async ({ page }) => {
    // Click create button
    await page.click('text=Create Tenant')
    
    // Fill form
    await page.fill('input[name="name"]', 'E2E Test Tenant')
    await page.fill('input[name="slug"]', 'e2e-test-tenant')
    
    // Submit
    await page.click('button[type="submit"]')
    
    // Verify success
    await expect(page.locator('text=E2E Test Tenant')).toBeVisible()
  })
})
