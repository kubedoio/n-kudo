#!/bin/sh
# Fix API URL in built JS files
# This replaces the hardcoded backend URL with /api for proxying

JS_FILE="/usr/share/nginx/html/assets/index-*.js"

# Replace backend:8443 with localhost:8443 (for external access)
# or use sed to replace with the actual IP
if [ -n "$VITE_API_BASE_URL" ]; then
    # Extract just the path if it's /api
    if echo "$VITE_API_BASE_URL" | grep -q "^/"; then
        # Replace full URL with just the path
        sed -i "s|https://backend:8443|$VITE_API_BASE_URL|g" $JS_FILE
        echo "Updated API base URL to: $VITE_API_BASE_URL"
    else
        # Keep the full URL for external access
        sed -i "s|https://backend:8443|$VITE_API_BASE_URL|g" $JS_FILE
        echo "Updated API base URL to: $VITE_API_BASE_URL"
    fi
fi

# Execute the original nginx command
exec "$@"
