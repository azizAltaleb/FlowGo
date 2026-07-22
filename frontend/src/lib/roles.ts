import type { IdentityResponse } from "@/lib/api";

export const FLOWGO_ADMIN_ROLE = "flowgo admin";
export const FLOWGO_MODELER_ROLE = "flowgo modeler";
export const FLOWGO_CLIENT_ROLE = "flowgo client";

export const STATIC_FLOWGO_ROLES = [
  FLOWGO_ADMIN_ROLE,
  FLOWGO_MODELER_ROLE,
  FLOWGO_CLIENT_ROLE,
];

function normalizeRole(role: string) {
  return role.trim().toLowerCase();
}

export function identityRoles(identity: IdentityResponse | null): string[] {
  return identity?.principal?.roles || [];
}

export function hasRole(identity: IdentityResponse | null, role: string): boolean {
  const wanted = normalizeRole(role);
  return identityRoles(identity).some((candidate) => normalizeRole(candidate) === wanted);
}

export function hasAnyRole(identity: IdentityResponse | null, roles: string[]): boolean {
  return roles.some((role) => hasRole(identity, role));
}

export function hasFlexRole(identity: IdentityResponse | null): boolean {
  const staticRoles = new Set(STATIC_FLOWGO_ROLES.map(normalizeRole));
  return identityRoles(identity).some((role) => !staticRoles.has(normalizeRole(role)));
}

export function isAdmin(identity: IdentityResponse | null): boolean {
  return hasRole(identity, FLOWGO_ADMIN_ROLE);
}

export function isModeler(identity: IdentityResponse | null): boolean {
  return hasRole(identity, FLOWGO_MODELER_ROLE);
}

export function isClientOnly(identity: IdentityResponse | null): boolean {
  return hasRole(identity, FLOWGO_CLIENT_ROLE) && !isAdmin(identity) && !isModeler(identity);
}
