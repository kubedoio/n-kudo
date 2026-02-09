/**
 * Authentication utilities for E2E tests
 * Handles admin key setup, tenant creation, and API key management
 */

import { Page, BrowserContext } from '@playwright/test';

// Storage keys (must match the frontend)
const API_KEY_STORAGE_KEY = 'n-kudo-api-key';
const ADMIN_KEY_STORAGE_KEY = 'n-kudo-admin-key';

export interface TestTenant {
  id: string;
  slug: string;
  name: string;
  apiKey: string;
}

/**
 * Set admin key in localStorage via page evaluation
 */
export async function setAdminKey(page: Page, adminKey: string): Promise<void> {
  await page.evaluate((key) => {
    localStorage.setItem('n-kudo-admin-key', key);
  }, adminKey);
}

/**
 * Set admin key in browser context storage state
 */
export async function setAdminKeyInContext(context: BrowserContext, adminKey: string): Promise<void> {
  await context.addInitScript((key) => {
    localStorage.setItem('n-kudo-admin-key', key);
  }, adminKey);
}

/**
 * Set API key in localStorage via page evaluation
 */
export async function setApiKey(page: Page, apiKey: string): Promise<void> {
  await page.evaluate((key) => {
    localStorage.setItem('n-kudo-api-key', key);
  }, apiKey);
}

/**
 * Clear all auth keys from localStorage
 */
export async function clearAuthKeys(page: Page): Promise<void> {
  await page.evaluate(() => {
    localStorage.removeItem('n-kudo-api-key');
    localStorage.removeItem('n-kudo-admin-key');
  });
}

/**
 * Get admin key from localStorage
 */
export async function getAdminKey(page: Page): Promise<string | null> {
  return await page.evaluate(() => {
    return localStorage.getItem('n-kudo-admin-key');
  });
}

/**
 * Get API key from localStorage
 */
export async function getApiKey(page: Page): Promise<string | null> {
  return await page.evaluate(() => {
    return localStorage.getItem('n-kudo-api-key');
  });
}

/**
 * Login as admin by setting the admin key
 */
export async function loginAsAdmin(page: Page, adminKey: string = process.env.ADMIN_KEY || 'dev-admin-key'): Promise<void> {
  await setAdminKey(page, adminKey);
  // Reload to apply the admin key
  await page.reload();
}

/**
 * Login as tenant by setting the API key
 */
export async function loginAsTenant(page: Page, apiKey: string): Promise<void> {
  await setApiKey(page, apiKey);
  // Reload to apply the API key
  await page.reload();
}

/**
 * Wait for auth to be ready (check if keys are set)
 */
export async function waitForAuthReady(page: Page, timeout: number = 5000): Promise<void> {
  await page.waitForFunction(() => {
    return localStorage.getItem('n-kudo-admin-key') !== null ||
           localStorage.getItem('n-kudo-api-key') !== null;
  }, { timeout });
}
