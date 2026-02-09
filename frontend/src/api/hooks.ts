/**
 * TanStack Query (React Query) hooks for n-kudo API
 * Provides caching, refetching, and optimistic updates
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  UseQueryOptions,
  UseMutationOptions,
} from '@tanstack/react-query';

import {
  // API functions
  getHealth,
  getTenant,
  createTenant,
  listTenants,
  createAPIKey,
  listAPIKeys,
  revokeAPIKey,
  createSite,
  listSites,
  getSiteByTenant,
  issueToken,
  applyPlan,
  listVMs,
  getVMBySite,
  listHosts,
  listExecutions,
  listExecutionLogs,
  listPendingPlans,
  listEnrollmentTokens,
} from './api';
import {
  // Types
  Tenant,
  Site,
  Host,
  MicroVM,
  Execution,
  ExecutionLog,
  APIKey,
  CreateTenantRequest,
  CreateSiteRequest,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  IssueEnrollmentTokenResponse,
  ApplyPlanRequest,
  ApplyPlanResponse,
  PlanAction,
  EnrollmentToken,
} from './types';
import { APIErrorResponse } from './client';

// ============================================
// Query Keys
// ============================================

const queryKeys = {
  health: ['health'] as const,
  tenants: ['tenants'] as const,
  tenant: (id: string) => ['tenants', id] as const,
  sites: (tenantId: string) => ['tenants', tenantId, 'sites'] as const,
  site: (tenantId: string, siteId: string) => ['tenants', tenantId, 'sites', siteId] as const,
  hosts: (siteId: string) => ['sites', siteId, 'hosts'] as const,
  vms: (siteId: string) => ['sites', siteId, 'vms'] as const,
  vm: (siteId: string, vmId: string) => ['sites', siteId, 'vms', vmId] as const,
  executions: (siteId: string, options?: { status?: string; limit?: number }) => 
    ['sites', siteId, 'executions', options] as const,
  executionLogs: (executionId: string) => ['executions', executionId, 'logs'] as const,
  pendingPlans: (siteId: string) => ['sites', siteId, 'pending-plans'] as const,
  enrollmentTokens: (tenantId: string) => ['tenants', tenantId, 'enrollment-tokens'] as const,
};

// ============================================
// Health Hooks
// ============================================

/**
 * Hook to check API health
 */
export const useHealth = (options?: UseQueryOptions<void, APIErrorResponse>) => {
  return useQuery<void, APIErrorResponse>({
    queryKey: queryKeys.health,
    queryFn: getHealth,
    retry: false,
    ...options,
  });
};

// ============================================
// Tenant Hooks
// ============================================

/**
 * Hook to list all tenants
 */
export const useTenants = (options?: UseQueryOptions<Tenant[], APIErrorResponse>) => {
  return useQuery<Tenant[], APIErrorResponse>({
    queryKey: queryKeys.tenants,
    queryFn: listTenants,
    ...options,
  });
};

/**
 * Hook to get a single tenant by ID
 */
export const useTenant = (
  tenantId: string,
  options?: UseQueryOptions<Tenant, APIErrorResponse>
) => {
  return useQuery<Tenant, APIErrorResponse>({
    queryKey: queryKeys.tenant(tenantId),
    queryFn: () => getTenant(tenantId),
    enabled: !!tenantId,
    ...options,
  });
};

/**
 * Hook to create a new tenant
 */
export const useCreateTenant = (
  options?: UseMutationOptions<Tenant, APIErrorResponse, CreateTenantRequest, unknown>
) => {
  const queryClient = useQueryClient();

  return useMutation<Tenant, APIErrorResponse, CreateTenantRequest, unknown>({
    mutationFn: createTenant,
    onSuccess: (data, variables, context) => {
      // Invalidate tenants list
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants });
      // Call user's onSuccess if provided (TanStack Query v5 signature)
      options?.onSuccess?.(data, variables, context, {} as never);
    },
    ...options,
  });
};

// ============================================
// API Key Hooks
// ============================================

interface CreateAPIKeyVariables {
  tenantId: string;
  data: CreateAPIKeyRequest;
}

/**
 * Hook to list API keys for a tenant
 */
export const useAPIKeys = (
  tenantId: string,
  options?: UseQueryOptions<APIKey[], APIErrorResponse>
) => {
  return useQuery<APIKey[], APIErrorResponse>({
    queryKey: ['tenants', tenantId, 'api-keys'],
    queryFn: () => listAPIKeys(tenantId),
    enabled: !!tenantId,
    ...options,
  });
};

/**
 * Hook to create a new API key for a tenant
 */
export const useCreateAPIKey = (
  options?: UseMutationOptions<CreateAPIKeyResponse, APIErrorResponse, CreateAPIKeyVariables>
) => {
  const queryClient = useQueryClient();

  return useMutation<CreateAPIKeyResponse, APIErrorResponse, CreateAPIKeyVariables>({
    mutationFn: ({ tenantId, data }) => createAPIKey(tenantId, data),
    onSuccess: (data, variables, context) => {
      // Invalidate api-keys query for the tenant
      queryClient.invalidateQueries({
        queryKey: ['tenants', variables.tenantId, 'api-keys'],
      });
      // Call user's onSuccess if provided
      options?.onSuccess?.(data, variables, context, {} as never);
    },
    ...options,
  });
};

interface RevokeAPIKeyVariables {
  tenantId: string;
  keyId: string;
}

/**
 * Hook to revoke (delete) an API key for a tenant
 */
export const useRevokeAPIKey = (
  options?: UseMutationOptions<void, APIErrorResponse, RevokeAPIKeyVariables>
) => {
  const queryClient = useQueryClient();

  return useMutation<void, APIErrorResponse, RevokeAPIKeyVariables>({
    mutationFn: ({ tenantId, keyId }) => revokeAPIKey(tenantId, keyId),
    onSuccess: (data, variables, context) => {
      // Invalidate api-keys query for the tenant
      queryClient.invalidateQueries({
        queryKey: ['tenants', variables.tenantId, 'api-keys'],
      });
      // Call user's onSuccess if provided
      options?.onSuccess?.(data, variables, context, {} as never);
    },
    ...options,
  });
};

// ============================================
// Site Hooks
// ============================================

interface CreateSiteVariables {
  tenantId: string;
  data: CreateSiteRequest;
}

/**
 * Hook to list sites for a tenant
 */
export const useSites = (
  tenantId: string,
  options?: UseQueryOptions<Site[], APIErrorResponse>
) => {
  return useQuery<Site[], APIErrorResponse>({
    queryKey: queryKeys.sites(tenantId),
    queryFn: () => listSites(tenantId),
    enabled: !!tenantId,
    ...options,
  });
};

/**
 * Hook to get a specific site
 */
export const useSite = (
  tenantId: string,
  siteId: string,
  options?: UseQueryOptions<Site | null, APIErrorResponse>
) => {
  return useQuery<Site | null, APIErrorResponse>({
    queryKey: queryKeys.site(tenantId, siteId),
    queryFn: () => getSiteByTenant(tenantId, siteId),
    enabled: !!tenantId && !!siteId,
    ...options,
  });
};

/**
 * Hook to create a new site
 */
export const useCreateSite = (
  options?: UseMutationOptions<Site, APIErrorResponse, CreateSiteVariables, unknown>
) => {
  const queryClient = useQueryClient();

  return useMutation<Site, APIErrorResponse, CreateSiteVariables, unknown>({
    mutationFn: ({ tenantId, data }) => createSite(tenantId, data),
    onSuccess: (data, variables, context) => {
      // Invalidate sites list for the tenant
      queryClient.invalidateQueries({
        queryKey: queryKeys.sites(variables.tenantId),
      });
      // Call user's onSuccess if provided (TanStack Query v5 signature)
      options?.onSuccess?.(data, variables, context, {} as never);
    },
    ...options,
  });
};

// ============================================
// Enrollment Token Hooks
// ============================================

interface IssueTokenVariables {
  tenantId: string;
  siteId: string;
  ttl?: number;
}

/**
 * Hook to issue an enrollment token
 */
export const useIssueToken = (
  options?: UseMutationOptions<IssueEnrollmentTokenResponse, APIErrorResponse, IssueTokenVariables>
) => {
  return useMutation<IssueEnrollmentTokenResponse, APIErrorResponse, IssueTokenVariables>({
    mutationFn: ({ tenantId, siteId, ttl }) => issueToken(tenantId, siteId, ttl),
    ...options,
  });
};

// ============================================
// Enrollment Token Hooks
// ============================================

/**
 * Hook to list enrollment tokens for a tenant
 */
export const useEnrollmentTokens = (
  tenantId: string,
  options?: UseQueryOptions<EnrollmentToken[], APIErrorResponse>
) => {
  return useQuery<EnrollmentToken[], APIErrorResponse>({
    queryKey: queryKeys.enrollmentTokens(tenantId),
    queryFn: () => listEnrollmentTokens(tenantId),
    enabled: !!tenantId,
    ...options,
  });
};

// ============================================
// Plan Hooks
// ============================================

interface ApplyPlanVariables {
  siteId: string;
  plan: ApplyPlanRequest;
}

interface ApplyPlanFromActionsVariables {
  siteId: string;
  idempotencyKey: string;
  actions: PlanAction[];
  clientRequestId?: string;
}

/**
 * Hook to apply a plan to a site
 */
export const useApplyPlan = (
  options?: UseMutationOptions<ApplyPlanResponse, APIErrorResponse, ApplyPlanVariables, unknown>
) => {
  const queryClient = useQueryClient();

  return useMutation<ApplyPlanResponse, APIErrorResponse, ApplyPlanVariables, unknown>({
    mutationFn: ({ siteId, plan }) => applyPlan(siteId, plan),
    onSuccess: (data, variables, context) => {
      // Invalidate VMs and hosts lists since plan execution may change them
      queryClient.invalidateQueries({
        queryKey: queryKeys.vms(variables.siteId),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.hosts(variables.siteId),
      });
      // Call user's onSuccess if provided (TanStack Query v5 signature)
      options?.onSuccess?.(data, variables, context, {} as never);
    },
    ...options,
  });
};

/**
 * Hook to apply a plan from actions (convenience wrapper)
 */
export const useApplyPlanFromActions = (
  options?: UseMutationOptions<ApplyPlanResponse, APIErrorResponse, ApplyPlanFromActionsVariables, unknown>
) => {
  const queryClient = useQueryClient();

  return useMutation<ApplyPlanResponse, APIErrorResponse, ApplyPlanFromActionsVariables, unknown>({
    mutationFn: ({ siteId, idempotencyKey, actions, clientRequestId }) => 
      applyPlan(siteId, {
        idempotency_key: idempotencyKey,
        client_request_id: clientRequestId,
        actions,
      }),
    onSuccess: (data, variables, context) => {
      // Invalidate VMs and hosts lists since plan execution may change them
      queryClient.invalidateQueries({
        queryKey: queryKeys.vms(variables.siteId),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.hosts(variables.siteId),
      });
      // Call user's onSuccess if provided (TanStack Query v5 signature)
      options?.onSuccess?.(data, variables, context, {} as never);
    },
    ...options,
  });
};

// ============================================
// VM Hooks
// ============================================

/**
 * Hook to list VMs for a site
 */
export const useVMs = (
  siteId: string,
  options?: UseQueryOptions<MicroVM[], APIErrorResponse>
) => {
  return useQuery<MicroVM[], APIErrorResponse>({
    queryKey: queryKeys.vms(siteId),
    queryFn: () => listVMs(siteId),
    enabled: !!siteId,
    ...options,
  });
};

/**
 * Hook to get a specific VM
 */
export const useVM = (
  siteId: string,
  vmId: string,
  options?: UseQueryOptions<MicroVM | null, APIErrorResponse>
) => {
  return useQuery<MicroVM | null, APIErrorResponse>({
    queryKey: queryKeys.vm(siteId, vmId),
    queryFn: () => getVMBySite(siteId, vmId),
    enabled: !!siteId && !!vmId,
    ...options,
  });
};

// ============================================
// Host Hooks
// ============================================

/**
 * Hook to list hosts for a site
 */
export const useHosts = (
  siteId: string,
  options?: UseQueryOptions<Host[], APIErrorResponse>
) => {
  return useQuery<Host[], APIErrorResponse>({
    queryKey: queryKeys.hosts(siteId),
    queryFn: () => listHosts(siteId),
    enabled: !!siteId,
    ...options,
  });
};

// ============================================
// Execution Hooks
// ============================================

interface UseExecutionsOptions {
  status?: string;  // comma-separated statuses
  limit?: number;
}

/**
 * Hook to list executions for a site
 */
export const useExecutions = (
  siteId: string,
  options?: UseExecutionsOptions,
  queryOptions?: UseQueryOptions<Execution[], APIErrorResponse>
) => {
  return useQuery<Execution[], APIErrorResponse>({
    queryKey: queryKeys.executions(siteId, options),
    queryFn: () => listExecutions(siteId, options),
    enabled: !!siteId,
    ...queryOptions,
  });
};

/**
 * Hook to list execution logs
 */
export const useExecutionLogs = (
  executionId: string,
  limit?: number,
  options?: UseQueryOptions<ExecutionLog[], APIErrorResponse>
) => {
  return useQuery<ExecutionLog[], APIErrorResponse>({
    queryKey: [...queryKeys.executionLogs(executionId), { limit }],
    queryFn: () => listExecutionLogs(executionId, limit),
    enabled: !!executionId,
    ...options,
  });
};

/**
 * Hook to list pending plans for a site
 */
export const usePendingPlans = (
  siteId: string,
  options?: UseQueryOptions<Execution[], APIErrorResponse>
) => {
  return useQuery<Execution[], APIErrorResponse>({
    queryKey: queryKeys.pendingPlans(siteId),
    queryFn: () => listPendingPlans(siteId),
    enabled: !!siteId,
    ...options,
  });
};

// ============================================
// Combined/Convenience Hooks
// ============================================

/**
 * Hook to refresh all data for a tenant
 */
export const useRefreshTenantData = () => {
  const queryClient = useQueryClient();

  return (tenantId: string) => {
    queryClient.invalidateQueries({
      queryKey: queryKeys.sites(tenantId),
    });
  };
};

/**
 * Hook to refresh all data for a site
 */
export const useRefreshSiteData = () => {
  const queryClient = useQueryClient();

  return (siteId: string) => {
    queryClient.invalidateQueries({
      queryKey: queryKeys.vms(siteId),
    });
    queryClient.invalidateQueries({
      queryKey: queryKeys.hosts(siteId),
    });
  };
};

// ============================================
// Re-export query keys for external use
// ============================================

export { queryKeys };
