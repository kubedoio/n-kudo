import { Site, EnrollmentToken, Host, MicroVM, Execution, LogEntry } from './types';

export class ApiError extends Error {
    constructor(public status: number, message: string, public body?: any) {
        super(message);
    }
}

export interface ApiConfig {
    baseUrl: string;
    tenantId: string;
    apiKey: string;
}

export class ApiClient {
    constructor(private config: ApiConfig) { }

    private async request<T>(path: string, options: RequestInit = {}): Promise<T> {
        const url = `${this.config.baseUrl}${path}`;
        const headers = new Headers(options.headers);
        headers.set('X-API-Key', this.config.apiKey);
        headers.set('Content-Type', 'application/json');

        const response = await fetch(url, { ...options, headers });

        if (!response.ok) {
            const body = await response.json().catch(() => ({}));
            throw new ApiError(response.status, body.error || response.statusText, body);
        }

        if (response.status === 204) {
            return {} as T;
        }

        return response.json();
    }

    // Health
    async healthz() {
        return this.request<{ status: string }>('/healthz');
    }

    // Sites
    async listSites(): Promise<{ sites: Site[] }> {
        return this.request(`/tenants/${this.config.tenantId}/sites`);
    }

    async createSite(name: string): Promise<Site> {
        return this.request(`/tenants/${this.config.tenantId}/sites`, {
            method: 'POST',
            body: JSON.stringify({ name }),
        });
    }

    // Enrollment
    async createEnrollmentToken(siteId: string): Promise<EnrollmentToken> {
        return this.request(`/tenants/${this.config.tenantId}/enrollment-tokens`, {
            method: 'POST',
            body: JSON.stringify({ site_id: siteId }),
        });
    }

    // Plans & Executions
    async applyPlan(siteId: string, actions: any[]): Promise<{ plan: any; executions: Execution[] }> {
        return this.request(`/sites/${siteId}/plans`, {
            method: 'POST',
            body: JSON.stringify({ actions }),
        });
    }

    async listHosts(siteId: string): Promise<{ hosts: Host[] }> {
        return this.request(`/sites/${siteId}/hosts`);
    }

    async listVMs(siteId: string): Promise<{ vms: MicroVM[] }> {
        return this.request(`/sites/${siteId}/vms`);
    }

    async listExecutionLogs(executionId: string): Promise<{ logs: LogEntry[] }> {
        return this.request(`/executions/${executionId}/logs`);
    }
}
