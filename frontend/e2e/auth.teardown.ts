/**
 * Global teardown for E2E tests
 * Cleans up test tenant and data
 */

import { FullConfig } from '@playwright/test';
import { TestAPIClient, GlobalTestData } from './utils/api';
import * as fs from 'fs';
import * as path from 'path';

// Configuration
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8443';
const ADMIN_KEY = process.env.ADMIN_KEY || 'dev-admin-key';

const TEST_DATA_DIR = path.join(__dirname, '.test-data');
const TEST_DATA_FILE = path.join(TEST_DATA_DIR, 'test-tenant.json');

/**
 * Global teardown function
 */
async function globalTeardown(config: FullConfig): Promise<void> {
  console.log('\n🧹 Running global teardown...');

  // Check if test data exists
  if (!fs.existsSync(TEST_DATA_FILE)) {
    console.log('ℹ️  No test data file found, skipping cleanup');
    return;
  }

  const client = new TestAPIClient(API_BASE_URL, ADMIN_KEY);
  
  try {
    await client.init();

    // Read test data
    const testData: GlobalTestData = JSON.parse(fs.readFileSync(TEST_DATA_FILE, 'utf-8'));

    console.log(`🗑️  Cleaning up test tenant: ${testData.tenant.slug}`);

    // Delete the test tenant (this should cascade and clean up related data)
    try {
      await client.deleteTenant(testData.tenant.id);
      console.log(`✅ Test tenant deleted: ${testData.tenant.id}`);
    } catch (error) {
      console.warn(`⚠️  Failed to delete test tenant: ${(error as Error).message}`);
    }

    // Clean up test data file
    try {
      fs.unlinkSync(TEST_DATA_FILE);
      console.log('✅ Test data file cleaned up');
    } catch (error) {
      console.warn(`⚠️  Failed to clean up test data file: ${(error as Error).message}`);
    }

    console.log('🎉 Global teardown complete');

  } catch (error) {
    console.error('❌ Global teardown error:', (error as Error).message);
    // Don't throw - we want to clean up as much as possible
  } finally {
    await client.dispose();
  }
}

export default globalTeardown;
