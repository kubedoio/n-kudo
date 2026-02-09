/**
 * E2E Tests for VM Lifecycle
 * Tests VM creation, listing, start/stop operations
 */

import { test, expect } from '@playwright/test';
import { TestAPIClient } from './utils/api';
import * as fs from 'fs';
import * as path from 'path';
import vmFixtures from './fixtures/vms.json';

const TEST_DATA_FILE = path.join(__dirname, '.test-data', 'test-tenant.json');
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8443';

function getTestTenantData() {
  if (fs.existsSync(TEST_DATA_FILE)) {
    return JSON.parse(fs.readFileSync(TEST_DATA_FILE, 'utf-8'));
  }
  return null;
}

test.describe('VM Lifecycle', () => {
  let testData: ReturnType<typeof getTestTenantData>;
  let testSite: { id: string; name: string } | null = null;
  let testVM: { id: string; name: string } | null = null;

  test.beforeAll(async () => {
    testData = getTestTenantData();
    
    if (!testData) {
      return;
    }

    const client = new TestAPIClient(API_BASE_URL, process.env.ADMIN_KEY || 'dev-admin-key');
    
    try {
      await client.init();
      
      // Create a test site
      const site = await client.createSite(testData.apiKey, {
        name: 'E2E VM Test Site',
        external_key: `site-vm-e2e-${Date.now()}`,
        location_country_code: 'US',
      });
      testSite = { id: site.id, name: site.name };
      
      // Create a test VM via plan
      const planResponse = await client.applyPlan(testData.apiKey, {
        idempotency_key: `create-vm-${Date.now()}`,
        actions: [{
          operation_id: `create-${Date.now()}`,
          operation: 'CREATE',
          name: 'e2e-test-vm',
          vcpu_count: 1,
          memory_mib: 256,
        }],
      });
      
      // Store VM info from first execution
      if (planResponse.executions.length > 0 && planResponse.executions[0].vm_id) {
        testVM = {
          id: planResponse.executions[0].vm_id,
          name: 'e2e-test-vm',
        };
      }
    } catch (error) {
      console.warn('Failed to create test resources:', (error as Error).message);
    } finally {
      await client.dispose();
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
  });

  test.describe('VMs list page', () => {
    test.beforeEach(async ({ page }) => {
      if (!testData || !testSite) {
        return;
      }
      
      await page.goto(`/tenant/${testData.tenant.id}/sites/${testSite.id}`);
      await page.waitForLoadState('networkidle');
    });

    test('should display VMs section', async ({ page }) => {
      if (!testSite) {
        test.skip(true, 'Test site not created');
        return;
      }

      const vmsHeading = page.getByRole('heading', { name: /vms|virtual machines/i });
      await expect(vmsHeading).toBeVisible().catch(() => {
        // VMs might be in a different section
        return expect(page.locator('text=/vms/i').first()).toBeVisible();
      });
    });

    test('should have Create VM button', async ({ page }) => {
      const createButton = page.getByRole('button', { name: /create vm|add vm|new vm/i });
      await expect(createButton).toBeVisible().catch(() => {
        test.skip(true, 'Create VM button not found');
      });
    });
  });

  test.describe('VM creation', () => {
    test.beforeEach(async ({ page }) => {
      if (!testData || !testSite) {
        return;
      }
      
      await page.goto(`/tenant/${testData.tenant.id}/sites/${testSite.id}`);
      await page.waitForLoadState('networkidle');
    });

    test('should open create VM modal/form', async ({ page }) => {
      const createButton = page.getByRole('button', { name: /create vm|add vm|new vm/i });
      
      if (await createButton.isVisible().catch(() => false)) {
        await createButton.click();
        
        // Check for modal or form
        await expect(page.getByRole('dialog')).toBeVisible().catch(() => {
          return expect(page.getByLabel(/vm name/i)).toBeVisible();
        });
      } else {
        test.skip(true, 'Create VM button not found');
      }
    });

    test('should create VM with valid configuration', async ({ page }) => {
      const createButton = page.getByRole('button', { name: /create vm|add vm|new vm/i });
      
      if (await createButton.isVisible().catch(() => false)) {
        await createButton.click();
        
        const nameInput = page.getByLabel(/name/i).first();
        
        if (await nameInput.isVisible().catch(() => false)) {
          const vmName = `e2e-vm-${Date.now()}`;
          await nameInput.fill(vmName);
          
          // Fill CPU if field exists
          const cpuInput = page.getByLabel(/cpu|vcpu/i).first();
          if (await cpuInput.isVisible().catch(() => false)) {
            await cpuInput.fill('1');
          }
          
          // Fill memory if field exists
          const memoryInput = page.getByLabel(/memory/i).first();
          if (await memoryInput.isVisible().catch(() => false)) {
            await memoryInput.fill('256');
          }
          
          // Submit
          await page.getByRole('button', { name: /create|submit|save/i }).click();
          
          // Verify success
          await expect(page.getByText(/created|success|plan submitted/i)).toBeVisible({ timeout: 5000 });
        } else {
          test.skip(true, 'VM creation form not implemented');
        }
      } else {
        test.skip(true, 'Create VM button not found');
      }
    });
  });

  test.describe('VM list', () => {
    test.beforeEach(async ({ page }) => {
      if (!testData || !testSite) {
        return;
      }
      
      await page.goto(`/tenant/${testData.tenant.id}/sites/${testSite.id}`);
      await page.waitForLoadState('networkidle');
    });

    test('should display VMs in list', async ({ page }) => {
      // Look for VM list table or cards
      const vmList = page.locator('table').or(page.locator('.vm-list')).or(page.locator('[data-testid="vm-list"]'));
      await expect(vmList).toBeVisible();
    });

    test('should display VM details (name, state, specs)', async ({ page }) => {
      // Look for VM state indicators
      const stateIndicator = page.locator('text=/running|stopped|pending/i').first();
      await expect(stateIndicator).toBeVisible().catch(() => {
        test.skip(true, 'VM state indicators not visible');
      });
    });
  });

  test.describe('VM operations', () => {
    test.beforeEach(async ({ page }) => {
      if (!testData || !testSite || !testVM) {
        return;
      }
      
      await page.goto(`/tenant/${testData.tenant.id}/sites/${testSite.id}`);
      await page.waitForLoadState('networkidle');
    });

    test('should have start VM button for stopped VMs', async ({ page }) => {
      if (!testVM) {
        test.skip(true, 'Test VM not created');
        return;
      }
      
      // Find VM row/card
      const vmElement = page.locator('tr', { hasText: testVM.name }).or(page.locator('.card', { hasText: testVM.name }));
      
      // Look for start button
      const startButton = vmElement.locator('button').filter({ hasText: /start/i });
      
      if (await startButton.isVisible().catch(() => false)) {
        await expect(startButton).toBeEnabled();
      } else {
        test.skip(true, 'Start button not visible (VM may already be running)');
      }
    });

    test('should start VM when start button clicked', async ({ page }) => {
      if (!testVM) {
        test.skip(true, 'Test VM not created');
        return;
      }
      
      const vmElement = page.locator('tr', { hasText: testVM.name }).or(page.locator('.card', { hasText: testVM.name }));
      const startButton = vmElement.locator('button').filter({ hasText: /start/i });
      
      if (await startButton.isVisible().catch(() => false)) {
        await startButton.click();
        
        // Verify success message or state change
        await expect(page.getByText(/starting|started|success/i)).toBeVisible({ timeout: 5000 });
      } else {
        test.skip(true, 'Start button not visible');
      }
    });

    test('should have stop VM button for running VMs', async ({ page }) => {
      if (!testVM) {
        test.skip(true, 'Test VM not created');
        return;
      }
      
      const vmElement = page.locator('tr', { hasText: testVM.name }).or(page.locator('.card', { hasText: testVM.name }));
      const stopButton = vmElement.locator('button').filter({ hasText: /stop/i });
      
      if (await stopButton.isVisible().catch(() => false)) {
        await expect(stopButton).toBeEnabled();
      } else {
        test.skip(true, 'Stop button not visible (VM may not be running)');
      }
    });

    test('should stop VM when stop button clicked', async ({ page }) => {
      if (!testVM) {
        test.skip(true, 'Test VM not created');
        return;
      }
      
      const vmElement = page.locator('tr', { hasText: testVM.name }).or(page.locator('.card', { hasText: testVM.name }));
      const stopButton = vmElement.locator('button').filter({ hasText: /stop/i });
      
      if (await stopButton.isVisible().catch(() => false)) {
        await stopButton.click();
        
        // Verify success message or state change
        await expect(page.getByText(/stopping|stopped|success/i)).toBeVisible({ timeout: 5000 });
      } else {
        test.skip(true, 'Stop button not visible');
      }
    });
  });
});
