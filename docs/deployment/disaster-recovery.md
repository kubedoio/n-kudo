# Disaster Recovery Guide

This guide covers backup, restore, and disaster recovery procedures for N-Kudo control plane.

## Table of Contents

- [Overview](#overview)
- [RPO and RTO Targets](#rpo-and-rto-targets)
- [Backup Strategy](#backup-strategy)
- [Automated Backups](#automated-backups)
- [Manual Backups](#manual-backups)
- [Restore Procedures](#restore-procedures)
- [Disaster Recovery Scenarios](#disaster-recovery-scenarios)
- [Failover Procedures](#failover-procedures)
- [Best Practices](#best-practices)

## Overview

N-Kudo provides comprehensive backup and disaster recovery capabilities:

- **Database Backups**: Full PostgreSQL database dumps with compression and encryption
- **Tenant State Export**: JSON-based export of tenant-specific data
- **Automated Scheduling**: Cron-based backup scheduler
- **S3 Integration**: Automatic upload to S3-compatible storage
- **Retention Management**: Automatic cleanup of old backups

## RPO and RTO Targets

| Metric | Target | Description |
|--------|--------|-------------|
| **RPO** (Recovery Point Objective) | 24 hours | Maximum data loss window |
| **RTO** (Recovery Time Objective) | 4 hours | Maximum downtime for full recovery |

These targets assume:
- Daily automated backups at 2 AM UTC
- Backup storage on durable storage (S3 or local RAID)
- Documented restore procedures

## Backup Strategy

### Backup Components

1. **PostgreSQL Database**: Complete database dump including:
   - Tenant configurations
   - Sites and agents
   - VM states and plans
   - Audit logs
   - Certificate history

2. **PKI Infrastructure** (if not using external CA):
   - CA private keys
   - Certificate files
   - CRL data

3. **Configuration**:
   - Environment variables
   - TLS certificates
   - Docker compose files

### Backup Storage

- **Primary**: Local backup directory (`/var/backups/n-kudo`)
- **Secondary**: S3-compatible object storage
- **Retention**: 30 days by default (configurable)

## Automated Backups

### Backup Scheduler

The backup scheduler runs as a separate service:

```bash
# Build the scheduler
go build -o bin/backup-scheduler ./cmd/backup-scheduler

# Run with environment variables
DATABASE_URL="postgres://..." \
BACKUP_SCHEDULE="0 2 * * *" \
BACKUP_DIR="/var/backups/n-kudo" \
  ./bin/backup-scheduler
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/nkudo` | PostgreSQL connection string |
| `BACKUP_SCHEDULE` | `0 2 * * *` | Cron schedule expression |
| `BACKUP_DIR` | `/var/backups/n-kudo` | Local backup directory |
| `BACKUP_RETENTION_DAYS` | `30` | Days to keep backups |
| `BACKUP_COMPRESS` | `true` | Enable gzip compression |
| `BACKUP_ENCRYPT` | `false` | Enable encryption |
| `BACKUP_ENCRYPTION_KEY` | - | Encryption passphrase |
| `BACKUP_S3_BUCKET` | - | S3 bucket name |
| `BACKUP_S3_ENDPOINT` | - | S3 endpoint URL (for MinIO) |
| `BACKUP_TIMEOUT_MINUTES` | `60` | Backup timeout |

### Docker Compose Integration

Add to `docker-compose.yml`:

```yaml
backup-scheduler:
  build:
    context: .
    dockerfile: deployments/Dockerfile.backup
  environment:
    - DATABASE_URL=postgres://postgres:postgres@postgres:5432/nkudo?sslmode=disable
    - BACKUP_SCHEDULE=0 2 * * *
    - BACKUP_DIR=/backups
    - BACKUP_RETENTION_DAYS=30
    - BACKUP_S3_BUCKET=my-backups
    - AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
    - AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
  volumes:
    - backup-data:/backups
  depends_on:
    - postgres
```

## Manual Backups

### Using the Backup Script

```bash
# Simple backup
./scripts/backup.sh

# Backup with specific directory
./scripts/backup.sh -d /mnt/backups/n-kudo

# Encrypted backup
BACKUP_ENCRYPTION_KEY="your-secret-key" ./scripts/backup.sh --encrypt

# Backup to S3
./scripts/backup.sh --s3-bucket "my-backups" --s3-endpoint "https://s3.amazonaws.com"

# Custom retention
./scripts/backup.sh -k 14  # Keep 14 days
```

### Using pg_dump Directly

```bash
# Basic dump
pg_dump -d "postgres://user:pass@host/db" -F c -f backup.dump

# With compression
pg_dump -d "postgres://user:pass@host/db" | gzip > backup.sql.gz

# Specific tables
pg_dump -d "postgres://user:pass@host/db" --table 'public.tenants' --table 'public.sites'
```

## Restore Procedures

### Automated Restore Script

The restore script provides an interactive restore process:

```bash
# Restore with confirmation
./scripts/restore.sh /backups/backup_20240101_120000.sql.gz

# Restore encrypted backup
./scripts/restore.sh -k "your-secret-key" /backups/backup_20240101_120000.sql.gz.enc

# Force restore without confirmation
./scripts/restore.sh -f /backups/backup_20240101_120000.sql
```

### Manual Restore Steps

1. **Stop the control plane** (to prevent writes during restore):
```bash
docker compose stop control-plane
```

2. **Restore the database**:
```bash
# Plain SQL file
psql -d "postgres://user:pass@host/db" -f backup.sql

# Compressed file
gunzip -c backup.sql.gz | psql -d "postgres://user:pass@host/db"

# Encrypted and compressed
openssl enc -aes-256-cbc -d -in backup.sql.gz.enc -out - -pass pass:"key" | \
  gunzip | psql -d "postgres://user:pass@host/db"
```

3. **Verify restore**:
```bash
# Check tenant count
psql -d "postgres://user:pass@host/db" -c "SELECT COUNT(*) FROM tenants;"

# Check agent count
psql -d "postgres://user:pass@host/db" -c "SELECT COUNT(*) FROM agents;"
```

4. **Start the control plane**:
```bash
docker compose start control-plane
```

### Verification

Use the verify script to check backup integrity:

```bash
# Verify backup file
./scripts/verify-backup.sh /backups/backup_20240101_120000.sql.gz

# Verbose verification
./scripts/verify-backup.sh -v /backups/backup_20240101_120000.sql
```

## Disaster Recovery Scenarios

### Scenario 1: Database Corruption

**Symptoms**: Database errors, data inconsistencies

**Recovery Steps**:
1. Identify the last good backup
2. Stop the control plane
3. Restore from backup
4. Verify data integrity
5. Restart services

### Scenario 2: Complete Server Failure

**Symptoms**: Server not responding, hardware failure

**Recovery Steps**:
1. Provision new server
2. Install N-Kudo (use original configuration)
3. Restore database from S3 or backup storage
4. Restore TLS certificates
5. Verify agent connectivity
6. Update DNS if needed

### Scenario 3: Accidental Data Deletion

**Symptoms**: Missing tenants, sites, or VMs

**Recovery Steps**:
1. Determine scope of deletion
2. If recent, check if point-in-time recovery is available
3. For complete restore, follow standard restore procedure
4. For partial restore, export/import specific tenant data

### Scenario 4: Region Outage

**Symptoms**: Entire region unavailable

**Recovery Steps**:
1. Activate DR site (if configured)
2. Update agent endpoints to DR control plane
3. Restore latest backup to DR database
4. Monitor agent reconnection

## Failover Procedures

### Active-Passive Failover

For setups with a standby control plane:

1. **Promote Standby**:
```bash
# On standby server
docker compose up -d
./scripts/restore.sh -f /backups/latest-backup.sql.gz
```

2. **Update Agents**:
```bash
# Update agent configuration to point to new control plane
# This requires agent restart with new configuration
```

3. **Update DNS/Load Balancer**:
```bash
# Point DNS to standby IP
# Or update load balancer configuration
```

### DNS Failover

Using health-checked DNS records:

1. Configure health checks on primary endpoint
2. Set up DNS failover to secondary IP
3. Ensure TTL is low (< 5 minutes) for quick failover

## Best Practices

### Backup Strategy

1. **Regular Testing**: Test restore procedures monthly
2. **Offsite Storage**: Always copy backups to offsite/S3 storage
3. **Encryption**: Enable encryption for production backups
4. **Monitoring**: Alert on backup failures
5. **Documentation**: Keep DR documentation updated

### Security

1. **Encryption Keys**: Store separately from backups
2. **Access Control**: Restrict backup directory access
3. **Audit**: Log all backup/restore operations
4. **Rotation**: Rotate encryption keys periodically

### Monitoring

Set up alerts for:
- Backup job failures
- Low disk space on backup storage
- S3 upload failures
- Backup file size anomalies

Example Prometheus alerting rules:

```yaml
groups:
  - name: backup
    rules:
      - alert: BackupJobFailed
        expr: time() - backup_last_success_timestamp > 90000  # 25 hours
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Backup job has not succeeded in 25 hours"
      
      - alert: BackupFileTooSmall
        expr: backup_file_size_bytes < 1048576  # 1 MB
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Latest backup file is suspiciously small"
```

### Testing Checklist

Run through this checklist quarterly:

- [ ] Verify backup files exist and are readable
- [ ] Test restore to staging environment
- [ ] Verify data integrity after restore
- [ ] Test encrypted backup restore
- [ ] Verify S3 upload/download
- [ ] Test failover procedures
- [ ] Update DR documentation

## Emergency Contacts

Document your team's emergency contacts and escalation procedures:

| Role | Contact | Escalation |
|------|---------|------------|
| On-call Engineer | ... | ... |
| Database Admin | ... | ... |
| Infrastructure Lead | ... | ... |
| Product Owner | ... | ... |

---

## Quick Reference

**Backup**:
```bash
./scripts/backup.sh --encrypt --s3-bucket "my-backups"
```

**Restore**:
```bash
./scripts/restore.sh -f /backups/backup_YYYYMMDD_HHMMSS.sql.gz
```

**Verify**:
```bash
./scripts/verify-backup.sh /backups/backup_YYYYMMDD_HHMMSS.sql.gz
```

**List Backups**:
```bash
ls -la /var/backups/n-kudo/
```

**Check Scheduler**:
```bash
docker logs n-kudo-backup-scheduler-1
```
