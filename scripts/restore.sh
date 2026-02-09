#!/usr/bin/env bash
#
# N-Kudo Restore Script
# Interactive database restore with decryption and decompression handling
#

set -euo pipefail

# Default configuration
DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/nkudo?sslmode=disable}"
ENCRYPTION_KEY="${BACKUP_ENCRYPTION_KEY:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_prompt() {
    echo -e "${BLUE}[PROMPT]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS] <backup-file>

Restore N-Kudo database from backup file.

OPTIONS:
    -f, --force             Skip confirmation prompt
    -k, --key KEY           Encryption key (or set BACKUP_ENCRYPTION_KEY)
    -d, --database URL      Database URL (default: from DATABASE_URL env)
    -h, --help              Show this help message

ARGUMENTS:
    backup-file             Path to backup file (.sql, .sql.gz, or .sql.gz.enc)

EXAMPLES:
    # Restore with confirmation
    $0 /backups/backup_20240101_120000.sql

    # Restore encrypted backup
    $0 -k "my-secret-key" /backups/backup_20240101_120000.sql.gz.enc

    # Force restore without confirmation
    $0 -f /backups/backup_20240101_120000.sql.gz

EOF
}

# Parse arguments
FORCE=false
BACKUP_FILE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--force)
            FORCE=true
            shift
            ;;
        -k|--key)
            ENCRYPTION_KEY="$2"
            shift 2
            ;;
        -d|--database)
            DATABASE_URL="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            if [[ -z "$BACKUP_FILE" ]]; then
                BACKUP_FILE="$1"
            else
                log_error "Multiple backup files specified"
                exit 1
            fi
            shift
            ;;
    esac
done

# Validate backup file
if [[ -z "$BACKUP_FILE" ]]; then
    log_error "No backup file specified"
    usage
    exit 1
fi

if [[ ! -f "$BACKUP_FILE" ]]; then
    log_error "Backup file not found: $BACKUP_FILE"
    exit 1
fi

# Detect backup type
IS_ENCRYPTED=false
IS_COMPRESSED=false

if [[ "$BACKUP_FILE" == *.enc ]]; then
    IS_ENCRYPTED=true
fi
if [[ "$BACKUP_FILE" == *.gz ]]; then
    IS_COMPRESSED=true
fi

# Check for encryption key if needed
if [[ "$IS_ENCRYPTED" == true && -z "$ENCRYPTION_KEY" ]]; then
    log_prompt "Enter encryption key:"
    read -s ENCRYPTION_KEY
    echo
    if [[ -z "$ENCRYPTION_KEY" ]]; then
        log_error "Encryption key is required for encrypted backups"
        exit 1
    fi
fi

# Display restore information
echo ""
echo "========================================"
echo "       N-Kudo Database Restore"
echo "========================================"
echo ""
echo "Backup file:    $BACKUP_FILE"
echo "Database:       ${DATABASE_URL%%@*}"  # Hide password
if [[ "$DATABASE_URL" == *@* ]]; then
    echo "                @${DATABASE_URL#*@}"
fi
echo "Compressed:     $IS_COMPRESSED"
echo "Encrypted:      $IS_ENCRYPTED"
echo ""
echo "========================================"
echo ""

# Confirmation prompt
if [[ "$FORCE" == false ]]; then
    log_warn "WARNING: This will OVERWRITE the current database!"
    log_prompt "Are you sure you want to continue? (yes/no):"
    read CONFIRM
    if [[ "$CONFIRM" != "yes" ]]; then
        log_info "Restore cancelled"
        exit 0
    fi
fi

# Create temporary directory for working files
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

WORKING_FILE="$BACKUP_FILE"

# Decrypt if needed
if [[ "$IS_ENCRYPTED" == true ]]; then
    log_info "Decrypting backup..."
    DECRYPTED_FILE="${TEMP_DIR}/backup_decrypted"
    
    # Determine the file extension after .enc
    BASE_NAME=$(basename "$BACKUP_FILE" .enc)
    if [[ "$BASE_NAME" == *.gz ]]; then
        DECRYPTED_FILE="${DECRYPTED_FILE}.gz"
    fi
    
    if openssl enc -aes-256-cbc -d \
        -in "$WORKING_FILE" \
        -out "$DECRYPTED_FILE" \
        -pass "pass:$ENCRYPTION_KEY"; then
        WORKING_FILE="$DECRYPTED_FILE"
        log_info "Decryption completed"
    else
        log_error "Decryption failed - check your encryption key"
        exit 1
    fi
fi

# Decompress if needed
if [[ "$IS_COMPRESSED" == true || "$WORKING_FILE" == *.gz ]]; then
    log_info "Decompressing backup..."
    DECOMPRESSED_FILE="${TEMP_DIR}/backup_decompressed.sql"
    
    if gunzip -c "$WORKING_FILE" > "$DECOMPRESSED_FILE"; then
        WORKING_FILE="$DECOMPRESSED_FILE"
        log_info "Decompression completed"
    else
        log_error "Decompression failed"
        exit 1
    fi
fi

# Verify the SQL file
log_info "Verifying backup file..."
if ! head -n 5 "$WORKING_FILE" | grep -q "PostgreSQL\|CREATE\|INSERT\|--"; then
    log_warn "Backup file may not be a valid PostgreSQL dump"
    log_prompt "Continue anyway? (yes/no):"
    read CONFIRM
    if [[ "$CONFIRM" != "yes" ]]; then
        log_info "Restore cancelled"
        exit 0
    fi
fi

# Check database connection
log_info "Checking database connection..."
if ! psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1; then
    log_error "Cannot connect to database"
    exit 1
fi

# Perform restore
log_info "Starting database restore..."
log_warn "This may take several minutes depending on backup size"
echo ""

if psql "$DATABASE_URL" -f "$WORKING_FILE"; then
    echo ""
    log_info "Restore completed successfully!"
else
    echo ""
    log_error "Restore failed!"
    exit 1
fi

# Post-restore information
echo ""
echo "========================================"
echo "           Restore Summary"
echo "========================================"
echo ""
echo "Database restored from:"
echo "  $BACKUP_FILE"
echo ""
echo "Next steps:"
echo "  1. Verify application functionality"
echo "  2. Check control plane logs"
echo "  3. Test agent connectivity"
echo ""
echo "========================================"

exit 0
