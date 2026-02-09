#!/usr/bin/env bash
# Quick API smoke test against running control-plane
set -euo pipefail

CP="${CONTROL_PLANE_URL:-https://localhost:8443}"
ADMIN_KEY="${ADMIN_KEY:-dev-admin-key}"

echo "=== Quick API Smoke Test ==="
echo "Target: $CP"
echo ""

# Health check
echo -n "1. Health check... "
curl -skf "$CP/healthz" > /dev/null && echo "✅" || { echo "❌"; exit 1; }

# Create tenant
echo -n "2. Create tenant... "
RESP=$(curl -sk -X POST "$CP/tenants" \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"slug":"smoke-'$(date +%s)'","name":"Smoke Test"}')
TENANT_ID=$(echo "$RESP" | jq -r '.id')
[ -n "$TENANT_ID" ] && echo "✅ (${TENANT_ID:0:8})" || { echo "❌ $RESP"; exit 1; }

# Create API key  
echo -n "3. Create API key... "
KEY_RESP=$(curl -sk -X POST "$CP/tenants/$TENANT_ID/api-keys" \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -d '{"name":"smoke"}')
API_KEY=$(echo "$KEY_RESP" | jq -r '.api_key')
[ -n "$API_KEY" ] && echo "✅" || { echo "❌ $KEY_RESP"; exit 1; }

# Create site
echo -n "4. Create site... "
SITE_RESP=$(curl -sk -X POST "$CP/tenants/$TENANT_ID/sites" \
  -H "X-API-Key: $API_KEY" \
  -d '{"name":"smoke-site"}')
SITE_ID=$(echo "$SITE_RESP" | jq -r '.id')
[ -n "$SITE_ID" ] && echo "✅ (${SITE_ID:0:8})" || { echo "❌ $SITE_RESP"; exit 1; }

# Issue token
echo -n "5. Issue enrollment token... "
TOKEN_RESP=$(curl -sk -X POST "$CP/tenants/$TENANT_ID/enrollment-tokens" \
  -H "X-API-Key: $API_KEY" \
  -d "{\"site_id\":\"$SITE_ID\",\"expires_in_seconds\":60}")
TOKEN=$(echo "$TOKEN_RESP" | jq -r '.token')
[ -n "$TOKEN" ] && echo "✅" || { echo "❌ $TOKEN_RESP"; exit 1; }

# List sites
echo -n "6. List sites... "
SITES_RESP=$(curl -sk "$CP/tenants/$TENANT_ID/sites" -H "X-API-Key: $API_KEY")
COUNT=$(echo "$SITES_RESP" | jq '.sites | length')
echo "✅ ($COUNT sites)"

echo ""
echo "=== Smoke test passed! ✅ ==="
echo ""
echo "To enroll an agent:"
echo "  ./bin/edge enroll --control-plane $CP --token '$TOKEN' --insecure-skip-verify"
