/**
 * Hook to revoke (delete) an API key for a tenant
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { revokeAPIKey } from '../api';
import { APIErrorResponse } from '../client';

interface RevokeAPIKeyVariables {
  tenantId: string;
  keyId: string;
}

export function useRevokeAPIKey() {
  const queryClient = useQueryClient();

  return useMutation<void, APIErrorResponse, RevokeAPIKeyVariables>({
    mutationFn: ({ tenantId, keyId }) => revokeAPIKey(tenantId, keyId),
    onSuccess: (_data, variables) => {
      // Invalidate api-keys query for the tenant
      queryClient.invalidateQueries({
        queryKey: ['tenants', variables.tenantId, 'api-keys'],
      });
    },
  });
}
