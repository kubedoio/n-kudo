/**
 * Global setup for E2E tests
 * Creates a test tenant and API key that can be used across all tests
 */

import { FullConfig } from '@playwright/test';
import { TestAPIClient, Tenant, CreateAPIKeyResponse } from './utils/api';
import * as fs from 'fs';
import * as path from 'path';

// Configuration
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8443';
const ADMIN_KEY = process.env.ADMIN_KEY || 'dev-admin-key';

// Test data storage
const TEST_DATA_DIR = path.join(__dirname, '.test-data');
const TEST_DATA_FILE = path.join(TEST_DATA_DIR, 'test-tenant.json');

export interface GlobalTestData {
  tenant: Tenant;
  apiKey: string;
  createdAt: string;
}

/**
 * Generate a unique slug for test tenant
 */
function generateTestSlug(): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 8);
  return `e2e-test-${timestamp}-${random}`;
}

/**
 * Global setup function
 */
async function globalSetup(config: FullConfig): Promise<void> {
  console.log('🔧 Running global setup...');

  // Check if backend is available
  const client = new TestAPIClient(API_BASE_URL, ADMIN_KEY);
  
  try {
    await client.init();
    
    // Test connection to backend
    try {
      await client.listTenants();
      console.log('✅ Backend connection successful');
    } catch (error) {
      console.warn('⚠️  Could not connect to backend:', (error as Error).message);
      console.warn('   Tests may fail if backend is not running');
    }

    // Create test data directory
    if (!fs.existsSync(TEST_DATA_DIR)) {
      fs.mkdirSync(TEST_DATA_DIR, { recursive: true });
    }

    // Create a test tenant
    const slug = generateTestSlug();
    const name = `E2E Test Tenant ${new Date().toISOString()}`;

    console.log(`📝 Creating test tenant: ${slug}`);
    
    const tenant = await client.createTenant({
      slug,
      name,
      primary_region: 'us-east-1',
      data_retention_days: 1, // Short retention for test data
    });

    console.log(`✅ Tenant created: ${tenant.id}`);

    // Create API key for the tenant
    const apiKeyResponse = await client.createApiKey(tenant.id, 'E2E Test Key');
    
    console.log(`✅ API key created: ${apiKeyResponse.id}`);

    // Store test data for use in tests
    const testData: GlobalTestData = {
      tenant,
      apiKey: apiKeyResponse.api_key,
      createdAt: new Date().toISOString(),
    };

    fs.writeFileSync(TEST_DATA_FILE, JSON.stringify(testData, null, 2));
    
    console.log('🎉 Global setup complete');
    console.log(`   Tenant ID: ${tenant.id}`);
    console.log(`   Tenant Slug: ${tenant.slug}`);
    console.log(`   API Key: ${apiKeyResponse.api_key.substring(0, 20)}...`);

    // Set environment variable for tests to access
    process.env.E2E_TEST_TENANT_ID = tenant.id;
    process.env.E2E_TEST_TENANT_SLUG = tenant.slug;
    process.env.E2E_TEST_API_KEY = apiKeyResponse.api_key;
    process.env.E2E_TEST_DATA_FILE = TEST_DATA_FILE;

  } catch (error) {
    console.error('❌ Global setup failed:', (error as Error).message);
    throw error;
  } finally {
    await client.dispose();
  }
}

export default globalSetup;
