/**
 * Axios HTTP client for n-kudo API
 * Handles base URL, authentication headers, and error handling
 */

import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig } from 'axios';

// API Key storage keys
const API_KEY_STORAGE_KEY = 'n-kudo-api-key';
const ADMIN_KEY_STORAGE_KEY = 'n-kudo-admin-key';

// Get base URL from environment variable or use default
const getBaseURL = (): string => {
  return import.meta.env.VITE_API_BASE_URL || 'https://localhost:8443';
};

// Create axios instance with default config
export const apiClient: AxiosInstance = axios.create({
  baseURL: getBaseURL(),
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000, // 30 second timeout
});

// ============================================
// Request Interceptor - Add auth headers
// ============================================

apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {
    // Check if this is an admin endpoint
    const isAdminEndpoint = config.url?.startsWith('/tenants') && 
                           (config.method === 'post' || config.url.includes('/api-keys'));

    if (isAdminEndpoint) {
      // Use admin key for admin endpoints
      const adminKey = localStorage.getItem(ADMIN_KEY_STORAGE_KEY);
      if (adminKey) {
        config.headers['X-Admin-Key'] = adminKey;
      }
    } else {
      // Use API key for tenant-scoped endpoints
      const apiKey = localStorage.getItem(API_KEY_STORAGE_KEY);
      if (apiKey) {
        config.headers['X-API-Key'] = apiKey;
      }
    }

    return config;
  },
  (error: AxiosError): Promise<AxiosError> => {
    return Promise.reject(error);
  }
);

// ============================================
// Response Interceptor - Handle errors
// ============================================

apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError): Promise<never> => {
    if (error.response) {
      // Server responded with error status
      const status = error.response.status;
      const data = error.response.data as { message?: string; code?: string };

      switch (status) {
        case 401:
          console.error('Authentication failed: Invalid or missing API key');
          // Optionally redirect to login or clear stored keys
          break;
        case 403:
          console.error('Authorization failed: Insufficient permissions');
          break;
        case 404:
          console.error('Resource not found');
          break;
        case 409:
          console.error('Conflict: Resource already exists or conflict in request');
          break;
        case 422:
          console.error('Validation error:', data?.message || 'Invalid request data');
          break;
        case 429:
          console.error('Rate limit exceeded. Please try again later.');
          break;
        case 500:
          console.error('Server error. Please try again later.');
          break;
        default:
          console.error(`HTTP ${status} error:`, data?.message || error.message);
      }
    } else if (error.request) {
      // Request was made but no response received
      console.error('Network error: No response received from server');
    } else {
      // Error in setting up the request
      console.error('Request error:', error.message);
    }

    return Promise.reject(error);
  }
);

// ============================================
// API Key Management
// ============================================

export const apiKeyStorage = {
  /**
   * Store API key in localStorage
   */
  setApiKey(key: string): void {
    localStorage.setItem(API_KEY_STORAGE_KEY, key);
  },

  /**
   * Get API key from localStorage
   */
  getApiKey(): string | null {
    return localStorage.getItem(API_KEY_STORAGE_KEY);
  },

  /**
   * Remove API key from localStorage
   */
  removeApiKey(): void {
    localStorage.removeItem(API_KEY_STORAGE_KEY);
  },

  /**
   * Check if API key exists
   */
  hasApiKey(): boolean {
    return !!localStorage.getItem(API_KEY_STORAGE_KEY);
  },

  /**
   * Store admin key in localStorage
   */
  setAdminKey(key: string): void {
    localStorage.setItem(ADMIN_KEY_STORAGE_KEY, key);
  },

  /**
   * Get admin key from localStorage
   */
  getAdminKey(): string | null {
    return localStorage.getItem(ADMIN_KEY_STORAGE_KEY);
  },

  /**
   * Remove admin key from localStorage
   */
  removeAdminKey(): void {
    localStorage.removeItem(ADMIN_KEY_STORAGE_KEY);
  },

  /**
   * Clear all stored keys
   */
  clearAll(): void {
    localStorage.removeItem(API_KEY_STORAGE_KEY);
    localStorage.removeItem(ADMIN_KEY_STORAGE_KEY);
  },
};

// ============================================
// Error Handling Helper
// ============================================

export interface APIErrorResponse {
  message: string;
  statusCode: number;
  code?: string;
}

export const handleAPIError = (error: unknown): APIErrorResponse => {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<{ message?: string; code?: string }>;
    return {
      message: axiosError.response?.data?.message || axiosError.message || 'Unknown error',
      statusCode: axiosError.response?.status || 500,
      code: axiosError.response?.data?.code,
    };
  }
  
  if (error instanceof Error) {
    return {
      message: error.message,
      statusCode: 500,
    };
  }
  
  return {
    message: 'An unexpected error occurred',
    statusCode: 500,
  };
};

export default apiClient;
