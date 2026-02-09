import { http, HttpResponse } from 'msw'

// API base URL
const API_BASE = 'https://localhost:8443'

// Mock data
export const mockTenants = [
  { 
    id: '1', 
    name: 'Test Tenant', 
    slug: 'test', 
    primary_region: 'us-east-1', 
    data_retention_days: 30, 
    created_at: '2024-01-01T00:00:00Z' 
  },
  { 
    id: '2', 
    name: 'Another Tenant', 
    slug: 'another', 
    primary_region: 'eu-west-1', 
    data_retention_days: 60, 
    created_at: '2024-02-01T00:00:00Z' 
  }
]

export const mockSites = [
  {
    id: 'site-1',
    tenant_id: '1',
    name: 'Test Site',
    external_key: 'test-site',
    location_country_code: 'US',
    connectivity_state: 'ONLINE',
    last_heartbeat_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z'
  }
]

export const mockAPIKeys = [
  {
    id: 'key-1',
    tenant_id: '1',
    name: 'Test API Key',
    created_at: '2024-01-01T00:00:00Z',
    expires_at: null,
    last_used_at: null
  }
]

export const mockVMs = [
  {
    id: 'vm-1',
    site_id: 'site-1',
    host_id: 'host-1',
    name: 'Test VM',
    state: 'RUNNING',
    vcpu_count: 2,
    memory_mib: 4096,
    updated_at: '2024-01-01T00:00:00Z'
  }
]

export const mockHosts = [
  {
    id: 'host-1',
    hostname: 'test-host',
    cpu_cores_total: 8,
    memory_bytes_total: 17179869184,
    storage_bytes_total: 107374182400,
    kvm_available: true,
    cloud_hypervisor_available: true,
    last_facts_at: '2024-01-01T00:00:00Z',
    agent_state: 'connected'
  }
]

export const mockExecutions = [
  {
    id: 'exec-1',
    plan_id: 'plan-1',
    operation_id: 'op-1',
    operation_type: 'CREATE',
    state: 'SUCCEEDED',
    vm_id: 'vm-1',
    error_code: null,
    error_message: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  }
]

export const handlers = [
  // Health endpoint
  http.get(`${API_BASE}/healthz`, () => {
    return HttpResponse.json({ status: 'healthy' })
  }),

  // Tenants endpoints
  http.get(`${API_BASE}/tenants`, () => {
    return HttpResponse.json(mockTenants)
  }),

  http.get(`${API_BASE}/tenants/:id`, ({ params }) => {
    const tenant = mockTenants.find(t => t.id === params.id)
    if (!tenant) {
      return HttpResponse.json({ message: 'Tenant not found' }, { status: 404 })
    }
    return HttpResponse.json(tenant)
  }),

  http.post(`${API_BASE}/tenants`, async ({ request }) => {
    const data = await request.json() as { name: string; slug: string }
    const newTenant = {
      id: 'new-tenant-id',
      name: data.name,
      slug: data.slug,
      primary_region: 'us-east-1',
      data_retention_days: 30,
      created_at: new Date().toISOString()
    }
    return HttpResponse.json(newTenant, { status: 201 })
  }),

  // Sites endpoints
  http.get(`${API_BASE}/tenants/:tenantId/sites`, () => {
    return HttpResponse.json({ sites: mockSites })
  }),

  http.get(`${API_BASE}/tenants/:tenantId/sites/:siteId`, ({ params }) => {
    const site = mockSites.find(s => s.id === params.siteId)
    if (!site) {
      return HttpResponse.json({ message: 'Site not found' }, { status: 404 })
    }
    return HttpResponse.json(site)
  }),

  // API Keys endpoints
  http.get(`${API_BASE}/tenants/:tenantId/api-keys`, () => {
    return HttpResponse.json({ api_keys: mockAPIKeys })
  }),

  http.post(`${API_BASE}/tenants/:tenantId/api-keys`, async ({ request }) => {
    const data = await request.json() as { name: string }
    const newKey = {
      id: 'new-key-id',
      tenant_id: '1',
      name: data.name,
      api_key: 'nk_test_' + Math.random().toString(36).substring(2),
      expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString()
    }
    return HttpResponse.json(newKey, { status: 201 })
  }),

  http.delete(`${API_BASE}/tenants/:tenantId/api-keys/:keyId`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // VMs endpoints
  http.get(`${API_BASE}/sites/:siteId/vms`, () => {
    return HttpResponse.json({ vms: mockVMs })
  }),

  // Hosts endpoints
  http.get(`${API_BASE}/sites/:siteId/hosts`, () => {
    return HttpResponse.json({ hosts: mockHosts })
  }),

  // Executions endpoints
  http.get(`${API_BASE}/sites/:siteId/executions`, () => {
    return HttpResponse.json({ executions: mockExecutions })
  }),

  // Plans endpoints
  http.post(`${API_BASE}/sites/:siteId/plans`, async () => {
    return HttpResponse.json({
      plan_id: 'plan-1',
      plan_version: 1,
      plan_status: 'PENDING',
      deduplicated: false,
      executions: mockExecutions
    })
  }),

  // Enrollment tokens
  http.get(`${API_BASE}/tenants/:tenantId/enrollment-tokens`, () => {
    return HttpResponse.json([])
  }),
]
