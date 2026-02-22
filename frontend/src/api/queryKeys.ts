export const queryKeys = {
    health: ['health'] as const,
    sites: (tenantId: string) => ['sites', tenantId] as const,
    hosts: (siteId: string) => ['hosts', siteId] as const,
    vms: (siteId: string) => ['vms', siteId] as const,
    executions: (siteId: string) => ['executions', siteId] as const,
    executionLogs: (executionId: string) => ['executionLogs', executionId] as const,
}
