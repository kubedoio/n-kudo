#!/usr/bin/env bash
#
# N-Kudo Backup Script
# Performs manual database backup with optional encryption and S3 upload
#

set -euo pipefail

# Default configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/n-kudo}"
DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/nkudo?sslmode=disable}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
COMPRESS="${BACKUP_COMPRESS:-true}"
ENCRYPT="${BACKUP_ENCRYPT:-false}"
ENCRYPTION_KEY="${BACKUP_ENCRYPTION_KEY:-}"
S3_BUCKET="${BACKUP_S3_BUCKET:-}"
S3_ENDPOINT="${BACKUP_S3_ENDPOINT:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Perform manual database backup for N-Kudo control plane.

OPTIONS:
    -d, --dir DIR           Backup directory (default: $BACKUP_DIR)
    -k, --keep-days DAYS    Retention days for cleanup (default: $RETENTION_DAYS)
    --no-compress           Disable compression
    --encrypt               Enable encryption (requires BACKUP_ENCRYPTION_KEY)
    --s3-bucket BUCKET      S3 bucket for upload
    --s3-endpoint URL       S3 endpoint URL
    -h, --help              Show this help message

ENVIRONMENT VARIABLES:
    DATABASE_URL            PostgreSQL connection string
    BACKUP_ENCRYPTION_KEY   Encryption key (required if --encrypt is used)
    BACKUP_S3_BUCKET        S3 bucket name
    BACKUP_S3_ENDPOINT      S3 endpoint URL (optional, for MinIO)

EXAMPLES:
    # Simple backup
    $0

    # Backup with encryption
    BACKUP_ENCRYPTION_KEY="my-secret-key" $0 --encrypt

    # Backup to S3
    $0 --s3-bucket "my-backups" --s3-endpoint "https://s3.amazonaws.com"

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        -k|--keep-days)
            RETENTION_DAYS="$2"
            shift 2
            ;;
        --no-compress)
            COMPRESS="false"
            shift
            ;;
        --encrypt)
            ENCRYPT="true"
            shift
            ;;
        --s3-bucket)
            S3_BUCKET="$2"
            shift 2
            ;;
        --s3-endpoint)
            S3_ENDPOINT="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate configuration
if [[ "$ENCRYPT" == "true" && -z "$ENCRYPTION_KEY" ]]; then
    log_error "Encryption enabled but BACKUP_ENCRYPTION_KEY is not set"
    exit 1
fi

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Generate backup filename
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="backup_${TIMESTAMP}.sql"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_NAME}"

log_info "Starting backup..."
log_info "Database: ${DATABASE_URL%%://*}"  # Log only the scheme, hide credentials
log_info "Backup path: $BACKUP_PATH"

# Perform database dump
log_info "Running pg_dump..."
if ! pg_dump -d "$DATABASE_URL" -F p -f "$BACKUP_PATH"; then
    log_error "pg_dump failed"
    rm -f "$BACKUP_PATH"
    exit 1
fi

log_info "Database dump completed: $(du -h "$BACKUP_PATH" | cut -f1)"

# Compress if enabled
if [[ "$COMPRESS" == "true" ]]; then
    log_info "Compressing backup..."
    if gzip -c "$BACKUP_PATH" > "${BACKUP_PATH}.gz"; then
        rm -f "$BACKUP_PATH"
        BACKUP_PATH="${BACKUP_PATH}.gz"
        log_info "Compression completed: $(du -h "$BACKUP_PATH" | cut -f1)"
    else
        log_warn "Compression failed, keeping uncompressed backup"
    fi
fi

# Encrypt if enabled
if [[ "$ENCRYPT" == "true" ]]; then
    log_info "Encrypting backup..."
    if openssl enc -aes-256-cbc -salt \
        -in "$BACKUP_PATH" \
        -out "${BACKUP_PATH}.enc" \
        -pass "pass:$ENCRYPTION_KEY"; then
        rm -f "$BACKUP_PATH"
        BACKUP_PATH="${BACKUP_PATH}.enc"
        log_info "Encryption completed"
    else
        log_error "Encryption failed"
        rm -f "$BACKUP_PATH"
        exit 1
    fi
fi

# Upload to S3 if configured
if [[ -n "$S3_BUCKET" ]]; then
    log_info "Uploading to S3 bucket: $S3_BUCKET"
    
    AWS_ARGS=("s3" "cp" "$BACKUP_PATH" "s3://${S3_BUCKET}/")
    if [[ -n "$S3_ENDPOINT" ]]; then
        AWS_ARGS+=("--endpoint-url" "$S3_ENDPOINT")
    fi
    
    if aws "${AWS_ARGS[@]}"; then
        log_info "S3 upload completed"
        
        # Optionally remove local backup after successful S3 upload
        # rm -f "$BACKUP_PATH"
    else
        log_warn "S3 upload failed, keeping local backup"
    fi
fi

log_info "Backup completed: $BACKUP_PATH"

# Cleanup old backups
if [[ "$RETENTION_DAYS" -gt 0 ]]; then
    log_info "Cleaning up backups older than $RETENTION_DAYS days..."
    
    DELETED=0
    CUTOFF_DATE=$(date -d "-$RETENTION_DAYS days" +%Y%m%d 2>/dev/null || date -v-${RETENTION_DAYS}d +%Y%m%d)
    
    while IFS= read -r file; do
        if [[ -f "$file" ]]; then
            rm -f "$file"
            ((DELETED++)) || true
            log_info "Deleted: $(basename "$file")"
        fi
    done < <(find "$BACKUP_DIR" -name "backup_*.sql*" -type f -mtime +$RETENTION_DAYS 2>/dev/null)
    
    log_info "Cleanup completed: $DELETED old backup(s) removed"
fi

# Print summary
log_info "Backup Summary:"
echo "  - File: $BACKUP_PATH"
echo "  - Size: $(du -h "$BACKUP_PATH" 2>/dev/null | cut -f1 || echo 'unknown')"
echo "  - Compressed: $COMPRESS"
echo "  - Encrypted: $ENCRYPT"
echo "  - S3 Upload: $([[ -n "$S3_BUCKET" ]] && echo 'yes' || echo 'no')"

exit 0
