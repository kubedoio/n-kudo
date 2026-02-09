# n-kudo Frontend E2E Tests

This directory contains end-to-end tests for the n-kudo frontend using [Playwright](https://playwright.dev/).

## Structure

```
e2e/
├── auth.setup.ts          # Global setup - creates test tenant and API key
├── auth.teardown.ts       # Global teardown - cleans up test data
├── fixtures/              # Test data and mock fixtures
│   ├── tenants.json       # Sample tenant data
│   ├── sites.json         # Sample site data
│   └── vms.json           # Sample VM data
├── utils/                 # Test utilities
│   ├── auth.ts            # Authentication helpers
│   └── api.ts             # Backend API client
├── tenants.spec.ts        # Tenant management tests
├── sites.spec.ts          # Site management tests
├── vms.spec.ts            # VM lifecycle tests
└── plans.spec.ts          # Plan execution tests
```

## Running Tests

### Prerequisites

1. Install dependencies:
   ```bash
   cd frontend
   npm install
   npx playwright install
   ```

2. Start the backend services:
   ```bash
   cd deployments
   docker compose up -d
   ```

### Run All Tests

```bash
npm run test:e2e
```

### Run Tests with UI Mode

```bash
npm run test:e2e:ui
```

### Run Specific Test File

```bash
npx playwright test tenants.spec.ts
```

### Run Tests in Specific Browser

```bash
npx playwright test --project=chromium
npx playwright test --project=firefox
npx playwright test --project=webkit
```

## Configuration

### Environment Variables

- `API_BASE_URL` - Backend API base URL (default: `http://localhost:8443`)
- `ADMIN_KEY` - Admin API key for tenant management (default: `dev-admin-key`)

### Test Data

Tests use a dedicated test tenant created during global setup (`auth.setup.ts`).
The tenant is automatically cleaned up after all tests complete (`auth.teardown.ts`).

## Writing Tests

### Basic Test Structure

```typescript
import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './utils/auth';

test('should display tenants list', async ({ page }) => {
  // Navigate and login
  await page.goto('/admin/tenants');
  await loginAsAdmin(page);
  
  // Assertions
  await expect(page.getByRole('heading', { name: /tenants/i })).toBeVisible();
});
```

### Using API Helpers

```typescript
import { TestAPIClient } from './utils/api';

const client = new TestAPIClient();
await client.init();

// Create test data via API
const tenant = await client.createTenant({
  slug: 'test-tenant',
  name: 'Test Tenant',
});

await client.dispose();
```

### Skipping Tests Conditionally

Tests that require specific UI elements skip gracefully if those elements don't exist:

```typescript
test('should create VM', async ({ page }) => {
  const createButton = page.getByRole('button', { name: /create vm/i });
  
  if (await createButton.isVisible().catch(() => false)) {
    // Test logic
  } else {
    test.skip(true, 'Create VM button not found');
  }
});
```

## CI/CD

E2E tests run automatically on:
- Push to `main` or `master` branches
- Pull requests targeting `main` or `master`
- Manual workflow dispatch

See `.github/workflows/e2e.yml` for the CI configuration.

## Troubleshooting

### Tests fail with connection errors

Ensure the backend is running:
```bash
cd deployments
docker compose up -d
```

### Tests fail with authentication errors

Check that the admin key is correctly set:
```bash
export ADMIN_KEY=your-admin-key
npm run test:e2e
```

### View test reports locally

```bash
npx playwright show-report
```

### Debug a specific test

```bash
npx playwright test tenants.spec.ts --debug
```
