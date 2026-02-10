import { apiClient, authStorage } from './client';
import type { AuthResponse, UserInfo, TenantInfo } from './client';

export interface RegisterRequest {
  email: string;
  password: string;
  display_name: string;
  tenant_name: string;
  tenant_slug?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

/**
 * Register a new user with a new tenant
 */
export async function register(data: RegisterRequest): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>('/auth/register', data);
  
  // Store auth data
  authStorage.setToken(response.data.token);
  authStorage.setUserInfo(response.data.user);
  
  return response.data;
}

/**
 * Login with email and password
 */
export async function login(data: LoginRequest): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>('/auth/login', data);
  
  // Store auth data
  authStorage.setToken(response.data.token);
  authStorage.setUserInfo(response.data.user);
  
  return response.data;
}

/**
 * Get current user profile
 */
export async function getMe(): Promise<UserInfo> {
  const response = await apiClient.get<UserInfo>('/auth/me');
  
  // Update stored user info
  authStorage.setUserInfo(response.data);
  
  return response.data;
}

/**
 * Logout the current user
 */
export function logout(): void {
  authStorage.logout();
  // Reload page to clear any cached state
  window.location.href = '/login';
}

/**
 * Check if user is authenticated
 */
export function isAuthenticated(): boolean {
  return authStorage.isAuthenticated();
}

/**
 * Get stored user info
 */
export function getCurrentUser(): UserInfo | null {
  return authStorage.getUserInfo();
}

/**
 * Switch to a different project
 * @param projectId - The project ID to switch to
 * @returns New auth response with updated token
 */
export async function switchProject(projectId: string): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>(`/projects/${projectId}/switch`);
  
  // Update auth data with new token
  authStorage.setToken(response.data.token);
  authStorage.setUserInfo(response.data.user);
  
  return response.data;
}

/**
 * Resend email verification
 * @returns Success message
 */
export async function resendVerification(): Promise<{ message: string }> {
  const response = await apiClient.post<{ message: string }>('/auth/resend-verification');
  return response.data;
}

export type { AuthResponse, UserInfo, TenantInfo };
