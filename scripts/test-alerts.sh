#!/usr/bin/env bash
# =============================================================================
# n-kudo Test Alerts Script
# =============================================================================
# This script generates test metrics to trigger alerts in Prometheus.
# Use this for validating alert rules and notification routing.
#
# Usage:
#   ./scripts/test-alerts.sh [alert-type] [duration]
#
# Alert Types:
#   high-error-rate      - Trigger HighErrorRate alert
#   agents-offline       - Trigger AgentsOffline alert  
#   cert-expiring        - Trigger CertificateExpiringSoon alert
#   heartbeat-failures   - Trigger HeartbeatFailures alert
#   db-errors            - Trigger DatabaseConnectionErrors alert
#   slo-breach           - Trigger SLOBreached alert
#   all                  - Trigger all alerts (default)
#
# Examples:
#   ./scripts/test-alerts.sh                    # Trigger all alerts for 5 min
#   ./scripts/test-alerts.sh high-error-rate    # Trigger specific alert
#   ./scripts/test-alerts.sh all 600            # Trigger for 10 minutes
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Configuration
ALERT_TYPE="${1:-all}"
DURATION="${2:-300}"  # Default 5 minutes
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-https://localhost:8443}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
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

# Check if required tools are available
check_prerequisites() {
    local missing=()
    
    if ! command -v curl &> /dev/null; then
        missing+=("curl")
    fi
    
    if ! command -v jq &> /dev/null; then
        missing+=("jq")
    fi
    
    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing[*]}"
        exit 1
    fi
}

# Check if Prometheus is reachable
check_prometheus() {
    log_info "Checking Prometheus at ${PROMETHEUS_URL}..."
    
    if curl -sf "${PROMETHEUS_URL}/-/healthy" > /dev/null 2>&1; then
        log_success "Prometheus is healthy"
    else
        log_error "Cannot connect to Prometheus at ${PROMETHEUS_URL}"
        log_info "Make sure Prometheus is running or set PROMETHEUS_URL"
        exit 1
    fi
}

# Push test metric to Prometheus pushgateway (if available) or use metrics endpoint
push_test_metric() {
    local metric_name="$1"
    local value="$2"
    local labels="${3:-}"
    
    log_info "Setting metric ${metric_name}${labels} = ${value}"
    
    # For direct metric injection, we'll use a workaround via the control plane
    # In production, you'd use a pushgateway or direct instrumentation
}

# Generate high error rate by making failing requests
trigger_high_error_rate() {
    log_info "Triggering HighErrorRate alert..."
    log_info "Making requests that will return 5xx errors for ${DURATION} seconds"
    
    local end_time=$(($(date +%s) + DURATION))
    local count=0
    
    while [ $(date +%s) -lt $end_time ]; do
        # Make requests to invalid endpoints to generate 5xx errors
        curl -sfk "${CONTROL_PLANE_URL}/api/v1/invalid-endpoint-trigger-error" > /dev/null 2>&1 || true
        curl -sfk "${CONTROL_PLANE_URL}/api/v1/agents/invalid-id" > /dev/null 2>&1 || true
        
        count=$((count + 1))
        if [ $((count % 10)) -eq 0 ]; then
            log_info "Sent $count error-generating requests..."
        fi
        
        # Small delay to not overwhelm
        sleep 0.1
    done
    
    log_success "High error rate test completed"
}

# Trigger agents offline alert
trigger_agents_offline() {
    log_info "Triggering AgentsOffline alert..."
    log_info "Simulating offline agents for ${DURATION} seconds"
    
    # This would require manipulating the database or agent heartbeats
    # For testing, we can inject a test metric if using pushgateway
    
    log_warn "To trigger this alert in production:"
    log_warn "  1. Stop 5+ edge agent services, OR"
    log_warn "  2. Insert test data: INSERT INTO agents (id, status, last_heartbeat) VALUES (...)"
    
    # Simulate by creating a temporary metric file if using node_exporter textfile
    if [ -d "/var/lib/node_exporter/textfile_collector" ]; then
        cat > /var/lib/node_exporter/textfile_collector/nkudo_test.prom <<EOF
# HELP nkudo_agents_offline_total Test metric for agents offline
# TYPE nkudo_agents_offline_total gauge
nkudo_agents_offline_total 6
# HELP nkudo_agents_total Test metric for total agents
# TYPE nkudo_agents_total gauge  
nkudo_agents_total 10
EOF
        log_success "Created test metric file"
        sleep $DURATION
        rm -f /var/lib/node_exporter/textfile_collector/nkudo_test.prom
    fi
}

# Trigger certificate expiring alert
trigger_cert_expiring() {
    log_info "Triggering CertificateExpiringSoon alert..."
    
    log_warn "To trigger this alert:"
    log_warn "  1. Create a test certificate expiring soon, OR"
    log_warn "  2. Manually set metric: nkudo_certificate_expiry_timestamp = now() + 3 days"
    
    # If using pushgateway, we could push a test metric
    local test_expiry=$(date -d "+3 days" +%s)
    log_info "Example metric to push:"
    log_info "  nkudo_certificate_expiry_timestamp{common_name=\"test-cert\"} ${test_expiry}"
}

# Trigger heartbeat failures
trigger_heartbeat_failures() {
    log_info "Triggering HeartbeatFailures alert..."
    log_info "Simulating heartbeat failures for ${DURATION} seconds"
    
    # This would require manipulating the heartbeat mechanism
    log_warn "To trigger this alert:"
    log_warn "  1. Block network connectivity between agents and control plane, OR"
    log_warn "  2. Stop the control plane temporarily, OR"
    log_warn "  3. Inject test metric: nkudo_heartbeat_failures_total"
}

# Trigger database connection errors
trigger_db_errors() {
    log_info "Triggering DatabaseConnectionErrors alert..."
    
    log_warn "To trigger this alert:"
    log_warn "  1. Temporarily stop the PostgreSQL container, OR"
    log_warn "  2. Set incorrect DATABASE_URL to force connection errors"
    
    # Check if we can temporarily disrupt DB connectivity
    read -p "Stop PostgreSQL container temporarily? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if command -v docker &> /dev/null; then
            log_info "Stopping PostgreSQL for 30 seconds..."
            docker stop nkudo-postgres
            sleep 30
            log_info "Starting PostgreSQL..."
            docker start nkudo-postgres
            log_success "Database test completed"
        else
            log_error "Docker not available"
        fi
    fi
}

# Trigger SLO breach
trigger_slo_breach() {
    log_info "Triggering SLOBreached alert..."
    log_info "Simulating SLO error budget exhaustion"
    
    log_warn "To trigger this alert:"
    log_warn "  1. Generate many errors to exhaust error budget, OR"
    log_warn "  2. Inject test metrics for error budget usage"
    
    # Combine with high error rate to exhaust budget
    trigger_high_error_rate
}

# Display current alert status
show_alert_status() {
    log_info "Checking current alert status..."
    
    # Query Prometheus for firing alerts
    local alerts
    if alerts=$(curl -sf "${PROMETHEUS_URL}/api/v1/alerts" 2>/dev/null | jq -r '.data.alerts[] | select(.state == "firing") | "\(.labels.alertname): \(.labels.severity)"' 2>/dev/null); then
        if [ -n "$alerts" ]; then
            log_warn "Currently firing alerts:"
            echo "$alerts" | while read -r alert; do
                echo "  - $alert"
            done
        else
            log_info "No alerts currently firing"
        fi
    else
        log_warn "Could not query Prometheus alerts"
    fi
}

# Reset all test state
reset_tests() {
    log_info "Resetting test state..."
    
    # Remove any test metric files
    if [ -d "/var/lib/node_exporter/textfile_collector" ]; then
        rm -f /var/lib/node_exporter/textfile_collector/nkudo_test.prom
    fi
    
    # Clear any test alerts in Alertmanager
    curl -sf -X POST "${PROMETHEUS_URL}/api/v1/admin/tsdb/delete_series?match[]={__name__=~\"nkudo_test_.*\"}" > /dev/null 2>&1 || true
    
    log_success "Test state reset"
}

# Main execution
main() {
    echo "============================================================================="
    echo "                     n-kudo Alert Testing Tool"
    echo "============================================================================="
    echo ""
    
    check_prerequisites
    check_prometheus
    
    # Show current status
    show_alert_status
    echo ""
    
    # Confirm with user for destructive tests
    if [[ "$ALERT_TYPE" == "db-errors" ]] || [[ "$ALERT_TYPE" == "all" ]]; then
        log_warn "Some tests may temporarily disrupt service availability"
        read -p "Continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Aborted"
            exit 0
        fi
    fi
    
    log_info "Starting alert test: type=${ALERT_TYPE}, duration=${DURATION}s"
    echo ""
    
    case "$ALERT_TYPE" in
        high-error-rate)
            trigger_high_error_rate
            ;;
        agents-offline)
            trigger_agents_offline
            ;;
        cert-expiring)
            trigger_cert_expiring
            ;;
        heartbeat-failures)
            trigger_heartbeat_failures
            ;;
        db-errors)
            trigger_db_errors
            ;;
        slo-breach)
            trigger_slo_breach
            ;;
        all)
            log_info "Running all alert tests sequentially..."
            trigger_high_error_rate
            sleep 5
            trigger_agents_offline
            sleep 5
            trigger_cert_expiring
            sleep 5
            trigger_heartbeat_failures
            sleep 5
            trigger_db_errors
            sleep 5
            trigger_slo_breach
            ;;
        *)
            log_error "Unknown alert type: $ALERT_TYPE"
            echo "Valid types: high-error-rate, agents-offline, cert-expiring, heartbeat-failures, db-errors, slo-breach, all"
            exit 1
            ;;
    esac
    
    echo ""
    log_info "Waiting for alerts to fire in Prometheus (may take up to 2 minutes)..."
    sleep 30
    
    show_alert_status
    
    echo ""
    log_info "Test complete. To clean up test state, run:"
    log_info "  $0 reset"
}

# Handle reset command
if [[ "$ALERT_TYPE" == "reset" ]]; then
    reset_tests
    exit 0
fi

main "$@"
