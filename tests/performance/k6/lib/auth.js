/**
 * Shared auth headers for k6 scenarios against auth-enabled stacks.
 */
export function authHeaders(extra = {}) {
  const headers = { ...extra };
  const token = __ENV.AUTH_TOKEN || "";
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}
