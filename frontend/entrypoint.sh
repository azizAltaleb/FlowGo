#!/bin/sh
set -eu

escape_js_string() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

api_url="${VITE_API_URL:-/api}"
oidc_authority="${FRONTEND_AUTH_OIDC_AUTHORITY:-${VITE_OIDC_AUTHORITY:-}}"
oidc_client_id="${FRONTEND_AUTH_OIDC_CLIENT_ID:-${VITE_OIDC_CLIENT_ID:-}}"
oidc_client_id_file="${FRONTEND_AUTH_OIDC_CLIENT_ID_FILE:-}"

if [ -n "$oidc_client_id_file" ]; then
  timeout="${FRONTEND_AUTH_OIDC_CLIENT_ID_FILE_TIMEOUT_SECONDS:-120}"
  while [ ! -s "$oidc_client_id_file" ] && [ "$timeout" -gt 0 ]; do
    sleep 1
    timeout=$((timeout - 1))
  done
  if [ -s "$oidc_client_id_file" ]; then
    oidc_client_id="$(cat "$oidc_client_id_file")"
  fi
fi

cat > /usr/share/nginx/html/runtime-config.js <<EOF
window.__WORKFLOWSA_RUNTIME_CONFIG__ = {
  apiUrl: "$(escape_js_string "$api_url")",
  oidcAuthority: "$(escape_js_string "$oidc_authority")",
  oidcClientId: "$(escape_js_string "$oidc_client_id")"
};
EOF

exec "$@"
