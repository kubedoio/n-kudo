/**
 * Hook for fetching executions with optional filters
 */

import { useQuery } from '@tanstack/react-query';
import { listExecutions } from '../api';
import { Execution } from '../types';
import { APIErrorResponse } from '../client';

export interface UseExecutionsOptions {
  status?: string;  // comma-separated statuses
  limit?: number;
}

/**
 * Query key factory for executions
 */
const executionsQueryKey = (siteId: string, options?: UseExecutionsOptions) =>
  ['sites', siteId, 'executions', options] as const;

/**
 * Hook to list executions for a site
 * @param siteId - Site UUID
 * @param options - Optional filters (status, limit)
 * @returns Query result with executions data
 */
export function useExecutions(
  siteId: string,
  options?: UseExecutionsOptions
) {
  return useQuery<Execution[], APIErrorResponse>({
    queryKey: executionsQueryKey(siteId, options),
    queryFn: () => listExecutions(siteId, options),
    enabled: !!siteId,
  });
}

// Export query key for external use
export { executionsQueryKey };
