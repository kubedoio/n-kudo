/**
 * Hook to list API keys for a tenant
 */

import { useQuery } from '@tanstack/react-query';
import { listAPIKeys } from '../api';
import { APIKey } from '../types';
import { APIErrorResponse } from '../client';

export function useAPIKeys(tenantId: string) {
  return useQuery<APIKey[], APIErrorResponse>({
    queryKey: ['tenants', tenantId, 'api-keys'],
    queryFn: () => listAPIKeys(tenantId),
    enabled: !!tenantId,
  });
}
