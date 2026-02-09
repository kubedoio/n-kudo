#!/usr/bin/env bash
# Live API test against running control-plane
set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-https://localhost:8443}"
ADMIN_KEY="${ADMIN_KEY:-dev-admin-key}"

echo "=== Testing against ${CONTROL_PLANE_URL} ==="

# 1. Health check
echo -n "Health check... "
curl -skf "${CONTROL_PLANE_URL}/healthz" > /dev/null && echo "✅" || { echo "❌"; exit 1; }

# 2. Create tenant
echo -n "Create tenant... "
TENANT_RESP=$(curl -sk -X POST "${CONTROL_PLANE_URL}/tenants" \
  -H "X-Admin-Key: ${ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"slug\":\"test-live-$(date +%s)\",\"name\":\"Live Test Tenant\"}")
TENANT_ID=$(echo "$TENANT_RESP" | jq -r '.id')
if [ -z "$TENANT_ID" ] || [ "$TENANT_ID" = "null" ]; then
  echo "❌ Failed to create tenant: $TENANT_RESP"
  exit 1
fi
echo "✅ (${TENANT_ID:0:8})"

# 3. Create API key
echo -n "Create API key... "
API_KEY_RESP=$(curl -sk -X POST "${CONTROL_PLANE_URL}/tenants/${TENANT_ID}/api-keys" \
  -H "X-Admin-Key: ${ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-key"}')
API_KEY=$(echo "$API_KEY_RESP" | jq -r '.api_key')
if [ -z "$API_KEY" ] || [ "$API_KEY" = "null" ]; then
  echo "❌ Failed to create API key: $API_KEY_RESP"
  exit 1
fi
echo "✅ (${API_KEY:0:15}...)"

# 4. Create site
echo -n "Create site... "
SITE_RESP=$(curl -sk -X POST "${CONTROL_PLANE_URL}/tenants/${TENANT_ID}/sites" \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-site","location_country_code":"US"}')
SITE_ID=$(echo "$SITE_RESP" | jq -r '.id')
if [ -z "$SITE_ID" ] || [ "$SITE_ID" = "null" ]; then
  echo "❌ Failed to create site: $SITE_RESP"
  exit 1
fi
echo "✅ (${SITE_ID:0:8})"

# 5. Issue enrollment token
echo -n "Issue enrollment token... "
TOKEN_RESP=$(curl -sk -X POST "${CONTROL_PLANE_URL}/tenants/${TENANT_ID}/enrollment-tokens" \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"site_id\":\"${SITE_ID}\",\"expires_in_seconds\":300}")
TOKEN=$(echo "$TOKEN_RESP" | jq -r '.token')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "❌ Failed to create token: $TOKEN_RESP"
  exit 1
fi
echo "✅"

# 6. List sites
echo -n "List sites... "
SITES_RESP=$(curl -sk "${CONTROL_PLANE_URL}/tenants/${TENANT_ID}/sites" \
  -H "X-API-Key: ${API_KEY}")
echo "✅ ($(echo "$SITES_RESP" | jq '.sites | length') sites)"

# 7. Apply plan
echo -n "Apply plan... "
PLAN_RESP=$(curl -sk -X POST "${CONTROL_PLANE_URL}/sites/${SITE_ID}/plans" \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{
    \"idempotency_key\": \"test-plan-$(date +%s)\",
    \"actions\": [
      {\"operation_id\":\"op-1\",\"operation\":\"CREATE\",\"vm_id\":\"test-vm-001\",\"name\":\"Test VM\",\"vcpu_count\":1,\"memory_mib\":256}
    ]
  }")
PLAN_ID=$(echo "$PLAN_RESP" | jq -r '.plan_id')
if [ -z "$PLAN_ID" ] || [ "$PLAN_ID" = "null" ]; then
  echo "❌ Failed to apply plan: $PLAN_RESP"
  exit 1
fi
echo "✅ (${PLAN_ID:0:8})"

# 8. Query hosts
echo -n "List hosts... "
HOSTS_RESP=$(curl -sk "${CONTROL_PLANE_URL}/sites/${SITE_ID}/hosts" \
  -H "X-API-Key: ${API_KEY}")
if [ "$(echo "$HOSTS_RESP" | jq -r '.hosts // "null"')" = "null" ]; then
  echo "⚠️ (no hosts yet - expected before agent enrollment)"
else
  echo "✅ ($(echo "$HOSTS_RESP" | jq '.hosts | length') hosts)"
fi

# 9. Query VMs
echo -n "List VMs... "
VMS_RESP=$(curl -sk "${CONTROL_PLANE_URL}/sites/${SITE_ID}/vms" \
  -H "X-API-Key: ${API_KEY}")
echo "✅ ($(echo "$VMS_RESP" | jq '.vms | length') vms)"

# 10. Query executions for plan
echo -n "Query executions... "
EXEC_COUNT=$(echo "$PLAN_RESP" | jq '.executions | length')
echo "✅ ($EXEC_COUNT executions)"

echo ""
echo "=== All API tests passed ✅ ==="
echo ""
echo "Test artifacts:"
echo "  tenant_id:    ${TENANT_ID}"
echo "  site_id:      ${SITE_ID}"
echo "  api_key:      ${API_KEY}"
echo "  enroll_token: ${TOKEN}"
echo "  plan_id:      ${PLAN_ID}"
