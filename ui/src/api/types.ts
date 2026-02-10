import { z } from 'zod';

export const SiteSchema = z.object({
    id: z.string(),
    tenant_id: z.string(),
    name: z.string(),
    connectivity_state: z.string(),
    last_heartbeat_at: z.string().optional().nullable(),
    created_at: z.string(),
});

export type Site = z.infer<typeof SiteSchema>;

export const HostFactsSchema = z.object({
    hostname: z.string(),
    os_type: z.string(),
    arch: z.string(),
    cpu_cores: z.number(),
    memory_total: z.number(),
    memory_free: z.number(),
    kvm_available: z.boolean(),
    cloud_hypervisor_available: z.boolean(),
});

export const HostSchema = z.object({
    id: z.string(),
    tenant_id: z.string(),
    site_id: z.string(),
    hostname: z.string(),
    last_facts: HostFactsSchema.optional(),
    last_facts_at: z.string().optional().nullable(),
    updated_at: z.string(),
});

export type Host = z.infer<typeof HostSchema>;

export const MicroVMSchema = z.object({
    id: z.string(),
    tenant_id: z.string(),
    site_id: z.string(),
    host_id: z.string().optional().nullable(),
    name: z.string(),
    state: z.string(),
    vcpu_count: z.number(),
    memory_mib: z.number(),
    last_transition_at: z.string().optional().nullable(),
    updated_at: z.string(),
});

export type MicroVM = z.infer<typeof MicroVMSchema>;

export const ExecutionSchema = z.object({
    id: z.string(),
    tenant_id: z.string(),
    site_id: z.string(),
    plan_id: z.string(),
    operation_id: z.string(),
    operation_type: z.string(),
    state: z.string(),
    error_code: z.string().optional().nullable(),
    error_message: z.string().optional().nullable(),
    started_at: z.string().optional().nullable(),
    completed_at: z.string().optional().nullable(),
    updated_at: z.string(),
});

export type Execution = z.infer<typeof ExecutionSchema>;

export const LogEntrySchema = z.object({
    tenant_id: z.string(),
    site_id: z.string(),
    agent_id: z.string(),
    execution_id: z.string(),
    action_id: z.string(),
    level: z.string(),
    message: z.string(),
    timestamp: z.string(),
});

export type LogEntry = z.infer<typeof LogEntrySchema>;

export const EnrollmentTokenSchema = z.object({
    token_id: z.string(),
    site_id: z.string(),
    token: z.string(),
    expires_at: z.string(),
    one_time: z.boolean(),
});

export type EnrollmentToken = z.infer<typeof EnrollmentTokenSchema>;
