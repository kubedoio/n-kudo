#!/bin/bash
# setup.sh - Quick setup script for n-kudo development environment
#
# Usage:
#   ./scripts/setup.sh         # Full setup with dev credentials
#   ./scripts/setup.sh --prod  # Setup with production credentials
#   ./scripts/setup.sh --reset # Reset environment (removes volumes)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Parse arguments
MODE="dev"
RESET=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --prod)
            MODE="prod"
            shift
            ;;
        --dev)
            MODE="dev"
            shift
            ;;
        --reset)
            RESET=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --dev      Use development-friendly credentials (default)"
            echo "  --prod     Use production-secure credentials"
            echo "  --reset    Reset environment (removes volumes and rebuilds)"
            echo "  -h, --help Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0              # Quick dev setup"
            echo "  $0 --reset      # Reset and start fresh"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║          n-kudo Setup Script                           ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

if ! command -v docker compose &> /dev/null && ! docker-compose version &> /dev/null; then
    echo -e "${RED}Error: Docker Compose is not installed${NC}"
    exit 1
fi

# Reset if requested
if [ "$RESET" = true ]; then
    echo -e "${YELLOW}Resetting environment...${NC}"
    cd "$PROJECT_ROOT"
    docker compose down -v 2>/dev/null || true
    echo -e "${GREEN}✓ Environment reset${NC}"
    echo ""
fi

# Generate environment file
if [ ! -f "$PROJECT_ROOT/.env" ] || [ "$RESET" = true ]; then
    echo -e "${CYAN}Generating environment file...${NC}"
    if [ "$MODE" = "dev" ]; then
        "$SCRIPT_DIR/generate-env.sh" --dev --save --quiet
    else
        "$SCRIPT_DIR/generate-env.sh" --save --quiet
    fi
    echo -e "${GREEN}✓ Environment file created${NC}"
else
    echo -e "${YELLOW}Using existing .env file (use --reset to recreate)${NC}"
fi

echo ""
echo -e "${CYAN}Starting services...${NC}"
echo ""

# Build and start services
cd "$PROJECT_ROOT"
docker compose up --build -d

echo ""
echo -e "${GREEN}✓ Services started successfully!${NC}"
echo ""

# Wait for services to be healthy
echo -e "${CYAN}Waiting for services to be healthy...${NC}"
sleep 3

# Check health
MAX_RETRIES=30
RETRY=0
while [ $RETRY -lt $MAX_RETRIES ]; do
    if docker compose ps | grep -q "healthy"; then
        break
    fi
    RETRY=$((RETRY + 1))
    sleep 2
done

# Display status
echo ""
echo -e "${BLUE}══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}                    Setup Complete!                       ${NC}"
echo -e "${BLUE}══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${CYAN}Access Points:${NC}"
echo "  • Frontend Dashboard: ${GREEN}http://localhost:3000${NC}"
echo "  • Backend API:        ${GREEN}https://localhost:8443${NC}"
echo "  • Database (psql):    ${GREEN}localhost:5432${NC}"
echo ""

# Display credentials from .env
if [ -f "$PROJECT_ROOT/.env" ]; then
    ADMIN_KEY=$(grep "^ADMIN_KEY=" "$PROJECT_ROOT/.env" | cut -d '=' -f2)
    echo -e "${CYAN}Credentials:${NC}"
    echo "  • Admin Key: ${GREEN}$ADMIN_KEY${NC}"
    echo ""
fi

echo -e "${CYAN}Useful Commands:${NC}"
echo "  View logs:        docker compose logs -f"
echo "  Stop services:    docker compose down"
echo "  Reset everything: $0 --reset"
echo ""
echo -e "${CYAN}Get Enrollment Token:${NC}"
echo "  curl -k -H \"X-Admin-Key: <admin-key>\" \"https://localhost:8443/admin/enroll-token?site_id=<site-id>\""
echo ""
