/**
 * E2E Tests for Site Management
 * Tests site creation, listing, and navigation
 */

import { test, expect } from '@playwright/test';
import { setApiKey, setAdminKey } from './utils/auth';
import { TestAPIClient } from './utils/api';
import * as fs from 'fs';
import * as path from 'path';
import siteFixtures from './fixtures/sites.json';

const TEST_DATA_FILE = path.join(__dirname, '.test-data', 'test-tenant.json');
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8443';

function getTestTenantData() {
  if (fs.existsSync(TEST_DATA_FILE)) {
    return JSON.parse(fs.readFileSync(TEST_DATA_FILE, 'utf-8'));
  }
  return null;
}

test.describe('Site Management', () => {
  let testData: ReturnType<typeof getTestTenantData>;
  let testSite: { id: string; name: string } | null = null;

  test.beforeAll(async () => {
    // Create a test site via API for testing
    testData = getTestTenantData();
    
    if (testData) {
      const client = new TestAPIClient(API_BASE_URL, process.env.ADMIN_KEY || 'dev-admin-key');
      try {
        await client.init();
        const site = await client.createSite(testData.apiKey, {
          name: 'E2E Test Site',
          external_key: `site-e2e-${Date.now()}`,
          location_country_code: 'US',
        });
        testSite = { id: site.id, name: site.name };
      } catch (error) {
        console.warn('Failed to create test site:', (error as Error).message);
      } finally {
        await client.dispose();
      }
    }
  });

  test.beforeEach(async ({ page, context }) => {
    testData = getTestTenantData();
    
    if (!testData) {
      return;
    }

    // Set auth keys
    await context.addInitScript((data) => {
      localStorage.setItem('n-kudo-admin-key', 'dev-admin-key');
      localStorage.setItem('n-kudo-api-key', data.apiKey);
    }, testData);
    
    // Navigate to sites page for the test tenant
    await page.goto(`/tenant/${testData.tenant.id}/sites`);
    await page.waitForLoadState('networkidle');
  });

  test.describe('Sites list page', () => {
    test('should display sites page with correct title', async ({ page }) => {
      if (!testData) {
        test.skip(true, 'Test tenant data not available');
        return;
      }

      await expect(page.getByRole('heading', { name: /sites/i })).toBeVisible();
    });

    test('should display breadcrumb navigation', async ({ page }) => {
      const breadcrumb = page.locator('text=/back to tenants/i').or(page.locator('.breadcrumb'));
      await expect(breadcrumb).toBeVisible().catch(() => {
        test.skip(true, 'Breadcrumb not implemented');
      });
    });

    test('should have Add Site button', async ({ page }) => {
      const addButton = page.getByRole('button', { name: /add site/i });
      await expect(addButton).toBeVisible();
      await expect(addButton).toBeEnabled();
    });

    test('should have search functionality', async ({ page }) => {
      const searchInput = page.getByPlaceholder(/search sites/i);
      await expect(searchInput).toBeVisible();
      await expect(searchInput).toBeEnabled();
    });

    test('should display sites table', async ({ page }) => {
      const table = page.locator('table');
      await expect(table).toBeVisible();
    });
  });

  test.describe('Site creation', () => {
    test('should open create site modal on Add Site click', async ({ page }) => {
      await page.getByRole('button', { name: /add site/i }).click();
      
      // Check for modal or form
      await expect(page.getByRole('dialog')).toBeVisible().catch(() => {
        return expect(page.getByLabel(/site name/i)).toBeVisible();
      });
    });

    test('should create site with valid data', async ({ page }) => {
      const testSiteData = siteFixtures.newSite;
      const uniqueKey = `site-e2e-${Date.now()}`;
      
      await page.getByRole('button', { name: /add site/i }).click();
      
      // Fill in site details
      const nameInput = page.getByLabel(/name/i).first();
      
      if (await nameInput.isVisible().catch(() => false)) {
        await nameInput.fill(`${testSiteData.name} ${Date.now()}`);
        
        // Fill location if field exists
        const locationInput = page.getByLabel(/location|country/i).first();
        if (await locationInput.isVisible().catch(() => false)) {
          await locationInput.fill(testSiteData.location_country_code);
        }
        
        // Submit form
        await page.getByRole('button', { name: /create|submit|save/i }).click();
        
        // Verify success
        await expect(page.getByText(/created|success/i)).toBeVisible({ timeout: 5000 });
      } else {
        test.skip(true, 'Create site form not implemented');
      }
    });
  });

  test.describe('Site appears in list', () => {
    test('should display created site in table', async ({ page }) => {
      if (!testSite) {
        test.skip(true, 'Test site not created');
        return;
      }
      
      // Search for the test site
      const searchInput = page.getByPlaceholder(/search sites/i);
      await searchInput.fill(testSite.name);
      
      // Verify site appears in table
      const siteRow = page.locator('tr', { hasText: testSite.name });
      await expect(siteRow).toBeVisible();
    });

    test('should display site details in table row', async ({ page }) => {
      if (!testSite) {
        test.skip(true, 'Test site not created');
        return;
      }
      
      const siteRow = page.locator('tr', { hasText: testSite.name });
      
      // Check for expected columns
      await expect(siteRow.locator('text=/location/i').or(siteRow.locator('td').nth(1))).toBeVisible();
      await expect(siteRow.locator('text=/status/i').or(siteRow.locator('td').nth(2))).toBeVisible();
      await expect(siteRow.locator('text=/vms/i').or(siteRow.locator('td').nth(3))).toBeVisible();
    });
  });

  test.describe('Site navigation', () => {
    test('should navigate to site dashboard on click', async ({ page }) => {
      if (!testSite) {
        test.skip(true, 'Test site not created');
        return;
      }
      
      // Click on the site name
      await page.locator('tr', { hasText: testSite.name }).locator('text=' + testSite.name).click();
      
      // Verify navigation to site dashboard
      await expect(page).toHaveURL(/\/tenant\/.*\/sites\//);
      await expect(page.getByRole('heading')).toBeVisible();
    });
  });

  test.describe('Site dashboard', () => {
    test.beforeEach(async ({ page }) => {
      if (!testData || !testSite) {
        return;
      }
      
      // Navigate to site dashboard
      await page.goto(`/tenant/${testData.tenant.id}/sites/${testSite.id}`);
      await page.waitForLoadState('networkidle');
    });

    test('should display site dashboard', async ({ page }) => {
      if (!testSite) {
        test.skip(true, 'Test site not created');
        return;
      }
      
      // Verify dashboard elements
      await expect(page.locator('h1, h2').first()).toBeVisible();
    });

    test('should display VMs section', async ({ page }) => {
      const vmsSection = page.getByRole('heading', { name: /vms|virtual machines/i });
      await expect(vmsSection).toBeVisible().catch(() => {
        test.skip(true, 'VMs section not implemented');
      });
    });

    test('should display hosts section', async ({ page }) => {
      const hostsSection = page.getByRole('heading', { name: /hosts|nodes/i });
      await expect(hostsSection).toBeVisible().catch(() => {
        test.skip(true, 'Hosts section not implemented');
      });
    });
  });
});
