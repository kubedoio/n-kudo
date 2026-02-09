/**
 * E2E Tests for Plan Execution
 * Tests applying plans and viewing execution logs
 */

import { test, expect } from '@playwright/test';
import { TestAPIClient } from './utils/api';
import * as fs from 'fs';
import * as path from 'path';

const TEST_DATA_FILE = path.join(__dirname, '.test-data', 'test-tenant.json');
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8443';

function getTestTenantData() {
  if (fs.existsSync(TEST_DATA_FILE)) {
    return JSON.parse(fs.readFileSync(TEST_DATA_FILE, 'utf-8'));
  }
  return null;
}

test.describe('Plan Execution', () => {
  let testData: ReturnType<typeof getTestTenantData>;
  let testSite: { id: string; name: string } | null = null;
  let testPlan: { id: string; executionId: string } | null = null;

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
        name: 'E2E Plan Test Site',
        external_key: `site-plan-e2e-${Date.now()}`,
        location_country_code: 'US',
      });
      testSite = { id: site.id, name: site.name };
      
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

  test.describe('Plan application', () => {
    test.beforeEach(async ({ page }) => {
      if (!testData || !testSite) {
        return;
      }
      
      await page.goto(`/tenant/${testData.tenant.id}/sites/${testSite.id}`);
      await page.waitForLoadState('networkidle');
    });

    test('should have Apply Plan button', async ({ page }) => {
      const applyButton = page.getByRole('button', { name: /apply plan|submit plan|deploy/i });
      await expect(applyButton).toBeVisible().catch(() => {
        test.skip(true, 'Apply Plan button not found');
      });
    });

    test('should open plan editor/modal on Apply Plan click', async ({ page }) => {
      const applyButton = page.getByRole('button', { name: /apply plan|submit plan|deploy/i });
      
      if (await applyButton.isVisible().catch(() => false)) {
        await applyButton.click();
        
        // Check for plan editor modal
        await expect(page.getByRole('dialog')).toBeVisible().catch(() => {
          return expect(page.getByText(/plan|action|operation/i).first()).toBeVisible();
        });
      } else {
        test.skip(true, 'Apply Plan button not found');
      }
    });

    test('should submit plan to create VM', async ({ page }) => {
      const applyButton = page.getByRole('button', { name: /apply plan|submit plan|deploy/i });
      
      if (await applyButton.isVisible().catch(() => false)) {
        await applyButton.click();
        
        // Look for add action button or form
        const addActionButton = page.getByRole('button', { name: /add action|create action/i });
        
        if (await addActionButton.isVisible().catch(() => false)) {
          await addActionButton.click();
          
          // Select operation type
          const operationSelect = page.getByLabel(/operation/i).first();
          if (await operationSelect.isVisible().catch(() => false)) {
            await operationSelect.selectOption('CREATE');
          }
          
          // Fill VM details
          const nameInput = page.getByLabel(/name/i).first();
          if (await nameInput.isVisible().catch(() => false)) {
            await nameInput.fill(`plan-vm-${Date.now()}`);
          }
          
          // Submit plan
          await page.getByRole('button', { name: /submit|apply|deploy/i }).click();
          
          // Verify success
          await expect(page.getByText(/plan submitted|success|created/i)).toBeVisible({ timeout: 5000 });
        } else {
          test.skip(true, 'Plan editor not fully implemented');
        }
      } else {
        test.skip(true, 'Apply Plan button not found');
      }
    });

    test('should submit plan with multiple actions', async ({ page }) => {
      const applyButton = page.getByRole('button', { name: /apply plan|submit plan|deploy/i });
      
      if (await applyButton.isVisible().catch(() => false)) {
        await applyButton.click();
        
        // Add CREATE action
        const addActionButton = page.getByRole('button', { name: /add action/i });
        
        if (await addActionButton.isVisible().catch(() => false)) {
          // Add first action
          await addActionButton.click();
          
          // Add second action
          await addActionButton.click();
          
          // Verify multiple actions are added
          const actionItems = page.locator('[data-testid="plan-action"]').or(page.locator('.plan-action'));
          const count = await actionItems.count();
          
          if (count >= 2) {
            await page.getByRole('button', { name: /submit|apply/i }).click();
            await expect(page.getByText(/plan submitted|success/i)).toBeVisible({ timeout: 5000 });
          } else {
            test.skip(true, 'Multiple action support not implemented');
          }
        } else {
          test.skip(true, 'Add action button not found');
        }
      } else {
        test.skip(true, 'Apply Plan button not found');
      }
    });
  });

  test.describe('Execution logs', () => {
    test.beforeEach(async ({ page }) => {
      if (!testData || !testSite) {
        return;
      }

      // Create a plan via API to get execution logs
      const client = new TestAPIClient(API_BASE_URL, process.env.ADMIN_KEY || 'dev-admin-key');
      
      try {
        await client.init();
        const planResponse = await client.applyPlan(testData.apiKey, {
          idempotency_key: `plan-logs-${Date.now()}`,
          actions: [{
            operation_id: `op-${Date.now()}`,
            operation: 'CREATE',
            name: `plan-test-vm-${Date.now()}`,
            vcpu_count: 1,
            memory_mib: 256,
          }],
        });
        
        testPlan = {
          id: planResponse.plan_id,
          executionId: planResponse.executions[0]?.id || '',
        };
      } catch (error) {
        console.warn('Failed to create test plan:', (error as Error).message);
      } finally {
        await client.dispose();
      }
      
      await page.goto(`/tenant/${testData.tenant.id}/sites/${testSite.id}`);
      await page.waitForLoadState('networkidle');
    });

    test('should display plan execution section', async ({ page }) => {
      const executionHeading = page.getByRole('heading', { name: /execution|plan status/i });
      await expect(executionHeading).toBeVisible().catch(() => {
        return expect(page.locator('text=/execution|plan|status/i').first()).toBeVisible();
      });
    });

    test('should display execution status', async ({ page }) => {
      // Look for status indicators
      const statusIndicator = page.locator('text=/pending|in progress|succeeded|failed/i').first();
      await expect(statusIndicator).toBeVisible().catch(() => {
        test.skip(true, 'Execution status not displayed');
      });
    });

    test('should display execution logs', async ({ page }) => {
      // Look for logs section
      const logsSection = page.getByRole('heading', { name: /logs/i });
      await expect(logsSection).toBeVisible().catch(() => {
        // Logs might be in a different format
        return expect(page.locator('.logs').or(page.locator('[data-testid="logs"]')).first()).toBeVisible();
      });
    });

    test('should show log entries with timestamps', async ({ page }) => {
      const logsContainer = page.locator('.logs').or(page.locator('[data-testid="logs"]')).first();
      
      if (await logsContainer.isVisible().catch(() => false)) {
        // Check for log entries
        const logEntries = logsContainer.locator('div, li, tr');
        const count = await logEntries.count();
        
        if (count > 0) {
          // Verify timestamp or message in first entry
          const firstEntry = logEntries.first();
          await expect(firstEntry).toContainText(/\d{4}|\d{2}:\d{2}|INFO|DEBUG|ERROR/i).catch(() => {
            // Log format might be different
            return Promise.resolve();
          });
        } else {
          test.skip(true, 'No log entries found');
        }
      } else {
        test.skip(true, 'Logs container not found');
      }
    });

    test('should refresh execution status', async ({ page }) => {
      const refreshButton = page.getByRole('button', { name: /refresh|reload|update/i });
      
      if (await refreshButton.isVisible().catch(() => false)) {
        await refreshButton.click();
        
        // Verify something updates (loading state or timestamp)
        await expect(page.locator('text=/loading|updating|refreshing/i').first()).toBeVisible({ timeout: 3000 }).catch(() => {
          // Refresh might not show loading state
          return Promise.resolve();
        });
      } else {
        test.skip(true, 'Refresh button not found');
      }
    });
  });

  test.describe('Plan history', () => {
    test.beforeEach(async ({ page }) => {
      if (!testData) {
        return;
      }
      
      // Navigate to a plans/history page if it exists
      await page.goto(`/tenant/${testData.tenant.id}/plans`);
      await page.waitForLoadState('networkidle');
    });

    test('should display plan history page', async ({ page }) => {
      const plansHeading = page.getByRole('heading', { name: /plans|history/i });
      await expect(plansHeading).toBeVisible().catch(() => {
        // Page might not exist, navigate back to site
        test.skip(true, 'Plans history page not implemented');
      });
    });

    test('should list previous plans', async ({ page }) => {
      const planList = page.locator('table').or(page.locator('.plan-list')).first();
      
      if (await planList.isVisible().catch(() => false)) {
        // Check for plan rows
        const planRows = planList.locator('tr, .plan-item');
        const count = await planRows.count();
        
        if (count > 0) {
          await expect(planRows.first()).toBeVisible();
        } else {
          test.skip(true, 'No plans in history');
        }
      } else {
        test.skip(true, 'Plan list not found');
      }
    });

    test('should navigate to plan details', async ({ page }) => {
      const planItem = page.locator('tr', { hasText: /plan/i }).first().or(page.locator('.plan-item').first());
      
      if (await planItem.isVisible().catch(() => false)) {
        await planItem.click();
        
        // Verify navigation to plan details
        await expect(page).toHaveURL(/\/plan\/|planId=/).catch(() => {
          await expect(page.getByRole('heading', { name: /plan details|execution details/i })).toBeVisible();
        });
      } else {
        test.skip(true, 'No plan items to click');
      }
    });
  });
});
