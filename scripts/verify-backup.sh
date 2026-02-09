#!/usr/bin/env bash
#
# N-Kudo Backup Verification Script
# Verifies backup file integrity
#

set -euo pipefail

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

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS] <backup-file>

Verify integrity of N-Kudo database backup files.

OPTIONS:
    -v, --verbose           Show detailed verification output
    -h, --help              Show this help message

ARGUMENTS:
    backup-file             Path to backup file (.sql, .sql.gz, or .sql.enc)

EXAMPLES:
    # Verify a backup file
    $0 /backups/backup_20240101_120000.sql.gz

    # Verbose verification
    $0 -v /backups/backup_20240101_120000.sql

EOF
}

# Parse arguments
VERBOSE=false
BACKUP_FILE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
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

# Get file information
FILE_NAME=$(basename "$BACKUP_FILE")
FILE_SIZE=$(stat -f%z "$BACKUP_FILE" 2>/dev/null || stat -c%s "$BACKUP_FILE" 2>/dev/null || echo "unknown")
FILE_SIZE_HUMAN=$(du -h "$BACKUP_FILE" 2>/dev/null | cut -f1 || echo "unknown")

# Detect backup type
IS_ENCRYPTED=false
IS_COMPRESSED=false

if [[ "$BACKUP_FILE" == *.enc ]]; then
    IS_ENCRYPTED=true
fi
if [[ "$BACKUP_FILE" == *.gz ]]; then
    IS_COMPRESSED=true
fi

# Print header
echo ""
echo "========================================"
echo "      N-Kudo Backup Verification"
echo "========================================"
echo ""
echo "File:       $FILE_NAME"
echo "Size:       $FILE_SIZE_HUMAN ($FILE_SIZE bytes)"
echo "Compressed: $IS_COMPRESSED"
echo "Encrypted:  $IS_ENCRYPTED"
echo ""
echo "========================================"
echo ""

# Track verification status
ALL_PASSED=true

# Check file is readable
if [[ -r "$BACKUP_FILE" ]]; then
    log_success "File is readable"
else
    log_error "File is not readable"
    ALL_PASSED=false
fi

# Check file is non-empty
if [[ "$FILE_SIZE" -gt 0 ]]; then
    log_success "File is not empty"
else
    log_error "File is empty"
    ALL_PASSED=false
fi

# Verify based on file type
if [[ "$IS_ENCRYPTED" == true ]]; then
    log_warn "Encrypted backup - limited verification possible"
    log_info "To fully verify, decrypt the backup first:"
    echo "  openssl enc -aes-256-cbc -d -in $BACKUP_FILE -out decrypted.sql.gz"
    
    # Check if file looks like an openssl encrypted file
    # OpenSSL encrypted files typically start with "Salted__"
    if [[ "$VERBOSE" == true ]]; then
        FIRST_BYTES=$(xxd -l 8 "$BACKUP_FILE" 2>/dev/null || head -c 8 "$BACKUP_FILE" | od -A x -t x1z -v | head -1)
        log_info "First bytes: $FIRST_BYTES"
    fi
    
    # Try to detect if it's actually encrypted (not just named .enc)
    MAGIC=$(head -c 8 "$BACKUP_FILE")
    if [[ "$MAGIC" == "Salted__" ]]; then
        log_success "File appears to be valid OpenSSL encrypted format"
    else
        log_warn "File does not have expected OpenSSL header (may not be encrypted)"
    fi
    
elif [[ "$IS_COMPRESSED" == true ]]; then
    log_info "Verifying gzip compressed backup..."
    
    if command -v gzip &> /dev/null; then
        if gzip -t "$BACKUP_FILE" 2>&1; then
            log_success "Gzip integrity check passed"
            
            if [[ "$VERBOSE" == true ]]; then
                # Get compression info
                ORIG_SIZE=$(gzip -l "$BACKUP_FILE" 2>/dev/null | tail -1 | awk '{print $2}')
                COMP_RATIO=$(gzip -l "$BACKUP_FILE" 2>/dev/null | tail -1 | awk '{print $3}')
                if [[ -n "$ORIG_SIZE" ]]; then
                    echo "  Original size: $ORIG_SIZE bytes"
                    echo "  Compression ratio: $COMP_RATIO%"
                fi
                
                # Show first few lines of SQL content
                echo ""
                log_info "First 10 lines of SQL content:"
                echo "---"
                gunzip -c "$BACKUP_FILE" 2>/dev/null | head -n 10
                echo "---"
            fi
            
            # Check for SQL content indicators
            if gunzip -c "$BACKUP_FILE" 2>/dev/null | head -n 20 | grep -q "PostgreSQL\|CREATE\|INSERT\|COPY\|--"; then
                log_success "File contains valid SQL content indicators"
            else
                log_warn "File does not contain expected SQL content indicators"
                ALL_PASSED=false
            fi
        else
            log_error "Gzip integrity check failed - file may be corrupted"
            ALL_PASSED=false
        fi
    else
        log_warn "gzip not available, skipping compression verification"
    fi
else
    # Plain SQL file
    log_info "Verifying plain SQL backup..."
    
    if command -v head &> /dev/null; then
        # Check for SQL content indicators
        if head -n 50 "$BACKUP_FILE" | grep -q "PostgreSQL\|CREATE\|INSERT\|COPY\|--"; then
            log_success "File contains valid SQL content indicators"
        else
            log_warn "File does not contain expected SQL content indicators"
            ALL_PASSED=false
        fi
        
        # Check for proper line endings
        if file "$BACKUP_FILE" | grep -q "text"; then
            log_success "File is valid text format"
        fi
        
        if [[ "$VERBOSE" == true ]]; then
            # Count SQL statements
            CREATE_COUNT=$(grep -c "^CREATE " "$BACKUP_FILE" 2>/dev/null || echo "0")
            INSERT_COUNT=$(grep -c "^INSERT " "$BACKUP_FILE" 2>/dev/null || echo "0")
            COPY_COUNT=$(grep -c "^COPY " "$BACKUP_FILE" 2>/dev/null || echo "0")
            
            echo ""
            log_info "SQL statement counts:"
            echo "  CREATE statements: $CREATE_COUNT"
            echo "  INSERT statements: $INSERT_COUNT"
            echo "  COPY statements: $COPY_COUNT"
            
            echo ""
            log_info "First 10 lines:"
            echo "---"
            head -n 10 "$BACKUP_FILE"
            echo "---"
        fi
    fi
fi

# Check file modification time
if command -v stat &> /dev/null; then
    MOD_TIME=$(stat -c "%y" "$BACKUP_FILE" 2>/dev/null || stat -f "%Sm" "$BACKUP_FILE" 2>/dev/null)
    if [[ -n "$MOD_TIME" ]]; then
        log_info "Backup created: $MOD_TIME"
    fi
fi

# Summary
echo ""
echo "========================================"
if [[ "$ALL_PASSED" == true ]]; then
    log_success "Verification PASSED"
    exit 0
else
    log_error "Verification FAILED or incomplete"
    exit 1
fi
