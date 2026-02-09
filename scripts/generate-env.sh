#!/bin/bash
# generate-env.sh - Generate secure environment variables for n-kudo
#
# Usage:
#   ./scripts/generate-env.sh          # Generate and display values
#   ./scripts/generate-env.sh --save   # Save to .env file (backups existing)
#   ./scripts/generate-env.sh --dev    # Generate dev-friendly values

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_ROOT/.env"
ENV_EXAMPLE="$PROJECT_ROOT/.env.example"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Generate a random password
generate_password() {
    local length=${1:-32}
    openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c "$length"
}

# Generate a random hex string
generate_hex() {
    local length=${1:-32}
    openssl rand -hex "$((length / 2))"
}

# Generate a dev-friendly readable key
generate_dev_key() {
    local prefix=${1:-key}
    local suffix=$(openssl rand -hex 4)
    echo "${prefix}-dev-${suffix}"
}

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --save       Save generated values to .env file (backs up existing)"
    echo "  --dev        Generate development-friendly (readable) values"
    echo "  --quiet      Only output errors"
    echo "  -h, --help   Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Display generated values"
    echo "  $0 --save             # Save secure values to .env"
    echo "  $0 --dev --save       # Save dev-friendly values to .env"
}

# Parse arguments
SAVE_MODE=false
DEV_MODE=false
QUIET=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --save)
            SAVE_MODE=true
            shift
            ;;
        --dev)
            DEV_MODE=true
            shift
            ;;
        --quiet)
            QUIET=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Check for openssl
if ! command -v openssl &> /dev/null; then
    echo -e "${RED}Error: openssl is required but not installed.${NC}"
    exit 1
fi

# Generate values
if [ "$DEV_MODE" = true ]; then
    # Development-friendly values
    POSTGRES_USER="nkudo"
    POSTGRES_PASSWORD="nkudo-dev-password-$(generate_dev_key)"
    POSTGRES_DB="nkudo"
    ADMIN_KEY="dev-admin-key-$(openssl rand -hex 4)"
    CA_COMMON_NAME="n-kudo-dev-ca"
else
    # Production-secure values
    POSTGRES_USER="nkudo"
    POSTGRES_PASSWORD="$(generate_password 32)"
    POSTGRES_DB="nkudo"
    ADMIN_KEY="nk_$(generate_hex 32)"
    CA_COMMON_NAME="n-kudo-agent-ca"
fi

# Generate timestamp
GENERATED_AT=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Create environment file content
generate_env_content() {
    cat << EOF
# n-kudo Docker Compose Environment Configuration
# Generated: $GENERATED_AT
# Mode: $([ "$DEV_MODE" = true ] && echo "development" || echo "production")
#
# WARNING: Keep this file secret! It contains sensitive credentials.

# =============================================
# Database Configuration
# =============================================
POSTGRES_USER=$POSTGRES_USER
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
POSTGRES_DB=$POSTGRES_DB

# =============================================
# Control Plane (Backend) Configuration
# =============================================
ADMIN_KEY=$ADMIN_KEY
CONTROL_PLANE_ADDR=:8443
DATABASE_URL=postgres://\${POSTGRES_USER}:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}?sslmode=disable
CA_COMMON_NAME=$CA_COMMON_NAME

# =============================================
# Frontend Configuration
# =============================================
VITE_API_BASE_URL=https://backend:8443
EOF
}

# Output to console (unless quiet mode)
if [ "$QUIET" = false ]; then
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║        n-kudo Environment Generator                    ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    if [ "$DEV_MODE" = true ]; then
        echo -e "${YELLOW}Mode: Development (human-readable values)${NC}"
    else
        echo -e "${YELLOW}Mode: Production (secure random values)${NC}"
    fi
    echo ""
    echo "Generated values:"
    echo "─────────────────────────────────────────────────────────"
    echo -e "POSTGRES_USER:     ${GREEN}$POSTGRES_USER${NC}"
    echo -e "POSTGRES_PASSWORD: ${GREEN}$POSTGRES_PASSWORD${NC}"
    echo -e "POSTGRES_DB:       ${GREEN}$POSTGRES_DB${NC}"
    echo -e "ADMIN_KEY:         ${GREEN}$ADMIN_KEY${NC}"
    echo -e "CA_COMMON_NAME:    ${GREEN}$CA_COMMON_NAME${NC}"
    echo "─────────────────────────────────────────────────────────"
    echo ""
fi

# Save to file if requested
if [ "$SAVE_MODE" = true ]; then
    # Backup existing .env if it exists
    if [ -f "$ENV_FILE" ]; then
        BACKUP_FILE="$ENV_FILE.backup.$(date +%Y%m%d%H%M%S)"
        cp "$ENV_FILE" "$BACKUP_FILE"
        if [ "$QUIET" = false ]; then
            echo -e "${YELLOW}Existing .env backed up to: $BACKUP_FILE${NC}"
        fi
    fi
    
    # Write new .env file
    generate_env_content > "$ENV_FILE"
    chmod 600 "$ENV_FILE"
    
    if [ "$QUIET" = false ]; then
        echo -e "${GREEN}✓ Environment saved to: $ENV_FILE${NC}"
        echo -e "${YELLOW}  File permissions set to 600 (owner read/write only)${NC}"
        echo ""
        echo -e "${BLUE}Next steps:${NC}"
        echo "  1. Run: docker compose up -d"
        echo "  2. Access frontend at: http://localhost:3000"
        echo "  3. Admin API available at: https://localhost:8443"
        echo ""
    fi
else
    if [ "$QUIET" = false ]; then
        echo -e "${YELLOW}To save these values to .env, run:${NC}"
        echo "  $0 --save"
        echo ""
        echo -e "${YELLOW}For development-friendly values:${NC}"
        echo "  $0 --dev --save"
        echo ""
    fi
fi

# Also output enrollment helper if not quiet
if [ "$QUIET" = false ]; then
    echo -e "${BLUE}Quick reference commands:${NC}"
    echo "─────────────────────────────────────────────────────────"
    echo "# Start all services:"
    echo "  docker compose up -d"
    echo ""
    echo "# Get an enrollment token for edge agents:"
    echo "  curl -k -H \"X-Admin-Key: $ADMIN_KEY\" \"https://localhost:8443/admin/enroll-token?site_id=<site-id>\""
    echo ""
    echo "# View logs:"
    echo "  docker compose logs -f"
    echo "─────────────────────────────────────────────────────────"
    echo ""
fi
