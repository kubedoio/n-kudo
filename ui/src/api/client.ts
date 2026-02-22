import { Site, EnrollmentToken, Host, MicroVM, Execution, LogEntry, Template, CreateTemplateRequest, DeployTemplateRequest } from './types';

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

    // Templates
    async listTemplates(): Promise<{ templates: Template[] }> {
        return this.request(`/tenants/${this.config.tenantId}/templates`);
    }

    async getTemplate(templateId: string): Promise<Template> {
        return this.request(`/tenants/${this.config.tenantId}/templates/${templateId}`);
    }

    async createTemplate(req: CreateTemplateRequest): Promise<Template> {
        return this.request(`/tenants/${this.config.tenantId}/templates`, {
            method: 'POST',
            body: JSON.stringify(req),
        });
    }

    async updateTemplate(templateId: string, req: CreateTemplateRequest): Promise<Template> {
        return this.request(`/tenants/${this.config.tenantId}/templates/${templateId}`, {
            method: 'PUT',
            body: JSON.stringify(req),
        });
    }

    async deleteTemplate(templateId: string): Promise<void> {
        return this.request(`/tenants/${this.config.tenantId}/templates/${templateId}`, {
            method: 'DELETE',
        });
    }

    // Deployment
    async deployTemplate(req: DeployTemplateRequest): Promise<{ plan_id: string; executions: any[] }> {
        return this.request(`/tenants/${this.config.tenantId}/templates/deploy`, {
            method: 'POST',
            body: JSON.stringify(req),
        });
    }

    // Deploy template actions to a specific site
    async deployTemplateToSite(siteId: string, actions: any[]): Promise<{ plan: any; executions: any[] }> {
        return this.request(`/sites/${siteId}/plans`, {
            method: 'POST',
            body: JSON.stringify({
                actions,
                idempotency_key: `designer-${Date.now()}`
            }),
        });
    }

    // Export template as plan actions
    async previewTemplateDeployment(templateId: string, siteId: string): Promise<{ actions: any[] }> {
        return this.request(`/tenants/${this.config.tenantId}/templates/${templateId}/preview?site_id=${siteId}`);
    }
}
