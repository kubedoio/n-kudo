#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-https://localhost:8443}"
ADMIN_KEY="${ADMIN_KEY:-dev-admin-key}"

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing command: $1" >&2; exit 1; }
}

need curl
need jq
need openssl

echo "1) Create tenant"
TENANT_JSON=$(curl -sk -X POST "$BASE_URL/tenants" \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"slug":"acme","name":"Acme Corp","primary_region":"eu-central-1"}')
TENANT_ID=$(echo "$TENANT_JSON" | jq -r '.id')
echo "tenant_id=$TENANT_ID"

echo "2) Create tenant API key"
KEY_JSON=$(curl -sk -X POST "$BASE_URL/tenants/$TENANT_ID/api-keys" \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"dashboard-key"}')
API_KEY=$(echo "$KEY_JSON" | jq -r '.api_key')
echo "api_key=$API_KEY"

echo "3) Create site"
SITE_JSON=$(curl -sk -X POST "$BASE_URL/tenants/$TENANT_ID/sites" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"berlin-site-1","location_country_code":"DE"}')
SITE_ID=$(echo "$SITE_JSON" | jq -r '.id')
echo "site_id=$SITE_ID"

echo "4) Issue one-time enrollment token"
TOKEN_JSON=$(curl -sk -X POST "$BASE_URL/tenants/$TENANT_ID/enrollment-tokens" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"site_id\":\"$SITE_ID\",\"expires_in_seconds\":900}")
ENROLL_TOKEN=$(echo "$TOKEN_JSON" | jq -r '.token')
echo "enrollment_token=$ENROLL_TOKEN"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

openssl req -new -newkey rsa:2048 -nodes \
  -keyout "$TMPDIR/agent.key" \
  -out "$TMPDIR/agent.csr" \
  -subj "/CN=pending-agent" >/dev/null 2>&1
CSR_PEM=$(awk '{printf "%s\\n", $0}' "$TMPDIR/agent.csr")

echo "5) Enroll agent"
ENROLL_JSON=$(curl -sk -X POST "$BASE_URL/enroll" \
  -H "Content-Type: application/json" \
  -d "{\"enrollment_token\":\"$ENROLL_TOKEN\",\"hostname\":\"edge-host-01\",\"agent_version\":\"0.1.0\",\"os\":\"linux\",\"arch\":\"amd64\",\"csr_pem\":\"$CSR_PEM\"}")
AGENT_ID=$(echo "$ENROLL_JSON" | jq -r '.agent_id')
CA_PEM=$(echo "$ENROLL_JSON" | jq -r '.ca_certificate_pem')
AGENT_CERT=$(echo "$ENROLL_JSON" | jq -r '.client_certificate_pem')
printf "%s\n" "$CA_PEM" > "$TMPDIR/ca.pem"
printf "%s\n" "$AGENT_CERT" > "$TMPDIR/agent.crt"
echo "agent_id=$AGENT_ID"

echo "6) Send heartbeat via mTLS"
HEARTBEAT_JSON=$(curl -sk --cert "$TMPDIR/agent.crt" --key "$TMPDIR/agent.key" \
  -X POST "$BASE_URL/agents/heartbeat" \
  -H "Content-Type: application/json" \
  -d "{\"agent_id\":\"$AGENT_ID\",\"heartbeat_seq\":1,\"hostname\":\"edge-host-01\",\"agent_version\":\"0.1.0\",\"os\":\"linux\",\"arch\":\"amd64\",\"cpu_cores_total\":8,\"memory_bytes_total\":17179869184,\"storage_bytes_total\":137438953472,\"kvm_available\":true,\"cloud_hypervisor_available\":true}")
echo "$HEARTBEAT_JSON" | jq

echo "7) Verify inventory endpoints"
curl -sk "$BASE_URL/tenants/$TENANT_ID/sites" -H "X-API-Key: $API_KEY" | jq
curl -sk "$BASE_URL/sites/$SITE_ID/hosts" -H "X-API-Key: $API_KEY" | jq
