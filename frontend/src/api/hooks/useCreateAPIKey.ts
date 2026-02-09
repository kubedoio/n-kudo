/**
 * Hook to create a new API key for a tenant
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createAPIKey } from '../api';
import { CreateAPIKeyRequest, CreateAPIKeyResponse } from '../types';
import { APIErrorResponse } from '../client';

interface CreateAPIKeyVariables {
  tenantId: string;
  data: CreateAPIKeyRequest;
}

export function useCreateAPIKey() {
  const queryClient = useQueryClient();

  return useMutation<CreateAPIKeyResponse, APIErrorResponse, CreateAPIKeyVariables>({
    mutationFn: ({ tenantId, data }) => createAPIKey(tenantId, data),
    onSuccess: (_data, variables) => {
      // Invalidate api-keys query for the tenant
      queryClient.invalidateQueries({
        queryKey: ['tenants', variables.tenantId, 'api-keys'],
      });
    },
  });
}
