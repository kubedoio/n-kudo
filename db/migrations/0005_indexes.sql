-- Performance indexes for frequently queried tables
-- These indexes improve query performance for common access patterns

-- Index for listing agents by tenant and site (used in authorization checks)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_agents_tenant_site 
ON agents(tenant_id, site_id);

-- Index for fetching latest heartbeats (used by offline sweeper and status queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_heartbeats_agent_ingested 
ON heartbeats(agent_id, ingested_at DESC);

-- Index for fetching executions by plan (used in plan status rollup)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_executions_plan_id 
ON executions(plan_id);

-- Index for listing audit events by tenant with time-based filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_events_tenant_occurred 
ON audit_events(tenant_id, occurred_at DESC);

-- Index for fetching execution logs by execution with time ordering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_execution_logs_execution_emitted 
ON execution_logs(execution_id, emitted_at DESC);
