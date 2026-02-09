# Phase 5 Task 1: Documentation & Deployment

## Task Description
Polish Docker Compose setup, create deployment guides, and operational runbooks.

## Requirements

### 1. Docker Compose Production Polish

**File:** `docker-compose.yml`

Current gaps to address:
- Health checks for all services
- Resource limits (CPU/memory)
- Restart policies
- Log rotation
- Network security
- Volume backups

```yaml
# Add to each service
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s

deploy:
  resources:
    limits:
      cpus: '1.0'
      memory: 512M
    reservations:
      cpus: '0.5'
      memory: 256M

logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

### 2. Environment Configuration Template

**File:** `.env.production.example`

```bash
# Database
DATABASE_URL=postgres://nkudo:changeme@postgres:5432/nkudo?sslmode=require
DB_MAX_CONNECTIONS=25
DB_CONN_MAX_LIFETIME=30m

# Security
ADMIN_KEY=change-this-to-32-char-secret-key
ENCRYPTION_KEY=change-this-to-32-char-aes-key
AGENT_CERT_TTL=24h

# Server
SERVER_BIND=:8443
CORS_ALLOWED_ORIGINS=https://app.nkudo.io
RATE_LIMIT_ENABLED=true

# Features
METRICS_ENABLED=true
LOG_LEVEL=info
LOG_FORMAT=json

# External
NETBIRD_MANAGEMENT_URL=
S3_ENDPOINT=
S3_BUCKET=
```

### 3. Deployment Guides

**File:** `docs/deployment/docker-compose.md`

```markdown
# Docker Compose Deployment Guide

## Prerequisites
- Docker 24.0+
- Docker Compose 2.20+
- 4GB RAM minimum
- 20GB disk space

## Quick Start
1. Copy environment file: `cp .env.production.example .env`
2. Generate secrets: `./scripts/generate-secrets.sh`
3. Start services: `docker compose up -d`
4. Verify health: `./scripts/health-check.sh`

## Production Checklist
- [ ] Change default passwords
- [ ] Enable TLS certificates
- [ ] Configure backups
- [ ] Set up monitoring
- [ ] Configure log aggregation
```

**File:** `docs/deployment/kubernetes.md`

Basic k8s manifests:
- Namespace
- ConfigMap for configuration
- Secret for sensitive data
- Deployment for control-plane
- Service for API exposure
- Ingress with TLS
- StatefulSet for PostgreSQL
- PersistentVolumeClaims

### 4. Operational Runbooks

**File:** `docs/runbooks/agent-troubleshooting.md`

Common issues and resolution steps:
- Agent won't enroll
- Certificate expired
- Heartbeat failures
- VM won't start
- Network connectivity issues

**File:** `docs/runbooks/restore-from-backup.md`

Step-by-step database restore procedures.

**File:** `docs/runbooks/security-incident.md`

Incident response for:
- Compromised agent
- Stolen API key
- Unauthorized access attempt

### 5. API Documentation

**File:** `docs/api/openapi.yaml`

Complete OpenAPI 3.0 spec for all endpoints.

### 6. README Updates

Update main README with:
- Architecture diagram
- Quick start guide
- Feature checklist
- Security considerations

## Deliverables
1. Updated `docker-compose.yml` with production settings
2. `.env.production.example` template
3. `docs/deployment/` with Docker Compose and K8s guides
4. `docs/runbooks/` with operational procedures
5. `docs/api/openapi.yaml` complete API spec
6. Updated `README.md`

## Estimated Effort
8-10 hours
