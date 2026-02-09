# Phase 5 Task 6: Backup & Disaster Recovery

## Task Description
Implement database backup automation, state export/import, and multi-region setup foundation.

## Requirements

### 1. Automated Database Backups

**File:** `internal/controlplane/backup/manager.go`

```go
package backup

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "time"
)

// Manager handles database backups
type Manager struct {
    config Config
}

type Config struct {
    DatabaseURL      string
    BackupDir        string
    RetentionDays    int
    Schedule         string // cron expression
    Compress         bool
    Encrypt          bool
    EncryptionKey    string
    S3Bucket         string
    S3Endpoint       string
}

// Backup performs a database backup
func (m *Manager) Backup(ctx context.Context) (*BackupResult, error) {
    timestamp := time.Now().UTC().Format("20060102_150405")
    filename := fmt.Sprintf("nkudo_backup_%s.sql", timestamp)
    
    if m.config.Compress {
        filename += ".gz"
    }
    
    filepath := filepath.Join(m.config.BackupDir, filename)
    
    // Create pg_dump command
    cmd := exec.CommandContext(ctx, "pg_dump", m.config.DatabaseURL)
    
    if m.config.Compress {
        cmd = exec.CommandContext(ctx, "sh", "-c", 
            fmt.Sprintf("pg_dump %s | gzip > %s", m.config.DatabaseURL, filepath))
    } else {
        cmd = exec.CommandContext(ctx, "pg_dump", "-f", filepath, m.config.DatabaseURL)
    }
    
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("backup failed: %w", err)
    }
    
    // Encrypt if enabled
    if m.config.Encrypt {
        encryptedPath := filepath + ".enc"
        if err := m.encryptFile(filepath, encryptedPath); err != nil {
            return nil, fmt.Errorf("encryption failed: %w", err)
        }
        os.Remove(filepath)
        filepath = encryptedPath
    }
    
    // Upload to S3 if configured
    if m.config.S3Bucket != "" {
        if err := m.uploadToS3(ctx, filepath); err != nil {
            return nil, fmt.Errorf("s3 upload failed: %w", err)
        }
    }
    
    return &BackupResult{
        Path:      filepath,
        Size:      getFileSize(filepath),
        CreatedAt: time.Now().UTC(),
    }, nil
}

// Restore restores database from backup
func (m *Manager) Restore(ctx context.Context, backupPath string) error {
    // Decrypt if needed
    if filepath.Ext(backupPath) == ".enc" {
        decryptedPath := backupPath[:len(backupPath)-4]
        if err := m.decryptFile(backupPath, decryptedPath); err != nil {
            return fmt.Errorf("decryption failed: %w", err)
        }
        backupPath = decryptedPath
        defer os.Remove(decryptedPath)
    }
    
    // Decompress if needed
    if filepath.Ext(backupPath) == ".gz" {
        cmd := exec.CommandContext(ctx, "sh", "-c",
            fmt.Sprintf("gunzip < %s | psql %s", backupPath, m.config.DatabaseURL))
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("restore failed: %w", err)
        }
    } else {
        cmd := exec.CommandContext(ctx, "psql", "-f", backupPath, m.config.DatabaseURL)
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("restore failed: %w", err)
        }
    }
    
    return nil
}

// CleanupOldBackups removes backups older than retention period
func (m *Manager) CleanupOldBackups(ctx context.Context) error {
    cutoff := time.Now().UTC().AddDate(0, 0, -m.config.RetentionDays)
    
    entries, err := os.ReadDir(m.config.BackupDir)
    if err != nil {
        return err
    }
    
    for _, entry := range entries {
        info, err := entry.Info()
        if err != nil {
            continue
        }
        
        if info.ModTime().Before(cutoff) {
            os.Remove(filepath.Join(m.config.BackupDir, entry.Name()))
        }
    }
    
    return nil
}
```

### 2. Backup Scheduler

**File:** `cmd/backup-scheduler/main.go`

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/kubedoio/n-kudo/internal/controlplane/backup"
    "github.com/robfig/cron/v3"
)

func main() {
    config := backup.Config{
        DatabaseURL:   os.Getenv("DATABASE_URL"),
        BackupDir:     getEnv("BACKUP_DIR", "/var/backups/nkudo"),
        RetentionDays: getEnvInt("BACKUP_RETENTION_DAYS", 30),
        Schedule:      getEnv("BACKUP_SCHEDULE", "0 2 * * *"), // Daily at 2 AM
        Compress:      true,
        Encrypt:       os.Getenv("BACKUP_ENCRYPT_KEY") != "",
        EncryptionKey: os.Getenv("BACKUP_ENCRYPT_KEY"),
        S3Bucket:      os.Getenv("BACKUP_S3_BUCKET"),
        S3Endpoint:    os.Getenv("BACKUP_S3_ENDPOINT"),
    }
    
    manager := backup.NewManager(config)
    
    c := cron.New()
    c.AddFunc(config.Schedule, func() {
        log.Println("Starting scheduled backup...")
        
        ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
        defer cancel()
        
        result, err := manager.Backup(ctx)
        if err != nil {
            log.Printf("Backup failed: %v", err)
            return
        }
        
        log.Printf("Backup completed: %s (%d bytes)", result.Path, result.Size)
        
        // Cleanup old backups
        if err := manager.CleanupOldBackups(ctx); err != nil {
            log.Printf("Cleanup failed: %v", err)
        }
    })
    
    c.Start()
    log.Printf("Backup scheduler started (schedule: %s)", config.Schedule)
    
    // Wait for shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    <-sigCh
    
    c.Stop()
    log.Println("Backup scheduler stopped")
}
```

### 3. State Export/Import

**File:** `internal/controlplane/backup/state.go`

```go
package backup

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
)

// TenantState represents a complete tenant export
type TenantState struct {
    Version     string                 `json:"version"`
    ExportedAt  time.Time              `json:"exported_at"`
    Tenant      store.Tenant           `json:"tenant"`
    Sites       []store.Site           `json:"sites"`
    APIKeys     []store.APIKey         `json:"api_keys"`
    Agents      []AgentState           `json:"agents"`
    VMs         []store.MicroVM        `json:"vms"`
    AuditEvents []store.AuditEvent     `json:"audit_events,omitempty"`
}

type AgentState struct {
    Agent       store.Agent            `json:"agent"`
    Heartbeats  []store.Heartbeat      `json:"heartbeats,omitempty"`
}

// ExportTenant exports all data for a tenant
func (m *Manager) ExportTenant(ctx context.Context, tenantID string) (*TenantState, error) {
    state := &TenantState{
        Version:    "1.0",
        ExportedAt: time.Now().UTC(),
    }
    
    // Export tenant
    tenant, err := m.repo.GetTenant(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    state.Tenant = tenant
    
    // Export sites
    sites, err := m.repo.ListSites(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    state.Sites = sites
    
    // Export API keys
    keys, err := m.repo.ListAPIKeys(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    state.APIKeys = keys
    
    // Export agents with limited heartbeats
    agents, err := m.repo.ListAgents(ctx, tenantID)
    for _, agent := range agents {
        heartbeats, _ := m.repo.ListAgentHeartbeats(ctx, agent.ID, 100)
        state.Agents = append(state.Agents, AgentState{
            Agent:      agent,
            Heartbeats: heartbeats,
        })
    }
    
    return state, nil
}

// ImportTenant imports tenant data from state
func (m *Manager) ImportTenant(ctx context.Context, state *TenantState) error {
    // Create tenant
    if _, err := m.repo.CreateTenant(ctx, state.Tenant); err != nil {
        return fmt.Errorf("create tenant: %w", err)
    }
    
    // Import sites
    for _, site := range state.Sites {
        if _, err := m.repo.CreateSite(ctx, site); err != nil {
            return fmt.Errorf("create site: %w", err)
        }
    }
    
    // Import API keys
    for _, key := range state.APIKeys {
        if _, err := m.repo.CreateAPIKey(ctx, key); err != nil {
            return fmt.Errorf("create api key: %w", err)
        }
    }
    
    return nil
}

// SaveToFile saves state to JSON file
func (s *TenantState) SaveToFile(path string) error {
    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()
    
    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    return encoder.Encode(s)
}

// LoadFromFile loads state from JSON file
func LoadFromFile(path string) (*TenantState, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var state TenantState
    if err := json.NewDecoder(file).Decode(&state); err != nil {
        return nil, err
    }
    return &state, nil
}
```

### 4. Disaster Recovery Scripts

**File:** `scripts/backup.sh`

```bash
#!/bin/bash
set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/nkudo}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
S3_BUCKET="${S3_BUCKET:-}"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="nkudo_backup_${TIMESTAMP}.sql.gz"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_FILE}"

# Create backup directory
mkdir -p "${BACKUP_DIR}"

echo "Starting backup at ${TIMESTAMP}..."

# Perform backup
pg_dump "${DATABASE_URL}" | gzip > "${BACKUP_PATH}"

# Encrypt if key provided
if [[ -n "${BACKUP_ENCRYPT_KEY}" ]]; then
    echo "Encrypting backup..."
    openssl enc -aes-256-cbc -salt -in "${BACKUP_PATH}" -out "${BACKUP_PATH}.enc" -k "${BACKUP_ENCRYPT_KEY}"
    rm "${BACKUP_PATH}"
    BACKUP_PATH="${BACKUP_PATH}.enc"
fi

# Upload to S3 if configured
if [[ -n "${S3_BUCKET}" ]]; then
    echo "Uploading to S3..."
    aws s3 cp "${BACKUP_PATH}" "s3://${S3_BUCKET}/"
fi

# Cleanup old backups
echo "Cleaning up old backups..."
find "${BACKUP_DIR}" -name "nkudo_backup_*" -mtime +${RETENTION_DAYS} -delete

echo "Backup completed: ${BACKUP_PATH}"
```

**File:** `scripts/restore.sh`

```bash
#!/bin/bash
set -e

# Usage: ./restore.sh <backup_file>

BACKUP_FILE="$1"

if [[ -z "${BACKUP_FILE}" ]]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

if [[ ! -f "${BACKUP_FILE}" ]]; then
    echo "Backup file not found: ${BACKUP_FILE}"
    exit 1
fi

echo "WARNING: This will restore the database from backup."
echo "Current data will be overwritten!"
read -p "Are you sure? (yes/no): " CONFIRM

if [[ "${CONFIRM}" != "yes" ]]; then
    echo "Restore cancelled."
    exit 0
fi

# Decrypt if needed
if [[ "${BACKUP_FILE}" == *.enc ]]; then
    echo "Decrypting backup..."
    openssl enc -aes-256-cbc -d -in "${BACKUP_FILE}" -out "${BACKUP_FILE%.enc}" -k "${BACKUP_ENCRYPT_KEY}"
    BACKUP_FILE="${BACKUP_FILE%.enc}"
fi

# Restore
echo "Restoring database..."
if [[ "${BACKUP_FILE}" == *.gz ]]; then
    gunzip < "${BACKUP_FILE}" | psql "${DATABASE_URL}"
else
    psql "${DATABASE_URL}" -f "${BACKUP_FILE}"
fi

echo "Restore completed!"
```

### 5. Multi-Region Foundation

**File:** `docs/deployment/multi-region.md`

Architecture for future multi-region deployment:

```
┌─────────────────┐     ┌─────────────────┐
│   eu-central    │────▶│    us-east      │
│   (Primary)     │◄────│   (Replica)     │
└─────────────────┘     └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│   PostgreSQL    │◄────│  PostgreSQL     │
│   (Primary)     │  ▲  │  (Read Replica) │
└─────────────────┘  │  └─────────────────┘
                     │
              (Streaming Replication)
```

### 6. Backup Verification

**File:** `scripts/verify-backup.sh`

```bash
#!/bin/bash
# Verify backup integrity

BACKUP_FILE="$1"

echo "Verifying backup: ${BACKUP_FILE}"

# Check file exists and has size
if [[ ! -s "${BACKUP_FILE}" ]]; then
    echo "ERROR: Backup file is empty or missing"
    exit 1
fi

# Check if gzip is valid (if compressed)
if [[ "${BACKUP_FILE}" == *.gz ]]; then
    if ! gunzip -t "${BACKUP_FILE}" 2>/dev/null; then
        echo "ERROR: Backup file is corrupted (gzip test failed)"
        exit 1
    fi
fi

echo "Backup verification passed!"
```

## Deliverables
1. `internal/controlplane/backup/manager.go` - Backup/restore manager
2. `internal/controlplane/backup/state.go` - State export/import
3. `cmd/backup-scheduler/main.go` - Backup scheduler
4. `scripts/backup.sh` - Manual backup script
5. `scripts/restore.sh` - Restore script
6. `scripts/verify-backup.sh` - Backup verification
7. `docs/deployment/disaster-recovery.md` - DR procedures
8. `docs/deployment/multi-region.md` - Multi-region architecture

## Database Migrations

**File:** `db/migrations/0005_backup_tracking.sql`

```sql
-- Track backup operations
CREATE TABLE backup_history (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL, -- 'running', 'completed', 'failed'
    path TEXT NOT NULL,
    size_bytes BIGINT,
    checksum TEXT,
    error_message TEXT
);

CREATE INDEX idx_backup_history_started ON backup_history(started_at DESC);
```

## Estimated Effort
8-10 hours
