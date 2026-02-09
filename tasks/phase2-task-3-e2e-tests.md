# Phase 2 Task 3: E2E Tests with Playwright

## Task Description
Implement end-to-end tests using Playwright for critical user journeys.

## Prerequisites
- Phase 1 complete
- Frontend build working
- Playwright already in devDependencies

## Acceptance Criteria
- [ ] Playwright configured for the project
- [ ] E2E tests for critical paths
- [ ] Tests run with `npx playwright test`
- [ ] Screenshot on failure configured
- [ ] CI integration ready

## Critical User Paths to Test

### Path 1: Tenant Creation Flow
```
Login → Navigate to Tenants → Click Create → Fill Form → Submit → Verify Created
```

### Path 2: VM Lifecycle
```
Login → Select Tenant → Select Site → Create VM → Wait for Running → Stop VM → Delete VM
```

### Path 3: API Key Management
```
Login → Select Tenant → API Keys Tab → Create Key → Copy Key → Revoke Key
```

### Path 4: Enrollment Flow
```
Login → Select Tenant → Issue Token → Copy Token → (Simulate agent enrollment) → Verify Host Connected
```

## Playwright Configuration

Update `frontend/playwright.config.ts` (if exists) or create:
```typescript
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
})
```

## Test Files

### `frontend/e2e/tenant-creation.spec.ts`
```typescript
import { test, expect } from '@playwright/test'

test.describe('Tenant Creation', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate and login (or set auth token)
    await page.goto('/admin/tenants')
  })

  test('user can create a new tenant', async ({ page }) => {
    // Click create button
    await page.click('text=Create Tenant')
    
    // Fill form
    await page.fill('[name="name"]', 'E2E Test Tenant')
    await page.fill('[name="slug"]', 'e2e-test-tenant')
    await page.fill('[name="primary_region"]', 'us-east-1')
    
    // Submit
    await page.click('text=Create')
    
    // Verify success
    await expect(page.locator('text=E2E Test Tenant')).toBeVisible()
  })
})
```

### `frontend/e2e/vm-lifecycle.spec.ts`
```typescript
import { test, expect } from '@playwright/test'

test.describe('VM Lifecycle', () => {
  test('user can create, stop, and delete a VM', async ({ page }) => {
    // Navigate to site
    await page.goto('/tenant/test-tenant/sites/test-site')
    
    // Create VM
    await page.click('text=Create VM')
    await page.fill('[name="name"]', 'e2e-test-vm')
    await page.fill('[name="vcpu_count"]', '2')
    await page.fill('[name="memory_mib"]', '512')
    await page.click('text=Create')
    
    // Verify VM appears
    await expect(page.locator('text=e2e-test-vm')).toBeVisible()
    
    // Stop VM
    await page.click('[data-testid="stop-vm-e2e-test-vm"]')
    await page.click('text=Confirm')
    
    // Verify stopped
    await expect(page.locator('text=STOPPED')).toBeVisible()
    
    // Delete VM
    await page.click('[data-testid="delete-vm-e2e-test-vm"]')
    await page.click('text=Confirm')
    
    // Verify deleted
    await expect(page.locator('text=e2e-test-vm')).not.toBeVisible()
  })
})
```

### `frontend/e2e/api-keys.spec.ts`
```typescript
import { test, expect } from '@playwright/test'

test.describe('API Key Management', () => {
  test('user can create and revoke API key', async ({ page }) => {
    // Navigate to tenant detail - API Keys tab
    await page.goto('/admin/tenants/test-tenant?tab=apikeys')
    
    // Create API key
    await page.click('text=Create API Key')
    await page.fill('[name="name"]', 'e2e-test-key')
    await page.click('text=Create')
    
    // Verify key displayed (copy it)
    const keyText = await page.locator('[data-testid="api-key-value"]').textContent()
    expect(keyText).toMatch(/^nk_/)
    
    // Close modal
    await page.click('text=Close')
    
    // Verify key in list
    await expect(page.locator('text=e2e-test-key')).toBeVisible()
    
    // Revoke key
    await page.click('[data-testid="revoke-key-e2e-test-key"]')
    await page.click('text=Confirm')
    
    // Verify revoked
    await expect(page.locator('text=e2e-test-key')).not.toBeVisible()
  })
})
```

## Test Data Setup

Create `frontend/e2e/setup.ts`:
```typescript
import { test as base } from '@playwright/test'

export const test = base.extend({
  // Add custom fixtures
  authenticatedPage: async ({ page }, use) => {
    // Setup auth
    await page.goto('/')
    // Set admin key in localStorage or cookie
    await page.evaluate(() => {
      localStorage.setItem('admin-key', 'test-admin-key')
    })
    await use(page)
  }
})
```

## Test Data Attributes

Add `data-testid` attributes to key elements:
- `data-testid="create-tenant-btn"`
- `data-testid="tenant-name-input"`
- `data-testid="create-vm-btn"`
- `data-testid="stop-vm-{vmId}"`
- `data-testid="delete-vm-{vmId}"`
- `data-testid="api-key-value"`

## Running Tests

```bash
cd /srv/data01/kubedo/n-kudo/frontend

# Install Playwright browsers
npx playwright install

# Run tests
npx playwright test

# Run with UI mode
npx playwright test --ui

# Run specific test
npx playwright test tenant-creation

# Generate report
npx playwright show-report
```

## CI Integration

Add to `.github/workflows/test.yml`:
```yaml
- name: Run Playwright tests
  run: npx playwright test
  env:
    CI: true
```

## Definition of Done
- [ ] Playwright configured
- [ ] 4+ critical paths tested
- [ ] Screenshots on failure
- [ ] Tests run in headless mode
- [ ] Test data attributes added to UI
- [ ] Documentation for running tests

## Estimated Effort
6-8 hours
