import type { IdentityConfigResponse, IdentityResponse } from "@/lib/api";
import { isAdmin, isClientOnly, isModeler, hasFlexRole } from "@/lib/roles";

export function isIdentityConsolePath(path: string): boolean {
  return (
    path === "/identity" ||
    path === "/sdk-clients" ||
    path.startsWith("/identity/") ||
    path.startsWith("/sdk-clients/")
  );
}

export function isBundledZitadelMode(config: IdentityConfigResponse | null | undefined): boolean {
  return config?.deployment_mode === "zitadel";
}

export function canAccessIdentityConsole(
  identity: IdentityResponse | null,
  config: IdentityConfigResponse | null | undefined,
): boolean {
  return isBundledZitadelMode(config) && isAdmin(identity);
}

type PathAccessInput = {
  path: string;
  identity: IdentityResponse | null;
  config: IdentityConfigResponse | null | undefined;
  showDecisionsUi?: boolean;
  showIncidentsUi?: boolean;
};

/**
 * Path gates for DashboardLayout. Identity/SDK Clients are admin-only and only
 * in bundled ZITADEL mode — listed explicitly so a failed admin claim cannot
 * silently fall through as a generic deny without those paths in the allowlist.
 */
export function isDashboardPathAllowed({
  path,
  identity,
  config,
  showDecisionsUi = false,
  showIncidentsUi = false,
}: PathAccessInput): boolean {
  const admin = isAdmin(identity);
  const modeler = isModeler(identity);
  const flexUser = hasFlexRole(identity);
  const clientOnly = isClientOnly(identity);

  if (clientOnly) {
    return false;
  }

  if (isIdentityConsolePath(path)) {
    return canAccessIdentityConsole(identity, config);
  }

  if (path === "/incidents" || path.startsWith("/incidents/")) {
    if (!showIncidentsUi) {
      return false;
    }
    return admin || (flexUser && !modeler);
  }

  if (admin) {
    return true;
  }

  if (modeler && (path === "/processes" || (showDecisionsUi && path === "/decisions") || path.startsWith("/modeler"))) {
    return true;
  }

  if (flexUser && !modeler && !admin && path === "/inbox") {
    return true;
  }

  if (flexUser && !modeler && (path === "/instances" || path.startsWith("/instances/") || path === "/history")) {
    return true;
  }

  return false;
}

export function identityConsoleDeniedMessage(
  identity: IdentityResponse | null,
  config: IdentityConfigResponse | null | undefined,
): string {
  if (!isBundledZitadelMode(config)) {
    return "Identity and SDK Clients are only available when ArtificialFlow runs in bundled ZITADEL mode.";
  }
  if (!isAdmin(identity)) {
    return "Identity and SDK Clients require the artificialflow admin role. Ask an administrator to grant access, or open a page your role allows.";
  }
  return "You do not have access to this page.";
}
