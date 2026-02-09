/**
 * Backend API helpers for test setup
 * Provides direct API access for creating test data
 */

import { request, APIRequestContext } from '@playwright/test';

// Default API configuration
const DEFAULT_API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8443';
const DEFAULT_ADMIN_KEY = process.env.ADMIN_KEY || 'dev-admin-key';

// Types based on API models
export interface Tenant {
  id: string;
  slug: string;
  name: string;
  primary_region: string;
  data_retention_days: number;
  created_at: string;
}

export interface CreateTenantRequest {
  slug: string;
  name: string;
  primary_region?: string;
  data_retention_days?: number;
}

export interface CreateAPIKeyResponse {
  id: string;
  tenant_id: string;
  name: string;
  api_key: string;
  expires_at: string;
}

export interface Site {
  id: string;
  tenant_id: string;
  name: string;
  external_key: string;
  location_country_code: string;
  connectivity_state: string;
  last_heartbeat_at: string;
  created_at: string;
}

export interface CreateSiteRequest {
  name: string;
  external_key?: string;
  location_country_code?: string;
}

export interface MicroVM {
  id: string;
  site_id: string;
  host_id: string;
  name: string;
  state: string;
  vcpu_count: number;
  memory_mib: number;
  updated_at: string;
}

export interface ApplyPlanRequest {
  idempotency_key: string;
  client_request_id?: string;
  actions: PlanAction[];
}

export interface PlanAction {
  operation_id?: string;
  operation: 'CREATE' | 'START' | 'STOP' | 'DELETE';
  vm_id?: string;
  name?: string;
  vcpu_count?: number;
  memory_mib?: number;
}

export interface ApplyPlanResponse {
  plan_id: string;
  plan_version: number;
  plan_status: string;
  deduplicated: boolean;
  executions: Execution[];
}

export interface Execution {
  id: string;
  plan_id: string;
  operation_id: string;
  operation_type: string;
  state: string;
  vm_id: string;
  error_code: string;
  error_message: string;
}

/**
 * API client for test setup
 */
export class TestAPIClient {
  private requestContext: APIRequestContext | null = null;
  private baseURL: string;
  private adminKey: string;

  constructor(baseURL: string = DEFAULT_API_BASE_URL, adminKey: string = DEFAULT_ADMIN_KEY) {
    this.baseURL = baseURL;
    this.adminKey = adminKey;
  }

  /**
   * Initialize the API request context
   */
  async init(): Promise<void> {
    this.requestContext = await request.newContext({
      baseURL: this.baseURL,
      extraHTTPHeaders: {
        'Content-Type': 'application/json',
        'X-Admin-Key': this.adminKey,
      },
    });
  }

  /**
   * Dispose of the request context
   */
  async dispose(): Promise<void> {
    if (this.requestContext) {
      await this.requestContext.dispose();
      this.requestContext = null;
    }
  }

  /**
   * Create a new tenant
   */
  async createTenant(requestData: CreateTenantRequest): Promise<Tenant> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const response = await this.requestContext.post('/tenants', {
      data: requestData,
    });

    if (!response.ok()) {
      const errorText = await response.text();
      throw new Error(`Failed to create tenant: ${response.status()} - ${errorText}`);
    }

    return await response.json() as Tenant;
  }

  /**
   * Create an API key for a tenant
   */
  async createApiKey(tenantId: string, name: string = 'E2E Test Key'): Promise<CreateAPIKeyResponse> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const response = await this.requestContext.post(`/tenants/${tenantId}/api-keys`, {
      data: { name },
    });

    if (!response.ok()) {
      const errorText = await response.text();
      throw new Error(`Failed to create API key: ${response.status()} - ${errorText}`);
    }

    return await response.json() as CreateAPIKeyResponse;
  }

  /**
   * List all tenants
   */
  async listTenants(): Promise<Tenant[]> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const response = await this.requestContext.get('/tenants');

    if (!response.ok()) {
      const errorText = await response.text();
      throw new Error(`Failed to list tenants: ${response.status()} - ${errorText}`);
    }

    const data = await response.json() as { tenants: Tenant[] };
    return data.tenants || [];
  }

  /**
   * Delete a tenant
   */
  async deleteTenant(tenantId: string): Promise<void> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const response = await this.requestContext.delete(`/tenants/${tenantId}`);

    if (!response.ok() && response.status() !== 404) {
      const errorText = await response.text();
      throw new Error(`Failed to delete tenant: ${response.status()} - ${errorText}`);
    }
  }

  /**
   * Create a site for a tenant (requires tenant API key)
   */
  async createSite(tenantApiKey: string, requestData: CreateSiteRequest): Promise<Site> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    // Create a new context with tenant API key
    const tenantContext = await request.newContext({
      baseURL: this.baseURL,
      extraHTTPHeaders: {
        'Content-Type': 'application/json',
        'X-API-Key': tenantApiKey,
      },
    });

    try {
      const response = await tenantContext.post('/sites', {
        data: requestData,
      });

      if (!response.ok()) {
        const errorText = await response.text();
        throw new Error(`Failed to create site: ${response.status()} - ${errorText}`);
      }

      return await response.json() as Site;
    } finally {
      await tenantContext.dispose();
    }
  }

  /**
   * List sites for a tenant
   */
  async listSites(tenantApiKey: string): Promise<Site[]> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const tenantContext = await request.newContext({
      baseURL: this.baseURL,
      extraHTTPHeaders: {
        'Content-Type': 'application/json',
        'X-API-Key': tenantApiKey,
      },
    });

    try {
      const response = await tenantContext.get('/sites');

      if (!response.ok()) {
        const errorText = await response.text();
        throw new Error(`Failed to list sites: ${response.status()} - ${errorText}`);
      }

      const data = await response.json() as { sites: Site[] };
      return data.sites || [];
    } finally {
      await tenantContext.dispose();
    }
  }

  /**
   * Apply a plan to create/start/stop/delete VMs
   */
  async applyPlan(tenantApiKey: string, requestData: ApplyPlanRequest): Promise<ApplyPlanResponse> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const tenantContext = await request.newContext({
      baseURL: this.baseURL,
      extraHTTPHeaders: {
        'Content-Type': 'application/json',
        'X-API-Key': tenantApiKey,
      },
    });

    try {
      const response = await tenantContext.post('/plans', {
        data: requestData,
      });

      if (!response.ok()) {
        const errorText = await response.text();
        throw new Error(`Failed to apply plan: ${response.status()} - ${errorText}`);
      }

      return await response.json() as ApplyPlanResponse;
    } finally {
      await tenantContext.dispose();
    }
  }

  /**
   * List VMs for a site
   */
  async listVMs(tenantApiKey: string, siteId?: string): Promise<MicroVM[]> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const tenantContext = await request.newContext({
      baseURL: this.baseURL,
      extraHTTPHeaders: {
        'Content-Type': 'application/json',
        'X-API-Key': tenantApiKey,
      },
    });

    try {
      const url = siteId ? `/sites/${siteId}/vms` : '/vms';
      const response = await tenantContext.get(url);

      if (!response.ok()) {
        const errorText = await response.text();
        throw new Error(`Failed to list VMs: ${response.status()} - ${errorText}`);
      }

      const data = await response.json() as { vms: MicroVM[] };
      return data.vms || [];
    } finally {
      await tenantContext.dispose();
    }
  }

  /**
   * Get plan execution status
   */
  async getPlanStatus(tenantApiKey: string, planId: string): Promise<unknown> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const tenantContext = await request.newContext({
      baseURL: this.baseURL,
      extraHTTPHeaders: {
        'Content-Type': 'application/json',
        'X-API-Key': tenantApiKey,
      },
    });

    try {
      const response = await tenantContext.get(`/plans/${planId}`);

      if (!response.ok()) {
        const errorText = await response.text();
        throw new Error(`Failed to get plan status: ${response.status()} - ${errorText}`);
      }

      return await response.json();
    } finally {
      await tenantContext.dispose();
    }
  }

  /**
   * Get execution logs
   */
  async getExecutionLogs(tenantApiKey: string, executionId: string): Promise<unknown[]> {
    if (!this.requestContext) {
      throw new Error('API client not initialized. Call init() first.');
    }

    const tenantContext = await request.newContext({
      baseURL: this.baseURL,
      extraHTTPHeaders: {
        'Content-Type': 'application/json',
        'X-API-Key': tenantApiKey,
      },
    });

    try {
      const response = await tenantContext.get(`/executions/${executionId}/logs`);

      if (!response.ok()) {
        const errorText = await response.text();
        throw new Error(`Failed to get execution logs: ${response.status()} - ${errorText}`);
      }

      const data = await response.json() as { logs: unknown[] };
      return data.logs || [];
    } finally {
      await tenantContext.dispose();
    }
  }
}

/**
 * Create a test tenant with API key
 */
export async function createTestTenant(
  slug: string,
  name: string,
  baseURL?: string,
  adminKey?: string
): Promise<{ tenant: Tenant; apiKeyResponse: CreateAPIKeyResponse }> {
  const client = new TestAPIClient(baseURL, adminKey);
  
  try {
    await client.init();
    
    const tenant = await client.createTenant({
      slug,
      name,
      primary_region: 'us-east-1',
      data_retention_days: 30,
    });

    const apiKeyResponse = await client.createApiKey(tenant.id, `E2E Test Key for ${slug}`);

    return { tenant, apiKeyResponse };
  } finally {
    await client.dispose();
  }
}

/**
 * Cleanup test tenant
 */
export async function cleanupTestTenant(tenantId: string, baseURL?: string, adminKey?: string): Promise<void> {
  const client = new TestAPIClient(baseURL, adminKey);
  
  try {
    await client.init();
    await client.deleteTenant(tenantId);
  } finally {
    await client.dispose();
  }
}
