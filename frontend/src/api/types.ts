/**
 * TypeScript interfaces for n-kudo API models
 * Generated from OpenAPI specification
 */

// ============================================
// Core Models
// ============================================

export interface Tenant {
  id: string;
  slug: string;
  name: string;
  primary_region: string;
  data_retention_days: number;
  created_at: string;
}

export interface Site {
  id: string;
  tenant_id: string;
  name: string;
  external_key: string;
  location_country_code: string;
  connectivity_state: string;
  last_heartbeat_at: string;
  created_at: string;
}

export interface Host {
  id: string;
  hostname: string;
  cpu_cores_total: number;
  memory_bytes_total: number;
  storage_bytes_total: number;
  kvm_available: boolean;
  cloud_hypervisor_available: boolean;
  last_facts_at: string;
  agent_state: string;
}

export interface MicroVM {
  id: string;
  site_id: string;
  host_id: string;
  name: string;
  state: string;
  vcpu_count: number;
  memory_mib: number;
  updated_at: string;
}

export interface Execution {
  id: string;
  plan_id: string;
  operation_id: string;
  operation_type: 'CREATE' | 'START' | 'STOP' | 'DELETE';
  state: 'PENDING' | 'IN_PROGRESS' | 'SUCCEEDED' | 'FAILED';
  vm_id: string;
  error_code: string | null;
  error_message: string | null;
  created_at: string;
  updated_at: string;
}

export interface ExecutionLog {
  id: number;
  execution_id: string;
  sequence: number;
  severity: string;
  message: string;
  emitted_at: string;
}

// ============================================
// Request/Response Models
// ============================================

export interface CreateTenantRequest {
  slug: string;
  name: string;
  primary_region?: string;
  data_retention_days?: number;
}

export interface CreateSiteRequest {
  name: string;
  external_key?: string;
  location_country_code?: string;
}

export interface APIKey {
  id: string;
  tenant_id: string;
  name: string;
  created_at: string;
  expires_at: string | null;
  last_used_at: string | null;
}

export interface CreateAPIKeyRequest {
  name?: string;
  expires_in_seconds?: number;
}

export interface CreateAPIKeyResponse {
  id: string;
  tenant_id: string;
  name: string;
  api_key: string;
  expires_at: string;
}

export interface IssueEnrollmentTokenRequest {
  site_id: string;
  expires_in_seconds?: number;
}

export interface IssueEnrollmentTokenResponse {
  token_id: string;
  site_id: string;
  token: string;
  expires_at: string;
  one_time: boolean;
}

export interface EnrollmentToken {
  id: string;
  site_id: string;
  site_name: string;
  created_at: string;
  expires_at: string;
  consumed: boolean;
  consumed_at: string | null;
  consumed_by_agent_id: string | null;
}

export interface ApplyPlanRequest {
  idempotency_key: string;
  client_request_id?: string;
  actions: PlanAction[];
}

export interface ApplyPlanResponse {
  plan_id: string;
  plan_version: number;
  plan_status: string;
  deduplicated: boolean;
  executions: Execution[];
}

// ============================================
// Action & Plan Models
// ============================================

export type PlanOperation = 'CREATE' | 'START' | 'STOP' | 'DELETE';

export interface PlanAction {
  operation_id?: string;
  operation: PlanOperation;
  vm_id?: string;
  name?: string;
  vcpu_count?: number;
  memory_mib?: number;
}

// ============================================
// List Response Wrappers
// ============================================

export interface ListSitesResponse {
  sites: Site[];
}

export interface ListHostsResponse {
  hosts: Host[];
}

export interface ListVMsResponse {
  vms: MicroVM[];
}

export interface ListExecutionLogsResponse {
  logs: ExecutionLog[];
}

// ============================================
// Enrollment & Agent Models
// ============================================

export interface EnrollRequest {
  enrollment_token: string;
  agent_version?: string;
  hostname: string;
  os?: string;
  arch?: string;
  kernel_version?: string;
  csr_pem: string;
}

export interface EnrollResponse {
  tenant_id: string;
  site_id: string;
  host_id: string;
  agent_id: string;
  client_certificate_pem: string;
  ca_certificate_pem: string;
  refresh_token: string;
  heartbeat_endpoint: string;
  heartbeat_interval_sec: number;
}

export interface HeartbeatRequest {
  agent_id: string;
  heartbeat_seq?: number;
  agent_version?: string;
  os?: string;
  arch?: string;
  kernel_version?: string;
  hostname?: string;
  cpu_cores_total?: number;
  memory_bytes_total?: number;
  storage_bytes_total?: number;
  kvm_available?: boolean;
  cloud_hypervisor_available?: boolean;
  microvms?: MicroVM[];
  execution_updates?: ExecutionUpdate[];
}

export interface ExecutionUpdate {
  execution_id: string;
  state: 'PENDING' | 'IN_PROGRESS' | 'SUCCEEDED' | 'FAILED';
  error_code?: string;
  error_message?: string;
  updated_at?: string;
}

export interface IngestLogsRequest {
  agent_id: string;
  entries?: LogEntry[];
}

export interface LogEntry {
  execution_id: string;
  sequence: number;
  severity: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
  message: string;
  emitted_at: string;
}

// ============================================
// Error Types
// ============================================

export interface APIError {
  message: string;
  statusCode?: number;
  code?: string;
}

// Re-export TableColumn type for convenience
export type { Column as TableColumn } from '@/components/common/Table';
