#!/bin/sh
set -eu

escape_js_string() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

api_url="${ARTIFICIALFLOW_API_URL:-${FLOWGO_API_URL:-${VITE_API_URL:-/api}}}"
oidc_authority="${ARTIFICIALFLOW_OIDC_AUTHORITY:-${FLOWGO_OIDC_AUTHORITY:-${FRONTEND_AUTH_OIDC_AUTHORITY:-${VITE_OIDC_AUTHORITY:-}}}}"
oidc_client_id="${ARTIFICIALFLOW_OIDC_CLIENT_ID:-${FLOWGO_OIDC_CLIENT_ID:-${FRONTEND_AUTH_OIDC_CLIENT_ID:-${VITE_OIDC_CLIENT_ID:-}}}}"
oidc_client_id_file="${ARTIFICIALFLOW_OIDC_CLIENT_ID_FILE:-${FLOWGO_OIDC_CLIENT_ID_FILE:-${FRONTEND_AUTH_OIDC_CLIENT_ID_FILE:-}}}"

if [ -n "$oidc_client_id_file" ]; then
  timeout="${ARTIFICIALFLOW_OIDC_CLIENT_ID_FILE_TIMEOUT_SECONDS:-${FLOWGO_OIDC_CLIENT_ID_FILE_TIMEOUT_SECONDS:-${FRONTEND_AUTH_OIDC_CLIENT_ID_FILE_TIMEOUT_SECONDS:-120}}}"
  while [ ! -s "$oidc_client_id_file" ] && [ "$timeout" -gt 0 ]; do
    sleep 1
    timeout=$((timeout - 1))
  done
  if [ -s "$oidc_client_id_file" ]; then
    oidc_client_id="$(cat "$oidc_client_id_file")"
  fi
fi

cat > /usr/share/nginx/html/runtime-config.js <<EOF
window.__ARTIFICIALFLOW_RUNTIME_CONFIG__ = {
  apiUrl: "$(escape_js_string "$api_url")",
  oidcAuthority: "$(escape_js_string "$oidc_authority")",
  oidcClientId: "$(escape_js_string "$oidc_client_id")"
};
EOF

exec "$@"
