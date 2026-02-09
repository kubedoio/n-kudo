/**
 * API functions for n-kudo control plane
 * All endpoints from the OpenAPI specification
 */

import { apiClient, handleAPIError, APIErrorResponse } from './client';
import {
  // Models
  Tenant,
  Site,
  Host,
  MicroVM,
  Execution,
  ExecutionLog,
  APIKey,
  // Requests
  CreateTenantRequest,
  CreateSiteRequest,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  IssueEnrollmentTokenRequest,
  IssueEnrollmentTokenResponse,
  ApplyPlanRequest,
  ApplyPlanResponse,
  // List Responses
  ListSitesResponse,
  ListHostsResponse,
  ListVMsResponse,
  ListExecutionLogsResponse,
  // Enrollment
  EnrollRequest,
  EnrollResponse,
  HeartbeatRequest,
  IngestLogsRequest,
  // Enrollment Tokens
  EnrollmentToken,
} from './types';

// ============================================
// Health
// ============================================

/**
 * Check service health
 * @returns Promise resolving when service is healthy
 */
export const getHealth = async (): Promise<void> => {
  try {
    await apiClient.get('/healthz');
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// Tenants
// ============================================

/**
 * Create a new tenant (admin only)
 * @param data - Tenant creation data
 * @returns Created tenant
 */
export const createTenant = async (data: CreateTenantRequest): Promise<Tenant> => {
  try {
    const response = await apiClient.post<Tenant>('/tenants', data);
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * List all tenants (admin only)
 * Note: This endpoint is not explicitly in OpenAPI but commonly needed
 * @returns Array of tenants
 */
export const listTenants = async (): Promise<Tenant[]> => {
  try {
    const response = await apiClient.get<Tenant[]>('/tenants');
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Get a tenant by ID (admin only)
 * Note: This endpoint is not explicitly in OpenAPI but commonly needed
 * @param tenantId - Tenant UUID
 * @returns Tenant
 */
export const getTenant = async (tenantId: string): Promise<Tenant> => {
  try {
    const response = await apiClient.get<Tenant>(`/tenants/${tenantId}`);
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// API Keys
// ============================================

/**
 * List all API keys for a tenant (admin only)
 * @param tenantId - Tenant UUID
 * @returns Array of API keys
 */
export const listAPIKeys = async (tenantId: string): Promise<APIKey[]> => {
  try {
    const response = await apiClient.get<{ api_keys: APIKey[] }>(
      `/tenants/${tenantId}/api-keys`
    );
    return response.data.api_keys;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Create a new API key for a tenant (admin only)
 * @param tenantId - Tenant UUID
 * @param data - API key creation data
 * @returns Created API key response (includes the actual key)
 */
export const createAPIKey = async (
  tenantId: string,
  data: CreateAPIKeyRequest
): Promise<CreateAPIKeyResponse> => {
  try {
    const response = await apiClient.post<CreateAPIKeyResponse>(
      `/tenants/${tenantId}/api-keys`,
      data
    );
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Revoke (delete) an API key for a tenant (admin only)
 * @param tenantId - Tenant UUID
 * @param keyId - API Key UUID
 */
export const revokeAPIKey = async (tenantId: string, keyId: string): Promise<void> => {
  try {
    await apiClient.delete(`/tenants/${tenantId}/api-keys/${keyId}`);
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// Sites
// ============================================

/**
 * Create a new site for a tenant
 * @param tenantId - Tenant UUID
 * @param data - Site creation data
 * @returns Created site
 */
export const createSite = async (
  tenantId: string,
  data: CreateSiteRequest
): Promise<Site> => {
  try {
    const response = await apiClient.post<Site>(`/tenants/${tenantId}/sites`, data);
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * List all sites for a tenant
 * @param tenantId - Tenant UUID
 * @returns Array of sites
 */
export const listSites = async (tenantId: string): Promise<Site[]> => {
  try {
    const response = await apiClient.get<ListSitesResponse>(`/tenants/${tenantId}/sites`);
    return response.data.sites;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Get a specific site by ID
 * Note: Uses listSites and filters since single site GET is not in OpenAPI
 * @param siteId - Site UUID
 * @returns Site or null if not found
 */
export const getSite = async (_siteId: string): Promise<Site | null> => {
  try {
    // Since there's no direct GET /sites/{siteId} endpoint,
    // we would need to know the tenantId to list sites
    // This is a limitation of the current API design
    // For now, we'll throw an error indicating this needs tenant context
    throw new Error('getSite requires tenantId. Use listSites(tenantId) and filter instead.');
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Get a site by ID within a specific tenant
 * @param tenantId - Tenant UUID
 * @param siteId - Site UUID
 * @returns Site or null if not found
 */
export const getSiteByTenant = async (
  tenantId: string,
  siteId: string
): Promise<Site | null> => {
  try {
    const sites = await listSites(tenantId);
    return sites.find((site) => site.id === siteId) || null;
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// Enrollment Tokens
// ============================================

/**
 * Issue a one-time enrollment token for a site
 * @param tenantId - Tenant UUID
 * @param siteId - Site UUID
 * @param ttl - Time to live in seconds (default: 900)
 * @returns Enrollment token response
 */
export const issueToken = async (
  tenantId: string,
  siteId: string,
  ttl: number = 900
): Promise<IssueEnrollmentTokenResponse> => {
  try {
    const request: IssueEnrollmentTokenRequest = {
      site_id: siteId,
      expires_in_seconds: ttl,
    };
    const response = await apiClient.post<IssueEnrollmentTokenResponse>(
      `/tenants/${tenantId}/enrollment-tokens`,
      request
    );
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * List all enrollment tokens for a tenant
 * @param tenantId - Tenant UUID
 * @returns Array of enrollment tokens
 */
export const listEnrollmentTokens = async (tenantId: string): Promise<EnrollmentToken[]> => {
  try {
    const response = await apiClient.get<EnrollmentToken[]>(`/tenants/${tenantId}/enrollment-tokens`);
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// Enrollment (Agent-facing, no auth required)
// ============================================

/**
 * Enroll an edge agent (token -> mTLS cert)
 * @param data - Enrollment request data
 * @returns Enrollment response with certificates
 */
export const enrollAgent = async (data: EnrollRequest): Promise<EnrollResponse> => {
  try {
    const response = await apiClient.post<EnrollResponse>('/enroll', data);
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Send agent heartbeat (mTLS)
 * @param data - Heartbeat request data
 */
export const sendHeartbeat = async (data: HeartbeatRequest): Promise<void> => {
  try {
    await apiClient.post('/agents/heartbeat', data);
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Ingest agent logs (mTLS, best effort)
 * @param data - Log ingestion request data
 */
export const ingestLogs = async (data: IngestLogsRequest): Promise<void> => {
  try {
    await apiClient.post('/agents/logs', data);
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// Plans
// ============================================

/**
 * Apply a plan to a site and return execution status
 * @param siteId - Site UUID
 * @param plan - Plan application request
 * @returns Plan application response with executions
 */
export const applyPlan = async (
  siteId: string,
  plan: ApplyPlanRequest
): Promise<ApplyPlanResponse> => {
  try {
    const response = await apiClient.post<ApplyPlanResponse>(
      `/sites/${siteId}/plans`,
      plan
    );
    return response.data;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * List pending plans/executions for a site
 * This endpoint queries executions with PENDING or IN_PROGRESS states
 * @param siteId - Site UUID
 * @returns Array of pending executions
 */
export const listPendingPlans = async (siteId: string): Promise<Execution[]> => {
  try {
    const response = await apiClient.get<{ executions: Execution[] }>(`/sites/${siteId}/executions?status=PENDING,IN_PROGRESS`);
    return response.data.executions;
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// VMs (MicroVMs)
// ============================================

/**
 * List all VMs for a site
 * @param siteId - Site UUID
 * @returns Array of microVMs
 */
export const listVMs = async (siteId: string): Promise<MicroVM[]> => {
  try {
    const response = await apiClient.get<ListVMsResponse>(`/sites/${siteId}/vms`);
    return response.data.vms;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Get a specific VM by ID
 * @param vmId - VM UUID
 * @returns MicroVM or null if not found
 */
export const getVM = async (_vmId: string): Promise<MicroVM | null> => {
  try {
    // Since there's no direct GET /vms/{vmId} endpoint,
    // we would need to know the siteId to list VMs
    throw new Error('getVM requires siteId. Use listVMs(siteId) and filter instead.');
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * Get a VM by ID within a specific site
 * @param siteId - Site UUID
 * @param vmId - VM UUID
 * @returns MicroVM or null if not found
 */
export const getVMBySite = async (
  siteId: string,
  vmId: string
): Promise<MicroVM | null> => {
  try {
    const vms = await listVMs(siteId);
    return vms.find((vm) => vm.id === vmId) || null;
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// Hosts
// ============================================

/**
 * List all hosts for a site
 * @param siteId - Site UUID
 * @returns Array of hosts
 */
export const listHosts = async (siteId: string): Promise<Host[]> => {
  try {
    const response = await apiClient.get<ListHostsResponse>(`/sites/${siteId}/hosts`);
    return response.data.hosts;
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// Executions
// ============================================

interface ListExecutionsOptions {
  status?: string;  // comma-separated statuses
  limit?: number;
}

/**
 * List executions for a site with optional filters
 * @param siteId - Site UUID
 * @param options - Optional filters (status, limit)
 * @returns Array of executions
 */
export const listExecutions = async (
  siteId: string,
  options?: ListExecutionsOptions
): Promise<Execution[]> => {
  try {
    const params: { status?: string; limit?: number } = {};
    if (options?.status !== undefined) {
      params.status = options.status;
    }
    if (options?.limit !== undefined) {
      params.limit = options.limit;
    }
    const response = await apiClient.get<{ executions: Execution[] }>(
      `/sites/${siteId}/executions`,
      { params }
    );
    return response.data.executions;
  } catch (error) {
    throw handleAPIError(error);
  }
};

/**
 * List logs for an execution
 * @param executionId - Execution UUID
 * @param limit - Maximum number of logs to return
 * @returns Array of execution logs
 */
export const listExecutionLogs = async (
  executionId: string,
  limit?: number
): Promise<ExecutionLog[]> => {
  try {
    const params: { limit?: number } = {};
    if (limit !== undefined) {
      params.limit = limit;
    }
    const response = await apiClient.get<ListExecutionLogsResponse>(
      `/executions/${executionId}/logs`,
      { params }
    );
    return response.data.logs;
  } catch (error) {
    throw handleAPIError(error);
  }
};

// ============================================
// Re-export types for convenience
// ============================================

export type { APIErrorResponse };
