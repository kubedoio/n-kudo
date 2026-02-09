/**
 * n-kudo API Client
 * 
 * This module provides a complete TypeScript API client for the n-kudo control plane.
 * 
 * Usage:
 * ```typescript
 * import { 
 *   apiKeyStorage, 
 *   useSites, 
 *   useVMs, 
 *   useApplyPlan 
 * } from '@/api';
 * 
 * // Set API key
 * apiKeyStorage.setApiKey('your-api-key');
 * 
 * // Use in React components
 * const { data: sites, isLoading } = useSites(tenantId);
 * ```
 */

// Export client and utilities
export { apiClient, apiKeyStorage, handleAPIError } from './client';
export type { APIErrorResponse } from './client';

// Export all types
export * from './types';

// Export API functions
export * from './api';

// Export React Query hooks
export * from './hooks';
