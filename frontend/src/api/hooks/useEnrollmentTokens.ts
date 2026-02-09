/**
 * Hook for fetching enrollment tokens for a tenant
 */

import { useQuery } from '@tanstack/react-query';
import { listEnrollmentTokens } from '../api';
import { EnrollmentToken } from '../types';
import { APIErrorResponse } from '../client';

/**
 * Query key for enrollment tokens
 */
export const enrollmentTokensQueryKey = (tenantId: string) =>
  ['tenants', tenantId, 'enrollment-tokens'] as const;

/**
 * Hook to list enrollment tokens for a tenant
 * @param tenantId - Tenant UUID
 * @returns Query result with enrollment tokens
 */
export function useEnrollmentTokens(tenantId: string) {
  return useQuery<EnrollmentToken[], APIErrorResponse>({
    queryKey: enrollmentTokensQueryKey(tenantId),
    queryFn: () => listEnrollmentTokens(tenantId),
    enabled: !!tenantId,
  });
}
