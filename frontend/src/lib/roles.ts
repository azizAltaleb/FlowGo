import type { IdentityResponse } from "@/lib/api";

export const ARTIFICIALFLOW_ADMIN_ROLE = "artificialflow admin";
export const ARTIFICIALFLOW_MODELER_ROLE = "artificialflow modeler";
export const ARTIFICIALFLOW_CLIENT_ROLE = "artificialflow client";

export const LEGACY_FLOWGO_ADMIN_ROLE = "flowgo admin";
export const LEGACY_FLOWGO_MODELER_ROLE = "flowgo modeler";
export const LEGACY_FLOWGO_CLIENT_ROLE = "flowgo client";

export const STATIC_ARTIFICIALFLOW_ROLES = [
  ARTIFICIALFLOW_ADMIN_ROLE,
  ARTIFICIALFLOW_MODELER_ROLE,
  ARTIFICIALFLOW_CLIENT_ROLE,
];

function normalizeRole(role: string) {
  return role.trim().toLowerCase();
}

export function canonicalizeRole(role: string): string {
  const normalized = normalizeRole(role);
  switch (normalized) {
    case ARTIFICIALFLOW_ADMIN_ROLE:
    case LEGACY_FLOWGO_ADMIN_ROLE:
      return ARTIFICIALFLOW_ADMIN_ROLE;
    case ARTIFICIALFLOW_MODELER_ROLE:
    case LEGACY_FLOWGO_MODELER_ROLE:
      return ARTIFICIALFLOW_MODELER_ROLE;
    case ARTIFICIALFLOW_CLIENT_ROLE:
    case LEGACY_FLOWGO_CLIENT_ROLE:
      return ARTIFICIALFLOW_CLIENT_ROLE;
    default:
      return role.trim();
  }
}

export function canonicalizeRoles(roles: string[]): string[] {
  const seen = new Set<string>();
  return roles.reduce<string[]>((canonical, role) => {
    const value = canonicalizeRole(role);
    const key = normalizeRole(value);
    if (!value || seen.has(key)) return canonical;
    seen.add(key);
    canonical.push(value);
    return canonical;
  }, []);
}

export function identityRoles(identity: IdentityResponse | null): string[] {
  return canonicalizeRoles(identity?.principal?.roles || []);
}

export function hasRole(identity: IdentityResponse | null, role: string): boolean {
  const wanted = normalizeRole(canonicalizeRole(role));
  return identityRoles(identity).some((candidate) => normalizeRole(candidate) === wanted);
}

export function hasAnyRole(identity: IdentityResponse | null, roles: string[]): boolean {
  return roles.some((role) => hasRole(identity, role));
}

export function hasFlexRole(identity: IdentityResponse | null): boolean {
  const staticRoles = new Set(STATIC_ARTIFICIALFLOW_ROLES.map(normalizeRole));
  return identityRoles(identity).some((role) => !staticRoles.has(normalizeRole(role)));
}

export function isAdmin(identity: IdentityResponse | null): boolean {
  return hasRole(identity, ARTIFICIALFLOW_ADMIN_ROLE);
}

export function isModeler(identity: IdentityResponse | null): boolean {
  return hasRole(identity, ARTIFICIALFLOW_MODELER_ROLE);
}

export function isClientOnly(identity: IdentityResponse | null): boolean {
  return hasRole(identity, ARTIFICIALFLOW_CLIENT_ROLE) && !isAdmin(identity) && !isModeler(identity);
}

// Deprecated compatibility aliases for callers migrating with this release.
export const FLOWGO_ADMIN_ROLE = ARTIFICIALFLOW_ADMIN_ROLE;
export const FLOWGO_MODELER_ROLE = ARTIFICIALFLOW_MODELER_ROLE;
export const FLOWGO_CLIENT_ROLE = ARTIFICIALFLOW_CLIENT_ROLE;
export const STATIC_FLOWGO_ROLES = STATIC_ARTIFICIALFLOW_ROLES;
